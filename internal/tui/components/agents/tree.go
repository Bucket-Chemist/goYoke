// Package agents implements the agent tree view and detail components for the
// GOgent-Fortress TUI. It has no dependency on the model package; it imports
// only state and config to keep the import graph acyclic.
package agents

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// AgentSelectedMsg is emitted by AgentTreeModel when the user moves the
// cursor to a different agent (Up/Down navigation) or explicitly selects one
// (Enter). The parent model should update AgentDetailModel in response.
type AgentSelectedMsg struct {
	AgentID string
}

// TreePulseTickMsg is emitted by the lazy pulse ticker to toggle icon brightness
// for running agents. The parent model must forward it to AgentTreeModel.Update.
type TreePulseTickMsg struct{}

// AgentDetailFocusMsg is emitted when the user presses Enter on a tree node.
// The parent model should transfer keyboard focus from the tree to the detail panel.
type AgentDetailFocusMsg struct {
	AgentID string
}

// AgentTreeFocusMsg is emitted when the user presses Escape in the detail panel.
// The parent model should transfer keyboard focus back to the tree.
type AgentTreeFocusMsg struct{}

// ---------------------------------------------------------------------------
// Status styling helpers
// ---------------------------------------------------------------------------

// statusIcon returns the ASCII icon rune for the given agent status.
func statusIcon(s state.AgentStatus) rune {
	switch s {
	case state.StatusRunning:
		return config.IconRunning
	case state.StatusComplete:
		return config.IconComplete
	case state.StatusError:
		return config.IconError
	case state.StatusKilled:
		return config.IconCancelled
	default: // StatusPending and any unknown
		return config.IconPending
	}
}

// statusStyle returns the lipgloss.Style used to render the icon for the
// given agent status.
func statusStyle(s state.AgentStatus) lipgloss.Style {
	switch s {
	case state.StatusRunning:
		return lipgloss.NewStyle().Foreground(config.ColorWarning)
	case state.StatusComplete:
		return lipgloss.NewStyle().Foreground(config.ColorSuccess)
	case state.StatusError:
		return lipgloss.NewStyle().Foreground(config.ColorError)
	case state.StatusKilled:
		return lipgloss.NewStyle().Foreground(config.ColorError)
	default:
		return lipgloss.NewStyle().Foreground(config.ColorMuted)
	}
}

// StatusRowStyle returns the lipgloss.Style used to render the ENTIRE row for
// the given agent status. Unlike statusStyle (which colors only the status
// icon), this style is applied to the complete row text — indent, icon, label,
// dot leaders, and value — in full-mode tree rendering.
//
// Color scheme:
//   - Running:  dim green (success color, no bold — less emphasis than Complete)
//   - Complete: bright green bold (success color + bold)
//   - Error:    red
//   - Killed:   yellow strikethrough
//   - Pending:  muted (grey)
func StatusRowStyle(s state.AgentStatus) lipgloss.Style {
	switch s {
	case state.StatusRunning:
		return lipgloss.NewStyle().Foreground(config.ColorSuccess)
	case state.StatusComplete:
		return lipgloss.NewStyle().Foreground(config.ColorSuccess).Bold(true)
	case state.StatusError:
		return lipgloss.NewStyle().Foreground(config.ColorError)
	case state.StatusKilled:
		return lipgloss.NewStyle().Foreground(config.ColorWarning).Strikethrough(true)
	default: // StatusPending and any unknown
		return lipgloss.NewStyle().Foreground(config.ColorMuted)
	}
}

// ---------------------------------------------------------------------------
// TreeDensity
// ---------------------------------------------------------------------------

// TreeDensity controls the information density of AgentTreeModel rendering.
type TreeDensity int

const (
	// DensityStandard is the default two-column dot-leader layout (existing behaviour).
	DensityStandard TreeDensity = iota
	// DensityCompact renders each agent as a single compact line: tree prefix + icon + 2-char abbreviation.
	DensityCompact
	// DensityVerbose extends each node with a second metadata line showing status, tier, duration, and cost.
	DensityVerbose
)

// densityCount is the total number of TreeDensity values used for modular cycling.
const densityCount = 3

// ---------------------------------------------------------------------------
// AgentTreeModel
// ---------------------------------------------------------------------------

// AgentTreeModel is the Bubbletea sub-model for the agent hierarchy tree.
// It maintains a flat DFS-ordered list of tree nodes (pre-cached by the
// parent model via SetNodes) and handles keyboard navigation.
//
// The zero value is not usable; use NewAgentTreeModel instead.
type AgentTreeModel struct {
	treeNodes    []*state.AgentTreeNode
	selectedIdx  int
	focused      bool
	width        int
	height       int
	scrollOffset int
	density      TreeDensity
	// reduceMotion suppresses the pulse animation (WCAG 2.3.1).
	// When true, running agents always show a static bright icon.
	reduceMotion bool
	// pulseBright tracks the current pulse phase for running-agent icons.
	// Toggled every 500 ms by TreePulseTickMsg when at least one agent is running.
	pulseBright bool
	// tickRunning is true while a SchedulePulseTick Cmd is in flight.
	// Prevents duplicate tick goroutines when SetNodes is called repeatedly.
	tickRunning bool
}

// NewAgentTreeModel returns an AgentTreeModel ready for use.
func NewAgentTreeModel() AgentTreeModel {
	return AgentTreeModel{}
}

// SetNodes replaces the cached tree data. This must be called from the parent
// model's Update method — never from View.
func (m *AgentTreeModel) SetNodes(nodes []*state.AgentTreeNode) {
	m.treeNodes = nodes
	// Clamp selectedIdx in case the tree shrank.
	if len(nodes) == 0 {
		m.selectedIdx = 0
		m.scrollOffset = 0
		return
	}
	if m.selectedIdx >= len(nodes) {
		m.selectedIdx = len(nodes) - 1
	}
	m.clampScroll()
}

// SetFocused sets whether the tree has keyboard focus.
func (m *AgentTreeModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize updates the viewport dimensions for responsive rendering and scroll
// boundary calculations.
func (m *AgentTreeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.clampScroll()
}

// Density returns the current tree density setting.
func (m AgentTreeModel) Density() TreeDensity {
	return m.density
}

// SetDensity sets the tree density to the given value.
func (m *AgentTreeModel) SetDensity(d TreeDensity) {
	m.density = d
}

// CycleDensity advances through Standard → Compact → Verbose → Standard.
func (m *AgentTreeModel) CycleDensity() {
	m.density = TreeDensity((int(m.density) + 1) % densityCount)
}

// SetReduceMotion controls whether the pulse animation is suppressed.
// When true, running agents show a static bright icon instead of pulsing.
func (m *AgentTreeModel) SetReduceMotion(v bool) {
	m.reduceMotion = v
}

// hasRunningAgents reports whether any node in the current tree has
// StatusRunning. Used to drive the lazy pulse tick scheduler.
func (m AgentTreeModel) hasRunningAgents() bool {
	for _, node := range m.treeNodes {
		if node.Agent.Status == state.StatusRunning {
			return true
		}
	}
	return false
}

// SchedulePulseTick returns a Cmd that fires TreePulseTickMsg after 500 ms.
// The caller should schedule this at startup and whenever a running agent
// first appears. AgentTreeModel re-schedules lazily from within Update.
func SchedulePulseTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return TreePulseTickMsg{}
	})
}

// MaybeStartPulseTick returns a SchedulePulseTick Cmd when there are running
// agents and no tick is currently in flight. Returns nil otherwise.
//
// Call this after every SetNodes invocation so the tick starts automatically
// when the first running agent appears and stops when all agents are idle.
func (m *AgentTreeModel) MaybeStartPulseTick() tea.Cmd {
	if !m.tickRunning && m.hasRunningAgents() {
		m.tickRunning = true
		return SchedulePulseTick()
	}
	return nil
}

// PulseBright returns the current pulse phase for running-agent icons.
// True means the icon is in the bright phase; false means the dim phase.
// This accessor exists primarily for testing.
func (m AgentTreeModel) PulseBright() bool {
	return m.pulseBright
}

// SelectedID returns the agent ID of the currently highlighted tree node, or
// an empty string when the tree is empty.
func (m AgentTreeModel) SelectedID() string {
	if len(m.treeNodes) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.treeNodes) {
		return ""
	}
	return m.treeNodes[m.selectedIdx].Agent.ID
}

// TreeNodes returns the cached flat tree node slice. The returned slice is
// the same backing array used internally; callers must not modify it.
// This accessor is intended for testing and should not be used in View().
func (m AgentTreeModel) TreeNodes() []*state.AgentTreeNode {
	return m.treeNodes
}

// clampScroll adjusts scrollOffset so that selectedIdx is always visible
// within the viewport.
func (m *AgentTreeModel) clampScroll() {
	if m.height <= 0 {
		return
	}
	if m.selectedIdx < m.scrollOffset {
		m.scrollOffset = m.selectedIdx
	}
	if m.selectedIdx >= m.scrollOffset+m.height {
		m.scrollOffset = m.selectedIdx - m.height + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// Init implements tea.Model. The tree view requires no startup commands.
func (m AgentTreeModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Handles TreePulseTickMsg (regardless of focus)
// and, when focused, Up/Down navigation and Enter to emit AgentSelectedMsg.
func (m AgentTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle pulse tick regardless of focus state.
	if _, ok := msg.(TreePulseTickMsg); ok {
		m.pulseBright = !m.pulseBright
		// Reschedule only while at least one agent is still running.
		// This is the lazy-tick invariant: no CPU cost when all agents are idle.
		if m.hasRunningAgents() {
			return m, SchedulePulseTick()
		}
		m.tickRunning = false
		return m, nil
	}

	if !m.focused {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.clampScroll()
			return m, emitSelected(m.SelectedID())
		}

	case "down", "j":
		if m.selectedIdx < len(m.treeNodes)-1 {
			m.selectedIdx++
			m.clampScroll()
			return m, emitSelected(m.SelectedID())
		}

	case "enter":
		id := m.SelectedID()
		if id != "" {
			// Emit both: select the agent AND focus the detail panel.
			return m, tea.Batch(
				emitSelected(id),
				func() tea.Msg { return AgentDetailFocusMsg{AgentID: id} },
			)
		}
	}

	return m, nil
}

// emitSelected returns a Cmd that emits AgentSelectedMsg for the given ID.
func emitSelected(id string) tea.Cmd {
	return func() tea.Msg {
		return AgentSelectedMsg{AgentID: id}
	}
}

// pulseIconStyle returns the lipgloss.Style for a running agent's status icon,
// applying the current pulse phase. Non-running statuses should use statusStyle.
//
// When reduceMotion is true, always returns the bright (static) style.
// When pulseBright is true (or reduceMotion), returns a bold warning style.
// When pulseBright is false, returns a muted style (dim phase).
func (m AgentTreeModel) pulseIconStyle() lipgloss.Style {
	if m.reduceMotion || m.pulseBright {
		return lipgloss.NewStyle().Foreground(config.ColorWarning).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(config.ColorMuted)
}

// ---------------------------------------------------------------------------
// RenderMode
// ---------------------------------------------------------------------------

// RenderMode controls how AgentTreeModel renders its content.
type RenderMode int

const (
	// RenderFull renders the complete tree with labels and activity preview.
	RenderFull RenderMode = iota
	// RenderIconRail renders a compact icon + 2-char abbreviation + cost/status
	// view for narrow right panels (< 28 columns).
	RenderIconRail
)

// Render renders the tree using the specified mode and available width.
// RenderIconRail renders the compact icon rail for narrow panels.
// RenderFull dispatches based on the current density setting:
//   - DensityStandard: existing dot-leader layout (delegates to View())
//   - DensityCompact:  single line per agent — tree prefix + icon + 2-char abbreviation
//   - DensityVerbose:  two lines per agent — standard row + metadata line
func (m AgentTreeModel) Render(mode RenderMode, width int) string {
	if mode == RenderIconRail {
		return m.renderIconRail(width)
	}
	switch m.density {
	case DensityCompact:
		return m.renderCompactDensity()
	case DensityVerbose:
		return m.renderVerboseDensity()
	default: // DensityStandard
		return m.View()
	}
}

// renderIconRail renders the agent tree as a compact icon rail.
// Each visible line: {treePrefix}{statusIcon} {abbrev} {costOrStatus}
func (m AgentTreeModel) renderIconRail(width int) string {
	if len(m.treeNodes) == 0 {
		return config.StyleMuted.Render("No agents")
	}

	start := m.scrollOffset
	end := start + m.height
	if end > len(m.treeNodes) || m.height <= 0 {
		end = len(m.treeNodes)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		node := m.treeNodes[i]
		line := m.renderIconNode(node, i == m.selectedIdx, width)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderIconNode renders a single tree node in icon rail mode.
// Format: {prefix}{icon} {abbrev} {value}
func (m AgentTreeModel) renderIconNode(node *state.AgentTreeNode, selected bool, width int) string {
	// Build tree connector prefix (same structure as full mode).
	var prefixBuf strings.Builder
	if node.Depth > 0 {
		for range node.Depth - 1 {
			prefixBuf.WriteString("│ ")
		}
		if node.IsLast {
			prefixBuf.WriteString("└─")
		} else {
			prefixBuf.WriteString("├─")
		}
	}
	prefixStr := prefixBuf.String()
	prefixW := lipgloss.Width(prefixStr)

	// Status icon.
	icon := string(statusIcon(node.Agent.Status))
	var iconStr string
	if node.Agent.Status == state.StatusRunning {
		iconStr = m.pulseIconStyle().Render(icon)
	} else {
		iconStr = statusStyle(node.Agent.Status).Render(icon)
	}

	// 2-char uppercase abbreviation.
	abbrev := agentAbbrev(node.Agent.AgentType)

	// Cost or status value — truncated to remaining available width.
	// Fixed overhead: prefix + icon(1) + space(1) + abbrev(2) + space(1) = prefixW + 5
	value := iconRailValue(node.Agent)
	available := width - prefixW - 5
	if available < 0 {
		available = 0
	}
	value = util.Truncate(value, available)

	if selected {
		plain := prefixStr + icon + " " + abbrev + " " + value
		return config.StyleHighlight.Render(plain)
	}
	return prefixStr + iconStr + " " + abbrev + " " + value
}

// agentAbbrev returns a 2-char uppercase abbreviation for the given agent type.
// Uses the first 2 runes of agentType, uppercased. Single-char types are padded
// with a trailing space; empty types return two spaces.
func agentAbbrev(agentType string) string {
	runes := []rune(strings.ToUpper(agentType))
	switch len(runes) {
	case 0:
		return "  "
	case 1:
		return string(runes[0]) + " "
	default:
		return string(runes[:2])
	}
}

// iconRailValue returns the compact value string shown in icon rail mode.
// Displays cost if non-zero; otherwise a short status word.
func iconRailValue(a *state.Agent) string {
	if a.Cost > 0 {
		return fmt.Sprintf("$%.2f", a.Cost)
	}
	switch a.Status {
	case state.StatusRunning:
		return "run"
	case state.StatusComplete:
		return "done"
	case state.StatusError:
		return "fail"
	case state.StatusKilled:
		return "kill"
	default:
		return "wait"
	}
}

// View implements tea.Model. It renders the agent hierarchy as a scrollable
// tree with dot-leader rows. Each row uses indentation (2 spaces per depth
// level) rather than box-drawing connectors. The view is a pure function of
// the model state — no I/O is performed here.
func (m AgentTreeModel) View() string {
	if len(m.treeNodes) == 0 {
		return config.StyleMuted.Render("No agents")
	}

	// Determine the visible window.
	start := m.scrollOffset
	end := start + m.height
	if end > len(m.treeNodes) || m.height <= 0 {
		end = len(m.treeNodes)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		node := m.treeNodes[i]
		line := m.renderNode(node, i == m.selectedIdx)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Trim the trailing newline added by the last iteration.
	out := strings.TrimRight(sb.String(), "\n")
	return out
}

// Search implements state.SearchSource for the agent tree.
//
// It performs a case-insensitive substring search across agent type and
// description fields and returns matching state.SearchResult values.
// An empty query always returns nil.
func (m *AgentTreeModel) Search(query string) []state.SearchResult {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var results []state.SearchResult
	for _, node := range m.treeNodes {
		agent := node.Agent
		nameLower := strings.ToLower(agent.AgentType)
		descLower := strings.ToLower(agent.Description)
		nameMatch := strings.Contains(nameLower, q)
		descMatch := strings.Contains(descLower, q)
		if !nameMatch && !descMatch {
			continue
		}
		score := 100
		if nameMatch {
			score = 150
		}
		results = append(results, state.SearchResult{
			Source: "agents",
			Label:  agent.AgentType,
			Detail: agent.Description,
			Score:  score,
		})
	}
	return results
}

// buildTreeValue returns the plain-text right-column value for a full-mode
// tree row. It shows AC progress (if any) followed by a cost amount or short
// status word.
func buildTreeValue(a *state.Agent) string {
	var parts []string

	if len(a.AcceptanceCriteria) > 0 {
		done := 0
		for _, ac := range a.AcceptanceCriteria {
			if ac.Completed {
				done++
			}
		}
		parts = append(parts, fmt.Sprintf("%d/%d AC", done, len(a.AcceptanceCriteria)))
	}

	if a.Cost > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", a.Cost))
	} else {
		switch a.Status {
		case state.StatusRunning:
			parts = append(parts, "run")
		case state.StatusComplete:
			parts = append(parts, "done")
		case state.StatusError:
			parts = append(parts, "fail")
		case state.StatusKilled:
			parts = append(parts, "kill")
		default:
			parts = append(parts, "wait")
		}
	}

	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// DensityCompact rendering
// ---------------------------------------------------------------------------

// renderCompactDensity renders the tree in compact density mode.
// Each agent occupies a single line: {treePrefix}{icon} {abbrev}
// No dot-leaders and no right-aligned value column.
func (m AgentTreeModel) renderCompactDensity() string {
	if len(m.treeNodes) == 0 {
		return config.StyleMuted.Render("No agents")
	}

	start := m.scrollOffset
	end := start + m.height
	if end > len(m.treeNodes) || m.height <= 0 {
		end = len(m.treeNodes)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		node := m.treeNodes[i]
		line := m.renderCompactNode(node, i == m.selectedIdx)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderCompactNode renders a single node in compact density mode.
// Format: {treePrefix}{icon} {abbrev}
func (m AgentTreeModel) renderCompactNode(node *state.AgentTreeNode, selected bool) string {
	a := node.Agent

	// Build tree connector prefix identical to icon rail mode.
	var prefixBuf strings.Builder
	if node.Depth > 0 {
		for range node.Depth - 1 {
			prefixBuf.WriteString("│ ")
		}
		if node.IsLast {
			prefixBuf.WriteString("└─")
		} else {
			prefixBuf.WriteString("├─")
		}
	}
	prefixStr := prefixBuf.String()

	icon := string(statusIcon(a.Status))
	abbrev := agentAbbrev(a.AgentType)

	if selected {
		plain := prefixStr + icon + " " + abbrev
		return config.StyleHighlight.Render(plain)
	}
	var iconStr string
	if a.Status == state.StatusRunning {
		iconStr = m.pulseIconStyle().Render(icon)
	} else {
		iconStr = statusStyle(a.Status).Render(icon)
	}
	return prefixStr + iconStr + " " + abbrev
}

// ---------------------------------------------------------------------------
// DensityVerbose rendering
// ---------------------------------------------------------------------------

// renderVerboseDensity renders the tree in verbose density mode.
// Each agent occupies two lines:
//   - Line 1: standard dot-leader row (identical to DensityStandard)
//   - Line 2: indented metadata — status | tier | duration | cost
func (m AgentTreeModel) renderVerboseDensity() string {
	if len(m.treeNodes) == 0 {
		return config.StyleMuted.Render("No agents")
	}

	start := m.scrollOffset
	end := start + m.height
	if end > len(m.treeNodes) || m.height <= 0 {
		end = len(m.treeNodes)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		node := m.treeNodes[i]
		// Line 1: standard dot-leader row.
		sb.WriteString(m.renderNode(node, i == m.selectedIdx))
		sb.WriteByte('\n')
		// Line 2: metadata.
		sb.WriteString(m.renderVerboseMeta(node))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderVerboseMeta renders the second metadata line for verbose density mode.
// Format: {indent}  {status} | {tier} | {duration} | {cost}
func (m AgentTreeModel) renderVerboseMeta(node *state.AgentTreeNode) string {
	a := node.Agent

	// Indent matches the node depth (2 spaces per level) plus 2 for the icon column.
	indent := strings.Repeat("  ", node.Depth) + "  "

	tier := a.Tier
	if tier == "" {
		tier = "-"
	}

	dur := formatAgentDuration(a)

	costStr := "-"
	if a.Cost > 0 {
		costStr = fmt.Sprintf("$%.2f", a.Cost)
	}

	meta := fmt.Sprintf("%s%s | %s | %s | %s", indent, a.Status.String(), tier, dur, costStr)

	// Truncate to available width so long metadata doesn't wrap.
	if lipgloss.Width(meta) > m.width {
		meta = util.Truncate(meta, m.width)
	}

	return config.StyleMuted.Render(meta)
}

// ---------------------------------------------------------------------------
// Standard (dot-leader) rendering
// ---------------------------------------------------------------------------

// renderNode renders a single tree row using the dot-leader layout.
//
// Format: {indent}{icon} {label} {dots} {value}
//
//   - indent: 2 spaces × depth (no box-drawing characters)
//   - icon: status icon character
//   - label: agent type, truncated to fit available width
//   - dots: "." repeated to fill the gap between label and value
//   - value: right-aligned cost ($X.XX) or short status word, optionally
//     preceded by AC progress ("N/M AC")
//
// All width arithmetic uses lipgloss.Width for ANSI safety.
func (m AgentTreeModel) renderNode(node *state.AgentTreeNode, selected bool) string {
	a := node.Agent

	// Indent: 2 spaces per depth level.
	indent := strings.Repeat("  ", node.Depth)
	indentW := node.Depth * 2

	// Status icon (always 1 visual column wide).
	icon := string(statusIcon(a.Status))

	// Right column: plain-text value (no ANSI).
	rightPlain := buildTreeValue(a)
	rightW := lipgloss.Width(rightPlain)

	// Row layout (visual widths):
	//   indent(indentW) + icon(1) + " "(1) + label(labelW) + " "(1) + dots(dotsW) + " "(1) + right(rightW)
	//   = indentW + 4 + labelW + dotsW + rightW  =  m.width
	//
	// Reserve at least 1 dot column so the leader is always visible.
	maxLabelW := m.width - indentW - 4 - rightW - 1
	if maxLabelW < 1 {
		maxLabelW = 1
	}
	label := util.Truncate(a.AgentType, maxLabelW)
	labelW := lipgloss.Width(label)

	dotsW := m.width - indentW - 4 - labelW - rightW
	if dotsW < 1 {
		dotsW = 1
	}
	dots := strings.Repeat(".", dotsW)

	if selected {
		plain := indent + icon + " " + label + " " + dots + " " + rightPlain
		return config.StyleHighlight.Render(plain)
	}
	rowStyle := StatusRowStyle(a.Status)
	if a.Status == state.StatusRunning {
		// Render the icon with pulse styling; the rest of the row uses rowStyle.
		iconRendered := m.pulseIconStyle().Render(icon)
		rest := " " + label + " " + dots + " " + rightPlain
		return indent + iconRendered + rowStyle.Render(rest)
	}
	plain := indent + icon + " " + label + " " + dots + " " + rightPlain
	return rowStyle.Render(plain)
}

