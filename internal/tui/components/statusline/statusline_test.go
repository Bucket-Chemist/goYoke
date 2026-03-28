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
// View — field presence (new TS-style layout)
// ---------------------------------------------------------------------------

func TestStatusLineViewContainsCostField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.SessionCost = 1.2345
	view := m.View()
	if !strings.Contains(view, "$1.23") {
		t.Errorf("View() missing cost value '$1.23'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsContextPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 75.5
	m.ContextCapacity = 200_000
	m.ContextUsedTokens = 151_000
	view := m.View()
	// renderContextBar uses %.0f formatting — 75.5 rounds to "76%".
	if !strings.Contains(view, "76%") {
		t.Errorf("View() missing context percent '76%%'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsPermissionMode(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.PermissionMode = "delegate"
	view := m.View()
	// New layout: [delegate] badge
	if !strings.Contains(view, "[delegate]") {
		t.Errorf("View() missing permission badge '[delegate]'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsModelField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ActiveModel = "claude-sonnet-4-6"
	view := m.View()
	// New layout: [claude-sonnet-4-6] badge
	if !strings.Contains(view, "[claude-sonnet-4-6]") {
		t.Errorf("View() missing model badge; got:\n%s", view)
	}
}

func TestStatusLineViewContainsProviderField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Provider = "anthropic"
	view := m.View()
	// Provider now shown as project name with 📁 icon
	if !strings.Contains(view, "anthropic") {
		t.Errorf("View() missing provider/project 'anthropic'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsGitBranch(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.GitBranch = "tui-migration"
	view := m.View()
	if !strings.Contains(view, "tui-migration") {
		t.Errorf("View() missing branch name; got:\n%s", view)
	}
}

func TestStatusLineViewContainsAuthStatus(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.AuthStatus = "authenticated"
	view := m.View()
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
	_ = newModel.View()
}

func TestStatusLineUpdateUnknownMsg(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update("unknown")
	assert.Nil(t, cmd, "unknown message should return nil command")
	_ = newModel.View()
}

// ---------------------------------------------------------------------------
// SetWidth
// ---------------------------------------------------------------------------

func TestStatusLineSetWidth(t *testing.T) {
	m := statusline.NewStatusLineModel(80)
	m.SetWidth(200)
	_ = m.View()
}

// ---------------------------------------------------------------------------
// All fields populated
// ---------------------------------------------------------------------------

func TestStatusLineAllFieldsPopulated(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	m.SessionCost = 0.0042
	m.TokenCount = 8500
	m.ContextPercent = 12.3
	m.ContextCapacity = 200_000
	m.ContextUsedTokens = 24_600
	m.PermissionMode = "bypass"
	m.ActiveModel = "claude-opus-4-6"
	m.Provider = "anthropic"
	m.GitBranch = "main"
	m.AuthStatus = "ok"

	view := m.View()

	// New layout uses badges and different formatting
	expected := []string{
		"$0.0042",          // cost value
		"12%",              // context percent
		"24.6K/200K",       // token count / capacity
		"[bypass]",         // permission badge
		"[claude-opus-4-6]", // model badge
		"anthropic",        // project/provider
		"main",             // git branch
		"ok",               // auth status
	}
	for _, want := range expected {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q; got:\n%s", want, view)
		}
	}
}

// ---------------------------------------------------------------------------
// Git branch / auth messages
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
// Tick messages
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

func TestStartTicks_ReturnsBatchCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.StartTicks()
	assert.NotNil(t, cmd, "StartTicks() should return a non-nil batch command")
}

func TestGitBranchCmdExecution(t *testing.T) {
	msg := statusline.ExecuteGitBranchCmdForTest()
	assert.NotNil(t, msg, "gitBranchCmd() should return a non-nil message")
}

func TestAuthStatusCmdExecution(t *testing.T) {
	msg := statusline.ExecuteAuthStatusCmdForTest()
	assert.NotNil(t, msg, "authStatusCmd() should return a non-nil message")
}

func TestScheduleTicksCmdExecution(t *testing.T) {
	gitCmd := statusline.ScheduleGitBranchTickForTest()
	assert.NotNil(t, gitCmd, "scheduleGitBranchTick() should return a non-nil command")

	authCmd := statusline.ScheduleAuthStatusTickForTest()
	assert.NotNil(t, authCmd, "scheduleAuthStatusTick() should return a non-nil command")
}

// ---------------------------------------------------------------------------
// FormatCost via View
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
// formatTokens
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
// Session timer + spinner ticks
// ---------------------------------------------------------------------------

func TestUpdate_SessionTimerTickMsg_ReturnsCmd(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	_, cmd := m.Update(statusline.SessionTimerTickMsgForTest(time.Now()))
	assert.NotNil(t, cmd, "sessionTimerTickMsg should return a non-nil command")
}

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
// SetStreaming
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
// Streaming indicator
// ---------------------------------------------------------------------------

func TestView_StreamingIndicator_WhenStreaming(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.Streaming = true
	view := m.View()
	assert.Contains(t, view, "streaming", "View() should show 'streaming' when Streaming")
}

func TestView_NoStreamingIndicator_WhenNotStreaming(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.Streaming = false
	view := m.View()
	assert.NotContains(t, view, "streaming", "View() should not show 'streaming' when not Streaming")
}

// ---------------------------------------------------------------------------
// Elapsed time
// ---------------------------------------------------------------------------

func TestView_ShowsElapsedTime_WhenSessionStartSet(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.SessionStart = time.Now().Add(-90 * time.Second) // 1m 30s ago
	view := m.View()
	assert.Contains(t, view, "1m", "View() should show elapsed minutes")
}

func TestView_NoElapsedTime_WhenSessionStartZero(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	view := m.View()
	assert.NotContains(t, view, "⏱", "View() should not show elapsed timer before session starts")
}

// ---------------------------------------------------------------------------
// parseAuthStatus
// ---------------------------------------------------------------------------

func TestParseAuthStatus(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		wantC string
	}{
		{name: "empty string", raw: "", wantC: "N/A"},
		{name: "email and method", raw: "Logged in via claude.ai\nAccount: admin@exactmass.org", wantC: "claude.ai"},
		{name: "email present", raw: "admin@exactmass.org", wantC: "admin@exactmass.org"},
		{name: "fallback to first line", raw: "Authenticated", wantC: "Authenticated"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statusline.ParseAuthStatusForTest(tc.raw)
			assert.Contains(t, got, tc.wantC, "parseAuthStatus(%q)", tc.raw)
		})
	}
}

// ---------------------------------------------------------------------------
// StartTicks includes session timer
// ---------------------------------------------------------------------------

func TestStartTicks_IncludesSessionTimer(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.StartTicks()
	assert.NotNil(t, cmd)
}

func TestScheduleSessionTimerTick_ReturnsCmd(t *testing.T) {
	cmd := statusline.ScheduleSessionTimerTickForTest()
	assert.NotNil(t, cmd)
}

func TestScheduleSpinnerTick_ReturnsCmd(t *testing.T) {
	cmd := statusline.ScheduleSpinnerTickForTest()
	assert.NotNil(t, cmd)
}

// ---------------------------------------------------------------------------
// Semantic color helpers
// ---------------------------------------------------------------------------

func TestCostStyle_Thresholds(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	theme := config.DefaultTheme()
	m.SetTheme(theme)

	tests := []struct {
		name      string
		cost      float64
		wantStyle string
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
				assert.Equal(t, successStyle, got)
			case "warning":
				assert.Equal(t, warningStyle, got)
			case "error":
				assert.Equal(t, errorStyle, got)
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
		{"at warning threshold", 70, "warning"},
		{"below error threshold", 89, "warning"},
		{"at error threshold", 90, "error"},
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
				assert.Equal(t, successStyle, got)
			case "warning":
				assert.Equal(t, warningStyle, got)
			case "error":
				assert.Equal(t, errorStyle, got)
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
				assert.Equal(t, successStyle, got)
			case "warning":
				assert.Equal(t, warningStyle, got)
			case "error":
				assert.Equal(t, errorStyle, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetTheme
// ---------------------------------------------------------------------------

func TestSetTheme_UpdatesTheme(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	lightTheme := config.NewTheme(config.ThemeLight)
	m.SetTheme(lightTheme)
	got := m.CostStyleForTest(0.01)
	assert.Equal(t, lightTheme.SuccessStyle(), got)
}

// ---------------------------------------------------------------------------
// UncommittedCount message
// ---------------------------------------------------------------------------

func TestUncommittedCountMsg_Updates(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	assert.Equal(t, 0, m.UncommittedCount)

	newModel, cmd := m.Update(statusline.UncommittedCountMsgForTest(7))
	assert.Nil(t, cmd)
	assert.Equal(t, 7, newModel.UncommittedCount)
}

func TestUncommittedCountCmd_Execution(t *testing.T) {
	msg := statusline.ExecuteUncommittedCountCmdForTest()
	assert.NotNil(t, msg)
}

// ---------------------------------------------------------------------------
// Agent count and uncommitted in view
// ---------------------------------------------------------------------------

func TestView_IncludesAgentCount(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.AgentCount = 3
	view := m.View()
	assert.Contains(t, view, "agents:3", "View() should include 'agents:3'")
}

func TestView_IncludesUncommittedCount_WhenPositive(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.UncommittedCount = 5
	m.GitBranch = "main"
	view := m.View()
	assert.Contains(t, view, "~5", "View() should include '~5' for uncommitted files")
}

func TestView_ExcludesUncommittedCount_WhenZero(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.UncommittedCount = 0
	m.GitBranch = "main"
	view := m.View()
	assert.NotContains(t, view, "~0", "View() should not show '~0' when uncommitted is 0")
}

// ---------------------------------------------------------------------------
// Two rows preserved
// ---------------------------------------------------------------------------

func TestView_TwoRowsWithNewFields(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	m.AgentCount = 2
	m.UncommittedCount = 4
	m.SessionCost = 1.50
	m.ContextPercent = 85
	m.PermissionMode = "allow-all"
	m.GitBranch = "main"
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	assert.Equal(t, 2, len(lines), "View() should still produce exactly 2 rows")
}

// ---------------------------------------------------------------------------
// Context bar
// ---------------------------------------------------------------------------

func TestRenderContextBar_ZeroPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 0
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 0
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "0%")
	assert.Contains(t, result, "0/1M")
}

func TestRenderContextBar_25Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 25
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 250_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "25%")
	assert.Contains(t, result, "250K/1M")
	assert.Contains(t, result, "▓")
}

func TestRenderContextBar_50Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 50
	m.ContextCapacity = 200_000
	m.ContextUsedTokens = 100_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "50%")
	assert.Contains(t, result, "100K/200K")
}

func TestRenderContextBar_75Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 75
	m.ContextCapacity = 200_000
	m.ContextUsedTokens = 150_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "75%")
	assert.Contains(t, result, "150K/200K")
}

func TestRenderContextBar_100Percent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 100
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 1_000_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "100%")
	assert.Contains(t, result, "▓▓▓▓▓▓▓▓▓▓")
	assert.Contains(t, result, "1M/1M")
}

func TestRenderContextBar_NarrowFallback(t *testing.T) {
	m := statusline.NewStatusLineModel(79)
	m.ContextPercent = 52
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 520_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "52%")
	assert.Contains(t, result, "520K/1M")
	assert.NotContains(t, result, "▓")
}

func TestRenderContextBar_NegativePercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = -10
	m.ContextCapacity = 200_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "0%")
}

func TestRenderContextBar_OverPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 150
	m.ContextCapacity = 200_000
	m.ContextUsedTokens = 300_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "100%")
	assert.Contains(t, result, "▓▓▓▓▓▓▓▓▓▓")
}

func TestRenderContextBar_ContainsBarChars(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 60
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 600_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "▓")
	assert.Contains(t, result, "░")
	assert.Contains(t, result, "600K/1M")
}

func TestRenderContextBar_NoCapacity(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 0
	m.ContextCapacity = 0
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "░░░░░░░░░░")
	assert.Contains(t, result, "—")
}

func TestView_ContextBar_Integration(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 40
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 400_000
	view := m.View()
	assert.Contains(t, view, "40%")
	assert.Contains(t, view, "400K/1M")
}

// ---------------------------------------------------------------------------
// Plan mode
// ---------------------------------------------------------------------------

func TestView_PlanMode_Inactive_NoLabel(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	view := m.View()
	assert.NotContains(t, view, "[PLAN", "inactive plan mode must not show [PLAN] label")
}

func TestView_PlanMode_Active_StepsUnknown(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	statusline.SetPlanFieldsForTest(&m, true, 0, 0)
	view := m.View()
	assert.Contains(t, view, "[PLAN]", "active plan mode with unknown steps must show [PLAN]")
}

func TestView_PlanMode_Active_StepsKnown(t *testing.T) {
	tests := []struct {
		name    string
		step    int
		total   int
		wantSub string
	}{
		{"first step", 1, 5, "[PLAN 1/5]"},
		{"middle step", 2, 5, "[PLAN 2/5]"},
		{"last step", 5, 5, "[PLAN 5/5]"},
		{"single step plan", 1, 1, "[PLAN 1/1]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := statusline.NewStatusLineModel(200)
			statusline.SetPlanFieldsForTest(&m, true, tc.step, tc.total)
			view := m.View()
			assert.Contains(t, view, tc.wantSub)
		})
	}
}

func TestView_PlanMode_Deactivated_LabelDisappears(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	statusline.SetPlanFieldsForTest(&m, true, 3, 5)
	activeView := m.View()
	assert.Contains(t, activeView, "[PLAN")

	statusline.SetPlanFieldsForTest(&m, false, 0, 0)
	inactiveView := m.View()
	assert.NotContains(t, inactiveView, "[PLAN")
}

func TestView_PlanMode_TwoRowsPreserved(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	statusline.SetPlanFieldsForTest(&m, true, 2, 7)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	assert.Equal(t, 2, len(lines), "View() must still produce exactly 2 rows in plan mode")
}
