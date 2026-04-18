package mcp_test

// BenchmarkUDSRoundTrip measures the latency of one IPC round-trip over a
// real Unix domain socket using the full production path:
//
//	UDSClient.SendRequest → net.Conn (unix) → IPCBridge handler → net.Conn (unix)
//
// This file is in the external test package (mcp_test) for the same reason as
// server_integration_test.go: the mcp and bridge packages mutually import each
// other, which an external test package breaks.
//
// Target threshold (TUI-040): < 5 ms per round-trip.
// Typical loopback UDS latency on Linux is ~50–200 µs, well under the target.
//
// Run:
//
//	go test -bench=BenchmarkUDS -benchmem -count=5 ./internal/tui/mcp/
//
// Skip in short mode:
//
//	go test -bench=BenchmarkUDS -short ./internal/tui/mcp/   # skipped

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/bridge"
	tuimcp "github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// benchAutoResolveSender
//
// Implements the messageSender interface expected by bridge.NewIPCBridge.
// Immediately resolves every modal_request it receives so that
// UDSClient.SendRequest unblocks with minimum latency.
// ---------------------------------------------------------------------------

type benchAutoResolveSender struct {
	mu     sync.Mutex
	bridge *bridge.IPCBridge
}

func (s *benchAutoResolveSender) setBridge(br *bridge.IPCBridge) {
	s.mu.Lock()
	s.bridge = br
	s.mu.Unlock()
}

func (s *benchAutoResolveSender) Send(msg tea.Msg) {
	s.mu.Lock()
	br := s.bridge
	s.mu.Unlock()

	if br == nil {
		return
	}
	if bm, ok := msg.(model.BridgeModalRequestMsg); ok {
		br.ResolveModalSimple(bm.RequestID, "Yes")
	}
}

// ---------------------------------------------------------------------------
// newBenchBridgeHarness
//
// Starts a real IPCBridge + connects a UDSClient.  Uses b.Setenv so that
// GOFORTRESS_SOCKET and XDG_RUNTIME_DIR are automatically restored after the
// benchmark.  The cleanup function shuts down the bridge.
// ---------------------------------------------------------------------------

func newBenchBridgeHarness(b *testing.B) (*bridge.IPCBridge, *tuimcp.UDSClient, func()) {
	b.Helper()

	tmpDir := b.TempDir()
	b.Setenv("XDG_RUNTIME_DIR", tmpDir)

	as := &benchAutoResolveSender{}
	br, err := bridge.NewIPCBridge(as)
	if err != nil {
		b.Fatalf("NewIPCBridge: %v", err)
	}
	as.setBridge(br)
	br.Start()

	b.Setenv("GOFORTRESS_SOCKET", br.SocketPath())
	uds := tuimcp.NewUDSClient()

	// Eagerly connect so the benchmark loop does not pay dial overhead.
	if err := uds.Connect(); err != nil {
		br.Shutdown()
		b.Fatalf("UDSClient.Connect: %v", err)
	}

	return br, uds, func() { br.Shutdown() }
}

// ---------------------------------------------------------------------------
// BenchmarkUDSRoundTrip
// ---------------------------------------------------------------------------

// BenchmarkUDSRoundTrip measures one modal_request → modal_response round-trip
// over a real Unix domain socket.
//
// Per-iteration overhead includes:
//  1. json.Encoder.Encode (client side)
//  2. Kernel UDS send + goroutine context switch (client → bridge)
//  3. json.Decoder.Decode + dispatch inside the bridge
//  4. benchAutoResolveSender.Send (synchronous ResolveModalSimple)
//  5. json.Encoder.Encode for the response (bridge side)
//  6. Kernel UDS send + context switch (bridge → client)
//  7. json.Decoder.Decode (client side)
func BenchmarkUDSRoundTrip(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping UDS benchmark in short mode")
	}

	b.ReportAllocs()

	_, uds, cleanup := newBenchBridgeHarness(b)
	defer cleanup()

	// Pre-marshal the payload to eliminate allocation from the hot loop.
	payload := tuimcp.ModalRequestPayload{
		Message: "Benchmark: should we proceed?",
		Options: []string{"Yes", "No"},
		Default: "Yes",
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		b.Fatalf("marshal payload: %v", err)
	}

	b.ResetTimer()
	for i := range b.N {
		req := tuimcp.IPCRequest{
			Type:    tuimcp.TypeModalRequest,
			ID:      fmt.Sprintf("bench-%d", i),
			Payload: rawPayload,
		}

		resp, sendErr := uds.SendRequest(req)
		if sendErr != nil {
			b.Fatalf("SendRequest iteration %d: %v", i, sendErr)
		}
		if resp == nil {
			b.Fatalf("nil response at iteration %d", i)
		}
	}
}

// BenchmarkUDSRoundTripParallel measures concurrent round-trips.
//
// UDSClient serialises requests internally via a mutex, so parallel goroutines
// queue behind each other.  This benchmark captures scheduler overhead and
// mutex contention under parallel load rather than true concurrency.
func BenchmarkUDSRoundTripParallel(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping UDS benchmark in short mode")
	}

	b.ReportAllocs()

	_, uds, cleanup := newBenchBridgeHarness(b)
	defer cleanup()

	payload := tuimcp.ModalRequestPayload{
		Message: "Parallel bench",
		Options: []string{"Yes"},
	}
	rawPayload, _ := json.Marshal(payload)

	var (
		seqMu   sync.Mutex
		counter int
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			seqMu.Lock()
			id := fmt.Sprintf("par-%d", counter)
			counter++
			seqMu.Unlock()

			req := tuimcp.IPCRequest{
				Type:    tuimcp.TypeModalRequest,
				ID:      id,
				Payload: rawPayload,
			}
			resp, err := uds.SendRequest(req)
			if err != nil {
				b.Errorf("SendRequest %s: %v", id, err)
				return
			}
			_ = resp
		}
	})
}
