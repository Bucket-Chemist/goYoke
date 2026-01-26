# GOfortress TUI - Handover Document

**Date:** 2026-01-26
**Status:** Blocked - evaluating alternative approaches

---

## Executive Summary

GOfortress attempts to wrap Claude Code CLI in a Bubbletea TUI. After extensive debugging, we identified multiple issues but the TUI remains non-functional. The root causes span Claude Code CLI bugs, incorrect message formats, and Go channel lifecycle issues.

**Recommendation:** Consider using an existing Go wrapper (humanlayer/claudecode-go) or the official Claude Agent SDK rather than raw CLI wrapping.

---

## What We Tried

### Fixes Applied

| Fix | File | Status |
|-----|------|--------|
| Added `--verbose` flag | subprocess.go | ✅ Applied |
| Channel lifecycle (sync.Once, no defer close) | subprocess.go | ✅ Applied |
| Fresh channels on restart | subprocess.go | ✅ Applied |
| Event pump (waitForEvent) | panel.go | ✅ Applied |
| processStoppedMsg handling | panel.go | ✅ Applied |
| Message format v1: `{"content":"..."}` | events.go | ❌ Wrong |
| Message format v2: `{"type":"user","message":{"role":"user","content":"..."}}` | events.go | ❌ Still wrong |

### What Still Doesn't Work

- Panics: `send on closed channel`, `close of closed channel`
- TUI unresponsive
- Banner not rendering
- Claude process communication fails

---

## Root Causes Identified (From Research)

### 1. Message Format Wrong (P0)

**Current (WRONG):**
```json
{"type":"user","message":{"role":"user","content":"string here"}}
```

**Correct:**
```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"string here"}]}}
```

The `content` field must be an **array of ContentBlock objects**, not a string.

### 2. Claude Code Bug #1920 (P1)

The final `{"type":"result",...}` event sometimes **never arrives** after tool execution. This causes the event pump to block indefinitely. Requires timeout handling.

### 3. Missing `--debug-to-stderr` Flag (P1)

Debug output can corrupt the JSON stream on stdout. Add `--debug-to-stderr` to keep stdout clean.

### 4. exec.Command Wait() Called Too Early (P1)

Go docs: "it is incorrect to call Wait before all reads from the pipe have completed."

Current code calls `cmd.Wait()` in `monitorRestart()` without waiting for `readEvents()` and `readStderr()` goroutines to finish reading. Needs WaitGroup coordination.

### 5. No Generation Tracking for Restarts (P2)

When process restarts, old goroutines may still hold references and write to channels. Need versioned/generation-tagged channels.

---

## Correct Claude Code CLI Invocation

```bash
claude --print \
  --verbose \                     # REQUIRED with stream-json
  --debug-to-stderr \             # Keep stdout clean
  --input-format stream-json \
  --output-format stream-json \
  --include-partial-messages \    # Optional: token-level streaming
  --session-id <uuid>
```

**Input message format:**
```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Your message"}]}}
```

**Output event types:**
- `system` (subtype: init, hook_started, hook_response)
- `assistant` (Claude's responses)
- `result` (completion - may not arrive due to bug #1920)

---

## Alternative Approaches to Evaluate

### 1. Existing Go Wrappers

| Library | Notes |
|---------|-------|
| humanlayer/claudecode-go | Production-quality, documented bug workarounds |
| lancekrogers/claude-code-go | Full streaming, MCP integration |
| severity1/claude-agent-sdk-go | Clean functional options API |
| f-pisani/claude-code-sdk-go | Channel-based message/error handling |

### 2. Claude Agent SDK (Python/TypeScript)

Official SDK with proper type safety. Could wrap via subprocess from Go or use as separate backend.

### 3. MCP Protocol

Claude Code can act as MCP server. Build TUI as MCP client for standardized interface with better error handling.

### 4. Toad Pattern

Python/Textual TUI that successfully wraps Claude Code using ACP (Agent Client Protocol).

---

## Files Changed

```
internal/cli/subprocess.go    - --verbose, channel lifecycle, Send()
internal/cli/events.go        - UserMessage struct (still wrong)
internal/cli/events_test.go   - Updated for new struct
internal/cli/subprocess_test.go - Updated for new struct
internal/tui/claude/panel.go  - Event pump, mouse, sidebar width
internal/tui/claude/input.go  - Early returns
internal/tui/claude/events.go - Sidebar truncation
internal/tui/layout/banner.go - Width guards
internal/tui/layout/banner_test.go - Test fix
```

---

## Unfixed Items (If Continuing This Approach)

1. **Fix UserContent.Content to be `[]ContentBlock`** not string
2. **Add `--debug-to-stderr`** to CLI args
3. **Add WaitGroup** to coordinate pipe reads before Wait()
4. **Add timeout** for missing result events (Bug #1920)
5. **Add generation tracking** for restart isolation
6. **Consider `--no-hooks`** flag for testing without GOgent hooks

---

## Test Commands

```bash
# Test correct message format directly
echo '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}' | \
  claude --print --verbose --debug-to-stderr \
         --input-format stream-json --output-format stream-json \
         --session-id $(uuidgen) 2>/dev/null

# Test without hooks
claude --print --verbose --no-hooks "say hello"

# Check version
claude --version
```

---

## Session Context

- Branch: master
- 51+ uncommitted files (includes these fixes)
- Build passes: `go build ./cmd/gofortress/`
- Tests pass: `go test ./...`
- Runtime: Still panics

---

## Decision Point

The raw CLI wrapping approach has significant complexity due to:
- Undocumented/changing message formats
- Known Claude Code bugs (#1920, #3187, #771)
- Subprocess lifecycle edge cases

**Alternatives may provide faster path to working TUI.**
