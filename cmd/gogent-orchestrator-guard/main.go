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

	// Extract agent metadata from transcript using EXISTING routing function
	metadata, parseErr := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if parseErr != nil {
		// Non-fatal: graceful degradation (allow completion if can't parse)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Allowing completion (parsing failure is non-blocking)\n")
		outputAllow("Transcript parsing failed - allowing by default")
		return
	}

	// Check if orchestrator type using EXISTING GetAgentClass function
	agentClass := routing.GetAgentClass(metadata.AgentID)
	if agentClass != routing.ClassOrchestrator {
		// Silent pass-through for non-orchestrator agents
		outputAllow(fmt.Sprintf("Non-orchestrator agent (%s) - no guard needed", metadata.AgentID))
		return
	}

	// Analyze transcript for background task tracking
	// Uses GOgent-078 enforcement.NewTranscriptAnalyzer() implementation
	analyzer := routing.NewTranscriptAnalyzer(event.TranscriptPath)
	if err := analyzer.Analyze(); err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Analysis failed: %v\n", err)
		outputAllow("Analysis failed - allowing by default")
		return
	}

	// Generate guard response (block if uncollected tasks)
	// Uses GOgent-077 enforcement.GenerateGuardResponse() implementation
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
