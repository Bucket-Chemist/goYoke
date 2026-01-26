# GOgent-111: Decision Outcome Recording Verification

---
ticket_id: GOgent-111
title: Verify Decision Outcome Recording in SubagentStop Handler
status: completed
priority: medium
estimated_hours: 1
actual_hours: 0.5
depends_on: [GOgent-110]
blocks: [GOgent-112]
needs_planning: false
completed_date: 2026-01-26
resolution: documented_limitation
---

## Summary

Verify that `UpdateDecisionOutcome()` is called when agents complete, linking routing decisions to their execution outcomes. This enables the append-only ML training data pattern.

## Background

The ML telemetry system uses a **dual-file append-only pattern** for thread safety:

1. **Initial record** (routing-decisions.jsonl): Logged at PreToolUse by gogent-validate
2. **Outcome update** (routing-decision-updates.jsonl): Should be logged at SubagentStop by gogent-agent-endstate

The join happens at read-time in `gogent-ml-export` via `DecisionID` field.

**Question to verify:** Is `UpdateDecisionOutcome()` actually being called in the SubagentStop flow?

## Acceptance Criteria

- [x] Confirm `gogent-agent-endstate` calls `telemetry.UpdateDecisionOutcome()` or equivalent - **NOT CALLED**
- [x] If NOT called: implement the missing wiring - **NOT FEASIBLE (see Investigation Results)**
- [x] Verify `routing-decision-updates.jsonl` is populated after agent completion - **NOT POPULATED**
- [x] Confirm `DecisionID` correlation works (updates match decisions) - **CANNOT WORK (missing propagation)**
- [x] Integration test: spawn agent, verify outcome recorded - **DOCUMENTED AS LIMITATION**

## Investigation Steps

### Step 1: Check Current Implementation

```bash
# Search for UpdateDecisionOutcome usage
grep -r "UpdateDecisionOutcome" cmd/gogent-agent-endstate/

# Search for any outcome update logic
grep -r "outcome" cmd/gogent-agent-endstate/main.go
```

### Step 2: Trace the Decision ID Flow

The flow should be:
1. `gogent-validate` creates `RoutingDecision` with `DecisionID`
2. `DecisionID` passed through to agent execution (how?)
3. `gogent-agent-endstate` receives `DecisionID` in event
4. `UpdateDecisionOutcome(decisionID, success, duration, cost, escalated)` called

**Key question:** How does the DecisionID propagate from PreToolUse to SubagentStop?

### Step 3: If Wiring Missing

If `UpdateDecisionOutcome` is not being called, implement:

```go
// In cmd/gogent-agent-endstate/main.go
func processEvent(event *routing.SubagentStopEvent) (*workflow.EndstateResponse, error) {
    // ... existing code ...

    // NEW: Update routing decision outcome
    if event.DecisionID != "" {
        success := metadata.IsSuccess()
        duration := int64(metadata.DurationMs)
        cost := calculateCost(metadata) // Need to implement
        escalated := false // Derive from metadata

        if err := telemetry.UpdateDecisionOutcome(
            event.DecisionID, success, duration, cost, escalated,
        ); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Failed to update decision outcome: %v\n", err)
        }
    }

    // ... rest of function ...
}
```

## Technical Notes

### Append-Only Pattern (from routing_decision.go)

```go
type DecisionOutcomeUpdate struct {
    DecisionID         string  `json:"decision_id"`
    OutcomeSuccess     bool    `json:"outcome_success"`
    OutcomeDurationMs  int64   `json:"outcome_duration_ms"`
    OutcomeCost        float64 `json:"outcome_cost"`
    EscalationRequired bool    `json:"escalation_required"`
    UpdateTimestamp    int64   `json:"update_timestamp"`
}
```

### Read-Time Reconciliation

ML export joins on `DecisionID`:
```
routing-decisions.jsonl + routing-decision-updates.jsonl → training_data.csv
```

## Success Metrics

- `routing-decision-updates.jsonl` contains entries after agent completions
- Updates can be joined to decisions by `DecisionID`
- No orphaned updates (every update has matching decision)
- Integration test `TestMLTelemetry_DecisionUpdates` passes

## Files to Check/Modify

| File | Action |
|------|--------|
| `cmd/gogent-agent-endstate/main.go` | Check/add UpdateDecisionOutcome call |
| `pkg/routing/subagent_stop_event.go` | Check if DecisionID field exists |
| `pkg/telemetry/routing_decision.go` | Reference for UpdateDecisionOutcome signature |

## Decision

**If DecisionID propagation is not feasible:** Document as known limitation. The append-only pattern still works for collaboration logging; routing decisions just won't have outcome data. This is acceptable for Phase 2.

## Investigation Results

**Date:** 2026-01-26
**Finding:** Decision outcome recording is **not implemented** and **cannot be implemented** without Claude Code infrastructure changes.

### Root Cause Analysis

The dual-file append-only pattern requires:
1. **Initial decision** - ✅ Logged by `gogent-validate` at PreToolUse with unique `DecisionID`
2. **Outcome update** - ❌ **NOT logged** - requires `DecisionID` propagation

**The missing link:** `DecisionID` propagation from PreToolUse to SubagentStop

### Evidence

1. **`cmd/gogent-agent-endstate/main.go`** - No calls to `UpdateDecisionOutcome()`
   ```go
   // Only logs collaboration data, NOT outcome updates
   func processEvent(event *routing.SubagentStopEvent) {
       // ... logs collaboration ...
       telemetry.LogCollaboration(collab) // ✅
       // telemetry.UpdateDecisionOutcome() // ❌ MISSING - cannot implement
   }
   ```

2. **`pkg/routing/events.go:254`** - `SubagentStopEvent` has no `DecisionID` field
   ```go
   type SubagentStopEvent struct {
       HookEventName  string `json:"hook_event_name"`
       SessionID      string `json:"session_id"`
       TranscriptPath string `json:"transcript_path"`
       // DecisionID string `json:"decision_id"` // ❌ DOES NOT EXIST
   }
   ```

3. **Test `TestMLTelemetry_DecisionUpdates`** - Passes but doesn't verify functionality
   - Test only checks file is append-only (non-shrinking)
   - Passes even when file is empty/missing (lineCount: 0 >= 0 ✅)
   - Does not verify that updates are actually written

### Why Implementation Is Not Feasible

To implement decision outcome recording, we need:

**Option A: Event schema extension (requires Claude Code changes)**
- Add `decision_id` field to `SubagentStopEvent` in Claude Code
- Have Claude Code propagate `DecisionID` from PreToolUse response to SubagentStop event
- **Blocker:** We cannot modify Claude Code's hook event schemas

**Option B: Correlation by SessionID + Timestamp (fragile)**
- Match decisions to outcomes by `session_id` and approximate timing
- **Problems:**
  - Multiple decisions per session → ambiguous matches
  - Timing-based matching is unreliable
  - Violates append-only design (requires read-modify-write)

**Option C: Store DecisionID in transcript metadata (circular dependency)**
- Write `DecisionID` to transcript file during PreToolUse
- Read it back in SubagentStop
- **Problems:**
  - PreToolUse hook doesn't have write access to transcript
  - Transcript doesn't exist yet at PreToolUse time
  - Would require additional file I/O and state management

### Workaround: Collaboration Logging Still Works

While decision outcome recording is blocked, **collaboration logging is functional**:
- ✅ `gogent-agent-endstate` logs `AgentCollaboration` records
- ✅ Includes outcome data: `child_success`, `child_duration_ms`
- ✅ Can be used for agent delegation pattern analysis

The difference:
- **Collaboration:** Captures "which agent executed" (available in transcript)
- **Decision outcome:** Captures "was this routing decision correct" (requires DecisionID)

## Recommendation

**Document as known limitation for Phase 2.** Decision outcome recording is valuable for supervised learning (tier selection optimization) but not blocking for:
- Phase 2 (observability): Collaboration patterns provide sufficient data
- Phase 3 (optimization): Can revisit if Claude Code adds DecisionID propagation

**Alternative approach:** Use collaboration success rates as proxy for routing quality.

## Notes

The primary value of outcome recording is enabling supervised ML:
- **With outcomes:** Can train tier selection model (features → tier → success)
- **Without outcomes:** Limited to clustering and pattern analysis

Given this is Phase 2 (observability), outcome recording is valuable but not blocking.

**Status:** Documented limitation. No implementation changes required.
