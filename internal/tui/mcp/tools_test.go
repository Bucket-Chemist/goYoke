package mcp

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// -----------------------------------------------------------------------------
// Protocol type serialisation
// -----------------------------------------------------------------------------

func TestIPCRequestRoundtrip(t *testing.T) {
	payload := ModalRequestPayload{
		Message: "Allow Write to /tmp/foo?",
		Options: []string{"Allow", "Deny"},
		Default: "Deny",
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	req := IPCRequest{
		Type:    TypeModalRequest,
		ID:      "req-1",
		Payload: raw,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var got IPCRequest
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, req.Type, got.Type)
	assert.Equal(t, req.ID, got.ID)

	var gotPayload ModalRequestPayload
	require.NoError(t, json.Unmarshal(got.Payload, &gotPayload))
	assert.Equal(t, payload.Message, gotPayload.Message)
	assert.Equal(t, payload.Options, gotPayload.Options)
	assert.Equal(t, payload.Default, gotPayload.Default)
}

func TestIPCResponseRoundtrip(t *testing.T) {
	resp := IPCResponse{
		Type:    TypeModalResponse,
		ID:      "req-1",
		Payload: json.RawMessage(`{"value":"Allow"}`),
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var got IPCResponse
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, resp.Type, got.Type)
	assert.Equal(t, resp.ID, got.ID)

	var mp ModalResponsePayload
	require.NoError(t, json.Unmarshal(got.Payload, &mp))
	assert.Equal(t, "Allow", mp.Value)
}

func TestAllPayloadTypesSerialise(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{"AgentRegisterPayload", AgentRegisterPayload{AgentID: "a1", AgentType: "go-pro", ParentID: "p0"}},
		{"AgentUpdatePayload", AgentUpdatePayload{AgentID: "a1", Status: "running"}},
		{"AgentActivityPayload", AgentActivityPayload{AgentID: "a1", Tool: "Read"}},
		{"ToastPayload", ToastPayload{Message: "hello", Level: "info"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.payload)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}

// -----------------------------------------------------------------------------
// test_mcp_ping handler
// -----------------------------------------------------------------------------

func TestHandleTestMcpPing_NoEcho(t *testing.T) {
	_, out, err := handleTestMcpPing(context.Background(), nil, PingInput{})
	require.NoError(t, err)
	assert.Equal(t, "PONG", out.Status)
	assert.Nil(t, out.Echo)
	// Timestamp must be parseable as RFC3339.
	_, parseErr := time.Parse(time.RFC3339, out.Timestamp)
	assert.NoError(t, parseErr, "timestamp should be RFC3339")
}

func TestHandleTestMcpPing_WithEcho(t *testing.T) {
	msg := "hello world"
	_, out, err := handleTestMcpPing(context.Background(), nil, PingInput{Echo: &msg})
	require.NoError(t, err)
	assert.Equal(t, "PONG", out.Status)
	require.NotNil(t, out.Echo)
	assert.Equal(t, msg, *out.Echo)
}

// -----------------------------------------------------------------------------
// mock UDS listener helper
// -----------------------------------------------------------------------------

// mockUDS sets up a real Unix domain socket listener in a temp directory,
// sets GOFORTRESS_SOCKET, and returns a UDSClient wired to it.  The cleanup
// function must be deferred by the caller.
//
// The responder is called in a goroutine for each accepted connection and must
// read exactly one IPCRequest and write exactly one IPCResponse.
func mockUDS(t *testing.T, responder func(req IPCRequest) IPCResponse) (*UDSClient, func()) {
	t.Helper()
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "test.sock")

	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err, "mock UDS listener")

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		defer conn.Close()

		dec := json.NewDecoder(conn)
		enc := json.NewEncoder(conn)

		var req IPCRequest
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := responder(req)
		_ = enc.Encode(resp)
	}()

	t.Setenv("GOFORTRESS_SOCKET", sockPath)
	client := NewUDSClient()

	cleanup := func() {
		_ = ln.Close()
	}
	return client, cleanup
}

// -----------------------------------------------------------------------------
// ask_user handler
// -----------------------------------------------------------------------------

func TestHandleAskUser_Success(t *testing.T) {
	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		assert.Equal(t, TypeModalRequest, req.Type)
		var p ModalRequestPayload
		require.NoError(t, json.Unmarshal(req.Payload, &p))
		assert.Equal(t, "What colour?", p.Message)

		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Blue"})
		return IPCResponse{
			Type:    TypeModalResponse,
			ID:      req.ID,
			Payload: respPayload,
		}
	})
	defer cleanup()

	_, out, err := handleAskUser(context.Background(), nil, AskUserInput{Message: "What colour?"}, uds)
	require.NoError(t, err)
	assert.Equal(t, "Blue", out.Answer)
}

func TestHandleAskUser_MissingMessage(t *testing.T) {
	// Should fail validation before any UDS call.
	uds := &UDSClient{}
	_, _, err := handleAskUser(context.Background(), nil, AskUserInput{}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestHandleAskUser_NoTUI(t *testing.T) {
	t.Setenv("GOFORTRESS_SOCKET", "")
	uds := NewUDSClient()
	_, _, err := handleAskUser(context.Background(), nil, AskUserInput{Message: "hello"}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TUI not connected")
}

// -----------------------------------------------------------------------------
// confirm_action handler
// -----------------------------------------------------------------------------

func TestHandleConfirmAction_Allow(t *testing.T) {
	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		var p ModalRequestPayload
		require.NoError(t, json.Unmarshal(req.Payload, &p))
		assert.Equal(t, []string{"Allow", "Deny"}, p.Options)

		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Allow"})
		return IPCResponse{Type: TypeModalResponse, ID: req.ID, Payload: respPayload}
	})
	defer cleanup()

	_, out, err := handleConfirmAction(context.Background(), nil,
		ConfirmActionInput{Action: "delete /tmp/foo"}, uds)
	require.NoError(t, err)
	assert.True(t, out.Confirmed)
	assert.False(t, out.Cancelled)
}

func TestHandleConfirmAction_Deny(t *testing.T) {
	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Deny"})
		return IPCResponse{Type: TypeModalResponse, ID: req.ID, Payload: respPayload}
	})
	defer cleanup()

	_, out, err := handleConfirmAction(context.Background(), nil,
		ConfirmActionInput{Action: "format disk"}, uds)
	require.NoError(t, err)
	assert.False(t, out.Confirmed)
	assert.True(t, out.Cancelled)
}

func TestHandleConfirmAction_Destructive_PrefixAdded(t *testing.T) {
	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		var p ModalRequestPayload
		require.NoError(t, json.Unmarshal(req.Payload, &p))
		assert.True(t, strings.HasPrefix(p.Message, "[DESTRUCTIVE]"),
			"destructive action should have [DESTRUCTIVE] prefix")

		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Allow"})
		return IPCResponse{Type: TypeModalResponse, ID: req.ID, Payload: respPayload}
	})
	defer cleanup()

	_, out, err := handleConfirmAction(context.Background(), nil,
		ConfirmActionInput{Action: "rm -rf /", Destructive: true}, uds)
	require.NoError(t, err)
	assert.True(t, out.Confirmed)
}

func TestHandleConfirmAction_MissingAction(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleConfirmAction(context.Background(), nil, ConfirmActionInput{}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action is required")
}

// -----------------------------------------------------------------------------
// request_input handler
// -----------------------------------------------------------------------------

func TestHandleRequestInput_Success(t *testing.T) {
	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		var p ModalRequestPayload
		require.NoError(t, json.Unmarshal(req.Payload, &p))
		assert.Contains(t, p.Message, "Enter your name")

		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Alice"})
		return IPCResponse{Type: TypeModalResponse, ID: req.ID, Payload: respPayload}
	})
	defer cleanup()

	_, out, err := handleRequestInput(context.Background(), nil,
		RequestInputInput{Prompt: "Enter your name", Placeholder: "e.g. Bob"}, uds)
	require.NoError(t, err)
	assert.Equal(t, "Alice", out.Value)
}

func TestHandleRequestInput_MissingPrompt(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleRequestInput(context.Background(), nil, RequestInputInput{}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

// -----------------------------------------------------------------------------
// select_option handler
// -----------------------------------------------------------------------------

func TestHandleSelectOption_Success(t *testing.T) {
	options := []SelectOptionEntry{
		{Label: "Small", Value: "sm"},
		{Label: "Medium", Value: "md"},
		{Label: "Large", Value: "lg"},
	}

	uds, cleanup := mockUDS(t, func(req IPCRequest) IPCResponse {
		var p ModalRequestPayload
		require.NoError(t, json.Unmarshal(req.Payload, &p))
		assert.Equal(t, []string{"Small", "Medium", "Large"}, p.Options)

		respPayload, _ := json.Marshal(ModalResponsePayload{Value: "Medium"})
		return IPCResponse{Type: TypeModalResponse, ID: req.ID, Payload: respPayload}
	})
	defer cleanup()

	_, out, err := handleSelectOption(context.Background(), nil,
		SelectOptionInput{Message: "Choose size", Options: options}, uds)
	require.NoError(t, err)
	assert.Equal(t, "md", out.Selected)
	assert.Equal(t, 1, out.Index)
}

func TestHandleSelectOption_MissingMessage(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSelectOption(context.Background(), nil,
		SelectOptionInput{Options: []SelectOptionEntry{{Label: "A", Value: "a"}}}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestHandleSelectOption_EmptyOptions(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSelectOption(context.Background(), nil,
		SelectOptionInput{Message: "choose"}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "options list is required")
}

// -----------------------------------------------------------------------------
// spawn_agent handler
// -----------------------------------------------------------------------------

func TestHandleSpawnAgent_MissingAgent(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Description: "d", Prompt: "p"}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent is required")
}

func TestHandleSpawnAgent_MissingDescription(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Agent: "go-pro", Prompt: "p"}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestHandleSpawnAgent_MissingPrompt(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Agent: "go-pro", Description: "d"}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestHandleSpawnAgent_UnknownAgent(t *testing.T) {
	// Provide a minimal agents-index.json so LoadAgentIndex succeeds.
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(agentDir, "agents-index.json"),
		minimalAgentsIndex("dummy-agent"),
		0o644,
	))
	t.Setenv("GOGENT_PROJECT_DIR", dir)

	uds := &UDSClient{} // no TUI needed — stub returns before UDS call
	_, out, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Agent: "totally-unknown", Description: "d", Prompt: "p"}, uds)
	require.NoError(t, err) // structured failure, not an error
	assert.False(t, out.Success)
	assert.Contains(t, out.Error, "unknown agent")
}

// -----------------------------------------------------------------------------
// team_run handler
// -----------------------------------------------------------------------------

func TestHandleTeamRun_MissingTeamDir(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleTeamRun(context.Background(), nil, TeamRunInput{}, uds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team_dir is required")
}

func TestHandleTeamRun_NonexistentDir(t *testing.T) {
	uds := &UDSClient{}
	_, out, err := handleTeamRun(context.Background(), nil,
		TeamRunInput{TeamDir: "/nonexistent/path/to/team"}, uds)
	require.NoError(t, err) // structured failure, not an error
	assert.False(t, out.Success)
	// The handler checks for config.json; the directory does not exist so the
	// error message reports that config.json was not found.
	assert.Contains(t, out.Result, "config.json not found")
}

func TestHandleTeamRun_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	uds := &UDSClient{}

	// Write a minimal config.json so the handler progresses past the
	// config-presence check to the binary-lookup step.
	configData := []byte(`{"background_pid": 0}`)
	require.NoError(t, os.WriteFile(dir+"/config.json", configData, 0o644))

	_, out, err := handleTeamRun(context.Background(), nil,
		TeamRunInput{TeamDir: dir}, uds)
	require.NoError(t, err)
	assert.Equal(t, dir, out.TeamDir)
	// Binary presence is environment-dependent; both outcomes are valid.
	assert.NotEmpty(t, out.Result)
}

// -----------------------------------------------------------------------------
// UDS client — no socket env
// -----------------------------------------------------------------------------

func TestUDSClient_NoSocket(t *testing.T) {
	t.Setenv("GOFORTRESS_SOCKET", "")
	c := NewUDSClient()
	assert.Empty(t, c.sockEnv)
	err := c.Connect()
	assert.ErrorIs(t, err, ErrTUINotConnected)
}

// -----------------------------------------------------------------------------
// Tool registration smoke test
// -----------------------------------------------------------------------------

func TestRegisterAll_ToolsDiscoverable(t *testing.T) {
	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "test", Version: "0.0.1"},
		nil,
	)

	uds := &UDSClient{} // no TUI needed for registration
	RegisterAll(server, uds)

	ct, cs := mcpsdk.NewInMemoryTransports()
	mcpClient := mcpsdk.NewClient(
		&mcpsdk.Implementation{Name: "test-client", Version: "0.0.1"},
		nil,
	)

	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, cs)
	}()

	session, err := mcpClient.Connect(ctx, ct, nil)
	require.NoError(t, err)

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

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
		"team_run",
	}
	for _, name := range expected {
		assert.Contains(t, names, name, "tool %s should be registered", name)
	}
}

// -----------------------------------------------------------------------------
// buildSpawnArgs
// -----------------------------------------------------------------------------

func TestBuildSpawnArgs_Defaults(t *testing.T) {
	agent := &routing.Agent{
		ID:    "go-pro",
		Name:  "GO Pro",
		Model: "sonnet",
	}
	input := SpawnAgentInput{
		Agent:       "go-pro",
		Description: "test",
		Prompt:      "do stuff",
	}
	args := buildSpawnArgs(agent, input)
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "--model")
	// --timeout is NOT passed as a CLI flag; managed by spawner's time.AfterFunc.
	assert.NotContains(t, args, "--timeout")
}

func TestBuildSpawnArgs_ModelOverride(t *testing.T) {
	agent := &routing.Agent{
		ID:    "go-pro",
		Name:  "GO Pro",
		Model: "sonnet",
	}
	input := SpawnAgentInput{
		Agent:  "go-pro",
		Model:  "opus",
		Prompt: "p",
	}
	args := buildSpawnArgs(agent, input)

	modelIdx := -1
	for i, a := range args {
		if a == "--model" {
			modelIdx = i
		}
	}
	require.NotEqual(t, -1, modelIdx, "--model flag must be present")
	require.Less(t, modelIdx+1, len(args), "--model must have a value")
	assert.Equal(t, "opus", args[modelIdx+1])
}

func TestBuildSpawnArgs_AllowedToolsOverride(t *testing.T) {
	agent := &routing.Agent{
		ID:    "go-pro",
		Name:  "GO Pro",
		Model: "sonnet",
	}
	input := SpawnAgentInput{
		Agent:        "go-pro",
		Prompt:       "p",
		AllowedTools: []string{"Read", "Write"},
	}
	args := buildSpawnArgs(agent, input)
	assert.Contains(t, args, "--allowedTools")

	toolsIdx := -1
	for i, a := range args {
		if a == "--allowedTools" {
			toolsIdx = i
		}
	}
	require.NotEqual(t, -1, toolsIdx)
	assert.Equal(t, "Read,Write", args[toolsIdx+1])
}

func TestBuildSpawnArgs_CustomTimeout(t *testing.T) {
	// Timeout is managed by the spawner's time.AfterFunc, NOT as a CLI flag.
	// Verify it does NOT appear in args.
	agent := &routing.Agent{ID: "go-pro", Model: "sonnet"}
	input := SpawnAgentInput{Prompt: "p", Timeout: 60000}
	args := buildSpawnArgs(agent, input)
	assert.NotContains(t, args, "--timeout")
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// minimalAgentsIndex returns a bytes slice containing a minimal but valid
// agents-index.json with one agent of the given ID.
func minimalAgentsIndex(agentID string) []byte {
	data := `{
  "version": "2.6.0",
  "generated_at": "2026-03-23T00:00:00Z",
  "description": "test index",
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
}`
	return []byte(data)
}
