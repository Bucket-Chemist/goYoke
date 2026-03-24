// Package modals implements the modal dialog system for the GOgent-Fortress TUI.
// This file implements PlanViewModal: a full-screen, scrollable, Glamour-rendered
// plan viewer that activates via alt+v when the right panel is in RPMPlanPreview mode.
package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// PlanViewClosedMsg
// ---------------------------------------------------------------------------

// PlanViewClosedMsg is sent when the user dismisses the plan viewer with
// Escape or 'q'. AppModel.Update handles this message to clear any overlay
// state (no extra cleanup is required at present).
type PlanViewClosedMsg struct{}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// planBorderFrame accounts for the left+right border chars (1 each).
	planBorderFrame = 2
	// planHeaderRows is the number of rows consumed by the title header.
	planHeaderRows = 3
	// planFooterRows is the number of rows consumed by the hint footer.
	planFooterRows = 2
	// planContentMargin is subtracted from the outer width when pre-rendering
	// markdown so Glamour word-wrap respects the border padding.
	planContentMargin = 4
)

// ---------------------------------------------------------------------------
// Lipgloss styles (created once at package level)
// ---------------------------------------------------------------------------

var (
	planBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorPrimary)

	planTitleStyle = config.StyleTitle.Copy()

	planDividerStyle = config.StyleSubtle.Copy()

	planFooterStyle = config.StyleSubtle.Copy()
)

// ---------------------------------------------------------------------------
// PlanViewModal
// ---------------------------------------------------------------------------

// PlanViewModal is a full-screen Glamour-rendered plan viewer overlay. It is
// displayed on top of the normal layout when the user presses alt+v while the
// right panel is in RPMPlanPreview mode.
//
// Design contract:
//   - Glamour rendering is performed in SetContent, NEVER in View.
//   - View is a pure, fast function of model state.
//   - Esc and 'q' close the modal and emit PlanViewClosedMsg.
//
// The zero value is not usable; use NewPlanViewModal instead.
type PlanViewModal struct {
	// content is the raw markdown set by the caller.
	content string

	// rendered is the Glamour-processed terminal text stored at SetContent
	// time. View() reads this field without performing any I/O.
	rendered string

	// viewport handles scroll position and keyboard-driven scrolling.
	viewport viewport.Model

	// width and height are the outer terminal dimensions (including border).
	width  int
	height int

	// active controls whether the modal is currently shown.
	active bool
}

// NewPlanViewModal returns a PlanViewModal in its initial (inactive) state.
func NewPlanViewModal() PlanViewModal {
	return PlanViewModal{
		viewport: viewport.New(0, 0),
	}
}

// ---------------------------------------------------------------------------
// Mutators
// ---------------------------------------------------------------------------

// Show activates the modal so it is rendered by the next View() call.
func (m *PlanViewModal) Show() {
	m.active = true
}

// Hide deactivates the modal.
func (m *PlanViewModal) Hide() {
	m.active = false
}

// IsActive reports whether the modal is currently shown.
func (m PlanViewModal) IsActive() bool {
	return m.active
}

// SetContent pre-renders the markdown via Glamour and populates the viewport.
// This must be called from Update — never from View.
//
// width is the outer modal width (including border). The content render width
// is narrowed by planContentMargin to leave room for padding.
func (m *PlanViewModal) SetContent(markdown string, width int) {
	m.content = markdown

	renderWidth := width - planContentMargin
	if renderWidth < 20 {
		renderWidth = 20
	}

	rendered, _ := util.RenderMarkdown(markdown, renderWidth)
	m.rendered = rendered

	m.rebuildViewport(width, m.height)
}

// SetSize updates the terminal dimensions and resizes the internal viewport.
// Call this on every tea.WindowSizeMsg before the first render.
func (m *PlanViewModal) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.rebuildViewport(w, h)
}

// rebuildViewport recomputes the viewport dimensions and reloads the rendered
// content.  The viewport inner width/height account for the border frame,
// header rows, and footer rows.
func (m *PlanViewModal) rebuildViewport(w, h int) {
	innerW := w - planBorderFrame
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - planBorderFrame - planHeaderRows - planFooterRows
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

// Update processes tea.Msg events.  When the modal is not active all messages
// are ignored.  Esc and 'q' close the modal and emit PlanViewClosedMsg;
// all other keys are forwarded to the viewport for scrolling.
func (m PlanViewModal) Update(msg tea.Msg) (PlanViewModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.active = false
			return m, func() tea.Msg { return PlanViewClosedMsg{} }
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
// modal is not active so callers can cheaply short-circuit.
//
// View is a pure function — no I/O, no rendering.
func (m PlanViewModal) View() string {
	if !m.active {
		return ""
	}
	return renderPlanView(m)
}

// renderPlanView builds the full modal string from the pre-rendered content.
func renderPlanView(m PlanViewModal) string {
	innerW := m.viewport.Width
	if innerW < 1 {
		innerW = m.width - planBorderFrame
		if innerW < 1 {
			innerW = 40
		}
	}

	// Header: title + divider.
	title := planTitleStyle.Render("Plan Preview")
	divider := planDividerStyle.Render(dividerLine(innerW))

	// Viewport content.
	scrollPct := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPct = int(m.viewport.ScrollPercent() * 100)
	}
	vpView := m.viewport.View()

	// Footer: keyboard hints + scroll position.
	footer := planFooterStyle.Render(
		fmt.Sprintf("↑/↓/pgup/pgdn: scroll  q/esc: close  %3d%%", scrollPct),
	)

	inner := strings.Join([]string{title, divider, vpView, footer}, "\n")

	return planBorderStyle.
		Width(innerW).
		Render(inner)
}

// dividerLine returns a horizontal rule string of the given width, capped at
// the inner content width to prevent overflow.
func dividerLine(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 80 {
		width = 80
	}
	return strings.Repeat("\u2500", width)
}
