// Package state provides shared, thread-safe state containers for the
// goYoke TUI.
//
// TaskEntry is defined here so that both the model package (interface
// definition) and the taskboard component can reference it without model
// importing taskboard — keeping the import graph acyclic.
package state

// TaskEntry represents a single task tracked by the task board.
type TaskEntry struct {
	// ID is the unique identifier for this task.
	ID string
	// Content is the human-readable task description.
	Content string
	// Status is the lifecycle state: "pending", "in_progress", or "completed".
	Status string
	// ActiveForm is an optional label for the form step currently active
	// (e.g. the tool name or sub-step in progress).
	ActiveForm string
}
