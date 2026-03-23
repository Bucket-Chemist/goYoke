// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the root AppModel: the single top-level tea.Model that
// owns all application state and implements The Elm Architecture.
//
// Downstream tickets fill in the placeholder child models:
//   - ClaudePanelModel — TUI-022 (Claude conversation panel)
//   - AgentTreeModel   — TUI-020 (Agent tree view + detail)
//   - AgentDetailModel — TUI-020 (Agent tree view + detail)
//   - ToastModel       — TUI-025 (Toast notification system)
package model

import (
	"encoding/json"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/banner"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
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
	cliDriver   cliDriverWidget
	bridge      bridgeWidget
	modalQueue  *modals.ModalQueue
	permHandler *modals.PermissionHandler
}

// ---------------------------------------------------------------------------
// Layout constants
//
// These define the fixed-height allocations for chrome rows.
// ---------------------------------------------------------------------------

const (
	bannerHeight    = 3 // rounded border top + title + border bottom
	tabBarHeight    = 1 // single-row strip
	statusLineHeight = 2 // two-row status bar
	borderFrame     = 2 // border chars on each axis (1 left + 1 right)
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
// Placeholder child model types
//
// Each placeholder is an empty struct.  Later tickets replace these stubs
// with full implementations.  The TODO comments note the ticket responsible.
// ---------------------------------------------------------------------------

// ClaudePanelModel is a placeholder for the Claude conversation panel.
// TODO(TUI-022): implement conversation history + text input.
type ClaudePanelModel struct{}

// AgentTreeModel is a placeholder for the agent list tree.
// TODO(TUI-020): implement hierarchical agent list with status icons.
type AgentTreeModel struct{}

// AgentDetailModel is a placeholder for the agent detail view.
// TODO(TUI-020): implement agent detail panel with tool history.
type AgentDetailModel struct{}

// ToastModel is a placeholder for the toast notification overlay.
// TODO(TUI-025): implement transient toast stack with level colours.
type ToastModel struct{}

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

	// Child model placeholders (filled by downstream tickets)
	claudePanel ClaudePanelModel
	agentTree   AgentTreeModel
	agentDetail AgentDetailModel
	toasts      ToastModel

	// Diff history (post-hoc diffs from Write/Edit/Bash tool results).
	// TUI-022 renders these inline in the Claude panel.
	diffs []DiffEntry

	// Infrastructure
	keys config.KeyMap

	// Shared external components (see sharedState for rationale).
	// Access via m.shared.cliDriver and m.shared.bridge.
	shared *sharedState

	// Startup / session state (TUI-016).
	cliReady       bool   // true after SystemInitEvent processed
	sessionID      string // from SystemInitEvent
	activeModel    string // from SystemInitEvent
	reconnectCount int    // number of reconnection attempts made
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
		modalQueue:  &mq,
		permHandler: modals.NewPermissionHandler(&mq),
	}
	return AppModel{
		focus:          FocusClaude,
		rightPanelMode: RPMAgents,
		activeTab:      TabChat,
		keys:           keys,
		banner:         banner.NewBannerModel(0),
		statusLine:     statusline.NewStatusLineModel(0),
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

		// Propagate terminal size to modal queue for correct centering.
		if m.shared != nil && m.shared.modalQueue != nil {
			m.shared.modalQueue.SetTermSize(msg.Width, msg.Height)
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
		return m, m.waitForCLIEvent()

	case cli.AssistantEvent:
		// Assistant turn fragment — placeholder; TUI-022 fills this in.
		return m, m.waitForCLIEvent()

	case cli.UserEvent:
		// Extract post-hoc diffs from tool_use_result events.
		m = m.extractDiffs(msg)
		return m, m.waitForCLIEvent()

	case cli.ResultEvent:
		// Session turn complete — update cost display.
		m.statusLine.SessionCost = msg.TotalCostUSD
		return m, m.waitForCLIEvent()

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
		// Agent lifecycle events — placeholder; TUI-019/020 fills these in.
		return m, m.waitForCLIEvent()

	case ToastMsg:
		// Toast notification — placeholder; TUI-025 fills this in.
		return m, m.waitForCLIEvent()

	// -----------------------------------------------------------------
	// Remaining CLI event types — re-subscribe without side effects.
	// -----------------------------------------------------------------

	case cli.SystemHookEvent, cli.SystemStatusEvent, cli.RateLimitEvent,
		cli.StreamEvent, cli.CLIUnknownEvent:
		return m, m.waitForCLIEvent()
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
		return m, nil

	case key.Matches(msg, m.keys.Global.CycleRightPanel):
		m.rightPanelMode = NextRightPanelMode(m.rightPanelMode)
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
// Full implementation deferred to TUI-010.
func (m AppModel) handleClaudeKey(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

// handleAgentsKey processes key events when the agent tree holds focus.
// Full implementation deferred to TUI-011.
func (m AppModel) handleAgentsKey(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
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
	dims.contentHeight = m.height - bannerHeight - tabBarHeight - statusLineHeight
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

	return lipgloss.JoinVertical(lipgloss.Top,
		bannerView,
		tabBarView,
		mainArea,
		statusLineView,
	)
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

	// Placeholder content — replaced by TUI-010 ClaudePanelModel.
	content := config.StyleSubtle.Render("Claude panel  [focus=" + m.focus.String() + "]")

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

	// Placeholder content — replaced by TUI-011 / TUI-012 components.
	content := config.StyleSubtle.Render(m.rightPanelMode.String() + "  [panel=" + m.rightPanelMode.String() + "]")

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
