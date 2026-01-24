---
id: GOgent-075
title: SubagentStop Event Parsing for Orchestrator
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-063"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 5
---

### GOgent-075: SubagentStop Event Parsing for Orchestrator

**Time**: 1.5 hours
**Dependencies**: GOgent-063

**Task**:
Parse SubagentStop events specifically for orchestrator/architect agents.

**File**: `pkg/enforcement/orchestrator_events.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// OrchestratorStopEvent represents orchestrator completion
type OrchestratorStopEvent struct {
	Type          string `json:"type"`           // "stop"
	HookEventName string `json:"hook_event_name"` // "SubagentStop"
	AgentID       string `json:"agent_id"`       // "orchestrator", "architect"
	AgentModel    string `json:"agent_model"`
	ExitCode      int    `json:"exit_code"`
	TranscriptPath string `json:"transcript_path"` // Path to agent output
	Duration      int    `json:"duration_ms"`
	OutputTokens  int    `json:"output_tokens"`
}

// IsOrchestratorType checks if this is an orchestrator/architect agent
func (e *OrchestratorStopEvent) IsOrchestratorType() bool {
	return e.AgentID == "orchestrator" || e.AgentID == "architect"
}

// ParseOrchestratorStopEvent reads and validates orchestrator stop event
func ParseOrchestratorStopEvent(r io.Reader, timeout time.Duration) (*OrchestratorStopEvent, error) {
	type result struct {
		event *OrchestratorStopEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[orchestrator-guard] Failed to read STDIN: %w", err)}
			return
		}

		var event OrchestratorStopEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[orchestrator-guard] Failed to parse JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[orchestrator-guard] STDIN read timeout after %v", timeout)
	}
}
```

**Tests**: `pkg/enforcement/orchestrator_events_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestParseOrchestratorStopEvent(t *testing.T) {
	jsonInput := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"agent_model": "sonnet",
		"exit_code": 0,
		"transcript_path": "/tmp/transcript.md",
		"duration_ms": 5000,
		"output_tokens": 2048
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !event.IsOrchestratorType() {
		t.Error("Should identify as orchestrator type")
	}
}

func TestIsOrchestratorType(t *testing.T) {
	tests := []struct {
		agentID    string
		isOrch     bool
	}{
		{"orchestrator", true},
		{"architect", true},
		{"python-pro", false},
		{"code-reviewer", false},
	}

	for _, tc := range tests {
		event := &OrchestratorStopEvent{AgentID: tc.agentID}
		if got := event.IsOrchestratorType(); got != tc.isOrch {
			t.Errorf("AgentID %s: expected %v, got %v", tc.agentID, tc.isOrch, got)
		}
	}
}
```

**Acceptance Criteria**:
- [ ] `ParseOrchestratorStopEvent()` reads SubagentStop events
- [ ] `IsOrchestratorType()` correctly identifies orchestrator/architect
- [ ] Implements 5s timeout
- [ ] Tests verify parsing and type detection
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Orchestrator-guard only activates for orchestrator/architect agents.

---
