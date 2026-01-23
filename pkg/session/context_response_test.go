package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSessionStartResponse_Startup(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:    "startup",
		RoutingSummary: "ROUTING TIERS ACTIVE:\n  • haiku: find files...",
		GitInfo:        "GIT: Branch: main | Uncommitted: 2 file(s)",
		ProjectInfo: &ProjectDetectionResult{
			Type:        ProjectGo,
			Indicators:  []string{"go.mod"},
			Conventions: []string{"go.md"},
		},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	// Verify valid JSON
	var parsed SessionStartResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Verify hook event name
	if parsed.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("Expected hookEventName 'SessionStart', got: %s", parsed.HookSpecificOutput.HookEventName)
	}

	context := parsed.HookSpecificOutput.AdditionalContext

	// Verify content
	if !strings.Contains(context, "SESSION INITIALIZED (startup)") {
		t.Error("Should indicate startup session")
	}

	if !strings.Contains(context, "ROUTING TIERS") {
		t.Error("Should include routing summary")
	}

	if !strings.Contains(context, "GIT:") {
		t.Error("Should include git info")
	}

	if !strings.Contains(context, "go") {
		t.Error("Should include project type")
	}

	if !strings.Contains(context, "hooks are ACTIVE") {
		t.Error("Should include hook status footer")
	}
}

func TestGenerateSessionStartResponse_Resume(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:      "resume",
		RoutingSummary:   "ROUTING TIERS ACTIVE:\n  • haiku: find files...",
		HandoffSummary:   "# Session Handoff\n\nLast session completed feature X.",
		PendingLearnings: "⚠️ PENDING LEARNINGS: 3 sharp edge(s)",
		ProjectInfo: &ProjectDetectionResult{
			Type: ProjectGeneric,
		},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	var parsed SessionStartResponse
	json.Unmarshal([]byte(response), &parsed)
	context := parsed.HookSpecificOutput.AdditionalContext

	// Verify resume-specific content
	if !strings.Contains(context, "SESSION INITIALIZED (resume)") {
		t.Error("Should indicate resume session")
	}

	if !strings.Contains(context, "PREVIOUS SESSION HANDOFF") {
		t.Error("Should include handoff header for resume")
	}

	if !strings.Contains(context, "feature X") {
		t.Error("Should include handoff content")
	}

	if !strings.Contains(context, "PENDING LEARNINGS") {
		t.Error("Should include pending learnings warning")
	}
}

func TestGenerateSessionStartResponse_StartupNoHandoff(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:    "startup",
		HandoffSummary: "# Some handoff content", // Should be ignored for startup
		ProjectInfo:    &ProjectDetectionResult{Type: ProjectGeneric},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	var parsed SessionStartResponse
	json.Unmarshal([]byte(response), &parsed)
	context := parsed.HookSpecificOutput.AdditionalContext

	// Handoff should NOT be included for startup
	if strings.Contains(context, "PREVIOUS SESSION HANDOFF") {
		t.Error("Startup session should not include handoff")
	}
}

func TestGenerateSessionStartResponse_Nil(t *testing.T) {
	_, err := GenerateSessionStartResponse(nil)

	if err == nil {
		t.Error("Expected error for nil context, got nil")
	}
}

func TestGenerateErrorResponse(t *testing.T) {
	response := GenerateErrorResponse("Test error message")

	var parsed SessionStartResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "SESSION START ERROR") {
		t.Error("Should contain error indicator")
	}

	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "Test error message") {
		t.Error("Should contain error message")
	}
}
