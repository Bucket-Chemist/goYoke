package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixtureConfig reads a fixture JSON, patches __PROJECT_ROOT__ with a real temp dir,
// creates required stdin files, and returns a TeamRunner ready for validation.
func loadFixtureConfig(t *testing.T, fixtureName string) (*TeamRunner, string) {
	t.Helper()

	// Read fixture
	data, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	require.NoError(t, err)

	// Create temp dirs for project_root and teamDir
	projectRoot := t.TempDir()
	teamDir := t.TempDir()

	// Patch __PROJECT_ROOT__ placeholder
	patched := strings.ReplaceAll(string(data), "__PROJECT_ROOT__", projectRoot)

	// Parse config
	var config TeamConfig
	require.NoError(t, json.Unmarshal([]byte(patched), &config))

	// Patch on_complete_script to "echo" for portability
	for i := range config.Waves {
		if config.Waves[i].OnCompleteScript != nil && *config.Waves[i].OnCompleteScript != "" {
			echo := "echo"
			config.Waves[i].OnCompleteScript = &echo
		}
	}

	// Create stdin files referenced by members
	for _, wave := range config.Waves {
		for _, member := range wave.Members {
			if member.StdinFile != "" {
				stdinPath := filepath.Join(teamDir, member.StdinFile)
				require.NoError(t, os.WriteFile(stdinPath, []byte("{}"), 0644))
			}
		}
	}

	// Write config to teamDir
	configData, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(teamDir, ConfigFileName), configData, 0644))

	// Create TeamRunner
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)

	return runner, teamDir
}

func TestFixture_ReviewConfigValid(t *testing.T) {
	runner, _ := loadFixtureConfig(t, "review_config.json")

	err := runner.ValidateConfig()
	assert.NoError(t, err)

	// Verify structure
	assert.Equal(t, "code-review", runner.config.TeamName)
	assert.Equal(t, "review", runner.config.WorkflowType)
	assert.Len(t, runner.config.Waves, 1, "should have 1 wave")
	assert.Len(t, runner.config.Waves[0].Members, 3, "wave 1 should have 3 members")
}

func TestFixture_ImplementConfigValid(t *testing.T) {
	runner, _ := loadFixtureConfig(t, "implement_config.json")

	err := runner.ValidateConfig()
	assert.NoError(t, err)

	// Verify structure
	assert.Equal(t, "implementation", runner.config.TeamName)
	assert.Equal(t, "implementation", runner.config.WorkflowType)
	assert.Len(t, runner.config.Waves, 2, "should have 2 waves")
	assert.Len(t, runner.config.Waves[0].Members, 1, "wave 1 should have 1 member")
	assert.Len(t, runner.config.Waves[1].Members, 1, "wave 2 should have 1 member")
}

func TestFixture_BraintrustConfigValid(t *testing.T) {
	runner, _ := loadFixtureConfig(t, "braintrust_config.json")

	err := runner.ValidateConfig()
	assert.NoError(t, err)

	// Verify structure
	assert.Equal(t, "braintrust", runner.config.TeamName)
	assert.Equal(t, "orchestration", runner.config.WorkflowType)
	assert.Len(t, runner.config.Waves, 2, "should have 2 waves")

	// Verify on_complete_script (patched to "echo" by loadFixtureConfig)
	assert.NotNil(t, runner.config.Waves[0].OnCompleteScript)
	assert.Equal(t, "echo", *runner.config.Waves[0].OnCompleteScript)
}

func TestFixture_AllConfigsHaveRequiredFields(t *testing.T) {
	fixtures := []string{
		"review_config.json",
		"implement_config.json",
		"braintrust_config.json",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			runner, _ := loadFixtureConfig(t, fixture)
			config := runner.config

			assert.NotEmpty(t, config.TeamName, "team_name should not be empty")
			assert.NotEmpty(t, config.WorkflowType, "workflow_type should not be empty")
			assert.Greater(t, config.BudgetMaxUSD, 0.0, "budget_max_usd should be > 0")
			assert.Equal(t, "pending", config.Status, "status should be pending")
			assert.NotEmpty(t, config.SessionID, "session_id should not be empty")
			assert.NotEmpty(t, config.CreatedAt, "created_at should not be empty")
		})
	}
}

func TestFixture_ReviewMemberModels(t *testing.T) {
	runner, _ := loadFixtureConfig(t, "review_config.json")

	// All review wave 1 members should have model "haiku"
	wave := runner.config.Waves[0]
	for _, member := range wave.Members {
		assert.Equal(t, "haiku", member.Model,
			"review member %s should use haiku model", member.Name)
	}
}

func TestFixture_BraintrustMemberModels(t *testing.T) {
	runner, _ := loadFixtureConfig(t, "braintrust_config.json")

	// All braintrust members (both waves) should have model "opus"
	for waveIdx, wave := range runner.config.Waves {
		for _, member := range wave.Members {
			assert.Equal(t, "opus", member.Model,
				"braintrust wave %d member %s should use opus model",
				waveIdx+1, member.Name)
		}
	}
}
