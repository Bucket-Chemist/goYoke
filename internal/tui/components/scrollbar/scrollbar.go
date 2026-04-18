// Package scrollbar renders a single-column vertical scrollbar track and thumb.
// It is a pure rendering utility with no Bubbletea model state; callers pass
// viewport geometry and scroll position and receive a styled string.
package scrollbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

const (
	trackChar = "░" // U+2591 LIGHT SHADE
	thumbChar  = "▓" // U+2593 DARK SHADE
)

// Render returns a single-column vertical scrollbar string of exactly
// viewportHeight lines (one styled character per line, joined by "\n").
//
// Returns an empty string when contentHeight <= viewportHeight, indicating
// no scrollbar is needed.
//
// Track cells are styled with config.ColorMuted; thumb cells with
// config.ColorPrimary.
func Render(viewportHeight, contentHeight, scrollOffset int) string {
	return RenderStyled(viewportHeight, contentHeight, scrollOffset,
		config.ColorMuted, config.ColorPrimary)
}

// RenderStyled is the themed variant of Render. It accepts explicit
// lipgloss.AdaptiveColor values for the track and thumb, enabling callers to
// apply any color palette without coupling to the package-level defaults.
func RenderStyled(
	viewportHeight, contentHeight, scrollOffset int,
	trackColor, thumbColor lipgloss.AdaptiveColor,
) string {
	if contentHeight <= viewportHeight {
		return ""
	}

	trackStyle := lipgloss.NewStyle().Foreground(trackColor)
	thumbStyle := lipgloss.NewStyle().Foreground(thumbColor)

	// Thumb occupies a fraction of the track proportional to the viewport.
	thumbSize := viewportHeight * viewportHeight / contentHeight
	if thumbSize < 1 {
		thumbSize = 1
	}

	// Clamp scrollOffset to the valid range before computing position.
	maxOffset := contentHeight - viewportHeight
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}

	// Map scrollOffset onto the available track positions above the thumb.
	thumbPos := 0
	if trackPositions := viewportHeight - thumbSize; trackPositions > 0 {
		thumbPos = scrollOffset * trackPositions / maxOffset
	}

	lines := make([]string, viewportHeight)
	for i := range viewportHeight {
		if i >= thumbPos && i < thumbPos+thumbSize {
			lines[i] = thumbStyle.Render(thumbChar)
		} else {
			lines[i] = trackStyle.Render(trackChar)
		}
	}

	return strings.Join(lines, "\n")
}
