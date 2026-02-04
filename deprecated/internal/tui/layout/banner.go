package layout

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View represents the active view in the TUI
type View int

const (
	ViewClaude View = iota
	ViewAgents
	ViewStats
	ViewQuery
)

// BannerModel represents the persistent top banner with navigation tabs
type BannerModel struct {
	activeView View
	sessionID  string
	cost       float64
	width      int
}

// NewBannerModel creates a new banner model with the given session ID
func NewBannerModel(sessionID string) BannerModel {
	return BannerModel{
		activeView: ViewClaude,
		sessionID:  sessionID,
		cost:       0.0,
		width:      0,
	}
}

// SetActiveView updates the currently active view
func (m *BannerModel) SetActiveView(view View) {
	m.activeView = view
}

// SetCost updates the running cost total
func (m *BannerModel) SetCost(cost float64) {
	m.cost = cost
}

// SetWidth updates the banner width
func (m *BannerModel) SetWidth(width int) {
	m.width = width
}

// View renders the banner as a string
func (m BannerModel) View() string {
	// Guard against zero or very narrow terminals
	if m.width < 20 {
		return "" // Don't render if too small
	}

	tabs := []struct {
		key   string
		label string
		view  View
	}{
		{"1", "Claude", ViewClaude},
		{"2", "Agents", ViewAgents},
		{"3", "Stats", ViewStats},
		{"4", "Query", ViewQuery},
	}

	var tabStrings []string
	for _, tab := range tabs {
		label := fmt.Sprintf("[%s] %s", tab.key, tab.label)
		if tab.view == m.activeView {
			tabStrings = append(tabStrings, activeTabStyle.Render(label))
		} else {
			tabStrings = append(tabStrings, inactiveTabStyle.Render(label))
		}
	}

	tabSection := strings.Join(tabStrings, "  ")

	// Session info (right-aligned)
	sessionInfo := fmt.Sprintf("Session: %s | Cost: $%.2f",
		truncateSessionID(m.sessionID),
		m.cost,
	)
	sessionRendered := sessionInfoStyle.Render(sessionInfo)

	// Calculate padding for right alignment safely
	// Account for bannerStyle padding (1 on each side = 2 total)
	availableWidth := m.width - 2
	usedWidth := lipgloss.Width(tabSection) + lipgloss.Width(sessionRendered)
	padding := availableWidth - usedWidth

	// Handle narrow terminals - truncate session info if needed
	if padding < 1 {
		// Terminal too narrow for full content - progressively truncate
		if m.width < 40 {
			// Very narrow - no session info
			sessionRendered = ""
		} else {
			// Narrow - minimal session info (just ID)
			sessionInfo = truncateSessionID(m.sessionID)
			sessionRendered = sessionInfoStyle.Render(sessionInfo)
		}
		// Recalculate padding with truncated content
		usedWidth = lipgloss.Width(tabSection) + lipgloss.Width(sessionRendered)
		padding = availableWidth - usedWidth
		if padding < 1 {
			padding = 1 // Minimum spacing
		}
	}

	content := tabSection + strings.Repeat(" ", padding) + sessionRendered

	return bannerStyle.Width(m.width).Render(content)
}

// truncateSessionID truncates session ID to 8 characters max
func truncateSessionID(id string) string {
	if id == "" {
		return "--------" // Placeholder for empty ID
	}
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// Styles for banner rendering
var (
	bannerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan")).
			Background(lipgloss.Color("236"))

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Background(lipgloss.Color("236"))

	sessionInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Background(lipgloss.Color("236"))
)
