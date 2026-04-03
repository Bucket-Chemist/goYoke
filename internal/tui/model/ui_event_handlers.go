// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all UI and bridge event handlers for AppModel's Update
// method, plus the session persistence helper. Extracted from app.go as part
// of TUI-043.
package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
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

	m.propagateContentSizes()

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

	// Ensure focus state is propagated on startup. WindowSizeMsg is the first
	// message delivered by Bubbletea, so this is the earliest point where we
	// can focus the Claude panel's text input for immediate typing.
	m.syncFocusState()

	return m, nil
}

// propagateContentSizes recomputes the layout dimensions from the current
// terminal size and propagates the new content heights to all child components.
//
// This MUST be called whenever the layout geometry changes — not just on
// tea.WindowSizeMsg but also when the taskboard appears/disappears, because
// the taskboard's dynamic height affects the available content area for the
// Claude panel, agent tree, and all right-panel components.
//
// Without this re-propagation, components retain stale heights from the last
// WindowSizeMsg and render more rows than the layout expects, causing the
// banner and tab bar to be pushed off the top of the screen.
func (m *AppModel) propagateContentSizes() {
	dims := m.computeLayout()
	drawerH, _ := m.computeDrawerLayout(dims)
	mainH := dims.contentHeight - drawerH
	if mainH < 1 {
		mainH = 1
	}

	if m.shared.claudePanel != nil {
		// Subtract separatorHeight so the panel's internal viewport is sized to
		// contentHeight-2 (one row for the separator, one for the input line),
		// leaving the layout composition in renderLeftPanel to fill all rows.
		m.shared.claudePanel.SetSize(dims.leftWidth, dims.contentHeight-separatorHeight)
	}
	m.agentTree.SetSize(dims.rightWidth, mainH/2)
	m.agentDetail.SetSize(dims.rightWidth, mainH/2)
	if m.shared.toasts != nil {
		m.shared.toasts.SetSize(m.width, m.height)
	}
	if m.shared.teamList != nil {
		m.shared.teamList.SetSize(dims.rightWidth, mainH)
	}

	// Right-panel components (TUI-032).
	if m.shared.dashboard != nil {
		m.shared.dashboard.SetSize(dims.rightWidth, mainH)
	}
	if m.shared.settings != nil {
		m.shared.settings.SetSize(dims.rightWidth, mainH)
	}
	if m.shared.telemetry != nil {
		m.shared.telemetry.SetSize(dims.rightWidth, mainH)
	}
	if m.shared.planPreview != nil {
		m.shared.planPreview.SetSize(dims.rightWidth, mainH)
	}
	if m.shared.drawerStack != nil {
		// Size the teams health widget and refresh the teams drawer content.
		if m.shared.teamsHealth != nil {
			m.shared.teamsHealth.SetSize(dims.rightWidth-4, drawerH-5) // inner: -4 width (drawer+panel borders), -5 height (borders+header+divider+footer)
			m.shared.teamsHealth.SetTier(dims.tier)
			if m.shared.teamsHealth.HasData() {
				m.shared.drawerStack.RefreshTeamsContent(m.shared.teamsHealth.View())
			} else {
				m.shared.drawerStack.ClearTeamsContent()
			}
		}
		m.shared.drawerStack.SetSize(dims.rightWidth, drawerH)
	}
	if m.shared.taskBoard != nil {
		m.shared.taskBoard.SetSize(m.width, m.height)
	}

	// Responsive layout tier to all tier-aware widgets (TUI-058).
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

// handleBridgeModalRequest handles BridgeModalRequestMsg: routes to the options
// drawer on Standard+ tiers, or falls back to the ModalQueue overlay on Compact.
func (m AppModel) handleBridgeModalRequest(msg BridgeModalRequestMsg) (tea.Model, tea.Cmd) {
	// TDS-006: Route to options drawer when available and not Compact tier.
	dims := m.computeLayout()
	if dims.tier != LayoutCompact &&
		m.shared != nil && m.shared.drawerStack != nil &&
		len(msg.Options) > 0 {
		m.shared.drawerStack.SetActiveModal(msg.RequestID, msg.Message, msg.Options)
		return m, nil
	}

	// Fallback: existing ModalQueue/PermissionHandler path.
	if m.shared != nil && m.shared.permHandler != nil {
		cmd := m.shared.permHandler.HandleBridgeRequest(
			msg.RequestID, msg.Message, msg.Options,
		)
		return m, cmd
	}
	return m, nil
}

// handleCLIPermissionRequest handles CLIPermissionRequestMsg: presents a
// permission modal with Allow/Deny/Allow for Session options.
func (m AppModel) handleCLIPermissionRequest(msg CLIPermissionRequestMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.permHandler == nil {
		return m, nil
	}

	// Build a human-readable message showing the tool name and input.
	message := fmt.Sprintf("Tool Permission Required\n\nTool: %s", msg.ToolName)
	if len(msg.ToolInput) > 0 {
		// Try to extract "command" field for Bash, fall back to raw JSON.
		var input map[string]interface{}
		if err := json.Unmarshal(msg.ToolInput, &input); err == nil {
			if cmd, ok := input["command"].(string); ok {
				message += fmt.Sprintf("\nCommand: %s", cmd)
			}
		}
	}

	options := []string{"Allow", "Deny", "Allow for Session"}

	// Route through PermissionHandler using the FlowToolPermission type.
	cmd := m.shared.permHandler.HandlePermGateRequest(
		msg.RequestID, message, options, msg.TimeoutMS,
	)
	return m, cmd
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

	// Interrupt confirmation: fire the CLI interrupt only when the user
	// selected "Yes" (not cancelled and value == "Yes").
	if msg.RequestID == interruptConfirmRequestID {
		if !msg.Response.Cancelled && msg.Response.Value == "Yes" {
			// Clear streaming state immediately — don't wait for ResultEvent
			// which may never arrive if the CLI is stuck on an MCP tool call
			// (e.g. spawn_agent blocked on cmd.Wait for a Setsid subprocess).
			m.statusLine.Streaming = false

			// Tell claude panel streaming is done so input unblocks.
			// panel.go:448 gates Enter on m.streaming; sending ResultMsg{}
			// sets it to false.
			if m.shared.claudePanel != nil {
				m.shared.claudePanel.HandleMsg(ResultMsg{})
			}

			// Kill all running spawned agent processes whose PIDs are
			// tracked in the registry. Spawned agents use Setsid: true
			// and are unreachable by SIGINT to the CLI process group.
			m.killRunningAgents()

			var toastCmd tea.Cmd
			if m.shared.toasts != nil {
				toastCmd = m.shared.toasts.HandleMsg(ToastMsg{
					Text:  "Interrupted active agent",
					Level: ToastLevelWarn,
				})
			}

			// Restart the CLI driver. SIGINT to the process group kills
			// both the claude subprocess and gofortress-mcp (Go default
			// SIGINT = exit). The old driver is in DriverDead state and
			// Start() requires DriverIdle, so we must create a fresh one.
			// This mirrors the provider_switch.go and perm_mode.go patterns.
			var startCmd tea.Cmd
			if m.shared.cliDriver != nil {
				_ = m.shared.cliDriver.Shutdown()

				opts := m.shared.baseCLIOpts // value copy preserves Verbose, Debug, MCPConfigPath, etc.
				opts.SessionID = m.sessionID // resume same session
				opts.Model = m.activeModel
				if m.shared.providerState != nil {
					cfg := m.shared.providerState.GetActiveConfig()
					opts.AdapterPath = cfg.AdapterPath
					opts.ProjectDir = m.shared.providerState.GetActiveProjectDir()
				}

				newDriver := cli.NewCLIDriver(opts)
				m.shared.cliDriver = newDriver
				if m.shared.claudePanel != nil {
					m.shared.claudePanel.SetSender(newDriver)
				}
				m.reconnectCount = 0
				m.reconnectSeq++
				startCmd = newDriver.Start()
			}

			return m, tea.Batch(toastCmd, startCmd)
		}
		return m, nil
	}

	result, cmd := m.shared.permHandler.HandleResponse(msg)
	if result != nil {
		// Check if this was a permission gate flow.
		if m.shared.permHandler.WasPermGateFlow(result.RequestID) {
			if m.shared.bridge != nil {
				decision := mapPermGateDecision(result.Value, result.Cancelled)
				m.shared.bridge.ResolvePermGate(result.RequestID, decision)
			}
		} else {
			// Existing modal flow — deliver via ResolveModalSimple.
			if m.shared.bridge != nil {
				value := result.Value
				if result.Cancelled {
					value = ""
				}
				m.shared.bridge.ResolveModalSimple(result.RequestID, value)
			}
		}
	}
	return m, cmd
}

// mapPermGateDecision converts modal option labels to permission decisions.
func mapPermGateDecision(value string, cancelled bool) string {
	if cancelled {
		return "deny"
	}
	switch value {
	case "Allow":
		return "allow"
	case "Allow for Session":
		return "allow_session"
	default:
		return "deny"
	}
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
			ID:          msg.AgentID,
			AgentType:   msg.AgentType,
			ParentID:    parentID,
			Model:       msg.Model,
			Tier:        msg.Tier,
			Description: msg.Description,
			Conventions: msg.Conventions,
			Prompt:      msg.Prompt,
			Status:      state.StatusRunning,
			StartedAt:   time.Now(),
		})
	case AgentUpdatedMsg:
		_ = m.shared.agentRegistry.Update(msg.AgentID, func(a *state.Agent) {
			a.Status = parseAgentStatus(msg.Status)
		})
		// Store PID separately — the second "running" notification from
		// runSubprocess carries the PID but would be rejected by
		// isValidTransition (Running→Running is not valid).
		if msg.PID > 0 {
			m.shared.agentRegistry.SetPID(msg.AgentID, msg.PID)
		}
	case AgentActivityMsg:
		m.shared.agentRegistry.AppendActivity(msg.AgentID, state.AgentActivity{
			Type:      "tool_use",
			ToolName:  msg.ToolName,
			Target:    msg.Target,
			Preview:   msg.Preview,
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
	case "killed":
		return state.StatusKilled
	default:
		return state.StatusPending
	}
}

// killRunningAgents sends SIGTERM to all spawned agent process groups that
// are still in StatusRunning and have a known PID. Called when the user
// confirms the ESC interrupt to clean up subprocesses that use Setsid: true
// and are therefore unreachable by SIGINT to the parent CLI process group.
func (m *AppModel) killRunningAgents() {
	if m.shared == nil || m.shared.agentRegistry == nil {
		return
	}
	m.shared.agentRegistry.KillRunning()
	// Refresh the tree view so killed agents show their new status.
	m.shared.agentRegistry.InvalidateTreeCache()
	m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
	m.statusLine.AgentCount = m.shared.agentRegistry.Count().Total
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

// handleCWDChanged applies the new working directory: calls os.Chdir, sets
// the GOGENT_CWD env var (read by spawner.go for cmd.Dir), and updates the
// status line display.
func (m AppModel) handleCWDChanged(msg CWDChangedMsg) (tea.Model, tea.Cmd) {
	if err := os.Chdir(msg.Path); err != nil {
		log.Printf("[cwd] chdir failed: %v", err)
		return m, nil
	}
	os.Setenv("GOGENT_CWD", msg.Path)
	m.statusLine.CWD = msg.Path
	log.Printf("[cwd] changed to %s", msg.Path)
	return m, func() tea.Msg {
		return ToastMsg{Text: "CWD: " + msg.Path, Level: ToastLevelInfo}
	}
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

	// Push plan content to drawer if available (TDS-007).
	if msg.Active && m.shared != nil && m.shared.drawerStack != nil {
		if m.shared.planPreview != nil {
			planContent := m.shared.planPreview.Content()
			if planContent != "" {
				m.shared.drawerStack.SetPlanContent(planContent)
			}
		}
	}
	// Clear plan drawer when plan mode deactivates.
	if !msg.Active && m.shared != nil && m.shared.drawerStack != nil {
		m.shared.drawerStack.ClearPlanContent()
	}

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

// handleModelSwitchRequest processes a /model command from the Claude panel.
//
// When ModelID is empty it lists available models as a system message.
// When ModelID is set it validates the model against the active provider,
// guards against switching while streaming, updates ProviderState, and
// restarts the CLI driver.
func (m AppModel) handleModelSwitchRequest(msg ModelSwitchRequestMsg) (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.providerState == nil {
		return m, nil
	}

	ps := m.shared.providerState
	cfg := ps.GetActiveConfig()

	// No arg: list available models for the active provider.
	if msg.ModelID == "" {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Available models for %s:\n", cfg.Name))
		currentModel := ps.GetActiveModel()
		for _, mc := range cfg.Models {
			marker := "  "
			if mc.ID == currentModel {
				marker = "▸ "
			}
			sb.WriteString(fmt.Sprintf("%s%s — %s\n", marker, mc.ID, mc.Description))
		}
		sb.WriteString("\nUsage: /model <name>")

		if m.shared.claudePanel != nil {
			m.shared.claudePanel.AppendSystemMessage(sb.String())
		}
		return m, nil
	}

	// Guard: refuse while streaming — the CLI restart would kill the active response.
	if m.shared.claudePanel != nil && m.shared.claudePanel.IsStreaming() {
		return m, func() tea.Msg {
			return ToastMsg{
				Text:  "Cannot switch model while streaming. Wait for the response to finish.",
				Level: ToastLevelWarn,
			}
		}
	}

	// Validate that the requested model belongs to the active provider.
	var found bool
	var displayName string
	for _, mc := range cfg.Models {
		if mc.ID == msg.ModelID {
			found = true
			displayName = mc.DisplayName
			break
		}
	}
	if !found {
		ids := make([]string, 0, len(cfg.Models))
		for _, mc := range cfg.Models {
			ids = append(ids, mc.ID)
		}
		return m, func() tea.Msg {
			return ToastMsg{
				Text:  fmt.Sprintf("Unknown model %q for %s. Valid: %s", msg.ModelID, cfg.Name, strings.Join(ids, ", ")),
				Level: ToastLevelError,
			}
		}
	}

	// Already on this model — no-op.
	if ps.GetActiveModel() == msg.ModelID {
		return m, func() tea.Msg {
			return ToastMsg{
				Text:  fmt.Sprintf("Already using %s.", displayName),
				Level: ToastLevelInfo,
			}
		}
	}

	// Persist current session ID so the new driver can resume.
	if m.sessionID != "" {
		ps.SetSessionID(m.sessionID)
	}

	// Set the new model in provider state.
	if err := ps.SetActiveModel(msg.ModelID); err != nil {
		return m, func() tea.Msg {
			return ToastMsg{
				Text:  fmt.Sprintf("Model switch failed: %v", err),
				Level: ToastLevelError,
			}
		}
	}

	// Restart the CLI driver with the new model.
	model, cmd := m.restartCLIDriver()
	appModel := model.(AppModel)

	// Emit a toast confirming the switch.
	toastCmd := func() tea.Msg {
		return ToastMsg{
			Text:  fmt.Sprintf("Switched to %s for %s.", displayName, cfg.Name),
			Level: ToastLevelSuccess,
		}
	}

	return appModel, tea.Batch(cmd, toastCmd)
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
