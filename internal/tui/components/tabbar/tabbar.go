// Package tabbar implements the horizontal tab-bar component for the
// GOgent-Fortress TUI. It renders a single-row strip of named tabs and
// handles Alt+key shortcuts that jump directly to each tab.
package tabbar

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// flashDuration is the total duration of the accent flash animation.
const flashDuration = 300 * time.Millisecond

// tabFlashTickMsg is an internal tick message used to clear the flash state
// after flashDuration has elapsed.  It is unexported so that only the tabbar
// package schedules it, keeping the animation fully self-contained.
type tabFlashTickMsg struct{}

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

	// Flash animation state (TUI-061).
	// flashTab is the zero-based index of the tab currently flashing, or -1
	// when no flash is active.  flashStart records when the flash began so
	// that View can determine whether the 300 ms window has passed.
	flashTab   int
	flashStart time.Time
}

// NewTabBarModel returns a TabBarModel initialised with all four default tabs,
// the Chat tab active, and the supplied key map and terminal width.
func NewTabBarModel(keys config.KeyMap, width int) TabBarModel {
	return TabBarModel{
		tabs:      defaultTabs(),
		activeTab: model.TabChat,
		width:     width,
		keys:      keys,
		flashTab:  -1,
	}
}

// Init implements tea.Model. The tab bar requires no startup commands.
func (m TabBarModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles:
//   - tea.WindowSizeMsg — updates the internal width for responsive rendering.
//   - tea.KeyMsg — switches the active tab when an Alt+key binding matches.
//   - tabFlashTickMsg — clears the flash state after flashDuration elapses.
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

	case tabFlashTickMsg:
		if m.flashTab != -1 && time.Since(m.flashStart) >= flashDuration {
			m.flashTab = -1
		}
	}

	return m, nil
}

// HandleMsg processes a tea.Msg for internal tab bar state that must mutate
// the receiver in place.  It is part of the tabBarWidget interface and is
// called by AppModel.Update for message types the tab bar owns.
//
// Currently handles:
//   - model.TabFlashMsg — activates the 300 ms accent flash on the given tab.
//   - tabFlashTickMsg   — clears the flash when the timer fires.
func (m *TabBarModel) HandleMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case model.TabFlashMsg:
		m.flashTab = msg.TabIndex
		m.flashStart = time.Now()
		return m.scheduleFlashTick()

	case tabFlashTickMsg:
		if m.flashTab != -1 && time.Since(m.flashStart) >= flashDuration {
			m.flashTab = -1
		}
	}
	return nil
}

// scheduleFlashTick returns a Cmd that fires a tabFlashTickMsg after the
// remaining flash duration has elapsed.  It guarantees at least one tick
// even when called very late in the flash window.
func (m *TabBarModel) scheduleFlashTick() tea.Cmd {
	remaining := flashDuration - time.Since(m.flashStart)
	if remaining <= 0 {
		remaining = time.Millisecond
	}
	return tea.Tick(remaining, func(_ time.Time) tea.Msg {
		return tabFlashTickMsg{}
	})
}

// flashStyle is the accent style applied to the active tab during a flash.
// It uses the accent color (magenta) as a foreground with bold text to make
// the flash visually distinct from the normal highlight style.
var flashStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(config.ColorAccent).
	Background(lipgloss.Color("5"))

// View implements tea.Model. It renders a single-row horizontal strip of tab
// labels. The active tab is styled with config.StyleHighlight; inactive tabs
// use config.StyleSubtle. Tabs are separated by a vertical bar divider.
//
// When a flash is active (flashTab != -1 and within flashDuration), the
// flashing tab is rendered with flashStyle instead of the normal active style.
func (m TabBarModel) View() string {
	var parts []string

	isFlashing := m.flashTab != -1 && time.Since(m.flashStart) < flashDuration

	for i, tab := range m.tabs {
		var label string
		switch {
		case tab.ID == m.activeTab && isFlashing && i == m.flashTab:
			label = flashStyle.Padding(0, 1).Render(tab.Label)
		case tab.ID == m.activeTab:
			label = config.StyleHighlight.Padding(0, 1).Render(tab.Label)
		default:
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

// IsFlashing reports whether a flash animation is currently active.
// The flash is active when flashTab != -1 and the flash window has not elapsed.
func (m TabBarModel) IsFlashing() bool {
	return m.flashTab != -1 && time.Since(m.flashStart) < flashDuration
}

// FlashTab returns the zero-based index of the tab currently flashing, or -1
// when no flash is active.
func (m TabBarModel) FlashTab() int {
	return m.flashTab
}

// FlashStart returns the time at which the current flash began.
// The value is meaningful only when IsFlashing() returns true.
func (m TabBarModel) FlashStart() time.Time {
	return m.flashStart
}

// BackdateFlashStart shifts flashStart backwards by d, making elapsed time
// appear larger.  This is used in tests to simulate the flash window expiring
// without actually sleeping.
func (m *TabBarModel) BackdateFlashStart(d time.Duration) {
	m.flashStart = m.flashStart.Add(-d)
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
