package routing

import (
	"fmt"
	"slices"
)

// SubagentTypeValidation represents result of subagent_type check
type SubagentTypeValidation struct {
	Valid            bool
	RequestedType    string
	AllowedTypes     []string
	Agent            string
	ErrorMessage     string
}

// ValidateSubagentType checks if Task uses correct subagent_type for agent
func ValidateSubagentType(schema *Schema, targetAgent string, requestedType string, agentTaskNames map[string]string) *SubagentTypeValidation {
	result := &SubagentTypeValidation{
		Agent:         targetAgent,
		RequestedType: requestedType,
	}

	// If no agent specified, can't validate
	if targetAgent == "" {
		result.Valid = true
		return result
	}

	// Use schema method to get allowed types
	allowedTypes, err := schema.GetAllowedSubagentTypes(targetAgent)
	if err != nil {
		// Agent not in mapping, allow (might be custom agent)
		result.Valid = true
		return result
	}

	result.AllowedTypes = allowedTypes

	// Check if requested type is in allowed list
	if !slices.Contains(allowedTypes, requestedType) {
		// Also accept the agent's Task tool subagent_type name as a valid alias
		if agentTaskNames != nil {
			if taskName, ok := agentTaskNames[targetAgent]; ok && taskName == requestedType {
				result.Valid = true
				result.AllowedTypes = append(allowedTypes, taskName)
				return result
			}
		}

		result.Valid = false
		result.ErrorMessage = fmt.Sprintf(
			"[task-validation] Invalid subagent_type for agent '%s'. Allowed: %v. Requested: '%s'. Subagent_type mismatch causes wrong tool permissions. See routing-schema.json → agent_subagent_mapping.",
			targetAgent,
			allowedTypes,
			requestedType,
		)
		return result
	}

	result.Valid = true
	return result
}

// FormatSubagentTypeError creates detailed error with fix suggestion
func (v *SubagentTypeValidation) FormatSubagentTypeError() string {
	if v.Valid {
		return ""
	}

	// Suggest primary type (first in list)
	primaryType := ""
	if len(v.AllowedTypes) > 0 {
		primaryType = v.AllowedTypes[0]
	}

	return fmt.Sprintf(
		"%s\n\nFix: Change subagent_type to '%s' in Task() call.\nExample: Task({subagent_type: '%s', prompt: 'AGENT: %s\\n\\n...'})",
		v.ErrorMessage,
		primaryType,
		primaryType,
		v.Agent,
	)
}
