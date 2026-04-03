package mcp

import "encoding/json"

// PermGateRequestPayload is the payload for a TypePermGateRequest message.
type PermGateRequestPayload struct {
	// ToolName is the name of the tool awaiting permission.
	ToolName string `json:"tool_name"`
	// ToolInput is the raw JSON arguments passed to the tool.
	ToolInput json.RawMessage `json:"tool_input"`
	// SessionID identifies the agent session requesting permission.
	SessionID string `json:"session_id"`
	// TimeoutMS is how long the TUI should wait for a user decision before
	// applying a default deny. Zero means no timeout.
	TimeoutMS int `json:"timeout_ms"`
}

// PermGateResponsePayload is the payload for a TypePermGateResponse message.
type PermGateResponsePayload struct {
	// Decision is the user's choice: "allow", "deny", or "allow_session".
	Decision string `json:"decision"`
}
