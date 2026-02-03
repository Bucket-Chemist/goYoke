# Gemini-Slave Debug Summary

**Session Date:** 2026-02-03
**Status:** STDIN CAPTURE INCONSISTENT - needs investigation

---

## What Was Done This Session

### Protocol Updates (COMPLETED)
- Created 4 new protocols: `memory-drift`, `benchmark-score`, `deps`, `api-surface`
- Updated existing protocols: `mapper`, `debugger`, `architect`, `memory-audit`, `benchmark-audit`
- Removed deprecated `scout` protocol
- Updated `~/.claude/agents/gemini-slave/gemini-slave.md` frontmatter
- Updated `~/.local/bin/gemini-slave` wrapper routing

### Test Fixes (COMPLETED)
- Fixed `cmd/gogent-scout/main_test.go` - removed obsolete Gemini delegation tests
- Fixed `pkg/routing/schema_test.go` - updated to test mapper instead of removed scout
- Rebuilt all binaries

### Settings Fix (COMPLETED)
- Changed `~/.gemini-slave/.gemini/settings.json`: `disableYoloMode: false`

---

## THE PROBLEM

The gemini-slave wrapper has **inconsistent stdin capture**. Sometimes it works, sometimes it returns "INPUT CONTEXT is empty".

### Observed Behavior
1. Small inputs (< 100 bytes) work reliably
2. Large inputs (26KB+) fail intermittently
3. Adding debug `echo` statements to stderr seemed to make it work temporarily
4. Removing debug statements broke it again

### Current Wrapper State
File: `~/.local/bin/gemini-slave`

The stdin capture section has been modified multiple times. Current approach uses temp file:
```bash
STDIN_FILE=$(mktemp)
if [ -p /dev/stdin ]; then
    cat > "$STDIN_FILE"
elif [ ! -t 0 ]; then
    cat > "$STDIN_FILE"
fi
```

This approach also fails.

---

## WORKING TEST COMMAND

```bash
# Small input - works
echo "package main\nfunc main() {}" | gemini-slave mapper "Map entry points"

# Large input - FAILS intermittently
find cmd/gogent-scout -name "*.go" -exec cat {} \; | gemini-slave mapper "Map entry points"
```

---

## SUSPECTED ROOT CAUSE

Bash stdin buffering or timing issue when:
1. Input is piped from `find -exec cat`
2. Script does other operations before reading stdin
3. Possibly related to `set -euo pipefail`

---

## NEXT STEPS TO TRY

1. **Move stdin capture to VERY FIRST line** after shebang, before any argument parsing
2. **Try different stdin detection**: `[ ! -t 0 ]` vs `[ -p /dev/stdin ]`
3. **Try `read` loop** instead of `cat`
4. **Check if gemini CLI itself** has issues with large stdin when using `-p` flag
5. **Consider removing `-p` flag** and putting everything in stdin

---

## FILES MODIFIED THIS SESSION

- `~/.gemini-slave/protocols/mapper.md` - removed architectural_note
- `~/.gemini-slave/protocols/debugger.md` - added JSON output, constraints
- `~/.gemini-slave/protocols/architect.md` - added JSON output
- `~/.gemini-slave/protocols/memory-audit.md` - added reference to memory-drift
- `~/.gemini-slave/protocols/benchmark-audit.md` - removed Swarm Candidates
- `~/.gemini-slave/protocols/memory-drift.md` - NEW
- `~/.gemini-slave/protocols/benchmark-score.md` - NEW
- `~/.gemini-slave/protocols/deps.md` - NEW
- `~/.gemini-slave/protocols/api-surface.md` - NEW
- `~/.claude/agents/gemini-slave/gemini-slave.md` - synced frontmatter
- `~/.local/bin/gemini-slave` - multiple stdin capture attempts
- `~/.gemini-slave/.gemini/settings.json` - enabled yolo mode
- `cmd/gogent-scout/main_test.go` - removed obsolete tests
- `pkg/routing/schema_test.go` - fixed external tier test

---

## QUICK RESTORE

If you need to restore a known-working wrapper version, the key elements are:
1. `--yolo` flag must be present
2. `-e none` to disable extensions
3. `HOME` override to `$HOME/.gemini-slave`
4. Stdin must be captured and included in prompt as `### INPUT CONTEXT` section
