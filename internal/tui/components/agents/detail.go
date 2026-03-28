package agents

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Detail Section
// ---------------------------------------------------------------------------

// DetailSection represents one collapsible section in the agent detail panel.
type DetailSection struct {
	Title    string
	Expanded bool
	render   func(a *state.Agent, w int) string // renders section content
	visible  func(a *state.Agent) bool          // returns false to hide section entirely
}

// ---------------------------------------------------------------------------
// AgentDetailModel
// ---------------------------------------------------------------------------

// AgentDetailModel renders the full details of a single agent with collapsible
// sections. Navigation: Up/Down move between sections, Enter toggles collapse.
// Content is scrollable via a viewport.
type AgentDetailModel struct {
	agent       *state.Agent
	sections    []DetailSection
	selectedIdx int
	width       int
	height      int
	vp          viewport.Model
	focused     bool
}

// NewAgentDetailModel returns an AgentDetailModel with default sections.
func NewAgentDetailModel() AgentDetailModel {
	m := AgentDetailModel{
		vp: viewport.New(40, 10),
	}
	m.sections = m.defaultSections()
	return m
}

// defaultSections builds the standard section list.
func (m AgentDetailModel) defaultSections() []DetailSection {
	return []DetailSection{
		{
			Title:    "Overview",
			Expanded: true,
			render:   renderOverview,
			visible:  alwaysVisible,
		},
		{
			Title:    "Context",
			Expanded: false,
			render:   renderContext,
			visible:  alwaysVisible,
		},
		{
			Title:    "Prompt",
			Expanded: false,
			render:   renderPrompt,
			visible:  hasPrompt,
		},
		{
			Title:    "Activity",
			Expanded: true,
			render:   renderActivity,
			visible:  isRunningOrHasActivity,
		},
		{
			Title:    "Error",
			Expanded: true, // errors always start expanded for immediate visibility
			render:   renderError,
			visible:  hasError,
		},
	}
}

// SetAgent sets the agent whose details are displayed.
func (m *AgentDetailModel) SetAgent(agent *state.Agent) {
	m.agent = agent
	m.syncViewport()
}

// HasAgent reports whether an agent is currently set.
func (m AgentDetailModel) HasAgent() bool {
	return m.agent != nil
}

// SetSize updates the viewport dimensions.
func (m *AgentDetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.vp.Width = width
	m.vp.Height = height
	m.syncViewport()
}

// SetFocused enables or disables keyboard input for section navigation.
func (m *AgentDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model.
func (m AgentDetailModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Handles keyboard navigation for sections.
func (m AgentDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused || m.agent == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		visible := m.visibleSections()
		if len(visible) == 0 {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			m.syncViewport()
		case "down", "j":
			if m.selectedIdx < len(visible)-1 {
				m.selectedIdx++
			}
			m.syncViewport()
		case "enter", " ":
			idx := m.resolveVisibleIndex(m.selectedIdx)
			if idx >= 0 && idx < len(m.sections) {
				m.sections[idx].Expanded = !m.sections[idx].Expanded
				m.syncViewport()
			}
		case "backspace", "left", "h":
			// Return focus to the agent tree. Esc is reserved for global
			// interrupt (CLI process SIGINT).
			m.focused = false
			return m, func() tea.Msg { return AgentTreeFocusMsg{} }
		}
	}

	// Let viewport handle scroll (pgup/pgdn/mousewheel).
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m AgentDetailModel) View() string {
	if m.agent == nil {
		return config.StyleMuted.Render("Select an agent")
	}
	return m.vp.View()
}

// ---------------------------------------------------------------------------
// Internal rendering
// ---------------------------------------------------------------------------

// syncViewport rebuilds the full content and sets it on the viewport.
func (m *AgentDetailModel) syncViewport() {
	if m.agent == nil {
		m.vp.SetContent("")
		return
	}

	var sb strings.Builder
	visIdx := 0
	for i, sec := range m.sections {
		if sec.visible != nil && !sec.visible(m.agent) {
			continue
		}

		// Section header with collapse indicator.
		indicator := "▸"
		if sec.Expanded {
			indicator = "▼"
		}
		header := fmt.Sprintf("%s %s", indicator, sec.Title)
		if m.focused && visIdx == m.selectedIdx {
			sb.WriteString(config.StyleHighlight.Render(header))
		} else {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
		}
		sb.WriteByte('\n')

		// Section content (only if expanded).
		if sec.Expanded {
			content := m.sections[i].render(m.agent, m.contentWidth())
			if content != "" {
				sb.WriteString(content)
				sb.WriteByte('\n')
			}
		}

		visIdx++
	}

	m.vp.SetContent(strings.TrimRight(sb.String(), "\n"))
}

// visibleSections returns indices of sections visible for the current agent.
func (m AgentDetailModel) visibleSections() []int {
	var indices []int
	for i, sec := range m.sections {
		if sec.visible == nil || sec.visible(m.agent) {
			indices = append(indices, i)
		}
	}
	return indices
}

// resolveVisibleIndex maps a visual selection index to the actual section index.
func (m AgentDetailModel) resolveVisibleIndex(visIdx int) int {
	visible := m.visibleSections()
	if visIdx >= 0 && visIdx < len(visible) {
		return visible[visIdx]
	}
	return -1
}

// contentWidth returns the usable text width inside the detail pane.
func (m AgentDetailModel) contentWidth() int {
	w := m.width - 4 // indent margin
	if w < 20 {
		return 20
	}
	return w
}

// ---------------------------------------------------------------------------
// Section visibility predicates
// ---------------------------------------------------------------------------

func alwaysVisible(_ *state.Agent) bool { return true }

func hasPrompt(a *state.Agent) bool {
	return a.Prompt != ""
}

func hasActivity(a *state.Agent) bool {
	return a.Activity != nil || len(a.RecentActivity) > 0
}

func isRunningOrHasActivity(a *state.Agent) bool {
	return a.Status == state.StatusRunning || a.Activity != nil || len(a.RecentActivity) > 0
}

func hasError(a *state.Agent) bool {
	return a.Status == state.StatusError && a.ErrorOutput != ""
}

// ---------------------------------------------------------------------------
// Section renderers
// ---------------------------------------------------------------------------

func renderOverview(a *state.Agent, _ int) string {
	labelStyle := config.StyleSubtle
	valueStyle := lipgloss.NewStyle()

	var sb strings.Builder
	row := func(label, value string, valStyle lipgloss.Style) {
		sb.WriteString("  ")
		sb.WriteString(labelStyle.Render(fmt.Sprintf("%-10s", label+":")))
		sb.WriteString(" ")
		sb.WriteString(valStyle.Render(value))
		sb.WriteByte('\n')
	}

	// Status — colored.
	statusStyle := statusStyleFor(a.Status)
	row("Status", capitalise(a.Status.String()), statusStyle)
	row("Type", a.AgentType, valueStyle)
	row("Model", a.Model, valueStyle)
	row("Tier", a.Tier, valueStyle)
	row("Duration", formatAgentDuration(a), valueStyle)
	row("Cost", fmt.Sprintf("$%.3f", a.Cost), valueStyle)
	row("Tokens", formatTokens(a.Tokens), valueStyle)

	return strings.TrimRight(sb.String(), "\n")
}

func renderContext(a *state.Agent, w int) string {
	var sb strings.Builder
	if len(a.Conventions) > 0 {
		sb.WriteString("  Conventions:\n")
		for _, c := range a.Conventions {
			sb.WriteString(fmt.Sprintf("    • %s\n", c))
		}
	} else {
		sb.WriteString("  No conventions loaded\n")
	}
	if a.Description != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", a.Description))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderPrompt(a *state.Agent, w int) string {
	if a.Prompt == "" {
		return "  (no prompt available)"
	}
	// Indent each line of the prompt.
	lines := strings.Split(a.Prompt, "\n")
	var sb strings.Builder
	for _, line := range lines {
		if len(line) > w {
			line = line[:w-1] + "…"
		}
		sb.WriteString("  ")
		sb.WriteString(config.StyleMuted.Render(line))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderActivity(a *state.Agent, _ int) string {
	if len(a.RecentActivity) == 0 && a.Activity == nil {
		if a.Status == state.StatusRunning {
			return config.StyleMuted.Render("  ⏳ Subprocess running...")
		}
		return "  (idle)"
	}

	// When we have a rolling buffer, render each entry.
	if len(a.RecentActivity) > 0 {
		var sb strings.Builder
		for i, act := range a.RecentActivity {
			isLast := i == len(a.RecentActivity)-1
			label := act.Target
			if label == "" {
				label = act.Type
			}
			if act.Preview != "" {
				label += " " + act.Preview
			}
			if isLast && a.Status == state.StatusRunning {
				sb.WriteString(fmt.Sprintf("  ⏳ [%s]\n", label))
			} else {
				sb.WriteString(config.StyleMuted.Render(fmt.Sprintf("  ✓ [%s]", label)) + "\n")
			}
		}
		return strings.TrimRight(sb.String(), "\n")
	}

	// Fallback: single Activity set (e.g. from SDK agent sync, no rolling buffer).
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  [%s] %s", a.Activity.Type, a.Activity.Target))
	if a.Activity.Preview != "" {
		sb.WriteString(fmt.Sprintf(" — %s", a.Activity.Preview))
	}
	return sb.String()
}

func renderError(a *state.Agent, w int) string {
	if a.ErrorOutput == "" {
		return ""
	}
	wrapped := wordWrap(a.ErrorOutput, w-2)
	var sb strings.Builder
	for _, line := range strings.Split(wrapped, "\n") {
		sb.WriteString("  ")
		sb.WriteString(config.StyleError.Render(line))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func statusStyleFor(s state.AgentStatus) lipgloss.Style {
	switch s {
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

func formatAgentDuration(a *state.Agent) string {
	switch a.Status {
	case state.StatusComplete, state.StatusError, state.StatusKilled:
		if a.Duration > 0 {
			return fmtDuration(a.Duration)
		}
	case state.StatusRunning:
		if !a.StartedAt.IsZero() {
			return fmtDuration(time.Since(a.StartedAt))
		}
	}
	return "—"
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

// capitalise returns s with the first letter uppercased.
func capitalise(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// wordWrap wraps text at maxWidth, preserving existing newlines.
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
