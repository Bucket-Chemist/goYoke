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
			return m, emitSelected(id)
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

// View implements tea.Model. It renders the agent hierarchy as a scrollable
// tree using Unicode box-drawing connectors. The view is a pure function of
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

// renderNode renders a single tree row. selected indicates whether this row is
// the currently highlighted one.
func (m AgentTreeModel) renderNode(node *state.AgentTreeNode, selected bool) string {
	// ------------------------------------------------------------------
	// Build indent + connector prefix
	// ------------------------------------------------------------------
	// For each ancestor level (depth-1 levels of parent indentation) we
	// draw "│ " (two cells). At the node's own depth we draw "└─" or "├─".
	var prefix strings.Builder
	if node.Depth > 0 {
		for range node.Depth - 1 {
			prefix.WriteString("│ ")
		}
		if node.IsLast {
			prefix.WriteString("└─")
		} else {
			prefix.WriteString("├─")
		}
	}

	// ------------------------------------------------------------------
	// Status icon
	// ------------------------------------------------------------------
	icon := string(statusIcon(node.Agent.Status))
	iconStr := statusStyle(node.Agent.Status).Render(icon)

	// ------------------------------------------------------------------
	// Main label: "{type}: {description}"
	// ------------------------------------------------------------------
	label := fmt.Sprintf("%s: %s", node.Agent.AgentType, node.Agent.Description)

	// ------------------------------------------------------------------
	// Activity preview (dimmed, appended in brackets)
	// ------------------------------------------------------------------
	var activityStr string
	if node.Agent.Activity != nil && node.Agent.Activity.Preview != "" {
		preview := truncate(node.Agent.Activity.Preview, 40)
		activityStr = " " + config.StyleMuted.Render("["+preview+"]")
	}

	// ------------------------------------------------------------------
	// Assemble full row, truncating to fit width
	// ------------------------------------------------------------------
	prefixStr := prefix.String()
	// Calculate available width for label (rough: subtract prefix, icon, spaces)
	overhead := len(prefixStr) + 2 // icon + one space
	available := m.width - overhead
	if available < 1 {
		available = 1
	}
	label = truncate(label, available)

	row := prefixStr + iconStr + " " + label + activityStr

	if selected {
		// Highlight the entire row.
		return config.StyleHighlight.Render(prefixStr+icon+" "+label) + activityStr
	}
	return row
}

// truncate shortens s to at most maxLen runes, appending "…" if it was cut.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
