---
id: GOgent-070
title: PostToolUse Event Parsing
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-056"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 5
---

### GOgent-070: PostToolUse Event Parsing

**Time**: 1.5 hours
**Dependencies**: GOgent-056 (event parsing pattern)

**Task**:
Parse PostToolUse events that trigger attention-gate.

**File**: `pkg/observability/events.go`

**Imports**:
```go
package observability

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// PostToolUseEvent represents tool usage that triggers attention-gate
type PostToolUseEvent struct {
	Type          string `json:"type"`           // "post-tool-use"
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`      // e.g., "Read", "Write", "Bash"
	ToolCategory  string `json:"tool_category"`  // "file", "execution", "search"
	Duration      int    `json:"duration_ms"`    // Execution time
	Success       bool   `json:"success"`        // true if tool succeeded
	SessionID     string `json:"session_id"`     // Session identifier
}

// ParsePostToolUseEvent reads PostToolUse event from STDIN
func ParsePostToolUseEvent(r io.Reader, timeout time.Duration) (*PostToolUseEvent, error) {
	type result struct {
		event *PostToolUseEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[attention-gate] Failed to read STDIN: %w", err)}
			return
		}

		var event PostToolUseEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[attention-gate] Failed to parse JSON: %w", err)}
			return
		}

		// Default type if not specified
		if event.Type == "" {
			event.Type = "post-tool-use"
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[attention-gate] STDIN read timeout after %v", timeout)
	}
}
```

**Tests**: `pkg/observability/events_test.go`

```go
package observability

import (
	"strings"
	"testing"
	"time"
)

func TestParsePostToolUseEvent(t *testing.T) {
	jsonInput := `{
		"type": "post-tool-use",
		"hook_event_name": "PostToolUse",
		"tool_name": "Read",
		"tool_category": "file",
		"duration_ms": 100,
		"success": true,
		"session_id": "sess-123"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParsePostToolUseEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Read" {
		t.Errorf("Expected Read, got: %s", event.ToolName)
	}

	if !event.Success {
		t.Error("Expected success")
	}
}

func TestParsePostToolUseEvent_InvalidJSON(t *testing.T) {
	reader := strings.NewReader(`{invalid}`)
	_, err := ParsePostToolUseEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}
```

**Acceptance Criteria**:
- [ ] `ParsePostToolUseEvent()` reads PostToolUse events
- [ ] Implements 5s timeout
- [ ] Validates JSON structure
- [ ] Tests verify parsing and timeout
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Event parsing is required for every tool call hook invocation.

---
