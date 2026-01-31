package claude

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

// ModalType identifies the kind of modal displayed
type ModalType int

const (
	NoModal ModalType = iota
	ConfirmModal
	TextInputModal
	SelectionModal
)

// ModalState holds the current modal's state
type ModalState struct {
	Active     bool
	Type       ModalType
	Prompt     callback.PromptRequest
	TextInput  textinput.Model
	SelectList list.Model
	Server     *callback.Server
}

// NewModalState creates an empty modal state
func NewModalState() ModalState {
	ti := textinput.New()
	ti.Placeholder = "Type your response..."
	ti.CharLimit = 500
	ti.Width = 40

	return ModalState{
		TextInput: ti,
	}
}

// MCPPromptMsg is sent when a prompt request arrives
type MCPPromptMsg struct {
	Request callback.PromptRequest
	Server  *callback.Server
}

// MCPResponseSentMsg is sent after response is delivered
type MCPResponseSentMsg struct {
	PromptID string
}

// HandlePrompt activates a modal for the given prompt
func (m *ModalState) HandlePrompt(prompt callback.PromptRequest, server *callback.Server) tea.Cmd {
	m.Active = true
	m.Prompt = prompt
	m.Server = server

	switch prompt.Type {
	case "confirm":
		m.Type = ConfirmModal
		return nil

	case "input", "ask":
		if len(prompt.Options) > 0 {
			m.Type = SelectionModal
			m.SelectList = createSelectList(prompt.Options)
			return nil
		}
		m.Type = TextInputModal
		m.TextInput.Reset()
		if prompt.Default != "" {
			m.TextInput.SetValue(prompt.Default)
		}
		return m.TextInput.Focus()

	case "select":
		m.Type = SelectionModal
		m.SelectList = createSelectList(prompt.Options)
		return nil

	default:
		// Fallback to text input
		m.Type = TextInputModal
		m.TextInput.Reset()
		return m.TextInput.Focus()
	}
}

// SendResponse sends the response and closes the modal
func (m *ModalState) SendResponse(value string, cancelled bool) {
	if m.Server == nil {
		return
	}

	// Use server's SendResponse which finds the correct channel
	m.Server.SendResponse(callback.PromptResponse{
		ID:        m.Prompt.ID,
		Value:     value,
		Cancelled: cancelled,
	})

	m.Active = false
	m.Server = nil
}

// listItem implements list.Item for selection
type listItem struct {
	title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }

func createSelectList(options []string) list.Model {
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = listItem{title: opt}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	// Truncate long items to prevent overflow
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.MaxWidth(38)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.MaxWidth(38)

	// Width reduced to 42 to fit within modal (50 width - 4 padding - margin for list styling)
	l := list.New(items, delegate, 42, min(len(options)+4, 12))
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return l
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
