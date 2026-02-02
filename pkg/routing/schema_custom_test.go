package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStaffArchitectCriticalReviewIntegration verifies the specific integration of the new agent.
func TestStaffArchitectCriticalReviewIntegration(t *testing.T) {
	// 1. Setup a schema that mimics the expected structure for the new agent
	schema := Schema{
		Version: "2.5.0",
		Tiers: map[string]TierConfig{
			"haiku":    {},
			"sonnet":   {},
			"opus":     {},
			"external": {},
		},
		TierLevels: TierLevels{
			Haiku:    1,
			Sonnet:   3,
			Opus:     4,
			External: 0,
		},
		SubagentTypesConfig: SubagentTypesConfig{
			Plan: SubagentType{}, // The expected type for this agent (changed from Explore in v2.3.0)
		},
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Plan"),
		},
	}

	// 2. Test GetSubagentTypeForAgent
	t.Run("GetSubagentTypeForAgent", func(t *testing.T) {
		subagentType, err := schema.GetSubagentTypeForAgent("staff-architect-critical-review")
		require.NoError(t, err, "Should find staff-architect-critical-review agent")
		assert.Equal(t, "Plan", subagentType, "Should map to Plan subagent type")
	})

	// 3. Test Validate (ensure the agent is included in the internal iteration list)
	t.Run("Validate", func(t *testing.T) {
		// We set up a schema where "Explore" is a valid subagent type
		// If the internal loop in Validate() skipped StaffArchitectCriticalReview, 
		// it wouldn't check if "Explore" was valid for it. 
		// But here we want to ensure Validate() passes when correctly configured.
		err := schema.Validate()
		require.NoError(t, err, "Validate() should pass for correctly configured new agent")

		// Now intentionally break it to ensure it IS being checked
		invalidSchema := schema
		invalidSchema.AgentSubagentMapping.StaffArchitectCriticalReview = NewFlexibleSubagentType("InvalidType")
		err = invalidSchema.Validate()
		require.Error(t, err, "Validate() should fail for invalid subagent type on new agent")
		assert.Contains(t, err.Error(), "Invalid subagent_type reference", "Error message should mention invalid type")
	})

	// 4. Test ValidateAgentSubagentPair
	t.Run("ValidateAgentSubagentPair", func(t *testing.T) {
		// Valid pair
		err := schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Plan")
		require.NoError(t, err, "Should accept valid pairing")

		// Invalid pair
		err = schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Bash")
		require.Error(t, err, "Should reject invalid pairing")
		assert.Contains(t, err.Error(), "Invalid subagent_type", "Error should mention invalid subagent type")
	})
}
