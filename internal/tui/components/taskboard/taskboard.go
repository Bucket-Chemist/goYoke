// Package taskboard implements the task board overlay for the
// goYoke TUI. It renders a compact strip showing active or completed
// tasks and is toggled by the Alt+B keybinding.
//
// The component is interactive: cursor navigation (j/k or up/down) moves
// between tasks, and filter shortcuts (a/r/p/d) restrict the visible set.
// State is managed via Toggle, CycleView, SetTasks, and HandleMsg.
package taskboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/Bucket-Chemist/goYoke/internal/tui/util"
)

// TaskEntry is re-exported from the state package for backward compatibility.
// New code should use state.TaskEntry directly.
type TaskEntry = state.TaskEntry

// ---------------------------------------------------------------------------
// TaskFilterMode
// ---------------------------------------------------------------------------

// TaskFilterMode controls which subset of tasks is shown in the board.
type TaskFilterMode int

const (
	// FilterAll shows every task regardless of status.
	FilterAll TaskFilterMode = iota
	// FilterRunning shows only tasks with status "in_progress".
	FilterRunning
	// FilterPending shows only tasks with status "pending".
	FilterPending
	// FilterDone shows only tasks with status "completed".
	FilterDone
)

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

// TaskBoardModel is the interactive model for the task board overlay. When not
// visible (the default) View returns an empty string and Height returns 0 so
// the layout code can safely include it without rendering extra whitespace.
//
// The zero value is not usable; use NewTaskBoardModel instead.
type TaskBoardModel struct {
	width      int
	height     int
	visible    bool // toggled by Alt+B
	showDone   bool // false=Active view, true=Done view (legacy CycleView)
	tasks      []TaskEntry
	cursor     int
	filterMode TaskFilterMode
	theme      config.Theme
	filtered   []TaskEntry // filtered view of tasks
}

// NewTaskBoardModel returns a TaskBoardModel in the hidden state with the
// default theme applied. The board auto-shows when SetTasks is called with
// a non-empty task list. Toggle manually with Alt+B.
func NewTaskBoardModel() TaskBoardModel {
	return TaskBoardModel{
		theme: config.DefaultTheme(),
	}
}

// SetSize updates the rendering dimensions.
func (m *TaskBoardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetTier satisfies the taskBoardWidget interface.  Tier-specific rendering
// adaptations are reserved for a future ticket; this is a no-op placeholder.
func (m *TaskBoardModel) SetTier(_ model.LayoutTier) {}

// SetTheme replaces the theme used for status badge styling.
func (m *TaskBoardModel) SetTheme(t config.Theme) {
	m.theme = t
}

// Toggle flips the visible state of the task board.
func (m *TaskBoardModel) Toggle() {
	m.visible = !m.visible
}

// CycleView flips between the Active and Done task views.
// Deprecated: use filterMode shortcuts instead. Kept for backward compatibility.
func (m *TaskBoardModel) CycleView() {
	m.showDone = !m.showDone
}

// SetTasks replaces the current task list with a defensive copy of tasks and
// refreshes the filtered view. Auto-shows the board when tasks arrive.
func (m *TaskBoardModel) SetTasks(tasks []TaskEntry) {
	if len(tasks) == 0 {
		m.tasks = nil
		m.filtered = nil
		m.cursor = 0
		return
	}
	cp := make([]TaskEntry, len(tasks))
	copy(cp, tasks)
	m.tasks = cp
	m.applyFilter()
	// Auto-show the board when tasks are populated for the first time.
	// The user can still hide it with Alt+B.
	m.visible = true
}

// IsVisible returns true when the task board toggle is on. Height() and View()
// independently guard against rendering when there are no tasks, so IsVisible
// reflects the user's toggle intent without coupling to task state.
func (m TaskBoardModel) IsVisible() bool {
	return m.visible
}

// Height returns the number of terminal rows the task board occupies.
// Returns 0 when not visible or when there are no tasks.
// MUST match View() line count exactly or the layout overflows and clips
// the banner/tab bar off the top of the screen.
func (m TaskBoardModel) Height() int {
	if !m.visible || len(m.tasks) == 0 {
		return 0
	}
	tasks := m.displayTasks()
	// 1 progress summary + 1 filter bar + min(len(tasks), maxRows) task rows.
	h := 2 + min(len(tasks), maxRows)
	// +1 for the "Form: ..." detail row when the selected task has an ActiveForm.
	if m.cursor < len(tasks) && tasks[m.cursor].ActiveForm != "" {
		h++
	}
	return h
}

// HandleMsg handles keyboard messages when the task board is visible. It
// satisfies the taskBoardWidget interface extension added in TUI-055.
func (m *TaskBoardModel) HandleMsg(msg tea.Msg) tea.Cmd {
	if !m.visible || len(m.tasks) == 0 {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "a":
			m.filterMode = FilterAll
			m.applyFilter()
		case "r":
			m.filterMode = FilterRunning
			m.applyFilter()
		case "p":
			m.filterMode = FilterPending
			m.applyFilter()
		case "d":
			m.filterMode = FilterDone
			m.applyFilter()
		}
	}
	return nil
}

// applyFilter rebuilds m.filtered based on the current filterMode and resets
// the cursor to 0 so it always lands within the new result set.
func (m *TaskBoardModel) applyFilter() {
	var result []TaskEntry
	for _, t := range m.tasks {
		if m.matchesFilter(t) {
			result = append(result, t)
		}
	}
	m.filtered = result
	m.cursor = 0
}

// matchesFilter returns true when the task should appear in the current filter
// mode.
func (m TaskBoardModel) matchesFilter(t TaskEntry) bool {
	switch m.filterMode {
	case FilterRunning:
		return t.Status == "in_progress"
	case FilterPending:
		return t.Status == "pending"
	case FilterDone:
		return t.Status == "completed"
	default: // FilterAll
		return true
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the task board overlay strip. Returns an empty string when the
// board is not visible. It is a pure function of the model state — no I/O is
// performed here.
func (m TaskBoardModel) View() string {
	if !m.visible || len(m.tasks) == 0 {
		return ""
	}

	tasks := m.displayTasks()

	var sb strings.Builder

	// Progress summary header.
	sb.WriteString(m.renderProgressSummary())
	sb.WriteByte('\n')

	// Filter indicator bar.
	sb.WriteString(m.renderFilterBar())
	sb.WriteByte('\n')

	if len(tasks) == 0 {
		// All tasks filtered out — show nothing rather than "(none)" to keep
		// Height() and View() line counts consistent.
		return sb.String()
	}

	// Render up to maxRows tasks.
	limit := min(len(tasks), maxRows)
	for i, task := range tasks[:limit] {
		sb.WriteString(m.renderTask(task, i == m.cursor))
		sb.WriteByte('\n')
	}

	// Expanded detail for the selected task.
	if m.cursor < len(tasks) {
		selected := tasks[m.cursor]
		if selected.ActiveForm != "" {
			detail := config.StyleSubtle.Render("  Form: " + selected.ActiveForm)
			sb.WriteString(detail)
			sb.WriteByte('\n')
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderProgressSummary returns the top summary line: N running • N pending • N done.
func (m TaskBoardModel) renderProgressSummary() string {
	var running, pending, done int
	for _, t := range m.tasks {
		switch t.Status {
		case "in_progress":
			running++
		case "pending":
			pending++
		case "completed":
			done++
		}
	}

	runningPart := m.theme.InfoStyle().Render(fmt.Sprintf("%d running", running))
	pendingPart := m.theme.WarningStyle().Render(fmt.Sprintf("%d pending", pending))
	donePart := m.theme.SuccessStyle().Render(fmt.Sprintf("%d done", done))
	sep := config.StyleSubtle.Render(" • ")

	return config.StyleTitle.Render("Tasks: ") + runningPart + sep + pendingPart + sep + donePart
}

// renderFilterBar returns the filter tab indicator line.
func (m TaskBoardModel) renderFilterBar() string {
	type tab struct {
		mode  TaskFilterMode
		label string
	}
	tabs := []tab{
		{FilterAll, "All"},
		{FilterRunning, "Running"},
		{FilterPending, "Pending"},
		{FilterDone, "Done"},
	}

	var parts []string
	for _, tb := range tabs {
		if tb.mode == m.filterMode {
			parts = append(parts, config.StyleHighlight.Render("["+tb.label+"]"))
		} else {
			parts = append(parts, config.StyleSubtle.Render(tb.label))
		}
	}
	return strings.Join(parts, " ")
}

// displayTasks returns the filtered task list used for rendering. When
// filterMode is FilterAll the legacy showDone view is ignored; caller controls
// display via filterMode. The filtered field is always the authoritative source.
func (m TaskBoardModel) displayTasks() []TaskEntry {
	if m.filtered != nil || len(m.tasks) == 0 {
		return m.filtered
	}
	// Fallback: rebuild on the fly (handles zero-value model).
	var result []TaskEntry
	for _, t := range m.tasks {
		if m.matchesFilter(t) {
			result = append(result, t)
		}
	}
	return result
}

// visibleTasks returns the subset of tasks matching the current showDone view.
// Kept for backward compatibility with existing tests.
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

// renderTask renders a single task row. isSelected adds a cursor indicator.
func (m TaskBoardModel) renderTask(t TaskEntry, isSelected bool) string {
	icon := taskIcon(t.Status)
	content := util.Truncate(t.Content, truncLen-1)
	badge := m.statusBadge(t.Status)

	var cursor string
	if isSelected {
		cursor = config.StyleHighlight.Render("> ")
	} else {
		cursor = "  "
	}

	if isSelected {
		line := cursor + config.StyleHighlight.Render(icon) + " " + badge + " " + config.StyleHighlight.Render(content)
		return line
	}
	return cursor + taskIconStyled(m.theme, t.Status, icon) + " " + badge + " " + content
}

// statusBadge returns a colored status label for the task.
func (m TaskBoardModel) statusBadge(status string) string {
	switch status {
	case "in_progress":
		return m.theme.InfoStyle().Render("[running]")
	case "pending":
		return m.theme.WarningStyle().Render("[pending]")
	case "completed":
		return m.theme.SuccessStyle().Render("[done]")
	default:
		return m.theme.ErrorStyle().Render("[error]")
	}
}

// taskIconStyled returns the icon character styled according to task status.
func taskIconStyled(theme config.Theme, status, icon string) string {
	switch status {
	case "in_progress":
		return theme.InfoStyle().Render(icon)
	case "completed":
		return theme.SuccessStyle().Render(icon)
	case "pending":
		return theme.WarningStyle().Render(icon)
	default:
		return theme.ErrorStyle().Render(icon)
	}
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
