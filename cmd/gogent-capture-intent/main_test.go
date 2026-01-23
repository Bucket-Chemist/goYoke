package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

func TestExtractIntent_SingleQuestionWithOptions(t *testing.T) {
	hookInput := HookInput{
		SessionID: "test-session-123",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [{
			"question": "Which approach should we use?",
			"header": "Implementation Strategy",
			"options": [
				{"label": "Option A", "description": "Fast but complex"},
				{"label": "Option B", "description": "Slow but simple"}
			]
		}]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "Option A"}
	}`)

	intent, err := extractIntent(hookInput)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if intent.Question != "Which approach should we use?" {
		t.Errorf("Expected question 'Which approach should we use?', got: %s", intent.Question)
	}
	if intent.Response != "Option A" {
		t.Errorf("Expected response 'Option A', got: %s", intent.Response)
	}
	if intent.Confidence != "explicit" {
		t.Errorf("Expected confidence 'explicit', got: %s", intent.Confidence)
	}
	if intent.Context != "Implementation Strategy" {
		t.Errorf("Expected context 'Implementation Strategy', got: %s", intent.Context)
	}
	if intent.Source != "ask_user" {
		t.Errorf("Expected source 'ask_user', got: %s", intent.Source)
	}
	if intent.SessionID != "test-session-123" {
		t.Errorf("Expected session ID 'test-session-123', got: %s", intent.SessionID)
	}
	if intent.ToolContext != "AskUserQuestion" {
		t.Errorf("Expected tool context 'AskUserQuestion', got: %s", intent.ToolContext)
	}
	if intent.Timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestExtractIntent_FreeFormQuestion(t *testing.T) {
	hookInput := HookInput{
		SessionID: "test-session-456",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [{
			"question": "What should we call this feature?",
			"header": "Naming Convention"
		}]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "MyAwesomeFeature"}
	}`)

	intent, err := extractIntent(hookInput)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if intent.Question != "What should we call this feature?" {
		t.Errorf("Expected question 'What should we call this feature?', got: %s", intent.Question)
	}
	if intent.Response != "MyAwesomeFeature" {
		t.Errorf("Expected response 'MyAwesomeFeature', got: %s", intent.Response)
	}
	if intent.Confidence != "inferred" {
		t.Errorf("Expected confidence 'inferred' for free-form, got: %s", intent.Confidence)
	}
	if intent.Context != "Naming Convention" {
		t.Errorf("Expected context 'Naming Convention', got: %s", intent.Context)
	}
	if intent.SessionID != "test-session-456" {
		t.Errorf("Expected session ID 'test-session-456', got: %s", intent.SessionID)
	}
}

func TestExtractIntent_NoQuestions(t *testing.T) {
	hookInput := HookInput{
		SessionID: "test-session-789",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": []
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {}
	}`)

	_, err := extractIntent(hookInput)
	if err == nil {
		t.Error("Expected error for empty questions array")
	}
	if !strings.Contains(err.Error(), "no questions") {
		t.Errorf("Expected 'no questions' error, got: %v", err)
	}
}

func TestExtractIntent_MultipleQuestions(t *testing.T) {
	// The implementation takes only the first question
	hookInput := HookInput{
		SessionID: "test-session-multi",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [
			{
				"question": "First question?",
				"header": "First Context"
			},
			{
				"question": "Second question?",
				"header": "Second Context"
			}
		]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "First answer", "1": "Second answer"}
	}`)

	intent, err := extractIntent(hookInput)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should extract only first question
	if intent.Question != "First question?" {
		t.Errorf("Expected first question, got: %s", intent.Question)
	}
	if intent.Context != "First Context" {
		t.Errorf("Expected first context, got: %s", intent.Context)
	}
}

func TestAppendIntent_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Test question?",
		Response:    "Test response",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "test-123",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file exists
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	if _, err := os.Stat(intentsPath); os.IsNotExist(err) {
		t.Fatal("Expected file to be created")
	}

	// Verify file contents
	data, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test-123") {
		t.Error("Expected session ID in file")
	}
	if !strings.Contains(content, "Test question?") {
		t.Error("Expected question in file")
	}
	if !strings.Contains(content, "Test response") {
		t.Error("Expected response in file")
	}
	if !strings.Contains(content, "AskUserQuestion") {
		t.Error("Expected tool context in file")
	}

	// Verify it's valid JSON
	var parsed session.UserIntent
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	// Verify fields match
	if parsed.SessionID != "test-123" {
		t.Errorf("Expected session ID 'test-123', got: %s", parsed.SessionID)
	}
	if parsed.Question != "Test question?" {
		t.Errorf("Expected question 'Test question?', got: %s", parsed.Question)
	}
}

func TestAppendIntent_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Write multiple intents
	for i := 0; i < 3; i++ {
		intent := &session.UserIntent{
			Timestamp:   int64(1705000000 + i),
			Question:    "Question " + string(rune('A'+i)) + "?",
			Response:    "Response " + string(rune('A'+i)),
			Confidence:  "explicit",
			Source:      "ask_user",
			SessionID:   "test-multi",
			ToolContext: "AskUserQuestion",
		}

		if err := appendIntent(intent); err != nil {
			t.Fatalf("Failed to append intent %d: %v", i, err)
		}
	}

	// Verify file contains all intents
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON and contains expected data
	expectedQuestions := []string{"Question A?", "Question B?", "Question C?"}
	for i, line := range lines {
		var parsed session.UserIntent
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Failed to parse line %d: %v", i, err)
		}
		if parsed.Question != expectedQuestions[i] {
			t.Errorf("Line %d: expected question '%s', got '%s'", i, expectedQuestions[i], parsed.Question)
		}
	}
}

func TestAppendIntent_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Test?",
		Response:    "Yes",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "test-perms",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file permissions are 0644
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	info, err := os.Stat(intentsPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0644)
	if mode.Perm() != expectedMode {
		t.Errorf("Expected file mode %o, got %o", expectedMode, mode.Perm())
	}
}

func TestRun_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Simulate full hook input via environment manipulation
	// Note: This test verifies the integration but uses direct function calls
	// for testability (mocking stdin is complex)

	hookInput := HookInput{
		SessionID: "integration-test-789",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [{
			"question": "Integration test question?",
			"header": "Testing",
			"options": [{"label": "Yes"}, {"label": "No"}]
		}]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "Yes"}
	}`)

	// Extract and append
	intent, err := extractIntent(hookInput)
	if err != nil {
		t.Fatalf("Integration test failed at extraction: %v", err)
	}

	err = appendIntent(intent)
	if err != nil {
		t.Fatalf("Integration test failed at append: %v", err)
	}

	// Verify end-to-end result
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("Integration test: failed to read result: %v", err)
	}

	var parsed session.UserIntent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Integration test: invalid JSON: %v", err)
	}

	if parsed.SessionID != "integration-test-789" {
		t.Errorf("Integration test: expected session ID 'integration-test-789', got: %s", parsed.SessionID)
	}
	if parsed.Question != "Integration test question?" {
		t.Errorf("Integration test: unexpected question: %s", parsed.Question)
	}
	if parsed.Response != "Yes" {
		t.Errorf("Integration test: unexpected response: %s", parsed.Response)
	}
	if parsed.Confidence != "explicit" {
		t.Errorf("Integration test: unexpected confidence: %s", parsed.Confidence)
	}
}
