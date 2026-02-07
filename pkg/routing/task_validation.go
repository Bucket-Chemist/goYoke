package routing

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
)

// TaskValidationResult represents result of Task tool validation
type TaskValidationResult struct {
	Allowed        bool
	BlockReason    string
	Violation      *Violation
	Recommendation string
}

// ValidateTaskInvocation checks if Task tool usage is allowed
func ValidateTaskInvocation(schema *Schema, taskInput map[string]interface{}, sessionID string) *TaskValidationResult {
	result := &TaskValidationResult{Allowed: true}

	// Extract model, prompt, and resume
	model, _ := taskInput["model"].(string)
	prompt, _ := taskInput["prompt"].(string)
	resume, _ := taskInput["resume"].(string)

	// Extract target agent from prompt (pattern: "AGENT: agent-id")
	targetAgent := extractAgentFromPrompt(prompt)

	// Check if opus invocations are blocked
	opusConfig, exists := schema.Tiers["opus"]
	if !exists {
		return result // No opus config, allow
	}

	taskBlocked := opusConfig.TaskInvocationBlocked
	if !taskBlocked {
		return result // Blocking not enabled, allow
	}

	// Check 1: If model is opus, check if agent is in the allowlist
	// Einstein and other opus-tier agents are now in the allowlist and callable via Task(model: opus)
	if model == "opus" {
		// Allow if agent is in the allowlist (e.g., planner, architect, staff-architect-critical-review)
		if isInAllowlist(targetAgent, opusConfig.TaskInvocationAllowlist) {
			return result // Allowed via allowlist
		}

		// Allow resume calls: the original spawn was already validated.
		// Resume prompts lack the "AGENT:" preamble so agent extraction returns "".
		if resume != "" {
			return result // Allowed via resume bypass
		}

		// Block: opus model requested but agent not in allowlist
		result.Allowed = false
		result.BlockReason = fmt.Sprintf("Task(model: opus) blocked for agent '%s'. Agent not in allowlist. Allowlisted agents: %v. For standalone deep analysis, use /einstein.", targetAgent, opusConfig.TaskInvocationAllowlist)
		result.Recommendation = "Either use an allowlisted agent (planner, architect, staff-architect-critical-review) or generate GAP document and run /einstein."

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "blocked_task_opus",
			Model:         "opus",
			Agent:         targetAgent,
			Reason:        "model_is_opus_agent_not_allowlisted",
		}

		return result
	}

	// Check 3: If agent is in opus allowlist but model is NOT opus, block and require opus
	// This ensures opus-tier agents (staff-architect, architect, planner, python-architect)
	// always run at their intended tier, regardless of how they're invoked.
	if isInAllowlist(targetAgent, opusConfig.TaskInvocationAllowlist) && model != "opus" {
		result.Allowed = false
		result.BlockReason = fmt.Sprintf(
			"Agent '%s' requires model: opus (currently: %s). This agent is opus-tier and must run at full capability. Add model: \"opus\" to your Task() call.",
			targetAgent,
			model,
		)
		result.Recommendation = fmt.Sprintf(
			"Change Task invocation to include model: \"opus\". Example: Task({model: \"opus\", subagent_type: \"Plan\", prompt: \"AGENT: %s\\n...\"})",
			targetAgent,
		)

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "opus_agent_wrong_model",
			Model:         model,
			Agent:         targetAgent,
			Reason:        fmt.Sprintf("agent_requires_opus_got_%s", model),
		}

		return result
	}

	return result
}

// isInAllowlist checks if an agent is in the opus Task invocation allowlist.
// Returns false if agent is empty or allowlist is empty/nil.
func isInAllowlist(agent string, allowlist []string) bool {
	if agent == "" || len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		if agent == allowed {
			return true
		}
	}
	return false
}

// extractAgentFromPrompt finds "AGENT: agent-id" pattern in prompt
func extractAgentFromPrompt(prompt string) string {
	re := regexp.MustCompile(`AGENT:\s*([a-z-]+)`)
	matches := re.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// AgentConfig represents agent metadata from agents-index.json
type AgentConfig struct {
	Name                string               `json:"name"`
	Model               string               `json:"model"`
	SubagentType        string               `json:"subagent_type"`
	AllowedModels       []string             `json:"allowed_models,omitempty"`
	ContextRequirements *ContextRequirements `json:"context_requirements,omitempty"`
	CliFlags            *AgentCliFlags       `json:"cli_flags,omitempty"`
}

// AgentsIndex represents the full agents-index.json structure
type AgentsIndex struct {
	Agents map[string]AgentConfig `json:"agents"`
}

// ValidateModelMatch checks if Task model matches agent's expected model
// Warning messages are logged to violations.jsonl with type "model_mismatch_warning"
// and included in CLI output's additionalContext field
func ValidateModelMatch(agentName string, agentConfig *AgentConfig, requestedModel string) (bool, string) {
	// If agent specifies allowed_models, check against that list
	if len(agentConfig.AllowedModels) > 0 {
		if slices.Contains(agentConfig.AllowedModels, requestedModel) {
			return true, ""
		}

		return false, fmt.Sprintf(
			"[task-validation] Model mismatch. Agent expects models: %v. Requested: %s. This may cause unexpected behavior.",
			agentConfig.AllowedModels,
			requestedModel,
		)
	}

	// Otherwise check against single model field
	if agentConfig.Model != requestedModel {
		return false, fmt.Sprintf(
			"[task-validation] Model mismatch. Agent '%s' expects model '%s'. Requested: '%s'. This may cause suboptimal performance.",
			agentName,
			agentConfig.Model,
			requestedModel,
		)
	}

	return true, ""
}

const (
	// MaxNestingDepth prevents runaway nesting
	MaxNestingDepth = 10

	// DefaultNestingLevel for fail-closed behavior
	DefaultNestingLevel = 1
)

// GetNestingLevel returns the current nesting level from environment.
// If GOGENT_NESTING_LEVEL is unset, checks CLAUDE_CODE_NESTING_LEVEL to
// distinguish root sessions (level 0) from unknown contexts (fail-closed to 1).
func GetNestingLevel() int {
	levelStr := os.Getenv("GOGENT_NESTING_LEVEL")

	if levelStr == "" {
		// If Claude Code's own nesting level is also unset or "0", we're at root
		claudeLevel := os.Getenv("CLAUDE_CODE_NESTING_LEVEL")
		if claudeLevel == "" || claudeLevel == "0" {
			return 0
		}
		// Claude says nested but GOGENT not set = fail-closed
		return DefaultNestingLevel
	}

	level, err := strconv.Atoi(levelStr)

	// Invalid = fail-closed
	if err != nil {
		return DefaultNestingLevel
	}

	// Out of range = fail-closed
	if level < 0 || level > MaxNestingDepth {
		return DefaultNestingLevel
	}

	return level
}

// IsNestingLevelExplicit returns true if GOGENT_NESTING_LEVEL was set explicitly.
// Used for telemetry to distinguish real Level 0 from assumed nesting.
func IsNestingLevelExplicit() bool {
	return os.Getenv("GOGENT_NESTING_LEVEL") != ""
}

// ValidateTaskNestingLevel checks if Task() is allowed at current nesting level.
// Returns nil if allowed, error with guidance if blocked.
func ValidateTaskNestingLevel() error {
	level := GetNestingLevel()

	if level > 0 {
		return &NestingLevelError{
			Level:   level,
			Message: fmt.Sprintf(
				"Task() blocked at nesting level %d. "+
					"Subagents cannot spawn sub-subagents via Task(). "+
					"Use MCP spawn_agent tool instead: "+
					"mcp__gofortress__spawn_agent({agent: '...', prompt: '...'})",
				level,
			),
		}
	}

	return nil
}

// NestingLevelError represents a Task() blocked due to nesting level.
type NestingLevelError struct {
	Level   int
	Message string
}

func (e *NestingLevelError) Error() string {
	return e.Message
}

// ValidateTaskAtNestingLevel checks if a Task() call is allowed at the given nesting level.
// Returns nil if allowed, or a block response map if blocked.
//
// At nesting level 0 (Router), all models are allowed.
// At nesting level 1+ (sub-agents), only haiku and sonnet are allowed.
// Task(opus) is blocked at Level 1+ to prevent expensive delegation chains.
func ValidateTaskAtNestingLevel(nestingLevel int, toolInput map[string]interface{}) map[string]interface{} {
	// Level 0 (Router) can spawn anything
	if nestingLevel == 0 {
		return nil
	}

	// Parse task input to extract model
	taskInput, err := ParseTaskInput(toolInput)
	if err != nil {
		// Cannot parse - block defensively
		return BlockResponseForNesting(nestingLevel, "unknown", "failed to parse task input")
	}

	// Check if model is opus (or empty, which defaults to opus behavior)
	if taskInput.Model == "opus" {
		return BlockResponseForNesting(nestingLevel, "opus", "opus model explicitly requested")
	}

	// Empty model field is ambiguous - block defensively
	if taskInput.Model == "" {
		return BlockResponseForNesting(nestingLevel, "", "model not specified")
	}

	// haiku and sonnet allowed at Level 1+
	return nil
}

// BlockResponseForNesting creates the standard block response for nesting violations.
// This is specifically for blocking Task(opus) at nesting levels 1+.
func BlockResponseForNesting(level int, model, reason string) map[string]interface{} {
	message := fmt.Sprintf(
		"Task(opus) blocked at nesting level %d. Only the Router (Level 0) can spawn Opus agents. Use Task(haiku) or Task(sonnet) for delegation.",
		level,
	)

	return map[string]interface{}{
		"decision": "block",
		"reason":   message,
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "opus_blocked_at_nesting_level",
			"nestingLevel":             level,
			"model":                    model,
			"blockReason":              reason,
			"suggestion":               "Use Task(haiku) or Task(sonnet) for delegation, or mcp__gofortress__spawn_agent for MCP spawning.",
		},
	}
}
