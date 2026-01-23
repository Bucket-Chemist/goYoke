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

func main() {
	// Get configuration
	projectDir := getProjectDir()

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

		// Capture sharp edge
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

		// Return blocking response
		outputBlock(failure.File, count, failure.ErrorType)
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

// outputBlock returns a hook-compliant blocking response
func outputBlock(file string, count int, errorType string) {
	response := map[string]interface{}{
		"decision": "block",
		"reason":   fmt.Sprintf("⚠️ SHARP EDGE DETECTED: %d consecutive failures on '%s' (%s)", count, file, errorType),
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": "PostToolUse",
			"additionalContext": fmt.Sprintf(
				"🔴 DEBUGGING LOOP DETECTED (%d failures on %s):\n"+
					"1. STOP current approach\n"+
					"2. Sharp edge auto-logged to pending-learnings.jsonl\n"+
					"3. Analyze root cause - what assumption is wrong?\n"+
					"4. Consider escalation to next tier\n"+
					"5. Check sharp-edges.yaml for similar patterns",
				count, file),
		},
	}
	outputJSON(response)
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
