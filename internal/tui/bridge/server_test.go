package bridge

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/mcp"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
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
	bridgeMsg, ok := waitFor[BridgeModalRequestMsg](ms, testTimeout)
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
	assert.Equal(t, "info", msg.Level)
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
	_, ok := waitFor[BridgeModalRequestMsg](ms, testTimeout)
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
	assert.Contains(t, path, "gofortress-")
	assert.Contains(t, path, ".sock")
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
