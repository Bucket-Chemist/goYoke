package model

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
)

// ---------------------------------------------------------------------------
// Mock types for interface testing
// ---------------------------------------------------------------------------

// mockClaudePanel satisfies claudePanelWidget for testing.
type mockClaudePanel struct {
	handleMsgCalled bool
	lastMsg         tea.Msg
	viewOutput      string
	focused         bool
	width, height   int
}

func (m *mockClaudePanel) HandleMsg(msg tea.Msg) tea.Cmd {
	m.handleMsgCalled = true
	m.lastMsg = msg
	return nil
}
func (m *mockClaudePanel) View() string          { return m.viewOutput }
func (m *mockClaudePanel) SetSize(w, h int)      { m.width = w; m.height = h }
func (m *mockClaudePanel) SetFocused(f bool)     { m.focused = f }
func (m *mockClaudePanel) IsStreaming() bool      { return false }

// mockToast satisfies toastWidget for testing.
type mockToast struct {
	handleMsgCalled bool
	empty           bool
	viewOutput      string
	width, height   int
}

func (m *mockToast) HandleMsg(msg tea.Msg) tea.Cmd {
	m.handleMsgCalled = true
	return nil
}
func (m *mockToast) View() string      { return m.viewOutput }
func (m *mockToast) SetSize(w, h int)  { m.width = w; m.height = h }
func (m *mockToast) IsEmpty() bool     { return m.empty }

// ---------------------------------------------------------------------------
// TabID
// ---------------------------------------------------------------------------

func TestTabIDString(t *testing.T) {
	tests := []struct {
		name     string
		tab      TabID
		expected string
	}{
		{"TabChat", TabChat, "Chat"},
		{"TabAgentConfig", TabAgentConfig, "Agent Config"},
		{"TabTeamConfig", TabTeamConfig, "Team Config"},
		{"TabTelemetry", TabTelemetry, "Telemetry"},
		{"unknown", TabID(99), "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.tab.String()
			if got != tc.expected {
				t.Errorf("TabID(%d).String() = %q; want %q", int(tc.tab), got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewAppModel
// ---------------------------------------------------------------------------

func TestNewAppModel_Defaults(t *testing.T) {
	m := NewAppModel()

	if m.focus != FocusClaude {
		t.Errorf("focus = %v; want FocusClaude", m.focus)
	}
	if m.rightPanelMode != RPMAgents {
		t.Errorf("rightPanelMode = %v; want RPMAgents", m.rightPanelMode)
	}
	if m.activeTab != TabChat {
		t.Errorf("activeTab = %v; want TabChat", m.activeTab)
	}
	if m.ready {
		t.Error("ready = true; want false before first WindowSizeMsg")
	}
	// shared state must be initialised by NewAppModel.
	if m.shared == nil {
		t.Fatal("shared = nil; want non-nil sharedState")
	}
	if m.shared.modalQueue == nil {
		t.Error("shared.modalQueue = nil; want non-nil ModalQueue")
	}
	if m.shared.permHandler == nil {
		t.Error("shared.permHandler = nil; want non-nil PermissionHandler")
	}
}

func TestNewAppModel_SharedStateRegistries(t *testing.T) {
	m := NewAppModel()
	if m.shared.agentRegistry == nil {
		t.Error("shared.agentRegistry = nil; want non-nil")
	}
	if m.shared.costTracker == nil {
		t.Error("shared.costTracker = nil; want non-nil")
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInit_ReturnsEnterAltScreen(t *testing.T) {
	m := NewAppModel()
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("Init() returned nil; want tea.EnterAltScreen command")
	}

	// tea.EnterAltScreen is a sequence command.  We verify it is non-nil and
	// not tea.Quit by executing it and checking the message type is not an
	// exit message.  A more robust check would use bubbletea internals, but
	// the public API only guarantees cmd != nil.
	msg := cmd()
	if _, isQuit := msg.(tea.QuitMsg); isQuit {
		t.Error("Init() returned tea.Quit; want tea.EnterAltScreen")
	}
}

// ---------------------------------------------------------------------------
// Update — WindowSizeMsg
// ---------------------------------------------------------------------------

func TestUpdate_WindowSizeMsg_StoresDimensions(t *testing.T) {
	m := NewAppModel()

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(AppModel)

	if result.width != 120 {
		t.Errorf("width = %d; want 120", result.width)
	}
	if result.height != 40 {
		t.Errorf("height = %d; want 40", result.height)
	}
	if !result.ready {
		t.Error("ready = false after WindowSizeMsg; want true")
	}
	if cmd != nil {
		t.Error("cmd != nil after WindowSizeMsg; want nil")
	}
}

func TestUpdate_WindowSizeMsg_SetsReady(t *testing.T) {
	m := NewAppModel()

	if m.ready {
		t.Fatal("precondition failed: ready should be false before first WindowSizeMsg")
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	result := updated.(AppModel)

	if !result.ready {
		t.Error("ready = false; want true after WindowSizeMsg")
	}
}

func TestUpdate_WindowSizeMsg_PropagatesWidthToChildren(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(AppModel)

	// Verify that chrome components received the width update by checking
	// that View() renders without panicking and produces non-empty output.
	// The tabBar field is nil in model-package tests because the tabbar
	// package imports model (creating a cycle); its propagation is tested
	// via the integration path in the app entry-point.
	bannerView := result.banner.View()
	if bannerView == "" {
		t.Error("banner.View() is empty after WindowSizeMsg; expected propagated width")
	}
	statusView := result.statusLine.View()
	if statusView == "" {
		t.Error("statusLine.View() is empty after WindowSizeMsg; expected propagated width")
	}
}

func TestWindowSizeMsg_PropagatesSizeToChildren(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if mock.width == 0 || mock.height == 0 {
		t.Errorf("claude panel size not propagated: got %dx%d", mock.width, mock.height)
	}
}

// ---------------------------------------------------------------------------
// Update — unknown message
// ---------------------------------------------------------------------------

func TestUpdate_UnknownMsg_ReturnsModelUnchanged(t *testing.T) {
	m := NewAppModel()

	// Deliver an unknown message type; model should be returned unchanged.
	type unknownMsg struct{}
	updated, cmd := m.Update(unknownMsg{})

	result := updated.(AppModel)
	if result.ready {
		t.Error("ready changed on unknown message; want unchanged")
	}
	if cmd != nil {
		t.Error("cmd != nil for unknown message; want nil")
	}
}

// ---------------------------------------------------------------------------
// Update — CLI events with component wiring
// ---------------------------------------------------------------------------

func TestUpdate_AssistantEvent_ForwardsToClaude(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	ev := cli.AssistantEvent{
		Message: cli.AssistantMessage{
			Content: []cli.ContentBlock{
				{Type: "text", Text: "Hello world"},
			},
		},
	}
	m.Update(ev)

	if !mock.handleMsgCalled {
		t.Error("claudePanel.HandleMsg not called for AssistantEvent")
	}
}

func TestUpdate_ResultEvent_UpdatesCostTracker(t *testing.T) {
	m := NewAppModel()
	m.Update(cli.ResultEvent{
		TotalCostUSD: 1.23,
		SessionID:    "s1",
		Subtype:      "success",
	})

	got := m.shared.costTracker.GetSessionCost()
	if got != 1.23 {
		t.Errorf("costTracker.GetSessionCost() = %f; want 1.23", got)
	}
}

func TestUpdate_ToastMsg_ForwardsToToast(t *testing.T) {
	m := NewAppModel()
	mock := &mockToast{empty: true}
	m.shared.toasts = mock

	m.Update(ToastMsg{Text: "hello", Level: "info"})

	if !mock.handleMsgCalled {
		t.Error("toast.HandleMsg not called for ToastMsg")
	}
}

func TestHandleClaudeKey_ForwardsToPanel(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	m.focus = FocusClaude

	// Simulate a key press when Claude panel has focus.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if !mock.handleMsgCalled {
		t.Error("claudePanel.HandleMsg not called for key event with Claude focus")
	}
}

func TestRenderLeftPanel_UsesClaudePanelView(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	mock := &mockClaudePanel{viewOutput: "CLAUDE_PANEL_OUTPUT"}
	m.shared.claudePanel = mock

	output := m.View()
	if !strings.Contains(output, "CLAUDE_PANEL_OUTPUT") {
		t.Error("renderLeftPanel should use claudePanel.View() output")
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_BeforeReady_ReturnsInitializing(t *testing.T) {
	m := NewAppModel()

	view := m.View()
	if view != "Initializing..." {
		t.Errorf("View() before ready = %q; want %q", view, "Initializing...")
	}
}

func TestView_AfterReady_ContainsBannerText(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)

	view := m.View()
	// The banner renders "GOgent-Fortress" — verify it is present.
	if !strings.Contains(view, "GOgent-Fortress") {
		t.Errorf("View() does not contain banner text %q; got %q", "GOgent-Fortress", view)
	}
}

func TestView_AfterReady_ContainsFocusState(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	view := m.View()
	if !strings.Contains(view, "Claude") {
		t.Errorf("View() = %q; want to contain focus state %q", view, "Claude")
	}
}

func TestView_AfterReady_NonEmpty(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)

	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Error("View() returned empty string after ready; want non-empty layout")
	}
}

// ---------------------------------------------------------------------------
// Layout — responsive breakpoints
// ---------------------------------------------------------------------------

func TestComputeLayout_WideTerminal_ShowsRightPanel70_30(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40

	dims := m.computeLayout()

	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 120; want true")
	}

	// At width=120, left outer = 84 (70%), right outer = 36 (30%).
	// Inner widths subtract borderFrame (2).
	wantLeftInner := int(float64(120)*0.70) - borderFrame            // 84 - 2 = 82
	wantRightInner := (120 - int(float64(120)*0.70)) - borderFrame   // 36 - 2 = 34

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

func TestComputeLayout_MediumTerminal_ShowsRightPanel75_25(t *testing.T) {
	m := NewAppModel()
	m.width = 90
	m.height = 30

	dims := m.computeLayout()

	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 90; want true")
	}

	// At width=90, left outer = floor(90*0.75) = 67, right outer = 23.
	width90 := m.width // 90, as a non-constant to allow float conversion
	leftOuter := int(float64(width90) * 0.75)
	wantLeftInner := leftOuter - borderFrame
	wantRightInner := (width90 - leftOuter) - borderFrame

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

func TestComputeLayout_NarrowTerminal_HidesRightPanel(t *testing.T) {
	m := NewAppModel()
	m.width = 79
	m.height = 24

	dims := m.computeLayout()

	if dims.showRightPanel {
		t.Error("showRightPanel = true at width 79; want false (right panel hidden)")
	}

	wantLeftInner := 79 - borderFrame // 77
	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d (full width minus border)", dims.leftWidth, wantLeftInner)
	}
}

func TestComputeLayout_ExactBreakpointAt80_ShowsRightPanel(t *testing.T) {
	m := NewAppModel()
	m.width = 80
	m.height = 24

	dims := m.computeLayout()

	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 80; want true (80 is the inclusive lower bound)")
	}
}

func TestComputeLayout_ExactBreakpointAt100_Uses70_30(t *testing.T) {
	m := NewAppModel()
	m.width = 100
	m.height = 30

	dims := m.computeLayout()

	leftOuter := int(float64(100) * 0.70)
	wantLeftInner := leftOuter - borderFrame
	wantRightInner := (100 - leftOuter) - borderFrame

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth at width=100: got %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth at width=100: got %d; want %d", dims.rightWidth, wantRightInner)
	}
}

func TestComputeLayout_ContentHeight(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40

	dims := m.computeLayout()

	// contentHeight = height - bannerHeight(3) - tabBarHeight(1) - statusLineHeight(2)
	wantHeight := 40 - bannerHeight - tabBarHeight - statusLineHeight // 34
	if dims.contentHeight != wantHeight {
		t.Errorf("contentHeight = %d; want %d", dims.contentHeight, wantHeight)
	}
}

func TestComputeLayout_MinimumContentHeight(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	// Height so small content area would be negative.
	m.height = 1

	dims := m.computeLayout()

	if dims.contentHeight < 1 {
		t.Errorf("contentHeight = %d; want >= 1 (floor at 1)", dims.contentHeight)
	}
}

// ---------------------------------------------------------------------------
// Layout — right panel mode rendering
// ---------------------------------------------------------------------------

func TestView_RightPanel_ShowsAgentsMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	m.rightPanelMode = RPMAgents

	view := m.View()
	// agentTree.View() renders "No agents" when the tree is empty.
	if !strings.Contains(view, "agents") {
		t.Errorf("View() does not contain %q for RPMAgents; got %q", "agents", view)
	}
}

func TestView_RightPanel_ShowsDashboardMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	m.rightPanelMode = RPMDashboard

	view := m.View()
	if !strings.Contains(view, "Dashboard") {
		t.Errorf("View() does not contain %q for RPMDashboard; got %q", "Dashboard", view)
	}
}

func TestView_NarrowTerminal_HidesRightPanel(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 79, Height: 24})
	m = updated.(AppModel)
	// At width=79, right panel is hidden; rightPanelMode content must not appear
	// via the panel itself (it will still be in focus state content for left panel).
	m.rightPanelMode = RPMDashboard

	// Just verify View() renders without panic.
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string for narrow terminal")
	}
}

// ---------------------------------------------------------------------------
// Key handling — global keys
// ---------------------------------------------------------------------------

func TestHandleKey_ForceQuit_ReturnsQuitCmd(t *testing.T) {
	m := NewAppModel()
	// Deliver a WindowSizeMsg first so the model is ready.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("cmd = nil after ctrl+c; want tea.Quit command")
	}

	// Executing the quit command should produce a QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T; want tea.QuitMsg", msg)
	}
	_ = updated
}

func TestHandleKey_ToggleFocus_AdvancesFocus(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	if m.focus != FocusClaude {
		t.Fatalf("precondition: focus = %v; want FocusClaude", m.focus)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(AppModel)

	if result.focus != FocusAgents {
		t.Errorf("focus after tab = %v; want FocusAgents", result.focus)
	}
	if cmd != nil {
		t.Error("cmd != nil after ToggleFocus; want nil")
	}
}

func TestHandleKey_ToggleFocus_WrapsAround(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Tab once → FocusAgents, tab again → back to FocusClaude.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.(AppModel).Update(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(AppModel)

	if result.focus != FocusClaude {
		t.Errorf("focus after two tabs = %v; want FocusClaude", result.focus)
	}
}

func TestHandleKey_CycleRightPanel_AdvancesMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	if m.rightPanelMode != RPMAgents {
		t.Fatalf("precondition: rightPanelMode = %v; want RPMAgents", m.rightPanelMode)
	}

	// alt+r triggers CycleRightPanel.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r"), Alt: true})
	result := updated.(AppModel)

	if result.rightPanelMode != RPMDashboard {
		t.Errorf("rightPanelMode after alt+r = %v; want RPMDashboard", result.rightPanelMode)
	}
	if cmd != nil {
		t.Error("cmd != nil after CycleRightPanel; want nil")
	}
}

// ---------------------------------------------------------------------------
// Key handling — modal state
// ---------------------------------------------------------------------------

func TestHandleKey_Modal_CancelDismissesModal(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Simulate an active modal by injecting a BridgeModalRequestMsg with a
	// Confirm-style option set (Yes / No).
	updated, _ = m.Update(BridgeModalRequestMsg{
		RequestID: "test-cancel",
		Message:   "Continue?",
		Options:   []string{"Yes", "No"},
	})
	m = updated.(AppModel)

	// Verify the modal queue is now active.
	if !m.shared.modalQueue.IsActive() {
		t.Fatal("modal queue not active after BridgeModalRequestMsg")
	}

	// Esc triggers ModalCancel — the modal should emit a ModalResponseMsg
	// (cancelled) which is handled in Update.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(AppModel)

	// The Esc key produces a cmd that returns a ModalResponseMsg.
	// Execute it and feed it back through Update to resolve the modal.
	if cmd != nil {
		msg := cmd()
		updated2, _ := result.Update(msg)
		result = updated2.(AppModel)
	}

	if result.shared.modalQueue.IsActive() {
		t.Error("modal queue still active after ModalCancel; want inactive")
	}
}

func TestHandleKey_Modal_GlobalKeysNotFiredWhenModalActive(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Activate a modal via BridgeModalRequestMsg.
	updated, _ = m.Update(BridgeModalRequestMsg{
		RequestID: "test-global",
		Message:   "Proceed?",
		Options:   []string{"Yes", "No"},
	})
	m = updated.(AppModel)
	initialFocus := m.focus

	if !m.shared.modalQueue.IsActive() {
		t.Fatal("precondition: modal queue not active")
	}

	// Tab would normally toggle focus but should be swallowed by modal.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(AppModel)

	if result.focus != initialFocus {
		t.Errorf("focus changed while modal active; want unchanged %v, got %v",
			initialFocus, result.focus)
	}
	// Modal should still be active (tab is not ModalCancel).
	if !result.shared.modalQueue.IsActive() {
		t.Error("modal queue inactive after non-cancel key; want modal to remain active")
	}
}

// ---------------------------------------------------------------------------
// Message types — compile-time presence checks
//
// These tests do not assert runtime behaviour; they ensure the message types
// exist and can be constructed without panicking.
// ---------------------------------------------------------------------------

func TestMessageTypes_Constructible(t *testing.T) {
	// CLI event messages
	_ = SystemInitMsg{SessionID: "sess-1"}
	_ = StatusUpdateMsg{Status: "Thinking"}
	_ = CompactMsg{Text: "summary"}
	_ = AssistantMsg{Text: "hello", Streaming: true}
	_ = ToolResultMsg{ToolName: "Read", Result: "...", Success: true}
	_ = ResultMsg{SessionID: "sess-1", CostUSD: 0.01, DurationMS: 1234}
	_ = StreamEventMsg{EventType: "text", Data: []byte(`{}`)}
	_ = CLIEventMsg{RawType: "unknown", Data: []byte(`{}`)}

	// UI messages
	_ = ModalRequestMsg{Title: "Confirm", Options: []string{"Yes", "No"}}
	_ = ModalResponseMsg{SelectedIndex: 0, Cancelled: false}
	_ = ToastMsg{Text: "done", Level: "info"}
	_ = TickMsg{Time: time.Now()}

	// Agent messages
	_ = AgentRegisteredMsg{AgentID: "a1", AgentType: "go-pro", ParentID: ""}
	_ = AgentUpdatedMsg{AgentID: "a1", Status: "running"}
	_ = AgentActivityMsg{AgentID: "a1", ToolName: "Bash", Streaming: true}

	// Team messages
	_ = TeamUpdateMsg{TeamDir: "/tmp/team", Status: "running", TaskID: "task-001"}
}

func TestMessageTypes_Count(t *testing.T) {
	// Verify we have at least 15 message types by checking the named set.
	// This test will fail to compile if any type listed in the spec is missing.
	types := []interface{}{
		SystemInitMsg{},
		StatusUpdateMsg{},
		CompactMsg{},
		AssistantMsg{},
		ToolResultMsg{},
		ResultMsg{},
		StreamEventMsg{},
		CLIEventMsg{},
		ModalRequestMsg{},
		ModalResponseMsg{},
		ToastMsg{},
		TickMsg{},
		AgentRegisteredMsg{},
		AgentUpdatedMsg{},
		AgentActivityMsg{},
		TeamUpdateMsg{},
	}

	const minRequired = 15
	if len(types) < minRequired {
		t.Errorf("message type count = %d; want >= %d", len(types), minRequired)
	}
}

// ---------------------------------------------------------------------------
// DiffEntry struct
// ---------------------------------------------------------------------------

func TestDiffEntry_Fields(t *testing.T) {
	entry := DiffEntry{
		FilePath: "/tmp/foo.go",
		Patch:    []byte(`{"hunks":[]}`),
	}

	if entry.FilePath != "/tmp/foo.go" {
		t.Errorf("FilePath = %q; want %q", entry.FilePath, "/tmp/foo.go")
	}
	if string(entry.Patch) != `{"hunks":[]}` {
		t.Errorf("Patch = %s; want %s", entry.Patch, `{"hunks":[]}`)
	}
}

// ---------------------------------------------------------------------------
// Setter methods
// ---------------------------------------------------------------------------

func TestSetTabBar_StoresWidget(t *testing.T) {
	m := NewAppModel()
	if m.tabBar != nil {
		t.Fatal("precondition: tabBar should be nil before SetTabBar")
	}
	// Use a nil tabBarWidget; just verify the assignment doesn't panic.
	m.SetTabBar(nil)
	// nil is a valid tabBarWidget — View guards against it.
}

func TestSetClaudePanel_StoresWidget(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{viewOutput: "panel"}
	m.SetClaudePanel(mock)
	if m.shared.claudePanel == nil {
		t.Error("shared.claudePanel = nil after SetClaudePanel; want non-nil")
	}
}

func TestSetToasts_StoresWidget(t *testing.T) {
	m := NewAppModel()
	mock := &mockToast{empty: true}
	m.SetToasts(mock)
	if m.shared.toasts == nil {
		t.Error("shared.toasts = nil after SetToasts; want non-nil")
	}
}

// ---------------------------------------------------------------------------
// extractDiffs
// ---------------------------------------------------------------------------

func TestExtractDiffs_SingleObjectWithPatch(t *testing.T) {
	m := NewAppModel()
	ev := cli.UserEvent{
		ToolUseResult: []byte(`{"filePath":"/tmp/foo.go","structuredPatch":{"hunks":[]}}`),
	}
	result := m.extractDiffs(ev)
	if len(result.diffs) != 1 {
		t.Fatalf("diffs count = %d; want 1", len(result.diffs))
	}
	if result.diffs[0].FilePath != "/tmp/foo.go" {
		t.Errorf("FilePath = %q; want /tmp/foo.go", result.diffs[0].FilePath)
	}
}

func TestExtractDiffs_ArrayWithPatch(t *testing.T) {
	m := NewAppModel()
	ev := cli.UserEvent{
		ToolUseResult: []byte(`[{"filePath":"/a.go","structuredPatch":{}},{"filePath":"/b.go","structuredPatch":{}}]`),
	}
	result := m.extractDiffs(ev)
	if len(result.diffs) != 2 {
		t.Fatalf("diffs count = %d; want 2", len(result.diffs))
	}
}

func TestExtractDiffs_EmptyToolUseResult_NoChange(t *testing.T) {
	m := NewAppModel()
	ev := cli.UserEvent{}
	result := m.extractDiffs(ev)
	if len(result.diffs) != 0 {
		t.Errorf("diffs count = %d; want 0 for empty ToolUseResult", len(result.diffs))
	}
}

func TestExtractDiffs_NoPatch_NoEntry(t *testing.T) {
	m := NewAppModel()
	ev := cli.UserEvent{
		ToolUseResult: []byte(`{"filePath":"/tmp/foo.go"}`),
	}
	result := m.extractDiffs(ev)
	if len(result.diffs) != 0 {
		t.Errorf("diffs count = %d; want 0 when structuredPatch is absent", len(result.diffs))
	}
}

// ---------------------------------------------------------------------------
// Agent lifecycle messages
// ---------------------------------------------------------------------------

func TestUpdate_AgentRegisteredMsg_RefreshesTree(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true

	// Deliver an AgentRegisteredMsg — should not panic and should return nil cmd.
	updated, cmd := m.Update(AgentRegisteredMsg{AgentID: "a1", AgentType: "go-pro"})
	_ = updated.(AppModel)
	if cmd != nil {
		t.Error("cmd != nil for AgentRegisteredMsg; want nil")
	}
}

func TestUpdate_AgentUpdatedMsg_RefreshesTree(t *testing.T) {
	m := NewAppModel()

	updated, cmd := m.Update(AgentUpdatedMsg{AgentID: "a1", Status: "running"})
	_ = updated.(AppModel)
	if cmd != nil {
		t.Error("cmd != nil for AgentUpdatedMsg; want nil")
	}
}

func TestUpdate_AgentActivityMsg_RefreshesTree(t *testing.T) {
	m := NewAppModel()

	updated, cmd := m.Update(AgentActivityMsg{AgentID: "a1", ToolName: "Bash"})
	_ = updated.(AppModel)
	if cmd != nil {
		t.Error("cmd != nil for AgentActivityMsg; want nil")
	}
}

// ---------------------------------------------------------------------------
// handleAgentsKey
// ---------------------------------------------------------------------------

func TestHandleAgentsKey_NavigationDown(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = FocusAgents

	// With an empty tree, navigation should not panic.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_ = updated.(AppModel)
}

func TestHandleAgentsKey_NavigationUp(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = FocusAgents

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	_ = updated.(AppModel)
}

// ---------------------------------------------------------------------------
// syncFocusState
// ---------------------------------------------------------------------------

func TestSyncFocusState_PropagatesFocusToClaudePanel(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	// Claude has focus by default; syncFocusState should set focused=true.
	m.syncFocusState()
	if !mock.focused {
		t.Error("claudePanel.focused = false after syncFocusState with FocusClaude; want true")
	}

	// Switch focus to agents; claude panel should become unfocused.
	m.focus = FocusAgents
	m.syncFocusState()
	if mock.focused {
		t.Error("claudePanel.focused = true after syncFocusState with FocusAgents; want false")
	}
}

func TestSyncFocusState_NilClaudePanel_NoPanic(t *testing.T) {
	m := NewAppModel()
	// No claude panel wired — should not panic.
	m.syncFocusState()
}

// ---------------------------------------------------------------------------
// Toast forwarding for tick messages
// ---------------------------------------------------------------------------

func TestUpdate_UnknownMsg_ForwardsToToast(t *testing.T) {
	m := NewAppModel()
	mock := &mockToast{empty: true}
	m.shared.toasts = mock

	// An unknown message type should be forwarded to toast HandleMsg.
	type someTickMsg struct{}
	m.Update(someTickMsg{})

	if !mock.handleMsgCalled {
		t.Error("toast.HandleMsg not called for unknown message; want forwarding for tick-based expiry")
	}
}

// ---------------------------------------------------------------------------
// renderLayout — toast path
// ---------------------------------------------------------------------------

func TestRenderLayout_ToastVisible_WhenNotEmpty(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	mock := &mockToast{empty: false, viewOutput: "TOAST_OUTPUT"}
	m.shared.toasts = mock

	view := m.View()
	if !strings.Contains(view, "TOAST_OUTPUT") {
		t.Error("renderLayout should include toast view when toasts are not empty")
	}
}

func TestRenderLayout_ToastHidden_WhenEmpty(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	mock := &mockToast{empty: true, viewOutput: "TOAST_OUTPUT"}
	m.shared.toasts = mock

	view := m.View()
	if strings.Contains(view, "TOAST_OUTPUT") {
		t.Error("renderLayout should NOT include toast view when toasts are empty")
	}
}

// ---------------------------------------------------------------------------
// AssistantEvent — no-op when no text blocks
// ---------------------------------------------------------------------------

func TestUpdate_AssistantEvent_NoTextBlocks_NoForwarding(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	// Tool-use block only; no text content.
	ev := cli.AssistantEvent{
		Message: cli.AssistantMessage{
			Content: []cli.ContentBlock{
				{Type: "tool_use", Text: ""},
			},
		},
	}
	m.Update(ev)

	if mock.handleMsgCalled {
		t.Error("claudePanel.HandleMsg called for tool_use-only AssistantEvent; want no forwarding")
	}
}

// ---------------------------------------------------------------------------
// ResultEvent — forwards to claude panel
// ---------------------------------------------------------------------------

func TestUpdate_ResultEvent_ForwardsToClaude(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	m.Update(cli.ResultEvent{
		TotalCostUSD: 0.05,
		SessionID:    "s1",
		Subtype:      "success",
	})

	if !mock.handleMsgCalled {
		t.Error("claudePanel.HandleMsg not called for ResultEvent; want forwarding")
	}
}
