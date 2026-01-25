package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Get project directory from environment or current directory
	// Priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > cwd
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	// Load routing schema
	schema, err := routing.LoadSchema()
	if err != nil {
		outputError(fmt.Sprintf("Failed to load routing schema: %v", err))
		os.Exit(1)
	}

	// Parse event from STDIN with timeout
	event, err := parseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only validate Task tool
	if event.ToolName != "Task" {
		// Pass through for non-Task tools
		fmt.Println("{}")
		return
	}

	// Log routing decision for Task() calls (GOgent-087e)
	if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
		decision := telemetry.NewRoutingDecision(
			event.SessionID,
			taskInput.Prompt,
			taskInput.Model,
			extractAgentFromPrompt(taskInput.Prompt),
		)
		if logErr := telemetry.LogRoutingDecision(decision); logErr != nil {
			// Non-blocking: log warning but continue execution
			fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log routing decision: %v\n", logErr)
		}
	} else {
		// Failed to parse task input - log warning but continue
		fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to parse Task input for routing decision: %v\n", err)
	}

	// Create validation orchestrator
	orchestrator := routing.NewValidationOrchestrator(schema, projectDir, nil)

	// Validate task
	result := orchestrator.ValidateTask(event.ToolInput, event.SessionID)

	// Output result
	outputResult(result, event.SessionID)

	// Log violations if any
	for _, violation := range result.Violations {
		if err := routing.LogViolation(violation, projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log violation: %v\n", err)
		}
	}
}

// parseEvent reads and parses ToolEvent from STDIN with timeout
func parseEvent(r io.Reader, timeout time.Duration) (*routing.ToolEvent, error) {
	type result struct {
		event *routing.ToolEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		reader := bufio.NewReader(r)
		data, err := io.ReadAll(reader)
		if err != nil {
			ch <- result{nil, err}
			return
		}

		var event routing.ToolEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("invalid JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("STDIN read timeout after %v", timeout)
	}
}

// outputResult writes validation result as JSON to STDOUT
func outputResult(result *routing.ValidationResult, _ string) {
	output := make(map[string]any)

	// Always include decision for consistent parsing
	output["decision"] = result.Decision

	if result.Decision == "block" {
		output["reason"] = result.Reason
		output["hookSpecificOutput"] = map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": result.Reason,
		}
	} else {
		// Allow with optional warnings
		if result.ModelMismatch != "" {
			output["reason"] = result.ModelMismatch
			output["hookSpecificOutput"] = map[string]any{
				"hookEventName":     "PreToolUse",
				"additionalContext": "⚠️ " + result.ModelMismatch,
			}
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

// outputError writes error message in hook format
func outputError(message string) {
	output := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PreToolUse",
			"additionalContext": "🔴 " + message,
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

// extractAgentFromPrompt extracts agent ID from "AGENT: agent-name" prefix.
// Returns "unknown" if the AGENT: prefix is not found.
// This parses the delegation prompt structure where agents are identified
// by an "AGENT: agent-id" line at the start of the prompt.
func extractAgentFromPrompt(prompt string) string {
	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if agent, found := strings.CutPrefix(trimmed, "AGENT:"); found {
			agent = strings.TrimSpace(agent)
			if agent != "" {
				return agent
			}
		}
	}
	return "unknown"
}
