package routing

import (
	"encoding/json"
	"fmt"
)

// NewValidationOrchestrator creates orchestrator with all dependencies loaded
func NewValidationOrchestrator(schema *Schema, projectDir string, agentsIndex *AgentsIndex, agentTaskNames map[string]string) *ValidationOrchestrator {
	return &ValidationOrchestrator{
		Schema:         schema,
		ProjectDir:     projectDir,
		AgentsIndex:    agentsIndex,
		AgentTaskNames: agentTaskNames,
	}
}

// ValidationOrchestrator coordinates all Task validation checks
type ValidationOrchestrator struct {
	Schema         *Schema
	ProjectDir     string
	AgentsIndex    *AgentsIndex
	AgentTaskNames map[string]string
}

// ValidationResult combines all validation outcomes
type ValidationResult struct {
	Decision            string                  `json:"decision"` // "allow" or "block"
	Reason              string                  `json:"reason,omitempty"`
	EinsteinBlocked     *TaskValidationResult   `json:"einstein_blocked,omitempty"`
	ModelMismatch       string                  `json:"model_mismatch,omitempty"`
	CeilingViolation    string                  `json:"ceiling_violation,omitempty"`
	SubagentTypeInvalid *SubagentTypeValidation `json:"subagent_type_invalid,omitempty"`
	Violations          []*Violation            `json:"violations,omitempty"`
}

// ValidateTask runs all validation checks on Task invocation
func (v *ValidationOrchestrator) ValidateTask(taskInput map[string]interface{}, sessionID string) *ValidationResult {
	result := &ValidationResult{
		Decision: "allow",
	}

	// Extract fields
	model, _ := taskInput["model"].(string)
	prompt, _ := taskInput["prompt"].(string)
	subagentType, _ := taskInput["subagent_type"].(string)
	resume, _ := taskInput["resume"].(string)
	targetAgent := extractAgentFromPrompt(prompt)

	// Check 1: Einstein/Opus blocking (with allowlist)
	einsteinCheck := ValidateTaskInvocation(v.Schema, taskInput, sessionID)
	if !einsteinCheck.Allowed {
		result.Decision = "block"
		result.Reason = einsteinCheck.BlockReason
		result.EinsteinBlocked = einsteinCheck
		if einsteinCheck.Violation != nil {
			result.Violations = append(result.Violations, einsteinCheck.Violation)
		}
		return result // Hard block, no further checks
	}

	// Determine if agent is in opus allowlist (for ceiling bypass)
	opusAllowlisted := false
	if opusConfig, exists := v.Schema.Tiers["opus"]; exists {
		opusAllowlisted = isInAllowlist(targetAgent, opusConfig.TaskInvocationAllowlist)
	}

	// Check 2: Model mismatch (warning only, not blocking)
	if v.AgentsIndex != nil && targetAgent != "" {
		if agentConfig, exists := v.AgentsIndex.Agents[targetAgent]; exists {
			matches, warning := ValidateModelMatch(targetAgent, &agentConfig, model)
			if !matches {
				result.ModelMismatch = warning
				// Don't block, just warn
			}
		}
	}

	// Check 3: Delegation ceiling
	// SKIP ceiling check if agent is in opus allowlist or resuming a previous agent
	if !opusAllowlisted && resume == "" {
		ceiling, err := LoadDelegationCeiling(v.ProjectDir)
		if err == nil && ceiling != nil {
			allowed, ceilingMsg := CheckDelegationCeiling(v.Schema, ceiling, model)
			if !allowed {
				result.Decision = "block"
				result.Reason = ceilingMsg
				result.CeilingViolation = ceilingMsg

				// Log violation
				violation := &Violation{
					SessionID:     sessionID,
					ViolationType: "delegation_ceiling",
					Model:         model,
					Agent:         targetAgent,
					Reason:        fmt.Sprintf("Ceiling: %s, Requested: %s", ceiling.MaxTier, model),
				}
				result.Violations = append(result.Violations, violation)
				return result // Hard block
			}
		}
	}

	// Check 4: Subagent_type validation (skip for resume — original spawn was validated)
	if resume != "" {
		return result
	}
	subagentCheck := ValidateSubagentType(v.Schema, targetAgent, subagentType, v.AgentTaskNames)
	if !subagentCheck.Valid {
		result.Decision = "block"
		result.Reason = subagentCheck.FormatSubagentTypeError()
		result.SubagentTypeInvalid = subagentCheck

		// Log violation
		violation := &Violation{
			SessionID:     sessionID,
			ViolationType: "subagent_type_mismatch",
			Agent:         targetAgent,
			Reason:        fmt.Sprintf("Allowed: %v, Requested: %s", subagentCheck.AllowedTypes, subagentCheck.RequestedType),
		}
		result.Violations = append(result.Violations, violation)
		return result // Hard block
	}

	return result
}

// ToJSON serializes validation result to JSON
func (v *ValidationResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
