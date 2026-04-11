// Package agents implements the agent tree view and detail components for the
// GOgent-Fortress TUI. It has no dependency on the model package; it imports
// only state and config to keep the import graph acyclic.
package agents

import (
	"fmt"
	"strings"

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

// Update implements tea.Model. When focused, Up/Down navigate the tree and
// Enter emits AgentSelectedMsg.
func (m AgentTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
// RenderFull delegates to View() so existing behaviour is unchanged.
// RenderIconRail renders the compact icon rail for narrow panels.
func (m AgentTreeModel) Render(mode RenderMode, width int) string {
	if mode == RenderIconRail {
		return m.renderIconRail(width)
	}
	return m.View()
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
	iconStr := statusStyle(node.Agent.Status).Render(icon)

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
	plain := indent + icon + " " + label + " " + dots + " " + rightPlain
	return rowStyle.Render(plain)
}

