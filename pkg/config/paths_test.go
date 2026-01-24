package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestGetGOgentDir_XDG_RUNTIME_DIR(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set XDG_RUNTIME_DIR (highest priority)
	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)
	os.Setenv("XDG_CACHE_HOME", "/should/not/use/this")

	result := GetGOgentDir()
	expected := filepath.Join(testDir, "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetGOgentDir to create directory")
	}
}

func TestGetGOgentDir_XDG_CACHE_HOME(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR, set XDG_CACHE_HOME (second priority)
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	result := GetGOgentDir()
	expected := filepath.Join(testDir, "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetGOgentDir to create directory")
	}
}

func TestGetGOgentDir_Fallback(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset both XDG vars (fallback to ~/.cache/gogent)
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_CACHE_HOME")

	result := GetGOgentDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_EmptyXDGVars(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set to empty strings (should fallback)
	os.Setenv("XDG_RUNTIME_DIR", "")
	os.Setenv("XDG_CACHE_HOME", "")

	result := GetGOgentDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetTierFilePath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetTierFilePath()
	expected := filepath.Join(testDir, "gogent", "current-tier")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetMaxDelegationPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetMaxDelegationPath()
	expected := filepath.Join(testDir, "gogent", "max_delegation")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetViolationsLogPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetViolationsLogPath()
	expected := filepath.Join(testDir, "gogent", "routing-violations.jsonl")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify filename ends with .jsonl
	if !strings.HasSuffix(result, ".jsonl") {
		t.Error("Expected violations log to have .jsonl extension")
	}
}

func TestGetGOgentDir_PriorityOrder(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	runtimeDir := t.TempDir()
	cacheDir := t.TempDir()

	// Both set: XDG_RUNTIME_DIR wins
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	result := GetGOgentDir()
	expected := filepath.Join(runtimeDir, "gogent")

	if result != expected {
		t.Errorf("XDG_RUNTIME_DIR should have priority. Expected %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_CreatesDirectory(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	gogentPath := filepath.Join(testDir, "gogent")

	// Ensure directory doesn't exist yet
	os.RemoveAll(gogentPath)

	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetGOgentDir()

	// Verify directory was created
	info, err := os.Stat(result)
	if os.IsNotExist(err) {
		t.Error("GetGOgentDir should create directory if it doesn't exist")
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Error("GetGOgentDir should create a directory, not a file")
	}

	// Verify permissions (0755)
	if info.Mode().Perm() != 0755 {
		t.Errorf("Expected permissions 0755, got %o", info.Mode().Perm())
	}
}

func TestGetProjectViolationsLogPath(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
		expected   string
	}{
		{
			name:       "absolute path",
			projectDir: "/home/user/my-project",
			expected:   "/home/user/my-project/.claude/memory/routing-violations.jsonl",
		},
		{
			name:       "relative path",
			projectDir: "my-project",
			expected:   "my-project/.claude/memory/routing-violations.jsonl",
		},
		{
			name:       "nested project",
			projectDir: "/home/user/workspace/nested/project",
			expected:   "/home/user/workspace/nested/project/.claude/memory/routing-violations.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectViolationsLogPath(tt.projectDir)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			// Verify path ends with .jsonl
			if !strings.HasSuffix(result, ".jsonl") {
				t.Error("Expected path to end with .jsonl")
			}

			// Verify path contains .claude/memory
			if !strings.Contains(result, ".claude/memory") {
				t.Error("Expected path to contain .claude/memory")
			}
		})
	}
}

func TestGetToolCounterPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR so XDG_CACHE_HOME takes priority
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	result := GetToolCounterPath()
	expected := filepath.Join(testDir, "gogent", "tool-counter")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify filename is "tool-counter"
	if filepath.Base(result) != "tool-counter" {
		t.Errorf("Expected filename 'tool-counter', got %s", filepath.Base(result))
	}
}

func TestToolCounter_Initialize(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR so XDG_CACHE_HOME takes priority
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Initialize counter
	err := InitializeToolCounter()
	if err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Verify file was created with correct content
	path := GetToolCounterPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read counter file: %v", err)
	}

	if string(data) != "0" {
		t.Errorf("Expected counter to be initialized to '0', got %s", string(data))
	}

	// Verify file permissions (0644)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat counter file: %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("Expected permissions 0644, got %o", info.Mode().Perm())
	}
}

func TestToolCounter_GetCount_NotInitialized(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR so XDG_CACHE_HOME takes priority
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Get count before initialization (file doesn't exist)
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed when file doesn't exist: %v", err)
	}

	// Should return 0 when file doesn't exist
	if count != 0 {
		t.Errorf("Expected count=0 when file doesn't exist, got %d", count)
	}
}

func TestToolCounter_Increment(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR so XDG_CACHE_HOME takes priority
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Initialize counter
	if err := InitializeToolCounter(); err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Verify initial count
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected initial count=0, got %d", count)
	}

	// Increment 5 times
	for i := 1; i <= 5; i++ {
		if err := IncrementToolCount(); err != nil {
			t.Fatalf("IncrementToolCount failed on iteration %d: %v", i, err)
		}

		count, err := GetToolCount()
		if err != nil {
			t.Fatalf("GetToolCount failed after increment %d: %v", i, err)
		}

		if count != i {
			t.Errorf("Expected count=%d after %d increments, got %d", i, i, count)
		}
	}
}

func TestToolCounter_ConcurrentIncrement(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR so XDG_CACHE_HOME takes priority
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Initialize counter
	if err := InitializeToolCounter(); err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Run 100 concurrent increments (10 goroutines × 10 increments)
	const numGoroutines = 10
	const incrementsPerGoroutine = 10
	const expectedTotal = numGoroutines * incrementsPerGoroutine

	errChan := make(chan error, numGoroutines*incrementsPerGoroutine)
	doneChan := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			for i := 0; i < incrementsPerGoroutine; i++ {
				if err := IncrementToolCount(); err != nil {
					errChan <- err
					return
				}
			}
			doneChan <- true
		}()
	}

	// Wait for all goroutines to complete
	for g := 0; g < numGoroutines; g++ {
		select {
		case <-doneChan:
			// Success
		case err := <-errChan:
			t.Fatalf("Concurrent increment failed: %v", err)
		}
	}

	// Verify final count is exactly expectedTotal
	finalCount, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed after concurrent increments: %v", err)
	}

	if finalCount != expectedTotal {
		t.Errorf("Expected count=%d after concurrent increments, got %d (lost updates detected)", expectedTotal, finalCount)
	}
}

func TestGetToolCount_InvalidContent(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Write invalid content to counter file
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte("not-a-number"), 0644); err != nil {
		t.Fatalf("Failed to write invalid content: %v", err)
	}

	// Should return error for invalid content
	_, err := GetToolCount()
	if err == nil {
		t.Error("Expected error for invalid counter content, got nil")
	}
	if !strings.Contains(err.Error(), "invalid tool counter value") {
		t.Errorf("Expected 'invalid tool counter value' error, got: %v", err)
	}
}

func TestGetToolCount_EmptyFile(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Create empty counter file
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// Empty file should return 0 (not error)
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed on empty file: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for empty file, got %d", count)
	}
}

func TestGetToolCount_WhitespaceOnly(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Create file with only whitespace
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte("  \n\t  "), 0644); err != nil {
		t.Fatalf("Failed to write whitespace file: %v", err)
	}

	// Whitespace-only file should return 0 (not error)
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed on whitespace file: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for whitespace file, got %d", count)
	}
}

func TestIncrementToolCount_FromNonExistent(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Don't initialize - file doesn't exist
	// IncrementToolCount should create file and set to 1
	if err := IncrementToolCount(); err != nil {
		t.Fatalf("IncrementToolCount failed on non-existent file: %v", err)
	}

	// Verify count is 1
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 after first increment, got %d", count)
	}
}

func TestIncrementToolCount_InvalidInitialContent(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Write invalid content
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte("invalid"), 0644); err != nil {
		t.Fatalf("Failed to write invalid content: %v", err)
	}

	// IncrementToolCount should return error
	err := IncrementToolCount()
	if err == nil {
		t.Error("Expected error for invalid counter content, got nil")
	}
	if !strings.Contains(err.Error(), "invalid counter value") {
		t.Errorf("Expected 'invalid counter value' error, got: %v", err)
	}
}

func TestIncrementToolCount_EmptyInitialContent(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Create empty file
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// IncrementToolCount should treat empty as 0 and increment to 1
	if err := IncrementToolCount(); err != nil {
		t.Fatalf("IncrementToolCount failed on empty file: %v", err)
	}

	// Verify count is 1
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 after incrementing empty file, got %d", count)
	}
}

func TestIncrementToolCount_LargeValue(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	// Initialize with large value
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte("9999"), 0644); err != nil {
		t.Fatalf("Failed to write large value: %v", err)
	}

	// Increment
	if err := IncrementToolCount(); err != nil {
		t.Fatalf("IncrementToolCount failed: %v", err)
	}

	// Verify count is 10000
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed: %v", err)
	}
	if count != 10000 {
		t.Errorf("Expected count=10000, got %d", count)
	}
}

func TestGetGOgentDir_XDGRuntimeDirInvalidPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set XDG_RUNTIME_DIR to a path that cannot be created (e.g., invalid path with null byte)
	// This will trigger the error path and fallback to XDG_CACHE_HOME
	os.Setenv("XDG_RUNTIME_DIR", "/dev/null/cannot-create-here")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	result := GetGOgentDir()
	expected := filepath.Join(testDir, "gogent")

	if result != expected {
		t.Errorf("Expected fallback to XDG_CACHE_HOME: %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_BothXDGFail(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Set both XDG paths to invalid locations
	os.Setenv("XDG_RUNTIME_DIR", "/dev/null/invalid1")
	os.Setenv("XDG_CACHE_HOME", "/dev/null/invalid2")

	// Set valid HOME so we can test the .cache/gogent fallback
	testDir := t.TempDir()
	os.Setenv("HOME", testDir)

	result := GetGOgentDir()
	expected := filepath.Join(testDir, ".cache", "gogent")

	if result != expected {
		t.Errorf("Expected fallback to HOME/.cache/gogent: %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_AllPathsFail(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Set all paths to invalid locations
	os.Setenv("XDG_RUNTIME_DIR", "/dev/null/invalid1")
	os.Setenv("XDG_CACHE_HOME", "/dev/null/invalid2")
	os.Setenv("HOME", "/dev/null/invalid3")

	result := GetGOgentDir()

	// Should fallback to /tmp/gogent-fallback
	if !strings.Contains(result, "gogent-fallback") {
		t.Errorf("Expected /tmp fallback containing 'gogent-fallback', got: %s", result)
	}
	if !strings.HasPrefix(result, os.TempDir()) {
		t.Errorf("Expected path to start with TempDir (%s), got: %s", os.TempDir(), result)
	}
}

func TestGetGOgentDir_HomeDirFails(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Unset all XDG vars so it tries to use HOME
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_CACHE_HOME")

	// Set HOME to invalid path that will fail MkdirAll
	os.Setenv("HOME", "/dev/null/nohome")

	result := GetGOgentDir()

	// Should fallback to /tmp/gogent-fallback
	if !strings.Contains(result, "gogent-fallback") {
		t.Errorf("Expected /tmp fallback, got: %s", result)
	}
}

func TestInitializeToolCounter_ErrorPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set to invalid path to trigger WriteFile error
	os.Setenv("XDG_RUNTIME_DIR", "/dev/null/invalid-write")
	os.Unsetenv("XDG_CACHE_HOME")

	// This will use the fallback path which should succeed
	// To actually test the error path, we'd need to create the directory
	// but make it read-only. Let's do that:
	testDir := t.TempDir()
	gogentDir := filepath.Join(testDir, "gogent")
	os.MkdirAll(gogentDir, 0755)

	// Make directory read-only
	os.Chmod(gogentDir, 0444)
	defer os.Chmod(gogentDir, 0755) // Restore for cleanup

	os.Setenv("XDG_RUNTIME_DIR", testDir)

	// Initialize should fail due to read-only directory
	err := InitializeToolCounter()
	if err == nil {
		t.Error("Expected error when writing to read-only directory, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize tool counter") {
		t.Errorf("Expected 'failed to initialize tool counter' error, got: %v", err)
	}
}

func TestGetToolCountAndIncrement(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// First increment
	count, err := GetToolCountAndIncrement()
	if err != nil {
		t.Fatalf("GetToolCountAndIncrement failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got: %d", count)
	}

	// Second increment
	count, err = GetToolCountAndIncrement()
	if err != nil {
		t.Fatalf("GetToolCountAndIncrement failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got: %d", count)
	}

	// Third increment
	count, err = GetToolCountAndIncrement()
	if err != nil {
		t.Fatalf("GetToolCountAndIncrement failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got: %d", count)
	}
}

func TestShouldRemind(t *testing.T) {
	tests := []struct {
		count    int
		expected bool
	}{
		{0, false},
		{5, false},
		{9, false},
		{10, true}, // ReminderInterval = 10
		{11, false},
		{19, false},
		{20, true},
		{30, true},
		{100, true},
	}

	for _, tc := range tests {
		got := ShouldRemind(tc.count)
		if got != tc.expected {
			t.Errorf("ShouldRemind(%d) = %v, expected %v", tc.count, got, tc.expected)
		}
	}
}

func TestShouldFlush(t *testing.T) {
	tests := []struct {
		count    int
		expected bool
	}{
		{0, false},
		{10, false},
		{19, false},
		{20, true}, // FlushInterval = 20
		{21, false},
		{40, true},
		{60, true},
	}

	for _, tc := range tests {
		got := ShouldFlush(tc.count)
		if got != tc.expected {
			t.Errorf("ShouldFlush(%d) = %v, expected %v", tc.count, got, tc.expected)
		}
	}
}

func TestCounterAtomicity(t *testing.T) {
	// Test concurrent increments
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := GetToolCountAndIncrement()
			if err != nil {
				t.Errorf("Concurrent increment failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify final count
	counterPath := GetToolCounterPath()
	data, _ := os.ReadFile(counterPath)
	finalCount, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	if finalCount != goroutines {
		t.Errorf("Expected final count %d, got %d (atomicity failure)", goroutines, finalCount)
	}
}

func TestGetToolCountAndIncrement_FromNonExistent(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Don't initialize - file doesn't exist
	// GetToolCountAndIncrement should create file and set to 1
	count, err := GetToolCountAndIncrement()
	if err != nil {
		t.Fatalf("GetToolCountAndIncrement failed on non-existent file: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after first increment from non-existent, got %d", count)
	}
}

func TestGetToolCountAndIncrement_EmptyInitialContent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create empty file
	path := GetToolCounterPath()
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// GetToolCountAndIncrement should treat empty as 0 and increment to 1
	count, err := GetToolCountAndIncrement()
	if err != nil {
		t.Fatalf("GetToolCountAndIncrement failed on empty file: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after incrementing empty file, got %d", count)
	}
}
