package config

import (
	"os"
	"path/filepath"
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
		os.MkdirAll(dir, 0755)
		return dir
	}

	// Try XDG_CACHE_HOME (user-configurable cache directory)
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		os.MkdirAll(dir, 0755)
		return dir
	}

	// Fallback: ~/.cache/gogent (XDG default when XDG_CACHE_HOME unset)
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".cache", "gogent")
	os.MkdirAll(dir, 0755)
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
