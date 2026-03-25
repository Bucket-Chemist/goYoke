package claude_test

// panel_rendering_test.go contains focused tests for the ClaudePanel
// Glamour markdown rendering pipeline. It verifies which messages go
// through Glamour (completed assistant) versus which are rendered as
// plain text (streaming assistant, user messages).
//
// Helpers (newPanel, stripANSI, assistantMsg, streamingMsg) are defined
// in panel_test.go in this same package.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// ---------------------------------------------------------------------------
// TestRendering_CompletedAssistant_GlamourApplied
//
// A completed (non-streaming) assistant message must pass through Glamour.
// Glamour converts "# Heading" into styled terminal text — the raw "# "
// prefix must not appear in the output, and ANSI codes must be present.
// ---------------------------------------------------------------------------

func TestRendering_CompletedAssistant_GlamourApplied(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("# Hello World"))

	view := m2.View()

	// Raw markdown prefix "# " should be absent — Glamour converts it.
	plain := stripANSI(view)
	assert.NotContains(t, plain, "# Hello World",
		"completed assistant message: raw '# ' heading prefix should be absent after Glamour rendering")

	// The heading text itself should still be present (just styled).
	assert.Contains(t, plain, "Hello World",
		"completed assistant message: heading text should appear in rendered output")

	// ANSI escape codes should be present (Glamour adds styling).
	assert.Contains(t, view, "\x1b[",
		"completed assistant message: ANSI codes should be present after Glamour rendering")
}

// ---------------------------------------------------------------------------
// TestRendering_StreamingAssistant_PlainText
//
// While a message is being streamed (streaming=true, isLast=true), the panel
// must NOT invoke Glamour — the raw markdown must be visible as-is.
// ---------------------------------------------------------------------------

func TestRendering_StreamingAssistant_PlainText(t *testing.T) {
	m := newPanel()
	// Send a streaming snapshot with markdown content.
	m2, _ := m.Update(model.AssistantMsg{Text: "# Streaming Heading", Streaming: true})

	require.True(t, m2.IsStreaming(), "pre-condition: panel should be streaming")

	// The raw markdown prefix "# " should still be present because Glamour
	// is suppressed while streaming.
	plain := stripANSI(m2.View())
	assert.Contains(t, plain, "# Streaming Heading",
		"streaming assistant message: raw markdown should be visible (no Glamour)")
}

// ---------------------------------------------------------------------------
// TestRendering_CodeBlock_SyntaxHighlighted
//
// A completed assistant message containing a fenced Go code block should
// contain the code's text content after Glamour rendering.
// ---------------------------------------------------------------------------

func TestRendering_CodeBlock_SyntaxHighlighted(t *testing.T) {
	src := "```go\nfmt.Println(\"hello\")\n```"
	m := newPanel()
	m2, _ := m.Update(assistantMsg(src))

	// The raw fence markers should be absent — Glamour converts them.
	plain := stripANSI(m2.View())
	assert.NotContains(t, plain, "```go",
		"completed assistant message: raw fenced code markers should be absent after Glamour rendering")

	// The actual code content must survive rendering.
	assert.Contains(t, plain, "fmt.Println",
		"completed assistant message: code content should be present after Glamour rendering")
}

// ---------------------------------------------------------------------------
// TestRendering_UserMessage_NoGlamour
//
// User messages must NOT pass through Glamour — the raw text (including any
// markdown characters) should appear verbatim in the view.
// ---------------------------------------------------------------------------

func TestRendering_UserMessage_NoGlamour(t *testing.T) {
	m := newPanel()
	// Type and submit a message that contains markdown syntax.
	m2, _ := sendAndCapture(m, "# My heading")

	view := m2.View()
	plain := stripANSI(view)

	// User messages bypass Glamour — "# My heading" must appear as-is.
	assert.Contains(t, plain, "# My heading",
		"user message: raw markdown text should appear verbatim (no Glamour)")
}

// ---------------------------------------------------------------------------
// TestRendering_EmptyContent_NoPanic
//
// An assistant message with empty content must render without panicking.
// ---------------------------------------------------------------------------

func TestRendering_EmptyContent_NoPanic(t *testing.T) {
	m := newPanel()

	assert.NotPanics(t, func() {
		m2, _ := m.Update(assistantMsg(""))
		// Ensure View() also doesn't panic.
		_ = m2.View()
	}, "empty assistant message content must not cause a panic")
}

// ---------------------------------------------------------------------------
// TestRendering_LargeContent_NoTruncation
//
// A large assistant message should be fully present in the viewport content
// (scrollable, not truncated). We check that the last line is reachable by
// scrolling — verified by asserting the text exists in the rendered output
// after GotoBottom is implied by autoScroll.
// ---------------------------------------------------------------------------

func TestRendering_LargeContent_NoTruncation(t *testing.T) {
	// Build a message with 50 distinct lines so we can verify none are lost.
	var lines []string
	for i := range 50 {
		lines = append(lines, strings.Repeat("word ", 10)+string(rune('A'+i%26)))
	}
	largeContent := strings.Join(lines, "\n")

	m := newPanel()
	m.SetSize(120, 40) // wider/taller panel to reduce wrapping complexity
	m2, _ := m.Update(assistantMsg(largeContent))

	// The viewport content is set via syncViewport; we can retrieve it by
	// calling View() which renders the full content string into the viewport.
	// Because autoScroll is enabled, GotoBottom is called, so the last line
	// of content is visible. We verify the last unique marker survives.
	view := m2.View()
	plain := stripANSI(view)

	// The first and last unique markers should both be present somewhere in
	// the full content (viewport may paginate but the data must not be lost).
	// We assert on the rendered message text rather than the viewport clip.
	msgs := m2.Messages()
	require.Len(t, msgs, 1, "large content: expected exactly one message")
	assert.Equal(t, largeContent, msgs[0].Content,
		"large content: full message content must be stored without truncation")

	// The view should contain at least some of the content words.
	assert.Contains(t, plain, "word",
		"large content: viewport should show content words")
}

// ---------------------------------------------------------------------------
// TestRendering_MultipleMessages_AllRendered
//
// Multiple completed assistant messages in sequence must all be rendered
// through Glamour — not just the last one.
// ---------------------------------------------------------------------------

func TestRendering_MultipleMessages_AllRendered(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(assistantMsg("# First"))
	m3, _ := m2.Update(assistantMsg("# Second"))
	m4, _ := m3.Update(assistantMsg("# Third"))

	view := m4.View()
	plain := stripANSI(view)

	// All three heading texts must appear.
	assert.Contains(t, plain, "First",
		"first assistant message should be visible")
	assert.Contains(t, plain, "Second",
		"second assistant message should be visible")
	assert.Contains(t, plain, "Third",
		"third assistant message should be visible")

	// None of the raw "# " markdown prefixes should survive — all three
	// messages pass through Glamour because none is the current streaming msg.
	assert.NotContains(t, plain, "# First",
		"first message: raw heading prefix should be absent (Glamour applied)")
	assert.NotContains(t, plain, "# Second",
		"second message: raw heading prefix should be absent (Glamour applied)")
	assert.NotContains(t, plain, "# Third",
		"third message: raw heading prefix should be absent (Glamour applied)")
}

// ---------------------------------------------------------------------------
// TestRendering_StreamingSnapshot_Replaces
//
// Multiple streaming snapshots must REPLACE (not concatenate) in the view.
// The viewport should show only the latest snapshot text.
// ---------------------------------------------------------------------------

func TestRendering_StreamingSnapshot_Replaces(t *testing.T) {
	m := newPanel()

	// First snapshot.
	m2, _ := m.Update(model.AssistantMsg{Text: "Snapshot one", Streaming: true})
	// Second snapshot (full accumulated text, not a delta).
	m3, _ := m2.Update(model.AssistantMsg{Text: "Snapshot one two", Streaming: true})
	// Third snapshot.
	m4, _ := m3.Update(model.AssistantMsg{Text: "Snapshot one two three", Streaming: true})

	view := m4.View()
	plain := stripANSI(view)

	// Latest snapshot text must be visible.
	assert.Contains(t, plain, "Snapshot one two three",
		"latest streaming snapshot must be visible")

	// Only one assistant message should exist in the model.
	msgs := m4.Messages()
	assert.Len(t, msgs, 1,
		"multiple streaming snapshots must replace, resulting in a single message")

	// Content must exactly match the last snapshot.
	if len(msgs) == 1 {
		assert.Equal(t, "Snapshot one two three", msgs[0].Content,
			"message content must match the final streaming snapshot")
	}
}
