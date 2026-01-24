package workflow

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// EndstateResponse represents the response to SubagentStop
type EndstateResponse struct {
	HookEventName     string   `json:"hookEventName"`
	Decision          string   `json:"decision"` // "prompt", "silent"
	AdditionalContext string   `json:"additionalContext"`
	Tier              string   `json:"tier,omitempty"`
	AgentClass        string   `json:"agentClass,omitempty"`
	Recommendations   []string `json:"recommendations,omitempty"`
}

// GenerateEndstateResponse creates tier-specific response based on agent completion.
// If metadata is nil, generates generic response (graceful degradation).
func GenerateEndstateResponse(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata) *EndstateResponse {
	// Graceful degradation: if metadata parsing failed, use defaults
	if metadata == nil {
		metadata = &routing.ParsedAgentMetadata{
			AgentID:  "unknown",
			Tier:     "unknown",
			ExitCode: 0,
		}
	}

	agentClass := routing.GetAgentClass(metadata.AgentID)
	isSuccess := metadata.IsSuccess()

	response := &EndstateResponse{
		HookEventName: "SubagentStop",
		Tier:          metadata.Tier,
		AgentClass:    string(agentClass),
	}

	if !isSuccess {
		// Agent failed - always prompt
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"⚠️ AGENT FAILED\n\nAgent: %s (tier: %s)\nExit Code: %d\nDuration: %dms\n\n"+
				"Reasons to investigate:\n"+
				"• Check error logs\n"+
				"• Review agent transcript for blocker\n"+
				"• Consider escalation to higher tier\n"+
				"• Retry with modified prompt or scope",
			metadata.AgentID, metadata.Tier, metadata.ExitCode, metadata.DurationMs)
		response.Recommendations = []string{
			"review_error_cause",
			"check_transcript",
			"consider_escalation",
		}
		return response
	}

	// Agent succeeded - tier-specific prompts
	switch agentClass {
	case routing.ClassOrchestrator:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ ORCHESTRATOR COMPLETED\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Orchestration checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Have you updated TODOs based on decisions made?\n"+
				"3. [ ] Did the agent spawn background tasks? Collected all results?\n"+
				"4. [ ] Should architectural decisions be captured in memory?\n"+
				"5. [ ] Are any follow-up tickets needed?\n\n"+
				"Recommended next step: Capture key decisions and verify background task collection.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"update_todos",
			"verify_background_collection",
			"capture_decisions",
			"knowledge_compound",
		}

	case routing.ClassImplementation:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ IMPLEMENTATION COMPLETE\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Implementation checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Did tests pass (if agent added tests)?\n"+
				"3. [ ] Review implementation against conventions (python.md, go.md, etc.)\n"+
				"4. [ ] Any integration issues with existing code?\n"+
				"5. [ ] Document any workarounds or tradeoffs\n\n"+
				"Recommended next step: Verify test coverage and review against style conventions.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"verify_tests",
			"review_conventions",
			"check_integration",
		}

	case routing.ClassSpecialist:
		response.Decision = "prompt"
		response.AdditionalContext = fmt.Sprintf(
			"✅ SPECIALIST COMPLETED\n\n"+
				"Agent: %s | Tier: %s | Duration: %dms | Tokens: %d\n\n"+
				"Specialist checklist:\n"+
				"1. ✅ Agent completed successfully\n"+
				"2. [ ] Output meets quality standards?\n"+
				"3. [ ] Follow-up actions identified in output?\n"+
				"4. [ ] Any issues need escalation?\n\n"+
				"Recommended next step: Review specialist output and execute follow-up actions.",
			metadata.AgentID, metadata.Tier, metadata.DurationMs, metadata.OutputTokens)
		response.Recommendations = []string{
			"review_output",
			"execute_followups",
		}

	case routing.ClassCoordination:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf(
			"✅ Coordination agent %s completed in %dms",
			metadata.AgentID, metadata.DurationMs)
		response.Recommendations = []string{
			"continue_workflow",
		}

	default:
		response.Decision = "silent"
		response.AdditionalContext = fmt.Sprintf("Agent %s completed (exit: %d)", metadata.AgentID, metadata.ExitCode)
	}

	return response
}

// Marshal writes the EndstateResponse as JSON to the provided writer.
// Replaces manual JSON formatting with json.Marshal for robustness.
func (r *EndstateResponse) Marshal(w io.Writer) error {
	// Wrap in hookSpecificOutput structure
	wrapper := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     r.HookEventName,
			"decision":          r.Decision,
			"additionalContext": r.AdditionalContext,
			"metadata": map[string]interface{}{
				"tier":            r.Tier,
				"agentClass":      r.AgentClass,
				"recommendations": r.Recommendations,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(wrapper); err != nil {
		return fmt.Errorf("[agent-endstate] Failed to marshal JSON: %w", err)
	}
	return nil
}
