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

// GetGOgentDir returns XDG-compliant gogent directory.
// Priority: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent
//
// This ensures compliance with XDG Base Directory Specification:
// - XDG_RUNTIME_DIR: Session-specific runtime files (auto-cleaned on logout)
// - XDG_CACHE_HOME: User-level cache (persistent across sessions)
// - ~/.cache/gogent: Standard fallback when XDG vars not set
func GetGOgentDir() string {
	// Try XDG_RUNTIME_DIR (systemd standard, session-scoped)
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create gogent dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Try XDG_CACHE_HOME (user-configurable cache directory)
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create gogent dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Fallback: ~/.cache/gogent (XDG default when XDG_CACHE_HOME unset)
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to get home directory: %v. Using /tmp fallback.\n", err)
		dir := filepath.Join(os.TempDir(), "gogent-fallback")
		os.MkdirAll(dir, 0755)
		return dir
	}

	dir := filepath.Join(home, ".cache", "gogent")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to create gogent dir at %s: %v. Using /tmp fallback.\n", dir, err)
		dir = filepath.Join(os.TempDir(), "gogent-fallback")
		os.MkdirAll(dir, 0755)
		return dir
	}
	return dir
}

// GetTierFilePath returns path to current-tier state file.
func GetTierFilePath() string {
	return filepath.Join(GetGOgentDir(), "current-tier")
}

// GetMaxDelegationPath returns path to max_delegation ceiling file.
func GetMaxDelegationPath() string {
	return filepath.Join(GetGOgentDir(), "max_delegation")
}

// GetViolationsLogPath returns path to routing violations log (JSONL format).
func GetViolationsLogPath() string {
	return filepath.Join(GetGOgentDir(), "routing-violations.jsonl")
}

// GetProjectViolationsLogPath returns project-scoped violation log path.
// Used for dual-write pattern - integrates with session archive.
func GetProjectViolationsLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "routing-violations.jsonl")
}

// GetToolCounterPath returns path to tool counter file.
func GetToolCounterPath() string {
	return filepath.Join(GetGOgentDir(), "tool-counter")
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

// GetGOgentDataDir returns XDG-compliant data directory for persistent files.
// Priority: XDG_DATA_HOME > ~/.local/share/gogent
// Use for: ML telemetry, training datasets, long-term logs
//
// This differs from GetGOgentDir() which uses XDG_CACHE_HOME for non-essential cached data.
// Per XDG Base Directory Specification:
// - XDG_CACHE_HOME: Non-essential cached data (may be cleared)
// - XDG_DATA_HOME: User-specific data files (should persist)
func GetGOgentDataDir() string {
	// Try XDG_DATA_HOME (user-configurable data directory)
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[config] Failed to create gogent data dir at %s: %v. Trying fallback.\n", dir, err)
		} else {
			return dir
		}
	}

	// Fallback: ~/.local/share/gogent (XDG default when XDG_DATA_HOME unset)
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to get home directory: %v. Using /tmp fallback.\n", err)
		dir := filepath.Join(os.TempDir(), "gogent-data")
		os.MkdirAll(dir, 0755)
		return dir
	}

	dir := filepath.Join(home, ".local", "share", "gogent")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[config] Failed to create gogent data dir at %s: %v. Using /tmp fallback.\n", dir, err)
		dir = filepath.Join(os.TempDir(), "gogent-data")
		os.MkdirAll(dir, 0755)
		return dir
	}
	return dir
}

// GetMLToolEventsPath returns path for ML tool events log.
// This persistent data file tracks tool usage for ML-based routing optimization.
func GetMLToolEventsPath() string {
	return filepath.Join(GetGOgentDataDir(), "tool-events.jsonl")
}

// GetRoutingDecisionsPath returns path for routing decisions log.
// This persistent data file tracks routing outcomes for model training.
func GetRoutingDecisionsPath() string {
	return filepath.Join(GetGOgentDataDir(), "routing-decisions.jsonl")
}

// GetCollaborationsPath returns path for agent collaborations log.
// This persistent data file tracks multi-agent workflows for pattern analysis.
func GetCollaborationsPath() string {
	return filepath.Join(GetGOgentDataDir(), "agent-collaborations.jsonl")
}

// GetRoutingDecisionsPathWithProjectDir returns routing decisions path, checking GOGENT_PROJECT_DIR first.
// Priority:
//  1. If GOGENT_PROJECT_DIR is set: $GOGENT_PROJECT_DIR/.gogent/routing-decisions.jsonl
//  2. Otherwise: XDG data directory (GetGOgentDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetRoutingDecisionsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".gogent", "routing-decisions.jsonl")
	}
	return GetRoutingDecisionsPath()
}

// GetRoutingDecisionUpdatesPathWithProjectDir returns routing decision updates path, checking GOGENT_PROJECT_DIR first.
// Priority:
//  1. If GOGENT_PROJECT_DIR is set: $GOGENT_PROJECT_DIR/.gogent/routing-decision-updates.jsonl
//  2. Otherwise: XDG data directory (GetGOgentDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetRoutingDecisionUpdatesPathWithProjectDir() string {
	if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".gogent", "routing-decision-updates.jsonl")
	}
	return filepath.Join(GetGOgentDataDir(), "routing-decision-updates.jsonl")
}

// GetCollaborationsPathWithProjectDir returns collaborations path, checking GOGENT_PROJECT_DIR first.
// Priority:
//  1. If GOGENT_PROJECT_DIR is set: $GOGENT_PROJECT_DIR/.gogent/agent-collaborations.jsonl
//  2. Otherwise: XDG data directory (GetGOgentDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetCollaborationsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".gogent", "agent-collaborations.jsonl")
	}
	return GetCollaborationsPath()
}

// GetMLToolEventsPathWithProjectDir returns ML tool events path, checking GOGENT_PROJECT_DIR first.
// Priority:
//  1. If GOGENT_PROJECT_DIR is set: $GOGENT_PROJECT_DIR/.gogent/ml-tool-events.jsonl
//  2. Otherwise: XDG data directory (GetGOgentDataDir())
//
// This enables test isolation while maintaining production XDG compliance.
func GetMLToolEventsPathWithProjectDir() string {
	if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".gogent", "ml-tool-events.jsonl")
	}
	return GetMLToolEventsPath()
}
