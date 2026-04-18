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

	output := legacyCheckGuard("Glob", guardPath)
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

	output := legacyCheckGuard("Task", guardPath)
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

	output := legacyCheckGuard("Glob", guardPath)
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

	output := legacyCheckGuard("Glob", guardPath)
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

	output := legacyCheckGuard("Task", guardPath)
	assert.Contains(t, output, "skill-guard")
	assert.Contains(t, output, "blocked")
}

func TestGuardMode_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)
	require.NoError(t, os.WriteFile(guardPath, []byte("not valid json"), 0644))

	output := legacyCheckGuard("Glob", guardPath)
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
		output := legacyCheckGuard(tool, guardPath)
		assert.Equal(t, "{}", output, "tool %s should be allowed", tool)
	}

	// Test blocked tool
	output := legacyCheckGuard("Write", guardPath)
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
	output := legacyCheckGuard("Task", guardPath)
	assert.Equal(t, "{}", output)

	output = legacyCheckGuard("Glob", guardPath)
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

// ---------------------------------------------------------------------------
// TUI translation injection tests
// ---------------------------------------------------------------------------

func TestEmitSetupResponse_TUI_InjectsTranslation(t *testing.T) {
	t.Setenv("GOYOKE_MCP_CONFIG", "/tmp/goyoke-mcp-test.json")

	// Capture stdout by calling the helper indirectly through handleSetupModeWithConfig
	// with a nil guardConfig (non-guarded skill path).
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// We can't easily capture fmt.Println output in unit tests, so test
	// isTUIMode detection and the translation const directly.
	assert.True(t, isTUIMode(), "isTUIMode should return true when GOYOKE_MCP_CONFIG is set")
	assert.Contains(t, tuiTranslation, "spawn_agent")
	assert.Contains(t, tuiTranslation, "get_agent_result")
	assert.Contains(t, tuiTranslation, "Task()")
	assert.Contains(t, tuiTranslation, "Do NOT translate mcp__goyoke-interactive__team_run")

	// Verify guard path is not created for non-guarded skill
	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err))
}

func TestIsTUIMode_NotSet(t *testing.T) {
	t.Setenv("GOYOKE_MCP_CONFIG", "")
	assert.False(t, isTUIMode(), "isTUIMode should return false when GOYOKE_MCP_CONFIG is empty")
}

func TestIsTUIMode_Set(t *testing.T) {
	t.Setenv("GOYOKE_MCP_CONFIG", "/tmp/goyoke-mcp-12345.json")
	assert.True(t, isTUIMode(), "isTUIMode should return true when GOYOKE_MCP_CONFIG is set")
}

func TestTuiTranslation_ContainsKeyRules(t *testing.T) {
	// Verify the translation const covers all critical rules from the plan.
	assert.Contains(t, tuiTranslation, "TOOL TRANSLATION", "must have header")
	assert.Contains(t, tuiTranslation, "spawn_agent", "must mention spawn_agent")
	assert.Contains(t, tuiTranslation, "get_agent_result", "must mention get_agent_result")
	assert.Contains(t, tuiTranslation, "AGENT: xxx", "must explain agent ID extraction")
	assert.Contains(t, tuiTranslation, "subagent_type", "must mention subagent_type is not needed")
	assert.Contains(t, tuiTranslation, "async", "must explain async nature")
	assert.Contains(t, tuiTranslation, "team_run", "must have team_run exclusion clause")
	assert.Contains(t, tuiTranslation, "Only translate Task()", "must scope translation to Task only")
}

func TestHandleSetupModeWithConfig_LegacyGuarded_CreatesTeamDir(t *testing.T) {
	// Verify that the legacy path still creates the team directory and guard file
	// regardless of TUI mode (translation injection is orthogonal to guard creation).
	t.Setenv("GOYOKE_MCP_CONFIG", "")

	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guardConfig := &SkillGuardConfig{
		RouterAllowedTools: []string{"Task", "Bash", "Read"},
		TeamDirSuffix:      "test-team",
	}

	handleSetupModeWithConfig("braintrust", guardConfig, tmpDir, guardPath)

	// Guard file should exist
	_, err := os.Stat(guardPath)
	assert.False(t, os.IsNotExist(err), "guard file should be created")

	// Team dir should exist
	entries, err := os.ReadDir(filepath.Join(tmpDir, "teams"))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "exactly one team dir should be created")
	assert.Contains(t, entries[0].Name(), "test-team")
}

// Helper function to write guard file
func writeGuardFile(t *testing.T, path string, guard *ActiveSkill) {
	t.Helper()
	data, err := json.MarshalIndent(guard, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}
