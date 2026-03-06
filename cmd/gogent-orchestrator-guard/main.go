package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/enforcement"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event using EXISTING routing function
	event, err := routing.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Extract agent metadata: uses direct event fields (v2.1.69+) with transcript fallback
	metadata, parseErr := routing.EnrichMetadataFromEvent(event)
	if parseErr != nil {
		// Non-fatal: graceful degradation (allow completion if can't parse)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Allowing completion (parsing failure is non-blocking)\n")
		outputAllow("Transcript parsing failed - allowing by default")
		return
	}

	// Check delegation requirements for all agents (MCP-SPAWN-014)
	// This validates must_delegate and min_delegations from agents-index.json
	delegationResponse, err := routing.ValidateDelegationFromTranscript(event.TranscriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Delegation validation failed: %v\n", err)
		// Non-fatal: fail-open on validation errors
	} else if delegationResponse.Decision == routing.DecisionBlock {
		// Delegation requirement not met - block completion
		if err := delegationResponse.Marshal(os.Stdout); err != nil {
			outputError(fmt.Sprintf("Failed to marshal delegation response: %v", err))
			os.Exit(1)
		}
		return
	}

	// Check if orchestrator type for background task validation
	agentClass := routing.GetAgentClass(metadata.AgentID)
	if agentClass != routing.ClassOrchestrator {
		// Non-orchestrator agent - delegation check passed, allow completion
		outputAllow(fmt.Sprintf("Agent %s delegation requirements met", metadata.AgentID))
		return
	}

	// Orchestrator-specific: Analyze transcript for background task tracking
	// Use agent_transcript_path when available (v2.1.69+)
	analyzerPath := event.TranscriptPath
	if event.AgentTranscriptPath != "" {
		analyzerPath = event.AgentTranscriptPath
	}
	analyzer := routing.NewTranscriptAnalyzer(analyzerPath)
	if err := analyzer.Analyze(); err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Analysis failed: %v\n", err)
		outputAllow("Analysis failed - allowing by default")
		return
	}

	// Generate guard response (block if uncollected tasks)
	response := enforcement.GenerateGuardResponse(analyzer, metadata)

	// Output response as JSON to stdout
	if err := response.Marshal(os.Stdout); err != nil {
		outputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

func outputAllow(reason string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "reason": "%s"
  }
}`, escapeJSON(reason))
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, escapeJSON(message))
}

func escapeJSON(s string) string {
	// Basic JSON escaping for embedded strings
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
