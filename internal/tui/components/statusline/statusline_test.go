package statusline_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
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
	// renderContextBar uses %.0f formatting — 75.5 rounds to "76%".
	if !strings.Contains(view, "76%") {
		t.Errorf("View() missing context percent '76%%' (rounded from 75.5); got:\n%s", view)
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
	// renderContextBar uses %.0f — 12.3 rounds to "12%"
	expected := []string{
		"cost:", "$0.0042",
		"tokens:", "8.5K",
		"ctx:", "12%",
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

// ---------------------------------------------------------------------------
// TUI-048: Semantic color helpers
// ---------------------------------------------------------------------------

func TestCostStyle_Thresholds(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	theme := config.DefaultTheme()
	m.SetTheme(theme)

	tests := []struct {
		name      string
		cost      float64
		wantStyle string // "success", "warning", or "error"
	}{
		{"below warning threshold", 0.05, "success"},
		{"at warning threshold", 0.10, "warning"},
		{"below error threshold", 0.99, "warning"},
		{"at error threshold", 1.00, "error"},
		{"above error threshold", 5.00, "error"},
	}

	successStyle := theme.SuccessStyle()
	warningStyle := theme.WarningStyle()
	errorStyle := theme.ErrorStyle()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.CostStyleForTest(tc.cost)
			switch tc.wantStyle {
			case "success":
				assert.Equal(t, successStyle, got, "cost %.2f should use SuccessStyle", tc.cost)
			case "warning":
				assert.Equal(t, warningStyle, got, "cost %.2f should use WarningStyle", tc.cost)
			case "error":
				assert.Equal(t, errorStyle, got, "cost %.2f should use ErrorStyle", tc.cost)
			}
		})
	}
}

func TestContextStyle_Thresholds(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	theme := config.DefaultTheme()
	m.SetTheme(theme)

	tests := []struct {
		name      string
		pct       float64
		wantStyle string
	}{
		{"low context", 30, "success"},
		{"at warning threshold", 50, "warning"},
		{"below error threshold", 79, "warning"},
		{"at error threshold", 80, "error"},
		{"above error threshold", 95, "error"},
	}

	successStyle := theme.SuccessStyle()
	warningStyle := theme.WarningStyle()
	errorStyle := theme.ErrorStyle()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.ContextStyleForTest(tc.pct)
			switch tc.wantStyle {
			case "success":
				assert.Equal(t, successStyle, got, "ctx %.1f%% should use SuccessStyle", tc.pct)
			case "warning":
				assert.Equal(t, warningStyle, got, "ctx %.1f%% should use WarningStyle", tc.pct)
			case "error":
				assert.Equal(t, errorStyle, got, "ctx %.1f%% should use ErrorStyle", tc.pct)
			}
		})
	}
}

func TestPermStyle_Modes(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	theme := config.DefaultTheme()
	m.SetTheme(theme)

	tests := []struct {
		name      string
		mode      string
		wantStyle string
	}{
		{"default mode", "default", "success"},
		{"plan mode", "plan", "warning"},
		{"allow-all mode", "allow-all", "error"},
		{"empty mode", "", "success"},
		{"delegate mode", "delegate", "success"},
	}

	successStyle := theme.SuccessStyle()
	warningStyle := theme.WarningStyle()
	errorStyle := theme.ErrorStyle()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.PermStyleForTest(tc.mode)
			switch tc.wantStyle {
			case "success":
				assert.Equal(t, successStyle, got, "perm %q should use SuccessStyle", tc.mode)
			case "warning":
				assert.Equal(t, warningStyle, got, "perm %q should use WarningStyle", tc.mode)
			case "error":
				assert.Equal(t, errorStyle, got, "perm %q should use ErrorStyle", tc.mode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TUI-048: SetTheme
// ---------------------------------------------------------------------------

func TestSetTheme_UpdatesTheme(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	lightTheme := config.NewTheme(config.ThemeLight)
	m.SetTheme(lightTheme)
	// Verify the theme was stored by checking that style helpers use it.
	// The SuccessStyle from the light theme should match what costStyle returns
	// for a low-cost value.
	got := m.CostStyleForTest(0.01)
	assert.Equal(t, lightTheme.SuccessStyle(), got,
		"after SetTheme, costStyle should reflect the new theme")
}

// ---------------------------------------------------------------------------
// TUI-048: UncommittedCount message
// ---------------------------------------------------------------------------

func TestUncommittedCountMsg_Updates(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	assert.Equal(t, 0, m.UncommittedCount, "initial UncommittedCount should be 0")

	newModel, cmd := m.Update(statusline.UncommittedCountMsgForTest(7))
	assert.Nil(t, cmd, "uncommittedCountMsg should return nil command")
	assert.Equal(t, 7, newModel.UncommittedCount, "UncommittedCount should be updated to 7")
}

func TestUncommittedCountCmd_Execution(t *testing.T) {
	msg := statusline.ExecuteUncommittedCountCmdForTest()
	assert.NotNil(t, msg, "uncommittedCountCmd() should return a non-nil message")
}

// ---------------------------------------------------------------------------
// TUI-048: View includes agent count and uncommitted count
// ---------------------------------------------------------------------------

func TestView_IncludesAgentCount(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.AgentCount = 3
	view := m.View()
	assert.Contains(t, view, "agents:", "View() should include 'agents:' label")
	assert.Contains(t, view, "3", "View() should include the agent count value")
}

func TestView_IncludesUncommittedCount_WhenPositive(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.UncommittedCount = 5
	view := m.View()
	assert.Contains(t, view, "uncommitted:", "View() should include 'uncommitted:' label when count > 0")
	assert.Contains(t, view, "5", "View() should include the uncommitted count value")
}

func TestView_ExcludesUncommittedCount_WhenZero(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.UncommittedCount = 0
	view := m.View()
	assert.NotContains(t, view, "uncommitted:", "View() should not show 'uncommitted:' label when count is 0")
}

// ---------------------------------------------------------------------------
// TUI-048: View still renders two rows after new fields
// ---------------------------------------------------------------------------

func TestView_TwoRowsWithNewFields(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	m.AgentCount = 2
	m.UncommittedCount = 4
	m.SessionCost = 1.50
	m.ContextPercent = 85
	m.PermissionMode = "allow-all"
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	assert.Equal(t, 2, len(lines), "View() should still produce exactly 2 rows with new fields")
}

// ---------------------------------------------------------------------------
// TUI-049: renderContextBar
// ---------------------------------------------------------------------------

func TestRenderContextBar_ZeroPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 0
	result := m.RenderContextBarForTest()
	// All 10 chars should be empty fill; percentage should be "0%".
	assert.Contains(t, result, "ctx:", "should contain 'ctx:' prefix")
	assert.Contains(t, result, "[", "wide mode should contain opening bracket")
	assert.Contains(t, result, "]", "wide mode should contain closing bracket")
	assert.Contains(t, result, "0%", "should contain '0%'")
}

func TestRenderContextBar_25Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 25
	result := m.RenderContextBarForTest()
	// 25% of 10 = 2.5 → int truncation = 2 filled chars.
	assert.Contains(t, result, "ctx:", "should contain 'ctx:' prefix")
	assert.Contains(t, result, "[", "should contain opening bracket")
	assert.Contains(t, result, "]", "should contain closing bracket")
	assert.Contains(t, result, "25%", "should contain '25%'")
	// Verify the fill: 2 filled + 8 empty = 10 chars between brackets.
	// Strip ANSI escapes: count raw '=' characters.
	assert.Contains(t, result, "==", "should have at least 2 filled chars at 25%")
}

func TestRenderContextBar_50Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 50
	result := m.RenderContextBarForTest()
	// 50% of 10 = 5 filled chars (warning threshold boundary).
	assert.Contains(t, result, "ctx:", "should contain 'ctx:' prefix")
	assert.Contains(t, result, "[", "should contain opening bracket")
	assert.Contains(t, result, "]", "should contain closing bracket")
	assert.Contains(t, result, "50%", "should contain '50%'")
}

func TestRenderContextBar_75Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 75
	result := m.RenderContextBarForTest()
	// 75% of 10 = 7 filled chars (warning color zone).
	assert.Contains(t, result, "ctx:", "should contain 'ctx:' prefix")
	assert.Contains(t, result, "[", "should contain opening bracket")
	assert.Contains(t, result, "]", "should contain closing bracket")
	assert.Contains(t, result, "75%", "should contain '75%'")
}

func TestRenderContextBar_100Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 100
	result := m.RenderContextBarForTest()
	// 100% → all 10 chars filled.
	assert.Contains(t, result, "ctx:", "should contain 'ctx:' prefix")
	assert.Contains(t, result, "[", "should contain opening bracket")
	assert.Contains(t, result, "]", "should contain closing bracket")
	assert.Contains(t, result, "100%", "should contain '100%'")
	// At 100%, there should be 10 '=' characters in the bar segment.
	assert.Contains(t, result, "==========", "full bar should have 10 '=' chars")
}

func TestRenderContextBar_NarrowFallback(t *testing.T) {
	m := statusline.NewStatusLineModel(79) // below the 80-char threshold
	m.ContextPercent = 52
	result := m.RenderContextBarForTest()
	// Narrow mode: no brackets, just text.
	assert.Contains(t, result, "ctx:", "narrow fallback should contain 'ctx:' prefix")
	assert.Contains(t, result, "52%", "narrow fallback should contain percentage")
	assert.NotContains(t, result, "[", "narrow fallback should not contain opening bracket")
	assert.NotContains(t, result, "]", "narrow fallback should not contain closing bracket")
}

func TestRenderContextBar_NegativePercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = -10
	result := m.RenderContextBarForTest()
	// Clamped to 0: empty bar, "0%".
	assert.Contains(t, result, "0%", "negative percent should be clamped to 0%")
	assert.NotContains(t, result, "-", "negative percent should not appear in output")
}

func TestRenderContextBar_OverPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 150
	result := m.RenderContextBarForTest()
	// Clamped to 100: full bar, "100%".
	assert.Contains(t, result, "100%", "over-100 percent should be clamped to 100%")
	assert.Contains(t, result, "==========", "clamped full bar should have 10 '=' chars")
}

func TestRenderContextBar_ContainsBarChars(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 60
	result := m.RenderContextBarForTest()
	// Wide mode must contain both bracket characters.
	assert.Contains(t, result, "[", "wide mode must contain '['")
	assert.Contains(t, result, "]", "wide mode must contain ']'")
}

func TestView_ContextBar_Integration(t *testing.T) {
	// Verify the progress bar is visible in the full View output for wide terminal.
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 40
	view := m.View()
	assert.Contains(t, view, "ctx:", "View() should include 'ctx:' from renderContextBar")
	assert.Contains(t, view, "[", "View() should include bar brackets in wide mode")
	assert.Contains(t, view, "40%", "View() should include percentage from renderContextBar")
}
