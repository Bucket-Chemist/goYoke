package agents

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// AgentDetailModel
// ---------------------------------------------------------------------------

// AgentDetailModel renders the full details of a single agent. It is
// display-only: Update is a no-op and the model never emits commands.
//
// The zero value is usable and renders the empty-state placeholder.
type AgentDetailModel struct {
	agent  *state.Agent
	width  int
	height int
}

// NewAgentDetailModel returns an AgentDetailModel in the empty state.
func NewAgentDetailModel() AgentDetailModel {
	return AgentDetailModel{}
}

// SetAgent sets the agent whose details are displayed. Passing nil clears the
// detail pane and shows the empty-state placeholder. This must be called from
// the parent model's Update method — never from View.
func (m *AgentDetailModel) SetAgent(agent *state.Agent) {
	m.agent = agent
}

// HasAgent reports whether an agent is currently set. Returns false when the
// detail pane is in its empty state (showing the "Select an agent" placeholder).
func (m AgentDetailModel) HasAgent() bool {
	return m.agent != nil
}

// SetSize updates the viewport dimensions for responsive rendering.
func (m *AgentDetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model. The detail view requires no startup commands.
func (m AgentDetailModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. The detail pane is display-only; all messages
// are ignored and no commands are emitted.
func (m AgentDetailModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model. It renders the full detail view for the currently
// set agent, or a placeholder when no agent is selected.
func (m AgentDetailModel) View() string {
	if m.agent == nil {
		return config.StyleMuted.Render("Select an agent")
	}

	labelStyle := config.StyleSubtle
	valueStyle := lipgloss.NewStyle()

	var sb strings.Builder

	// Helper: render one key/value row.
	row := func(label, value string, valStyle lipgloss.Style) {
		sb.WriteString(labelStyle.Render(fmt.Sprintf("%-10s", label+":")))
		sb.WriteString("  ")
		sb.WriteString(valStyle.Render(value))
		sb.WriteByte('\n')
	}

	// Status — colored by lifecycle state (title-case, e.g. "Running").
	row("Status", capitalise(m.agent.Status.String()), m.statusValueStyle())

	// Type / Model / Tier.
	row("Type", m.agent.AgentType, valueStyle)
	row("Model", m.agent.Model, valueStyle)
	row("Tier", m.agent.Tier, valueStyle)

	// Duration — show elapsed if still running, final duration otherwise.
	row("Duration", m.formatDuration(), valueStyle)

	// Cost.
	row("Cost", fmt.Sprintf("$%.3f", m.agent.Cost), valueStyle)

	// Tokens — formatted with thousands separator.
	row("Tokens", formatTokens(m.agent.Tokens), valueStyle)

	// Activity — most recent fine-grained action.
	if m.agent.Activity != nil && m.agent.Activity.Preview != "" {
		row("Activity", m.agent.Activity.Preview, valueStyle)
	}

	// Error — only shown for error status.
	if m.agent.Status == state.StatusError && m.agent.ErrorOutput != "" {
		sb.WriteString(labelStyle.Render(fmt.Sprintf("%-10s", "Error:")) + "\n")
		wrapped := wordWrap(m.agent.ErrorOutput, m.contentWidth())
		sb.WriteString(config.StyleError.Render(wrapped))
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}

// statusValueStyle returns the appropriate lipgloss style for the status
// value string.
func (m AgentDetailModel) statusValueStyle() lipgloss.Style {
	switch m.agent.Status {
	case state.StatusRunning:
		return config.StyleWarning
	case state.StatusComplete:
		return config.StyleSuccess
	case state.StatusError, state.StatusKilled:
		return config.StyleError
	default:
		return config.StyleMuted
	}
}

// formatDuration returns a human-readable elapsed/final duration string.
func (m AgentDetailModel) formatDuration() string {
	switch m.agent.Status {
	case state.StatusComplete, state.StatusError, state.StatusKilled:
		if m.agent.Duration > 0 {
			return fmtDuration(m.agent.Duration)
		}
	case state.StatusRunning:
		if !m.agent.StartedAt.IsZero() {
			return fmtDuration(time.Since(m.agent.StartedAt))
		}
	}
	return "—"
}

// contentWidth returns the usable text width inside the detail pane.
func (m AgentDetailModel) contentWidth() int {
	w := m.width - 14 // approximate label + spacing overhead
	if w < 20 {
		return 20
	}
	return w
}

// fmtDuration formats a duration as "Xm Ys" or "Xs".
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

// formatTokens formats an integer count with thousands separators.
func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	// Build from right, inserting commas every three digits.
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

// capitalise returns s with the first Unicode letter uppercased and the rest
// lowercased (title-case for single-word status strings like "running" → "Running").
func capitalise(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// wordWrap wraps text at maxWidth rune-columns, preserving existing newlines.
func wordWrap(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	var out strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out.WriteByte('\n')
			continue
		}
		lineLen := 0
		for i, word := range words {
			wl := len([]rune(word))
			if i == 0 {
				out.WriteString(word)
				lineLen = wl
				continue
			}
			if lineLen+1+wl > maxWidth {
				out.WriteByte('\n')
				out.WriteString(word)
				lineLen = wl
			} else {
				out.WriteByte(' ')
				out.WriteString(word)
				lineLen += 1 + wl
			}
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}
