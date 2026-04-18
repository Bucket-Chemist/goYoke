// Package claude implements the Claude conversation panel for the
// GOgent-Fortress TUI.
package claude

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ---------------------------------------------------------------------------
// SearchState
// ---------------------------------------------------------------------------

// SearchState is the lifecycle state of the search overlay.
type SearchState int

const (
	// SearchInactive means the search overlay is hidden.
	SearchInactive SearchState = iota
	// SearchActive means the search overlay is visible and accepting input.
	SearchActive
)

// ---------------------------------------------------------------------------
// Package-level styles for the search bar
// ---------------------------------------------------------------------------

var (
	searchBarStyle = lipgloss.NewStyle().
			Foreground(config.ColorPrimary).
			Bold(true)

	searchResultStyle = config.StyleMuted.Copy()
)

// ---------------------------------------------------------------------------
// SearchModel
// ---------------------------------------------------------------------------

// SearchModel is the Bubbletea sub-model for the in-panel search overlay.
// It renders a one-line search bar and tracks matched message indices.
//
// The zero value is not usable; use NewSearchModel instead.
type SearchModel struct {
	state      SearchState
	query      textinput.Model
	results    []int // indices into a messages slice
	currentIdx int   // position in results (-1 = none selected)
	width      int
}

// NewSearchModel creates and returns a SearchModel ready for embedding.
// The search overlay is initially inactive.
func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search…"
	ti.CharLimit = 256

	return SearchModel{
		state:      SearchInactive,
		query:      ti,
		results:    nil,
		currentIdx: -1,
	}
}

// ---------------------------------------------------------------------------
// State accessors
// ---------------------------------------------------------------------------

// IsActive returns true when the search overlay is visible.
func (s *SearchModel) IsActive() bool {
	return s.state == SearchActive
}

// HasResults returns true when at least one match exists.
func (s SearchModel) HasResults() bool {
	return len(s.results) > 0
}

// CurrentResultIndex returns the message index of the currently highlighted
// result, or -1 when no result is selected.
func (s SearchModel) CurrentResultIndex() int {
	if len(s.results) == 0 || s.currentIdx < 0 || s.currentIdx >= len(s.results) {
		return -1
	}
	return s.results[s.currentIdx]
}

// ---------------------------------------------------------------------------
// Activation
// ---------------------------------------------------------------------------

// Activate shows the search input and focuses it.
func (s *SearchModel) Activate() {
	s.state = SearchActive
	s.query.Focus()
}

// Deactivate hides the search overlay and clears results.
func (s *SearchModel) Deactivate() {
	s.state = SearchInactive
	s.query.Blur()
	s.query.SetValue("")
	s.results = nil
	s.currentIdx = -1
}

// ---------------------------------------------------------------------------
// Search execution
// ---------------------------------------------------------------------------

// ExecuteSearch searches msgs for the current query string (case-insensitive)
// and populates the results slice with the indices of matching messages.
// results and currentIdx are reset before each search.
func (s *SearchModel) ExecuteSearch(msgs []state.DisplayMessage) {
	s.results = nil
	s.currentIdx = -1

	q := strings.ToLower(s.query.Value())
	if q == "" {
		return
	}

	for i, msg := range msgs {
		if strings.Contains(strings.ToLower(msg.Content), q) {
			s.results = append(s.results, i)
		}
	}

	if len(s.results) > 0 {
		s.currentIdx = 0
	}
}

// NextResult advances the highlighted result by one, wrapping around.
func (s *SearchModel) NextResult() {
	if len(s.results) == 0 {
		return
	}
	s.currentIdx = (s.currentIdx + 1) % len(s.results)
}

// PrevResult moves the highlighted result back by one, wrapping around.
func (s *SearchModel) PrevResult() {
	if len(s.results) == 0 {
		return
	}
	s.currentIdx = (s.currentIdx - 1 + len(s.results)) % len(s.results)
}

// Query returns the current search query string.
func (s SearchModel) Query() string {
	return s.query.Value()
}

// SetWidth records the terminal width for use by View.
func (s *SearchModel) SetWidth(w int) {
	s.width = w
}

// ---------------------------------------------------------------------------
// tea.Model-style methods
// ---------------------------------------------------------------------------

// Update handles incoming messages for the search overlay.
//
// Key behaviour:
//   - Printable runes / backspace → forwarded to the textinput, triggering
//     a re-search via the returned SearchQueryChangedMsg.
//   - ctrl+n → NextResult.
//   - ctrl+p → PrevResult.
//   - esc → Deactivate.
//   - enter → Deactivate while preserving results (caller can still read
//     CurrentResultIndex after deactivation).
func (s SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	if s.state != SearchActive {
		return s, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to the textinput (e.g. Blink).
		var cmd tea.Cmd
		s.query, cmd = s.query.Update(msg)
		return s, cmd
	}

	switch keyMsg.String() {
	case "esc":
		s.Deactivate()
		return s, nil

	case "enter":
		// Keep results but close the overlay so the user can navigate the
		// viewport to the highlighted message.
		s.state = SearchInactive
		s.query.Blur()
		return s, nil

	case "ctrl+n":
		s.NextResult()
		return s, nil

	case "ctrl+p":
		s.PrevResult()
		return s, nil
	}

	// All other keys: forward to the textinput.
	prevVal := s.query.Value()
	var tiCmd tea.Cmd
	s.query, tiCmd = s.query.Update(msg)
	newVal := s.query.Value()

	// If the query changed, signal that the caller should re-run the search.
	if newVal != prevVal {
		return s, tea.Batch(tiCmd, func() tea.Msg {
			return SearchQueryChangedMsg(newVal)
		})
	}

	return s, tiCmd
}

// View renders the search bar overlay as a single styled line.
// Returns an empty string when the search overlay is inactive.
func (s SearchModel) View() string {
	if s.state != SearchActive {
		return ""
	}

	queryView := s.query.View()

	var resultInfo string
	if len(s.results) == 0 {
		if s.query.Value() != "" {
			resultInfo = searchResultStyle.Render(" (no results)")
		}
	} else {
		resultInfo = searchResultStyle.Render(
			fmt.Sprintf(" (%d of %d)", s.currentIdx+1, len(s.results)),
		)
	}

	label := searchBarStyle.Render("[Search: ")
	closing := searchBarStyle.Render("]")

	return label + queryView + closing + resultInfo
}

// ---------------------------------------------------------------------------
// SearchQueryChangedMsg
// ---------------------------------------------------------------------------

// SearchQueryChangedMsg is emitted by SearchModel.Update whenever the user
// changes the search query. The parent component should call
// search.ExecuteSearch(messages) in response.
type SearchQueryChangedMsg string
