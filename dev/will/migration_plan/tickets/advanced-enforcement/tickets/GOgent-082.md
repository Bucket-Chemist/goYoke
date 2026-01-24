---
id: GOgent-082
title: Integration Tests for doc-theater
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-081"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-082: Integration Tests for doc-theater

**Time**: 1.5 hours
**Dependencies**: GOgent-081

**Task**:
End-to-end tests for documentation theater detection.

**File**: `pkg/enforcement/doc_theater_integration_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestDocTheaterWorkflow_TheaterDetected(t *testing.T) {
	// Parse event
	eventJSON := `{
		"type": "pre-tool-use",
		"hook_event_name": "PreToolUse",
		"tool_name": "Edit",
		"file_path": "/home/user/.claude/CLAUDE.md"
	}`

	event := &PreToolUseEvent{
		ToolName:  "Edit",
		FilePath:  "/home/user/.claude/CLAUDE.md",
	}

	// Event validation
	if !event.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md file")
	}

	if !event.IsWriteOperation() {
		t.Fatal("Should detect Edit operation")
	}

	// Content with theater patterns
	content := `## Gate 6: Task Invocation

You MUST NOT invoke Task(opus) directly.
This is BLOCKED by the system.
Never use direct opus calls.
`

	// Detect patterns
	pd := NewPatternDetector()
	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect theater patterns")
	}

	if !pd.HasDocumentationTheater(content) {
		t.Error("Should identify as documentation theater")
	}

	// Generate warning
	warning := GenerateWarning(results, "CLAUDE.md")
	if !strings.Contains(warning, "DOCUMENTATION THEATER") {
		t.Error("Warning should indicate theater detected")
	}
}

func TestDocTheaterWorkflow_LegitimateContent(t *testing.T) {
	event := &PreToolUseEvent{
		ToolName: "Edit",
		FilePath: "/home/user/.claude/CLAUDE.md",
	}

	if !event.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md")
	}

	// Legitimate content (no theater)
	content := `## Enforcement Architecture

Enforcement requires three components:
1. Declarative Rule (routing-schema.json)
2. Programmatic Check (validate-routing.sh hook)
3. Reference Documentation (CLAUDE.md points to enforcement)

See LLM-guidelines.md § Enforcement Architecture.
`

	pd := NewPatternDetector()
	results := pd.Detect(content)

	if len(results) > 0 {
		t.Errorf("Should not flag legitimate content, found: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not identify legitimate content as theater")
	}
}

func TestDocTheaterWorkflow_NonClaude(t *testing.T) {
	// Writing to non-CLAUDE.md file
	event := &PreToolUseEvent{
		ToolName: "Edit",
		FilePath: "/path/to/project.md",
	}

	if event.IsClaudeMDFile() {
		t.Error("Should not match non-CLAUDE.md files")
	}

	// Even with theater patterns, should not trigger
	content := "MUST NOT use this BLOCKED NEVER"

	pd := NewPatternDetector()
	results := pd.Detect(content)

	// Pattern detector would find them, but hook should skip
	// since not CLAUDE.md file
	if len(results) > 0 && event.IsClaudeMDFile() {
		t.Error("Hook should skip non-CLAUDE.md files")
	}
}
```

**Acceptance Criteria**:
- [ ] Event parsing for CLAUDE.md detection works
- [ ] Theater patterns detected correctly
- [ ] Legitimate content not flagged
- [ ] Non-CLAUDE.md files skipped
- [ ] Warning generated with remediation
- [ ] JSON response valid
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Integration tests ensure doc-theater hook catches real documentation theater.

---
