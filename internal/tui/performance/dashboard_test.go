package performance

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New()

	assert.Equal(t, ViewClaude, m.activeView, "should start with Claude view")
	assert.Equal(t, "current", m.sessionFilter, "should default to current session filter")
	assert.Equal(t, 0.0, m.sessionCost, "should start with zero cost")
	assert.False(t, m.ready, "should not be ready initially")
	assert.False(t, m.showHelp, "should not show help initially")
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil command")
}

func TestWindowSizeMsg(t *testing.T) {
	m := New()
	assert.False(t, m.ready, "should not be ready before WindowSizeMsg")

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := m.Update(msg)

	m = updatedModel.(Model)
	assert.True(t, m.ready, "should be ready after WindowSizeMsg")
	assert.Equal(t, 100, m.width, "should update width")
	assert.Equal(t, 30, m.height, "should update height")
	assert.Nil(t, cmd, "should not return command")
}

func TestViewSwitchingWithNumbers(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected ViewID
	}{
		{"switch to claude", "1", ViewClaude},
		{"switch to agents", "2", ViewAgents},
		{"switch to stats", "3", ViewStats},
		{"switch to query", "4", ViewQuery},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.ready = true

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
			updatedModel, _ := m.Update(msg)

			m = updatedModel.(Model)
			assert.Equal(t, tc.expected, m.activeView)
		})
	}
}

func TestTabCycling(t *testing.T) {
	tests := []struct {
		name      string
		startView ViewID
		key       string
		expected  ViewID
	}{
		{"tab from claude", ViewClaude, "tab", ViewAgents},
		{"tab from agents", ViewAgents, "tab", ViewStats},
		{"tab from stats", ViewStats, "tab", ViewQuery},
		{"tab from query wraps", ViewQuery, "tab", ViewClaude},
		{"shift+tab from claude wraps", ViewClaude, "shift+tab", ViewQuery},
		{"shift+tab from query", ViewQuery, "shift+tab", ViewStats},
		{"shift+tab from stats", ViewStats, "shift+tab", ViewAgents},
		{"shift+tab from agents", ViewAgents, "shift+tab", ViewClaude},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.ready = true
			m.activeView = tc.startView

			var msg tea.KeyMsg
			if tc.key == "tab" {
				msg = tea.KeyMsg{Type: tea.KeyTab}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyShiftTab}
			}

			updatedModel, _ := m.Update(msg)
			m = updatedModel.(Model)

			assert.Equal(t, tc.expected, m.activeView)
		})
	}
}

func TestHelpToggle(t *testing.T) {
	m := New()
	m.ready = true

	// Toggle help on
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)
	assert.True(t, m.showHelp, "help should be visible after pressing ?")

	// Any key should dismiss help
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)
	assert.False(t, m.showHelp, "help should be dismissed after any key")
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{
			name: "quit with q",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
		},
		{
			name: "quit with ctrl+c",
			key:  tea.KeyMsg{Type: tea.KeyCtrlC},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.ready = true

			_, cmd := m.Update(tc.key)
			assert.NotNil(t, cmd, "should return quit command")
		})
	}
}

func TestViewBeforeReady(t *testing.T) {
	m := New()
	view := m.View()

	assert.Equal(t, "Initializing...", view, "should show initializing message before ready")
}

func TestRenderContent(t *testing.T) {
	tests := []struct {
		name     string
		view     ViewID
		expected string
	}{
		{
			name:     "claude view",
			view:     ViewClaude,
			expected: "Claude Conversation Panel",
		},
		{
			name:     "agents view",
			view:     ViewAgents,
			expected: "Agent Tree View",
		},
		{
			name:     "stats view",
			view:     ViewStats,
			expected: "Performance Statistics",
		},
		{
			name:     "query view",
			view:     ViewQuery,
			expected: "Query Interface",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.ready = true
			m.width = 100
			m.height = 30
			m.activeView = tc.view

			content := m.renderContent()
			assert.Contains(t, content, tc.expected)
		})
	}
}

func TestRenderBanner(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30
	m.sessionFilter = "today"

	banner := m.renderBanner()

	// Should contain tab labels
	assert.Contains(t, banner, "Claude")
	assert.Contains(t, banner, "Agents")
	assert.Contains(t, banner, "Stats")
	assert.Contains(t, banner, "Query")

	// Should contain session filter
	assert.Contains(t, banner, "Filter: today")
}

func TestRenderStatusBar(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30
	m.sessionCost = 1.25

	statusBar := m.renderStatusBar()

	// Should contain keyboard shortcuts
	assert.Contains(t, statusBar, "[Tab]")
	assert.Contains(t, statusBar, "[?]")
	assert.Contains(t, statusBar, "[q]")

	// Should contain cost
	assert.Contains(t, statusBar, "Cost: $1.25")
}

func TestRenderHelp(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	help := m.renderHelp()

	// Should contain help sections
	assert.Contains(t, help, "Navigation")
	assert.Contains(t, help, "Claude Panel")
	assert.Contains(t, help, "Agent Tree")
	assert.Contains(t, help, "General")

	// Should contain keyboard shortcuts
	assert.Contains(t, help, "[1-4]")
	assert.Contains(t, help, "[Tab]")
	assert.Contains(t, help, "[Shift+Tab]")
	assert.Contains(t, help, "[Ctrl+C]")
}

func TestFullView(t *testing.T) {
	m := New()

	// Simulate window size
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	view := m.View()

	// Should not be "Initializing..." anymore
	assert.NotEqual(t, "Initializing...", view)

	// Should contain banner elements
	assert.Contains(t, view, "Claude")
	assert.Contains(t, view, "Agents")

	// Should contain content
	assert.True(t, len(view) > 0, "view should have content")
}

func TestViewWithHelp(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30
	m.showHelp = true

	view := m.View()

	// When help is showing, should only show help overlay
	assert.Contains(t, view, "Keyboard Shortcuts")
	assert.Contains(t, view, "Navigation")
}

func TestActiveTabHighlighting(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	// Switch to each view and verify rendering
	for viewID := ViewClaude; viewID <= ViewQuery; viewID++ {
		m.activeView = viewID
		banner := m.renderBanner()

		// Banner should contain tab indicators
		require.NotEmpty(t, banner, "banner should not be empty")
	}
}

func TestResizeUpdatesLayout(t *testing.T) {
	m := New()

	// Initial size
	msg1 := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := m.Update(msg1)
	m = updatedModel.(Model)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)

	// Resize
	msg2 := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ = m.Update(msg2)
	m = updatedModel.(Model)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestSessionFilterDisplay(t *testing.T) {
	filters := []string{"current", "today", "week", "all"}

	for _, filter := range filters {
		t.Run(filter, func(t *testing.T) {
			m := New()
			m.ready = true
			m.width = 100
			m.height = 30
			m.sessionFilter = filter

			banner := m.renderBanner()
			assert.Contains(t, banner, "Filter: "+filter)
		})
	}
}

func TestMinimumDimensions(t *testing.T) {
	m := New()

	// Very small terminal
	msg := tea.WindowSizeMsg{Width: 10, Height: 5}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestHelpDismissalWithDifferentKeys(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("a")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
		{Type: tea.KeySpace},
	}

	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			m := New()
			m.ready = true
			m.showHelp = true

			updatedModel, _ := m.Update(key)
			m = updatedModel.(Model)

			assert.False(t, m.showHelp, "help should be dismissed")
		})
	}
}

func TestBannerWidthHandling(t *testing.T) {
	m := New()
	m.ready = true
	m.sessionFilter = "current"

	// Test various widths
	widths := []int{40, 80, 120, 200}

	for _, width := range widths {
		m.width = width
		m.height = 30

		banner := m.renderBanner()

		// Banner should not be empty
		assert.NotEmpty(t, banner)

		// Should handle narrow terminals gracefully
		if width < 50 {
			// Padding might be minimal but shouldn't crash
			assert.NotPanics(t, func() {
				m.renderBanner()
			})
		}
	}
}

func TestStatusBarCostFormatting(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		expected string
	}{
		{"zero cost", 0.0, "$0.00"},
		{"small cost", 0.15, "$0.15"},
		{"large cost", 123.45, "$123.45"},
		{"very large cost", 9999.99, "$9999.99"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.ready = true
			m.width = 100
			m.height = 30
			m.sessionCost = tc.cost

			statusBar := m.renderStatusBar()
			assert.Contains(t, statusBar, "Cost: "+tc.expected)
		})
	}
}

func TestContentHeightCalculation(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100

	heights := []int{5, 10, 20, 30, 50}

	for _, height := range heights {
		m.height = height

		// Should not panic with various heights
		assert.NotPanics(t, func() {
			m.renderContent()
		})
	}
}

func TestUnknownViewHandling(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	// Set invalid view ID
	m.activeView = ViewID(999)

	content := m.renderContent()
	assert.Contains(t, content, "Unknown view")
}

func TestSequentialTabPresses(t *testing.T) {
	m := New()
	m.ready = true

	// Start at Claude
	assert.Equal(t, ViewClaude, m.activeView)

	// Press Tab 4 times, should cycle back to Claude
	for i := 0; i < 4; i++ {
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	assert.Equal(t, ViewClaude, m.activeView, "should wrap around to Claude")
}

func TestSequentialShiftTabPresses(t *testing.T) {
	m := New()
	m.ready = true

	// Start at Claude
	assert.Equal(t, ViewClaude, m.activeView)

	// Press Shift+Tab 4 times, should cycle back to Claude
	for i := 0; i < 4; i++ {
		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	assert.Equal(t, ViewClaude, m.activeView, "should wrap around to Claude")
}

func TestMixedTabNavigation(t *testing.T) {
	m := New()
	m.ready = true

	// Tab forward twice
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, ViewStats, m.activeView)

	// Tab backward once
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, ViewAgents, m.activeView)
}

func TestHelpNotShownInitially(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	view := m.View()

	// Should not contain help-specific content initially
	assert.NotContains(t, view, "Press any key to close")
}

func TestAllViewsRenderWithoutPanic(t *testing.T) {
	views := []ViewID{ViewClaude, ViewAgents, ViewStats, ViewQuery}

	for _, viewID := range views {
		t.Run(fmt.Sprintf("view_%d", viewID), func(t *testing.T) {
			m := New()
			m.ready = true
			m.width = 100
			m.height = 30
			m.activeView = viewID

			assert.NotPanics(t, func() {
				view := m.View()
				assert.NotEmpty(t, view)
			})
		})
	}
}

func TestBannerContainsAllTabs(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	banner := m.renderBanner()

	// Should contain all tab labels with numbers
	assert.Contains(t, banner, "[1]")
	assert.Contains(t, banner, "[2]")
	assert.Contains(t, banner, "[3]")
	assert.Contains(t, banner, "[4]")
	assert.Contains(t, banner, "Claude")
	assert.Contains(t, banner, "Agents")
	assert.Contains(t, banner, "Stats")
	assert.Contains(t, banner, "Query")
}

func TestHelpContainsAllSections(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	help := m.renderHelp()

	sections := []string{
		"Navigation",
		"Claude Panel",
		"Agent Tree",
		"General",
	}

	for _, section := range sections {
		assert.Contains(t, help, section, "help should contain section: "+section)
	}
}

func TestIgnoreUnknownKeys(t *testing.T) {
	m := New()
	m.ready = true
	initialView := m.activeView

	// Press various unknown keys
	unknownKeys := []string{"x", "y", "z", "5", "6", "0"}

	for _, key := range unknownKeys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	// View should not have changed
	assert.Equal(t, initialView, m.activeView, "unknown keys should not change view")
}

func TestStatusBarContainsAllShortcuts(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	statusBar := m.renderStatusBar()

	shortcuts := []string{"[Tab]", "[?]", "[q]"}

	for _, shortcut := range shortcuts {
		assert.Contains(t, statusBar, shortcut)
	}
}

func TestViewComponentsAreSeparate(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 100
	m.height = 30

	view := m.View()

	// View should contain elements from all three components
	// Banner content
	assert.True(t, strings.Contains(view, "Claude") || strings.Contains(view, "Agents"))

	// Status bar content
	assert.Contains(t, view, "[Tab]")

	// Cost display
	assert.Contains(t, view, "Cost:")
}
