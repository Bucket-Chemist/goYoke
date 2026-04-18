// Package dashboard implements the session dashboard panel for the
// goYoke TUI. It displays a summary of key session metrics grouped
// into collapsible sections with keyboard navigation.
//
// The component supports expand/collapse per section via Up/Down/Enter when
// focused. State is set exclusively via SetData; no I/O is performed in View.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Section and metric types
// ---------------------------------------------------------------------------

// DashboardSection represents a single collapsible section in the dashboard.
type DashboardSection struct {
	// Title is the header text rendered for the section.
	Title string
	// Expanded controls whether the metrics are visible below the header.
	Expanded bool
}

// metricRow is a single key-value display row within a section.
type metricRow struct {
	label string
	value string
}

// ---------------------------------------------------------------------------
// DashboardModel
// ---------------------------------------------------------------------------

// DashboardModel is the interactive model for the session dashboard panel.
// It renders key session metrics grouped into collapsible sections and
// supports keyboard navigation when focused.
//
// The zero value is not usable; use NewDashboardModel instead.
type DashboardModel struct {
	width  int
	height int

	// Data fields — set via SetData.
	sessionCost  float64
	totalTokens  int64
	messageCount int
	agentCount   int
	teamCount    int
	sessionStart time.Time

	// Section collapse/expand state.
	sections []DashboardSection
	// cursor is the index of the currently selected section header.
	cursor int
	// focused controls whether keyboard events are processed.
	focused bool
}

// Section index constants for readability.
const (
	sectionSession     = 0
	sectionCostTokens  = 1
	sectionAgents      = 2
	sectionPerformance = 3
)

// NewDashboardModel returns a DashboardModel with four sections. Section 0
// (Session Overview) starts expanded; the remaining sections start collapsed.
func NewDashboardModel() DashboardModel {
	return DashboardModel{
		sections: []DashboardSection{
			{Title: "Session Overview", Expanded: true},
			{Title: "Cost & Tokens", Expanded: false},
			{Title: "Agent Activity", Expanded: false},
			{Title: "Performance", Expanded: false},
		},
		cursor: 0,
	}
}

// ---------------------------------------------------------------------------
// Public setters
// ---------------------------------------------------------------------------

// SetSize updates the rendering dimensions. Call this on every
// tea.WindowSizeMsg so the component is aware of available space.
func (m *DashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetTier satisfies the dashboardWidget interface. Tier-specific rendering
// adaptations are reserved for a future ticket; this is a no-op placeholder.
func (m *DashboardModel) SetTier(_ model.LayoutTier) {}

// SetData updates all dashboard metrics in a single call. sessionStart should
// be the wall-clock time the session began; pass a zero time.Time when the
// session start is unknown.
func (m *DashboardModel) SetData(cost float64, tokens int64, msgs, agents, teams int, start time.Time) {
	m.sessionCost = cost
	m.totalTokens = tokens
	m.messageCount = msgs
	m.agentCount = agents
	m.teamCount = teams
	m.sessionStart = start
}

// SetFocused sets whether the panel processes keyboard events.
func (m *DashboardModel) SetFocused(focused bool) {
	m.focused = focused
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update processes a tea.Msg. When focused it handles Up/Down navigation and
// Enter to toggle expand/collapse on the selected section. The receiver is
// mutated in place; only the Cmd is returned so the interface stays simple.
func (m *DashboardModel) Update(msg tea.Msg) tea.Cmd {
	if !m.focused {
		return nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch keyMsg.String() {
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "enter", " ":
		m.toggleSection()
	}

	return nil
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

// moveUp moves the cursor to the previous section, clamping at 0.
func (m *DashboardModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// moveDown moves the cursor to the next section, clamping at the last.
func (m *DashboardModel) moveDown() {
	if m.cursor < len(m.sections)-1 {
		m.cursor++
	}
}

// toggleSection flips the Expanded flag of the currently selected section.
func (m *DashboardModel) toggleSection() {
	if m.cursor >= 0 && m.cursor < len(m.sections) {
		m.sections[m.cursor].Expanded = !m.sections[m.cursor].Expanded
	}
}

// ---------------------------------------------------------------------------
// Metric helpers
// ---------------------------------------------------------------------------

// sectionMetrics returns all metric rows for the given section index.
// Placeholder rows are included for data not yet wired up.
func (m DashboardModel) sectionMetrics(idx int) []metricRow {
	switch idx {
	case sectionSession:
		return []metricRow{
			{label: "Session ID:", value: "—"},
			{label: "Duration:", value: m.formatDuration()},
			{label: "Model:", value: "—"},
			{label: "Provider:", value: "—"},
		}

	case sectionCostTokens:
		return []metricRow{
			{label: "Total Cost:", value: formatCost(m.sessionCost)},
			{label: "Tokens:", value: formatInt64(m.totalTokens)},
			{label: "Messages:", value: fmt.Sprintf("%d", m.messageCount)},
			{label: "Per-Agent:", value: "—"},
			{label: "Context %:", value: "—"},
		}

	case sectionAgents:
		return []metricRow{
			{label: "Agents:", value: fmt.Sprintf("%d", m.agentCount)},
			{label: "Teams:", value: fmt.Sprintf("%d", m.teamCount)},
			{label: "Active:", value: "—"},
			{label: "Completed:", value: "—"},
			{label: "Errors:", value: "—"},
		}

	case sectionPerformance:
		return []metricRow{
			{label: "Events/sec:", value: "—"},
			{label: "Modal Latency:", value: "—"},
			{label: "Render Time:", value: "—"},
		}
	}

	return nil
}

// summaryMetric returns the single most-informative metric for a collapsed
// section header. It is shown inline on the header row.
func (m DashboardModel) summaryMetric(idx int) string {
	switch idx {
	case sectionSession:
		return m.formatDuration()
	case sectionCostTokens:
		return formatCost(m.sessionCost)
	case sectionAgents:
		return fmt.Sprintf("%d agents", m.agentCount)
	case sectionPerformance:
		return "—"
	}
	return "—"
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the dashboard as a set of collapsible sections. It is a pure
// function of the model state — no I/O is performed here.
func (m DashboardModel) View() string {
	var sb strings.Builder

	icons := config.DefaultTheme().Icons()

	// Global header.
	sb.WriteString(config.StyleTitle.Render("Session Dashboard"))
	sb.WriteByte('\n')
	sb.WriteString(config.StyleSubtle.Render(divider(m.width)))
	sb.WriteByte('\n')

	for i, sec := range m.sections {
		selected := i == m.cursor

		// Choose indicator based on expand state.
		indicator := icons.Pending // collapsed: ○ / .
		if sec.Expanded {
			indicator = icons.Running // expanded: ▶ / >
		}

		// Build header line.
		headerText := indicator + " " + sec.Title

		var headerStyle lipgloss.Style
		if selected && m.focused {
			headerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorAccent).
				Reverse(true)
		} else {
			headerStyle = config.StyleTitle
		}

		// When collapsed, append a summary metric after the title.
		if !sec.Expanded {
			summary := m.summaryMetric(i)
			headerLine := headerStyle.Render(headerText)
			summaryStr := config.StyleSubtle.Render("  " + summary)
			sb.WriteString(headerLine)
			sb.WriteString(summaryStr)
		} else {
			sb.WriteString(headerStyle.Render(headerText))
		}
		sb.WriteByte('\n')

		// Render metric rows only when expanded.
		if sec.Expanded {
			rows := m.sectionMetrics(i)
			for _, row := range rows {
				labelStr := config.StyleHighlight.Render(fmt.Sprintf("  %-14s", row.label))
				valueStr := config.StyleSubtle.Render(row.value)
				sb.WriteString(labelStr)
				sb.WriteString(valueStr)
				sb.WriteByte('\n')
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// formatDuration returns a human-readable elapsed time string.
func (m DashboardModel) formatDuration() string {
	if m.sessionStart.IsZero() {
		return "\u2014" // em dash
	}
	d := time.Since(m.sessionStart).Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

// divider returns a horizontal rule string fitting the given width. It falls
// back to a fixed-width rule when width is zero.
func divider(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 40 {
		width = 40
	}
	return strings.Repeat("\u2500", width) // box-drawing horizontal line
}

// formatCost formats a float64 cost as a USD string.
func formatCost(cost float64) string {
	if cost == 0 {
		return "$0.00"
	}
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// formatInt64 formats an int64 with thousands separators.
func formatInt64(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}
