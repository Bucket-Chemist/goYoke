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
	binaryPath := "../../bin/goyoke-archive"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("goyoke-archive binary not built. Run: make build-archive")
	}

	// Setup temp project directory
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create temp metrics files
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	goyokeDir := filepath.Join(runtimeDir, "goyoke")
	os.MkdirAll(goyokeDir, 0755)

	// Tool counter (new format: single file with count)
	counterFile := filepath.Join(goyokeDir, "tool-counter")
	os.WriteFile(counterFile, []byte("3"), 0644)

	// Error log
	errorLog := filepath.Join(goyokeDir, "claude-error-patterns.jsonl")
	os.WriteFile(errorLog, []byte(`{"error":"test"}
`), 0644)

	// Violations log (in runtime dir - matches config.GetViolationsLogPath())
	violationsLog := filepath.Join(goyokeDir, "routing-violations.jsonl")
	os.WriteFile(violationsLog, []byte(`{"violation":"test"}
`), 0644)

	// Prepare SessionEnd JSON
	sessionEvent := map[string]interface{}{
		"session_id":      "integration-test-session",
		"timestamp":       1234567890,
		"hook_event_name": "SessionEnd",
	}
	eventJSON, _ := json.Marshal(sessionEvent)

	// Invoke goyoke-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOYOKE_PROJECT_DIR="+tmpDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("goyoke-archive execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// SessionEnd hooks output empty JSON per Claude Code schema
	// (SessionEnd doesn't support hookSpecificOutput)
	output := bytes.TrimSpace(stdout.Bytes())
	if string(output) != "{}" {
		t.Errorf("Expected empty JSON '{}' on stdout, got: %s", output)
	}

	// Informational message should be on stderr
	if !bytes.Contains(stderr.Bytes(), []byte("SESSION ARCHIVED")) {
		t.Errorf("Expected SESSION ARCHIVED message on stderr, got: %s", stderr.String())
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
	binaryPath := "../../bin/goyoke-archive"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("goyoke-archive binary not built")
	}

	// Invalid JSON input
	cmd := exec.Command(binaryPath)
	cmd.Stdin = bytes.NewReader([]byte("{invalid json"))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}

	// SessionEnd outputs empty JSON on stdout even on error (schema compliance)
	output := bytes.TrimSpace(stdout.Bytes())
	if string(output) != "{}" {
		t.Errorf("Expected empty JSON '{}' on stdout, got: %s", output)
	}

	// Error message should be on stderr with component tag
	stderrOutput := stderr.String()
	if !bytes.Contains(stderr.Bytes(), []byte("[goyoke-archive]")) {
		t.Errorf("Error message should contain [goyoke-archive] component tag on stderr: %s", stderrOutput)
	}

	// Error emoji should be present
	if !bytes.Contains(stderr.Bytes(), []byte("🔴")) {
		t.Errorf("Error message should contain error emoji on stderr: %s", stderrOutput)
	}
}
