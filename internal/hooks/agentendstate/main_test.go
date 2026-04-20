package agentendstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
	"github.com/Bucket-Chemist/goYoke/pkg/workflow"
)

// =============================================================================
// Collaboration Logging Tests (goYoke-088c)
// =============================================================================

func TestCollaborationLogging(t *testing.T) {
	// Setup temp directory for XDG_DATA_HOME
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create test collaboration
	collab := telemetry.NewAgentCollaboration(
		"sess-123",
		"terminal",
		"codebase-search",
		"spawn",
	)
	collab.ChildSuccess = true
	collab.ChildDurationMs = 1500
	collab.ChainDepth = 1

	// Log collaboration
	err := telemetry.LogCollaboration(collab)
	if err != nil {
		t.Fatalf("Failed to log collaboration: %v", err)
	}

	// Verify file created
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Collaboration log file should exist")
	}

	// Read and verify content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logged telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if logged.ParentAgent != "terminal" {
		t.Errorf("ParentAgent = %v, want terminal", logged.ParentAgent)
	}
	if logged.ChildAgent != "codebase-search" {
		t.Errorf("ChildAgent = %v, want codebase-search", logged.ChildAgent)
	}
	if logged.DelegationType != "spawn" {
		t.Errorf("DelegationType = %v, want spawn", logged.DelegationType)
	}
	if !logged.ChildSuccess {
		t.Error("ChildSuccess should be true")
	}
	if logged.ChildDurationMs != 1500 {
		t.Errorf("ChildDurationMs = %v, want 1500", logged.ChildDurationMs)
	}
	if logged.ChainDepth != 1 {
		t.Errorf("ChainDepth = %v, want 1", logged.ChainDepth)
	}
}

func TestCollaboration_MetadataExtraction(t *testing.T) {
	// Test that metadata is correctly extracted from transcript
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "python-pro",
		DurationMs: 2500,
		ExitCode:   0, // Success
	}

	collab := telemetry.NewAgentCollaboration(
		"sess-123",
		"terminal",
		metadata.AgentID,
		"spawn",
	)
	collab.ChildSuccess = metadata.IsSuccess()
	collab.ChildDurationMs = int64(metadata.DurationMs)

	if collab.ChildAgent != "python-pro" {
		t.Errorf("ChildAgent = %v, want python-pro", collab.ChildAgent)
	}
	if !collab.ChildSuccess {
		t.Error("ChildSuccess should be true when ExitCode is 0")
	}
	if collab.ChildDurationMs != 2500 {
		t.Errorf("ChildDurationMs = %v, want 2500", collab.ChildDurationMs)
	}
}

func TestCollaborationLogging_NonBlocking(t *testing.T) {
	// Verify collaboration logging errors don't fail the hook
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", "/nonexistent/readonly/path")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	collab := telemetry.NewAgentCollaboration(
		"sess-123",
		"terminal",
		"agent",
		"spawn",
	)

	// Should return error but not panic - hook continues
	err := telemetry.LogCollaboration(collab)
	if err == nil {
		t.Log("Warning: Expected error with invalid path, got nil (permissions may allow creation)")
	}
	// The important thing is we don't panic
}

func TestLogCollaboration_Integration(t *testing.T) {
	// Integration test: Full flow through logCollaboration helper
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create test event and metadata
	event := &routing.SubagentStopEvent{
		SessionID:      "test-session-456",
		TranscriptPath: "/tmp/test-transcript.jsonl",
		HookEventName:  "SubagentStop",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		DurationMs: 3200,
		ExitCode:   0,
	}

	// Call the logCollaboration helper
	err := logCollaboration(event, metadata)
	if err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Verify file created
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("Collaboration log file should exist after logCollaboration()")
	}

	// Read and verify content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logged telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify collaboration fields
	if logged.SessionID != "test-session-456" {
		t.Errorf("SessionID = %v, want test-session-456", logged.SessionID)
	}
	if logged.ParentAgent != "terminal" {
		t.Errorf("ParentAgent = %v, want terminal", logged.ParentAgent)
	}
	if logged.ChildAgent != "orchestrator" {
		t.Errorf("ChildAgent = %v, want orchestrator", logged.ChildAgent)
	}
	if logged.DelegationType != "spawn" {
		t.Errorf("DelegationType = %v, want spawn", logged.DelegationType)
	}
	if !logged.ChildSuccess {
		t.Error("ChildSuccess should be true when ExitCode is 0")
	}
	if logged.ChildDurationMs != 3200 {
		t.Errorf("ChildDurationMs = %v, want 3200", logged.ChildDurationMs)
	}
	if logged.ChainDepth != 1 {
		t.Errorf("ChainDepth = %v, want 1 (root-level delegation)", logged.ChainDepth)
	}
}

func TestLogCollaboration_FailureCase(t *testing.T) {
	// Test collaboration logging when agent fails (ExitCode != 0)
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "test-session-failure",
		TranscriptPath: "/tmp/test-transcript.jsonl",
		HookEventName:  "SubagentStop",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "python-pro",
		DurationMs: 1800,
		ExitCode:   1, // Failure
	}

	err := logCollaboration(event, metadata)
	if err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Read and verify ChildSuccess is false
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logged telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if logged.ChildSuccess {
		t.Error("ChildSuccess should be false when ExitCode is 1")
	}
}

func TestLogCollaboration_MultipleAgents(t *testing.T) {
	// Test multiple collaborations are logged correctly
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	agents := []string{"codebase-search", "python-pro", "orchestrator", "architect"}
	for i, agentID := range agents {
		event := &routing.SubagentStopEvent{
			SessionID:      "multi-agent-test",
			TranscriptPath: "/tmp/test.jsonl",
			HookEventName:  "SubagentStop",
		}

		metadata := &routing.ParsedAgentMetadata{
			AgentID:    agentID,
			DurationMs: 1000 + i*100,
			ExitCode:   0,
		}

		if err := logCollaboration(event, metadata); err != nil {
			t.Fatalf("logCollaboration() failed for %s: %v", agentID, err)
		}
	}

	// Read log file
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Split into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(agents) {
		t.Errorf("Expected %d lines, got %d", len(agents), len(lines))
	}

	// Verify each agent logged
	for i, line := range lines {
		var collab telemetry.AgentCollaboration
		if err := json.Unmarshal([]byte(line), &collab); err != nil {
			t.Errorf("Line %d invalid JSON: %v", i, err)
			continue
		}

		if collab.ChildAgent != agents[i] {
			t.Errorf("Line %d: ChildAgent = %v, want %v", i, collab.ChildAgent, agents[i])
		}
		if collab.ParentAgent != "terminal" {
			t.Errorf("Line %d: ParentAgent should always be terminal, got %v", i, collab.ParentAgent)
		}
		if collab.ChainDepth != 1 {
			t.Errorf("Line %d: ChainDepth should always be 1, got %v", i, collab.ChainDepth)
		}
	}
}

func TestLogCollaboration_ConcurrentWrites(t *testing.T) {
	// Verify collaboration logging is safe for concurrent use
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Launch 10 concurrent writes
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := &routing.SubagentStopEvent{
				SessionID:      "concurrent-test",
				TranscriptPath: "/tmp/test.jsonl",
				HookEventName:  "SubagentStop",
			}

			metadata := &routing.ParsedAgentMetadata{
				AgentID:    "test-agent",
				DurationMs: id * 100,
				ExitCode:   0,
			}

			done <- logCollaboration(event, metadata)
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent write %d failed: %v", i, err)
		}
	}

	// Verify all 10 collaborations were written
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 JSONL lines, got %d", len(lines))
	}
}

func TestLogCollaboration_PerformanceRegression(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "perf-test",
		TranscriptPath: "/tmp/test.jsonl",
		HookEventName:  "SubagentStop",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "python-pro",
		DurationMs: 1500,
		ExitCode:   0,
	}

	// Measure execution time
	start := time.Now()
	err := logCollaboration(event, metadata)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Verify latency < 10ms (non-blocking requirement)
	if duration > 10*time.Millisecond {
		t.Errorf("Collaboration logging took %v, exceeds 10ms threshold", duration)
	}
}

func TestLogCollaboration_TerminalAsParent(t *testing.T) {
	// Verify that parent is always "terminal" in SubagentStop context
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "terminal-parent-test",
		TranscriptPath: "/tmp/test.jsonl",
		HookEventName:  "SubagentStop",
	}

	// Test different child agents
	childAgents := []string{"orchestrator", "architect", "python-pro", "codebase-search"}
	for _, childAgent := range childAgents {
		metadata := &routing.ParsedAgentMetadata{
			AgentID:    childAgent,
			DurationMs: 1000,
			ExitCode:   0,
		}

		if err := logCollaboration(event, metadata); err != nil {
			t.Fatalf("logCollaboration() failed for %s: %v", childAgent, err)
		}
	}

	// Read all logged collaborations
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify all have ParentAgent = "terminal"
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i, line := range lines {
		var collab telemetry.AgentCollaboration
		if err := json.Unmarshal([]byte(line), &collab); err != nil {
			t.Errorf("Line %d invalid JSON: %v", i, err)
			continue
		}

		if collab.ParentAgent != "terminal" {
			t.Errorf("Line %d: ParentAgent = %v, want terminal (SubagentStop always has terminal as parent)", i, collab.ParentAgent)
		}
	}
}

func TestLogCollaboration_ChainDepthAlwaysOne(t *testing.T) {
	// Verify that ChainDepth is always 1 for root-level delegations
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "chain-depth-test",
		TranscriptPath: "/tmp/test.jsonl",
		HookEventName:  "SubagentStop",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		DurationMs: 2000,
		ExitCode:   0,
	}

	if err := logCollaboration(event, metadata); err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Read and verify ChainDepth
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var collab telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &collab); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if collab.ChainDepth != 1 {
		t.Errorf("ChainDepth = %v, want 1 (SubagentStop represents root-level delegation)", collab.ChainDepth)
	}
}

func TestLogCollaboration_DelegationTypeAlwaysSpawn(t *testing.T) {
	// Verify that DelegationType is always "spawn" in SubagentStop context
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "delegation-type-test",
		TranscriptPath: "/tmp/test.jsonl",
		HookEventName:  "SubagentStop",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "python-pro",
		DurationMs: 1500,
		ExitCode:   0,
	}

	if err := logCollaboration(event, metadata); err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Read and verify DelegationType
	logPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var collab telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &collab); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if collab.DelegationType != "spawn" {
		t.Errorf("DelegationType = %v, want spawn (SubagentStop represents spawn delegation)", collab.DelegationType)
	}
}

// =============================================================================
// Main Function Integration Tests
// =============================================================================

func TestOutputError(t *testing.T) {
	// Test outputError produces valid JSON
	// Cannot directly test stdout, but can verify it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputError() panicked: %v", r)
		}
	}()

	outputError("test error message")
}

func TestMain_EndToEnd(t *testing.T) {
	// End-to-end test: Create transcript, simulate SubagentStop event, verify outputs
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create a test transcript file with agent metadata
	transcriptPath := filepath.Join(tmpDir, "test-transcript.jsonl")
	transcriptContent := `{"content":"AGENT: python-pro","role":"user","timestamp":1705708800}
{"content":"Executing task","role":"assistant","timestamp":1705708805}
{"model":"sonnet","timestamp":1705708806}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	// Create SubagentStop event
	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "e2e-test-session",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// Parse metadata from transcript
	metadata, err := routing.ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("ParseTranscriptForMetadata() failed: %v", err)
	}

	// Verify metadata parsed correctly
	if metadata.AgentID != "python-pro" {
		t.Errorf("AgentID = %v, want python-pro", metadata.AgentID)
	}
	if metadata.AgentModel != "sonnet" {
		t.Errorf("AgentModel = %v, want sonnet", metadata.AgentModel)
	}

	// Generate endstate response
	response := workflow.GenerateEndstateResponse(event, metadata)
	if response == nil {
		t.Fatal("GenerateEndstateResponse() returned nil")
	}

	// Log endstate
	if err := workflow.LogEndstate(event, metadata, response); err != nil {
		t.Logf("LogEndstate warning: %v (may fail if .claude dir doesn't exist)", err)
	}

	// Log collaboration (the function under test)
	if err := logCollaboration(event, metadata); err != nil {
		t.Fatalf("logCollaboration() failed: %v", err)
	}

	// Verify collaboration was logged
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(collabPath); os.IsNotExist(err) {
		t.Fatal("Collaboration log should exist after end-to-end flow")
	}

	// Read and verify collaboration content
	data, err := os.ReadFile(collabPath)
	if err != nil {
		t.Fatalf("Failed to read collaboration log: %v", err)
	}

	var collab telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &collab); err != nil {
		t.Fatalf("Failed to unmarshal collaboration: %v", err)
	}

	// Verify collaboration fields match event/metadata
	if collab.SessionID != "e2e-test-session" {
		t.Errorf("SessionID = %v, want e2e-test-session", collab.SessionID)
	}
	if collab.ParentAgent != "terminal" {
		t.Errorf("ParentAgent = %v, want terminal", collab.ParentAgent)
	}
	if collab.ChildAgent != "python-pro" {
		t.Errorf("ChildAgent = %v, want python-pro", collab.ChildAgent)
	}
	if collab.DelegationType != "spawn" {
		t.Errorf("DelegationType = %v, want spawn", collab.DelegationType)
	}
	if !collab.ChildSuccess {
		t.Error("ChildSuccess should be true (ExitCode = 0)")
	}
	if collab.ChainDepth != 1 {
		t.Errorf("ChainDepth = %v, want 1", collab.ChainDepth)
	}
}

func TestProcessEvent_Success(t *testing.T) {
	// Test processEvent with valid transcript
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create transcript
	transcriptPath := filepath.Join(tmpDir, "test.jsonl")
	transcriptContent := `{"content":"AGENT: python-pro","timestamp":1000}
{"model":"sonnet","timestamp":1005}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	event := &routing.SubagentStopEvent{
		SessionID:      "process-event-test",
		TranscriptPath: transcriptPath,
		HookEventName:  "SubagentStop",
	}

	// Call processEvent
	response, err := processEvent(event)
	if err != nil {
		t.Fatalf("processEvent() failed: %v", err)
	}

	if response == nil {
		t.Fatal("processEvent() returned nil response")
	}

	// Verify collaboration was logged
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(collabPath); os.IsNotExist(err) {
		t.Error("Collaboration should be logged by processEvent()")
	}
}

func TestProcessEvent_MissingTranscript(t *testing.T) {
	// Test processEvent with missing transcript (should not fail)
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		SessionID:      "missing-transcript-test",
		TranscriptPath: "/nonexistent/file.jsonl",
		HookEventName:  "SubagentStop",
	}

	// Should not fail even with missing transcript
	response, err := processEvent(event)
	if err != nil {
		t.Fatalf("processEvent() should handle missing transcript gracefully: %v", err)
	}

	if response == nil {
		t.Fatal("processEvent() returned nil response")
	}

	// Verify collaboration was still logged (with empty metadata)
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(collabPath); os.IsNotExist(err) {
		t.Error("Collaboration should be logged even with missing transcript")
	}
}

func TestProcessEvent_CollaborationLogging(t *testing.T) {
	// Verify processEvent calls logCollaboration correctly
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create transcript with agent metadata
	transcriptPath := filepath.Join(tmpDir, "collab-test.jsonl")
	transcriptContent := `{"content":"AGENT: orchestrator","timestamp":2000}
{"model":"sonnet","timestamp":2100}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("Failed to create transcript: %v", err)
	}

	event := &routing.SubagentStopEvent{
		SessionID:      "collab-test-session",
		TranscriptPath: transcriptPath,
		HookEventName:  "SubagentStop",
	}

	// Process event
	_, err := processEvent(event)
	if err != nil {
		t.Fatalf("processEvent() failed: %v", err)
	}

	// Verify collaboration content
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(collabPath)
	if err != nil {
		t.Fatalf("Failed to read collaboration: %v", err)
	}

	var collab telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &collab); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify processEvent populated collaboration correctly
	if collab.SessionID != "collab-test-session" {
		t.Errorf("SessionID = %v, want collab-test-session", collab.SessionID)
	}
	if collab.ChildAgent != "orchestrator" {
		t.Errorf("ChildAgent = %v, want orchestrator", collab.ChildAgent)
	}
	if collab.ParentAgent != "terminal" {
		t.Errorf("ParentAgent = %v, want terminal", collab.ParentAgent)
	}
}

func TestProcessEvent_NilMetadata(t *testing.T) {
	// Test processEvent's nil metadata handling path
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create event with nonexistent transcript (triggers nil metadata check)
	event := &routing.SubagentStopEvent{
		SessionID:      "nil-metadata-test",
		TranscriptPath: "/does/not/exist.jsonl",
		HookEventName:  "SubagentStop",
	}

	// Call processEvent (should handle gracefully)
	response, err := processEvent(event)
	if err != nil {
		t.Fatalf("processEvent() should handle nil metadata: %v", err)
	}
	if response == nil {
		t.Fatal("processEvent() returned nil response")
	}

	// Verify collaboration was still logged
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(collabPath); os.IsNotExist(err) {
		t.Error("Collaboration should be logged even with nil metadata")
	}
}

func TestMain_EmptyMetadataHandling(t *testing.T) {
	// Test that main() handles empty metadata gracefully
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "empty-metadata-test",
		TranscriptPath: "/nonexistent/transcript.jsonl",
		StopHookActive: true,
	}

	// ParseTranscriptForMetadata returns empty struct for nonexistent file
	metadata, _ := routing.ParseTranscriptForMetadata(event.TranscriptPath)

	// Verify metadata has empty AgentID
	if metadata.AgentID != "" {
		t.Errorf("Expected empty AgentID for nonexistent file, got %v", metadata.AgentID)
	}

	// Generate response (should handle empty metadata)
	response := workflow.GenerateEndstateResponse(event, metadata)
	if response == nil {
		t.Fatal("GenerateEndstateResponse() should handle empty metadata gracefully")
	}

	// Verify logCollaboration works with empty metadata
	if err := logCollaboration(event, metadata); err != nil {
		t.Fatalf("logCollaboration() failed with empty metadata: %v", err)
	}

	// Verify collaboration logged with empty agent string (as-is behavior)
	collabPath := filepath.Join(tmpDir, "goyoke", "agent-collaborations.jsonl")
	data, err := os.ReadFile(collabPath)
	if err != nil {
		t.Fatalf("Failed to read collaboration log: %v", err)
	}

	var collab telemetry.AgentCollaboration
	if err := json.Unmarshal(data, &collab); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Empty AgentID is passed through as-is (main() doesn't modify it)
	if collab.ChildAgent != "" {
		t.Errorf("ChildAgent = %v, want empty string (metadata.AgentID was empty)", collab.ChildAgent)
	}
}
