package archive

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Sharp Edges Tests
// ============================================================================

func TestListSharpEdges_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	// Create empty file
	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(""), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No sharp edges recorded") {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestListSharpEdges_WithSeverityFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"file":"a.go","error_type":"err1","consecutive_failures":3,"timestamp":1705000000,"severity":"high"}
{"file":"b.go","error_type":"err2","consecutive_failures":3,"timestamp":1705000001,"severity":"low"}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges", "--severity", "high"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "a.go") {
		t.Error("Expected a.go (high severity)")
	}
	if strings.Contains(output, "b.go") {
		t.Error("Expected b.go filtered out (low severity)")
	}
	if !strings.Contains(output, "Total: 1") {
		t.Error("Expected total of 1")
	}
}

func TestListSharpEdges_WithUnresolvedFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	// Use distinct filenames that don't contain each other as substrings
	content := `{"file":"fixed_edge.go","error_type":"type_error","consecutive_failures":3,"timestamp":1705000000,"resolved_at":1705001000}
{"file":"open_edge.go","error_type":"nil_pointer","consecutive_failures":5,"timestamp":1705000001}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges", "--unresolved"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if strings.Contains(output, "fixed_edge.go") {
		t.Error("Expected fixed_edge.go (resolved) filtered out")
	}
	if !strings.Contains(output, "open_edge.go") {
		t.Error("Expected open_edge.go (unresolved) in output")
	}
	if !strings.Contains(output, "Open") {
		t.Error("Expected 'Open' status for unresolved edge")
	}
	if !strings.Contains(output, "Total: 1") {
		t.Error("Expected total of 1 unresolved edge")
	}
}

func TestListSharpEdges_WithFileFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"file":"pkg/auth/handler.go","error_type":"err1","consecutive_failures":3,"timestamp":1705000000}
{"file":"cmd/main.go","error_type":"err2","consecutive_failures":2,"timestamp":1705000001}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges", "--file", "pkg/*"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "pkg/auth/handler.go") {
		t.Error("Expected pkg/auth/handler.go in output")
	}
	if strings.Contains(output, "cmd/main.go") {
		t.Error("Expected cmd/main.go filtered out")
	}
}

func TestListSharpEdges_WithErrorTypeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"file":"a.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000000}
{"file":"b.go","error_type":"type_mismatch","consecutive_failures":2,"timestamp":1705000001}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges", "--error-type", "nil_pointer"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "nil_pointer") {
		t.Error("Expected nil_pointer in output")
	}
	if strings.Contains(output, "type_mismatch") {
		t.Error("Expected type_mismatch filtered out")
	}
}

func TestListSharpEdges_WithSinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	// One recent, one old
	now := time.Now()
	recentTimestamp := now.AddDate(0, 0, -2).Unix()
	oldTimestamp := now.AddDate(0, 0, -30).Unix()

	content := `{"file":"recent.go","error_type":"err1","consecutive_failures":3,"timestamp":` + itoa(recentTimestamp) + `}
{"file":"old.go","error_type":"err2","consecutive_failures":2,"timestamp":` + itoa(oldTimestamp) + `}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges", "--since", "7d"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "recent.go") {
		t.Error("Expected recent.go in output")
	}
	if strings.Contains(output, "old.go") {
		t.Error("Expected old.go filtered out (>7 days)")
	}
}

func TestListSharpEdges_TableFormat(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"file":"test.go","error_type":"compile_error","consecutive_failures":5,"timestamp":1705000000,"severity":"high","resolved_at":1705001000}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "File") {
		t.Error("Expected 'File' header")
	}
	if !strings.Contains(output, "Error Type") {
		t.Error("Expected 'Error Type' header")
	}
	if !strings.Contains(output, "Failures") {
		t.Error("Expected 'Failures' header")
	}
	if !strings.Contains(output, "Severity") {
		t.Error("Expected 'Severity' header")
	}
	if !strings.Contains(output, "Status") {
		t.Error("Expected 'Status' header")
	}

	// Verify data
	if !strings.Contains(output, "test.go") {
		t.Error("Expected test.go in output")
	}
	if !strings.Contains(output, "compile_error") {
		t.Error("Expected compile_error in output")
	}
	if !strings.Contains(output, "5") {
		t.Error("Expected failure count 5")
	}
	if !strings.Contains(output, "high") {
		t.Error("Expected severity high")
	}
	if !strings.Contains(output, "Resolved") {
		t.Error("Expected 'Resolved' status")
	}
}

// ============================================================================
// User Intents Tests
// ============================================================================

func TestListUserIntents_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(""), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No user intents recorded") {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestListUserIntents_WithSourceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"timestamp":1705000000,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":1705000001,"question":"Q2","response":"A2","confidence":"explicit","source":"hook_prompt"}`

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents", "--source", "ask_user"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Q1") {
		t.Error("Expected Q1 (ask_user source)")
	}
	if strings.Contains(output, "Q2") {
		t.Error("Expected Q2 filtered out (hook_prompt source)")
	}
	if !strings.Contains(output, "Total: 1") {
		t.Error("Expected total of 1")
	}
}

func TestListUserIntents_HasActionFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"timestamp":1705000000,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user","action_taken":"Did X"}
{"timestamp":1705000001,"question":"Q2","response":"A2","confidence":"explicit","source":"ask_user"}`

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents", "--has-action"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Q1") {
		t.Error("Expected Q1 (has action)")
	}
	if strings.Contains(output, "Q2") {
		t.Error("Expected Q2 filtered out (no action)")
	}
}

func TestListUserIntents_WithConfidenceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"timestamp":1705000000,"question":"Explicit Q","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":1705000001,"question":"Inferred Q","response":"A2","confidence":"inferred","source":"hook_prompt"}`

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents", "--confidence", "explicit"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Explicit Q") {
		t.Error("Expected 'Explicit Q' in output")
	}
	if strings.Contains(output, "Inferred Q") {
		t.Error("Expected 'Inferred Q' filtered out")
	}
}

func TestListUserIntents_WithSinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	// One recent, one old
	now := time.Now()
	recentTimestamp := now.AddDate(0, 0, -2).Unix()
	oldTimestamp := now.AddDate(0, 0, -30).Unix()

	content := `{"timestamp":` + itoa(recentTimestamp) + `,"question":"Recent Q","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":` + itoa(oldTimestamp) + `,"question":"Old Q","response":"A2","confidence":"explicit","source":"ask_user"}`

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents", "--since", "7d"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Recent Q") {
		t.Error("Expected 'Recent Q' in output")
	}
	if strings.Contains(output, "Old Q") {
		t.Error("Expected 'Old Q' filtered out (>7 days)")
	}
}

func TestListUserIntents_TableFormat(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	content := `{"timestamp":1705000000,"question":"Should I use React?","response":"Yes, use React","confidence":"explicit","source":"ask_user"}`

	os.WriteFile(filepath.Join(claudeDir, "user-intents.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "Timestamp") {
		t.Error("Expected 'Timestamp' header")
	}
	if !strings.Contains(output, "Category") {
		t.Error("Expected 'Category' header")
	}
	if !strings.Contains(output, "Source") {
		t.Error("Expected 'Source' header")
	}
	if !strings.Contains(output, "Question") {
		t.Error("Expected 'Question' header")
	}
	if !strings.Contains(output, "Response") {
		t.Error("Expected 'Response' header")
	}

	// Verify data
	if !strings.Contains(output, "ask_user") {
		t.Error("Expected ask_user source")
	}
	// Note: Category column now shown instead of confidence in table
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestTruncateForTable(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"short", 5, "short"},
		{"exactfit", 8, "exactfit"},
		{"a", 3, "a"},
		{"ab", 3, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},     // maxLen <= 3, no ellipsis
		{"abcde", 4, "a..."},   // maxLen > 3, with ellipsis
		{"", 10, ""},
		{"verylongstring", 7, "very..."},
	}

	for _, tt := range tests {
		result := truncateForTable(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateForTable(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestParseSinceFilter_Duration(t *testing.T) {
	// Test duration format
	result := parseSinceFilter("7d")
	expected := time.Now().AddDate(0, 0, -7)

	// Allow 1 second tolerance for time comparison
	diff := result.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("parseSinceFilter(7d) time mismatch: got %v, want ~%v", result, expected)
	}
}

func TestParseSinceFilter_Date(t *testing.T) {
	// Test date format
	result := parseSinceFilter("2026-01-15")
	expected := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	if !result.Equal(expected) {
		t.Errorf("parseSinceFilter(2026-01-15) = %v, want %v", result, expected)
	}
}

// ============================================================================
// Help Text Integration Tests
// ============================================================================

func TestPrintHelp_IncludesSharpEdgesCommands(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify sharp-edges section
	if !strings.Contains(output, "Sharp Edge Commands:") {
		t.Error("Expected 'Sharp Edge Commands:' section in help")
	}
	if !strings.Contains(output, "sharp-edges") {
		t.Error("Expected 'sharp-edges' subcommand in help")
	}
	if !strings.Contains(output, "--severity") {
		t.Error("Expected '--severity' flag in help")
	}
	if !strings.Contains(output, "--unresolved") {
		t.Error("Expected '--unresolved' flag in help")
	}
	if !strings.Contains(output, "--error-type") {
		t.Error("Expected '--error-type' flag in help")
	}
}

func TestPrintHelp_IncludesUserIntentsCommands(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify user-intents section
	if !strings.Contains(output, "User Intent Commands:") {
		t.Error("Expected 'User Intent Commands:' section in help")
	}
	if !strings.Contains(output, "user-intents") {
		t.Error("Expected 'user-intents' subcommand in help")
	}
	if !strings.Contains(output, "--source") {
		t.Error("Expected '--source' flag in help")
	}
	if !strings.Contains(output, "--confidence") {
		t.Error("Expected '--confidence' flag in help")
	}
	if !strings.Contains(output, "--has-action") {
		t.Error("Expected '--has-action' flag in help")
	}
}

// ============================================================================
// Missing File Tests (graceful handling)
// ============================================================================

func TestListSharpEdges_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Do NOT create pending-learnings.jsonl

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show empty message, not error
	if !strings.Contains(output, "No sharp edges recorded") {
		t.Errorf("Expected graceful empty message for missing file, got: %s", output)
	}
}

func TestListUserIntents_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Do NOT create user-intents.jsonl

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "user-intents"}
	defer func() { os.Args = oldArgs }()

	listUserIntents()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show empty message, not error
	if !strings.Contains(output, "No user intents recorded") {
		t.Errorf("Expected graceful empty message for missing file, got: %s", output)
	}
}

// ============================================================================
// Default Severity Display Test
// ============================================================================

func TestListSharpEdges_DefaultSeverityDash(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(claudeDir, 0755)

	// Edge without severity field
	content := `{"file":"no-severity.go","error_type":"test_error","consecutive_failures":3,"timestamp":1705000000}`

	os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(content), 0644)

	os.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
	defer os.Unsetenv("GOYOKE_PROJECT_DIR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"goyoke-archive", "sharp-edges"}
	defer func() { os.Args = oldArgs }()

	listSharpEdges()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should display "-" for missing severity
	if !strings.Contains(output, "-") {
		t.Error("Expected '-' for missing severity")
	}
}

// Helper to convert int64 to string
func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
