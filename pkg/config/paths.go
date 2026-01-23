package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
