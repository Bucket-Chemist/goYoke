package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// validateSandboxPath tests
// ---------------------------------------------------------------------------

func TestValidateSandboxPath_UnderProjectRoot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	err := validateSandboxPath(filepath.Join(dir, "some", "file.go"))
	require.NoError(t, err)
}

func TestValidateSandboxPath_UnderClaudeDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GOFORTRESS_PROJECT_ROOT", "")
	t.Setenv("GOGENT_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	claudePath := filepath.Join(homeDir, ".claude", "skills", "foo.md")
	err := validateSandboxPath(claudePath)
	require.NoError(t, err)
}

func TestValidateSandboxPath_DotDotRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	// Construct path with raw .. components (not via filepath.Join which cleans them).
	rawPath := dir + "/subdir/../../etc/passwd"
	err := validateSandboxPath(rawPath)
	require.Error(t, err, "path with .. traversal should be rejected")
	assert.Contains(t, err.Error(), "path traversal")
}

func TestValidateSandboxPath_GitDirRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	err := validateSandboxPath(filepath.Join(dir, ".git", "hooks", "pre-commit"))
	require.Error(t, err, "writes to .git directories should be rejected")
	assert.Contains(t, err.Error(), ".git")
}

func TestValidateSandboxPath_OutsideAllowed(t *testing.T) {
	dir := t.TempDir()
	otherDir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)
	t.Setenv("GOGENT_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	err := validateSandboxPath(filepath.Join(otherDir, "file.go"))
	require.Error(t, err, "paths outside allowed dirs should be rejected")
	assert.Contains(t, err.Error(), "not under allowed")
}

// ---------------------------------------------------------------------------
// handleSandboxWrite tests
// ---------------------------------------------------------------------------

func TestHandleSandboxWrite_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	destPath := filepath.Join(dir, "test_output.txt")
	content := "hello world"

	_, out, err := handleSandboxWrite(context.Background(), nil, SandboxWriteInput{
		Content:  content,
		DestPath: destPath,
	})
	require.NoError(t, err)
	assert.True(t, out.Success, "expected success, got error: %s", out.Error)
	assert.Equal(t, filepath.Clean(destPath), out.Path)
	assert.Equal(t, len(content), out.BytesWritten)

	// Verify file content on disk.
	data, readErr := os.ReadFile(out.Path)
	require.NoError(t, readErr)
	assert.Equal(t, content, string(data))

	// Verify default mode 0644.
	info, statErr := os.Stat(out.Path)
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestHandleSandboxWrite_MakeExecutable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	destPath := filepath.Join(dir, "script.sh")
	content := "#!/bin/bash\necho hello"

	_, out, err := handleSandboxWrite(context.Background(), nil, SandboxWriteInput{
		Content:        content,
		DestPath:       destPath,
		MakeExecutable: true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success, "expected success, got error: %s", out.Error)

	info, statErr := os.Stat(out.Path)
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestHandleSandboxWrite_ContentTooLarge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	largeContent := strings.Repeat("a", maxContentSize+1)

	_, out, err := handleSandboxWrite(context.Background(), nil, SandboxWriteInput{
		Content:  largeContent,
		DestPath: filepath.Join(dir, "large.txt"),
	})
	require.NoError(t, err, "oversized content should be a soft error, not a Go error")
	assert.False(t, out.Success)
	assert.Contains(t, out.Error, "exceeds maximum")
}

// ---------------------------------------------------------------------------
// handleSandboxStatus tests
// ---------------------------------------------------------------------------

func TestHandleSandboxStatus_EmptyHistory(t *testing.T) {
	// Reset state for a clean baseline.
	sandboxState.mu.Lock()
	sandboxState.writeHistory = nil
	sandboxState.mu.Unlock()

	_, out, err := handleSandboxStatus(context.Background(), nil, SandboxStatusInput{})
	require.NoError(t, err)
	assert.Empty(t, out.WriteHistory)
	assert.NotNil(t, out.Allowlist)
}

func TestHandleSandboxStatus_AfterWrite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOFORTRESS_PROJECT_ROOT", dir)

	// Reset state for a clean baseline.
	sandboxState.mu.Lock()
	sandboxState.writeHistory = nil
	sandboxState.mu.Unlock()

	content := "status test content"
	destPath := filepath.Join(dir, "status_test.txt")

	_, writeOut, err := handleSandboxWrite(context.Background(), nil, SandboxWriteInput{
		Content:  content,
		DestPath: destPath,
	})
	require.NoError(t, err)
	require.True(t, writeOut.Success, "write must succeed: %s", writeOut.Error)

	_, statusOut, err := handleSandboxStatus(context.Background(), nil, SandboxStatusInput{})
	require.NoError(t, err)
	require.NotEmpty(t, statusOut.WriteHistory, "write history must not be empty after a write")

	found := false
	for _, entry := range statusOut.WriteHistory {
		if entry.Path == filepath.Clean(destPath) {
			found = true
			assert.Equal(t, len(content), entry.Bytes)
			assert.NotEmpty(t, entry.Timestamp)
			break
		}
	}
	assert.True(t, found, "written file path must appear in write history")
}
