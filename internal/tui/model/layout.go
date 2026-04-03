// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains layout constants, dimension computation, and rendering
// helpers for the Lipgloss-based terminal layout.
package model

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// truncateHeight truncates s to at most maxLines newline-delimited lines.
// This is necessary because lipgloss.Height() only pads shorter content — it
// does NOT clip content that exceeds the specified height. Without explicit
// truncation, any overflow (stale panel dimensions, unaccounted chrome rows,
// phantom JoinVertical blank lines) propagates to the final render and pushes
// content off-screen.
func truncateHeight(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// ---------------------------------------------------------------------------
// Layout constants
//
// These define the fixed-height allocations for chrome rows.
// ---------------------------------------------------------------------------

const (
	bannerHeight      = 3 // rounded border top + title + border bottom
	tabBarHeight      = 1 // single-row strip
	statusLineHeight  = 2 // two-row status bar
	hintBarHeight     = 1 // single-row keyboard hint bar (TUI-060)
	breadcrumbHeight  = 1 // single-row breadcrumb trail (TUI-063)
	borderFrame       = 2 // border chars on each axis (1 left + 1 right)
	separatorHeight   = 1 // horizontal separator between conversation viewport and input (TUI-L02)
)

// ---------------------------------------------------------------------------
// LayoutTier
// ---------------------------------------------------------------------------

// LayoutTier describes the responsive breakpoint tier for the current terminal
// width.  It is stored in layoutDims so that components can adapt their
// rendering to the available horizontal space without re-computing breakpoints.
type LayoutTier int

const (
	// LayoutCompact is used when the terminal is narrower than 80 columns.
	// Only a single column is shown; the right panel is hidden.
	LayoutCompact LayoutTier = iota

	// LayoutStandard covers 80–119 columns.  Both panels are visible with a
	// 75/25 (80–99) or 70/30 (100–119) split — matching the pre-TUI-058
	// behaviour exactly.
	LayoutStandard

	// LayoutWide covers 120–179 columns.  The right panel receives a larger
	// share of the available space (60/40 split).
	LayoutWide

	// LayoutUltra covers terminals that are 180 columns or wider.  Both
	// panels receive an equal share (50/50 split).
	LayoutUltra
)

// String returns the human-readable tier name.
func (t LayoutTier) String() string {
	switch t {
	case LayoutCompact:
		return "compact"
	case LayoutStandard:
		return "standard"
	case LayoutWide:
		return "wide"
	case LayoutUltra:
		return "ultra"
	default:
		return "unknown"
	}
}

// ---------------------------------------------------------------------------
// layoutDims
// ---------------------------------------------------------------------------

// layoutDims holds the pre-computed panel dimensions for the current terminal
// size.  It is recomputed on every WindowSizeMsg and passed to the rendering
// helpers so the View method stays free of arithmetic.
type layoutDims struct {
	// leftWidth and rightWidth are the inner content widths (without borders).
	leftWidth  int
	rightWidth int

	// contentHeight is the number of rows available for both panels after
	// subtracting banner, tab bar, and status line heights.
	contentHeight int

	// showRightPanel is false when the terminal is too narrow to display both
	// panels side-by-side.
	showRightPanel bool

	// tier identifies the responsive breakpoint tier computed from the current
	// terminal width.
	tier LayoutTier
}

// ---------------------------------------------------------------------------
// Layout computation
// ---------------------------------------------------------------------------

// computeLayout calculates panel dimensions from the current terminal size.
//
// Responsive breakpoints:
//   - width < 80   → LayoutCompact:  single-column (right panel hidden)
//   - width 80–99  → LayoutStandard: left 75%, right 25%
//   - width 100–119 → LayoutStandard: left 70%, right 30%
//   - width 120–179 → LayoutWide:    left 60%, right 40%
//   - width >= 180  → LayoutUltra:   left 50%, right 50%
//
// Border frame (1 char per edge = 2 per axis) is subtracted from each panel
// inner width so that the borders do not overflow the terminal width.
func (m AppModel) computeLayout() layoutDims {
	dims := layoutDims{}

	// Content rows available after chrome.
	providerTabH := 0
	if m.shared != nil && m.shared.providerTabBar != nil {
		providerTabH = m.shared.providerTabBar.Height()
	}
	taskBoardH := 0
	if m.shared != nil && m.shared.taskBoard != nil {
		taskBoardH = m.shared.taskBoard.Height()
	}
	hintH := 0
	if m.shared != nil && m.shared.hintBar != nil && m.shared.hintBar.IsVisible() {
		hintH = hintBarHeight
	}
	bcH := 0
	if m.shared != nil && m.shared.breadcrumb != nil {
		// The breadcrumb row is only allocated when there are crumbs to show.
		// View() returns "" when empty; we still subtract the row to avoid a
		// layout reflow on the first navigation event.  Components that set
		// crumbs on startup ensure the row is always present.
		bcH = breadcrumbHeight
	}
	dims.contentHeight = m.height - bannerHeight - tabBarHeight - providerTabH - statusLineHeight - taskBoardH - hintH - bcH - borderFrame
	if dims.contentHeight < 1 {
		dims.contentHeight = 1
	}

	// Determine the responsive tier from terminal width.
	var tier LayoutTier
	switch {
	case m.width < 80:
		tier = LayoutCompact
	case m.width < 120:
		tier = LayoutStandard
	case m.width < 180:
		tier = LayoutWide
	default:
		tier = LayoutUltra
	}
	dims.tier = tier

	if tier == LayoutCompact {
		// Narrow: single column, right panel hidden.
		dims.showRightPanel = false
		dims.leftWidth = m.width - borderFrame
		if dims.leftWidth < 1 {
			dims.leftWidth = 1
		}
		return dims
	}

	dims.showRightPanel = true

	// Per-tier left-panel ratio.
	var leftRatio float64
	switch tier {
	case LayoutStandard:
		// Preserve exact pre-TUI-058 sub-breakpoints within Standard.
		if m.width < 100 {
			leftRatio = 0.75
		} else {
			leftRatio = 0.70
		}
	case LayoutWide:
		leftRatio = 0.60
	case LayoutUltra:
		leftRatio = 0.50
	}

	// Compute outer column widths, then subtract border frame for inner.
	leftOuter := int(float64(m.width) * leftRatio)
	rightOuter := m.width - leftOuter

	dims.leftWidth = leftOuter - borderFrame
	dims.rightWidth = rightOuter - borderFrame

	if dims.leftWidth < 1 {
		dims.leftWidth = 1
	}
	if dims.rightWidth < 1 {
		dims.rightWidth = 1
	}

	return dims
}

// computeDrawerLayout returns the height and width for the drawer rendered
// at the bottom of the right panel.
//
// Rules:
//   - Compact tier: always (0, 0) — drawers suppressed
//   - nil drawerStack: (0, 0)
//   - Expanded (has content): 30% of contentHeight, capped at 12, minimum 5
//   - Minimized (no content): 2-row tab strip at right panel width
func (m AppModel) computeDrawerLayout(dims layoutDims) (height, width int) {
	if dims.tier == LayoutCompact || m.shared == nil || m.shared.drawerStack == nil {
		return 0, 0
	}
	drawerWidth := dims.rightWidth
	// Each minimized drawer = 3 rows (border top + label + border bottom).
	// 3 drawers all minimized = 9 rows minimum.
	const numDrawers = 3
	const minimizedH = 3
	minH := numDrawers * minimizedH

	hasContent := m.shared.drawerStack.OptionsHasContent() || m.shared.drawerStack.PlanHasContent() || m.shared.drawerStack.TeamsHasContent()
	if hasContent {
		h := dims.contentHeight * 40 / 100
		if h < minH {
			h = minH
		}
		if h > dims.contentHeight-5 {
			h = dims.contentHeight - 5
		}
		if h < minH {
			h = minH
		}
		return h, drawerWidth
	}
	// All minimized.
	h := minH
	if h > dims.contentHeight-1 {
		h = dims.contentHeight - 1
	}
	if h < 1 {
		h = 1
	}
	return h, drawerWidth
}

// ---------------------------------------------------------------------------
// Layout rendering
// ---------------------------------------------------------------------------

// renderLayout composes the full Lipgloss layout.
//
// Structure (top to bottom):
//
//	Banner     (3 rows, full width)
//	TabBar     (1 row, full width)
//	Main area  (left + optional right panel)
//	StatusLine (2 rows, full width)
//
// When a modal is active the layout is rendered as normal and then replaced by
// the modal overlay via lipgloss.Place so the modal appears centered on screen.
func (m AppModel) renderLayout() string {
	// Modal overlay takes full precedence: render and return immediately.
	if m.shared != nil && m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		m.shared.modalQueue.SetTermSize(m.width, m.height)
		return m.shared.modalQueue.View()
	}

	// Plan view modal renders as a full-screen overlay (lower priority than
	// ModalQueue but higher than the normal layout).
	if m.shared != nil && m.shared.planViewModal.IsActive() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.shared.planViewModal.View(),
		)
	}

	// CWD selector renders as a full-screen overlay (lower priority than
	// plan view, higher than search overlay and normal content).
	if m.shared != nil && m.shared.cwdSelector != nil && m.shared.cwdSelector.IsActive() {
		return m.shared.cwdSelector.View()
	}

	// Search overlay renders over the full layout (lower priority than modals
	// and plan view, higher than normal content).
	// If a modal or plan view opened while the search overlay was active,
	// deactivate the overlay so z-ordering remains correct.
	if m.shared != nil && m.shared.searchOverlay != nil {
		if m.shared.searchOverlay.IsActive() {
			return m.shared.searchOverlay.View()
		}
	}

	dims := m.computeLayout()

	bannerView := m.banner.View()

	var tabBarView string
	if m.tabBar != nil {
		tabBarView = m.tabBar.View()
	}

	statusLineView := m.statusLine.View()

	mainArea := m.renderMain(dims)

	parts := []string{bannerView, tabBarView}

	// Insert provider tab bar between the tab bar and main content area.
	if m.shared != nil && m.shared.providerTabBar != nil && m.shared.providerTabBar.IsVisible() {
		parts = append(parts, m.shared.providerTabBar.View())
	}

	// Insert breadcrumb trail below the provider tab bar and above the main
	// content area (TUI-063).  View() returns "" when no crumbs are set, in
	// which case no empty row is emitted.
	if m.shared != nil && m.shared.breadcrumb != nil {
		if view := m.shared.breadcrumb.View(); view != "" {
			parts = append(parts, view)
		}
	}

	parts = append(parts, mainArea)

	// Task board renders between main area and toast/status line.
	// Its height is already subtracted from contentHeight in computeLayout,
	// so the main area (chat + agents) shrinks to accommodate it while
	// the banner, tab bar, and status line remain unaffected.
	if m.shared != nil && m.shared.taskBoard != nil && m.shared.taskBoard.IsVisible() {
		parts = append(parts, m.shared.taskBoard.View())
	}

	// Toast notifications render between main area and status line.
	if m.shared != nil && m.shared.toasts != nil && !m.shared.toasts.IsEmpty() {
		parts = append(parts, m.shared.toasts.View())
	}

	// Hint bar renders between toasts and status line (TUI-060).
	if m.shared != nil && m.shared.hintBar != nil && m.shared.hintBar.IsVisible() {
		parts = append(parts, m.shared.hintBar.View())
	}

	parts = append(parts, statusLineView)

	output := lipgloss.JoinVertical(lipgloss.Top, parts...)

	// Final safety net: clip the composed layout to the terminal height.
	// This catches any overflow from toasts (whose height is not subtracted
	// from contentHeight), phantom blank lines from JoinVertical with "",
	// or stale component dimensions after a missed propagateContentSizes call.
	return truncateHeight(output, m.height)
}

// renderMain renders the split content area (left panel + optional right panel).
// The left panel uses the full content height.  The right panel handles its
// drawer allocation internally in renderRightPanel (TUI-002).
func (m AppModel) renderMain(dims layoutDims) string {
	panelH := dims.contentHeight

	leftPanel := m.renderLeftPanel(dims, panelH)

	if !dims.showRightPanel {
		return leftPanel
	}
	rightPanel := m.renderRightPanel(dims, panelH)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// renderHorizontalSeparator renders a single-row horizontal rule at the given
// width using the muted style.  It visually separates the conversation viewport
// from the input box in the left (Claude) panel.
func renderHorizontalSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	return config.StyleMuted.Render(strings.Repeat("─", width))
}

// renderLeftPanel renders the left panel content with the appropriate focus
// border.  The content depends on the active tab:
//   - TabChat       → Claude conversation panel
//   - TabTeamConfig → Team list
//   - TabAgentConfig, TabTelemetry → placeholder
//
// panelH is the content height after subtracting the drawer allocation.
func (m AppModel) renderLeftPanel(dims layoutDims, panelH int) string {
	focused := m.focus == FocusClaude

	var content string
	switch m.activeTab {
	case TabChat:
		if m.shared != nil && m.shared.claudePanel != nil {
			conversation := m.shared.claudePanel.ViewConversation()
			separator := renderHorizontalSeparator(dims.leftWidth)
			input := m.shared.claudePanel.ViewInput()
			composed := lipgloss.JoinVertical(lipgloss.Left, conversation, separator, input)
			content = m.shared.claudePanel.ApplyOverlay(composed)
		} else {
			content = config.StyleSubtle.Render("Claude panel  [focus=" + m.focus.String() + "]")
		}
	case TabTeamConfig:
		if m.shared != nil && m.shared.teamList != nil {
			content = m.shared.teamList.View()
		} else {
			content = config.StyleSubtle.Render("Team Config — no teams loaded")
		}
	case TabAgentConfig:
		content = config.StyleSubtle.Render("Agent Config (not yet implemented)")
	case TabTelemetry:
		content = config.StyleSubtle.Render("Telemetry → right panel (use Tab to focus)")
	default:
		content = config.StyleSubtle.Render(m.activeTab.String())
	}

	// Clip inner content — lipgloss.Height() pads but does NOT truncate.
	content = truncateHeight(content, panelH)

	var style lipgloss.Style
	if focused {
		style = config.StyleFocusedBorder
	} else {
		style = config.StyleUnfocusedBorder
	}

	return style.
		Width(dims.leftWidth).
		Height(panelH).
		Render(content)
}

// renderRightPanel renders the right-side panel whose content depends on the
// active RightPanelMode.  panelH is the total column height; the drawer
// allocation is computed internally and the remaining space goes to content.
func (m AppModel) renderRightPanel(dims layoutDims, panelH int) string {
	focused := m.focus == FocusAgents

	// Compute drawer allocation within the right panel; content gets the rest.
	drawerH, drawerW := m.computeDrawerLayout(dims)
	contentH := panelH - drawerH
	if contentH < 1 {
		contentH = 1
	}

	var content string
	switch m.rightPanelMode {
	case RPMAgents:
		treeView := m.agentTree.View()
		detailView := m.agentDetail.View()
		content = lipgloss.JoinVertical(lipgloss.Left, treeView, detailView)
	case RPMDashboard:
		if m.shared != nil && m.shared.dashboard != nil {
			content = m.shared.dashboard.View()
		} else {
			content = config.StyleSubtle.Render("Dashboard")
		}
	case RPMSettings:
		if m.shared != nil && m.shared.settings != nil {
			content = m.shared.settings.View()
		} else {
			content = config.StyleSubtle.Render("Settings")
		}
	case RPMTelemetry:
		if m.shared != nil && m.shared.telemetry != nil {
			content = m.shared.telemetry.View()
		} else {
			content = config.StyleSubtle.Render("Telemetry")
		}
	case RPMPlanPreview:
		if m.shared != nil && m.shared.planPreview != nil {
			content = m.shared.planPreview.View()
		} else {
			content = config.StyleSubtle.Render("Plan Preview")
		}
	default:
		content = config.StyleSubtle.Render(m.rightPanelMode.String())
	}

	// Clip inner content, then pad to exactly contentH rows so the drawer
	// is flush with the bottom of the panel (not top-justified).
	content = truncateHeight(content, contentH)
	content = lipgloss.NewStyle().Width(dims.rightWidth).Height(contentH).Render(content)

	// Drawer renders INSIDE the panel border at the bottom of the right column.
	if drawerH > 0 && m.shared != nil && m.shared.drawerStack != nil {
		m.shared.drawerStack.SetSize(drawerW, drawerH)
		drawerView := m.shared.drawerStack.View()
		content = lipgloss.JoinVertical(lipgloss.Left, content, drawerView)
	}

	// Final clip to panelH to prevent any overflow from drawer chrome.
	content = truncateHeight(content, panelH)

	var style lipgloss.Style
	if focused {
		style = config.StyleFocusedBorder
	} else {
		style = config.StyleUnfocusedBorder
	}

	return style.
		Width(dims.rightWidth).
		Height(panelH).
		Render(content)
}
