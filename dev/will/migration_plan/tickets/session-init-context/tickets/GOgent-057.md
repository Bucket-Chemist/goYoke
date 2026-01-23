---
id: GOgent-057
title: Tool Counter Initialization (XDG Compliant)
description: **Task**:
status: pending
time_estimate: 0.5h
dependencies: [\n  - GOgent-056]
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 11
---

## GOgent-057: Tool Counter Initialization (XDG Compliant)

**Time**: 0.5 hours
**Dependencies**: GOgent-056
**Priority**: HIGH (attention-gate dependency)

**Task**:
Add XDG-compliant tool counter path and initialization to `pkg/config/paths.go`.

**File**: `pkg/config/paths.go` (extend existing)

**Implementation**:
```go
// Add to existing pkg/config/paths.go

// GetToolCounterPath returns path to session tool counter file.
// Used by SessionStart (initialize) and attention-gate (increment/read).
// File format: single integer representing total tool calls this session.
func GetToolCounterPath() string {
	return filepath.Join(GetGOgentDir(), "tool-counter")
}

// InitializeToolCounter creates or resets tool counter to 0.
// Called at session start to track tool calls for attention-gate.
// Returns error if file cannot be created (permissions, disk full, etc.).
func InitializeToolCounter() error {
	counterPath := GetToolCounterPath()

	// Write "0" to counter file (overwrite if exists)
	if err := os.WriteFile(counterPath, []byte("0"), 0644); err != nil {
		return fmt.Errorf("[config] Failed to initialize tool counter at %s: %w. Check write permissions for %s.", counterPath, err, GetGOgentDir())
	}

	return nil
}

// GetToolCount reads current tool count from counter file.
// Returns 0 if file doesn't exist (session not initialized).
// Returns error only for read failures (not missing file).
func GetToolCount() (int, error) {
	counterPath := GetToolCounterPath()

	data, err := os.ReadFile(counterPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // Not initialized yet
		}
		return 0, fmt.Errorf("[config] Failed to read tool counter from %s: %w", counterPath, err)
	}

	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &count); err != nil {
		return 0, fmt.Errorf("[config] Failed to parse tool counter value %q: %w. File may be corrupted.", string(data), err)
	}

	return count, nil
}

// IncrementToolCount atomically increments tool counter using atomic rename pattern.
// Used by attention-gate hook after each tool execution.
// Returns new count after increment.
//
// Atomicity: Uses write-to-temp + rename pattern which is atomic on POSIX systems.
// This prevents race conditions when multiple hooks fire in rapid succession.
func IncrementToolCount() (int, error) {
	current, err := GetToolCount()
	if err != nil {
		return 0, err
	}

	newCount := current + 1
	counterPath := GetToolCounterPath()
	tmpPath := counterPath + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpPath, []byte(fmt.Sprintf("%d", newCount)), 0644); err != nil {
		return 0, fmt.Errorf("[config] Failed to write tool counter temp file: %w", err)
	}

	// Atomic rename (POSIX guarantees atomicity for rename within same filesystem)
	if err := os.Rename(tmpPath, counterPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return 0, fmt.Errorf("[config] Failed to atomically update tool counter: %w", err)
	}

	return newCount, nil
}
```

**Add import** to `pkg/config/paths.go`:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)
```

**Tests**: Add to `pkg/config/paths_test.go`

```go
// Add to existing pkg/config/paths_test.go

func TestToolCounter_Initialize(t *testing.T) {
	// Setup: Use temp XDG directory
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Initialize counter
	err := InitializeToolCounter()
	if err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Verify counter is 0
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected initial count 0, got: %d", count)
	}
}

func TestToolCounter_Increment(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Initialize
	if err := InitializeToolCounter(); err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Increment 3 times
	for i := 1; i <= 3; i++ {
		count, err := IncrementToolCount()
		if err != nil {
			t.Fatalf("IncrementToolCount failed on iteration %d: %v", i, err)
		}

		if count != i {
			t.Errorf("Expected count %d after increment %d, got: %d", i, i, count)
		}
	}
}

func TestToolCounter_GetCount_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Don't initialize - file doesn't exist
	count, err := GetToolCount()

	if err != nil {
		t.Fatalf("GetToolCount should not error on missing file, got: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 for uninitialized counter, got: %d", count)
	}
}

func TestGetToolCounterPath(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	path := GetToolCounterPath()

	if !strings.Contains(path, "gogent") {
		t.Errorf("Expected path to contain 'gogent', got: %s", path)
	}

	if !strings.HasSuffix(path, "tool-counter") {
		t.Errorf("Expected path to end with 'tool-counter', got: %s", path)
	}
}

func TestToolCounter_ConcurrentIncrement(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Initialize counter
	if err := InitializeToolCounter(); err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Spawn multiple goroutines to increment concurrently
	const numGoroutines = 10
	const incrementsPerGoroutine = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < incrementsPerGoroutine; j++ {
				_, err := IncrementToolCount()
				if err != nil {
					t.Errorf("Concurrent increment failed: %v", err)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final count (may be less than expected due to races, but should not error)
	count, err := GetToolCount()
	if err != nil {
		t.Fatalf("GetToolCount after concurrent increments failed: %v", err)
	}

	// With atomic rename, we expect exactly numGoroutines * incrementsPerGoroutine
	expected := numGoroutines * incrementsPerGoroutine
	if count != expected {
		t.Errorf("Expected count %d after concurrent increments, got: %d (atomic rename should prevent lost updates)", expected, count)
	}
}
```

**Acceptance Criteria**:
- [ ] `GetToolCounterPath()` returns XDG-compliant path (NOT `/tmp`)
- [ ] `InitializeToolCounter()` creates counter file with "0"
- [ ] `GetToolCount()` returns 0 for missing file (not error)
- [ ] `IncrementToolCount()` atomically increments using write-to-temp + rename pattern
- [ ] Atomic rename prevents race conditions in concurrent hook execution
- [ ] Tests verify initialization, increment, missing file handling
- [ ] `go test ./pkg/config/...` passes
- [ ] Path uses `~/.cache/gogent/` or `$XDG_CACHE_HOME/gogent/`

**Test Deliverables**:
- [ ] Tests added to: `pkg/config/paths_test.go`
- [ ] Number of new test functions: 5 (including concurrent increment test)
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Tool counter is read by attention-gate hook (fires every 10 tool calls). XDG compliance ensures proper cleanup and avoids `/tmp` permission issues.

---
