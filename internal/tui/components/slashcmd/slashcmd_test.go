package slashcmd

import (
	"os"
	"path/filepath"
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

func TestNewSlashCmdModel_WithExtraCommands(t *testing.T) {
	extra := []SlashCommand{
		{Name: "custom-one", Description: "First custom command"},
		{Name: "custom-two", Description: "Second custom command"},
	}
	base := len(DefaultCommands())
	m := NewSlashCmdModel(extra...)
	if len(m.commands) != base+2 {
		t.Errorf("expected %d commands (defaults + 2 extra), got %d", base+2, len(m.commands))
	}
	// Verify extra commands appear in the list.
	names := make(map[string]bool, len(m.commands))
	for _, c := range m.commands {
		names[c.Name] = true
	}
	for _, e := range extra {
		if !names[e.Name] {
			t.Errorf("expected extra command %q in model commands", e.Name)
		}
	}
}

func TestNewSlashCmdModel_NoExtraBackwardCompat(t *testing.T) {
	// Calling with no args must behave exactly as before.
	m := NewSlashCmdModel()
	if len(m.commands) != len(DefaultCommands()) {
		t.Errorf("expected %d default commands when called with no args, got %d",
			len(DefaultCommands()), len(m.commands))
	}
}

func TestHelpText_IncludesExtraCommands(t *testing.T) {
	extra := SlashCommand{Name: "my-skill", Description: "My custom skill"}
	text := HelpText(extra)
	if !strings.Contains(text, "/my-skill") {
		t.Error("HelpText with extra commands should include /my-skill")
	}
	if !strings.Contains(text, "My custom skill") {
		t.Error("HelpText with extra commands should include description")
	}
}

func TestHelpText_NoExtraBackwardCompat(t *testing.T) {
	text := HelpText()
	if !strings.Contains(text, "/explore") {
		t.Error("HelpText() with no args should still include /explore")
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
// LoadSkillCommands
// ---------------------------------------------------------------------------

// makeSkillsDir creates a temporary config directory structure for testing.
// skills is a map of skill name → SKILL.md content ("" means no SKILL.md).
func makeSkillsDir(t *testing.T, skills map[string]string) string {
	t.Helper()
	configDir := t.TempDir()
	skillsDir := filepath.Join(configDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("makeSkillsDir: %v", err)
	}
	for name, content := range skills {
		dir := filepath.Join(skillsDir, name)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("makeSkillsDir mkdir %s: %v", name, err)
		}
		if content != "" {
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatalf("makeSkillsDir write SKILL.md for %s: %v", name, err)
			}
		}
	}
	return configDir
}

func skillNames(cmds []SlashCommand) []string {
	names := make([]string, len(cmds))
	for i, c := range cmds {
		names[i] = c.Name
	}
	return names
}

func TestLoadSkillCommands_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		skills     map[string]string // name → SKILL.md content ("" means no SKILL.md)
		noSkipsDir bool              // if true, don't create skills/ dir at all
		wantNames  []string
		wantDescs  map[string]string // name → expected description
		wantAbsent []string          // names that must NOT appear
	}{
		{
			name:       "no skills directory returns nil",
			noSkipsDir: true,
			wantNames:  nil,
		},
		{
			name:      "skill without SKILL.md gets default description",
			skills:    map[string]string{"my-skill": ""},
			wantNames: []string{"my-skill"},
			wantDescs: map[string]string{"my-skill": "Skill: my-skill"},
		},
		{
			name: "skill with frontmatter description extracted",
			skills: map[string]string{
				"scout": "---\nname: Scout\ndescription: Fast file metrics\n---\n\nBody.",
			},
			wantNames: []string{"scout"},
			wantDescs: map[string]string{"scout": "Fast file metrics"},
		},
		{
			name: "skill with quoted description strips quotes",
			skills: map[string]string{
				"planner": "---\ndescription: \"Plan a feature end-to-end\"\n---\n",
			},
			wantNames: []string{"planner"},
			wantDescs: map[string]string{"planner": "Plan a feature end-to-end"},
		},
		{
			name: "skill with no frontmatter uses default description",
			skills: map[string]string{
				"nofront": "Just plain markdown, no frontmatter.",
			},
			wantNames: []string{"nofront"},
			wantDescs: map[string]string{"nofront": "Skill: nofront"},
		},
		{
			name: "local command names are filtered out",
			skills: map[string]string{
				"clear": "", "exit": "", "quit": "", "help": "",
				"cwd": "", "model": "", "effort": "",
				"safe-skill": "",
			},
			wantNames:  []string{"safe-skill"},
			wantAbsent: []string{"clear", "exit", "quit", "help", "cwd", "model", "effort"},
		},
		{
			name: "multiple skills all returned",
			skills: map[string]string{
				"alpha": "---\ndescription: Alpha skill\n---\n",
				"beta":  "",
				"gamma": "---\ndescription: Gamma skill\n---\n",
			},
			wantNames: []string{"alpha", "beta", "gamma"},
			wantDescs: map[string]string{
				"alpha": "Alpha skill",
				"beta":  "Skill: beta",
				"gamma": "Gamma skill",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var configDir string
			if tc.noSkipsDir {
				configDir = t.TempDir()
			} else {
				configDir = makeSkillsDir(t, tc.skills)
			}

			cmds := LoadSkillCommands(configDir)

			names := make(map[string]bool, len(cmds))
			descs := make(map[string]string, len(cmds))
			for _, c := range cmds {
				names[c.Name] = true
				descs[c.Name] = c.Description
			}

			for _, want := range tc.wantNames {
				if !names[want] {
					t.Errorf("expected skill %q in result, got %v", want, skillNames(cmds))
				}
			}
			for skillName, wantDesc := range tc.wantDescs {
				got := descs[skillName]
				if got != wantDesc {
					t.Errorf("skill %q: want description %q, got %q", skillName, wantDesc, got)
				}
			}
			for _, absent := range tc.wantAbsent {
				if names[absent] {
					t.Errorf("skill %q should be filtered out but was present", absent)
				}
			}
		})
	}
}

func TestLoadSkillCommands_NonDirEntriesIgnored(t *testing.T) {
	configDir := t.TempDir()
	skillsDir := filepath.Join(configDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "not-a-dir.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(skillsDir, "real-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmds := LoadSkillCommands(configDir)
	names := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		names[c.Name] = true
	}

	if names["not-a-dir.md"] {
		t.Error("regular file 'not-a-dir.md' should not appear as a command")
	}
	if !names["real-skill"] {
		t.Error("expected 'real-skill' directory to appear as a command")
	}
}

// ---------------------------------------------------------------------------
// parseFrontmatterDescription
// ---------------------------------------------------------------------------

func TestParseFrontmatterDescription_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "no frontmatter returns empty",
			content: "# Just markdown",
			want:    "",
		},
		{
			name:    "frontmatter with description",
			content: "---\nname: foo\ndescription: My description\n---\n",
			want:    "My description",
		},
		{
			name:    "double-quoted description strips quotes",
			content: "---\ndescription: \"Quoted value\"\n---\n",
			want:    "Quoted value",
		},
		{
			name:    "single-quoted description strips quotes",
			content: "---\ndescription: 'Single quoted'\n---\n",
			want:    "Single quoted",
		},
		{
			name:    "frontmatter without description returns empty",
			content: "---\nname: foo\nauthor: bar\n---\n",
			want:    "",
		},
		{
			name:    "unclosed frontmatter returns empty",
			content: "---\ndescription: orphan\n",
			want:    "",
		},
		{
			name:    "empty description field returns empty string",
			content: "---\ndescription:\n---\n",
			want:    "",
		},
		{
			name:    "description with extra whitespace trimmed",
			content: "---\ndescription:   padded value   \n---\n",
			want:    "padded value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFrontmatterDescription(tc.content)
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
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
