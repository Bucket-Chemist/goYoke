// Package model defines shared state types for the goYoke TUI.
// This file contains handlers for the remote-action message types (HL-005).
// Each handler delegates to an existing AppModel handler rather than
// duplicating side effects — the Bubbletea goroutine remains the sole
// serialisation point for all subprocess writes and state mutations.
package model

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleRemoteSubmitPrompt handles RemoteSubmitPromptMsg: submits a user
// prompt through the existing CLI driver send path. All subprocess writes are
// serialised on the Bubbletea goroutine; the control server must never call
// CLIDriver.SendMessage() directly.
//
// When ResponseCh is non-nil it receives nil on successful queue or an error
// when the driver is unavailable. The channel must be buffered (capacity ≥ 1).
func (m AppModel) handleRemoteSubmitPrompt(msg RemoteSubmitPromptMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.cliDriver == nil {
		sendRemoteResponse(msg.ResponseCh, fmt.Errorf("cli driver not available"))
		return m, nil
	}

	sendCmd := m.shared.cliDriver.SendMessage(msg.Prompt)

	if msg.ResponseCh != nil {
		responseCh := msg.ResponseCh
		if sendCmd != nil {
			origCmd := sendCmd
			sendCmd = func() tea.Msg {
				result := origCmd()
				if result == nil {
					responseCh <- nil
				} else {
					responseCh <- fmt.Errorf("send failed: driver disconnected")
				}
				return result
			}
		} else {
			// Driver accepted synchronously (nil-cmd path: mock or no-op driver).
			responseCh <- nil
		}
	}

	return m, sendCmd
}

// handleRemoteInterrupt handles RemoteInterruptMsg: interrupts the active CLI
// operation via the existing CLIDriver.Interrupt() method (mutex-protected).
func (m AppModel) handleRemoteInterrupt(msg RemoteInterruptMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.cliDriver == nil {
		sendRemoteResponse(msg.ResponseCh, fmt.Errorf("cli driver not available"))
		return m, nil
	}

	err := m.shared.cliDriver.Interrupt()
	sendRemoteResponse(msg.ResponseCh, err)
	return m, nil
}

// handleRemoteRespondModal handles RemoteRespondModalMsg: delivers a user
// response to a pending bridge modal request via the existing
// bridge.ResolveModalSimple path.
func (m AppModel) handleRemoteRespondModal(msg RemoteRespondModalMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.bridge == nil {
		sendRemoteResponse(msg.ResponseCh, fmt.Errorf("bridge not available"))
		return m, nil
	}

	m.shared.bridge.ResolveModalSimple(msg.RequestID, msg.Value)
	sendRemoteResponse(msg.ResponseCh, nil)
	return m, nil
}

// handleRemoteRespondPermission handles RemoteRespondPermissionMsg: delivers a
// permission decision to a pending bridge permission gate via the existing
// bridge.ResolvePermGate path.
func (m AppModel) handleRemoteRespondPermission(msg RemoteRespondPermissionMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.bridge == nil {
		sendRemoteResponse(msg.ResponseCh, fmt.Errorf("bridge not available"))
		return m, nil
	}

	m.shared.bridge.ResolvePermGate(msg.RequestID, msg.Decision)
	sendRemoteResponse(msg.ResponseCh, nil)
	return m, nil
}

// handleRemoteSetModel handles RemoteSetModelMsg: changes the active model by
// reusing the existing handleModelSwitchRequest flow (provider state update
// and CLI driver restart). ResponseCh is signaled before the switch executes.
func (m AppModel) handleRemoteSetModel(msg RemoteSetModelMsg) (tea.Model, tea.Cmd) {
	sendRemoteResponse(msg.ResponseCh, nil)
	return m.handleModelSwitchRequest(ModelSwitchRequestMsg{ModelID: msg.ModelID})
}

// handleRemoteSetEffort handles RemoteSetEffortMsg: changes the effort level
// by reusing the existing handleEffortChangeRequest flow (activeEffort update
// and CLI driver restart). ResponseCh is signaled before the change executes.
func (m AppModel) handleRemoteSetEffort(msg RemoteSetEffortMsg) (tea.Model, tea.Cmd) {
	sendRemoteResponse(msg.ResponseCh, nil)
	return m.handleEffortChangeRequest(EffortChangeRequestMsg{Level: msg.Level})
}

// handleRemoteSetCWD handles RemoteSetCWDMsg: changes the working directory
// by reusing the existing handleCWDChanged flow (os.Chdir, GOYOKE_CWD env
// var, and status line update). Errors from os.Chdir are logged by the
// delegate; ResponseCh always receives nil to indicate the request was accepted.
func (m AppModel) handleRemoteSetCWD(msg RemoteSetCWDMsg) (tea.Model, tea.Cmd) {
	sendRemoteResponse(msg.ResponseCh, nil)
	return m.handleCWDChanged(CWDChangedMsg{Path: msg.Path})
}

// sendRemoteResponse writes err to ch when ch is non-nil. The write is
// non-blocking because all ResponseCh channels must be buffered (capacity ≥ 1)
// by the caller. Writing to a nil channel is a no-op.
func sendRemoteResponse(ch chan<- error, err error) {
	if ch != nil {
		ch <- err
	}
}
