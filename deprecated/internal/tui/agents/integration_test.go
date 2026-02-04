package agents

import (
	"fmt"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RealWorldScenario simulates a realistic agent delegation tree
// This demonstrates the model working with telemetry events similar to GOgent-109
func TestIntegration_RealWorldScenario(t *testing.T) {
	tree := NewAgentTree("session-real-world")

	// Simulate a complex delegation scenario:
	// Terminal spawns orchestrator
	// Orchestrator spawns python-pro and go-pro
	// Python-pro spawns haiku-scout
	// All complete in reverse order

	now := time.Now()

	// 1. Terminal spawns orchestrator
	orchestratorSpawn := &telemetry.AgentLifecycleEvent{
		EventID:         "evt-001",
		SessionID:       "session-real-world",
		EventType:       "spawn",
		AgentID:         "orchestrator",
		ParentAgent:     "",
		Tier:            "sonnet",
		TaskDescription: "Coordinate multi-language refactoring",
		DecisionID:      "dec-001",
		Timestamp:       now.Unix(),
	}
	err := tree.ProcessSpawn(orchestratorSpawn)
	require.NoError(t, err)

	// 2. Orchestrator spawns python-pro (500ms later)
	pythonSpawn := &telemetry.AgentLifecycleEvent{
		EventID:         "evt-002",
		SessionID:       "session-real-world",
		EventType:       "spawn",
		AgentID:         "python-pro",
		ParentAgent:     "orchestrator",
		Tier:            "sonnet",
		TaskDescription: "Refactor Python modules",
		DecisionID:      "dec-002",
		Timestamp:       now.Add(500 * time.Millisecond).Unix(),
	}
	err = tree.ProcessSpawn(pythonSpawn)
	require.NoError(t, err)

	// Verify orchestrator transitioned to running
	orchestrator, exists := tree.GetNode("orchestrator")
	require.True(t, exists)
	assert.Equal(t, StatusRunning, orchestrator.Status)

	// 3. Orchestrator spawns go-pro (1s later)
	goSpawn := &telemetry.AgentLifecycleEvent{
		EventID:         "evt-003",
		SessionID:       "session-real-world",
		EventType:       "spawn",
		AgentID:         "go-pro",
		ParentAgent:     "orchestrator",
		Tier:            "sonnet",
		TaskDescription: "Refactor Go packages",
		DecisionID:      "dec-003",
		Timestamp:       now.Add(1 * time.Second).Unix(),
	}
	err = tree.ProcessSpawn(goSpawn)
	require.NoError(t, err)

	// 4. Python-pro spawns haiku-scout (1.5s later)
	scoutSpawn := &telemetry.AgentLifecycleEvent{
		EventID:         "evt-004",
		SessionID:       "session-real-world",
		EventType:       "spawn",
		AgentID:         "haiku-scout",
		ParentAgent:     "python-pro",
		Tier:            "haiku",
		TaskDescription: "Find all Python test files",
		DecisionID:      "dec-004",
		Timestamp:       now.Add(1500 * time.Millisecond).Unix(),
	}
	err = tree.ProcessSpawn(scoutSpawn)
	require.NoError(t, err)

	// Verify tree structure at this point
	stats := tree.GetStats()
	assert.Equal(t, 4, stats.TotalAgents)
	assert.Equal(t, 4, stats.ActiveAgents)
	assert.Equal(t, 0, stats.CompletedAgents)
	assert.Equal(t, 0, stats.OrphanedNodes)

	// Verify hierarchy
	assert.Len(t, orchestrator.Children, 2) // python-pro and go-pro

	pythonPro, _ := tree.GetNode("python-pro")
	assert.Len(t, pythonPro.Children, 1) // haiku-scout

	goPro, _ := tree.GetNode("go-pro")
	assert.Len(t, goPro.Children, 0) // no children

	// 5. Haiku-scout completes first (2s runtime)
	scoutComplete := &telemetry.AgentLifecycleEvent{
		EventID:    "evt-005",
		SessionID:  "session-real-world",
		EventType:  "complete",
		AgentID:    "haiku-scout",
		Success:    ptrBool(true),
		DurationMs: ptrInt64(2000),
		Timestamp:  now.Add(3500 * time.Millisecond).Unix(),
	}
	err = tree.ProcessComplete(scoutComplete)
	require.NoError(t, err)

	stats = tree.GetStats()
	assert.Equal(t, 3, stats.ActiveAgents)
	assert.Equal(t, 1, stats.CompletedAgents)

	// 6. Go-pro completes (5s runtime)
	goComplete := &telemetry.AgentLifecycleEvent{
		EventID:    "evt-006",
		SessionID:  "session-real-world",
		EventType:  "complete",
		AgentID:    "go-pro",
		Success:    ptrBool(true),
		DurationMs: ptrInt64(5000),
		Timestamp:  now.Add(6 * time.Second).Unix(),
	}
	err = tree.ProcessComplete(goComplete)
	require.NoError(t, err)

	// 7. Python-pro completes (6s runtime)
	pythonComplete := &telemetry.AgentLifecycleEvent{
		EventID:    "evt-007",
		SessionID:  "session-real-world",
		EventType:  "complete",
		AgentID:    "python-pro",
		Success:    ptrBool(true),
		DurationMs: ptrInt64(6000),
		Timestamp:  now.Add(6500 * time.Millisecond).Unix(),
	}
	err = tree.ProcessComplete(pythonComplete)
	require.NoError(t, err)

	// 8. Orchestrator completes (7s runtime)
	orchestratorComplete := &telemetry.AgentLifecycleEvent{
		EventID:    "evt-008",
		SessionID:  "session-real-world",
		EventType:  "complete",
		AgentID:    "orchestrator",
		Success:    ptrBool(true),
		DurationMs: ptrInt64(7000),
		Timestamp:  now.Add(7 * time.Second).Unix(),
	}
	err = tree.ProcessComplete(orchestratorComplete)
	require.NoError(t, err)

	// Final verification
	stats = tree.GetStats()
	assert.Equal(t, 4, stats.TotalAgents)
	assert.Equal(t, 0, stats.ActiveAgents)
	assert.Equal(t, 4, stats.CompletedAgents)
	assert.Equal(t, 0, stats.ErroredAgents)

	// Verify durations were captured
	scout, _ := tree.GetNode("haiku-scout")
	assert.Equal(t, 2*time.Second, *scout.Duration)

	orchestratorNode, _ := tree.GetNode("orchestrator")
	assert.Equal(t, 7*time.Second, *orchestratorNode.Duration)

	// Test tree traversal
	var visitOrder []string
	tree.WalkTree(func(node *AgentNode) bool {
		visitOrder = append(visitOrder, node.AgentID)
		return true
	})

	// Should visit in depth-first order
	expected := []string{"orchestrator", "python-pro", "haiku-scout", "go-pro"}
	assert.Equal(t, expected, visitOrder)

	// Test JSON serialization
	jsonData, err := tree.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify GetActiveAgents returns empty when all complete
	activeAgents := tree.GetActiveAgents()
	assert.Empty(t, activeAgents)
}

// TestIntegration_ErrorScenario simulates an agent that fails
func TestIntegration_ErrorScenario(t *testing.T) {
	tree := NewAgentTree("session-error")

	now := time.Now()

	// Spawn agent
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:         "evt-001",
		SessionID:       "session-error",
		EventType:       "spawn",
		AgentID:         "failing-agent",
		ParentAgent:     "",
		Tier:            "sonnet",
		TaskDescription: "This will fail",
		Timestamp:       now.Unix(),
	})

	// Agent fails
	errorMsg := "Module not found: missing_module"
	tree.ProcessComplete(&telemetry.AgentLifecycleEvent{
		EventID:      "evt-002",
		SessionID:    "session-error",
		EventType:    "complete",
		AgentID:      "failing-agent",
		Success:      ptrBool(false),
		ErrorMessage: &errorMsg,
		DurationMs:   ptrInt64(500),
		Timestamp:    now.Add(500 * time.Millisecond).Unix(),
	})

	stats := tree.GetStats()
	assert.Equal(t, 1, stats.TotalAgents)
	assert.Equal(t, 0, stats.ActiveAgents)
	assert.Equal(t, 0, stats.CompletedAgents)
	assert.Equal(t, 1, stats.ErroredAgents)

	node, exists := tree.GetNode("failing-agent")
	require.True(t, exists)
	assert.Equal(t, StatusError, node.Status)
	assert.NotNil(t, node.CompleteEvent)
	assert.Equal(t, &errorMsg, node.CompleteEvent.ErrorMessage)
}

// TestIntegration_OutOfOrderEvents simulates events arriving out of order
func TestIntegration_OutOfOrderEvents(t *testing.T) {
	tree := NewAgentTree("session-out-of-order")

	now := time.Now()

	// Child spawns before parent (network delay, etc.)
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "evt-002",
		SessionID:   "session-out-of-order",
		EventType:   "spawn",
		AgentID:     "child",
		ParentAgent: "parent",
		Tier:        "haiku",
		Timestamp:   now.Add(1 * time.Second).Unix(),
	})

	// Child should be orphaned
	stats := tree.GetStats()
	assert.Equal(t, 1, stats.OrphanedNodes)

	// Parent spawns later
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "evt-001",
		SessionID:   "session-out-of-order",
		EventType:   "spawn",
		AgentID:     "parent",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   now.Unix(),
	})

	// Child should now be attached
	stats = tree.GetStats()
	assert.Equal(t, 0, stats.OrphanedNodes)

	parent, exists := tree.GetNode("parent")
	require.True(t, exists)
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, "child", parent.Children[0].AgentID)
}

// TestIntegration_DeepNesting simulates a deeply nested agent tree
func TestIntegration_DeepNesting(t *testing.T) {
	tree := NewAgentTree("session-deep")

	now := time.Now()

	// Create a 10-level deep tree
	parentID := ""
	for i := 0; i < 10; i++ {
		agentID := fmt.Sprintf("agent-level-%d", i)
		tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
			EventID:     fmt.Sprintf("evt-%d", i),
			SessionID:   "session-deep",
			EventType:   "spawn",
			AgentID:     agentID,
			ParentAgent: parentID,
			Tier:        "haiku",
			Timestamp:   now.Add(time.Duration(i) * time.Second).Unix(),
		})
		parentID = agentID
	}

	stats := tree.GetStats()
	assert.Equal(t, 10, stats.TotalAgents)

	// Verify depth by walking tree
	depth := 0
	maxDepth := 0
	tree.WalkTree(func(node *AgentNode) bool {
		if len(node.Children) > 0 {
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		} else {
			// Leaf node
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return true
	})

	assert.Equal(t, 9, maxDepth) // 10 levels = depth 9 (0-indexed)
}
