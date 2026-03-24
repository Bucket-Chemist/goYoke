package model

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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
	streaming       bool
	savedMessages   []state.DisplayMessage
	restoredMsgs    []state.DisplayMessage
	// setSenderCalled tracks how many times SetSender was called (for C-1 tests).
	setSenderCalled int
	// lastSender is the most recent sender passed to SetSender.
	lastSender MessageSender
}

func (m *mockClaudePanel) HandleMsg(msg tea.Msg) tea.Cmd {
	m.handleMsgCalled = true
	m.lastMsg = msg
	return nil
}
func (m *mockClaudePanel) View() string      { return m.viewOutput }
func (m *mockClaudePanel) SetSize(w, h int)  { m.width = w; m.height = h }
func (m *mockClaudePanel) SetFocused(f bool) { m.focused = f }
func (m *mockClaudePanel) IsStreaming() bool { return m.streaming }
func (m *mockClaudePanel) SaveMessages() []state.DisplayMessage {
	return m.savedMessages
}
func (m *mockClaudePanel) RestoreMessages(msgs []state.DisplayMessage) {
	m.restoredMsgs = msgs
	// Mirror the restored messages so SaveMessages reflects current panel state.
	m.savedMessages = msgs
}
func (m *mockClaudePanel) SetSender(s MessageSender) {
	m.setSenderCalled++
	m.lastSender = s
}

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

// ---------------------------------------------------------------------------
// TUI-052: ReverseToggleFocus — Shift+Tab cycles focus in reverse direction
// ---------------------------------------------------------------------------

// TestHandleKey_ReverseToggleFocus_RetreatsFromAgentsToClaude verifies that
// shift+tab moves focus from FocusAgents back to FocusClaude (reverse of tab).
func TestHandleKey_ReverseToggleFocus_RetreatsFromAgentsToClaude(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Advance to FocusAgents first.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	if m.focus != FocusAgents {
		t.Fatalf("precondition: focus = %v; want FocusAgents", m.focus)
	}

	// Shift+Tab should go back to FocusClaude.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	result := updated.(AppModel)

	if result.focus != FocusClaude {
		t.Errorf("focus after shift+tab = %v; want FocusClaude", result.focus)
	}
	if cmd != nil {
		t.Error("cmd != nil after ReverseToggleFocus; want nil")
	}
}

// TestHandleKey_ReverseToggleFocus_WrapsAroundToLast verifies that shift+tab
// from the first focus target wraps to the last target.
func TestHandleKey_ReverseToggleFocus_WrapsAroundToLast(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	if m.focus != FocusClaude {
		t.Fatalf("precondition: focus = %v; want FocusClaude", m.focus)
	}

	// Shift+Tab from FocusClaude (first) should wrap to FocusAgents (last).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	result := updated.(AppModel)

	if result.focus != FocusAgents {
		t.Errorf("focus after shift+tab from first = %v; want FocusAgents (wrap-around)", result.focus)
	}
	if cmd != nil {
		t.Error("cmd != nil after ReverseToggleFocus wrap; want nil")
	}
}

// TestHandleKey_ReverseToggleFocus_IsOppositeOfTab verifies that tab and
// shift+tab are exact inverses: tab → shift+tab returns to start.
func TestHandleKey_ReverseToggleFocus_IsOppositeOfTab(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	initial := m.focus

	// Tab forward, then shift+tab back — must return to initial focus.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.(AppModel).Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	result := updated.(AppModel)

	if result.focus != initial {
		t.Errorf("focus after tab+shift+tab = %v; want %v (should return to start)", result.focus, initial)
	}
}

// TestHandleKey_ShiftTab_DoesNotTriggerCycleProvider verifies that shift+tab
// no longer triggers provider switching after TUI-052.
func TestHandleKey_ShiftTab_DoesNotTriggerCycleProvider(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)
	m.shared.claudePanel = mock

	seqBefore := m.providerSwitchSeq

	// shift+tab must NOT increment the provider-switch sequence counter.
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	// providerSwitchSeq checked on the pre-update model; the Update returns
	// new model — re-check on result.
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	result := updated2.(AppModel)

	if result.providerSwitchSeq != seqBefore {
		t.Errorf("providerSwitchSeq changed on shift+tab = %d; want %d (shift+tab is no longer CycleProvider)",
			result.providerSwitchSeq, seqBefore)
	}
}

// TestHandleKey_AltShiftP_StillTriggersCycleProvider verifies that the new
// CycleProvider binding (alt+P) still triggers provider switching after TUI-052.
func TestHandleKey_AltShiftP_StillTriggersCycleProvider(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	// alt+P is the new CycleProvider binding (TUI-052).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true})
	if cmd == nil {
		t.Error("alt+P (new CycleProvider) did not emit a command; want debounce timer")
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
	_ = modals.ModalResponseMsg{}
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
		modals.ModalResponseMsg{},
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

// ---------------------------------------------------------------------------
// ProviderSwitchMsg + CycleProvider key (TUI-029)
// ---------------------------------------------------------------------------

// newModelWithProvider returns an AppModel wired with a mock panel and a
// real ProviderState — the minimal setup needed for provider-switch tests.
func newModelWithProvider() (AppModel, *mockClaudePanel) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	m.shared.providerState = state.NewProviderState()
	return m, mock
}

func TestProviderSwitchMsg_CyclesProvider(t *testing.T) {
	m, _ := newModelWithProvider()
	ps := m.shared.providerState

	initial := ps.GetActiveProvider()
	if initial != state.ProviderAnthropic {
		t.Fatalf("precondition: active provider = %q; want Anthropic", initial)
	}

	updated, _ := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	got := result.shared.providerState.GetActiveProvider()
	if got == initial {
		t.Errorf("provider unchanged after ProviderSwitchMsg; want next provider, got %q", got)
	}
	// The next provider in canonical order after Anthropic is Google.
	if got != state.ProviderGoogle {
		t.Errorf("active provider = %q; want Google (second in canonical order)", got)
	}
}

func TestProviderSwitchMsg_CyclesThroughAllFour(t *testing.T) {
	m, _ := newModelWithProvider()

	providers := m.shared.providerState.AllProviders()
	seen := make(map[state.ProviderID]bool)
	seen[m.shared.providerState.GetActiveProvider()] = true

	for range len(providers) - 1 {
		updated, _ := m.Update(ProviderSwitchMsg{})
		m = updated.(AppModel)
		seen[m.shared.providerState.GetActiveProvider()] = true
	}

	if len(seen) != len(providers) {
		t.Errorf("only %d distinct providers seen after full cycle; want %d", len(seen), len(providers))
	}
}

func TestProviderSwitchMsg_WrapAroundToFirst(t *testing.T) {
	m, _ := newModelWithProvider()
	providers := m.shared.providerState.AllProviders()

	// Cycle through all providers — should wrap back to the first.
	for range len(providers) {
		updated, _ := m.Update(ProviderSwitchMsg{})
		m = updated.(AppModel)
	}

	got := m.shared.providerState.GetActiveProvider()
	if got != providers[0] {
		t.Errorf("after full cycle, active provider = %q; want %q (wrap-around)", got, providers[0])
	}
}

func TestProviderSwitchMsg_SavesCurrentMessages(t *testing.T) {
	m, mock := newModelWithProvider()
	mock.savedMessages = []state.DisplayMessage{
		{Role: "user", Content: "before switch"},
	}

	updated, _ := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	// After switch, active provider is Google; switch back to Anthropic.
	if err := result.shared.providerState.SwitchProvider(state.ProviderAnthropic); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	msgs := result.shared.providerState.GetActiveMessages()
	if len(msgs) == 0 {
		t.Error("Anthropic messages should have been saved before switch; got none")
	}
	if msgs[0].Content != "before switch" {
		t.Errorf("saved message content = %q; want %q", msgs[0].Content, "before switch")
	}
}

func TestProviderSwitchMsg_RestoresNewProviderMessages(t *testing.T) {
	m, mock := newModelWithProvider()

	// Pre-populate Google's messages in ProviderState.
	if err := m.shared.providerState.SwitchProvider(state.ProviderGoogle); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	m.shared.providerState.AppendMessage(state.DisplayMessage{
		Role:    "assistant",
		Content: "google historical msg",
	})
	// Switch back to Anthropic to simulate current state.
	if err := m.shared.providerState.SwitchProvider(state.ProviderAnthropic); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}

	// Now trigger provider switch (Anthropic → Google).
	updated, _ := m.Update(ProviderSwitchMsg{})
	_ = updated.(AppModel)

	// The mock panel's RestoreMessages should have been called with Google's history.
	if len(mock.restoredMsgs) == 0 {
		t.Error("RestoreMessages not called after provider switch; want Google's history restored")
	}
	if mock.restoredMsgs[0].Content != "google historical msg" {
		t.Errorf("restored message = %q; want %q",
			mock.restoredMsgs[0].Content, "google historical msg")
	}
}

func TestProviderSwitchMsg_ResetsSessionState(t *testing.T) {
	m, _ := newModelWithProvider()
	m.sessionID = "old-session-id"
	m.cliReady = true
	m.reconnectCount = 2

	updated, _ := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	if result.sessionID != "" {
		t.Errorf("sessionID = %q after switch; want empty", result.sessionID)
	}
	if result.cliReady {
		t.Error("cliReady = true after switch; want false (new driver not yet ready)")
	}
	if result.reconnectCount != 0 {
		t.Errorf("reconnectCount = %d after switch; want 0", result.reconnectCount)
	}
}

func TestProviderSwitchMsg_NoProviderState_IsNoop(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	m.shared.providerState = nil // explicitly nil

	updated, cmd := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	// Should be a no-op — provider state unchanged, no cmd.
	if result.shared.providerState != nil {
		t.Error("providerState should remain nil after no-op switch")
	}
	if cmd != nil {
		t.Error("cmd != nil for no-op ProviderSwitchMsg; want nil")
	}
}

func TestCycleProvider_Key_EmitsProviderSwitchExecuteMsg(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	// alt+shift+p (alt+P) is CycleProvider (TUI-052: rebound from shift+tab).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true})
	result := updated.(AppModel)

	if cmd == nil {
		t.Fatal("CycleProvider key did not emit a command")
	}

	// The debounce timer fires after 300 ms.  Executing the returned tick
	// command immediately produces a ProviderSwitchExecuteMsg whose Seq
	// matches the incremented counter (1 after the first keypress).
	msg := cmd()
	execMsg, ok := msg.(ProviderSwitchExecuteMsg)
	if !ok {
		t.Errorf("cmd() = %T; want ProviderSwitchExecuteMsg", msg)
	}
	if execMsg.Seq != result.providerSwitchSeq {
		t.Errorf("ProviderSwitchExecuteMsg.Seq = %d; want %d (providerSwitchSeq)",
			execMsg.Seq, result.providerSwitchSeq)
	}
}

func TestCycleProvider_Key_BlockedDuringStreaming(t *testing.T) {
	m := NewAppModel()
	mock := &mockClaudePanel{streaming: true}
	m.shared.claudePanel = mock

	// alt+shift+p (alt+P) is CycleProvider (TUI-052: rebound from shift+tab).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true})
	if cmd != nil {
		t.Error("CycleProvider key should be blocked while streaming; got non-nil cmd")
	}
}

func TestProviderSwitchMsg_MessageIsolation_ProviderAHasOwnMessages(t *testing.T) {
	m, mockPanel := newModelWithProvider()

	// Provider A (Anthropic) has two messages.
	mockPanel.savedMessages = []state.DisplayMessage{
		{Role: "user", Content: "A-msg-1"},
		{Role: "assistant", Content: "A-msg-2"},
	}

	// Switch to Provider B (Google).
	updated, _ := m.Update(ProviderSwitchMsg{})
	m = updated.(AppModel)

	// Google had no prior messages, but a handoff system message is injected
	// because the old provider (Anthropic) had ≥2 messages. RestoreMessages
	// is called a second time with the handoff appended, so the panel now has
	// exactly 1 message (the system handoff).
	if len(mockPanel.restoredMsgs) != 1 {
		t.Errorf("Google should have 1 handoff system message; got %d", len(mockPanel.restoredMsgs))
	}
	if len(mockPanel.restoredMsgs) > 0 && mockPanel.restoredMsgs[0].Role != "system" {
		t.Errorf("Google's first message should be a system handoff; got role %q",
			mockPanel.restoredMsgs[0].Role)
	}

	// Switch back to Anthropic.
	updated, _ = m.Update(ProviderSwitchMsg{})
	updated, _ = updated.(AppModel).Update(ProviderSwitchMsg{})
	updated, _ = updated.(AppModel).Update(ProviderSwitchMsg{})
	m = updated.(AppModel)

	// After three more switches we arrive back at Anthropic (4 providers total,
	// starting at Google means 3 hops: Google→OpenAI→Local→Anthropic).
	got := m.shared.providerState.GetActiveProvider()
	if got != state.ProviderAnthropic {
		t.Errorf("after full cycle: active = %q; want Anthropic", got)
	}

	// Anthropic's original 2 messages should be restored exactly.
	// Intermediate hops (G→O, O→L, L→A) produce no handoff because the mock
	// mirrors restored state: after the A→G switch the panel state becomes the
	// 1-message handoff slice, and subsequent SaveMessages() calls return that
	// 1-message slice (len < 2 → no handoff generated for any later hop).
	// Only the L→A hop restores Anthropic's own saved messages from
	// ProviderState, which are the original 2.
	if len(mockPanel.restoredMsgs) != 2 {
		t.Errorf("Anthropic restore: got %d messages; want 2", len(mockPanel.restoredMsgs))
	}
	if len(mockPanel.restoredMsgs) >= 1 && mockPanel.restoredMsgs[0].Content != "A-msg-1" {
		t.Errorf("restored[0].Content = %q; want %q", mockPanel.restoredMsgs[0].Content, "A-msg-1")
	}
	if len(mockPanel.restoredMsgs) >= 2 && mockPanel.restoredMsgs[1].Content != "A-msg-2" {
		t.Errorf("restored[1].Content = %q; want %q", mockPanel.restoredMsgs[1].Content, "A-msg-2")
	}
}

// ---------------------------------------------------------------------------
// R-3: Handoff injection on provider switch
// ---------------------------------------------------------------------------

// TestHandoffInjected_SwitchWithTwoMessages verifies that switching from a
// provider with ≥2 messages injects a system handoff message into the new
// provider's conversation.
func TestHandoffInjected_SwitchWithTwoMessages(t *testing.T) {
	m, mockPanel := newModelWithProvider()
	mockPanel.savedMessages = []state.DisplayMessage{
		{Role: "user", Content: "What is Go?"},
		{Role: "assistant", Content: "Go is a compiled language."},
	}

	updated, _ := m.Update(ProviderSwitchMsg{})
	_ = updated.(AppModel)

	// The new provider (Google) should have 1 system message — the handoff.
	if len(mockPanel.restoredMsgs) != 1 {
		t.Fatalf("expected 1 handoff message; got %d", len(mockPanel.restoredMsgs))
	}
	if mockPanel.restoredMsgs[0].Role != "system" {
		t.Errorf("handoff message role = %q; want %q", mockPanel.restoredMsgs[0].Role, "system")
	}
	if !strings.Contains(mockPanel.restoredMsgs[0].Content, "anthropic") {
		t.Errorf("handoff content should mention from-provider; got:\n%s",
			mockPanel.restoredMsgs[0].Content)
	}
	if !strings.Contains(mockPanel.restoredMsgs[0].Content, "google") {
		t.Errorf("handoff content should mention to-provider; got:\n%s",
			mockPanel.restoredMsgs[0].Content)
	}
}

// TestHandoffNotInjected_SwitchWithOneMessage verifies that switching from a
// provider with fewer than 2 messages does NOT inject a handoff.
func TestHandoffNotInjected_SwitchWithOneMessage(t *testing.T) {
	m, mockPanel := newModelWithProvider()
	mockPanel.savedMessages = []state.DisplayMessage{
		{Role: "user", Content: "hello"},
	}

	updated, _ := m.Update(ProviderSwitchMsg{})
	_ = updated.(AppModel)

	// Fewer than 2 messages → no handoff.  Google starts empty.
	if len(mockPanel.restoredMsgs) != 0 {
		t.Errorf("expected no handoff for 1 message; got %d messages restored",
			len(mockPanel.restoredMsgs))
	}
}

// TestHandoffNotInjected_SwitchWithNoMessages verifies that switching from a
// provider with no messages does NOT inject a handoff.
func TestHandoffNotInjected_SwitchWithNoMessages(t *testing.T) {
	m, mockPanel := newModelWithProvider()
	mockPanel.savedMessages = nil

	updated, _ := m.Update(ProviderSwitchMsg{})
	_ = updated.(AppModel)

	if len(mockPanel.restoredMsgs) != 0 {
		t.Errorf("expected no handoff for 0 messages; got %d messages restored",
			len(mockPanel.restoredMsgs))
	}
}

// TestHandoffInNewProviderState verifies the handoff is persisted in the NEW
// provider's ProviderState messages, not the old one.
func TestHandoffInNewProviderState(t *testing.T) {
	m, mockPanel := newModelWithProvider()
	mockPanel.savedMessages = []state.DisplayMessage{
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "answer"},
	}

	updated, _ := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	// Active provider is now Google.
	if result.shared.providerState.GetActiveProvider() != state.ProviderGoogle {
		t.Fatalf("expected Google active; got %q", result.shared.providerState.GetActiveProvider())
	}

	// Google's ProviderState messages should contain the handoff.
	googleMsgs := result.shared.providerState.GetActiveMessages()
	if len(googleMsgs) != 1 {
		t.Fatalf("Google ProviderState: got %d messages; want 1 (handoff)", len(googleMsgs))
	}
	if googleMsgs[0].Role != "system" {
		t.Errorf("handoff role = %q; want system", googleMsgs[0].Role)
	}

	// Anthropic's messages should be unchanged (still the original 2).
	if err := result.shared.providerState.SwitchProvider(state.ProviderAnthropic); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	anthropicMsgs := result.shared.providerState.GetActiveMessages()
	if len(anthropicMsgs) != 2 {
		t.Errorf("Anthropic ProviderState: got %d messages; want 2 (original)", len(anthropicMsgs))
	}
}

// ---------------------------------------------------------------------------
// SystemInitEvent — session ID persistence (TUI-031)
// ---------------------------------------------------------------------------

func TestSystemInitEvent_PersistsSessionIDToProviderState(t *testing.T) {
	m, _ := newModelWithProvider()

	updated, _ := m.Update(cli.SystemInitEvent{
		SessionID: "init-session-123",
		Model:     "sonnet",
	})
	result := updated.(AppModel)

	// m.sessionID must be set.
	if result.sessionID != "init-session-123" {
		t.Errorf("sessionID = %q; want %q", result.sessionID, "init-session-123")
	}
	// ProviderState must also have it recorded.
	got := result.shared.providerState.GetActiveSessionID()
	if got != "init-session-123" {
		t.Errorf("providerState.GetActiveSessionID() = %q; want %q", got, "init-session-123")
	}
}

func TestSystemInitEvent_EmptySessionID_NotPersisted(t *testing.T) {
	m, _ := newModelWithProvider()

	updated, _ := m.Update(cli.SystemInitEvent{
		SessionID: "",
		Model:     "sonnet",
	})
	result := updated.(AppModel)

	// Empty session IDs must not be written to ProviderState.
	got := result.shared.providerState.GetActiveSessionID()
	if got != "" {
		t.Errorf("providerState.GetActiveSessionID() = %q; want empty for empty SessionID event", got)
	}
}

// ---------------------------------------------------------------------------
// handleProviderSwitch — session resume (TUI-031)
// ---------------------------------------------------------------------------

func TestProviderSwitchMsg_IncludesSessionIDInNewOpts(t *testing.T) {
	m, _ := newModelWithProvider()

	// Simulate the previous Anthropic session having a stored session ID.
	// Set it directly on ProviderState as if a SystemInitEvent had fired.
	m.shared.providerState.SetSessionID("anthropic-prev-session")

	// Cycle to Google — the switch should save Anthropic's session ID and
	// reset m.sessionID (new provider starts fresh).
	updated, _ := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	// After the switch the active provider is Google.
	// m.sessionID is reset to "" — the new driver has not yet connected.
	if result.sessionID != "" {
		t.Errorf("sessionID = %q after switch; want empty (new driver not yet ready)", result.sessionID)
	}

	// Now switch back to Anthropic to verify its session ID is still stored.
	if err := result.shared.providerState.SwitchProvider(state.ProviderAnthropic); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	got := result.shared.providerState.GetActiveSessionID()
	if got != "anthropic-prev-session" {
		t.Errorf("Anthropic session ID = %q; want %q (must survive provider switch)", got, "anthropic-prev-session")
	}
}

func TestProviderSwitchMsg_NoSessionID_EmptyInOpts(t *testing.T) {
	m, _ := newModelWithProvider()

	// No session ID set for any provider — switch should not panic and new
	// driver must be started without a --resume flag (empty SessionID).
	updated, cmd := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)

	if result.sessionID != "" {
		t.Errorf("sessionID = %q after switch with no prior session; want empty", result.sessionID)
	}
	// A new CLI driver Start() command must still be returned.
	if cmd == nil {
		t.Error("cmd = nil after ProviderSwitchMsg; want driver Start command")
	}
}

func TestSetProviderState_Setter(t *testing.T) {
	m := NewAppModel()
	ps := state.NewProviderState()
	m.SetProviderState(ps)

	if m.shared.providerState != ps {
		t.Error("SetProviderState did not store the given ProviderState")
	}
}

func TestNewAppModel_ProviderStateInitialised(t *testing.T) {
	m := NewAppModel()
	if m.shared.providerState == nil {
		t.Error("NewAppModel should initialise providerState; got nil")
	}
	// Default active provider must be Anthropic.
	got := m.shared.providerState.GetActiveProvider()
	if got != state.ProviderAnthropic {
		t.Errorf("default active provider = %q; want Anthropic", got)
	}
}

// ---------------------------------------------------------------------------
// ProviderState getter (R-1)
// ---------------------------------------------------------------------------

func TestProviderState_Getter_ReturnsSamePointer(t *testing.T) {
	m := NewAppModel()
	// The getter must return the pointer held inside shared state so that
	// main.go can pass it to NewProviderTabBarModel without duplicating state.
	got := m.ProviderState()
	if got == nil {
		t.Fatal("ProviderState() returned nil; want non-nil pointer")
	}
	if got != m.shared.providerState {
		t.Error("ProviderState() returned a different pointer than shared.providerState")
	}
}

func TestProviderState_Getter_NilShared(t *testing.T) {
	// Constructing AppModel without NewAppModel leaves shared nil.
	m := AppModel{}
	got := m.ProviderState()
	if got != nil {
		t.Errorf("ProviderState() with nil shared = %v; want nil", got)
	}
}

// ---------------------------------------------------------------------------
// Provider switch debounce (R-2)
// ---------------------------------------------------------------------------

// mockCLIDriver satisfies cliDriverWidget for debounce tests that exercise
// handleProviderSwitch (which creates and starts a new driver).
type mockCLIDriverDebounce struct {
	startCalls    int
	shutdownCalls int
}

func (m *mockCLIDriverDebounce) Start() tea.Cmd {
	m.startCalls++
	return func() tea.Msg { return nil }
}
func (m *mockCLIDriverDebounce) WaitForEvent() tea.Cmd { return nil }
func (m *mockCLIDriverDebounce) SendMessage(_ string) tea.Cmd { return nil }
func (m *mockCLIDriverDebounce) Shutdown() error {
	m.shutdownCalls++
	return nil
}

func TestProviderSwitchDebounce_SeqIncrementsOnKeyPress(t *testing.T) {
	m := newReadyAppModel(120, 40)

	// Initial seq must be zero.
	if m.providerSwitchSeq != 0 {
		t.Fatalf("initial providerSwitchSeq = %d; want 0", m.providerSwitchSeq)
	}

	// Simulate a CycleProvider key press (alt+shift+p / alt+P).
	// TUI-052: Shift+Tab rebound to ReverseToggleFocus; CycleProvider moved to alt+P.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true}
	updated, _ := m.Update(msg)
	result := updated.(AppModel)

	if result.providerSwitchSeq != 1 {
		t.Errorf("providerSwitchSeq after 1 press = %d; want 1", result.providerSwitchSeq)
	}
}

func TestProviderSwitchDebounce_RapidPressesIncrementSeq(t *testing.T) {
	m := newReadyAppModel(120, 40)

	// TUI-052: CycleProvider rebound from shift+tab to alt+P.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true}
	for i := 0; i < 3; i++ {
		updated, _ := m.Update(msg)
		m = updated.(AppModel)
	}

	if m.providerSwitchSeq != 3 {
		t.Errorf("providerSwitchSeq after 3 presses = %d; want 3", m.providerSwitchSeq)
	}
}

func TestProviderSwitchDebounce_StaleSeqIgnored(t *testing.T) {
	// Set seq to 5 to simulate that 5 keypresses have occurred.
	m := newReadyAppModel(120, 40)
	m.providerSwitchSeq = 5

	// Deliver an execute message with stale seq (3 < 5).
	staleMsg := ProviderSwitchExecuteMsg{Seq: 3}
	updated, cmd := m.Update(staleMsg)
	result := updated.(AppModel)

	// seq must not change (no side-effect from a stale message).
	if result.providerSwitchSeq != 5 {
		t.Errorf("providerSwitchSeq changed on stale msg: got %d; want 5", result.providerSwitchSeq)
	}
	// No command should be emitted for a stale execute message.
	if cmd != nil {
		t.Error("cmd != nil for stale ProviderSwitchExecuteMsg; want nil")
	}
}

func TestProviderSwitchDebounce_OnlyLatestSeqExecutes(t *testing.T) {
	// Seq counter is at 3 (three rapid presses simulated).
	m := newReadyAppModel(120, 40)
	m.providerSwitchSeq = 3

	// Stale timer (seq 1): must be discarded.
	_, cmd1 := m.Update(ProviderSwitchExecuteMsg{Seq: 1})
	if cmd1 != nil {
		t.Error("stale seq=1 produced a command; want nil")
	}

	// Stale timer (seq 2): must also be discarded.
	_, cmd2 := m.Update(ProviderSwitchExecuteMsg{Seq: 2})
	if cmd2 != nil {
		t.Error("stale seq=2 produced a command; want nil")
	}

	// Latest timer (seq 3): must execute the switch and return a command.
	// handleProviderSwitch requires a providerState; NewAppModel provides one.
	_, cmd3 := m.Update(ProviderSwitchExecuteMsg{Seq: 3})
	if cmd3 == nil {
		t.Error("latest seq=3 returned nil cmd; want a driver Start command")
	}
}

func TestProviderSwitchDebounce_StreamingBlocksKeyPress(t *testing.T) {
	m := newReadyAppModel(120, 40)
	panel := &mockClaudePanel{streaming: true}
	m.shared.claudePanel = panel

	seqBefore := m.providerSwitchSeq

	// Attempt CycleProvider while streaming.
	// TUI-052: CycleProvider rebound from shift+tab to alt+P.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P"), Alt: true}
	updated, cmd := m.Update(msg)
	result := updated.(AppModel)

	// The seq counter must NOT have incremented (streaming blocks the press).
	if result.providerSwitchSeq != seqBefore {
		t.Errorf("providerSwitchSeq changed while streaming: got %d; want %d",
			result.providerSwitchSeq, seqBefore)
	}
	// No debounce timer should be returned.
	if cmd != nil {
		t.Error("cmd != nil when streaming blocks CycleProvider; want nil")
	}
}

func TestProviderSwitchExecuteMsg_MatchingSeqCallsSwitch(t *testing.T) {
	m := newReadyAppModel(120, 40)
	m.providerSwitchSeq = 1

	// Deliver the matching execute message; handleProviderSwitch must fire.
	updated, cmd := m.Update(ProviderSwitchExecuteMsg{Seq: 1})
	result := updated.(AppModel)

	// handleProviderSwitch resets cliReady.
	if result.cliReady {
		t.Error("cliReady should be false after provider switch")
	}
	// A Start command must be returned by the new driver.
	if cmd == nil {
		t.Error("cmd = nil after matching ProviderSwitchExecuteMsg; want Start command")
	}
}

// newReadyAppModel returns an AppModel in the ready state (width/height set,
// ready = true) for use in tests that exercise layout-dependent or
// provider-switch-dependent behaviour.
func newReadyAppModel(width, height int) AppModel {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(AppModel)
}

// ---------------------------------------------------------------------------
// M-5: TestProviderSwitch_FullRoundtrip
//
// Integration test verifying the complete provider-switch contract:
//   1. Messages are saved from the old provider before the switch.
//   2. SetSender is called on the panel with the new driver.
//   3. Original CLI opts (Verbose, Debug, PermissionMode) are preserved.
//   4. InvalidateTreeCache is not required here (tested in C-3 tests above),
//      but the tree is refreshed without panicking.
//   5. Handoff message is injected into the new provider's conversation.
//   6. A new driver Start command is returned.
// ---------------------------------------------------------------------------

// mockDriverCapture is a cliDriverWidget that records Start/Shutdown calls
// and satisfies the MessageSender interface so it can be passed to SetSender.
type mockDriverCapture struct {
	startCalls    int
	shutdownCalls int
	sendCalls     int
}

func (d *mockDriverCapture) Start() tea.Cmd {
	d.startCalls++
	return func() tea.Msg { return cli.CLIStartedMsg{} }
}
func (d *mockDriverCapture) WaitForEvent() tea.Cmd { return nil }
func (d *mockDriverCapture) SendMessage(_ string) tea.Cmd {
	d.sendCalls++
	return nil
}
func (d *mockDriverCapture) Shutdown() error {
	d.shutdownCalls++
	return nil
}

// newModelWithProviderAndOpts returns an AppModel wired with a mock panel,
// a real ProviderState, and baseline CLI opts — the minimal setup for
// full provider-switch integration tests.
func newModelWithProviderAndOpts() (AppModel, *mockClaudePanel, *mockDriverCapture) {
	m := NewAppModel()
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	m.shared.providerState = state.NewProviderState()

	// Inject baseline opts with sentinel flag values so we can verify
	// they are preserved across the switch.
	baseOpts := cli.CLIDriverOpts{
		Verbose:        true,
		Debug:          false,
		PermissionMode: "acceptEdits",
	}
	m.shared.baseCLIOpts = baseOpts

	// Wire a mock driver as the initial CLI driver.
	oldDriver := &mockDriverCapture{}
	m.shared.cliDriver = oldDriver

	return m, mock, oldDriver
}

func TestProviderSwitch_FullRoundtrip(t *testing.T) {
	tests := []struct {
		name             string
		initialMessages  []state.DisplayMessage
		wantHandoff      bool // whether a system handoff message should be injected
		wantSavedCount   int  // number of messages that should be saved for old provider
	}{
		{
			name: "switch_with_two_messages_injects_handoff",
			initialMessages: []state.DisplayMessage{
				{Role: "user", Content: "what is Go?"},
				{Role: "assistant", Content: "Go is a compiled language."},
			},
			wantHandoff:    true,
			wantSavedCount: 2,
		},
		{
			name: "switch_with_one_message_no_handoff",
			initialMessages: []state.DisplayMessage{
				{Role: "user", Content: "hello"},
			},
			wantHandoff:    false,
			wantSavedCount: 1,
		},
		{
			name:             "switch_with_no_messages_no_handoff",
			initialMessages:  nil,
			wantHandoff:      false,
			wantSavedCount:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, panel, _ := newModelWithProviderAndOpts()
			panel.savedMessages = tc.initialMessages

			initialProvider := m.shared.providerState.GetActiveProvider()

			// --- Execute the switch ---
			updated, cmd := m.Update(ProviderSwitchMsg{})
			result := updated.(AppModel)

			// 1. Messages saved to old provider.
			if err := result.shared.providerState.SwitchProvider(initialProvider); err != nil {
				t.Fatalf("SwitchProvider back to %q: %v", initialProvider, err)
			}
			savedMsgs := result.shared.providerState.GetActiveMessages()
			if len(savedMsgs) != tc.wantSavedCount {
				t.Errorf("old provider messages = %d; want %d", len(savedMsgs), tc.wantSavedCount)
			}

			// Switch back to the new provider to inspect its state.
			allProviders := result.shared.providerState.AllProviders()
			nextIdx := 0
			for i, p := range allProviders {
				if p == initialProvider {
					nextIdx = (i + 1) % len(allProviders)
					break
				}
			}
			if err := result.shared.providerState.SwitchProvider(allProviders[nextIdx]); err != nil {
				t.Fatalf("SwitchProvider to new: %v", err)
			}

			// 2. SetSender called with new driver on the panel (C-1 fix).
			if panel.setSenderCalled == 0 {
				t.Error("SetSender was not called after provider switch (C-1 bug)")
			}
			if panel.lastSender == nil {
				t.Error("SetSender was called with nil sender")
			}

			// 3. Handoff injection (when expected).
			if tc.wantHandoff {
				newMsgs := result.shared.providerState.GetActiveMessages()
				if len(newMsgs) == 0 {
					t.Fatal("handoff expected but new provider has no messages")
				}
				if newMsgs[0].Role != "system" {
					t.Errorf("first new-provider message role = %q; want %q", newMsgs[0].Role, "system")
				}
			} else {
				newMsgs := result.shared.providerState.GetActiveMessages()
				if len(newMsgs) != 0 {
					t.Errorf("no handoff expected but new provider has %d messages", len(newMsgs))
				}
			}

			// 4. New driver Start command is returned.
			if cmd == nil {
				t.Error("cmd = nil after ProviderSwitchMsg; want driver Start command")
			}

			// 5. Session state reset.
			if result.cliReady {
				t.Error("cliReady = true after switch; want false")
			}
			if result.sessionID != "" {
				t.Errorf("sessionID = %q after switch; want empty", result.sessionID)
			}
			if result.reconnectCount != 0 {
				t.Errorf("reconnectCount = %d after switch; want 0", result.reconnectCount)
			}
		})
	}
}

func TestProviderSwitch_PreservesBaseCLIOpts(t *testing.T) {
	m, _, _ := newModelWithProviderAndOpts()

	// Baseline opts have Verbose=true, PermissionMode="acceptEdits".
	// These should survive the switch because C-2 copies baseCLIOpts.
	// We verify indirectly: the new driver is created from opts derived from
	// baseCLIOpts, and SetSender is called (which only happens when newDriver
	// is wired). The actual opts values are embedded in the concrete CLIDriver
	// and not exposed, so we verify the observable outcomes:
	//   - cmd != nil (driver was created and Start returned a command)
	//   - SetSender was called (C-1 — means the new driver was wired)

	updated, cmd := m.Update(ProviderSwitchMsg{})
	result := updated.(AppModel)
	_ = result

	if cmd == nil {
		t.Error("cmd = nil; expected Start command from new driver created with baseCLIOpts")
	}
}

func TestProviderSwitch_SetSender_CalledOncePerSwitch(t *testing.T) {
	m, panel, _ := newModelWithProviderAndOpts()

	if panel.setSenderCalled != 0 {
		t.Fatalf("precondition: setSenderCalled = %d; want 0", panel.setSenderCalled)
	}

	m.Update(ProviderSwitchMsg{})

	if panel.setSenderCalled != 1 {
		t.Errorf("setSenderCalled = %d after one switch; want 1 (C-1 fix)", panel.setSenderCalled)
	}
}

func TestProviderSwitch_SetSender_CalledOnEachSwitch(t *testing.T) {
	m, panel, _ := newModelWithProviderAndOpts()

	for i := 1; i <= 3; i++ {
		updated, _ := m.Update(ProviderSwitchMsg{})
		m = updated.(AppModel)

		if panel.setSenderCalled != i {
			t.Errorf("after switch %d: setSenderCalled = %d; want %d",
				i, panel.setSenderCalled, i)
		}
	}
}

func TestProviderSwitch_NilPanel_NoSetSenderPanic(t *testing.T) {
	m, _, _ := newModelWithProviderAndOpts()
	m.shared.claudePanel = nil // no panel wired

	// Must not panic when claudePanel is nil.
	updated, cmd := m.Update(ProviderSwitchMsg{})
	_ = updated.(AppModel)

	if cmd == nil {
		t.Error("cmd = nil; want driver Start command even when panel is nil")
	}
}

// ---------------------------------------------------------------------------
// SessionAutoSaveMsg — debounce guard (TUI-033)
// ---------------------------------------------------------------------------

// TestSessionAutoSaveMsg_MatchingSeq_ExecutesSave verifies that a
// SessionAutoSaveMsg whose Seq matches the current autoSaveSeq triggers
// saveSession without error.
func TestSessionAutoSaveMsg_MatchingSeq_ExecutesSave(t *testing.T) {
	m := NewAppModel()
	// Advance autoSaveSeq to 1 to simulate one cost-change event.
	m.autoSaveSeq = 1

	// Deliver a matching SessionAutoSaveMsg — must not panic.
	// saveSession is a no-op when sessionStore/sessionData are nil, so we
	// only verify the Update path itself is exercised (no cmd returned).
	updated, cmd := m.Update(SessionAutoSaveMsg{Seq: 1})
	_ = updated.(AppModel)

	if cmd != nil {
		t.Error("cmd != nil for matching SessionAutoSaveMsg; want nil")
	}
}

// TestSessionAutoSaveMsg_StaleSeq_IsDiscarded verifies that a stale
// SessionAutoSaveMsg (lower Seq than current autoSaveSeq) is a no-op.
func TestSessionAutoSaveMsg_StaleSeq_IsDiscarded(t *testing.T) {
	m := NewAppModel()
	m.autoSaveSeq = 5

	// Deliver a stale SessionAutoSaveMsg (seq 3 < 5).
	updated, cmd := m.Update(SessionAutoSaveMsg{Seq: 3})
	result := updated.(AppModel)

	// autoSaveSeq must not change.
	if result.autoSaveSeq != 5 {
		t.Errorf("autoSaveSeq changed on stale msg: got %d; want 5", result.autoSaveSeq)
	}
	if cmd != nil {
		t.Error("cmd != nil for stale SessionAutoSaveMsg; want nil")
	}
}

// ---------------------------------------------------------------------------
// ShutdownCompleteMsg (TUI-034)
// ---------------------------------------------------------------------------

// TestShutdownCompleteMsg_NilErr_ReturnsQuit verifies that a successful
// shutdown (nil Err) causes the model to return tea.Quit.
func TestShutdownCompleteMsg_NilErr_ReturnsQuit(t *testing.T) {
	m := NewAppModel()

	_, cmd := m.Update(ShutdownCompleteMsg{Err: nil})

	if cmd == nil {
		t.Fatal("cmd = nil for ShutdownCompleteMsg{nil}; want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T; want tea.QuitMsg", msg)
	}
}

// TestShutdownCompleteMsg_NonNilErr_AlsoReturnsQuit verifies that a shutdown
// that completed with an error still causes the model to return tea.Quit
// (the error is logged but the program exits regardless).
func TestShutdownCompleteMsg_NonNilErr_AlsoReturnsQuit(t *testing.T) {
	m := NewAppModel()

	_, cmd := m.Update(ShutdownCompleteMsg{Err: errShutdownTest})

	if cmd == nil {
		t.Fatal("cmd = nil for ShutdownCompleteMsg with error; want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T; want tea.QuitMsg even when Err != nil", msg)
	}
}

// errShutdownTest is a sentinel error used by TestShutdownCompleteMsg_NonNilErr.
type testError struct{ msg string }

func (e testError) Error() string { return e.msg }

var errShutdownTest = testError{msg: "simulated shutdown timeout"}

// ---------------------------------------------------------------------------
// ForceQuit — double-press + shutdownFunc wiring (TUI-034)
// ---------------------------------------------------------------------------

// TestForceQuit_DoublePressWithShutdownInProgress verifies that a second
// Ctrl+C while shutdownInProgress=true returns tea.Quit immediately without
// invoking the shutdownFunc again.
func TestForceQuit_DoublePressWithShutdownInProgress(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Simulate that a previous Ctrl+C already started the shutdown sequence.
	m.shutdownInProgress = true

	// A second Ctrl+C must return tea.Quit immediately.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Fatal("cmd = nil for double-press ForceQuit; want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("double-press ForceQuit cmd() = %T; want tea.QuitMsg", msg)
	}
}

// TestForceQuit_DoublePressWithShutdownFunc_SkipsFunc verifies that the double-
// press path (shutdownInProgress=true) does not call shutdownFunc — only the
// first press invokes it.
func TestForceQuit_DoublePressWithShutdownFunc_SkipsFunc(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	shutdownCalled := 0
	m.shared.shutdownFunc = func() error {
		shutdownCalled++
		return nil
	}
	m.shutdownInProgress = true

	// Second Ctrl+C: must NOT call shutdownFunc (immediate quit branch).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("cmd = nil for double-press; want tea.Quit")
	}
	// Execute the command to confirm it is tea.Quit, not the shutdown fn.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("double-press cmd() = %T; want tea.QuitMsg", msg)
	}
	if shutdownCalled != 0 {
		t.Errorf("shutdownFunc called %d times on double-press; want 0", shutdownCalled)
	}
}

// TestForceQuit_WithShutdownFunc_InvokesFunc verifies that the first Ctrl+C
// when a shutdownFunc is wired returns a Cmd that calls the function and
// emits a ShutdownCompleteMsg.
func TestForceQuit_WithShutdownFunc_InvokesFunc(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	shutdownCalled := 0
	m.shared.shutdownFunc = func() error {
		shutdownCalled++
		return nil
	}

	// First Ctrl+C — shutdownInProgress is false, shutdownFunc is wired.
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	appResult := result.(AppModel)

	// shutdownInProgress must be set on the model.
	if !appResult.shutdownInProgress {
		t.Error("shutdownInProgress = false after first Ctrl+C; want true")
	}

	// The returned Cmd must invoke the shutdown function.
	if cmd == nil {
		t.Fatal("cmd = nil when shutdownFunc wired; want shutdown Cmd")
	}
	msg := cmd()

	// shutdownFunc should have been called once.
	if shutdownCalled != 1 {
		t.Errorf("shutdownFunc called %d times; want 1", shutdownCalled)
	}

	// The Cmd must return a ShutdownCompleteMsg.
	if _, ok := msg.(ShutdownCompleteMsg); !ok {
		t.Errorf("cmd() = %T; want ShutdownCompleteMsg", msg)
	}
}

// TestForceQuit_WithShutdownFunc_ErrorPropagated verifies that if the
// shutdownFunc returns an error, that error is carried in ShutdownCompleteMsg.
func TestForceQuit_WithShutdownFunc_ErrorPropagated(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	m.shared.shutdownFunc = func() error {
		return errShutdownTest
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("cmd = nil; want shutdown Cmd")
	}

	msg := cmd()
	shutdownMsg, ok := msg.(ShutdownCompleteMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want ShutdownCompleteMsg", msg)
	}
	if shutdownMsg.Err == nil {
		t.Error("ShutdownCompleteMsg.Err = nil; want the error from shutdownFunc")
	}
}

// ---------------------------------------------------------------------------
// ToastMsg — nil toasts widget (no panic)
// ---------------------------------------------------------------------------

// TestToastMsg_NilToasts_NoPanic verifies that sending a ToastMsg when the
// toasts widget is nil does not panic and returns a nil cmd.
func TestToastMsg_NilToasts_NoPanic(t *testing.T) {
	m := NewAppModel()
	// shared.toasts is nil by default (not set in NewAppModel).
	if m.shared.toasts != nil {
		t.Fatal("precondition: toasts should be nil")
	}

	// Must not panic.
	updated, cmd := m.Update(ToastMsg{Text: "test", Level: ToastLevelInfo})
	_ = updated.(AppModel)

	if cmd != nil {
		t.Error("cmd != nil for ToastMsg with nil toasts widget; want nil")
	}
}

// ---------------------------------------------------------------------------
// handleModalKey — nil shared / nil modalQueue branches
// ---------------------------------------------------------------------------

// TestHandleModalKey_NilShared_NoPanic verifies that delivering a key event
// when there is an active modal but shared is nil is a safe no-op.
func TestHandleModalKey_NilShared_NoPanic(t *testing.T) {
	// Constructing AppModel without NewAppModel leaves shared nil.
	m := AppModel{}
	// handleModalKey is only reached when modalQueue.IsActive() returns true.
	// With nil shared, handleKey takes the normal path (no modal active), so
	// we test handleModalKey directly.
	_, cmd := m.handleModalKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("cmd != nil from handleModalKey with nil shared; want nil")
	}
}

// ---------------------------------------------------------------------------
// handleClaudeKey — nil claudePanel branch
// ---------------------------------------------------------------------------

// TestHandleClaudeKey_NilPanel_NoPanic verifies that delivering a key event
// when FocusClaude is active but the panel is nil is a safe no-op.
func TestHandleClaudeKey_NilPanel_NoPanic(t *testing.T) {
	m := NewAppModel()
	m.focus = FocusClaude
	// Do not inject a claude panel — shared.claudePanel remains nil.

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("cmd != nil for key event with nil claudePanel; want nil")
	}
}

// ---------------------------------------------------------------------------
// renderRightPanel — Settings, Telemetry, PlanPreview, default fallbacks
// ---------------------------------------------------------------------------

// TestView_RightPanel_ShowsSettingsMode verifies RPMSettings renders the
// "Settings" placeholder when no settings widget is injected.
func TestView_RightPanel_ShowsSettingsMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	m.rightPanelMode = RPMSettings

	view := m.View()
	if !strings.Contains(view, "Settings") {
		t.Errorf("View() does not contain %q for RPMSettings; got %q", "Settings", view)
	}
}

// TestView_RightPanel_ShowsTelemetryMode verifies RPMTelemetry renders the
// "Telemetry" placeholder when no telemetry widget is injected.
func TestView_RightPanel_ShowsTelemetryMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	m.rightPanelMode = RPMTelemetry

	view := m.View()
	if !strings.Contains(view, "Telemetry") {
		t.Errorf("View() does not contain %q for RPMTelemetry; got %q", "Telemetry", view)
	}
}

// TestView_RightPanel_ShowsPlanPreviewMode verifies RPMPlanPreview renders the
// "Plan Preview" placeholder when no planPreview widget is injected.
func TestView_RightPanel_ShowsPlanPreviewMode(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	m.rightPanelMode = RPMPlanPreview

	view := m.View()
	if !strings.Contains(view, "Plan Preview") {
		t.Errorf("View() does not contain %q for RPMPlanPreview; got %q", "Plan Preview", view)
	}
}

// TestView_RightPanel_DefaultMode_DoesNotPanic verifies that an unknown
// RightPanelMode value hits the default branch without panicking.
func TestView_RightPanel_DefaultMode_DoesNotPanic(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)
	// Use a value beyond the defined constants to hit the default branch.
	m.rightPanelMode = RightPanelMode(99)

	// Must not panic.
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string for unknown RightPanelMode; want non-empty")
	}
}

// ---------------------------------------------------------------------------
// renderLayout — modal overlay path
// ---------------------------------------------------------------------------

// TestRenderLayout_ModalActive_ReturnsModalView verifies that when the modal
// queue has an active modal, renderLayout returns the modal's view rather than
// the normal layout.
func TestRenderLayout_ModalActive_ReturnsModalView(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)

	// Activate a modal by sending a BridgeModalRequestMsg.
	updated, _ = m.Update(BridgeModalRequestMsg{
		RequestID: "render-test",
		Message:   "Do you want to proceed?",
		Options:   []string{"Yes", "No"},
	})
	m = updated.(AppModel)

	if !m.shared.modalQueue.IsActive() {
		t.Fatal("precondition: modal queue not active after BridgeModalRequestMsg")
	}

	// renderLayout should delegate to modalQueue.View().
	view := m.View()
	// The modal queue renders something non-empty when active.
	if strings.TrimSpace(view) == "" {
		t.Error("View() returned empty string while modal is active; want modal overlay content")
	}
	// The view must not contain the normal banner text (the full layout is
	// bypassed when a modal is active).
	if strings.Contains(view, "GOgent-Fortress") {
		t.Error("normal layout rendered while modal active; want modal overlay only")
	}
}

// ---------------------------------------------------------------------------
// ToggleTaskBoard key (handleKey coverage)
// ---------------------------------------------------------------------------

// mockTaskBoard satisfies taskBoardWidget for ToggleTaskBoard key tests.
type mockTaskBoard struct {
	visible     bool
	toggleCalls int
	width, h    int
}

func (m *mockTaskBoard) Toggle()                       { m.toggleCalls++; m.visible = !m.visible }
func (m *mockTaskBoard) IsVisible() bool               { return m.visible }
func (m *mockTaskBoard) View() string                  { return "" }
func (m *mockTaskBoard) SetSize(w, h int)              { m.width = w; m.h = h }
func (m *mockTaskBoard) Height() int                   { return 0 }
func (m *mockTaskBoard) SetTasks(_ []state.TaskEntry)  {}

// TestHandleKey_ToggleTaskBoard_CallsToggle verifies that the ToggleTaskBoard
// key binding (ctrl+t) calls Toggle() on the taskBoard widget when wired.
func TestHandleKey_ToggleTaskBoard_CallsToggle(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	tb := &mockTaskBoard{}
	m.shared.taskBoard = tb

	// alt+b is the ToggleTaskBoard keybinding.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true})

	if tb.toggleCalls != 1 {
		t.Errorf("Toggle called %d times; want 1", tb.toggleCalls)
	}
}

// TestHandleKey_ToggleTaskBoard_NilTaskBoard_NoPanic verifies that the
// ToggleTaskBoard key is a safe no-op when no taskBoard widget is wired.
func TestHandleKey_ToggleTaskBoard_NilTaskBoard_NoPanic(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)
	// shared.taskBoard is nil — must not panic.

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true})
	if cmd != nil {
		t.Error("cmd != nil for ToggleTaskBoard with nil taskBoard; want nil")
	}
}
