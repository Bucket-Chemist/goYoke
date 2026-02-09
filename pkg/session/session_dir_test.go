package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSessionDir_Basic(t *testing.T) {
	projectDir := t.TempDir()
	sessionID := "test-session-123"

	sessionDir, err := CreateSessionDir(projectDir, sessionID)
	require.NoError(t, err)

	expectedPath := filepath.Join(projectDir, ".claude", "sessions", sessionID)
	assert.Equal(t, expectedPath, sessionDir)

	// Verify directory exists
	info, err := os.Stat(sessionDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateSessionDir_UnknownID(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{"empty string", ""},
		{"unknown", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()

			sessionDir, err := CreateSessionDir(projectDir, tc.sessionID)
			require.NoError(t, err)

			// Should have generated timestamp-based ID
			sessionID := filepath.Base(sessionDir)
			_, parseErr := time.Parse("20060102-150405", sessionID)
			assert.NoError(t, parseErr, "generated sessionID should be valid timestamp")
		})
	}
}

func TestCreateSessionDir_AlreadyExists(t *testing.T) {
	projectDir := t.TempDir()
	sessionID := "existing-session"

	// Create once
	sessionDir1, err := CreateSessionDir(projectDir, sessionID)
	require.NoError(t, err)

	// Create again - should be idempotent
	sessionDir2, err := CreateSessionDir(projectDir, sessionID)
	require.NoError(t, err)

	assert.Equal(t, sessionDir1, sessionDir2)
}

func TestWriteCurrentSession(t *testing.T) {
	projectDir := t.TempDir()
	sessionDir := filepath.Join(projectDir, ".claude", "sessions", "test-123")

	err := WriteCurrentSession(projectDir, sessionDir)
	require.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(filepath.Join(projectDir, ".claude", "current-session"))
	require.NoError(t, err)
	assert.Equal(t, sessionDir+"\n", string(content))
}

func TestReadCurrentSession_Exists(t *testing.T) {
	projectDir := t.TempDir()
	sessionDir := filepath.Join(projectDir, ".claude", "sessions", "test-456")

	// Write first
	err := WriteCurrentSession(projectDir, sessionDir)
	require.NoError(t, err)

	// Read back
	result, err := ReadCurrentSession(projectDir)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, result)
}

func TestReadCurrentSession_Missing(t *testing.T) {
	projectDir := t.TempDir()

	result, err := ReadCurrentSession(projectDir)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestReadCurrentSessionFromEnv(t *testing.T) {
	projectDir := t.TempDir()
	sessionDir := filepath.Join(projectDir, ".claude", "sessions", "env-test")

	// Write current session
	err := WriteCurrentSession(projectDir, sessionDir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		envVar string
	}{
		{"GOGENT_PROJECT_ROOT", "GOGENT_PROJECT_ROOT"},
		{"GOGENT_PROJECT_DIR", "GOGENT_PROJECT_DIR"},
		{"CLAUDE_PROJECT_DIR", "CLAUDE_PROJECT_DIR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envVar, projectDir)

			result, err := ReadCurrentSessionFromEnv()
			require.NoError(t, err)
			assert.Equal(t, sessionDir, result)
		})
	}
}

func TestReadCurrentSessionFromEnv_NoEnv(t *testing.T) {
	// Ensure no env vars are set
	t.Setenv("GOGENT_PROJECT_ROOT", "")
	t.Setenv("GOGENT_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	result, err := ReadCurrentSessionFromEnv()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestSetupTmpSymlink_Fresh(t *testing.T) {
	projectDir := t.TempDir()
	sessionDir := filepath.Join(projectDir, ".claude", "sessions", "fresh-session")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	err := SetupTmpSymlink(projectDir, sessionDir)
	require.NoError(t, err)

	// Verify symlink exists and points to sessionDir
	tmpPath := filepath.Join(projectDir, ".claude", "tmp")
	info, err := os.Lstat(tmpPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be symlink")

	target, err := os.Readlink(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, target)
}

func TestSetupTmpSymlink_ExistingSymlink(t *testing.T) {
	projectDir := t.TempDir()
	oldSessionDir := filepath.Join(projectDir, ".claude", "sessions", "old-session")
	newSessionDir := filepath.Join(projectDir, ".claude", "sessions", "new-session")
	require.NoError(t, os.MkdirAll(oldSessionDir, 0755))
	require.NoError(t, os.MkdirAll(newSessionDir, 0755))

	// Create initial symlink
	err := SetupTmpSymlink(projectDir, oldSessionDir)
	require.NoError(t, err)

	// Replace with new symlink
	err = SetupTmpSymlink(projectDir, newSessionDir)
	require.NoError(t, err)

	// Verify it points to new session
	tmpPath := filepath.Join(projectDir, ".claude", "tmp")
	target, err := os.Readlink(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, newSessionDir, target)
}

func TestSetupTmpSymlink_ExistingDir(t *testing.T) {
	projectDir := t.TempDir()
	sessionDir := filepath.Join(projectDir, ".claude", "sessions", "migrate-session")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	// Create real directory with some files
	tmpPath := filepath.Join(projectDir, ".claude", "tmp")
	require.NoError(t, os.MkdirAll(tmpPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpPath, "file1.txt"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpPath, "file2.txt"), []byte("test"), 0644))

	err := SetupTmpSymlink(projectDir, sessionDir)
	require.NoError(t, err)

	// Verify symlink now exists
	info, err := os.Lstat(tmpPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be symlink")

	// Verify old content was migrated
	migratedPath := filepath.Join(projectDir, ".claude", "tmp.pre-sessions")
	entries, err := os.ReadDir(migratedPath)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "should have migrated 2 files")
}

func TestMigrateExistingTmp_AlreadyMigrated(t *testing.T) {
	projectDir := t.TempDir()

	// Create both source and dest
	tmpPath := filepath.Join(projectDir, ".claude", "tmp")
	require.NoError(t, os.MkdirAll(tmpPath, 0755))

	migratedPath := filepath.Join(projectDir, ".claude", "tmp.pre-sessions")
	require.NoError(t, os.MkdirAll(migratedPath, 0755))

	err := migrateExistingTmp(projectDir)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "already exists"), "should indicate dest exists")
}
