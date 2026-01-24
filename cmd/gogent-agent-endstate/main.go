package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/workflow"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event (uses ACTUAL schema)
	event, err := routing.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Parse transcript for agent metadata (CRITICAL: metadata not in event directly)
	metadata, parseErr := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if parseErr != nil {
		// Non-fatal: transcript parsing failure
		// Continue with nil metadata - GenerateEndstateResponse will use defaults
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "Continuing with default agent metadata...\n")
	}

	// Generate response (accepts event + metadata, handles nil gracefully)
	response := workflow.GenerateEndstateResponse(event, metadata)

	// Log decision (non-blocking if fails)
	// CORRECTED: LogEndstate requires metadata parameter
	// Ensure metadata is not nil before logging (graceful degradation)
	if metadata == nil {
		// Use default metadata for logging
		metadata = &routing.ParsedAgentMetadata{
			AgentID:  "unknown",
			Tier:     "unknown",
			ExitCode: 0,
		}
	}

	if err := workflow.LogEndstate(event, metadata, response); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log endstate: %v\n", err)
		// Don't exit - logging failure is non-fatal
	}

	// Output response as JSON to stdout
	if err := response.Marshal(os.Stdout); err != nil {
		outputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "silent",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
