package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/memory"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
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
		// NEW AGENTS (added in schema v2.4.0):
		"typescript-pro",
		"react-pro",
		"backend-reviewer",
		"frontend-reviewer",
		"standards-reviewer",
		"review-orchestrator",
		// NEW AGENT (added in schema v2.5.0):
		"impl-manager",
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
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to load sharp edges index: %v\n", err)
		index = &memory.SharpEdgeIndex{} // Empty index
	}

	// === ATTENTION-GATE LOGIC ===
	// Increment tool counter FIRST - we want to count ALL tool calls regardless of parse success
	// This ensures handoff metrics are accurate even when event parsing fails
	toolCount, counterErr := config.GetToolCountAndIncrement()
	if counterErr != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: counter error: %v\n", counterErr)
		toolCount = 0 // Continue with count=0 on error
	}

	// Parse PostToolUse event from STDIN
	event, err := routing.ParsePostToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		// Non-fatal: might be non-JSON or timeout, just pass through
		// Counter already incremented above, so handoff will still be accurate
		fmt.Println("{}")
		return
	}

	// === VITEST CLEANUP ===
	// Clean up vitest processes that may persist after typescript-pro/react-pro agents complete
	cleanupVitestProcesses(event)

	// === ML TOOL EVENT LOGGING (goYoke-087d) ===
	// Log to global and project-scoped paths (non-blocking)
	if mlErr := telemetry.LogMLToolEvent(event, projectDir); mlErr != nil {
		// Log error but don't fail hook - ML logging is non-critical
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] ML logging warning: %v\n", mlErr)
	}

	// Check reminder threshold
	var reminderMsg string
	if config.ShouldRemind(toolCount) {
		summary := "See ~/.claude/routing-schema.json for tier mappings"
		reminderMsg = session.GenerateRoutingReminder(toolCount, summary)
	}

	// Check flush threshold
	var flushMsg string
	if config.ShouldFlush(toolCount) {
		shouldFlush, _, flushErr := session.ShouldFlushLearnings(projectDir)
		if flushErr != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: flush check error: %v\n", flushErr)
		} else if shouldFlush {
			ctx, archiveErr := session.ArchivePendingLearnings(projectDir)
			if archiveErr != nil {
				fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: archive error: %v\n", archiveErr)
			} else {
				flushMsg = session.GenerateFlushNotification(ctx)
			}
		}
	}

	// === SANDBOX BLOCK DETECTION ===
	if sandboxMsg := detectSandboxBlock(event); sandboxMsg != "" {
		resp := buildCombinedResponse(nil, sandboxMsg, "")
		if reminderMsg != "" || flushMsg != "" {
			// Merge attention-gate messages into sandbox response.
			existing := ""
			if ctx, ok := resp.HookSpecificOutput["additionalContext"].(string); ok {
				existing = ctx
			}
			var parts []string
			if existing != "" {
				parts = append(parts, existing)
			}
			if reminderMsg != "" {
				parts = append(parts, reminderMsg)
			}
			if flushMsg != "" {
				parts = append(parts, flushMsg)
			}
			resp.AddField("additionalContext", strings.Join(parts, "\n\n"))
		}
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling sandbox response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	// === EXISTING: SHARP-EDGE LOGIC ===
	// Detect failure
	failure := routing.DetectFailure(event)

	// If no failure detected and no attention-gate messages, pass through
	if failure == nil {
		if reminderMsg == "" && flushMsg == "" {
			fmt.Println("{}")
			return
		}
		// No failure but have attention-gate messages
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	// Log failure to tracker
	if err := memory.LogFailure(failure); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to log failure: %v\n", err)
	}

	// Check failure count
	count, err := memory.GetFailureCount(failure.File, failure.ErrorType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to get failure count: %v\n", err)
		// Still output attention-gate messages if present
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
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
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to write learning: %v\n", err)
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

		// Merge attention-gate messages into blocking response
		if reminderMsg != "" || flushMsg != "" {
			// Get existing additionalContext
			existingContext := ""
			if ctx, ok := resp.HookSpecificOutput["additionalContext"].(string); ok {
				existingContext = ctx
			}

			var parts []string
			if existingContext != "" {
				parts = append(parts, existingContext)
			}
			if reminderMsg != "" {
				parts = append(parts, reminderMsg)
			}
			if flushMsg != "" {
				parts = append(parts, flushMsg)
			}
			resp.AddField("additionalContext", strings.Join(parts, "\n\n"))
		}

		// Marshal and output response
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			// Fallback to simple JSON
			fmt.Println("{}")
		}
		return
	}

	// Check warning threshold (threshold - 1)
	if count == threshold-1 {
		// Output warning with attention-gate messages
		resp := buildWarningResponse(failure.File, count, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	// Below threshold - combine with attention-gate messages if any
	if reminderMsg != "" || flushMsg != "" {
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	// No messages at all, pass through
	fmt.Println("{}")
}

// writePendingLearning appends a SharpEdge to pending-learnings.jsonl
func writePendingLearning(projectDir string, edge session.SharpEdge) error {
	pendingPath := filepath.Join(config.ProjectMemoryDir(projectDir), "pending-learnings.jsonl")

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


// buildCombinedResponse merges sharp-edge failure info with attention-gate messages.
func buildCombinedResponse(failure *routing.FailureInfo, reminderMsg, flushMsg string) *routing.HookResponse {
	response := routing.NewPassResponse("PostToolUse")

	// Build context parts
	var contextParts []string

	// Note: FailureInfo doesn't have a Context field, so we only include attention-gate messages

	// Add reminder message
	if reminderMsg != "" {
		contextParts = append(contextParts, reminderMsg)
	}

	// Add flush message
	if flushMsg != "" {
		contextParts = append(contextParts, flushMsg)
	}

	// Combine all contexts
	if len(contextParts) > 0 {
		response.AddField("additionalContext", strings.Join(contextParts, "\n\n"))
	}

	return response
}

// buildWarningResponse creates a warning response at threshold-1, merged with attention-gate messages.
func buildWarningResponse(file string, count int, reminderMsg, flushMsg string) *routing.HookResponse {
	response := routing.NewPassResponse("PostToolUse")

	var contextParts []string

	// Add warning message
	warningMsg := fmt.Sprintf(
		"⚠️ WARNING: %d failures on '%s'. One more failure triggers sharp edge capture.",
		count, file,
	)
	contextParts = append(contextParts, warningMsg)

	// Add reminder message
	if reminderMsg != "" {
		contextParts = append(contextParts, reminderMsg)
	}

	// Add flush message
	if flushMsg != "" {
		contextParts = append(contextParts, flushMsg)
	}

	response.AddField("additionalContext", strings.Join(contextParts, "\n\n"))

	return response
}

// cleanupVitestProcesses kills lingering vitest processes after typescript-pro/react-pro agents complete.
// Vitest processes can persist after subagent termination, consuming CPU/RAM.
// This cleanup runs silently - errors are ignored as processes may already be dead.
func cleanupVitestProcesses(event *routing.PostToolEvent) {
	// Only check Bash tool invocations
	if event.ToolName != "Bash" {
		return
	}

	// Extract command from tool input
	commandRaw, ok := event.ToolInput["command"]
	if !ok {
		return
	}

	command, ok := commandRaw.(string)
	if !ok {
		return
	}

	// Check if command contains "vitest"
	if !strings.Contains(command, "vitest") {
		return
	}

	// Kill vitest processes (ignore errors - process may already be dead)
	cmd := exec.Command("pkill", "-f", "vitest")
	_ = cmd.Run()

	// Log cleanup for debugging
	fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Cleaned up vitest processes\n")
}

// detectSandboxBlock returns a suggestion message when a Write or Edit tool call
// targeting a .claude/ path appears to have been blocked by CC's sandbox.
// Returns an empty string when the event does not match the pattern.
func detectSandboxBlock(event *routing.PostToolEvent) string {
	// Only inspect Write and Edit tool calls.
	if event.ToolName != "Write" && event.ToolName != "Edit" {
		return ""
	}

	if event.ToolResponse == nil {
		return ""
	}

	// Extract file path from tool input.
	filePath := ""
	if fp, ok := event.ToolInput["file_path"].(string); ok {
		filePath = fp
	}
	if filePath == "" {
		return ""
	}

	// Only fire for paths under .claude/ or .goyoke/.
	if !strings.Contains(filePath, ".claude/") && !strings.Contains(filePath, "/.claude/") &&
		!strings.Contains(filePath, ".goyoke/") && !strings.Contains(filePath, "/.goyoke/") {
		return ""
	}

	// Detect an error/block in the tool response.
	isError := false
	if errFlag, ok := event.ToolResponse["is_error"].(bool); ok && errFlag {
		isError = true
	}
	if errStr, ok := event.ToolResponse["error"].(string); ok && errStr != "" {
		isError = true
	}
	if output, ok := event.ToolResponse["output"].(string); ok {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "sensitive") || strings.Contains(lower, "permission") {
			isError = true
		}
	}

	if !isError {
		return ""
	}

	return fmt.Sprintf(
		"[sandbox] Write blocked on sensitive path %q. Use the /sandbox skill or write the file manually.",
		filePath,
	)
}

// getProjectDir returns the project directory from environment or cwd.
// Priority: GOYOKE_PROJECT_DIR > CLAUDE_PROJECT_DIR > CWD
func getProjectDir() string {
	// 1. goYoke-specific override (highest priority)
	if dir := os.Getenv("GOYOKE_PROJECT_DIR"); dir != "" {
		return dir
	}
	// 2. Claude Code standard
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	// 3. Current working directory (fallback)
	dir, _ := os.Getwd()
	return dir
}
