// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all setter/injector methods on AppModel.
// Extracted from app.go as part of TUI-043.
package model

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/session"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

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

// SetSearchOverlay injects the unified search overlay into the shared state.
// The overlay is activated by ctrl+f (TUI-059).
func (m *AppModel) SetSearchOverlay(so searchOverlayWidget) {
	m.shared.searchOverlay = so
}

// SetHintBar injects the context-aware keyboard hint bar into the shared
// state. The hint bar renders a single muted row of shortcuts below the
// main content area (TUI-060).
func (m *AppModel) SetHintBar(hb hintBarWidget) {
	m.shared.hintBar = hb
}

// SetTheme updates the active theme in shared state.  It is called when the
// user explicitly selects a new color theme (e.g. from the settings panel or
// via ThemeChangedMsg).  The pointer receiver is required because this method
// mutates the shared heap-allocated state.
//
// Components that hold a reference to sharedState will see the new theme on
// their next render cycle without any additional message dispatch.
func (m *AppModel) SetTheme(t config.Theme) {
	if m.shared != nil {
		m.shared.activeTheme = &t
	}
}

// Theme returns the current active theme.  It uses a value receiver because
// it is read-only.  If shared state has not been initialised or activeTheme
// is nil (should not occur after NewAppModel), DefaultTheme is returned as a
// safe fallback.
func (m AppModel) Theme() config.Theme {
	if m.shared != nil && m.shared.activeTheme != nil {
		return *m.shared.activeTheme
	}
	return config.DefaultTheme()
}
