// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all UI and bridge event handlers for AppModel's Update
// method, plus the session persistence helper. Extracted from app.go as part
// of TUI-043.
package model

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// handleWindowSize handles tea.WindowSizeMsg: updates terminal dimensions,
// propagates to all chrome components, and recomputes layout for child panels.
func (m AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true

	// Propagate width to all chrome components.
	m.banner.SetWidth(msg.Width)
	if m.tabBar != nil {
		m.tabBar.SetWidth(msg.Width)
	}
	m.statusLine.SetWidth(msg.Width)
	if m.shared != nil && m.shared.providerTabBar != nil {
		m.shared.providerTabBar.SetWidth(msg.Width)
	}

	// Propagate terminal size to modal queue for correct centering.
	if m.shared != nil && m.shared.modalQueue != nil {
		m.shared.modalQueue.SetTermSize(msg.Width, msg.Height)
	}

	// Propagate size to child components.
	dims := m.computeLayout()
	if m.shared.claudePanel != nil {
		m.shared.claudePanel.SetSize(dims.leftWidth, dims.contentHeight)
	}
	m.agentTree.SetSize(dims.rightWidth, dims.contentHeight/2)
	m.agentDetail.SetSize(dims.rightWidth, dims.contentHeight/2)
	if m.shared.toasts != nil {
		m.shared.toasts.SetSize(msg.Width, msg.Height)
	}
	if m.shared.teamList != nil {
		m.shared.teamList.SetSize(dims.rightWidth, dims.contentHeight)
	}

	// Propagate size to right-panel components (TUI-032).
	if m.shared.dashboard != nil {
		m.shared.dashboard.SetSize(dims.rightWidth, dims.contentHeight)
	}
	if m.shared.settings != nil {
		m.shared.settings.SetSize(dims.rightWidth, dims.contentHeight)
	}
	if m.shared.telemetry != nil {
		m.shared.telemetry.SetSize(dims.rightWidth, dims.contentHeight)
	}
	if m.shared.planPreview != nil {
		m.shared.planPreview.SetSize(dims.rightWidth, dims.contentHeight)
	}
	if m.shared.taskBoard != nil {
		m.shared.taskBoard.SetSize(msg.Width, msg.Height)
	}

	return m, nil
}

// handleProviderSwitchMsg handles ProviderSwitchMsg: delegates to the
// provider-switching flow in provider_switch.go.
func (m AppModel) handleProviderSwitchMsg() (tea.Model, tea.Cmd) {
	return m.handleProviderSwitch()
}

// handleProviderSwitchExecuteMsg handles ProviderSwitchExecuteMsg: applies the
// debounce guard and delegates to the provider-switching flow.
func (m AppModel) handleProviderSwitchExecuteMsg(msg ProviderSwitchExecuteMsg) (tea.Model, tea.Cmd) {
	// Debounce guard: only the most recent timer executes the switch.
	// Earlier timers (lower Seq values) are silently discarded, which
	// prevents rapid Shift+Tab presses from triggering multiple
	// CLI driver shutdown+restart cycles.
	if msg.Seq != m.providerSwitchSeq {
		return m, nil
	}
	return m.handleProviderSwitch()
}

// handleBridgeModalRequest handles BridgeModalRequestMsg: dispatches to the
// permission handler which enqueues the appropriate modal(s).
func (m AppModel) handleBridgeModalRequest(msg BridgeModalRequestMsg) (tea.Model, tea.Cmd) {
	if m.shared != nil && m.shared.permHandler != nil {
		cmd := m.shared.permHandler.HandleBridgeRequest(
			msg.RequestID, msg.Message, msg.Options,
		)
		return m, cmd
	}
	return m, nil
}

// handleModalResponse handles modals.ModalResponseMsg: advances the permission
// flow, resolves the bridge goroutine, and activates the next queued modal.
func (m AppModel) handleModalResponse(msg modals.ModalResponseMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.permHandler == nil {
		return m, nil
	}

	// Advance queue: pop active modal and activate next if any.
	if m.shared.modalQueue != nil {
		m.shared.modalQueue.Resolve(msg.Response)
	}

	result, cmd := m.shared.permHandler.HandleResponse(msg)
	if result != nil {
		// Flow complete — deliver response to the bridge goroutine.
		if m.shared.bridge != nil {
			value := result.Value
			if result.Cancelled {
				value = ""
			}
			m.shared.bridge.ResolveModalSimple(result.RequestID, value)
		}
	}
	return m, cmd
}

// handleAgentRegistryMsg handles AgentRegisteredMsg, AgentUpdatedMsg, and
// AgentActivityMsg: refreshes the agent tree view.
func (m AppModel) handleAgentRegistryMsg() (tea.Model, tea.Cmd) {
	if m.shared.agentRegistry != nil {
		// C-3: invalidate before reading Tree() so the view reflects any
		// registry mutations that occurred before this message was dispatched.
		m.shared.agentRegistry.InvalidateTreeCache()
		m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
	}
	return m, nil
}

// handleAgentSelected handles agents.AgentSelectedMsg: loads the selected
// agent's details into the detail panel.
func (m AppModel) handleAgentSelected(msg agents.AgentSelectedMsg) (tea.Model, tea.Cmd) {
	if m.shared.agentRegistry != nil {
		agent := m.shared.agentRegistry.Get(msg.AgentID)
		if agent != nil {
			m.agentDetail.SetAgent(agent)
		}
	}
	return m, nil
}

// handleToastMsg handles ToastMsg: forwards the toast notification to the
// toast component.
func (m AppModel) handleToastMsg(msg ToastMsg) (tea.Model, tea.Cmd) {
	if m.shared != nil && m.shared.toasts != nil {
		cmd := m.shared.toasts.HandleMsg(msg)
		return m, cmd
	}
	return m, nil
}

// handleShutdownComplete handles ShutdownCompleteMsg: logs any shutdown error
// and exits the program.
func (m AppModel) handleShutdownComplete(msg ShutdownCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		log.Printf("[shutdown] completed with error: %v", msg.Err)
	}
	return m, tea.Quit
}

// handleSessionAutoSave handles SessionAutoSaveMsg: applies the debounce guard
// and triggers a session save.
func (m AppModel) handleSessionAutoSave(msg SessionAutoSaveMsg) (tea.Model, tea.Cmd) {
	// Debounce guard: only the most recent timer fires the save.
	if msg.Seq != m.autoSaveSeq {
		return m, nil
	}
	m.saveSession()
	return m, nil
}

// handleThemeChanged handles ThemeChangedMsg: builds a new Theme for the
// requested variant, stores it in sharedState, and records the variant for
// session persistence.
//
// Propagation strategy: components that hold a pointer to sharedState read
// activeTheme directly on their next render cycle — no additional dispatch is
// needed.  Components without sharedState access continue using package-level
// config defaults until TUI-048/TUI-050 add SetTheme() to their widget
// interfaces.
func (m AppModel) handleThemeChanged(msg ThemeChangedMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil {
		return m, nil
	}
	newTheme := config.NewTheme(msg.Variant)
	m.shared.activeTheme = &newTheme
	m.shared.themeVariant = msg.Variant
	return m, nil
}

// saveSession snapshots the current application state into the session store.
// It persists both the session metadata (cost, provider IDs, model selections)
// and conversation histories for all providers that have messages.
//
// Errors are logged but not returned because saves happen asynchronously from
// debounced timers and during shutdown; there is no actionable recovery path.
func (m AppModel) saveSession() {
	if m.shared == nil || m.shared.sessionStore == nil || m.shared.sessionData == nil {
		return
	}

	sd := m.shared.sessionData

	// Snapshot cost from the tracker.
	if m.shared.costTracker != nil {
		sd.Cost = m.shared.costTracker.GetSessionCost()
	}

	// Snapshot provider state.
	if m.shared.providerState != nil {
		sd.ProviderSessionIDs = m.shared.providerState.ExportSessionIDs()
		sd.ProviderModels = m.shared.providerState.ExportModels()
		sd.ActiveProvider = m.shared.providerState.GetActiveProvider()
	}

	// Snapshot theme variant.  ThemeDark (0) is the default and is omitted
	// from the JSON output by the omitempty tag on SessionData.ThemeVariant.
	sd.ThemeVariant = int(m.shared.themeVariant)

	// Save session metadata.
	if err := m.shared.sessionStore.SaveSession(sd); err != nil {
		log.Printf("[session] save session: %v", err)
	}

	// Save conversation histories for all providers with messages.
	if m.shared.providerState != nil {
		allMsgs := m.shared.providerState.ExportAllMessages()
		for provider, msgs := range allMsgs {
			if err := m.shared.sessionStore.SaveConversationHistory(sd.ID, provider, msgs); err != nil {
				log.Printf("[session] save history provider=%s: %v", provider, err)
			}
		}
	}
}
