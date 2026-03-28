// Package tabbar exports internal symbols for use in external tests (_test
// package).  This file is compiled only during testing.
package tabbar

import tea "github.com/charmbracelet/bubbletea"

// ExportedFlashTick returns a tabFlashTickMsg as a tea.Msg so that external
// test packages can deliver it to TabBarModel.Update without knowing the
// concrete unexported type.
func ExportedFlashTick() tea.Msg {
	return tabFlashTickMsg{}
}
