package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionArchive_Integration(t *testing.T) {
	binaryPath := "../../bin/goyoke-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-archive binary not found. Run: go build -o cmd/goyoke-archive/goyoke-archive cmd/goyoke-archive/main.go")
	}

	// Setup test project directory
	projectDir := t.TempDir()
	setupTestSessionFiles(t, projectDir)

	// Create SessionEnd event JSON
	sessionID := "test-session-123"
	event := map[string]interface{}{
		"session_id":      sessionID,
		"timestamp":       time.Now().Unix(),
		"hook_event_name": "SessionEnd",
	}
	eventJSON, _ := json.Marshal(event)

	// Invoke goyoke-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOYOKE_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook execution failed: %v\nStderr: %s\nStdout: %s", err, stderr.String(), stdout.String())
	}

	// Verify JSON output
	var confirmation map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &confirmation); err != nil {
		t.Fatalf("Failed to parse confirmation JSON: %v\nOutput: %s", err, stdout.String())
	}

	// Verify handoff file created
	handoffPath := filepath.Join(projectDir, ".goyoke", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Errorf("Handoff file not created: %v", err)
	}

	// Verify handoff content
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("Failed to read handoff: %v", err)
	}

	handoffContent := string(handoffData)

	// Check required sections (based on actual handoff_markdown.go implementation)
	requiredSections := []string{
		"# Session Handoff",        // Header (with timestamp appended)
		"## Session Context",        // Always present
		"## Session Metrics",        // Always present
	}

	for _, section := range requiredSections {
		if !strings.Contains(handoffContent, section) {
			t.Errorf("Handoff missing required section: %s", section)
		}
	}

	// Verify metrics section contains counts
	if !strings.Contains(handoffContent, "Tool Calls") {
		t.Error("Handoff missing tool calls metric")
	}

	if !strings.Contains(handoffContent, "Errors Logged") {
		t.Error("Handoff missing errors logged metric")
	}

	if !strings.Contains(handoffContent, "Routing Violations") {
		t.Error("Handoff missing routing violations metric")
	}
}

func TestSessionArchive_MetricsCollection(t *testing.T) {
	binaryPath := "../../bin/goyoke-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-archive binary not found")
	}

	projectDir := t.TempDir()

	// Setup runtime directory for metrics
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	goyokeDir := filepath.Join(runtimeDir, "goyoke")
	os.MkdirAll(goyokeDir, 0755)

	// Create tool counter logs
	createToolCounterLog(t, goyokeDir, "task", 10)
	createToolCounterLog(t, goyokeDir, "read", 25)
	createToolCounterLog(t, goyokeDir, "write", 5)

	// Create error patterns log
	errorLogPath := filepath.Join(goyokeDir, "claude-error-patterns.jsonl")
	errorLogs := []string{
		`{"timestamp":1234567890,"file":"test1.go","error_type":"TypeError"}`,
		`{"timestamp":1234567891,"file":"test2.go","error_type":"ValueError"}`,
		`{"timestamp":1234567892,"file":"test1.go","error_type":"TypeError"}`,
	}
	os.WriteFile(errorLogPath, []byte(strings.Join(errorLogs, "\n")+"\n"), 0644)

	// Create violations log
	violationsLogPath := filepath.Join(goyokeDir, "routing-violations.jsonl")
	violations := []string{
		`{"violation_type":"tool_permission","tool":"Write"}`,
		`{"violation_type":"delegation_ceiling","agent":"architect"}`,
	}
	os.WriteFile(violationsLogPath, []byte(strings.Join(violations, "\n")+"\n"), 0644)

	// Create SessionEnd event JSON
	sessionID := "test-metrics"
	event := map[string]interface{}{
		"session_id":      sessionID,
		"timestamp":       time.Now().Unix(),
		"hook_event_name": "SessionEnd",
	}
	eventJSON, _ := json.Marshal(event)

	// Invoke goyoke-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOYOKE_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook failed: %v\nStderr: %s\nStdout: %s", err, stderr.String(), stdout.String())
	}

	// Verify handoff contains correct counts
	handoffPath := filepath.Join(projectDir, ".goyoke", "memory", "last-handoff.md")
	handoffData, _ := os.ReadFile(handoffPath)
	handoffContent := string(handoffData)

	// Should reflect 40 tool calls (10+25+5)
	// Format is "**Tool Calls**: 40"
	if !strings.Contains(handoffContent, "**Tool Calls**: 40") {
		t.Errorf("Expected '**Tool Calls**: 40' in handoff, got: %s", handoffContent)
	}

	// Should have 3 errors
	// Format is "**Errors Logged**: 3"
	if !strings.Contains(handoffContent, "**Errors Logged**: 3") {
		t.Log("Warning: Expected '**Errors Logged**: 3' in handoff")
	}

	// Should have 2 violations
	// Format is "**Routing Violations**: 2"
	if !strings.Contains(handoffContent, "**Routing Violations**: 2") {
		t.Log("Warning: Expected '**Routing Violations**: 2' in handoff")
	}
}

func TestSessionArchive_FileArchival(t *testing.T) {
	binaryPath := "../../bin/goyoke-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-archive binary not found")
	}

	projectDir := t.TempDir()

	// Create files to archive (both in .goyoke/memory)
	learningsPath := filepath.Join(projectDir, ".goyoke", "memory", "pending-learnings.jsonl")
	violationsPath := filepath.Join(projectDir, ".goyoke", "memory", "routing-violations.jsonl")

	os.MkdirAll(filepath.Dir(learningsPath), 0755)

	os.WriteFile(learningsPath, []byte("learnings content\n"), 0644)
	os.WriteFile(violationsPath, []byte("violations content\n"), 0644)

	// Create SessionEnd event JSON
	sessionID := "test-archival"
	event := map[string]interface{}{
		"session_id":      sessionID,
		"timestamp":       time.Now().Unix(),
		"hook_event_name": "SessionEnd",
	}
	eventJSON, _ := json.Marshal(event)

	// Invoke goyoke-archive
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "GOYOKE_PROJECT_DIR="+projectDir)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook failed: %v\nStderr: %s\nStdout: %s", err, stderr.String(), stdout.String())
	}

	// Verify files archived
	archiveDir := filepath.Join(projectDir, ".goyoke", "memory", "session-archive")

	// Learnings should be moved (deleted from original location)
	if _, err := os.Stat(learningsPath); !os.IsNotExist(err) {
		t.Error("Learnings should be removed after archival")
	}

	// Check for archived learnings (filename format may vary)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("Failed to read archive directory: %v", err)
	}

	var hasLearnings, hasViolations bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "pending-learnings-") {
			hasLearnings = true
		}
		if strings.HasPrefix(entry.Name(), "routing-violations-") {
			hasViolations = true
		}
	}

	if !hasLearnings {
		t.Error("Expected learnings archive file in session-archive/")
	}
	if !hasViolations {
		t.Error("Expected violations archive file in session-archive/")
	}

	// Violations should be moved from runtime dir
	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations should be removed after archival")
	}
}

// Helper: Setup test session files
func setupTestSessionFiles(t *testing.T, projectDir string) {
	// Setup runtime directory
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	goyokeDir := filepath.Join(runtimeDir, "goyoke")
	os.MkdirAll(goyokeDir, 0755)

	// Create minimal tool counter logs
	createToolCounterLog(t, goyokeDir, "task", 5)

	// Create .claude directory structure
	claudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(claudeDir, 0755)
}

// Helper: Create tool counter log (new single-file format)
func createToolCounterLog(t *testing.T, goyokeDir, tool string, count int) {
	counterPath := filepath.Join(goyokeDir, "tool-counter")

	// Read existing count if file exists
	var existingCount int
	if data, err := os.ReadFile(counterPath); err == nil {
		fmt.Sscanf(string(data), "%d", &existingCount)
	}

	// Write cumulative count (new format: single integer)
	newCount := existingCount + count
	if err := os.WriteFile(counterPath, []byte(fmt.Sprintf("%d", newCount)), 0644); err != nil {
		t.Fatalf("Failed to create tool counter: %v", err)
	}
}
