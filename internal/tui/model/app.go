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
	teamDetail  TeamDetailWidget
	// drawerStack is the collapsible drawer stack (TDS-004).
	// It manages the options and plan drawers in the right panel.
	drawerStack drawerStackWidget
	// teamsHealth is the team health dashboard widget (TUI-003).
	// Concrete implementation lives in the teams package (TUI-005).
	teamsHealth teamsHealthWidget
	// teamNotifiedAt records when the last TeamUpdateMsg arrived.
	// The poll-tick handler suppresses ClearTeamsContent within a 10s
	// grace window to prevent a race where the poll clears the drawer
	// before the team becomes visible to the filesystem scanner.
	teamNotifiedAt time.Time
	taskBoard      taskBoardWidget

	// planViewModal is the full-screen plan viewer overlay (TUI-056).
	// It is activated by alt+v when rightPanelMode == RPMPlanPreview.
	planViewModal modals.PlanViewModal
	helpModal     modals.HelpModal // full-screen keyboard shortcut reference (alt+h)

	// optionsViewModal is the full-screen options viewer overlay (alt+o).
	optionsViewModal modals.OptionsViewModal

	// modelModal is the interactive model selector overlay (/model).
	modelModal modals.ModelModal

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

	// cwdSelector is the modal overlay for changing the working directory.
	// It propagates CWD changes to os.Chdir and GOYOKE_CWD env var so that
	// spawned Claude CLI subprocesses inherit the desired scope.
	cwdSelector cwdSelectorWidget

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

	// reduceMotion disables animations when true (WCAG 2.3.1).
	// Set via Settings → Display → Reduce Motion toggle.
	reduceMotion bool
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
		modalQueue:    &mq,
		permHandler:   modals.NewPermissionHandler(&mq),
		agentRegistry: state.NewAgentRegistry(),
		costTracker:   state.NewCostTracker(),
		providerState: state.NewProviderState(),
		activeTheme:   &defaultTheme,
		themeVariant:  config.ThemeDark,
		planViewModal:    modals.NewPlanViewModal(),
		helpModal:        modals.NewHelpModal(),
		optionsViewModal: modals.NewOptionsViewModal(),
		modelModal:       modals.NewModelModal(),
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
	// Model switching
	// -----------------------------------------------------------------

	case ModelSwitchRequestMsg:
		return m.handleModelSwitchRequest(msg)

	case modals.ModelSelectedMsg:
		m.shared.modelModal.Hide()
		return m.handleModelSwitchRequest(ModelSwitchRequestMsg{ModelID: msg.ModelID})

	case modals.ModelModalClosedMsg:
		m.shared.modelModal.Hide()
		m.updateHintContext()
		return m, nil

	// -----------------------------------------------------------------
	// Effort switching
	// -----------------------------------------------------------------

	case EffortChangeRequestMsg:
		return m.handleEffortChangeRequest(msg)

	// -----------------------------------------------------------------
	// Bridge events (from MCP server via UDS)
	// -----------------------------------------------------------------

	case BridgeModalRequestMsg:
		return m.handleBridgeModalRequest(msg)

	case CLIPermissionRequestMsg:
		return m.handleCLIPermissionRequest(msg)

	case drawer.ModalResponseMsg:
		// TDS-006: Drawer resolved a modal — deliver response to the bridge.
		if m.shared != nil && m.shared.bridge != nil {
			value := msg.Value
			if msg.Cancelled {
				value = ""
			}
			m.shared.bridge.ResolveModalSimple(msg.RequestID, value)
		}
		// Clear stale drawer modal state when the response came from the
		// full-screen OptionsViewModal (which doesn't call ClearActiveModal).
		if m.shared != nil && m.shared.drawerStack != nil &&
			m.shared.drawerStack.OptionsActiveRequestID() == msg.RequestID {
			m.shared.drawerStack.ClearOptionsModal()
		}
		// Snap focus back to Claude if the options drawer just minimized.
		if m.focus == FocusOptionsDrawer && m.shared != nil && m.shared.drawerStack != nil {
			if !m.shared.drawerStack.OptionsHasContent() {
				m.focus = FocusClaude
				m.syncFocusState()
				m.updateHintContext()
				m.updateBreadcrumbs()
				m.propagateContentSizes()
			}
		}
		return m, nil

	case drawer.PlanViewRequestMsg:
		// TDS-007: Open full-screen plan view modal (same as alt+v from RPMPlanPreview).
		if m.shared != nil && m.shared.planPreview != nil {
			content := m.shared.planPreview.Content()
			if content != "" {
				m.shared.planViewModal.SetContent(content, m.width)
				m.shared.planViewModal.SetSize(m.width, m.height)
				m.shared.planViewModal.Show()
			}
		}
		return m, nil

	case drawer.OptionsViewRequestMsg:
		return m.handleOptionsViewRequest(msg)

	case modals.OptionsViewClosedMsg:
		return m, nil

	case modals.ModalResponseMsg:
		return m.handleModalResponse(msg)

	// -----------------------------------------------------------------
	// CWD selector
	// -----------------------------------------------------------------

	case OpenCWDSelectorMsg:
		if m.shared != nil && m.shared.cwdSelector != nil {
			m.shared.cwdSelector.SetSize(m.width, m.height)
			m.shared.cwdSelector.Show()
		}
		return m, nil

	case CWDChangedMsg:
		return m.handleCWDChanged(msg)

	// -----------------------------------------------------------------
	// Agent and team events (from bridge)
	// -----------------------------------------------------------------

	case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:
		return m.handleAgentRegistryMsg(msg)

	case AgentTodoUpdateMsg:
		return m.handleAgentTodoUpdate(msg)

	case agents.AgentSelectedMsg:
		return m.handleAgentSelected(msg)

	case agents.AgentDetailFocusMsg:
		// Transfer focus from tree to detail panel.
		m.agentTree.SetFocused(false)
		m.agentDetail.SetFocused(true)
		// Also select the agent.
		if m.shared.agentRegistry != nil {
			if a := m.shared.agentRegistry.Get(msg.AgentID); a != nil {
				m.agentDetail.SetAgent(a)
			}
		}
		return m, nil

	case agents.AgentTreeFocusMsg:
		// Transfer focus back from detail to tree.
		m.agentDetail.SetFocused(false)
		m.agentTree.SetFocused(true)
		return m, nil

	case agents.TreePulseTickMsg:
		// Forward pulse tick to the tree; the tree self-reschedules lazily.
		result, cmd := m.agentTree.Update(msg)
		if updated, ok := result.(agents.AgentTreeModel); ok {
			m.agentTree = updated
		}
		return m, cmd

	case ToastMsg:
		return m.handleToastMsg(msg)

	// -----------------------------------------------------------------
	// Shutdown (TUI-034)
	// -----------------------------------------------------------------

	case ShutdownRequestMsg:
		// /exit or /quit from the ClaudePanel — same path as first Ctrl+C.
		if m.shutdownInProgress {
			return m, nil
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
	// Drawer messages (TDS-005)
	// -----------------------------------------------------------------

	case DrawerContentMsg:
		if m.shared != nil && m.shared.drawerStack != nil {
			switch msg.DrawerID {
			case "options":
				m.shared.drawerStack.SetOptionsContent(msg.Content)
			case "plan":
				m.shared.drawerStack.SetPlanContent(msg.Content)
			}
		}
		return m, nil

	case DrawerMinimizeMsg:
		if m.shared != nil && m.shared.drawerStack != nil {
			switch msg.DrawerID {
			case "options":
				m.shared.drawerStack.ClearOptionsContent()
			case "plan":
				m.shared.drawerStack.ClearPlanContent()
			}
		}
		return m, nil

	// -----------------------------------------------------------------
	// Team update (immediate scan + drawer expand)
	// -----------------------------------------------------------------

	case TeamUpdateMsg:
		return m.handleTeamUpdate(msg)

	// -----------------------------------------------------------------
	// Tab flash animation (TUI-061)
	// -----------------------------------------------------------------

	case TabFlashMsg:
		return m.handleTabFlash(msg)

	// -----------------------------------------------------------------
	// Plan view / help modals — deactivate themselves on Esc/q.
	// -----------------------------------------------------------------

	case modals.PlanViewClosedMsg, modals.HelpClosedMsg:
		return m, nil

	// -----------------------------------------------------------------
	// Remaining CLI event types — re-subscribe without side effects.
	// -----------------------------------------------------------------

	case cli.SystemHookEvent, cli.SystemStatusEvent, cli.RateLimitEvent,
		cli.StreamEvent, cli.CLIUnknownEvent:
		return m, m.waitForCLIEvent()
	}

	// ─── FORWARDING CASCADE ──────────────────────────────────────────────
	// Contract: each forwarder returns non-nil ONLY for its own package's
	// unexported tick type. The cascade exits on the first non-nil cmd, so
	// a forwarder that claims a foreign type starves downstream forwarders.
	// All tick types are distinct named structs — do NOT use interface
	// matching here.
	// Order: 1. StatusLine  2. Toast  3. TabBar  4. TeamList  5. TeamDetail
	// ─────────────────────────────────────────────────────────────────────

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
		prevToastH := m.shared.toasts.Height()
		cmd := m.shared.toasts.HandleMsg(msg)
		if m.shared.toasts.Height() != prevToastH {
			m.propagateContentSizes()
		}
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

	// Forward unhandled messages to the team list for autonomous 2-second
	// poll ticks.  pollTickMsg is unexported from the teams package, so
	// AppModel cannot type-switch on it; forwarding here ensures the poll
	// cycle is self-sustaining once started by Init → startTeamPolling.
	if m.shared != nil && m.shared.teamList != nil {
		cmd := m.shared.teamList.HandleMsg(msg)
		if cmd != nil {
			// Poll tick processed — refresh teams drawer with latest health data.
			if m.shared.drawerStack != nil && m.shared.teamsHealth != nil {
				if m.shared.teamsHealth.HasData() {
					if !m.shared.drawerStack.TeamsHasContent() {
						// First team discovered — force-expand the drawer.
						m.shared.drawerStack.SetTeamsContent(m.shared.teamsHealth.View())
					} else if m.shared.teamsHealth.HasRunningTeam() &&
						m.shared.drawerStack.TeamsIsMinimized() {
						// A team is actively running and the drawer was minimized —
						// force-expand so the user sees live progress without manually
						// reopening the drawer.
						m.shared.drawerStack.SetTeamsContent(m.shared.teamsHealth.View())
					} else {
						m.shared.drawerStack.RefreshTeamsContent(m.shared.teamsHealth.View())
					}
				} else {
					// Don't clear teams drawer within 10s of a TeamUpdateMsg —
					// the team may not yet be visible to the filesystem scanner
					// (ensureTeamVisible race or symlink propagation delay).
					if m.shared.teamNotifiedAt.IsZero() || time.Since(m.shared.teamNotifiedAt) > 10*time.Second {
						m.shared.drawerStack.ClearTeamsContent()
					}
				}
			}
			// Populate status line team indicator from teamsHealth.
			tid := m.shared.teamsHealth.TeamIndicator()
			m.statusLine.TeamActive = tid.Active
			m.statusLine.TeamName = tid.Name
			m.statusLine.TeamMemberStatuses = tid.MemberStatuses
			m.statusLine.TeamCurrentWave = tid.CurrentWave
			m.statusLine.TeamTotalWaves = tid.TotalWaves
			m.statusLine.TeamCost = tid.Cost

			// Refresh the team detail panel so it shows up-to-date member state.
			if m.shared.teamDetail != nil {
				m.shared.teamDetail.Refresh()
			}
			return m, cmd
		}
	}

	// Forward unhandled messages to the team detail for TeamSelectedMsg
	// (emitted as a tea.Cmd result by the team list when the cursor moves).
	// TeamSelectedMsg is unexported from the teams package so AppModel cannot
	// type-switch on it directly; the detail model handles it internally.
	if m.shared != nil && m.shared.teamDetail != nil {
		m.shared.teamDetail.HandleMsg(msg)
	}

	// Forward unhandled messages to the claude panel for timestamp tick
	// processing (UX-024). timestampTickMsg is unexported from the claude
	// package, so AppModel cannot type-switch on it; forwarding here keeps the
	// self-scheduling 60-second tick alive as long as timestamps are enabled.
	if m.shared != nil && m.shared.claudePanel != nil {
		if cmd := m.shared.claudePanel.HandleMsg(msg); cmd != nil {
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
