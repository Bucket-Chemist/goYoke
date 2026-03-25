package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/google/uuid"

	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

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

// SpawnAgentInput is the input for the spawn_agent tool.
type SpawnAgentInput struct {
	// Agent is the agent ID from agents-index.json. Required.
	Agent string `json:"agent"`
	// Description is a brief human-readable description logged for observability.
	Description string `json:"description"`
	// Prompt is the task prompt sent to the agent. Required.
	Prompt string `json:"prompt"`
	// Model overrides the agent's default model.
	Model string `json:"model,omitempty"`
	// Timeout is the deadline in milliseconds (default: 300000).
	Timeout int `json:"timeout,omitempty"`
	// AllowedTools overrides the agent's cli_flags.allowed_tools list.
	AllowedTools []string `json:"allowedTools,omitempty"`
	// MaxBudget is a soft cost ceiling in USD.
	MaxBudget float64 `json:"maxBudget,omitempty"`
	// CallerType self-identifies the spawning agent for validation.
	CallerType string `json:"caller_type,omitempty"`
}

// SpawnAgentOutput is the response from spawn_agent.
// Note: Truncated is a field unique to the standalone binary.
type SpawnAgentOutput struct {
	AgentID   string  `json:"agentId"`
	Agent     string  `json:"agent"`
	Success   bool    `json:"success"`
	Output    string  `json:"output"`
	Error     string  `json:"error,omitempty"`
	Cost      float64 `json:"cost"`
	Turns     int     `json:"turns"`
	Duration  string  `json:"duration"`
	Truncated bool    `json:"truncated,omitempty"`
}

// -----------------------------------------------------------------------------
// Tool registration
// -----------------------------------------------------------------------------

// RegisterAll registers all tools on server.
func RegisterAll(server *mcpsdk.Server) {
	registerTestMcpPing(server)
	registerSpawnAgent(server)
}

// registerTestMcpPing registers the test_mcp_ping tool.
func registerTestMcpPing(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "test_mcp_ping",
		Description: "Returns a PONG response with the current timestamp. Used for MCP connectivity validation.",
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

// registerSpawnAgent registers the spawn_agent tool.
func registerSpawnAgent(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "spawn_agent",
		Description: "Spawn a GOgent-Fortress subagent by ID. Validates agent config, injects identity context, and runs the claude CLI subprocess. Returns output and cost telemetry.",
	}, handleSpawnAgent)
}

// handleSpawnAgent handles the spawn_agent tool call.
// It validates the request, loads agent config, injects identity context, and
// runs a claude CLI subprocess. All subprocess errors are returned as soft
// errors in SpawnAgentOutput (not as Go errors) so the MCP caller can inspect
// them without a protocol-level failure.
func handleSpawnAgent(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	input SpawnAgentInput,
) (*mcpsdk.CallToolResult, SpawnAgentOutput, error) {
	// 1. Validate nesting depth before doing any real work.
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

	// 3. Load agent index (cached to avoid redundant disk reads on parallel spawns).
	index, err := routing.LoadAgentsIndexCached()
	if err != nil {
		return nil, SpawnAgentOutput{}, fmt.Errorf("spawn_agent: load agent index: %w", err)
	}

	// 4. Verify the requested agent exists.
	agent, err := index.GetAgentByID(input.Agent)
	if err != nil {
		return nil, SpawnAgentOutput{
			Agent:   input.Agent,
			Success: false,
			Error:   fmt.Sprintf("unknown agent: %s", input.Agent),
		}, nil
	}

	// 5. Validate the spawned_by / can_spawn relationship.
	// parentType comes from the GOGENT_PARENT_AGENT env var (set by a prior
	// spawn_agent invocation) and is considered verified. callerType is
	// self-reported by the MCP caller and is treated as claimed/unverified.
	parentType := os.Getenv("GOGENT_PARENT_AGENT")
	vr := validateRelationship(index, parentType, input.Agent, input.CallerType)
	if !vr.Valid {
		return nil, SpawnAgentOutput{
			Agent:   input.Agent,
			Success: false,
			Error:   "spawn validation failed: " + strings.Join(vr.Errors, "; "),
		}, nil
	}
	if len(vr.Warnings) > 0 {
		slog.Warn("spawn_agent relationship warnings", "warnings", vr.Warnings, "agent", input.Agent)
	}

	// 6. Generate a unique agent instance ID for this invocation.
	agentID := uuid.New().String()

	// 7. Build the augmented prompt with agent identity and context.
	augmented, err := routing.BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, input.Prompt)
	if err != nil {
		slog.Warn("failed to build agent context", "err", err, "agent", agent.ID)
		augmented = input.Prompt
	}

	// 8. Run the subprocess.
	start := time.Now()
	result, runErr := runSubprocess(ctx, agent, input, augmented, agentID)
	duration := time.Since(start).Round(time.Millisecond).String()

	// 9. Build response — subprocess errors are soft (returned in output, not propagated).
	success := runErr == nil
	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
	}

	output := ""
	cost := 0.0
	turns := 0
	truncated := false
	if result != nil {
		output = result.Result
		cost = result.TotalCostUSD
		turns = result.NumTurns
		truncated = result.Truncated
	}

	return nil, SpawnAgentOutput{
		AgentID:   agentID,
		Agent:     agent.ID,
		Success:   success,
		Output:    output,
		Error:     errMsg,
		Cost:      cost,
		Turns:     turns,
		Duration:  duration,
		Truncated: truncated,
	}, nil
}
