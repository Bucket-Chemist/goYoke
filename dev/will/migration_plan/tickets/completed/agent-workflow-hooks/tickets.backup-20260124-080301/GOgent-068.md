---
id: GOgent-068
title: Tool Counter Management
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-056"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-068: Tool Counter Management

**Time**: 1.5 hours
**Dependencies**: GOgent-056 (counter initialization pattern)

**Task**:
Manage persistent tool call counter for attention-gate triggering.

**File**: `pkg/observability/counter.go`

**Imports**:
```go
package observability

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)
```

**Implementation**:
```go
// ToolCounter manages tool call counting for attention-gate
type ToolCounter struct {
	filepath string
	mu       sync.Mutex
}

const (
	COUNTER_FILE = "/tmp/claude-tool-counter"
	REMINDER_INTERVAL = 10  // Inject reminder every N tools
	FLUSH_INTERVAL    = 20  // Flush learnings every N tools
)

// NewToolCounter creates counter instance
func NewToolCounter() *ToolCounter {
	return &ToolCounter{
		filepath: COUNTER_FILE,
	}
}

// Increment adds 1 to counter and returns new value
func (tc *ToolCounter) Increment() (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	current, err := tc.read()
	if err != nil {
		return 0, err
	}

	next := current + 1

	if err := tc.write(next); err != nil {
		return 0, err
	}

	return next, nil
}

// Read returns current counter value
func (tc *ToolCounter) Read() (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.read()
}

// Reset sets counter to 0
func (tc *ToolCounter) Reset() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.write(0)
}

// ShouldRemind returns true if reminder should be injected
func (tc *ToolCounter) ShouldRemind(currentCount int) bool {
	return currentCount > 0 && currentCount%REMINDER_INTERVAL == 0
}

// ShouldFlush returns true if pending learnings should be flushed
func (tc *ToolCounter) ShouldFlush(currentCount int) bool {
	return currentCount > 0 && currentCount%FLUSH_INTERVAL == 0
}

// read reads counter from file (not thread-safe, use with lock)
func (tc *ToolCounter) read() (int, error) {
	data, err := os.ReadFile(tc.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize if missing
			return 0, nil
		}
		return 0, fmt.Errorf("[attention-gate] Failed to read counter: %w", err)
	}

	count, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("[attention-gate] Failed to parse counter: %w", err)
	}

	return count, nil
}

// write writes counter to file (not thread-safe, use with lock)
func (tc *ToolCounter) write(count int) error {
	if err := os.WriteFile(tc.filepath, []byte(strconv.Itoa(count)), 0644); err != nil {
		return fmt.Errorf("[attention-gate] Failed to write counter: %w", err)
	}
	return nil
}
```

**Tests**: `pkg/observability/counter_test.go`

```go
package observability

import (
	"os"
	"testing"
)

func TestToolCounter_Increment(t *testing.T) {
	// Clean up any existing counter
	os.Remove(COUNTER_FILE)

	counter := NewToolCounter()

	// First increment
	val, err := counter.Increment()
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}

	if val != 1 {
		t.Errorf("Expected 1, got: %d", val)
	}

	// Second increment
	val, err = counter.Increment()
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}

	if val != 2 {
		t.Errorf("Expected 2, got: %d", val)
	}

	// Cleanup
	os.Remove(COUNTER_FILE)
}

func TestToolCounter_Read(t *testing.T) {
	os.Remove(COUNTER_FILE)

	counter := NewToolCounter()

	// Before any increments
	val, err := counter.Read()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if val != 0 {
		t.Errorf("Expected 0 on missing file, got: %d", val)
	}

	os.Remove(COUNTER_FILE)
}

func TestToolCounter_ShouldRemind(t *testing.T) {
	tests := []struct {
		count          int
		shouldRemind   bool
	}{
		{0, false},
		{1, false},
		{9, false},
		{10, true},  // REMINDER_INTERVAL = 10
		{11, false},
		{20, true},  // Also multiple of 10
		{30, true},
	}

	counter := NewToolCounter()

	for _, tc := range tests {
		if got := counter.ShouldRemind(tc.count); got != tc.shouldRemind {
			t.Errorf("ShouldRemind(%d) = %v, expected %v", tc.count, got, tc.shouldRemind)
		}
	}
}

func TestToolCounter_ShouldFlush(t *testing.T) {
	tests := []struct {
		count       int
		shouldFlush bool
	}{
		{0, false},
		{1, false},
		{19, false},
		{20, true},  // FLUSH_INTERVAL = 20
		{21, false},
		{40, true},  // Also multiple of 20
	}

	counter := NewToolCounter()

	for _, tc := range tests {
		if got := counter.ShouldFlush(tc.count); got != tc.shouldFlush {
			t.Errorf("ShouldFlush(%d) = %v, expected %v", tc.count, got, tc.shouldFlush)
		}
	}
}

func TestToolCounter_Reset(t *testing.T) {
	os.Remove(COUNTER_FILE)

	counter := NewToolCounter()

	// Increment a few times
	counter.Increment()
	counter.Increment()
	counter.Increment()

	// Reset
	if err := counter.Reset(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	// Verify
	val, _ := counter.Read()
	if val != 0 {
		t.Errorf("Expected 0 after reset, got: %d", val)
	}

	os.Remove(COUNTER_FILE)
}
```

**Acceptance Criteria**:
- [ ] `Increment()` reads, increments, and writes counter atomically
- [ ] Thread-safe with mutex lock
- [ ] `Read()` returns current value
- [ ] `Reset()` sets counter to 0
- [ ] `ShouldRemind()` returns true every 10 tools
- [ ] `ShouldFlush()` returns true every 20 tools
- [ ] Handles missing file gracefully (defaults to 0)
- [ ] Tests verify increment, read, thresholds
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Counter management is core to attention-gate triggering. Must be thread-safe and persistent across tool calls.

---
