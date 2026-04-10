package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
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
	// Timeout is the deadline in milliseconds (default: 600000).
	Timeout int `json:"timeout,omitempty" jsonschema:"Timeout in ms (default 600000)"`
	// AllowedTools overrides the agent's cli_flags.allowed_tools list.
	AllowedTools []string `json:"allowedTools,omitempty" jsonschema:"Tool allowlist override"`
	// MaxBudget is a soft cost ceiling in USD.
	MaxBudget float64 `json:"maxBudget,omitempty" jsonschema:"Soft cost ceiling in USD"`
	// CallerType self-identifies the spawning agent for validation.
	CallerType string `json:"caller_type,omitempty" jsonschema:"Spawning agent ID for validation"`
	// AcceptanceCriteria is an optional list of criteria the agent must satisfy.
	// For Sonnet+ tier agents these are appended to the prompt as a TodoWrite
	// task list.  Haiku agents (<tier 2) ignore this field.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty" jsonschema:"Acceptance criteria to inject into prompt"`
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
	store := NewAgentStore()

	registerTestMcpPing(server)
	registerAskUser(server, uds)
	registerConfirmAction(server, uds)
	registerRequestInput(server, uds)
	registerSelectOption(server, uds)
	registerSpawnAgent(server, uds, store)
	registerGetAgentResult(server, store)
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
func registerSpawnAgent(server *mcpsdk.Server, uds *UDSClient, store *AgentStore) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "spawn_agent",
		Description: "Spawn a GOgent-Fortress subagent by ID. Launches the subprocess asynchronously and returns immediately with an agentId. Use get_agent_result to poll for the result.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SpawnAgentInput) (*mcpsdk.CallToolResult, SpawnAgentOutput, error) {
		return handleSpawnAgent(ctx, req, input, uds, store)
	})
}

// handleSpawnAgent handles the spawn_agent tool call.
// It validates the request, loads agent config, injects identity context, and
// launches a claude CLI subprocess ASYNCHRONOUSLY.  Returns immediately with
// the agentId.  The caller uses get_agent_result to poll for completion.
//
// CRITICAL: The subprocess goroutine uses context.Background(), NOT the MCP
// request context.  The MCP request context is cancelled when handleSpawnAgent
// returns, which would kill the subprocess.  runSubprocess manages its own
// timeout via time.AfterFunc.
func handleSpawnAgent(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input SpawnAgentInput,
	uds *UDSClient,
	store *AgentStore,
) (*mcpsdk.CallToolResult, SpawnAgentOutput, error) {
	// 1. Validate nesting depth.
	if err := validateNestingDepth(); err != nil {
		return nil, SpawnAgentOutput{
			Agent:   input.Agent,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 2. Validate required fields.
	if input.Agent == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: agent is required")
	}
	if input.Description == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: description is required")
	}
	if input.Prompt == "" {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: prompt is required")
	}

	// 3. Load agent index.
	index, err := routing.LoadAgentIndex()
	if err != nil {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: load agent index: %w", err)
	}

	// 4. Verify the requested agent exists.
	agent, err := index.GetAgentByID(input.Agent)
	if err != nil {
		return nil, SpawnAgentOutput{
			AgentID: "",
			Agent:   input.Agent,
			Success: false,
			Error:   fmt.Sprintf("unknown agent: %s", input.Agent),
		}, nil
	}

	// 4b. Validate relationship constraints (M-3 fix: parity with standalone).
	parentType := os.Getenv("GOGENT_PARENT_AGENT")
	vr := validateRelationship(index, parentType, input.Agent, input.CallerType)
	if !vr.Valid {
		return nil, SpawnAgentOutput{
			Agent:   input.Agent,
			Success: false,
			Error:   "spawn validation failed: " + fmt.Sprintf("%v", vr.Errors),
		}, nil
	}
	if len(vr.Warnings) > 0 {
		slog.Warn("spawn_agent relationship warnings", "warnings", vr.Warnings, "agent", input.Agent)
	}

	// 5. Generate a unique agent instance ID.
	agentID := uuid.New().String()

	// 6. Build the augmented prompt with agent identity and context.
	// m-1 fix: pass agent.ContextRequirements (was nil, parity with standalone).
	augmented, err := routing.BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, input.Prompt)
	if err != nil {
		slog.Warn("failed to build agent context", "err", err, "agent", agent.ID)
		augmented = input.Prompt
	}

	// 6b. Merge and inject acceptance criteria for Sonnet+ tier agents.
	// Haiku agents (tier < 2) are skipped — they lack the TodoWrite tool.
	mergedAC := mergeAcceptanceCriteria(nil, input.AcceptanceCriteria)
	if len(mergedAC) > 0 && agentTierNumber(agent.Tier) >= 2 {
		var sb strings.Builder
		sb.WriteString(augmented)
		sb.WriteString("\n\n## Acceptance Criteria (MANDATORY)\n\n")
		sb.WriteString("You MUST use the TodoWrite tool to track these criteria. Rules:\n")
		sb.WriteString("1. Create your TodoWrite task list using the EXACT text below as task content — do NOT paraphrase or abbreviate.\n")
		sb.WriteString("2. Mark each criterion as completed (status: \"completed\") when satisfied.\n")
		sb.WriteString("3. If you cannot satisfy a criterion, mark it as status \"in_progress\" with a note — never silently skip.\n")
		sb.WriteString("4. Before finishing, call TodoWrite one final time with ALL criteria to confirm their status.\n\n")
		for _, criterion := range mergedAC {
			sb.WriteString("- [ ] ")
			sb.WriteString(criterion)
			sb.WriteString("\n")
		}
		augmented = sb.String()
	} else {
		// Tier < 2 or no AC — clear merged list so it is not sent to TUI.
		mergedAC = nil
	}

	// 7. Notify the TUI that a new agent has been registered.
	promptPreview := augmented
	if len(promptPreview) > 2000 {
		promptPreview = promptPreview[:2000] + "\n[TRUNCATED]"
	}
	tierStr := fmt.Sprintf("%v", agent.Tier)
	modelStr := input.Model
	if modelStr == "" {
		modelStr = agent.Model
	}
	uds.notify(TypeAgentRegister, AgentRegisterPayload{
		AgentID:            agentID,
		AgentType:          agent.ID,
		ParentID:           parentType,
		Model:              modelStr,
		Tier:               tierStr,
		Description:        input.Description,
		Conventions:        agent.ConventionsRequired,
		Prompt:             promptPreview,
		AcceptanceCriteria: mergedAC,
	})

	// 8. Notify the TUI that the agent is queued (will become "running"
	//    once it acquires a concurrency slot).
	uds.notify(TypeAgentUpdate, AgentUpdatePayload{
		AgentID: agentID,
		Status:  "queued",
	})

	// 9. Register in the store and launch the subprocess asynchronously.
	store.Register(agentID, agent.ID)

	slog.Info("spawn_agent: queued subprocess (async)",
		"agent", input.Agent,
		"agentId", agentID,
		"description", input.Description,
		"maxConcurrent", maxConcurrentSpawns,
	)

	// Capture values needed by the goroutine (avoid closure over mutable state).
	capturedAgent := agent
	capturedInput := input
	capturedAugmented := augmented

	go func() {
		// Acquire a concurrency slot. Blocks if maxConcurrentSpawns are
		// already running. This prevents Anthropic API 429 rate-limit
		// errors when multiple agents are spawned in parallel.
		store.SpawnSem <- struct{}{}
		defer func() { <-store.SpawnSem }()

		// Now running — update TUI status from "queued" to "running".
		uds.notify(TypeAgentUpdate, AgentUpdatePayload{
			AgentID: agentID,
			Status:  "running",
		})
		slog.Info("spawn_agent: acquired slot, launching subprocess",
			"agent", capturedInput.Agent, "agentId", agentID)

		// CRITICAL: use context.Background() — the MCP request context is
		// already cancelled by the time this goroutine runs.  runSubprocess
		// manages its own timeout internally via time.AfterFunc.
		bgCtx := context.Background()
		start := time.Now()
		result, runErr := runSubprocess(bgCtx, capturedAgent, capturedInput, capturedAugmented, agentID, uds)
		duration := time.Since(start).Round(time.Millisecond).String()

		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}

		output := ""
		cost := 0.0
		turns := 0
		if result != nil {
			output = result.Result
			cost = result.TotalCostUSD
			turns = result.NumTurns
		}

		// Update the store — this signals any waiters on get_agent_result.
		store.Complete(agentID, output, errMsg, cost, turns, duration)

		// Notify the TUI that the agent has completed.
		status := "complete"
		if errMsg != "" {
			status = "error"
		}
		uds.notify(TypeAgentUpdate, AgentUpdatePayload{
			AgentID: agentID,
			Status:  status,
		})

		slog.Info("spawn_agent: subprocess finished",
			"agent", capturedInput.Agent,
			"agentId", agentID,
			"success", errMsg == "",
			"duration", duration,
			"cost", cost,
		)
	}()

	// 10. Return immediately — subprocess runs in background.
	return nil, SpawnAgentOutput{
		AgentID:  agentID,
		Agent:    agent.ID,
		Success:  true,
		Output:   "Agent launched asynchronously. Use get_agent_result to poll for the result.",
		Duration: "0ms",
	}, nil
}

// agentTierNumber converts a routing.Agent.Tier value (any — float64 or string)
// to a float64 for numeric comparison.  Non-numeric tiers (e.g. "external")
// return 0.
func agentTierNumber(tier any) float64 {
	switch v := tier.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// mergeAcceptanceCriteria combines defaults and caller-provided criteria,
// returning a deduplicated list (case-insensitive comparison).  Either argument
// may be nil.
func mergeAcceptanceCriteria(defaults, caller []string) []string {
	seen := make(map[string]struct{}, len(defaults)+len(caller))
	var merged []string
	for _, ac := range append(defaults, caller...) {
		key := strings.ToLower(strings.TrimSpace(ac))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, ac)
	}
	return merged
}

// GetAgentResultInput is the input for the get_agent_result tool.
type GetAgentResultInput struct {
	// AgentID is the ID returned by spawn_agent.
	AgentID string `json:"agentId" jsonschema:"Agent ID returned by spawn_agent"`
	// Wait blocks until the agent completes or timeout is reached.
	// If false, returns immediately with current status.
	Wait bool `json:"wait,omitempty" jsonschema:"Block until agent completes (default: false)"`
	// TimeoutMs is the max time to wait when Wait=true (default: 600000 = 10min).
	TimeoutMs int `json:"timeout_ms,omitempty" jsonschema:"Max wait time in ms (default 600000)"`
}

// GetAgentResultOutput is the response from get_agent_result.
type GetAgentResultOutput struct {
	AgentID  string  `json:"agentId"`
	Agent    string  `json:"agent"`
	Status   string  `json:"status"` // "running", "complete", "error"
	Success  bool    `json:"success"`
	Output   string  `json:"output,omitempty"`
	Error    string  `json:"error,omitempty"`
	Cost     float64 `json:"cost,omitempty"`
	Turns    int     `json:"turns,omitempty"`
	Duration string  `json:"duration,omitempty"`
}

// registerGetAgentResult registers the get_agent_result tool.
func registerGetAgentResult(server *mcpsdk.Server, store *AgentStore) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_agent_result",
		Description: "Get the result of an async spawn_agent call. Returns immediately with status, or blocks until completion if wait=true.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetAgentResultInput) (*mcpsdk.CallToolResult, GetAgentResultOutput, error) {
		return handleGetAgentResult(ctx, input, store)
	})
}

// handleGetAgentResult handles the get_agent_result tool call.
func handleGetAgentResult(
	ctx context.Context,
	input GetAgentResultInput,
	store *AgentStore,
) (*mcpsdk.CallToolResult, GetAgentResultOutput, error) {
	if input.AgentID == "" {
		return nil, GetAgentResultOutput{}, fmt.Errorf("get_agent_result: agentId is required")
	}

	entry := store.Get(input.AgentID)
	if entry == nil {
		return nil, GetAgentResultOutput{
			AgentID: input.AgentID,
			Status:  "not_found",
			Error:   "no agent with this ID (may have expired or wrong ID)",
		}, nil
	}

	// If already done or caller doesn't want to wait, return immediately.
	if entry.State != AgentStateRunning || !input.Wait {
		return nil, entryToOutput(entry), nil
	}

	// Block until done or timeout.
	timeoutMs := input.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 600_000 // 10 minutes default
	}

	doneCh := store.DoneChan(input.AgentID)
	if doneCh == nil {
		// Race: completed between Get and DoneChan.
		entry = store.Get(input.AgentID)
		return nil, entryToOutput(entry), nil
	}

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-doneCh:
		entry = store.Get(input.AgentID)
		return nil, entryToOutput(entry), nil
	case <-timer.C:
		entry = store.Get(input.AgentID)
		out := entryToOutput(entry)
		out.Error = "wait timed out, agent still running"
		return nil, out, nil
	case <-ctx.Done():
		entry = store.Get(input.AgentID)
		return nil, entryToOutput(entry), nil
	}
}

// entryToOutput converts an agentEntry to GetAgentResultOutput.
func entryToOutput(entry *agentEntry) GetAgentResultOutput {
	if entry == nil {
		return GetAgentResultOutput{Status: "not_found"}
	}
	return GetAgentResultOutput{
		AgentID:  entry.AgentID,
		Agent:    entry.Agent,
		Status:   string(entry.State),
		Success:  entry.State == AgentStateComplete,
		Output:   entry.Output,
		Error:    entry.Error,
		Cost:     entry.Cost,
		Turns:    entry.Turns,
		Duration: entry.Duration,
	}
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

// teamRunPollInterval is the initial backoff delay when polling config.json
// for the background_pid written by gogent-team-run on startup.
const teamRunPollInterval = 100 * time.Millisecond

// teamRunMaxPollInterval caps the exponential backoff used during polling.
const teamRunMaxPollInterval = 500 * time.Millisecond

// teamRunDefaultWaitTimeoutMs is used when WaitForStart is true but
// TimeoutMs is not specified by the caller.
const teamRunDefaultWaitTimeoutMs = 5_000

// teamRunConfig is the minimal subset of gogent-team-run's config.json that
// team_run needs to read after launch.
type teamRunConfig struct {
	BackgroundPID int `json:"background_pid"`
}

// handleTeamRun handles the team_run tool call.
// It launches gogent-team-run as a detached background process, registers
// a toast notification with the TUI, and optionally polls config.json for
// the background_pid written by the daemon on startup.
func handleTeamRun(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	input TeamRunInput,
	uds *UDSClient,
) (*mcpsdk.CallToolResult, TeamRunOutput, error) {
	if input.TeamDir == "" {
		return nil, TeamRunOutput{}, fmt.Errorf("team_run: team_dir is required")
	}

	// 1. Validate the team directory exists and contains config.json.
	configPath := input.TeamDir + "/config.json"
	if _, err := os.Stat(configPath); err != nil {
		return nil, TeamRunOutput{
			Success: false,
			TeamDir: input.TeamDir,
			Result:  fmt.Sprintf("config.json not found in %s", input.TeamDir),
		}, nil
	}

	// 2. Locate the gogent-team-run binary.
	binary, err := exec.LookPath("gogent-team-run")
	if err != nil {
		return nil, TeamRunOutput{
			Success: false,
			TeamDir: input.TeamDir,
			Result:  "gogent-team-run binary not found in PATH",
		}, nil
	}

	// 3. Build the command.  Use exec.Command (NOT CommandContext) because
	//    gogent-team-run is a long-lived daemon that must outlive this MCP call.
	//    exec.CommandContext would send SIGKILL when the handler's ctx is cancelled
	//    (which happens as soon as handleTeamRun returns), killing the runner
	//    mid-flight even though Setsid:true detaches it from the terminal.
	//    The daemon manages its own lifecycle via context + signal handling.
	cmd := exec.Command(binary, input.TeamDir) //nolint:gosec // binary path from LookPath, team_dir validated
	// Explicitly inherit environment so GOFORTRESS_SOCKET reaches the daemon
	// for UDS bridge agent notifications. When cmd.Env is nil Go inherits
	// os.Environ() implicitly, but making this explicit prevents silent
	// breakage if Env is set to a filtered list in the future.
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach: create new process group so signals don't cascade.
	}
	// Discard stdout/stderr; gogent-team-run writes its own logs.
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// 4. Start the subprocess (non-blocking).
	if err := cmd.Start(); err != nil {
		slog.Error("team_run: failed to start subprocess",
			"binary", binary,
			"team_dir", input.TeamDir,
			"err", err,
		)
		uds.notify(TypeToast, ToastPayload{
			Message: fmt.Sprintf("team_run failed to start: %v", err),
			Level:   "error",
		})
		return nil, TeamRunOutput{
			Success: false,
			TeamDir: input.TeamDir,
			Result:  fmt.Sprintf("failed to start gogent-team-run: %v", err),
		}, nil
	}

	pid := cmd.Process.Pid
	slog.Info("team_run: subprocess started",
		"pid", pid,
		"team_dir", input.TeamDir,
	)

	// 5. Reap the process in a background goroutine to prevent zombies.
	//    The process is detached (Setsid), so this goroutine only waits for
	//    the process table entry — it does not affect the team run itself.
	go func() {
		waitErr := cmd.Wait()
		if waitErr != nil {
			slog.Info("team_run: subprocess exited", "pid", pid, "err", waitErr)
			uds.notify(TypeToast, ToastPayload{
				Message: fmt.Sprintf("team_run (pid %d) exited: %v", pid, waitErr),
				Level:   "warn",
			})
		} else {
			slog.Info("team_run: subprocess exited cleanly", "pid", pid)
			uds.notify(TypeToast, ToastPayload{
				Message: fmt.Sprintf("team_run (pid %d) completed", pid),
				Level:   "info",
			})
		}
	}()

	// 5a. Ensure the team is visible to the TUI's team poller.
	//     The TUI polls {tuiSessionDir}/teams/ every 2s. If the team_dir
	//     lives under a different session tree (e.g. the Claude Code CLI
	//     session at ~/.claude/sessions/), the poller won't find it.
	//     We symlink into the TUI's teams dir so both systems see the team.
	ensureTeamVisible(input.TeamDir)

	// 6. Notify the TUI immediately that the team has started.
	uds.notify(TypeToast, ToastPayload{
		Message: fmt.Sprintf("team_run started (pid %d): %s", pid, input.TeamDir),
		Level:   "info",
	})
	// 6b. Send a team update so the TUI scans immediately and auto-expands
	// the teams drawer. Without this, discovery depends on the 2s poll tick
	// chain which may not have fired yet (or may have died).
	uds.notify(TypeTeamUpdate, TeamUpdatePayload{
		TeamDir: input.TeamDir,
		Status:  "running",
	})

	// 7. If the caller does not want to wait for startup verification, return
	//    immediately with the process PID.
	waitForStart := input.WaitForStart
	if !waitForStart {
		return nil, TeamRunOutput{
			Success:       true,
			TeamDir:       input.TeamDir,
			BackgroundPID: pid,
			Monitor:       "/team-status",
			Result:        fmt.Sprintf("gogent-team-run started (pid %d)", pid),
			Cancel:        "/team-cancel",
		}, nil
	}

	// 8. Poll config.json for the background_pid written by the daemon.
	//    gogent-team-run writes its background PID to config.json after
	//    forking to the background; we poll until it appears or we time out.
	timeoutMs := input.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = teamRunDefaultWaitTimeoutMs
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	delay := teamRunPollInterval
	backgroundPID := 0

	for time.Now().Before(deadline) {
		time.Sleep(delay)
		delay *= 2
		if delay > teamRunMaxPollInterval {
			delay = teamRunMaxPollInterval
		}

		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			// config.json may be mid-write; retry.
			continue
		}

		var cfg teamRunConfig
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr != nil {
			// Partial write in progress; retry.
			continue
		}

		if cfg.BackgroundPID > 0 {
			backgroundPID = cfg.BackgroundPID
			break
		}
	}

	slog.Info("team_run: startup poll complete",
		"pid", pid,
		"background_pid", backgroundPID,
		"team_dir", input.TeamDir,
	)

	return nil, TeamRunOutput{
		Success:       true,
		TeamDir:       input.TeamDir,
		BackgroundPID: backgroundPID,
		Monitor:       "/team-status",
		Result:        fmt.Sprintf("gogent-team-run started (pid %d, background_pid %d)", pid, backgroundPID),
		Cancel:        "/team-cancel",
	}, nil
}

// buildSpawnArgs constructs the claude CLI arguments for an agent spawn.
// Timeout is NOT passed as a CLI flag (not supported by claude CLI).
// The spawner manages timeout via time.AfterFunc in runSubprocess().
func buildSpawnArgs(agent *routing.Agent, input SpawnAgentInput) []string {
	// Use --output-format stream-json for spawned agents so we can parse
	// NDJSON in real-time and report live tool activity to the TUI.
	// --verbose is required for stream-json with -p (claude CLI 2.1.81+).
	// parseCLIOutput already handles NDJSON by scanning for type=="result".
	// --permission-mode bypassPermissions is required because -p (print mode)
	// has no interactive terminal to approve permissions.
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--permission-mode", "bypassPermissions"}

	// Model: prefer explicit override, fall back to agent config.
	model := input.Model
	if model == "" {
		model = agent.Model
	}
	args = append(args, "--model", model)

	// MCP config: only for interactive agents when the config path is available.
	// GOFORTRESS_MCP_CONFIG is set exclusively by the TUI (cmd/gofortress/main.go),
	// so hasMCP is true only in TUI context — never for gogent-team-run or plain CLI.
	mcpConfigPath := os.Getenv("GOFORTRESS_MCP_CONFIG")
	hasMCP := agent.Interactive && mcpConfigPath != ""
	if hasMCP {
		args = append(args, "--mcp-config", mcpConfigPath)
	}

	// Allowed tools: prefer explicit override, fall back to agent config.
	tools := input.AllowedTools
	if len(tools) == 0 {
		tools = agent.GetAllowedTools()
	}
	if hasMCP {
		// Merge MCP tool glob for interactive agents so spawned Claude can call
		// ask_user, confirm_action, spawn_agent, etc.
		tools = append(tools, "mcp__gofortress-interactive__*")
		// Block built-in equivalents — MCP provides these via the TUI bridge.
		// --disallowedTools removes them from the model's context entirely,
		// preventing the LLM from reaching for tools that don't work in -p mode.
		args = append(args, "--disallowedTools", "Task,AskUserQuestion")
	}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}

	// Apply additional CLI flags from agent config (e.g. future per-agent flags).
	// --permission-mode is filtered since buildSpawnArgs already sets bypassPermissions.
	if agent.CliFlags != nil {
		for i := 0; i < len(agent.CliFlags.AdditionalFlags); i++ {
			flag := agent.CliFlags.AdditionalFlags[i]
			if flag == "--permission-mode" && i+1 < len(agent.CliFlags.AdditionalFlags) {
				i++ // skip value — already set above
				continue
			}
			args = append(args, flag)
		}
	}

	// Optional cost ceiling.
	if input.MaxBudget > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.4f", input.MaxBudget))
	}

	return args
}

// ensureTeamVisible creates a symlink in the TUI's teams directory if the
// team_dir is outside the TUI's session tree.  This bridges the gap between
// Claude Code CLI sessions (~/.claude/sessions/) and TUI sessions
// (~/.gogent/sessions/) so the TUI's 2-second poller discovers teams created
// by the router regardless of which session system created them.
//
// Errors are logged but never returned — team visibility in the drawer is
// a nice-to-have, not a launch blocker.
func ensureTeamVisible(teamDir string) {
	// Read the TUI's current session from ~/.gogent/current-session.
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("ensureTeamVisible: cannot resolve home dir", "err", err)
		return
	}
	markerPath := filepath.Join(home, ".gogent", "current-session")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		slog.Info("ensureTeamVisible: no TUI session marker", "path", markerPath, "err", err)
		return
	}
	tuiSessionDir := strings.TrimSpace(string(data))
	if tuiSessionDir == "" {
		return
	}

	tuiTeamsDir := filepath.Join(tuiSessionDir, "teams")
	teamBase := filepath.Base(teamDir)
	symlinkPath := filepath.Join(tuiTeamsDir, teamBase)

	// If the team_dir is already inside the TUI's teams directory, nothing to do.
	if filepath.Dir(teamDir) == tuiTeamsDir {
		return
	}

	// If the symlink already exists (idempotent), skip.
	if _, err := os.Lstat(symlinkPath); err == nil {
		return
	}

	// Create the teams dir if needed and place the symlink.
	if err := os.MkdirAll(tuiTeamsDir, 0o755); err != nil {
		slog.Warn("ensureTeamVisible: cannot create TUI teams dir", "path", tuiTeamsDir, "err", err)
		return
	}
	if err := os.Symlink(teamDir, symlinkPath); err != nil {
		slog.Warn("ensureTeamVisible: symlink failed", "target", teamDir, "link", symlinkPath, "err", err)
		return
	}
	slog.Info("ensureTeamVisible: symlinked team into TUI session",
		"team_dir", teamDir,
		"symlink", symlinkPath,
	)
}
