---
id: GOgent-064
title: Tier-Specific Response Generation
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-063"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-064: Tier-Specific Response Generation

**Time**: 2 hours
**Dependencies**: GOgent-063

**Task**:
Generate appropriate follow-up responses based on agent class and tier.

**File**: `pkg/workflow/responses.go`

**Imports**:
```go
package workflow

import (
	"fmt"
	"strings"
)
```

**Implementation**:
```go
// EndstateResponse represents the response to SubagentStop
type EndstateResponse struct {
	HookEventName     string `json:"hookEventName"`
	Decision          string `json:"decision"` // "prompt", "silent"
	AdditionalContext string `json:"additionalContext"`
	Tier              string `json:"tier"`
	AgentClass        string `json:"agentClass"`
	Recommendations   []string `json:"recommendations"`
}

// GenerateEndstateResponse creates tier-specific response based on agent completion
func GenerateEndstateResponse(event *SubagentStopEvent) *EndstateResponse {
	agentClass := event.GetAgentClass()
	isSuccess := event.IsSuccess()

	response := &EndstateResponse{
		HookEventName: "SubagentStop",
		Tier:          event.Tier,
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
			event.AgentID, event.Tier, event.ExitCode, event.Duration)
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
			event.AgentID, event.Tier, event.Duration, event.OutputTokens)
		response.Recommendations = []string{
			"update_todos",
			"verify_background_collection",
			"capture_decisions",
			"proposal_compound",
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
			event.AgentID, event.Tier, event.Duration, event.OutputTokens)
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
			event.AgentID, event.Tier, event.Duration, event.OutputTokens)
		response.Recommendations = []string{
			"review_output",
			"execute_followups",
		}

	case ClassCoordination:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf(
			"✅ Coordination agent %s completed in %dms",
			event.AgentID, event.Duration)
		response.Recommendations = []string{
			"continue_workflow",
		}

	default:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf("Agent %s completed (exit: %d)", event.AgentID, event.ExitCode)
	}

	return response
}

// FormatResponseJSON creates hook response format
func (r *EndstateResponse) FormatJSON() string {
	output := fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "%s",
    "decision": "%s",
    "additionalContext": "%s",
    "metadata": {
      "tier": "%s",
      "agentClass": "%s",
      "recommendations": %s
    }
  }
}`,
		r.HookEventName,
		escapeJSON(r.Decision),
		escapeJSON(r.AdditionalContext),
		r.Tier,
		r.AgentClass,
		formatRecommendations(r.Recommendations),
	)
	return output
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func formatRecommendations(recs []string) string {
	if len(recs) == 0 {
		return "[]"
	}
	var quoted []string
	for _, rec := range recs {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, rec))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
```

**Tests**: `pkg/workflow/responses_test.go`

```go
package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateEndstateResponse_OrchestratorSuccess(t *testing.T) {
	event := &SubagentStopEvent{
		AgentID:      "orchestrator",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     0,
		Duration:     5000,
		OutputTokens: 2048,
	}

	response := GenerateEndstateResponse(event)

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

func TestGenerateEndstateResponse_ImplementationSuccess(t *testing.T) {
	event := &SubagentStopEvent{
		AgentID:      "python-pro",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     0,
		Duration:     3000,
		OutputTokens: 1024,
	}

	response := GenerateEndstateResponse(event)

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
		AgentID:      "orchestrator",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     1,
		Duration:     2000,
		OutputTokens: 512,
	}

	response := GenerateEndstateResponse(event)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision on failure, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "AGENT FAILED") {
		t.Error("Should indicate failure")
	}

	if !strings.Contains(response.AdditionalContext, "exit code") {
		t.Error("Should include exit code")
	}
}

func TestGenerateEndstateResponse_CoordinationAgent(t *testing.T) {
	event := &SubagentStopEvent{
		AgentID:      "haiku-scout",
		AgentModel:   "haiku",
		Tier:         "haiku",
		ExitCode:     0,
		Duration:     1000,
		OutputTokens: 256,
	}

	response := GenerateEndstateResponse(event)

	if response.Decision != "silent" {
		t.Errorf("Expected silent decision for coordination agent, got: %s", response.Decision)
	}
}

func TestFormatResponseJSON_ValidJSON(t *testing.T) {
	event := &SubagentStopEvent{
		AgentID:      "orchestrator",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     0,
		Duration:     5000,
		OutputTokens: 2048,
	}

	response := GenerateEndstateResponse(event)
	jsonStr := response.FormatJSON()

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v. Output: %s", err, jsonStr)
	}

	if _, ok := parsed["hookSpecificOutput"]; !ok {
		t.Fatal("Missing hookSpecificOutput")
	}
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`hello "world"`, `hello \"world\"`},
		{"line1\nline2", `line1\nline2`},
		{`back\slash`, `back\\slash`},
		{"tab\there", `tab\there`},
	}

	for _, tc := range tests {
		result := escapeJSON(tc.input)
		if result != tc.expected {
			t.Errorf("escapeJSON(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
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
- [ ] `GenerateEndstateResponse()` creates tier-specific responses
- [ ] Orchestrator: prompts for TODO updates and background task verification
- [ ] Implementation: prompts for test verification and convention review
- [ ] Specialist: prompts for output review and follow-up execution
- [ ] Coordination: silent (no prompt)
- [ ] Failed agents: always prompt with error context
- [ ] `FormatResponseJSON()` outputs valid JSON with proper escaping
- [ ] Tests verify all agent classes and success/failure paths
- [ ] `go test ./pkg/workflow` passes

**Why This Matters**: Response generation drives user experience. Each agent class needs different follow-up prompts to enforce discipline.

---
