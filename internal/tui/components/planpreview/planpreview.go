// Package planpreview implements the plan preview panel for the
// GOgent-Fortress TUI. It renders a Markdown implementation plan in a
// scrollable viewport using Glamour.
//
// The component is display-only: markdown rendering is performed in
// SetContent (called from Update), never in View. No I/O occurs in View.
package planpreview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// PlanPreviewModel
// ---------------------------------------------------------------------------

// PlanPreviewModel is the display model for the plan preview panel. It holds
// the raw markdown and the pre-rendered terminal text. Rendering is performed
// in SetContent so that View stays a pure, fast function.
//
// The zero value is not usable; use NewPlanPreviewModel instead.
type PlanPreviewModel struct {
	width    int
	height   int
	content  string // raw markdown
	rendered string // glamour-rendered terminal text
	viewport viewport.Model
	hasContent bool
}

// NewPlanPreviewModel returns a PlanPreviewModel with an initialised viewport.
func NewPlanPreviewModel() PlanPreviewModel {
	vp := viewport.New(0, 0)
	return PlanPreviewModel{
		viewport: vp,
	}
}

// SetSize updates the rendering dimensions and resizes the viewport. Call
// this on every tea.WindowSizeMsg.
func (m *PlanPreviewModel) SetSize(w, h int) {
	m.width = w
	// Reserve 3 rows for the header (title + divider + blank).
	contentH := h - 3
	if contentH < 1 {
		contentH = 1
	}
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = contentH

	// Re-render at new width if content is present.
	if m.hasContent {
		m.reRender()
	}
}

// SetContent sets the raw markdown content and triggers a re-render. This
// must be called from the parent model's Update method — never from View.
func (m *PlanPreviewModel) SetContent(markdown string) {
	m.content = markdown
	m.hasContent = markdown != ""
	if m.hasContent {
		m.reRender()
	}
}

// Content returns the raw markdown currently loaded in the panel. Returns ""
// when no plan has been set. This method satisfies the planPreviewWidget
// interface so the AppModel can pass the markdown to PlanViewModal.SetContent.
func (m *PlanPreviewModel) Content() string {
	return m.content
}

// ClearContent resets the panel to the empty state.
func (m *PlanPreviewModel) ClearContent() {
	m.content = ""
	m.rendered = ""
	m.hasContent = false
	m.viewport.SetContent("")
	m.viewport.GotoTop()
}

// reRender re-renders m.content using Glamour and updates the viewport.
// Graceful degradation: if rendering fails, raw markdown is used.
func (m *PlanPreviewModel) reRender() {
	rendered, _ := util.RenderMarkdown(m.content, m.viewport.Width)
	m.rendered = rendered
	m.viewport.SetContent(m.rendered)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the plan preview panel. It is a pure function of the model
// state — no I/O is performed here.
func (m PlanPreviewModel) View() string {
	var sb strings.Builder

	// Header.
	sb.WriteString(config.StyleTitle.Render("Plan Preview"))
	sb.WriteByte('\n')
	sb.WriteString(config.StyleSubtle.Render(divider(m.width)))
	sb.WriteByte('\n')

	if !m.hasContent {
		sb.WriteString(config.StyleSubtle.Render("No plan loaded"))
		return sb.String()
	}

	sb.WriteString(m.viewport.View())
	return sb.String()
}

// divider returns a horizontal rule string fitting the given width.
func divider(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 40 {
		width = 40
	}
	return strings.Repeat("\u2500", width)
}
