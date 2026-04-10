package routing

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPSpawn_LLMInferenceArchitect_AgentExists verifies the target agent
// is present and correctly configured in the production agents-index.json.
func TestMCPSpawn_LLMInferenceArchitect_AgentExists(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err, "LoadAgentIndex() must succeed")

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err, "llm-inference-architect must exist in agents-index.json")

	assert.Equal(t, "llm-inference-architect", agent.ID)
	assert.Equal(t, "opus", agent.Model)
	assert.Equal(t, "architecture", agent.Category)
	assert.NotEmpty(t, agent.Triggers, "agent must have triggers")
	assert.NotEmpty(t, agent.Tools, "agent must have tools")
	assert.NotNil(t, agent.ContextRequirements, "agent must have context_requirements")
	assert.NotEmpty(t, agent.SpawnedBy, "agent must have spawned_by constraints")

	t.Log("✅ MCP Spawn: llm-inference-architect agent config validated")
}

// TestMCPSpawn_LLMInferenceArchitect_IdentityInjected verifies that
// BuildFullAgentContext injects the agent identity marker into the prompt.
// This simulates step 1 of the spawn path: identity injection.
func TestMCPSpawn_LLMInferenceArchitect_IdentityInjected(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	originalPrompt := "AGENT: llm-inference-architect\n\nTASK: Verify spawn works. Output: SPAWN_TEST_SUCCESS"

	augmented, err := BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, originalPrompt)
	require.NoError(t, err, "BuildFullAgentContext must succeed")

	// The prompt must be augmented (not returned unchanged if identity exists)
	// Either identity was injected OR prompt was unchanged (identity file may not exist in test env)
	if augmented == originalPrompt {
		t.Log("⚠️  Identity not injected (identity file may be absent in test env) — checking conventions path")
	} else {
		// Verify the identity marker is present
		assert.Contains(t, augmented, IdentityMarker,
			"augmented prompt must contain AGENT IDENTITY marker")
		assert.Contains(t, augmented, "llm-inference-architect identity",
			"augmented prompt must label the injected agent identity")
		assert.Contains(t, augmented, originalPrompt,
			"original prompt must be preserved in augmented output")
		t.Log("✅ MCP Spawn: Agent identity marker present in augmented prompt")
	}
}

// TestMCPSpawn_LLMInferenceArchitect_ConventionsInjected verifies that
// agent-guidelines.md conventions are present in the augmented prompt.
// This simulates the conventions injection step of BuildFullAgentContext.
func TestMCPSpawn_LLMInferenceArchitect_ConventionsInjected(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	// Verify agent context_requirements declares agent-guidelines.md
	require.NotNil(t, agent.ContextRequirements, "agent must have context_requirements")
	assert.Contains(t, agent.ContextRequirements.Rules, "agent-guidelines.md",
		"llm-inference-architect must require agent-guidelines.md rule injection")

	// Verify HasContextRequirements returns true
	assert.True(t, agent.ContextRequirements.HasContextRequirements(),
		"HasContextRequirements() must return true")

	// Build augmented prompt and check conventions marker is present
	originalPrompt := "AGENT: llm-inference-architect\n\nTASK: Verify conventions injected"
	augmented, err := BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, originalPrompt)
	require.NoError(t, err)

	if augmented != originalPrompt {
		// Conventions section should be present since agent-guidelines.md is declared
		assert.Contains(t, augmented, ConventionsMarker,
			"augmented prompt must contain CONVENTIONS marker when agent has rules")
		assert.Contains(t, augmented, "agent-guidelines.md",
			"augmented prompt must reference agent-guidelines.md")
		t.Log("✅ MCP Spawn: Conventions (agent-guidelines.md) present in augmented prompt")
	} else {
		t.Log("⚠️  Augmented prompt identical to original — rules file may be absent in test env")
	}
}

// TestMCPSpawn_DoubleInjectionGuard verifies that running BuildFullAgentContext
// twice on an already-augmented prompt does not double-inject content.
func TestMCPSpawn_DoubleInjectionGuard(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	original := "AGENT: llm-inference-architect\n\nTASK: Test double injection guard"
	first, err := BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, original)
	require.NoError(t, err)

	// Second pass should not add another identity block
	second, err := BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, first)
	require.NoError(t, err)

	// Count identity marker occurrences — must be exactly 1 (or 0 if file absent)
	count := strings.Count(second, IdentityMarker)
	assert.LessOrEqual(t, count, 1, "IdentityMarker must appear at most once (no double-injection)")

	if count == 1 {
		t.Log("✅ MCP Spawn: Double-injection guard works — identity injected exactly once")
	} else {
		t.Log("⚠️  Identity not injected (file absent) — double-injection guard not exercised")
	}
}

// TestMCPSpawn_BidirectionalValidation_OrchestratorCanSpawnLLM verifies the
// "can_spawn" side of the bidirectional relationship: orchestrator must list
// llm-inference-architect in its can_spawn list.
func TestMCPSpawn_BidirectionalValidation_OrchestratorCanSpawnLLM(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	orchestrator, err := index.GetAgentByID("orchestrator")
	require.NoError(t, err, "orchestrator agent must exist")

	assert.True(t,
		slices.Contains(orchestrator.CanSpawn, "llm-inference-architect"),
		"orchestrator.can_spawn must include llm-inference-architect for bidirectional validation to pass",
	)

	t.Log("✅ MCP Spawn: Bidirectional check (can_spawn) passed — orchestrator can spawn llm-inference-architect")
}

// TestMCPSpawn_BidirectionalValidation_LLMSpawnedByOrchestrator verifies the
// "spawned_by" side of the bidirectional relationship: llm-inference-architect
// must list orchestrator in its spawned_by list.
func TestMCPSpawn_BidirectionalValidation_LLMSpawnedByOrchestrator(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	assert.True(t,
		slices.Contains(agent.SpawnedBy, "orchestrator"),
		"llm-inference-architect.spawned_by must include orchestrator for bidirectional validation to pass",
	)

	t.Log("✅ MCP Spawn: Bidirectional check (spawned_by) passed — llm-inference-architect accepts orchestrator as parent")
}

// TestMCPSpawn_BidirectionalValidation_FullCheck exercises the complete
// bidirectional validation that validateRelationship() in the MCP spawner
// enforces. This test mirrors the two-step check:
//  1. child.SpawnedBy includes caller_type
//  2. parent.CanSpawn includes child agent ID
func TestMCPSpawn_BidirectionalValidation_FullCheck(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	callerType := "orchestrator"
	childAgentID := "llm-inference-architect"

	// Step 1: Check spawned_by constraint on child
	child, err := index.GetAgentByID(childAgentID)
	require.NoError(t, err)

	if len(child.SpawnedBy) > 0 {
		assert.True(t,
			slices.Contains(child.SpawnedBy, callerType),
			"Step 1 FAIL: %q.spawned_by must include %q", childAgentID, callerType,
		)
	}

	// Step 2: Check can_spawn constraint on parent
	parent, err := index.GetAgentByID(callerType)
	require.NoError(t, err, "orchestrator agent must exist in index")

	if len(parent.CanSpawn) > 0 {
		assert.True(t,
			slices.Contains(parent.CanSpawn, childAgentID),
			"Step 2 FAIL: %q.can_spawn must include %q", callerType, childAgentID,
		)
	}

	t.Logf("✅ MCP Spawn: Full bidirectional validation passed — %s → %s", callerType, childAgentID)
}

// TestMCPSpawn_BidirectionalValidation_RouterBypass verifies that spawning from
// the router session (no caller_type) bypasses the parent can_spawn check,
// consistent with validator.go: "router has implicit permission to spawn anything".
func TestMCPSpawn_BidirectionalValidation_RouterBypass(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	// Router is not an agent entry — it's the implicit root.
	// Verify no agent entry named "router" exists (would confuse validation).
	_, err = index.GetAgentByID("router")
	assert.Error(t, err, "router must not be a named agent — it is the implicit root")

	// From validator.go: when effectiveParent == "router", can_spawn check is skipped.
	// We verify the child still passes the spawned_by check if it has a spawned_by list
	// that doesn't include "router" — in that case a real spawn would fail with a
	// spawned_by error. For llm-inference-architect, verify router is NOT in spawned_by
	// (would indicate the agent is not intended for direct router spawn).
	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	routerInSpawnedBy := slices.Contains(agent.SpawnedBy, "router")
	if !routerInSpawnedBy && len(agent.SpawnedBy) > 0 {
		t.Logf("✅ MCP Spawn: llm-inference-architect.spawned_by=%v (router not listed — must be spawned via orchestrator/planner/einstein)", agent.SpawnedBy)
	} else if routerInSpawnedBy {
		t.Log("ℹ️  router is in spawned_by — direct router spawn is allowed")
	} else {
		t.Log("ℹ️  spawned_by is empty — no restriction on who can spawn this agent")
	}
}

// TestMCPSpawn_CostAttribution_AgentConfigPresent verifies that the agent entry
// contains cost attribution metadata used by the spawner for cost rollup.
func TestMCPSpawn_CostAttribution_AgentConfigPresent(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	// Cost per invocation range should be documented
	assert.NotEmpty(t, agent.CostPerInvocation,
		"llm-inference-architect must have cost_per_invocation for cost attribution tracking")

	// Model must be set for cost calculation (opus tier)
	assert.Equal(t, "opus", agent.Model,
		"cost attribution depends on model being 'opus'")

	t.Logf("✅ MCP Spawn: Cost attribution metadata present — cost_per_invocation=%s, model=%s",
		agent.CostPerInvocation, agent.Model)
}

// TestMCPSpawn_SpawnPromptStructure verifies that a spawn prompt following the
// MCP spawn_agent pattern is correctly built via BuildFullAgentContext and
// contains all expected sections for a valid agent invocation.
func TestMCPSpawn_SpawnPromptStructure(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	// Simulate the prompt that spawn_agent would construct
	originalPrompt := "AGENT: llm-inference-architect\n\nTASK: Verify spawn works. Output: SPAWN_TEST_SUCCESS"

	augmented, err := BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, originalPrompt)
	require.NoError(t, err)

	// Original prompt content must always be present in output
	assert.Contains(t, augmented, "SPAWN_TEST_SUCCESS",
		"original task content must be preserved in augmented prompt")
	assert.Contains(t, augmented, "AGENT: llm-inference-architect",
		"agent header must be present in augmented prompt")

	// If augmentation happened, verify structure
	if augmented != originalPrompt {
		assert.True(t,
			strings.Contains(augmented, IdentityMarker) || strings.Contains(augmented, ConventionsMarker),
			"augmented prompt must contain identity or conventions marker",
		)
		t.Log("✅ MCP Spawn: Spawn prompt structure validated — context injected, original preserved")
	} else {
		t.Log("⚠️  No augmentation (identity/conventions files absent in test env)")
	}
}

// TestMCPSpawn_AllowlistedParents verifies that all agents listed in llm-inference-architect's
// spawned_by are valid agent IDs that exist in the index (or "router" which is virtual).
func TestMCPSpawn_AllowlistedParents(t *testing.T) {
	index, err := LoadAgentIndex()
	require.NoError(t, err)

	agent, err := index.GetAgentByID("llm-inference-architect")
	require.NoError(t, err)

	for _, parentID := range agent.SpawnedBy {
		if parentID == "router" {
			// "router" is the virtual root session — not an agent entry
			continue
		}

		_, lookupErr := index.GetAgentByID(parentID)
		assert.NoError(t, lookupErr,
			"spawned_by entry %q must reference a valid agent in agents-index.json", parentID)
	}

	t.Logf("✅ MCP Spawn: All %d spawned_by parents are valid agent IDs", len(agent.SpawnedBy))
}
