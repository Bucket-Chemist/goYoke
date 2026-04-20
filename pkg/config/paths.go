package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	ReminderInterval = 10 // Inject reminder every N tools
	FlushInterval    = 20 // Flush learnings every N tools
)

// GetgoYokeDir returns XDG-compliant goyoke directory.
// Priority: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/goyoke
//
// This ensures compliance with XDG Base Directory Specification:
// - XDG_RUNTIME_DIR: Session-specific runtime files (auto-cleaned on logout)
// - XDG_CACHE_HOME: User-level cache (persistent across sessions)
// - ~/.cache/goyoke: Standard fallback when XDG vars not set
func GetgoYokeDir() string {
	// Try XDG_RUNTIME_DIR (systemd standard, session-scoped)
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "goyoke")
		if err := ensureWritableDir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create goyoke dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Try XDG_CACHE_HOME (user-configurable cache directory)
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "goyoke")
		if err := ensureWritableDir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create goyoke dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Fallback: ~/.cache/goyoke (XDG default when XDG_CACHE_HOME unset)
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to get home directory: %v. Using /tmp fallback.\n", err)
		dir := tempFallbackDir("goyoke-fallback")
		if fallbackErr := ensureWritableDirWithPerm(dir, 0700, true); fallbackErr != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create fallback goyoke dir at %s: %v\n", dir, fallbackErr)
		}
		return dir
	}

	dir := filepath.Join(home, ".cache", "goyoke")
	if err := ensureWritableDir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to create goyoke dir at %s: %v. Using /tmp fallback.\n", dir, err)
		dir = tempFallbackDir("goyoke-fallback")
		if fallbackErr := ensureWritableDirWithPerm(dir, 0700, true); fallbackErr != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create fallback goyoke dir at %s: %v\n", dir, fallbackErr)
		}
		return dir
	}
	return dir
}

// ensureWritableDir makes sure dir exists and accepts file creation.
func ensureWritableDir(dir string) error {
	return ensureWritableDirWithPerm(dir, 0755, false)
}

func ensureWritableDirWithPerm(dir string, perm os.FileMode, forcePerm bool) error {
	existed := false
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		existed = true
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	if forcePerm || !existed {
		if err := os.Chmod(dir, perm); err != nil && !os.IsPermission(err) && !os.IsNotExist(err) {
			return err
		}
	}

	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	} else if err != nil {
		return err
	}

	probe, err := os.CreateTemp(dir, ".goyoke-write-check-*")
	if err != nil {
		return err
	}

	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func tempFallbackDir(prefix string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", prefix, os.Getuid()))
}

// GetTierFilePath returns path to current-tier state file.
func GetTierFilePath() string {
	return filepath.Join(GetgoYokeDir(), "current-tier")
}

// GetMaxDelegationPath returns path to max_delegation ceiling file.
func GetMaxDelegationPath() string {
	return filepath.Join(GetgoYokeDir(), "max_delegation")
}

// GetViolationsLogPath returns path to routing violations log (JSONL format).
func GetViolationsLogPath() string {
	return filepath.Join(GetgoYokeDir(), "routing-violations.jsonl")
}

// RuntimeDir returns the project-scoped runtime directory.
// Priority: GOYOKE_RUNTIME_DIR env var > {projectDir}/.goyoke
// Creates the directory if it does not exist (idempotent via os.MkdirAll).
func RuntimeDir(projectDir string) string {
	if override := os.Getenv("GOYOKE_RUNTIME_DIR"); override != "" {
		os.MkdirAll(override, 0755)
		return override
	}
	dir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(dir, 0755)
	return dir
}

// ProjectMemoryDir returns the memory subdirectory within the project runtime dir.
// Creates the directory if it does not exist.
func ProjectMemoryDir(projectDir string) string {
	dir := filepath.Join(RuntimeDir(projectDir), "memory")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetProjectViolationsLogPath returns project-scoped violation log path.
// Used for dual-write pattern - integrates with session archive.
func GetProjectViolationsLogPath(projectDir string) string {
	return filepath.Join(ProjectMemoryDir(projectDir), "routing-violations.jsonl")
}

// GetToolCounterPath returns path to tool counter file.
func GetToolCounterPath() string {
	return filepath.Join(GetgoYokeDir(), "tool-counter")
}

// InitializeToolCounter creates/resets tool counter to 0.
func InitializeToolCounter() error {
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte("0"), 0644); err != nil {
		return fmt.Errorf("failed to initialize tool counter at %s: %w", path, err)
	}
	return nil
}

// GetToolCount reads current tool count. Returns 0 if file doesn't exist or is empty.
func GetToolCount() (int, error) {
	path := GetToolCounterPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read tool counter at %s: %w", path, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		// Empty file treated as 0 (can happen during concurrent writes)
		return 0, nil
	}

	count, err := strconv.Atoi(content)
	if err != nil {
		return 0, fmt.Errorf("invalid tool counter value at %s: %w", path, err)
	}
	return count, nil
}

// IncrementToolCount atomically increments tool counter.
// Uses file locking (flock) to ensure true atomicity in concurrent scenarios.
func IncrementToolCount() error {
	path := GetToolCounterPath()

	// Open file for read-write, create if doesn't exist
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open counter file at %s: %w", path, err)
	}
	defer file.Close()

	// Acquire exclusive lock (blocks until available)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock counter file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Read current count
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read counter: %w", err)
	}

	count := 0
	if len(data) > 0 {
		content := strings.TrimSpace(string(data))
		if content != "" {
			count, err = strconv.Atoi(content)
			if err != nil {
				return fmt.Errorf("invalid counter value: %w", err)
			}
		}
	}

	// Increment
	newCount := count + 1

	// Write back (truncate first)
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate counter file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek counter file: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(newCount)); err != nil {
		return fmt.Errorf("failed to write new count: %w", err)
	}

	return nil
}

// GetToolCountAndIncrement atomically reads current count and increments.
// Returns the count AFTER incrementing.
// Uses existing flock pattern from IncrementToolCount for atomicity.
func GetToolCountAndIncrement() (int, error) {
	path := GetToolCounterPath()

	// Open file for read-write, create if doesn't exist
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("[config] failed to open counter file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock (blocks until available)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return 0, fmt.Errorf("[config] failed to lock counter file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Read current count
	var count int
	data, err := io.ReadAll(file)
	if err == nil && len(data) > 0 {
		content := strings.TrimSpace(string(data))
		if content != "" {
			count, _ = strconv.Atoi(content)
		}
	}

	// Increment
	count++

	// Write back (truncate first)
	if err := file.Truncate(0); err != nil {
		return 0, fmt.Errorf("[config] failed to truncate counter: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, fmt.Errorf("[config] failed to seek counter: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(count)); err != nil {
		return 0, fmt.Errorf("[config] failed to write counter: %w", err)
	}

	return count, nil
}

// ShouldRemind returns true if reminder should be injected at this count.
func ShouldRemind(count int) bool {
	return count > 0 && count%ReminderInterval == 0
}

// ShouldFlush returns true if pending learnings should be flushed at this count.
func ShouldFlush(count int) bool {
	return count > 0 && count%FlushInterval == 0
}

// GetgoYokeDataDir returns XDG-compliant data directory for persistent files.
// Priority: XDG_DATA_HOME > ~/.local/share/goyoke
// Use for: ML telemetry, training datasets, long-term logs
//
// This differs from GetgoYokeDir() which uses XDG_CACHE_HOME for non-essential cached data.
// Per XDG Base Directory Specification:
// - XDG_CACHE_HOME: Non-essential cached data (may be cleared)
// - XDG_DATA_HOME: User-specific data files (should persist)
func GetgoYokeDataDir() string {
	// Try XDG_DATA_HOME (user-configurable data directory)
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "goyoke")
		if err := ensureWritableDir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create goyoke data dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Fallback: ~/.local/share/goyoke (XDG default when XDG_DATA_HOME unset)
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to get home directory: %v. Using /tmp fallback.\n", err)
		dir := tempFallbackDir("goyoke-data")
		if fallbackErr := ensureWritableDirWithPerm(dir, 0700, true); fallbackErr != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create fallback goyoke data dir at %s: %v\n", dir, fallbackErr)
		}
		return dir
	}

	dir := filepath.Join(home, ".local", "share", "goyoke")
	if err := ensureWritableDir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to create goyoke data dir at %s: %v. Using /tmp fallback.\n", dir, err)
		dir = tempFallbackDir("goyoke-data")
		if fallbackErr := ensureWritableDirWithPerm(dir, 0700, true); fallbackErr != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create fallback goyoke data dir at %s: %v\n", dir, fallbackErr)
		}
		return dir
	}
	return dir
}

// GetMLToolEventsPath returns path for ML tool events log.
// This persistent data file tracks tool usage for ML-based routing optimization.
func GetMLToolEventsPath() string {
	return filepath.Join(GetgoYokeDataDir(), "tool-events.jsonl")
}

// GetRoutingDecisionsPath returns path for routing decisions log.
// This persistent data file tracks routing outcomes for model training.
func GetRoutingDecisionsPath() string {
	return filepath.Join(GetgoYokeDataDir(), "routing-decisions.jsonl")
}

// GetCollaborationsPath returns path for agent collaborations log.
// This persistent data file tracks multi-agent workflows for pattern analysis.
func GetCollaborationsPath() string {
	return filepath.Join(GetgoYokeDataDir(), "agent-collaborations.jsonl")
}

// GetRoutingDecisionsPathWithProjectDir returns routing decisions path, checking GOYOKE_PROJECT_DIR first.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/routing-decisions.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetRoutingDecisionsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "routing-decisions.jsonl")
	}
	return GetRoutingDecisionsPath()
}

// GetRoutingDecisionUpdatesPathWithProjectDir returns routing decision updates path, checking GOYOKE_PROJECT_DIR first.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/routing-decision-updates.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetRoutingDecisionUpdatesPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "routing-decision-updates.jsonl")
	}
	return filepath.Join(GetgoYokeDataDir(), "routing-decision-updates.jsonl")
}

// GetCollaborationsPathWithProjectDir returns collaborations path, checking GOYOKE_PROJECT_DIR first.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/agent-collaborations.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetCollaborationsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "agent-collaborations.jsonl")
	}
	return GetCollaborationsPath()
}

// GetMLToolEventsPathWithProjectDir returns ML tool events path, checking GOYOKE_PROJECT_DIR first.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/ml-tool-events.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetMLToolEventsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "ml-tool-events.jsonl")
	}
	return GetMLToolEventsPath()
}

// GetAgentLifecyclePathWithProjectDir returns agent lifecycle path, checking GOYOKE_PROJECT_DIR first.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/agent-lifecycle.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetAgentLifecyclePathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "agent-lifecycle.jsonl")
	}
	return filepath.Join(GetgoYokeDataDir(), "agent-lifecycle.jsonl")
}

// GetReviewFindingsPathWithProjectDir returns path for review findings log.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/review-findings.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetReviewFindingsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "review-findings.jsonl")
	}
	return filepath.Join(GetgoYokeDataDir(), "review-findings.jsonl")
}

// GetReviewOutcomesPathWithProjectDir returns path for review outcome updates.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/review-outcomes.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetReviewOutcomesPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "review-outcomes.jsonl")
	}
	return filepath.Join(GetgoYokeDataDir(), "review-outcomes.jsonl")
}

// GetSharpEdgeHitsPathWithProjectDir returns path for sharp edge correlation log.
// Priority:
//  1. If GOYOKE_PROJECT_DIR is set: $GOYOKE_PROJECT_DIR/.goyoke/sharp-edge-hits.jsonl
//  2. Otherwise: XDG data directory (GetgoYokeDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetSharpEdgeHitsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".goyoke", "sharp-edge-hits.jsonl")
	}
	return filepath.Join(GetgoYokeDataDir(), "sharp-edge-hits.jsonl")
}

// GetGuardsDir returns the path to the session-scoped guard directory.
// Creates the directory via os.MkdirAll if it doesn't exist.
func GetGuardsDir() string {
	dir := filepath.Join(GetgoYokeDir(), "guards")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to create guards dir at %s: %v\n", dir, err)
	}
	return dir
}

// GetGuardFilePath returns the path to the guard JSON file for a given session ID.
func GetGuardFilePath(sessionID string) string {
	return filepath.Join(GetGuardsDir(), sessionID+".json")
}

// GetGuardLockPath returns the path to the guard lock file for a given session ID.
func GetGuardLockPath(sessionID string) string {
	return filepath.Join(GetGuardsDir(), sessionID+".lock")
}
