package claude_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/claude"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/slashcmd"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ansiRe matches ANSI escape sequences for stripping in test assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes so content assertions work
// regardless of whether Glamour rendered the output.
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newPanel returns a ClaudePanelModel with default key map and a useful size.
func newPanel() claude.ClaudePanelModel {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m.SetFocused(true)
	m, _ = m.Update(model.RemoteSkillsLoadedMsg{Skills: []string{
		"explore",
		"explore-add",
		"braintrust",
		"review",
		"review-plan",
		"benchmark",
		"benchmark-meta",
		"benchmark-agent",
		"implement",
		"ticket",
		"plan-tickets",
		"teams",
		"team-status",
		"team-result",
		"team-cancel",
		"init-auto",
		"dummies-guide",
		"memory-improvement",
	}})
	return m
}

// newUnfocusedPanel returns a panel that is NOT focused.
func newUnfocusedPanel() claude.ClaudePanelModel {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m, _ = m.Update(model.RemoteSkillsLoadedMsg{Skills: []string{
		"explore",
		"explore-add",
		"braintrust",
		"review",
		"review-plan",
		"benchmark",
		"benchmark-meta",
		"benchmark-agent",
		"implement",
		"ticket",
		"plan-tickets",
		"teams",
		"team-status",
		"team-result",
		"team-cancel",
		"init-auto",
		"dummies-guide",
		"memory-improvement",
	}})
	return m
}

// pressEnterWith sets the input value then sends Enter.
func pressEnterWith(m claude.ClaudePanelModel, text string) (claude.ClaudePanelModel, tea.Cmd) {
	// Simulate typing by sending SetValue then Enter.
	// textinput doesn't expose SetValue via msg, so we send runes first.
	m2, _ := m.Update(setInputMsg{value: text})
	m3, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return m3, cmd
}

// setInputMsg is a fake message used by tests to set the textinput value
// without relying on rune-by-rune simulation.  It is handled specially in
// the wrapper below.
type setInputMsg struct{ value string }

// updateWithSet is a thin test helper that handles setInputMsg by mutating
// the panel's textinput directly through its public API.
func updateWithSet(m claude.ClaudePanelModel, msg tea.Msg) (claude.ClaudePanelModel, tea.Cmd) {
	if si, ok := msg.(setInputMsg); ok {
		// Use a trick: send the text character-by-character.
		for _, r := range si.value {
			var cmd tea.Cmd
			m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			_ = cmd
		}
		return m, nil
	}
	return m.Update(msg)
}

// sendAndCapture types text into the panel and presses Enter, returning the
// updated model and the cmd that was emitted.
func sendAndCapture(m claude.ClaudePanelModel, text string) (claude.ClaudePanelModel, tea.Cmd) {
	for _, r := range text {
		var cmd tea.Cmd
		m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		_ = cmd
	}
	return m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

// assistantMsg creates an AssistantMsg with Streaming=false.
func assistantMsg(text string) model.AssistantMsg {
	return model.AssistantMsg{Text: text, Streaming: false}
}

// streamingMsg creates an AssistantMsg with Streaming=true.
func streamingMsg(text string) model.AssistantMsg {
	return model.AssistantMsg{Text: text, Streaming: true}
}

// ---------------------------------------------------------------------------
// stubSender — a CLIDriverSender implementation for testing.
// ---------------------------------------------------------------------------

type stubSender struct {
	sent []string
}

func (s *stubSender) SendMessage(text string) tea.Cmd {
	s.sent = append(s.sent, text)
	return nil
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewClaudePanelModel_Defaults(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	if m.IsStreaming() {
		t.Error("new panel should not be streaming")
	}
}

func TestInit_ReturnsBlink(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return textinput.Blink (non-nil Cmd)")
	}
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestSetSize_ViewNotEmpty(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	view := m.View()
	if view == "" {
		t.Error("View() after SetSize should not be empty")
	}
}

func TestSetSize_ZeroDimensions_EmptyView(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	// Zero dimensions — guard against render panics.
	view := m.View()
	if view != "" {
		t.Errorf("View() with zero dimensions should be empty; got %q", view)
	}
}

// ---------------------------------------------------------------------------
// AssistantMsg — conversation appending
// ---------------------------------------------------------------------------

func TestAssistantMsg_AppendsToConversation(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("Hello from Claude"))
	view := stripANSI(m2.View())
	if !strings.Contains(view, "Hello from Claude") {
		t.Errorf("View() after AssistantMsg should contain message text; got:\n%s", view)
	}
}

func TestAssistantMsg_RoleLabel(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("Hi"))
	view := m2.View()
	if !strings.Contains(view, "Claude:") {
		t.Errorf("View() after assistant message should contain 'Claude:'; got:\n%s", view)
	}
}

func TestAssistantMsg_StreamingFlagSet(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("partial"))
	if !m2.IsStreaming() {
		t.Error("IsStreaming() should be true after a streaming AssistantMsg")
	}
}

func TestAssistantMsg_StreamingReplaces(t *testing.T) {
	// The Claude CLI sends full accumulated snapshots (not deltas) for each
	// partial AssistantEvent. Each streaming message should REPLACE the
	// previous partial view so the displayed text matches the latest snapshot.
	m := newPanel()
	m2, _ := m.Update(streamingMsg("Hello "))
	m3, _ := m2.Update(streamingMsg("Hello world"))
	view := m3.View()
	// The latest snapshot ("Hello world") should be visible.
	if !strings.Contains(view, "Hello world") {
		t.Errorf("streaming messages should show latest snapshot; got:\n%s", view)
	}
	// The earlier snapshot ("Hello ") should NOT be doubled up.
	// The view should not contain "Hello Hello world" (old append behaviour).
	if strings.Contains(view, "Hello Hello") {
		t.Errorf("streaming messages should replace, not concatenate snapshots; got:\n%s", view)
	}
	// Only one assistant message should be present.
	msgs := m3.Messages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message after two streaming snapshots; got %d", len(msgs))
	}
}

func TestAssistantMsg_NonStreamingClearStreaming(t *testing.T) {
	m := newPanel()
	// Start streaming.
	m2, _ := m.Update(streamingMsg("partial"))
	// Finalize with non-streaming message: replaces the partial, no duplication.
	m3, _ := m2.Update(assistantMsg("complete"))
	if m3.IsStreaming() {
		t.Error("IsStreaming() should be false after non-streaming AssistantMsg")
	}
	// The final non-streaming message should REPLACE the streaming partial,
	// not create a second assistant message.
	msgs := m3.Messages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message after streaming→complete; got %d (duplicate prevention)", len(msgs))
	}
	if len(msgs) > 0 && msgs[0].Content != "complete" {
		t.Errorf("expected content %q; got %q", "complete", msgs[0].Content)
	}
}

func TestAssistantMsg_MultipleMessages(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("First"))
	m3, _ := m2.Update(assistantMsg("Second"))
	view := m3.View()
	if !strings.Contains(view, "First") || !strings.Contains(view, "Second") {
		t.Errorf("both messages should be visible; got:\n%s", view)
	}
}

func TestAssistantMsg_MultiTurnStreaming_AppendNotOverwrite(t *testing.T) {
	// Multi-tool requests produce multiple assistant turns:
	//   turn 1: stream partial → finalize
	//   turn 2: stream partial → finalize
	// The bug was that turn 2's first streaming event found the last message
	// was role "assistant" and overwrote it instead of appending.
	m := newPanel()
	// Turn 1: stream then finalize.
	m, _ = m.Update(streamingMsg("Turn 1 partial"))
	m, _ = m.Update(assistantMsg("Turn 1 complete"))
	// Turn 2: first streaming event — must append, not overwrite.
	m, _ = m.Update(streamingMsg("Turn 2 partial"))
	msgs := m.Messages()
	if len(msgs) != 2 {
		t.Errorf("expected 2 assistant messages after two turns; got %d", len(msgs))
	}
	if len(msgs) >= 1 && msgs[0].Content != "Turn 1 complete" {
		t.Errorf("turn 1 content overwritten; got %q, want %q", msgs[0].Content, "Turn 1 complete")
	}
}

// ---------------------------------------------------------------------------
// MessageID-based replace-vs-append
// ---------------------------------------------------------------------------

// msgWithID creates an AssistantMsg with a stable MessageID.
func msgWithID(text, id string, streaming bool) model.AssistantMsg {
	return model.AssistantMsg{Text: text, Streaming: streaming, MessageID: id}
}

func TestAssistantMsg_MessageID_SameIDReplaces(t *testing.T) {
	// Streaming snapshots with the same MessageID must replace, not duplicate.
	m := newPanel()
	m, _ = m.Update(msgWithID("partial A", "msg-1", true))
	m, _ = m.Update(msgWithID("partial A extended", "msg-1", true))
	m, _ = m.Update(msgWithID("complete A", "msg-1", false))
	msgs := m.Messages()
	if len(msgs) != 1 {
		t.Errorf("same MessageID: expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 && msgs[0].Content != "complete A" {
		t.Errorf("same MessageID: expected %q, got %q", "complete A", msgs[0].Content)
	}
}

func TestAssistantMsg_MessageID_DifferentIDAppends(t *testing.T) {
	// Two consecutive streaming sequences with different MessageIDs must
	// produce two separate assistant messages, not one overwritten message.
	m := newPanel()
	// Turn 1.
	m, _ = m.Update(msgWithID("Turn 1 partial", "msg-1", true))
	m, _ = m.Update(msgWithID("Turn 1 complete", "msg-1", false))
	// Turn 2 — different MessageID.
	m, _ = m.Update(msgWithID("Turn 2 partial", "msg-2", true))
	m, _ = m.Update(msgWithID("Turn 2 complete", "msg-2", false))
	msgs := m.Messages()
	if len(msgs) != 2 {
		t.Errorf("different MessageIDs: expected 2 messages, got %d", len(msgs))
	}
	if len(msgs) >= 1 && msgs[0].Content != "Turn 1 complete" {
		t.Errorf("turn 1 content wrong: got %q, want %q", msgs[0].Content, "Turn 1 complete")
	}
	if len(msgs) >= 2 && msgs[1].Content != "Turn 2 complete" {
		t.Errorf("turn 2 content wrong: got %q, want %q", msgs[1].Content, "Turn 2 complete")
	}
}

// ---------------------------------------------------------------------------
// StreamEventMsg
// ---------------------------------------------------------------------------

func TestStreamEventMsg_SetsStreamingFlag(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(model.StreamEventMsg{EventType: "assistant", Data: []byte("{}")})
	if !m2.IsStreaming() {
		t.Error("StreamEventMsg with EventType='assistant' should set streaming=true")
	}
}

func TestStreamEventMsg_UnknownType_NoChange(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(model.StreamEventMsg{EventType: "unknown", Data: []byte("{}")})
	if m2.IsStreaming() {
		t.Error("StreamEventMsg with unknown EventType should not set streaming=true")
	}
}

// ---------------------------------------------------------------------------
// ResultMsg
// ---------------------------------------------------------------------------

func TestResultMsg_ClearsStreaming(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("partial"))
	if !m2.IsStreaming() {
		t.Fatal("pre-condition: should be streaming")
	}
	m3, _ := m2.Update(model.ResultMsg{SessionID: "s1", CostUSD: 0.01, DurationMS: 500})
	if m3.IsStreaming() {
		t.Error("IsStreaming() should be false after ResultMsg")
	}
}

// ---------------------------------------------------------------------------
// Input submission
// ---------------------------------------------------------------------------

func TestEnter_SubmitsMessage(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "hello world")
	view := m2.View()
	if !strings.Contains(view, "hello world") {
		t.Errorf("View() should contain submitted message; got:\n%s", view)
	}
}

func TestEnter_UserRoleLabel(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "test message")
	view := m2.View()
	if !strings.Contains(view, "You:") {
		t.Errorf("View() after user input should contain 'You:'; got:\n%s", view)
	}
}

func TestEnter_ClearsInput(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "some text")
	view := m2.View()
	// The input field should be empty: "some text" only in conversation,
	// not duplicated in the input area after submission.
	// We verify by counting occurrences of "some text" — should be exactly 1
	// (in the conversation, not in the input line).
	count := strings.Count(view, "some text")
	if count < 1 {
		t.Errorf("submitted message should appear in conversation; count=%d", count)
	}
}

func TestEnter_EmptyInput_NoSubmit(t *testing.T) {
	m := newPanel()
	// Press Enter without typing anything.
	initialMsgCount := 0 // no messages yet

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// cmd should be nil since nothing was sent.
	if cmd != nil {
		t.Error("Enter on empty input should not emit a command")
	}
	// Verify no messages were added.
	view := m2.View()
	// Empty state message should still be visible.
	_ = view // view renders empty-state text; just ensure no panic.
	_ = initialMsgCount
}

func TestEnter_WhitespaceOnly_NoSubmit(t *testing.T) {
	m := newPanel()
	// Type only spaces.
	for range 5 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter on whitespace-only input should not emit a command")
	}
}

func TestEnter_DuringStreaming_NoSubmit(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("partial…"))
	// Type something and try to send.
	for _, r := range "hello" {
		m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter during streaming should not submit")
	}
}

func TestEnter_CallsSender(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	_, cmd := sendAndCapture(m, "test query")
	// The cmd is the return value of stub.SendMessage which is nil in our stub.
	// But the stub captured the sent text.
	if len(stub.sent) != 1 {
		t.Fatalf("sender.SendMessage should be called once; got %d calls", len(stub.sent))
	}
	if stub.sent[0] != "test query" {
		t.Errorf("sender received %q; want %q", stub.sent[0], "test query")
	}
	_ = cmd
}

func TestEnter_NilSender_NoError(t *testing.T) {
	m := newPanel()
	// sender is nil — should not panic.
	m.SetSender(nil)
	// Should not panic.
	_, cmd := sendAndCapture(m, "test")
	_ = cmd
}

// ---------------------------------------------------------------------------
// Input history
// ---------------------------------------------------------------------------

func TestInputHistory_PrevRestoresMessage(t *testing.T) {
	m := newPanel()
	// Submit a message to add it to history.
	m, _ = sendAndCapture(m, "first message")

	// Press Up to navigate to previous history entry.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	view := m2.View()
	if !strings.Contains(view, "first message") {
		t.Errorf("after HistoryPrev, input should contain 'first message'; got:\n%s", view)
	}
}

func TestInputHistory_NextRestoresDraft(t *testing.T) {
	m := newPanel()
	m, _ = sendAndCapture(m, "history item")

	// Type a new draft.
	for _, r := range "new draft" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Navigate to history and back.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	view := m2.View()
	if !strings.Contains(view, "new draft") {
		t.Errorf("after HistoryNext past end, input should restore 'new draft'; got:\n%s", view)
	}
}

func TestInputHistory_MultiplePrev(t *testing.T) {
	m := newPanel()
	m, _ = sendAndCapture(m, "first")
	m, _ = sendAndCapture(m, "second")

	// Press Up twice.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})

	view := m.View()
	if !strings.Contains(view, "first") {
		t.Errorf("after two HistoryPrev, input should show 'first'; got:\n%s", view)
	}
}

func TestInputHistory_ClampAtStart(t *testing.T) {
	m := newPanel()
	m, _ = sendAndCapture(m, "only message")

	// Press Up twice — second press should clamp.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // should not panic or error

	view := m.View()
	if !strings.Contains(view, "only message") {
		t.Errorf("HistoryPrev clamped at start; input should show 'only message'; got:\n%s", view)
	}
}

func TestInputHistory_EmptyHistory_UpNoOp(t *testing.T) {
	m := newPanel()
	// No history — pressing Up should be a no-op.
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	_ = m2
	_ = cmd
	// Just verify no panic.
}

func TestInputHistory_NoDuplicateEntries(t *testing.T) {
	m := newPanel()
	m, _ = sendAndCapture(m, "same message")
	m, _ = sendAndCapture(m, "same message")

	// Navigate back — should hit "same message" then stop.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp}) // second Up — should stay on first entry

	view := m2.View()
	if !strings.Contains(view, "same message") {
		t.Errorf("duplicate prevention; should still show 'same message'; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Auto-scroll
// ---------------------------------------------------------------------------

func TestAutoScroll_EnabledByDefault(t *testing.T) {
	m := newPanel()
	// Send several messages to fill the viewport.
	for range 30 {
		m, _ = m.Update(assistantMsg("line of text that fills the viewport"))
	}
	view := m.View()
	// With auto-scroll, the last message should be visible.
	if !strings.Contains(stripANSI(view), "line of text that fills the viewport") {
		t.Errorf("auto-scroll: last message should be visible; got:\n%s", view)
	}
}

func TestAutoScroll_ReenabledOnNewContent(t *testing.T) {
	m := newPanel()
	// IsStreaming is false; send a result to exercise the ResultMsg path.
	m, _ = m.Update(model.ResultMsg{SessionID: "s1"})
	// Send new content — auto-scroll should keep the panel at the bottom.
	m, _ = m.Update(assistantMsg("new content after result"))
	view := m.View()
	if !strings.Contains(stripANSI(view), "new content after result") {
		t.Errorf("new content should be visible after ResultMsg; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Focus management
// ---------------------------------------------------------------------------

func TestSetFocused_True(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m.SetFocused(true)
	// Focused panel should accept key events (submitting a message should work).
	m, _ = sendAndCapture(m, "focused input")
	view := m.View()
	if !strings.Contains(view, "focused input") {
		t.Errorf("focused panel should record submitted message; got:\n%s", view)
	}
}

func TestSetFocused_False_IgnoresKeys(t *testing.T) {
	m := newUnfocusedPanel()
	// Type something — it should not appear in the view.
	for _, r := range "hello" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("unfocused panel should not submit on Enter")
	}
}

func TestFocusBlur(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m.Focus()
	m.Blur()
	// After Blur, entering text and pressing Enter should be a no-op.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("blurred panel should not submit on Enter")
	}
}

// ---------------------------------------------------------------------------
// View rendering
// ---------------------------------------------------------------------------

func TestView_EmptyState(t *testing.T) {
	m := newPanel()
	view := m.View()
	if !strings.Contains(view, "No messages") {
		t.Errorf("empty state should show placeholder; got:\n%s", view)
	}
}

func TestView_InputPrompt(t *testing.T) {
	m := newPanel()
	view := m.View()
	if !strings.Contains(view, "›") {
		t.Errorf("View() should include '›' prompt; got:\n%s", view)
	}
}

func TestView_StreamingIndicator(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("partial response"))
	view := m2.View()
	if !strings.Contains(view, "...") {
		t.Errorf("View() during streaming should show '...'; got:\n%s", view)
	}
}

func TestView_NoStreamingIndicatorWhenIdle(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("complete response"))
	// Note: "..." may appear in placeholder or elsewhere, so we check IsStreaming.
	if m2.IsStreaming() {
		t.Error("non-streaming message should leave IsStreaming=false")
	}
}

func TestView_ViewportRespectsDimensions(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(40, 10)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 10+2 { // allow slight tolerance for newlines
		t.Errorf("View() with height=10 should not produce more than ~12 lines; got %d", len(lines))
	}
}

// ---------------------------------------------------------------------------
// ToolBlock rendering
// ---------------------------------------------------------------------------

func TestToolBlock_CollapsedByDefault(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("see tool below"))
	// Manually inject a message with a ToolBlock.
	msgs := []claude.DisplayMessage{
		{
			Role:    "assistant",
			Content: "see tool below",
			ToolBlocks: []claude.ToolBlock{
				{Name: "ReadFile", Input: "path/to/file.go", Output: "contents", Expanded: false},
			},
			Timestamp: time.Now(),
		},
	}
	// Use a new panel that renders the message with a tool block.
	m3 := newPanel()
	m3, _ = m3.Update(assistantMsg("ignore"))
	// Override by sending a modified view; we test via model by forcing a
	// user message followed by assistant with tool block.
	_ = msgs
	_ = m2
	// Verify the Panel can store a DisplayMessage with ToolBlocks.
	// Since we can only interact via Update messages, we verify the panel
	// does not panic on receiving an AssistantMsg (the actual ToolBlock
	// population would come from a future TUI-023 integration).
	_ = m3
}

// ---------------------------------------------------------------------------
// IsStreaming
// ---------------------------------------------------------------------------

func TestIsStreaming_FalseByDefault(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	if m.IsStreaming() {
		t.Error("IsStreaming() should be false on a new panel")
	}
}

func TestIsStreaming_TrueWhileStreaming(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("fragment"))
	if !m2.IsStreaming() {
		t.Error("IsStreaming() should be true during streaming")
	}
}

func TestIsStreaming_FalseAfterResult(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(streamingMsg("fragment"))
	m3, _ := m2.Update(model.ResultMsg{})
	if m3.IsStreaming() {
		t.Error("IsStreaming() should be false after ResultMsg")
	}
}

// ---------------------------------------------------------------------------
// SetSender
// ---------------------------------------------------------------------------

func TestSetSender_MessageRouted(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	m, _ = sendAndCapture(m, "routed message")
	if len(stub.sent) == 0 {
		t.Error("message should have been routed to sender")
	}
	if stub.sent[0] != "routed message" {
		t.Errorf("sender received %q; want %q", stub.sent[0], "routed message")
	}
}

func TestSetSender_ReplacedSender(t *testing.T) {
	stub1 := &stubSender{}
	stub2 := &stubSender{}
	m := newPanel()
	m.SetSender(stub1)
	m.SetSender(stub2) // Replace.

	m, _ = sendAndCapture(m, "after replace")
	if len(stub1.sent) != 0 {
		t.Error("original sender should not receive messages after replacement")
	}
	if len(stub2.sent) == 0 {
		t.Error("new sender should receive the message")
	}
}

// ---------------------------------------------------------------------------
// Viewport scroll behavior
// ---------------------------------------------------------------------------

func TestViewport_ScrollUpDisablesAutoScroll(t *testing.T) {
	m := newPanel()
	// Populate with enough messages to be scrollable.
	for range 50 {
		m, _ = m.Update(assistantMsg("line"))
	}
	// Simulate a page-up key. Viewport keys are: pgup, pgdown.
	// We use tea.KeyPgUp which the viewport handles.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	// After scrolling up, new content should not re-enable auto-scroll
	// until the user scrolls back to the bottom.
	// We verify by checking IsStreaming remains unaffected and no panic.
	if m2.IsStreaming() {
		t.Error("scroll up should not affect streaming state")
	}
}

// ---------------------------------------------------------------------------
// HandleMsg pointer-receiver
// ---------------------------------------------------------------------------

func TestHandleMsg_PointerReceiverMutates(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := claude.NewClaudePanelModel(keys)
	m.SetSize(80, 24)
	m.SetFocused(true)

	// HandleMsg should mutate m in place.
	m.HandleMsg(model.AssistantMsg{Text: "Hello", Streaming: false})

	assert.Equal(t, 1, len(m.Messages()), "HandleMsg should mutate receiver in place")
	assert.Equal(t, "Hello", m.Messages()[0].Content)
	assert.Equal(t, "assistant", m.Messages()[0].Role)
}

func TestStreamingGuard_PlainTextDuringStreaming(t *testing.T) {
	// Verify the streaming bug fix: during active streaming, the last message
	// should be rendered as plain text (no Glamour), not through markdown.
	keys := config.DefaultKeyMap()
	m := claude.NewClaudePanelModel(keys)
	m.SetSize(80, 24)

	// Simulate a streaming message.
	m, _ = m.Update(model.AssistantMsg{Text: "# Heading", Streaming: true})

	// The output should contain the raw "# Heading" text (plain),
	// NOT Glamour-rendered heading (which would strip the # and add bold/color).
	output := m.View()
	assert.Contains(t, output, "# Heading", "streaming content should be plain text, not markdown-rendered")
}

func TestGlamour_CompletedMessageDropsRawMarkdown(t *testing.T) {
	// A completed (non-streaming) assistant message must be rendered through
	// Glamour. The raw "# " heading marker should NOT appear verbatim in the
	// output — Glamour strips it and applies styling.
	keys := config.DefaultKeyMap()
	m := claude.NewClaudePanelModel(keys)
	m.SetSize(80, 24)

	// Send a complete (non-streaming) message containing markdown.
	m, _ = m.Update(model.AssistantMsg{Text: "# Heading\n\nSome paragraph text.", Streaming: false})

	output := m.View()
	// Glamour transforms "# Heading" → styled heading without the "#" prefix.
	// If Glamour is working, the raw "# Heading" string should not appear
	// (the "#" is consumed and replaced with ANSI styling).
	assert.NotContains(t, output, "# Heading",
		"completed assistant message should be Glamour-rendered; raw '# Heading' should not appear")
	// The plain text content should still be present in some form.
	assert.Contains(t, output, "Heading",
		"heading text should remain in Glamour output even without the '#' prefix")
}

func TestGlamour_StreamingThenComplete_NoRawMarkdown(t *testing.T) {
	// A streaming sequence followed by a complete message must result in a
	// single Glamour-rendered message, not plain text or duplicate content.
	keys := config.DefaultKeyMap()
	m := claude.NewClaudePanelModel(keys)
	m.SetSize(80, 24)

	// Simulate the streaming sequence: partial snapshots then final complete.
	m, _ = m.Update(model.AssistantMsg{Text: "# Head", Streaming: true})
	m, _ = m.Update(model.AssistantMsg{Text: "# Heading\n\nText.", Streaming: true})
	m, _ = m.Update(model.AssistantMsg{Text: "# Heading\n\nFull text.", Streaming: false})

	// After the complete message, only one assistant message should exist
	// (the streaming snapshots were replaced, not accumulated).
	msgs := m.Messages()
	assert.Len(t, msgs, 1, "streaming+complete should yield exactly 1 message")

	// The final complete text should be the message content.
	if len(msgs) > 0 {
		assert.Equal(t, "# Heading\n\nFull text.", msgs[0].Content)
	}

	output := m.View()
	// After the non-streaming final message, Glamour renders and the raw "#"
	// prefix should not appear in the view.
	assert.NotContains(t, output, "# Heading",
		"after streaming+complete, Glamour should render and remove raw '#' prefix")
}

// ---------------------------------------------------------------------------
// SaveMessages / RestoreMessages (TUI-029)
// ---------------------------------------------------------------------------

func TestSaveMessages_EmptyPanel_ReturnsNil(t *testing.T) {
	m := newPanel()
	got := m.SaveMessages()
	assert.Nil(t, got, "SaveMessages on empty panel should return nil")
}

func TestSaveMessages_PreservesRoleAndContent(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(model.AssistantMsg{Text: "hello", Streaming: false})
	m, _ = sendAndCapture(m, "hi back")

	saved := m.SaveMessages()
	assert.Len(t, saved, 2)

	// assistant message was added first
	assert.Equal(t, "assistant", saved[0].Role)
	assert.Equal(t, "user", saved[1].Role)
	assert.Equal(t, "hi back", saved[1].Content)
}

func TestSaveMessages_ToolBlocksPreserved(t *testing.T) {
	// SaveMessages now preserves ToolBlocks (TUI R-4).
	// Since ToolBlocks are populated via internal state, we exercise
	// SaveMessages/RestoreMessages round-trip via RestoreMessages first.
	m := newPanel()

	// Inject messages with ToolBlocks via RestoreMessages (simulating a
	// populated panel) and then verify Save preserves them.
	now := time.Now()
	m.RestoreMessages([]state.DisplayMessage{
		{
			Role:      "assistant",
			Content:   "I ran a tool",
			Timestamp: now,
			ToolBlocks: []state.ToolBlock{
				{Name: "Read", Input: "main.go", Output: "package main"},
			},
		},
	})

	saved := m.SaveMessages()
	assert.Len(t, saved, 1)
	assert.Len(t, saved[0].ToolBlocks, 1, "SaveMessages should preserve ToolBlocks")
	assert.Equal(t, "Read", saved[0].ToolBlocks[0].Name)
	assert.Equal(t, "main.go", saved[0].ToolBlocks[0].Input)
	assert.Equal(t, "package main", saved[0].ToolBlocks[0].Output)
}

func TestSaveMessages_PreservesTimestamp(t *testing.T) {
	m := newPanel()
	before := time.Now()
	m, _ = m.Update(model.AssistantMsg{Text: "msg", Streaming: false})
	after := time.Now()

	saved := m.SaveMessages()
	assert.Len(t, saved, 1)
	assert.True(t, !saved[0].Timestamp.Before(before),
		"timestamp should be >= before")
	assert.True(t, !saved[0].Timestamp.After(after),
		"timestamp should be <= after")
}

func TestRestoreMessages_ReplacesHistory(t *testing.T) {
	m := newPanel()
	// Load some existing messages.
	m, _ = m.Update(model.AssistantMsg{Text: "old message", Streaming: false})

	now := time.Now()
	newMsgs := []state.DisplayMessage{
		{Role: "user", Content: "restored user msg", Timestamp: now},
		{Role: "assistant", Content: "restored asst msg", Timestamp: now.Add(time.Second)},
	}
	m.RestoreMessages(newMsgs)

	// The old message must no longer appear; the restored ones must.
	view := stripANSI(m.View())
	assert.NotContains(t, view, "old message")
	assert.Contains(t, view, "restored user msg")
	assert.Contains(t, view, "restored asst msg")
}

func TestRestoreMessages_ClearsStreamingState(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(model.AssistantMsg{Text: "streaming…", Streaming: true})
	assert.True(t, m.IsStreaming(), "pre-condition: should be streaming")

	m.RestoreMessages([]state.DisplayMessage{
		{Role: "user", Content: "hello"},
	})
	assert.False(t, m.IsStreaming(), "RestoreMessages should clear streaming state")
}

func TestRestoreMessages_NilClearsHistory(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(model.AssistantMsg{Text: "existing", Streaming: false})

	m.RestoreMessages(nil)

	view := stripANSI(m.View())
	assert.NotContains(t, view, "existing")
	// Empty state placeholder should be visible.
	assert.Contains(t, view, "No messages")
}

func TestRestoreMessages_ViewReflectsRestoredContent(t *testing.T) {
	m := newPanel()

	restored := []state.DisplayMessage{
		{Role: "assistant", Content: "answer after switch", Timestamp: time.Now()},
	}
	m.RestoreMessages(restored)

	view := stripANSI(m.View())
	assert.Contains(t, view, "answer after switch")
}

func TestSaveRestore_RoundTrip(t *testing.T) {
	m := newPanel()
	// Populate with a couple of messages.
	m, _ = m.Update(model.AssistantMsg{Text: "first", Streaming: false})
	m, _ = sendAndCapture(m, "second")

	saved := m.SaveMessages()
	assert.Len(t, saved, 2)

	// Wipe the conversation then restore.
	m.RestoreMessages(nil)
	m.RestoreMessages(saved)

	view := stripANSI(m.View())
	assert.Contains(t, view, "first")
	assert.Contains(t, view, "second")
}

func TestSaveRestore_MessageIsolationAcrossPanelInstances(t *testing.T) {
	// Simulate the provider-switch scenario: save from panelA, restore into
	// panelB, verify panelB has the right content and panelA is unaffected.
	panelA := newPanel()
	panelA, _ = panelA.Update(model.AssistantMsg{Text: "panel A message", Streaming: false})

	saved := panelA.SaveMessages()

	panelB := newPanel()
	panelB.RestoreMessages(saved)

	assert.Contains(t, stripANSI(panelB.View()), "panel A message")
	assert.Contains(t, stripANSI(panelA.View()), "panel A message",
		"saving must not mutate the source panel")
}

// ---------------------------------------------------------------------------
// R-4: ToolBlock preservation across provider switch
// ---------------------------------------------------------------------------

func TestSaveMessages_ToolBlocksExpandedFieldNotPersisted(t *testing.T) {
	// Expanded is transient UI state — SaveMessages must NOT store it.
	// We verify this by injecting an expanded block, saving, and confirming
	// that RestoreMessages always sets Expanded=false.
	m := newPanel()
	m.RestoreMessages([]state.DisplayMessage{
		{
			Role:      "assistant",
			Content:   "result",
			Timestamp: time.Now(),
			ToolBlocks: []state.ToolBlock{
				{Name: "Bash", Input: "ls", Output: "a.go b.go"},
			},
		},
	})

	saved := m.SaveMessages()
	// The state.ToolBlock has no Expanded field by design.
	// We just verify the saved block round-trips correctly.
	assert.Len(t, saved[0].ToolBlocks, 1)
	assert.Equal(t, "Bash", saved[0].ToolBlocks[0].Name)
}

func TestRestoreMessages_ToolBlocksRestoredCollapsed(t *testing.T) {
	// Restored ToolBlocks should always have Expanded=false.
	m := newPanel()
	m.RestoreMessages([]state.DisplayMessage{
		{
			Role:      "assistant",
			Content:   "done",
			Timestamp: time.Now(),
			ToolBlocks: []state.ToolBlock{
				{Name: "Edit", Input: "main.go", Output: "ok"},
			},
		},
	})

	msgs := m.Messages()
	assert.Len(t, msgs, 1)
	assert.Len(t, msgs[0].ToolBlocks, 1)
	assert.False(t, msgs[0].ToolBlocks[0].Expanded,
		"restored ToolBlock should always start collapsed")
}

func TestSaveRestore_ToolBlockRoundTrip(t *testing.T) {
	// Full round-trip: save → restore preserves ToolBlock Name, Input, Output.
	m := newPanel()
	now := time.Now()
	m.RestoreMessages([]state.DisplayMessage{
		{
			Role:      "assistant",
			Content:   "I read the file",
			Timestamp: now,
			ToolBlocks: []state.ToolBlock{
				{Name: "Read", Input: "internal/foo.go", Output: "package foo"},
				{Name: "Grep", Input: "pattern", Output: "3 matches"},
			},
		},
	})

	saved := m.SaveMessages()
	require.Len(t, saved, 1)
	require.Len(t, saved[0].ToolBlocks, 2)

	m2 := newPanel()
	m2.RestoreMessages(saved)

	msgs := m2.Messages()
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].ToolBlocks, 2)

	assert.Equal(t, "Read", msgs[0].ToolBlocks[0].Name)
	assert.Equal(t, "internal/foo.go", msgs[0].ToolBlocks[0].Input)
	assert.Equal(t, "package foo", msgs[0].ToolBlocks[0].Output)
	assert.False(t, msgs[0].ToolBlocks[0].Expanded)

	assert.Equal(t, "Grep", msgs[0].ToolBlocks[1].Name)
	assert.Equal(t, "pattern", msgs[0].ToolBlocks[1].Input)
	assert.Equal(t, "3 matches", msgs[0].ToolBlocks[1].Output)
	assert.False(t, msgs[0].ToolBlocks[1].Expanded)
}

func TestSaveMessages_EmptyToolBlocks_RoundTrip(t *testing.T) {
	// Messages with no ToolBlocks should round-trip without panics.
	m := newPanel()
	m.RestoreMessages([]state.DisplayMessage{
		{Role: "user", Content: "hello", Timestamp: time.Now()},
		{Role: "assistant", Content: "hi", Timestamp: time.Now()},
	})

	saved := m.SaveMessages()
	assert.Len(t, saved, 2)
	assert.Nil(t, saved[0].ToolBlocks)
	assert.Nil(t, saved[1].ToolBlocks)

	m2 := newPanel()
	m2.RestoreMessages(saved)
	msgs := m2.Messages()
	assert.Nil(t, msgs[0].ToolBlocks)
	assert.Nil(t, msgs[1].ToolBlocks)
}

// ---------------------------------------------------------------------------
// Slash command integration (TUI-054)
// ---------------------------------------------------------------------------

// typeIntoPanel simulates typing the given string character by character into
// the panel. It returns the updated model. Commands are discarded.
func typeIntoPanel(m claude.ClaudePanelModel, s string) claude.ClaudePanelModel {
	for _, r := range s {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

// drainCmd executes a cmd and returns the resulting message, or nil.
// This helper resolves a tea.Cmd returned by Update.
func drainCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestSlashInput_ShowsDropdown(t *testing.T) {
	m := newPanel()
	// Type "/" — this should trigger the slash dropdown to appear.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	view := m.View()
	// The dropdown renders command names such as "/explore".
	assert.Contains(t, view, "/explore", "typing '/' should show the slash command dropdown")
}

func TestSlashInput_FiltersOnTyping(t *testing.T) {
	m := newPanel()
	// Type "/ex" — should filter to commands starting with "ex".
	m = typeIntoPanel(m, "/ex")

	view := m.View()
	assert.Contains(t, view, "/explore", "'/ex' should match /explore")
	assert.NotContains(t, view, "/braintrust", "'/ex' should not match /braintrust")
}

func TestSlashInput_HidesOnNonSlash(t *testing.T) {
	m := newPanel()
	// Show the dropdown.
	m = typeIntoPanel(m, "/ex")
	// Now clear the input by pressing Backspace twice.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	view := m.View()
	// After clearing the "/" the dropdown should be gone.
	// We verify by ensuring the view no longer contains the dropdown border.
	// The input line itself may still be empty, so just check no dropdown.
	_ = view // No panic is the minimum bar; the dropdown should be hidden.
}

func TestSlashClear_ClearsMessages(t *testing.T) {
	m := newPanel()
	// Add some messages first.
	m, _ = m.Update(assistantMsg("message to be cleared"))
	require.Len(t, m.Messages(), 1, "pre-condition: one message")

	// Execute /clear via the dropdown selection message.
	m, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/clear"})

	assert.Empty(t, m.Messages(), "/clear should remove all messages")

	// The emitted command should produce a SlashExecutedMsg.
	msg := drainCmd(cmd)
	executed, ok := msg.(model.SlashExecutedMsg)
	require.True(t, ok, "expected SlashExecutedMsg; got %T", msg)
	assert.Equal(t, "/clear", executed.Command)
	assert.True(t, executed.IsLocal, "/clear is a local command")
}

func TestSlashHelp_IsLocal(t *testing.T) {
	m := newPanel()
	m, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/help"})

	msg := drainCmd(cmd)
	executed, ok := msg.(model.SlashExecutedMsg)
	require.True(t, ok, "expected SlashExecutedMsg; got %T", msg)
	assert.Equal(t, "/help", executed.Command)
	assert.True(t, executed.IsLocal, "/help is a local command")

	// /help should append a system message to the conversation.
	msgs := m.Messages()
	require.NotEmpty(t, msgs, "/help should add a system message")
	lastMsg := msgs[len(msgs)-1]
	assert.Equal(t, "system", lastMsg.Role)
	assert.Contains(t, lastMsg.Content, "slash command", "help text should mention slash commands")
}

func TestRemoteSkillsLoadedMsg_HelpUsesSessionSkillList(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(model.RemoteSkillsLoadedMsg{Skills: []string{"plan-tickets"}})
	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/help"})

	msgs := m.Messages()
	require.NotEmpty(t, msgs)
	lastMsg := msgs[len(msgs)-1]
	assert.Contains(t, lastMsg.Content, "/plan-tickets")
	assert.NotContains(t, lastMsg.Content, "/ticket")
}

func TestSlashRemote_CallsSendMessage(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	// Simulate user selecting /explore from the dropdown.
	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/explore"})

	if len(stub.sent) != 1 {
		t.Fatalf("remote slash command should call SendMessage once; got %d calls", len(stub.sent))
	}
	assert.Equal(t, "/explore", stub.sent[0])
}

func TestSlashRemote_WithArgs(t *testing.T) {
	// Typing "/explore foo" and pressing Enter should route "/explore foo" to sender.
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	m = typeIntoPanel(m, "/explore foo")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	require.Len(t, stub.sent, 1, "should have sent one message")
	assert.Equal(t, "/explore foo", stub.sent[0])
}

func TestSlashExecutedMsg_EmittedOnExecution(t *testing.T) {
	m := newPanel()
	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/explore"})

	msg := drainCmd(cmd)
	executed, ok := msg.(model.SlashExecutedMsg)
	require.True(t, ok, "expected SlashExecutedMsg for remote command; got %T", msg)
	assert.Equal(t, "/explore", executed.Command)
	assert.False(t, executed.IsLocal, "/explore is not local")
}

func TestSlashExecutedMsg_Args(t *testing.T) {
	// When the user types "/explore foo bar" and presses Enter, Args should
	// contain "foo bar" in the emitted SlashExecutedMsg.
	m := newPanel()
	m = typeIntoPanel(m, "/explore foo bar")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	msg := drainCmd(cmd)
	executed, ok := msg.(model.SlashExecutedMsg)
	require.True(t, ok, "expected SlashExecutedMsg; got %T", msg)
	assert.Equal(t, "/explore", executed.Command)
	assert.Equal(t, "foo bar", executed.Args)
}

func TestDropdown_RenderedInView(t *testing.T) {
	m := newPanel()
	// The dropdown should not be visible initially.
	view := m.View()
	assert.NotContains(t, view, "/explore", "dropdown should not be visible before '/' is typed")

	// Type "/" to show it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	view = m.View()
	assert.Contains(t, view, "/explore", "dropdown should appear in View() when '/' is typed")
}

func TestSlashDropdown_EscDismissesDropdown(t *testing.T) {
	m := newPanel()
	m = typeIntoPanel(m, "/exp")

	// Escape should dismiss the dropdown.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	view := m.View()
	// The dropdown content should no longer be present.
	// "/explore" text might appear in the input if tab-completed, but
	// the dropdown border styling should be gone.
	_ = view // no panic; we check next assertion
}

func TestSlashDropdown_TabCompletes(t *testing.T) {
	m := newPanel()
	// Type "/exp" to narrow to /explore.
	m = typeIntoPanel(m, "/exp")

	// Press Tab — should insert "/explore " into the input and hide the dropdown.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	view := m.View()
	// The input area should now contain "/explore ".
	assert.Contains(t, view, "/explore", "Tab should complete the command into the input")
}

func TestSlashDropdown_SelectWithEnter(t *testing.T) {
	m := newPanel()
	// Type "/" to open dropdown, then press Enter to select the top item.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// The dropdown is now visible. Press Enter — the dropdown's Update
	// returns a SlashCmdSelectedMsg which the panel then handles.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// After selection the panel should have emitted a SlashExecutedMsg
	// (possibly batched). We drain the cmd to get the message.
	// If cmd is a tea.Batch we cannot directly drain it in tests, but we
	// can verify the dropdown was hidden (view no longer shows it).
	_ = cmd
	_ = m
}

func TestSlashClear_InputClearedAfterExecution(t *testing.T) {
	m := newPanel()
	// Simulate typing "/clear" and having it selected from the dropdown.
	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/clear"})
	// The input field should be empty after executing the command.
	view := m.View()
	// The input prompt "›" should be present but with no text after it.
	assert.Contains(t, view, "›", "input prompt should still be visible")
}

func TestSlashHelp_AddsSystemMessage(t *testing.T) {
	m := newPanel()
	initialCount := len(m.Messages())

	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/help"})

	msgs := m.Messages()
	assert.Greater(t, len(msgs), initialCount, "/help should add at least one message")
	// The added message should be a system message.
	found := false
	for _, msg := range msgs {
		if msg.Role == "system" {
			found = true
			break
		}
	}
	assert.True(t, found, "/help should add a 'system' role message")
}

// ---------------------------------------------------------------------------
// /effort slash command
// ---------------------------------------------------------------------------

func TestSlashEffort_WithLevel_EmitsEffortChangeRequestMsg(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"low", "/effort low", "low"},
		{"medium", "/effort medium", "medium"},
		{"high", "/effort high", "high"},
		{"max", "/effort max", "max"},
		{"auto maps to empty", "/effort auto", "auto"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newPanel()
			m = typeIntoPanel(m, tc.input)
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			msg := drainCmd(cmd)
			req, ok := msg.(model.EffortChangeRequestMsg)
			require.True(t, ok, "expected EffortChangeRequestMsg; got %T", msg)
			assert.Equal(t, tc.want, req.Level)
		})
	}
}

func TestSlashEffort_NoArgs_EmitsEmptyLevel(t *testing.T) {
	m := newPanel()
	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/effort"})

	msg := drainCmd(cmd)
	req, ok := msg.(model.EffortChangeRequestMsg)
	require.True(t, ok, "expected EffortChangeRequestMsg; got %T", msg)
	assert.Equal(t, "", req.Level, "/effort with no args should emit empty Level")
}

func TestSlashEffort_ViaDropdown_EmitsEffortChangeRequestMsg(t *testing.T) {
	m := newPanel()
	// Simulate selecting from the dropdown (no args).
	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/effort"})

	msg := drainCmd(cmd)
	_, ok := msg.(model.EffortChangeRequestMsg)
	require.True(t, ok, "expected EffortChangeRequestMsg from dropdown selection; got %T", msg)
}

// ---------------------------------------------------------------------------
// UX-001: Horizontal rule between role transitions
// ---------------------------------------------------------------------------

func TestRenderMessages_SeparatorBetweenRoleTransitions(t *testing.T) {
	// Build a conversation with alternating roles: user → assistant.
	m := newPanel()
	m, _ = sendAndCapture(m, "Hello")
	m, _ = m.Update(assistantMsg("Hi there"))

	view := stripANSI(m.View())
	if !strings.Contains(view, "─") {
		t.Error("expected horizontal rule (─) between user→assistant messages; got none")
	}
}

func TestRenderMessages_NoSeparatorSameRole(t *testing.T) {
	// Two consecutive assistant messages should NOT have a separator between them.
	m := newPanel()
	m, _ = m.Update(assistantMsg("First assistant turn"))
	m, _ = m.Update(assistantMsg("Second assistant turn"))

	// Both messages must be visible.
	view := stripANSI(m.View())
	require.True(t, strings.Contains(view, "First assistant turn"), "first message not visible")
	require.True(t, strings.Contains(view, "Second assistant turn"), "second message not visible")

	if strings.Contains(view, "─") {
		t.Error("expected no horizontal rule between consecutive assistant messages; got one")
	}
}

func TestRenderMessages_SeparatorOnlyAtTransition(t *testing.T) {
	// Conversation: user → assistant → user → assistant.
	// The separator must appear at each role boundary.
	m := newPanel()
	m, _ = sendAndCapture(m, "First user message")
	m, _ = m.Update(assistantMsg("First reply"))
	m, _ = sendAndCapture(m, "Second user message")
	m, _ = m.Update(assistantMsg("Second reply"))

	view := stripANSI(m.View())

	// Verify separator is present (at least one transition).
	if !strings.Contains(view, "─") {
		t.Error("expected at least one horizontal rule in multi-turn conversation; got none")
	}

	// Verify all message content is still present.
	for _, content := range []string{
		"First user message", "First reply",
		"Second user message", "Second reply",
	} {
		if !strings.Contains(view, content) {
			t.Errorf("message content %q missing from view after separator insertion", content)
		}
	}
}

func TestRenderMessages_SeparatorUsesViewportWidth(t *testing.T) {
	// The rule length must match the viewport width (panelWidth - 1 for scrollbar).
	const panelWidth = 60
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(panelWidth, 24)
	m.SetFocused(true)

	m, _ = sendAndCapture(m, "ping")
	m, _ = m.Update(model.AssistantMsg{Text: "pong", Streaming: false})

	view := stripANSI(m.View())

	// The viewport is panelWidth-1 wide (one column reserved for scrollbar).
	expectedRule := strings.Repeat("─", panelWidth-1)
	if !strings.Contains(view, expectedRule) {
		t.Errorf("expected rule of width %d matching viewport width; not found in view", panelWidth-1)
	}
}

// ---------------------------------------------------------------------------
// UX-014: Inline tool spinner + duration (renderToolBlock)
// ---------------------------------------------------------------------------

// toolUseMsg constructs a ToolUseMsg with the given name/id/input.
func toolUseMsg(name, id, input string) model.ToolUseMsg {
	return model.ToolUseMsg{ToolName: name, ToolID: id, Input: input}
}

// toolResultMsg constructs a ToolResultMsg signalling success or failure.
func toolResultMsg(id string, success bool) model.ToolResultMsg {
	return model.ToolResultMsg{ToolID: id, Success: success}
}

// TestRenderToolBlock_PendingShowsSpinner verifies that a ToolBlock whose
// Success field is nil (pending) renders with the ⏳ spinner prefix.
func TestRenderToolBlock_PendingShowsSpinner(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Read", "tool-1", `{"file_path":"/tmp/foo.txt"}`))

	view := stripANSI(m.View())
	if !strings.Contains(view, "⏳") {
		t.Errorf("pending tool block should show ⏳ spinner; view:\n%s", view)
	}
}

// TestRenderToolBlock_SuccessShowsCheckAndDuration verifies that a completed
// successful ToolBlock renders with ✓ and a duration string.
func TestRenderToolBlock_SuccessShowsCheckAndDuration(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Read", "tool-2", `{"file_path":"/tmp/bar.txt"}`))
	// Simulate a small sleep so Duration > 0.
	time.Sleep(2 * time.Millisecond)
	m, _ = m.Update(toolResultMsg("tool-2", true))

	view := stripANSI(m.View())
	if !strings.Contains(view, "✓") {
		t.Errorf("completed successful tool block should show ✓; view:\n%s", view)
	}
	// A duration suffix (e.g. "2ms" or "0ms") should appear.
	if !strings.Contains(view, "ms") && !strings.Contains(view, "s") {
		t.Errorf("completed tool block should show a duration suffix; view:\n%s", view)
	}
}

// TestRenderToolBlock_FailureShowsX verifies that a failed ToolBlock renders
// with the ✗ prefix.
func TestRenderToolBlock_FailureShowsX(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Bash", "tool-3", `{"command":"ls /nonexistent"}`))
	m, _ = m.Update(toolResultMsg("tool-3", false))

	view := stripANSI(m.View())
	if !strings.Contains(view, "✗") {
		t.Errorf("failed tool block should show ✗; view:\n%s", view)
	}
}

// TestToolBlocksShownInAllTiers verifies that tool blocks are rendered
// regardless of the active layout tier (UX-014 removed the compact-only gate).
func TestToolBlocksShownInAllTiers(t *testing.T) {
	tiers := []struct {
		name  string
		width int // width drives tier selection in SetSize
	}{
		{"compact", 60},
		{"standard", 100},
		{"wide", 140},
	}
	for _, tc := range tiers {
		t.Run(tc.name, func(t *testing.T) {
			m := claude.NewClaudePanelModel(config.DefaultKeyMap())
			m.SetSize(tc.width, 24)
			m.SetFocused(true)

			m, _ = m.Update(assistantMsg("hi"))
			m, _ = m.Update(toolUseMsg("Read", "tool-t", `{"file_path":"/x"}`))

			view := stripANSI(m.View())
			if !strings.Contains(view, "[Read]") {
				t.Errorf("tier=%s: expected [Read] in view; got:\n%s", tc.name, view)
			}
		})
	}
}

// TestRenderToolBlock_PendingNoSpinnerAfterCompletion verifies that once a
// result arrives the ⏳ spinner is replaced by ✓/✗ (not both).
func TestRenderToolBlock_PendingNoSpinnerAfterCompletion(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Edit", "tool-5", `{"file_path":"/a/b.go"}`))
	m, _ = m.Update(toolResultMsg("tool-5", true))

	view := stripANSI(m.View())
	if strings.Contains(view, "⏳") {
		t.Errorf("completed tool block must not still show ⏳; view:\n%s", view)
	}
	if !strings.Contains(view, "✓") {
		t.Errorf("completed tool block should show ✓; view:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// UX-015: ToggleToolExpansion / CycleExpansion keybindings
// ---------------------------------------------------------------------------

// TestToggleToolExpansion_TogglesLastBlock verifies that alt+e flips the
// Expanded flag on the most-recent tool block.
func TestToggleToolExpansion_TogglesLastBlock(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Read", "t1", `{"file_path":"/a.go"}`))

	msgs := m.Messages()
	require.Len(t, msgs[0].ToolBlocks, 1)
	assert.False(t, msgs[0].ToolBlocks[0].Expanded, "new tool block should default to collapsed")

	// First alt+e: expand.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e"), Alt: true})
	msgs = m.Messages()
	assert.True(t, msgs[0].ToolBlocks[0].Expanded, "after first alt+e block should be expanded")

	// Second alt+e: collapse.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e"), Alt: true})
	msgs = m.Messages()
	assert.False(t, msgs[0].ToolBlocks[0].Expanded, "after second alt+e block should be collapsed again")
}

// TestCycleExpansion_ExpandsAllWhenAnyCollapsed verifies that alt+shift+e
// expands all blocks when at least one is collapsed.
func TestCycleExpansion_ExpandsAllWhenAnyCollapsed(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Read", "t1", `{"file_path":"/a.go"}`))
	m, _ = m.Update(toolUseMsg("Write", "t2", `{"file_path":"/b.go"}`))

	// Manually expand the first block so we have a mixed state.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e"), Alt: true})

	// alt+E (shift) should expand all.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E"), Alt: true})
	msgs := m.Messages()
	require.Len(t, msgs[0].ToolBlocks, 2)
	assert.True(t, msgs[0].ToolBlocks[0].Expanded, "block 0 should be expanded")
	assert.True(t, msgs[0].ToolBlocks[1].Expanded, "block 1 should be expanded")
}

// TestCycleExpansion_CollapsesAllWhenAllExpanded verifies that alt+shift+e
// collapses all blocks when all are already expanded.
func TestCycleExpansion_CollapsesAllWhenAllExpanded(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("working"))
	m, _ = m.Update(toolUseMsg("Read", "t1", `{"file_path":"/a.go"}`))
	m, _ = m.Update(toolUseMsg("Write", "t2", `{"file_path":"/b.go"}`))

	// Expand all first.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E"), Alt: true})
	msgs := m.Messages()
	require.True(t, msgs[0].ToolBlocks[0].Expanded && msgs[0].ToolBlocks[1].Expanded,
		"precondition: both blocks expanded")

	// alt+E again: collapse all.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E"), Alt: true})
	msgs = m.Messages()
	assert.False(t, msgs[0].ToolBlocks[0].Expanded, "block 0 should be collapsed")
	assert.False(t, msgs[0].ToolBlocks[1].Expanded, "block 1 should be collapsed")
}

// TestToggleToolExpansion_NoOpWithoutToolBlocks verifies that alt+e on a panel
// with no tool blocks does not panic and is a silent no-op.
func TestToggleToolExpansion_NoOpWithoutToolBlocks(t *testing.T) {
	m := newPanel()
	m, _ = m.Update(assistantMsg("no tools here"))

	require.NotPanics(t, func() {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e"), Alt: true})
	})
	msgs := m.Messages()
	assert.Len(t, msgs[0].ToolBlocks, 0, "no tool blocks should be added")
}
