// Package model defines shared state types for the goYoke TUI.
// This file contains all keyboard event handlers for AppModel.
// Extracted from app.go as part of TUI-043.
package model

import (
	"fmt"
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/agents"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/drawer"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/modals"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// interruptConfirmRequestID is the modal request ID used for the ESC-while-streaming
// interrupt confirmation dialog. It is matched in handleModalResponse to execute
// the actual CLI interrupt only after the user selects "Yes".
const interruptConfirmRequestID = "interrupt-confirm"

// agentKillConfirmPrefix is the ModalRequest.ID prefix for the Ctrl+X surgical
// agent kill confirmation. The agent ID is appended after the prefix so that
// handleModalResponse can extract it without any additional state.
const agentKillConfirmPrefix = "agent-kill-confirm:"

// handleKey routes a KeyMsg based on modal and focus state.
func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handled, model, cmd := m.handleOverlayKey(msg); handled {
		return model, cmd
	}
	if model, cmd := m.handleGlobalKey(msg); model != nil {
		return model, cmd
	}
	return m.handleFocusedKey(msg)
}

// handleOverlayKey checks whether any overlay (modal, search, vim, etc.) is
// active and should capture the key exclusively. Returns (true, ...) when
// the key was consumed so handleKey can return immediately.
func (m AppModel) handleOverlayKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	if m.shared == nil {
		return false, m, nil
	}

	if m.shared.helpModal.IsActive() {
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("help")
		}
		updated, c := m.shared.helpModal.Update(msg)
		m.shared.helpModal = updated
		if !m.shared.helpModal.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	if m.shared.planViewModal.IsActive() {
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("plan")
		}
		updated, c := m.shared.planViewModal.Update(msg)
		m.shared.planViewModal = updated
		if !m.shared.planViewModal.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	if m.shared.optionsViewModal.IsActive() {
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("options")
		}
		updated, c := m.shared.optionsViewModal.Update(msg)
		m.shared.optionsViewModal = updated
		if !m.shared.optionsViewModal.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	if m.shared.modelModal.IsActive() {
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("model")
		}
		updated, c := m.shared.modelModal.Update(msg)
		m.shared.modelModal = updated
		if !m.shared.modelModal.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	if m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		if m.shared.hintBar != nil {
			m.shared.hintBar.SetContext("modal")
		}
		result, c := m.handleModalKey(msg)
		return true, result, c
	}

	if m.shared.cwdSelector != nil && m.shared.cwdSelector.IsActive() {
		c := m.shared.cwdSelector.HandleMsg(msg)
		if !m.shared.cwdSelector.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	if m.shared.searchOverlay != nil && m.shared.searchOverlay.IsActive() {
		c := m.shared.searchOverlay.HandleMsg(msg)
		if !m.shared.searchOverlay.IsActive() {
			m.updateHintContext()
		}
		return true, m, c
	}

	// Vim mode (TUI-062): VimNormal consumes navigation keys; VimInsert passes through.
	if m.keys.VimEnabled {
		if consumed, result, c := m.handleVimKey(msg); consumed {
			return true, result, c
		}
	}

	return false, m, nil
}

// handleGlobalKey checks keys that should work regardless of focus. Returns a
// non-nil tea.Model when the key was consumed. Returns (nil, nil) to signal
// that the key should fall through to handleFocusedKey (only the Interrupt key
// when not streaming does this).
func (m AppModel) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Global.ForceQuit):
		if m.shutdownInProgress {
			return m, tea.Quit
		}
		m.shutdownInProgress = true
		if m.shared != nil && m.shared.shutdownFunc != nil {
			shutdownFn := m.shared.shutdownFunc
			return m, func() tea.Msg {
				err := shutdownFn()
				return ShutdownCompleteMsg{Err: err}
			}
		}
		m.saveSession()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Global.ToggleFocus):
		var expanded []string
		if m.shared != nil && m.shared.drawerStack != nil {
			expanded = m.shared.drawerStack.ExpandedDrawers()
		}
		ring := FocusRing(expanded)
		m.focus = FocusNextInRing(m.focus, ring)
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		m.propagateContentSizes()
		return m, tabFlashCmd(int(m.activeTab))

	case key.Matches(msg, m.keys.Global.ReverseToggleFocus):
		var expanded []string
		if m.shared != nil && m.shared.drawerStack != nil {
			expanded = m.shared.drawerStack.ExpandedDrawers()
		}
		ring := FocusRing(expanded)
		m.focus = FocusPrevInRing(m.focus, ring)
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		m.propagateContentSizes()
		return m, tabFlashCmd(int(m.activeTab))

	case key.Matches(msg, m.keys.Global.CycleRightPanel):
		m.rightPanelMode = NextRightPanelMode(m.rightPanelMode)
		m.updateHintContext()
		m.updateBreadcrumbs()
		return m, nil

	case key.Matches(msg, m.keys.Global.CycleProvider):
		if m.shared != nil && m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming() {
			return m, nil
		}
		m.providerSwitchSeq++
		seq := m.providerSwitchSeq
		return m, tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
			return ProviderSwitchExecuteMsg{Seq: seq}
		})

	case key.Matches(msg, m.keys.Global.CyclePermMode):
		if m.shared != nil && m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming() {
			return m, nil
		}
		return m.handlePermModeCycle()

	case key.Matches(msg, m.keys.Global.ToggleTaskBoard):
		if m.shared != nil && m.shared.taskBoard != nil {
			m.shared.taskBoard.Toggle()
			m.propagateContentSizes()
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ViewPlan):
		if m.shared != nil && m.shared.planPreview != nil {
			markdown := m.shared.planPreview.Content()
			if markdown != "" {
				m.shared.planViewModal.SetContent(markdown, m.width)
				m.shared.planViewModal.SetSize(m.width, m.height)
				m.shared.planViewModal.Show()
				if m.shared.hintBar != nil {
					m.shared.hintBar.SetContext("plan")
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ViewOptions):
		if m.shared != nil && m.shared.drawerStack != nil {
			cmd := m.shared.drawerStack.HandleKey(string(drawer.DrawerOptions), msg)
			return m, cmd
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.Interrupt):
		// Escape while streaming: confirm before sending SIGINT. Uses
		// statusLine.Streaming (not claudePanel.IsStreaming) because the panel
		// flag is only true during LLM text bursts, not during tool execution.
		if m.statusLine.Streaming {
			if m.shared != nil && m.shared.modalQueue != nil {
				m.shared.modalQueue.Push(modals.ModalRequest{
					ID:      interruptConfirmRequestID,
					Type:    modals.Confirm,
					Header:  "Interrupt Agent",
					Message: "Interrupt the active agent?",
				})
				if !m.shared.modalQueue.IsActive() {
					m.shared.modalQueue.Activate()
				}
			}
			return m, nil
		}
		// Not streaming — signal not consumed; caller routes to handleFocusedKey.
		return nil, nil

	case key.Matches(msg, m.keys.Global.Search):
		if m.shared != nil && m.shared.searchOverlay != nil {
			m.shared.searchOverlay.SetSize(m.width, m.height)
			m.shared.searchOverlay.Activate()
			if m.shared.hintBar != nil {
				m.shared.hintBar.SetContext("search")
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ChangeCWD):
		if m.shared != nil && m.shared.cwdSelector != nil {
			m.shared.cwdSelector.SetSize(m.width, m.height)
			m.shared.cwdSelector.Show()
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ToggleMouse):
		m.mouseEnabled = !m.mouseEnabled
		m.statusLine.MouseEnabled = m.mouseEnabled
		if m.mouseEnabled {
			return m, tea.EnableMouseCellMotion
		}
		return m, tea.DisableMouse

	case key.Matches(msg, m.keys.Global.ToggleSimpleMode):
		m.simpleMode = !m.simpleMode
		m.propagateContentSizes()
		return m, nil

	case key.Matches(msg, m.keys.Global.ToggleFigures):
		if m.shared != nil && m.shared.drawerStack != nil {
			m.shared.drawerStack.ToggleFiguresDrawer()
			m.propagateContentSizes()
		}
		return m, nil

	case key.Matches(msg, m.keys.Global.ShowHelp):
		if m.shared != nil {
			m.shared.helpModal.SetSize(m.width, m.height)
			m.shared.helpModal.Show()
			if m.shared.hintBar != nil {
				m.shared.hintBar.SetContext("help")
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Tab.TabChat),
		key.Matches(msg, m.keys.Tab.TabAgentConfig),
		key.Matches(msg, m.keys.Tab.TabTeamConfig),
		key.Matches(msg, m.keys.Tab.TabTelemetry):
		if m.tabBar != nil {
			cmd := m.tabBar.HandleMsg(msg)
			m.activeTab = m.tabBar.ActiveTab()
			if m.activeTab == TabTeamConfig {
				m.rightPanelMode = RPMTeams
			} else if m.rightPanelMode == RPMTeams {
				m.rightPanelMode = RPMAgents
			}
			m.updateHintContext()
			m.updateBreadcrumbs()
			return m, cmd
		}
		return m, nil
	}

	return nil, nil
}

// handleFocusedKey routes keys to the currently focused panel.
func (m AppModel) handleFocusedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Figures drawer: Tab/Shift+Tab cycle diagrams rather than advancing
	// the global focus ring (CM-014).
	if m.focus == FocusFiguresDrawer && (msg.String() == "tab" || msg.String() == "shift+tab") {
		return m.handleFiguresDrawerKey(msg)
	}
	switch m.focus {
	case FocusClaude:
		return m.handleClaudeKey(msg)
	case FocusAgents:
		return m.handleAgentsKey(msg)
	case FocusPlanDrawer, FocusOptionsDrawer, FocusTeamsDrawer:
		return m.handleDrawerKey(msg)
	case FocusFiguresDrawer:
		return m.handleFiguresDrawerKey(msg)
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

// handleClaudeKey processes key events when the left panel holds focus.
// The target widget depends on the active tab.
func (m AppModel) handleClaudeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case TabChat:
		if m.shared != nil && m.shared.claudePanel != nil {
			cmd := m.shared.claudePanel.HandleMsg(msg)
			return m, cmd
		}
	case TabTeamConfig:
		if m.shared != nil && m.shared.teamList != nil {
			cmd := m.shared.teamList.HandleMsg(msg)
			return m, cmd
		}
	}
	return m, nil
}

// handleAgentsKey processes key events when the agent tree holds focus.
// When the right panel is showing the dashboard it forwards events to the
// dashboard component instead of the agent tree.
func (m AppModel) handleAgentsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+X: surgical agent kill — intercept BEFORE detail panel
	// forwarding so it works regardless of which child has focus.
	if key.Matches(msg, m.keys.Agent.AgentKill) {
		return m.handleAgentKillKey()
	}

	// alt+d: cycle tree density (UX-022). Works in RPMAgents regardless of
	// whether the tree or detail sub-panel has focus.
	if key.Matches(msg, m.keys.Agent.CycleDensity) {
		m.agentTree.CycleDensity()
		return m, nil
	}

	switch m.rightPanelMode {
	case RPMDashboard:
		if m.shared != nil && m.shared.dashboard != nil {
			cmd := m.shared.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil
	default:
		// Forward keys to detail panel when it has focus, otherwise to tree.
		if m.agentDetail.HasAgent() {
			result, cmd := m.agentDetail.Update(msg)
			if updated, ok := result.(tea.Model); ok {
				if detail, ok := updated.(agents.AgentDetailModel); ok {
					m.agentDetail = detail
				}
			}
			if cmd != nil {
				return m, cmd
			}
		}
		result, cmd := m.agentTree.Update(msg)
		if updated, ok := result.(agents.AgentTreeModel); ok {
			m.agentTree = updated
		}
		return m, cmd
	}
}

// handleAgentKillKey handles Ctrl+X when the agent tree has focus. It looks up
// the selected agent and, if it is running with a known PID, pushes a
// confirmation modal. The modal response is handled in handleModalResponse via
// the agentKillConfirmPrefix on the request ID.
func (m AppModel) handleAgentKillKey() (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.agentRegistry == nil || m.shared.modalQueue == nil {
		return m, nil
	}

	id := m.agentTree.SelectedID()
	if id == "" {
		return m, nil
	}

	registry := m.shared.agentRegistry
	agent := registry.Get(id)
	if agent == nil || agent.Status != state.StatusRunning || agent.PID <= 0 {
		return m, nil
	}

	desc := fmt.Sprintf(
		"Type: %s\nID: %.12s…\nPID: %d\nStatus: %s",
		agent.AgentType, agent.ID, agent.PID, agent.Status,
	)
	if agent.Description != "" {
		desc += fmt.Sprintf("\nTask: %s", agent.Description)
	}
	childCount := registry.CountRunningDescendants(id)
	if childCount > 0 {
		desc += fmt.Sprintf("\n\n⚠ This will also kill %d running child agent(s)", childCount)
	}

	req := modals.ModalRequest{
		ID:      agentKillConfirmPrefix + id,
		Type:    modals.Confirm,
		Header:  "Kill Agent?",
		Message: desc,
		Options: []string{"Yes, kill", "Cancel"},
	}

	m.shared.modalQueue.Push(req)
	if !m.shared.modalQueue.IsActive() {
		m.shared.modalQueue.Activate()
	}
	return m, nil
}

// handleDrawerKey routes keyboard events to the focused drawer via the drawer
// stack. The focusedDrawer ID is derived from the current FocusTarget.
// If the focused drawer is minimized (no longer in the expanded ring) after
// handling the key, focus snaps back to FocusClaude (TDS-008).
func (m AppModel) handleDrawerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.drawerStack == nil {
		return m, nil
	}
	var drawerID string
	switch m.focus {
	case FocusPlanDrawer:
		drawerID = "plan"
	case FocusOptionsDrawer:
		drawerID = "options"
	case FocusTeamsDrawer:
		drawerID = "teams"
	default:
		return m, nil
	}
	cmd := m.shared.drawerStack.HandleKey(drawerID, msg)

	// Edge case: if the focused drawer was just minimized (Esc), snap focus
	// back to FocusClaude.
	expanded := m.shared.drawerStack.ExpandedDrawers()
	ring := FocusRing(expanded)
	found := false
	for _, t := range ring {
		if t == m.focus {
			found = true
			break
		}
	}
	if !found {
		m.focus = FocusClaude
		m.syncFocusState()
		m.updateHintContext()
		m.updateBreadcrumbs()
		m.propagateContentSizes()
	}

	return m, cmd
}

// handleFiguresDrawerKey processes key events when the figures drawer holds
// focus. Tab/Shift+Tab cycle the selected diagram. Enter opens the current
// diagram in a browser. y copies the source to the clipboard (best-effort).
// All other keys are forwarded to the drawer for scroll and esc handling.
func (m AppModel) handleFiguresDrawerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.drawerStack == nil {
		return m, nil
	}

	switch msg.String() {
	case "tab":
		n := len(m.shared.figuresState.Diagrams)
		if n > 0 {
			m.shared.figuresState.Selected = (m.shared.figuresState.Selected + 1) % n
			m.shared.drawerStack.RefreshFiguresContent(
				drawer.FormatFiguresContent(m.shared.figuresState),
			)
		}
		return m, nil

	case "shift+tab":
		n := len(m.shared.figuresState.Diagrams)
		if n > 0 {
			m.shared.figuresState.Selected = (m.shared.figuresState.Selected - 1 + n) % n
			m.shared.drawerStack.RefreshFiguresContent(
				drawer.FormatFiguresContent(m.shared.figuresState),
			)
		}
		return m, nil

	case "enter":
		if len(m.shared.figuresState.Diagrams) > 0 {
			d := m.shared.figuresState.Diagrams[m.shared.figuresState.Selected]
			if content, err := os.ReadFile(d.Path); err == nil {
				_ = drawer.OpenInBrowser(d.Name, string(content))
			}
		}
		return m, nil

	case "y":
		if len(m.shared.figuresState.Diagrams) > 0 {
			d := m.shared.figuresState.Diagrams[m.shared.figuresState.Selected]
			if content, err := os.ReadFile(d.Path); err == nil {
				_ = clipboard.WriteAll(string(content))
			}
		}
		return m, nil

	default:
		cmd := m.shared.drawerStack.HandleKey(string(drawer.DrawerFigures), msg)
		// If Esc minimized the drawer, snap focus back to Claude.
		expanded := m.shared.drawerStack.ExpandedDrawers()
		ring := FocusRing(expanded)
		found := false
		for _, t := range ring {
			if t == FocusFiguresDrawer {
				found = true
				break
			}
		}
		if !found {
			m.focus = FocusClaude
			m.syncFocusState()
			m.updateHintContext()
			m.updateBreadcrumbs()
			m.propagateContentSizes()
		}
		return m, cmd
	}
}

// syncFocusState propagates the current focus state to child components.
func (m *AppModel) syncFocusState() {
	if m.shared != nil && m.shared.claudePanel != nil {
		m.shared.claudePanel.SetFocused(m.focus == FocusClaude && m.activeTab == TabChat)
	}
	m.agentTree.SetFocused(m.focus == FocusAgents && m.rightPanelMode == RPMAgents)
	if m.shared != nil && m.shared.dashboard != nil {
		m.shared.dashboard.SetFocused(m.focus == FocusAgents && m.rightPanelMode == RPMDashboard)
	}
	// Drawer focus (TDS-008).
	if m.shared != nil && m.shared.drawerStack != nil {
		m.shared.drawerStack.SetPlanFocused(m.focus == FocusPlanDrawer)
		m.shared.drawerStack.SetOptionsFocused(m.focus == FocusOptionsDrawer)
		m.shared.drawerStack.SetTeamsFocused(m.focus == FocusTeamsDrawer)
		m.shared.drawerStack.SetFiguresFocused(m.focus == FocusFiguresDrawer)
	}
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
		case RPMTeams:
			crumbs = []string{"Teams", "Detail"}
		default:
			crumbs = []string{"Agents", m.rightPanelMode.String()}
		}

	case FocusPlanDrawer:
		crumbs = []string{"Drawer", "Plan"}

	case FocusOptionsDrawer:
		crumbs = []string{"Drawer", "Options"}

	case FocusTeamsDrawer:
		crumbs = []string{"Drawer", "Teams"}

	case FocusFiguresDrawer:
		crumbs = []string{"Drawer", "Figures"}

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
	case m.shared.helpModal.IsActive():
		m.shared.hintBar.SetContext("help")
	case m.shared.planViewModal.IsActive():
		m.shared.hintBar.SetContext("plan")
	case m.shared.optionsViewModal.IsActive():
		m.shared.hintBar.SetContext("options")
	case m.shared.modelModal.IsActive():
		m.shared.hintBar.SetContext("model")
	case m.shared.modalQueue != nil && m.shared.modalQueue.IsActive():
		m.shared.hintBar.SetContext("modal")
	case m.shared.searchOverlay != nil && m.shared.searchOverlay.IsActive():
		m.shared.hintBar.SetContext("search")
	case m.rightPanelMode == RPMSettings:
		m.shared.hintBar.SetContext("settings")
	case m.focus == FocusPlanDrawer || m.focus == FocusOptionsDrawer || m.focus == FocusTeamsDrawer || m.focus == FocusFiguresDrawer:
		m.shared.hintBar.SetContext("drawer")
	default:
		m.shared.hintBar.SetContext("main")
	}
}
