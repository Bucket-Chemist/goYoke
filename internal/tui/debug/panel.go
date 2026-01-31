package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Entry represents a single debug log entry
type Entry struct {
	Timestamp time.Time
	Source    string
	Message   string
}

// PanelModel is the debug overlay panel
type PanelModel struct {
	entries []Entry
	maxSize int
	width   int
	height  int
	visible bool
}

// NewPanelModel creates a new debug panel
func NewPanelModel() PanelModel {
	return PanelModel{
		entries: make([]Entry, 0, 50),
		maxSize: 50,
		width:   60,
		height:  20,
	}
}

// AddEntry adds a debug entry (call via tea.Msg)
func (m *PanelModel) AddEntry(source, message string) {
	entry := Entry{
		Timestamp: time.Now(),
		Source:    source,
		Message:   message,
	}
	m.entries = append(m.entries, entry)
	if len(m.entries) > m.maxSize {
		m.entries = m.entries[1:]
	}
}

// Toggle visibility
func (m *PanelModel) Toggle() {
	m.visible = !m.visible
}

// IsVisible returns visibility state
func (m PanelModel) IsVisible() bool {
	return m.visible
}

// SetSize updates panel dimensions
func (m *PanelModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the debug panel as overlay content
func (m PanelModel) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("Debug Log (ctrl+d to close)"))
	b.WriteString("\n\n")

	// Show recent entries (fit to height)
	availableLines := m.height - 4
	start := 0
	if len(m.entries) > availableLines {
		start = len(m.entries) - availableLines
	}

	if len(m.entries) == 0 {
		b.WriteString(emptyStyle.Render("No debug entries yet"))
	} else {
		for _, e := range m.entries[start:] {
			ts := e.Timestamp.Format("15:04:05.000")
			src := sourceStyle.Render(fmt.Sprintf("[%s]", truncateSource(e.Source)))
			msg := msgStyle.Render(truncateMsg(e.Message, m.width-20))
			b.WriteString(fmt.Sprintf("%s %s %s\n", ts, src, msg))
		}
	}

	return panelStyle.
		Width(m.width).
		Height(m.height).
		Render(b.String())
}

func truncateSource(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func truncateMsg(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	sourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("cyan"))

	msgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)
