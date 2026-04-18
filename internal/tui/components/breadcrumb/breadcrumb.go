// Package breadcrumb implements a navigation breadcrumb trail for the
// GOgent-Fortress TUI. It renders a single-row trail of navigation context
// items between the tab bar and main content area, updating as the user
// switches focus or panel modes.
package breadcrumb

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// BreadcrumbItem
// ---------------------------------------------------------------------------

// BreadcrumbItem represents one level in the navigation trail.
// Label is the display text shown to the user. Key is an optional keyboard
// shortcut hint available for future extension.
type BreadcrumbItem struct {
	// Label is the display text for this breadcrumb segment.
	Label string
	// Key is an optional keyboard shortcut hint (e.g. "Tab").
	// Reserved for future display; not rendered in the current implementation.
	Key string
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

// breadcrumbMutedStyle renders ancestor crumbs (all except the last) in a
// de-emphasised appearance so the current location stands out.
var breadcrumbMutedStyle = lipgloss.NewStyle().
	Foreground(config.ColorMuted)

// breadcrumbActiveStyle renders the final (current) crumb in the primary
// color so it is visually distinct from ancestor crumbs.
var breadcrumbActiveStyle = lipgloss.NewStyle().
	Foreground(config.ColorPrimary).
	Bold(true)

// breadcrumbArrowStyle renders the separator arrow between crumbs.
var breadcrumbArrowStyle = lipgloss.NewStyle().
	Foreground(config.ColorMuted)

// ---------------------------------------------------------------------------
// internal renderedCrumb
// ---------------------------------------------------------------------------

// renderedCrumb holds both the plain-text (for width measurement) and styled
// (for final output) representations of a single crumb.
type renderedCrumb struct {
	plain  string
	styled string
}

// ---------------------------------------------------------------------------
// BreadcrumbModel
// ---------------------------------------------------------------------------

// BreadcrumbModel renders a navigation breadcrumb trail.
// It is a lightweight component with no Bubbletea tea.Model implementation:
// AppModel owns the pointer and calls setters directly, matching the pattern
// established by HintBarModel (TUI-060).
type BreadcrumbModel struct {
	items []BreadcrumbItem
	width int
	theme config.Theme
}

// NewBreadcrumbModel returns a BreadcrumbModel initialised with the default
// theme and no items.
func NewBreadcrumbModel() *BreadcrumbModel {
	return &BreadcrumbModel{
		theme: config.DefaultTheme(),
	}
}

// SetCrumbs replaces the current navigation trail with label-only items,
// constructing a BreadcrumbItem for each label with an empty Key field.
// This is the primary setter used by AppModel via the breadcrumbWidget
// interface: the model package passes plain strings to stay decoupled from
// this package's BreadcrumbItem type.
// Calling SetCrumbs with nil or an empty slice clears the trail; View will
// return an empty string until crumbs are set again.
func (m *BreadcrumbModel) SetCrumbs(labels []string) {
	if len(labels) == 0 {
		m.items = nil
		return
	}
	items := make([]BreadcrumbItem, len(labels))
	for i, l := range labels {
		items[i] = BreadcrumbItem{Label: l}
	}
	m.items = items
}

// SetCrumbItems replaces the current navigation trail with the given items,
// preserving the Key field for each entry.  Use this method when keyboard
// shortcut hints must be stored alongside labels.
func (m *BreadcrumbModel) SetCrumbItems(items []BreadcrumbItem) {
	m.items = items
}

// SetWidth updates the terminal width used for truncation in View.
func (m *BreadcrumbModel) SetWidth(width int) {
	m.width = width
}

// SetTheme applies a new color theme to the breadcrumb component.
func (m *BreadcrumbModel) SetTheme(theme config.Theme) {
	m.theme = theme
}

// View renders the breadcrumb trail to a string.
//
// Rendering rules:
//   - Returns "" when items is empty (caller must not allocate a row).
//   - Items are separated by the Arrow icon from the active theme's IconSet.
//   - All items except the last are rendered in a muted/dim style.
//   - The last item (current location) is rendered bold in the primary color.
//   - When the total plain-text length exceeds width, items are dropped from
//     the left with a "..." prefix so the current location always remains
//     visible.
func (m *BreadcrumbModel) View() string {
	if len(m.items) == 0 {
		return ""
	}

	arrow := m.theme.Icons().Arrow
	sep := " " + arrow + " "
	sepLen := len(sep)

	// Build plain and styled representations of each crumb.
	crumbs := make([]renderedCrumb, len(m.items))
	for i, item := range m.items {
		label := item.Label
		if i == len(m.items)-1 {
			// Last item: active / current location.
			crumbs[i] = renderedCrumb{
				plain:  label,
				styled: breadcrumbActiveStyle.Render(label),
			}
		} else {
			crumbs[i] = renderedCrumb{
				plain:  label,
				styled: breadcrumbMutedStyle.Render(label),
			}
		}
	}

	// If width is unset or very small, render only the last (current) crumb
	// without attempting truncation arithmetic to avoid edge-case panics.
	if m.width <= 0 {
		return crumbs[len(crumbs)-1].styled
	}

	totalLen := measureTrail(crumbs, sepLen)

	if totalLen <= m.width {
		// Everything fits: join in order and return.
		return joinRenderedCrumbs(crumbs, arrow)
	}

	// Truncate from the left: drop leading crumbs until the trail fits.
	// Always keep the last (current) crumb visible. A "..." prefix is
	// prepended to signal that ancestor items were omitted.
	ellipsis := "..."
	ellipsisLen := len(ellipsis)
	// The prefix counts as: ellipsis + one separator gap.
	prefixLen := ellipsisLen + sepLen

	lastCrumb := crumbs[len(crumbs)-1]

	// Find the smallest start index such that the truncated trail fits.
	// We start at 1 (drop crumb[0] first) and increase until it fits.
	// If even a single crumb + prefix does not fit, fall back to showing
	// just the last crumb (current location must always be visible).
	start := 1
	for start < len(crumbs) {
		trailLen := measureTrail(crumbs[start:], sepLen) + prefixLen
		if trailLen <= m.width {
			break
		}
		start++
	}

	if start >= len(crumbs) {
		// Nothing fits with the ellipsis prefix — show only the current crumb.
		return lastCrumb.styled
	}

	var sb strings.Builder
	sb.WriteString(breadcrumbMutedStyle.Render(ellipsis))
	sb.WriteString(buildSep(arrow))
	sb.WriteString(joinRenderedCrumbs(crumbs[start:], arrow))
	return sb.String()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// measureTrail computes the plain-text display width of a crumbs slice when
// joined by the separator (length sepLen per gap).
func measureTrail(crumbs []renderedCrumb, sepLen int) int {
	total := 0
	for i, c := range crumbs {
		total += len(c.plain)
		if i < len(crumbs)-1 {
			total += sepLen
		}
	}
	return total
}

// joinRenderedCrumbs joins all crumbs with the styled arrow separator.
func joinRenderedCrumbs(crumbs []renderedCrumb, arrow string) string {
	if len(crumbs) == 0 {
		return ""
	}
	parts := make([]string, len(crumbs))
	for i, c := range crumbs {
		parts[i] = c.styled
	}
	return strings.Join(parts, buildSep(arrow))
}

// buildSep returns the styled separator string (" › ") used between crumbs.
func buildSep(arrow string) string {
	return breadcrumbArrowStyle.Render(" " + arrow + " ")
}
