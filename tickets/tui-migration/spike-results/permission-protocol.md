# TUI-001 Spike: Permission Wire Format Capture

**Date:** 2026-03-23
**Claude Code Version:** 2.1.76
**CLI Flags Tested:** `--output-format stream-json`, `--input-format stream-json`, `--include-partial-messages`, `--permission-mode {default,acceptEdits}`

---

## Executive Summary

**The ticket assumed `--permission-prompt-tool stdio` exists. It does not.**

There is **NO interactive permission prompt protocol** in Claude Code's stream-json pipe mode. Instead:

- Tools requiring permission are **auto-approved** (`acceptEdits`, `bypassPermissions`) or **silently denied** (`default`).
- Denied tools appear in the `result` event's `permission_denials` array with full input.
- The Go TUI must implement its own permission layer using one of the architecture options documented below.

---

## 1. NDJSON Event Types (Complete Catalog)

All events are newline-delimited JSON on **stdout** when using `--input-format stream-json --output-format stream-json`.

### 1.1 Top-Level Event Types

| Type | Subtypes | Description |
|------|----------|-------------|
| `system` | `hook_started`, `hook_response`, `init` | Session lifecycle events |
| `assistant` | — | LLM response (text, tool_use, thinking blocks) |
| `user` | — | Tool results returned to LLM |
| `rate_limit_event` | — | Rate limit status check |
| `result` | `success`, `error` | Final event with costs and permission_denials |
| `stream_event` | (see §2) | Raw SSE streaming events (only with `--include-partial-messages`) |

### 1.2 system:init Event

Emitted once at session start. Contains full session configuration.

```json
{
  "type": "system",
  "subtype": "init",
  "cwd": "/path/to/workdir",
  "session_id": "uuid",
  "tools": ["Task", "Bash", "Read", "Write", "Edit", ...],
  "mcp_servers": [{"name": "...", "status": "..."}],
  "model": "claude-opus-4-6[1m]",
  "permissionMode": "acceptEdits",
  "slash_commands": ["..."],
  "agents": ["..."],
  "skills": ["..."],
  "plugins": [],
  "claude_code_version": "2.1.76",
  "output_style": "default",
  "fast_mode_state": "off",
  "apiKeySource": "none",
  "uuid": "uuid"
}
```

**Key fields for Go TUI:**
- `tools` — available tool list
- `model` — active model with context suffix
- `permissionMode` — active permission mode
- `session_id` — for resume support
- `agents` — available agent types

### 1.3 assistant Event

Contains LLM output as content blocks. Emitted per-block (text and tool_use arrive as separate events with the same `message.id`).

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-opus-4-6",
    "id": "msg_xxx",
    "type": "message",
    "role": "assistant",
    "content": [
      {
        "type": "text",
        "text": "Response text here"
      }
    ],
    "stop_reason": null,
    "usage": {
      "input_tokens": 3,
      "cache_creation_input_tokens": 16151,
      "cache_read_input_tokens": 16324,
      "output_tokens": 2,
      "service_tier": "standard"
    },
    "context_management": null
  },
  "parent_tool_use_id": null,
  "session_id": "uuid",
  "uuid": "uuid"
}
```

**Content block types:**

| Block Type | Fields | Notes |
|-----------|--------|-------|
| `text` | `text` | Markdown response text |
| `tool_use` | `id`, `name`, `input`, `caller` | Tool invocation. `caller.type` = `"direct"` |
| `thinking` | `thinking`, `signature` | Extended thinking (when enabled) |

**tool_use example:**
```json
{
  "type": "tool_use",
  "id": "toolu_01xxx",
  "name": "Write",
  "input": {
    "file_path": "/path/to/file.txt",
    "content": "file contents"
  },
  "caller": {
    "type": "direct"
  }
}
```

### 1.4 user Event (Tool Results)

Contains tool execution results. Includes an extra `tool_use_result` field with structured data.

**Success (text file read):**
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "toolu_01xxx",
        "type": "tool_result",
        "content": "     1→Hello world - modify me\n"
      }
    ]
  },
  "parent_tool_use_id": null,
  "session_id": "uuid",
  "uuid": "uuid",
  "tool_use_result": {
    "type": "text",
    "file": {
      "filePath": "/path/to/file.txt",
      "content": "Hello world - modify me\n",
      "numLines": 2,
      "startLine": 1,
      "totalLines": 2
    }
  }
}
```

**Success (file write/edit):**
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "toolu_01xxx",
        "type": "tool_result",
        "content": "The file /path/to/file.txt has been updated successfully."
      }
    ]
  },
  "tool_use_result": {
    "type": "update",
    "filePath": "/path/to/file.txt",
    "content": "new content",
    "structuredPatch": [
      {
        "oldStart": 1,
        "oldLines": 1,
        "newStart": 1,
        "newLines": 1,
        "lines": ["-old line", "+new line"]
      }
    ],
    "originalFile": "old content"
  }
}
```

**Error (permission denied):**
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "type": "tool_result",
        "content": "Claude requested permissions to write to /path/file.txt, but you haven't granted it yet.",
        "is_error": true,
        "tool_use_id": "toolu_01xxx"
      }
    ]
  },
  "tool_use_result": "Error: Claude requested permissions to write to /path/file.txt, but you haven't granted it yet."
}
```

### 1.5 rate_limit_event

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

### 1.6 result Event (Final)

Always the last event. Contains session summary, costs, and **permission_denials**.

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
    }
  },
  "modelUsage": {
    "claude-opus-4-6[1m]": {
      "inputTokens": 5,
      "outputTokens": 510,
      "cacheReadInputTokens": 81696,
      "cacheCreationInputTokens": 16706,
      "costUSD": 0.158,
      "contextWindow": 1000000,
      "maxOutputTokens": 32000
    }
  },
  "permission_denials": [],
  "fast_mode_state": "off"
}
```

**`permission_denials` (when tools are denied):**
```json
"permission_denials": [
  {
    "tool_name": "Write",
    "tool_use_id": "toolu_01xxx",
    "tool_input": {
      "file_path": "/path/to/file.txt",
      "content": "file contents"
    }
  },
  {
    "tool_name": "AskUserQuestion",
    "tool_use_id": "toolu_01yyy",
    "tool_input": {
      "questions": [{"question": "What color?", "header": "Color", "options": [...]}]
    }
  }
]
```

---

## 2. Streaming Events (`--include-partial-messages`)

When `--include-partial-messages` is enabled, raw Anthropic API SSE events are wrapped in `stream_event` NDJSON envelopes. These arrive **between** the aggregated `assistant`/`user` events.

### 2.1 stream_event.event Types

| Event Type | Description |
|-----------|-------------|
| `message_start` | New message begins. Contains initial usage/model info. |
| `content_block_start` | New content block (text, tool_use, thinking). Has block type + initial data. |
| `content_block_delta` | Incremental content. `text_delta` for text, `input_json_delta` for tool input. |
| `content_block_stop` | Content block complete. |
| `message_delta` | Message-level update (stop_reason changes). |
| `message_stop` | Message complete. |

### 2.2 Streaming a tool_use (real-time tool input construction)

The Go TUI can observe tool input being constructed token-by-token:

```
content_block_start → {"type":"tool_use", "name":"Read", "id":"toolu_01xxx"}
content_block_delta → {"type":"input_json_delta", "partial_json":""}
content_block_delta → {"type":"input_json_delta", "partial_json":"{\""}
content_block_delta → {"type":"input_json_delta", "partial_json":"file"}
content_block_delta → {"type":"input_json_delta", "partial_json":"_p"}
content_block_delta → {"type":"input_json_delta", "partial_json":"at"}
content_block_delta → {"type":"input_json_delta", "partial_json":"h\": \""}
content_block_delta → {"type":"input_json_delta", "partial_json":"/tmp/tui-s"}
content_block_delta → {"type":"input_json_delta", "partial_json":"pike-001/t"}
content_block_delta → {"type":"input_json_delta", "partial_json":"est-file.tx"}
content_block_delta → {"type":"input_json_delta", "partial_json":"t\"}"}
content_block_stop
```

**Use in Go TUI:** Accumulate `partial_json` fragments. Once `content_block_stop` arrives, parse the complete JSON to get the tool input. Display the tool name from `content_block_start` immediately for real-time activity indication.

### 2.3 Event Ordering

```
system:hook_started
system:hook_response
system:init
stream_event(message_start)         ← API stream begins
stream_event(content_block_start)   ← text block
stream_event(content_block_delta)   ← text tokens...
stream_event(content_block_stop)
stream_event(content_block_start)   ← tool_use block
stream_event(content_block_delta)   ← input_json_delta tokens...
stream_event(content_block_stop)
stream_event(message_delta)         ← stop_reason: "tool_use"
stream_event(message_stop)
assistant                           ← AGGREGATED: full message with all blocks
                                    ← Tool executes here (not visible in stream)
user                                ← tool_result
stream_event(message_start)         ← Next turn begins
...
assistant                           ← AGGREGATED: response
result                              ← Final: costs, permission_denials
```

**Critical observation:** The aggregated `assistant` event (with tool_use) is emitted AFTER the streaming events for that message. The tool then executes, and the `user` event with tool_result follows. There is **no stream event during tool execution** — the Go TUI sees tool activity via `content_block_start(tool_use)` then silence until `user(tool_result)`.

---

## 3. Permission Behavior by Mode

| Permission Mode | Write/Edit | Bash | AskUserQuestion | EnterPlanMode |
|----------------|------------|------|-----------------|---------------|
| `acceptEdits` | ✅ Auto-approved | ✅ Auto-approved | ❌ Denied | ❌ Denied |
| `bypassPermissions` | ✅ Bypassed | ✅ Bypassed | ❌ Denied | ❌ Denied |
| `default` | ❌ Denied | ❌ Denied | ❌ Denied | ❌ Denied |
| `dontAsk` | ❌ Denied | ❌ Denied | ❌ Denied | ❌ Denied |

**Key findings:**

1. **`acceptEdits` approves Write/Edit/Bash but NOT interactive tools.** AskUserQuestion gets denied even in acceptEdits mode.
2. **`default` in pipe mode denies everything that would need a prompt.** There is no way to interactively approve.
3. **Denied tools appear in `permission_denials`** with full tool input — the Go TUI can replay/handle them.
4. **`--permission-prompt-tool stdio` does NOT exist.** This was the ticket's assumption. The flag does not exist in Claude Code 2.1.76.

### 3.1 What happens when a tool is denied

The denial is NOT a special event type. It's a normal `user` event with `is_error: true`:

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [{
      "type": "tool_result",
      "content": "Claude requested permissions to write to /path, but you haven't granted it yet.",
      "is_error": true,
      "tool_use_id": "toolu_01xxx"
    }]
  }
}
```

The LLM sees this error and typically responds with text like "Please approve the write when prompted." The session may then end (stop_reason: "end_turn") with the denial recorded in `result.permission_denials`.

---

## 4. Bidirectional Protocol (stdin/stdout)

### 4.1 Stdin Format

With `--input-format stream-json`, the parent process sends user messages as NDJSON on stdin:

```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Your prompt here"}]}}
```

**Rules:**
- One JSON object per line
- Each line must be valid JSON (no trailing content)
- Session stays open until stdin is closed (EOF) or `result` event is emitted
- Multiple user messages can be sent (multi-turn)

### 4.2 Stdout Format

All NDJSON events listed in §1 are emitted on stdout. Stderr receives hook failure messages only.

### 4.3 Session Lifecycle

```
Parent process                         Claude CLI
    │                                      │
    │──── stdin: user message ──────────►  │
    │                                      │
    │  ◄──── stdout: system:init ──────────│
    │  ◄──── stdout: assistant (text) ─────│
    │  ◄──── stdout: assistant (tool_use) ─│
    │                                      │ ← tool executes internally
    │  ◄──── stdout: user (tool_result) ───│
    │  ◄──── stdout: assistant (response) ─│
    │  ◄──── stdout: result ───────────────│
    │                                      │
    │──── stdin: EOF (close) ───────────►  │ ← session ends
```

### 4.4 `--replay-user-messages` Flag

When set, user messages sent via stdin are echoed back on stdout. Useful for the Go TUI to confirm message receipt and display in the conversation panel.

---

## 5. Architecture Options for Go TUI

### Option A: `acceptEdits` + Custom Permission Layer (RECOMMENDED)

```
Go TUI ──stdin──► Claude CLI (--permission-mode acceptEdits)
   │   ◄──stdout── NDJSON events
   │
   ├─ Parse assistant tool_use events
   ├─ For tools needing user approval (Write, Edit, Bash):
   │   → Show diff/preview in TUI modal
   │   → Tool already executed (acceptEdits approved it)
   │   → Display result
   ├─ For AskUserQuestion (denied by CLI):
   │   → Read tool_input from permission_denials
   │   → Show question modal in TUI
   │   → Send user's answer as next stdin message
   └─ For EnterPlanMode (denied by CLI):
       → Read plan context from tool_input
       → Show plan approval modal
       → Send approval as next stdin message
```

**Pros:** Simple, tools execute fast (no permission delay)
**Cons:** User sees result AFTER execution, can't cancel mid-flight. Write/Edit are fire-and-forget.
**Mitigation:** Use `--allowedTools` to restrict dangerous tools; show diffs post-hoc.

### Option B: `default` + Selective Retry

```
Go TUI ──stdin──► Claude CLI (--permission-mode default)
   │   ◄──stdout── NDJSON events
   │
   ├─ Parse permission_denials from result event
   ├─ For each denied tool:
   │   → Show approval modal with tool_input details
   │   → If approved: re-send prompt with --permission-mode acceptEdits
   │   → If denied: continue without that tool
   └─ Track which tools are pre-approved for future turns
```

**Pros:** User approves before execution
**Cons:** Costs 2x API turns for every permission (denied turn + approved retry)
**Not recommended** due to cost and latency doubling.

### Option C: MCP Side-Channel (TUI-004 Spike)

```
Go TUI ←──UDS──→ MCP Server (Go) ←──stdio──→ Claude CLI
   │
   ├─ MCP server registers tools: ask_user, confirm_action, select_option
   ├─ Claude CLI invokes MCP tools for user interaction
   ├─ MCP server forwards to Go TUI via UDS
   ├─ Go TUI shows modal, returns result via UDS
   └─ MCP server returns result to Claude CLI
```

**Pros:** Proper interactive flow, Claude can ask questions mid-conversation
**Cons:** Requires MCP server implementation (TUI-014), more complex architecture
**This is the approach used by the existing TypeScript TUI** (via SDK's canUseTool callback).

### Option D: Hybrid (RECOMMENDED FINAL)

Combine Options A and C:

1. **Phase 1 (TUI-001 through TUI-016):** Use Option A (`acceptEdits`)
   - Get basic TUI working fast
   - Tool activity visible via `stream_event` parsing
   - Post-hoc diff display for Write/Edit

2. **Phase 2 (TUI-014, TUI-018):** Add MCP side-channel (Option C)
   - Interactive permission modals
   - AskUserQuestion forwarding
   - Plan mode approval flow

---

## 6. Implications for Other Tickets

| Ticket | Impact |
|--------|--------|
| **TUI-012** (NDJSON event types) | Use the event catalog in §1. All types documented with field-level detail. |
| **TUI-013** (CLI subprocess driver) | Use bidirectional `--input-format stream-json --output-format stream-json`. Include `--include-partial-messages` for real-time streaming. |
| **TUI-014** (Go MCP server) | Required for Option C permission flow. MCP tools handle interactive prompts. |
| **TUI-018** (Permission flow) | No `control_request` protocol exists. Must use Option A (acceptEdits) or Option D (hybrid). |
| **TUI-004** (Side channel IPC) | UDS is the right choice for MCP-to-TUI communication. |

---

## 7. Raw Captures

### 7.1 Read Operation (acceptEdits)

File: `/tmp/tui-spike-001/ndjson-read.log`

Event sequence: `system:hook_started → system:hook_response → system:init → assistant(text) → assistant(tool_use:Read) → rate_limit_event → user(tool_result:text) → assistant(text) → result`

### 7.2 Write Operation (acceptEdits)

File: `/tmp/tui-spike-001/stderr-write.log`

Event sequence: `system:init → assistant(text) → assistant(tool_use:Write) → user(tool_result:error "read first") → rate_limit_event → assistant(thinking) → assistant(tool_use:Read) → user(tool_result:text) → assistant(tool_use:Write) → user(tool_result:update) → assistant(text) → result`

### 7.3 Write Operation (default — DENIED)

File: `/tmp/tui-spike-001/stdout-default-bidir.log`

Event sequence: `system:init → assistant(text) → assistant(tool_use:Read) → user(tool_result:text) → assistant(text) → assistant(tool_use:Write) → user(tool_result:error "permission denied") → assistant(text "blocked") → result(permission_denials:[Write])`

### 7.4 AskUserQuestion (acceptEdits — DENIED)

File: `/tmp/tui-spike-001/stdout-askuser.log`

Event sequence: `system:init → assistant(text) → assistant(tool_use:ToolSearch) → user(tool_result) → assistant(tool_use:AskUserQuestion) → user(tool_result:error "Answer questions?") → assistant(text) → result(permission_denials:[AskUserQuestion])`

### 7.5 Partial Messages (include-partial-messages)

File: `/tmp/tui-spike-001/stdout-partial.log`

50 events total. `stream_event` subtypes observed: `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`.

---

## 8. Conclusions

1. **`--permission-prompt-tool stdio` does not exist.** The ticket's central assumption was wrong. This spike caught it before implementation began.

2. **The NDJSON wire format is well-structured** with 6 top-level event types and 6 streaming subtypes. Parsing is straightforward.

3. **Permission handling in pipe mode is binary:** auto-approve or deny. No interactive middle ground exists in the CLI itself.

4. **The Go TUI should use `acceptEdits` for Phase 1** and add MCP-based interactive permissions in Phase 2 (TUI-014 + TUI-018).

5. **`--include-partial-messages` is essential** for real-time tool activity display. The `content_block_start(tool_use)` event gives immediate tool name visibility.

6. **`tool_use_result` is a bonus field** on `user` events — it contains structured data (file content, diffs, patches) that the Go TUI can render directly without re-parsing the flat text content.

---

_Generated by TUI-001 spike, 2026-03-23_
