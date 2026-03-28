package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knownAgentID is the agent ID present in the shared test fixture.
// All tests that call handleSpawnAgent with a valid agent must use this ID.
const knownAgentID = "test-agent-stub"

// TestMain initialises a shared agents-index.json fixture before any test runs.
// This is necessary because LoadAgentsIndexCached caches the first load for the
// lifetime of the process — writing the fixture once ensures all tests in this
// binary share a consistent view.
func TestMain(m *testing.M) {
	// Build a temp index with both the known agent and any IDs needed by tests.
	// "nonexistent-xyz" is intentionally absent to exercise the unknown-agent path.
	dir, err := os.MkdirTemp("", "gofortress-mcp-standalone-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	indexDir := filepath.Join(dir, ".claude", "agents")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		panic(err)
	}
	indexPath := filepath.Join(indexDir, "agents-index.json")
	if err := os.WriteFile(indexPath, testAgentsIndex(knownAgentID), 0o644); err != nil {
		panic(err)
	}

	os.Setenv("GOGENT_AGENTS_INDEX", indexPath) //nolint:errcheck
	os.Exit(m.Run())
}

// testAgentsIndex returns a minimal but valid agents-index.json bytes slice
// containing agents with the given IDs. Version must match
// EXPECTED_AGENT_INDEX_VERSION in pkg/routing/agents.go.
func testAgentsIndex(agentIDs ...string) []byte {
	agents := ""
	tiers := ""
	for i, id := range agentIDs {
		if i > 0 {
			agents += ","
			tiers += ","
		}
		agents += `
    {
      "id": "` + id + `",
      "name": "Test Agent ` + id + `",
      "model": "sonnet",
      "thinking": false,
      "tier": 2,
      "category": "implementation",
      "path": "agents/test/` + id + `.md",
      "triggers": ["test"],
      "tools": ["Read"],
      "auto_activate": null,
      "description": "test agent"
    }`
		tiers += `"` + id + `"`
	}

	return []byte(`{
  "version": "2.6.0",
  "generated_at": "2026-03-24T00:00:00Z",
  "description": "test index",
  "agents": [` + agents + `
  ],
  "routing_rules": {
    "intent_gate": {"description": "test", "types": []},
    "scout_first_protocol": {
      "description": "test",
      "triggers": [],
      "skip_when": [],
      "primary": "haiku-scout",
      "fallback": "codebase-search",
      "output": ".claude/tmp/scout_metrics.json"
    },
    "complexity_routing": {
      "description": "test",
      "calculator": "gogent-score",
      "thresholds": {},
      "force_external_if": "tokens > 50000"
    },
    "auto_fire": {},
    "model_tiers": {
      "sonnet": [` + tiers + `]
    }
  },
  "state_management": {
    "description": "test",
    "tmp_directory": ".claude/tmp",
    "files": {},
    "cleanup": {"trigger": "session_end", "action": "delete"}
  }
}`)
}

// writeTempAgentsIndex writes an agents-index.json fixture containing the given
// agent IDs to a temp directory and sets GOGENT_AGENTS_INDEX so
// LoadAgentsIndexCached picks it up.
//
// IMPORTANT: LoadAgentsIndexCached caches the first loaded index for the
// lifetime of the process. Tests that call the handler directly must share
// a single fixture that includes all agents they reference, written before
// any handler call in the test process. Use a TestMain or call this once with
// all required agent IDs.
func writeTempAgentsIndex(t *testing.T, agentIDs ...string) {
	t.Helper()

	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(indexDir, 0o755))
	indexPath := filepath.Join(indexDir, "agents-index.json")
	require.NoError(t, os.WriteFile(indexPath, testAgentsIndex(agentIDs...), 0o644))

	t.Setenv("GOGENT_AGENTS_INDEX", indexPath)
}

// newTestSession creates an in-process MCP server and returns a connected client
// session. Uses mcpsdk.NewInMemoryTransports for fast, no-I/O testing.
func newTestSession(t *testing.T) *mcpsdk.ClientSession {
	t.Helper()

	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "gofortress-mcp-standalone-test", Version: "1.0.0"},
		nil,
	)
	RegisterAll(server)

	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcpsdk.NewClient(
		&mcpsdk.Implementation{Name: "test-client", Version: "0.0.1"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "client.Connect")

	return session
}

// extractResult decodes a CallToolResult's StructuredContent (or Content[0].Text
// fallback) into dst. Mirrors the helper pattern from server_integration_test.go.
func extractResult(t *testing.T, result *mcpsdk.CallToolResult, dst any) {
	t.Helper()

	if result.StructuredContent != nil {
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err, "marshal StructuredContent")
		require.NoError(t, json.Unmarshal(raw, dst), "unmarshal StructuredContent")
		return
	}

	require.NotEmpty(t, result.Content, "result has neither StructuredContent nor Content")
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok, "Content[0] must be *TextContent, got %T", result.Content[0])
	require.NoError(t, json.Unmarshal([]byte(tc.Text), dst), "unmarshal Content[0].Text")
}

// ---------------------------------------------------------------------------
// TestRegisterAll
// ---------------------------------------------------------------------------

// TestRegisterAll verifies that RegisterAll registers exactly 3 tools:
// spawn_agent, test_mcp_ping, and get_spawn_result.
func TestRegisterAll(t *testing.T) {
	session := newTestSession(t)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err, "ListTools")
	require.Len(t, tools.Tools, 3, "expected exactly 3 tools")

	names := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "spawn_agent")
	assert.Contains(t, names, "test_mcp_ping")
	assert.Contains(t, names, "get_spawn_result")
}

// ---------------------------------------------------------------------------
// TestHandleTestMcpPing
// ---------------------------------------------------------------------------

func TestHandleTestMcpPing_Echo(t *testing.T) {
	session := newTestSession(t)

	echo := "hello"
	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "test_mcp_ping",
		Arguments: map[string]any{"echo": echo},
	})
	require.NoError(t, err, "CallTool test_mcp_ping")
	assert.False(t, result.IsError, "result should not be an error")

	var out PingOutput
	extractResult(t, result, &out)

	assert.Equal(t, "PONG", out.Status)
	require.NotNil(t, out.Echo, "echo should be reflected")
	assert.Equal(t, echo, *out.Echo)

	// Validate RFC3339 timestamp.
	_, err = time.Parse(time.RFC3339, out.Timestamp)
	assert.NoError(t, err, "timestamp must be valid RFC3339: %s", out.Timestamp)
}

func TestHandleTestMcpPing_NoEcho(t *testing.T) {
	session := newTestSession(t)

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "test_mcp_ping",
		Arguments: map[string]any{},
	})
	require.NoError(t, err, "CallTool test_mcp_ping")
	assert.False(t, result.IsError, "result should not be an error")

	var out PingOutput
	extractResult(t, result, &out)

	assert.Equal(t, "PONG", out.Status)
	assert.Nil(t, out.Echo, "echo should be nil when not provided")

	_, err = time.Parse(time.RFC3339, out.Timestamp)
	assert.NoError(t, err, "timestamp must be valid RFC3339: %s", out.Timestamp)
}

// ---------------------------------------------------------------------------
// TestHandleSpawnAgent — validation paths
// ---------------------------------------------------------------------------

func TestHandleSpawnAgent_MissingAgent(t *testing.T) {
	ctx := context.Background()
	_, _, err := handleSpawnAgent(ctx, nil, SpawnAgentInput{
		Agent:       "",
		Description: "test",
		Prompt:      "do something",
	})
	require.Error(t, err, "expected error for missing agent")
	assert.Contains(t, err.Error(), "agent is required")
}

func TestHandleSpawnAgent_MissingDescription(t *testing.T) {
	ctx := context.Background()
	_, _, err := handleSpawnAgent(ctx, nil, SpawnAgentInput{
		Agent:       "some-agent",
		Description: "",
		Prompt:      "do something",
	})
	require.Error(t, err, "expected error for missing description")
	assert.Contains(t, err.Error(), "description is required")
}

func TestHandleSpawnAgent_MissingPrompt(t *testing.T) {
	ctx := context.Background()
	_, _, err := handleSpawnAgent(ctx, nil, SpawnAgentInput{
		Agent:       "some-agent",
		Description: "test",
		Prompt:      "",
	})
	require.Error(t, err, "expected error for missing prompt")
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestHandleSpawnAgent_UnknownAgent(t *testing.T) {
	// The TestMain fixture contains knownAgentID but not "nonexistent-xyz".
	ctx := context.Background()
	_, out, err := handleSpawnAgent(ctx, nil, SpawnAgentInput{
		Agent:       "nonexistent-xyz",
		Description: "test spawn",
		Prompt:      "do something",
	})

	// Unknown agent is a soft error (returned in output, not as Go error).
	require.NoError(t, err, "unknown agent should not return a Go error")
	assert.False(t, out.Success)
	assert.Contains(t, out.Error, "unknown agent")
	assert.Contains(t, out.Error, "nonexistent-xyz")
}

func TestHandleSpawnAgent_KnownAgent_AttemptSpawn(t *testing.T) {
	// The TestMain fixture contains knownAgentID. We use a short-lived context so
	// any real claude subprocess is killed promptly. The call must not return a Go
	// error — subprocess failures are reported as soft errors in SpawnAgentOutput.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, out, err := handleSpawnAgent(ctx, nil, SpawnAgentInput{
		Agent:       knownAgentID,
		Description: "test spawn",
		Prompt:      "do something useful",
	})

	require.NoError(t, err, "subprocess failure must be a soft error in SpawnAgentOutput, not a Go error")
	assert.Equal(t, knownAgentID, out.Agent)
	// AgentID must be a non-empty UUID.
	assert.NotEmpty(t, out.AgentID)
	// Duration must be a non-empty string (e.g. "1ms").
	assert.NotEmpty(t, out.Duration)
}
