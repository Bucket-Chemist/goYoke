package taskboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

func TestNewTaskBoardModel(t *testing.T) {
	m := NewTaskBoardModel()
	if m.visible {
		t.Error("expected visible=false on new model")
	}
	if m.showDone {
		t.Error("expected showDone=false on new model")
	}
	if len(m.tasks) != 0 {
		t.Errorf("expected empty tasks, got %d", len(m.tasks))
	}
}

func TestSetSize(t *testing.T) {
	m := NewTaskBoardModel()
	m.SetSize(100, 40)
	if m.width != 100 || m.height != 40 {
		t.Errorf("expected 100x40, got %dx%d", m.width, m.height)
	}
}

func TestToggle(t *testing.T) {
	m := NewTaskBoardModel()
	// IsVisible requires both visible flag AND tasks.
	m.SetTasks([]TaskEntry{{ID: "1", Status: "pending"}})
	// SetTasks auto-shows → visible=true, tasks non-empty → IsVisible=true.
	if !m.IsVisible() {
		t.Error("expected visible after SetTasks")
	}
	m.Toggle() // hide
	if m.IsVisible() {
		t.Error("expected invisible after Toggle")
	}
	m.Toggle() // show again
	if !m.IsVisible() {
		t.Error("expected visible after second Toggle")
	}
}

func TestCycleView(t *testing.T) {
	m := NewTaskBoardModel()
	if m.showDone {
		t.Error("expected showDone=false initially")
	}
	m.CycleView()
	if !m.showDone {
		t.Error("expected showDone=true after CycleView")
	}
	m.CycleView()
	if m.showDone {
		t.Error("expected showDone=false after second CycleView")
	}
}

func TestSetTasks_Nil(t *testing.T) {
	m := NewTaskBoardModel()
	m.SetTasks(nil)
	if m.tasks != nil {
		t.Error("expected nil tasks after SetTasks(nil)")
	}
}

func TestSetTasks_CopiesSlice(t *testing.T) {
	m := NewTaskBoardModel()
	tasks := []TaskEntry{{ID: "1", Content: "task one", Status: "pending"}}
	m.SetTasks(tasks)
	// Mutate original — model should not reflect the change.
	tasks[0].Content = "MUTATED"
	if m.tasks[0].Content == "MUTATED" {
		t.Error("SetTasks did not defensively copy the tasks slice")
	}
}

func TestView_Hidden(t *testing.T) {
	m := NewTaskBoardModel()
	view := m.View()
	if view != "" {
		t.Errorf("expected empty view when hidden, got %q", view)
	}
}

func TestHeight_Hidden(t *testing.T) {
	m := NewTaskBoardModel()
	if m.Height() != 0 {
		t.Errorf("expected height 0 when hidden, got %d", m.Height())
	}
}

func TestHeight_Visible(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle() // now visible
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "task 1", Status: "pending"},
		{ID: "2", Content: "task 2", Status: "in_progress"},
	})
	h := m.Height()
	// 1 progress summary + 1 filter bar + 2 task rows = 4
	if h != 4 {
		t.Errorf("expected height 4, got %d", h)
	}
}

func TestHeight_CapAtMaxRows(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	tasks := make([]TaskEntry, 20)
	for i := range tasks {
		tasks[i] = TaskEntry{ID: string(rune('a' + i)), Content: "task", Status: "pending"}
	}
	m.SetTasks(tasks)
	h := m.Height()
	// 1 progress summary + 1 filter bar + maxRows = 10
	if h != maxRows+2 {
		t.Errorf("expected height %d, got %d", maxRows+2, h)
	}
}

func TestView_AllFilter_ShowsAllTasks(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "Implementing feature X", Status: "in_progress"},
		{ID: "2", Content: "Running tests", Status: "pending"},
		{ID: "3", Content: "Updating docs", Status: "completed"},
	})
	view := m.View()
	// Default filter is FilterAll — all tasks visible.
	if !strings.Contains(view, "Implementing feature X") {
		t.Errorf("expected in_progress task in FilterAll view, got:\n%s", view)
	}
	if !strings.Contains(view, "Running tests") {
		t.Errorf("expected pending task in FilterAll view, got:\n%s", view)
	}
	if !strings.Contains(view, "Updating docs") {
		t.Errorf("expected completed task in FilterAll view, got:\n%s", view)
	}
}

func TestView_DoneFilter_ShowsOnlyCompleted(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "Done task", Status: "completed"},
		{ID: "2", Content: "Pending task", Status: "pending"},
	})
	// Activate FilterDone via HandleMsg.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	view := m.View()
	if !strings.Contains(view, "[Done]") {
		t.Errorf("expected '[Done]' active filter label, got:\n%s", view)
	}
	if !strings.Contains(view, "Done task") {
		t.Errorf("expected completed task in Done filter view, got:\n%s", view)
	}
	if strings.Contains(view, "Pending task") {
		t.Errorf("pending task should not appear in Done filter view, got:\n%s", view)
	}
}

func TestView_EmptyActiveTasks(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	// Toggled on but no tasks — View/Height return empty/0, IsVisible reflects toggle.
	if m.View() != "" {
		t.Errorf("expected empty view with no tasks, got:\n%s", m.View())
	}
	if m.Height() != 0 {
		t.Errorf("expected height 0 with no tasks, got %d", m.Height())
	}
}

func TestView_ProgressSummary_Counts(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "in_progress"},
	})
	view := m.View()
	if !strings.Contains(view, "1 running") {
		t.Errorf("expected '1 running' in progress summary, got:\n%s", view)
	}
	if !strings.Contains(view, "1 pending") {
		t.Errorf("expected '1 pending' in progress summary, got:\n%s", view)
	}
	if !strings.Contains(view, "Tasks:") {
		t.Errorf("expected 'Tasks:' header label, got:\n%s", view)
	}
}

func TestTruncate(t *testing.T) {
	// Tests use util.Truncate which appends "…" beyond maxRunes.
	// Production code calls util.Truncate(s, truncLen-1) to keep total within truncLen.
	tests := []struct {
		s        string
		maxRunes int
		want     string
	}{
		{"hello", 10, "hello"},
		{"hello world", 4, "hell…"},
		{"a", 1, "a"},
		{"", 10, ""},
	}
	for _, tc := range tests {
		got := util.Truncate(tc.s, tc.maxRunes)
		if got != tc.want {
			t.Errorf("util.Truncate(%q, %d): want %q, got %q", tc.s, tc.maxRunes, tc.want, got)
		}
	}
}

func TestTaskIcon(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"pending"},
		{"in_progress"},
		{"completed"},
		{"unknown"},
	}
	for _, tc := range tests {
		got := taskIcon(tc.status)
		if got == "" {
			t.Errorf("taskIcon(%q) returned empty string", tc.status)
		}
	}
}

func TestVisibleTasks_Filter(t *testing.T) {
	m := NewTaskBoardModel()
	m.SetTasks([]TaskEntry{
		{ID: "1", Status: "pending"},
		{ID: "2", Status: "in_progress"},
		{ID: "3", Status: "completed"},
	})

	// Active view.
	active := m.visibleTasks()
	if len(active) != 2 {
		t.Errorf("expected 2 active tasks, got %d", len(active))
	}

	// Done view.
	m.showDone = true
	done := m.visibleTasks()
	if len(done) != 1 {
		t.Errorf("expected 1 done task, got %d", len(done))
	}
}

// ---------------------------------------------------------------------------
// TUI-055: TaskFilterMode enum tests
// ---------------------------------------------------------------------------

func TestTaskFilterMode_Values(t *testing.T) {
	if FilterAll != 0 {
		t.Errorf("FilterAll: want 0, got %d", FilterAll)
	}
	if FilterRunning != 1 {
		t.Errorf("FilterRunning: want 1, got %d", FilterRunning)
	}
	if FilterPending != 2 {
		t.Errorf("FilterPending: want 2, got %d", FilterPending)
	}
	if FilterDone != 3 {
		t.Errorf("FilterDone: want 3, got %d", FilterDone)
	}
}

// ---------------------------------------------------------------------------
// TUI-055: HandleMsg cursor navigation tests
// ---------------------------------------------------------------------------

func TestHandleMsg_DownMoveCursor(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "pending"},
		{ID: "3", Content: "C", Status: "pending"},
	})
	if m.cursor != 0 {
		t.Fatalf("initial cursor: want 0, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after down: want cursor 1, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("after j: want cursor 2, got %d", m.cursor)
	}
}

func TestHandleMsg_UpMoveCursor(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "pending"},
	})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("after down: want 1, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("after up: want cursor 0, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("after k at top: want cursor 0, got %d", m.cursor)
	}
}

func TestHandleMsg_CursorClamp(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "pending"},
	})

	// Can't go below 0.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("up at top: want 0, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("k at top: want 0, got %d", m.cursor)
	}

	// Move to last item.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	// Can't go past last item.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("down past end: want 1, got %d", m.cursor)
	}
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("j past end: want 1, got %d", m.cursor)
	}
}

// ---------------------------------------------------------------------------
// TUI-055: HandleMsg filter mode tests
// ---------------------------------------------------------------------------

func TestHandleMsg_FilterAll(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.filterMode = FilterRunning // start in non-all mode
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "in_progress"},
	})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.filterMode != FilterAll {
		t.Errorf("after 'a': want FilterAll, got %d", m.filterMode)
	}
	if len(m.filtered) != 2 {
		t.Errorf("FilterAll should show 2 tasks, got %d", len(m.filtered))
	}
}

func TestHandleMsg_FilterRunning(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "in_progress"},
		{ID: "3", Content: "C", Status: "in_progress"},
	})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.filterMode != FilterRunning {
		t.Errorf("after 'r': want FilterRunning, got %d", m.filterMode)
	}
	if len(m.filtered) != 2 {
		t.Errorf("FilterRunning: want 2 tasks, got %d", len(m.filtered))
	}
	for _, task := range m.filtered {
		if task.Status != "in_progress" {
			t.Errorf("FilterRunning: unexpected status %q", task.Status)
		}
	}
}

func TestHandleMsg_FilterPending(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "pending"},
		{ID: "3", Content: "C", Status: "in_progress"},
	})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if m.filterMode != FilterPending {
		t.Errorf("after 'p': want FilterPending, got %d", m.filterMode)
	}
	if len(m.filtered) != 2 {
		t.Errorf("FilterPending: want 2 tasks, got %d", len(m.filtered))
	}
}

func TestHandleMsg_FilterDone(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "completed"},
		{ID: "2", Content: "B", Status: "in_progress"},
		{ID: "3", Content: "C", Status: "completed"},
	})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.filterMode != FilterDone {
		t.Errorf("after 'd': want FilterDone, got %d", m.filterMode)
	}
	if len(m.filtered) != 2 {
		t.Errorf("FilterDone: want 2 tasks, got %d", len(m.filtered))
	}
}

func TestHandleMsg_FilterResetsCursor(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Status: "in_progress"},
		{ID: "2", Status: "in_progress"},
		{ID: "3", Status: "completed"},
	})
	// Move cursor to last item.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("setup: want cursor 2, got %d", m.cursor)
	}
	// Switch filter — cursor must reset.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.cursor != 0 {
		t.Errorf("after filter change: want cursor 0, got %d", m.cursor)
	}
}

// HandleMsg is a no-op when the board is not visible.
func TestHandleMsg_NotVisibleIsNoop(t *testing.T) {
	m := NewTaskBoardModel()
	m.SetTasks([]TaskEntry{
		{ID: "1", Status: "pending"},
		{ID: "2", Status: "pending"},
	})
	// SetTasks auto-shows; toggle off so we can test the hidden state.
	m.Toggle()
	// Board is hidden — down key must not move cursor.
	m.HandleMsg(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 0 {
		t.Errorf("hidden board: want cursor 0, got %d", m.cursor)
	}
}

// ---------------------------------------------------------------------------
// TUI-055: View rendering tests
// ---------------------------------------------------------------------------

func TestView_ProgressSummary_AllStatuses(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Status: "in_progress"},
		{ID: "2", Status: "pending"},
		{ID: "3", Status: "completed"},
	})
	view := m.View()
	for _, want := range []string{"1 running", "1 pending", "1 done"} {
		if !strings.Contains(view, want) {
			t.Errorf("progress summary missing %q:\n%s", want, view)
		}
	}
}

func TestView_FilterIndicator_ActiveHighlighted(t *testing.T) {
	tests := []struct {
		name       string
		key        rune
		wantActive string
	}{
		{"all", 'a', "[All]"},
		{"running", 'r', "[Running]"},
		{"pending", 'p', "[Pending]"},
		{"done", 'd', "[Done]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewTaskBoardModel()
			m.Toggle()
			m.SetTasks([]TaskEntry{{ID: "1", Status: "in_progress"}})
			m.HandleMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
			view := m.View()
			if !strings.Contains(view, tc.wantActive) {
				t.Errorf("expected %q highlighted in filter bar, got:\n%s", tc.wantActive, view)
			}
		})
	}
}

func TestView_StatusBadges(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "Running task", Status: "in_progress"},
		{ID: "2", Content: "Pending task", Status: "pending"},
		{ID: "3", Content: "Done task", Status: "completed"},
	})
	view := m.View()
	if !strings.Contains(view, "[running]") {
		t.Errorf("expected [running] badge, got:\n%s", view)
	}
	if !strings.Contains(view, "[pending]") {
		t.Errorf("expected [pending] badge, got:\n%s", view)
	}
	if !strings.Contains(view, "[done]") {
		t.Errorf("expected [done] badge, got:\n%s", view)
	}
}

func TestView_CursorHighlight(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "First task", Status: "pending"},
		{ID: "2", Content: "Second task", Status: "pending"},
	})
	view := m.View()
	// The cursor indicator ">" should appear for the selected (first) item.
	if !strings.Contains(view, ">") {
		t.Errorf("expected cursor indicator '>' in view, got:\n%s", view)
	}
}

func TestSetTasks_AppliesFilter(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	// Set FilterRunning before calling SetTasks.
	m.filterMode = FilterRunning
	m.SetTasks([]TaskEntry{
		{ID: "1", Status: "in_progress"},
		{ID: "2", Status: "pending"},
		{ID: "3", Status: "in_progress"},
	})
	if len(m.filtered) != 2 {
		t.Errorf("SetTasks with FilterRunning: want 2, got %d", len(m.filtered))
	}
}
