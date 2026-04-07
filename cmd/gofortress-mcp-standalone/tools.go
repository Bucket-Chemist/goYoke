package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	// AcceptanceCriteria is an optional list of criteria the agent must satisfy.
	// For Sonnet+ tier agents (tier >= 2) these are appended to the prompt as a
	// TodoWrite task list.  Haiku agents (tier < 2) ignore this field.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	// Background, when true, starts the subprocess and returns immediately
	// with the agentId. Use get_spawn_result to collect the result later.
	Background bool `json:"background,omitempty"`
}

// GetSpawnResultInput is the input for the get_spawn_result tool.
type GetSpawnResultInput struct {
	// SpawnID is the agentId returned by a background spawn_agent call.
	SpawnID string `json:"spawn_id"`
	// Block, when true, waits for the spawn to complete (up to Timeout).
	// When false, returns the current status immediately.
	Block bool `json:"block,omitempty"`
	// Timeout is the maximum wait time in milliseconds when Block is true.
	// Defaults to 300000 (5 minutes).
	Timeout int `json:"timeout,omitempty"`
}

// GetSpawnResultOutput is the response from get_spawn_result.
type GetSpawnResultOutput struct {
	SpawnID  string           `json:"spawn_id"`
	Status   SpawnStatus      `json:"status"`
	Result   *SpawnAgentOutput `json:"result,omitempty"` // nil while running
	Error    string           `json:"error,omitempty"`
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
// Global state
// -----------------------------------------------------------------------------

// bgStore holds results from background spawn_agent calls. It is initialised
// once at tool registration time and shared across all tool handlers.
var bgStore = NewBackgroundSpawnStore()

// -----------------------------------------------------------------------------
// Tool registration
// -----------------------------------------------------------------------------

// RegisterAll registers all tools on server.
func RegisterAll(server *mcpsdk.Server) {
	registerTestMcpPing(server)
	registerSpawnAgent(server)
	registerGetSpawnResult(server)
	registerSandboxWrite(server)
	registerSandboxStatus(server)
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

	// 7b. Merge and inject acceptance criteria for Sonnet+ tier agents.
	// Haiku agents (tier < 2) are skipped — they lack the TodoWrite tool.
	mergedAC := mergeAcceptanceCriteria(nil, input.AcceptanceCriteria)
	if len(mergedAC) > 0 && agentTierNumber(agent.Tier) >= 2 {
		var sb strings.Builder
		sb.WriteString(augmented)
		sb.WriteString("\n\n## Acceptance Criteria\n")
		sb.WriteString("You MUST use the TodoWrite tool to create a task list from these criteria and mark each as completed when done.\n")
		for _, criterion := range mergedAC {
			sb.WriteString("- [ ] ")
			sb.WriteString(criterion)
			sb.WriteString("\n")
		}
		augmented = sb.String()
		writeInitialACSidecar(agentID, mergedAC)
	} else {
		mergedAC = nil
	}

	// 8. Background mode: launch subprocess in a goroutine and return immediately.
	if input.Background {
		bgStore.Register(agentID, agent.ID)
		slog.Info("spawn_agent: background spawn started", "agent", agent.ID, "agentId", agentID)

		// Use context.Background() — the MCP request context is cancelled when
		// this handler returns, which would kill the subprocess immediately.
		go func() {
			bgCtx := context.Background()
			start := time.Now()
			result, runErr := runSubprocess(bgCtx, agent, input, augmented, agentID)
			duration := time.Since(start).Round(time.Millisecond).String()

			out := buildSpawnOutput(agentID, agent.ID, result, runErr, duration)
			bgStore.Complete(agentID, &out)
			slog.Info("spawn_agent: background spawn finished",
				"agent", agent.ID, "agentId", agentID, "success", out.Success, "duration", duration)
		}()

		return nil, SpawnAgentOutput{
			AgentID: agentID,
			Agent:   agent.ID,
			Success: true,
			Output:  fmt.Sprintf("Background spawn started. Use get_spawn_result with spawn_id=%q to collect the result.", agentID),
		}, nil
	}

	// 9. Synchronous mode (default): run subprocess and wait for completion.
	start := time.Now()
	result, runErr := runSubprocess(ctx, agent, input, augmented, agentID)
	duration := time.Since(start).Round(time.Millisecond).String()

	return nil, buildSpawnOutput(agentID, agent.ID, result, runErr, duration), nil
}

// buildSpawnOutput constructs a SpawnAgentOutput from subprocess results.
// Extracted to avoid duplicating the response-building logic between
// synchronous and background code paths.
func buildSpawnOutput(agentID, agentType string, result *cliResult, runErr error, duration string) SpawnAgentOutput {
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

	return SpawnAgentOutput{
		AgentID:   agentID,
		Agent:     agentType,
		Success:   success,
		Output:    output,
		Error:     errMsg,
		Cost:      cost,
		Turns:     turns,
		Duration:  duration,
		Truncated: truncated,
	}
}

// -----------------------------------------------------------------------------
// get_spawn_result tool
// -----------------------------------------------------------------------------

// registerGetSpawnResult registers the get_spawn_result tool.
func registerGetSpawnResult(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_spawn_result",
		Description: "Retrieve the result of a background spawn_agent call. Use block=true to wait for completion, or block=false to poll current status.",
	}, handleGetSpawnResult)
}

// handleGetSpawnResult handles the get_spawn_result tool call.
func handleGetSpawnResult(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input GetSpawnResultInput,
) (*mcpsdk.CallToolResult, GetSpawnResultOutput, error) {
	if input.SpawnID == "" {
		return nil, GetSpawnResultOutput{}, fmt.Errorf("get_spawn_result: spawn_id is required")
	}

	// Non-blocking: return current snapshot.
	if !input.Block {
		snap, ok := bgStore.Get(input.SpawnID)
		if !ok {
			return nil, GetSpawnResultOutput{
				SpawnID: input.SpawnID,
				Error:   fmt.Sprintf("unknown spawn_id: %s", input.SpawnID),
			}, nil
		}
		return nil, GetSpawnResultOutput{
			SpawnID: input.SpawnID,
			Status:  snap.Status,
			Result:  snap.Result,
		}, nil
	}

	// Blocking: wait for completion.
	timeoutMS := defaultTimeoutMS
	if input.Timeout > 0 {
		timeoutMS = input.Timeout
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond

	result, err := bgStore.Wait(input.SpawnID, timeout)
	if err != nil {
		return nil, GetSpawnResultOutput{
			SpawnID: input.SpawnID,
			Status:  SpawnStatusRunning,
			Error:   err.Error(),
		}, nil
	}

	// Determine final status from the store snapshot.
	status := SpawnStatusCompleted
	snap, ok := bgStore.Get(input.SpawnID)
	if ok {
		status = snap.Status
	}

	return nil, GetSpawnResultOutput{
		SpawnID: input.SpawnID,
		Status:  status,
		Result:  result,
	}, nil
}

// -----------------------------------------------------------------------------
// Acceptance criteria helpers
// -----------------------------------------------------------------------------

// acSidecarEntry mirrors state.AcceptanceCriterion for the standalone sidecar
// file format. Using a local type avoids importing the TUI internal package.
type acSidecarEntry struct {
	Text      string `json:"Text"`
	Completed bool   `json:"Completed"`
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

// writeInitialACSidecar writes the initial AC state (all Completed=false) to
// SESSION_DIR/ac/{agentID}.json.  Called immediately after AC injection so
// downstream tools can observe the criteria before the agent runs.
func writeInitialACSidecar(agentID string, criteria []string) {
	sessionDir := os.Getenv("GOGENT_SESSION_DIR")
	if sessionDir == "" {
		slog.Warn("writeInitialACSidecar: GOGENT_SESSION_DIR not set, skipping sidecar write",
			"agentID", agentID)
		return
	}

	acDir := filepath.Join(sessionDir, "ac")
	if err := os.MkdirAll(acDir, 0o755); err != nil {
		slog.Warn("writeInitialACSidecar: failed to create ac dir", "err", err, "dir", acDir)
		return
	}

	entries := make([]acSidecarEntry, len(criteria))
	for i, c := range criteria {
		entries[i] = acSidecarEntry{Text: c, Completed: false}
	}

	data, err := json.Marshal(entries)
	if err != nil {
		slog.Warn("writeInitialACSidecar: failed to marshal AC state", "err", err, "agentID", agentID)
		return
	}

	sidecarPath := filepath.Join(acDir, agentID+".json")
	if err := os.WriteFile(sidecarPath, data, 0o644); err != nil {
		slog.Warn("writeInitialACSidecar: failed to write sidecar", "err", err, "path", sidecarPath)
	}
}
