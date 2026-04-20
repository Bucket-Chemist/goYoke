package skillguard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
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

	guard := &config.ActiveSkill{
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

	guard := &config.ActiveSkill{
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

	guard := &config.ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read"},
		CreatedAt:          time.Now().Add(-31 * time.Minute).UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	output := legacyCheckGuard("Glob", guardPath)
	assert.Equal(t, "{}", output)

	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "guard file should be deleted")
}

func TestGuardMode_EmptyAllowedTools(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &config.ActiveSkill{
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

	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "malformed guard file should be deleted")
}

func TestGuardMode_MultipleAllowedTools(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &config.ActiveSkill{
		Skill:              "review",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	for _, tool := range guard.RouterAllowedTools {
		output := legacyCheckGuard(tool, guardPath)
		assert.Equal(t, "{}", output, "tool %s should be allowed", tool)
	}

	output := legacyCheckGuard("Write", guardPath)
	assert.Contains(t, output, "blocked")
}

func TestGuardMode_InvalidCreatedAtTime(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	guard := &config.ActiveSkill{
		Skill:              "braintrust",
		TeamDir:            "/tmp/test-team",
		RouterAllowedTools: []string{"Task"},
		CreatedAt:          "invalid-time-format",
	}
	writeGuardFile(t, guardPath, guard)

	output := legacyCheckGuard("Task", guardPath)
	assert.Equal(t, "{}", output)

	output = legacyCheckGuard("Glob", guardPath)
	assert.Contains(t, output, "blocked")
}

func writeGuardFile(t *testing.T, path string, guard *config.ActiveSkill) {
	t.Helper()
	data, err := json.MarshalIndent(guard, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}
