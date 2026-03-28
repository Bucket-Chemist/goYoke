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

// TestIntegration_ToolManifest verifies that ListTools returns exactly 3 tools:
// spawn_agent, test_mcp_ping, and get_spawn_result.
func TestIntegration_ToolManifest(t *testing.T) {
	session, _ := newTestServer(t)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err, "ListTools")

	names := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		names[i] = tool.Name
	}

	assert.Len(t, names, 3, "expected exactly 3 tools")
	assert.Contains(t, names, "spawn_agent")
	assert.Contains(t, names, "test_mcp_ping")
	assert.Contains(t, names, "get_spawn_result")
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

// ---------------------------------------------------------------------------
// TestIntegration_GetSpawnResult_UnknownID
// ---------------------------------------------------------------------------

// TestIntegration_GetSpawnResult_UnknownID verifies that get_spawn_result with
// an unknown spawn_id returns an error message without a protocol-level failure.
func TestIntegration_GetSpawnResult_UnknownID(t *testing.T) {
	session, _ := newTestServer(t)

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_spawn_result",
		Arguments: map[string]any{
			"spawn_id": "nonexistent-id",
			"block":    false,
		},
	})
	require.NoError(t, err, "get_spawn_result must not return a protocol error")
	assert.False(t, result.IsError, "result must not be flagged as protocol error")

	var out GetSpawnResultOutput
	extractResultInteg(t, result, &out)

	assert.Equal(t, "nonexistent-id", out.SpawnID)
	assert.Contains(t, out.Error, "unknown spawn_id")
}

// ---------------------------------------------------------------------------
// TestIntegration_GetSpawnResult_NonBlocking
// ---------------------------------------------------------------------------

// TestIntegration_GetSpawnResult_NonBlocking verifies that a registered
// background spawn can be polled with block=false and returns the running status
// before completion, then the completed status after.
func TestIntegration_GetSpawnResult_NonBlocking(t *testing.T) {
	session, _ := newTestServer(t)

	// Manually register a background spawn in the store.
	agentID := "test-bg-nonblock"
	bgStore.Register(agentID, "go-pro")

	// Poll while running.
	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_spawn_result",
		Arguments: map[string]any{
			"spawn_id": agentID,
			"block":    false,
		},
	})
	require.NoError(t, err)

	var out GetSpawnResultOutput
	extractResultInteg(t, result, &out)
	assert.Equal(t, SpawnStatusRunning, out.Status)
	assert.Nil(t, out.Result, "result must be nil while running")

	// Complete it.
	bgStore.Complete(agentID, &SpawnAgentOutput{
		AgentID: agentID,
		Agent:   "go-pro",
		Success: true,
		Output:  "background result",
		Cost:    0.03,
	})

	// Poll again — should be completed now.
	result2, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_spawn_result",
		Arguments: map[string]any{
			"spawn_id": agentID,
			"block":    false,
		},
	})
	require.NoError(t, err)

	var out2 GetSpawnResultOutput
	extractResultInteg(t, result2, &out2)
	assert.Equal(t, SpawnStatusCompleted, out2.Status)
	require.NotNil(t, out2.Result, "result must be populated after completion")
	assert.Equal(t, "background result", out2.Result.Output)
	assert.Equal(t, 0.03, out2.Result.Cost)
}

// ---------------------------------------------------------------------------
// TestIntegration_GetSpawnResult_Blocking
// ---------------------------------------------------------------------------

// TestIntegration_GetSpawnResult_Blocking verifies that block=true waits for
// the spawn to complete and returns the result.
func TestIntegration_GetSpawnResult_Blocking(t *testing.T) {
	session, _ := newTestServer(t)

	agentID := "test-bg-blocking"
	bgStore.Register(agentID, "go-pro")

	// Complete after a delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		bgStore.Complete(agentID, &SpawnAgentOutput{
			AgentID: agentID,
			Agent:   "go-pro",
			Success: true,
			Output:  "waited for this",
		})
	}()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_spawn_result",
		Arguments: map[string]any{
			"spawn_id": agentID,
			"block":    true,
			"timeout":  5000,
		},
	})
	require.NoError(t, err)

	var out GetSpawnResultOutput
	extractResultInteg(t, result, &out)
	assert.Equal(t, SpawnStatusCompleted, out.Status)
	require.NotNil(t, out.Result)
	assert.Equal(t, "waited for this", out.Result.Output)
}

// ---------------------------------------------------------------------------
// TestIntegration_GetSpawnResult_BlockTimeout
// ---------------------------------------------------------------------------

// TestIntegration_GetSpawnResult_BlockTimeout verifies that block=true with a
// short timeout returns an error when the spawn doesn't complete in time.
func TestIntegration_GetSpawnResult_BlockTimeout(t *testing.T) {
	session, _ := newTestServer(t)

	agentID := "test-bg-timeout"
	bgStore.Register(agentID, "go-pro")
	// Never complete — should hit timeout.

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "get_spawn_result",
		Arguments: map[string]any{
			"spawn_id": agentID,
			"block":    true,
			"timeout":  100, // 100ms
		},
	})
	require.NoError(t, err, "must not be a protocol error")

	var out GetSpawnResultOutput
	extractResultInteg(t, result, &out)
	assert.Equal(t, SpawnStatusRunning, out.Status)
	assert.Contains(t, out.Error, "timeout")
}
