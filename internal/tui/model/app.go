// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the root AppModel: the single top-level tea.Model that
// owns all application state and implements The Elm Architecture.
package model

import (
	"encoding/json"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/banner"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/taskboard"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// tabBarWidget
//
// tabBarWidget is a minimal interface that decouples AppModel from the
// concrete tabbar.TabBarModel type.  The tabbar package imports the model
// package (for model.TabID), so a direct import of tabbar here would create
// a circular dependency.  The interface breaks the cycle while still allowing
// AppModel to call View() and SetWidth() on the component.
// ---------------------------------------------------------------------------

// tabBarWidget is the interface satisfied by tabbar.TabBarModel.
type tabBarWidget interface {
	View() string
	SetWidth(int)
}

// ---------------------------------------------------------------------------
// cliDriverWidget
//
// cliDriverWidget is the interface satisfied by cli.CLIDriver. Defining it
// here in the model package avoids a circular import: the cli package imports
// bubbletea but not model; the model package imports bubbletea but must not
// import cli. The interface breaks the cycle while still allowing AppModel
// to drive the CLI subprocess lifecycle.
// ---------------------------------------------------------------------------

// cliDriverWidget is the interface satisfied by cli.CLIDriver.
type cliDriverWidget interface {
	Start() tea.Cmd
	WaitForEvent() tea.Cmd
	SendMessage(text string) tea.Cmd
	Shutdown() error
}

// ---------------------------------------------------------------------------
// bridgeWidget
//
// bridgeWidget is the interface satisfied by bridge.IPCBridge. Defining it
// here avoids a circular import between the model and bridge packages.
// ResolveModalSimple accepts a plain string value rather than
// mcp.ModalResponsePayload so that the model package does not need to import
// the mcp package.  IPCBridge.ResolveModalSimple wraps ResolveModal.
// ---------------------------------------------------------------------------

// bridgeWidget is the interface satisfied by bridge.IPCBridge.
type bridgeWidget interface {
	Start()
	SocketPath() string
	Shutdown()
	// ResolveModalSimple delivers the user's response to the bridge goroutine
	// that is blocking on the given requestID.  value is the selected option
	// label or free-text entered by the user.  An empty value with cancelled
	// semantics should be represented by calling ResolveModalSimple with an
	// empty string (the bridge always receives a value; cancellation is
	// communicated by convention with the empty string or a dedicated sentinel).
	ResolveModalSimple(requestID string, value string)
}

// ---------------------------------------------------------------------------
// claudePanelWidget
//
// claudePanelWidget is the interface satisfied by *claude.ClaudePanelModel.
// claude imports model, so a direct import here would create a cycle.
// ---------------------------------------------------------------------------

// claudePanelWidget is the interface satisfied by *claude.ClaudePanelModel.
type claudePanelWidget interface {
	HandleMsg(msg tea.Msg) tea.Cmd
	View() string
	SetSize(width, height int)
	SetFocused(focused bool)
	IsStreaming() bool
	// SaveMessages returns a snapshot of the current conversation history.
	// ToolBlocks are omitted; only the role, content, and timestamp are kept.
	SaveMessages() []state.DisplayMessage
	// RestoreMessages replaces the conversation history with the given
	// messages, resets streaming state, and redraws the viewport.
	RestoreMessages([]state.DisplayMessage)
}

// ---------------------------------------------------------------------------
// toastWidget
//
// toastWidget is the interface satisfied by *toast.ToastModel.
// ---------------------------------------------------------------------------

// toastWidget is the interface satisfied by *toast.ToastModel.
type toastWidget interface {
	HandleMsg(msg tea.Msg) tea.Cmd
	View() string
	SetSize(width, height int)
	IsEmpty() bool
}

// ---------------------------------------------------------------------------
// teamListWidget
//
// teamListWidget is the interface satisfied by *teams.TeamListModel.
// ---------------------------------------------------------------------------

// teamListWidget is the interface satisfied by *teams.TeamListModel.
type teamListWidget interface {
	HandleMsg(msg tea.Msg) tea.Cmd
	View() string
	SetSize(width, height int)
	StartPolling(teamsDir string) tea.Cmd
}

// ---------------------------------------------------------------------------
// providerTabBarWidget
//
// providerTabBarWidget is the interface satisfied by
// providers.ProviderTabBarModel. The providers package imports state (for
// state.ProviderID) but not model, so there is no circular import.  The
// interface is defined here in model to keep the widget coupling pattern
// consistent with tabBarWidget, cliDriverWidget, etc.
// ---------------------------------------------------------------------------

// providerTabBarWidget is the interface satisfied by providers.ProviderTabBarModel.
type providerTabBarWidget interface {
	View() string
	SetActive(state.ProviderID)
	SetWidth(int)
	IsVisible() bool
	Height() int
}

// ---------------------------------------------------------------------------
// dashboardWidget
//
// dashboardWidget is the interface satisfied by
// *dashboard.DashboardModel. The dashboard package has no dependency on
// model, so there is no import cycle; the interface is defined here to keep
// the widget coupling pattern consistent.
// ---------------------------------------------------------------------------

// dashboardWidget is the interface satisfied by *dashboard.DashboardModel.
type dashboardWidget interface {
	View() string
	SetSize(w, h int)
	SetData(cost float64, tokens int64, msgs, agents, teams int, start time.Time)
}

// ---------------------------------------------------------------------------
// settingsWidget
//
// settingsWidget is the interface satisfied by *settings.SettingsModel.
// ---------------------------------------------------------------------------

// settingsWidget is the interface satisfied by *settings.SettingsModel.
type settingsWidget interface {
	View() string
	SetSize(w, h int)
	SetConfig(model, provider, permMode, sessionDir string, mcpServers []string)
}

// ---------------------------------------------------------------------------
// telemetryWidget
//
// telemetryWidget is the interface satisfied by *telemetry.TelemetryModel.
// ---------------------------------------------------------------------------

// telemetryWidget is the interface satisfied by *telemetry.TelemetryModel.
type telemetryWidget interface {
	HandleMsg(msg tea.Msg) tea.Cmd
	View() string
	SetSize(w, h int)
}

// ---------------------------------------------------------------------------
// planPreviewWidget
//
// planPreviewWidget is the interface satisfied by *planpreview.PlanPreviewModel.
// ---------------------------------------------------------------------------

// planPreviewWidget is the interface satisfied by *planpreview.PlanPreviewModel.
type planPreviewWidget interface {
	View() string
	SetSize(w, h int)
	SetContent(markdown string)
	ClearContent()
}

// ---------------------------------------------------------------------------
// taskBoardWidget
//
// taskBoardWidget is the interface satisfied by *taskboard.TaskBoardModel.
// ---------------------------------------------------------------------------

// taskBoardWidget is the interface satisfied by *taskboard.TaskBoardModel.
type taskBoardWidget interface {
	View() string
	SetSize(w, h int)
	Toggle()
	IsVisible() bool
	Height() int
	SetTasks(tasks []taskboard.TaskEntry)
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
	// Right-panel widgets (TUI-032).
	dashboard   dashboardWidget
	settings    settingsWidget
	telemetry   telemetryWidget
	planPreview planPreviewWidget
	taskBoard   taskBoardWidget
}

// ---------------------------------------------------------------------------
// Layout constants
//
// These define the fixed-height allocations for chrome rows.
// ---------------------------------------------------------------------------

const (
	bannerHeight     = 3 // rounded border top + title + border bottom
	tabBarHeight     = 1 // single-row strip
	statusLineHeight = 2 // two-row status bar
	borderFrame      = 2 // border chars on each axis (1 left + 1 right)
)

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
// layoutDims
// ---------------------------------------------------------------------------

// layoutDims holds the pre-computed panel dimensions for the current terminal
// size.  It is recomputed on every WindowSizeMsg and passed to the rendering
// helpers so the View method stays free of arithmetic.
type layoutDims struct {
	// leftWidth and rightWidth are the inner content widths (without borders).
	leftWidth  int
	rightWidth int

	// contentHeight is the number of rows available for both panels after
	// subtracting banner, tab bar, and status line heights.
	contentHeight int

	// showRightPanel is false when the terminal is too narrow to display both
	// panels side-by-side.
	showRightPanel bool
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
				m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
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
				m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
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

		cmds = append(cmds, m.waitForCLIEvent())
		return m, tea.Batch(cmds...)

	case cli.CLIDisconnectedMsg:
		// Subprocess exited or pipe broken — attempt reconnection.
		if msg.Err != nil && m.reconnectCount < maxReconnectAttempts {
			m.reconnectCount++
			return m, reconnectAfterDelay(m.reconnectCount)
		}
		// Exceeded retries or clean exit — remain disconnected.
		return m, nil

	case CLIReconnectMsg:
		// Reconnection timer fired — restart the CLI subprocess.
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
	// Remaining CLI event types — re-subscribe without side effects.
	// -----------------------------------------------------------------

	case cli.SystemHookEvent, cli.SystemStatusEvent, cli.RateLimitEvent,
		cli.StreamEvent, cli.CLIUnknownEvent:
		return m, m.waitForCLIEvent()
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

// handleProviderSwitch implements the provider-cycling flow (TUI-029):
//
//  1. Save the current conversation history to the active provider slot.
//  2. Save the current session ID to the active provider slot.
//  3. Cycle to the next provider in the canonical order.
//  4. Restore the new provider's conversation history into the panel.
//  5. Shutdown the old CLI driver.
//  6. Create a new CLI driver configured for the new provider.
//  7. Start the new CLI driver.
//
// The method is a no-op when no ProviderState is wired (e.g. in tests that
// do not inject one).
func (m AppModel) handleProviderSwitch() (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.providerState == nil {
		return m, nil
	}

	ps := m.shared.providerState

	// 1. Persist current conversation to the active provider slot.
	// Capture the old provider and its messages BEFORE cycling for handoff generation.
	oldProvider := ps.GetActiveProvider()
	var oldMsgs []state.DisplayMessage
	if m.shared.claudePanel != nil {
		oldMsgs = m.shared.claudePanel.SaveMessages()
		ps.SetActiveMessages(oldMsgs)
	}

	// 2. Persist current session ID to the active provider slot.
	if m.sessionID != "" {
		ps.SetSessionID(m.sessionID)
	}

	// 3. Cycle to the next provider in the canonical ordered list.
	providers := ps.AllProviders()
	nextIdx := 0
	for i, p := range providers {
		if p == oldProvider {
			nextIdx = (i + 1) % len(providers)
			break
		}
	}
	if err := ps.SwitchProvider(providers[nextIdx]); err != nil {
		// Unknown provider — this should never happen with the hardcoded list.
		return m, nil
	}

	// Update provider tab bar highlight to reflect the new active provider.
	if m.shared.providerTabBar != nil {
		m.shared.providerTabBar.SetActive(ps.GetActiveProvider())
	}

	// 4. Restore the new provider's conversation history.
	if m.shared.claudePanel != nil {
		newMsgs := ps.GetActiveMessages()
		m.shared.claudePanel.RestoreMessages(newMsgs)
	}

	// 4.5. Inject handoff context so the new provider knows what was being discussed.
	handoff := buildHandoffSummary(oldMsgs, oldProvider, ps.GetActiveProvider())
	if handoff != "" {
		ps.AppendMessage(state.DisplayMessage{
			Role:      "system",
			Content:   handoff,
			Timestamp: time.Now(),
		})
		// Re-restore so the injected handoff message is visible in the panel.
		if m.shared.claudePanel != nil {
			m.shared.claudePanel.RestoreMessages(ps.GetActiveMessages())
		}
	}

	// 5. Shutdown the old CLI driver.
	if m.shared.cliDriver != nil {
		_ = m.shared.cliDriver.Shutdown()
	}

	// 6. Build CLI driver options for the new provider.
	cfg := ps.GetActiveConfig()
	model := ps.GetActiveModel()

	opts := cli.CLIDriverOpts{
		Model:          model,
		SessionID:      ps.GetActiveSessionID(), // Resume if provider was used before (TUI-031)
		PermissionMode: "acceptEdits",
		AdapterPath:    cfg.AdapterPath,
		ProjectDir:     ps.GetActiveProjectDir(),
	}
	// Materialise env-var keys for the new provider.  The values are
	// intentionally left empty here: the actual credentials must be present
	// in the process environment already (set by the user before launch).
	// We only pass the map so the driver knows which vars are relevant.
	if len(cfg.EnvVars) > 0 {
		envCopy := make(map[string]string, len(cfg.EnvVars))
		for k := range cfg.EnvVars {
			envCopy[k] = "" // empty — real value comes from os.Environ()
		}
		opts.EnvVars = envCopy
	}

	// Reset per-session state so the new provider starts fresh.
	m.cliReady = false
	m.sessionID = ""
	m.activeModel = model
	m.reconnectCount = 0

	// 7. Create, wire, and start the new CLI driver.
	newDriver := cli.NewCLIDriver(opts)
	m.shared.cliDriver = newDriver

	return m, newDriver.Start()
}

// syncFocusState propagates the current focus state to child components.
func (m *AppModel) syncFocusState() {
	if m.shared != nil && m.shared.claudePanel != nil {
		m.shared.claudePanel.SetFocused(m.focus == FocusClaude)
	}
	m.agentTree.SetFocused(m.focus == FocusAgents)
}

// ---------------------------------------------------------------------------
// Layout helpers
// ---------------------------------------------------------------------------

// computeLayout calculates panel dimensions from the current terminal size.
//
// Responsive breakpoints:
//   - width < 80  → single-column (right panel hidden)
//   - width < 100 → left 75%, right 25%
//   - width >= 100 → left 70%, right 30%
//
// Border frame (1 char per edge = 2 per axis) is subtracted from each panel
// inner width so that the borders do not overflow the terminal width.
func (m AppModel) computeLayout() layoutDims {
	dims := layoutDims{}

	// Content rows available after chrome.
	providerTabH := 0
	if m.shared != nil && m.shared.providerTabBar != nil {
		providerTabH = m.shared.providerTabBar.Height()
	}
	taskBoardH := 0
	if m.shared != nil && m.shared.taskBoard != nil {
		taskBoardH = m.shared.taskBoard.Height()
	}
	dims.contentHeight = m.height - bannerHeight - tabBarHeight - providerTabH - statusLineHeight - taskBoardH
	if dims.contentHeight < 1 {
		dims.contentHeight = 1
	}

	if m.width < 80 {
		// Narrow: single column, right panel hidden.
		dims.showRightPanel = false
		dims.leftWidth = m.width - borderFrame
		if dims.leftWidth < 1 {
			dims.leftWidth = 1
		}
		return dims
	}

	dims.showRightPanel = true

	var leftRatio float64
	if m.width < 100 {
		leftRatio = 0.75
	} else {
		leftRatio = 0.70
	}

	// Compute outer column widths, then subtract border frame for inner.
	leftOuter := int(float64(m.width) * leftRatio)
	rightOuter := m.width - leftOuter

	dims.leftWidth = leftOuter - borderFrame
	dims.rightWidth = rightOuter - borderFrame

	if dims.leftWidth < 1 {
		dims.leftWidth = 1
	}
	if dims.rightWidth < 1 {
		dims.rightWidth = 1
	}

	return dims
}

// renderLayout composes the full Lipgloss layout.
//
// Structure (top to bottom):
//
//	Banner     (3 rows, full width)
//	TabBar     (1 row, full width)
//	Main area  (left + optional right panel)
//	StatusLine (2 rows, full width)
//
// When a modal is active the layout is rendered as normal and then replaced by
// the modal overlay via lipgloss.Place so the modal appears centered on screen.
func (m AppModel) renderLayout() string {
	// Modal overlay takes full precedence: render and return immediately.
	if m.shared != nil && m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		m.shared.modalQueue.SetTermSize(m.width, m.height)
		return m.shared.modalQueue.View()
	}

	dims := m.computeLayout()

	bannerView := m.banner.View()

	var tabBarView string
	if m.tabBar != nil {
		tabBarView = m.tabBar.View()
	}

	statusLineView := m.statusLine.View()

	mainArea := m.renderMain(dims)

	parts := []string{bannerView, tabBarView}

	// Insert provider tab bar between the tab bar and main content area.
	if m.shared != nil && m.shared.providerTabBar != nil && m.shared.providerTabBar.IsVisible() {
		parts = append(parts, m.shared.providerTabBar.View())
	}

	parts = append(parts, mainArea)

	// Task board overlay renders between main area and toast/status line.
	if m.shared != nil && m.shared.taskBoard != nil && m.shared.taskBoard.IsVisible() {
		parts = append(parts, m.shared.taskBoard.View())
	}

	// Toast notifications render between main area and status line.
	if m.shared != nil && m.shared.toasts != nil && !m.shared.toasts.IsEmpty() {
		parts = append(parts, m.shared.toasts.View())
	}

	parts = append(parts, statusLineView)
	return lipgloss.JoinVertical(lipgloss.Top, parts...)
}

// renderMain renders the split content area (left panel + optional right panel).
func (m AppModel) renderMain(dims layoutDims) string {
	leftPanel := m.renderLeftPanel(dims)

	if !dims.showRightPanel {
		return leftPanel
	}

	rightPanel := m.renderRightPanel(dims)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// renderLeftPanel renders the Claude conversation panel with the appropriate
// focus border.
func (m AppModel) renderLeftPanel(dims layoutDims) string {
	focused := m.focus == FocusClaude

	var content string
	if m.shared != nil && m.shared.claudePanel != nil {
		content = m.shared.claudePanel.View()
	} else {
		content = config.StyleSubtle.Render("Claude panel  [focus=" + m.focus.String() + "]")
	}

	var style lipgloss.Style
	if focused {
		style = config.StyleFocusedBorder
	} else {
		style = config.StyleUnfocusedBorder
	}

	return style.
		Width(dims.leftWidth).
		Height(dims.contentHeight).
		Render(content)
}

// renderRightPanel renders the right-side panel whose content depends on the
// active RightPanelMode.
func (m AppModel) renderRightPanel(dims layoutDims) string {
	focused := m.focus == FocusAgents

	var content string
	switch m.rightPanelMode {
	case RPMAgents:
		treeView := m.agentTree.View()
		detailView := m.agentDetail.View()
		content = lipgloss.JoinVertical(lipgloss.Left, treeView, detailView)
	case RPMDashboard:
		if m.shared != nil && m.shared.dashboard != nil {
			content = m.shared.dashboard.View()
		} else {
			content = config.StyleSubtle.Render("Dashboard")
		}
	case RPMSettings:
		if m.shared != nil && m.shared.settings != nil {
			content = m.shared.settings.View()
		} else {
			content = config.StyleSubtle.Render("Settings")
		}
	case RPMTelemetry:
		if m.shared != nil && m.shared.telemetry != nil {
			content = m.shared.telemetry.View()
		} else {
			content = config.StyleSubtle.Render("Telemetry")
		}
	case RPMPlanPreview:
		if m.shared != nil && m.shared.planPreview != nil {
			content = m.shared.planPreview.View()
		} else {
			content = config.StyleSubtle.Render("Plan Preview")
		}
	default:
		content = config.StyleSubtle.Render(m.rightPanelMode.String())
	}

	var style lipgloss.Style
	if focused {
		style = config.StyleFocusedBorder
	} else {
		style = config.StyleUnfocusedBorder
	}

	return style.
		Width(dims.rightWidth).
		Height(dims.contentHeight).
		Render(content)
}
