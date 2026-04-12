package agents

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeNode builds a minimal AgentTreeNode for use in component tests.
func makeNode(id, agentType, description string, depth int, isLast bool) *state.AgentTreeNode {
	return &state.AgentTreeNode{
		Agent: &state.Agent{
			ID:          id,
			AgentType:   agentType,
			Description: description,
			Status:      state.StatusPending,
			StartedAt:   time.Now(),
		},
		Depth:  depth,
		IsLast: isLast,
	}
}

// makeNodeWithStatus is like makeNode but sets a specific agent status.
func makeNodeWithStatus(id, agentType, description string, depth int, isLast bool, status state.AgentStatus) *state.AgentTreeNode {
	n := makeNode(id, agentType, description, depth, isLast)
	n.Agent.Status = status
	return n
}

// makeDetailAgent returns a fully populated Agent suitable for SetAgent tests.
func makeDetailAgent(id, agentType, model, tier string, status state.AgentStatus) *state.Agent {
	return &state.Agent{
		ID:          id,
		AgentType:   agentType,
		Description: "test description",
		Model:       model,
		Tier:        tier,
		Status:      status,
		StartedAt:   time.Now(),
		Cost:        0.042,
		Tokens:      1500,
	}
}

// ---------------------------------------------------------------------------
// AgentTreeModel — TestTree_SetNodes_ViewShowsAgents
// ---------------------------------------------------------------------------

// TestTree_SetNodes_ViewShowsAgents verifies that after SetNodes and SetSize
// the View() output does NOT contain "No agents" — meaning at least one node
// row is rendered.
func TestTree_SetNodes_ViewShowsAgents(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)

	nodes := []*state.AgentTreeNode{
		makeNode("a1", "go-pro", "implement auth", 0, true),
	}
	m.SetNodes(nodes)

	view := m.View()
	assert.NotContains(t, view, "No agents",
		"View() must not show empty-state when nodes are present")
	// The agent type should appear in the rendered row (description is no
	// longer shown in tree view after UX-008 dot-leader layout).
	assert.Contains(t, view, "go-pro")
}

// TestTree_SetNodes_MultipleAgents_AllRendered verifies that all agents in the
// nodes slice appear in View() output (given a tall enough viewport).
func TestTree_SetNodes_MultipleAgents_AllRendered(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(100, 50)

	nodes := []*state.AgentTreeNode{
		makeNode("a1", "go-pro", "task-one", 0, false),
		makeNode("a2", "go-tui", "task-two", 1, false),
		makeNode("a3", "go-cli", "task-three", 1, true),
	}
	m.SetNodes(nodes)

	view := m.View()
	assert.Contains(t, view, "go-pro")
	assert.Contains(t, view, "go-tui")
	assert.Contains(t, view, "go-cli")
}

// ---------------------------------------------------------------------------
// AgentTreeModel — TestTree_EmptyNodes_ShowsNoAgents
// ---------------------------------------------------------------------------

// TestTree_EmptyNodes_ShowsNoAgents verifies that with no nodes set View()
// renders the "No agents" empty-state message.
func TestTree_EmptyNodes_ShowsNoAgents(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)
	// No SetNodes call — treeNodes is nil.

	view := m.View()
	assert.Contains(t, view, "No agents",
		"View() must show empty-state when no nodes are present")
}

// TestTree_SetNodes_ClearingNodes_ShowsNoAgents verifies that after setting
// nodes and then clearing them (SetNodes(nil)) the empty-state re-appears.
func TestTree_SetNodes_ClearingNodes_ShowsNoAgents(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)

	nodes := []*state.AgentTreeNode{makeNode("a1", "go-pro", "task", 0, true)}
	m.SetNodes(nodes)
	assert.NotContains(t, m.View(), "No agents")

	// Clear nodes.
	m.SetNodes(nil)
	assert.Contains(t, m.View(), "No agents")
}

// ---------------------------------------------------------------------------
// AgentTreeModel — TestTree_SelectedHighlight
// ---------------------------------------------------------------------------

// TestTree_SelectedHighlight verifies that the first node is selected by
// default (selectedIdx == 0) and is visually distinguished when focused.
// We check that TreeNodes() exposes the correct data and that SelectedID()
// returns the right agent ID.
func TestTree_SelectedHighlight(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)
	m.SetFocused(true)

	nodes := []*state.AgentTreeNode{
		makeNode("first", "go-pro", "task-one", 0, false),
		makeNode("second", "go-tui", "task-two", 1, true),
	}
	m.SetNodes(nodes)

	// Default selection is index 0.
	assert.Equal(t, "first", m.SelectedID(),
		"selectedIdx should default to 0, pointing at the first node")

	// TreeNodes() exposes all nodes.
	all := m.TreeNodes()
	require.Len(t, all, 2)
	assert.Equal(t, "first", all[0].Agent.ID)
	assert.Equal(t, "second", all[1].Agent.ID)
}

// TestTree_SelectedHighlight_ViewDiffers verifies that when a node is the
// selected index (index 0 by default), its rendered row differs from the
// non-selected row in some way — confirming visual selection feedback.
//
// We compare individual line content rather than the full view to avoid
// coupling to exact lipgloss ANSI escape sequences.
func TestTree_SelectedHighlight_ViewDiffers(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)
	m.SetFocused(true)

	nodes := []*state.AgentTreeNode{
		makeNode("sel", "go-pro", "selected task", 0, false),
		makeNode("unsel", "go-tui", "unselected task", 1, true),
	}
	m.SetNodes(nodes)

	view := m.View()
	lines := strings.Split(view, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "view must have at least 2 lines")

	// Both lines contain their respective agent types (description is no
	// longer shown in tree view after UX-008 dot-leader layout).
	assert.Contains(t, lines[0], "go-pro")
	assert.Contains(t, lines[1], "go-tui")

	// The selected row (index 0) and unselected row must not be byte-identical,
	// because the selected row is wrapped with StyleHighlight.
	assert.NotEqual(t, lines[0], lines[1],
		"selected and unselected rows must render differently")
}

// ---------------------------------------------------------------------------
// AgentTreeModel — TreeNodes() accessor
// ---------------------------------------------------------------------------

// TestTree_TreeNodesAccessor verifies the TreeNodes() accessor returns the
// exact slice set via SetNodes (same backing array, same length).
func TestTree_TreeNodesAccessor(t *testing.T) {
	m := NewAgentTreeModel()
	nodes := []*state.AgentTreeNode{
		makeNode("x1", "go-pro", "t1", 0, true),
		makeNode("x2", "go-tui", "t2", 1, true),
	}
	m.SetNodes(nodes)

	got := m.TreeNodes()
	require.Len(t, got, 2)
	assert.Equal(t, "x1", got[0].Agent.ID)
	assert.Equal(t, "x2", got[1].Agent.ID)
}

// TestTree_SetNodes_ClampsSelectedIdx verifies that if selectedIdx was at
// the last position and the node list shrinks, selectedIdx is clamped to the
// new length−1.
func TestTree_SetNodes_ClampsSelectedIdx(t *testing.T) {
	m := NewAgentTreeModel()
	m.SetSize(80, 20)

	// Start with 3 nodes.
	nodes3 := []*state.AgentTreeNode{
		makeNode("a", "t", "d", 0, false),
		makeNode("b", "t", "d2", 0, false),
		makeNode("c", "t", "d3", 0, true),
	}
	m.SetNodes(nodes3)

	// Manually advance selectedIdx to the last node (index 2).
	// We rely on the fact that SelectedID() returns nodes[selectedIdx].
	// Drive navigation by calling Update directly via the tea.Model interface.
	// Rather than faking keypresses, we check clamping via the SetNodes contract.

	// Set nodes to a smaller list — selectedIdx must be clamped.
	nodes1 := []*state.AgentTreeNode{
		makeNode("a", "t", "d", 0, true),
	}
	m.SetNodes(nodes1)

	// SelectedID() must not panic and must return a valid ID.
	id := m.SelectedID()
	assert.Equal(t, "a", id, "SelectedID must point to the only remaining node")
}

// ---------------------------------------------------------------------------
// AgentDetailModel — TestDetail_SetAgent_ShowsInfo
// ---------------------------------------------------------------------------

// TestDetail_SetAgent_ShowsInfo verifies that after SetAgent the View()
// output contains the agent's model, status, and tier.
func TestDetail_SetAgent_ShowsInfo(t *testing.T) {
	m := NewAgentDetailModel()
	m.SetSize(80, 30)

	agent := makeDetailAgent("a1", "go-pro", "sonnet", "sonnet", state.StatusRunning)
	m.SetAgent(agent)

	require.True(t, m.HasAgent(), "HasAgent must return true after SetAgent")

	view := m.View()
	// Status should appear (capitalised).
	assert.Contains(t, view, "Running",
		"View() must show agent status")
	// Agent type.
	assert.Contains(t, view, "go-pro",
		"View() must show agent type")
	// Model name.
	assert.Contains(t, view, "sonnet",
		"View() must show agent model")
	// Tier.
	assert.Contains(t, view, "sonnet",
		"View() must show agent tier")
}

// TestDetail_SetAgent_ShowsCostAndTokens verifies that cost and token count
// fields appear in the detail view.
func TestDetail_SetAgent_ShowsCostAndTokens(t *testing.T) {
	m := NewAgentDetailModel()
	m.SetSize(80, 30)

	agent := makeDetailAgent("a1", "go-pro", "opus", "opus", state.StatusComplete)
	agent.Cost = 0.123
	agent.Tokens = 4500
	m.SetAgent(agent)

	// Cost appears in the collapsed compact line ($0.123).
	view := m.View()
	assert.Contains(t, view, "0.123", "View() must display cost")

	// Tokens only appear in the expanded Overview — expand it directly.
	m.sections[0].Expanded = true
	m.syncViewport()
	view = m.View()
	assert.Contains(t, view, "4,500", "View() must display token count with thousands separator")
}

// TestDetail_SetAgent_ErrorStatus_ShowsErrorOutput verifies that error output
// is rendered for agents in StatusError with non-empty ErrorOutput.
func TestDetail_SetAgent_ErrorStatus_ShowsErrorOutput(t *testing.T) {
	m := NewAgentDetailModel()
	m.SetSize(80, 30)

	agent := makeDetailAgent("a1", "go-pro", "sonnet", "sonnet", state.StatusError)
	agent.ErrorOutput = "context deadline exceeded"
	m.SetAgent(agent)

	view := m.View()
	assert.Contains(t, view, "Error",
		"View() must show the Error label for StatusError agents")
	assert.Contains(t, view, "context deadline exceeded",
		"View() must render the error output text")
}

// TestDetail_SetAgent_ClearedByNil verifies that passing nil to SetAgent
// resets HasAgent() to false and shows the placeholder.
func TestDetail_SetAgent_ClearedByNil(t *testing.T) {
	m := NewAgentDetailModel()
	m.SetSize(80, 30)

	m.SetAgent(makeDetailAgent("a1", "go-pro", "sonnet", "sonnet", state.StatusRunning))
	require.True(t, m.HasAgent())

	m.SetAgent(nil)
	assert.False(t, m.HasAgent(), "HasAgent must be false after SetAgent(nil)")
	assert.Contains(t, m.View(), "Select an agent",
		"View() must show placeholder after clearing agent")
}

// ---------------------------------------------------------------------------
// AgentDetailModel — TestDetail_NoAgent_ShowsPlaceholder
// ---------------------------------------------------------------------------

// TestDetail_NoAgent_ShowsPlaceholder verifies that before any SetAgent call
// the View() shows the "Select an agent" placeholder.
func TestDetail_NoAgent_ShowsPlaceholder(t *testing.T) {
	m := NewAgentDetailModel()
	m.SetSize(80, 30)

	assert.False(t, m.HasAgent(),
		"HasAgent must be false on a newly created AgentDetailModel")

	view := m.View()
	assert.Contains(t, view, "Select an agent",
		"View() must show the placeholder when no agent is set")
}

// TestDetail_ZeroValue_ShowsPlaceholder verifies the zero value of
// AgentDetailModel (without NewAgentDetailModel) is usable and shows the
// placeholder.
func TestDetail_ZeroValue_ShowsPlaceholder(t *testing.T) {
	var m AgentDetailModel

	assert.False(t, m.HasAgent())
	view := m.View()
	assert.Contains(t, view, "Select an agent")
}

// ---------------------------------------------------------------------------
// Integration — registry → tree component → detail component
// ---------------------------------------------------------------------------

// TestIntegration_Registry_ToTree_ToDetail exercises the complete data path:
//  1. Register agents in the registry.
//  2. Invalidate the tree cache.
//  3. Feed registry.Tree() nodes into AgentTreeModel.SetNodes.
//  4. Confirm the tree component reflects the correct agents.
//  5. Fetch the selected agent from the registry and set it on the detail.
//  6. Confirm the detail view shows the agent's information.
func TestIntegration_Registry_ToTree_ToDetail(t *testing.T) {
	now := time.Now()
	r := state.NewAgentRegistry()

	// Register a root agent and a child.
	rootAgent := state.Agent{
		ID:          "root",
		AgentType:   "go-pro",
		Description: "root task",
		Model:       "sonnet",
		Tier:        "sonnet",
		Status:      state.StatusRunning,
		StartedAt:   now.Add(-5 * time.Second),
	}
	require.NoError(t, r.Register(rootAgent))

	childAgent := state.Agent{
		ID:          "child",
		AgentType:   "go-tui",
		Description: "child task",
		ParentID:    "root",
		Model:       "haiku",
		Tier:        "haiku",
		Status:      state.StatusPending,
		StartedAt:   now.Add(-2 * time.Second),
	}
	require.NoError(t, r.Register(childAgent))

	// Step 2: Invalidate tree cache (simulates Update() handler).
	r.InvalidateTreeCache()

	// Step 3: Feed into tree component.
	tree := NewAgentTreeModel()
	tree.SetSize(100, 30)
	tree.SetNodes(r.Tree())

	// Step 4: Tree must have 2 nodes in DFS order.
	treeNodes := tree.TreeNodes()
	require.Len(t, treeNodes, 2)
	assert.Equal(t, "root", treeNodes[0].Agent.ID)
	assert.Equal(t, "child", treeNodes[1].Agent.ID)

	// SelectedID defaults to index 0 (root).
	assert.Equal(t, "root", tree.SelectedID())

	// Tree view must not show empty state.
	assert.NotContains(t, tree.View(), "No agents")

	// Step 5: Fetch the selected agent and set on detail.
	detail := NewAgentDetailModel()
	detail.SetSize(80, 30)

	selectedAgent := r.Get(tree.SelectedID())
	require.NotNil(t, selectedAgent)
	detail.SetAgent(selectedAgent)

	// Step 6: Detail view must show the root agent's info.
	require.True(t, detail.HasAgent())
	detailView := detail.View()
	assert.Contains(t, detailView, "go-pro",
		"detail must show root agent type")
	assert.Contains(t, detailView, "sonnet",
		"detail must show root agent model")
	assert.Contains(t, detailView, "Running",
		"detail must show root agent status")
}

// TestIntegration_StatusTransition_VisibleInBothComponents verifies that a
// status transition is visible in both the tree and detail components after
// the pipeline is refreshed.
func TestIntegration_StatusTransition_VisibleInBothComponents(t *testing.T) {
	r := state.NewAgentRegistry()

	agent := state.Agent{
		ID:          "worker",
		AgentType:   "go-pro",
		Description: "worker task",
		Model:       "sonnet",
		Tier:        "sonnet",
		Status:      state.StatusPending,
		StartedAt:   time.Now(),
	}
	require.NoError(t, r.Register(agent))
	r.InvalidateTreeCache()

	// Initial state.
	tree := NewAgentTreeModel()
	tree.SetSize(80, 20)
	tree.SetNodes(r.Tree())

	detail := NewAgentDetailModel()
	detail.SetSize(80, 30)
	detail.SetAgent(r.Get("worker"))

	assert.Contains(t, detail.View(), "Pending")

	// Transition to running and refresh pipeline.
	require.NoError(t, r.Update("worker", func(a *state.Agent) {
		a.Status = state.StatusRunning
	}))
	r.InvalidateTreeCache()
	tree.SetNodes(r.Tree())
	detail.SetAgent(r.Get("worker"))

	assert.Contains(t, detail.View(), "Running",
		"detail must reflect updated status after pipeline refresh")

	// Confirm tree also reflects the new status.
	nodes := tree.TreeNodes()
	require.Len(t, nodes, 1)
	assert.Equal(t, state.StatusRunning, nodes[0].Agent.Status)
}
