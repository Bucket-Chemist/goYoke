// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the root AppModel: the single top-level tea.Model that
// owns all application state and implements The Elm Architecture.
package model

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/banner"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/settingstree"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/session"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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

	// planViewModal is the full-screen plan viewer overlay (TUI-056).
	// It is activated by alt+v when rightPanelMode == RPMPlanPreview.
	planViewModal modals.PlanViewModal

	// searchOverlay is the unified cross-panel fuzzy search overlay (TUI-059).
	// It is activated by ctrl+f and queries all registered SearchSources.
	searchOverlay searchOverlayWidget

	// hintBar is the context-aware keyboard hint bar (TUI-060).
	// It renders a single muted row of key:description pairs between the
	// task board / toasts and the status line.
	hintBar hintBarWidget

	// breadcrumb is the navigation breadcrumb trail (TUI-063).
	// It renders a single row between the tab bar / provider bar and the
	// main content area showing the current navigation context.
	breadcrumb breadcrumbWidget

	// Session persistence (TUI-033).
	sessionStore *session.Store
	sessionData  *session.SessionData

	// Graceful shutdown (TUI-034).
	// shutdownFunc is called to trigger the sequenced shutdown.  It is a
	// func() error (ShutdownManager.Shutdown) stored as a closure so the
	// model package does not import the lifecycle package.
	shutdownFunc func() error

	// Theme (TUI-046).
	// activeTheme is the current color theme.  It is always non-nil after
	// NewAppModel() runs; the zero value of Theme is unusable (styles are
	// empty lipgloss.Style values), so we store a pointer and initialise it
	// in NewAppModel with DefaultTheme().
	activeTheme *config.Theme
	// themeVariant tracks which variant produced activeTheme so it can be
	// persisted to SessionData without re-deriving the variant from the Theme
	// value (which would require a reverse lookup).
	themeVariant config.ThemeVariant
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
	defaultTheme := config.DefaultTheme()
	shared := &sharedState{
		modalQueue:    &mq,
		permHandler:   modals.NewPermissionHandler(&mq),
		agentRegistry: state.NewAgentRegistry(),
		costTracker:   state.NewCostTracker(),
		providerState: state.NewProviderState(),
		activeTheme:   &defaultTheme,
		themeVariant:  config.ThemeDark,
		planViewModal: modals.NewPlanViewModal(),
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

// Update is the sole mutation point for AppModel state.  It dispatches all
// incoming tea.Msg values to focused handler methods and returns the updated
// model together with any commands to run.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	// -----------------------------------------------------------------
	// CLI lifecycle events (TUI-016)
	// -----------------------------------------------------------------

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

	// -----------------------------------------------------------------
	// Provider switching (TUI-029)
	// -----------------------------------------------------------------

	case ProviderSwitchMsg:
		return m.handleProviderSwitchMsg()

	case ProviderSwitchExecuteMsg:
		return m.handleProviderSwitchExecuteMsg(msg)

	// -----------------------------------------------------------------
	// Bridge events (from MCP server via UDS)
	// -----------------------------------------------------------------

	case BridgeModalRequestMsg:
		return m.handleBridgeModalRequest(msg)

	case modals.ModalResponseMsg:
		return m.handleModalResponse(msg)

	// -----------------------------------------------------------------
	// Agent and team events (from bridge)
	// -----------------------------------------------------------------

	case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:
		return m.handleAgentRegistryMsg()

	case agents.AgentSelectedMsg:
		return m.handleAgentSelected(msg)

	case ToastMsg:
		return m.handleToastMsg(msg)

	// -----------------------------------------------------------------
	// Shutdown (TUI-034)
	// -----------------------------------------------------------------

	case ShutdownCompleteMsg:
		return m.handleShutdownComplete(msg)

	// -----------------------------------------------------------------
	// Session persistence (TUI-033)
	// -----------------------------------------------------------------

	case SessionAutoSaveMsg:
		return m.handleSessionAutoSave(msg)

	// -----------------------------------------------------------------
	// Theme switching (TUI-046)
	// -----------------------------------------------------------------

	case ThemeChangedMsg:
		return m.handleThemeChanged(msg)

	// -----------------------------------------------------------------
	// Settings panel changes (TUI-051)
	// -----------------------------------------------------------------

	case settingstree.SettingChangedMsg:
		return m.handleSettingChanged(msg)

	// -----------------------------------------------------------------
	// Plan mode tracking (TUI-057)
	// -----------------------------------------------------------------

	case PlanStepMsg:
		return m.handlePlanStep(msg)

	// -----------------------------------------------------------------
	// Tab flash animation (TUI-061)
	// -----------------------------------------------------------------

	case TabFlashMsg:
		return m.handleTabFlash(msg)

	// -----------------------------------------------------------------
	// Plan view modal (TUI-056)
	// -----------------------------------------------------------------

	case modals.PlanViewClosedMsg:
		// Nothing extra to do; the modal deactivated itself on Esc/q.
		// The renderLayout guard already checks IsActive before rendering.
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

	// Forward unhandled messages to the tab bar for flash tick processing
	// (TUI-061).  The tabFlashTickMsg type is unexported from the tabbar
	// package, so AppModel cannot type-switch on it; forwarding all
	// unhandled messages here ensures the tick clears the flash state.
	if m.tabBar != nil {
		cmd := m.tabBar.HandleMsg(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	return m, nil
}

// View renders the current application state to a string.  It is called by
// Bubbletea after every Update and must be fast and free of side effects.
func (m AppModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.renderLayout()
}
