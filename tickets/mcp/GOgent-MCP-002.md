---
id: GOgent-MCP-002
title: "Callback Client Library"
time: "2 hours"
priority: HIGH
dependencies: "GOgent-MCP-001"
status: pending
---

# GOgent-MCP-002: Callback Client Library


**Time:** 2 hours
**Dependencies:** GOgent-MCP-001
**Priority:** HIGH

**Task:**
Implement an HTTP client that the MCP server uses to communicate with the TUI's Unix socket server.

**File:** `internal/callback/client.go`

**Imports:**
```go
package callback

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "time"
)
```

**Implementation:**
```go
// Client communicates with the TUI callback server
type Client struct {
    httpClient *http.Client
    socketPath string
}

// NewClient creates a callback client from environment
func NewClient() (*Client, error) {
    socketPath := os.Getenv("GOFORTRESS_SOCKET")
    if socketPath == "" {
        return nil, fmt.Errorf("[callback-client] GOFORTRESS_SOCKET not set. MCP server must be spawned by gofortress.")
    }

    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                dialer := net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(ctx, "unix", socketPath)
            },
            MaxIdleConns:      5,
            IdleConnTimeout:   90 * time.Second,
            DisableKeepAlives: false,
        },
        Timeout: 5 * time.Minute, // Long timeout for user interaction
    }

    return &Client{
        httpClient: client,
        socketPath: socketPath,
    }, nil
}

// NewClientWithPath creates a client with explicit socket path
func NewClientWithPath(socketPath string) *Client {
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                dialer := net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(ctx, "unix", socketPath)
            },
        },
        Timeout: 5 * time.Minute,
    }

    return &Client{
        httpClient: client,
        socketPath: socketPath,
    }
}

// SendPrompt sends a prompt request and waits for response
func (c *Client) SendPrompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://unix/prompt", bytes.NewReader(body))
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Request failed: %w. Verify TUI is running.", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return PromptResponse{}, fmt.Errorf("[callback-client] Server returned %d", resp.StatusCode)
    }

    var response PromptResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to decode response: %w", err)
    }

    return response, nil
}

// HealthCheck verifies the TUI server is reachable
func (c *Client) HealthCheck(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "http://unix/health", nil)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("[callback-client] Health check failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("[callback-client] Health check returned %d", resp.StatusCode)
    }
    return nil
}
```

**Tests:**
```go
package callback

import (
    "context"
    "os"
    "testing"
    "time"
)

func TestClient_WithServer(t *testing.T) {
    // Start server
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

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

func TestClient_MissingSocket(t *testing.T) {
    // Unset environment variable
    os.Unsetenv("GOFORTRESS_SOCKET")

    _, err := NewClient()
    if err == nil {
        t.Error("Expected error for missing socket path")
    }
}
```

**Acceptance Criteria:**
- [ ] Client created from GOFORTRESS_SOCKET env var
- [ ] Health check returns nil on healthy server
- [ ] SendPrompt returns user response
- [ ] Proper error messages with context
- [ ] Connection timeout handling

**Test Deliverables:**
- [ ] Test file created: `internal/callback/client_test.go`
- [ ] Coverage achieved: >85%
- [ ] Tests passing

**Why This Matters:**
The client library is used by the MCP server binary to call back to the TUI. Clean abstraction here makes the MCP server implementation straightforward.


