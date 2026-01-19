package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseTranscript_ValidFile tests parsing a valid JSONL file with multiple events.
func TestParseTranscript_ValidFile(t *testing.T) {
	// Create test data
	events := []ToolEvent{
		{
			ToolName:      "Read",
			ToolInput:     map[string]interface{}{"file_path": "/test/file.go"},
			SessionID:     "session-123",
			HookEventName: "PreToolUse",
			CapturedAt:    1705000000,
		},
		{
			ToolName:      "Edit",
			ToolInput:     map[string]interface{}{"file_path": "/test/file.go", "old_string": "old", "new_string": "new"},
			SessionID:     "session-123",
			HookEventName: "PreToolUse",
			CapturedAt:    1705000001,
		},
		{
			ToolName:      "Bash",
			ToolInput:     map[string]interface{}{"command": "go test ./..."},
			SessionID:     "session-123",
			HookEventName: "PreToolUse",
			CapturedAt:    1705000002,
		},
	}

	// Create temp file
	tmpFile := createTempTranscript(t, events)
	defer os.Remove(tmpFile)

	// Parse transcript
	parsed, err := ParseTranscript(tmpFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed: %v", err)
	}

	// Validate results
	if len(parsed) != len(events) {
		t.Fatalf("Expected %d events, got %d", len(events), len(parsed))
	}

	for i, expected := range events {
		actual := parsed[i]
		if actual.ToolName != expected.ToolName {
			t.Errorf("Event %d: expected ToolName %s, got %s", i, expected.ToolName, actual.ToolName)
		}
		if actual.SessionID != expected.SessionID {
			t.Errorf("Event %d: expected SessionID %s, got %s", i, expected.SessionID, actual.SessionID)
		}
		if actual.HookEventName != expected.HookEventName {
			t.Errorf("Event %d: expected HookEventName %s, got %s", i, expected.HookEventName, actual.HookEventName)
		}
		if actual.CapturedAt != expected.CapturedAt {
			t.Errorf("Event %d: expected CapturedAt %d, got %d", i, expected.CapturedAt, actual.CapturedAt)
		}
	}
}

// TestParseTranscript_EmptyFile tests parsing an empty file.
// Should return empty slice with no error.
func TestParseTranscript_EmptyFile(t *testing.T) {
	// Create empty temp file
	tmpFile := createTempFile(t, "")
	defer os.Remove(tmpFile)

	// Parse transcript
	parsed, err := ParseTranscript(tmpFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed on empty file: %v", err)
	}

	if len(parsed) != 0 {
		t.Errorf("Expected empty slice, got %d events", len(parsed))
	}
}

// TestParseTranscript_MalformedJSON tests error handling for invalid JSON.
// Should return error with line number.
func TestParseTranscript_MalformedJSON(t *testing.T) {
	content := `{"tool_name":"Read","session_id":"session-123","hook_event_name":"PreToolUse","captured_at":1705000000}
{"tool_name":"Edit","session_id":"session-123","hook_event_name":"PreToolUse",INVALID JSON HERE
{"tool_name":"Bash","session_id":"session-123","hook_event_name":"PreToolUse","captured_at":1705000002}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	// Parse transcript
	_, err := ParseTranscript(tmpFile)
	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}

	// Validate error message contains line number
	errMsg := err.Error()
	if !strings.Contains(errMsg, "[transcript]") {
		t.Errorf("Error message should contain [transcript] prefix: %s", errMsg)
	}
	if !strings.Contains(errMsg, "line 2") {
		t.Errorf("Error message should contain line number: %s", errMsg)
	}
}

// TestParseTranscript_FileNotFound tests error handling for missing file.
func TestParseTranscript_FileNotFound(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist.jsonl")

	// Parse transcript
	_, err := ParseTranscript(nonExistentPath)
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}

	// Validate error message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "[transcript]") {
		t.Errorf("Error message should contain [transcript] prefix: %s", errMsg)
	}
	if !strings.Contains(errMsg, "File not found") {
		t.Errorf("Error message should contain 'File not found': %s", errMsg)
	}
	if !strings.Contains(errMsg, nonExistentPath) {
		t.Errorf("Error message should contain file path: %s", errMsg)
	}
}

// TestParseTranscript_LargeFile tests performance with large transcript file (1000+ events).
func TestParseTranscript_LargeFile(t *testing.T) {
	// Create 1500 events
	eventCount := 1500
	events := make([]ToolEvent, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = ToolEvent{
			ToolName:      "Read",
			ToolInput:     map[string]interface{}{"file_path": "/test/file.go", "line": i},
			SessionID:     "session-large",
			HookEventName: "PreToolUse",
			CapturedAt:    int64(1705000000 + i),
		}
	}

	// Create temp file
	tmpFile := createTempTranscript(t, events)
	defer os.Remove(tmpFile)

	// Parse transcript
	parsed, err := ParseTranscript(tmpFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed on large file: %v", err)
	}

	// Validate count
	if len(parsed) != eventCount {
		t.Errorf("Expected %d events, got %d", eventCount, len(parsed))
	}

	// Spot check first and last events
	if parsed[0].CapturedAt != 1705000000 {
		t.Errorf("First event CapturedAt mismatch: got %d", parsed[0].CapturedAt)
	}
	if parsed[eventCount-1].CapturedAt != int64(1705000000+eventCount-1) {
		t.Errorf("Last event CapturedAt mismatch: got %d", parsed[eventCount-1].CapturedAt)
	}
}

// TestParseTranscript_EmptyLines tests that empty lines are skipped gracefully.
func TestParseTranscript_EmptyLines(t *testing.T) {
	content := `{"tool_name":"Read","tool_input":{},"session_id":"session-123","hook_event_name":"PreToolUse","captured_at":1705000000}

{"tool_name":"Edit","tool_input":{},"session_id":"session-123","hook_event_name":"PreToolUse","captured_at":1705000001}

{"tool_name":"Bash","tool_input":{},"session_id":"session-123","hook_event_name":"PreToolUse","captured_at":1705000002}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	// Parse transcript
	parsed, err := ParseTranscript(tmpFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed with empty lines: %v", err)
	}

	// Should have 3 events (empty lines skipped)
	if len(parsed) != 3 {
		t.Errorf("Expected 3 events (empty lines skipped), got %d", len(parsed))
	}

	// Validate event sequence
	expectedTools := []string{"Read", "Edit", "Bash"}
	for i, expected := range expectedTools {
		if parsed[i].ToolName != expected {
			t.Errorf("Event %d: expected ToolName %s, got %s", i, expected, parsed[i].ToolName)
		}
	}
}

// Helper: createTempTranscript creates a temp file with JSONL events.
func createTempTranscript(t *testing.T, events []ToolEvent) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "transcript.jsonl")

	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer f.Close()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			t.Fatalf("Failed to write event: %v", err)
		}
	}

	return tmpFile
}

// Helper: createTempFile creates a temp file with arbitrary content.
func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tmpFile
}
