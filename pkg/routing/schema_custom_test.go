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
			"haiku":  {},
			"sonnet": {},
			"opus":   {},
		},
		TierLevels: TierLevels{
			Haiku:  1,
			Sonnet: 3,
			Opus:   4,
		},
		SubagentTypesConfig: SubagentTypesConfig{
			Analysis: SubagentType{}, // Informational category
		},
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review"),
		},
	}

	// 2. Test GetSubagentTypeForAgent
	t.Run("GetSubagentTypeForAgent", func(t *testing.T) {
		subagentType, err := schema.GetSubagentTypeForAgent("staff-architect-critical-review")
		require.NoError(t, err, "Should find staff-architect-critical-review agent")
		assert.Equal(t, "Staff Architect Critical Review", subagentType, "Should map to CC type name")
	})

	// 3. Test Validate (ensure the agent is included in the internal iteration list)
	t.Run("Validate", func(t *testing.T) {
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
		err := schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Staff Architect Critical Review")
		require.NoError(t, err, "Should accept valid pairing")

		// Invalid pair
		err = schema.ValidateAgentSubagentPair("staff-architect-critical-review", "Python Pro")
		require.Error(t, err, "Should reject invalid pairing")
		assert.Contains(t, err.Error(), "Invalid subagent_type", "Error should mention invalid subagent type")
	})
}
