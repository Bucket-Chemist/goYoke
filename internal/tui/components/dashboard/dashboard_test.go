package dashboard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Existing tests (backward compatibility)
// ---------------------------------------------------------------------------

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel()
	if m.width != 0 || m.height != 0 {
		t.Errorf("expected zero dimensions, got w=%d h=%d", m.width, m.height)
	}
	if m.sessionCost != 0 {
		t.Errorf("expected zero cost, got %v", m.sessionCost)
	}
}

func TestSetSize(t *testing.T) {
	m := NewDashboardModel()
	m.SetSize(80, 24)
	if m.width != 80 || m.height != 24 {
		t.Errorf("expected 80x24, got %dx%d", m.width, m.height)
	}
}

func TestSetData(t *testing.T) {
	m := NewDashboardModel()
	start := time.Now()
	m.SetData(1.23, 5000, 10, 3, 2, start)
	if m.sessionCost != 1.23 {
		t.Errorf("expected cost 1.23, got %v", m.sessionCost)
	}
	if m.totalTokens != 5000 {
		t.Errorf("expected 5000 tokens, got %d", m.totalTokens)
	}
	if m.messageCount != 10 {
		t.Errorf("expected 10 messages, got %d", m.messageCount)
	}
	if m.agentCount != 3 {
		t.Errorf("expected 3 agents, got %d", m.agentCount)
	}
	if m.teamCount != 2 {
		t.Errorf("expected 2 teams, got %d", m.teamCount)
	}
	if !m.sessionStart.Equal(start) {
		t.Errorf("expected session start %v, got %v", start, m.sessionStart)
	}
}

func TestView_ContainsHeader(t *testing.T) {
	m := NewDashboardModel()
	view := m.View()
	if !strings.Contains(view, "Session Dashboard") {
		t.Errorf("expected 'Session Dashboard' in view, got:\n%s", view)
	}
}

func TestView_ContainsAllLabels(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(0.45, 12450, 8, 3, 1, time.Now())
	// Expand all sections so all labels appear.
	for i := range m.sections {
		m.sections[i].Expanded = true
	}
	view := m.View()

	labels := []string{"Duration:", "Total Cost:", "Tokens:", "Messages:", "Agents:", "Teams:"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("expected label %q in view, got:\n%s", label, view)
		}
	}
}

func TestView_CostFormatting(t *testing.T) {
	tests := []struct {
		cost     float64
		contains string
	}{
		{0, "$0.00"},
		{0.45, "$0.45"},
		{0.001, "$0.0010"},
		{1.23, "$1.23"},
	}
	for _, tc := range tests {
		m := NewDashboardModel()
		m.SetData(tc.cost, 0, 0, 0, 0, time.Time{})
		// Expand cost section so the value is rendered.
		m.sections[sectionCostTokens].Expanded = true
		view := m.View()
		if !strings.Contains(view, tc.contains) {
			t.Errorf("cost %v: expected %q in view, got:\n%s", tc.cost, tc.contains, view)
		}
	}
}

func TestView_TokenFormatting(t *testing.T) {
	tests := []struct {
		tokens   int64
		contains string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12450, "12,450"},
		{1000000, "1,000,000"},
	}
	for _, tc := range tests {
		m := NewDashboardModel()
		m.SetData(0, tc.tokens, 0, 0, 0, time.Time{})
		// Expand cost section so tokens are rendered.
		m.sections[sectionCostTokens].Expanded = true
		view := m.View()
		if !strings.Contains(view, tc.contains) {
			t.Errorf("tokens %d: expected %q in view, got:\n%s", tc.tokens, tc.contains, view)
		}
	}
}

func TestView_DurationZeroTime(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(0, 0, 0, 0, 0, time.Time{}) // zero sessionStart
	// Section 0 (Session Overview) starts expanded by default.
	view := m.View()
	if !strings.Contains(view, "\u2014") { // em dash
		t.Errorf("expected em dash for unknown duration, got:\n%s", view)
	}
}

func TestView_DurationNonZeroTime(t *testing.T) {
	m := NewDashboardModel()
	// Use a start time well in the past to guarantee a non-zero duration.
	m.SetData(0, 0, 0, 0, 0, time.Now().Add(-5*time.Minute))
	// Section 0 (Session Overview) starts expanded by default.
	view := m.View()
	// Should contain a "m" (minutes) character.
	if !strings.Contains(view, "m ") {
		t.Errorf("expected duration with minutes in view, got:\n%s", view)
	}
}

func TestView_EmptyState(t *testing.T) {
	m := NewDashboardModel()
	view := m.View()
	// Should not panic and should produce non-empty output.
	if view == "" {
		t.Error("expected non-empty view for zero-value model")
	}
}

func TestDivider(t *testing.T) {
	tests := []struct {
		width    int
		expected int // expected rune count
	}{
		{0, 20},  // fallback
		{30, 30}, // normal
		{50, 40}, // capped at 40
	}
	for _, tc := range tests {
		d := divider(tc.width)
		got := len([]rune(d))
		if got != tc.expected {
			t.Errorf("divider(%d): expected %d runes, got %d", tc.width, tc.expected, got)
		}
	}
}

func TestFormatInt64(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
	}
	for _, tc := range tests {
		got := formatInt64(tc.n)
		if got != tc.want {
			t.Errorf("formatInt64(%d): want %q, got %q", tc.n, tc.want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// New tests: initial section state
// ---------------------------------------------------------------------------

func TestNewDashboardModel_SectionCount(t *testing.T) {
	m := NewDashboardModel()
	if len(m.sections) != 4 {
		t.Errorf("expected 4 sections, got %d", len(m.sections))
	}
}

func TestNewDashboardModel_SectionTitles(t *testing.T) {
	m := NewDashboardModel()
	want := []string{"Session Overview", "Cost & Tokens", "Agent Activity", "Performance"}
	for i, title := range want {
		if m.sections[i].Title != title {
			t.Errorf("section[%d]: want %q, got %q", i, title, m.sections[i].Title)
		}
	}
}

func TestNewDashboardModel_InitialExpandState(t *testing.T) {
	m := NewDashboardModel()
	// Section 0 starts expanded; sections 1-3 start collapsed.
	if !m.sections[0].Expanded {
		t.Error("section 0 (Session Overview) should start expanded")
	}
	for i := 1; i < len(m.sections); i++ {
		if m.sections[i].Expanded {
			t.Errorf("section %d (%s) should start collapsed", i, m.sections[i].Title)
		}
	}
}

func TestNewDashboardModel_InitialCursor(t *testing.T) {
	m := NewDashboardModel()
	if m.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.cursor)
	}
}

// ---------------------------------------------------------------------------
// New tests: SetFocused
// ---------------------------------------------------------------------------

func TestSetFocused(t *testing.T) {
	m := NewDashboardModel()
	if m.focused {
		t.Error("expected initially unfocused")
	}
	m.SetFocused(true)
	if !m.focused {
		t.Error("expected focused after SetFocused(true)")
	}
	m.SetFocused(false)
	if m.focused {
		t.Error("expected unfocused after SetFocused(false)")
	}
}

// ---------------------------------------------------------------------------
// New tests: keyboard navigation
// ---------------------------------------------------------------------------

func TestNavigation_DownMovesCursor(t *testing.T) {
	tests := []struct {
		name       string
		startAt    int
		keyPresses int
		wantCursor int
	}{
		{"single down from 0", 0, 1, 1},
		{"double down from 0", 0, 2, 2},
		{"down to last", 0, 3, 3},
		{"down clamps at last", 0, 4, 3}, // 4 presses on 4 sections → stays at 3
		{"down from middle", 2, 1, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewDashboardModel()
			m.SetFocused(true)
			m.cursor = tc.startAt
			for i := 0; i < tc.keyPresses; i++ {
				m.Update(tea.KeyMsg{Type: tea.KeyDown})
			}
			if m.cursor != tc.wantCursor {
				t.Errorf("after %d down presses from %d: want cursor %d, got %d",
					tc.keyPresses, tc.startAt, tc.wantCursor, m.cursor)
			}
		})
	}
}

func TestNavigation_UpMovesCursor(t *testing.T) {
	tests := []struct {
		name       string
		startAt    int
		keyPresses int
		wantCursor int
	}{
		{"single up from 3", 3, 1, 2},
		{"double up from 3", 3, 2, 1},
		{"up to top", 3, 3, 0},
		{"up clamps at 0", 3, 4, 0}, // 4 presses on 4 sections → stays at 0
		{"up from middle", 2, 1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewDashboardModel()
			m.SetFocused(true)
			m.cursor = tc.startAt
			for i := 0; i < tc.keyPresses; i++ {
				m.Update(tea.KeyMsg{Type: tea.KeyUp})
			}
			if m.cursor != tc.wantCursor {
				t.Errorf("after %d up presses from %d: want cursor %d, got %d",
					tc.keyPresses, tc.startAt, tc.wantCursor, m.cursor)
			}
		})
	}
}

func TestNavigation_VimKeys(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)

	// j moves down
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("'j' should move cursor to 1, got %d", m.cursor)
	}

	// k moves up
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Errorf("'k' should move cursor back to 0, got %d", m.cursor)
	}
}

func TestNavigation_IgnoredWhenUnfocused(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(false)
	m.cursor = 0

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 0 {
		t.Errorf("unfocused: cursor should not move on Down, got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("unfocused: cursor should not move on Up, got %d", m.cursor)
	}
}

// ---------------------------------------------------------------------------
// New tests: expand/collapse toggle
// ---------------------------------------------------------------------------

func TestToggle_EnterExpandsCollapsedSection(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)
	m.cursor = 1 // Cost & Tokens — starts collapsed

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.sections[1].Expanded {
		t.Error("enter on collapsed section 1 should expand it")
	}
}

func TestToggle_EnterCollapsesExpandedSection(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)
	m.cursor = 0 // Session Overview — starts expanded

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.sections[0].Expanded {
		t.Error("enter on expanded section 0 should collapse it")
	}
}

func TestToggle_SpaceAlsoToggles(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)
	m.cursor = 2 // Agent Activity — starts collapsed

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.sections[2].Expanded {
		t.Error("space on collapsed section 2 should expand it")
	}
}

func TestToggle_RoundTrip(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)
	m.cursor = 3 // Performance

	// Expand
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.sections[3].Expanded {
		t.Error("first enter should expand section 3")
	}
	// Collapse
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.sections[3].Expanded {
		t.Error("second enter should collapse section 3")
	}
}

func TestToggle_IgnoredWhenUnfocused(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(false)
	m.cursor = 1
	originalExpanded := m.sections[1].Expanded

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.sections[1].Expanded != originalExpanded {
		t.Error("unfocused: enter should not toggle section")
	}
}

// ---------------------------------------------------------------------------
// New tests: expanded section shows correct metrics
// ---------------------------------------------------------------------------

func TestExpandedSection_SessionOverview(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(0, 0, 0, 0, 0, time.Now().Add(-2*time.Minute))
	// Section 0 is expanded by default.
	view := m.View()

	want := []string{"Session Overview", "Duration:", "Session ID:", "Model:", "Provider:"}
	for _, s := range want {
		if !strings.Contains(view, s) {
			t.Errorf("expanded Session Overview: expected %q in view", s)
		}
	}
}

func TestExpandedSection_CostTokens(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(1.5, 10000, 5, 2, 1, time.Time{})
	m.sections[sectionCostTokens].Expanded = true
	view := m.View()

	want := []string{"Cost & Tokens", "Total Cost:", "$1.50", "Tokens:", "10,000", "Messages:", "5"}
	for _, s := range want {
		if !strings.Contains(view, s) {
			t.Errorf("expanded Cost & Tokens: expected %q in view", s)
		}
	}
}

func TestExpandedSection_AgentActivity(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(0, 0, 0, 4, 2, time.Time{})
	m.sections[sectionAgents].Expanded = true
	view := m.View()

	want := []string{"Agent Activity", "Agents:", "4", "Teams:", "2", "Active:", "Completed:", "Errors:"}
	for _, s := range want {
		if !strings.Contains(view, s) {
			t.Errorf("expanded Agent Activity: expected %q in view", s)
		}
	}
}

func TestExpandedSection_Performance(t *testing.T) {
	m := NewDashboardModel()
	m.sections[sectionPerformance].Expanded = true
	view := m.View()

	want := []string{"Performance", "Events/sec:", "Modal Latency:", "Render Time:"}
	for _, s := range want {
		if !strings.Contains(view, s) {
			t.Errorf("expanded Performance: expected %q in view", s)
		}
	}
}

// ---------------------------------------------------------------------------
// New tests: collapsed section shows summary metric only
// ---------------------------------------------------------------------------

func TestCollapsedSection_ShowsSummaryMetric(t *testing.T) {
	tests := []struct {
		name           string
		sectionIdx     int
		sectionTitle   string
		setupData      func(*DashboardModel)
		wantSummary    string
		hiddenLabel    string
	}{
		{
			name:         "cost summary when collapsed",
			sectionIdx:   sectionCostTokens,
			sectionTitle: "Cost & Tokens",
			setupData: func(m *DashboardModel) {
				m.SetData(2.50, 0, 0, 0, 0, time.Time{})
			},
			wantSummary: "$2.50",
			hiddenLabel: "Tokens:",
		},
		{
			name:         "agent summary when collapsed",
			sectionIdx:   sectionAgents,
			sectionTitle: "Agent Activity",
			setupData: func(m *DashboardModel) {
				m.SetData(0, 0, 0, 7, 0, time.Time{})
			},
			wantSummary: "7 agents",
			hiddenLabel: "Active:",
		},
		{
			name:         "performance summary when collapsed",
			sectionIdx:   sectionPerformance,
			sectionTitle: "Performance",
			setupData:    func(m *DashboardModel) {},
			wantSummary:  "—",
			hiddenLabel:  "Render Time:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewDashboardModel()
			tc.setupData(&m)
			// Ensure section is collapsed.
			m.sections[tc.sectionIdx].Expanded = false

			view := m.View()

			if !strings.Contains(view, tc.sectionTitle) {
				t.Errorf("collapsed: expected section title %q in view", tc.sectionTitle)
			}
			if !strings.Contains(view, tc.wantSummary) {
				t.Errorf("collapsed: expected summary %q in view, got:\n%s", tc.wantSummary, view)
			}
			if strings.Contains(view, tc.hiddenLabel) {
				t.Errorf("collapsed: metric label %q should NOT appear in collapsed view", tc.hiddenLabel)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// New tests: section state icons appear in view
// ---------------------------------------------------------------------------

func TestView_SectionIcons(t *testing.T) {
	m := NewDashboardModel()
	// Section 0 expanded → Running icon (▶ or >)
	// Section 1 collapsed → Pending icon (○ or .)
	view := m.View()

	// We look for any recognizable icon character. The exact character depends
	// on the terminal unicode support (▶ or >), but at minimum the view must
	// contain some non-empty indicator. We test that neither "▶" nor ">" is
	// completely absent (one must appear since the ASCII fallback uses ">").
	hasExpanded := strings.Contains(view, "▶") || strings.Contains(view, ">")
	if !hasExpanded {
		t.Error("view should contain an expanded indicator (▶ or >)")
	}

	hasCollapsed := strings.Contains(view, "○") || strings.Contains(view, ".")
	if !hasCollapsed {
		t.Error("view should contain a collapsed indicator (○ or .)")
	}
}

// ---------------------------------------------------------------------------
// New tests: Update returns nil Cmd (no I/O side effects)
// ---------------------------------------------------------------------------

func TestUpdate_ReturnsNilCmd(t *testing.T) {
	m := NewDashboardModel()
	m.SetFocused(true)

	keys := []tea.KeyMsg{
		{Type: tea.KeyDown},
		{Type: tea.KeyUp},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
	}

	for _, k := range keys {
		cmd := m.Update(k)
		if cmd != nil {
			t.Errorf("Update(%v): expected nil Cmd, got non-nil", k)
		}
	}
}

// ---------------------------------------------------------------------------
// New tests: summaryMetric helper
// ---------------------------------------------------------------------------

func TestSummaryMetric(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(3.14, 0, 0, 5, 0, time.Time{})

	if s := m.summaryMetric(sectionCostTokens); s != "$3.14" {
		t.Errorf("cost summary: want $3.14, got %q", s)
	}
	if s := m.summaryMetric(sectionAgents); s != "5 agents" {
		t.Errorf("agent summary: want '5 agents', got %q", s)
	}
	if s := m.summaryMetric(sectionPerformance); s != "—" {
		t.Errorf("performance summary: want '—', got %q", s)
	}
}

// ---------------------------------------------------------------------------
// New tests: sectionMetrics row count sanity
// ---------------------------------------------------------------------------

func TestSectionMetrics_RowCounts(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(1, 1, 1, 1, 1, time.Now())

	tests := []struct {
		idx      int
		minRows  int
		name     string
	}{
		{sectionSession, 4, "Session Overview"},
		{sectionCostTokens, 5, "Cost & Tokens"},
		{sectionAgents, 5, "Agent Activity"},
		{sectionPerformance, 3, "Performance"},
	}

	for _, tc := range tests {
		rows := m.sectionMetrics(tc.idx)
		if len(rows) < tc.minRows {
			t.Errorf("%s: expected at least %d metric rows, got %d", tc.name, tc.minRows, len(rows))
		}
	}
}
