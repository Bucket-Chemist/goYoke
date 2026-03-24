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
	SetTasks(tasks []state.TaskEntry)
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
