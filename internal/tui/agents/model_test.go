package agents

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create bool pointer
func ptrBool(b bool) *bool {
	return &b
}

// Helper to create int64 pointer
func ptrInt64(i int64) *int64 {
	return &i
}

// TestAgentNode_IsActive verifies active status detection
func TestAgentNode_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   AgentStatus
		expected bool
	}{
		{"spawning is active", StatusSpawning, true},
		{"running is active", StatusRunning, true},
		{"completed is not active", StatusCompleted, false},
		{"error is not active", StatusError, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := &AgentNode{Status: tc.status}
			assert.Equal(t, tc.expected, node.IsActive())
		})
	}
}

// TestAgentNode_GetDuration verifies duration calculations
func TestAgentNode_GetDuration(t *testing.T) {
	now := time.Now()
	spawnTime := now.Add(-5 * time.Second)

	t.Run("returns stored duration if set", func(t *testing.T) {
		duration := 10 * time.Second
		node := &AgentNode{
			SpawnTime: spawnTime,
			Duration:  &duration,
		}
		assert.Equal(t, 10*time.Second, node.GetDuration())
	})

	t.Run("calculates from spawn to complete time", func(t *testing.T) {
		completeTime := now
		node := &AgentNode{
			SpawnTime:    spawnTime,
			CompleteTime: &completeTime,
		}
		duration := node.GetDuration()
		assert.InDelta(t, 5*time.Second, duration, float64(100*time.Millisecond))
	})

	t.Run("calculates elapsed time if not completed", func(t *testing.T) {
		node := &AgentNode{
			SpawnTime: spawnTime,
		}
		duration := node.GetDuration()
		// Should be approximately 5 seconds (with small tolerance)
		assert.Greater(t, duration, 4*time.Second)
		assert.Less(t, duration, 6*time.Second)
	})
}

// TestAgentTree_ProcessSpawn verifies spawn event handling
func TestAgentTree_ProcessSpawn(t *testing.T) {
	tree := NewAgentTree("session-123")

	t.Run("creates root node", func(t *testing.T) {
		event := &telemetry.AgentLifecycleEvent{
			EventID:         "event-1",
			SessionID:       "session-123",
			EventType:       "spawn",
			AgentID:         "agent-parent",
			ParentAgent:     "",
			Tier:            "sonnet",
			TaskDescription: "Main task",
			Timestamp:       time.Now().Unix(),
		}

		err := tree.ProcessSpawn(event)
		require.NoError(t, err)

		assert.NotNil(t, tree.Root)
		assert.Equal(t, "agent-parent", tree.Root.AgentID)
		assert.Equal(t, StatusSpawning, tree.Root.Status)
		assert.Equal(t, 1, tree.TotalAgents)
		assert.Equal(t, 1, tree.ActiveAgents)
	})

	t.Run("creates child node and links to parent", func(t *testing.T) {
		childEvent := &telemetry.AgentLifecycleEvent{
			EventID:         "event-2",
			SessionID:       "session-123",
			EventType:       "spawn",
			AgentID:         "agent-child",
			ParentAgent:     "agent-parent",
			Tier:            "haiku",
			TaskDescription: "Child task",
			Timestamp:       time.Now().Unix(),
		}

		err := tree.ProcessSpawn(childEvent)
		require.NoError(t, err)

		assert.Equal(t, 2, tree.TotalAgents)
		assert.Equal(t, 2, tree.ActiveAgents)

		// Parent should have child
		assert.Len(t, tree.Root.Children, 1)
		assert.Equal(t, "agent-child", tree.Root.Children[0].AgentID)

		// Parent should transition to running
		assert.Equal(t, StatusRunning, tree.Root.Status)
	})

	t.Run("idempotent - duplicate spawn ignored", func(t *testing.T) {
		duplicateEvent := &telemetry.AgentLifecycleEvent{
			EventID:     "event-1-dup",
			SessionID:   "session-123",
			EventType:   "spawn",
			AgentID:     "agent-parent", // Same ID
			ParentAgent: "",
			Tier:        "sonnet",
			Timestamp:   time.Now().Unix(),
		}

		err := tree.ProcessSpawn(duplicateEvent)
		require.NoError(t, err)

		// Stats should not change
		assert.Equal(t, 2, tree.TotalAgents)
	})
}

// TestAgentTree_ProcessComplete verifies complete event handling
func TestAgentTree_ProcessComplete(t *testing.T) {
	tree := NewAgentTree("session-123")

	// Setup: spawn an agent
	spawnTime := time.Now()
	spawnEvent := &telemetry.AgentLifecycleEvent{
		EventID:     "event-1",
		SessionID:   "session-123",
		EventType:   "spawn",
		AgentID:     "agent-1",
		ParentAgent: "",
		Tier:        "haiku",
		Timestamp:   spawnTime.Unix(),
	}
	tree.ProcessSpawn(spawnEvent)

	t.Run("marks agent as completed on success", func(t *testing.T) {
		completeTime := spawnTime.Add(2 * time.Second)
		completeEvent := &telemetry.AgentLifecycleEvent{
			EventID:    "event-2",
			SessionID:  "session-123",
			EventType:  "complete",
			AgentID:    "agent-1",
			Success:    ptrBool(true),
			DurationMs: ptrInt64(2000),
			Timestamp:  completeTime.Unix(),
		}

		err := tree.ProcessComplete(completeEvent)
		require.NoError(t, err)

		node, exists := tree.GetNode("agent-1")
		require.True(t, exists)

		assert.Equal(t, StatusCompleted, node.Status)
		assert.NotNil(t, node.CompleteTime)
		assert.NotNil(t, node.Duration)
		assert.Equal(t, 2*time.Second, *node.Duration)

		assert.Equal(t, 0, tree.ActiveAgents)
		assert.Equal(t, 1, tree.CompletedAgents)
		assert.Equal(t, 0, tree.ErroredAgents)
	})
}

// TestAgentTree_ProcessCompleteError verifies error handling
func TestAgentTree_ProcessCompleteError(t *testing.T) {
	tree := NewAgentTree("session-123")

	// Setup: spawn an agent
	spawnEvent := &telemetry.AgentLifecycleEvent{
		EventID:     "event-1",
		SessionID:   "session-123",
		EventType:   "spawn",
		AgentID:     "agent-error",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	}
	tree.ProcessSpawn(spawnEvent)

	t.Run("marks agent as error on failure", func(t *testing.T) {
		errMsg := "Task failed"
		completeEvent := &telemetry.AgentLifecycleEvent{
			EventID:      "event-2",
			SessionID:    "session-123",
			EventType:    "complete",
			AgentID:      "agent-error",
			Success:      ptrBool(false),
			ErrorMessage: &errMsg,
			Timestamp:    time.Now().Unix(),
		}

		err := tree.ProcessComplete(completeEvent)
		require.NoError(t, err)

		node, exists := tree.GetNode("agent-error")
		require.True(t, exists)

		assert.Equal(t, StatusError, node.Status)
		assert.Equal(t, 0, tree.ActiveAgents)
		assert.Equal(t, 0, tree.CompletedAgents)
		assert.Equal(t, 1, tree.ErroredAgents)
	})

	t.Run("returns error for unknown agent", func(t *testing.T) {
		completeEvent := &telemetry.AgentLifecycleEvent{
			EventID:   "event-3",
			SessionID: "session-123",
			EventType: "complete",
			AgentID:   "unknown-agent",
			Success:   ptrBool(true),
			Timestamp: time.Now().Unix(),
		}

		err := tree.ProcessComplete(completeEvent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown agent")
	})
}

// TestAgentTree_MultiLevel verifies multi-level tree construction
func TestAgentTree_MultiLevel(t *testing.T) {
	tree := NewAgentTree("session-multi")

	// Level 0: root
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e1",
		SessionID:   "session-multi",
		EventType:   "spawn",
		AgentID:     "orchestrator",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	// Level 1: orchestrator spawns two agents
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e2",
		SessionID:   "session-multi",
		EventType:   "spawn",
		AgentID:     "python-pro",
		ParentAgent: "orchestrator",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e3",
		SessionID:   "session-multi",
		EventType:   "spawn",
		AgentID:     "go-pro",
		ParentAgent: "orchestrator",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	// Level 2: python-pro spawns scout
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e4",
		SessionID:   "session-multi",
		EventType:   "spawn",
		AgentID:     "haiku-scout",
		ParentAgent: "python-pro",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	// Verify tree structure
	assert.Equal(t, 4, tree.TotalAgents)

	// Root has 2 children
	assert.Len(t, tree.Root.Children, 2)

	// Python-pro has 1 child
	pythonPro, exists := tree.GetNode("python-pro")
	require.True(t, exists)
	assert.Len(t, pythonPro.Children, 1)
	assert.Equal(t, "haiku-scout", pythonPro.Children[0].AgentID)

	// Go-pro has no children
	goPro, exists := tree.GetNode("go-pro")
	require.True(t, exists)
	assert.Len(t, goPro.Children, 0)
}

// TestAgentTree_OrphanedNodes verifies handling of out-of-order events
func TestAgentTree_OrphanedNodes(t *testing.T) {
	tree := NewAgentTree("session-orphan")

	// Spawn child before parent
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e1",
		SessionID:   "session-orphan",
		EventType:   "spawn",
		AgentID:     "child",
		ParentAgent: "parent",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	// Child should be orphaned
	stats := tree.GetStats()
	assert.Equal(t, 1, stats.OrphanedNodes)

	// Now spawn parent
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e2",
		SessionID:   "session-orphan",
		EventType:   "spawn",
		AgentID:     "parent",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	// Orphan should be attached
	stats = tree.GetStats()
	assert.Equal(t, 0, stats.OrphanedNodes)

	parent, exists := tree.GetNode("parent")
	require.True(t, exists)
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, "child", parent.Children[0].AgentID)
}

// TestAgentTree_GetActiveAgents verifies active agent filtering
func TestAgentTree_GetActiveAgents(t *testing.T) {
	tree := NewAgentTree("session-active")

	// Spawn 3 agents
	for i := 0; i < 3; i++ {
		tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
			EventID:     fmt.Sprintf("e%d", i),
			SessionID:   "session-active",
			EventType:   "spawn",
			AgentID:     fmt.Sprintf("agent-%d", i),
			ParentAgent: "",
			Tier:        "haiku",
			Timestamp:   time.Now().Unix(),
		})
	}

	// All should be active
	active := tree.GetActiveAgents()
	assert.Len(t, active, 3)

	// Complete one agent
	tree.ProcessComplete(&telemetry.AgentLifecycleEvent{
		EventID:   "ec1",
		SessionID: "session-active",
		EventType: "complete",
		AgentID:   "agent-0",
		Success:   ptrBool(true),
		Timestamp: time.Now().Unix(),
	})

	// Now only 2 should be active
	active = tree.GetActiveAgents()
	assert.Len(t, active, 2)
}

// TestAgentTree_WalkTree verifies depth-first traversal
func TestAgentTree_WalkTree(t *testing.T) {
	tree := NewAgentTree("session-walk")

	// Build tree:
	//   root
	//   ├── child1
	//   └── child2
	//       └── grandchild
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e1",
		SessionID:   "session-walk",
		EventType:   "spawn",
		AgentID:     "root",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e2",
		SessionID:   "session-walk",
		EventType:   "spawn",
		AgentID:     "child1",
		ParentAgent: "root",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e3",
		SessionID:   "session-walk",
		EventType:   "spawn",
		AgentID:     "child2",
		ParentAgent: "root",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e4",
		SessionID:   "session-walk",
		EventType:   "spawn",
		AgentID:     "grandchild",
		ParentAgent: "child2",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	t.Run("visits all nodes in depth-first order", func(t *testing.T) {
		var visited []string
		tree.WalkTree(func(node *AgentNode) bool {
			visited = append(visited, node.AgentID)
			return true // Continue
		})

		// Should visit: root, child1, child2, grandchild
		assert.Equal(t, []string{"root", "child1", "child2", "grandchild"}, visited)
	})

	t.Run("stops traversal when function returns false", func(t *testing.T) {
		var visited []string
		tree.WalkTree(func(node *AgentNode) bool {
			visited = append(visited, node.AgentID)
			return node.AgentID != "child1" // Stop after child1
		})

		// Should only visit root and child1
		assert.Equal(t, []string{"root", "child1"}, visited)
	})

	t.Run("handles empty tree", func(t *testing.T) {
		emptyTree := NewAgentTree("empty")
		var visited []string
		emptyTree.WalkTree(func(node *AgentNode) bool {
			visited = append(visited, node.AgentID)
			return true
		})
		assert.Empty(t, visited)
	})
}

// TestAgentTree_ToJSON verifies JSON serialization
func TestAgentTree_ToJSON(t *testing.T) {
	tree := NewAgentTree("session-json")

	// Build simple tree
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:         "e1",
		SessionID:       "session-json",
		EventType:       "spawn",
		AgentID:         "parent",
		ParentAgent:     "",
		Tier:            "sonnet",
		TaskDescription: "Main task",
		Timestamp:       time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:         "e2",
		SessionID:       "session-json",
		EventType:       "spawn",
		AgentID:         "child",
		ParentAgent:     "parent",
		Tier:            "haiku",
		TaskDescription: "Sub task",
		Timestamp:       time.Now().Unix(),
	})

	// Complete child
	tree.ProcessComplete(&telemetry.AgentLifecycleEvent{
		EventID:    "e3",
		SessionID:  "session-json",
		EventType:  "complete",
		AgentID:    "child",
		Success:    ptrBool(true),
		DurationMs: ptrInt64(1000),
		Timestamp:  time.Now().Unix(),
	})

	t.Run("serializes to valid JSON", func(t *testing.T) {
		data, err := tree.ToJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "session-json", result["session_id"])

		// Check stats
		stats := result["stats"].(map[string]interface{})
		assert.Equal(t, float64(2), stats["total"])
		assert.Equal(t, float64(1), stats["active"])
		assert.Equal(t, float64(1), stats["completed"])

		// Check tree structure
		treeData := result["tree"].(map[string]interface{})
		assert.Equal(t, "parent", treeData["agent_id"])

		children := treeData["children"].([]interface{})
		assert.Len(t, children, 1)

		child := children[0].(map[string]interface{})
		assert.Equal(t, "child", child["agent_id"])
		assert.Equal(t, "completed", child["status"])
		assert.NotNil(t, child["duration_ms"])
	})

	t.Run("handles empty tree", func(t *testing.T) {
		emptyTree := NewAgentTree("empty")
		data, err := emptyTree.ToJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Nil(t, result["tree"])
	})
}

// TestAgentTree_ConcurrentAccess verifies thread safety
func TestAgentTree_ConcurrentAccess(t *testing.T) {
	tree := NewAgentTree("session-concurrent")

	// Spawn root
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e0",
		SessionID:   "session-concurrent",
		EventType:   "spawn",
		AgentID:     "root",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	// Concurrently spawn and read
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
				EventID:     fmt.Sprintf("e%d", i+1),
				SessionID:   "session-concurrent",
				EventType:   "spawn",
				AgentID:     fmt.Sprintf("agent-%d", i),
				ParentAgent: "root",
				Tier:        "haiku",
				Timestamp:   time.Now().Unix(),
			})
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = tree.GetActiveAgents()
			_ = tree.GetStats()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// Verify final state
	stats := tree.GetStats()
	assert.Equal(t, 101, stats.TotalAgents) // root + 100 children
}

// TestAgentTree_GetChildren verifies child retrieval
func TestAgentTree_GetChildren(t *testing.T) {
	tree := NewAgentTree("session-children")

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e1",
		SessionID:   "session-children",
		EventType:   "spawn",
		AgentID:     "parent",
		ParentAgent: "",
		Tier:        "sonnet",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e2",
		SessionID:   "session-children",
		EventType:   "spawn",
		AgentID:     "child1",
		ParentAgent: "parent",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:     "e3",
		SessionID:   "session-children",
		EventType:   "spawn",
		AgentID:     "child2",
		ParentAgent: "parent",
		Tier:        "haiku",
		Timestamp:   time.Now().Unix(),
	})

	t.Run("returns all children", func(t *testing.T) {
		children := tree.GetChildren("parent")
		assert.Len(t, children, 2)

		ids := []string{children[0].AgentID, children[1].AgentID}
		assert.Contains(t, ids, "child1")
		assert.Contains(t, ids, "child2")
	})

	t.Run("returns nil for non-existent parent", func(t *testing.T) {
		children := tree.GetChildren("unknown")
		assert.Nil(t, children)
	})

	t.Run("returns empty for leaf node", func(t *testing.T) {
		children := tree.GetChildren("child1")
		assert.Empty(t, children)
	})
}
