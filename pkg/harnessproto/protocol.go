// Package harnessproto defines the stable public wire contract for Harness Link.
// It is transport-neutral: callers may use any framing (UDS, stdio, etc.) to
// exchange the JSON-encoded envelopes defined here.
//
// All types in this package are self-contained and must not import any
// internal/ package.
package harnessproto

import (
	"encoding/json"
	"time"
)

// Protocol identity constants.
const (
	// ProtocolName is the canonical name embedded in every envelope.
	ProtocolName = "harness-link"

	// ProtocolVersion is the current wire-protocol version.
	// Bump the major segment on breaking schema changes.
	ProtocolVersion = "1.0.0"
)

// Operation kind constants used in the Request.Kind discriminator field.
const (
	KindPing              = "ping"
	KindGetSnapshot       = "get_snapshot"
	KindSubmitPrompt      = "submit_prompt"
	KindInterrupt         = "interrupt"
	KindRespondModal      = "respond_modal"
	KindRespondPermission = "respond_permission"
	KindSetModel          = "set_model"
	KindSetEffort         = "set_effort"
	KindSetCWD            = "set_cwd"
)

// Error code constants used in ErrorDetail.Code.
const (
	// ErrUnsupportedOperation is returned when the server does not implement
	// the requested operation kind.
	ErrUnsupportedOperation = "unsupported_operation"

	// ErrBadRequest is returned when the request payload is malformed or
	// missing required fields.
	ErrBadRequest = "bad_request"

	// ErrUnavailableState is returned when the operation is valid but cannot
	// be completed given the current TUI session state (e.g. no active session).
	ErrUnavailableState = "unavailable_state"

	// ErrVersionMismatch is returned when the client's protocol_version is
	// incompatible with the server's.
	ErrVersionMismatch = "version_mismatch"
)

// Request is the client-to-server envelope.
// Kind selects the operation; Payload carries operation-specific parameters
// as a raw JSON object and may be omitted for zero-parameter operations.
type Request struct {
	Protocol        string          `json:"protocol"`
	ProtocolVersion string          `json:"protocol_version"`
	Kind            string          `json:"kind"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

// Response is the server-to-client envelope.
// OK is true on success; on failure OK is false and Error is populated.
// Payload carries operation-specific results and may be omitted for void
// operations.
type Response struct {
	Protocol        string          `json:"protocol"`
	ProtocolVersion string          `json:"protocol_version"`
	Kind            string          `json:"kind"`
	OK              bool            `json:"ok"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	Error           *ErrorDetail    `json:"error,omitempty"`
}

// ErrorDetail carries structured error information in a Response.
// Code is always one of the Err* constants; Message is a human-readable
// explanation intended for logs and diagnostics.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// -----------------------------------------------------------------------
// Operation payload types — client → server
// -----------------------------------------------------------------------

// SubmitPromptRequest is the payload for KindSubmitPrompt.
type SubmitPromptRequest struct {
	Text string `json:"text"`
}

// RespondModalRequest is the payload for KindRespondModal.
type RespondModalRequest struct {
	// Selection is the label or index of the option the harness chose.
	Selection string `json:"selection"`
}

// RespondPermissionRequest is the payload for KindRespondPermission.
type RespondPermissionRequest struct {
	Allow bool `json:"allow"`
}

// SetModelRequest is the payload for KindSetModel.
type SetModelRequest struct {
	Model string `json:"model"`
}

// SetEffortRequest is the payload for KindSetEffort.
type SetEffortRequest struct {
	Effort string `json:"effort"`
}

// SetCWDRequest is the payload for KindSetCWD.
type SetCWDRequest struct {
	CWD string `json:"cwd"`
}

// -----------------------------------------------------------------------
// Snapshot types — server → client (response payload for KindGetSnapshot)
// -----------------------------------------------------------------------

// SessionSnapshot is the authoritative semantic state of a running goYoke TUI
// session, as seen by an external harness.
//
// Fields without omitempty are always present in the JSON output.
// Fields with omitempty are elided when they carry their zero value, which
// external consumers should interpret as "not applicable" rather than
// "explicitly false/empty".
type SessionSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	Protocol        string    `json:"protocol"`
	ProtocolVersion string    `json:"protocol_version"`

	// SessionID is the live provider/CLI session identifier assigned by the
	// active backend (for example the Claude CLI session ID). It may be empty
	// during early startup before the provider emits its first init event.
	SessionID string `json:"session_id,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
	Effort    string `json:"effort,omitempty"`
	CWD       string `json:"cwd,omitempty"`

	// Status is a coarse session-state label (e.g. "idle", "streaming",
	// "waiting_modal", "waiting_permission", "shutting_down").
	Status    string `json:"status"`
	Streaming bool   `json:"streaming"`

	Reconnecting bool `json:"reconnecting,omitempty"`
	ShuttingDown bool `json:"shutting_down,omitempty"`

	ActiveTab string `json:"active_tab,omitempty"`
	Focus     string `json:"focus,omitempty"`

	PlanActive bool `json:"plan_active,omitempty"`
	PlanStep   int  `json:"plan_step,omitempty"`
	PlanTotal  int  `json:"plan_total,omitempty"`

	// Agents is the list of currently tracked agent processes. It is always
	// present (may be an empty array when no agents are active).
	Agents []AgentSummary `json:"agents"`

	Team    *TeamSummary   `json:"team,omitempty"`
	Pending *PendingPrompt `json:"pending,omitempty"`

	LastUser      string   `json:"last_user,omitempty"`
	LastAssistant string   `json:"last_assistant,omitempty"`
	LastError     string   `json:"last_error,omitempty"`
	Highlights    []string `json:"highlights,omitempty"`

	// StateHash changes whenever operator-visible state changes.
	StateHash string `json:"state_hash"`

	// PublishHash changes only when a human notification is warranted (a
	// subset of state changes).
	PublishHash string `json:"publish_hash"`
}

// AgentSummary is a lightweight representation of a single agent process.
type AgentSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Model  string `json:"model,omitempty"`
}

// TeamSummary is a lightweight representation of a team orchestration run.
type TeamSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status,omitempty"`
	Members int    `json:"members,omitempty"`
}

// PendingPrompt describes a modal or permission request that is waiting for a
// harness response.
type PendingPrompt struct {
	// Kind is "modal" or "permission".
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
}
