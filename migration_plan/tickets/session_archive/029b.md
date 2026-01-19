---
ticket_id: GOgent-029b
title: "Capture Full Error Messages in Sharp Edges"
status: pending
dependencies: [GOgent-029]
estimated_hours: 0.5
phase: 2
priority: CRITICAL
---

# GOgent-029b: Capture Full Error Messages in Sharp Edges

## Description
Extend FormatPendingLearnings() to include full error messages from failure events, not just error types.

## Implementation Intention
Provide complete error context in sharp edges to avoid re-reading logs during review.

## Intended End State
- SharpEdge struct includes ErrorMessage field
- Error message captured during sharp edge creation
- Formatted output includes first 100 chars of error
- Test coverage ≥80%

## Dependencies
- GOgent-029: Base pending learnings formatting complete

## Acceptance Criteria
- [ ] SharpEdge struct extended with ErrorMessage field
- [ ] FormatPendingLearnings() includes error message in output
- [ ] Error message truncated to 100 chars in display
- [ ] Full error preserved in JSONL
- [ ] Tests verify error message capture
- [ ] `go test ./pkg/session` passes
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: Run `make test-ecosystem` and verify ALL PASS

## Implementation Details

### Files Modified
- `pkg/session/learnings.go` - Extend SharpEdge struct and formatting

### Enhanced SharpEdge Struct
```go
// SharpEdge represents a captured sharp edge from detector
type SharpEdge struct {
	File               string `json:"file"`
	ErrorType          string `json:"error_type"`
	ErrorMessage       string `json:"error_message,omitempty"` // NEW
	ConsecutiveFailures int   `json:"consecutive_failures"`
	LastError          string `json:"last_error,omitempty"`
	Timestamp          int64  `json:"timestamp"`
}
```

### Enhanced Formatting
```go
func FormatPendingLearnings(pendingPath string) ([]string, error) {
	// ...existing code to read JSONL...

	for scanner.Scan() {
		// ...existing parsing...

		// Enhanced format with error message
		formatted := fmt.Sprintf("- **%s**: %s (%d failures)",
			edge.File,
			edge.ErrorType,
			edge.ConsecutiveFailures,
		)

		// Add error message if available
		if edge.ErrorMessage != "" {
			truncated := edge.ErrorMessage
			if len(truncated) > 100 {
				truncated = truncated[:100] + "..."
			}
			formatted += fmt.Sprintf("\n  Error: `%s`", truncated)
		}

		learnings = append(learnings, formatted)
	}

	return learnings, nil
}
```

### Expected Output Format
```markdown
- **src/main.go**: TypeError (3 failures)
  Error: `invalid type assertion: field is bool, not interface{}`

- **pkg/utils.go**: nil_pointer (2 failures)
  Error: `panic: runtime error: invalid memory address or nil pointer dereference`
```

### Integration with Sharp Edge Capture
**Dependency Note**: This ticket extends the formatting function from GOgent-029. The actual error message capture during sharp edge creation happens in GOgent-037b (sharp-edge-detector hook). That ticket will update CaptureSharpEdge() to extract error messages from PostToolUse event's tool_response fields and write them to the JSONL file.

This ticket (029b) only handles the *display* of error messages that are already in the JSONL file.

### Tests
```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatPendingLearnings_WithErrorMessage(t *testing.T) {
	// Create temp dir and test file with error messages
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "with-errors.jsonl")

	content := `{"file":"src/main.go","error_type":"type_mismatch","error_message":"invalid type assertion: field is bool, not interface{}","consecutive_failures":3,"timestamp":1674000000}
{"file":"pkg/utils.go","error_type":"nil_pointer","error_message":"panic: runtime error: invalid memory address or nil pointer dereference","consecutive_failures":2,"timestamp":1674000001}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 2 {
		t.Errorf("Expected 2 learnings, got: %d", len(learnings))
	}

	// Verify error message is included
	if !strings.Contains(learnings[0], "Error: `invalid type assertion") {
		t.Errorf("Expected error message in output, got: %s", learnings[0])
	}

	// Verify second error message
	if !strings.Contains(learnings[1], "Error: `panic: runtime error") {
		t.Errorf("Expected error message in output, got: %s", learnings[1])
	}
}

func TestFormatPendingLearnings_ErrorMessageTruncation(t *testing.T) {
	// Create test file with long error message (>100 chars)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "long-error.jsonl")

	longError := "This is a very long error message that exceeds one hundred characters and should be truncated to prevent display overflow in the formatted output"
	content := `{"file":"src/main.go","error_type":"long_error","error_message":"` + longError + `","consecutive_failures":1,"timestamp":1674000000}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 1 {
		t.Errorf("Expected 1 learning, got: %d", len(learnings))
	}

	// Verify truncation
	if !strings.Contains(learnings[0], "...") {
		t.Errorf("Expected truncation marker '...', got: %s", learnings[0])
	}

	// Verify error message is not longer than 100 chars + "Error: `" + "`" + "..." = ~110 chars
	// Extract the error portion
	parts := strings.Split(learnings[0], "Error: `")
	if len(parts) < 2 {
		t.Fatalf("Could not find error message in output: %s", learnings[0])
	}

	errorPart := strings.TrimSuffix(parts[1], "`")
	if len(errorPart) > 103 { // 100 chars + "..."
		t.Errorf("Error message not truncated properly, length: %d", len(errorPart))
	}
}

func TestFormatPendingLearnings_WithoutErrorMessage(t *testing.T) {
	// Test backward compatibility - old format without error_message field
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "no-error-message.jsonl")

	content := `{"file":"src/main.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1674000000}
{"file":"pkg/utils.go","error_type":"nil_pointer","consecutive_failures":2,"timestamp":1674000001}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 2 {
		t.Errorf("Expected 2 learnings, got: %d", len(learnings))
	}

	// Verify no error message is shown (old format)
	if strings.Contains(learnings[0], "Error:") {
		t.Errorf("Expected no error message for old format, got: %s", learnings[0])
	}

	// Verify basic format still works
	expected := "- **src/main.go**: type_mismatch (3 failures)"
	if learnings[0] != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, learnings[0])
	}
}

func TestFormatPendingLearnings_SpecialCharacters(t *testing.T) {
	// Test error messages with special characters (quotes, newlines, etc.)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "special-chars.jsonl")

	// Note: In JSON, newlines and quotes are escaped
	content := `{"file":"src/main.go","error_type":"parse_error","error_message":"Expected '}', found '\"'","consecutive_failures":1,"timestamp":1674000000}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 1 {
		t.Errorf("Expected 1 learning, got: %d", len(learnings))
	}

	// Verify special characters are preserved
	if !strings.Contains(learnings[0], "Expected '}'") {
		t.Errorf("Expected special characters preserved, got: %s", learnings[0])
	}
}

func TestFormatPendingLearnings_VerifyMarkdownFormat(t *testing.T) {
	// Verify output is valid markdown
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "markdown-format.jsonl")

	content := `{"file":"src/main.go","error_type":"type_error","error_message":"type mismatch","consecutive_failures":2,"timestamp":1674000000}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 1 {
		t.Errorf("Expected 1 learning, got: %d", len(learnings))
	}

	// Verify markdown list format
	if !strings.HasPrefix(learnings[0], "- ") {
		t.Errorf("Expected markdown list item (starts with '- '), got: %s", learnings[0])
	}

	// Verify bold file name
	if !strings.Contains(learnings[0], "**src/main.go**") {
		t.Errorf("Expected bold file name, got: %s", learnings[0])
	}

	// Verify error message is in backticks
	if !strings.Contains(learnings[0], "Error: `") || !strings.Contains(learnings[0], "`") {
		t.Errorf("Expected error message in backticks, got: %s", learnings[0])
	}
}
```

### Test Deliverables (MANDATORY)
- [ ] Test file created: `pkg/session/learnings_test.go` (extends from GOgent-029)
- [ ] Number of test functions: 5 additional tests (total 10 with GOgent-029)
- [ ] Coverage achieved: ≥80%
- [ ] Tests passing: ✅ (output: `go test ./pkg/session`)
- [ ] Race detector clean: ✅ (output: `go test -race ./pkg/session`)
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: Run `make test-ecosystem` and verify ALL PASS
- [ ] Ecosystem test output saved to: `test/audit/GOgent-029b/`
- [ ] Test audit updated: `/test/INDEX.md` row added

**CRITICAL**: The `make test-ecosystem` command MUST pass before ticket can be marked complete. This is NON-NEGOTIABLE.

## Time Estimate
0.5 hours

## Why This Matters
Error messages are essential for understanding sharp edges. Without them, reviewers must re-read session logs to understand what failed. This wastes time and breaks the review flow.
