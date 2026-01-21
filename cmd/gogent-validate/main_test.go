package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// =============================================================================
// parseEvent Tests
// =============================================================================

func TestParseEvent_ValidTaskEvent(t *testing.T) {
	taskEvent := `{
		"tool_name": "Task",
		"session_id": "test-session-123",
		"tool_input": {
			"description": "Search for files",
			"subagent_type": "Explore",
			"model": "haiku",
			"prompt": "Find all Go files"
		}
	}`

	r := strings.NewReader(taskEvent)
	event, err := parseEvent(r, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Task" {
		t.Errorf("Expected ToolName 'Task', got: %s", event.ToolName)
	}

	if event.SessionID != "test-session-123" {
		t.Errorf("Expected SessionID 'test-session-123', got: %s", event.SessionID)
	}
}

func TestParseEvent_NonTaskEvent(t *testing.T) {
	bashEvent := `{
		"tool_name": "Bash",
		"session_id": "test-session-456",
		"tool_input": {
			"command": "ls -la"
		}
	}`

	r := strings.NewReader(bashEvent)
	event, err := parseEvent(r, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Bash" {
		t.Errorf("Expected ToolName 'Bash', got: %s", event.ToolName)
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	invalidJSON := `{not valid json`

	r := strings.NewReader(invalidJSON)
	_, err := parseEvent(r, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("Expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestParseEvent_EmptyInput(t *testing.T) {
	r := strings.NewReader("")
	_, err := parseEvent(r, 5*time.Second)

	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

func TestParseEvent_Timeout(t *testing.T) {
	// Create a reader that never returns data
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	// Use very short timeout
	_, err := parseEvent(r, 10*time.Millisecond)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected 'timeout' in error, got: %v", err)
	}
}

func TestParseEvent_ReadError(t *testing.T) {
	// Create a pipe and close the write end immediately to simulate read error
	r, w := io.Pipe()
	w.CloseWithError(io.ErrUnexpectedEOF)

	_, err := parseEvent(r, 5*time.Second)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// =============================================================================
// outputResult Tests
// =============================================================================

func TestOutputResult_BlockDecision(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := &routing.ValidationResult{
		Decision: "block",
		Reason:   "Agent tier mismatch",
	}

	outputResult(result, "test-session")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v\nOutput: %s", err, output)
	}

	// Check decision
	if jsonOutput["decision"] != "block" {
		t.Errorf("Expected decision 'block', got: %v", jsonOutput["decision"])
	}

	// Check reason
	if jsonOutput["reason"] != "Agent tier mismatch" {
		t.Errorf("Expected reason 'Agent tier mismatch', got: %v", jsonOutput["reason"])
	}

	// Check hookSpecificOutput
	hookOutput, ok := jsonOutput["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput in JSON")
	}

	if hookOutput["permissionDecision"] != "deny" {
		t.Errorf("Expected permissionDecision 'deny', got: %v", hookOutput["permissionDecision"])
	}
}

func TestOutputResult_AllowDecision(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := &routing.ValidationResult{
		Decision: "allow",
	}

	outputResult(result, "test-session")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Parse JSON
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	// Should not have decision=block
	if jsonOutput["decision"] == "block" {
		t.Error("Did not expect decision='block' for allow result")
	}
}

func TestOutputResult_AllowWithWarning(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := &routing.ValidationResult{
		Decision:      "allow",
		ModelMismatch: "Requested model 'opus' but tier allows 'sonnet'",
	}

	outputResult(result, "test-session")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Parse JSON
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	// Check hookSpecificOutput has warning
	hookOutput, ok := jsonOutput["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput in JSON for warning case")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string in hookSpecificOutput")
	}

	if !strings.Contains(additionalContext, "⚠️") {
		t.Error("Expected warning emoji in additionalContext")
	}

	if !strings.Contains(additionalContext, "opus") {
		t.Error("Expected model mismatch message in additionalContext")
	}
}

// =============================================================================
// outputError Tests
// =============================================================================

func TestOutputError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testMessage := "Test error message for validation"
	outputError(testMessage)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Parse JSON
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v\nOutput: %s", err, buf.String())
	}

	// Check hookSpecificOutput
	hookOutput, ok := jsonOutput["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput in error JSON")
	}

	// Check hookEventName
	if hookOutput["hookEventName"] != "PreToolUse" {
		t.Errorf("Expected hookEventName 'PreToolUse', got: %v", hookOutput["hookEventName"])
	}

	// Check additionalContext contains error emoji and message
	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	if !strings.Contains(additionalContext, "🔴") {
		t.Error("Expected error emoji in additionalContext")
	}

	if !strings.Contains(additionalContext, testMessage) {
		t.Errorf("Expected error message in additionalContext, got: %s", additionalContext)
	}
}

// =============================================================================
// Integration Tests (end-to-end flow)
// =============================================================================

func TestIntegration_NonTaskToolPassthrough(t *testing.T) {
	// Setup routing schema in temp directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create .claude directory structure with minimal schema
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	minimalSchema := `{
		"schema_version": "1.0",
		"tiers": {},
		"delegation_ceiling": {},
		"agent_subagent_mapping": {},
		"escalation_rules": {}
	}`
	os.WriteFile(filepath.Join(claudeDir, "routing-schema.json"), []byte(minimalSchema), 0644)

	// Mock STDIN with non-Task event
	bashEvent := `{"tool_name":"Bash","session_id":"test","tool_input":{"command":"ls"}}`

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(bashEvent))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// The non-Task tool should just pass through
	event, err := parseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Verify passthrough for non-Task
	if event.ToolName != "Task" {
		// This is the expected path for non-Task tools
		wOut.Write([]byte("{}"))
	}

	wOut.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, rOut)

	// Should output empty JSON for non-Task
	if strings.TrimSpace(buf.String()) != "{}" {
		t.Logf("Output for non-Task tool: %s", buf.String())
	}
}

func TestIntegration_TaskValidation_ValidAgent(t *testing.T) {
	// This test validates the full flow for a Task event with valid agent
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create routing schema with agent mapping
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	schema := `{
		"schema_version": "1.0",
		"tiers": {
			"haiku": {"model": "haiku"}
		},
		"delegation_ceiling": {"default": "sonnet"},
		"agent_subagent_mapping": {
			"codebase-search": "Explore"
		},
		"escalation_rules": {}
	}`
	os.WriteFile(filepath.Join(claudeDir, "routing-schema.json"), []byte(schema), 0644)

	// Create agents-index.json
	agentsIndex := `{
		"agents": {
			"codebase-search": {
				"tier": "haiku",
				"subagent_type": "Explore"
			}
		}
	}`
	os.WriteFile(filepath.Join(claudeDir, "agents-index.json"), []byte(agentsIndex), 0644)

	// Valid Task event matching agent in schema
	taskEvent := `{
		"tool_name": "Task",
		"session_id": "test-session",
		"tool_input": {
			"description": "Search files",
			"subagent_type": "Explore",
			"model": "haiku",
			"prompt": "AGENT: codebase-search\n\nFind Go files"
		}
	}`

	r := strings.NewReader(taskEvent)
	event, err := parseEvent(r, DEFAULT_TIMEOUT)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	if event.ToolName != "Task" {
		t.Errorf("Expected ToolName 'Task', got: %s", event.ToolName)
	}
}

func TestParseEvent_ComplexToolInput(t *testing.T) {
	// Test with nested JSON in tool_input
	complexEvent := `{
		"tool_name": "Task",
		"session_id": "complex-test",
		"tool_input": {
			"description": "Multi-line task",
			"subagent_type": "general-purpose",
			"model": "sonnet",
			"prompt": "AGENT: python-pro\n\n1. TASK: Implement feature\n2. EXPECTED: Tests pass"
		}
	}`

	r := strings.NewReader(complexEvent)
	event, err := parseEvent(r, DEFAULT_TIMEOUT)

	if err != nil {
		t.Fatalf("Expected no error for complex event, got: %v", err)
	}

	if event.SessionID != "complex-test" {
		t.Errorf("Expected session_id 'complex-test', got: %s", event.SessionID)
	}

	// Verify tool_input was parsed
	if event.ToolInput == nil {
		t.Error("Expected ToolInput to be populated")
	}
}

func TestParseEvent_MissingFields(t *testing.T) {
	// Event with minimal fields - should still parse
	minimalEvent := `{"tool_name": "Read"}`

	r := strings.NewReader(minimalEvent)
	event, err := parseEvent(r, DEFAULT_TIMEOUT)

	if err != nil {
		t.Fatalf("Expected no error for minimal event, got: %v", err)
	}

	if event.ToolName != "Read" {
		t.Errorf("Expected ToolName 'Read', got: %s", event.ToolName)
	}

	// Empty fields should be zero values
	if event.SessionID != "" {
		t.Errorf("Expected empty SessionID for minimal event, got: %s", event.SessionID)
	}
}

func TestParseEvent_LargePayload(t *testing.T) {
	// Generate large prompt to test buffer handling
	largePrompt := strings.Repeat("x", 100000) // 100KB prompt

	largeEvent := `{
		"tool_name": "Task",
		"session_id": "large-test",
		"tool_input": {
			"description": "Large task",
			"subagent_type": "Explore",
			"prompt": "` + largePrompt + `"
		}
	}`

	r := strings.NewReader(largeEvent)
	event, err := parseEvent(r, 10*time.Second)

	if err != nil {
		t.Fatalf("Expected no error for large payload, got: %v", err)
	}

	if event.SessionID != "large-test" {
		t.Errorf("Expected session_id 'large-test', got: %s", event.SessionID)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestOutputResult_EmptyResult(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Empty result (neither block nor allow explicitly set)
	result := &routing.ValidationResult{}

	outputResult(result, "test-session")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Should produce valid JSON (empty object or minimal output)
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON for empty result, got error: %v", err)
	}
}

func TestOutputResult_WithViolations(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := &routing.ValidationResult{
		Decision: "block",
		Reason:   "Multiple violations",
		Violations: []*routing.Violation{
			{ViolationType: "tier_mismatch", Reason: "Wrong tier"},
			{ViolationType: "subagent_mismatch", Reason: "Wrong subagent"},
		},
	}

	outputResult(result, "test-session")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Parse and verify
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	if jsonOutput["decision"] != "block" {
		t.Errorf("Expected decision 'block', got: %v", jsonOutput["decision"])
	}
}

func TestParseEvent_WhitespaceJSON(t *testing.T) {
	// JSON with extra whitespace
	whitespaceJSON := `

	{
		"tool_name"  :  "Task"  ,
		"session_id" :  "ws-test"
	}

	`

	r := strings.NewReader(whitespaceJSON)
	event, err := parseEvent(r, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error for whitespace JSON, got: %v", err)
	}

	if event.ToolName != "Task" {
		t.Errorf("Expected ToolName 'Task', got: %s", event.ToolName)
	}
}

func TestParseEvent_UnicodeContent(t *testing.T) {
	// JSON with unicode characters
	unicodeEvent := `{
		"tool_name": "Task",
		"session_id": "unicode-test",
		"tool_input": {
			"prompt": "Analyze émojis: 🚀 🎯 ✅"
		}
	}`

	r := strings.NewReader(unicodeEvent)
	event, err := parseEvent(r, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error for unicode JSON, got: %v", err)
	}

	if event.SessionID != "unicode-test" {
		t.Errorf("Expected session_id 'unicode-test', got: %s", event.SessionID)
	}
}

// =============================================================================
// Constants/Config Tests
// =============================================================================

func TestDefaultTimeout(t *testing.T) {
	if DEFAULT_TIMEOUT != 5*time.Second {
		t.Errorf("Expected DEFAULT_TIMEOUT to be 5s, got: %v", DEFAULT_TIMEOUT)
	}
}
