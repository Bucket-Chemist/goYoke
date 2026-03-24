// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all keyboard event handlers for AppModel.
// Extracted from app.go as part of TUI-043.
package model

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
)

// handleKey routes a KeyMsg based on modal and focus state.
func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While the plan view modal is open, forward all keys to it.
	if m.shared != nil && m.shared.planViewModal.IsActive() {
		updated, cmd := m.shared.planViewModal.Update(msg)
		m.shared.planViewModal = updated
		return m, cmd
	}

	// While a modal is open only modal keys are active.
	if m.shared != nil && m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		return m.handleModalKey(msg)
	}

	// While the search overlay is active it captures all key events.
	// Dismiss the search overlay if a modal opens (handled in renderLayout).
	if m.shared != nil && m.shared.searchOverlay != nil && m.shared.searchOverlay.IsActive() {
		cmd := m.shared.searchOverlay.HandleMsg(msg)
		return m, cmd
	}

	// Global keys are checked before focus-specific routing.
	switch {
	case key.Matches(msg, m.keys.Global.ForceQuit):
		// Double-Ctrl+C: second press forces immediate exit (TUI-034).
		if m.shutdownInProgress {
			return m, tea.Quit
		}

		// First Ctrl+C: initiate graceful shutdown sequence.
		m.shutdownInProgress = true

		// If ShutdownManager is wired, run sequenced shutdown in background
		// and quit when it completes. Otherwise, fall back to save + quit.
		if m.shared != nil && m.shared.shutdownFunc != nil {
			shutdownFn := m.shared.shutdownFunc
			return m, func() tea.Msg {
				err := shutdownFn()
				return ShutdownCompleteMsg{Err: err}
			}
		}

		// Fallback: save session directly and quit (pre-TUI-034 behaviour).
		m.saveSession()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Global.ToggleFocus):
		m.focus = FocusNext(m.focus)
		m.syncFocusState()
		return m, nil

	// TUI-052: Shift+Tab triggers reverse focus cycling.
	case key.Matches(msg, m.keys.Global.ReverseToggleFocus):
		m.focus = FocusPrev(m.focus)
		m.syncFocusState()
		return m, nil

	case key.Matches(msg, m.keys.Global.CycleRightPanel):
		m.rightPanelMode = NextRightPanelMode(m.rightPanelMode)
		return m, nil

	case key.Matches(msg, m.keys.Global.CycleProvider):
		// Block provider switching while an assistant response is streaming.
		if m.shared != nil && m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming() {
			return m, nil
		}
		// Debounce: increment the sequence counter and fire a 300 ms timer.
		// Only the timer carrying the latest Seq will execute the switch;
		// any earlier timers are silently discarded in the
		// ProviderSwitchExecuteMsg handler.
		m.providerSwitchSeq++
		seq := m.providerSwitchSeq
		return m, tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
			return ProviderSwitchExecuteMsg{Seq: seq}
		})

	case key.Matches(msg, m.keys.Global.ToggleTaskBoard):
		if m.shared != nil && m.shared.taskBoard != nil {
			m.shared.taskBoard.Toggle()
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ViewPlan):
		// Only open the plan viewer when the right panel is showing the plan.
		if m.rightPanelMode == RPMPlanPreview && m.shared != nil {
			markdown := ""
			if m.shared.planPreview != nil {
				markdown = m.shared.planPreview.Content()
			}
			m.shared.planViewModal.SetContent(markdown, m.width)
			m.shared.planViewModal.SetSize(m.width, m.height)
			m.shared.planViewModal.Show()
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.Search):
		// ctrl+f opens the unified cross-panel search overlay (TUI-059).
		if m.shared != nil && m.shared.searchOverlay != nil {
			m.shared.searchOverlay.SetSize(m.width, m.height)
			m.shared.searchOverlay.Activate()
		}
		return m, nil
	}

	// Focus-specific routing.
	switch m.focus {
	case FocusClaude:
		return m.handleClaudeKey(msg)
	case FocusAgents:
		return m.handleAgentsKey(msg)
	}

	return m, nil
}

// handleModalKey forwards all key events to the active ModalModel via the
// ModalQueue.UpdateActive method.  The queue's ModalModel produces a
// ModalResponseMsg when the user confirms or cancels, which is then handled
// in Update → modals.ModalResponseMsg case.
func (m AppModel) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.modalQueue == nil {
		return m, nil
	}
	cmd := m.shared.modalQueue.UpdateActive(msg)
	return m, cmd
}

// handleClaudeKey processes key events when the Claude panel holds focus.
func (m AppModel) handleClaudeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shared != nil && m.shared.claudePanel != nil {
		cmd := m.shared.claudePanel.HandleMsg(msg)
		return m, cmd
	}
	return m, nil
}

// handleAgentsKey processes key events when the agent tree holds focus.
func (m AppModel) handleAgentsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.agentTree.Update(msg)
	if updated, ok := result.(agents.AgentTreeModel); ok {
		m.agentTree = updated
	}
	return m, cmd
}

// syncFocusState propagates the current focus state to child components.
func (m *AppModel) syncFocusState() {
	if m.shared != nil && m.shared.claudePanel != nil {
		m.shared.claudePanel.SetFocused(m.focus == FocusClaude)
	}
	m.agentTree.SetFocused(m.focus == FocusAgents)
}
