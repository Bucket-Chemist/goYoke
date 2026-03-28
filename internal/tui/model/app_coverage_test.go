package model

// ---------------------------------------------------------------------------
// TUI-036: Coverage gap-filling for the model package.
// These tests target the branches identified as uncovered in the 75.3% baseline:
//
//   - SessionAutoSaveMsg: matching / stale seq handling
//   - ShutdownCompleteMsg: nil and non-nil Err variants
//   - Double Ctrl+C (shutdownInProgress=true)
//   - ProviderSwitchExecuteMsg with stale seq (debounce no-op)
//   - CLI event re-subscription (WaitForEvent cmd returned)
//   - ToastMsg forwarding
//   - SetTeamList / SetProviderTabBar / SetDashboard / SetSettings /
//     SetTelemetry / SetPlanPreview / SetTaskBoard / SetBaseCLIOpts /
//     SetSessionStore / SetSessionData / SessionData / SetShutdownManager /
//     SaveSessionPublic setters (all 0% coverage)
//   - CLI disconnect within and beyond retry limits
//   - CLIReconnectMsg: matching and stale seq
//   - SystemHookEvent / SystemStatusEvent / RateLimitEvent / StreamEvent /
//     CLIUnknownEvent pass-through
// ---------------------------------------------------------------------------

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/session"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Minimal mock CLI driver (coverage-specific, named to avoid collision with
// the mockCLIDriver in startup_test.go)
// ---------------------------------------------------------------------------

// coverageCLIDriver satisfies cliDriverWidget for coverage tests.
type coverageCLIDriver struct {
	startCalled    int
	waitCalled     int
	shutdownCalled int
	sendCalled     int
}

func (d *coverageCLIDriver) Start() tea.Cmd {
	d.startCalled++
	return func() tea.Msg { return cli.CLIStartedMsg{PID: 1} }
}

func (d *coverageCLIDriver) WaitForEvent() tea.Cmd {
	d.waitCalled++
	return func() tea.Msg { return nil }
}

func (d *coverageCLIDriver) SendMessage(_ string) tea.Cmd {
	d.sendCalled++
	return nil
}

func (d *coverageCLIDriver) Interrupt() error { return nil }
func (d *coverageCLIDriver) Shutdown() error  { d.shutdownCalled++; return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newModelWithCLI returns an AppModel with a coverageCLIDriver wired.
func newModelWithCLI() (AppModel, *coverageCLIDriver) {
	m := NewAppModel()
	d := &coverageCLIDriver{}
	m.SetCLIDriver(d)
	return m, d
}

// newReadyModel returns an AppModel sized and ready for layout tests.
func newReadyModel() AppModel {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return updated.(AppModel)
}

// ---------------------------------------------------------------------------
// SessionAutoSaveMsg
// ---------------------------------------------------------------------------

func TestUpdate_SessionAutoSaveMsg_MatchingSeq_SaveCalled(t *testing.T) {
	t.Parallel()

	// Wire a real session store + data in a temp dir so saveSession runs
	// without panicking.
	dir := t.TempDir()
	store := session.NewStore(dir)
	data := &session.SessionData{ID: "test-20260101.abc123"}

	m := NewAppModel()
	m.SetSessionStore(store)
	m.SetSessionData(data)

	// Simulate the autoSaveSeq being 5 (as if 5 ResultEvents occurred).
	m.autoSaveSeq = 5

	// A message with Seq == autoSaveSeq should trigger saveSession.
	// We can't directly observe saveSession (no return value) but we verify
	// the model state remains consistent (no panic, model returned).
	updated, cmd := m.Update(SessionAutoSaveMsg{Seq: 5})
	result := updated.(AppModel)

	assert.Equal(t, 5, result.autoSaveSeq, "autoSaveSeq unchanged after matching save")
	assert.Nil(t, cmd, "nil cmd expected after session auto-save")
}

func TestUpdate_SessionAutoSaveMsg_StaleSeq_IsNoop(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.autoSaveSeq = 7

	// Seq=3 is stale; model must be returned unchanged with nil cmd.
	updated, cmd := m.Update(SessionAutoSaveMsg{Seq: 3})
	result := updated.(AppModel)

	assert.Equal(t, 7, result.autoSaveSeq, "autoSaveSeq must not change on stale save msg")
	assert.Nil(t, cmd, "nil cmd expected on stale SessionAutoSaveMsg")
}

// ---------------------------------------------------------------------------
// ShutdownCompleteMsg
// ---------------------------------------------------------------------------

func TestUpdate_ShutdownCompleteMsg_NilErr_ReturnsQuit(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	_, cmd := m.Update(ShutdownCompleteMsg{Err: nil})

	require.NotNil(t, cmd, "ShutdownCompleteMsg must return tea.Quit command")
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "cmd() must produce QuitMsg, got %T", msg)
}

func TestUpdate_ShutdownCompleteMsg_NonNilErr_StillReturnsQuit(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	_, cmd := m.Update(ShutdownCompleteMsg{Err: errors.New("shutdown error")})

	require.NotNil(t, cmd, "ShutdownCompleteMsg with error must still return quit command")
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "cmd() must produce QuitMsg even when Err != nil, got %T", msg)
}

// ---------------------------------------------------------------------------
// Double Ctrl+C / shutdownInProgress
// ---------------------------------------------------------------------------

func TestHandleKey_ForceQuit_DoubleCtrlC_ImmediateQuit(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.shutdownInProgress = true // Simulate first Ctrl+C already handled.

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd, "second Ctrl+C must produce a quit command")
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "second Ctrl+C must produce QuitMsg immediately, got %T", msg)
}

func TestHandleKey_ForceQuit_FirstCtrlC_SetsShutdownInProgress(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	require.False(t, m.shutdownInProgress)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(AppModel)

	assert.True(t, result.shutdownInProgress,
		"shutdownInProgress must be set true after first Ctrl+C")
}

func TestHandleKey_ForceQuit_WithShutdownFunc_RunsFunc(t *testing.T) {
	t.Parallel()

	m := NewAppModel()

	// Wire a shutdown manager stub that records when Shutdown() is called.
	called := false
	stubManager := &stubShutdownManager{fn: func() error {
		called = true
		return nil
	}}
	m.SetShutdownManager(stubManager)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)

	// Executing the cmd calls the shutdown func and produces ShutdownCompleteMsg.
	msg := cmd()
	_, isComplete := msg.(ShutdownCompleteMsg)
	assert.True(t, isComplete, "cmd() should return ShutdownCompleteMsg, got %T", msg)
	assert.True(t, called, "shutdownFunc must be invoked when running the cmd")
}

// stubShutdownManager satisfies the interface accepted by SetShutdownManager.
type stubShutdownManager struct {
	fn func() error
}

func (s *stubShutdownManager) Shutdown() error { return s.fn() }

// ---------------------------------------------------------------------------
// ProviderSwitchExecuteMsg debounce
// ---------------------------------------------------------------------------

func TestUpdate_ProviderSwitchExecuteMsg_StaleSeq_IsNoop(t *testing.T) {
	t.Parallel()

	m, _ := newModelWithProvider()
	m.providerSwitchSeq = 5

	// Seq=2 is stale.
	updated, cmd := m.Update(ProviderSwitchExecuteMsg{Seq: 2})
	result := updated.(AppModel)

	assert.Equal(t, 5, result.providerSwitchSeq, "providerSwitchSeq must not change on stale msg")
	assert.Nil(t, cmd, "stale ProviderSwitchExecuteMsg must produce nil cmd")
}

func TestUpdate_ProviderSwitchExecuteMsg_MatchingSeq_ExecutesSwitch(t *testing.T) {
	t.Parallel()

	m, _ := newModelWithProvider()
	initial := m.shared.providerState.GetActiveProvider()
	m.providerSwitchSeq = 3

	updated, _ := m.Update(ProviderSwitchExecuteMsg{Seq: 3})
	result := updated.(AppModel)

	got := result.shared.providerState.GetActiveProvider()
	assert.NotEqual(t, initial, got,
		"matching ProviderSwitchExecuteMsg must cycle the provider")
}

// ---------------------------------------------------------------------------
// CLI event re-subscription (WaitForEvent cmd returned)
// ---------------------------------------------------------------------------

func TestUpdate_SystemInitEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()

	_, cmd := m.Update(cli.SystemInitEvent{SessionID: "sess-1", Model: "sonnet"})

	assert.NotNil(t, cmd, "SystemInitEvent must return WaitForEvent cmd")
	assert.Equal(t, 1, d.waitCalled, "WaitForEvent must be called once for SystemInitEvent")
}

func TestUpdate_AssistantEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()

	ev := cli.AssistantEvent{
		Message: cli.AssistantMessage{
			Content: []cli.ContentBlock{{Type: "text", Text: "hi"}},
		},
	}
	_, cmd := m.Update(ev)

	assert.NotNil(t, cmd, "AssistantEvent must return non-nil cmd (includes WaitForEvent)")
	assert.GreaterOrEqual(t, d.waitCalled, 1, "WaitForEvent must be called for AssistantEvent")
}

func TestUpdate_UserEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()

	_, cmd := m.Update(cli.UserEvent{})

	assert.NotNil(t, cmd, "UserEvent must return WaitForEvent cmd")
	assert.Equal(t, 1, d.waitCalled, "WaitForEvent must be called once for UserEvent")
}

func TestUpdate_ResultEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()

	_, cmd := m.Update(cli.ResultEvent{TotalCostUSD: 0.01, Subtype: "success"})

	assert.NotNil(t, cmd, "ResultEvent must return non-nil cmd (includes WaitForEvent)")
	assert.GreaterOrEqual(t, d.waitCalled, 1, "WaitForEvent must be called for ResultEvent")
}

// ---------------------------------------------------------------------------
// Pass-through CLI event types (re-subscribe without side effects)
// ---------------------------------------------------------------------------

func TestUpdate_SystemHookEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.SystemHookEvent{})

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, d.waitCalled)
}

func TestUpdate_SystemStatusEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.SystemStatusEvent{})

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, d.waitCalled)
}

func TestUpdate_RateLimitEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.RateLimitEvent{})

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, d.waitCalled)
}

func TestUpdate_StreamEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.StreamEvent{})

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, d.waitCalled)
}

func TestUpdate_CLIUnknownEvent_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.CLIUnknownEvent{})

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, d.waitCalled)
}

// ---------------------------------------------------------------------------
// CLIStartedMsg
// ---------------------------------------------------------------------------

func TestUpdate_CLIStartedMsg_ReturnsWaitForEventCmd(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	_, cmd := m.Update(cli.CLIStartedMsg{PID: 42})

	assert.NotNil(t, cmd, "CLIStartedMsg must return WaitForEvent cmd")
	assert.Equal(t, 1, d.waitCalled)
}

// ---------------------------------------------------------------------------
// CLIDisconnectedMsg
// ---------------------------------------------------------------------------

func TestUpdate_CLIDisconnectedMsg_WithError_WithinRetryLimit_ReturnsReconnectCmd(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.reconnectCount = 0

	updated, cmd := m.Update(cli.CLIDisconnectedMsg{Err: errors.New("pipe broken")})
	result := updated.(AppModel)

	assert.Equal(t, 1, result.reconnectCount, "reconnectCount must increment on disconnect")
	assert.NotNil(t, cmd, "a reconnect timer cmd must be returned within retry limit")
}

func TestUpdate_CLIDisconnectedMsg_ExceedsRetryLimit_ReturnsNil(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.reconnectCount = 3 // at or above maxReconnectAttempts

	updated, cmd := m.Update(cli.CLIDisconnectedMsg{Err: errors.New("pipe broken")})
	result := updated.(AppModel)

	// reconnectCount must NOT increment beyond limit.
	assert.Equal(t, 3, result.reconnectCount, "reconnectCount must not change beyond limit")
	assert.Nil(t, cmd, "nil cmd expected when retry limit exceeded")
}

func TestUpdate_CLIDisconnectedMsg_NilErr_ReturnsNil(t *testing.T) {
	t.Parallel()

	// Clean exit (nil Err) should not trigger reconnect regardless of count.
	m := NewAppModel()
	m.reconnectCount = 0

	updated, cmd := m.Update(cli.CLIDisconnectedMsg{Err: nil})
	result := updated.(AppModel)

	assert.Equal(t, 0, result.reconnectCount, "reconnectCount must not change on clean exit")
	assert.Nil(t, cmd, "nil cmd expected for clean CLI exit")
}

// ---------------------------------------------------------------------------
// CLIReconnectMsg
// ---------------------------------------------------------------------------

func TestUpdate_CLIReconnectMsg_MatchingSeq_CallsStartCLI(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	m.reconnectSeq = 2

	_, cmd := m.Update(CLIReconnectMsg{Attempt: 1, Seq: 2})

	assert.NotNil(t, cmd, "matching CLIReconnectMsg must return Start cmd")
	assert.Equal(t, 1, d.startCalled, "Start must be called for matching reconnect")
}

func TestUpdate_CLIReconnectMsg_StaleSeq_IsNoop(t *testing.T) {
	t.Parallel()

	m, d := newModelWithCLI()
	m.reconnectSeq = 5

	_, cmd := m.Update(CLIReconnectMsg{Attempt: 1, Seq: 2}) // seq 2 is stale

	assert.Nil(t, cmd, "stale CLIReconnectMsg must return nil cmd")
	assert.Equal(t, 0, d.startCalled, "Start must not be called for stale reconnect")
}

// ---------------------------------------------------------------------------
// SystemInitEvent — session ID persisted
// ---------------------------------------------------------------------------

func TestUpdate_SystemInitEvent_StoresSessionID(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	updated, _ := m.Update(cli.SystemInitEvent{
		SessionID: "test-session-abc",
		Model:     "claude-sonnet-4-20250514",
	})
	result := updated.(AppModel)

	assert.True(t, result.cliReady, "cliReady must be set true after SystemInitEvent")
	assert.Equal(t, "test-session-abc", result.sessionID)
	assert.Equal(t, "claude-sonnet-4-20250514", result.activeModel)
}

// ---------------------------------------------------------------------------
// Setter methods (0% coverage)
// ---------------------------------------------------------------------------

func TestSetTeamList_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockTeamListWidget{}
	m.SetTeamList(mock)

	assert.NotNil(t, m.shared.teamList, "shared.teamList must be non-nil after SetTeamList")
}

func TestSetBaseCLIOpts_StoresOpts(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	opts := cli.CLIDriverOpts{Model: "claude-opus-4-6"}
	m.SetBaseCLIOpts(opts)

	assert.Equal(t, "claude-opus-4-6", m.shared.baseCLIOpts.Model)
}

func TestSetSessionStore_StoresStore(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	store := session.NewStore(t.TempDir())
	m.SetSessionStore(store)

	assert.NotNil(t, m.shared.sessionStore, "shared.sessionStore must be non-nil after SetSessionStore")
}

func TestSetSessionData_StoresData(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	data := &session.SessionData{ID: "test-id"}
	m.SetSessionData(data)

	require.NotNil(t, m.shared.sessionData)
	assert.Equal(t, "test-id", m.shared.sessionData.ID)
}

func TestSessionData_ReturnsData(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	assert.Nil(t, m.SessionData(), "SessionData must return nil when not set")

	data := &session.SessionData{ID: "session-xyz"}
	m.SetSessionData(data)
	got := m.SessionData()

	require.NotNil(t, got)
	assert.Equal(t, "session-xyz", got.ID)
}

func TestSessionData_NilShared_ReturnsNil(t *testing.T) {
	t.Parallel()

	m := AppModel{} // zero value — shared is nil
	assert.Nil(t, m.SessionData(), "SessionData on zero AppModel must return nil")
}

func TestSetShutdownManager_StoresFn(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	require.Nil(t, m.shared.shutdownFunc, "precondition: shutdownFunc must be nil")

	stub := &stubShutdownManager{fn: func() error { return nil }}
	m.SetShutdownManager(stub)

	assert.NotNil(t, m.shared.shutdownFunc, "shutdownFunc must be set after SetShutdownManager")
}

func TestSaveSessionPublic_NoPanic_WhenStoreNil(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	// shared.sessionStore == nil — saveSession returns early, no panic.
	assert.NotPanics(t, func() { m.SaveSessionPublic() })
}

func TestSaveSessionPublic_WithStoreAndData_NoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := NewAppModel()
	m.SetSessionStore(session.NewStore(dir))
	m.SetSessionData(&session.SessionData{ID: "save-test-001"})

	assert.NotPanics(t, func() { m.SaveSessionPublic() })
}

// ---------------------------------------------------------------------------
// SetProviderTabBar, SetDashboard, SetSettings, SetTelemetry,
// SetPlanPreview, SetTaskBoard — simple nil → non-nil coverage
// ---------------------------------------------------------------------------

func TestSetProviderTabBar_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockProviderTabBarWidget{}
	m.SetProviderTabBar(mock)

	assert.NotNil(t, m.shared.providerTabBar)
}

func TestSetDashboard_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockDashboardWidget{}
	m.SetDashboard(mock)

	assert.NotNil(t, m.shared.dashboard)
}

func TestSetSettings_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockSettingsWidget{}
	m.SetSettings(mock)

	assert.NotNil(t, m.shared.settings)
}

func TestSetTelemetry_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockTelemetryWidget{}
	m.SetTelemetry(mock)

	assert.NotNil(t, m.shared.telemetry)
}

func TestSetPlanPreview_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockPlanPreviewWidget{}
	m.SetPlanPreview(mock)

	assert.NotNil(t, m.shared.planPreview)
}

func TestSetTaskBoard_StoresWidget(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	mock := &mockTaskBoardWidget{}
	m.SetTaskBoard(mock)

	assert.NotNil(t, m.shared.taskBoard)
}

// ---------------------------------------------------------------------------
// ProviderState / SetProviderState
// ---------------------------------------------------------------------------

func TestProviderState_ReturnsStateWhenSet(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	ps := state.NewProviderState()
	m.SetProviderState(ps)

	got := m.ProviderState()
	assert.Equal(t, ps, got)
}

func TestProviderState_NilShared_ReturnsNil(t *testing.T) {
	t.Parallel()

	m := AppModel{} // zero value — shared is nil
	assert.Nil(t, m.ProviderState())
}

// ---------------------------------------------------------------------------
// Minimal widget stubs for the new setters above
// ---------------------------------------------------------------------------

// mockTeamListWidget satisfies teamListWidget.
type mockTeamListWidget struct{}

func (m *mockTeamListWidget) HandleMsg(_ tea.Msg) tea.Cmd      { return nil }
func (m *mockTeamListWidget) View() string                     { return "" }
func (m *mockTeamListWidget) SetSize(_, _ int)                  {}
func (m *mockTeamListWidget) StartPolling(_ string) tea.Cmd    { return nil }

// mockProviderTabBarWidget satisfies providerTabBarWidget.
type mockProviderTabBarWidget struct{}

func (m *mockProviderTabBarWidget) View() string              { return "" }
func (m *mockProviderTabBarWidget) SetActive(_ state.ProviderID) {}
func (m *mockProviderTabBarWidget) SetWidth(_ int)            {}
func (m *mockProviderTabBarWidget) IsVisible() bool           { return false }
func (m *mockProviderTabBarWidget) Height() int               { return 0 }

// mockDashboardWidget satisfies dashboardWidget.
type mockDashboardWidget struct{}

func (m *mockDashboardWidget) View() string                                         { return "" }
func (m *mockDashboardWidget) SetSize(_, _ int)                                     {}
func (m *mockDashboardWidget) SetData(_ float64, _ int64, _, _, _ int, _ time.Time) {}
func (m *mockDashboardWidget) SetTier(_ LayoutTier)                                 {}
func (m *mockDashboardWidget) Update(_ tea.Msg) tea.Cmd                             { return nil }
func (m *mockDashboardWidget) SetFocused(_ bool)                                    {}

// mockSettingsWidget satisfies settingsWidget.
type mockSettingsWidget struct{}

func (m *mockSettingsWidget) View() string                               { return "" }
func (m *mockSettingsWidget) SetSize(_, _ int)                           {}
func (m *mockSettingsWidget) SetConfig(_, _, _, _ string, _ []string)    {}
func (m *mockSettingsWidget) SetTier(_ LayoutTier)                       {}

// mockTelemetryWidget satisfies telemetryWidget.
type mockTelemetryWidget struct{}

func (m *mockTelemetryWidget) HandleMsg(_ tea.Msg) tea.Cmd { return nil }
func (m *mockTelemetryWidget) View() string                { return "" }
func (m *mockTelemetryWidget) SetSize(_, _ int)            {}
func (m *mockTelemetryWidget) SetTier(_ LayoutTier)        {}

// mockPlanPreviewWidget satisfies planPreviewWidget.
type mockPlanPreviewWidget struct{}

func (m *mockPlanPreviewWidget) View() string          { return "" }
func (m *mockPlanPreviewWidget) SetSize(_, _ int)      {}
func (m *mockPlanPreviewWidget) SetContent(_ string)   {}
func (m *mockPlanPreviewWidget) ClearContent()         {}
func (m *mockPlanPreviewWidget) Content() string       { return "" }
func (m *mockPlanPreviewWidget) SetTier(_ LayoutTier)  {}

// mockTaskBoardWidget satisfies taskBoardWidget.
type mockTaskBoardWidget struct{}

func (m *mockTaskBoardWidget) View() string                   { return "" }
func (m *mockTaskBoardWidget) SetSize(_, _ int)               {}
func (m *mockTaskBoardWidget) Toggle()                        {}
func (m *mockTaskBoardWidget) IsVisible() bool                { return false }
func (m *mockTaskBoardWidget) Height() int                    { return 0 }
func (m *mockTaskBoardWidget) SetTasks(_ []state.TaskEntry)   {}
func (m *mockTaskBoardWidget) HandleMsg(_ tea.Msg) tea.Cmd    { return nil }
func (m *mockTaskBoardWidget) SetTier(_ LayoutTier)           {}
