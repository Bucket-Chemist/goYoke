// +build ignore

// Example program to demonstrate the dashboard
// Run with: go run internal/tui/performance/example_main.go

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/tui/performance"
)

func main() {
	p := tea.NewProgram(performance.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
