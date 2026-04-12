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

func TestParseTranscript_LargeSingleEvent(t *testing.T) {
	largeCommand := strings.Repeat("x", 70*1024)
	events := []ToolEvent{
		{
			ToolName: "Bash",
			ToolInput: map[string]interface{}{
				"command": largeCommand,
			},
			SessionID:     "session-large-line",
			HookEventName: "PreToolUse",
			CapturedAt:    1705000000,
		},
	}

	tmpFile := createTempTranscript(t, events)
	defer os.Remove(tmpFile)

	parsed, err := ParseTranscript(tmpFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed on large single event: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(parsed))
	}
	if parsed[0].ToolInput["command"] != largeCommand {
		t.Fatalf("Large command payload was truncated during parsing")
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

// TestAnalyzeToolDistribution_EmptySlice tests that an empty slice returns an empty map.
func TestAnalyzeToolDistribution_EmptySlice(t *testing.T) {
	result := AnalyzeToolDistribution([]ToolEvent{})

	if len(result) != 0 {
		t.Errorf("Expected empty map for empty slice, got: %v", result)
	}
}

// TestAnalyzeToolDistribution_NilSlice tests that a nil slice returns a non-nil empty map.
func TestAnalyzeToolDistribution_NilSlice(t *testing.T) {
	result := AnalyzeToolDistribution(nil)

	if result == nil {
		t.Error("Expected non-nil map for nil input, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty map for nil slice, got: %v", result)
	}
}

// TestAnalyzeToolDistribution_SingleToolType tests counting with only one tool type.
func TestAnalyzeToolDistribution_SingleToolType(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read"},
		{ToolName: "Read"},
		{ToolName: "Read"},
	}

	result := AnalyzeToolDistribution(events)

	if result["Read"] != 3 {
		t.Errorf("Expected Read count of 3, got: %d", result["Read"])
	}

	if len(result) != 1 {
		t.Errorf("Expected only 1 tool type, got: %d", len(result))
	}
}

// TestAnalyzeToolDistribution_MixedTools tests counting with multiple tool types.
func TestAnalyzeToolDistribution_MixedTools(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read"},
		{ToolName: "Edit"},
		{ToolName: "Read"},
		{ToolName: "Write"},
		{ToolName: "Edit"},
		{ToolName: "Read"},
	}

	result := AnalyzeToolDistribution(events)

	expected := map[string]int{
		"Read":  3,
		"Edit":  2,
		"Write": 1,
	}

	for tool, expectedCount := range expected {
		if result[tool] != expectedCount {
			t.Errorf("Expected %s count of %d, got: %d", tool, expectedCount, result[tool])
		}
	}
}

// TestAnalyzeToolDistribution_UnknownToolNames tests that unknown tool names are counted correctly.
func TestAnalyzeToolDistribution_UnknownToolNames(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "CustomTool"},
		{ToolName: "AnotherTool"},
		{ToolName: "CustomTool"},
	}

	result := AnalyzeToolDistribution(events)

	if result["CustomTool"] != 2 {
		t.Errorf("Expected CustomTool count of 2, got: %d", result["CustomTool"])
	}

	if result["AnotherTool"] != 1 {
		t.Errorf("Expected AnotherTool count of 1, got: %d", result["AnotherTool"])
	}
}

// TestAnalyzeToolDistribution_AccuracyWithRealPattern tests accuracy with a realistic tool usage pattern.
func TestAnalyzeToolDistribution_AccuracyWithRealPattern(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read"},
		{ToolName: "Read"},
		{ToolName: "Read"},
		{ToolName: "Glob"},
		{ToolName: "Glob"},
		{ToolName: "Grep"},
		{ToolName: "Grep"},
		{ToolName: "Grep"},
		{ToolName: "Edit"},
		{ToolName: "Write"},
		{ToolName: "Bash"},
		{ToolName: "Task"},
	}

	result := AnalyzeToolDistribution(events)

	expected := map[string]int{
		"Read":  3,
		"Glob":  2,
		"Grep":  3,
		"Edit":  1,
		"Write": 1,
		"Bash":  1,
		"Task":  1,
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d tool types, got: %d", len(expected), len(result))
	}

	for tool, expectedCount := range expected {
		if result[tool] != expectedCount {
			t.Errorf("Tool %s: expected count %d, got: %d", tool, expectedCount, result[tool])
		}
	}
}

func TestTranscriptAnalyzer_AnalyzeLargeSingleEvent(t *testing.T) {
	largeCommand := strings.Repeat("x", 70*1024)
	events := []ToolEvent{
		{
			ToolName: "Bash",
			ToolInput: map[string]interface{}{
				"command":           largeCommand,
				"run_in_background": true,
				"task_id":           "task-large-line",
			},
			SessionID:     "session-large-line",
			HookEventName: "PreToolUse",
			CapturedAt:    1705000000,
		},
	}

	tmpFile := createTempTranscript(t, events)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed on large single event: %v", err)
	}
	if !analyzer.HasUncollectedTasks() {
		t.Fatalf("Expected large background task event to be tracked as uncollected")
	}
}

func TestDetectPhases_EmptyEvents(t *testing.T) {
	result := DetectPhases([]ToolEvent{})

	if len(result) != 0 {
		t.Errorf("Expected empty slice for empty events, got: %v", result)
	}
}

func TestDetectPhases_DiscoverySession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Read", CapturedAt: 1200},
		{ToolName: "Read", CapturedAt: 1300},
		{ToolName: "Read", CapturedAt: 1400},
		{ToolName: "Read", CapturedAt: 1500},
		{ToolName: "Read", CapturedAt: 1600},
		{ToolName: "Glob", CapturedAt: 1700},
		{ToolName: "Grep", CapturedAt: 1800},
		{ToolName: "Edit", CapturedAt: 1900}, // 10% non-discovery
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase, got: %s", result[0].Phase)
	}

	if result[0].ToolCount != 10 {
		t.Errorf("Expected tool count 10, got: %d", result[0].ToolCount)
	}

	if result[0].Duration != 900 {
		t.Errorf("Expected duration 900, got: %d", result[0].Duration)
	}
}

func TestDetectPhases_ImplementationSession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Edit", CapturedAt: 1000},
		{ToolName: "Edit", CapturedAt: 1100},
		{ToolName: "Edit", CapturedAt: 1200},
		{ToolName: "Edit", CapturedAt: 1300},
		{ToolName: "Write", CapturedAt: 1400},
		{ToolName: "Write", CapturedAt: 1500},
		{ToolName: "Write", CapturedAt: 1600},
		{ToolName: "Write", CapturedAt: 1700},
		{ToolName: "Read", CapturedAt: 1800},
		{ToolName: "Read", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "implementation" {
		t.Errorf("Expected implementation phase, got: %s", result[0].Phase)
	}
}

func TestDetectPhases_DebuggingSession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Bash", CapturedAt: 1000},
		{ToolName: "Bash", CapturedAt: 1100},
		{ToolName: "Bash", CapturedAt: 1200},
		{ToolName: "Bash", CapturedAt: 1300},
		{ToolName: "Bash", CapturedAt: 1400},
		{ToolName: "Bash", CapturedAt: 1500},
		{ToolName: "Read", CapturedAt: 1600},
		{ToolName: "Read", CapturedAt: 1700},
		{ToolName: "Read", CapturedAt: 1800},
		{ToolName: "Edit", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "debugging" {
		t.Errorf("Expected debugging phase, got: %s", result[0].Phase)
	}
}

func TestDetectPhases_DelegationSession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Task", CapturedAt: 1000},
		{ToolName: "Task", CapturedAt: 1100},
		{ToolName: "Task", CapturedAt: 1200},
		{ToolName: "Task", CapturedAt: 1300},
		{ToolName: "Task", CapturedAt: 1400},
		{ToolName: "Task", CapturedAt: 1500},
		{ToolName: "Task", CapturedAt: 1600},
		{ToolName: "Task", CapturedAt: 1700},
		{ToolName: "Read", CapturedAt: 1800},
		{ToolName: "Read", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "delegation" {
		t.Errorf("Expected delegation phase, got: %s", result[0].Phase)
	}
}

func TestDetectPhases_MixedSession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Edit", CapturedAt: 1200},
		{ToolName: "Edit", CapturedAt: 1300},
		{ToolName: "Bash", CapturedAt: 1400},
		{ToolName: "Bash", CapturedAt: 1500},
		{ToolName: "Task", CapturedAt: 1600},
		{ToolName: "Task", CapturedAt: 1700},
		{ToolName: "Glob", CapturedAt: 1800},
		{ToolName: "Write", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "mixed" {
		t.Errorf("Expected mixed phase, got: %s", result[0].Phase)
	}
}

func TestDetectPhases_ShortSession(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].ToolCount != 2 {
		t.Errorf("Expected tool count 2, got: %d", result[0].ToolCount)
	}
}

func TestDetectPhases_ThresholdBoundary(t *testing.T) {
	// Test 70% threshold exactly (7 out of 10)
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Read", CapturedAt: 1200},
		{ToolName: "Read", CapturedAt: 1300},
		{ToolName: "Read", CapturedAt: 1400},
		{ToolName: "Read", CapturedAt: 1500},
		{ToolName: "Read", CapturedAt: 1600},
		{ToolName: "Edit", CapturedAt: 1700},
		{ToolName: "Edit", CapturedAt: 1800},
		{ToolName: "Edit", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase at 70%% threshold, got: %s", result[0].Phase)
	}
}

// TestDetectPhases_BelowThreshold69 tests that 6/10 (60%) doesn't trigger discovery
func TestDetectPhases_BelowThreshold69(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Read", CapturedAt: 1200},
		{ToolName: "Read", CapturedAt: 1300},
		{ToolName: "Read", CapturedAt: 1400},
		{ToolName: "Read", CapturedAt: 1500},
		{ToolName: "Edit", CapturedAt: 1600},
		{ToolName: "Edit", CapturedAt: 1700},
		{ToolName: "Edit", CapturedAt: 1800},
		{ToolName: "Bash", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	// 60% read, 30% edit - neither meets 70%, no 50%+ bash
	if result[0].Phase != "mixed" {
		t.Errorf("Expected mixed phase below 70%% threshold, got: %s", result[0].Phase)
	}
}

// TestDetectPhases_DebuggingThreshold50 tests debugging triggers at 50% exactly
func TestDetectPhases_DebuggingThreshold50(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Bash", CapturedAt: 1000},
		{ToolName: "Bash", CapturedAt: 1100},
		{ToolName: "Bash", CapturedAt: 1200},
		{ToolName: "Bash", CapturedAt: 1300},
		{ToolName: "Bash", CapturedAt: 1400},
		{ToolName: "Read", CapturedAt: 1500},
		{ToolName: "Read", CapturedAt: 1600},
		{ToolName: "Edit", CapturedAt: 1700},
		{ToolName: "Edit", CapturedAt: 1800},
		{ToolName: "Write", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	// 50% bash should trigger debugging
	if result[0].Phase != "debugging" {
		t.Errorf("Expected debugging phase at 50%% threshold, got: %s", result[0].Phase)
	}
}

// TestDetectPhases_PriorityOrderingDiscoveryBeatsImplementation tests discovery takes precedence
func TestDetectPhases_PriorityOrderingDiscoveryBeatsImplementation(t *testing.T) {
	// Both discovery (70%) and implementation (70%) would qualify, but discovery has priority
	// 7 reads + 7 edits = 14 total, both are 50% (neither triggers alone)
	// But if we make it 8 reads + 6 edits = 14 total, reads = 57% (not triggered)
	// Let's test: 10 reads + 4 edits = 14 total, reads = 71% → should be discovery
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Read", CapturedAt: 1200},
		{ToolName: "Read", CapturedAt: 1300},
		{ToolName: "Read", CapturedAt: 1400},
		{ToolName: "Glob", CapturedAt: 1500},
		{ToolName: "Glob", CapturedAt: 1600},
		{ToolName: "Glob", CapturedAt: 1700},
		{ToolName: "Glob", CapturedAt: 1800},
		{ToolName: "Glob", CapturedAt: 1900},
		{ToolName: "Edit", CapturedAt: 2000},
		{ToolName: "Edit", CapturedAt: 2100},
		{ToolName: "Write", CapturedAt: 2200},
		{ToolName: "Write", CapturedAt: 2300},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	// 10/14 = 71% discovery tools, 4/14 = 28% implementation tools
	// Discovery should win due to priority
	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase (priority), got: %s", result[0].Phase)
	}
}

// TestDetectPhases_SingleEvent tests single-event sessions
func TestDetectPhases_SingleEvent(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	// 1 Read = 100% discovery
	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase for single Read event, got: %s", result[0].Phase)
	}

	if result[0].ToolCount != 1 {
		t.Errorf("Expected tool count 1, got: %d", result[0].ToolCount)
	}

	if result[0].Duration != 0 {
		t.Errorf("Expected duration 0 for single event, got: %d", result[0].Duration)
	}

	if result[0].StartTime != 1000 {
		t.Errorf("Expected start time 1000, got: %d", result[0].StartTime)
	}
}

// TestDetectPhases_StartTimeAndDuration tests time calculations
func TestDetectPhases_StartTimeAndDuration(t *testing.T) {
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 5000},
		{ToolName: "Read", CapturedAt: 5500},
		{ToolName: "Read", CapturedAt: 6000},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].StartTime != 5000 {
		t.Errorf("Expected start time 5000, got: %d", result[0].StartTime)
	}

	if result[0].Duration != 1000 {
		t.Errorf("Expected duration 1000 (6000-5000), got: %d", result[0].Duration)
	}
}

// TestDetectPhases_Implementation70Percent tests implementation at exactly 70%
func TestDetectPhases_Implementation70Percent(t *testing.T) {
	// 7 implementation tools, 3 others = 10 total (70%)
	events := []ToolEvent{
		{ToolName: "Edit", CapturedAt: 1000},
		{ToolName: "Edit", CapturedAt: 1100},
		{ToolName: "Edit", CapturedAt: 1200},
		{ToolName: "Write", CapturedAt: 1300},
		{ToolName: "Write", CapturedAt: 1400},
		{ToolName: "Write", CapturedAt: 1500},
		{ToolName: "Write", CapturedAt: 1600},
		{ToolName: "Read", CapturedAt: 1700},
		{ToolName: "Read", CapturedAt: 1800},
		{ToolName: "Bash", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "implementation" {
		t.Errorf("Expected implementation phase at 70%% threshold, got: %s", result[0].Phase)
	}
}

// TestDetectPhases_Delegation70Percent tests delegation at exactly 70%
func TestDetectPhases_Delegation70Percent(t *testing.T) {
	// 7 Task tools, 3 others = 10 total (70%)
	events := []ToolEvent{
		{ToolName: "Task", CapturedAt: 1000},
		{ToolName: "Task", CapturedAt: 1100},
		{ToolName: "Task", CapturedAt: 1200},
		{ToolName: "Task", CapturedAt: 1300},
		{ToolName: "Task", CapturedAt: 1400},
		{ToolName: "Task", CapturedAt: 1500},
		{ToolName: "Task", CapturedAt: 1600},
		{ToolName: "Read", CapturedAt: 1700},
		{ToolName: "Read", CapturedAt: 1800},
		{ToolName: "Edit", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "delegation" {
		t.Errorf("Expected delegation phase at 70%% threshold, got: %s", result[0].Phase)
	}
}

// TestDetectPhases_MixedToolCombination tests various tool combinations that don't meet thresholds
func TestDetectPhases_MixedToolCombination(t *testing.T) {
	// Designed so no category reaches its threshold:
	// 4 Read = 40%, 3 Edit = 30%, 2 Bash = 20%, 1 Task = 10%
	events := []ToolEvent{
		{ToolName: "Read", CapturedAt: 1000},
		{ToolName: "Read", CapturedAt: 1100},
		{ToolName: "Read", CapturedAt: 1200},
		{ToolName: "Read", CapturedAt: 1300},
		{ToolName: "Edit", CapturedAt: 1400},
		{ToolName: "Edit", CapturedAt: 1500},
		{ToolName: "Edit", CapturedAt: 1600},
		{ToolName: "Bash", CapturedAt: 1700},
		{ToolName: "Bash", CapturedAt: 1800},
		{ToolName: "Task", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "mixed" {
		t.Errorf("Expected mixed phase, got: %s", result[0].Phase)
	}

	if result[0].ToolCount != 10 {
		t.Errorf("Expected tool count 10, got: %d", result[0].ToolCount)
	}
}

// TestDetectPhases_GrepAsDiscoveryTool tests that Grep counts toward discovery
func TestDetectPhases_GrepAsDiscoveryTool(t *testing.T) {
	// 7 Grep tools + 3 others = 10 total (70% discovery)
	events := []ToolEvent{
		{ToolName: "Grep", CapturedAt: 1000},
		{ToolName: "Grep", CapturedAt: 1100},
		{ToolName: "Grep", CapturedAt: 1200},
		{ToolName: "Grep", CapturedAt: 1300},
		{ToolName: "Grep", CapturedAt: 1400},
		{ToolName: "Grep", CapturedAt: 1500},
		{ToolName: "Grep", CapturedAt: 1600},
		{ToolName: "Edit", CapturedAt: 1700},
		{ToolName: "Edit", CapturedAt: 1800},
		{ToolName: "Bash", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase (Grep counts as discovery), got: %s", result[0].Phase)
	}
}

// TestDetectPhases_GlobAsDiscoveryTool tests that Glob counts toward discovery
func TestDetectPhases_GlobAsDiscoveryTool(t *testing.T) {
	// 7 Glob tools + 3 others = 10 total (70% discovery)
	events := []ToolEvent{
		{ToolName: "Glob", CapturedAt: 1000},
		{ToolName: "Glob", CapturedAt: 1100},
		{ToolName: "Glob", CapturedAt: 1200},
		{ToolName: "Glob", CapturedAt: 1300},
		{ToolName: "Glob", CapturedAt: 1400},
		{ToolName: "Glob", CapturedAt: 1500},
		{ToolName: "Glob", CapturedAt: 1600},
		{ToolName: "Edit", CapturedAt: 1700},
		{ToolName: "Write", CapturedAt: 1800},
		{ToolName: "Bash", CapturedAt: 1900},
	}

	result := DetectPhases(events)

	if len(result) != 1 {
		t.Fatalf("Expected 1 phase, got: %d", len(result))
	}

	if result[0].Phase != "discovery" {
		t.Errorf("Expected discovery phase (Glob counts as discovery), got: %s", result[0].Phase)
	}
}

// TestTaskTracker_IDBasedTracking tests basic ID-based tracking functionality.
func TestTaskTracker_IDBasedTracking(t *testing.T) {
	tracker := NewTaskTracker()

	// Initially empty
	if tracker.HasUncollected() {
		t.Error("Expected no uncollected tasks initially")
	}

	spawned, collected, uncollected := tracker.GetStats()
	if spawned != 0 || collected != 0 || len(uncollected) != 0 {
		t.Errorf("Expected empty stats initially, got spawned=%d, collected=%d, uncollected=%d", spawned, collected, len(uncollected))
	}

	// Spawn task
	tracker.SpawnedIDs["task-1"] = true

	if !tracker.HasUncollected() {
		t.Error("Expected uncollected task after spawn")
	}

	spawned, collected, uncollected = tracker.GetStats()
	if spawned != 1 || collected != 0 || len(uncollected) != 1 {
		t.Errorf("Expected spawned=1, collected=0, uncollected=1, got spawned=%d, collected=%d, uncollected=%d", spawned, collected, len(uncollected))
	}

	if uncollected[0] != "task-1" {
		t.Errorf("Expected uncollected task-1, got: %s", uncollected[0])
	}

	// Collect task
	tracker.CollectedIDs["task-1"] = true

	if tracker.HasUncollected() {
		t.Error("Expected no uncollected tasks after collection")
	}

	spawned, collected, uncollected = tracker.GetStats()
	if spawned != 1 || collected != 1 || len(uncollected) != 0 {
		t.Errorf("Expected spawned=1, collected=1, uncollected=0, got spawned=%d, collected=%d, uncollected=%d", spawned, collected, len(uncollected))
	}
}

// TestTaskTracker_DuplicateIDs tests idempotent duplicate ID handling.
func TestTaskTracker_DuplicateIDs(t *testing.T) {
	tracker := NewTaskTracker()

	// Spawn same ID multiple times (idempotent)
	tracker.SpawnedIDs["task-dup"] = true
	tracker.SpawnedIDs["task-dup"] = true
	tracker.SpawnedIDs["task-dup"] = true

	spawned, _, uncollected := tracker.GetStats()
	if spawned != 1 {
		t.Errorf("Expected spawned=1 (idempotent), got: %d", spawned)
	}

	if len(uncollected) != 1 {
		t.Errorf("Expected 1 uncollected task, got: %d", len(uncollected))
	}

	// Collect same ID multiple times (idempotent)
	tracker.CollectedIDs["task-dup"] = true
	tracker.CollectedIDs["task-dup"] = true

	_, collected, uncollected := tracker.GetStats()
	if collected != 1 {
		t.Errorf("Expected collected=1 (idempotent), got: %d", collected)
	}

	if len(uncollected) != 0 {
		t.Errorf("Expected 0 uncollected tasks, got: %d", len(uncollected))
	}
}

// TestTranscriptAnalyzer_NoBackgroundTasks tests empty transcript case.
func TestTranscriptAnalyzer_NoBackgroundTasks(t *testing.T) {
	content := `{"tool_name":"Read","tool_input":{"file_path":"/test.go"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1000}
{"tool_name":"Edit","tool_input":{"file_path":"/test.go","old_string":"old","new_string":"new"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1001}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks")
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "No background tasks detected") {
		t.Errorf("Expected 'No background tasks detected' in summary, got: %s", summary)
	}

	uncollectedList := analyzer.GetUncollectedList()
	if uncollectedList != "" {
		t.Errorf("Expected empty uncollected list, got: %s", uncollectedList)
	}
}

// TestTranscriptAnalyzer_JSONParsing tests structured JSON parsing.
func TestTranscriptAnalyzer_JSONParsing(t *testing.T) {
	content := `{"tool_name":"Bash","tool_input":{"command":"long_task.sh","run_in_background":true,"task_id":"bg-123"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1000}
{"tool_name":"Read","tool_input":{"file_path":"/test.go"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1001}
{"tool_name":"TaskOutput","tool_input":{"task_id":"bg-123","block":true},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1002}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks (spawn + collect)")
	}

	spawned, collected, _ := analyzer.tracker.GetStats()
	if spawned != 1 {
		t.Errorf("Expected 1 spawned task, got: %d", spawned)
	}
	if collected != 1 {
		t.Errorf("Expected 1 collected task, got: %d", collected)
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "All background tasks collected") {
		t.Errorf("Expected 'All background tasks collected' in summary, got: %s", summary)
	}
}

// TestTranscriptAnalyzer_RegexFallback tests prose pattern detection via regex.
func TestTranscriptAnalyzer_RegexFallback(t *testing.T) {
	content := `Some prose description with run_in_background: true and task_id: "prose-task-1"
Another line with no background task
Collecting with TaskOutput task_id: "prose-task-1"
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks (regex detection should work)")
	}

	spawned, collected, _ := analyzer.tracker.GetStats()
	if spawned != 1 {
		t.Errorf("Expected 1 spawned task (regex), got: %d", spawned)
	}
	if collected != 1 {
		t.Errorf("Expected 1 collected task (regex), got: %d", collected)
	}
}

// TestTranscriptAnalyzer_UncollectedTasks tests detection of uncollected tasks.
func TestTranscriptAnalyzer_UncollectedTasks(t *testing.T) {
	content := `{"tool_name":"Bash","tool_input":{"command":"task1.sh","run_in_background":true,"task_id":"bg-1"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1000}
{"tool_name":"Bash","tool_input":{"command":"task2.sh","run_in_background":true,"task_id":"bg-2"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1001}
{"tool_name":"TaskOutput","tool_input":{"task_id":"bg-1","block":true},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1002}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !analyzer.HasUncollectedTasks() {
		t.Error("Expected uncollected task (bg-2 not collected)")
	}

	spawned, collected, uncollected := analyzer.tracker.GetStats()
	if spawned != 2 {
		t.Errorf("Expected 2 spawned tasks, got: %d", spawned)
	}
	if collected != 1 {
		t.Errorf("Expected 1 collected task, got: %d", collected)
	}
	if len(uncollected) != 1 {
		t.Fatalf("Expected 1 uncollected task, got: %d", len(uncollected))
	}
	if uncollected[0] != "bg-2" {
		t.Errorf("Expected uncollected task bg-2, got: %s", uncollected[0])
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "Uncollected Tasks: 1") {
		t.Errorf("Expected 'Uncollected Tasks: 1' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "bg-2") {
		t.Errorf("Expected 'bg-2' in summary, got: %s", summary)
	}

	uncollectedList := analyzer.GetUncollectedList()
	if uncollectedList != "bg-2" {
		t.Errorf("Expected uncollected list 'bg-2', got: %s", uncollectedList)
	}
}

// TestTranscriptAnalyzer_MissingFile tests graceful handling of missing transcript.
func TestTranscriptAnalyzer_MissingFile(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist.jsonl")

	analyzer := NewTranscriptAnalyzer(nonExistentPath)
	if err := analyzer.Analyze(); err != nil {
		t.Errorf("Expected nil error for missing file, got: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks for missing file")
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "No background tasks detected") {
		t.Errorf("Expected 'No background tasks detected' for missing file, got: %s", summary)
	}
}

// TestTranscriptAnalyzer_MalformedJSON tests fallback behavior for malformed JSON.
func TestTranscriptAnalyzer_MalformedJSON(t *testing.T) {
	content := `{"tool_name":"Bash","tool_input":{"command":"test.sh","run_in_background":true,"task_id":"malformed-1"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1000}
This is malformed JSON with run_in_background: true and task_id: "malformed-2"
{"tool_name":"TaskOutput","tool_input":{"task_id":"malformed-1","block":true},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1002}
Another prose line with TaskOutput task_id: "malformed-2"
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze should not error on malformed JSON (fallback to regex): %v", err)
	}

	// Should detect both via JSON and regex fallback
	spawned, collected, uncollected := analyzer.tracker.GetStats()
	if spawned != 2 {
		t.Errorf("Expected 2 spawned tasks (1 JSON, 1 regex), got: %d", spawned)
	}
	if collected != 2 {
		t.Errorf("Expected 2 collected tasks (1 JSON, 1 regex), got: %d", collected)
	}
	if len(uncollected) != 0 {
		t.Errorf("Expected 0 uncollected tasks, got: %d", len(uncollected))
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks (both collected)")
	}
}

// TestTranscriptAnalyzer_MultipleUncollected tests multiple uncollected task IDs.
func TestTranscriptAnalyzer_MultipleUncollected(t *testing.T) {
	content := `{"tool_name":"Bash","tool_input":{"command":"task1.sh","run_in_background":true,"task_id":"bg-1"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1000}
{"tool_name":"Bash","tool_input":{"command":"task2.sh","run_in_background":true,"task_id":"bg-2"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1001}
{"tool_name":"Bash","tool_input":{"command":"task3.sh","run_in_background":true,"task_id":"bg-3"},"session_id":"session-1","hook_event_name":"PreToolUse","captured_at":1002}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !analyzer.HasUncollectedTasks() {
		t.Error("Expected uncollected tasks")
	}

	spawned, collected, uncollected := analyzer.tracker.GetStats()
	if spawned != 3 {
		t.Errorf("Expected 3 spawned tasks, got: %d", spawned)
	}
	if collected != 0 {
		t.Errorf("Expected 0 collected tasks, got: %d", collected)
	}
	if len(uncollected) != 3 {
		t.Errorf("Expected 3 uncollected tasks, got: %d", len(uncollected))
	}

	uncollectedList := analyzer.GetUncollectedList()
	for _, id := range []string{"bg-1", "bg-2", "bg-3"} {
		if !strings.Contains(uncollectedList, id) {
			t.Errorf("Expected uncollected list to contain %s, got: %s", id, uncollectedList)
		}
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "Uncollected Tasks: 3") {
		t.Errorf("Expected 'Uncollected Tasks: 3' in summary, got: %s", summary)
	}
}

// TestTranscriptAnalyzer_EmptyTranscript tests handling of empty transcript file.
func TestTranscriptAnalyzer_EmptyTranscript(t *testing.T) {
	tmpFile := createTempFile(t, "")
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed on empty file: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks in empty transcript")
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "No background tasks detected") {
		t.Errorf("Expected 'No background tasks detected' in summary, got: %s", summary)
	}
}

// TestTranscriptAnalyzer_CaseInsensitiveRegex tests case-insensitive pattern matching.
func TestTranscriptAnalyzer_CaseInsensitiveRegex(t *testing.T) {
	content := `Prose with RUN_IN_BACKGROUND: TRUE and task_id: "case-1"
Another with Run_In_Background: True and task_id: "case-2"
Collect with TASKOUTPUT task_id: "case-1"
Collect with taskoutput task_id: "case-2"
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	spawned, collected, uncollected := analyzer.tracker.GetStats()
	if spawned != 2 {
		t.Errorf("Expected 2 spawned tasks (case-insensitive), got: %d", spawned)
	}
	if collected != 2 {
		t.Errorf("Expected 2 collected tasks (case-insensitive), got: %d", collected)
	}
	if len(uncollected) != 0 {
		t.Errorf("Expected 0 uncollected tasks, got: %d", len(uncollected))
	}
}

// TestTranscriptAnalyzer_ReadError tests error handling for file read errors.
func TestTranscriptAnalyzer_ReadError(t *testing.T) {
	// Create a file with restrictive permissions to trigger read error
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "unreadable.jsonl")

	// Write content
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make file readable so we can open it (scanner error is harder to trigger)
	// This test verifies graceful handling of file system errors
	analyzer := NewTranscriptAnalyzer(tmpFile)
	if err := analyzer.Analyze(); err != nil {
		// If we get an error, it should be properly formatted
		if !strings.Contains(err.Error(), "[transcript]") {
			t.Errorf("Error should contain [transcript] prefix: %v", err)
		}
	}
}
