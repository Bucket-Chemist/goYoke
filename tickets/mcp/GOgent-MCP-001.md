---
id: GOgent-MCP-001
title: "Unix Socket HTTP Server"
time: "4 hours"
priority: HIGH
dependencies: "None"
status: pending
---

# GOgent-MCP-001: Unix Socket HTTP Server


**Time:** 4 hours
**Dependencies:** None
**Priority:** HIGH (critical path)

**Task:**
Implement an HTTP server that listens on a Unix socket for receiving prompt requests from the MCP server.

**File:** `internal/callback/server.go`

**Imports:**
```go
package callback

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)
```

**Implementation:**
```go
// PromptRequest represents a request from the MCP server
type PromptRequest struct {
    ID      string   `json:"id"`
    Type    string   `json:"type"`    // "ask", "confirm", "input", "select"
    Message string   `json:"message"`
    Options []string `json:"options,omitempty"`
    Default string   `json:"default,omitempty"`
}

// PromptResponse represents the TUI's response
type PromptResponse struct {
    ID        string `json:"id"`
    Value     string `json:"value"`
    Cancelled bool   `json:"cancelled"`
    Error     string `json:"error,omitempty"`
}

// Server handles HTTP requests over Unix socket
type Server struct {
    socketPath string
    listener   net.Listener
    httpServer *http.Server

    // Channel for sending prompts to TUI
    PromptChan chan PromptRequest

    // Map of pending responses (keyed by prompt ID)
    pending   map[string]chan PromptResponse
    pendingMu sync.RWMutex

    // Lifecycle
    started bool
    mu      sync.Mutex
}

// NewServer creates a new callback server
func NewServer(pid int) *Server {
    socketPath := getSocketPath(pid)
    return &Server{
        socketPath: socketPath,
        PromptChan: make(chan PromptRequest, 10),
        pending:    make(map[string]chan PromptResponse),
    }
}

// getSocketPath returns the socket path for the given PID
func getSocketPath(pid int) string {
    // Prefer XDG_RUNTIME_DIR for better security (per-user, in-memory)
    if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
        return filepath.Join(runtimeDir, fmt.Sprintf("gofortress-%d.sock", pid))
    }
    return filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-%d.sock", pid))
}

// Start begins listening for HTTP requests
func (s *Server) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.started {
        return fmt.Errorf("[callback] Server already started")
    }

    // Remove stale socket
    os.Remove(s.socketPath)

    var err error
    s.listener, err = net.Listen("unix", s.socketPath)
    if err != nil {
        return fmt.Errorf("[callback] Failed to listen on %s: %w. Check permissions and path length.", s.socketPath, err)
    }

    // Set restrictive permissions (owner only)
    if err := os.Chmod(s.socketPath, 0600); err != nil {
        s.listener.Close()
        return fmt.Errorf("[callback] Failed to set socket permissions: %w", err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/prompt", s.handlePrompt)
    mux.HandleFunc("/confirm", s.handleConfirm)
    mux.HandleFunc("/health", s.handleHealth)

    s.httpServer = &http.Server{
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 5 * time.Minute, // Long timeout for user interaction
        BaseContext:  func(_ net.Listener) context.Context { return ctx },
    }

    go func() {
        if err := s.httpServer.Serve(s.listener); err != http.ErrServerClosed {
            // Log error but don't crash - graceful degradation
            fmt.Fprintf(os.Stderr, "[callback] Server error: %v\n", err)
        }
    }()

    s.started = true
    return nil
}

// handlePrompt processes prompt requests from MCP server
func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req PromptRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
        return
    }

    // Generate ID if not provided
    if req.ID == "" {
        req.ID = fmt.Sprintf("prompt-%d", time.Now().UnixNano())
    }

    // Create response channel for this prompt
    respChan := make(chan PromptResponse, 1)
    s.pendingMu.Lock()
    s.pending[req.ID] = respChan
    s.pendingMu.Unlock()

    defer func() {
        s.pendingMu.Lock()
        delete(s.pending, req.ID)
        s.pendingMu.Unlock()
    }()

    // Send to TUI for display
    select {
    case s.PromptChan <- req:
        // Sent successfully
    case <-r.Context().Done():
        http.Error(w, "Request cancelled", http.StatusRequestTimeout)
        return
    case <-time.After(5 * time.Second):
        http.Error(w, "TUI not responding", http.StatusServiceUnavailable)
        return
    }

    // Wait for user response (long-poll)
    select {
    case resp := <-respChan:
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    case <-r.Context().Done():
        http.Error(w, "Timeout waiting for user response", http.StatusGatewayTimeout)
    }
}

// handleConfirm processes yes/no confirmation requests
func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
    // Reuse prompt handler with type="confirm"
    s.handlePrompt(w, r)
}

// handleHealth returns server status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SendResponse sends a response for a pending prompt
func (s *Server) SendResponse(resp PromptResponse) error {
    s.pendingMu.RLock()
    ch, ok := s.pending[resp.ID]
    s.pendingMu.RUnlock()

    if !ok {
        return fmt.Errorf("[callback] No pending prompt with ID: %s", resp.ID)
    }

    select {
    case ch <- resp:
        return nil
    default:
        return fmt.Errorf("[callback] Response channel full for prompt: %s", resp.ID)
    }
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !s.started {
        return nil
    }

    if s.httpServer != nil {
        if err := s.httpServer.Shutdown(ctx); err != nil {
            return fmt.Errorf("[callback] Shutdown error: %w", err)
        }
    }

    s.started = false
    return nil
}

// Cleanup removes the socket file
func (s *Server) Cleanup() {
    os.Remove(s.socketPath)
}

// SocketPath returns the socket path
func (s *Server) SocketPath() string {
    return s.socketPath
}
```

**Tests:**
```go
package callback

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "os"
    "testing"
    "time"
)

func TestServer_StartAndShutdown(t *testing.T) {
    s := NewServer(os.Getpid())
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
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

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
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

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
```

**Acceptance Criteria:**
- [ ] Server starts and listens on Unix socket
- [ ] Socket has 0600 permissions (owner only)
- [ ] Health endpoint returns 200 OK
- [ ] Prompt endpoint blocks until response sent
- [ ] SendResponse delivers to correct pending channel
- [ ] Graceful shutdown closes connections
- [ ] Cleanup removes socket file

**Test Deliverables:**
- [ ] Test file created: `internal/callback/server_test.go`
- [ ] Number of test functions: 3+
- [ ] Coverage achieved: >85%
- [ ] Tests passing: `go test ./internal/callback/...`
- [ ] Race detector clean: `go test -race ./internal/callback/...`

**Why This Matters:**
This is the IPC backbone that enables the three-process architecture. Without reliable socket communication, the MCP server cannot call back to the TUI for user prompts.


