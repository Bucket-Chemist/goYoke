package model

import (
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeMsg(role, content string) state.DisplayMessage {
	return state.DisplayMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func makeMsgWithTools(role, content string, toolCount int) state.DisplayMessage {
	msg := makeMsg(role, content)
	for i := 0; i < toolCount; i++ {
		msg.ToolBlocks = append(msg.ToolBlocks, state.ToolBlock{
			Name:  "Read",
			Input: "file.go",
		})
	}
	return msg
}

// ---------------------------------------------------------------------------
// truncateStr
// ---------------------------------------------------------------------------

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string untouched",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact boundary untouched",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "hello world this is a long string",
			maxLen:   10,
			expected: "hello worl...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen one",
			input:    "ab",
			maxLen:   1,
			expected: "a...",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := truncateStr(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("truncateStr(%q, %d) = %q; want %q", tc.input, tc.maxLen, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildHandoffSummary
// ---------------------------------------------------------------------------

func TestBuildHandoffSummary_FewerThanTwoMessages_ReturnsEmpty(t *testing.T) {
	tests := []struct {
		name string
		msgs []state.DisplayMessage
	}{
		{"no messages", nil},
		{"one message", []state.DisplayMessage{makeMsg("user", "hello")}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildHandoffSummary(tc.msgs, state.ProviderAnthropic, state.ProviderGoogle)
			if got != "" {
				t.Errorf("expected empty string for %d messages; got %q", len(tc.msgs), got)
			}
		})
	}
}

func TestBuildHandoffSummary_ExactlyTwoMessages_IncludesBoth(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "What is Go?"),
		makeMsg("assistant", "Go is a programming language."),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if got == "" {
		t.Fatal("expected non-empty summary for 2 messages")
	}
	if !strings.Contains(got, "What is Go?") {
		t.Errorf("summary should contain last user message; got:\n%s", got)
	}
	if !strings.Contains(got, "Go is a programming language.") {
		t.Errorf("summary should contain last assistant message; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_ProviderNamesInHeader(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "hello"),
		makeMsg("assistant", "hi there"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if !strings.Contains(got, string(state.ProviderAnthropic)) {
		t.Errorf("summary should contain fromProvider %q; got:\n%s", state.ProviderAnthropic, got)
	}
	if !strings.Contains(got, string(state.ProviderGoogle)) {
		t.Errorf("summary should contain toProvider %q; got:\n%s", state.ProviderGoogle, got)
	}
}

func TestBuildHandoffSummary_MoreThanTenMessages_OnlyScansLastTen(t *testing.T) {
	// Build 15 messages: the first 5 are "old content", last 10 are "recent content".
	var msgs []state.DisplayMessage
	for i := 0; i < 5; i++ {
		msgs = append(msgs, makeMsg("user", "old user message"))
		msgs = append(msgs, makeMsg("assistant", "old assistant message"))
	}
	// Add 5 more user+assistant pairs as "recent".
	for i := 0; i < 5; i++ {
		msgs = append(msgs, makeMsg("user", "recent user"))
		msgs = append(msgs, makeMsg("assistant", "recent assistant"))
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	// The summary must show "Last request: recent user" (not "old user message").
	if !strings.Contains(got, "recent user") {
		t.Errorf("summary should contain last user message from recent window; got:\n%s", got)
	}
	// The summary must NOT show the oldest content as the "last" messages.
	// We verify that the "old" content does not appear in the last-request/last-response lines.
	// The header line counts are based on the full message list.
	if strings.Contains(got, "Last request: old user message") {
		t.Errorf("summary should not show oldest message as last request; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_OnlyUserMessages_NoLastResponseLine(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "first question"),
		makeMsg("user", "second question"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if strings.Contains(got, "Last response:") {
		t.Errorf("summary should not have Last response line with no assistant messages; got:\n%s", got)
	}
	if !strings.Contains(got, "Last request:") {
		t.Errorf("summary should have Last request line; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_OnlyAssistantMessages_NoLastRequestLine(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("assistant", "first reply"),
		makeMsg("assistant", "second reply"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if strings.Contains(got, "Last request:") {
		t.Errorf("summary should not have Last request line with no user messages; got:\n%s", got)
	}
	if !strings.Contains(got, "Last response:") {
		t.Errorf("summary should have Last response line; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_LongContent_Truncated(t *testing.T) {
	longContent := strings.Repeat("a", 300)
	msgs := []state.DisplayMessage{
		makeMsg("user", longContent),
		makeMsg("assistant", longContent),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	// The truncated content plus "..." should appear but not the full 300-char string.
	expectedTruncated := strings.Repeat("a", maxContentLen) + "..."
	if !strings.Contains(got, expectedTruncated) {
		t.Errorf("summary should contain truncated content %q; got:\n%s", expectedTruncated, got)
	}
}

func TestBuildHandoffSummary_MessageCountsAccurate(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "u1"),
		makeMsg("assistant", "a1"),
		makeMsg("user", "u2"),
		makeMsg("assistant", "a2"),
		makeMsg("user", "u3"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	// 5 total, 3 user, 2 assistant.
	if !strings.Contains(got, "5 messages") {
		t.Errorf("summary should report 5 messages; got:\n%s", got)
	}
	if !strings.Contains(got, "3 user") {
		t.Errorf("summary should report 3 user; got:\n%s", got)
	}
	if !strings.Contains(got, "2 assistant") {
		t.Errorf("summary should report 2 assistant; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_ToolCallsReported(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "do something"),
		makeMsgWithTools("assistant", "done", 3),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if !strings.Contains(got, "3 tool calls") {
		t.Errorf("summary should report 3 tool calls; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_NoToolCalls_NoToolLine(t *testing.T) {
	msgs := []state.DisplayMessage{
		makeMsg("user", "hello"),
		makeMsg("assistant", "hi"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if strings.Contains(got, "tool calls") {
		t.Errorf("summary should not mention tool calls when there are none; got:\n%s", got)
	}
}

func TestBuildHandoffSummary_ReturnsLastNotFirst(t *testing.T) {
	// The summary must show the LAST user/assistant message, not the first.
	msgs := []state.DisplayMessage{
		makeMsg("user", "first user message"),
		makeMsg("assistant", "first assistant message"),
		makeMsg("user", "last user message"),
		makeMsg("assistant", "last assistant message"),
	}

	got := buildHandoffSummary(msgs, state.ProviderAnthropic, state.ProviderGoogle)

	if !strings.Contains(got, "last user message") {
		t.Errorf("summary should show last user message; got:\n%s", got)
	}
	if !strings.Contains(got, "last assistant message") {
		t.Errorf("summary should show last assistant message; got:\n%s", got)
	}
	if strings.Contains(got, "Last request: first user message") {
		t.Errorf("summary should not show first user message as last request; got:\n%s", got)
	}
}
