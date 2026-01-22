package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
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
func outputResult(result *routing.ValidationResult, sessionID string) {
	output := make(map[string]interface{})

	// Always include decision for consistent parsing
	output["decision"] = result.Decision

	if result.Decision == "block" {
		output["reason"] = result.Reason
		output["hookSpecificOutput"] = map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": result.Reason,
		}
	} else {
		// Allow with optional warnings
		if result.ModelMismatch != "" {
			output["reason"] = result.ModelMismatch
			output["hookSpecificOutput"] = map[string]interface{}{
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
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PreToolUse",
			"additionalContext": "🔴 " + message,
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}
