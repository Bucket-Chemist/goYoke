package bridge

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Mock messageSender
// ---------------------------------------------------------------------------

// mockSender captures every message sent via Send() for later inspection.
type mockSender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (m *mockSender) Send(msg tea.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
}

type stubListener struct {
	closeCount int
}

func (l *stubListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("stub listener closed")
}

func (l *stubListener) Close() error {
	l.closeCount++
	return nil
}

func (l *stubListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "stub", Net: "unix"}
}

// waitFor blocks until at least one message of type T is received or the
// timeout expires.  It returns the first matching message and true, or the
// zero value and false on timeout.
func waitFor[T any](ms *mockSender, timeout time.Duration) (T, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		for _, msg := range ms.msgs {
			if v, ok := msg.(T); ok {
				ms.mu.Unlock()
				return v, true
			}
		}
		ms.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	var zero T
	return zero, false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testTimeout = 2 * time.Second

// startBridge creates a bridge with a mock sender and starts it.
// The caller is responsible for calling b.Shutdown().
func startBridge(t *testing.T) (*IPCBridge, *mockSender) {
	t.Helper()
	ms := &mockSender{}
	b, err := NewIPCBridge(ms)
	require.NoError(t, err)
	b.Start()
	return b, ms
}

// dialBridge opens a JSON-over-UDS connection to the bridge socket.
func dialBridge(t *testing.T, socketPath string) (net.Conn, *json.Encoder, *json.Decoder) {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	return conn, json.NewEncoder(conn), json.NewDecoder(conn)
}

// sendRequest marshals req and writes it to enc.
func sendRequest(t *testing.T, enc *json.Encoder, req mcp.IPCRequest) {
	t.Helper()
	require.NoError(t, enc.Encode(req))
}

// buildRequest builds an IPCRequest with a JSON-marshalled payload.
func buildRequest(t *testing.T, typ, id string, payload any) mcp.IPCRequest {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return mcp.IPCRequest{Type: typ, ID: id, Payload: json.RawMessage(raw)}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestModalRequestResponseRoundtrip verifies the full modal flow:
//  1. Client sends modal_request
//  2. Bridge delivers BridgeModalRequestMsg to the sender
//  3. Test calls ResolveModal (simulating user input)
//  4. Bridge sends IPCResponse back to the client
func TestModalRequestResponseRoundtrip(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	// Send a modal_request.
	req := buildRequest(t, mcp.TypeModalRequest, "req-modal-1", mcp.ModalRequestPayload{
		Message: "Allow Write to /tmp/foo?",
		Options: []string{"Allow", "Allow Always", "Deny"},
	})
	sendRequest(t, enc, req)

	// Wait for the BridgeModalRequestMsg to arrive via program.Send().
	bridgeMsg, ok := waitFor[model.BridgeModalRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Equal(t, "req-modal-1", bridgeMsg.RequestID)
	assert.Equal(t, "Allow Write to /tmp/foo?", bridgeMsg.Message)
	assert.Equal(t, []string{"Allow", "Allow Always", "Deny"}, bridgeMsg.Options)

	// Simulate the user selecting "Allow".
	go b.ResolveModal("req-modal-1", mcp.ModalResponsePayload{Value: "Allow"})

	// Expect an IPCResponse with the selected value on the connection.
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp))

	assert.Equal(t, mcp.TypeModalResponse, resp.Type)
	assert.Equal(t, "req-modal-1", resp.ID)

	var respPayload mcp.ModalResponsePayload
	require.NoError(t, json.Unmarshal(resp.Payload, &respPayload))
	assert.Equal(t, "Allow", respPayload.Value)
}

// TestAgentRegisterDelivery verifies that agent_register requests are
// forwarded as model.AgentRegisteredMsg.
func TestAgentRegisterDelivery(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypeAgentRegister, "req-reg-1", mcp.AgentRegisterPayload{
		AgentID:   "agent-42",
		AgentType: "go-pro",
		ParentID:  "agent-root",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.AgentRegisteredMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for AgentRegisteredMsg")
	assert.Equal(t, "agent-42", msg.AgentID)
	assert.Equal(t, "go-pro", msg.AgentType)
	assert.Equal(t, "agent-root", msg.ParentID)
}

// TestAgentUpdateDelivery verifies that agent_update requests are forwarded
// as model.AgentUpdatedMsg.
func TestAgentUpdateDelivery(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypeAgentUpdate, "req-upd-1", mcp.AgentUpdatePayload{
		AgentID: "agent-42",
		Status:  "done",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.AgentUpdatedMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for AgentUpdatedMsg")
	assert.Equal(t, "agent-42", msg.AgentID)
	assert.Equal(t, "done", msg.Status)
}

// TestAgentActivityDelivery verifies that agent_activity requests are
// forwarded as model.AgentActivityMsg.
func TestAgentActivityDelivery(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypeAgentActivity, "req-act-1", mcp.AgentActivityPayload{
		AgentID: "agent-42",
		Tool:    "Write",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.AgentActivityMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for AgentActivityMsg")
	assert.Equal(t, "agent-42", msg.AgentID)
	assert.Equal(t, "Write", msg.ToolName)
}

// TestToastDelivery verifies that toast requests are forwarded as
// model.ToastMsg.
func TestToastDelivery(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypeToast, "req-toast-1", mcp.ToastPayload{
		Message: "Agent completed",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for ToastMsg")
	assert.Equal(t, "Agent completed", msg.Text)
	assert.Equal(t, model.ToastLevelInfo, msg.Level)
}

// TestShutdownCancelsPendingModal verifies that Shutdown unblocks a goroutine
// waiting for a modal response: the modal response channel is closed, and the
// server-side handler returns without writing to the connection.
func TestShutdownCancelsPendingModal(t *testing.T) {
	b, ms := startBridge(t)

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	// Send a modal_request.
	req := buildRequest(t, mcp.TypeModalRequest, "req-shutdown-1", mcp.ModalRequestPayload{
		Message: "Continue?",
		Options: []string{"Yes", "No"},
	})
	sendRequest(t, enc, req)

	// Wait until the bridge has sent the BridgeModalRequestMsg (i.e. the
	// handleModal goroutine is now blocking on the response channel).
	_, ok := waitFor[model.BridgeModalRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg before shutdown")

	// Shutdown the bridge.  The modal handler goroutine should unblock via
	// the b.done channel and NOT write a response.
	shutdownDone := make(chan struct{})
	go func() {
		b.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		// Good — Shutdown returned.
	case <-time.After(testTimeout):
		t.Fatal("Shutdown timed out")
	}

	// Set a short read deadline on the connection.  We expect either EOF or a
	// timeout — not a valid response, because Shutdown cancelled the modal.
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var resp mcp.IPCResponse
	err := dec.Decode(&resp)
	// We expect an error (EOF / timeout / closed) — not a successful decode.
	assert.Error(t, err, "expected no response after shutdown-cancelled modal")
}

// TestMultipleConcurrentConnections verifies that the bridge handles multiple
// simultaneous connections, each sending independent fire-and-forget messages.
func TestMultipleConcurrentConnections(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	const numConns = 5
	var wg sync.WaitGroup

	for i := range numConns {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			conn, enc, _ := dialBridge(t, b.SocketPath())
			defer conn.Close()

			req := buildRequest(t, mcp.TypeToast, "", mcp.ToastPayload{
				Message: "msg from conn",
				Level:   "info",
			})
			sendRequest(t, enc, req)
			// Give the bridge a moment to process.
			time.Sleep(20 * time.Millisecond)
			_ = idx
		}(i)
	}

	wg.Wait()

	// All numConns toast messages should eventually arrive.
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		count := 0
		for _, msg := range ms.msgs {
			if _, ok := msg.(model.ToastMsg); ok {
				count++
			}
		}
		ms.mu.Unlock()
		if count >= numConns {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("expected %d ToastMsg, got fewer within timeout", numConns)
}

// TestSocketPathIsSetCorrectly verifies that SocketPath returns a non-empty
// string that reflects the current PID.
func TestSocketPathIsSetCorrectly(t *testing.T) {
	b, _ := startBridge(t)
	defer b.Shutdown()

	path := b.SocketPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "goyoke-")
	assert.Contains(t, path, ".sock")
}

func TestBuildSocketPath_FallsBackToTmpWhenCandidatesAreTooLong(t *testing.T) {
	longBase := filepath.Join("/tmp", strings.Repeat("x", 120))
	t.Setenv("XDG_RUNTIME_DIR", longBase)
	t.Setenv("TMPDIR", longBase)

	path := buildSocketPath()

	assert.Equal(t, filepath.Join("/tmp", fmt.Sprintf("goyoke-%d.sock", os.Getpid())), path)
	assert.LessOrEqual(t, len(path), maxUnixSocketPathBytes)
}

// TestUnknownRequestTypeIsIgnored verifies that an unrecognised request type
// does not cause a panic or crash the connection handler.
func TestUnknownRequestTypeIsIgnored(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	// Send an unknown type followed by a known type.
	unknown := mcp.IPCRequest{
		Type:    "unknown_type",
		ID:      "req-unknown-1",
		Payload: json.RawMessage(`{}`),
	}
	require.NoError(t, enc.Encode(unknown))

	// Now send a known toast so we can confirm the handler is still alive.
	req := buildRequest(t, mcp.TypeToast, "req-toast-after", mcp.ToastPayload{
		Message: "still alive",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "timed out — connection dropped after unknown request type")
	assert.Equal(t, "still alive", msg.Text)
}

// TestResolveModalNoopOnMissingID verifies that calling ResolveModal with a
// non-existent request ID does not panic.
func TestResolveModalNoopOnMissingID(t *testing.T) {
	b, _ := startBridge(t)
	defer b.Shutdown()

	// Should not panic.
	assert.NotPanics(t, func() {
		b.ResolveModal("nonexistent-id", mcp.ModalResponsePayload{Value: "ok"})
	})
}

// ---------------------------------------------------------------------------
// W-4: ResolveModal non-blocking send
// ---------------------------------------------------------------------------

// TestResolveModal_DuplicateCallDoesNotDeadlock verifies that calling
// ResolveModal twice for the same requestID does not deadlock.  The buffered
// channel holds one response; the second call hits the default branch and
// drops the duplicate, logging a warning instead of blocking while holding
// b.mu.
func TestResolveModal_DuplicateCallDoesNotDeadlock(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	// Send a modal_request.
	req := buildRequest(t, mcp.TypeModalRequest, "req-dup-1", mcp.ModalRequestPayload{
		Message: "Duplicate test?",
		Options: []string{"Yes", "No"},
	})
	sendRequest(t, enc, req)

	// Wait for the BridgeModalRequestMsg so the channel is registered.
	_, ok := waitFor[model.BridgeModalRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")

	// First call delivers the response.
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.ResolveModal("req-dup-1", mcp.ModalResponsePayload{Value: "Yes"})
		// Second call must not deadlock or panic — the channel is already full
		// (or the first call already drained it and deleted the entry).
		b.ResolveModal("req-dup-1", mcp.ModalResponsePayload{Value: "duplicate"})
	}()

	select {
	case <-done:
		// Good — both calls returned without deadlocking.
	case <-time.After(testTimeout):
		t.Fatal("ResolveModal deadlocked on duplicate call")
	}

	// The first response should arrive on the connection.
	conn.SetReadDeadline(time.Now().Add(testTimeout))
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp))
	assert.Equal(t, mcp.TypeModalResponse, resp.Type)
}

// ---------------------------------------------------------------------------
// W-5/M-4: acceptLoop continues on transient errors
// ---------------------------------------------------------------------------

// TestAcceptLoop_ContinuesAfterShutdown verifies that closing the listener
// via Shutdown causes acceptLoop to return (not retry indefinitely).
// This is the existing shutdown path — keeping it green after the retry
// change.
func TestAcceptLoop_ContinuesAfterShutdown(t *testing.T) {
	b, _ := startBridge(t)

	shutdownDone := make(chan struct{})
	go func() {
		b.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		// Good — Shutdown returned, meaning acceptLoop exited.
	case <-time.After(testTimeout):
		t.Fatal("Shutdown timed out — acceptLoop may be looping on transient errors")
	}
}

func TestShutdown_IsIdempotent(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "goyoke-test.sock")
	require.NoError(t, os.WriteFile(socketPath, []byte("stale"), 0o600))
	listener := &stubListener{}
	b := &IPCBridge{
		socketPath:       socketPath,
		listener:         listener,
		pendingModals:    make(map[string]chan mcp.ModalResponsePayload),
		pendingPermGates: make(map[string]chan mcp.PermGateResponsePayload),
		done:             make(chan struct{}),
	}

	require.NotPanics(t, func() {
		b.Shutdown()
		b.Shutdown()
	})
	assert.Equal(t, 1, listener.closeCount, "listener should only close once")
	_, err := os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err), "socket path should be removed on first shutdown")
}

// TestAcceptLoop_NewConnectionAfterTransientError verifies that the bridge
// can still accept new connections after a transient error would have
// previously caused the acceptLoop to exit.  We test this by starting a
// bridge, accepting a first connection, and then establishing a second
// connection to confirm the loop is still running.
func TestAcceptLoop_NewConnectionAfterTransientError(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	// First connection — verify bridge is working.
	conn1, enc1, _ := dialBridge(t, b.SocketPath())
	defer conn1.Close()

	req1 := buildRequest(t, mcp.TypeToast, "req-first", mcp.ToastPayload{
		Message: "first connection",
		Level:   "info",
	})
	sendRequest(t, enc1, req1)

	_, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for first ToastMsg")

	// Second connection — verifies the acceptLoop is still running.
	conn2, enc2, _ := dialBridge(t, b.SocketPath())
	defer conn2.Close()

	req2 := buildRequest(t, mcp.TypeToast, "req-second", mcp.ToastPayload{
		Message: "second connection",
		Level:   "info",
	})
	sendRequest(t, enc2, req2)

	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		count := 0
		for _, msg := range ms.msgs {
			if tm, ok := msg.(model.ToastMsg); ok && tm.Text == "second connection" {
				count++
			}
		}
		ms.mu.Unlock()
		if count > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("second connection message never arrived — acceptLoop may have exited prematurely")
}

// ---------------------------------------------------------------------------
// TUI-036: Coverage gap-filling for bridge package
// ---------------------------------------------------------------------------

// TestResolveModalSimple_RoutesCorrectly verifies that ResolveModalSimple
// delivers the response to a pending modal goroutine (the 0% function).
func TestResolveModalSimple_RoutesCorrectly(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypeModalRequest, "req-simple-1", mcp.ModalRequestPayload{
		Message: "Simple?",
		Options: []string{"Yes", "No"},
	})
	sendRequest(t, enc, req)

	_, ok := waitFor[model.BridgeModalRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")

	// Use ResolveModalSimple instead of ResolveModal.
	go b.ResolveModalSimple("req-simple-1", "Yes")

	conn.SetReadDeadline(time.Now().Add(testTimeout))
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp))

	assert.Equal(t, mcp.TypeModalResponse, resp.Type)
	assert.Equal(t, "req-simple-1", resp.ID)

	var payload mcp.ModalResponsePayload
	require.NoError(t, json.Unmarshal(resp.Payload, &payload))
	assert.Equal(t, "Yes", payload.Value)
}

// TestHandleAgentRegister_MalformedPayload_NoPanic verifies that a
// malformed payload in an agent_register request does not crash the handler
// (the error branch at 60%).
func TestHandleAgentRegister_MalformedPayload_NoPanic(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	// Send a malformed payload (valid JSON but wrong type for AgentRegisterPayload).
	malformed := mcp.IPCRequest{
		Type:    mcp.TypeAgentRegister,
		ID:      "bad-reg-1",
		Payload: json.RawMessage(`"not an object"`),
	}
	require.NoError(t, enc.Encode(malformed))

	// After the error, send a valid toast to confirm the connection is still alive.
	req := buildRequest(t, mcp.TypeToast, "after-bad-reg", mcp.ToastPayload{
		Message: "still alive after bad register",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "handler crashed after malformed agent_register")
	assert.Equal(t, "still alive after bad register", msg.Text)
}

// TestHandleAgentUpdate_MalformedPayload_NoPanic mirrors the above for
// agent_update.
func TestHandleAgentUpdate_MalformedPayload_NoPanic(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	malformed := mcp.IPCRequest{
		Type:    mcp.TypeAgentUpdate,
		ID:      "bad-upd-1",
		Payload: json.RawMessage(`"not an object"`),
	}
	require.NoError(t, enc.Encode(malformed))

	req := buildRequest(t, mcp.TypeToast, "after-bad-upd", mcp.ToastPayload{
		Message: "still alive after bad update",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "handler crashed after malformed agent_update")
	assert.Equal(t, "still alive after bad update", msg.Text)
}

// TestHandleAgentActivity_MalformedPayload_NoPanic mirrors the above for
// agent_activity.
func TestHandleAgentActivity_MalformedPayload_NoPanic(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	malformed := mcp.IPCRequest{
		Type:    mcp.TypeAgentActivity,
		ID:      "bad-act-1",
		Payload: json.RawMessage(`"not an object"`),
	}
	require.NoError(t, enc.Encode(malformed))

	req := buildRequest(t, mcp.TypeToast, "after-bad-act", mcp.ToastPayload{
		Message: "still alive after bad activity",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	msg, ok := waitFor[model.ToastMsg](ms, testTimeout)
	require.True(t, ok, "handler crashed after malformed agent_activity")
	assert.Equal(t, "still alive after bad activity", msg.Text)
}

// TestHandleToast_MalformedPayload_NoPanic verifies that a malformed toast
// payload does not crash the connection handler.
func TestHandleToast_MalformedPayload_NoPanic(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, _ := dialBridge(t, b.SocketPath())
	defer conn.Close()

	malformed := mcp.IPCRequest{
		Type:    mcp.TypeToast,
		ID:      "bad-toast-1",
		Payload: json.RawMessage(`"not an object"`),
	}
	require.NoError(t, enc.Encode(malformed))

	// Send a second valid toast to confirm the connection stayed alive.
	req := buildRequest(t, mcp.TypeToast, "after-bad-toast", mcp.ToastPayload{
		Message: "still alive after bad toast",
		Level:   "info",
	})
	sendRequest(t, enc, req)

	// Wait for the second (valid) toast to arrive.
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		for _, msg := range ms.msgs {
			if tm, ok := msg.(model.ToastMsg); ok && tm.Text == "still alive after bad toast" {
				ms.mu.Unlock()
				return
			}
		}
		ms.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("handler crashed after malformed toast payload")
}

// ---------------------------------------------------------------------------
// PERM-002: permission_gate_request tests
// ---------------------------------------------------------------------------

// TestPermGateRequestResponseRoundtrip verifies the full permission gate flow:
//  1. Client sends permission_gate_request
//  2. Bridge delivers CLIPermissionRequestMsg to the sender
//  3. Test calls ResolvePermGate("allow") simulating a user decision
//  4. Bridge sends TypePermGateResponse back to the client
func TestPermGateRequestResponseRoundtrip(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	toolInput := json.RawMessage(`{"command":"ls -la"}`)
	req := buildRequest(t, mcp.TypePermGateRequest, "req-perm-1", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: toolInput,
		SessionID: "session-42",
		TimeoutMS: 5000,
	})
	sendRequest(t, enc, req)

	// Wait for CLIPermissionRequestMsg to arrive via program.Send().
	permMsg, ok := waitFor[model.CLIPermissionRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")
	assert.Equal(t, "req-perm-1", permMsg.RequestID)
	assert.Equal(t, "Bash", permMsg.ToolName)
	assert.Equal(t, 5000, permMsg.TimeoutMS)

	// Simulate the user selecting "allow".
	go b.ResolvePermGate("req-perm-1", "allow")

	// Expect a TypePermGateResponse on the connection.
	conn.SetReadDeadline(time.Now().Add(testTimeout))
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp))

	assert.Equal(t, mcp.TypePermGateResponse, resp.Type)
	assert.Equal(t, "req-perm-1", resp.ID)

	var respPayload mcp.PermGateResponsePayload
	require.NoError(t, json.Unmarshal(resp.Payload, &respPayload))
	assert.Equal(t, "allow", respPayload.Decision)
}

// TestPermGateTimeout verifies that a permission gate request with a short
// TimeoutMS is automatically denied when no ResolvePermGate call is made.
func TestPermGateTimeout(t *testing.T) {
	b, ms := startBridge(t)
	defer b.Shutdown()

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypePermGateRequest, "req-perm-timeout", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{}`),
		TimeoutMS: 100, // 100 ms — will fire before test timeout
	})
	sendRequest(t, enc, req)

	// Wait for CLIPermissionRequestMsg to confirm the request was received.
	_, ok := waitFor[model.CLIPermissionRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")

	// Do NOT call ResolvePermGate — wait for the auto-deny.
	conn.SetReadDeadline(time.Now().Add(testTimeout))
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp))

	assert.Equal(t, mcp.TypePermGateResponse, resp.Type)
	assert.Equal(t, "req-perm-timeout", resp.ID)

	var respPayload mcp.PermGateResponsePayload
	require.NoError(t, json.Unmarshal(resp.Payload, &respPayload))
	assert.Equal(t, "deny", respPayload.Decision)
}

// TestPermGateShutdown verifies that calling Shutdown() while a permission
// gate request is pending unblocks the handler cleanly without writing a
// response to the connection.
func TestPermGateShutdown(t *testing.T) {
	b, ms := startBridge(t)

	conn, enc, dec := dialBridge(t, b.SocketPath())
	defer conn.Close()

	req := buildRequest(t, mcp.TypePermGateRequest, "req-perm-shutdown", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{}`),
		TimeoutMS: 30000,
	})
	sendRequest(t, enc, req)

	// Wait until the bridge has injected CLIPermissionRequestMsg (i.e. the
	// handlePermGate goroutine is now blocking on the response channel).
	_, ok := waitFor[model.CLIPermissionRequestMsg](ms, testTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg before shutdown")

	// Shutdown the bridge. The perm gate handler goroutine should unblock via
	// the b.done channel and NOT write a response.
	shutdownDone := make(chan struct{})
	go func() {
		b.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		// Good — Shutdown returned.
	case <-time.After(testTimeout):
		t.Fatal("Shutdown timed out")
	}

	// Set a short read deadline. We expect EOF or timeout — not a valid response.
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var resp mcp.IPCResponse
	err := dec.Decode(&resp)
	assert.Error(t, err, "expected no response after shutdown-cancelled permission gate")
}

// TestNewIPCBridge_RemovesStaleSocket verifies that NewIPCBridge succeeds
// even when a socket file already exists at the target path (stale socket
// removal code path).
func TestNewIPCBridge_RemovesStaleSocket(t *testing.T) {
	// Point XDG_RUNTIME_DIR to a temp dir so we control the socket path.
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	// Pre-create a file at the path the bridge would use.
	stalePath := filepath.Join(dir, fmt.Sprintf("goyoke-%d.sock", os.Getpid()))
	require.NoError(t, os.WriteFile(stalePath, []byte("stale"), 0o600))

	// NewIPCBridge should remove the stale file and succeed.
	ms := &mockSender{}
	b, err := NewIPCBridge(ms)
	require.NoError(t, err, "NewIPCBridge must succeed when stale socket exists")
	defer b.Shutdown()

	assert.NotEmpty(t, b.SocketPath())
}
