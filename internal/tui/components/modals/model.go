package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// ModalResponseMsg
//
// ModalResponseMsg is the tea.Msg emitted by ModalModel when the user
// confirms or cancels.  It is defined here (not in the model package) to
// keep the modals package self-contained and avoid a circular dependency:
// model → modals → model.
//
// AppModel.Update type-switches on this message to advance the modal queue
// and deliver the response.
// ---------------------------------------------------------------------------

// ModalResponseMsg is sent by ModalModel when the user makes a selection or
// presses Escape.
type ModalResponseMsg struct {
	// RequestID mirrors ModalRequest.ID so callers can route the response.
	RequestID string
	// Response is the user's answer.
	Response ModalResponse
}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// modalMinWidth is the minimum inner content width for a rendered modal.
	modalMinWidth = 40
	// modalMaxWidth caps the modal so it never fills a very wide terminal.
	modalMaxWidth = 70
	// modalPaddingH is horizontal padding (left + right) inside the border.
	modalPaddingH = 4
	// modalPaddingV is vertical padding (top + bottom) inside the border.
	modalPaddingV = 2
)

// ---------------------------------------------------------------------------
// Lipgloss styles (package-level so they are created once)
// ---------------------------------------------------------------------------

var (
	// modalBorderStyle is the outer box rendered as a double border.
	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(config.ColorPrimary).
				Padding(1, 2)

	// modalHeaderStyle is applied to the modal heading row.
	modalHeaderStyle = config.StyleTitle.Copy()

	// modalMessageStyle is applied to the body paragraph.
	modalMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "15"})

	// modalSelectedStyle highlights the cursor row.
	modalSelectedStyle = config.StyleHighlight.Copy().PaddingLeft(0)

	// modalOptionStyle renders non-selected options.
	modalOptionStyle = config.StyleSubtle.Copy().PaddingLeft(0)

	// modalHintStyle renders the keyboard hint line at the bottom.
	modalHintStyle = config.StyleSubtle.Copy()
)

// ---------------------------------------------------------------------------
// ModalModel
// ---------------------------------------------------------------------------

// ModalModel is a Bubbletea tea.Model that renders a single centered modal
// overlay.  It is created by ModalQueue.Activate and should not be
// constructed directly.
//
// The zero value is not usable; use newModalModel instead.
type ModalModel struct {
	// request is the immutable specification for this modal.
	request ModalRequest

	// effectiveOptions is the final option list shown in the UI.
	// For Confirm modals it is always ["Yes", "No"].
	// For Permission modals it is ["Allow", "Deny"].
	// For Ask modals it is request.Options + ["Other..."].
	effectiveOptions []string

	// selectedIdx is the currently highlighted option index.
	selectedIdx int

	// inputMode is true when the user is typing free text (Input modal type
	// or Ask "Other..." entry selected).
	inputMode bool

	// textInput is the bubbles text-input sub-component used for Input and
	// Ask-Other modes.
	textInput textinput.Model

	// keys is the modal keybinding set from config.
	keys config.ModalKeys

	// termWidth and termHeight are set by the parent before rendering to
	// enable correct centering via lipgloss.Place.
	termWidth  int
	termHeight int
}

// newModalModel constructs a ModalModel from a ModalRequest.
func newModalModel(req ModalRequest, km config.KeyMap) ModalModel {
	ti := textinput.New()
	ti.Placeholder = "Type your response..."
	ti.CharLimit = 512

	if req.Type == Input {
		ti.Focus()
	}

	opts := buildOptions(req)

	return ModalModel{
		request:          req,
		effectiveOptions: opts,
		keys:             km.Modal,
		textInput:        ti,
		inputMode:        req.Type == Input,
		// Default terminal dimensions; overridden by SetTermSize before View.
		termWidth:  80,
		termHeight: 24,
	}
}

// buildOptions derives the effective option list for a ModalRequest.
func buildOptions(req ModalRequest) []string {
	switch req.Type {
	case Confirm:
		return []string{"Yes", "No"}
	case Permission:
		return []string{"Allow", "Deny"}
	case Ask:
		opts := make([]string, 0, len(req.Options)+1)
		opts = append(opts, req.Options...)
		opts = append(opts, "Other...")
		return opts
	case Select:
		if len(req.Options) > 0 {
			return req.Options
		}
		return nil
	case Input:
		// No option list; text field only.
		return nil
	default:
		return req.Options
	}
}

// SetTermSize injects the current terminal dimensions so View() can centre
// the overlay correctly.  Call this before the first render and on every
// tea.WindowSizeMsg.
func (m *ModalModel) SetTermSize(w, h int) {
	m.termWidth = w
	m.termHeight = h
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model.  Modals are created with state already set, so
// no startup commands are needed.
func (m ModalModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.  It handles keyboard navigation, selection,
// cancellation, and text input for all five modal types.
func (m ModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward all messages to the text input so it handles cursor blink, etc.
	if m.inputMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey processes keyboard events.
func (m ModalModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Escape always cancels regardless of mode.
	if key.Matches(msg, m.keys.ModalCancel) {
		return m, m.emitResponse(ModalResponse{
			Type:      m.request.Type,
			Cancelled: true,
		})
	}

	// In text-input mode route most keys to the text input.
	if m.inputMode {
		switch {
		case key.Matches(msg, m.keys.ModalSelect):
			val := strings.TrimSpace(m.textInput.Value())
			return m, m.emitResponse(ModalResponse{
				Type:  m.request.Type,
				Value: val,
			})
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// Navigation and selection for option-list modals.
	switch {
	case key.Matches(msg, m.keys.ModalUp):
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case key.Matches(msg, m.keys.ModalDown):
		if m.selectedIdx < len(m.effectiveOptions)-1 {
			m.selectedIdx++
		}
		return m, nil

	case key.Matches(msg, m.keys.ModalSelect):
		return m.confirmSelection()

	default:
		// For Ask modals, any printable character that is not a navigation key
		// triggers free-text "Other" mode.
		if m.request.Type == Ask && isPrintable(msg) {
			otherIdx := len(m.effectiveOptions) - 1
			m.selectedIdx = otherIdx
			m.inputMode = true
			m.textInput.Focus()
			// Seed the text input with the typed character.
			m.textInput.SetValue(string(msg.Runes))
			m.textInput, _ = m.textInput.Update(msg)
			// Move cursor to end.
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(tea.KeyMsg{Type: tea.KeyEnd})
			return m, cmd
		}
	}

	return m, nil
}

// confirmSelection resolves the modal with the currently selected option.
func (m ModalModel) confirmSelection() (tea.Model, tea.Cmd) {
	if len(m.effectiveOptions) == 0 {
		// Input modal without options — should not reach here since inputMode
		// is set on construction, but guard defensively.
		return m, m.emitResponse(ModalResponse{
			Type:  m.request.Type,
			Value: m.textInput.Value(),
		})
	}

	selected := m.effectiveOptions[m.selectedIdx]

	// For Ask modals, selecting "Other..." enters free-text mode.
	if m.request.Type == Ask && selected == "Other..." {
		m.inputMode = true
		m.textInput.Focus()
		return m, nil
	}

	return m, m.emitResponse(ModalResponse{
		Type:  m.request.Type,
		Value: selected,
	})
}

// emitResponse returns a tea.Cmd that delivers ModalResponseMsg to the
// Bubbletea runtime.
func (m ModalModel) emitResponse(resp ModalResponse) tea.Cmd {
	// Also forward via channel if the caller is blocking on one.
	if m.request.ResponseCh != nil {
		select {
		case m.request.ResponseCh <- resp:
		default:
			// Non-blocking send: if the channel is full the caller is not
			// waiting; proceed without blocking.
		}
	}
	return func() tea.Msg {
		return ModalResponseMsg{
			RequestID: m.request.ID,
			Response:  resp,
		}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the modal as a centered overlay using lipgloss.Place.
// The overlay is always rendered on top of whatever background exists; the
// caller (AppModel) is responsible for rendering the background first and
// then overlaying this view.
func (m ModalModel) View() string {
	content := m.renderContent()

	// Calculate box width: clamp between min and max, but never exceed terminal.
	maxW := modalMaxWidth
	if m.termWidth-4 < maxW {
		maxW = m.termWidth - 4
	}
	if maxW < modalMinWidth {
		maxW = modalMinWidth
	}

	box := modalBorderStyle.Copy().Width(maxW).Render(content)

	return lipgloss.Place(
		m.termWidth, m.termHeight,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderContent builds the string that goes inside the border box.
func (m ModalModel) renderContent() string {
	var sb strings.Builder

	// Header row.
	header := m.request.Header
	if header == "" {
		header = m.request.Type.String()
	}
	sb.WriteString(modalHeaderStyle.Render(header))
	sb.WriteString("\n\n")

	// Body message.
	if m.request.Message != "" {
		sb.WriteString(modalMessageStyle.Render(m.request.Message))
		sb.WriteString("\n\n")
	}

	// Per-type content area.
	switch m.request.Type {
	case Input:
		sb.WriteString(m.renderInputField())
	default:
		sb.WriteString(m.renderOptionList())
	}

	// Keyboard hints.
	sb.WriteString("\n")
	sb.WriteString(modalHintStyle.Render(m.hintLine()))

	return sb.String()
}

// renderOptionList renders the selectable option rows for Ask, Confirm,
// Select, and Permission modals.
func (m ModalModel) renderOptionList() string {
	if len(m.effectiveOptions) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, opt := range m.effectiveOptions {
		cursor := "  "
		if i == m.selectedIdx {
			cursor = "> "
		}

		line := cursor + opt

		if i == m.selectedIdx {
			// If we are in inputMode and this is the "Other..." entry, show
			// the text input inline.
			if m.inputMode && m.request.Type == Ask && opt == "Other..." {
				line = cursor + m.textInput.View()
			}
			sb.WriteString(modalSelectedStyle.Render(line))
		} else {
			sb.WriteString(modalOptionStyle.Render(line))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderInputField renders the text input for the Input modal type.
func (m ModalModel) renderInputField() string {
	return fmt.Sprintf("%s\n", m.textInput.View())
}

// hintLine returns the keyboard shortcut hint string for the current modal
// state.
func (m ModalModel) hintLine() string {
	if m.inputMode {
		return "enter: confirm  esc: cancel"
	}
	switch m.request.Type {
	case Ask, Select:
		return "↑/↓: navigate  enter: select  esc: cancel"
	case Confirm, Permission:
		return "↑/↓: navigate  enter: confirm  esc: cancel"
	case Input:
		return "enter: submit  esc: cancel"
	default:
		return "enter: confirm  esc: cancel"
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isPrintable returns true when the KeyMsg carries printable rune input that
// should seed a free-text entry field.
func isPrintable(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && len(msg.Runes) > 0
}
