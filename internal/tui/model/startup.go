// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the startup sequence helpers for wiring the CLI driver
// and IPC bridge into the Bubbletea event loop (TUI-016).
package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// maxReconnectAttempts is the maximum number of times AppModel will try to
// restart the CLI subprocess after an unexpected disconnect.  After this
// limit is reached the model stops retrying and remains in a disconnected
// state until the user exits.
const maxReconnectAttempts = 3

// reconnectAfterDelay returns a tea.Cmd that fires a CLIReconnectMsg after a
// back-off delay proportional to the attempt number.
//
// Back-off schedule:
//
//	attempt 1 → 2 s
//	attempt 2 → 4 s
//	attempt 3 → 6 s
func reconnectAfterDelay(attempt int, seq int) tea.Cmd {
	delay := time.Duration(attempt) * 2 * time.Second
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return CLIReconnectMsg{Attempt: attempt, Seq: seq}
	})
}
