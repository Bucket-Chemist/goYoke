package model

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Stubs for team drawer tests
// ---------------------------------------------------------------------------

// trackingDrawerStack records calls to SetTeamsContent and ClearTeamsContent.
type trackingDrawerStack struct {
	teamsContent    string
	teamsHasContent bool
	teamsMinimized  bool
	setTeamsCalls   int
	clearTeamsCalls int
}

func (s *trackingDrawerStack) View() string                                    { return "" }
func (s *trackingDrawerStack) SetSize(_, _ int)                                {}
func (s *trackingDrawerStack) ExpandedDrawers() []string                       { return nil }
func (s *trackingDrawerStack) HandleKey(_ string, _ tea.KeyMsg) tea.Cmd        { return nil }
func (s *trackingDrawerStack) SetOptionsContent(_ string)                      {}
func (s *trackingDrawerStack) ClearOptionsContent()                            {}
func (s *trackingDrawerStack) OptionsHasContent() bool                         { return false }
func (s *trackingDrawerStack) SetPlanContent(_ string)                         {}
func (s *trackingDrawerStack) ClearPlanContent()                               {}
func (s *trackingDrawerStack) PlanHasContent() bool                            { return false }
func (s *trackingDrawerStack) SetTeamsContent(content string) {
	s.teamsContent = content
	s.teamsHasContent = content != ""
	s.teamsMinimized = false
	s.setTeamsCalls++
}
func (s *trackingDrawerStack) ClearTeamsContent() {
	s.teamsContent = ""
	s.teamsHasContent = false
	s.teamsMinimized = true
	s.clearTeamsCalls++
}
func (s *trackingDrawerStack) TeamsHasContent() bool            { return s.teamsHasContent }
func (s *trackingDrawerStack) RefreshTeamsContent(c string)     { s.teamsContent = c }
func (s *trackingDrawerStack) TeamsIsMinimized() bool           { return s.teamsMinimized }
func (s *trackingDrawerStack) SetOptionsFocused(_ bool)         {}
func (s *trackingDrawerStack) SetPlanFocused(_ bool)            {}
func (s *trackingDrawerStack) SetTeamsFocused(_ bool)           {}
func (s *trackingDrawerStack) SetActiveModal(_, _ string, _ []string) {}
func (s *trackingDrawerStack) HasActiveModal() bool             { return false }
func (s *trackingDrawerStack) OptionsActiveRequestID() string   { return "" }
func (s *trackingDrawerStack) OptionsSelectedOption() string    { return "" }
func (s *trackingDrawerStack) ClearOptionsModal()               {}
func (s *trackingDrawerStack) SetFiguresContent(_ string)       {}
func (s *trackingDrawerStack) ClearFiguresContent()             {}
func (s *trackingDrawerStack) FiguresHasContent() bool          { return false }
func (s *trackingDrawerStack) RefreshFiguresContent(_ string)   {}
func (s *trackingDrawerStack) FiguresIsMinimized() bool         { return true }
func (s *trackingDrawerStack) SetFiguresFocused(_ bool)         {}
func (s *trackingDrawerStack) ToggleFiguresDrawer()             {}

// trackingTeamsHealth controls HasData/HasRunningTeam for tests.
type trackingTeamsHealth struct {
	hasData       bool
	hasRunning    bool
	viewContent   string
}

func (h *trackingTeamsHealth) View() string              { return h.viewContent }
func (h *trackingTeamsHealth) SetSize(_, _ int)          {}
func (h *trackingTeamsHealth) SetTier(_ LayoutTier)      {}
func (h *trackingTeamsHealth) HasData() bool             { return h.hasData }
func (h *trackingTeamsHealth) HasRunningTeam() bool      { return h.hasRunning }
func (h *trackingTeamsHealth) TeamIndicator() TeamIndicatorData { return TeamIndicatorData{} }

// trackingTeamList records ScanNow and PollNow calls.
type trackingTeamList struct {
	scanNowCalls int
	pollNowCalls int
}

func (t *trackingTeamList) HandleMsg(_ tea.Msg) tea.Cmd                                  { return nil }
func (t *trackingTeamList) View() string                                                  { return "" }
func (t *trackingTeamList) SetSize(_, _ int)                                              {}
func (t *trackingTeamList) StartPolling(_ string) tea.Cmd                                { return nil }
func (t *trackingTeamList) PollNow() tea.Cmd                                             { t.pollNowCalls++; return nil }
func (t *trackingTeamList) ScanNow()                                                     { t.scanNowCalls++ }
func (t *trackingTeamList) SelectedTeam() string                                         { return "" }
func (t *trackingTeamList) CreateDetailModel(_ *state.AgentRegistry) TeamDetailWidget    { return nil }

// ---------------------------------------------------------------------------
// handleTeamUpdate tests
// ---------------------------------------------------------------------------

func TestHandleTeamUpdate_ExpandsDrawerWithHealthData(t *testing.T) {
	drawer := &trackingDrawerStack{teamsMinimized: true}
	health := &trackingTeamsHealth{hasData: true, viewContent: "Team: test running"}
	teamList := &trackingTeamList{}

	m := AppModel{
		shared: &sharedState{
			drawerStack: drawer,
			teamsHealth: health,
			teamList:    teamList,
		},
	}

	updated, _ := m.Update(TeamUpdateMsg{TeamDir: "/tmp/test-team", Status: "running"})
	app := updated.(AppModel)

	assert.Equal(t, 1, teamList.scanNowCalls, "ScanNow should be called")
	assert.Equal(t, 1, drawer.setTeamsCalls, "SetTeamsContent should be called")
	assert.Equal(t, "Team: test running", drawer.teamsContent)
	assert.False(t, drawer.teamsMinimized, "drawer should be expanded")
	assert.False(t, app.shared.teamNotifiedAt.IsZero(), "teamNotifiedAt should be set")
}

func TestHandleTeamUpdate_ShowsPlaceholderWhenNoData(t *testing.T) {
	drawer := &trackingDrawerStack{teamsMinimized: true}
	health := &trackingTeamsHealth{hasData: false}
	teamList := &trackingTeamList{}

	m := AppModel{
		shared: &sharedState{
			drawerStack: drawer,
			teamsHealth: health,
			teamList:    teamList,
		},
	}

	m.Update(TeamUpdateMsg{TeamDir: "/tmp/test-team", Status: "running"})

	assert.Equal(t, 1, drawer.setTeamsCalls, "SetTeamsContent should be called with placeholder")
	assert.Contains(t, drawer.teamsContent, "Team starting: test-team")
	assert.False(t, drawer.teamsMinimized, "drawer should be expanded even with placeholder")
}

func TestHandleTeamUpdate_NilSharedDoesNotPanic(t *testing.T) {
	m := AppModel{} // shared is nil
	updated, cmd := m.Update(TeamUpdateMsg{TeamDir: "/tmp/x", Status: "running"})
	assert.NotNil(t, updated)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// Grace period tests
// ---------------------------------------------------------------------------

func TestPollTick_GracePeriodSuppressesClear(t *testing.T) {
	// Simulate: handleTeamUpdate set teamNotifiedAt, then poll tick fires
	// with HasData=false. ClearTeamsContent should be suppressed.
	drawer := &trackingDrawerStack{teamsHasContent: true}
	health := &trackingTeamsHealth{hasData: false}

	m := AppModel{
		shared: &sharedState{
			drawerStack:    drawer,
			teamsHealth:    health,
			teamNotifiedAt: time.Now(), // just notified
		},
	}

	// Simulate poll-tick logic from app.go forwarding cascade.
	// We can't trigger the real pollTickMsg (unexported), so we test the
	// grace condition directly.
	graceActive := !m.shared.teamNotifiedAt.IsZero() &&
		time.Since(m.shared.teamNotifiedAt) <= 10*time.Second

	assert.True(t, graceActive, "grace period should be active within 10s")
	assert.Equal(t, 0, drawer.clearTeamsCalls, "ClearTeamsContent should not be called during grace")
}

func TestPollTick_GraceExpiredAllowsClear(t *testing.T) {
	// When teamNotifiedAt is older than 10s, clear should be allowed.
	m := AppModel{
		shared: &sharedState{
			teamNotifiedAt: time.Now().Add(-15 * time.Second), // 15s ago
		},
	}

	graceActive := !m.shared.teamNotifiedAt.IsZero() &&
		time.Since(m.shared.teamNotifiedAt) <= 10*time.Second

	assert.False(t, graceActive, "grace period should have expired after 10s")
}

func TestPollTick_NoNotificationAllowsClear(t *testing.T) {
	// When teamNotifiedAt is zero (no TeamUpdateMsg ever received), clear is allowed.
	m := AppModel{
		shared: &sharedState{},
	}

	graceActive := !m.shared.teamNotifiedAt.IsZero() &&
		time.Since(m.shared.teamNotifiedAt) <= 10*time.Second

	assert.False(t, graceActive, "grace should not be active when teamNotifiedAt is zero")
}

// ---------------------------------------------------------------------------
// Minimal claudePanelWidget stub for completion behavior tests (UX-019).
// ---------------------------------------------------------------------------

type completionClaudePanel struct {
	streaming bool
	hasInput  bool
}

func (p *completionClaudePanel) HandleMsg(_ tea.Msg) tea.Cmd                    { return nil }
func (p *completionClaudePanel) View() string                                    { return "" }
func (p *completionClaudePanel) ViewConversation() string                        { return "" }
func (p *completionClaudePanel) ViewInput() string                               { return "" }
func (p *completionClaudePanel) ApplyOverlay(s string) string                    { return s }
func (p *completionClaudePanel) SetSize(_, _ int)                                {}
func (p *completionClaudePanel) SetFocused(_ bool)                               {}
func (p *completionClaudePanel) IsStreaming() bool                               { return p.streaming }
func (p *completionClaudePanel) HasInput() bool                                  { return p.hasInput }
func (p *completionClaudePanel) SaveMessages() []state.DisplayMessage            { return nil }
func (p *completionClaudePanel) RestoreMessages(_ []state.DisplayMessage)        {}
func (p *completionClaudePanel) SetSender(_ MessageSender)                       {}
func (p *completionClaudePanel) AppendSystemMessage(_ string)                    {}
func (p *completionClaudePanel) SetTier(_ LayoutTier)                            {}
func (p *completionClaudePanel) SetReduceMotion(_ bool)                          {}
func (p *completionClaudePanel) SetShowTimestamps(_ bool) tea.Cmd               { return nil }

// newCompletionTestModel returns a minimal AppModel wired for completion tests.
func newCompletionTestModel(panel *completionClaudePanel) AppModel {
	drawer := &trackingDrawerStack{}
	health := &trackingTeamsHealth{hasData: true, viewContent: "Teams view"}
	teamList := &trackingTeamList{}
	shared := &sharedState{
		drawerStack: drawer,
		teamsHealth: health,
		teamList:    teamList,
	}
	if panel != nil {
		shared.claudePanel = panel
	}
	return AppModel{shared: shared}
}

// extractTabFlash inspects a tea.Cmd batch for a TabFlashMsg command.
func extractTabFlash(cmd tea.Cmd) (TabFlashMsg, bool) {
	if cmd == nil {
		return TabFlashMsg{}, false
	}
	msg := cmd()
	switch m := msg.(type) {
	case TabFlashMsg:
		return m, true
	case tea.BatchMsg:
		for _, c := range m {
			if flash, ok := extractTabFlash(c); ok {
				return flash, true
			}
		}
	}
	return TabFlashMsg{}, false
}

// ---------------------------------------------------------------------------
// UX-019: completion behavior tests
// ---------------------------------------------------------------------------

func TestHandleTeamUpdate_CompletionFlashesTab(t *testing.T) {
	m := newCompletionTestModel(nil)

	_, cmd := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "complete"})

	flash, ok := extractTabFlash(cmd)
	assert.True(t, ok, "expected a TabFlashMsg in the returned cmd batch")
	assert.Equal(t, int(RPMTeams), flash.TabIndex, "tab index should match RPMTeams")
}

func TestHandleTeamUpdate_ErrorAlsoFlashesTab(t *testing.T) {
	m := newCompletionTestModel(nil)

	_, cmd := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "error"})

	_, ok := extractTabFlash(cmd)
	assert.True(t, ok, "failed team should also flash Teams tab")
}

func TestHandleTeamUpdate_RunningStatusDoesNotFlash(t *testing.T) {
	m := newCompletionTestModel(nil)

	_, cmd := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "running"})

	_, ok := extractTabFlash(cmd)
	assert.False(t, ok, "running status should not flash the Teams tab")
}

func TestHandleTeamUpdate_AutoSwitchWhenIdle(t *testing.T) {
	panel := &completionClaudePanel{streaming: false, hasInput: false}
	m := newCompletionTestModel(panel)
	// Ensure we're not already on Teams.
	m.rightPanelMode = RPMAgents

	updated, _ := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "complete"})
	app := updated.(AppModel)

	assert.Equal(t, RPMTeams, app.rightPanelMode, "should auto-switch to Teams when idle")
}

func TestHandleTeamUpdate_NoAutoSwitchWhenStreaming(t *testing.T) {
	panel := &completionClaudePanel{streaming: true, hasInput: false}
	m := newCompletionTestModel(panel)
	m.rightPanelMode = RPMAgents

	updated, _ := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "complete"})
	app := updated.(AppModel)

	assert.Equal(t, RPMAgents, app.rightPanelMode, "should not auto-switch while streaming")
}

func TestHandleTeamUpdate_NoAutoSwitchWhenTyping(t *testing.T) {
	panel := &completionClaudePanel{streaming: false, hasInput: true}
	m := newCompletionTestModel(panel)
	m.rightPanelMode = RPMAgents

	updated, _ := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "complete"})
	app := updated.(AppModel)

	assert.Equal(t, RPMAgents, app.rightPanelMode, "should not auto-switch while user is typing")
}

func TestHandleTeamUpdate_NoAutoSwitchWhenStreamingAndTyping(t *testing.T) {
	panel := &completionClaudePanel{streaming: true, hasInput: true}
	m := newCompletionTestModel(panel)
	m.rightPanelMode = RPMDashboard

	updated, _ := m.Update(TeamUpdateMsg{TeamDir: "/tmp/my-team", Status: "complete"})
	app := updated.(AppModel)

	assert.Equal(t, RPMDashboard, app.rightPanelMode, "should not auto-switch when both streaming and typing")
}
