package enforcement

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// GuardResponse represents the structured output for orchestrator-guard hook.
// This structure is serialized to JSON and returned to Claude Code for blocking/allowing decisions.
type GuardResponse struct {
	HookEventName     string   `json:"hookEventName"`
	Decision          string   `json:"decision"` // "allow" or "block"
	Reason            string   `json:"reason"`
	AdditionalContext string   `json:"additionalContext"`
	RemediationSteps  []string `json:"remediationSteps"`
}

// GenerateGuardResponse creates a GuardResponse based on transcript analysis.
// Decision logic:
//   - "allow" if !analyzer.HasUncollectedTasks()
//   - "block" if uncollected tasks exist
//
// Parameters:
//   - analyzer: TranscriptAnalyzer with background task tracking state
//   - metadata: ParsedAgentMetadata with agent ID and model info
//
// Returns:
//   - *GuardResponse: structured response for hook system
func GenerateGuardResponse(analyzer *routing.TranscriptAnalyzer, metadata *routing.ParsedAgentMetadata) *GuardResponse {
	resp := &GuardResponse{
		HookEventName: "SubagentStop",
	}

	if !analyzer.HasUncollectedTasks() {
		// Allow completion
		resp.Decision = "allow"
		resp.Reason = "All background tasks collected"
		resp.AdditionalContext = formatAllowContext(analyzer, metadata)
		resp.RemediationSteps = []string{} // Empty for allow case
	} else {
		// Block completion
		resp.Decision = "block"
		resp.Reason = "Orchestrator completed with uncollected background tasks"
		resp.AdditionalContext = formatBlockContext(analyzer, metadata)
		resp.RemediationSteps = []string{
			"identify_uncollected_task_ids",
			"call_TaskOutput_for_each",
			"wait_for_all_collections",
			"verify_results_in_transcript",
		}
	}

	return resp
}

// formatAllowContext generates the AdditionalContext string for allow decisions.
func formatAllowContext(analyzer *routing.TranscriptAnalyzer, metadata *routing.ParsedAgentMetadata) string {
	var sb strings.Builder

	sb.WriteString("✅ ORCHESTRATOR COMPLETION ALLOWED\n\n")
	sb.WriteString(fmt.Sprintf("Agent: %s (model: %s)\n", metadata.AgentID, metadata.AgentModel))
	sb.WriteString(analyzer.GetSummary())
	sb.WriteString("\nOrchestrator followed fan-out/fan-in pattern correctly.\n")
	sb.WriteString("All spawned background tasks were collected before completion.\n")

	return sb.String()
}

// formatBlockContext generates the AdditionalContext string for block decisions.
func formatBlockContext(analyzer *routing.TranscriptAnalyzer, metadata *routing.ParsedAgentMetadata) string {
	var sb strings.Builder

	sb.WriteString("🛑 ORCHESTRATOR COMPLETION BLOCKED\n\n")
	sb.WriteString(fmt.Sprintf("Agent: %s (model: %s)\n", metadata.AgentID, metadata.AgentModel))
	sb.WriteString(analyzer.GetSummary())
	sb.WriteString("\nVIOLATION: Fan-out without fan-in\n")
	sb.WriteString("Reference: ~/.claude/rules/LLM-guidelines.md § 2.2 (Background Task Collection)\n\n")
	sb.WriteString("Uncollected Tasks: ")
	sb.WriteString(analyzer.GetUncollectedList())
	sb.WriteString("\n\nREQUIRED ACTIONS:\n")
	sb.WriteString("1. Call TaskOutput({task_id: \"...\", block: true}) for each uncollected task\n")
	sb.WriteString("2. Wait for all collections to complete\n")
	sb.WriteString("3. Verify results appear in transcript\n")
	sb.WriteString("4. THEN synthesize/conclude orchestration\n")

	return sb.String()
}

// FormatJSON serializes the GuardResponse to JSON string.
// Uses manual JSON formatting for precise control over escaping and structure.
//
// Returns:
//   - string: valid JSON representation of GuardResponse
func (r *GuardResponse) FormatJSON() string {
	var sb strings.Builder

	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  \"hookEventName\": \"%s\",\n", escapeJSON(r.HookEventName)))
	sb.WriteString(fmt.Sprintf("  \"decision\": \"%s\",\n", escapeJSON(r.Decision)))
	sb.WriteString(fmt.Sprintf("  \"reason\": \"%s\",\n", escapeJSON(r.Reason)))
	sb.WriteString(fmt.Sprintf("  \"hookSpecificOutput\": {\n"))
	sb.WriteString(fmt.Sprintf("    \"additionalContext\": \"%s\",\n", escapeJSON(r.AdditionalContext)))
	sb.WriteString(fmt.Sprintf("    \"remediationSteps\": %s\n", formatStringArray(r.RemediationSteps)))
	sb.WriteString("  }\n")
	sb.WriteString("}")

	return sb.String()
}

// escapeJSON escapes special characters for JSON string values.
// Handles: quotes, newlines, tabs, carriage returns, backslashes.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // Backslash first
	s = strings.ReplaceAll(s, "\"", "\\\"") // Quotes
	s = strings.ReplaceAll(s, "\n", "\\n")  // Newlines
	s = strings.ReplaceAll(s, "\r", "\\r")  // Carriage returns
	s = strings.ReplaceAll(s, "\t", "\\t")  // Tabs
	return s
}

// formatStringArray formats a string slice as a JSON array.
// Returns "[]" for nil/empty slices.
func formatStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i, item := range arr {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeJSON(item)))
	}
	sb.WriteString("]")

	return sb.String()
}

// Marshal writes the GuardResponse to the provided writer as JSON.
// Wraps the response in the hookSpecificOutput structure expected by Claude Code.
//
// Parameters:
//   - w: io.Writer to write JSON output to (typically os.Stdout)
//
// Returns:
//   - error: nil on success, error with context on failure
func (r *GuardResponse) Marshal(w io.Writer) error {
	// Wrap in hookSpecificOutput structure
	wrapper := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     r.HookEventName,
			"decision":          r.Decision,
			"reason":            r.Reason,
			"additionalContext": r.AdditionalContext,
			"remediationSteps":  r.RemediationSteps,
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(wrapper); err != nil {
		return fmt.Errorf("[orchestrator-guard] Failed to marshal JSON: %w", err)
	}

	return nil
}
