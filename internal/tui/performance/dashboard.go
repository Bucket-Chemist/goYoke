package performance

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewID represents the different dashboard views
type ViewID int

const (
	ViewClaude ViewID = iota
	ViewAgents
	ViewStats
	ViewQuery
)

// Model is the main dashboard model
type Model struct {
	width  int
	height int
	ready  bool

	activeView ViewID
	showHelp   bool

	// Session filter options: "current", "today", "week", "all"
	sessionFilter string

	// Placeholder for session cost (will be calculated in later tickets)
	sessionCost float64

	// Sub-components will be added in later tickets:
	// claudePanel  claude.Model     // GOgent-118
	// agentTree    agents.Model     // GOgent-116
	// statsPanel   stats.Model      // Future
	// queryPanel   query.Model      // Future
}

// New creates a new dashboard model with default values
func New() Model {
	return Model{
		activeView:    ViewClaude,
		sessionFilter: "current",
		sessionCost:   0.0,
	}
}

// Init initializes the dashboard (satisfies tea.Model interface)
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If help is showing, any key dismisses it
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true
		return m, nil

	case "1":
		m.activeView = ViewClaude

	case "2":
		m.activeView = ViewAgents

	case "3":
		m.activeView = ViewStats

	case "4":
		m.activeView = ViewQuery

	case "tab":
		m.activeView = (m.activeView + 1) % 4

	case "shift+tab":
		m.activeView = (m.activeView + 3) % 4
	}

	return m, nil
}

// View renders the dashboard
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.showHelp {
		return m.renderHelp()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderBanner(),
		m.renderContent(),
		m.renderStatusBar(),
	)
}

// renderBanner renders the top banner with tabs and session info
func (m Model) renderBanner() string {
	tabs := []struct {
		id    ViewID
		label string
		key   string
	}{
		{ViewClaude, "Claude", "1"},
		{ViewAgents, "Agents", "2"},
		{ViewStats, "Stats", "3"},
		{ViewQuery, "Query", "4"},
	}

	var rendered []string
	for _, tab := range tabs {
		label := fmt.Sprintf("[%s] %s", tab.key, tab.label)
		if tab.id == m.activeView {
			rendered = append(rendered, activeTabStyle.Render(label))
		} else {
			rendered = append(rendered, inactiveTabStyle.Render(label))
		}
	}

	tabBar := strings.Join(rendered, " │ ")

	// Right side: session info
	sessionInfo := fmt.Sprintf("Filter: %s", m.sessionFilter)

	// Calculate padding to push session info to the right
	leftWidth := lipgloss.Width(tabBar)
	rightWidth := lipgloss.Width(sessionInfo)
	padding := m.width - leftWidth - rightWidth - 4
	if padding < 1 {
		padding = 1
	}

	return bannerStyle.Width(m.width).Render(
		tabBar + strings.Repeat(" ", padding) + sessionInfo,
	)
}

// renderContent renders the main content area based on active view
func (m Model) renderContent() string {
	var content string

	switch m.activeView {
	case ViewClaude:
		content = "Claude Conversation Panel\n\n" +
			"This view will display the Claude conversation interface.\n" +
			"(Implementation: GOgent-118)"

	case ViewAgents:
		content = "Agent Tree View\n\n" +
			"This view will display the hierarchical agent tree.\n" +
			"(Implementation: GOgent-116)"

	case ViewStats:
		content = "Performance Statistics\n\n" +
			"This view will display session metrics and performance data.\n" +
			"(Future implementation)"

	case ViewQuery:
		content = "Query Interface\n\n" +
			"This view will provide tools for querying archived sessions.\n" +
			"(Future implementation)"

	default:
		content = "Unknown view"
	}

	// Calculate available height for content
	// Total height - banner (1) - status bar (1) - padding
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	return contentStyle.
		Width(m.width - 4).
		Height(contentHeight).
		Render(content)
}

// renderStatusBar renders the bottom status bar with shortcuts and cost
func (m Model) renderStatusBar() string {
	left := "[Tab] Switch View  [?] Help  [q] Quit"
	right := fmt.Sprintf("Cost: $%.2f", m.sessionCost)

	// Calculate padding
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := m.width - leftWidth - rightWidth - 4
	if padding < 1 {
		padding = 1
	}

	return statusBarStyle.Width(m.width).Render(
		left + strings.Repeat(" ", padding) + right,
	)
}

// renderHelp renders the centered help overlay
func (m Model) renderHelp() string {
	help := `GOgent TUI - Keyboard Shortcuts
═══════════════════════════════

Navigation
  [1-4]       Switch to view by number
  [Tab]       Next view
  [Shift+Tab] Previous view
  [?]         Toggle this help

Claude Panel
  [Enter]     Send message
  [Esc]       Cancel input
  [Ctrl+L]    Clear conversation

Agent Tree
  [↑/↓]       Navigate agents
  [Enter]     View agent details
  [q]         Query selected agent

General
  [Ctrl+C]    Quit
  [r]         Refresh data

Press any key to close...`

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		helpStyle.Render(help),
	)
}
