package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// ---------------------------------------------------------------------------
// isGuardStale tests
// ---------------------------------------------------------------------------

func TestIsGuardStale_NoLockFile(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "nonexistent.lock")
	assert.True(t, isGuardStale(lockPath), "missing lock file should be stale")
}

func TestIsGuardStale_UnlockedFile(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	require.NoError(t, os.WriteFile(lockPath, nil, 0644))

	assert.True(t, isGuardStale(lockPath), "lock file with no LOCK_EX should be stale")
}

func TestIsGuardStale_LockedFile(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	// Open and acquire exclusive lock (simulating lock-holder daemon).
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)
	defer fd.Close()

	require.NoError(t, syscall.Flock(int(fd.Fd()), syscall.LOCK_EX))
	defer syscall.Flock(int(fd.Fd()), syscall.LOCK_UN) //nolint:errcheck

	assert.False(t, isGuardStale(lockPath), "lock file with LOCK_EX held should NOT be stale")
}

func TestIsGuardStale_ReleasedLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	// Acquire then release.
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)
	require.NoError(t, syscall.Flock(int(fd.Fd()), syscall.LOCK_EX))
	require.NoError(t, syscall.Flock(int(fd.Fd()), syscall.LOCK_UN))
	fd.Close()

	assert.True(t, isGuardStale(lockPath), "released lock should be stale")
}

// ---------------------------------------------------------------------------
// handleGuardMode tests (session-scoped path)
// ---------------------------------------------------------------------------

func makeEvent(sessionID, toolName string) *routing.ToolEvent {
	return &routing.ToolEvent{
		ToolName:  toolName,
		SessionID: sessionID,
	}
}

func writeV2Guard(t *testing.T, sessionID string, guard *config.ActiveSkill) {
	t.Helper()
	path := config.GetGuardFilePath(sessionID)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	data, err := json.MarshalIndent(guard, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}

// holdLock acquires LOCK_EX on the guard lock file and returns a cleanup function.
func holdLock(t *testing.T, sessionID string) func() {
	t.Helper()
	lockPath := config.GetGuardLockPath(sessionID)
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0755))

	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)
	require.NoError(t, syscall.Flock(int(fd.Fd()), syscall.LOCK_EX))

	return func() {
		syscall.Flock(int(fd.Fd()), syscall.LOCK_UN) //nolint:errcheck
		fd.Close()
	}
}

func TestHandleGuardMode_EmptySessionID(t *testing.T) {
	event := makeEvent("", "Write")
	// Empty session ID with no legacy guard → allow.
	t.Setenv("GOGENT_SESSION_DIR", t.TempDir())
	output := handleGuardMode(event)
	assert.Equal(t, "{}", output)
}

func TestHandleGuardMode_NoGuardFile(t *testing.T) {
	sessionID := "test-session-no-guard"
	event := makeEvent(sessionID, "Write")
	// No legacy guard either.
	t.Setenv("GOGENT_SESSION_DIR", t.TempDir())
	output := handleGuardMode(event)
	assert.Equal(t, "{}", output)
}

func TestHandleGuardMode_ActiveGuard_AllowedTool(t *testing.T) {
	sessionID := "test-session-allowed-" + t.Name()
	unlock := holdLock(t, sessionID)
	defer unlock()

	writeV2Guard(t, sessionID, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "review",
		RouterAllowedTools: []string{"Task", "Bash", "Read"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
	})
	defer os.Remove(config.GetGuardFilePath(sessionID))
	defer os.Remove(config.GetGuardLockPath(sessionID))

	output := handleGuardMode(makeEvent(sessionID, "Read"))
	assert.Equal(t, "{}", output)
}

func TestHandleGuardMode_ActiveGuard_BlockedTool(t *testing.T) {
	sessionID := "test-session-blocked-" + t.Name()
	unlock := holdLock(t, sessionID)
	defer unlock()

	writeV2Guard(t, sessionID, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "review",
		RouterAllowedTools: []string{"Task", "Bash", "Read"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
	})
	defer os.Remove(config.GetGuardFilePath(sessionID))
	defer os.Remove(config.GetGuardLockPath(sessionID))

	output := handleGuardMode(makeEvent(sessionID, "Write"))
	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "review")
}

func TestHandleGuardMode_StaleGuard_AutoCleanup(t *testing.T) {
	sessionID := "test-session-stale-" + t.Name()

	// Write guard but do NOT hold the lock → stale.
	writeV2Guard(t, sessionID, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "review",
		RouterAllowedTools: []string{"Task"},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
	})
	// Create the lock file but don't hold LOCK_EX.
	lockPath := config.GetGuardLockPath(sessionID)
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0755))
	require.NoError(t, os.WriteFile(lockPath, nil, 0644))

	// No legacy guard either.
	t.Setenv("GOGENT_SESSION_DIR", t.TempDir())

	output := handleGuardMode(makeEvent(sessionID, "Write"))
	assert.Equal(t, "{}", output, "stale guard should allow all tools")

	// Guard file should be cleaned up.
	_, err := os.Stat(config.GetGuardFilePath(sessionID))
	assert.True(t, os.IsNotExist(err), "stale guard file should be deleted")
}

func TestHandleGuardMode_MalformedJSON(t *testing.T) {
	sessionID := "test-session-malformed-" + t.Name()
	guardPath := config.GetGuardFilePath(sessionID)
	require.NoError(t, os.MkdirAll(filepath.Dir(guardPath), 0755))
	require.NoError(t, os.WriteFile(guardPath, []byte("not json"), 0644))
	defer os.Remove(guardPath)

	output := handleGuardMode(makeEvent(sessionID, "Write"))
	assert.Equal(t, "{}", output)

	// Malformed file should be deleted.
	_, err := os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err))
}

// ---------------------------------------------------------------------------
// checkAllowList tests
// ---------------------------------------------------------------------------

func TestCheckAllowList_AllAllowed(t *testing.T) {
	guard := &config.ActiveSkill{
		Skill:              "review",
		RouterAllowedTools: []string{"Task", "Bash", "Read", "Glob", "Grep", "ToolSearch"},
	}
	for _, tool := range guard.RouterAllowedTools {
		assert.Equal(t, "{}", checkAllowList(tool, guard), "tool %s should be allowed", tool)
	}
}

func TestCheckAllowList_Blocked(t *testing.T) {
	guard := &config.ActiveSkill{
		Skill:              "review",
		RouterAllowedTools: []string{"Task", "Bash"},
	}
	output := checkAllowList("Write", guard)
	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "review")
	assert.Contains(t, output, "Write")
}

func TestCheckAllowList_MCP_Tools(t *testing.T) {
	guard := &config.ActiveSkill{
		Skill: "review",
		RouterAllowedTools: []string{
			"mcp__gofortress-interactive__spawn_agent",
			"mcp__gofortress-interactive__team_run",
		},
	}
	assert.Equal(t, "{}", checkAllowList("mcp__gofortress-interactive__spawn_agent", guard))
	assert.Equal(t, "{}", checkAllowList("mcp__gofortress-interactive__team_run", guard))
	assert.Contains(t, checkAllowList("mcp__gofortress-interactive__ask_user", guard), "blocked")
}

// ---------------------------------------------------------------------------
// Concurrent session isolation
// ---------------------------------------------------------------------------

func TestHandleGuardMode_ConcurrentSessions_Isolated(t *testing.T) {
	sessionA := "test-concurrent-A-" + t.Name()
	sessionB := "test-concurrent-B-" + t.Name()

	unlockA := holdLock(t, sessionA)
	defer unlockA()
	unlockB := holdLock(t, sessionB)
	defer unlockB()

	writeV2Guard(t, sessionA, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "review",
		RouterAllowedTools: []string{"Task", "Read"},
		SessionID:          sessionA,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	})
	defer os.Remove(config.GetGuardFilePath(sessionA))
	defer os.Remove(config.GetGuardLockPath(sessionA))

	writeV2Guard(t, sessionB, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "implement",
		RouterAllowedTools: []string{"Task", "Bash"},
		SessionID:          sessionB,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	})
	defer os.Remove(config.GetGuardFilePath(sessionB))
	defer os.Remove(config.GetGuardLockPath(sessionB))

	// Session A: Read allowed, Bash blocked.
	assert.Equal(t, "{}", handleGuardMode(makeEvent(sessionA, "Read")))
	assert.Contains(t, handleGuardMode(makeEvent(sessionA, "Bash")), "blocked")

	// Session B: Bash allowed, Read blocked.
	assert.Equal(t, "{}", handleGuardMode(makeEvent(sessionB, "Bash")))
	assert.Contains(t, handleGuardMode(makeEvent(sessionB, "Read")), "blocked")
}

func TestHandleGuardMode_SessionA_Stale_SessionB_Active(t *testing.T) {
	sessionA := "test-stale-A-" + t.Name()
	sessionB := "test-active-B-" + t.Name()

	// Session A: guard exists but no lock held (stale).
	writeV2Guard(t, sessionA, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "review",
		RouterAllowedTools: []string{"Task"},
		SessionID:          sessionA,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	})
	lockPathA := config.GetGuardLockPath(sessionA)
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPathA), 0755))
	require.NoError(t, os.WriteFile(lockPathA, nil, 0644))
	defer os.Remove(config.GetGuardFilePath(sessionA))
	defer os.Remove(lockPathA)

	// Session B: guard with active lock.
	unlockB := holdLock(t, sessionB)
	defer unlockB()
	writeV2Guard(t, sessionB, &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "implement",
		RouterAllowedTools: []string{"Task", "Bash"},
		SessionID:          sessionB,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	})
	defer os.Remove(config.GetGuardFilePath(sessionB))
	defer os.Remove(config.GetGuardLockPath(sessionB))

	// No legacy guard.
	t.Setenv("GOGENT_SESSION_DIR", t.TempDir())

	// Session A: stale → allow everything.
	assert.Equal(t, "{}", handleGuardMode(makeEvent(sessionA, "Write")))

	// Session B: still active → enforce.
	assert.Equal(t, "{}", handleGuardMode(makeEvent(sessionB, "Bash")))
	assert.Contains(t, handleGuardMode(makeEvent(sessionB, "Write")), "blocked")
}
