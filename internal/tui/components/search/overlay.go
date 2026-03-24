// Package search implements the unified cross-panel fuzzy search overlay for
// the GOgent-Fortress TUI (TUI-059).
//
// The overlay is activated globally by ctrl+f and queries all registered
// state.SearchSource implementations simultaneously, merging results by score.
package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SearchResultSelectedMsg is emitted when the user confirms a search result.
type SearchResultSelectedMsg struct {
	// Result is the confirmed search result.
	Result state.SearchResult
}

// SearchDeactivatedMsg is emitted when the overlay closes without a selection.
type SearchDeactivatedMsg struct{}

// ---------------------------------------------------------------------------
// Package-level styles
// ---------------------------------------------------------------------------

var (
	overlayBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	resultSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	resultNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	resultDetailStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	resultSourceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Italic(true)

	noResultsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// ---------------------------------------------------------------------------
// SearchOverlayModel
// ---------------------------------------------------------------------------

// maxResults is the maximum number of results displayed in the overlay.
const maxResults = 10

// SearchOverlayModel is the unified cross-panel search overlay.
//
// It satisfies model.searchOverlayWidget via pointer receivers that mutate
// the model in place (following the same HandleMsg pattern as other widgets).
//
// The zero value is not usable; use NewSearchOverlayModel instead.
type SearchOverlayModel struct {
	query    textinput.Model
	results  []state.SearchResult
	selected int
	active   bool
	width    int
	height   int
	sources  []state.SearchSource
}

// NewSearchOverlayModel returns a SearchOverlayModel ready for use.
// The overlay is initially inactive.
func NewSearchOverlayModel() *SearchOverlayModel {
	ti := textinput.New()
	ti.Placeholder = "Search across panels…"
	ti.CharLimit = 256

	return &SearchOverlayModel{
		query:    ti,
		selected: 0,
	}
}

// ---------------------------------------------------------------------------
// searchOverlayWidget interface implementation
// ---------------------------------------------------------------------------

// HandleMsg processes a tea.Msg, mutates the overlay in place, and returns any
// Cmd to run.  When active the overlay captures all key events:
//
//   - esc      → Deactivate; emits SearchDeactivatedMsg
//   - up / k   → move cursor up
//   - down / j → move cursor down
//   - enter    → emit SearchResultSelectedMsg (or SearchDeactivatedMsg if no results)
//   - any other key → forwarded to the textinput; triggers a re-search
func (m *SearchOverlayModel) HandleMsg(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to the textinput (e.g. cursor blink).
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		return cmd
	}

	switch keyMsg.String() {
	case "esc":
		m.Deactivate()
		return func() tea.Msg { return SearchDeactivatedMsg{} }

	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
		return nil

	case "down", "j":
		if m.selected < len(m.results)-1 {
			m.selected++
		}
		return nil

	case "enter":
		if len(m.results) > 0 && m.selected < len(m.results) {
			result := m.results[m.selected]
			m.Deactivate()
			return func() tea.Msg {
				return SearchResultSelectedMsg{Result: result}
			}
		}
		m.Deactivate()
		return func() tea.Msg { return SearchDeactivatedMsg{} }
	}

	// All other keys: forward to textinput and re-execute search.
	prevQuery := m.query.Value()
	var tiCmd tea.Cmd
	m.query, tiCmd = m.query.Update(msg)
	if m.query.Value() != prevQuery {
		m.executeSearch()
	}
	return tiCmd
}

// View renders the overlay to a string. Returns "" when not active.
//
// Layout (centred floating panel):
//
//	╭─ Search ──────────────────────╮
//	│ Search: <textinput>           │
//	│ ─────────────────────────     │
//	│ ▶ [conversation] You: foo     │
//	│   ↳ Message 3                 │
//	│   [agents] go-pro             │
//	│   ↳ implement feature         │
//	│ ─────────────────────────     │
//	│ ↑↓ navigate  enter select     │
//	╰───────────────────────────────╯
func (m *SearchOverlayModel) View() string {
	if !m.active {
		return ""
	}

	overlayWidth := m.width - 8
	if overlayWidth < 40 {
		overlayWidth = 40
	}
	if overlayWidth > 80 {
		overlayWidth = 80
	}
	innerWidth := overlayWidth - 4 // border (1 each side) + padding (1 each side)

	var sb strings.Builder

	// Header.
	sb.WriteString(headerStyle.Render("Search"))
	sb.WriteByte('\n')

	// Query line.
	sb.WriteString(inputLabelStyle.Render("Search: "))
	sb.WriteString(m.query.View())
	sb.WriteByte('\n')

	// Divider.
	if innerWidth > 0 {
		sb.WriteString(strings.Repeat("─", innerWidth))
	}
	sb.WriteByte('\n')

	// Results.
	if len(m.results) == 0 {
		if m.query.Value() == "" {
			sb.WriteString(noResultsStyle.Render("Type to search…"))
		} else {
			sb.WriteString(noResultsStyle.Render("No results"))
		}
		sb.WriteByte('\n')
	} else {
		for i, res := range m.results {
			// Source tag + cursor.
			sourceTag := resultSourceStyle.Render(fmt.Sprintf("[%s]", res.Source))
			cursor := "  "
			if i == m.selected {
				cursor = "▶ "
			}

			labelLine := cursor + sourceTag + " " + res.Label
			if innerWidth > 0 && len(labelLine) > innerWidth {
				labelLine = labelLine[:innerWidth-1] + "…"
			}

			if i == m.selected {
				sb.WriteString(resultSelectedStyle.Render(labelLine))
			} else {
				sb.WriteString(resultNormalStyle.Render(labelLine))
			}
			sb.WriteByte('\n')

			// Detail line (indented, dimmed).
			if res.Detail != "" {
				detail := "  ↳ " + res.Detail
				if innerWidth > 0 && len(detail) > innerWidth {
					detail = detail[:innerWidth-1] + "…"
				}
				sb.WriteString(resultDetailStyle.Render(detail))
				sb.WriteByte('\n')
			}
		}
	}

	// Footer hint.
	if innerWidth > 0 {
		sb.WriteString(strings.Repeat("─", innerWidth))
	}
	sb.WriteByte('\n')
	sb.WriteString(hintStyle.Render("↑↓ navigate  enter select  esc close"))

	content := overlayBorderStyle.Width(overlayWidth).Render(sb.String())

	// Centre the panel on screen.
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// SetSize updates the terminal dimensions used for overlay layout.
func (m *SearchOverlayModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// IsActive returns true when the overlay is currently displayed.
func (m *SearchOverlayModel) IsActive() bool {
	return m.active
}

// Activate shows the overlay and focuses the query input.
func (m *SearchOverlayModel) Activate() {
	m.active = true
	m.query.Focus()
	m.selected = 0
	// Re-execute any pre-existing query so results appear immediately.
	m.executeSearch()
}

// Deactivate hides the overlay and clears the query.
func (m *SearchOverlayModel) Deactivate() {
	m.active = false
	m.query.Blur()
	m.query.SetValue("")
	m.results = nil
	m.selected = 0
}

// SetSources replaces the registered search sources. If the overlay is active,
// a re-search is executed immediately so results stay current.
func (m *SearchOverlayModel) SetSources(sources []state.SearchSource) {
	m.sources = sources
	if m.active {
		m.executeSearch()
	}
}

// ---------------------------------------------------------------------------
// Public accessors (for testing)
// ---------------------------------------------------------------------------

// Results returns a copy of the current result slice. Intended for tests.
func (m *SearchOverlayModel) Results() []state.SearchResult {
	out := make([]state.SearchResult, len(m.results))
	copy(out, m.results)
	return out
}

// Selected returns the index of the currently highlighted result.
func (m *SearchOverlayModel) Selected() int {
	return m.selected
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// executeSearch queries all registered sources with the current query, merges
// the results by score (descending), and caps the list at maxResults.
func (m *SearchOverlayModel) executeSearch() {
	q := strings.TrimSpace(m.query.Value())
	if q == "" {
		m.results = nil
		m.selected = 0
		return
	}

	var merged []state.SearchResult
	for _, src := range m.sources {
		if src == nil {
			continue
		}
		merged = append(merged, src.Search(q)...)
	}

	// Sort by score descending, then label ascending for stable ordering.
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Score != merged[j].Score {
			return merged[i].Score > merged[j].Score
		}
		return merged[i].Label < merged[j].Label
	})

	if len(merged) > maxResults {
		merged = merged[:maxResults]
	}

	m.results = merged

	// Clamp cursor to valid range.
	if m.selected >= len(m.results) {
		m.selected = 0
	}
}
