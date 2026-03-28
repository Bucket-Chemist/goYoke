// Package claude_test provides tests for ClaudePanelModel.Search (TUI-059).
package claude_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/claude"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// panelWithMessages builds a ClaudePanelModel pre-loaded with messages.
func panelWithMessages(msgs []state.DisplayMessage) *claude.ClaudePanelModel {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	m.RestoreMessages(msgs)
	return &m
}

// ---------------------------------------------------------------------------
// ClaudePanelModel.Search tests
// ---------------------------------------------------------------------------

func TestClaudeSearch_EmptyQueryReturnsNil(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user", Content: "hello world"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("")
	assert.Nil(t, results)
}

func TestClaudeSearch_NoMatchReturnsEmpty(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user", Content: "hello world"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("zzz")
	assert.Empty(t, results)
}

func TestClaudeSearch_MatchReturnsResult(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user",      Content: "tell me about golang"},
		{Role: "assistant", Content: "Golang is a great language"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("golang")
	assert.NotEmpty(t, results)
	for _, r := range results {
		assert.Equal(t, "conversation", r.Source)
	}
}

func TestClaudeSearch_CaseInsensitive(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user", Content: "Hello World"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("hello")
	require.NotEmpty(t, results)
	assert.Equal(t, "conversation", results[0].Source)
}

func TestClaudeSearch_PrefixMatchScoresHigher(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user", Content: "middle match here foo"},
		{Role: "user", Content: "foo prefix match"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("foo")
	require.Len(t, results, 2)
	// "foo prefix match" starts with "foo" — higher score.
	prefixResult := findByContent(results, "foo prefix match")
	middleResult := findByContent(results, "middle match here foo")
	require.NotNil(t, prefixResult, "prefix match must appear in results")
	require.NotNil(t, middleResult, "interior match must appear in results")
	assert.Greater(t, prefixResult.Score, middleResult.Score)
}

func TestClaudeSearch_LabelContainsRoleAndContent(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "assistant", Content: "the answer is 42"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("answer")
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Label, "assistant")
}

func TestClaudeSearch_DetailContainsMessageIndex(t *testing.T) {
	msgs := []state.DisplayMessage{
		{Role: "user", Content: "first message foo"},
		{Role: "user", Content: "second message bar"},
		{Role: "user", Content: "third message foo"},
	}
	m := panelWithMessages(msgs)
	results := m.Search("foo")
	require.Len(t, results, 2)
	details := make(map[string]bool)
	for _, r := range results {
		details[r.Detail] = true
	}
	assert.True(t, details["Message 1"] || details["Message 3"],
		"details must indicate message indices")
}

func TestClaudeSearch_MultipleMatches(t *testing.T) {
	var msgs []state.DisplayMessage
	for i := range 5 {
		msgs = append(msgs, state.DisplayMessage{
			Role:    "user",
			Content: fmt.Sprintf("message %d contains needle", i),
		})
	}
	m := panelWithMessages(msgs)
	results := m.Search("needle")
	assert.Len(t, results, 5)
}

func TestClaudeSearch_EmptyMessagesReturnsNil(t *testing.T) {
	m := claude.NewClaudePanelModel(config.DefaultKeyMap())
	m.SetSize(80, 24)
	results := m.Search("anything")
	assert.Nil(t, results)
}

// findByContent returns the first SearchResult whose Label contains snippet.
func findByContent(results []state.SearchResult, snippet string) *state.SearchResult {
	for i := range results {
		if containsStr(results[i].Label, snippet) {
			return &results[i]
		}
	}
	return nil
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}
