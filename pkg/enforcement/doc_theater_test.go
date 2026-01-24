package enforcement

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// TestAnalyzeToolEventForDocTheater_WriteWithTheater verifies theater detection on Write operations
func TestAnalyzeToolEventForDocTheater_WriteWithTheater(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"content": `## Gate 6: Task Invocation

You MUST NOT invoke Task(opus) directly.
This is BLOCKED (by the system).
Never use direct opus calls.
`,
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results == nil {
		t.Fatal("Expected detection results, got nil")
	}

	if len(results) == 0 {
		t.Fatal("Expected theater patterns to be detected")
	}

	// Verify at least MUST NOT is detected
	foundMustNot := false
	foundBlocked := false
	for _, result := range results {
		if strings.Contains(result.FirstMatch, "MUST NOT") {
			foundMustNot = true
		}
		if strings.Contains(result.FirstMatch, "BLOCKED") {
			foundBlocked = true
		}
	}

	if !foundMustNot {
		t.Error("Expected to detect MUST NOT pattern")
	}

	if !foundBlocked {
		t.Error("Expected to detect BLOCKED pattern (with parentheses)")
	}
}

// TestAnalyzeToolEventForDocTheater_EditWithTheater verifies theater detection on Edit operations
func TestAnalyzeToolEventForDocTheater_EditWithTheater(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"old_string": "old content",
			"new_string": `## Routing Rules

You CANNOT use codebase-search without asking first.
This is FORBIDDEN by policy.
NEVER use files directly.
`,
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results == nil {
		t.Fatal("Expected detection results, got nil")
	}

	if len(results) == 0 {
		t.Fatal("Expected theater patterns to be detected")
	}

	// Verify we get FORBIDDEN (warning) and NEVER use (critical)
	hasWarning := false
	hasCritical := false
	for _, result := range results {
		if result.Severity == "warning" {
			hasWarning = true
		}
		if result.Severity == "critical" {
			hasCritical = true
		}
	}

	if !hasWarning {
		t.Error("Expected at least one warning severity pattern (FORBIDDEN or YOU CANNOT)")
	}

	if !hasCritical {
		t.Error("Expected at least one critical severity pattern (NEVER use)")
	}
}

// TestAnalyzeToolEventForDocTheater_LegitimateEnforcement verifies clean content is not flagged
func TestAnalyzeToolEventForDocTheater_LegitimateEnforcement(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"new_string": `## Gate 6: Einstein Escalation

Einstein invocation via Task tool is prevented by the validate-routing.sh hook.
See routing-schema.json → opus.task_invocation_blocked for the rule definition.

When Einstein triggers fire, follow the escalate_to_einstein protocol.
Reference: ~/.claude/skills/einstein/SKILL.md
`,
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	// CRITICAL: Legitimate enforcement references should NOT trigger detection
	// This content describes enforcement without using imperative theater patterns
	if results != nil && len(results) > 0 {
		t.Errorf("Expected clean content to pass, but got %d detections: %v", len(results), results)
	}
}

// TestAnalyzeToolEventForDocTheater_WorkflowDescription verifies descriptive content is not flagged
func TestAnalyzeToolEventForDocTheater_WorkflowDescription(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/project/.claude/CLAUDE.md",
			"content": `## Enforcement Architecture

Enforcement requires three components:
1. Declarative Rule (routing-schema.json)
2. Programmatic Check (validate-routing.sh hook)
3. Reference Documentation (CLAUDE.md points to enforcement)

See LLM-guidelines.md § Enforcement Architecture.
`,
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results != nil && len(results) > 0 {
		t.Errorf("Expected workflow description to pass, but got %d detections: %v", len(results), results)
	}
}

// TestAnalyzeToolEventForDocTheater_NonClaudeFile verifies non-CLAUDE.md files are skipped
func TestAnalyzeToolEventForDocTheater_NonClaudeFile(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/project/README.md",
			"new_string": "You MUST NOT use this pattern. This is BLOCKED. NEVER do this.",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	// Non-CLAUDE.md files should be skipped entirely
	if results != nil {
		t.Errorf("Expected nil for non-CLAUDE.md file, got %d results", len(results))
	}
}

// TestAnalyzeToolEventForDocTheater_ReadOperation verifies read operations are skipped
func TestAnalyzeToolEventForDocTheater_ReadOperation(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Read",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results != nil {
		t.Error("Expected nil for Read operation, got results")
	}
}

// TestAnalyzeToolEventForDocTheater_GlobOperation verifies non-write tools are skipped
func TestAnalyzeToolEventForDocTheater_GlobOperation(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Glob",
		ToolInput: map[string]interface{}{
			"pattern": "*.md",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results != nil {
		t.Error("Expected nil for Glob operation, got results")
	}
}

// TestAnalyzeToolEventForDocTheater_ClaudeVariants verifies CLAUDE.en.md and other variants
func TestAnalyzeToolEventForDocTheater_ClaudeVariants(t *testing.T) {
	variants := []string{
		"/home/user/.claude/CLAUDE.md",
		"/home/user/.claude/CLAUDE.en.md",
		"/home/user/project/CLAUDE.md",
		"/home/user/project/CLAUDE.fr.md",
	}

	theaterContent := "You MUST NOT do this. NEVER use this pattern."

	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			event := &routing.ToolEvent{
				ToolName: "Write",
				ToolInput: map[string]interface{}{
					"file_path": variant,
					"content":   theaterContent,
				},
			}

			results := AnalyzeToolEventForDocTheater(event)

			if results == nil || len(results) == 0 {
				t.Errorf("Expected theater detection for CLAUDE.md variant %s", variant)
			}
		})
	}
}

// TestAnalyzeToolEventForDocTheater_EmptyContent verifies handling of empty content
func TestAnalyzeToolEventForDocTheater_EmptyContent(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"content":   "",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	// Empty content should return nil (nothing to analyze)
	if results != nil {
		t.Error("Expected nil for empty content, got results")
	}
}

// TestAnalyzeToolEventForDocTheater_MissingFilePath verifies graceful handling of missing file_path
func TestAnalyzeToolEventForDocTheater_MissingFilePath(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"content": "Some content with MUST NOT",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	// Missing file_path should be skipped (IsClaudeMDFile returns false)
	if results != nil {
		t.Error("Expected nil for missing file_path, got results")
	}
}

// TestAnalyzeToolEventForDocTheater_NilEvent verifies nil event handling
func TestAnalyzeToolEventForDocTheater_NilEvent(t *testing.T) {
	// This test verifies defensive behavior - though production code should never pass nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil event, but did not panic")
		}
	}()

	AnalyzeToolEventForDocTheater(nil)
}

// TestAnalyzeToolEventForDocTheater_MultiplePatterns verifies multiple patterns are detected
func TestAnalyzeToolEventForDocTheater_MultiplePatterns(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"new_string": `## Rules

You MUST NOT use pattern A.
This is BLOCKED (per policy).
NEVER use pattern B.
This is FORBIDDEN.
YOU CANNOT use pattern C.
`,
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results == nil || len(results) < 4 {
		t.Fatalf("Expected at least 4 patterns (MUST NOT, BLOCKED, NEVER, FORBIDDEN), got %d", len(results))
	}

	// Verify pattern diversity
	patterns := make(map[string]bool)
	for _, result := range results {
		patterns[result.Pattern] = true
	}

	if len(patterns) < 4 {
		t.Errorf("Expected at least 4 unique patterns, got %d: %v", len(patterns), patterns)
	}
}

// TestAnalyzeToolEventForDocTheater_CaseSensitivity verifies case-insensitive detection
func TestAnalyzeToolEventForDocTheater_CaseSensitivity(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"lowercase", "you must not do this"},
		{"uppercase", "YOU MUST NOT DO THIS"},
		{"mixed", "You MuSt NoT do this"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := &routing.ToolEvent{
				ToolName: "Write",
				ToolInput: map[string]interface{}{
					"file_path": "/home/user/.claude/CLAUDE.md",
					"content":   tc.content,
				},
			}

			results := AnalyzeToolEventForDocTheater(event)

			if results == nil || len(results) == 0 {
				t.Errorf("Expected case-insensitive detection for %q", tc.content)
			}
		})
	}
}

// TestAnalyzeToolEventForDocTheater_PartialContent verifies detection in partial edits
func TestAnalyzeToolEventForDocTheater_PartialContent(t *testing.T) {
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"old_string": "Some existing content",
			"new_string": "Updated content with MUST NOT pattern",
		},
	}

	results := AnalyzeToolEventForDocTheater(event)

	if results == nil || len(results) == 0 {
		t.Error("Expected detection in partial Edit content")
	}
}
