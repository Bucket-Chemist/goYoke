package agents_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeAgent constructs a minimal state.Agent for testing.
func makeAgent(id, parentID, agentType, description string, status state.AgentStatus) *state.Agent {
	return &state.Agent{
		ID:          id,
		ParentID:    parentID,
		AgentType:   agentType,
		Description: description,
		Status:      status,
		StartedAt:   time.Now(),
	}
}

// makeNode constructs an AgentTreeNode for use in tests.
func makeNode(agent *state.Agent, depth int, isLast bool) *state.AgentTreeNode {
	return &state.AgentTreeNode{
		Agent:  agent,
		Depth:  depth,
		IsLast: isLast,
	}
}

// singleNodeTree returns a one-node tree (root only).
func singleNodeTree() []*state.AgentTreeNode {
	return []*state.AgentTreeNode{
		makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true),
	}
}

// threeNodeTree returns root + two children.
func threeNodeTree() []*state.AgentTreeNode {
	root := makeAgent("root", "", "orchestrator", "orchestrate", state.StatusRunning)
	child1 := makeAgent("c1", "root", "go-pro", "implement feature", state.StatusComplete)
	child2 := makeAgent("c2", "root", "code-reviewer", "review code", state.StatusPending)
	return []*state.AgentTreeNode{
		makeNode(root, 0, true),
		makeNode(child1, 1, false),
		makeNode(child2, 1, true),
	}
}

// deepTree returns a 4-level deep tree.
func deepTree() []*state.AgentTreeNode {
	a := makeAgent("a", "", "root-agent", "root", state.StatusRunning)
	b := makeAgent("b", "a", "child-agent", "child", state.StatusRunning)
	c := makeAgent("c", "b", "grandchild", "grandchild", state.StatusComplete)
	d := makeAgent("d", "c", "great-grandchild", "deep", state.StatusError)
	return []*state.AgentTreeNode{
		makeNode(a, 0, true),
		makeNode(b, 1, true),
		makeNode(c, 2, true),
		makeNode(d, 3, true),
	}
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewAgentTreeModel(t *testing.T) {
	m := agents.NewAgentTreeModel()
	if m.SelectedID() != "" {
		t.Errorf("SelectedID() on empty model = %q; want empty string", m.SelectedID())
	}
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

// ---------------------------------------------------------------------------
// SetNodes
// ---------------------------------------------------------------------------

func TestSetNodes_PopulatesTree(t *testing.T) {
	m := agents.NewAgentTreeModel()
	nodes := singleNodeTree()
	m.SetNodes(nodes)
	if m.SelectedID() != "root" {
		t.Errorf("SelectedID() after SetNodes = %q; want %q", m.SelectedID(), "root")
	}
}

func TestSetNodes_ClampsSelection(t *testing.T) {
	m := agents.NewAgentTreeModel()
	// Start with 3 nodes, navigate to last.
	m.SetNodes(threeNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	newM, _ = newM.(agents.AgentTreeModel).Update(tea.KeyMsg{Type: tea.KeyDown})
	tm := newM.(agents.AgentTreeModel)
	// Now shrink tree to 1 node.
	tm.SetNodes(singleNodeTree())
	// selectedIdx should be clamped to 0.
	if tm.SelectedID() != "root" {
		t.Errorf("SelectedID() after shrink = %q; want %q", tm.SelectedID(), "root")
	}
}

func TestSetNodes_EmptyTree(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(nil)
	if m.SelectedID() != "" {
		t.Errorf("SelectedID() on nil nodes = %q; want empty", m.SelectedID())
	}
}

// ---------------------------------------------------------------------------
// Keyboard navigation
// ---------------------------------------------------------------------------

func TestNavigateDown(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd == nil {
		t.Error("Down key with movement should emit a command")
	}
	tm := newM.(agents.AgentTreeModel)
	if tm.SelectedID() != "c1" {
		t.Errorf("after Down SelectedID() = %q; want %q", tm.SelectedID(), "c1")
	}
}

func TestNavigateUp(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	// Move down then back up.
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	newM, cmd := newM.(agents.AgentTreeModel).Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd == nil {
		t.Error("Up key with movement should emit a command")
	}
	tm := newM.(agents.AgentTreeModel)
	if tm.SelectedID() != "root" {
		t.Errorf("after Up SelectedID() = %q; want %q", tm.SelectedID(), "root")
	}
}

func TestNavigateDown_Clamps(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(singleNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	// Navigate down on a 1-item list should not move.
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_ = cmd
	tm := newM.(agents.AgentTreeModel)
	if tm.SelectedID() != "root" {
		t.Errorf("Down on single-item list should stay at root; got %q", tm.SelectedID())
	}
}

func TestNavigateUp_Clamps(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(singleNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	_ = cmd
	tm := newM.(agents.AgentTreeModel)
	if tm.SelectedID() != "root" {
		t.Errorf("Up on single-item list should stay at root; got %q", tm.SelectedID())
	}
}

func TestEnterEmitsAgentSelectedMsg(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(singleNodeTree())
	m.SetFocused(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should return a non-nil command")
	}
	// Enter emits a batched command (AgentSelectedMsg + AgentDetailFocusMsg).
	// tea.Batch returns a BatchMsg which is a []Cmd.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Enter command produced %T; want tea.BatchMsg", msg)
	}

	// Find AgentSelectedMsg in the batch.
	foundSelected := false
	foundFocus := false
	for _, c := range batch {
		if c == nil {
			continue
		}
		m := c()
		if sel, ok := m.(agents.AgentSelectedMsg); ok {
			foundSelected = true
			if sel.AgentID != "root" {
				t.Errorf("AgentSelectedMsg.AgentID = %q; want %q", sel.AgentID, "root")
			}
		}
		if _, ok := m.(agents.AgentDetailFocusMsg); ok {
			foundFocus = true
		}
	}
	if !foundSelected {
		t.Error("Enter batch missing AgentSelectedMsg")
	}
	if !foundFocus {
		t.Error("Enter batch missing AgentDetailFocusMsg")
	}
}

func TestDownEmitsAgentSelectedMsg(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd == nil {
		t.Fatal("Down key should return a non-nil command")
	}
	msg := cmd()
	sel, ok := msg.(agents.AgentSelectedMsg)
	if !ok {
		t.Fatalf("Down command produced %T; want AgentSelectedMsg", msg)
	}
	if sel.AgentID != "c1" {
		t.Errorf("AgentSelectedMsg.AgentID = %q; want %q", sel.AgentID, "c1")
	}
}

// ---------------------------------------------------------------------------
// Focus gating
// ---------------------------------------------------------------------------

func TestUnfocused_IgnoresKeys(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetFocused(false)
	m.SetSize(80, 20)

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Error("unfocused tree should not emit commands on key press")
	}
	if newM.(agents.AgentTreeModel).SelectedID() != "root" {
		t.Error("unfocused tree should not move selection")
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_EmptyState(t *testing.T) {
	m := agents.NewAgentTreeModel()
	view := m.View()
	if !strings.Contains(view, "No agents") {
		t.Errorf("empty tree View() should contain 'No agents'; got:\n%s", view)
	}
}

func TestView_ContainsAgentTypes(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(120, 20)
	view := m.View()

	for _, typ := range []string{"orchestrator", "go-pro", "code-reviewer"} {
		if !strings.Contains(view, typ) {
			t.Errorf("View() missing agent type %q; got:\n%s", typ, view)
		}
	}
}

func TestView_ContainsStatusIcons(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(120, 20)
	view := m.View()

	// Running root: '>', Complete child1: '*', Pending child2: '.'
	for _, icon := range []string{">", "*", "."} {
		if !strings.Contains(view, icon) {
			t.Errorf("View() missing status icon %q; got:\n%s", icon, view)
		}
	}
}

func TestView_ContainsTreeConnectors(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(120, 20)
	view := m.View()

	// The two children should trigger at least one connector.
	if !strings.Contains(view, "├─") && !strings.Contains(view, "└─") {
		t.Errorf("View() missing tree connectors (├─ or └─); got:\n%s", view)
	}
}

func TestView_DeepTree(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(deepTree())
	m.SetSize(120, 20)
	view := m.View()

	// All four agent types must appear.
	for _, typ := range []string{"root-agent", "child-agent", "grandchild", "great-grandchild"} {
		if !strings.Contains(view, typ) {
			t.Errorf("deep tree View() missing %q; got:\n%s", typ, view)
		}
	}
	// depth ≥ 2 produces "│ " indentation.
	if !strings.Contains(view, "│") {
		t.Errorf("deep tree View() missing '│' indentation; got:\n%s", view)
	}
}

func TestView_ActivityPreview(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true)
	node.Agent.Activity = &state.AgentActivity{
		Type:    "tool_use",
		Target:  "Read",
		Preview: "Reading file.go",
	}
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(120, 20)
	view := m.View()

	if !strings.Contains(view, "Reading file.go") {
		t.Errorf("View() missing activity preview; got:\n%s", view)
	}
}

func TestView_Scrolling(t *testing.T) {
	// Build 10-node tree and a viewport of height 3.
	var nodes []*state.AgentTreeNode
	for i := range 10 {
		id := fmt.Sprintf("agent-%d", i)
		typ := fmt.Sprintf("type-%d", i)
		nodes = append(nodes, makeNode(makeAgent(id, "", typ, "task", state.StatusPending), 0, i == 9))
	}

	m := agents.NewAgentTreeModel()
	m.SetNodes(nodes)
	m.SetFocused(true)
	m.SetSize(120, 3)

	// Only the first 3 agents should be visible initially.
	view := m.View()
	if !strings.Contains(view, "type-0") {
		t.Errorf("initial scroll: type-0 should be visible; got:\n%s", view)
	}
	if strings.Contains(view, "type-9") {
		t.Errorf("initial scroll: type-9 should NOT be visible; got:\n%s", view)
	}
}

func TestView_MultipleRows(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(120, 20)
	view := m.View()

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("three-node tree should render 3 lines; got %d:\n%s", len(lines), view)
	}
}

// ---------------------------------------------------------------------------
// AC progress display
// ---------------------------------------------------------------------------

func TestView_ACProgress_Incomplete(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true)
	node.Agent.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "ac1", Completed: true},
		{Text: "ac2", Completed: true},
		{Text: "ac3", Completed: false},
		{Text: "ac4", Completed: false},
	}
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(120, 20)
	view := m.View()

	if !strings.Contains(view, "2/4 AC") {
		t.Errorf("View() with 2/4 complete AC should contain '2/4 AC'; got:\n%s", view)
	}
}

func TestView_ACProgress_Complete(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true)
	node.Agent.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "ac1", Completed: true},
		{Text: "ac2", Completed: true},
		{Text: "ac3", Completed: true},
		{Text: "ac4", Completed: true},
	}
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(120, 20)
	view := m.View()

	if !strings.Contains(view, "4/4 AC") {
		t.Errorf("View() with 4/4 complete AC should contain '4/4 AC'; got:\n%s", view)
	}
}

func TestView_NoAC(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true)
	// AcceptanceCriteria is nil by default — no AC text should appear.
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(120, 20)
	view := m.View()

	if strings.Contains(view, "AC") {
		t.Errorf("View() without AC should not contain 'AC'; got:\n%s", view)
	}
}

func TestView_ACBeforeActivity(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "root task", state.StatusRunning), 0, true)
	node.Agent.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "ac1", Completed: true},
		{Text: "ac2", Completed: false},
	}
	node.Agent.Activity = &state.AgentActivity{
		Type:    "tool_use",
		Target:  "Read",
		Preview: "Reading file.go",
	}
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(120, 20)
	view := m.View()

	acPos := strings.Index(view, "AC")
	actPos := strings.Index(view, "Reading file.go")
	if acPos == -1 {
		t.Fatalf("View() missing AC progress; got:\n%s", view)
	}
	if actPos == -1 {
		t.Fatalf("View() missing activity preview; got:\n%s", view)
	}
	if acPos > actPos {
		t.Errorf("AC progress (pos %d) should appear before activity preview (pos %d); got:\n%s", acPos, actPos, view)
	}
}

func TestAlternativeKeys(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetFocused(true)
	m.SetSize(80, 20)

	// vi-style 'j' for down.
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd == nil {
		t.Error("'j' key should emit a command")
	}
	tm := newM.(agents.AgentTreeModel)
	if tm.SelectedID() != "c1" {
		t.Errorf("after 'j' SelectedID() = %q; want %q", tm.SelectedID(), "c1")
	}

	// vi-style 'k' for up.
	newM2, cmd2 := tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd2 == nil {
		t.Error("'k' key should emit a command")
	}
	tm2 := newM2.(agents.AgentTreeModel)
	if tm2.SelectedID() != "root" {
		t.Errorf("after 'k' SelectedID() = %q; want %q", tm2.SelectedID(), "root")
	}
}
