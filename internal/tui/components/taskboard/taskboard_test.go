package taskboard

import (
	"strings"
	"testing"
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
	if m.IsVisible() {
		t.Error("expected invisible initially")
	}
	m.Toggle()
	if !m.IsVisible() {
		t.Error("expected visible after first Toggle")
	}
	m.Toggle()
	if m.IsVisible() {
		t.Error("expected invisible after second Toggle")
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
	// 1 header + 2 task rows = 3
	if h != 3 {
		t.Errorf("expected height 3, got %d", h)
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
	// 1 header + maxRows = 9
	if h != maxRows+1 {
		t.Errorf("expected height %d, got %d", maxRows+1, h)
	}
}

func TestView_ActiveTasks(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "Implementing feature X", Status: "in_progress"},
		{ID: "2", Content: "Running tests", Status: "pending"},
		{ID: "3", Content: "Updating docs", Status: "completed"}, // should be filtered
	})
	view := m.View()
	if !strings.Contains(view, "Active") {
		t.Errorf("expected 'Active' label, got:\n%s", view)
	}
	if !strings.Contains(view, "Implementing feature X") {
		t.Errorf("expected active task in view, got:\n%s", view)
	}
	// Completed tasks should NOT appear in Active view.
	if strings.Contains(view, "Updating docs") {
		t.Errorf("completed task should not appear in Active view, got:\n%s", view)
	}
}

func TestView_DoneTasks(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.CycleView() // switch to Done
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "Done task", Status: "completed"},
		{ID: "2", Content: "Pending task", Status: "pending"}, // should be filtered
	})
	view := m.View()
	if !strings.Contains(view, "Done") {
		t.Errorf("expected 'Done' label, got:\n%s", view)
	}
	if !strings.Contains(view, "Done task") {
		t.Errorf("expected completed task in Done view, got:\n%s", view)
	}
	if strings.Contains(view, "Pending task") {
		t.Errorf("pending task should not appear in Done view, got:\n%s", view)
	}
}

func TestView_EmptyActiveTasks(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	// No tasks set — should show (none).
	view := m.View()
	if !strings.Contains(view, "(none)") {
		t.Errorf("expected '(none)' for empty task list, got:\n%s", view)
	}
}

func TestView_TaskCount(t *testing.T) {
	m := NewTaskBoardModel()
	m.Toggle()
	m.SetTasks([]TaskEntry{
		{ID: "1", Content: "A", Status: "pending"},
		{ID: "2", Content: "B", Status: "in_progress"},
	})
	view := m.View()
	if !strings.Contains(view, "(2)") {
		t.Errorf("expected task count '(2)' in header, got:\n%s", view)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell\u2026"},
		{"a", 1, "\u2026"},
		{"", 10, ""},
	}
	for _, tc := range tests {
		got := truncate(tc.s, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d): want %q, got %q", tc.s, tc.maxLen, tc.want, got)
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
