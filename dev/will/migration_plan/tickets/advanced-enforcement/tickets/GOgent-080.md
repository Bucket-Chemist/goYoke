---
id: GOgent-080
title: PreToolUse Event Parsing
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-069"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-080: PreToolUse Event Parsing

**Time**: 1.5 hours
**Dependencies**: GOgent-069 (event parsing pattern)

**Task**:
Parse PreToolUse events for Write/Edit on CLAUDE.md files.

**File**: `pkg/enforcement/doc_events.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)
```

**Implementation**:
```go
// PreToolUseEvent represents tool usage before execution
type PreToolUseEvent struct {
	Type          string `json:"type"`           // "pre-tool-use"
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`      // "Write", "Edit", etc.
	FilePath      string `json:"file_path"`      // Path being modified
	SessionID     string `json:"session_id"`
}

// IsClaude MDFile checks if file is a CLAUDE.md configuration file
func (e *PreToolUseEvent) IsClaudeMDFile() bool {
	filename := filepath.Base(e.FilePath)
	// Check for CLAUDE.md or variants like CLAUDE.en.md
	if filename == "CLAUDE.md" || strings.HasPrefix(filename, "CLAUDE.") && strings.HasSuffix(filename, ".md") {
		return true
	}
	return false
}

// IsWriteOperation checks if this is a write/edit operation
func (e *PreToolUseEvent) IsWriteOperation() bool {
	return e.ToolName == "Write" || e.ToolName == "Edit"
}

// ParsePreToolUseEvent reads PreToolUse event from STDIN
func ParsePreToolUseEvent(r io.Reader, timeout time.Duration) (*PreToolUseEvent, error) {
	type result struct {
		event *PreToolUseEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[doc-theater] Failed to read STDIN: %w", err)}
			return
		}

		var event PreToolUseEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[doc-theater] Failed to parse JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[doc-theater] STDIN read timeout after %v", timeout)
	}
}
```

**Tests**: `pkg/enforcement/doc_events_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestParsePreToolUseEvent(t *testing.T) {
	jsonInput := `{
		"type": "pre-tool-use",
		"hook_event_name": "PreToolUse",
		"tool_name": "Edit",
		"file_path": "/home/user/.claude/CLAUDE.md",
		"session_id": "sess-123"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParsePreToolUseEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Edit" {
		t.Errorf("Expected Edit, got: %s", event.ToolName)
	}
}

func TestIsClaudeMDFile(t *testing.T) {
	tests := []struct {
		path      string
		isClaude  bool
	}{
		{"/path/to/CLAUDE.md", true},
		{"/path/to/CLAUDE.en.md", true},
		{"./CLAUDE.md", true},
		{"/path/to/other.md", false},
		{"/path/to/CLAUDE.txt", false},
	}

	for _, tc := range tests {
		event := &PreToolUseEvent{FilePath: tc.path}
		if got := event.IsClaudeMDFile(); got != tc.isClaude {
			t.Errorf("File %s: expected %v, got %v", tc.path, tc.isClaude, got)
		}
	}
}

func TestIsWriteOperation(t *testing.T) {
	tests := []struct {
		tool      string
		isWrite   bool
	}{
		{"Write", true},
		{"Edit", true},
		{"Read", false},
		{"Bash", false},
	}

	for _, tc := range tests {
		event := &PreToolUseEvent{ToolName: tc.tool}
		if got := event.IsWriteOperation(); got != tc.isWrite {
			t.Errorf("Tool %s: expected %v, got %v", tc.tool, tc.isWrite, got)
		}
	}
}
```

**Acceptance Criteria**:
- [ ] `ParsePreToolUseEvent()` reads PreToolUse events
- [ ] `IsClaudeMDFile()` detects CLAUDE.md variants
- [ ] `IsWriteOperation()` detects Write/Edit tools
- [ ] Implements 5s timeout
- [ ] Tests verify parsing and detection
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Event filtering is required to target only CLAUDE.md writes.

---
