package routing

import (
	"fmt"
	"strings"
)

// ToolPermission represents the result of a tool permission check.
type ToolPermission struct {
	Allowed         bool
	CurrentTier     string
	Tool            string
	AllowedTools    []string
	RecommendedTier string
}

// CheckToolPermission validates if a tool is allowed for the current tier.
// It checks the schema's tier configuration and returns detailed permission info.
//
// Parameters:
//   - schema: The routing schema containing tier configurations
//   - currentTier: The tier to check permissions for (e.g., "haiku", "sonnet")
//   - toolName: The tool being requested (e.g., "Read", "Write", "Task")
//
// Returns:
//   - ToolPermission with Allowed=true if tool is permitted
//   - ToolPermission with Allowed=false and RecommendedTier if denied
func CheckToolPermission(schema *Schema, currentTier string, toolName string) *ToolPermission {
	result := &ToolPermission{
		CurrentTier: currentTier,
		Tool:        toolName,
	}

	// Get tier config
	tierConfig, exists := schema.Tiers[currentTier]
	if !exists {
		result.Allowed = false
		result.RecommendedTier = "unknown"
		return result
	}

	// Get allowed tools for this tier
	allowedTools := tierConfig.Tools
	result.AllowedTools = allowedTools

	// Check for wildcard (["*"] means all tools allowed)
	if len(allowedTools) == 1 && allowedTools[0] == "*" {
		result.Allowed = true
		return result
	}

	// Check if tool is in allowed list
	for _, allowed := range allowedTools {
		if allowed == toolName {
			result.Allowed = true
			return result
		}
	}

	// Tool not allowed - find which tier does allow it
	result.Allowed = false
	result.RecommendedTier = findTierForTool(schema, toolName)

	return result
}

// findTierForTool searches the schema for the lowest tier that allows the specified tool.
// It checks tiers in order: haiku → haiku_thinking → sonnet → opus → external.
//
// Returns the tier name if found, or "unknown" if no tier allows the tool.
func findTierForTool(schema *Schema, toolName string) string {
	// Check tiers in ascending order (lowest to highest capability)
	tierOrder := []string{"haiku", "haiku_thinking", "sonnet", "opus", "external"}

	for _, tier := range tierOrder {
		tierConfig, exists := schema.Tiers[tier]
		if !exists {
			continue
		}

		// Check for wildcard
		if len(tierConfig.Tools) == 1 && tierConfig.Tools[0] == "*" {
			return tier
		}

		// Check if tool is in this tier's allowed list
		for _, tool := range tierConfig.Tools {
			if tool == toolName {
				return tier
			}
		}
	}

	return "unknown"
}

// FormatPermissionError creates a formatted error message following the standard:
// "[component] What. Why. How to fix."
//
// The error message includes:
//   - Tool name and current tier
//   - List of allowed tools for current tier
//   - Recommended tier that allows the tool
//   - Override suggestion using --force-tier flag
func (p *ToolPermission) FormatPermissionError() string {
	allowedStr := strings.Join(p.AllowedTools, ", ")

	return fmt.Sprintf(
		"[routing] Tool '%s' not permitted at tier '%s'. Allowed tools for %s: [%s]. Tool '%s' requires tier: %s. Use --force-tier=%s to override.",
		p.Tool,
		p.CurrentTier,
		p.CurrentTier,
		allowedStr,
		p.Tool,
		p.RecommendedTier,
		p.RecommendedTier,
	)
}
