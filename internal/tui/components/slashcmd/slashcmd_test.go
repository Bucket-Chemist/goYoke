package slashcmd

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// DefaultCommands
// ---------------------------------------------------------------------------

func TestDefaultCommands_HasAtLeast16(t *testing.T) {
	cmds := DefaultCommands()
	if len(cmds) < 16 {
		t.Errorf("expected at least 16 default commands, got %d", len(cmds))
	}
}

func TestDefaultCommands_NamesNotEmpty(t *testing.T) {
	for _, cmd := range DefaultCommands() {
		if cmd.Name == "" {
			t.Errorf("command has empty Name: %+v", cmd)
		}
		if cmd.Description == "" {
			t.Errorf("command %q has empty Description", cmd.Name)
		}
	}
}

func TestDefaultCommands_ContainsExpectedCommands(t *testing.T) {
	want := []string{
		"explore", "braintrust", "review", "implement",
		"ticket", "plan-tickets", "teams", "benchmark",
	}
	cmds := DefaultCommands()
	names := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		names[c.Name] = true
	}
	for _, w := range want {
		if !names[w] {
			t.Errorf("expected command %q not found in DefaultCommands()", w)
		}
	}
}

// ---------------------------------------------------------------------------
// NewSlashCmdModel
// ---------------------------------------------------------------------------

func TestNewSlashCmdModel_StartsHidden(t *testing.T) {
	m := NewSlashCmdModel()
	if m.IsVisible() {
		t.Error("expected IsVisible()==false on new model")
	}
}

func TestNewSlashCmdModel_HasDefaultMaxVisible(t *testing.T) {
	m := NewSlashCmdModel()
	if m.maxVisible != defaultMaxVisible {
		t.Errorf("expected maxVisible=%d, got %d", defaultMaxVisible, m.maxVisible)
	}
}

func TestNewSlashCmdModel_FilteredEqualsAll(t *testing.T) {
	m := NewSlashCmdModel()
	if len(m.filtered) != len(m.commands) {
		t.Errorf("expected filtered len %d == commands len %d", len(m.filtered), len(m.commands))
	}
}

// ---------------------------------------------------------------------------
// Show / Hide
// ---------------------------------------------------------------------------

func TestShow_MakesVisible(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	if !m.IsVisible() {
		t.Error("expected IsVisible()==true after Show(\"\")")
	}
}

func TestShow_ResetsSelection(t *testing.T) {
	m := NewSlashCmdModel()
	m.selected = 5
	m.Show("")
	if m.selected != 0 {
		t.Errorf("expected selection reset to 0, got %d", m.selected)
	}
}

func TestHide_MakesInvisible(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Hide()
	if m.IsVisible() {
		t.Error("expected IsVisible()==false after Hide()")
	}
}

func TestShow_NoMatch_DoesNotMakeVisible(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("zzz_no_match")
	if m.IsVisible() {
		t.Error("expected IsVisible()==false when Show produces no matches")
	}
}

// ---------------------------------------------------------------------------
// Filter
// ---------------------------------------------------------------------------

func TestFilter_EmptyQuery_ShowsAll(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Filter("")
	if len(m.filtered) != len(m.commands) {
		t.Errorf("expected all %d commands, got %d", len(m.commands), len(m.filtered))
	}
}

func TestFilter_PrefixMatch(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Filter("ex")
	for _, cmd := range m.filtered {
		if !strings.HasPrefix(cmd.Name, "ex") {
			t.Errorf("filter 'ex' matched unexpected command %q", cmd.Name)
		}
	}
	// "explore" and "explore-add" both start with "ex".
	found := make(map[string]bool)
	for _, cmd := range m.filtered {
		found[cmd.Name] = true
	}
	if !found["explore"] {
		t.Error("expected 'explore' in filtered results for query 'ex'")
	}
	if !found["explore-add"] {
		t.Error("expected 'explore-add' in filtered results for query 'ex'")
	}
}

func TestFilter_CaseInsensitive(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Filter("EX")
	found := false
	for _, cmd := range m.filtered {
		if cmd.Name == "explore" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'explore' to match case-insensitive filter 'EX'")
	}
}

func TestFilter_NoMatch_Hides(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Filter("zzz")
	if m.IsVisible() {
		t.Error("expected dropdown to hide when filter matches nothing")
	}
}

func TestFilter_SingleMatch(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.Filter("braintr")
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match for 'braintr', got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "braintrust" {
		t.Errorf("expected 'braintrust', got %q", m.filtered[0].Name)
	}
}

func TestFilter_SlashPrefix_Stripped(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	// Querying with "/" prefix should work the same as without.
	m.Filter("/explore")
	found := false
	for _, cmd := range m.filtered {
		if cmd.Name == "explore" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'explore' to match filter '/explore' (slash stripped)")
	}
}

// ---------------------------------------------------------------------------
// Selected
// ---------------------------------------------------------------------------

func TestSelected_ReturnsHighlightedItem(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 0
	got := m.Selected()
	if got.Name == "" {
		t.Error("Selected() returned zero value when filtered is non-empty")
	}
}

func TestSelected_EmptyFiltered_ReturnsZero(t *testing.T) {
	m := NewSlashCmdModel()
	// Do not Show — filtered still has commands, but let's manually empty it.
	m.filtered = nil
	got := m.Selected()
	if got.Name != "" {
		t.Errorf("Selected() returned non-zero value for empty filtered list: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

func TestNavigation_Down(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	initial := m.selected

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.selected != initial+1 {
		t.Errorf("expected selection %d after down, got %d", initial+1, updated.selected)
	}
}

func TestNavigation_Up(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if updated.selected != 1 {
		t.Errorf("expected selection 1 after up, got %d", updated.selected)
	}
}

func TestNavigation_DownArrow(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	initial := m.selected

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.selected != initial+1 {
		t.Errorf("expected selection %d after Down arrow, got %d", initial+1, updated.selected)
	}
}

func TestNavigation_UpArrow(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updated.selected != 2 {
		t.Errorf("expected selection 2 after Up arrow, got %d", updated.selected)
	}
}

func TestNavigation_Clamp_AtTop(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 0

	// Pressing up at position 0 should clamp — not underflow.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updated.selected != 0 {
		t.Errorf("expected selection to clamp at 0, got %d", updated.selected)
	}
}

func TestNavigation_Clamp_AtBottom(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = len(m.filtered) - 1

	// Pressing down at the last item should clamp.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.selected != len(m.filtered)-1 {
		t.Errorf("expected selection to clamp at %d, got %d", len(m.filtered)-1, updated.selected)
	}
}

// ---------------------------------------------------------------------------
// Enter — emits SlashCmdSelectedMsg
// ---------------------------------------------------------------------------

func TestEnter_EmitsSelectedMsg(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 0
	firstName := m.filtered[0].Name

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd after Enter")
	}

	msg := cmd()
	selectedMsg, ok := msg.(SlashCmdSelectedMsg)
	if !ok {
		t.Fatalf("expected SlashCmdSelectedMsg, got %T", msg)
	}
	want := "/" + firstName
	if selectedMsg.Command != want {
		t.Errorf("expected Command=%q, got %q", want, selectedMsg.Command)
	}
}

func TestEnter_HidesDropdown(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.IsVisible() {
		t.Error("expected dropdown to hide after Enter")
	}
}

func TestEnter_EmptyFiltered_NoOp(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.filtered = nil // simulate empty

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd when filtered list is empty on Enter")
	}
}

// ---------------------------------------------------------------------------
// Escape — hides dropdown
// ---------------------------------------------------------------------------

func TestEscape_Hides(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	// tea.KeyEsc renders as "esc" via msg.String(); both "esc" and "escape"
	// are handled in the switch.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.IsVisible() {
		t.Error("expected dropdown to hide after Escape (KeyEsc)")
	}
}

func TestEscape_NoCmd(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("expected nil cmd after Escape")
	}
}

// ---------------------------------------------------------------------------
// Not visible — ignores keys
// ---------------------------------------------------------------------------

func TestUpdate_NotVisible_IgnoresKeys(t *testing.T) {
	m := NewSlashCmdModel()
	// Do not call Show — dropdown is hidden.

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd when dropdown is hidden")
	}
	if updated.IsVisible() {
		t.Error("expected dropdown to remain hidden")
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_EmptyWhenHidden(t *testing.T) {
	m := NewSlashCmdModel()
	if m.View() != "" {
		t.Errorf("expected empty View when hidden, got %q", m.View())
	}
}

func TestView_ContainsCommandNames(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	view := m.View()
	if !strings.Contains(view, "/explore") {
		t.Errorf("expected View to contain '/explore', got:\n%s", view)
	}
}

func TestView_ContainsDescription(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	view := m.View()
	// The first command is "explore" — its description should appear.
	if !strings.Contains(view, "Structured codebase exploration") {
		t.Errorf("expected description in View, got:\n%s", view)
	}
}

func TestView_HighlightsSelected(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	m.selected = 0
	firstName := m.filtered[0].Name
	secondName := m.filtered[1].Name

	// Both items must appear in View regardless of which is selected.
	view := m.View()
	if !strings.Contains(view, "/"+firstName) {
		t.Errorf("View() should contain /%s", firstName)
	}
	if !strings.Contains(view, "/"+secondName) {
		t.Errorf("View() should contain /%s when within maxVisible window", secondName)
	}

	// The selected field must change when we navigate.
	// (Visual differentiation via ANSI is stripped in non-TTY test contexts;
	// we verify the state machine rather than the rendered ANSI codes.)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.selected != 1 {
		t.Errorf("expected selected==1 after Down, got %d", updated.selected)
	}
	// View after navigation still contains both names.
	updatedView := updated.View()
	if !strings.Contains(updatedView, "/"+firstName) {
		t.Errorf("updated View() should still contain /%s", firstName)
	}
	if !strings.Contains(updatedView, "/"+secondName) {
		t.Errorf("updated View() should contain /%s", secondName)
	}
}

func TestView_ScrollWindow(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	// Move well past maxVisible to trigger scrolling.
	m.selected = m.maxVisible + 2
	m.clampScroll()

	view := m.View()
	// View should still be non-empty.
	if view == "" {
		t.Error("expected non-empty View after scrolling")
	}
}

// ---------------------------------------------------------------------------
// SetWidth
// ---------------------------------------------------------------------------

func TestSetWidth(t *testing.T) {
	m := NewSlashCmdModel()
	m.SetWidth(80)
	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
}

// ---------------------------------------------------------------------------
// scrollIndicator
// ---------------------------------------------------------------------------

func TestScrollIndicator_NoneWhenAllVisible(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")
	// Force a small filtered list so everything fits.
	m.filtered = m.filtered[:2]
	m.maxVisible = 8

	got := m.scrollIndicator(0, 2)
	if got != "" {
		t.Errorf("expected empty indicator when all items visible, got %q", got)
	}
}

func TestScrollIndicator_ShowsDownWhenMoreBelow(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	got := m.scrollIndicator(0, m.maxVisible)
	if !strings.Contains(got, "↓") {
		t.Errorf("expected '↓' when items exist below window, got %q", got)
	}
}

func TestScrollIndicator_ShowsUpWhenMoreAbove(t *testing.T) {
	m := NewSlashCmdModel()
	m.Show("")

	// Scroll down so there are items above.
	start := 3
	end := start + m.maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	got := m.scrollIndicator(start, end)
	if !strings.Contains(got, "↑") {
		t.Errorf("expected '↑' when items exist above window, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Table-driven: various query → expected match count
// ---------------------------------------------------------------------------

func TestFilter_TableDriven(t *testing.T) {
	tests := []struct {
		query        string
		wantAtLeast  int
		wantAtMost   int
		description  string
	}{
		{"", len(DefaultCommands()), len(DefaultCommands()), "empty query returns all"},
		{"ex", 2, 5, "'ex' matches explore, explore-add"},
		{"team", 3, 5, "'team' matches team-status, team-result, team-cancel, teams"},
		{"benchmark", 2, 4, "'benchmark' matches benchmark, benchmark-meta, benchmark-agent"},
		{"review", 2, 4, "'review' matches review, review-plan"},
		{"b", 2, 6, "'b' matches benchmark*, braintrust"},
		{"plan", 1, 3, "'plan' matches plan-tickets"},
		{"zzz_nomatch", 0, 0, "no-match returns 0"},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			m := NewSlashCmdModel()
			m.Show("")
			m.Filter(tc.query)

			got := len(m.filtered)
			if got < tc.wantAtLeast {
				t.Errorf("query %q: got %d matches, want at least %d", tc.query, got, tc.wantAtLeast)
			}
			if got > tc.wantAtMost {
				t.Errorf("query %q: got %d matches, want at most %d", tc.query, got, tc.wantAtMost)
			}
		})
	}
}
