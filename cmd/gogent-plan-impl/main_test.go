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

// TestEnrichedPlanJSON_IgnoresEnrichmentFields verifies that gogent-plan-impl
// safely handles enriched plan JSON. Specifically:
//   - json.Unmarshal silently drops unknown fields (enrichment_version,
//     review_annotations, harmonization_log, readiness_score)
//   - version "1.0.0" check still passes (enrichment_version is a separate key)
//   - implicit_dependencies on tasks are ignored by Kahn's algorithm
//   - wave output is identical to the equivalent unenriched plan
func TestEnrichedPlanJSON_IgnoresEnrichmentFields(t *testing.T) {
	// task-003 has implicit_dependency on task-002 but NOT in blocked_by.
	// Wave computation must ignore implicit_dependencies, placing task-003
	// in wave 0 alongside task-001 (not forced to wait for task-002).
	enrichedJSON := `{
		"version": "1.0.0",
		"enrichment_version": "1.0.0",
		"project": {
			"language": "go",
			"conventions_file": "go.md"
		},
		"tasks": [
			{
				"task_id": "task-001",
				"subject": "Feature 1",
				"description": "Description 1",
				"agent": "go-pro",
				"target_packages": ["cmd/foo"],
				"blocked_by": [],
				"acceptance_criteria": ["criterion 1"],
				"implicit_dependencies": []
			},
			{
				"task_id": "task-002",
				"subject": "Feature 2",
				"description": "Description 2",
				"agent": "go-pro",
				"target_packages": ["cmd/bar"],
				"blocked_by": ["task-001"],
				"acceptance_criteria": ["criterion 2"],
				"implicit_dependencies": [
					{
						"depends_on": "task-001",
						"reason": "Shares config patterns",
						"confidence": 0.8,
						"promoted": false
					}
				]
			},
			{
				"task_id": "task-003",
				"subject": "Feature 3",
				"description": "Description 3",
				"agent": "go-pro",
				"target_packages": ["cmd/baz"],
				"blocked_by": [],
				"acceptance_criteria": ["criterion 3"],
				"implicit_dependencies": [
					{
						"depends_on": "task-002",
						"reason": "Uses same interface",
						"confidence": 0.6,
						"promoted": false
					}
				]
			}
		],
		"review_annotations": [
			{
				"finding_id": "F-001",
				"severity": "minor",
				"classification": "augmentation",
				"classification_confidence": 0.9,
				"mapped_tasks": ["task-001"],
				"mapping_method": "semantic",
				"description": "Consider adding more acceptance criteria",
				"recommendation": "Add edge case criteria",
				"auto_applied": false
			}
		],
		"harmonization_log": [
			{
				"change": "Reordered task-002 and task-003",
				"rationale": "Dependency order",
				"affected_tasks": ["task-002", "task-003"],
				"source_finding": "F-001"
			}
		],
		"readiness_score": {
			"total": 85,
			"dimensions": {
				"fix_coverage": 4,
				"dep_validity": 5,
				"schema_completeness": 5
			},
			"formula": "total = (fix_coverage * 7) + (dep_validity * 7) + (schema_completeness * 6)",
			"floor_rule": "if fix_coverage < 2, total is capped at min(computed_total, 49)",
			"thresholds": {"ready": 70, "caveats": 50, "not_ready": 0}
		}
	}`

	// Parse enriched plan — must not error
	var plan ImplementationPlan
	err := json.Unmarshal([]byte(enrichedJSON), &plan)
	require.NoError(t, err, "enriched plan JSON must parse without error")

	// version field must be "1.0.0" — enrichment_version is a separate unknown key
	assert.Equal(t, "1.0.0", plan.Version)

	// All three tasks must be parsed
	require.Len(t, plan.Tasks, 3)

	// Validate against known agents — version check must pass
	knownAgents := []string{"go-pro"}
	require.NoError(t, validatePlan(&plan, knownAgents))

	// Compute waves — implicit_dependencies must NOT affect wave computation.
	// Expected waves based solely on blocked_by:
	//   Wave 0: task-001, task-003 (task-003 has no blocked_by; implicit ignored)
	//   Wave 1: task-002 (blocked_by: ["task-001"])
	waves, err := computeWaves(plan.Tasks)
	require.NoError(t, err)
	require.Len(t, waves, 2)

	// Wave 0: task-001 and task-003 (sorted by task_id)
	assert.Len(t, waves[0], 2)
	assert.Equal(t, "task-001", waves[0][0].TaskID)
	assert.Equal(t, "task-003", waves[0][1].TaskID)

	// Wave 1: task-002 only (depends on task-001 via blocked_by)
	assert.Len(t, waves[1], 1)
	assert.Equal(t, "task-002", waves[1][0].TaskID)
}

// --- Enrichment: implicit dependency warnings ---

func TestWarnImplicitDeps_UnpromotedWarns(t *testing.T) {
	plan := &ImplementationPlan{
		Tasks: []Task{
			{
				TaskID: "task-001",
				ImplicitDependencies: []ImplicitDependency{
					{DependsOn: "task-000", Reason: "shares config", Confidence: 0.85, Promoted: false},
				},
			},
			{
				TaskID: "task-002",
				ImplicitDependencies: []ImplicitDependency{
					{DependsOn: "task-001", Reason: "uses same interface", Confidence: 0.60, Promoted: true},
				},
			},
		},
	}

	warnings := warnImplicitDeps(plan)

	// Only the unpromoted dep on task-001 should warn; the promoted one on task-002 is silent.
	require.Len(t, warnings, 3, "expect 3 lines for one unpromoted dep")
	assert.Contains(t, warnings[0], "task-001")
	assert.Contains(t, warnings[0], "task-000")
	assert.Contains(t, warnings[0], "0.85")
	assert.Contains(t, warnings[1], "shares config")
	assert.Contains(t, warnings[2], "/refine-plan --promote-dep task-001:task-000")
}

func TestWarnImplicitDeps_AllPromotedSilent(t *testing.T) {
	plan := &ImplementationPlan{
		Tasks: []Task{
			{
				TaskID: "task-001",
				ImplicitDependencies: []ImplicitDependency{
					{DependsOn: "task-000", Reason: "reason", Confidence: 0.9, Promoted: true},
				},
			},
		},
	}
	assert.Nil(t, warnImplicitDeps(plan))
}

func TestWarnImplicitDeps_NoImplicitDeps(t *testing.T) {
	plan := &ImplementationPlan{
		Tasks: []Task{
			{TaskID: "task-001"},
			{TaskID: "task-002"},
		},
	}
	assert.Nil(t, warnImplicitDeps(plan))
}

// --- Enrichment: readiness score ---

func TestFormatReadinessScore_Ready(t *testing.T) {
	plan := &ImplementationPlan{
		ReadinessScore: &ReadinessScore{
			Total:      78,
			Dimensions: map[string]int{"dep_validity": 5, "fix_coverage": 4, "schema_completeness": 4},
		},
	}
	out := formatReadinessScore(plan)
	assert.Contains(t, out, "78/100 (ready)")
	// Dimensions sorted alphabetically: dep_validity, fix_coverage, schema_completeness
	assert.Contains(t, out, "dep_validity: 5/5")
	assert.Contains(t, out, "fix_coverage: 4/5")
	assert.Contains(t, out, "schema_completeness: 4/5")
}

func TestFormatReadinessScore_Caveats(t *testing.T) {
	plan := &ImplementationPlan{
		ReadinessScore: &ReadinessScore{
			Total:      60,
			Dimensions: map[string]int{"fix_coverage": 3, "dep_validity": 4, "schema_completeness": 3},
		},
	}
	out := formatReadinessScore(plan)
	assert.Contains(t, out, "60/100 (caveats)")
}

func TestFormatReadinessScore_NotReady(t *testing.T) {
	plan := &ImplementationPlan{
		ReadinessScore: &ReadinessScore{
			Total:      40,
			Dimensions: map[string]int{"fix_coverage": 1, "dep_validity": 3, "schema_completeness": 2},
		},
	}
	out := formatReadinessScore(plan)
	assert.Contains(t, out, "40/100 (not ready)")
}

func TestFormatReadinessScore_NoEnrichment(t *testing.T) {
	plan := &ImplementationPlan{}
	assert.Equal(t, "", formatReadinessScore(plan))
}

func TestReadinessScoreWarning_LowScore(t *testing.T) {
	plan := &ImplementationPlan{
		ReadinessScore: &ReadinessScore{Total: 40, Dimensions: map[string]int{}},
	}
	w := readinessScoreWarning(plan)
	assert.Contains(t, w, "40/100 < 50")
}

func TestReadinessScoreWarning_HighScore(t *testing.T) {
	plan := &ImplementationPlan{
		ReadinessScore: &ReadinessScore{Total: 78, Dimensions: map[string]int{}},
	}
	assert.Equal(t, "", readinessScoreWarning(plan))
}

func TestReadinessScoreWarning_NoEnrichment(t *testing.T) {
	plan := &ImplementationPlan{}
	assert.Equal(t, "", readinessScoreWarning(plan))
}

// --- Enrichment: review annotation injection ---

func TestReviewAnnotations_InjectedIntoStdin(t *testing.T) {
	plan := ImplementationPlan{
		Version: "1.0.0",
		Project: Project{Language: "go", ConventionsFile: "go.md"},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Feature 1",
				Description:        "Desc 1",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
			{
				TaskID:             "task-002",
				Subject:            "Feature 2",
				Description:        "Desc 2",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/bar"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 2"},
			},
		},
		ReviewAnnotations: []ReviewAnnotation{
			{
				FindingID:      "C-1",
				Classification: "correction",
				Recommendation: "Use deep copy instead of shallow copy",
				AutoApplied:    false,
				MappedTasks:    []string{"task-001"},
			},
			{
				FindingID:      "A-1",
				Classification: "augmentation",
				Recommendation: "Add edge case criteria",
				AutoApplied:    true,
				MappedTasks:    []string{"task-001"},
			},
			{
				FindingID:      "W-1",
				Classification: "warning",
				Recommendation: "Consider thread safety",
				AutoApplied:    false,
				MappedTasks:    []string{"task-002"},
			},
		},
	}

	waves, err := computeWaves(plan.Tasks)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = generateStdinFiles(plan, waves, "/project", tmpDir)
	require.NoError(t, err)

	// task-001: correction C-1 → corrections_to_address; augmentation A-1 auto-applied → fixes_incorporated
	data, err := os.ReadFile(filepath.Join(tmpDir, "stdin_task-001.json"))
	require.NoError(t, err)

	var stdin1 stdinSchema
	require.NoError(t, json.Unmarshal(data, &stdin1))
	require.NotNil(t, stdin1.ReviewFindings)
	require.Len(t, stdin1.ReviewFindings.CorrectionsToAddress, 1)
	assert.Contains(t, stdin1.ReviewFindings.CorrectionsToAddress[0], "C-1")
	assert.Contains(t, stdin1.ReviewFindings.CorrectionsToAddress[0], "Use deep copy")
	require.Len(t, stdin1.ReviewFindings.FixesIncorporated, 1)
	assert.Contains(t, stdin1.ReviewFindings.FixesIncorporated[0], "A-1")
	assert.Contains(t, stdin1.ReviewFindings.FixesIncorporated[0], "auto-applied")
	assert.Nil(t, stdin1.ReviewFindings.ReviewNotes)

	// task-002: warning W-1 → review_notes
	data, err = os.ReadFile(filepath.Join(tmpDir, "stdin_task-002.json"))
	require.NoError(t, err)

	var stdin2 stdinSchema
	require.NoError(t, json.Unmarshal(data, &stdin2))
	require.NotNil(t, stdin2.ReviewFindings)
	require.Len(t, stdin2.ReviewFindings.ReviewNotes, 1)
	assert.Contains(t, stdin2.ReviewFindings.ReviewNotes[0], "W-1")
	assert.Contains(t, stdin2.ReviewFindings.ReviewNotes[0], "thread safety")
	assert.Nil(t, stdin2.ReviewFindings.CorrectionsToAddress)
	assert.Nil(t, stdin2.ReviewFindings.FixesIncorporated)
}

func TestUnenrichedPlan_NoReviewFindings(t *testing.T) {
	// Unenriched plan has no review_annotations → stdin review_findings must be absent (nil pointer → omitted from JSON)
	plan := ImplementationPlan{
		Version: "1.0.0",
		Project: Project{Language: "go", ConventionsFile: "go.md"},
		Tasks: []Task{
			{
				TaskID:             "task-001",
				Subject:            "Feature 1",
				Description:        "Desc 1",
				Agent:              "go-pro",
				TargetPackages:     []string{"cmd/foo"},
				BlockedBy:          []string{},
				AcceptanceCriteria: []string{"criterion 1"},
			},
		},
	}

	waves, err := computeWaves(plan.Tasks)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, generateStdinFiles(plan, waves, "/project", tmpDir))

	data, err := os.ReadFile(filepath.Join(tmpDir, "stdin_task-001.json"))
	require.NoError(t, err)

	var stdin stdinSchema
	require.NoError(t, json.Unmarshal(data, &stdin))
	assert.Nil(t, stdin.ReviewFindings, "unenriched plan must produce no review_findings in stdin")

	// Also verify the JSON itself doesn't contain the key
	assert.NotContains(t, string(data), "review_findings")
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
