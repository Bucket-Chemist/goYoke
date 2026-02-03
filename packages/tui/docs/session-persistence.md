# Session Persistence (TUI-016)

Session persistence implementation for GOfortress TUI with Go CLI compatibility.

## Overview

Sessions are saved to `~/.claude/sessions/` in JSON format compatible with the Go TUI for rollback support.

## CLI Usage

```bash
# List available sessions
gofortress-tui --list

# Resume a specific session
gofortress-tui --session <session-id>

# Enable verbose logging
gofortress-tui --verbose

# Fallback to Go TUI (legacy mode)
gofortress-tui --legacy
```

## Session File Format

**Location:** `~/.claude/sessions/<session-id>.json`

**Format (Go-compatible):**
```json
{
  "id": "abc123def456",
  "name": "optional-session-name",
  "created_at": "2026-02-01T10:00:00Z",
  "last_used": "2026-02-01T11:30:00Z",
  "cost": 0.42,
  "tool_calls": 127
}
```

### Field Specifications

- **id** (required): Unique session identifier (UUID or nanoid)
- **name** (optional): Human-readable session name
- **created_at** (required): ISO 8601 timestamp with Z suffix
- **last_used** (required): ISO 8601 timestamp with Z suffix (auto-updated on save)
- **cost** (required): Total cost in USD (float)
- **tool_calls** (required): Total number of tool invocations (integer)

**Critical:** Field names use `snake_case` (not camelCase) for Go compatibility.

## Implementation Files

### Core Files

- **src/cli.ts** - CLI argument parsing with Commander
- **src/hooks/useSession.ts** - Session file operations (CRUD)
- **src/commands/list.tsx** - List sessions command UI
- **tests/integration/session.test.ts** - Integration tests

### Modified Files

- **src/index.tsx** - Entry point with CLI integration
- **src/App.tsx** - Session loading and auto-save

## Auto-Save Behavior

Sessions are automatically saved when:

- Cost changes (after tool execution)
- Token counts are updated

The `last_used` timestamp is automatically updated on each save.

## Session Resumption

When resuming a session with `--session <id>`:

1. Session file is loaded from disk
2. Session state (ID, cost) is restored to store
3. If loading fails, a new session is created
4. Loading screen is shown during resume

## Testing

```bash
# Run session integration tests
npm test -- session.test.ts

# Run all tests
npm test
```

### Test Coverage

- ✅ Session CRUD operations (create, read, update, delete)
- ✅ Session listing with sorting (by last_used descending)
- ✅ File format validation (snake_case, ISO 8601 dates)
- ✅ Error handling (invalid JSON, missing files, missing directory)
- ✅ Go compatibility (field names, date formats)

## Go TUI Compatibility

Sessions written by this TUI can be read by the Go TUI and vice versa. This ensures:

- **Rollback support**: Can switch back to Go TUI if needed
- **Data continuity**: Session history preserved across implementations
- **Format validation**: Tests verify Go compatibility

## Environment Variables

- **HOME**: Session directory location (`$HOME/.claude/sessions`)
  - Can be overridden for testing
- **VERBOSE**: Enable verbose logging when set to "1"

## Future Enhancements

Potential improvements (not implemented):

- Session naming/renaming command
- Session deletion command
- Session export/import
- Session search/filter
- Session statistics dashboard

## Architecture Notes

### Why snake_case?

Go's JSON marshaling uses struct tags to map between Go's PascalCase and JSON's snake_case. To maintain compatibility, we use snake_case in JSON files even though TypeScript typically uses camelCase.

### Why separate session directory?

Sessions are stored in `~/.claude/sessions/` (not with other TUI state) to:

- Enable sharing with Go TUI
- Simplify backup/migration
- Keep session files human-readable
- Allow manual inspection/editing

### Auto-save strategy

Sessions save on every cost change rather than on exit because:

- Prevents data loss on crash
- Enables real-time session monitoring
- Matches Go TUI behavior
- Simplifies implementation (no exit hooks needed)
