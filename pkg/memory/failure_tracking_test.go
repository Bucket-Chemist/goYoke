package memory

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// setupTestFile creates a temporary JSONL file for testing
func setupTestFile(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	// Override DefaultStoragePath for tests
	originalPath := DefaultStoragePath
	cleanup := func() {
		DefaultStoragePath = originalPath
	}

	// Use absolute path directly (no ~ expansion in tests)
	DefaultStoragePath = testPath

	return testPath, cleanup
}

// TestLogFailure_Success tests basic logging functionality
func TestLogFailure_Success(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "syntax_error",
		Timestamp: time.Now().Unix(),
		Tool:      "Edit",
	}

	if err := LogFailure(info); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatalf("File was not created: %s", testPath)
	}

	// Verify content
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var logged routing.FailureInfo
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to unmarshal logged data: %v", err)
	}

	if logged.File != info.File || logged.ErrorType != info.ErrorType {
		t.Errorf("Logged data mismatch: got %+v, want %+v", logged, info)
	}
}

// TestLogFailure_NilInfo tests error handling for nil input
func TestLogFailure_NilInfo(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	err := LogFailure(nil)
	if err == nil {
		t.Fatal("Expected error for nil info, got nil")
	}
	if err.Error() != "failure info cannot be nil" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestCompositeKey_PreventsFalsePositives tests that different error types
// on the same file are counted separately (composite key: file + error_type)
func TestCompositeKey_PreventsFalsePositives(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log 3 different error types on same file
	errors := []string{"syntax_error", "type_error", "import_error"}
	for _, errType := range errors {
		if err := LogFailure(&routing.FailureInfo{
			File:      "main.py",
			ErrorType: errType,
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Each error type should have count=1 (not 3)
	for _, errType := range errors {
		count, err := GetFailureCount("main.py", errType)
		if err != nil {
			t.Fatalf("GetFailureCount failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count=1 for %s, got %d", errType, count)
		}
	}
}

// TestDebuggingLoop_ThresholdDetection tests that 3+ identical failures trigger threshold
func TestDebuggingLoop_ThresholdDetection(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log same error 3 times
	for i := 0; i < 3; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "buggy.go",
			ErrorType: "nil_pointer",
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Should detect debugging loop
	isLoop, err := IsDebuggingLoop("buggy.go", "nil_pointer")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if !isLoop {
		t.Error("Expected debugging loop to be detected")
	}

	// Different file should not trigger
	isLoop, err = IsDebuggingLoop("other.go", "nil_pointer")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop for different file")
	}

	// Different error type should not trigger
	isLoop, err = IsDebuggingLoop("buggy.go", "syntax_error")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop for different error type")
	}
}

// TestTimeWindow_ExcludesOldEntries tests that failures outside the time window are ignored
func TestTimeWindow_ExcludesOldEntries(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	// Set a small time window for testing
	os.Setenv("GOYOKE_FAILURE_WINDOW", "60")
	defer os.Unsetenv("GOYOKE_FAILURE_WINDOW")

	now := time.Now().Unix()
	old := now - 120 // 2 minutes ago (outside 60s window)

	// Log 2 old failures
	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "timeout",
			Timestamp: old,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Log 1 recent failure
	if err := LogFailure(&routing.FailureInfo{
		File:      "test.go",
		ErrorType: "timeout",
		Timestamp: now,
	}); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Should only count recent failure
	count, err := GetFailureCount("test.go", "timeout")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 (old entries excluded), got %d", count)
	}
}

// TestGetFailureCount_MissingFile tests graceful handling when tracker file doesn't exist
func TestGetFailureCount_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist.jsonl")

	originalPath := DefaultStoragePath
	DefaultStoragePath = nonExistentPath
	defer func() { DefaultStoragePath = originalPath }()

	// Should return 0, not error
	count, err := GetFailureCount("any.go", "any_error")
	if err != nil {
		t.Fatalf("Unexpected error for missing file: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for missing file, got %d", count)
	}
}

// TestGetFailureCount_EmptyFile tests handling of empty JSONL file
func TestGetFailureCount_EmptyFile(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	// Create empty file
	if err := os.WriteFile(testPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	count, err := GetFailureCount("any.go", "any_error")
	if err != nil {
		t.Fatalf("Unexpected error for empty file: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for empty file, got %d", count)
	}
}

// TestGetFailureCount_MalformedJSON tests graceful handling of corrupted entries
func TestGetFailureCount_MalformedJSON(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Write mix of valid and invalid entries
	content := `{"file":"test.go","error_type":"valid1","timestamp":` + itoa(now) + `}
{bad json here}
{"file":"test.go","error_type":"valid2","timestamp":` + itoa(now) + `}
`
	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should count valid entries only
	count, err := GetFailureCount("test.go", "valid1")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 (malformed line skipped), got %d", count)
	}

	count, err = GetFailureCount("test.go", "valid2")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 (malformed line skipped), got %d", count)
	}
}

func TestGetFailureCount_LargeLine(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()
	largeMatch := strings.Repeat("m", 70*1024)
	content := `{"file":"test.go","error_type":"large_error","timestamp":` + itoa(now) + `,"error_match":"` + largeMatch + `"}`
	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	count, err := GetFailureCount("test.go", "large_error")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected count=1, got %d", count)
	}
}

// TestClearFailures_RemovesMatchingEntries tests that ClearFailures removes only matching entries
func TestClearFailures_RemovesMatchingEntries(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log multiple failures
	failures := []struct {
		file      string
		errorType string
	}{
		{"main.go", "syntax_error"},
		{"main.go", "syntax_error"},  // duplicate
		{"main.go", "type_error"},    // different error
		{"other.go", "syntax_error"}, // different file
	}

	for _, f := range failures {
		if err := LogFailure(&routing.FailureInfo{
			File:      f.file,
			ErrorType: f.errorType,
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Clear main.go/syntax_error
	if err := ClearFailures("main.go", "syntax_error"); err != nil {
		t.Fatalf("ClearFailures failed: %v", err)
	}

	// Should be cleared
	count, err := GetFailureCount("main.go", "syntax_error")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 after clear, got %d", count)
	}

	// Other entries should remain
	count, err = GetFailureCount("main.go", "type_error")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 for type_error, got %d", count)
	}

	count, err = GetFailureCount("other.go", "syntax_error")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 for other.go, got %d", count)
	}
}

// TestClearFailures_MissingFile tests that ClearFailures handles missing file gracefully
func TestClearFailures_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist.jsonl")

	originalPath := DefaultStoragePath
	DefaultStoragePath = nonExistentPath
	defer func() { DefaultStoragePath = originalPath }()

	// Should not error on missing file
	if err := ClearFailures("any.go", "any_error"); err != nil {
		t.Fatalf("Unexpected error for missing file: %v", err)
	}
}

// TestEnvironmentVariables_Configuration tests env var overrides
func TestEnvironmentVariables_Configuration(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected int
		getter   func() int
	}{
		{
			name:     "GOYOKE_MAX_FAILURES",
			envVar:   "GOYOKE_MAX_FAILURES",
			envValue: "5",
			expected: 5,
			getter:   getMaxFailures,
		},
		{
			name:     "GOYOKE_FAILURE_WINDOW",
			envVar:   "GOYOKE_FAILURE_WINDOW",
			envValue: "600",
			expected: 600,
			getter:   getFailureWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			got := tt.getter()
			if got != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, got)
			}
		})
	}
}

// TestEnvironmentVariables_InvalidValues tests fallback to defaults
func TestEnvironmentVariables_InvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected int
		getter   func() int
	}{
		{"invalid_max_failures", "GOYOKE_MAX_FAILURES", "invalid", DefaultMaxFailures, getMaxFailures},
		{"invalid_window", "GOYOKE_FAILURE_WINDOW", "invalid", DefaultFailureWindow, getFailureWindow},
		{"negative_max", "GOYOKE_MAX_FAILURES", "-1", DefaultMaxFailures, getMaxFailures},
		{"zero_window", "GOYOKE_FAILURE_WINDOW", "0", DefaultFailureWindow, getFailureWindow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			got := tt.getter()
			if got != tt.expected {
				t.Errorf("Expected fallback to %d, got %d", tt.expected, got)
			}
		})
	}
}

// TestConcurrentAccess_MultipleWrites tests basic concurrent write safety
// Note: Real concurrent access would require file locking (out of scope for this ticket)
func TestConcurrentAccess_MultipleWrites(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Sequential writes simulating concurrent access pattern
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(id int) {
			err := LogFailure(&routing.FailureInfo{
				File:      "concurrent.go",
				ErrorType: "race_condition",
				Timestamp: now,
				Tool:      "Bash",
			})
			if err != nil {
				t.Errorf("Concurrent write %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all writes succeeded (basic check, not true concurrent safety)
	count, err := GetFailureCount("concurrent.go", "race_condition")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count=3 after concurrent writes, got %d", count)
	}
}

// TestExpandPath tests home directory expansion
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantHome bool
	}{
		{"tilde_only", "~", false, true},
		{"tilde_path", "~/.goyoke/file.jsonl", false, true},
		{"absolute", "/tmp/file.jsonl", false, false},
		{"relative", "file.jsonl", false, false},
		{"empty", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantHome {
				home, _ := os.UserHomeDir()
				if tt.input == "~" && got != home {
					t.Errorf("Expected %s, got %s", home, got)
				}
				if tt.input != "~" && !filepath.IsAbs(got) {
					t.Errorf("Expected absolute path, got %s", got)
				}
			}
		})
	}
}

// TestIsDebuggingLoop_EdgeCases tests edge cases for debugging loop detection
func TestIsDebuggingLoop_EdgeCases(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Test exactly at threshold
	os.Setenv("GOYOKE_MAX_FAILURES", "2")
	defer os.Unsetenv("GOYOKE_MAX_FAILURES")

	// Log 2 failures (exactly at threshold)
	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "edge.go",
			ErrorType: "boundary_test",
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Should trigger at threshold
	isLoop, err := IsDebuggingLoop("edge.go", "boundary_test")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if !isLoop {
		t.Error("Expected debugging loop at threshold")
	}

	// Clear and verify
	if err := ClearFailures("edge.go", "boundary_test"); err != nil {
		t.Fatalf("ClearFailures failed: %v", err)
	}

	isLoop, err = IsDebuggingLoop("edge.go", "boundary_test")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Expected no debugging loop after clear")
	}
}

// TestLogFailure_DirectoryCreationError tests error when parent directory cannot be created
func TestLogFailure_DirectoryCreationError(t *testing.T) {
	// Create a file where the directory should be (prevents mkdir)
	tmpFile := filepath.Join(t.TempDir(), "blockdir")
	if err := os.WriteFile(tmpFile, []byte("block"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	originalPath := DefaultStoragePath
	DefaultStoragePath = filepath.Join(tmpFile, "subdir", "file.jsonl")
	defer func() { DefaultStoragePath = originalPath }()

	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test",
		Timestamp: time.Now().Unix(),
	}

	err := LogFailure(info)
	if err == nil {
		t.Fatal("Expected error when directory creation fails")
	}
}

// TestLogFailure_FileOpenError tests error handling when file cannot be opened
func TestLogFailure_FileOpenError(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "readonly", "file.jsonl")

	// Create directory as read-only to prevent file creation
	if err := os.MkdirAll(filepath.Dir(testPath), 0555); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer os.Chmod(filepath.Dir(testPath), 0755) // restore for cleanup

	originalPath := DefaultStoragePath
	DefaultStoragePath = testPath
	defer func() { DefaultStoragePath = originalPath }()

	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test",
		Timestamp: time.Now().Unix(),
	}

	err := LogFailure(info)
	if err == nil {
		t.Error("Expected error when file cannot be opened")
	}
}

// TestExpandPath_ErrorCases tests error handling in path expansion
func TestExpandPath_HomeError(t *testing.T) {
	// This test verifies that expandPath handles home directory expansion correctly
	// The actual error condition (UserHomeDir() failing) is difficult to trigger in tests
	// but we can verify the normal path works
	result, err := expandPath("~/test/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "~/test/path" {
		t.Error("Expected path expansion to occur")
	}
}

// TestRewriteFailures_AtomicBehavior tests the atomic rewrite functionality
func TestRewriteFailures_AtomicBehavior(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Create initial entries
	initial := []routing.FailureInfo{
		{File: "a.go", ErrorType: "err1", Timestamp: now},
		{File: "b.go", ErrorType: "err2", Timestamp: now},
	}

	for _, entry := range initial {
		if err := LogFailure(&entry); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Rewrite with filtered set
	filtered := []routing.FailureInfo{
		{File: "b.go", ErrorType: "err2", Timestamp: now},
	}

	if err := rewriteFailures(testPath, filtered); err != nil {
		t.Fatalf("rewriteFailures failed: %v", err)
	}

	// Verify only filtered entry remains
	count, err := GetFailureCount("a.go", "err1")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for removed entry, got %d", count)
	}

	count, err = GetFailureCount("b.go", "err2")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count=1 for kept entry, got %d", count)
	}
}

// TestRewriteFailures_EmptyList tests rewriting with empty list
func TestRewriteFailures_EmptyList(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	// Create some entries
	if err := LogFailure(&routing.FailureInfo{
		File:      "test.go",
		ErrorType: "err",
		Timestamp: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Rewrite with empty list
	if err := rewriteFailures(testPath, []routing.FailureInfo{}); err != nil {
		t.Fatalf("rewriteFailures failed: %v", err)
	}

	// File should exist but be empty
	count, err := GetFailureCount("test.go", "err")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 after clearing, got %d", count)
	}
}

// TestScanFailures_EarlyStop tests visitor pattern with early stop
func TestScanFailures_EarlyStop(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log 5 entries
	for i := 0; i < 5; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "err",
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Count visits, stop after 3
	visits := 0
	err := scanFailures(testPath, func(info *routing.FailureInfo) bool {
		visits++
		return visits < 3 // stop after 3
	})

	if err != nil {
		t.Fatalf("scanFailures failed: %v", err)
	}

	if visits != 3 {
		t.Errorf("Expected 3 visits (early stop), got %d", visits)
	}
}

// TestIsDebuggingLoop_BelowThreshold tests behavior below threshold
func TestIsDebuggingLoop_BelowThreshold(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log 2 failures (below default threshold of 3)
	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "err",
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	isLoop, err := IsDebuggingLoop("test.go", "err")
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Expected no debugging loop below threshold")
	}
}

// TestIsDebuggingLoop_PropagatesError tests error propagation
func TestIsDebuggingLoop_PropagatesError(t *testing.T) {
	// Point to a directory (not a file) to trigger read error
	tmpDir := t.TempDir()
	originalPath := DefaultStoragePath
	DefaultStoragePath = tmpDir // directory, not file
	defer func() { DefaultStoragePath = originalPath }()

	// Create the path as a directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	_, err := IsDebuggingLoop("test.go", "err")
	if err == nil {
		t.Error("Expected error when reading directory as file")
	}
}

// TestGetFailureCount_ReadError tests error handling when file cannot be read
func TestGetFailureCount_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "unreadable.jsonl")

	// Create file
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Make it unreadable
	if err := os.Chmod(testPath, 0000); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer os.Chmod(testPath, 0644) // restore for cleanup

	originalPath := DefaultStoragePath
	DefaultStoragePath = testPath
	defer func() { DefaultStoragePath = originalPath }()

	_, err := GetFailureCount("test.go", "err")
	if err == nil {
		t.Error("Expected error when file cannot be read")
	}
}

// TestClearFailures_ReadError tests error handling during clear operation
func TestClearFailures_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "unreadable.jsonl")

	// Create file
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Make it unreadable
	if err := os.Chmod(testPath, 0000); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer os.Chmod(testPath, 0644) // restore for cleanup

	originalPath := DefaultStoragePath
	DefaultStoragePath = testPath
	defer func() { DefaultStoragePath = originalPath }()

	err := ClearFailures("test.go", "err")
	if err == nil {
		t.Error("Expected error when file cannot be read")
	}
}

// TestLogFailure_MultipleEntries tests appending multiple distinct entries
func TestLogFailure_MultipleEntries(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	entries := []routing.FailureInfo{
		{File: "a.go", ErrorType: "err1", Timestamp: now, Tool: "Edit"},
		{File: "b.go", ErrorType: "err2", Timestamp: now, Tool: "Bash"},
		{File: "c.go", ErrorType: "err3", Timestamp: now, Tool: "Write"},
	}

	for _, entry := range entries {
		if err := LogFailure(&entry); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Verify file contains all entries
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if scanner.Text() != "" {
			lines++
		}
	}

	if lines != 3 {
		t.Errorf("Expected 3 lines in file, got %d", lines)
	}

	// Verify each entry can be retrieved
	for _, entry := range entries {
		count, err := GetFailureCount(entry.File, entry.ErrorType)
		if err != nil {
			t.Fatalf("GetFailureCount failed for %s/%s: %v", entry.File, entry.ErrorType, err)
		}
		if count != 1 {
			t.Errorf("Expected count=1 for %s/%s, got %d", entry.File, entry.ErrorType, count)
		}
	}
}

// TestLogFailure_WithAllFields tests logging with all FailureInfo fields populated
func TestLogFailure_WithAllFields(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	info := &routing.FailureInfo{
		File:       "complex.go",
		ErrorType:  "complex_error",
		Timestamp:  time.Now().Unix(),
		Tool:       "Edit",
		ExitCode:   127,
		ErrorMatch: "command not found",
	}

	if err := LogFailure(info); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Verify by reading back
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var logged routing.FailureInfo
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if logged.ExitCode != info.ExitCode {
		t.Errorf("ExitCode mismatch: got %d, want %d", logged.ExitCode, info.ExitCode)
	}
	if logged.ErrorMatch != info.ErrorMatch {
		t.Errorf("ErrorMatch mismatch: got %s, want %s", logged.ErrorMatch, info.ErrorMatch)
	}
}

// TestGetFailureCount_MultipleTimeWindows tests filtering across different time windows
func TestGetFailureCount_MultipleTimeWindows(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	os.Setenv("GOYOKE_FAILURE_WINDOW", "120")
	defer os.Unsetenv("GOYOKE_FAILURE_WINDOW")

	now := time.Now().Unix()
	old := now - 180   // 3 minutes ago (outside 120s window)
	recent := now - 60 // 1 minute ago (inside 120s window)

	// Log failures at different times
	for i := 0; i < 3; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "err",
			Timestamp: old,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "err",
			Timestamp: recent,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Should only count recent ones
	count, err := GetFailureCount("test.go", "err")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count=2 (recent only), got %d", count)
	}
}

// TestClearFailures_MultipleCompositeKeys tests clearing with various key combinations
func TestClearFailures_MultipleCompositeKeys(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Create a matrix of failures
	files := []string{"a.go", "b.go"}
	errors := []string{"err1", "err2"}

	for _, file := range files {
		for _, errType := range errors {
			if err := LogFailure(&routing.FailureInfo{
				File:      file,
				ErrorType: errType,
				Timestamp: now,
			}); err != nil {
				t.Fatalf("LogFailure failed: %v", err)
			}
		}
	}

	// Clear one specific combination
	if err := ClearFailures("a.go", "err1"); err != nil {
		t.Fatalf("ClearFailures failed: %v", err)
	}

	// Verify cleared
	count, err := GetFailureCount("a.go", "err1")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for cleared entry, got %d", count)
	}

	// Verify others remain
	remaining := []struct{ file, errType string }{
		{"a.go", "err2"},
		{"b.go", "err1"},
		{"b.go", "err2"},
	}

	for _, r := range remaining {
		count, err := GetFailureCount(r.file, r.errType)
		if err != nil {
			t.Fatalf("GetFailureCount failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count=1 for %s/%s, got %d", r.file, r.errType, count)
		}
	}
}

// TestScanFailures_AllEntries tests scanning all entries without early stop
func TestScanFailures_AllEntries(t *testing.T) {
	testPath, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	// Log 10 entries
	for i := 0; i < 10; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      "test.go",
			ErrorType: "err",
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Count all visits
	visits := 0
	err := scanFailures(testPath, func(info *routing.FailureInfo) bool {
		visits++
		return true // continue to end
	})

	if err != nil {
		t.Fatalf("scanFailures failed: %v", err)
	}

	if visits != 10 {
		t.Errorf("Expected 10 visits, got %d", visits)
	}
}

// TestExpandPath_VariousPaths tests path expansion with different input formats
func TestExpandPath_VariousPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result string, err error)
	}{
		{
			name:  "home_relative",
			input: "~/Documents/test.json",
			check: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !filepath.IsAbs(result) {
					t.Errorf("Expected absolute path, got %s", result)
				}
				if !strings.Contains(result, "Documents") {
					t.Errorf("Expected path to contain Documents, got %s", result)
				}
			},
		},
		{
			name:  "absolute_unchanged",
			input: "/var/log/test.jsonl",
			check: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != "/var/log/test.jsonl" {
					t.Errorf("Expected unchanged path, got %s", result)
				}
			},
		},
		{
			name:  "relative_unchanged",
			input: "local/file.jsonl",
			check: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != "local/file.jsonl" {
					t.Errorf("Expected unchanged path, got %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			tt.check(t, result, err)
		})
	}
}

// TestIntegration_FullWorkflow tests complete debugging loop detection workflow
func TestIntegration_FullWorkflow(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	os.Setenv("GOYOKE_MAX_FAILURES", "3")
	os.Setenv("GOYOKE_FAILURE_WINDOW", "300")
	defer func() {
		os.Unsetenv("GOYOKE_MAX_FAILURES")
		os.Unsetenv("GOYOKE_FAILURE_WINDOW")
	}()

	file := "integration.go"
	errType := "integration_error"
	now := time.Now().Unix()

	// Phase 1: Log failures below threshold
	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      file,
			ErrorType: errType,
			Timestamp: now,
			Tool:      "Edit",
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Check - should not be debugging loop
	isLoop, err := IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop at 2 failures")
	}

	// Phase 2: Log one more to hit threshold
	if err := LogFailure(&routing.FailureInfo{
		File:      file,
		ErrorType: errType,
		Timestamp: now,
		Tool:      "Edit",
	}); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Check - should now be debugging loop
	isLoop, err = IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if !isLoop {
		t.Error("Expected debugging loop at 3 failures")
	}

	// Phase 3: Clear failures after resolution
	if err := ClearFailures(file, errType); err != nil {
		t.Fatalf("ClearFailures failed: %v", err)
	}

	// Check - should no longer be debugging loop
	isLoop, err = IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop after clear")
	}

	// Verify count is zero
	count, err := GetFailureCount(file, errType)
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 after clear, got %d", count)
	}
}

// TestIntegration_MultipleFilesConcurrent tests handling multiple files simultaneously
func TestIntegration_MultipleFilesConcurrent(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	now := time.Now().Unix()

	files := []string{"file1.go", "file2.go", "file3.go"}
	errors := []string{"syntax", "type", "runtime"}

	// Log failures for all combinations
	for _, file := range files {
		for _, errType := range errors {
			for i := 0; i < 2; i++ { // 2 failures each
				if err := LogFailure(&routing.FailureInfo{
					File:      file,
					ErrorType: errType,
					Timestamp: now,
				}); err != nil {
					t.Fatalf("LogFailure failed: %v", err)
				}
			}
		}
	}

	// Verify each combination has exactly 2 failures
	for _, file := range files {
		for _, errType := range errors {
			count, err := GetFailureCount(file, errType)
			if err != nil {
				t.Fatalf("GetFailureCount failed for %s/%s: %v", file, errType, err)
			}
			if count != 2 {
				t.Errorf("Expected count=2 for %s/%s, got %d", file, errType, count)
			}
		}
	}

	// Clear one specific combination
	if err := ClearFailures("file1.go", "syntax"); err != nil {
		t.Fatalf("ClearFailures failed: %v", err)
	}

	// Verify only that combination is cleared
	count, err := GetFailureCount("file1.go", "syntax")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count=0 for cleared entry, got %d", count)
	}

	// Verify all others remain
	count, err = GetFailureCount("file1.go", "type")
	if err != nil {
		t.Fatalf("GetFailureCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count=2 for non-cleared entry, got %d", count)
	}
}

// TestIntegration_TimeWindowBoundary tests behavior at time window boundaries
func TestIntegration_TimeWindowBoundary(t *testing.T) {
	_, cleanup := setupTestFile(t)
	defer cleanup()

	os.Setenv("GOYOKE_FAILURE_WINDOW", "10")
	defer os.Unsetenv("GOYOKE_FAILURE_WINDOW")

	now := time.Now().Unix()
	old := now - 11 // Outside window

	file := "boundary.go"
	errType := "boundary_error"

	// Log 3 old failures (outside window)
	for i := 0; i < 3; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      file,
			ErrorType: errType,
			Timestamp: old,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Should not trigger debugging loop (all outside window)
	isLoop, err := IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop for failures outside window")
	}

	// Log 2 recent failures (inside window)
	for i := 0; i < 2; i++ {
		if err := LogFailure(&routing.FailureInfo{
			File:      file,
			ErrorType: errType,
			Timestamp: now,
		}); err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}
	}

	// Should still not trigger (only 2 in window)
	isLoop, err = IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if isLoop {
		t.Error("Unexpected debugging loop at 2 recent failures")
	}

	// Log one more recent failure
	if err := LogFailure(&routing.FailureInfo{
		File:      file,
		ErrorType: errType,
		Timestamp: now,
	}); err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Now should trigger (3 in window)
	isLoop, err = IsDebuggingLoop(file, errType)
	if err != nil {
		t.Fatalf("IsDebuggingLoop failed: %v", err)
	}
	if !isLoop {
		t.Error("Expected debugging loop at 3 recent failures")
	}
}

// Helper function to convert int64 to string
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

// TestExtractFunctionFromStackTrace_PythonTraceback tests Python function extraction
func TestExtractFunctionFromStackTrace_PythonTraceback(t *testing.T) {
	errorOutput := `Traceback (most recent call last):
  File "test.py", line 10, in calculateTotal
    result = x / y
ZeroDivisionError: division by zero`

	funcName := ExtractFunctionFromStackTrace(errorOutput)
	if funcName != "calculateTotal" {
		t.Errorf("Expected 'calculateTotal', got '%s'", funcName)
	}
}

// TestExtractFunctionFromStackTrace_GoPanic tests Go panic function extraction
func TestExtractFunctionFromStackTrace_GoPanic(t *testing.T) {
	errorOutput := `panic: runtime error: invalid memory address
goroutine 1 [running]:
main.processData(0xc000...)`

	funcName := ExtractFunctionFromStackTrace(errorOutput)
	if funcName != "processData" {
		t.Errorf("Expected 'processData', got '%s'", funcName)
	}
}

// TestExtractFunctionFromStackTrace_GenericError tests graceful degradation
func TestExtractFunctionFromStackTrace_GenericError(t *testing.T) {
	errorOutput := "error: operation failed"

	funcName := ExtractFunctionFromStackTrace(errorOutput)
	if funcName != "" {
		t.Errorf("Expected empty string for generic error, got '%s'", funcName)
	}
}

// TestExtractFunctionFromStackTrace_EmptyInput tests empty error output
func TestExtractFunctionFromStackTrace_EmptyInput(t *testing.T) {
	funcName := ExtractFunctionFromStackTrace("")
	if funcName != "" {
		t.Errorf("Expected empty string for empty input, got '%s'", funcName)
	}
}

// TestExtractFunctionFromStackTrace_MultipleFunctions tests first match behavior
func TestExtractFunctionFromStackTrace_MultipleFunctions(t *testing.T) {
	// Python traceback with nested function calls
	errorOutput := `Traceback (most recent call last):
  File "test.py", line 15, in outerFunction
    result = innerFunction()
  File "test.py", line 10, in innerFunction
    raise ValueError("test")
ValueError: test`

	funcName := ExtractFunctionFromStackTrace(errorOutput)
	// Should match first occurrence
	if funcName != "outerFunction" {
		t.Errorf("Expected 'outerFunction', got '%s'", funcName)
	}
}

// TestExtractFunctionFromStackTrace_GoNestedCalls tests Go stack with multiple functions
func TestExtractFunctionFromStackTrace_GoNestedCalls(t *testing.T) {
	errorOutput := `panic: test panic
goroutine 1 [running]:
main.helper(0x1)
    /path/to/main.go:10 +0x20
main.process(0x2)
    /path/to/main.go:15 +0x30`

	funcName := ExtractFunctionFromStackTrace(errorOutput)
	// Should match first function call
	if funcName != "helper" {
		t.Errorf("Expected 'helper', got '%s'", funcName)
	}
}
