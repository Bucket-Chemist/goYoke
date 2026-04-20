package teams_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/teams"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeConfig(name, status, createdAt string) teams.TeamConfig {
	return teams.TeamConfig{
		TeamName:     name,
		WorkflowType: "braintrust",
		Status:       status,
		CreatedAt:    createdAt,
	}
}

func makeConfigWithWaves(name, status string, waves []teams.Wave) teams.TeamConfig {
	cfg := makeConfig(name, status, "2026-01-01T00:00:00Z")
	cfg.Waves = waves
	return cfg
}

// ---------------------------------------------------------------------------
// NewTeamRegistry
// ---------------------------------------------------------------------------

func TestNewTeamRegistry_IsEmpty(t *testing.T) {
	r := teams.NewTeamRegistry()
	assert.Equal(t, 0, r.Count())
}

// ---------------------------------------------------------------------------
// TeamRegistry.Update — new team
// ---------------------------------------------------------------------------

func TestTeamRegistry_Update_NewTeam(t *testing.T) {
	r := teams.NewTeamRegistry()
	cfg := makeConfig("alpha", "running", "2026-01-01T10:00:00Z")

	r.Update("/sessions/alpha", cfg, nil)

	assert.Equal(t, 1, r.Count())
	ts := r.Get("/sessions/alpha")
	require.NotNil(t, ts)
	assert.Equal(t, "alpha", ts.Config.TeamName)
	assert.Equal(t, "running", ts.Config.Status)
}

// ---------------------------------------------------------------------------
// TeamRegistry.Update — existing team
// ---------------------------------------------------------------------------

func TestTeamRegistry_Update_Existing(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/alpha", makeConfig("alpha", "running", "2026-01-01T10:00:00Z"), nil)

	// Update same dir with new status.
	updated := makeConfig("alpha", "completed", "2026-01-01T10:00:00Z")
	r.Update("/sessions/alpha", updated, nil)

	assert.Equal(t, 1, r.Count(), "count should still be 1 after update")
	ts := r.Get("/sessions/alpha")
	require.NotNil(t, ts)
	assert.Equal(t, "completed", ts.Config.Status)
}

// ---------------------------------------------------------------------------
// TeamRegistry.All — sorted by CreatedAt descending
// ---------------------------------------------------------------------------

func TestTeamRegistry_All_SortedByCreatedAt(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/old", makeConfig("old", "completed", "2025-12-01T00:00:00Z"), nil)
	r.Update("/sessions/mid", makeConfig("mid", "running", "2026-01-01T00:00:00Z"), nil)
	r.Update("/sessions/new", makeConfig("new", "pending", "2026-06-01T00:00:00Z"), nil)

	all := r.All()
	require.Len(t, all, 3)

	// Newest first.
	assert.Equal(t, "new", all[0].Config.TeamName)
	assert.Equal(t, "mid", all[1].Config.TeamName)
	assert.Equal(t, "old", all[2].Config.TeamName)
}

func TestTeamRegistry_All_ReturnsEmptyWhenNoTeams(t *testing.T) {
	r := teams.NewTeamRegistry()
	all := r.All()
	assert.Empty(t, all)
}

// ---------------------------------------------------------------------------
// TeamRegistry.Get — returns copy (mutation safety)
// ---------------------------------------------------------------------------

func TestTeamRegistry_Get_ReturnsCopy(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/alpha", makeConfig("alpha", "running", "2026-01-01T00:00:00Z"), nil)

	ts := r.Get("/sessions/alpha")
	require.NotNil(t, ts)

	// Mutate the returned copy.
	ts.Config.Status = "MUTATED"

	// Registry must be unaffected.
	ts2 := r.Get("/sessions/alpha")
	require.NotNil(t, ts2)
	assert.Equal(t, "running", ts2.Config.Status, "registry state must not be affected by caller mutation")
}

func TestTeamRegistry_Get_ReturnsNilForMissing(t *testing.T) {
	r := teams.NewTeamRegistry()
	ts := r.Get("/sessions/nonexistent")
	assert.Nil(t, ts)
}

// ---------------------------------------------------------------------------
// TeamRegistry.Count
// ---------------------------------------------------------------------------

func TestTeamRegistry_Count_MultipleTeams(t *testing.T) {
	r := teams.NewTeamRegistry()
	for i := range 5 {
		dir := "/sessions/team-" + string(rune('a'+i))
		r.Update(dir, makeConfig("t", "pending", "2026-01-01T00:00:00Z"), nil)
	}
	assert.Equal(t, 5, r.Count())
}

// ---------------------------------------------------------------------------
// TeamState.TotalCostUSD
// ---------------------------------------------------------------------------

func TestTeamState_TotalCostUSD_SumsMemberCosts(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfigWithWaves("alpha", "running", []teams.Wave{
			{
				WaveNumber: 1,
				Members: []teams.Member{
					{Name: "a", CostUSD: 0.45},
					{Name: "b", CostUSD: 0.32},
				},
			},
			{
				WaveNumber: 2,
				Members: []teams.Member{
					{Name: "c", CostUSD: 0.10},
				},
			},
		}),
	}

	total := ts.TotalCostUSD()
	assert.InDelta(t, 0.87, total, 0.001)
}

func TestTeamState_TotalCostUSD_ZeroWhenNoWaves(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfig("alpha", "running", "2026-01-01T00:00:00Z"),
	}
	assert.Equal(t, 0.0, ts.TotalCostUSD())
}

// ---------------------------------------------------------------------------
// TeamState.CurrentWaveNumber
// ---------------------------------------------------------------------------

func TestTeamState_CurrentWaveNumber_ReturnsActiveWave(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfigWithWaves("alpha", "running", []teams.Wave{
			{WaveNumber: 1, Members: []teams.Member{{Status: "completed"}}},
			{WaveNumber: 2, Members: []teams.Member{{Status: "running"}}},
			{WaveNumber: 3, Members: []teams.Member{{Status: "pending"}}},
		}),
	}
	assert.Equal(t, 2, ts.CurrentWaveNumber())
}

func TestTeamState_CurrentWaveNumber_FallsBackToFirstWave(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfigWithWaves("alpha", "pending", []teams.Wave{
			{WaveNumber: 1, Members: []teams.Member{{Status: "pending"}}},
			{WaveNumber: 2, Members: []teams.Member{{Status: "pending"}}},
		}),
	}
	assert.Equal(t, 1, ts.CurrentWaveNumber())
}

func TestTeamState_CurrentWaveNumber_NoWavesReturnsOne(t *testing.T) {
	ts := &teams.TeamState{
		Config: makeConfig("alpha", "running", "2026-01-01T00:00:00Z"),
	}
	assert.Equal(t, 1, ts.CurrentWaveNumber())
}

// ---------------------------------------------------------------------------
// TeamRegistry.All — wave slice copy safety
// ---------------------------------------------------------------------------

func TestTeamRegistry_All_WavesCopied(t *testing.T) {
	r := teams.NewTeamRegistry()
	cfg := makeConfigWithWaves("alpha", "running", []teams.Wave{
		{WaveNumber: 1, Members: []teams.Member{{Name: "member-a", CostUSD: 1.0}}},
	})
	r.Update("/sessions/alpha", cfg, nil)

	all := r.All()
	require.Len(t, all, 1)

	// Mutate returned wave.
	all[0].Config.Waves[0].Members[0].Name = "MUTATED"

	// Registry should not reflect the mutation.
	ts := r.Get("/sessions/alpha")
	require.NotNil(t, ts)
	assert.Equal(t, "member-a", ts.Config.Waves[0].Members[0].Name)
}

// ---------------------------------------------------------------------------
// TeamRegistry.Update — LastPolled is set
// ---------------------------------------------------------------------------

func TestTeamRegistry_Update_SetsLastPolled(t *testing.T) {
	r := teams.NewTeamRegistry()
	before := time.Now()
	r.Update("/sessions/alpha", makeConfig("alpha", "running", "2026-01-01T00:00:00Z"), nil)
	after := time.Now()

	ts := r.Get("/sessions/alpha")
	require.NotNil(t, ts)
	assert.True(t, !ts.LastPolled.Before(before), "LastPolled should be >= before")
	assert.True(t, !ts.LastPolled.After(after), "LastPolled should be <= after")
}

// ---------------------------------------------------------------------------
// TeamRegistry.MostRecentRunning
// ---------------------------------------------------------------------------

func TestTeamRegistry_MostRecentRunning_ReturnsRunningTeam(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/done", makeConfig("done", "completed", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/run", makeConfig("run", "running", "2026-01-01T00:00:00Z"), nil)

	ts := r.MostRecentRunning()
	require.NotNil(t, ts)
	assert.Equal(t, "running", ts.Config.Status)
	assert.Equal(t, "run", ts.Config.TeamName)
}

func TestTeamRegistry_MostRecentRunning_FallsBackToMostRecent(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/old", makeConfig("old", "completed", "2026-01-01T00:00:00Z"), nil)
	r.Update("/sessions/new", makeConfig("new", "completed", "2026-06-01T00:00:00Z"), nil)

	ts := r.MostRecentRunning()
	require.NotNil(t, ts)
	assert.Equal(t, "new", ts.Config.TeamName, "should return most recent when none running")
}

func TestTeamRegistry_MostRecentRunning_ReturnsNilWhenEmpty(t *testing.T) {
	r := teams.NewTeamRegistry()
	ts := r.MostRecentRunning()
	assert.Nil(t, ts)
}
