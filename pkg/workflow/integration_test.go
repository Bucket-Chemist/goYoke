package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// createMockTranscript creates a realistic transcript file matching the format
// expected by ParseTranscriptForMetadata (pkg/routing/events.go:248-315).
//
// Actual format:
// - {"content": "AGENT: <agent-id>", "timestamp": <unix-epoch-float>}  // Agent identification
// - {"model": "<model-name>", "timestamp": <unix-epoch-float>}        // Model information
// - {"role": "error", "timestamp": <unix-epoch-float>}                // Optional: indicates failure
//
// This helper creates transcripts that will be properly parsed by the workflow.
func createMockTranscript(t *testing.T, path, agentID, model string, exitCode int) {
	t.Helper()

	baseTime := float64(time.Now().Unix())

	// Create transcript with actual format used by ParseTranscriptForMetadata
	var lines []string

	// Agent identification line
	agentEntry := map[string]interface{}{
		"content":   "AGENT: " + agentID,
		"timestamp": baseTime,
	}
	data, _ := json.Marshal(agentEntry)
	lines = append(lines, string(data))

	// Model information line
	modelEntry := map[string]interface{}{
		"model":     model,
		"timestamp": baseTime + 1.0,
	}
	data, _ = json.Marshal(modelEntry)
	lines = append(lines, string(data))

	// If failure, add error role entry
	if exitCode != 0 {
		errorEntry := map[string]interface{}{
			"role":      "error",
			"timestamp": baseTime + 3.0,
		}
		data, _ = json.Marshal(errorEntry)
		lines = append(lines, string(data))
	}

	// Completion timestamp for duration calculation
	completionEntry := map[string]interface{}{
		"timestamp": baseTime + 5.0,
	}
	data, _ = json.Marshal(completionEntry)
	lines = append(lines, string(data))

	// Write to file as JSONL
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create mock transcript: %v", err)
	}
}

func TestAgentEndstateWorkflow_OrchestratorSuccess(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Create mock transcript with agent metadata (ACTUAL format)
	createMockTranscript(t, transcriptPath, "orchestrator", "sonnet", 0)

	// Simulate full workflow: event → transcript parsing → response → log
	// Uses ACTUAL SubagentStop schema
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-orchestrator",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	event, err := routing.ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse transcript for agent metadata
	metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Failed to parse transcript: %v", err)
	}

	// Verify metadata extraction worked
	if metadata.AgentID != "orchestrator" {
		t.Errorf("Expected agentID=orchestrator, got: %s", metadata.AgentID)
	}

	if metadata.Tier != "sonnet" {
		t.Errorf("Expected tier=sonnet, got: %s", metadata.Tier)
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Errorf("Expected prompt, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "ORCHESTRATOR COMPLETED") {
		t.Error("Should indicate orchestrator completion")
	}

	if !strings.Contains(response.AdditionalContext, "background tasks") {
		t.Error("Should prompt for background task verification")
	}

	// Verify JSON formatting
	var buf strings.Builder
	if err := response.Marshal(&buf); err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	if !strings.Contains(buf.String(), "hookSpecificOutput") {
		t.Error("JSON should contain hookSpecificOutput")
	}
}

func TestAgentEndstateWorkflow_ImplementationFailure(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript-failure.jsonl")

	// Create mock transcript with failure metadata (exitCode=1)
	createMockTranscript(t, transcriptPath, "python-pro", "sonnet", 1)

	// Uses ACTUAL SubagentStop schema
	eventJSON := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-failure",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(eventJSON)
	event, err := routing.ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse transcript for agent metadata
	metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Failed to parse transcript: %v", err)
	}

	if metadata.ExitCode == 0 {
		t.Error("Should detect failure from transcript (role: error)")
	}

	response := GenerateEndstateResponse(event, metadata)

	if response.Decision != "prompt" {
		t.Error("Should prompt on failure")
	}

	if !strings.Contains(response.AdditionalContext, "AGENT FAILED") {
		t.Error("Should indicate failure")
	}

	if !strings.Contains(response.AdditionalContext, "Exit Code: 1") {
		t.Error("Should show exit code")
	}
}

func TestAgentEndstateWorkflow_AllAgentClasses(t *testing.T) {
	tests := []struct {
		agentID          string
		model            string
		expectedDecision string
		shouldPrompt     bool
	}{
		{"orchestrator", "sonnet", "prompt", true},
		{"architect", "sonnet", "prompt", true},
		{"python-pro", "sonnet", "prompt", true},
		{"code-reviewer", "haiku", "prompt", true},
		{"haiku-scout", "haiku", "silent", false},
		{"codebase-search", "haiku", "silent", false},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			// Use t.TempDir() for each test case
			tmpDir := t.TempDir()
			transcriptPath := filepath.Join(tmpDir, tc.agentID+"-transcript.jsonl")

			// Create mock transcript for this agent
			createMockTranscript(t, transcriptPath, tc.agentID, tc.model, 0)

			// Uses ACTUAL SubagentStop schema
			event := &routing.SubagentStopEvent{
				HookEventName:  "SubagentStop",
				SessionID:      "test-session-" + tc.agentID,
				TranscriptPath: transcriptPath,
				StopHookActive: true,
			}

			// Parse transcript for metadata
			metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
			if err != nil {
				t.Fatalf("Agent %s: failed to parse transcript: %v", tc.agentID, err)
			}

			// Verify metadata extraction
			if metadata.AgentID != tc.agentID {
				t.Errorf("Agent %s: expected agentID=%s, got %s",
					tc.agentID, tc.agentID, metadata.AgentID)
			}

			response := GenerateEndstateResponse(event, metadata)

			if response.Decision != tc.expectedDecision {
				t.Errorf("Agent %s: expected decision %s, got %s",
					tc.agentID, tc.expectedDecision, response.Decision)
			}
		})
	}
}

func TestAgentEndstateWorkflow_SimulationHarness(t *testing.T) {
	// Simulation harness integration test example
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "sim-transcript.jsonl")

	// Simulate complete agent lifecycle
	createMockTranscript(t, transcriptPath, "python-pro", "sonnet", 0)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "sim-session-001",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Fatalf("Simulation: failed to parse transcript: %v", err)
	}

	response := GenerateEndstateResponse(event, metadata)

	// Verify complete workflow produces valid output
	if response.Decision == "" {
		t.Error("Simulation: response should have decision")
	}

	// Verify logging integration (uses XDG path, so set temp env)
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	if err := LogEndstate(event, metadata, response); err != nil {
		t.Errorf("Simulation: logging failed: %v", err)
	}

	// Verify log was written
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("Simulation: failed to read logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Simulation: expected 1 log entry, got %d", len(logs))
	}

	if len(logs) > 0 && logs[0].AgentID != "python-pro" {
		t.Errorf("Simulation: expected python-pro log, got %s", logs[0].AgentID)
	}
}

func TestAgentEndstateWorkflow_TranscriptParsingFailure(t *testing.T) {
	// Test graceful degradation when transcript parsing fails
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "missing-transcript.jsonl")

	// Don't create the file - simulate missing transcript

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-session-missing",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// ParseTranscriptForMetadata should return partial metadata with error
	metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if err == nil {
		t.Error("Expected error for missing transcript")
	}

	// Should still get partial metadata (graceful degradation)
	if metadata == nil {
		t.Fatal("Metadata should not be nil even on error")
	}

	// Response generation should handle nil/partial metadata gracefully
	response := GenerateEndstateResponse(event, metadata)

	if response == nil {
		t.Fatal("Response should not be nil even with missing transcript")
	}

	// Graceful degradation: should produce silent response for unknown agent
	if response.Decision != "silent" {
		t.Errorf("Expected silent decision for unknown agent, got: %s", response.Decision)
	}
}

func TestAgentEndstateWorkflow_MalformedTranscript(t *testing.T) {
	// Test graceful degradation with malformed JSONL
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed-transcript.jsonl")

	// Create file with malformed JSON
	content := `{"invalid json without closing
this is not json at all
{"content": "AGENT: orchestrator", "timestamp": 1234567890.0}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-session-malformed",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// Should not crash - graceful degradation
	metadata, err := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		t.Logf("Expected error for malformed transcript: %v", err)
	}

	// Should still extract what it can from the valid line
	if metadata.AgentID != "orchestrator" {
		t.Errorf("Should have extracted agentID from valid line, got: %s", metadata.AgentID)
	}

	// Response generation should work
	response := GenerateEndstateResponse(event, metadata)
	if response == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestAgentEndstateWorkflow_DurationCalculation(t *testing.T) {
	// Verify duration is calculated correctly from transcript timestamps
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "duration-transcript.jsonl")

	baseTime := float64(1234567890)
	lines := []string{
		`{"content": "AGENT: orchestrator", "timestamp": ` + formatFloat(baseTime) + `}`,
		`{"model": "sonnet", "timestamp": ` + formatFloat(baseTime+1.0) + `}`,
		`{"timestamp": ` + formatFloat(baseTime+10.5) + `}`, // 10.5 seconds later
	}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	metadata, err := routing.ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to parse transcript: %v", err)
	}

	// Duration should be lastTimestamp - firstTimestamp = 10.5 seconds = 10 ms (int truncation)
	expectedDuration := 10 // int(10.5) = 10
	if metadata.DurationMs != expectedDuration {
		t.Errorf("Expected duration %dms, got %dms", expectedDuration, metadata.DurationMs)
	}
}

func TestAgentEndstateWorkflow_ModelTierMapping(t *testing.T) {
	// Test that model names are correctly mapped to tiers
	tests := []struct {
		model        string
		expectedTier string
	}{
		{"claude-haiku-4", "haiku"},
		{"claude-sonnet-4", "sonnet"},
		{"claude-opus-4", "opus"},
		{"haiku", "haiku"},
		{"sonnet", "sonnet"},
		{"opus", "opus"},
		{"unknown-model", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			tmpDir := t.TempDir()
			transcriptPath := filepath.Join(tmpDir, "model-transcript.jsonl")

			// Create transcript with specific model
			lines := []string{
				`{"content": "AGENT: test-agent", "timestamp": 1234567890.0}`,
				`{"model": "` + tc.model + `", "timestamp": 1234567891.0}`,
			}
			if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
				t.Fatal(err)
			}

			metadata, err := routing.ParseTranscriptForMetadata(transcriptPath)
			if err != nil {
				t.Fatalf("Failed to parse transcript: %v", err)
			}

			if metadata.Tier != tc.expectedTier {
				t.Errorf("Model %s: expected tier %s, got %s",
					tc.model, tc.expectedTier, metadata.Tier)
			}
		})
	}
}

// formatFloat formats float64 for JSON without scientific notation
func formatFloat(f float64) string {
	// Format with one decimal place, then trim trailing .0
	s := fmt.Sprintf("%.1f", f)
	if strings.HasSuffix(s, ".0") {
		return s[:len(s)-2]
	}
	return s
}
