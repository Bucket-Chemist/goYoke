package mcp

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/callback"
)

// TestMCPIntegration_FullRoundTrip tests complete request-response cycle
// with multiple sequential prompts
func TestMCPIntegration_FullRoundTrip(t *testing.T) {
	// Start callback server with unique PID to avoid conflicts
	server := callback.NewServer(os.Getpid() + 30000)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Create client
	client := callback.NewClientWithPath(server.SocketPath())

	// Simulate TUI handling prompts in background
	var wg sync.WaitGroup
	wg.Add(1)
	promptsDone := make(chan struct{})

	go func() {
		defer wg.Done()
		promptCount := 0
		for {
			select {
			case req := <-server.PromptChan:
				// Simulate user thinking time
				time.Sleep(50 * time.Millisecond)

				// Send response back
				if err := server.SendResponse(callback.PromptResponse{
					ID:    req.ID,
					Value: "user-response",
				}); err != nil {
					t.Errorf("Failed to send response: %v", err)
				}

				promptCount++
				if promptCount >= 5 {
					return
				}
			case <-promptsDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send multiple prompts sequentially
	for i := 0; i < 5; i++ {
		resp, err := client.SendPrompt(ctx, callback.PromptRequest{
			Type:    "ask",
			Message: fmt.Sprintf("Test question %d?", i+1),
		})
		if err != nil {
			t.Errorf("Prompt %d failed: %v", i+1, err)
			continue
		}
		if resp.Value != "user-response" {
			t.Errorf("Prompt %d: expected 'user-response', got %q", i+1, resp.Value)
		}
	}

	close(promptsDone)
	wg.Wait()
}

// TestMCPIntegration_ConcurrentPrompts verifies concurrent prompt handling
// without ID conflicts or race conditions
func TestMCPIntegration_ConcurrentPrompts(t *testing.T) {
	server := callback.NewServer(os.Getpid() + 31000)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Handle prompts in background - echo ID back
	go func() {
		for {
			select {
			case req := <-server.PromptChan:
				// Echo the ID back to verify no ID mixup
				if err := server.SendResponse(callback.PromptResponse{
					ID:    req.ID,
					Value: req.ID,
				}); err != nil {
					t.Logf("Failed to send response: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send concurrent prompts
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			resp, err := client.SendPrompt(ctx, callback.PromptRequest{
				ID:      id,
				Type:    "ask",
				Message: "Concurrent test",
			})
			if err != nil {
				errors <- fmt.Errorf("prompt %s failed: %w", id, err)
				return
			}
			if resp.Value != id {
				errors <- fmt.Errorf("ID mismatch: expected %s, got %s", id, resp.Value)
			}
		}(fmt.Sprintf("concurrent-%d", i))
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

// TestMCPIntegration_Timeout verifies timeout behavior when TUI doesn't respond
func TestMCPIntegration_Timeout(t *testing.T) {
	server := callback.NewServer(os.Getpid() + 32000)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Set short client timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Receive prompt but don't respond - simulates timeout
	promptReceived := make(chan struct{})
	go func() {
		<-server.PromptChan // Receive but don't respond
		close(promptReceived)
	}()

	// Send prompt - should timeout
	_, err := client.SendPrompt(ctxTimeout, callback.PromptRequest{
		Type:    "ask",
		Message: "Should timeout",
	})

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Verify prompt was actually received
	select {
	case <-promptReceived:
		// Good - prompt was received, just not responded to
	case <-time.After(1 * time.Second):
		t.Error("Prompt was not received by server")
	}
}

// TestMCPIntegration_MemoryLeak verifies no goroutine leaks after many operations
func TestMCPIntegration_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	server := callback.NewServer(os.Getpid() + 33000)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Handle prompts automatically
	go func() {
		for {
			select {
			case req := <-server.PromptChan:
				server.SendResponse(callback.PromptResponse{
					ID:    req.ID,
					Value: "response",
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Run many requests
	for i := 0; i < 100; i++ {
		_, err := client.SendPrompt(ctx, callback.PromptRequest{
			Type:    "ask",
			Message: "Memory test",
		})
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// Allow cleanup
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	// Check goroutine count hasn't grown unbounded
	finalGoroutines := runtime.NumGoroutine()
	growth := finalGoroutines - baselineGoroutines

	// Allow some growth (buffering, cleanup delays) but not unbounded
	// Threshold of 10 is reasonable for this test
	if growth > 10 {
		t.Errorf("Potential goroutine leak: baseline=%d, final=%d, growth=%d",
			baselineGoroutines, finalGoroutines, growth)
	}
}

// TestMCPIntegration_MixedPromptTypes verifies different prompt types work correctly
func TestMCPIntegration_MixedPromptTypes(t *testing.T) {
	server := callback.NewServer(os.Getpid() + 34000)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Handle different prompt types
	go func() {
		for {
			select {
			case req := <-server.PromptChan:
				var response string
				switch req.Type {
				case "ask":
					response = "answer"
				case "confirm":
					response = "yes"
				case "select":
					if len(req.Options) > 0 {
						response = req.Options[0]
					} else {
						response = "default"
					}
				case "input":
					response = "user input"
				default:
					response = "unknown"
				}

				server.SendResponse(callback.PromptResponse{
					ID:    req.ID,
					Value: response,
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	// Test different prompt types
	testCases := []struct {
		name     string
		request  callback.PromptRequest
		expected string
	}{
		{
			name: "ask",
			request: callback.PromptRequest{
				Type:    "ask",
				Message: "Question?",
			},
			expected: "answer",
		},
		{
			name: "confirm",
			request: callback.PromptRequest{
				Type:    "confirm",
				Message: "Proceed?",
			},
			expected: "yes",
		},
		{
			name: "select",
			request: callback.PromptRequest{
				Type:    "select",
				Message: "Choose:",
				Options: []string{"option1", "option2", "option3"},
			},
			expected: "option1",
		},
		{
			name: "input",
			request: callback.PromptRequest{
				Type:    "input",
				Message: "Enter value:",
			},
			expected: "user input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.SendPrompt(ctx, tc.request)
			if err != nil {
				t.Fatalf("Prompt failed: %v", err)
			}
			if resp.Value != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, resp.Value)
			}
		})
	}
}

// TestMCPIntegration_ErrorResponse verifies error handling in responses
func TestMCPIntegration_ErrorResponse(t *testing.T) {
	server := callback.NewServer(os.Getpid() + 35000)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Send error response
	go func() {
		req := <-server.PromptChan
		server.SendResponse(callback.PromptResponse{
			ID:    req.ID,
			Error: "simulated error",
		})
	}()

	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		Type:    "ask",
		Message: "This will error",
	})

	// Client should successfully receive the response (no network error)
	if err != nil {
		t.Fatalf("Expected no error from client, got: %v", err)
	}

	// But the response should contain the error
	if resp.Error != "simulated error" {
		t.Errorf("Expected error 'simulated error', got %q", resp.Error)
	}
}

// TestMCPIntegration_CancelledResponse verifies cancelled flag handling
func TestMCPIntegration_CancelledResponse(t *testing.T) {
	server := callback.NewServer(os.Getpid() + 36000)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Wait for server to be ready
	if err := waitForSocket(server.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := callback.NewClientWithPath(server.SocketPath())

	// Send cancelled response
	go func() {
		req := <-server.PromptChan
		server.SendResponse(callback.PromptResponse{
			ID:        req.ID,
			Cancelled: true,
		})
	}()

	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		Type:    "ask",
		Message: "User will cancel this",
	})

	if err != nil {
		t.Fatalf("Expected no error from client, got: %v", err)
	}

	if !resp.Cancelled {
		t.Error("Expected Cancelled=true, got false")
	}
}

// waitForSocket waits for socket to be ready to accept HTTP requests
// Helper function reused from callback tests
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if socket file exists
		if _, err := os.Stat(socketPath); err == nil {
			// Socket exists, give it a moment to start accepting connections
			time.Sleep(50 * time.Millisecond)
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("socket not ready after %v", timeout)
}
