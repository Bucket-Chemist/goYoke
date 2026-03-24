// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the root AppModel: the single top-level tea.Model that
// owns all application state and implements The Elm Architecture.
package model

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/banner"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/session"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// Widget interfaces are defined in interfaces.go.
// Layout constants and rendering are in layout.go.
// Provider switch flow is in provider_switch.go.

// ---------------------------------------------------------------------------
// TabID
// ---------------------------------------------------------------------------

// TabID identifies the active top-level tab in the TUI.
type TabID int

const (
	// TabChat is the default conversation tab.
	TabChat TabID = iota

	// TabAgentConfig shows the agent configuration editor.
	TabAgentConfig

	// TabTeamConfig shows the team configuration editor.
	TabTeamConfig

	// TabTelemetry shows the routing-decisions telemetry view.
	TabTelemetry
)

// String returns a human-readable name for the tab.
func (t TabID) String() string {
	switch t {
	case TabChat:
		return "Chat"
	case TabAgentConfig:
		return "Agent Config"
	case TabTeamConfig:
		return "Team Config"
	case TabTelemetry:
		return "Telemetry"
	default:
		return "Unknown"
	}
}

// ---------------------------------------------------------------------------
// DiffEntry
// ---------------------------------------------------------------------------

// DiffEntry holds a structured diff produced by a Write/Edit/Bash tool call.
// The Patch field is the raw structuredPatch JSON from the tool_use_result
// event.  TUI-022 (Claude panel) will render these entries inline.
type DiffEntry struct {
	// FilePath is the absolute path of the file that was modified.
	FilePath string
	// Patch is the raw structuredPatch value from the CLI event.
	Patch json.RawMessage
}

// ---------------------------------------------------------------------------
// sharedState
//
// sharedState holds the mutable external references that must survive the
// value-copy that tea.NewProgram performs on the AppModel. Because AppModel
// follows the Bubbletea convention of value receivers (Update returns a new
// copy), any pointer assigned directly to an AppModel field would be lost
// after the first Update. Placing those pointers in a shared heap-allocated
// struct solves the problem: both main.go and the program's internal copy
// point to the same sharedState.
// ---------------------------------------------------------------------------

// sharedState holds the external component references shared between main.go
// and the AppModel copy held inside tea.Program.
type sharedState struct {
	cliDriver      cliDriverWidget
	bridge         bridgeWidget
	modalQueue     *modals.ModalQueue
	permHandler    *modals.PermissionHandler
	agentRegistry  *state.AgentRegistry
	costTracker    *state.CostTracker
	claudePanel    claudePanelWidget
	toasts         toastWidget
	teamList       teamListWidget
	providerState  *state.ProviderState
	providerTabBar providerTabBarWidget
	// baseCLIOpts holds the CLI driver options supplied at startup (C-2).
	// handleProviderSwitch copies this struct and overrides only the fields
	// that change per provider (Model, SessionID, AdapterPath, ProjectDir,
	// EnvVars), so that Verbose, Debug, PermissionMode, MCPConfigPath, and
	// any other flags are transparently preserved across provider switches.
	baseCLIOpts cli.CLIDriverOpts
	// Right-panel widgets (TUI-032).
	dashboard   dashboardWidget
	settings    settingsWidget
	telemetry   telemetryWidget
	planPreview planPreviewWidget
	taskBoard   taskBoardWidget

	// Session persistence (TUI-033).
	sessionStore *session.Store
	sessionData  *session.SessionData

	// Graceful shutdown (TUI-034).
	// shutdownFunc is called to trigger the sequenced shutdown.  It is a
	// func() error (ShutdownManager.Shutdown) stored as a closure so the
	// model package does not import the lifecycle package.
	shutdownFunc func() error
}

// ---------------------------------------------------------------------------
// AppModel
// ---------------------------------------------------------------------------

// AppModel is the root tea.Model for the GOgent-Fortress TUI.  It owns all
// application state and delegates rendering and key handling to child
// components.
//
// The zero value is not usable; use NewAppModel instead.
type AppModel struct {
	// Terminal state
	width  int
	height int
	ready  bool // set true on first WindowSizeMsg

	// Focus state
	focus          FocusTarget
	rightPanelMode RightPanelMode

	// Tab state
	activeTab TabID

	// Chrome components (fully implemented)
	banner     banner.BannerModel
	tabBar     tabBarWidget
	statusLine statusline.StatusLineModel

	// Child models (directly importable — agents doesn't import model)
	agentTree   agents.AgentTreeModel
	agentDetail agents.AgentDetailModel

	// Diff history (post-hoc diffs from Write/Edit/Bash tool results).
	// TUI-022 renders these inline in the Claude panel.
	diffs []DiffEntry

	// Infrastructure
	keys config.KeyMap

	// Shared external components (see sharedState for rationale).
	// Access via m.shared.cliDriver, m.shared.bridge, etc.
	shared *sharedState

	// Startup / session state (TUI-016).
	cliReady       bool   // true after SystemInitEvent processed
	sessionID      string // from SystemInitEvent
	activeModel    string // from SystemInitEvent
	reconnectCount int    // number of reconnection attempts made

	// Provider switch debounce (R-2).
	// providerSwitchSeq increments on every CycleProvider keypress.
	// Only a ProviderSwitchExecuteMsg whose Seq matches the current counter
	// triggers the actual switch; all earlier timers are silently discarded.
	providerSwitchSeq int

	// reconnectSeq increments on every provider switch so that CLIReconnectMsg
	// timers created before the switch are silently discarded.
	reconnectSeq int

	// autoSaveSeq increments on every cost-change event.  Only the
	// SessionAutoSaveMsg with the highest Seq is executed; earlier timers are
	// discarded (5 s debounce cooldown, TUI-033).
	autoSaveSeq int

	// shutdownInProgress is set true on the first Ctrl+C.  A second Ctrl+C
	// within the shutdown window forces an immediate tea.Quit without waiting
	// for the graceful sequence to complete (TUI-034).
	shutdownInProgress bool
}

// NewAppModel returns an AppModel initialised with sensible defaults:
//   - focus on the Claude panel
//   - right panel showing the agent list
//   - Chat tab active
//   - default keybindings loaded
//
// The tabBar field is not set here because the tabbar package imports the
// model package (for model.TabID), which would create an import cycle.
// Callers that need a live tab bar should call SetTabBar after construction;
// see the application main.go for the canonical wiring.
func NewAppModel() AppModel {
	keys := config.DefaultKeyMap()
	mq := modals.NewModalQueue(keys)
	shared := &sharedState{
		modalQueue:    &mq,
		permHandler:   modals.NewPermissionHandler(&mq),
		agentRegistry: state.NewAgentRegistry(),
		costTracker:   state.NewCostTracker(),
		providerState: state.NewProviderState(),
	}
	return AppModel{
		focus:          FocusClaude,
		rightPanelMode: RPMAgents,
		activeTab:      TabChat,
		keys:           keys,
		banner:         banner.NewBannerModel(0),
		statusLine:     statusline.NewStatusLineModel(0),
		agentTree:      agents.NewAgentTreeModel(),
		agentDetail:    agents.NewAgentDetailModel(),
		shared:         shared,
	}
}

// SetTabBar injects a tab bar component into the model.  This setter exists
// because the tabbar package imports model (for model.TabID), which prevents
// model from importing tabbar directly.  The application entry point calls
// this method after creating both the AppModel and the TabBarModel.
func (m *AppModel) SetTabBar(tb tabBarWidget) {
	m.tabBar = tb
}

// SetCLIDriver injects the CLI driver into the shared state.  Because
// tea.NewProgram copies the AppModel by value, the driver must be stored in
// the shared state pointer so that both the main.go reference and the model
// copy inside tea.Program see the same driver.
func (m *AppModel) SetCLIDriver(d cliDriverWidget) {
	m.shared.cliDriver = d
}

// SetBridge injects the IPC bridge into the shared state.  See SetCLIDriver
// for the rationale behind the shared state pattern.
func (m *AppModel) SetBridge(b bridgeWidget) {
	m.shared.bridge = b
}

// SetClaudePanel injects the Claude conversation panel.
func (m *AppModel) SetClaudePanel(cp claudePanelWidget) {
	m.shared.claudePanel = cp
}

// SetToasts injects the toast notification model.
func (m *AppModel) SetToasts(t toastWidget) {
	m.shared.toasts = t
}

// SetTeamList injects the team list model.
func (m *AppModel) SetTeamList(tl teamListWidget) {
	m.shared.teamList = tl
}

// SetProviderState injects a pre-configured ProviderState into the shared
// state. Callers that need to override the default four-provider configuration
// (e.g. for testing or custom provider lists) should call this setter before
// the program starts.
func (m *AppModel) SetProviderState(ps *state.ProviderState) {
	m.shared.providerState = ps
}

// ProviderState returns the ProviderState held in shared state.  The
// main.go entry point calls this to obtain the pointer needed to construct
// the ProviderTabBarModel without having to duplicate the creation logic.
// Returns nil when shared state has not been initialised.
func (m *AppModel) ProviderState() *state.ProviderState {
	if m.shared == nil {
		return nil
	}
	return m.shared.providerState
}

// SetProviderTabBar injects the provider tab bar component into the shared
// state. Because the providers package imports state (for state.ProviderID)
// but not model, there is no import cycle; the interface is defined in this
// package and the concrete type is injected from the application entry point.
func (m *AppModel) SetProviderTabBar(ptb providerTabBarWidget) {
	m.shared.providerTabBar = ptb
}

// SetDashboard injects the session dashboard component into the shared state.
func (m *AppModel) SetDashboard(d dashboardWidget) {
	m.shared.dashboard = d
}

// SetSettings injects the settings panel component into the shared state.
func (m *AppModel) SetSettings(s settingsWidget) {
	m.shared.settings = s
}

// SetTelemetry injects the telemetry panel component into the shared state.
func (m *AppModel) SetTelemetry(t telemetryWidget) {
	m.shared.telemetry = t
}

// SetPlanPreview injects the plan preview panel component into the shared state.
func (m *AppModel) SetPlanPreview(pp planPreviewWidget) {
	m.shared.planPreview = pp
}

// SetTaskBoard injects the task board overlay component into the shared state.
func (m *AppModel) SetTaskBoard(tb taskBoardWidget) {
	m.shared.taskBoard = tb
}

// SetBaseCLIOpts stores the CLI driver options supplied at startup so that
// handleProviderSwitch can reconstruct a correctly-configured driver for each
// provider without losing flags (--verbose, --debug, --permission-mode, etc.)
// that were passed on the command line.
func (m *AppModel) SetBaseCLIOpts(opts cli.CLIDriverOpts) {
	m.shared.baseCLIOpts = opts
}

// SetSessionStore injects the session persistence store into the shared state.
// The store manages session metadata and conversation history files.
func (m *AppModel) SetSessionStore(store *session.Store) {
	m.shared.sessionStore = store
}

// SetSessionData injects the initial session data into the shared state.
// On session resume, this is populated from LoadSession; for new sessions,
// the caller creates a fresh SessionData with NewSessionID().
func (m *AppModel) SetSessionData(data *session.SessionData) {
	m.shared.sessionData = data
}

// SessionData returns the current session data held in shared state.
// Returns nil when no session data has been set.
func (m *AppModel) SessionData() *session.SessionData {
	if m.shared == nil {
		return nil
	}
	return m.shared.sessionData
}

// SetShutdownManager stores a shutdown function (typically
// ShutdownManager.Shutdown) in the shared state.  The function is invoked
// when the user triggers graceful shutdown (Ctrl+C) or when the OS delivers
// SIGINT/SIGTERM.  Stored as a func() error to avoid importing the lifecycle
// package from model.
func (m *AppModel) SetShutdownManager(sm interface{ Shutdown() error }) {
	m.shared.shutdownFunc = sm.Shutdown
}

// SaveSessionPublic is the public entry point for session save, used by the
// ShutdownManager's sessionSaver callback.  It delegates to the private
// saveSession method which snapshots cost, provider state, and conversation
// histories to disk.
func (m *AppModel) SaveSessionPublic() {
	m.saveSession()
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init returns the initial command for the program.  Using tea.EnterAltScreen
// preserves the user's scrollback history and ensures a clean exit.
// If a CLI driver has been injected, Init also schedules the Start command so
// the subprocess launches immediately after the Bubbletea runtime begins.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.startCLI(),
		m.statusLine.StartTicks(),
	)
}

// startCLI returns the CLI Start command, or nil when no driver is wired.
// Returning nil from a tea.Cmd is a no-op in Bubbletea.
func (m AppModel) startCLI() tea.Cmd {
	if m.shared == nil || m.shared.cliDriver == nil {
		return nil
	}
	return m.shared.cliDriver.Start()
}

// Update is the sole mutation point for AppModel state.  It handles all
// incoming tea.Msg values and returns the updated model together with any
// commands to run.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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

	case tea.KeyMsg:
		return m.handleKey(msg)

	// -----------------------------------------------------------------
	// CLI lifecycle events (TUI-016)
	// -----------------------------------------------------------------

	case cli.CLIStartedMsg:
		// Subprocess started — begin listening for NDJSON events.
		return m, m.waitForCLIEvent()

	case cli.SystemInitEvent:
		// CLI session is ready; record session metadata.
		m.cliReady = true
		m.sessionID = msg.SessionID
		m.activeModel = msg.Model
		// Persist session ID to active provider for resume support (TUI-031).
		if m.shared != nil && m.shared.providerState != nil && msg.SessionID != "" {
			m.shared.providerState.SetSessionID(msg.SessionID)
		}
		// Sync status line with session metadata.
		m.statusLine.ActiveModel = msg.Model
		m.statusLine.PermissionMode = msg.PermissionMode
		if m.shared != nil && m.shared.providerState != nil {
			m.statusLine.Provider = string(m.shared.providerState.GetActiveProvider())
		}
		if m.statusLine.SessionStart.IsZero() {
			m.statusLine.SessionStart = time.Now()
		}

		// Register the root "Router" agent so the agent tree shows the
		// session immediately (matching Node.js TUI behaviour).
		if m.shared != nil && m.shared.agentRegistry != nil {
			tier := "sonnet"
			modelLower := strings.ToLower(msg.Model)
			if strings.Contains(modelLower, "haiku") {
				tier = "haiku"
			} else if strings.Contains(modelLower, "opus") {
				tier = "opus"
			}
			_ = m.shared.agentRegistry.Register(state.Agent{
				ID:          "router-root",
				AgentType:   "router",
				Description: "Router",
				Model:       msg.Model,
				Tier:        tier,
				Status:      state.StatusRunning,
				StartedAt:   time.Now(),
			})
			m.shared.agentRegistry.InvalidateTreeCache()
			m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
		}

		return m, m.waitForCLIEvent()

	case cli.AssistantEvent:
		var cmds []tea.Cmd

		// Forward text content to Claude panel.
		if m.shared.claudePanel != nil {
			for _, block := range msg.Message.Content {
				if block.Type == "text" && block.Text != "" {
					streaming := msg.Message.StopReason == nil
					cmd := m.shared.claudePanel.HandleMsg(AssistantMsg{
						Text:      block.Text,
						Streaming: streaming,
					})
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}

		// Sync agent registry from Task tool_use blocks.
		if m.shared.agentRegistry != nil {
			result := cli.SyncAssistantEvent(msg, m.shared.agentRegistry)
			if len(result.Registered) > 0 || len(result.Activity) > 0 {
				// C-3: invalidate before reading Tree() so the view reflects
				// the mutations that SyncAssistantEvent just applied.
				m.shared.agentRegistry.InvalidateTreeCache()
				m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
			}
		}

		// Update streaming indicator: if content is present and stop_reason is
		// nil, the assistant is still generating (streaming=true).
		if len(msg.Message.Content) > 0 {
			streaming := msg.Message.StopReason == nil
			if streaming && !m.statusLine.Streaming {
				cmd := m.statusLine.SetStreaming(true)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

		cmds = append(cmds, m.waitForCLIEvent())
		return m, tea.Batch(cmds...)

	case cli.UserEvent:
		// Extract post-hoc diffs.
		m = m.extractDiffs(msg)

		// Sync agent registry from tool_result blocks.
		if m.shared.agentRegistry != nil {
			result := cli.SyncUserEvent(msg, m.shared.agentRegistry)
			if len(result.Updated) > 0 || len(result.Activity) > 0 {
				// C-3: invalidate before reading Tree() so the view reflects
				// the mutations that SyncUserEvent just applied.
				m.shared.agentRegistry.InvalidateTreeCache()
				m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
			}
		}

		// The CLI has echoed back a user message — the assistant is about to
		// respond. Show the thinking indicator if not already streaming.
		if !m.statusLine.Streaming {
			cmd := m.statusLine.SetStreaming(true)
			if cmd != nil {
				return m, tea.Batch(m.waitForCLIEvent(), cmd)
			}
		}

		return m, m.waitForCLIEvent()

	case cli.ResultEvent:
		var cmds []tea.Cmd

		// Update cost tracker (single source of truth).
		if m.shared.costTracker != nil {
			m.shared.costTracker.UpdateSessionCost(msg.TotalCostUSD)
		}
		m.statusLine.SessionCost = msg.TotalCostUSD

		// Accumulate session token counts from aggregate usage.
		m.statusLine.TokenCount += msg.Usage.InputTokens + msg.Usage.OutputTokens

		// Update context window percentage from per-model usage if available.
		if entry, ok := msg.ModelUsage[m.activeModel]; ok && entry.ContextWindow > 0 {
			used := entry.InputTokens + entry.CacheReadInputTokens + entry.CacheCreationInputTokens
			m.statusLine.ContextPercent = float64(used) / float64(entry.ContextWindow) * 100
		}

		// Clear streaming indicator — the turn is complete.
		m.statusLine.Streaming = false

		// Forward to Claude panel to finalize streaming.
		if m.shared.claudePanel != nil {
			cmd := m.shared.claudePanel.HandleMsg(ResultMsg{
				SessionID:  msg.SessionID,
				CostUSD:    msg.TotalCostUSD,
				DurationMS: msg.DurationMS,
			})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Schedule debounced session auto-save (5 s cooldown, TUI-033).
		m.autoSaveSeq++
		seq := m.autoSaveSeq
		cmds = append(cmds, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
			return SessionAutoSaveMsg{Seq: seq}
		}))

		cmds = append(cmds, m.waitForCLIEvent())
		return m, tea.Batch(cmds...)

	case cli.CLIDisconnectedMsg:
		// Subprocess exited or pipe broken — attempt reconnection.
		if msg.Err != nil && m.reconnectCount < maxReconnectAttempts {
			m.reconnectCount++
			return m, reconnectAfterDelay(m.reconnectCount, m.reconnectSeq)
		}
		// Exceeded retries or clean exit — remain disconnected.
		return m, nil

	case CLIReconnectMsg:
		// Discard stale timers created before the last provider switch.
		if msg.Seq != m.reconnectSeq {
			return m, nil
		}
		return m, m.startCLI()

	// -----------------------------------------------------------------
	// Provider switching (TUI-029)
	// -----------------------------------------------------------------

	case ProviderSwitchMsg:
		return m.handleProviderSwitch()

	case ProviderSwitchExecuteMsg:
		// Debounce guard: only the most recent timer executes the switch.
		// Earlier timers (lower Seq values) are silently discarded, which
		// prevents rapid Shift+Tab presses from triggering multiple
		// CLI driver shutdown+restart cycles.
		if msg.Seq != m.providerSwitchSeq {
			return m, nil
		}
		return m.handleProviderSwitch()

	// -----------------------------------------------------------------
	// Bridge events (from MCP server via UDS)
	// -----------------------------------------------------------------

	case BridgeModalRequestMsg:
		// Dispatch to permission handler which enqueues the appropriate modal(s).
		if m.shared != nil && m.shared.permHandler != nil {
			cmd := m.shared.permHandler.HandleBridgeRequest(
				msg.RequestID, msg.Message, msg.Options,
			)
			return m, cmd
		}
		return m, nil

	case modals.ModalResponseMsg:
		// A modal step completed — advance the flow via permission handler.
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

	// -----------------------------------------------------------------
	// Agent and team events (from bridge)
	// -----------------------------------------------------------------

	case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:
		// Refresh tree view. These are internal UI messages, NOT CLI events.
		if m.shared.agentRegistry != nil {
			// C-3: invalidate before reading Tree() so the view reflects any
			// registry mutations that occurred before this message was dispatched.
			m.shared.agentRegistry.InvalidateTreeCache()
			m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
		}
		return m, nil

	case agents.AgentSelectedMsg:
		if m.shared.agentRegistry != nil {
			agent := m.shared.agentRegistry.Get(msg.AgentID)
			if agent != nil {
				m.agentDetail.SetAgent(agent)
			}
		}
		return m, nil

	case ToastMsg:
		// Toast notification — forward to toast component.
		if m.shared != nil && m.shared.toasts != nil {
			cmd := m.shared.toasts.HandleMsg(msg)
			return m, cmd
		}
		return m, nil

	// -----------------------------------------------------------------
	// Shutdown (TUI-034)
	// -----------------------------------------------------------------

	case ShutdownCompleteMsg:
		// Graceful shutdown sequence completed — exit the program.
		if msg.Err != nil {
			log.Printf("[shutdown] completed with error: %v", msg.Err)
		}
		return m, tea.Quit

	// -----------------------------------------------------------------
	// Session persistence (TUI-033)
	// -----------------------------------------------------------------

	case SessionAutoSaveMsg:
		// Debounce guard: only the most recent timer fires the save.
		if msg.Seq != m.autoSaveSeq {
			return m, nil
		}
		m.saveSession()
		return m, nil

	// -----------------------------------------------------------------
	// Remaining CLI event types — re-subscribe without side effects.
	// -----------------------------------------------------------------

	case cli.SystemHookEvent, cli.SystemStatusEvent, cli.RateLimitEvent,
		cli.StreamEvent, cli.CLIUnknownEvent:
		return m, m.waitForCLIEvent()
	}

	// Forward unhandled messages to status line (handles its own tick types).
	{
		updated, cmd := m.statusLine.Update(msg)
		if cmd != nil {
			m.statusLine = updated
			return m, cmd
		}
		m.statusLine = updated
	}

	// Forward unhandled messages to toast for tick-based expiry.
	if m.shared != nil && m.shared.toasts != nil {
		cmd := m.shared.toasts.HandleMsg(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	return m, nil
}

// extractDiffs inspects a UserEvent for tool_use_result blocks that carry a
// structuredPatch field and appends any found patches to m.diffs.
// This implements the post-hoc diff display path for Write/Edit/Bash tools
// (Path 1 of Option D hybrid permission flow).
func (m AppModel) extractDiffs(ev cli.UserEvent) AppModel {
	if len(ev.ToolUseResult) == 0 {
		return m
	}

	// tool_use_result can be a single object or an array of objects.
	// Try single object first.
	var single toolUseResultWithPatch
	if err := json.Unmarshal(ev.ToolUseResult, &single); err == nil && single.FilePath != "" {
		if len(single.StructuredPatch) > 0 {
			m.diffs = append(m.diffs, DiffEntry{
				FilePath: single.FilePath,
				Patch:    single.StructuredPatch,
			})
		}
		return m
	}

	// Try array variant.
	var many []toolUseResultWithPatch
	if err := json.Unmarshal(ev.ToolUseResult, &many); err == nil {
		for _, r := range many {
			if r.FilePath != "" && len(r.StructuredPatch) > 0 {
				m.diffs = append(m.diffs, DiffEntry{
					FilePath: r.FilePath,
					Patch:    r.StructuredPatch,
				})
			}
		}
	}
	return m
}

// toolUseResultWithPatch is a partial unmarshal target for the ToolUseResult
// JSON field on cli.UserEvent.  Only the fields relevant to diff extraction
// are decoded; all other fields are ignored.
type toolUseResultWithPatch struct {
	FilePath        string          `json:"filePath"`
	StructuredPatch json.RawMessage `json:"structuredPatch,omitempty"`
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

// waitForCLIEvent returns the WaitForEvent command from the CLI driver, or
// nil when no driver is wired.  It is called after every handled CLI event to
// maintain the re-subscription chain.
func (m AppModel) waitForCLIEvent() tea.Cmd {
	if m.shared == nil || m.shared.cliDriver == nil {
		return nil
	}
	return m.shared.cliDriver.WaitForEvent()
}

// View renders the current application state to a string.  It is called by
// Bubbletea after every Update and must be fast and free of side effects.
func (m AppModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.renderLayout()
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// handleKey routes a KeyMsg based on modal and focus state.
func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While a modal is open only modal keys are active.
	if m.shared != nil && m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		return m.handleModalKey(msg)
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
