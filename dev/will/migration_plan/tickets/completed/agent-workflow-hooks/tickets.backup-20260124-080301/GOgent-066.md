---
id: GOgent-066
title: Integration Tests for agent-endstate
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-065"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-066: Integration Tests for agent-endstate

**Time**: 1.5 hours
**Dependencies**: GOgent-065

**Task**:
Comprehensive tests covering event parsing → response generation → logging workflow.

**File**: `pkg/workflow/integration_test.go`

```go
package workflow

import (
	"strings"
	"testing"
	"time"
)

func TestAgentEndstateWorkflow_OrchestratorSuccess(t *testing.T) {
	// Simulate full workflow: event → response → log
	eventJSON := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"agent_model": "sonnet",
		"tier": "sonnet",
		"exit_code": 0,
		"duration_ms": 5000,
		"output_tokens": 2048
	}`

	reader := strings.NewReader(eventJSON)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	response := GenerateEndstateResponse(event)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "verify_background") {
		t.Error("Should prompt for background task verification")
	}

	// Verify JSON formatting
	jsonOutput := response.FormatJSON()
	if !strings.Contains(jsonOutput, "hookSpecificOutput") {
		t.Error("JSON should contain hookSpecificOutput")
	}
}

func TestAgentEndstateWorkflow_ImplementationFailure(t *testing.T) {
	eventJSON := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "python-pro",
		"agent_model": "sonnet",
		"tier": "sonnet",
		"exit_code": 1,
		"duration_ms": 3000,
		"output_tokens": 512
	}`

	reader := strings.NewReader(eventJSON)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	if event.IsSuccess() {
		t.Error("Should detect failure")
	}

	response := GenerateEndstateResponse(event)

	if response.Decision != "prompt" {
		t.Error("Should prompt on failure")
	}

	if !strings.Contains(response.AdditionalContext, "FAILED") {
		t.Error("Should indicate failure")
	}
}

func TestAgentEndstateWorkflow_AllAgentClasses(t *testing.T) {
	tests := []struct {
		agentID         string
		expectedDecision string
		shouldPrompt    bool
	}{
		{"orchestrator", "prompt", true},
		{"architect", "prompt", true},
		{"python-pro", "prompt", true},
		{"code-reviewer", "prompt", true},
		{"haiku-scout", "silent", false},
		{"codebase-search", "silent", false},
	}

	for _, tc := range tests {
		event := &SubagentStopEvent{
			AgentID:      tc.agentID,
			AgentModel:   "sonnet",
			Tier:         "sonnet",
			ExitCode:     0,
			Duration:     1000,
			OutputTokens: 512,
		}

		response := GenerateEndstateResponse(event)

		if response.Decision != tc.expectedDecision {
			t.Errorf("Agent %s: expected decision %s, got %s",
				tc.agentID, tc.expectedDecision, response.Decision)
		}
	}
}
```

**Acceptance Criteria**:
- [ ] Full workflow (event → response → JSON) works end-to-end
- [ ] All agent classes tested
- [ ] Success and failure paths tested
- [ ] Response JSON is valid and contains expected fields
- [ ] Integration tests verify multi-component interaction
- [ ] `go test ./pkg/workflow` passes

**Why This Matters**: Integration tests catch workflow issues that unit tests miss.

---
