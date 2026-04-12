package teams

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		input string
		check func(got string) bool
	}{
		{
			name:  "empty string",
			input: "",
			check: func(got string) bool { return got == "" },
		},
		{
			name:  "invalid format",
			input: "not-a-date",
			check: func(got string) bool { return got == "" },
		},
		{
			// RFC3339 has second precision; the parsed time is within 1s of now,
			// so the result is "just now" or at most "1s ago".
			name:  "just now",
			input: now.Format(time.RFC3339),
			check: func(got string) bool {
				return got == "just now" || got == "1s ago"
			},
		},
		{
			name:  "seconds ago",
			input: now.Add(-45 * time.Second).Format(time.RFC3339),
			check: func(got string) bool { return strings.HasSuffix(got, "s ago") },
		},
		{
			name:  "minutes ago",
			input: now.Add(-5 * time.Minute).Format(time.RFC3339),
			check: func(got string) bool {
				return strings.Contains(got, "m") && strings.HasSuffix(got, "ago")
			},
		},
		{
			name:  "hours ago",
			input: now.Add(-2 * time.Hour).Format(time.RFC3339),
			check: func(got string) bool {
				return strings.HasPrefix(got, "2h") && strings.HasSuffix(got, "ago")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRelativeTime(tc.input)
			if !tc.check(got) {
				t.Errorf("formatRelativeTime(%q) = %q, failed check", tc.input, got)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0B"},
		{500, "500B"},
		{1023, "1023B"},
		{1024, "1KB"},
		{1500, "1KB"},
		{1048576, "1.0MB"},
		{2621440, "2.5MB"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := formatBytes(tc.input)
			if got != tc.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHealthIcon(t *testing.T) {
	tests := []struct {
		name     string
		member   Member
		wantIcon string
	}{
		{"healthy running", Member{Status: "running", HealthStatus: "healthy"}, "●"},
		{"default running (empty health)", Member{Status: "running"}, "●"},
		{"stall_warning", Member{Status: "running", HealthStatus: "stall_warning"}, "▲"},
		{"stalled", Member{Status: "running", HealthStatus: "stalled"}, "●"},
		{"pending", Member{Status: "pending"}, "◻"},
		{"failed", Member{Status: "failed"}, "✕"},
		{"completed", Member{Status: "completed"}, "●"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := healthIcon(tc.member)
			if !strings.Contains(got, tc.wantIcon) {
				t.Errorf("healthIcon(%+v) = %q, want icon %q", tc.member, got, tc.wantIcon)
			}
		})
	}
}

func TestBudgetColor(t *testing.T) {
	// budgetColor returns exactly one of the three package-level style vars.
	// reflect.DeepEqual is safe here because all three vars are value types
	// with the same *Renderer pointer (the default renderer).
	if !reflect.DeepEqual(budgetColor(0.50), config.StyleSuccess) {
		t.Error("50% usage should return StyleSuccess")
	}
	if !reflect.DeepEqual(budgetColor(0.699), config.StyleSuccess) {
		t.Error("69.9% usage should return StyleSuccess")
	}
	if !reflect.DeepEqual(budgetColor(0.70), config.StyleWarning) {
		t.Error("70% usage should return StyleWarning")
	}
	if !reflect.DeepEqual(budgetColor(0.90), config.StyleWarning) {
		t.Error("90% usage should return StyleWarning")
	}
	if !reflect.DeepEqual(budgetColor(0.901), config.StyleError) {
		t.Error("90.1% usage should return StyleError")
	}
	if !reflect.DeepEqual(budgetColor(0.95), config.StyleError) {
		t.Error("95% usage should return StyleError")
	}
}

func TestTeamsHealthModel_EmptyState(t *testing.T) {
	reg := NewTeamRegistry()
	m := NewTeamsHealthModel(reg)
	view := m.View()
	if !strings.Contains(view, "No active teams") {
		t.Errorf("empty registry should render 'No active teams', got: %q", view)
	}
}

func TestCalcProgress(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		timeout time.Duration
		want    float64
	}{
		{"zero timeout is indeterminate", 30 * time.Second, 0, -1.0},
		{"negative timeout is indeterminate", 30 * time.Second, -time.Second, -1.0},
		{"negative elapsed clamps to 0", -5 * time.Second, 60 * time.Second, 0.0},
		{"0% progress", 0, 60 * time.Second, 0.0},
		{"50% progress", 30 * time.Second, 60 * time.Second, 0.5},
		{"100% progress exact", 60 * time.Second, 60 * time.Second, 1.0},
		{"overtime capped at 100%", 90 * time.Second, 60 * time.Second, 1.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calcProgress(tc.elapsed, tc.timeout)
			if got != tc.want {
				t.Errorf("calcProgress(%v, %v) = %v, want %v", tc.elapsed, tc.timeout, got, tc.want)
			}
		})
	}
}

func TestMemberStatusStyle(t *testing.T) {
	tests := []struct {
		status string
		want   lipgloss.Style
	}{
		{"running", config.StyleSuccess},
		{"completed", config.StyleSuccess.Bold(true)},
		{"failed", config.StyleError},
		{"killed", config.StyleWarning},
		{"pending", config.StyleMuted},
		{"skipped", config.StyleMuted},
		{"unknown", config.StyleMuted},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := memberStatusStyle(tc.status)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("memberStatusStyle(%q): style mismatch", tc.status)
			}
		})
	}
}

func TestRenderMemberProgressBar(t *testing.T) {
	m := NewTeamsHealthModel(NewTeamRegistry())
	m.SetSize(80, 24)

	started := time.Now().Add(-2 * time.Minute).Format(time.RFC3339)
	completed := time.Now().Add(-time.Minute).Format(time.RFC3339)

	tests := []struct {
		name        string
		member      Member
		wantEmpty   bool
		wantContain []string
	}{
		{
			name:      "pending returns empty string",
			member:    Member{Name: "agent1", Status: "pending"},
			wantEmpty: true,
		},
		{
			name:      "skipped returns empty string",
			member:    Member{Name: "agent1", Status: "skipped"},
			wantEmpty: true,
		},
		{
			name:        "running shows indeterminate bar",
			member:      Member{Name: "agent1", Status: "running", StartedAt: &started},
			wantContain: []string{"agent1", "[", "]", "—"},
		},
		{
			name:        "completed shows 100% bar",
			member:      Member{Name: "agent2", Status: "completed", StartedAt: &started, CompletedAt: &completed},
			wantContain: []string{"agent2", "[", "]", "100%"},
		},
		{
			name:        "failed shows 100% bar",
			member:      Member{Name: "agent3", Status: "failed", StartedAt: &started, CompletedAt: &completed},
			wantContain: []string{"agent3", "[", "]", "100%"},
		},
		{
			name:        "killed shows 100% bar",
			member:      Member{Name: "agent4", Status: "killed", StartedAt: &started, CompletedAt: &completed},
			wantContain: []string{"agent4", "[", "]", "100%"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.renderMemberProgressBar(tc.member)
			if tc.wantEmpty {
				if got != "" {
					t.Errorf("renderMemberProgressBar: got %q, want empty string", got)
				}
				return
			}
			for _, sub := range tc.wantContain {
				if !strings.Contains(got, sub) {
					t.Errorf("renderMemberProgressBar: result %q missing %q", got, sub)
				}
			}
		})
	}
}

func TestRenderWaveProgressBar(t *testing.T) {
	m := NewTeamsHealthModel(NewTeamRegistry())
	m.SetSize(80, 24)

	tests := []struct {
		name        string
		wave        Wave
		wantEmpty   bool
		wantContain []string
	}{
		{
			name:      "empty wave returns empty string",
			wave:      Wave{WaveNumber: 1},
			wantEmpty: true,
		},
		{
			name: "all pending shows 0/N",
			wave: Wave{WaveNumber: 1, Members: []Member{
				{Name: "a1", Status: "pending"},
				{Name: "a2", Status: "pending"},
			}},
			wantContain: []string{"[", "]", "0/2"},
		},
		{
			name: "partial completion shows count",
			wave: Wave{WaveNumber: 1, Members: []Member{
				{Name: "a1", Status: "completed"},
				{Name: "a2", Status: "running"},
				{Name: "a3", Status: "pending"},
			}},
			wantContain: []string{"[", "]", "1/3"},
		},
		{
			name: "all completed shows full count",
			wave: Wave{WaveNumber: 1, Members: []Member{
				{Name: "a1", Status: "completed"},
				{Name: "a2", Status: "completed"},
			}},
			wantContain: []string{"[", "]", "2/2"},
		},
		{
			name: "failed member counts toward done",
			wave: Wave{WaveNumber: 1, Members: []Member{
				{Name: "a1", Status: "failed"},
				{Name: "a2", Status: "pending"},
			}},
			wantContain: []string{"[", "]", "1/2"},
		},
		{
			name: "killed member counts toward done",
			wave: Wave{WaveNumber: 1, Members: []Member{
				{Name: "a1", Status: "killed"},
				{Name: "a2", Status: "completed"},
				{Name: "a3", Status: "running"},
			}},
			wantContain: []string{"[", "]", "2/3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.renderWaveProgressBar(tc.wave)
			if tc.wantEmpty {
				if got != "" {
					t.Errorf("renderWaveProgressBar: got %q, want empty string", got)
				}
				return
			}
			for _, sub := range tc.wantContain {
				if !strings.Contains(got, sub) {
					t.Errorf("renderWaveProgressBar: result %q missing %q", got, sub)
				}
			}
		})
	}
}

func TestTeamsHealthModel_HasRunningTeam(t *testing.T) {
	t.Run("empty registry returns false", func(t *testing.T) {
		reg := NewTeamRegistry()
		m := NewTeamsHealthModel(reg)
		if m.HasRunningTeam() {
			t.Error("HasRunningTeam should return false on empty registry")
		}
	})

	t.Run("running team returns true", func(t *testing.T) {
		reg := NewTeamRegistry()
		cfg := TeamConfig{TeamName: "active", Status: "running", CreatedAt: "2026-01-01T00:00:00Z"}
		reg.Update("/sessions/active", cfg, nil)
		m := NewTeamsHealthModel(reg)
		if !m.HasRunningTeam() {
			t.Error("HasRunningTeam should return true when a running team exists")
		}
	})

	t.Run("completed team returns false", func(t *testing.T) {
		reg := NewTeamRegistry()
		cfg := TeamConfig{TeamName: "done", Status: "completed", CreatedAt: "2026-01-01T00:00:00Z"}
		reg.Update("/sessions/done", cfg, nil)
		m := NewTeamsHealthModel(reg)
		if m.HasRunningTeam() {
			t.Error("HasRunningTeam should return false when only a completed team exists")
		}
	})
}
