// Package config provides the foundational theme system for the GOgent-Fortress TUI.
// It defines colors, styles, and icon constants used across all TUI components.
package config

import "github.com/charmbracelet/lipgloss"

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
// The zero value is not usable; use DefaultTheme instead.
type Theme struct {
	// Colors
	ColorPrimary   lipgloss.AdaptiveColor
	ColorSecondary lipgloss.AdaptiveColor
	ColorAccent    lipgloss.AdaptiveColor
	ColorSuccess   lipgloss.AdaptiveColor
	ColorWarning   lipgloss.AdaptiveColor
	ColorError     lipgloss.AdaptiveColor
	ColorMuted     lipgloss.AdaptiveColor

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

// DefaultTheme returns the standard GOgent-Fortress theme.
func DefaultTheme() Theme {
	return Theme{
		ColorPrimary:   ColorPrimary,
		ColorSecondary: ColorSecondary,
		ColorAccent:    ColorAccent,
		ColorSuccess:   ColorSuccess,
		ColorWarning:   ColorWarning,
		ColorError:     ColorError,
		ColorMuted:     ColorMuted,

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
