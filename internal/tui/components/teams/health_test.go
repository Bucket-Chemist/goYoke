package teams

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
