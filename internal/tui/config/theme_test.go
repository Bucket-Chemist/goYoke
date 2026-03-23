package config

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Color constants
// ---------------------------------------------------------------------------

func TestColors_AllDefined(t *testing.T) {
	tests := []struct {
		name  string
		color lipgloss.AdaptiveColor
		light string
		dark  string
	}{
		{"ColorPrimary", ColorPrimary, "6", "6"},
		{"ColorSecondary", ColorSecondary, "4", "4"},
		{"ColorAccent", ColorAccent, "5", "5"},
		{"ColorSuccess", ColorSuccess, "2", "2"},
		{"ColorWarning", ColorWarning, "3", "3"},
		{"ColorError", ColorError, "1", "1"},
		{"ColorMuted", ColorMuted, "8", "8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.light, tc.color.Light, "light variant")
			assert.Equal(t, tc.dark, tc.color.Dark, "dark variant")
		})
	}
}

// ---------------------------------------------------------------------------
// Icon constants
// ---------------------------------------------------------------------------

func TestIcons_AllDefined(t *testing.T) {
	tests := []struct {
		name string
		icon rune
	}{
		{"IconRunning", IconRunning},
		{"IconComplete", IconComplete},
		{"IconError", IconError},
		{"IconPending", IconPending},
		{"IconCancelled", IconCancelled},
		{"IconPaused", IconPaused},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEqual(t, rune(0), tc.icon, "icon must not be zero rune")
		})
	}
}

func TestIcons_Values(t *testing.T) {
	assert.Equal(t, '>', IconRunning)
	assert.Equal(t, '*', IconComplete)
	assert.Equal(t, '!', IconError)
	assert.Equal(t, '.', IconPending)
	assert.Equal(t, 'x', IconCancelled)
	assert.Equal(t, '~', IconPaused)
}

func TestIcons_AreDistinct(t *testing.T) {
	icons := []rune{
		IconRunning,
		IconComplete,
		IconError,
		IconPending,
		IconCancelled,
		IconPaused,
	}

	seen := make(map[rune]string)
	names := []string{
		"IconRunning",
		"IconComplete",
		"IconError",
		"IconPending",
		"IconCancelled",
		"IconPaused",
	}

	for i, icon := range icons {
		prev, exists := seen[icon]
		assert.False(t, exists, "icon %s duplicates %s", names[i], prev)
		seen[icon] = names[i]
	}
}

// ---------------------------------------------------------------------------
// Border styles — focused vs unfocused
// ---------------------------------------------------------------------------

func TestBorderStyles_FocusedUsesRoundedBorder(t *testing.T) {
	// Render a small box with the focused style and confirm it uses
	// rounded-corner characters (lipgloss RoundedBorder corners are '╭').
	rendered := StyleFocusedBorder.Width(5).Height(1).Render("x")
	assert.True(t,
		strings.Contains(rendered, "╭") || strings.Contains(rendered, "╰"),
		"focused border should use rounded corners, got:\n%s", rendered,
	)
}

func TestBorderStyles_UnfocusedUsesNormalBorder(t *testing.T) {
	// NormalBorder uses '+' corners and '-'/'|' sides (or similar box-drawing).
	rendered := StyleUnfocusedBorder.Width(5).Height(1).Render("x")
	// Normal border in lipgloss uses '+' for corners.
	assert.True(t,
		strings.Contains(rendered, "+") ||
			strings.Contains(rendered, "┌") ||
			strings.Contains(rendered, "─"),
		"unfocused border should use normal/straight corners, got:\n%s", rendered,
	)
}

func TestBorderStyles_FocusedAndUnfocusedDiffer(t *testing.T) {
	focused := StyleFocusedBorder.Width(5).Height(1).Render("x")
	unfocused := StyleUnfocusedBorder.Width(5).Height(1).Render("x")
	assert.NotEqual(t, focused, unfocused,
		"focused and unfocused border styles must render differently")
}

// ---------------------------------------------------------------------------
// DefaultTheme
// ---------------------------------------------------------------------------

func TestDefaultTheme_ReturnsNonZeroStyles(t *testing.T) {
	theme := DefaultTheme()

	// Spot-check that critical theme fields are populated (non-zero style).
	// lipgloss.Style has no IsZero(), so we verify by rendering a known string
	// and checking a non-empty result.
	tests := []struct {
		name    string
		renderF func() string
	}{
		{"FocusedBorder", func() string { return theme.FocusedBorder.Width(4).Height(1).Render("a") }},
		{"UnfocusedBorder", func() string { return theme.UnfocusedBorder.Width(4).Height(1).Render("a") }},
		{"Title", func() string { return theme.Title.Render("title") }},
		{"Error", func() string { return theme.Error.Render("err") }},
		{"Success", func() string { return theme.Success.Render("ok") }},
		{"Warning", func() string { return theme.Warning.Render("warn") }},
		{"Muted", func() string { return theme.Muted.Render("hint") }},
		{"Highlight", func() string { return theme.Highlight.Render("hl") }},
		{"Subtle", func() string { return theme.Subtle.Render("subtle") }},
		{"StatusBar", func() string { return theme.StatusBar.Render("status") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.renderF(), "rendered output must not be empty")
		})
	}
}

func TestDefaultTheme_ColorsMatchPackageConstants(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, ColorPrimary, theme.ColorPrimary)
	assert.Equal(t, ColorSecondary, theme.ColorSecondary)
	assert.Equal(t, ColorAccent, theme.ColorAccent)
	assert.Equal(t, ColorSuccess, theme.ColorSuccess)
	assert.Equal(t, ColorWarning, theme.ColorWarning)
	assert.Equal(t, ColorError, theme.ColorError)
	assert.Equal(t, ColorMuted, theme.ColorMuted)
}

func TestDefaultTheme_BorderStylesDiffer(t *testing.T) {
	theme := DefaultTheme()
	focused := theme.FocusedBorder.Width(5).Height(1).Render("t")
	unfocused := theme.UnfocusedBorder.Width(5).Height(1).Render("t")
	assert.NotEqual(t, focused, unfocused,
		"theme focused and unfocused borders must render differently")
}
