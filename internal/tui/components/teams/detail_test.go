package teams_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/teams"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ptr returns a pointer to s.
func ptr(s string) *string {
	return &s
}

// makeFullTeamState returns a TeamState with waves and members for testing.
func makeFullTeamState() *teams.TeamState {
	return &teams.TeamState{
		Dir: "/sessions/full-team",
		Config: teams.TeamConfig{
			TeamName:     "full-team",
			WorkflowType: "braintrust",
			Status:       "running",
			CreatedAt:    "2026-01-01T10:00:00Z",
			Waves: []teams.Wave{
				{
					WaveNumber:  1,
					Description: "Analysis phase",
					Members: []teams.Member{
						{
							Name:      "einstein",
							Agent:     "einstein",
							Status:    "completed",
							CostUSD:   0.95,
							StartedAt: ptr("2026-01-01T10:00:00Z"),
							CompletedAt: ptr("2026-01-01T10:03:42Z"),
						},
						{
							Name:    "staff-arch",
							Agent:   "staff-architect-critical-review",
							Status:  "running",
							CostUSD: 0.45,
							StartedAt: ptr("2026-01-01T10:00:00Z"),
						},
					},
				},
				{
					WaveNumber:  2,
					Description: "Synthesis phase",
					Members: []teams.Member{
						{
							Name:   "beethoven",
							Agent:  "beethoven",
							Status: "pending",
						},
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewTeamDetailModel(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	assert.Nil(t, m.Init())
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_EmptyTeam(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	view := m.View()
	assert.Contains(t, view, "Select a team")
}

func TestTeamDetailModel_View_SetTeamNil_ShowsEmptyState(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetTeam(makeFullTeamState())
	m.SetTeam(nil)
	view := m.View()
	assert.Contains(t, view, "Select a team")
}

// ---------------------------------------------------------------------------
// Header row
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_ShowsTeamName(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()
	assert.Contains(t, view, "full-team")
}

func TestTeamDetailModel_View_ShowsWorkflowType(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()
	assert.Contains(t, view, "braintrust")
}

// ---------------------------------------------------------------------------
// Status + cost
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_ShowsStatus(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()
	assert.Contains(t, view, "running")
}

func TestTeamDetailModel_TotalCost_SumsMemberCosts(t *testing.T) {
	ts := makeFullTeamState()
	// Wave1: einstein $0.95 + staff-arch $0.45 = $1.40
	// Wave2: beethoven $0.00
	expected := 0.95 + 0.45
	assert.InDelta(t, expected, ts.TotalCostUSD(), 0.001)

	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	// The cost should appear somewhere in the view (formatted as $X.XXX).
	assert.Contains(t, view, "1.400", "total cost should appear in view")
}

// ---------------------------------------------------------------------------
// Wave-grouped rendering
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_WaveGrouped(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()

	// Both wave headers should appear.
	assert.Contains(t, view, "Wave 1")
	assert.Contains(t, view, "Wave 2")
	// Wave descriptions should appear.
	assert.Contains(t, view, "Analysis phase")
	assert.Contains(t, view, "Synthesis phase")
}

func TestTeamDetailModel_View_ShowsMemberNames(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()

	assert.Contains(t, view, "einstein")
	assert.Contains(t, view, "staff-arch")
	assert.Contains(t, view, "beethoven")
}

func TestTeamDetailModel_View_ShowsMemberStatus(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()

	assert.Contains(t, view, "completed")
	assert.Contains(t, view, "running")
	assert.Contains(t, view, "pending")
}

func TestTeamDetailModel_View_ShowsMemberCost(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()

	// einstein $0.95 should appear.
	assert.Contains(t, view, "0.95")
}

func TestTeamDetailModel_View_ShowsDashForZeroCost(t *testing.T) {
	ts := makeFullTeamState()
	// beethoven has CostUSD = 0, should show "—"
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	assert.Contains(t, view, "—")
}

// ---------------------------------------------------------------------------
// Status icons
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_ShowsStatusIconsForMembers(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(makeFullTeamState())
	view := m.View()

	// completed → '*', running → '>', pending → '.'
	for _, icon := range []string{"*", ">", "."} {
		assert.Contains(t, view, icon, "view should contain icon %q", icon)
	}
}

// ---------------------------------------------------------------------------
// Elapsed time rendering
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_ShowsElapsedForCompletedMember(t *testing.T) {
	ts := makeFullTeamState()
	// einstein: started 10:00:00, completed 10:03:42 → 3m42s
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	// Should contain minutes and seconds for einstein.
	assert.Contains(t, view, "3m", "completed member should show minutes elapsed")
}

func TestTeamDetailModel_View_ShowsDashForNilStartedAt(t *testing.T) {
	ts := makeFullTeamState()
	// beethoven has no StartedAt.
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	assert.Contains(t, view, "—", "pending member with nil StartedAt should show —")
}

// ---------------------------------------------------------------------------
// No waves
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_NoWaves_ShowsPlaceholder(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfig("empty-team", "pending", "2026-01-01T00:00:00Z"),
	}
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	assert.Contains(t, view, "No waves")
}

// ---------------------------------------------------------------------------
// Update is a no-op
// ---------------------------------------------------------------------------

func TestTeamDetailModel_Update_NoOp(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetTeam(makeFullTeamState())
	updated, cmd := m.Update(nil)
	assert.Nil(t, cmd)
	_ = updated
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestTeamDetailModel_SetSize_DoesNotPanic(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(0, 0)
	m.SetTeam(makeFullTeamState())
	_ = m.View() // should not panic
}

// ---------------------------------------------------------------------------
// Failed team status
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_FailedStatus(t *testing.T) {
	ts := &teams.TeamState{
		Config: teams.TeamConfig{
			TeamName:     "fail-team",
			WorkflowType: "impl",
			Status:       "failed",
			CreatedAt:    "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{
					WaveNumber:  1,
					Description: "Wave 1",
					Members: []teams.Member{
						{Name: "worker", Status: "failed", CostUSD: 0.10},
					},
				},
			},
		},
	}
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	assert.Contains(t, view, "failed")
	assert.Contains(t, view, "!")
}

// ---------------------------------------------------------------------------
// Divider
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_ContainsDivider(t *testing.T) {
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(40, 20)
	m.SetTeam(makeFullTeamState())
	view := m.View()
	assert.Contains(t, view, "─", "view should contain a horizontal divider")
}

// ---------------------------------------------------------------------------
// Multiple waves — all rendered
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_AllWavesRendered(t *testing.T) {
	ts := &teams.TeamState{
		Config: teams.TeamConfig{
			TeamName:  "multi-wave",
			Status:    "running",
			CreatedAt: "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{WaveNumber: 1, Description: "Phase 1", Members: []teams.Member{{Name: "a", Status: "completed"}}},
				{WaveNumber: 2, Description: "Phase 2", Members: []teams.Member{{Name: "b", Status: "running"}}},
				{WaveNumber: 3, Description: "Phase 3", Members: []teams.Member{{Name: "c", Status: "pending"}}},
			},
		},
	}
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()

	for _, expect := range []string{"Wave 1", "Wave 2", "Wave 3", "Phase 1", "Phase 2", "Phase 3", "a", "b", "c"} {
		assert.Contains(t, view, expect, "view should contain %q", expect)
	}
}

// ---------------------------------------------------------------------------
// Elapsed formatting
// ---------------------------------------------------------------------------

func TestTeamDetailModel_ElapsedFormatting_SubMinute(t *testing.T) {
	// Member with ~30s elapsed.
	start := time.Now().Add(-30 * time.Second).UTC().Format(time.RFC3339)
	end := time.Now().UTC().Format(time.RFC3339)
	ts := &teams.TeamState{
		Config: teams.TeamConfig{
			TeamName:  "timing-team",
			Status:    "completed",
			CreatedAt: "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{
					WaveNumber: 1,
					Members: []teams.Member{
						{Name: "fast", Status: "completed", CostUSD: 0.01, StartedAt: &start, CompletedAt: &end},
					},
				},
			},
		},
	}
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	// Should show something ending in "s" (seconds).
	assert.True(t, strings.Contains(view, "s"), "sub-minute elapsed should contain 's'; got:\n%s", view)
}

// ---------------------------------------------------------------------------
// Empty member name falls back to agent field
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_EmptyNameFallsBackToAgent(t *testing.T) {
	ts := &teams.TeamState{
		Config: teams.TeamConfig{
			TeamName:  "fallback-team",
			Status:    "running",
			CreatedAt: "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{
					WaveNumber: 1,
					Members: []teams.Member{
						{Name: "", Agent: "go-pro", Status: "running"},
					},
				},
			},
		},
	}
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	require.Contains(t, view, "go-pro", "view should fall back to Agent field when Name is empty")
}

// ---------------------------------------------------------------------------
// Completion summary (UX-028)
// ---------------------------------------------------------------------------

func TestTeamDetailModel_View_CompletedTeam_ShowsSummary(t *testing.T) {
	// A completed team with no stream files on disk still shows the summary
	// line (with "no file changes" since stream files are absent).
	ts := &teams.TeamState{
		Dir: t.TempDir(), // empty dir — no stream files
		Config: teams.TeamConfig{
			TeamName:  "done-team",
			Status:    "completed",
			CreatedAt: "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{WaveNumber: 1, Members: []teams.Member{
					{Name: "worker", Agent: "go-pro", Status: "completed", CostUSD: 1.0},
				}},
			},
		},
	}
	reg := teams.NewTeamRegistry()
	reg.Update(ts.Dir, ts.Config, nil)
	m := teams.NewTeamDetailModel(reg, nil)
	m.SetSize(120, 40)
	m.Refresh()
	view := m.View()
	assert.Contains(t, view, "✓ done", "completed team should show checkmark summary")
	assert.Contains(t, view, "no file changes", "no stream files → no file changes")
}

func TestTeamDetailModel_View_RunningTeam_NoSummary(t *testing.T) {
	// A running team must NOT show the completion summary.
	ts := makeFullTeamState() // status = "running"
	m := teams.NewTeamDetailModel(nil, nil)
	m.SetSize(120, 40)
	m.SetTeam(ts)
	view := m.View()
	assert.NotContains(t, view, "✓ done", "running team must not show completion summary")
}

func TestTeamDetailModel_View_CompleteStatus_ShowsSummary(t *testing.T) {
	// "complete" (without 'd') is also a valid terminal status.
	ts := &teams.TeamState{
		Dir: t.TempDir(),
		Config: teams.TeamConfig{
			TeamName:  "alt-complete",
			Status:    "complete",
			CreatedAt: "2026-01-01T00:00:00Z",
			Waves: []teams.Wave{
				{WaveNumber: 1, Members: []teams.Member{
					{Name: "a", Agent: "go-pro", Status: "completed", CostUSD: 0.5},
				}},
			},
		},
	}
	reg := teams.NewTeamRegistry()
	reg.Update(ts.Dir, ts.Config, nil)
	m := teams.NewTeamDetailModel(reg, nil)
	m.SetSize(120, 40)
	m.Refresh()
	view := m.View()
	assert.Contains(t, view, "✓ done", "status='complete' should also show summary")
}
