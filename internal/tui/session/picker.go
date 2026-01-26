package session

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

// PickerModel is a Bubbletea modal for selecting, resuming, or deleting sessions
type PickerModel struct {
	sessions []cli.Session
	cursor   int
	width    int
	height   int
	selected *cli.Session
}

// NewPickerModel creates a PickerModel with the given sessions
func NewPickerModel(sessions []cli.Session) PickerModel {
	return PickerModel{
		sessions: sessions,
		cursor:   0,
		selected: nil,
	}
}

// Init initializes the picker (no initial commands)
func (m PickerModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the session picker
func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.sessions) {
				m.selected = &m.sessions[m.cursor]
				return m, tea.Quit
			}
		case "d":
			// Delete selected session
			if m.cursor < len(m.sessions) {
				return m, m.deleteSession(m.sessions[m.cursor].ID)
			}
		case "esc", "q":
			m.selected = nil
			return m, tea.Quit
		}

	case sessionDeletedMsg:
		// Remove deleted session from list
		for i, s := range m.sessions {
			if s.ID == msg.id {
				m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
				// Adjust cursor if needed
				if m.cursor >= len(m.sessions) && m.cursor > 0 {
					m.cursor--
				}
				break
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// deleteSession returns a command that emits a sessionDeletedMsg
func (m PickerModel) deleteSession(id string) tea.Cmd {
	return func() tea.Msg {
		return sessionDeletedMsg{id: id}
	}
}

// sessionDeletedMsg is emitted when a session is deleted
type sessionDeletedMsg struct {
	id string
}

// View renders the session picker modal
func (m PickerModel) View() string {
	if len(m.sessions) == 0 {
		return modalStyle.Width(m.width).Height(m.height).Render(
			headerStyle.Render("No Sessions Found") + "\n\n" +
				hintStyle.Render("[Esc] Close"),
		)
	}

	var b strings.Builder

	b.WriteString(headerStyle.Render("Select Session"))
	b.WriteString("\n")
	if m.width > 4 {
		b.WriteString(strings.Repeat("-", m.width-4))
	}
	b.WriteString("\n\n")

	for i, session := range m.sessions {
		line := m.renderSession(session, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[↑/↓ or j/k] Navigate  [Enter] Resume  [d] Delete  [Esc/q] Cancel"))

	return modalStyle.Width(m.width).Height(m.height).Render(b.String())
}

// renderSession formats a single session line with selection indicator
func (m PickerModel) renderSession(session cli.Session, selected bool) string {
	// Use name if set, otherwise truncate ID to 8 chars
	name := session.ID
	if len(name) > 8 {
		name = name[:8]
	}
	if session.Name != "" {
		name = session.Name
	}

	age := formatAge(session.LastUsed)

	line := fmt.Sprintf("%s - Cost: $%.2f - %d tools - %s",
		name,
		session.Cost,
		session.ToolCalls,
		age,
	)

	if selected {
		return selectedStyle.Render("> " + line)
	}
	return "  " + line
}

// Selected returns the selected session, or nil if cancelled
func (m PickerModel) Selected() *cli.Session {
	return m.selected
}

// SetSize updates the width and height of the picker
func (m *PickerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// formatAge formats a timestamp as a human-readable age string
func formatAge(t time.Time) string {
	age := time.Since(t)

	if age < time.Hour {
		minutes := int(age.Minutes())
		if minutes < 1 {
			return "<1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	}

	if age < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	}

	days := int(age.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

// Styles for the session picker modal
var (
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan"))
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
