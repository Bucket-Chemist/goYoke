# Corpus Logger

Go-based hook for capturing Claude Code events to test fixtures.

## Purpose

This tool captures all hook events from Claude Code sessions and appends them to a JSONL file for analysis and testing. It validates the event schema defined in the gogent-fortress migration project before Week 1 (GOgent-001).

## Features

- **Zero dependencies**: stdlib only
- **Non-blocking**: 5-second STDIN timeout prevents hanging
- **XDG-compliant**: Respects `XDG_RUNTIME_DIR`, `XDG_CACHE_HOME`, or falls back to `~/.cache/gogent`
- **Pass-through**: Echoes unchanged input to STDOUT for hook chain compatibility
- **Graceful errors**: Logs errors to stderr but never fails the hook chain

## Building

```bash
go build -o corpus-logger main.go
```

## Installation

Copy the compiled binary to Claude Code hooks directory:

```bash
cp corpus-logger ~/.claude/hooks/zzz-corpus-logger
chmod +x ~/.claude/hooks/zzz-corpus-logger
```

The `zzz-` prefix ensures it runs last in the hook chain.

## Output Location

Events are appended to:
1. `$XDG_RUNTIME_DIR/gogent/event-corpus-raw.jsonl` (if set)
2. `$XDG_CACHE_HOME/gogent/event-corpus-raw.jsonl` (if set)
3. `~/.cache/gogent/event-corpus-raw.jsonl` (fallback)

## Event Schema

Each line is a JSON object with:

```json
{
  "tool_name": "Task",
  "tool_input": {"model": "sonnet", "prompt": "...", "subagent_type": "general-purpose"},
  "tool_response": null,
  "session_id": "abc-123",
  "hook_event_name": "PreToolUse",
  "captured_at": 1737840000
}
```

Fields:
- `tool_name`: Name of the tool being invoked (Task, Bash, Read, etc.)
- `tool_input`: Input parameters (only present for PreToolUse events)
- `tool_response`: Output/response (only present for PostToolUse events)
- `session_id`: Claude Code session identifier
- `hook_event_name`: Hook event type (PreToolUse, PostToolUse, SessionStart, etc.)
- `captured_at`: Unix timestamp (seconds since epoch)

## Testing

```bash
go test -v
```

Tests validate:
- JSON parsing for valid/invalid events
- XDG path resolution priority
- Empty input handling
- Error conditions

## Integration Test

```bash
echo '{"tool_name":"Test","session_id":"test-123","hook_event_name":"PreToolUse"}' | ./corpus-logger
```

Should:
1. Echo the JSON to stdout (pass-through)
2. Append the event with `captured_at` timestamp to output file
3. Exit with code 0

## Design Decisions

### Why Go?
- First Go code for gogent-fortress migration (validates tooling setup)
- Compiled binary: no runtime dependencies
- Fast STDIN processing: ~2ms per event
- Easy cross-compilation for future deployments

### Why JSONL?
- Append-only: no file locking issues
- One event per line: easy parsing with `jq` or standard tools
- Streamable: can process files while they're being written

### Why XDG compliance?
- `XDG_RUNTIME_DIR`: session-specific, auto-cleaned on logout
- `XDG_CACHE_HOME`: user-configurable persistent cache
- `~/.cache`: standard fallback for all Linux systems

### Why 5-second timeout?
- Prevents indefinite blocking on STDIN (per M-6 fix)
- Sufficient for all realistic hook event sizes (< 100KB)
- Fails fast if hook chain is broken

### Why pass-through before processing?
- Hook chain continues immediately
- Corpus logging is fire-and-forget (errors don't block workflow)
- Maintains exact input (no transformation risk)

## Maintenance

This is a standalone tool with no external dependencies. Update only if:
1. Event schema changes (modify `HookEvent` struct)
2. Output format changes (modify `appendEvent` function)
3. XDG path priority changes (modify `resolveOutputPath` function)

## License

Part of the gogent-fortress migration project.
