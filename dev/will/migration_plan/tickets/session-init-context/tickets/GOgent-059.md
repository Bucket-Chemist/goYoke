---
id: GOgent-059
title: Handoff Document Loader
description: **Task**:
status: pending
time_estimate: 1h
dependencies: []
priority: MEDIUM
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 13
---

## GOgent-059: Handoff Document Loader

**Time**: 1 hour
**Dependencies**: None
**Priority**: MEDIUM

**Task**:
Add handoff loading function for SessionStart resume sessions.

**File**: `pkg/session/context_loader.go` (new file)

**Implementation**:
```go
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadHandoffSummary loads previous session handoff for resume sessions.
// Returns first 30 lines of last-handoff.md with truncation indicator.
// Returns empty string with no error if handoff doesn't exist.
func LoadHandoffSummary(projectDir string) (string, error) {
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")

	// Check if handoff exists
	info, err := os.Stat(handoffPath)
	if os.IsNotExist(err) {
		return "", nil // No handoff is normal
	}
	if err != nil {
		return "", fmt.Errorf("[context-loader] Failed to stat handoff at %s: %w", handoffPath, err)
	}

	// Don't read excessively large files
	const maxSize = 50 * 1024 // 50KB limit
	if info.Size() > maxSize {
		return fmt.Sprintf("Handoff file too large (%d bytes). See: %s", info.Size(), handoffPath), nil
	}

	// Read handoff file
	data, err := os.ReadFile(handoffPath)
	if err != nil {
		return "", fmt.Errorf("[context-loader] Failed to read handoff from %s: %w", handoffPath, err)
	}

	content := string(data)

	// Return first 30 lines with truncation indicator
	lines := strings.Split(content, "\n")
	totalLines := len(lines) // Capture BEFORE truncation
	if totalLines > 30 {
		truncatedCount := totalLines - 30
		lines = lines[:30]
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("(... %d lines truncated. Full handoff: %s)", truncatedCount, handoffPath))
	}

	return strings.Join(lines, "\n"), nil
}

// CheckPendingLearnings checks for accumulated sharp edges requiring review.
// Returns warning message if pending learnings exist, empty string otherwise.
func CheckPendingLearnings(projectDir string) (string, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	// Check if file exists and has content
	info, err := os.Stat(pendingPath)
	if os.IsNotExist(err) {
		return "", nil // No pending learnings
	}
	if err != nil {
		return "", fmt.Errorf("[context-loader] Failed to stat pending learnings at %s: %w", pendingPath, err)
	}

	if info.Size() == 0 {
		return "", nil // Empty file
	}

	// Count lines (each line is one sharp edge)
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		return "", fmt.Errorf("[context-loader] Failed to read pending learnings: %w", err)
	}

	lineCount := strings.Count(string(data), "\n")
	if lineCount == 0 && len(data) > 0 {
		lineCount = 1 // Single line without newline
	}

	return fmt.Sprintf("⚠️ PENDING LEARNINGS: %d sharp edge(s) from previous sessions need review.\n   Path: %s", lineCount, pendingPath), nil
}

// FormatGitInfo formats GitInfo struct for context injection.
// Uses existing collectGitInfo() from handoff.go.
func FormatGitInfo(projectDir string) string {
	info := collectGitInfo(projectDir)

	if info.Branch == "" {
		return "" // Not a git repo
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Branch: %s", info.Branch))

	if info.IsDirty {
		uncommittedCount := len(info.Uncommitted)
		parts = append(parts, fmt.Sprintf("Uncommitted: %d file(s)", uncommittedCount))
	} else {
		parts = append(parts, "Clean working tree")
	}

	return "GIT: " + strings.Join(parts, " | ")
}
```

**Tests**: `pkg/session/context_loader_test.go` (new file)

```go
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHandoffSummary_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create mock handoff
	handoffContent := `# Session Handoff

## Summary
Last session implemented feature X.

## Sharp Edges
- Edge 1: Type mismatch in parser

## Next Steps
- Complete testing
`
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	os.WriteFile(handoffPath, []byte(handoffContent), 0644)

	// Load summary
	summary, err := LoadHandoffSummary(tmpDir)

	if err != nil {
		t.Fatalf("LoadHandoffSummary failed: %v", err)
	}

	if !strings.Contains(summary, "Session Handoff") {
		t.Error("Summary should contain handoff content")
	}

	if !strings.Contains(summary, "feature X") {
		t.Error("Summary should contain session summary")
	}
}

func TestLoadHandoffSummary_Missing(t *testing.T) {
	tmpDir := t.TempDir()

	summary, err := LoadHandoffSummary(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing handoff, got: %v", err)
	}

	if summary != "" {
		t.Errorf("Expected empty string for missing handoff, got: %s", summary)
	}
}

func TestLoadHandoffSummary_Truncation(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create large handoff (40 lines)
	var lines []string
	for i := 1; i <= 40; i++ {
		lines = append(lines, fmt.Sprintf("Line %d: content content content", i))
	}
	handoffContent := strings.Join(lines, "\n")

	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	os.WriteFile(handoffPath, []byte(handoffContent), 0644)

	// Load summary
	summary, err := LoadHandoffSummary(tmpDir)

	if err != nil {
		t.Fatalf("LoadHandoffSummary failed: %v", err)
	}

	// Should contain first lines
	if !strings.Contains(summary, "Line 1:") {
		t.Error("Summary should contain first line")
	}

	// Should indicate truncation with correct count (40 - 30 = 10 lines)
	if !strings.Contains(summary, "truncated") {
		t.Error("Summary should indicate truncation")
	}
	if !strings.Contains(summary, "10 lines truncated") {
		t.Errorf("Should show correct truncation count (10), got: %s", summary)
	}

	// Should NOT contain line 35
	if strings.Contains(summary, "Line 35:") {
		t.Error("Should truncate after 30 lines")
	}
}

func TestCheckPendingLearnings_HasLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings (3 entries)
	pendingContent := `{"ts":123,"file":"test.go","error_type":"type_mismatch"}
{"ts":456,"file":"main.go","error_type":"nil_pointer"}
{"ts":789,"file":"utils.go","error_type":"bounds_check"}
`
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(pendingPath, []byte(pendingContent), 0644)

	// Check pending learnings
	message, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("CheckPendingLearnings failed: %v", err)
	}

	if !strings.Contains(message, "PENDING LEARNINGS") {
		t.Error("Message should indicate pending learnings")
	}

	if !strings.Contains(message, "3 sharp edge") {
		t.Errorf("Should detect 3 sharp edges, got: %s", message)
	}
}

func TestCheckPendingLearnings_None(t *testing.T) {
	tmpDir := t.TempDir()

	message, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if message != "" {
		t.Errorf("Expected empty message for no pending learnings, got: %s", message)
	}
}

func TestCheckPendingLearnings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create empty file
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(pendingPath, []byte(""), 0644)

	message, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on empty file, got: %v", err)
	}

	if message != "" {
		t.Errorf("Expected empty message for empty file, got: %s", message)
	}
}

func TestFormatGitInfo_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	info := FormatGitInfo(tmpDir)

	if info != "" {
		t.Errorf("Expected empty string for non-git repo, got: %s", info)
	}
}
```

**Acceptance Criteria**:
- [ ] `LoadHandoffSummary()` reads from `.claude/memory/last-handoff.md`
- [ ] Returns first 30 lines with truncation indicator for large files
- [ ] Handles missing handoff gracefully (returns empty string, not error)
- [ ] `CheckPendingLearnings()` counts lines in `pending-learnings.jsonl`
- [ ] `FormatGitInfo()` reuses existing `collectGitInfo()` function
- [ ] Tests verify content loading, missing files, truncation
- [ ] `go test ./pkg/session/...` passes

**Test Deliverables**:
- [ ] Test file created: `pkg/session/context_loader_test.go`
- [ ] Test file size: ~180 lines
- [ ] Number of test functions: 7
- [ ] Coverage achieved: >85%
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Handoff loading enables multi-session continuity. Resume sessions need context from previous work to maintain coherent agent behavior.

---
