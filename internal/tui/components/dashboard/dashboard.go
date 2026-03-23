// Package dashboard implements the session dashboard panel for the
// GOgent-Fortress TUI. It displays a summary of key session metrics in a
// compact key-value layout.
//
// The component is display-only: Update is a no-op and state is set
// exclusively via SetData. No I/O is performed in View.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// DashboardModel
// ---------------------------------------------------------------------------

// DashboardModel is the display-only model for the session dashboard panel.
// It renders key session metrics: cost, tokens, messages, agents, teams, and
// elapsed duration.
//
// The zero value is usable and renders a zeroed-out dashboard.
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
}

// NewDashboardModel returns a DashboardModel with sensible zero defaults.
func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

// SetSize updates the rendering dimensions. Call this on every
// tea.WindowSizeMsg so the component is aware of available space.
func (m *DashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

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

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the dashboard as a headed key-value list. It is a pure
// function of the model state — no I/O is performed here.
func (m DashboardModel) View() string {
	var sb strings.Builder

	// Header.
	sb.WriteString(config.StyleTitle.Render("Session Dashboard"))
	sb.WriteByte('\n')
	sb.WriteString(config.StyleSubtle.Render(divider(m.width)))
	sb.WriteByte('\n')

	// Row helper: fixed-width label on the left, value on the right.
	row := func(label, value string) {
		labelStr := config.StyleHighlight.Render(fmt.Sprintf("%-12s", label))
		valueStr := config.StyleSubtle.Render(value)
		sb.WriteString(labelStr)
		sb.WriteString(valueStr)
		sb.WriteByte('\n')
	}

	// Cost row — use the existing FormatCost helper from state.
	row("Cost:", formatCost(m.sessionCost))

	// Tokens — formatted with thousands separators.
	row("Tokens:", formatInt64(m.totalTokens))

	// Message count.
	row("Messages:", fmt.Sprintf("%d", m.messageCount))

	// Agent count.
	row("Agents:", fmt.Sprintf("%d", m.agentCount))

	// Team count.
	row("Teams:", fmt.Sprintf("%d", m.teamCount))

	// Duration — computed from sessionStart; show "—" when zero.
	row("Duration:", m.formatDuration())

	return strings.TrimRight(sb.String(), "\n")
}

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
