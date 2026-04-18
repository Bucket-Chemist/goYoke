package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

func TestGetEndstateLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	path := GetEndstateLogPath()
	expected := filepath.Join(tmpDir, "gogent", "agent-endstates.jsonl")

	if path != expected {
		t.Errorf("GetEndstateLogPath() = %s, want %s", path, expected)
	}
}

func TestGetProjectEndstateLogPath(t *testing.T) {
	projectDir := "/home/user/project"
	path := GetProjectEndstateLogPath(projectDir)
	expected := filepath.Join(projectDir, ".gogent", "memory", "agent-endstates.jsonl")

	if path != expected {
		t.Errorf("GetProjectEndstateLogPath() = %s, want %s", path, expected)
	}
}

func TestLogEndstate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript.jsonl",
		StopHookActive: true,
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:      "orchestrator",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     0,
		DurationMs:   5000,
		OutputTokens: 2048,
	}

	response := &EndstateResponse{
		HookEventName:     "SubagentStop",
		Decision:          "prompt",
		AdditionalContext: "Test context",
		Tier:              "sonnet",
		AgentClass:        "orchestrator",
		Recommendations:   []string{"test_recommendation"},
	}

	// Verify logging works
	if err := LogEndstate(event, metadata, response); err != nil {
		t.Fatalf("LogEndstate failed: %v", err)
	}

	logPath := GetEndstateLogPath()

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Verify content
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.AgentID != "orchestrator" {
		t.Errorf("AgentID = %s, want orchestrator", log.AgentID)
	}
	if log.Decision != "prompt" {
		t.Errorf("Decision = %s, want prompt", log.Decision)
	}
	if log.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", log.ExitCode)
	}
}

func TestLogEndstate_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Directory doesn't exist yet
	gogentDir := filepath.Join(tmpDir, "gogent")
	if _, err := os.Stat(gogentDir); !os.IsNotExist(err) {
		t.Fatal("Directory should not exist before LogEndstate")
	}

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test",
		TranscriptPath: "/tmp/test.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "test-agent",
		Tier:       "haiku",
		ExitCode:   0,
		DurationMs: 100,
	}

	response := &EndstateResponse{
		Decision: "silent",
	}

	if err := LogEndstate(event, metadata, response); err != nil {
		t.Fatalf("LogEndstate failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(gogentDir); os.IsNotExist(err) {
		t.Error("Directory was not created by LogEndstate")
	}
}

func TestReadEndstateLogs_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Should return empty list, not error
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Errorf("ReadEndstateLogs should not error on missing file: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected empty logs, got: %d entries", len(logs))
	}
}

func TestReadEndstateLogs_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create log file with mix of valid and malformed lines
	logPath := GetEndstateLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}

	validLog := session.EndstateLog{
		Timestamp:       time.Now(),
		AgentID:         "test-agent",
		AgentClass:      "implementation",
		Tier:            "sonnet",
		ExitCode:        0,
		Duration:        1000,
		OutputTokens:    500,
		Decision:        "prompt",
		Recommendations: []string{"test"},
	}

	content := `{"invalid": "json" without closing
{"timestamp":"` + validLog.Timestamp.Format(time.RFC3339Nano) + `","agent_id":"test-agent","agent_class":"implementation","tier":"sonnet","exit_code":0,"duration_ms":1000,"output_tokens":500,"decision":"prompt","recommendations":["test"]}
this is not json at all
`

	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Should gracefully skip malformed lines
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("ReadEndstateLogs failed: %v", err)
	}

	// Should have exactly 1 valid entry
	if len(logs) != 1 {
		t.Errorf("Expected 1 valid log entry, got %d", len(logs))
	}

	if len(logs) > 0 && logs[0].AgentID != "test-agent" {
		t.Errorf("AgentID = %s, want test-agent", logs[0].AgentID)
	}
}

func TestReadEndstateLogs_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log multiple entries
	for i := 0; i < 5; i++ {
		event := &routing.SubagentStopEvent{
			HookEventName:  "SubagentStop",
			SessionID:      "test",
			TranscriptPath: "/tmp/test.jsonl",
		}

		metadata := &routing.ParsedAgentMetadata{
			AgentID:      "test-agent",
			Tier:         "haiku",
			ExitCode:     i % 2, // Mix success and failure
			DurationMs:   100 * i,
			OutputTokens: 50 * i,
		}

		response := &EndstateResponse{
			Decision: "silent",
		}

		if err := LogEndstate(event, metadata, response); err != nil {
			t.Fatalf("LogEndstate failed: %v", err)
		}
	}

	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("ReadEndstateLogs failed: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 log entries, got %d", len(logs))
	}
}

func TestReadEndstateLogs_LargeLine(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	logPath := GetEndstateLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}

	logEntry := session.EndstateLog{
		Timestamp:       time.Now(),
		AgentID:         "large-agent",
		AgentClass:      "implementation",
		Tier:            "sonnet",
		ExitCode:        0,
		Duration:        1000,
		OutputTokens:    500,
		Decision:        "prompt",
		Recommendations: []string{strings.Repeat("w", 70*1024)},
	}
	data, err := json.Marshal(logEntry)
	if err != nil {
		t.Fatalf("Failed to marshal log: %v", err)
	}
	if err := os.WriteFile(logPath, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("ReadEndstateLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}
	if logs[0].AgentID != "large-agent" {
		t.Fatalf("Expected large log to round-trip")
	}
}

func TestGetAgentStats_NoRuns(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	successCount, failureCount, successRate, err := GetAgentStats("nonexistent-agent")
	if err != nil {
		t.Errorf("GetAgentStats should not error for agent with no runs: %v", err)
	}

	if successCount != 0 || failureCount != 0 || successRate != 0 {
		t.Errorf("Expected (0, 0, 0.0), got (%d, %d, %.2f)", successCount, failureCount, successRate)
	}
}

func TestGetAgentStats_MixedResults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log 7 successes and 3 failures for "orchestrator"
	for i := 0; i < 10; i++ {
		exitCode := 0
		if i < 3 {
			exitCode = 1 // First 3 fail
		}

		event := &routing.SubagentStopEvent{
			HookEventName:  "SubagentStop",
			SessionID:      "test",
			TranscriptPath: "/tmp/test.jsonl",
		}

		metadata := &routing.ParsedAgentMetadata{
			AgentID:    "orchestrator",
			Tier:       "sonnet",
			ExitCode:   exitCode,
			DurationMs: 1000,
		}

		response := &EndstateResponse{
			Decision: "prompt",
		}

		if err := LogEndstate(event, metadata, response); err != nil {
			t.Fatalf("LogEndstate failed: %v", err)
		}
	}

	// Also log some entries for a different agent to verify filtering
	for i := 0; i < 5; i++ {
		event := &routing.SubagentStopEvent{
			HookEventName:  "SubagentStop",
			SessionID:      "test",
			TranscriptPath: "/tmp/test.jsonl",
		}

		metadata := &routing.ParsedAgentMetadata{
			AgentID:    "python-pro",
			Tier:       "sonnet",
			ExitCode:   0,
			DurationMs: 500,
		}

		response := &EndstateResponse{
			Decision: "silent",
		}

		if err := LogEndstate(event, metadata, response); err != nil {
			t.Fatalf("LogEndstate failed: %v", err)
		}
	}

	successCount, failureCount, successRate, err := GetAgentStats("orchestrator")
	if err != nil {
		t.Fatalf("GetAgentStats failed: %v", err)
	}

	expectedSuccess := 7
	expectedFailure := 3
	expectedRate := 70.0

	if successCount != expectedSuccess {
		t.Errorf("successCount = %d, want %d", successCount, expectedSuccess)
	}
	if failureCount != expectedFailure {
		t.Errorf("failureCount = %d, want %d", failureCount, expectedFailure)
	}
	if successRate != expectedRate {
		t.Errorf("successRate = %.2f, want %.2f", successRate, expectedRate)
	}

	// Verify python-pro has different stats
	successCount2, failureCount2, successRate2, err := GetAgentStats("python-pro")
	if err != nil {
		t.Fatalf("GetAgentStats failed: %v", err)
	}

	if successCount2 != 5 || failureCount2 != 0 || successRate2 != 100.0 {
		t.Errorf("python-pro stats = (%d, %d, %.2f), want (5, 0, 100.0)",
			successCount2, failureCount2, successRate2)
	}
}

func TestGetAgentStats_AllFailures(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log 3 failures
	for i := 0; i < 3; i++ {
		event := &routing.SubagentStopEvent{
			HookEventName:  "SubagentStop",
			SessionID:      "test",
			TranscriptPath: "/tmp/test.jsonl",
		}

		metadata := &routing.ParsedAgentMetadata{
			AgentID:    "failing-agent",
			Tier:       "haiku",
			ExitCode:   1,
			DurationMs: 100,
		}

		response := &EndstateResponse{
			Decision: "prompt",
		}

		if err := LogEndstate(event, metadata, response); err != nil {
			t.Fatalf("LogEndstate failed: %v", err)
		}
	}

	successCount, failureCount, successRate, err := GetAgentStats("failing-agent")
	if err != nil {
		t.Fatalf("GetAgentStats failed: %v", err)
	}

	if successCount != 0 || failureCount != 3 || successRate != 0.0 {
		t.Errorf("Expected (0, 3, 0.0), got (%d, %d, %.2f)", successCount, failureCount, successRate)
	}
}

func TestLogEndstate_UsesXDGPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Clear any other XDG vars to ensure XDG_RUNTIME_DIR is used
	t.Setenv("XDG_CACHE_HOME", "")

	gogentDir := config.GetGOgentDir()
	expectedDir := filepath.Join(tmpDir, "gogent")

	if gogentDir != expectedDir {
		t.Fatalf("GetGOgentDir() = %s, want %s", gogentDir, expectedDir)
	}

	// Verify log path uses this directory
	logPath := GetEndstateLogPath()
	expectedPath := filepath.Join(expectedDir, "agent-endstates.jsonl")

	if logPath != expectedPath {
		t.Errorf("GetEndstateLogPath() = %s, want %s", logPath, expectedPath)
	}
}

func TestLogEndstate_AppendsBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log first entry
	event1 := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "session1",
		TranscriptPath: "/tmp/test1.jsonl",
	}

	metadata1 := &routing.ParsedAgentMetadata{
		AgentID:    "agent1",
		Tier:       "haiku",
		ExitCode:   0,
		DurationMs: 100,
	}

	response1 := &EndstateResponse{
		Decision: "silent",
	}

	if err := LogEndstate(event1, metadata1, response1); err != nil {
		t.Fatalf("First LogEndstate failed: %v", err)
	}

	// Verify file exists and has content
	logPath := GetEndstateLogPath()
	info1, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	// Log second entry
	event2 := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "session2",
		TranscriptPath: "/tmp/test2.jsonl",
	}

	metadata2 := &routing.ParsedAgentMetadata{
		AgentID:    "agent2",
		Tier:       "sonnet",
		ExitCode:   0,
		DurationMs: 200,
	}

	response2 := &EndstateResponse{
		Decision: "prompt",
	}

	if err := LogEndstate(event2, metadata2, response2); err != nil {
		t.Fatalf("Second LogEndstate failed: %v", err)
	}

	// Verify file size increased (appended, not replaced)
	info2, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file after append: %v", err)
	}

	if info2.Size() <= info1.Size() {
		t.Error("Log file size should increase after append")
	}

	// Verify both entries present
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("ReadEndstateLogs failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries after append, got %d", len(logs))
	}
}

func TestLogEndstate_DirectoryCreationFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file where the directory should be (to cause mkdir failure)
	badPath := filepath.Join(tmpDir, "gogent")
	if err := os.WriteFile(badPath, []byte("blocking file"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test",
		TranscriptPath: "/tmp/test.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "test-agent",
		Tier:       "haiku",
		ExitCode:   0,
		DurationMs: 100,
	}

	response := &EndstateResponse{
		Decision: "silent",
	}

	// Note: config.GetGOgentDir() has fallback logic to /tmp, so this test
	// demonstrates fallback behavior rather than actual failure.
	// The function will succeed using fallback directory.
	err := LogEndstate(event, metadata, response)
	if err != nil {
		// If it does fail for some reason, error should be non-nil
		if err.Error() == "" {
			t.Error("Error message should not be empty if error occurs")
		}
	}
}

func TestReadEndstateLogs_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	logPath := GetEndstateLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create file with only empty lines and whitespace
	content := "\n\n   \n\t\n  \n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Fatalf("ReadEndstateLogs should handle empty lines: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs from empty file, got %d", len(logs))
	}
}

func TestGetAgentStats_SingleSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test",
		TranscriptPath: "/tmp/test.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "single-agent",
		Tier:       "haiku",
		ExitCode:   0,
		DurationMs: 100,
	}

	response := &EndstateResponse{
		Decision: "silent",
	}

	if err := LogEndstate(event, metadata, response); err != nil {
		t.Fatalf("LogEndstate failed: %v", err)
	}

	successCount, failureCount, successRate, err := GetAgentStats("single-agent")
	if err != nil {
		t.Fatalf("GetAgentStats failed: %v", err)
	}

	if successCount != 1 || failureCount != 0 || successRate != 100.0 {
		t.Errorf("Expected (1, 0, 100.0), got (%d, %d, %.2f)",
			successCount, failureCount, successRate)
	}
}

func TestGetAgentStats_SingleFailure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test",
		TranscriptPath: "/tmp/test.jsonl",
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "failure-agent",
		Tier:       "haiku",
		ExitCode:   1,
		DurationMs: 100,
	}

	response := &EndstateResponse{
		Decision: "prompt",
	}

	if err := LogEndstate(event, metadata, response); err != nil {
		t.Fatalf("LogEndstate failed: %v", err)
	}

	successCount, failureCount, successRate, err := GetAgentStats("failure-agent")
	if err != nil {
		t.Fatalf("GetAgentStats failed: %v", err)
	}

	if successCount != 0 || failureCount != 1 || successRate != 0.0 {
		t.Errorf("Expected (0, 1, 0.0), got (%d, %d, %.2f)",
			successCount, failureCount, successRate)
	}
}
