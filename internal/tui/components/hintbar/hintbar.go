// Package hintbar implements a context-aware keyboard hint bar for the
// GOgent-Fortress TUI. It renders a single row of muted key:description pairs
// that update based on the current focus/state context.
package hintbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// HintItem
// ---------------------------------------------------------------------------

// HintItem is a single keyboard shortcut hint.
type HintItem struct {
	// Key is the key label shown to the user, e.g. "Tab", "ctrl+f", "alt+v".
	Key string
	// Desc is the short description shown after the key, e.g. "next panel".
	Desc string
}

// ---------------------------------------------------------------------------
// Context hint sets
// ---------------------------------------------------------------------------

// hintSets maps context names to their corresponding hint items.
// Each set is ordered from most-important to least-important so that
// truncation from the right drops low-priority hints first.
var hintSets = map[string][]HintItem{
	"main": {
		{Key: "Tab", Desc: "next panel"},
		{Key: "Shift+Tab", Desc: "prev panel"},
		{Key: "ctrl+f", Desc: "search"},
		{Key: "/", Desc: "slash cmd"},
	},
	"settings": {
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "toggle"},
		{Key: "Esc", Desc: "close"},
	},
	"search": {
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "select"},
		{Key: "Esc", Desc: "close"},
	},
	"modal": {
		{Key: "Enter", Desc: "confirm"},
		{Key: "Esc", Desc: "cancel"},
	},
	"plan": {
		{Key: "alt+v", Desc: "view plan"},
		{Key: "Esc", Desc: "exit plan"},
	},
}

// separator is the string used between hint items in the rendered row.
const separator = "  "

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

// hintKeyStyle renders the key label in a slightly brighter muted color.
var hintKeyStyle = lipgloss.NewStyle().
	Foreground(config.ColorMuted).
	Bold(true)

// hintDescStyle renders the description in standard muted color.
var hintDescStyle = lipgloss.NewStyle().
	Foreground(config.ColorMuted)

// hintSepStyle renders the separator between hint items.
var hintSepStyle = lipgloss.NewStyle().
	Foreground(config.ColorMuted)

// ---------------------------------------------------------------------------
// HintBarModel
// ---------------------------------------------------------------------------

// HintBarModel renders context-aware keyboard hints as a single-line bar.
// It is a lightweight value-type component: no tea.Model implementation is
// needed because it has no internal tick, animation, or command dispatch.
// AppModel owns the pointer and calls the setters directly.
type HintBarModel struct {
	hints   []HintItem
	context string
	width   int
	visible bool
}

// NewHintBarModel returns a HintBarModel initialised for the "main" context
// and set to visible.
func NewHintBarModel() *HintBarModel {
	h := &HintBarModel{
		visible: true,
	}
	h.SetContext("main")
	return h
}

// SetContext switches the active hint set.  If the context name is not
// recognised the model falls back to the "main" hint set.
func (h *HintBarModel) SetContext(context string) {
	h.context = context
	hints, ok := hintSets[context]
	if !ok {
		hints = hintSets["main"]
	}
	h.hints = hints
}

// SetWidth updates the terminal width used for truncation in View.
func (h *HintBarModel) SetWidth(width int) {
	h.width = width
}

// IsVisible returns true when the hint bar is currently shown.
func (h *HintBarModel) IsVisible() bool {
	return h.visible
}

// Show makes the hint bar visible.
func (h *HintBarModel) Show() {
	h.visible = true
}

// Hide hides the hint bar.
func (h *HintBarModel) Hide() {
	h.visible = false
}

// View renders the hint bar to a string.  Returns an empty string when
// not visible or when there are no hints to show.
//
// Rendering format: "key:desc  key:desc  key:desc"
//
// If the total rendered width exceeds the terminal width, hints are dropped
// from the right (least-important first) until the line fits.  A trailing
// "..." is appended when items are omitted so the user knows the list is
// truncated.
func (h *HintBarModel) View() string {
	if !h.visible || len(h.hints) == 0 {
		return ""
	}

	// Build individual rendered hint tokens (unstyled plain text lengths are
	// needed to measure width; styled versions are assembled at the end).
	type renderedHint struct {
		plain  string // used for width measurement
		styled string // used for final output
	}

	rendered := make([]renderedHint, 0, len(h.hints))
	for _, hint := range h.hints {
		plain := hint.Key + ":" + hint.Desc
		styled := hintKeyStyle.Render(hint.Key) +
			hintDescStyle.Render(":"+hint.Desc)
		rendered = append(rendered, renderedHint{plain: plain, styled: styled})
	}

	// If width is unset (0) or very small, return the first hint only without
	// attempting truncation arithmetic to avoid negative-width edge cases.
	if h.width <= 0 {
		return rendered[0].styled
	}

	ellipsis := "..."
	ellipsisLen := len(ellipsis)

	// Determine how many hints fit within the available width.
	// We iterate forward, accumulating the plain-text width of each token
	// plus the separator.  If adding the next token would overflow we stop
	// and check whether we need to append the ellipsis.
	sepLen := len(separator)
	totalLen := 0
	fitCount := 0

	for i, r := range rendered {
		tokenLen := len(r.plain)
		if i > 0 {
			tokenLen += sepLen
		}
		if totalLen+tokenLen > h.width {
			break
		}
		totalLen += tokenLen
		fitCount++
	}

	if fitCount == 0 {
		// Not even one hint fits: return nothing to avoid visual clutter.
		return ""
	}

	// If we fit all hints, just join and return.
	if fitCount == len(rendered) {
		parts := make([]string, fitCount)
		for i := range fitCount {
			parts[i] = rendered[i].styled
		}
		return strings.Join(parts, hintSepStyle.Render(separator))
	}

	// Some hints were dropped: re-fit with ellipsis appended.
	// Reserve space for the separator + ellipsis at the end.
	totalLen = 0
	fitCount = 0
	for i, r := range rendered {
		tokenLen := len(r.plain)
		if i > 0 {
			tokenLen += sepLen
		}
		// Reserve space for "  ..." after the last included item.
		trailingSpace := sepLen + ellipsisLen
		if totalLen+tokenLen+trailingSpace > h.width {
			break
		}
		totalLen += tokenLen
		fitCount++
	}

	if fitCount == 0 {
		return ""
	}

	parts := make([]string, fitCount)
	for i := range fitCount {
		parts[i] = rendered[i].styled
	}
	joined := strings.Join(parts, hintSepStyle.Render(separator))
	return joined + hintSepStyle.Render(separator) + hintDescStyle.Render(ellipsis)
}
