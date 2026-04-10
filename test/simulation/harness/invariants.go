package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Invariant represents a property that must always hold.
type Invariant struct {
	ID    string
	Name  string
	Check func(input interface{}, output string, exitCode int, tempDir string) (bool, string)
}

// InvariantResult captures the outcome of checking an invariant.
type InvariantResult struct {
	InvariantID string `json:"invariant_id"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message,omitempty"`
	Input       string `json:"input,omitempty"`
}

// PreToolUseInvariants defines properties for gogent-validate.
var PreToolUseInvariants = []Invariant{
	{
		ID:   "P1",
		Name: "never_crash",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			if exitCode != 0 {
				return false, "exit code was non-zero"
			}
			return true, ""
		},
	},
	{
		ID:   "P2",
		Name: "valid_json_output",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			// Empty output is valid (pass-through case)
			if output == "" || output == "{}" {
				return true, ""
			}

			var parsed interface{}
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				return false, "output is not valid JSON: " + err.Error()
			}
			return true, ""
		},
	},
	{
		ID:   "P3",
		Name: "non_task_passthrough",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			// Check if input is a non-Task tool
			inputMap, ok := input.(map[string]interface{})
			if !ok {
				return true, "" // Can't determine, assume pass
			}

			toolName, _ := inputMap["tool_name"].(string)
			if toolName == "Task" {
				return true, "" // This invariant doesn't apply to Task
			}

			// Non-Task should return empty or minimal output
			if output != "" && output != "{}" && output != "{}\n" {
				return false, "non-Task tool should return empty/minimal output"
			}
			return true, ""
		},
	},
	{
		ID:   "P4",
		Name: "opus_always_blocked",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			// Check if input is Task with model=opus
			inputMap, ok := input.(map[string]interface{})
			if !ok {
				return true, ""
			}

			toolName, _ := inputMap["tool_name"].(string)
			if toolName != "Task" {
				return true, ""
			}

			toolInput, ok := inputMap["tool_input"].(map[string]interface{})
			if !ok {
				return true, ""
			}

			model, _ := toolInput["model"].(string)
			if model != "opus" {
				return true, ""
			}

			// Opus should be blocked
			var outputMap map[string]interface{}
			if err := json.Unmarshal([]byte(output), &outputMap); err != nil {
				return false, "cannot parse output for opus check"
			}

			decision, _ := outputMap["decision"].(string)
			if decision != "block" {
				return false, "opus model was not blocked"
			}
			return true, ""
		},
	},
	{
		ID:   "P5",
		Name: "decision_is_allow_or_block",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			// Only applies to Task tools
			inputMap, ok := input.(map[string]interface{})
			if !ok {
				return true, ""
			}

			toolName, _ := inputMap["tool_name"].(string)
			if toolName != "Task" {
				return true, ""
			}

			if output == "" || output == "{}" {
				return true, "" // No decision is valid for pass-through
			}

			var outputMap map[string]interface{}
			if err := json.Unmarshal([]byte(output), &outputMap); err != nil {
				return true, "" // Already checked by P2
			}

			decision, exists := outputMap["decision"].(string)
			if !exists {
				return true, "" // No decision field is allowed
			}

			if decision != "allow" && decision != "block" {
				return false, "decision must be 'allow' or 'block', got: " + decision
			}
			return true, ""
		},
	},
}

// SessionEndInvariants defines properties for gogent-archive.
var SessionEndInvariants = []Invariant{
	{
		ID:   "S1",
		Name: "never_crash",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			if exitCode != 0 {
				return false, "exit code was non-zero"
			}
			return true, ""
		},
	},
	{
		ID:   "S2",
		Name: "handoff_created",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			handoffPath := filepath.Join(tempDir, ".gogent", "memory", "handoffs.jsonl")
			if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
				return false, "handoffs.jsonl was not created"
			}
			return true, ""
		},
	},
	{
		ID:   "S3",
		Name: "schema_version_current",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			handoffPath := filepath.Join(tempDir, ".gogent", "memory", "handoffs.jsonl")
			data, err := os.ReadFile(handoffPath)
			if err != nil {
				return false, "cannot read handoffs.jsonl: " + err.Error()
			}

			// Read last line (most recent handoff)
			lines := splitLines(string(data))
			if len(lines) == 0 {
				return false, "handoffs.jsonl is empty"
			}

			lastLine := lines[len(lines)-1]
			var handoff map[string]interface{}
			if err := json.Unmarshal([]byte(lastLine), &handoff); err != nil {
				return false, "cannot parse last handoff: " + err.Error()
			}

			version, _ := handoff["schema_version"].(string)
			if version != "1.3" {
				return false, "schema_version is not 1.3, got: " + version
			}
			return true, ""
		},
	},
	{
		ID:   "S4",
		Name: "markdown_created",
		Check: func(input interface{}, output string, exitCode int, tempDir string) (bool, string) {
			mdPath := filepath.Join(tempDir, ".gogent", "memory", "last-handoff.md")
			if _, err := os.Stat(mdPath); os.IsNotExist(err) {
				return false, "last-handoff.md was not created"
			}
			return true, ""
		},
	},
}

// CheckInvariants checks all invariants in a list against given execution.
func CheckInvariants(invariants []Invariant, input interface{}, output string, exitCode int, tempDir string) []InvariantResult {
	var results []InvariantResult

	inputJSON, _ := json.Marshal(input)

	for _, inv := range invariants {
		passed, message := inv.Check(input, output, exitCode, tempDir)
		results = append(results, InvariantResult{
			InvariantID: inv.ID,
			Passed:      passed,
			Message:     message,
			Input:       string(inputJSON),
		})
	}

	return results
}

// AllPassed returns true if all invariant results passed.
func AllPassed(results []InvariantResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// FailedInvariants returns only the failed invariant results.
func FailedInvariants(results []InvariantResult) []InvariantResult {
	var failed []InvariantResult
	for _, r := range results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}
	return failed
}

// splitLines splits string into lines, removing empty trailing line.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
