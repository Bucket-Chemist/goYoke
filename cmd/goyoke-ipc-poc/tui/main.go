// Package main implements the TUI side of the IPC spike POC.
// Validates Unix domain socket IPC: creates socket, accepts MCP connection,
// receives JSON requests, auto-responds after 10ms delay (simulating modal),
// measures roundtrip latency.
//
// NOTE: This spike intentionally omits Bubble Tea to allow headless testing.
// The Program.Send() injection pattern is well-established in Bubbletea;
// what this spike validates is the UDS transport, JSON protocol, and latency.
// Production code (TUI-015) will use Program.Send() from the socket reader goroutine.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// IPCRequest is a modal prompt sent from the MCP side to the TUI.
type IPCRequest struct {
	Type    string   `json:"type"`
	ID      string   `json:"id"`
	Message string   `json:"message"`
	Options []string `json:"options"`
}

// IPCResponse is the TUI's answer sent back to the MCP side.
type IPCResponse struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Value string `json:"value"`
}

// runtimeDir returns XDG_RUNTIME_DIR, falling back to os.TempDir.
func runtimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	return os.TempDir()
}

// cleanupStaleSockets removes orphaned poc sockets. Mirrors internal/lifecycle/process.go.
func cleanupStaleSockets() {
	matches, _ := filepath.Glob(filepath.Join(runtimeDir(), "goyoke-poc-*.sock"))
	for _, path := range matches {
		base := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "goyoke-poc-"), ".sock")
		pid, err := strconv.Atoi(base)
		if err != nil || pid <= 0 {
			continue
		}
		proc, err := os.FindProcess(pid)
		if err != nil || proc.Signal(syscall.Signal(0)) != nil {
			if os.Remove(path) == nil {
				fmt.Fprintf(os.Stderr, "[tui] Removed stale socket: %s\n", path)
			}
		}
	}
}

func main() {
	modalDelay := flag.Duration("delay", 10*time.Millisecond, "Simulated modal response delay")
	flag.Parse()

	cleanupStaleSockets()

	sockPath := filepath.Join(runtimeDir(), fmt.Sprintf("goyoke-poc-%d.sock", os.Getpid()))
	_ = os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[tui] Listen failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(sockPath)

	fmt.Fprintf(os.Stderr, "[tui] Socket: %s\n", sockPath)
	_ = os.Setenv("GOYOKE_SOCKET", sockPath)

	// Clean up socket on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; ln.Close(); os.Remove(sockPath); os.Exit(0) }()

	// Accept one connection (production would accept multiple).
	conn, err := ln.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[tui] Accept failed: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Fprintf(os.Stderr, "[tui] MCP connected\n")

	// Channel simulates Program.Send() — in production, the reader goroutine
	// calls p.Send(ipcRequestMsg{...}) which delivers to Update() via this same
	// channel-based pattern internally.
	msgCh := make(chan IPCRequest, 8)

	// Reader goroutine: decode JSON requests from socket → channel.
	// In production: replace channel send with p.Send(ipcRequestMsg{...}).
	go func() {
		dec := json.NewDecoder(conn)
		for {
			var req IPCRequest
			if err := dec.Decode(&req); err != nil {
				close(msgCh)
				return
			}
			msgCh <- req
		}
	}()

	// Main loop: receive requests, simulate modal delay, send response.
	// In production: this logic lives in Model.Update().
	enc := json.NewEncoder(conn)
	var totalRTT time.Duration
	count := 0

	for req := range msgCh {
		receivedAt := time.Now()

		// Simulate modal interaction delay.
		time.Sleep(*modalDelay)

		// Auto-respond with first option (production: user picks interactively).
		value := "Allow"
		if len(req.Options) > 0 {
			value = req.Options[0]
		}

		resp := IPCResponse{Type: "modal_response", ID: req.ID, Value: value}
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "[tui] Write error: %v\n", err)
			break
		}

		rtt := time.Since(receivedAt)
		totalRTT += rtt
		count++
		fmt.Fprintf(os.Stderr, "[tui] %s → %s (%v)\n", req.ID, value, rtt)
	}

	if count > 0 {
		avg := totalRTT / time.Duration(count)
		fmt.Fprintf(os.Stderr, "[tui] Summary: %d requests, avg RTT %v (includes %v modal delay)\n",
			count, avg, *modalDelay)
	}
}
