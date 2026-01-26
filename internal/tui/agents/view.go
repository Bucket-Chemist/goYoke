package agents

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the tree view component for the agent delegation tree
type Model struct {
	tree   *AgentTree
	width  int
	height int

	// Selection and navigation
	selectedID   string
	cursorPos    int
	visibleNodes []string // Ordered list of visible node IDs

	// Display
	expanded     map[string]bool // Track expanded/collapsed nodes
	scrollOffset int

	// Styling
	styles Styles

	// Focus state
	focused bool
}

// Styles contains all lipgloss styles for the tree view
type Styles struct {
	Border          lipgloss.Style
	Title           lipgloss.Style
	StatusSpawning  lipgloss.Style
	StatusRunning   lipgloss.Style
	StatusCompleted lipgloss.Style
	StatusError     lipgloss.Style
	Selected        lipgloss.Style
	Normal          lipgloss.Style
	TreeLine        lipgloss.Style
	Empty           lipgloss.Style
}

// DefaultStyles creates the default style set for the tree view
func DefaultStyles() Styles {
	return Styles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),

		StatusSpawning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Gray

		StatusRunning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")), // Blue

		StatusCompleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")), // Green

		StatusError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red

		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Bold(true),

		Normal: lipgloss.NewStyle(),

		TreeLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		Empty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true),
	}
}

// New creates a new tree view model
func New(tree *AgentTree) Model {
	return Model{
		tree:     tree,
		expanded: make(map[string]bool),
		styles:   DefaultStyles(),
	}
}

// Init initializes the model (satisfies tea.Model)
func (m Model) Init() tea.Cmd {
	return nil
}

// AgentUpdateMsg notifies the tree view of tree changes
type AgentUpdateMsg struct {
	Tree *AgentTree
}

// SelectionMsg notifies listeners of agent selection
type SelectionMsg struct {
	AgentID string
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKey(msg)

	case AgentUpdateMsg:
		m.tree = msg.Tree
		m.rebuildVisibleNodes()
		return m, nil
	}

	return m, nil
}

// handleKey processes keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursorPos > 0 {
			m.cursorPos--
			// Update selected ID
			if m.cursorPos >= 0 && m.cursorPos < len(m.visibleNodes) {
				m.selectedID = m.visibleNodes[m.cursorPos]
			}
			// Ensure visible
			availableHeight := m.height - 8
			if availableHeight < 1 {
				availableHeight = 1
			}
			if m.cursorPos < m.scrollOffset {
				m.scrollOffset = m.cursorPos
			}
		}

	case "down", "j":
		maxPos := len(m.visibleNodes) - 1
		if m.cursorPos < maxPos {
			m.cursorPos++
			// Update selected ID
			if m.cursorPos >= 0 && m.cursorPos < len(m.visibleNodes) {
				m.selectedID = m.visibleNodes[m.cursorPos]
			}
			// Ensure visible
			availableHeight := m.height - 8
			if availableHeight < 1 {
				availableHeight = 1
			}
			if m.cursorPos >= m.scrollOffset+availableHeight {
				m.scrollOffset = m.cursorPos - availableHeight + 1
			}
		}

	case "enter":
		return m, m.selectNode()

	case " ", "space":
		if m.selectedID != "" {
			node, exists := m.tree.GetNode(m.selectedID)
			if exists && len(node.Children) > 0 {
				m.expanded[m.selectedID] = !m.expanded[m.selectedID]
				m.rebuildVisibleNodes()
			}
		}

	case "r":
		return m, m.refresh()
	}

	return m, nil
}

// View renders the tree view
func (m Model) View() string {
	if m.tree == nil || m.tree.Root == nil {
		empty := m.styles.Empty.Render("No agents running")
		return m.styles.Border.
			Width(m.width - 4).
			Height(m.height - 4).
			Render(empty)
	}

	// Rebuild visible nodes if needed
	if len(m.visibleNodes) == 0 {
		m.rebuildVisibleNodes()
	}

	// Render header
	var content strings.Builder
	title := m.styles.Title.Render("Agent Tree")
	stats := m.renderStats()
	content.WriteString(title)
	content.WriteString(" ")
	content.WriteString(stats)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", m.width-6))
	content.WriteString("\n")

	// Render tree
	tree := m.renderTree()
	content.WriteString(tree)

	return m.styles.Border.
		Width(m.width - 4).
		Height(m.height - 4).
		Render(content.String())
}

// renderStats renders the tree statistics
func (m Model) renderStats() string {
	stats := m.tree.GetStats()
	return m.styles.Normal.Render(
		fmt.Sprintf("(%d active, %d completed, %d errors)",
			stats.ActiveAgents,
			stats.CompletedAgents,
			stats.ErroredAgents,
		),
	)
}

// renderTree renders the hierarchical tree structure
func (m Model) renderTree() string {
	if m.tree.Root == nil {
		return m.styles.Empty.Render("No agents")
	}

	var lines []string

	// Build visible lines with proper context for tree structure
	m.tree.WalkTree(func(node *AgentNode) bool {
		depth := m.getDepth(node)
		isLast := m.isLastChild(node)

		line := m.renderNode(node, depth, isLast)
		lines = append(lines, line)

		// Skip children if collapsed
		if !m.expanded[node.AgentID] && len(node.Children) > 0 {
			return false // Don't visit children
		}
		return true
	})

	// Apply scrolling
	visible := m.getVisibleLines(lines)

	return strings.Join(visible, "\n")
}

// renderNode renders a single node with indentation and status
func (m Model) renderNode(node *AgentNode, depth int, isLast bool) string {
	// Tree structure characters
	indent := m.renderIndent(depth, isLast)

	// Status icon
	icon := m.getStatusIcon(node.Status)

	// Agent info
	info := fmt.Sprintf("%s %s", node.Tier, node.AgentID)

	// Duration
	duration := m.formatDuration(node)

	// Expand/collapse indicator
	expandIndicator := ""
	if len(node.Children) > 0 {
		if m.expanded[node.AgentID] {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	}

	// Combine
	line := indent + expandIndicator + icon + " " + info + duration

	// Apply selection style
	if node.AgentID == m.selectedID {
		return m.styles.Selected.Render(line)
	}

	// Apply status color
	style := m.getStatusColor(node.Status)
	return style.Render(line)
}

// renderIndent creates the tree structure indentation
func (m Model) renderIndent(depth int, isLast bool) string {
	if depth == 0 {
		return ""
	}

	var indent string
	for i := 0; i < depth-1; i++ {
		indent += "│  "
	}

	if isLast {
		indent += "└─ "
	} else {
		indent += "├─ "
	}

	return m.styles.TreeLine.Render(indent)
}

// getStatusIcon returns the icon for a given status
func (m Model) getStatusIcon(status AgentStatus) string {
	switch status {
	case StatusSpawning:
		return "⏳"
	case StatusRunning:
		return "●"
	case StatusCompleted:
		return "✓"
	case StatusError:
		return "✗"
	default:
		return "?"
	}
}

// getStatusColor returns the lipgloss style for a given status
func (m Model) getStatusColor(status AgentStatus) lipgloss.Style {
	switch status {
	case StatusSpawning:
		return m.styles.StatusSpawning
	case StatusRunning:
		return m.styles.StatusRunning
	case StatusCompleted:
		return m.styles.StatusCompleted
	case StatusError:
		return m.styles.StatusError
	default:
		return m.styles.Normal
	}
}

// formatDuration formats the duration for display
func (m Model) formatDuration(node *AgentNode) string {
	duration := node.GetDuration()

	if node.Status == StatusCompleted || node.Status == StatusError {
		return fmt.Sprintf(" (%s)", duration.Truncate(time.Millisecond))
	}

	return fmt.Sprintf(" (%s)", duration.Truncate(time.Millisecond))
}

// getDepth calculates the depth of a node in the tree
func (m Model) getDepth(node *AgentNode) int {
	depth := 0
	current := node

	for current.ParentID != "" {
		parent, exists := m.tree.GetNode(current.ParentID)
		if !exists {
			break
		}
		depth++
		current = parent
	}

	return depth
}

// isLastChild checks if a node is the last child of its parent
func (m Model) isLastChild(node *AgentNode) bool {
	if node.ParentID == "" {
		// Root node or orphan
		return true
	}

	parent, exists := m.tree.GetNode(node.ParentID)
	if !exists {
		return true
	}

	if len(parent.Children) == 0 {
		return true
	}

	lastChild := parent.Children[len(parent.Children)-1]
	return lastChild.AgentID == node.AgentID
}

// rebuildVisibleNodes rebuilds the list of visible node IDs
func (m *Model) rebuildVisibleNodes() {
	m.visibleNodes = make([]string, 0)

	if m.tree == nil || m.tree.Root == nil {
		return
	}

	m.tree.WalkTree(func(node *AgentNode) bool {
		m.visibleNodes = append(m.visibleNodes, node.AgentID)

		// Skip children if collapsed
		if !m.expanded[node.AgentID] && len(node.Children) > 0 {
			return false
		}
		return true
	})

	// Update cursor position if it's out of bounds
	if m.cursorPos >= len(m.visibleNodes) {
		m.cursorPos = len(m.visibleNodes) - 1
	}
	if m.cursorPos < 0 {
		m.cursorPos = 0
	}

	// Update selected ID
	if len(m.visibleNodes) > 0 {
		m.selectedID = m.visibleNodes[m.cursorPos]
	} else {
		m.selectedID = ""
	}
}

// getVisibleLines returns the visible portion of lines based on scroll offset
func (m Model) getVisibleLines(lines []string) []string {
	// Calculate available height (subtract border, title, stats, etc.)
	availableHeight := m.height - 8
	if availableHeight < 1 {
		availableHeight = 1
	}

	start := m.scrollOffset
	end := m.scrollOffset + availableHeight

	if start >= len(lines) {
		start = len(lines) - 1
		if start < 0 {
			start = 0
		}
	}

	if end > len(lines) {
		end = len(lines)
	}

	if start >= end {
		return []string{}
	}

	return lines[start:end]
}

// moveUp moves the cursor up
func (m *Model) moveUp() {
	if m.cursorPos > 0 {
		m.cursorPos--
		m.updateSelectedID()
		m.ensureVisible()
	}
}

// moveDown moves the cursor down
func (m *Model) moveDown() {
	maxPos := len(m.visibleNodes) - 1
	if m.cursorPos < maxPos {
		m.cursorPos++
		m.updateSelectedID()
		m.ensureVisible()
	}
}

// updateSelectedID updates the selected agent ID based on cursor position
func (m *Model) updateSelectedID() {
	if m.cursorPos >= 0 && m.cursorPos < len(m.visibleNodes) {
		m.selectedID = m.visibleNodes[m.cursorPos]
	}
}

// ensureVisible adjusts scroll offset to keep cursor visible
func (m *Model) ensureVisible() {
	availableHeight := m.height - 8
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Scroll up if cursor is above visible area
	if m.cursorPos < m.scrollOffset {
		m.scrollOffset = m.cursorPos
	}

	// Scroll down if cursor is below visible area
	if m.cursorPos >= m.scrollOffset+availableHeight {
		m.scrollOffset = m.cursorPos - availableHeight + 1
	}
}

// toggleExpand toggles the expand/collapse state of the selected node
func (m *Model) toggleExpand() {
	if m.selectedID == "" {
		return
	}

	node, exists := m.tree.GetNode(m.selectedID)
	if !exists || len(node.Children) == 0 {
		return
	}

	m.expanded[m.selectedID] = !m.expanded[m.selectedID]
}

// selectNode returns a command that emits a SelectionMsg
func (m Model) selectNode() tea.Cmd {
	if m.selectedID == "" {
		return nil
	}

	return func() tea.Msg {
		return SelectionMsg{AgentID: m.selectedID}
	}
}

// refresh returns a command to refresh the tree from telemetry
func (m Model) refresh() tea.Cmd {
	// This will be implemented when we integrate with TelemetryWatcher
	// For now, just return nil
	return nil
}

// SetFocused sets the focus state of the tree view
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// GetSelectedAgent returns the currently selected agent node
func (m Model) GetSelectedAgent() (*AgentNode, bool) {
	if m.selectedID == "" {
		return nil, false
	}

	return m.tree.GetNode(m.selectedID)
}

// SetSize updates the width and height of the tree view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// ExpandAll expands all nodes in the tree
func (m *Model) ExpandAll() {
	if m.tree == nil {
		return
	}

	m.tree.WalkTree(func(node *AgentNode) bool {
		if len(node.Children) > 0 {
			m.expanded[node.AgentID] = true
		}
		return true
	})

	m.rebuildVisibleNodes()
}

// CollapseAll collapses all nodes in the tree
func (m *Model) CollapseAll() {
	m.expanded = make(map[string]bool)
	m.rebuildVisibleNodes()
}
