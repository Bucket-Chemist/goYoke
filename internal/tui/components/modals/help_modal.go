// Package modals implements the modal dialog system for the goYoke TUI.
// This file implements HelpModal: a full-screen, scrollable keyboard shortcut
// reference overlay activated by alt+h.
package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// HelpClosedMsg
// ---------------------------------------------------------------------------

// HelpClosedMsg is sent when the user dismisses the help overlay with
// Escape or 'q'.
type HelpClosedMsg struct{}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	helpBorderFrame = 2
	helpHeaderRows  = 3
	helpFooterRows  = 2
	helpMaxWidth    = 60
	helpMaxHeight   = 35
)

// ---------------------------------------------------------------------------
// Lipgloss styles (created once at package level)
// ---------------------------------------------------------------------------

var (
	helpBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorPrimary)

	helpTitleStyle   = config.StyleTitle.Copy()
	helpDividerStyle = config.StyleSubtle.Copy()
	helpFooterStyle  = config.StyleSubtle.Copy()

	helpSectionStyle = lipgloss.NewStyle().Bold(true).
				Foreground(config.ColorPrimary)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(config.ColorPrimary)

	helpDescStyle = config.StyleSubtle.Copy()
)

// ---------------------------------------------------------------------------
// HelpModal
// ---------------------------------------------------------------------------

// HelpModal is a full-screen keyboard shortcut reference overlay.
//
// Design contract:
//   - Content is built in Show(), NEVER in View.
//   - View is a pure, fast function of model state.
//   - Esc and 'q' close the modal and emit HelpClosedMsg.
//
// The zero value is not usable; use NewHelpModal instead.
type HelpModal struct {
	// rendered is the pre-built help text set in Show().
	rendered string

	// viewport handles scroll position and keyboard-driven scrolling.
	viewport viewport.Model

	// width and height are the outer terminal dimensions (including border).
	width  int
	height int

	// active controls whether the modal is currently shown.
	active bool

	// km is the key map used to generate help content.
	km config.KeyMap
}

// NewHelpModal returns a HelpModal in its initial (inactive) state.
func NewHelpModal() HelpModal {
	return HelpModal{
		viewport: viewport.New(0, 0),
		km:       config.DefaultKeyMap(),
	}
}

// ---------------------------------------------------------------------------
// Mutators
// ---------------------------------------------------------------------------

// Show activates the modal and (re)builds the help content.
func (m *HelpModal) Show() {
	m.rendered = buildHelpContent(m.km)
	m.rebuildViewport(m.width, m.height)
	m.active = true
}

// Hide deactivates the modal.
func (m *HelpModal) Hide() {
	m.active = false
}

// IsActive reports whether the modal is currently shown.
func (m HelpModal) IsActive() bool {
	return m.active
}

// SetSize updates the terminal dimensions and resizes the internal viewport.
func (m *HelpModal) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.rebuildViewport(w, h)
}

// rebuildViewport recomputes viewport dimensions and reloads content.
// The modal is capped to helpMaxWidth x helpMaxHeight so it floats as a
// centered box rather than filling the entire terminal.
func (m *HelpModal) rebuildViewport(w, h int) {
	innerW := w - helpBorderFrame
	if innerW > helpMaxWidth {
		innerW = helpMaxWidth
	}
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - helpBorderFrame - helpHeaderRows - helpFooterRows
	if innerH > helpMaxHeight {
		innerH = helpMaxHeight
	}
	if innerH < 1 {
		innerH = 1
	}

	m.viewport.Width = innerW
	m.viewport.Height = innerH

	if m.rendered != "" {
		m.viewport.SetContent(m.rendered)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update processes tea.Msg events. When the modal is not active all messages
// are ignored. Esc and 'q' close the modal and emit HelpClosedMsg; all other
// keys are forwarded to the viewport for scrolling.
func (m HelpModal) Update(msg tea.Msg) (HelpModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.active = false
			return m, func() tea.Msg { return HelpClosedMsg{} }
		}
	}

	// Forward to viewport for scroll keys (up, down, pgup, pgdn, etc.).
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the modal as a full-terminal overlay. It returns "" when the
// modal is not active.
func (m HelpModal) View() string {
	if !m.active {
		return ""
	}
	return renderHelpView(m)
}

// renderHelpView builds the full modal string.
func renderHelpView(m HelpModal) string {
	innerW := m.viewport.Width
	if innerW < 1 {
		innerW = m.width - helpBorderFrame
		if innerW < 1 {
			innerW = 40
		}
	}

	title := helpTitleStyle.Render("Keyboard Shortcuts")
	divider := helpDividerStyle.Render(helpDividerLine(innerW))

	scrollPct := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPct = int(m.viewport.ScrollPercent() * 100)
	}
	vpView := m.viewport.View()

	footer := helpFooterStyle.Render(
		fmt.Sprintf("esc/q: close  ↑/↓/pgup/pgdn: scroll  %3d%%", scrollPct),
	)

	inner := strings.Join([]string{title, divider, vpView, footer}, "\n")

	return helpBorderStyle.
		Width(innerW).
		Render(inner)
}

// helpDividerLine returns a horizontal rule string capped at a sensible width.
func helpDividerLine(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 80 {
		width = 80
	}
	return strings.Repeat("\u2500", width)
}

// ---------------------------------------------------------------------------
// Content builder
// ---------------------------------------------------------------------------

// buildHelpContent generates the formatted keybinding reference from km.
// Keys and descriptions are extracted via key.Binding.Help().
func buildHelpContent(km config.KeyMap) string {
	var sb strings.Builder

	writeSection := func(title string, rows [][2]string) {
		sb.WriteString(helpSectionStyle.Render(title))
		sb.WriteString("\n")
		sb.WriteString(helpDividerStyle.Render(strings.Repeat("\u2500", 38)))
		sb.WriteString("\n")
		for _, row := range rows {
			keyCol := helpKeyStyle.Render(fmt.Sprintf("%-14s", row[0]))
			descCol := helpDescStyle.Render(row[1])
			sb.WriteString(fmt.Sprintf("  %s  %s\n", keyCol, descCol))
		}
		sb.WriteString("\n")
	}

	// Global section
	g := km.Global
	writeSection("Global", [][2]string{
		{g.ToggleFocus.Help().Key, g.ToggleFocus.Help().Desc},
		{g.ReverseToggleFocus.Help().Key, g.ReverseToggleFocus.Help().Desc},
		{g.CycleProvider.Help().Key, g.CycleProvider.Help().Desc},
		{g.CycleRightPanel.Help().Key, g.CycleRightPanel.Help().Desc},
		{g.CyclePermMode.Help().Key, g.CyclePermMode.Help().Desc},
		{g.Interrupt.Help().Key, g.Interrupt.Help().Desc},
		{g.ForceQuit.Help().Key, g.ForceQuit.Help().Desc},
		{g.ClearScreen.Help().Key, g.ClearScreen.Help().Desc},
		{g.ToggleTaskBoard.Help().Key, g.ToggleTaskBoard.Help().Desc},
		{g.ViewPlan.Help().Key, g.ViewPlan.Help().Desc},
		{g.Search.Help().Key, g.Search.Help().Desc},
		{g.ChangeCWD.Help().Key, g.ChangeCWD.Help().Desc},
		{g.ShowHelp.Help().Key, g.ShowHelp.Help().Desc},
		{g.ToggleMouse.Help().Key, g.ToggleMouse.Help().Desc},
	})

	// Tabs section
	t := km.Tab
	writeSection("Tabs", [][2]string{
		{t.TabChat.Help().Key, t.TabChat.Help().Desc},
		{t.TabAgentConfig.Help().Key, t.TabAgentConfig.Help().Desc},
		{t.TabTeamConfig.Help().Key, t.TabTeamConfig.Help().Desc},
		{t.TabTelemetry.Help().Key, t.TabTelemetry.Help().Desc},
	})

	// Claude Panel section
	c := km.Claude
	writeSection("Claude Panel", [][2]string{
		{c.Submit.Help().Key, c.Submit.Help().Desc},
		{c.HistoryPrev.Help().Key, c.HistoryPrev.Help().Desc},
		{c.HistoryNext.Help().Key, c.HistoryNext.Help().Desc},
		{c.ToggleToolExpansion.Help().Key, c.ToggleToolExpansion.Help().Desc},
		{c.CycleExpansion.Help().Key, c.CycleExpansion.Help().Desc},
		{c.Search.Help().Key, c.Search.Help().Desc},
		{c.SearchNext.Help().Key, c.SearchNext.Help().Desc},
		{c.SearchPrev.Help().Key, c.SearchPrev.Help().Desc},
		{c.CopyLastResponse.Help().Key, c.CopyLastResponse.Help().Desc},
	})

	// Agent Panel section
	a := km.Agent
	writeSection("Agent Panel", [][2]string{
		{a.AgentUp.Help().Key, a.AgentUp.Help().Desc},
		{a.AgentDown.Help().Key, a.AgentDown.Help().Desc},
		{a.AgentExpand.Help().Key, a.AgentExpand.Help().Desc},
		{a.AgentKill.Help().Key, a.AgentKill.Help().Desc},
		{a.CycleDensity.Help().Key, a.CycleDensity.Help().Desc},
	})

	// Modal section
	mo := km.Modal
	writeSection("Modal", [][2]string{
		{mo.ModalUp.Help().Key, mo.ModalUp.Help().Desc},
		{mo.ModalDown.Help().Key, mo.ModalDown.Help().Desc},
		{mo.ModalSelect.Help().Key, mo.ModalSelect.Help().Desc},
		{mo.ModalCancel.Help().Key, mo.ModalCancel.Help().Desc},
	})

	return sb.String()
}
