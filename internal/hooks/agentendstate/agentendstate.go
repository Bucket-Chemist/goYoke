// Package agentendstate implements the goyoke-agent-endstate SubagentStop hook.
package agentendstate

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
	"github.com/Bucket-Chemist/goYoke/pkg/workflow"
)

const defaultTimeout = 5 * time.Second

// Main is the entrypoint for the goyoke-agent-endstate hook.
func Main() {
	event, err := routing.ParseSubagentStopEvent(os.Stdin, defaultTimeout)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	response, err := processEvent(event)
	if err != nil {
		outputError(fmt.Sprintf("Processing failed: %v", err))
		os.Exit(1)
	}

	if err := response.Marshal(os.Stdout); err != nil {
		outputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

// processEvent handles SubagentStop event processing.
func processEvent(event *routing.SubagentStopEvent) (*workflow.EndstateResponse, error) {
	metadata, parseErr := routing.EnrichMetadataFromEvent(event)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "Continuing with default agent metadata...\n")
	}

	response := workflow.GenerateEndstateResponse(event, metadata)

	if metadata == nil {
		metadata = &routing.ParsedAgentMetadata{
			AgentID:  "unknown",
			Tier:     "unknown",
			ExitCode: 0,
		}
	}

	if err := workflow.LogEndstate(event, metadata, response); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log endstate: %v\n", err)
	}

	if err := logCollaboration(event, metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log collaboration: %v\n", err)
	}

	if err := logLifecycleComplete(event, metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log lifecycle complete: %v\n", err)
	}

	checkAcceptanceCriteria(metadata.AgentID)

	return response, nil
}

// checkAcceptanceCriteria reads the AC sidecar and warns on unmet criteria.
func checkAcceptanceCriteria(agentID string) {
	sessionDir := routing.GetSessionDir()
	if sessionDir == "" || agentID == "" {
		slog.Debug("AC check skipped: no session dir or agent ID", "agentID", agentID)
		return
	}

	acPath := filepath.Join(sessionDir, "ac", agentID+".json")
	acData, err := os.ReadFile(acPath)
	if err != nil {
		slog.Debug("no AC sidecar", "path", acPath, "err", err)
		return
	}

	var criteria []struct {
		Text      string `json:"text"`
		Completed bool   `json:"completed"`
	}
	if jsonErr := json.Unmarshal(acData, &criteria); jsonErr != nil {
		slog.Warn("malformed AC sidecar", "path", acPath, "err", jsonErr)
		return
	}

	incomplete := 0
	for _, c := range criteria {
		if !c.Completed {
			incomplete++
		}
	}
	if incomplete > 0 {
		fmt.Fprintf(os.Stderr, "[agent-endstate] WARN: %d/%d acceptance criteria unmet for %s\n",
			incomplete, len(criteria), agentID)
	}
}

// logCollaboration records agent delegation patterns for ML analysis.
func logCollaboration(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata) error {
	collab := telemetry.NewAgentCollaboration(
		event.SessionID,
		"terminal",
		metadata.AgentID,
		"spawn",
	)

	collab.ChildSuccess = metadata.IsSuccess()
	collab.ChildDurationMs = int64(metadata.DurationMs)
	collab.ChainDepth = 1

	return telemetry.LogCollaboration(collab)
}

// logLifecycleComplete records agent completion for TUI real-time tracking.
func logLifecycleComplete(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata) error {
	lifecycle := telemetry.NewAgentLifecycleEvent(
		event.SessionID,
		"complete",
		metadata.AgentID,
		"terminal",
		metadata.Tier,
		"",
		"",
	)

	success := metadata.IsSuccess()
	duration := int64(metadata.DurationMs)
	lifecycle.Success = &success
	lifecycle.DurationMs = &duration

	return telemetry.LogAgentLifecycle(lifecycle)
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
