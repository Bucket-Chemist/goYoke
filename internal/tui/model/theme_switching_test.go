// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains tests for the theme switching infrastructure added in
// TUI-046: ThemeChangedMsg, SetTheme/Theme methods, and sharedState fields.
package model

import (
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestTheme_DefaultIsNonNil
// ---------------------------------------------------------------------------

// TestTheme_DefaultIsNonNil verifies that a freshly constructed AppModel
// always returns a valid (non-zero) Theme from its Theme() method.
func TestTheme_DefaultIsNonNil(t *testing.T) {
	m := NewAppModel()

	theme := m.Theme()

	// A valid Theme has non-nil color fields (lipgloss.AdaptiveColor is a
	// struct, so we check that at least one derived style renders non-empty).
	assert.NotEmpty(t,
		theme.FocusedBorder.String(),
		"default theme FocusedBorder should produce non-empty style",
	)
}

// TestTheme_DefaultMatchesDefaultTheme verifies that the theme returned by a
// newly constructed AppModel is structurally equal to config.DefaultTheme().
func TestTheme_DefaultMatchesDefaultTheme(t *testing.T) {
	m := NewAppModel()

	got := m.Theme()
	want := config.DefaultTheme()

	assert.Equal(t, want.ColorPrimary, got.ColorPrimary)
	assert.Equal(t, want.ColorSecondary, got.ColorSecondary)
	assert.Equal(t, want.ColorAccent, got.ColorAccent)
	assert.Equal(t, want.ColorError, got.ColorError)
	assert.Equal(t, want.ColorMuted, got.ColorMuted)
}

// ---------------------------------------------------------------------------
// TestSetTheme_PersistsThroughSharedState
// ---------------------------------------------------------------------------

// TestSetTheme_PersistsThroughSharedState verifies that SetTheme stores the
// new theme in sharedState and that Theme() returns it on the next call.
func TestSetTheme_PersistsThroughSharedState(t *testing.T) {
	m := NewAppModel()

	highContrast := config.NewTheme(config.ThemeHighContrast)
	m.SetTheme(highContrast)

	got := m.Theme()

	assert.Equal(t, highContrast.ColorPrimary, got.ColorPrimary,
		"Theme() should return the theme set via SetTheme()")
	assert.Equal(t, highContrast.ColorError, got.ColorError)
}

// TestSetTheme_OverwritesPreviousTheme verifies that successive SetTheme calls
// each overwrite the previous value.
func TestSetTheme_OverwritesPreviousTheme(t *testing.T) {
	m := NewAppModel()

	m.SetTheme(config.NewTheme(config.ThemeLight))
	m.SetTheme(config.NewTheme(config.ThemeHighContrast))

	got := m.Theme()
	want := config.NewTheme(config.ThemeHighContrast)

	assert.Equal(t, want.ColorPrimary, got.ColorPrimary,
		"second SetTheme should overwrite first")
}

// ---------------------------------------------------------------------------
// TestThemeChangedMsg_UpdatesActiveTheme
// ---------------------------------------------------------------------------

// TestThemeChangedMsg_UpdatesActiveTheme verifies that sending ThemeChangedMsg
// through Update() causes Theme() to return the newly activated theme.
func TestThemeChangedMsg_UpdatesActiveTheme(t *testing.T) {
	m := NewAppModel()

	updated, cmd := m.Update(ThemeChangedMsg{Variant: config.ThemeLight})
	require.Nil(t, cmd, "ThemeChangedMsg should produce no Cmd")

	appModel, ok := updated.(AppModel)
	require.True(t, ok, "Update must return AppModel")

	got := appModel.Theme()
	want := config.NewTheme(config.ThemeLight)

	assert.Equal(t, want.ColorMuted, got.ColorMuted,
		"active theme colors should reflect ThemeLight after ThemeChangedMsg")
}

// ---------------------------------------------------------------------------
// TestThemeChangedMsg_AllVariants (table-driven)
// ---------------------------------------------------------------------------

// TestThemeChangedMsg_AllVariants is a table-driven test covering all three
// defined ThemeVariant values. For each variant it sends ThemeChangedMsg and
// verifies that Theme() returns a theme built from config.NewTheme(variant).
func TestThemeChangedMsg_AllVariants(t *testing.T) {
	tests := []struct {
		name    string
		variant config.ThemeVariant
	}{
		{"dark", config.ThemeDark},
		{"light", config.ThemeLight},
		{"high_contrast", config.ThemeHighContrast},
	}

	for _, tc := range tests {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			updated, cmd := m.Update(ThemeChangedMsg{Variant: tc.variant})
			require.Nil(t, cmd)

			appModel, ok := updated.(AppModel)
			require.True(t, ok)

			got := appModel.Theme()
			want := config.NewTheme(tc.variant)

			assert.Equal(t, want.ColorPrimary, got.ColorPrimary,
				"variant=%v: ColorPrimary mismatch", tc.variant)
			assert.Equal(t, want.ColorError, got.ColorError,
				"variant=%v: ColorError mismatch", tc.variant)
			assert.Equal(t, want.ColorMuted, got.ColorMuted,
				"variant=%v: ColorMuted mismatch", tc.variant)
		})
	}
}

// ---------------------------------------------------------------------------
// TestThemeChangedMsg_PersistsVariantInSharedState
// ---------------------------------------------------------------------------

// TestThemeChangedMsg_PersistsVariantInSharedState verifies that handling
// ThemeChangedMsg also updates sharedState.themeVariant so that saveSession
// can write the correct value to SessionData.ThemeVariant.
func TestThemeChangedMsg_PersistsVariantInSharedState(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(ThemeChangedMsg{Variant: config.ThemeHighContrast})
	appModel := updated.(AppModel)

	// Access the private sharedState field directly (white-box test).
	require.NotNil(t, appModel.shared)
	assert.Equal(t, config.ThemeHighContrast, appModel.shared.themeVariant,
		"themeVariant in sharedState should track the last ThemeChangedMsg variant")
}

// ---------------------------------------------------------------------------
// TestTheme_NilSharedStateReturnsDefault
// ---------------------------------------------------------------------------

// TestTheme_NilSharedStateReturnsDefault verifies that Theme() returns a safe
// default even when sharedState is nil (defensive guard, not a normal path).
func TestTheme_NilSharedStateReturnsDefault(t *testing.T) {
	m := AppModel{} // zero value, shared is nil

	got := m.Theme()
	want := config.DefaultTheme()

	assert.Equal(t, want.ColorPrimary, got.ColorPrimary,
		"nil sharedState should fall back to DefaultTheme()")
}

// ---------------------------------------------------------------------------
// TestThemeChangedMsg_NilSharedStateIsNoop
// ---------------------------------------------------------------------------

// TestThemeChangedMsg_NilSharedStateIsNoop verifies that handleThemeChanged
// is safe when sharedState is nil (defensive guard).
func TestThemeChangedMsg_NilSharedStateIsNoop(t *testing.T) {
	m := AppModel{} // zero value, shared is nil

	// Should not panic.
	updated, cmd := m.Update(ThemeChangedMsg{Variant: config.ThemeLight})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated)
}
