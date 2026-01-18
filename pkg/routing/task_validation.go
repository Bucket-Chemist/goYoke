package routing

import (
	"fmt"
	"regexp"
	"slices"
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

	// Extract model and prompt
	model, _ := taskInput["model"].(string)
	prompt, _ := taskInput["prompt"].(string)

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

	// Block 1: Model is opus (regardless of target agent)
	if model == "opus" {
		result.Allowed = false
		result.BlockReason = "Task(model: opus) causes 60K token inheritance ($3.30 cost). Use /einstein slash command instead ($0.92 cost)."
		result.Recommendation = "Generate GAP document to .claude/tmp/einstein-gap-{timestamp}.md, then notify user to run /einstein. See GAP-003b for rationale."

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "blocked_task_opus",
			Model:         "opus",
			Agent:         targetAgent,
			Reason:        "model_is_opus",
		}

		return result
	}

	// Block 2: Target agent is einstein (regardless of model specified)
	if targetAgent == "einstein" {
		result.Allowed = false
		result.BlockReason = fmt.Sprintf("Einstein must be invoked via /einstein slash command, not Task tool (even with model: %s). Task tool causes 60K token inheritance.", model)
		result.Recommendation = "Generate GAP document, then notify user to run /einstein."

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "blocked_task_einstein",
			Model:         model,
			Agent:         "einstein",
			Reason:        "agent_is_einstein",
		}

		return result
	}

	return result
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
	Model         string   `json:"model"`
	SubagentType  string   `json:"subagent_type"`
	AllowedModels []string `json:"allowed_models,omitempty"`
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
