package main

import (
	"bytes"
	"encoding/json"
	"os"
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
		{`mixed "quotes" and \backslash`, `mixed \"quotes\" and \\backslash`},
	}

	for _, test := range tests {
		result := escapeJSON(test.input)
		if result != test.expected {
			t.Errorf("escapeJSON(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestAllowResponse verifies allowResponse produces valid JSON.
func TestAllowResponse(t *testing.T) {
	response := allowResponse()

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		t.Fatalf("allowResponse produced invalid JSON: %v\nOutput: %s", err, response)
	}

	// Verify structure
	hookOutput, ok := result["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput")
	}

	if decision, ok := hookOutput["decision"].(string); !ok || decision != "allow" {
		t.Errorf("Expected decision 'allow', got %v", hookOutput["decision"])
	}

	if hookEventName, ok := hookOutput["hookEventName"].(string); !ok || hookEventName != "PreToolUse" {
		t.Errorf("Expected hookEventName 'PreToolUse', got %v", hookOutput["hookEventName"])
	}
}

// TestWarnResponse verifies warnResponse produces valid JSON with warning message.
func TestWarnResponse(t *testing.T) {
	testMessage := "Test warning message"
	response := warnResponse(testMessage)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		t.Fatalf("warnResponse produced invalid JSON: %v\nOutput: %s", err, response)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})

	if decision, ok := hookOutput["decision"].(string); !ok || decision != "warn" {
		t.Errorf("Expected decision 'warn', got %v", hookOutput["decision"])
	}

	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, testMessage) {
		t.Errorf("Expected additionalContext to contain warning message, got %v", hookOutput["additionalContext"])
	}
}

// TestBlockResponse verifies blockResponse produces valid JSON with block message.
func TestBlockResponse(t *testing.T) {
	testMessage := "Theater pattern detected"
	response := blockResponse(testMessage)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		t.Fatalf("blockResponse produced invalid JSON: %v\nOutput: %s", err, response)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})

	if decision, ok := hookOutput["decision"].(string); !ok || decision != "block" {
		t.Errorf("Expected decision 'block', got %v", hookOutput["decision"])
	}

	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, "BLOCKED") {
		t.Errorf("Expected additionalContext to contain BLOCKED prefix, got %v", hookOutput["additionalContext"])
	}

	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, testMessage) {
		t.Errorf("Expected additionalContext to contain message, got %v", hookOutput["additionalContext"])
	}
}

// TestNonWriteOperationPassthrough verifies silent passthrough for Read operations.
func TestNonWriteOperationPassthrough(t *testing.T) {
	event := map[string]interface{}{
		"tool_name":       "Read",
		"tool_input":      map[string]interface{}{"file_path": "/test/CLAUDE.md"},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Non-write operation produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' for Read operation, got '%s'", decision)
	}
}

// TestNonClaudeMDPassthrough verifies silent passthrough for non-CLAUDE.md files.
func TestNonClaudeMDPassthrough(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/README.md",
			"content":   "Some content with MUST NOT pattern",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Non-CLAUDE.md file produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' for non-CLAUDE.md file, got '%s'", decision)
	}
}

// TestCleanContentAllowed verifies allow decision for clean content without theater patterns.
func TestCleanContentAllowed(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.md",
			"content":   "This is clean documentation without enforcement theater.",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Clean content produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' for clean content, got '%s'", decision)
	}
}

// TestTheaterPatternWarning verifies warn decision for MUST NOT patterns (default mode).
func TestTheaterPatternWarning(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.md",
			"content":   "You MUST NOT use this tool without permission.",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Theater pattern produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "warn" {
		t.Errorf("Expected decision 'warn' for theater pattern (default mode), got '%s'", decision)
	}

	// Verify warning message is present
	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, "DOCUMENTATION THEATER") {
		t.Errorf("Expected additionalContext to contain warning about documentation theater, got %v", hookOutput["additionalContext"])
	}
}

// TestTheaterPatternBlocking verifies block decision when GOYOKE_DOC_THEATER_BLOCK=true.
func TestTheaterPatternBlocking(t *testing.T) {
	// Set environment variable for blocking mode
	oldEnv := os.Getenv("GOYOKE_DOC_THEATER_BLOCK")
	os.Setenv("GOYOKE_DOC_THEATER_BLOCK", "true")
	defer func() {
		if oldEnv == "" {
			os.Unsetenv("GOYOKE_DOC_THEATER_BLOCK")
		} else {
			os.Setenv("GOYOKE_DOC_THEATER_BLOCK", oldEnv)
		}
	}()

	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.md",
			"content":   "This is BLOCKED and you MUST NOT use it.",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Theater pattern in blocking mode produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "block" {
		t.Errorf("Expected decision 'block' when blocking mode enabled, got '%s'", decision)
	}

	// Verify block message
	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, "BLOCKED") {
		t.Errorf("Expected additionalContext to contain BLOCKED prefix, got %v", hookOutput["additionalContext"])
	}
}

// TestEditOperationDetection verifies detection works for Edit tool as well as Write.
func TestEditOperationDetection(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Edit",
		"tool_input": map[string]interface{}{
			"file_path":  "/test/CLAUDE.md",
			"old_string": "old content",
			"new_string": "You MUST NOT do this operation.",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Edit operation produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "warn" {
		t.Errorf("Expected decision 'warn' for Edit operation with theater pattern, got '%s'", decision)
	}
}

// TestEmptyContentAllowed verifies allow decision when content is empty.
func TestEmptyContentAllowed(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.md",
			"content":   "",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Empty content produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow' for empty content, got '%s'", decision)
	}
}

// TestOutputError verifies outputError produces valid JSON with error message.
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

// TestMultiplePatternDetection verifies detection of multiple theater patterns.
func TestMultiplePatternDetection(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.md",
			"content": `
This is BLOCKED for security reasons.
You MUST NOT use this feature.
NEVER execute this command.
This is FORBIDDEN by policy.
`,
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("Multiple patterns produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "warn" {
		t.Errorf("Expected decision 'warn' for multiple patterns, got '%s'", decision)
	}

	// Verify warning mentions multiple patterns
	if context, ok := hookOutput["additionalContext"].(string); !ok || !strings.Contains(context, "enforcement pattern") {
		t.Errorf("Expected additionalContext to mention enforcement patterns, got %v", hookOutput["additionalContext"])
	}
}

// TestClaudeMDVariantDetection verifies detection works for CLAUDE.en.md and similar variants.
func TestClaudeMDVariantDetection(t *testing.T) {
	event := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/test/CLAUDE.en.md",
			"content":   "You MUST NOT do this.",
		},
		"session_id":      "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at":     1000,
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
		t.Fatalf("CLAUDE variant produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := result["hookSpecificOutput"].(map[string]interface{})
	if decision := hookOutput["decision"].(string); decision != "warn" {
		t.Errorf("Expected decision 'warn' for CLAUDE.en.md variant, got '%s'", decision)
	}
}
