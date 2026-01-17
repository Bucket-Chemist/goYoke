# Event Corpus (GOgent-008b)

This directory will contain the curated event corpus for validation testing.

## Status

**CAPTURE IN PROGRESS** (Started: 2026-01-16)

The corpus logger is actively capturing events. Check progress with:
```bash
./scripts/check-corpus-progress.sh
```

## Expected Files

- `event-corpus.json` - Curated corpus with ≥95 real ToolEvent samples (created after capture completes)

## Corpus Format

The corpus will be a JSON array of ToolEvent objects with the following structure:

```json
[
  {
    "tool_name": "Read",
    "tool_input": {
      "file_path": "/path/to/file.go"
    },
    "tool_response": {
      "content": "...",
      "success": true
    },
    "session_id": "session-abc123",
    "hook_event_name": "PreToolUse" | "PostToolUse",
    "captured_at": 1768544392
  }
]
```

### Required Fields (from GOgent-003/006)

- `tool_name` (string) - Name of the tool invoked
- `tool_input` (object or null) - Tool parameters
- `tool_response` (object, PostToolUse only) - Tool execution result
- `session_id` (string) - Session identifier
- `hook_event_name` (string) - Event type: "PreToolUse" or "PostToolUse"
- `captured_at` (number) - Unix timestamp

## Curation Process

Once ≥95 events are captured, run:
```bash
./scripts/curate-corpus.sh
```

This will:
1. Filter raw events (remove invalid/null entries)
2. Convert JSONL to JSON array
3. Validate JSON structure
4. Output to `test/fixtures/event-corpus.json`
5. Show event distribution statistics

## Usage

After curation, the corpus will be used for:

- **GOgent-006**: XDG path resolution validation
- **GOgent-007**: Tool permission check validation
- **GOgent-008**: Hook response format validation
- **GOgent-009**: Error message format validation
- **GOgent-041**: Test harness for corpus replay
- **GOgent-047**: Regression testing (Go vs Bash)

The corpus provides real-world event data to ensure our Go implementation matches actual Claude Code behavior.
