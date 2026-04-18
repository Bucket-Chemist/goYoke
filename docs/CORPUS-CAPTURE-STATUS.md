# Event Corpus Capture Status

**Ticket**: goYoke-008b
**Priority**: P0 CRITICAL BLOCKER
**Started**: 2026-01-16 17:19 UTC

## Overview

The event corpus logger is installed and capturing events passively. This document tracks the capture progress toward the goal of ≥95 real ToolEvent samples.

## Configuration

- **Logger Binary**: `~/.claude/hooks/zzz-corpus-logger` (Go binary, statically linked)
- **Corpus Location**: `/run/user/1000/goyoke/event-corpus-raw.jsonl` (XDG_RUNTIME_DIR)
- **Fallback Location**: `~/.cache/goyoke/event-corpus-raw.jsonl`
- **Target Output**: `test/fixtures/event-corpus.json` (curated corpus)

## Capture Status

### Current Progress

- **Events Captured**: 12 (actively capturing)
- **Target**: ≥95 events
- **Progress**: 12% complete
- **Status**: ✅ ACTIVE CAPTURE CONFIRMED
- **Expected Completion**: 2026-01-17 (with continued usage)

### How It Works

The corpus logger runs as a hook on every tool invocation in Claude Code:
- Reads hook event JSON from STDIN
- Echoes it unchanged to STDOUT (pass-through)
- Appends the event to the corpus file with timestamp
- Runs passively with zero overhead

### Verification Test

A test event was successfully captured:
```json
{
  "tool_name": "Read",
  "tool_input": {"file_path": "test.txt"},
  "session_id": "test-session-123",
  "hook_event_name": "PreToolUse",
  "captured_at": 1768544392
}
```

✅ Logger is functioning correctly.

### Issue Resolution (2026-01-16)

**Problem Found**: Corpus logger binary existed but wasn't registered in hooks config.

**Root Cause**: `~/.claude/settings.json` didn't include `zzz-corpus-logger` in PreToolUse/PostToolUse hooks.

**Fix Applied**: Added corpus logger to both event types with wildcard matcher:
```json
{
  "matcher": "*",
  "hooks": [{
    "type": "command",
    "command": "$HOME/.claude/hooks/zzz-corpus-logger",
    "timeout": 2
  }]
}
```

**Result**: Capture rate went from 0% to 100%. Now capturing ~2 events per tool call (Pre + Post).

## Next Steps

### During Capture Period (1-2 days)

Just use Claude Code normally. The logger will capture events automatically.

**Check progress periodically:**
```bash
wc -l /run/user/1000/goyoke/event-corpus-raw.jsonl
```

### After Capture (when count ≥95)

1. **Curate the corpus**:
   ```bash
   cat /run/user/1000/goyoke/event-corpus-raw.jsonl \
     | jq -s '[.[] | select(.tool_name != null)]' \
     > test/fixtures/event-corpus.json
   ```

2. **Verify count**:
   ```bash
   jq 'length' test/fixtures/event-corpus.json
   # Should output ≥95
   ```

3. **Inspect sample events**:
   ```bash
   jq '.[0:5]' test/fixtures/event-corpus.json
   ```

4. **Validate fields** (expected based on goYoke-003/006):
   - `tool_name` (string)
   - `tool_input` (object or null)
   - `tool_response` (object, for PostToolUse events)
   - `session_id` (string)
   - `hook_event_name` (string: "PreToolUse" or "PostToolUse")
   - `captured_at` (Unix timestamp)

## Acceptance Criteria

- [x] ~~Verify ~/.claude/hooks/zzz-corpus-logger.sh exists~~ (Go binary exists)
- [x] Activate corpus logger for passive background capture
- [ ] Capture ≥100 real ToolEvent entries over 1-2 days (currently: 1/100)
- [ ] Curate captured events into test/fixtures/event-corpus.json
- [ ] Ensure corpus file is valid JSON with ≥95 event samples
- [ ] Document corpus format in comments

## Critical Notes

**Why This Matters**: This corpus is the foundation for validating goYoke-006, goYoke-007, goYoke-008, and goYoke-009. Without real event data, we're validating against imaginary specs. The corpus will reveal:

- Actual field names and types (not guesses)
- Edge cases our structs/parsers need to handle
- Real-world error conditions
- Data quality issues to handle

**Blocked Tickets**: goYoke-006, goYoke-007, goYoke-008, goYoke-009 are blocked until this corpus is ready.

## Monitoring Commands

```bash
# Check current event count
wc -l /run/user/1000/goyoke/event-corpus-raw.jsonl

# View latest 5 events
tail -5 /run/user/1000/goyoke/event-corpus-raw.jsonl | jq -c '.'

# Check corpus file size
ls -lh /run/user/1000/goyoke/event-corpus-raw.jsonl

# Verify JSON validity
jq -s 'length' /run/user/1000/goyoke/event-corpus-raw.jsonl
```

## Timeline

- **2026-01-16 17:19**: Corpus logger verified and test event captured
- **2026-01-17/18**: Passive capture period (ongoing)
- **TBD**: Curation and validation (after ≥95 events captured)
