// Package mcp_test contains integration tests for the mcp package.
//
// It is declared as an external test package (mcp_test rather than mcp) to
// avoid the import cycle:
//
//	mcp (package) → bridge (package) → mcp (package)
//
// Using an external test package breaks the cycle because Go's build system
// treats external test packages as separate compilation units that may import
// both the package under test and its dependents.
//
// TUI-038: MCP server integration test.
//
// Wires real components end-to-end:
//   - Real IPCBridge (TUI-side UDS listener)
//   - Real MCP server (mcpsdk.Server) with all 8 tools registered
//   - Real UDSClient connecting to the bridge
//   - In-memory MCP transport (no stdin/stdout pipes required)
//   - integrationMockSender captures Bubbletea messages for assertion
//
// Test architecture (per test):
//
//	MCP Client ──in-memory──► MCP Server ──UDSClient──► IPCBridge ──sender──► captured msgs
//	                                                        │
//	                                            ResolveModal (simulated user)
//
// Run with:
//
//	go test -v -run TestIntegration ./internal/tui/mcp/
//	go test -race -run TestIntegration ./internal/tui/mcp/
package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/bridge"
	tuimcp "github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Integration test helpers
// ---------------------------------------------------------------------------

const integrationTimeout = 10 * time.Second

// integrationMockSender captures every message sent via Send().
// It satisfies the unexported messageSender interface in bridge by implementing
// the tea.Program-compatible Send method.
type integrationMockSender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (m *integrationMockSender) Send(msg tea.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
}

// waitForIntegrationMsg blocks until at least one message of type T is
// received, or the given timeout expires.  Returns the first matching message
// and true, or the zero value and false on timeout.
func waitForIntegrationMsg[T any](ms *integrationMockSender, timeout time.Duration) (T, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ms.mu.Lock()
		for _, msg := range ms.msgs {
			if v, ok := msg.(T); ok {
				ms.mu.Unlock()
				return v, true
			}
		}
		ms.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	var zero T
	return zero, false
}

// integrationHarness wires together a real IPCBridge, real UDSClient, real
// MCP server and an in-memory MCP client session.  GOFORTRESS_SOCKET is set
// to the bridge's socket path for the duration of the test.
type integrationHarness struct {
	bridge  *bridge.IPCBridge
	sender  *integrationMockSender
	session *mcpsdk.ClientSession
	ctx     context.Context
	cancel  context.CancelFunc
}

// newIntegrationHarness creates and connects all components.  Cleanup stops
// the server context and shuts down the bridge.
func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)

	// 1. Start a real IPCBridge.  XDG_RUNTIME_DIR is redirected to a temp
	//    directory so the socket path is isolated per test.
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	ms := &integrationMockSender{}
	b, err := bridge.NewIPCBridge(ms)
	require.NoError(t, err, "NewIPCBridge")
	b.Start()

	// 2. Point GOFORTRESS_SOCKET at the bridge so UDSClient can connect.
	t.Setenv("GOFORTRESS_SOCKET", b.SocketPath())

	// 3. Create the real UDSClient (reads GOFORTRESS_SOCKET from env).
	uds := tuimcp.NewUDSClient()

	// 4. Build the real MCP server with all 8 tools registered.
	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "gofortress-mcp-test", Version: "1.0.0"},
		nil,
	)
	tuimcp.RegisterAll(server, uds)

	// 5. Wire the server and client via in-memory transports (no I/O required).
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	mcpClient := mcpsdk.NewClient(
		&mcpsdk.Implementation{Name: "test-client", Version: "0.0.1"},
		nil,
	)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "mcpClient.Connect")

	h := &integrationHarness{
		bridge:  b,
		sender:  ms,
		session: session,
		ctx:     ctx,
		cancel:  cancel,
	}

	t.Cleanup(func() {
		cancel()
		b.Shutdown()
	})

	return h
}

// callTool calls a named MCP tool and returns the result.
func (h *integrationHarness) callTool(t *testing.T, toolName string, args any) *mcpsdk.CallToolResult {
	t.Helper()
	result, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	require.NoError(t, err, "CallTool(%s)", toolName)
	return result
}

// extractStructuredContent decodes the StructuredContent field of a
// CallToolResult into dst.  Falls back to Content[0].Text when
// StructuredContent is absent (e.g. when the tool returned an empty output
// struct that serialises to "null").
func extractStructuredContent(t *testing.T, result *mcpsdk.CallToolResult, dst any) {
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

// integrationAgentsIndex returns a minimal but valid agents-index.json bytes
// slice containing a single agent with the given ID.  Duplicated from the
// internal test helper in tools_test.go because mcp_test cannot access that
// unexported symbol.
func integrationAgentsIndex(agentID string) []byte {
	return []byte(`{
  "version": "2.7.0",
  "generated_at": "2026-03-23T00:00:00Z",
  "description": "integration test index",
  "agents": [
    {
      "id": "` + agentID + `",
      "name": "Test Agent",
      "model": "sonnet",
      "thinking": false,
      "tier": 2,
      "category": "implementation",
      "path": "agents/test/test.md",
      "triggers": ["test trigger"],
      "tools": ["Read"],
      "auto_activate": null,
      "description": "test agent"
    }
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
      "sonnet": ["` + agentID + `"]
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

// ---------------------------------------------------------------------------
// (a) TestIntegration_McpPing — no UDS required
// ---------------------------------------------------------------------------

// TestIntegration_McpPing verifies that test_mcp_ping returns PONG with an
// RFC3339 timestamp and reflects the optional echo field.
func TestIntegration_McpPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	echoMsg := "hello-integration"
	result := h.callTool(t, "test_mcp_ping", map[string]any{
		"echo": echoMsg,
	})
	assert.False(t, result.IsError, "test_mcp_ping must not set IsError")

	var out tuimcp.PingOutput
	extractStructuredContent(t, result, &out)

	assert.Equal(t, "PONG", out.Status)
	require.NotNil(t, out.Echo, "echo must be present when provided")
	assert.Equal(t, echoMsg, *out.Echo)
	_, parseErr := time.Parse(time.RFC3339, out.Timestamp)
	assert.NoError(t, parseErr, "timestamp must be valid RFC3339")
}

// TestIntegration_McpPing_NoEcho verifies test_mcp_ping works when the echo
// field is omitted (nil in the response).
func TestIntegration_McpPing_NoEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	result := h.callTool(t, "test_mcp_ping", map[string]any{})
	assert.False(t, result.IsError)

	var out tuimcp.PingOutput
	extractStructuredContent(t, result, &out)
	assert.Equal(t, "PONG", out.Status)
	assert.Nil(t, out.Echo)
}

// ---------------------------------------------------------------------------
// (b) TestIntegration_AskUser — full modal round-trip via UDS
// ---------------------------------------------------------------------------

// TestIntegration_AskUser verifies the full ask_user modal round-trip:
//  1. MCP client calls ask_user → handler blocks on UDS send
//  2. UDSClient forwards modal_request to IPCBridge via UDS
//  3. Bridge delivers BridgeModalRequestMsg to the TUI (integrationMockSender)
//  4. Test simulates user selecting "Yes" via bridge.ResolveModal
//  5. Bridge sends modal_response back over UDS → handler unblocks
//  6. MCP caller receives AskUserOutput{Answer: "Yes"}
func TestIntegration_AskUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	type toolResult struct {
		result *mcpsdk.CallToolResult
		err    error
	}
	resultCh := make(chan toolResult, 1)

	// Tool call blocks until the modal is resolved — run in goroutine.
	go func() {
		r, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
			Name: "ask_user",
			Arguments: map[string]any{
				"message": "Should we proceed?",
				"options": []string{"Yes", "No", "Maybe"},
			},
		})
		resultCh <- toolResult{r, err}
	}()

	// Wait for the bridge to deliver the BridgeModalRequestMsg.
	bridgeMsg, ok := waitForIntegrationMsg[model.BridgeModalRequestMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Equal(t, "Should we proceed?", bridgeMsg.Message)
	assert.Equal(t, []string{"Yes", "No", "Maybe"}, bridgeMsg.Options)
	assert.NotEmpty(t, bridgeMsg.RequestID, "RequestID must be populated")

	// Simulate the user selecting "Yes".
	h.bridge.ResolveModalSimple(bridgeMsg.RequestID, "Yes")

	// Collect and verify the tool result.
	select {
	case tr := <-resultCh:
		require.NoError(t, tr.err, "ask_user CallTool must not return a protocol error")
		assert.False(t, tr.result.IsError, "ask_user result must not set IsError")

		var out tuimcp.AskUserOutput
		extractStructuredContent(t, tr.result, &out)
		assert.Equal(t, "Yes", out.Answer)

	case <-time.After(integrationTimeout):
		t.Fatal("timed out waiting for ask_user tool result")
	}
}

// ---------------------------------------------------------------------------
// (c) TestIntegration_SpawnAgent — agent registration via UDS
// ---------------------------------------------------------------------------

// TestIntegration_SpawnAgent verifies that a valid spawn_agent call sends both
// agent_register and agent_update notifications to the TUI via UDS.
func TestIntegration_SpawnAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("GOFORTRESS_SPAWN_INTEGRATION") == "" {
		t.Skip("set GOFORTRESS_SPAWN_INTEGRATION=1 to run live subprocess spawn tests")
	}

	// Set up a minimal agents-index.json so LoadAgentIndex succeeds.
	agentDir := filepath.Join(t.TempDir(), ".claude", "agents")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(agentDir, "agents-index.json"),
		integrationAgentsIndex("go-pro"),
		0o644,
	))
	// GOGENT_PROJECT_DIR must point to the directory that contains .claude/.
	t.Setenv("GOGENT_AGENTS_INDEX", "") // clear any env-inherited override
	t.Setenv("GOGENT_PROJECT_DIR", filepath.Dir(filepath.Dir(agentDir)))

	h := newIntegrationHarness(t)

	result := h.callTool(t, "spawn_agent", map[string]any{
		"agent":       "go-pro",
		"description": "integration test spawn",
		"prompt":      "do something useful",
	})
	assert.False(t, result.IsError, "spawn_agent must not set IsError for a valid agent")

	var out tuimcp.SpawnAgentOutput
	extractStructuredContent(t, result, &out)
	assert.True(t, out.Success, "spawn_agent must report success")
	assert.Equal(t, "go-pro", out.Agent)
	assert.NotEmpty(t, out.AgentID, "AgentID must be a non-empty UUID")

	// The agent_register notification must arrive at the TUI via UDS.
	regMsg, ok := waitForIntegrationMsg[model.AgentRegisteredMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for AgentRegisteredMsg from spawn_agent")
	assert.Equal(t, "go-pro", regMsg.AgentType, "AgentType must match the requested agent")
	assert.Equal(t, out.AgentID, regMsg.AgentID, "AgentID must match the spawn_agent response")

	// The agent_update notification (status="stub") must also arrive.
	updMsg, ok := waitForIntegrationMsg[model.AgentUpdatedMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for AgentUpdatedMsg from spawn_agent")
	assert.Equal(t, out.AgentID, updMsg.AgentID, "AgentID in update must match spawn response")
}

// ---------------------------------------------------------------------------
// (d) TestIntegration_ToolManifest — all 8 tools present
// ---------------------------------------------------------------------------

// TestIntegration_ToolManifest verifies that the MCP server exposes exactly
// the 8 expected tools.
func TestIntegration_ToolManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	tools, err := h.session.ListTools(h.ctx, nil)
	require.NoError(t, err, "ListTools must not error")

	names := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		names[i] = tool.Name
	}

	expected := []string{
		"test_mcp_ping",
		"ask_user",
		"confirm_action",
		"request_input",
		"select_option",
		"spawn_agent",
		"get_agent_result",
		"team_run",
	}

	assert.Len(t, names, len(expected), "server must expose exactly %d tools", len(expected))
	for _, name := range expected {
		assert.Contains(t, names, name, "tool %q must be in the server manifest", name)
	}
}

// ---------------------------------------------------------------------------
// Additional integration scenarios (modal tools)
// ---------------------------------------------------------------------------

// TestIntegration_ConfirmAction_Allow verifies confirm_action returns
// confirmed=true when the user selects "Allow".
func TestIntegration_ConfirmAction_Allow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	resultCh := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		r, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
			Name: "confirm_action",
			Arguments: map[string]any{
				"action": "delete all temporary files",
			},
		})
		if err == nil {
			resultCh <- r
		}
	}()

	bridgeMsg, ok := waitForIntegrationMsg[model.BridgeModalRequestMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Contains(t, bridgeMsg.Options, "Allow")
	assert.Contains(t, bridgeMsg.Options, "Deny")

	h.bridge.ResolveModalSimple(bridgeMsg.RequestID, "Allow")

	select {
	case result := <-resultCh:
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		var out tuimcp.ConfirmActionOutput
		extractStructuredContent(t, result, &out)
		assert.True(t, out.Confirmed)
		assert.False(t, out.Cancelled)

	case <-time.After(integrationTimeout):
		t.Fatal("timed out waiting for confirm_action tool result")
	}
}

// TestIntegration_ConfirmAction_Deny verifies confirm_action returns
// confirmed=false when the user selects "Deny", and that a destructive action
// is prefixed with "[DESTRUCTIVE]".
func TestIntegration_ConfirmAction_Deny(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	resultCh := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		r, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
			Name: "confirm_action",
			Arguments: map[string]any{
				"action":      "rm -rf /",
				"destructive": true,
			},
		})
		if err == nil {
			resultCh <- r
		}
	}()

	bridgeMsg, ok := waitForIntegrationMsg[model.BridgeModalRequestMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Contains(t, bridgeMsg.Message, "[DESTRUCTIVE]",
		"destructive action must be prefixed with [DESTRUCTIVE]")

	h.bridge.ResolveModalSimple(bridgeMsg.RequestID, "Deny")

	select {
	case result := <-resultCh:
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		var out tuimcp.ConfirmActionOutput
		extractStructuredContent(t, result, &out)
		assert.False(t, out.Confirmed)
		assert.True(t, out.Cancelled)

	case <-time.After(integrationTimeout):
		t.Fatal("timed out waiting for confirm_action tool result")
	}
}

// TestIntegration_RequestInput verifies the request_input tool delivers the
// user-typed value back through the full UDS round-trip.
func TestIntegration_RequestInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	resultCh := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		r, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
			Name: "request_input",
			Arguments: map[string]any{
				"prompt":      "Enter your name",
				"placeholder": "e.g. Alice",
			},
		})
		if err == nil {
			resultCh <- r
		}
	}()

	bridgeMsg, ok := waitForIntegrationMsg[model.BridgeModalRequestMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Contains(t, bridgeMsg.Message, "Enter your name")
	assert.Contains(t, bridgeMsg.Message, "e.g. Alice",
		"placeholder must be appended to the message")

	h.bridge.ResolveModalSimple(bridgeMsg.RequestID, "Bob")

	select {
	case result := <-resultCh:
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		var out tuimcp.RequestInputOutput
		extractStructuredContent(t, result, &out)
		assert.Equal(t, "Bob", out.Value)

	case <-time.After(integrationTimeout):
		t.Fatal("timed out waiting for request_input tool result")
	}
}

// TestIntegration_SelectOption verifies that select_option maps the user's
// chosen label back to its corresponding value and index.
func TestIntegration_SelectOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)

	options := []map[string]any{
		{"label": "Small", "value": "sm"},
		{"label": "Medium", "value": "md"},
		{"label": "Large", "value": "lg"},
	}

	resultCh := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		r, err := h.session.CallTool(h.ctx, &mcpsdk.CallToolParams{
			Name: "select_option",
			Arguments: map[string]any{
				"message": "Choose size",
				"options": options,
			},
		})
		if err == nil {
			resultCh <- r
		}
	}()

	bridgeMsg, ok := waitForIntegrationMsg[model.BridgeModalRequestMsg](h.sender, integrationTimeout)
	require.True(t, ok, "timed out waiting for BridgeModalRequestMsg")
	assert.Equal(t, "Choose size", bridgeMsg.Message)
	// The bridge receives only label strings (not full option objects).
	assert.Equal(t, []string{"Small", "Medium", "Large"}, bridgeMsg.Options)

	// User picks "Medium" (label).
	h.bridge.ResolveModalSimple(bridgeMsg.RequestID, "Medium")

	select {
	case result := <-resultCh:
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		var out tuimcp.SelectOptionOutput
		extractStructuredContent(t, result, &out)
		assert.Equal(t, "md", out.Selected, "Selected must be the value, not the label")
		assert.Equal(t, 1, out.Index, "Medium is at index 1")

	case <-time.After(integrationTimeout):
		t.Fatal("timed out waiting for select_option tool result")
	}
}

// TestIntegration_TeamRun_ExistingDir verifies that team_run returns a
// structured (non-error) response for an existing directory, regardless of
// whether the gogent-team-run binary is available.
func TestIntegration_TeamRun_ExistingDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := newIntegrationHarness(t)
	teamDir := t.TempDir()

	result := h.callTool(t, "team_run", map[string]any{
		"team_dir": teamDir,
	})

	// team_run returns a structured failure (not IsError) when the binary is
	// absent; either success or structured failure is acceptable here.
	assert.False(t, result.IsError, "team_run must not set IsError (protocol-level error)")
	require.NotEmpty(t, result.Content, "team_run must return non-empty Content")

	var out tuimcp.TeamRunOutput
	extractStructuredContent(t, result, &out)
	assert.Equal(t, teamDir, out.TeamDir, "team_dir must be echoed back")
}
