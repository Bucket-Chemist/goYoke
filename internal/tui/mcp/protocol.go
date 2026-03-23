// Package mcp defines the IPC protocol types for communication between
// the gofortress-mcp MCP server and the GOgent-Fortress TUI over a Unix
// domain socket.
//
// The transport is newline-delimited JSON over a single persistent UDS
// connection established at startup.  All messages are correlated by the
// ID field.
package mcp

import "encoding/json"

// IPCRequest is sent from the MCP server to the TUI via UDS.
type IPCRequest struct {
	// Type is the request discriminator (e.g. TypeModalRequest).
	Type string `json:"type"`
	// ID is a unique request identifier used to correlate responses.
	ID string `json:"id"`
	// Payload is the type-specific JSON payload.
	Payload json.RawMessage `json:"payload"`
}

// IPCResponse is sent from the TUI back to the MCP server.
type IPCResponse struct {
	// Type is the response discriminator (e.g. TypeModalResponse).
	Type string `json:"type"`
	// ID matches the originating IPCRequest.ID.
	ID string `json:"id"`
	// Payload is the type-specific JSON payload.
	Payload json.RawMessage `json:"payload"`
}

// Request type constants — sent from MCP server to TUI.
const (
	// TypeModalRequest asks the TUI to display a modal and return the user's
	// selection.
	TypeModalRequest = "modal_request"

	// TypeAgentRegister registers a new subagent in the TUI agent panel.
	TypeAgentRegister = "agent_register"

	// TypeAgentUpdate updates the status of an already-registered subagent.
	TypeAgentUpdate = "agent_update"

	// TypeAgentActivity reports live tool activity for an agent.
	TypeAgentActivity = "agent_activity"

	// TypeToast requests a transient notification toast in the TUI.
	TypeToast = "toast"
)

// Response type constants — sent from TUI back to MCP server.
const (
	// TypeModalResponse carries the user's selection from a modal dialog.
	TypeModalResponse = "modal_response"
)

// ModalRequestPayload is the payload for a TypeModalRequest message.
type ModalRequestPayload struct {
	// Message is the question or prompt displayed to the user.
	Message string `json:"message"`
	// Options is an optional list of button/choice labels.
	// When empty the TUI renders a free-text input.
	Options []string `json:"options,omitempty"`
	// Default is the pre-selected option (must be present in Options).
	Default string `json:"default,omitempty"`
}

// ModalResponsePayload is the payload for a TypeModalResponse message.
type ModalResponsePayload struct {
	// Value is the option label chosen (or free text entered) by the user.
	Value string `json:"value"`
}

// AgentRegisterPayload is the payload for a TypeAgentRegister message.
type AgentRegisterPayload struct {
	// AgentID is the unique identifier for the spawned agent process.
	AgentID string `json:"agentId"`
	// AgentType is the agent definition ID (e.g. "go-pro").
	AgentType string `json:"agentType"`
	// ParentID optionally identifies the spawning agent.
	ParentID string `json:"parentId,omitempty"`
}

// AgentUpdatePayload is the payload for a TypeAgentUpdate message.
type AgentUpdatePayload struct {
	// AgentID identifies the agent whose status changed.
	AgentID string `json:"agentId"`
	// Status is the new lifecycle status (e.g. "running", "done", "error").
	Status string `json:"status"`
}

// AgentActivityPayload is the payload for a TypeAgentActivity message.
type AgentActivityPayload struct {
	// AgentID identifies the agent performing the tool call.
	AgentID string `json:"agentId"`
	// Tool is the name of the tool being invoked.
	Tool string `json:"tool"`
}

// ToastPayload is the payload for a TypeToast message.
type ToastPayload struct {
	// Message is the human-readable notification text.
	Message string `json:"message"`
	// Level is the severity: "info", "warn", or "error".
	Level string `json:"level"`
}
