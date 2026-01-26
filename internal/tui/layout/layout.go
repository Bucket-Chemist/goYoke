package layout

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/claude"
)

// Layout constants define the proportional split and minimum dimensions
const (
	LeftPanelRatio  = 0.70
	RightPanelRatio = 0.30
	MinLeftWidth    = 40
	MinRightWidth   = 20
)

// FocusedPanel indicates which panel currently has focus
type FocusedPanel int

const (
	FocusLeft FocusedPanel = iota
	FocusRight
)

// Model represents the main TUI layout integrating Claude panel and agent tree
type Model struct {
	banner      BannerModel
	claudePanel claude.PanelModel
	agentTree   agents.Model
	agentDetail agents.DetailModel
	width       int
	height      int
	focused     FocusedPanel
	activeView  View
}

// NewModel creates a new main layout model
func NewModel(claudePanel claude.PanelModel, agentTree agents.Model, sessionID string) Model {
	return Model{
		banner:      NewBannerModel(sessionID),
		claudePanel: claudePanel,
		agentTree:   agentTree,
		agentDetail: agents.NewDetailModel(),
		focused:     FocusLeft,
		activeView:  ViewClaude,
	}
}

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	return m.claudePanel.Init()
}

// Update implements tea.Model.Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.activeView = ViewClaude
			m.banner.SetActiveView(ViewClaude)
			return m, nil

		case "2":
			m.activeView = ViewAgents
			m.banner.SetActiveView(ViewAgents)
			return m, nil

		case "3":
			m.activeView = ViewStats
			m.banner.SetActiveView(ViewStats)
			return m, nil

		case "4":
			m.activeView = ViewQuery
			m.banner.SetActiveView(ViewQuery)
			return m, nil

		case "tab":
			// Toggle focus between panels
			if m.focused == FocusLeft {
				m.focused = FocusRight
				m.claudePanel.Blur()
				m.agentTree.SetFocused(true)
			} else {
				m.focused = FocusLeft
				m.claudePanel.Focus()
				m.agentTree.SetFocused(false)
			}
			return m, nil

		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case agents.SelectionMsg:
		// Tree selection changed - update detail panel
		if agent, ok := m.agentTree.GetSelectedAgent(); ok {
			m.agentDetail.SetAgent(agent)
		}

	case cli.Event:
		// Forward CLI events to claude panel even when not focused
		// (to keep cost and conversation history in sync)
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.claudePanel.Update(msg)
		m.claudePanel = model.(claude.PanelModel)
		cmds = append(cmds, cmd)

		// Update banner cost from claude panel
		m.banner.SetCost(m.claudePanel.GetCost())
		return m, tea.Batch(cmds...)
	}

	// Forward message to focused panel
	if m.focused == FocusLeft {
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.claudePanel.Update(msg)
		m.claudePanel = model.(claude.PanelModel)
		cmds = append(cmds, cmd)

		// Update banner cost from claude panel
		m.banner.SetCost(m.claudePanel.GetCost())
	} else {
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.agentTree.Update(msg)
		m.agentTree = model.(agents.Model)
		cmds = append(cmds, cmd)

		// Update detail panel with currently selected agent
		if agent, ok := m.agentTree.GetSelectedAgent(); ok {
			m.agentDetail.SetAgent(agent)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.View
func (m Model) View() string {
	// Render banner at top (1 line height)
	m.banner.SetWidth(m.width)
	bannerView := m.banner.View()

	// Calculate content height (total height - banner height)
	const bannerHeight = 1
	contentHeight := m.height - bannerHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	leftWidth, rightWidth := m.calculateLayout()

	// Left panel (Claude interface)
	leftContent := m.claudePanel.View()
	leftPanel := leftPanelStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render(leftContent)

	// Right panel (Agent tree + detail vertically stacked)
	treeHeight := contentHeight / 2
	detailHeight := contentHeight - treeHeight

	// Update right panel component sizes
	m.agentTree.SetSize(rightWidth, treeHeight)
	m.agentDetail.SetSize(rightWidth, detailHeight)

	treeView := m.agentTree.View()
	detailView := m.agentDetail.View()

	rightContent := lipgloss.JoinVertical(
		lipgloss.Left,
		treeView,
		detailView,
	)

	rightPanel := rightPanelStyle.
		Width(rightWidth).
		Height(contentHeight).
		Render(rightContent)

	// Apply focus indicator (cyan border for focused panel)
	if m.focused == FocusLeft {
		leftPanel = focusedStyle.Render(leftPanel)
	} else {
		rightPanel = focusedStyle.Render(rightPanel)
	}

	// Join left and right panels horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	// Join banner and main content vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		bannerView,
		mainContent,
	)
}

// calculateLayout computes left and right panel widths enforcing minimums
func (m Model) calculateLayout() (leftWidth, rightWidth int) {
	available := m.width - 1 // Reserve space for border

	// Calculate ideal widths based on ratios
	leftWidth = int(float64(available) * LeftPanelRatio)
	rightWidth = available - leftWidth

	// Check if both minimums can be satisfied
	minTotal := MinLeftWidth + MinRightWidth
	if available < minTotal {
		// Can't satisfy both minimums - prioritize right panel minimum
		rightWidth = MinRightWidth
		leftWidth = available - rightWidth
		if leftWidth < 0 {
			leftWidth = 0
		}
		return leftWidth, rightWidth
	}

	// Enforce minimum widths with priority
	if rightWidth < MinRightWidth {
		// Right panel needs to be expanded to minimum
		rightWidth = MinRightWidth
		leftWidth = available - rightWidth
	} else if leftWidth < MinLeftWidth {
		// Left panel needs to be expanded to minimum
		leftWidth = MinLeftWidth
		rightWidth = available - leftWidth
	}

	return leftWidth, rightWidth
}

// updateSizes propagates size updates to all child components
func (m *Model) updateSizes() {
	// Account for banner height (1 line)
	const bannerHeight = 1
	contentHeight := m.height - bannerHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Update banner width
	m.banner.SetWidth(m.width)

	leftWidth, rightWidth := m.calculateLayout()

	// Update left panel (Claude interface)
	m.claudePanel.SetSize(leftWidth, contentHeight)

	// Update right panel components (tree + detail split vertically)
	treeHeight := contentHeight / 2
	detailHeight := contentHeight - treeHeight

	m.agentTree.SetSize(rightWidth, treeHeight)
	m.agentDetail.SetSize(rightWidth, detailHeight)
}

// Styles for layout rendering
var (
	leftPanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true)

	rightPanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder())

	focusedStyle = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("cyan"))
)
