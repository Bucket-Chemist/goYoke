# Einstein Theoretical Analysis: Background Team Orchestration

> **Problem Brief**: `/home/doktersmol/Documents/goYoke/tickets/team-coordination/IMPLEMENTATION-PLAN.md`
> **Analysis Focus**: First principles decomposition, root cause analysis, challenged assumptions, novel approaches
> **Timestamp**: 2026-02-06T12:00:00Z

---

## Executive Summary

The implementation plan proposes a fundamentally sound architectural move -- separating orchestration planning (LLM, foreground) from orchestration execution (Go binary, background) -- but rests on several assumptions that deserve scrutiny. The deepest insight is that the plan introduces a **third process supervision topology** into a system that already has two, and the long-term cost of this triplication may exceed the cost of fixing the root cause: that the SDK `query()` async iterator blocks the React render loop not because of a fundamental limitation, but because of how the TUI consumes events. The Go binary is an effective bypass of this architectural constraint, but it is worth examining whether it is the *right* bypass.

---

## Root Cause Analysis

### Surface Problem

The TUI freezes for ~5 minutes during multi-agent orchestration workflows (braintrust, review, implementation).

### Underlying Cause

The `useClaudeQuery` hook (line 730: `for await (const event of eventStream)`) consumes the SDK event stream synchronously within a React render callback. While `for await` does yield between iterations (it is cooperative, not blocking), the practical effect is that:

1. The `sendMessage` function does not return until the full query completes or errors
2. The TUI's `isStreaming` state locks out user input during the entire query
3. When an orchestrator (Mozart) spawns subagents, each spawn runs a nested `claude -p` process that can take 2+ minutes, and the orchestrator's own `query()` session waits for those spawn completions before yielding the next event

The freeze is not in the JavaScript event loop itself -- Node.js can still process timers and I/O callbacks. The freeze is in the **application-level state machine**: the TUI considers itself "streaming" and does not accept new user messages until the result event arrives.

### Fundamental Issue

The system conflates **query execution** with **user interaction lock**. There is no mechanism to have an active query running while simultaneously accepting new user input. This is an application-level design choice, not a platform limitation. The `query()` function returns an async iterator that could be consumed in the background while the UI remains interactive -- but the TUI's state model does not support this.

### Evidence Chain

1. `useClaudeQuery.ts` line 630: `setIsStreaming(true)` locks the UI
2. `useClaudeQuery.ts` line 730: `for await (const event of eventStream)` runs to completion
3. `useClaudeQuery.ts` line 576-578: `setIsStreaming(false)` only fires in the `handleResultEvent` callback
4. Between lines 630 and 576, the user cannot send another message (guarded by `streamingRef.current` check at line 600)
5. The `spawn_agent` MCP tool (line 176-317) blocks within the `query()` event stream -- it returns a promise that resolves only when the child process completes

Therefore: the TUI does not freeze because the event loop is blocked. It freezes because the **application state model** treats "Claude is thinking" as a mutex on user interaction.

---

## Conceptual Framework

### Primary Lens: Process Supervision Theory (Erlang/OTP Model)

The goYoke system now has three distinct process supervision topologies:

| Topology | Supervisor | Children | Lifecycle Binding | Restart Strategy |
|----------|-----------|----------|-------------------|-----------------|
| **TUI-native** (ProcessRegistry) | TUI Node.js process | `claude` CLI via `spawn()` | Bound to TUI process | SIGTERM cascade on TUI exit |
| **MCP spawn_agent** | `spawnAgent.ts` within TUI | `claude` CLI via `spawn()` | Bound to TUI process via registry | SIGTERM + timeout + SIGKILL |
| **goyoke-team-run** (proposed) | Go binary (detached) | `claude -p` via `exec.Command` | **Intentionally unbound** from TUI | SIGTERM cascade on SIGTERM; heartbeat for orphan detection |

In Erlang/OTP terms, the first two are **linked processes** (crash propagation) while the third is a **detached supervisor** (independent lifecycle). This is a correct decomposition for the stated goal -- background execution should survive TUI restarts -- but it creates a **split-brain problem** for process accounting:

**Key Insight 1**: Two systems now track `claude` processes. The ProcessRegistry tracks TUI-spawned processes. `goyoke-team-run` tracks its own children via config.json PIDs. Neither knows about the other's children. On TUI shutdown, ProcessRegistry cleans up its children but has no knowledge of `goyoke-team-run`'s children (which are intentionally kept alive). On `goyoke-team-run` crash, its children become orphans that the TUI's ProcessRegistry cannot clean up (it never registered them).

**Key Insight 2**: The heartbeat file is the only bridge between these two supervision domains. If the Go binary crashes, the heartbeat goes stale, and "the next TUI session start" can detect orphans. But this means orphan detection is **eventually consistent with a granularity of "next session start"** -- potentially hours or never if the user does not start a new session.

### Alternative Lens: Unix Process Group Model

The plan uses `nohup` to detach `goyoke-team-run` from the TUI's process group. This is the correct Unix primitive for "survive parent death," but it has implications:

1. `nohup` does not create a new process group. The Go binary inherits the TUI's process group unless it explicitly calls `setsid()` or `setpgid()`.
2. If the Go binary inherits the TUI's process group, `kill -TERM -<pgid>` (sent by the shell on job control operations) will hit both the TUI and the Go binary.
3. The Go binary's children (claude processes) inherit the Go binary's process group, which may or may not be the TUI's.
4. `Ctrl+C` in the TUI terminal sends SIGINT to the **foreground process group**, which may include the Go binary if it inherited the group.

The plan's signal handling (section 2.4) assumes the Go binary receives SIGTERM explicitly via `/team-cancel`. It does not account for SIGINT propagation through process groups, or for the scenario where the TUI terminal is closed (SIGHUP) while the Go binary is still running.

**Key Insight 3**: The `nohup` command handles SIGHUP but not SIGINT. If the user presses Ctrl+C in the TUI while a team is running, the SIGINT may propagate to the Go binary through the process group, causing an unintended team cancellation.

---

## First Principles Analysis

### Starting Axioms

1. **Axiom: LLMs are expensive for coordination work.** An LLM waiting for a subprocess to exit costs tokens (at minimum, the context window is kept warm; in practice, the LLM session holds context that accrues cost on the next turn). Therefore, separating "plan" from "execute" is economically rational.

2. **Axiom: Interactive applications must not block on I/O waits.** A TUI that freezes for 5 minutes provides negative value (worse than no TUI at all, because the user could run CLI commands directly). Therefore, long-running background work must be decoupled from the UI event loop.

3. **Axiom: Process supervision requires a single source of truth.** When multiple supervisors manage overlapping process sets, coordination failures produce orphans, zombie processes, or split-brain state. Every process should have exactly one supervisor.

4. **Axiom: Structured contracts reduce integration fragility.** When agents communicate through defined JSON schemas rather than free-form text, downstream consumers can parse programmatically, validate structurally, and fail explicitly rather than silently.

5. **Axiom: Prompting is not a contract enforcement mechanism.** An LLM instructed to "write JSON matching this schema" will do so most of the time but not all of the time. The failure mode is silent -- the agent exits 0 but the output is malformed or missing fields. This is fundamentally different from a compiled interface where type violations are caught at build time.

### Derived Implications

**From Axiom 1**: The three-phase pattern (PLAN foreground, EXECUTE background, DELIVER on-demand) is correctly derived. The planning phase requires LLM intelligence; the execution phase is mechanical coordination; delivery is reading files. Only the first phase should burn LLM tokens.

**From Axiom 2**: The Go binary approach correctly addresses this. However, Axiom 2 does not require a *separate process*. It requires decoupling the background work from the UI's interaction state machine. An alternative satisfying the same axiom: make the TUI's query consumption non-blocking (multiple concurrent `query()` streams, UI accepts input during streaming).

**From Axiom 3**: The plan violates this axiom by design. There are now two supervisors for `claude` processes: ProcessRegistry (TUI-bound) and `goyoke-team-run` (detached). The plan acknowledges this ("dual process management" architecture smell in the stdout schema example) but defers unification to "Phase 5." This is a technical debt decision, not a principled one.

**From Axiom 4**: The structured I/O schemas are the strongest part of the plan. They transform agents from opaque text-in/text-out boxes into components with defined interfaces. This unlocks jq-based tooling, programmatic validation, and inter-wave preprocessing. This value exists independent of whether the execution engine is a Go binary, a shell script, or a modified TUI.

**From Axiom 5**: The plan's stdout validation (section 1D, `validateStdout()`) checks only the envelope (`$schema`, `status` fields present). Content validation is "prompt engineering's job." This is honest but creates a failure mode: agents that write syntactically valid JSON with semantically empty content (e.g., all fields are empty strings) will pass validation. The downstream consumer (Beethoven) then receives garbage and produces garbage. The plan's retry logic (retry once on failure) does not help with this failure mode because the agent "succeeded" from the scheduler's perspective.

### Novel Conclusions

**Conclusion 1: The structured I/O schemas should be extracted as an independent, reusable primitive**, not coupled to the Go binary. They add value to the existing `spawn_agent` path too. An agent spawned via MCP `spawn_agent` could also be instructed to write stdout JSON to a file. This would give the current (non-background) workflow the same structured output, audit trail, and programmatic validation benefits.

**Conclusion 2: The Go binary is a workaround for the TUI's concurrency model, not a fundamental architectural necessity.** If the TUI supported multiple concurrent `query()` streams with independent state tracking, the same parallelism could be achieved within the existing process topology. The Go binary adds value through process detachment (survives TUI exit) and cost savings (no LLM tokens for coordination), but the parallelism benefit alone does not justify a new supervision topology.

**Conclusion 3: The `nohup` detachment strategy is fragile for production use.** A more robust approach would be a proper daemon pattern: the Go binary forks, calls `setsid()` to create a new session, closes inherited file descriptors, and writes its PID to a known location. This is what real process supervisors (systemd, supervisord) do. The `nohup` command is a development convenience, not an operational primitive.

---

## Novel Approaches

### Approach 1: TUI Concurrency Fix (Minimal Architecture Change)

**Concept**: Instead of introducing a third process topology, fix the root cause -- the TUI's inability to handle concurrent queries.

The `useClaudeQuery` hook currently treats the streaming state as a global mutex. Modifying it to support multiple concurrent streams would allow:
- `/braintrust` starts a query that spawns Mozart
- Mozart's `spawn_agent` calls run as child processes within the same TUI process
- User can continue interacting (start another query, run `/review`, etc.)
- Each query gets its own streaming state, visible in the UI

**Rationale**: This eliminates the need for a separate Go binary for parallelism. It does not address the "survive TUI exit" requirement, but that requirement should be questioned -- how often does a user actually close the TUI during a 5-minute orchestration?

**Theoretical Tradeoffs**:
- Gains: No new process topology; no split-brain supervision; no file-based IPC; simpler system
- Losses: No TUI detachment (work dies with TUI); more complex TUI state management; does not save LLM tokens for coordination

### Approach 2: Socket-Based Supervisor (Proper Daemon Pattern)

**Concept**: Instead of `nohup` with file-based IPC, implement `goyoke-team-run` as a proper daemon with a Unix domain socket for real-time communication.

The Go binary:
- Starts as a daemon (fork, setsid, close fds)
- Listens on `{team_dir}/supervisor.sock`
- Accepts connections from TUI for status queries, cancellation, and live streaming of events
- Writes config.json as a persistence layer (survives daemon crash), not as the primary IPC mechanism

The TUI:
- Connects to `supervisor.sock` when user runs `/team-status`
- Gets real-time event stream (process started, process completed, wave advanced)
- Can send commands (cancel, abort)

**Rationale**: File-based IPC (polling config.json) has inherent latency and race conditions. Socket-based IPC provides real-time bidirectional communication. This is how Docker communicates with containerd, how systemd communicates with services, and how language servers communicate with editors.

**Theoretical Tradeoffs**:
- Gains: Real-time status updates; proper daemon lifecycle; bidirectional communication; no polling
- Losses: Higher implementation complexity; new dependency (Unix sockets in Go); more moving parts
- Assessment: Probably overengineered for the current use case but architecturally superior

### Approach 3: Extend MCP spawn_agent with Detach Mode

**Concept**: Instead of a new Go binary, extend the existing `spawn_agent` MCP tool with a `detach: true` option that:
- Spawns the `claude` CLI process in a new session (setsid)
- Returns immediately with the PID and a team directory path
- Writes the child's PID to a file for later management
- Does NOT wait for the process to complete

The wave scheduling logic could live in a simple shell script (`goyoke-wave-run.sh`) that:
- Reads config.json
- For each wave: spawns all members, waits for PIDs, runs inter-wave scripts
- Updates config.json between waves

**Rationale**: This keeps the spawning mechanism unified (one path, one code path, one validation path) while adding detachment as a capability rather than a separate system. The shell script for wave scheduling is arguably simpler than a Go binary for what is fundamentally sequential-within-wave, parallel-across-members logic.

**Theoretical Tradeoffs**:
- Gains: Unified spawning path; no new binary; reuses existing validation and cost tracking
- Losses: Shell scripts are fragile for PID management; no typed error handling; harder to test
- Assessment: Attractive for simplicity but may hit the limits of shell scripting for process management

### Synthesis Approach

The strongest design combines elements:

1. **From Approach 1**: Fix the TUI's concurrent query support regardless -- this is a UX deficiency that should be addressed independently of background orchestration.

2. **From the original plan**: Keep the Go binary for background execution and structured I/O. The Go binary provides genuine value for: detached execution, typed process management, atomic config writes, and cost accounting.

3. **From Approach 2**: Use a Unix domain socket for real-time status instead of polling config.json. This can be Phase 5 work -- start with file-based IPC, upgrade to sockets when the polling becomes inadequate.

4. **From Approach 3**: Unify the config reading. `goyoke-team-run` should read `agents-index.json` through the same parsed types as `spawnAgent.ts`. Consider generating Go types from the JSON schema to prevent drift.

5. **Critical addition not in any approach**: Implement proper daemon lifecycle (setsid, PID file, clean shutdown) instead of `nohup`. This prevents SIGINT propagation and ensures clean process group isolation.

---

## Theoretical Tradeoffs

| Dimension | Go Binary (Plan) | TUI Concurrency Fix | Socket Daemon | Extended spawn_agent |
|-----------|------------------|---------------------|---------------|---------------------|
| Parallelism | Full (OS-level) | Full (async) | Full (OS-level) | Full (OS-level) |
| TUI Detachment | Yes (nohup) | No | Yes (proper daemon) | Partial (detached child) |
| Supervision Unity | Violated (dual) | Preserved (single) | Violated but managed | Preserved (single) |
| Implementation Cost | High (new binary) | Medium (refactor hook) | Very High (daemon + socket) | Low-Medium (extend existing) |
| LLM Token Savings | Yes (no LLM for coordination) | No (orchestrator still runs) | Yes | Partial (scheduler is shell) |
| Process Group Safety | Fragile (nohup) | N/A | Strong (setsid) | Depends on implementation |
| Schema Value | Bundled with binary | Independent primitive | Bundled with daemon | Independent primitive |
| Failure Detection | Heartbeat (eventual) | In-process (immediate) | Socket disconnect (immediate) | Heartbeat or poll |

---

## Assumptions Surfaced

| Assumption | Confidence | Impact if Wrong |
|------------|------------|-----------------|
| `claude -p --output-format json` produces stable, parseable JSON output with cost_usd field | Medium | Cost tracking breaks silently; Go binary gets 0.0 cost for all agents |
| Agents instructed via prompt will reliably write stdout JSON files | Low-Medium | Wave 2 agents receive no/garbage input; synthesis fails. Retry does not help for "succeeded with bad output" |
| `nohup` provides sufficient process detachment for production use | Low | SIGINT propagation kills background teams; SIGHUP may not be the only signal concern |
| The TUI's streaming lock is the root cause of the freeze, not something deeper in the SDK | High | If the SDK itself serializes tool calls, background execution becomes the only option |
| Heartbeat file + next-session cleanup is sufficient for orphan detection | Medium | If user does not start a new session, orphaned claude processes run indefinitely consuming API credits |
| Users actually need to interact with the TUI during long orchestrations | Medium | If users typically context-switch to a different terminal anyway, the TUI freeze is less painful than assumed |
| File-based IPC (config.json polling) provides adequate latency for /team-status | High | Polling at 1-second intervals is fine for human-readable status; only matters if real-time streaming is needed |
| The Go binary and the MCP spawn_agent will not drift in their agent configuration parsing | Medium | Divergent agent behavior depending on spawn path; hard to debug |
| Inter-wave bash scripts with jq can reliably extract structured content from agent output | Medium-High | jq is robust for JSON; risk is agents producing invalid JSON, which is handled by the `2>/dev/null || echo` pattern |

---

## Open Questions

Questions that require further investigation or are outside theoretical scope:

1. **What is the actual behavior of `query()` under concurrent invocation?** Can the TUI call `query()` twice simultaneously, or does the SDK enforce single-session semantics? If the SDK blocks concurrent queries, Approach 1 is impossible without SDK changes.

2. **Does `claude -p` respect `CLAUDE_CODE_EFFORT_LEVEL` as an environment variable?** The plan assumes this, and `spawnAgent.ts` sets it, but I have not verified that the CLI actually reads this env var. If not, effortLevel injection is inert.

3. **What happens when `goyoke-team-run` is killed with `kill -9` (SIGKILL)?** The signal handler (section 2.4) cannot catch SIGKILL. Children inherit SIGKILL behavior only if the OS implements it via process groups. Otherwise, children become orphans with no heartbeat cleanup trigger until the next session.

4. **Is the `--permission-mode delegate` flag sufficient for headless agents?** If any tool requires explicit permission that `delegate` does not cover, the headless agent will hang waiting for user input that never arrives, eventually timing out.

5. **How does the Go binary discover the project root?** The plan shows `cmd.Dir = projectRoot` but does not specify how `projectRoot` is resolved. If it reads from config.json's `team_dir` and traverses up, this is brittle. If it reads from an env var, who sets it?

6. **What is the cost of a `claude -p` session that reads a stdin file, writes a stdout file, and exits?** The plan assumes this is comparable to the current `spawn_agent` cost, but the prompt envelope adds overhead (self-orientation protocol, key paths table, constraint section). If the envelope adds 2K tokens to every agent invocation, the cumulative cost across a 3-agent braintrust run is non-trivial.

---

## Handoff Notes for Beethoven

### Key Theoretical Insights

1. **The root cause is the TUI's application-level streaming mutex, not the JavaScript event loop.** The `for await` loop yields cooperatively; the freeze comes from `isStreaming` locking out user input. A Go binary is a correct solution but not the only one.

2. **Three supervision topologies is a significant architectural risk.** ProcessRegistry (TUI), spawn_agent (MCP/TUI), and goyoke-team-run (detached Go) each track processes independently. The heartbeat-based orphan detection is eventually consistent with session-start granularity. This should be flagged as a first-class concern, not a "Phase 5" deferral.

3. **The structured I/O schemas are independently valuable** and should be designed as a reusable primitive, not coupled to the Go binary. They benefit the existing spawn_agent path equally.

4. **`nohup` is insufficient for production process detachment.** Proper daemon pattern (setsid, PID file, fd closure) prevents SIGINT propagation and process group contamination. This is a concrete implementation concern the Staff-Architect should verify.

5. **Prompting is not contract enforcement.** The plan's stdout validation checks structural envelope only. Semantic emptiness (valid JSON, empty content) passes validation. This failure mode is high-probability initially and not addressed by the retry mechanism.

### Points Requiring Practical Validation

- Whether `claude -p --output-format json` output format is documented and stable
- Whether `query()` supports concurrent invocations from the same TUI process
- Whether `nohup` actually prevents SIGINT propagation in the TUI's terminal context
- Whether the prompt envelope overhead materially affects per-agent cost
- Whether `registerChildProcessCleanup()` wiring (Phase 0) actually resolves the orphan problem for the existing spawn path

### Potential Conflicts with Practical Concerns

- Theory says "fix the TUI concurrency model"; practice may say "too risky to refactor the query hook"
- Theory says "proper daemon, not nohup"; practice may say "nohup works for MVP"
- Theory says "unified supervision topology"; practice may say "two separate systems is acceptable complexity for v1"
- Theory says "structured schemas are independently valuable"; practice may say "don't scope-creep the Go binary plan"

---

## Metadata

```yaml
analysis_id: einstein-team-coord-2026-02-06
problem_brief_id: IMPLEMENTATION-PLAN-team-coordination
frameworks_applied:
  - Process Supervision Theory (Erlang/OTP)
  - Unix Process Group Model
  - Contract-Driven Design
assumptions_surfaced: 9
novel_approaches_proposed: 4 (3 alternatives + 1 synthesis)
files_examined:
  - /home/doktersmol/Documents/goYoke/tickets/team-coordination/IMPLEMENTATION-PLAN.md
  - /home/doktersmol/Documents/goYoke/.claude/agents/agents-index.json
  - /home/doktersmol/Documents/goYoke/PARALLEL-ORCHESTRATION-DESIGN.md
  - /home/doktersmol/Documents/goYoke/packages/tui/src/mcp/tools/spawnAgent.ts
  - /home/doktersmol/Documents/goYoke/packages/tui/src/spawn/processRegistry.ts
  - /home/doktersmol/Documents/goYoke/packages/tui/src/lifecycle/shutdown.ts
  - /home/doktersmol/Documents/goYoke/packages/tui/src/index.tsx
  - /home/doktersmol/Documents/goYoke/packages/tui/src/hooks/useClaudeQuery.ts
```
