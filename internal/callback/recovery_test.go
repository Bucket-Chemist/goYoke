package callback

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestSendPromptWithRetry_Success verifies successful retry after transient failure
func TestSendPromptWithRetry_Success(t *testing.T) {
	s := NewServer(os.Getpid() + 30000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	c := NewClientWithPath(s.SocketPath())

	// Handler responds to prompt
	go func() {
		req := <-s.PromptChan
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: "success",
		})
	}()

	resp, err := c.SendPromptWithRetry(ctx, PromptRequest{
		ID:      "retry-success",
		Type:    "ask",
		Message: "Test",
	})

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if resp.Value != "success" {
		t.Errorf("Expected 'success', got %q", resp.Value)
	}
}

// TestSendPromptWithRetry_EventualSuccess verifies retry with exponential backoff
func TestSendPromptWithRetry_EventualSuccess(t *testing.T) {
	// Use unreachable server initially, then start it
	socketPath := "/tmp/retry-test-socket.sock"
	c := NewClientWithPath(socketPath)

	// Start retry in background
	resultChan := make(chan error, 1)
	go func() {
		ctx := context.Background()
		_, err := c.SendPromptWithRetry(ctx, PromptRequest{
			ID:      "retry-eventual",
			Type:    "ask",
			Message: "Test",
		})
		resultChan <- err
	}()

	// Wait for first retry attempt to fail
	time.Sleep(150 * time.Millisecond)

	// Now start the server
	s := NewServer(os.Getpid() + 30001)
	ctx := context.Background()

	// Override socket path
	s.socketPath = socketPath

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	// Handler responds to prompt
	go func() {
		select {
		case req := <-s.PromptChan:
			s.SendResponse(PromptResponse{
				ID:    req.ID,
				Value: "eventual success",
			})
		case <-time.After(2 * time.Second):
			// Timeout - no prompt received
		}
	}()

	// Wait for result
	select {
	case err := <-resultChan:
		if err != nil {
			t.Errorf("Expected eventual success, got error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Test timeout - retry should have succeeded")
	}
}

// TestSendPromptWithRetry_MaxRetriesExceeded verifies error after max retries
func TestSendPromptWithRetry_MaxRetriesExceeded(t *testing.T) {
	// Point to non-existent socket
	c := NewClientWithPath("/tmp/nonexistent-retry-socket.sock")
	ctx := context.Background()

	start := time.Now()
	_, err := c.SendPromptWithRetry(ctx, PromptRequest{
		ID:      "retry-max",
		Type:    "ask",
		Message: "This will fail",
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max retries")
	}

	// Verify exponential backoff: 100ms + 200ms + 400ms = 700ms minimum
	if elapsed < 700*time.Millisecond {
		t.Errorf("Expected at least 700ms for retries, got %v", elapsed)
	}

	// Should not exceed 1 second (some margin for execution time)
	if elapsed > 1200*time.Millisecond {
		t.Errorf("Retries took too long: %v", elapsed)
	}
}

// TestSendPromptWithRetry_ContextCancellation verifies no retry on cancellation
func TestSendPromptWithRetry_ContextCancellation(t *testing.T) {
	// Point to non-existent socket
	c := NewClientWithPath("/tmp/cancel-retry-socket.sock")

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first failure
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.SendPromptWithRetry(ctx, PromptRequest{
		ID:      "retry-cancel",
		Type:    "ask",
		Message: "This will be cancelled",
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error from cancellation")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Should return quickly after cancellation, not wait for all retries
	if elapsed > 500*time.Millisecond {
		t.Errorf("Cancellation took too long: %v (should be <500ms)", elapsed)
	}
}

// TestSendPromptWithRetry_ContextTimeout verifies timeout handling
func TestSendPromptWithRetry_ContextTimeout(t *testing.T) {
	c := NewClientWithPath("/tmp/timeout-retry-socket.sock")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.SendPromptWithRetry(ctx, PromptRequest{
		ID:      "retry-timeout",
		Type:    "ask",
		Message: "This will timeout",
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Should timeout around 200ms, not wait for all retries
	if elapsed > 400*time.Millisecond {
		t.Errorf("Timeout took too long: %v (should be ~200ms)", elapsed)
	}
}

// TestHealthMonitor_Healthy verifies health monitoring of healthy server
func TestHealthMonitor_Healthy(t *testing.T) {
	s := NewServer(os.Getpid() + 30002)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	c := NewClientWithPath(s.SocketPath())

	unhealthyCalled := atomic.Int32{}
	monitor := NewHealthMonitor(c, 100*time.Millisecond, func() {
		unhealthyCalled.Add(1)
	})

	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	monitor.Start(monitorCtx)

	// Wait for several health checks
	time.Sleep(350 * time.Millisecond)

	if !monitor.IsHealthy() {
		t.Error("Monitor should report healthy")
	}

	if unhealthyCalled.Load() > 0 {
		t.Errorf("onUnhealthy should not be called for healthy server, called %d times", unhealthyCalled.Load())
	}
}

// TestHealthMonitor_BecomeUnhealthy verifies detection of server failure
func TestHealthMonitor_BecomeUnhealthy(t *testing.T) {
	s := NewServer(os.Getpid() + 30003)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		s.Cleanup()
		t.Fatalf("Socket not ready: %v", err)
	}

	c := NewClientWithPath(s.SocketPath())

	unhealthyCalled := atomic.Int32{}
	monitor := NewHealthMonitor(c, 100*time.Millisecond, func() {
		unhealthyCalled.Add(1)
	})

	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	monitor.Start(monitorCtx)

	// Wait for initial healthy checks
	time.Sleep(150 * time.Millisecond)

	if !monitor.IsHealthy() {
		t.Error("Monitor should initially be healthy")
	}

	// Shutdown server to make it unhealthy
	s.Shutdown(ctx)
	s.Cleanup()

	// Wait for health check to detect failure
	time.Sleep(250 * time.Millisecond)

	if monitor.IsHealthy() {
		t.Error("Monitor should detect unhealthy state")
	}

	if unhealthyCalled.Load() != 1 {
		t.Errorf("onUnhealthy should be called exactly once, called %d times", unhealthyCalled.Load())
	}
}

// TestHealthMonitor_TransitionOnly verifies onUnhealthy called only on transition
func TestHealthMonitor_TransitionOnly(t *testing.T) {
	// Start with non-existent server
	c := NewClientWithPath("/tmp/transition-test-socket.sock")

	unhealthyCalled := atomic.Int32{}
	monitor := NewHealthMonitor(c, 50*time.Millisecond, func() {
		unhealthyCalled.Add(1)
	})

	// Monitor starts with healthy=true (optimistic)
	// First check will fail and trigger transition

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx)

	// Wait for multiple health check cycles
	time.Sleep(250 * time.Millisecond)

	// Should transition to unhealthy exactly once
	if unhealthyCalled.Load() != 1 {
		t.Errorf("Expected onUnhealthy called once during transition, got %d calls", unhealthyCalled.Load())
	}

	if monitor.IsHealthy() {
		t.Error("Monitor should be unhealthy")
	}
}

// TestHealthMonitor_ContextCancellation verifies graceful shutdown
func TestHealthMonitor_ContextCancellation(t *testing.T) {
	s := NewServer(os.Getpid() + 30004)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	c := NewClientWithPath(s.SocketPath())

	monitor := NewHealthMonitor(c, 50*time.Millisecond, nil)

	monitorCtx, cancel := context.WithCancel(ctx)
	monitor.Start(monitorCtx)

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel and verify it stops gracefully
	cancel()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// If we get here without deadlock, the monitor stopped correctly
}

// TestHealthMonitor_RecoverFromUnhealthy verifies recovery detection
func TestHealthMonitor_RecoverFromUnhealthy(t *testing.T) {
	socketPath := "/tmp/recover-test-socket.sock"

	// Start with no server (will be unhealthy)
	c := NewClientWithPath(socketPath)

	unhealthyCalled := atomic.Int32{}
	monitor := NewHealthMonitor(c, 100*time.Millisecond, func() {
		unhealthyCalled.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx)

	// Wait for transition to unhealthy
	time.Sleep(200 * time.Millisecond)

	if monitor.IsHealthy() {
		t.Error("Monitor should be unhealthy initially")
	}

	if unhealthyCalled.Load() != 1 {
		t.Errorf("Expected 1 unhealthy call, got %d", unhealthyCalled.Load())
	}

	// Now start the server
	s := NewServer(os.Getpid() + 30005)
	s.socketPath = socketPath

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	// Wait for monitor to detect recovery
	time.Sleep(250 * time.Millisecond)

	if !monitor.IsHealthy() {
		t.Error("Monitor should recover to healthy state")
	}

	// Should not call onUnhealthy again (only on healthy→unhealthy transition)
	if unhealthyCalled.Load() != 1 {
		t.Errorf("onUnhealthy should not be called on recovery, got %d calls total", unhealthyCalled.Load())
	}
}

// TestHealthMonitor_NilCallback verifies nil callback doesn't panic
func TestHealthMonitor_NilCallback(t *testing.T) {
	c := NewClientWithPath("/tmp/nil-callback-socket.sock")

	// Create monitor with nil callback
	monitor := NewHealthMonitor(c, 50*time.Millisecond, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	monitor.Start(ctx)

	// Wait for health checks to run
	time.Sleep(150 * time.Millisecond)

	// Should transition to unhealthy without panicking
	if monitor.IsHealthy() {
		t.Error("Monitor should be unhealthy")
	}

	// If we get here without panic, test passes
}
