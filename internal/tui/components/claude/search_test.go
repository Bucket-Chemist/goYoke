package claude_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/claude"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeMessages builds a slice of DisplayMessages from role/content pairs for
// use in search tests.
func makeMessages(pairs ...string) []state.DisplayMessage {
	if len(pairs)%2 != 0 {
		panic("makeMessages: pairs must be even (role, content, role, content, ...)")
	}
	msgs := make([]state.DisplayMessage, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		msgs = append(msgs, state.DisplayMessage{
			Role:      pairs[i],
			Content:   pairs[i+1],
			Timestamp: time.Now(),
		})
	}
	return msgs
}

// typeIntoSearch simulates typing a string into the search model by sending
// individual KeyRunes messages.
func typeIntoSearch(s claude.SearchModel, text string) (claude.SearchModel, []tea.Cmd) {
	var cmds []tea.Cmd
	for _, r := range text {
		var cmd tea.Cmd
		s, cmd = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		cmds = append(cmds, cmd)
	}
	return s, cmds
}

// ---------------------------------------------------------------------------
// NewSearchModel
// ---------------------------------------------------------------------------

func TestNewSearchModel_Inactive(t *testing.T) {
	s := claude.NewSearchModel()
	assert.False(t, s.IsActive())
}

func TestNewSearchModel_NoResults(t *testing.T) {
	s := claude.NewSearchModel()
	assert.False(t, s.HasResults())
	assert.Equal(t, -1, s.CurrentResultIndex())
}

func TestNewSearchModel_ViewEmpty(t *testing.T) {
	s := claude.NewSearchModel()
	assert.Empty(t, s.View(), "inactive search model should render nothing")
}

// ---------------------------------------------------------------------------
// Activate / Deactivate
// ---------------------------------------------------------------------------

func TestActivate_SetsActive(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()
	assert.True(t, s.IsActive())
}

func TestDeactivate_ClearsState(t *testing.T) {
	msgs := makeMessages("user", "hello world")
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "hello")
	s.ExecuteSearch(msgs)
	require.True(t, s.HasResults())

	s.Deactivate()
	assert.False(t, s.IsActive())
	assert.False(t, s.HasResults())
	assert.Equal(t, -1, s.CurrentResultIndex())
}

func TestActivate_ViewNonEmpty(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()
	view := s.View()
	assert.NotEmpty(t, view, "active search model should render the search bar")
	assert.Contains(t, view, "Search:")
}

// ---------------------------------------------------------------------------
// ExecuteSearch
// ---------------------------------------------------------------------------

func TestExecuteSearch_ExactMatch(t *testing.T) {
	msgs := makeMessages(
		"user", "hello world",
		"assistant", "goodbye moon",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "hello")
	s.ExecuteSearch(msgs)

	require.True(t, s.HasResults())
	assert.Equal(t, 0, s.CurrentResultIndex())
}

func TestExecuteSearch_CaseInsensitive(t *testing.T) {
	msgs := makeMessages("assistant", "Hello World")
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "HELLO")
	s.ExecuteSearch(msgs)

	assert.True(t, s.HasResults())
}

func TestExecuteSearch_NoMatch(t *testing.T) {
	msgs := makeMessages("user", "hello world")
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "xyz")
	s.ExecuteSearch(msgs)

	assert.False(t, s.HasResults())
	assert.Equal(t, -1, s.CurrentResultIndex())
}

func TestExecuteSearch_MultipleMatches(t *testing.T) {
	msgs := makeMessages(
		"user", "go is great",
		"assistant", "yes, go is fast",
		"user", "I love go",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)

	require.True(t, s.HasResults())
	// All three messages contain "go"; first result should be index 0.
	assert.Equal(t, 0, s.CurrentResultIndex())
}

func TestExecuteSearch_EmptyQuery_NoResults(t *testing.T) {
	msgs := makeMessages("user", "hello world")
	s := claude.NewSearchModel()
	s.Activate()
	// Do not type anything — query is empty.
	s.ExecuteSearch(msgs)

	assert.False(t, s.HasResults())
}

func TestExecuteSearch_ResetsResultsOnNewCall(t *testing.T) {
	msgs := makeMessages(
		"user", "alpha beta",
		"assistant", "gamma delta",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "alpha")
	s.ExecuteSearch(msgs)
	require.Len(t, s.View(), len(s.View())) // just ensure non-panic

	// Change query and re-search.
	s.Deactivate()
	s.Activate()
	s, _ = typeIntoSearch(s, "gamma")
	s.ExecuteSearch(msgs)

	require.True(t, s.HasResults())
	// gamma is in index 1.
	assert.Equal(t, 1, s.CurrentResultIndex())
}

// ---------------------------------------------------------------------------
// NextResult / PrevResult
// ---------------------------------------------------------------------------

func TestNextResult_WrapsAround(t *testing.T) {
	msgs := makeMessages(
		"user", "go test one",
		"assistant", "go test two",
		"user", "go test three",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)
	require.Len(t, msgs, 3)

	// Advance past the last result — should wrap to 0.
	s.NextResult() // → 1
	s.NextResult() // → 2
	s.NextResult() // → 0 (wrap)
	assert.Equal(t, 0, s.CurrentResultIndex())
}

func TestPrevResult_WrapsAround(t *testing.T) {
	msgs := makeMessages(
		"user", "go test one",
		"assistant", "go test two",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)

	// At index 0, PrevResult should wrap to last (index 1).
	s.PrevResult()
	assert.Equal(t, 1, s.CurrentResultIndex())
}

func TestNextResult_NoResults_IsNoOp(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()
	// No ExecuteSearch called.
	s.NextResult()
	assert.Equal(t, -1, s.CurrentResultIndex())
}

func TestPrevResult_NoResults_IsNoOp(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()
	s.PrevResult()
	assert.Equal(t, -1, s.CurrentResultIndex())
}

// ---------------------------------------------------------------------------
// Update key handling
// ---------------------------------------------------------------------------

func TestUpdate_EscapeDeactivates(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, s.IsActive())
}

func TestUpdate_EnterDeactivatesWithResults(t *testing.T) {
	msgs := makeMessages("user", "hello world")
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "hello")
	s.ExecuteSearch(msgs)
	require.True(t, s.HasResults())

	// Enter closes the overlay.
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, s.IsActive())
	// Results should still be accessible (caller can read CurrentResultIndex).
	assert.True(t, s.HasResults())
}

func TestUpdate_CtrlN_AdvancesResult(t *testing.T) {
	msgs := makeMessages(
		"user", "go test one",
		"assistant", "go test two",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)
	require.Equal(t, 0, s.CurrentResultIndex())

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	assert.Equal(t, 1, s.CurrentResultIndex())
}

func TestUpdate_CtrlP_MovesBack(t *testing.T) {
	msgs := makeMessages(
		"user", "go test one",
		"assistant", "go test two",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)
	s.NextResult() // move to 1

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	assert.Equal(t, 0, s.CurrentResultIndex())
}

func TestUpdate_TypingEmitsQueryChangedMsg(t *testing.T) {
	s := claude.NewSearchModel()
	s.Activate()

	var gotMsg claude.SearchQueryChangedMsg
	s2, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Execute the command and collect the message.
	if cmd != nil {
		for _, msg := range executeBatch(cmd) {
			if qm, ok := msg.(claude.SearchQueryChangedMsg); ok {
				gotMsg = qm
			}
		}
	}
	_ = s2

	assert.Equal(t, claude.SearchQueryChangedMsg("h"), gotMsg)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_ShowsResultCount(t *testing.T) {
	msgs := makeMessages(
		"user", "go is great",
		"assistant", "yes go is fast",
	)
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "go")
	s.ExecuteSearch(msgs)

	view := s.View()
	assert.Contains(t, view, "1 of 2")
}

func TestView_ShowsNoResultsWhenQueryHasNoMatches(t *testing.T) {
	msgs := makeMessages("user", "hello world")
	s := claude.NewSearchModel()
	s.Activate()
	s, _ = typeIntoSearch(s, "xyz")
	s.ExecuteSearch(msgs)

	view := s.View()
	assert.Contains(t, view, "no results")
}

func TestView_EmptyWhenInactive(t *testing.T) {
	s := claude.NewSearchModel()
	assert.Empty(t, s.View())
}

// ---------------------------------------------------------------------------
// Helpers for test-only command execution
// ---------------------------------------------------------------------------

// executeBatch runs a tea.Cmd and collects all messages it produces.
// For batched commands, it recursively executes each inner command.
func executeBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	// tea.BatchMsg is a slice of tea.Cmd.
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, executeBatch(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}
