# Callback Recovery Usage Guide

This document demonstrates how to use the retry logic and health monitoring features added in GOgent-MCP-013.

## SendPromptWithRetry

The `SendPromptWithRetry` method provides automatic retry with exponential backoff for transient network failures.

### Basic Usage

```go
import (
    "context"
    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

func example() error {
    client, err := callback.NewClient()
    if err != nil {
        return err
    }

    ctx := context.Background()

    // Use SendPromptWithRetry instead of SendPrompt for resilience
    resp, err := client.SendPromptWithRetry(ctx, callback.PromptRequest{
        ID:      "example-1",
        Type:    "ask",
        Message: "Continue with operation?",
    })

    if err != nil {
        // All 3 retries failed or context cancelled
        return err
    }

    // Process response
    if resp.Cancelled {
        return fmt.Errorf("user cancelled")
    }

    return nil
}
```

### Retry Behavior

- **Max retries:** 3 attempts
- **Backoff:** Exponential (100ms, 200ms, 400ms)
- **Total max time:** ~700ms for all retries
- **Cancellation:** Returns immediately on context cancellation (no retry)

### When to Use

Use `SendPromptWithRetry` when:
- Network conditions may be unreliable
- You want automatic recovery from transient failures
- The operation is idempotent (safe to retry)

Use regular `SendPrompt` when:
- You want immediate failure feedback
- You're implementing custom retry logic
- The operation should not be retried

## ServerHealthMonitor

The `ServerHealthMonitor` tracks callback server health and notifies you when it becomes unavailable.

### Basic Setup

```go
import (
    "context"
    "log"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

func main() {
    client, err := callback.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    // Create health monitor with 5-second check interval
    monitor := callback.NewHealthMonitor(
        client,
        5*time.Second,
        func() {
            // Called when server transitions from healthy to unhealthy
            log.Println("[WARNING] Callback server became unreachable")
            log.Println("[WARNING] TUI prompts will fail until server recovers")
        },
    )

    ctx := context.Background()

    // Start monitoring in background
    monitor.Start(ctx)

    // Check health status anytime
    if monitor.IsHealthy() {
        log.Println("Server is healthy")
    }

    // Application continues...
}
```

### Health Check Behavior

- Runs in background goroutine
- Periodic checks via ticker (configurable interval)
- Each check has 5-second timeout
- Tracks state transitions (healthy ↔ unhealthy)
- Calls `onUnhealthy` callback ONLY on transition (not on every failed check)
- Stops gracefully on context cancellation

### Integration Example

Here's how you might integrate health monitoring with the MCP server:

```go
func main() {
    client, err := callback.NewClient()
    if err != nil {
        logger.Fatal("Failed to create callback client", "error", err)
    }

    // Track health state for graceful degradation
    var serverHealthy atomic.Bool
    serverHealthy.Store(true)

    monitor := callback.NewHealthMonitor(
        client,
        10*time.Second,
        func() {
            serverHealthy.Store(false)
            logger.Warn("TUI server unreachable - prompts will be degraded")
        },
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    monitor.Start(ctx)

    // In your tool handlers, check health before operations
    if !serverHealthy.Load() {
        logger.Warn("Attempting prompt with unhealthy server")
        // Consider using SendPromptWithRetry here
    }

    // Or check monitor directly
    if !monitor.IsHealthy() {
        return errors.New("callback server unavailable")
    }
}
```

### Graceful Shutdown

```go
func main() {
    client, _ := callback.NewClient()
    monitor := callback.NewHealthMonitor(client, 5*time.Second, nil)

    ctx, cancel := context.WithCancel(context.Background())
    monitor.Start(ctx)

    // ... application runs ...

    // On shutdown, cancel context to stop health monitoring
    cancel()

    // Health monitor goroutine will exit cleanly
}
```

## Combining Retry and Health Monitoring

For maximum resilience, combine both features:

```go
type MCPServer struct {
    client  *callback.Client
    monitor *callback.ServerHealthMonitor
    healthy atomic.Bool
}

func NewMCPServer() (*MCPServer, error) {
    client, err := callback.NewClient()
    if err != nil {
        return nil, err
    }

    s := &MCPServer{client: client}
    s.healthy.Store(true)

    s.monitor = callback.NewHealthMonitor(
        client,
        10*time.Second,
        func() {
            s.healthy.Store(false)
            logger.Warn("Callback server became unavailable")
        },
    )

    return s, nil
}

func (s *MCPServer) Start(ctx context.Context) {
    s.monitor.Start(ctx)
}

func (s *MCPServer) AskUser(ctx context.Context, message string) (string, error) {
    // Use retry for transient failures
    resp, err := s.client.SendPromptWithRetry(ctx, callback.PromptRequest{
        Type:    "ask",
        Message: message,
    })

    if err != nil {
        // Log health state for debugging
        if !s.monitor.IsHealthy() {
            logger.Error("Prompt failed - server unhealthy", "error", err)
        } else {
            logger.Error("Prompt failed - unexpected", "error", err)
        }
        return "", err
    }

    return resp.Value, nil
}
```

## Error Handling

### Retry Errors

```go
resp, err := client.SendPromptWithRetry(ctx, req)
if err != nil {
    // Check if it's a cancellation
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("operation cancelled: %w", err)
    }

    // Max retries exceeded
    return fmt.Errorf("callback server unreachable after retries: %w", err)
}
```

### Health Transitions

```go
monitor := callback.NewHealthMonitor(client, 5*time.Second, func() {
    // Transition to unhealthy - take action

    // Option 1: Disable features that need callbacks
    disableInteractiveFeatures()

    // Option 2: Notify user
    logger.Error("Interactive prompts unavailable - TUI connection lost")

    // Option 3: Attempt reconnection
    go attemptReconnect()
})
```

## Testing

Both features are designed to be testable:

```go
func TestWithRetry(t *testing.T) {
    client := callback.NewClientWithPath("/tmp/test.sock")

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    _, err := client.SendPromptWithRetry(ctx, req)
    // Test retry behavior
}

func TestHealthMonitor(t *testing.T) {
    client := callback.NewClientWithPath("/tmp/test.sock")

    called := atomic.Bool{}
    monitor := callback.NewHealthMonitor(client, 100*time.Millisecond, func() {
        called.Store(true)
    })

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    monitor.Start(ctx)
    time.Sleep(200 * time.Millisecond)

    if !called.Load() {
        t.Error("Expected unhealthy callback")
    }
}
```

## Performance Considerations

### Retry Overhead

- First attempt: Normal latency
- Second attempt: +100ms backoff
- Third attempt: +200ms backoff
- Total overhead (all retries): ~700ms

Use timeouts to bound total retry time:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

resp, err := client.SendPromptWithRetry(ctx, req)
// Will return after 2s max, even if retries remain
```

### Health Monitor Overhead

- CPU: Minimal (goroutine sleeps on ticker)
- Network: One HTTP GET per interval
- Memory: ~1KB (goroutine stack + monitor struct)

Recommended intervals:
- Production: 10-30 seconds
- Development: 5-10 seconds
- Testing: 100ms-1s

## Migration Guide

### Updating Existing Code

Before (no retry):
```go
resp, err := client.SendPrompt(ctx, req)
if err != nil {
    return err
}
```

After (with retry):
```go
resp, err := client.SendPromptWithRetry(ctx, req)
if err != nil {
    return err
}
```

### Adding Health Monitoring

```diff
 func main() {
     client, err := callback.NewClient()
     if err != nil {
         log.Fatal(err)
     }
+
+    monitor := callback.NewHealthMonitor(client, 10*time.Second, func() {
+        log.Println("WARNING: Callback server unavailable")
+    })
+
+    ctx, cancel := context.WithCancel(context.Background())
+    defer cancel()
+
+    monitor.Start(ctx)

     // Rest of application
 }
```
