package dashboard

import (
	"strings"
	"testing"
	"time"
)

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
	view := m.View()

	labels := []string{"Cost:", "Tokens:", "Messages:", "Agents:", "Teams:", "Duration:"}
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
		view := m.View()
		if !strings.Contains(view, tc.contains) {
			t.Errorf("tokens %d: expected %q in view, got:\n%s", tc.tokens, tc.contains, view)
		}
	}
}

func TestView_DurationZeroTime(t *testing.T) {
	m := NewDashboardModel()
	m.SetData(0, 0, 0, 0, 0, time.Time{}) // zero sessionStart
	view := m.View()
	if !strings.Contains(view, "\u2014") { // em dash
		t.Errorf("expected em dash for unknown duration, got:\n%s", view)
	}
}

func TestView_DurationNonZeroTime(t *testing.T) {
	m := NewDashboardModel()
	// Use a start time well in the past to guarantee a non-zero duration.
	m.SetData(0, 0, 0, 0, 0, time.Now().Add(-5*time.Minute))
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
