package agents

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// TestNew verifies model initialization
func TestNew(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)

	if model.tree != tree {
		t.Error("Tree not set correctly")
	}

	if model.expanded == nil {
		t.Error("Expanded map not initialized")
	}

	if model.cursorPos != 0 {
		t.Error("Cursor should start at 0")
	}
}

// TestGetStatusIcon verifies status icon mapping
func TestGetStatusIcon(t *testing.T) {
	model := New(NewAgentTree("test-session"))

	tests := []struct {
		status   AgentStatus
		expected string
	}{
		{StatusSpawning, "⏳"},
		{StatusRunning, "●"},
		{StatusCompleted, "✓"},
		{StatusError, "✗"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			icon := model.getStatusIcon(tt.status)
			if icon != tt.expected {
				t.Errorf("Expected icon %s, got %s", tt.expected, icon)
			}
		})
	}
}

// TestRenderEmptyTree verifies empty tree rendering
func TestRenderEmptyTree(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)
	model.width = 80
	model.height = 20

	view := model.View()

	if !strings.Contains(view, "No agents running") {
		t.Error("Empty tree should display 'No agents running' message")
	}
}

// TestProcessSpawnAndRender verifies rendering after spawn event
func TestProcessSpawnAndRender(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Process spawn event
	event := &telemetry.AgentLifecycleEvent{
		AgentID:         "test-agent",
		SessionID:       "test-session",
		Tier:            "sonnet",
		TaskDescription: "Test task",
		Timestamp:       time.Now().Unix(),
	}

	err := tree.ProcessSpawn(event)
	if err != nil {
		t.Fatalf("ProcessSpawn failed: %v", err)
	}

	// Create model and render
	model := New(tree)
	model.width = 80
	model.height = 20
	(&model).rebuildVisibleNodes()

	view := model.View()

	// Check that view contains agent information
	if !strings.Contains(view, "test-agent") {
		t.Error("View should contain agent ID")
	}

	if !strings.Contains(view, "sonnet") {
		t.Error("View should contain tier")
	}

	// Check for spawning icon
	if !strings.Contains(view, "⏳") {
		t.Error("View should contain spawning icon")
	}
}

// TestRenderHierarchy verifies hierarchical rendering
func TestRenderHierarchy(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create parent agent
	parent := &telemetry.AgentLifecycleEvent{
		AgentID:   "parent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(parent)

	// Create child agent
	child := &telemetry.AgentLifecycleEvent{
		AgentID:     "child",
		SessionID:   "test-session",
		Tier:        "haiku",
		ParentAgent: "parent",
		Timestamp:   time.Now().Unix(),
	}
	tree.ProcessSpawn(child)

	// Create model and render
	model := New(tree)
	model.width = 80
	model.height = 20
	model.ExpandAll() // Expand all nodes
	(&model).rebuildVisibleNodes()

	view := model.View()

	// Check for both agents
	if !strings.Contains(view, "parent") {
		t.Error("View should contain parent agent")
	}

	if !strings.Contains(view, "child") {
		t.Error("View should contain child agent")
	}

	// Child should appear after parent in the output
	parentIdx := strings.Index(view, "parent")
	childIdx := strings.Index(view, "child")

	if childIdx <= parentIdx {
		t.Error("Child should appear after parent in view")
	}
}

// TestNavigationUpDown verifies cursor movement
func TestNavigationUpDown(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Add multiple agents
	for i := 0; i < 3; i++ {
		event := &telemetry.AgentLifecycleEvent{
			AgentID:   string(rune('a' + i)),
			SessionID: "test-session",
			Tier:      "haiku",
			Timestamp: time.Now().Unix(),
		}
		tree.ProcessSpawn(event)
	}

	model := New(tree)
	(&model).ExpandAll() // Expand all nodes so all are visible
	(&model).rebuildVisibleNodes()

	// Initial position should be 0
	if model.cursorPos != 0 {
		t.Errorf("Initial cursor position should be 0, got %d", model.cursorPos)
	}

	// Verify all 3 nodes are visible
	if len(model.visibleNodes) != 3 {
		t.Errorf("Expected 3 visible nodes, got %d", len(model.visibleNodes))
	}

	// Move down
	(&model).moveDown()
	if model.cursorPos != 1 {
		t.Errorf("After moveDown, cursor should be 1, got %d", model.cursorPos)
	}

	// Move down again
	(&model).moveDown()
	if model.cursorPos != 2 {
		t.Errorf("After second moveDown, cursor should be 2, got %d", model.cursorPos)
	}

	// Try to move down past end (should stay at 2)
	(&model).moveDown()
	if model.cursorPos != 2 {
		t.Errorf("Cursor should not move past end, got %d", model.cursorPos)
	}

	// Move up
	(&model).moveUp()
	if model.cursorPos != 1 {
		t.Errorf("After moveUp, cursor should be 1, got %d", model.cursorPos)
	}

	// Move up again
	(&model).moveUp()
	if model.cursorPos != 0 {
		t.Errorf("After second moveUp, cursor should be 0, got %d", model.cursorPos)
	}

	// Try to move up past start (should stay at 0)
	(&model).moveUp()
	if model.cursorPos != 0 {
		t.Errorf("Cursor should not move past start, got %d", model.cursorPos)
	}
}

// TestExpandCollapse verifies expand/collapse functionality
func TestExpandCollapse(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create parent with children
	parent := &telemetry.AgentLifecycleEvent{
		AgentID:   "parent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(parent)

	child := &telemetry.AgentLifecycleEvent{
		AgentID:     "child",
		SessionID:   "test-session",
		Tier:        "haiku",
		ParentAgent: "parent",
		Timestamp:   time.Now().Unix(),
	}
	tree.ProcessSpawn(child)

	model := New(tree)
	(&model).rebuildVisibleNodes()

	// Initially collapsed - should only see parent
	if len(model.visibleNodes) != 1 {
		t.Errorf("Initially collapsed, should have 1 visible node, got %d", len(model.visibleNodes))
	}

	// Expand parent
	model.selectedID = "parent"
	model.toggleExpand()
	(&model).rebuildVisibleNodes()

	// Should now see both parent and child
	if len(model.visibleNodes) != 2 {
		t.Errorf("After expand, should have 2 visible nodes, got %d", len(model.visibleNodes))
	}

	// Collapse parent
	model.toggleExpand()
	(&model).rebuildVisibleNodes()

	// Should only see parent again
	if len(model.visibleNodes) != 1 {
		t.Errorf("After collapse, should have 1 visible node, got %d", len(model.visibleNodes))
	}
}

// TestExpandAllCollapseAll verifies expand/collapse all functionality
func TestExpandAllCollapseAll(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create multi-level hierarchy
	events := []*telemetry.AgentLifecycleEvent{
		{
			AgentID:   "root",
			SessionID: "test-session",
			Tier:      "sonnet",
			Timestamp: time.Now().Unix(),
		},
		{
			AgentID:     "child1",
			SessionID:   "test-session",
			Tier:        "haiku",
			ParentAgent: "root",
			Timestamp:   time.Now().Unix(),
		},
		{
			AgentID:     "child2",
			SessionID:   "test-session",
			Tier:        "haiku",
			ParentAgent: "root",
			Timestamp:   time.Now().Unix(),
		},
		{
			AgentID:     "grandchild",
			SessionID:   "test-session",
			Tier:        "haiku",
			ParentAgent: "child1",
			Timestamp:   time.Now().Unix(),
		},
	}

	for _, event := range events {
		tree.ProcessSpawn(event)
	}

	model := New(tree)
	(&model).rebuildVisibleNodes()

	// Initially collapsed - only root visible
	if len(model.visibleNodes) != 1 {
		t.Errorf("Initially collapsed, expected 1 visible node, got %d", len(model.visibleNodes))
	}

	// Expand all
	model.ExpandAll()

	// All nodes should be visible
	if len(model.visibleNodes) != 4 {
		t.Errorf("After ExpandAll, expected 4 visible nodes, got %d", len(model.visibleNodes))
	}

	// Collapse all
	model.CollapseAll()

	// Only root visible again
	if len(model.visibleNodes) != 1 {
		t.Errorf("After CollapseAll, expected 1 visible node, got %d", len(model.visibleNodes))
	}
}

// TestGetDepth verifies depth calculation
func TestGetDepth(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create multi-level hierarchy
	events := []*telemetry.AgentLifecycleEvent{
		{
			AgentID:   "root",
			SessionID: "test-session",
			Tier:      "sonnet",
			Timestamp: time.Now().Unix(),
		},
		{
			AgentID:     "child",
			SessionID:   "test-session",
			Tier:        "haiku",
			ParentAgent: "root",
			Timestamp:   time.Now().Unix(),
		},
		{
			AgentID:     "grandchild",
			SessionID:   "test-session",
			Tier:        "haiku",
			ParentAgent: "child",
			Timestamp:   time.Now().Unix(),
		},
	}

	for _, event := range events {
		tree.ProcessSpawn(event)
	}

	model := New(tree)

	tests := []struct {
		agentID       string
		expectedDepth int
	}{
		{"root", 0},
		{"child", 1},
		{"grandchild", 2},
	}

	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			node, exists := tree.GetNode(tt.agentID)
			if !exists {
				t.Fatalf("Node %s not found", tt.agentID)
			}

			depth := model.getDepth(node)
			if depth != tt.expectedDepth {
				t.Errorf("Expected depth %d, got %d", tt.expectedDepth, depth)
			}
		})
	}
}

// TestUpdateMessage verifies AgentUpdateMsg handling
func TestUpdateMessage(t *testing.T) {
	tree1 := NewAgentTree("session-1")
	model := New(tree1)

	// Create a new tree
	tree2 := NewAgentTree("session-2")
	event := &telemetry.AgentLifecycleEvent{
		AgentID:   "new-agent",
		SessionID: "session-2",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree2.ProcessSpawn(event)

	// Send update message
	msg := AgentUpdateMsg{Tree: tree2}
	updatedModel, _ := model.Update(msg)

	// Check that tree was updated
	m := updatedModel.(Model)
	if m.tree != tree2 {
		t.Error("Tree should be updated after AgentUpdateMsg")
	}
}

// TestWindowSizeMsg verifies window size handling
func TestWindowSizeMsg(t *testing.T) {
	model := New(NewAgentTree("test-session"))

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}

	if m.height != 50 {
		t.Errorf("Expected height 50, got %d", m.height)
	}
}

// TestKeyboardInput verifies keyboard handling
func TestKeyboardInput(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Add agents
	for i := 0; i < 3; i++ {
		event := &telemetry.AgentLifecycleEvent{
			AgentID:   string(rune('a' + i)),
			SessionID: "test-session",
			Tier:      "haiku",
			Timestamp: time.Now().Unix(),
		}
		tree.ProcessSpawn(event)
	}

	model := New(tree)
	model.SetFocused(true)
	(&model).ExpandAll() // Expand all nodes
	(&model).rebuildVisibleNodes()

	tests := []struct {
		key            string
		expectedCursor int
	}{
		{"down", 1},
		{"down", 2},
		{"up", 1},
		{"j", 2},
		{"k", 1},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var msg tea.KeyMsg
			switch tt.key {
			case "down":
				msg = tea.KeyMsg{Type: tea.KeyDown}
			case "up":
				msg = tea.KeyMsg{Type: tea.KeyUp}
			case "j":
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
			case "k":
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
			}

			updatedModel, _ := model.handleKey(msg)
			m := updatedModel.(Model)

			if m.cursorPos != tt.expectedCursor {
				t.Errorf("After key %s, expected cursor %d, got %d",
					tt.key, tt.expectedCursor, m.cursorPos)
			}

			// Update model for next iteration
			model = m
		})
	}
}

// TestFocusState verifies focus state handling
func TestFocusState(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)

	// Initially not focused
	if model.focused {
		t.Error("Model should not be focused initially")
	}

	// Set focused
	model.SetFocused(true)
	if !model.focused {
		t.Error("Model should be focused after SetFocused(true)")
	}

	// Keys should work when focused
	(&model).rebuildVisibleNodes()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")}
	_, cmd := model.handleKey(msg)

	// Should handle key (no error)
	if cmd != nil {
		// This is expected for some keys (like enter)
	}

	// Set unfocused
	model.SetFocused(false)
	if model.focused {
		t.Error("Model should not be focused after SetFocused(false)")
	}
}

// TestScrolling verifies scroll offset calculation
func TestScrolling(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Add many agents to force scrolling
	for i := 0; i < 20; i++ {
		event := &telemetry.AgentLifecycleEvent{
			AgentID:   string(rune('a' + i)),
			SessionID: "test-session",
			Tier:      "haiku",
			Timestamp: time.Now().Unix(),
		}
		tree.ProcessSpawn(event)
	}

	model := New(tree)
	model.width = 80
	model.height = 10 // Small height to force scrolling
	(&model).ExpandAll() // Expand all nodes
	(&model).rebuildVisibleNodes()

	// Move down many times
	for i := 0; i < 15; i++ {
		(&model).moveDown()
	}

	// Scroll offset should have adjusted
	if model.scrollOffset == 0 {
		t.Error("Scroll offset should have adjusted when cursor moves off screen")
	}

	// Cursor should still be visible
	availableHeight := model.height - 8
	if model.cursorPos < model.scrollOffset ||
		model.cursorPos >= model.scrollOffset+availableHeight {
		t.Error("Cursor should be within visible area")
	}
}

// TestGetSelectedAgent verifies selected agent retrieval
func TestGetSelectedAgent(t *testing.T) {
	tree := NewAgentTree("test-session")

	event := &telemetry.AgentLifecycleEvent{
		AgentID:   "test-agent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(event)

	model := New(tree)
	(&model).rebuildVisibleNodes()

	// Get selected agent
	node, exists := model.GetSelectedAgent()

	if !exists {
		t.Error("Selected agent should exist")
	}

	if node.AgentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got '%s'", node.AgentID)
	}
}

// TestStatusTransitions verifies rendering after status changes
func TestStatusTransitions(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Spawn agent
	spawnEvent := &telemetry.AgentLifecycleEvent{
		AgentID:   "test-agent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(spawnEvent)

	model := New(tree)
	model.width = 80
	model.height = 20
	(&model).rebuildVisibleNodes()

	// Check spawning status
	view := model.View()
	if !strings.Contains(view, "⏳") {
		t.Error("Should show spawning icon")
	}

	// Complete agent
	success := true
	durationMs := int64(1000)
	completeEvent := &telemetry.AgentLifecycleEvent{
		AgentID:   "test-agent",
		SessionID: "test-session",
		Timestamp: time.Now().Add(time.Second).Unix(),
		Success:   &success,
		DurationMs: &durationMs,
	}
	tree.ProcessComplete(completeEvent)

	// Update model and check completed status
	model.tree = tree
	(&model).rebuildVisibleNodes()
	view = model.View()

	if !strings.Contains(view, "✓") {
		t.Error("Should show completed icon")
	}
}

// TestNewWithManager verifies model initialization with SubagentManager
func TestNewWithManager(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create a mock manager (nil is acceptable for testing)
	model := NewWithManager(tree, nil)

	if model.tree != tree {
		t.Error("Tree not set correctly")
	}

	if model.expanded == nil {
		t.Error("Expanded map not initialized")
	}

	if model.queryInput.Value() != "" {
		t.Error("Query input should be empty")
	}
}

// TestShowPickerKey verifies 's' key shows picker when manager is set
func TestShowPickerKey(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Create mock SubagentManager with test agents
	baseCfg := cli.Config{Model: "sonnet"}
	mgr := cli.NewSubagentManager(baseCfg)
	mgr.Register(cli.SubagentConfig{
		Name:        "test-agent",
		Description: "Test agent",
		Tier:        "sonnet",
	})

	model := NewWithManager(tree, mgr)
	model.SetFocused(true)
	model.width = 80
	model.height = 40

	// Press 's' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updatedModel, _ := model.handleKey(msg)
	m := updatedModel.(Model)

	if !m.showPicker {
		t.Error("Picker should be shown after 's' key")
	}

	if m.picker == nil {
		t.Error("Picker should be initialized")
	}
}

// TestShowPickerNoManager verifies 's' key does nothing without manager
func TestShowPickerNoManager(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)
	model.SetFocused(true)

	// Press 's' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updatedModel, _ := model.handleKey(msg)
	m := updatedModel.(Model)

	if m.showPicker {
		t.Error("Picker should not be shown without manager")
	}
}

// TestQueryModeEnter verifies 'q' key enters query mode for running agent
func TestQueryModeEnter(t *testing.T) {
	t.Skip("Requires mock ClaudeProcess - implementation test, not unit test")

	// This test would require spawning a real Claude process or creating
	// a mock implementation. The logic is covered by integration tests.
}

// TestQueryModeIgnoreNonRunning verifies 'q' key does nothing for non-running agent
func TestQueryModeIgnoreNonRunning(t *testing.T) {
	tree := NewAgentTree("test-session")

	// Spawn agent but don't mark it running
	event := &telemetry.AgentLifecycleEvent{
		AgentID:   "stopped-agent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(event)

	baseCfg := cli.Config{Model: "sonnet"}
	mgr := cli.NewSubagentManager(baseCfg)

	model := NewWithManager(tree, mgr)
	model.SetFocused(true)
	model.selectedID = "stopped-agent"
	(&model).rebuildVisibleNodes()

	// Press 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updatedModel, _ := model.handleKey(msg)
	m := updatedModel.(Model)

	if m.queryMode {
		t.Error("Query mode should not activate for non-running agent")
	}
}

// TestStopAgentKey verifies 'x' key triggers stop command
func TestStopAgentKey(t *testing.T) {
	tree := NewAgentTree("test-session")

	event := &telemetry.AgentLifecycleEvent{
		AgentID:   "test-agent",
		SessionID: "test-session",
		Tier:      "sonnet",
		Timestamp: time.Now().Unix(),
	}
	tree.ProcessSpawn(event)

	baseCfg := cli.Config{Model: "sonnet"}
	mgr := cli.NewSubagentManager(baseCfg)

	model := NewWithManager(tree, mgr)
	model.SetFocused(true)
	model.selectedID = "test-agent"
	(&model).rebuildVisibleNodes()

	// Press 'x' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	updatedModel, cmd := model.handleKey(msg)

	// Should return a command to stop agent
	if cmd == nil {
		t.Error("Expected command to stop agent")
	}

	m := updatedModel.(Model)
	if m.selectedID != "test-agent" {
		t.Error("Selected ID should not change")
	}
}

// TestPickerCancelMsg verifies PickerCancelMsg closes picker
func TestPickerCancelMsg(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)

	// Simulate picker being shown
	model.showPicker = true
	picker := NewPickerModel([]cli.SubagentConfig{})
	model.picker = &picker

	// Send cancel message
	msg := PickerCancelMsg{}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.showPicker {
		t.Error("Picker should be hidden after cancel")
	}

	if m.picker != nil {
		t.Error("Picker reference should be cleared")
	}
}

// TestSpawnAgentMsg verifies SpawnAgentMsg triggers spawn
func TestSpawnAgentMsg(t *testing.T) {
	tree := NewAgentTree("test-session")

	baseCfg := cli.Config{Model: "sonnet"}
	mgr := cli.NewSubagentManager(baseCfg)
	mgr.Register(cli.SubagentConfig{
		Name:        "test-agent",
		Description: "Test agent",
		Tier:        "sonnet",
	})

	model := NewWithManager(tree, mgr)
	model.showPicker = true

	// Send spawn message
	msg := SpawnAgentMsg{AgentName: "test-agent"}
	updatedModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("Expected spawn command")
	}

	m := updatedModel.(Model)
	if m.showPicker {
		t.Error("Picker should be hidden after spawn")
	}
}

// TestQueryInputSendOnEnter verifies query sends on enter
func TestQueryInputSendOnEnter(t *testing.T) {
	tree := NewAgentTree("test-session")

	baseCfg := cli.Config{Model: "sonnet"}
	mgr := cli.NewSubagentManager(baseCfg)

	model := NewWithManager(tree, mgr)
	model.SetFocused(true)
	model.queryMode = true
	model.queryAgent = "test-agent"
	model.queryInput.SetValue("test query")

	// Press enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.handleQueryInput(msg)

	// Should generate send command
	if cmd == nil {
		t.Error("Expected query send command")
	}

	m := updatedModel.(Model)
	if m.queryMode {
		t.Error("Query mode should exit after send")
	}

	if m.queryInput.Value() != "" {
		t.Error("Query input should be cleared")
	}
}

// TestQueryInputCancelOnEsc verifies query cancels on esc
func TestQueryInputCancelOnEsc(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)
	model.SetFocused(true)
	model.queryMode = true
	model.queryAgent = "test-agent"
	model.queryInput.SetValue("partial query")

	// Press esc
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.handleQueryInput(msg)

	m := updatedModel.(Model)
	if m.queryMode {
		t.Error("Query mode should exit on esc")
	}

	if m.queryAgent != "" {
		t.Error("Query agent should be cleared")
	}

	if m.queryInput.Value() != "" {
		t.Error("Query input should be cleared")
	}
}

// TestViewRendersPickerOverlay verifies View renders picker when shown
func TestViewRendersPickerOverlay(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)
	model.width = 80
	model.height = 40

	// Show picker
	model.showPicker = true
	picker := NewPickerModel([]cli.SubagentConfig{
		{Name: "test-agent", Description: "Test", Tier: "sonnet"},
	})
	picker.SetSize(76, 36)
	model.picker = &picker

	view := model.View()

	// Should contain picker content
	if !strings.Contains(view, "Select Agent to Spawn") {
		t.Error("View should contain picker title when picker is shown")
	}
}

// TestViewRendersQueryInput verifies View renders query input when in query mode
func TestViewRendersQueryInput(t *testing.T) {
	tree := NewAgentTree("test-session")
	model := New(tree)
	model.width = 80
	model.height = 40
	model.queryMode = true
	model.queryAgent = "test-agent"

	view := model.View()

	// Should contain query prompt
	if !strings.Contains(view, "Query agent") {
		t.Error("View should contain query prompt")
	}

	if !strings.Contains(view, "test-agent") {
		t.Error("View should show agent being queried")
	}
}
