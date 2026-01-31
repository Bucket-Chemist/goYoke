package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Minimal reproduction of the gofortress layout to debug mouse events

type innerModel struct {
	focused bool
	clicks  int
	lastMsg string
}

func (m innerModel) Init() tea.Cmd { return nil }

func (m innerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		m.lastMsg = fmt.Sprintf("INNER got mouse: focused=%v action=%v button=%v", m.focused, msg.Action, msg.Button)
		if m.focused {
			m.clicks++
		}
	}
	return m, nil
}

func (m innerModel) View() string {
	status := "UNFOCUSED"
	if m.focused {
		status = "FOCUSED"
	}
	return fmt.Sprintf("[%s] Clicks: %d\nLast: %s", status, m.clicks, m.lastMsg)
}

func (m *innerModel) Focus()          { m.focused = true }
func (m *innerModel) Blur()           { m.focused = false }
func (m innerModel) IsFocused() bool  { return m.focused }

type outerModel struct {
	left     innerModel
	right    innerModel
	focusedL bool // true = left focused
	debug    []string
}

func (m outerModel) Init() tea.Cmd {
	return nil
}

func (m outerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "tab" {
			m.focusedL = !m.focusedL
			if m.focusedL {
				m.left.Focus()
				m.right.Blur()
			} else {
				m.left.Blur()
				m.right.Focus()
			}
			m.debug = append(m.debug, fmt.Sprintf("TAB: focusedL=%v left.focused=%v right.focused=%v", m.focusedL, m.left.focused, m.right.focused))
		}

	case tea.MouseMsg:
		m.debug = append(m.debug, fmt.Sprintf("OUTER got mouse: action=%v button=%v x=%d", msg.Action, msg.Button, msg.X))

		// Click-to-focus logic (like layout.go)
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if msg.X < 40 {
				if !m.focusedL {
					m.focusedL = true
					m.left.Focus()
					m.right.Blur()
					m.debug = append(m.debug, "-> Switched focus to LEFT")
				}
			} else {
				if m.focusedL {
					m.focusedL = false
					m.left.Blur()
					m.right.Focus()
					m.debug = append(m.debug, "-> Switched focus to RIGHT")
				}
			}
		}

		// Forward to focused panel (like layout.go lines 153-178)
		if m.focusedL {
			var model tea.Model
			model, _ = m.left.Update(msg)
			m.left = model.(innerModel)
			m.debug = append(m.debug, fmt.Sprintf("-> Forwarded to LEFT, left.focused=%v", m.left.focused))
		} else {
			var model tea.Model
			model, _ = m.right.Update(msg)
			m.right = model.(innerModel)
			m.debug = append(m.debug, fmt.Sprintf("-> Forwarded to RIGHT, right.focused=%v", m.right.focused))
		}
	}

	// Keep last 15 debug lines
	if len(m.debug) > 15 {
		m.debug = m.debug[len(m.debug)-15:]
	}

	return m, nil
}

func (m outerModel) View() string {
	leftStyle := lipgloss.NewStyle().Width(38).Border(lipgloss.NormalBorder())
	rightStyle := lipgloss.NewStyle().Width(38).Border(lipgloss.NormalBorder())

	if m.focusedL {
		leftStyle = leftStyle.BorderForeground(lipgloss.Color("green"))
	} else {
		rightStyle = rightStyle.BorderForeground(lipgloss.Color("green"))
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render("LEFT PANEL\n"+m.left.View()),
		rightStyle.Render("RIGHT PANEL\n"+m.right.View()),
	)

	debug := "\n--- DEBUG LOG ---\n"
	for _, d := range m.debug {
		debug += d + "\n"
	}

	return panels + debug + "\nPress 'q' to quit, 'tab' to switch focus, click panels"
}

func main() {
	m := outerModel{
		left:     innerModel{focused: true},
		right:    innerModel{focused: false},
		focusedL: true,
	}

	p := tea.NewProgram(m, tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
