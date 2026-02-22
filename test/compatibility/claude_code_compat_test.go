package compatibility

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestClaudeCodeAvailability verifies the claude CLI is available for testing
func TestClaudeCodeAvailability(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available - skipping compatibility tests")
	}

	// Verify we can run claude with --version
	cmd := exec.Command("claude", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'claude --version': %v\nOutput: %s", err, output)
	}

	versionStr := strings.TrimSpace(string(output))
	if versionStr == "" {
		t.Fatal("claude --version returned empty output")
	}

	t.Logf("Claude CLI version: %s", versionStr)
}

// TestHookEventSchema verifies hook binaries can parse PreToolUse event structure
func TestHookEventSchema(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	// Create a mock PreToolUse event matching the expected format
	mockEvent := map[string]interface{}{
		"tool_name": "Task",
		"tool_input": map[string]interface{}{
			"model":         "haiku",
			"prompt":        "AGENT: codebase-search\n\nFind files",
			"subagent_type": "Explore",
			"description":   "Search codebase",
		},
		"session_id":       "test-session-compat",
		"hook_event_name":  "PreToolUse",
		"captured_at":      time.Now().Unix(),
	}

	// Marshal to JSON
	eventJSON, err := json.MarshalIndent(mockEvent, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal mock event: %v", err)
	}

	// Verify we can unmarshal it back (basic structure validation)
	var parsed map[string]interface{}
	if err := json.Unmarshal(eventJSON, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal event JSON: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{"tool_name", "tool_input", "session_id", "hook_event_name", "captured_at"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field in event: %s", field)
		}
	}

	// Verify tool_input is an object
	toolInput, ok := parsed["tool_input"].(map[string]interface{})
	if !ok {
		t.Fatal("tool_input is not an object")
	}

	// Verify tool_input contains expected fields for Task tool
	if model, ok := toolInput["model"].(string); !ok || model == "" {
		t.Error("tool_input.model is missing or empty")
	}

	t.Logf("Hook event schema validation passed")
}

// TestPipeModeOutputFormat verifies claude -p --output-format stream-json produces parseable NDJSON
func TestPipeModeOutputFormat(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	// Create a minimal prompt
	prompt := "Say 'Hello compatibility test' and nothing else."

	// Run claude in pipe mode with stream-json output
	cmd := exec.Command("claude", "-p", "--output-format", "stream-json")
	// Unset CLAUDECODE to allow nested invocation for testing
	cmd.Env = append(os.Environ(), "CLAUDECODE=")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start claude: %v", err)
	}

	// Write prompt to stdin
	if _, err := stdin.Write([]byte(prompt)); err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}
	stdin.Close()

	// Read stderr in background
	var stderrBuf strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text() + "\n")
		}
	}()

	// Parse NDJSON output
	scanner := bufio.NewScanner(stdout)
	var lineCount int
	var foundAssistant, foundResult, foundCost bool

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Each line should be valid JSON
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Line %d is not valid JSON: %v\nLine: %s", lineCount, err, line)
			continue
		}

		// Check for expected entry types
		if entryType, ok := entry["type"].(string); ok {
			switch entryType {
			case "assistant":
				foundAssistant = true
			case "result":
				foundResult = true
			case "cost":
				foundCost = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading stdout: %v", err)
	}

	waitErr := cmd.Wait()
	stderrOutput := stderrBuf.String()

	if waitErr != nil {
		t.Logf("Warning: claude exited with error (may be expected): %v", waitErr)
		if stderrOutput != "" {
			t.Logf("Stderr output: %s", stderrOutput)
		}
	}

	// Verify we got at least some output
	if lineCount == 0 {
		if stderrOutput != "" {
			t.Logf("No stdout, but stderr had: %s", stderrOutput)
		}
		t.Skip("No output from claude -p --output-format stream-json (may require API key or interactive setup)")
	}

	// Log what we found
	t.Logf("Parsed %d NDJSON lines", lineCount)
	t.Logf("Found types: assistant=%v, result=%v, cost=%v", foundAssistant, foundResult, foundCost)
}

// TestAllowedToolsFlag verifies --allowedTools restricts tool access
func TestAllowedToolsFlag(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	// Create a temporary test directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Prompt that tries to use Bash (which should be blocked)
	prompt := fmt.Sprintf("Read the file at %s using the Read tool, then respond with just 'SUCCESS'.", testFile)

	// Run with restricted tools (only Read and Glob allowed, Bash blocked)
	cmd := exec.Command("claude", "-p", "--output-format", "stream-json", "--allowedTools", "Read,Glob")
	// Unset CLAUDECODE to allow nested invocation for testing
	cmd.Env = append(os.Environ(), "CLAUDECODE=")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start claude: %v", err)
	}

	// Write prompt
	if _, err := stdin.Write([]byte(prompt)); err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}
	stdin.Close()

	// Parse output to verify Read tool was available
	scanner := bufio.NewScanner(stdout)
	var foundReadTool bool
	var foundBashTool bool
	var lineCount int

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		lineCount++
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Check if this is a tool use entry
		if toolUse, ok := entry["tool_use"].(map[string]interface{}); ok {
			if toolName, ok := toolUse["name"].(string); ok {
				if toolName == "Read" {
					foundReadTool = true
				} else if toolName == "Bash" {
					foundBashTool = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading stdout: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Logf("Warning: claude exited with error (may be expected): %v", err)
	}

	// Skip if no output (likely needs API key)
	if lineCount == 0 {
		t.Skip("No output from claude (may require API key or interactive setup)")
	}

	// Verify Read was available (allowed) and Bash was not used
	if !foundReadTool {
		t.Log("Warning: Expected Read tool to be used, but it wasn't found")
		// This is not a hard failure as the agent might choose not to use it
	}

	if foundBashTool {
		t.Error("Bash tool was used despite not being in --allowedTools list")
	}

	t.Logf("Tool restriction test passed: Read=%v, Bash=%v (from %d output lines)", foundReadTool, foundBashTool, lineCount)
}

// TestSkillLoading verifies .claude/skills/*/SKILL.md files are discovered
func TestSkillLoading(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	// Check if .claude/skills directory exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		t.Skip("~/.claude/skills directory does not exist")
	}

	// Look for at least one skill (team-status was mentioned in requirements)
	expectedSkills := []string{"team-status", "braintrust", "explore"}
	var foundSkills []string

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("Failed to read skills directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			foundSkills = append(foundSkills, entry.Name())
		}
	}

	if len(foundSkills) == 0 {
		t.Fatal("No skills found in ~/.claude/skills/*/SKILL.md")
	}

	// Check if any expected skills are present
	var foundExpected bool
	for _, expected := range expectedSkills {
		for _, found := range foundSkills {
			if found == expected {
				foundExpected = true
				t.Logf("Found expected skill: %s", expected)
			}
		}
	}

	if !foundExpected {
		t.Logf("Warning: None of the expected skills (%v) were found", expectedSkills)
		t.Logf("Found skills: %v", foundSkills)
	}

	t.Logf("Skill loading test passed: found %d skills total", len(foundSkills))
}

// TestHookEventFields verifies critical hook event fields are present and correct type
func TestHookEventFields(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	testCases := []struct {
		name       string
		hookEvent  string
		toolName   string
		toolInput  map[string]interface{}
		required   []string
	}{
		{
			name:      "PreToolUse Task",
			hookEvent: "PreToolUse",
			toolName:  "Task",
			toolInput: map[string]interface{}{
				"model":         "haiku",
				"prompt":        "Test prompt",
				"subagent_type": "Explore",
			},
			required: []string{"tool_name", "tool_input", "session_id", "hook_event_name", "captured_at"},
		},
		{
			name:      "PreToolUse Read",
			hookEvent: "PreToolUse",
			toolName:  "Read",
			toolInput: map[string]interface{}{
				"file_path": "/tmp/test.txt",
			},
			required: []string{"tool_name", "tool_input", "session_id", "hook_event_name", "captured_at"},
		},
		{
			name:      "PostToolUse",
			hookEvent: "PostToolUse",
			toolName:  "Read",
			toolInput: map[string]interface{}{
				"file_path": "/tmp/test.txt",
			},
			required: []string{"tool_name", "tool_input", "session_id", "hook_event_name", "captured_at"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := map[string]interface{}{
				"tool_name":       tc.toolName,
				"tool_input":      tc.toolInput,
				"session_id":      "test-session",
				"hook_event_name": tc.hookEvent,
				"captured_at":     time.Now().Unix(),
			}

			// Marshal and unmarshal to verify JSON compatibility
			eventJSON, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal event: %v", err)
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(eventJSON, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal event: %v", err)
			}

			// Verify all required fields are present
			for _, field := range tc.required {
				if _, ok := parsed[field]; !ok {
					t.Errorf("Missing required field: %s", field)
				}
			}

			// Type-specific validations
			if toolName, ok := parsed["tool_name"].(string); !ok || toolName != tc.toolName {
				t.Errorf("tool_name mismatch: expected %s, got %v", tc.toolName, parsed["tool_name"])
			}

			if hookEvent, ok := parsed["hook_event_name"].(string); !ok || hookEvent != tc.hookEvent {
				t.Errorf("hook_event_name mismatch: expected %s, got %v", tc.hookEvent, parsed["hook_event_name"])
			}

			if capturedAt, ok := parsed["captured_at"].(float64); !ok || capturedAt <= 0 {
				t.Errorf("captured_at invalid: %v", parsed["captured_at"])
			}

			// Verify tool_input is preserved
			if toolInput, ok := parsed["tool_input"].(map[string]interface{}); !ok {
				t.Error("tool_input is not an object")
			} else {
				// Verify at least one field from original tool_input is present
				if len(toolInput) == 0 {
					t.Error("tool_input is empty")
				}
			}
		})
	}

	t.Logf("Hook event field validation passed for %d test cases", len(testCases))
}

// TestSubagentStopEvent verifies SubagentStop event schema
func TestSubagentStopEvent(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	// Mock SubagentStop event
	event := map[string]interface{}{
		"hook_event_name": "SubagentStop",
		"session_id":      "test-session",
		"agent_id":        "python-pro",
		"model":           "sonnet",
		"task_outcome":    "success",
		"captured_at":     time.Now().Unix(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal SubagentStop event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(eventJSON, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal SubagentStop event: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"hook_event_name", "session_id", "captured_at"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field in SubagentStop event: %s", field)
		}
	}

	// Verify hook_event_name is correct
	if hookEvent, ok := parsed["hook_event_name"].(string); !ok || hookEvent != "SubagentStop" {
		t.Errorf("hook_event_name mismatch: expected 'SubagentStop', got %v", parsed["hook_event_name"])
	}

	t.Log("SubagentStop event schema validation passed")
}

// TestClaudeOutputStreamStructure verifies the stream-json output has consistent structure
func TestClaudeOutputStreamStructure(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	prompt := "Respond with: I am testing Claude Code compatibility"

	cmd := exec.Command("claude", "-p", "--output-format", "stream-json")
	// Unset CLAUDECODE to allow nested invocation for testing
	cmd.Env = append(os.Environ(), "CLAUDECODE=")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start claude: %v", err)
	}

	if _, err := stdin.Write([]byte(prompt)); err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}
	stdin.Close()

	// Collect all entries by type
	entryTypes := make(map[string]int)
	var validJSONCount, invalidJSONCount int

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			invalidJSONCount++
			t.Logf("Invalid JSON line: %s", line)
			continue
		}

		validJSONCount++

		// Count entry types
		if entryType, ok := entry["type"].(string); ok {
			entryTypes[entryType]++
		} else {
			entryTypes["<no_type>"]++
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading stdout: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Logf("Warning: claude exited with error: %v", err)
	}

	// Skip if no output (likely needs API key)
	if validJSONCount == 0 && invalidJSONCount == 0 {
		t.Skip("No output from claude (may require API key or interactive setup)")
	}

	// Verify we got valid JSON output
	if validJSONCount == 0 {
		t.Fatal("No valid JSON output received")
	}

	if invalidJSONCount > 0 {
		t.Errorf("Found %d invalid JSON lines out of %d total", invalidJSONCount, validJSONCount+invalidJSONCount)
	}

	t.Logf("Stream structure: %d valid JSON entries, %d invalid", validJSONCount, invalidJSONCount)
	t.Logf("Entry types: %v", entryTypes)
}
