// Package claude implements the Claude conversation panel for the
// GOgent-Fortress TUI. It renders the assistant/user message history,
// handles streaming updates, and provides a text-input line for composing
// new messages.
package claude

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/scrollbar"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/slashcmd"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// CLIDriverSender is a package-level type alias retained for backward
// compatibility.  New code should use model.MessageSender directly.
// The two interfaces are structurally identical; this alias exists so that
// call sites that already reference claude.CLIDriverSender continue to compile.
//
// Deprecated: use model.MessageSender.
type CLIDriverSender = model.MessageSender

// ---------------------------------------------------------------------------
// Display types
// ---------------------------------------------------------------------------

// ToolBlock is an alias for state.ToolBlock — the canonical definition.
type ToolBlock = state.ToolBlock

// DisplayMessage is an alias for state.DisplayMessage — the canonical definition.
type DisplayMessage = state.DisplayMessage

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

	// history holds previously submitted messages with cross-session
	// persistence to ~/.claude/input-history.json (shared with TS TUI).
	history *InputHistory
	// historyIdx is the current position when navigating history.
	// -1 means "not currently in history mode" (showing the live draft).
	historyIdx int

	// draftInput preserves the user's in-progress text when they start
	// navigating history so it can be restored on HistoryNext at the end.
	draftInput string

	// streaming is true while an assistant response is being streamed.
	// Used for the "..." display indicator and as a fallback replace/append
	// decision when MessageID is not available.
	streaming bool

	// currentMsgID is the Message.ID of the assistant message currently being
	// streamed.  It is set when the first fragment of a new turn arrives and
	// cleared on ResultMsg.  When non-empty, replace-vs-append decisions use
	// this ID rather than the streaming boolean to avoid cross-turn overwrites.
	currentMsgID string

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
	sender model.MessageSender

	// search is the in-panel search overlay (TUI-035).
	search SearchModel

	// slashCmd is the slash-command autocomplete dropdown (TUI-054).
	slashCmd slashcmd.SlashCmdModel

	// tier tracks the current layout tier so renderMessage can suppress
	// inline tool blocks when the Activity panel is visible (TUI-L05).
	tier model.LayoutTier
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
		search:     NewSearchModel(),
		slashCmd:   slashcmd.NewSlashCmdModel(),
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

	case model.ToolUseMsg:
		m = m.handleToolUseMsg(msg)
		return m, nil

	case model.ToolResultMsg:
		m = m.handleToolResultMsg(msg)
		return m, nil

	case model.StreamEventMsg:
		// Raw stream events are currently handled via AssistantMsg; preserve
		// the streaming flag only.
		if msg.EventType == "assistant" {
			m.streaming = true
		}
		return m, nil

	case model.ResultMsg:
		m.streaming = false
		m.currentMsgID = ""
		m.syncViewport()
		if m.autoScroll {
			m.vp.GotoBottom()
		}
		return m, nil

	case slashcmd.SlashCmdSelectedMsg:
		// User selected a command from the dropdown via Enter.
		// Execute the command immediately (no args appended).
		m.slashCmd.Hide()
		m.input.Reset()
		return m.executeSlashCommand(msg.Command)

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKey(msg)
	}

	// Forward mouse events to the viewport unconditionally so scroll wheel
	// works even when the text input is focused.  Key events only go to the
	// viewport when the input is blurred (to avoid stealing arrow keys etc.).
	var vpCmd, tiCmd tea.Cmd

	switch msg.(type) {
	case tea.MouseMsg:
		// Mouse wheel scroll → viewport always.
		m.vp, vpCmd = m.vp.Update(msg)
		cmds = append(cmds, vpCmd)
		if !m.vp.AtBottom() {
			m.autoScroll = false
		} else {
			m.autoScroll = true
		}
	default:
		// Non-mouse messages: only forward to viewport when input is blurred.
		if !m.input.Focused() {
			m.vp, vpCmd = m.vp.Update(msg)
			cmds = append(cmds, vpCmd)
		}
	}

	m.input, tiCmd = m.input.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

// handleAssistantMsg appends or updates the last assistant message.
//
// The Claude CLI with --include-partial-messages emits multiple AssistantEvent
// messages per turn while streaming. Each partial event carries the FULL
// accumulated text so far (a snapshot), not an incremental delta. The final
// event has stop_reason set and Streaming=false.
//
// Replace-vs-append decision (in priority order):
//
//  1. MessageID-based (preferred): when msg.MessageID is non-empty, replace
//     the last assistant message only when its ID matches m.currentMsgID.
//     A new or different ID always appends, preventing cross-turn overwrites.
//
//  2. Streaming-bool fallback: when MessageID is empty (legacy / test paths),
//     the original Streaming-boolean logic is used so existing callers and
//     tests that do not set MessageID continue to work unchanged.
func (m ClaudePanelModel) handleAssistantMsg(
	msg model.AssistantMsg,
	cmds []tea.Cmd,
) (ClaudePanelModel, []tea.Cmd) {
	if msg.MessageID != "" {
		// --- MessageID-based path (preferred) ---
		m.streaming = msg.Streaming
		sameMsg := msg.MessageID == m.currentMsgID &&
			len(m.messages) > 0 &&
			m.messages[len(m.messages)-1].Role == "assistant"
		if sameMsg {
			// Same turn: replace snapshot.
			m.messages[len(m.messages)-1].Content = msg.Text
			if !msg.Streaming {
				m.messages[len(m.messages)-1].Timestamp = time.Now()
			}
		} else {
			// New turn (different or first ID): append.
			m.currentMsgID = msg.MessageID
			m.messages = append(m.messages, DisplayMessage{
				Role:      "assistant",
				Content:   msg.Text,
				Timestamp: time.Now(),
			})
		}
		if !msg.Streaming {
			m.currentMsgID = ""
		}
	} else if msg.Streaming {
		// --- Legacy streaming-bool path ---
		alreadyStreaming := m.streaming
		m.streaming = true
		// Replace only if we were already streaming (same turn, cumulative
		// snapshot). If we weren't streaming, this is a new turn — append.
		if alreadyStreaming && len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			m.messages[len(m.messages)-1].Content = msg.Text
		} else {
			m.messages = append(m.messages, DisplayMessage{
				Role:      "assistant",
				Content:   msg.Text,
				Timestamp: time.Now(),
			})
		}
	} else {
		// --- Legacy non-streaming path ---
		// Complete (non-streaming) assistant message. Track whether we were
		// previously streaming so we know whether to replace or append.
		wasStreaming := m.streaming
		m.streaming = false
		if wasStreaming && len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			// Replace the in-progress streaming fragment with the final,
			// complete text. This clears any partial snapshot.
			m.messages[len(m.messages)-1].Content = msg.Text
			m.messages[len(m.messages)-1].Timestamp = time.Now()
		} else {
			// Fresh turn: no prior streaming message to replace.
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

// handleToolUseMsg appends a collapsed tool block to the current assistant
// message so the user can see what tools the router is invoking.
func (m ClaudePanelModel) handleToolUseMsg(msg model.ToolUseMsg) ClaudePanelModel {
	// Append to the last assistant message, or create one if none exists.
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Role != "assistant" {
		m.messages = append(m.messages, DisplayMessage{
			Role:      "assistant",
			Content:   "",
			Timestamp: time.Now(),
		})
	}
	last := &m.messages[len(m.messages)-1]
	last.ToolBlocks = append(last.ToolBlocks, ToolBlock{
		Name:   msg.ToolName,
		ToolID: msg.ToolID,
		Input:  msg.Input,
	})

	m.syncViewport()
	if m.autoScroll {
		m.vp.GotoBottom()
	}
	return m
}

// handleToolResultMsg finds the ToolBlock matching the given ToolID and sets
// its Success field so the collapsed view can show ✓ or ✗.
func (m ClaudePanelModel) handleToolResultMsg(msg model.ToolResultMsg) ClaudePanelModel {
	// Walk messages backwards — the matching tool_use is almost always recent.
	for i := len(m.messages) - 1; i >= 0; i-- {
		for j := range m.messages[i].ToolBlocks {
			if m.messages[i].ToolBlocks[j].ToolID == msg.ToolID {
				s := msg.Success
				m.messages[i].ToolBlocks[j].Success = &s
				m.syncViewport()
				return m
			}
		}
	}
	return m
}

// handleKey processes keyboard input while the panel is focused.
func (m ClaudePanelModel) handleKey(msg tea.KeyMsg) (ClaudePanelModel, tea.Cmd) {
	var cmds []tea.Cmd

	// ---------------------------------------------------------------------------
	// Search mode — route all keys to the search overlay when active.
	// ---------------------------------------------------------------------------
	if m.search.IsActive() {
		switch msg.String() {
		case "ctrl+n":
			m.search.NextResult()
			m.scrollToSearchResult()
			return m, nil
		case "ctrl+p":
			m.search.PrevResult()
			m.scrollToSearchResult()
			return m, nil
		}

		var searchCmd tea.Cmd
		prevQuery := m.search.Query()
		m.search, searchCmd = m.search.Update(msg)
		cmds = append(cmds, searchCmd)

		// If the search query changed, re-run the search immediately so that
		// results update in the same frame without a round-trip through the
		// Bubbletea event loop.
		if m.search.Query() != prevQuery {
			m.search.ExecuteSearch(m.messages)
			m.scrollToSearchResult()
			m.syncViewport()
		}

		// If the search was deactivated by Enter/Esc, scroll to the result.
		if !m.search.IsActive() {
			m.scrollToSearchResult()
		}

		return m, tea.Batch(cmds...)
	}

	// ---------------------------------------------------------------------------
	// Slash command dropdown — intercept navigation keys when visible.
	// ---------------------------------------------------------------------------
	if m.slashCmd.IsVisible() {
		switch msg.String() {
		case "up", "down", "enter":
			// Forward navigation/selection to the dropdown.
			// Note: "k"/"j" are NOT intercepted — they must reach the text
			// input so the user can type commands like "/ticket".
			var scCmd tea.Cmd
			m.slashCmd, scCmd = m.slashCmd.Update(msg)
			return m, scCmd

		case "escape", "esc":
			// Dismiss the dropdown and let the input keep the "/" text.
			m.slashCmd.Hide()
			return m, nil

		case "tab":
			// Tab-complete: insert the selected command name into the input
			// and let the user add args before pressing Enter.
			sel := m.slashCmd.Selected()
			if sel.Name != "" {
				m.slashCmd.Hide()
				newVal := "/" + sel.Name + " "
				m.input.SetValue(newVal)
				m.input.CursorEnd()
			}
			return m, nil
		}
		// All other keys (printable chars, backspace) fall through to normal
		// input handling below, then re-evaluate the dropdown filter.
	}

	// ---------------------------------------------------------------------------
	// Normal mode key handling.
	// ---------------------------------------------------------------------------

	switch {
	case key.Matches(msg, m.keys.Search):
		// "/" activates search mode ONLY when the textinput is not focused.
		// When the textinput is focused the "/" rune should go to the input
		// so the slash-command dropdown can engage.
		if !m.input.Focused() {
			m.search.Activate()
			m.search.SetWidth(m.width)
			return m, nil
		}
		// Input is focused: forward "/" to the textinput below.

	case key.Matches(msg, m.keys.CopyLastResponse):
		// Copy the last assistant message to the clipboard.
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == "assistant" && m.messages[i].Content != "" {
				_ = util.CopyToClipboard(m.messages[i].Content)
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Submit):
		text := strings.TrimSpace(m.input.Value())
		if text == "" || m.streaming {
			return m, nil
		}
		// If the dropdown is still somehow visible (shouldn't happen after
		// Enter interception above), hide it.
		m.slashCmd.Hide()

		// If this looks like a slash command, execute it.
		if strings.HasPrefix(text, "/") {
			m.historyIdx = -1
			m.draftInput = ""
			m.input.Reset()
			return m.executeSlashCommand(text)
		}

		// Record in history and persist to disk (shared with TS TUI).
		if m.history != nil {
			m.history.Add(text)
			_ = m.history.Save()
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

	// PgUp/PgDown scroll the viewport even when the input is focused.
	// This provides scroll access without conflicting with text editing.
	switch msg.String() {
	case "pgup":
		m.vp.HalfViewUp()
		m.autoScroll = false
		return m, nil
	case "pgdown":
		m.vp.HalfViewDown()
		if m.vp.AtBottom() {
			m.autoScroll = true
		}
		return m, nil
	}

	// Forward all other key events to the textinput.
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)

	// After any text change, update the slash-command dropdown state.
	m.updateSlashDropdown()

	// Check if the viewport was scrolled by user (not textinput key).
	if !m.vp.AtBottom() {
		m.autoScroll = false
	}

	return m, tiCmd
}

// scrollToSearchResult scrolls the viewport to the message at the current
// search result index, if one exists.  It is a best-effort scroll: the
// viewport does not support per-line seeking, so we estimate the target
// offset from the number of messages above the result.
func (m *ClaudePanelModel) scrollToSearchResult() {
	idx := m.search.CurrentResultIndex()
	if idx < 0 || idx >= len(m.messages) {
		return
	}
	// Re-render so the viewport content is up to date.
	m.syncViewport()
	// Approximate position: scroll so the target message is near the top.
	// Each message occupies at least 2 lines (label + content).
	const avgLinesPerMsg = 3
	targetLine := idx * avgLinesPerMsg
	m.vp.SetYOffset(targetLine)
}

// navigateHistoryPrev moves one step backward in input history.
// History is stored newest-first (index 0 = most recent).
func (m ClaudePanelModel) navigateHistoryPrev() ClaudePanelModel {
	if m.history == nil || m.history.Len() == 0 {
		return m
	}
	// Capture the current draft on first navigation.
	if m.historyIdx == -1 {
		m.draftInput = m.input.Value()
		m.historyIdx = 0 // newest entry
	} else if m.historyIdx < m.history.Len()-1 {
		m.historyIdx++ // move toward older entries
	}
	m.input.SetValue(m.history.Get(m.historyIdx))
	m.input.CursorEnd()
	return m
}

// navigateHistoryNext moves one step forward in input history (toward newer,
// or restores the draft when past the newest entry).
func (m ClaudePanelModel) navigateHistoryNext() ClaudePanelModel {
	if m.historyIdx == -1 {
		return m
	}
	if m.historyIdx > 0 {
		m.historyIdx-- // move toward newer entries
		m.input.SetValue(m.history.Get(m.historyIdx))
		m.input.CursorEnd()
	} else {
		// Past the newest: restore draft.
		m.historyIdx = -1
		m.input.SetValue(m.draftInput)
		m.input.CursorEnd()
	}
	return m
}

// ---------------------------------------------------------------------------
// Slash command helpers (TUI-054)
// ---------------------------------------------------------------------------

// updateSlashDropdown checks the current input value and shows, filters, or
// hides the slash command dropdown accordingly. It is called after every
// key event that may have changed the input text.
func (m *ClaudePanelModel) updateSlashDropdown() {
	val := m.input.Value()
	if !strings.HasPrefix(val, "/") {
		m.slashCmd.Hide()
		return
	}
	// Strip the leading "/" for the filter query.
	query := strings.TrimPrefix(val, "/")
	if m.slashCmd.IsVisible() {
		m.slashCmd.Filter(query)
	} else {
		m.slashCmd.Show(query)
	}
	// Propagate the current width so the dropdown renders at the right size.
	m.slashCmd.SetWidth(m.width)
}

// executeSlashCommand parses and executes the given slash command string.
// cmd may include optional arguments after the command name (e.g. "/explore foo").
// It returns the updated model and any commands to run.
func (m ClaudePanelModel) executeSlashCommand(cmd string) (ClaudePanelModel, tea.Cmd) {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	command := parts[0] // includes the "/" prefix
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	switch command {
	case "/exit", "/quit":
		return m, func() tea.Msg {
			return model.ShutdownRequestMsg{}
		}

	case "/clear":
		m.messages = nil
		m.syncViewport()
		return m, func() tea.Msg {
			return model.SlashExecutedMsg{Command: command, Args: args, IsLocal: true}
		}

	case "/cwd":
		return m, func() tea.Msg {
			return model.OpenCWDSelectorMsg{}
		}

	case "/model":
		// Emit a message for AppModel to handle — it owns ProviderState and
		// the CLI driver restart flow. The panel stays decoupled.
		return m, func() tea.Msg {
			return model.ModelSwitchRequestMsg{ModelID: args}
		}

	case "/help":
		helpText := slashcmd.HelpText()
		m.messages = append(m.messages, DisplayMessage{
			Role:      "system",
			Content:   helpText,
			Timestamp: time.Now(),
		})
		m.syncViewport()
		if m.autoScroll {
			m.vp.GotoBottom()
		}
		return m, func() tea.Msg {
			return model.SlashExecutedMsg{Command: command, Args: args, IsLocal: true}
		}

	default:
		// Remote command — forward the full slash invocation to the CLI.
		text := command
		if args != "" {
			text = command + " " + args
		}
		var cmds []tea.Cmd
		if m.sender != nil {
			cmds = append(cmds, m.sender.SendMessage(text))
		}
		cmds = append(cmds, func() tea.Msg {
			return model.SlashExecutedMsg{Command: command, Args: args, IsLocal: false}
		})
		return m, tea.Batch(cmds...)
	}
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

	conv := m.ViewConversation()
	input := m.ViewInput()
	composed := lipgloss.JoinVertical(lipgloss.Left, conv, input)
	return m.ApplyOverlay(composed)
}

// ViewConversation returns the scrollable viewport with optional search bar.
// It includes the streaming indicator ("...") appended to viewport content
// when a response is being streamed. This is everything above the input line.
func (m ClaudePanelModel) ViewConversation() string {
	if m.search.IsActive() {
		searchView := m.search.View()
		searchH := lipgloss.Height(searchView)
		vpCopy := m.vp
		if vpCopy.Height > searchH {
			vpCopy.Height -= searchH
		}
		vpView := vpCopy.View()
		sb := scrollbar.Render(vpCopy.Height, vpCopy.TotalLineCount(), vpCopy.YOffset)
		if sb != "" {
			vpView = lipgloss.JoinHorizontal(lipgloss.Top, vpView, sb)
		}
		return lipgloss.JoinVertical(lipgloss.Left, searchView, vpView)
	}
	return m.viewportWithScrollbar()
}

// ViewInput returns the input prompt and text input only.
// It does not include the streaming indicator or the slash command dropdown.
func (m ClaudePanelModel) ViewInput() string {
	prompt := inputPromptStyle.Render("› ")
	return prompt + m.input.View()
}

// ApplyOverlay takes a fully composed panel string (conversation joined with
// input) and applies the slash command dropdown overlay when it is visible.
// The dropdown floats over the viewport content just above the input line.
// When no dropdown is active the composed string is returned unchanged.
func (m ClaudePanelModel) ApplyOverlay(composed string) string {
	if !m.slashCmd.IsVisible() {
		return composed
	}

	dropdownView := m.slashCmd.View()
	dropdownH := lipgloss.Height(dropdownView)
	inputH := lipgloss.Height(m.ViewInput())

	// Split the composed string into lines and overlay the dropdown on the
	// lines just above the input line.
	lines := strings.Split(composed, "\n")
	totalLines := len(lines)
	// The dropdown replaces lines ending at (totalLines - inputH - 1),
	// i.e. the bottom of the viewport region.
	overlayEnd := totalLines - inputH
	overlayStart := overlayEnd - dropdownH
	if overlayStart < 0 {
		overlayStart = 0
	}

	dropdownLines := strings.Split(dropdownView, "\n")
	for i, dl := range dropdownLines {
		idx := overlayStart + i
		if idx >= 0 && idx < overlayEnd {
			lines[idx] = dl
		}
	}
	return strings.Join(lines, "\n")
}

// viewportWithScrollbar renders the viewport and, when content overflows,
// joins a single-column scrollbar on the right edge.
func (m ClaudePanelModel) viewportWithScrollbar() string {
	vpView := m.vp.View()
	sb := scrollbar.Render(m.vp.Height, m.vp.TotalLineCount(), m.vp.YOffset)
	if sb == "" {
		return vpView
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, vpView, sb)
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
			// RenderMarkdown logs warnings internally when Glamour fails and
			// gracefully falls back to the original plain text.
			rendered, _ := util.RenderMarkdown(msg.Content, m.vp.Width)
			rendered = strings.TrimRight(rendered, "\n") + "\n"
			sb.WriteString(rendered)
		} else {
			sb.WriteString(msg.Content)
			sb.WriteByte('\n')
		}
	}

	// Render tool blocks inline only in compact mode (width < 80).
	// In standard/wide/ultra tiers the Activity panel is visible and
	// tool blocks are shown there instead (TUI-L05).
	if m.tier == model.LayoutCompact {
		for _, tb := range msg.ToolBlocks {
			sb.WriteString(renderToolBlock(tb, m.vp.Width))
		}
	}

	return sb.String()
}

// extractToolDisplayInput parses rawInput as JSON and returns a meaningful
// single-line summary of the key parameter in priority order:
// file_path > path > command > pattern > query > url > description.
// Falls back to rawInput unchanged when parsing fails or no known field exists.
func extractToolDisplayInput(rawInput string) string {
	if rawInput == "" {
		return ""
	}
	var fields struct {
		FilePath    string `json:"file_path"`
		Path        string `json:"path"`
		Command     string `json:"command"`
		Pattern     string `json:"pattern"`
		Query       string `json:"query"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(rawInput), &fields); err != nil {
		return rawInput
	}
	switch {
	case fields.FilePath != "":
		return fields.FilePath
	case fields.Path != "":
		return fields.Path
	case fields.Command != "":
		return util.Truncate(fields.Command, 80)
	case fields.Pattern != "":
		return fields.Pattern
	case fields.Query != "":
		return fields.Query
	case fields.URL != "":
		return fields.URL
	case fields.Description != "":
		return util.Truncate(fields.Description, 80)
	}
	return rawInput
}

// renderToolBlock renders a ToolBlock either collapsed (just the name) or
// expanded (name + input + output).
func renderToolBlock(tb ToolBlock, _ int) string {
	// Status prefix: ✓ for success, ✗ for failure, empty while pending.
	prefix := "  "
	if tb.Success != nil {
		if *tb.Success {
			prefix = "  ✓ "
		} else {
			prefix = "  ✗ "
		}
	}

	if !tb.Expanded {
		if tb.Input != "" {
			display := extractToolDisplayInput(tb.Input)
			return prefix + toolNameStyle.Render(fmt.Sprintf("[%s]", tb.Name)) +
				" " + config.StyleSubtle.Render(display) + "\n"
		}
		return prefix + toolNameStyle.Render(fmt.Sprintf("[%s]", tb.Name)) + "\n"
	}
	var sb strings.Builder
	sb.WriteString(prefix + toolNameStyle.Render(fmt.Sprintf("[%s]", tb.Name)))
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

	vpW := width
	if width > 20 {
		vpW = width - 1 // reserve one column for the scrollbar
	}
	m.vp.Width = vpW
	m.vp.Height = vpH
	m.input.Width = width - 3 // subtract prompt width ("› ")

	// Re-sync so content is re-wrapped to the new width.
	m.syncViewport()

	if m.autoScroll {
		m.vp.GotoBottom()
	}
}

// SetFocused sets whether the panel accepts keyboard input.  When focused,
// the textinput cursor is shown.  If the search overlay is active, focusing
// the panel does not steal focus from the search input.
func (m *ClaudePanelModel) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		if !m.search.IsActive() {
			m.input.Focus()
		}
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
// SetCLIDriver / SetBridge pattern used in AppModel.  The parameter type is
// model.MessageSender so that claudePanelWidget (defined in the model package)
// and this concrete method share the same named type, satisfying Go's strict
// interface method-signature matching.
func (m *ClaudePanelModel) SetSender(sender model.MessageSender) {
	m.sender = sender
}

// SetHistory injects a loaded InputHistory for cross-session prompt
// persistence. Called from main.go after loading from disk.
func (m *ClaudePanelModel) SetHistory(h *InputHistory) {
	m.history = h
}

// SetTier stores the current layout tier so renderMessage can conditionally
// render inline tool blocks only in compact mode (TUI-L05).
func (m *ClaudePanelModel) SetTier(tier model.LayoutTier) {
	m.tier = tier
}

// IsStreaming returns true while an assistant response is being streamed.
func (m ClaudePanelModel) IsStreaming() bool {
	return m.streaming
}

// refreshViewport rebuilds the viewport content string from the current
// messages slice.  It is a thin wrapper around syncViewport that is safe to
// call from pointer-receiver methods.
func (m *ClaudePanelModel) refreshViewport() {
	m.syncViewport()
	if m.autoScroll {
		m.vp.GotoBottom()
	}
}

// SaveMessages returns a snapshot of the current conversation as
// state.DisplayMessage values. ToolBlocks are preserved so they survive
// provider switches (TUI R-4). The transient Expanded field is excluded;
// all blocks start collapsed on restore.
func (m *ClaudePanelModel) SaveMessages() []state.DisplayMessage {
	if len(m.messages) == 0 {
		return nil
	}
	// Types are identical (aliases) — shallow copy is sufficient.
	// Reset Expanded to false so restored conversations start collapsed.
	result := make([]state.DisplayMessage, len(m.messages))
	copy(result, m.messages)
	for i := range result {
		if len(result[i].ToolBlocks) == 0 {
			result[i].ToolBlocks = nil // preserve nil semantics
			continue
		}
		blocks := make([]state.ToolBlock, len(result[i].ToolBlocks))
		copy(blocks, result[i].ToolBlocks)
		for j := range blocks {
			blocks[j].Expanded = false
		}
		result[i].ToolBlocks = blocks
	}
	return result
}

// RestoreMessages replaces the conversation history with the given messages,
// resets streaming state, enables auto-scroll, and redraws the viewport.
// Passing nil or an empty slice clears the conversation. ToolBlocks are
// restored with Expanded=false so they always start collapsed (TUI R-4).
func (m *ClaudePanelModel) RestoreMessages(msgs []state.DisplayMessage) {
	// Types are identical (aliases) — copy and reset Expanded to false.
	m.messages = make([]DisplayMessage, len(msgs))
	copy(m.messages, msgs)
	for i := range m.messages {
		if len(m.messages[i].ToolBlocks) == 0 {
			m.messages[i].ToolBlocks = nil
			continue
		}
		blocks := make([]state.ToolBlock, len(m.messages[i].ToolBlocks))
		copy(blocks, m.messages[i].ToolBlocks)
		for j := range blocks {
			blocks[j].Expanded = false
		}
		m.messages[i].ToolBlocks = blocks
	}
	m.streaming = false
	m.autoScroll = true
	m.refreshViewport()
}

// AppendSystemMessage adds a system-role message to the conversation history
// and scrolls to bottom. Used by AppModel for informational output such as
// the /model listing.
func (m *ClaudePanelModel) AppendSystemMessage(text string) {
	m.messages = append(m.messages, DisplayMessage{
		Role:      "system",
		Content:   text,
		Timestamp: time.Now(),
	})
	m.syncViewport()
	if m.autoScroll {
		m.vp.GotoBottom()
	}
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

// Search implements state.SearchSource for the conversation history.
//
// It performs a case-insensitive substring search across all message contents
// and returns matching state.SearchResult values sorted by relevance.
// An empty query always returns nil.
func (m *ClaudePanelModel) Search(query string) []state.SearchResult {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var results []state.SearchResult
	for i, msg := range m.messages {
		content := strings.ToLower(msg.Content)
		if !strings.Contains(content, q) {
			continue
		}
		// Score: prefix match scores higher than interior match.
		score := 100
		if strings.HasPrefix(content, q) {
			score = 200
		}
		label := truncateForSearch(msg.Content, 60)
		results = append(results, state.SearchResult{
			Source: "conversation",
			Label:  fmt.Sprintf("[%s] %s", msg.Role, label),
			Detail: fmt.Sprintf("Message %d", i+1),
			Score:  score,
		})
	}
	return results
}

// truncateForSearch truncates s to at most max runes, appending "…" when
// truncated. Used to produce concise search result labels.
func truncateForSearch(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
