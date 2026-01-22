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
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
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

func TestOutputJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testData := map[string]string{
		"decision": "block",
		"reason":   "test reason",
	}

	outputJSON(testData)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("outputJSON produced invalid JSON: %v", err)
	}

	if result["decision"] != "block" {
		t.Errorf("decision = %v, want block", result["decision"])
	}
	if result["reason"] != "test reason" {
		t.Errorf("reason = %v, want test reason", result["reason"])
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
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
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
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
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
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
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
