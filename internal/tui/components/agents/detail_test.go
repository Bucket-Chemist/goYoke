package agents_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fullyPopulatedAgent returns a state.Agent with every field set for thorough
// detail-pane testing.
func fullyPopulatedAgent() *state.Agent {
	return &state.Agent{
		ID:          "agent-xyz",
		ParentID:    "",
		AgentType:   "go-pro",
		Description: "Implement OAuth flow",
		Model:       "sonnet",
		Tier:        "sonnet",
		Status:      state.StatusRunning,
		Activity: &state.AgentActivity{
			Type:      "tool_use",
			Target:    "Read",
			Preview:   "Read /internal/auth/handler.go",
			Timestamp: time.Now(),
		},
		StartedAt:   time.Now().Add(-2*time.Minute - 15*time.Second),
		Duration:    0,
		Cost:        0.045,
		Tokens:      12450,
		Children:    []string{},
		Conventions: []string{"go.md", "go-bubbletea.md"},
		Prompt:      "AGENT: go-pro\n\nTASK: Implement OAuth flow",
	}
}

func errorAgent() *state.Agent {
	a := fullyPopulatedAgent()
	a.Status = state.StatusError
	a.ErrorOutput = "panic: runtime error: index out of range [5] with length 3\ngoroutine 1 [running]"
	a.Duration = 45 * time.Second
	a.StartedAt = time.Time{} // zero to force Duration path
	return a
}

func completeAgent() *state.Agent {
	a := fullyPopulatedAgent()
	a.Status = state.StatusComplete
	a.Duration = 2*time.Minute + 15*time.Second
	return a
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewAgentDetailModel(t *testing.T) {
	m := agents.NewAgentDetailModel()
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

func TestView_EmptyStateDetail(t *testing.T) {
	m := agents.NewAgentDetailModel()
	view := m.View()
	if !strings.Contains(view, "Select an agent") {
		t.Errorf("empty detail View() should contain 'Select an agent'; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// SetAgent
// ---------------------------------------------------------------------------

func TestSetAgent_ShowsAgentType(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "go-pro") {
		t.Errorf("View() should contain AgentType 'go-pro'; got:\n%s", view)
	}
}

func TestSetAgent_ShowsModel(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "sonnet") {
		t.Errorf("View() should contain Model 'sonnet'; got:\n%s", view)
	}
}

func TestSetAgent_ShowsTier(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)
	view := m.View()
	// Tier label
	if !strings.Contains(view, "Tier") {
		t.Errorf("View() should contain 'Tier' label; got:\n%s", view)
	}
}

func TestSetAgent_ShowsCost(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	a.Cost = 0.045
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "0.045") {
		t.Errorf("View() should contain cost '0.045'; got:\n%s", view)
	}
}

func TestSetAgent_ShowsTokens(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	a.Tokens = 12450
	m.SetAgent(a)
	view := m.View()
	// Formatted as 12,450
	if !strings.Contains(view, "12,450") {
		t.Errorf("View() should contain tokens '12,450'; got:\n%s", view)
	}
}

func TestSetAgent_ShowsActivity(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "Read /internal/auth/handler.go") {
		t.Errorf("View() should contain activity preview; got:\n%s", view)
	}
}

func TestSetAgent_ShowsStatus(t *testing.T) {
	tests := []struct {
		status state.AgentStatus
		want   string
	}{
		{state.StatusRunning, "Running"},
		{state.StatusComplete, "Complete"},
		{state.StatusError, "Error"},
		{state.StatusPending, "Pending"},
		{state.StatusKilled, "Killed"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			m := agents.NewAgentDetailModel()
			m.SetSize(80, 40)
			a := fullyPopulatedAgent()
			a.Status = tc.status
			m.SetAgent(a)
			view := m.View()
			if !strings.Contains(view, tc.want) {
				t.Errorf("View() should contain status %q; got:\n%s", tc.want, view)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error output
// ---------------------------------------------------------------------------

func TestSetAgent_ErrorOutputDisplayed(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := errorAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "index out of range") {
		t.Errorf("error agent View() should contain error output; got:\n%s", view)
	}
}

func TestSetAgent_ErrorOutputNotShownForRunning(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent() // StatusRunning
	a.ErrorOutput = "should not appear"
	m.SetAgent(a)
	view := m.View()
	if strings.Contains(view, "should not appear") {
		t.Errorf("running agent View() should NOT show ErrorOutput; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Duration formatting
// ---------------------------------------------------------------------------

func TestView_DurationRunning(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent() // StatusRunning, StartedAt ~2m15s ago
	m.SetAgent(a)
	view := m.View()
	// Should show something with minutes and seconds (e.g. "2m 15s")
	if !strings.Contains(view, "m ") && !strings.Contains(view, "s") {
		t.Errorf("running agent View() should contain duration; got:\n%s", view)
	}
}

func TestView_DurationComplete(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := completeAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "2m") {
		t.Errorf("complete agent View() should contain '2m' duration; got:\n%s", view)
	}
}

func TestView_DurationPendingShowsDash(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	a.Status = state.StatusPending
	a.StartedAt = time.Time{} // zero
	a.Duration = 0
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "—") {
		t.Errorf("pending agent View() should contain '—' for duration; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Update is a no-op
// ---------------------------------------------------------------------------

func TestUpdate_NoOp(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetAgent(fullyPopulatedAgent())

	newM, cmd := m.Update(nil)
	if cmd != nil {
		t.Error("Update() should return nil command")
	}
	_ = newM
}

// ---------------------------------------------------------------------------
// SetAgent nil clears the detail
// ---------------------------------------------------------------------------

func TestSetAgent_Nil_ShowsEmptyState(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetAgent(fullyPopulatedAgent())
	m.SetAgent(nil)
	view := m.View()
	if !strings.Contains(view, "Select an agent") {
		t.Errorf("View() after SetAgent(nil) should show empty state; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Token formatting edge cases
// ---------------------------------------------------------------------------

func TestFormatTokens_SmallNumber(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	a.Tokens = 999
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "999") {
		t.Errorf("View() should contain '999' tokens; got:\n%s", view)
	}
}

func TestFormatTokens_MillionPlus(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	a := fullyPopulatedAgent()
	a.Tokens = 1234567
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "1,234,567") {
		t.Errorf("View() should contain '1,234,567' tokens; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestSetSize_DoesNotPanic(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(0, 0)
	m.SetAgent(fullyPopulatedAgent())
	// Should not panic when width is 0.
	_ = m.View()
}

// ---------------------------------------------------------------------------
// Field labels presence
// ---------------------------------------------------------------------------

func TestView_AllFieldLabelsPresent(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	m.SetAgent(fullyPopulatedAgent())
	view := m.View()

	labels := []string{"Status", "Type", "Model", "Tier", "Duration", "Cost", "Tokens"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("View() missing field label %q; got:\n%s", label, view)
		}
	}
}

// ---------------------------------------------------------------------------
// Collapsible sections
// ---------------------------------------------------------------------------

func TestView_SectionHeaders(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	m.SetAgent(fullyPopulatedAgent())
	view := m.View()

	// Overview and Activity start expanded (▼), Context starts collapsed (▸).
	if !strings.Contains(view, "▼ Overview") {
		t.Errorf("View() should show '▼ Overview' (expanded); got:\n%s", view)
	}
	if !strings.Contains(view, "▸ Context") {
		t.Errorf("View() should show '▸ Context' (collapsed); got:\n%s", view)
	}
}

func TestView_ConventionsShownWhenContextExpanded(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)

	// By default Context is collapsed — conventions should NOT appear.
	view := m.View()
	if strings.Contains(view, "go.md") {
		t.Errorf("collapsed Context section should NOT show 'go.md'; got:\n%s", view)
	}
}

func TestView_PromptSectionVisible(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	m.SetAgent(a)
	view := m.View()
	if !strings.Contains(view, "Prompt") {
		t.Errorf("View() should show 'Prompt' section header; got:\n%s", view)
	}
}

func TestView_ErrorSectionHiddenForRunning(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	m.SetAgent(fullyPopulatedAgent()) // StatusRunning, no error
	view := m.View()
	if strings.Contains(view, "▸ Error") || strings.Contains(view, "▼ Error") {
		t.Errorf("running agent should NOT show Error section; got:\n%s", view)
	}
}

func TestView_ErrorSectionVisibleForError(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	m.SetAgent(errorAgent())
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("error agent should show Error section; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Acceptance Criteria
// ---------------------------------------------------------------------------

func TestView_ACSection_EmptyNotShown(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	// AcceptanceCriteria is nil/empty by default.
	m.SetAgent(a)
	view := m.View()
	if strings.Contains(view, "Acceptance Criteria") {
		t.Errorf("empty AC list should NOT render section; got:\n%s", view)
	}
}

func TestView_ACSection_MixedCriteria(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	a.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "AC checklist renders with indicators", Completed: true},
		{Text: "Pending items show unchecked box", Completed: false},
	}
	m.SetAgent(a)
	view := m.View()

	if !strings.Contains(view, "Acceptance Criteria") {
		t.Errorf("View() should show 'Acceptance Criteria' section; got:\n%s", view)
	}
	if !strings.Contains(view, "[x]") {
		t.Errorf("View() should show '[x]' for completed item; got:\n%s", view)
	}
	if !strings.Contains(view, "[ ]") {
		t.Errorf("View() should show '[ ]' for pending item; got:\n%s", view)
	}
	if !strings.Contains(view, "AC checklist renders with indicators") {
		t.Errorf("View() should show completed item text; got:\n%s", view)
	}
	if !strings.Contains(view, "Pending items show unchecked box") {
		t.Errorf("View() should show pending item text; got:\n%s", view)
	}
}

func TestView_ACSection_AllCompleted(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	a.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "First criterion done", Completed: true},
		{Text: "Second criterion done", Completed: true},
	}
	m.SetAgent(a)
	view := m.View()

	if strings.Contains(view, "[ ]") {
		t.Errorf("all-completed AC should not show '[ ]'; got:\n%s", view)
	}
	// Both should show as completed.
	checkCount := strings.Count(view, "[x]")
	if checkCount < 2 {
		t.Errorf("all-completed AC should show 2 '[x]' markers, got %d; view:\n%s", checkCount, view)
	}
}

func TestView_ACSection_AllPending(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(120, 40)
	a := fullyPopulatedAgent()
	a.AcceptanceCriteria = []state.AcceptanceCriterion{
		{Text: "Not done yet", Completed: false},
		{Text: "Also pending", Completed: false},
	}
	m.SetAgent(a)
	view := m.View()

	if strings.Contains(view, "[x]") {
		t.Errorf("all-pending AC should not show '[x]'; got:\n%s", view)
	}
	uncheckedCount := strings.Count(view, "[ ]")
	if uncheckedCount < 2 {
		t.Errorf("all-pending AC should show 2 '[ ]' markers, got %d; view:\n%s", uncheckedCount, view)
	}
}

// ---------------------------------------------------------------------------
// Render — RenderFull
// ---------------------------------------------------------------------------

func TestDetailRender_FullMatchesView(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 40)
	m.SetAgent(fullyPopulatedAgent())

	if got, want := m.Render(agents.RenderFull, 80), m.View(); got != want {
		t.Errorf("Render(RenderFull, 80) != View()\nRender: %q\nView:   %q", got, want)
	}
}

func TestDetailRender_FullNilAgent(t *testing.T) {
	m := agents.NewAgentDetailModel()

	if got, want := m.Render(agents.RenderFull, 80), m.View(); got != want {
		t.Errorf("Render(RenderFull) on nil agent != View()")
	}
}

// ---------------------------------------------------------------------------
// Render — RenderIconRail (compact detail)
// ---------------------------------------------------------------------------

func TestDetailRender_CompactNilAgent(t *testing.T) {
	m := agents.NewAgentDetailModel()
	view := m.Render(agents.RenderIconRail, 22)
	// Should not panic and should return non-empty placeholder.
	if view == "" {
		t.Error("Render(RenderIconRail) on nil agent should not return empty string")
	}
}

func TestDetailRender_CompactShowsStatus(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(40, 10)
	a := fullyPopulatedAgent() // StatusRunning
	m.SetAgent(a)

	view := m.Render(agents.RenderIconRail, 40)
	if !strings.Contains(view, "Running") {
		t.Errorf("compact detail should contain 'Running'; got:\n%s", view)
	}
}

func TestDetailRender_CompactShowsModel(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(40, 10)
	a := fullyPopulatedAgent() // model="sonnet"
	m.SetAgent(a)

	view := m.Render(agents.RenderIconRail, 40)
	if !strings.Contains(view, "sonnet") {
		t.Errorf("compact detail should contain model name; got:\n%s", view)
	}
}

func TestDetailRender_CompactShowsCost(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(40, 10)
	a := fullyPopulatedAgent()
	a.Cost = 0.045
	m.SetAgent(a)

	view := m.Render(agents.RenderIconRail, 40)
	if !strings.Contains(view, "0.045") {
		t.Errorf("compact detail should contain cost value; got:\n%s", view)
	}
}

func TestDetailRender_CompactWidthBoundaries(t *testing.T) {
	m := agents.NewAgentDetailModel()
	m.SetSize(80, 20)
	a := fullyPopulatedAgent()
	a.Cost = 1.98
	m.SetAgent(a)

	for _, width := range []int{15, 22, 28, 29, 30, 31, 32, 45} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			view := m.Render(agents.RenderIconRail, width)
			if view == "" {
				t.Fatalf("Render(RenderIconRail, %d) returned empty string", width)
			}
			for i, line := range strings.Split(view, "\n") {
				w := lipgloss.Width(line)
				if w > width {
					t.Errorf("line %d: lipgloss.Width=%d exceeds width=%d: %q",
						i, w, width, line)
				}
			}
		})
	}
}

