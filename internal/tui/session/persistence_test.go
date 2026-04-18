package session

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewSessionID
// ---------------------------------------------------------------------------

func TestNewSessionID_Format(t *testing.T) {
	id := NewSessionID()

	// Must match YYYYMMDD.{UUID}
	re := regexp.MustCompile(`^\d{8}\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	assert.Regexp(t, re, id, "session ID must match YYYYMMDD.UUID format")
}

func TestNewSessionID_Uniqueness(t *testing.T) {
	const n = 100
	seen := make(map[string]struct{}, n)
	for range n {
		id := NewSessionID()
		_, duplicate := seen[id]
		assert.False(t, duplicate, "duplicate session ID generated: %s", id)
		seen[id] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// SaveSession / LoadSession roundtrip
// ---------------------------------------------------------------------------

func TestSaveLoadSession_Roundtrip(t *testing.T) {
	store := NewStore(t.TempDir())

	original := &SessionData{
		ID:        "20260323.test-roundtrip-id",
		Name:      "test session",
		CreatedAt: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
		LastUsed:  time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
		Cost:      1.23,
		ToolCalls: 42,
		ProviderSessionIDs: map[state.ProviderID]string{
			state.ProviderAnthropic: "sess-abc",
			state.ProviderGoogle:    "sess-xyz",
		},
		ProviderModels: map[state.ProviderID]string{
			state.ProviderAnthropic: "sonnet",
		},
		ActiveProvider: state.ProviderAnthropic,
	}

	require.NoError(t, store.SaveSession(original))

	loaded, err := store.LoadSession(original.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, original.ID, loaded.ID)
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.CreatedAt.UTC(), loaded.CreatedAt.UTC())
	assert.Equal(t, original.Cost, loaded.Cost)
	assert.Equal(t, original.ToolCalls, loaded.ToolCalls)
	assert.Equal(t, original.ProviderSessionIDs, loaded.ProviderSessionIDs)
	assert.Equal(t, original.ProviderModels, loaded.ProviderModels)
	assert.Equal(t, original.ActiveProvider, loaded.ActiveProvider)
}

// ---------------------------------------------------------------------------
// LoadSession — missing file
// ---------------------------------------------------------------------------

func TestLoadSession_MissingFile(t *testing.T) {
	store := NewStore(t.TempDir())

	data, err := store.LoadSession("20260323.nonexistent-id")
	assert.NoError(t, err)
	assert.Nil(t, data)
}

// ---------------------------------------------------------------------------
// LoadSession — invalid JSON
// ---------------------------------------------------------------------------

func TestLoadSession_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const id = "20260323.bad-json-id"
	sessionDir := filepath.Join(dir, id)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "session.json"), []byte("{not valid json}"), 0o644))

	data, err := store.LoadSession(id)
	assert.Error(t, err)
	assert.Nil(t, data)
}

// ---------------------------------------------------------------------------
// SaveSession — atomic write (no .tmp remains)
// ---------------------------------------------------------------------------

func TestSaveSession_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	sd := &SessionData{
		ID:             "20260323.atomic-test",
		ActiveProvider: state.ProviderAnthropic,
	}
	require.NoError(t, store.SaveSession(sd))

	// No leftover .tmp file should exist.
	tmpPath := store.sessionFilePath(sd.ID) + ".tmp"
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "tmp file must not exist after successful save")
}

// ---------------------------------------------------------------------------
// SaveSession — empty ID
// ---------------------------------------------------------------------------

func TestSaveSession_EmptyID(t *testing.T) {
	store := NewStore(t.TempDir())

	err := store.SaveSession(&SessionData{})
	assert.ErrorIs(t, err, ErrEmptySessionID)
}

// ---------------------------------------------------------------------------
// SaveSession — LastUsed updated
// ---------------------------------------------------------------------------

func TestSaveSession_UpdatesLastUsed(t *testing.T) {
	store := NewStore(t.TempDir())

	before := time.Now()

	sd := &SessionData{
		ID:             "20260323.lastsused-test",
		LastUsed:       time.Time{}, // zero value
		ActiveProvider: state.ProviderAnthropic,
	}
	require.NoError(t, store.SaveSession(sd))

	after := time.Now()

	// The in-memory struct must have been updated.
	assert.True(t, sd.LastUsed.After(before) || sd.LastUsed.Equal(before),
		"LastUsed must be >= before-save timestamp")
	assert.True(t, sd.LastUsed.Before(after) || sd.LastUsed.Equal(after),
		"LastUsed must be <= after-save timestamp")

	// The persisted value must match.
	loaded, err := store.LoadSession(sd.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.WithinDuration(t, sd.LastUsed, loaded.LastUsed, time.Second)
}

// ---------------------------------------------------------------------------
// SetupSessionDir
// ---------------------------------------------------------------------------

func TestSetupSessionDir_Creates(t *testing.T) {
	// We need a writable parent to simulate ~/.goyoke, so we create a temp dir
	// structured as: tmpRoot/sessions (= baseDir) with tmpRoot as goyokeDir.
	tmpRoot := t.TempDir()
	baseDir := filepath.Join(tmpRoot, "sessions")
	store := NewStore(baseDir)

	const sessionID = "20260323.setup-test"
	sessionDir, err := store.SetupSessionDir(sessionID)
	require.NoError(t, err)

	// Session directory must exist.
	info, err := os.Stat(sessionDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// current-session marker file must contain the session dir path.
	markerPath := filepath.Join(tmpRoot, "current-session")
	contents, err := os.ReadFile(markerPath)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, string(contents))

	// tmp symlink must point to the session directory.
	tmpLink := filepath.Join(tmpRoot, "tmp")
	target, err := os.Readlink(tmpLink)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, target)
}

func TestSetupSessionDir_OverwriteSymlink(t *testing.T) {
	tmpRoot := t.TempDir()
	baseDir := filepath.Join(tmpRoot, "sessions")
	store := NewStore(baseDir)

	const firstID = "20260323.first-session"
	const secondID = "20260323.second-session"

	firstDir, err := store.SetupSessionDir(firstID)
	require.NoError(t, err)

	secondDir, err := store.SetupSessionDir(secondID)
	require.NoError(t, err)

	// The tmp symlink must now point to the second session, not the first.
	tmpLink := filepath.Join(tmpRoot, "tmp")
	target, err := os.Readlink(tmpLink)
	require.NoError(t, err)
	assert.Equal(t, secondDir, target)
	assert.NotEqual(t, firstDir, target)

	// The current-session marker must also reflect the second session.
	markerPath := filepath.Join(tmpRoot, "current-session")
	contents, err := os.ReadFile(markerPath)
	require.NoError(t, err)
	assert.Equal(t, secondDir, string(contents))
}

// ---------------------------------------------------------------------------
// LoadSession — empty ID
// ---------------------------------------------------------------------------

func TestLoadSession_EmptyID(t *testing.T) {
	store := NewStore(t.TempDir())

	data, err := store.LoadSession("")
	assert.ErrorIs(t, err, ErrEmptySessionID)
	assert.Nil(t, data)
}

// ---------------------------------------------------------------------------
// SetupSessionDir — empty ID
// ---------------------------------------------------------------------------

func TestSetupSessionDir_EmptyID(t *testing.T) {
	store := NewStore(t.TempDir())

	dir, err := store.SetupSessionDir("")
	assert.ErrorIs(t, err, ErrEmptySessionID)
	assert.Empty(t, dir)
}

// ---------------------------------------------------------------------------
// NewStore — empty baseDir falls back to DefaultBaseDir
// ---------------------------------------------------------------------------

func TestNewStore_EmptyBaseDirUsesDefault(t *testing.T) {
	t.Setenv("GOYOKE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	store := NewStore("")
	assert.NotEmpty(t, store.baseDir)
	assert.Contains(t, store.baseDir, ".goyoke")
	assert.Contains(t, store.baseDir, "sessions")
}

// ---------------------------------------------------------------------------
// SessionDir helper
// ---------------------------------------------------------------------------

func TestSessionDir_ReturnsExpectedPath(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	got := store.SessionDir("20260323.some-id")
	assert.Equal(t, filepath.Join(dir, "20260323.some-id"), got)
}

// ---------------------------------------------------------------------------
// DefaultBaseDir
// ---------------------------------------------------------------------------

func TestDefaultBaseDir(t *testing.T) {
	t.Setenv("GOYOKE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dir := DefaultBaseDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, ".goyoke")
	assert.Contains(t, dir, "sessions")
}

// ---------------------------------------------------------------------------
// SaveSession — MkdirAll failure (base dir is a file)
// ---------------------------------------------------------------------------

func TestSaveSession_MkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file at what would be the session directory path.
	// MkdirAll will fail when the parent is a file, not a directory.
	conflictPath := filepath.Join(dir, "20260323.conflict-id")
	require.NoError(t, os.WriteFile(conflictPath, []byte("blocker"), 0o644))

	store := NewStore(dir)
	sd := &SessionData{
		ID:             "20260323.conflict-id",
		ActiveProvider: state.ProviderAnthropic,
	}

	err := store.SaveSession(sd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create dir")
}

// ---------------------------------------------------------------------------
// LoadSession — empty ID in file triggers validation error
// ---------------------------------------------------------------------------

func TestLoadSession_EmptyIDInFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const id = "20260323.empty-id-in-file"
	sessionDir := filepath.Join(dir, id)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	// Write valid JSON but with an empty ID field.
	require.NoError(t, os.WriteFile(
		filepath.Join(sessionDir, "session.json"),
		[]byte(`{"id":"","name":"test"}`),
		0o644,
	))

	data, err := store.LoadSession(id)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "empty ID")
}

// ---------------------------------------------------------------------------
// SetupSessionDir — regular directory at tmp path (not symlink)
// ---------------------------------------------------------------------------

func TestSetupSessionDir_RegularDirAtTmpPath(t *testing.T) {
	tmpRoot := t.TempDir()
	baseDir := filepath.Join(tmpRoot, "sessions")
	store := NewStore(baseDir)

	// Pre-create a regular (empty) directory at the tmp link location.
	// os.Remove will succeed on an empty directory.
	tmpLink := filepath.Join(tmpRoot, "tmp")
	require.NoError(t, os.MkdirAll(tmpLink, 0o755))

	const sessionID = "20260323.dir-at-tmp"
	sessionDir, err := store.SetupSessionDir(sessionID)
	require.NoError(t, err)

	// Verify symlink was created correctly despite pre-existing dir.
	target, err := os.Readlink(tmpLink)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, target)
}

// ---------------------------------------------------------------------------
// SaveSession — all fields populated (comprehensive roundtrip)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// SaveSession — WriteFile failure (read-only session dir)
// ---------------------------------------------------------------------------

func TestSaveSession_WriteFileFailure(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	const id = "20260323.readonly-session"
	sessionDir := filepath.Join(dir, id)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	// Make the session dir read-only so WriteFile fails.
	require.NoError(t, os.Chmod(sessionDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(sessionDir, 0o755) })

	sd := &SessionData{
		ID:             id,
		ActiveProvider: state.ProviderAnthropic,
	}
	err := store.SaveSession(sd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write tmp file")
}

// ---------------------------------------------------------------------------
// SetupSessionDir — write marker failure
// ---------------------------------------------------------------------------

func TestSetupSessionDir_WriteMarkerFailure(t *testing.T) {
	tmpRoot := t.TempDir()
	baseDir := filepath.Join(tmpRoot, "sessions")
	store := NewStore(baseDir)

	// Pre-create baseDir so session dir creation succeeds, but make the
	// parent dir (claudeDir = tmpRoot) read-only so the marker write fails.
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	// Write an existing current-session file, then make the parent read-only.
	markerPath := filepath.Join(tmpRoot, "current-session")
	require.NoError(t, os.WriteFile(markerPath, []byte("old"), 0o644))
	require.NoError(t, os.Chmod(tmpRoot, 0o555))
	t.Cleanup(func() { _ = os.Chmod(tmpRoot, 0o755) })

	_, err := store.SetupSessionDir("20260323.marker-fail")
	assert.Error(t, err)
	// Might fail at marker write or symlink step depending on OS.
}

func TestSaveLoadSession_AllFieldsPopulated(t *testing.T) {
	store := NewStore(t.TempDir())

	original := &SessionData{
		ID:        "20260323.all-fields",
		Name:      "full session",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Cost:      15.9876,
		ToolCalls: 999,
		ProviderSessionIDs: map[state.ProviderID]string{
			state.ProviderAnthropic: "sess-1",
			state.ProviderGoogle:    "sess-2",
			state.ProviderOpenAI:    "sess-3",
			state.ProviderLocal:     "sess-4",
		},
		ProviderModels: map[state.ProviderID]string{
			state.ProviderAnthropic: "opus",
			state.ProviderGoogle:    "gemini-pro",
			state.ProviderOpenAI:    "gpt-4-turbo",
			state.ProviderLocal:     "llama3.1:70b",
		},
		ActiveProvider: state.ProviderGoogle,
	}

	require.NoError(t, store.SaveSession(original))

	loaded, err := store.LoadSession(original.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, original.ID, loaded.ID)
	assert.Equal(t, original.Name, loaded.Name)
	assert.InDelta(t, original.Cost, loaded.Cost, 0.0001)
	assert.Equal(t, original.ToolCalls, loaded.ToolCalls)
	assert.Equal(t, 4, len(loaded.ProviderSessionIDs))
	assert.Equal(t, 4, len(loaded.ProviderModels))
	assert.Equal(t, state.ProviderGoogle, loaded.ActiveProvider)
}

// ---------------------------------------------------------------------------
// ThemeVariant field (TUI-046)
// ---------------------------------------------------------------------------

// TestSessionData_ThemeVariantField verifies that the ThemeVariant int field
// round-trips correctly through JSON save/load.  The field uses int (not
// config.ThemeVariant) to avoid an import cycle between the session and config
// packages; callers in the model package perform the int↔ThemeVariant cast.
func TestSessionData_ThemeVariantField(t *testing.T) {
	store := NewStore(t.TempDir())

	tests := []struct {
		name    string
		variant int
	}{
		{"dark (default, omitempty)", 0},
		{"light", 1},
		{"high_contrast", 2},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sd := &SessionData{
				ID:           "20260323.theme-" + tc.name,
				ThemeVariant: tc.variant,
			}
			require.NoError(t, store.SaveSession(sd))

			loaded, err := store.LoadSession(sd.ID)
			require.NoError(t, err)
			require.NotNil(t, loaded)

			assert.Equal(t, tc.variant, loaded.ThemeVariant,
				"ThemeVariant=%d should round-trip through JSON", tc.variant)
		})
	}
}

// TestSessionData_ThemeVariantOmitEmpty verifies that the ThemeVariant field
// is absent from the serialized JSON when its value is 0 (ThemeDark default),
// keeping session files compact for the common case.
func TestSessionData_ThemeVariantOmitEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	sd := &SessionData{
		ID:           "20260323.omitempty-theme",
		ThemeVariant: 0, // ThemeDark — should be omitted by omitempty
	}
	require.NoError(t, store.SaveSession(sd))

	path := store.sessionFilePath(sd.ID)
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"theme_variant"`,
		"ThemeVariant=0 should be omitted from JSON by omitempty")
}

// TestSessionData_ThemeVariantNonZeroPresent verifies that a non-zero
// ThemeVariant IS written to JSON (omitempty only suppresses the zero value).
func TestSessionData_ThemeVariantNonZeroPresent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	sd := &SessionData{
		ID:           "20260323.theme-present",
		ThemeVariant: 2, // ThemeHighContrast
	}
	require.NoError(t, store.SaveSession(sd))

	path := store.sessionFilePath(sd.ID)
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"theme_variant"`,
		"non-zero ThemeVariant must be written to JSON")
	assert.Contains(t, string(data), `2`,
		"ThemeVariant value 2 must appear in JSON")
}
