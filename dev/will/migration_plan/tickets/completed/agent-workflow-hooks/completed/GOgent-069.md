---
id: GOgent-069
title: Reminder & Flush Logic
description: "Generate routing compliance reminders and auto-flush pending learnings. Use pkg/session (NOT pkg/observability) and make thresholds configurable via environment variables."
status: pending
time_estimate: 2h
dependencies: ["GOgent-068"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 10
---

### GOgent-069: Reminder & Flush Logic

**Time**: 2 hours
**Dependencies**: GOgent-068

**Task**:
Generate routing compliance reminders and auto-flush pending learnings. Use pkg/session (NOT pkg/observability) and make thresholds configurable via environment variables.

**CRITICAL**: This ticket originally proposed creating `pkg/observability/gate.go`, but session-related functions belong in `pkg/session/`. Check if `CheckPendingLearnings` already exists in `context_loader.go` before implementing.

**File**: `pkg/session/attention_gate.go` (NEW file in existing package)

**Check for Existing Implementation**:
Before implementing, verify if these functions already exist:
- `CheckPendingLearnings()` - may exist in `context_loader.go`
- `ArchivePendingLearnings()` - may partially exist
- If found, REUSE existing implementations

**Imports**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	EntryCount       int    `json:"entries_flushed"`
	ArchivedFile     string `json:"archived_to"`
	PendingRemaining int    `json:"remaining_entries"`
}

const (
	DefaultFlushThreshold = 5
)

// GetFlushThreshold returns configurable flush threshold from env var.
// Defaults to 5 if not set or invalid.
func GetFlushThreshold() int {
	if v := os.Getenv("GOGENT_FLUSH_THRESHOLD"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return DefaultFlushThreshold
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
// NOTE: Check if this already exists in context_loader.go before implementing
func CheckPendingLearnings(projectDir string) (int, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return 0, nil // No pending learnings
	}
	if err != nil {
		return 0, fmt.Errorf("[session] Failed to read pending learnings: %w", err)
	}

	// Count lines
	lineCount := strings.Count(string(data), "\n")

	return lineCount, nil
}

// ShouldFlushLearnings checks if pending learnings exceed configurable threshold
func ShouldFlushLearnings(projectDir string) (bool, int, error) {
	count, err := CheckPendingLearnings(projectDir)
	if err != nil {
		return false, 0, err
	}

	threshold := GetFlushThreshold()
	return count >= threshold, count, nil
}

// ArchivePendingLearnings moves entries to timestamped archive
func ArchivePendingLearnings(projectDir string) (*FlushContext, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	sharpEdgesDir := filepath.Join(projectDir, ".claude", "memory", "sharp-edges")

	// Acquire lock to prevent concurrent flush
	lockPath := pendingPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("[session] Failed to create lock file: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return nil, fmt.Errorf("[session] Failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return &FlushContext{EntryCount: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[session] Failed to read pending: %w", err)
	}

	// Create archive directory
	if err := os.MkdirAll(sharpEdgesDir, 0755); err != nil {
		return nil, fmt.Errorf("[session] Failed to create archive dir: %w", err)
	}

	// Create timestamped archive file
	timestamp := time.Now().Format("20060102-150405")
	archivePath := filepath.Join(sharpEdgesDir, fmt.Sprintf("auto-flush-%s.jsonl", timestamp))

	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return nil, fmt.Errorf("[session] Failed to write archive: %w", err)
	}

	// Clear pending learnings
	if err := os.WriteFile(pendingPath, []byte(""), 0644); err != nil {
		return nil, fmt.Errorf("[session] Failed to clear pending: %w", err)
	}

	// Count entries
	entryCount := strings.Count(string(data), "\n")

	return &FlushContext{
		EntryCount:       entryCount,
		ArchivedFile:     archivePath,
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
			"hookEventName":     "PostToolUse",
			"additionalContext": additionalContext,
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}
```

**Tests**: `pkg/session/attention_gate_test.go` (add to existing package tests)

```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetFlushThreshold_Default(t *testing.T) {
	// Clear env var
	os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d, got: %d", DefaultFlushThreshold, threshold)
	}
}

func TestGetFlushThreshold_EnvVar(t *testing.T) {
	// Set custom threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "10")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != 10 {
		t.Errorf("Expected 10 from env var, got: %d", threshold)
	}
}

func TestGetFlushThreshold_InvalidEnvVar(t *testing.T) {
	// Set invalid threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "invalid")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d on invalid env, got: %d", DefaultFlushThreshold, threshold)
	}
}

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

func TestShouldFlushLearnings_ConfigurableThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Set custom threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "3")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	// Create pending learnings with 4 entries (above threshold of 3)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 4; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count >= custom threshold")
	}

	if count != 4 {
		t.Errorf("Expected 4 entries, got: %d", count)
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
- [x] Functions added to pkg/session/ (NOT pkg/observability)
- [x] CheckPendingLearnings() reused if already exists in context_loader.go
- [x] GetFlushThreshold() reads GOGENT_FLUSH_THRESHOLD env var
- [x] Defaults to 5 if env var not set or invalid
- [x] GenerateRoutingReminder() creates compliance message every 10 tools
- [x] CheckPendingLearnings() counts JSONL entries correctly
- [x] ShouldFlushLearnings() uses configurable threshold
- [x] ArchivePendingLearnings() creates timestamped archive and clears pending
- [x] Archive directory created if missing
- [x] GenerateFlushNotification() explains archival and next steps
- [x] File locking prevents concurrent flush corruption
- [x] Uses syscall.Flock() for cross-process safety
- [x] Tests verify env var configuration, reminder generation, counting, flushing
- [x] Tests use t.TempDir() for isolation
- [x] `go test ./pkg/session` passes

**Why This Matters**: Attention-gate prevents instruction degradation and data loss. Reminders keep routing discipline. Flushing prevents sharp edge loss. Configurable thresholds adapt to different project sizes.

**References**:
- Check existing: pkg/session/context_loader.go for CheckPendingLearnings
- GAP Analysis: REFACTORING-MAP.md Section 2, GOgent-069
- Env var pattern: Similar to config.GetGOgentDir() with XDG env var support

---
