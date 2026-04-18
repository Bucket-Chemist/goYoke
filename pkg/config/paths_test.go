package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestGetgoYokeDir_XDG_RUNTIME_DIR(t *testing.T) {
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

	result := GetgoYokeDir()
	expected := filepath.Join(testDir, "goyoke")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetgoYokeDir to create directory")
	}
}

func TestGetgoYokeDir_XDG_CACHE_HOME(t *testing.T) {
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

	result := GetgoYokeDir()
	expected := filepath.Join(testDir, "goyoke")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetgoYokeDir to create directory")
	}
}

func TestGetgoYokeDir_FallsBackWhenRuntimeDirIsNotWritable(t *testing.T) {
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	runtimeRoot := t.TempDir()
	cacheRoot := t.TempDir()
	runtimeGoyoke := filepath.Join(runtimeRoot, "goyoke")
	if err := os.MkdirAll(runtimeGoyoke, 0755); err != nil {
		t.Fatalf("Failed to create runtime dir: %v", err)
	}
	if err := os.Chmod(runtimeGoyoke, 0555); err != nil {
		t.Fatalf("Failed to chmod runtime dir: %v", err)
	}
	defer os.Chmod(runtimeGoyoke, 0755) //nolint:errcheck

	probePath := filepath.Join(runtimeGoyoke, "probe")
	if err := os.WriteFile(probePath, []byte("x"), 0644); err == nil {
		_ = os.Remove(probePath)
		t.Skip("filesystem permissions do not block writes in this environment")
	}

	os.Setenv("XDG_RUNTIME_DIR", runtimeRoot)
	os.Setenv("XDG_CACHE_HOME", cacheRoot)

	result := GetgoYokeDir()
	expected := filepath.Join(cacheRoot, "goyoke")
	if result != expected {
		t.Fatalf("Expected fallback to %s, got %s", expected, result)
	}
}

func TestGetgoYokeDir_Fallback(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Unset both XDG vars (fallback to ~/.cache/goyoke)
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_CACHE_HOME")
	home := t.TempDir()
	os.Setenv("HOME", home)

	result := GetgoYokeDir()
	expected := filepath.Join(home, ".cache", "goyoke")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetgoYokeDir_EmptyXDGVars(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Set to empty strings (should fallback)
	os.Setenv("XDG_RUNTIME_DIR", "")
	os.Setenv("XDG_CACHE_HOME", "")
	home := t.TempDir()
	os.Setenv("HOME", home)

	result := GetgoYokeDir()
	expected := filepath.Join(home, ".cache", "goyoke")

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
	expected := filepath.Join(testDir, "goyoke", "current-tier")

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
	expected := filepath.Join(testDir, "goyoke", "max_delegation")

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
	expected := filepath.Join(testDir, "goyoke", "routing-violations.jsonl")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify filename ends with .jsonl
	if !strings.HasSuffix(result, ".jsonl") {
		t.Error("Expected violations log to have .jsonl extension")
	}
}

func TestGetgoYokeDir_PriorityOrder(t *testing.T) {
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

	result := GetgoYokeDir()
	expected := filepath.Join(runtimeDir, "goyoke")

	if result != expected {
		t.Errorf("XDG_RUNTIME_DIR should have priority. Expected %s, got %s", expected, result)
	}
}

func TestGetgoYokeDir_CreatesDirectory(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	goyokePath := filepath.Join(testDir, "goyoke")

	// Ensure directory doesn't exist yet
	os.RemoveAll(goyokePath)

	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetgoYokeDir()

	// Verify directory was created
	info, err := os.Stat(result)
	if os.IsNotExist(err) {
		t.Error("GetgoYokeDir should create directory if it doesn't exist")
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Error("GetgoYokeDir should create a directory, not a file")
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
			expected:   "/home/user/my-project/.goyoke/memory/routing-violations.jsonl",
		},
		{
			name:       "relative path",
			projectDir: "my-project",
			expected:   "my-project/.goyoke/memory/routing-violations.jsonl",
		},
		{
			name:       "nested project",
			projectDir: "/home/user/workspace/nested/project",
			expected:   "/home/user/workspace/nested/project/.goyoke/memory/routing-violations.jsonl",
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

			// Verify path contains .goyoke/memory
			if !strings.Contains(result, ".goyoke/memory") {
				t.Error("Expected path to contain .goyoke/memory")
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
	expected := filepath.Join(testDir, "goyoke", "tool-counter")

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

func TestGetgoYokeDir_XDGRuntimeDirInvalidPath(t *testing.T) {
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

	result := GetgoYokeDir()
	expected := filepath.Join(testDir, "goyoke")

	if result != expected {
		t.Errorf("Expected fallback to XDG_CACHE_HOME: %s, got %s", expected, result)
	}
}

func TestGetgoYokeDir_BothXDGFail(t *testing.T) {
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

	// Set valid HOME so we can test the .cache/goyoke fallback
	testDir := t.TempDir()
	os.Setenv("HOME", testDir)

	result := GetgoYokeDir()
	expected := filepath.Join(testDir, ".cache", "goyoke")

	if result != expected {
		t.Errorf("Expected fallback to HOME/.cache/goyoke: %s, got %s", expected, result)
	}
}

func TestGetgoYokeDir_AllPathsFail(t *testing.T) {
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

	result := GetgoYokeDir()

	// Should fallback to /tmp/goyoke-fallback
	if !strings.Contains(result, "goyoke-fallback") {
		t.Errorf("Expected /tmp fallback containing 'goyoke-fallback', got: %s", result)
	}
	if !strings.HasPrefix(result, os.TempDir()) {
		t.Errorf("Expected path to start with TempDir (%s), got: %s", os.TempDir(), result)
	}
	if !strings.HasSuffix(result, fmt.Sprintf("-%d", os.Getuid())) {
		t.Errorf("Expected per-user fallback suffix, got: %s", result)
	}
	info, err := os.Stat(result)
	if err != nil {
		t.Fatalf("Expected fallback dir to exist: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("Expected fallback permissions 0700, got %o", info.Mode().Perm())
	}
}

func TestGetgoYokeDir_HomeDirFails(t *testing.T) {
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

	result := GetgoYokeDir()

	// Should fallback to /tmp/goyoke-fallback
	if !strings.Contains(result, "goyoke-fallback") {
		t.Errorf("Expected /tmp fallback, got: %s", result)
	}
	if !strings.HasSuffix(result, fmt.Sprintf("-%d", os.Getuid())) {
		t.Errorf("Expected per-user fallback suffix, got: %s", result)
	}
}

func TestInitializeToolCounter_FallsBackFromReadOnlyRuntimeDir(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	testDir := t.TempDir()
	goyokeDir := filepath.Join(testDir, "goyoke")
	if err := os.MkdirAll(goyokeDir, 0755); err != nil {
		t.Fatalf("Failed to create runtime dir: %v", err)
	}

	if err := os.Chmod(goyokeDir, 0444); err != nil {
		t.Fatalf("Failed to chmod runtime dir: %v", err)
	}
	defer os.Chmod(goyokeDir, 0755) //nolint:errcheck

	os.Setenv("XDG_RUNTIME_DIR", testDir)
	cacheDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	counterPath := filepath.Join(cacheDir, "goyoke", "tool-counter")
	if err := InitializeToolCounter(); err != nil {
		t.Fatalf("Expected fallback initialization to succeed, got: %v", err)
	}
	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("Expected counter file at fallback path %s: %v", counterPath, err)
	}
	if string(data) != "0" {
		t.Fatalf("Expected initialized counter to be 0, got %q", string(data))
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

// Tests for GetgoYokeDataDir() and related path helpers

func TestGetgoYokeDataDir_XDGSet(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := t.TempDir()
	os.Setenv("XDG_DATA_HOME", testPath)

	dir := GetgoYokeDataDir()
	expected := filepath.Join(testPath, "goyoke")

	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	// Verify directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Expected GetgoYokeDataDir to create directory")
	}
}

func TestGetgoYokeDataDir_Fallback(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	os.Unsetenv("XDG_DATA_HOME")
	home := t.TempDir()
	os.Setenv("HOME", home)

	dir := GetgoYokeDataDir()
	expected := filepath.Join(home, ".local", "share", "goyoke")

	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	// Verify directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Expected GetgoYokeDataDir to create directory in fallback location")
	}
}

func TestGetgoYokeDataDir_CreatesDirectory(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testDir := t.TempDir()
	goyokePath := filepath.Join(testDir, "goyoke")

	// Ensure directory doesn't exist yet
	os.RemoveAll(goyokePath)

	os.Setenv("XDG_DATA_HOME", testDir)

	result := GetgoYokeDataDir()

	// Verify directory was created
	info, err := os.Stat(result)
	if os.IsNotExist(err) {
		t.Error("GetgoYokeDataDir should create directory if it doesn't exist")
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Error("GetgoYokeDataDir should create a directory, not a file")
	}

	// Verify permissions (0755)
	if info.Mode().Perm() != 0755 {
		t.Errorf("Expected permissions 0755, got %o", info.Mode().Perm())
	}
}

func TestGetgoYokeDataDir_EmptyXDGVar(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Set to empty string (should fallback)
	os.Setenv("XDG_DATA_HOME", "")
	home := t.TempDir()
	os.Setenv("HOME", home)

	dir := GetgoYokeDataDir()
	expected := filepath.Join(home, ".local", "share", "goyoke")

	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestGetgoYokeDataDir_XDGInvalidPath(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Set XDG_DATA_HOME to a path that cannot be created
	os.Setenv("XDG_DATA_HOME", "/dev/null/cannot-create-here")
	home := t.TempDir()
	os.Setenv("HOME", home)

	dir := GetgoYokeDataDir()
	expected := filepath.Join(home, ".local", "share", "goyoke")

	if dir != expected {
		t.Errorf("Expected fallback to ~/.local/share/goyoke: %s, got %s", expected, dir)
	}
}

func TestGetgoYokeDataDir_HomeDirFails(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Unset XDG_DATA_HOME
	os.Unsetenv("XDG_DATA_HOME")

	// Set HOME to invalid path that will fail MkdirAll
	os.Setenv("HOME", "/dev/null/nohome")

	result := GetgoYokeDataDir()

	// Should fallback to /tmp/goyoke-data
	if !strings.Contains(result, "goyoke-data") {
		t.Errorf("Expected /tmp fallback containing 'goyoke-data', got: %s", result)
	}
	if !strings.HasPrefix(result, os.TempDir()) {
		t.Errorf("Expected path to start with TempDir (%s), got: %s", os.TempDir(), result)
	}
	if !strings.HasSuffix(result, fmt.Sprintf("-%d", os.Getuid())) {
		t.Errorf("Expected per-user fallback suffix, got: %s", result)
	}
	info, err := os.Stat(result)
	if err != nil {
		t.Fatalf("Expected fallback dir to exist: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("Expected fallback permissions 0700, got %o", info.Mode().Perm())
	}
}

func TestGetgoYokeDataDir_AllPathsFail(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Set XDG_DATA_HOME to invalid path
	os.Setenv("XDG_DATA_HOME", "/dev/null/invalid1")
	// Set HOME to invalid path
	os.Setenv("HOME", "/dev/null/invalid2")

	result := GetgoYokeDataDir()

	// Should fallback to /tmp/goyoke-data
	if !strings.Contains(result, "goyoke-data") {
		t.Errorf("Expected /tmp fallback containing 'goyoke-data', got: %s", result)
	}
	if !strings.HasPrefix(result, os.TempDir()) {
		t.Errorf("Expected path to start with TempDir (%s), got: %s", os.TempDir(), result)
	}
	if !strings.HasSuffix(result, fmt.Sprintf("-%d", os.Getuid())) {
		t.Errorf("Expected per-user fallback suffix, got: %s", result)
	}
}

func TestGetgoYokeDataDir_SeparateFromCache(t *testing.T) {
	// Save original env
	origXDGData := os.Getenv("XDG_DATA_HOME")
	origXDGCache := os.Getenv("XDG_CACHE_HOME")
	origXDGRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDGData)
		os.Setenv("XDG_CACHE_HOME", origXDGCache)
		os.Setenv("XDG_RUNTIME_DIR", origXDGRuntime)
	}()

	// Set up separate directories for data and cache
	dataTempDir := t.TempDir()
	cacheTempDir := t.TempDir()
	runtimeTempDir := t.TempDir()

	os.Setenv("XDG_DATA_HOME", dataTempDir)
	os.Setenv("XDG_CACHE_HOME", cacheTempDir)
	os.Setenv("XDG_RUNTIME_DIR", runtimeTempDir)

	dataDir := GetgoYokeDataDir()
	cacheDir := GetgoYokeDir()

	// Verify they use different base directories
	if strings.HasPrefix(dataDir, runtimeTempDir) || strings.HasPrefix(dataDir, cacheTempDir) {
		t.Errorf("GetgoYokeDataDir should not use cache/runtime dirs. Got: %s", dataDir)
	}

	if strings.HasPrefix(cacheDir, dataTempDir) {
		t.Errorf("GetgoYokeDir should not use data dir. Got: %s", cacheDir)
	}

	// Verify data dir uses XDG_DATA_HOME
	expectedDataDir := filepath.Join(dataTempDir, "goyoke")
	if dataDir != expectedDataDir {
		t.Errorf("Expected data dir %s, got %s", expectedDataDir, dataDir)
	}

	// Verify cache dir uses XDG_RUNTIME_DIR (highest priority for cache)
	expectedCacheDir := filepath.Join(runtimeTempDir, "goyoke")
	if cacheDir != expectedCacheDir {
		t.Errorf("Expected cache dir %s, got %s", expectedCacheDir, cacheDir)
	}
}

func TestGetMLToolEventsPath(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := t.TempDir()
	os.Setenv("XDG_DATA_HOME", testPath)

	path := GetMLToolEventsPath()

	// Verify path ends with tool-events.jsonl
	if !strings.HasSuffix(path, "tool-events.jsonl") {
		t.Errorf("Path should end with tool-events.jsonl, got %s", path)
	}

	// Verify path is within data directory
	dataDir := GetgoYokeDataDir()
	if !strings.HasPrefix(path, dataDir) {
		t.Errorf("Expected path to be within data directory %s, got %s", dataDir, path)
	}

	// Verify exact path
	expected := filepath.Join(testPath, "goyoke", "tool-events.jsonl")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestGetRoutingDecisionsPath(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := t.TempDir()
	os.Setenv("XDG_DATA_HOME", testPath)

	path := GetRoutingDecisionsPath()

	// Verify path ends with routing-decisions.jsonl
	if !strings.HasSuffix(path, "routing-decisions.jsonl") {
		t.Errorf("Path should end with routing-decisions.jsonl, got %s", path)
	}

	// Verify path is within data directory
	dataDir := GetgoYokeDataDir()
	if !strings.HasPrefix(path, dataDir) {
		t.Errorf("Expected path to be within data directory %s, got %s", dataDir, path)
	}

	// Verify exact path
	expected := filepath.Join(testPath, "goyoke", "routing-decisions.jsonl")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestGetCollaborationsPath(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := t.TempDir()
	os.Setenv("XDG_DATA_HOME", testPath)

	path := GetCollaborationsPath()

	// Verify path ends with agent-collaborations.jsonl
	if !strings.HasSuffix(path, "agent-collaborations.jsonl") {
		t.Errorf("Path should end with agent-collaborations.jsonl, got %s", path)
	}

	// Verify path is within data directory
	dataDir := GetgoYokeDataDir()
	if !strings.HasPrefix(path, dataDir) {
		t.Errorf("Expected path to be within data directory %s, got %s", dataDir, path)
	}

	// Verify exact path
	expected := filepath.Join(testPath, "goyoke", "agent-collaborations.jsonl")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestMLPathHelpers_AllUseDataDir(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := t.TempDir()
	os.Setenv("XDG_DATA_HOME", testPath)

	dataDir := GetgoYokeDataDir()

	// All ML path helpers should use GetgoYokeDataDir(), not GetgoYokeDir()
	paths := []struct {
		name string
		path string
	}{
		{"GetMLToolEventsPath", GetMLToolEventsPath()},
		{"GetRoutingDecisionsPath", GetRoutingDecisionsPath()},
		{"GetCollaborationsPath", GetCollaborationsPath()},
	}

	for _, tc := range paths {
		if !strings.HasPrefix(tc.path, dataDir) {
			t.Errorf("%s should be within data directory %s, got %s", tc.name, dataDir, tc.path)
		}
	}
}

func TestMLPathHelpers_Fallback(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Unset XDG_DATA_HOME to test fallback behavior
	os.Unsetenv("XDG_DATA_HOME")
	home := t.TempDir()
	os.Setenv("HOME", home)

	expectedDataDir := filepath.Join(home, ".local", "share", "goyoke")

	paths := []struct {
		name     string
		path     string
		filename string
	}{
		{"GetMLToolEventsPath", GetMLToolEventsPath(), "tool-events.jsonl"},
		{"GetRoutingDecisionsPath", GetRoutingDecisionsPath(), "routing-decisions.jsonl"},
		{"GetCollaborationsPath", GetCollaborationsPath(), "agent-collaborations.jsonl"},
	}

	for _, tc := range paths {
		expected := filepath.Join(expectedDataDir, tc.filename)
		if tc.path != expected {
			t.Errorf("%s with fallback: expected %s, got %s", tc.name, expected, tc.path)
		}
	}
}

func TestGetgoYokeDataDir_FallsBackWhenDataDirIsNotWritable(t *testing.T) {
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	xdgDataHome := t.TempDir()
	readOnlyParent := filepath.Join(xdgDataHome, "goyoke")
	if err := os.MkdirAll(readOnlyParent, 0755); err != nil {
		t.Fatalf("Failed to create read-only parent: %v", err)
	}
	if err := os.Chmod(readOnlyParent, 0555); err != nil {
		t.Fatalf("Failed to chmod read-only parent: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyParent, 0755)
	})

	home := t.TempDir()
	os.Setenv("XDG_DATA_HOME", xdgDataHome)
	os.Setenv("HOME", home)

	dir := GetgoYokeDataDir()
	expected := filepath.Join(home, ".local", "share", "goyoke")
	if dir != expected {
		t.Fatalf("Expected fallback to %s, got %s", expected, dir)
	}

	testFile := filepath.Join(dir, "telemetry.jsonl")
	if err := os.WriteFile(testFile, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("Expected fallback data dir to be writable: %v", err)
	}
}

func TestRuntimeDir_Default(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()
	os.Unsetenv("GOYOKE_RUNTIME_DIR")

	projectDir := t.TempDir()
	result := RuntimeDir(projectDir)
	expected := filepath.Join(projectDir, ".goyoke")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected RuntimeDir to create directory")
	}
}

func TestRuntimeDir_EnvOverride(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()

	overrideDir := t.TempDir()
	os.Setenv("GOYOKE_RUNTIME_DIR", overrideDir)

	projectDir := t.TempDir()
	result := RuntimeDir(projectDir)

	if result != overrideDir {
		t.Errorf("Expected GOYOKE_RUNTIME_DIR override %s, got %s", overrideDir, result)
	}
}

func TestProjectMemoryDir(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()
	os.Unsetenv("GOYOKE_RUNTIME_DIR")

	projectDir := t.TempDir()
	result := ProjectMemoryDir(projectDir)
	expected := filepath.Join(projectDir, ".goyoke", "memory")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected ProjectMemoryDir to create directory")
	}
}

func TestGetProjectViolationsLogPath_UsesGoyokeMemory(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()
	os.Unsetenv("GOYOKE_RUNTIME_DIR")

	projectDir := t.TempDir()
	result := GetProjectViolationsLogPath(projectDir)
	expected := filepath.Join(projectDir, ".goyoke", "memory", "routing-violations.jsonl")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestRuntimeDir_WriteCanary validates that .goyoke/ paths are writable.
// This is the migration canary — if .goyoke/ writes fail, the .claude/ → .goyoke/
// runtime I/O migration is broken.
func TestRuntimeDir_WriteCanary(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()
	os.Unsetenv("GOYOKE_RUNTIME_DIR")

	projectDir := t.TempDir()

	// Write to RuntimeDir
	runtimeDir := RuntimeDir(projectDir)
	testFile := filepath.Join(runtimeDir, "canary.txt")
	if err := os.WriteFile(testFile, []byte("canary"), 0644); err != nil {
		t.Fatalf("Failed to write to RuntimeDir: %v", err)
	}

	// Write to ProjectMemoryDir
	memDir := ProjectMemoryDir(projectDir)
	memFile := filepath.Join(memDir, "canary.jsonl")
	if err := os.WriteFile(memFile, []byte(`{"test":true}`+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write to ProjectMemoryDir: %v", err)
	}

	// Verify reads back
	data, err := os.ReadFile(memFile)
	if err != nil {
		t.Fatalf("Failed to read back canary file: %v", err)
	}
	if string(data) != `{"test":true}`+"\n" {
		t.Errorf("Canary data mismatch: got %q", string(data))
	}
}

func TestRuntimeDir_Idempotent(t *testing.T) {
	orig := os.Getenv("GOYOKE_RUNTIME_DIR")
	defer func() { os.Setenv("GOYOKE_RUNTIME_DIR", orig) }()
	os.Unsetenv("GOYOKE_RUNTIME_DIR")

	projectDir := t.TempDir()
	result1 := RuntimeDir(projectDir)
	result2 := RuntimeDir(projectDir)

	if result1 != result2 {
		t.Errorf("Expected idempotent results, got %s and %s", result1, result2)
	}
	if _, err := os.Stat(result1); os.IsNotExist(err) {
		t.Error("Expected directory to exist after second call")
	}
}

func TestGetGuardsDir_ReturnsPathUndergoYokeDir(t *testing.T) {
	testDir := t.TempDir()
	orig := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", orig)
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	guardsDir := GetGuardsDir()
	goyokeDir := GetgoYokeDir()

	if !strings.HasPrefix(guardsDir, goyokeDir) {
		t.Errorf("GetGuardsDir() %q not under GetgoYokeDir() %q", guardsDir, goyokeDir)
	}
	if filepath.Base(guardsDir) != "guards" {
		t.Errorf("Expected path ending in /guards, got %q", guardsDir)
	}
}

func TestGetGuardsDir_CreatesDirectory(t *testing.T) {
	testDir := t.TempDir()
	orig := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", orig)
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	guardsDir := GetGuardsDir()
	if _, err := os.Stat(guardsDir); os.IsNotExist(err) {
		t.Errorf("Expected GetGuardsDir() to create directory %q", guardsDir)
	}
}

func TestGetGuardFilePath_IncludesSessionID(t *testing.T) {
	testDir := t.TempDir()
	orig := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", orig)
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	sessionID := "test-session-abc123"
	path := GetGuardFilePath(sessionID)

	if !strings.Contains(path, sessionID) {
		t.Errorf("GetGuardFilePath() %q does not contain session ID %q", path, sessionID)
	}
	if filepath.Ext(path) != ".json" {
		t.Errorf("GetGuardFilePath() %q does not end in .json", path)
	}
	expected := filepath.Join(GetGuardsDir(), sessionID+".json")
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}
}

func TestGetGuardLockPath_IncludesSessionID(t *testing.T) {
	testDir := t.TempDir()
	orig := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", orig)
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	sessionID := "test-session-abc123"
	path := GetGuardLockPath(sessionID)

	if !strings.Contains(path, sessionID) {
		t.Errorf("GetGuardLockPath() %q does not contain session ID %q", path, sessionID)
	}
	if filepath.Ext(path) != ".lock" {
		t.Errorf("GetGuardLockPath() %q does not end in .lock", path)
	}
	expected := filepath.Join(GetGuardsDir(), sessionID+".lock")
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}
}
