// Package claude implements the Claude conversation panel for the
// GOgent-Fortress TUI. It renders the assistant/user message history,
// handles streaming updates, and provides a text-input line for composing
// new messages.
package claude

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// CLIDriverSender interface
//
// CLIDriverSender is a minimal interface that decouples ClaudePanelModel from
// the concrete cli package.  AppModel wires the actual CLI driver into the
// panel via SetSender.
// ---------------------------------------------------------------------------

// CLIDriverSender allows the panel to send a user message to the active
// Claude CLI session without importing the cli package directly.
type CLIDriverSender interface {
	// SendMessage submits text to the CLI driver and returns a Cmd that
	// delivers the result as a tea.Msg when complete.
	SendMessage(text string) tea.Cmd
}

// ---------------------------------------------------------------------------
// Display types
// ---------------------------------------------------------------------------

// ToolBlock represents a single tool invocation that is embedded inside a
// DisplayMessage.  By default it is collapsed; the user can expand it to
// see the full input/output summaries.
type ToolBlock struct {
	// Name is the tool name, e.g. "Read" or "Bash".
	Name string
	// Input is a short human-readable summary of the tool arguments.
	Input string
	// Output is a short human-readable summary of the tool result.
	Output string
	// Expanded controls whether the full Input/Output is shown.
	Expanded bool
}

// DisplayMessage is one entry in the conversation history.  It corresponds
// to a single assistant or user turn.
type DisplayMessage struct {
	// Role is "user", "assistant", or "system".
	Role string
	// Content is the plain-text body of the message.
	Content string
	// ToolBlocks lists any tool calls embedded in an assistant message.
	ToolBlocks []ToolBlock
	// Timestamp is when the message was first created or last updated.
	Timestamp time.Time
}

// ---------------------------------------------------------------------------
// Package-level styles
// ---------------------------------------------------------------------------

var (
	// userRoleStyle renders the "You:" prefix for user messages.
	userRoleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(config.ColorPrimary)

	// assistantRoleStyle renders the "Claude:" prefix for assistant messages.
	assistantRoleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorAccent)

	// systemRoleStyle renders the "System:" prefix for system messages.
	systemRoleStyle = config.StyleMuted.Copy().Bold(true)

	// toolNameStyle renders the collapsed tool-block name.
	toolNameStyle = config.StyleSubtle.Copy()

	// inputPromptStyle renders the "› " prompt prefix.
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(config.ColorPrimary).
				Bold(true)

	// streamingStyle renders the "..." streaming indicator.
	streamingStyle = config.StyleMuted.Copy()
)

// ---------------------------------------------------------------------------
// ClaudePanelModel
// ---------------------------------------------------------------------------

// ClaudePanelModel is the Bubbletea sub-model for the Claude conversation
// panel.  It renders the conversation history in a scrollable viewport and
// provides a text-input line for composing messages.
//
// The zero value is not usable; use NewClaudePanelModel instead.
type ClaudePanelModel struct {
	// messages holds the full conversation history in display order.
	messages []DisplayMessage

	// viewport renders the scrollable conversation history.
	vp viewport.Model

	// input is the text-input widget for composing messages.
	input textinput.Model

	// inputHistory holds previously submitted messages, oldest first.
	inputHistory []string
	// historyIdx is the current position when navigating history.
	// -1 means "not currently in history mode" (showing the live draft).
	historyIdx int

	// draftInput preserves the user's in-progress text when they start
	// navigating history so it can be restored on HistoryNext at the end.
	draftInput string

	// streaming is true while an assistant response is being streamed.
	streaming bool

	// autoScroll is true when the viewport should follow new content.
	autoScroll bool

	// width and height are the outer dimensions of the panel.
	width  int
	height int

	// focused controls whether keyboard input is accepted.
	focused bool

	// keys is a copy of the Claude-specific keybindings.
	keys config.ClaudeKeys

	// sender is the injected CLI driver used to submit messages.
	// It is nil until SetSender is called.
	sender CLIDriverSender
}

// NewClaudePanelModel creates a ClaudePanelModel ready for embedding.
// The viewport and textinput are initialised with zero dimensions; call
// SetSize before the first render.
func NewClaudePanelModel(keys config.KeyMap) ClaudePanelModel {
	ti := textinput.New()
	ti.Placeholder = "Message Claude…"
	ti.CharLimit = 4096

	vp := viewport.New(0, 0)

	return ClaudePanelModel{
		vp:         vp,
		input:      ti,
		historyIdx: -1,
		autoScroll: true,
		keys:       keys.Claude,
	}
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model. It returns textinput.Blink so the cursor
// animates when the panel is first shown.
func (m ClaudePanelModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model. It handles all incoming messages and keyboard
// events, returning the updated model and any commands to run.
func (m ClaudePanelModel) Update(msg tea.Msg) (ClaudePanelModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case model.AssistantMsg:
		m, cmds = m.handleAssistantMsg(msg, cmds)
		return m, tea.Batch(cmds...)

	case model.StreamEventMsg:
		// Raw stream events are currently handled via AssistantMsg; preserve
		// the streaming flag only.
		if msg.EventType == "assistant" {
			m.streaming = true
		}
		return m, nil

	case model.ResultMsg:
		m.streaming = false
		m.syncViewport()
		if m.autoScroll {
			m.vp.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKey(msg)
	}

	// Forward remaining messages to the viewport (scroll key handling) and
	// the textinput (cursor blink).
	var vpCmd, tiCmd tea.Cmd

	// Only forward to viewport when the input is NOT capturing keys.
	if !m.input.Focused() {
		m.vp, vpCmd = m.vp.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	m.input, tiCmd = m.input.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

// handleAssistantMsg appends or updates the last assistant message.
func (m ClaudePanelModel) handleAssistantMsg(
	msg model.AssistantMsg,
	cmds []tea.Cmd,
) (ClaudePanelModel, []tea.Cmd) {
	if msg.Streaming {
		m.streaming = true
		// If the last message is already an in-progress assistant fragment,
		// append to it; otherwise start a new one.
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			m.messages[len(m.messages)-1].Content += msg.Text
		} else {
			m.messages = append(m.messages, DisplayMessage{
				Role:      "assistant",
				Content:   msg.Text,
				Timestamp: time.Now(),
			})
		}
	} else {
		// Complete (non-streaming) assistant message.
		m.streaming = false
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" &&
			m.messages[len(m.messages)-1].Content == "" {
			// Patch empty streaming stub.
			m.messages[len(m.messages)-1].Content = msg.Text
			m.messages[len(m.messages)-1].Timestamp = time.Now()
		} else {
			m.messages = append(m.messages, DisplayMessage{
				Role:      "assistant",
				Content:   msg.Text,
				Timestamp: time.Now(),
			})
		}
	}

	m.syncViewport()

	// Re-enable autoScroll when new content arrives and we are at the bottom.
	if m.vp.AtBottom() {
		m.autoScroll = true
	}

	if m.autoScroll {
		m.vp.GotoBottom()
	}

	return m, cmds
}

// handleKey processes keyboard input while the panel is focused.
func (m ClaudePanelModel) handleKey(msg tea.KeyMsg) (ClaudePanelModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Submit):
		text := strings.TrimSpace(m.input.Value())
		if text == "" || m.streaming {
			return m, nil
		}
		// Record in history (avoid duplicates at the top).
		if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != text {
			m.inputHistory = append(m.inputHistory, text)
		}
		m.historyIdx = -1
		m.draftInput = ""
		m.input.Reset()

		// Append user message to conversation.
		m.messages = append(m.messages, DisplayMessage{
			Role:      "user",
			Content:   text,
			Timestamp: time.Now(),
		})
		m.syncViewport()
		m.autoScroll = true
		m.vp.GotoBottom()

		if m.sender != nil {
			cmds = append(cmds, m.sender.SendMessage(text))
		}
		return m, tea.Batch(cmds...)

	case key.Matches(msg, m.keys.HistoryPrev):
		m = m.navigateHistoryPrev()
		return m, nil

	case key.Matches(msg, m.keys.HistoryNext):
		m = m.navigateHistoryNext()
		return m, nil
	}

	// Forward all other key events to the textinput.
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)

	// Check if the viewport was scrolled by user (not textinput key).
	if !m.vp.AtBottom() {
		m.autoScroll = false
	}

	return m, tiCmd
}

// navigateHistoryPrev moves one step backward in input history.
func (m ClaudePanelModel) navigateHistoryPrev() ClaudePanelModel {
	if len(m.inputHistory) == 0 {
		return m
	}
	// Capture the current draft on first navigation.
	if m.historyIdx == -1 {
		m.draftInput = m.input.Value()
		m.historyIdx = len(m.inputHistory) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}
	m.input.SetValue(m.inputHistory[m.historyIdx])
	m.input.CursorEnd()
	return m
}

// navigateHistoryNext moves one step forward in input history (or restores
// the draft when past the end of history).
func (m ClaudePanelModel) navigateHistoryNext() ClaudePanelModel {
	if m.historyIdx == -1 {
		return m
	}
	if m.historyIdx < len(m.inputHistory)-1 {
		m.historyIdx++
		m.input.SetValue(m.inputHistory[m.historyIdx])
		m.input.CursorEnd()
	} else {
		// Past the end: restore draft.
		m.historyIdx = -1
		m.input.SetValue(m.draftInput)
		m.input.CursorEnd()
	}
	return m
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View implements tea.Model. It renders the scrollable conversation viewport
// above the fixed input line. No I/O is performed here.
func (m ClaudePanelModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	inputLine := m.renderInputLine()
	inputH := lipgloss.Height(inputLine)

	// The viewport takes all remaining vertical space.
	vpH := m.height - inputH
	if vpH < 1 {
		vpH = 1
	}
	if m.vp.Height != vpH {
		// Non-mutating: we only update in SetSize / explicit calls.  A
		// transient height mismatch here is acceptable.
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.vp.View(),
		inputLine,
	)
}

// renderInputLine builds the single-line input area including the "› " prompt.
func (m ClaudePanelModel) renderInputLine() string {
	prompt := inputPromptStyle.Render("› ")
	inputView := m.input.View()

	line := prompt + inputView

	if m.streaming {
		indicator := streamingStyle.Render("  ...")
		line += indicator
	}

	return line
}

// ---------------------------------------------------------------------------
// Content rendering helpers
// ---------------------------------------------------------------------------

// syncViewport rebuilds the viewport content from the current messages slice.
// It must be called every time messages changes.
func (m *ClaudePanelModel) syncViewport() {
	m.vp.SetContent(m.renderMessages())
}

// renderMessages renders the full conversation history as a single string
// suitable for viewport.SetContent.
func (m ClaudePanelModel) renderMessages() string {
	if len(m.messages) == 0 {
		return config.StyleMuted.Render("No messages yet. Start typing below.")
	}

	var sb strings.Builder
	for i, msg := range m.messages {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(m.renderMessage(msg, i == len(m.messages)-1))
	}

	if m.streaming {
		sb.WriteString("\n" + streamingStyle.Render("..."))
	}

	return sb.String()
}

// renderMessage renders a single DisplayMessage as a labelled block.
//
// For completed assistant messages, the content is passed through Glamour for
// markdown rendering (syntax-highlighted code blocks, styled headings, etc.).
// While streaming is active the in-progress assistant fragment is rendered as
// plain text to avoid the performance cost of calling Glamour on every token.
//
// isLast must be true when this is the last message in the conversation, so
// the streaming guard can correctly suppress Glamour for the live fragment.
func (m ClaudePanelModel) renderMessage(msg DisplayMessage, isLast bool) string {
	var sb strings.Builder

	// Role label.
	switch msg.Role {
	case "user":
		sb.WriteString(userRoleStyle.Render("You:"))
	case "assistant":
		sb.WriteString(assistantRoleStyle.Render("Claude:"))
	default:
		sb.WriteString(systemRoleStyle.Render("System:"))
	}
	sb.WriteByte('\n')

	// Content body.
	if msg.Content != "" {
		// Render assistant content through Glamour when the message is
		// complete (not currently being streamed into).  While streaming,
		// use plain text to keep rendering cost negligible.
		isLastMsg := isLast

		if msg.Role == "assistant" && !(m.streaming && isLastMsg) {
			// Completed assistant message — render markdown.
			rendered, _ := util.RenderMarkdown(msg.Content, m.width)
			sb.WriteString(rendered)
		} else {
			sb.WriteString(msg.Content)
			sb.WriteByte('\n')
		}
	}

	// Collapsed tool blocks.
	for _, tb := range msg.ToolBlocks {
		sb.WriteString(renderToolBlock(tb, m.width))
	}

	return sb.String()
}

// renderToolBlock renders a ToolBlock either collapsed (just the name) or
// expanded (name + input + output).
func renderToolBlock(tb ToolBlock, _ int) string {
	if !tb.Expanded {
		return toolNameStyle.Render(fmt.Sprintf("  [tool: %s]", tb.Name)) + "\n"
	}
	var sb strings.Builder
	sb.WriteString(toolNameStyle.Render(fmt.Sprintf("  [tool: %s]", tb.Name)))
	sb.WriteByte('\n')
	if tb.Input != "" {
		sb.WriteString(config.StyleSubtle.Render("    in:  "+tb.Input) + "\n")
	}
	if tb.Output != "" {
		sb.WriteString(config.StyleSubtle.Render("    out: "+tb.Output) + "\n")
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Public mutators
// ---------------------------------------------------------------------------

// SetSize updates the width and height of the panel and resizes the viewport
// and textinput accordingly.  This must be called from the parent model's
// Update on every tea.WindowSizeMsg.
func (m *ClaudePanelModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Reserve one line for the input row.
	vpH := height - 1
	if vpH < 1 {
		vpH = 1
	}

	m.vp.Width = width
	m.vp.Height = vpH
	m.input.Width = width - 3 // subtract prompt width ("› ")

	// Re-sync so content is re-wrapped to the new width.
	m.syncViewport()

	if m.autoScroll {
		m.vp.GotoBottom()
	}
}

// SetFocused sets whether the panel accepts keyboard input.  When focused,
// the textinput cursor is shown.
func (m *ClaudePanelModel) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

// Focus enables keyboard input for the panel.
func (m *ClaudePanelModel) Focus() {
	m.SetFocused(true)
}

// Blur disables keyboard input for the panel.
func (m *ClaudePanelModel) Blur() {
	m.SetFocused(false)
}

// SetSender injects the CLI driver implementation.  This mirrors the
// SetCLIDriver / SetBridge pattern used in AppModel.
func (m *ClaudePanelModel) SetSender(sender CLIDriverSender) {
	m.sender = sender
}

// IsStreaming returns true while an assistant response is being streamed.
func (m ClaudePanelModel) IsStreaming() bool {
	return m.streaming
}

// Messages returns a copy of the conversation history. It is intended for
// testing and diagnostic use; callers must not modify the returned slice.
func (m ClaudePanelModel) Messages() []DisplayMessage {
	out := make([]DisplayMessage, len(m.messages))
	copy(out, m.messages)
	return out
}

// HandleMsg is the pointer-receiver equivalent of Update. It mutates the
// model in place and returns only the tea.Cmd. This satisfies the
// claudePanelWidget interface defined in the model package, breaking the
// circular import between model and claude.
func (m *ClaudePanelModel) HandleMsg(msg tea.Msg) tea.Cmd {
	updated, cmd := m.Update(msg)
	*m = updated
	return cmd
}
