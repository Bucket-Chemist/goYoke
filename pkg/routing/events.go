package routing

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ToolEvent represents PreToolUse events from Claude Code hooks.
// These events are emitted before a tool is invoked.
type ToolEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`
}

// PostToolEvent represents PostToolUse events with execution results.
// These events include both the input and the tool's response.
type PostToolEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  map[string]interface{} `json:"tool_response"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`
}

// VALIDATION NOTES (GOgent-006):
//
// Struct validated against 100+ real production events from event-corpus.json.
// All events conform to this structure. Key validation findings:
//
// 1. CWD Field: Not present in any corpus events. Not added to struct.
// 2. Timestamp: All events use "captured_at" (Unix epoch int64). No alternatives found.
// 3. ToolInput: Always a JSON object (map[string]interface{}). Never null or string.
// 4. Field Visibility: All 5 fields required and present in 100% of events.
// 5. PostToolUse: Adds tool_response field (map[string]interface{}) to base structure.
//
// Corpus coverage: Task, Read, Write, Edit, Bash, Glob, Grep tools.
// Event types: PreToolUse, PostToolUse.
// Validation date: 2026-01-16

// TaskInput represents Task tool_input structure.
// This is the specific structure for Task tool invocations.
type TaskInput struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
	Description  string `json:"description"`
}

// ParseToolEvent reads JSON from io.Reader and parses into ToolEvent.
// Returns an error if JSON parsing fails or required fields are missing.
// Uses ReadStdin for timeout protection.
func ParseToolEvent(r io.Reader, timeout time.Duration) (*ToolEvent, error) {
	data, err := ReadStdin(r, timeout)
	if err != nil {
		return nil, err
	}

	var event ToolEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse JSON: %w. Input: %s. Ensure hook receives valid JSON from Claude Code.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required fields
	if event.ToolName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.HookEventName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing hook_event_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	return &event, nil
}

// ParsePostToolEvent reads and parses PostToolUse events.
// Returns an error if JSON parsing fails or required fields are missing.
// Uses ReadStdin for timeout protection.
func ParsePostToolEvent(r io.Reader, timeout time.Duration) (*PostToolEvent, error) {
	data, err := ReadStdin(r, timeout)
	if err != nil {
		return nil, err
	}

	var event PostToolEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse JSON: %w. Input: %s. Ensure hook receives valid JSON from Claude Code.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required fields
	if event.ToolName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.HookEventName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing hook_event_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.ToolResponse == nil {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_response. PostToolUse event incomplete: %s. Ensure hook emits complete PostToolEvent structure.",
			truncate(data, 200),
		)
	}

	return &event, nil
}

// ParseTaskInput extracts Task parameters from tool_input map.
// Returns an error if the prompt field is missing (required).
// Other fields are optional and may be empty strings.
func ParseTaskInput(toolInput map[string]interface{}) (*TaskInput, error) {
	// Marshal to JSON bytes
	data, err := json.Marshal(toolInput)
	if err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to marshal tool_input: %w. Input may contain non-serializable types.",
			err,
		)
	}

	// Unmarshal to TaskInput struct
	var taskInput TaskInput
	if err := json.Unmarshal(data, &taskInput); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse Task tool_input: %w. Input: %s. Ensure Task tool_input follows expected schema.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required field
	if taskInput.Prompt == "" {
		return nil, fmt.Errorf(
			"[event-parser] Task input missing required field 'prompt'. Input: %s. Task invocations must include prompt.",
			truncate(data, 200),
		)
	}

	return &taskInput, nil
}

// truncate limits data to maxLen for error messages.
// Appends "... (truncated)" if data exceeds maxLen.
func truncate(data []byte, maxLen int) string {
	if len(data) <= maxLen {
		return string(data)
	}
	return string(data[:maxLen]) + "... (truncated)"
}
