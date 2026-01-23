package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/memory"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestProject creates a temporary test project directory structure
func setupTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create required directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".claude", "memory"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".gogent"), 0755))

	return tmpDir
}

// createTrackerEntries pre-populates the failure tracker with entries
// Note: Call this AFTER setting up memory.DefaultStoragePath
func createTrackerEntries(t *testing.T, entries []routing.FailureInfo) {
	t.Helper()

	for _, entry := range entries {
		require.NoError(t, memory.LogFailure(&entry))
	}
}

// TestWorkflow_SingleFailure_PassThrough tests that a single failure is logged but doesn't trigger
func TestWorkflow_SingleFailure_PassThrough(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	// Set environment
	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	// Override default path
	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Simulate first failure
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{
			"exit_code": 1.0,
			"output":    "TypeError: foo",
		},
		CapturedAt: time.Now().Unix(),
	}

	// Detect and log failure
	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	require.NoError(t, memory.LogFailure(info))

	// Verify logged
	count, err := memory.GetFailureCount(info.File, info.ErrorType)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should have 1 failure logged")

	// Verify not debugging loop yet
	isLoop, err := memory.IsDebuggingLoop(info.File, info.ErrorType)
	require.NoError(t, err)
	assert.False(t, isLoop, "Should not be debugging loop at 1 failure")
}

// TestWorkflow_TwoFailures_Warning tests that 2 failures trigger warning state
func TestWorkflow_TwoFailures_Warning(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Pre-populate with 1 failure (general_error from exit code 1)
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "general_error", Timestamp: now - 10, Tool: "Bash"},
	})

	// Simulate second failure (exit code 1 produces general_error)
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{
			"exit_code": 1.0,
			"output":    "TypeError: foo",
		},
		CapturedAt: now,
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	require.NoError(t, memory.LogFailure(info))

	// Verify count
	count, err := memory.GetFailureCount("test.py", "general_error")
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should have 2 failures")

	// Still not debugging loop
	isLoop, err := memory.IsDebuggingLoop("test.py", "general_error")
	require.NoError(t, err)
	assert.False(t, isLoop, "Should not be debugging loop at 2 failures")
}

// TestWorkflow_ThreeFailures_Block tests that 3 failures trigger blocking
func TestWorkflow_ThreeFailures_Block(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Pre-populate with 2 failures (general_error from exit code 1)
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "general_error", Timestamp: now - 20, Tool: "Bash"},
		{File: "test.py", ErrorType: "general_error", Timestamp: now - 10, Tool: "Bash"},
	})

	// Simulate third failure
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{
			"exit_code": 1.0,
			"output":    "TypeError: foo",
		},
		CapturedAt: now,
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	require.NoError(t, memory.LogFailure(info))

	// Verify count
	count, err := memory.GetFailureCount("test.py", "general_error")
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 failures")

	// Should now be debugging loop
	isLoop, err := memory.IsDebuggingLoop("test.py", "general_error")
	require.NoError(t, err)
	assert.True(t, isLoop, "Should be debugging loop at 3 failures")
}

// TestWorkflow_MultipleFiles_Independent tests that different files are tracked separately
func TestWorkflow_MultipleFiles_Independent(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Log 2 failures for file A, 2 for file B (same error type)
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "fileA.py", ErrorType: "typeerror", Timestamp: now - 30, Tool: "Bash"},
		{File: "fileA.py", ErrorType: "typeerror", Timestamp: now - 20, Tool: "Bash"},
		{File: "fileB.py", ErrorType: "typeerror", Timestamp: now - 10, Tool: "Bash"},
		{File: "fileB.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
	})

	// Both files should have 2 failures (independent tracking)
	countA, err := memory.GetFailureCount("fileA.py", "typeerror")
	require.NoError(t, err)
	assert.Equal(t, 2, countA, "File A should have 2 failures")

	countB, err := memory.GetFailureCount("fileB.py", "typeerror")
	require.NoError(t, err)
	assert.Equal(t, 2, countB, "File B should have 2 failures")

	// Neither should be debugging loop
	isLoopA, _ := memory.IsDebuggingLoop("fileA.py", "typeerror")
	isLoopB, _ := memory.IsDebuggingLoop("fileB.py", "typeerror")
	assert.False(t, isLoopA, "File A should not be in loop")
	assert.False(t, isLoopB, "File B should not be in loop")
}

// TestWorkflow_MixedErrors_NoFalsePositive tests composite key correctness
func TestWorkflow_MixedErrors_NoFalsePositive(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Log 3 DIFFERENT errors on same file (should not trigger)
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "typeerror", Timestamp: now - 20, Tool: "Bash"},
		{File: "test.py", ErrorType: "valueerror", Timestamp: now - 10, Tool: "Bash"},
		{File: "test.py", ErrorType: "syntaxerror", Timestamp: now, Tool: "Bash"},
	})

	// Each error type should have count=1
	countType, _ := memory.GetFailureCount("test.py", "typeerror")
	countValue, _ := memory.GetFailureCount("test.py", "valueerror")
	countSyntax, _ := memory.GetFailureCount("test.py", "syntaxerror")

	assert.Equal(t, 1, countType, "TypeError count should be 1")
	assert.Equal(t, 1, countValue, "ValueError count should be 1")
	assert.Equal(t, 1, countSyntax, "SyntaxError count should be 1")

	// None should trigger debugging loop
	isLoopType, _ := memory.IsDebuggingLoop("test.py", "typeerror")
	isLoopValue, _ := memory.IsDebuggingLoop("test.py", "valueerror")
	isLoopSyntax, _ := memory.IsDebuggingLoop("test.py", "syntaxerror")

	assert.False(t, isLoopType, "TypeError should not trigger loop")
	assert.False(t, isLoopValue, "ValueError should not trigger loop")
	assert.False(t, isLoopSyntax, "SyntaxError should not trigger loop")
}

// TestWorkflow_SameError_Trigger tests that 3 identical errors trigger
func TestWorkflow_SameError_Trigger(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Log 3 SAME errors (should trigger)
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "typeerror", Timestamp: now - 20, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: now - 10, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
	})

	// Should have count=3
	count, err := memory.GetFailureCount("test.py", "typeerror")
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 failures of same type")

	// Should trigger debugging loop
	isLoop, err := memory.IsDebuggingLoop("test.py", "typeerror")
	require.NoError(t, err)
	assert.True(t, isLoop, "Should trigger debugging loop with 3 same errors")
}

// TestWorkflow_CaptureToPendingLearnings tests that sharp edges are captured
func TestWorkflow_CaptureToPendingLearnings(t *testing.T) {
	projectDir := setupTestProject(t)
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	// Create a sharp edge entry
	sharpEdge := map[string]interface{}{
		"file":                "test.py",
		"error_type":          "typeerror",
		"consecutive_failures": 3,
		"timestamp":           time.Now().Unix(),
	}

	// Write to pending learnings
	file, err := os.Create(pendingPath)
	require.NoError(t, err)
	defer file.Close()

	data, err := json.Marshal(sharpEdge)
	require.NoError(t, err)

	_, err = file.WriteString(string(data) + "\n")
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(pendingPath)
	require.NoError(t, err, "pending-learnings.jsonl should be created")

	// Verify content
	content, err := os.ReadFile(pendingPath)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(content[:len(content)-1], &parsed)) // Remove trailing newline

	assert.Equal(t, "test.py", parsed["file"])
	assert.Equal(t, "typeerror", parsed["error_type"])
	assert.Equal(t, float64(3), parsed["consecutive_failures"]) // JSON unmarshals numbers as float64
}

// TestWorkflow_TimeWindowFiltering tests that old failures are excluded
func TestWorkflow_TimeWindowFiltering(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)
	t.Setenv("GOGENT_MAX_FAILURES", "3")
	t.Setenv("GOGENT_FAILURE_WINDOW", "60") // 60 second window

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()
	old := now - 120 // 2 minutes ago (outside 60s window)

	// Log 2 old failures + 1 recent
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "typeerror", Timestamp: old, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: old - 10, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
	})

	// Should only count recent (1)
	count, err := memory.GetFailureCount("test.py", "typeerror")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should only count failures in time window")

	// Should not trigger debugging loop
	isLoop, err := memory.IsDebuggingLoop("test.py", "typeerror")
	require.NoError(t, err)
	assert.False(t, isLoop, "Should not trigger with old failures excluded")
}

// TestWorkflow_ClearAfterResolution tests that clearing works
func TestWorkflow_ClearAfterResolution(t *testing.T) {
	projectDir := setupTestProject(t)
	trackerPath := filepath.Join(projectDir, ".gogent", "failure-tracker.jsonl")

	t.Setenv("GOGENT_STORAGE_PATH", trackerPath)

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	now := time.Now().Unix()

	// Log 3 failures
	createTrackerEntries(t, []routing.FailureInfo{
		{File: "test.py", ErrorType: "typeerror", Timestamp: now - 20, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: now - 10, Tool: "Bash"},
		{File: "test.py", ErrorType: "typeerror", Timestamp: now, Tool: "Bash"},
	})

	// Verify count before clear
	count, _ := memory.GetFailureCount("test.py", "typeerror")
	assert.Equal(t, 3, count, "Should have 3 failures before clear")

	// Clear
	require.NoError(t, memory.ClearFailures("test.py", "typeerror"))

	// Verify count after clear
	count, err := memory.GetFailureCount("test.py", "typeerror")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should have 0 failures after clear")

	// Should not be debugging loop
	isLoop, err := memory.IsDebuggingLoop("test.py", "typeerror")
	require.NoError(t, err)
	assert.False(t, isLoop, "Should not be debugging loop after clear")
}
