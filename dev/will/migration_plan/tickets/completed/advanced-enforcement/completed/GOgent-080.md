---
id: GOgent-080
title: ToolEvent Helper Functions
description: Add helper functions to existing ToolEvent struct for doc-theater enforcement
status: pending
time_estimate: 0.5h
dependencies: ["GOgent-069"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 5
---

### GOgent-080: ToolEvent Helper Functions

**Time**: 0.5 hours
**Dependencies**: GOgent-069 (ToolEvent struct already exists)

**Task**:
Add helper methods to existing `ToolEvent` struct in `pkg/routing/events.go` to support doc-theater enforcement. These methods extract data from the `tool_input` map for Write/Edit operations.

**IMPORTANT**: Do NOT create new event types. The existing `ToolEvent` struct already matches Claude Code's schema. This ticket ONLY adds helper methods.

**File**: `pkg/routing/events.go` (EXISTING FILE - append methods only)

**Implementation** (add these methods to existing ToolEvent):

```go
// ExtractFilePath gets file_path from tool_input.
// Returns empty string if file_path is not present or not a string.
func (e *ToolEvent) ExtractFilePath() string {
	if path, ok := e.ToolInput["file_path"].(string); ok {
		return path
	}
	return ""
}

// ExtractWriteContent gets content for Write tool or new_string for Edit tool.
// Returns empty string if neither field is present or not a string.
func (e *ToolEvent) ExtractWriteContent() string {
	// Write tool uses "content" field
	if content, ok := e.ToolInput["content"].(string); ok {
		return content
	}
	// Edit tool uses "new_string" field
	if newStr, ok := e.ToolInput["new_string"].(string); ok {
		return newStr
	}
	return ""
}

// IsClaudeMDFile checks if target is a CLAUDE.md file (or variant like CLAUDE.en.md).
// Returns false if file_path cannot be extracted.
func (e *ToolEvent) IsClaudeMDFile() bool {
	path := e.ExtractFilePath()
	if path == "" {
		return false
	}
	filename := filepath.Base(path)
	return filename == "CLAUDE.md" ||
		(strings.HasPrefix(filename, "CLAUDE.") && strings.HasSuffix(filename, ".md"))
}

// IsWriteOperation checks if this is a Write or Edit operation.
func (e *ToolEvent) IsWriteOperation() bool {
	return e.ToolName == "Write" || e.ToolName == "Edit"
}
```

**Add import** (if not already present):
```go
import (
	"path/filepath"
	"strings"
)
```

**Tests**: `pkg/routing/events_test.go` (EXISTING FILE - append tests)

```go
func TestToolEvent_ExtractFilePath(t *testing.T) {
	tests := []struct {
		name      string
		toolInput map[string]interface{}
		expected  string
	}{
		{
			name:      "valid file_path",
			toolInput: map[string]interface{}{"file_path": "/home/user/CLAUDE.md"},
			expected:  "/home/user/CLAUDE.md",
		},
		{
			name:      "missing file_path",
			toolInput: map[string]interface{}{"other": "value"},
			expected:  "",
		},
		{
			name:      "file_path wrong type",
			toolInput: map[string]interface{}{"file_path": 123},
			expected:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{ToolInput: tc.toolInput}
			if got := event.ExtractFilePath(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_ExtractWriteContent(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolInput map[string]interface{}
		expected  string
	}{
		{
			name:      "Write with content",
			toolName:  "Write",
			toolInput: map[string]interface{}{"content": "file contents"},
			expected:  "file contents",
		},
		{
			name:      "Edit with new_string",
			toolName:  "Edit",
			toolInput: map[string]interface{}{"new_string": "replacement text"},
			expected:  "replacement text",
		},
		{
			name:      "no content fields",
			toolName:  "Write",
			toolInput: map[string]interface{}{"other": "value"},
			expected:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{
				ToolName:  tc.toolName,
				ToolInput: tc.toolInput,
			}
			if got := event.ExtractWriteContent(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_IsClaudeMDFile(t *testing.T) {
	tests := []struct {
		name      string
		toolInput map[string]interface{}
		expected  bool
	}{
		{
			name:      "CLAUDE.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.md"},
			expected:  true,
		},
		{
			name:      "CLAUDE.en.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.en.md"},
			expected:  true,
		},
		{
			name:      "other.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/other.md"},
			expected:  false,
		},
		{
			name:      "CLAUDE.txt",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.txt"},
			expected:  false,
		},
		{
			name:      "no file_path",
			toolInput: map[string]interface{}{},
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{ToolInput: tc.toolInput}
			if got := event.IsClaudeMDFile(); got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_IsWriteOperation(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"Write", true},
		{"Edit", true},
		{"Read", false},
		{"Bash", false},
		{"Task", false},
	}

	for _, tc := range tests {
		t.Run(tc.toolName, func(t *testing.T) {
			event := &ToolEvent{ToolName: tc.toolName}
			if got := event.IsWriteOperation(); got != tc.expected {
				t.Errorf("tool %s: expected %v, got %v", tc.toolName, tc.expected, got)
			}
		})
	}
}
```

**Acceptance Criteria**:
- [x] `ExtractFilePath()` method added to ToolEvent
- [x] `ExtractWriteContent()` method added to ToolEvent
- [x] `IsClaudeMDFile()` method added to ToolEvent
- [x] `IsWriteOperation()` method added to ToolEvent
- [x] All tests pass: `go test -v ./pkg/routing`

**Why This Matters**:
These helper methods provide clean access to tool_input data for doc-theater enforcement. They work with the EXISTING ToolEvent struct that already matches Claude Code's schema. No new structs or files needed - just extending existing functionality.

---
