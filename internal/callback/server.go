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

// RegisterPending registers a response channel for a prompt ID.
// This allows the TUI to register channels for prompts it will handle.
func (s *Server) RegisterPending(id string, ch chan PromptResponse) {
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()
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
