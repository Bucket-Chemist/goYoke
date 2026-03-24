// Package skeleton implements skeleton loading screens for the GOgent-Fortress
// TUI. Skeletons render a shimmering placeholder structure that visually
// matches the target component's layout, reducing perceived loading time when
// CLI initialisation takes longer than 500 ms.
//
// Usage pattern:
//
//	sk := skeleton.New(skeleton.SkeletonConversation)
//	sk = sk.SetSize(width, height)
//
//	// In Update:
//	case util.AnimateTickMsg:
//	    sk, cmd = sk.Update(msg)
//	    cmds = append(cmds, cmd)
//
//	// In View:
//	if sk.ShouldShow(elapsed) {
//	    return sk.View()
//	}
package skeleton

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// showThreshold is the minimum elapsed time before a skeleton is shown.
// Skeletons are hidden when content loads within this window, preventing a
// flash of placeholder content on fast machines.
const showThreshold = 500 * time.Millisecond

// shimmerWidth is the fraction of the total line width occupied by the
// shimmer highlight band. A value of 0.25 means the band is 25% of the line.
const shimmerFraction = 0.25

// totalFrames is the number of animation frames for one full shimmer pass.
// The shimmer band travels from position 0 to (width + shimmerWidth) over
// totalFrames frames, then wraps back to the start.
const totalFrames = 90

// ---------------------------------------------------------------------------
// SkeletonVariant
// ---------------------------------------------------------------------------

// SkeletonVariant selects the visual layout pattern of the skeleton.
type SkeletonVariant int

const (
	// SkeletonConversation renders horizontal lines of varying width, mimicking
	// a chat conversation panel with alternating message bubbles.
	SkeletonConversation SkeletonVariant = iota

	// SkeletonAgentTree renders indented lines at varying depths, mimicking a
	// collapsible agent-tree panel.
	SkeletonAgentTree

	// SkeletonSettings renders key-value pairs with a label column and a wider
	// value column, mimicking a settings panel.
	SkeletonSettings

	// SkeletonDashboard renders metric-card blocks arranged in a grid, mimicking
	// the telemetry/dashboard panel.
	SkeletonDashboard
)

// ---------------------------------------------------------------------------
// lineSpec describes one skeleton row
// ---------------------------------------------------------------------------

// lineSpec describes a single skeleton line: its indent depth and width as a
// fraction of the available content width. These are intentionally kept as
// ratios so the layout scales correctly with any terminal width.
type lineSpec struct {
	indent    int     // left indent in characters
	widthFrac float64 // fraction of content width [0.0, 1.0]
}

// lineSets contains the ordered row specifications for each variant.
// The patterns are repeated (cycled) to fill the available height.
var lineSets = map[SkeletonVariant][]lineSpec{
	SkeletonConversation: {
		{indent: 0, widthFrac: 0.55},
		{indent: 0, widthFrac: 0.40},
		{indent: 0, widthFrac: 0.70},
		{indent: 4, widthFrac: 0.50},
		{indent: 4, widthFrac: 0.35},
		{indent: 4, widthFrac: 0.60},
	},
	SkeletonAgentTree: {
		{indent: 0, widthFrac: 0.50},
		{indent: 2, widthFrac: 0.45},
		{indent: 4, widthFrac: 0.40},
		{indent: 4, widthFrac: 0.35},
		{indent: 2, widthFrac: 0.42},
		{indent: 4, widthFrac: 0.38},
	},
	SkeletonSettings: {
		{indent: 0, widthFrac: 0.25},
		{indent: 0, widthFrac: 0.25},
		{indent: 0, widthFrac: 0.25},
		{indent: 0, widthFrac: 0.25},
		{indent: 0, widthFrac: 0.25},
		{indent: 0, widthFrac: 0.25},
	},
	SkeletonDashboard: {
		{indent: 0, widthFrac: 0.45},
		{indent: 0, widthFrac: 0.45},
		{indent: 0, widthFrac: 0.45},
		{indent: 0, widthFrac: 0.45},
	},
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

// skeletonBaseStyle is the muted background color for skeleton lines.
// It uses ANSI color 8 (gray) as a background to represent empty space.
var skeletonBaseStyle = lipgloss.NewStyle().
	Background(config.ColorMuted).
	Foreground(config.ColorMuted)

// skeletonShimmerStyle is the slightly brighter highlight band that slides
// across each line. It uses the secondary (blue) color to differentiate from
// the base.
var skeletonShimmerStyle = lipgloss.NewStyle().
	Background(config.ColorSecondary).
	Foreground(config.ColorSecondary)

// ---------------------------------------------------------------------------
// SkeletonModel
// ---------------------------------------------------------------------------

// SkeletonModel is the Bubbletea component for skeleton loading screens.
//
// SkeletonModel is a value type (consistent with the project's component
// pattern). Copy it when embedding in a parent model struct.
//
// The zero value is not usable; use New instead.
type SkeletonModel struct {
	variant SkeletonVariant
	width   int
	height  int
	frame   int  // current animation frame [0, totalFrames)
	active  bool // true while the shimmer animation is running
}

// New returns a SkeletonModel for the given variant.
// The model is active (animating) by default. Call SetSize to provide
// dimensions before calling View.
func New(variant SkeletonVariant) SkeletonModel {
	return SkeletonModel{
		variant: variant,
		active:  true,
	}
}

// SetSize returns a copy of m with the given width and height applied.
// This follows the value-receiver / copy pattern used by other components.
func (m SkeletonModel) SetSize(width, height int) SkeletonModel {
	m.width = width
	m.height = height
	return m
}

// ShouldShow returns true when elapsed time exceeds the 500 ms threshold.
// Callers should check this before rendering the skeleton so that fast
// initialisation does not produce a visible flash of placeholder content.
func (m SkeletonModel) ShouldShow(elapsed time.Duration) bool {
	return elapsed >= showThreshold
}

// Active returns true if the skeleton is currently animating.
func (m SkeletonModel) Active() bool {
	return m.active
}

// Update handles Bubbletea messages. It advances the shimmer frame on each
// util.AnimateTickMsg and schedules the next tick while active.
func (m SkeletonModel) Update(msg tea.Msg) (SkeletonModel, tea.Cmd) {
	switch msg.(type) {
	case util.AnimateTickMsg:
		if !m.active {
			return m, nil
		}
		m.frame = (m.frame + 1) % totalFrames
		return m, util.AnimateTickCmd()
	}
	return m, nil
}

// View renders the skeleton to a string. The output is a block of styled
// lines matching the variant's layout with a shimmer band sliding across.
//
// View returns an empty string when width or height is zero.
func (m SkeletonModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	specs := lineSets[m.variant]
	if len(specs) == 0 {
		return ""
	}

	// Number of content rows to render (leave no blank lines beyond height).
	rowCount := m.height

	var sb strings.Builder
	for row := range rowCount {
		spec := specs[row%len(specs)]

		// For settings variant, render key + value side by side.
		if m.variant == SkeletonSettings {
			line := m.renderSettingsRow(spec, row)
			sb.WriteString(line)
		} else {
			line := m.renderLine(spec, row)
			sb.WriteString(line)
		}

		if row < rowCount-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// Internal rendering helpers
// ---------------------------------------------------------------------------

// renderLine renders a single skeleton line for Conversation, AgentTree, and
// Dashboard variants. It fills lineWidth characters with the shimmer band at
// the appropriate position for the current frame.
func (m SkeletonModel) renderLine(spec lineSpec, row int) string {
	// Available width after the indent.
	contentWidth := m.width - spec.indent
	if contentWidth <= 0 {
		return strings.Repeat(" ", m.width)
	}

	lineWidth := int(float64(contentWidth) * spec.widthFrac)
	if lineWidth <= 0 {
		return strings.Repeat(" ", m.width)
	}

	shimmerW := max(1, int(float64(lineWidth)*shimmerFraction))

	// Shimmer position: the band travels over [0, lineWidth+shimmerW) in
	// totalFrames steps. We stagger rows by their index so adjacent rows do
	// not all flash in sync, making the animation feel more organic.
	rowOffset := (row * totalFrames / max(1, m.height))
	pos := ((m.frame + rowOffset) % totalFrames) * (lineWidth + shimmerW) / totalFrames

	// Build the line character by character.
	var sb strings.Builder

	// Indent prefix (always plain space, not colored).
	sb.WriteString(strings.Repeat(" ", spec.indent))

	for col := range lineWidth {
		if col >= pos && col < pos+shimmerW {
			// Shimmer band: render a highlighted character.
			sb.WriteString(skeletonShimmerStyle.Render(" "))
		} else {
			// Base skeleton bar.
			sb.WriteString(skeletonBaseStyle.Render(" "))
		}
	}

	// Pad right side to full terminal width with plain spaces.
	rightPad := m.width - spec.indent - lineWidth
	if rightPad > 0 {
		sb.WriteString(strings.Repeat(" ", rightPad))
	}

	return sb.String()
}

// renderSettingsRow renders a settings-style row: a narrow key column
// followed by a wider value column, separated by a gap.
func (m SkeletonModel) renderSettingsRow(spec lineSpec, row int) string {
	// Key column: ~25% of width; value column: ~55% of width; gap: ~2 chars.
	keyWidth := int(float64(m.width) * 0.25)
	gap := 2
	valWidth := int(float64(m.width) * 0.50)
	if keyWidth <= 0 {
		keyWidth = 1
	}
	if valWidth <= 0 {
		valWidth = 1
	}

	shimmerW := max(1, int(float64(keyWidth+valWidth)*shimmerFraction))
	totalBarWidth := keyWidth + gap + valWidth

	rowOffset := (row * totalFrames / max(1, m.height))
	pos := ((m.frame + rowOffset) % totalFrames) * (totalBarWidth + shimmerW) / totalFrames

	var sb strings.Builder

	for col := range totalBarWidth {
		// Leave the gap as plain space.
		if col >= keyWidth && col < keyWidth+gap {
			sb.WriteByte(' ')
			continue
		}
		if col >= pos && col < pos+shimmerW {
			sb.WriteString(skeletonShimmerStyle.Render(" "))
		} else {
			sb.WriteString(skeletonBaseStyle.Render(" "))
		}
	}

	// Right padding.
	rightPad := m.width - totalBarWidth
	if rightPad > 0 {
		sb.WriteString(strings.Repeat(" ", rightPad))
	}

	return sb.String()
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
