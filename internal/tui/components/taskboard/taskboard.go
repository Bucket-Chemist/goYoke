// Package taskboard implements the task board overlay for the
// GOgent-Fortress TUI. It renders a compact strip showing active or completed
// tasks and is toggled by the Alt+B keybinding.
//
// The component is display-only: no I/O is performed in View. State is
// managed via Toggle, CycleView, and SetTasks.
package taskboard

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// TaskEntry is re-exported from the state package for backward compatibility.
// New code should use state.TaskEntry directly.
type TaskEntry = state.TaskEntry

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// maxRows is the maximum number of task rows rendered at once.
	maxRows = 8
	// truncLen is the maximum rune length for a task content line.
	truncLen = 60
)

// ---------------------------------------------------------------------------
// TaskBoardModel
// ---------------------------------------------------------------------------

// TaskBoardModel is the display model for the task board overlay. When not
// visible (the default) View returns an empty string and Height returns 0 so
// the layout code can safely include it without rendering extra whitespace.
//
// The zero value is not usable; use NewTaskBoardModel instead.
type TaskBoardModel struct {
	width    int
	height   int
	visible  bool // toggled by Alt+B
	showDone bool // false=Active view, true=Done view
	tasks    []TaskEntry
}

// NewTaskBoardModel returns a TaskBoardModel in the hidden state.
func NewTaskBoardModel() TaskBoardModel {
	return TaskBoardModel{}
}

// SetSize updates the rendering dimensions.
func (m *TaskBoardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Toggle flips the visible state of the task board.
func (m *TaskBoardModel) Toggle() {
	m.visible = !m.visible
}

// CycleView flips between the Active and Done task views.
func (m *TaskBoardModel) CycleView() {
	m.showDone = !m.showDone
}

// SetTasks replaces the current task list with a defensive copy of tasks.
func (m *TaskBoardModel) SetTasks(tasks []TaskEntry) {
	if len(tasks) == 0 {
		m.tasks = nil
		return
	}
	cp := make([]TaskEntry, len(tasks))
	copy(cp, tasks)
	m.tasks = cp
}

// IsVisible returns true when the task board overlay is shown.
func (m TaskBoardModel) IsVisible() bool {
	return m.visible
}

// Height returns the number of terminal rows the task board occupies.
// Returns 0 when not visible.
func (m TaskBoardModel) Height() int {
	if !m.visible {
		return 0
	}
	tasks := m.visibleTasks()
	// 1 header row + min(len(tasks), maxRows) task rows + 0 padding.
	rows := 1 + min(len(tasks), maxRows)
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the task board overlay strip. Returns an empty string when the
// board is not visible. It is a pure function of the model state — no I/O is
// performed here.
func (m TaskBoardModel) View() string {
	if !m.visible {
		return ""
	}

	tasks := m.visibleTasks()

	var sb strings.Builder

	// Header line.
	viewLabel := "Active"
	if m.showDone {
		viewLabel = "Done"
	}
	header := config.StyleTitle.Render(
		fmt.Sprintf("Tasks [%s] (%d)", viewLabel, len(tasks)),
	)
	sb.WriteString(header)
	sb.WriteByte('\n')

	if len(tasks) == 0 {
		sb.WriteString(config.StyleSubtle.Render("  (none)"))
		return sb.String()
	}

	// Render up to maxRows tasks.
	limit := min(len(tasks), maxRows)
	for _, task := range tasks[:limit] {
		sb.WriteString(m.renderTask(task))
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}

// visibleTasks returns the subset of tasks matching the current showDone view.
func (m TaskBoardModel) visibleTasks() []TaskEntry {
	var result []TaskEntry
	for _, t := range m.tasks {
		if m.showDone {
			if t.Status == "completed" {
				result = append(result, t)
			}
		} else {
			if t.Status != "completed" {
				result = append(result, t)
			}
		}
	}
	return result
}

// renderTask renders a single task row.
func (m TaskBoardModel) renderTask(t TaskEntry) string {
	icon := taskIcon(t.Status)
	content := util.Truncate(t.Content, truncLen-1)

	var line string
	if m.showDone {
		// Done view: checkmark prefix.
		line = config.StyleSuccess.Render(icon) + " " + config.StyleSubtle.Render(content)
	} else {
		// Active view: status icon + content.
		line = config.StyleHighlight.Render(icon) + " " + content
	}
	return line
}

// taskIcon returns the status icon character for a task.
func taskIcon(status string) string {
	switch status {
	case "in_progress":
		return string(config.IconRunning)
	case "completed":
		return string(config.IconComplete)
	case "pending":
		return string(config.IconPending)
	default:
		return string(config.IconPending)
	}
}

