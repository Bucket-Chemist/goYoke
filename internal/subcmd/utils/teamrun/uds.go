package teamrun

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
)

// IPC message type constants for agent lifecycle notifications.
const (
	typeAgentRegister   = "agent_register"
	typeAgentUpdate     = "agent_update"
	typeAgentActivity   = "agent_activity"
	typeAgentTodoUpdate = "agent_todo_update"
	typeToast           = "toast"
	typeTeamUpdate      = "team_update"
)

// ipcRequest is a fire-and-forget notification sent to the TUI over UDS.
// Mirrors internal/tui/mcp/protocol.go IPCRequest — copied here because
// cmd/ cannot import internal/tui packages.
type ipcRequest struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

// agentRegisterPayload mirrors protocol.go AgentRegisterPayload.
type agentRegisterPayload struct {
	AgentID            string   `json:"agentId"`
	AgentType          string   `json:"agentType"`
	ParentID           string   `json:"parentId,omitempty"`
	Model              string   `json:"model,omitempty"`
	Tier               string   `json:"tier,omitempty"`
	Description        string   `json:"description,omitempty"`
	Conventions        []string `json:"conventions,omitempty"`
	Prompt             string   `json:"prompt,omitempty"`
	AcceptanceCriteria []string `json:"acceptanceCriteria,omitempty"`
}

// agentUpdatePayload mirrors protocol.go AgentUpdatePayload.
type agentUpdatePayload struct {
	AgentID string `json:"agentId"`
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
}

// agentActivityPayload mirrors protocol.go AgentActivityPayload.
type agentActivityPayload struct {
	AgentID string `json:"agentId"`
	Tool    string `json:"tool"`
	Target  string `json:"target,omitempty"`
	Preview string `json:"preview,omitempty"`
}

// toastPayload mirrors protocol.go ToastPayload. Sent to TUI to display
// actionable notifications.
type toastPayload struct {
	Message string `json:"message"`
	Level   string `json:"level"` // "info", "warn", "error"
}

// teamUpdatePayload mirrors protocol.go TeamUpdatePayload. Sent to TUI when a
// team completes or fails so the Teams tab can flash and auto-switch (UX-019).
type teamUpdatePayload struct {
	TeamDir string `json:"teamDir"`
	Status  string `json:"status"`
}

// agentTodoUpdatePayload mirrors protocol.go AgentTodoUpdatePayload.
type agentTodoUpdatePayload struct {
	AgentID string     `json:"agentId"`
	Todos   []todoItem `json:"todos"`
}

// TeamRunUDSClient is a fire-and-forget UDS client for sending agent lifecycle
// notifications to the TUI's IPCBridge.  It is safe to use from multiple
// goroutines.  When socketPath is empty the client operates in no-op mode:
// all calls return immediately without allocating.
type TeamRunUDSClient struct {
	conn   net.Conn
	sendCh chan ipcRequest
	done   chan struct{}
	noop   bool
}

// NewTeamRunUDSClient creates a TeamRunUDSClient.  If socketPath is empty the
// client is created in noop mode and all notify calls are zero-cost no-ops.
// If socketPath is non-empty but the dial fails, a warning is logged and the
// client falls back to noop mode.
func NewTeamRunUDSClient(socketPath string) *TeamRunUDSClient {
	if socketPath == "" {
		return &TeamRunUDSClient{noop: true}
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		slog.Warn("team-run UDS: dial failed, running in noop mode", "socket", socketPath, "err", err)
		return &TeamRunUDSClient{noop: true}
	}

	c := &TeamRunUDSClient{
		conn:   conn,
		sendCh: make(chan ipcRequest, 256),
		done:   make(chan struct{}),
	}
	go c.senderLoop()
	return c
}

// notify marshals payload and enqueues an ipcRequest for delivery.
// Non-blocking: if the send buffer is full the message is dropped with a
// warning.  Calls on a noop client return immediately.
func (c *TeamRunUDSClient) notify(msgType string, payload any) {
	if c.noop {
		return
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("team-run UDS: marshal failed", "type", msgType, "err", err)
		return
	}

	req := ipcRequest{
		Type:    msgType,
		ID:      uuid.New().String(),
		Payload: raw,
	}

	select {
	case c.sendCh <- req:
	default:
		slog.Warn("team-run UDS: send buffer full, dropping message", "type", msgType)
	}
}

// senderLoop drains sendCh and writes each message to the UDS connection.
// It exits when sendCh is closed or a write error occurs.
func (c *TeamRunUDSClient) senderLoop() {
	defer close(c.done)
	enc := json.NewEncoder(c.conn)
	for req := range c.sendCh {
		if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			slog.Warn("team-run UDS: set write deadline failed", "err", err)
		}
		if err := enc.Encode(req); err != nil {
			slog.Warn("team-run UDS: write failed, closing connection", "err", err)
			c.conn.Close()
			return
		}
	}
}

// Close drains remaining messages and closes the connection.  Safe to call on
// a noop client.
func (c *TeamRunUDSClient) Close() {
	if c.noop {
		return
	}
	close(c.sendCh)
	<-c.done
	c.conn.Close()
}

// isNoop reports whether the client is operating in no-op mode.
func (c *TeamRunUDSClient) isNoop() bool {
	return c.noop
}
