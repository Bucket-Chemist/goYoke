package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountLogLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty file",
			content:  "",
			expected: 0,
		},
		{
			name:     "single line",
			content:  "error log entry\n",
			expected: 1,
		},
		{
			name:     "multiple lines",
			content:  "line1\nline2\nline3\n",
			expected: 3,
		},
		{
			name:     "mixed empty lines",
			content:  "line1\n\nline2\n  \nline3\n",
			expected: 3,
		},
		{
			name:     "only empty lines",
			content:  "\n\n  \n\t\n",
			expected: 0,
		},
		{
			name:     "no trailing newline",
			content:  "line1\nline2",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "logtest-*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			// Write content
			if _, err := tmpfile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpfile.Close()

			// Test countLogLines
			got, err := countLogLines(tmpfile.Name())
			if err != nil {
				t.Errorf("countLogLines() error = %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("countLogLines() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCountLogLines_MissingFile(t *testing.T) {
	// Non-existent file should return 0, not error
	got, err := countLogLines("/nonexistent/path/file.log")
	if err != nil {
		t.Errorf("countLogLines() on missing file should not error, got: %v", err)
	}
	if got != 0 {
		t.Errorf("countLogLines() on missing file = %d, want 0", got)
	}
}

func TestCountToolCalls(t *testing.T) {
	// Create temp directory for test counters
	tmpdir, err := os.MkdirTemp("", "toolcount-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	// Save original Glob behavior by testing with real /tmp
	// Create actual counter files in /tmp for realistic test
	counterFiles := []string{
		filepath.Join("/tmp", "claude-tool-counter-test1"),
		filepath.Join("/tmp", "claude-tool-counter-test2"),
		filepath.Join("/tmp", "claude-tool-counter-test3"),
	}

	// Write counter files with specific counts
	counts := []int{5, 3, 2}
	for i, path := range counterFiles {
		content := ""
		for j := 0; j < counts[i]; j++ {
			content += "tool call\n"
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write counter file: %v", err)
		}
		defer os.Remove(path)
	}

	// Test countToolCalls
	got, err := countToolCalls()
	if err != nil {
		t.Errorf("countToolCalls() error = %v", err)
		return
	}

	// Should count at least our test files (may include others from system)
	expectedMin := 10 // 5 + 3 + 2 from our test files
	if got < expectedMin {
		t.Errorf("countToolCalls() = %d, want at least %d", got, expectedMin)
	}
}

func TestCountToolCalls_NoCounters(t *testing.T) {
	// Clean up any existing counter files first
	matches, _ := filepath.Glob("/tmp/claude-tool-counter-*")
	for _, m := range matches {
		os.Remove(m)
	}

	// Test with no counter files
	got, err := countToolCalls()
	if err != nil {
		t.Errorf("countToolCalls() with no counters should not error, got: %v", err)
	}
	if got != 0 {
		t.Errorf("countToolCalls() with no counters = %d, want 0", got)
	}
}

func TestCollectSessionMetrics(t *testing.T) {
	// Create temporary log files
	tmpdir, err := os.MkdirTemp("", "metrics-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	// Create mock error log
	errorLog := filepath.Join(tmpdir, "errors.jsonl")
	errorContent := `{"error": "test1"}
{"error": "test2"}
{"error": "test3"}
`
	if err := os.WriteFile(errorLog, []byte(errorContent), 0644); err != nil {
		t.Fatalf("Failed to write error log: %v", err)
	}

	// Create mock violations log
	violationsLog := filepath.Join(tmpdir, "violations.jsonl")
	violationsContent := `{"violation": "test1"}
{"violation": "test2"}
`
	if err := os.WriteFile(violationsLog, []byte(violationsContent), 0644); err != nil {
		t.Fatalf("Failed to write violations log: %v", err)
	}

	// Note: We can't easily mock config.GetViolationsLogPath() without modifying production code
	// So we'll test the core logic through individual function tests
	// Full integration test would require dependency injection

	t.Run("returns metrics struct", func(t *testing.T) {
		metrics, err := CollectSessionMetrics("test-session-123")
		if err != nil {
			t.Errorf("CollectSessionMetrics() error = %v", err)
			return
		}
		if metrics == nil {
			t.Error("CollectSessionMetrics() returned nil metrics")
			return
		}
		if metrics.SessionID != "test-session-123" {
			t.Errorf("SessionID = %s, want test-session-123", metrics.SessionID)
		}
		// Tool calls, errors, violations will be 0 or system values
		// We verify the struct is populated without errors
	})
}

func TestCollectSessionMetrics_MissingLogs(t *testing.T) {
	// Test that missing log files don't cause errors
	metrics, err := CollectSessionMetrics("test-session-missing")
	if err != nil {
		t.Errorf("CollectSessionMetrics() with missing logs error = %v", err)
		return
	}
	if metrics == nil {
		t.Error("CollectSessionMetrics() returned nil metrics")
		return
	}
	if metrics.SessionID != "test-session-missing" {
		t.Errorf("SessionID = %s, want test-session-missing", metrics.SessionID)
	}
	// Counts should be 0 or system values, but no error
}

func TestGetErrorLogPath(t *testing.T) {
	path := getErrorLogPath()
	expected := "/tmp/claude-error-patterns.jsonl"
	if path != expected {
		t.Errorf("getErrorLogPath() = %s, want %s", path, expected)
	}
}
