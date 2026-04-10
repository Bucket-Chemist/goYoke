package main

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// permTimeout tests
// =============================================================================

func TestPermTimeout_Default(t *testing.T) {
	t.Setenv("GOGENT_PERM_TIMEOUT", "")
	assert.Equal(t, defaultPermTimeout, permTimeout())
}

func TestPermTimeout_Custom(t *testing.T) {
	t.Setenv("GOGENT_PERM_TIMEOUT", "5000")
	assert.Equal(t, 5*time.Second, permTimeout())
}

func TestPermTimeout_Invalid(t *testing.T) {
	t.Setenv("GOGENT_PERM_TIMEOUT", "abc")
	assert.Equal(t, defaultPermTimeout, permTimeout())
}

func TestPermTimeout_Zero(t *testing.T) {
	// Zero is treated as invalid — falls back to default.
	t.Setenv("GOGENT_PERM_TIMEOUT", "0")
	assert.Equal(t, defaultPermTimeout, permTimeout())
}

func TestPermTimeout_Negative(t *testing.T) {
	// Negative values are treated as invalid — falls back to default.
	t.Setenv("GOGENT_PERM_TIMEOUT", "-100")
	assert.Equal(t, defaultPermTimeout, permTimeout())
}

// =============================================================================
// RequestPermission error-path tests
// =============================================================================

func TestRequestPermission_NoSocket(t *testing.T) {
	t.Setenv("GOFORTRESS_SOCKET", "")

	_, err := RequestPermission("Bash", []byte(`{}`), "session-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No TUI bridge available")
}

func TestRequestPermission_BadSocketPath(t *testing.T) {
	// Point at a path that does not exist — dial will fail.
	t.Setenv("GOFORTRESS_SOCKET", "/nonexistent/path/to/bridge.sock")

	_, err := RequestPermission("Bash", []byte(`{}`), "session-2")

	require.Error(t, err)
	// Should be a dial error, not a "no bridge" error.
	assert.Contains(t, err.Error(), "UDS connection failed")
}

// =============================================================================
// RequestPermission happy-path tests (real UDS listener)
// =============================================================================

// startMockBridge creates a temporary UDS listener and returns the socket path
// and a channel that will receive the first accepted connection. The listener
// is closed via t.Cleanup.
func startMockBridge(t *testing.T) (socketPath string, acceptCh <-chan net.Conn) {
	t.Helper()

	dir := t.TempDir()
	socketPath = filepath.Join(dir, "test.sock")

	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	ch := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		ch <- conn
	}()

	return socketPath, ch
}

// sendPermResponse reads exactly one newline-delimited IPCRequest from conn,
// verifies its type, and writes back an IPCResponse whose payload is a
// PermGateResponsePayload with the given decision. The request ID is echoed so
// the caller's correlation check passes.
func sendPermResponse(t *testing.T, conn net.Conn, decision string) {
	t.Helper()
	defer conn.Close()

	// Read the request line.
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	require.NoError(t, err, "reading request from conn")

	var req mcp.IPCRequest
	require.NoError(t, json.Unmarshal(trimNewline(buf[:n]), &req), "unmarshal request")
	assert.Equal(t, mcp.TypePermGateRequest, req.Type)

	// Build response payload.
	respPayload, err := json.Marshal(mcp.PermGateResponsePayload{Decision: decision})
	require.NoError(t, err)

	resp := mcp.IPCResponse{
		Type:    mcp.TypePermGateResponse,
		ID:      req.ID,
		Payload: json.RawMessage(respPayload),
	}

	respBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	_, err = fmt.Fprintf(conn, "%s\n", respBytes)
	require.NoError(t, err)
}

// trimNewline removes a trailing newline from b (handles both \n and \r\n).
func trimNewline(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func TestRequestPermission_AllowResponse(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	t.Setenv("GOGENT_PERM_TIMEOUT", "2000")

	// Serve the mock response in a goroutine.
	go func() {
		conn := <-acceptCh
		sendPermResponse(t, conn, "allow")
	}()

	decision, err := RequestPermission("Bash", []byte(`{"command":"ls"}`), "session-allow")

	require.NoError(t, err)
	assert.Equal(t, "allow", decision)
}

func TestRequestPermission_DenyResponse(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	t.Setenv("GOGENT_PERM_TIMEOUT", "2000")

	go func() {
		conn := <-acceptCh
		sendPermResponse(t, conn, "deny")
	}()

	decision, err := RequestPermission("Bash", []byte(`{"command":"rm -rf /"}`), "session-deny")

	require.NoError(t, err)
	assert.Equal(t, "deny", decision)
}

func TestRequestPermission_AllowSessionResponse(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	t.Setenv("GOGENT_PERM_TIMEOUT", "2000")

	go func() {
		conn := <-acceptCh
		sendPermResponse(t, conn, "allow_session")
	}()

	decision, err := RequestPermission("Bash", []byte(`{}`), "session-allow-session")

	require.NoError(t, err)
	assert.Equal(t, "allow_session", decision)
}

// =============================================================================
// Timeout test
// =============================================================================

func TestRequestPermission_Timeout(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	// Short timeout so the test runs fast.
	t.Setenv("GOGENT_PERM_TIMEOUT", "200")

	// Accept the connection but never respond — triggers the read deadline.
	go func() {
		conn := <-acceptCh
		// Hold the connection open until the test ends.
		defer conn.Close()
		time.Sleep(5 * time.Second)
	}()

	_, err := RequestPermission("Bash", []byte(`{}`), "session-timeout")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// =============================================================================
// Malformed-response tests
// =============================================================================

func TestRequestPermission_MalformedResponseJSON(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	t.Setenv("GOGENT_PERM_TIMEOUT", "2000")

	go func() {
		conn := <-acceptCh
		defer conn.Close()
		// Read and discard the request.
		buf := make([]byte, 4096)
		conn.Read(buf) //nolint:errcheck
		// Write garbage JSON back.
		fmt.Fprintf(conn, "NOT_VALID_JSON\n") //nolint:errcheck
	}()

	_, err := RequestPermission("Bash", []byte(`{}`), "session-malformed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "UDS connection failed")
}

func TestRequestPermission_ResponseIDMismatch(t *testing.T) {
	socketPath, acceptCh := startMockBridge(t)
	t.Setenv("GOFORTRESS_SOCKET", socketPath)
	t.Setenv("GOGENT_PERM_TIMEOUT", "2000")

	go func() {
		conn := <-acceptCh
		defer conn.Close()

		// Read the real request so the write-side unblocks.
		buf := make([]byte, 4096)
		conn.Read(buf) //nolint:errcheck

		// Respond with a deliberately wrong ID.
		respPayload, _ := json.Marshal(mcp.PermGateResponsePayload{Decision: "allow"})
		resp := mcp.IPCResponse{
			Type:    mcp.TypePermGateResponse,
			ID:      "wrong-id-000",
			Payload: json.RawMessage(respPayload),
		}
		respBytes, _ := json.Marshal(resp)
		fmt.Fprintf(conn, "%s\n", respBytes) //nolint:errcheck
	}()

	_, err := RequestPermission("Bash", []byte(`{}`), "session-id-mismatch")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "response ID mismatch")
}
