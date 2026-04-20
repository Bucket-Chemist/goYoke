package claude_test

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/claude"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// ultrathinkActive detection
// ---------------------------------------------------------------------------

func TestUltrathinkActive_SetOnSubmitWithUltrathink(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "ultrathink about this problem")
	if !m2.IsUltrathinkActive() {
		t.Error("IsUltrathinkActive() should be true after submitting a message containing 'ultrathink'")
	}
}

func TestUltrathinkActive_CaseInsensitive(t *testing.T) {
	cases := []string{
		"ULTRATHINK about this",
		"UltraThink deeply",
		"please ultrathink",
	}
	for _, text := range cases {
		m := newPanel()
		m2, _ := sendAndCapture(m, text)
		if !m2.IsUltrathinkActive() {
			t.Errorf("IsUltrathinkActive() should be true for input %q", text)
		}
	}
}

func TestUltrathinkActive_NotSetWithoutKeyword(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "think about this problem")
	if m2.IsUltrathinkActive() {
		t.Error("IsUltrathinkActive() should be false when 'ultrathink' is absent")
	}
}

func TestUltrathinkActive_ClearedOnResultMsg(t *testing.T) {
	m := newPanel()
	m2, _ := sendAndCapture(m, "ultrathink about this")
	if !m2.IsUltrathinkActive() {
		t.Fatal("precondition: IsUltrathinkActive() should be true")
	}

	m3, _ := m2.Update(model.ResultMsg{})
	if m3.IsUltrathinkActive() {
		t.Error("IsUltrathinkActive() should be cleared on ResultMsg")
	}
}

// ---------------------------------------------------------------------------
// Thinking indicator rendering
// ---------------------------------------------------------------------------

func TestThinkingIndicator_RainbowWhenUltrathinkActive(t *testing.T) {
	m := newPanel()
	// Submit with ultrathink so ultrathinkActive = true.
	m2, _ := sendAndCapture(m, "ultrathink this")
	// Simulate assistant starting to stream so messages slice is non-empty.
	m3, _ := m2.Update(streamingMsg("..."))
	// Activate thinking.
	m4, _ := m3.Update(model.ThinkingActiveMsg{Active: true})

	view := stripANSI(m4.ViewConversation())
	if !strings.Contains(view, "Thinking...") {
		t.Errorf("ViewConversation() should contain 'Thinking...' when thinkingActive && ultrathinkActive; got:\n%s", view)
	}
}

func TestThinkingIndicator_MutedWhenNotUltrathink(t *testing.T) {
	m := newPanel()
	// Submit without ultrathink.
	m2, _ := sendAndCapture(m, "think about this")
	// Simulate assistant starting to stream so messages slice is non-empty.
	m3, _ := m2.Update(streamingMsg("..."))
	// Activate thinking.
	m4, _ := m3.Update(model.ThinkingActiveMsg{Active: true})

	view := stripANSI(m4.ViewConversation())
	if !strings.Contains(view, "thinking...") {
		t.Errorf("ViewConversation() should contain 'thinking...' when thinkingActive && !ultrathinkActive; got:\n%s", view)
	}
}

func TestThinkingIndicator_StreamingDotsWhenNotThinking(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m.SetFocused(true)

	// Add a streaming assistant message so renderMessages has content to append the indicator to.
	m2, _ := m.Update(streamingMsg("partial response"))

	view := stripANSI(m2.ViewConversation())
	if !strings.Contains(view, "...") {
		t.Errorf("ViewConversation() should contain '...' streaming indicator when streaming && !thinkingActive; got:\n%s", view)
	}
}

func TestThinkingIndicator_NoDotsWhenThinkingActive(t *testing.T) {
	m := newPanel()
	// Add a streaming assistant message to populate messages.
	m2, _ := m.Update(streamingMsg("partial response"))
	// Activate thinking — replaces the streaming dots.
	m3, _ := m2.Update(model.ThinkingActiveMsg{Active: true})

	view := stripANSI(m3.ViewConversation())
	// Should show "thinking..." not bare "..."
	if strings.Contains(view, "\n...") {
		t.Errorf("ViewConversation() should not show bare '...' when thinkingActive; got:\n%s", view)
	}
	if !strings.Contains(view, "thinking...") {
		t.Errorf("ViewConversation() should show 'thinking...' when thinkingActive; got:\n%s", view)
	}
}
