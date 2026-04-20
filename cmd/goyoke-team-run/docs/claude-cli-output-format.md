# Claude CLI JSON Output Format

**Verified on**: 2026-02-07
**Claude CLI version**: 2.1.32 (Claude Code)
**Command**: `claude -p --output-format json`

## Output Structure

The output is a **JSON array** of event objects, NOT a single JSON object. Each element has a `type` field.

```
[ {init_event}, {assistant_event}, ..., {result_event} ]
```

For cost extraction, only the **last element** (type=`result`) matters.

## Event Types

### 1. Init Event (always first)

```json
{
  "type": "system",
  "subtype": "init",
  "cwd": "/path/to/cwd",
  "session_id": "uuid",
  "tools": ["Task", "Read", "Write", ...],
  "mcp_servers": [],
  "model": "claude-opus-4-6",
  "permissionMode": "default",
  "claude_code_version": "2.1.32",
  "agents": [...],
  "skills": [...],
  "uuid": "uuid"
}
```

### 2. Assistant Event(s) (one per turn)

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-opus-4-6",
    "id": "msg_...",
    "type": "message",
    "role": "assistant",
    "content": [{"type": "text", "text": "..."}],
    "stop_reason": null,
    "usage": {
      "input_tokens": 2,
      "cache_creation_input_tokens": 20157,
      "cache_read_input_tokens": 17844,
      "output_tokens": 1,
      "service_tier": "standard"
    }
  },
  "session_id": "uuid",
  "uuid": "uuid"
}
```

### 3. Tool Use Events (if tools are called)

Tool use appears as `type: "assistant"` with `content[].type: "tool_use"`.
Tool results appear as `type: "user"` with `content[].type: "tool_result"`.

### 4. Result Event (always last)

This is the **primary event for cost extraction**.

#### Success Case

```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 2457,
  "duration_api_ms": 2428,
  "num_turns": 1,
  "result": "4",
  "stop_reason": null,
  "session_id": "uuid",
  "total_cost_usd": 0.13503825000000003,
  "usage": {
    "input_tokens": 2,
    "cache_creation_input_tokens": 20157,
    "cache_read_input_tokens": 17844,
    "output_tokens": 5,
    "server_tool_use": {
      "web_search_requests": 0,
      "web_fetch_requests": 0
    },
    "service_tier": "standard"
  },
  "modelUsage": {
    "claude-opus-4-6": {
      "inputTokens": 2,
      "outputTokens": 5,
      "cacheReadInputTokens": 17844,
      "cacheCreationInputTokens": 20157,
      "webSearchRequests": 0,
      "costUSD": 0.13503825000000003,
      "contextWindow": 200000,
      "maxOutputTokens": 32000
    }
  },
  "permission_denials": [],
  "uuid": "uuid"
}
```

#### Error Case (invalid model)

```json
{
  "type": "result",
  "subtype": "success",
  "is_error": true,
  "duration_ms": 439,
  "duration_api_ms": 0,
  "num_turns": 1,
  "result": "There's an issue with the selected model...",
  "stop_reason": "stop_sequence",
  "session_id": "uuid",
  "total_cost_usd": 0,
  "usage": {
    "input_tokens": 0,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "output_tokens": 0
  },
  "modelUsage": {},
  "permission_denials": [],
  "uuid": "uuid"
}
```

Note: `subtype` is still `"success"` even when `is_error: true`.

#### Budget Exceeded Case

```json
{
  "type": "result",
  "subtype": "error_max_budget_usd",
  "is_error": false,
  "duration_ms": 10845,
  "duration_api_ms": 0,
  "num_turns": 1,
  "stop_reason": null,
  "session_id": "uuid",
  "total_cost_usd": 0.14341325,
  "usage": { ... },
  "modelUsage": {
    "claude-opus-4-6": {
      "inputTokens": 2,
      "outputTokens": 335,
      "cacheReadInputTokens": 17844,
      "cacheCreationInputTokens": 20177,
      "costUSD": 0.14341325,
      "contextWindow": 200000,
      "maxOutputTokens": 32000
    }
  },
  "permission_denials": [],
  "errors": []
}
```

Note: Budget exceeded uses `subtype: "error_max_budget_usd"` but `is_error: false`.
The `result` field is **absent** (not null, entirely missing).
Cost is still reported in `total_cost_usd`.

## Cost Fields

| Field Path | Type | Always Present | Notes |
|------------|------|----------------|-------|
| `total_cost_usd` | float64 | Yes (in result event) | **Primary field**. Includes all turns. |
| `modelUsage.<model>.costUSD` | float64 | Yes (if any API calls made) | Per-model breakdown. Key is model ID string. |
| `usage.input_tokens` | int | Yes | Top-level aggregate. May be 0 if modelUsage has the real values. |
| `usage.output_tokens` | int | Yes | Top-level aggregate. |
| `modelUsage.<model>.inputTokens` | int | Yes | Per-model token counts. |
| `modelUsage.<model>.outputTokens` | int | Yes | Per-model token counts. |
| `modelUsage.<model>.cacheReadInputTokens` | int | Yes | Cache read tokens. |
| `modelUsage.<model>.cacheCreationInputTokens` | int | Yes | Cache creation tokens. |

**Key finding**: The field is `total_cost_usd` (NOT `cost_usd`). The existing `spawnAgent.ts` code that tries `cost_usd || total_cost_usd` has the fallback order backwards — `total_cost_usd` is the actual field.

## Result Event Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"result"` |
| `subtype` | string | `"success"` or `"error_max_budget_usd"` |
| `is_error` | bool | `true` if agent encountered an error |
| `duration_ms` | int | Total wall-clock time including overhead |
| `duration_api_ms` | int | Time spent in API calls only |
| `num_turns` | int | Number of agentic turns taken |
| `result` | string? | Agent's final text output. **Missing (not null) on budget exceeded.** |
| `stop_reason` | string? | Why the agent stopped. `null` for normal, `"stop_sequence"` for errors. |
| `session_id` | string | Session UUID |
| `total_cost_usd` | float64 | Total cost for entire session |
| `usage` | object | Aggregate token usage |
| `modelUsage` | object | Per-model usage breakdown (key = model ID) |
| `permission_denials` | array | Tool permission denials (empty if --allowedTools covers all used tools) |

## Parsing Strategy for Go Binary

1. Parse entire stdout as JSON array: `[]json.RawMessage`
2. Find last element (type=`result`)
3. Unmarshal into `CLIResultEvent` struct
4. Read `total_cost_usd` directly (always present, float64)
5. Read `result` for agent output text (may be missing on budget exceeded)
6. Check `is_error` for error detection
7. Check `subtype` for budget exceeded detection

## Verified Test Results

| Scenario | total_cost_usd | is_error | subtype | result present |
|----------|----------------|----------|---------|----------------|
| Short response ("What is 2+2?") | 0.135 | false | success | Yes |
| Write tool test (TC-001) | 0.269 | false | success | Yes |
| Read tool test (TC-001) | 0.166 | false | success | Yes |
| Invalid model | 0 | true | success | Yes (error message) |
| Budget exceeded (--max-budget-usd 0.001) | 0.143 | false | error_max_budget_usd | No |

## Compatibility Notes

- `spawnAgent.ts:parseCliOutput()` currently parses `result || output` and `cost_usd || total_cost_usd`
- The actual field names are `result` and `total_cost_usd`
- `cost_usd` does NOT exist — the fallback in spawnAgent.ts never triggers on the primary
- `output` does NOT exist — `result` is the correct field
- The Go binary should use `total_cost_usd` as the **only** cost field (no fallback needed)
