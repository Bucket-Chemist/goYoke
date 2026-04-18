package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// CreateSessionDir creates the session directory at {projectDir}/.goyoke/sessions/{sessionID}/
// If sessionID is "unknown" or empty, generates a timestamp-based fallback.
// Returns the absolute path to the created directory.
func CreateSessionDir(projectDir, sessionID string) (string, error) {
	if sessionID == "" || sessionID == "unknown" {
		sessionID = time.Now().Format("20060102-150405")
	}

	sessionDir := filepath.Join(config.RuntimeDir(projectDir), "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("create session dir: %w", err)
	}

	return sessionDir, nil
}

// WriteCurrentSession writes the session directory path to {projectDir}/.goyoke/current-session
func WriteCurrentSession(projectDir, sessionDir string) error {
	runtimeDir := config.RuntimeDir(projectDir)

	currentSessionPath := filepath.Join(runtimeDir, "current-session")
	content := sessionDir + "\n"
	if err := os.WriteFile(currentSessionPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write current-session: %w", err)
	}
	return nil
}

// ReadCurrentSession reads the current session directory path from {projectDir}/.goyoke/current-session
// Returns empty string (no error) if the file doesn't exist.
func ReadCurrentSession(projectDir string) (string, error) {
	currentSessionPath := filepath.Join(config.RuntimeDir(projectDir), "current-session")
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
// Checks GOYOKE_PROJECT_ROOT → GOYOKE_PROJECT_DIR → CLAUDE_PROJECT_DIR in order.
// Returns empty string (no error) if no env var is set.
func ReadCurrentSessionFromEnv() (string, error) {
	projectDir := os.Getenv("GOYOKE_PROJECT_ROOT")
	if projectDir == "" {
		projectDir = os.Getenv("GOYOKE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		return "", nil
	}

	return ReadCurrentSession(projectDir)
}

