package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseCLIEvent — table-driven tests.
//
// All JSON fixtures are taken directly from the TUI-003 spike catalog
// (tickets/tui-migration/spike-results/ndjson-catalog.md) to ensure that
// real-world event shapes are covered.
// ---------------------------------------------------------------------------

func TestParseCLIEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string // reflect type name we assert via type-switch
		wantErr     bool
		wantNilBoth bool // (nil, nil) for blank lines
		check       func(t *testing.T, got any)
	}{
		// -------------------------------------------------------------------
		// Blank / whitespace lines.
		// -------------------------------------------------------------------
		{
			name:        "empty line returns nil nil",
			input:       "",
			wantNilBoth: true,
		},
		{
			name:        "whitespace-only line returns nil nil",
			input:       "   \t\n  ",
			wantNilBoth: true,
		},

		// -------------------------------------------------------------------
		// system:init
		// -------------------------------------------------------------------
		{
			name: "system:init basic",
			input: `{
				"type":"system","subtype":"init",
				"cwd":"/home/user/project",
				"session_id":"sess-001",
				"tools":["Task","Bash","Read","Write","Edit","Glob","Grep"],
				"mcp_servers":[{"name":"goyoke-poc","status":"connected"}],
				"model":"claude-opus-4-6[1m]",
				"permissionMode":"acceptEdits",
				"apiKeySource":"none",
				"claude_code_version":"2.1.76",
				"output_style":"default",
				"fast_mode_state":"off",
				"agents":["general-purpose","GO Pro"],
				"uuid":"uuid-sys-init"
			}`,
			wantType: "SystemInitEvent",
			check: func(t *testing.T, got any) {
				ev, ok := got.(SystemInitEvent)
				require.True(t, ok)
				assert.Equal(t, "system", ev.Type)
				assert.Equal(t, "init", ev.Subtype)
				assert.Equal(t, "/home/user/project", ev.CWD)
				assert.Equal(t, "sess-001", ev.SessionID)
				assert.Equal(t, "claude-opus-4-6[1m]", ev.Model)
				assert.Equal(t, "acceptEdits", ev.PermissionMode)
				assert.Equal(t, "2.1.76", ev.ClaudeCodeVersion)
				assert.Equal(t, "off", ev.FastModeState)
				require.Len(t, ev.MCPServers, 1)
				assert.Equal(t, "goyoke-poc", ev.MCPServers[0].Name)
				assert.Equal(t, "connected", ev.MCPServers[0].Status)
				tools := ev.ToolNames()
				assert.Equal(t, []string{"Task", "Bash", "Read", "Write", "Edit", "Glob", "Grep"}, tools)
			},
		},
		{
			name: "system:init tools as object array",
			input: `{
				"type":"system","subtype":"init",
				"cwd":"/tmp","session_id":"s","model":"m",
				"permissionMode":"default","claude_code_version":"1.0",
				"tools":[{"name":"Read","description":"reads a file"},{"name":"Write"}],
				"uuid":"u"
			}`,
			wantType: "SystemInitEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemInitEvent)
				assert.Equal(t, []string{"Read", "Write"}, ev.ToolNames())
			},
		},
		{
			name: "system:init empty tools",
			input: `{
				"type":"system","subtype":"init",
				"cwd":"/tmp","session_id":"s","model":"m",
				"permissionMode":"default","claude_code_version":"1.0",
				"tools":[],
				"uuid":"u"
			}`,
			wantType: "SystemInitEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemInitEvent)
				names := ev.ToolNames()
				assert.Empty(t, names)
			},
		},

		// -------------------------------------------------------------------
		// system:hook_started
		// -------------------------------------------------------------------
		{
			name: "system:hook_started",
			input: `{
				"type":"system","subtype":"hook_started",
				"hook_id":"hook-001","hook_name":"SessionStart:startup",
				"hook_event":"SessionStart","uuid":"u1","session_id":"s1"
			}`,
			wantType: "SystemHookEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemHookEvent)
				assert.Equal(t, "hook_started", ev.Subtype)
				assert.Equal(t, "hook-001", ev.HookID)
				assert.Equal(t, "SessionStart:startup", ev.HookName)
				assert.Equal(t, "SessionStart", ev.HookEvent)
				assert.Empty(t, ev.Outcome) // not set for hook_started
			},
		},

		// -------------------------------------------------------------------
		// system:hook_response
		// -------------------------------------------------------------------
		{
			name: "system:hook_response approved",
			input: `{
				"type":"system","subtype":"hook_response",
				"hook_id":"hook-001","hook_name":"SessionStart:startup",
				"hook_event":"SessionStart",
				"output":"ok","stdout":"loaded","stderr":"","exit_code":0,"outcome":"approved",
				"uuid":"u2","session_id":"s1"
			}`,
			wantType: "SystemHookEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemHookEvent)
				assert.Equal(t, "hook_response", ev.Subtype)
				assert.Equal(t, "approved", ev.Outcome)
				assert.Equal(t, 0, ev.ExitCode)
				assert.Equal(t, "loaded", ev.Stdout)
			},
		},
		{
			name: "system:hook_response error",
			input: `{
				"type":"system","subtype":"hook_response",
				"hook_id":"hook-002","hook_name":"PreToolUse:validate",
				"hook_event":"PreToolUse",
				"output":"","stdout":"","stderr":"validation failed","exit_code":1,"outcome":"error",
				"uuid":"u3","session_id":"s1"
			}`,
			wantType: "SystemHookEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemHookEvent)
				assert.Equal(t, "error", ev.Outcome)
				assert.Equal(t, 1, ev.ExitCode)
				assert.Equal(t, "validation failed", ev.Stderr)
			},
		},

		// -------------------------------------------------------------------
		// system:status (known-but-unobserved subtype → SystemStatusEvent)
		// -------------------------------------------------------------------
		{
			name: "system:status permission mode change",
			input: `{
				"type":"system","subtype":"status",
				"status":"ready","permissionMode":"bypassPermissions",
				"uuid":"u4","session_id":"s1"
			}`,
			wantType: "SystemStatusEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemStatusEvent)
				assert.Equal(t, "status", ev.Subtype)
				require.NotNil(t, ev.Status)
				assert.Equal(t, "ready", *ev.Status)
				assert.Equal(t, "bypassPermissions", ev.PermissionMode)
			},
		},
		{
			name: "system:compact_boundary routes to SystemStatusEvent",
			input: `{"type":"system","subtype":"compact_boundary","uuid":"u5"}`,
			wantType: "SystemStatusEvent",
			check: func(t *testing.T, got any) {
				ev := got.(SystemStatusEvent)
				assert.Equal(t, "compact_boundary", ev.Subtype)
			},
		},

		// -------------------------------------------------------------------
		// assistant
		// -------------------------------------------------------------------
		{
			name: "assistant text block",
			input: `{
				"type":"assistant",
				"message":{
					"id":"msg_001","type":"message","role":"assistant","model":"claude-opus-4-6",
					"content":[{"type":"text","text":"Hello, world!"}],
					"stop_reason":null,
					"usage":{"input_tokens":5,"output_tokens":3,"service_tier":"standard"}
				},
				"parent_tool_use_id":null,
				"session_id":"s1","uuid":"u6"
			}`,
			wantType: "AssistantEvent",
			check: func(t *testing.T, got any) {
				ev := got.(AssistantEvent)
				assert.Equal(t, "assistant", ev.Type)
				assert.Equal(t, "msg_001", ev.Message.ID)
				assert.Equal(t, "assistant", ev.Message.Role)
				assert.Nil(t, ev.Message.StopReason)
				require.Len(t, ev.Message.Content, 1)
				assert.Equal(t, "text", ev.Message.Content[0].Type)
				assert.Equal(t, "Hello, world!", ev.Message.Content[0].Text)
				assert.Nil(t, ev.ParentToolUseID)
				require.NotNil(t, ev.Message.Usage)
				assert.Equal(t, 5, ev.Message.Usage.InputTokens)
				assert.Equal(t, 3, ev.Message.Usage.OutputTokens)
				assert.Equal(t, "standard", ev.Message.Usage.ServiceTier)
			},
		},
		{
			name: "assistant tool_use block with caller",
			input: `{
				"type":"assistant",
				"message":{
					"id":"msg_002","type":"message","role":"assistant","model":"claude-opus-4-6",
					"content":[{
						"type":"tool_use","id":"toolu_001","name":"Read",
						"input":{"file_path":"/tmp/test.go"},
						"caller":{"type":"direct"}
					}],
					"stop_reason":"tool_use",
					"usage":{"input_tokens":100,"output_tokens":50,
						"cache_creation_input_tokens":16151,
						"cache_read_input_tokens":16324}
				},
				"parent_tool_use_id":null,
				"session_id":"s1","uuid":"u7"
			}`,
			wantType: "AssistantEvent",
			check: func(t *testing.T, got any) {
				ev := got.(AssistantEvent)
				require.Len(t, ev.Message.Content, 1)
				blk := ev.Message.Content[0]
				assert.Equal(t, "tool_use", blk.Type)
				assert.Equal(t, "toolu_001", blk.ID)
				assert.Equal(t, "Read", blk.Name)
				require.NotNil(t, blk.Caller)
				assert.Equal(t, "direct", blk.Caller.Type)
				require.NotNil(t, ev.Message.StopReason)
				assert.Equal(t, "tool_use", *ev.Message.StopReason)
				assert.Equal(t, 16151, ev.Message.Usage.CacheCreationInputTokens)
				assert.Equal(t, 16324, ev.Message.Usage.CacheReadInputTokens)
			},
		},
		{
			name: "assistant thinking block",
			input: `{
				"type":"assistant",
				"message":{
					"id":"msg_003","type":"message","role":"assistant","model":"claude-opus-4-6",
					"content":[{
						"type":"thinking",
						"thinking":"Let me think about this carefully...",
						"signature":"base64sig=="
					}],
					"stop_reason":"end_turn",
					"usage":{"input_tokens":10,"output_tokens":20}
				},
				"parent_tool_use_id":null,
				"session_id":"s1","uuid":"u8"
			}`,
			wantType: "AssistantEvent",
			check: func(t *testing.T, got any) {
				ev := got.(AssistantEvent)
				require.Len(t, ev.Message.Content, 1)
				blk := ev.Message.Content[0]
				assert.Equal(t, "thinking", blk.Type)
				assert.Equal(t, "Let me think about this carefully...", blk.Thinking)
				assert.Equal(t, "base64sig==", blk.Signature)
			},
		},
		{
			name: "assistant with parent_tool_use_id (subagent message)",
			input: `{
				"type":"assistant",
				"message":{"id":"msg_sub","type":"message","role":"assistant","model":"claude-haiku",
					"content":[],"stop_reason":"end_turn",
					"usage":{"input_tokens":1,"output_tokens":1}},
				"parent_tool_use_id":"toolu_parent",
				"session_id":"s1","uuid":"u9"
			}`,
			wantType: "AssistantEvent",
			check: func(t *testing.T, got any) {
				ev := got.(AssistantEvent)
				require.NotNil(t, ev.ParentToolUseID)
				assert.Equal(t, "toolu_parent", *ev.ParentToolUseID)
			},
		},

		// -------------------------------------------------------------------
		// user
		// -------------------------------------------------------------------
		{
			name: "user tool_result text",
			input: `{
				"type":"user",
				"message":{
					"role":"user",
					"content":[{
						"tool_use_id":"toolu_001","type":"tool_result",
						"content":"file contents here"
					}]
				},
				"parent_tool_use_id":null,
				"session_id":"s1","uuid":"u10",
				"tool_use_result":{"file":{"filePath":"/tmp/test.go","content":"package main","numLines":1,"startLine":1,"totalLines":1}}
			}`,
			wantType: "UserEvent",
			check: func(t *testing.T, got any) {
				ev := got.(UserEvent)
				assert.Equal(t, "user", ev.Type)
				require.Len(t, ev.Message.Content, 1)
				blk := ev.Message.Content[0]
				assert.Equal(t, "tool_result", blk.Type)
				assert.Equal(t, "toolu_001", blk.ToolUseID)
				assert.Nil(t, ev.ParentToolUseID)
				assert.NotEmpty(t, ev.ToolUseResult)
			},
		},
		{
			name: "user tool_result error variant",
			input: `{
				"type":"user",
				"message":{
					"role":"user",
					"content":[{
						"type":"tool_result",
						"content":"Error: file not found",
						"is_error":true,
						"tool_use_id":"toolu_002"
					}]
				},
				"session_id":"s1","uuid":"u11"
			}`,
			wantType: "UserEvent",
			check: func(t *testing.T, got any) {
				ev := got.(UserEvent)
				require.Len(t, ev.Message.Content, 1)
				blk := ev.Message.Content[0]
				assert.True(t, blk.IsError)
			},
		},
		{
			name: "user with string tool_use_result (permission denial)",
			input: `{
				"type":"user",
				"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"denied"}]},
				"tool_use_result":"Claude requested permissions but they were denied.",
				"session_id":"s1","uuid":"u12"
			}`,
			wantType: "UserEvent",
			check: func(t *testing.T, got any) {
				ev := got.(UserEvent)
				// ToolUseResult should be the raw JSON (a quoted string).
				assert.NotEmpty(t, ev.ToolUseResult)
			},
		},

		// -------------------------------------------------------------------
		// rate_limit_event
		// -------------------------------------------------------------------
		{
			name: "rate_limit_event allowed",
			input: `{
				"type":"rate_limit_event",
				"rate_limit_info":{
					"status":"allowed",
					"resetsAt":1774252800,
					"rateLimitType":"five_hour",
					"overageStatus":"rejected",
					"overageDisabledReason":"org_level_disabled",
					"isUsingOverage":false
				},
				"uuid":"u13","session_id":"s1"
			}`,
			wantType: "RateLimitEvent",
			check: func(t *testing.T, got any) {
				ev := got.(RateLimitEvent)
				assert.Equal(t, "rate_limit_event", ev.Type)
				assert.Equal(t, "allowed", ev.RateLimitInfo.Status)
				assert.Equal(t, "five_hour", ev.RateLimitInfo.RateLimitType)
				assert.Equal(t, int64(1774252800), ev.RateLimitInfo.ResetsAt)
				assert.False(t, ev.RateLimitInfo.IsUsingOverage)
				assert.Equal(t, "org_level_disabled", ev.RateLimitInfo.OverageDisabledReason)
			},
		},

		// -------------------------------------------------------------------
		// result
		// -------------------------------------------------------------------
		{
			name: "result success",
			input: `{
				"type":"result","subtype":"success",
				"is_error":false,
				"duration_ms":16872,"duration_api_ms":16747,"num_turns":3,
				"result":"Final response text","stop_reason":"end_turn",
				"session_id":"s1","total_cost_usd":0.158,
				"usage":{
					"input_tokens":5,"cache_creation_input_tokens":16706,
					"cache_read_input_tokens":81696,"output_tokens":510,
					"service_tier":"standard"
				},
				"modelUsage":{
					"claude-opus-4-6[1m]":{
						"inputTokens":5,"outputTokens":510,
						"cacheReadInputTokens":81696,
						"cacheCreationInputTokens":16706,
						"costUSD":0.158,"contextWindow":1000000,"maxOutputTokens":32000
					}
				},
				"permission_denials":[{
					"tool_name":"Write",
					"tool_use_id":"toolu_denied",
					"tool_input":{"file_path":"/etc/passwd","content":""}
				}],
				"fast_mode_state":"off","uuid":"u14"
			}`,
			wantType: "ResultEvent",
			check: func(t *testing.T, got any) {
				ev := got.(ResultEvent)
				assert.Equal(t, "result", ev.Type)
				assert.Equal(t, "success", ev.Subtype)
				assert.False(t, ev.IsError)
				assert.Equal(t, int64(16872), ev.DurationMS)
				assert.Equal(t, int64(16747), ev.DurationAPIMS)
				assert.Equal(t, 3, ev.NumTurns)
				assert.Equal(t, "Final response text", ev.Result)
				assert.InDelta(t, 0.158, ev.TotalCostUSD, 0.0001)
				assert.Equal(t, 510, ev.Usage.OutputTokens)
				assert.Equal(t, 16706, ev.Usage.CacheCreationInputTokens)
				assert.Equal(t, "standard", ev.Usage.ServiceTier)
				require.NotNil(t, ev.ModelUsage)
				entry, ok := ev.ModelUsage["claude-opus-4-6[1m]"]
				require.True(t, ok)
				assert.Equal(t, 5, entry.InputTokens)
				assert.Equal(t, 1000000, entry.ContextWindow)
				require.Len(t, ev.PermissionDenials, 1)
				assert.Equal(t, "Write", ev.PermissionDenials[0].ToolName)
				assert.Equal(t, "off", ev.FastModeState)
			},
		},
		{
			name: "result error subtype",
			input: `{
				"type":"result","subtype":"error",
				"is_error":true,
				"duration_ms":1000,"duration_api_ms":900,"num_turns":1,
				"result":"","stop_reason":"","session_id":"s1",
				"total_cost_usd":0.0,
				"usage":{"input_tokens":0,"output_tokens":0},
				"uuid":"u15"
			}`,
			wantType: "ResultEvent",
			check: func(t *testing.T, got any) {
				ev := got.(ResultEvent)
				assert.True(t, ev.IsError)
				assert.Equal(t, "error", ev.Subtype)
			},
		},

		// -------------------------------------------------------------------
		// stream_event
		// -------------------------------------------------------------------
		{
			name: "stream_event content_block_start tool_use",
			input: `{
				"type":"stream_event",
				"event":{
					"type":"content_block_start","index":1,
					"content_block":{"type":"tool_use","id":"toolu_xxx","name":"Read","input":{}}
				},
				"session_id":"s1","parent_tool_use_id":null,"uuid":"u16"
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				assert.Equal(t, "stream_event", ev.Type)
				assert.Equal(t, "content_block_start", ev.Event.Type)
				assert.Equal(t, 1, ev.Event.Index)
				require.NotNil(t, ev.Event.ContentBlock)
				assert.Equal(t, "tool_use", ev.Event.ContentBlock.Type)
				assert.Equal(t, "Read", ev.Event.ContentBlock.Name)
				assert.Nil(t, ev.ParentToolUseID)
			},
		},
		{
			name: "stream_event content_block_delta input_json_delta",
			input: `{
				"type":"stream_event",
				"event":{
					"type":"content_block_delta","index":1,
					"delta":{"type":"input_json_delta","partial_json":"/tmp/file.txt"}
				},
				"session_id":"s1","uuid":"u17"
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				assert.Equal(t, "content_block_delta", ev.Event.Type)
				require.NotNil(t, ev.Event.Delta)
				assert.Equal(t, "input_json_delta", ev.Event.Delta.Type)
				assert.Equal(t, "/tmp/file.txt", ev.Event.Delta.PartialJSON)
			},
		},
		{
			name: "stream_event content_block_delta text_delta",
			input: `{
				"type":"stream_event",
				"event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello "}}
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				require.NotNil(t, ev.Event.Delta)
				assert.Equal(t, "text_delta", ev.Event.Delta.Type)
				assert.Equal(t, "Hello ", ev.Event.Delta.Text)
			},
		},
		{
			name: "stream_event content_block_delta thinking_delta",
			input: `{
				"type":"stream_event",
				"event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"reasoning..."}}
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				require.NotNil(t, ev.Event.Delta)
				assert.Equal(t, "thinking_delta", ev.Event.Delta.Type)
				assert.Equal(t, "reasoning...", ev.Event.Delta.Thinking)
			},
		},
		{
			name: "stream_event content_block_delta signature_delta",
			input: `{
				"type":"stream_event",
				"event":{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"abc123=="}}
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				require.NotNil(t, ev.Event.Delta)
				assert.Equal(t, "signature_delta", ev.Event.Delta.Type)
				assert.Equal(t, "abc123==", ev.Event.Delta.Signature)
			},
		},
		{
			name: "stream_event content_block_stop",
			input: `{
				"type":"stream_event",
				"event":{"type":"content_block_stop","index":1},
				"session_id":"s1","uuid":"u18"
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				assert.Equal(t, "content_block_stop", ev.Event.Type)
				assert.Equal(t, 1, ev.Event.Index)
			},
		},
		{
			name: "stream_event message_start",
			input: `{
				"type":"stream_event",
				"event":{"type":"message_start","message":{"id":"msg_start","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6"}},
				"session_id":"s1","uuid":"u19"
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				assert.Equal(t, "message_start", ev.Event.Type)
				assert.NotEmpty(t, ev.Event.Message)
			},
		},
		{
			name: "stream_event message_stop",
			input: `{
				"type":"stream_event",
				"event":{"type":"message_stop"},
				"session_id":"s1","uuid":"u20"
			}`,
			wantType: "StreamEvent",
			check: func(t *testing.T, got any) {
				ev := got.(StreamEvent)
				assert.Equal(t, "message_stop", ev.Event.Type)
			},
		},

		// -------------------------------------------------------------------
		// CLIUnknownEvent — graceful fallback.
		// -------------------------------------------------------------------
		{
			name:     "unknown event type preserved",
			input:    `{"type":"future_event","some_field":"some_value","nested":{"x":1}}`,
			wantType: "CLIUnknownEvent",
			check: func(t *testing.T, got any) {
				ev, ok := got.(CLIUnknownEvent)
				require.True(t, ok)
				assert.Equal(t, "future_event", ev.Type)
				assert.NotEmpty(t, ev.RawJSON)
				// Verify raw bytes are preserved faithfully.
				var raw map[string]any
				require.NoError(t, json.Unmarshal(ev.RawJSON, &raw))
				assert.Equal(t, "some_value", raw["some_field"])
			},
		},
		{
			name:     "empty type field falls through to unknown",
			input:    `{"type":"","data":"x"}`,
			wantType: "CLIUnknownEvent",
			check: func(t *testing.T, got any) {
				ev := got.(CLIUnknownEvent)
				assert.Equal(t, "", ev.Type)
			},
		},

		// -------------------------------------------------------------------
		// Error cases.
		// -------------------------------------------------------------------
		{
			name:    "malformed JSON returns error",
			input:   `{not valid json`,
			wantErr: true,
		},
		{
			name:    "truncated JSON returns error",
			input:   `{"type":"assistant","message":{`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			if tc.wantNilBoth {
				assert.Nil(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ToolNames helper — standalone tests.
// ---------------------------------------------------------------------------

func TestSystemInitEvent_ToolNames(t *testing.T) {
	t.Run("nil tools field returns nil", func(t *testing.T) {
		ev := SystemInitEvent{}
		assert.Nil(t, ev.ToolNames())
	})

	t.Run("null JSON tools field returns nil", func(t *testing.T) {
		ev := SystemInitEvent{Tools: json.RawMessage("null")}
		assert.Nil(t, ev.ToolNames())
	})

	t.Run("string array", func(t *testing.T) {
		ev := SystemInitEvent{Tools: json.RawMessage(`["Read","Write","Bash"]`)}
		assert.Equal(t, []string{"Read", "Write", "Bash"}, ev.ToolNames())
	})

	t.Run("object array extracts name field", func(t *testing.T) {
		ev := SystemInitEvent{Tools: json.RawMessage(`[{"name":"Read"},{"name":"Write"}]`)}
		assert.Equal(t, []string{"Read", "Write"}, ev.ToolNames())
	})
}

// ---------------------------------------------------------------------------
// Struct field coverage — verify optional pointer fields marshal correctly.
// ---------------------------------------------------------------------------

func TestMessageUsage_ZeroValues(t *testing.T) {
	u := MessageUsage{InputTokens: 10, OutputTokens: 5}
	data, err := json.Marshal(u)
	require.NoError(t, err)

	var back MessageUsage
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, u, back)
	// Optional fields omitted when zero.
	assert.NotContains(t, string(data), "cache_creation_input_tokens")
	assert.NotContains(t, string(data), "service_tier")
}

func TestModelUsageEntry_RoundTrip(t *testing.T) {
	entry := ModelUsageEntry{
		InputTokens:  100,
		OutputTokens: 50,
		CostUSD:      0.042,
		ContextWindow: 200000,
	}
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var back ModelUsageEntry
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, entry, back)
}

func TestPermissionDenial_RoundTrip(t *testing.T) {
	pd := PermissionDenial{
		ToolName:  "Write",
		ToolUseID: "toolu_abc",
		ToolInput: json.RawMessage(`{"file_path":"/etc/passwd"}`),
	}
	data, err := json.Marshal(pd)
	require.NoError(t, err)

	var back PermissionDenial
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, pd.ToolName, back.ToolName)
	assert.Equal(t, pd.ToolUseID, back.ToolUseID)
}

func TestContentBlock_ToolResult_IsError(t *testing.T) {
	line := []byte(`{
		"type":"user",
		"message":{"role":"user","content":[{
			"type":"tool_result","tool_use_id":"t1",
			"content":"Error: not found","is_error":true
		}]},
		"session_id":"s","uuid":"u"
	}`)
	got, err := ParseCLIEvent(line)
	require.NoError(t, err)
	ev := got.(UserEvent)
	require.Len(t, ev.Message.Content, 1)
	assert.True(t, ev.Message.Content[0].IsError)
}

// ---------------------------------------------------------------------------
// JSON error paths — verify all branches in ParseCLIEvent return errors when
// the outer discriminator succeeds but the full unmarshal fails.
// ---------------------------------------------------------------------------

func TestParseCLIEvent_JSONErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "assistant with bad message field",
			input: `{"type":"assistant","message":"not-an-object"}`,
		},
		{
			name:  "user with bad message field",
			input: `{"type":"user","message":"not-an-object"}`,
		},
		{
			name:  "rate_limit_event with bad rate_limit_info",
			input: `{"type":"rate_limit_event","rate_limit_info":"bad"}`,
		},
		{
			name:  "result with bad usage",
			input: `{"type":"result","usage":"bad"}`,
		},
		{
			name:  "stream_event with bad event field",
			input: `{"type":"stream_event","event":"bad"}`,
		},
		{
			name:  "system:init with bad mcp_servers",
			input: `{"type":"system","subtype":"init","mcp_servers":"bad"}`,
		},
		{
			name:  "system:hook_started with bad exit_code",
			input: `{"type":"system","subtype":"hook_started","exit_code":"not-int"}`,
		},
		{
			name:  "system:status with bad status",
			input: `{"type":"system","subtype":"status","status":99}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseCLIEvent([]byte(tc.input))
			require.Error(t, err, "expected error for input: %s", tc.input)
			assert.Nil(t, got)
		})
	}
}

// TestParseCLIEvent_NoBubbletea verifies that the cli package itself does not
// import bubbletea by checking that ParseCLIEvent returns plain Go values (any),
// not bubbletea tea.Msg values. This is a compile-time guarantee enforced by
// the lack of import in events.go; this test documents the design intent.
func TestParseCLIEvent_ReturnsAny(t *testing.T) {
	got, err := ParseCLIEvent([]byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u"}`))
	require.NoError(t, err)
	require.NotNil(t, got)
	_, ok := got.(ResultEvent)
	assert.True(t, ok, "expected ResultEvent, got %T", got)
}
