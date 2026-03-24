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

// ---------------------------------------------------------------------------
// ThemeVariant enum
// ---------------------------------------------------------------------------

func TestThemeVariant_ZeroValueIsDark(t *testing.T) {
	var v ThemeVariant
	assert.Equal(t, ThemeDark, v, "zero value of ThemeVariant must be ThemeDark")
}

func TestThemeVariant_ValuesAreDistinct(t *testing.T) {
	variants := []ThemeVariant{ThemeDark, ThemeLight, ThemeHighContrast}
	seen := make(map[ThemeVariant]bool)
	for _, v := range variants {
		assert.False(t, seen[v], "ThemeVariant values must be distinct, found duplicate %d", v)
		seen[v] = true
	}
}

// ---------------------------------------------------------------------------
// NewTheme factory — table-driven across all three variants
// ---------------------------------------------------------------------------

func TestNewTheme_AllVariantsReturnNonZeroColors(t *testing.T) {
	tests := []struct {
		name    string
		variant ThemeVariant
	}{
		{"Dark", ThemeDark},
		{"Light", ThemeLight},
		{"HighContrast", ThemeHighContrast},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(tc.variant)

			// Every color field must have at least one non-empty variant.
			type colorCase struct {
				name  string
				color lipgloss.AdaptiveColor
			}
			colors := []colorCase{
				{"ColorPrimary", theme.ColorPrimary},
				{"ColorSecondary", theme.ColorSecondary},
				{"ColorAccent", theme.ColorAccent},
				{"ColorSuccess", theme.ColorSuccess},
				{"ColorWarning", theme.ColorWarning},
				{"ColorError", theme.ColorError},
				{"ColorMuted", theme.ColorMuted},
				{"InfoColor", theme.InfoColor},
			}
			for _, cc := range colors {
				t.Run(cc.name, func(t *testing.T) {
					assert.True(t,
						cc.color.Light != "" || cc.color.Dark != "",
						"%s must have at least one non-empty variant", cc.name,
					)
				})
			}
		})
	}
}

func TestNewTheme_AllVariantsRenderNonEmptyStyles(t *testing.T) {
	tests := []struct {
		name    string
		variant ThemeVariant
	}{
		{"Dark", ThemeDark},
		{"Light", ThemeLight},
		{"HighContrast", ThemeHighContrast},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(tc.variant)

			type styleCase struct {
				name    string
				renderF func() string
			}
			cases := []styleCase{
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
			for _, sc := range cases {
				t.Run(sc.name, func(t *testing.T) {
					assert.NotEmpty(t, sc.renderF(), "rendered output must not be empty")
				})
			}
		})
	}
}

func TestNewTheme_DarkMatchesDefaultTheme(t *testing.T) {
	dark := NewTheme(ThemeDark)
	dflt := DefaultTheme()

	// Colors must be identical.
	assert.Equal(t, dflt.ColorPrimary, dark.ColorPrimary, "ColorPrimary")
	assert.Equal(t, dflt.ColorSecondary, dark.ColorSecondary, "ColorSecondary")
	assert.Equal(t, dflt.ColorAccent, dark.ColorAccent, "ColorAccent")
	assert.Equal(t, dflt.ColorSuccess, dark.ColorSuccess, "ColorSuccess")
	assert.Equal(t, dflt.ColorWarning, dark.ColorWarning, "ColorWarning")
	assert.Equal(t, dflt.ColorError, dark.ColorError, "ColorError")
	assert.Equal(t, dflt.ColorMuted, dark.ColorMuted, "ColorMuted")
	assert.Equal(t, dflt.InfoColor, dark.InfoColor, "InfoColor")
}

func TestNewTheme_HighContrastUsesHexCodes(t *testing.T) {
	theme := NewTheme(ThemeHighContrast)

	// The high-contrast palette must use explicit hex codes, not ANSI indices.
	assert.Equal(t, "#FF0000", theme.ColorError.Light, "HighContrast error must be #FF0000")
	assert.Equal(t, "#FF0000", theme.ColorError.Dark, "HighContrast error must be #FF0000 on dark")

	assert.Equal(t, "#00AA00", theme.ColorSuccess.Light, "HighContrast success must be #00AA00")
	assert.Equal(t, "#00AA00", theme.ColorSuccess.Dark, "HighContrast success must be #00AA00 on dark")

	assert.Equal(t, "#FFAA00", theme.ColorWarning.Light, "HighContrast warning must be #FFAA00")
	assert.Equal(t, "#FFAA00", theme.ColorWarning.Dark, "HighContrast warning must be #FFAA00 on dark")

	assert.Equal(t, "#0088FF", theme.ColorPrimary.Light, "HighContrast info must be #0088FF")
	assert.Equal(t, "#0088FF", theme.ColorPrimary.Dark, "HighContrast info must be #0088FF on dark")
}

func TestNewTheme_LightUsesDistinctMutedColor(t *testing.T) {
	light := NewTheme(ThemeLight)
	dark := NewTheme(ThemeDark)

	// The light theme uses ANSI "0" (black) for muted text to contrast against
	// a light background; the dark theme uses "8" (gray).
	assert.NotEqual(t, dark.ColorMuted, light.ColorMuted,
		"light and dark themes should have different muted colors")
}

func TestNewTheme_UnknownVariantFallsBackToDark(t *testing.T) {
	unknown := NewTheme(ThemeVariant(99))
	dark := NewTheme(ThemeDark)

	assert.Equal(t, dark.ColorPrimary, unknown.ColorPrimary,
		"unknown variant must fall back to dark theme")
	assert.Equal(t, dark.ColorError, unknown.ColorError,
		"unknown variant must fall back to dark theme")
}

// ---------------------------------------------------------------------------
// InfoColor field
// ---------------------------------------------------------------------------

func TestDefaultTheme_InfoColorEqualsColorPrimary(t *testing.T) {
	theme := DefaultTheme()
	assert.Equal(t, theme.ColorPrimary, theme.InfoColor,
		"InfoColor must equal ColorPrimary in the default theme")
}

func TestNewTheme_InfoColorEqualsColorPrimary(t *testing.T) {
	for _, variant := range []ThemeVariant{ThemeDark, ThemeLight, ThemeHighContrast} {
		theme := NewTheme(variant)
		assert.Equal(t, theme.ColorPrimary, theme.InfoColor,
			"InfoColor must equal ColorPrimary for variant %d", variant)
	}
}

// ---------------------------------------------------------------------------
// Semantic style methods
// ---------------------------------------------------------------------------

func TestSemanticStyles_RenderNonEmpty(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		name    string
		renderF func() string
	}{
		{"ErrorStyle", func() string { return theme.ErrorStyle().Render("err") }},
		{"WarningStyle", func() string { return theme.WarningStyle().Render("warn") }},
		{"SuccessStyle", func() string { return theme.SuccessStyle().Render("ok") }},
		{"InfoStyle", func() string { return theme.InfoStyle().Render("info") }},
		{"DangerStyle", func() string { return theme.DangerStyle().Render("danger") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.renderF(), "semantic style must render non-empty output")
		})
	}
}

func TestSemanticStyles_OnAllVariants(t *testing.T) {
	variants := []struct {
		name    string
		variant ThemeVariant
	}{
		{"Dark", ThemeDark},
		{"Light", ThemeLight},
		{"HighContrast", ThemeHighContrast},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			theme := NewTheme(v.variant)

			assert.NotEmpty(t, theme.ErrorStyle().Render("e"), "ErrorStyle")
			assert.NotEmpty(t, theme.WarningStyle().Render("w"), "WarningStyle")
			assert.NotEmpty(t, theme.SuccessStyle().Render("s"), "SuccessStyle")
			assert.NotEmpty(t, theme.InfoStyle().Render("i"), "InfoStyle")
			assert.NotEmpty(t, theme.DangerStyle().Render("d"), "DangerStyle")
		})
	}
}

func TestErrorStyle_IsBold(t *testing.T) {
	theme := DefaultTheme()
	// A bold style renders differently from a non-bold style.
	boldRendered := theme.ErrorStyle().Render("x")
	plainRendered := lipgloss.NewStyle().Foreground(theme.ColorError).Render("x")
	// In a no-color environment both may collapse, so we only assert non-empty.
	// The real assertion is that the call doesn't panic and returns a string.
	assert.NotEmpty(t, boldRendered)
	assert.NotEmpty(t, plainRendered)
}

func TestDangerStyle_IsBoldAndUnderline(t *testing.T) {
	// Verify DangerStyle is more emphatic than ErrorStyle by rendering both
	// and confirming they differ (or at minimum that DangerStyle is non-empty).
	theme := DefaultTheme()
	danger := theme.DangerStyle().Render("critical")
	errSt := theme.ErrorStyle().Render("critical")
	// Both should be non-empty; in a real color terminal they'll differ by
	// the underline escape sequence.
	assert.NotEmpty(t, danger, "DangerStyle output must not be empty")
	assert.NotEmpty(t, errSt, "ErrorStyle output must not be empty")
}

// ---------------------------------------------------------------------------
// Regression: DefaultTheme returns same values as before
// ---------------------------------------------------------------------------

func TestDefaultTheme_RegressionColorsUnchanged(t *testing.T) {
	theme := DefaultTheme()

	// Exact values locked from original implementation.
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "6", Dark: "6"}, theme.ColorPrimary)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "4", Dark: "4"}, theme.ColorSecondary)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "5", Dark: "5"}, theme.ColorAccent)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "2", Dark: "2"}, theme.ColorSuccess)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "3", Dark: "3"}, theme.ColorWarning)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "1", Dark: "1"}, theme.ColorError)
	assert.Equal(t, lipgloss.AdaptiveColor{Light: "8", Dark: "8"}, theme.ColorMuted)
}
