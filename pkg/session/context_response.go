package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContextComponents holds all context pieces for session initialization
type ContextComponents struct {
	SessionType      string                  // "startup" or "resume"
	RoutingSummary   string                  // From schema.FormatTierSummary()
	HandoffSummary   string                  // From LoadHandoffSummary() - resume only
	PendingLearnings string                  // From CheckPendingLearnings()
	GitInfo          string                  // From FormatGitInfo()
	ProjectInfo      *ProjectDetectionResult // From DetectProjectType()
	SessionDir       string                  // From CreateSessionDir() - absolute path
}

// SessionStartResponse is the hook output format for SessionStart
type SessionStartResponse struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// HookSpecificOutput contains the context injection payload
type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// GenerateSessionStartResponse creates the complete context injection response.
// Output follows Claude Code hook response format.
func GenerateSessionStartResponse(ctx *ContextComponents) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("[context-response] ContextComponents nil. Cannot generate response without context.")
	}

	var contextParts []string

	// Session header
	header := fmt.Sprintf("🚀 SESSION INITIALIZED (%s)", ctx.SessionType)
	contextParts = append(contextParts, header)

	// Routing summary (always include)
	if ctx.RoutingSummary != "" {
		contextParts = append(contextParts, ctx.RoutingSummary)
	}

	// Handoff for resume sessions only
	if ctx.SessionType == "resume" && ctx.HandoffSummary != "" {
		contextParts = append(contextParts, "PREVIOUS SESSION HANDOFF:\n"+ctx.HandoffSummary)
	}

	// Pending learnings warning (if any)
	if ctx.PendingLearnings != "" {
		contextParts = append(contextParts, ctx.PendingLearnings)
	}

	// Git info (if in git repo)
	if ctx.GitInfo != "" {
		contextParts = append(contextParts, ctx.GitInfo)
	}

	// Project type detection
	if ctx.ProjectInfo != nil {
		contextParts = append(contextParts, FormatProjectType(ctx.ProjectInfo))
	}

	// Session directory
	if ctx.SessionDir != "" {
		sessionInfo := fmt.Sprintf("SESSION_DIR: %s\nAll session artifacts are written to this directory. .goyoke/tmp/ symlinks here.", ctx.SessionDir)
		contextParts = append(contextParts, sessionInfo)
	}

	// Hook status footer
	contextParts = append(contextParts, "Routing hooks are ACTIVE. Tool usage validated against routing-schema.json.")

	// Combine all parts
	fullContext := strings.Join(contextParts, "\n\n")

	// Build response
	response := SessionStartResponse{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: fullContext,
		},
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("[context-response] Failed to marshal response: %w", err)
	}

	return string(data), nil
}

// GenerateErrorResponse creates an error response in hook format.
// Errors are displayed but don't block session start.
func GenerateErrorResponse(message string) string {
	response := SessionStartResponse{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: fmt.Sprintf("🔴 SESSION START ERROR: %s\n\nSession continues but context injection failed.", message),
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}
