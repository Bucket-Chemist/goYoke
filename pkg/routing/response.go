package routing

import (
	"encoding/json"
	"fmt"
	"io"
)

// Decision values for hook responses
// Claude Code expects: "approve" | "block" (optional)
const (
	DecisionApprove = "approve" // Allow the tool to proceed
	DecisionBlock   = "block"   // Block the tool from executing

	// Legacy aliases (deprecated - use DecisionApprove/DecisionBlock)
	DecisionWarn = "approve" // Mapped to approve (warn not supported by Claude Code)
	DecisionPass = "approve" // Mapped to approve (pass not supported by Claude Code)
)

// HookResponse represents the JSON response structure for hooks.
// This is the canonical response format that all hooks must emit.
// Empty Decision/Reason fields are omitted from JSON output to avoid schema violations.
type HookResponse struct {
	Decision           string                 `json:"decision,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	HookSpecificOutput map[string]interface{} `json:"hookSpecificOutput"`
}

// NewBlockResponse creates a HookResponse with decision="block".
// hookEventName is automatically populated in hookSpecificOutput.
func NewBlockResponse(hookEventName, reason string) *HookResponse {
	return &HookResponse{
		Decision: DecisionBlock,
		Reason:   reason,
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": hookEventName,
		},
	}
}

// NewWarnResponse creates a HookResponse with decision="warn".
// hookEventName is automatically populated in hookSpecificOutput.
func NewWarnResponse(hookEventName, reason string) *HookResponse {
	return &HookResponse{
		Decision: DecisionWarn,
		Reason:   reason,
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": hookEventName,
		},
	}
}

// NewPassResponse creates a HookResponse with no decision/reason fields.
// This is used for context-only responses like additionalContext injection.
func NewPassResponse(hookEventName string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": hookEventName,
		},
	}
}

// NewModifyResponse creates a HookResponse that modifies the tool input.
// This is used by PreToolUse hooks to inject conventions into Task prompts.
// The updatedInput map should contain the complete modified tool input.
// Decision is omitted (not required for modify responses per Claude Code schema).
func NewModifyResponse(hookEventName string, updatedInput map[string]interface{}) *HookResponse {
	return &HookResponse{
		// Decision omitted - not required for modify responses
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": hookEventName,
			"updatedInput":  updatedInput,
		},
	}
}

// AddField adds a custom field to hookSpecificOutput.
// This allows hooks to include tool-specific data in the response.
func (r *HookResponse) AddField(key string, value interface{}) {
	if r.HookSpecificOutput == nil {
		r.HookSpecificOutput = make(map[string]interface{})
	}
	r.HookSpecificOutput[key] = value
}

// SetDecision sets the decision field of the HookResponse.
// Use the Decision* constants for valid values.
func (r *HookResponse) SetDecision(decision string) {
	r.Decision = decision
}

// GetDecision retrieves the decision field from the HookResponse.
func (r *HookResponse) GetDecision() string {
	return r.Decision
}

// HasUpdatedInput returns true if this response modifies the tool input.
func (r *HookResponse) HasUpdatedInput() bool {
	if r.HookSpecificOutput == nil {
		return false
	}
	_, ok := r.HookSpecificOutput["updatedInput"]
	return ok
}

// GetUpdatedInput returns the updatedInput map if present, nil otherwise.
func (r *HookResponse) GetUpdatedInput() map[string]interface{} {
	if r.HookSpecificOutput == nil {
		return nil
	}
	if ui, ok := r.HookSpecificOutput["updatedInput"].(map[string]interface{}); ok {
		return ui
	}
	return nil
}

// Validate checks that the HookResponse has valid decision values and required fields.
// Returns an error if validation fails.
func (r *HookResponse) Validate() error {
	// Validate decision if present
	// Claude Code expects: "approve" | "block" (optional)
	if r.Decision != "" {
		if r.Decision != DecisionApprove && r.Decision != DecisionBlock {
			return fmt.Errorf(
				"[hook-response] Invalid decision value %q. Must be %q or %q (Claude Code schema). Use DecisionApprove or DecisionBlock constants.",
				r.Decision,
				DecisionApprove,
				DecisionBlock,
			)
		}
	}

	// If decision is block or warn, reason is required
	if (r.Decision == DecisionBlock || r.Decision == DecisionWarn) && r.Reason == "" {
		return fmt.Errorf(
			"[hook-response] Decision %q requires non-empty reason field. Provide context for the decision.",
			r.Decision,
		)
	}

	// hookEventName is required in hookSpecificOutput
	if r.HookSpecificOutput == nil {
		return fmt.Errorf(
			"[hook-response] Missing hookSpecificOutput. All hook responses must include hookEventName in hookSpecificOutput.",
		)
	}

	hookEventName, ok := r.HookSpecificOutput["hookEventName"]
	if !ok {
		return fmt.Errorf(
			"[hook-response] Missing hookEventName in hookSpecificOutput. All hook responses must identify the triggering event.",
		)
	}

	if hookEventNameStr, ok := hookEventName.(string); !ok || hookEventNameStr == "" {
		return fmt.Errorf(
			"[hook-response] hookEventName must be a non-empty string. Got: %v (type: %T). Ensure hookEventName is populated correctly.",
			hookEventName,
			hookEventName,
		)
	}

	return nil
}

// Marshal writes the HookResponse as indented JSON to the provided writer.
// Returns an error if JSON marshaling or writing fails.
func (r *HookResponse) Marshal(w io.Writer) error {
	// Validate before marshaling
	if err := r.Validate(); err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"[hook-response] Failed to marshal JSON: %w. Response may contain non-serializable types.",
			err,
		)
	}

	// Write to output
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf(
			"[hook-response] Failed to write JSON output: %w. Check output destination is writable.",
			err,
		)
	}

	return nil
}
