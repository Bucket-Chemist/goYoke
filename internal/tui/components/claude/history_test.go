package claude_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/claude"
)

// ---------------------------------------------------------------------------
// NewInputHistory
// ---------------------------------------------------------------------------

func TestNewInputHistory_Empty(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	require.NotNil(t, h)
	assert.Nil(t, h.All())
	assert.Equal(t, 0, h.Len())
}

// ---------------------------------------------------------------------------
// Add — newest-first, any-position dedup
// ---------------------------------------------------------------------------

func TestAdd_PrependsEntry(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("first")
	h.Add("second")
	// newest first
	assert.Equal(t, []string{"second", "first"}, h.All())
}

func TestAdd_AnyPositionDedup(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("a")
	h.Add("b")
	h.Add("a") // moves "a" to front
	assert.Equal(t, []string{"a", "b"}, h.All())
}

func TestAdd_ConsecutiveDedup(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("same")
	h.Add("same")
	assert.Equal(t, 1, h.Len())
}

func TestAdd_EmptyString_Skipped(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("")
	assert.Nil(t, h.All())
}

func TestAdd_TrimsOldestWhenFull(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	// Add 101 entries — exceeds maxHistorySize (100)
	for i := range 101 {
		h.Add(string(rune('a' + (i % 26))))
	}
	assert.LessOrEqual(t, h.Len(), 100)
}

// ---------------------------------------------------------------------------
// Get / Len
// ---------------------------------------------------------------------------

func TestGet_ReturnsNewestFirst(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("old")
	h.Add("new")
	assert.Equal(t, "new", h.Get(0))
	assert.Equal(t, "old", h.Get(1))
}

func TestGet_OutOfRange_ReturnsEmpty(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	assert.Equal(t, "", h.Get(0))
	assert.Equal(t, "", h.Get(-1))
}

// ---------------------------------------------------------------------------
// All — independent copy
// ---------------------------------------------------------------------------

func TestAll_ReturnsIndependentCopy(t *testing.T) {
	h := claude.NewInputHistory(t.TempDir())
	h.Add("original")

	got := h.All()
	got[0] = "mutated"
	assert.Equal(t, []string{"original"}, h.All())
}

// ---------------------------------------------------------------------------
// Save / LoadInputHistory — TS-compatible format
// ---------------------------------------------------------------------------

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := claude.NewInputHistory(dir)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	require.NoError(t, h.Save())

	loaded := claude.LoadInputHistory(dir)
	require.NotNil(t, loaded)
	assert.Equal(t, h.All(), loaded.All())
}

func TestSave_WritesPlainJSONArray(t *testing.T) {
	dir := t.TempDir()
	h := claude.NewInputHistory(dir)
	h.Add("old")
	h.Add("new")
	require.NoError(t, h.Save())

	data, err := os.ReadFile(filepath.Join(dir, "input-history.json"))
	require.NoError(t, err)

	var arr []string
	require.NoError(t, json.Unmarshal(data, &arr))
	assert.Equal(t, []string{"new", "old"}, arr, "file should be plain JSON array, newest first")
}

func TestLoadInputHistory_MissingFile_ReturnsEmpty(t *testing.T) {
	h := claude.LoadInputHistory(t.TempDir())
	require.NotNil(t, h)
	assert.Nil(t, h.All())
}

func TestLoadInputHistory_TSFormat(t *testing.T) {
	dir := t.TempDir()
	// Write TS TUI format: plain JSON array, newest first.
	data := []byte(`["newest","middle","oldest"]`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input-history.json"), data, 0o600))

	h := claude.LoadInputHistory(dir)
	assert.Equal(t, []string{"newest", "middle", "oldest"}, h.All())
}

func TestLoadInputHistory_LegacyGoFormat(t *testing.T) {
	dir := t.TempDir()
	// Write legacy Go format: object with oldest-first entries.
	data := []byte(`{"entries":["oldest","middle","newest"],"max_size":500}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inputhistory.json"), data, 0o600))

	h := claude.LoadInputHistory(dir)
	// Should be reversed to newest-first.
	assert.Equal(t, []string{"newest", "middle", "oldest"}, h.All())
}

func TestLoadInputHistory_InvalidJSON_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "input-history.json"),
		[]byte("{not valid json}"), 0o600))

	h := claude.LoadInputHistory(dir)
	require.NotNil(t, h)
	assert.Nil(t, h.All())
}

func TestSave_CreatesDirectoryIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested")
	h := claude.NewInputHistory(dir)
	h.Add("entry")
	require.NoError(t, h.Save())

	_, err := os.Stat(dir)
	assert.NoError(t, err)
}

func TestSave_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	h1 := claude.NewInputHistory(dir)
	h1.Add("original")
	require.NoError(t, h1.Save())

	h2 := claude.NewInputHistory(dir)
	h2.Add("updated")
	require.NoError(t, h2.Save())

	loaded := claude.LoadInputHistory(dir)
	assert.Equal(t, []string{"updated"}, loaded.All())
}
