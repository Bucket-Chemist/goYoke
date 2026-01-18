package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogViolation(t *testing.T) {
	// Setup temp directory for XDG_RUNTIME_DIR
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create violation
	v := &Violation{
		SessionID:     "test-session-123",
		ViolationType: "tool_permission",
		Tool:          "Write",
		Reason:        "Tier haiku cannot use Write",
		Allowed:       "Read, Glob, Grep",
	}

	// Log violation (empty projectDir for backward compatibility)
	if err := LogViolation(v, ""); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Read log file
	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file at %s: %v", logPath, err)
	}

	// Parse JSONL
	var logged Violation
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to parse logged violation: %v", err)
	}

	// Verify fields
	if logged.SessionID != "test-session-123" {
		t.Errorf("Expected session_id 'test-session-123', got: %s", logged.SessionID)
	}

	if logged.ViolationType != "tool_permission" {
		t.Errorf("Expected violation_type 'tool_permission', got: %s", logged.ViolationType)
	}

	if logged.Tool != "Write" {
		t.Errorf("Expected tool 'Write', got: %s", logged.Tool)
	}

	if logged.Reason != "Tier haiku cannot use Write" {
		t.Errorf("Expected reason about haiku tier, got: %s", logged.Reason)
	}

	if logged.Allowed != "Read, Glob, Grep" {
		t.Errorf("Expected allowed tools, got: %s", logged.Allowed)
	}

	// Verify timestamp populated
	if logged.Timestamp == "" {
		t.Error("Expected timestamp to be populated")
	}

	// Verify timestamp is valid RFC3339
	if _, err := time.Parse(time.RFC3339, logged.Timestamp); err != nil {
		t.Errorf("Timestamp not in RFC3339 format: %v", err)
	}
}

func TestLogViolation_AppendMode(t *testing.T) {
	// Setup temp directory
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log first violation
	v1 := &Violation{
		SessionID:     "session-1",
		ViolationType: "tool_permission",
		Tool:          "Write",
		Reason:        "First violation",
	}
	if err := LogViolation(v1, ""); err != nil {
		t.Fatalf("Failed to log first violation: %v", err)
	}

	// Log second violation
	v2 := &Violation{
		SessionID:     "session-2",
		ViolationType: "delegation_ceiling",
		Agent:         "architect",
		Reason:        "Second violation",
	}
	if err := LogViolation(v2, ""); err != nil {
		t.Fatalf("Failed to log second violation: %v", err)
	}

	// Read entire log file
	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Split by newlines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines, got %d", len(lines))
	}

	// Parse first line
	var logged1 Violation
	if err := json.Unmarshal([]byte(lines[0]), &logged1); err != nil {
		t.Fatalf("Failed to parse first line: %v", err)
	}
	if logged1.SessionID != "session-1" {
		t.Errorf("First line: expected session-1, got %s", logged1.SessionID)
	}

	// Parse second line
	var logged2 Violation
	if err := json.Unmarshal([]byte(lines[1]), &logged2); err != nil {
		t.Fatalf("Failed to parse second line: %v", err)
	}
	if logged2.SessionID != "session-2" {
		t.Errorf("Second line: expected session-2, got %s", logged2.SessionID)
	}
}

func TestLogViolation_CreatesLogFile(t *testing.T) {
	// Setup temp directory
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")

	// Verify log doesn't exist yet
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("Log file should not exist yet")
	}

	// Log violation
	v := &Violation{
		SessionID:     "test",
		ViolationType: "test_violation",
		Reason:        "Testing file creation",
	}
	if err := LogViolation(v, ""); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Verify log now exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file should have been created")
	}
}

func TestLogViolation_AllFields(t *testing.T) {
	// Setup temp directory
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create violation with all fields populated
	v := &Violation{
		SessionID:     "full-test",
		ViolationType: "subagent_type_mismatch",
		Agent:         "tech-docs-writer",
		Model:         "haiku",
		Tool:          "Task",
		Reason:        "Agent requires general-purpose subagent_type",
		Allowed:       "general-purpose",
		Override:      "--force-tier=sonnet",
	}

	if err := LogViolation(v, ""); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Read and verify
	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var logged Violation
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify all fields
	tests := []struct {
		name     string
		expected string
		actual   string
	}{
		{"SessionID", "full-test", logged.SessionID},
		{"ViolationType", "subagent_type_mismatch", logged.ViolationType},
		{"Agent", "tech-docs-writer", logged.Agent},
		{"Model", "haiku", logged.Model},
		{"Tool", "Task", logged.Tool},
		{"Reason", "Agent requires general-purpose subagent_type", logged.Reason},
		{"Allowed", "general-purpose", logged.Allowed},
		{"Override", "--force-tier=sonnet", logged.Override},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, tt.actual)
		}
	}

	// Verify timestamp
	if logged.Timestamp == "" {
		t.Error("Timestamp should be populated")
	}
}

func TestLogViolation_JSONLFormat(t *testing.T) {
	// Setup temp directory
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Log multiple violations
	violations := []*Violation{
		{SessionID: "s1", ViolationType: "type1", Reason: "reason1"},
		{SessionID: "s2", ViolationType: "type2", Reason: "reason2"},
		{SessionID: "s3", ViolationType: "type3", Reason: "reason3"},
	}

	for _, v := range violations {
		if err := LogViolation(v, ""); err != nil {
			t.Fatalf("Failed to log violation: %v", err)
		}
	}

	// Read file
	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	// Verify JSONL format (each line is valid JSON)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var v Violation
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i+1, err)
		}
	}

	// Verify last line ends with newline (JSONL standard)
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("JSONL file should end with newline")
	}
}

func TestLogViolation_OmitemptyFields(t *testing.T) {
	// Setup temp directory
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create violation with only required fields
	v := &Violation{
		SessionID:     "minimal",
		ViolationType: "test",
		Reason:        "Testing omitempty",
	}

	if err := LogViolation(v, ""); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Read raw JSON
	logPath := filepath.Join(tmpDir, "gogent", "routing-violations.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	// Verify optional fields are not present when empty
	jsonStr := string(data)
	if strings.Contains(jsonStr, `"agent":""`) {
		t.Error("Empty agent field should be omitted")
	}
	if strings.Contains(jsonStr, `"model":""`) {
		t.Error("Empty model field should be omitted")
	}
	if strings.Contains(jsonStr, `"tool":""`) {
		t.Error("Empty tool field should be omitted")
	}
	if strings.Contains(jsonStr, `"allowed":""`) {
		t.Error("Empty allowed field should be omitted")
	}
	if strings.Contains(jsonStr, `"override":""`) {
		t.Error("Empty override field should be omitted")
	}

	// But required fields should be present
	if !strings.Contains(jsonStr, `"session_id":"minimal"`) {
		t.Error("session_id should be present")
	}
	if !strings.Contains(jsonStr, `"violation_type":"test"`) {
		t.Error("violation_type should be present")
	}
	if !strings.Contains(jsonStr, `"reason":"Testing omitempty"`) {
		t.Error("reason should be present")
	}
	if !strings.Contains(jsonStr, `"timestamp":`) {
		t.Error("timestamp should be present")
	}
}

// Test new enhanced fields are marshaled correctly
func TestViolation_EnhancedFields(t *testing.T) {
	v := &Violation{
		SessionID:       "test-123",
		ViolationType:   "tier_mismatch",
		Agent:           "tech-docs-writer",
		File:            "docs/system-guide.md",
		CurrentTier:     "haiku",
		RequiredTier:    "haiku_thinking",
		TaskDescription: "Update system guide with new routing rules",
		HookDecision:    "block",
		ProjectDir:      "/home/user/my-project",
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify all new fields present in JSON
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	expectedFields := []string{
		"file", "current_tier", "required_tier",
		"task_description", "hook_decision", "project_dir",
	}
	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing field: %s", field)
		}
	}
}

// Test dual-write pattern
func TestLogViolation_DualWrite(t *testing.T) {
	// Setup temp directories for both logs
	tmpGlobal := t.TempDir()
	tmpProject := t.TempDir()

	// Mock config paths
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpGlobal)

	// Log violation with project directory
	v := &Violation{
		SessionID:     "test-dual",
		ViolationType: "tool_permission",
		Tool:          "Write",
		Reason:        "Test dual write",
	}

	if err := LogViolation(v, tmpProject); err != nil {
		t.Fatalf("LogViolation failed: %v", err)
	}

	// Verify global log exists
	globalPath := filepath.Join(tmpGlobal, "gogent", "routing-violations.jsonl")
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Global log not created")
	}

	// Verify project log exists
	projectPath := filepath.Join(tmpProject, ".claude", "memory", "routing-violations.jsonl")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Project log not created")
	}

	// Verify both logs contain same entry
	globalData, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("Failed to read global log: %v", err)
	}
	projectData, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("Failed to read project log: %v", err)
	}

	if string(globalData) != string(projectData) {
		t.Error("Global and project logs diverged")
	}

	// Verify ProjectDir was populated
	var logged Violation
	if err := json.Unmarshal(globalData, &logged); err != nil {
		t.Fatalf("Failed to parse logged violation: %v", err)
	}
	if logged.ProjectDir != tmpProject {
		t.Errorf("Expected ProjectDir %q, got %q", tmpProject, logged.ProjectDir)
	}
}

// Test dual-write handles project log failure gracefully
func TestLogViolation_ProjectLogFailureGraceful(t *testing.T) {
	tmpGlobal := t.TempDir()

	// Mock config paths
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpGlobal)

	// Use invalid project directory (write will fail)
	invalidProjectDir := "/dev/null/invalid"

	v := &Violation{
		SessionID:     "test-graceful",
		ViolationType: "test",
		Reason:        "Test graceful degradation",
	}

	// Should NOT return error even if project log fails
	if err := LogViolation(v, invalidProjectDir); err != nil {
		t.Errorf("LogViolation should not fail when project log fails: %v", err)
	}

	// Global log should still be written
	globalPath := filepath.Join(tmpGlobal, "gogent", "routing-violations.jsonl")
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Global log should be created even if project log fails")
	}
}
