package modals

// BenchmarkModalRoundTrip measures the latency of the full modal lifecycle
// inside a single ModalQueue: Push → Activate → Resolve.
//
// Target threshold (TUI-040): < 100 ms.
// In practice the operation is pure in-memory channel + struct work; expect
// sub-microsecond timing with near-zero allocations.
//
// Run:
//
//	go test -bench=BenchmarkModal -benchmem -count=5 ./internal/tui/components/modals/

import (
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// BenchmarkModalRoundTrip benchmarks Push → Activate → Resolve for a single
// Confirm modal.  No channel blocking occurs: the modal is resolved
// synchronously by calling Resolve directly after Activate.
func BenchmarkModalRoundTrip(b *testing.B) {
	b.ReportAllocs()

	km := config.DefaultKeyMap()

	// Pre-build a reusable ModalRequest to avoid allocation inside the loop.
	req := ModalRequest{
		ID:      "bench-request-1",
		Type:    Confirm,
		Message: "Should the benchmark proceed?",
		Header:  "Confirm",
	}

	b.ResetTimer()
	for b.Loop() {
		q := NewModalQueue(km)

		// Push enqueues the request.
		q.Push(req)

		// Activate pops from the queue and creates a ModalModel.
		activated := q.Activate()
		if !activated {
			b.Fatal("Activate returned false on non-empty queue")
		}

		// Resolve closes the active modal and attempts to activate the next
		// queued item (none here, so IsActive becomes false again).
		q.Resolve(ModalResponse{
			Type:  Confirm,
			Value: "Yes",
		})
	}
}

// BenchmarkModalRoundTripWithChannel benchmarks the Push → Activate → Resolve
// cycle when a ResponseCh is attached.  This exercises the non-blocking send
// path inside Resolve, which is the production code path used by the bridge.
func BenchmarkModalRoundTripWithChannel(b *testing.B) {
	b.ReportAllocs()

	km := config.DefaultKeyMap()

	b.ResetTimer()
	for b.Loop() {
		ch := make(chan ModalResponse, 1)
		req := ModalRequest{
			ID:         "bench-request-ch",
			Type:       Confirm,
			Message:    "Proceed?",
			ResponseCh: ch,
		}

		q := NewModalQueue(km)
		q.Push(req)
		q.Activate()

		resp := ModalResponse{Type: Confirm, Value: "Yes"}
		q.Resolve(resp)

		// Drain the channel to confirm the send happened.
		select {
		case <-ch:
		default:
			b.Fatal("ResponseCh was not written by Resolve")
		}
	}
}

// BenchmarkModalQueueBurst benchmarks enqueueing N modals, activating and
// resolving them sequentially (simulating a burst of permission requests).
func BenchmarkModalQueueBurst(b *testing.B) {
	b.ReportAllocs()

	km := config.DefaultKeyMap()
	const burstSize = 10

	reqs := make([]ModalRequest, burstSize)
	for i := range burstSize {
		reqs[i] = ModalRequest{
			ID:      benchModalID(i),
			Type:    Confirm,
			Message: "Permission required for tool call",
			Header:  "Tool Permission",
		}
	}

	b.ResetTimer()
	for b.Loop() {
		q := NewModalQueue(km)

		// Enqueue all N requests.
		for _, r := range reqs {
			q.Push(r)
		}

		// Activate and resolve each one sequentially.
		for {
			q.Activate()
			if !q.IsActive() {
				break
			}
			q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})
		}
	}
}

// benchModalID returns a deterministic string ID for bench index i.
func benchModalID(i int) string {
	ids := [...]string{
		"req-00", "req-01", "req-02", "req-03", "req-04",
		"req-05", "req-06", "req-07", "req-08", "req-09",
	}
	if i < len(ids) {
		return ids[i]
	}
	return "req-xx"
}
