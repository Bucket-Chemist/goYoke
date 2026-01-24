package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEscapeJSON verifies JSON escaping in helper functions.
func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain", "plain"},
		{`text with "quotes"`, `text with \"quotes\"`},
		{"text with \nnewline", `text with \nnewline`},
		{`backslash\test`, `backslash\\test`},
	}

	for _, test := range tests {
		result := escapeJSON(test.input)
		if result != test.expected {
			t.Errorf("escapeJSON(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestOutputAllow verifies outputAllow produces valid JSON.
func TestOutputAllow(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputAllow("Test reason")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("outputAllow produced invalid JSON: %v\nOutput: %s", err, output)
	}

	// Verify structure
	hookOutput, ok := result["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput")
	}

	if decision, ok := hookOutput["decision"].(string); !ok || decision != "allow" {
		t.Errorf("Expected decision 'allow', got %v", hookOutput["decision"])
	}

	if reason, ok := hookOutput["reason"].(string); !ok || reason != "Test reason" {
		t.Errorf("Expected reason 'Test reason', got %v", hookOutput["reason"])
	}
}

// TestOutputError verifies outputError produces valid JSON with error context.
func TestOutputError(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputError("Test error message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("outputError produced invalid JSON: %v\nOutput: %s", err, output)
	}

	// Verify structure
	hookOutput, ok := result["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput")
	}

	if decision, ok := hookOutput["decision"].(string); !ok || decision != "allow" {
		t.Errorf("Expected decision 'allow' (error degrades to allow), got %v", hookOutput["decision"])
	}

	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, "Test error message") {
		t.Errorf("Expected additionalContext to contain error message, got %v", hookOutput["additionalContext"])
	}
}

// TestNonOrchestratorPassthrough verifies silent pass-through for non-orchestrator agents.
func TestNonOrchestratorPassthrough(t *testing.T) {
	// Create test transcript
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Transcript for python-pro agent (implementation class)
	content := `{"content": "AGENT: python-pro", "model": "sonnet"}
{"tool_name": "Read", "tool_input": {"file_path": "/test/file.py"}, "captured_at": 1000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Run main (will exit early with outputAllow for non-orchestrator)
	main()

	wo.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(ro)
	output := buf.String()

	// Verify output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Non-orchestrator passthrough produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' for non-orchestrator, got '%s'", decision)
	}
}

// TestParsingFailureGracefulDegradation verifies graceful degradation on transcript parse failure.
func TestParsingFailureGracefulDegradation(t *testing.T) {
	// Create test event with non-existent transcript
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  "/nonexistent/path/to/transcript.jsonl",
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	ro, wo, _ := os.Pipe()
	re, we, _ := os.Pipe()
	os.Stdout = wo
	os.Stderr = we

	// Run main
	main()

	wo.Close()
	we.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut bytes.Buffer
	bufOut.ReadFrom(ro)
	output := bufOut.String()

	var bufErr bytes.Buffer
	bufErr.ReadFrom(re)
	stderr := bufErr.String()

	// Verify warning logged to stderr
	if !strings.Contains(stderr, "Warning: Failed to parse transcript") {
		t.Errorf("Expected warning in stderr, got: %s", stderr)
	}

	// Verify allow decision
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Parse failure produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' on parsing failure, got '%s'", decision)
	}

	if reason := hookOutput["reason"].(string); !strings.Contains(reason, "parsing failed") {
		t.Errorf("Expected reason to mention parsing failure, got '%s'", reason)
	}
}

// TestOrchestratorWithAllTasksCollected verifies allow decision for orchestrator with all tasks collected.
func TestOrchestratorWithAllTasksCollected(t *testing.T) {
	// Create test transcript
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Transcript for orchestrator with all tasks collected
	content := `{"content": "AGENT: orchestrator", "model": "sonnet"}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-2"}, "captured_at": 2000}
{"tool_name": "TaskOutput", "tool_input": {"task_id": "bg-1"}, "captured_at": 3000}
{"tool_name": "TaskOutput", "tool_input": {"task_id": "bg-2"}, "captured_at": 4000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Run main
	main()

	wo.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(ro)
	output := buf.String()

	// Verify output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Orchestrator with all tasks collected produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' when all tasks collected, got '%s'", decision)
	}

	if reason := hookOutput["reason"].(string); !strings.Contains(reason, "All background tasks collected") {
		t.Errorf("Expected reason to mention all tasks collected, got '%s'", reason)
	}
}

// TestOrchestratorWithUncollectedTasks verifies block decision for orchestrator with uncollected tasks.
func TestOrchestratorWithUncollectedTasks(t *testing.T) {
	// Create test transcript
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Transcript for orchestrator with uncollected task
	content := `{"content": "AGENT: orchestrator", "model": "sonnet"}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-2"}, "captured_at": 2000}
{"tool_name": "TaskOutput", "tool_input": {"task_id": "bg-1"}, "captured_at": 3000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Run main
	main()

	wo.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(ro)
	output := buf.String()

	// Verify output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Orchestrator with uncollected tasks produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "block" {
		t.Errorf("Expected decision 'block' when tasks uncollected, got '%s'", decision)
	}

	if reason := hookOutput["reason"].(string); !strings.Contains(reason, "uncollected") {
		t.Errorf("Expected reason to mention uncollected tasks, got '%s'", reason)
	}

	// Verify remediation steps present
	steps, ok := hookOutput["remediationSteps"].([]interface{})
	if !ok || len(steps) == 0 {
		t.Errorf("Expected remediationSteps to be non-empty for block decision")
	}
}

// TestAnalysisFailureGracefulDegradation verifies graceful degradation when analysis fails.
func TestAnalysisFailureGracefulDegradation(t *testing.T) {
	// Create test transcript with malformed content
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Malformed content that should cause analysis issues
	content := "not valid json at all\n"
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Run main - should degrade gracefully
	main()

	wo.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	var bufOut bytes.Buffer
	bufOut.ReadFrom(ro)
	output := bufOut.String()

	// Verify output is valid JSON with allow decision (graceful degradation)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Analysis failure produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' on analysis failure (graceful degradation), got '%s'", decision)
	}
}

// TestArchitectAgentWithUncollectedTasks verifies architect (orchestrator class) is also guarded.
func TestArchitectAgentWithUncollectedTasks(t *testing.T) {
	// Create test transcript
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Transcript for architect with uncollected task
	content := `{"content": "AGENT: architect", "model": "sonnet"}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-plan"}, "captured_at": 1000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event
	event := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "test-session",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}

	eventJSON, _ := json.Marshal(event)

	// Simulate stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write(eventJSON)
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Run main
	main()

	wo.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(ro)
	output := buf.String()

	// Verify output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Architect agent produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})

	// Architect is orchestrator class, should be blocked
	if decision := hookOutput["decision"].(string); decision != "block" {
		t.Errorf("Expected decision 'block' for architect with uncollected tasks, got '%s'", decision)
	}
}
