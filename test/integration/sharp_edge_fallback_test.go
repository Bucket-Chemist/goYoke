package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/memory"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// itoa converts int64 to string
func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// TestFallback_MissingProjectDir tests behavior when GOGENT_PROJECT_DIR not set
func TestFallback_MissingProjectDir(t *testing.T) {
	// Unset GOGENT_PROJECT_DIR if it exists
	originalDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Unsetenv("GOGENT_PROJECT_DIR")
	defer func() {
		if originalDir != "" {
			os.Setenv("GOGENT_PROJECT_DIR", originalDir)
		}
	}()

	// Set explicit storage path for test
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should still work with explicit path
	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	err := memory.LogFailure(info)
	assert.NoError(t, err, "Should work without GOGENT_PROJECT_DIR when path is explicit")
}

// TestFallback_NonExistentDir tests behavior when project dir doesn't exist
func TestFallback_NonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does", "not", "exist", "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = nonExistent
	defer func() { memory.DefaultStoragePath = originalPath }()

	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	// Should create parent directories automatically
	err := memory.LogFailure(info)
	assert.NoError(t, err, "Should create directories automatically")

	// Verify file was created
	_, err = os.Stat(nonExistent)
	assert.NoError(t, err, "File should be created")
}

// TestFallback_ReadOnlyDir tests behavior when cannot write to directory
func TestFallback_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test requires non-root user (root can write to read-only dirs)")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0755))

	// Make directory read-only
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup

	trackerPath := filepath.Join(readOnlyDir, "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	// Should return error when cannot write
	err := memory.LogFailure(info)
	assert.Error(t, err, "Should error when directory is read-only")
}

// TestFallback_CorruptedTrackerLog tests handling of malformed JSONL
func TestFallback_CorruptedTrackerLog(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	// Use current timestamps to be within the failure window (default 300 seconds)
	now := time.Now().Unix()

	// Write corrupted JSONL (mix of valid and invalid)
	corruptedContent := `{"file":"test1.go","error_type":"error1","timestamp":` + itoa(now-10) + `}
{this is not valid json
{"file":"test2.go","error_type":"error2","timestamp":` + itoa(now-5) + `}
invalid line here
{"file":"test3.go","error_type":"error3","timestamp":` + itoa(now-2) + `}
`
	require.NoError(t, os.WriteFile(trackerPath, []byte(corruptedContent), 0644))

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should gracefully skip malformed lines
	count, err := memory.GetFailureCount("test2.go", "error2")
	require.NoError(t, err, "Should skip malformed lines gracefully")
	assert.Equal(t, 1, count, "Should count valid entries")

	// Verify all valid entries are readable
	countTest1, _ := memory.GetFailureCount("test1.go", "error1")
	countTest3, _ := memory.GetFailureCount("test3.go", "error3")

	assert.Equal(t, 1, countTest1, "Should read first valid entry")
	assert.Equal(t, 1, countTest3, "Should read last valid entry")
}

// TestFallback_MissingTrackerLog tests first run when log file doesn't exist
func TestFallback_MissingTrackerLog(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentLog := filepath.Join(tmpDir, "does-not-exist.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = nonExistentLog
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should return 0 count for missing file (not an error)
	count, err := memory.GetFailureCount("any.go", "any_error")
	require.NoError(t, err, "Missing file should not be an error")
	assert.Equal(t, 0, count, "Should return 0 for missing file")

	// Should not be debugging loop
	isLoop, err := memory.IsDebuggingLoop("any.go", "any_error")
	require.NoError(t, err)
	assert.False(t, isLoop, "Should not be loop when file missing")

	// Clear should be no-op for missing file
	err = memory.ClearFailures("any.go", "any_error")
	assert.NoError(t, err, "Clear should be no-op for missing file")
}

// TestFallback_EmptyTrackerLog tests handling of empty log file
func TestFallback_EmptyTrackerLog(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	// Create empty file
	require.NoError(t, os.WriteFile(trackerPath, []byte(""), 0644))

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should handle empty file gracefully
	count, err := memory.GetFailureCount("any.go", "any_error")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should return 0 for empty file")
}

// TestFallback_PartiallyWrittenEntry tests atomic write behavior
func TestFallback_PartiallyWrittenEntry(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	// Simulate partially written JSON (truncated)
	partialJSON := `{"file":"test.go","error_type":"error1","timest`

	require.NoError(t, os.WriteFile(trackerPath, []byte(partialJSON), 0644))

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should skip invalid line
	count, err := memory.GetFailureCount("test.go", "error1")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should skip truncated JSON")
}

// TestFallback_InvalidEnvironmentVariables tests fallback to defaults
func TestFallback_InvalidEnvironmentVariables(t *testing.T) {
	testCases := []struct {
		name     string
		envVar   string
		envValue string
	}{
		{"invalid_max_failures", "GOGENT_MAX_FAILURES", "invalid"},
		{"negative_max_failures", "GOGENT_MAX_FAILURES", "-1"},
		{"zero_max_failures", "GOGENT_MAX_FAILURES", "0"},
		{"invalid_window", "GOGENT_FAILURE_WINDOW", "not_a_number"},
		{"negative_window", "GOGENT_FAILURE_WINDOW", "-100"},
		{"zero_window", "GOGENT_FAILURE_WINDOW", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envVar, tc.envValue)

			tmpDir := t.TempDir()
			trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

			originalPath := memory.DefaultStoragePath
			memory.DefaultStoragePath = trackerPath
			defer func() { memory.DefaultStoragePath = originalPath }()

			// Should use defaults when env vars are invalid
			info := &routing.FailureInfo{
				File:      "test.go",
				ErrorType: "test_error",
				Timestamp: time.Now().Unix(),
			}

			// Should work with defaults
			err := memory.LogFailure(info)
			assert.NoError(t, err, "Should fall back to defaults")

			count, err := memory.GetFailureCount("test.go", "test_error")
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

// TestFallback_LongFilePath tests handling of extremely long file paths
func TestFallback_LongFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Create very long file path (approaching filesystem limits)
	longPath := "/very/long/path/" + string(make([]byte, 200))
	for i := range longPath[16:] {
		longPath = longPath[:16+i] + "a" + longPath[17+i:]
	}

	info := &routing.FailureInfo{
		File:      longPath,
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	// Should handle long paths
	err := memory.LogFailure(info)
	assert.NoError(t, err, "Should handle long file paths")

	// Should be retrievable
	count, err := memory.GetFailureCount(longPath, "test_error")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestFallback_FilePermissionChange tests recovery when permissions change
func TestFallback_FilePermissionChange(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test requires non-root user")
	}

	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = trackerPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Create file with initial data
	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}
	require.NoError(t, memory.LogFailure(info))

	// Make file unreadable
	require.NoError(t, os.Chmod(trackerPath, 0000))

	// Should error when file is unreadable
	_, err := memory.GetFailureCount("test.go", "test_error")
	assert.Error(t, err, "Should error when file is unreadable")

	// Restore permissions
	require.NoError(t, os.Chmod(trackerPath, 0644))

	// Should work again
	count, err := memory.GetFailureCount("test.go", "test_error")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should work after permissions restored")
}

// TestFallback_BrokenSymlink tests handling of broken symlinks
func TestFallback_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "target.jsonl")
	linkPath := filepath.Join(tmpDir, "link.jsonl")

	// Create symlink to non-existent target
	require.NoError(t, os.Symlink(targetPath, linkPath))

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = linkPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Should create target through symlink
	info := &routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	err := memory.LogFailure(info)
	assert.NoError(t, err, "Should create target file through symlink")

	// Verify target was created
	_, err = os.Stat(targetPath)
	assert.NoError(t, err, "Target file should exist")
}

// TestFallback_ConcurrentDirectoryCreation tests race in directory creation
func TestFallback_ConcurrentDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "failure-tracker.jsonl")

	originalPath := memory.DefaultStoragePath
	memory.DefaultStoragePath = deepPath
	defer func() { memory.DefaultStoragePath = originalPath }()

	// Multiple goroutines trying to log simultaneously
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			info := &routing.FailureInfo{
				File:      "test.go",
				ErrorType: "test_error",
				Timestamp: time.Now().Unix(),
			}
			err := memory.LogFailure(info)
			assert.NoError(t, err, "Goroutine %d should succeed", id)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all logged
	count, err := memory.GetFailureCount("test.go", "test_error")
	require.NoError(t, err)
	assert.Equal(t, 5, count, "Should have 5 entries")
}

// TestFallback_JSONMarshalError tests handling of unmarshalable data
func TestFallback_JSONMarshalError(t *testing.T) {
	// routing.FailureInfo should always be marshalable, but we can test the pattern
	tmpDir := t.TempDir()
	trackerPath := filepath.Join(tmpDir, "failure-tracker.jsonl")

	// Write valid entry
	info := routing.FailureInfo{
		File:      "test.go",
		ErrorType: "test_error",
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(info)
	require.NoError(t, err, "FailureInfo should always be marshalable")

	require.NoError(t, os.WriteFile(trackerPath, data, 0644))

	// Verify it's readable
	content, err := os.ReadFile(trackerPath)
	require.NoError(t, err)

	var parsed routing.FailureInfo
	require.NoError(t, json.Unmarshal(content, &parsed))
	assert.Equal(t, info.File, parsed.File)
}
