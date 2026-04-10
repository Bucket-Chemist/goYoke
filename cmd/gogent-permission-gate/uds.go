package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/mcp"
	"github.com/google/uuid"
)

// defaultPermTimeout is used when GOGENT_PERM_TIMEOUT is not set.
const defaultPermTimeout = 30 * time.Second

// permTimeout reads the timeout from GOGENT_PERM_TIMEOUT (milliseconds).
// Falls back to defaultPermTimeout on missing or invalid values.
func permTimeout() time.Duration {
	raw := os.Getenv("GOGENT_PERM_TIMEOUT")
	if raw == "" {
		return defaultPermTimeout
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return defaultPermTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

// RequestPermission contacts the TUI bridge over UDS and returns the user's
// decision for the tool invocation described by toolName, toolInputJSON, and
// sessionID.
//
// Possible return values: "allow", "deny", "allow_session".
//
// Errors are returned as descriptive strings prefixed with a short tag so the
// caller can map them to block responses with actionable messages.
func RequestPermission(toolName string, toolInputJSON []byte, sessionID string) (string, error) {
	socketPath := os.Getenv("GOFORTRESS_SOCKET")
	if socketPath == "" {
		return "", fmt.Errorf("No TUI bridge available")
	}

	timeout := permTimeout()

	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		// Distinguish refused connections from other dial errors.
		if isRefused(err) {
			return "", fmt.Errorf("Bridge not reachable")
		}
		return "", fmt.Errorf("UDS connection failed: %v", err)
	}
	defer conn.Close()

	// Build and send the permission gate request.
	reqID := uuid.New().String()

	payload := mcp.PermGateRequestPayload{
		ToolName:  toolName,
		ToolInput: json.RawMessage(toolInputJSON),
		SessionID: sessionID,
		TimeoutMS: int(timeout.Milliseconds()),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("UDS connection failed: marshal payload: %v", err)
	}

	req := mcp.IPCRequest{
		Type:    mcp.TypePermGateRequest,
		ID:      reqID,
		Payload: json.RawMessage(payloadBytes),
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("UDS connection failed: marshal request: %v", err)
	}

	// Set write deadline.
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", fmt.Errorf("UDS connection failed: set write deadline: %v", err)
	}

	// Newline-delimited JSON framing — write request followed by \n.
	if _, err := fmt.Fprintf(conn, "%s\n", reqBytes); err != nil {
		return "", fmt.Errorf("UDS connection failed: write: %v", err)
	}

	// Set read deadline based on configured timeout.
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", fmt.Errorf("UDS connection failed: set read deadline: %v", err)
	}

	// Read newline-delimited response.
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			if isTimeout(err) {
				return "", fmt.Errorf("Permission request timed out")
			}
			return "", fmt.Errorf("UDS connection failed: read: %v", err)
		}
		return "", fmt.Errorf("Permission request timed out")
	}

	var ipcResp mcp.IPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &ipcResp); err != nil {
		return "", fmt.Errorf("UDS connection failed: unmarshal response: %v", err)
	}

	// Validate correlation ID.
	if ipcResp.ID != reqID {
		return "", fmt.Errorf("UDS connection failed: response ID mismatch")
	}

	var respPayload mcp.PermGateResponsePayload
	if err := json.Unmarshal(ipcResp.Payload, &respPayload); err != nil {
		return "", fmt.Errorf("UDS connection failed: unmarshal response payload: %v", err)
	}

	return respPayload.Decision, nil
}

// isRefused returns true when err is a connection-refused error.
func isRefused(err error) bool {
	if err == nil {
		return false
	}
	// net.OpError wraps a syscall.Errno on Unix.
	if netErr, ok := err.(*net.OpError); ok {
		return netErr.Op == "dial" && isErrnoRefused(netErr.Err)
	}
	return false
}

// isErrnoRefused inspects the inner error from a net.OpError.
func isErrnoRefused(err error) bool {
	// Use string matching as a portable fallback — avoids syscall import on
	// all target platforms.
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "connection refused" ||
		msg == "connect: connection refused"
}

// isTimeout returns true when err is a deadline-exceeded (timeout) error.
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
