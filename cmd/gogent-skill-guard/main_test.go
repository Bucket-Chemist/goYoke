package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuardMode_NoGuardFile(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	output := handleGuardMode("Glob", guardPath)
	assert.Equal(t, "{}", output)
}

func TestGuardMode_AllowedTool(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read", "AskUserQuestion", "Skill"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	output := handleGuardMode("Task", guardPath)
	assert.Equal(t, "{}", output)
}

func TestGuardMode_BlockedTool(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read", "AskUserQuestion", "Skill"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	output := handleGuardMode("Glob", guardPath)
	assert.Contains(t, output, "skill-guard")
	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "braintrust")
}

func TestGuardMode_StaleGuard(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read"},
		CreatedAt:          time.Now().Add(-31 * time.Minute).UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	output := handleGuardMode("Glob", guardPath)
	assert.Equal(t, "{}", output)

	// File should be deleted
	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "guard file should be deleted")
}

func TestGuardMode_EmptyAllowedTools(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	output := handleGuardMode("Task", guardPath)
	assert.Contains(t, output, "skill-guard")
	assert.Contains(t, output, "blocked")
}

func TestGuardMode_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)
	require.NoError(t, os.WriteFile(guardPath, []byte("not valid json"), 0644))

	output := handleGuardMode("Glob", guardPath)
	assert.Equal(t, "{}", output)

	// File should be deleted
	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "malformed guard file should be deleted")
}

func TestGuardMode_MultipleAllowedTools(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "review",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Test all allowed tools pass through
	for _, tool := range guard.RouterAllowedTools {
		output := handleGuardMode(tool, guardPath)
		assert.Equal(t, "{}", output, "tool %s should be allowed", tool)
	}

	// Test blocked tool
	output := handleGuardMode("Write", guardPath)
	assert.Contains(t, output, "blocked")
}

func TestGuardMode_InvalidCreatedAtTime(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task"},
		CreatedAt:          "invalid-time-format",
	}
	writeGuardFile(t, guardPath, guard)

	// Should still check allowed tools even if time parse fails
	output := handleGuardMode("Task", guardPath)
	assert.Equal(t, "{}", output)

	output = handleGuardMode("Glob", guardPath)
	assert.Contains(t, output, "blocked")
}

func TestExtractSkillName(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "valid skill name",
			input:    map[string]interface{}{"skill": "braintrust"},
			expected: "braintrust",
		},
		{
			name:     "empty input",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "skill key with non-string value",
			input:    map[string]interface{}{"skill": 123},
			expected: "",
		},
		{
			name:     "skill key with empty string",
			input:    map[string]interface{}{"skill": ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSkillName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to write guard file
func writeGuardFile(t *testing.T, path string, guard *ActiveSkill) {
	t.Helper()
	data, err := json.MarshalIndent(guard, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}
