package performance

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - adaptive for light/dark terminals
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	// Banner styles
	bannerStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a2e")).
		Foreground(lipgloss.Color("#eaeaea")).
		Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(accent)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(subtle)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#16213e")).
		Foreground(lipgloss.Color("#a0a0a0")).
		Padding(0, 1)

	// Help overlay
	helpStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		Padding(1, 2).
		Width(50)

	// Content area
	contentStyle = lipgloss.NewStyle().
		Padding(1, 2)
)
