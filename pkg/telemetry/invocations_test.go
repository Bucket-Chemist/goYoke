package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogInvocation_GlobalWrite(t *testing.T) {
	// Use temp directory to avoid polluting real cache
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:       "test-session",
		InvocationID:    "inv-001",
		Agent:           "python-pro",
		Model:           "sonnet",
		Tier:            "sonnet",
		DurationMs:      1500,
		InputTokens:     1000,
		OutputTokens:    500,
		Success:         true,
		TaskDescription: "Implement feature X",
		ToolsUsed:       []string{"Read", "Write", "Edit"},
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created
	globalPath := GetInvocationsLogPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Errorf("Expected global log file to exist at %s", globalPath)
	}

	// Verify content
	data, _ := os.ReadFile(globalPath)
	if !strings.Contains(string(data), "python-pro") {
		t.Error("Expected log to contain agent name")
	}
	if !strings.Contains(string(data), "test-session") {
		t.Error("Expected log to contain session ID")
	}
}

func TestLogInvocation_DualWrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	projectDir := filepath.Join(tmpDir, "test-project")

	inv := &AgentInvocation{
		SessionID:    "dual-write-test",
		InvocationID: "inv-002",
		Agent:        "orchestrator",
		Model:        "sonnet",
		Tier:         "sonnet",
		DurationMs:   2000,
		InputTokens:  2000,
		OutputTokens: 1000,
		Success:      true,
		ToolsUsed:    []string{},
	}

	err := LogInvocation(inv, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify both files exist
	globalPath := GetInvocationsLogPath()
	projectPath := GetProjectInvocationsLogPath(projectDir)

	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Expected global log to exist")
	}
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Expected project log to exist")
	}

	// Verify project directory was populated
	data, _ := os.ReadFile(projectPath)
	var logged AgentInvocation
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		t.Fatal("Expected at least one line in project log")
	}
	json.Unmarshal([]byte(lines[0]), &logged)
	if logged.ProjectDir != projectDir {
		t.Errorf("Expected ProjectDir '%s', got '%s'", projectDir, logged.ProjectDir)
	}
}

func TestLogInvocation_TimestampAutoPopulated(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:    "timestamp-test",
		InvocationID: "inv-003",
		Agent:        "haiku-scout",
		Model:        "haiku",
		Tier:         "haiku",
		Success:      true,
		ToolsUsed:    []string{},
	}

	// Timestamp should be empty before logging
	if inv.Timestamp != "" {
		t.Error("Expected empty timestamp before logging")
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Timestamp should be populated after logging
	if inv.Timestamp == "" {
		t.Error("Expected timestamp to be populated after logging")
	}

	// Verify RFC3339 format
	if !strings.Contains(inv.Timestamp, "T") || !strings.Contains(inv.Timestamp, ":") {
		t.Errorf("Expected RFC3339 format, got: %s", inv.Timestamp)
	}
}

func TestLogInvocation_FailureInvocation(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:       "failure-test",
		InvocationID:    "inv-004",
		Agent:           "python-pro",
		Model:           "sonnet",
		Tier:            "sonnet",
		DurationMs:      500,
		InputTokens:     500,
		OutputTokens:    100,
		Success:         false,
		ErrorType:       "tool_permission",
		TaskDescription: "Attempted forbidden operation",
		ToolsUsed:       []string{"Write"},
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify error type was logged
	data, _ := os.ReadFile(GetInvocationsLogPath())
	if !strings.Contains(string(data), "tool_permission") {
		t.Error("Expected log to contain error_type")
	}
	if !strings.Contains(string(data), `"success":false`) {
		t.Error("Expected log to show success:false")
	}
}

func TestLogInvocation_ThinkingTokens(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:      "thinking-test",
		InvocationID:   "inv-005",
		Agent:          "architect",
		Model:          "sonnet",
		Tier:           "sonnet",
		DurationMs:     3000,
		InputTokens:    1500,
		OutputTokens:   800,
		ThinkingTokens: 2000,
		Success:        true,
		ToolsUsed:      []string{},
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify thinking tokens were logged
	data, _ := os.ReadFile(GetInvocationsLogPath())
	if !strings.Contains(string(data), `"thinking_tokens":2000`) {
		t.Error("Expected log to contain thinking_tokens")
	}
}

func TestLogInvocation_ParentTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:       "parent-task-test",
		InvocationID:    "inv-006",
		Agent:           "python-pro",
		Model:           "sonnet",
		Tier:            "sonnet",
		Success:         true,
		TaskDescription: "Sub-task implementation",
		ParentTaskID:    "parent-task-001",
		ToolsUsed:       []string{"Edit"},
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify parent task ID was logged
	data, _ := os.ReadFile(GetInvocationsLogPath())
	if !strings.Contains(string(data), `"parent_task_id":"parent-task-001"`) {
		t.Error("Expected log to contain parent_task_id")
	}
}

func TestLogInvocation_ProjectWriteFailure_GracefulDegradation(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	// Create a non-writable project directory
	projectDir := filepath.Join(tmpDir, "readonly-project")
	claudeMemDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(claudeMemDir, 0755)

	// Create the invocations file as a directory (will cause write failure)
	invocationsPath := filepath.Join(claudeMemDir, "agent-invocations.jsonl")
	os.Mkdir(invocationsPath, 0755)

	inv := &AgentInvocation{
		SessionID:    "graceful-degradation-test",
		InvocationID: "inv-007",
		Agent:        "codebase-search",
		Model:        "haiku",
		Tier:         "haiku",
		Success:      true,
		ToolsUsed:    []string{"Glob", "Grep"},
	}

	// Should NOT return error - global write should succeed
	err := LogInvocation(inv, projectDir)
	if err != nil {
		t.Errorf("Expected no error (graceful degradation), got: %v", err)
	}

	// Verify global log was written
	globalPath := GetInvocationsLogPath()
	data, _ := os.ReadFile(globalPath)
	if !strings.Contains(string(data), "codebase-search") {
		t.Error("Expected global log to contain invocation despite project write failure")
	}
}

func TestLoadInvocations_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "invocations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","session_id":"s1","invocation_id":"i1","agent":"python-pro","model":"sonnet","tier":"sonnet","duration_ms":1000,"input_tokens":500,"output_tokens":250,"success":true,"tools_used":[]}
{"timestamp":"2026-01-22T10:01:00Z","session_id":"s1","invocation_id":"i2","agent":"orchestrator","model":"sonnet","tier":"sonnet","duration_ms":2000,"input_tokens":1000,"output_tokens":500,"success":true,"tools_used":[]}`

	os.WriteFile(logPath, []byte(content), 0644)

	invocations, err := LoadInvocations(logPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(invocations) != 2 {
		t.Errorf("Expected 2 invocations, got: %d", len(invocations))
	}

	if invocations[0].Agent != "python-pro" {
		t.Errorf("Expected first agent 'python-pro', got: %s", invocations[0].Agent)
	}
	if invocations[1].Agent != "orchestrator" {
		t.Errorf("Expected second agent 'orchestrator', got: %s", invocations[1].Agent)
	}
}

func TestLoadInvocations_MissingFile(t *testing.T) {
	invocations, err := LoadInvocations("/nonexistent/path.jsonl")
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(invocations) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d", len(invocations))
	}
}

func TestLoadInvocations_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "invocations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","session_id":"s1","invocation_id":"i1","agent":"valid","success":true,"tools_used":[]}
invalid json line
{"timestamp":"2026-01-22T10:02:00Z","session_id":"s1","invocation_id":"i2","agent":"also-valid","success":true,"tools_used":[]}`

	os.WriteFile(logPath, []byte(content), 0644)

	invocations, err := LoadInvocations(logPath)
	if err != nil {
		t.Fatalf("Expected no error (malformed skipped), got: %v", err)
	}

	if len(invocations) != 2 {
		t.Errorf("Expected 2 valid invocations (skipping malformed), got: %d", len(invocations))
	}
}

func TestLoadInvocations_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "invocations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","session_id":"s1","invocation_id":"i1","agent":"valid1","success":true,"tools_used":[]}


{"timestamp":"2026-01-22T10:02:00Z","session_id":"s1","invocation_id":"i2","agent":"valid2","success":true,"tools_used":[]}
`

	os.WriteFile(logPath, []byte(content), 0644)

	invocations, err := LoadInvocations(logPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(invocations) != 2 {
		t.Errorf("Expected 2 invocations (empty lines skipped), got: %d", len(invocations))
	}
}

func TestLoadInvocations_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "empty.jsonl")

	os.WriteFile(logPath, []byte(""), 0644)

	invocations, err := LoadInvocations(logPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(invocations) != 0 {
		t.Errorf("Expected 0 invocations for empty file, got: %d", len(invocations))
	}
}

func TestGetInvocationsLogPath_XDGCompliance(t *testing.T) {
	// Test with XDG_RUNTIME_DIR set (highest priority)
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	path := GetInvocationsLogPath()
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("Expected path to start with XDG_RUNTIME_DIR, got: %s", path)
	}
	if !strings.HasSuffix(path, "agent-invocations.jsonl") {
		t.Errorf("Expected path to end with agent-invocations.jsonl, got: %s", path)
	}
}

func TestGetInvocationsLogPath_XDGCacheHome(t *testing.T) {
	// Test with XDG_CACHE_HOME set (second priority)
	tmpDir := t.TempDir()
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	path := GetInvocationsLogPath()
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("Expected path to start with XDG_CACHE_HOME, got: %s", path)
	}
	if !strings.Contains(path, "gogent") {
		t.Errorf("Expected path to contain 'gogent', got: %s", path)
	}
}

func TestGetProjectInvocationsLogPath(t *testing.T) {
	projectDir := "/home/user/my-project"
	path := GetProjectInvocationsLogPath(projectDir)

	expectedPath := filepath.Join(projectDir, ".claude", "memory", "agent-invocations.jsonl")
	if path != expectedPath {
		t.Errorf("Expected path '%s', got: '%s'", expectedPath, path)
	}
}

func TestLogInvocation_DirectoryAutoCreated(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	// Use a nested project directory that doesn't exist
	projectDir := filepath.Join(tmpDir, "deeply", "nested", "project")

	inv := &AgentInvocation{
		SessionID:    "dir-creation-test",
		InvocationID: "inv-008",
		Agent:        "tech-docs-writer",
		Model:        "haiku",
		Tier:         "haiku_thinking",
		Success:      true,
		ToolsUsed:    []string{"Write"},
	}

	err := LogInvocation(inv, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify directories were created
	projectPath := GetProjectInvocationsLogPath(projectDir)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Expected project log directory to be auto-created")
	}
}

func TestLogInvocation_MultipleInvocations(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	invocations := []*AgentInvocation{
		{SessionID: "multi-test", InvocationID: "i1", Agent: "agent1", Success: true, ToolsUsed: []string{}},
		{SessionID: "multi-test", InvocationID: "i2", Agent: "agent2", Success: true, ToolsUsed: []string{}},
		{SessionID: "multi-test", InvocationID: "i3", Agent: "agent3", Success: false, ErrorType: "timeout", ToolsUsed: []string{}},
	}

	for _, inv := range invocations {
		if err := LogInvocation(inv, ""); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	}

	// Load and verify all were written
	loaded, err := LoadInvocations(GetInvocationsLogPath())
	if err != nil {
		t.Fatalf("Failed to load invocations: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 invocations, got: %d", len(loaded))
	}

	// Verify order (oldest first)
	if loaded[0].Agent != "agent1" {
		t.Errorf("Expected first agent 'agent1', got: %s", loaded[0].Agent)
	}
	if loaded[2].ErrorType != "timeout" {
		t.Errorf("Expected third invocation to have error_type 'timeout', got: %s", loaded[2].ErrorType)
	}
}

func TestLogInvocation_EmptyToolsUsed(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:    "empty-tools-test",
		InvocationID: "inv-009",
		Agent:        "haiku-scout",
		Model:        "haiku",
		Tier:         "haiku",
		Success:      true,
		ToolsUsed:    []string{},
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify tools_used is serialized as empty array
	data, _ := os.ReadFile(GetInvocationsLogPath())
	if !strings.Contains(string(data), `"tools_used":[]`) {
		t.Error("Expected tools_used to be serialized as empty array")
	}
}

func TestLogInvocation_NilToolsUsed(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:    "nil-tools-test",
		InvocationID: "inv-010",
		Agent:        "code-reviewer",
		Model:        "haiku",
		Tier:         "haiku_thinking",
		Success:      true,
		ToolsUsed:    nil,
	}

	err := LogInvocation(inv, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify nil tools_used is handled (serialized as null)
	data, _ := os.ReadFile(GetInvocationsLogPath())
	if !strings.Contains(string(data), `"tools_used":null`) {
		t.Error("Expected nil tools_used to be serialized as null")
	}
}

func TestAgentInvocation_JSONRoundtrip(t *testing.T) {
	original := AgentInvocation{
		Timestamp:       "2026-01-22T10:00:00Z",
		SessionID:       "roundtrip-test",
		InvocationID:    "inv-rt",
		Agent:           "python-pro",
		Model:           "sonnet",
		Tier:            "sonnet",
		DurationMs:      1500,
		InputTokens:     1000,
		OutputTokens:    500,
		ThinkingTokens:  2000,
		Success:         true,
		TaskDescription: "Test task",
		ParentTaskID:    "parent-123",
		ToolsUsed:       []string{"Read", "Write"},
		ProjectDir:      "/test/project",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var loaded AgentInvocation
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify all fields round-trip correctly
	if loaded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: %v != %v", loaded.Timestamp, original.Timestamp)
	}
	if loaded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: %v != %v", loaded.SessionID, original.SessionID)
	}
	if loaded.InvocationID != original.InvocationID {
		t.Errorf("InvocationID mismatch: %v != %v", loaded.InvocationID, original.InvocationID)
	}
	if loaded.Agent != original.Agent {
		t.Errorf("Agent mismatch: %v != %v", loaded.Agent, original.Agent)
	}
	if loaded.Model != original.Model {
		t.Errorf("Model mismatch: %v != %v", loaded.Model, original.Model)
	}
	if loaded.Tier != original.Tier {
		t.Errorf("Tier mismatch: %v != %v", loaded.Tier, original.Tier)
	}
	if loaded.DurationMs != original.DurationMs {
		t.Errorf("DurationMs mismatch: %v != %v", loaded.DurationMs, original.DurationMs)
	}
	if loaded.InputTokens != original.InputTokens {
		t.Errorf("InputTokens mismatch: %v != %v", loaded.InputTokens, original.InputTokens)
	}
	if loaded.OutputTokens != original.OutputTokens {
		t.Errorf("OutputTokens mismatch: %v != %v", loaded.OutputTokens, original.OutputTokens)
	}
	if loaded.ThinkingTokens != original.ThinkingTokens {
		t.Errorf("ThinkingTokens mismatch: %v != %v", loaded.ThinkingTokens, original.ThinkingTokens)
	}
	if loaded.Success != original.Success {
		t.Errorf("Success mismatch: %v != %v", loaded.Success, original.Success)
	}
	if loaded.TaskDescription != original.TaskDescription {
		t.Errorf("TaskDescription mismatch: %v != %v", loaded.TaskDescription, original.TaskDescription)
	}
	if loaded.ParentTaskID != original.ParentTaskID {
		t.Errorf("ParentTaskID mismatch: %v != %v", loaded.ParentTaskID, original.ParentTaskID)
	}
	if loaded.ProjectDir != original.ProjectDir {
		t.Errorf("ProjectDir mismatch: %v != %v", loaded.ProjectDir, original.ProjectDir)
	}
	if len(loaded.ToolsUsed) != len(original.ToolsUsed) {
		t.Errorf("ToolsUsed length mismatch: %v != %v", len(loaded.ToolsUsed), len(original.ToolsUsed))
	}
}

func TestAgentInvocation_OmitEmptyFields(t *testing.T) {
	// Test that optional fields are omitted when empty
	inv := AgentInvocation{
		Timestamp:    "2026-01-22T10:00:00Z",
		SessionID:    "omit-test",
		InvocationID: "inv-omit",
		Agent:        "haiku-scout",
		Model:        "haiku",
		Tier:         "haiku",
		Success:      true,
		ToolsUsed:    []string{},
		// These should be omitted: ThinkingTokens, ErrorType, ParentTaskID, ProjectDir
	}

	data, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Verify omitempty fields are NOT present
	if strings.Contains(jsonStr, "thinking_tokens") {
		t.Error("Expected thinking_tokens to be omitted when 0")
	}
	if strings.Contains(jsonStr, "error_type") {
		t.Error("Expected error_type to be omitted when empty")
	}
	if strings.Contains(jsonStr, "parent_task_id") {
		t.Error("Expected parent_task_id to be omitted when empty")
	}
	if strings.Contains(jsonStr, "project_dir") {
		t.Error("Expected project_dir to be omitted when empty")
	}
}

func TestLogInvocation_GlobalWriteFailure(t *testing.T) {
	// Create a directory where we can't write
	tmpDir := t.TempDir()

	// Make the gogent directory a file instead of directory (causes mkdir to fail)
	gogentPath := filepath.Join(tmpDir, "gogent")
	os.WriteFile(gogentPath, []byte("not a directory"), 0644)

	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	inv := &AgentInvocation{
		SessionID:    "global-failure-test",
		InvocationID: "inv-fail",
		Agent:        "test-agent",
		Success:      true,
		ToolsUsed:    []string{},
	}

	err := LogInvocation(inv, "")
	if err == nil {
		t.Error("Expected error when global write fails")
	}
	if !strings.Contains(err.Error(), "[invocations] Failed to write global log") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}
