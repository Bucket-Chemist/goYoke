package callback

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestClient_WithServer verifies client-server integration
func TestClient_WithServer(t *testing.T) {
	// Start server with unique PID to avoid conflicts
	s := NewServer(os.Getpid() + 20000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Create client
	c := NewClientWithPath(s.SocketPath())

	// Test health check
	if err := c.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test prompt round-trip
	go func() {
		req := <-s.PromptChan
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: "user input",
		})
	}()

	resp, err := c.SendPrompt(ctx, PromptRequest{
		ID:      "test-client-1",
		Type:    "ask",
		Message: "Test?",
	})
	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}

	if resp.Value != "user input" {
		t.Errorf("Expected 'user input', got %q", resp.Value)
	}
}

// TestClient_MissingSocket verifies error handling when env var is not set
func TestClient_MissingSocket(t *testing.T) {
	// Save and restore env var
	oldSocket := os.Getenv("GOFORTRESS_SOCKET")
	defer func() {
		if oldSocket != "" {
			os.Setenv("GOFORTRESS_SOCKET", oldSocket)
		} else {
			os.Unsetenv("GOFORTRESS_SOCKET")
		}
	}()

	// Unset environment variable
	os.Unsetenv("GOFORTRESS_SOCKET")

	_, err := NewClient()
	if err == nil {
		t.Error("Expected error for missing socket path")
	}
	if err != nil && err.Error() != "[callback-client] GOFORTRESS_SOCKET not set. MCP server must be spawned by gofortress." {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestClient_NewClientWithEnv verifies environment-based client creation
func TestClient_NewClientWithEnv(t *testing.T) {
	s := NewServer(os.Getpid() + 21000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Save and restore env var
	oldSocket := os.Getenv("GOFORTRESS_SOCKET")
	defer func() {
		if oldSocket != "" {
			os.Setenv("GOFORTRESS_SOCKET", oldSocket)
		} else {
			os.Unsetenv("GOFORTRESS_SOCKET")
		}
	}()

	// Set environment variable to server's socket path
	os.Setenv("GOFORTRESS_SOCKET", s.SocketPath())

	// Create client from environment
	c, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client from env: %v", err)
	}

	// Verify health check works
	if err := c.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// TestClient_HealthCheckUnreachable verifies error when server is unreachable
func TestClient_HealthCheckUnreachable(t *testing.T) {
	// Create client pointing to non-existent socket
	c := NewClientWithPath("/tmp/nonexistent-socket.sock")
	ctx := context.Background()

	err := c.HealthCheck(ctx)
	if err == nil {
		t.Error("Expected error for unreachable server")
	}
}

// TestClient_SendPromptTimeout verifies timeout handling
func TestClient_SendPromptTimeout(t *testing.T) {
	s := NewServer(os.Getpid() + 22000)
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

	// Create context with short timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Send prompt but don't respond - should timeout
	_, err := c.SendPrompt(ctxTimeout, PromptRequest{
		ID:      "timeout-test",
		Type:    "ask",
		Message: "This will timeout",
	})

	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestClient_SendPromptWithOptions verifies prompts with options are transmitted correctly
func TestClient_SendPromptWithOptions(t *testing.T) {
	s := NewServer(os.Getpid() + 23000)
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

	// Verify options are received correctly
	go func() {
		req := <-s.PromptChan
		if len(req.Options) != 3 {
			t.Errorf("Expected 3 options, got %d", len(req.Options))
		}
		if req.Default != "b" {
			t.Errorf("Expected default 'b', got %q", req.Default)
		}
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: req.Options[1],
		})
	}()

	resp, err := c.SendPrompt(ctx, PromptRequest{
		ID:      "options-test",
		Type:    "select",
		Message: "Choose:",
		Options: []string{"a", "b", "c"},
		Default: "b",
	})

	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}

	if resp.Value != "b" {
		t.Errorf("Expected 'b', got %q", resp.Value)
	}
}

// TestClient_SendPromptCancellation verifies context cancellation handling
func TestClient_SendPromptCancellation(t *testing.T) {
	s := NewServer(os.Getpid() + 24000)
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

	// Create cancellable context
	ctxCancel, cancel := context.WithCancel(ctx)

	// Cancel immediately
	cancel()

	// Attempt to send prompt with cancelled context
	_, err := c.SendPrompt(ctxCancel, PromptRequest{
		ID:      "cancel-test",
		Type:    "ask",
		Message: "This will be cancelled",
	})

	if err == nil {
		t.Error("Expected cancellation error")
	}
}

// TestClient_MultiplePrompts verifies multiple sequential prompts work
func TestClient_MultiplePrompts(t *testing.T) {
	s := NewServer(os.Getpid() + 25000)
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

	// Handler for multiple prompts
	go func() {
		for i := 0; i < 3; i++ {
			req := <-s.PromptChan
			s.SendResponse(PromptResponse{
				ID:    req.ID,
				Value: req.Message + " response",
			})
		}
	}()

	// Send three prompts sequentially
	for i := 0; i < 3; i++ {
		resp, err := c.SendPrompt(ctx, PromptRequest{
			ID:      "multi-test",
			Type:    "ask",
			Message: "Prompt" + string(rune('1'+i)),
		})

		if err != nil {
			t.Fatalf("Prompt %d failed: %v", i+1, err)
		}

		expectedPrefix := "Prompt" + string(rune('1'+i))
		if resp.Value != expectedPrefix+" response" {
			t.Errorf("Prompt %d: expected %q, got %q", i+1, expectedPrefix+" response", resp.Value)
		}
	}
}

// TestClient_InvalidSocketPath verifies error with invalid socket path
func TestClient_InvalidSocketPath(t *testing.T) {
	// Create client with invalid path
	c := NewClientWithPath("")
	ctx := context.Background()

	err := c.HealthCheck(ctx)
	if err == nil {
		t.Error("Expected error for empty socket path")
	}
}

// TestClient_ServerError verifies handling of non-200 responses
func TestClient_ServerError(t *testing.T) {
	// This test would require a mock server that returns errors
	// For now, we verify the error path exists by testing unreachable server
	c := NewClientWithPath("/tmp/error-test-socket.sock")
	ctx := context.Background()

	_, err := c.SendPrompt(ctx, PromptRequest{
		ID:      "error-test",
		Type:    "ask",
		Message: "This should fail",
	})

	if err == nil {
		t.Error("Expected error for unreachable server")
	}
}

// TestClient_HealthCheckMultipleTimes verifies repeated health checks work
func TestClient_HealthCheckMultipleTimes(t *testing.T) {
	s := NewServer(os.Getpid() + 26000)
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

	// Perform multiple health checks
	for i := 0; i < 5; i++ {
		if err := c.HealthCheck(ctx); err != nil {
			t.Errorf("Health check %d failed: %v", i+1, err)
		}
	}
}
