// Package tabbar implements the horizontal tab-bar component for the
// GOgent-Fortress TUI. It renders a single-row strip of named tabs and
// handles Alt+key shortcuts that jump directly to each tab.
package tabbar

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// TabDefinition associates a tab identifier with its display label.
type TabDefinition struct {
	ID    model.TabID
	Label string
}

// defaultTabs returns the canonical ordered list of tabs for the application.
func defaultTabs() []TabDefinition {
	return []TabDefinition{
		{ID: model.TabChat, Label: "Chat"},
		{ID: model.TabAgentConfig, Label: "Agent Config"},
		{ID: model.TabTeamConfig, Label: "Team Config"},
		{ID: model.TabTelemetry, Label: "Telemetry"},
	}
}

// TabBarModel is the Bubbletea model for the horizontal tab bar.
// It maintains the ordered list of tabs, the currently active tab, and the
// keybindings required to switch tabs.
//
// The zero value is not usable; use NewTabBarModel instead.
type TabBarModel struct {
	tabs      []TabDefinition
	activeTab model.TabID
	width     int
	keys      config.KeyMap
}

// NewTabBarModel returns a TabBarModel initialised with all four default tabs,
// the Chat tab active, and the supplied key map and terminal width.
func NewTabBarModel(keys config.KeyMap, width int) TabBarModel {
	return TabBarModel{
		tabs:      defaultTabs(),
		activeTab: model.TabChat,
		width:     width,
		keys:      keys,
	}
}

// Init implements tea.Model. The tab bar requires no startup commands.
func (m TabBarModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles:
//   - tea.WindowSizeMsg — updates the internal width for responsive rendering.
//   - tea.KeyMsg — switches the active tab when an Alt+key binding matches.
func (m TabBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Tab.TabChat):
			m.activeTab = model.TabChat
		case key.Matches(msg, m.keys.Tab.TabAgentConfig):
			m.activeTab = model.TabAgentConfig
		case key.Matches(msg, m.keys.Tab.TabTeamConfig):
			m.activeTab = model.TabTeamConfig
		case key.Matches(msg, m.keys.Tab.TabTelemetry):
			m.activeTab = model.TabTelemetry
		}
	}

	return m, nil
}

// View implements tea.Model. It renders a single-row horizontal strip of tab
// labels. The active tab is styled with config.StyleHighlight; inactive tabs
// use config.StyleSubtle. Tabs are separated by a vertical bar divider.
func (m TabBarModel) View() string {
	var parts []string

	for _, tab := range m.tabs {
		var label string
		if tab.ID == m.activeTab {
			label = config.StyleHighlight.Padding(0, 1).Render(tab.Label)
		} else {
			label = config.StyleSubtle.Padding(0, 1).Render(tab.Label)
		}
		parts = append(parts, label)
	}

	divider := config.StyleSubtle.Render("|")
	row := strings.Join(parts, divider)

	// Pad the row to the full terminal width so the background fills evenly.
	return lipgloss.NewStyle().
		Width(m.width).
		Render(row)
}

// ActiveTab returns the currently active tab identifier.
func (m TabBarModel) ActiveTab() model.TabID {
	return m.activeTab
}

// SetActiveTab sets the active tab to id. If id does not match any defined
// tab the active tab is unchanged.
func (m *TabBarModel) SetActiveTab(id model.TabID) {
	for _, tab := range m.tabs {
		if tab.ID == id {
			m.activeTab = id
			return
		}
	}
}

// SetWidth updates the tab bar width for responsive resizing.
func (m *TabBarModel) SetWidth(w int) {
	m.width = w
}
