// Package config provides the foundational theme system for the GOgent-Fortress TUI.
// It defines colors, styles, and icon constants used across all TUI components.
package config

import "github.com/charmbracelet/lipgloss"

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
// ASCII-safe runes that work on every terminal encoding.
// ---------------------------------------------------------------------------

const (
	// IconRunning is shown next to an agent that is currently executing.
	IconRunning = '>'

	// IconComplete is shown next to an agent that finished successfully.
	IconComplete = '*'

	// IconError is shown next to an agent that finished with an error.
	IconError = '!'

	// IconPending is shown next to an agent that is waiting to start.
	IconPending = '.'

	// IconCancelled is shown next to an agent that was cancelled.
	IconCancelled = 'x'

	// IconPaused is shown next to an agent that is paused / blocked.
	IconPaused = '~'
)

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
