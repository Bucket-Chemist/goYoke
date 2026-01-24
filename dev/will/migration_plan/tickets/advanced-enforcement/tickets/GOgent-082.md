---
id: GOgent-082
title: Integration Tests for doc-theater
description: End-to-end tests for documentation theater detection using simulation harness
status: pending
time_estimate: 2h
dependencies: ["GOgent-081"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-082: Integration Tests for doc-theater

**Time**: 2 hours
**Dependencies**: GOgent-081

**Task**:
End-to-end tests for documentation theater detection using simulation harness with deterministic fixtures.

**File**: `pkg/enforcement/doc_theater_integration_test.go`

```go
package enforcement

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/test/simulation/harness"
)

// TestDocTheaterWorkflow_TheaterDetected verifies detection of naked enforcement language
func TestDocTheaterWorkflow_TheaterDetected(t *testing.T) {
	// Use simulation harness for event generation
	gen := harness.NewGenerator()
	event := gen.ToolEvent("Edit", map[string]interface{}{
		"file_path": "/home/user/.claude/CLAUDE.md",
		"old_string": "old content",
		"new_string": `## Gate 6: Task Invocation

You MUST NOT invoke Task(opus) directly.
This is BLOCKED by the system.
Never use direct opus calls.
`,
	})

	// Parse as routing event
	toolEvent := routing.ParseToolEvent(event)

	// Event validation
	if !toolEvent.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md file")
	}

	if !toolEvent.IsWriteOperation() {
		t.Fatal("Should detect Edit operation")
	}

	// Extract content from tool input
	content := toolEvent.ExtractWriteContent()
	if content == "" {
		t.Fatal("Should extract content from tool_input.new_string")
	}

	// Detect patterns
	pd := routing.NewPatternDetector()
	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect theater patterns")
	}

	// Verify specific patterns detected
	patterns := []string{"MUST NOT", "BLOCKED", "Never"}
	for _, pattern := range patterns {
		found := false
		for _, result := range results {
			if strings.Contains(result.Pattern, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Should detect pattern: %s", pattern)
		}
	}

	if !pd.HasDocumentationTheater(content) {
		t.Error("Should identify as documentation theater")
	}

	// Generate warning
	warning := routing.GenerateWarning(results, "CLAUDE.md")
	if !strings.Contains(warning, "DOCUMENTATION THEATER") {
		t.Error("Warning should indicate theater detected")
	}

	if !strings.Contains(warning, "routing-schema.json") {
		t.Error("Warning should reference enforcement architecture")
	}
}

// CRITICAL: TestDocTheaterWorkflow_LegitimateEnforcementReference verifies
// that content WITH enforcement references is NOT flagged as theater.
// This is the baseline test for correct enforcement language.
func TestDocTheaterWorkflow_LegitimateEnforcementReference(t *testing.T) {
	gen := harness.NewGenerator()
	event := gen.ToolEvent("Edit", map[string]interface{}{
		"file_path": "/home/user/.claude/CLAUDE.md",
		"new_string": `## Gate 6: Einstein Escalation

Einstein invocation via Task tool is **blocked by validate-routing.sh (line 87)**.
See routing-schema.json → opus.task_invocation_blocked.

When Einstein triggers fire, use escalate_to_einstein protocol.
Reference: ~/.claude/skills/einstein/SKILL.md
`,
	})

	toolEvent := routing.ParseToolEvent(event)

	if !toolEvent.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md")
	}

	// Extract content
	content := toolEvent.ExtractWriteContent()
	if content == "" {
		t.Fatal("Should extract content")
	}

	// Detect patterns
	pd := routing.NewPatternDetector()
	results := pd.Detect(content)

	// CRITICAL: Content with enforcement references should NOT trigger
	if len(results) > 0 {
		t.Errorf("Should NOT flag content with enforcement references, got patterns: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not identify legitimate enforcement references as theater")
	}
}

// TestDocTheaterWorkflow_LegitimateWorkflowDescription verifies that
// descriptive workflow content is not flagged
func TestDocTheaterWorkflow_LegitimateWorkflowDescription(t *testing.T) {
	gen := harness.NewGenerator()
	event := gen.ToolEvent("Edit", map[string]interface{}{
		"file_path": "/home/user/.claude/CLAUDE.md",
		"new_string": `## Enforcement Architecture

Enforcement requires three components:
1. Declarative Rule (routing-schema.json)
2. Programmatic Check (validate-routing.sh hook)
3. Reference Documentation (CLAUDE.md points to enforcement)

See LLM-guidelines.md § Enforcement Architecture.
`,
	})

	toolEvent := routing.ParseToolEvent(event)

	if !toolEvent.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md")
	}

	content := toolEvent.ExtractWriteContent()
	if content == "" {
		t.Fatal("Should extract content")
	}

	pd := routing.NewPatternDetector()
	results := pd.Detect(content)

	if len(results) > 0 {
		t.Errorf("Should not flag legitimate workflow description, found: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not identify legitimate content as theater")
	}
}

// TestDocTheaterWorkflow_NonClaude verifies non-CLAUDE.md files are skipped
func TestDocTheaterWorkflow_NonClaude(t *testing.T) {
	gen := harness.NewGenerator()
	event := gen.ToolEvent("Edit", map[string]interface{}{
		"file_path": "/path/to/project.md",
		"new_string": "MUST NOT use this BLOCKED NEVER",
	})

	toolEvent := routing.ParseToolEvent(event)

	if toolEvent.IsClaudeMDFile() {
		t.Error("Should not match non-CLAUDE.md files")
	}

	// Even with theater patterns, hook should skip non-CLAUDE.md files
	content := toolEvent.ExtractWriteContent()
	pd := routing.NewPatternDetector()
	results := pd.Detect(content)

	// Pattern detector would find them, but hook filtering should prevent execution
	if len(results) > 0 && toolEvent.IsClaudeMDFile() {
		t.Error("Hook should skip non-CLAUDE.md files")
	}
}

// TestDocTheaterWorkflow_ReadOperation verifies read operations are skipped
func TestDocTheaterWorkflow_ReadOperation(t *testing.T) {
	gen := harness.NewGenerator()
	event := gen.ToolEvent("Read", map[string]interface{}{
		"file_path": "/home/user/.claude/CLAUDE.md",
	})

	toolEvent := routing.ParseToolEvent(event)

	if toolEvent.IsClaudeMDFile() {
		t.Log("Correctly detected CLAUDE.md file")
	}

	if toolEvent.IsWriteOperation() {
		t.Error("Read operation should not be classified as write operation")
	}

	// Hook should skip read operations entirely
}

// TestContentExtraction verifies content extraction from different tool inputs
func TestContentExtraction(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolInput map[string]interface{}
		wantEmpty bool
	}{
		{
			name:     "Edit with new_string",
			toolName: "Edit",
			toolInput: map[string]interface{}{
				"file_path":  "/path/CLAUDE.md",
				"new_string": "content here",
			},
			wantEmpty: false,
		},
		{
			name:     "Write with content",
			toolName: "Write",
			toolInput: map[string]interface{}{
				"file_path": "/path/CLAUDE.md",
				"content":   "written content",
			},
			wantEmpty: false,
		},
		{
			name:     "Read operation",
			toolName: "Read",
			toolInput: map[string]interface{}{
				"file_path": "/path/CLAUDE.md",
			},
			wantEmpty: true,
		},
	}

	gen := harness.NewGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := gen.ToolEvent(tt.toolName, tt.toolInput)
			toolEvent := routing.ParseToolEvent(event)
			content := toolEvent.ExtractWriteContent()

			if tt.wantEmpty && content != "" {
				t.Errorf("Expected empty content for %s, got: %s", tt.toolName, content)
			}

			if !tt.wantEmpty && content == "" {
				t.Errorf("Expected non-empty content for %s, got empty", tt.toolName)
			}
		})
	}
}
```

**Required Fixtures**: Create deterministic test fixtures in `test/simulation/fixtures/deterministic/doc-theater/`:

1. **01_theater_detected.json**:
```json
{
  "type": "pre-tool-use",
  "hook_event_name": "PreToolUse",
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/home/user/.claude/CLAUDE.md",
    "old_string": "old content",
    "new_string": "You MUST NOT do this. This is BLOCKED. NEVER use this pattern."
  }
}
```

2. **02_legitimate_enforcement.json**:
```json
{
  "type": "pre-tool-use",
  "hook_event_name": "PreToolUse",
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/home/user/.claude/CLAUDE.md",
    "new_string": "Blocked by validate-routing.sh line 87. See routing-schema.json."
  }
}
```

3. **03_non_claude_file.json**:
```json
{
  "type": "pre-tool-use",
  "hook_event_name": "PreToolUse",
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/home/user/project/README.md",
    "new_string": "MUST NOT BLOCKED NEVER"
  }
}
```

4. **04_read_operation.json**:
```json
{
  "type": "pre-tool-use",
  "hook_event_name": "PreToolUse",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/home/user/.claude/CLAUDE.md"
  }
}
```

**Acceptance Criteria**:
- [ ] Event generation uses simulation harness (not pseudo-events)
- [ ] Theater patterns detected in naked enforcement language
- [ ] **BASELINE TEST**: Content WITH enforcement references NOT flagged
- [ ] Legitimate workflow description not flagged
- [ ] Non-CLAUDE.md files skipped
- [ ] Read operations skipped
- [ ] Content extraction works for Edit/Write operations
- [ ] Deterministic fixtures created in test/simulation/fixtures/deterministic/doc-theater/
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Integration tests ensure doc-theater hook correctly distinguishes between documentation theater (naked imperatives) and legitimate enforcement references (pointing to actual programmatic enforcement).

---
