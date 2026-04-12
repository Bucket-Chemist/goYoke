package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEcosystem_GOgent002 validates the complete schema loading pipeline.
// This retrospectively tests GOgent-002 (schema.go) from an ecosystem perspective.
func TestEcosystem_GOgent002(t *testing.T) {
	// Load production schema
	schema, err := LoadSchema()
	require.NoError(t, err, "LoadSchema() failed - GOgent-002 regression")

	// Validate schema passes validation
	err = schema.Validate()
	require.NoError(t, err, "Schema validation failed - GOgent-002 regression")

	// Verify critical v2.5.0 fields are present
	assert.Equal(t, "2.5.0", schema.Version)
	assert.NotEmpty(t, schema.Tiers, "Tiers map should be populated")
	assert.NotEmpty(t, schema.SubagentTypesConfig.Exploration.Tools, "SubagentTypes config incomplete")
	assert.NotEmpty(t, schema.AgentSubagentMapping.CodebaseSearch, "AgentSubagentMapping incomplete")

	t.Log("✅ Ecosystem Test GOgent-002: Schema loading pipeline operational")
}

// TestEcosystem_GOgent003 validates schema + agents integration.
// This retrospectively tests GOgent-003 (agents.go) integration with schema.
func TestEcosystem_GOgent003(t *testing.T) {
	// Load both production artifacts
	schema, err := LoadSchema()
	require.NoError(t, err, "Schema load failed")

	index, err := LoadAgentIndex()
	require.NoError(t, err, "AgentIndex load failed - GOgent-003 regression")

	// Cross-validate: Verify schema's agent mappings reference agents that exist in index
	agentIDs := []struct {
		name string
		id   string
	}{
		{"codebase-search", "codebase-search"},
		{"python-pro", "python-pro"},
		{"go-pro", "go-pro"},
		{"orchestrator", "orchestrator"},
	}

	for _, agent := range agentIDs {
		// Verify agent exists in index
		_, err := index.GetAgentByID(agent.id)
		require.NoError(t, err, "Agent %s not found in index", agent.id)

		// Verify schema has subagent_type mapping for this agent
		subagentType, err := schema.GetSubagentTypeForAgent(agent.id)
		require.NoError(t, err, "Schema missing mapping for %s", agent.id)
		assert.NotEmpty(t, subagentType, "Empty subagent_type for %s", agent.id)

		// Verify the pairing is valid
		err = schema.ValidateAgentSubagentPair(agent.id, subagentType)
		require.NoError(t, err, "Invalid agent-subagent pairing for %s", agent.id)
	}

	t.Log("✅ Ecosystem Test GOgent-003: Schema-Agent integration validated")
}

// TestEcosystem_GOgent004a is a placeholder for future validation integration.
// GOgent-004a ticket has not yet been implemented.
func TestEcosystem_GOgent004a(t *testing.T) {
	t.Skip("GOgent-004a not yet implemented - validation engine pending")
}

// TestEcosystem_AllAgentsMappedCorrectly verifies all agents in production agents-index.json
// have valid subagent_type mappings in routing-schema.json.
// This is the core invariant of the routing system.
func TestEcosystem_AllAgentsMappedCorrectly(t *testing.T) {
	schema, err := LoadSchema()
	require.NoError(t, err)

	index, err := LoadAgentIndex()
	require.NoError(t, err)

	// Expected v2.3.0 agents (from agents_test.go)
	expectedAgents := []string{
		"memory-archivist",
		"codebase-search",
		"librarian",
		"scaffolder",
		"tech-docs-writer",
		"code-reviewer",
		"python-pro",
		"python-ux",
		"r-pro",
		"r-shiny-pro",
		"go-pro",
		"go-cli",
		"go-tui",
		"go-api",
		"go-concurrent",
		"orchestrator",
		"architect",
		"einstein",
		"staff-architect-critical-review",
		"haiku-scout",
	}

	for _, agentID := range expectedAgents {
		t.Run(agentID, func(t *testing.T) {
			// Verify agent exists
			_, err := index.GetAgentByID(agentID)
			require.NoError(t, err, "Agent %s not found in agents-index.json", agentID)

			// Verify schema has mapping (now returns CC-specific type name)
			subagentType, err := schema.GetSubagentTypeForAgent(agentID)
			require.NoError(t, err, "Schema missing mapping for agent %s", agentID)
			assert.NotEmpty(t, subagentType, "Empty subagent_type for %s", agentID)

			// Verify pairing self-validates
			err = schema.ValidateAgentSubagentPair(agentID, subagentType)
			require.NoError(t, err, "Invalid pairing: agent=%s, subagent_type=%s", agentID, subagentType)
		})
	}

	t.Logf("✅ Ecosystem Test: All %d agents correctly mapped to subagent_types", len(expectedAgents))
}

// TestEcosystem_BackwardCompatibility validates that no breaking changes were introduced
// to the schema or agents APIs between GOgent-002 and GOgent-003.
func TestEcosystem_BackwardCompatibility(t *testing.T) {
	schema, err := LoadSchema()
	require.NoError(t, err)

	index, err := LoadAgentIndex()
	require.NoError(t, err)

	// Test cases that should remain stable across versions
	t.Run("SchemaAPIStability", func(t *testing.T) {
		// GetTier should work for all standard tiers
		for _, tier := range []string{"haiku", "haiku_thinking", "sonnet", "opus"} {
			_, err := schema.GetTier(tier)
			assert.NoError(t, err, "GetTier(%s) failed", tier)
		}

		// GetTierLevel should work for all standard tiers
		for _, tier := range []string{"haiku", "haiku_thinking", "sonnet", "opus"} {
			_, err := schema.GetTierLevel(tier)
			assert.NoError(t, err, "GetTierLevel(%s) failed", tier)
		}
	})

	t.Run("AgentIndexAPIStability", func(t *testing.T) {
		// GetAgentsByTier should work for tiers that have agents
		// Note: haiku_thinking is a valid tier in schema but may not have agents in model_tiers
		tiersWithAgents := []string{"haiku", "sonnet", "opus"}
		for _, tier := range tiersWithAgents {
			agents, err := index.GetAgentsByTier(tier)
			assert.NoError(t, err, "GetAgentsByTier(%s) failed", tier)
			assert.NotEmpty(t, agents, "Tier %s should have agents", tier)
		}

		// haiku_thinking is a valid schema tier but may not have agents mapped yet
		_, err = index.GetAgentsByTier("haiku_thinking")
		if err != nil {
			t.Logf("haiku_thinking tier has no agents (OK if tier not in model_tiers): %v", err)
		}

		// FindAgentByCategory should work
		categories := index.FindAgentByCategory("language")
		assert.NotEmpty(t, categories, "Should have language category agents")
	})

	t.Run("CrossReferenceStability", func(t *testing.T) {
		// All agents in model_tiers should exist
		for tierName, agentIDs := range index.RoutingRules.ModelTiers {
			for _, agentID := range agentIDs {
				_, err := index.GetAgentByID(agentID)
				assert.NoError(t, err, "Tier %s references missing agent: %s", tierName, agentID)
			}
		}
	})

	t.Log("✅ Ecosystem Test: Backward compatibility validated")
}
