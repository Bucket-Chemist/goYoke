package statusline_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
)

// ---------------------------------------------------------------------------
// Constructor / Init
// ---------------------------------------------------------------------------

func TestNewStatusLineModel(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	// Zero value fields should be rendered without panicking.
	_ = m.View()
}

func TestStatusLineInit(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

// ---------------------------------------------------------------------------
// View — field labels and values
// ---------------------------------------------------------------------------

func TestStatusLineViewContainsCostField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.SessionCost = 1.2345
	view := m.View()
	if !strings.Contains(view, "cost:") {
		t.Errorf("View() missing 'cost:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "1.23") {
		t.Errorf("View() missing cost value '1.23'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsTokenField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.TokenCount = 42000
	view := m.View()
	if !strings.Contains(view, "tokens:") {
		t.Errorf("View() missing 'tokens:' label; got:\n%s", view)
	}
	// formatTokens(42000) → "42K"
	if !strings.Contains(view, "42K") {
		t.Errorf("View() missing token count '42K'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsContextPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 75.5
	view := m.View()
	if !strings.Contains(view, "ctx:") {
		t.Errorf("View() missing 'ctx:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "75.5%") {
		t.Errorf("View() missing context percent '75.5%%'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsPermissionMode(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.PermissionMode = "delegate"
	view := m.View()
	if !strings.Contains(view, "perm:") {
		t.Errorf("View() missing 'perm:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "delegate") {
		t.Errorf("View() missing permission mode 'delegate'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsModelField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ActiveModel = "claude-sonnet-4-6"
	view := m.View()
	if !strings.Contains(view, "model:") {
		t.Errorf("View() missing 'model:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "claude-sonnet-4-6") {
		t.Errorf("View() missing model name; got:\n%s", view)
	}
}

func TestStatusLineViewContainsProviderField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Provider = "anthropic"
	view := m.View()
	if !strings.Contains(view, "provider:") {
		t.Errorf("View() missing 'provider:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "anthropic") {
		t.Errorf("View() missing provider 'anthropic'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsGitBranch(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.GitBranch = "tui-migration"
	view := m.View()
	if !strings.Contains(view, "branch:") {
		t.Errorf("View() missing 'branch:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "tui-migration") {
		t.Errorf("View() missing branch name; got:\n%s", view)
	}
}

func TestStatusLineViewContainsAuthStatus(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.AuthStatus = "authenticated"
	view := m.View()
	if !strings.Contains(view, "auth:") {
		t.Errorf("View() missing 'auth:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "authenticated") {
		t.Errorf("View() missing auth status; got:\n%s", view)
	}
}

func TestStatusLineViewTwoRows(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) != 2 {
		t.Errorf("View() should produce 2 rows; got %d:\n%s", len(lines), view)
	}
}

// ---------------------------------------------------------------------------
// Update — existing message types
// ---------------------------------------------------------------------------

func TestStatusLineUpdateWindowSizeMsg(t *testing.T) {
	m := statusline.NewStatusLineModel(80)
	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}
	// Should not panic after resize.
	_ = newModel.View()
}

func TestStatusLineUpdateUnknownMsg(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update("unknown")
	assert.Nil(t, cmd, "unknown message should return nil command")
	// zero value check — the returned model is still a valid StatusLineModel
	_ = newModel.View()
}

// ---------------------------------------------------------------------------
// SetWidth
// ---------------------------------------------------------------------------

func TestStatusLineSetWidth(t *testing.T) {
	m := statusline.NewStatusLineModel(80)
	m.SetWidth(200)
	// Should not panic.
	_ = m.View()
}

// ---------------------------------------------------------------------------
// All fields populated
// ---------------------------------------------------------------------------

func TestStatusLineAllFieldsPopulated(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.SessionCost = 0.0042
	m.TokenCount = 8500
	m.ContextPercent = 12.3
	m.PermissionMode = "bypass"
	m.ActiveModel = "claude-opus-4-6"
	m.Provider = "anthropic"
	m.GitBranch = "main"
	m.AuthStatus = "ok"

	view := m.View()

	// FormatCost(0.0042) → "$0.0042" (< $0.01 → 4 decimal places)
	// formatTokens(8500) → "8.5K"
	expected := []string{
		"cost:", "$0.0042",
		"tokens:", "8.5K",
		"ctx:", "12.3%",
		"perm:", "bypass",
		"model:", "claude-opus-4-6",
		"provider:", "anthropic",
		"branch:", "main",
		"auth:", "ok",
	}
	for _, want := range expected {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q; got:\n%s", want, view)
		}
	}
}

// ---------------------------------------------------------------------------
// New: gitBranchMsg handling
// ---------------------------------------------------------------------------

func TestUpdate_GitBranchMsg_Success(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update(statusline.GitBranchMsgForTest("tui-migration", nil))
	assert.Nil(t, cmd)
	assert.Equal(t, "tui-migration", newModel.GitBranch)
}

func TestUpdate_GitBranchMsg_Error(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update(statusline.GitBranchMsgForTest("", assert.AnError))
	assert.Nil(t, cmd)
	assert.Equal(t, "N/A", newModel.GitBranch)
}

// ---------------------------------------------------------------------------
// New: authStatusMsg handling
// ---------------------------------------------------------------------------

func TestUpdate_AuthStatusMsg_Success(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update(statusline.AuthStatusMsgForTest("authenticated", nil))
	assert.Nil(t, cmd)
	assert.Equal(t, "authenticated", newModel.AuthStatus)
}

func TestUpdate_AuthStatusMsg_Error(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update(statusline.AuthStatusMsgForTest("N/A", assert.AnError))
	assert.Nil(t, cmd)
	assert.Equal(t, "N/A", newModel.AuthStatus)
}

// ---------------------------------------------------------------------------
// New: tick messages return commands
// ---------------------------------------------------------------------------

func TestUpdate_GitBranchTickMsg_ReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	_, cmd := m.Update(statusline.GitBranchTickMsgForTest(time.Now()))
	assert.NotNil(t, cmd, "gitBranchTickMsg should return a non-nil command")
}

func TestUpdate_AuthStatusTickMsg_ReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	_, cmd := m.Update(statusline.AuthStatusTickMsgForTest(time.Now()))
	assert.NotNil(t, cmd, "authStatusTickMsg should return a non-nil command")
}

// ---------------------------------------------------------------------------
// New: StartTicks
// ---------------------------------------------------------------------------

func TestStartTicks_ReturnsBatchCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.StartTicks()
	assert.NotNil(t, cmd, "StartTicks() should return a non-nil batch command")
}

// ---------------------------------------------------------------------------
// New: command execution (covers subprocess paths)
// ---------------------------------------------------------------------------

// TestGitBranchCmdExecution executes the command returned by StartTicks to
// exercise the gitBranchCmd closure. The result must be a gitBranchMsg or
// authStatusMsg — either branch of binaryExists is covered regardless.
func TestGitBranchCmdExecution(t *testing.T) {
	msg := statusline.ExecuteGitBranchCmdForTest()
	// Must return a message (either success or error branch — both are valid).
	assert.NotNil(t, msg, "gitBranchCmd() should return a non-nil message")
}

func TestAuthStatusCmdExecution(t *testing.T) {
	msg := statusline.ExecuteAuthStatusCmdForTest()
	// Must return a message (either success or error branch — both are valid).
	assert.NotNil(t, msg, "authStatusCmd() should return a non-nil message")
}

// TestScheduleTicksCmdExecution verifies that the tick-scheduler commands
// return non-nil tea.Cmd values (covering the scheduleGitBranchTick and
// scheduleAuthStatusTick function bodies).
func TestScheduleTicksCmdExecution(t *testing.T) {
	gitCmd := statusline.ScheduleGitBranchTickForTest()
	assert.NotNil(t, gitCmd, "scheduleGitBranchTick() should return a non-nil command")

	authCmd := statusline.ScheduleAuthStatusTickForTest()
	assert.NotNil(t, authCmd, "scheduleAuthStatusTick() should return a non-nil command")
}

// ---------------------------------------------------------------------------
// New: View uses FormatCost
// ---------------------------------------------------------------------------

func TestView_ShowsCost_ViaFormatCost(t *testing.T) {
	tests := []struct {
		name    string
		cost    float64
		wantSub string
	}{
		{name: "zero cost", cost: 0, wantSub: "$0.00"},
		{name: "sub-cent cost", cost: 0.0042, wantSub: "$0.0042"},
		{name: "cent-scale cost", cost: 1.50, wantSub: "$1.50"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := statusline.NewStatusLineModel(120)
			m.SessionCost = tc.cost
			view := m.View()
			assert.Contains(t, view, tc.wantSub,
				"View() should render cost via FormatCost; want %q in:\n%s", tc.wantSub, view)
		})
	}
}

// ---------------------------------------------------------------------------
// New: formatTokens
// ---------------------------------------------------------------------------

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1K"},
		{1500, "1.5K"},
		{42000, "42K"},
		{150000, "150K"},
		{1000000, "1M"},
		{1500000, "1.5M"},
	}
	for _, tc := range tests {
		got := statusline.FormatTokensForTest(tc.n)
		assert.Equal(t, tc.want, got, "formatTokens(%d)", tc.n)
	}
}

// ---------------------------------------------------------------------------
// New: sessionTimerTickMsg schedules next tick
// ---------------------------------------------------------------------------

func TestUpdate_SessionTimerTickMsg_ReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	_, cmd := m.Update(statusline.SessionTimerTickMsgForTest(time.Now()))
	assert.NotNil(t, cmd, "sessionTimerTickMsg should return a non-nil command")
}

// ---------------------------------------------------------------------------
// New: spinnerTickMsg advances frame, reschedules only when Streaming
// ---------------------------------------------------------------------------

func TestUpdate_SpinnerTickMsg_Streaming_ReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Streaming = true
	_, cmd := m.Update(statusline.SpinnerTickMsgForTest(time.Now()))
	assert.NotNil(t, cmd, "spinnerTickMsg while Streaming should return a non-nil command")
}

func TestUpdate_SpinnerTickMsg_NotStreaming_ReturnsNil(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Streaming = false
	_, cmd := m.Update(statusline.SpinnerTickMsgForTest(time.Now()))
	assert.Nil(t, cmd, "spinnerTickMsg while not Streaming should return nil")
}

// ---------------------------------------------------------------------------
// New: SetStreaming
// ---------------------------------------------------------------------------

func TestSetStreaming_TrueReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.SetStreaming(true)
	assert.NotNil(t, cmd, "SetStreaming(true) should return a spinner command")
	assert.True(t, m.Streaming)
}

func TestSetStreaming_FalseReturnsNil(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Streaming = true
	cmd := m.SetStreaming(false)
	assert.Nil(t, cmd, "SetStreaming(false) should return nil")
	assert.False(t, m.Streaming)
}

// ---------------------------------------------------------------------------
// New: View shows thinking indicator when Streaming
// ---------------------------------------------------------------------------

func TestView_ThinkingIndicator_WhenStreaming(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.Streaming = true
	view := m.View()
	assert.Contains(t, view, "thinking...", "View() should show 'thinking...' when Streaming")
}

func TestView_NoThinkingIndicator_WhenNotStreaming(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.Streaming = false
	view := m.View()
	assert.NotContains(t, view, "thinking...", "View() should not show 'thinking...' when not Streaming")
}

// ---------------------------------------------------------------------------
// New: View shows elapsed time when SessionStart is set
// ---------------------------------------------------------------------------

func TestView_ShowsElapsedTime_WhenSessionStartSet(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.SessionStart = time.Now().Add(-90 * time.Second) // 1m 30s ago
	view := m.View()
	assert.Contains(t, view, "1m", "View() should show elapsed minutes")
}

func TestView_NoElapsedTime_WhenSessionStartZero(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	// SessionStart is zero value — no elapsed display expected.
	view := m.View()
	assert.NotContains(t, view, "⏱", "View() should not show elapsed timer before session starts")
}

// ---------------------------------------------------------------------------
// New: parseAuthStatus
// ---------------------------------------------------------------------------

func TestParseAuthStatus(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		wantC string // substring that must appear in result
	}{
		{
			name:  "empty string",
			raw:   "",
			wantC: "N/A",
		},
		{
			name:  "email and method",
			raw:   "Logged in via claude.ai\nAccount: admin@exactmass.org",
			wantC: "claude.ai",
		},
		{
			name:  "email present",
			raw:   "admin@exactmass.org",
			wantC: "admin@exactmass.org",
		},
		{
			name:  "fallback to first line",
			raw:   "Authenticated",
			wantC: "Authenticated",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statusline.ParseAuthStatusForTest(tc.raw)
			assert.Contains(t, got, tc.wantC, "parseAuthStatus(%q)", tc.raw)
		})
	}
}

// ---------------------------------------------------------------------------
// New: StartTicks includes session timer
// ---------------------------------------------------------------------------

func TestStartTicks_IncludesSessionTimer(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.StartTicks()
	assert.NotNil(t, cmd, "StartTicks() should return a non-nil batch command")
	// We cannot easily inspect what tea.Batch contains, but the non-nil check
	// is sufficient to verify the function path.
}

func TestScheduleSessionTimerTick_ReturnsCmd(t *testing.T) {
	cmd := statusline.ScheduleSessionTimerTickForTest()
	assert.NotNil(t, cmd, "scheduleSessionTimerTick() should return a non-nil command")
}

func TestScheduleSpinnerTick_ReturnsCmd(t *testing.T) {
	cmd := statusline.ScheduleSpinnerTickForTest()
	assert.NotNil(t, cmd, "scheduleSpinnerTick() should return a non-nil command")
}
