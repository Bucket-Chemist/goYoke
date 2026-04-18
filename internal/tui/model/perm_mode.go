// Package model defines shared state types for the goYoke TUI.
// This file implements permission-mode cycling (Alt+p).
package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/cli"
)

// permModes is the canonical cycle order for permission escalation modes.
// Matches the --permission-mode flag accepted by the Claude CLI.
var permModes = []string{"default", "acceptEdits", "plan"}

// handlePermModeCycle cycles through permission modes and restarts the CLI
// driver with the new mode.  The session ID is preserved so the conversation
// context is not lost.
//
// Follows the same restart pattern as handleProviderSwitch (provider_switch.go).
func (m AppModel) handlePermModeCycle() (tea.Model, tea.Cmd) {
	current := m.statusLine.PermissionMode

	// Find current index in the cycle.
	idx := 0
	for i, mode := range permModes {
		if mode == current {
			idx = i
			break
		}
	}
	next := permModes[(idx+1)%len(permModes)]

	// Update the status line immediately so the user sees feedback.
	m.statusLine.PermissionMode = next

	// Persist into baseCLIOpts so future provider switches also use the new mode.
	if m.shared != nil {
		m.shared.baseCLIOpts.PermissionMode = next
	}

	// Restart the CLI driver with the new permission mode, preserving the
	// current session so the conversation is not lost.
	if m.shared == nil || m.shared.cliDriver == nil {
		return m, nil
	}

	_ = m.shared.cliDriver.Shutdown()

	opts := m.shared.baseCLIOpts // value copy preserves Verbose, Debug, MCPConfigPath, etc.
	opts.PermissionMode = next
	opts.SessionID = m.sessionID  // resume same session
	opts.Model = m.activeModel

	m.cliReady = false

	newDriver := cli.NewCLIDriver(opts)
	m.shared.cliDriver = newDriver

	if m.shared.claudePanel != nil {
		m.shared.claudePanel.SetSender(newDriver)
	}

	return m, newDriver.Start()
}
