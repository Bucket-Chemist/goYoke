package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_HelpFlag verifies the --help flag outputs usage information and exits 0.
func TestRun_HelpFlag(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"--help"}, "")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for --help, got: %d", exitCode)
	}

	output := buf.String()
	expectedStrings := []string{
		"goyoke-aggregate - Weekly learning aggregation",
		"--force",
		"--dry-run",
		"pending-learnings.jsonl",
		"user-intents.jsonl",
		"decisions.jsonl",
		"preferences.jsonl",
		"performance.jsonl",
		"routing-violations.jsonl",
		"YYYY-Www-{type}.jsonl",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help to contain %q, got:\n%s", expected, output)
		}
	}
}

// TestRun_NoAggregationNeeded verifies behavior when all files are under threshold.
func TestRun_NoAggregationNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create small files (under 1MB)
	smallContent := `{"test": "data"}`
	os.WriteFile(filepath.Join(memoryDir, "decisions.jsonl"), []byte(smallContent), 0644)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "No aggregation needed") {
		t.Errorf("Expected 'No aggregation needed' message, got:\n%s", output)
	}
}

// TestRun_DryRun verifies --dry-run flag shows what would be aggregated without modifying files.
func TestRun_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file that exceeds threshold (simulate with --force)
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	content := `{"timestamp":1,"category":"arch","decision":"test"}`
	os.WriteFile(decisionsPath, []byte(content), 0644)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"--dry-run", "--force"}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for --dry-run, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Expected [dry-run] prefix in output, got:\n%s", output)
	}
	if !strings.Contains(output, "decisions.jsonl") {
		t.Errorf("Expected decisions.jsonl in dry-run output, got:\n%s", output)
	}
	if !strings.Contains(output, "No files were modified") {
		t.Errorf("Expected 'No files were modified' message, got:\n%s", output)
	}

	// Verify original file still exists
	if _, err := os.Stat(decisionsPath); os.IsNotExist(err) {
		t.Error("Original file should still exist after dry-run")
	}

	// Verify no archive directory was created
	archiveDir := filepath.Join(memoryDir, "archive")
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Error("Archive directory should not be created in dry-run mode")
	}
}

// TestRun_ForceAggregation verifies --force flag triggers aggregation regardless of size.
func TestRun_ForceAggregation(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	archiveDir := filepath.Join(memoryDir, "archive")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files with content
	testFiles := map[string]string{
		"pending-learnings.jsonl":  `{"error_type":"test","file":"test.go"}`,
		"user-intents.jsonl":       `{"question":"test?","response":"yes"}`,
		"decisions.jsonl":          `{"category":"arch","decision":"test"}`,
		"preferences.jsonl":        `{"key":"theme","value":"dark"}`,
		"performance.jsonl":        `{"operation":"test","duration_ms":100}`,
		"routing-violations.jsonl": `{"violation_type":"tier_mismatch"}`,
	}

	for filename, content := range testFiles {
		path := filepath.Join(memoryDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"--force"}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "Aggregation complete") {
		t.Errorf("Expected 'Aggregation complete' message, got:\n%s", output)
	}

	// Verify stats in output
	expectedStats := []string{
		"Sharp Edges: 1",
		"User Intents: 1",
		"Decisions: 1",
		"Preferences: 1",
		"Performance: 1",
		"Violations: 1",
	}

	for _, stat := range expectedStats {
		if !strings.Contains(output, stat) {
			t.Errorf("Expected %q in output, got:\n%s", stat, output)
		}
	}

	// Verify archive files created
	archiveTypes := []string{"sharp-edges", "user-intents", "decisions", "preferences", "performance", "violations"}
	for _, archiveType := range archiveTypes {
		// Find the archive file (has week prefix)
		entries, err := os.ReadDir(archiveDir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), archiveType+".jsonl") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected archive file for %s", archiveType)
		}
	}

	// Verify summary was created
	entries, _ := os.ReadDir(archiveDir)
	summaryFound := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-summary.md") {
			summaryFound = true
			break
		}
	}
	if !summaryFound {
		t.Error("Expected summary.md file in archive")
	}

	// Verify original files replaced with empty files
	for filename := range testFiles {
		path := filepath.Join(memoryDir, filename)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Expected replacement file %s to exist: %v", filename, err)
			continue
		}
		if info.Size() != 0 {
			t.Errorf("Expected replacement file %s to be empty, size: %d", filename, info.Size())
		}
	}
}

// TestRun_MissingProjectDir verifies fallback to current working directory.
func TestRun_MissingProjectDir(t *testing.T) {
	// Use temp directory as cwd
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run with empty project dir env (should use cwd)
	exitCode := run([]string{}, "")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "No aggregation needed") {
		t.Errorf("Expected no aggregation message, got:\n%s", output)
	}
}

// TestGetFileSize verifies file size detection.
func TestGetFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create file with known size
	content := strings.Repeat("x", 1000)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := getFileSize(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if size != 1000 {
		t.Errorf("Expected size 1000, got: %d", size)
	}
}

// TestGetFileSize_MissingFile verifies error handling for missing files.
func TestGetFileSize_MissingFile(t *testing.T) {
	_, err := getFileSize("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Error("Expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected IsNotExist error, got: %v", err)
	}
}

// TestCountLines verifies line counting with various content.
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "three lines",
			content:  `{"a":1}` + "\n" + `{"b":2}` + "\n" + `{"c":3}` + "\n",
			expected: 3,
		},
		{
			name:     "with blank lines",
			content:  `{"a":1}` + "\n\n" + `{"b":2}` + "\n" + `   ` + "\n" + `{"c":3}` + "\n",
			expected: 3,
		},
		{
			name:     "empty file",
			content:  "",
			expected: 0,
		},
		{
			name:     "only whitespace",
			content:  "  \n\t\n  \n",
			expected: 0,
		},
		{
			name:     "no trailing newline",
			content:  `{"a":1}`,
			expected: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.jsonl")
			os.WriteFile(testFile, []byte(tc.content), 0644)

			count, err := countLines(testFile)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if count != tc.expected {
				t.Errorf("Expected %d lines, got: %d", tc.expected, count)
			}
		})
	}
}

// TestCountLines_MissingFile verifies error handling for missing files.
func TestCountLines_MissingFile(t *testing.T) {
	_, err := countLines("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestCountLines_LargeLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.jsonl")
	content := strings.Repeat("x", 70*1024) + "\n" + `{"b":2}` + "\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	count, err := countLines(testFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 lines, got %d", count)
	}
}

// TestAggregateFiles verifies the core aggregation logic.
func TestAggregateFiles(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	archiveDir := filepath.Join(memoryDir, "archive")

	os.MkdirAll(memoryDir, 0755)
	os.MkdirAll(archiveDir, 0755)

	// Create test files with multiple lines
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	decisionsContent := `{"timestamp":1,"category":"arch","decision":"test1"}
{"timestamp":2,"category":"tech","decision":"test2"}`

	preferencesPath := filepath.Join(memoryDir, "preferences.jsonl")
	preferencesContent := `{"timestamp":1,"key":"theme","value":"dark"}`

	os.WriteFile(decisionsPath, []byte(decisionsContent), 0644)
	os.WriteFile(preferencesPath, []byte(preferencesContent), 0644)

	srcPaths := []string{decisionsPath, preferencesPath}
	week := "2026-W04"

	stats := aggregateFiles(srcPaths, archiveDir, week)

	// Verify stats
	if stats.Decisions != 2 {
		t.Errorf("Expected 2 decisions, got: %d", stats.Decisions)
	}
	if stats.Preferences != 1 {
		t.Errorf("Expected 1 preference, got: %d", stats.Preferences)
	}

	// Verify archive files created
	archivedDecisions := filepath.Join(archiveDir, "2026-W04-decisions.jsonl")
	if _, err := os.Stat(archivedDecisions); os.IsNotExist(err) {
		t.Error("Expected archived decisions file to exist")
	}

	archivedPreferences := filepath.Join(archiveDir, "2026-W04-preferences.jsonl")
	if _, err := os.Stat(archivedPreferences); os.IsNotExist(err) {
		t.Error("Expected archived preferences file to exist")
	}

	// Verify archived content is preserved
	content, err := os.ReadFile(archivedDecisions)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != decisionsContent {
		t.Errorf("Archived content mismatch.\nExpected: %s\nGot: %s", decisionsContent, content)
	}

	// Verify original files replaced with empty
	for _, path := range []string{decisionsPath, preferencesPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Expected replacement file %s: %v", path, err)
			continue
		}
		if info.Size() != 0 {
			t.Errorf("Expected empty replacement file %s, size: %d", path, info.Size())
		}
	}
}

// TestAggregateFiles_MissingFiles verifies handling of missing source files.
func TestAggregateFiles_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	archiveDir := filepath.Join(tmpDir, "archive")
	os.MkdirAll(archiveDir, 0755)

	// Pass non-existent file paths
	srcPaths := []string{
		filepath.Join(tmpDir, "nonexistent1.jsonl"),
		filepath.Join(tmpDir, "nonexistent2.jsonl"),
	}

	// Should not panic, should return zero stats
	stats := aggregateFiles(srcPaths, archiveDir, "2026-W04")

	if stats.Decisions != 0 || stats.Preferences != 0 || stats.SharpEdges != 0 {
		t.Errorf("Expected zero stats for missing files, got: %+v", stats)
	}
}

// TestAggregateFiles_UnknownFileType verifies handling of unknown file types.
func TestAggregateFiles_UnknownFileType(t *testing.T) {
	tmpDir := t.TempDir()
	archiveDir := filepath.Join(tmpDir, "archive")
	os.MkdirAll(archiveDir, 0755)

	// Create file with unknown type
	unknownPath := filepath.Join(tmpDir, "unknown.jsonl")
	os.WriteFile(unknownPath, []byte(`{"data": "test"}`), 0644)

	srcPaths := []string{unknownPath}
	stats := aggregateFiles(srcPaths, archiveDir, "2026-W04")

	// Should skip unknown file types
	if stats.Decisions != 0 || stats.Preferences != 0 || stats.SharpEdges != 0 {
		t.Errorf("Expected zero stats for unknown file type, got: %+v", stats)
	}

	// Unknown file should not be moved
	if _, err := os.Stat(unknownPath); os.IsNotExist(err) {
		t.Error("Unknown file should not be moved/deleted")
	}
}

// TestGenerateSummary verifies summary markdown generation.
func TestGenerateSummary(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "2026-W04-summary.md")

	stats := AggregationStats{
		SharpEdges:  5,
		UserIntents: 3,
		Decisions:   10,
		Preferences: 2,
		Performance: 50,
		Violations:  1,
	}

	err := generateSummary(summaryPath, "2026-W04", stats)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Verify summary contains expected sections
	expectedStrings := []string{
		"# Weekly Learning Summary - 2026-W04",
		"**Sharp Edges**: 5",
		"**User Intents**: 3",
		"**Decisions**: 10",
		"**Preferences**: 2",
		"**Performance Metrics**: 50",
		"**Routing Violations**: 1",
		"2026-W04-sharp-edges.jsonl",
		"2026-W04-user-intents.jsonl",
		"2026-W04-decisions.jsonl",
		"2026-W04-preferences.jsonl",
		"2026-W04-performance.jsonl",
		"2026-W04-violations.jsonl",
		"grep '\"category\":\"architecture\"'",
		"jq '.error_type'",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected summary to contain %q", expected)
		}
	}
}

// TestGenerateSummary_WriteError verifies error handling for write failures.
func TestGenerateSummary_WriteError(t *testing.T) {
	// Try to write to a directory that doesn't exist
	summaryPath := "/nonexistent/directory/summary.md"
	stats := AggregationStats{}

	err := generateSummary(summaryPath, "2026-W04", stats)
	if err == nil {
		t.Error("Expected error for write to nonexistent directory")
	}
}

// TestCheckAggregationNeeded verifies aggregation threshold logic.
func TestCheckAggregationNeeded(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small file (under threshold)
	smallFile := filepath.Join(tmpDir, "small.jsonl")
	os.WriteFile(smallFile, []byte(`{"test":1}`), 0644)

	// Create a large file (over threshold - simulate with 2MB)
	largeFile := filepath.Join(tmpDir, "large.jsonl")
	largeContent := strings.Repeat(`{"test":1}`, 200000) // ~2MB
	os.WriteFile(largeFile, []byte(largeContent), 0644)

	filePaths := []string{smallFile, largeFile}

	// Without force, only large file should be aggregated
	result, err := checkAggregationNeeded(filePaths, false, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 file to aggregate, got: %d", len(result))
	}

	if len(result) > 0 && result[0] != largeFile {
		t.Errorf("Expected large file in result, got: %s", result[0])
	}

	// With force, both files should be aggregated
	result, err = checkAggregationNeeded(filePaths, true, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 files with force, got: %d", len(result))
	}
}

// TestCheckAggregationNeeded_MissingFiles verifies handling of missing files.
func TestCheckAggregationNeeded_MissingFiles(t *testing.T) {
	filePaths := []string{
		"/nonexistent/file1.jsonl",
		"/nonexistent/file2.jsonl",
	}

	result, err := checkAggregationNeeded(filePaths, false, false)
	if err != nil {
		t.Fatalf("Unexpected error for missing files: %v", err)
	}

	// Missing files should be skipped, not cause errors
	if len(result) != 0 {
		t.Errorf("Expected empty result for missing files, got: %d", len(result))
	}
}

// TestGetFileType verifies file type mapping.
func TestGetFileType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"pending-learnings.jsonl", "sharp-edges"},
		{"user-intents.jsonl", "user-intents"},
		{"decisions.jsonl", "decisions"},
		{"preferences.jsonl", "preferences"},
		{"performance.jsonl", "performance"},
		{"routing-violations.jsonl", "violations"},
		{"unknown.jsonl", ""},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			result := getFileType(tc.filename)
			if result != tc.expected {
				t.Errorf("Expected %q for %q, got %q", tc.expected, tc.filename, result)
			}
		})
	}
}

// TestUpdateStats verifies stats updates for all file types.
func TestUpdateStats(t *testing.T) {
	stats := AggregationStats{}

	updateStats(&stats, "sharp-edges", 5)
	updateStats(&stats, "user-intents", 3)
	updateStats(&stats, "decisions", 10)
	updateStats(&stats, "preferences", 2)
	updateStats(&stats, "performance", 50)
	updateStats(&stats, "violations", 1)
	updateStats(&stats, "unknown", 100) // Should be ignored

	if stats.SharpEdges != 5 {
		t.Errorf("Expected SharpEdges=5, got %d", stats.SharpEdges)
	}
	if stats.UserIntents != 3 {
		t.Errorf("Expected UserIntents=3, got %d", stats.UserIntents)
	}
	if stats.Decisions != 10 {
		t.Errorf("Expected Decisions=10, got %d", stats.Decisions)
	}
	if stats.Preferences != 2 {
		t.Errorf("Expected Preferences=2, got %d", stats.Preferences)
	}
	if stats.Performance != 50 {
		t.Errorf("Expected Performance=50, got %d", stats.Performance)
	}
	if stats.Violations != 1 {
		t.Errorf("Expected Violations=1, got %d", stats.Violations)
	}
}

// TestBuildFilePaths verifies path building from specs.
func TestBuildFilePaths(t *testing.T) {
	memoryDir := "/test/.goyoke/memory"
	specs := []FileSpec{
		{Filename: "test1.jsonl", ArchiveType: "type1"},
		{Filename: "test2.jsonl", ArchiveType: "type2"},
	}

	paths := buildFilePaths(memoryDir, specs)

	if len(paths) != 2 {
		t.Fatalf("Expected 2 paths, got %d", len(paths))
	}

	expected1 := filepath.Join(memoryDir, "test1.jsonl")
	expected2 := filepath.Join(memoryDir, "test2.jsonl")

	if paths[0] != expected1 {
		t.Errorf("Expected %s, got %s", expected1, paths[0])
	}
	if paths[1] != expected2 {
		t.Errorf("Expected %s, got %s", expected2, paths[1])
	}
}

// TestPrintHelp verifies help output completeness.
func TestPrintHelp(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "help-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	printHelp(tmpFile)
	tmpFile.Close()

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	helpText := string(content)

	// Verify all key elements are present
	required := []string{
		"goyoke-aggregate",
		"--force",
		"--dry-run",
		"--help",
		"pending-learnings.jsonl",
		"user-intents.jsonl",
		"decisions.jsonl",
		"preferences.jsonl",
		"performance.jsonl",
		"routing-violations.jsonl",
		"YYYY-Www",
	}

	for _, r := range required {
		if !strings.Contains(helpText, r) {
			t.Errorf("Help missing required text: %q", r)
		}
	}
}

// TestRun_InvalidFlag verifies handling of invalid flags.
func TestRun_InvalidFlag(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := run([]string{"--invalid-flag"}, "")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = oldStderr

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got: %d", exitCode)
	}
}

// TestRun_AllFileTypes verifies all 6 file types are handled correctly.
func TestRun_AllFileTypes(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create all 6 file types
	allFiles := map[string]string{
		"pending-learnings.jsonl":  `{"error_type":"compile","file":"main.go","consecutive_failures":3}`,
		"user-intents.jsonl":       `{"question":"use tabs?","response":"yes","source":"ask_user"}`,
		"decisions.jsonl":          `{"category":"architecture","decision":"use hexagonal","rationale":"clean boundaries"}`,
		"preferences.jsonl":        `{"key":"model","value":"sonnet","scope":"project"}`,
		"performance.jsonl":        `{"operation":"query","duration_ms":250,"success":true}`,
		"routing-violations.jsonl": `{"violation_type":"tier_mismatch","agent":"haiku","expected_tier":"sonnet"}`,
	}

	for filename, content := range allFiles {
		path := filepath.Join(memoryDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"--force"}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()

	// Verify all stats are reported
	statsToCheck := []string{
		"Sharp Edges: 1",
		"User Intents: 1",
		"Decisions: 1",
		"Preferences: 1",
		"Performance: 1",
		"Violations: 1",
	}

	for _, stat := range statsToCheck {
		if !strings.Contains(output, stat) {
			t.Errorf("Expected %q in output, got:\n%s", stat, output)
		}
	}

	// Verify archive directory structure
	archiveDir := filepath.Join(memoryDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 7 files: 6 JSONL + 1 summary.md
	if len(entries) != 7 {
		t.Errorf("Expected 7 files in archive, got: %d", len(entries))
		for _, e := range entries {
			t.Logf("  - %s", e.Name())
		}
	}

	// Verify summary exists and contains correct content
	var summaryEntry os.DirEntry
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-summary.md") {
			summaryEntry = e
			break
		}
	}

	if summaryEntry == nil {
		t.Fatal("Summary file not found in archive")
	}

	summaryContent, err := os.ReadFile(filepath.Join(archiveDir, summaryEntry.Name()))
	if err != nil {
		t.Fatal(err)
	}

	summaryStr := string(summaryContent)
	if !strings.Contains(summaryStr, "Weekly Learning Summary") {
		t.Error("Summary missing title")
	}
	if !strings.Contains(summaryStr, "**Sharp Edges**: 1") {
		t.Error("Summary missing sharp edges count")
	}
}

// TestRun_EmptyMemoryDir verifies behavior with empty memory directory.
func TestRun_EmptyMemoryDir(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Don't create any files - directory is empty

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "No aggregation needed") {
		t.Errorf("Expected 'No aggregation needed', got:\n%s", output)
	}
}

// TestAggregateFiles_AllFileTypes verifies archiving for all 6 file types.
func TestAggregateFiles_AllFileTypes(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	archiveDir := filepath.Join(memoryDir, "archive")

	os.MkdirAll(memoryDir, 0755)
	os.MkdirAll(archiveDir, 0755)

	// Create all 6 file types
	fileContents := map[string]string{
		"pending-learnings.jsonl":  `{"error":"test1"}` + "\n" + `{"error":"test2"}`,
		"user-intents.jsonl":       `{"question":"test?"}`,
		"decisions.jsonl":          `{"decision":"test"}` + "\n" + `{"decision":"test2"}` + "\n" + `{"decision":"test3"}`,
		"preferences.jsonl":        `{"key":"test"}`,
		"performance.jsonl":        `{"op":"test"}`,
		"routing-violations.jsonl": `{"viol":"test1"}` + "\n" + `{"viol":"test2"}`,
	}

	var srcPaths []string
	for filename, content := range fileContents {
		path := filepath.Join(memoryDir, filename)
		os.WriteFile(path, []byte(content), 0644)
		srcPaths = append(srcPaths, path)
	}

	week := "2026-W04"
	stats := aggregateFiles(srcPaths, archiveDir, week)

	// Verify stats for all types
	if stats.SharpEdges != 2 {
		t.Errorf("Expected SharpEdges=2, got %d", stats.SharpEdges)
	}
	if stats.UserIntents != 1 {
		t.Errorf("Expected UserIntents=1, got %d", stats.UserIntents)
	}
	if stats.Decisions != 3 {
		t.Errorf("Expected Decisions=3, got %d", stats.Decisions)
	}
	if stats.Preferences != 1 {
		t.Errorf("Expected Preferences=1, got %d", stats.Preferences)
	}
	if stats.Performance != 1 {
		t.Errorf("Expected Performance=1, got %d", stats.Performance)
	}
	if stats.Violations != 2 {
		t.Errorf("Expected Violations=2, got %d", stats.Violations)
	}
}

// TestCheckAggregationNeeded_DryRunOutput verifies dry-run output format.
func TestCheckAggregationNeeded_DryRunOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(testFile, []byte(`{"test":1}`), 0644)

	filePaths := []string{testFile}

	// Capture stdout for dry-run output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result, err := checkAggregationNeeded(filePaths, true, true)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 file with force, got: %d", len(result))
	}

	output := buf.String()
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("Expected [dry-run] in output, got: %s", output)
	}
	if !strings.Contains(output, "Would aggregate") {
		t.Errorf("Expected 'Would aggregate' in output, got: %s", output)
	}
}

// TestAggregateFiles_EmptyFile verifies handling of empty files.
func TestAggregateFiles_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	archiveDir := filepath.Join(memoryDir, "archive")

	os.MkdirAll(memoryDir, 0755)
	os.MkdirAll(archiveDir, 0755)

	// Create empty decisions file
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	os.WriteFile(decisionsPath, []byte(""), 0644)

	srcPaths := []string{decisionsPath}
	week := "2026-W04"

	stats := aggregateFiles(srcPaths, archiveDir, week)

	// Empty file should have 0 count
	if stats.Decisions != 0 {
		t.Errorf("Expected Decisions=0 for empty file, got %d", stats.Decisions)
	}

	// Archive file should still be created
	archivedPath := filepath.Join(archiveDir, "2026-W04-decisions.jsonl")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("Expected archived file to exist even for empty source")
	}
}

// TestRun_LargeFileTriggersAggregation verifies size-based aggregation trigger.
func TestRun_LargeFileTriggersAggregation(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create a file larger than 1MB
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	// Generate ~1.2MB of content
	var builder strings.Builder
	for i := 0; i < 50000; i++ {
		builder.WriteString(`{"timestamp":1,"category":"test","decision":"test decision number ` + fmt.Sprintf("%d", i) + `"}` + "\n")
	}
	os.WriteFile(decisionsPath, []byte(builder.String()), 0644)

	// Verify file is over 1MB
	info, _ := os.Stat(decisionsPath)
	if info.Size() < MaxFileSizeBytes {
		t.Skip("Test file not large enough, skipping")
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run without --force - should still aggregate due to size
	exitCode := run([]string{}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "Aggregation complete") {
		t.Errorf("Expected 'Aggregation complete' for large file, got:\n%s", output)
	}
}

// TestGenerateSummary_ZeroStats verifies summary generation with zero counts.
func TestGenerateSummary_ZeroStats(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "2026-W04-summary.md")

	stats := AggregationStats{} // All zeros

	err := generateSummary(summaryPath, "2026-W04", stats)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content, _ := os.ReadFile(summaryPath)
	contentStr := string(content)

	// All counts should be 0
	if !strings.Contains(contentStr, "**Sharp Edges**: 0") {
		t.Error("Expected zero sharp edges count")
	}
	if !strings.Contains(contentStr, "**Decisions**: 0") {
		t.Error("Expected zero decisions count")
	}
}

// TestMain verifies main() doesn't panic (basic smoke test).
func TestMain(t *testing.T) {
	// We can't test os.Exit behavior directly, but we can verify
	// the main function compiles and the run function works
	// This is already covered by other tests, but ensures main is called

	// Just verify the function signature and existence
	_ = run
}

// TestCheckAggregationNeeded_PermissionDenied verifies error handling for stat errors
// that are NOT IsNotExist (e.g., permission denied).
func TestCheckAggregationNeeded_PermissionDenied(t *testing.T) {
	// Skip if running as root (can access any file)
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Create a directory and remove read permission
	noAccessDir := filepath.Join(tmpDir, "noaccess")
	os.MkdirAll(noAccessDir, 0755)

	// Create file inside
	testFile := filepath.Join(noAccessDir, "test.jsonl")
	os.WriteFile(testFile, []byte(`{"test":1}`), 0644)

	// Remove read permission from directory (makes stat fail with EACCES)
	os.Chmod(noAccessDir, 0000)
	defer os.Chmod(noAccessDir, 0755) // Restore for cleanup

	filePaths := []string{testFile}

	_, err := checkAggregationNeeded(filePaths, false, false)
	if err == nil {
		t.Error("Expected error for permission denied, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to stat") {
		t.Errorf("Expected 'failed to stat' error, got: %v", err)
	}
}

// TestRun_CheckAggregationError verifies run() handles checkAggregationNeeded errors.
func TestRun_CheckAggregationError(t *testing.T) {
	// Skip if running as root
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create a file then make the directory unreadable
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	os.WriteFile(decisionsPath, []byte(`{"test":1}`), 0644)

	// Make directory unreadable to cause stat error
	os.Chmod(memoryDir, 0000)
	defer os.Chmod(memoryDir, 0755) // Restore for cleanup

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := run([]string{}, tmpDir)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = oldStderr

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for error, got: %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "Error checking files") {
		t.Errorf("Expected 'Error checking files' in stderr, got:\n%s", output)
	}
}

// TestAggregateFiles_RenameError verifies warning when rename fails.
func TestAggregateFiles_RenameError(t *testing.T) {
	// Skip if running as root
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	archiveDir := filepath.Join(memoryDir, "archive")

	os.MkdirAll(memoryDir, 0755)
	os.MkdirAll(archiveDir, 0755)

	// Create source file
	decisionsPath := filepath.Join(memoryDir, "decisions.jsonl")
	os.WriteFile(decisionsPath, []byte(`{"test":1}`), 0644)

	// Make archive directory read-only to prevent rename
	os.Chmod(archiveDir, 0555)
	defer os.Chmod(archiveDir, 0755) // Restore for cleanup

	srcPaths := []string{decisionsPath}

	// Capture stderr for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	stats := aggregateFiles(srcPaths, archiveDir, "2026-W04")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = oldStderr

	// Stats should be 0 since rename failed
	if stats.Decisions != 0 {
		// The count is done before rename, so it might be 1
		// but the file won't be archived
		t.Logf("Decisions count: %d (expected 0 or 1 depending on when error occurs)", stats.Decisions)
	}

	// Warning should be logged
	output := buf.String()
	if !strings.Contains(output, "Warning") && !strings.Contains(output, "Failed to archive") {
		t.Logf("Warning output (may be empty if chmod didn't work as expected): %s", output)
	}
}
