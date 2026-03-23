# TUI-003 Spike: NDJSON Event Catalog

**Date:** 2026-03-23
**Claude Code Version:** 2.1.76
**Data Sources:** TUI-001 captures (5 sessions), TUI-002 captures (2 sessions), TUI-003 multi-tool capture
**Flags:** `--input-format stream-json --output-format stream-json --include-partial-messages`

---

## 1. Complete Event Type Taxonomy

### 1.1 Top-Level Types (6 observed + 1 known)

| Type | Subtypes Observed | Description |
|------|-------------------|-------------|
| `system` | `init`, `hook_started`, `hook_response` | Session lifecycle. Known but unobserved: `status`, `compact_boundary` |
| `assistant` | — | LLM output (text, tool_use, thinking content blocks) |
| `user` | — | Tool execution results |
| `rate_limit_event` | — | Rate limit status after API call |
| `result` | `success` | Final session event with costs. Known but unobserved: `error` |
| `stream_event` | — | Raw SSE streaming (only with `--include-partial-messages`) |
| *(unknown)* | — | Parser must handle gracefully via log-and-continue |

### 1.2 stream_event.event.type Values (6 observed)

| SSE Type | Description |
|----------|-------------|
| `message_start` | New API message begins. Contains initial usage/model info. |
| `content_block_start` | New content block. `content_block.type` = `text` / `tool_use` / `thinking` |
| `content_block_delta` | Incremental content. `delta.type` = `text_delta` / `input_json_delta` / `thinking_delta` / `signature_delta` |
| `content_block_stop` | Content block complete. |
| `message_delta` | Message-level update (stop_reason changes). |
| `message_stop` | Message complete. |

### 1.3 Content Block Types

| Block Type | Appears In | Fields |
|-----------|------------|--------|
| `text` | `content_block_start` | `text` (empty initially) |
| `tool_use` | `content_block_start` | `id`, `name`, `input` (empty initially) |
| `thinking` | `content_block_start` | `thinking` (empty initially) |
| `text_delta` | `content_block_delta` | `text` (incremental) |
| `input_json_delta` | `content_block_delta` | `partial_json` (incremental JSON fragments) |
| `thinking_delta` | `content_block_delta` | `thinking` (incremental) |
| `signature_delta` | `content_block_delta` | `signature` (thinking signature, incremental) |

### 1.4 tool_use_result Types (4 observed)

| Type | Trigger | Key Fields |
|------|---------|------------|
| `text` | Read tool | `file.filePath`, `file.content`, `file.numLines`, `file.startLine`, `file.totalLines` |
| `update` | Write/Edit tool | `filePath`, `content`, `structuredPatch[].{oldStart,oldLines,newStart,newLines,lines[]}`, `originalFile` |
| *(Bash object)* | Bash tool | `stdout`, `stderr`, `interrupted`, `isImage`, `noOutputExpected` |
| *(string)* | Errors/denials | Plain error text (e.g., "Claude requested permissions...") |

---

## 2. Event Schemas

### 2.1 system:init

```json
{
  "type": "system",
  "subtype": "init",
  "cwd": "/path/to/workdir",
  "session_id": "uuid",
  "tools": ["Task", "Bash", "Read", "Write", "Edit", "Glob", "Grep", "..."],
  "mcp_servers": [
    {"name": "gofortress-poc", "status": "connected"}
  ],
  "model": "claude-opus-4-6[1m]",
  "permissionMode": "acceptEdits",
  "slash_commands": ["debug", "simplify", "..."],
  "apiKeySource": "none",
  "claude_code_version": "2.1.76",
  "output_style": "default",
  "agents": ["general-purpose", "Explore", "Plan", "GO Pro", "..."],
  "skills": ["debug", "simplify", "ticket", "..."],
  "plugins": [],
  "uuid": "uuid",
  "fast_mode_state": "off"
}
```

**Go TUI must extract:** `tools` (for tool panel), `mcp_servers` (verify gofortress connected), `model`, `permissionMode`, `session_id` (for resume), `claude_code_version`.

### 2.2 system:hook_started / system:hook_response

```json
{"type": "system", "subtype": "hook_started",
 "hook_id": "uuid", "hook_name": "SessionStart:startup",
 "hook_event": "SessionStart", "uuid": "uuid", "session_id": "uuid"}

{"type": "system", "subtype": "hook_response",
 "hook_id": "uuid", "hook_name": "SessionStart:startup",
 "hook_event": "SessionStart",
 "output": "...", "stdout": "...", "stderr": "...",
 "exit_code": 0, "outcome": "approved",
 "uuid": "uuid", "session_id": "uuid"}
```

**Go TUI:** Display hook status in status line. `outcome` values: `"approved"`, `"error"`.

### 2.3 assistant

Emitted per content block (same `message.id` for blocks in one turn).

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-opus-4-6",
    "id": "msg_xxx",
    "type": "message",
    "role": "assistant",
    "content": [
      {"type": "text", "text": "Response text"},
      {"type": "tool_use", "id": "toolu_xxx", "name": "Read",
       "input": {"file_path": "/path"}, "caller": {"type": "direct"}},
      {"type": "thinking", "thinking": "...", "signature": "base64..."}
    ],
    "stop_reason": null,
    "usage": {
      "input_tokens": 3,
      "cache_creation_input_tokens": 16151,
      "cache_read_input_tokens": 16324,
      "output_tokens": 2,
      "service_tier": "standard",
      "cache_creation": {
        "ephemeral_5m_input_tokens": 0,
        "ephemeral_1h_input_tokens": 16151
      }
    },
    "context_management": null
  },
  "parent_tool_use_id": null,
  "session_id": "uuid",
  "uuid": "uuid"
}
```

**Content block types:** `text`, `tool_use`, `thinking`
**`parent_tool_use_id`:** Non-null for subagent messages (identifies parent Task tool_use)
**`stop_reason`:** `null` while streaming, `"end_turn"` / `"tool_use"` when complete
**`caller.type`:** `"direct"` for normal tool use

### 2.4 user (tool results)

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "toolu_xxx",
        "type": "tool_result",
        "content": "tool output text"
      }
    ]
  },
  "parent_tool_use_id": null,
  "session_id": "uuid",
  "uuid": "uuid",
  "tool_use_result": { /* structured data — see §1.4 */ }
}
```

**Error variant:**
```json
{
  "content": [{
    "type": "tool_result",
    "content": "Error message",
    "is_error": true,
    "tool_use_id": "toolu_xxx"
  }]
}
```

**`tool_use_result` (bonus field):** Structured data the Go TUI can render directly:
- Read: file content, line numbers, metadata
- Write/Edit: structured diff patches
- Bash: stdout/stderr, interrupted flag
- Errors: plain string

### 2.5 rate_limit_event

```json
{
  "type": "rate_limit_event",
  "rate_limit_info": {
    "status": "allowed",
    "resetsAt": 1774252800,
    "rateLimitType": "five_hour",
    "overageStatus": "rejected",
    "overageDisabledReason": "org_level_disabled",
    "isUsingOverage": false
  },
  "uuid": "uuid",
  "session_id": "uuid"
}
```

**Go TUI:** Update status line with rate limit info. Show warning if `status != "allowed"`.

### 2.6 result

```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 16872,
  "duration_api_ms": 16747,
  "num_turns": 3,
  "result": "Final text response",
  "stop_reason": "end_turn",
  "session_id": "uuid",
  "total_cost_usd": 0.158,
  "usage": {
    "input_tokens": 5,
    "cache_creation_input_tokens": 16706,
    "cache_read_input_tokens": 81696,
    "output_tokens": 510,
    "server_tool_use": {
      "web_search_requests": 0,
      "web_fetch_requests": 0
    },
    "service_tier": "standard",
    "cache_creation": {
      "ephemeral_1h_input_tokens": 16706,
      "ephemeral_5m_input_tokens": 0
    }
  },
  "modelUsage": {
    "claude-opus-4-6[1m]": {
      "inputTokens": 5,
      "outputTokens": 510,
      "cacheReadInputTokens": 81696,
      "cacheCreationInputTokens": 16706,
      "webSearchRequests": 0,
      "costUSD": 0.158,
      "contextWindow": 1000000,
      "maxOutputTokens": 32000
    }
  },
  "permission_denials": [
    {
      "tool_name": "Write",
      "tool_use_id": "toolu_xxx",
      "tool_input": {"file_path": "/path", "content": "..."}
    }
  ],
  "fast_mode_state": "off",
  "uuid": "uuid"
}
```

**Go TUI must extract:** `total_cost_usd`, `modelUsage` (per-model breakdown), `num_turns`, `duration_ms`, `permission_denials`, `session_id`.

### 2.7 stream_event (with --include-partial-messages)

```json
{
  "type": "stream_event",
  "event": {
    "type": "content_block_start",
    "index": 1,
    "content_block": {
      "type": "tool_use",
      "id": "toolu_xxx",
      "name": "Read",
      "input": {}
    }
  },
  "session_id": "uuid",
  "parent_tool_use_id": null,
  "uuid": "uuid"
}
```

**Delta example (input_json_delta):**
```json
{
  "type": "stream_event",
  "event": {
    "type": "content_block_delta",
    "index": 1,
    "delta": {
      "type": "input_json_delta",
      "partial_json": "/tmp/file.txt"
    }
  }
}
```

---

## 3. Event Ordering per Turn

```
system:hook_started          ← Hook fires (if configured)
system:hook_response         ← Hook result
system:init                  ← Once per session (first event after hooks)

[Per API turn:]
stream_event(message_start)
  stream_event(content_block_start: text)
    stream_event(content_block_delta: text_delta) × N
  stream_event(content_block_stop)
  stream_event(content_block_start: tool_use)
    stream_event(content_block_delta: input_json_delta) × N
  stream_event(content_block_stop)
  stream_event(message_delta)            ← stop_reason set
  stream_event(message_stop)
assistant                                ← AGGREGATED full message

[Tool execution — no events during this phase]

rate_limit_event                         ← After API call
user                                     ← Tool result(s)

[Next turn repeats from message_start...]

result                                   ← Final event: costs, denials
```

### 3.1 Critical Timing Observations

1. **`assistant` (aggregated) arrives AFTER all `stream_event`s for that message** — for real-time display, parse `stream_event`s
2. **No events during tool execution** — gap between `assistant(tool_use)` and `user(tool_result)`
3. **`content_block_start(tool_use)` gives tool name immediately** — use for "Running Read..." display
4. **`rate_limit_event` arrives between turns** — not guaranteed every turn
5. **Multiple tool_use blocks can appear in one assistant message** — parallel tool execution

---

## 4. Stdin Protocol

### 4.1 Format

With `--input-format stream-json`, send NDJSON on stdin:

```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Your message here"}]}}
```

**Rules:**
- One JSON object per line (no trailing content on same line)
- Session stays open until stdin EOF or `result` event
- Multiple messages allowed (multi-turn)
- `--replay-user-messages` echoes messages back on stdout for acknowledgment

### 4.2 Without --input-format stream-json

With plain `-p` mode, prompt is passed as CLI argument or on stdin as plain text. Stream-json input is required for bidirectional control.

---

## 5. Known but Unobserved Events

These exist in the TypeScript TUI's event handling but were not triggered during spike captures:

| Event | Source | When |
|-------|--------|------|
| `system:status` | SessionManager.ts:571 | Permission mode changes mid-session |
| `system:compact_boundary` | SessionManager.ts:571 | Context window compaction occurs |
| `result:error` | SessionManager.ts:571 | API error or session failure |

**Recommendation for Go parser:** Use `json.RawMessage` for unknown fields. Emit `CLIUnknownMsg` for unrecognized types — don't crash.

---

## 6. Comparison with Braintrust v2.0 Catalog (Section 4.3)

| Braintrust Prediction | Actual |
|-----------------------|--------|
| "6-7 event types" | ✅ 6 confirmed + stream_event = 7 total |
| system.init with tools/model | ✅ Confirmed, plus agents, skills, plugins, slash_commands |
| assistant with text/tool_use | ✅ Plus thinking blocks |
| user with tool_result | ✅ Plus structured tool_use_result bonus field |
| result with cost data | ✅ Plus modelUsage per-model breakdown, permission_denials |
| "NDJSON on stdout" | ✅ All events on stdout, hook errors only on stderr |
| "control_request for permissions" | ❌ Does not exist (TUI-001 finding) |

**New discoveries not in braintrust:**
- `stream_event` wrapping raw SSE (with `--include-partial-messages`)
- `rate_limit_event` as distinct type
- `tool_use_result` structured bonus field (diffs, file metadata, bash stdout/stderr)
- `thinking` + `signature_delta` content block types
- `hook_started` / `hook_response` system subtypes
- `caller` field on tool_use blocks

---

## 7. Go Type Mapping Recommendations (for TUI-012)

```go
// Top-level discriminator
type CLIEvent struct {
    Type    string          `json:"type"`
    Subtype string          `json:"subtype,omitempty"`
    Raw     json.RawMessage `json:"-"` // preserve for unknowns
}

// Specific types
type SystemInitEvent struct { ... }      // system:init
type AssistantEvent struct { ... }       // assistant
type UserEvent struct { ... }            // user
type RateLimitEvent struct { ... }       // rate_limit_event
type ResultEvent struct { ... }          // result
type StreamEvent struct { ... }          // stream_event
type HookEvent struct { ... }           // system:hook_started/hook_response

// Content blocks (in assistant messages)
type TextBlock struct { Text string }
type ToolUseBlock struct { ID, Name string; Input json.RawMessage; Caller CallerInfo }
type ThinkingBlock struct { Thinking, Signature string }

// Tool results (in user messages, structured bonus field)
type FileReadResult struct { FilePath, Content string; NumLines, StartLine, TotalLines int }
type FileUpdateResult struct { FilePath, Content string; StructuredPatch []PatchHunk; OriginalFile string }
type BashResult struct { Stdout, Stderr string; Interrupted, IsImage, NoOutputExpected bool }
```

---

_Generated by TUI-003 spike, 2026-03-23. Consolidates data from TUI-001 (5 captures) + TUI-002 (2 captures) + TUI-003 (1 capture)._
