//go:build integration

package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer creates an in-process MCP server with all tools registered and
// returns a connected client session. Uses mcpsdk.NewInMemoryTransports so no
// real I/O or binary pre-build is required.
//
// The caller does not need to call cancel — t.Cleanup handles teardown.
func newTestServer(t *testing.T) (*mcpsdk.ClientSession, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "gofortress-mcp-standalone", Version: "1.0.0"},
		nil,
	)
	RegisterAll(server)

	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcpsdk.NewClient(
		&mcpsdk.Implementation{Name: "test-client", Version: "0.0.1"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "client.Connect")

	t.Cleanup(func() { cancel() })

	return session, cancel
}

// extractResultInteg decodes a CallToolResult's StructuredContent (or falls
// back to Content[0].Text) into dst. Mirrors extractStructuredContent in
// internal/tui/mcp/server_integration_test.go (lines 178-192).
//
// Named extractResultInteg to avoid a redeclaration conflict with the
// extractResult helper already declared in tools_test.go (same package).
func extractResultInteg(t *testing.T, result *mcpsdk.CallToolResult, dst any) {
	t.Helper()

	if result.StructuredContent != nil {
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err, "marshal StructuredContent")
		require.NoError(t, json.Unmarshal(raw, dst), "unmarshal StructuredContent into dst")
		return
	}

	require.NotEmpty(t, result.Content, "CallToolResult: Content and StructuredContent are both absent")
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok, "Content[0] must be *TextContent, got %T", result.Content[0])
	require.NoError(t, json.Unmarshal([]byte(tc.Text), dst), "unmarshal Content[0].Text into dst")
}

// ---------------------------------------------------------------------------
// TestIntegration_ToolManifest
// ---------------------------------------------------------------------------

// TestIntegration_ToolManifest verifies that ListTools returns exactly 2 tools:
// spawn_agent and test_mcp_ping.
func TestIntegration_ToolManifest(t *testing.T) {
	session, _ := newTestServer(t)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err, "ListTools")

	names := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		names[i] = tool.Name
	}

	assert.Len(t, names, 2, "expected exactly 2 tools")
	assert.Contains(t, names, "spawn_agent")
	assert.Contains(t, names, "test_mcp_ping")
}

// ---------------------------------------------------------------------------
// TestIntegration_Ping
// ---------------------------------------------------------------------------

// TestIntegration_Ping verifies that test_mcp_ping returns PONG, reflects the
// echo field, and produces a valid RFC3339 timestamp.
func TestIntegration_Ping(t *testing.T) {
	session, _ := newTestServer(t)

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "test_mcp_ping",
		Arguments: map[string]any{"echo": "integration"},
	})
	require.NoError(t, err, "CallTool test_mcp_ping")
	assert.False(t, result.IsError, "result must not be a protocol error")

	var out PingOutput
	extractResultInteg(t, result, &out)

	assert.Equal(t, "PONG", out.Status)
	require.NotNil(t, out.Echo, "echo field must be reflected")
	assert.Equal(t, "integration", *out.Echo)

	_, parseErr := time.Parse(time.RFC3339, out.Timestamp)
	assert.NoError(t, parseErr, "timestamp must be valid RFC3339: %s", out.Timestamp)
}

// ---------------------------------------------------------------------------
// TestIntegration_SpawnAgent_UnknownAgent
// ---------------------------------------------------------------------------

// TestIntegration_SpawnAgent_UnknownAgent verifies that calling spawn_agent
// with a nonexistent agent ID returns SpawnAgentOutput{Success:false} without
// panicking or producing a protocol-level error.
//
// The TestMain fixture (from tools_test.go) is already loaded by the time this
// test runs and contains knownAgentID but not "nonexistent-xyz". We rely on
// that shared fixture so we do not disturb the process-wide
// LoadAgentsIndexCached state.
func TestIntegration_SpawnAgent_UnknownAgent(t *testing.T) {
	session, _ := newTestServer(t)

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "spawn_agent",
		Arguments: map[string]any{
			"agent":       "nonexistent-xyz",
			"description": "test unknown agent",
			"prompt":      "do something",
		},
	})
	require.NoError(t, err, "spawn_agent must not return a protocol error for unknown agent")
	assert.False(t, result.IsError, "result must not be flagged as a protocol error")

	var out SpawnAgentOutput
	extractResultInteg(t, result, &out)

	assert.False(t, out.Success, "success must be false for unknown agent")
	assert.Contains(t, out.Error, "unknown agent", "error must mention 'unknown agent'")
}
