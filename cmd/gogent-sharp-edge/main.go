package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/memory"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

const (
	// DEFAULT_TIMEOUT for reading STDIN events
	DEFAULT_TIMEOUT = 5 * time.Second
)

// getAgentDirectories returns a list of agent directories to scan for sharp-edges.yaml files.
// It constructs paths based on the home directory structure:
// ~/.claude/agents/{agent-name}/sharp-edges.yaml
//
// Returns:
//   - []string: List of absolute paths to agent directories
//
// Note: This function returns all common agent directories. LoadSharpEdgesIndex
// will skip any that don't exist or lack sharp-edges.yaml files.
func getAgentDirectories() []string {
	home := os.Getenv("HOME")
	if home == "" {
		// Fallback to current user's home directory
		home = filepath.Join("/home", os.Getenv("USER"))
	}

	claudeAgentsDir := filepath.Join(home, ".claude", "agents")

	// Common agent directories
	// These match the agents defined in agents-index.json
	agents := []string{
		"python-pro",
		"python-ux",
		"go-pro",
		"go-cli",
		"go-tui",
		"go-api",
		"go-concurrent",
		"r-pro",
		"r-shiny-pro",
		"codebase-search",
		"scaffolder",
		"tech-docs-writer",
		"librarian",
		"code-reviewer",
		"orchestrator",
		"architect",
	}

	dirs := make([]string, 0, len(agents))
	for _, agent := range agents {
		dirs = append(dirs, filepath.Join(claudeAgentsDir, agent))
	}

	return dirs
}

func main() {
	// Get configuration
	projectDir := getProjectDir()

	// Load sharp edges index early (before event parsing)
	// This allows pattern matching to work even if loading fails gracefully
	agentDirs := getAgentDirectories()
	index, err := memory.LoadSharpEdgesIndex(agentDirs)
	if err != nil {
		// Log warning but continue - we'll just not have pattern matching
		fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: Failed to load sharp edges index: %v\n", err)
		index = &memory.SharpEdgeIndex{} // Empty index
	}

	// Parse PostToolUse event from STDIN
	event, err := routing.ParsePostToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		// Non-fatal: might be non-JSON or timeout, just pass through
		fmt.Println("{}")
		return
	}

	// Detect failure
	failure := routing.DetectFailure(event)
	if failure == nil {
		// No failure detected, pass through
		fmt.Println("{}")
		return
	}

	// Log failure to tracker
	if err := memory.LogFailure(failure); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: Failed to log failure: %v\n", err)
	}

	// Check failure count
	count, err := memory.GetFailureCount(failure.File, failure.ErrorType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: Failed to get failure count: %v\n", err)
		fmt.Println("{}")
		return
	}

	threshold := memory.DefaultMaxFailures

	// Check threshold
	if count >= threshold {
		// Extract attempted change from tool input
		attemptedChange := session.ExtractAttemptedChange(event)

		// Extract code snippet around the error (if file path is valid)
		codeSnippet := ""
		if failure.File != "unknown" && failure.File != "" {
			snippet, _ := session.ExtractCodeSnippet(failure.File, 0, 2)
			codeSnippet = snippet
		}

		// Capture sharp edge to pending learnings
		edge := session.SharpEdge{
			File:                failure.File,
			ErrorType:           failure.ErrorType,
			ConsecutiveFailures: count,
			Timestamp:           failure.Timestamp,
			Context:             fmt.Sprintf("Tool: %s", failure.Tool),
			Type:                "sharp_edge",
			Tool:                failure.Tool,
			CodeSnippet:         codeSnippet,
			AttemptedChange:     attemptedChange,
			Status:              "pending_review",
		}

		// Write to pending-learnings.jsonl
		if err := writePendingLearning(projectDir, edge); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: Failed to write learning: %v\n", err)
		}

		// Build SharpEdge struct for pattern matching
		// Extract error message from tool response
		errorMsg := failure.ErrorMatch
		if errorMsg == "" {
			// Try to extract from tool response output
			if event.ToolResponse != nil {
				if output, ok := event.ToolResponse["output"].(string); ok {
					errorMsg = output
				} else if errStr, ok := event.ToolResponse["error"].(string); ok {
					errorMsg = errStr
				}
			}
		}

		patternEdge := &memory.SharpEdge{
			File:         failure.File,
			ErrorType:    failure.ErrorType,
			ErrorMessage: errorMsg,
		}

		// Generate enhanced blocking response with pattern matching
		resp := memory.GenerateBlockingResponse(patternEdge, index, count)

		// Marshal and output response
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Error marshaling response: %v\n", err)
			// Fallback to simple JSON
			fmt.Println("{}")
		}
		return
	}

	// Check warning threshold (threshold - 1)
	if count == threshold-1 {
		outputWarning(failure.File, count)
		return
	}

	// Below threshold, pass through
	fmt.Println("{}")
}

// writePendingLearning appends a SharpEdge to pending-learnings.jsonl
func writePendingLearning(projectDir string, edge session.SharpEdge) error {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pendingPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Marshal edge to JSON
	data, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("marshal edge: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(pendingPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}


// outputWarning returns a warning response at threshold-1
func outputWarning(file string, count int) {
	response := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": "PostToolUse",
			"additionalContext": fmt.Sprintf(
				"⚠️ WARNING: %d failures on '%s'. One more failure triggers sharp edge capture.",
				count, file),
		},
	}
	outputJSON(response)
}

// outputJSON marshals and prints a JSON response
func outputJSON(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Println(string(data))
}

// getProjectDir returns the project directory from environment or cwd
func getProjectDir() string {
	if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	dir, _ := os.Getwd()
	return dir
}
