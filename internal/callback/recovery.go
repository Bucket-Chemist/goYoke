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
	client      *Client
	interval    time.Duration
	healthy     bool
	onUnhealthy func()
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(client *Client, interval time.Duration, onUnhealthy func()) *ServerHealthMonitor {
	return &ServerHealthMonitor{
		client:      client,
		interval:    interval,
		healthy:     true,
		onUnhealthy: onUnhealthy,
	}
}

// Start begins health monitoring in a background goroutine
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

// IsHealthy returns the current health status
func (m *ServerHealthMonitor) IsHealthy() bool {
	return m.healthy
}
