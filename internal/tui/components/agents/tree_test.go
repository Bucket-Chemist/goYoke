package agents_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	// Children at depth 1 use 2-space indentation instead of box-drawing connectors.
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	foundIndented := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Errorf("View() should have at least one child line with 2-space indent; got:\n%s", view)
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
	// depth ≥ 2 produces at least 4-space indentation (2 spaces per level).
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	found4space := false
	for _, line := range lines {
		if strings.HasPrefix(line, "    ") {
			found4space = true
			break
		}
	}
	if !found4space {
		t.Errorf("deep tree View() missing depth-2 indentation (4 spaces); got:\n%s", view)
	}
}

func TestView_NoActivityInTree(t *testing.T) {
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

	// Activity preview is shown in the detail panel only, not the tree view.
	if strings.Contains(view, "Reading file.go") {
		t.Errorf("View() should NOT show activity preview in tree; got:\n%s", view)
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


// ---------------------------------------------------------------------------
// Inline cost display (UX-010)
// ---------------------------------------------------------------------------

func TestView_InlineCost_ShowsDollarWhenPositive(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "task", state.StatusRunning), 0, true)
	node.Agent.Cost = 2.47
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(80, 20)
	view := m.View()

	if !strings.Contains(view, "$2.47") {
		t.Errorf("View() should show '$2.47' for agent with cost; got:\n%s", view)
	}
	// Should NOT show status text when cost is present.
	if strings.Contains(view, "run") {
		t.Errorf("View() should not show status text when cost > 0; got:\n%s", view)
	}
}

func TestView_InlineCost_ShowsStatusWhenZero(t *testing.T) {
	tests := []struct {
		status state.AgentStatus
		want   string
	}{
		{state.StatusRunning, "run"},
		{state.StatusComplete, "done"},
		{state.StatusError, "fail"},
		{state.StatusPending, "wait"},
		{state.StatusKilled, "kill"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			node := makeNode(makeAgent("root", "", "go-pro", "task", tc.status), 0, true)
			// Cost is 0 (default) — should show status string.
			m := agents.NewAgentTreeModel()
			m.SetNodes([]*state.AgentTreeNode{node})
			m.SetSize(80, 20)
			view := m.View()

			if !strings.Contains(view, tc.want) {
				t.Errorf("status %s: want %q in view; got:\n%s", tc.status, tc.want, view)
			}
		})
	}
}

func TestView_InlineCost_TwoDecimalPlaces(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "task", state.StatusComplete), 0, true)
	node.Agent.Cost = 0.1
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(80, 20)
	view := m.View()

	if !strings.Contains(view, "$0.10") {
		t.Errorf("View() should format cost to 2 decimal places; got:\n%s", view)
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

// ---------------------------------------------------------------------------
// Render — RenderFull
// ---------------------------------------------------------------------------

func TestRender_FullMatchesView(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(80, 20)

	if got, want := m.Render(agents.RenderFull, 80), m.View(); got != want {
		t.Errorf("Render(RenderFull, 80) != View()\nRender: %q\nView:   %q", got, want)
	}
}

func TestRender_Full_EmptyMatchesView(t *testing.T) {
	m := agents.NewAgentTreeModel()
	if got, want := m.Render(agents.RenderFull, 80), m.View(); got != want {
		t.Errorf("Render(RenderFull) on empty tree != View()")
	}
}

// ---------------------------------------------------------------------------
// Render — RenderIconRail
// ---------------------------------------------------------------------------

func TestRender_IconRail_EmptyTree(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetSize(22, 20)

	view := m.Render(agents.RenderIconRail, 22)
	if !strings.Contains(view, "No agents") {
		t.Errorf("Render(RenderIconRail) on empty tree should contain 'No agents'; got:\n%s", view)
	}
}

func TestRender_IconRail_ContainsAbbreviations(t *testing.T) {
	m := agents.NewAgentTreeModel()
	// threeNodeTree: root=orchestrator, c1=go-pro, c2=code-reviewer
	m.SetNodes(threeNodeTree())
	m.SetSize(22, 20)

	view := m.Render(agents.RenderIconRail, 22)

	// First 2 chars uppercase: "OR", "GO", "CO"
	for _, abbrev := range []string{"OR", "GO", "CO"} {
		if !strings.Contains(view, abbrev) {
			t.Errorf("Render(RenderIconRail, 22) missing abbrev %q; got:\n%s", abbrev, view)
		}
	}
}

func TestRender_IconRail_ShowsCostWhenNonZero(t *testing.T) {
	node := makeNode(makeAgent("root", "", "go-pro", "task", state.StatusRunning), 0, true)
	node.Agent.Cost = 1.98
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(22, 20)

	view := m.Render(agents.RenderIconRail, 22)
	if !strings.Contains(view, "$1.98") {
		t.Errorf("icon rail should show '$1.98' for agent with cost; got:\n%s", view)
	}
}

func TestRender_IconRail_ShowsStatusWhenNoCost(t *testing.T) {
	tests := []struct {
		status state.AgentStatus
		want   string
	}{
		{state.StatusRunning, "run"},
		{state.StatusComplete, "done"},
		{state.StatusError, "fail"},
		{state.StatusPending, "wait"},
		{state.StatusKilled, "kill"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			node := makeNode(makeAgent("root", "", "go-pro", "task", tc.status), 0, true)
			// Cost is 0 (default) — should show status string.
			m := agents.NewAgentTreeModel()
			m.SetNodes([]*state.AgentTreeNode{node})
			m.SetSize(22, 20)

			view := m.Render(agents.RenderIconRail, 22)
			if !strings.Contains(view, tc.want) {
				t.Errorf("status %s: want %q in icon rail; got:\n%s", tc.status, tc.want, view)
			}
		})
	}
}

func TestRender_IconRail_CorrectLineCount(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(80, 20) // height 20, tree has 3 nodes

	view := m.Render(agents.RenderIconRail, 22)
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("3-node tree icon rail should produce 3 lines; got %d:\n%s", len(lines), view)
	}
}

// TestRender_IconRail_WidthBoundaries verifies that every rendered line fits within
// the specified width at the boundary widths required by UX-003.
func TestRender_IconRail_WidthBoundaries(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(80, 20)

	for _, width := range []int{15, 22, 28, 29, 30, 31, 32, 45} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			view := m.Render(agents.RenderIconRail, width)
			if view == "" {
				t.Fatalf("Render(RenderIconRail, %d) returned empty string", width)
			}
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			for i, line := range lines {
				w := lipgloss.Width(line)
				if w > width {
					t.Errorf("line %d: lipgloss.Width=%d exceeds available width=%d: %q",
						i, w, width, line)
				}
			}
		})
	}
}

func TestRender_IconRail_TreeConnectors(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(80, 20)

	view := m.Render(agents.RenderIconRail, 45)
	// Children at depth 1 should still have tree connectors.
	if !strings.Contains(view, "├─") && !strings.Contains(view, "└─") {
		t.Errorf("icon rail should preserve tree connectors; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Dot-leader layout (UX-008)
// ---------------------------------------------------------------------------

func TestView_DotLeaderLayout(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(threeNodeTree())
	m.SetSize(45, 20)
	view := m.View()

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	for i, line := range lines {
		// Each line must contain at least two consecutive dots (the dot leader).
		if !strings.Contains(line, "..") {
			t.Errorf("line %d: missing dot leaders; got:\n%q", i, line)
		}
	}
}

func TestView_NoBoxDrawing(t *testing.T) {
	m := agents.NewAgentTreeModel()
	m.SetNodes(deepTree())
	m.SetSize(80, 20)
	view := m.View()

	for _, ch := range []string{"├─", "└─", "│"} {
		if strings.Contains(view, ch) {
			t.Errorf("View() (full mode) must not contain box-drawing %q; got:\n%s", ch, view)
		}
	}
}

func TestView_WidthBoundaries(t *testing.T) {
	for _, w := range []int{22, 30, 45, 80} {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			m := agents.NewAgentTreeModel()
			m.SetNodes(threeNodeTree())
			m.SetSize(w, 20)
			view := m.View()
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			for i, line := range lines {
				got := lipgloss.Width(line)
				if got > w {
					t.Errorf("line %d at width=%d: lipgloss.Width=%d exceeds width: %q",
						i, w, got, line)
				}
			}
		})
	}
}

// TestView_ANSISafeWidth verifies that every rendered line has exactly m.width
// visual columns even after lipgloss ANSI styling is applied.
func TestView_ANSISafeWidth(t *testing.T) {
	for _, w := range []int{22, 30, 45, 80} {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			m := agents.NewAgentTreeModel()
			m.SetNodes(threeNodeTree())
			m.SetSize(w, 20)
			view := m.View()
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			for i, line := range lines {
				got := lipgloss.Width(line)
				if got != w {
					t.Errorf("ANSI width: line %d at width=%d: lipgloss.Width=%d (want %d): %q",
						i, w, got, w, line)
				}
			}
		})
	}
}

func TestView_RightAlignedValues(t *testing.T) {
	// Status words are the last text on every unstyled line.
	node := makeNode(makeAgent("root", "", "go-pro", "task", state.StatusComplete), 0, true)
	node.Agent.Cost = 1.50
	m := agents.NewAgentTreeModel()
	m.SetNodes([]*state.AgentTreeNode{node})
	m.SetSize(45, 20)
	view := m.View()

	// The cost string must be the rightmost content (exact width == 45).
	got := lipgloss.Width(view)
	if got != 45 {
		t.Errorf("right-aligned row: lipgloss.Width=%d, want 45: %q", got, view)
	}
	if !strings.Contains(view, "$1.50") {
		t.Errorf("cost not present in row: %q", view)
	}
}

// ---------------------------------------------------------------------------
// Status row style (UX-009)
// ---------------------------------------------------------------------------

func TestStatusRowStyle(t *testing.T) {
	// Complete must be bold (distinguishes it from Running at same color).
	complete := agents.StatusRowStyle(state.StatusComplete)
	if !complete.GetBold() {
		t.Error("StatusRowStyle(Complete) must be bold")
	}

	// Running must NOT be bold (dimmer treatment than Complete).
	running := agents.StatusRowStyle(state.StatusRunning)
	if running.GetBold() {
		t.Error("StatusRowStyle(Running) must not be bold")
	}

	// Killed must have strikethrough.
	killed := agents.StatusRowStyle(state.StatusKilled)
	if !killed.GetStrikethrough() {
		t.Error("StatusRowStyle(Killed) must have strikethrough")
	}

	// Pending and Error must not have strikethrough.
	for _, s := range []state.AgentStatus{state.StatusPending, state.StatusError} {
		style := agents.StatusRowStyle(s)
		if style.GetStrikethrough() {
			t.Errorf("StatusRowStyle(%s) must not have strikethrough", s)
		}
	}

	// All statuses must set a foreground color (not NoColor).
	statuses := []state.AgentStatus{
		state.StatusRunning,
		state.StatusComplete,
		state.StatusError,
		state.StatusKilled,
		state.StatusPending,
	}
	for _, s := range statuses {
		style := agents.StatusRowStyle(s)
		fg := style.GetForeground()
		if _, isNone := fg.(lipgloss.NoColor); isNone {
			t.Errorf("StatusRowStyle(%s) must set a foreground color", s)
		}
	}
}

func TestView_FullRowColorByStatus(t *testing.T) {
	// Build a tree with one agent per status so every code path in
	// StatusRowStyle is exercised through renderNode.
	statusCases := []struct {
		status  state.AgentStatus
		agentType string
	}{
		{state.StatusRunning, "running-agent"},
		{state.StatusComplete, "complete-agent"},
		{state.StatusError, "error-agent"},
		{state.StatusKilled, "killed-agent"},
		{state.StatusPending, "pending-agent"},
	}

	var nodes []*state.AgentTreeNode
	for i, tc := range statusCases {
		a := makeAgent(fmt.Sprintf("id-%d", i), "", tc.agentType, "desc", tc.status)
		nodes = append(nodes, makeNode(a, 0, i == len(statusCases)-1))
	}

	m := agents.NewAgentTreeModel()
	m.SetNodes(nodes)
	m.SetSize(80, 20)
	view := m.View()

	// Every agent type must appear in the rendered output.
	for _, tc := range statusCases {
		if !strings.Contains(view, tc.agentType) {
			t.Errorf("View() should contain agent type %q; got:\n%s", tc.agentType, view)
		}
	}

	// Every line must have the correct visual width (row styling must not alter width).
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	for i, line := range lines {
		got := lipgloss.Width(line)
		if got != 80 {
			t.Errorf("line %d: lipgloss.Width=%d, want 80: %q", i, got, line)
		}
	}
}
