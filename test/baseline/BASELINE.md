# Performance Baseline (Bash Hooks)

**Date:** 2026-01-15
**System:** Linux doktersmol-framework 6.18.4-2-cachyos #1 SMP PREEMPT_DYNAMIC x86-64 GNU/Linux
**Memory:** 15Gi
**CPU:** 12th Gen Intel(R) Core(TM) i7-1255U

## Latency Measurements (100 events each)

| Hook | Average Latency | p99 Estimate | Notes |
|------|----------------|--------------|-------|
| validate-routing.sh | 43ms | ~129ms | Most complex hook (401 lines) |
| session-archive.sh | 19ms | ~57ms | File I/O heavy (111 lines) |
| sharp-edge-detector.sh | 36ms | ~108ms | Pattern matching (105 lines) |

## Event Corpus Strategy

**Corpus Logger**: Go implementation at `/home/doktersmol/Documents/l-a-g-GO/corpus-logger/`

**Approach**: Real production event capture
- Hook installed at `~/.claude/hooks/zzz-corpus-logger` (Go binary)
- Captures to `/run/user/1000/gogent/event-corpus-raw.jsonl` (XDG_RUNTIME_DIR)
- Pass-through design (doesn't interfere with Claude Code)
- Schema validation: Events parsed into Go structs before capture

**Why Go for corpus capture**:
1. Validates our event schema before GOgent-001 implementation
2. Same parsing logic that will process events in production
3. Proves Go can handle real-time event capture (<1ms overhead)
4. Output is both test fixture AND proof of correctness

**Event Schema** (validated by Go structs):
```go
type HookEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
    ToolResponse  map[string]interface{} `json:"tool_response,omitempty"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`
}
```

## Performance SLA

**Target:** Go hooks ≤ Bash average latency
**Acceptable:** +20% degradation (e.g., if Bash is 43ms, Go <52ms OK)
**Unacceptable:** >10ms p99 latency for any hook

## Corpus Locations

- **Live Capture:** `/run/user/1000/gogent/event-corpus-raw.jsonl` (growing with usage)
- **Curated Corpus:** Will be extracted after sufficient events captured
- **Test Fixtures:** `/home/doktersmol/Documents/l-a-g-GO/test/fixtures/event-corpus.json`
- **Go Logger Source:** `/home/doktersmol/Documents/l-a-g-GO/corpus-logger/`

## Corpus Collection Status

**Hook Installed:** ✅ `~/.claude/hooks/zzz-corpus-logger`
**Capturing Events:** ✅ Real production events
**Target:** 100+ diverse events covering:
- Task events (haiku, sonnet, opus)
- Read, Write, Edit events
- Bash events with successes and failures (for sharp-edge testing)
- Glob, Grep events
- Override flags (--force-tier, --force-delegation)
- Invalid subagent_type cases
- Opus blocking triggers

## Go Logger Implementation

**Features**:
- STDIN reading with 5-second timeout (M-6 fix)
- XDG-compliant path resolution (M-2 fix)
- Zero external dependencies (stdlib only)
- Pass-through design (echoes unchanged to STDOUT)
- Graceful error handling (never breaks hook chain)
- Test coverage: 50% (table-driven tests for edge cases)

**Build**: `cd corpus-logger && go build -o corpus-logger main.go`
**Install**: `./install.sh`
**Test**: `go test -v ./...`

## Next Steps

1. Use Claude Code normally - events will capture automatically
2. Once 100+ events captured, curate with jq:
   ```bash
   # Extract diverse sample
   cat /run/user/1000/gogent/event-corpus-raw.jsonl | \
     jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, events: .}) | ...' \
     > ~/gogent-baseline/event-corpus.json
   ```
3. Copy curated corpus to project test fixtures
4. Proceed to Week 1 (GOgent-001)

---

**Generated:** Thu 15 Jan 2026 19:14:19 AEDT
**Updated:** Thu 15 Jan 2026 19:37:00 AEDT
**Baseline Version:** 1.1 (Go corpus logger)
