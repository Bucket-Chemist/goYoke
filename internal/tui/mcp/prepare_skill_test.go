package mcp

import (
	"encoding/json"
	"os"
	"syscall"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/skillsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockStore_AcquireAndRelease(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	ls := NewLockStore()
	sessionID := "test-lock-session"

	err := ls.Acquire(sessionID)
	require.NoError(t, err)

	// Lock file should exist.
	lockPath := config.GetGuardLockPath(sessionID)
	assert.FileExists(t, lockPath)

	// Release should clean up.
	ls.Release(sessionID)
	assert.NoFileExists(t, lockPath)
}

func TestLockStore_Acquire_Idempotent(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	ls := NewLockStore()
	sessionID := "test-idempotent"

	require.NoError(t, ls.Acquire(sessionID))
	require.NoError(t, ls.Acquire(sessionID))

	ls.Release(sessionID)
}

func TestLockStore_Release_Idempotent(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	ls := NewLockStore()
	ls.Release("never-acquired")
}

func TestLockStore_CloseAll(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	ls := NewLockStore()
	require.NoError(t, ls.Acquire("session-a"))
	require.NoError(t, ls.Acquire("session-b"))

	ls.CloseAll()

	assert.NoFileExists(t, config.GetGuardLockPath("session-a"))
	assert.NoFileExists(t, config.GetGuardLockPath("session-b"))
}

func TestLockStore_Acquire_DetectedByIsGuardStale(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	ls := NewLockStore()
	sessionID := "test-flock-detection"
	lockPath := config.GetGuardLockPath(sessionID)

	require.NoError(t, ls.Acquire(sessionID))

	// Verify the exclusive lock is held by attempting a non-blocking shared lock
	// from this process (same technique as isGuardStale in daemon.go).
	fd, err := os.Open(lockPath)
	require.NoError(t, err)
	defer fd.Close()

	err = syscall.Flock(int(fd.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	assert.ErrorIs(t, err, syscall.EWOULDBLOCK,
		"shared lock should fail with EWOULDBLOCK when exclusive lock is held")

	// After release, shared lock should succeed (guard is stale).
	ls.Release(sessionID)

	// Lock file is removed by Release, so re-check is not needed —
	// isGuardStale returns true when the file doesn't exist.
}

func TestHandlePrepareSkill_Setup_NonTeamSkill(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	origSessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	origMCPConfig := os.Getenv("GOYOKE_MCP_CONFIG")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
		os.Setenv("GOYOKE_SESSION_DIR", origSessionDir)
		os.Setenv("GOYOKE_MCP_CONFIG", origMCPConfig)
	}()

	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	os.Setenv("GOYOKE_SESSION_DIR", t.TempDir())
	os.Setenv("GOYOKE_MCP_CONFIG", "")

	ls := NewLockStore()
	uds := NewUDSClient()
	input := PrepareSkillInput{Skill: "dummies-guide"}

	_, output, err := handlePrepareSkill(nil, nil, input, uds, ls)
	require.NoError(t, err)
	assert.Equal(t, "dummies-guide", output.Skill)
	assert.False(t, output.GuardActive)
	assert.Empty(t, output.TeamDir)
	assert.Empty(t, output.TUITranslation)
}

func TestHandlePrepareSkill_MissingSessionDir(t *testing.T) {
	origSessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	defer os.Setenv("GOYOKE_SESSION_DIR", origSessionDir)
	os.Setenv("GOYOKE_SESSION_DIR", "")

	ls := NewLockStore()
	uds := NewUDSClient()
	input := PrepareSkillInput{Skill: "braintrust"}

	_, output, err := handlePrepareSkill(nil, nil, input, uds, ls)
	require.NoError(t, err)
	assert.Contains(t, output.Error, "GOYOKE_SESSION_DIR not set")
}

func TestHandlePrepareSkill_TUITranslation_OnlyInTUIMode(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	origSessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	origMCPConfig := os.Getenv("GOYOKE_MCP_CONFIG")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
		os.Setenv("GOYOKE_SESSION_DIR", origSessionDir)
		os.Setenv("GOYOKE_MCP_CONFIG", origMCPConfig)
	}()

	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	os.Setenv("GOYOKE_SESSION_DIR", t.TempDir())

	ls := NewLockStore()
	uds := NewUDSClient()

	// Without GOYOKE_MCP_CONFIG → no translation
	os.Setenv("GOYOKE_MCP_CONFIG", "")
	_, output, err := handlePrepareSkill(nil, nil, PrepareSkillInput{Skill: "dummies-guide"}, uds, ls)
	require.NoError(t, err)
	assert.Empty(t, output.TUITranslation)

	// With GOYOKE_MCP_CONFIG → translation injected
	os.Setenv("GOYOKE_MCP_CONFIG", "/tmp/fake-mcp-config.json")
	_, output, err = handlePrepareSkill(nil, nil, PrepareSkillInput{Skill: "dummies-guide"}, uds, ls)
	require.NoError(t, err)
	assert.Contains(t, output.TUITranslation, "TOOL TRANSLATION")
}

func TestHandlePrepareSkill_Setup_WithFixture(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	origSessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	origSessionID := os.Getenv("GOYOKE_SESSION_ID")
	origMCPConfig := os.Getenv("GOYOKE_MCP_CONFIG")
	origConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
		os.Setenv("GOYOKE_SESSION_DIR", origSessionDir)
		os.Setenv("GOYOKE_SESSION_ID", origSessionID)
		os.Setenv("GOYOKE_MCP_CONFIG", origMCPConfig)
		os.Setenv("CLAUDE_CONFIG_DIR", origConfigDir)
	}()

	xdgDir := t.TempDir()
	sessionDir := t.TempDir()
	configDir := t.TempDir()

	os.Setenv("XDG_RUNTIME_DIR", xdgDir)
	os.Setenv("GOYOKE_SESSION_DIR", sessionDir)
	os.Setenv("GOYOKE_SESSION_ID", "test-fixture-session")
	os.Setenv("GOYOKE_MCP_CONFIG", "")
	os.Setenv("CLAUDE_CONFIG_DIR", configDir)

	// Write a fixture agents-index.json
	agentsDir := configDir + "/agents"
	require.NoError(t, os.MkdirAll(agentsDir, 0755))
	index := map[string]any{
		"skill_guards": map[string]any{
			"braintrust": map[string]any{
				"router_allowed_tools": []string{"Task", "Bash", "Read"},
				"team_dir_suffix":      "braintrust",
			},
		},
	}
	data, _ := json.Marshal(index)
	require.NoError(t, os.WriteFile(agentsDir+"/agents-index.json", data, 0644))

	ls := NewLockStore()
	defer ls.CloseAll()
	uds := NewUDSClient()

	// Setup
	_, output, err := handlePrepareSkill(nil, nil, PrepareSkillInput{Skill: "braintrust"}, uds, ls)
	require.NoError(t, err)
	assert.True(t, output.GuardActive)
	assert.NotEmpty(t, output.TeamDir)
	assert.DirExists(t, output.TeamDir)
	assert.Contains(t, output.TeamDir, "braintrust")
	assert.Equal(t, []string{"Task", "Bash", "Read"}, output.RouterAllowedTools)

	// Guard file should exist
	guardPath := config.GetGuardFilePath("test-fixture-session")
	assert.FileExists(t, guardPath)

	// Read and verify guard content
	guardData, err := os.ReadFile(guardPath)
	require.NoError(t, err)
	var guard config.ActiveSkill
	require.NoError(t, json.Unmarshal(guardData, &guard))
	assert.Equal(t, 2, guard.FormatVersion)
	assert.Equal(t, "braintrust", guard.Skill)
	assert.Equal(t, "test-fixture-session", guard.SessionID)

	// Release
	_, releaseOutput, err := handlePrepareSkill(nil, nil, PrepareSkillInput{Skill: "braintrust", Release: true}, uds, ls)
	require.NoError(t, err)
	assert.True(t, releaseOutput.Released)

	// Guard file should be gone
	assert.NoFileExists(t, guardPath)
}

func TestHandlePrepareSkill_Release_Idempotent(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	origSessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
		os.Setenv("GOYOKE_SESSION_DIR", origSessionDir)
	}()
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	os.Setenv("GOYOKE_SESSION_DIR", t.TempDir())

	ls := NewLockStore()
	uds := NewUDSClient()

	// Release without setup — should not error
	_, output, err := handlePrepareSkill(nil, nil, PrepareSkillInput{Skill: "braintrust", Release: true}, uds, ls)
	require.NoError(t, err)
	assert.True(t, output.Released)
}

// isGuardStale is defined in cmd/goyoke-skill-guard/daemon.go and is not
// importable here. TestLockStore_Acquire_DetectedByIsGuardStale above
// replicates its core logic (LOCK_SH|LOCK_NB → EWOULDBLOCK) to verify
// cross-process flock recognition without creating a circular import.
var _ = skillsetup.RemoveGuardFiles // ensure pkg/skillsetup is used
