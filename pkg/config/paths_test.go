package config

import (
	"os"
	"path/filepath"
	"strings"
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
