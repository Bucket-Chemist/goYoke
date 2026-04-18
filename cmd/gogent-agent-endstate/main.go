package main

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

	// Process event (testable logic extracted)
	response, err := processEvent(event)
	if err != nil {
		outputError(fmt.Sprintf("Processing failed: %v", err))
		os.Exit(1)
	}

	// Output response as JSON to stdout
	if err := response.Marshal(os.Stdout); err != nil {
		outputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

// processEvent handles SubagentStop event processing (testable)
func processEvent(event *routing.SubagentStopEvent) (*workflow.EndstateResponse, error) {
	// Extract agent metadata: uses direct event fields (v2.1.69+) with transcript fallback
	metadata, parseErr := routing.EnrichMetadataFromEvent(event)
	if parseErr != nil {
		// Non-fatal: transcript parsing failure
		// Continue with empty metadata - GenerateEndstateResponse will use defaults
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

	// Log collaboration (GOgent-088c - non-blocking)
	// This captures agent delegation patterns for ML optimization
	if err := logCollaboration(event, metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log collaboration: %v\n", err)
		// Don't exit - logging failure is non-fatal
	}

	// Emit lifecycle complete event for TUI real-time tracking (GOgent-109)
	if err := logLifecycleComplete(event, metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log lifecycle complete: %v\n", err)
		// Don't exit - logging failure is non-fatal
	}

	// AC check — entirely defensive, must never crash the hook.
	// TODO: AC sidecar cleanup (removing SESSION_DIR/ac/{agentID}.json) should happen
	// in gogent-archive (SessionEnd hook) after all endstate hooks have run.
	checkAcceptanceCriteria(metadata.AgentID)

	return response, nil
}

// checkAcceptanceCriteria reads the AC sidecar written by the NDJSON goroutine
// (task-004) at SESSION_DIR/ac/{agentID}.json and warns on unmet criteria.
// Entirely defensive: file-not-found and parse errors are logged but never fatal.
func checkAcceptanceCriteria(agentID string) {
	sessionDir := routing.GetSessionDir()
	if sessionDir == "" || agentID == "" {
		slog.Debug("AC check skipped: no session dir or agent ID", "agentID", agentID)
		return
	}

	acPath := filepath.Join(sessionDir, "ac", agentID+".json")
	acData, err := os.ReadFile(acPath)
	if err != nil {
		// No sidecar = no AC = skip (backward compatible)
		slog.Debug("no AC sidecar", "path", acPath, "err", err)
		return
	}

	var criteria []struct {
		Text      string `json:"text"`
		Completed bool   `json:"completed"`
	}
	if jsonErr := json.Unmarshal(acData, &criteria); jsonErr != nil {
		slog.Warn("malformed AC sidecar", "path", acPath, "err", jsonErr)
		// Continue without AC check — do NOT panic
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
// Non-blocking: errors are logged to stderr but execution continues.
func logCollaboration(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata) error {
	// Create collaboration record
	collab := telemetry.NewAgentCollaboration(
		event.SessionID,
		"terminal",        // Parent is always terminal in SubagentStop context
		metadata.AgentID,  // Child agent from transcript metadata
		"spawn",           // DelegationType
	)

	// Set outcome from metadata
	collab.ChildSuccess = metadata.IsSuccess()
	collab.ChildDurationMs = int64(metadata.DurationMs)
	collab.ChainDepth = 1 // SubagentStop always represents root-level delegation

	// Log to persistent storage
	return telemetry.LogCollaboration(collab)
}

// logLifecycleComplete records agent completion for TUI real-time tracking.
// Non-blocking: errors are logged to stderr but execution continues.
func logLifecycleComplete(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata) error {
	// Create lifecycle complete event
	lifecycle := telemetry.NewAgentLifecycleEvent(
		event.SessionID,
		"complete",
		metadata.AgentID,
		"terminal", // Parent is always terminal in SubagentStop context
		metadata.Tier,
		"", // No description on completion
		"", // TODO: Correlate with spawn DecisionID (requires passing through metadata)
	)

	// Set outcome from metadata
	success := metadata.IsSuccess()
	duration := int64(metadata.DurationMs)
	lifecycle.Success = &success
	lifecycle.DurationMs = &duration

	// Log to persistent storage
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
