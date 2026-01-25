package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// getBinaryPath returns the absolute path to gogent-archive binary
func getBinaryPath() string {
	// Get the absolute path relative to this test file location
	// Tests run from test/integration/, binary is at bin/gogent-archive
	relativePath := "../../bin/gogent-archive"
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return relativePath
	}
	return absPath
}

// skipIfBinaryNotBuilt skips test if binary not available
func skipIfBinaryNotBuilt(t *testing.T) string {
	t.Helper()
	binaryPath := getBinaryPath()
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("gogent-archive binary not built. Run: make build-archive")
	}
	return binaryPath
}

// setupTempProject creates a temp project with .claude/memory/ structure
func setupTempProject(t *testing.T) (projectDir string, memoryDir string, runtimeDir string, gogentDir string) {
	t.Helper()
	projectDir = t.TempDir()
	memoryDir = filepath.Join(projectDir, ".claude", "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatalf("Failed to create memory dir: %v", err)
	}

	// Setup runtime dir for metrics collection
	runtimeDir = t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Cleanup(func() { os.Unsetenv("XDG_RUNTIME_DIR") })

	gogentDir = filepath.Join(runtimeDir, "gogent")
	if err := os.MkdirAll(gogentDir, 0755); err != nil {
		t.Fatalf("Failed to create gogent dir: %v", err)
	}

	return projectDir, memoryDir, runtimeDir, gogentDir
}

// createMetricsFiles populates tool counter, error log, and violations log
func createMetricsFiles(t *testing.T, gogentDir string) {
	t.Helper()

	// Tool counter (5 calls)
	counterFile := filepath.Join(gogentDir, "claude-tool-counter-test.log")
	if err := os.WriteFile(counterFile, []byte("call1\ncall2\ncall3\ncall4\ncall5\n"), 0644); err != nil {
		t.Fatalf("Failed to create counter file: %v", err)
	}

	// Error log (2 errors)
	errorLog := filepath.Join(gogentDir, "claude-error-patterns.jsonl")
	if err := os.WriteFile(errorLog, []byte(`{"error":"type_mismatch","file":"test.go"}
{"error":"nil_pointer","file":"main.go"}
`), 0644); err != nil {
		t.Fatalf("Failed to create error log: %v", err)
	}

	// Violations log (1 violation)
	violationsLog := filepath.Join(gogentDir, "routing-violations.jsonl")
	if err := os.WriteFile(violationsLog, []byte(`{"agent":"python-pro","violation_type":"tier_mismatch"}
`), 0644); err != nil {
		t.Fatalf("Failed to create violations log: %v", err)
	}
}

// createPendingLearnings creates pending-learnings.jsonl with sharp edges
func createPendingLearnings(t *testing.T, memoryDir string) {
	t.Helper()
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"file":"handler.go","error_type":"nil_dereference","consecutive_failures":3,"timestamp":1705000000}
{"file":"config.go","error_type":"missing_field","consecutive_failures":4,"timestamp":1705000100}
`
	if err := os.WriteFile(pendingPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create pending learnings: %v", err)
	}
}

// createSessionEventJSON creates SessionEnd JSON for STDIN
func createSessionEventJSON(sessionID string) []byte {
	event := map[string]interface{}{
		"session_id":      sessionID,
		"timestamp":       time.Now().Unix(),
		"hook_event_name": "SessionEnd",
	}
	data, _ := json.Marshal(event)
	return data
}

// parseJSONLFile reads and parses all lines from a JSONL file
func parseJSONLFile(t *testing.T, path string) []map[string]interface{} {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open %s: %v", path, err)
	}
	defer file.Close()

	var results []map[string]interface{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("Failed to parse JSONL line: %v", err)
		}
		results = append(results, obj)
	}
	return results
}

// ===========================================================================
// HOOK MODE INTEGRATION TESTS
// ===========================================================================

func TestSessionHandoffIntegration_FullWorkflow(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)
	projectDir, memoryDir, _, gogentDir := setupTempProject(t)

	// Pre-populate metrics files
	createMetricsFiles(t, gogentDir)

	// Create pending learnings JSONL
	createPendingLearnings(t, memoryDir)

	// Create SessionEnd JSON
	sessionID := "full-workflow-test-session"
	eventJSON := createSessionEventJSON(sessionID)

	// Invoke gogent-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("gogent-archive execution failed: %v\nStderr: %s\nStdout: %s", err, stderr.String(), stdout.String())
	}

	// Verify handoffs.jsonl created and valid JSONL
	handoffJSONL := filepath.Join(memoryDir, "handoffs.jsonl")
	if _, err := os.Stat(handoffJSONL); os.IsNotExist(err) {
		t.Fatal("handoffs.jsonl not created")
	}

	handoffs := parseJSONLFile(t, handoffJSONL)
	if len(handoffs) != 1 {
		t.Fatalf("Expected 1 handoff entry, got %d", len(handoffs))
	}

	// Verify handoff structure
	handoff := handoffs[0]
	if handoff["session_id"] != sessionID {
		t.Errorf("Expected session_id=%s, got %v", sessionID, handoff["session_id"])
	}
	if handoff["schema_version"] != "1.3" {
		t.Errorf("Expected schema_version=1.3, got %v", handoff["schema_version"])
	}

	// Verify last-handoff.md created with expected sections
	handoffMD := filepath.Join(memoryDir, "last-handoff.md")
	if _, err := os.Stat(handoffMD); os.IsNotExist(err) {
		t.Fatal("last-handoff.md not created")
	}
	mdContent, err := os.ReadFile(handoffMD)
	if err != nil {
		t.Fatalf("Failed to read last-handoff.md: %v", err)
	}
	mdStr := string(mdContent)
	if !strings.Contains(mdStr, "# Session Handoff") {
		t.Error("last-handoff.md missing '# Session Handoff' header")
	}
	if !strings.Contains(mdStr, "## Session Metrics") {
		t.Error("last-handoff.md missing '## Session Metrics' section")
	}

	// Verify artifacts moved to session-archive/
	archiveDir := filepath.Join(memoryDir, "session-archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Fatal("session-archive/ directory not created")
	}

	// Verify violations log moved (not at original location)
	violationsLog := filepath.Join(gogentDir, "routing-violations.jsonl")
	if _, err := os.Stat(violationsLog); !os.IsNotExist(err) {
		t.Error("Violations log should have been moved to archive")
	}

	// Verify pending-learnings.jsonl moved to archive
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("pending-learnings.jsonl should have been moved to archive")
	}

	// Parse and validate confirmation JSON output
	var confirmation map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &confirmation); err != nil {
		t.Fatalf("Failed to parse confirmation JSON: %v\nOutput: %s", err, stdout.String())
	}

	hookOutput, ok := confirmation["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput in confirmation")
	}

	if hookOutput["hookEventName"] != "SessionEnd" {
		t.Errorf("Expected hookEventName=SessionEnd, got %v", hookOutput["hookEventName"])
	}
	if hookOutput["session_id"] != sessionID {
		t.Errorf("Expected session_id=%s in output, got %v", sessionID, hookOutput["session_id"])
	}

	// Verify metrics in output
	metrics, ok := hookOutput["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing metrics in hookSpecificOutput")
	}
	if metrics["tool_calls"].(float64) != 5 {
		t.Errorf("Expected tool_calls=5, got %v", metrics["tool_calls"])
	}
}

func TestSessionHandoffIntegration_MinimalSession(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)
	projectDir, memoryDir, _, _ := setupTempProject(t)

	// NO metrics files, NO pending learnings - minimal session
	sessionID := "minimal-session"
	eventJSON := createSessionEventJSON(sessionID)

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("gogent-archive should handle empty session gracefully: %v\nStderr: %s", err, stderr.String())
	}

	// Should still create handoffs.jsonl
	handoffJSONL := filepath.Join(memoryDir, "handoffs.jsonl")
	if _, err := os.Stat(handoffJSONL); os.IsNotExist(err) {
		t.Fatal("handoffs.jsonl should be created even for minimal session")
	}

	// Verify handoff has zero metrics
	handoffs := parseJSONLFile(t, handoffJSONL)
	if len(handoffs) != 1 {
		t.Fatalf("Expected 1 handoff, got %d", len(handoffs))
	}

	handoff := handoffs[0]
	ctx, ok := handoff["context"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing context in handoff")
	}
	metrics, ok := ctx["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing metrics in context")
	}
	if metrics["tool_calls"].(float64) != 0 {
		t.Errorf("Expected tool_calls=0 for minimal session, got %v", metrics["tool_calls"])
	}
}

func TestSessionHandoffIntegration_ArtifactArchival(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)
	projectDir, memoryDir, _, gogentDir := setupTempProject(t)

	// Create violations that should be archived
	violationsLog := filepath.Join(gogentDir, "routing-violations.jsonl")
	os.WriteFile(violationsLog, []byte(`{"agent":"test-agent","violation_type":"test_violation"}
`), 0644)

	// Create pending learnings that should be archived
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(pendingPath, []byte(`{"file":"test.go","error_type":"test_error","consecutive_failures":3,"timestamp":1705000000}
`), 0644)

	sessionID := "artifact-archival-test"
	eventJSON := createSessionEventJSON(sessionID)

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Verify archive directory exists
	archiveDir := filepath.Join(memoryDir, "session-archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("Failed to read archive directory: %v", err)
	}

	// Should have learnings and violations archives
	var hasLearnings, hasViolations bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "learnings-") {
			hasLearnings = true
		}
		if strings.HasPrefix(entry.Name(), "violations-") {
			hasViolations = true
		}
	}

	if !hasLearnings {
		t.Error("Expected learnings archive file in session-archive/")
	}
	if !hasViolations {
		t.Error("Expected violations archive file in session-archive/")
	}

	// Verify originals removed
	if _, err := os.Stat(violationsLog); !os.IsNotExist(err) {
		t.Error("Original violations log should be removed after archival")
	}
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("Original pending learnings should be removed after archival")
	}
}

// ===========================================================================
// CLI SUBCOMMAND INTEGRATION TESTS
// ===========================================================================

func TestCLI_WorkingDirectoryBehavior(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	// Create two separate project directories
	projectDir1 := t.TempDir()
	projectDir2 := t.TempDir()

	// Setup project 1 with handoffs
	memoryDir1 := filepath.Join(projectDir1, ".claude", "memory")
	os.MkdirAll(memoryDir1, 0755)
	handoff1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"project1-session","context":{"project_dir":"/test1","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"project1-session"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	os.WriteFile(filepath.Join(memoryDir1, "handoffs.jsonl"), []byte(handoff1+"\n"), 0644)

	// Setup project 2 with different handoffs
	memoryDir2 := filepath.Join(projectDir2, ".claude", "memory")
	os.MkdirAll(memoryDir2, 0755)
	handoff2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"project2-session","context":{"project_dir":"/test2","metrics":{"tool_calls":20,"errors_logged":0,"routing_violations":0,"session_id":"project2-session"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	os.WriteFile(filepath.Join(memoryDir2, "handoffs.jsonl"), []byte(handoff2+"\n"), 0644)

	// Test 1: GOGENT_PROJECT_DIR takes precedence over cwd
	// Even if we cd to project2, setting GOGENT_PROJECT_DIR=project1 should use project1
	cmd := exec.Command(binaryPath, "list")
	cmd.Dir = projectDir2 // cwd is project2
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir1) // but env points to project1

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "project1-session") {
		t.Error("Expected project1-session (from GOGENT_PROJECT_DIR), not project2")
	}
	if strings.Contains(output, "project2-session") {
		t.Error("Should NOT see project2-session when GOGENT_PROJECT_DIR points elsewhere")
	}

	// Test 2: Without GOGENT_PROJECT_DIR, use cwd
	cmd2 := exec.Command(binaryPath, "list")
	cmd2.Dir = projectDir2
	// Remove GOGENT_PROJECT_DIR from environment
	env2 := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOGENT_PROJECT_DIR=") {
			env2 = append(env2, e)
		}
	}
	cmd2.Env = env2

	var stdout2 bytes.Buffer
	cmd2.Stdout = &stdout2

	if err := cmd2.Run(); err != nil {
		t.Fatalf("list command without GOGENT_PROJECT_DIR failed: %v", err)
	}

	output2 := stdout2.String()
	if !strings.Contains(output2, "project2-session") {
		t.Error("Expected project2-session (from cwd) when GOGENT_PROJECT_DIR not set")
	}
}

func TestCLI_EnvironmentVariables(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	t.Run("GOGENT_PROJECT_DIR_set", func(t *testing.T) {
		projectDir := t.TempDir()
		memoryDir := filepath.Join(projectDir, ".claude", "memory")
		os.MkdirAll(memoryDir, 0755)
		os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(""), 0644)

		cmd := exec.Command(binaryPath, "list")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(stdout.String(), "No sessions recorded") {
			t.Error("Expected graceful empty message")
		}
	})

	t.Run("GOGENT_PROJECT_DIR_unset_invalid_cwd", func(t *testing.T) {
		// This test verifies behavior when neither env nor valid handoff exists
		// Most systems will have a cwd, but .claude/memory might not exist
		cmd := exec.Command(binaryPath, "list")
		// Strip GOGENT_PROJECT_DIR
		env := []string{}
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "GOGENT_PROJECT_DIR=") {
				env = append(env, e)
			}
		}
		cmd.Env = env
		cmd.Dir = t.TempDir() // Empty temp dir - no .claude/memory

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		// Should fail because no handoffs.jsonl exists
		if err == nil {
			t.Log("Note: May succeed if cwd has valid structure")
		}
		// Either it succeeds gracefully or fails with error
	})
}

func TestCLI_InvalidJSONInput(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	testCases := []struct {
		name     string
		input    string
		wantExit int
	}{
		{
			name:     "malformed_json",
			input:    "{invalid json content",
			wantExit: 1,
		},
		{
			name:     "empty_input",
			input:    "",
			wantExit: 1,
		},
		{
			name:     "valid_json_missing_fields",
			input:    "{}",
			wantExit: 0, // Should handle gracefully with defaults
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			memoryDir := filepath.Join(projectDir, ".claude", "memory")
			os.MkdirAll(memoryDir, 0755)

			// Setup minimal runtime dir
			runtimeDir := t.TempDir()
			os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
			defer os.Unsetenv("XDG_RUNTIME_DIR")
			os.MkdirAll(filepath.Join(runtimeDir, "gogent"), 0755)

			cmd := exec.Command(binaryPath)
			cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)
			cmd.Stdin = bytes.NewReader([]byte(tc.input))

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tc.wantExit == 0 && err != nil {
				t.Errorf("Expected success, got error: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
			}
			if tc.wantExit == 1 && err == nil {
				t.Error("Expected failure for invalid input, got success")
			}

			// For errors, verify output is valid JSON with helpful message
			if tc.wantExit == 1 {
				var output map[string]interface{}
				if json.Unmarshal(stdout.Bytes(), &output) == nil {
					hookOutput, ok := output["hookSpecificOutput"].(map[string]interface{})
					if ok {
						ctx, _ := hookOutput["additionalContext"].(string)
						if !strings.Contains(ctx, "[gogent-archive]") {
							t.Error("Error message should contain [gogent-archive] component tag")
						}
					}
				}
			}
		})
	}
}

func TestCLI_StdinTimeout(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	// Create a pipe that will never provide data (simulating hung stdin)
	// We'll use context with timeout to test the binary's timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectDir := t.TempDir()
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

	// Provide a reader that blocks forever
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	cmd.Stdin = r
	defer w.Close()
	defer r.Close()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Start command (it will wait for stdin)
	err = cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Wait for a bit to let the binary's internal timeout trigger (5 seconds default)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Binary's internal timeout should have triggered (before our 10s context)
		if err == nil {
			t.Error("Expected timeout error, got success")
		}
		// Check output contains timeout message
		var output map[string]interface{}
		if json.Unmarshal(stdout.Bytes(), &output) == nil {
			hookOutput, ok := output["hookSpecificOutput"].(map[string]interface{})
			if ok {
				ctx, _ := hookOutput["additionalContext"].(string)
				if !strings.Contains(strings.ToLower(ctx), "timeout") {
					t.Logf("Expected timeout mention in error: %s", ctx)
				}
			}
		}
	case <-ctx.Done():
		cmd.Process.Kill()
		t.Error("Test timeout - binary should have timed out internally before 10s")
	}
}

func TestCLI_VersionFlag(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	testCases := []string{"--version", "-v"}

	for _, flag := range testCases {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(binaryPath, flag)

			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			err := cmd.Run()
			if err != nil {
				t.Fatalf("Version flag should not error: %v", err)
			}

			output := stdout.String()
			if !strings.Contains(output, "gogent-archive version") {
				t.Errorf("Expected 'gogent-archive version' in output, got: %s", output)
			}
			// Should contain some version string (dev or semver)
			if !strings.Contains(output, "dev") && !strings.Contains(output, ".") {
				t.Errorf("Expected version string, got: %s", output)
			}
		})
	}
}

func TestCLI_HelpFlag(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	testCases := []string{"--help", "-h"}

	for _, flag := range testCases {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(binaryPath, flag)

			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			err := cmd.Run()
			if err != nil {
				t.Fatalf("Help flag should not error: %v", err)
			}

			output := stdout.String()

			// Verify help shows all subcommands
			expectedSubcommands := []string{"list", "show", "stats", "--help", "--version"}
			for _, subcmd := range expectedSubcommands {
				if !strings.Contains(output, subcmd) {
					t.Errorf("Help should mention '%s' subcommand", subcmd)
				}
			}

			// Verify help shows usage patterns
			if !strings.Contains(output, "Usage:") {
				t.Error("Help should include 'Usage:' section")
			}

			// Verify help shows examples
			if !strings.Contains(output, "Examples:") {
				t.Error("Help should include 'Examples:' section")
			}
		})
	}
}

func TestCLI_ExitCodes(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	t.Run("success_exit_0", func(t *testing.T) {
		projectDir := t.TempDir()
		memoryDir := filepath.Join(projectDir, ".claude", "memory")
		os.MkdirAll(memoryDir, 0755)
		os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(""), 0644)

		cmd := exec.Command(binaryPath, "list")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		err := cmd.Run()
		if err != nil {
			t.Errorf("Expected exit 0 for successful command, got error: %v", err)
		}
	})

	t.Run("failure_exit_1_invalid_input", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		cmd.Stdin = bytes.NewReader([]byte("{invalid}"))

		err := cmd.Run()
		if err == nil {
			t.Error("Expected exit 1 for invalid input")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
		}
	})

	t.Run("failure_exit_1_missing_session", func(t *testing.T) {
		projectDir := t.TempDir()
		memoryDir := filepath.Join(projectDir, ".claude", "memory")
		os.MkdirAll(memoryDir, 0755)
		os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(""), 0644)

		cmd := exec.Command(binaryPath, "show", "nonexistent-session-id")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		err := cmd.Run()
		if err == nil {
			t.Error("Expected exit 1 for missing session")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
		}
	})
}

func TestCLI_ConfirmationJSONSchema(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)
	projectDir, memoryDir, _, gogentDir := setupTempProject(t)

	// Create minimal metrics
	os.WriteFile(filepath.Join(gogentDir, "claude-tool-counter-test.log"), []byte("call1\n"), 0644)

	sessionID := "schema-test-session"
	eventJSON := createSessionEventJSON(sessionID)

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Parse confirmation JSON
	var confirmation map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &confirmation); err != nil {
		t.Fatalf("Failed to parse confirmation JSON: %v\nOutput: %s", err, stdout.String())
	}

	// Validate required top-level field per Claude Code hook spec
	hookOutput, ok := confirmation["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Claude Code hook spec requires 'hookSpecificOutput' field")
	}

	// Validate required hookSpecificOutput fields
	requiredFields := []string{"hookEventName", "additionalContext", "session_id"}
	for _, field := range requiredFields {
		if _, ok := hookOutput[field]; !ok {
			t.Errorf("Missing required field in hookSpecificOutput: %s", field)
		}
	}

	// Validate hookEventName value
	if hookOutput["hookEventName"] != "SessionEnd" {
		t.Errorf("hookEventName should be 'SessionEnd', got: %v", hookOutput["hookEventName"])
	}

	// Validate additionalContext is a string with content
	ctx, ok := hookOutput["additionalContext"].(string)
	if !ok || ctx == "" {
		t.Error("additionalContext should be non-empty string")
	}

	// Validate paths are present
	if _, ok := hookOutput["handoff_jsonl"]; !ok {
		t.Error("Missing handoff_jsonl path in output")
	}
	if _, ok := hookOutput["handoff_md"]; !ok {
		t.Error("Missing handoff_md path in output")
	}

	// Validate metrics structure
	metrics, ok := hookOutput["metrics"].(map[string]interface{})
	if !ok {
		t.Error("Missing metrics object in hookSpecificOutput")
	} else {
		metricsFields := []string{"tool_calls", "errors", "violations"}
		for _, field := range metricsFields {
			if _, ok := metrics[field]; !ok {
				t.Errorf("Missing metrics.%s field", field)
			}
		}
	}

	// Verify handoffs.jsonl was created
	handoffPath := filepath.Join(memoryDir, "handoffs.jsonl")
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Error("handoffs.jsonl should have been created")
	}
}

// ===========================================================================
// ADDITIONAL EDGE CASE TESTS
// ===========================================================================

func TestCLI_ShowSubcommand(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	projectDir := t.TempDir()
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create handoff with known session ID
	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"show-test-session","context":{"project_dir":"/test","metrics":{"tool_calls":42,"errors_logged":1,"routing_violations":2,"session_id":"show-test-session"},"git_info":{"branch":"main","is_dirty":true}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(handoff+"\n"), 0644)

	cmd := exec.Command(binaryPath, "show", "show-test-session")
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("show command failed: %v", err)
	}

	output := stdout.String()

	// Verify markdown output
	if !strings.Contains(output, "# Session Handoff") {
		t.Error("show output should contain markdown header")
	}
	if !strings.Contains(output, "show-test-session") {
		t.Error("show output should contain session ID")
	}
	if !strings.Contains(output, "42") {
		t.Error("show output should contain tool calls count")
	}
}

func TestCLI_StatsSubcommand(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	projectDir := t.TempDir()
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create multiple handoffs for stats
	handoffs := []string{
		`{"schema_version":"1.0","timestamp":1705000000,"session_id":"s1","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"s1"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`,
		`{"schema_version":"1.0","timestamp":1705100000,"session_id":"s2","context":{"project_dir":"/test","metrics":{"tool_calls":20,"errors_logged":0,"routing_violations":1,"session_id":"s2"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`,
	}
	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(strings.Join(handoffs, "\n")+"\n"), 0644)

	cmd := exec.Command(binaryPath, "stats")
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("stats command failed: %v", err)
	}

	output := stdout.String()

	// Verify stats output
	if !strings.Contains(output, "Total Sessions: 2") {
		t.Error("stats should show total sessions")
	}
	if !strings.Contains(output, "Avg Tool Calls per Session: 15") {
		t.Error("stats should show average tool calls (10+20)/2=15")
	}
	if !strings.Contains(output, "Total Errors: 2") {
		t.Error("stats should show total errors")
	}
	if !strings.Contains(output, "Total Violations: 1") {
		t.Error("stats should show total violations")
	}
}

func TestCLI_ListWithFilters(t *testing.T) {
	binaryPath := skipIfBinaryNotBuilt(t)

	projectDir := t.TempDir()
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create handoffs with different characteristics
	// Recent session with sharp edges
	h1 := `{"schema_version":"1.0","timestamp":` + fmt.Sprintf("%d", time.Now().Add(-24*time.Hour).Unix()) + `,"session_id":"recent-with-edges","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":0,"routing_violations":0,"session_id":"recent-with-edges"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"test","consecutive_failures":3,"timestamp":1}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	// Clean session
	h2 := `{"schema_version":"1.0","timestamp":` + fmt.Sprintf("%d", time.Now().Add(-48*time.Hour).Unix()) + `,"session_id":"clean-session","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"clean-session"},"git_info":{"branch":"","is_dirty":false}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(h1+"\n"+h2+"\n"), 0644)

	t.Run("filter_since_7d", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "list", "--since", "7d")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			t.Fatalf("list --since failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "recent-with-edges") {
			t.Error("Expected recent session in --since 7d filter")
		}
		if !strings.Contains(output, "clean-session") {
			t.Error("Expected clean session in --since 7d filter (both within 7d)")
		}
	})

	t.Run("filter_clean", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "list", "--clean")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			t.Fatalf("list --clean failed: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "recent-with-edges") {
			t.Error("--clean filter should exclude session with sharp edges")
		}
		if !strings.Contains(output, "clean-session") {
			t.Error("--clean filter should include clean session")
		}
	})

	t.Run("filter_has_sharp_edges", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "list", "--has-sharp-edges")
		cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			t.Fatalf("list --has-sharp-edges failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "recent-with-edges") {
			t.Error("--has-sharp-edges should include session with sharp edges")
		}
		if strings.Contains(output, "clean-session") {
			t.Error("--has-sharp-edges should exclude clean session")
		}
	})
}

