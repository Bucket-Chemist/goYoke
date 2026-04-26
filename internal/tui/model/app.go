// Package model defines shared state types for the goYoke TUI.
// This file contains the root AppModel: the single top-level tea.Model that
// owns all application state and implements The Elm Architecture.
package model

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/cli"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/agents"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/banner"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/drawer"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/modals"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/settingstree"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
	"github.com/Bucket-Chemist/goYoke/internal/tui/session"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// Widget interfaces are defined in interfaces.go.
// Layout constants and rendering are in layout.go.
// Provider switch flow is in provider_switch.go.
// Keyboard handlers are in key_handlers.go.
// CLI event handlers are in cli_event_handlers.go.
// UI event handlers are in ui_event_handlers.go.

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
	// ── CLI / Bridge ──────────────────────────────────────────────────────
	cliDriver cliDriverWidget
	bridge    bridgeWidget
	// baseCLIOpts holds the CLI driver options supplied at startup (C-2).
	// handleProviderSwitch copies this struct and overrides only the fields
	// that change per provider (Model, SessionID, AdapterPath, ProjectDir,
	// EnvVars), so that Verbose, Debug, PermissionMode, MCPConfigPath, and
	// any other flags are transparently preserved across provider switches.
	baseCLIOpts cli.CLIDriverOpts

	// ── State registries ──────────────────────────────────────────────────
	agentRegistry *state.AgentRegistry
	costTracker   *state.CostTracker
	providerState *state.ProviderState

	// ── Permission / modal queue ──────────────────────────────────────────
	modalQueue  *modals.ModalQueue
	permHandler *modals.PermissionHandler

	// ── Primary panel widgets ─────────────────────────────────────────────
	claudePanel    claudePanelWidget
	toasts         toastWidget
	teamList       teamListWidget
	providerTabBar providerTabBarWidget

	// ── Right-panel widgets (TUI-032) ─────────────────────────────────────
	dashboard   dashboardWidget
	settings    settingsWidget
	telemetry   telemetryWidget
	planPreview planPreviewWidget
	teamDetail  TeamDetailWidget

	// ── Drawer stack (TDS-004) ────────────────────────────────────────────
	drawerStack drawerStackWidget

	// ── Team health (TUI-003 / TUI-005) ──────────────────────────────────
	teamsHealth teamsHealthWidget
	// teamNotifiedAt records when the last TeamUpdateMsg arrived.
	// The poll-tick handler suppresses ClearTeamsContent within a 10s
	// grace window to prevent a race where the poll clears the drawer
	// before the team becomes visible to the filesystem scanner.
	teamNotifiedAt time.Time
	taskBoard      taskBoardWidget

	// ── Overlay modals ────────────────────────────────────────────────────
	// planViewModal is the full-screen plan viewer overlay (TUI-056).
	planViewModal    modals.PlanViewModal
	helpModal        modals.HelpModal // full-screen keyboard shortcut reference (alt+h)
	optionsViewModal modals.OptionsViewModal
	modelModal       modals.ModelModal

	// ── Navigation overlays ───────────────────────────────────────────────
	searchOverlay searchOverlayWidget
	hintBar       hintBarWidget
	breadcrumb    breadcrumbWidget
	cwdSelector   cwdSelectorWidget

	// ── Session persistence (TUI-033) ─────────────────────────────────────
	sessionStore *session.Store
	sessionData  *session.SessionData

	// ── Observability (HL-004 / HL-004A) ─────────────────────────────────
	// snapshotStore holds the latest SessionSnapshot and notifies subscribers
	// on every meaningful state transition. Stored as a pointer so it survives
	// Bubbletea's value-copy semantics and can be shared with the control server
	// (HL-006) and downstream relays (HL-013) without coupling to their packages.
	snapshotStore *observability.SnapshotStore
	// lastPublishTime records when the last snapshot was published. Used by
	// publishSnapshotDebounced to rate-limit streaming-token publications.
	lastPublishTime time.Time
	// harnessSessionUpdater mirrors the provider/CLI session ID into harness
	// discovery metadata once SystemInit has assigned it.
	harnessSessionUpdater func(string)

	// ── Lifecycle ─────────────────────────────────────────────────────────
	// shutdownFunc is called to trigger the sequenced shutdown (TUI-034).
	// Stored as a closure so the model package does not import lifecycle.
	shutdownFunc func() error

	// ── Theme / display (TUI-046) ─────────────────────────────────────────
	// activeTheme is always non-nil after NewAppModel(); zero value unusable.
	activeTheme  *config.Theme
	themeVariant config.ThemeVariant
	// reduceMotion disables animations when true (WCAG 2.3.1).
	reduceMotion bool

	// ── Figures state (CM-014) ────────────────────────────────────────────
	// figuresState tracks the diagrams loaded in the figures drawer and
	// which is currently selected for cycling/opening.
	figuresState drawer.FiguresState
}

// ---------------------------------------------------------------------------
// AppModel
// ---------------------------------------------------------------------------

// AppModel is the root tea.Model for the goYoke TUI.  It owns all
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
	activeEffort   string // current --effort value; empty omits the flag
	context1M      bool   // true if initial session resolved to a [1m] model
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

	// mouseEnabled tracks whether tea mouse capture is active; true at startup.
	mouseEnabled bool

	// simpleMode hides the right panel when true, giving the conversation panel
	// 100% of the terminal width (UX-007). Toggled by alt+\. Persisted to
	// SessionData.SimpleMode across restarts.
	simpleMode bool

	// iconRailMode is true when the right panel is narrow enough to activate the
	// compact icon rail rendering (UX-003). Updated with hysteresis in
	// propagateContentSizes: switches to icon rail when rightWidth < 28, reverts
	// to full rendering when rightWidth >= 32. The [28, 32) band is a dead zone
	// that preserves the current mode to prevent flicker during resize.
	iconRailMode bool
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
	defaultTheme := config.DefaultTheme()
	shared := &sharedState{
		modalQueue:       &mq,
		permHandler:      modals.NewPermissionHandler(&mq),
		agentRegistry:    state.NewAgentRegistry(),
		costTracker:      state.NewCostTracker(),
		providerState:    state.NewProviderState(),
		activeTheme:      &defaultTheme,
		themeVariant:     config.ThemeDark,
		planViewModal:    modals.NewPlanViewModal(),
		helpModal:        modals.NewHelpModal(),
		optionsViewModal: modals.NewOptionsViewModal(),
		modelModal:       modals.NewModelModal(),
		snapshotStore:    observability.New(),
	}
	m := AppModel{
		focus:          FocusClaude,
		rightPanelMode: RPMAgents,
		activeTab:      TabChat,
		keys:           keys,
		banner:         banner.NewBannerModel(0),
		statusLine:     statusline.NewStatusLineModel(0),
		agentTree:      agents.NewAgentTreeModel(),
		agentDetail:    agents.NewAgentDetailModel(),
		shared:         shared,
		mouseEnabled:   true,
	}
	m.statusLine.MouseEnabled = true
	return m
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
		m.startTeamPolling(),
		m.discoverFiguresCmd(),
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

// startTeamPolling returns the initial team-list poll command, or nil when
// no team list is wired or StartPolling has not been called yet.  The poll
// fires a package-private pollTickMsg that must be forwarded to the team
// list via the unhandled-message fallthrough in Update().
func (m AppModel) startTeamPolling() tea.Cmd {
	if m.shared == nil || m.shared.teamList == nil {
		return nil
	}
	return m.shared.teamList.PollNow()
}

// Update is the sole mutation point for AppModel state. Each case delegates
// to a named handler; no inline bodies are permitted here.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	// CLI lifecycle (TUI-016)
	case cli.CLIStartedMsg:
		return m.handleCLIStarted()
	case cli.SystemInitEvent:
		return m.handleSystemInit(msg)
	case cli.AssistantEvent:
		return m.handleAssistantEvent(msg)
	case cli.UserEvent:
		return m.handleUserEvent(msg)
	case cli.ResultEvent:
		return m.handleResultEvent(msg)
	case cli.CLIDisconnectedMsg:
		return m.handleCLIDisconnected(msg)
	case CLIReconnectMsg:
		return m.handleCLIReconnect(msg)
	// Provider switching (TUI-029)
	case ProviderSwitchMsg:
		return m.handleProviderSwitchMsg()
	case ProviderSwitchExecuteMsg:
		return m.handleProviderSwitchExecuteMsg(msg)
	// Model / effort switching
	case ModelSwitchRequestMsg:
		return m.handleModelSwitchRequest(msg)
	case modals.ModelSelectedMsg:
		return m.handleModelSelected(msg)
	case modals.ModelModalClosedMsg:
		return m.handleModelModalClosed()
	case EffortChangeRequestMsg:
		return m.handleEffortChangeRequest(msg)
	// Harness slash commands (HL-010)
	case HarnessLinkRequestMsg:
		return m.handleHarnessLink(msg)
	case HarnessUnlinkRequestMsg:
		return m.handleHarnessUnlink(msg)
	case HarnessStatusRequestMsg:
		return m.handleHarnessStatus()
	case HarnessResultMsg:
		return m.handleHarnessResult(msg)
	// Bridge events (MCP server via UDS)
	case BridgeModalRequestMsg:
		return m.handleBridgeModalRequest(msg)
	case CLIPermissionRequestMsg:
		return m.handleCLIPermissionRequest(msg)
	case drawer.ModalResponseMsg:
		return m.handleDrawerModalResponse(msg)
	case drawer.PlanViewRequestMsg:
		return m.handlePlanViewRequest()
	case drawer.OptionsViewRequestMsg:
		return m.handleOptionsViewRequest(msg)
	case modals.OptionsViewClosedMsg:
		return m, nil
	case modals.ModalResponseMsg:
		return m.handleModalResponse(msg)
	// CWD selector
	case OpenCWDSelectorMsg:
		return m.handleOpenCWDSelector()
	case CWDChangedMsg:
		return m.handleCWDChanged(msg)
	// Remote action messages (HL-005) — injected via program.Send() by control server
	case RemoteSubmitPromptMsg:
		return m.handleRemoteSubmitPrompt(msg)
	case RemoteInterruptMsg:
		return m.handleRemoteInterrupt(msg)
	case RemoteRespondModalMsg:
		return m.handleRemoteRespondModal(msg)
	case RemoteRespondPermissionMsg:
		return m.handleRemoteRespondPermission(msg)
	case RemoteSetModelMsg:
		return m.handleRemoteSetModel(msg)
	case RemoteSetEffortMsg:
		return m.handleRemoteSetEffort(msg)
	case RemoteSetCWDMsg:
		return m.handleRemoteSetCWD(msg)
	// Agent and team events
	case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:
		return m.handleAgentRegistryMsg(msg)
	case AgentTodoUpdateMsg:
		return m.handleAgentTodoUpdate(msg)
	case agents.AgentSelectedMsg:
		return m.handleAgentSelected(msg)
	case agents.AgentDetailFocusMsg:
		return m.handleAgentDetailFocus(msg)
	case agents.AgentTreeFocusMsg:
		return m.handleAgentTreeFocus()
	case agents.TreePulseTickMsg:
		return m.handleTreePulseTick(msg)
	case ToastMsg:
		return m.handleToastMsg(msg)
	// Shutdown (TUI-034)
	case ShutdownRequestMsg:
		return m.handleShutdownRequest()
	case ShutdownCompleteMsg:
		return m.handleShutdownComplete(msg)
	// Session / theme / settings / plan
	case SessionAutoSaveMsg:
		return m.handleSessionAutoSave(msg)
	case ThemeChangedMsg:
		return m.handleThemeChanged(msg)
	case settingstree.SettingChangedMsg:
		return m.handleSettingChanged(msg)
	case PlanStepMsg:
		return m.handlePlanStep(msg)
	// Drawer messages (TDS-005)
	case DrawerContentMsg:
		return m.handleDrawerContent(msg)
	case DrawerMinimizeMsg:
		return m.handleDrawerMinimize(msg)
	case drawer.FiguresContentMsg:
		return m.handleFiguresContent(msg)
	// Team update and tab flash
	case TeamUpdateMsg:
		return m.handleTeamUpdate(msg)
	case TabFlashMsg:
		return m.handleTabFlash(msg)
	// Modals that self-deactivate on Esc/q
	case modals.PlanViewClosedMsg, modals.HelpClosedMsg:
		return m, nil
	// Remaining CLI events — re-subscribe without side effects
	case cli.SystemHookEvent, cli.SystemStatusEvent, cli.RateLimitEvent,
		cli.StreamEvent, cli.CLIUnknownEvent:
		return m, m.waitForCLIEvent()
	}
	return m.handleUnrouted(msg)
}

// View renders the current application state to a string.  It is called by
// Bubbletea after every Update and must be fast and free of side effects.
func (m AppModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.renderLayout()
}
