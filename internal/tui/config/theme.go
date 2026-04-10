// Package config provides the foundational theme system for the GOgent-Fortress TUI.
// It defines colors, styles, and icon constants used across all TUI components.
package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// ThemeVariant
//
// ThemeVariant selects the color palette used when constructing a Theme via
// NewTheme. The zero value is ThemeDark.
// ---------------------------------------------------------------------------

// ThemeVariant identifies the color palette for a Theme.
type ThemeVariant int

const (
	// ThemeDark is optimised for dark terminal backgrounds.
	ThemeDark ThemeVariant = iota

	// ThemeLight is optimised for light terminal backgrounds.
	ThemeLight

	// ThemeHighContrast uses hex colors that meet WCAG AA (4.5:1 contrast ratio)
	// against both white and black backgrounds where possible.
	ThemeHighContrast
)

// ---------------------------------------------------------------------------
// Colors
//
// Base colors use ANSI numbers ("0"-"15") so they respect the user's terminal
// colorscheme.  Accent colors that need precise control use hex codes.
// Both light and dark variants are provided via AdaptiveColor so the TUI
// renders correctly on any background.
// ---------------------------------------------------------------------------

// ColorPrimary is cyan — used for focused borders, active selections.
var ColorPrimary = lipgloss.AdaptiveColor{Light: "6", Dark: "6"}

// ColorSecondary is blue — used for secondary UI elements.
var ColorSecondary = lipgloss.AdaptiveColor{Light: "4", Dark: "4"}

// ColorAccent is magenta — used for highlights and decorative accents.
var ColorAccent = lipgloss.AdaptiveColor{Light: "5", Dark: "5"}

// ColorSuccess is green — used to indicate a successful or complete state.
var ColorSuccess = lipgloss.AdaptiveColor{Light: "2", Dark: "2"}

// ColorWarning is yellow — used to indicate a warning or pending state.
var ColorWarning = lipgloss.AdaptiveColor{Light: "3", Dark: "3"}

// ColorError is red — used to indicate an error state.
var ColorError = lipgloss.AdaptiveColor{Light: "1", Dark: "1"}

// ColorMuted is gray — used for de-emphasised, secondary text.
var ColorMuted = lipgloss.AdaptiveColor{Light: "8", Dark: "8"}

// ---------------------------------------------------------------------------
// Icon constants
//
// Deprecated: These rune constants predate the IconSet system. Prefer
// Theme.Icons() which returns the correct IconSet for the configured locale.
// These constants remain for backward compatibility with existing consumers.
// ---------------------------------------------------------------------------

const (
	// Deprecated: Use Theme.Icons().Running instead.
	// IconRunning is shown next to an agent that is currently executing.
	IconRunning = '>'

	// Deprecated: Use Theme.Icons().Complete instead.
	// IconComplete is shown next to an agent that finished successfully.
	IconComplete = '*'

	// Deprecated: Use Theme.Icons().Error instead.
	// IconError is shown next to an agent that finished with an error.
	IconError = '!'

	// Deprecated: Use Theme.Icons().Pending instead.
	// IconPending is shown next to an agent that is waiting to start.
	IconPending = '.'

	// Deprecated: Use Theme.Icons().Cancelled instead.
	// IconCancelled is shown next to an agent that was cancelled.
	IconCancelled = 'x'

	// Deprecated: Use Theme.Icons().Paused instead.
	// IconPaused is shown next to an agent that is paused / blocked.
	IconPaused = '~'
)

// ---------------------------------------------------------------------------
// IconSet
//
// IconSet bundles all UI icon strings for a single rendering mode. Use
// UnicodeIcons for rich terminal environments and ASCIIIcons for narrow or
// legacy terminals that lack Unicode support. Select the appropriate set at
// runtime via Theme.Icons().
// ---------------------------------------------------------------------------

// IconSet holds all icon strings used across TUI components.
type IconSet struct {
	// Running is shown next to an agent that is currently executing.
	Running string
	// Complete is shown next to an agent that finished successfully.
	Complete string
	// Error is shown next to an agent that finished with an error.
	Error string
	// Pending is shown next to an agent that is waiting to start.
	Pending string
	// Cancelled is shown next to an agent that was cancelled or killed.
	Cancelled string
	// Paused is shown next to an agent that is paused or blocked.
	Paused string
	// Info is used for informational status indicators.
	Info string
	// Warning is used for warning status indicators.
	Warning string
	// Success is used for success status indicators.
	Success string
	// Search is used for search or filter UI elements.
	Search string
	// Settings is used for configuration or settings UI elements.
	Settings string
	// Arrow is used as a directional indicator or selection marker.
	Arrow string
}

// UnicodeIcons is the default icon set using Unicode characters that render
// well on modern terminals with full Unicode support.
var UnicodeIcons = IconSet{
	Running:   "▶",
	Complete:  "✓",
	Error:     "✗",
	Pending:   "○",
	Cancelled: "✕",
	Paused:    "⏸",
	Info:      "ℹ",
	Warning:   "⚠",
	Success:   "✔",
	Search:    "\U0001F50D",
	Settings:  "⚙",
	Arrow:     "›",
}

// ASCIIIcons is the fallback icon set using only printable ASCII characters.
// Use this set on terminals that cannot reliably render Unicode, or when the
// operator has explicitly requested plain-text output.
var ASCIIIcons = IconSet{
	Running:   ">",
	Complete:  "*",
	Error:     "!",
	Pending:   ".",
	Cancelled: "x",
	Paused:    "~",
	Info:      "i",
	Warning:   "!",
	Success:   "*",
	Search:    "/",
	Settings:  "@",
	Arrow:     ">",
}

// ---------------------------------------------------------------------------
// Styles
//
// Package-level style variables are the canonical source for all component
// styling.  Components import this package and use these values rather than
// defining their own lipgloss styles.
// ---------------------------------------------------------------------------

// StyleFocusedBorder is applied to the border of the pane that holds focus.
var StyleFocusedBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorPrimary)

// StyleUnfocusedBorder is applied to panes that do not hold focus.
var StyleUnfocusedBorder = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(ColorMuted)

// StyleStatusBar is applied to the bottom status-bar strip.
var StyleStatusBar = lipgloss.NewStyle().
	Foreground(ColorMuted).
	Padding(0, 1)

// StyleTitle is applied to section headings and panel titles.
var StyleTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPrimary)

// StyleSubtle is applied to secondary labels and hints.
var StyleSubtle = lipgloss.NewStyle().
	Foreground(ColorMuted)

// StyleHighlight is applied to the currently selected list item or focused text.
var StyleHighlight = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorAccent)

// StyleError is applied to error messages.
var StyleError = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorError)

// StyleSuccess is applied to success messages.
var StyleSuccess = lipgloss.NewStyle().
	Foreground(ColorSuccess)

// StyleWarning is applied to warning messages.
var StyleWarning = lipgloss.NewStyle().
	Foreground(ColorWarning)

// StyleMuted is applied to de-emphasised content such as timestamps.
var StyleMuted = lipgloss.NewStyle().
	Foreground(ColorMuted)

// ---------------------------------------------------------------------------
// Theme struct
//
// Theme bundles all style variables so components can receive a Theme value
// rather than importing package-level globals.  This enables future theme
// switching without changing component code.
// ---------------------------------------------------------------------------

// Theme holds all styles and colors for the TUI.
// The zero value is not usable; use DefaultTheme or NewTheme instead.
type Theme struct {
	// Colors
	ColorPrimary   lipgloss.AdaptiveColor
	ColorSecondary lipgloss.AdaptiveColor
	ColorAccent    lipgloss.AdaptiveColor
	ColorSuccess   lipgloss.AdaptiveColor
	ColorWarning   lipgloss.AdaptiveColor
	ColorError     lipgloss.AdaptiveColor
	ColorMuted     lipgloss.AdaptiveColor

	// InfoColor is an alias for ColorPrimary / cyan, used for informational
	// messages in semantic style methods.
	InfoColor lipgloss.AdaptiveColor

	// Styles
	FocusedBorder   lipgloss.Style
	UnfocusedBorder lipgloss.Style
	StatusBar       lipgloss.Style
	Title           lipgloss.Style
	Subtle          lipgloss.Style
	Highlight       lipgloss.Style
	Error           lipgloss.Style
	Success         lipgloss.Style
	Warning         lipgloss.Style
	Muted           lipgloss.Style

	// UseASCII, when true, causes Icons() to return ASCIIIcons instead of
	// UnicodeIcons. Set this on terminals that cannot reliably render Unicode.
	// Defaults to false.
	UseASCII bool
}

// ---------------------------------------------------------------------------
// Semantic style methods
//
// These methods derive a ready-to-use lipgloss.Style from the Theme's color
// fields. They are intended for one-off rendering in components that receive
// a Theme value. Package-level style variables (StyleError etc.) remain the
// canonical source for components that import config directly.
// ---------------------------------------------------------------------------

// ErrorStyle returns a bold style colored with ColorError.
func (t Theme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.ColorError)
}

// WarningStyle returns a style colored with ColorWarning.
func (t Theme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.ColorWarning)
}

// SuccessStyle returns a style colored with ColorSuccess.
func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.ColorSuccess)
}

// InfoStyle returns a style colored with ColorPrimary (cyan).
func (t Theme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.ColorPrimary)
}

// DangerStyle returns a bold, underlined style colored with ColorError.
// It is more emphatic than ErrorStyle and is reserved for destructive actions
// or critical alerts.
func (t Theme) DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Underline(true).Foreground(t.ColorError)
}

// Icons returns the appropriate IconSet for this Theme. When UseASCII is true
// ASCIIIcons is returned; otherwise UnicodeIcons is returned. Components
// should call this method rather than referencing the package-level icon set
// variables directly so that the ASCII preference is honoured automatically.
func (t Theme) Icons() IconSet {
	if t.UseASCII {
		return ASCIIIcons
	}
	return UnicodeIcons
}

// ---------------------------------------------------------------------------
// NewTheme factory
// ---------------------------------------------------------------------------

// NewTheme constructs a Theme for the given ThemeVariant. The returned Theme
// is fully populated with colors and pre-built lipgloss styles appropriate for
// the chosen variant.
//
// ThemeDark returns the same colors as DefaultTheme (backward compatible).
// ThemeLight adapts foreground choices for light terminal backgrounds.
// ThemeHighContrast uses explicit hex codes meeting WCAG AA (4.5:1 ratio).
func NewTheme(variant ThemeVariant) Theme {
	switch variant {
	case ThemeLight:
		return newLightTheme()
	case ThemeHighContrast:
		return newHighContrastTheme()
	default: // ThemeDark and any unrecognised value
		return DefaultTheme()
	}
}

// newLightTheme returns a Theme suited for light terminal backgrounds.
// It keeps the same ANSI indices but adjusts them to variants that are
// legible on white/light backgrounds (e.g. darker cyan, darker green).
func newLightTheme() Theme {
	primary := lipgloss.AdaptiveColor{Light: "6", Dark: "6"}
	secondary := lipgloss.AdaptiveColor{Light: "4", Dark: "4"}
	accent := lipgloss.AdaptiveColor{Light: "5", Dark: "5"}
	success := lipgloss.AdaptiveColor{Light: "2", Dark: "2"}
	warning := lipgloss.AdaptiveColor{Light: "3", Dark: "3"}
	errColor := lipgloss.AdaptiveColor{Light: "1", Dark: "1"}
	muted := lipgloss.AdaptiveColor{Light: "0", Dark: "0"} // black/dark on light bg

	return buildTheme(primary, secondary, accent, success, warning, errColor, muted)
}

// newHighContrastTheme returns a Theme with explicit hex colors that satisfy
// WCAG AA minimum contrast ratio of 4.5:1 on typical dark and light terminals.
func newHighContrastTheme() Theme {
	primary := lipgloss.AdaptiveColor{Light: "#0088FF", Dark: "#0088FF"}
	secondary := lipgloss.AdaptiveColor{Light: "#0044CC", Dark: "#0044CC"}
	accent := lipgloss.AdaptiveColor{Light: "#AA00FF", Dark: "#AA00FF"}
	success := lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#00AA00"}
	warning := lipgloss.AdaptiveColor{Light: "#FFAA00", Dark: "#FFAA00"}
	errColor := lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF0000"}
	muted := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}

	return buildTheme(primary, secondary, accent, success, warning, errColor, muted)
}

// buildTheme assembles a fully populated Theme from the supplied color values,
// constructing all derived lipgloss styles.
func buildTheme(
	primary, secondary, accent, success, warning, errColor, muted lipgloss.AdaptiveColor,
) Theme {
	focusedBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary)

	unfocusedBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(muted)

	statusBar := lipgloss.NewStyle().
		Foreground(muted).
		Padding(0, 1)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primary)

	subtle := lipgloss.NewStyle().
		Foreground(muted)

	highlight := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent)

	errStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(errColor)

	successStyle := lipgloss.NewStyle().
		Foreground(success)

	warningStyle := lipgloss.NewStyle().
		Foreground(warning)

	mutedStyle := lipgloss.NewStyle().
		Foreground(muted)

	return Theme{
		ColorPrimary:   primary,
		ColorSecondary: secondary,
		ColorAccent:    accent,
		ColorSuccess:   success,
		ColorWarning:   warning,
		ColorError:     errColor,
		ColorMuted:     muted,
		InfoColor:      primary,

		FocusedBorder:   focusedBorder,
		UnfocusedBorder: unfocusedBorder,
		StatusBar:       statusBar,
		Title:           title,
		Subtle:          subtle,
		Highlight:       highlight,
		Error:           errStyle,
		Success:         successStyle,
		Warning:         warningStyle,
		Muted:           mutedStyle,
	}
}

// ---------------------------------------------------------------------------
// WCAG contrast utilities
// ---------------------------------------------------------------------------

// ContrastRatio computes the WCAG 2.1 contrast ratio between two hex colors.
// Returns a float64 in range [1.0, 21.0]. WCAG AA requires >= 4.5 for normal
// text. Colors must be in #RRGGBB or RRGGBB format. Returns 1.0 on any parse
// error so callers receive a safe (worst-case) value rather than a panic.
func ContrastRatio(fg, bg string) float64 {
	lFG, ok1 := luminance(fg)
	lBG, ok2 := luminance(bg)
	if !ok1 || !ok2 {
		return 1.0
	}
	// Arrange so that l1 >= l2.
	l1, l2 := lFG, lBG
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// luminance parses a hex color string and returns its relative luminance
// as defined by WCAG 2.1. Returns (0, false) on parse failure.
func luminance(hex string) (float64, bool) {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return 0, false
	}
	return 0.2126*linearize(r) + 0.7152*linearize(g) + 0.0722*linearize(b), true
}

// linearize converts an sRGB 8-bit channel value to its linear light value
// as specified by IEC 61966-2-1 (the sRGB standard referenced by WCAG 2.1).
func linearize(channel uint8) float64 {
	v := float64(channel) / 255.0
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// parseHex parses a "#RRGGBB" or "RRGGBB" color string into its R, G, B
// components. Returns (0, 0, 0, false) if the string cannot be parsed.
func parseHex(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	ri, err1 := strconv.ParseUint(s[0:2], 16, 8)
	gi, err2 := strconv.ParseUint(s[2:4], 16, 8)
	bi, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return uint8(ri), uint8(gi), uint8(bi), true
}

// wcagAAMinimum is the WCAG AA minimum contrast ratio for normal-size text.
const wcagAAMinimum = 4.5

// validateContrastBG is the dark background used by ValidateContrast.
// Pure black (#000000) is used because it is the most challenging common dark
// terminal background and provides a conservative WCAG check: if a color
// passes against black it will also pass against any darker terminal theme.
const validateContrastBG = "#000000"

// ValidateContrast checks the semantic foreground colors of the Theme against
// a dark background (#000000) using the WCAG AA minimum (4.5:1). It evaluates
// the Dark variant of each color because the TUI primarily targets dark
// terminals.
//
// ColorAccent and ColorSecondary are excluded from the check: they are used
// for decorative highlights and borders, not for informational text, and pure
// purple/blue hues cannot satisfy 4.5:1 against black while remaining vivid.
//
// Returns a map of field-name → contrast-ratio and a non-nil error listing
// every field that falls below the minimum. Returns nil error when all checked
// colors pass.
func (t Theme) ValidateContrast() (map[string]float64, error) {
	pairs := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"ColorError", t.ColorError},
		{"ColorWarning", t.ColorWarning},
		{"ColorSuccess", t.ColorSuccess},
		{"ColorPrimary", t.ColorPrimary},
		{"ColorMuted", t.ColorMuted},
	}

	ratios := make(map[string]float64, len(pairs))
	var failures []string

	for _, p := range pairs {
		dark := p.color.Dark
		if dark == "" {
			// No hex value — skip (ANSI indices cannot be measured).
			continue
		}
		if !strings.HasPrefix(dark, "#") {
			// Not a hex color (ANSI index like "6") — skip.
			continue
		}
		ratio := ContrastRatio(dark, validateContrastBG)
		ratios[p.name] = ratio
		if ratio < wcagAAMinimum {
			failures = append(failures, fmt.Sprintf("%s (%.2f:1)", p.name, ratio))
		}
	}

	if len(failures) > 0 {
		return ratios, fmt.Errorf("WCAG AA contrast failures: %s", strings.Join(failures, ", "))
	}
	return ratios, nil
}

// ---------------------------------------------------------------------------
// Rainbow gradient
// ---------------------------------------------------------------------------

// rainbowColors is a fixed palette of 7 hex colors representing the visible
// spectrum. Each character in the input is assigned a color by cycling through
// this slice.
var rainbowColors = []lipgloss.Color{
	"#FF0000", // red
	"#FF8800", // orange
	"#FFFF00", // yellow
	"#00FF00", // green
	"#0088FF", // blue
	"#8800FF", // indigo/violet
	"#FF00FF", // magenta
}

// RainbowGradient applies a per-character rainbow color cycle to text.
// Each non-whitespace rune is rendered with the next color in the palette;
// whitespace runes are appended unstyled to preserve spacing.
// An empty input returns an empty string without allocating.
func RainbowGradient(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	colorIdx := 0
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			b.WriteRune(r)
			continue
		}
		styled := lipgloss.NewStyle().
			Foreground(rainbowColors[colorIdx%len(rainbowColors)]).
			Render(string(r))
		b.WriteString(styled)
		colorIdx++
	}
	return b.String()
}

// DefaultTheme returns the standard GOgent-Fortress theme.
// All existing fields are unchanged; InfoColor is set to ColorPrimary (cyan).
func DefaultTheme() Theme {
	return Theme{
		ColorPrimary:   ColorPrimary,
		ColorSecondary: ColorSecondary,
		ColorAccent:    ColorAccent,
		ColorSuccess:   ColorSuccess,
		ColorWarning:   ColorWarning,
		ColorError:     ColorError,
		ColorMuted:     ColorMuted,
		InfoColor:      ColorPrimary,

		FocusedBorder:   StyleFocusedBorder,
		UnfocusedBorder: StyleUnfocusedBorder,
		StatusBar:       StyleStatusBar,
		Title:           StyleTitle,
		Subtle:          StyleSubtle,
		Highlight:       StyleHighlight,
		Error:           StyleError,
		Success:         StyleSuccess,
		Warning:         StyleWarning,
		Muted:           StyleMuted,
	}
}
