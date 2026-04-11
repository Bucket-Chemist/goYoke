// export_test.go exposes unexported types and constructors for use in
// statusline_test (external test package). This file is compiled only during
// testing.
package statusline

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GitBranchMsgForTest constructs a gitBranchMsg for use in tests.
func GitBranchMsgForTest(branch string, err error) gitBranchMsg {
	return gitBranchMsg{Branch: branch, Err: err}
}

// AuthStatusMsgForTest constructs an authStatusMsg for use in tests.
func AuthStatusMsgForTest(status string, err error) authStatusMsg {
	return authStatusMsg{Status: status, Err: err}
}

// GitBranchTickMsgForTest constructs a gitBranchTickMsg for use in tests.
func GitBranchTickMsgForTest(t time.Time) gitBranchTickMsg {
	return gitBranchTickMsg(t)
}

// AuthStatusTickMsgForTest constructs an authStatusTickMsg for use in tests.
func AuthStatusTickMsgForTest(t time.Time) authStatusTickMsg {
	return authStatusTickMsg(t)
}

// SessionTimerTickMsgForTest constructs a sessionTimerTickMsg for use in tests.
func SessionTimerTickMsgForTest(t time.Time) sessionTimerTickMsg {
	return sessionTimerTickMsg(t)
}

// SpinnerTickMsgForTest constructs a spinnerTickMsg for use in tests.
func SpinnerTickMsgForTest(t time.Time) spinnerTickMsg {
	return spinnerTickMsg(t)
}

// ExecuteGitBranchCmdForTest calls gitBranchCmd() and executes the returned
// closure, returning the resulting tea.Msg. Used to cover the subprocess path.
func ExecuteGitBranchCmdForTest() tea.Msg {
	cmd := gitBranchCmd()
	return cmd()
}

// ExecuteAuthStatusCmdForTest calls authStatusCmd() and executes the returned
// closure, returning the resulting tea.Msg.
func ExecuteAuthStatusCmdForTest() tea.Msg {
	cmd := authStatusCmd()
	return cmd()
}

// ScheduleGitBranchTickForTest returns the tea.Cmd from scheduleGitBranchTick.
func ScheduleGitBranchTickForTest() tea.Cmd {
	return scheduleGitBranchTick()
}

// ScheduleAuthStatusTickForTest returns the tea.Cmd from scheduleAuthStatusTick.
func ScheduleAuthStatusTickForTest() tea.Cmd {
	return scheduleAuthStatusTick()
}

// ScheduleSessionTimerTickForTest returns the tea.Cmd from scheduleSessionTimerTick.
func ScheduleSessionTimerTickForTest() tea.Cmd {
	return scheduleSessionTimerTick()
}

// ScheduleSpinnerTickForTest returns the tea.Cmd from scheduleSpinnerTick.
func ScheduleSpinnerTickForTest() tea.Cmd {
	return scheduleSpinnerTick()
}

// ParseAuthStatusForTest exposes parseAuthStatus for unit testing.
func ParseAuthStatusForTest(raw string) string {
	return parseAuthStatus(raw)
}

// FormatTokensForTest exposes formatTokens for unit testing.
func FormatTokensForTest(n int) string {
	return formatTokens(n)
}

// UncommittedCountMsgForTest constructs an uncommittedCountMsg for use in tests.
func UncommittedCountMsgForTest(n int) uncommittedCountMsg {
	return uncommittedCountMsg(n)
}

// ExecuteUncommittedCountCmdForTest calls uncommittedCountCmd() and executes
// the returned closure, returning the resulting tea.Msg.
func ExecuteUncommittedCountCmdForTest() tea.Msg {
	cmd := uncommittedCountCmd()
	return cmd()
}

// CostStyleForTest exposes costStyle for unit testing.
func (m StatusLineModel) CostStyleForTest(cost float64) lipgloss.Style {
	return m.costStyle(cost)
}

// ContextStyleForTest exposes contextStyle for unit testing.
func (m StatusLineModel) ContextStyleForTest(pct float64) lipgloss.Style {
	return m.contextStyle(pct)
}

// PermStyleForTest exposes permStyle for unit testing.
func (m StatusLineModel) PermStyleForTest(mode string) lipgloss.Style {
	return m.permStyle(mode)
}

// RenderContextBarForTest exposes renderContextBar for unit testing.
func (m StatusLineModel) RenderContextBarForTest() string {
	return m.renderContextBar()
}

// PlanActiveMsgForTest constructs the field values used to exercise plan mode
// rendering without requiring an import of the model package.
// Call SetPlanFields on a StatusLineModel then call View() to observe output.
func SetPlanFieldsForTest(m *StatusLineModel, active bool, step, total int) {
	m.PlanActive = active
	m.PlanStep = step
	m.PlanTotalSteps = total
}

// RenderTeamIndicatorForTest exposes renderTeamIndicator for direct unit testing.
func (m StatusLineModel) RenderTeamIndicatorForTest() string {
	return m.renderTeamIndicator()
}

// SetTeamFieldsForTest sets all team-related fields on a StatusLineModel for testing.
func SetTeamFieldsForTest(m *StatusLineModel, active bool, name string, statuses []string, wave, total int, cost float64) {
	m.TeamActive = active
	m.TeamName = name
	m.TeamMemberStatuses = statuses
	m.TeamCurrentWave = wave
	m.TeamTotalWaves = total
	m.TeamCost = cost
}

// RenderAgentSparklineForTest exposes renderAgentSparkline for direct unit testing.
func (m StatusLineModel) RenderAgentSparklineForTest() string {
	return m.renderAgentSparkline()
}

// SpinnerIdxForTest returns the internal spinner frame index for testing.
func (m StatusLineModel) SpinnerIdxForTest() int {
	return m.spinnerIdx
}
