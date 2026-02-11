package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeWaves_HappyPath(t *testing.T) {
	tasks := []Task{
		{TaskID: "task-001", BlockedBy: []string{}},
		{TaskID: "task-002", BlockedBy: []string{}},
		{TaskID: "task-003", BlockedBy: []string{"task-001"}},
	}

	waves, err := computeWaves(tasks)
	require.NoError(t, err)
	require.Len(t, waves, 2)

	// Wave 0: task-001 and task-002 (sorted)
	assert.Len(t, waves[0], 2)
	assert.Equal(t, "task-001", waves[0][0].TaskID)
	assert.Equal(t, "task-002", waves[0][1].TaskID)

	// Wave 1: task-003
	assert.Len(t, waves[1], 1)
	assert.Equal(t, "task-003", waves[1][0].TaskID)
}

func TestComputeWaves_CircularDependency(t *testing.T) {
	tasks := []Task{
		{TaskID: "task-001", BlockedBy: []string{"task-002"}},
		{TaskID: "task-002", BlockedBy: []string{"task-001"}},
	}

	waves, err := computeWaves(tasks)
	assert.Nil(t, waves)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
	assert.True(t, isReferentialIntegrityError(err))
}

func TestComputeWaves_LinearChain(t *testing.T) {
	tasks := []Task{
		{TaskID: "task-001", BlockedBy: []string{}},
		{TaskID: "task-002", BlockedBy: []string{"task-001"}},
		{TaskID: "task-003", BlockedBy: []string{"task-002"}},
	}

	waves, err := computeWaves(tasks)
	require.NoError(t, err)
	require.Len(t, waves, 3)

	assert.Len(t, waves[0], 1)
	assert.Equal(t, "task-001", waves[0][0].TaskID)

	assert.Len(t, waves[1], 1)
	assert.Equal(t, "task-002", waves[1][0].TaskID)

	assert.Len(t, waves[2], 1)
	assert.Equal(t, "task-003", waves[2][0].TaskID)
}

func TestComputeWaves_SingleTask(t *testing.T) {
	tasks := []Task{
		{TaskID: "task-001", BlockedBy: []string{}},
	}

	waves, err := computeWaves(tasks)
	require.NoError(t, err)
	require.Len(t, waves, 1)
	require.Len(t, waves[0], 1)
	assert.Equal(t, "task-001", waves[0][0].TaskID)
}

func TestValidatePlan_ValidPlan(t *testing.T) {
	plan := &ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Implement feature",
				Description:        "Full description",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
		},
	}

	knownAgents := []string{"go-pro", "go-cli", "python-pro"}
	err := validatePlan(plan, knownAgents)
	assert.NoError(t, err)
}

func TestValidatePlan_UnknownAgent(t *testing.T) {
	plan := &ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Implement feature",
				Description:        "Full description",
				Agent:              "unknown-agent",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
		},
	}

	knownAgents := []string{"go-pro", "go-cli"}
	err := validatePlan(plan, knownAgents)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-agent")
	assert.True(t, isSchemaValidationError(err))
}

func TestValidatePlan_UnknownDependency(t *testing.T) {
	plan := &ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Implement feature",
				Description:        "Full description",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{"task-999"},
				AcceptanceCriteria: []string{"criterion 1"},
			},
		},
	}

	knownAgents := []string{"go-pro"}
	err := validatePlan(plan, knownAgents)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task-999")
	assert.Contains(t, err.Error(), "referential_integrity")
	assert.True(t, isReferentialIntegrityError(err))
}

func TestValidatePlan_DuplicateTaskID(t *testing.T) {
	plan := &ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Feature 1",
				Description:        "Description 1",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
			{
				TaskID:             "task-001",
				Subject:            "Feature 2",
				Description:        "Description 2",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/bar"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 2"},
			},
		},
	}

	knownAgents := []string{"go-pro"}
	err := validatePlan(plan, knownAgents)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate task_id")
	assert.True(t, isSchemaValidationError(err))
}

func TestValidatePlan_InvalidTaskIDFormat(t *testing.T) {
	plan := &ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "foo",
				Subject:            "Feature",
				Description:        "Description",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
		},
	}

	knownAgents := []string{"go-pro"}
	err := validatePlan(plan, knownAgents)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match pattern")
	assert.True(t, isSchemaValidationError(err))
}

func TestGenerateStdinFile_BlocksComputation(t *testing.T) {
	plan := ImplementationPlan{
		Version: "1.0.0",
		Project: Project{
			Language:        "go",
			ConventionsFile: "go.md",
		},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Feature 1",
				Description:        "Description 1",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
			{
				TaskID:             "task-002",
				Subject:            "Feature 2",
				Description:        "Description 2",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/bar"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 2"},
			},
			{
				TaskID:             "task-003",
				Subject:            "Feature 3",
				Description:        "Description 3",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/baz"},
				BlockedBy:          []string{"task-001", "task-002"},
				AcceptanceCriteria: []string{"criterion 3"},
			},
		},
	}

	waves, err := computeWaves(plan.Tasks)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = generateStdinFiles(plan, waves, "/project", tmpDir)
	require.NoError(t, err)

	// Check task-001 has blocks=["task-003"]
	data, err := os.ReadFile(filepath.Join(tmpDir, "stdin_task-001.json"))
	require.NoError(t, err)

	var stdin1 stdinSchema
	err = json.Unmarshal(data, &stdin1)
	require.NoError(t, err)
	assert.Equal(t, []string{"task-003"}, stdin1.Task.Blocks)

	// Check task-002 has blocks=["task-003"]
	data, err = os.ReadFile(filepath.Join(tmpDir, "stdin_task-002.json"))
	require.NoError(t, err)

	var stdin2 stdinSchema
	err = json.Unmarshal(data, &stdin2)
	require.NoError(t, err)
	assert.Equal(t, []string{"task-003"}, stdin2.Task.Blocks)

	// Check task-003 has blocks=[]
	data, err = os.ReadFile(filepath.Join(tmpDir, "stdin_task-003.json"))
	require.NoError(t, err)

	var stdin3 stdinSchema
	err = json.Unmarshal(data, &stdin3)
	require.NoError(t, err)
	assert.Equal(t, []string{}, stdin3.Task.Blocks)
}

func TestGenerateConfig_WaveStructure(t *testing.T) {
	tasks := []Task{
		{TaskID: "task-001", Agent: "go-pro", Subject: "Feature 1", BlockedBy: []string{}},
		{TaskID: "task-002", Agent: "go-cli", Subject: "Feature 2", BlockedBy: []string{}},
		{TaskID: "task-003", Agent: "go-tui", Subject: "Feature 3", BlockedBy: []string{"task-001"}},
	}

	waves, err := computeWaves(tasks)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err = generateConfig(waves, "/project", tmpDir, configPath)
	require.NoError(t, err)

	// Read and verify config
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config genTeamConfig
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify workflow type
	assert.Equal(t, "implementation", config.WorkflowType)

	// Verify wave structure
	require.Len(t, config.Waves, 2)

	// Wave 1 (1-indexed)
	assert.Equal(t, 1, config.Waves[0].WaveNumber)
	assert.Len(t, config.Waves[0].Members, 2)
	assert.Equal(t, "task-001", config.Waves[0].Members[0].Name)
	assert.Equal(t, "go-pro", config.Waves[0].Members[0].Agent)
	assert.Equal(t, "sonnet", config.Waves[0].Members[0].Model)
	assert.Equal(t, "task-002", config.Waves[0].Members[1].Name)
	assert.Equal(t, "go-cli", config.Waves[0].Members[1].Agent)

	// Wave 2
	assert.Equal(t, 2, config.Waves[1].WaveNumber)
	assert.Len(t, config.Waves[1].Members, 1)
	assert.Equal(t, "task-003", config.Waves[1].Members[0].Name)
	assert.Equal(t, "go-tui", config.Waves[1].Members[0].Agent)
}
