# Critical Review: Background Team Orchestration via `gogent-team-run`

**Reviewed:** 2026-02-06
**Reviewer:** Staff Architect Critical Review
**Input:** `/home/doktersmol/Documents/GOgent-Fortress/tickets/team-coordination/IMPLEMENTATION-PLAN.md`
**Verdict:** APPROVE_WITH_CONDITIONS
**Confidence:** MEDIUM

---

## Executive Assessment

This is a well-structured plan for a real problem (TUI freezing during multi-agent workflows). The schema design is thorough, the failure mode catalog is honest, and the phased approach is sound. However, the plan contains several assumptions about Claude CLI behavior that are contradicted by evidence already in the codebase, a race condition in the core loop, and underspecification of exactly how the Go binary obtains tool permissions for spawned agents. These issues are fixable but must be addressed before implementation begins, or the Go binary will not work on first contact with reality.

**Issue Counts:**

- Critical: 2 (must fix)
- High: 4 (should fix)
- Medium: 5 (consider fixing)
- Low: 3 (nice to have)

**Commendations:** 5

**Go/No-Go Recommendation:**
Fix the two critical issues (permission-mode and concurrent config.json writes), then proceed. The plan is solid enough that a Go developer familiar with the codebase could implement it within the estimated timeline, provided the critical issues are resolved in the schema/design phase rather than discovered during Phase 2.

---

## Layer 1: Hidden Assumptions

### Assumption Register

| # | Assumption | Source | Verified? | Risk if False | Severity |
|---|-----------|--------|-----------|---------------|----------|
| A-1 | `--permission-mode delegate` grants tool access in `-p` mode | Phase 2, section 2.3 line 1606 | **CONTRADICTED** by `docs/PERMISSION_HANDLING.md` | Agents cannot use Write/Read/Bash tools; entire system non-functional | CRITICAL |
| A-2 | `claude -p --output-format json` returns JSON with `cost_usd` field | Phase 2, section 2.6 | **PARTIALLY VERIFIED** -- `spawnAgent.ts:parseCliOutput()` parses `cost_usd \|\| total_cost_usd` but field names are guesswork | Cost tracking returns 0.0 for all agents; budget enforcement broken | HIGH |
| A-3 | `cmd.ProcessState.Exited()` is available after `cmd.Wait()` returns | Phase 2, section 2.4 signal handler | **UNVERIFIED** -- Go's `ProcessState` does not have an `Exited()` method that works the way shown | Signal handler crashes with nil pointer or method-not-found | HIGH |
| A-4 | Atomic rename on same filesystem is always safe | Phase 1, section 1B | **VERIFIED** for ext4/btrfs on Linux | Low risk on target platform | LOW |
| A-5 | LLM agents will comply with stdout JSON schema via prompt engineering alone | Phase 1D, line 734-736 | **UNVERIFIED** -- acknowledged in plan as F7 "HIGH initially" | Downstream consumers (Beethoven, inter-wave scripts) get malformed input | MEDIUM |
| A-6 | `nohup` process detachment survives TUI exit | Phase 4, section 4.0 | **VERIFIED** -- standard Unix behavior | N/A | LOW |
| A-7 | `jq` is available on the target system | Phase 1G inter-wave script, Phase 4 launch verification | **UNVERIFIED** -- CachyOS/Arch may not have jq in minimal installs | Inter-wave scripts fail; launch verification fails | MEDIUM |
| A-8 | Claude CLI `-p` flag supports `--model` override | Phase 2, section 2.3 | **LIKELY TRUE** -- `spawnAgent.ts:buildCliArgs()` already uses this pattern | Low risk | LOW |
| A-9 | Multiple goroutines can safely call `writeConfigAtomic` concurrently | Phase 2, section 2.3 (inside `spawnAndWait` called from goroutines) | **FALSE** -- no mutex protects config writes | Config.json corruption or lost updates when wave members finish simultaneously | CRITICAL |
| A-10 | `CLAUDE_CODE_EFFORT_LEVEL` env var is respected by `claude -p` | Phase 2, section 2.3 line 1616 | **UNVERIFIED** -- this is a GOgent-specific convention, may only be read by hooks | Effort level has no effect on spawned agents | MEDIUM |

---

### A-1 Detail: Permission Mode is Broken in Pipe Mode (CRITICAL)

**Plan states (section 2.3, line 1606):**
```go
"--permission-mode", "delegate", // Auto-approve tool use
```

**Codebase evidence contradicts this.** From `/home/doktersmol/Documents/GOgent-Fortress/docs/PERMISSION_HANDLING.md` (lines 42-44):

> **Key Discovery:** The `--permission-mode` flag appears to be designed for interactive CLI sessions, not `stream-json` mode. In stream-json mode, permission events are always **error notifications**, not **permission requests**.

And line 54:
> Even when we specify `--permission-mode delegate`, suggesting the flag is ignored in stream-json mode.

The existing `spawnAgent.ts` (line 335) does use `--permission-mode delegate` BUT it also uses `--output-format json` (not `stream-json`). The `-p` pipe mode combined with `--output-format json` may behave differently from `stream-json`, but this has never been validated for tool permission grants.

**The actual working pattern** is `--allowedTools` pre-approval, as documented in `PERMISSION_HANDLING.md` lines 56-67 and implemented in `spawnAgent.ts` lines 337-339. The implementation plan makes **zero mention** of `--allowedTools`.

**Impact:** If `--permission-mode delegate` does not grant tool permissions in pipe mode, every spawned agent will fail on its first Read/Write/Bash call. The entire system is non-functional.

**Recommendation:** The Go binary MUST use `--allowedTools` to pre-approve the tools each agent needs, matching the pattern already proven in `spawnAgent.ts`. Add `--allowedTools "Read,Write,Glob,Grep,Bash,Edit"` to the CLI args. Optionally also include `--permission-mode delegate` as belt-and-suspenders, but do not rely on it alone. Add an explicit test in the Phase 2 quality gate: "Agent spawned via `claude -p` can successfully use Write tool."

---

### A-9 Detail: Concurrent Config.json Writes (CRITICAL)

**Plan states (section 2.3, line 1640):**
```go
member.PID = cmd.Process.Pid
writeConfigAtomic(teamDir, config)  // Record PID for /team-cancel
```

This is called inside `spawnAndWait()`, which is invoked as a goroutine per wave member (line 1554):
```go
go spawnAndWait(teamDir, config, member, agentsIndex, &wg)
```

When Wave 1 has 2+ members (e.g., einstein + staff-arch), both goroutines will:
1. Read the shared `config` pointer
2. Modify `member.PID`, `member.Status`, etc.
3. Call `writeConfigAtomic()` which marshals the entire config to JSON

This is a **classic data race**. Two goroutines concurrently modifying the same `config` struct and writing it to disk will produce:
- Lost updates (goroutine A writes, goroutine B overwrites with stale data)
- Corrupted in-memory state (concurrent field writes on non-atomic struct fields)

**Impact:** PID tracking, status updates, and cost tracking will be unreliable. In the worst case, a goroutine reads a half-written struct field and produces garbage JSON.

**Recommendation:** Add a `sync.Mutex` to protect all config reads and writes:

```go
type TeamRunner struct {
    config    *TeamConfig
    configMu  sync.Mutex
    teamDir   string
}

func (r *TeamRunner) updateMember(name string, fn func(*Member)) {
    r.configMu.Lock()
    defer r.configMu.Unlock()
    for i := range r.config.Members {
        if r.config.Members[i].Name == name {
            fn(&r.config.Members[i])
            break
        }
    }
    writeConfigAtomic(r.teamDir, r.config)
}
```

This is a fundamental concurrency issue that must be designed into the architecture, not patched later.

---

## Layer 2: Dependency Analysis

### External Dependencies

| Dependency | Version Coupling | Stability | Risk | Mitigation |
|-----------|-----------------|-----------|------|------------|
| Claude CLI `-p` flag | Tight | Likely stable (documented) | LOW | Already used in spawnAgent.ts |
| Claude CLI `--output-format json` structure | **Tight** | **Undocumented internal format** | **HIGH** | Parse defensively, log raw on failure |
| Claude CLI `--allowedTools` flag | Tight | Documented | LOW | Already used in spawnAgent.ts |
| Claude CLI `--max-budget-usd` flag | Loose | Documented | LOW | Already used in spawnAgent.ts |
| `jq` binary on PATH | Hard dependency for inter-wave scripts | N/A | MEDIUM | Add to Phase 0 prerequisites or rewrite in Go |
| `nohup` binary on PATH | Hard dependency for launch | Standard on all Linux | LOW | N/A |

### Internal Dependencies

| Component | Coupling | Risk |
|-----------|----------|------|
| `agents-index.json` schema | Go binary must parse `model`, `effortLevel` fields | MEDIUM -- schema changes break binary. Plan acknowledges this (A-2 in their register). |
| Session directory structure | `packages/tui/.claude/sessions/{YYMMDD}.{sessionId}/` | MEDIUM -- if TUI changes session dir naming, Go binary can't find teams |
| `processRegistry.ts` (TUI) | Parallel, not integrated | HIGH -- see Architecture Smells below |
| Hook system (`gogent-validate`, etc.) | Not triggered by `gogent-team-run` spawns | HIGH -- see F-NEW-1 below |

### Hidden Dependency: Hook Bypass

**The plan does not address this.** When `gogent-team-run` spawns `claude -p` processes directly, those processes are independent CLI sessions. They will load their own `.claude/settings.json` and hooks. This means:

1. `gogent-validate` (PreToolUse) WILL fire inside spawned agents -- this is fine
2. `gogent-sharp-edge` (PostToolUse) WILL fire -- this is fine
3. `gogent-agent-endstate` (SubagentStop) will fire for any Task() calls inside agents -- this is fine
4. `gogent-archive` (SessionEnd) will fire when agents exit -- but writes to the AGENT's session dir, not the team dir

**The implication:** ML telemetry, handoffs, and sharp-edge captures from team-spawned agents will scatter into individual agent session directories, not the team directory. This is not a blocker but means the `/team-result` command cannot use hook-generated artifacts. The plan's approach of having agents write stdout files directly is correct and side-steps this issue, but it should be explicitly documented.

**Severity:** MEDIUM

---

## Layer 3: Failure Mode Analysis

### Plan's Catalog Assessment

The plan's F1-F10 catalog is good. Probabilities and impacts are reasonable. However, several failure modes are missing.

### Missing Failure Modes

| ID | Failure Mode | Probability | Impact | Detection | Recovery | Severity |
|----|-------------|------------|--------|-----------|----------|----------|
| F-NEW-1 | Agent uses Task() inside team-spawned session | MEDIUM | MEDIUM | `gogent-validate` blocks it | Agent fails the Task() call, may still complete via other means. BUT: if agent's workflow depends on Task(), it will fail entirely. | HIGH |
| F-NEW-2 | Concurrent config.json writes corrupt state | HIGH (every multi-member wave) | HIGH | Inconsistent PID/status after wave | None without mutex -- see A-9 | CRITICAL |
| F-NEW-3 | Agent writes stdout file in wrong format (valid JSON, wrong schema) | HIGH initially | MEDIUM | `validateStdout()` only checks envelope fields | Inter-wave script `jq` extractions silently produce empty output | MEDIUM |
| F-NEW-4 | Go binary launched but team_dir doesn't exist | LOW | HIGH | `readConfig` fails | Binary exits immediately; runner.log shows error | LOW |
| F-NEW-5 | Two teams launched to same directory simultaneously | LOW | HIGH | Second binary overwrites first's config.json | Add flock or PID-file check | MEDIUM |
| F-NEW-6 | Agent reads config.json while Go binary is mid-write | MEDIUM | LOW | Atomic rename prevents this | N/A -- correctly handled by plan | LOW |

### F-NEW-1 Detail: Task() Availability in Team-Spawned Agents

Agents spawned by `gogent-team-run` via `claude -p` are Level 0 CLI sessions from Claude's perspective (they don't inherit `GOGENT_NESTING_LEVEL` context for Task() blocking -- wait, the plan DOES set `GOGENT_NESTING_LEVEL=2` in the env). This means:

- `gogent-validate` sees `GOGENT_NESTING_LEVEL=2` and the agents are treated as Level 2 subagents
- Task() calls from these agents will go through `gogent-validate` which checks the `task_invocation_blocked` rule
- If Task(opus) is attempted, it will be blocked -- this is correct behavior

However, the plan sets these agents as independent CLI sessions that CAN use Task() (unlike MCP-spawned agents which cannot). The `claude -p` process has full Task() capability. This means:

- Einstein could potentially spawn sub-agents via Task() -- is this intended?
- The `agents-index.json` shows Einstein has `spawned_by: ["mozart"]` and no `can_spawn` field
- But nothing in the Go binary prevents Einstein from calling Task(haiku) or Task(sonnet)

**This is not necessarily a bug** -- it may be desirable for agents to have Task() access. But it is an undocumented behavioral difference from MCP-spawned agents where Task() is unavailable. The plan should explicitly state whether team-spawned agents should have Task() access and whether this changes their behavior or cost profile.

**Severity:** HIGH (design decision needed, not a bug per se)

### Rollback Assessment

| Phase | Rollback Path | Assessment |
|-------|--------------|------------|
| Phase 0 | Revert signal handler changes | Clean -- isolated TypeScript changes |
| Phase 1 | Delete schema files | Clean -- no runtime dependency yet |
| Phase 2 | Remove binary, delete `cmd/gogent-team-run/` | Clean -- binary is additive |
| Phase 3 | Remove skill definitions | Clean -- skills are additive |
| Phase 4 | Revert orchestrator prompts to current behavior | **Risky** -- current behavior is the fallback, but prompt changes may have subtle interactions |

**Overall rollback assessment:** Good. Each phase is additive. The only risk is Phase 4 where orchestrator prompt changes could be hard to revert cleanly if partial.

---

## Layer 4: Cost-Benefit Analysis

### Is 13-19 days justified?

**For the braintrust workflow alone?** Probably not. The current flow works (TUI freezes for ~5 minutes, but produces results).

**For the broader platform?** Yes. The team pattern is a foundation for:
1. Parallel review (4 reviewers, ~4x faster)
2. Parallel implementation (wave-based task execution)
3. Non-blocking TUI (user can continue working)

The ROI is positive if orchestration is used 3+ times per week, which based on the routing telemetry system it clearly is.

### Maintenance Cost of Third Spawning Path

This is the plan's biggest architectural cost. Today there are two spawn paths:
1. `Task()` -- Claude Code native, router-level only
2. `mcp__gofortress__spawn_agent` -- TypeScript MCP tool, TUI subagents

The plan adds:
3. `gogent-team-run` spawning `claude -p` -- Go binary, background orchestration

**Three paths means three places to update when:**
- Claude CLI flags change
- Agent configuration schema changes
- Cost tracking format changes
- Permission model changes

The plan partially mitigates this by having all three paths read `agents-index.json` for agent configuration. But CLI flag construction is duplicated between `spawnAgent.ts:buildCliArgs()` and the Go binary's `spawnAndWait()`.

**Recommendation:** Extract CLI arg construction into a shared location. Options:
- A Go function in `pkg/` that both `spawnAgent.ts` and `gogent-team-run` reference (requires spawnAgent.ts to call a Go binary for args -- too complex)
- A JSON config in `agents-index.json` that specifies per-agent CLI flags (simpler, both consumers read the same file)
- Accept the duplication for now, document it as tech debt

**Severity:** MEDIUM -- acceptable for MVP, but document the duplication and plan convergence.

### Complexity Budget

| Component | Essential Complexity | Accidental Complexity |
|-----------|---------------------|----------------------|
| Wave scheduler | Essential -- core value proposition | None |
| Stdin/stdout schemas | Essential -- structured IPC | Moderate -- 6 different schema types is ambitious for MVP |
| Inter-wave scripts | Essential for braintrust | Low |
| Heartbeat/orphan detection | Essential for reliability | Low |
| Budget enforcement | Essential for cost control | Low |
| Prompt envelope | Essential for agent orientation | Low |
| 3 team templates | **Defer review + implementation templates to Phase 5** | Could start with braintrust only |

**Recommendation:** Consider shipping Phase 1-4 with braintrust template only. Add review and implementation templates after the braintrust flow is proven. This reduces schema design time from 2-3 days to 1-2 days and reduces Phase 4 from 3-4 days to 1-2 days (only Mozart rewrite needed).

**Severity:** LOW -- optimization, not a blocker.

---

## Layer 5: Testing Strategy

### What's Specified

The quality gates (lines 2134-2164) are good. They include integration testing with real Claude CLI, budget enforcement testing, signal handling testing, and end-to-end braintrust runs.

### What's Missing

| Gap | Risk | Recommendation | Severity |
|-----|------|---------------|----------|
| No unit tests specified for Go binary | Code regressions during development | Add table-driven tests for: `buildPromptEnvelope()`, `extractCostFromCLIOutput()`, `validateStdout()`, wave ordering logic | HIGH |
| No race condition testing | Data races in concurrent wave execution | Run Go binary under `-race` flag during testing; add specific test for 2+ goroutines writing config simultaneously | HIGH |
| No test for agent stdout schema compliance | Agents don't produce expected JSON | Create a mock prompt that asks agent to write a known stdout file; validate schema post-hoc | MEDIUM |
| No test for stale heartbeat cleanup | Orphans accumulate across sessions | Test: start binary, kill -9 it, verify next session detects stale heartbeat and cleans up PIDs | MEDIUM |
| No performance/load test | Unknown behavior with 4+ parallel agents (review template) | Test review template with 4 parallel agents; measure memory, disk I/O, CLI startup time | LOW |

### Testing Strategy Recommendation

```
Unit tests (Go):
  - buildPromptEnvelope: verify absolute paths, template expansion
  - extractCostFromCLIOutput: valid JSON, malformed JSON, missing fields
  - validateStdout: valid envelope, missing fields, empty file, non-JSON
  - wave ordering: verify wave 2 waits for wave 1

Integration tests (with real CLI):
  - Single agent spawn + stdout write + validate
  - Two parallel agents (wave 1) + sequential agent (wave 2)
  - Budget exceeded mid-wave
  - SIGTERM forwarding to children
  - Agent timeout + retry

Race condition tests:
  - go test -race ./cmd/gogent-team-run/...
  - Specific test: 4 goroutines completing simultaneously, verify config.json consistency
```

**Severity:** HIGH overall -- the plan has good quality gates but no unit test plan.

---

## Layer 6: Architecture Smells

### Smell 1: Dual Process Management (Acknowledged)

The plan acknowledges this (line 898): "Two systems tracking processes." The Go binary tracks PIDs in config.json; the TUI's `ProcessRegistry` tracks spawned processes in memory. Team-spawned agents are invisible to `ProcessRegistry`.

**Impact:** If the TUI exits, `ProcessRegistry.cleanupAll()` will NOT kill team-spawned agents (because they were launched via `nohup` and are not registered). This is by design -- the Go binary manages its own lifecycle. But it means:

- `/team-cancel` must work via the Go binary's config.json PID, not the TUI's process registry
- The Phase 0 fix (wiring ProcessRegistry cleanup) does NOT help with team-spawned agents
- Orphan detection relies entirely on the heartbeat mechanism

**This is acceptable** but should be explicitly documented: "Team-spawned agents are outside TUI process management by design. Orphan cleanup is via heartbeat expiry, not ProcessRegistry."

**Severity:** MEDIUM

### Smell 2: File-Based IPC Without Locking

The plan uses files (config.json, stdin/stdout files) as the IPC mechanism. This is appropriate for the use case (processes don't need real-time communication), but:

- No file locking on config.json reads (slash commands read while Go binary writes)
- Atomic rename handles write-time corruption but not read-time races
- On Linux, `os.Rename` is atomic on the same filesystem (verified for ext4/btrfs)

**Assessment:** The atomic write pattern is sufficient. Readers may occasionally read stale data (old config.json before rename completes), but this only affects `/team-status` display accuracy by fractions of a second. Acceptable.

**Severity:** LOW

### Smell 3: Schema Enforcement via Prompting

The stdout schemas are enforced purely by prompt engineering (telling the agent "write JSON matching this template"). The Go binary validates only the envelope (`$schema`, `status` fields exist). Content validation is explicitly punted.

**This is the right call for MVP.** Attempting to validate deeply nested JSON schemas in Go adds complexity and brittleness. The inter-wave scripts use `jq` with fallback patterns (`2>/dev/null || echo "(fallback)"`) which gracefully handles malformed content.

**Severity:** LOW -- correct design decision.

### Smell 4: Recursive Retry in spawnAndWait

The plan's retry logic (line 1686) calls `spawnAndWait` recursively:

```go
if member.Status == "failed" && member.RetryCount < member.MaxRetries {
    member.RetryCount++
    // ...
    spawnAndWait(teamDir, config, member, agentsIndex, wg)
    return
}
```

This works but has a subtle issue: `wg.Done()` is deferred at the top of `spawnAndWait`. On retry, the recursive call will also defer `wg.Done()`, causing the WaitGroup counter to go negative (panic) because the original `wg.Add(1)` only counted once.

**Fix:** Either:
1. Move retry logic to the caller (the wave loop), not inside `spawnAndWait`
2. Call `wg.Add(1)` before each recursive retry
3. Restructure as a for-loop instead of recursion

**Severity:** HIGH -- this will panic at runtime on any retry.

---

## Layer 7: Contractor Readiness

### The Monday Morning Test

Can a Go developer start Monday with zero questions?

**Answer: Almost.** The plan is unusually detailed for a design document. Specific code snippets, file paths, and JSON schemas are provided. However, the following gaps would block or slow a contractor:

### Ambiguities

| Location | Ambiguity | Resolution Needed |
|----------|-----------|-------------------|
| Phase 2, section 2.3 | "Auto-approve tool use" via `--permission-mode delegate` | Must specify `--allowedTools` instead (see C-1) |
| Phase 2, section 2.6 | `cost_usd` field name in CLI JSON output | Need to verify actual field names by running `claude -p --output-format json` and inspecting output |
| Phase 2, section 2.2 | `projectRoot` used in `cmd.Dir` (line 1611) but never defined | How does the Go binary determine project root? From config.json? From cwd? From env var? |
| Phase 4, section 4.2 | "Router handles directly" for review workflow | Router is Opus-tier. Is writing 4 stdin JSON files "trivial enough" for Router? Or should a Sonnet agent prepare them? |
| Phase 3 | Slash commands as "skill definitions" | No file paths or implementation details for how skills are registered in the system |

### Missing Specifications

| What's Missing | Impact | Severity |
|---------------|--------|----------|
| How `projectRoot` is determined by the Go binary | Contractor will guess wrong | HIGH |
| Error handling for `loadAgentsIndex()` when file not found | Binary crashes on misconfigured system | MEDIUM |
| Log format for `runner.log` (structured or free-form?) | Inconsistent with other Go binaries | LOW |
| How slash commands discover the current session directory | Cannot implement `/team-status` without this | HIGH |
| Whether Go binary should be installed to `~/.local/bin/` like other binaries or elsewhere | Contractor doesn't know build/install target | LOW |

### Red Flag Phrases

| Phrase | Location | Issue |
|--------|----------|-------|
| "Auto-approve tool use" | Section 2.3 comment | Implies `--permission-mode delegate` is sufficient; contradicted by evidence |
| "or a new cleanup hook" | Section 2.5, line 1746 | Vague -- is this in scope or not? |
| "Future: status bar indicator" | Section 7 risk table | Fine as future work, clearly scoped out |
| "Content validation is NOT done by Go binary. It's prompt engineering's job" | Section 1D, line 1074-1075 | Clear and correct, but should note that inter-wave scripts may fail silently on bad content |

### Contractor Readiness Verdict

**Ready with conditions.** A Go developer with access to this plan and the codebase could implement Phases 0-2 after the critical issues are resolved. Phases 3-4 need more detail on skill registration and session directory discovery.

---

## Commendations

1. **Thorough schema design.** The field-by-field rationale table (lines 279-321) with "Who Reads It / Who Writes It" is excellent. This prevents ambiguity about ownership.

2. **Honest failure mode catalog.** F1-F10 with probability/impact ratings is unusually good for a design document. The plan does not pretend everything will work perfectly.

3. **Atomic write pattern.** Identifying config.json corruption as a risk and designing the write-tmp-then-rename pattern upfront prevents a class of bugs that usually surface in production.

4. **Inter-wave script with graceful degradation.** The `jq` extraction with `2>/dev/null || echo "(fallback)"` patterns means a malformed agent output does not crash the pipeline. This is defensive in the right way.

5. **Clear separation of planner vs. executor.** The design that "planner writes stdin files, Go binary reads them" and "agents write stdout files, Go binary validates envelope only" creates clean boundaries with minimal coupling.

---

## Recommendations

### Must Fix (Before Implementation)

1. **C-1: Add `--allowedTools` to CLI args** (replaces reliance on `--permission-mode delegate` alone). Verify by running a single `claude -p` agent with Write tool access and confirming it works. Add to Phase 2 quality gate.

2. **C-2: Add `sync.Mutex` for config.json access.** Design a `TeamRunner` struct with a mutex-protected `updateConfig()` method. All goroutines must go through this method to read or write config state. Run `go test -race` as part of CI.

### Should Fix (Before Phase 2 Completion)

3. **H-1: Fix recursive retry WaitGroup panic.** Restructure retry as a for-loop inside `spawnAndWait` or move retry logic to the wave loop caller.

4. **H-2: Determine `projectRoot` resolution.** Specify whether the Go binary reads it from config.json, an env var, or discovers it by walking up to find `go.mod`/`.claude/`. Document in section 2.3.

5. **H-3: Verify CLI JSON output field names.** Run `echo "hello" | claude -p --output-format json` and document the actual output structure. Update `extractCostFromCLIOutput()` to match reality. Consider using the `--max-budget-usd` flag as a secondary budget enforcement mechanism (the CLI itself would abort on budget).

6. **H-4: Document Task() availability in team-spawned agents.** Explicitly state whether team agents should have Task() access. If not, add `GOGENT_TASK_BLOCKED=true` to the env and check it in `gogent-validate`.

### Consider (Post-MVP)

7. **M-1: Replace `jq` dependency with Go-native extraction.** Write `gogent-team-prepare-synthesis` as a Go binary instead of a bash script. This eliminates the `jq` dependency and matches the `cmd/` pattern.

8. **M-2: Add unit test plan for Go binary.** Table-driven tests for `buildPromptEnvelope()`, `extractCostFromCLIOutput()`, `validateStdout()`, wave ordering.

9. **M-3: Reduce MVP scope to braintrust template only.** Defer review and implementation templates to post-MVP. This cuts 3-5 days of effort and lets you validate the pattern before scaling it.

10. **M-4: Add file locking or PID-file check to prevent duplicate team launches.** Use `flock` on config.json or check `background_pid` is null before launching.

11. **M-5: Specify session directory discovery for slash commands.** How does `/team-status` find the current session's teams directory? From env var? From TUI state? This blocks Phase 3 implementation.

### Low Priority

12. **L-1: Structured logging for runner.log.** Use JSON lines format to match the project's telemetry patterns (`ml-tool-events.jsonl`, etc.).

13. **L-2: Add `GOGENT_TEAM_DIR` to spawned agent env.** Already in the plan (line 1620) -- just confirming this is good for agent self-orientation.

14. **L-3: Document that team-spawned agents are outside ProcessRegistry scope.** Add a note to the architecture docs so future developers don't try to wire them through the TUI's process tracking.

---

## Final Sign-Off

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-02-06
**Plan Size:** ~2200 lines, 4 phases, 7 sub-phases in Phase 1

**Conditions for Approval:**

- [ ] C-1: `--allowedTools` added to CLI args, `--permission-mode delegate` demoted to optional
- [ ] C-2: Mutex-protected config.json writes designed into `TeamRunner` struct

**Post-Approval Monitoring:**

- Watch Phase 2 for: CLI output format surprises, permission denials, race conditions
- Watch Phase 4 for: Router complexity when preparing review/implementation stdin files
- Benchmark: Does 4-agent parallel launch cause measurable system load? (CPU, memory, disk)
- Track: Agent stdout schema compliance rate across first 10 braintrust runs -- expect iteration needed
