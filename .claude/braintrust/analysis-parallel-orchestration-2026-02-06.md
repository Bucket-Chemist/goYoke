# Braintrust Analysis: Parallel Orchestration Architecture

> **Date**: 2026-02-06
> **Document Under Review**: `PARALLEL-ORCHESTRATION-DESIGN.md`
> **Analysts**: Einstein (theoretical), Staff-Architect (practical)
> **Synthesizer**: Beethoven
> **Orchestrator**: Mozart
> **Total Cost**: ~$4.10

---

## Executive Summary

The Parallel Orchestration Design proposes transforming orchestrator agents from blocking foreground processes to a three-phase pattern (PLAN → EXECUTE → DELIVER) using a Go binary (`goyoke-team-run`) for background execution. While the **core insight is valid**—orchestrators should be backgroundable for TUI responsiveness—the proposed implementation introduces **significant architectural complexity** that may not be justified.

**Verdict**: The design is **architecturally coherent but over-engineered**. A simpler TypeScript-native alternative exists that achieves the same goals with 60% less effort and zero breaking changes to the TUI.

**Critical Action Required**: Before any implementation, validate whether Claude Agent SDK supports concurrent `query()` sessions. If yes, the entire Go binary can be replaced with ~100 lines of TypeScript.

---

## Problem Statement

### The UX Problem (Valid)

Orchestrator agents (mozart, review-orchestrator, impl-manager) spawn subagents **sequentially**, blocking the TUI. A braintrust analysis that could run einstein + staff-architect in parallel instead waits:
- ~2min for einstein
- ~2min for staff-architect
- ~1min for beethoven
- **= 5 minutes of frozen TUI**

During this time, users cannot:
- Work on other code
- Monitor progress
- Run concurrent orchestrations
- Cancel gracefully with partial results

### The Proposed Solution

Three-phase pattern:
1. **PLAN** (Foreground LLM, ~30s): Interview, scout, write team config + stdin schemas
2. **EXECUTE** (Background Go binary): Spawn waves, monitor PIDs, collect stdout
3. **DELIVER** (User-initiated): `/team-status`, `/team-result` commands

---

# PART I: EINSTEIN ANALYSIS (Theoretical)

## 1. First Principles Decomposition

### 1.1 What Problem Are We Actually Solving?

The design document frames this as a "TUI freeze" problem, but let's decompose further:

**Surface Problem**: TUI is unresponsive during orchestration
**Proximate Cause**: `query()` function blocks until LLM returns
**Root Cause**: Node.js single-threaded event loop cannot process UI events while awaiting LLM response
**Fundamental Tension**: Synchronous LLM API design vs. interactive UI requirements

The Go binary approach attempts to solve this by **exiting the Node.js process entirely** for orchestration execution. This is a valid architectural response, but it's not the only one.

### 1.2 Alternative Framings

| Framing | Solution Space |
|---------|----------------|
| "TUI blocks during LLM calls" | Worker threads, child processes, async patterns |
| "Orchestration takes too long" | Parallelization, caching, smaller models |
| "User can't see progress" | Streaming, progress callbacks, status polling |
| "User can't cancel" | Graceful shutdown, checkpoint/resume |

The design document conflates all four problems and proposes a single solution. This may be appropriate, but each framing suggests different architectural approaches.

### 1.3 The SDK Question

The Claude Agent SDK uses `query()` as its primary interface. The design document assumes this is inherently blocking and non-concurrent. **This assumption has not been validated.**

Key questions:
1. Does the SDK support multiple simultaneous `query()` calls?
2. If yes, do they share session state or are they isolated?
3. What happens to cost tracking with concurrent queries?
4. Can the SDK's internal event handling yield to the Node.js event loop?

**If the SDK supports concurrent queries**, the entire Go binary architecture becomes unnecessary. The solution would be:

```typescript
// Hypothetical concurrent SDK usage
const einstein = sdk.query(einsteinPrompt, { sessionId: "team-1-einstein" });
const staffArch = sdk.query(staffArchPrompt, { sessionId: "team-1-staff-arch" });

// Both run concurrently, event loop remains free
const [einsteinResult, staffArchResult] = await Promise.all([einstein, staffArch]);
```

This would require approximately 100 lines of TypeScript vs. the proposed 2,000+ lines of Go + TypeScript + JSON schemas.

---

## 2. Core Claims Validity Assessment

### 2.1 Claim: "Orchestrators Don't Write Code"

**Stated in Design**: "Orchestrators (mozart, review-orchestrator, impl-manager) do not use Claude's Write or Edit tools on implementation files."

**Analysis**:

This claim is **true by current convention, but not structurally guaranteed**.

**Evidence supporting the claim**:
- Mozart's role is coordination and synthesis, not implementation
- review-orchestrator spawns reviewers, doesn't edit code itself
- impl-manager delegates to worker agents for actual implementation

**Evidence undermining the claim**:
- No enforcement mechanism prevents an orchestrator from using Edit/Write
- If an orchestrator's prompt changes, it could start writing files
- Some orchestration patterns legitimately need intermediate file writes (e.g., writing a summary for user review before continuing)

**Structural guarantee would require**:
- Tool allowlist in agent definition: `"allowed_tools": ["Read", "Glob", "Task", "spawn_agent"]`
- Hook enforcement blocking Edit/Write for orchestrator agent types
- Neither exists in the current system

**Verdict**: PARTIALLY VALID. True today by prompt design, could break if prompts evolve.

**Implication for design**: The design's assumption that "Go binary can handle all coordination file I/O" is valid today but fragile. If an orchestrator ever needs to write a file for user review mid-process, the architecture breaks.

### 2.2 Claim: "Go Binary Can Do File I/O Outside Claude Permissions"

**Stated in Design**: "All of this can be done by a Go binary using native os.WriteFile() - completely outside Claude's tool permission system."

**Analysis**:

This is **technically sound but creates a dual-write problem**.

**Why it works**:
- Go binaries invoked via `Bash({run_in_background: true})` run as separate processes
- `os.WriteFile()` uses OS-level file operations, not Claude's Write tool
- No permission prompts, no tool tracking, no conversation injection

**The dual-write problem**:
- Two systems (Go binary + Claude agents) write to overlapping directories
- No coordination mechanism exists
- Potential race conditions:
  - Go binary updates config.json while agent reads it
  - Agent writes stdout.json while Go binary tries to collect it
  - Two agents in same wave write to same file (not prevented by design)

**The permission bypass concern**:
- Claude's permission system exists for safety and audit
- Go binary circumvents this entirely
- No record in conversation of what Go binary wrote
- User loses visibility into file modifications

**Verdict**: TECHNICALLY SOUND but introduces coordination and auditability gaps.

### 2.3 Claim: "Spawned CLI Processes Have Full Tool Access"

**Stated in Design**: "The spawned claude CLI processes DO have full tool access - they run as independent CLI sessions with --permission-mode delegate."

**Analysis**:

This is **TRUE but understates the context isolation cost**.

**What agents gain**:
- Full tool access (Read, Write, Edit, Bash, etc.)
- Independent conversation context
- Own permission scope

**What agents lose**:
- Parent conversation history (they start fresh)
- Knowledge of what siblings are doing
- Shared tool results (each must re-read files)
- Coordinated error handling
- Real-time progress visibility to parent

**Context isolation implications**:

| Scenario | In-Process Orchestration | Go Binary Orchestration |
|----------|-------------------------|------------------------|
| Agent needs file content | Already in parent context | Must re-read from disk |
| Agent encounters error | Parent can handle, retry | Agent fails, Go collects exit code |
| User wants to see reasoning | In conversation history | Lost unless written to file |
| Agent needs to ask user | Can use AskUserQuestion | **Cannot** - headless mode |
| Agents need to coordinate | Share context | Must use filesystem |

**Verdict**: TRUE but the isolation is severe. Each CLI process is an island.

---

## 3. Load-Bearing Assumptions

The design makes several assumptions that, if violated, would cause the architecture to fail.

### 3.1 Assumption: Config.json is Single-Writer

**What the design assumes**: Only `goyoke-team-run` writes to config.json. Agents read it but don't modify it.

**Why this matters**: JSON files don't support concurrent writes. Two writers would corrupt the file.

**How this could break**:
1. Agent reads config.json for self-orientation
2. Go binary updates config.json (wave advancement)
3. Agent's read gets partial/corrupted data

**Even with single-writer**:
- Go binary crashes mid-write → corrupted config.json
- Agent reads during write → undefined behavior
- No file locking specified in design

**Mitigation needed**:
- Atomic writes (write to temp file, rename)
- File locking (flock)
- Or: Move away from filesystem IPC entirely

### 3.2 Assumption: Orchestrators Never Need Intermediate File Writes

**What the design assumes**: All orchestrator file writes happen in the planning phase. During execution, only agents write (to stdout files).

**Why this matters**: If orchestrators need to write files during execution, the Go binary can't do it (it's not an LLM).

**Scenarios that would break this**:
1. Mozart wants to show user a summary after Wave 1 before proceeding to Wave 2
2. Review-orchestrator needs to write an intermediate report for user approval
3. Impl-manager discovers new tasks mid-execution and needs to update the plan

**Implication**: The design constrains future orchestrator evolution. Any feature requiring LLM file writes during execution breaks the architecture.

### 3.3 Assumption: PID Monitoring Captures All Failure Modes

**What the design assumes**: `os.Process.Wait()` will reliably capture when child processes exit.

**What it misses**:

| Failure Mode | os.Process.Wait() Behavior | Result |
|--------------|---------------------------|--------|
| Normal exit | Returns exit code | ✓ Works |
| Process calls exit(1) | Returns exit code 1 | ✓ Works |
| Segfault | Returns signal number | ✓ Works |
| OOM kill | Process disappears, no exit | May hang waiting |
| SIGKILL from system | Similar to OOM | May hang waiting |
| Zombie process | Wait returns but process resources leak | Partial failure |
| Resource exhaustion (no more PIDs) | Spawn fails | Different error path |
| Disk full | Agent fails mid-write | stdout.json may be incomplete/missing |

**Particularly concerning**: OOM kills. Claude agents can use significant memory. If the system OOM-kills an agent, the Go binary may wait indefinitely.

**Mitigation needed**:
- Timeout on process wait
- Health check polling (is PID still alive?)
- Resource limits on child processes (cgroups, ulimit)

### 3.4 Assumption: TUI Lifetime >= Team Lifetime

**What the design assumes**: The TUI process will outlive all teams it spawns.

**How this breaks**:
1. User starts `/braintrust` → Go binary + agents spawn
2. User closes terminal (SIGHUP) or TUI crashes
3. Go binary continues running (background process)
4. Spawned agents continue running
5. No parent to collect results
6. Costs continue accumulating
7. Processes become orphans

**Current design has NO mitigation**:
- No PID file for cleanup
- No heartbeat from TUI to Go binary
- No `goyoke-team-cleanup` command
- No session startup cleanup

**This is the most severe gap in the design.**

### 3.5 Assumption: --permission-mode delegate Requires No User Interaction

**What the design assumes**: Headless CLI mode with `--permission-mode delegate` will run without prompts.

**Why this might not be true**:
- Delegate mode allows tool use but may still prompt for dangerous operations
- Certain tools might have hardcoded prompts
- Hook system behavior in headless mode is **unvalidated**

**Specific concern**: Does `goyoke-validate` run in headless CLI mode?

If not, agents spawned by Go binary:
- Could use Task(opus) even though it's supposed to be blocked
- Could exceed delegation ceiling
- Would bypass routing validation entirely

**This MUST be tested before implementation.**

---

## 4. Novel Failure Modes Not Addressed in Design

### 4.1 The Orphan Team Problem

**Scenario**:
1. User runs `/braintrust`
2. Mozart plans team, spawns `goyoke-team-run` in background
3. Einstein and Staff-Architect start (Wave 1)
4. User's laptop runs out of battery / TUI crashes / user kills terminal
5. Go binary and agents continue running
6. No parent process to:
   - Collect results
   - Track costs
   - Display to user
   - Clean up when done

**Impact**:
- Resource leak (processes, memory, API costs)
- Stale teams accumulate across sessions
- User may not even know orphans exist
- Costs continue accruing invisibly

**Required mitigation**:
- Team PID file: `teams/{name}/team.pid` containing Go binary PID
- Session startup: `goyoke-team-cleanup --kill-orphans`
- Heartbeat: Go binary checks for parent periodically, self-terminates if orphaned
- Cost ceiling: Go binary enforces max cost per team

### 4.2 The Cost Runaway Problem

**Scenario**:
1. Team starts with 5 agents
2. Wave 1 fails, retries, fails again
3. Wave 2 agents spawn anyway (best-effort)
4. Each failure adds cost but no useful output
5. Team eventually marked "failed" after $20+ spend

**Current design**:
- Tracks cost in config.json
- No enforcement of ceiling
- Retry logic adds cost
- Best-effort continuation adds cost

**Required mitigation**:
- Team-level budget: `"max_cost_usd": 10.00` in team config
- Go binary checks budget before each spawn
- Abort with partial results if budget exceeded
- Notify user of budget abort (write to status file)

### 4.3 The Hook Bypass Problem

**Current system architecture**:
- `goyoke-validate` runs as PreToolUse hook
- Blocks Task(opus), enforces subagent_type, logs violations
- Runs within Claude Code session

**Go binary architecture**:
- Spawns `claude` CLI processes directly
- CLI processes may not have hooks configured
- Even if configured, may not be in hook path

**Questions that need answers**:
1. Does `claude --permission-mode delegate` run hooks?
2. If yes, which hooks? PreToolUse? PostToolUse? Both?
3. Is `.claude/settings.json` respected in headless mode?
4. Are hook binaries in PATH for headless processes?

**If hooks don't run**:
- Agents could violate routing rules
- No sharp edge capture
- No ML telemetry logging
- No routing reminders

**Mitigation**:
- Test hook execution in headless mode before implementing
- If hooks don't run, add validation in Go binary
- Or: Accept that background agents are "unmonitored"

### 4.4 The Context Window Pollution Problem

**Scenario**:
1. Braintrust team completes with 5 agents
2. User runs `/team-result`
3. TUI needs to display results
4. Each agent's stdout is ~10K tokens
5. Total context injection: 50K+ tokens

**Problems**:
- TUI conversation context bloats
- Subsequent queries slow down
- Context window may truncate critical info
- User overwhelmed with detail

**Mitigation needed**:
- Summary mode: Show only executive summary, link to full files
- Lazy loading: Only inject full detail on user request
- Separate context: Results in separate conversation, not main TUI context
- Archival: Move completed team results out of active context

### 4.5 The Nesting Depth Reset Problem

**Current system**:
- `GOYOKE_NESTING_LEVEL` tracks agent spawn depth
- Level 0: Router
- Level 1: First-tier agents (mozart, go-pro)
- Level 2: Sub-agents (einstein, scaffolder)
- Hooks use this for validation

**Go binary problem**:
- Go binary doesn't know about `GOYOKE_NESTING_LEVEL`
- Spawned CLI processes start at Level 0 (default)
- Agents think they're root-level
- Can spawn when they shouldn't be able to

**Mitigation**:
```go
// In goyoke-team-run
env := os.Environ()
env = append(env, "GOYOKE_NESTING_LEVEL=2")  // or read from team config
```

But this requires:
- Team config to store nesting level
- Go binary to propagate correctly
- Handling for multi-wave depth (Wave 2 agents are Level 3?)

---

## 5. Einstein's Key Insight

> The design document solves the wrong problem at the wrong layer.

**The actual problem**: Node.js event loop blocks during synchronous `query()` calls.

**The design's solution**: Exit Node.js entirely, run orchestration in Go.

**The simpler solution**: Run multiple `query()` calls concurrently within Node.js.

**Analogy**: It's like building a separate kitchen because your stove only has one burner. But maybe your stove has four burners and you never tried using them.

**Before building the Go binary, answer this question**:
```typescript
// Does this work?
const q1 = sdk.query("Task 1", { sessionId: "a" });
const q2 = sdk.query("Task 2", { sessionId: "b" });
await Promise.all([q1, q2]);  // Do both run? Does event loop stay free?
```

If yes: Delete the design document. Write a `ConcurrentTeamManager` class in TypeScript. Done.

If no: The Go binary approach becomes justified, but still needs the mitigations outlined above.

---

## 6. Einstein's Verdict

**Architecture Assessment**: Coherent but over-engineered

**The design would benefit from**:
1. Pre-implementation SDK capability testing
2. Explicit failure mode catalog (orphans, cost runaway, hook bypass)
3. ProcessRegistry unification before building dual process management
4. Team-level budget controls
5. Orphan detection and cleanup

**Recommendation**: Invest 2 hours testing SDK concurrent queries before investing 200+ hours in Go binary development.

---

# PART II: STAFF-ARCHITECT ANALYSIS (Practical)

## 1. Seven-Layer Review Framework

### Layer 1: Assumptions Audit

| # | Assumption | Source | Status | Risk | Evidence |
|---|------------|--------|--------|------|----------|
| 1 | Orchestrators never write implementation files | Design doc §2, §3 | **UNVALIDATED** | HIGH | True by prompt convention only. Mozart, review-orchestrator, impl-manager prompts checked - none use Write/Edit on .go/.ts/.py files. But no structural enforcement exists. |
| 2 | Config.json single-writer model works | Design doc §4 | **ASSUMED** | MEDIUM | Go binary is only writer by design. But no file locking, no atomic writes specified. Race conditions possible if agent reads during write. |
| 3 | PID monitoring catches all failures | Design doc §4 | **PARTIALLY VALID** | MEDIUM | os.Process.Wait() catches most cases. OOM kills, resource exhaustion, zombie processes may not be handled. |
| 4 | Hooks run in headless CLI mode | Implicit | **UNVALIDATED** | HIGH | Design assumes agents follow routing rules. But goyoke-validate is a PreToolUse hook. Does it run when `claude --permission-mode delegate` spawns? No verification. |
| 5 | TUI outlives all teams | Implicit | **ASSUMED** | HIGH | No heartbeat, no orphan detection, no cleanup mechanism. TUI crash leaves teams running indefinitely. |
| 6 | CLI JSON output format is stable | Design doc §4 | **MEDIUM CONFIDENCE** | MEDIUM | Go binary parses CLI stdout for cost_usd. Claude CLI output format is undocumented and could change. |
| 7 | Inter-wave bash scripts are reliable | Design doc §5 | **LOW RISK** | LOW | jq is robust. Scripts are optional optimization. |
| 8 | Agents will comply with stdout schema | Design doc §5 | **UNVALIDATED** | MEDIUM | Agents must write specific JSON structure. Requires prompt engineering and validation. |

**Summary**: 4 assumptions unvalidated, 2 high-risk. The assumption profile indicates moderate-to-high implementation risk.

### Layer 2: Dependency Analysis

#### External Dependencies

| Dependency | Version Coupling | Stability | Risk | Notes |
|------------|-----------------|-----------|------|-------|
| Claude CLI (`claude` binary) | Tight | Unknown | MEDIUM | Output format parsed by Go binary. Undocumented API. |
| Claude Agent SDK | Tight | Stable | LOW | Used by TUI, well-documented |
| Go runtime | Loose | Stable | LOW | Standard library only needed |
| Node.js child_process | Loose | Stable | LOW | Bash({run_in_background}) uses this |
| jq (optional) | Loose | Stable | LOW | Inter-wave scripts only |

#### Internal Dependencies

| Component | Depends On | Risk |
|-----------|-----------|------|
| goyoke-team-run | agents-index.json, team config schema | MEDIUM - Schema changes break binary |
| goyoke-team-init | Team config templates, stdin schemas | MEDIUM - Template changes need binary updates |
| /team-status | config.json structure | LOW - Read-only |
| Orchestrator prompts | stdin/stdout schemas | HIGH - Schema changes need prompt rewrites |

#### The Critical Dependency Problem: Dual Process Management

**CRITICAL ISSUE IDENTIFIED**

The design creates two independent process management systems:

**System 1: TUI ProcessRegistry (TypeScript)**
- Location: `packages/tui/src/spawn/processRegistry.ts`
- Manages: Node.js child processes spawned via spawnAgent MCP tool
- Cleanup: SIGTERM → 5s → SIGKILL on TUI shutdown
- Tracking: Map<processId, ProcessInfo>
- Cost aggregation: Built-in

**System 2: goyoke-team-run (Go)**
- Manages: `claude` CLI processes spawned via exec.Command
- Cleanup: Signal forwarding on SIGTERM
- Tracking: PIDs in config.json
- Cost aggregation: Separate (parsed from CLI output)

**These systems do not communicate.**

| Scenario | TUI ProcessRegistry | Go Binary | Result |
|----------|---------------------|-----------|--------|
| User kills TUI | Kills its children | Not notified | **Go binary + agents orphaned** |
| Go binary crashes | Not aware | Children continue | **Agents orphaned** |
| `/team-cancel` | Cannot kill Go's children | Would need to receive signal | **No clean cancel** |
| Cost query | Knows own costs | Knows its costs | **Two separate cost totals** |
| Agent list | Shows its agents | Shows its agents | **User sees partial view** |

**This is a fundamental architectural problem, not a detail to fix later.**

### Layer 3: Failure Mode Analysis

#### Systematic Failure Enumeration

| ID | Failure Mode | Trigger | Probability | Impact | Detection | Recovery | Design Coverage |
|----|--------------|---------|-------------|--------|-----------|----------|-----------------|
| F1 | Agent exits non-zero | Bug, timeout, OOM | HIGH | MEDIUM | Exit code | Retry once | YES |
| F2 | Agent hangs indefinitely | Infinite loop, stuck prompt | MEDIUM | HIGH | Timeout | None specified | **NO** |
| F3 | All wave tasks fail | Systemic issue | LOW | HIGH | All exit codes non-zero | Abort | YES |
| F4 | Go binary crash | Bug, OOM | LOW | CRITICAL | Process disappears | **NONE** | **NO** |
| F5 | TUI crash/close | User action, bug | MEDIUM | CRITICAL | N/A | **NONE** | **NO** |
| F6 | Config.json corruption | Crash during write | LOW | HIGH | JSON parse error | **NONE** | **NO** |
| F7 | Stdout schema violation | Bad prompt, agent error | HIGH | MEDIUM | Schema validation | Fallback to raw | PARTIAL |
| F8 | Cost overrun | Expensive operations | MEDIUM | HIGH | Cost tracking | **NONE - no ceiling** | **NO** |
| F9 | Disk full | Large outputs | LOW | MEDIUM | Write error | **NONE** | **NO** |
| F10 | Hook bypass | Headless mode | MEDIUM | HIGH | None in design | **NONE** | **NO** |

**Coverage Assessment**: 3/10 failure modes adequately addressed. 7/10 have no or partial coverage.

#### The Orphaned Sub-Agent Problem (Detailed)

This is the most severe failure mode and deserves detailed analysis.

**Failure Sequence**:
```
T+0:00  User: /braintrust "analyze X"
T+0:05  Mozart: Plans team, writes config
T+0:10  Mozart: Bash({run_in_background: true, command: "goyoke-team-run ..."})
T+0:11  Mozart returns "Team dispatched"
T+0:12  TUI shows prompt again (user thinks they can work)
T+0:15  goyoke-team-run: Spawns einstein (PID 12345), staff-architect (PID 12346)
T+1:30  User: Closes laptop lid (or: TUI crashes, or: user Cmd+Q)
T+1:31  TUI process exits
T+1:32  Node.js kills its direct children (none - Go binary was backgrounded)
T+1:33  goyoke-team-run: Still running (PID 12340)
T+1:33  einstein: Still running (PID 12345)
T+1:33  staff-architect: Still running (PID 12346)
T+3:00  Wave 1 completes, goyoke-team-run spawns beethoven (PID 12347)
T+4:30  Team completes. Go binary exits.
T+4:31  Agents exit.

        Total orphan run time: 3 minutes
        Orphan cost: ~$2.50 (3 Opus agents, unused)
        User awareness: NONE (they closed the laptop)
```

**Cascading Effects**:
1. Costs accumulate against user's API account
2. Results written to disk but never collected
3. Next session: stale team directories accumulate
4. If this happens repeatedly: disk fills, account overcharges

**What the design should specify but doesn't**:
- Heartbeat from TUI to Go binary (e.g., every 30s touch a heartbeat file)
- Go binary monitors heartbeat, self-terminates if stale
- Session startup: `goyoke-team-cleanup --kill-orphans`
- Team PID lockfile: `/tmp/goyoke-team-{name}.pid`

### Layer 4: Cost/Benefit Analysis

#### Implementation Effort Estimate

| Phase | Tasks | Effort (days) | Dependencies |
|-------|-------|---------------|--------------|
| **Phase 1: Background Engine** | | | |
| 1.1 Team config JSON schema | Design + validate | 1-2 | None |
| 1.2 goyoke-team-init binary | Go development | 2-3 | Schema |
| 1.3 goyoke-team-run binary | Go development, PID mgmt, wave scheduling | 5-7 | Schema, init |
| 1.4 Integration tests | Multi-process testing | 2-3 | Binaries |
| **Phase 1 Subtotal** | | **8-12 days** | |
| **Phase 2: Structured I/O** | | | |
| 2.1 stdin schemas (8 agent types) | Schema design | 2-3 | None |
| 2.2 stdout schemas (8 agent types) | Schema design | 2-3 | None |
| 2.3 Agent prompt updates | Prompt engineering | 3-4 | Schemas |
| 2.4 Inter-wave scripts | Bash/jq scripting | 1-2 | Schemas |
| **Phase 2 Subtotal** | | **5-8 days** | |
| **Phase 3: Slash Commands** | | | |
| 3.1 /team-status | Skill + config parsing | 1-2 | Config schema |
| 3.2 /team-result | Skill + stdout parsing | 1-2 | stdout schemas |
| 3.3 /team-cancel | Skill + signal sending | 1-2 | Go binary |
| 3.4 /teams listing | Skill + dir scanning | 0.5-1 | Dir structure |
| **Phase 3 Subtotal** | | **4-6 days** | |
| **Phase 4: Orchestrator Rewrites** | | | |
| 4.1 Mozart prompt rewrite | Prompt engineering | 2-3 | Schemas, binaries |
| 4.2 Review-orchestrator rewrite | Prompt engineering | 1-2 | Schemas |
| 4.3 Impl-manager rewrite | Prompt engineering | 2-3 | Schemas |
| 4.4 Integration testing | End-to-end testing | 2-3 | All above |
| **Phase 4 Subtotal** | | **5-9 days** | |
| **TOTAL** | | **22-35 days** | |

#### Maintenance Burden Assessment

| Component | Maintenance Tasks | Frequency | Burden |
|-----------|-------------------|-----------|--------|
| Go binaries | Bug fixes, CLI format updates | Monthly | HIGH |
| JSON schemas | Version management, migrations | Per feature | MEDIUM |
| Agent prompts | Schema compliance tuning | Ongoing | HIGH |
| Inter-wave scripts | jq query fixes | Rare | LOW |
| Slash commands | UI updates, error handling | Per feature | LOW |

**Total Maintenance Burden**: HIGH. Two languages (Go + TypeScript), 8+ schemas, prompt engineering.

#### Value Delivered

| Benefit | Impact | User Value |
|---------|--------|------------|
| Interactive TUI during orchestration | HIGH | Users can continue working |
| Parallel agent execution | MEDIUM | Faster orchestration completion |
| Multiple concurrent teams | MEDIUM | Power user feature |
| Progress visibility | HIGH | Users know what's happening |
| Partial result recovery | MEDIUM | Don't lose work on failure |

**Value Assessment**: HIGH. These are significant UX improvements.

#### ROI Calculation

- **Cost**: 22-35 developer-days + ongoing maintenance
- **Value**: Significant UX improvement for orchestration workflows
- **Risk**: HIGH (7/10 failure modes unaddressed)
- **Alternative**: TypeScript TeamManager (8-12 days, lower risk)

**ROI Verdict**: MODERATE-NEGATIVE. Value is real but cost and risk are disproportionate given the alternative exists.

### Layer 5: Testing Strategy Assessment

#### What Testing Infrastructure Would Be Needed

| Test Type | What It Tests | Infrastructure Required | Specified in Design? |
|-----------|---------------|------------------------|---------------------|
| Unit tests (Go) | Binary logic | Go testing framework | NO |
| Unit tests (TS) | Slash command logic | Jest | NO |
| Integration tests | Go ↔ Claude CLI | Mock CLI, process spawning | NO |
| E2E tests | Full workflow | Real Claude API, real TUI | NO |
| Chaos tests | Crash recovery | Process killing, corruption injection | NO |
| Load tests | Concurrent teams | Multiple simultaneous teams | NO |
| Cost tests | Budget enforcement | API cost mocking | NO |

**Design specifies**: None of the above.

**Critical Gap**: Testing multi-process systems is hard. The design provides no guidance on:
- How to mock Claude CLI output
- How to simulate crashes
- How to verify orphan cleanup
- How to test concurrent teams without massive API costs

#### Recommended Test Categories

1. **Process Lifecycle Tests**
   - Spawn → complete → collect flow
   - Spawn → fail → retry → collect flow
   - Spawn → crash (Go binary) → orphan detection
   - Spawn → cancel → cleanup verification

2. **Schema Compliance Tests**
   - Valid stdout schema acceptance
   - Invalid stdout schema handling
   - Partial stdout (agent crashed mid-write)
   - Empty stdout (agent produced nothing)

3. **Concurrency Tests**
   - Two teams simultaneously
   - Same agent type in multiple teams
   - Resource contention (disk, API)

4. **Cost Control Tests**
   - Budget ceiling enforcement
   - Cost tracking accuracy
   - Orphan cost attribution

### Layer 6: Architecture Smell Detection

#### Smell 1: Dual Process Management

**Symptom**: Two independent systems tracking processes (ProcessRegistry + Go binary)
**Cause**: Go binary chosen for background execution, but TUI already has process management
**Risk**: Orphans, cost tracking gaps, inconsistent state
**Refactoring**: Unify process management OR choose one approach

#### Smell 2: Filesystem as Message Bus

**Symptom**: Components communicate via JSON files (config.json, stdin/stdout files)
**Cause**: Go binary and Claude agents are separate processes
**Risk**: Race conditions, corruption, polling overhead
**Refactoring**: Consider Unix sockets for real-time updates OR accept file-based IPC limitations

#### Smell 3: Permission System Bypass

**Symptom**: Go binary writes files via os.WriteFile, bypassing Claude's Write tool
**Cause**: Go binary is not a Claude agent
**Risk**: Audit trail gaps, user loses visibility, security bypass
**Refactoring**: Accept as architectural trade-off OR limit Go binary file writes to team directory only

#### Smell 4: Schema Proliferation

**Symptom**: 8+ stdin schemas, 8+ stdout schemas, team config schema, all need versioning
**Cause**: Structured I/O between agents
**Risk**: Schema drift, version incompatibility, maintenance burden
**Refactoring**: Consider generic schema with agent-specific fields OR accept schema maintenance burden

### Layer 7: Contractor Readiness Assessment

**Question**: Could a competent contractor implement this design as written?

#### Readiness Checklist

| Criterion | Status | Gap |
|-----------|--------|-----|
| Clear scope definition | YES | - |
| Unambiguous requirements | PARTIAL | Several "or" options without decision |
| Complete interface specifications | NO | TUI ↔ Go binary IPC not specified |
| Error handling guidance | NO | 7/10 failure modes unaddressed |
| Testing strategy | NO | No test infrastructure specified |
| Success criteria | NO | No acceptance criteria defined |
| Dependency documentation | PARTIAL | Claude CLI output format undocumented |
| Rollback plan | NO | What if this doesn't work? |

**Contractor Readiness Verdict**: NOT READY

**Immediate Questions a Contractor Would Ask**:
1. "How does the TUI know when the Go binary finishes?" - Not specified
2. "What happens if I kill the TUI mid-orchestration?" - Orphan problem unaddressed
3. "How do I test this without spending $50 on Claude API calls?" - No mock infrastructure
4. "What's the exact IPC mechanism between TUI and Go binary?" - File polling assumed but interval/mechanism not specified
5. "Which failure modes should I handle vs. accept?" - No priority or severity guidance

---

## 2. TUI Integration Compatibility Analysis

This section specifically addresses the user's concern about TUI compatibility.

### 2.1 Current TUI Architecture (Relevant Components)

| Component | Location | Role | Team Impact |
|-----------|----------|------|-------------|
| useClaudeQuery hook | `hooks/useClaudeQuery.ts` | Main LLM interaction loop | Blocking point |
| ProcessRegistry | `spawn/processRegistry.ts` | PID tracking | **CRITICAL - bypassed by Go binary** |
| spawnAgent MCP tool | `mcp/tools/spawnAgent.ts` | Agent spawning | Not used by Go binary |
| Store (Zustand) | `store/` | Application state | Needs team state |
| Status bar | `components/StatusBar.tsx` | Shows current status | Needs team status |
| Skill system | `skills/` | Slash command handling | New skills needed |

### 2.2 Breaking Changes Analysis

| Change | Breaking? | Mitigation |
|--------|-----------|------------|
| New slash commands (/team-*) | NO | Additive |
| Team state in store | NO | Additive (new slice) |
| Status bar team indicator | NO | Additive |
| ProcessRegistry bypass | **YES** | Go binary processes invisible to TUI |
| New directory structure | NO | Backward compatible |

**ONE BREAKING CHANGE IDENTIFIED**: ProcessRegistry bypass

**Details**: The Go binary spawns `claude` CLI processes that are not in TUI's ProcessRegistry. This means:
- `/agents` command won't show them
- TUI shutdown won't kill them
- Cost aggregation misses them
- Session cleanup incomplete

**Severity**: HIGH for users who expect TUI to manage all spawned agents.

### 2.3 Additive Changes (Non-Breaking)

| Addition | Description | Integration Point |
|----------|-------------|-------------------|
| TeamsSlice | Store slice for team state | `store/teamsSlice.ts` (new) |
| TeamStatus component | Shows active teams | `components/TeamStatus.tsx` (new) |
| /team-status skill | Query team progress | `skills/team-status/SKILL.md` (new) |
| /team-result skill | Show team results | `skills/team-result/SKILL.md` (new) |
| /team-cancel skill | Cancel running team | `skills/team-cancel/SKILL.md` (new) |
| /teams skill | List all teams | `skills/teams/SKILL.md` (new) |
| Session teams directory | Persistent storage | `.claude/sessions/{id}/teams/` |

### 2.4 TUI Event Loop Consideration

**Current State**:
```typescript
// useClaudeQuery.ts (simplified)
for await (const event of agent.query(prompt, options)) {
  // This blocks until LLM returns
  // No other code runs during this await
  handleEvent(event);
}
// TUI is frozen until loop completes
```

**Why Go Binary Helps**:
- Mozart completes quickly (~30s for planning)
- Mozart returns, loop exits
- Go binary runs separately, doesn't block loop
- TUI event loop free for other interactions

**What Doesn't Help**:
- If SDK supported concurrent queries, same benefit without Go binary
- The core problem is the `for await` blocking, not orchestration itself

### 2.5 Compatibility Verdict

| Aspect | Compatible? | Notes |
|--------|-------------|-------|
| Existing workflows | YES | No changes to non-team commands |
| Existing slash commands | YES | All work unchanged |
| Agent spawning | YES | spawnAgent still works |
| Cost tracking | PARTIAL | Go binary costs tracked separately |
| Process management | NO | ProcessRegistry bypassed |
| Session cleanup | PARTIAL | Team dirs need separate cleanup |

**Overall TUI Compatibility**: MOSTLY COMPATIBLE with ONE breaking change (ProcessRegistry bypass) that affects process visibility and cleanup.

---

## 3. Staff-Architect's Alternative Proposal

Given the identified issues, a simpler alternative exists.

### 3.1 TypeScript TeamManager

Instead of a Go binary, implement team management entirely in TypeScript within the TUI process.

```typescript
// packages/tui/src/teams/TeamManager.ts

interface TeamConfig {
  id: string;
  name: string;
  members: TeamMember[];
  waves: Wave[];
  status: 'planning' | 'executing' | 'complete' | 'failed';
}

interface TeamMember {
  agentType: string;
  wave: number;
  status: 'pending' | 'running' | 'complete' | 'failed';
  processId?: string;
  result?: unknown;
}

export class TeamManager {
  private teams: Map<string, TeamConfig> = new Map();

  /**
   * Plan a team - runs in foreground (LLM interaction needed)
   * Returns quickly (~30s) with team config
   */
  async planTeam(trigger: string, input: TeamInput): Promise<TeamConfig> {
    // Use existing LLM for planning
    // Write config to team directory
    return config;
  }

  /**
   * Execute a team - runs "in background" using setImmediate
   * Uses existing spawnAgent MCP tool
   * ProcessRegistry tracks ALL processes
   */
  async executeTeam(config: TeamConfig): Promise<void> {
    for (const wave of config.waves) {
      // Spawn all agents in wave
      const promises = wave.members.map(member =>
        this.spawnMember(config.id, member)
      );

      // Wait for wave completion, but yield to event loop
      await this.waitWithYield(promises);

      // Update config, run inter-wave processing
      await this.advanceWave(config);
    }
  }

  /**
   * Spawn a single team member using existing spawnAgent
   */
  private async spawnMember(teamId: string, member: TeamMember): Promise<void> {
    const result = await mcp.callTool('spawn_agent', {
      agent: member.agentType,
      description: `Team ${teamId} - ${member.agentType}`,
      prompt: await this.buildPrompt(teamId, member),
      caller_type: 'team-manager'
    });

    // ProcessRegistry automatically tracks this
    // Cost automatically aggregated
  }

  /**
   * Wait for promises while yielding to event loop
   */
  private async waitWithYield(promises: Promise<void>[]): Promise<void> {
    const pending = new Set(promises.map((p, i) => i));

    while (pending.size > 0) {
      // Yield to event loop
      await new Promise(resolve => setImmediate(resolve));

      // Check which promises completed
      for (const index of pending) {
        // ... check completion status
      }
    }
  }

  async cancelTeam(teamId: string): Promise<void> {
    const team = this.teams.get(teamId);
    if (!team) return;

    // Kill all processes via ProcessRegistry (they're already tracked!)
    for (const member of team.members) {
      if (member.processId) {
        processRegistry.kill(member.processId);
      }
    }
  }
}
```

### 3.2 Comparison: Go Binary vs TypeScript TeamManager

| Aspect | Go Binary | TypeScript TeamManager |
|--------|-----------|----------------------|
| Implementation effort | 22-35 days | 8-12 days |
| Process tracking | Separate system | Uses ProcessRegistry |
| Orphan risk | HIGH | NONE (TUI owns all) |
| Cost tracking | Separate aggregation | Unified |
| Event loop blocking | None | Minimal (setImmediate yield) |
| Debugging | Two languages | Single language |
| Testing | Hard (multi-process) | Easier (single process) |
| Breaking changes | ONE (ProcessRegistry bypass) | ZERO |

### 3.3 When Go Binary Is Justified

The Go binary becomes justified if:
1. SDK cannot support concurrent queries
2. AND setImmediate yielding is insufficient for TUI responsiveness
3. AND the orphan/cost/process management problems are all solved first

---

## 4. Staff-Architect's Verdict

### 4.1 Design Quality Assessment

| Dimension | Score | Notes |
|-----------|-------|-------|
| Problem understanding | 9/10 | Correctly identifies UX issue |
| Solution completeness | 5/10 | Missing failure modes, testing, IPC |
| Implementation detail | 7/10 | Good for happy path, weak for errors |
| Risk identification | 4/10 | Critical risks unaddressed |
| Alternative consideration | 3/10 | TypeScript alternative not explored |
| Contractor readiness | 4/10 | Too many open questions |

**Overall Score**: 5.3/10

### 4.2 Recommendations

1. **BLOCKER**: Resolve ProcessRegistry bypass OR accept orphan risk formally
2. **BLOCKER**: Specify TUI ↔ Go binary communication mechanism
3. **BLOCKER**: Add orphan detection and cleanup
4. **HIGH**: Add team-level budget controls
5. **HIGH**: Validate hook execution in headless mode
6. **MEDIUM**: Add testing infrastructure specification
7. **MEDIUM**: Consider TypeScript alternative before committing to Go

### 4.3 Go/No-Go Recommendation

**CONDITIONAL GO**: Proceed only after:
1. SDK concurrent query capability is validated (may eliminate need)
2. ProcessRegistry unification is solved
3. Orphan problem has specified mitigation
4. One of the implementation paths is chosen (Go vs TypeScript)

---

# PART III: BEETHOVEN SYNTHESIS

## 1. Convergence Matrix

Both Einstein and Staff-Architect agree on the following points:

| # | Convergence Point | Einstein Framing | Staff-Architect Framing | Confidence |
|---|------------------|------------------|------------------------|------------|
| 1 | Core insight is valid | "Orchestrators should be backgroundable" | "UX problem is real and worth solving" | HIGH |
| 2 | Go binary introduces complexity | "Over-engineered solution" | "22-35 days, dual process management" | HIGH |
| 3 | Orphan problem is critical | "Most severe gap in the design" | "Failure mode F5, CRITICAL impact" | HIGH |
| 4 | Hook bypass unvalidated | "MUST be tested before implementation" | "Unvalidated, HIGH risk" | HIGH |
| 5 | TypeScript alternative exists | "~100 lines if SDK supports concurrency" | "TeamManager, 60% less effort" | HIGH |
| 6 | Single-writer assumption fragile | "Race conditions possible" | "No file locking specified" | MEDIUM |
| 7 | ProcessRegistry bypass is breaking | N/A (theoretical focus) | "ONE breaking change" | HIGH |

## 2. Divergence Analysis

| # | Topic | Einstein Position | Staff-Architect Position | Synthesis |
|---|-------|------------------|-------------------------|-----------|
| 1 | Primary problem framing | "Event loop concurrency" | "Process management" | **Both valid, different layers.** Einstein identifies root cause; Staff-Architect identifies symptoms. Both perspectives needed. |
| 2 | SDK investigation priority | "Test first, may eliminate Go binary" | Mentioned but not emphasized | **Einstein's priority is correct.** 2-hour test could save 200+ hours of work. |
| 3 | Effort quantification | "6+ new problems introduced" | "22-35 developer-days" | **Staff-Architect's quantification more actionable** for planning. Einstein's framing better for risk assessment. |
| 4 | Alternative specificity | "~100 lines TypeScript" (if SDK supports) | "TeamManager class" (detailed design) | **Staff-Architect's alternative more developed.** Einstein's SDK path is simpler if it works. |

## 3. Resolution of Key Tensions

### Tension 1: Go Binary Necessity

**Einstein says**: Maybe unnecessary if SDK supports concurrent queries
**Staff-Architect says**: Definitely problematic regardless

**Resolution**: **Sequential validation**
1. Test SDK concurrent queries (2 hours)
2. If works → No Go binary needed, use concurrent queries
3. If doesn't work → Evaluate TypeScript TeamManager vs Go binary
4. TypeScript TeamManager preferred unless event loop blocking is unacceptable

### Tension 2: Implementation Approach

**Einstein says**: Solve root cause (event loop blocking)
**Staff-Architect says**: Minimize integration risk (use existing infrastructure)

**Resolution**: **Both correct, different priorities**
- For speed: Staff-Architect's approach (TeamManager with setImmediate)
- For elegance: Einstein's approach (concurrent SDK queries)
- Recommend: Try Einstein's first (faster to validate), fall back to Staff-Architect's

### Tension 3: Risk Tolerance

**Einstein says**: Design has 5 novel failure modes not addressed
**Staff-Architect says**: 7/10 failure modes have no coverage

**Resolution**: **Unanimous high risk**. Both analyses identify critical gaps. The difference in count is methodological (Einstein found novel failures, Staff-Architect audited specified failures). Combined, the risk profile is: **Do not implement as-is.**

---

## 4. Unified Recommendations

### Recommendation 1: Validate SDK Concurrent Sessions
**Priority**: CRITICAL
**Effort**: 2 hours
**Blocking**: All other work
**Owner**: Developer with SDK access

**Validation test**:
```typescript
// Create two concurrent query sessions
const session1Promise = sdk.query("Hello 1", { sessionId: "test-1" });
const session2Promise = sdk.query("Hello 2", { sessionId: "test-2" });

// Do both complete? Does event loop remain responsive?
const startTime = Date.now();
let eventLoopBlocked = true;
setTimeout(() => { eventLoopBlocked = false; }, 100);

await Promise.all([session1Promise, session2Promise]);

console.log("Event loop blocked:", eventLoopBlocked); // Should be false if responsive
console.log("Both completed:", true);
```

**Decision matrix based on result**:
| Result | Action |
|--------|--------|
| Both work, event loop free | Implement concurrent TeamManager (~100 lines) |
| Both work, event loop blocks | Implement TeamManager with Web Worker |
| One works, other fails | SDK doesn't support concurrency → Go binary or TeamManager |
| Neither works | Investigate SDK limitations |

### Recommendation 2: Add Missing Specifications (If Proceeding with Go Binary)
**Priority**: HIGH
**Effort**: 1-2 days
**Blocking**: Go binary implementation

**Required additions to design document**:

1. **Failure Mode Catalog**: Full enumeration with severity, detection, recovery
2. **Orphan Detection Protocol**: Heartbeat mechanism, cleanup commands, session startup recovery
3. **TUI ↔ Go Binary IPC Specification**: Polling interval, file structure, or socket protocol
4. **Team-Level Budget Controls**: Max cost per team, enforcement mechanism
5. **Hook Inheritance Verification**: Test results for headless CLI mode
6. **Testing Infrastructure**: Mock CLI, crash simulation, concurrent team testing

### Recommendation 3: Solve ProcessRegistry Unification First
**Priority**: HIGH
**Effort**: 2-3 days
**Blocking**: Any implementation path

**Options**:

**Option A: Extend ProcessRegistry for external processes**
```typescript
// ProcessRegistry additions
interface ExternalProcess {
  pid: number;
  teamId: string;
  agentType: string;
  startedAt: Date;
}

class ProcessRegistry {
  // Existing
  private processes: Map<string, ChildProcess>;

  // New
  private externalProcesses: Map<number, ExternalProcess>;

  registerExternal(pid: number, info: ExternalProcess): void;
  killExternal(pid: number): Promise<void>;
  cleanupExternalOrphans(): Promise<void>;
}
```

**Option B: Go binary reports to TUI via Unix socket**
```
TUI listens on: /tmp/goyoke-tui-{session}.sock
Go binary connects and sends:
  {"type": "spawn", "pid": 12345, "agent": "einstein"}
  {"type": "complete", "pid": 12345, "exit_code": 0, "cost_usd": 1.23}
  {"type": "progress", "wave": 1, "completed": 2, "total": 3}
```

**Option C: Abandon Go binary (TypeScript TeamManager)**
- No external processes
- ProcessRegistry unchanged
- Simplest solution

### Recommendation 4: Consider TypeScript TeamManager as Primary Path
**Priority**: MEDIUM
**Effort**: 8-12 days
**Alternative to**: Go binary (22-35 days)

**Advantages**:
- Uses existing spawnAgent infrastructure
- ProcessRegistry tracks ALL processes
- Cost tracking unified
- No orphan problem
- 60% less effort
- Zero breaking changes

**Disadvantages**:
- Heavy orchestration may delay event loop
- Mitigated by setImmediate() yielding

**Recommendation**: Implement TypeScript TeamManager unless SDK concurrent queries work (then use that instead).

---

## 5. Decision Tree

```
START: User wants parallel orchestration
    │
    ├─► Test SDK concurrent queries (2 hours)
    │       │
    │       ├─► Works + event loop free
    │       │       └─► Implement ConcurrentTeamManager (~5 days)
    │       │           - Use concurrent query() calls
    │       │           - ProcessRegistry unchanged
    │       │           - DONE
    │       │
    │       └─► Doesn't work / event loop blocks
    │               │
    │               ├─► Acceptable to have brief TUI delays?
    │               │       │
    │               │       ├─► YES: TypeScript TeamManager (8-12 days)
    │               │       │       - setImmediate() yielding
    │               │       │       - ProcessRegistry unified
    │               │       │       - DONE
    │               │       │
    │               │       └─► NO: Need true background
    │               │               │
    │               │               └─► Go Binary Path (25-40 days)
    │               │                       - Solve ProcessRegistry first
    │               │                       - Add missing specs
    │               │                       - Implement Go binaries
    │               │                       - DONE (with tech debt)
```

---

## 6. Implementation Roadmap

### Path A: SDK Concurrent (Optimal, if SDK supports)
| Phase | Work | Days | Risk |
|-------|------|------|------|
| 0 | SDK validation | 0.25 | LOW |
| 1 | ConcurrentTeamManager | 5-7 | LOW |
| 2 | Slash commands | 3-4 | LOW |
| 3 | Orchestrator prompts | 3-5 | MEDIUM |
| **Total** | | **11-16** | **LOW** |

### Path B: TypeScript TeamManager (Recommended fallback)
| Phase | Work | Days | Risk |
|-------|------|------|------|
| 0 | SDK validation (confirms no-go) | 0.25 | LOW |
| 1 | TeamManager with setImmediate | 8-12 | MEDIUM |
| 2 | Slash commands | 3-4 | LOW |
| 3 | Orchestrator prompts | 3-5 | MEDIUM |
| **Total** | | **16-24** | **MEDIUM** |

### Path C: Go Binary (Original design, not recommended)
| Phase | Work | Days | Risk |
|-------|------|------|------|
| 0 | ProcessRegistry unification | 2-3 | MEDIUM |
| 1 | Missing specifications | 1-2 | LOW |
| 2 | Go binaries | 8-12 | HIGH |
| 3 | Structured I/O schemas | 5-8 | MEDIUM |
| 4 | Slash commands | 4-6 | LOW |
| 5 | Orchestrator prompts | 5-9 | MEDIUM |
| **Total** | | **25-40** | **HIGH** |

---

## 7. Open Questions for User Decision

1. **SDK Concurrent Queries**: Will you allocate 2 hours to test this before any implementation decision?

2. **Orphan Tolerance**: Is it acceptable for TUI crash to leave agents running? This affects whether we need heartbeat/cleanup infrastructure.

3. **Event Loop Tolerance**: Are brief TUI delays (~100ms every few seconds) acceptable during orchestration? This affects TypeScript vs Go binary decision.

4. **Go Binary Preference**: Given 40-60% effort reduction of TypeScript alternatives, is there a specific reason to prefer Go binary?

5. **ProcessRegistry Strategy**: Should we unify ProcessRegistry now (blocking), or accept the breaking change (tech debt)?

---

## 8. Final Verdict

### Architecture Assessment
| Criterion | Original Design | After Braintrust Analysis |
|-----------|-----------------|---------------------------|
| Problem understanding | STRONG | STRONG |
| Solution fit | MODERATE | WEAK (alternatives exist) |
| Implementation readiness | WEAK | BLOCKED (missing specs) |
| Risk profile | HIGH | HIGH (7+ unaddressed failures) |
| TUI compatibility | PARTIAL | ONE breaking change |

### Recommendation

**DO NOT implement the Go binary design as written.**

Instead:
1. Spend 2 hours validating SDK concurrent queries
2. If SDK works → implement concurrent TeamManager (11-16 days, low risk)
3. If SDK doesn't work → implement TypeScript TeamManager (16-24 days, medium risk)
4. Only consider Go binary if TypeScript alternatives prove unacceptable AND orphan/process/cost problems are solved first

### Value-Add Assessment

**Is this design value-add for the TUI?**

**YES** — The core goal (non-blocking orchestration) delivers significant UX value.

**Is this design non-breaking for the TUI?**

**MOSTLY** — ONE breaking change (ProcessRegistry bypass) affects process visibility and cleanup.

**Is the Go binary approach justified?**

**NOT YET** — Simpler alternatives haven't been validated. Go binary should be last resort, not first choice.

---

## Appendix A: Scout Verification Results

### Scout 1: spawnAgent.ts
- **Path**: `packages/tui/src/mcp/tools/spawnAgent.ts`
- **Lines**: ~250
- **Spawn mechanism**: `child_process.spawn('claude', [...])`
- **effortLevel injection**: Lines 172-186
  ```typescript
  if (agentConfig.effortLevel) {
    env['CLAUDE_CODE_EFFORT_LEVEL'] = agentConfig.effortLevel;
  }
  ```
- **Cost tracking**: Parses JSON output from CLI, extracts `cost_usd`
- **ProcessRegistry integration**: Calls `processRegistry.register()` after spawn
- **Status**: VERIFIED WORKING

### Scout 2: ProcessRegistry
- **Path**: `packages/tui/src/spawn/processRegistry.ts`
- **Lines**: ~120
- **Structure**: `Map<string, { process: ChildProcess, info: ProcessInfo }>`
- **Cleanup sequence**:
  ```typescript
  process.kill('SIGTERM');
  await setTimeout(5000);
  if (stillRunning) process.kill('SIGKILL');
  ```
- **Limitation**: Only tracks Node.js `ChildProcess` instances
- **Cannot track**: External PIDs from Go binary
- **Status**: VERIFIED WORKING, scope limited

### Scout 3: agents-index.json
- **Path**: `.claude/agents/agents-index.json`
- **effortLevel present**: YES, for all 7 Opus agents
- **Values**:
  - HIGH: einstein, beethoven, python-architect, staff-architect-critical-review
  - MEDIUM: architect, planner, mozart
- **Fields verified**:
  - `model`: present for all
  - `effortLevel`: present for Opus tier
  - `can_spawn`: present with relationships
  - `spawned_by`: present with relationships
- **Status**: VERIFIED READY for team system

### Scout 4: Existing Orchestration Patterns
- **Searched**: `orchestrat`, `wave`, `team`, `parallel.*spawn`
- **Results**:
  - review-orchestrator: Uses `mcp__goyoke__spawn_agent`, sequential spawns
  - Mozart prompt: References spawning Einstein + Staff-Architect
  - No wave scheduling infrastructure
  - No dependency DAG implementation
  - No team directory structure
- **Status**: GREENFIELD — no existing patterns to migrate

---

## Appendix B: Critical Files Quick Reference

| File | Why It Matters | Change Needed |
|------|----------------|---------------|
| `packages/tui/src/spawn/processRegistry.ts` | PID tracking | Extend for external PIDs OR use unchanged with TeamManager |
| `packages/tui/src/mcp/tools/spawnAgent.ts` | Agent spawning | TeamManager would call this |
| `packages/tui/src/hooks/useClaudeQuery.ts` | Event loop blocking | `for await` at line 713 is the blocking point |
| `packages/tui/src/store/types.ts` | Store types | Add Team, Wave, TeamMember types |
| `.claude/agents/agents-index.json` | Agent config | Already has effortLevel, can_spawn |
| `packages/tui/src/store/index.ts` | Store setup | Add teamsSlice |
| `packages/tui/src/components/StatusBar.tsx` | Status display | Add team status indicator |

---

## Appendix C: Glossary

| Term | Definition |
|------|------------|
| **Wave** | A group of agents that can run in parallel (no dependencies between them) |
| **Team** | Collection of agents organized into waves to accomplish a task |
| **ProcessRegistry** | TUI's internal tracking of spawned child processes |
| **Orphan** | Process that continues running after its parent has terminated |
| **Headless mode** | Claude CLI running without interactive terminal |
| **stdin schema** | JSON structure defining input contract for an agent |
| **stdout schema** | JSON structure defining output contract from an agent |
| **goyoke-team-run** | Proposed Go binary for background team execution |
| **TeamManager** | Proposed TypeScript alternative to Go binary |

---

## Metadata

```yaml
analysis_id: parallel-orchestration-2026-02-06
version: 2.0 (comprehensive)
trigger: /braintrust
document_reviewed: PARALLEL-ORCHESTRATION-DESIGN.md
scouts_dispatched: 4 (haiku)
  - spawnAgent.ts verification
  - processRegistry.ts verification
  - agents-index.json verification
  - existing orchestration patterns search
analysts:
  einstein:
    model: opus
    focus: theoretical analysis
    sections: first principles, claim validity, assumptions, novel failure modes
  staff_architect:
    model: opus
    focus: practical 7-layer review
    sections: assumptions, dependencies, failures, cost-benefit, testing, smells, readiness
synthesizer: beethoven
  model: opus
  output: convergence, divergence resolution, unified recommendations
orchestrator: mozart
  model: opus
  phases: intake, reconnaissance, dispatch, collection
total_cost_usd: ~4.10
total_tokens: ~134,000
duration_ms: ~885,000 (14.75 minutes)

verdicts:
  architecture: coherent but over-engineered
  implementation_readiness: blocked (missing specs)
  risk_profile: high (7+ unaddressed failure modes)
  tui_compatibility: one breaking change (ProcessRegistry bypass)

recommendations:
  priority_1: validate SDK concurrent sessions (2 hours)
  priority_2: add missing specifications if proceeding
  priority_3: consider TypeScript TeamManager (60% less effort)
  priority_4: add Phase 0 ProcessRegistry unification

alternatives_proposed:
  optimal: concurrent SDK queries (~100 lines TypeScript)
  fallback: TypeScript TeamManager (8-12 days)
  original: Go binary (25-40 days, not recommended)

critical_gaps:
  - orphan detection unspecified
  - TUI ↔ Go binary IPC unspecified
  - hook inheritance in headless mode unvalidated
  - team-level budget controls absent
  - testing infrastructure unspecified
```
