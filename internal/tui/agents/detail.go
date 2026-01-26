package agents

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DetailModel represents the agent detail sidebar component
type DetailModel struct {
	agent  *AgentNode
	width  int
	height int
}

// NewDetailModel creates a new detail model
func NewDetailModel() DetailModel {
	return DetailModel{}
}

// SetAgent sets the agent to display details for
func (m *DetailModel) SetAgent(agent *AgentNode) {
	m.agent = agent
}

// SetSize updates the width and height of the detail panel
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the detail panel
func (m DetailModel) View() string {
	if m.agent == nil {
		return emptyDetailStyle.Render("No agent selected")
	}

	var b strings.Builder

	// Header section
	b.WriteString(detailHeaderStyle.Render(fmt.Sprintf("Selected: %s", m.agent.AgentID)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Tier: %s", m.agent.Tier))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Status: %s %s", m.agent.Status, m.statusIndicator()))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Duration: %s", m.durationString()))
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("-", m.width-2))
	b.WriteString("\n")

	// Task description section
	b.WriteString(detailLabelStyle.Render("Task Description:"))
	b.WriteString("\n")
	if m.agent.Description != "" {
		// Word wrap the description
		wrapped := wordWrap(m.agent.Description, m.width-4)
		b.WriteString(wrapped)
	} else {
		b.WriteString(emptyDetailStyle.Render("(no description)"))
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("-", m.width-2))
	b.WriteString("\n")

	// Keyboard hints (dynamic based on agent state)
	b.WriteString(hintStyle.Render("[Space] Expand/Collapse"))
	b.WriteString("\n")

	if m.agent.IsActive() {
		b.WriteString(hintStyle.Render("[q] Query agent"))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("[x] Stop agent"))
		b.WriteString("\n")
	}

	b.WriteString(hintStyle.Render("[s] Spawn new agent"))

	return b.String()
}

// statusIndicator returns the Unicode status indicator for the agent
func (m DetailModel) statusIndicator() string {
	switch m.agent.Status {
	case StatusSpawning:
		return "⏳"
	case StatusRunning:
		return "⟳"
	case StatusCompleted:
		return "✓"
	case StatusError:
		return "✗"
	default:
		return ""
	}
}

// durationString formats the duration string with real-time updates for active agents
func (m DetailModel) durationString() string {
	if m.agent.CompleteTime != nil {
		// Completed agent - show final duration
		return fmt.Sprintf("%.1fs", m.agent.GetDuration().Seconds())
	}
	// Active agent - show elapsed time with ellipsis
	elapsed := time.Since(m.agent.SpawnTime)
	return fmt.Sprintf("%.1fs...", elapsed.Seconds())
}

// wordWrap wraps text to the specified width
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}

	var result strings.Builder
	words := strings.Fields(s)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)

		// If adding this word would exceed width, start a new line
		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}

		// Add space before word if not at start of line
		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}

		// Add the word
		result.WriteString(word)
		lineLen += wordLen

		// Continue to next word
		if i < len(words)-1 && lineLen > 0 {
			// Continue on same line if possible
		}
	}

	return result.String()
}

// Styles for the detail panel
var (
	detailHeaderStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	detailLabelStyle  = lipgloss.NewStyle().Bold(true)
	emptyDetailStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	hintStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
