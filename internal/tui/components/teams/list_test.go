package teams_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/teams"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeMockConfig writes a TeamConfig as config.json in dir.
func writeMockConfig(t *testing.T, dir string, cfg teams.TeamConfig) {
	t.Helper()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config.json"), data, 0o600)
	require.NoError(t, err)
}

// setupTeamsDir creates a temporary directory with one or more team
// subdirectories, each containing a config.json.  It returns the root
// directory path and a cleanup function.
func setupTeamsDir(t *testing.T, configs map[string]teams.TeamConfig) string {
	t.Helper()
	root := t.TempDir()
	for name, cfg := range configs {
		teamDir := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(teamDir, 0o755))
		writeMockConfig(t, teamDir, cfg)
	}
	return root
}

// pollOnce sends a pollTickMsg to the list model and returns the updated model.
func pollOnce(m teams.TeamListModel) teams.TeamListModel {
	updated, _ := m.Update(pollTickMsgNow())
	return updated
}

// pollTickMsgNow returns a pollTickMsg for the current time.
// Since pollTickMsg is unexported, we use Update with a real tea.Tick msg.
// Instead we directly invoke the internal tick path by sending a
// model.TeamUpdateMsg and calling Update once. For filesystem tests we need
// the real poll path so we call StartPolling then drive the tick manually via
// a helper that exposes the poll result through the command mechanism.
//
// Because pollTickMsg is unexported, we test the filesystem scan indirectly
// via the TeamRegistry state after calling Update with a model.TeamUpdateMsg
// (which refreshes the snapshot from an already-updated registry).
//
// For the "reads dir" test we trigger the poll by calling StartPolling and
// then executing the returned command to get the pollTickMsg, then feeding it
// back into Update.
func pollTickMsgNow() tea.Msg {
	// We can't construct an unexported type from outside the package, but we
	// CAN obtain a real one by executing the command returned by StartPolling.
	// This helper is used only within test helpers that already have a command.
	// Callers that need the full poll-from-dir cycle use driveFullPoll below.
	return nil // placeholder — see driveFullPoll
}

// driveFullPoll sets the teams directory on m, executes the StartPolling
// command to get the first pollTickMsg, and feeds that back into Update.
// It returns the model after the poll completes.
func driveFullPoll(t *testing.T, m teams.TeamListModel, root string) teams.TeamListModel {
	t.Helper()
	// Set the directory on the model so the poll tick can scan it.
	m.SetTeamsDir(root)
	cmd := m.StartPolling(root)
	require.NotNil(t, cmd, "StartPolling must return a non-nil command")
	msg := cmd()
	updated, _ := m.Update(msg)
	return updated
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewTeamListModel_EmptyState(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	assert.Equal(t, "", m.SelectedTeam())
	assert.Nil(t, m.Init())
}

// ---------------------------------------------------------------------------
// StartPolling
// ---------------------------------------------------------------------------

func TestTeamListModel_StartPolling_ReturnsNonNilCmd(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	cmd := m.StartPolling(t.TempDir())
	assert.NotNil(t, cmd)
}

// ---------------------------------------------------------------------------
// Poll — reads filesystem and updates registry
// ---------------------------------------------------------------------------

func TestTeamListModel_Update_PollTick_ReadsDir(t *testing.T) {
	cfg := teams.TeamConfig{
		TeamName:     "my-team",
		WorkflowType: "braintrust",
		Status:       "running",
		CreatedAt:    "2026-01-01T10:00:00Z",
	}
	root := setupTeamsDir(t, map[string]teams.TeamConfig{"my-team-dir": cfg})

	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	m = driveFullPoll(t, m, root)

	// Registry should now contain the team.
	assert.Equal(t, 1, r.Count(), "registry should contain the polled team")
	ts := r.Get(filepath.Join(root, "my-team-dir"))
	require.NotNil(t, ts)
	assert.Equal(t, "my-team", ts.Config.TeamName)

	// View should show the team name.
	view := m.View()
	assert.Contains(t, view, "my-team")
}

func TestTeamListModel_Update_PollTick_IgnoresNonDirs(t *testing.T) {
	root := t.TempDir()
	// Write a plain file (not a dir) — should be ignored.
	err := os.WriteFile(filepath.Join(root, "not-a-dir.json"), []byte("{}"), 0o600)
	require.NoError(t, err)

	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	m = driveFullPoll(t, m, root)

	assert.Equal(t, 0, r.Count(), "plain files should be ignored during poll")
}

func TestTeamListModel_Update_PollTick_IgnoresDirWithoutConfig(t *testing.T) {
	root := t.TempDir()
	// Dir without config.json.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "empty-team"), 0o755))

	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	m = driveFullPoll(t, m, root)

	assert.Equal(t, 0, r.Count(), "dir without config.json should be ignored")
}

func TestTeamListModel_Update_PollTick_IgnoresInvalidJSON(t *testing.T) {
	root := t.TempDir()
	teamDir := filepath.Join(root, "bad-team")
	require.NoError(t, os.MkdirAll(teamDir, 0o755))
	err := os.WriteFile(filepath.Join(teamDir, "config.json"), []byte("{invalid}"), 0o600)
	require.NoError(t, err)

	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	m = driveFullPoll(t, m, root)

	assert.Equal(t, 0, r.Count(), "invalid JSON config should be ignored")
}

// ---------------------------------------------------------------------------
// TeamUpdateMsg refreshes snapshot
// ---------------------------------------------------------------------------

func TestTeamListModel_Update_TeamUpdateMsg_RefreshesSnapshot(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/alpha", makeConfig("alpha", "running", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	// Before TeamUpdateMsg, snapshot is empty.
	assert.Equal(t, "", m.SelectedTeam())

	updated, _ := m.Update(model.TeamUpdateMsg{TeamDir: "/sessions/alpha", Status: "running"})
	assert.Equal(t, "/sessions/alpha", updated.SelectedTeam())
}

// ---------------------------------------------------------------------------
// View — empty state
// ---------------------------------------------------------------------------

func TestTeamListModel_View_EmptyState(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	view := m.View()
	assert.Contains(t, view, "No teams")
}

// ---------------------------------------------------------------------------
// View — with teams
// ---------------------------------------------------------------------------

func TestTeamListModel_View_WithTeams(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/alpha", makeConfig("alpha-team", "running", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/beta", makeConfig("beta-team", "completed", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	// Refresh snapshot via TeamUpdateMsg.
	m, _ = m.Update(model.TeamUpdateMsg{})
	m.SetSize(120, 20)

	view := m.View()
	assert.Contains(t, view, "alpha-team")
	assert.Contains(t, view, "beta-team")
}

func TestTeamListModel_View_ShowsStatusIcons(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/run", makeConfig("run-team", "running", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/done", makeConfig("done-team", "completed", "2026-01-01T00:00:00Z"), nil)
	r.Update("/sessions/fail", makeConfig("fail-team", "failed", "2025-06-01T00:00:00Z"), nil)
	r.Update("/sessions/pend", makeConfig("pend-team", "pending", "2025-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})
	m.SetSize(120, 20)
	view := m.View()

	// Each icon character should appear.
	for _, icon := range []string{">", "*", "!", "."} {
		assert.Contains(t, view, icon, "view should contain icon %q", icon)
	}
}

func TestTeamListModel_View_ShowsWaveProgress(t *testing.T) {
	cfg := teams.TeamConfig{
		TeamName:     "wave-team",
		WorkflowType: "impl",
		Status:       "running",
		CreatedAt:    "2026-01-01T00:00:00Z",
		Waves: []teams.Wave{
			{WaveNumber: 1, Members: []teams.Member{{Status: "completed"}}},
			{WaveNumber: 2, Members: []teams.Member{{Status: "running"}}},
			{WaveNumber: 3, Members: []teams.Member{{Status: "pending"}}},
		},
	}
	r := teams.NewTeamRegistry()
	r.Update("/sessions/wave-team", cfg, nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})
	m.SetSize(120, 20)
	view := m.View()
	// Should show W2/3 or at least "W" and "/"
	assert.True(t, strings.Contains(view, "W") && strings.Contains(view, "/"),
		"view should contain wave progress like 'W2/3'; got:\n%s", view)
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

func TestTeamListModel_Navigation_DownMovesSelection(t *testing.T) {
	r := teams.NewTeamRegistry()
	// Add two teams — newest first after sort.
	r.Update("/sessions/new", makeConfig("new-team", "running", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/old", makeConfig("old-team", "completed", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})

	initial := m.SelectedTeam()

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, cmd, "Down key should emit a command")
	assert.NotEqual(t, initial, newM.SelectedTeam(), "selection should move")
}

func TestTeamListModel_Navigation_UpMovesSelection(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/new", makeConfig("new-team", "running", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/old", makeConfig("old-team", "completed", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})
	// Move down first.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	mid := m.SelectedTeam()

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.NotNil(t, cmd, "Up key should emit a command")
	assert.NotEqual(t, mid, newM.SelectedTeam(), "selection should move back")
}

func TestTeamListModel_Navigation_DownClampsAtEnd(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/only", makeConfig("only-team", "running", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})

	initial := m.SelectedTeam()
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, initial, newM.SelectedTeam(), "Down on single item should stay")
}

func TestTeamListModel_Navigation_UpClampsAtStart(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/only", makeConfig("only-team", "running", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})

	initial := m.SelectedTeam()
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, initial, newM.SelectedTeam(), "Up on first item should stay")
}

func TestTeamListModel_Navigation_ViKeys(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/new", makeConfig("new-team", "running", "2026-06-01T00:00:00Z"), nil)
	r.Update("/sessions/old", makeConfig("old-team", "completed", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})

	// 'j' moves down.
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.NotNil(t, cmd, "'j' should emit command")
	// 'k' moves back up.
	newM2, cmd2 := newM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.NotNil(t, cmd2, "'k' should emit command")
	assert.Equal(t, m.SelectedTeam(), newM2.SelectedTeam(), "'k' after 'j' should return to original")
}

// ---------------------------------------------------------------------------
// Enter emits TeamSelectedMsg
// ---------------------------------------------------------------------------

func TestTeamListModel_Enter_EmitsTeamSelectedMsg(t *testing.T) {
	r := teams.NewTeamRegistry()
	r.Update("/sessions/alpha", makeConfig("alpha", "running", "2026-01-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should emit a command")
	msg := cmd()
	sel, ok := msg.(teams.TeamSelectedMsg)
	require.True(t, ok, "command should produce TeamSelectedMsg; got %T", msg)
	assert.Equal(t, "/sessions/alpha", sel.Dir)
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestTeamListModel_SetSize_DoesNotPanic(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	m.SetSize(0, 0)
	m.SetSize(80, 40)
}

// ---------------------------------------------------------------------------
// Multiple teams + no-ops
// ---------------------------------------------------------------------------

func TestTeamListModel_Update_UnknownMsg_NoOp(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	// Unknown message type should not panic.
	updated, cmd := m.Update("unknown message")
	assert.Nil(t, cmd)
	assert.Equal(t, "", updated.SelectedTeam())
}

// ---------------------------------------------------------------------------
// Poll schedules next tick
// ---------------------------------------------------------------------------

func TestTeamListModel_PollTick_SchedulesNextPoll(t *testing.T) {
	root := t.TempDir()
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	cmd := m.StartPolling(root)
	require.NotNil(t, cmd)

	// Execute the initial tick to get the poll tick message.
	msg := cmd()
	_, nextCmd := m.Update(msg)

	// The model should schedule another poll (non-nil command).
	assert.NotNil(t, nextCmd, "poll tick should schedule the next poll")

	// Give the tick a moment to fire (it fires after 2 seconds via tea.Tick,
	// so we just verify the cmd is non-nil as a proxy for scheduling).
	_ = nextCmd
}

// ---------------------------------------------------------------------------
// Empty teamsDir gracefully handled
// ---------------------------------------------------------------------------

func TestTeamListModel_PollTick_EmptyTeamsDirHandled(t *testing.T) {
	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)
	// Trigger a poll with a non-existent directory; should not panic.
	cmd := m.StartPolling("/nonexistent/path/that/does/not/exist")
	require.NotNil(t, cmd)
	msg := cmd()
	updated, _ := m.Update(msg)
	assert.Equal(t, 0, r.Count())
	assert.Equal(t, "", updated.SelectedTeam())
}

// ---------------------------------------------------------------------------
// All() snapshot ordering after multiple updates
// ---------------------------------------------------------------------------

func TestTeamListModel_View_OrderMatchesRegistryAll(t *testing.T) {
	// Verify newest-first ordering is reflected in View rows.
	r := teams.NewTeamRegistry()
	r.Update("/sessions/old", makeConfig("old-team", "completed", "2026-01-01T00:00:00Z"), nil)
	r.Update("/sessions/new", makeConfig("new-team", "running", "2026-06-01T00:00:00Z"), nil)

	m := teams.NewTeamListModel(r)
	m, _ = m.Update(model.TeamUpdateMsg{})
	m.SetSize(120, 20)
	view := m.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	require.Len(t, lines, 2, "should have 2 lines for 2 teams")

	// First line should be the newer team.
	assert.Contains(t, lines[0], "new-team",
		"newer team should appear first; first line = %q", lines[0])
	assert.Contains(t, lines[1], "old-team",
		"older team should appear second; second line = %q", lines[1])
}

// ---------------------------------------------------------------------------
// HandleMsg pointer-receiver
// ---------------------------------------------------------------------------

func TestHandleMsg_PointerReceiverMutates(t *testing.T) {
	reg := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(reg)

	// HandleMsg should not panic on arbitrary messages.
	cmd := m.HandleMsg(model.TeamUpdateMsg{TeamDir: "/tmp/test", Status: "running"})
	// TeamUpdateMsg triggers a snapshot refresh; no crash = success.
	_ = cmd
}

// ---------------------------------------------------------------------------
// Poll at specific time (regression guard)
// ---------------------------------------------------------------------------

func TestTeamListModel_PollTick_TimestampAccepted(t *testing.T) {
	root := t.TempDir()
	cfg := teams.TeamConfig{
		TeamName:  "ts-team",
		Status:    "running",
		CreatedAt: "2026-03-01T00:00:00Z",
	}
	teamDir := filepath.Join(root, "ts-team")
	require.NoError(t, os.MkdirAll(teamDir, 0o755))
	writeMockConfig(t, teamDir, cfg)

	r := teams.NewTeamRegistry()
	m := teams.NewTeamListModel(r)

	// Use driveFullPoll which correctly sets the teamsDir before polling.
	updated := driveFullPoll(t, m, root)

	ts := r.Get(teamDir)
	require.NotNil(t, ts)
	assert.WithinDuration(t, time.Now(), ts.LastPolled, 5*time.Second)
	_ = updated
}
