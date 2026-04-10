package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFixturePath returns the path to the test agents-index.json
func testFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "agents-index.json")
}

func TestSetupMode_BraintrustCreatesTeamDir(t *testing.T) {
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	config := loadSkillGuardConfigFrom("braintrust", testFixturePath(t))
	require.NotNil(t, config, "braintrust config should exist in fixture")

	handleSetupModeWithConfig("braintrust", config, sessionDir, guardPath)

	// Verify guard file
	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard ActiveSkill
	require.NoError(t, json.Unmarshal(data, &guard))

	assert.Equal(t, "braintrust", guard.Skill)
	assert.Contains(t, guard.TeamDir, ".braintrust")
	assert.ElementsMatch(t, []string{
		"Task", "Agent", "Bash", "Read", "Glob", "Grep", "ToolSearch",
		"AskUserQuestion", "Skill",
		"mcp__gofortress-interactive__spawn_agent",
		"mcp__gofortress-interactive__team_run",
		"mcp__gofortress-interactive__get_agent_result",
		"mcp__gofortress-interactive__ask_user",
	}, guard.RouterAllowedTools)

	// Verify team dir exists
	_, err = os.Stat(guard.TeamDir)
	assert.NoError(t, err, "team dir should exist")
}

func TestSetupMode_ReviewCreatesTeamDir(t *testing.T) {
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	config := loadSkillGuardConfigFrom("review", testFixturePath(t))
	require.NotNil(t, config)

	handleSetupModeWithConfig("review", config, sessionDir, guardPath)

	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard ActiveSkill
	require.NoError(t, json.Unmarshal(data, &guard))

	assert.Equal(t, "review", guard.Skill)
	assert.Contains(t, guard.TeamDir, ".code-review")
	assert.Contains(t, guard.RouterAllowedTools, "Glob", "review skill should include Glob")

	_, err = os.Stat(guard.TeamDir)
	assert.NoError(t, err)
}

func TestSetupMode_ImplementCreatesTeamDir(t *testing.T) {
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	config := loadSkillGuardConfigFrom("implement", testFixturePath(t))
	require.NotNil(t, config)

	handleSetupModeWithConfig("implement", config, sessionDir, guardPath)

	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard ActiveSkill
	require.NoError(t, json.Unmarshal(data, &guard))

	assert.Equal(t, "implement", guard.Skill)
	assert.Contains(t, guard.TeamDir, ".implementation")

	_, err = os.Stat(guard.TeamDir)
	assert.NoError(t, err)
}

func TestSetupMode_NonTeamSkillNoGuard(t *testing.T) {
	// Test that a skill not in skill_guards returns nil config
	config := loadSkillGuardConfigFrom("dummies-guide", testFixturePath(t))
	assert.Nil(t, config, "non-team skill should not have a guard config")

	// Test that handleSetupModeWithConfig with nil config creates nothing
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	handleSetupModeWithConfig("dummies-guide", nil, sessionDir, guardPath)

	// No guard file should exist
	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "no guard file for non-team skill")

	// No teams directory should exist
	_, err = os.Stat(filepath.Join(sessionDir, "teams"))
	assert.True(t, os.IsNotExist(err), "no teams dir for non-team skill")
}

func TestSetupMode_TeamDirHasTimestamp(t *testing.T) {
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	config := loadSkillGuardConfigFrom("braintrust", testFixturePath(t))
	require.NotNil(t, config)

	handleSetupModeWithConfig("braintrust", config, sessionDir, guardPath)

	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard ActiveSkill
	require.NoError(t, json.Unmarshal(data, &guard))

	// Team dir name should match {unix_timestamp}.{suffix} pattern
	dirName := filepath.Base(guard.TeamDir)
	matched, err := regexp.MatchString(`^\d+\.braintrust$`, dirName)
	require.NoError(t, err)
	assert.True(t, matched, "team dir %q should match timestamp.suffix pattern", dirName)
}

func TestSetupMode_GuardFileContents(t *testing.T) {
	sessionDir := t.TempDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	config := loadSkillGuardConfigFrom("review", testFixturePath(t))
	require.NotNil(t, config)

	beforeSetup := time.Now().UTC()
	handleSetupModeWithConfig("review", config, sessionDir, guardPath)

	// Read and verify full round-trip
	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard ActiveSkill
	require.NoError(t, json.Unmarshal(data, &guard))

	assert.Equal(t, "review", guard.Skill)
	assert.NotEmpty(t, guard.TeamDir)
	assert.Equal(t, config.RouterAllowedTools, guard.RouterAllowedTools)

	// Verify CreatedAt is valid RFC3339 and recent
	createdAt, err := time.Parse(time.RFC3339, guard.CreatedAt)
	require.NoError(t, err, "CreatedAt should be valid RFC3339")
	assert.WithinDuration(t, beforeSetup, createdAt, 5*time.Second, "CreatedAt should be recent")
}

// TestFixtureMatchesProduction verifies that the testdata agents-index.json
// fixture contains the same skill_guards keys as the production file.
// This prevents silent fixture drift when skills are added/removed.
func TestFixtureMatchesProduction(t *testing.T) {
	// Load production agents-index.json
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		t.Skip("Cannot resolve config dir:", err)
	}
	prodPath := filepath.Join(configDir, "agents", "agents-index.json")
	prodData, err := os.ReadFile(prodPath)
	if err != nil {
		t.Skip("Production agents-index.json not found:", err)
	}

	// Load test fixture
	fixturePath := filepath.Join("testdata", "agents-index.json")
	fixtureData, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "test fixture should exist")

	// Parse both
	type indexFile struct {
		SkillGuards map[string]json.RawMessage `json:"skill_guards"`
	}

	var prod, fixture indexFile
	require.NoError(t, json.Unmarshal(prodData, &prod))
	require.NoError(t, json.Unmarshal(fixtureData, &fixture))

	// Verify fixture has all production keys
	for key := range prod.SkillGuards {
		assert.Contains(t, fixture.SkillGuards, key,
			"fixture missing production skill_guard %q — update testdata/agents-index.json", key)
	}

	// Verify fixture doesn't have extra keys
	for key := range fixture.SkillGuards {
		assert.Contains(t, prod.SkillGuards, key,
			"fixture has extra skill_guard %q not in production — update testdata/agents-index.json", key)
	}

	// Verify each skill's fields match
	for key := range prod.SkillGuards {
		if _, ok := fixture.SkillGuards[key]; !ok {
			continue // Already flagged above
		}

		type guardConfig struct {
			RouterAllowedTools []string `json:"router_allowed_tools"`
			TeamDirSuffix      string   `json:"team_dir_suffix"`
		}

		var prodGuard, fixtureGuard guardConfig
		require.NoError(t, json.Unmarshal(prod.SkillGuards[key], &prodGuard))
		require.NoError(t, json.Unmarshal(fixture.SkillGuards[key], &fixtureGuard))

		assert.Equal(t, prodGuard.TeamDirSuffix, fixtureGuard.TeamDirSuffix,
			"team_dir_suffix mismatch for %q", key)
		assert.ElementsMatch(t, prodGuard.RouterAllowedTools, fixtureGuard.RouterAllowedTools,
			"router_allowed_tools mismatch for %q", key)
	}
}
