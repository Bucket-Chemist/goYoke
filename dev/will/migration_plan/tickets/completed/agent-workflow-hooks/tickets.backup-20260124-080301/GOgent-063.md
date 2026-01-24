---
id: GOgent-063
title: Define SubagentStop Event Structs
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-056"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-063: Define SubagentStop Event Structs

**Time**: 1.5 hours
**Dependencies**: GOgent-056 (STDIN timeout pattern)

**Task**:
Parse SubagentStop events and detect agent completion type.

**File**: `pkg/workflow/events.go`

**Imports**:
```go
package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// SubagentStopEvent represents agent completion event
type SubagentStopEvent struct {
	Type          string `json:"type"`           // "stop"
	HookEventName string `json:"hook_event_name"` // "SubagentStop"
	AgentID       string `json:"agent_id"`       // e.g., "orchestrator", "python-pro"
	AgentModel    string `json:"agent_model"`    // "haiku", "sonnet", "opus"
	Tier          string `json:"tier"`           // "haiku", "sonnet", "opus"
	ExitCode      int    `json:"exit_code"`      // 0 = success, non-zero = failure
	Duration      int    `json:"duration_ms"`    // Execution time in milliseconds
	OutputTokens  int    `json:"output_tokens"`  // Tokens used
}

// AgentClass represents agent classification
type AgentClass string

const (
	ClassOrchestrator     AgentClass = "orchestrator"
	ClassImplementation   AgentClass = "implementation"
	ClassSpecialist       AgentClass = "specialist"
	ClassCoordination     AgentClass = "coordination"
	ClassReview           AgentClass = "review"
	ClassUnknown          AgentClass = "unknown"
)

// GetAgentClass returns the class of agent based on agent_id
func (e *SubagentStopEvent) GetAgentClass() AgentClass {
	switch e.AgentID {
	case "orchestrator", "architect", "einstein":
		return ClassOrchestrator
	case "python-pro", "python-ux", "go-pro", "r-pro", "r-shiny-pro":
		return ClassImplementation
	case "code-reviewer", "librarian", "tech-docs-writer", "scaffolder":
		return ClassSpecialist
	case "codebase-search", "haiku-scout":
		return ClassCoordination
	default:
		return ClassUnknown
	}
}

// ParseSubagentStopEvent reads SubagentStop event from STDIN
func ParseSubagentStopEvent(r io.Reader, timeout time.Duration) (*SubagentStopEvent, error) {
	type result struct {
		event *SubagentStopEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[agent-endstate] Failed to read STDIN: %w", err)}
			return
		}

		var event SubagentStopEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[agent-endstate] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		// Validate required fields
		if event.AgentID == "" {
			ch <- result{nil, fmt.Errorf("[agent-endstate] Missing required field: agent_id")}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[agent-endstate] STDIN read timeout after %v", timeout)
	}
}

// IsSuccess returns true if agent completed successfully
func (e *SubagentStopEvent) IsSuccess() bool {
	return e.ExitCode == 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Tests**: `pkg/workflow/events_test.go`

```go
package workflow

import (
	"strings"
	"testing"
	"time"
)

func TestParseSubagentStopEvent_Success(t *testing.T) {
	jsonInput := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"agent_model": "sonnet",
		"tier": "sonnet",
		"exit_code": 0,
		"duration_ms": 5000,
		"output_tokens": 1024
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.AgentID != "orchestrator" {
		t.Errorf("Expected orchestrator, got: %s", event.AgentID)
	}

	if !event.IsSuccess() {
		t.Error("Expected success")
	}

	if event.GetAgentClass() != ClassOrchestrator {
		t.Error("Expected orchestrator class")
	}
}

func TestParseSubagentStopEvent_Failure(t *testing.T) {
	jsonInput := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "python-pro",
		"agent_model": "sonnet",
		"tier": "sonnet",
		"exit_code": 1,
		"duration_ms": 3000,
		"output_tokens": 512
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.IsSuccess() {
		t.Error("Expected failure (exit_code=1)")
	}

	if event.GetAgentClass() != ClassImplementation {
		t.Error("Expected implementation class")
	}
}

func TestGetAgentClass_All(t *testing.T) {
	tests := []struct {
		agentID       string
		expectedClass AgentClass
	}{
		{"orchestrator", ClassOrchestrator},
		{"architect", ClassOrchestrator},
		{"einstein", ClassOrchestrator},
		{"python-pro", ClassImplementation},
		{"python-ux", ClassImplementation},
		{"go-pro", ClassImplementation},
		{"r-pro", ClassImplementation},
		{"r-shiny-pro", ClassImplementation},
		{"code-reviewer", ClassSpecialist},
		{"librarian", ClassSpecialist},
		{"codebase-search", ClassCoordination},
		{"haiku-scout", ClassCoordination},
		{"unknown-agent", ClassUnknown},
	}

	for _, tc := range tests {
		event := &SubagentStopEvent{AgentID: tc.agentID}
		if got := event.GetAgentClass(); got != tc.expectedClass {
			t.Errorf("AgentID %s: expected %s, got %s", tc.agentID, tc.expectedClass, got)
		}
	}
}

func TestParseSubagentStopEvent_MissingAgentID(t *testing.T) {
	jsonInput := `{
		"type": "stop",
		"hook_event_name": "SubagentStop"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for missing agent_id")
	}

	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("Error should mention agent_id, got: %v", err)
	}
}

func TestParseSubagentStopEvent_Timeout(t *testing.T) {
	reader := &blockingReader{}
	_, err := ParseSubagentStopEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

type blockingReader struct{}

func (b *blockingReader) Read(p []byte) (n int, err error) {
	time.Sleep(10 * time.Second)
	return 0, nil
}
```

**Acceptance Criteria**:
- [ ] `ParseSubagentStopEvent()` reads SubagentStop events from STDIN
- [ ] Implements 5s timeout on STDIN read
- [ ] `GetAgentClass()` correctly classifies all agent types
- [ ] `IsSuccess()` correctly identifies exit codes
- [ ] Validates required fields (agent_id)
- [ ] Tests cover success, failure, all agent classes, missing fields, timeout
- [ ] `go test ./pkg/workflow` passes

**Why This Matters**: SubagentStop is critical hook that fires when any agent completes. Correct parsing enables tier-specific follow-up actions.

---
