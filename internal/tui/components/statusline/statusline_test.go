package statusline_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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

	// costStyle thresholds: green <$1, yellow $1–$5, red >$5 (all bold).
	tests := []struct {
		name      string
		cost      float64
		wantStyle string
	}{
		{"zero cost", 0.00, "success"},
		{"below $1 threshold", 0.99, "success"},
		{"at $1 threshold", 1.00, "warning"},
		{"between $1 and $5", 2.50, "warning"},
		{"just below $5", 4.99, "warning"},
		{"at $5 threshold", 5.00, "error"},
		{"above $5 threshold", 10.00, "error"},
	}

	// costStyle always adds Bold(true); compare against bolded base styles.
	successStyle := theme.SuccessStyle().Bold(true)
	warningStyle := theme.WarningStyle().Bold(true)
	errorStyle := theme.ErrorStyle().Bold(true)

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
	assert.Equal(t, lightTheme.SuccessStyle().Bold(true), got)
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
	m.AgentStats = state.AgentStats{Total: 3, Running: 3}
	view := m.View()
	assert.Contains(t, view, "agents: 3/3", "View() should include 'agents: 3/3'")
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
	m.AgentStats = state.AgentStats{Total: 2, Running: 2}
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
	assert.Contains(t, result, "█")
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
	assert.Contains(t, result, "██████████")
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
	assert.NotContains(t, result, "█")
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
	assert.Contains(t, result, "██████████")
}

func TestRenderContextBar_ContainsBarChars(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 60
	m.ContextCapacity = 1_000_000
	m.ContextUsedTokens = 600_000
	result := m.RenderContextBarForTest()
	assert.Contains(t, result, "█")
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

// ---------------------------------------------------------------------------
// Team indicator
// ---------------------------------------------------------------------------

func TestView_TeamIndicator_Hidden_WhenInactive(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	m.TeamActive = false
	view := m.View()
	assert.NotContains(t, view, "⚡", "View() must not show team indicator when TeamActive is false")
}

func TestView_TeamIndicator_Visible_WhenActive(t *testing.T) {
	m := statusline.NewStatusLineModel(300)
	statusline.SetTeamFieldsForTest(&m, true, "impl", []string{"running", "complete", "pending", "pending"}, 2, 4, 0.38)
	view := m.View()
	assert.Contains(t, view, "⚡", "View() must show ⚡ prefix when team is active")
	assert.Contains(t, view, "impl", "View() must show team name")
	assert.Contains(t, view, "2/4", "View() must show wave progress")
	assert.Contains(t, view, "$0.38", "View() must show team cost")
}

func TestView_TeamIndicator_MemberDots_Colors(t *testing.T) {
	m := statusline.NewStatusLineModel(300)
	// One member per distinct status to verify dot characters are rendered.
	statuses := []string{"running", "complete", "pending", "failed", "error", "skipped", "killed"}
	statusline.SetTeamFieldsForTest(&m, true, "test", statuses, 1, 1, 0.10)
	view := m.View()
	// Filled dot for active/terminal statuses.
	assert.Contains(t, view, "●", "View() must render filled dots for running/complete/failed/error/skipped/killed")
	// Empty dot for pending.
	assert.Contains(t, view, "○", "View() must render empty dot for pending")
}

func TestView_TeamIndicator_Disappears_WhenDeactivated(t *testing.T) {
	m := statusline.NewStatusLineModel(300)
	statusline.SetTeamFieldsForTest(&m, true, "myteam", []string{"running"}, 1, 3, 0.05)
	activeView := m.View()
	assert.Contains(t, activeView, "⚡", "View() must show team indicator when active")

	statusline.SetTeamFieldsForTest(&m, false, "", nil, 0, 0, 0)
	inactiveView := m.View()
	assert.NotContains(t, inactiveView, "⚡", "View() must hide team indicator after deactivation")
}

func TestRenderTeamIndicator_Empty_WhenInactive(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	m.TeamActive = false
	result := m.RenderTeamIndicatorForTest()
	assert.Equal(t, "", result, "renderTeamIndicator() must return empty string when TeamActive is false")
}

func TestRenderTeamIndicator_ContainsAllParts(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	statusline.SetTeamFieldsForTest(&m, true, "longteamname", []string{"running", "pending"}, 3, 5, 1.25)
	result := m.RenderTeamIndicatorForTest()
	// Name is truncated to 8 runes: "longtea…"
	assert.Contains(t, result, "⚡", "renderTeamIndicator() must contain ⚡ prefix")
	assert.Contains(t, result, "longtea", "renderTeamIndicator() must contain truncated team name")
	assert.Contains(t, result, "3/5", "renderTeamIndicator() must contain wave progress")
	assert.Contains(t, result, "$1.25", "renderTeamIndicator() must contain team cost")
	assert.Contains(t, result, "●", "renderTeamIndicator() must contain filled dot for running member")
	assert.Contains(t, result, "○", "renderTeamIndicator() must contain empty dot for pending member")
}

func TestView_TeamIndicator_TwoRowsPreserved(t *testing.T) {
	m := statusline.NewStatusLineModel(300)
	statusline.SetTeamFieldsForTest(&m, true, "team", []string{"running", "complete"}, 1, 2, 0.15)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	assert.Equal(t, 2, len(lines), "View() must produce exactly 2 rows when team indicator is active")
}

// ---------------------------------------------------------------------------
// Agent sparkline
// ---------------------------------------------------------------------------

func TestRenderAgentSparkline_Empty(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	// Zero AgentStats → empty string
	result := m.RenderAgentSparklineForTest()
	assert.Equal(t, "", result, "renderAgentSparkline() must return empty string when Total==0")
}

func TestRenderAgentSparkline_AllRunning(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.AgentStats = state.AgentStats{Total: 3, Running: 3}
	result := m.RenderAgentSparklineForTest()
	assert.Contains(t, result, "3/3", "renderAgentSparkline() must contain running/total ratio")
	// Three filled dots for three running agents.
	count := strings.Count(result, "●")
	assert.Equal(t, 3, count, "renderAgentSparkline() must render 3 filled dots for 3 running agents")
}

func TestRenderAgentSparkline_MixedStates(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.AgentStats = state.AgentStats{Total: 4, Running: 1, Complete: 1, Pending: 1, Error: 1}
	result := m.RenderAgentSparklineForTest()
	assert.Contains(t, result, "1/4", "renderAgentSparkline() must contain running/total ratio")
	// 3 filled dots (running, complete, error) + 1 empty dot (pending) = 4 characters total.
	filledCount := strings.Count(result, "●")
	emptyCount := strings.Count(result, "○")
	assert.Equal(t, 3, filledCount, "renderAgentSparkline() must render 3 filled dots for running+complete+error")
	assert.Equal(t, 1, emptyCount, "renderAgentSparkline() must render 1 empty dot for pending")
}

func TestRenderAgentSparkline_AllComplete(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.AgentStats = state.AgentStats{Total: 3, Running: 0, Complete: 3}
	result := m.RenderAgentSparklineForTest()
	assert.Contains(t, result, "0/3", "renderAgentSparkline() must contain 0/3 when all complete")
	count := strings.Count(result, "●")
	assert.Equal(t, 3, count, "renderAgentSparkline() must render 3 filled dots for 3 complete agents")
}

// ---------------------------------------------------------------------------
// UX-017: Adaptive height — Height() and boundary tests
// ---------------------------------------------------------------------------

func TestHeight_Compact(t *testing.T) {
	m := statusline.NewStatusLineModel(60)
	assert.Equal(t, 1, m.Height(), "compact tier (<80) must be 1 row")
}

func TestHeight_Standard(t *testing.T) {
	m80 := statusline.NewStatusLineModel(80)
	assert.Equal(t, 1, m80.Height(), "standard tier lower bound (80) must be 1 row")

	m119 := statusline.NewStatusLineModel(119)
	assert.Equal(t, 1, m119.Height(), "standard tier upper bound (119) must be 1 row")
}

func TestHeight_Wide(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	assert.Equal(t, 2, m.Height(), "wide tier lower bound (120) must be 2 rows")
}

func TestHeight_Ultra(t *testing.T) {
	m := statusline.NewStatusLineModel(200)
	assert.Equal(t, 2, m.Height(), "ultra tier (200) must be 2 rows")
}

func TestViewCompact_SingleRow(t *testing.T) {
	m := statusline.NewStatusLineModel(100)
	m.SessionCost = 0.50
	m.ActiveModel = "sonnet"
	view := m.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.Equal(t, 1, len(lines), "compact view (width=100) must be exactly 1 line")
}

func TestViewFull_TwoRows(t *testing.T) {
	m := statusline.NewStatusLineModel(150)
	m.SessionCost = 0.50
	m.ActiveModel = "sonnet"
	view := m.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.Equal(t, 2, len(lines), "full view (width=150) must be exactly 2 lines")
}

// TestView_HeightMatchesRendered is the invariant test: Height() must equal
// lipgloss.Height(View()) at every tier boundary width.
func TestView_HeightMatchesRendered(t *testing.T) {
	for _, w := range []int{79, 80, 119, 120} {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			m := statusline.NewStatusLineModel(w)
			m.SessionCost = 1.23
			m.ActiveModel = "sonnet"
			rendered := m.View()
			assert.Equal(t, m.Height(), lipgloss.Height(rendered),
				"Height() must match actual rendered height at width=%d", w)
		})
	}
}

// ---------------------------------------------------------------------------
// Reduce Motion (UX-020)
// ---------------------------------------------------------------------------

// TestSpinner_ReduceMotion_NoAdvance verifies that spinnerTickMsg does not
// advance the frame index when ReduceMotion is true.
func TestSpinner_ReduceMotion_NoAdvance(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ReduceMotion = true
	m.Streaming = true

	// Confirm starting index is 0.
	if m.SpinnerIdxForTest() != 0 {
		t.Fatalf("precondition: spinnerIdx should be 0, got %d", m.SpinnerIdxForTest())
	}

	tick := statusline.SpinnerTickMsgForTest(time.Now())
	result, _ := m.Update(tick)

	if result.SpinnerIdxForTest() != 0 {
		t.Errorf("spinnerIdx advanced despite ReduceMotion=true: got %d, want 0",
			result.SpinnerIdxForTest())
	}
}

// TestSpinner_ReduceMotion_Off_Advances verifies the normal path still works:
// spinnerTickMsg advances the frame index when ReduceMotion is false.
func TestSpinner_ReduceMotion_Off_Advances(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ReduceMotion = false
	m.Streaming = true

	tick := statusline.SpinnerTickMsgForTest(time.Now())
	result, _ := m.Update(tick)

	if result.SpinnerIdxForTest() != 1 {
		t.Errorf("spinnerIdx did not advance: got %d, want 1", result.SpinnerIdxForTest())
	}
}

// TestView_ReduceMotion_StaticSpinner verifies that when ReduceMotion is true
// and Streaming is true, View renders a static "⠿ streaming" indicator instead
// of an animated Braille frame.
func TestView_ReduceMotion_StaticSpinner(t *testing.T) {
	for _, width := range []int{80, 120, 160} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			m := statusline.NewStatusLineModel(width)
			m.ReduceMotion = true
			m.Streaming = true

			view := m.View()

			if !strings.Contains(view, "⠿ streaming") {
				t.Errorf("expected static '⠿ streaming' indicator in View(), got:\n%s", view)
			}
			// The animated frames cycle through ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏ — confirm none appear
			// immediately after "streaming" (which would indicate animation ran).
			animatedFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			for _, frame := range animatedFrames {
				if strings.Contains(view, frame+" streaming") {
					t.Errorf("animated frame %q should not appear in View() when ReduceMotion=true", frame)
				}
			}
		})
	}
}

// TestView_ReduceMotion_Off_AnimatedSpinner verifies that when ReduceMotion is
// false the normal animated frame is rendered.
func TestView_ReduceMotion_Off_AnimatedSpinner(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ReduceMotion = false
	m.Streaming = true
	// Frame index 0 → first Braille char "⠋".

	view := m.View()

	if !strings.Contains(view, "⠋ streaming") {
		t.Errorf("expected animated '⠋ streaming' in View() when ReduceMotion=false, got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Cost flash — CheckCostFlash / CostFlashExpiredMsg
// ---------------------------------------------------------------------------

// TestCheckCostFlash exercises all branch conditions as a table-driven test.
func TestCheckCostFlash(t *testing.T) {
	flashStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))

	tests := []struct {
		name             string
		prevCost         float64
		newCost          float64
		flashEnabled     bool
		reduceMotion     bool
		wantCmd          bool   // whether a tea.Cmd is returned
		wantFlashActive  bool   // whether costFlashUntil is non-zero
		wantPrevCost     float64
	}{
		{
			name:            "flash triggers when cost increases and enabled",
			prevCost:        0.10,
			newCost:         0.20,
			flashEnabled:    true,
			reduceMotion:    false,
			wantCmd:         true,
			wantFlashActive: true,
			wantPrevCost:    0.20,
		},
		{
			name:            "no flash when disabled (default)",
			prevCost:        0.10,
			newCost:         0.20,
			flashEnabled:    false,
			reduceMotion:    false,
			wantCmd:         false,
			wantFlashActive: false,
			wantPrevCost:    0.20,
		},
		{
			name:            "no flash when reduce-motion enabled",
			prevCost:        0.10,
			newCost:         0.20,
			flashEnabled:    true,
			reduceMotion:    true,
			wantCmd:         false,
			wantFlashActive: false,
			wantPrevCost:    0.20,
		},
		{
			name:            "no flash when cost stays same",
			prevCost:        0.10,
			newCost:         0.10,
			flashEnabled:    true,
			reduceMotion:    false,
			wantCmd:         false,
			wantFlashActive: false,
			wantPrevCost:    0.10,
		},
		{
			name:            "no flash when cost decreases",
			prevCost:        0.50,
			newCost:         0.30,
			flashEnabled:    true,
			reduceMotion:    false,
			wantCmd:         false,
			wantFlashActive: false,
			wantPrevCost:    0.30,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := statusline.NewStatusLineModel(120)
			m.CostFlashEnabled = tc.flashEnabled
			m.ReduceMotion = tc.reduceMotion
			// Seed prevCost directly to avoid triggering flash side-effects.
			m.SetPrevCostForTest(tc.prevCost)

			// Simulate the cost update.
			m.SessionCost = tc.newCost
			cmd := m.CheckCostFlash()

			if tc.wantCmd {
				assert.NotNil(t, cmd, "CheckCostFlash() should return a Cmd when flash triggers")
			} else {
				assert.Nil(t, cmd, "CheckCostFlash() should return nil when flash does not trigger")
			}

			if tc.wantFlashActive {
				assert.False(t, m.CostFlashUntilForTest().IsZero(), "costFlashUntil should be set")
			} else {
				assert.True(t, m.CostFlashUntilForTest().IsZero(), "costFlashUntil should be zero")
			}

			assert.InDelta(t, tc.wantPrevCost, m.PrevCostForTest(), 1e-9, "prevCost mismatch")
		})
	}

	// Verify that while flash is active, activeCostStyle returns bright white bold.
	t.Run("active flash style is bright white bold", func(t *testing.T) {
		m := statusline.NewStatusLineModel(120)
		m.CostFlashEnabled = true
		m.ReduceMotion = false
		m.SetPrevCostForTest(0.10)
		m.SessionCost = 0.20
		_ = m.CheckCostFlash()

		got := m.ActiveCostStyleForTest()
		assert.Equal(t, flashStyle, got, "activeCostStyle() should return bright-white bold during flash")
	})
}

// TestCostFlashExpiredMsg_ClearsCostFlashUntil verifies the expiry message
// resets the flash state in Update().
func TestCostFlashExpiredMsg_ClearsCostFlashUntil(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.CostFlashEnabled = true
	m.SetPrevCostForTest(0.20)
	m.SessionCost = 0.50
	_ = m.CheckCostFlash() // flash is now active

	assert.False(t, m.CostFlashUntilForTest().IsZero(), "flash should be active before expiry")

	newM, cmd := m.Update(statusline.CostFlashExpiredMsgForTest())
	assert.Nil(t, cmd, "CostFlashExpiredMsg should return nil command")
	assert.True(t, newM.CostFlashUntilForTest().IsZero(), "costFlashUntil should be zero after expiry")
}

// TestCostFlashDefault_IsOff verifies the default is off for new models.
func TestCostFlashDefault_IsOff(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	assert.False(t, m.CostFlashEnabled, "CostFlashEnabled should default to false")
}
