package modals

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Styles (package-level, created once)
// ---------------------------------------------------------------------------

var (
	// confirm2MatchedStyle renders the portion of the phrase the user has
	// typed correctly — bright green, bold.
	confirm2MatchedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorSuccess)

	// confirm2UnmatchedStyle renders the remaining untyped portion of the
	// required phrase — muted/dim so it reads as "still needed".
	confirm2UnmatchedStyle = lipgloss.NewStyle().
				Foreground(config.ColorMuted)

	// confirm2DangerStyle renders the required phrase label using the
	// DangerStyle semantics (bold + underline + red) from the theme system.
	confirm2DangerStyle = config.DefaultTheme().DangerStyle()

	// confirm2HintMatchStyle renders the hint line when the phrase is fully
	// matched and the Enter key is available.
	confirm2HintMatchStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorSuccess)

	// confirm2HintWaitStyle renders the hint line while the phrase is still
	// incomplete (Enter is not yet active).
	confirm2HintWaitStyle = lipgloss.NewStyle().
				Foreground(config.ColorMuted)
)

// ---------------------------------------------------------------------------
// TwoStepConfirmModal
// ---------------------------------------------------------------------------

// TwoStepConfirmModal requires the user to type a confirmation phrase before
// a destructive action is executed. The action is only unlocked when the typed
// phrase exactly matches RequiredPhrase (case-insensitive).
//
// Lifecycle:
//  1. Render the modal via View().
//  2. Forward every tea.Msg via Update().
//  3. When Update returns a non-nil tea.Cmd the user has either confirmed
//     (IsConfirmed() == true) or cancelled (response.Cancelled == true).
//
// The zero value is not usable; use NewTwoStepConfirmModal instead.
type TwoStepConfirmModal struct {
	// Title is rendered as the modal heading.
	Title string

	// Description explains what will happen if confirmed.
	Description string

	// RequiredPhrase is the exact string the user must type (checked
	// case-insensitively).
	RequiredPhrase string

	// requestID is propagated into ModalResponseMsg so callers can correlate
	// responses. Set via the WithRequestID option or left empty.
	requestID string

	// responseCh is an optional blocking channel for callers that need to
	// synchronously wait on the response (mirrors ModalRequest.ResponseCh).
	responseCh chan ModalResponse

	// input is the bubbles text-input sub-component.
	input textinput.Model

	// confirmed is set to true once the typed phrase matches RequiredPhrase.
	confirmed bool

	// keys carries the modal-specific keybindings (Escape = cancel,
	// Enter = confirm when ready).
	keys config.ModalKeys

	// termWidth and termHeight are used to centre the overlay.
	termWidth  int
	termHeight int
}

// NewTwoStepConfirmModal constructs a TwoStepConfirmModal ready to embed in a
// Bubbletea model.  The textinput is focused immediately so the user can start
// typing without an extra click.
func NewTwoStepConfirmModal(title, description, requiredPhrase string) TwoStepConfirmModal {
	ti := textinput.New()
	ti.Placeholder = "Type the phrase above to confirm…"
	ti.CharLimit = 256
	ti.Focus()

	km := config.DefaultKeyMap()

	return TwoStepConfirmModal{
		Title:          title,
		Description:    description,
		RequiredPhrase: requiredPhrase,
		input:          ti,
		keys:           km.Modal,
		// Default fallback dimensions; overridden by SetTermSize before View.
		termWidth:  80,
		termHeight: 24,
	}
}

// WithRequestID sets the requestID that will be echoed in ModalResponseMsg.
// Call this immediately after construction when the modal originates from a
// ModalRequest.
func (m TwoStepConfirmModal) WithRequestID(id string) TwoStepConfirmModal {
	m.requestID = id
	return m
}

// WithResponseCh attaches a buffered channel that will receive the
// ModalResponse when the modal is resolved.  Mirrors ModalRequest.ResponseCh.
func (m TwoStepConfirmModal) WithResponseCh(ch chan ModalResponse) TwoStepConfirmModal {
	m.responseCh = ch
	return m
}

// SetTermSize injects the current terminal dimensions for correct centering.
// Call this from the parent's Update on every tea.WindowSizeMsg.
func (m *TwoStepConfirmModal) SetTermSize(w, h int) {
	m.termWidth = w
	m.termHeight = h
}

// IsConfirmed returns true when the typed phrase exactly matches
// RequiredPhrase (case-insensitive) and the user has pressed Enter.
func (m TwoStepConfirmModal) IsConfirmed() bool {
	return m.confirmed
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model.  Returns a textinput blink command so the cursor
// is animated from the first frame.
func (m TwoStepConfirmModal) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.  It handles keyboard events and delegates all
// other messages to the embedded textinput.
func (m TwoStepConfirmModal) Update(msg tea.Msg) (TwoStepConfirmModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward non-key messages to the text input (cursor blink, etc.).
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleKey processes keyboard events.
func (m TwoStepConfirmModal) handleKey(msg tea.KeyMsg) (TwoStepConfirmModal, tea.Cmd) {
	// Escape always cancels regardless of typed text.
	if key.Matches(msg, m.keys.ModalCancel) {
		return m, m.emitResponse(ModalResponse{
			Type:      TwoStepConfirm,
			Cancelled: true,
		})
	}

	// Enter only confirms when the phrase matches.
	if key.Matches(msg, m.keys.ModalSelect) {
		if m.phraseMatches() {
			m.confirmed = true
			return m, m.emitResponse(ModalResponse{
				Type:  TwoStepConfirm,
				Value: m.input.Value(),
			})
		}
		// Enter pressed before phrase matches — ignore, do not emit.
		return m, nil
	}

	// All other keys go to the text input.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// phraseMatches returns true when the currently typed text exactly matches
// RequiredPhrase, case-insensitively.
func (m TwoStepConfirmModal) phraseMatches() bool {
	return strings.EqualFold(strings.TrimSpace(m.input.Value()), m.RequiredPhrase)
}

// emitResponse returns a tea.Cmd that delivers ModalResponseMsg to the
// Bubbletea runtime and optionally sends on responseCh for blocking callers.
func (m TwoStepConfirmModal) emitResponse(resp ModalResponse) tea.Cmd {
	if m.responseCh != nil {
		select {
		case m.responseCh <- resp:
		default:
			// Non-blocking send: proceed if channel is full.
		}
	}
	return func() tea.Msg {
		return ModalResponseMsg{
			RequestID: m.requestID,
			Response:  resp,
		}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the two-step confirmation modal as a centered overlay.
// It follows the same layout conventions as ModalModel.View():
//   - Double border colored with ColorError (red)
//   - Drop shadow (right + bottom)
//   - Centered via lipgloss.Place
func (m TwoStepConfirmModal) View() string {
	content := m.renderContent()

	// Clamp box width between modalMinWidth and modalMaxWidth.
	maxW := modalMaxWidth
	if m.termWidth-4 < maxW {
		maxW = m.termWidth - 4
	}
	if maxW < modalMinWidth {
		maxW = modalMinWidth
	}

	box := modalBorderBase.Copy().
		BorderForeground(config.ColorError).
		Width(maxW).
		Render(content)

	boxRenderedWidth := lipgloss.Width(box)
	shadow := renderShadow(boxRenderedWidth)
	shadowedBox := appendRightShadow(box) + "\n" + shadow

	return lipgloss.Place(
		m.termWidth, m.termHeight,
		lipgloss.Center, lipgloss.Center,
		shadowedBox,
	)
}

// renderContent assembles the inner string for the border box.
func (m TwoStepConfirmModal) renderContent() string {
	var sb strings.Builder

	// Header row — uses the error icon to signal destructive action.
	icon := config.UnicodeIcons.Error
	title := m.Title
	if title == "" {
		title = TwoStepConfirm.String()
	}
	sb.WriteString(modalHeaderStyle.Copy().Foreground(config.ColorError).Render(icon + " " + title))
	sb.WriteString("\n\n")

	// Description body.
	if m.Description != "" {
		sb.WriteString(modalMessageStyle.Render(m.Description))
		sb.WriteString("\n\n")
	}

	// Required phrase label.
	sb.WriteString(modalMessageStyle.Render("Type the following phrase to confirm:"))
	sb.WriteString("\n")
	sb.WriteString("  " + confirm2DangerStyle.Render(m.RequiredPhrase))
	sb.WriteString("\n\n")

	// Progress feedback line: shows how much of the phrase the user has typed.
	sb.WriteString(m.renderProgress())
	sb.WriteString("\n\n")

	// Text input field.
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")

	// Keyboard hint.
	sb.WriteString(m.renderHint())

	return sb.String()
}

// renderProgress renders a visual indicator showing matched vs unmatched
// portions of the required phrase relative to the current input.
//
// Examples (RequiredPhrase = "delete session"):
//   - typed ""          → dim "delete session"
//   - typed "del"       → green "del" + dim "ete session"
//   - typed "delete session" → green "delete session"
//   - typed "xyz"       → dim "delete session"  (no prefix match)
func (m TwoStepConfirmModal) renderProgress() string {
	required := strings.ToLower(m.RequiredPhrase)
	typed := strings.ToLower(strings.TrimSpace(m.input.Value()))

	// Find the length of the longest common prefix.
	matchLen := 0
	maxLen := len(typed)
	if maxLen > len(required) {
		maxLen = len(required)
	}
	for i := 0; i < maxLen; i++ {
		if required[i] == typed[i] {
			matchLen++
		} else {
			break
		}
	}

	// Use original-case RequiredPhrase for display.
	display := m.RequiredPhrase
	if matchLen == 0 {
		return "  " + confirm2UnmatchedStyle.Render(display)
	}
	if matchLen >= len(display) {
		// Full phrase matched.
		return "  " + confirm2MatchedStyle.Render(display)
	}
	matched := confirm2MatchedStyle.Render(display[:matchLen])
	unmatched := confirm2UnmatchedStyle.Render(display[matchLen:])
	return "  " + matched + unmatched
}

// renderHint returns the keyboard hint line, styled green when the phrase is
// matched (Enter is active) and muted when it is not.
func (m TwoStepConfirmModal) renderHint() string {
	if m.phraseMatches() {
		return confirm2HintMatchStyle.Render("enter: confirm  esc: cancel")
	}
	return confirm2HintWaitStyle.Render("type phrase to enable enter  esc: cancel")
}
