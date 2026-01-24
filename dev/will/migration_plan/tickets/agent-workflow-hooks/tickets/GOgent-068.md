---
id: GOgent-068
title: Tool Counter Threshold Functions
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-056"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-068: Tool Counter Threshold Functions

**Time**: 1 hour
**Dependencies**: GOgent-056 (counter initialization pattern)

**Task**:
Extend existing tool counter in `pkg/config/paths.go` with threshold check functions for attention-gate triggering. DO NOT create new package.

**CRITICAL**: This ticket originally proposed creating `pkg/observability/counter.go`, but pkg/config/paths.go ALREADY HAS a working tool counter implementation (lines 80-168) with XDG-compliant paths and syscall.Flock atomicity. We only need to ADD threshold functions.

**File**: `pkg/config/paths.go` (EXTEND existing file)

**Implementation Note - Counter Reset**:

The tool counter must be reset on SessionStart to avoid cross-session state pollution. This should be handled by `cmd/gogent-load-context/main.go` (SessionStart handler):

```go
// Required addition to cmd/gogent-load-context/main.go (SessionStart handler):
// Reset tool counter for new session
if err := config.InitializeToolCounter(); err != nil {
    fmt.Fprintf(os.Stderr, "[load-context] Warning: counter reset failed: %v\n", err)
}
```

**Existing Implementation** (already in pkg/config/paths.go):
```go
// GetToolCounterPath returns path to tool counter file.
func GetToolCounterPath() string {
	return filepath.Join(GetGOgentDir(), "tool-counter")
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

	// ... increment logic (validated, working)
}
```

**New Functions to ADD** (add these to pkg/config/paths.go):
```go
const (
	ReminderInterval = 10  // Inject reminder every N tools
	FlushInterval    = 20  // Flush learnings every N tools
)

// GetToolCountAndIncrement atomically reads current count and increments.
// Returns the count AFTER incrementing.
// Uses existing flock pattern from IncrementToolCount for atomicity.
func GetToolCountAndIncrement() (int, error) {
	path := GetToolCounterPath()

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("[config] failed to open counter file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return 0, fmt.Errorf("[config] failed to lock counter file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Read current count
	var count int
	data, err := io.ReadAll(file)
	if err == nil && len(data) > 0 {
		count, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}

	// Increment
	count++

	// Write back
	if err := file.Truncate(0); err != nil {
		return 0, fmt.Errorf("[config] failed to truncate counter: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, fmt.Errorf("[config] failed to seek counter: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(count)); err != nil {
		return 0, fmt.Errorf("[config] failed to write counter: %w", err)
	}

	return count, nil
}

// ShouldRemind returns true if reminder should be injected at this count.
func ShouldRemind(count int) bool {
	return count > 0 && count%ReminderInterval == 0
}

// ShouldFlush returns true if pending learnings should be flushed at this count.
func ShouldFlush(count int) bool {
	return count > 0 && count%FlushInterval == 0
}
```

**Tests**: `pkg/config/paths_test.go` (add to existing test file)

```go
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
}

func TestShouldRemind(t *testing.T) {
	tests := []struct {
		count    int
		expected bool
	}{
		{0, false},
		{5, false},
		{9, false},
		{10, true},   // ReminderInterval = 10
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
		{20, true},   // FlushInterval = 20
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
```

**Acceptance Criteria**:
- [ ] Functions added to pkg/config/paths.go (NOT new package pkg/observability)
- [ ] Uses existing syscall.Flock() atomicity pattern (NOT mutex)
- [ ] GetToolCountAndIncrement() atomically reads and increments
- [ ] ShouldRemind() returns true every 10 tools
- [ ] ShouldFlush() returns true every 20 tools
- [ ] Tool counter reset on SessionStart via gogent-load-context
- [ ] Counter reset errors are non-fatal (warning only)
- [ ] Tests use t.TempDir() for isolation
- [ ] Tests verify atomicity with concurrent goroutines
- [ ] Tests added to pkg/config/paths_test.go (NOT new test file)
- [ ] `go test ./pkg/config` passes

**Why This Matters**: Threshold functions trigger attention-gate behavior at correct intervals. Extending existing implementation maintains architectural consistency and avoids duplication.

**References**:
- Existing counter: pkg/config/paths.go lines 80-168
- GAP Analysis: REFACTORING-MAP.md Section 2, GOgent-068
- Duplication analysis: GAP-ANALYSIS Appendix A

---
