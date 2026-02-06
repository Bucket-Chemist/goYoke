# Backend Review Findings

## Summary
- **Total issues found**: 14
- **CRITICAL**: 3 | **HIGH**: 6 | **MEDIUM**: 4 | **LOW**: 1

## Findings

### [CRITICAL] WaitGroup Misuse Still Present Despite Refactoring Instructions

**Tickets**: TC-003, TC-008, TC-011
**Category**: concurrency
**Description**: TC-003 specifies fixing the WaitGroup panic by replacing recursion with iteration. However, the proposed pattern in TC-008 still has a subtle issue: `spawnAndWait()` calls `wg.Done()` in a `defer` at the function start, but the refactored for-loop with `continue` statements might execute multiple times in a single function invocation. If `spawnAndWait()` is ever called from multiple goroutines within the same wave (parallel execution per TC-008 lines 169-182), and one goroutine's retry loop spans longer than expected, there's a window where the WaitGroup counter could temporarily go negative if `wg.Add()` is called after `wg.Done()` from a previous invocation. This is NOT the case in the current design because `wg.Add(1)` is called in the caller (`runWaves()` line 171) before spawning, but it's a footgun if someone refactors the wave scheduler.

**Recommendation**: Add a comment in `spawnAndWait()` documenting: "The caller (runWaves) must call wg.Add(1) exactly once before spawning this goroutine. This function will call wg.Done() exactly once, regardless of retry count." Consider extracting the retry loop to a separate internal function to prevent accidental re-entrance.

**Evidence**: TC-008 lines 169-182 (wave execution), TC-003 lines 37-84 (refactored pattern), TC-011 lines 324-408 (test assumes single Done per invocation)

---

### [CRITICAL] Budget Gate Race Condition Between Check and Deduction

**Tickets**: TC-002, TC-008
**Category**: concurrency
**Description**: TC-008's wave scheduler (lines 173-179) checks budget BEFORE acquiring the mutex, then releases the mutex before spawning. This creates a race:

```
Thread 1: Check budget (OK, $1.50 remaining), release lock
Thread 2: Check budget (OK, $1.50 remaining), release lock
Thread 1: Spawn agent (costs $1.25), deduct budget → $0.25
Thread 2: Spawn agent (costs $1.30), deduct budget → ERROR: -$1.05
```

The specification says "Budget gate blocks new spawns when budget exhausted" (TC-008 line 166), but the check is outside the critical section.

**Recommendation**: Move the budget check INSIDE the `tr.updateMember()` call, or better yet, check inside the function after acquiring the mutex. Alternatively, use `tryUpdateMember()` that returns false if budget insufficient, and break the wave loop on failure.

**Evidence**: TC-008 lines 173-179 (check outside lock), TC-008 lines 283-285 (deduction inside lock), TC-011 lines 414-486 (test assumes atomic check+deduct)

---

### [CRITICAL] PID File Double-Start Race Condition

**Tickets**: TC-004, TC-016
**Category**: daemon pattern, process management
**Description**: TC-004 and TC-016 both describe PID file locking, but there's a TOCTOU (Time-Of-Check-Time-Of-Use) race: two processes can simultaneously check, find no PID file, and both write. The specification checks process existence, then writes PID atomically, but between check and write another process can write first.

**Recommendation**: Use OS-level atomic operations: Open with `O_CREAT | O_EXCL` (atomic create-or-fail). If file exists, check liveness of existing PID. If dead, remove and retry with exclusive create.

**Evidence**: TC-004 lines 113-132 (check then write), TC-016 lines 27-52 (check then write), TC-016 lines 99-114 (success criteria don't account for simultaneous launches)

---

### [HIGH] Incomplete Cost Extraction Error Handling

**Tickets**: TC-005, TC-008
**Category**: process management, error handling
**Description**: TC-008 specifies that cost extraction failure logs a warning and continues with `cost = 0`. This means agent completes but cost is unmeasured, budget tracking becomes inaccurate, and multiple unmeasured agents could deplete budget faster than config.json reports. TC-005 specifies defensive parsing with fallback, but TC-008 doesn't distinguish between "cost extraction failed" (continue) and "no cost field in output" (continue). These are different errors.

**Recommendation**: Add error levels: ErrNoCostField (degrade gracefully), ErrMalformedJSON (mark agent as warning). Also add periodic budget sync check: after every 5 agents, re-read actual CLI output from runner.log and verify cost totals.

**Evidence**: TC-005 lines 103-124 (fallback parsing), TC-008 lines 276-280 (continue on cost error), TC-008 lines 282-285 (deduct cost without verification)

---

### [HIGH] Heartbeat Granularity Too Coarse for Hung Detection

**Tickets**: TC-004, TC-008, TC-012
**Category**: process management, monitoring
**Description**: TC-008 specifies heartbeat every 30 seconds. TC-012 specifies stale detection after 60 seconds. This means an agent that hangs 10 seconds after heartbeat touch shows as "healthy" for up to 40 seconds while actually hung.

**Recommendation**: Reduce heartbeat interval to 10 seconds, stale threshold to 30 seconds. Add per-agent timeout tracking. Document that heartbeat freshness indicates the main gogent-team-run process is alive, not that agents are making progress.

**Evidence**: TC-008 lines 419-434 (30s interval), TC-012 lines 84-91 (60s stale threshold)

---

### [HIGH] Missing Agent Timeout Enforcement

**Tickets**: TC-008, TC-009
**Category**: process management, error handling
**Description**: TC-009 specifies `timeout_ms` field in team config, but TC-008's `spawnAndWait()` doesn't enforce it. `cmd.Wait()` blocks indefinitely until process exits. If an agent hangs and the user doesn't send `/team-cancel`, the team waits forever.

**Recommendation**: Implement timeout per member using `exec.CommandContext()` with `context.WithTimeout()`. On deadline exceeded, kill the process and mark as failed with timeout message.

**Evidence**: TC-009 lines 96 (timeout_ms defined), TC-008 lines 239-265 (no timeout enforcement)

---

### [HIGH] Child Process Cleanup Not Guaranteed on Context Cancellation

**Tickets**: TC-003, TC-004, TC-008
**Category**: process management, daemon pattern
**Description**: TC-008's signal handler calls `killAllChildren()` with a 5-second grace period. However, the PID is registered AFTER `cmd.Start()` completes. If a process starts between context cancellation and killAllChildren() lock acquisition, it won't be killed.

**Recommendation**: Make process registration atomic with start: register PID immediately after `cmd.Start()`, before `cmd.Wait()`. Add a finalizer goroutine that kills remaining children after context cancellation.

**Evidence**: TC-008 lines 239-266 (register after start), TC-004 lines 204-228 (kill logic)

---

### [HIGH] Config.json Consistency Not Verified After Concurrent Updates

**Tickets**: TC-002, TC-008, TC-011
**Category**: concurrency, error handling
**Description**: TC-008 calls `writeConfigAtomic()` after every `updateMember()`. With 4 members updating concurrently, there's a potential Lost Update problem. The mutex protects individual member updates, but verify that TC-008 doesn't have direct config access bypassing updateMember(). TC-008 lines 283-285 show budget deduction directly, not via updateMember().

**Recommendation**: Verify ALL config modifications go through updateMember() or a similar locked method. Direct budget deduction at line 283-285 is a bypass risk.

**Evidence**: TC-002 lines 67-83 (updateMember locks), TC-008 lines 283-285 (direct budget deduction)

---

### [MEDIUM] CLI Argument Ordering Inconsistency

**Tickets**: TC-001, TC-008, TC-014
**Category**: security, process management
**Description**: TC-001 specifies flag order: `--permission-mode delegate` BEFORE `--allowedTools`. But TC-008 and TC-014 don't enforce this order. Verify with `claude --help` whether order matters.

**Recommendation**: Verify order matters, then standardize across all tickets.

**Evidence**: TC-001 lines 120-121, TC-008 lines 310-325, TC-014 lines 322-343

---

### [MEDIUM] Missing Explicit Error on Agent Not Found in agents-index.json

**Tickets**: TC-008, TC-014
**Category**: error handling
**Description**: TC-008 loads agent config and uses defaults if missing, but doesn't validate the agent exists. A typo like "einsteinn" gets default permissions instead of failing fast.

**Recommendation**: Add pre-flight validation: fail startup if referenced agent not found in agents-index.json.

**Evidence**: TC-008 lines 227-233 (warn + defaults), TC-014 lines 322-328

---

### [MEDIUM] Wave Completion Check Doesn't Account for Partial Failure

**Tickets**: TC-003, TC-008
**Category**: error handling
**Description**: TC-008 doesn't define failure semantics: does a failed member block wave progression? Inter-wave script expects all stdout files, but failed member never produces stdout.

**Recommendation**: Add explicit wave failure policy to TC-008. Document whether inter-wave script gracefully handles missing outputs.

**Evidence**: TC-008 lines 185-193, TC-010 lines 85-112

---

### [MEDIUM] Missing Env Var for Project Root Fallback

**Tickets**: TC-006, TC-013
**Category**: configuration, error handling
**Description**: TC-006 specifies project root resolution from env vars, but TC-013 doesn't document how these env vars are set. Falls back to `pwd` which might be wrong.

**Recommendation**: Document in TC-013 that TUI must set `GOGENT_PROJECT_DIR`. Add validation that pwd matches project root.

**Evidence**: TC-006 lines 107-117, TC-013 lines 161-177

---

### [MEDIUM] No Atomic Test for Config.json Crash Safety

**Tickets**: TC-002, TC-008, TC-011
**Category**: testing
**Description**: TC-011 tests concurrent updates but doesn't test crash safety: kill process mid-write, verify config.json remains valid JSON.

**Recommendation**: Add integration test: start team, kill with SIGKILL during update, verify config.json valid, restart successfully.

**Evidence**: TC-008 lines 107-125, TC-011 lines 553-599

---

### [LOW] Inconsistent Error Message Formatting

**Tickets**: TC-004, TC-012
**Category**: user experience
**Description**: Different error message formats between TC-004 and TC-012.

**Recommendation**: Establish error message convention with consistent prefix, severity, and actionable next steps.

**Evidence**: TC-004 lines 298-299, TC-012 lines 185-207
