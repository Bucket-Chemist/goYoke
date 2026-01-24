package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseOrchestratorStopEvent tests successful parsing of SubagentStop event
// and extraction of agent metadata from transcript file.
func TestParseOrchestratorStopEvent(t *testing.T) {
	// Create temporary transcript file
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: orchestrator", "role": "system"}
{"timestamp": 1700000500, "model": "claude-sonnet-4", "role": "assistant"}
{"timestamp": 1700005000, "content": "Planning complete", "role": "assistant"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to create transcript file: %v", err)
	}

	// Create SubagentStop event JSON
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-orch",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "orchestrator" {
		t.Errorf("Expected agent_id 'orchestrator', got: %s", metadata.AgentID)
	}

	if metadata.AgentModel != "claude-sonnet-4" {
		t.Errorf("Expected model 'claude-sonnet-4', got: %s", metadata.AgentModel)
	}

	if metadata.Tier != "sonnet" {
		t.Errorf("Expected tier 'sonnet', got: %s", metadata.Tier)
	}

	if metadata.DurationMs != 5000 {
		t.Errorf("Expected duration 5000ms, got: %d", metadata.DurationMs)
	}

	if !metadata.IsSuccess() {
		t.Error("Expected success (exit_code=0)")
	}
}

// TestParseOrchestratorStopEvent_MissingTranscript tests error handling
// when the transcript file does not exist.
func TestParseOrchestratorStopEvent_MissingTranscript(t *testing.T) {
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-missing",
		"transcript_path": "/nonexistent/path/transcript.jsonl",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for missing transcript file")
	}

	if !strings.Contains(err.Error(), "[orchestrator-guard]") {
		t.Errorf("Error should have [orchestrator-guard] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}

	// Should still return metadata (graceful degradation)
	if metadata == nil {
		t.Error("Expected metadata to be returned even on error")
	}
}

// TestParseOrchestratorStopEvent_InvalidJSON tests error handling
// when the SubagentStop event JSON is malformed.
func TestParseOrchestratorStopEvent_InvalidJSON(t *testing.T) {
	invalidJSON := `{"hook_event_name": malformed`

	reader := strings.NewReader(invalidJSON)
	_, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "[orchestrator-guard]") {
		t.Errorf("Error should have [orchestrator-guard] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to parse") {
		t.Errorf("Error should mention parsing failure, got: %v", err)
	}
}

// TestParseOrchestratorStopEvent_MissingSessionID tests that validation
// errors from ParseSubagentStopEvent propagate correctly.
func TestParseOrchestratorStopEvent_MissingSessionID(t *testing.T) {
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"transcript_path": "/tmp/test.jsonl"
	}`

	reader := strings.NewReader(eventJSON)
	_, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for missing session_id")
	}

	if !strings.Contains(err.Error(), "[orchestrator-guard]") {
		t.Errorf("Error should have [orchestrator-guard] prefix, got: %v", err)
	}
}

// TestParseOrchestratorStopEvent_ArchitectAgent tests parsing with architect agent.
func TestParseOrchestratorStopEvent_ArchitectAgent(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "architect-transcript.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: architect", "role": "system"}
{"timestamp": 1700001000, "model": "claude-sonnet-4", "role": "assistant"}
{"timestamp": 1700003000, "content": "Plan created", "role": "assistant"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-architect",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "architect" {
		t.Errorf("Expected agent_id 'architect', got: %s", metadata.AgentID)
	}
}

// TestParseOrchestratorStopEvent_ErrorInTranscript tests handling of
// error markers in transcript (exit_code should be 1).
func TestParseOrchestratorStopEvent_ErrorInTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "error-transcript.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: orchestrator"}
{"timestamp": 1700001000, "role": "error", "content": "Task failed"}
{"timestamp": 1700002000, "content": "Cleanup"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-error",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ExitCode != 1 {
		t.Errorf("Expected exit_code 1 for error transcript, got: %d", metadata.ExitCode)
	}

	if metadata.IsSuccess() {
		t.Error("Expected IsSuccess() to be false when error present")
	}
}

// TestParseOrchestratorStopEvent_Timeout tests timeout handling.
func TestParseOrchestratorStopEvent_Timeout(t *testing.T) {
	reader := &slowReader{
		delay: 2 * time.Second,
		data:  `{"hook_event_name":"SubagentStop"}`,
	}

	_, err := ParseOrchestratorStopEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestIsOrchestratorType tests agent type detection with table-driven approach.
func TestIsOrchestratorType(t *testing.T) {
	tests := []struct {
		name     string
		agentID  string
		expected bool
	}{
		{
			name:     "orchestrator agent",
			agentID:  "orchestrator",
			expected: true,
		},
		{
			name:     "architect agent",
			agentID:  "architect",
			expected: true,
		},
		{
			name:     "python-pro agent",
			agentID:  "python-pro",
			expected: false,
		},
		{
			name:     "go-pro agent",
			agentID:  "go-pro",
			expected: false,
		},
		{
			name:     "einstein agent",
			agentID:  "einstein",
			expected: false,
		},
		{
			name:     "codebase-search agent",
			agentID:  "codebase-search",
			expected: false,
		},
		{
			name:     "tech-docs-writer agent",
			agentID:  "tech-docs-writer",
			expected: false,
		},
		{
			name:     "empty agent ID",
			agentID:  "",
			expected: false,
		},
		{
			name:     "unknown agent",
			agentID:  "unknown-agent",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metadata := &ParsedAgentMetadata{
				AgentID: tc.agentID,
			}

			result := IsOrchestratorType(metadata)

			if result != tc.expected {
				t.Errorf("Expected IsOrchestratorType(%q) = %v, got %v",
					tc.agentID, tc.expected, result)
			}
		})
	}
}

// TestIsOrchestratorType_NilMetadata tests handling of nil metadata.
func TestIsOrchestratorType_NilMetadata(t *testing.T) {
	result := IsOrchestratorType(nil)

	if result != false {
		t.Errorf("Expected IsOrchestratorType(nil) = false, got %v", result)
	}
}

// TestIsOrchestratorType_CaseSensitivity tests that agent ID comparison is case-sensitive.
func TestIsOrchestratorType_CaseSensitivity(t *testing.T) {
	tests := []struct {
		agentID  string
		expected bool
	}{
		{"orchestrator", true},
		{"Orchestrator", false},
		{"ORCHESTRATOR", false},
		{"architect", true},
		{"Architect", false},
		{"ARCHITECT", false},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			metadata := &ParsedAgentMetadata{
				AgentID: tc.agentID,
			}

			result := IsOrchestratorType(metadata)

			if result != tc.expected {
				t.Errorf("Expected %q to return %v, got %v",
					tc.agentID, tc.expected, result)
			}
		})
	}
}

// TestParseOrchestratorStopEvent_MultipleAgents tests parsing transcripts
// with multiple agent references (should use first AGENT: line).
func TestParseOrchestratorStopEvent_MultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "multi-agent.jsonl")

	// First AGENT: line should be used
	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: orchestrator", "role": "system"}
{"timestamp": 1700001000, "content": "Delegating to AGENT: python-pro", "role": "assistant"}
{"timestamp": 1700002000, "model": "claude-sonnet-4", "role": "assistant"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-multi",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "orchestrator" {
		t.Errorf("Expected first agent_id 'orchestrator', got: %s", metadata.AgentID)
	}
}

// TestParseOrchestratorStopEvent_EmptyTranscript tests handling of empty transcript files.
func TestParseOrchestratorStopEvent_EmptyTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty.jsonl")

	if err := os.WriteFile(transcriptPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty transcript: %v", err)
	}

	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-empty",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error for empty transcript, got: %v", err)
	}

	// Should return default metadata
	if metadata.AgentID != "" {
		t.Errorf("Expected empty agent_id, got: %s", metadata.AgentID)
	}

	if metadata.ExitCode != 0 {
		t.Errorf("Expected default exit_code 0, got: %d", metadata.ExitCode)
	}
}
