// Package sharpedge implements the goyoke-sharp-edge PostToolUse hook.
package sharpedge

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
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
)

const defaultTimeout = 5 * time.Second

// getAgentIDs returns agent IDs discovered via the Resolver union layer.
func getAgentIDs() []string {
	r, err := resolve.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: could not create resolver: %v\n", err)
		return nil
	}

	entries, err := r.ReadDir("agents")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: could not read agents dir: %v\n", err)
		return nil
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		ids = append(ids, name)
	}

	fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Loaded %d agent IDs\n", len(ids))
	return ids
}

// Main is the entrypoint for the goyoke-sharp-edge hook.
func Main() {
	projectDir := getProjectDir()

	agentIDs := getAgentIDs()
	index, err := memory.LoadSharpEdgesIndex(agentIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to load sharp edges index: %v\n", err)
		index = &memory.SharpEdgeIndex{}
	}

	toolCount, counterErr := config.GetToolCountAndIncrement()
	if counterErr != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: counter error: %v\n", counterErr)
		toolCount = 0
	}

	event, err := routing.ParsePostToolEvent(os.Stdin, defaultTimeout)
	if err != nil {
		fmt.Println("{}")
		return
	}

	cleanupVitestProcesses(event)

	if mlErr := telemetry.LogMLToolEvent(event, projectDir); mlErr != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] ML logging warning: %v\n", mlErr)
	}

	var reminderMsg string
	if config.ShouldRemind(toolCount) {
		summary := "See ~/.claude/routing-schema.json for tier mappings"
		reminderMsg = session.GenerateRoutingReminder(toolCount, summary)
	}

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

	if sandboxMsg := detectSandboxBlock(event); sandboxMsg != "" {
		resp := buildCombinedResponse(nil, sandboxMsg, "")
		if reminderMsg != "" || flushMsg != "" {
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

	failure := routing.DetectFailure(event)

	if failure == nil {
		if reminderMsg == "" && flushMsg == "" {
			fmt.Println("{}")
			return
		}
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	if err := memory.LogFailure(failure); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to log failure: %v\n", err)
	}

	count, err := memory.GetFailureCount(failure.File, failure.ErrorType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to get failure count: %v\n", err)
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	threshold := memory.DefaultMaxFailures

	if count >= threshold {
		attemptedChange := session.ExtractAttemptedChange(event)

		codeSnippet := ""
		if failure.File != "unknown" && failure.File != "" {
			snippet, _ := session.ExtractCodeSnippet(failure.File, 0, 2)
			codeSnippet = snippet
		}

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

		if err := writePendingLearning(projectDir, edge); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Warning: Failed to write learning: %v\n", err)
		}

		errorMsg := failure.ErrorMatch
		if errorMsg == "" {
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

		resp := memory.GenerateBlockingResponse(patternEdge, index, count)

		if reminderMsg != "" || flushMsg != "" {
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

		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	if count == threshold-1 {
		resp := buildWarningResponse(failure.File, count, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	if reminderMsg != "" || flushMsg != "" {
		resp := buildCombinedResponse(nil, reminderMsg, flushMsg)
		if err := resp.Marshal(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Error marshaling response: %v\n", err)
			fmt.Println("{}")
		}
		return
	}

	fmt.Println("{}")
}

// writePendingLearning appends a SharpEdge to pending-learnings.jsonl.
func writePendingLearning(projectDir string, edge session.SharpEdge) error {
	pendingPath := filepath.Join(config.ProjectMemoryDir(projectDir), "pending-learnings.jsonl")

	if err := os.MkdirAll(filepath.Dir(pendingPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("marshal edge: %w", err)
	}

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
func buildCombinedResponse(_ *routing.FailureInfo, reminderMsg, flushMsg string) *routing.HookResponse {
	response := routing.NewPassResponse("PostToolUse")

	var contextParts []string

	if reminderMsg != "" {
		contextParts = append(contextParts, reminderMsg)
	}

	if flushMsg != "" {
		contextParts = append(contextParts, flushMsg)
	}

	if len(contextParts) > 0 {
		response.AddField("additionalContext", strings.Join(contextParts, "\n\n"))
	}

	return response
}

// buildWarningResponse creates a warning response at threshold-1.
func buildWarningResponse(file string, count int, reminderMsg, flushMsg string) *routing.HookResponse {
	response := routing.NewPassResponse("PostToolUse")

	var contextParts []string

	warningMsg := fmt.Sprintf(
		"⚠️ WARNING: %d failures on '%s'. One more failure triggers sharp edge capture.",
		count, file,
	)
	contextParts = append(contextParts, warningMsg)

	if reminderMsg != "" {
		contextParts = append(contextParts, reminderMsg)
	}

	if flushMsg != "" {
		contextParts = append(contextParts, flushMsg)
	}

	response.AddField("additionalContext", strings.Join(contextParts, "\n\n"))

	return response
}

// cleanupVitestProcesses kills lingering vitest processes after agent completion.
func cleanupVitestProcesses(event *routing.PostToolEvent) {
	if event.ToolName != "Bash" {
		return
	}

	commandRaw, ok := event.ToolInput["command"]
	if !ok {
		return
	}

	command, ok := commandRaw.(string)
	if !ok {
		return
	}

	if !strings.Contains(command, "vitest") {
		return
	}

	cmd := exec.Command("pkill", "-f", "vitest")
	_ = cmd.Run()

	fmt.Fprintf(os.Stderr, "[goyoke-sharp-edge] Cleaned up vitest processes\n")
}

// detectSandboxBlock returns a suggestion message when Write/Edit to .claude/ path is blocked.
func detectSandboxBlock(event *routing.PostToolEvent) string {
	if event.ToolName != "Write" && event.ToolName != "Edit" {
		return ""
	}

	if event.ToolResponse == nil {
		return ""
	}

	filePath := ""
	if fp, ok := event.ToolInput["file_path"].(string); ok {
		filePath = fp
	}
	if filePath == "" {
		return ""
	}

	if !strings.Contains(filePath, ".claude/") && !strings.Contains(filePath, "/.claude/") &&
		!strings.Contains(filePath, ".goyoke/") && !strings.Contains(filePath, "/.goyoke/") {
		return ""
	}

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
func getProjectDir() string {
	if dir := os.Getenv("GOYOKE_PROJECT_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	dir, _ := os.Getwd()
	return dir
}
