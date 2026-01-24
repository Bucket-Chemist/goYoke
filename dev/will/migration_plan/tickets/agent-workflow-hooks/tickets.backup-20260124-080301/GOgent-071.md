---
id: GOgent-071
title: Integration Tests for attention-gate
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-070"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-071: Integration Tests for attention-gate

**Time**: 1.5 hours
**Dependencies**: GOgent-070

**Task**:
End-to-end tests for tool counter → reminder/flush workflow.

**File**: `pkg/observability/integration_test.go`

```go
package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttentionGateWorkflow_ReminderAt10(t *testing.T) {
	os.Remove(COUNTER_FILE)
	defer os.Remove(COUNTER_FILE)

	counter := NewToolCounter()

	// Increment to 10 (should trigger reminder)
	for i := 0; i < 10; i++ {
		counter.Increment()
	}

	current, _ := counter.Read()
	if !counter.ShouldRemind(current) {
		t.Error("Should trigger reminder at tool #10")
	}

	summary := "haiku: find... sonnet: implement..."
	reminder := GenerateRoutingReminder(current, summary)

	if !strings.Contains(reminder, "checkpoint") {
		t.Error("Reminder should indicate checkpoint")
	}
}

func TestAttentionGateWorkflow_FlushAt20(t *testing.T) {
	os.Remove(COUNTER_FILE)
	defer os.Remove(COUNTER_FILE)

	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 6; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	counter := NewToolCounter()

	// Increment to 20 (should trigger flush)
	for i := 0; i < 20; i++ {
		counter.Increment()
	}

	current, _ := counter.Read()
	if !counter.ShouldFlush(current) {
		t.Error("Should trigger flush at tool #20")
	}

	shouldFlush, count, _ := ShouldFlushLearnings(tmpDir)
	if !shouldFlush {
		t.Error("Should need flush (count >= 5)")
	}

	ctx, _ := ArchivePendingLearnings(tmpDir)
	if ctx.EntryCount != 6 {
		t.Errorf("Should archive 6 entries, got: %d", ctx.EntryCount)
	}

	notification := GenerateFlushNotification(ctx)
	if !strings.Contains(notification, "6") {
		t.Error("Notification should mention entry count")
	}
}

func TestAttentionGateWorkflow_NoFlushBelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Only 3 entries (below threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{}
{}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, _ := ShouldFlushLearnings(tmpDir)

	if shouldFlush {
		t.Error("Should not flush below threshold")
	}

	if count != 2 {
		t.Errorf("Expected 2 entries, got: %d", count)
	}
}
```

**Acceptance Criteria**:
- [ ] Tool counter increment and threshold checks work
- [ ] Reminder injected at tool #10, #20, #30, etc.
- [ ] Flush only happens at tool #20, #40, etc. AND count >= 5
- [ ] Archive created with timestamp
- [ ] Pending learnings cleared after flush
- [ ] Notification generated correctly
- [ ] Tests verify full workflow
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Integration tests ensure counter → reminder and flush logic works together correctly.

---
