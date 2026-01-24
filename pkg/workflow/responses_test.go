package workflow

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestGenerateEndstateResponse_OrchestratorSuccess(t *testing.T) {
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
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
	event := &routing.SubagentStopEvent{
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
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
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
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
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
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
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

func TestGenerateEndstateResponse_SpecialistSuccess(t *testing.T) {
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:      "tech-docs-writer",
		AgentModel:   "claude-haiku-4",
		Tier:         "haiku",
		ExitCode:     0,
		DurationMs:   2500,
		OutputTokens: 800,
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "SPECIALIST COMPLETED") {
		t.Error("Should indicate specialist completion")
	}

	if !contains(response.Recommendations, "review_output") {
		t.Error("Should recommend output review")
	}

	if !contains(response.Recommendations, "execute_followups") {
		t.Error("Should recommend executing follow-ups")
	}
}

func TestGenerateEndstateResponse_AllAgentClasses(t *testing.T) {
	tests := []struct {
		name           string
		agentID        string
		expectedClass  string
		expectedPrompt bool
	}{
		{"Orchestrator", "orchestrator", "orchestrator", true},
		{"Architect", "architect", "orchestrator", true},
		{"Einstein", "einstein", "orchestrator", true},
		{"PythonPro", "python-pro", "implementation", true},
		{"GoPro", "go-pro", "implementation", true},
		{"CodeReviewer", "code-reviewer", "specialist", true},
		{"Librarian", "librarian", "specialist", true},
		{"TechDocs", "tech-docs-writer", "specialist", true},
		{"CodebaseSearch", "codebase-search", "coordination", false},
		{"HaikuScout", "haiku-scout", "coordination", false},
		{"Unknown", "unknown-agent", "unknown", false},
	}

	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &routing.ParsedAgentMetadata{
				AgentID:      tt.agentID,
				AgentModel:   "claude-sonnet-4",
				Tier:         "sonnet",
				ExitCode:     0,
				DurationMs:   1000,
				OutputTokens: 500,
			}

			response := GenerateEndstateResponse(event, metadata)

			if response.AgentClass != tt.expectedClass {
				t.Errorf("Expected agent class %s, got: %s", tt.expectedClass, response.AgentClass)
			}

			expectedDecision := "silent"
			if tt.expectedPrompt {
				expectedDecision = "prompt"
			}

			if response.Decision != expectedDecision {
				t.Errorf("Expected decision %s, got: %s", expectedDecision, response.Decision)
			}
		})
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

func TestMarshal_JSONEscaping(t *testing.T) {
	response := &EndstateResponse{
		HookEventName:     "SubagentStop",
		Decision:          "prompt",
		AdditionalContext: "Special chars: \"quotes\", \nnewlines, \ttabs, and \\backslashes",
		Tier:              "sonnet",
		AgentClass:        "orchestrator",
		Recommendations:   []string{"test"},
	}

	var buf bytes.Buffer
	if err := response.Marshal(&buf); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify JSON is parseable (proper escaping)
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON escaping failed: %v. Output: %s", err, buf.String())
	}

	hookOutput := parsed["hookSpecificOutput"].(map[string]interface{})
	context := hookOutput["additionalContext"].(string)

	// Verify special characters preserved
	if !strings.Contains(context, "\"quotes\"") {
		t.Error("Quotes should be preserved")
	}
	if !strings.Contains(context, "\n") {
		t.Error("Newlines should be preserved")
	}
}

func TestGenerateEndstateResponse_NilEvent(t *testing.T) {
	// Should not crash with nil event
	metadata := &routing.ParsedAgentMetadata{
		AgentID:      "orchestrator",
		AgentModel:   "claude-sonnet-4",
		Tier:         "sonnet",
		ExitCode:     0,
		DurationMs:   5000,
		OutputTokens: 2048,
	}

	response := GenerateEndstateResponse(nil, metadata)

	if response == nil {
		t.Fatal("Should return response even with nil event")
	}

	// Event is not used in current implementation, so response should still be valid
	if response.Decision != "prompt" {
		t.Errorf("Expected prompt decision, got: %s", response.Decision)
	}
}

func TestGenerateEndstateResponse_AllRecommendations(t *testing.T) {
	// Test that all expected recommendations are present for each agent class
	tests := []struct {
		agentID               string
		expectedRecommendations []string
	}{
		{
			"orchestrator",
			[]string{"update_todos", "verify_background_collection", "capture_decisions", "knowledge_compound"},
		},
		{
			"python-pro",
			[]string{"verify_tests", "review_conventions", "check_integration"},
		},
		{
			"tech-docs-writer",
			[]string{"review_output", "execute_followups"},
		},
		{
			"haiku-scout",
			[]string{"continue_workflow"},
		},
	}

	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			metadata := &routing.ParsedAgentMetadata{
				AgentID:      tt.agentID,
				AgentModel:   "claude-sonnet-4",
				Tier:         "sonnet",
				ExitCode:     0,
				DurationMs:   1000,
				OutputTokens: 500,
			}

			response := GenerateEndstateResponse(event, metadata)

			for _, expectedRec := range tt.expectedRecommendations {
				if !contains(response.Recommendations, expectedRec) {
					t.Errorf("Missing expected recommendation: %s", expectedRec)
				}
			}
		})
	}
}

func TestGenerateEndstateResponse_FailureRecommendations(t *testing.T) {
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:      "python-pro",
		AgentModel:   "claude-sonnet-4",
		Tier:         "sonnet",
		ExitCode:     1, // Failure
		DurationMs:   2000,
		OutputTokens: 512,
	}

	response := GenerateEndstateResponse(event, metadata)

	expectedRecommendations := []string{
		"review_error_cause",
		"check_transcript",
		"consider_escalation",
	}

	for _, expectedRec := range expectedRecommendations {
		if !contains(response.Recommendations, expectedRec) {
			t.Errorf("Missing expected failure recommendation: %s", expectedRec)
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
