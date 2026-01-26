package agents

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

// PickerModel is a TUI component for selecting an agent to spawn
type PickerModel struct {
	agents   []cli.SubagentConfig
	cursor   int
	selected string
	width    int
	height   int
	focused  bool

	// Filtering
	filter     string
	filtered   []cli.SubagentConfig
	filterMode bool

	styles PickerStyles
}

// PickerStyles defines styling for the picker
type PickerStyles struct {
	Title    lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
	Tier     lipgloss.Style
	Desc     lipgloss.Style
	Filter   lipgloss.Style
	Help     lipgloss.Style
}

// DefaultPickerStyles returns default styling
func DefaultPickerStyles() PickerStyles {
	return PickerStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Bold(true),
		Normal: lipgloss.NewStyle(),
		Tier: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(8),
		Desc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		Filter: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// SpawnAgentMsg is emitted when user selects an agent to spawn
type SpawnAgentMsg struct {
	AgentName string
}

// PickerCancelMsg is emitted when user cancels the picker
type PickerCancelMsg struct{}

// NewPickerModel creates a new picker with the given agents
func NewPickerModel(agents []cli.SubagentConfig) PickerModel {
	return PickerModel{
		agents:   agents,
		filtered: agents,
		cursor:   0,
		styles:   DefaultPickerStyles(),
	}
}

// Init implements tea.Model
func (m PickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.filterMode {
			return m.handleFilterInput(msg)
		}
		return m.handleNavigation(msg)
	}

	return m, nil
}

// handleNavigation processes navigation keys
func (m PickerModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "enter":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			return m, m.selectAgent()
		}

	case "esc", "q":
		return m, m.cancel()

	case "/":
		m.filterMode = true
		m.filter = ""

	case "home", "g":
		m.cursor = 0

	case "end", "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	}

	return m, nil
}

// handleFilterInput processes filter input
func (m PickerModel) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filterMode = false
		return m, nil

	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}

	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
		}
	}

	return m, nil
}

// applyFilter filters agents based on current filter string
func (m *PickerModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.agents
		return
	}

	lower := strings.ToLower(m.filter)
	m.filtered = nil
	for _, agent := range m.agents {
		if strings.Contains(strings.ToLower(agent.Name), lower) ||
			strings.Contains(strings.ToLower(agent.Description), lower) {
			m.filtered = append(m.filtered, agent)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	}
}

// selectAgent emits SpawnAgentMsg
func (m PickerModel) selectAgent() tea.Cmd {
	if m.cursor >= len(m.filtered) {
		return nil
	}
	selected := m.filtered[m.cursor]
	return func() tea.Msg {
		return SpawnAgentMsg{AgentName: selected.Name}
	}
}

// cancel emits PickerCancelMsg
func (m PickerModel) cancel() tea.Cmd {
	return func() tea.Msg {
		return PickerCancelMsg{}
	}
}

// View implements tea.Model
func (m PickerModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("Select Agent to Spawn"))
	b.WriteString("\n")

	// Filter indicator
	if m.filterMode {
		b.WriteString(m.styles.Filter.Render("Filter: " + m.filter + "█"))
		b.WriteString("\n")
	} else if m.filter != "" {
		b.WriteString(m.styles.Filter.Render("Filter: " + m.filter))
		b.WriteString("\n")
	}

	// Handle uninitialized width
	separatorWidth := m.width - 4
	if separatorWidth < 0 {
		separatorWidth = 0
	}
	b.WriteString(strings.Repeat("─", separatorWidth))
	b.WriteString("\n")

	// Agent list
	if len(m.filtered) == 0 {
		b.WriteString(m.styles.Desc.Render("No agents match filter"))
		b.WriteString("\n")
	} else {
		// Calculate visible range (scroll support)
		visibleCount := m.height - 8
		if visibleCount < 3 {
			visibleCount = 3
		}

		start := 0
		if m.cursor >= visibleCount {
			start = m.cursor - visibleCount + 1
		}
		end := start + visibleCount
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := start; i < end; i++ {
			agent := m.filtered[i]

			// Indicator
			indicator := "  "
			if i == m.cursor {
				indicator = "▸ "
			}

			// Format: [tier] name - description
			tier := m.styles.Tier.Render(fmt.Sprintf("[%s]", agent.Tier))
			name := agent.Name
			desc := agent.Description

			// Truncate description to fit
			maxDescLen := m.width - len(name) - 15
			if maxDescLen > 0 && len(desc) > maxDescLen {
				desc = desc[:maxDescLen-3] + "..."
			}

			line := fmt.Sprintf("%s%s %s - %s", indicator, tier, name, m.styles.Desc.Render(desc))

			if i == m.cursor {
				line = m.styles.Selected.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Help line
	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("↑/↓ navigate • enter select • / filter • esc cancel"))

	return b.String()
}

// SetSize updates the picker dimensions
func (m *PickerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Focus sets focus state
func (m *PickerModel) Focus() {
	m.focused = true
}

// Blur removes focus
func (m *PickerModel) Blur() {
	m.focused = false
}

// GetSelected returns the currently highlighted agent name
func (m PickerModel) GetSelected() string {
	if m.cursor < len(m.filtered) {
		return m.filtered[m.cursor].Name
	}
	return ""
}
