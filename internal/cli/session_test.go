package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_ListSessions(t *testing.T) {
	// Create temp directory for test sessions
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	// Create test sessions with different timestamps
	now := time.Now()
	sessions := []Session{
		{
			ID:        "session-1",
			Name:      "First Session",
			CreatedAt: now.Add(-3 * time.Hour),
			LastUsed:  now.Add(-3 * time.Hour),
			Cost:      1.23,
			ToolCalls: 10,
		},
		{
			ID:        "session-2",
			Name:      "Second Session",
			CreatedAt: now.Add(-2 * time.Hour),
			LastUsed:  now.Add(-1 * time.Hour), // Most recent
			Cost:      2.45,
			ToolCalls: 20,
		},
		{
			ID:        "session-3",
			CreatedAt: now.Add(-4 * time.Hour),
			LastUsed:  now.Add(-2 * time.Hour),
			Cost:      0.50,
			ToolCalls: 5,
		},
	}

	// Write test sessions
	for _, s := range sessions {
		err := sm.saveSession(s)
		require.NoError(t, err)
	}

	// List sessions
	result, err := sm.ListSessions()
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Verify sorted by LastUsed descending
	assert.Equal(t, "session-2", result[0].ID, "Most recent should be first")
	assert.Equal(t, "session-3", result[1].ID)
	assert.Equal(t, "session-1", result[2].ID, "Oldest should be last")
}

func TestSessionManager_ListSessions_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	result, err := sm.ListSessions()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestSessionManager_ListSessions_SkipsCorrupt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	// Create valid session
	validSession := Session{
		ID:        "valid",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
	err = sm.saveSession(validSession)
	require.NoError(t, err)

	// Create corrupt session file
	corruptPath := filepath.Join(tmpDir, "corrupt.json")
	err = os.WriteFile(corruptPath, []byte("{invalid json}"), 0644)
	require.NoError(t, err)

	// List should return only valid session
	result, err := sm.ListSessions()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "valid", result[0].ID)
}

func TestSessionManager_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	// Create and save session
	now := time.Now().Truncate(time.Second) // Truncate for JSON round-trip
	session := Session{
		ID:        "test-session",
		Name:      "Test Session",
		CreatedAt: now,
		LastUsed:  now,
		Cost:      3.45,
		ToolCalls: 15,
	}

	err = sm.saveSession(session)
	require.NoError(t, err)

	// Load session
	loaded, err := sm.loadSession("test-session.json")
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, session.ID, loaded.ID)
	assert.Equal(t, session.Name, loaded.Name)
	assert.Equal(t, session.Cost, loaded.Cost)
	assert.Equal(t, session.ToolCalls, loaded.ToolCalls)
	assert.True(t, session.CreatedAt.Equal(loaded.CreatedAt))
	assert.True(t, session.LastUsed.Equal(loaded.LastUsed))
}

func TestSessionManager_DeleteSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	// Create session
	session := Session{
		ID:        "delete-me",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
	err = sm.saveSession(session)
	require.NoError(t, err)

	// Verify exists
	sessions, err := sm.ListSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	// Delete session
	err = sm.DeleteSession("delete-me")
	require.NoError(t, err)

	// Verify deleted
	sessions, err = sm.ListSessions()
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionManager_UpdateLastUsed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{sessionsDir: tmpDir}

	// Create session with old timestamp
	oldTime := time.Now().Add(-1 * time.Hour)
	session := Session{
		ID:        "update-test",
		CreatedAt: oldTime,
		LastUsed:  oldTime,
	}
	err = sm.saveSession(session)
	require.NoError(t, err)

	// Wait a small amount to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update last used
	err = sm.updateLastUsed("update-test")
	require.NoError(t, err)

	// Load and verify timestamp updated
	loaded, err := sm.loadSession("update-test.json")
	require.NoError(t, err)
	assert.True(t, loaded.LastUsed.After(oldTime), "LastUsed should be updated")
}

func TestNewSessionManager(t *testing.T) {
	// This test creates in actual home directory - be careful
	sm, err := NewSessionManager()
	require.NoError(t, err)
	assert.NotNil(t, sm)

	// Verify directory exists
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".claude", "sessions")
	assert.Equal(t, expectedDir, sm.sessionsDir)

	// Verify directory is accessible
	_, err = os.Stat(sm.sessionsDir)
	assert.NoError(t, err)
}

func TestSession_MarshalJSON(t *testing.T) {
	now := time.Now()
	session := Session{
		ID:        "json-test",
		Name:      "Test Session",
		CreatedAt: now,
		LastUsed:  now,
		Cost:      1.50,
		ToolCalls: 8,
	}

	data, err := json.Marshal(session)
	require.NoError(t, err)

	// Verify can unmarshal
	var decoded Session
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, session.ID, decoded.ID)
	assert.Equal(t, session.Name, decoded.Name)
	assert.Equal(t, session.Cost, decoded.Cost)
	assert.Equal(t, session.ToolCalls, decoded.ToolCalls)
}

func TestSession_EmptyName(t *testing.T) {
	// Verify that empty Name field is omitted in JSON (omitempty tag)
	session := Session{
		ID:        "no-name",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	data, err := json.Marshal(session)
	require.NoError(t, err)

	// Parse as generic map to check field presence
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	// Name should not be in JSON when empty
	_, hasName := m["name"]
	assert.False(t, hasName, "Empty name should be omitted from JSON")
}
