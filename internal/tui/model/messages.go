// Package model defines shared state types for the GOgent-Fortress TUI.
// This file defines all tea.Msg types used across the event-driven architecture.
// Message types are the contracts between components; define them here so all
// components can import without circular dependencies.
package model

import (
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// CLI event messages
//
// These messages carry events from the NDJSON stream produced by the Claude
// CLI subprocess.  Each type corresponds to one event class in the NDJSON
// catalog (TUI-003).
// ---------------------------------------------------------------------------

// SystemInitMsg is emitted when the CLI session initialises and a session ID
// is first assigned.
type SystemInitMsg struct {
	SessionID string
}

// StatusUpdateMsg carries a plain-text status string from the CLI.  Examples
// include "Thinking…" or "Executing tool".
type StatusUpdateMsg struct {
	Status string
}

// CompactMsg is emitted when the CLI emits a compacted representation of a
// long output — typically used to summarise a large tool result.
type CompactMsg struct {
	Text string
}

// AssistantMsg carries a fragment of assistant output text.  When Streaming
// is true the fragment is an in-progress delta; when false it is a complete
// turn.
type AssistantMsg struct {
	Text      string
	Streaming bool
}

// ToolResultMsg carries the result of a single tool invocation.
type ToolResultMsg struct {
	ToolName string
	Result   string
	Success  bool
}

// ResultMsg is emitted at the end of a CLI session turn and summarises cost
// and duration.
type ResultMsg struct {
	SessionID  string
	CostUSD    float64
	DurationMS int64
}

// StreamEventMsg wraps a raw stream event before it has been decoded into a
// typed message.  Components that need to introspect the raw bytes can match
// on this type.
type StreamEventMsg struct {
	EventType string
	Data      []byte
}

// CLIEventMsg is the fallback message type for NDJSON events whose type is
// not recognised by the decoder.  It preserves the raw bytes so that future
// handlers can be added without dropping events.
type CLIEventMsg struct {
	RawType string
	Data    []byte
}

// ---------------------------------------------------------------------------
// UI messages
//
// These messages drive modal dialogs, toast notifications, and periodic ticks.
// ---------------------------------------------------------------------------

// ModalRequestMsg asks the root AppModel to display a modal dialog.  The
// Title is shown as the modal heading and Options lists the selectable items.
type ModalRequestMsg struct {
	Title   string
	Options []string
}

// ToastLevel is the severity level of a toast notification.
type ToastLevel string

const (
	// ToastLevelInfo is used for informational notifications.
	ToastLevelInfo ToastLevel = "info"
	// ToastLevelWarn is used for warning notifications.
	ToastLevelWarn ToastLevel = "warn"
	// ToastLevelError is used for error notifications.
	ToastLevelError ToastLevel = "error"
	// ToastLevelSuccess is used for success notifications.
	ToastLevelSuccess ToastLevel = "success"
)

// ToastMsg requests a transient notification.
type ToastMsg struct {
	Text  string
	Level ToastLevel
}

// TickMsg carries the current wall-clock time and is used by any component
// that needs periodic refresh (e.g. elapsed-time counters).
type TickMsg struct {
	Time time.Time
}

// ---------------------------------------------------------------------------
// Agent messages
//
// These messages reflect lifecycle events for individual agent processes
// tracked by the TUI.
// ---------------------------------------------------------------------------

// AgentRegisteredMsg is emitted when a new agent process is first detected.
// ParentID is empty for root-level agents.
type AgentRegisteredMsg struct {
	AgentID   string
	AgentType string
	ParentID  string
}

// AgentUpdatedMsg is emitted when an existing agent's status changes.
// Common Status values: "running", "complete", "error", "cancelled".
type AgentUpdatedMsg struct {
	AgentID string
	Status  string
}

// AgentActivityMsg is emitted when an agent starts or finishes streaming a
// tool call.  When Streaming is true the tool call is in progress.
type AgentActivityMsg struct {
	AgentID   string
	ToolName  string
	Streaming bool
}

// ---------------------------------------------------------------------------
// Team messages
//
// These messages reflect lifecycle events for gogent-team-run sessions.
// ---------------------------------------------------------------------------

// TeamUpdateMsg is emitted when a team's overall status or an individual
// task's status changes.  TaskID is empty when the update concerns the whole
// team rather than a single task.
type TeamUpdateMsg struct {
	TeamDir string
	Status  string
	TaskID  string
}

// ---------------------------------------------------------------------------
// Startup messages (TUI-016)
//
// These messages drive the CLI + bridge startup sequence wired in app.go.
// ---------------------------------------------------------------------------

// CLIReadyMsg is sent by the startup sequence after the SystemInitEvent has
// been processed and the session ID is available.
type CLIReadyMsg struct {
	// SessionID is the claude session identifier from SystemInitEvent.
	SessionID string
	// Model is the active model name (e.g. "claude-opus-4-6").
	Model string
	// Tools is the list of tool names available in this session.
	Tools []string
}

// StartupErrorMsg is sent when a startup component fails to initialise.
type StartupErrorMsg struct {
	// Component names the subsystem that failed: "bridge", "cli", or "mcp".
	Component string
	// Err is the underlying error.
	Err error
}

// CLIReconnectMsg is sent by a reconnection timer to trigger a fresh
// Start() call on the CLI driver after a disconnect.
type CLIReconnectMsg struct {
	// Attempt is the 1-based reconnection attempt number.
	Attempt int
	// Seq is the reconnect sequence number at the time the timer was created.
	// Stale timers (from before a provider switch) carry a lower Seq and are
	// discarded, preventing ghost reconnections after the CLI driver is replaced.
	Seq int
}

// ---------------------------------------------------------------------------
// Bridge messages (TUI-016)
//
// BridgeModalRequestMsg is defined here (in model) rather than in the bridge
// package so that AppModel.Update can type-switch on it without creating a
// circular import.  The bridge package already imports model for other message
// types, so placing this type here is consistent with that dependency
// direction.
// ---------------------------------------------------------------------------

// BridgeModalRequestMsg is sent by the IPC bridge when the MCP server asks
// the TUI to display a modal dialog and return the user's selection.
// RequestID must be passed to AppModel's ResolveModal call so the bridge can
// correlate the response.
type BridgeModalRequestMsg struct {
	// RequestID is the IPC request identifier used to route the response.
	RequestID string
	// Message is the human-readable prompt displayed to the user.
	Message string
	// Options lists the selectable button labels. Empty means free-text input.
	Options []string
}

// ---------------------------------------------------------------------------
// Provider messages (TUI-029)
//
// ProviderSwitchMsg drives the provider-switching flow wired in app.go.
// ---------------------------------------------------------------------------

// ProviderSwitchMsg is emitted when the user cycles to the next provider.
// The handler saves the current conversation state, switches provider,
// restores the new provider's conversation state, and restarts the CLI driver.
//
// This type is retained for programmatic (non-debounced) provider switches.
// Key-press driven switches go through the ProviderSwitchExecuteMsg path.
type ProviderSwitchMsg struct{}

// ProviderSwitchExecuteMsg is the debounced execution of a provider switch.
// It fires 300 ms after the last CycleProvider keypress. The Seq field
// carries the sequence counter value at the time the timer was created;
// handlers ignore messages whose Seq does not match the model's current
// providerSwitchSeq counter, providing natural debounce cancellation without
// any explicit timer management.
type ProviderSwitchExecuteMsg struct {
	// Seq is the sequence counter at the time the debounce timer was created.
	// Stale timers (from earlier keypresses) have a lower Seq and are discarded.
	Seq int
}

// ---------------------------------------------------------------------------
// Session persistence messages (TUI-033)
// ---------------------------------------------------------------------------

// SessionAutoSaveMsg is emitted by a debounced timer (5 s cooldown) after a
// cost-change event.  The Seq field carries the sequence counter at the time
// the timer was created; stale timers (from earlier events) have a lower Seq
// and are silently discarded.
type SessionAutoSaveMsg struct {
	// Seq is the auto-save sequence counter at timer creation time.
	Seq int
}

// ---------------------------------------------------------------------------
// Shutdown messages (TUI-034)
// ---------------------------------------------------------------------------

// ShutdownCompleteMsg is sent after the ShutdownManager finishes its
// sequenced shutdown.  The Err field carries ErrShutdownTimeout if the
// total budget was exceeded, or nil on success.
type ShutdownCompleteMsg struct {
	Err error
}

// ---------------------------------------------------------------------------
// Theme messages (TUI-046)
// ---------------------------------------------------------------------------

// ThemeChangedMsg is sent when the user selects a different color theme.
// The handler in AppModel.Update calls config.NewTheme(Variant) to build
// the full Theme value and stores it in sharedState.activeTheme.
//
// Components that hold a pointer to sharedState read activeTheme directly
// (no additional message dispatch needed). Components without sharedState
// access continue using package-level config defaults until TUI-048/050
// add SetTheme() to their widget interfaces.
type ThemeChangedMsg struct {
	// Variant is the color palette to activate.
	Variant config.ThemeVariant
}
