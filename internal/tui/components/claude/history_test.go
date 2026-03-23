package claude_test

import (
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

func TestNewInputHistory_Defaults(t *testing.T) {
	h := claude.NewInputHistory(0)
	require.NotNil(t, h)
	assert.Equal(t, 500, h.MaxSize)
	assert.Nil(t, h.All())
}

func TestNewInputHistory_CustomSize(t *testing.T) {
	h := claude.NewInputHistory(10)
	assert.Equal(t, 10, h.MaxSize)
}

func TestNewInputHistory_NegativeSize_UsesDefault(t *testing.T) {
	h := claude.NewInputHistory(-5)
	assert.Equal(t, 500, h.MaxSize)
}

// ---------------------------------------------------------------------------
// Add
// ---------------------------------------------------------------------------

func TestAdd_AppendsEntry(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("hello")
	assert.Equal(t, []string{"hello"}, h.All())
}

func TestAdd_MultipleEntries(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")
	assert.Equal(t, []string{"first", "second", "third"}, h.All())
}

func TestAdd_ConsecutiveDuplicate_Skipped(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("same")
	h.Add("same")
	assert.Len(t, h.All(), 1, "consecutive duplicate should not be added")
}

func TestAdd_NonConsecutiveDuplicate_Added(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("a")
	h.Add("b")
	h.Add("a") // same as first but NOT consecutive — should be added
	assert.Len(t, h.All(), 3)
}

func TestAdd_EmptyString_Skipped(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("")
	assert.Nil(t, h.All(), "empty string should not be added")
}

func TestAdd_TrimsOldestWhenFull(t *testing.T) {
	h := claude.NewInputHistory(3)
	h.Add("a")
	h.Add("b")
	h.Add("c")
	h.Add("d") // causes trim: "a" should be removed

	entries := h.All()
	require.Len(t, entries, 3)
	assert.Equal(t, []string{"b", "c", "d"}, entries)
}

func TestAdd_TrimKeepsExactMaxSize(t *testing.T) {
	h := claude.NewInputHistory(5)
	for i := range 10 {
		h.Add(string(rune('a' + i)))
	}
	assert.Len(t, h.All(), 5)
}

// ---------------------------------------------------------------------------
// All
// ---------------------------------------------------------------------------

func TestAll_ReturnsIndependentCopy(t *testing.T) {
	h := claude.NewInputHistory(10)
	h.Add("original")

	got := h.All()
	got[0] = "mutated"

	// History should be unaffected.
	assert.Equal(t, []string{"original"}, h.All())
}

func TestAll_EmptyHistory_ReturnsNil(t *testing.T) {
	h := claude.NewInputHistory(10)
	assert.Nil(t, h.All())
}

// ---------------------------------------------------------------------------
// Save / LoadInputHistory
// ---------------------------------------------------------------------------

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := claude.NewInputHistory(100)
	h.Add("first entry")
	h.Add("second entry")
	h.Add("third entry")

	require.NoError(t, h.Save(dir))

	loaded, err := claude.LoadInputHistory(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, h.All(), loaded.All())
	assert.Equal(t, 100, loaded.MaxSize)
}

func TestLoadInputHistory_MissingFile_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	// No file has been written.

	h, err := claude.LoadInputHistory(dir)
	assert.NoError(t, err)
	require.NotNil(t, h)
	assert.Nil(t, h.All(), "missing file should return empty history")
	assert.Equal(t, 500, h.MaxSize)
}

func TestLoadInputHistory_InvalidJSON_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inputhistory.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid json}"), 0o600))

	h, err := claude.LoadInputHistory(dir)
	assert.NoError(t, err, "invalid JSON should not propagate as an error")
	require.NotNil(t, h)
	assert.Nil(t, h.All(), "invalid JSON should return empty history")
}

func TestSave_CreatesDirectoryIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	h := claude.NewInputHistory(10)
	h.Add("entry")

	require.NoError(t, h.Save(dir))

	_, err := os.Stat(dir)
	assert.NoError(t, err, "Save should create missing directories")
}

func TestSave_AtomicWrite_TargetExists(t *testing.T) {
	dir := t.TempDir()
	// Write initial history.
	h1 := claude.NewInputHistory(10)
	h1.Add("original")
	require.NoError(t, h1.Save(dir))

	// Overwrite with new history.
	h2 := claude.NewInputHistory(10)
	h2.Add("updated")
	require.NoError(t, h2.Save(dir))

	loaded, err := claude.LoadInputHistory(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"updated"}, loaded.All())
}

func TestSaveLoad_EmptyHistory(t *testing.T) {
	dir := t.TempDir()
	h := claude.NewInputHistory(50)
	// Add nothing — save an empty history.
	require.NoError(t, h.Save(dir))

	loaded, err := claude.LoadInputHistory(dir)
	require.NoError(t, err)
	assert.Nil(t, loaded.All())
	assert.Equal(t, 50, loaded.MaxSize)
}

func TestLoadInputHistory_ZeroMaxSize_UsesDefault(t *testing.T) {
	dir := t.TempDir()
	// Write a file with max_size=0 (e.g. produced by older version).
	path := filepath.Join(dir, "inputhistory.json")
	raw := []byte(`{"entries":["a","b"],"max_size":0}`)
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	h, err := claude.LoadInputHistory(dir)
	require.NoError(t, err)
	assert.Equal(t, 500, h.MaxSize, "zero max_size should be replaced with default")
	assert.Equal(t, []string{"a", "b"}, h.All())
}
