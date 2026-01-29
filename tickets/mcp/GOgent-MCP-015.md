---
id: GOgent-MCP-015
title: "Comprehensive Test Suite"
time: "6 hours"
priority: HIGH
dependencies: "All previous tickets"
status: pending
---

# GOgent-MCP-015: Comprehensive Test Suite


**Time:** 6 hours
**Dependencies:** All previous tickets
**Priority:** HIGH

**Task:**
Create comprehensive test suite covering all MCP integration scenarios.

**File:** `internal/mcp/integration_test.go`

**Implementation:**
```go
package mcp

import (
    "context"
    "os"
    "sync"
    "testing"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

func TestMCPIntegration_FullRoundTrip(t *testing.T) {
    // Start callback server
    server := callback.NewServer(os.Getpid())
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    // Create client
    client := callback.NewClientWithPath(server.SocketPath())

    // Simulate TUI handling prompts
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for req := range server.PromptChan {
            // Simulate user response
            time.Sleep(50 * time.Millisecond)
            server.SendResponse(callback.PromptResponse{
                ID:    req.ID,
                Value: "user-response",
            })
        }
    }()

    // Send multiple prompts
    for i := 0; i < 5; i++ {
        resp, err := client.SendPrompt(ctx, callback.PromptRequest{
            Type:    "ask",
            Message: "Test question?",
        })
        if err != nil {
            t.Errorf("Prompt %d failed: %v", i, err)
        }
        if resp.Value != "user-response" {
            t.Errorf("Expected 'user-response', got %q", resp.Value)
        }
    }
}

func TestMCPIntegration_ConcurrentPrompts(t *testing.T) {
    server := callback.NewServer(os.Getpid())
    ctx := context.Background()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    client := callback.NewClientWithPath(server.SocketPath())

    // Handle prompts
    go func() {
        for req := range server.PromptChan {
            server.SendResponse(callback.PromptResponse{
                ID:    req.ID,
                Value: req.ID, // Echo back the ID
            })
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
                errors <- err
                return
            }
            if resp.Value != id {
                errors <- fmt.Errorf("ID mismatch: expected %s, got %s", id, resp.Value)
            }
        }(fmt.Sprintf("concurrent-%d", i))
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Error(err)
    }
}

func TestMCPIntegration_Timeout(t *testing.T) {
    server := callback.NewServer(os.Getpid())
    ctx := context.Background()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    client := callback.NewClientWithPath(server.SocketPath())
    client.httpClient.Timeout = 100 * time.Millisecond

    // Don't handle prompt - should timeout
    go func() {
        <-server.PromptChan // Receive but don't respond
    }()

    _, err := client.SendPrompt(ctx, callback.PromptRequest{
        Type:    "ask",
        Message: "Should timeout",
    })

    if err == nil {
        t.Error("Expected timeout error, got nil")
    }
}
```

**Acceptance Criteria:**
- [ ] Full round-trip test passes
- [ ] Concurrent prompt handling works
- [ ] Timeout behavior correct
- [ ] Memory leak tests (24h simulation)
- [ ] Coverage >80% for all MCP packages


