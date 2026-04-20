// Package validate implements the goyoke-validate PreToolUse hook.
package validate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
)

const (
	defaultTimeout        = 5 * time.Second
	maxTaskNestingLevel   = 0 // Strict: Only Router (Level 0) can use Task()
)

// getLogPath returns the path for validation logs.
func getLogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "goyoke-fortress", "validate.log")
}

// logValidation writes a validation event to the log file for visibility.
func logValidation(sessionID, agent, model, decision, reason string) {
	logPath := getLogPath()

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02T15:04:05")
	shortSession := sessionID
	if len(sessionID) > 8 {
		shortSession = sessionID[:8]
	}

	entry := fmt.Sprintf("[%s] session=%s decision=%-5s agent=%-20s model=%-6s",
		timestamp, shortSession, decision, agent, model)
	if reason != "" {
		entry += fmt.Sprintf(" reason=%s", reason)
	}
	entry += "\n"

	f.WriteString(entry)
}

// Main is the entrypoint for the goyoke-validate hook.
func Main() {
	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	schema, err := routing.LoadSchema()
	if err != nil {
		outputError(fmt.Sprintf("Failed to load routing schema: %v", err))
		os.Exit(1)
	}

	event, err := parseEvent(os.Stdin, defaultTimeout)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	if event.ToolName != "Task" && event.ToolName != "Agent" {
		fmt.Println("{}")
		return
	}

	nestingLevel := routing.GetNestingLevel()
	isExplicit := routing.IsNestingLevelExplicit()

	if nestingLevel > maxTaskNestingLevel {
		if blockResponse := routing.ValidateTaskAtNestingLevel(nestingLevel, event.ToolInput); blockResponse != nil {
			model := "unknown"
			if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
				model = taskInput.Model
				if model == "" {
					model = "unspecified"
				}
			}

			logNestingBlock(event, nestingLevel, isExplicit, model)

			outputJSON(blockResponse)
			return
		}
	}

	var decisionID string
	if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
		decision := telemetry.NewRoutingDecision(
			event.SessionID,
			taskInput.Prompt,
			taskInput.Model,
			extractAgentFromPrompt(taskInput.Prompt),
		)
		decisionID = decision.DecisionID
		if logErr := telemetry.LogRoutingDecision(decision); logErr != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to log routing decision: %v\n", logErr)
		}

		lifecycle := telemetry.NewAgentLifecycleEvent(
			event.SessionID,
			"spawn",
			extractAgentFromPrompt(taskInput.Prompt),
			"terminal",
			taskInput.Model,
			taskInput.Prompt,
			decisionID,
		)
		if logErr := telemetry.LogAgentLifecycle(lifecycle); logErr != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to log lifecycle spawn: %v\n", logErr)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to parse Task input for routing decision: %v\n", err)
	}

	agentTaskNames := buildAgentTaskNames()

	orchestrator := routing.NewValidationOrchestrator(schema, projectDir, nil, agentTaskNames)

	result := orchestrator.ValidateTask(event.ToolInput, event.SessionID)

	agent := "unknown"
	model := "unknown"
	var taskInput *routing.TaskInput
	if ti, err := routing.ParseTaskInput(event.ToolInput); err == nil {
		taskInput = ti
		agent = extractAgentFromPrompt(taskInput.Prompt)
		if agent == "unknown" && taskInput.Resume != "" {
			agent = "resumed:" + taskInput.Resume[:min(len(taskInput.Resume), 8)]
		}
		model = taskInput.Model
		if model == "" {
			model = "default"
		}
	}

	if (result.Decision == "allow" || result.Decision == "") && taskInput != nil {
		if agent != "" && agent != "unknown" {
			agentConfig, err := loadAgentConfig(agent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to load agent config: %v\n", err)
			} else if agentConfig != nil {
				taskFiles := routing.ExtractFilesFromPrompt(taskInput.Prompt)

				augmentedPrompt, err := routing.BuildFullAgentContext(
					agent,
					agentConfig.ContextRequirements,
					taskFiles,
					taskInput.Prompt,
				)

				if err != nil {
					fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to build augmented prompt: %v\n", err)
				} else if augmentedPrompt != taskInput.Prompt {
					updatedInput := map[string]any{
						"prompt":        augmentedPrompt,
						"model":         taskInput.Model,
						"subagent_type": taskInput.SubagentType,
						"description":   taskInput.Description,
					}

					if taskInput.MaxTurns > 0 {
						updatedInput["max_turns"] = taskInput.MaxTurns
					}
					if taskInput.RunInBackground {
						updatedInput["run_in_background"] = taskInput.RunInBackground
					}

					resp := routing.NewModifyResponse("PreToolUse", updatedInput)
					if err := resp.Marshal(os.Stdout); err != nil {
						fmt.Fprintf(os.Stderr, "[goyoke-validate] Error: Failed to marshal response: %v\n", err)
						os.Exit(1)
					}
					fmt.Println()

					logValidation(event.SessionID, agent, model, "modify", "conventions injected")
					return
				}
			}
		}
	}

	logValidation(event.SessionID, agent, model, result.Decision, result.Reason)

	outputResult(result, event.SessionID)

	for _, violation := range result.Violations {
		if err := routing.LogViolation(violation, projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-validate] Warning: Failed to log violation: %v\n", err)
		}
	}
}

// parseEvent reads and parses ToolEvent from STDIN with timeout.
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

// outputResult writes validation result as JSON to STDOUT.
func outputResult(result *routing.ValidationResult, _ string) {
	output := make(map[string]any)

	output["decision"] = result.Decision

	if result.Decision == "block" {
		output["reason"] = result.Reason
		output["hookSpecificOutput"] = map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": result.Reason,
		}
	} else {
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

// outputError writes error message in hook format.
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

func readMergedAgentsIndex() ([]byte, error) {
	r, err := resolve.NewFromEnv()
	if err != nil {
		return nil, err
	}
	results, err := r.ReadFileAll("agents/agents-index.json")
	if err != nil {
		return nil, err
	}
	switch len(results) {
	case 1:
		return results[0], nil
	case 2:
		return resolve.MergeAgentIndexJSON(results[1], results[0])
	default:
		return nil, fmt.Errorf("unexpected number of agents-index.json layers: %d", len(results))
	}
}

func buildAgentTaskNames() map[string]string {
	data, err := readMergedAgentsIndex()
	if err != nil {
		return nil
	}

	var index struct {
		Agents []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil
	}

	result := make(map[string]string)
	for _, agent := range index.Agents {
		if agent.ID != "" && agent.Name != "" {
			result[agent.ID] = agent.Name
		}
	}
	return result
}

func loadAgentConfig(agentID string) (*routing.AgentConfig, error) {
	data, err := readMergedAgentsIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to read agents-index.json: %w", err)
	}

	var index struct {
		Agents []struct {
			ID                  string                       `json:"id"`
			Name                string                       `json:"name"`
			Model               string                       `json:"model"`
			SubagentType        string                       `json:"subagent_type"`
			ContextRequirements *routing.ContextRequirements `json:"context_requirements"`
			CliFlags            *routing.AgentCliFlags       `json:"cli_flags,omitempty"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse agents-index.json: %w", err)
	}

	for _, agent := range index.Agents {
		if agent.ID == agentID {
			return &routing.AgentConfig{
				Name:                agent.Name,
				Model:               agent.Model,
				SubagentType:        agent.SubagentType,
				ContextRequirements: agent.ContextRequirements,
				CliFlags:            agent.CliFlags,
			}, nil
		}
	}

	return nil, nil
}

// logNestingBlock logs Task() blocks due to nesting level for telemetry.
func logNestingBlock(event *routing.ToolEvent, level int, explicit bool, model string) {
	telemetryData := map[string]any{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"event":          "task_blocked_nesting",
		"session_id":     event.SessionID,
		"nesting_level":  level,
		"level_explicit": explicit,
		"tool_name":      event.ToolName,
		"model":          model,
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	telemetryPath := filepath.Join(dataHome, "goyoke", "nesting-blocks.jsonl")

	appendJSONL(telemetryPath, telemetryData)
}

// appendJSONL appends a JSON object to a JSONL file.
func appendJSONL(path string, data any) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.Encode(data)
}

// outputJSON writes a map as formatted JSON to stdout.
func outputJSON(data map[string]any) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
