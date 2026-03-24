// Package config provides the foundational theme system for the GOgent-Fortress TUI.
// This file defines vim-style keybindings as an optional overlay on the standard
// keybinding set.  Vim mode is off by default and can be toggled via the settings
// panel (TUI-062).
package config

import "github.com/charmbracelet/bubbles/key"

// ---------------------------------------------------------------------------
// VimMode
// ---------------------------------------------------------------------------

// VimMode represents the current vim input mode when vim keybindings are
// enabled.  The zero value (VimNormal) is the default resting state.
type VimMode int

const (
	// VimNormal is the movement-and-command state.  j/k/h/l control cursor
	// navigation; pressing i transitions to VimInsert.
	VimNormal VimMode = iota

	// VimInsert is the text-entry state.  All keys pass through to the
	// focused component.  Pressing Esc returns to VimNormal.
	VimInsert
)

// String returns the display name for the mode.  It is used by the status
// line vim indicator.
func (m VimMode) String() string {
	switch m {
	case VimNormal:
		return "NORMAL"
	case VimInsert:
		return "INSERT"
	default:
		return "UNKNOWN"
	}
}

// ---------------------------------------------------------------------------
// VimKeys
// ---------------------------------------------------------------------------

// VimKeys holds the key bindings that are active when VimEnabled is true.
// They are applied as an overlay: in VimNormal mode the navigation bindings
// are intercepted before the standard key-dispatch; in VimInsert mode all
// keys pass through to the standard handlers unchanged.
type VimKeys struct {
	// Up moves the selection/scroll upward (k).
	Up key.Binding

	// Down moves the selection/scroll downward (j).
	Down key.Binding

	// Left switches focus to the left panel (h).
	Left key.Binding

	// Right switches focus to the right panel (l).
	Right key.Binding

	// Top scrolls to the beginning of the current list/view (g).
	Top key.Binding

	// Bottom scrolls to the end of the current list/view (G).
	Bottom key.Binding

	// Insert enters VimInsert mode (i).
	Insert key.Binding

	// Normal returns to VimNormal mode (Esc).
	Normal key.Binding
}

// DefaultVimKeys returns a VimKeys set populated with the standard vi/vim
// single-key bindings.
func DefaultVimKeys() VimKeys {
	return VimKeys{
		Up: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "left panel"),
		),
		Right: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "right panel"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Insert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert mode"),
		),
		Normal: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal mode"),
		),
	}
}
