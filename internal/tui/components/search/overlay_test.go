// Package search_test provides tests for the unified search overlay (TUI-059).
package search_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/search"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

// staticSource is a SearchSource that always returns a fixed result set.
type staticSource struct {
	source  string
	results []state.SearchResult
}

func (s *staticSource) Search(query string) []state.SearchResult {
	if query == "" {
		return nil
	}
	return s.results
}

// filterSource returns only results whose Label contains query.
type filterSource struct {
	source string
	items  []state.SearchResult
}

func (f *filterSource) Search(query string) []state.SearchResult {
	if query == "" {
		return nil
	}
	var out []state.SearchResult
	for _, item := range f.items {
		if containsCI(item.Label, query) {
			out = append(out, item)
		}
	}
	return out
}

func containsCI(s, sub string) bool {
	return len(s) >= len(sub) &&
		func() bool {
			sl := toLower(s)
			subl := toLower(sub)
			for i := 0; i <= len(sl)-len(subl); i++ {
				if sl[i:i+len(subl)] == subl {
					return true
				}
			}
			return false
		}()
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range b {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Activation / Deactivation
// ---------------------------------------------------------------------------

func TestNewSearchOverlayModel_InactiveByDefault(t *testing.T) {
	m := search.NewSearchOverlayModel()
	assert.False(t, m.IsActive(), "overlay must be inactive after construction")
}

func TestActivate_SetsActive(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.Activate()
	assert.True(t, m.IsActive())
}

func TestDeactivate_ClearsState(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "foo", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()

	// Type a query so results are populated.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("foo")})

	m.Deactivate()
	assert.False(t, m.IsActive())
	assert.Empty(t, m.Results())
	assert.Equal(t, 0, m.Selected())
}

func TestDeactivate_WhenAlreadyInactive_IsNoop(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.Deactivate() // should not panic
	assert.False(t, m.IsActive())
}

// ---------------------------------------------------------------------------
// Result ordering (higher score first)
// ---------------------------------------------------------------------------

func TestResultOrdering_HigherScoreFirst(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "low",  Score: 50},
			{Source: "test", Label: "high", Score: 200},
			{Source: "test", Label: "mid",  Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.SetSize(120, 40)
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	results := m.Results()
	require.Len(t, results, 3)
	assert.Equal(t, "high", results[0].Label, "highest score must be first")
	assert.Equal(t, "mid",  results[1].Label)
	assert.Equal(t, "low",  results[2].Label)
}

func TestResultOrdering_TieByLabelAscending(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "zebra",  Score: 100},
			{Source: "test", Label: "alpha",  Score: 100},
			{Source: "test", Label: "middle", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	results := m.Results()
	require.Len(t, results, 3)
	assert.Equal(t, "alpha",  results[0].Label)
	assert.Equal(t, "middle", results[1].Label)
	assert.Equal(t, "zebra",  results[2].Label)
}

// ---------------------------------------------------------------------------
// Cursor navigation
// ---------------------------------------------------------------------------

func TestNavigation_DownMovesCursor(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "a", Score: 200},
			{Source: "test", Label: "b", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	assert.Equal(t, 0, m.Selected())
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, 1, m.Selected())
}

func TestNavigation_UpMovesCursor(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "a", Score: 200},
			{Source: "test", Label: "b", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // down to idx 1
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // back up
	assert.Equal(t, 0, m.Selected())
}

func TestNavigation_DoesNotGoAboveZero(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "only", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // up on first
	assert.Equal(t, 0, m.Selected())
}

func TestNavigation_DoesNotGoBelowLast(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "only", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // down on last
	assert.Equal(t, 0, m.Selected())
}

func TestNavigation_ArrowKeysAlsoWork(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "a", Score: 200},
			{Source: "test", Label: "b", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	// Seed query so that staticSource returns results.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	// Use dedicated KeyType for arrow keys.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.Selected())
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.Selected())
}

// ---------------------------------------------------------------------------
// Source dispatch (multiple sources queried)
// ---------------------------------------------------------------------------

func TestSourceDispatch_MultipleSourcesQueried(t *testing.T) {
	src1 := &staticSource{
		source: "conv",
		results: []state.SearchResult{
			{Source: "conversation", Label: "msg1", Score: 100},
		},
	}
	src2 := &staticSource{
		source: "agents",
		results: []state.SearchResult{
			{Source: "agents", Label: "go-pro", Score: 150},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src1, src2})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	results := m.Results()
	require.Len(t, results, 2, "both sources must contribute results")

	sources := make(map[string]bool)
	for _, r := range results {
		sources[r.Source] = true
	}
	assert.True(t, sources["conversation"], "conversation source must be present")
	assert.True(t, sources["agents"], "agents source must be present")
}

func TestSourceDispatch_NilSourceIsSkipped(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "result", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{nil, src, nil})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	results := m.Results()
	require.Len(t, results, 1)
	assert.Equal(t, "result", results[0].Label)
}

// ---------------------------------------------------------------------------
// Query filtering
// ---------------------------------------------------------------------------

func TestEmptyQuery_ReturnsNoResults(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "should not appear", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	// No query typed — results should be nil.
	assert.Empty(t, m.Results())
}

func TestQuery_ResultsUpdateOnTyping(t *testing.T) {
	src := &filterSource{
		source: "test",
		items: []state.SearchResult{
			{Source: "test", Label: "golang", Score: 100},
			{Source: "test", Label: "python", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})

	results := m.Results()
	require.Len(t, results, 1)
	assert.Equal(t, "golang", results[0].Label)
}

func TestQuery_MaxResultsCapped(t *testing.T) {
	// Build a source with 15 results.
	var items []state.SearchResult
	for i := range 15 {
		items = append(items, state.SearchResult{
			Source: "test",
			Label:  string(rune('a' + i)),
			Score:  100,
		})
	}
	src := &staticSource{source: "test", results: items}

	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	results := m.Results()
	assert.LessOrEqual(t, len(results), 10, "results must be capped at maxResults=10")
}

// ---------------------------------------------------------------------------
// Esc closes overlay
// ---------------------------------------------------------------------------

func TestEsc_ClosesOverlay(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.Activate()
	assert.True(t, m.IsActive())

	cmd := m.HandleMsg(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.IsActive())

	// The returned Cmd must emit SearchDeactivatedMsg.
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(search.SearchDeactivatedMsg)
	assert.True(t, ok, "esc must emit SearchDeactivatedMsg")
}

// ---------------------------------------------------------------------------
// Enter selects a result
// ---------------------------------------------------------------------------

func TestEnter_EmitsSearchResultSelectedMsg(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "test", Label: "selected", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	cmd := m.HandleMsg(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	sel, ok := msg.(search.SearchResultSelectedMsg)
	require.True(t, ok, "enter must emit SearchResultSelectedMsg")
	assert.Equal(t, "selected", sel.Result.Label)
	assert.False(t, m.IsActive(), "overlay must close after selection")
}

func TestEnter_NoResults_EmitsDeactivatedMsg(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.Activate()

	cmd := m.HandleMsg(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(search.SearchDeactivatedMsg)
	assert.True(t, ok)
	assert.False(t, m.IsActive())
}

// ---------------------------------------------------------------------------
// SetSources / SetSize
// ---------------------------------------------------------------------------

func TestSetSources_ReplacesExistingSources(t *testing.T) {
	src1 := &staticSource{
		source: "first",
		results: []state.SearchResult{
			{Source: "first", Label: "first-result", Score: 100},
		},
	}
	src2 := &staticSource{
		source: "second",
		results: []state.SearchResult{
			{Source: "second", Label: "second-result", Score: 100},
		},
	}

	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src1})
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Len(t, m.Results(), 1)
	assert.Equal(t, "first-result", m.Results()[0].Label)

	m.SetSources([]state.SearchSource{src2})
	assert.Len(t, m.Results(), 1)
	assert.Equal(t, "second-result", m.Results()[0].Label)
}

func TestSetSize_UpdatesDimensions(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.SetSize(120, 40)
	// No assertion on internal fields (unexported), but View() should not panic.
	m.Activate()
	_ = m.View()
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_EmptyWhenInactive(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.SetSize(120, 40)
	assert.Equal(t, "", m.View())
}

func TestView_NonEmptyWhenActive(t *testing.T) {
	m := search.NewSearchOverlayModel()
	m.SetSize(120, 40)
	m.Activate()
	v := m.View()
	assert.NotEmpty(t, v)
}

func TestView_ContainsResultLabels(t *testing.T) {
	src := &staticSource{
		source: "test",
		results: []state.SearchResult{
			{Source: "conversation", Label: "hello world", Score: 100},
		},
	}
	m := search.NewSearchOverlayModel()
	m.SetSources([]state.SearchSource{src})
	m.SetSize(120, 40)
	m.Activate()
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	v := m.View()
	assert.Contains(t, v, "hello world")
}

// ---------------------------------------------------------------------------
// HandleMsg is a no-op when inactive
// ---------------------------------------------------------------------------

func TestHandleMsg_NoopWhenInactive(t *testing.T) {
	m := search.NewSearchOverlayModel()
	cmd := m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Nil(t, cmd)
	assert.False(t, m.IsActive())
}
