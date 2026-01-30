package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

// waitForSocket waits for socket to be ready to accept HTTP requests
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 500 * time.Millisecond,
	}

	for time.Now().Before(deadline) {
		// Try a health check request
		resp, err := client.Get("http://unix/health")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return os.ErrNotExist
}

func TestServer_StartAndShutdown(t *testing.T) {
	// Use unique PID to avoid conflicts between parallel tests
	s := NewServer(os.Getpid() + 1000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()

	// Verify socket exists
	if _, err := os.Stat(s.SocketPath()); os.IsNotExist(err) {
		t.Error("Socket file not created")
	}

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown error: %v", err)
	}
}

func TestServer_HealthCheck(t *testing.T) {
	// Use unique PID to avoid conflicts between parallel tests
	s := NewServer(os.Getpid() + 2000)
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

	// Create HTTP client using Unix socket
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
	}

	resp, err := client.Get("http://unix/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestServer_PromptRoundTrip(t *testing.T) {
	// Use unique PID to avoid conflicts between parallel tests
	s := NewServer(os.Getpid() + 3000)
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

	// Start goroutine to handle prompt
	go func() {
		req := <-s.PromptChan
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: "test response",
		})
	}()

	// Send prompt request
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
		Timeout: 5 * time.Second,
	}

	reqBody, _ := json.Marshal(PromptRequest{
		ID:      "test-1",
		Type:    "ask",
		Message: "Test question?",
	})

	resp, err := client.Post("http://unix/prompt", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Prompt request failed: %v", err)
	}
	defer resp.Body.Close()

	var response PromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Value != "test response" {
		t.Errorf("Expected 'test response', got %q", response.Value)
	}
}

func TestServer_SocketPermissions(t *testing.T) {
	s := NewServer(os.Getpid() + 4000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	// Check socket permissions are 0600
	info, err := os.Stat(s.SocketPath())
	if err != nil {
		t.Fatalf("Failed to stat socket: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestServer_ConfirmEndpoint(t *testing.T) {
	s := NewServer(os.Getpid() + 5000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Start goroutine to handle confirm
	go func() {
		req := <-s.PromptChan
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: "yes",
		})
	}()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
		Timeout: 5 * time.Second,
	}

	reqBody, _ := json.Marshal(PromptRequest{
		ID:      "confirm-1",
		Type:    "confirm",
		Message: "Proceed?",
	})

	resp, err := client.Post("http://unix/confirm", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Confirm request failed: %v", err)
	}
	defer resp.Body.Close()

	var response PromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Value != "yes" {
		t.Errorf("Expected 'yes', got %q", response.Value)
	}
}

func TestServer_PromptWithOptions(t *testing.T) {
	s := NewServer(os.Getpid() + 6000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Verify prompt with options and default
	go func() {
		req := <-s.PromptChan
		if len(req.Options) != 3 {
			t.Errorf("Expected 3 options, got %d", len(req.Options))
		}
		if req.Default != "option2" {
			t.Errorf("Expected default 'option2', got %q", req.Default)
		}
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: req.Default,
		})
	}()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
		Timeout: 5 * time.Second,
	}

	reqBody, _ := json.Marshal(PromptRequest{
		ID:      "select-1",
		Type:    "select",
		Message: "Choose one:",
		Options: []string{"option1", "option2", "option3"},
		Default: "option2",
	})

	resp, err := client.Post("http://unix/prompt", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Prompt request failed: %v", err)
	}
	defer resp.Body.Close()

	var response PromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Value != "option2" {
		t.Errorf("Expected 'option2', got %q", response.Value)
	}
}

func TestServer_InvalidMethod(t *testing.T) {
	s := NewServer(os.Getpid() + 7000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
	}

	// GET on /prompt should fail (requires POST)
	resp, err := client.Get("http://unix/prompt")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", resp.StatusCode)
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	s := NewServer(os.Getpid() + 8000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
	}

	// Send invalid JSON
	resp, err := client.Post("http://unix/prompt", "application/json", bytes.NewReader([]byte("invalid json")))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestServer_DoubleStart(t *testing.T) {
	s := NewServer(os.Getpid() + 9000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	// Try to start again - should error
	if err := s.Start(ctx); err == nil {
		t.Error("Expected error when starting server twice")
	}
}

func TestServer_SendResponseToNonexistentPrompt(t *testing.T) {
	s := NewServer(os.Getpid() + 10000)

	// Send response without starting server or creating pending prompt
	err := s.SendResponse(PromptResponse{
		ID:    "nonexistent",
		Value: "test",
	})

	if err == nil {
		t.Error("Expected error when sending response to nonexistent prompt")
	}
}

func TestServer_PromptGeneratesID(t *testing.T) {
	s := NewServer(os.Getpid() + 11000)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer s.Cleanup()
	defer s.Shutdown(ctx)

	if err := waitForSocket(s.SocketPath(), 2*time.Second); err != nil {
		t.Fatalf("Socket not ready: %v", err)
	}

	// Start goroutine to handle prompt and verify ID was generated
	go func() {
		req := <-s.PromptChan
		// Verify ID was auto-generated (non-empty)
		if req.ID == "" {
			t.Error("Expected auto-generated ID, got empty string")
		}
		s.SendResponse(PromptResponse{
			ID:    req.ID,
			Value: "response",
		})
	}()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.SocketPath())
			},
		},
		Timeout: 5 * time.Second,
	}

	// Send request WITHOUT ID field
	reqBody, _ := json.Marshal(PromptRequest{
		Type:    "ask",
		Message: "Question without ID?",
	})

	resp, err := client.Post("http://unix/prompt", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Prompt request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestServer_XDGRuntimeDirPath(t *testing.T) {
	// Test socket path with XDG_RUNTIME_DIR set
	oldXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_RUNTIME_DIR", oldXDG)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
	}()

	os.Setenv("XDG_RUNTIME_DIR", "/test/runtime")
	s := NewServer(12345)
	expectedPath := "/test/runtime/gofortress-12345.sock"

	if s.SocketPath() != expectedPath {
		t.Errorf("Expected socket path %q, got %q", expectedPath, s.SocketPath())
	}
}

func TestServer_TempDirFallback(t *testing.T) {
	// Test socket path fallback to temp dir when XDG_RUNTIME_DIR is not set
	oldXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_RUNTIME_DIR", oldXDG)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
	}()

	os.Unsetenv("XDG_RUNTIME_DIR")
	s := NewServer(12345)

	// Should use os.TempDir()
	if !bytes.Contains([]byte(s.SocketPath()), []byte(os.TempDir())) {
		t.Errorf("Expected socket path to contain temp dir %q, got %q", os.TempDir(), s.SocketPath())
	}
	if !bytes.Contains([]byte(s.SocketPath()), []byte("gofortress-12345.sock")) {
		t.Errorf("Expected socket path to contain 'gofortress-12345.sock', got %q", s.SocketPath())
	}
}

func TestServer_ShutdownNotStarted(t *testing.T) {
	s := NewServer(os.Getpid() + 12000)
	ctx := context.Background()

	// Shutdown without starting should not error
	if err := s.Shutdown(ctx); err != nil {
		t.Errorf("Unexpected error shutting down non-started server: %v", err)
	}
}
