// Package orchestratorguard implements the gogent-orchestrator-guard hook.
// It validates background task collection before orchestrator completion.
package orchestratorguard

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/enforcement"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// DefaultTimeout is the read timeout for stdin events.
const DefaultTimeout = 5 * time.Second

// Main is the entrypoint for the gogent-orchestrator-guard hook.
func Main() {
	event, err := routing.ParseSubagentStopEvent(os.Stdin, DefaultTimeout)
	if err != nil {
		OutputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	metadata, parseErr := routing.EnrichMetadataFromEvent(event)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Allowing completion (parsing failure is non-blocking)\n")
		OutputAllow("Transcript parsing failed - allowing by default")
		return
	}

	delegationResponse, err := routing.ValidateDelegationFromTranscript(event.TranscriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Delegation validation failed: %v\n", err)
	} else if delegationResponse.Decision == routing.DecisionBlock {
		if err := delegationResponse.Marshal(os.Stdout); err != nil {
			OutputError(fmt.Sprintf("Failed to marshal delegation response: %v", err))
			os.Exit(1)
		}
		return
	}

	agentClass := routing.GetAgentClass(metadata.AgentID)
	if agentClass != routing.ClassOrchestrator {
		OutputAllow(fmt.Sprintf("Agent %s delegation requirements met", metadata.AgentID))
		return
	}

	analyzerPath := event.TranscriptPath
	if event.AgentTranscriptPath != "" {
		analyzerPath = event.AgentTranscriptPath
	}
	analyzer := routing.NewTranscriptAnalyzer(analyzerPath)
	if err := analyzer.Analyze(); err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Analysis failed: %v\n", err)
		OutputAllow("Analysis failed - allowing by default")
		return
	}

	response := enforcement.GenerateGuardResponse(analyzer, metadata)

	if err := response.Marshal(os.Stdout); err != nil {
		OutputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

// OutputAllow writes an allow decision to stdout.
func OutputAllow(reason string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "reason": "%s"
  }
}`, EscapeJSON(reason))
}

// OutputError writes an error response (degrades to allow) to stdout.
func OutputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, EscapeJSON(message))
}

// EscapeJSON performs basic JSON escaping for embedded strings.
func EscapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
