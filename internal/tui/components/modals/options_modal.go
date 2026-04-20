// Package modals implements the modal dialog system for the goYoke TUI.
// This file implements OptionsViewModal: a full-screen options viewer that
// activates via alt+o when the options drawer has content or an active modal.
//
// Two modes:
//   - View mode: scrollable plain-text content; Esc/q closes.
//   - Interactive mode: option-selection list; Enter selects, Esc cancels.
package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/drawer"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// OptionsViewClosedMsg
// ---------------------------------------------------------------------------

// OptionsViewClosedMsg is sent when the user dismisses the options viewer in
// view mode with Escape or 'q'.
type OptionsViewClosedMsg struct{}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// optsBorderFrame accounts for the left+right border chars (1 each).
	optsBorderFrame = 2
	// optsHeaderRows is the number of rows consumed by the title header.
	optsHeaderRows = 3
	// optsFooterRows is the number of rows consumed by the hint footer.
	optsFooterRows = 2
)

// ---------------------------------------------------------------------------
// Lipgloss styles (created once at package level)
// ---------------------------------------------------------------------------

var (
	optsBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorPrimary)

	optsTitleStyle = config.StyleTitle.Copy()

	optsDividerStyle = config.StyleSubtle.Copy()

	optsFooterStyle = config.StyleSubtle.Copy()

	optsSelectedStyle = lipgloss.NewStyle().
				Foreground(config.ColorPrimary).
				Bold(true)
)

// ---------------------------------------------------------------------------
// OptionsViewModal
// ---------------------------------------------------------------------------

// OptionsViewModal is a full-screen options viewer overlay. It operates in one
// of two modes depending on whether the options drawer has an active modal:
//
//   - View mode: renders plain-text drawer content in a scrollable viewport.
//     Esc or 'q' close the modal and emit OptionsViewClosedMsg.
//
//   - Interactive mode: renders a selectable option list. Up/Down navigate,
//     Enter emits drawer.ModalResponseMsg with the selection, Esc emits
//     drawer.ModalResponseMsg with Cancelled:true.
//
// Design contract:
//   - View is a pure, fast function of model state.
//   - All state mutations happen in Show* and Update.
//
// The zero value is not usable; use NewOptionsViewModal instead.
type OptionsViewModal struct {
	// active controls whether the modal is currently shown.
	active bool

	// interactive indicates the modal is in option-selection mode.
	interactive bool

	// --- view mode fields ---

	// content is the plain-text drawer content set by ShowContent.
	content string

	// viewport handles scroll position for view mode.
	viewport viewport.Model

	// --- interactive mode fields ---

	// requestID is the bridge request ID to include in ModalResponseMsg.
	requestID string

	// message is the modal prompt text.
	message string

	// options is the list of selectable option labels.
	options []string

	// selectedIdx is the currently highlighted option index.
	selectedIdx int

	// --- layout ---

	// width and height are the outer terminal dimensions (including border).
	width  int
	height int
}

// NewOptionsViewModal returns an OptionsViewModal in its initial (inactive) state.
func NewOptionsViewModal() OptionsViewModal {
	return OptionsViewModal{
		viewport: viewport.New(0, 0),
	}
}

// ---------------------------------------------------------------------------
// Mutators
// ---------------------------------------------------------------------------

// IsActive reports whether the modal is currently shown.
func (m OptionsViewModal) IsActive() bool { return m.active }

// ShowContent activates the modal in view mode with the given plain-text content.
func (m *OptionsViewModal) ShowContent(content string, width, height int) {
	m.interactive = false
	m.content = content
	m.requestID = ""
	m.message = ""
	m.options = nil
	m.selectedIdx = 0
	m.width = width
	m.height = height
	m.rebuildViewport(width, height)
	m.viewport.SetContent(content)
	m.active = true
}

// ShowInteractive activates the modal in interactive mode with the given
// option-selection state from the options drawer.
func (m *OptionsViewModal) ShowInteractive(requestID, message string, options []string, selectedIdx, width, height int) {
	m.interactive = true
	m.content = ""
	m.requestID = requestID
	m.message = message
	m.options = options
	m.selectedIdx = selectedIdx
	m.width = width
	m.height = height
	m.rebuildViewport(width, height)
	m.viewport.SetContent(m.formatInteractiveContent())
	m.active = true
}

// SetSize updates the terminal dimensions and resizes the internal viewport.
func (m *OptionsViewModal) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.rebuildViewport(w, h)
}

// rebuildViewport recomputes the viewport dimensions.
func (m *OptionsViewModal) rebuildViewport(w, h int) {
	innerW := w - optsBorderFrame
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - optsBorderFrame - optsHeaderRows - optsFooterRows
	if innerH < 1 {
		innerH = 1
	}
	m.viewport.Width = innerW
	m.viewport.Height = innerH
}

// formatInteractiveContent renders the prompt and option list with a cursor
// indicator on the currently selected option.
func (m OptionsViewModal) formatInteractiveContent() string {
	var sb strings.Builder
	sb.WriteString(m.message)
	sb.WriteString("\n\n")
	for i, opt := range m.options {
		if i == m.selectedIdx {
			sb.WriteString(optsSelectedStyle.Render("  ▸ "+opt) + "\n")
		} else {
			sb.WriteString("    " + opt + "\n")
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update processes tea.Msg events. When the modal is not active all messages
// are ignored. Behaviour differs by mode:
//
//   - View mode: Esc and 'q' close the modal and emit OptionsViewClosedMsg.
//     All other keys are forwarded to the viewport for scrolling.
//
//   - Interactive mode: Up/Down/j/k navigate options, Enter confirms the
//     selection (emits drawer.ModalResponseMsg), Esc cancels (emits
//     drawer.ModalResponseMsg with Cancelled:true).
func (m OptionsViewModal) Update(msg tea.Msg) (OptionsViewModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.interactive {
			return m.handleInteractiveKey(keyMsg)
		}
		return m.handleViewKey(keyMsg)
	}

	// Forward non-key messages to viewport (scroll position etc.).
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m OptionsViewModal) handleViewKey(msg tea.KeyMsg) (OptionsViewModal, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.active = false
		return m, func() tea.Msg { return OptionsViewClosedMsg{} }
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m OptionsViewModal) handleInteractiveKey(msg tea.KeyMsg) (OptionsViewModal, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.viewport.SetContent(m.formatInteractiveContent())
			m.scrollToSelected()
		}
		return m, nil
	case "down", "j":
		if m.selectedIdx < len(m.options)-1 {
			m.selectedIdx++
			m.viewport.SetContent(m.formatInteractiveContent())
			m.scrollToSelected()
		}
		return m, nil
	case "enter":
		var selected string
		if m.selectedIdx < len(m.options) {
			selected = m.options[m.selectedIdx]
		}
		reqID := m.requestID
		m.active = false
		return m, func() tea.Msg {
			return drawer.ModalResponseMsg{RequestID: reqID, Value: selected}
		}
	case "esc":
		reqID := m.requestID
		m.active = false
		return m, func() tea.Msg {
			return drawer.ModalResponseMsg{RequestID: reqID, Value: "", Cancelled: true}
		}
	}
	return m, nil
}

// scrollToSelected adjusts the viewport offset so the currently selected
// option line is visible.
func (m *OptionsViewModal) scrollToSelected() {
	if len(m.options) == 0 {
		return
	}
	msgLines := strings.Count(m.message, "\n") + 1
	targetLine := msgLines + 1 + m.selectedIdx // +1 for blank line after message
	if targetLine < m.viewport.YOffset {
		m.viewport.YOffset = targetLine
	} else if targetLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = targetLine - m.viewport.Height + 1
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the modal as a full-terminal overlay. Returns "" when inactive.
// View is a pure function — no I/O, no rendering.
func (m OptionsViewModal) View() string {
	if !m.active {
		return ""
	}
	return renderOptionsView(m)
}

// renderOptionsView builds the full modal string.
func renderOptionsView(m OptionsViewModal) string {
	innerW := m.viewport.Width
	if innerW < 1 {
		innerW = m.width - optsBorderFrame
		if innerW < 1 {
			innerW = 40
		}
	}

	// Header.
	title := optsTitleStyle.Render("Options")
	divider := optsDividerStyle.Render(optsDividerLine(innerW))

	// Scroll percent for view mode.
	scrollPct := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPct = int(m.viewport.ScrollPercent() * 100)
	}
	vpView := m.viewport.View()

	// Footer differs by mode.
	var footer string
	if m.interactive {
		footer = optsFooterStyle.Render("↑/↓: navigate  Enter: select  Esc: cancel")
	} else {
		footer = optsFooterStyle.Render(
			fmt.Sprintf("↑/↓/pgup/pgdn: scroll  q/esc: close  %3d%%", scrollPct),
		)
	}

	inner := strings.Join([]string{title, divider, vpView, footer}, "\n")

	return optsBorderStyle.
		Width(innerW).
		Render(inner)
}

// optsDividerLine returns a horizontal rule string of the given width.
func optsDividerLine(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 80 {
		width = 80
	}
	return strings.Repeat("\u2500", width)
}
