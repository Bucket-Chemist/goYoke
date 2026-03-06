package routing

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test ParseToolEvent with valid PreToolUse events
func TestParseToolEvent_ValidEvents(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedTool  string
		expectedEvent string
	}{
		{
			name: "Task tool",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "haiku",
					"prompt": "AGENT: codebase-search\n\nFind files",
					"subagent_type": "Explore",
					"description": "Search codebase"
				},
				"session_id": "test-session-1",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465005
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Read tool",
			json: `{
				"tool_name": "Read",
				"tool_input": {
					"file_path": "/home/user/project/src/module.py"
				},
				"session_id": "test-session-read",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465029
			}`,
			expectedTool:  "Read",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Write tool",
			json: `{
				"tool_name": "Write",
				"tool_input": {
					"file_path": "/home/user/project/src/new_file.py",
					"content": "# Module\npass"
				},
				"session_id": "test-session-write",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465049
			}`,
			expectedTool:  "Write",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Edit tool",
			json: `{
				"tool_name": "Edit",
				"tool_input": {
					"file_path": "/home/user/project/src/edit_file.py",
					"old_string": "old_code",
					"new_string": "new_code"
				},
				"session_id": "test-session-edit",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465064
			}`,
			expectedTool:  "Edit",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Bash tool",
			json: `{
				"tool_name": "Bash",
				"tool_input": {
					"command": "pytest tests/test_1.py"
				},
				"session_id": "test-session-bash",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465079
			}`,
			expectedTool:  "Bash",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Glob tool",
			json: `{
				"tool_name": "Glob",
				"tool_input": {
					"pattern": "**/*.py"
				},
				"session_id": "test-session-glob",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465089
			}`,
			expectedTool:  "Glob",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Grep tool",
			json: `{
				"tool_name": "Grep",
				"tool_input": {
					"pattern": "TODO|FIXME|XXX"
				},
				"session_id": "test-session-grep",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465099
			}`,
			expectedTool:  "Grep",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with sonnet model",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: python-pro\n\nImplement function",
					"subagent_type": "general-purpose",
					"description": "Python implementation"
				},
				"session_id": "test-session-sonnet",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465010
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with opus model",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "opus",
					"prompt": "AGENT: einstein\n\nDeep analysis",
					"subagent_type": "general-purpose",
					"description": "Opus task"
				},
				"session_id": "test-session-opus",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465022
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with Plan subagent_type",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "opus",
					"prompt": "AGENT: architect\n\nArchitecture design",
					"subagent_type": "Plan",
					"description": "Ceiling test above"
				},
				"session_id": "test-ceiling",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465026
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with general-purpose subagent_type",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: python-pro\n\nImplement feature",
					"subagent_type": "general-purpose",
					"description": "Ceiling test below"
				},
				"session_id": "test-ceiling-gp",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465025
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with Explore subagent_type",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: tech-docs-writer\n\nWrite documentation",
					"subagent_type": "Explore",
					"description": "Invalid subagent_type test"
				},
				"session_id": "test-subagent",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465027
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with force-tier override",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: python-pro\n\n--force-tier=haiku\n\nSimple function",
					"subagent_type": "general-purpose",
					"description": "Override tier test"
				},
				"session_id": "test-override",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465020
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "Task with force-delegation override",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: orchestrator\n\n--force-delegation=opus\n\nComplex planning",
					"subagent_type": "Plan",
					"description": "Override delegation test"
				},
				"session_id": "test-override-del",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465021
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
		{
			name: "R implementation task",
			json: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: r-pro\n\nImplement R function",
					"subagent_type": "general-purpose",
					"description": "R implementation"
				},
				"session_id": "test-r",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465104
			}`,
			expectedTool:  "Task",
			expectedEvent: "PreToolUse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			event, err := ParseToolEvent(reader, 1*time.Second)

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if event.ToolName != tc.expectedTool {
				t.Errorf("expected tool_name %q, got %q", tc.expectedTool, event.ToolName)
			}

			if event.HookEventName != tc.expectedEvent {
				t.Errorf("expected hook_event_name %q, got %q", tc.expectedEvent, event.HookEventName)
			}

			if event.ToolInput == nil {
				t.Error("expected tool_input to be non-nil")
			}

			if event.SessionID == "" {
				t.Error("expected session_id to be non-empty")
			}

			if event.CapturedAt == 0 {
				t.Error("expected captured_at to be non-zero")
			}
		})
	}
}

// Test ParsePostToolEvent with valid PostToolUse events
func TestParsePostToolEvent_ValidEvents(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		expectedTool string
		checkExit    bool
		expectedExit int
	}{
		{
			name: "Bash success",
			json: `{
				"tool_name": "Bash",
				"tool_response": {
					"exit_code": 0,
					"stdout": "Test passed",
					"stderr": ""
				},
				"tool_input": {
					"command": "pytest tests/test_1.py"
				},
				"session_id": "test-bash-post-1",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465084
			}`,
			expectedTool: "Bash",
			checkExit:    true,
			expectedExit: 0,
		},
		{
			name: "Bash failure",
			json: `{
				"tool_name": "Bash",
				"tool_response": {
					"exit_code": 1,
					"stdout": "",
					"stderr": "Error: Test failed"
				},
				"tool_input": {
					"command": "pytest tests/test_2.py"
				},
				"session_id": "test-bash-post-2",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465085
			}`,
			expectedTool: "Bash",
			checkExit:    true,
			expectedExit: 1,
		},
		{
			name: "Read tool response",
			json: `{
				"tool_name": "Read",
				"tool_response": {
					"content": "file content here",
					"lines": 42
				},
				"tool_input": {
					"file_path": "/home/user/project/src/module.py"
				},
				"session_id": "test-read-post",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465200
			}`,
			expectedTool: "Read",
			checkExit:    false,
		},
		{
			name: "Write tool response",
			json: `{
				"tool_name": "Write",
				"tool_response": {
					"success": true,
					"bytes_written": 256
				},
				"tool_input": {
					"file_path": "/home/user/project/src/new_file.py",
					"content": "# Module\npass"
				},
				"session_id": "test-write-post",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465201
			}`,
			expectedTool: "Write",
			checkExit:    false,
		},
		{
			name: "Task tool response",
			json: `{
				"tool_name": "Task",
				"tool_response": {
					"result": "completed",
					"output": "Implementation complete"
				},
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: python-pro\n\nImplement function",
					"subagent_type": "general-purpose",
					"description": "Python implementation"
				},
				"session_id": "test-task-post",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465202
			}`,
			expectedTool: "Task",
			checkExit:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			event, err := ParsePostToolEvent(reader, 1*time.Second)

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if event.ToolName != tc.expectedTool {
				t.Errorf("expected tool_name %q, got %q", tc.expectedTool, event.ToolName)
			}

			if event.HookEventName != "PostToolUse" {
				t.Errorf("expected hook_event_name %q, got %q", "PostToolUse", event.HookEventName)
			}

			if event.ToolInput == nil {
				t.Error("expected tool_input to be non-nil")
			}

			if event.ToolResponse == nil {
				t.Error("expected tool_response to be non-nil")
			}

			if tc.checkExit {
				exitCode, ok := event.ToolResponse["exit_code"]
				if !ok {
					t.Error("expected exit_code in tool_response")
				}
				// JSON numbers unmarshal as float64
				if int(exitCode.(float64)) != tc.expectedExit {
					t.Errorf("expected exit_code %d, got %v", tc.expectedExit, exitCode)
				}
			}
		})
	}
}

// Test error cases for ParseToolEvent
func TestParseToolEvent_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedError string
	}{
		{
			name:          "malformed JSON",
			json:          `{"tool_name": "Task", "invalid json`,
			expectedError: "[event-parser] Failed to parse JSON",
		},
		{
			name: "missing tool_name",
			json: `{
				"tool_input": {"model": "haiku"},
				"session_id": "test",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465005
			}`,
			expectedError: "[event-parser] Missing tool_name",
		},
		{
			name: "missing hook_event_name",
			json: `{
				"tool_name": "Task",
				"tool_input": {"model": "haiku"},
				"session_id": "test",
				"captured_at": 1768465005
			}`,
			expectedError: "[event-parser] Missing hook_event_name",
		},
		{
			name: "empty tool_name",
			json: `{
				"tool_name": "",
				"tool_input": {"model": "haiku"},
				"session_id": "test",
				"hook_event_name": "PreToolUse",
				"captured_at": 1768465005
			}`,
			expectedError: "[event-parser] Missing tool_name",
		},
		{
			name: "empty hook_event_name",
			json: `{
				"tool_name": "Task",
				"tool_input": {"model": "haiku"},
				"session_id": "test",
				"hook_event_name": "",
				"captured_at": 1768465005
			}`,
			expectedError: "[event-parser] Missing hook_event_name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			event, err := ParseToolEvent(reader, 1*time.Second)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if event != nil {
				t.Errorf("expected nil event, got: %v", event)
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error containing %q, got: %v", tc.expectedError, err)
			}

			if !strings.Contains(err.Error(), "[event-parser]") {
				t.Errorf("error should have [event-parser] prefix, got: %v", err)
			}
		})
	}
}

// Test error cases for ParsePostToolEvent
func TestParsePostToolEvent_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedError string
	}{
		{
			name:          "malformed JSON",
			json:          `{"tool_name": "Bash", "invalid`,
			expectedError: "[event-parser] Failed to parse JSON",
		},
		{
			name: "missing tool_response",
			json: `{
				"tool_name": "Bash",
				"tool_input": {"command": "test"},
				"session_id": "test",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465084
			}`,
			expectedError: "[event-parser] Missing tool_response",
		},
		{
			name: "missing tool_name",
			json: `{
				"tool_response": {"exit_code": 0},
				"tool_input": {"command": "test"},
				"session_id": "test",
				"hook_event_name": "PostToolUse",
				"captured_at": 1768465084
			}`,
			expectedError: "[event-parser] Missing tool_name",
		},
		{
			name: "missing hook_event_name",
			json: `{
				"tool_name": "Bash",
				"tool_response": {"exit_code": 0},
				"tool_input": {"command": "test"},
				"session_id": "test",
				"captured_at": 1768465084
			}`,
			expectedError: "[event-parser] Missing hook_event_name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			event, err := ParsePostToolEvent(reader, 1*time.Second)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if event != nil {
				t.Errorf("expected nil event, got: %v", event)
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error containing %q, got: %v", tc.expectedError, err)
			}
		})
	}
}

// Test ParseTaskInput with various Task structures
func TestParseTaskInput_ValidCases(t *testing.T) {
	tests := []struct {
		name              string
		toolInput         map[string]interface{}
		expectedModel     string
		expectedSubagent  string
		expectedPromptLen int
	}{
		{
			name: "complete Task input",
			toolInput: map[string]interface{}{
				"model":         "haiku",
				"prompt":        "AGENT: codebase-search\n\nFind files",
				"subagent_type": "Explore",
				"description":   "Search codebase",
			},
			expectedModel:     "haiku",
			expectedSubagent:  "Explore",
			expectedPromptLen: 34,
		},
		{
			name: "sonnet model",
			toolInput: map[string]interface{}{
				"model":         "sonnet",
				"prompt":        "AGENT: python-pro\n\nImplement function",
				"subagent_type": "general-purpose",
				"description":   "Python implementation",
			},
			expectedModel:     "sonnet",
			expectedSubagent:  "general-purpose",
			expectedPromptLen: 37,
		},
		{
			name: "opus model",
			toolInput: map[string]interface{}{
				"model":         "opus",
				"prompt":        "AGENT: einstein\n\nDeep analysis",
				"subagent_type": "general-purpose",
				"description":   "Opus task",
			},
			expectedModel:     "opus",
			expectedSubagent:  "general-purpose",
			expectedPromptLen: 30,
		},
		{
			name: "Plan subagent_type",
			toolInput: map[string]interface{}{
				"model":         "opus",
				"prompt":        "AGENT: architect\n\nArchitecture design",
				"subagent_type": "Plan",
				"description":   "Architecture planning",
			},
			expectedModel:     "opus",
			expectedSubagent:  "Plan",
			expectedPromptLen: 37,
		},
		{
			name: "minimal Task input with only prompt",
			toolInput: map[string]interface{}{
				"prompt": "Some prompt text",
			},
			expectedModel:     "",
			expectedSubagent:  "",
			expectedPromptLen: 16,
		},
		{
			name: "Task with force-tier override",
			toolInput: map[string]interface{}{
				"model":         "sonnet",
				"prompt":        "AGENT: python-pro\n\n--force-tier=haiku\n\nSimple function",
				"subagent_type": "general-purpose",
				"description":   "Override tier test",
			},
			expectedModel:     "sonnet",
			expectedSubagent:  "general-purpose",
			expectedPromptLen: 54,
		},
		{
			name: "Task with force-delegation override",
			toolInput: map[string]interface{}{
				"model":         "sonnet",
				"prompt":        "AGENT: orchestrator\n\n--force-delegation=opus\n\nComplex planning",
				"subagent_type": "Plan",
				"description":   "Override delegation test",
			},
			expectedModel:     "sonnet",
			expectedSubagent:  "Plan",
			expectedPromptLen: 62,
		},
		{
			name: "long prompt",
			toolInput: map[string]interface{}{
				"model":         "sonnet",
				"prompt":        strings.Repeat("A", 5000),
				"subagent_type": "general-purpose",
				"description":   "Large prompt test",
			},
			expectedModel:     "sonnet",
			expectedSubagent:  "general-purpose",
			expectedPromptLen: 5000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			taskInput, err := ParseTaskInput(tc.toolInput)

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if taskInput.Model != tc.expectedModel {
				t.Errorf("expected model %q, got %q", tc.expectedModel, taskInput.Model)
			}

			if taskInput.SubagentType != tc.expectedSubagent {
				t.Errorf("expected subagent_type %q, got %q", tc.expectedSubagent, taskInput.SubagentType)
			}

			if len(taskInput.Prompt) != tc.expectedPromptLen {
				t.Errorf("expected prompt length %d, got %d", tc.expectedPromptLen, len(taskInput.Prompt))
			}
		})
	}
}

// Test ParseTaskInput error cases
func TestParseTaskInput_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		toolInput     map[string]interface{}
		expectedError string
	}{
		{
			name: "missing prompt",
			toolInput: map[string]interface{}{
				"model":         "haiku",
				"subagent_type": "Explore",
				"description":   "No prompt",
			},
			expectedError: "[event-parser] Task input missing required field 'prompt'",
		},
		{
			name: "empty prompt",
			toolInput: map[string]interface{}{
				"model":         "haiku",
				"prompt":        "",
				"subagent_type": "Explore",
				"description":   "Empty prompt",
			},
			expectedError: "[event-parser] Task input missing required field 'prompt'",
		},
		{
			name: "nil tool_input",
			toolInput: map[string]interface{}{
				"model": "haiku",
			},
			expectedError: "[event-parser] Task input missing required field 'prompt'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			taskInput, err := ParseTaskInput(tc.toolInput)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if taskInput != nil {
				t.Errorf("expected nil taskInput, got: %v", taskInput)
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error containing %q, got: %v", tc.expectedError, err)
			}
		})
	}
}

// Test timeout integration with ParseToolEvent
func TestParseToolEvent_Timeout(t *testing.T) {
	reader := &slowReader{
		delay: 2 * time.Second,
		data:  `{"tool_name":"Task"}`,
	}

	event, err := ParseToolEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if event != nil {
		t.Errorf("expected nil event, got: %v", event)
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

// Test timeout integration with ParsePostToolEvent
func TestParsePostToolEvent_Timeout(t *testing.T) {
	reader := &slowReader{
		delay: 2 * time.Second,
		data:  `{"tool_name":"Bash","tool_response":{}}`,
	}

	event, err := ParsePostToolEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if event != nil {
		t.Errorf("expected nil event, got: %v", event)
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

// Test empty input handling
func TestParseToolEvent_EmptyInput(t *testing.T) {
	reader := &immediateEOFReader{}

	event, err := ParseToolEvent(reader, 1*time.Second)

	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}

	if event != nil {
		t.Errorf("expected nil event, got: %v", event)
	}

	if !strings.Contains(err.Error(), "No data received") {
		t.Errorf("expected 'No data received' error, got: %v", err)
	}
}

// Test truncate function
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		maxLen   int
		expected string
	}{
		{
			name:     "data shorter than maxLen",
			data:     []byte("short"),
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "data equal to maxLen",
			data:     []byte("exactly10!"),
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "data longer than maxLen",
			data:     []byte("this is a very long string that should be truncated"),
			maxLen:   20,
			expected: "this is a very long ... (truncated)",
		},
		{
			name:     "empty data",
			data:     []byte(""),
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen zero",
			data:     []byte("data"),
			maxLen:   0,
			expected: "... (truncated)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncate(tc.data, tc.maxLen)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// CRITICAL TEST: Corpus Replay
// This test validates parsing against ALL 100+ real production events from the corpus.
func TestParseEventCorpus(t *testing.T) {
	// Read corpus file
	corpusPath := "../../test/fixtures/event-corpus.json"
	data, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Fatalf("failed to read corpus file: %v", err)
	}

	// Parse as JSON array
	var events []map[string]interface{}
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("failed to parse corpus JSON: %v", err)
	}

	t.Logf("Corpus contains %d events", len(events))

	preToolCount := 0
	postToolCount := 0
	parseErrors := []string{}

	// Iterate all events
	for i, eventMap := range events {
		// Marshal back to JSON for parsing
		eventJSON, err := json.Marshal(eventMap)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: marshal error: %v", i, err))
			continue
		}

		// Determine event type from hook_event_name
		hookEventName, ok := eventMap["hook_event_name"].(string)
		if !ok {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: missing hook_event_name", i))
			continue
		}

		reader := strings.NewReader(string(eventJSON))

		if hookEventName == "PreToolUse" {
			preToolCount++
			event, err := ParseToolEvent(reader, 1*time.Second)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("Event %d (PreToolUse): %v", i, err))
				continue
			}
			if event.ToolName == "" {
				parseErrors = append(parseErrors, fmt.Sprintf("Event %d: parsed but empty tool_name", i))
			}
		} else if hookEventName == "PostToolUse" {
			postToolCount++
			event, err := ParsePostToolEvent(reader, 1*time.Second)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("Event %d (PostToolUse): %v", i, err))
				continue
			}
			if event.ToolName == "" {
				parseErrors = append(parseErrors, fmt.Sprintf("Event %d: parsed but empty tool_name", i))
			}
		} else {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: unknown hook_event_name: %s", i, hookEventName))
		}
	}

	// Report results
	t.Logf("PreToolUse events: %d", preToolCount)
	t.Logf("PostToolUse events: %d", postToolCount)

	if len(parseErrors) > 0 {
		t.Errorf("Failed to parse %d/%d corpus events:", len(parseErrors), len(events))
		for _, errMsg := range parseErrors {
			t.Logf("  - %s", errMsg)
		}
		t.Fatalf("Corpus replay test failed")
	}

	t.Logf("✓ Successfully parsed %d/%d corpus events", len(events), len(events))
}

// Test error message format compliance
func TestErrorMessageFormat(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedParts []string
		parseFunc     func(io.Reader, time.Duration) error
	}{
		{
			name:          "ParseToolEvent malformed JSON",
			json:          `{"tool_name": malformed`,
			expectedParts: []string{"[event-parser]", "Failed to parse JSON", "Ensure hook receives valid JSON"},
			parseFunc: func(r io.Reader, timeout time.Duration) error {
				_, err := ParseToolEvent(r, timeout)
				return err
			},
		},
		{
			name:          "ParseToolEvent missing tool_name",
			json:          `{"hook_event_name":"PreToolUse"}`,
			expectedParts: []string{"[event-parser]", "Missing tool_name", "Ensure hook emits complete"},
			parseFunc: func(r io.Reader, timeout time.Duration) error {
				_, err := ParseToolEvent(r, timeout)
				return err
			},
		},
		{
			name:          "ParsePostToolEvent missing tool_response",
			json:          `{"tool_name":"Bash","hook_event_name":"PostToolUse"}`,
			expectedParts: []string{"[event-parser]", "Missing tool_response", "PostToolUse event incomplete"},
			parseFunc: func(r io.Reader, timeout time.Duration) error {
				_, err := ParsePostToolEvent(r, timeout)
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			err := tc.parseFunc(reader, 1*time.Second)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, part := range tc.expectedParts {
				if !strings.Contains(errMsg, part) {
					t.Errorf("error message should contain %q, got: %v", part, errMsg)
				}
			}
		})
	}
}

// Test ParseTaskInput from real corpus Task events
func TestParseTaskInput_FromCorpus(t *testing.T) {
	corpusPath := "../../test/fixtures/event-corpus.json"
	data, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Fatalf("failed to read corpus file: %v", err)
	}

	var events []map[string]interface{}
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("failed to parse corpus JSON: %v", err)
	}

	taskCount := 0
	parseErrors := []string{}

	for i, eventMap := range events {
		toolName, ok := eventMap["tool_name"].(string)
		if !ok || toolName != "Task" {
			continue
		}

		taskCount++

		toolInput, ok := eventMap["tool_input"].(map[string]interface{})
		if !ok {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: tool_input not a map", i))
			continue
		}

		taskInput, err := ParseTaskInput(toolInput)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: %v", i, err))
			continue
		}

		if taskInput.Prompt == "" {
			parseErrors = append(parseErrors, fmt.Sprintf("Event %d: parsed but empty prompt", i))
		}
	}

	t.Logf("Task events in corpus: %d", taskCount)

	if len(parseErrors) > 0 {
		t.Errorf("Failed to parse %d Task inputs:", len(parseErrors))
		for _, errMsg := range parseErrors {
			t.Logf("  - %s", errMsg)
		}
		t.Fatalf("Task input parsing test failed")
	}

	t.Logf("✓ Successfully parsed %d Task inputs from corpus", taskCount)
}

// ============================================================================
// SubagentStop Event Tests (GOgent-063)
// ============================================================================

// Test ParseSubagentStopEvent with valid ACTUAL schema
func TestParseSubagentStopEvent_Success(t *testing.T) {
	// ACTUAL schema from GOgent-063a validation
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-12345",
		"transcript_path": "/tmp/test-transcript.jsonl",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "test-session-12345" {
		t.Errorf("Expected session ID test-session-12345, got: %s", event.SessionID)
	}

	if event.TranscriptPath != "/tmp/test-transcript.jsonl" {
		t.Errorf("Expected transcript path /tmp/test-transcript.jsonl, got: %s", event.TranscriptPath)
	}

	if event.HookEventName != "SubagentStop" {
		t.Errorf("Expected hook_event_name SubagentStop, got: %s", event.HookEventName)
	}

	if !event.StopHookActive {
		t.Error("Expected StopHookActive to be true")
	}
}

// Test ParseSubagentStopEvent with missing session_id
func TestParseSubagentStopEvent_MissingSessionID(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"transcript_path": "/tmp/test.jsonl"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for missing session_id")
	}

	if !strings.Contains(err.Error(), "session_id") {
		t.Errorf("Error should mention session_id, got: %v", err)
	}
}

// Test ParseSubagentStopEvent with missing transcript_path
func TestParseSubagentStopEvent_MissingTranscriptPath(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session-123"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for missing transcript_path")
	}

	if !strings.Contains(err.Error(), "transcript_path") {
		t.Errorf("Error should mention transcript_path, got: %v", err)
	}
}

// Test ParseSubagentStopEvent with invalid hook_event_name
func TestParseSubagentStopEvent_InvalidHookEventName(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "PreToolUse",
		"session_id": "test-session-123",
		"transcript_path": "/tmp/test.jsonl"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for invalid hook_event_name")
	}

	if !strings.Contains(err.Error(), "expected SubagentStop") {
		t.Errorf("Error should mention expected SubagentStop, got: %v", err)
	}
}

// Test ParseSubagentStopEvent with malformed JSON
func TestParseSubagentStopEvent_MalformedJSON(t *testing.T) {
	jsonInput := `{"hook_event_name": malformed`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "Failed to parse JSON") {
		t.Errorf("Error should mention JSON parsing failure, got: %v", err)
	}
}

// Test ParseSubagentStopEvent with timeout
func TestParseSubagentStopEvent_Timeout(t *testing.T) {
	reader := &slowReader{
		delay: 2 * time.Second,
		data:  `{"hook_event_name":"SubagentStop"}`,
	}

	_, err := ParseSubagentStopEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should mention timeout, got: %v", err)
	}
}

// Test ParseTranscriptForMetadata with valid transcript
func TestParseTranscriptForMetadata_Success(t *testing.T) {
	// Create mock transcript file
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: orchestrator", "role": "system"}
{"timestamp": 1700000100, "model": "claude-sonnet-4", "role": "assistant"}
{"timestamp": 1700005000, "content": "Task complete", "role": "assistant"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to write mock transcript: %v", err)
	}

	metadata, err := ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "orchestrator" {
		t.Errorf("Expected agent_id orchestrator, got: %s", metadata.AgentID)
	}

	if metadata.AgentModel != "claude-sonnet-4" {
		t.Errorf("Expected model claude-sonnet-4, got: %s", metadata.AgentModel)
	}

	if metadata.Tier != "sonnet" {
		t.Errorf("Expected tier sonnet, got: %s", metadata.Tier)
	}

	if metadata.DurationMs != 5000 {
		t.Errorf("Expected duration 5000ms, got: %d", metadata.DurationMs)
	}

	if !metadata.IsSuccess() {
		t.Error("Expected success (exit_code=0)")
	}
}

// Test ParseTranscriptForMetadata with non-existent file
func TestParseTranscriptForMetadata_NonExistentFile(t *testing.T) {
	metadata, err := ParseTranscriptForMetadata("/nonexistent/path/transcript.jsonl")

	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Should return partial metadata on error
	if metadata == nil {
		t.Fatal("Expected partial metadata, got nil")
	}

	if metadata.ExitCode != 0 {
		t.Errorf("Expected default ExitCode 0, got: %d", metadata.ExitCode)
	}
}

// Test ParseTranscriptForMetadata with malformed JSONL (graceful degradation)
func TestParseTranscriptForMetadata_MalformedJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: python-pro"}
{this is not valid json}
{"timestamp": 1700005000, "model": "claude-haiku-4"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to write mock transcript: %v", err)
	}

	metadata, err := ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("Expected graceful degradation, got error: %v", err)
	}

	// Should extract what it can
	if metadata.AgentID != "python-pro" {
		t.Errorf("Expected agent_id python-pro despite malformed lines, got: %s", metadata.AgentID)
	}

	if metadata.Tier != "haiku" {
		t.Errorf("Expected tier haiku, got: %s", metadata.Tier)
	}
}

// Test ParseTranscriptForMetadata with error markers
func TestParseTranscriptForMetadata_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "error-transcript.jsonl")

	transcriptData := `{"timestamp": 1700000000, "content": "AGENT: go-pro"}
{"timestamp": 1700001000, "role": "error", "content": "Something failed"}
{"timestamp": 1700002000, "content": "Attempt recovery"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to write mock transcript: %v", err)
	}

	metadata, err := ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ExitCode != 1 {
		t.Errorf("Expected exit_code 1 when error role present, got: %d", metadata.ExitCode)
	}

	if metadata.IsSuccess() {
		t.Error("Expected IsSuccess() to return false when exit_code=1")
	}
}

// Test ParseTranscriptForMetadata with empty file
func TestParseTranscriptForMetadata_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty.jsonl")

	if err := os.WriteFile(transcriptPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty transcript: %v", err)
	}

	metadata, err := ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	// Should return default metadata
	if metadata.ExitCode != 0 {
		t.Errorf("Expected default ExitCode 0, got: %d", metadata.ExitCode)
	}

	if metadata.AgentID != "" {
		t.Errorf("Expected empty AgentID, got: %s", metadata.AgentID)
	}
}

// Test GetAgentClass with all agent types
func TestGetAgentClass_All(t *testing.T) {
	tests := []struct {
		agentID       string
		expectedClass AgentClass
	}{
		// Orchestrator class
		{"orchestrator", ClassOrchestrator},
		{"architect", ClassOrchestrator},
		{"einstein", ClassOrchestrator},
		{"planner", ClassOrchestrator},
		{"mozart", ClassOrchestrator},
		{"review-orchestrator", ClassOrchestrator},
		{"impl-manager", ClassOrchestrator},
		{"python-architect", ClassOrchestrator},
		// Implementation class
		{"python-pro", ClassImplementation},
		{"python-ux", ClassImplementation},
		{"go-pro", ClassImplementation},
		{"go-cli", ClassImplementation},
		{"go-tui", ClassImplementation},
		{"go-api", ClassImplementation},
		{"go-concurrent", ClassImplementation},
		{"r-pro", ClassImplementation},
		{"r-shiny-pro", ClassImplementation},
		{"typescript-pro", ClassImplementation},
		{"react-pro", ClassImplementation},
		// Specialist class
		{"code-reviewer", ClassSpecialist},
		{"librarian", ClassSpecialist},
		{"tech-docs-writer", ClassSpecialist},
		{"scaffolder", ClassSpecialist},
		{"backend-reviewer", ClassSpecialist},
		{"frontend-reviewer", ClassSpecialist},
		{"standards-reviewer", ClassSpecialist},
		{"memory-archivist", ClassSpecialist},
		// Coordination class
		{"codebase-search", ClassCoordination},
		{"haiku-scout", ClassCoordination},
		// Analysis class
		{"beethoven", ClassAnalysis},
		{"staff-architect-critical-review", ClassAnalysis},
		{"gemini-slave", ClassAnalysis},
		// Unknown
		{"unknown-agent", ClassUnknown},
		{"", ClassUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			got := GetAgentClass(tc.agentID)
			if got != tc.expectedClass {
				t.Errorf("AgentID %q: expected %s, got %s", tc.agentID, tc.expectedClass, got)
			}
		})
	}
}

// Test deriveTierFromModel with various model names
func TestDeriveTierFromModel(t *testing.T) {
	tests := []struct {
		model        string
		expectedTier string
	}{
		{"claude-haiku-4", "haiku"},
		{"claude-haiku-3.5", "haiku"},
		{"CLAUDE-HAIKU-4", "haiku"}, // Case insensitive
		{"claude-sonnet-4", "sonnet"},
		{"claude-sonnet-3.5", "sonnet"},
		{"claude-opus-4", "opus"},
		{"claude-opus-3", "opus"},
		{"unknown-model", "unknown"},
		{"gpt-4", "unknown"},
		{"", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			got := deriveTierFromModel(tc.model)
			if got != tc.expectedTier {
				t.Errorf("Model %q: expected tier %q, got %q", tc.model, tc.expectedTier, got)
			}
		})
	}
}

// Test ParsedAgentMetadata.IsSuccess()
func TestParsedAgentMetadata_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		expected bool
	}{
		{"success", 0, true},
		{"failure", 1, false},
		{"error code 2", 2, false},
		{"negative exit code", -1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metadata := &ParsedAgentMetadata{ExitCode: tc.exitCode}
			got := metadata.IsSuccess()
			if got != tc.expected {
				t.Errorf("ExitCode %d: expected IsSuccess()=%v, got %v", tc.exitCode, tc.expected, got)
			}
		})
	}
}

// Test ParseTranscriptForMetadata duration calculation edge cases
func TestParseTranscriptForMetadata_DurationEdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		transcriptData   string
		expectedDuration int
	}{
		{
			name: "no timestamps",
			transcriptData: `{"content": "AGENT: test"}
{"model": "claude-haiku-4"}`,
			expectedDuration: 0,
		},
		{
			name:             "single timestamp",
			transcriptData:   `{"timestamp": 1700000000, "content": "AGENT: test"}`,
			expectedDuration: 0,
		},
		{
			name: "same timestamps",
			transcriptData: `{"timestamp": 1700000000, "content": "start"}
{"timestamp": 1700000000, "content": "end"}`,
			expectedDuration: 0,
		},
		{
			name: "multiple timestamps",
			transcriptData: `{"timestamp": 1700000000}
{"timestamp": 1700001000}
{"timestamp": 1700002500}`,
			expectedDuration: 2500,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

			if err := os.WriteFile(transcriptPath, []byte(tc.transcriptData), 0644); err != nil {
				t.Fatalf("Failed to write transcript: %v", err)
			}

			metadata, err := ParseTranscriptForMetadata(transcriptPath)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if metadata.DurationMs != tc.expectedDuration {
				t.Errorf("Expected duration %dms, got %dms", tc.expectedDuration, metadata.DurationMs)
			}
		})
	}
}

// Test SubagentStopEvent with stop_hook_active variations
func TestParseSubagentStopEvent_StopHookActiveVariations(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{
			name: "stop_hook_active true",
			json: `{
				"hook_event_name": "SubagentStop",
				"session_id": "test",
				"transcript_path": "/tmp/test.jsonl",
				"stop_hook_active": true
			}`,
			expected: true,
		},
		{
			name: "stop_hook_active false",
			json: `{
				"hook_event_name": "SubagentStop",
				"session_id": "test",
				"transcript_path": "/tmp/test.jsonl",
				"stop_hook_active": false
			}`,
			expected: false,
		},
		{
			name: "stop_hook_active missing (defaults to false)",
			json: `{
				"hook_event_name": "SubagentStop",
				"session_id": "test",
				"transcript_path": "/tmp/test.jsonl"
			}`,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.json)
			event, err := ParseSubagentStopEvent(reader, 5*time.Second)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if event.StopHookActive != tc.expected {
				t.Errorf("Expected StopHookActive=%v, got %v", tc.expected, event.StopHookActive)
			}
		})
	}
}

// ============================================================================
// v2.1.69 Compatibility Tests
// ============================================================================

// Test SubagentStopEvent with v2.1.69 fields
func TestParseSubagentStopEvent_WithV2169Fields(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-v2169",
		"transcript_path": "/path/to/session.jsonl",
		"stop_hook_active": false,
		"agent_id": "go-pro",
		"agent_type": "GO Pro",
		"agent_transcript_path": "/path/to/subagents/agent-abc123.jsonl",
		"last_assistant_message": "Implementation complete.",
		"cwd": "/home/user/project",
		"permission_mode": "default"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.AgentID != "go-pro" {
		t.Errorf("Expected agent_id 'go-pro', got: %s", event.AgentID)
	}
	if event.AgentType != "GO Pro" {
		t.Errorf("Expected agent_type 'GO Pro', got: %s", event.AgentType)
	}
	if event.AgentTranscriptPath != "/path/to/subagents/agent-abc123.jsonl" {
		t.Errorf("Expected agent_transcript_path, got: %s", event.AgentTranscriptPath)
	}
	if event.LastAssistantMessage != "Implementation complete." {
		t.Errorf("Expected last_assistant_message, got: %s", event.LastAssistantMessage)
	}
	if event.CWD != "/home/user/project" {
		t.Errorf("Expected cwd '/home/user/project', got: %s", event.CWD)
	}
	if event.PermissionMode != "default" {
		t.Errorf("Expected permission_mode 'default', got: %s", event.PermissionMode)
	}
}

// Test SubagentStopEvent with only agent_transcript_path (no transcript_path)
func TestParseSubagentStopEvent_AgentTranscriptPathOnly(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-atp-only",
		"agent_transcript_path": "/path/to/subagents/agent-def456.jsonl",
		"agent_id": "python-pro"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error when agent_transcript_path is present, got: %v", err)
	}

	if event.TranscriptPath != "" {
		t.Errorf("Expected empty transcript_path, got: %s", event.TranscriptPath)
	}
	if event.AgentTranscriptPath != "/path/to/subagents/agent-def456.jsonl" {
		t.Errorf("Expected agent_transcript_path, got: %s", event.AgentTranscriptPath)
	}
}

// Test SubagentStopEvent missing both transcript paths rejects
func TestParseSubagentStopEvent_MissingBothTranscriptPaths(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-no-paths",
		"agent_id": "python-pro"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSubagentStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Fatal("Expected error when both transcript paths are missing")
	}

	if !strings.Contains(err.Error(), "transcript_path or agent_transcript_path") {
		t.Errorf("Error should mention both paths, got: %v", err)
	}
}

// ============================================================================
// EnrichMetadataFromEvent Tests (v2.1.69 optimization)
// ============================================================================

func TestEnrichMetadataFromEvent_DirectFieldsOnly(t *testing.T) {
	// Event has agent_id but no transcript → returns partial metadata
	event := &SubagentStopEvent{
		HookEventName: "SubagentStop",
		SessionID:     "test-direct",
		AgentID:       "go-pro",
		AgentType:     "GO Pro",
		// No transcript_path → transcript parsing will fail
	}

	metadata, err := EnrichMetadataFromEvent(event)
	if err != nil {
		t.Fatalf("Expected no error with direct fields, got: %v", err)
	}

	if metadata.AgentID != "go-pro" {
		t.Errorf("Expected agent_id 'go-pro', got: %s", metadata.AgentID)
	}
}

func TestEnrichMetadataFromEvent_FallbackToTranscript(t *testing.T) {
	// Event has no direct fields → falls back to transcript
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcriptContent := `{"content":"AGENT: python-pro","role":"user","timestamp":1705708800}
{"model":"claude-sonnet-4-5","timestamp":1705708810}
{"content":"Done","role":"assistant","timestamp":1705708820}
`
	os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

	event := &SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-fallback",
		TranscriptPath: transcriptPath,
	}

	metadata, err := EnrichMetadataFromEvent(event)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "python-pro" {
		t.Errorf("Expected agent_id 'python-pro' from transcript, got: %s", metadata.AgentID)
	}
	if metadata.Tier != "sonnet" {
		t.Errorf("Expected tier 'sonnet' from transcript, got: %s", metadata.Tier)
	}
}

func TestEnrichMetadataFromEvent_MixedSources(t *testing.T) {
	// Event has agent_id, transcript has duration/model → merged
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcriptContent := `{"content":"AGENT: python-pro","role":"user","timestamp":1705708800}
{"model":"claude-sonnet-4-5","timestamp":1705708810}
{"content":"Done","role":"assistant","timestamp":1705708820}
`
	os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

	event := &SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-mixed",
		TranscriptPath: transcriptPath,
		AgentID:        "go-pro", // Direct field takes precedence
	}

	metadata, err := EnrichMetadataFromEvent(event)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Event field wins for AgentID
	if metadata.AgentID != "go-pro" {
		t.Errorf("Expected agent_id 'go-pro' from event (precedence), got: %s", metadata.AgentID)
	}
	// Transcript provides model/tier
	if metadata.Tier != "sonnet" {
		t.Errorf("Expected tier 'sonnet' from transcript, got: %s", metadata.Tier)
	}
	// Transcript provides duration
	if metadata.DurationMs == 0 {
		t.Error("Expected non-zero duration from transcript")
	}
}

func TestEnrichMetadataFromEvent_PrefersAgentTranscriptPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Session transcript (wrong agent)
	sessionPath := filepath.Join(tmpDir, "session.jsonl")
	os.WriteFile(sessionPath, []byte(`{"content":"AGENT: wrong-agent","timestamp":1705708800}
`), 0644)

	// Agent transcript (correct agent)
	agentPath := filepath.Join(tmpDir, "agent.jsonl")
	os.WriteFile(agentPath, []byte(`{"content":"AGENT: correct-agent","timestamp":1705708800}
{"timestamp":1705708810}
`), 0644)

	event := &SubagentStopEvent{
		HookEventName:      "SubagentStop",
		SessionID:          "test-atp",
		TranscriptPath:     sessionPath,
		AgentTranscriptPath: agentPath, // Should use this one
	}

	metadata, err := EnrichMetadataFromEvent(event)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "correct-agent" {
		t.Errorf("Expected agent_id from agent_transcript_path, got: %s", metadata.AgentID)
	}
}

func TestEnrichMetadataFromEvent_EventAndTranscriptDisagree(t *testing.T) {
	// Event has agent_id "go-pro", transcript has "AGENT: python-pro" → event wins
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	os.WriteFile(transcriptPath, []byte(`{"content":"AGENT: python-pro","timestamp":1705708800}
{"timestamp":1705708810}
`), 0644)

	event := &SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-disagree",
		TranscriptPath: transcriptPath,
		AgentID:        "go-pro", // Event field wins
	}

	metadata, err := EnrichMetadataFromEvent(event)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "go-pro" {
		t.Errorf("Expected event agent_id 'go-pro' to take precedence, got: %s", metadata.AgentID)
	}
}

func TestNormalizeAgentType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GO Pro", "go-pro"},
		{"Python Pro", "python-pro"},
		{"Explore", "explore"},
		{"Codebase Search", "codebase-search"},
		{"GO TUI (Bubbletea)", "go-tui"},       // parenthetical stripped
		{"GO CLI (Cobra)", "go-cli"},             // parenthetical stripped
		{"GO API (HTTP Client)", "go-api"},       // parenthetical stripped
		{"Python UX (PySide6)", "python-ux"},     // parenthetical stripped
		{"  Haiku Scout  ", "haiku-scout"},
		{"R Pro", "r-pro"},
		{"TypeScript Pro", "typescript-pro"},
		{"Staff Architect Critical Review", "staff-architect-critical-review"},
		{"haiku-scout", "haiku-scout"},           // already normalized
		{"", ""},                                  // empty string
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeAgentType(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeAgentType(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ============================================================================
// ToolEvent Helper Methods Tests (GOgent-080)
// ============================================================================

func TestToolEvent_ExtractFilePath(t *testing.T) {
	tests := []struct {
		name      string
		toolInput map[string]interface{}
		expected  string
	}{
		{
			name:      "valid file_path",
			toolInput: map[string]interface{}{"file_path": "/home/user/CLAUDE.md"},
			expected:  "/home/user/CLAUDE.md",
		},
		{
			name:      "missing file_path",
			toolInput: map[string]interface{}{"other": "value"},
			expected:  "",
		},
		{
			name:      "file_path wrong type",
			toolInput: map[string]interface{}{"file_path": 123},
			expected:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{ToolInput: tc.toolInput}
			if got := event.ExtractFilePath(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_ExtractWriteContent(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolInput map[string]interface{}
		expected  string
	}{
		{
			name:      "Write with content",
			toolName:  "Write",
			toolInput: map[string]interface{}{"content": "file contents"},
			expected:  "file contents",
		},
		{
			name:      "Edit with new_string",
			toolName:  "Edit",
			toolInput: map[string]interface{}{"new_string": "replacement text"},
			expected:  "replacement text",
		},
		{
			name:      "no content fields",
			toolName:  "Write",
			toolInput: map[string]interface{}{"other": "value"},
			expected:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{
				ToolName:  tc.toolName,
				ToolInput: tc.toolInput,
			}
			if got := event.ExtractWriteContent(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_IsClaudeMDFile(t *testing.T) {
	tests := []struct {
		name      string
		toolInput map[string]interface{}
		expected  bool
	}{
		{
			name:      "CLAUDE.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.md"},
			expected:  true,
		},
		{
			name:      "CLAUDE.en.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.en.md"},
			expected:  true,
		},
		{
			name:      "other.md",
			toolInput: map[string]interface{}{"file_path": "/path/to/other.md"},
			expected:  false,
		},
		{
			name:      "CLAUDE.txt",
			toolInput: map[string]interface{}{"file_path": "/path/to/CLAUDE.txt"},
			expected:  false,
		},
		{
			name:      "no file_path",
			toolInput: map[string]interface{}{},
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &ToolEvent{ToolInput: tc.toolInput}
			if got := event.IsClaudeMDFile(); got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestToolEvent_IsWriteOperation(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"Write", true},
		{"Edit", true},
		{"Read", false},
		{"Bash", false},
		{"Task", false},
	}

	for _, tc := range tests {
		t.Run(tc.toolName, func(t *testing.T) {
			event := &ToolEvent{ToolName: tc.toolName}
			if got := event.IsWriteOperation(); got != tc.expected {
				t.Errorf("tool %s: expected %v, got %v", tc.toolName, tc.expected, got)
			}
		})
	}
}

// ============================================================================
// PostToolEvent ML Telemetry Tests (GOgent-086b)
// ============================================================================

// TestPostToolEvent_MLFields tests marshaling and unmarshaling of ML telemetry fields
func TestPostToolEvent_MLFields(t *testing.T) {
	event := PostToolEvent{
		// Core fields
		ToolName:      "Read",
		SessionID:     "sess-123",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465000,
		ToolInput:     map[string]interface{}{"file_path": "/test.go"},
		ToolResponse:  map[string]interface{}{"success": true},

		// ML telemetry fields
		DurationMs:       150,
		InputTokens:      1024,
		OutputTokens:     512,
		Model:            "claude-haiku-4",
		Tier:             "haiku",
		Success:          true,
		SequenceIndex:    5,
		PreviousTools:    []string{"Glob", "Grep", "Read"},
		PreviousOutcomes: []bool{true, true, true},
		TaskType:         "search",
		TaskDomain:       "python",
		SelectedTier:     "haiku",
		SelectedAgent:    "codebase-search",
		EventID:          "evt-abc123",
		TargetSize:       5000,
		CoverageAchieved: 0.85,
		EntitiesFound:    12,
	}

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var parsed PostToolEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify core fields
	if parsed.ToolName != "Read" {
		t.Errorf("Expected ToolName Read, got %s", parsed.ToolName)
	}

	// Verify ML fields
	if parsed.DurationMs != 150 {
		t.Errorf("Expected DurationMs 150, got %d", parsed.DurationMs)
	}

	if parsed.InputTokens != 1024 {
		t.Errorf("Expected InputTokens 1024, got %d", parsed.InputTokens)
	}

	if parsed.OutputTokens != 512 {
		t.Errorf("Expected OutputTokens 512, got %d", parsed.OutputTokens)
	}

	if parsed.Model != "claude-haiku-4" {
		t.Errorf("Expected Model claude-haiku-4, got %s", parsed.Model)
	}

	if parsed.Tier != "haiku" {
		t.Errorf("Expected Tier haiku, got %s", parsed.Tier)
	}

	if !parsed.Success {
		t.Error("Expected Success true, got false")
	}

	if parsed.SequenceIndex != 5 {
		t.Errorf("Expected SequenceIndex 5, got %d", parsed.SequenceIndex)
	}

	if len(parsed.PreviousTools) != 3 {
		t.Errorf("Expected 3 PreviousTools, got %d", len(parsed.PreviousTools))
	}

	if parsed.PreviousTools[0] != "Glob" {
		t.Errorf("Expected PreviousTools[0] Glob, got %s", parsed.PreviousTools[0])
	}

	if len(parsed.PreviousOutcomes) != 3 {
		t.Errorf("Expected 3 PreviousOutcomes, got %d", len(parsed.PreviousOutcomes))
	}

	if !parsed.PreviousOutcomes[0] {
		t.Error("Expected PreviousOutcomes[0] true")
	}

	if parsed.TaskType != "search" {
		t.Errorf("Expected TaskType search, got %s", parsed.TaskType)
	}

	if parsed.TaskDomain != "python" {
		t.Errorf("Expected TaskDomain python, got %s", parsed.TaskDomain)
	}

	if parsed.SelectedTier != "haiku" {
		t.Errorf("Expected SelectedTier haiku, got %s", parsed.SelectedTier)
	}

	if parsed.SelectedAgent != "codebase-search" {
		t.Errorf("Expected SelectedAgent codebase-search, got %s", parsed.SelectedAgent)
	}

	if parsed.EventID != "evt-abc123" {
		t.Errorf("Expected EventID evt-abc123, got %s", parsed.EventID)
	}

	if parsed.TargetSize != 5000 {
		t.Errorf("Expected TargetSize 5000, got %d", parsed.TargetSize)
	}

	if parsed.CoverageAchieved != 0.85 {
		t.Errorf("Expected CoverageAchieved 0.85, got %f", parsed.CoverageAchieved)
	}

	if parsed.EntitiesFound != 12 {
		t.Errorf("Expected EntitiesFound 12, got %d", parsed.EntitiesFound)
	}
}

// TestPostToolEvent_BackwardCompatibility verifies old JSON without ML fields parses correctly
func TestPostToolEvent_BackwardCompatibility(t *testing.T) {
	// Old JSON format without ML fields
	oldJSON := `{
		"tool_name": "Read",
		"session_id": "sess-123",
		"hook_event_name": "PostToolUse",
		"captured_at": 1234567890,
		"tool_input": {"file_path": "/test.go"},
		"tool_response": {"success": true}
	}`

	var event PostToolEvent
	if err := json.Unmarshal([]byte(oldJSON), &event); err != nil {
		t.Fatalf("Failed to parse old format: %v", err)
	}

	// Core fields should be populated
	if event.ToolName != "Read" {
		t.Errorf("Expected ToolName Read, got %s", event.ToolName)
	}

	if event.SessionID != "sess-123" {
		t.Errorf("Expected SessionID sess-123, got %s", event.SessionID)
	}

	// ML fields should be zero values
	if event.DurationMs != 0 {
		t.Errorf("Expected DurationMs 0, got %d", event.DurationMs)
	}

	if event.SequenceIndex != 0 {
		t.Errorf("Expected SequenceIndex 0, got %d", event.SequenceIndex)
	}

	if event.Model != "" {
		t.Errorf("Expected Model empty, got %s", event.Model)
	}

	if event.Success {
		t.Error("Expected Success false (zero value), got true")
	}

	if event.PreviousTools != nil {
		t.Errorf("Expected PreviousTools nil, got %v", event.PreviousTools)
	}

	if event.PreviousOutcomes != nil {
		t.Errorf("Expected PreviousOutcomes nil, got %v", event.PreviousOutcomes)
	}
}

// TestPostToolEvent_OmitEmpty verifies empty ML fields are omitted from JSON
func TestPostToolEvent_OmitEmpty(t *testing.T) {
	event := PostToolEvent{
		ToolName:      "Read",
		SessionID:     "sess-123",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465000,
		ToolInput:     map[string]interface{}{"file_path": "/test.go"},
		ToolResponse:  map[string]interface{}{"success": true},
		// No ML fields set
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// ML fields should not appear in JSON when empty
	// Note: Go's json encoder does NOT omit false booleans even with omitempty,
	// so we exclude "success" from this check
	mlFields := []string{
		"duration_ms",
		"input_tokens",
		"output_tokens",
		"model",
		"tier",
		"sequence_index",
		"previous_tools",
		"previous_outcomes",
		"task_type",
		"task_domain",
		"selected_tier",
		"selected_agent",
		"event_id",
		"target_size",
		"coverage_achieved",
		"entities_found",
	}

	for _, field := range mlFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("Empty field %q should be omitted from JSON, but found in: %s", field, jsonStr)
		}
	}

	// Success field: Go's JSON encoder includes false booleans even with omitempty
	// This is expected Go behavior, so we just verify the value is false
	if event.Success {
		t.Error("Expected Success to be false (zero value)")
	}

	// Core fields should still be present
	if !strings.Contains(jsonStr, "tool_name") {
		t.Error("Core field tool_name should be present")
	}

	if !strings.Contains(jsonStr, "session_id") {
		t.Error("Core field session_id should be present")
	}
}

// TestPostToolEvent_PartialMLFields tests events with only some ML fields populated
func TestPostToolEvent_PartialMLFields(t *testing.T) {
	event := PostToolEvent{
		ToolName:      "Task",
		SessionID:     "sess-456",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465100,
		ToolInput:     map[string]interface{}{"model": "sonnet"},
		ToolResponse:  map[string]interface{}{"output": "result"},

		// Only partial ML fields
		Model:   "claude-sonnet-4",
		Tier:    "sonnet",
		Success: true,
		EventID: "evt-def456",
		// Other fields left empty
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed PostToolEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Populated fields should be present
	if parsed.Model != "claude-sonnet-4" {
		t.Errorf("Expected Model claude-sonnet-4, got %s", parsed.Model)
	}

	if parsed.Tier != "sonnet" {
		t.Errorf("Expected Tier sonnet, got %s", parsed.Tier)
	}

	if !parsed.Success {
		t.Error("Expected Success true")
	}

	if parsed.EventID != "evt-def456" {
		t.Errorf("Expected EventID evt-def456, got %s", parsed.EventID)
	}

	// Empty fields should be zero values
	if parsed.DurationMs != 0 {
		t.Errorf("Expected DurationMs 0, got %d", parsed.DurationMs)
	}

	if parsed.SequenceIndex != 0 {
		t.Errorf("Expected SequenceIndex 0, got %d", parsed.SequenceIndex)
	}

	if parsed.TaskType != "" {
		t.Errorf("Expected TaskType empty, got %s", parsed.TaskType)
	}
}

// TestPostToolEvent_SequenceTracking specifically tests sequence tracking fields
func TestPostToolEvent_SequenceTracking(t *testing.T) {
	event := PostToolEvent{
		ToolName:      "Edit",
		SessionID:     "sess-789",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465200,
		ToolInput:     map[string]interface{}{"file_path": "/code.go"},
		ToolResponse:  map[string]interface{}{"success": true},

		SequenceIndex: 10,
		PreviousTools: []string{
			"Glob", "Grep", "Read", "Read", "Read",
			"Edit", "Bash", "Read", "Edit", "Bash",
		},
		PreviousOutcomes: []bool{
			true, true, true, true, true,
			true, false, true, true, true,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed PostToolEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.SequenceIndex != 10 {
		t.Errorf("Expected SequenceIndex 10, got %d", parsed.SequenceIndex)
	}

	if len(parsed.PreviousTools) != 10 {
		t.Fatalf("Expected 10 PreviousTools, got %d", len(parsed.PreviousTools))
	}

	if parsed.PreviousTools[0] != "Glob" {
		t.Errorf("Expected first tool Glob, got %s", parsed.PreviousTools[0])
	}

	if parsed.PreviousTools[9] != "Bash" {
		t.Errorf("Expected last tool Bash, got %s", parsed.PreviousTools[9])
	}

	if len(parsed.PreviousOutcomes) != 10 {
		t.Fatalf("Expected 10 PreviousOutcomes, got %d", len(parsed.PreviousOutcomes))
	}

	// Check specific outcome (index 6 should be false)
	if parsed.PreviousOutcomes[6] {
		t.Error("Expected PreviousOutcomes[6] false (Bash failure)")
	}
}

// TestPostToolEvent_UnderstandingContext tests understanding context fields
func TestPostToolEvent_UnderstandingContext(t *testing.T) {
	event := PostToolEvent{
		ToolName:      "Grep",
		SessionID:     "sess-abc",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465300,
		ToolInput:     map[string]interface{}{"pattern": "func"},
		ToolResponse:  map[string]interface{}{"matches": 42},

		TargetSize:       150000,
		CoverageAchieved: 0.92,
		EntitiesFound:    42,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed PostToolEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.TargetSize != 150000 {
		t.Errorf("Expected TargetSize 150000, got %d", parsed.TargetSize)
	}

	if parsed.CoverageAchieved != 0.92 {
		t.Errorf("Expected CoverageAchieved 0.92, got %f", parsed.CoverageAchieved)
	}

	if parsed.EntitiesFound != 42 {
		t.Errorf("Expected EntitiesFound 42, got %d", parsed.EntitiesFound)
	}
}

// TestPostToolEvent_RoutingInfo tests routing information fields
func TestPostToolEvent_RoutingInfo(t *testing.T) {
	event := PostToolEvent{
		ToolName:      "Task",
		SessionID:     "sess-routing",
		HookEventName: "PostToolUse",
		CapturedAt:    1768465400,
		ToolInput: map[string]interface{}{
			"model":         "sonnet",
			"subagent_type": "general-purpose",
		},
		ToolResponse: map[string]interface{}{"output": "complete"},

		Model:         "claude-sonnet-4",
		Tier:          "sonnet",
		SelectedTier:  "sonnet",
		SelectedAgent: "python-pro",
		TaskType:      "implementation",
		TaskDomain:    "python",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed PostToolEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.SelectedTier != "sonnet" {
		t.Errorf("Expected SelectedTier sonnet, got %s", parsed.SelectedTier)
	}

	if parsed.SelectedAgent != "python-pro" {
		t.Errorf("Expected SelectedAgent python-pro, got %s", parsed.SelectedAgent)
	}

	if parsed.TaskType != "implementation" {
		t.Errorf("Expected TaskType implementation, got %s", parsed.TaskType)
	}

	if parsed.TaskDomain != "python" {
		t.Errorf("Expected TaskDomain python, got %s", parsed.TaskDomain)
	}
}

// TestPostToolEvent_PerformanceMetrics tests performance-related fields
func TestPostToolEvent_PerformanceMetrics(t *testing.T) {
	tests := []struct {
		name         string
		durationMs   int64
		inputTokens  int
		outputTokens int
	}{
		{"fast operation", 50, 100, 50},
		{"slow operation", 5000, 50000, 10000},
		{"zero tokens", 100, 0, 0},
		{"large context", 2000, 150000, 25000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := PostToolEvent{
				ToolName:      "Read",
				SessionID:     "sess-perf",
				HookEventName: "PostToolUse",
				CapturedAt:    1768465500,
				ToolInput:     map[string]interface{}{},
				ToolResponse:  map[string]interface{}{},

				DurationMs:   tc.durationMs,
				InputTokens:  tc.inputTokens,
				OutputTokens: tc.outputTokens,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed PostToolEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.DurationMs != tc.durationMs {
				t.Errorf("Expected DurationMs %d, got %d", tc.durationMs, parsed.DurationMs)
			}

			if parsed.InputTokens != tc.inputTokens {
				t.Errorf("Expected InputTokens %d, got %d", tc.inputTokens, parsed.InputTokens)
			}

			if parsed.OutputTokens != tc.outputTokens {
				t.Errorf("Expected OutputTokens %d, got %d", tc.outputTokens, parsed.OutputTokens)
			}
		})
	}
}

// TestPostToolEvent_ExistingParserUnchanged verifies ParsePostToolEvent still works
func TestPostToolEvent_ExistingParserUnchanged(t *testing.T) {
	// Test that existing ParsePostToolEvent function handles new fields
	jsonWithMLFields := `{
		"tool_name": "Read",
		"session_id": "sess-parser",
		"hook_event_name": "PostToolUse",
		"captured_at": 1768465600,
		"tool_input": {"file_path": "/test.go"},
		"tool_response": {"success": true},
		"duration_ms": 150,
		"model": "claude-haiku-4",
		"tier": "haiku",
		"sequence_index": 3
	}`

	reader := strings.NewReader(jsonWithMLFields)
	event, err := ParsePostToolEvent(reader, 1*time.Second)

	if err != nil {
		t.Fatalf("ParsePostToolEvent failed with ML fields: %v", err)
	}

	// Core fields work
	if event.ToolName != "Read" {
		t.Errorf("Expected ToolName Read, got %s", event.ToolName)
	}

	// ML fields parsed correctly
	if event.DurationMs != 150 {
		t.Errorf("Expected DurationMs 150, got %d", event.DurationMs)
	}

	if event.Model != "claude-haiku-4" {
		t.Errorf("Expected Model claude-haiku-4, got %s", event.Model)
	}

	if event.SequenceIndex != 3 {
		t.Errorf("Expected SequenceIndex 3, got %d", event.SequenceIndex)
	}
}
