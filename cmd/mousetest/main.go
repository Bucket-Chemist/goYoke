package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	events []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		m.events = append(m.events, fmt.Sprintf("KEY: %s", msg.String()))
	case tea.MouseMsg:
		m.events = append(m.events, fmt.Sprintf("MOUSE: action=%v button=%v x=%d y=%d",
			msg.Action, msg.Button, msg.X, msg.Y))
	}

	// Keep only last 10 events
	if len(m.events) > 10 {
		m.events = m.events[len(m.events)-10:]
	}

	return m, nil
}

func (m model) View() string {
	s := "Mouse Test - Click anywhere, press 'q' to quit\n"
	s += "================================================\n\n"

	if len(m.events) == 0 {
		s += "(no events yet - try clicking or pressing keys)\n"
	}

	for _, e := range m.events {
		s += e + "\n"
	}

	return s
}

func main() {
	p := tea.NewProgram(
		model{},
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
