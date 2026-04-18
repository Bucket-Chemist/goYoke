package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixtureMatchesProduction verifies that the testdata agents-index.json
// fixture contains the same skill_guards keys as the production file.
// This prevents silent fixture drift when skills are added/removed.
func TestFixtureMatchesProduction(t *testing.T) {
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		t.Skip("Cannot resolve config dir:", err)
	}
	prodPath := filepath.Join(configDir, "agents", "agents-index.json")
	prodData, err := os.ReadFile(prodPath)
	if err != nil {
		t.Skip("Production agents-index.json not found:", err)
	}

	fixturePath := filepath.Join("testdata", "agents-index.json")
	fixtureData, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "test fixture should exist")

	type indexFile struct {
		SkillGuards map[string]json.RawMessage `json:"skill_guards"`
	}

	var prod, fixture indexFile
	require.NoError(t, json.Unmarshal(prodData, &prod))
	require.NoError(t, json.Unmarshal(fixtureData, &fixture))

	for key := range prod.SkillGuards {
		assert.Contains(t, fixture.SkillGuards, key,
			"fixture missing production skill_guard %q — update testdata/agents-index.json", key)
	}

	for key := range fixture.SkillGuards {
		assert.Contains(t, prod.SkillGuards, key,
			"fixture has extra skill_guard %q not in production — update testdata/agents-index.json", key)
	}

	for key := range prod.SkillGuards {
		if _, ok := fixture.SkillGuards[key]; !ok {
			continue
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
