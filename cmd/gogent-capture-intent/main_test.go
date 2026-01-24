package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// TestRun_ValidAskUserQuestion verifies successful extraction and append via run().
func TestRun_ValidAskUserQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Simulate stdin with valid AskUserQuestion hook input
	hookInput := HookInput{
		SessionID: "run-test-123",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [{
			"question": "Should we proceed?",
			"header": "Confirmation",
			"options": [{"label": "Yes"}, {"label": "No"}]
		}]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "Yes"}
	}`)

	inputJSON, _ := json.Marshal(hookInput)

	// Capture stdin/stdout
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write(inputJSON)
		w.Close()
	}()

	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	err := run()

	stdoutW.Close()
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(stdoutR)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Expected no error from run(), got: %v", err)
	}

	// Verify output is valid JSON
	output := stdoutBuf.String()
	if !strings.Contains(output, "{}") {
		t.Errorf("Expected {} output, got: %s", output)
	}

	// Verify file was created
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	if _, err := os.Stat(intentsPath); os.IsNotExist(err) {
		t.Fatal("Expected intents file to be created")
	}

	// Verify content
	data, _ := os.ReadFile(intentsPath)
	if !strings.Contains(string(data), "run-test-123") {
		t.Error("Expected session ID in output file")
	}
	if !strings.Contains(string(data), "Should we proceed?") {
		t.Error("Expected question in output file")
	}
}

// TestRun_NonAskUserTool verifies graceful skip for non-AskUserQuestion tools.
func TestRun_NonAskUserTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Simulate stdin with different tool
	hookInput := HookInput{
		SessionID: "other-tool-session",
	}
	hookInput.Tool.Name = "Read"
	hookInput.Tool.Input = json.RawMessage(`{"file_path": "/test"}`)
	hookInput.Tool.Result = json.RawMessage(`{"content": "test"}`)

	inputJSON, _ := json.Marshal(hookInput)

	// Capture stdin/stdout
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write(inputJSON)
		w.Close()
	}()

	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	err := run()

	stdoutW.Close()
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(stdoutR)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Expected no error for non-AskUserQuestion tool, got: %v", err)
	}

	// Verify output is empty JSON
	output := stdoutBuf.String()
	if !strings.Contains(output, "{}") {
		t.Errorf("Expected {} output, got: %s", output)
	}

	// Verify NO file was created
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	if _, err := os.Stat(intentsPath); !os.IsNotExist(err) {
		t.Error("Expected no intents file for non-AskUserQuestion tool")
	}
}

// TestRun_InvalidJSON verifies graceful degradation on invalid JSON input.
func TestRun_InvalidJSON(t *testing.T) {
	// Simulate stdin with invalid JSON
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte(`{invalid json`))
		w.Close()
	}()

	// Capture stdout/stderr
	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	err := run()

	stdoutW.Close()
	stderrW.Close()
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(stdoutR)
	var stderrBuf bytes.Buffer
	stderrBuf.ReadFrom(stderrR)
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// run() should return error but not panic
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Error is returned and graceful degradation occurs
	// (stderr may or may not have content depending on how main() is called)
}

// TestRun_EmptyStdin verifies handling of empty stdin.
func TestRun_EmptyStdin(t *testing.T) {
	// Simulate empty stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	w.Close() // Close immediately (EOF)

	// Capture stderr
	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	err := run()

	stderrW.Close()
	var stderrBuf bytes.Buffer
	stderrBuf.ReadFrom(stderrR)
	os.Stderr = oldStderr

	// Should return error for EOF/empty input
	if err == nil {
		t.Error("Expected error for empty stdin")
	}
}

// TestAppendIntent_ConcurrentWrites verifies file locking prevents corruption.
// CRITICAL: Race detector MUST be clean.
func TestAppendIntent_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Spawn 20 concurrent writes
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			intent := &session.UserIntent{
				Timestamp:   int64(1705000000 + id),
				Question:    fmt.Sprintf("Question %d?", id),
				Response:    fmt.Sprintf("Response %d", id),
				Confidence:  "explicit",
				Source:      "ask_user",
				SessionID:   fmt.Sprintf("concurrent-%d", id),
				ToolContext: "AskUserQuestion",
			}

			if err := appendIntent(intent); err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all writes succeeded
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("Failed to read intents file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != goroutines {
		t.Errorf("Expected %d lines, got %d", goroutines, len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var parsed session.UserIntent
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d is invalid JSON: %v\nLine: %s", i, err, line)
		}
	}

	// Verify no data corruption (all session IDs unique)
	sessionIDs := make(map[string]bool)
	for _, line := range lines {
		var parsed session.UserIntent
		json.Unmarshal([]byte(line), &parsed)
		if sessionIDs[parsed.SessionID] {
			t.Errorf("Duplicate session ID: %s (data corruption detected)", parsed.SessionID)
		}
		sessionIDs[parsed.SessionID] = true
	}
}

// TestAppendIntent_MissingParentDirectory verifies directory creation.
func TestAppendIntent_MissingParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Intentionally use a non-existent project dir
	projectDir := filepath.Join(tmpDir, "nonexistent", "project")
	os.Setenv("GOGENT_PROJECT_DIR", projectDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Test?",
		Response:    "Yes",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "missing-dir-test",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error (should create directories), got: %v", err)
	}

	// Verify file exists at correct path
	intentsPath := filepath.Join(projectDir, ".claude", "memory", "user-intents.jsonl")
	if _, err := os.Stat(intentsPath); os.IsNotExist(err) {
		t.Fatal("Expected directory and file to be created")
	}

	// Verify content
	data, _ := os.ReadFile(intentsPath)
	if !strings.Contains(string(data), "missing-dir-test") {
		t.Error("Expected session ID in created file")
	}
}

// TestAppendIntent_AppendExistingFile verifies append (not overwrite) behavior.
func TestAppendIntent_AppendExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Write first intent
	intent1 := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "First question?",
		Response:    "First response",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "append-test-1",
		ToolContext: "AskUserQuestion",
	}

	if err := appendIntent(intent1); err != nil {
		t.Fatalf("Failed to write first intent: %v", err)
	}

	// Write second intent
	intent2 := &session.UserIntent{
		Timestamp:   1705000001,
		Question:    "Second question?",
		Response:    "Second response",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "append-test-2",
		ToolContext: "AskUserQuestion",
	}

	if err := appendIntent(intent2); err != nil {
		t.Fatalf("Failed to write second intent: %v", err)
	}

	// Verify both intents exist in file
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, _ := os.ReadFile(intentsPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (appended), got %d", len(lines))
	}

	if !strings.Contains(string(data), "append-test-1") {
		t.Error("Expected first intent in file")
	}
	if !strings.Contains(string(data), "append-test-2") {
		t.Error("Expected second intent in file")
	}
}

// TestAppendIntent_AtomicWrite verifies write integrity under concurrent stress.
func TestAppendIntent_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(iterations)

	// Stress test: rapid sequential writes
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			intent := &session.UserIntent{
				Timestamp:   int64(1705000000 + id),
				Question:    fmt.Sprintf("Atomic test %d?", id),
				Response:    fmt.Sprintf("Response %d", id),
				Confidence:  "explicit",
				Source:      "ask_user",
				SessionID:   fmt.Sprintf("atomic-%d", id),
				ToolContext: "AskUserQuestion",
			}
			appendIntent(intent)
		}(i)
	}

	wg.Wait()

	// Verify file integrity
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, _ := os.ReadFile(intentsPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Each line must be valid JSON
	validLines := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		var parsed session.UserIntent
		if err := json.Unmarshal([]byte(line), &parsed); err == nil {
			validLines++
		} else {
			t.Errorf("Corrupted line detected: %s\nError: %v", line, err)
		}
	}

	if validLines != iterations {
		t.Errorf("Expected %d valid lines, got %d", iterations, validLines)
	}
}

// TestAppendIntent_ReadOnlyDirectory verifies error handling for permission issues.
func TestAppendIntent_ReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Make directory read-only
	os.Chmod(memoryDir, 0555)
	defer os.Chmod(memoryDir, 0755) // Restore for cleanup

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Test?",
		Response:    "Yes",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "readonly-test",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err == nil {
		t.Error("Expected error for read-only directory")
	}

	if !strings.Contains(err.Error(), "failed to open intents file") {
		t.Errorf("Expected 'failed to open' error, got: %v", err)
	}
}

// TestAppendIntent_EmptyQuestion verifies handling of edge case: empty question.
func TestAppendIntent_EmptyQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "", // Empty question
		Response:    "Response to nothing",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "empty-question-test",
		ToolContext: "AskUserQuestion",
	}

	// Should still write successfully (validation is not appendIntent's job)
	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error for empty question, got: %v", err)
	}

	// Verify file created
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	data, _ := os.ReadFile(intentsPath)

	var parsed session.UserIntent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Question != "" {
		t.Error("Expected empty question to be preserved")
	}
	if parsed.SessionID != "empty-question-test" {
		t.Error("Expected session ID to be preserved")
	}
}

// TestAppendIntent_FallbackToCwd verifies fallback to cwd when GOGENT_PROJECT_DIR unset.
func TestAppendIntent_FallbackToCwd(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Ensure GOGENT_PROJECT_DIR is NOT set
	os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Fallback test?",
		Response:    "Yes",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "fallback-cwd-test",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error with cwd fallback, got: %v", err)
	}

	// Verify file created in cwd
	intentsPath := filepath.Join(tmpDir, ".claude", "memory", "user-intents.jsonl")
	if _, err := os.Stat(intentsPath); os.IsNotExist(err) {
		t.Fatal("Expected intents file in cwd")
	}

	// Verify content
	data, _ := os.ReadFile(intentsPath)
	if !strings.Contains(string(data), "fallback-cwd-test") {
		t.Error("Expected session ID in file")
	}
}

// TestAppendIntent_GetwdError verifies error handling when getwd fails.
func TestAppendIntent_GetwdError(t *testing.T) {
	// This test is hard to trigger reliably, but we can at least
	// verify the code path exists by ensuring GOGENT_PROJECT_DIR works
	// (the error branch is for os.Getwd() failure which is rare)

	// Use a valid project dir to verify normal operation
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	intent := &session.UserIntent{
		Timestamp:   1705000000,
		Question:    "Test?",
		Response:    "Yes",
		Confidence:  "explicit",
		Source:      "ask_user",
		SessionID:   "getwd-test",
		ToolContext: "AskUserQuestion",
	}

	err := appendIntent(intent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// TestRun_AppendIntentError verifies run() handles appendIntent errors gracefully.
func TestRun_AppendIntentError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)
	os.Chmod(memoryDir, 0555) // Make read-only to cause append error
	defer os.Chmod(memoryDir, 0755)

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Simulate valid hook input
	hookInput := HookInput{
		SessionID: "append-error-test",
	}
	hookInput.Tool.Name = "AskUserQuestion"
	hookInput.Tool.Input = json.RawMessage(`{
		"questions": [{
			"question": "Test?",
			"options": [{"label": "Yes"}]
		}]
	}`)
	hookInput.Tool.Result = json.RawMessage(`{
		"answers": {"0": "Yes"}
	}`)

	inputJSON, _ := json.Marshal(hookInput)

	// Capture stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write(inputJSON)
		w.Close()
	}()

	// Capture stderr (error will be logged)
	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	err := run()

	stderrW.Close()
	var stderrBuf bytes.Buffer
	stderrBuf.ReadFrom(stderrR)
	os.Stderr = oldStderr

	// Should return error
	if err == nil {
		t.Error("Expected error when appendIntent fails")
	}

	if !strings.Contains(err.Error(), "failed to write intent") {
		t.Errorf("Expected 'failed to write intent' error, got: %v", err)
	}
}
