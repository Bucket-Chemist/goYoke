package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

// getLogPath returns the path for validation logs.
// Uses XDG_DATA_HOME/gogent-fortress/validate.log or falls back to ~/.local/share/
func getLogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "gogent-fortress", "validate.log")
}

// logValidation writes a validation event to the log file for visibility.
func logValidation(sessionID, agent, model, decision, reason string) {
	logPath := getLogPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return // Silent fail - don't block validation
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silent fail
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02T15:04:05")
	shortSession := sessionID
	if len(sessionID) > 8 {
		shortSession = sessionID[:8]
	}

	// Format: [timestamp] session=xxx decision=allow/block agent=name model=tier reason=...
	entry := fmt.Sprintf("[%s] session=%s decision=%-5s agent=%-20s model=%-6s",
		timestamp, shortSession, decision, agent, model)
	if reason != "" {
		entry += fmt.Sprintf(" reason=%s", reason)
	}
	entry += "\n"

	f.WriteString(entry)
}

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
	var decisionID string
	if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
		decision := telemetry.NewRoutingDecision(
			event.SessionID,
			taskInput.Prompt,
			taskInput.Model,
			extractAgentFromPrompt(taskInput.Prompt),
		)
		decisionID = decision.DecisionID // Capture for lifecycle correlation
		if logErr := telemetry.LogRoutingDecision(decision); logErr != nil {
			// Non-blocking: log warning but continue execution
			fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log routing decision: %v\n", logErr)
		}

		// Emit lifecycle spawn event for TUI real-time tracking (GOgent-109)
		lifecycle := telemetry.NewAgentLifecycleEvent(
			event.SessionID,
			"spawn",
			extractAgentFromPrompt(taskInput.Prompt),
			"terminal", // Parent is terminal for direct spawns
			taskInput.Model,
			taskInput.Prompt,
			decisionID,
		)
		if logErr := telemetry.LogAgentLifecycle(lifecycle); logErr != nil {
			fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log lifecycle spawn: %v\n", logErr)
		}
	} else {
		// Failed to parse task input - log warning but continue
		fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to parse Task input for routing decision: %v\n", err)
	}

	// Create validation orchestrator
	orchestrator := routing.NewValidationOrchestrator(schema, projectDir, nil)

	// Validate task
	result := orchestrator.ValidateTask(event.ToolInput, event.SessionID)

	// Extract agent and model for logging
	agent := "unknown"
	model := "unknown"
	if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
		agent = extractAgentFromPrompt(taskInput.Prompt)
		model = taskInput.Model
		if model == "" {
			model = "default"
		}
	}

	// Log validation for visibility (Option 2: explicit file logging)
	logValidation(event.SessionID, agent, model, result.Decision, result.Reason)

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
