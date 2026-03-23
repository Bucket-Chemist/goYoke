// Package cli provides Go struct definitions for all NDJSON event types
// emitted by the Claude Code CLI when invoked with --output-format stream-json.
//
// All types are pure data — this package has no dependency on bubbletea.
// The model package is responsible for wrapping these values into tea.Msg types.
//
// Source of truth: tickets/tui-migration/spike-results/ndjson-catalog.md (TUI-003).
package cli

import (
	"encoding/json"
	"strings"
)

// ---------------------------------------------------------------------------
// Internal discriminator — first-pass parsing only.
// ---------------------------------------------------------------------------

// rawEvent is used for two-pass JSON parsing. First unmarshal just the
// type/subtype discriminator, then unmarshal the full event.
type rawEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
}

// ---------------------------------------------------------------------------
// Helper types shared across multiple events.
// ---------------------------------------------------------------------------

// MCPServerInfo describes an MCP server connection reported by the CLI at
// session initialisation.
type MCPServerInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CallerInfo identifies who invoked a tool_use block. The Type field is
// typically "direct" for first-party tool calls.
type CallerInfo struct {
	Type string `json:"type"`
}

// ContentBlock is a polymorphic content block that can appear inside assistant
// or user messages. The Type field discriminates which variant is active; all
// fields not relevant to a given variant will be zero-valued.
//
// Variants:
//   - "text"        — Text field populated
//   - "tool_use"    — ID, Name, Input, Caller populated
//   - "tool_result" — ToolUseID, Content, IsError populated
//   - "thinking"    — Thinking, Signature populated
type ContentBlock struct {
	Type string `json:"type"`

	// text variant
	Text string `json:"text,omitempty"`

	// tool_use variant
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
	Caller *CallerInfo     `json:"caller,omitempty"`

	// tool_result variant
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // string or []ContentBlock
	IsError   bool            `json:"is_error,omitempty"`

	// thinking variant
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// StreamDelta carries incremental content in a content_block_delta stream
// event. The Type field discriminates which variant is populated.
//
// Variants:
//   - "text_delta"        — Text populated
//   - "input_json_delta"  — PartialJSON populated
//   - "thinking_delta"    — Thinking populated
//   - "signature_delta"   — Signature populated
type StreamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// StreamEventPayload is the inner "event" object inside a StreamEvent. The
// Type field matches the SSE event type from the Anthropic streaming API.
//
// Observed Type values: "message_start", "content_block_start",
// "content_block_delta", "content_block_stop", "message_delta",
// "message_stop".
type StreamEventPayload struct {
	Type         string          `json:"type"`
	Index        int             `json:"index,omitempty"`
	ContentBlock *ContentBlock   `json:"content_block,omitempty"`
	Delta        *StreamDelta    `json:"delta,omitempty"`
	Message      json.RawMessage `json:"message,omitempty"`
	Usage        json.RawMessage `json:"usage,omitempty"`
}

// ---------------------------------------------------------------------------
// Usage and cost types.
// ---------------------------------------------------------------------------

// MessageUsage holds token counts for a single API message.
type MessageUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens,omitempty"`
	ServiceTier              string `json:"service_tier,omitempty"`
}

// ResultUsage holds aggregate token counts in the final result event.
// ServerToolUse is kept as raw JSON because its fields vary across CLI
// versions (web_search_requests, web_fetch_requests, etc.).
type ResultUsage struct {
	InputTokens              int             `json:"input_tokens"`
	OutputTokens             int             `json:"output_tokens"`
	CacheCreationInputTokens int             `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int             `json:"cache_read_input_tokens,omitempty"`
	ServiceTier              string          `json:"service_tier,omitempty"`
	ServerToolUse            json.RawMessage `json:"server_tool_use,omitempty"`
}

// ModelUsageEntry holds per-model cost and token breakdown from the result
// event's modelUsage map.
type ModelUsageEntry struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens,omitempty"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow,omitempty"`
	MaxOutputTokens          int     `json:"maxOutputTokens,omitempty"`
}

// PermissionDenial records a single tool invocation that was denied before
// the session ended.
type PermissionDenial struct {
	ToolName  string          `json:"tool_name"`
	ToolUseID string          `json:"tool_use_id"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
}

// ---------------------------------------------------------------------------
// Top-level event types.
// ---------------------------------------------------------------------------

// SystemInitEvent is emitted once per session as the first non-hook event.
// It carries session metadata, available tools, and connected MCP servers.
//
// JSON: type="system", subtype="init"
type SystemInitEvent struct {
	Type               string          `json:"type"`
	Subtype            string          `json:"subtype"`
	CWD                string          `json:"cwd"`
	SessionID          string          `json:"session_id"`
	Tools              json.RawMessage `json:"tools"` // []string or []object — use ToolNames()
	MCPServers         []MCPServerInfo `json:"mcp_servers,omitempty"`
	Model              string          `json:"model"`
	PermissionMode     string          `json:"permissionMode"`
	APIKeySource       string          `json:"apiKeySource,omitempty"`
	ClaudeCodeVersion  string          `json:"claude_code_version"`
	OutputStyle        string          `json:"output_style,omitempty"`
	FastModeState      string          `json:"fast_mode_state,omitempty"`
	SlashCommands      []string        `json:"slash_commands,omitempty"`
	Agents             []string        `json:"agents,omitempty"`
	Skills             []string        `json:"skills,omitempty"`
	Plugins            json.RawMessage `json:"plugins,omitempty"`
	UUID               string          `json:"uuid"`
}

// ToolNames extracts tool names from the Tools field, which can be either
// []string or []object (with a "name" key) depending on CLI version.
func (e *SystemInitEvent) ToolNames() []string {
	if len(e.Tools) == 0 {
		return nil
	}

	// Try []string first (most common in observed data).
	var names []string
	if err := json.Unmarshal(e.Tools, &names); err == nil {
		return names
	}

	// Fall back to []{"name": "..."} object array.
	var objs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(e.Tools, &objs); err == nil {
		names = make([]string, 0, len(objs))
		for _, o := range objs {
			names = append(names, o.Name)
		}
		return names
	}

	return nil
}

// SystemHookEvent is emitted when a hook fires (hook_started) or completes
// (hook_response). It covers both subtypes; fields only present in
// hook_response (Output, Stdout, Stderr, ExitCode, Outcome) are zero-valued
// for hook_started events.
//
// JSON: type="system", subtype="hook_started" or "hook_response"
type SystemHookEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`

	// hook_response only
	Output   string `json:"output,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Outcome  string `json:"outcome,omitempty"`
}

// SystemStatusEvent is emitted when the CLI reports a permission mode change
// or session compaction boundary. These event subtypes ("status",
// "compact_boundary") exist in the CLI source but were not triggered during
// TUI-003 spike captures.
//
// JSON: type="system", subtype=anything other than "init"/"hook_started"/"hook_response"
type SystemStatusEvent struct {
	Type           string  `json:"type"`
	Subtype        string  `json:"subtype"`
	Status         *string `json:"status,omitempty"`
	PermissionMode string  `json:"permissionMode,omitempty"`
	UUID           string  `json:"uuid,omitempty"`
	SessionID      string  `json:"session_id,omitempty"`
}

// AssistantMessage is the message object nested inside AssistantEvent.
type AssistantMessage struct {
	ID                string          `json:"id"`
	Type              string          `json:"type"`
	Role              string          `json:"role"`
	Model             string          `json:"model"`
	Content           []ContentBlock  `json:"content"`
	StopReason        *string         `json:"stop_reason"`
	Usage             *MessageUsage   `json:"usage,omitempty"`
	ContextManagement json.RawMessage `json:"context_management,omitempty"`
}

// AssistantEvent carries LLM output. Multiple events can share the same
// Message.ID when they belong to the same assistant turn.
//
// ParentToolUseID is non-nil for subagent messages and identifies the parent
// Task tool_use block.
//
// JSON: type="assistant"
type AssistantEvent struct {
	Type            string           `json:"type"`
	Message         AssistantMessage `json:"message"`
	ParentToolUseID *string          `json:"parent_tool_use_id"`
	SessionID       string           `json:"session_id"`
	UUID            string           `json:"uuid"`
}

// UserMessage is the message object nested inside UserEvent.
type UserMessage struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// UserEvent carries tool execution results. ToolUseResult is a flexible bonus
// field containing structured data for the most recent tool call; its shape
// varies by tool (see ndjson-catalog.md §1.4).
//
// JSON: type="user"
type UserEvent struct {
	Type            string          `json:"type"`
	Message         UserMessage     `json:"message"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	SessionID       string          `json:"session_id"`
	UUID            string          `json:"uuid"`
	ToolUseResult   json.RawMessage `json:"tool_use_result,omitempty"`
}

// RateLimitInfo is the nested rate limit payload inside RateLimitEvent.
type RateLimitInfo struct {
	Status                string `json:"status"`
	RateLimitType         string `json:"rateLimitType"`
	OverageStatus         string `json:"overageStatus"`
	OverageDisabledReason string `json:"overageDisabledReason,omitempty"`
	ResetsAt              int64  `json:"resetsAt"`
	IsUsingOverage        bool   `json:"isUsingOverage"`
}

// RateLimitEvent is emitted after each API call with current quota status.
// The TUI should display a warning when RateLimitInfo.Status != "allowed".
//
// JSON: type="rate_limit_event"
type RateLimitEvent struct {
	Type          string        `json:"type"`
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          string        `json:"uuid"`
	SessionID     string        `json:"session_id"`
}

// ResultEvent is always the final event in a session. It carries aggregate
// cost, timing, per-model usage, and any permission denials that occurred.
//
// JSON: type="result", subtype="success" or "error"
type ResultEvent struct {
	Type               string                     `json:"type"`
	Subtype            string                     `json:"subtype"`
	IsError            bool                       `json:"is_error"`
	DurationMS         int64                      `json:"duration_ms"`
	DurationAPIMS      int64                      `json:"duration_api_ms"`
	NumTurns           int                        `json:"num_turns"`
	Result             string                     `json:"result"`
	StopReason         string                     `json:"stop_reason"`
	SessionID          string                     `json:"session_id"`
	UUID               string                     `json:"uuid"`
	TotalCostUSD       float64                    `json:"total_cost_usd"`
	Usage              ResultUsage                `json:"usage"`
	ModelUsage         map[string]ModelUsageEntry `json:"modelUsage,omitempty"`
	PermissionDenials  []PermissionDenial         `json:"permission_denials,omitempty"`
	FastModeState      string                     `json:"fast_mode_state,omitempty"`
}

// StreamEvent wraps a raw Anthropic SSE event emitted by the CLI when the
// --include-partial-messages flag is active.
//
// JSON: type="stream_event"
type StreamEvent struct {
	Type            string             `json:"type"`
	Event           StreamEventPayload `json:"event"`
	SessionID       string             `json:"session_id"`
	ParentToolUseID *string            `json:"parent_tool_use_id"`
	UUID            string             `json:"uuid"`
}

// CLIUnknownEvent is the fallback for NDJSON lines whose type is not
// recognised by this package. RawJSON preserves the original bytes so that
// callers can log or forward unrecognised events without data loss.
type CLIUnknownEvent struct {
	Type    string          `json:"type"`
	RawJSON json.RawMessage `json:"-"`
}

// ---------------------------------------------------------------------------
// ParseCLIEvent — two-pass parser.
// ---------------------------------------------------------------------------

// ParseCLIEvent parses a single NDJSON line from the Claude Code CLI stdout
// stream. It performs two-pass JSON parsing: first unmarshal the type and
// subtype discriminator, then unmarshal the full event into the correct Go
// type.
//
// Return values:
//   - (event, nil)  — successfully parsed event; event is one of the typed
//     event structs in this package
//   - (nil, nil)    — empty or whitespace-only line; caller should skip
//   - (nil, err)    — JSON parse error on a non-empty line
//
// The return type is any so that callers are not forced to import bubbletea.
// The model package wraps these values into tea.Msg types.
func ParseCLIEvent(line []byte) (any, error) {
	// Skip blank / whitespace-only lines silently.
	if len(strings.TrimSpace(string(line))) == 0 {
		return nil, nil
	}

	// First pass: extract discriminator.
	var raw rawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, err
	}

	// Second pass: unmarshal into the correct concrete type.
	switch raw.Type {
	case "system":
		return parseSystemEvent(raw.Subtype, line)

	case "assistant":
		var ev AssistantEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	case "user":
		var ev UserEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	case "rate_limit_event":
		var ev RateLimitEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	case "result":
		var ev ResultEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	case "stream_event":
		var ev StreamEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	default:
		return CLIUnknownEvent{Type: raw.Type, RawJSON: json.RawMessage(line)}, nil
	}
}

// parseSystemEvent routes system events to the correct typed struct based on
// their subtype.
func parseSystemEvent(subtype string, line []byte) (any, error) {
	switch subtype {
	case "init":
		var ev SystemInitEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	case "hook_started", "hook_response":
		var ev SystemHookEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil

	default:
		// Covers "status", "compact_boundary", and any future subtypes.
		var ev SystemStatusEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		return ev, nil
	}
}
