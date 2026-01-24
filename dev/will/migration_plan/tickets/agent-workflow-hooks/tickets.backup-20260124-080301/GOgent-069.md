---
id: GOgent-069
title: Reminder & Flush Logic
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-068"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-069: Reminder & Flush Logic

**Time**: 2 hours
**Dependencies**: GOgent-068

**Task**:
Generate routing compliance reminders and auto-flush pending learnings.

**File**: `pkg/observability/gate.go`

**Imports**:
```go
package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)
```

**Implementation**:
```go
// ReminderContext represents routing reminder context
type ReminderContext struct {
	ToolCount      int    `json:"tool_count"`
	CurrentSession string `json:"session_id"`
	TiersSummary   string `json:"routing_summary"`
}

// FlushContext represents learning flush context
type FlushContext struct {
	EntryCount      int      `json:"entries_flushed"`
	ArchivedFile    string   `json:"archived_to"`
	PendingRemaining int     `json:"remaining_entries"`
}

// GenerateRoutingReminder creates attention-gate reminder message
func GenerateRoutingReminder(toolCount int, routingSummary string) string {
	reminder := fmt.Sprintf(`🔔 ROUTING CHECKPOINT (Tool #%d)

Session routing compliance check:

ACTIVE ROUTING TIERS:
%s

At this checkpoint, verify:
1. ✅ Are you delegating exploratory work to codebase-search?
2. ✅ Are you using haiku for mechanical tasks?
3. ✅ Are you using sonnet for implementation?
4. ✅ Have you scouted unknown-scope tasks first?

If ANY of these need correction, pause and re-route.
See routing-schema.json for complete tier mappings.`,
		toolCount, routingSummary)

	return reminder
}

// CheckPendingLearnings counts entries in pending-learnings.jsonl
func CheckPendingLearnings(projectDir string) (int, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return 0, nil // No pending learnings
	}
	if err != nil {
		return 0, fmt.Errorf("[attention-gate] Failed to read pending learnings: %w", err)
	}

	// Count lines
	lineCount := strings.Count(string(data), "\n")

	return lineCount, nil
}

// ShouldFlushLearnings checks if pending learnings exceed threshold
func ShouldFlushLearnings(projectDir string) (bool, int, error) {
	count, err := CheckPendingLearnings(projectDir)
	if err != nil {
		return false, 0, err
	}

	const FLUSH_THRESHOLD = 5
	return count >= FLUSH_THRESHOLD, count, nil
}

// ArchivePendingLearnings moves entries to timestamped archive
func ArchivePendingLearnings(projectDir string) (*FlushContext, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	sharpEdgesDir := filepath.Join(projectDir, ".claude", "memory", "sharp-edges")

	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return &FlushContext{EntryCount: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[attention-gate] Failed to read pending: %w", err)
	}

	// Create archive directory
	if err := os.MkdirAll(sharpEdgesDir, 0755); err != nil {
		return nil, fmt.Errorf("[attention-gate] Failed to create archive dir: %w", err)
	}

	// Create timestamped archive file
	timestamp := time.Now().Format("20060102-150405")
	archivePath := filepath.Join(sharpEdgesDir, fmt.Sprintf("auto-flush-%s.jsonl", timestamp))

	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return nil, fmt.Errorf("[attention-gate] Failed to write archive: %w", err)
	}

	// Clear pending learnings
	if err := os.WriteFile(pendingPath, []byte(""), 0644); err != nil {
		return nil, fmt.Errorf("[attention-gate] Failed to clear pending: %w", err)
	}

	// Count entries
	entryCount := strings.Count(string(data), "\n")

	return &FlushContext{
		EntryCount:      entryCount,
		ArchivedFile:    archivePath,
		PendingRemaining: 0,
	}, nil
}

// GenerateFlushNotification creates notification about flushed learnings
func GenerateFlushNotification(ctx *FlushContext) string {
	notification := fmt.Sprintf(`📦 LEARNING AUTO-FLUSH

Archived %d sharp edges to:
%s

This prevents data loss on session interruption (Ctrl+C).

After session: Review auto-flush entries and decide:
- ✅ Merge into permanent sharp-edges if pattern confirmed
- ✅ Add to agent sharp-edges.yaml if agent-specific
- ❌ Delete if false alarm

See memory/sharp-edges/ for all archived learnings.`,
		ctx.EntryCount, ctx.ArchivedFile)

	return notification
}

// GenerateGateResponse creates attention-gate hook response
func GenerateGateResponse(shouldRemind bool, shouldFlush bool, reminderMsg string, flushMsg string) string {
	var contextParts []string

	if shouldRemind {
		contextParts = append(contextParts, reminderMsg)
	}

	if shouldFlush {
		contextParts = append(contextParts, flushMsg)
	}

	additionalContext := strings.Join(contextParts, "\n\n")

	response := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": "PostToolUse",
			"additionalContext": additionalContext,
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}
```

**Tests**: `pkg/observability/gate_test.go`

```go
package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRoutingReminder(t *testing.T) {
	summary := "haiku: find, search... sonnet: implement..."
	reminder := GenerateRoutingReminder(10, summary)

	if !strings.Contains(reminder, "Tool #10") {
		t.Error("Should include tool count")
	}

	if !strings.Contains(reminder, "codebase-search") {
		t.Error("Should mention codebase-search")
	}

	if !strings.Contains(reminder, "routing-schema.json") {
		t.Error("Should reference routing schema")
	}
}

func TestCheckPendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings file
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"ts":123,"file":"test.go"}
{"ts":456,"file":"main.go"}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	count, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to check: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 entries, got: %d", count)
	}
}

func TestShouldFlushLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings with 6 entries (above threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 6; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count >= 5")
	}

	if count != 6 {
		t.Errorf("Expected 6 entries, got: %d", count)
	}
}

func TestArchivePendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"ts":123}
{"ts":456}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	ctx, err := ArchivePendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to archive: %v", err)
	}

	if ctx.EntryCount != 2 {
		t.Errorf("Expected 2 entries archived, got: %d", ctx.EntryCount)
	}

	// Verify pending is cleared
	data, _ := os.ReadFile(pendingPath)
	if string(data) != "" {
		t.Error("Pending learnings should be cleared")
	}

	// Verify archive exists
	if _, err := os.Stat(ctx.ArchivedFile); os.IsNotExist(err) {
		t.Error("Archive file should exist")
	}
}

func TestGenerateFlushNotification(t *testing.T) {
	ctx := &FlushContext{
		EntryCount:   3,
		ArchivedFile: "/path/to/archive.jsonl",
	}

	notification := GenerateFlushNotification(ctx)

	if !strings.Contains(notification, "3") {
		t.Error("Should include entry count")
	}

	if !strings.Contains(notification, "archive") {
		t.Error("Should mention archive")
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateRoutingReminder()` creates compliant message every 10 tools
- [ ] `CheckPendingLearnings()` counts JSONL entries correctly
- [ ] `ShouldFlushLearnings()` returns true when count >= 5
- [ ] `ArchivePendingLearnings()` creates timestamped archive and clears pending
- [ ] Archive directory created if missing
- [ ] `GenerateFlushNotification()` explains archival and next steps
- [ ] Tests verify reminder generation, counting, flushing
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Attention-gate prevents instruction degradation and data loss. Reminders keep routing discipline. Flushing prevents sharp edge loss.

---
