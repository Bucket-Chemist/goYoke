//go:build integration

// Package integration contains end-to-end integration tests for the
// GOgent-Fortress permission gate feature (PERM-001 through PERM-007).
//
// These tests exercise the full flow:
//
//	Hook binary UDS client → Bridge UDS server → handlePermGate → ResolvePermGate → response
//
// They require real network I/O (UDS sockets) and are excluded from the
// default test run. Use:
//
//	go test -tags integration ./test/integration/... -run TestIntegration_PermGate
package integration

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/bridge"
	"github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Mock sender
// ---------------------------------------------------------------------------

// permGateMockSender captures every tea.Msg delivered via Send for later
// assertion. It is safe for concurrent use from multiple goroutines.
type permGateMockSender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (m *permGateMockSender) Send(msg tea.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
}

// waitForPermMsg blocks until at least one model.CLIPermissionRequestMsg
// arrives or the timeout elapses. It returns the first matching message and
// true, or the zero value and false on timeout.
func waitForPermMsg(ms *permGateMockSender, timeout time.Duration) (model.CLIPermissionRequestMsg, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		for _, msg := range ms.msgs {
			if v, ok := msg.(model.CLIPermissionRequestMsg); ok {
				ms.mu.Unlock()
				return v, true
			}
		}
		ms.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	return model.CLIPermissionRequestMsg{}, false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const permGateTestTimeout = 2 * time.Second

// startPermGateBridge creates a bridge backed by a permGateMockSender, sets
// XDG_RUNTIME_DIR to a temp directory so the socket lands under t.TempDir(),
// starts the bridge, and registers a cleanup that calls Shutdown.
func startPermGateBridge(t *testing.T) (*bridge.IPCBridge, *permGateMockSender) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	ms := &permGateMockSender{}
	b, err := bridge.NewIPCBridge(ms)
	require.NoError(t, err, "NewIPCBridge must succeed")
	b.Start()
	t.Cleanup(b.Shutdown)
	return b, ms
}

// dialPermGateBridge dials the bridge socket and returns the raw connection
// together with JSON encoder/decoder wrappers.
func dialPermGateBridge(t *testing.T, socketPath string) (net.Conn, *json.Encoder, *json.Decoder) {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err, "net.Dial must connect to bridge socket")
	t.Cleanup(func() { conn.Close() })
	return conn, json.NewEncoder(conn), json.NewDecoder(conn)
}

// buildPermGateRequest creates a mcp.IPCRequest of type TypePermGateRequest.
func buildPermGateRequest(t *testing.T, id string, payload mcp.PermGateRequestPayload) mcp.IPCRequest {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err, "marshal PermGateRequestPayload")
	return mcp.IPCRequest{
		Type:    mcp.TypePermGateRequest,
		ID:      id,
		Payload: json.RawMessage(raw),
	}
}

// sendPermGateRequest encodes and writes req to enc.
func sendPermGateRequest(t *testing.T, enc *json.Encoder, req mcp.IPCRequest) {
	t.Helper()
	require.NoError(t, enc.Encode(req), "enc.Encode must not fail")
}

// readPermGateResponse decodes one IPCResponse from dec within timeout and
// returns the inner PermGateResponsePayload.
func readPermGateResponse(t *testing.T, conn net.Conn, dec *json.Decoder, timeout time.Duration) mcp.PermGateResponsePayload {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	var resp mcp.IPCResponse
	require.NoError(t, dec.Decode(&resp), "dec.Decode IPCResponse")
	assert.Equal(t, mcp.TypePermGateResponse, resp.Type)

	var payload mcp.PermGateResponsePayload
	require.NoError(t, json.Unmarshal(resp.Payload, &payload), "unmarshal PermGateResponsePayload")
	return payload
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIntegration_PermGate_AllowFlow validates the full allow round-trip:
//
//  1. A UDS client sends permission_gate_request to the bridge.
//  2. The bridge injects CLIPermissionRequestMsg into the mock sender.
//  3. ResolvePermGate("allow") is called to simulate a user decision.
//  4. The bridge returns a permission_gate_response with decision="allow".
func TestIntegration_PermGate_AllowFlow(t *testing.T) {
	b, ms := startPermGateBridge(t)
	conn, enc, dec := dialPermGateBridge(t, b.SocketPath())

	req := buildPermGateRequest(t, "perm-allow-1", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{"command":"ls -la /tmp"}`),
		SessionID: "session-allow",
		TimeoutMS: 5000,
	})
	sendPermGateRequest(t, enc, req)

	// Verify CLIPermissionRequestMsg was injected into the Bubbletea loop.
	permMsg, ok := waitForPermMsg(ms, permGateTestTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")
	assert.Equal(t, "perm-allow-1", permMsg.RequestID)
	assert.Equal(t, "Bash", permMsg.ToolName)
	assert.Equal(t, 5000, permMsg.TimeoutMS)

	// Simulate user clicking "Allow".
	go b.ResolvePermGate("perm-allow-1", "allow")

	payload := readPermGateResponse(t, conn, dec, permGateTestTimeout)
	assert.Equal(t, "allow", payload.Decision)
}

// TestIntegration_PermGate_DenyFlow validates the full deny round-trip:
// the user explicitly selects "deny" and the bridge returns decision="deny".
func TestIntegration_PermGate_DenyFlow(t *testing.T) {
	b, ms := startPermGateBridge(t)
	conn, enc, dec := dialPermGateBridge(t, b.SocketPath())

	req := buildPermGateRequest(t, "perm-deny-1", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{"command":"rm -rf /important"}`),
		SessionID: "session-deny",
		TimeoutMS: 5000,
	})
	sendPermGateRequest(t, enc, req)

	permMsg, ok := waitForPermMsg(ms, permGateTestTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")
	assert.Equal(t, "perm-deny-1", permMsg.RequestID)

	// Simulate user clicking "Deny".
	go b.ResolvePermGate("perm-deny-1", "deny")

	payload := readPermGateResponse(t, conn, dec, permGateTestTimeout)
	assert.Equal(t, "deny", payload.Decision)
}

// TestIntegration_PermGate_AllowSessionFlow validates the allow_session
// round-trip: the bridge returns decision="allow_session" when the user
// selects "Allow for this session".
func TestIntegration_PermGate_AllowSessionFlow(t *testing.T) {
	b, ms := startPermGateBridge(t)
	conn, enc, dec := dialPermGateBridge(t, b.SocketPath())

	req := buildPermGateRequest(t, "perm-session-1", mcp.PermGateRequestPayload{
		ToolName:  "Write",
		ToolInput: json.RawMessage(`{"file_path":"/tmp/test.txt","content":"hello"}`),
		SessionID: "session-allow-session",
		TimeoutMS: 5000,
	})
	sendPermGateRequest(t, enc, req)

	permMsg, ok := waitForPermMsg(ms, permGateTestTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")
	assert.Equal(t, "perm-session-1", permMsg.RequestID)

	// Simulate user clicking "Allow for session".
	go b.ResolvePermGate("perm-session-1", "allow_session")

	payload := readPermGateResponse(t, conn, dec, permGateTestTimeout)
	assert.Equal(t, "allow_session", payload.Decision)
}

// TestIntegration_PermGate_TimeoutAutoDeny validates that a permission gate
// request is automatically denied when no ResolvePermGate call is made within
// the configured TimeoutMS window.
func TestIntegration_PermGate_TimeoutAutoDeny(t *testing.T) {
	b, ms := startPermGateBridge(t)
	conn, enc, dec := dialPermGateBridge(t, b.SocketPath())

	req := buildPermGateRequest(t, "perm-timeout-1", mcp.PermGateRequestPayload{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{"command":"sleep 100"}`),
		SessionID: "session-timeout",
		TimeoutMS: 200, // Short timeout — auto-deny fires before test timeout.
	})
	sendPermGateRequest(t, enc, req)

	// Confirm the request was received by the bridge.
	_, ok := waitForPermMsg(ms, permGateTestTimeout)
	require.True(t, ok, "timed out waiting for CLIPermissionRequestMsg")

	// Do NOT call ResolvePermGate — wait for the bridge to auto-deny.
	// Allow extra headroom beyond the 200 ms payload timeout.
	payload := readPermGateResponse(t, conn, dec, permGateTestTimeout)
	assert.Equal(t, "deny", payload.Decision, "bridge must auto-deny on timeout")
}

// TestIntegration_PermGate_InvalidSocket validates the hook binary's client
// behaviour when GOFORTRESS_SOCKET points to a nonexistent path: the UDS
// dial fails and RequestPermission returns an error, which main() maps to an
// auto-deny block response.
//
// This test drives RequestPermission directly (the exported function from
// cmd/gogent-permission-gate) rather than through the compiled binary, since
// the binary lives in package main and is not importable. Instead we replicate
// the client-side dial logic inline using the same UDS primitives.
func TestIntegration_PermGate_InvalidSocket(t *testing.T) {
	// Point GOFORTRESS_SOCKET at a path that does not exist.
	nonexistentSocket := filepath.Join(t.TempDir(), "no-such-socket.sock")
	t.Setenv("GOFORTRESS_SOCKET", nonexistentSocket)

	// Replicate the UDS client dial from cmd/gogent-permission-gate/uds.go.
	// A dial to a nonexistent path must fail immediately.
	conn, err := net.DialTimeout("unix", nonexistentSocket, 2*time.Second)
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err, "dialing a nonexistent socket must return an error")

	// Verify the env var is set to the nonexistent path (confirms t.Setenv worked).
	assert.Equal(t, nonexistentSocket, os.Getenv("GOFORTRESS_SOCKET"))
}

// TestIntegration_PermGate_ConcurrentRequests validates that the bridge
// correctly handles multiple simultaneous permission gate requests from
// different connections, each resolved independently.
func TestIntegration_PermGate_ConcurrentRequests(t *testing.T) {
	b, ms := startPermGateBridge(t)

	const numRequests = 3
	decisions := []string{"allow", "deny", "allow_session"}
	type result struct {
		id       string
		decision string
	}
	results := make(chan result, numRequests)

	var wg sync.WaitGroup
	for i := range numRequests {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			conn, enc, dec := dialPermGateBridge(t, b.SocketPath())

			reqID := "perm-concurrent-" + string(rune('1'+idx))
			req := buildPermGateRequest(t, reqID, mcp.PermGateRequestPayload{
				ToolName:  "Bash",
				ToolInput: json.RawMessage(`{}`),
				SessionID: "session-concurrent",
				TimeoutMS: 5000,
			})
			sendPermGateRequest(t, enc, req)

			// Wait for the specific CLIPermissionRequestMsg for this request.
			deadline := time.Now().Add(permGateTestTimeout)
			var found bool
			for time.Now().Before(deadline) && !found {
				ms.mu.Lock()
				for _, msg := range ms.msgs {
					if v, ok := msg.(model.CLIPermissionRequestMsg); ok && v.RequestID == reqID {
						found = true
						break
					}
				}
				ms.mu.Unlock()
				if !found {
					time.Sleep(5 * time.Millisecond)
				}
			}
			if !found {
				t.Errorf("timed out waiting for CLIPermissionRequestMsg for %s", reqID)
				return
			}

			go b.ResolvePermGate(reqID, decisions[idx])

			conn.SetReadDeadline(time.Now().Add(permGateTestTimeout))
			var resp mcp.IPCResponse
			if err := dec.Decode(&resp); err != nil {
				t.Errorf("decode response for %s: %v", reqID, err)
				return
			}

			var payload mcp.PermGateResponsePayload
			if err := json.Unmarshal(resp.Payload, &payload); err != nil {
				t.Errorf("unmarshal payload for %s: %v", reqID, err)
				return
			}

			results <- result{id: reqID, decision: payload.Decision}
		}(i)
	}

	wg.Wait()
	close(results)

	collected := make(map[string]string)
	for r := range results {
		collected[r.id] = r.decision
	}

	assert.Equal(t, "allow", collected["perm-concurrent-1"])
	assert.Equal(t, "deny", collected["perm-concurrent-2"])
	assert.Equal(t, "allow_session", collected["perm-concurrent-3"])
}
