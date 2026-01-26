package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEvent_SystemEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string
		wantSubtype string
		wantSession string
	}{
		{
			name:        "system init",
			input:       `{"type":"system","subtype":"init","session_id":"abc-123"}`,
			wantType:    "system",
			wantSubtype: "init",
			wantSession: "abc-123",
		},
		{
			name:        "hook started",
			input:       `{"type":"system","subtype":"hook_started","session_id":"test-456","hook_id":"hook-1","hook_name":"PreToolUse"}`,
			wantType:    "system",
			wantSubtype: "hook_started",
			wantSession: "test-456",
		},
		{
			name:        "hook response",
			input:       `{"type":"system","subtype":"hook_response","session_id":"test-789","hook_id":"hook-2","stdout":"output","exit_code":0}`,
			wantType:    "system",
			wantSubtype: "hook_response",
			wantSession: "test-789",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, event.Type)
			assert.Equal(t, tc.wantSubtype, event.Subtype)
			assert.True(t, event.IsSystem())
			assert.False(t, event.IsAssistant())
			assert.False(t, event.IsResult())
			assert.False(t, event.IsError())

			// Parse as SystemEvent
			sysEvent, err := event.AsSystem()
			require.NoError(t, err)
			assert.Equal(t, tc.wantSession, sysEvent.SessionID)
		})
	}
}

func TestParseEvent_AssistantEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   string
		wantText   string
		wantModel  string
		wantPartial bool
	}{
		{
			name:     "simple text response",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello!"}]}}`,
			wantType: "assistant",
			wantText: "Hello!",
		},
		{
			name:     "response with model",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Response"}],"model":"claude-3-sonnet-20240229"}}`,
			wantType: "assistant",
			wantText: "Response",
			wantModel: "claude-3-sonnet-20240229",
		},
		{
			name:        "partial streaming response",
			input:       `{"type":"assistant","message":{"content":[{"type":"text","text":"Hel"}]},"partial":true}`,
			wantType:    "assistant",
			wantText:    "Hel",
			wantPartial: true,
		},
		{
			name:     "response with usage",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Done"}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
			wantType: "assistant",
			wantText: "Done",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, event.Type)
			assert.True(t, event.IsAssistant())
			assert.False(t, event.IsSystem())
			assert.Equal(t, tc.wantPartial, event.IsPartial())

			// Parse as AssistantEvent
			asst, err := event.AsAssistant()
			require.NoError(t, err)
			require.Len(t, asst.Message.Content, 1)
			assert.Equal(t, "text", asst.Message.Content[0].Type)
			assert.Equal(t, tc.wantText, asst.Message.Content[0].Text)
			assert.Equal(t, tc.wantPartial, asst.Partial)

			if tc.wantModel != "" {
				assert.Equal(t, tc.wantModel, asst.Message.Model)
			}
		})
	}
}

func TestParseEvent_ResultEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   string
		wantError  bool
		wantResult string
	}{
		{
			name:       "success result",
			input:      `{"type":"result","subtype":"success","session_id":"test-1","is_error":false,"result":"Completed","duration_ms":1234,"total_cost_usd":0.05}`,
			wantType:   "result",
			wantError:  false,
			wantResult: "Completed",
		},
		{
			name:       "error result",
			input:      `{"type":"result","subtype":"error","session_id":"test-2","is_error":true,"result":"Failed","duration_ms":567}`,
			wantType:   "result",
			wantError:  true,
			wantResult: "Failed",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, event.Type)
			assert.True(t, event.IsResult())
			assert.False(t, event.IsAssistant())

			// Parse as ResultEvent
			result, err := event.AsResult()
			require.NoError(t, err)
			assert.Equal(t, tc.wantError, result.IsError)
			assert.Equal(t, tc.wantResult, result.Result)
		})
	}
}

func TestParseEvent_ErrorEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantError string
	}{
		{
			name:      "simple error",
			input:     `{"type":"error","error":"parse error"}`,
			wantType:  "error",
			wantError: "parse error",
		},
		{
			name:      "error with code and message",
			input:     `{"type":"error","error":"API error","code":"rate_limit","message":"Rate limit exceeded"}`,
			wantType:  "error",
			wantError: "API error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, event.Type)
			assert.True(t, event.IsError())
			assert.False(t, event.IsSystem())

			// Parse as ErrorEvent
			errEvent, err := event.AsError()
			require.NoError(t, err)
			assert.Equal(t, tc.wantError, errEvent.Error)
		})
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "malformed json",
			input: `{invalid}`,
		},
		{
			name:  "incomplete object",
			input: `{"type":"system"`,
		},
		{
			name:  "plain text",
			input: `not json`,
		},
		{
			name:  "empty string",
			input: ``,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseEvent([]byte(tc.input))
			assert.Error(t, err)
		})
	}
}

func TestParseEvent_UnknownType(t *testing.T) {
	// Unknown event types should parse successfully as base Event
	input := `{"type":"unknown","data":"something"}`
	event, err := ParseEvent([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "unknown", event.Type)
	assert.False(t, event.IsSystem())
	assert.False(t, event.IsAssistant())
	assert.False(t, event.IsResult())
	assert.False(t, event.IsError())
}

func TestEvent_AsWrongType(t *testing.T) {
	// Trying to parse as wrong type should return error
	input := `{"type":"system","subtype":"init","session_id":"test"}`
	event, err := ParseEvent([]byte(input))
	require.NoError(t, err)

	// Try to parse system event as assistant
	_, err = event.AsAssistant()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not assistant")

	// Try to parse system event as result
	_, err = event.AsResult()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not result")
}

func TestEvent_GetSessionID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSession string
	}{
		{
			name:        "system event",
			input:       `{"type":"system","subtype":"init","session_id":"sys-123"}`,
			wantSession: "sys-123",
		},
		{
			name:        "assistant event",
			input:       `{"type":"assistant","session_id":"asst-456","message":{"content":[]}}`,
			wantSession: "asst-456",
		},
		{
			name:        "result event",
			input:       `{"type":"result","session_id":"res-789","is_error":false,"result":"done","duration_ms":100}`,
			wantSession: "res-789",
		},
		{
			name:        "error event no session",
			input:       `{"type":"error","error":"test"}`,
			wantSession: "",
		},
		{
			name:        "unknown event type",
			input:       `{"type":"unknown"}`,
			wantSession: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantSession, event.GetSessionID())
		})
	}
}

func TestEvent_RoundTrip(t *testing.T) {
	// Test that we can marshal and unmarshal events without data loss
	tests := []struct {
		name  string
		event interface{}
	}{
		{
			name: "system event",
			event: SystemEvent{
				Event:     Event{Type: "system", Subtype: "init"},
				SessionID: "test-123",
				HookID:    "hook-1",
				HookName:  "PreToolUse",
			},
		},
		{
			name: "assistant event",
			event: AssistantEvent{
				Event:     Event{Type: "assistant"},
				SessionID: "test-456",
				Partial:   false,
				Message: AssistantMessage{
					Content: []ContentBlock{
						{Type: "text", Text: "Hello world"},
					},
					Model: "claude-3-sonnet-20240229",
					Usage: Usage{InputTokens: 10, OutputTokens: 5},
				},
			},
		},
		{
			name: "result event",
			event: ResultEvent{
				Event:        Event{Type: "result", Subtype: "success"},
				SessionID:    "test-789",
				IsError:      false,
				Result:       "Success",
				DurationMs:   1500,
				TotalCostUSD: 0.025,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tc.event)
			require.NoError(t, err)

			// Parse back
			event, err := ParseEvent(data)
			require.NoError(t, err)

			// Verify type matches
			switch tc.event.(type) {
			case SystemEvent:
				assert.True(t, event.IsSystem())
				sysEvent, err := event.AsSystem()
				require.NoError(t, err)
				assert.Equal(t, "test-123", sysEvent.SessionID)
			case AssistantEvent:
				assert.True(t, event.IsAssistant())
				asstEvent, err := event.AsAssistant()
				require.NoError(t, err)
				assert.Equal(t, "test-456", asstEvent.SessionID)
			case ResultEvent:
				assert.True(t, event.IsResult())
				resEvent, err := event.AsResult()
				require.NoError(t, err)
				assert.Equal(t, "test-789", resEvent.SessionID)
			}
		})
	}
}

func TestUserMessage_Marshaling(t *testing.T) {
	msg := UserMessage{
		Type: "user",
		Message: UserContent{
			Role: "user",
			Content: []ContentBlock{
				{Type: "text", Text: "Test message"},
			},
		},
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded UserMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "user", decoded.Type)
	assert.Equal(t, "user", decoded.Message.Role)
	assert.Equal(t, []ContentBlock{{Type: "text", Text: "Test message"}}, decoded.Message.Content)
}

func TestAssistantMessage_MultipleContentBlocks(t *testing.T) {
	input := `{
		"type": "assistant",
		"message": {
			"content": [
				{"type": "text", "text": "First block"},
				{"type": "text", "text": "Second block"},
				{"type": "tool_use", "id": "tool-1", "name": "Read"}
			],
			"model": "claude-3-sonnet-20240229"
		}
	}`

	event, err := ParseEvent([]byte(input))
	require.NoError(t, err)

	asst, err := event.AsAssistant()
	require.NoError(t, err)
	require.Len(t, asst.Message.Content, 3)

	assert.Equal(t, "text", asst.Message.Content[0].Type)
	assert.Equal(t, "First block", asst.Message.Content[0].Text)

	assert.Equal(t, "text", asst.Message.Content[1].Type)
	assert.Equal(t, "Second block", asst.Message.Content[1].Text)

	assert.Equal(t, "tool_use", asst.Message.Content[2].Type)
	assert.Equal(t, "tool-1", asst.Message.Content[2].ID)
	assert.Equal(t, "Read", asst.Message.Content[2].Name)
}

func TestTimestampedEvent_Parsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectTime    bool
		wantType      string
	}{
		{
			name:       "event with timestamp",
			input:      `{"type":"system","subtype":"init","session_id":"test","timestamp":"2026-01-26T10:00:00Z"}`,
			expectTime: true,
			wantType:   "system",
		},
		{
			name:       "event without timestamp",
			input:      `{"type":"assistant","message":{"content":[]}}`,
			expectTime: false,
			wantType:   "assistant",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			te, err := ParseTimestampedEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, te.Type)

			if tc.expectTime {
				assert.False(t, te.Timestamp.IsZero())
				expectedTime, _ := time.Parse(time.RFC3339, "2026-01-26T10:00:00Z")
				assert.Equal(t, expectedTime, te.Timestamp)
			} else {
				assert.True(t, te.Timestamp.IsZero())
			}
		})
	}
}

func TestSystemEvent_HookFields(t *testing.T) {
	input := `{
		"type": "system",
		"subtype": "hook_response",
		"session_id": "test-123",
		"hook_id": "hook-456",
		"hook_name": "PostToolUse",
		"exit_code": 0,
		"stdout": "Hook output here"
	}`

	event, err := ParseEvent([]byte(input))
	require.NoError(t, err)

	sysEvent, err := event.AsSystem()
	require.NoError(t, err)
	assert.Equal(t, "hook_response", sysEvent.Subtype)
	assert.Equal(t, "hook-456", sysEvent.HookID)
	assert.Equal(t, "PostToolUse", sysEvent.HookName)
	assert.Equal(t, 0, sysEvent.ExitCode)
	assert.Equal(t, "Hook output here", sysEvent.Stdout)
}

func TestResultEvent_CostTracking(t *testing.T) {
	input := `{
		"type": "result",
		"subtype": "success",
		"session_id": "test-123",
		"is_error": false,
		"result": "Task completed",
		"duration_ms": 5432,
		"total_cost_usd": 0.123
	}`

	event, err := ParseEvent([]byte(input))
	require.NoError(t, err)

	result, err := event.AsResult()
	require.NoError(t, err)
	assert.Equal(t, false, result.IsError)
	assert.Equal(t, int64(5432), result.DurationMs)
	assert.Equal(t, 0.123, result.TotalCostUSD)
}

func TestAssistantEvent_PartialMessages(t *testing.T) {
	// Simulate streaming: multiple partial messages followed by final
	partials := []string{
		`{"type":"assistant","message":{"content":[{"type":"text","text":"H"}]},"partial":true}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"He"}]},"partial":true}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Hel"}]},"partial":true}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]},"partial":false}`,
	}

	for i, input := range partials {
		event, err := ParseEvent([]byte(input))
		require.NoError(t, err, "failed to parse partial %d", i)

		assert.True(t, event.IsAssistant())

		isLast := i == len(partials)-1
		assert.Equal(t, !isLast, event.IsPartial(), "partial %d", i)

		asst, err := event.AsAssistant()
		require.NoError(t, err)
		assert.Equal(t, !isLast, asst.Partial)
	}
}

// TestParseEvent_RealClaudeOutput tests with mock Claude binary output format
func TestParseEvent_RealClaudeOutput(t *testing.T) {
	// These match the actual output from mock-claude.go
	tests := []struct {
		name string
		input string
		check func(t *testing.T, event Event)
	}{
		{
			name: "mock init event",
			input: `{"type":"system","subtype":"init","session_id":"test-123"}`,
			check: func(t *testing.T, event Event) {
				assert.True(t, event.IsSystem())
				sysEvent, err := event.AsSystem()
				require.NoError(t, err)
				assert.Equal(t, "init", sysEvent.Subtype)
				assert.Equal(t, "test-123", sysEvent.SessionID)
			},
		},
		{
			name: "mock echo response",
			input: `{"type":"assistant","message":{"content":[{"type":"text","text":"Echo: test message"}]}}`,
			check: func(t *testing.T, event Event) {
				assert.True(t, event.IsAssistant())
				asst, err := event.AsAssistant()
				require.NoError(t, err)
				require.Len(t, asst.Message.Content, 1)
				assert.Equal(t, "Echo: test message", asst.Message.Content[0].Text)
			},
		},
		{
			name: "mock error event",
			input: `{"type":"error","error":"scanner error: unexpected EOF"}`,
			check: func(t *testing.T, event Event) {
				assert.True(t, event.IsError())
				errEvent, err := event.AsError()
				require.NoError(t, err)
				assert.Contains(t, errEvent.Error, "scanner error")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tc.input))
			require.NoError(t, err)
			tc.check(t, event)
		})
	}
}
