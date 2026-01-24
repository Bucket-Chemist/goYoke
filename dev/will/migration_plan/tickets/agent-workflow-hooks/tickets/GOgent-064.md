---
id: GOgent-064
title: Tier-Specific Response Generation
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-063"]
priority: high
week: 4
tags: ["agent-endstate", "week-4", "schema-corrected"]
tests_required: true
acceptance_criteria_count: 10
---

### GOgent-064: Tier-Specific Response Generation

**Time**: 2 hours
**Dependencies**: GOgent-063 (UPDATED: now uses transcript-parsed metadata)

**CRITICAL UPDATE**: Function signature changed to accept ParsedAgentMetadata from transcript parsing.

**Task**:
Generate appropriate follow-up responses based on agent class and tier using transcript-parsed metadata.

**File**: `pkg/workflow/responses.go`

**Imports**:
```go
package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)
```

**Implementation**:
```go
// EndstateResponse represents the response to SubagentStop
type EndstateResponse struct {
	HookEventName     string   `json:"hookEventName"`
	Decision          string   `json:"decision"` // "prompt", "silent"
	AdditionalContext string   `json:"additionalContext"`
	Tier              string   `json:"tier,omitempty"`
	AgentClass        string   `json:"agentClass,omitempty"`
	Recommendations   []string `json:"recommendations,omitempty"`
}

// GenerateEndstateResponse creates tier-specific response based on agent completion.
// If metadata is nil, generates generic response (graceful degradation).
func GenerateEndstateResponse(event *SubagentStopEvent, metadata *ParsedAgentMetadata) *EndstateResponse {
	// Graceful degradation: if metadata parsing failed, use defaults
	if metadata == nil {
		metadata = &ParsedAgentMetadata{
			AgentID:  "unknown",
			Tier:     "unknown",
			ExitCode: 0,
		}
	}

	agentClass := GetAgentClass(metadata.AgentID)
	isSuccess := metadata.IsSuccess()

	response := &EndstateResponse{
		HookEventName: "SubagentStop",
		Tier:          metadata.Tier,
		AgentClass:    string(agentClass),
	}

	if !isSuccess {
		// Agent failed - always prompt
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"⚠️ AGENT FAILED\n\nAgent: %s (tier: %s)\nExit Code: %d\nDuration: %dms\n\n"+
				"Reasons to investigate:\n"+
				"• Check error logs\n"+
				"• Review agent transcript for blocker\n"+
				"• Consider escalation to higher tier\n"+
				"• Retry with modified prompt or scope",
			metadata.AgentID, metadata.Tier, metadata.ExitCode, metadata.DurationMs)
		response.Recommendations = []string{
			"review_error_cause",
			"check_transcript",
			"consider_escalation",
		}
		return response
	}

	// Agent succeeded - tier-specific prompts
	switch agentClass {
	case ClassOrchestrator:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ ORCHESTRATOR COMPLETED\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Orchestration checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Have you updated TODOs based on decisions made?\n"+
				"3. [ ] Did the agent spawn background tasks? Collected all results?\n"+
				"4. [ ] Should architectural decisions be captured in memory?\n"+
				"5. [ ] Are any follow-up tickets needed?\n\n"+
				"Recommended next step: Capture key decisions and verify background task collection.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"update_todos",
			"verify_background_collection",
			"capture_decisions",
			"knowledge_compound",
		}

	case ClassImplementation:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ IMPLEMENTATION COMPLETE\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Implementation checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Did tests pass (if agent added tests)?\n"+
				"3. [ ] Review implementation against conventions (python.md, go.md, etc.)\n"+
				"4. [ ] Any integration issues with existing code?\n"+
				"5. [ ] Document any workarounds or tradeoffs\n\n"+
				"Recommended next step: Verify test coverage and review against style conventions.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"verify_tests",
			"review_conventions",
			"check_integration",
		}

	case ClassSpecialist:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ SPECIALIST COMPLETED\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Specialist checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Output meets quality standards?\n"+
				"3. [ ] Follow-up actions identified in output?\n"+
				"4. [ ] Any issues need escalation?\n\n"+
				"Recommended next step: Review specialist output and execute follow-up actions.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"review_output",
			"execute_followups",
		}

	case ClassCoordination:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf(
			"✅ Coordination agent %s completed in %dms",
			metadata.AgentID, metadata.DurationMs)
		response.Recommendations = []string{
			"continue_workflow",
		}

	default:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf("Agent %s completed (exit: %d)", metadata.AgentID, metadata.ExitCode)
	}

	return response
}

// Marshal writes the EndstateResponse as JSON to the provided writer.
// Replaces manual JSON formatting with json.Marshal for robustness.
func (r *EndstateResponse) Marshal(w io.Writer) error {
	// Wrap in hookSpecificOutput structure
	wrapper := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     r.HookEventName,
			"decision":          r.Decision,
			"additionalContext": r.AdditionalContext,
			"metadata": map[string]interface{}{
				"tier":            r.Tier,
				"agentClass":      r.AgentClass,
				"recommendations": r.Recommendations,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(wrapper); err != nil {
		return fmt.Errorf("[agent-endstate] Failed to marshal JSON: %w", err)
	}
	return nil
}
```

**Tests**: `pkg/workflow/responses_test.go`

```go
package workflow

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateEndstateResponse_OrchestratorSuccess(t *testing.T) {
	event := &SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &ParsedAgentMetadata{
		AgentID:      "orchestrator",
		AgentModel:   "claude-sonnet-4",
		Tier:         "sonnet",
		ExitCode:     0,
		DurationMs:   5000,
		OutputTokens: 2048,
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "ORCHESTRATOR COMPLETED") {
		t.Error("Should indicate orchestrator completion")
	}

	if !strings.Contains(response.AdditionalContext, "background tasks") {
		t.Error("Should mention background task verification")
	}

	if !contains(response.Recommendations, "verify_background_collection") {
		t.Error("Should recommend background task verification")
	}
}

func TestGenerateEndstateResponse_NilMetadata(t *testing.T) {
	event := &SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	// Graceful degradation: nil metadata should not crash
	response := GenerateEndstateResponse(event, nil)

	if response == nil {
		t.Fatal("Should return response even with nil metadata")
	}

	if response.Decision != "silent" {
		t.Errorf("Expected silent decision for unknown agent, got: %s", response.Decision)
	}

	if response.Tier != "unknown" {
		t.Errorf("Expected unknown tier, got: %s", response.Tier)
	}
}

func TestGenerateEndstateResponse_ImplementationSuccess(t *testing.T) {
	event := &SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &ParsedAgentMetadata{
		AgentID:      "python-pro",
		AgentModel:   "claude-sonnet-4",
		Tier:         "sonnet",
		ExitCode:     0,
		DurationMs:   3000,
		OutputTokens: 1024,
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "IMPLEMENTATION COMPLETE") {
		t.Error("Should indicate implementation completion")
	}

	if !contains(response.Recommendations, "verify_tests") {
		t.Error("Should recommend test verification")
	}
}

func TestGenerateEndstateResponse_Failure(t *testing.T) {
	event := &SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &ParsedAgentMetadata{
		AgentID:      "orchestrator",
		AgentModel:   "claude-sonnet-4",
		Tier:         "sonnet",
		ExitCode:     1, // Failure
		DurationMs:   2000,
		OutputTokens: 512,
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision on failure, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "AGENT FAILED") {
		t.Error("Should indicate failure")
	}

	if !strings.Contains(response.AdditionalContext, "Exit Code: 1") {
		t.Error("Should include exit code")
	}
}

func TestGenerateEndstateResponse_CoordinationAgent(t *testing.T) {
	event := &SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &ParsedAgentMetadata{
		AgentID:      "haiku-scout",
		AgentModel:   "claude-haiku-4",
		Tier:         "haiku",
		ExitCode:     0,
		DurationMs:   1000,
		OutputTokens: 256,
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "silent" {
		t.Errorf("Expected silent decision for coordination agent, got: %s", response.Decision)
	}
}

func TestMarshal_ValidJSON(t *testing.T) {
	response := &EndstateResponse{
		HookEventName:     "SubagentStop",
		Decision:          "prompt",
		AdditionalContext: "Test context with \"quotes\" and\nnewlines",
		Tier:              "sonnet",
		AgentClass:        "orchestrator",
		Recommendations:   []string{"update_todos", "verify_tests"},
	}

	var buf bytes.Buffer
	if err := response.Marshal(&buf); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v. Output: %s", err, buf.String())
	}

	hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput")
	}

	if hookOutput["decision"] != "prompt" {
		t.Errorf("Expected decision=prompt, got: %v", hookOutput["decision"])
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

**Acceptance Criteria**:
- [ ] `GenerateEndstateResponse()` accepts (event, metadata) signature (UPDATED)
- [ ] Graceful degradation when metadata is nil (uses defaults)
- [ ] Orchestrator: prompts for TODO updates and background task verification
- [ ] Implementation: prompts for test verification and convention review
- [ ] Specialist: prompts for output review and follow-up execution
- [ ] Coordination: silent (no prompt)
- [ ] Failed agents: always prompt with error context
- [ ] `Marshal()` uses json.Encoder for robust output (replaces manual formatting)
- [ ] Tests verify all agent classes, success/failure paths, AND nil metadata handling
- [ ] `go test ./pkg/workflow` passes

**Why This Matters**: Response generation drives user experience. Each agent class needs different follow-up prompts to enforce discipline. Graceful degradation ensures the hook never crashes even if transcript parsing fails.

---
