package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateSessionDir creates the session directory at {projectDir}/.claude/sessions/{sessionID}/
// If sessionID is "unknown" or empty, generates a timestamp-based fallback.
// Returns the absolute path to the created directory.
func CreateSessionDir(projectDir, sessionID string) (string, error) {
	if sessionID == "" || sessionID == "unknown" {
		sessionID = time.Now().Format("20060102-150405")
	}

	sessionDir := filepath.Join(projectDir, ".claude", "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("create session dir: %w", err)
	}

	return sessionDir, nil
}

// WriteCurrentSession writes the session directory path to {projectDir}/.claude/current-session
func WriteCurrentSession(projectDir, sessionDir string) error {
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	currentSessionPath := filepath.Join(claudeDir, "current-session")
	content := sessionDir + "\n"
	if err := os.WriteFile(currentSessionPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write current-session: %w", err)
	}
	return nil
}

// ReadCurrentSession reads the current session directory path from {projectDir}/.claude/current-session
// Returns empty string (no error) if the file doesn't exist.
func ReadCurrentSession(projectDir string) (string, error) {
	currentSessionPath := filepath.Join(projectDir, ".claude", "current-session")
	content, err := os.ReadFile(currentSessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read current-session: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

// ReadCurrentSessionFromEnv resolves the project directory from environment variables
// and reads the current session directory path.
// Checks GOGENT_PROJECT_ROOT → GOGENT_PROJECT_DIR → CLAUDE_PROJECT_DIR in order.
// Returns empty string (no error) if no env var is set.
func ReadCurrentSessionFromEnv() (string, error) {
	projectDir := os.Getenv("GOGENT_PROJECT_ROOT")
	if projectDir == "" {
		projectDir = os.Getenv("GOGENT_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		return "", nil
	}

	return ReadCurrentSession(projectDir)
}

// SetupTmpSymlink creates/updates the {projectDir}/.claude/tmp symlink to point to sessionDir.
// If .claude/tmp exists as a real directory, migrates it first.
func SetupTmpSymlink(projectDir, sessionDir string) error {
	tmpPath := filepath.Join(projectDir, ".claude", "tmp")

	// Check what exists at tmpPath
	info, err := os.Lstat(tmpPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// Existing symlink - remove it
			fmt.Fprintf(os.Stderr, "[session-dir] Removing old .claude/tmp symlink\n")
			if err := os.Remove(tmpPath); err != nil {
				return fmt.Errorf("remove old symlink: %w", err)
			}
		} else if info.IsDir() {
			// Real directory - migrate it
			fmt.Fprintf(os.Stderr, "[session-dir] Found real directory at .claude/tmp, migrating...\n")
			if err := migrateExistingTmp(projectDir); err != nil {
				return err
			}
		} else {
			// Regular file or other - remove it
			if err := os.Remove(tmpPath); err != nil {
				return fmt.Errorf("remove existing file: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("lstat tmp path: %w", err)
	}

	// Create symlink
	if err := os.Symlink(sessionDir, tmpPath); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[session-dir] Created .claude/tmp → %s\n", sessionDir)
	return nil
}

// migrateExistingTmp moves {projectDir}/.claude/tmp to {projectDir}/.claude/tmp.pre-sessions
func migrateExistingTmp(projectDir string) error {
	source := filepath.Join(projectDir, ".claude", "tmp")
	dest := filepath.Join(projectDir, ".claude", "tmp.pre-sessions")

	// Count entries for logging
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read tmp dir: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[session-dir] Migrating %d entries from .claude/tmp/ to .claude/tmp.pre-sessions\n", len(entries))

	// Check if dest already exists
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("migration target %s already exists; previous migration may have occurred", dest)
	}

	// Rename
	if err := os.Rename(source, dest); err != nil {
		return fmt.Errorf("migrate tmp dir: %w", err)
	}

	return nil
}
