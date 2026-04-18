package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// sampleMessages returns a small slice of DisplayMessages for use in tests.
func sampleMessages() []state.DisplayMessage {
	return []state.DisplayMessage{
		{
			Role:      "user",
			Content:   "hello",
			Timestamp: time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC),
		},
		{
			Role:      "assistant",
			Content:   "hi there",
			Timestamp: time.Date(2026, 3, 23, 9, 0, 1, 0, time.UTC),
			ToolBlocks: []state.ToolBlock{
				{Name: "Read", Input: "file.go", Output: "package main"},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// SaveConversationHistory / LoadConversationHistory roundtrip
// ---------------------------------------------------------------------------

func TestSaveLoadHistory_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// The session dir must exist before saving history.
	const sessionID = "20260323.history-roundtrip"
	require.NoError(t, os.MkdirAll(store.SessionDir(sessionID), 0o755))

	msgs := sampleMessages()
	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, msgs))

	loaded, err := store.LoadConversationHistory(sessionID, state.ProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, loaded, len(msgs))

	assert.Equal(t, msgs[0].Role, loaded[0].Role)
	assert.Equal(t, msgs[0].Content, loaded[0].Content)
	assert.Equal(t, msgs[1].Role, loaded[1].Role)
	assert.Equal(t, msgs[1].Content, loaded[1].Content)
}

// ---------------------------------------------------------------------------
// LoadConversationHistory — missing file
// ---------------------------------------------------------------------------

func TestLoadHistory_MissingFile(t *testing.T) {
	store := NewStore(t.TempDir())

	msgs, err := store.LoadConversationHistory("20260323.no-file", state.ProviderAnthropic)
	assert.NoError(t, err)
	assert.Nil(t, msgs)
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — empty messages removes file
// ---------------------------------------------------------------------------

func TestSaveHistory_EmptyMessages(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.empty-msgs"
	require.NoError(t, os.MkdirAll(store.SessionDir(sessionID), 0o755))

	// First: save some messages so the file exists.
	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, sampleMessages()))

	histPath := store.historyFilePath(sessionID, state.ProviderAnthropic)
	_, err := os.Stat(histPath)
	require.NoError(t, err, "history file must exist after initial save")

	// Now save with nil — file must be removed.
	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, nil))

	_, err = os.Stat(histPath)
	assert.True(t, os.IsNotExist(err), "history file must be removed when messages is nil")
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — ToolBlocks preserved
// ---------------------------------------------------------------------------

func TestSaveHistory_WithToolBlocks(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.toolblocks"
	require.NoError(t, os.MkdirAll(store.SessionDir(sessionID), 0o755))

	msgs := []state.DisplayMessage{
		{
			Role:    "assistant",
			Content: "done",
			ToolBlocks: []state.ToolBlock{
				{Name: "Bash", Input: "go build ./...", Output: "ok", Expanded: false},
				{Name: "Edit", Input: "main.go:10", Output: "patched", Expanded: true},
			},
		},
	}

	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, msgs))

	loaded, err := store.LoadConversationHistory(sessionID, state.ProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Len(t, loaded[0].ToolBlocks, 2)

	assert.Equal(t, "Bash", loaded[0].ToolBlocks[0].Name)
	assert.Equal(t, "go build ./...", loaded[0].ToolBlocks[0].Input)
	assert.Equal(t, "ok", loaded[0].ToolBlocks[0].Output)
	assert.Equal(t, "Edit", loaded[0].ToolBlocks[1].Name)
	assert.Equal(t, "patched", loaded[0].ToolBlocks[1].Output)
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — multiple providers are independent
// ---------------------------------------------------------------------------

func TestSaveHistory_MultipleProviders(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.multi-provider"
	require.NoError(t, os.MkdirAll(store.SessionDir(sessionID), 0o755))

	anthropicMsgs := []state.DisplayMessage{
		{Role: "user", Content: "anthropic message"},
	}
	googleMsgs := []state.DisplayMessage{
		{Role: "user", Content: "google message"},
		{Role: "assistant", Content: "google reply"},
	}

	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, anthropicMsgs))
	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderGoogle, googleMsgs))

	loadedAnthropic, err := store.LoadConversationHistory(sessionID, state.ProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, loadedAnthropic, 1)
	assert.Equal(t, "anthropic message", loadedAnthropic[0].Content)

	loadedGoogle, err := store.LoadConversationHistory(sessionID, state.ProviderGoogle)
	require.NoError(t, err)
	require.Len(t, loadedGoogle, 2)
	assert.Equal(t, "google message", loadedGoogle[0].Content)
	assert.Equal(t, "google reply", loadedGoogle[1].Content)
}

// ---------------------------------------------------------------------------
// LoadConversationHistory — empty sessionID
// ---------------------------------------------------------------------------

func TestLoadHistory_EmptySessionID(t *testing.T) {
	store := NewStore(t.TempDir())

	msgs, err := store.LoadConversationHistory("", state.ProviderAnthropic)
	assert.ErrorIs(t, err, ErrEmptySessionID)
	assert.Nil(t, msgs)
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — empty sessionID
// ---------------------------------------------------------------------------

func TestSaveHistory_EmptySessionID(t *testing.T) {
	store := NewStore(t.TempDir())

	err := store.SaveConversationHistory("", state.ProviderAnthropic, sampleMessages())
	assert.ErrorIs(t, err, ErrEmptySessionID)
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — nil messages, no pre-existing file (idempotent)
// ---------------------------------------------------------------------------

func TestSaveHistory_NilMessages_NoFile(t *testing.T) {
	store := NewStore(t.TempDir())

	// No file exists; removing it should be a no-op (IsNotExist ignored).
	err := store.SaveConversationHistory("20260323.no-file-nil", state.ProviderAnthropic, nil)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — atomic write (no .tmp remains)
// ---------------------------------------------------------------------------

func TestSaveHistory_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.atomic-history"
	require.NoError(t, os.MkdirAll(store.SessionDir(sessionID), 0o755))

	require.NoError(t, store.SaveConversationHistory(sessionID, state.ProviderAnthropic, sampleMessages()))

	// Verify no .tmp file was left behind.
	histPath := store.historyFilePath(sessionID, state.ProviderAnthropic)
	tmpPath := histPath + ".tmp"

	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "tmp file must not exist after successful save")

	// The final file must exist.
	_, err = os.Stat(histPath)
	assert.NoError(t, err, "history file must exist after save")

	// And it must be in the expected location.
	expectedPath := filepath.Join(dir, sessionID, "history-anthropic.json")
	assert.Equal(t, expectedPath, histPath)
}

// ---------------------------------------------------------------------------
// LoadConversationHistory — invalid JSON
// ---------------------------------------------------------------------------

func TestLoadHistory_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.bad-json-hist"
	sessionDir := filepath.Join(dir, sessionID)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	histPath := store.historyFilePath(sessionID, state.ProviderAnthropic)
	require.NoError(t, os.WriteFile(histPath, []byte("{not valid json array}"), 0o644))

	msgs, err := store.LoadConversationHistory(sessionID, state.ProviderAnthropic)
	assert.Error(t, err)
	assert.Nil(t, msgs)
	assert.Contains(t, err.Error(), "decode JSON")
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — auto-creates session directory
// ---------------------------------------------------------------------------

func TestSaveHistory_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Do NOT pre-create the session dir — SaveConversationHistory should create it.
	const sessionID = "20260323.auto-mkdir"

	err := store.SaveConversationHistory(sessionID, state.ProviderAnthropic, sampleMessages())
	require.NoError(t, err)

	// Verify the session dir was created.
	info, err := os.Stat(store.SessionDir(sessionID))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify the history file was written.
	loaded, err := store.LoadConversationHistory(sessionID, state.ProviderAnthropic)
	require.NoError(t, err)
	assert.Len(t, loaded, len(sampleMessages()))
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — empty messages with empty sessionID (error path)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// SaveConversationHistory — MkdirAll failure (base dir is a file)
// ---------------------------------------------------------------------------

func TestSaveHistory_MkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file at the session dir path to block MkdirAll.
	conflictPath := filepath.Join(dir, "20260323.conflict-hist")
	require.NoError(t, os.WriteFile(conflictPath, []byte("blocker"), 0o644))

	store := NewStore(dir)
	err := store.SaveConversationHistory("20260323.conflict-hist", state.ProviderAnthropic, sampleMessages())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create dir")
}

// ---------------------------------------------------------------------------
// SaveConversationHistory — WriteFile failure (read-only dir)
// ---------------------------------------------------------------------------

func TestSaveHistory_WriteFileFailure(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const sessionID = "20260323.readonly-hist"
	sessionDir := filepath.Join(dir, sessionID)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	// Make dir read-only so WriteFile fails.
	require.NoError(t, os.Chmod(sessionDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(sessionDir, 0o755) })

	err := store.SaveConversationHistory(sessionID, state.ProviderAnthropic, sampleMessages())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write tmp file")
}

func TestSaveHistory_EmptyMessages_EmptySessionID(t *testing.T) {
	store := NewStore(t.TempDir())

	// Empty messages + empty sessionID should still return ErrEmptySessionID
	// (validation check happens before the len(messages)==0 short-circuit).
	err := store.SaveConversationHistory("", state.ProviderAnthropic, nil)
	assert.ErrorIs(t, err, ErrEmptySessionID)
}

// ---------------------------------------------------------------------------
// historyFilePath
// ---------------------------------------------------------------------------

func TestHistoryFilePath_Format(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	tests := []struct {
		name     string
		session  string
		provider state.ProviderID
		wantFile string
	}{
		{"anthropic", "sess-1", state.ProviderAnthropic, "history-anthropic.json"},
		{"google", "sess-1", state.ProviderGoogle, "history-google.json"},
		{"openai", "sess-1", state.ProviderOpenAI, "history-openai.json"},
		{"local", "sess-1", state.ProviderLocal, "history-local.json"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := store.historyFilePath(tc.session, tc.provider)
			assert.Equal(t, filepath.Join(dir, tc.session, tc.wantFile), got)
		})
	}
}
