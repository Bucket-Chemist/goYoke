package hintbar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LoadOnboarding
// ---------------------------------------------------------------------------

func TestLoadOnboarding_FileNotFound_ReturnsDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	state := LoadOnboarding()
	assert.Equal(t, 0, state.SessionCount, "missing file should return zero session count")
	assert.Nil(t, state.Dismissed, "missing file should return nil dismissed list")
}

// ---------------------------------------------------------------------------
// SaveOnboarding / LoadOnboarding round-trip
// ---------------------------------------------------------------------------

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	original := OnboardingState{
		SessionCount: 2,
		Dismissed:    []string{"tab-agents", "arrows-tabs"},
	}

	err := SaveOnboarding(original)
	require.NoError(t, err)

	loaded := LoadOnboarding()
	assert.Equal(t, original.SessionCount, loaded.SessionCount)
	assert.ElementsMatch(t, original.Dismissed, loaded.Dismissed)
}

func TestSaveAndLoad_ZeroState_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	err := SaveOnboarding(OnboardingState{SessionCount: 0, Dismissed: nil})
	require.NoError(t, err)

	loaded := LoadOnboarding()
	assert.Equal(t, 0, loaded.SessionCount)
}

func TestSaveAndLoad_CreatesDirIfMissing(t *testing.T) {
	tmp := t.TempDir()
	// Point XDG_DATA_HOME at a subdirectory that doesn't exist yet.
	t.Setenv("XDG_DATA_HOME", tmp+"/nonexistent/nested")

	err := SaveOnboarding(OnboardingState{SessionCount: 1})
	require.NoError(t, err, "SaveOnboarding should create missing parent directories")

	loaded := LoadOnboarding()
	assert.Equal(t, 1, loaded.SessionCount)
}

// ---------------------------------------------------------------------------
// IncrementSession
// ---------------------------------------------------------------------------

func TestIncrementSession_IncrementsCount(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	state1 := IncrementSession()
	assert.Equal(t, 1, state1.SessionCount, "first increment should yield session count 1")

	state2 := IncrementSession()
	assert.Equal(t, 2, state2.SessionCount, "second increment should yield session count 2")
}

func TestIncrementSession_PreservesDismissed(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Pre-seed a state with dismissed hints.
	require.NoError(t, SaveOnboarding(OnboardingState{
		SessionCount: 1,
		Dismissed:    []string{"tab-agents"},
	}))

	state := IncrementSession()
	assert.Equal(t, 2, state.SessionCount)
	assert.ElementsMatch(t, []string{"tab-agents"}, state.Dismissed,
		"IncrementSession should preserve dismissed hints")
}
