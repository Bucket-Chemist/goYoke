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

func TestPermGateRequestRoundtrip(t *testing.T) {
	toolInput := json.RawMessage(`{"path":"/tmp/secret.txt"}`)
	payload := PermGateRequestPayload{
		ToolName:  "Write",
		ToolInput: toolInput,
		SessionID: "sess-42",
		TimeoutMS: 30000,
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	req := IPCRequest{
		Type:    TypePermGateRequest,
		ID:      "perm-req-1",
		Payload: raw,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var got IPCRequest
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, req.Type, got.Type)
	assert.Equal(t, req.ID, got.ID)

	var gotPayload PermGateRequestPayload
	require.NoError(t, json.Unmarshal(got.Payload, &gotPayload))
	assert.Equal(t, payload.ToolName, gotPayload.ToolName)
	assert.Equal(t, payload.SessionID, gotPayload.SessionID)
	assert.Equal(t, payload.TimeoutMS, gotPayload.TimeoutMS)
	assert.JSONEq(t, string(toolInput), string(gotPayload.ToolInput))
}

func makeTestSocketPath(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "gofortress-uds-")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return filepath.Join(dir, name)
}

func TestPermGateResponseRoundtrip(t *testing.T) {
	decisions := []string{"allow", "deny", "allow_session"}

	for _, decision := range decisions {
		t.Run(decision, func(t *testing.T) {
			payload := PermGateResponsePayload{Decision: decision}
			raw, err := json.Marshal(payload)
			require.NoError(t, err)

			resp := IPCResponse{
				Type:    TypePermGateResponse,
				ID:      "perm-req-1",
				Payload: raw,
			}

			data, err := json.Marshal(resp)
			require.NoError(t, err)

			var got IPCResponse
			require.NoError(t, json.Unmarshal(data, &got))
			assert.Equal(t, resp.Type, got.Type)
			assert.Equal(t, resp.ID, got.ID)

			var gotPayload PermGateResponsePayload
			require.NoError(t, json.Unmarshal(got.Payload, &gotPayload))
			assert.Equal(t, decision, gotPayload.Decision)
		})
	}
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
		{"PermGateRequestPayload", PermGateRequestPayload{ToolName: "Read", ToolInput: json.RawMessage(`{}`), SessionID: "s1", TimeoutMS: 5000}},
		{"PermGateResponsePayload", PermGateResponsePayload{Decision: "allow"}},
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
	sockPath := makeTestSocketPath(t, "test.sock")

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
		SpawnAgentInput{Description: "d", Prompt: "p"}, uds, NewAgentStore())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent is required")
}

func TestHandleSpawnAgent_MissingDescription(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Agent: "go-pro", Prompt: "p"}, uds, NewAgentStore())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestHandleSpawnAgent_MissingPrompt(t *testing.T) {
	uds := &UDSClient{}
	_, _, err := handleSpawnAgent(context.Background(), nil,
		SpawnAgentInput{Agent: "go-pro", Description: "d"}, uds, NewAgentStore())
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
		SpawnAgentInput{Agent: "totally-unknown", Description: "d", Prompt: "p"}, uds, NewAgentStore())
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
// ensureTeamVisible — symlink bridge
// -----------------------------------------------------------------------------

func TestEnsureTeamVisible_CreatesSymlink(t *testing.T) {
	// Set up a fake ~/.gogent/current-session pointing to a temp TUI session.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("GOGENT_CONFIG_DIR", "")

	tuiSessionDir := filepath.Join(fakeHome, ".gogent", "sessions", "20260331.test-session")
	require.NoError(t, os.MkdirAll(tuiSessionDir, 0o755))

	markerDir := filepath.Join(fakeHome, ".gogent")
	require.NoError(t, os.WriteFile(
		filepath.Join(markerDir, "current-session"),
		[]byte(tuiSessionDir),
		0o644,
	))

	// Create a team dir outside the TUI session tree (simulating CC CLI session).
	ccTeamDir := filepath.Join(t.TempDir(), "teams", "20260331_120000.braintrust")
	require.NoError(t, os.MkdirAll(ccTeamDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(ccTeamDir, "config.json"),
		[]byte(`{"status":"running"}`),
		0o644,
	))

	// Call ensureTeamVisible — should create symlink.
	ensureTeamVisible(ccTeamDir)

	symlinkPath := filepath.Join(tuiSessionDir, "teams", "20260331_120000.braintrust")
	info, err := os.Lstat(symlinkPath)
	require.NoError(t, err, "symlink should exist")
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "should be a symlink")

	target, err := os.Readlink(symlinkPath)
	require.NoError(t, err)
	assert.Equal(t, ccTeamDir, target)
}

func TestEnsureTeamVisible_RespectsClaudeConfigDir(t *testing.T) {
	// When CLAUDE_CONFIG_DIR is set, ensureTeamVisible should read the marker
	// from $CLAUDE_CONFIG_DIR/current-session instead of ~/.gogent/current-session.
	fakeHome := t.TempDir()
	customConfig := filepath.Join(fakeHome, ".claude-em")
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", customConfig)
	t.Setenv("GOGENT_CONFIG_DIR", "")

	tuiSessionDir := filepath.Join(customConfig, "sessions", "20260410.custom-session")
	require.NoError(t, os.MkdirAll(tuiSessionDir, 0o755))

	// Write marker under custom config dir (where the TUI session store writes it).
	require.NoError(t, os.WriteFile(
		filepath.Join(customConfig, "current-session"),
		[]byte(tuiSessionDir),
		0o644,
	))

	// Do NOT create ~/.gogent/current-session — ensureTeamVisible must not fall back to it.

	ccTeamDir := filepath.Join(t.TempDir(), "teams", "test-team")
	require.NoError(t, os.MkdirAll(ccTeamDir, 0o755))

	ensureTeamVisible(ccTeamDir)

	symlinkPath := filepath.Join(tuiSessionDir, "teams", "test-team")
	info, err := os.Lstat(symlinkPath)
	require.NoError(t, err, "symlink should exist under CLAUDE_CONFIG_DIR session")
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "should be a symlink")

	target, err := os.Readlink(symlinkPath)
	require.NoError(t, err)
	assert.Equal(t, ccTeamDir, target)
}

func TestEnsureTeamVisible_RespectsGogentConfigDir(t *testing.T) {
	// GOGENT_CONFIG_DIR takes second priority after CLAUDE_CONFIG_DIR.
	fakeHome := t.TempDir()
	customConfig := filepath.Join(fakeHome, ".custom-gogent")
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("GOGENT_CONFIG_DIR", customConfig)

	tuiSessionDir := filepath.Join(customConfig, "sessions", "20260410.gogent-session")
	require.NoError(t, os.MkdirAll(tuiSessionDir, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(customConfig, "current-session"),
		[]byte(tuiSessionDir),
		0o644,
	))

	ccTeamDir := filepath.Join(t.TempDir(), "teams", "test-team")
	require.NoError(t, os.MkdirAll(ccTeamDir, 0o755))

	ensureTeamVisible(ccTeamDir)

	symlinkPath := filepath.Join(tuiSessionDir, "teams", "test-team")
	info, err := os.Lstat(symlinkPath)
	require.NoError(t, err, "symlink should exist under GOGENT_CONFIG_DIR session")
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
}

func TestEnsureTeamVisible_ClaudeConfigDirTakesPriority(t *testing.T) {
	// When both CLAUDE_CONFIG_DIR and GOGENT_CONFIG_DIR are set,
	// CLAUDE_CONFIG_DIR wins.
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude-em")
	gogentDir := filepath.Join(fakeHome, ".custom-gogent")
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	t.Setenv("GOGENT_CONFIG_DIR", gogentDir)

	// Set up marker under CLAUDE_CONFIG_DIR.
	claudeSession := filepath.Join(claudeDir, "sessions", "20260410.claude-session")
	require.NoError(t, os.MkdirAll(claudeSession, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "current-session"),
		[]byte(claudeSession),
		0o644,
	))

	// Set up marker under GOGENT_CONFIG_DIR (should be ignored).
	gogentSession := filepath.Join(gogentDir, "sessions", "20260410.gogent-session")
	require.NoError(t, os.MkdirAll(gogentSession, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(gogentDir, "current-session"),
		[]byte(gogentSession),
		0o644,
	))

	ccTeamDir := filepath.Join(t.TempDir(), "teams", "priority-team")
	require.NoError(t, os.MkdirAll(ccTeamDir, 0o755))

	ensureTeamVisible(ccTeamDir)

	// Symlink should be under CLAUDE_CONFIG_DIR session, not GOGENT_CONFIG_DIR.
	claudeSymlink := filepath.Join(claudeSession, "teams", "priority-team")
	gogentSymlink := filepath.Join(gogentSession, "teams", "priority-team")

	_, err := os.Lstat(claudeSymlink)
	assert.NoError(t, err, "symlink should exist under CLAUDE_CONFIG_DIR")

	_, err = os.Lstat(gogentSymlink)
	assert.True(t, os.IsNotExist(err), "symlink should NOT exist under GOGENT_CONFIG_DIR")
}

func TestEnsureTeamVisible_SkipsWhenAlreadyInTUITree(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("GOGENT_CONFIG_DIR", "")

	tuiSessionDir := filepath.Join(fakeHome, ".gogent", "sessions", "20260331.test-session")
	tuiTeamsDir := filepath.Join(tuiSessionDir, "teams")
	require.NoError(t, os.MkdirAll(tuiTeamsDir, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(fakeHome, ".gogent", "current-session"),
		[]byte(tuiSessionDir),
		0o644,
	))

	// Team dir is already inside the TUI's teams dir.
	teamDir := filepath.Join(tuiTeamsDir, "20260331_120000.braintrust")
	require.NoError(t, os.MkdirAll(teamDir, 0o755))

	// Should not create a symlink to itself.
	ensureTeamVisible(teamDir)

	entries, err := os.ReadDir(tuiTeamsDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no extra symlink should be created")
}

func TestEnsureTeamVisible_IdempotentOnRerun(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("GOGENT_CONFIG_DIR", "")

	tuiSessionDir := filepath.Join(fakeHome, ".gogent", "sessions", "20260331.test-session")
	require.NoError(t, os.MkdirAll(tuiSessionDir, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(fakeHome, ".gogent", "current-session"),
		[]byte(tuiSessionDir),
		0o644,
	))

	ccTeamDir := filepath.Join(t.TempDir(), "teams", "20260331_120000.braintrust")
	require.NoError(t, os.MkdirAll(ccTeamDir, 0o755))

	// Call twice — second call should not error.
	ensureTeamVisible(ccTeamDir)
	ensureTeamVisible(ccTeamDir)

	symlinkPath := filepath.Join(tuiSessionDir, "teams", "20260331_120000.braintrust")
	_, err := os.Lstat(symlinkPath)
	assert.NoError(t, err, "symlink should still exist after idempotent call")
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
		"get_agent_result",
		"team_run",
	}
	for _, name := range expected {
		assert.Contains(t, names, name, "tool %s should be registered", name)
	}
}

// -----------------------------------------------------------------------------
// get_agent_result handler
// -----------------------------------------------------------------------------

func TestHandleGetAgentResult_MissingAgentID(t *testing.T) {
	store := NewAgentStore()
	_, _, err := handleGetAgentResult(context.Background(), GetAgentResultInput{}, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agentId is required")
}

func TestHandleGetAgentResult_NotFound(t *testing.T) {
	store := NewAgentStore()
	_, out, err := handleGetAgentResult(context.Background(), GetAgentResultInput{
		AgentID: "nonexistent",
	}, store)
	require.NoError(t, err)
	assert.Equal(t, "not_found", out.Status)
}

func TestHandleGetAgentResult_RunningNoWait(t *testing.T) {
	store := NewAgentStore()
	store.Register("agent-1", "go-pro")

	_, out, err := handleGetAgentResult(context.Background(), GetAgentResultInput{
		AgentID: "agent-1",
	}, store)
	require.NoError(t, err)
	assert.Equal(t, "running", out.Status)
	assert.False(t, out.Success)
}

func TestHandleGetAgentResult_CompletedImmediate(t *testing.T) {
	store := NewAgentStore()
	store.Register("agent-1", "go-pro")
	store.Complete("agent-1", "result text", "", 0.25, 10, "2s")

	_, out, err := handleGetAgentResult(context.Background(), GetAgentResultInput{
		AgentID: "agent-1",
	}, store)
	require.NoError(t, err)
	assert.Equal(t, "complete", out.Status)
	assert.True(t, out.Success)
	assert.Equal(t, "result text", out.Output)
	assert.Equal(t, 0.25, out.Cost)
}

func TestHandleGetAgentResult_WaitForCompletion(t *testing.T) {
	store := NewAgentStore()
	store.Register("agent-1", "go-pro")

	// Complete in background after 50ms.
	go func() {
		time.Sleep(50 * time.Millisecond)
		store.Complete("agent-1", "async result", "", 0.10, 5, "50ms")
	}()

	_, out, err := handleGetAgentResult(context.Background(), GetAgentResultInput{
		AgentID:   "agent-1",
		Wait:      true,
		TimeoutMs: 5000,
	}, store)
	require.NoError(t, err)
	assert.Equal(t, "complete", out.Status)
	assert.Equal(t, "async result", out.Output)
}

func TestHandleGetAgentResult_WaitTimeout(t *testing.T) {
	store := NewAgentStore()
	store.Register("agent-1", "go-pro")
	// Never complete — should timeout.

	_, out, err := handleGetAgentResult(context.Background(), GetAgentResultInput{
		AgentID:   "agent-1",
		Wait:      true,
		TimeoutMs: 100,
	}, store)
	require.NoError(t, err)
	assert.Equal(t, "running", out.Status)
	assert.Contains(t, out.Error, "timed out")
}

// -----------------------------------------------------------------------------
// AgentStore
// -----------------------------------------------------------------------------

func TestAgentStore_EvictExpired(t *testing.T) {
	store := NewAgentStore()
	defer store.Stop()

	store.Register("old-1", "go-pro")
	store.Register("old-2", "go-pro")
	store.Register("still-running", "go-pro")

	// Complete two entries, then backdate their DoneAt.
	store.Complete("old-1", "done", "", 0, 0, "1s")
	store.Complete("old-2", "", "failed", 0, 0, "2s")

	// Backdate DoneAt to exceed TTL.
	store.mu.Lock()
	store.entries["old-1"].DoneAt = time.Now().Add(-31 * time.Minute)
	store.entries["old-2"].DoneAt = time.Now().Add(-31 * time.Minute)
	store.mu.Unlock()

	// Run eviction manually.
	store.evictExpired()

	assert.Nil(t, store.Get("old-1"), "old-1 should be evicted")
	assert.Nil(t, store.Get("old-2"), "old-2 should be evicted")
	assert.NotNil(t, store.Get("still-running"), "running entry must not be evicted")
	assert.Equal(t, 1, store.Len())
}

func TestAgentStore_RecentCompletedNotEvicted(t *testing.T) {
	store := NewAgentStore()
	defer store.Stop()

	store.Register("recent", "go-pro")
	store.Complete("recent", "done", "", 0, 0, "1s")

	store.evictExpired()

	assert.NotNil(t, store.Get("recent"), "recently completed entry must not be evicted")
}

func TestAgentStore_DoubleCompleteNoPanic(t *testing.T) {
	store := NewAgentStore()
	defer store.Stop()

	store.Register("agent-1", "go-pro")
	store.Complete("agent-1", "first", "", 0.1, 5, "1s")
	// Second complete should not panic or overwrite.
	store.Complete("agent-1", "second", "", 0.2, 10, "2s")

	entry := store.Get("agent-1")
	assert.Equal(t, "first", entry.Output, "second Complete should be ignored")
	assert.Equal(t, 0.1, entry.Cost)
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

func TestBuildSpawnArgs_MCPConfig_InteractiveWithEnv(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := &routing.Agent{ID: "spawn-agent", Model: "sonnet", Interactive: true}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.Contains(t, args, "--mcp-config")
	assert.Contains(t, args, "/tmp/mcp-config.json")

	// --mcp-config must appear before --allowedTools
	mcpIdx, toolsIdx := -1, -1
	for i, a := range args {
		if a == "--mcp-config" {
			mcpIdx = i
		}
		if a == "--allowedTools" {
			toolsIdx = i
		}
	}
	if toolsIdx != -1 {
		assert.Less(t, mcpIdx, toolsIdx, "--mcp-config must appear before --allowedTools")
	}

	// MCP glob must be in the allowedTools value
	for i, a := range args {
		if a == "--allowedTools" && i+1 < len(args) {
			assert.Contains(t, args[i+1], "mcp__gofortress-interactive__*")
		}
	}
}

func TestBuildSpawnArgs_MCPConfig_NonInteractiveAgent(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := &routing.Agent{ID: "go-pro", Model: "sonnet", Interactive: false}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.NotContains(t, args, "--mcp-config")
	// MCP glob must NOT appear for non-interactive agents
	for _, a := range args {
		assert.NotContains(t, a, "mcp__gofortress-interactive__*")
	}
}

func TestBuildSpawnArgs_MCPConfig_InteractiveNoEnv(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "")
	agent := &routing.Agent{ID: "spawn-agent", Model: "sonnet", Interactive: true}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.NotContains(t, args, "--mcp-config")
	for _, a := range args {
		assert.NotContains(t, a, "mcp__gofortress-interactive__*")
	}
}

func TestBuildSpawnArgs_MCPConfig_InteractiveWithExplicitAllowedTools(t *testing.T) {
	// MCP glob must be appended even when caller provides an explicit AllowedTools override.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/test-mcp.json")
	agent := &routing.Agent{ID: "spawn-agent", Model: "sonnet", Interactive: true}
	input := SpawnAgentInput{
		Prompt:       "p",
		AllowedTools: []string{"Read", "Write", "Bash"},
	}
	args := buildSpawnArgs(agent, input)

	assert.Contains(t, args, "--mcp-config")
	assert.Contains(t, args, "/tmp/test-mcp.json")

	toolsIdx := -1
	for i, a := range args {
		if a == "--allowedTools" {
			toolsIdx = i
		}
	}
	require.NotEqual(t, -1, toolsIdx, "--allowedTools must be present")
	require.Less(t, toolsIdx+1, len(args))
	toolsVal := args[toolsIdx+1]
	assert.Contains(t, toolsVal, "mcp__gofortress-interactive__*", "MCP glob must be appended even with explicit AllowedTools")
	assert.Contains(t, toolsVal, "Read", "explicit tools must be preserved")
	assert.Contains(t, toolsVal, "Write", "explicit tools must be preserved")
	assert.Contains(t, toolsVal, "Bash", "explicit tools must be preserved")
}

func TestBuildSpawnArgs_MCPConfig_NonInteractiveNoEnv(t *testing.T) {
	// Baseline: non-interactive agent with no env var — no --mcp-config, no MCP tools.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "")
	agent := &routing.Agent{ID: "go-pro", Model: "sonnet", Interactive: false}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.NotContains(t, args, "--mcp-config")
	for _, a := range args {
		assert.NotContains(t, a, "mcp__gofortress-interactive__*")
	}
}

func TestBuildSpawnArgs_DisallowedTools_InteractiveWithMCP(t *testing.T) {
	// Interactive agent + GOFORTRESS_MCP_CONFIG → must have --disallowedTools blocking
	// built-in equivalents that are replaced by MCP tools.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := &routing.Agent{ID: "mozart", Model: "opus", Interactive: true}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	disallowIdx := -1
	for i, a := range args {
		if a == "--disallowedTools" {
			disallowIdx = i
		}
	}
	require.NotEqual(t, -1, disallowIdx, "--disallowedTools must be present for interactive+MCP")
	require.Less(t, disallowIdx+1, len(args))
	assert.Contains(t, args[disallowIdx+1], "Task")
	assert.Contains(t, args[disallowIdx+1], "AskUserQuestion")
}

func TestBuildSpawnArgs_DisallowedTools_InteractiveNoMCP(t *testing.T) {
	// Interactive agent WITHOUT GOFORTRESS_MCP_CONFIG → no --disallowedTools.
	// Built-ins must remain available when MCP bridge is absent.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "")
	agent := &routing.Agent{ID: "mozart", Model: "opus", Interactive: true}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.NotContains(t, args, "--disallowedTools")
}

func TestBuildSpawnArgs_DisallowedTools_NonInteractiveWithMCP(t *testing.T) {
	// Non-interactive agent even with GOFORTRESS_MCP_CONFIG → no --disallowedTools.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := &routing.Agent{ID: "go-pro", Model: "sonnet", Interactive: false}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	assert.NotContains(t, args, "--disallowedTools")
}

func TestBuildSpawnArgs_AdditionalFlags(t *testing.T) {
	// additional_flags from agent config are applied (except --permission-mode).
	agent := &routing.Agent{
		ID:    "test-agent",
		Model: "sonnet",
		CliFlags: &routing.AgentCliFlags{
			AllowedTools:    []string{"Read", "Glob"},
			AdditionalFlags: []string{"--permission-mode", "delegate", "--effort", "high"},
		},
	}
	input := SpawnAgentInput{Prompt: "p"}
	args := buildSpawnArgs(agent, input)

	// --effort high should be present.
	effortIdx := -1
	for i, a := range args {
		if a == "--effort" {
			effortIdx = i
		}
	}
	require.NotEqual(t, -1, effortIdx, "--effort from additional_flags must be present")
	assert.Equal(t, "high", args[effortIdx+1])

	// --permission-mode from additional_flags must be filtered out
	// (buildSpawnArgs already sets bypassPermissions).
	permCount := 0
	for _, a := range args {
		if a == "--permission-mode" {
			permCount++
		}
	}
	assert.Equal(t, 1, permCount, "only the hardcoded --permission-mode should be present")
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// minimalAgentsIndex returns a bytes slice containing a minimal but valid
// agents-index.json with one agent of the given ID.
func minimalAgentsIndex(agentID string) []byte {
	data := `{
  "version": "2.7.0",
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
