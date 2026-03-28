// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all widget interface definitions used by AppModel to
// decouple from concrete component types and avoid circular imports.
package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	// HandleMsg forwards a tea.Msg to the tab bar for internal processing.
	// It mutates the receiver in place and returns any Cmd to schedule.
	// Currently used to deliver TabFlashMsg (TUI-061) to the flash animation
	// and to process tab-switching keys (Alt+C, Alt+A, Alt+T, Alt+Y).
	HandleMsg(msg tea.Msg) tea.Cmd
	// ActiveTab returns the currently active tab ID.
	ActiveTab() TabID
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
	// Interrupt sends SIGINT to the CLI subprocess, requesting it to stop
	// the current streaming response. Called by the Escape key handler.
	Interrupt() error
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

// MessageSender is a minimal interface for submitting a user message to the
// active CLI session.  It is defined here in the model package so that both
// the model package and the claude package (which imports model) can refer to
// the same named type, satisfying Go's strict interface method-signature
// matching without creating a circular import.
type MessageSender interface {
	// SendMessage submits text to the CLI driver and returns a Cmd that
	// delivers the result as a tea.Msg when complete.
	SendMessage(text string) tea.Cmd
}

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
	// SetSender updates the CLI driver used to submit user messages.
	// Called after a provider switch to wire the new driver into the panel.
	SetSender(s MessageSender)
	// SetTier notifies the component of the current responsive layout tier.
	// Components may use this to adapt their rendering in future tickets.
	SetTier(tier LayoutTier)
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
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
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
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
	// Update processes a tea.Msg when the dashboard holds keyboard focus.
	// The concrete type mutates itself in place and returns only the Cmd,
	// following the same pointer-receiver mutation pattern used by other
	// widget interfaces (claudePanelWidget, toastWidget, etc.).
	Update(msg tea.Msg) tea.Cmd
	// SetFocused controls whether the dashboard processes keyboard events.
	SetFocused(focused bool)
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
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
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
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
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
	// Content returns the raw markdown string currently loaded in the panel.
	// Returns "" when no plan has been set.
	Content() string
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
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
	SetTasks(tasks []state.TaskEntry)
	// HandleMsg forwards keyboard messages to the task board when it is
	// visible, enabling cursor navigation and filter mode switching.
	HandleMsg(msg tea.Msg) tea.Cmd
	// SetTier notifies the component of the current responsive layout tier.
	SetTier(tier LayoutTier)
}

// ---------------------------------------------------------------------------
// settingsTreeWidget
//
// settingsTreeWidget is the interface satisfied by
// settingstree.SettingsTreeModel. The settingstree package has no dependency
// on the model package, so the interface is defined here to keep the widget
// coupling pattern consistent with the other widget interfaces and to avoid
// any future circular import.
// ---------------------------------------------------------------------------

// settingsTreeWidget is the interface satisfied by settingstree.SettingsTreeModel.
type settingsTreeWidget interface {
	// Init returns the initial command (nil for this component).
	Init() tea.Cmd
	// View renders the settings tree to a string.
	View() string
	// SetSize updates the viewport dimensions.
	SetSize(w, h int)
	// SetFocused sets keyboard-focus state.
	SetFocused(bool)
	// SetValue updates the value of the node identified by key.
	// Intended for Display nodes whose values originate outside the component
	// (model name, git branch, etc.). A no-op when key is not found.
	SetValue(key, value string)
}

// ---------------------------------------------------------------------------
// slashCmdWidget
//
// slashCmdWidget is the interface satisfied by *slashcmd.SlashCmdModel.
// The slashcmd package has no dependency on the model package, so there is no
// circular import. The interface is defined here to keep the widget coupling
// pattern consistent with the other widget interfaces.
// ---------------------------------------------------------------------------

// slashCmdWidget is the interface satisfied by *slashcmd.SlashCmdModel.
type slashCmdWidget interface {
	// Show makes the dropdown visible and applies an initial filter query.
	Show(query string)
	// Hide closes the dropdown without emitting a selection message.
	Hide()
	// IsVisible returns true when the dropdown is currently shown.
	IsVisible() bool
	// Filter updates the filter query; hides the dropdown when no matches exist.
	Filter(query string)
	// View renders the dropdown to a string. Returns "" when not visible.
	View() string
	// SetWidth updates the terminal width used for rendering.
	SetWidth(w int)
}

// ---------------------------------------------------------------------------
// breadcrumbWidget  (TUI-063)
//
// breadcrumbWidget is the interface satisfied by *breadcrumb.BreadcrumbModel.
// Defining it here avoids a circular import: breadcrumb imports config but not
// model; model must not import breadcrumb directly.  The interface breaks the
// cycle while still allowing AppModel to drive the breadcrumb trail.
//
// SetCrumbs accepts plain label strings so that the model package does not
// need to reference breadcrumb.BreadcrumbItem.  The concrete implementation
// wraps each label into a BreadcrumbItem internally.
// ---------------------------------------------------------------------------

// breadcrumbWidget is the interface satisfied by *breadcrumb.BreadcrumbModel.
type breadcrumbWidget interface {
	// View renders the breadcrumb trail to a string.
	// Returns "" when no crumbs are set (no row should be allocated).
	View() string
	// SetCrumbs replaces the current navigation trail with label-only items.
	SetCrumbs(labels []string)
	// SetWidth updates the terminal width used for left-truncation.
	SetWidth(width int)
}

// ---------------------------------------------------------------------------
// hintBarWidget  (TUI-060)
//
// hintBarWidget is the interface satisfied by *hintbar.HintBarModel.
// Defining it here avoids a circular import: hintbar imports config but not
// model; model must not import hintbar directly.  The interface breaks the
// cycle while still allowing AppModel to drive the hint bar lifecycle.
// ---------------------------------------------------------------------------

// hintBarWidget is the interface satisfied by *hintbar.HintBarModel.
type hintBarWidget interface {
	// View renders the hint bar to a string. Returns "" when not visible.
	View() string
	// SetContext switches the active hint set by context name.
	// Recognised values: "main", "settings", "search", "modal", "plan".
	// Unknown values fall back to "main".
	SetContext(context string)
	// SetWidth updates the terminal width used for truncation.
	SetWidth(width int)
	// IsVisible returns true when the hint bar is currently shown.
	IsVisible() bool
	// Show makes the hint bar visible.
	Show()
	// Hide hides the hint bar.
	Hide()
}

// ---------------------------------------------------------------------------
// searchOverlayWidget  (TUI-059)
//
// searchOverlayWidget is the interface satisfied by *search.SearchOverlayModel.
// Defining it here keeps AppModel decoupled from the concrete search package
// (model must not import search — that would import cycle: search → state,
// model → agents → state, model → search → state is fine, but model →
// search while search imports state and model imports agents which does not
// import model means there is no cycle; however search does NOT import model,
// so there is no issue either way).
//
// state.SearchResult and state.SearchSource live in the state package so that
// components (claude, agents) that already import state can implement
// state.SearchSource without creating a circular import through model.
//
// HandleMsg follows the same pointer-receiver mutation pattern used by
// claudePanelWidget and toastWidget: the concrete type mutates itself in place
// and returns only the tea.Cmd, avoiding the self-returning interface problem
// that would arise from returning (searchOverlayWidget, tea.Cmd).
// ---------------------------------------------------------------------------

// searchOverlayWidget is the interface satisfied by *search.SearchOverlayModel.
type searchOverlayWidget interface {
	// HandleMsg processes a tea.Msg, mutates the widget in place, and
	// returns any Cmd to run.
	HandleMsg(msg tea.Msg) tea.Cmd
	// View renders the overlay to a string. Returns "" when not active.
	View() string
	// SetSize updates the terminal dimensions used for layout and centering.
	SetSize(width, height int)
	// IsActive returns true when the overlay is currently displayed.
	IsActive() bool
	// Activate shows the overlay and focuses the query input.
	Activate()
	// Deactivate hides the overlay and clears the query input.
	Deactivate()
	// SetSources replaces the registered search sources.
	SetSources(sources []state.SearchSource)
}

// ---------------------------------------------------------------------------
// drawerStackWidget  (TDS-004)
//
// drawerStackWidget is the interface satisfied by drawer.DrawerStack.
// Defining it here avoids a circular import: the drawer package imports
// config but not model; model must not import drawer directly.  Proxy
// methods (SetOptionsContent, etc.) avoid returning concrete drawer types
// through the interface, which would create import cycles.
// ---------------------------------------------------------------------------

// drawerStackWidget is the interface satisfied by drawer.DrawerStack.
type drawerStackWidget interface {
	View() string
	SetSize(w, h int)
	ExpandedDrawers() []string
	HandleKey(focusedDrawer string, msg tea.KeyMsg) tea.Cmd
	// Proxy methods avoid returning concrete types and import cycles.
	SetOptionsContent(content string)
	ClearOptionsContent()
	OptionsHasContent() bool
	SetPlanContent(content string)
	ClearPlanContent()
	PlanHasContent() bool
	SetOptionsFocused(focused bool)
	SetPlanFocused(focused bool)
	// Modal routing (TDS-006).
	SetActiveModal(requestID string, message string, options []string)
	HasActiveModal() bool
	OptionsActiveRequestID() string
	OptionsSelectedOption() string
}

// ---------------------------------------------------------------------------
// cwdSelectorWidget
//
// cwdSelectorWidget is the interface satisfied by *cwdselector.Model.
// Defining it here avoids a circular import: cwdselector imports config
// but not model; model must not import cwdselector directly.
// ---------------------------------------------------------------------------

// cwdSelectorWidget is the interface satisfied by *cwdselector.Model.
// It follows the pointer-receiver mutation pattern: the concrete type
// mutates itself in place and returns only the tea.Cmd.
type cwdSelectorWidget interface {
	// IsActive returns true when the modal is visible.
	IsActive() bool
	// Show makes the modal visible and resets selection state.
	Show()
	// SetSize updates the terminal dimensions for centering.
	SetSize(w, h int)
	// View renders the modal overlay. Returns "" when not active.
	View() string
	// HandleMsg processes a tea.Msg, mutates the widget in place, and
	// returns any Cmd to run.
	HandleMsg(msg tea.Msg) tea.Cmd
}
