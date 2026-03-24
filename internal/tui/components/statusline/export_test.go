// export_test.go exposes unexported types and constructors for use in
// statusline_test (external test package). This file is compiled only during
// testing.
package statusline

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
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
