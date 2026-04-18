// Package model — thinking state tracking tests.
//
// These tests verify that handleAssistantEvent correctly detects thinking
// blocks and emits ThinkingActiveMsg to the Claude panel.
package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/cli"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// collectingPanel extends mockClaudePanel to capture all received messages.
type collectingPanel struct {
	mockClaudePanel
	msgs []tea.Msg
}

func (p *collectingPanel) HandleMsg(msg tea.Msg) tea.Cmd {
	p.handleMsgCalled = true
	p.lastMsg = msg
	p.msgs = append(p.msgs, msg)
	return nil
}

// lastThinkingActiveMsg returns the last ThinkingActiveMsg received, and
// whether any was received at all.
func (p *collectingPanel) lastThinkingActiveMsg() (ThinkingActiveMsg, bool) {
	for i := len(p.msgs) - 1; i >= 0; i-- {
		if m, ok := p.msgs[i].(ThinkingActiveMsg); ok {
			return m, true
		}
	}
	return ThinkingActiveMsg{}, false
}

// newAppModelWithCollector returns an AppModel wired with a collectingPanel.
func newAppModelWithCollector() (AppModel, *collectingPanel) {
	m := newReadyAppModel(120, 40)
	p := &collectingPanel{}
	m.shared.claudePanel = p
	return m, p
}

// makeSingleBlockEvent constructs an AssistantEvent with a single content block.
func makeSingleBlockEvent(blockType, text, thinking string) cli.AssistantEvent {
	block := cli.ContentBlock{Type: blockType}
	if text != "" {
		block.Text = text
	}
	if thinking != "" {
		block.Thinking = thinking
	}
	return cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:      "msg-1",
			Role:    "assistant",
			Content: []cli.ContentBlock{block},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestHandleAssistantEvent_ThinkingBlock_EmitsThinkingActiveTrue verifies that
// an AssistantEvent containing a "thinking" block causes ThinkingActiveMsg
// {Active: true} to be delivered to the Claude panel.
func TestHandleAssistantEvent_ThinkingBlock_EmitsThinkingActiveTrue(t *testing.T) {
	m, panel := newAppModelWithCollector()

	ev := makeSingleBlockEvent("thinking", "", "Let me reason through this...")
	m.Update(ev)

	msg, ok := panel.lastThinkingActiveMsg()
	if !ok {
		t.Fatal("expected ThinkingActiveMsg to be sent to panel; none received")
	}
	if !msg.Active {
		t.Errorf("ThinkingActiveMsg.Active = false; want true when thinking block present")
	}
}

// TestHandleAssistantEvent_TextOnly_EmitsThinkingActiveFalse verifies that an
// AssistantEvent with only text blocks (no thinking blocks) emits
// ThinkingActiveMsg{Active: false} to signal the response phase.
func TestHandleAssistantEvent_TextOnly_EmitsThinkingActiveFalse(t *testing.T) {
	m, panel := newAppModelWithCollector()

	ev := makeSingleBlockEvent("text", "Here is my answer.", "")
	m.Update(ev)

	msg, ok := panel.lastThinkingActiveMsg()
	if !ok {
		t.Fatal("expected ThinkingActiveMsg to be sent to panel; none received")
	}
	if msg.Active {
		t.Errorf("ThinkingActiveMsg.Active = true; want false when only text blocks present")
	}
}

// TestHandleAssistantEvent_MixedBlocks_EmitsThinkingActiveTrue verifies that
// when BOTH thinking and text blocks are present, the thinking flag wins and
// ThinkingActiveMsg{Active: true} is emitted.
func TestHandleAssistantEvent_MixedBlocks_EmitsThinkingActiveTrue(t *testing.T) {
	m, panel := newAppModelWithCollector()

	ev := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:   "msg-mixed",
			Role: "assistant",
			Content: []cli.ContentBlock{
				{Type: "thinking", Thinking: "reasoning..."},
				{Type: "text", Text: "answer"},
			},
		},
	}
	m.Update(ev)

	msg, ok := panel.lastThinkingActiveMsg()
	if !ok {
		t.Fatal("expected ThinkingActiveMsg to be sent to panel; none received")
	}
	if !msg.Active {
		t.Errorf("ThinkingActiveMsg.Active = false; want true when thinking block present alongside text")
	}
}

// TestHandleAssistantEvent_NoTextNoThinking_NoThinkingActiveMsg verifies that
// an AssistantEvent with no text or thinking blocks (e.g. tool_use only) does
// NOT emit a ThinkingActiveMsg.
func TestHandleAssistantEvent_NoTextNoThinking_NoThinkingActiveMsg(t *testing.T) {
	m, panel := newAppModelWithCollector()

	ev := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:   "msg-tool-only",
			Role: "assistant",
			Content: []cli.ContentBlock{
				{Type: "tool_use", ID: "tool-1", Name: "SomeTool"},
			},
		},
	}
	m.Update(ev)

	_, ok := panel.lastThinkingActiveMsg()
	if ok {
		t.Error("ThinkingActiveMsg was sent but none expected for tool_use-only event")
	}
}
