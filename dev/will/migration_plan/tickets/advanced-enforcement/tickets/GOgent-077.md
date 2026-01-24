---
id: GOgent-077
title: Blocking Response Generation
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-076"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-077: Blocking Response Generation

**Time**: 1.5 hours
**Dependencies**: GOgent-076

**Task**:
Generate blocking response if background tasks uncollected.

**File**: `pkg/enforcement/blocking_response.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"strings"
)
```

**Implementation**:
```go
// GuardResponse represents orchestrator-guard decision
type GuardResponse struct {
	HookEventName     string `json:"hookEventName"`
	Decision          string `json:"decision"` // "allow" or "block"
	Reason            string `json:"reason"`
	AdditionalContext string `json:"additionalContext"`
	RemediationSteps  []string `json:"remediation_steps"`
}

// GenerateGuardResponse decides whether to allow orchestrator completion
func GenerateGuardResponse(analyzer *TranscriptAnalyzer, event *OrchestratorStopEvent) *GuardResponse {
	response := &GuardResponse{
		HookEventName: "SubagentStop",
	}

	// If no background tasks, allow completion
	if !analyzer.HasUncollectedTasks() {
		response.Decision = "allow"
		response.Reason = "No uncollected background tasks detected"
		response.AdditionalContext = fmt.Sprintf(
			"✅ ORCHESTRATOR COMPLETION ALLOWED\n\n"+
				"Agent: %s\n"+
				"Summary: %s\n\n"+
				"Safe to proceed with next steps.",
			event.AgentID,
			analyzer.GetSummary(),
		)
		return response
	}

	// Uncollected tasks - BLOCK
	response.Decision = "block"
	response.Reason = "Background tasks not collected"
	response.AdditionalContext = fmt.Sprintf(
		"🛑 ORCHESTRATOR COMPLETION BLOCKED\n\n"+
			"Agent: %s\nStatus: %s\n\n"+
			"VIOLATION: Background task fan-out/fan-in pattern not completed.\n\n"+
			"%s\n\n"+
			"From LLM-guidelines.md § 2.2 MANDATORY: Background Task Collection:\n"+
			"\"If you spawn background tasks, you MUST call TaskOutput() before concluding.\"\n\n"+
			"REQUIRED ACTIONS:\n"+
			"1. Spawn any missing TaskOutput calls\n"+
			"2. Use task_id from uncollected tasks above\n"+
			"3. Set block: true to wait for results\n"+
			"4. Proceed once all TaskOutput calls complete",
		event.AgentID,
		analyzer.GetSummary(),
		analyzer.GetUncollectedList(),
	)
	response.RemediationSteps = []string{
		"identify_uncollected_task_ids",
		"call_TaskOutput_for_each",
		"wait_for_all_collections",
		"verify_results_in_transcript",
	}

	return response
}

// FormatResponseJSON creates hook response format
func (r *GuardResponse) FormatJSON() string {
	remedJson := formatStringArray(r.RemediationSteps)

	output := fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "%s",
    "decision": "%s",
    "reason": "%s",
    "additionalContext": "%s",
    "remediation": %s
  }
}`,
		r.HookEventName,
		r.Decision,
		escapeJSON(r.Reason),
		escapeJSON(r.AdditionalContext),
		remedJson,
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

func formatStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	var quoted []string
	for _, item := range arr {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, item))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
```

**Tests**: `pkg/enforcement/blocking_response_test.go`

```go
package enforcement

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateGuardResponse_AllowCompletion(t *testing.T) {
	// No uncollected tasks
	analyzer := &TranscriptAnalyzer{
		tracker: &TaskTracker{
			SpawnedCount:   0,
			CollectedCount: 0,
		},
	}

	event := &OrchestratorStopEvent{
		AgentID: "orchestrator",
	}

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Errorf("Expected allow, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "ALLOWED") {
		t.Error("Should indicate completion allowed")
	}
}

func TestGenerateGuardResponse_BlockCompletion(t *testing.T) {
	// Uncollected tasks
	analyzer := &TranscriptAnalyzer{
		tracker: &TaskTracker{
			SpawnedCount:    3,
			CollectedCount:  1,
			UncollectedIDs:  []string{"bg-2", "bg-3"},
		},
	}

	event := &OrchestratorStopEvent{
		AgentID: "orchestrator",
	}

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "block" {
		t.Errorf("Expected block, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "BLOCKED") {
		t.Error("Should indicate completion blocked")
	}

	if !strings.Contains(response.AdditionalContext, "2 uncollected") {
		t.Error("Should mention uncollected count")
	}

	if !strings.Contains(response.AdditionalContext, "TaskOutput") {
		t.Error("Should mention TaskOutput fix")
	}
}

func TestFormatResponseJSON_Valid(t *testing.T) {
	response := &GuardResponse{
		HookEventName:    "SubagentStop",
		Decision:         "block",
		Reason:           "Test reason",
		AdditionalContext: "Test context",
		RemediationSteps: []string{"step1", "step2"},
	}

	jsonStr := response.FormatJSON()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if _, ok := parsed["hookSpecificOutput"]; !ok {
		t.Fatal("Missing hookSpecificOutput")
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateGuardResponse()` allows if no uncollected tasks
- [ ] Blocks if uncollected tasks detected
- [ ] Block response includes remediation steps
- [ ] References LLM-guidelines.md fan-out/fan-in pattern
- [ ] `FormatResponseJSON()` outputs valid JSON
- [ ] Tests verify allow and block paths
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Blocking response enforces fan-out/fan-in discipline programmatically.

---
