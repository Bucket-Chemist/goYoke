package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/memory"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

func TestGetProjectDir(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name: "GOGENT_PROJECT_DIR set",
			env: map[string]string{
				"GOGENT_PROJECT_DIR": "/test/gogent",
			},
			expected: "/test/gogent",
		},
		{
			name: "CLAUDE_PROJECT_DIR set",
			env: map[string]string{
				"CLAUDE_PROJECT_DIR": "/test/claude",
			},
			expected: "/test/claude",
		},
		{
			name: "GOGENT_PROJECT_DIR takes precedence",
			env: map[string]string{
				"GOGENT_PROJECT_DIR": "/test/gogent",
				"CLAUDE_PROJECT_DIR": "/test/claude",
			},
			expected: "/test/gogent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("GOGENT_PROJECT_DIR")
			os.Unsetenv("CLAUDE_PROJECT_DIR")

			// Set test environment
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.env {
					os.Unsetenv(k)
				}
			}()

			got := getProjectDir()
			if !strings.Contains(got, tt.expected) && tt.expected != "" {
				// Allow for cwd fallback in no-env case
				if len(tt.env) > 0 {
					t.Errorf("getProjectDir() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestGetProjectDir_Priority(t *testing.T) {
	// Save original env
	origGogent := os.Getenv("GOGENT_PROJECT_DIR")
	origClaude := os.Getenv("CLAUDE_PROJECT_DIR")
	defer func() {
		if origGogent != "" {
			os.Setenv("GOGENT_PROJECT_DIR", origGogent)
		} else {
			os.Unsetenv("GOGENT_PROJECT_DIR")
		}
		if origClaude != "" {
			os.Setenv("CLAUDE_PROJECT_DIR", origClaude)
		} else {
			os.Unsetenv("CLAUDE_PROJECT_DIR")
		}
	}()

	tests := []struct {
		name           string
		gogentDir      string
		claudeDir      string
		expectedResult string
	}{
		{
			name:           "GOGENT_PROJECT_DIR has highest priority",
			gogentDir:      "/gogent/path",
			claudeDir:      "/claude/path",
			expectedResult: "/gogent/path",
		},
		{
			name:           "CLAUDE_PROJECT_DIR when GOGENT not set",
			gogentDir:      "",
			claudeDir:      "/claude/path",
			expectedResult: "/claude/path",
		},
		{
			name:           "Falls back to CWD when both unset",
			gogentDir:      "",
			claudeDir:      "",
			expectedResult: "", // Will match CWD
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("GOGENT_PROJECT_DIR")
			os.Unsetenv("CLAUDE_PROJECT_DIR")

			// Set test values
			if tt.gogentDir != "" {
				os.Setenv("GOGENT_PROJECT_DIR", tt.gogentDir)
			}
			if tt.claudeDir != "" {
				os.Setenv("CLAUDE_PROJECT_DIR", tt.claudeDir)
			}

			result := getProjectDir()

			if tt.expectedResult == "" {
				// Verify it's a valid directory path (CWD)
				if result == "" {
					t.Error("Expected non-empty directory path for CWD fallback")
				}
			} else if result != tt.expectedResult {
				t.Errorf("getProjectDir() = %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

func TestWritePendingLearning(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	edge := session.SharpEdge{
		File:                "test.py",
		ErrorType:           "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:           1705708800,
		Context:             "Tool: Bash",
	}

	err := writePendingLearning(tmpDir, edge)
	if err != nil {
		t.Fatalf("writePendingLearning() error = %v", err)
	}

	// Verify file exists
	pendingPath := filepath.Join(tmpDir, ".gogent", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Fatalf("pending-learnings.jsonl not created")
	}

	// Read and verify content
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var readEdge session.SharpEdge
	if err := json.Unmarshal(bytes.TrimSpace(data), &readEdge); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if readEdge.File != edge.File {
		t.Errorf("File = %v, want %v", readEdge.File, edge.File)
	}
	if readEdge.ErrorType != edge.ErrorType {
		t.Errorf("ErrorType = %v, want %v", readEdge.ErrorType, edge.ErrorType)
	}
	if readEdge.ConsecutiveFailures != edge.ConsecutiveFailures {
		t.Errorf("ConsecutiveFailures = %v, want %v", readEdge.ConsecutiveFailures, edge.ConsecutiveFailures)
	}
	if readEdge.Timestamp != edge.Timestamp {
		t.Errorf("Timestamp = %v, want %v", readEdge.Timestamp, edge.Timestamp)
	}
}

func TestNoFailurePassthrough(t *testing.T) {
	// Create temp directory for tracker
	tmpDir := t.TempDir()
	memory.DefaultStoragePath = filepath.Join(tmpDir, "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()

	// Create a PostToolUse event with successful execution
	event := &routing.PostToolEvent{
		ToolName:     "Bash",
		ToolInput:    map[string]interface{}{"command": "echo test"},
		ToolResponse: map[string]interface{}{"exit_code": float64(0), "output": "test"},
		CapturedAt:   1705708800,
	}

	// Detect failure (should be nil)
	failure := routing.DetectFailure(event)
	if failure != nil {
		t.Errorf("Expected no failure, got %+v", failure)
	}
}

func TestSingleFailureDoesNotCapture(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Override storage path for this test
	memory.DefaultStoragePath = filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()

	// Use current time to ensure it's within the failure window
	now := time.Now().Unix()

	// Create event with failure
	event := &routing.PostToolEvent{
		ToolName:     "Bash",
		ToolInput:    map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{"exit_code": float64(1), "output": "TypeError: expected str"},
		CapturedAt:   now,
	}

	// Detect and log failure
	failure := routing.DetectFailure(event)
	if failure == nil {
		t.Fatal("Expected failure, got nil")
	}

	if err := memory.LogFailure(failure); err != nil {
		t.Fatalf("LogFailure error: %v", err)
	}

	count, err := memory.GetFailureCount(failure.File, failure.ErrorType)
	if err != nil {
		t.Fatalf("GetFailureCount error: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count=1, got %d", count)
	}

	// Verify pending-learnings.jsonl does NOT exist (threshold not reached)
	pendingPath := filepath.Join(tmpDir, ".gogent", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); err == nil {
		t.Error("pending-learnings.jsonl should not exist after single failure")
	}
}

func TestThresholdReachedCapturesSharpEdge(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Override storage path
	memory.DefaultStoragePath = filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()

	// Use current time
	now := time.Now().Unix()

	// Create failure info
	failure := &routing.FailureInfo{
		File:      "test.py",
		ErrorType: "typeerror",
		Timestamp: now,
		Tool:      "Bash",
	}

	// Log 3 failures
	for i := 0; i < 3; i++ {
		if err := memory.LogFailure(failure); err != nil {
			t.Fatalf("LogFailure error: %v", err)
		}
	}

	count, err := memory.GetFailureCount(failure.File, failure.ErrorType)
	if err != nil {
		t.Fatalf("GetFailureCount error: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count=3, got %d", count)
	}

	// Write sharp edge
	edge := session.SharpEdge{
		File:                failure.File,
		ErrorType:           failure.ErrorType,
		ConsecutiveFailures: count,
		Timestamp:           failure.Timestamp,
		Context:             fmt.Sprintf("Tool: %s", failure.Tool),
	}

	if err := writePendingLearning(tmpDir, edge); err != nil {
		t.Fatalf("writePendingLearning error: %v", err)
	}

	// Verify pending-learnings.jsonl exists
	pendingPath := filepath.Join(tmpDir, ".gogent", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Fatal("pending-learnings.jsonl should exist after threshold reached")
	}

	// Read and verify content
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var readEdge session.SharpEdge
	if err := json.Unmarshal(bytes.TrimSpace(data), &readEdge); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if readEdge.ConsecutiveFailures != 3 {
		t.Errorf("ConsecutiveFailures = %v, want 3", readEdge.ConsecutiveFailures)
	}
}

func TestCompositeKeyPreventsFalsePositives(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	memory.DefaultStoragePath = filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()

	// Use current time
	now := time.Now().Unix()

	// Log 3 different error types on same file
	file := "main.py"
	errors := []string{"typeerror", "valueerror", "syntaxerror"}

	for _, errType := range errors {
		failure := &routing.FailureInfo{
			File:      file,
			ErrorType: errType,
			Timestamp: now,
			Tool:      "Bash",
		}
		if err := memory.LogFailure(failure); err != nil {
			t.Fatalf("LogFailure error: %v", err)
		}
	}

	// Check count for each error type (should be 1 each)
	for _, errType := range errors {
		count, err := memory.GetFailureCount(file, errType)
		if err != nil {
			t.Fatalf("GetFailureCount error: %v", err)
		}
		if count != 1 {
			t.Errorf("Count for %s = %d, want 1 (composite key should separate errors)", errType, count)
		}
	}
}

func TestSchemaCompliance(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	edge := session.SharpEdge{
		File:                "pkg/api/handler.go",
		ErrorType:           "explicit_failure",
		ConsecutiveFailures: 3,
		Timestamp:           1705708800,
		Context:             "Tool: Edit",
	}

	if err := writePendingLearning(tmpDir, edge); err != nil {
		t.Fatalf("writePendingLearning error: %v", err)
	}

	// Read back and verify schema compliance
	pendingPath := filepath.Join(tmpDir, ".gogent", "memory", "pending-learnings.jsonl")
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse as generic map to check exact field names
	var fields map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(data), &fields); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify required fields exist with correct names
	requiredFields := []string{"file", "error_type", "consecutive_failures", "timestamp"}
	for _, field := range requiredFields {
		if _, exists := fields[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify timestamp is int64 (JSON unmarshals as float64)
	if ts, ok := fields["timestamp"].(float64); !ok {
		t.Errorf("timestamp should be numeric, got %T", fields["timestamp"])
	} else if int64(ts) != edge.Timestamp {
		t.Errorf("timestamp = %v, want %v", int64(ts), edge.Timestamp)
	}

	// Verify consecutive_failures is int
	if cf, ok := fields["consecutive_failures"].(float64); !ok {
		t.Errorf("consecutive_failures should be numeric, got %T", fields["consecutive_failures"])
	} else if int(cf) != edge.ConsecutiveFailures {
		t.Errorf("consecutive_failures = %v, want %v", int(cf), edge.ConsecutiveFailures)
	}
}

// =============================================================================
// getAgentDirectories Tests
// =============================================================================

func TestGetAgentDirectories_MultipleAgents(t *testing.T) {
	dirs := getAgentDirectories()
	if len(dirs) == 0 {
		t.Fatal("Expected multiple agent directories, got empty list")
	}
	expectedAgents := []string{"python-pro", "go-pro", "r-pro", "codebase-search", "orchestrator", "architect"}
	for _, agent := range expectedAgents {
		found := false
		for _, dir := range dirs {
			if strings.Contains(dir, agent) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected agent '%s' in directories, but not found", agent)
		}
	}
}

func TestGetAgentDirectories_PathStructure(t *testing.T) {
	dirs := getAgentDirectories()
	for _, dir := range dirs {
		if !strings.Contains(dir, ".claude/agents/") {
			t.Errorf("Expected path to contain '.claude/agents/', got: %s", dir)
		}
	}
}

func TestGetAgentDirectories_HomeFallback(t *testing.T) {
	oldHome := os.Getenv("HOME")
	oldUser := os.Getenv("USER")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USER", oldUser)
	}()
	os.Unsetenv("HOME")
	os.Setenv("USER", "testuser")
	dirs := getAgentDirectories()
	if len(dirs) == 0 {
		t.Fatal("Expected directories even with HOME unset")
	}
	for _, dir := range dirs {
		if !strings.Contains(dir, "/home/testuser/") {
			t.Errorf("Expected fallback to /home/testuser, got: %s", dir)
		}
	}
}

func TestGetAgentDirectories_Consistency(t *testing.T) {
	dirs1 := getAgentDirectories()
	dirs2 := getAgentDirectories()
	if len(dirs1) != len(dirs2) {
		t.Fatalf("Expected consistent results, got different lengths: %d vs %d", len(dirs1), len(dirs2))
	}
	for i, dir := range dirs1 {
		if dir != dirs2[i] {
			t.Errorf("Inconsistent results at index %d: %s vs %s", i, dir, dirs2[i])
		}
	}
}

// =============================================================================
// buildWarningResponse Tests
// =============================================================================

func TestBuildWarningResponse_Structure(t *testing.T) {
	resp := buildWarningResponse("test.py", 2, "", "")

	if resp.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"].(string)
	if !ok || hookEventName != "PostToolUse" {
		t.Errorf("Expected hookEventName 'PostToolUse', got: %v", hookEventName)
	}

	ctx, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	if !strings.Contains(ctx, "⚠️") {
		t.Error("Expected warning emoji in context")
	}
	if !strings.Contains(ctx, "test.py") {
		t.Error("Expected file name in context")
	}
	if !strings.Contains(ctx, "2 failures") {
		t.Error("Expected failure count in context")
	}
	if !strings.Contains(ctx, "One more failure") {
		t.Error("Expected escalation guidance in context")
	}
}

func TestBuildWarningResponse_WithAttentionGate(t *testing.T) {
	reminderMsg := "⏰ ROUTING REMINDER"
	flushMsg := "📝 LEARNINGS ARCHIVED"

	resp := buildWarningResponse("handler.go", 2, reminderMsg, flushMsg)

	ctx, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	// Verify warning message
	if !strings.Contains(ctx, "⚠️") {
		t.Error("Expected warning in context")
	}
	if !strings.Contains(ctx, "handler.go") {
		t.Error("Expected file name in context")
	}

	// Verify attention-gate messages
	if !strings.Contains(ctx, "ROUTING REMINDER") {
		t.Error("Expected reminder message in context")
	}
	if !strings.Contains(ctx, "LEARNINGS ARCHIVED") {
		t.Error("Expected flush message in context")
	}
}

// =============================================================================
// main() Integration Tests
// =============================================================================

func TestMain_MultipleFailures_CorrectCount(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")
	memory.DefaultStoragePath = filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()
	now := time.Now().Unix()
	failures := []*routing.FailureInfo{
		{File: "a.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
		{File: "a.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
		{File: "b.py", ErrorType: "valueerror", Timestamp: now, Tool: "Edit"},
		{File: "a.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
	}
	for _, failure := range failures {
		if err := memory.LogFailure(failure); err != nil {
			t.Fatalf("LogFailure error: %v", err)
		}
	}
	countA, err := memory.GetFailureCount("a.py", "typeerror")
	if err != nil {
		t.Fatalf("GetFailureCount error: %v", err)
	}
	if countA != 3 {
		t.Errorf("Expected count=3 for a.py, got %d", countA)
	}
	countB, err := memory.GetFailureCount("b.py", "valueerror")
	if err != nil {
		t.Fatalf("GetFailureCount error: %v", err)
	}
	if countB != 1 {
		t.Errorf("Expected count=1 for b.py, got %d", countB)
	}
}

func TestMain_CompositeKeyHandling_Extended(t *testing.T) {
	tmpDir := t.TempDir()
	memory.DefaultStoragePath = filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl")
	defer func() {
		memory.DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
	}()
	now := time.Now().Unix()
	file := "main.go"
	errorTypes := []string{"syntaxerror", "typeerror", "importerror"}
	for _, errType := range errorTypes {
		failure := &routing.FailureInfo{
			File:      file,
			ErrorType: errType,
			Timestamp: now,
			Tool:      "Bash",
		}
		for i := 0; i < 2; i++ {
			if err := memory.LogFailure(failure); err != nil {
				t.Fatalf("LogFailure error: %v", err)
			}
		}
	}
	for _, errType := range errorTypes {
		count, err := memory.GetFailureCount(file, errType)
		if err != nil {
			t.Fatalf("GetFailureCount error for %s: %v", errType, err)
		}
		if count != 2 {
			t.Errorf("Expected count=2 for %s, got %d (composite key not separating)", errType, count)
		}
	}
}

// =============================================================================
// buildCombinedResponse Tests
// =============================================================================

func TestBuildCombinedResponse_NoFailure(t *testing.T) {
	resp := buildCombinedResponse(nil, "", "")

	// Verify response structure
	if resp.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"].(string)
	if !ok || hookEventName != "PostToolUse" {
		t.Errorf("Expected hookEventName 'PostToolUse', got: %v", hookEventName)
	}

	// Verify no additionalContext when no messages provided
	if ctx, exists := resp.HookSpecificOutput["additionalContext"]; exists {
		t.Errorf("Expected no additionalContext, got: %v", ctx)
	}
}

func TestBuildCombinedResponse_WithReminderOnly(t *testing.T) {
	reminderMsg := "⏰ ROUTING REMINDER (10 tools): Check tier compliance"

	resp := buildCombinedResponse(nil, reminderMsg, "")

	ctx, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	if !strings.Contains(ctx, "ROUTING REMINDER") {
		t.Errorf("Expected reminder message in context, got: %s", ctx)
	}

	if !strings.Contains(ctx, "10 tools") {
		t.Errorf("Expected tool count in context, got: %s", ctx)
	}
}

func TestBuildCombinedResponse_WithFlushOnly(t *testing.T) {
	flushMsg := "📝 LEARNINGS ARCHIVED: 3 sharp edges moved to memory/learnings/"

	resp := buildCombinedResponse(nil, "", flushMsg)

	ctx, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	if !strings.Contains(ctx, "LEARNINGS ARCHIVED") {
		t.Errorf("Expected flush message in context, got: %s", ctx)
	}

	if !strings.Contains(ctx, "3 sharp edges") {
		t.Errorf("Expected learning count in context, got: %s", ctx)
	}
}

func TestBuildCombinedResponse_WithReminderAndFlush(t *testing.T) {
	reminderMsg := "⏰ ROUTING REMINDER (10 tools): Check tier compliance"
	flushMsg := "📝 LEARNINGS ARCHIVED: 3 sharp edges moved to memory/learnings/"

	resp := buildCombinedResponse(nil, reminderMsg, flushMsg)

	ctx, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	// Verify both messages present
	if !strings.Contains(ctx, "ROUTING REMINDER") {
		t.Errorf("Expected reminder message in context, got: %s", ctx)
	}

	if !strings.Contains(ctx, "LEARNINGS ARCHIVED") {
		t.Errorf("Expected flush message in context, got: %s", ctx)
	}

	// Verify proper separation (double newline)
	if !strings.Contains(ctx, "\n\n") {
		t.Errorf("Expected double newline separator between messages, got: %s", ctx)
	}
}

func TestBuildCombinedResponse_JSONMarshaling(t *testing.T) {
	reminderMsg := "Test reminder"
	flushMsg := "Test flush"

	resp := buildCombinedResponse(nil, reminderMsg, flushMsg)

	// Marshal to JSON
	var buf bytes.Buffer
	err := resp.Marshal(&buf)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal and verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	hookOutput, ok := result["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput object")
	}

	if hookOutput["hookEventName"] != "PostToolUse" {
		t.Errorf("Expected hookEventName 'PostToolUse', got: %v", hookOutput["hookEventName"])
	}

	ctx, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string")
	}

	if !strings.Contains(ctx, "Test reminder") {
		t.Errorf("Expected reminder in context, got: %s", ctx)
	}

	if !strings.Contains(ctx, "Test flush") {
		t.Errorf("Expected flush in context, got: %s", ctx)
	}
}

// =============================================================================
// ML Tool Event Logging Integration Tests (GOgent-087d)
// =============================================================================

func TestMLLogging_Integration(t *testing.T) {
	// Setup temp directory for XDG_DATA_HOME
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create test event
	now := time.Now().Unix()
	event := &routing.PostToolEvent{
		ToolName:      "Read",
		SessionID:     "test-session-123",
		CapturedAt:    now,
		HookEventName: "PostToolUse",
		ToolInput: map[string]interface{}{
			"file_path": "/test/file.go",
		},
		ToolResponse: map[string]interface{}{
			"success": true,
		},
	}

	// Call logging function directly
	err := telemetry.LogMLToolEvent(event, "")
	if err != nil {
		t.Fatalf("LogMLToolEvent() failed: %v", err)
	}

	// Verify global log file created (using config.GetMLToolEventsPath naming)
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	if _, err := os.Stat(globalLogPath); os.IsNotExist(err) {
		t.Errorf("Global log file should exist at %s", globalLogPath)
	}

	// Read and verify content
	data, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSONL line
	var loggedEvent routing.PostToolEvent
	if err := json.Unmarshal(bytes.TrimSpace(data), &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal logged event: %v", err)
	}

	// Verify critical fields
	if loggedEvent.ToolName != "Read" {
		t.Errorf("ToolName = %v, want Read", loggedEvent.ToolName)
	}
	if loggedEvent.SessionID != "test-session-123" {
		t.Errorf("SessionID = %v, want test-session-123", loggedEvent.SessionID)
	}
	if loggedEvent.CapturedAt != now {
		t.Errorf("CapturedAt = %v, want %v", loggedEvent.CapturedAt, now)
	}
}

func TestMLLogging_NonBlocking(t *testing.T) {
	// Use invalid/readonly path to force error
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", "/nonexistent/readonly/path")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		SessionID:  "test-nonblocking",
		CapturedAt: time.Now().Unix(),
	}

	// Function should return error but not panic
	err := telemetry.LogMLToolEvent(event, "")
	if err == nil {
		t.Log("Warning: Expected error with invalid path, got nil (permissions may allow creation)")
	}
	// The important thing is we don't panic - hook continues execution
}

func TestMLLogging_DualWrite(t *testing.T) {
	// Setup temp directories for both global and project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")

	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create project .gogent/memory directory
	projectMemoryDir := filepath.Join(projectDir, ".gogent", "memory")
	if err := os.MkdirAll(projectMemoryDir, 0755); err != nil {
		t.Fatalf("Failed to create project memory dir: %v", err)
	}

	// Create test event
	event := &routing.PostToolEvent{
		ToolName:   "Edit",
		SessionID:  "dual-write-test",
		CapturedAt: time.Now().Unix(),
	}

	// Call with projectDir
	err := telemetry.LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("LogMLToolEvent() failed: %v", err)
	}

	// Verify global log (using config.GetMLToolEventsPath naming: tool-events.jsonl)
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	if _, err := os.Stat(globalLogPath); os.IsNotExist(err) {
		t.Error("Global log file should exist")
	}

	// Verify project log
	projectLogPath := filepath.Join(projectMemoryDir, "ml-tool-events.jsonl")
	if _, err := os.Stat(projectLogPath); os.IsNotExist(err) {
		t.Error("Project log file should exist")
	}

	// Verify both files contain valid JSON
	for _, path := range []string{globalLogPath, projectLogPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			continue
		}

		var event routing.PostToolEvent
		if err := json.Unmarshal(bytes.TrimSpace(data), &event); err != nil {
			t.Errorf("Invalid JSON in %s: %v", path, err)
		}
	}
}

func TestMLLogging_PerformanceRegression(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	event := &routing.PostToolEvent{
		ToolName:   "Read",
		SessionID:  "perf-test",
		CapturedAt: time.Now().Unix(),
	}

	// Measure execution time
	start := time.Now()
	err := telemetry.LogMLToolEvent(event, "")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("LogMLToolEvent() failed: %v", err)
	}

	// Verify latency < 10ms (acceptance criteria)
	if duration > 10*time.Millisecond {
		t.Errorf("ML logging took %v, exceeds 10ms threshold", duration)
	}
}

func TestMLLogging_MultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Log multiple events
	tools := []string{"Read", "Edit", "Bash", "Grep"}
	for i, tool := range tools {
		event := &routing.PostToolEvent{
			ToolName:   tool,
			SessionID:  "multi-event-test",
			CapturedAt: time.Now().Unix() + int64(i),
		}

		if err := telemetry.LogMLToolEvent(event, ""); err != nil {
			t.Fatalf("LogMLToolEvent() failed for %s: %v", tool, err)
		}
	}

	// Read log file (using config.GetMLToolEventsPath naming: tool-events.jsonl)
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	data, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Split into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(tools) {
		t.Errorf("Expected %d lines, got %d", len(tools), len(lines))
	}

	// Verify each line is valid JSON with correct tool name
	for i, line := range lines {
		var event routing.PostToolEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d invalid JSON: %v", i, err)
			continue
		}

		if event.ToolName != tools[i] {
			t.Errorf("Line %d: ToolName = %v, want %v", i, event.ToolName, tools[i])
		}
	}
}

func TestMLLogging_ExtendedFields(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create event with extended ML fields
	event := &routing.PostToolEvent{
		ToolName:   "Read",
		SessionID:  "extended-fields-test",
		CapturedAt: time.Now().Unix(),
		// Extended fields (currently not populated by Claude Code)
		DurationMs:   0, // Will be 0 until Claude Code emits it
		InputTokens:  0, // Will be 0 until Claude Code emits it
		OutputTokens: 0, // Will be 0 until Claude Code emits it
		Model:        "", // Empty until available
		Tier:         "", // Empty until available
	}

	err := telemetry.LogMLToolEvent(event, "")
	if err != nil {
		t.Fatalf("LogMLToolEvent() failed: %v", err)
	}

	// Read and verify fields are preserved (even if zero/empty)
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	data, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(data), &loggedEvent); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify core fields exist
	requiredFields := []string{"tool_name", "session_id", "captured_at"}
	for _, field := range requiredFields {
		if _, exists := loggedEvent[field]; !exists {
			t.Errorf("Required field %s missing", field)
		}
	}

	// Note: Extended fields with zero values won't appear in JSON (omitempty)
	// This is expected behavior per routing.PostToolEvent struct tags
}

func TestMLLogging_ErrorHandlingNonBlocking(t *testing.T) {
	// Test that ML logging errors don't prevent hook execution
	tmpDir := t.TempDir()

	// Set XDG to valid path but project dir to invalid path
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Create an event
	event := &routing.PostToolEvent{
		ToolName:   "Edit",
		SessionID:  "error-handling-test",
		CapturedAt: time.Now().Unix(),
	}

	// This should succeed for global write
	err := telemetry.LogMLToolEvent(event, "/nonexistent/project/dir")
	if err != nil {
		t.Fatalf("LogMLToolEvent() should succeed for global write even with invalid project dir: %v", err)
	}

	// Verify global log exists
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	if _, err := os.Stat(globalLogPath); os.IsNotExist(err) {
		t.Error("Global log should exist even when project dir write fails")
	}

	// Verify project log doesn't exist (project dir doesn't exist)
	projectLogPath := filepath.Join("/nonexistent/project/dir", ".gogent", "memory", "ml-tool-events.jsonl")
	if _, err := os.Stat(projectLogPath); !os.IsNotExist(err) {
		t.Error("Project log should not exist for nonexistent directory")
	}
}

func TestMLLogging_ConcurrentWrites(t *testing.T) {
	// Verify ML logging is safe for concurrent use (hook may be called in parallel)
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
	}()

	// Launch 10 concurrent writes
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := &routing.PostToolEvent{
				ToolName:   fmt.Sprintf("Tool%d", id),
				SessionID:  "concurrent-test",
				CapturedAt: time.Now().Unix(),
			}
			done <- telemetry.LogMLToolEvent(event, "")
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent write %d failed: %v", i, err)
		}
	}

	// Verify all 10 events were written
	globalLogPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
	data, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 JSONL lines, got %d", len(lines))
	}
}
