package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSessionEndHook_EndToEnd(t *testing.T) {
	// Skip if binary not built
	binaryPath := "../../bin/gogent-archive"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("gogent-archive binary not built. Run: make build-archive")
	}

	// Setup temp project directory
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create temp metrics files
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	gogentDir := filepath.Join(runtimeDir, "gogent")
	os.MkdirAll(gogentDir, 0755)

	// Tool counter
	counterFile := filepath.Join(gogentDir, "claude-tool-counter-test.log")
	os.WriteFile(counterFile, []byte("line1\nline2\nline3\n"), 0644)

	// Error log
	errorLog := filepath.Join(gogentDir, "claude-error-patterns.jsonl")
	os.WriteFile(errorLog, []byte(`{"error":"test"}
`), 0644)

	// Violations log (in runtime dir - matches config.GetViolationsLogPath())
	violationsLog := filepath.Join(gogentDir, "routing-violations.jsonl")
	os.WriteFile(violationsLog, []byte(`{"violation":"test"}
`), 0644)

	// Prepare SessionEnd JSON
	sessionEvent := map[string]interface{}{
		"session_id":      "integration-test-session",
		"timestamp":       1234567890,
		"hook_event_name": "SessionEnd",
	}
	eventJSON, _ := json.Marshal(sessionEvent)

	// Invoke gogent-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+tmpDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("gogent-archive execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse output JSON
	var confirmation map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &confirmation); err != nil {
		t.Fatalf("Failed to parse confirmation JSON: %v\nOutput: %s", err, stdout.String())
	}

	// Verify hookSpecificOutput exists
	hookOutput, ok := confirmation["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput in confirmation")
	}

	if hookOutput["session_id"] != "integration-test-session" {
		t.Errorf("Expected session_id in output, got: %v", hookOutput)
	}

	// Verify handoff files created
	handoffJSONL := filepath.Join(memoryDir, "handoffs.jsonl")
	if _, err := os.Stat(handoffJSONL); os.IsNotExist(err) {
		t.Error("handoffs.jsonl not created")
	}

	handoffMD := filepath.Join(memoryDir, "last-handoff.md")
	if _, err := os.Stat(handoffMD); os.IsNotExist(err) {
		t.Error("last-handoff.md not created")
	}

	// Verify artifacts archived
	archiveDir := filepath.Join(memoryDir, "session-archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Error("session-archive/ directory not created")
	}

	// Original violations should be moved
	if _, err := os.Stat(violationsLog); !os.IsNotExist(err) {
		t.Error("Violations log should have been moved to archive")
	}
}

func TestSessionEndHook_ErrorHandling(t *testing.T) {
	binaryPath := "../../bin/gogent-archive"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("gogent-archive binary not built")
	}

	// Invalid JSON input
	cmd := exec.Command(binaryPath)
	cmd.Stdin = bytes.NewReader([]byte("{invalid json"))

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}

	// Error is written to stdout as JSON with component tag in message
	var errorOutput map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &errorOutput); err != nil {
		t.Fatalf("Failed to parse error output as JSON: %v\nOutput: %s", err, stdout.String())
	}

	// Check for hookSpecificOutput with error message containing component tag
	hookOutput, ok := errorOutput["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput in error output")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok || additionalContext == "" {
		t.Fatal("Missing or empty additionalContext in error output")
	}

	// Error message should contain component tag
	if !bytes.Contains([]byte(additionalContext), []byte("[gogent-archive]")) {
		t.Errorf("Error message should contain [gogent-archive] component tag: %s", additionalContext)
	}
}
