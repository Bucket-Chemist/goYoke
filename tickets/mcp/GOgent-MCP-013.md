---
id: GOgent-MCP-013
title: "Error Handling and Recovery"
description: "Implement comprehensive error handling with retries and graceful degradation for callback client"
time_estimate: "4h"
priority: MEDIUM
dependencies: ["GOgent-MCP-009"]
status: completed
---

# GOgent-MCP-013: Error Handling and Recovery


**Time:** 4 hours
**Dependencies:** Phase 3
**Priority:** MEDIUM

**Task:**
Implement comprehensive error handling with retries and graceful degradation.

**File:** `internal/callback/recovery.go`

**Implementation:**
```go
package callback

import (
    "context"
    "fmt"
    "time"
)

// SendPromptWithRetry attempts to send a prompt with exponential backoff
func (c *Client) SendPromptWithRetry(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    var lastErr error
    backoff := 100 * time.Millisecond
    maxRetries := 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := c.SendPrompt(ctx, req)
        if err == nil {
            return resp, nil
        }
        lastErr = err

        // Don't retry on context cancellation
        if ctx.Err() != nil {
            return PromptResponse{}, ctx.Err()
        }

        select {
        case <-ctx.Done():
            return PromptResponse{}, ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
        }
    }

    return PromptResponse{}, fmt.Errorf("[callback-client] Max retries exceeded: %w", lastErr)
}

// ServerHealthMonitor periodically checks server health
type ServerHealthMonitor struct {
    client   *Client
    interval time.Duration
    healthy  bool
    onUnhealthy func()
}

func NewHealthMonitor(client *Client, interval time.Duration, onUnhealthy func()) *ServerHealthMonitor {
    return &ServerHealthMonitor{
        client:      client,
        interval:    interval,
        healthy:     true,
        onUnhealthy: onUnhealthy,
    }
}

func (m *ServerHealthMonitor) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(m.interval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                err := m.client.HealthCheck(checkCtx)
                cancel()

                wasHealthy := m.healthy
                m.healthy = err == nil

                if wasHealthy && !m.healthy && m.onUnhealthy != nil {
                    m.onUnhealthy()
                }
            }
        }
    }()
}

func (m *ServerHealthMonitor) IsHealthy() bool {
    return m.healthy
}
```

**Acceptance Criteria:**
- [x] Retries with exponential backoff
- [x] Respects context cancellation
- [x] Health monitor detects failures
- [x] Callback on health state change


