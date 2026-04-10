// Package drawer implements a collapsible side-drawer component for the
// GOgent-Fortress TUI. Drawers can be minimised (compact tab) or expanded
// (bordered content pane with a scrollable viewport).
package drawer

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ModalResponseMsg is emitted when the user selects or cancels a modal
// displayed in the options drawer. AppModel catches this to resolve the bridge.
type ModalResponseMsg struct {
	RequestID string
	Value     string
	Cancelled bool
}

// PlanViewRequestMsg is emitted when the user presses Alt+V in the plan drawer
// to request the full-screen plan view modal.
type PlanViewRequestMsg struct{}

// OptionsViewRequestMsg is emitted when the user presses Alt+O in the options
// drawer to request the full-screen options view modal.
// It carries all drawer state the modal needs so no additional accessors are
// required at the call site.
type OptionsViewRequestMsg struct {
	// Content is the plain-text drawer content (view mode).
	Content string
	// Interactive indicates the drawer has an active modal selection.
	Interactive bool
	// RequestID is the bridge request ID (interactive mode only).
	RequestID string
	// Message is the modal prompt message (interactive mode only).
	Message string
	// Options is the list of selectable options (interactive mode only).
	Options []string
	// SelectedIdx is the currently highlighted option index (interactive mode only).
	SelectedIdx int
}

// DrawerState describes whether a drawer is currently collapsed or open.
type DrawerState int

const (
	// DrawerMinimized renders as a single compact tab row.
	DrawerMinimized DrawerState = iota
	// DrawerExpanded renders as a full bordered content pane.
	DrawerExpanded
)

// DrawerID is the stable identifier for a drawer instance.
type DrawerID string

const (
	// DrawerOptions is the id for the agent-options drawer.
	DrawerOptions DrawerID = "options"
	// DrawerPlan is the id for the plan-preview drawer.
	DrawerPlan DrawerID = "plan"
	// DrawerTeams is the id for the teams health drawer.
	DrawerTeams DrawerID = "teams"
)

// DrawerModel is the Bubbletea model for a single drawer pane.
// The zero value is not usable; use NewDrawerModel instead.
type DrawerModel struct {
	id         DrawerID
	state      DrawerState
	label      string
	icon       string
	content    string
	hasContent bool
	width      int
	height     int
	viewport   viewport.Model
	focused    bool

	// Modal interaction state (TDS-006).
	// When a BridgeModalRequestMsg is routed to this drawer, these fields
	// track the active modal so HandleKey can render options and deliver
	// the user's selection back to the bridge goroutine.
	activeRequestID string
	activeMessage   string
	activeOptions   []string
	selectedIdx     int
}

// NewDrawerModel returns a DrawerModel in the minimised state with no content.
func NewDrawerModel(id DrawerID, label string, icon string) DrawerModel {
	vp := viewport.New(0, 0)
	return DrawerModel{
		id:       id,
		state:    DrawerMinimized,
		label:    label,
		icon:     icon,
		viewport: vp,
	}
}

// SetSize updates the allocated width and height and refreshes the viewport
// dimensions to match the inner content area (subtracting borders and header).
func (m *DrawerModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Inner dimensions: borders take 2 cols wide and 2 rows tall,
	// header+divider+footer = 3 rows.  Total overhead = 5 rows.
	innerW := w - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - 5 // 2 (border top+bottom) + 1 header + 1 divider + 1 footer
	if innerH < 1 {
		innerH = 1
	}
	m.viewport.Width = innerW
	m.viewport.Height = innerH
	m.viewport.SetContent(m.content)
}

// SetContent stores the given text in the drawer, syncs the viewport, and
// auto-expands the drawer so the content becomes visible.
func (m *DrawerModel) SetContent(content string) {
	m.content = content
	m.hasContent = content != ""
	m.viewport.SetContent(content)
	if m.hasContent {
		m.state = DrawerExpanded
	}
}

// RefreshContent updates the drawer content without changing expansion state
// when the drawer already has content. On first content arrival it auto-expands;
// when content becomes empty it auto-minimises. Used for live-updating drawers
// (e.g. teams health) where the content changes every poll tick.
func (m *DrawerModel) RefreshContent(content string) {
	if content == "" {
		if m.hasContent {
			m.ClearContent()
		}
		return
	}
	if !m.hasContent {
		// First content — auto-expand.
		m.SetContent(content)
		return
	}
	// Update existing content without changing expansion state.
	m.content = content
	m.viewport.SetContent(content)
}

// ClearContent empties the drawer content and auto-minimises it.
func (m *DrawerModel) ClearContent() {
	m.content = ""
	m.hasContent = false
	m.viewport.SetContent("")
	m.state = DrawerMinimized
}

// Expand opens the drawer if it is not already expanded.
func (m *DrawerModel) Expand() { m.state = DrawerExpanded }

// Minimize collapses the drawer to its compact tab form.
func (m *DrawerModel) Minimize() { m.state = DrawerMinimized }

// Toggle switches between expanded and minimised states.
func (m *DrawerModel) Toggle() {
	if m.state == DrawerExpanded {
		m.state = DrawerMinimized
	} else {
		m.state = DrawerExpanded
	}
}

// SetFocused marks the drawer as focused (true) or unfocused (false),
// which controls the border style used during rendering.
func (m *DrawerModel) SetFocused(focused bool) { m.focused = focused }

// State returns the current DrawerState.
func (m DrawerModel) State() DrawerState { return m.state }

// HasContent reports whether the drawer holds any content.
func (m DrawerModel) HasContent() bool { return m.hasContent }

// ID returns the DrawerID of this drawer.
func (m DrawerModel) ID() DrawerID { return m.id }

// IsFocused reports whether the drawer currently holds focus.
func (m DrawerModel) IsFocused() bool { return m.focused }

// SetActiveModal stores the modal request state and renders options into the
// drawer content area, expanding it automatically.
func (m *DrawerModel) SetActiveModal(requestID string, message string, options []string) {
	m.activeRequestID = requestID
	m.activeMessage = message
	m.activeOptions = options
	m.selectedIdx = 0
	m.state = DrawerExpanded
	formatted := m.formatModalContent(message, options, 0)
	m.content = formatted
	m.hasContent = true
	m.viewport.SetContent(formatted)
}

// HasActiveModal returns true when the drawer is displaying a modal choice.
func (m *DrawerModel) HasActiveModal() bool { return m.activeRequestID != "" }

// ClearActiveModal removes modal state without affecting drawer expansion.
func (m *DrawerModel) ClearActiveModal() {
	m.activeRequestID = ""
	m.activeMessage = ""
	m.activeOptions = nil
	m.selectedIdx = 0
}

// SelectedOption returns the currently highlighted option label, or "" if none.
func (m *DrawerModel) SelectedOption() string {
	if len(m.activeOptions) == 0 || m.selectedIdx >= len(m.activeOptions) {
		return ""
	}
	return m.activeOptions[m.selectedIdx]
}

// ActiveRequestID returns the current modal request ID.
func (m *DrawerModel) ActiveRequestID() string { return m.activeRequestID }

// Content returns the current plain-text content of the drawer.
func (m DrawerModel) Content() string { return m.content }

// ActiveMessage returns the modal prompt message when a modal is active.
func (m DrawerModel) ActiveMessage() string { return m.activeMessage }

// ActiveOptions returns the list of selectable options when a modal is active.
func (m DrawerModel) ActiveOptions() []string { return m.activeOptions }

// SelectedIdx returns the currently highlighted option index when a modal is active.
func (m DrawerModel) SelectedIdx() int { return m.selectedIdx }

// formatModalContent renders the modal prompt and options list with a cursor
// indicator on the currently selected option.
func (m DrawerModel) formatModalContent(message string, options []string, selectedIdx int) string {
	var sb strings.Builder
	sb.WriteString(message)
	sb.WriteString("\n\n")
	for i, opt := range options {
		if i == selectedIdx {
			sb.WriteString("  ▸ " + opt + "\n")
		} else {
			sb.WriteString("    " + opt + "\n")
		}
	}
	sb.WriteString("\n[Enter] Select  [Esc] Cancel")
	return sb.String()
}

// ViewMinimized renders a compact bordered row: icon + label.
func (m DrawerModel) ViewMinimized() string {
	tab := config.StyleSubtle.Render(m.icon + " " + m.label)
	borderStyle := config.StyleUnfocusedBorder
	if m.focused {
		borderStyle = config.StyleFocusedBorder
	}
	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}
	return borderStyle.Width(innerW).Render(tab)
}

// ViewExpanded renders the full bordered content pane with:
//   - header: icon + label
//   - divider line
//   - scrollable viewport
//   - footer hint
func (m DrawerModel) ViewExpanded() string {
	borderStyle := config.StyleUnfocusedBorder
	if m.focused {
		borderStyle = config.StyleFocusedBorder
	}

	header := config.StyleSubtle.Render(m.icon + " " + m.label)

	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}

	divider := config.StyleMuted.Render(strings.Repeat("─", innerW))

	vpView := m.viewport.View()

	hint := "↑/↓ scroll • esc minimize"
	if m.id == DrawerPlan {
		hint = "↑/↓ scroll • alt+v full view • esc minimize"
	} else if m.id == DrawerOptions {
		hint = "↑/↓ scroll • alt+o full view • esc minimize"
	}
	hint = config.StyleMuted.Render(hint)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		vpView,
		hint,
	)

	return borderStyle.
		Width(innerW).
		Render(inner)
}

// View dispatches to ViewMinimized or ViewExpanded based on the current state.
func (m DrawerModel) View() string {
	if m.state == DrawerExpanded {
		return m.ViewExpanded()
	}
	return m.ViewMinimized()
}

// HandleKey processes keyboard input for the drawer.
// When a modal is active, up/down/enter/esc drive option selection.
// Otherwise, esc minimises and up/down/pgup/pgdn scroll the viewport.
// Returns any resulting tea.Cmd (nil for most operations).
func (m *DrawerModel) HandleKey(msg tea.KeyMsg) tea.Cmd {
	// Modal-active key handling (TDS-006).
	if m.HasActiveModal() {
		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			formatted := m.formatModalContent(m.activeMessage, m.activeOptions, m.selectedIdx)
			m.content = formatted
			m.viewport.SetContent(formatted)
		case "down", "j":
			if m.selectedIdx < len(m.activeOptions)-1 {
				m.selectedIdx++
			}
			formatted := m.formatModalContent(m.activeMessage, m.activeOptions, m.selectedIdx)
			m.content = formatted
			m.viewport.SetContent(formatted)
		case "enter":
			selected := m.SelectedOption()
			reqID := m.activeRequestID
			m.ClearActiveModal()
			m.ClearContent()
			return func() tea.Msg {
				return ModalResponseMsg{RequestID: reqID, Value: selected}
			}
		case "esc":
			reqID := m.activeRequestID
			m.ClearActiveModal()
			m.ClearContent()
			return func() tea.Msg {
				return ModalResponseMsg{RequestID: reqID, Value: "", Cancelled: true}
			}
		}
		// Swallow all other keys during modal interaction.
		return nil
	}

	// Plan drawer: Alt+V opens full-screen plan view (TDS-007).
	if msg.String() == "alt+v" && m.id == DrawerPlan {
		return func() tea.Msg { return PlanViewRequestMsg{} }
	}

	// Options drawer: Alt+O opens full-screen options view.
	if msg.String() == "alt+o" && m.id == DrawerOptions {
		if m.HasActiveModal() {
			reqID := m.activeRequestID
			message := m.activeMessage
			opts := m.activeOptions
			idx := m.selectedIdx
			return func() tea.Msg {
				return OptionsViewRequestMsg{
					Interactive: true,
					RequestID:   reqID,
					Message:     message,
					Options:     opts,
					SelectedIdx: idx,
				}
			}
		}
		content := m.content
		return func() tea.Msg {
			return OptionsViewRequestMsg{Content: content}
		}
	}

	// Normal (non-modal) key handling.
	switch msg.String() {
	case "esc":
		m.Minimize()
	case "up", "k":
		m.viewport.LineUp(1)
	case "down", "j":
		m.viewport.LineDown(1)
	case "pgup":
		m.viewport.HalfViewUp()
	case "pgdown":
		m.viewport.HalfViewDown()
	}
	return nil
}
