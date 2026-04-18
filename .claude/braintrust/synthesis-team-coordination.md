# Braintrust Synthesis: Background Team Orchestration via `goyoke-team-run`

> **Synthesized by Beethoven** (Final revision with user decisions locked in)
> **Timestamp**: 2026-02-06T14:15:00Z
> **Inputs**:
>   - Implementation Plan: `/home/doktersmol/Documents/goYoke/tickets/team-coordination/IMPLEMENTATION-PLAN.md`
>   - Einstein Analysis: `/home/doktersmol/Documents/goYoke/.claude/braintrust/einstein-team-coordination-analysis.md`
>   - Staff-Architect Review: `/home/doktersmol/Documents/goYoke/.claude/braintrust/staff-architect-team-coordination-review.md`

---

## 1. Executive Summary

The implementation plan for `goyoke-team-run` is **approved with conditions**. Both analysts agree the plan addresses a real problem (TUI freezing during multi-agent orchestration) with a sound architectural approach (separating planning from execution). The structured I/O schemas are the plan's strongest contribution.

**Decisions locked in:**

1. **All three templates** (braintrust, review, implementation) ship in Phase 1, with all three orchestrator rewrites in Phase 4. This broadens MVP scope but delivers full platform value.
2. **Full daemon pattern** for process detachment: `SysProcAttr{Setsid: true}`, close stdin, redirect stdout/stderr to log, PID file write/cleanup. This supports concurrent teams (braintrust + review running simultaneously) with proper double-start prevention.
3. **Go binary** for inter-wave scripts: `cmd/goyoke-team-prepare-synthesis/main.go` replaces the bash/jq script. No external dependencies; typed JSON parsing; matches the `cmd/` pattern.
4. **TUI concurrency** is a separate ticket. Does not block the Go binary work. Einstein's root cause analysis (streaming mutex in `useClaudeQuery.ts`) is captured in that ticket's description.

Two critical bugs must be fixed before implementation begins: the `--permission-mode delegate` flag does not grant tool access in pipe mode (use `--allowedTools` instead), and concurrent goroutines writing to a shared `config` struct without a mutex will cause data races. A third bug (recursive retry causing WaitGroup panic) must be fixed during Phase 2.

**Revised timeline**: 16-24 days (up from 13-19, reflecting broader scope and Go inter-wave binary).

---

## 2. Convergence Points

These are areas where both analysts independently agree. High confidence.

### 2.1 Strong Agreement

| # | Topic | Einstein's Position | Staff-Architect's Position | Unified Conclusion | Confidence |
|---|-------|--------------------|-----------------------------|---------------------|------------|
| C-1 | **Plan is fundamentally sound** | "Separating orchestration planning from execution is economically rational" (Axiom 1) | "APPROVE_WITH_CONDITIONS -- the plan is solid enough that a Go developer could implement it" | The core architecture (PLAN foreground, EXECUTE background, DELIVER on-demand) is correct. Proceed. | HIGH |
| C-2 | **Structured I/O schemas are the strongest element** | "The strongest part of the plan. They transform agents from opaque text-in/text-out boxes into components with defined interfaces." | "Thorough schema design... field-by-field rationale table is excellent" (Commendation 1) | The stdin/stdout schema design is production-quality. It should be preserved regardless of how the execution engine evolves. | HIGH |
| C-3 | **Three supervision topologies is a risk** | "Introduces a third process supervision topology... long-term cost of triplication may exceed the cost of fixing the root cause" | "Three paths means three places to update when CLI flags change, agent config changes, cost tracking changes" | Dual/triple process management is the plan's biggest architectural debt. Acceptable for MVP but must be tracked. | HIGH |
| C-4 | **`claude -p` output format is an unstable dependency** | "Assumption: `claude -p --output-format json` produces stable, parseable JSON -- Confidence: Medium" | "Undocumented internal format -- Risk: HIGH. Parse defensively, log raw on failure" | The cost extraction mechanism is fragile. Build it defensively from day one with fallbacks and raw logging. | HIGH |
| C-5 | **Prompt-based schema enforcement has limits** | "Prompting is not a contract enforcement mechanism. Semantic emptiness passes validation." (Axiom 5) | "Schema enforcement via prompting -- the right call for MVP" with graceful degradation via jq fallbacks | Accept prompt-based enforcement for MVP. Mitigate via envelope-only validation + fallback patterns. Plan for iteration. | HIGH |
| C-6 | **Atomic config.json writes are correct** | Listed as a derived implication of structured contracts | Commendation 3: "write-tmp-then-rename pattern upfront prevents a class of bugs" | The atomic write pattern is well-designed and sufficient for the file-based IPC model. | HIGH |
| C-7 | **Heartbeat-based orphan detection is eventually consistent** | "Orphan detection is eventually consistent with a granularity of 'next session start' -- potentially hours or never" | "Orphan detection relies entirely on the heartbeat mechanism" | Heartbeat expiry is the correct mechanism but has a gap: orphans persist until the next TUI session. Acceptable for MVP. | MEDIUM |

### 2.2 Complementary Insights

These are areas where both analysts analyzed the same topic but contributed different, non-conflicting perspectives that together give a fuller picture.

| Einstein Contribution | Staff-Architect Contribution | Combined Value |
|----------------------|------------------------------|----------------|
| Root cause is the TUI's `isStreaming` mutex on user input, not the JS event loop itself | Confirmed via code references: `setIsStreaming(true)` at line 630 locks the UI | The freeze is an application-level state machine choice, diagnosable and fixable independent of the Go binary |
| Process group implications of `nohup` (SIGINT propagation, `setsid` needed) | `nohup` behavior verified as standard Unix, but did not analyze process group edge cases | `nohup` is insufficient; full daemon pattern chosen (see Decision 2) |
| Inter-wave scripts add value independent of the Go binary | Inter-wave script with graceful degradation is a commendation (jq + fallback) | Inter-wave script will be rewritten as Go binary (see Decision 3) |
| Structured schemas should be a reusable primitive, not coupled to Go binary | Schemas work within the plan's Go binary context | Design schemas as standalone artifacts that could serve both spawn paths (long-term value) |
| `CLAUDE_CODE_EFFORT_LEVEL` env var may not be read by CLI | Listed as A-10, unverified, MEDIUM severity | Must verify before relying on it; low-cost check during Phase 2 |

---

## 3. Divergence Resolution

### D-1: `nohup` vs. Proper Daemon Pattern -- RESOLVED

**Einstein's Position**: `nohup` is "insufficient for production process detachment." A proper daemon pattern (fork, `setsid()`, close inherited fds, PID file) prevents SIGINT propagation and ensures clean process group isolation.

**Staff-Architect's Position**: `nohup` process detachment "verified -- standard Unix behavior." Listed as LOW severity, A-6.

**Resolution**: **User chose Option C: Full Daemon Pattern.**

The Go binary will implement:
- `cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}` for process group isolation
- Close stdin on the Go binary itself
- Redirect stdout/stderr to `runner.log`
- PID file write on startup, cleanup on exit
- PID file provides: discovery (`ls *.pid`), lockfile semantics, double-start prevention

This is the correct choice given the user's plan to run concurrent teams (braintrust + review simultaneously). PID files provide the coordination primitives needed for multi-team management.

**Priority**: HIGH
**Phase affected**: Phase 2 (Go binary startup and child spawning)

---

### D-2: Is the Go Binary the Right Solution? -- RESOLVED

**Einstein's Position**: The Go binary is "a workaround for the TUI's concurrency model, not a fundamental architectural necessity."

**Staff-Architect's Position**: Did not consider alternatives. The cost-benefit analysis says "Yes" for the broader platform value.

**Resolution**: Both perspectives remain valid but address different concerns. The Go binary proceeds as the primary deliverable. The TUI concurrency issue is filed as a separate ticket (TC-015) that does not block this work. Einstein's root cause analysis (streaming mutex in `useClaudeQuery.ts`) provides the technical foundation for that ticket.

**Priority**: Go binary is the current implementation. TUI concurrency is a separate MEDIUM-priority item.

---

### D-3: Severity of Third Spawning Path -- RESOLVED

**Einstein's Position**: "Significant architectural risk" and violation of single-supervisor axiom.

**Staff-Architect's Position**: "Smell 1: Dual Process Management" (MEDIUM severity). Acceptable for MVP.

**Resolution**: Accept for MVP. Mitigate by:
1. Adding a shared `cli_flags` field to `agents-index.json` (TC-014)
2. Documenting the three paths in ARCHITECTURE.md
3. Tracking as tech debt with concrete trigger: "When CLI flags change for the third time, unify"

**Priority**: MEDIUM
**Phase affected**: Phase 1 (schema enhancement)

---

### D-4: Socket-Based IPC vs. File-Based IPC -- RESOLVED

**Einstein's Position**: Socket-based IPC architecturally superior but "probably overengineered for the current use case."

**Staff-Architect's Position**: File-based IPC is "appropriate for the use case."

**Resolution**: Keep file-based IPC. No change to plan. Socket IPC is a future consideration only if real-time streaming or bidirectional control becomes a requirement.

**Priority**: NOT APPLICABLE for MVP.

---

### D-5: Task() Availability in Team-Spawned Agents -- RESOLVED

**Einstein's Position**: Did not explicitly address.

**Staff-Architect's Position**: Flagged as F-NEW-1 (HIGH severity design decision). Team-spawned agents via `claude -p` CAN use Task(), unlike MCP-spawned agents.

**Resolution**: Team-spawned agents SHOULD have Task() access at Level 2. `GOYOKE_NESTING_LEVEL=2` means `goyoke-validate` blocks Task(opus) but allows Task(haiku) and Task(sonnet). This is correct behavior -- Einstein should be able to spawn a haiku scout for codebase exploration but should not spawn another Opus agent. Document explicitly in Phase 1 and verify in Phase 2.

**Priority**: MEDIUM
**Phase affected**: Phase 1 (documentation), Phase 2 (verify behavior)

---

## 4. Decision Points -- ALL RESOLVED

### Decision 1: MVP Scope -- RESOLVED: Option B (All Three Templates)

| Option | Chosen | Effort Impact |
|--------|--------|---------------|
| A: Braintrust only | No | Would have saved 3-5 days |
| **B: All three templates** | **YES** | +3-5 days (Phase 1: +1 day, Phase 4: +2-3 days) |

**Rationale**: User wants full platform value. All three orchestrator rewrites (Mozart, review-orchestrator, impl-manager) ship in Phase 4. The broader scope is justified because the core wave scheduler (Phase 2) is the same regardless of template count -- the extra effort is in schema design and prompt rewrites, which are lower-risk.

---

### Decision 2: Process Detachment -- RESOLVED: Option C (Full Daemon Pattern)

| Option | Chosen | Implementation |
|--------|--------|---------------|
| A: Keep `nohup` | No | -- |
| B: Use `setsid` | No | -- |
| **C: Full daemon pattern** | **YES** | `SysProcAttr{Setsid: true}`, close stdin, redirect stdout/stderr, PID file |

**Rationale**: User plans concurrent teams (braintrust + review running simultaneously). PID files provide discovery, lockfile semantics, and double-start prevention. The full daemon pattern is the foundation for multi-team orchestration.

---

### Decision 3: Inter-Wave Scripts -- RESOLVED: Option B (Go Binary)

| Option | Chosen | Implementation |
|--------|--------|---------------|
| A: Keep bash + jq | No | -- |
| **B: Rewrite as Go binary** | **YES** | `cmd/goyoke-team-prepare-synthesis/main.go` |
| C: Ship with jq, add Go later | No | -- |

**Rationale**: No jq dependency. Matches existing `cmd/` pattern. Typed JSON parsing with proper error handling. +1-2 days effort.

---

### Decision 4: TUI Concurrent Query Support -- RESOLVED: Option A (Separate Ticket)

| Option | Chosen | Implementation |
|--------|--------|---------------|
| **A: Separate ticket** | **YES** | TC-015, does not block Go binary work |
| B: Fix TUI first | No | -- |

**Rationale**: The Go binary provides value beyond parallelism (detachment, token savings, typed process management). Einstein's root cause analysis goes into the separate ticket's description.

---

## 5. Final Ticket List

### Ticket Dependency Graph

```
TC-001 (permission flags)  ─────────────────────┐
TC-002 (mutex)  ────────────────────────────────┐│
TC-006 (projectRoot)  ─────────────────────────┐││
TC-014 (cli_flags in agents-index)  ──────────┐│││
                                               ││││
TC-004 (daemon pattern)  ─────────────────────┐││││
                                              │││││
Phase 1 schema work (TC-007, TC-009, TC-014) ─┤││││
                                              │││││
                                              ▼▼▼▼▼
                                       TC-008 (Go binary)
                                              │
                                              ├──► TC-003 (retry fix, inside binary)
                                              ├──► TC-005 (CLI format verification)
                                              │
                                              ▼
                                       TC-010 (inter-wave Go binary)
                                              │
                                              ▼
                                       TC-011 (unit tests)
                                              │
                                              ▼
                                       TC-012 (slash commands)
                                              │
                                              ▼
                                       TC-013 (orchestrator rewrites)

Independent:
  TC-015 (TUI concurrency) -- no dependencies on above
  TC-016 (duplicate launch prevention) -- after TC-008
```

---

### Phase 0: Foundation Fixes

#### TC-001: Replace `--permission-mode delegate` with `--allowedTools`

**Priority**: CRITICAL
**Phase**: 0 (must be resolved in design before Phase 2 implementation)
**Blocked By**: none
**Effort**: 0.5 days (design decision + documentation; implementation is part of TC-008)

**Description**:
The Go binary's `spawnAndWait()` function uses `--permission-mode delegate` to grant tool access to spawned agents. Evidence from `docs/PERMISSION_HANDLING.md` and `spawnAgent.ts` shows this flag is insufficient in pipe mode. The working pattern is `--allowedTools "Read,Write,Glob,Grep,Bash,Edit"`, as already implemented in `spawnAgent.ts:buildCliArgs()`.

Document the correct CLI arg pattern for the Go binary. Keep `--permission-mode delegate` as belt-and-suspenders but do not rely on it. The actual implementation happens in TC-008 (Go binary), but the design decision must be made here so TC-008 has the correct specification.

**Acceptance Criteria**:
- [ ] Design document specifies `--allowedTools` as the primary permission mechanism
- [ ] Per-agent tool lists defined (some agents may not need Bash)
- [ ] Verified with a real `claude -p` invocation that tools work (manual test)
- [ ] Added to Phase 2 quality gate: "Agent spawned via `claude -p` can successfully use Write tool"

**Source**: Staff-Architect C-1, Assumption A-1 (CRITICAL)

---

#### TC-002: Design Mutex-Protected Config Access

**Priority**: CRITICAL
**Phase**: 0 (must be designed before Phase 2 implementation)
**Blocked By**: none
**Effort**: 0.5 days (design; implementation is part of TC-008)

**Description**:
When Wave 1 has 2+ members, multiple goroutines concurrently modify the shared `config` struct (updating PID, status, cost) and call `writeConfigAtomic()`. This is a classic data race. The Go binary must use a `TeamRunner` struct with a `sync.Mutex` to serialize all config reads and writes.

Design the `TeamRunner` struct with:
- `config *TeamConfig` and `configMu sync.Mutex`
- `updateMember(name string, fn func(*Member))` method that locks, applies the function, and writes atomically
- All goroutines use `updateMember()` instead of directly modifying config

The actual implementation happens in TC-008, but the struct design must be specified here.

**Acceptance Criteria**:
- [ ] `TeamRunner` struct design documented with mutex strategy
- [ ] `go test -race` requirement added to TC-008 and TC-011
- [ ] Specific test case defined: 4 goroutines completing simultaneously, config.json has all 4 statuses correct

**Source**: Staff-Architect C-2, Assumption A-9 (CRITICAL)

---

### Phase 1: Schema Design

#### TC-006: Add `project_root` to Team Config Schema

**Priority**: HIGH
**Phase**: 1
**Blocked By**: none
**Effort**: 0.5 days

**Description**:
The Go binary uses `cmd.Dir = projectRoot` when spawning agents, but `projectRoot` is never defined in the config schema. The binary needs to know the project root for agents' Read/Write tool calls to resolve paths correctly.

Add `project_root` as a top-level field in the team config.json schema. The planner (Mozart or Router) resolves this at planning time and writes it into config.json. The Go binary reads it and sets `cmd.Dir` accordingly.

**Acceptance Criteria**:
- [ ] `project_root` field added to team config schema (top-level, required, absolute path)
- [ ] All three team templates (braintrust, review, implementation) include `project_root`
- [ ] Prompt envelope references `project_root` from config.json
- [ ] Stdin file `paths.project_root` matches config.json `project_root`

**Source**: Staff-Architect H-2 (HIGH), Ambiguity table

---

#### TC-007: Document Task() Access for Team-Spawned Agents

**Priority**: MEDIUM
**Phase**: 1
**Blocked By**: none
**Effort**: 0.5 days

**Description**:
Agents spawned by `goyoke-team-run` via `claude -p` are independent CLI sessions with full Task() capability. This differs from MCP-spawned agents where Task() is unavailable. The plan sets `GOYOKE_NESTING_LEVEL=2`, so `goyoke-validate` blocks Task(opus) but allows Task(haiku/sonnet).

Explicitly document this design decision: "Team-spawned agents have Task() access at Level 2, which means Task(haiku) and Task(sonnet) are allowed but Task(opus) is blocked by goyoke-validate."

**Acceptance Criteria**:
- [ ] Design decision documented in implementation plan and prompt envelope docs
- [ ] Verified during Phase 2: Einstein can call Task(haiku) but not Task(opus) when spawned by goyoke-team-run
- [ ] Prompt envelope mentions available capabilities

**Source**: Staff-Architect F-NEW-1 (HIGH)

---

#### TC-009: Design All Three Team Templates

**Priority**: HIGH
**Phase**: 1
**Blocked By**: none
**Effort**: 2-3 days

**Description**:
Design and write the complete schema set for all three workflow templates:

1. **Braintrust** (`braintrust.json`): Mozart interview-then-background. 3 members (einstein, staff-arch, beethoven), 2 waves, inter-wave synthesis script. Already partially specified in the implementation plan.

2. **Review** (`review.json`): Fully backgroundable. 4 reviewers (backend, frontend, standards, architecture) in 1 wave. Router fills stdin from git diff. No inter-wave script.

3. **Implementation** (`implementation.json`): Fully backgroundable. Dynamic members from specs.md task DAG. Multiple waves based on `blocked_by` relationships. Router fills stdin from ticket descriptions.

For each template, deliver:
- Team config template (`.claude/schemas/teams/{workflow}.json`)
- Stdin schema per agent type (`.claude/schemas/stdin/{agent}.json`)
- Stdout schema per agent type (`.claude/schemas/stdout/{agent}.json`)

**Acceptance Criteria**:
- [ ] All three team config templates written and validated by hand-filling
- [ ] Stdin files contain all information an agent needs (test by reading as a human)
- [ ] Stdout schemas are writable by an LLM given the stdin instructions
- [ ] Common envelope is consistent across all templates
- [ ] `project_root` field present in all templates (per TC-006)

**Source**: User Decision 1 (all three templates), Implementation Plan Phase 1

---

#### TC-014: Add `cli_flags` to agents-index.json

**Priority**: MEDIUM
**Phase**: 1
**Blocked By**: none
**Effort**: 1 day

**Description**:
Three spawn paths (`Task()`, `spawnAgent.ts`, `goyoke-team-run`) each construct CLI args independently. Adding a `cli_flags` field (or `allowed_tools` list) to agents-index.json lets all three paths read from the same source for per-agent tool permissions and other flags.

This mitigates the "three spawn paths drift" risk identified by both analysts.

**Acceptance Criteria**:
- [ ] `agents-index.json` has `allowed_tools` (or `cli_flags`) per agent
- [ ] `spawnAgent.ts` updated to read from this field instead of hardcoding
- [ ] Go binary design (TC-008) references this field
- [ ] Adding a new tool to an agent's allowed list requires changing one file, not three

**Source**: Synthesis of Einstein's "unified config reading" and Staff-Architect's "three paths to update" concern (Convergence C-3)

---

### Phase 2: Go Binary Implementation

#### TC-004: Implement Full Daemon Pattern in Go Binary

**Priority**: HIGH
**Phase**: 2
**Blocked By**: none
**Effort**: 1 day

**Description**:
Implement proper process detachment for `goyoke-team-run`:

1. **Process group isolation**: `cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}` when the Go binary spawns child `claude` processes. The binary itself should also call `syscall.Setsid()` early in `main()`.
2. **Close stdin**: The Go binary does not need stdin after reading config.json. Close it to prevent accidental terminal interaction.
3. **Redirect stdout/stderr**: Write to `{team_dir}/runner.log` (already in the plan; formalize it as part of daemon pattern).
4. **PID file**: Write `{team_dir}/goyoke-team-run.pid` on startup. Remove on clean exit. Check on startup for double-start prevention.
5. **Clean shutdown**: Signal handler removes PID file before exit.

The launch command changes from `nohup goyoke-team-run ...` to a direct invocation with output redirection (the binary handles its own detachment).

**Acceptance Criteria**:
- [ ] Pressing Ctrl+C in TUI does NOT kill the Go binary or its children
- [ ] Closing the TUI terminal does NOT kill the Go binary or its children
- [ ] `/team-cancel` (explicit SIGTERM via PID) still works
- [ ] `goyoke-team-run` is in its own session/process group (verify with `ps -o pid,pgid,sid`)
- [ ] PID file created on startup, removed on clean exit
- [ ] Second launch attempt to same team directory fails with clear error message
- [ ] `kill -9` of binary leaves PID file (stale PID detection works on next session)

**Source**: Einstein Key Insight 3, User Decision 2 (full daemon pattern)

---

#### TC-008: Implement `goyoke-team-run` Go Binary

**Priority**: CRITICAL
**Phase**: 2
**Blocked By**: TC-001, TC-002, TC-004, TC-006, TC-009, TC-014
**Effort**: 5-7 days

**Description**:
Implement the core Go binary at `cmd/goyoke-team-run/main.go`. This is the primary Phase 2 deliverable. It incorporates the designs from TC-001 (permission flags), TC-002 (mutex), TC-004 (daemon pattern), TC-006 (projectRoot), and TC-014 (cli_flags).

Core components:
1. **Config reader**: Parse `config.json` with `TeamRunner` struct (mutex-protected per TC-002)
2. **Wave scheduler**: Execute waves sequentially, members within a wave in parallel via goroutines
3. **Process spawner** (`spawnAndWait`): Build CLI args with `--allowedTools` (per TC-001), set `cmd.Dir` to `project_root` (per TC-006), read `allowed_tools` from agents-index.json (per TC-014)
4. **Prompt envelope builder** (`buildPromptEnvelope`): Construct agent orientation prompt from stdin file
5. **Cost extraction** (`extractCostFromCLIOutput`): Parse CLI JSON output defensively (per TC-005)
6. **Stdout validation** (`validateStdout`): Check envelope fields exist
7. **Signal handler**: SIGTERM cascade to children with 5s grace period, then SIGKILL
8. **Heartbeat**: Touch heartbeat file every 30s
9. **Budget gate**: Check `budget_remaining_usd` before each spawn
10. **Daemon lifecycle**: Per TC-004 (setsid, PID file, stdin close)

Retry logic must use a for-loop, NOT recursion (fixes TC-003).

**Acceptance Criteria**:
- [ ] Binary compiles and runs: `go build ./cmd/goyoke-team-run/...`
- [ ] At least 3 successful executions with real Claude CLI
- [ ] Budget ceiling prevents runaway (test with $0.50 budget)
- [ ] SIGTERM to team runner kills all children within 10 seconds
- [ ] Heartbeat file touched regularly, stale detection works
- [ ] Agent failure -> retry once -> failure correctly marked in config.json
- [ ] Atomic config.json writes verified (no corruption on kill -9)
- [ ] `go test -race ./cmd/goyoke-team-run/...` passes with zero race warnings
- [ ] Agents can successfully use Read, Write, Glob, Grep, Bash, Edit tools
- [ ] Cost tracking accurate to within 10% of actual API spend across 3+ runs

**Source**: Implementation Plan Phase 2, all critical/high findings

---

#### TC-003: Fix Recursive Retry WaitGroup Panic

**Priority**: HIGH
**Phase**: 2 (implemented as part of TC-008)
**Blocked By**: TC-002
**Effort**: Included in TC-008

**Description**:
The plan's `spawnAndWait()` function has `defer wg.Done()` at the top. On retry, it calls itself recursively, adding a second `defer wg.Done()`. Since `wg.Add(1)` was called only once, the WaitGroup counter goes negative, causing a panic.

Fix: Replace recursive retry with a for-loop inside `spawnAndWait()`:

```go
func spawnAndWait(teamDir string, config *TeamConfig, member *Member, ...) {
    defer wg.Done()
    for attempt := 0; attempt <= member.MaxRetries; attempt++ {
        // spawn, wait, validate
        if member.Status == "completed" {
            return
        }
        // prepare for retry
        member.RetryCount = attempt + 1
        member.Status = "pending"
    }
}
```

**Acceptance Criteria**:
- [ ] Agent that fails once and succeeds on retry completes without panic
- [ ] Agent that fails all retries is correctly marked as "failed" in config.json
- [ ] WaitGroup counter never goes negative (verified by `go test -race`)

**Source**: Staff-Architect Smell 4 (HIGH)

---

#### TC-005: Verify and Document CLI JSON Output Format

**Priority**: HIGH
**Phase**: 2 (before implementing cost extraction in TC-008)
**Blocked By**: none
**Effort**: 0.5 days

**Description**:
The `extractCostFromCLIOutput()` function assumes a `cost_usd` field. The existing `spawnAgent.ts:parseCliOutput()` parses `cost_usd || total_cost_usd`, suggesting the field name is uncertain. No documentation exists for this format.

Steps:
1. Run `echo "What is 2+2?" | claude -p --output-format json` and capture raw output
2. Document the actual field names and structure
3. Update `extractCostFromCLIOutput()` design to match reality
4. Add defensive fallbacks: try multiple field names, log raw output on parse failure
5. Consider using `--max-budget-usd` flag as secondary budget enforcement

**Acceptance Criteria**:
- [ ] Actual CLI JSON output format documented with a real example
- [ ] `extractCostFromCLIOutput()` handles the real format + at least one fallback
- [ ] On parse failure, raw output is logged to `runner.log` for debugging
- [ ] Cost tracking is accurate within 10% across 3+ test runs

**Source**: Einstein Assumption 1, Staff-Architect A-2 (HIGH), both analysts converge (Convergence C-4)

---

#### TC-010: Rewrite Inter-Wave Script as Go Binary

**Priority**: HIGH
**Phase**: 2
**Blocked By**: TC-008 (needs the team-run binary to exist for integration)
**Effort**: 1-2 days

**Description**:
Rewrite `goyoke-team-prepare-synthesis.sh` as `cmd/goyoke-team-prepare-synthesis/main.go`. This eliminates the `jq` dependency and provides typed JSON parsing with proper error handling.

The Go binary:
1. Reads `stdout_einstein.json` and `stdout_staff-arch.json` from the team directory
2. Extracts key sections (root cause, challenged assumptions, novel approaches, critical failure modes, architecture smells, contractor readiness, open questions)
3. Writes `pre-synthesis.md` as a curated markdown summary
4. Handles missing/malformed JSON gracefully (same resilience as the bash `2>/dev/null || echo "(fallback)"` pattern)

Usage: `goyoke-team-prepare-synthesis <team-dir>`

The `waves.{N}.on_complete_script` field in braintrust template references this binary instead of the bash script.

**Acceptance Criteria**:
- [ ] `goyoke-team-prepare-synthesis` Go binary produces equivalent output to the shell script
- [ ] No `jq` dependency
- [ ] Handles missing stdout files gracefully (writes fallback text)
- [ ] Handles malformed JSON gracefully (writes fallback text, does not crash)
- [ ] `go build ./cmd/goyoke-team-prepare-synthesis/...` succeeds
- [ ] Integration test: runs between wave 1 and wave 2 in a real team execution

**Source**: Staff-Architect M-1, A-7; User Decision 3 (Go binary)

---

#### TC-011: Unit Tests for Go Binary

**Priority**: HIGH
**Phase**: 2 (alongside TC-008)
**Blocked By**: TC-002 (mutex needed for concurrent tests)
**Effort**: 1-2 days

**Description**:
Create table-driven unit tests for core Go binary functions. The quality gates specify integration tests but no unit tests.

Test areas:
1. `buildPromptEnvelope()` -- verify absolute paths, template expansion, `reads_from` section
2. `extractCostFromCLIOutput()` -- valid JSON, malformed JSON, missing fields, empty input
3. `validateStdout()` -- valid envelope, missing `$schema`, missing `status`, empty file, non-JSON
4. Wave ordering logic -- verify wave 2 waits for wave 1, single-wave teams work
5. Budget gate logic -- verify spawn blocked when budget exceeded
6. `TeamRunner.updateMember()` -- concurrent access with race detector

**Acceptance Criteria**:
- [ ] Table-driven tests for all 6 areas above
- [ ] `go test -race -cover ./cmd/goyoke-team-run/...` passes
- [ ] Coverage target: 80%+ for core logic (not necessarily for CLI-interfacing code)
- [ ] `go test -race -cover ./cmd/goyoke-team-prepare-synthesis/...` passes

**Source**: Staff-Architect Layer 5 (HIGH)

---

### Phase 3: Slash Commands

#### TC-012: Implement Team Slash Commands

**Priority**: HIGH
**Phase**: 3
**Blocked By**: TC-008
**Effort**: 2-3 days

**Description**:
Implement four slash commands for team management:

1. **`/team-status`**: Read config.json for each team directory in current session. Display wave progress, member statuses, cost breakdown, timing. Show active and completed teams.

2. **`/team-result`**: Read stdout file of final-wave agent. For braintrust: extract `content.executive_summary` and `content.unified_recommendations` from `stdout_beethoven.json`. For review: aggregate findings by severity across all reviewer stdout files.

3. **`/team-cancel`**: Read config.json for `background_pid`. Send SIGTERM to `goyoke-team-run` process, which cascades to children.

4. **`/teams`**: List all teams in current session with status summary.

**Session directory discovery**: Define how slash commands find the current session's team directories. Options: env var set by TUI, filesystem scan of `sessions/*/teams/`, or TUI state query. This design decision must be made as part of this ticket.

**Acceptance Criteria**:
- [ ] `/team-status` shows accurate wave progress and costs for running teams
- [ ] `/team-status` shows completed teams with final cost and duration
- [ ] `/team-result` displays Beethoven's synthesis correctly (braintrust workflow)
- [ ] `/team-result` aggregates review findings by severity (review workflow)
- [ ] `/team-cancel` gracefully stops running team within 10 seconds
- [ ] `/teams` lists all teams in current session
- [ ] Session directory discovery mechanism documented and working

**Source**: Implementation Plan Phase 3, Staff-Architect M-5

---

### Phase 4: Orchestrator Rewrites

#### TC-013: Rewrite Orchestrator Prompts for Team Pattern

**Priority**: HIGH
**Phase**: 4
**Blocked By**: TC-012
**Effort**: 3-5 days

**Description**:
Rewrite all three orchestrator workflows to use the team pattern:

1. **Mozart (`/braintrust`)** -- Interview-then-background:
   - Foreground (~30s): Interview user, scout scope, determine team composition
   - Write config.json from braintrust template
   - Write stdin files for einstein, staff-arch, beethoven (absolute paths resolved)
   - Launch `goyoke-team-run` (daemon mode)
   - Verify PID in config.json
   - Return: "Braintrust team dispatched. Use /team-status to check progress."

2. **Review-Orchestrator (`/review`)** -- Fully backgroundable:
   - Router handles directly (~5s): compute git diff, fill stdin templates for 4 reviewers
   - Write config.json from review template
   - Launch `goyoke-team-run` (daemon mode)
   - Return immediately

3. **Impl-Manager (`/ticket`)** -- Fully backgroundable:
   - Router handles directly (~10s): read specs.md, build task DAG, identify waves
   - Write config.json from implementation template with dynamic members
   - Write stdin files for each worker agent
   - Launch `goyoke-team-run` (daemon mode)
   - Return immediately

**Acceptance Criteria**:
- [ ] `/braintrust` dispatches Einstein + Staff-Architect in parallel (Wave 1)
- [ ] TUI returns to user within 30 seconds of invoking /braintrust
- [ ] Beethoven receives both outputs and synthesizes (Wave 2)
- [ ] `/review` dispatches 4 reviewers in parallel, returns in <10 seconds
- [ ] `/ticket` builds task DAG from specs.md, dispatches workers in waves
- [ ] At least 3 successful end-to-end runs for each workflow
- [ ] Cost tracking accurate to within 10% of actual API spend

**Source**: Implementation Plan Phase 4, User Decision 1 (all three templates)

---

### Independent Tickets

#### TC-015: TUI Concurrent Query Support

**Priority**: MEDIUM
**Phase**: Independent (does not block Go binary work)
**Blocked By**: none
**Effort**: 3-5 days (estimate)

**Description**:
Fix the root cause of TUI freezing during agent orchestration: the `useClaudeQuery` hook treats the streaming state as a global mutex, preventing user interaction during any active query.

**Root cause analysis (from Einstein)**:
1. `useClaudeQuery.ts` line 630: `setIsStreaming(true)` locks the UI
2. `useClaudeQuery.ts` line 730: `for await (const event of eventStream)` runs to completion
3. `useClaudeQuery.ts` line 576-578: `setIsStreaming(false)` only fires in `handleResultEvent`
4. Between lines 630 and 576, user cannot send another message (guarded by `streamingRef.current` at line 600)
5. The `spawn_agent` MCP tool blocks within the `query()` event stream

The freeze is not in the JavaScript event loop -- Node.js can still process timers and I/O callbacks. The freeze is in the application-level state machine: the TUI considers itself "streaming" and does not accept new user messages until the result event arrives.

**Potential fix direction**: Modify `useClaudeQuery` to support multiple concurrent streams with independent streaming state. Each query gets its own `isStreaming` flag. The UI accepts input during active streaming.

**Open question**: Does the Claude Agent SDK's `query()` support concurrent invocations from the same process? If the SDK enforces single-session semantics, this approach requires SDK changes.

**Acceptance Criteria**:
- [ ] User can type new messages while a query is in progress
- [ ] Multiple concurrent `query()` streams work (or documented why not)
- [ ] No race conditions in UI state management
- [ ] Existing single-query workflow still works correctly

**Source**: Einstein Root Cause Analysis, Einstein Novel Approach 1, User Decision 4 (separate ticket)

---

#### TC-016: Duplicate Launch Prevention via PID File

**Priority**: LOW
**Phase**: Post-MVP (or incorporated into TC-004 daemon pattern)
**Blocked By**: TC-008
**Effort**: 0.5 days

**Description**:
Prevent two `goyoke-team-run` instances from launching for the same team directory. If the launch command is accidentally run twice, the second instance would overwrite the first's config.json.

This is partially addressed by TC-004 (PID file as part of daemon pattern). This ticket covers the edge case of stale PID file recovery.

**Acceptance Criteria**:
- [ ] Go binary checks for existing PID file on startup
- [ ] If PID file exists and process is alive, exit with error: "Team already running (PID NNNNN)"
- [ ] If PID file exists but process is dead, clean up stale file and proceed
- [ ] PID file removed on clean exit (covered by TC-004)

**Source**: Staff-Architect F-NEW-5 (MEDIUM)

---

## 6. Consolidated Risk Register

| # | Risk | Source | Likelihood | Impact | Mitigation | Ticket |
|---|------|--------|------------|--------|------------|--------|
| R-1 | `--permission-mode delegate` insufficient | Both | HIGH (proven) | CRITICAL | Use `--allowedTools` | TC-001 |
| R-2 | Config.json data race | Staff-Arch | HIGH (every multi-member wave) | HIGH | Mutex | TC-002 |
| R-3 | Retry WaitGroup panic | Staff-Arch | HIGH (every retry) | HIGH | For-loop refactor | TC-003 |
| R-4 | SIGINT/SIGHUP propagation | Einstein | MEDIUM | MEDIUM | Full daemon pattern | TC-004 |
| R-5 | CLI JSON output format unknown/unstable | Both | MEDIUM | HIGH | Verify + defensive parsing | TC-005 |
| R-6 | `projectRoot` undefined | Staff-Arch | HIGH (blocks Phase 2) | HIGH | Add to config.json | TC-006 |
| R-7 | Agent produces empty/malformed stdout | Both | HIGH initially | MEDIUM | Envelope validation + fallback | TC-008 |
| R-8 | Orphaned processes on binary crash | Both | LOW | HIGH | Heartbeat + PID file + session cleanup | TC-004 |
| R-9 | Three spawn paths drift | Both | MEDIUM (long-term) | MEDIUM | Shared `cli_flags` in agents-index | TC-014 |
| R-10 | Concurrent team launches to same dir | Staff-Arch | LOW | HIGH | PID file lockfile semantics | TC-016 |
| R-11 | `jq` not installed | Staff-Arch | N/A (eliminated) | N/A | Rewritten as Go binary | TC-010 |
| R-12 | TUI remains frozen during orchestration | Einstein | CERTAIN (current behavior) | MEDIUM | Separate ticket; Go binary mitigates | TC-015 |

---

## 7. Revised Implementation Timeline

Reflects all four user decisions:
- All three templates: +1 day Phase 1, +2-3 days Phase 4
- Full daemon pattern: +0.5 days Phase 2 (included in TC-004)
- Go inter-wave binary: +1-2 days Phase 2
- TUI concurrency: separate, not on critical path

| Phase | Tickets | Original Estimate | Revised Estimate | Key Changes |
|-------|---------|-------------------|------------------|-------------|
| 0 | TC-001, TC-002 | 1-2 days | 1-2 days | Design decisions for permission flags and mutex (implementation in Phase 2) |
| 1 | TC-006, TC-007, TC-009, TC-014 | 2-3 days | 3-4 days | All three templates (+1 day); add `project_root`, `cli_flags`, Task() documentation |
| 2 | TC-003, TC-004, TC-005, TC-008, TC-010, TC-011 | 5-7 days | 7-9 days | Core Go binary + daemon pattern + inter-wave Go binary + unit tests |
| 3 | TC-012 | 2-3 days | 2-3 days | Slash commands (unchanged) |
| 4 | TC-013 | 3-4 days | 3-5 days | All three orchestrator rewrites (braintrust + review + implementation) |
| **Total** | | **13-19 days** | **16-23 days** | |
| Independent | TC-015 | -- | 3-5 days | TUI concurrency (not on critical path) |

---

## 8. Open Questions Requiring Investigation

These emerged from both analyses and need answers before or during implementation:

1. **Does `query()` support concurrent invocations?** (Einstein Q1) -- Required for TC-015. If the SDK enforces single-session semantics, the TUI concurrency fix needs SDK changes.

2. **Does `claude -p` respect `CLAUDE_CODE_EFFORT_LEVEL`?** (Einstein Q2, Staff-Arch A-10) -- Verify by running `CLAUDE_CODE_EFFORT_LEVEL=high claude -p "hello"` and checking behavior. Low-cost check during Phase 2.

3. **What happens on `kill -9` of `goyoke-team-run`?** (Einstein Q3) -- Children may become orphans. PID file will be stale. Heartbeat stops. Next session detects stale heartbeat and cleans up PIDs. Document this as a known limitation.

4. **How do slash commands discover the current session directory?** (Staff-Arch M-5) -- Blocks TC-012 (Phase 3). Needs an answer before Phase 3 begins. Options: env var, TUI state, filesystem scan.

5. **What is the actual cost overhead of the prompt envelope?** (Einstein Q6) -- If the envelope adds 2K tokens per agent, cumulative overhead across a 3-agent braintrust run is measurable. Benchmark during Phase 2 testing.

---

## Appendix A: Analysis Summaries

### Einstein -- Key Takeaways

- Identified the root cause as the TUI's `isStreaming` mutex, not the JS event loop
- Applied Process Supervision Theory (Erlang/OTP) and Unix Process Group Model
- Proposed three alternative approaches (TUI concurrency fix, socket daemon, extended spawn_agent)
- Strongest theoretical insight: structured I/O schemas are independently valuable and should be decoupled from the Go binary
- Surfaced 9 assumptions, proposed 4 approaches (3 alternatives + 1 synthesis)

### Staff-Architect -- Key Takeaways

- Found 2 critical bugs (permission mode, data race), 4 high issues, 5 medium, 3 low
- Verified assumptions against codebase evidence (found A-1 contradicted by PERMISSION_HANDLING.md)
- Provided concrete Go code fixes (TeamRunner struct with mutex)
- Identified 5 missing failure modes not in the plan's catalog
- Gave 5 commendations (schema design, failure catalog, atomic writes, inter-wave degradation, planner/executor separation)
- Verdict: APPROVE_WITH_CONDITIONS (fix C-1 and C-2)

### Synthesis Quality Assessment

- **Convergence**: 7 strong agreement points -- high confidence in the plan's fundamental soundness
- **Divergences resolved**: 5 divergences identified, all resolved with user decisions
- **Unresolved tensions**: 0
- **Tickets produced**: 16 (2 critical, 7 high, 5 medium, 2 low)
- **Decision points**: 4 (all resolved by user)
- **Estimated total timeline**: 16-23 days (critical path) + 3-5 days (independent TUI work)
