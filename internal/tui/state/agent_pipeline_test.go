package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestRegistry_Register_IncreasesCount
// ---------------------------------------------------------------------------

// TestRegistry_Register_IncreasesCount verifies that each successful Register
// call increments Count().Total by one.
func TestRegistry_Register_IncreasesCount(t *testing.T) {
	r := NewAgentRegistry()

	assert.Equal(t, 0, r.Count().Total)

	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task-one", "")))
	assert.Equal(t, 1, r.Count().Total)

	require.NoError(t, r.Register(makeAgent("a2", "go-tui", "task-two", "")))
	assert.Equal(t, 2, r.Count().Total)

	require.NoError(t, r.Register(makeAgent("a3", "go-cli", "task-three", "")))
	assert.Equal(t, 3, r.Count().Total)
}

// ---------------------------------------------------------------------------
// TestRegistry_Register_AppearsInTree
// ---------------------------------------------------------------------------

// TestRegistry_Register_AppearsInTree verifies that after Register followed by
// InvalidateTreeCache, Tree() contains the registered agent as a node.
func TestRegistry_Register_AppearsInTree(t *testing.T) {
	r := NewAgentRegistry()

	require.NoError(t, r.Register(makeAgent("root-agent", "go-pro", "root task", "")))

	// Before cache invalidation Tree() must be nil (cache is stale).
	assert.Nil(t, r.Tree())

	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, "root-agent", nodes[0].Agent.ID)
	assert.Equal(t, "go-pro", nodes[0].Agent.AgentType)
}

// ---------------------------------------------------------------------------
// TestRegistry_UpdateStatus_ReflectsInTree
// ---------------------------------------------------------------------------

// TestRegistry_UpdateStatus_ReflectsInTree verifies that updating an agent's
// status via Update() and then calling InvalidateTreeCache causes Tree() to
// return the updated status.
func TestRegistry_UpdateStatus_ReflectsInTree(t *testing.T) {
	r := NewAgentRegistry()

	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	r.InvalidateTreeCache()

	// Confirm initial status in tree.
	nodes := r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, StatusPending, nodes[0].Agent.Status)

	// Transition to Running.
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusRunning }))

	// Tree cache is now stale; must invalidate before querying.
	r.InvalidateTreeCache()

	nodes = r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, StatusRunning, nodes[0].Agent.Status)

	// Transition to Complete.
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusComplete }))
	r.InvalidateTreeCache()

	nodes = r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, StatusComplete, nodes[0].Agent.Status)
}

// ---------------------------------------------------------------------------
// TestRegistry_ParentChild_TreeHierarchy
// ---------------------------------------------------------------------------

// TestRegistry_ParentChild_TreeHierarchy verifies that registering a child
// agent with a valid ParentID results in a two-node tree where the child is
// nested under the parent at depth 1.
func TestRegistry_ParentChild_TreeHierarchy(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()

	parent := makeAgent("parent", "go-pro", "parent task", "")
	parent.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(parent))

	child := makeAgent("child", "go-tui", "child task", "parent")
	child.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(child))

	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 2)

	// First node is the root (depth 0).
	assert.Equal(t, "parent", nodes[0].Agent.ID)
	assert.Equal(t, 0, nodes[0].Depth)

	// Second node is the child (depth 1).
	assert.Equal(t, "child", nodes[1].Agent.ID)
	assert.Equal(t, 1, nodes[1].Depth)
	assert.True(t, nodes[1].IsLast, "single child must be IsLast=true")
}

// TestRegistry_ParentChild_MultipleChildren verifies DFS ordering and IsLast
// semantics for a parent with two children.
func TestRegistry_ParentChild_MultipleChildren(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()

	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))

	c1 := makeAgent("c1", "go-tui", "first child", "root")
	c1.StartedAt = now.Add(-3 * time.Second)
	require.NoError(t, r.Register(c1))

	c2 := makeAgent("c2", "go-cli", "second child", "root")
	c2.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(c2))

	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 3)

	assert.Equal(t, "root", nodes[0].Agent.ID)
	assert.Equal(t, 0, nodes[0].Depth)

	assert.Equal(t, "c1", nodes[1].Agent.ID)
	assert.Equal(t, 1, nodes[1].Depth)
	assert.False(t, nodes[1].IsLast, "c1 is not the last child")

	assert.Equal(t, "c2", nodes[2].Agent.ID)
	assert.Equal(t, 1, nodes[2].Depth)
	assert.True(t, nodes[2].IsLast, "c2 is the last child")
}

// TestRegistry_ThreeLevelHierarchy verifies DFS traversal and depth values
// for a three-level tree: root → child → grandchild.
func TestRegistry_ThreeLevelHierarchy(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()

	root := makeAgent("root", "go-pro", "root task", "")
	root.StartedAt = now.Add(-3 * time.Second)
	require.NoError(t, r.Register(root))

	child := makeAgent("child", "go-tui", "child task", "root")
	child.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(child))

	grandchild := makeAgent("grandchild", "go-cli", "grandchild task", "child")
	grandchild.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(grandchild))

	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 3)

	// DFS order: root (0), child (1), grandchild (2).
	depths := []int{0, 1, 2}
	ids := []string{"root", "child", "grandchild"}
	for i, node := range nodes {
		assert.Equal(t, ids[i], node.Agent.ID, "node %d ID", i)
		assert.Equal(t, depths[i], node.Depth, "node %d depth", i)
	}
}

// ---------------------------------------------------------------------------
// TestPipeline_Register_TreeRefresh_DetailAutoSelect
// ---------------------------------------------------------------------------

// TestPipeline_Register_TreeRefresh_DetailAutoSelect exercises the full
// registry pipeline:
//  1. Register an agent
//  2. Invalidate the tree cache
//  3. Confirm the agent appears in the tree
//  4. Set the agent as selected
//  5. Confirm Selected() returns the agent ID
//
// This mirrors the sequence that the Bubbletea Update() handler follows when
// processing an AgentRegisteredMsg.
func TestPipeline_Register_TreeRefresh_DetailAutoSelect(t *testing.T) {
	r := NewAgentRegistry()

	agent := makeAgent("root-pipeline", "go-pro", "pipeline task", "")
	agent.Model = "sonnet"
	agent.Tier = "sonnet"
	require.NoError(t, r.Register(agent))

	// Step 2: Simulate what Update() does on AgentRegisteredMsg.
	r.InvalidateTreeCache()

	// Step 3: Tree must contain the agent.
	nodes := r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, "root-pipeline", nodes[0].Agent.ID)
	assert.Equal(t, "sonnet", nodes[0].Agent.Model)
	assert.Equal(t, "sonnet", nodes[0].Agent.Tier)

	// Step 4: Simulate auto-select of the root agent.
	r.SetSelected("root-pipeline")

	// Step 5: Selected() must return the agent ID.
	assert.Equal(t, "root-pipeline", r.Selected())

	// Confirm Get() returns full agent data for detail view.
	got := r.Get("root-pipeline")
	require.NotNil(t, got)
	assert.Equal(t, "sonnet", got.Model)
	assert.Equal(t, "sonnet", got.Tier)
	assert.Equal(t, StatusPending, got.Status)
}

// TestPipeline_MultipleAgents_SelectAndReselect tests selecting different
// agents in a tree and verifying Selected() tracks correctly.
func TestPipeline_MultipleAgents_SelectAndReselect(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()

	root := makeAgent("root", "go-pro", "root", "")
	root.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(root))

	child := makeAgent("child", "go-tui", "child", "root")
	child.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(child))

	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 2)

	// Select root.
	r.SetSelected(nodes[0].Agent.ID)
	assert.Equal(t, "root", r.Selected())

	// Move selection to child.
	r.SetSelected(nodes[1].Agent.ID)
	assert.Equal(t, "child", r.Selected())

	// Deselect.
	r.SetSelected("")
	assert.Equal(t, "", r.Selected())
}

// TestPipeline_StatusUpdate_TreeAndCount tests that a status update is
// consistently reflected in both Count() and Tree() after cache invalidation.
// a2 is registered as a child of a1 so both appear in the DFS tree walk.
func TestPipeline_StatusUpdate_TreeAndCount(t *testing.T) {
	r := NewAgentRegistry()

	// a1 is the root; a2 is its child — both appear in the DFS tree.
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task-one", "")))
	require.NoError(t, r.Register(makeAgent("a2", "go-tui", "task-two", "a1")))

	r.InvalidateTreeCache()

	// Both should be pending initially.
	stats := r.Count()
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Pending)
	assert.Equal(t, 0, stats.Running)

	// Both nodes must appear in the tree.
	nodes := r.Tree()
	require.Len(t, nodes, 2)

	// Transition a1 to running.
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusRunning }))
	r.InvalidateTreeCache()

	stats = r.Count()
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Running)

	// Find a1 in tree and confirm status.
	nodes = r.Tree()
	require.Len(t, nodes, 2)
	var foundRunning bool
	for _, n := range nodes {
		if n.Agent.ID == "a1" {
			assert.Equal(t, StatusRunning, n.Agent.Status)
			foundRunning = true
		}
	}
	assert.True(t, foundRunning, "a1 must appear in tree with StatusRunning")
}
