# Claude Agent SDK Concurrency Investigation

**Date**: 2026-02-07
**Ticket**: TC-019
**SDK Version**: 0.2.31 (`@anthropic-ai/claude-agent-sdk`)
**Node.js Version**: v25.4.0
**tsx Version**: 4.21.0

## Executive Summary

**The Claude Agent SDK v0.2.31 SUPPORTS concurrent `query()` calls from the same Node.js process.** All three empirical tests passed: concurrent queries complete independently, no cancellation occurs, and output streams are fully isolated.

This is a positive finding for TC-015 (TUI Concurrent Query Support). Phase 2 of TC-015 can proceed as designed with the per-query session management refactor.

## Test Results

### Test 1: Concurrent query() Calls

**Result**: PASS
**Behavior**: Both queries completed successfully
**Duration**: 8,433ms

**Setup**: Two `query()` calls started simultaneously:
- Query 1: "Respond with exactly: QUERY_ONE_COMPLETE" (model: haiku)
- Query 2: "Respond with exactly: QUERY_TWO_COMPLETE" (model: haiku)
- Both collected via `Promise.all()`

**Evidence**:
```json
{
  "test": "concurrent_queries",
  "query1_status": "completed",
  "query2_status": "completed",
  "query1_output": "QUERY_ONE_COMPLETE",
  "query2_output": "QUERY_TWO_COMPLETE",
  "behavior": "both_completed",
  "duration_ms": 8433.207505
}
```

**Finding**: The SDK spawns independent Claude CLI subprocesses for each `query()` call. Concurrent invocations do not interfere with each other.

### Test 2: Query Cancellation

**Result**: PASS
**Behavior**: Starting a second query does NOT cancel the first
**Duration**: 10,604ms

**Setup**:
- Query 1: "Count from 1 to 20, one number per line" (model: haiku)
- Wait 2 seconds
- Query 2: "Respond with exactly: QUERY_TWO_DONE" (model: haiku)
- Both collected via `Promise.all()`

**Evidence**:
```json
{
  "test": "query_cancellation",
  "query1_fate": "completed",
  "query2_fate": "completed",
  "query1_output": "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n16\n17\n18\n19\n20",
  "query2_output": "QUERY_TWO_DONE",
  "behavior": "both_completed",
  "duration_ms": 10603.5915
}
```

**Finding**: Query 1 completed its full output (1-20) despite Query 2 starting while it was running. No cancellation, no interruption, no queuing.

### Test 3: Stream Isolation

**Result**: PASS
**Behavior**: Output streams are completely isolated
**Duration**: 8,528ms

**Setup**:
- Query 1: "Repeat this exact string 5 times on separate lines: ALPHA" (model: haiku)
- Query 2: "Repeat this exact string 5 times on separate lines: BETA" (model: haiku)
- Both started simultaneously, collected via `Promise.all()`

**Evidence**:
```json
{
  "test": "stream_isolation",
  "query1_status": "completed",
  "query2_status": "completed",
  "query1_contaminated": false,
  "query2_contaminated": false,
  "query1_text": "ALPHA\nALPHA\nALPHA\nALPHA\nALPHA",
  "query2_text": "BETA\nBETA\nBETA\nBETA\nBETA",
  "behavior": "isolated",
  "duration_ms": 8528.401143
}
```

**Finding**: Query 1's output contained exclusively "ALPHA" text, Query 2's exclusively "BETA". No cross-contamination between streams.

### Test 4: Event Stream Ref Analysis (Code Review)

**Result**: Risk identified but NOT a blocker
**Risk Assessment**: MEDIUM (application-level, not SDK-level)

**Findings from code analysis of `useClaudeQuery.ts`**:

| Code Location | Pattern | Risk |
|---|---|---|
| Line 182: `eventStreamRef.current` | Single ref for active query | **Overwritten** if second query starts |
| Line 598-603: `streamingRef.current` guard | Blocks concurrent `sendMessage()` | **Prevents concurrency** at application layer |
| Line 721: `eventStreamRef.current` read | Used for `streamInput()` | Wrong query's stream if ref overwritten |
| Line 335-361: Built-in tool handling | Calls `eventStreamRef.current.streamInput()` | **Would target wrong query** if concurrent |

**Critical finding**: The SDK itself supports concurrency, but `useClaudeQuery.ts` uses a single `eventStreamRef` that assumes only one query at a time. TC-015's Phase 2 refactor (per-query state container with `ActiveQuery` map) correctly addresses this.

**Code path for the guard**:
```typescript
// Line 598-603 in useClaudeQuery.ts
if (streamingRef.current) {
  void logger.warn("Query already in progress, ignoring duplicate call");
  return;  // ← This is the application-level block, NOT an SDK limitation
}
```

## Conclusion

**SDK Supports Concurrent Queries**: YES

**Recommendation for TC-015**: Proceed with Phase 2 as designed. The refactor should:

1. Replace `streamingRef.current` boolean guard with per-query tracking (`ActiveQuery` map)
2. Replace `eventStreamRef.current` single ref with per-query event stream references
3. Update built-in tool handling to route `streamInput()` to the correct query's event stream
4. Update Zustand store from boolean `streaming` to `activeQueryCount: number`

## Architecture: How SDK Achieves Concurrency

Based on the test results and SDK type analysis, each `query()` call:

1. Spawns an independent Claude CLI subprocess (`claude` binary)
2. Communicates via stdio pipes (stdin/stdout/stderr)
3. Returns an AsyncGenerator that yields SDK messages from that specific subprocess
4. Has its own subprocess lifecycle (independent of other queries)

This is a process-level isolation model, not in-process concurrency. Each query gets its own Claude CLI instance.

```
Node.js Process (TUI)
│
├─► query() call 1 → Claude CLI subprocess 1 → API → Response stream 1
│
├─► query() call 2 → Claude CLI subprocess 2 → API → Response stream 2
│
└─► query() call N → Claude CLI subprocess N → API → Response stream N
```

## Implications for TUI Design

| Aspect | Current State | After TC-015 |
|---|---|---|
| Concurrent queries | Blocked by application guard | Supported (per-query state) |
| User input during streaming | Blocked | Allowed |
| Memory per query | N/A | ~1 CLI subprocess (~50-100MB) |
| API rate limits | Single query | Multiple concurrent → may hit rate limits |
| Cost tracking | Single query total | Sum across concurrent queries |

## Test Scripts

All test scripts are committed to the repository:

- `packages/tui/scripts/test-concurrent-queries.ts`
- `packages/tui/scripts/test-query-cancellation.ts`
- `packages/tui/scripts/test-stream-isolation.ts`

Run with: `npx tsx scripts/test-<name>.ts` from `packages/tui/`

## Open Questions Resolved

| Question | Answer |
|---|---|
| Does SDK support concurrent `query()` calls? | **YES** — each query spawns independent CLI subprocess |
| Does a second query cancel the first? | **NO** — both run to completion |
| Are output streams isolated? | **YES** — no cross-contamination |
| Is the freeze an SDK limitation? | **NO** — it's an application-level guard in `useClaudeQuery.ts` |
