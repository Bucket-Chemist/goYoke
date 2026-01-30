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
