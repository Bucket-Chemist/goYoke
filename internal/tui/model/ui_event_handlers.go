// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all UI and bridge event handlers for AppModel's Update
// method, plus the session persistence helper. Extracted from app.go as part
// of TUI-043.
package model

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/settingstree"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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

	// Propagate responsive layout tier to all tier-aware widgets (TUI-058).
	tier := dims.tier
	if m.shared.claudePanel != nil {
		m.shared.claudePanel.SetTier(tier)
	}
	if m.shared.toasts != nil {
		m.shared.toasts.SetTier(tier)
	}
	if m.shared.dashboard != nil {
		m.shared.dashboard.SetTier(tier)
	}
	if m.shared.settings != nil {
		m.shared.settings.SetTier(tier)
	}
	if m.shared.telemetry != nil {
		m.shared.telemetry.SetTier(tier)
	}
	if m.shared.planPreview != nil {
		m.shared.planPreview.SetTier(tier)
	}
	if m.shared.taskBoard != nil {
		m.shared.taskBoard.SetTier(tier)
	}

	// Propagate size to the search overlay so centering is correct (TUI-059).
	if m.shared.searchOverlay != nil {
		m.shared.searchOverlay.SetSize(msg.Width, msg.Height)
	}

	// Propagate terminal width to the hint bar for truncation (TUI-060).
	if m.shared.hintBar != nil {
		m.shared.hintBar.SetWidth(msg.Width)
	}

	// Propagate terminal width to the breadcrumb trail for truncation (TUI-063).
	if m.shared.breadcrumb != nil {
		m.shared.breadcrumb.SetWidth(msg.Width)
		// Set initial breadcrumb state based on the current focus and panel mode.
		// WindowSizeMsg is the first message after startup, so this ensures crumbs
		// are populated before the first render rather than waiting for a key press.
		m.updateBreadcrumbs()
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
// AgentActivityMsg: writes the incoming data to the registry, then refreshes
// the agent tree view and status line agent count.
//
// C-1 fix (2026-03-25): previous version discarded the message data and only
// refreshed the view from an empty registry. Now each message type is
// type-switched and the corresponding registry method is called.
func (m AppModel) handleAgentRegistryMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.shared.agentRegistry == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case AgentRegisteredMsg:
		// Default empty ParentID to the registry's root agent (matches TS TUI
		// spawnAgent.ts:232 which defaults to zustand.rootAgentId). Without
		// this, agents arrive as orphans — counted but invisible in the tree
		// because buildTree() DFS from root never reaches them.
		parentID := msg.ParentID
		rootID := m.shared.agentRegistry.RootID()
		if parentID == "" && rootID != "" && rootID != msg.AgentID {
			parentID = rootID
		}
		_ = m.shared.agentRegistry.Register(state.Agent{
			ID:        msg.AgentID,
			AgentType: msg.AgentType,
			ParentID:  parentID,
			Status:    state.StatusRunning,
			StartedAt: time.Now(),
		})
	case AgentUpdatedMsg:
		_ = m.shared.agentRegistry.Update(msg.AgentID, func(a *state.Agent) {
			a.Status = parseAgentStatus(msg.Status)
		})
	case AgentActivityMsg:
		m.shared.agentRegistry.SetActivity(msg.AgentID, state.AgentActivity{
			Type:      "tool_use",
			Target:    msg.ToolName,
			Timestamp: time.Now(),
		})
	}

	m.shared.agentRegistry.InvalidateTreeCache()
	m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
	m.statusLine.AgentCount = m.shared.agentRegistry.Count().Total
	return m, nil
}

// parseAgentStatus maps a status string from IPC messages to the typed
// AgentStatus enum. Unknown values default to StatusPending.
func parseAgentStatus(s string) state.AgentStatus {
	switch s {
	case "running":
		return state.StatusRunning
	case "complete":
		return state.StatusComplete
	case "error":
		return state.StatusError
	default:
		return state.StatusPending
	}
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

// handlePlanStep handles PlanStepMsg: updates the plan mode fields on the
// status line and, on the transition from inactive to active, emits a toast
// informing the user about the alt+v shortcut.
func (m AppModel) handlePlanStep(msg PlanStepMsg) (tea.Model, tea.Cmd) {
	wasActive := m.statusLine.PlanActive
	m.statusLine.PlanActive = msg.Active
	m.statusLine.PlanStep = msg.Step
	m.statusLine.PlanTotalSteps = msg.Total

	// Emit a toast only on the inactive → active transition.
	if msg.Active && !wasActive {
		if m.shared != nil && m.shared.toasts != nil {
			cmd := m.shared.toasts.HandleMsg(ToastMsg{
				Text:  "Plan mode active — press alt+v to view plan",
				Level: ToastLevelInfo,
			})
			return m, cmd
		}
	}

	return m, nil
}

// handleTabFlash handles TabFlashMsg: forwards the flash request to the tab
// bar widget so it can schedule its tick-based accent animation (TUI-061).
func (m AppModel) handleTabFlash(msg TabFlashMsg) (tea.Model, tea.Cmd) {
	if m.tabBar == nil {
		return m, nil
	}
	cmd := m.tabBar.HandleMsg(msg)
	return m, cmd
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

// handleSettingChanged handles settingstree.SettingChangedMsg: translates
// settings panel changes into the appropriate model mutations. Currently wires:
//   - "theme"      → builds and activates a new config.Theme via NewTheme()
//   - "ascii_icons" → toggles UseASCII on the active theme
//
// Unknown keys are silently ignored to keep the handler forward-compatible.
func (m AppModel) handleSettingChanged(msg settingstree.SettingChangedMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil {
		return m, nil
	}

	switch msg.Key {
	case "theme":
		var variant config.ThemeVariant
		switch msg.Value {
		case "Dark":
			variant = config.ThemeDark
		case "Light":
			variant = config.ThemeLight
		case "High Contrast":
			variant = config.ThemeHighContrast
		default:
			// Unrecognised theme value — no-op.
			return m, nil
		}
		newTheme := config.NewTheme(variant)
		m.shared.activeTheme = &newTheme
		m.shared.themeVariant = variant

	case "ascii_icons":
		if m.shared.activeTheme != nil {
			m.shared.activeTheme.UseASCII = msg.Value == "on"
		}

	case "vim_keys":
		enabled := msg.Value == "on"
		m.keys.VimEnabled = enabled
		if !enabled {
			// Reset mode so the next enable starts in NORMAL.
			m.keys.VimMode = config.VimNormal
			m.statusLine.VimMode = ""
		} else {
			m.statusLine.VimMode = m.keys.VimMode.String()
		}
		m.statusLine.VimEnabled = enabled
	}

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
