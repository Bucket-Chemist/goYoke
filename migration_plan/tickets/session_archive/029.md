---
ticket_id: GOgent-029
title: "Format Pending Learnings"
status: pending
dependencies: [GOgent-028]
estimated_hours: 1.0
phase: 1
priority: CRITICAL
---

# GOgent-029: Format Pending Learnings

## Description
Parse pending-learnings.jsonl and format sharp edges into markdown bullets.

## Implementation Intention
Format captured sharp edges for human-readable display in handoff document.

## Intended End State
- FormatPendingLearnings() reads JSONL and returns formatted strings
- Markdown bullets with file, error type, failure count
- Skips invalid JSON lines gracefully
- Test coverage ≥80%

## Dependencies
- GOgent-028: Base handoff generation complete

## Acceptance Criteria
- [ ] `FormatPendingLearnings()` reads JSONL file
- [ ] Parses SharpEdge structs from each line
- [ ] Formats as markdown bullets with file, error type, failure count
- [ ] Skips invalid JSON lines gracefully
- [ ] Returns nil for missing file (not an error)
- [ ] Tests verify formatting, empty file, invalid JSON
- [ ] `go test ./pkg/session` passes
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: Run `make test-ecosystem` and verify ALL PASS

## Implementation Details

### File Created
- `pkg/session/learnings.go`

### Imports
```go
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)
```

### Structs
```go
// SharpEdge represents a captured sharp edge from detector
type SharpEdge struct {
	File               string `json:"file"`
	ErrorType          string `json:"error_type"`
	ConsecutiveFailures int   `json:"consecutive_failures"`
	LastError          string `json:"last_error,omitempty"`
	Timestamp          int64  `json:"timestamp"`
}
```

### Implementation
```go
func FormatPendingLearnings(pendingPath string) ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		return nil, nil // Not an error - file may not exist
	}

	file, err := os.Open(pendingPath)
	if err != nil {
		return nil, fmt.Errorf("[learnings] Failed to open pending learnings at %s: %w", pendingPath, err)
	}
	defer file.Close()

	var learnings []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		var edge SharpEdge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			continue // Skip invalid JSON lines gracefully
		}

		formatted := fmt.Sprintf("- **%s**: %s (%d failures)",
			edge.File,
			edge.ErrorType,
			edge.ConsecutiveFailures,
		)

		learnings = append(learnings, formatted)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[learnings] Error reading pending learnings: %w", err)
	}

	return learnings, nil
}
```

### Output Format
```markdown
- **src/main.go**: type_mismatch (3 failures)
- **pkg/utils.go**: nil_pointer (2 failures)
```

### Tests
```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatPendingLearnings_ValidJSONL(t *testing.T) {
	// Create temp dir and test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "pending-learnings.jsonl")

	content := `{"file":"src/main.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1674000000}
{"file":"pkg/utils.go","error_type":"nil_pointer","consecutive_failures":2,"timestamp":1674000001}
{"file":"internal/handler.go","error_type":"parse_error","consecutive_failures":1,"timestamp":1674000002}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 3 {
		t.Errorf("Expected 3 learnings, got: %d", len(learnings))
	}

	// Verify first learning
	expected := "- **src/main.go**: type_mismatch (3 failures)"
	if learnings[0] != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, learnings[0])
	}

	// Verify second learning
	expected = "- **pkg/utils.go**: nil_pointer (2 failures)"
	if learnings[1] != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, learnings[1])
	}
}

func TestFormatPendingLearnings_MissingFile(t *testing.T) {
	// Use non-existent file path
	nonExistentPath := "/tmp/does-not-exist-12345.jsonl"

	learnings, err := FormatPendingLearnings(nonExistentPath)

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if learnings != nil {
		t.Errorf("Expected nil slice for missing file, got: %v", learnings)
	}
}

func TestFormatPendingLearnings_EmptyFile(t *testing.T) {
	// Create temp dir and empty test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.jsonl")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(learnings) != 0 {
		t.Errorf("Expected 0 learnings from empty file, got: %d", len(learnings))
	}
}

func TestFormatPendingLearnings_InvalidJSON(t *testing.T) {
	// Create temp dir and test file with mixed valid/invalid JSON
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mixed.jsonl")

	content := `{"file":"valid.go","error_type":"test_error","consecutive_failures":1,"timestamp":1674000000}
this is not json
{"file":"also_valid.go","error_type":"another_error","consecutive_failures":2,"timestamp":1674000001}
{incomplete json
{"file":"third_valid.go","error_type":"third_error","consecutive_failures":3,"timestamp":1674000002}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have 3 valid learnings (invalid lines skipped)
	if len(learnings) != 3 {
		t.Errorf("Expected 3 valid learnings, got: %d", len(learnings))
	}

	// Verify first valid line was parsed
	if !strings.Contains(learnings[0], "valid.go") {
		t.Errorf("Expected first learning to contain 'valid.go', got: %s", learnings[0])
	}

	// Verify third valid line was parsed
	if !strings.Contains(learnings[2], "third_valid.go") {
		t.Errorf("Expected third learning to contain 'third_valid.go', got: %s", learnings[2])
	}
}

func TestFormatPendingLearnings_BlankLines(t *testing.T) {
	// Create temp dir and test file with blank lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "blank-lines.jsonl")

	content := `{"file":"first.go","error_type":"error1","consecutive_failures":1,"timestamp":1674000000}

{"file":"second.go","error_type":"error2","consecutive_failures":2,"timestamp":1674000001}

`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	learnings, err := FormatPendingLearnings(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have 2 learnings (blank lines skipped)
	if len(learnings) != 2 {
		t.Errorf("Expected 2 learnings, got: %d", len(learnings))
	}
}
```

### Test Deliverables (MANDATORY)
- [ ] Test file created: `pkg/session/learnings_test.go`
- [ ] Number of test functions: 5
- [ ] Coverage achieved: ≥80%
- [ ] Tests passing: ✅ (output: `go test ./pkg/session`)
- [ ] Race detector clean: ✅ (output: `go test -race ./pkg/session`)
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: Run `make test-ecosystem` and verify ALL PASS
- [ ] Ecosystem test output saved to: `test/audit/GOgent-029/`
- [ ] Test audit updated: `/test/INDEX.md` row added

**CRITICAL**: The `make test-ecosystem` command MUST pass before ticket can be marked complete. This is NON-NEGOTIABLE.

## Time Estimate
1.0 hours

## Why This Matters
Pending learnings are critical for improving system. Formatting must be clear for human review.
