// Package main implements the MCP side of the IPC spike POC.
//
// Flow:
//  1. Read socket path from GOFORTRESS_SOCKET env var
//  2. Connect to the Unix domain socket with retry logic (5 attempts, exponential backoff)
//  3. Send 5 modal_request messages with sequential IDs
//  4. Wait for each modal_response before sending the next
//  5. Measure per-request roundtrip latency
//  6. Print per-request and average latency summary
//  7. Close the connection cleanly
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// IPCRequest is a modal prompt sent from the MCP side to the TUI.
type IPCRequest struct {
	Type    string   `json:"type"`    // "modal_request"
	ID      string   `json:"id"`      // "req-1"
	Message string   `json:"message"` // "Allow Write to /path?"
	Options []string `json:"options"` // ["Allow", "Allow Always", "Deny"]
}

// IPCResponse is the TUI's answer received from the socket.
type IPCResponse struct {
	Type  string `json:"type"`  // "modal_response"
	ID    string `json:"id"`    // "req-1"
	Value string `json:"value"` // "Allow"
}

// sampleRequests is the set of modal prompts the POC sends.
var sampleRequests = []IPCRequest{
	{
		Type:    "modal_request",
		ID:      "req-1",
		Message: "Allow Write to /home/user/.config/gogent.yaml?",
		Options: []string{"Allow", "Allow Always", "Deny"},
	},
	{
		Type:    "modal_request",
		ID:      "req-2",
		Message: "Allow Bash: rm -rf /tmp/gogent-cache?",
		Options: []string{"Allow", "Deny"},
	},
	{
		Type:    "modal_request",
		ID:      "req-3",
		Message: "Allow Read to /etc/passwd?",
		Options: []string{"Allow", "Allow Always", "Deny"},
	},
	{
		Type:    "modal_request",
		ID:      "req-4",
		Message: "Allow network request to api.anthropic.com?",
		Options: []string{"Allow", "Allow Always", "Deny"},
	},
	{
		Type:    "modal_request",
		ID:      "req-5",
		Message: "Spawn subagent go-pro for implementation?",
		Options: []string{"Allow", "Deny"},
	},
}

// connectWithRetry dials the Unix domain socket with exponential backoff.
// Attempts: 5, base delay: 100ms, doubles each attempt.
func connectWithRetry(sockPath string) (net.Conn, error) {
	const maxAttempts = 5
	delay := 100 * time.Millisecond

	for attempt := range maxAttempts {
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			fmt.Fprintf(os.Stderr, "[mcp] Connected on attempt %d\n", attempt+1)
			return conn, nil
		}
		if attempt < maxAttempts-1 {
			fmt.Fprintf(os.Stderr, "[mcp] Attempt %d failed (%v), retrying in %v\n", attempt+1, err, delay)
			time.Sleep(delay)
			delay *= 2
		} else {
			return nil, fmt.Errorf("connect after %d attempts: %w", maxAttempts, err)
		}
	}
	// Unreachable, but satisfies the compiler.
	return nil, fmt.Errorf("connect: exhausted %d attempts", maxAttempts)
}

func main() {
	sockPath := os.Getenv("GOFORTRESS_SOCKET")
	if sockPath == "" {
		fmt.Fprintln(os.Stderr, "[mcp] GOFORTRESS_SOCKET is not set")
		fmt.Fprintln(os.Stderr, "      Start the TUI first, then export the socket path:")
		fmt.Fprintln(os.Stderr, "      export GOFORTRESS_SOCKET=/run/user/1000/gofortress-poc-12345.sock")
		os.Exit(1)
	}

	conn, err := connectWithRetry(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[mcp] Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	var latencies []time.Duration

	for i, req := range sampleRequests {
		start := time.Now()

		// Send the request.
		if err := enc.Encode(req); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] Send error on req %d: %v\n", i+1, err)
			break
		}

		// Wait for the matching response.
		var resp IPCResponse
		if err := dec.Decode(&resp); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] Receive error on req %d: %v\n", i+1, err)
			break
		}

		rtt := time.Since(start)
		latencies = append(latencies, rtt)

		fmt.Printf("[mcp] req %s → %q  (%v)\n", req.ID, resp.Value, rtt)
	}

	// Print summary.
	if len(latencies) == 0 {
		fmt.Println("[mcp] No requests completed.")
		return
	}

	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	avg := total / time.Duration(len(latencies))

	fmt.Printf("\n[mcp] Summary: %d requests completed\n", len(latencies))
	for i, l := range latencies {
		fmt.Printf("        req-%d: %v\n", i+1, l)
	}
	fmt.Printf("        avg:   %v\n", avg)
}
