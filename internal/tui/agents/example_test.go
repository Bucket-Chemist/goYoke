package agents_test

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
	"time"
)

// ExampleAgentTree demonstrates how to use the agent tree model
// with telemetry events for real-time tracking
func ExampleAgentTree() {
	// Create a new tree for the session
	tree := agents.NewAgentTree("session-example")

	// Simulate receiving spawn events
	now := time.Now()

	// Terminal spawns orchestrator
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:         "evt-001",
		SessionID:       "session-example",
		EventType:       "spawn",
		AgentID:         "orchestrator",
		ParentAgent:     "",
		Tier:            "sonnet",
		TaskDescription: "Coordinate implementation",
		Timestamp:       now.Unix(),
	})

	// Orchestrator spawns python-pro
	tree.ProcessSpawn(&telemetry.AgentLifecycleEvent{
		EventID:         "evt-002",
		SessionID:       "session-example",
		EventType:       "spawn",
		AgentID:         "python-pro",
		ParentAgent:     "orchestrator",
		Tier:            "sonnet",
		TaskDescription: "Implement Python module",
		Timestamp:       now.Add(1 * time.Second).Unix(),
	})

	// Check active agents
	active := tree.GetActiveAgents()
	fmt.Printf("Active agents: %d\n", len(active))

	// Python-pro completes
	success := true
	durationMs := int64(3000)
	tree.ProcessComplete(&telemetry.AgentLifecycleEvent{
		EventID:    "evt-003",
		SessionID:  "session-example",
		EventType:  "complete",
		AgentID:    "python-pro",
		Success:    &success,
		DurationMs: &durationMs,
		Timestamp:  now.Add(4 * time.Second).Unix(),
	})

	// Get statistics
	stats := tree.GetStats()
	fmt.Printf("Total: %d, Active: %d, Completed: %d\n",
		stats.TotalAgents, stats.ActiveAgents, stats.CompletedAgents)

	// Walk the tree
	fmt.Println("Tree structure:")
	tree.WalkTree(func(node *agents.AgentNode) bool {
		fmt.Printf("  %s (%s) - %s\n", node.AgentID, node.Tier, node.Status)
		return true
	})

	// Output:
	// Active agents: 2
	// Total: 2, Active: 1, Completed: 1
	// Tree structure:
	//   orchestrator (sonnet) - running
	//   python-pro (sonnet) - completed
}

// ExampleAgentTree_integration demonstrates integration with TelemetryWatcher
// This shows the typical usage pattern in the TUI
func ExampleAgentTree_integration() {
	// In real TUI code, you would:
	//
	// 1. Create tree
	tree := agents.NewAgentTree("session-123")

	// 2. Listen to telemetry events (pseudo-code):
	// for event := range watcher.Events().AgentLifecycle {
	//     switch event.EventType {
	//     case "spawn":
	//         tree.ProcessSpawn(event)
	//     case "complete":
	//         tree.ProcessComplete(event)
	//     }
	//
	//     // Update TUI display
	//     updateAgentTreeView(tree)
	// }

	// 3. In your TUI render loop, you can:
	//
	// - Get active agents for status display
	active := tree.GetActiveAgents()
	for _, agent := range active {
		fmt.Printf("[%s] %s - Running for %v\n",
			agent.Tier, agent.AgentID, agent.GetDuration())
	}

	// - Walk tree for hierarchical display
	tree.WalkTree(func(node *agents.AgentNode) bool {
		indent := "" // Calculate based on depth
		fmt.Printf("%s├─ %s (%s)\n", indent, node.AgentID, node.Status)
		return true
	})

	// - Export to JSON for debugging
	// jsonData, _ := tree.ToJSON()

	fmt.Println("See TUI-AGENT-02 for Bubble Tea integration")

	// Output:
	// See TUI-AGENT-02 for Bubble Tea integration
}
