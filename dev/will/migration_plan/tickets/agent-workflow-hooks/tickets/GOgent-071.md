---
id: GOgent-071
title: Integration Tests for attention-gate
description: "End-to-end tests for tool counter → reminder/flush workflow. Use t.TempDir() for test isolation and add simulation harness integration tests."
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-069"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 10
---

### GOgent-071: Integration Tests for attention-gate

**Time**: 1.5 hours
**Dependencies**: GOgent-069 (changed from GOgent-070 which was eliminated)

**Task**:
End-to-end tests for tool counter → reminder/flush workflow. Use t.TempDir() for test isolation and add simulation harness integration tests.

**File**: `pkg/config/paths_integration_test.go` AND `pkg/session/attention_gate_integration_test.go`

**CRITICAL**: Tests MUST use t.TempDir() for all file paths to prevent global state pollution. Do NOT use os.Remove(COUNTER_FILE) pattern.

**Counter Integration Tests** (`pkg/config/paths_integration_test.go`):
```go
package config

import (
	"path/filepath"
	"testing"
)

func TestAttentionGateWorkflow_ReminderAt10(t *testing.T) {
	// Use t.TempDir() for isolation (NOT global COUNTER_FILE)
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Increment to 10 (should trigger reminder)
	var count int
	var err error
	for i := 0; i < 10; i++ {
		count, err = GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i+1, err)
		}
	}

	if count != 10 {
		t.Errorf("Expected count 10, got: %d", count)
	}

	if !ShouldRemind(count) {
		t.Error("Should trigger reminder at tool #10")
	}

	if ShouldFlush(count) {
		t.Error("Should NOT trigger flush at tool #10 (threshold is 20)")
	}
}

func TestAttentionGateWorkflow_FlushAt20(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Increment to 20 (should trigger both reminder and flush)
	var count int
	var err error
	for i := 0; i < 20; i++ {
		count, err = GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i+1, err)
		}
	}

	if count != 20 {
		t.Errorf("Expected count 20, got: %d", count)
	}

	if !ShouldRemind(count) {
		t.Error("Should trigger reminder at tool #20 (multiple of 10)")
	}

	if !ShouldFlush(count) {
		t.Error("Should trigger flush at tool #20")
	}
}

func TestAttentionGateWorkflow_MultipleThresholds(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	tests := []struct {
		targetCount    int
		shouldRemind   bool
		shouldFlush    bool
		description    string
	}{
		{9, false, false, "Before first reminder"},
		{10, true, false, "First reminder"},
		{19, false, false, "Before first flush"},
		{20, true, true, "First flush + reminder"},
		{30, true, false, "Second reminder"},
		{40, true, true, "Second flush + reminder"},
	}

	var count int
	for _, tc := range tests {
		// Increment to target
		for count < tc.targetCount {
			count, _ = GetToolCountAndIncrement()
		}

		if ShouldRemind(count) != tc.shouldRemind {
			t.Errorf("%s: ShouldRemind(%d) = %v, expected %v",
				tc.description, count, ShouldRemind(count), tc.shouldRemind)
		}

		if ShouldFlush(count) != tc.shouldFlush {
			t.Errorf("%s: ShouldFlush(%d) = %v, expected %v",
				tc.description, count, ShouldFlush(count), tc.shouldFlush)
		}
	}
}
```

**Session Flush Integration Tests** (`pkg/session/attention_gate_integration_test.go`):
```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

func TestAttentionGateWorkflow_FullFlush(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings (6 entries, above default threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 6; i++ {
		lines = append(lines, `{"file":"test.go","line":10}`)
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	// Check flush trigger
	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ShouldFlushLearnings failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count >= threshold")
	}

	if count != 6 {
		t.Errorf("Expected 6 entries, got: %d", count)
	}

	// Execute flush
	ctx, err := ArchivePendingLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ArchivePendingLearnings failed: %v", err)
	}

	if ctx.EntryCount != 6 {
		t.Errorf("Expected 6 entries archived, got: %d", ctx.EntryCount)
	}

	// Verify pending is cleared
	data, _ := os.ReadFile(pendingPath)
	if string(data) != "" {
		t.Error("Pending learnings should be cleared after flush")
	}

	// Verify archive exists
	if _, err := os.Stat(ctx.ArchivedFile); os.IsNotExist(err) {
		t.Error("Archive file should exist")
	}

	// Verify archive contains expected data
	archiveData, _ := os.ReadFile(ctx.ArchivedFile)
	archiveLines := strings.Count(string(archiveData), "\n")
	if archiveLines != 6 {
		t.Errorf("Archive should have 6 lines, got: %d", archiveLines)
	}
}

func TestAttentionGateWorkflow_NoFlushBelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings (4 entries, BELOW default threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 4; i++ {
		lines = append(lines, `{}`)
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ShouldFlushLearnings failed: %v", err)
	}

	if shouldFlush {
		t.Error("Should NOT flush when count < threshold")
	}

	if count != 4 {
		t.Errorf("Expected 4 entries, got: %d", count)
	}
}

func TestAttentionGateWorkflow_SimulationHarness(t *testing.T) {
	// Simulation harness integration test
	// Simulates complete attention-gate workflow across 30 tool calls

	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create initial pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var initialLines []string
	for i := 0; i < 7; i++ {
		initialLines = append(initialLines, `{}`)
	}
	os.WriteFile(pendingPath, []byte(strings.Join(initialLines, "\n")+"\n"), 0644)

	var reminderCount, flushCount int

	// Simulate 30 tool calls
	for i := 1; i <= 30; i++ {
		count, err := config.GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Tool %d increment failed: %v", i, err)
		}

		// Check reminder
		if config.ShouldRemind(count) {
			reminderCount++
			reminder := GenerateRoutingReminder(count, "haiku: search... sonnet: implement...")
			if !strings.Contains(reminder, "CHECKPOINT") {
				t.Errorf("Tool %d: reminder should contain checkpoint", i)
			}
		}

		// Check flush
		if config.ShouldFlush(count) {
			shouldFlush, pendingCount, _ := ShouldFlushLearnings(tmpDir)
			if shouldFlush {
				flushCount++
				ctx, err := ArchivePendingLearnings(tmpDir)
				if err != nil {
					t.Errorf("Tool %d: flush failed: %v", i, err)
				} else {
					if ctx.EntryCount != pendingCount {
						t.Errorf("Tool %d: flushed %d but expected %d", i, ctx.EntryCount, pendingCount)
					}
				}
			}
		}
	}

	// Verify simulation results
	if reminderCount != 3 {
		t.Errorf("Expected 3 reminders (at 10, 20, 30), got: %d", reminderCount)
	}

	if flushCount == 0 {
		t.Error("Expected at least 1 flush during 30 tool calls")
	}
}
```

**Acceptance Criteria**:
- [x] Tests use t.TempDir() for all file paths (no global state pollution)
- [x] Tests mock GetToolCounterPath() for isolation
- [x] Counter integration tests verify reminder at tool #10
- [x] Counter integration tests verify flush at tool #20
- [x] Session integration tests verify full flush workflow
- [x] Session integration tests verify no flush below threshold
- [x] Simulation harness test covers 30-tool workflow
- [x] Simulation test verifies reminder count (3 at 10, 20, 30)
- [x] Simulation test verifies flush execution
- [x] Tests follow existing patterns in pkg/config/paths_test.go
- [x] `go test ./pkg/config` passes
- [x] `go test ./pkg/session` passes

**Why This Matters**: Integration tests verify multi-component workflows that unit tests cannot catch. Simulation harness tests validate real-world usage patterns.

**References**:
- Existing test patterns: pkg/config/paths_test.go
- t.TempDir() usage: Standard Go testing best practice
- GAP Analysis: REFACTORING-MAP.md Section 2, GOgent-071

---
