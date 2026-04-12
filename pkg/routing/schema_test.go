package routing

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalProductionSchema tests unmarshaling the actual routing-schema.json.
// This verifies that all v2.5.0 fields are correctly mapped to Go structs.
func TestUnmarshalProductionSchema(t *testing.T) {
	// Load production schema from ~/.claude/routing-schema.json
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home := os.Getenv("HOME")
		require.NotEmpty(t, home, "HOME environment variable must be set")
		configHome = home + "/.config"
	}

	schemaPath := configHome + "/../.claude/routing-schema.json"
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read production routing-schema.json")

	var schema Schema
	err = json.Unmarshal(data, &schema)
	require.NoError(t, err, "Failed to unmarshal production schema")

	// Verify version
	assert.Equal(t, "2.5.0", schema.Version, "Schema version mismatch")

	// Verify all tiers exist
	assert.Contains(t, schema.Tiers, "haiku")
	assert.Contains(t, schema.Tiers, "haiku_thinking")
	assert.Contains(t, schema.Tiers, "sonnet")
	assert.Contains(t, schema.Tiers, "opus")
	// Verify tier_levels
	assert.Equal(t, 1, schema.TierLevels.Haiku)
	assert.Equal(t, 2, schema.TierLevels.HaikuThinking)
	assert.Equal(t, 3, schema.TierLevels.Sonnet)
	assert.Equal(t, 4, schema.TierLevels.Opus)
	// Verify informational SubagentType categories
	t.Run("SubagentTypeFields", func(t *testing.T) {
		exploration := schema.SubagentTypesConfig.Exploration
		assert.False(t, exploration.AllowsWrite, "exploration should not allow write")
		assert.Contains(t, exploration.Agents, "codebase-search")

		implementation := schema.SubagentTypesConfig.Implementation
		assert.True(t, implementation.AllowsWrite, "implementation should allow write")
		assert.Contains(t, implementation.Agents, "python-pro")

		planning := schema.SubagentTypesConfig.Planning
		assert.True(t, planning.AllowsWrite, "planning should allow write")
		assert.Contains(t, planning.Agents, "orchestrator")
	})

	// Verify DelegationCeiling fields (v2.3.0 critical metadata)
	t.Run("DelegationCeilingFields", func(t *testing.T) {
		dc := schema.DelegationCeiling
		assert.NotEmpty(t, dc.Description)
		assert.Contains(t, dc.File, "max_delegation")
		assert.Equal(t, "calculate-complexity.sh", dc.SetBy)
		assert.Equal(t, "validate-routing.sh", dc.EnforcedBy)
		assert.Contains(t, dc.Values, "haiku")
		assert.Contains(t, dc.Values, "haiku_thinking")
		assert.Contains(t, dc.Values, "sonnet")
		assert.NotEmpty(t, dc.Calculation)
		assert.Contains(t, dc.Calculation, "haiku")
	})

	// Verify BlockedPattern objects (not just strings)
	t.Run("BlockedPatterns", func(t *testing.T) {
		require.NotEmpty(t, schema.BlockedPatterns.Patterns, "No blocked patterns found")

		opusPattern := schema.BlockedPatterns.Patterns[0]
		assert.Equal(t, "Task.*model.*opus", opusPattern.Pattern)
		assert.NotEmpty(t, opusPattern.Reason)
		assert.NotEmpty(t, opusPattern.Alternative)
		assert.NotEmpty(t, opusPattern.CostImpact)
		assert.Contains(t, opusPattern.CostImpact, "$")
	})

	// Verify agent_subagent_mapping uses CC type names
	t.Run("AgentSubagentMapping", func(t *testing.T) {
		assert.True(t, schema.AgentSubagentMapping.CodebaseSearch.Contains("Codebase Search"))
		assert.True(t, schema.AgentSubagentMapping.PythonPro.Contains("Python Pro"))
		assert.True(t, schema.AgentSubagentMapping.GoPro.Contains("GO Pro"))
		assert.True(t, schema.AgentSubagentMapping.Orchestrator.Contains("Orchestrator"))
	})

	// Verify escalation_rules structure
	t.Run("EscalationRules", func(t *testing.T) {
		assert.NotEmpty(t, schema.EscalationRules.HaikuToHaikuThinking)
		assert.NotEmpty(t, schema.EscalationRules.HaikuToSonnet)
		opusRule := schema.EscalationRules.SonnetToOpus
		assert.NotEmpty(t, opusRule.Triggers)
		assert.Equal(t, "DO NOT use Task(opus). Generate GAP document instead.", opusRule.Action)
		assert.Equal(t, "escalate_to_einstein", opusRule.Protocol)
		assert.Contains(t, opusRule.OutputPath, "einstein-gap")
		assert.NotEmpty(t, opusRule.Notification)
	})

	// Verify opus tier configuration
	t.Run("OpusTierConfig", func(t *testing.T) {
		opus := schema.Tiers["opus"]
		assert.True(t, opus.TaskInvocationBlocked, "Opus should have task_invocation_blocked=true")
		assert.Equal(t, "escalate_to_einstein", opus.EscalationProtocol)
		assert.Contains(t, opus.Invocation, "/braintrust")
	})

	// Verify meta_rules
	t.Run("MetaRules", func(t *testing.T) {
		dt := schema.MetaRules.DocumentationTheater
		assert.NotEmpty(t, dt.Description)
		assert.Contains(t, dt.DetectionPatterns, "MUST NOT")
		assert.Contains(t, dt.DetectionPatterns, "NEVER use")
		assert.NotEmpty(t, dt.TargetFiles)
		assert.NotEmpty(t, dt.Enforcement)
	})
}

// TestSchemaValidate tests semantic validation logic.
func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name      string
		schema    Schema
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid schema",
			schema: Schema{
				Version: "2.5.0",
				Tiers: map[string]TierConfig{
					"haiku":  {},
					"sonnet": {},
				},
				TierLevels: TierLevels{
					Haiku:  1,
					Sonnet: 3,
				},
				SubagentTypesConfig: SubagentTypesConfig{
					Exploration: SubagentType{},
				},
				AgentSubagentMapping: AgentSubagentMapping{
					CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
				},
			},
			wantErr: false,
		},
		{
			name: "version mismatch",
			schema: Schema{
				Version: "1.0.0",
			},
			wantErr:   true,
			errSubstr: "version mismatch",
		},
		{
			name: "invalid tier name",
			schema: Schema{
				Version: "2.5.0",
				Tiers: map[string]TierConfig{
					"invalid-tier": {},
				},
			},
			wantErr:   true,
			errSubstr: "Invalid tier name",
		},
		{
			name: "external tier rejected",
			schema: Schema{
				Version: "2.5.0",
				Tiers: map[string]TierConfig{
					"haiku":    {},
					"external": {},
				},
			},
			wantErr:   true,
			errSubstr: "Invalid tier name",
		},
		{
			name: "undefined subagent_type reference",
			schema: Schema{
				Version: "2.5.0",
				Tiers: map[string]TierConfig{
					"haiku": {},
				},
				TierLevels: TierLevels{
					Haiku: 1,
				},
				SubagentTypesConfig: SubagentTypesConfig{},
				AgentSubagentMapping: AgentSubagentMapping{
					CodebaseSearch: NewFlexibleSubagentType("NonexistentType"),
				},
			},
			wantErr:   true,
			errSubstr: "Invalid subagent_type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.schema.Validate()
			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetTier tests tier retrieval.
func TestGetTier(t *testing.T) {
	schema := Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Model:   "haiku",
				Thinking: false,
			},
		},
	}

	tier, err := schema.GetTier("haiku")
	require.NoError(t, err)
	assert.Equal(t, "haiku", tier.Model)
	assert.False(t, tier.Thinking)

	_, err = schema.GetTier("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown tier")
}

// TestGetSubagentTypeForAgent tests agent-to-subagent_type lookup.
func TestGetSubagentTypeForAgent(t *testing.T) {
	schema := Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
			PythonPro:      NewFlexibleSubagentType("Python Pro"),
		},
	}

	subagentType, err := schema.GetSubagentTypeForAgent("codebase-search")
	require.NoError(t, err)
	assert.Equal(t, "Codebase Search", subagentType)

	subagentType, err = schema.GetSubagentTypeForAgent("python-pro")
	require.NoError(t, err)
	assert.Equal(t, "Python Pro", subagentType)

	_, err = schema.GetSubagentTypeForAgent("unknown-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown agent")
}

// TestValidateAgentSubagentPair tests agent-subagent_type pairing validation.
func TestValidateAgentSubagentPair(t *testing.T) {
	schema := Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
			PythonPro:      NewFlexibleSubagentType("Python Pro"),
		},
	}

	tests := []struct {
		name          string
		agentName     string
		subagentType  string
		wantErr       bool
		errSubstr     string
	}{
		{
			name:         "valid pairing - codebase-search + Codebase Search",
			agentName:    "codebase-search",
			subagentType: "Codebase Search",
			wantErr:      false,
		},
		{
			name:         "valid pairing - python-pro + Python Pro",
			agentName:    "python-pro",
			subagentType: "Python Pro",
			wantErr:      false,
		},
		{
			name:         "invalid pairing - wrong subagent_type",
			agentName:    "codebase-search",
			subagentType: "Python Pro",
			wantErr:      true,
			errSubstr:    "Invalid subagent_type",
		},
		{
			name:         "unknown agent",
			agentName:    "nonexistent-agent",
			subagentType: "Codebase Search",
			wantErr:      true,
			errSubstr:    "Unknown agent",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := schema.ValidateAgentSubagentPair(tc.agentName, tc.subagentType)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetTierLevel tests tier level retrieval.
func TestGetTierLevel(t *testing.T) {
	schema := Schema{
		TierLevels: TierLevels{
			Haiku:  1,
			Sonnet: 3,
			Opus:   4,
		},
	}

	level, err := schema.GetTierLevel("haiku")
	require.NoError(t, err)
	assert.Equal(t, 1, level)

	level, err = schema.GetTierLevel("opus")
	require.NoError(t, err)
	assert.Equal(t, 4, level)

	_, err = schema.GetTierLevel("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No tier level defined")
}

// TestLoadSchema tests schema loading from filesystem.
func TestLoadSchema(t *testing.T) {
	// This test verifies that LoadSchema() can read and validate production schema
	schema, err := LoadSchema()
	require.NoError(t, err, "LoadSchema should successfully load production schema")
	require.NotNil(t, schema)

	// Verify version matches expected
	assert.Equal(t, EXPECTED_SCHEMA_VERSION, schema.Version)

	// Verify Validate() passes
	err = schema.Validate()
	require.NoError(t, err, "Production schema should pass validation")
}

// TestSchema_FormatTierSummary tests the tier summary formatter with truncation.
func TestSchema_FormatTierSummary(t *testing.T) {
	schema := Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Patterns: []string{"find files", "search codebase", "grep pattern", "extra pattern 1", "extra pattern 2"},
				Tools:    []string{"Glob", "Grep", "Read", "WebFetch", "ExtraTool1", "ExtraTool2"},
			},
			"sonnet": {
				Patterns: []string{"implement", "refactor"},
				Tools:    []string{"Read", "Write", "Edit"},
			},
			"opus": {
				Patterns: []string{"deep analysis"},
				Tools:    []string{"Read"},
			},
		},
		DelegationCeiling: DelegationCeiling{
			SetBy: "calculate-complexity.sh",
		},
	}

	output := schema.FormatTierSummary()

	// Verify header
	assert.Contains(t, output, "ROUTING TIERS ACTIVE:")

	// Verify haiku tier with truncation (patterns: 5 → 3, tools: 6 → 4)
	assert.Contains(t, output, "haiku:")
	assert.Contains(t, output, "find files, search codebase, grep pattern...")
	assert.Contains(t, output, "Glob, Grep, Read, WebFetch...")

	// Verify sonnet tier without truncation (patterns: 2, tools: 3)
	assert.Contains(t, output, "sonnet:")
	assert.Contains(t, output, "implement, refactor")
	assert.Contains(t, output, "Read, Write, Edit")
	// Should NOT have ellipsis since under limit
	assert.NotContains(t, output, "implement, refactor...")
	assert.NotContains(t, output, "Read, Write, Edit...")

	// Verify opus tier (single items, no truncation)
	assert.Contains(t, output, "opus:")
	assert.Contains(t, output, "deep analysis")
	assert.Contains(t, output, "tools=[Read]")

	// Verify delegation ceiling
	assert.Contains(t, output, "DELEGATION CEILING: Set by calculate-complexity.sh")

	// Verify tier ordering (haiku before sonnet before opus)
	haikuIdx := strings.Index(output, "haiku:")
	sonnetIdx := strings.Index(output, "sonnet:")
	opusIdx := strings.Index(output, "opus:")
	assert.Less(t, haikuIdx, sonnetIdx, "haiku should appear before sonnet")
	assert.Less(t, sonnetIdx, opusIdx, "sonnet should appear before opus")
}

// TestSchema_FormatTierSummary_EmptyTiers tests formatter with empty/missing tiers.
func TestSchema_FormatTierSummary_EmptyTiers(t *testing.T) {
	schema := Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Patterns: []string{},
				Tools:    []string{},
			},
		},
		DelegationCeiling: DelegationCeiling{
			SetBy: "test-setter",
		},
	}

	output := schema.FormatTierSummary()

	// Should still include header
	assert.Contains(t, output, "ROUTING TIERS ACTIVE:")

	// Should include haiku tier even if empty
	assert.Contains(t, output, "haiku:")
	assert.Contains(t, output, "patterns=[]")
	assert.Contains(t, output, "tools=[]")

	// Should NOT include tiers that don't exist in map
	assert.NotContains(t, output, "sonnet:")
	assert.NotContains(t, output, "opus:")

	// Should include delegation ceiling
	assert.Contains(t, output, "DELEGATION CEILING: Set by test-setter")
}

// TestLoadAndFormatSchemaSummary tests the convenience function with production schema.
func TestLoadAndFormatSchemaSummary(t *testing.T) {
	summary, err := LoadAndFormatSchemaSummary()
	require.NoError(t, err, "LoadAndFormatSchemaSummary should succeed with production schema")
	require.NotEmpty(t, summary)

	// Verify expected content
	assert.Contains(t, summary, "ROUTING TIERS ACTIVE:")
	assert.Contains(t, summary, "DELEGATION CEILING:")

	// Should include at least haiku and sonnet tiers
	assert.Contains(t, summary, "haiku:")
	assert.Contains(t, summary, "sonnet:")
}

// TestLoadAndFormatSchemaSummary_MissingFile tests graceful handling of missing schema.
func TestLoadAndFormatSchemaSummary_MissingFile(t *testing.T) {
	// Temporarily set env var to a non-existent path
	originalEnv := os.Getenv("GOGENT_ROUTING_SCHEMA")
	os.Setenv("GOGENT_ROUTING_SCHEMA", "/nonexistent/path/to/schema.json")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GOGENT_ROUTING_SCHEMA", originalEnv)
		} else {
			os.Unsetenv("GOGENT_ROUTING_SCHEMA")
		}
	}()

	summary, err := LoadAndFormatSchemaSummary()

	// Should NOT return error for missing file (graceful fallback)
	require.NoError(t, err)
	assert.Contains(t, summary, "[No routing schema found - using defaults]")
}

// TestFlexibleSubagentType_UnmarshalJSON tests JSON unmarshaling for both formats.
func TestFlexibleSubagentType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTypes []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "single string (backwards compat)",
			input:     `"Explore"`,
			wantTypes: []string{"Explore"},
		},
		{
			name:      "array with one element",
			input:     `["general-purpose"]`,
			wantTypes: []string{"general-purpose"},
		},
		{
			name:      "array with multiple elements",
			input:     `["Plan", "Explore"]`,
			wantTypes: []string{"Plan", "Explore"},
		},
		{
			name:      "array with three elements",
			input:     `["Explore", "Plan", "general-purpose"]`,
			wantTypes: []string{"Explore", "Plan", "general-purpose"},
		},
		{
			name:      "empty array",
			input:     `[]`,
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name:      "null value",
			input:     `null`,
			wantErr:   true,
			errSubstr: "null/empty",
		},
		{
			name:      "number value",
			input:     `123`,
			wantErr:   true,
			errSubstr: "must be string or []string",
		},
		{
			name:      "object value",
			input:     `{"type": "Explore"}`,
			wantErr:   true,
			errSubstr: "must be string or []string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var f FlexibleSubagentType
			err := json.Unmarshal([]byte(tc.input), &f)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantTypes, f.types)
		})
	}
}

// TestFlexibleSubagentType_Contains tests the Contains method.
func TestFlexibleSubagentType_Contains(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		checkType string
		want      bool
	}{
		{
			name:      "single type - match",
			jsonInput: `"Explore"`,
			checkType: "Explore",
			want:      true,
		},
		{
			name:      "single type - no match",
			jsonInput: `"Explore"`,
			checkType: "Plan",
			want:      false,
		},
		{
			name:      "multiple types - match first",
			jsonInput: `["Plan", "Explore"]`,
			checkType: "Plan",
			want:      true,
		},
		{
			name:      "multiple types - match second",
			jsonInput: `["Plan", "Explore"]`,
			checkType: "Explore",
			want:      true,
		},
		{
			name:      "multiple types - no match",
			jsonInput: `["Plan", "Explore"]`,
			checkType: "general-purpose",
			want:      false,
		},
		{
			name:      "case sensitive - no match",
			jsonInput: `"Explore"`,
			checkType: "explore",
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var f FlexibleSubagentType
			err := json.Unmarshal([]byte(tc.jsonInput), &f)
			require.NoError(t, err)

			got := f.Contains(tc.checkType)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestFlexibleSubagentType_GetAll tests the GetAll method.
func TestFlexibleSubagentType_GetAll(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      []string
	}{
		{
			name:      "single type",
			jsonInput: `"Explore"`,
			want:      []string{"Explore"},
		},
		{
			name:      "two types",
			jsonInput: `["Plan", "Explore"]`,
			want:      []string{"Plan", "Explore"},
		},
		{
			name:      "three types",
			jsonInput: `["Explore", "Plan", "general-purpose"]`,
			want:      []string{"Explore", "Plan", "general-purpose"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var f FlexibleSubagentType
			err := json.Unmarshal([]byte(tc.jsonInput), &f)
			require.NoError(t, err)

			got := f.GetAll()
			assert.Equal(t, tc.want, got)

			// Verify returned slice is a copy, not the internal slice
			if len(got) > 0 {
				got[0] = "Modified"
				assert.NotEqual(t, "Modified", f.types[0], "GetAll should return a copy")
			}
		})
	}
}

// TestFlexibleSubagentType_Primary tests the Primary method.
func TestFlexibleSubagentType_Primary(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      string
	}{
		{
			name:      "single type",
			jsonInput: `"Explore"`,
			want:      "Explore",
		},
		{
			name:      "multiple types - returns first",
			jsonInput: `["Plan", "Explore", "general-purpose"]`,
			want:      "Plan",
		},
		{
			name:      "order matters",
			jsonInput: `["Explore", "Plan"]`,
			want:      "Explore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var f FlexibleSubagentType
			err := json.Unmarshal([]byte(tc.jsonInput), &f)
			require.NoError(t, err)

			got := f.Primary()
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestFlexibleSubagentType_Primary_Empty tests Primary on uninitialized type.
func TestFlexibleSubagentType_Primary_Empty(t *testing.T) {
	var f FlexibleSubagentType
	assert.Equal(t, "", f.Primary(), "Primary should return empty string for uninitialized type")
}

// TestFlexibleSubagentType_CompleteWorkflow tests a realistic workflow.
func TestFlexibleSubagentType_CompleteWorkflow(t *testing.T) {
	// Simulate routing-schema.json with CC type names
	schemaJSON := `{
		"codebase-search": "Codebase Search",
		"orchestrator": "Orchestrator",
		"staff-architect-critical-review": "Staff Architect Critical Review"
	}`

	var mapping map[string]FlexibleSubagentType
	err := json.Unmarshal([]byte(schemaJSON), &mapping)
	require.NoError(t, err)

	// Verify codebase-search
	codebaseSearch := mapping["codebase-search"]
	assert.True(t, codebaseSearch.Contains("Codebase Search"))
	assert.False(t, codebaseSearch.Contains("Orchestrator"))
	assert.Equal(t, "Codebase Search", codebaseSearch.Primary())
	assert.Equal(t, []string{"Codebase Search"}, codebaseSearch.GetAll())

	// Verify orchestrator
	orchestrator := mapping["orchestrator"]
	assert.True(t, orchestrator.Contains("Orchestrator"))
	assert.False(t, orchestrator.Contains("Codebase Search"))
	assert.Equal(t, "Orchestrator", orchestrator.Primary())

	// Verify staff-architect-critical-review
	staffArchitect := mapping["staff-architect-critical-review"]
	assert.True(t, staffArchitect.Contains("Staff Architect Critical Review"))
	assert.False(t, staffArchitect.Contains("Orchestrator"))
	assert.Equal(t, "Staff Architect Critical Review", staffArchitect.Primary())
	assert.Equal(t, []string{"Staff Architect Critical Review"}, staffArchitect.GetAll())
}

// TestMultiTypeValidation_EndToEnd tests the complete multi-type validation workflow.
func TestMultiTypeValidation_EndToEnd(t *testing.T) {
	// Create a schema — agents now use CC-specific type names
	schema := Schema{
		Version: "2.5.0",
		Tiers: map[string]TierConfig{
			"sonnet": {Model: "sonnet"},
		},
		TierLevels: TierLevels{
			Sonnet: 3,
		},
		SubagentTypesConfig: SubagentTypesConfig{
			Exploration: SubagentType{
				Description: "Exploration type",
				Tools:       []string{"Read", "Grep", "Glob"},
			},
			Planning: SubagentType{
				Description: "Planning type",
				Tools:       []string{"Read", "Write", "Edit"},
			},
		},
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review"),
		},
	}

	// Test 1: Validate returns the CC type name
	allowedTypes, err := schema.GetAllowedSubagentTypes("staff-architect-critical-review")
	require.NoError(t, err)
	assert.Equal(t, []string{"Staff Architect Critical Review"}, allowedTypes)

	// Test 2: Primary type
	primaryType, err := schema.GetSubagentTypeForAgent("staff-architect-critical-review")
	require.NoError(t, err)
	assert.Equal(t, "Staff Architect Critical Review", primaryType)

	// Test 3: CC type name validation passes
	err = schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Staff Architect Critical Review")
	assert.NoError(t, err)

	// Test 4: Wrong type fails
	err = schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Python Pro")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Staff Architect Critical Review")
	assert.Contains(t, err.Error(), "Python Pro")

	// Test 5: Schema validation passes
	err = schema.Validate()
	assert.NoError(t, err)
}
