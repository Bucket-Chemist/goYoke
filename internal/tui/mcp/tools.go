package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// -----------------------------------------------------------------------------
// UDS client
// -----------------------------------------------------------------------------

// UDSClient is a client for the side-channel Unix domain socket used to
// communicate with the GOgent-Fortress TUI.  Its zero value is not usable;
// use NewUDSClient.
type UDSClient struct {
	mu      sync.Mutex
	conn    net.Conn
	enc     *json.Encoder
	dec     *json.Decoder
	sockEnv string // value of GOFORTRESS_SOCKET at construction time
}

// NewUDSClient creates a UDSClient.  If the GOFORTRESS_SOCKET environment
// variable is not set the client is constructed but not connected; calls to
// SendRequest will return ErrTUINotConnected.
func NewUDSClient() *UDSClient {
	return &UDSClient{sockEnv: os.Getenv("GOFORTRESS_SOCKET")}
}

// ErrTUINotConnected is returned by interactive tools when GOFORTRESS_SOCKET
// is not set (i.e. the TUI is not running).
var ErrTUINotConnected = fmt.Errorf("TUI not connected: GOFORTRESS_SOCKET not set")

// Connect establishes the UDS connection with exponential backoff.
// It is called lazily by SendRequest on the first interactive tool call.
// Safe to call multiple times; subsequent calls are no-ops if already
// connected.
func (c *UDSClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // already connected
	}
	if c.sockEnv == "" {
		return ErrTUINotConnected
	}

	conn, err := connectWithRetry(c.sockEnv)
	if err != nil {
		return fmt.Errorf("UDS connect to %s: %w", c.sockEnv, err)
	}
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	return nil
}

// connectWithRetry dials the UDS path with exponential backoff.
// Base delay: 100ms, maximum 5 attempts.
func connectWithRetry(sockPath string) (net.Conn, error) {
	delay := 100 * time.Millisecond
	var lastErr error
	for attempt := range 5 {
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		slog.Debug("UDS connect attempt failed", "attempt", attempt+1, "delay", delay, "err", err)
		time.Sleep(delay)
		delay *= 2
	}
	return nil, fmt.Errorf("connect after 5 attempts: %w", lastErr)
}

// udsReadTimeout is the maximum time SendRequest will wait for a response.
// Modal tools (ask_user, confirm_action, etc.) can legitimately block for
// minutes while the user interacts with the TUI, so the deadline is generous.
// It exists solely to prevent a permanent hang if the TUI process dies.
const udsReadTimeout = 10 * time.Minute

// SendRequest sends req over the UDS and blocks until the matching response
// arrives.  The call is serialised by mu so only one in-flight request is
// supported at a time (sufficient for current tool set — modals are
// sequential by design).
//
// On a transient encode or decode error the connection is reset and the
// request is retried exactly once before the error is returned to the caller.
func (c *UDSClient) SendRequest(req IPCRequest) (*IPCResponse, error) {
	for attempt := range 2 {
		if err := c.Connect(); err != nil {
			return nil, err
		}

		resp, err := c.sendOnce(req)
		if err == nil {
			return resp, nil
		}

		if attempt == 0 {
			slog.Debug("UDS send failed, resetting connection for retry", "req", req.ID, "err", err)
			c.mu.Lock()
			if c.conn != nil {
				c.conn.Close()
			}
			c.conn = nil
			c.enc = nil
			c.dec = nil
			c.mu.Unlock()
			continue
		}

		return nil, err
	}
	return nil, fmt.Errorf("UDS send: exhausted retries for %s", req.ID)
}

// sendOnce performs a single encode-send-decode cycle under mu.
func (c *UDSClient) sendOnce(req IPCRequest) (*IPCResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.enc.Encode(req); err != nil {
		return nil, fmt.Errorf("UDS encode request %s: %w", req.ID, err)
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(udsReadTimeout)); err != nil {
		slog.Warn("UDS set read deadline failed", "req", req.ID, "err", err)
	}
	defer func() {
		if err := c.conn.SetReadDeadline(time.Time{}); err != nil {
			slog.Warn("UDS clear read deadline failed", "req", req.ID, "err", err)
		}
	}()

	var resp IPCResponse
	if err := c.dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("UDS decode response for %s: %w", req.ID, err)
	}

	if resp.ID != req.ID {
		return nil, fmt.Errorf("UDS correlation mismatch: sent %s, got %s", req.ID, resp.ID)
	}
	return &resp, nil
}

// send fires a one-way notification (no response expected).
// Caller must hold no lock; method acquires mu internally.
func (c *UDSClient) send(req IPCRequest) error {
	if err := c.Connect(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enc.Encode(req)
}

// sendModal is a convenience helper: it marshals payload, sends a
// TypeModalRequest, and decodes the ModalResponsePayload from the response.
func (c *UDSClient) sendModal(payload ModalRequestPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal modal request: %w", err)
	}

	req := IPCRequest{
		Type:    TypeModalRequest,
		ID:      uuid.New().String(),
		Payload: raw,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return "", fmt.Errorf("modal request: %w", err)
	}

	var mp ModalResponsePayload
	if err := json.Unmarshal(resp.Payload, &mp); err != nil {
		return "", fmt.Errorf("unmarshal modal response: %w", err)
	}
	return mp.Value, nil
}

// notify sends a one-way IPC message to the TUI, logging any errors at WARN
// level.  It is a best-effort call — tool handler responses are not delayed
// waiting for notification delivery.
func (c *UDSClient) notify(msgType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("failed to marshal IPC notification", "type", msgType, "err", err)
		return
	}
	if err := c.send(IPCRequest{
		Type:    msgType,
		ID:      uuid.New().String(),
		Payload: raw,
	}); err != nil {
		slog.Warn("failed to send IPC notification", "type", msgType, "err", err)
	}
}

// -----------------------------------------------------------------------------
// Tool input / output types
// -----------------------------------------------------------------------------

// PingInput is the input for the test_mcp_ping tool.
// All fields are optional.
type PingInput struct {
	// Echo is an optional string that is reflected back in the response.
	Echo *string `json:"echo,omitempty"`
}

// PingOutput is the response from test_mcp_ping.
type PingOutput struct {
	Status    string  `json:"status"`
	Timestamp string  `json:"timestamp"`
	Echo      *string `json:"echo"`
}

// AskUserInput is the input for the ask_user tool.
type AskUserInput struct {
	// Message is the question to present to the user. Required.
	Message string `json:"message" jsonschema:"The question or prompt to present to the user"`
	// Options is an optional list of choices.
	Options []string `json:"options,omitempty" jsonschema:"Optional list of predefined choices"`
	// Default is the pre-selected option (must appear in Options).
	Default string `json:"default,omitempty" jsonschema:"The default option (must be in Options)"`
}

// AskUserOutput is the response from ask_user.
type AskUserOutput struct {
	Answer string `json:"answer"`
}

// ConfirmActionInput is the input for the confirm_action tool.
type ConfirmActionInput struct {
	// Action describes what is about to happen. Required.
	Action string `json:"action" jsonschema:"Description of the action to confirm"`
	// Destructive hints that the action is irreversible.
	Destructive bool `json:"destructive,omitempty" jsonschema:"Whether the action is irreversible"`
}

// ConfirmActionOutput is the response from confirm_action.
type ConfirmActionOutput struct {
	Confirmed bool `json:"confirmed"`
	Cancelled bool `json:"cancelled"`
}

// RequestInputInput is the input for the request_input tool.
type RequestInputInput struct {
	// Prompt is the label shown above the text field. Required.
	Prompt string `json:"prompt" jsonschema:"Label displayed to the user"`
	// Placeholder is the ghost text shown in an empty field.
	Placeholder string `json:"placeholder,omitempty" jsonschema:"Placeholder text for the input field"`
}

// RequestInputOutput is the response from request_input.
type RequestInputOutput struct {
	Value string `json:"value"`
}

// SelectOptionEntry is a single option presented by select_option.
type SelectOptionEntry struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// SelectOptionInput is the input for the select_option tool.
type SelectOptionInput struct {
	// Message is the question displayed above the option list. Required.
	Message string `json:"message" jsonschema:"The selection prompt"`
	// Options is the list of label/value pairs. Required.
	Options []SelectOptionEntry `json:"options" jsonschema:"List of options with label and value"`
}

// SelectOptionOutput is the response from select_option.
type SelectOptionOutput struct {
	Selected string `json:"selected"`
	Index    int    `json:"index"`
}

// SpawnAgentInput is the input for the spawn_agent tool.
type SpawnAgentInput struct {
	// Agent is the agent ID from agents-index.json. Required.
	Agent string `json:"agent" jsonschema:"Agent ID from agents-index.json"`
	// Description is a brief human-readable description logged by the TUI.
	Description string `json:"description" jsonschema:"Brief description for logging"`
	// Prompt is the task prompt sent to the agent. Required.
	Prompt string `json:"prompt" jsonschema:"Task prompt for the agent"`
	// Model overrides the agent's default model.
	Model string `json:"model,omitempty" jsonschema:"Optional model override"`
	// Timeout is the deadline in milliseconds (default: 300000).
	Timeout int `json:"timeout,omitempty" jsonschema:"Timeout in ms (default 300000)"`
	// AllowedTools overrides the agent's cli_flags.allowed_tools list.
	AllowedTools []string `json:"allowedTools,omitempty" jsonschema:"Tool allowlist override"`
	// MaxBudget is a soft cost ceiling in USD.
	MaxBudget float64 `json:"maxBudget,omitempty" jsonschema:"Soft cost ceiling in USD"`
	// CallerType self-identifies the spawning agent for validation.
	CallerType string `json:"caller_type,omitempty" jsonschema:"Spawning agent ID for validation"`
}

// SpawnAgentOutput is the response from spawn_agent.
type SpawnAgentOutput struct {
	AgentID  string  `json:"agentId"`
	Agent    string  `json:"agent"`
	Success  bool    `json:"success"`
	Output   string  `json:"output"`
	Error    string  `json:"error,omitempty"`
	Cost     float64 `json:"cost"`
	Turns    int     `json:"turns"`
	Duration string  `json:"duration"`
}

// TeamRunInput is the input for the team_run tool.
type TeamRunInput struct {
	// TeamDir is the absolute path to the team configuration directory.
	// Required.
	TeamDir string `json:"team_dir" jsonschema:"Absolute path to team directory"`
	// WaitForStart blocks until gogent-team-run reports it has started.
	WaitForStart bool `json:"wait_for_start,omitempty" jsonschema:"Block until team starts"`
	// TimeoutMs is the deadline in milliseconds for the wait.
	TimeoutMs int `json:"timeout_ms,omitempty" jsonschema:"Wait timeout in ms"`
}

// TeamRunOutput is the response from team_run.
type TeamRunOutput struct {
	Success       bool   `json:"success"`
	TeamDir       string `json:"team_dir"`
	BackgroundPID int    `json:"background_pid"`
	Monitor       string `json:"monitor"`
	Result        string `json:"result"`
	Cancel        string `json:"cancel"`
}

// -----------------------------------------------------------------------------
// Tool handler constructors
// -----------------------------------------------------------------------------

// RegisterAll registers all 7 MCP tools on server.
func RegisterAll(server *mcpsdk.Server, uds *UDSClient) {
	registerTestMcpPing(server)
	registerAskUser(server, uds)
	registerConfirmAction(server, uds)
	registerRequestInput(server, uds)
	registerSelectOption(server, uds)
	registerSpawnAgent(server, uds)
	registerTeamRun(server, uds)
}

// registerTestMcpPing registers the test_mcp_ping tool.
func registerTestMcpPing(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "test_mcp_ping",
		Description: "Returns a PONG response with the current timestamp. Used for MCP connectivity validation. No UDS required.",
	}, handleTestMcpPing)
}

// handleTestMcpPing handles the test_mcp_ping tool call.
func handleTestMcpPing(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input PingInput,
) (*mcpsdk.CallToolResult, PingOutput, error) {
	return nil, PingOutput{
		Status:    "PONG",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Echo:      input.Echo,
	}, nil
}

// registerAskUser registers the ask_user tool.
func registerAskUser(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ask_user",
		Description: "Ask the user a question and return their answer. Optionally provide a list of predefined options.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input AskUserInput) (*mcpsdk.CallToolResult, AskUserOutput, error) {
		return handleAskUser(ctx, req, input, uds)
	})
}

// handleAskUser handles the ask_user tool call.
func handleAskUser(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input AskUserInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, AskUserOutput, error) {
	if input.Message == "" {
		return nil, AskUserOutput{}, fmt.Errorf("ask_user: message is required")
	}

	answer, err := uds.sendModal(ModalRequestPayload{
		Message: input.Message,
		Options: input.Options,
		Default: input.Default,
	})
	if err != nil {
		return nil, AskUserOutput{}, fmt.Errorf("ask_user: %w", err)
	}
	return nil, AskUserOutput{Answer: answer}, nil
}

// registerConfirmAction registers the confirm_action tool.
func registerConfirmAction(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "confirm_action",
		Description: "Ask the user to confirm or deny an action. Returns confirmed=true when the user clicks Allow.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input ConfirmActionInput) (*mcpsdk.CallToolResult, ConfirmActionOutput, error) {
		return handleConfirmAction(ctx, req, input, uds)
	})
}

// handleConfirmAction handles the confirm_action tool call.
func handleConfirmAction(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input ConfirmActionInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, ConfirmActionOutput, error) {
	if input.Action == "" {
		return nil, ConfirmActionOutput{}, fmt.Errorf("confirm_action: action is required")
	}

	msg := input.Action
	if input.Destructive {
		msg = "[DESTRUCTIVE] " + msg
	}

	selected, err := uds.sendModal(ModalRequestPayload{
		Message: msg,
		Options: []string{"Allow", "Deny"},
		Default: "Deny",
	})
	if err != nil {
		return nil, ConfirmActionOutput{}, fmt.Errorf("confirm_action: %w", err)
	}

	confirmed := strings.EqualFold(selected, "allow")
	return nil, ConfirmActionOutput{
		Confirmed: confirmed,
		Cancelled: !confirmed,
	}, nil
}

// registerRequestInput registers the request_input tool.
func registerRequestInput(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "request_input",
		Description: "Ask the user to type a free-text response and return the entered value.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input RequestInputInput) (*mcpsdk.CallToolResult, RequestInputOutput, error) {
		return handleRequestInput(ctx, req, input, uds)
	})
}

// handleRequestInput handles the request_input tool call.
func handleRequestInput(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input RequestInputInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, RequestInputOutput, error) {
	if input.Prompt == "" {
		return nil, RequestInputOutput{}, fmt.Errorf("request_input: prompt is required")
	}

	msg := input.Prompt
	if input.Placeholder != "" {
		msg = msg + " (" + input.Placeholder + ")"
	}

	value, err := uds.sendModal(ModalRequestPayload{
		Message: msg,
	})
	if err != nil {
		return nil, RequestInputOutput{}, fmt.Errorf("request_input: %w", err)
	}
	return nil, RequestInputOutput{Value: value}, nil
}

// registerSelectOption registers the select_option tool.
func registerSelectOption(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "select_option",
		Description: "Present the user with a labelled list of options and return the selected value and its index.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SelectOptionInput) (*mcpsdk.CallToolResult, SelectOptionOutput, error) {
		return handleSelectOption(ctx, req, input, uds)
	})
}

// handleSelectOption handles the select_option tool call.
func handleSelectOption(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input SelectOptionInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, SelectOptionOutput, error) {
	if input.Message == "" {
		return nil, SelectOptionOutput{}, fmt.Errorf("select_option: message is required")
	}
	if len(input.Options) == 0 {
		return nil, SelectOptionOutput{}, fmt.Errorf("select_option: options list is required")
	}

	labels := make([]string, len(input.Options))
	for i, opt := range input.Options {
		labels[i] = opt.Label
	}

	selected, err := uds.sendModal(ModalRequestPayload{
		Message: input.Message,
		Options: labels,
	})
	if err != nil {
		return nil, SelectOptionOutput{}, fmt.Errorf("select_option: %w", err)
	}

	// Map the returned label back to the value and index.
	for i, opt := range input.Options {
		if opt.Label == selected {
			return nil, SelectOptionOutput{Selected: opt.Value, Index: i}, nil
		}
	}

	// Fallback: selected label not in options (raw text input or mismatch).
	return nil, SelectOptionOutput{Selected: selected, Index: -1}, nil
}

// registerSpawnAgent registers the spawn_agent tool.
func registerSpawnAgent(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "spawn_agent",
		Description: "Spawn a GOgent-Fortress subagent by ID. Validates the agent configuration and runs the claude CLI subprocess. Returns the agent output and cost information.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SpawnAgentInput) (*mcpsdk.CallToolResult, SpawnAgentOutput, error) {
		return handleSpawnAgent(ctx, req, input, uds)
	})
}

// handleSpawnAgent handles the spawn_agent tool call.
// Currently implemented as a validated stub — agent config is loaded and
// validated but the CLI subprocess is not spawned.  Full subprocess
// management is tracked in a follow-up ticket.
func handleSpawnAgent(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input SpawnAgentInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, SpawnAgentOutput, error) {
	if input.Agent == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: agent is required")
	}
	if input.Description == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: description is required")
	}
	if input.Prompt == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: prompt is required")
	}

	// Validate the agent exists in agents-index.json.
	index, err := routing.LoadAgentIndex()
	if err != nil {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: load agent index: %w", err)
	}
	agent, err := index.GetAgentByID(input.Agent)
	if err != nil {
		return nil, SpawnAgentOutput{
			AgentID: "",
			Agent:   input.Agent,
			Success: false,
			Error:   fmt.Sprintf("unknown agent: %s", input.Agent),
		}, nil
	}

	agentID := uuid.New().String()

	// Notify the TUI that a new agent has been registered.
	uds.notify(TypeAgentRegister, AgentRegisterPayload{
		AgentID:   agentID,
		AgentType: agent.ID,
	})

	// Stub: return a placeholder result without spawning the subprocess.
	// Full implementation will build the `claude -p` command, set env vars,
	// stream NDJSON output, and parse cost/turn telemetry.
	slog.Info("spawn_agent stub invoked — subprocess not launched",
		"agent", input.Agent,
		"agentId", agentID,
		"description", input.Description,
	)

	uds.notify(TypeAgentUpdate, AgentUpdatePayload{
		AgentID: agentID,
		Status:  "stub",
	})

	return nil, SpawnAgentOutput{
		AgentID:  agentID,
		Agent:    agent.ID,
		Success:  true,
		Output:   fmt.Sprintf("[stub] spawn_agent called for %s — subprocess management not yet implemented", agent.ID),
		Duration: "0s",
	}, nil
}

// registerTeamRun registers the team_run tool.
func registerTeamRun(server *mcpsdk.Server, uds *UDSClient) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "team_run",
		Description: "Invoke gogent-team-run for a pre-configured team directory. The team runs in the background. Returns PID and monitoring instructions.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input TeamRunInput) (*mcpsdk.CallToolResult, TeamRunOutput, error) {
		return handleTeamRun(ctx, req, input, uds)
	})
}

// handleTeamRun handles the team_run tool call.
// Currently implemented as a validated stub — team_dir existence is checked
// but gogent-team-run is not launched.  Full background-process management
// is tracked in a follow-up ticket.
func handleTeamRun(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input TeamRunInput,
	_ *UDSClient,
) (*mcpsdk.CallToolResult, TeamRunOutput, error) {
	if input.TeamDir == "" {
		return nil, TeamRunOutput{}, fmt.Errorf("team_run: team_dir is required")
	}

	// Validate the team directory exists.
	if _, err := os.Stat(input.TeamDir); err != nil {
		return nil, TeamRunOutput{
			Success: false,
			TeamDir: input.TeamDir,
			Result:  fmt.Sprintf("team_dir not found: %s", input.TeamDir),
		}, nil
	}

	// Locate the gogent-team-run binary.
	binary, err := exec.LookPath("gogent-team-run")
	if err != nil {
		return nil, TeamRunOutput{
			Success: false,
			TeamDir: input.TeamDir,
			Result:  "gogent-team-run binary not found in PATH",
		}, nil
	}

	slog.Info("team_run stub invoked — subprocess not launched",
		"team_dir", input.TeamDir,
		"binary", binary,
	)

	return nil, TeamRunOutput{
		Success:       true,
		TeamDir:       input.TeamDir,
		BackgroundPID: 0,
		Monitor:       fmt.Sprintf("gogent-team-run %s &  # not launched (stub)", input.TeamDir),
		Result:        "[stub] team_run called — background process management not yet implemented",
		Cancel:        "kill 0  # no process to kill (stub)",
	}, nil
}

// buildSpawnArgs constructs the claude CLI arguments for an agent spawn.
// This is exported for testability even though the stub does not call it yet.
func buildSpawnArgs(agent *routing.Agent, input SpawnAgentInput) []string {
	args := []string{"-p", "--output-format", "stream-json", "--no-cache"}

	// Model
	model := agent.Model
	if input.Model != "" {
		model = input.Model
	}
	args = append(args, "--model", model)

	// Allowed tools
	tools := agent.GetAllowedTools()
	if len(input.AllowedTools) > 0 {
		tools = input.AllowedTools
	}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}

	// Timeout (default: 5 minutes)
	const defaultAgentTimeoutMS = 300_000
	timeout := defaultAgentTimeoutMS
	if input.Timeout > 0 {
		timeout = input.Timeout
	}
	args = append(args, "--timeout", strconv.Itoa(timeout))

	return args
}
