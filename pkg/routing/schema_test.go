package routing

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalProductionSchema tests unmarshaling the actual routing-schema.json.
// This verifies that all v2.2.0 fields are correctly mapped to Go structs.
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
	assert.Equal(t, "2.2.0", schema.Version, "Schema version mismatch")

	// Verify all tiers exist
	assert.Contains(t, schema.Tiers, "haiku")
	assert.Contains(t, schema.Tiers, "haiku_thinking")
	assert.Contains(t, schema.Tiers, "sonnet")
	assert.Contains(t, schema.Tiers, "opus")
	assert.Contains(t, schema.Tiers, "external")

	// Verify tier_levels
	assert.Equal(t, 1, schema.TierLevels.Haiku)
	assert.Equal(t, 2, schema.TierLevels.HaikuThinking)
	assert.Equal(t, 3, schema.TierLevels.Sonnet)
	assert.Equal(t, 4, schema.TierLevels.Opus)
	assert.Equal(t, 0, schema.TierLevels.External)

	// Verify critical SubagentType fields (v2.2.0 additions)
	t.Run("SubagentTypeFields", func(t *testing.T) {
		explore := schema.SubagentTypesConfig.Explore
		assert.False(t, explore.AllowsWrite, "Explore should not allow write")
		assert.False(t, explore.RespectsAgentYaml, "Explore should not respect agent.yaml")
		assert.Contains(t, explore.UseFor, "codebase-search")

		generalPurpose := schema.SubagentTypesConfig.GeneralPurpose
		assert.True(t, generalPurpose.AllowsWrite, "general-purpose should allow write")
		assert.True(t, generalPurpose.RespectsAgentYaml, "general-purpose should respect agent.yaml")
		assert.Contains(t, generalPurpose.UseFor, "python-pro")

		bash := schema.SubagentTypesConfig.Bash
		assert.False(t, bash.AllowsWrite, "Bash should not allow write")
		assert.Contains(t, bash.UseFor, "gemini-slave")

		plan := schema.SubagentTypesConfig.Plan
		assert.True(t, plan.AllowsWrite, "Plan should allow write")
		assert.Contains(t, plan.UseFor, "orchestrator")
	})

	// Verify DelegationCeiling fields (v2.2.0 critical metadata)
	t.Run("DelegationCeilingFields", func(t *testing.T) {
		dc := schema.DelegationCeiling
		assert.NotEmpty(t, dc.Description)
		assert.Equal(t, ".claude/tmp/max_delegation", dc.File)
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

	// Verify agent_subagent_mapping
	t.Run("AgentSubagentMapping", func(t *testing.T) {
		assert.Equal(t, "Explore", schema.AgentSubagentMapping.CodebaseSearch)
		assert.Equal(t, "general-purpose", schema.AgentSubagentMapping.PythonPro)
		assert.Equal(t, "general-purpose", schema.AgentSubagentMapping.GoPro)
		assert.Equal(t, "Plan", schema.AgentSubagentMapping.Orchestrator)
		assert.Equal(t, "Bash", schema.AgentSubagentMapping.GeminiSlave)
	})

	// Verify escalation_rules structure
	t.Run("EscalationRules", func(t *testing.T) {
		assert.NotEmpty(t, schema.EscalationRules.HaikuToHaikuThinking)
		assert.NotEmpty(t, schema.EscalationRules.HaikuToSonnet)
		assert.NotEmpty(t, schema.EscalationRules.AnyToExternal)

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
		assert.Contains(t, opus.Invocation, "/einstein")
	})

	// Verify external tier protocols
	t.Run("ExternalTierProtocols", func(t *testing.T) {
		external := schema.Tiers["external"]
		require.NotNil(t, external.Protocols, "External tier missing protocols")

		scout := external.Protocols["scout"]
		assert.Equal(t, "gemini-2.0-flash", scout.Model)
		assert.Equal(t, "json", scout.Output)

		mapper := external.Protocols["mapper"]
		assert.Equal(t, "gemini-2.0-pro", mapper.Model)
		assert.Equal(t, "json", mapper.Output)
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
				Version: "2.2.0",
				Tiers: map[string]TierConfig{
					"haiku":  {},
					"sonnet": {},
				},
				TierLevels: TierLevels{
					Haiku:  1,
					Sonnet: 3,
				},
				SubagentTypesConfig: SubagentTypesConfig{
					Explore: SubagentType{},
				},
				AgentSubagentMapping: AgentSubagentMapping{
					CodebaseSearch: "Explore",
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
				Version: "2.2.0",
				Tiers: map[string]TierConfig{
					"invalid-tier": {},
				},
			},
			wantErr:   true,
			errSubstr: "Invalid tier name",
		},
		{
			name: "undefined subagent_type reference",
			schema: Schema{
				Version: "2.2.0",
				Tiers: map[string]TierConfig{
					"haiku": {},
				},
				TierLevels: TierLevels{
					Haiku: 1,
				},
				SubagentTypesConfig: SubagentTypesConfig{},
				AgentSubagentMapping: AgentSubagentMapping{
					CodebaseSearch: "NonexistentType",
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
			CodebaseSearch: "Explore",
			PythonPro:      "general-purpose",
		},
	}

	subagentType, err := schema.GetSubagentTypeForAgent("codebase-search")
	require.NoError(t, err)
	assert.Equal(t, "Explore", subagentType)

	subagentType, err = schema.GetSubagentTypeForAgent("python-pro")
	require.NoError(t, err)
	assert.Equal(t, "general-purpose", subagentType)

	_, err = schema.GetSubagentTypeForAgent("unknown-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown agent")
}

// TestValidateAgentSubagentPair tests agent-subagent_type pairing validation.
func TestValidateAgentSubagentPair(t *testing.T) {
	schema := Schema{
		SubagentTypesConfig: SubagentTypesConfig{
			Explore:        SubagentType{},
			GeneralPurpose: SubagentType{},
		},
		AgentSubagentMapping: AgentSubagentMapping{
			CodebaseSearch: "Explore",
			PythonPro:      "general-purpose",
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
			name:         "valid pairing - codebase-search + Explore",
			agentName:    "codebase-search",
			subagentType: "Explore",
			wantErr:      false,
		},
		{
			name:         "valid pairing - python-pro + general-purpose",
			agentName:    "python-pro",
			subagentType: "general-purpose",
			wantErr:      false,
		},
		{
			name:         "invalid pairing - wrong subagent_type",
			agentName:    "codebase-search",
			subagentType: "general-purpose",
			wantErr:      true,
			errSubstr:    "Invalid subagent_type",
		},
		{
			name:         "unknown agent",
			agentName:    "nonexistent-agent",
			subagentType: "Explore",
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
