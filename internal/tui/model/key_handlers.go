// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all keyboard event handlers for AppModel.
// Extracted from app.go as part of TUI-043.
package model

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// handleKey routes a KeyMsg based on modal and focus state.
func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While the plan view modal is open, forward all keys to it.
	if m.shared != nil && m.shared.planViewModal.IsActive() {
		// Set hint context to "plan" while plan modal is active (TUI-060).
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("plan")
		}
		updated, cmd := m.shared.planViewModal.Update(msg)
		m.shared.planViewModal = updated
		// Restore main context when plan modal closes.
		if !m.shared.planViewModal.IsActive() {
			m.updateHintContext()
		}
		return m, cmd
	}

	// While a modal is open only modal keys are active.
	if m.shared != nil && m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		// Set hint context to "modal" while a permission modal is active (TUI-060).
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("modal")
		}
		return m.handleModalKey(msg)
	}

	// While the search overlay is active it captures all key events.
	// Dismiss the search overlay if a modal opens (handled in renderLayout).
	if m.shared != nil && m.shared.searchOverlay != nil && m.shared.searchOverlay.IsActive() {
		cmd := m.shared.searchOverlay.HandleMsg(msg)
		// Restore main context when search overlay deactivates (TUI-060).
		if !m.shared.searchOverlay.IsActive() {
			m.updateHintContext()
		}
		return m, cmd
	}

	// Vim mode overlay (TUI-062).
	// In VimNormal mode j/k/h/l are consumed as navigation commands and do
	// NOT fall through to the standard handlers below.  In VimInsert mode all
	// keys pass through unchanged so text input works normally.
	if m.keys.VimEnabled {
		if consumed, model, cmd := m.handleVimKey(msg); consumed {
			return model, cmd
		}
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
		m.updateHintContext()
		m.updateBreadcrumbs()
		// Flash the active tab to acknowledge the focus change (TUI-061).
		return m, tabFlashCmd(int(m.activeTab))

	// TUI-052: Shift+Tab triggers reverse focus cycling.
	case key.Matches(msg, m.keys.Global.ReverseToggleFocus):
		m.focus = FocusPrev(m.focus)
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		// Flash the active tab to acknowledge the focus change (TUI-061).
		return m, tabFlashCmd(int(m.activeTab))

	case key.Matches(msg, m.keys.Global.CycleRightPanel):
		m.rightPanelMode = NextRightPanelMode(m.rightPanelMode)
		m.updateHintContext()
		m.updateBreadcrumbs()
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
			if m.shared.hintBar != nil {
				m.shared.hintBar.SetContext("plan")
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.Search):
		// ctrl+f opens the unified cross-panel search overlay (TUI-059).
		if m.shared != nil && m.shared.searchOverlay != nil {
			m.shared.searchOverlay.SetSize(m.width, m.height)
			m.shared.searchOverlay.Activate()
			if m.shared.hintBar != nil {
				m.shared.hintBar.SetContext("search")
			}
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

// tabFlashCmd returns a Cmd that immediately delivers a TabFlashMsg for the
// given tab index.  It is used after focus changes to trigger the 300 ms
// accent flash on the active tab (TUI-061).
func tabFlashCmd(tabIndex int) tea.Cmd {
	return func() tea.Msg {
		return TabFlashMsg{TabIndex: tabIndex}
	}
}

// handleVimKey processes a key event when vim mode is enabled.  It returns
// (consumed, model, cmd).  When consumed is true the caller must return the
// supplied model and cmd immediately and skip standard key routing.
//
// VimInsert mode: only Esc is intercepted (to return to VimNormal); all other
// keys are NOT consumed so they pass through to the standard handlers.
//
// VimNormal mode: navigation keys (j/k/h/l/g/G) are consumed and mapped to
// their standard TUI equivalents; i enters VimInsert; Esc is left to the
// standard handler (it maps to Global.Interrupt) so interrupts still work.
func (m AppModel) handleVimKey(msg tea.KeyMsg) (consumed bool, model tea.Model, cmd tea.Cmd) {
	switch m.keys.VimMode {
	case config.VimInsert:
		if key.Matches(msg, m.keys.Vim.Normal) {
			m.keys.VimMode = config.VimNormal
			m.statusLine.VimMode = config.VimNormal.String()
			return true, m, nil
		}
		// All other keys pass through in insert mode.
		return false, m, nil

	case config.VimNormal:
		return m.handleVimNormalKey(msg)
	}

	return false, m, nil
}

// handleVimNormalKey processes navigation keys in VimNormal mode.  It maps
// vim bindings to the standard TUI actions without duplicating their logic:
//
//   - j  → same as pressing ↓ (focus-specific down action)
//   - k  → same as pressing ↑ (focus-specific up action)
//   - h  → same as shift+tab (reverse focus / left panel)
//   - l  → same as tab (forward focus / right panel)
//   - g  → scroll to top (sent as "home" equivalent via synthetic key)
//   - G  → scroll to bottom (sent as "end" equivalent via synthetic key)
//   - i  → enter VimInsert mode
//
// Returns (true, ...) when the key was consumed, (false, ...) otherwise so
// unrecognised keys fall through to global/focus-specific handlers.
func (m AppModel) handleVimNormalKey(msg tea.KeyMsg) (consumed bool, model tea.Model, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Vim.Down):
		// j → down: route to the focused panel.
		synth := tea.KeyMsg{Type: tea.KeyDown}
		switch m.focus {
		case FocusClaude:
			result, c := m.handleClaudeKey(synth)
			return true, result, c
		case FocusAgents:
			result, c := m.handleAgentsKey(synth)
			return true, result, c
		}
		return true, m, nil

	case key.Matches(msg, m.keys.Vim.Up):
		// k → up: route to the focused panel.
		synth := tea.KeyMsg{Type: tea.KeyUp}
		switch m.focus {
		case FocusClaude:
			result, c := m.handleClaudeKey(synth)
			return true, result, c
		case FocusAgents:
			result, c := m.handleAgentsKey(synth)
			return true, result, c
		}
		return true, m, nil

	case key.Matches(msg, m.keys.Vim.Right):
		// l → forward focus (same as tab).
		m.focus = FocusNext(m.focus)
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		return true, m, tabFlashCmd(int(m.activeTab))

	case key.Matches(msg, m.keys.Vim.Left):
		// h → reverse focus (same as shift+tab).
		m.focus = FocusPrev(m.focus)
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		return true, m, tabFlashCmd(int(m.activeTab))

	case key.Matches(msg, m.keys.Vim.Top):
		// g → scroll to top: send synthetic Home key to the focused panel.
		synth := tea.KeyMsg{Type: tea.KeyHome}
		switch m.focus {
		case FocusClaude:
			result, c := m.handleClaudeKey(synth)
			return true, result, c
		case FocusAgents:
			result, c := m.handleAgentsKey(synth)
			return true, result, c
		}
		return true, m, nil

	case key.Matches(msg, m.keys.Vim.Bottom):
		// G → scroll to bottom: send synthetic End key to the focused panel.
		synth := tea.KeyMsg{Type: tea.KeyEnd}
		switch m.focus {
		case FocusClaude:
			result, c := m.handleClaudeKey(synth)
			return true, result, c
		case FocusAgents:
			result, c := m.handleAgentsKey(synth)
			return true, result, c
		}
		return true, m, nil

	case key.Matches(msg, m.keys.Vim.Insert):
		// i → enter insert mode.
		m.keys.VimMode = config.VimInsert
		m.statusLine.VimMode = config.VimInsert.String()
		return true, m, nil
	}

	// Key not recognised as a vim binding — fall through to standard routing.
	return false, m, nil
}

// updateBreadcrumbs updates the breadcrumb trail based on the current focus
// and right-panel mode.  It is called after focus changes and panel mode
// changes to keep the trail in sync with the visible UI (TUI-063).
//
// Crumb mappings:
//
//	FocusClaude + not streaming → ["Claude", "Conversation"]
//	FocusClaude + streaming     → ["Claude", "Streaming..."]
//	FocusAgents + RPMAgents     → ["Agents", "Tree"]
//	FocusAgents + RPMDashboard  → ["Dashboard", "Overview"]
//	FocusAgents + RPMSettings   → ["Settings", "Display"]
//	FocusAgents + RPMPlanPreview → ["Plan", "Preview"]
//	FocusAgents + RPMTelemetry  → ["Telemetry", "Overview"]
func (m *AppModel) updateBreadcrumbs() {
	if m.shared == nil || m.shared.breadcrumb == nil {
		return
	}

	var crumbs []string

	switch m.focus {
	case FocusClaude:
		streaming := m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming()
		if streaming {
			crumbs = []string{"Claude", "Streaming..."}
		} else {
			crumbs = []string{"Claude", "Conversation"}
		}

	case FocusAgents:
		switch m.rightPanelMode {
		case RPMAgents:
			crumbs = []string{"Agents", "Tree"}
		case RPMDashboard:
			crumbs = []string{"Dashboard", "Overview"}
		case RPMSettings:
			crumbs = []string{"Settings", "Display"}
		case RPMPlanPreview:
			crumbs = []string{"Plan", "Preview"}
		case RPMTelemetry:
			crumbs = []string{"Telemetry", "Overview"}
		default:
			crumbs = []string{"Agents", m.rightPanelMode.String()}
		}

	default:
		crumbs = []string{m.focus.String()}
	}

	m.shared.breadcrumb.SetCrumbs(crumbs)
}

// updateHintContext updates the hint bar context based on the current
// application state.  It is called after focus changes and after overlays
// close to keep the hint bar in sync with the visible UI (TUI-060).
//
// Priority (highest to lowest):
//  1. Plan view modal active → "plan"
//  2. Modal queue active     → "modal"
//  3. Search overlay active  → "search"
//  4. Settings panel focused → "settings"
//  5. Default               → "main"
func (m *AppModel) updateHintContext() {
	if m.shared == nil || m.shared.hintBar == nil {
		return
	}
	switch {
	case m.shared.planViewModal.IsActive():
		m.shared.hintBar.SetContext("plan")
	case m.shared.modalQueue != nil && m.shared.modalQueue.IsActive():
		m.shared.hintBar.SetContext("modal")
	case m.shared.searchOverlay != nil && m.shared.searchOverlay.IsActive():
		m.shared.hintBar.SetContext("search")
	case m.rightPanelMode == RPMSettings:
		m.shared.hintBar.SetContext("settings")
	default:
		m.shared.hintBar.SetContext("main")
	}
}
