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
		{"orchestrator", ClassOrchestrator},
		{"architect", ClassOrchestrator},
		{"einstein", ClassOrchestrator},
		{"python-pro", ClassImplementation},
		{"python-ux", ClassImplementation},
		{"go-pro", ClassImplementation},
		{"r-pro", ClassImplementation},
		{"r-shiny-pro", ClassImplementation},
		{"code-reviewer", ClassSpecialist},
		{"librarian", ClassSpecialist},
		{"tech-docs-writer", ClassSpecialist},
		{"scaffolder", ClassSpecialist},
		{"codebase-search", ClassCoordination},
		{"haiku-scout", ClassCoordination},
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
			name: "single timestamp",
			transcriptData: `{"timestamp": 1700000000, "content": "AGENT: test"}`,
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
