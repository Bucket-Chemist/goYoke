package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayEvent_JSONRoundtrip(t *testing.T) {
	event := ReplayEvent{
		Timestamp:        1705000000,
		HookType:         "PostToolUse",
		ToolName:         "Bash",
		ToolInput:        map[string]interface{}{"command": "python test.py"},
		ToolResponse:     map[string]interface{}{"exit_code": float64(1), "output": "error"},
		Success:          false,
		ExpectedDecision: "block",
	}

	// Marshal
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded ReplayEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if decoded.HookType != "PostToolUse" {
		t.Errorf("HookType: got %q, want %q", decoded.HookType, "PostToolUse")
	}
	if decoded.ExpectedDecision != "block" {
		t.Errorf("ExpectedDecision: got %q, want %q", decoded.ExpectedDecision, "block")
	}
}

func TestReplaySession_JSONRoundtrip(t *testing.T) {
	session := ReplaySession{
		ID:          "test-session",
		Description: "Test session for debugging loop",
		Events: []ReplayEvent{
			{Timestamp: 1, HookType: "PostToolUse", ToolName: "Bash"},
			{Timestamp: 2, HookType: "PostToolUse", ToolName: "Bash"},
		},
		Expected: ReplayExpectations{
			SharpEdgesCreated: 1,
			BlockingResponses: 1,
			FilesCreated:      []string{".claude/memory/pending-learnings.jsonl"},
		},
	}

	// Marshal
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded ReplaySession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if decoded.ID != "test-session" {
		t.Errorf("ID: got %q, want %q", decoded.ID, "test-session")
	}
	if len(decoded.Events) != 2 {
		t.Errorf("Events count: got %d, want 2", len(decoded.Events))
	}
	if decoded.Expected.SharpEdgesCreated != 1 {
		t.Errorf("SharpEdgesCreated: got %d, want 1", decoded.Expected.SharpEdgesCreated)
	}
}

func TestCountJSONLLines(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    int
	}{
		{"empty", []byte{}, 0},
		{"single line", []byte(`{"a":1}`), 1},
		{"single line with newline", []byte(`{"a":1}` + "\n"), 1},
		{"two lines", []byte(`{"a":1}` + "\n" + `{"b":2}` + "\n"), 2},
		{"three lines no trailing", []byte(`{"a":1}` + "\n" + `{"b":2}` + "\n" + `{"c":3}`), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countJSONLLines(tt.content)
			if got != tt.want {
				t.Errorf("countJSONLLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is way too long", 10, "this is wa..."},
	}

	for _, tt := range tests {
		got := truncateOutput(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateOutput(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestSessionReplayer_LoadSession(t *testing.T) {
	// Create temp fixtures directory
	tmpDir, err := os.MkdirTemp("", "test-replayer-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionsDir := filepath.Join(tmpDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)

	// Create a test session file
	sessionContent := `{"session_id":"test-load","description":"Test loading","expected":{"sharp_edges_created":1}}
{"ts":1705000000,"hook_type":"PostToolUse","tool_name":"Bash","tool_input":{"command":"test"},"success":false}
{"ts":1705000010,"hook_type":"PostToolUse","tool_name":"Bash","tool_input":{"command":"test"},"success":false}
`
	sessionPath := filepath.Join(sessionsDir, "test-load.jsonl")
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("write session: %v", err)
	}

	// Create replayer and load
	replayer := NewSessionReplayer("", "", "", tmpDir)
	session, err := replayer.loadSession(sessionPath)
	if err != nil {
		t.Fatalf("loadSession failed: %v", err)
	}

	// Verify
	if session.ID != "test-load" {
		t.Errorf("ID: got %q, want %q", session.ID, "test-load")
	}
	if len(session.Events) != 2 {
		t.Errorf("Events: got %d, want 2", len(session.Events))
	}
	if session.Expected.SharpEdgesCreated != 1 {
		t.Errorf("SharpEdgesCreated: got %d, want 1", session.Expected.SharpEdgesCreated)
	}
	if session.Events[0].ToolName != "Bash" {
		t.Errorf("Event 0 ToolName: got %q, want %q", session.Events[0].ToolName, "Bash")
	}
}

func TestSessionReplayer_LoadSession_DeriveID(t *testing.T) {
	// Create temp fixtures directory
	tmpDir, err := os.MkdirTemp("", "test-replayer-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionsDir := filepath.Join(tmpDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)

	// Session file without explicit ID
	sessionContent := `{"description":"No explicit ID","expected":{}}
{"ts":1705000000,"hook_type":"PostToolUse","tool_name":"Bash","success":false}
`
	sessionPath := filepath.Join(sessionsDir, "derived-name.jsonl")
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("write session: %v", err)
	}

	replayer := NewSessionReplayer("", "", "", tmpDir)
	session, err := replayer.loadSession(sessionPath)
	if err != nil {
		t.Fatalf("loadSession failed: %v", err)
	}

	// ID should be derived from filename
	if session.ID != "derived-name" {
		t.Errorf("ID: got %q, want %q", session.ID, "derived-name")
	}
}

func TestSessionReplayer_ValidateExpectations(t *testing.T) {
	// Create temp dir with test files
	tmpDir, err := os.MkdirTemp("", "test-expectations-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup files
	os.MkdirAll(filepath.Join(tmpDir, ".claude", "memory"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl"),
		[]byte(`{"test":1}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "handoffs.jsonl"),
		[]byte(`{"schema_version":"1.3"}`), 0644)

	replayer := NewSessionReplayer("", "", "", "")

	t.Run("all pass", func(t *testing.T) {
		expected := ReplayExpectations{
			SharpEdgesCreated: 1,
			BlockingResponses: 1,
			HandoffCreated:    true,
			FilesCreated:      []string{".claude/memory/pending-learnings.jsonl"},
		}
		errors := replayer.validateExpectations(expected, tmpDir, 1, 1)
		if len(errors) != 0 {
			t.Errorf("expected no errors, got: %v", errors)
		}
	})

	t.Run("wrong sharp edge count", func(t *testing.T) {
		expected := ReplayExpectations{
			SharpEdgesCreated: 5, // Wrong
		}
		errors := replayer.validateExpectations(expected, tmpDir, 1, 0)
		if len(errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errors), errors)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		expected := ReplayExpectations{
			FilesCreated: []string{".claude/memory/nonexistent.jsonl"},
		}
		errors := replayer.validateExpectations(expected, tmpDir, 0, 0)
		if len(errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errors), errors)
		}
	})

	t.Run("file should not exist", func(t *testing.T) {
		expected := ReplayExpectations{
			FilesNotCreated: []string{".claude/memory/pending-learnings.jsonl"},
		}
		errors := replayer.validateExpectations(expected, tmpDir, 0, 0)
		if len(errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errors), errors)
		}
	})
}

func TestSessionReplayer_BuildEnv(t *testing.T) {
	replayer := &SessionReplayer{
		schemaPath: "/path/to/schema.json",
		agentsPath: "/path/to/agents.json",
	}

	tempDir := "/tmp/test-session"
	env := replayer.buildEnv(tempDir)

	// Check required variables exist
	envMap := make(map[string]string)
	for _, e := range env {
		parts := splitEnvVar(e)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["GOGENT_PROJECT_DIR"] != tempDir {
		t.Errorf("GOGENT_PROJECT_DIR: got %q, want %q", envMap["GOGENT_PROJECT_DIR"], tempDir)
	}
	if envMap["GOGENT_ROUTING_SCHEMA"] != "/path/to/schema.json" {
		t.Errorf("GOGENT_ROUTING_SCHEMA not set correctly")
	}
	if envMap["GOGENT_MAX_FAILURES"] != "3" {
		t.Errorf("GOGENT_MAX_FAILURES: got %q, want %q", envMap["GOGENT_MAX_FAILURES"], "3")
	}
}

// splitEnvVar splits KEY=VALUE into [KEY, VALUE]
func splitEnvVar(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
