# Weighted Implementation Plan: Parallel Orchestration

> **Author**: Router (Opus synthesis) with 4 Haiku scouts
> **Date**: 2026-02-06
> **Inputs**: PARALLEL-ORCHESTRATION-DESIGN.md (original), analysis-parallel-orchestration-2026-02-06.md (braintrust)
> **Status**: Tentative - pending user decisions on key tradeoffs

---

## Executive Summary

After testing the four critical hypotheses raised by the braintrust analysis against the actual codebase, I conclude:

1. **The braintrust was partially right** that the original Go binary scope was over-specified, but wrong that Go itself is the wrong language - the project already has 5+ Go binaries in `cmd/`, and Go's process management primitives are purpose-built for this task
2. **The braintrust was wrong** that SDK concurrent `query()` is a simple fix - a "multiple session spawning bug" was already encountered and guarded against in the TUI code
3. **The original design was right** that background execution outside Node.js is the robust path, and right that Go is the natural home for it
4. **Both documents missed** a critical pre-existing bug (ProcessRegistry cleanup never wired to shutdown) and both misdiagnosed the root cause of TUI blocking (it's React architecture, not event loop)

---

## Part I: Hypothesis Test Results

Four Haiku scouts were dispatched to test the braintrust's key assumptions against the actual codebase. Each hypothesis result changes the decision landscape in a specific way.

### H1: SDK Concurrent `query()` Support

**What we tested**: The braintrust's #1 recommendation was "spend 2 hours testing SDK concurrent queries before any implementation decision." Einstein argued this could eliminate the need for any background execution mechanism entirely.

**What we found**: The SDK's `query()` function returns an independent `AsyncGenerator<SDKMessage>` per call. The type definitions show no singleton patterns, global locks, or shared session state. Each Query instance carries its own session ID, model selection, and MCP server connections. At the protocol level, concurrent calls appear architecturally sound.

**But then there's the red flag**. In `useClaudeQuery.ts:598-601`:

```typescript
// GUARD: Prevent concurrent queries (fixes multiple session spawning bug)
if (streamingRef.current) {
  void logger.warn("Query already in progress, ignoring duplicate call");
  return;
}
```

This guard isn't speculative. The comment says *"fixes multiple session spawning bug"* - past tense. Someone already tried concurrent queries, it broke something, and this guard was the fix. The `streamingRef` pattern (using a React ref instead of state to avoid stale closures) suggests the bug was specifically about concurrent access to shared mutable state during streaming.

**Why this matters for the decision**: The braintrust's entire Path A ("SDK concurrent queries, ~100 lines, 5 days") rests on the assumption that concurrent `query()` is a simple validation. But the evidence says it was already validated and *failed*. The question isn't "does it work?" but "why did it break and can we fix it?" That's a fundamentally different, harder question.

The braintrust's estimate of "~100 lines of TypeScript" is also off by an order of magnitude. Concurrent queries in the TUI would require:
- Separate event handlers per background query (the current `handleAssistantEvent`, `handleUserEvent`, `handleResultEvent` all mutate shared Zustand state)
- Store isolation so background queries don't inject content into the main chat
- A way to handle `canUseTool` permission callbacks for background queries (currently opens modal dialogs - impossible for headless background work)
- Separate cost tracking aggregation
- Error isolation (one query failing shouldn't corrupt the other's state)

Realistic estimate: 300-500 lines of TypeScript plus significant architectural changes to `useClaudeQuery.ts` and the store. Maybe 8-12 days of careful React work, not "~100 lines."

**Confidence levels**: HIGH that the SDK supports concurrent calls at the protocol level. MEDIUM that they can work without state corruption in the TUI. LOW that this is a 2-hour validation.

**Decision impact**: The in-process TypeScript alternative (braintrust's preferred path) carries substantially more risk than presented. Background execution outside the TUI process avoids this entire problem space.

### H2: Hook Execution in Headless/Delegate Mode

**What we tested**: Both Einstein (section 4.3) and Staff-Architect (F10) flagged "hook bypass in headless mode" as an unvalidated, HIGH-risk assumption. The concern: if `gogent-validate` doesn't run when `claude --permission-mode delegate -p` spawns agents, those agents could violate routing rules, bypass the delegation ceiling, and produce no ML telemetry.

**What we found**: Hooks execute unconditionally. The evidence is overwhelming:

1. Hooks are configured in `.claude/settings.json` at the project level, not the session level. The CLI loads this file regardless of invocation flags.
2. `gogent-validate` (the PreToolUse hook) reads events from STDIN and processes them identically in all modes. It checks no environment variables related to permission mode or pipe mode.
3. `--permission-mode delegate` affects permission *dialogs* (whether the user is prompted before tool use), not hook *execution*. These are orthogonal concerns in the Claude CLI architecture.
4. Most compellingly: `spawnAgent.ts` already invokes `claude` with both `--permission-mode delegate` and `-p` flags (lines 328-335), and the entire validation/enforcement system works through this path. If hooks didn't run in this mode, the existing spawn system would be broken - and it's not.

**Why this matters**: This was listed as a BLOCKER by both Einstein and Staff-Architect. It's not. Agents spawned by `gogent-team-run` (whether Go or any other binary) WILL have routing validation, sharp edge capture, nesting level enforcement, and ML telemetry - because those are properties of the `claude` CLI process, not the parent that spawned it.

**Decision impact**: Eliminates one of the braintrust's four blockers entirely. Significantly de-risks the Go binary approach, since the concern about "unmonitored agents" is unfounded.

### H3: Event Loop Blocking vs TUI Architecture Constraint

**What we tested**: The original design claims "Node.js single-threaded event loop cannot process UI events while awaiting LLM response." The braintrust's Einstein identified this as the root cause and proposed solving it at the concurrency layer. We tested whether the event loop is actually blocked.

**What we found**: The event loop is NOT blocked. The `for await (const event of eventStream)` pattern at `useClaudeQuery.ts:730` is an async iterator. Between each `yield` from the event stream, the Node.js event loop is free to process other microtasks, timers, and I/O callbacks. If you scheduled a `setTimeout` during a query, it would fire normally.

The actual blocker is architectural, at the React/application layer:

1. `sendMessage()` is an async function that doesn't return until the *entire* query completes (all events processed, result received). The function occupies the user interaction flow for the full duration.
2. `streamingRef` is set to `true` at the start of `sendMessage()` and only cleared on result. Any subsequent call to `sendMessage()` is rejected.
3. The UI renders in "streaming" state, which disables the input area. The user *perceives* blocking because the UI tells them they can't type, not because the process is actually stuck.

This is a crucial distinction because it changes what needs to be solved:

| Root Cause | Solution |
|-----------|----------|
| Event loop blocked (what both documents claim) | Node.js worker threads, child processes, or exit Node.js entirely |
| Single-query React architecture (what actually happens) | Allow primary + background query flows with separate state, OR exit the query() flow entirely for orchestration |

The background execution approach (Go binary or otherwise) solves the right problem almost accidentally - by having Mozart return quickly and hand off execution, the `sendMessage()` call completes in ~30 seconds and the TUI returns to interactive state. The Go binary doesn't "unblock the event loop" (it was never blocked) - it **shortens the foreground query duration** to just the planning phase.

**Decision impact**: Both the Go binary approach and the in-process TS approach would work at the event loop level. The differentiator isn't "does the event loop stay free?" (it does either way) but "how cleanly can we separate foreground interaction from background execution?" The Go binary wins on separation cleanliness - it's a completely independent process with zero coupling to the TUI's React state.

### H4: ProcessRegistry Extensibility + Critical Pre-existing Bug

**What we tested**: The braintrust identified ProcessRegistry bypass as a breaking change and recommended unification as a Phase 0 blocker. We tested how coupled ProcessRegistry is to Node.js `ChildProcess` and whether extension is feasible.

**What we found about extensibility**: ProcessRegistry is tightly coupled to `ChildProcess` (the `ProcessInfo` interface requires it, termination logic calls `process.kill()` and checks `process.killed`), but the coupling surface is small - only 2 call sites in `spawnAgent.ts`. A `ProcessKiller` interface abstraction could decouple it in 2-3 days.

More importantly, a parallel system already exists: `pidTracker.ts` handles PID-based tracking for cross-session orphan recovery. It uses `process.kill(pid, 0)` for liveness checks and `process.kill(pid, 'SIGTERM')` for termination - exactly what's needed for external PIDs. The infrastructure for tracking Go binary's children already exists in a different module.

**What we found about the bug**: ProcessRegistry has a fully implemented `cleanupAll()` method with graceful SIGTERM-then-SIGKILL escalation. The lifecycle module has a `registerChildProcessCleanup()` function designed to wire cleanup into shutdown. **But nobody ever called it.** The signal handlers in `shutdown.ts` handle session state saving, Go hook invocation, and shutdown coordination - but never call ProcessRegistry.

This means every spawned agent today - every `spawn_agent` call, every Einstein, every Staff-Architect - survives a graceful TUI exit. They become orphans. This isn't a hypothetical concern about the Go binary design; it's a production bug affecting the current system.

The dual signal handler registration compounds this: `index.tsx` registers SIGINT/SIGTERM handlers for terminal cleanup, then `shutdown.ts` registers its own handlers which overwrite the first set. The terminal cleanup (cursor restoration, alternate buffer exit) never runs on graceful shutdown.

**Decision impact**:
1. Phase 0 must fix this regardless of which path is chosen. It's a pre-existing bug.
2. The braintrust's "ProcessRegistry unification as Phase 0 BLOCKER" is partially correct but for the wrong reason - the blocker isn't unification with Go binary, it's that the existing cleanup doesn't work at all.
3. For the Go binary specifically: since pidTracker already handles cross-session orphan recovery via PID files, and the team runner will write PIDs to config.json, the orphan problem has a recovery path even without ProcessRegistry integration. Full unification is a quality-of-life improvement, not a structural requirement.

---

## Part II: What Each Document Got Right (and Why)

### Original Design: Where It Earns Its Score

**The core architectural insight is correct and well-reasoned.** The observation that orchestrators don't use Write/Edit on implementation files - they Read, Plan, Coordinate, and Synthesize - means their execution phase is mechanical coordination, not creative LLM work. The Go binary doesn't replace LLM reasoning; it replaces the *waiting* between LLM calls. This is a clean separation of concerns that both the original author and the braintrust agree on.

**The team config schema (section 3.4) is production-quality.** The braintrust dismisses this as "schema proliferation," but the config schema serves three distinct purposes that can't be eliminated:
1. **Coordination state**: Which agents are in which wave, what's their status, what do they depend on. This is the minimum viable state for a wave scheduler.
2. **Audit trail**: What was asked, what was returned, how much it cost. Without this, team execution is a black box you can't debug.
3. **Agent self-orientation**: Long-running Claude agents can and do lose track of their task during context compression. The "read your stdin file" pattern (section 7) is essentially a persistent instruction pointer that survives context window truncation.

The braintrust's Staff-Architect scored "schema proliferation" as an architecture smell, but the alternative is free-form text prompts and outputs with no machine-parseable structure. That makes inter-wave processing impossible, cost tracking impossible, and debugging a team failure a manual exercise in reading raw LLM output. The schemas aren't over-engineering; they're the difference between a debuggable system and a prayer.

**The inter-wave script pattern (section 5) is a genuine cost optimization the braintrust undervalues.** The `gogent-team-prepare-synthesis.sh` script between Wave 1 (Einstein + Staff-Architect) and Wave 2 (Beethoven) extracts key sections via `jq` and produces a curated markdown summary. This reduces Beethoven's context load from two full JSON analyses (~20K tokens) to a focused summary (~3K tokens) plus the option to Read the full files for depth. At Opus pricing, this saves ~$0.75 per braintrust invocation. Over dozens of invocations, it pays for the day it took to write the script.

**The wave-based execution model with dependency DAG is the correct abstraction.** It generalizes cleanly across all three orchestration workflows: braintrust (2 waves), review (1 wave, all parallel), and implementation (N waves, DAG from specs.md). The braintrust's TypeScript TeamManager proposes the same structure, just implemented differently.

### Original Design: Where It Overreaches

**The `gogent-team-init` binary is unnecessary.** Creating a team directory, copying schema templates, and filling in stdin files is mechanical work that either the Router LLM can do in 5 seconds or a 20-line shell function can handle. A separate Go binary with its own CLI interface, argument parsing, and config loading is overbuilt for "mkdir + cp + sed."

**The Unix socket proposal (section 4) is premature optimization.** Config.json polling via slash commands is adequate for a system where teams complete in 2-5 minutes. Real-time streaming to the TUI is a nice-to-have that adds significant complexity (socket lifecycle management, reconnection, TUI event integration) for marginal UX benefit. The original design wisely marks this as "optional, Phase 5" but then spends a full paragraph speccing it out, which sends the wrong signal about priority.

**Multiple concurrent teams support is specified but not justified.** The design assumes users will run `/braintrust` and `/review` simultaneously. In practice, the user who invoked `/braintrust` is waiting for its result before taking action; they're unlikely to also fire `/review` while the first one runs. Supporting concurrent teams is architecturally cheap (each team gets its own directory and process) but testing it is expensive. Defer until a user actually needs it.

### Braintrust: Where Its Challenges Hold Up

**The dual process management concern (Staff-Architect Layer 2) is architecturally real.** Two independent systems tracking processes - TUI's ProcessRegistry for spawn_agent-spawned children, and the Go binary's PID tracking for team-spawned children - create a split-brain problem. The `/agents` command won't show team agents. Cost aggregation splits across two sources. Shutdown cleanup follows two paths.

The braintrust is correct that this is a *structural* problem, not a detail to fix later. However, the severity depends on whether you view the Go binary's agents as "TUI children" (they should be in ProcessRegistry) or "team children" (they belong to the team, tracked in config.json). The original design implicitly takes the second view - teams are self-contained units with their own lifecycle. The braintrust insists on the first view - the TUI should own everything.

My assessment: the team-centric view is correct for background execution. A team that runs while the TUI is closed is, by definition, not a TUI child. It's an autonomous unit that the TUI can *observe* but doesn't *own*. The right integration point is the slash commands (`/team-status`, `/team-cancel`), not ProcessRegistry.

**The orphan problem (Einstein 4.1, Staff-Architect F5) is confirmed and worse than described.** The braintrust identifies orphan risk for the Go binary design. Our scout discovered that orphans already happen for *all* spawned agents because ProcessRegistry cleanup isn't wired to shutdown. The braintrust's proposed mitigations (heartbeat file, session startup cleanup, PID lockfile) are all needed - but they're needed for the current system too, not just the proposed one.

**The cost runaway concern (Einstein 4.2, Staff-Architect F8) is valid and actionable.** A team-level budget ceiling (`max_cost_usd` in config.json, enforced by the team runner before each spawn) is cheap to implement and high-value. The original design tracks costs but doesn't enforce a ceiling. The braintrust correctly identifies this as a "MEDIUM probability, HIGH impact" risk that needs a mitigation.

### Braintrust: Where Its Challenges Fall Short

**The "~100 lines of TypeScript" estimate for concurrent SDK queries (Einstein section 5) is not credible.** As detailed in H1 above, the TUI already tried concurrent queries and added a guard to prevent them. The fix involves React state architecture changes, store isolation, permission callback redesign, and error isolation. Einstein's analogy - "it's like building a separate kitchen because your stove only has one burner" - is catchy but misleading. The stove has four burners, someone tried using two at once, and the kitchen caught fire. The question isn't whether the burners exist but why they set things on fire.

**The "two languages = high maintenance" argument (Staff-Architect Layer 4) mischaracterizes the project.** The project already IS two languages. All hook binaries are Go. All TUI code is TypeScript. These aren't in tension - they serve different architectural roles. The hooks are standalone binaries that read STDIN, make a decision, and write STDOUT. The TUI is a React application with state management, rendering, and user interaction. Adding another Go binary to the `cmd/` directory is consistent with the existing pattern, not breaking it.

The braintrust's maintenance burden estimate ("HIGH: two languages, 8+ schemas, prompt engineering") double-counts the multi-language overhead. Go binary maintenance is comparable to the existing hook binary maintenance - the same toolchain, the same patterns, the same testing approach. The incremental burden is the team runner's logic, not the language it's written in.

**The ProcessRegistry unification as "Phase 0 BLOCKER" (Staff-Architect 4.2) overstates the dependency.** Full ProcessRegistry unification is valuable but not blocking. The team runner writes PIDs to config.json. The pidTracker module handles cross-session orphan recovery. `/team-cancel` can send SIGTERM to the background PID from config.json. All critical operations work without ProcessRegistry knowing about team agents. Unification would provide a nicer `/agents` view and unified cost display, but these are UX polish, not safety requirements.

---

## Part III: Where Both Documents Are Wrong or Incomplete

### 3.1 Root Cause Misdiagnosis

Both documents frame the problem as "the event loop blocks during LLM queries." The original design says this explicitly: "Node.js single-threaded event loop cannot process UI events while awaiting LLM response." The braintrust's Einstein uses this framing to argue for concurrent SDK queries as the root-cause fix.

The actual root cause is the TUI's single-query React architecture. The `for await` loop yields between events; the event loop is free. What's not free is the `sendMessage()` function call, which occupies the user-facing interaction flow. The UI disables input not because the process is stuck, but because the React component is in "streaming" state.

This matters because it means:
- The Go binary approach works not by "freeing the event loop" but by **shortening Mozart's foreground time** to just planning (~30s instead of ~5min)
- The in-process TS approach would also work (setImmediate yielding), but not because it "unblocks the event loop" - it would need to create a secondary interaction pathway alongside the main one
- The SDK concurrent query approach would require the most architectural change, because it means fundamentally restructuring how the TUI manages query state

### 3.2 The ProcessRegistry Bug Neither Found

The most impactful finding from the hypothesis tests is something neither document discusses: `ProcessRegistry.cleanupAll()` is never called on TUI exit. This isn't a theoretical concern about a proposed design; it's a bug in production code that affects every spawned agent today.

The infrastructure is 90% complete:
- `ProcessRegistry` has enterprise-grade cleanup with SIGTERM→SIGKILL escalation
- `shutdown.ts` has `registerChildProcessCleanup()` for exactly this purpose
- `pidTracker.ts` has `cleanupOrphanedProcesses()` for cross-session recovery

But nobody connected them. The signal handlers in `shutdown.ts` overwrite the ones in `index.tsx` (terminal state restoration never runs), and neither handler chain calls `cleanupAll()`.

This should be fixed before any new work, regardless of which implementation path is chosen.

### 3.3 The Go Binary's Effort Is Inflated by Language Bias

The braintrust estimates 22-35 days for the Go binary path and frames this as disproportionate. But the estimate includes 8-12 days for "Go development" and 5-8 days for "schema design + prompt engineering" that would take the same time in any language.

The actual language-specific overhead of Go vs TypeScript for this task:
- CLI argument building: ~30 lines in Go vs ~30 lines in TS (trivial either way)
- JSON parsing: `json.Unmarshal` vs `JSON.parse` (slightly more verbose in Go, not meaningfully harder)
- Process spawning: `exec.Command().Start()` vs `child_process.spawn()` (equivalent)
- Signal handling: `signal.Notify` in Go is arguably *better* than Node.js `process.on`
- PID monitoring: `cmd.Process.Wait()` in Go is equivalent to ChildProcess `close` event

The language choice adds maybe 1-2 days of overhead, not the 10+ day gap the braintrust implies. The bulk of the effort is in wave scheduling logic, config.json state management, and error handling - all of which are language-agnostic.

For a project that already has `cmd/gogent-validate/`, `cmd/gogent-sharp-edge/`, `cmd/gogent-archive/`, `cmd/gogent-load-context/`, and `cmd/gogent-agent-endstate/` - all Go binaries - adding `cmd/gogent-team-run/` is the natural extension of the existing pattern.

### 3.4 The Braintrust's TypeScript TeamManager Has a Hidden Coupling

The Staff-Architect's alternative proposal (a TypeScript `TeamManager` class running in-process) uses `spawnAgent` MCP tool calls to spawn team members. This preserves ProcessRegistry tracking and cost aggregation. It looks elegant on paper.

But there's a hidden coupling: the `spawnAgent` MCP tool is registered on the MCP server that's attached to the *current query session*. To call `spawnAgent` from a `TeamManager` running outside the query loop, you'd need to either:
1. Extract the MCP tool handler into a standalone function (breaking the MCP abstraction)
2. Start a new query session just to call MCP tools (creating a meta-orchestration layer)
3. Call `spawn('claude', ...)` directly, bypassing spawnAgent entirely (losing ProcessRegistry integration - the exact problem the TS approach was supposed to avoid)

The braintrust's proposal avoids this problem because it's pseudocode that skips the details. In practice, calling `mcp.callTool('spawn_agent', {...})` from outside an active query session would require significant plumbing that isn't accounted for in the effort estimate.

### 3.5 Neither Document Addresses the "canUseTool" Problem for Background Agents

When the TUI spawns agents via `spawnAgent`, those agents run with `--permission-mode delegate`, which auto-approves tool use. But within the TUI's own query session, there's a `canUseTool` callback (`useClaudeQuery.ts:654-716`) that opens permission modals.

If concurrent SDK queries were used for background orchestration (braintrust's Path A), the background queries would hit `canUseTool` for every tool call. You'd need a separate permission policy for background queries (auto-approve) vs foreground queries (modal prompt). This is another piece of the "~100 lines" that's actually hundreds of lines of architecture.

The Go binary approach sidesteps this entirely - each `claude` CLI process uses `--permission-mode delegate`, and there's no modal UI to integrate with.

---

## Part IV: Architecture Options with Revised Assessment

### Option A: Go Binary (`gogent-team-run`) — RECOMMENDED

**Why Go is the right language for this binary:**

| Factor | Assessment |
|--------|-----------|
| **Existing pattern** | 5+ Go binaries already in `cmd/`. Same build system, same conventions, same testing patterns. |
| **Process management** | `exec.Command`, `cmd.Process.Wait()`, `syscall.Kill` - Go's stdlib is purpose-built for subprocess management. Goroutines + WaitGroup for parallel wave execution. |
| **Long-running background** | Single binary, no runtime dependency. Predictable memory. No GC pauses affecting PID monitoring. No event loop to reason about. |
| **Signal handling** | `signal.Notify` with channel-based dispatch is cleaner than Node.js `process.on`. Explicit signal forwarding to child processes is idiomatic Go. |
| **Atomic file I/O** | `os.WriteFile` + `os.Rename` for atomic config.json updates. `os.ReadFile` + `json.Unmarshal` for config loading. No external dependencies. |

**What the binary does and doesn't do:**

The binary is a **wave scheduler**, not an LLM. It doesn't generate prompts, make routing decisions, or evaluate output quality. It:
1. Reads a team config that was already populated by a planning LLM
2. For each wave, spawns `claude` CLI processes in parallel
3. Waits for all processes in the wave to exit
4. Collects their JSON output, updates config.json, tracks costs
5. Optionally runs an inter-wave bash script (jq extraction)
6. Advances to the next wave
7. Exits when all waves complete

This is exactly the kind of mechanical, process-heavy, I/O-centric work Go was designed for. The entire binary is probably 400-600 lines of Go - comparable to `gogent-validate` in complexity.

**Effort estimate**: 5-7 days for the binary itself, tested with mock CLI. This is less than the braintrust's 8-12 estimate because it excludes the ProcessRegistry unification they bundled in (deferred) and the "missing specifications" phase (handled by the schema design phase).

### Option B: In-Process TypeScript TeamManager — HIGHER RISK

**The appeal**: No new binary, ProcessRegistry tracks everything, cost tracking unified, zero orphan risk.

**The problems**:
1. The "multiple session spawning bug" hasn't been diagnosed. It could be a trivial React state issue (fixable in a day) or a fundamental SDK limitation (weeks to work around).
2. Calling `spawnAgent` from outside an active query session requires MCP plumbing that doesn't exist.
3. The `canUseTool` callback creates a permission model conflict for background queries.
4. Event stream handling needs isolation between foreground and background queries.

**When this becomes viable**: If someone diagnoses and fixes the session spawning bug, and if the MCP tool invocation can be decoupled from the query session. These are both "if" conditions with unknown effort.

### Option C: Background Node.js Script — VIABLE BUT MISALIGNED

**The appeal**: Same language as TUI, JSON is native, can reuse TS utilities.

**The problems**:
1. Requires Node.js runtime in PATH for background execution (Go binary is self-contained)
2. Doesn't match the project's existing `cmd/` pattern for standalone binaries
3. Process management in Node.js is adequate but less ergonomic than Go for this specific use case
4. A .mjs script in a Go-heavy `cmd/` directory is architecturally inconsistent

**When to choose this**: If Go build tooling isn't available or if the team prioritizes TypeScript familiarity over architectural consistency. Not recommended for this project.

---

## Part V: Weighted Scoring Matrix (Revised)

Scoring updated to reflect Go binary de-risking (hooks confirmed, effort deflated) and TypeScript risk inflation (SDK bug, MCP coupling).

| Criterion (Weight) | Option A: Go Binary | Option B: In-Process TS | Option C: Node.js Script |
|---|---|---|---|
| **Implementation effort (25%)** | 14-19 days (7/10) | 13-20 days, HIGH variance (5/10) | 14-19 days (7/10) |
| **Risk profile (25%)** | Medium - known patterns (7/10) | High - SDK bug unknown (4/10) | Medium (7/10) |
| **TUI compatibility (15%)** | Good - config.json IPC, slash commands (7/10) | Excellent - all in-process (9/10) | Good - same as Go (7/10) |
| **Maintenance burden (15%)** | Low - matches existing cmd/ pattern (7/10) | Low - single codebase (9/10) | Medium - inconsistent with cmd/ (5/10) |
| **Future extensibility (10%)** | Excellent - full control, cross-platform (9/10) | Limited - tied to SDK (5/10) | Good (7/10) |
| **Orphan resilience (10%)** | Medium - heartbeat + pidTracker (6/10) | Excellent - TUI owns all (9/10) | Medium - same as Go (6/10) |
| **WEIGHTED SCORE** | **7.05** | **6.05** | **6.65** |

The Go binary scores highest when accounting for architectural consistency with the existing project. The TypeScript in-process option scores highest on theoretical elegance but is penalized for the unknown SDK bug risk.

---

## Part VI: Recommended Implementation Phases (Go Binary Path)

### Phase 0: Foundation Fixes (1-2 days)

**Wire ProcessRegistry cleanup to TUI shutdown. Fix signal handler registration.**

This fixes a pre-existing bug affecting all spawned agents today. It's needed regardless of which path is chosen and should be done first.

```typescript
// One-line fix in TUI initialization
registerChildProcessCleanup(async () => {
  const registry = getProcessRegistry();
  await registry.cleanupAll();
});
```

The dual signal handler registration also needs fixing - `shutdown.ts` overwrites `index.tsx` handlers, meaning terminal state cleanup (cursor, alternate buffer) never runs.

### Phase 1: Team Config Schema + Directory Structure (2-3 days)

**Design the team configuration schema and directory conventions.**

The original design's schema (section 3.4) is well-designed. Simplifications:
- Drop `agent_id` UUID per member (file-based coordination doesn't need it)
- Drop explicit `output_format` field (use convention: `stdout_{name}.json`)
- Add `max_cost_usd` budget ceiling (braintrust recommendation)
- Add `heartbeat_file` path for orphan detection (braintrust recommendation)

Also create default team templates for braintrust, review, and implementation workflows. Start with minimal stdin/stdout schemas (generic structure with agent-specific sections) and evolve after real usage.

### Phase 2: `gogent-team-run` Go Binary (5-7 days)

**The core background execution engine.**

The binary reads a team config, executes waves of `claude` CLI processes, and collects results. Key implementation details drawn from the original design:

- Read `agents-index.json` for model + effortLevel per agent (same as `spawnAgent.ts`)
- Set `CLAUDE_CODE_EFFORT_LEVEL`, `GOGENT_NESTING_LEVEL`, `GOGENT_PARENT_AGENT` env vars
- Build CLI args: `claude -p --output-format json --model {model} --permission-mode delegate`
- Spawn all tasks in current wave via `exec.Command`, wait via `cmd.Process.Wait()`
- Parse JSON stdout for cost_usd, collect to `stdout_{agent}.json`
- Update config.json atomically (write temp, rename) after each completion
- Run optional inter-wave script (`on_complete_script` from wave config)
- Touch heartbeat file every 30s; self-terminate if heartbeat goes stale (orphan detection)
- Forward SIGTERM/SIGINT to all children, wait 5s grace, SIGKILL stragglers
- Enforce budget ceiling: check cumulative cost before each spawn, abort if exceeded

### Phase 3: Slash Commands (2-3 days)

**User-facing team monitoring via skill definitions.**

`/team-status` reads config.json and formats wave/member status. `/team-result` reads the final agent's stdout file and displays the executive summary. `/team-cancel` sends SIGTERM to the background PID from config.json. `/teams` lists the session's teams directory.

These are all read-only operations against the filesystem. Low complexity, low risk.

### Phase 4: Orchestrator Prompt Rewrites (3-4 days)

**Migrate orchestrators to the team pattern.**

Priority order by value:
1. **Mozart** (/braintrust) - highest value. Foreground: interview + plan (~30s). Background: `gogent-team-run`.
2. **Review-orchestrator** (/review) - fully backgroundable. Router reads git diff, creates team config, launches `gogent-team-run` directly. No foreground LLM needed.
3. **Impl-manager** (/ticket) - fully backgroundable. Router reads specs.md, builds task DAG, launches `gogent-team-run`.

### Phase 5: Hardening (Optional, deferred)

- ProcessRegistry extension for external PIDs (unified `/agents` view)
- Concurrent teams support testing
- SDK concurrent query investigation (may enable future in-process option)
- Status bar integration showing active background teams
- Real-time progress streaming (Unix socket or filesystem watch)

---

## Part VII: Risk Assessment (Consolidated)

### Risks Resolved by Hypothesis Testing

| Risk | Source | Resolution |
|------|--------|-----------|
| Hook bypass in headless mode | Einstein 4.3, Staff-Arch F10 | **ELIMINATED**: Hooks run unconditionally in delegate mode |
| Event loop permanently blocked | Original Design §1 | **CORRECTED**: Architecture constraint, not event loop. Background execution shortens foreground time. |
| ProcessRegistry unification as BLOCKER | Staff-Arch recommendation | **DOWNGRADED**: pidTracker covers orphan recovery. Unification is UX polish, not safety requirement. |

### Risks Remaining

| Risk | P | I | Mitigation | Source |
|------|---|---|-----------|--------|
| Go binary crash leaves orphans | MED | MED | Heartbeat file expiry → children self-terminate. pidTracker recovery on next session. | Einstein 4.1 |
| Agents don't comply with stdout schema | HIGH initially | MED | Validate output, fall back to raw text. Iterate prompts over 2-3 runs. | Staff-Arch F7 |
| Cost runaway in background team | MED | HIGH | Budget ceiling in config.json enforced before each spawn. | Einstein 4.2 |
| CLI JSON output format changes | LOW | HIGH | Pin claude CLI version. Parse defensively with fallbacks. | Staff-Arch Layer 2 |
| Config.json corruption on binary crash | LOW | HIGH | Atomic writes (temp + rename). JSON parse error recovery in slash commands. | Einstein 3.1 |
| User forgets about background team | MED | LOW | `/team-status` reminder. Future: status bar indicator. | Original Design §8 |

### Risks Accepted

| Risk | Why Accepted |
|------|-------------|
| ProcessRegistry doesn't track team agents | pidTracker covers orphan recovery. `/team-status` provides visibility. Full unification deferred to Phase 5. |
| No real-time progress streaming | Config.json polling via `/team-status` is sufficient for 2-5 minute team durations. |
| Single team at a time (initially) | Concurrent teams are architecturally supported (separate dirs + processes) but not tested until Phase 5. |

---

## Part VIII: Decision Points for User

### Decision 1: Go Binary vs Node.js Script

The analysis above recommends Go. The arguments: architectural consistency with 5+ existing Go binaries in `cmd/`, superior process management primitives, self-contained binary with no runtime dependency, and the language-specific effort overhead is 1-2 days, not the 10+ the braintrust implied.

If you disagree and prefer Node.js: the architecture doesn't change, only the implementation language of Phase 2. Effort is comparable.

### Decision 2: Structured I/O Schemas - Full or Minimal Start

**Full** (8 stdin + 8 stdout schemas, versioned): Strict contracts from day 1. Enables jq tooling and validation. 3-4 days of schema design before binary work begins.

**Minimal** (generic schema with agent-specific fields): Ship faster, learn what structure is actually needed from real usage. 1-2 days. Evolve to full schemas after patterns emerge.

Recommendation: Start minimal. The original design's schemas are well-designed but speculative - they haven't been tested against real agent output. Two or three real braintrust runs will reveal which fields agents actually populate, which they ignore, and which are missing.

### Decision 3: ProcessRegistry Unification Timing

**Now** (Phase 0, 2-3 extra days): Unified view in `/agents`, unified cost display, unified shutdown cleanup.

**Defer** (Phase 5, when needed): Ship the team system faster. Use pidTracker for safety. Accept split visibility until it becomes a real pain point.

Recommendation: Defer. The pre-existing cleanup bug (Phase 0 fix) handles the safety concern. ProcessRegistry unification is about UX, not correctness.

---

## Part IX: What to Keep from Each Document

### From Original Design (KEEP)

| Item | Section | Why Keep |
|------|---------|---------|
| Team config JSON schema | 3.4 | Well-designed coordination state. Simplify slightly (drop UUIDs, add budget ceiling). |
| Directory structure | 3.3 | Essential for auditability. Team execution without file artifacts is a black box. |
| Wave-based execution | 4, 9 | Correct abstraction. Generalizes across braintrust (2 wave), review (1 wave), impl (N wave). |
| Agent self-orientation | 7 | Production-critical. Agents DO lose context during long runs. Re-read-your-stdin is reliable recovery. |
| Inter-wave scripts | 5 | Real cost optimization. jq extraction between waves reduces downstream context and cost. |
| `/team-status` output format | 8 | Good UX design. Clear, scannable, informative. |
| Orchestrator backgroundability profiles | 3.2 | Correct classification: Mozart needs foreground interview, review/impl are fully backgroundable. |
| Cost tracking per team | 3.4 | Non-negotiable for background execution. Must know what background work costs. |
| Signal handling spec | 4 | SIGTERM → forward to children → wait → SIGKILL stragglers. Standard, correct. |
| effortLevel integration | 4 | Already implemented in agents-index.json. Binary reads it same way spawnAgent.ts does. |

### From Original Design (DROP or DEFER)

| Item | Section | Why Drop/Defer |
|------|---------|---------------|
| `gogent-team-init` binary | 6 | Overbuilt. mkdir + cp + template fill is 20 lines of shell or router LLM work. |
| Unix socket TUI integration | 4 | Premature. Config.json polling is adequate for 2-5 minute team durations. |
| Multiple concurrent teams | 5 | Untested luxury. Ship single-team first. Architecture supports concurrency by default (separate dirs). |
| Full SIGUSR1/SIGUSR2 spec | 4 | Nice for debugging but not MVP. SIGTERM/SIGINT forwarding is sufficient. |

### From Braintrust Analysis (KEEP)

| Item | Source | Why Keep |
|------|--------|---------|
| ProcessRegistry cleanup wiring | H4 finding | Pre-existing bug. Fix regardless of path chosen. |
| Budget ceiling enforcement | Einstein 4.2 | Cheap to implement, high safety value for background execution. |
| Orphan detection protocol | Einstein 4.1 | Heartbeat file + pidTracker recovery covers Go binary and TUI crash scenarios. |
| Atomic config.json writes | Einstein 3.1 | Write-to-temp-then-rename prevents corruption on crash. Standard Go pattern. |
| Failure mode catalog (F1-F10) | Staff-Arch Layer 3 | Valuable reference for error handling design in the binary. |
| Contractor readiness checklist | Staff-Arch Layer 7 | Good acceptance criteria framework. Adapt for ticket definitions. |
| Agent timeout handling | Staff-Arch F2 | Original design specifies no per-agent timeout. Add configurable timeout per spawn. |

### From Braintrust Analysis (REVISE or DROP)

| Item | Source | Why Revise/Drop |
|------|--------|----------------|
| "~100 lines of TypeScript" TeamManager | Einstein 5 | Underestimated 5-10x. SDK bug, MCP coupling, permission callbacks, state isolation not accounted for. |
| "2 hour SDK validation" as blocker | Beethoven Rec 1 | Valid as a PASS/FAIL test but don't block on it. Go binary path doesn't depend on the answer. |
| Hook bypass as BLOCKER | Einstein 4.3 | Resolved: hooks run in headless mode. Evidence from code + existing spawnAgent usage. |
| ProcessRegistry unification as Phase 0 BLOCKER | Staff-Arch Rec 3 | Overstated dependency. pidTracker handles safety. Unification is UX polish. Defer to Phase 5. |
| "Two languages = high maintenance" | Staff-Arch Layer 4 | Project already is two languages. Go binary matches existing cmd/ pattern. Incremental burden is minimal. |
| 22-35 day effort estimate for Go path | Staff-Arch Layer 4 | Inflated by bundling language-agnostic work (schemas, prompts, slash commands) into "Go binary cost." |

---

## Part X: Success Criteria

### MVP (End of Phase 2)

- [ ] `/braintrust` dispatches Einstein + Staff-Architect in parallel (Wave 1)
- [ ] TUI returns to user within 30 seconds of invoking /braintrust
- [ ] Beethoven receives both outputs and synthesizes (Wave 2)
- [ ] Team config.json updated in real-time with status, costs, PIDs
- [ ] Budget ceiling prevents runaway costs (configurable, default $15)
- [ ] SIGTERM to team runner kills all children within 10 seconds
- [ ] Heartbeat file enables orphan detection (stale after 60s → self-terminate)
- [ ] At least 3 successful /braintrust runs with team pattern

### Full Feature (End of Phase 4)

- [ ] All three orchestrators (/braintrust, /review, /ticket) use team pattern
- [ ] `/team-status` shows progress of all active teams
- [ ] `/team-result` displays synthesis output with executive summary
- [ ] `/team-cancel` gracefully stops a running team
- [ ] Cost tracking accurate to within 10% of actual API spend
- [ ] Review workflow completes 40-60% faster than sequential (4 parallel reviewers)

### Quality Gates Between Phases

- Phase 0 → 1: Verify spawned agent cleanup on graceful TUI exit (manual test)
- Phase 1 → 2: Schema validated by hand-filling a braintrust team config + stdin files
- Phase 2 → 3: At least 3 successful `gogent-team-run` executions with real Claude CLI
- Phase 3 → 4: `/team-status` and `/team-result` verified against real team output
- Phase 4 → 5: All three orchestrators running successfully for 1 week

---

## Appendix A: Hypothesis Test Evidence Summary

| Hypothesis | Scout | Key Files | Finding | Decision Impact |
|------------|-------|-----------|---------|----------------|
| H1: SDK concurrent queries | Haiku Scout 1 | `sdk.d.ts`, `useClaudeQuery.ts:598` | AsyncGenerator supports concurrency; TUI guards against it due to prior bug | In-process TS path riskier than braintrust claimed |
| H2: Hooks in headless mode | Haiku Scout 2 | `settings.json:29-102`, `main.go:84` | Hooks execute unconditionally in delegate mode | Eliminates braintrust BLOCKER. De-risks Go binary. |
| H3: Event loop blocking | Haiku Scout 3 + code analysis | `useClaudeQuery.ts:730`, `shutdown.ts` | React architecture constraint, not event loop. ProcessRegistry cleanup unwired. | Root cause is different than claimed. Background execution works for the right reason. |
| H4: ProcessRegistry extension | Haiku Scout 4 | `processRegistry.ts`, `pidTracker.ts` | Extensible (2 call sites). pidTracker already handles external PIDs. | Unification feasible but not blocking. Pre-existing cleanup bug discovered. |

## Appendix B: Revised Effort Comparison

| Path | Effort | Risk | Language Match | Score |
|------|--------|------|---------------|-------|
| **Go Binary (Recommended)** | 14-19 days | Medium | cmd/ pattern | **7.05** |
| Node.js Script | 14-19 days | Medium | TUI language | 6.65 |
| In-Process TypeScript | 13-20 days | HIGH (SDK bug) | Best fit | 6.05 |
| Original Design (as-written) | 22-35 days | High | cmd/ pattern | ~5.0 |
| Braintrust Path A (SDK concurrent) | 11-16 days | UNKNOWN | TUI language | Unscored |
| Braintrust Path B (TS TeamManager) | 16-24 days | Medium-High | TUI language | ~5.5 |

## Appendix C: Original Design Score Revision

The braintrust scored the original design 5.3/10. After hypothesis testing:

| Dimension | Braintrust | Revised | Reason |
|-----------|-----------|---------|--------|
| Problem understanding | 9/10 | 9/10 | Correctly identifies UX issue |
| Solution completeness | 5/10 | 6/10 | Hook concern resolved. Cost tracking included. |
| Implementation detail | 7/10 | 7/10 | Good for happy path, still weak on errors |
| Risk identification | 4/10 | 5/10 | Some risks overstated by braintrust |
| Alternative consideration | 3/10 | 4/10 | Go binary has more merit than braintrust credited |
| Contractor readiness | 4/10 | 4/10 | Still missing failure mode handling specs |

**Revised Overall**: 5.8/10 (up from 5.3). The original design is better than the braintrust gave it credit for, but the gaps in error handling and orphan management are real.
