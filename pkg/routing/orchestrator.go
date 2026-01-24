package routing

import (
	"fmt"
	"io"
	"time"
)

// ParseOrchestratorStopEvent parses SubagentStop event and extracts agent metadata.
// This function composes ParseSubagentStopEvent and ParseTranscriptForMetadata to
// provide orchestrator-guard with complete agent information in a single call.
//
// Returns ParsedAgentMetadata (possibly partial on transcript errors) and any error encountered.
// Follows graceful degradation: returns partial metadata even when transcript parsing fails.
func ParseOrchestratorStopEvent(r io.Reader, timeout time.Duration) (*ParsedAgentMetadata, error) {
	// Parse the SubagentStop event from stdin
	event, err := ParseSubagentStopEvent(r, timeout)
	if err != nil {
		return nil, fmt.Errorf("[orchestrator-guard] Failed to parse SubagentStop event: %w", err)
	}

	// Extract agent metadata from the transcript file
	// ParseTranscriptForMetadata returns partial metadata even on error (graceful degradation)
	metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		// Wrap error but still return partial metadata
		return metadata, fmt.Errorf("[orchestrator-guard] Failed to parse transcript at %s: %w", event.TranscriptPath, err)
	}

	return metadata, nil
}

// IsOrchestratorType returns true if the agent is orchestrator or architect.
// These agents require tier-specific follow-up prompts after completion.
//
// Returns false for all other agent types, including unknown/empty agents.
func IsOrchestratorType(metadata *ParsedAgentMetadata) bool {
	if metadata == nil {
		return false
	}

	return metadata.AgentID == "orchestrator" || metadata.AgentID == "architect"
}
