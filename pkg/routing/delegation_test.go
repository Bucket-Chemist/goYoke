package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDelegationRequirement_MozartValid(t *testing.T) {
	// Mozart requires 3 delegations, has 3 - should pass
	err := ValidateDelegationRequirement("mozart", 3)
	if err != nil {
		t.Errorf("Expected no error for mozart with 3 children, got: %v", err)
	}
}

func TestValidateDelegationRequirement_MozartInvalid(t *testing.T) {
	// Mozart requires 3 delegations, has 2 - should fail
	err := ValidateDelegationRequirement("mozart", 2)
	if err == nil {
		t.Error("Expected error for mozart with 2 children, got nil")
	}
	expectedMsg := "must delegate to at least 3 child agents (actual: 2)"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedMsg, err)
	}
}

func TestValidateDelegationRequirement_MozartZeroChildren(t *testing.T) {
	// Mozart requires 3 delegations, has 0 - should fail
	err := ValidateDelegationRequirement("mozart", 0)
	if err == nil {
		t.Error("Expected error for mozart with 0 children, got nil")
	}
	expectedMsg := "must delegate to at least 3 child agents (actual: 0)"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedMsg, err)
	}
}

func TestValidateDelegationRequirement_ReviewOrchestratorValid(t *testing.T) {
	// review-orchestrator requires 2 delegations, has 2 - should pass
	err := ValidateDelegationRequirement("review-orchestrator", 2)
	if err != nil {
		t.Errorf("Expected no error for review-orchestrator with 2 children, got: %v", err)
	}
}

func TestValidateDelegationRequirement_ReviewOrchestratorInvalid(t *testing.T) {
	// review-orchestrator requires 2 delegations, has 1 - should fail
	err := ValidateDelegationRequirement("review-orchestrator", 1)
	if err == nil {
		t.Error("Expected error for review-orchestrator with 1 child, got nil")
	}
	expectedMsg := "must delegate to at least 2 child agents (actual: 1)"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedMsg, err)
	}
}

func TestValidateDelegationRequirement_GoProNoRequirement(t *testing.T) {
	// go-pro has no delegation requirement - should pass with 0 children
	err := ValidateDelegationRequirement("go-pro", 0)
	if err != nil {
		t.Errorf("Expected no error for go-pro with 0 children, got: %v", err)
	}
}

func TestValidateDelegationRequirement_UnknownAgentValid(t *testing.T) {
	// Unknown agent - should pass (fail-open)
	err := ValidateDelegationRequirement("unknown-agent-123", 0)
	if err != nil {
		t.Errorf("Expected no error for unknown agent, got: %v", err)
	}
}

func TestValidateDelegationRequirement_MoreThanMinimum(t *testing.T) {
	// Mozart requires 3, has 5 - should pass
	err := ValidateDelegationRequirement("mozart", 5)
	if err != nil {
		t.Errorf("Expected no error for mozart with 5 children, got: %v", err)
	}
}

func TestBlockResponseForDelegation(t *testing.T) {
	response := BlockResponseForDelegation("mozart", 3, 1)

	// Validate structure
	if response.Decision != "block" {
		t.Errorf("Expected decision=block, got: %s", response.Decision)
	}

	if response.Reason == "" {
		t.Error("Expected non-empty reason")
	}

	if !contains(response.Reason, "mozart") {
		t.Errorf("Expected reason to mention agent ID, got: %s", response.Reason)
	}

	if !contains(response.Reason, "at least 3") {
		t.Errorf("Expected reason to mention requirement, got: %s", response.Reason)
	}

	// Check hookSpecificOutput fields
	hookOutput := response.HookSpecificOutput
	if hookOutput == nil {
		t.Fatal("Expected hookSpecificOutput to be present")
	}

	if hookOutput["hookEventName"] != "SubagentStop" {
		t.Errorf("Expected hookEventName=SubagentStop, got: %v", hookOutput["hookEventName"])
	}

	if hookOutput["agentId"] != "mozart" {
		t.Errorf("Expected agentId=mozart, got: %v", hookOutput["agentId"])
	}

	if hookOutput["requiredDelegations"] != 3 {
		t.Errorf("Expected requiredDelegations=3, got: %v", hookOutput["requiredDelegations"])
	}

	if hookOutput["actualDelegations"] != 1 {
		t.Errorf("Expected actualDelegations=1, got: %v", hookOutput["actualDelegations"])
	}

	if hookOutput["permissionDecision"] != "deny" {
		t.Errorf("Expected permissionDecision=deny, got: %v", hookOutput["permissionDecision"])
	}

	suggestion, ok := hookOutput["suggestion"].(string)
	if !ok || suggestion == "" {
		t.Error("Expected non-empty suggestion string")
	}

	// Validate response against schema
	if err := response.Validate(); err != nil {
		t.Errorf("Response failed validation: %v", err)
	}
}

func TestAllowResponseForDelegation(t *testing.T) {
	response := AllowResponseForDelegation("mozart", 3, 3)

	// Validate structure
	if response.Decision != DecisionApprove {
		t.Errorf("Expected decision=%s, got: %s", DecisionApprove, response.Decision)
	}

	if response.Reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Check hookSpecificOutput fields
	hookOutput := response.HookSpecificOutput
	if hookOutput == nil {
		t.Fatal("Expected hookSpecificOutput to be present")
	}

	if hookOutput["hookEventName"] != "SubagentStop" {
		t.Errorf("Expected hookEventName=SubagentStop, got: %v", hookOutput["hookEventName"])
	}

	if hookOutput["agentId"] != "mozart" {
		t.Errorf("Expected agentId=mozart, got: %v", hookOutput["agentId"])
	}

	if hookOutput["requiredDelegations"] != 3 {
		t.Errorf("Expected requiredDelegations=3, got: %v", hookOutput["requiredDelegations"])
	}

	if hookOutput["actualDelegations"] != 3 {
		t.Errorf("Expected actualDelegations=3, got: %v", hookOutput["actualDelegations"])
	}

	if hookOutput["permissionDecision"] != "allow" {
		t.Errorf("Expected permissionDecision=allow, got: %v", hookOutput["permissionDecision"])
	}

	// Validate response against schema
	if err := response.Validate(); err != nil {
		t.Errorf("Response failed validation: %v", err)
	}
}

func TestLoadAgentsIndex_FileNotFound(t *testing.T) {
	// Save original env vars
	origAgentsIndex := os.Getenv("GOYOKE_AGENTS_INDEX")
	origProjectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	defer func() {
		os.Setenv("GOYOKE_AGENTS_INDEX", origAgentsIndex)
		os.Setenv("GOYOKE_PROJECT_DIR", origProjectDir)
		ClearAgentsIndexCache()
	}()

	// Point to non-existent file
	os.Setenv("GOYOKE_AGENTS_INDEX", "/nonexistent/path/agents-index.json")
	os.Setenv("GOYOKE_PROJECT_DIR", "")
	ClearAgentsIndexCache()

	_, err := LoadAgentsIndexCached()
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestValidateDelegationRequirement_ConfigLoadError(t *testing.T) {
	// Save original env vars
	origAgentsIndex := os.Getenv("GOYOKE_AGENTS_INDEX")
	origProjectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	defer func() {
		os.Setenv("GOYOKE_AGENTS_INDEX", origAgentsIndex)
		os.Setenv("GOYOKE_PROJECT_DIR", origProjectDir)
		ClearAgentsIndexCache()
	}()

	// Point to non-existent file
	os.Setenv("GOYOKE_AGENTS_INDEX", "/nonexistent/path/agents-index.json")
	os.Setenv("GOYOKE_PROJECT_DIR", "")
	ClearAgentsIndexCache()

	// Should fail-open (return no error)
	err := ValidateDelegationRequirement("mozart", 0)
	if err != nil {
		t.Errorf("Expected fail-open behavior (no error), got: %v", err)
	}
}

func TestGetAgentDelegationConfig_Mozart(t *testing.T) {
	config := GetAgentDelegationConfig("mozart")
	if !config.MustDelegate {
		t.Error("Expected mozart.must_delegate=true")
	}
	if config.MinDelegations != 3 {
		t.Errorf("Expected mozart.min_delegations=3, got: %d", config.MinDelegations)
	}
}

func TestGetAgentDelegationConfig_ReviewOrchestrator(t *testing.T) {
	config := GetAgentDelegationConfig("review-orchestrator")
	if !config.MustDelegate {
		t.Error("Expected review-orchestrator.must_delegate=true")
	}
	if config.MinDelegations != 2 {
		t.Errorf("Expected review-orchestrator.min_delegations=2, got: %d", config.MinDelegations)
	}
}

func TestGetAgentDelegationConfig_GoPro(t *testing.T) {
	config := GetAgentDelegationConfig("go-pro")
	if config.MustDelegate {
		t.Error("Expected go-pro.must_delegate=false")
	}
	if config.MinDelegations != 0 {
		t.Errorf("Expected go-pro.min_delegations=0, got: %d", config.MinDelegations)
	}
}

func TestGetAgentDelegationConfig_UnknownAgent(t *testing.T) {
	config := GetAgentDelegationConfig("nonexistent-agent")
	if config.MustDelegate {
		t.Error("Expected unknown agent must_delegate=false (fail-open)")
	}
	if config.MinDelegations != 0 {
		t.Errorf("Expected unknown agent min_delegations=0, got: %d", config.MinDelegations)
	}
}

func TestCountChildAgentsFromTranscript(t *testing.T) {
	// Create temporary transcript file in JSONL format (ToolEvent per line)
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Create ToolEvent JSON objects
	events := []ToolEvent{
		{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 1000},
		{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 2000},
		{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 3000},
		{ToolName: "Read", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 4000},
	}

	var lines []string
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		lines = append(lines, string(data))
	}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	count, err := CountChildAgentsFromTranscript(transcriptPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 Task invocations, got: %d", count)
	}
}

func TestCountChildAgentsFromTranscript_NoTaskTools(t *testing.T) {
	// Create temporary transcript file in JSONL format (ToolEvent per line)
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Create ToolEvent JSON objects
	events := []ToolEvent{
		{ToolName: "Read", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 1000},
		{ToolName: "Write", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 2000},
		{ToolName: "Edit", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 3000},
	}

	var lines []string
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		lines = append(lines, string(data))
	}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	count, err := CountChildAgentsFromTranscript(transcriptPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 Task invocations, got: %d", count)
	}
}

func TestCountChildAgentsFromTranscript_FileNotFound(t *testing.T) {
	_, err := CountChildAgentsFromTranscript("/nonexistent/transcript.json")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestCountChildAgentsFromTranscript_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(transcriptPath, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := CountChildAgentsFromTranscript(transcriptPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestValidateDelegationFromTranscript_Success(t *testing.T) {
	// Create temporary transcript with mozart spawning 3 children
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Metadata line + ToolEvent lines
	metadataLine := `{"content":"AGENT: mozart","timestamp":1000}`
	event1, _ := json.Marshal(ToolEvent{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 2000})
	event2, _ := json.Marshal(ToolEvent{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 3000})
	event3, _ := json.Marshal(ToolEvent{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 4000})

	lines := []string{metadataLine, string(event1), string(event2), string(event3)}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	response, err := ValidateDelegationFromTranscript(transcriptPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if response.Decision != DecisionApprove {
		t.Errorf("Expected decision=%s, got: %s", DecisionApprove, response.Decision)
	}
}

func TestValidateDelegationFromTranscript_Violation(t *testing.T) {
	// Create temporary transcript with mozart spawning only 1 child
	// Must use ToolEvent JSONL format for ParseTranscript + content field for ParseTranscriptForMetadata
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// First line: metadata with AGENT: prefix for ParseTranscriptForMetadata
	// Following lines: ToolEvent JSON for CountChildAgentsFromTranscript
	metadataLine := `{"content":"AGENT: mozart","timestamp":1000}`

	// ToolEvent lines
	event1, _ := json.Marshal(ToolEvent{ToolName: "Task", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 2000})
	event2, _ := json.Marshal(ToolEvent{ToolName: "Read", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 3000})

	lines := []string{metadataLine, string(event1), string(event2)}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	response, err := ValidateDelegationFromTranscript(transcriptPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if response.Decision != DecisionBlock {
		t.Errorf("Expected decision=%s, got: %s (reason: %s)", DecisionBlock, response.Decision, response.Reason)
	}

	// Verify hookSpecificOutput
	hookOutput := response.HookSpecificOutput
	if hookOutput["agentId"] != "mozart" {
		t.Errorf("Expected agentId=mozart, got: %v", hookOutput["agentId"])
	}
	if hookOutput["requiredDelegations"] != 3 {
		t.Errorf("Expected requiredDelegations=3, got: %v", hookOutput["requiredDelegations"])
	}
	if hookOutput["actualDelegations"] != 1 {
		t.Errorf("Expected actualDelegations=1, got: %v", hookOutput["actualDelegations"])
	}
}

func TestValidateDelegationFromTranscript_NoRequirement(t *testing.T) {
	// Create temporary transcript with go-pro (no delegation requirement)
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Metadata line + ToolEvent lines
	metadataLine := `{"content":"AGENT: go-pro","timestamp":1000}`
	event1, _ := json.Marshal(ToolEvent{ToolName: "Read", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 2000})
	event2, _ := json.Marshal(ToolEvent{ToolName: "Write", SessionID: "test", HookEventName: "PreToolUse", CapturedAt: 3000})

	lines := []string{metadataLine, string(event1), string(event2)}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	response, err := ValidateDelegationFromTranscript(transcriptPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if response.Decision != DecisionApprove {
		t.Errorf("Expected decision=%s, got: %s", DecisionApprove, response.Decision)
	}

	if !contains(response.Reason, "no delegation requirement") {
		t.Errorf("Expected reason to mention no requirement, got: %s", response.Reason)
	}
}

func TestValidateDelegationFromTranscript_TranscriptParsingError(t *testing.T) {
	// Point to non-existent transcript
	response, err := ValidateDelegationFromTranscript("/nonexistent/transcript.json")
	if err != nil {
		t.Errorf("Expected no error (fail-open), got: %v", err)
	}

	if response.Decision != DecisionApprove {
		t.Errorf("Expected decision=%s (fail-open), got: %s", DecisionApprove, response.Decision)
	}

	if !contains(response.Reason, "parsing failed") {
		t.Errorf("Expected reason to mention parsing failure, got: %s", response.Reason)
	}
}

func TestClearAgentsIndexCache(t *testing.T) {
	// Load index to populate cache
	_, _ = LoadAgentsIndexCached()

	// Verify cache is populated
	agentsIndexMutex.RLock()
	cachePopulated := agentsIndexCache != nil
	agentsIndexMutex.RUnlock()

	if !cachePopulated {
		t.Skip("Cache not populated, skipping clear test")
	}

	// Clear cache
	ClearAgentsIndexCache()

	// Verify cache is cleared
	agentsIndexMutex.RLock()
	cacheCleared := agentsIndexCache == nil
	agentsIndexMutex.RUnlock()

	if !cacheCleared {
		t.Error("Expected cache to be cleared")
	}
}
