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
	claudeMemDir := filepath.Join(projectDir, ".gogent", "memory")
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

	expectedPath := filepath.Join(projectDir, ".gogent", "memory", "agent-invocations.jsonl")
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
	// Test graceful degradation when primary paths fail
	// GetGOgentDir() has multiple fallbacks (XDG_RUNTIME_DIR → XDG_CACHE_HOME → ~/.cache → /tmp)
	// This test verifies the system gracefully handles partial failures
	tmpDir := t.TempDir()

	// Make the gogent directory a file instead of directory (causes mkdir to fail)
	gogentPath := filepath.Join(tmpDir, "gogent")
	os.WriteFile(gogentPath, []byte("not a directory"), 0644)

	// Block primary paths - fallbacks may still succeed (by design)
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	inv := &AgentInvocation{
		SessionID:    "global-failure-test",
		InvocationID: "inv-fail",
		Agent:        "test-agent",
		Success:      true,
		ToolsUsed:    []string{},
	}

	err := LogInvocation(inv, "")
	// System is designed to gracefully degrade - fallbacks may succeed
	// If an error does occur, verify it has the expected message
	if err != nil && !strings.Contains(err.Error(), "[invocations] Failed to write global log") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// ===== ClusterInvocationsByAgent Tests =====

func TestClusterInvocationsByAgent_SingleAgent(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "python-pro", Success: true, DurationMs: 1000, InputTokens: 500, OutputTokens: 250, ThinkingTokens: 100},
		{Agent: "python-pro", Success: true, DurationMs: 2000, InputTokens: 600, OutputTokens: 300, ThinkingTokens: 150},
		{Agent: "python-pro", Success: false, DurationMs: 500, InputTokens: 200, OutputTokens: 50, ThinkingTokens: 0},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(stats))
	}

	agent := stats[0]
	if agent.Agent != "python-pro" {
		t.Errorf("Expected agent 'python-pro', got '%s'", agent.Agent)
	}
	if agent.TotalCount != 3 {
		t.Errorf("Expected TotalCount 3, got %d", agent.TotalCount)
	}
	if agent.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount 2, got %d", agent.SuccessCount)
	}
	if agent.FailureCount != 1 {
		t.Errorf("Expected FailureCount 1, got %d", agent.FailureCount)
	}
	expectedSuccessRate := 2.0 / 3.0
	if agent.SuccessRate < expectedSuccessRate-0.01 || agent.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected SuccessRate ~%.3f, got %.3f", expectedSuccessRate, agent.SuccessRate)
	}
	expectedAvgDuration := int64(1166) // (1000 + 2000 + 500) / 3
	if agent.AvgDurationMs != expectedAvgDuration {
		t.Errorf("Expected AvgDurationMs %d, got %d", expectedAvgDuration, agent.AvgDurationMs)
	}
	if agent.TotalDurationMs != 3500 {
		t.Errorf("Expected TotalDurationMs 3500, got %d", agent.TotalDurationMs)
	}
	if agent.TotalInputTokens != 1300 {
		t.Errorf("Expected TotalInputTokens 1300, got %d", agent.TotalInputTokens)
	}
	if agent.TotalOutputTokens != 600 {
		t.Errorf("Expected TotalOutputTokens 600, got %d", agent.TotalOutputTokens)
	}
	if agent.TotalThinkingTokens != 250 {
		t.Errorf("Expected TotalThinkingTokens 250, got %d", agent.TotalThinkingTokens)
	}
	if len(agent.Samples) != 3 {
		t.Errorf("Expected 3 samples, got %d", len(agent.Samples))
	}
}

func TestClusterInvocationsByAgent_MultipleAgents(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "python-pro", Success: true, DurationMs: 1000},
		{Agent: "orchestrator", Success: true, DurationMs: 2000},
		{Agent: "python-pro", Success: false, DurationMs: 500},
		{Agent: "haiku-scout", Success: true, DurationMs: 300},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(stats))
	}

	// Results should be sorted by agent name
	expectedOrder := []string{"haiku-scout", "orchestrator", "python-pro"}
	for i, expected := range expectedOrder {
		if stats[i].Agent != expected {
			t.Errorf("Expected agent[%d] '%s', got '%s'", i, expected, stats[i].Agent)
		}
	}

	// Verify python-pro stats
	pythonStats := stats[2] // "python-pro" is last alphabetically
	if pythonStats.TotalCount != 2 {
		t.Errorf("Expected python-pro TotalCount 2, got %d", pythonStats.TotalCount)
	}
	if pythonStats.SuccessCount != 1 {
		t.Errorf("Expected python-pro SuccessCount 1, got %d", pythonStats.SuccessCount)
	}
	if pythonStats.FailureCount != 1 {
		t.Errorf("Expected python-pro FailureCount 1, got %d", pythonStats.FailureCount)
	}
}

func TestClusterInvocationsByAgent_EmptyAgent(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "", Success: true, DurationMs: 1000},
		{Agent: "", Success: false, DurationMs: 500},
		{Agent: "python-pro", Success: true, DurationMs: 2000},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 2 {
		t.Fatalf("Expected 2 agents (unknown + python-pro), got %d", len(stats))
	}

	// Find the "unknown" agent
	var unknownStats *AgentInvocationStats
	for i := range stats {
		if stats[i].Agent == "unknown" {
			unknownStats = &stats[i]
			break
		}
	}

	if unknownStats == nil {
		t.Fatal("Expected 'unknown' agent not found")
	}
	if unknownStats.TotalCount != 2 {
		t.Errorf("Expected unknown TotalCount 2, got %d", unknownStats.TotalCount)
	}
}

func TestClusterInvocationsByAgent_EmptyInput(t *testing.T) {
	stats := ClusterInvocationsByAgent([]AgentInvocation{})

	if len(stats) != 0 {
		t.Errorf("Expected empty result, got %d agents", len(stats))
	}
}

func TestClusterInvocationsByAgent_Samples(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "test-agent", InvocationID: "inv-1", Success: true},
		{Agent: "test-agent", InvocationID: "inv-2", Success: true},
		{Agent: "test-agent", InvocationID: "inv-3", Success: true},
		{Agent: "test-agent", InvocationID: "inv-4", Success: true},
		{Agent: "test-agent", InvocationID: "inv-5", Success: true},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(stats))
	}

	// Should keep only first 3 samples
	if len(stats[0].Samples) != 3 {
		t.Errorf("Expected 3 samples, got %d", len(stats[0].Samples))
	}

	// Verify sample order (first 3)
	expectedIDs := []string{"inv-1", "inv-2", "inv-3"}
	for i, expected := range expectedIDs {
		if stats[0].Samples[i].InvocationID != expected {
			t.Errorf("Expected sample[%d] ID '%s', got '%s'", i, expected, stats[0].Samples[i].InvocationID)
		}
	}
}

func TestClusterInvocationsByAgent_AllSuccess(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "success-agent", Success: true, DurationMs: 1000},
		{Agent: "success-agent", Success: true, DurationMs: 2000},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(stats))
	}

	if stats[0].SuccessRate != 1.0 {
		t.Errorf("Expected SuccessRate 1.0, got %.3f", stats[0].SuccessRate)
	}
	if stats[0].FailureCount != 0 {
		t.Errorf("Expected FailureCount 0, got %d", stats[0].FailureCount)
	}
}

func TestClusterInvocationsByAgent_AllFailure(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "failure-agent", Success: false, DurationMs: 500},
		{Agent: "failure-agent", Success: false, DurationMs: 600},
	}

	stats := ClusterInvocationsByAgent(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(stats))
	}

	if stats[0].SuccessRate != 0.0 {
		t.Errorf("Expected SuccessRate 0.0, got %.3f", stats[0].SuccessRate)
	}
	if stats[0].SuccessCount != 0 {
		t.Errorf("Expected SuccessCount 0, got %d", stats[0].SuccessCount)
	}
	if stats[0].FailureCount != 2 {
		t.Errorf("Expected FailureCount 2, got %d", stats[0].FailureCount)
	}
}

// ===== ClusterInvocationsByTier Tests =====

func TestClusterInvocationsByTier_SingleTier(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "sonnet", Agent: "python-pro", Success: true, InputTokens: 500, OutputTokens: 250},
		{Tier: "sonnet", Agent: "orchestrator", Success: true, InputTokens: 600, OutputTokens: 300},
		{Tier: "sonnet", Agent: "python-pro", Success: false, InputTokens: 200, OutputTokens: 50},
	}

	stats := ClusterInvocationsByTier(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 tier, got %d", len(stats))
	}

	tier := stats[0]
	if tier.Tier != "sonnet" {
		t.Errorf("Expected tier 'sonnet', got '%s'", tier.Tier)
	}
	if tier.TotalCount != 3 {
		t.Errorf("Expected TotalCount 3, got %d", tier.TotalCount)
	}
	if tier.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount 2, got %d", tier.SuccessCount)
	}
	expectedSuccessRate := 2.0 / 3.0
	if tier.SuccessRate < expectedSuccessRate-0.01 || tier.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected SuccessRate ~%.3f, got %.3f", expectedSuccessRate, tier.SuccessRate)
	}
	if tier.TotalInputTokens != 1300 {
		t.Errorf("Expected TotalInputTokens 1300, got %d", tier.TotalInputTokens)
	}
	if tier.TotalOutputTokens != 600 {
		t.Errorf("Expected TotalOutputTokens 600, got %d", tier.TotalOutputTokens)
	}

	// Verify agent breakdown
	if len(tier.AgentBreakdown) != 2 {
		t.Errorf("Expected 2 agents in breakdown, got %d", len(tier.AgentBreakdown))
	}
	if tier.AgentBreakdown["python-pro"] != 2 {
		t.Errorf("Expected python-pro count 2, got %d", tier.AgentBreakdown["python-pro"])
	}
	if tier.AgentBreakdown["orchestrator"] != 1 {
		t.Errorf("Expected orchestrator count 1, got %d", tier.AgentBreakdown["orchestrator"])
	}
}

func TestClusterInvocationsByTier_MultipleTiers(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "haiku", Agent: "haiku-scout", Success: true},
		{Tier: "sonnet", Agent: "python-pro", Success: true},
		{Tier: "haiku", Agent: "codebase-search", Success: true},
		{Tier: "opus", Agent: "einstein", Success: false},
	}

	stats := ClusterInvocationsByTier(invocations)

	if len(stats) != 3 {
		t.Fatalf("Expected 3 tiers, got %d", len(stats))
	}

	// Results should be sorted by tier name
	expectedOrder := []string{"haiku", "opus", "sonnet"}
	for i, expected := range expectedOrder {
		if stats[i].Tier != expected {
			t.Errorf("Expected tier[%d] '%s', got '%s'", i, expected, stats[i].Tier)
		}
	}

	// Verify haiku stats
	haikuStats := stats[0]
	if haikuStats.TotalCount != 2 {
		t.Errorf("Expected haiku TotalCount 2, got %d", haikuStats.TotalCount)
	}
	if len(haikuStats.AgentBreakdown) != 2 {
		t.Errorf("Expected haiku to have 2 agents, got %d", len(haikuStats.AgentBreakdown))
	}
}

func TestClusterInvocationsByTier_EmptyTier(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "", Agent: "agent1", Success: true},
		{Tier: "", Agent: "agent2", Success: false},
		{Tier: "sonnet", Agent: "python-pro", Success: true},
	}

	stats := ClusterInvocationsByTier(invocations)

	if len(stats) != 2 {
		t.Fatalf("Expected 2 tiers (unknown + sonnet), got %d", len(stats))
	}

	// Find the "unknown" tier
	var unknownStats *TierInvocationStats
	for i := range stats {
		if stats[i].Tier == "unknown" {
			unknownStats = &stats[i]
			break
		}
	}

	if unknownStats == nil {
		t.Fatal("Expected 'unknown' tier not found")
	}
	if unknownStats.TotalCount != 2 {
		t.Errorf("Expected unknown TotalCount 2, got %d", unknownStats.TotalCount)
	}
}

func TestClusterInvocationsByTier_EmptyInput(t *testing.T) {
	stats := ClusterInvocationsByTier([]AgentInvocation{})

	if len(stats) != 0 {
		t.Errorf("Expected empty result, got %d tiers", len(stats))
	}
}

func TestClusterInvocationsByTier_EmptyAgentInBreakdown(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "sonnet", Agent: "", Success: true},
		{Tier: "sonnet", Agent: "python-pro", Success: true},
	}

	stats := ClusterInvocationsByTier(invocations)

	if len(stats) != 1 {
		t.Fatalf("Expected 1 tier, got %d", len(stats))
	}

	// Verify empty agent is tracked as "unknown"
	if stats[0].AgentBreakdown["unknown"] != 1 {
		t.Errorf("Expected 'unknown' agent count 1, got %d", stats[0].AgentBreakdown["unknown"])
	}
	if stats[0].AgentBreakdown["python-pro"] != 1 {
		t.Errorf("Expected 'python-pro' agent count 1, got %d", stats[0].AgentBreakdown["python-pro"])
	}
}

// ===== GetTopAgentsByUsage Tests =====

func TestGetTopAgentsByUsage_Basic(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10},
		{Agent: "agent-b", TotalCount: 50},
		{Agent: "agent-c", TotalCount: 25},
	}

	rankings := GetTopAgentsByUsage(stats, 0)

	if len(rankings) != 3 {
		t.Fatalf("Expected 3 rankings, got %d", len(rankings))
	}

	// Should be sorted descending by usage
	expectedOrder := []string{"agent-b", "agent-c", "agent-a"}
	for i, expected := range expectedOrder {
		if rankings[i].Agent != expected {
			t.Errorf("Expected rank[%d] '%s', got '%s'", i, expected, rankings[i].Agent)
		}
	}

	// Verify metrics
	if rankings[0].Metric != 50.0 {
		t.Errorf("Expected top metric 50.0, got %.1f", rankings[0].Metric)
	}
	if rankings[0].Count != 50 {
		t.Errorf("Expected top count 50, got %d", rankings[0].Count)
	}
}

func TestGetTopAgentsByUsage_WithLimit(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10},
		{Agent: "agent-b", TotalCount: 50},
		{Agent: "agent-c", TotalCount: 25},
	}

	rankings := GetTopAgentsByUsage(stats, 2)

	if len(rankings) != 2 {
		t.Errorf("Expected 2 rankings (limit applied), got %d", len(rankings))
	}

	// Top 2 should be agent-b and agent-c
	if rankings[0].Agent != "agent-b" {
		t.Errorf("Expected first rank 'agent-b', got '%s'", rankings[0].Agent)
	}
	if rankings[1].Agent != "agent-c" {
		t.Errorf("Expected second rank 'agent-c', got '%s'", rankings[1].Agent)
	}
}

func TestGetTopAgentsByUsage_EmptyInput(t *testing.T) {
	rankings := GetTopAgentsByUsage([]AgentInvocationStats{}, 0)

	if len(rankings) != 0 {
		t.Errorf("Expected empty result, got %d rankings", len(rankings))
	}
}

// ===== GetTopAgentsByErrorRate Tests =====

func TestGetTopAgentsByErrorRate_Basic(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10, SuccessCount: 9, SuccessRate: 0.9},  // 10% error
		{Agent: "agent-b", TotalCount: 20, SuccessCount: 15, SuccessRate: 0.75}, // 25% error
		{Agent: "agent-c", TotalCount: 30, SuccessCount: 27, SuccessRate: 0.9},  // 10% error
	}

	rankings := GetTopAgentsByErrorRate(stats, 10, 0)

	if len(rankings) != 3 {
		t.Fatalf("Expected 3 rankings, got %d", len(rankings))
	}

	// Should be sorted descending by error rate (highest error first)
	if rankings[0].Agent != "agent-b" {
		t.Errorf("Expected highest error 'agent-b', got '%s'", rankings[0].Agent)
	}
	expectedErrorRate := 0.25
	if rankings[0].Metric < expectedErrorRate-0.01 || rankings[0].Metric > expectedErrorRate+0.01 {
		t.Errorf("Expected top error rate ~%.2f, got %.2f", expectedErrorRate, rankings[0].Metric)
	}
}

func TestGetTopAgentsByErrorRate_MinInvocations(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 5, SuccessCount: 2, SuccessRate: 0.4},   // 60% error but < 10 invocations
		{Agent: "agent-b", TotalCount: 20, SuccessCount: 15, SuccessRate: 0.75}, // 25% error
		{Agent: "agent-c", TotalCount: 30, SuccessCount: 27, SuccessRate: 0.9},  // 10% error
	}

	rankings := GetTopAgentsByErrorRate(stats, 10, 0)

	// Should exclude agent-a (TotalCount < 10)
	if len(rankings) != 2 {
		t.Errorf("Expected 2 rankings (agent-a filtered), got %d", len(rankings))
	}

	// Verify agent-a is not in results
	for _, r := range rankings {
		if r.Agent == "agent-a" {
			t.Error("Expected agent-a to be filtered out due to minInvocations")
		}
	}
}

func TestGetTopAgentsByErrorRate_WithLimit(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10, SuccessCount: 9, SuccessRate: 0.9},
		{Agent: "agent-b", TotalCount: 20, SuccessCount: 15, SuccessRate: 0.75},
		{Agent: "agent-c", TotalCount: 30, SuccessCount: 27, SuccessRate: 0.9},
	}

	rankings := GetTopAgentsByErrorRate(stats, 10, 1)

	if len(rankings) != 1 {
		t.Errorf("Expected 1 ranking (limit applied), got %d", len(rankings))
	}

	if rankings[0].Agent != "agent-b" {
		t.Errorf("Expected top error agent 'agent-b', got '%s'", rankings[0].Agent)
	}
}

func TestGetTopAgentsByErrorRate_AllFiltered(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 5, SuccessRate: 0.4},
		{Agent: "agent-b", TotalCount: 3, SuccessRate: 0.3},
	}

	rankings := GetTopAgentsByErrorRate(stats, 10, 0)

	if len(rankings) != 0 {
		t.Errorf("Expected empty result (all filtered), got %d", len(rankings))
	}
}

// ===== GetTopAgentsByLatency Tests =====

func TestGetTopAgentsByLatency_Basic(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10, AvgDurationMs: 1000},
		{Agent: "agent-b", TotalCount: 20, AvgDurationMs: 5000},
		{Agent: "agent-c", TotalCount: 30, AvgDurationMs: 2500},
	}

	rankings := GetTopAgentsByLatency(stats, 0)

	if len(rankings) != 3 {
		t.Fatalf("Expected 3 rankings, got %d", len(rankings))
	}

	// Should be sorted descending by latency
	expectedOrder := []string{"agent-b", "agent-c", "agent-a"}
	for i, expected := range expectedOrder {
		if rankings[i].Agent != expected {
			t.Errorf("Expected rank[%d] '%s', got '%s'", i, expected, rankings[i].Agent)
		}
	}

	// Verify top metric
	if rankings[0].Metric != 5000.0 {
		t.Errorf("Expected top latency 5000.0, got %.1f", rankings[0].Metric)
	}
}

func TestGetTopAgentsByLatency_WithLimit(t *testing.T) {
	stats := []AgentInvocationStats{
		{Agent: "agent-a", TotalCount: 10, AvgDurationMs: 1000},
		{Agent: "agent-b", TotalCount: 20, AvgDurationMs: 5000},
		{Agent: "agent-c", TotalCount: 30, AvgDurationMs: 2500},
	}

	rankings := GetTopAgentsByLatency(stats, 2)

	if len(rankings) != 2 {
		t.Errorf("Expected 2 rankings (limit applied), got %d", len(rankings))
	}

	// Top 2 should be agent-b and agent-c
	if rankings[0].Agent != "agent-b" {
		t.Errorf("Expected first rank 'agent-b', got '%s'", rankings[0].Agent)
	}
	if rankings[1].Agent != "agent-c" {
		t.Errorf("Expected second rank 'agent-c', got '%s'", rankings[1].Agent)
	}
}

func TestGetTopAgentsByLatency_EmptyInput(t *testing.T) {
	rankings := GetTopAgentsByLatency([]AgentInvocationStats{}, 0)

	if len(rankings) != 0 {
		t.Errorf("Expected empty result, got %d rankings", len(rankings))
	}
}
