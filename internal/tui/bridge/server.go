// Package bridge implements the TUI-side Unix domain socket server that
// receives IPC requests from the gofortress-mcp MCP server and injects
// them into the Bubbletea event loop via program.Send().
//
// The bridge owns:
//   - One UDS listener at $XDG_RUNTIME_DIR/gofortress-{pid}.sock
//   - One goroutine per accepted connection (MCP server connects once)
//   - A pending modal map for correlating request IDs with response channels
//
// All model state mutations go through the messageSender interface so the
// bridge is testable without a real tea.Program.
package bridge

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/mcp"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

const maxUnixSocketPathBytes = 107

// messageSender is the subset of *tea.Program used by the bridge.
// Defining it as an interface lets tests inject a mock without a real
// Bubbletea program.
type messageSender interface {
	Send(msg tea.Msg)
}

// IPCBridge is the TUI-side Unix domain socket server.
// Its zero value is not usable; use NewIPCBridge instead.
type IPCBridge struct {
	socketPath       string
	listener         net.Listener
	sender           messageSender
	pendingModals    map[string]chan mcp.ModalResponsePayload
	pendingPermGates map[string]chan mcp.PermGateResponsePayload
	mu               sync.Mutex
	done             chan struct{}
}

// NewIPCBridge creates and binds a new IPCBridge.
//
// The socket path is:
//
//	$XDG_RUNTIME_DIR/gofortress-{pid}.sock   (preferred)
//	$TMPDIR/gofortress-{pid}.sock            (fallback)
//	/tmp/gofortress-{pid}.sock               (short-path fallback)
//
// Any stale socket at that path is removed before binding.
func NewIPCBridge(sender messageSender) (*IPCBridge, error) {
	socketPath := buildSocketPath()

	// Remove a stale socket left by a previous (crashed) process.
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove stale socket %s: %w", socketPath, err)
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix %s: %w", socketPath, err)
	}

	return &IPCBridge{
		socketPath:       socketPath,
		listener:         ln,
		sender:           sender,
		pendingModals:    make(map[string]chan mcp.ModalResponsePayload),
		pendingPermGates: make(map[string]chan mcp.PermGateResponsePayload),
		done:             make(chan struct{}),
	}, nil
}

// buildSocketPath returns the socket path for the current PID.
func buildSocketPath() string {
	filename := fmt.Sprintf("gofortress-%d.sock", os.Getpid())
	for _, base := range socketBaseCandidates() {
		path := filepath.Join(base, filename)
		if len(path) <= maxUnixSocketPathBytes {
			return path
		}
	}
	return filepath.Join("/tmp", filename)
}

func socketBaseCandidates() []string {
	candidates := []string{
		os.Getenv("XDG_RUNTIME_DIR"),
		os.TempDir(),
		"/tmp",
	}

	seen := make(map[string]struct{}, len(candidates))
	bases := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		bases = append(bases, cleaned)
	}
	return bases
}

// SocketPath returns the absolute path of the UDS so callers can set
// GOFORTRESS_SOCKET in child process environments.
func (b *IPCBridge) SocketPath() string {
	return b.socketPath
}

// Start launches the accept loop in a background goroutine.
// Each accepted connection is handled in its own goroutine.
// Start returns immediately; call Shutdown to stop the bridge.
func (b *IPCBridge) Start() {
	go b.acceptLoop()
}

// acceptLoop blocks on Accept until the listener is closed.
//
// Transient accept errors (e.g. EMFILE / ENFILE when the file-descriptor table
// is momentarily exhausted) are logged and retried after a brief backoff rather
// than causing the loop to exit permanently. Only a shutdown signal (b.done
// closed) causes the loop to exit.
func (b *IPCBridge) acceptLoop() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			// Listener closed via Shutdown — exit cleanly.
			select {
			case <-b.done:
				return
			default:
			}
			// Transient error: log, back off briefly, and retry.
			slog.Warn("bridge: accept error, retrying", "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go b.handleConnection(conn)
	}
}

// handleConnection processes all IPC requests from a single connection.
// It runs until the connection is closed (EOF) or the bridge shuts down.
func (b *IPCBridge) handleConnection(conn net.Conn) {
	defer conn.Close()

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req mcp.IPCRequest
		if err := dec.Decode(&req); err != nil {
			// EOF or closed connection — normal termination.
			return
		}

		b.dispatch(req, enc)
	}
}

// dispatch routes a decoded request to the appropriate handler.
func (b *IPCBridge) dispatch(req mcp.IPCRequest, enc *json.Encoder) {
	switch req.Type {
	case mcp.TypeModalRequest:
		b.handleModal(req, enc)
	case mcp.TypeAgentRegister:
		b.handleAgentRegister(req)
	case mcp.TypeAgentUpdate:
		b.handleAgentUpdate(req)
	case mcp.TypeAgentActivity:
		b.handleAgentActivity(req)
	case mcp.TypeToast:
		b.handleToast(req)
	case mcp.TypePermGateRequest:
		b.handlePermGate(req, enc)
	case mcp.TypeAgentTodoUpdate:
		b.handleAgentTodoUpdate(req)
	case mcp.TypeTeamUpdate:
		b.handleTeamUpdate(req)
	default:
		slog.Warn("bridge: unknown request type", "type", req.Type, "id", req.ID)
	}
}

// handleModal processes a modal_request: it injects a BridgeModalRequestMsg
// into the Bubbletea event loop and blocks until ResolveModal delivers the
// user's response (or Shutdown cancels the wait).
func (b *IPCBridge) handleModal(req mcp.IPCRequest, enc *json.Encoder) {
	var payload mcp.ModalRequestPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		slog.Error("bridge: unmarshal modal_request payload", "id", req.ID, "error", err)
		return
	}

	ch := make(chan mcp.ModalResponsePayload, 1)

	b.mu.Lock()
	b.pendingModals[req.ID] = ch
	b.mu.Unlock()

	// Inject the request into the Bubbletea event loop.
	b.sender.Send(model.BridgeModalRequestMsg{
		RequestID: req.ID,
		Message:   payload.Message,
		Options:   payload.Options,
	})

	// Block until the user responds or the bridge shuts down.
	var modalResp mcp.ModalResponsePayload
	select {
	case resp, ok := <-ch:
		if !ok {
			// Channel closed by Shutdown; send empty response.
			slog.Info("bridge: modal cancelled by shutdown", "id", req.ID)
			return
		}
		modalResp = resp
	case <-b.done:
		// Bridge is shutting down; clean up and return without sending.
		b.mu.Lock()
		delete(b.pendingModals, req.ID)
		b.mu.Unlock()
		return
	}

	// Marshal the response payload.
	rawPayload, err := json.Marshal(modalResp)
	if err != nil {
		slog.Error("bridge: marshal modal response payload", "id", req.ID, "error", err)
		return
	}

	resp := mcp.IPCResponse{
		Type:    mcp.TypeModalResponse,
		ID:      req.ID,
		Payload: json.RawMessage(rawPayload),
	}
	if err := enc.Encode(resp); err != nil {
		slog.Warn("bridge: write modal response", "id", req.ID, "error", err)
	}

	b.mu.Lock()
	delete(b.pendingModals, req.ID)
	b.mu.Unlock()
}

// handleAgentRegister processes an agent_register request (fire-and-forget).
func (b *IPCBridge) handleAgentRegister(req mcp.IPCRequest) {
	var p mcp.AgentRegisterPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal agent_register payload", "id", req.ID, "error", err)
		return
	}
	b.sender.Send(model.AgentRegisteredMsg{
		AgentID:            p.AgentID,
		AgentType:          p.AgentType,
		ParentID:           p.ParentID,
		Model:              p.Model,
		Tier:               p.Tier,
		Description:        p.Description,
		Conventions:        p.Conventions,
		Prompt:             p.Prompt,
		AcceptanceCriteria: p.AcceptanceCriteria,
	})
}

// handleAgentUpdate processes an agent_update request (fire-and-forget).
func (b *IPCBridge) handleAgentUpdate(req mcp.IPCRequest) {
	var p mcp.AgentUpdatePayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal agent_update payload", "id", req.ID, "error", err)
		return
	}
	b.sender.Send(model.AgentUpdatedMsg{
		AgentID: p.AgentID,
		Status:  p.Status,
		PID:     p.PID,
	})
}

// handleAgentActivity processes an agent_activity request (fire-and-forget).
func (b *IPCBridge) handleAgentActivity(req mcp.IPCRequest) {
	var p mcp.AgentActivityPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal agent_activity payload", "id", req.ID, "error", err)
		return
	}
	b.sender.Send(model.AgentActivityMsg{
		AgentID:  p.AgentID,
		ToolName: p.Tool,
		Target:   p.Target,
		Preview:  p.Preview,
	})
}

// handleAgentTodoUpdate processes an agent_todo_update request (fire-and-forget).
func (b *IPCBridge) handleAgentTodoUpdate(req mcp.IPCRequest) {
	var p mcp.AgentTodoUpdatePayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal agent_todo_update payload", "id", req.ID, "error", err)
		return
	}
	todos := make([]model.AgentTodoItem, len(p.Todos))
	for i, t := range p.Todos {
		todos[i] = model.AgentTodoItem{Content: t.Content, Status: t.Status}
	}
	b.sender.Send(model.AgentTodoUpdateMsg{
		AgentID: p.AgentID,
		Todos:   todos,
	})
}

// handleToast processes a toast request (fire-and-forget).
func (b *IPCBridge) handleToast(req mcp.IPCRequest) {
	var p mcp.ToastPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal toast payload", "id", req.ID, "error", err)
		return
	}
	b.sender.Send(model.ToastMsg{
		Text:  p.Message,
		Level: model.ToastLevel(p.Level),
	})
}

// handleTeamUpdate processes a team_update notification (fire-and-forget).
// It sends a TeamUpdateMsg to the Bubbletea event loop so the teams drawer
// can scan immediately and auto-expand.
func (b *IPCBridge) handleTeamUpdate(req mcp.IPCRequest) {
	var p mcp.TeamUpdatePayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		slog.Error("bridge: unmarshal team_update payload", "id", req.ID, "error", err)
		return
	}
	b.sender.Send(model.TeamUpdateMsg{
		TeamDir: p.TeamDir,
		Status:  p.Status,
	})
}

// handlePermGate processes a permission_gate_request: it injects a
// CLIPermissionRequestMsg into the Bubbletea event loop and blocks until
// ResolvePermGate delivers the user's decision (or the request times out,
// or Shutdown cancels the wait).
func (b *IPCBridge) handlePermGate(req mcp.IPCRequest, enc *json.Encoder) {
	var payload mcp.PermGateRequestPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		slog.Error("bridge: unmarshal permission_gate_request payload", "id", req.ID, "error", err)
		return
	}

	timeoutMS := payload.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30_000 // default 30 s
	}

	ch := make(chan mcp.PermGateResponsePayload, 1)

	b.mu.Lock()
	b.pendingPermGates[req.ID] = ch
	b.mu.Unlock()

	// Inject the request into the Bubbletea event loop.
	b.sender.Send(model.CLIPermissionRequestMsg{
		RequestID: req.ID,
		ToolName:  payload.ToolName,
		ToolInput: payload.ToolInput,
		TimeoutMS: timeoutMS,
	})

	// Default deny payload used on timeout.
	denyPayload := mcp.PermGateResponsePayload{Decision: "deny"}

	// Block until the user responds, the timeout fires, or the bridge shuts down.
	var permResp mcp.PermGateResponsePayload
	select {
	case resp, ok := <-ch:
		if !ok {
			// Channel closed by Shutdown; return without sending a response.
			slog.Info("bridge: permission gate cancelled by shutdown", "id", req.ID)
			return
		}
		permResp = resp
	case <-time.After(time.Duration(timeoutMS) * time.Millisecond):
		slog.Info("bridge: permission gate timed out, denying", "id", req.ID)
		permResp = denyPayload
	case <-b.done:
		// Bridge is shutting down; clean up and return without sending.
		b.mu.Lock()
		delete(b.pendingPermGates, req.ID)
		b.mu.Unlock()
		return
	}

	// C-1 fix: Remove the map entry BEFORE sending the response. This prevents
	// a concurrent ResolvePermGate from writing "allow" to the channel after the
	// timeout branch already consumed the deny value.
	b.mu.Lock()
	delete(b.pendingPermGates, req.ID)
	b.mu.Unlock()

	// Marshal the response payload.
	rawPayload, err := json.Marshal(permResp)
	if err != nil {
		slog.Error("bridge: marshal permission gate response payload", "id", req.ID, "error", err)
		return
	}

	resp := mcp.IPCResponse{
		Type:    mcp.TypePermGateResponse,
		ID:      req.ID,
		Payload: json.RawMessage(rawPayload),
	}
	if err := enc.Encode(resp); err != nil {
		slog.Warn("bridge: write permission gate response", "id", req.ID, "error", err)
	}
}

// ResolvePermGate is called by AppModel.Update when the user makes a
// permission decision. It delivers the response to the goroutine blocking
// in handlePermGate.
//
// The send is non-blocking (select/default): if the buffered channel is already
// full (e.g. a duplicate call for the same requestID), the response is dropped
// and a warning is logged instead of deadlocking while holding b.mu.
func (b *IPCBridge) ResolvePermGate(requestID string, decision string) {
	b.mu.Lock()
	ch, ok := b.pendingPermGates[requestID]
	if ok {
		select {
		case ch <- mcp.PermGateResponsePayload{Decision: decision}:
		default:
			slog.Warn("bridge: ResolvePermGate channel full, response dropped", "id", requestID)
		}
		delete(b.pendingPermGates, requestID)
	}
	b.mu.Unlock()
}

// ResolveModal is called by AppModel.Update when the user makes a modal
// selection. It delivers the response to the goroutine blocking in handleModal.
//
// The send is non-blocking (select/default): if the buffered channel is already
// full (e.g. a duplicate call for the same requestID), the response is dropped
// and a warning is logged instead of deadlocking while holding b.mu.
func (b *IPCBridge) ResolveModal(requestID string, response mcp.ModalResponsePayload) {
	b.mu.Lock()
	ch, ok := b.pendingModals[requestID]
	if ok {
		select {
		case ch <- response:
		default:
			slog.Warn("bridge: ResolveModal channel full, response dropped", "id", requestID)
		}
		delete(b.pendingModals, requestID)
	}
	b.mu.Unlock()
}

// ResolveModalSimple is a convenience wrapper around ResolveModal that accepts
// a plain string value instead of a mcp.ModalResponsePayload.  It is called by
// AppModel via the bridgeWidget interface, which cannot reference the mcp
// package directly without creating a circular import.
func (b *IPCBridge) ResolveModalSimple(requestID string, value string) {
	b.ResolveModal(requestID, mcp.ModalResponsePayload{Value: value})
}

// Shutdown stops the bridge: it signals all blocked modal handlers,
// closes the listener, removes the socket file, and drains the pending
// modal map.
func (b *IPCBridge) Shutdown() {
	// Signal all blocked handleModal goroutines to return.
	close(b.done)

	// Stop accepting new connections.
	b.listener.Close()

	// Remove the socket file.
	if err := os.Remove(b.socketPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("bridge: remove socket on shutdown", "path", b.socketPath, "error", err)
	}

	// Drain any channels still registered in the pending modal map.
	// A non-blocking send is used instead of close(ch) to eliminate the
	// double-close panic window: handleModal may have already received via
	// <-b.done and deleted its entry, or ResolveModal may have already sent
	// a value. The buffered channel (size 1) absorbs the signal if handleModal
	// has not yet read from it; the default branch is a safe no-op if it has.
	b.mu.Lock()
	for id, ch := range b.pendingModals {
		select {
		case ch <- mcp.ModalResponsePayload{}:
		default:
		}
		delete(b.pendingModals, id)
	}
	// Drain any channels still registered in the pending permission gate map.
	// Same reasoning as above — non-blocking send avoids double-close panics.
	for id, ch := range b.pendingPermGates {
		select {
		case ch <- mcp.PermGateResponsePayload{}:
		default:
		}
		delete(b.pendingPermGates, id)
	}
	b.mu.Unlock()
}
