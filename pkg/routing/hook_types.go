package routing

import "fmt"

// PreToolUseInput matches hook-io-schema.json#/definitions/PreToolUseInput
type PreToolUseInput struct {
	HookEventName string                 `json:"hook_event_name"`
	SessionID     string                 `json:"session_id"`
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
}

// PreToolUseOutput matches hook-io-schema.json#/definitions/PreToolUseOutput
type PreToolUseOutput struct {
	Decision           string                `json:"decision"` // "allow", "block", "modify"
	Reason             string                `json:"reason,omitempty"`
	HookSpecificOutput *PreToolUseHookOutput `json:"hookSpecificOutput,omitempty"`
}

type PreToolUseHookOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"` // "allow", "deny"
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	NestingLevel             int    `json:"nestingLevel,omitempty"`
	Suggestion               string `json:"suggestion,omitempty"`
}

// ValidateHookOutput validates output against schema (lightweight check)
func ValidateHookOutput(output interface{}) error {
	m, ok := output.(map[string]interface{})
	if !ok {
		return fmt.Errorf("hook output must be JSON object")
	}
	decision, ok := m["decision"].(string)
	if !ok {
		return fmt.Errorf("hook output missing required 'decision' field")
	}
	if decision != "allow" && decision != "block" && decision != "modify" {
		return fmt.Errorf("invalid decision: %s (must be allow/block/modify)", decision)
	}
	return nil
}
