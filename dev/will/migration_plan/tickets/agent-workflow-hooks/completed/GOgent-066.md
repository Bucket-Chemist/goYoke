---
id: GOgent-066
title: Integration Tests for agent-endstate
description: "Comprehensive tests covering event parsing → transcript parsing → response generation → logging workflow. Uses ACTUAL SubagentStop schema."
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-065"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 11
---

### GOgent-066: Integration Tests for agent-endstate

**Time**: 1.5 hours
**Dependencies**: GOgent-065

**Task**:
Comprehensive tests covering event parsing → transcript parsing → response generation → logging workflow. Uses ACTUAL SubagentStop schema.

**File**: `pkg/workflow/integration_test.go`

```go
package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createMockTranscript creates a mock transcript file for testing
func createMockTranscript(t *testing.T, path, agentID, model string) {
	t.Helper()

	// Create mock transcript with agent metadata
	transcript := []map[string]interface{}{
		{
			"type": "agent_start",
			"agent_id": agentID,
			"model": model,
			"timestamp": time.Now().Unix(),
		},
		{
			"type": "completion",
			"exit_code": 0,
			"output_tokens": 2048,
			"duration_ms": 5000,
		},
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create mock transcript: %v", err)
	}
	defer f.Close()

	for _, entry := range transcript {
		data, _ := json.Marshal(entry)
		f.WriteString(string(data) + "\n")
	}
}

func TestAgentEndstateWorkflow_OrchestratorSuccess(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Create mock transcript with agent metadata
	createMockTranscript(t, transcriptPath, "orchestrator", "sonnet")

	// Simulate full workflow: event → transcript parsing → response → log
	// Uses ACTUAL SubagentStop schema
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-orchestrator",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse transcript for agent metadata
	metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Failed to parse transcript: %v", err)
	}

	response := GenerateEndstateResponse(event, metadata)

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
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript-failure.jsonl")

	// Create mock transcript with failure metadata
	f, _ := os.Create(transcriptPath)
	defer f.Close()
	f.WriteString(`{"type":"agent_start","agent_id":"python-pro","model":"sonnet"}` + "\n")
	f.WriteString(`{"type":"completion","exit_code":1,"output_tokens":512,"duration_ms":3000}` + "\n")

	// Uses ACTUAL SubagentStop schema
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-failure",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse transcript for agent metadata
	metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Failed to parse transcript: %v", err)
	}

	if metadata.ExitCode == 0 {
		t.Error("Should detect failure from transcript")
	}

	response := GenerateEndstateResponse(event, metadata)

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
		model           string
		expectedDecision string
		shouldPrompt    bool
	}{
		{"orchestrator", "sonnet", "prompt", true},
		{"architect", "sonnet", "prompt", true},
		{"python-pro", "sonnet", "prompt", true},
		{"code-reviewer", "haiku", "prompt", true},
		{"haiku-scout", "haiku", "silent", false},
		{"codebase-search", "haiku", "silent", false},
	}

	for _, tc := range tests {
		// Use t.TempDir() for each test case
		tmpDir := t.TempDir()
		transcriptPath := filepath.Join(tmpDir, tc.agentID+"-transcript.jsonl")

		// Create mock transcript for this agent
		createMockTranscript(t, transcriptPath, tc.agentID, tc.model)

		// Uses ACTUAL SubagentStop schema
		event := &SubagentStopEvent{
			HookEventName:  "SubagentStop",
			SessionID:      "test-session-" + tc.agentID,
			TranscriptPath: transcriptPath,
			StopHookActive: true,
		}

		// Parse transcript for metadata
		metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
		if err != nil {
			t.Fatalf("Agent %s: failed to parse transcript: %v", tc.agentID, err)
		}

		response := GenerateEndstateResponse(event, metadata)

		if response.Decision != tc.expectedDecision {
			t.Errorf("Agent %s: expected decision %s, got %s",
				tc.agentID, tc.expectedDecision, response.Decision)
		}
	}
}

func TestAgentEndstateWorkflow_SimulationHarness(t *testing.T) {
	// Simulation harness integration test example
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "sim-transcript.jsonl")

	// Simulate complete agent lifecycle
	createMockTranscript(t, transcriptPath, "python-pro", "sonnet")

	event := &SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "sim-session-001",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Simulation: failed to parse transcript: %v", err)
	}

	response := GenerateEndstateResponse(event, metadata)

	// Verify complete workflow produces valid output
	if response.Decision == "" {
		t.Error("Simulation: response should have decision")
	}

	// Verify logging integration
	if err := LogEndstate(event, response); err != nil {
		t.Errorf("Simulation: logging failed: %v", err)
	}
}
```

**Acceptance Criteria**:
- [x] Uses ACTUAL SubagentStop schema (session_id, transcript_path, hook_event_name, stop_hook_active)
- [x] createMockTranscript() helper creates realistic transcript files
- [x] Tests use t.TempDir() for isolation (no global state pollution)
- [x] Full workflow (event → transcript parsing → metadata → response → log) works end-to-end
- [x] All agent classes tested with proper model mapping
- [x] Success and failure paths tested via transcript metadata
- [x] Response JSON is valid and contains expected fields
- [x] Simulation harness integration test added (TestAgentEndstateWorkflow_SimulationHarness)
- [x] Integration tests verify multi-component interaction
- [x] Graceful degradation when transcript parsing fails
- [x] `go test ./pkg/workflow` passes

**Why This Matters**: Integration tests catch workflow issues that unit tests miss.

---
