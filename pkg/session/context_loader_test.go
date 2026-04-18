package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHandoffSummary_Exists(t *testing.T) {
	// Setup: Create temp directory with handoff file
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	handoffPath := filepath.Join(claudeDir, "last-handoff.md")
	content := `# Session Handoff

## Context
Previous session completed goYoke-058.

## Sharp Edges
- File parsing failed on malformed JSON
- Required validation before processing

## Actions
1. Review pending learnings
2. Continue with goYoke-059
`
	require.NoError(t, os.WriteFile(handoffPath, []byte(content), 0644))

	// Execute
	result, err := LoadHandoffSummary(projectDir)

	// Verify
	require.NoError(t, err)
	assert.Contains(t, result, "# Session Handoff")
	assert.Contains(t, result, "Previous session completed goYoke-058")
	assert.Contains(t, result, "File parsing failed on malformed JSON")
	assert.NotContains(t, result, "truncated") // File is short, no truncation
}

func TestLoadHandoffSummary_Missing(t *testing.T) {
	// Setup: Empty project directory (no handoff file)
	projectDir := t.TempDir()

	// Execute
	result, err := LoadHandoffSummary(projectDir)

	// Verify: Missing file returns empty string, NOT error
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestLoadHandoffSummary_Truncation(t *testing.T) {
	// Setup: Create handoff file with 40 lines (should truncate to 30)
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	handoffPath := filepath.Join(claudeDir, "last-handoff.md")

	// Build 40-line content
	var lines []string
	lines = append(lines, "# Session Handoff")
	lines = append(lines, "")
	lines = append(lines, "## Context")
	for i := 1; i <= 37; i++ {
		lines = append(lines, "Line "+string(rune('A'+i%26)))
	}
	content := strings.Join(lines, "\n")
	require.NoError(t, os.WriteFile(handoffPath, []byte(content), 0644))

	// Execute
	result, err := LoadHandoffSummary(projectDir)

	// Verify
	require.NoError(t, err)

	// Should contain first 30 lines
	resultLines := strings.Split(result, "\n")
	firstContentLines := resultLines[:30] // Exclude truncation message
	assert.Equal(t, 30, len(firstContentLines))
	assert.Equal(t, "# Session Handoff", firstContentLines[0])

	// Should contain truncation notice
	assert.Contains(t, result, "10 lines truncated")
	assert.Contains(t, result, handoffPath)
}

func TestLoadHandoffSummary_TooLarge(t *testing.T) {
	// Setup: Create file larger than 50KB
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	handoffPath := filepath.Join(claudeDir, "last-handoff.md")

	// Create 60KB of content (larger than 50KB limit)
	largeContent := strings.Repeat("This is a long line of text to exceed the 50KB limit.\n", 1200)
	require.NoError(t, os.WriteFile(handoffPath, []byte(largeContent), 0644))

	// Execute
	result, err := LoadHandoffSummary(projectDir)

	// Verify: Should return error for oversized file
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
	assert.Contains(t, err.Error(), "bytes") // Check for byte count in message
	assert.Empty(t, result)
}

func TestCheckPendingLearnings_HasLearnings(t *testing.T) {
	// Setup: Create pending learnings file with 3 sharp edges
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	learningsPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"pkg/session/handoff.go","error_type":"parse_error","consecutive_failures":3}
{"file":"pkg/config/loader.go","error_type":"type_mismatch","consecutive_failures":4}
{"file":"cmd/validate/main.go","error_type":"nil_pointer","consecutive_failures":3}
`
	require.NoError(t, os.WriteFile(learningsPath, []byte(content), 0644))

	// Execute
	result, err := CheckPendingLearnings(projectDir)

	// Verify
	require.NoError(t, err)
	assert.Contains(t, result, "PENDING LEARNINGS")
	assert.Contains(t, result, "3 sharp edge(s)")
	assert.Contains(t, result, learningsPath)
}

func TestCheckPendingLearnings_None(t *testing.T) {
	// Setup: No pending learnings file exists
	projectDir := t.TempDir()

	// Execute
	result, err := CheckPendingLearnings(projectDir)

	// Verify: Missing file returns empty string, NOT error
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestCheckPendingLearnings_EmptyFile(t *testing.T) {
	// Setup: Empty pending learnings file
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	learningsPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	require.NoError(t, os.WriteFile(learningsPath, []byte(""), 0644))

	// Execute
	result, err := CheckPendingLearnings(projectDir)

	// Verify: Empty file returns empty string, NOT error
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestCheckPendingLearnings_LargeLine(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".goyoke", "memory")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	learningsPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	largeLine := strings.Repeat("x", 70*1024)
	require.NoError(t, os.WriteFile(learningsPath, []byte(largeLine+"\n"), 0644))

	result, err := CheckPendingLearnings(projectDir)
	require.NoError(t, err)
	assert.Contains(t, result, "1 sharp edge(s)")
}

func TestFormatGitInfo_NotGitRepo(t *testing.T) {
	// Note: t.TempDir() may be inside a git repo if tests run inside repo root
	// Git's --is-inside-work-tree traverses upward, so temp dirs inherit git context
	//
	// This test validates that FormatGitInfo handles non-git directories gracefully
	// by returning empty string. However, when run inside a git repo, the temp
	// directory is still considered part of that repo by git.
	//
	// We test the graceful degradation by checking if result is valid format or empty.
	projectDir := t.TempDir()
	result := FormatGitInfo(projectDir)

	// If we're inside a git repo (tests run in goYoke), accept git output
	// If truly not in git repo, expect empty string
	if result != "" {
		// Verify it has correct git format
		assert.Contains(t, result, "GIT: Branch:")
	} else {
		// Empty string is also valid (true non-git directory)
		assert.Equal(t, "", result)
	}
}

func TestFormatGitInfo_CleanRepo(t *testing.T) {
	// Setup: This test runs in actual goYoke repo
	// We'll test the format rather than exact values since git state varies

	// Skip if not in git repo (CI environments)
	projectDir := "."
	result := FormatGitInfo(projectDir)

	if result == "" {
		t.Skip("Not in git repository, skipping git info test")
	}

	// Verify format regardless of actual branch name
	assert.Contains(t, result, "GIT: Branch:")
	// Should contain either "Clean working tree" or "Uncommitted: N file(s)"
	if strings.Contains(result, "Clean") {
		assert.Contains(t, result, "Clean working tree")
	} else {
		assert.Contains(t, result, "Uncommitted:")
		assert.Contains(t, result, "file(s)")
	}
}

func TestFormatGitInfo_DirtyRepo(t *testing.T) {
	// This test validates the formatting logic, not actual git state
	// We test via the underlying collectGitInfo behavior

	projectDir := "."
	result := FormatGitInfo(projectDir)

	if result == "" {
		t.Skip("Not in git repository, skipping dirty repo test")
	}

	// Verify structure (exact content depends on actual repo state)
	parts := strings.Split(result, "|")
	assert.Equal(t, 2, len(parts), "Should have branch and status sections separated by |")
	assert.Contains(t, parts[0], "Branch:")

	// Second part should be either "Clean working tree" or "Uncommitted: N file(s)"
	secondPart := strings.TrimSpace(parts[1])
	assert.True(t,
		strings.Contains(secondPart, "Clean working tree") ||
			strings.Contains(secondPart, "Uncommitted:"),
		"Status should indicate clean or dirty state")
}
