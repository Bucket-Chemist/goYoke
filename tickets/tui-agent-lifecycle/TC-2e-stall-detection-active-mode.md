---
id: TC-2e
title: "Enable active stall detection after empirical validation"
description: "Graduate stall detection from shadow mode (observe-only) to active mode (SIGTERM stalled agents). Requires empirical validation from Phase 2d shadow mode data."
priority: MEDIUM
dependencies: ["2a", "2b", "2c", "2d"]
status: pending
blocked_by: "Phase 2d shadow mode must run across ~20 real sessions first"
---

# TC-2e: Active Stall Detection Mode

**Priority:** MEDIUM
**Dependencies:** 2a (progressTracker), 2b (health fields), 2c (health monitor), 2d (shadow mode)
**Blocked by:** Empirical gate — shadow mode must validate <5% false positive rate

## Context

Phase 2d ships stall detection in **shadow mode**: the health monitor goroutine observes agent activity via `progressTracker.LastActivity()`, updates `health_status` / `stall_count` in config.json, and logs `[SHADOW]` warnings to runner.log. It never kills processes.

This ticket enables **active mode**: when stall detection confidence is validated, allow the health monitor to send SIGTERM to genuinely stalled agents.

## Empirical Gate (MUST complete before implementation)

### Data Collection

Run ~20 real team-run sessions with shadow mode enabled (Phase 2d). Collect:

1. **Total stall_warning events** (from runner.log `[SHADOW]` entries)
2. **True positives**: agent was genuinely stalled (never produced output, process hung)
3. **False positives**: agent was alive and working (produced output after warning)
4. **Breakdown by model tier**: haiku vs sonnet vs opus false positive rates
5. **Breakdown by workflow**: braintrust vs implementation vs review

### Analysis Script

Create `scripts/analyze-stall-shadows.sh`:

```bash
#!/bin/bash
# Analyze shadow mode stall detection data across sessions
# Usage: analyze-stall-shadows.sh [sessions-dir]

SESSIONS_DIR="${1:-.claude/sessions}"

echo "=== Stall Detection Shadow Analysis ==="
echo ""

# Count total shadow warnings
total_warnings=$(grep -r '\[SHADOW\]' "$SESSIONS_DIR"/*/teams/*/runner.log 2>/dev/null | wc -l)
echo "Total shadow warnings: $total_warnings"

# Count unique agents that received warnings
warned_agents=$(grep -r '\[SHADOW\]' "$SESSIONS_DIR"/*/teams/*/runner.log 2>/dev/null \
    | grep -oP 'member \K\S+' | sort -u | wc -l)
echo "Unique agents warned: $warned_agents"

# Count agents that completed successfully AFTER receiving a warning
# (false positives - agent was working, not stalled)
for log in "$SESSIONS_DIR"/*/teams/*/runner.log; do
    team_dir=$(dirname "$log")
    config="$team_dir/config.json"
    [ -f "$config" ] || continue

    # Find members that had shadow warnings
    warned=$(grep '\[SHADOW\]' "$log" | grep -oP 'member \K\S+' | sort -u)
    for member in $warned; do
        # Check if member completed successfully
        status=$(python3 -c "
import json, sys
c = json.load(open('$config'))
for w in c['waves']:
    for m in w['members']:
        if m['name'] == '$member':
            print(m['status'])
            sys.exit()
" 2>/dev/null)
        if [ "$status" = "completed" ]; then
            echo "FALSE POSITIVE: $member in $(basename $team_dir) — warned but completed"
        elif [ "$status" = "failed" ]; then
            echo "TRUE POSITIVE:  $member in $(basename $team_dir) — warned and failed"
        fi
    done
done

echo ""
echo "=== Decision ==="
echo "If false positive rate < 5%: safe to enable active mode"
echo "If false positive rate 5-15%: tune stallWarningThreshold"
echo "If false positive rate > 15%: stdout streaming signal insufficient, needs /proc augmentation"
```

### Decision Criteria

| False Positive Rate | Action |
|---------------------|--------|
| < 5% | Proceed with active mode implementation |
| 5-15% | Increase `stallWarningThreshold` from 90s to 120-180s, re-collect data |
| > 15% | Do NOT enable active mode. Add /proc/{pid}/io as secondary signal (Phase 3), re-collect |

## Implementation (after gate passes)

### 1. Add active mode to health monitor

**File:** `cmd/gogent-team-run/spawn.go` (in `startHealthMonitor`)

Currently (shadow mode from 2d):
```go
if m.StallCount >= 3 {
    m.HealthStatus = "stalled"
}
log.Printf("[SHADOW] health: member %s stalled ...", m.Name)
```

Add active mode branch:
```go
if m.StallCount >= 3 {
    m.HealthStatus = "stalled"
    if stallMode == "active" {
        log.Printf("[ACTIVE] health: sending SIGTERM to stalled member %s (no output for %v)",
            m.Name, sinceActivity.Round(time.Second))
        // Signal the executeSpawn select{} to initiate graceful shutdown
        // Use a channel or context cancellation
        stallCancel()
    }
}
```

### 2. Wire stall cancellation into executeSpawn

**File:** `cmd/gogent-team-run/spawn.go` (in `executeSpawn`)

Add a stall cancellation channel to the select{} block:

```go
// Create stall detection context
stallCtx, stallCancel := context.WithCancel(ctx)
defer stallCancel()

// Pass stallCancel to health monitor
go startHealthMonitor(stallCtx, tr, waveIdx, memIdx, tracker, stallCancel)

select {
case <-ctx.Done():
    // Parent context cancelled
    syscall.Kill(-pid, syscall.SIGKILL)
    return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
case err := <-waitDone:
    // Process completed normally
    return &spawnResult{...}, nil
case <-time.After(cfg.timeout):
    // Hard timeout (safety net) — SIGTERM cascade from Phase 1a
    syscall.Kill(-pid, syscall.SIGTERM)
    select {
    case <-waitDone:
    case <-time.After(sigTermGracePeriod):
        syscall.Kill(-pid, syscall.SIGKILL)
    }
    return nil, fmt.Errorf("timeout after %v", cfg.timeout)
case <-stallCtx.Done():
    // Stall detected by health monitor — same SIGTERM cascade
    syscall.Kill(-pid, syscall.SIGTERM)
    select {
    case <-waitDone:
    case <-time.After(sigTermGracePeriod):
        syscall.Kill(-pid, syscall.SIGKILL)
    }
    return nil, fmt.Errorf("stall detected (no output for %v)", stallWarningThreshold*3)
}
```

### 3. Read mode from config

**File:** `cmd/gogent-team-run/spawn.go` (in `prepareSpawn` or `executeSpawn`)

```go
stallMode := "shadow" // default
tr.configMu.RLock()
if tr.config != nil && tr.config.StallDetectionMode != "" {
    stallMode = tr.config.StallDetectionMode
}
tr.configMu.RUnlock()
```

### 4. Update kill_reason for stall kills

```go
if strings.Contains(err.Error(), "stall detected") {
    tr.updateMember(waveIdx, memIdx, func(m *Member) {
        m.KillReason = "stall"
    })
}
```

### 5. Add stall_warning state to prevent wave cascade

**CRITICAL:** A stall-killed member must NOT trigger `skipRemainingWaves()` if the stall was a false positive risk. Add a `stall_warning` state that is non-cascading:

In `wave.go` `checkWaveFailures()`:
```go
// Only cascade on hard failures, not stall kills
if member.Status == "failed" && member.KillReason != "stall" {
    failed = append(failed, member.Name)
}
```

This prevents a false-positive stall kill from cascading and killing the entire team.

## Test Cases

| # | Test | Expected |
|---|------|----------|
| 1 | Shadow mode: stall_count >= 3, mode="shadow" | Log only, no SIGTERM |
| 2 | Active mode: stall_count >= 3, mode="active" | SIGTERM sent, process killed |
| 3 | Active mode: process produces output before stall threshold | No kill, stall_count resets |
| 4 | Active mode: stall kill sets kill_reason="stall" | kill_reason field correct |
| 5 | Wave cascade: stall-killed member does NOT skip remaining waves | checkWaveFailures excludes stall kills |
| 6 | Hard timeout still fires if stall detection misses | timeout case in select still works |
| 7 | Config with stall_detection_mode="" defaults to shadow | Shadow behavior |
| 8 | Config with stall_detection_mode="active" enables kills | Active behavior |

## Files to Modify

| File | Change |
|------|--------|
| `cmd/gogent-team-run/spawn.go` | Add stallCtx/stallCancel to executeSpawn select, read mode from config |
| `cmd/gogent-team-run/spawn.go` | Modify startHealthMonitor to accept stallCancel and mode |
| `cmd/gogent-team-run/wave.go` | Exclude stall kills from wave cascade in checkWaveFailures |
| `cmd/gogent-team-run/spawn.go` | Set kill_reason="stall" for stall-triggered kills |
| `cmd/gogent-team-run/validate_test.go` | Add tests for active vs shadow mode |
| `scripts/analyze-stall-shadows.sh` | NEW — analysis script for shadow data |

## Success Criteria

- [ ] Empirical gate passed: shadow mode data shows <5% false positive rate
- [ ] Active mode sends SIGTERM → grace → SIGKILL (same cascade as timeout)
- [ ] Stall kills set kill_reason="stall" in config.json
- [ ] Stall-killed members do NOT cascade wave failures
- [ ] Shadow mode still works (default behavior unchanged)
- [ ] Hard timeout still fires as safety net
- [ ] All existing tests pass
- [ ] 8 new test cases pass

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| False positive kills active agent | Medium | High ($1-5 wasted) | Non-cascading stall_warning state; shadow mode gate |
| Stall threshold too aggressive for Opus | Medium | Medium | Per-model-tier thresholds (Phase 3) |
| stallCancel race with normal completion | Low | Low | Defer stallCancel() ensures cleanup |
| Health monitor goroutine leak | Low | Low | Context cancellation on process exit |
