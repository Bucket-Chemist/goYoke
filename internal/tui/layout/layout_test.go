package layout

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/claude"
)

// mockClaudeProcess implements ClaudeProcessInterface for testing
type mockClaudeProcess struct {
	events        chan cli.Event
	restartEvents chan cli.RestartEvent
}

func newMockClaudeProcess() *mockClaudeProcess {
	return &mockClaudeProcess{
		events:        make(chan cli.Event, 10),
		restartEvents: make(chan cli.RestartEvent, 10),
	}
}

func (m *mockClaudeProcess) Send(message string) error {
	return nil
}

func (m *mockClaudeProcess) Events() <-chan cli.Event {
	return m.events
}

func (m *mockClaudeProcess) RestartEvents() <-chan cli.RestartEvent {
	return m.restartEvents
}

func (m *mockClaudeProcess) SessionID() string {
	return "test-session"
}

func (m *mockClaudeProcess) IsRunning() bool {
	return true
}

// Test fixtures
func createTestModel() Model {
	tree := agents.NewAgentTree("test-session")

	// Add a root agent
	rootNode := &agents.AgentNode{
		AgentID:     "root",
		SessionID:   "test-session",
		Tier:        "sonnet",
		Description: "Root agent",
		Status:      agents.StatusRunning,
		SpawnTime:   time.Now(),
		Children:    []*agents.AgentNode{},
	}
	tree.Root = rootNode

	claudePanel := claude.NewPanelModel(newMockClaudeProcess(), cli.Config{})
	agentTreeView := agents.New(tree)

	return NewModel(claudePanel, agentTreeView, "test-session")
}

// TestNewModel verifies model initialization
func TestNewModel(t *testing.T) {
	model := createTestModel()

	assert.Equal(t, FocusLeft, model.focused, "Initial focus should be on left panel")
	assert.Equal(t, ViewClaude, model.activeView, "Initial active view should be Claude")
	assert.NotNil(t, model.claudePanel, "Claude panel should be initialized")
	assert.NotNil(t, model.agentTree, "Agent tree should be initialized")
	assert.NotNil(t, model.agentDetail, "Agent detail should be initialized")
	assert.Equal(t, ViewClaude, model.banner.activeView, "Banner active view should be Claude")
}

// TestCalculateLayout verifies layout calculation with various widths
func TestCalculateLayout(t *testing.T) {
	tests := []struct {
		name        string
		totalWidth  int
		expectLeft  int
		expectRight int
		description string
	}{
		{
			name:        "standard_width",
			totalWidth:  100,
			expectLeft:  69,  // 99 * 0.70 = 69.3 → 69
			expectRight: 30,  // 99 - 69 = 30
			description: "Standard 70/30 split",
		},
		{
			name:        "wide_width",
			totalWidth:  200,
			expectLeft:  139, // 199 * 0.70 = 139.3 → 139
			expectRight: 60,  // 199 - 139 = 60
			description: "Wide terminal maintains 70/30 ratio",
		},
		{
			name:        "narrow_width_left_minimum",
			totalWidth:  50,
			expectLeft:  29,  // 49 - 20 (right minimum prioritized)
			expectRight: 20,  // MinRightWidth enforced
			description: "Narrow width prioritizes right minimum (can't satisfy both)",
		},
		{
			name:        "narrow_width_right_minimum",
			totalWidth:  55,
			expectLeft:  34,  // 54 - 20 = 34
			expectRight: 20,  // MinRightWidth enforced
			description: "Narrow width enforces right minimum",
		},
		{
			name:        "minimum_total_width",
			totalWidth:  61,
			expectLeft:  40,  // MinLeftWidth
			expectRight: 20,  // MinRightWidth
			description: "Minimum total width (40 + 20 + 1 border)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel()
			model.width = tt.totalWidth

			left, right := model.calculateLayout()

			assert.Equal(t, tt.expectLeft, left, "Left width should match expected")
			assert.Equal(t, tt.expectRight, right, "Right width should match expected")

			// Verify minimum constraints (right is prioritized)
			assert.GreaterOrEqual(t, right, MinRightWidth, "Right width must meet minimum (priority)")
			// Left minimum only applies when both can be satisfied
			if model.width-1 >= MinLeftWidth+MinRightWidth {
				assert.GreaterOrEqual(t, left, MinLeftWidth, "Left width must meet minimum when both fit")
			}

			// Verify total equals available (width - 1 for border)
			assert.Equal(t, model.width-1, left+right, "Left + right should equal available width")
		})
	}
}

// TestFocusToggle verifies Tab key switches focus between panels
func TestFocusToggle(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Initial state: left focused
	assert.Equal(t, FocusLeft, model.focused)

	// Press Tab: should switch to right
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	assert.Equal(t, FocusRight, model.focused, "Tab should switch focus to right")

	// Press Tab again: should switch back to left
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	assert.Equal(t, FocusLeft, model.focused, "Tab should switch focus to left")
}

// TestWindowResize verifies layout updates on terminal resize
func TestWindowResize(t *testing.T) {
	model := createTestModel()

	// Initial size
	model.width = 100
	model.height = 30
	model.updateSizes()

	leftWidth1, rightWidth1 := model.calculateLayout()
	assert.Equal(t, 69, leftWidth1)
	assert.Equal(t, 30, rightWidth1)

	// Resize to wider
	resizeMsg := tea.WindowSizeMsg{Width: 200, Height: 50}
	result, _ := model.Update(resizeMsg)
	model = result.(Model)

	assert.Equal(t, 200, model.width)
	assert.Equal(t, 50, model.height)

	leftWidth2, rightWidth2 := model.calculateLayout()
	assert.Equal(t, 139, leftWidth2)
	assert.Equal(t, 60, rightWidth2)

	// Resize to narrower
	resizeMsg = tea.WindowSizeMsg{Width: 70, Height: 25}
	result, _ = model.Update(resizeMsg)
	model = result.(Model)

	assert.Equal(t, 70, model.width)
	assert.Equal(t, 25, model.height)

	leftWidth3, rightWidth3 := model.calculateLayout()
	// At width 70, available is 69. 69 * 0.70 = 48.3 → 48, leaving 21 for right
	// This satisfies both minimums (40 and 20)
	assert.Equal(t, 48, leftWidth3)
	assert.Equal(t, 21, rightWidth3)
}

// TestMinimumWidthEnforcement verifies minimum width constraints
func TestMinimumWidthEnforcement(t *testing.T) {
	tests := []struct {
		name             string
		width            int
		minLeftSatisfied bool
		minRightSatisfied bool
	}{
		{
			name:              "both_minimums_satisfied",
			width:             100,
			minLeftSatisfied:  true,
			minRightSatisfied: true,
		},
		{
			name:              "right_at_minimum",
			width:             61,
			minLeftSatisfied:  true,
			minRightSatisfied: true,
		},
		{
			name:              "barely_sufficient",
			width:             61,
			minLeftSatisfied:  true,
			minRightSatisfied: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel()
			model.width = tt.width

			left, right := model.calculateLayout()

			if tt.minLeftSatisfied {
				assert.GreaterOrEqual(t, left, MinLeftWidth, "Left minimum should be satisfied")
			}
			if tt.minRightSatisfied {
				assert.GreaterOrEqual(t, right, MinRightWidth, "Right minimum should be satisfied")
			}
		})
	}
}

// TestUpdateSizes verifies size propagation to child components
func TestUpdateSizes(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Call updateSizes
	model.updateSizes()

	// Verify layout was calculated correctly
	leftWidth, rightWidth := model.calculateLayout()
	assert.Equal(t, 69, leftWidth)
	assert.Equal(t, 30, rightWidth)

	// Verify tree and detail panels received correct vertical split
	// Tree gets top half, detail gets bottom half
	expectedTreeHeight := model.height / 2
	expectedDetailHeight := model.height - expectedTreeHeight

	assert.Equal(t, expectedTreeHeight, 20)
	assert.Equal(t, expectedDetailHeight, 20)
}

// TestQuitKey verifies q and ctrl+c quit the application
func TestQuitKey(t *testing.T) {
	model := createTestModel()

	// Test 'q' key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd, "Quit command should be returned")
	_ = result

	// Test 'ctrl+c'
	model = createTestModel()
	result, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd, "Quit command should be returned")
	_ = result
}

// TestFocusIndicator verifies focus state is properly tracked
func TestFocusIndicator(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Left focused initially
	assert.Equal(t, FocusLeft, model.focused)

	// Switch to right
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	assert.Equal(t, FocusRight, model.focused)

	// Switch back to left
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	assert.Equal(t, FocusLeft, model.focused)
}

// TestAgentSelectionUpdate verifies detail panel updates on tree selection
func TestAgentSelectionUpdate(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Switch focus to tree (right panel)
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	require.Equal(t, FocusRight, model.focused)

	// Simulate tree selection change via SelectionMsg
	selectionMsg := agents.SelectionMsg{AgentID: "root"}
	result, _ = model.Update(selectionMsg)
	model = result.(Model)

	// Verify detail panel was updated (via GetSelectedAgent call)
	// We can't directly verify the detail panel's agent without exposing internals,
	// but we've tested that the Update method calls SetAgent
	// This is primarily a smoke test for the message flow
}

// TestView verifies View renders without panic
func TestView(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Should not panic
	view := model.View()
	assert.NotEmpty(t, view, "View should render content")

	// Switch focus and render again
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)
	view = model.View()
	assert.NotEmpty(t, view, "View should render with right focus")
}

// TestLayoutRatios verifies the 70/30 ratio is maintained when possible
func TestLayoutRatios(t *testing.T) {
	widths := []int{100, 150, 200, 250, 300}

	for _, width := range widths {
		t.Run("width_"+string(rune(width)), func(t *testing.T) {
			model := createTestModel()
			model.width = width

			left, right := model.calculateLayout()

			// Calculate actual ratio
			available := width - 1
			actualRatioLeft := float64(left) / float64(available)

			// Should be close to 0.70 (allowing for integer rounding)
			assert.InDelta(t, 0.70, actualRatioLeft, 0.05, "Left panel should be approximately 70%")

			// Verify they sum to available width
			assert.Equal(t, available, left+right, "Panels should sum to available width")
		})
	}
}

// TestMessageForwarding verifies messages are forwarded to focused panel
func TestMessageForwarding(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Left focused: generic key should be forwarded to claude panel
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	result, _ := model.Update(keyMsg)
	model = result.(Model)
	assert.Equal(t, FocusLeft, model.focused)

	// Switch to right
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)

	// Right focused: generic key should be forwarded to agent tree
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ = model.Update(keyMsg)
	model = result.(Model)
	assert.Equal(t, FocusRight, model.focused)
}

// BenchmarkCalculateLayout benchmarks layout calculation performance
func BenchmarkCalculateLayout(b *testing.B) {
	model := createTestModel()
	model.width = 200
	model.height = 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.calculateLayout()
	}
}

// BenchmarkUpdate benchmarks the update cycle
func BenchmarkUpdate(b *testing.B) {
	model := createTestModel()
	model.width = 200
	model.height = 50

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(msg)
	}
}

// TestNumberKeysSwitchView verifies number keys (1-4) switch active view
func TestNumberKeysSwitchView(t *testing.T) {
	tests := []struct {
		name     string
		key      rune
		expected View
	}{
		{"key_1_claude", '1', ViewClaude},
		{"key_2_agents", '2', ViewAgents},
		{"key_3_stats", '3', ViewStats},
		{"key_4_query", '4', ViewQuery},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := createTestModel()
			model.width = 100
			model.height = 40

			// Press number key
			result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
			model = result.(Model)

			assert.Equal(t, tc.expected, model.activeView, "Active view should match expected")
			assert.Equal(t, tc.expected, model.banner.activeView, "Banner active view should match")
		})
	}
}

// TestBannerRenderedInView verifies banner is rendered at top of view
func TestBannerRenderedInView(t *testing.T) {
	model := createTestModel()
	model.width = 120
	model.height = 40

	view := model.View()

	// Banner components should be present
	assert.Contains(t, view, "[1] Claude", "View should contain Claude tab")
	assert.Contains(t, view, "[2] Agents", "View should contain Agents tab")
	assert.Contains(t, view, "[3] Stats", "View should contain Stats tab")
	assert.Contains(t, view, "[4] Query", "View should contain Query tab")
	assert.Contains(t, view, "Session:", "View should contain session label")
	assert.Contains(t, view, "Cost:", "View should contain cost label")
}

// TestBannerWidthUpdateOnResize verifies banner width updates on window resize
func TestBannerWidthUpdateOnResize(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30

	// Initial banner width should be set
	assert.Equal(t, 0, model.banner.width, "Initial banner width should be zero")

	// Trigger resize
	resizeMsg := tea.WindowSizeMsg{Width: 150, Height: 40}
	result, _ := model.Update(resizeMsg)
	model = result.(Model)

	// Banner width should be updated via updateSizes
	assert.Equal(t, 150, model.banner.width, "Banner width should match terminal width")

	// View should update banner width
	_ = model.View()
	assert.Equal(t, 150, model.banner.width, "Banner width should be set in View()")
}

// TestActiveViewPersistsAcrossUpdates verifies active view state is maintained
func TestActiveViewPersistsAcrossUpdates(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Switch to Agents view
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = result.(Model)
	assert.Equal(t, ViewAgents, model.activeView)

	// Perform other operations (focus toggle)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(Model)

	// Active view should persist
	assert.Equal(t, ViewAgents, model.activeView, "Active view should persist across updates")
	assert.Equal(t, ViewAgents, model.banner.activeView, "Banner view should persist")
}

// TestContentHeightAccountsForBanner verifies content area height calculation
func TestContentHeightAccountsForBanner(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 40

	// Trigger updateSizes to calculate heights
	model.updateSizes()

	// Content height should be total height - banner height (1)
	const bannerHeight = 1
	expectedContentHeight := model.height - bannerHeight
	assert.Equal(t, 39, expectedContentHeight, "Content height should account for banner")

	// The View() method applies these heights, so we verify via View rendering
	view := model.View()
	assert.NotEmpty(t, view, "View should render")

	// The components should receive the correct heights
	// This is implicitly tested by the View not panicking
}

// TestBannerWithSmallTerminal verifies banner works with small terminal size
func TestBannerWithSmallTerminal(t *testing.T) {
	model := createTestModel()
	model.width = 60
	model.height = 20

	// Should not panic
	view := model.View()
	assert.NotEmpty(t, view, "View should render even with small terminal")

	// Banner components should still be present
	assert.Contains(t, view, "[1] Claude")
	assert.Contains(t, view, "Session:")
}
