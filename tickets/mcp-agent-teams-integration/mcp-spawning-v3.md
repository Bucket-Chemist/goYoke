# MCP-Based Agent Spawning Architecture v3

**Document ID**: `mcp-spawning-architecture-v3-2026-02-04`
**Status**: APPROVED FOR IMPLEMENTATION
**Braintrust Session**: Mozart → Einstein + Staff-Architect → Beethoven
**Date**: 2026-02-04

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Braintrust Analysis Sources](#2-braintrust-analysis-sources)
3. [Einstein Theoretical Analysis (Full)](#3-einstein-theoretical-analysis-full)
4. [Staff-Architect Practical Review (Full)](#4-staff-architect-practical-review-full)
5. [Beethoven Synthesis](#5-beethoven-synthesis)
6. [Unified Architectural Framework](#6-unified-architectural-framework)
7. [Implementation Tickets](#7-implementation-tickets)
8. [Bash Slicing Script](#8-bash-slicing-script)

---

## 1. Executive Summary

### The Problem

Claude Code's Task tool is only available at nesting level 0 (router). Subagents cannot spawn sub-subagents, breaking orchestrator patterns like Braintrust and review-orchestrator.

### The Solution

A **hybrid Task/CLI approach** that:
- Preserves Task() for Level 0→1 spawning (native, fast)
- Uses `claude -p` via MCP tools for Level 1→2+ spawning (full tool access)
- Hook-based level detection blocks Task() at Level 1+ with guidance
- Schema-driven JSON I/O for all CLI spawning

### Critical Path

**Phase 0 MUST pass before any implementation:**
1. Verify MCP tools accessible from Task()-spawned subagents
2. Verify CLI stdin piping produces valid JSON output
3. Create mock CLI infrastructure for testing

### Timeline

**16-21 days** (3 weeks with buffer), not the originally estimated 12 days.

### Verdict

**CONDITIONAL APPROVAL** - Proceed only after Phase 0 verification passes.

---

## 2. Braintrust Analysis Sources

This document synthesizes analysis from two Opus-tier agents:

| Agent | Role | Focus | Token Usage |
|-------|------|-------|-------------|
| **Einstein** | Theoretical Analysis | Paradigm soundness, failure modes, schema versioning | 88,192 tokens |
| **Staff-Architect** | Practical Review | 7-layer framework, code review, implementation roadmap | 86,001 tokens |

Both analyses were conducted in parallel and synthesized by Beethoven (this document).

---

## 3. Einstein Theoretical Analysis (Full)

### 3.1 Hybrid vs Full-CLI Paradigm

#### 3.1.1 Theoretical Assessment

The hybrid approach (Task for Level 0→1, CLI for Level 1+ spawning) is **theoretically sound but introduces conceptual complexity**.

**Semantic differences between Task-spawned and CLI-spawned agents:**

| Dimension | Task() Subagent | CLI-Spawned Agent |
|-----------|-----------------|-------------------|
| Process model | In-process, managed by Claude Code runtime | Independent OS process |
| Context inheritance | Implicit - conversation state, tool history, working memory | Explicit - only what is serialized in prompt |
| Session continuity | Same session ID, unified metrics | New session ID, fragmented metrics |
| Tool access | Restricted by nesting level | Full (Level 0 equivalent) |
| Cost tracking | Unified | Must be aggregated externally |
| Lifecycle | Managed by parent | Independent, requires explicit cleanup |

**The fundamental tension**: Task-spawned agents are semantically "children" within a conversation tree; CLI-spawned agents are semantically "collaborators" in a distributed system. The hybrid approach forces orchestrators to reason about two different mental models.

**Should the system treat them uniformly?**

Einstein argues **no** - but the distinction should be made **explicit and principled**:

1. **Task() agents** (Level 0→1) inherit conversational context and operate within the session boundary
2. **CLI agents** (Level 1+) are independent workers that communicate via structured I/O contracts

The Critical Review document correctly identifies this as a "paradigm shift, not workaround." The v2 architecture document's full-CLI approach has **stronger theoretical foundations** because it treats all agent spawning uniformly through one mechanism.

**Confidence level**: HIGH

#### 3.1.2 Recommendation

The **full-CLI approach is theoretically cleaner** for:
1. **Uniform spawning semantics**: All spawned agents behave identically regardless of depth
2. **Explicit context passing**: Forces disciplined contract design via schemas
3. **Simpler mental model**: Orchestrators don't need to reason about which mechanism to use
4. **Better isolation**: Failures in child agents don't corrupt parent state

**However**, the hybrid approach has pragmatic advantages:
1. **Lower latency** for Level 0→1 (no CLI cold start)
2. **Implicit context** reduces prompt engineering burden
3. **Unified cost tracking** within Claude Code session

**Synthesis**: Implement the **hybrid approach with explicit boundary documentation**. Task() for Level 0→1 (benefiting from native integration), CLI for Level 1+ (accepting the paradigm shift). Document that CLI-spawned agents are fundamentally different entities.

### 3.2 Hook-Based Level Detection

#### 3.2.1 Environment Variable Propagation Reliability

The `GOGENT_NESTING_LEVEL` propagation mechanism has **medium reliability** with specific edge cases.

**Reliable scenarios:**
- TUI spawns via `spawn()` with explicit `env` object
- Go hooks read `os.Getenv()` at process start

**Unreliable scenarios:**

1. **Subshell spawning**: If a CLI agent invokes bash scripts that spawn additional processes, env vars may not propagate unless explicitly exported

2. **SSH/remote execution**: If agents execute commands on remote systems, local env vars are lost

3. **Containerized execution**: Docker/Podman containers don't inherit host env unless explicitly passed via `-e`

4. **Process replacement**: `exec()` family calls preserve env, but some patterns (like `spawn()` with `shell: true`) can have subtle differences

#### 3.2.2 What Happens Without Proper Env Inheritance

If an agent spawns without `GOGENT_NESTING_LEVEL`:

1. **Current validation behavior** (from `task_validation.go`): The hook doesn't currently check nesting level - it only validates opus model and subagent_type
2. **With proposed changes**: `getNestingLevel()` would return 0 (default), allowing Task() invocation when it should be blocked

This is a **fail-open vulnerability**: Missing env vars grant more permissions, not fewer.

#### 3.2.3 Race Conditions and Edge Cases

**Race condition: Parallel spawns at same level**
```
Mozart (L1) spawns:
  - Einstein (L2) via MCP     [GOGENT_NESTING_LEVEL=2]
  - Staff-Architect (L2) via MCP [GOGENT_NESTING_LEVEL=2]
```
No race condition here - each spawn is independent.

**Edge case: Depth overflow**
If nesting exceeds int bounds (theoretical at depth > 2 billion), `strconv.Atoi` fails silently. Mitigation: Cap at reasonable max depth (e.g., 10).

**Edge case: Env var corruption**
If `GOGENT_NESTING_LEVEL=abc`, `strconv.Atoi` returns 0 with error ignored. This fails open.

**Confidence level**: MEDIUM - mechanism is sound but requires defensive coding

#### 3.2.4 Recommendation

1. **Fail-closed default**: If `GOGENT_NESTING_LEVEL` is missing or invalid, assume Level 1+ (conservative)
2. **Add validation**: Check for numeric-only values before parsing
3. **Cap depth**: Maximum 10 levels (no legitimate use case for deeper)
4. **Add telemetry**: Log every case where default is applied to detect propagation failures

### 3.3 Schema Versioning Strategy

#### 3.3.1 Theoretical Framework

Schema versioning for agent I/O contracts must handle three evolution scenarios:

1. **Backward compatible** (additive): New optional fields added
2. **Forward compatible** (conservative consumers): Unknown fields ignored
3. **Breaking changes**: Required field changes, type changes, semantic changes

#### 3.3.2 What Happens with Version Mismatches

**Older orchestrator spawns newer agent:**
- Orchestrator passes input with schema v1.0 fields
- Agent expects schema v1.1 with new required field
- **Failure mode**: Agent validation fails, returns error or behaves incorrectly

**Newer orchestrator spawns older agent:**
- Orchestrator passes input with schema v1.1 fields (including new required field)
- Agent validates against schema v1.0
- **Failure mode**: Agent ignores new field (if validation is lenient) or fails (if strict)

#### 3.3.3 Recommended Versioning Approach

Einstein recommends **dated semver with compatibility metadata**:

```json
{
  "schema_version": "1.2.0",
  "schema_date": "2026-02-04",
  "compatibility": {
    "min_version": "1.0.0",
    "max_version": null
  }
}
```

**Why not pure semver?**
- Agents don't have release cycles like libraries
- Date provides human-readable context
- Semver still communicates breaking vs non-breaking

**Why not hash-based?**
- Hashes provide no compatibility information
- Cannot determine if changes are breaking without comparing
- Poor human readability

**Registry requirement:**
```
schemas/
├── braintrust/
│   ├── input-v1.0.0.schema.json
│   ├── input-v1.1.0.schema.json    # Added optional field
│   ├── input-v2.0.0.schema.json    # Breaking: renamed required field
│   ├── output-v1.0.0.schema.json
│   └── CHANGELOG.md                # Documents changes
├── registry.json                   # Maps workflow -> current version
└── compatibility-matrix.json       # Which versions work together
```

**Confidence level**: HIGH

### 3.4 Session Containerization Model

#### 3.4.1 Structure Assessment

The proposed `sessions/{workflow}-{id}/` structure is **sufficient for single-orchestrator workflows** but **insufficient for nested orchestration**.

**Current proposal:**
```
sessions/
└── braintrust-2026-02-04-abc123/
    ├── session.json
    ├── inputs/
    ├── outputs/
    └── logs/
```

#### 3.4.2 Nested Sessions (Orchestrator Spawns Orchestrator)

Consider: `/ticket` spawns `impl-manager` which spawns `review-orchestrator` which spawns reviewers.

**Flat structure fails:**
```
sessions/
├── ticket-abc123/
│   └── ... impl-manager input
├── impl-manager-def456/     # Where is parent relationship?
│   └── ... review-orchestrator input
└── review-orchestrator-ghi789/   # Lost hierarchy
    └── ... reviewer inputs
```

**Hierarchical structure needed:**
```
sessions/
└── ticket-abc123/              # Epic root
    ├── session.json            # Epic metadata
    ├── impl-manager-def456/    # Child orchestrator
    │   ├── session.json
    │   ├── inputs/
    │   ├── outputs/
    │   └── review-orchestrator-ghi789/  # Grandchild
    │       ├── session.json
    │       ├── inputs/
    │       └── outputs/
    └── aggregate-output.json   # Epic-level synthesis
```

#### 3.4.3 Theoretical Model for Session Ownership and Cleanup

**Ownership rules:**
1. **Creator owns**: The orchestrator that creates a session owns cleanup responsibility
2. **Propagation**: Ownership propagates up on orchestrator failure
3. **Timeout**: Abandoned sessions are owned by the TUI (epic manager)

**Cleanup protocol:**
```
Session lifecycle:
  CREATED → RUNNING → COLLECTING → COMPLETE/ERROR → ARCHIVED/DELETED

Cleanup triggers:
  - Agent completes successfully → Outputs preserved, logs archived
  - Agent fails → Outputs preserved, error logged, parent notified
  - Orchestrator timeout → Children killed, sessions marked ORPHANED
  - TUI shutdown → All active sessions SUSPENDED
  - TUI restart → SUSPENDED sessions resumable (if inputs preserved)
```

**Confidence level**: MEDIUM - hierarchical model adds complexity but is theoretically necessary

#### 3.4.4 Recommendation

1. **Hierarchical session directories** with parent-child nesting
2. **Session references** instead of copies for cross-session data
3. **Cleanup responsibility chain**: child → parent → epic manager → TUI
4. **Retention policy**: Archive after N days, delete after M days

### 3.5 Failure Mode Analysis

#### 3.5.1 Theoretical Failure Modes of CLI Spawning

| Failure Mode | Probability | Impact | Detection Time |
|--------------|-------------|--------|----------------|
| CLI binary not found | LOW | CRITICAL | Immediate |
| CLI startup timeout | MEDIUM | HIGH | 10-30s |
| Model rate limit | MEDIUM | MEDIUM | At API call |
| Memory exhaustion | LOW | CRITICAL | During execution |
| Signal not propagated | MEDIUM | HIGH | At cleanup |
| JSON output corruption | LOW | HIGH | At parse |
| Schema validation failure | MEDIUM | MEDIUM | At spawn return |
| Orphan process after TUI crash | LOW | HIGH | Never (until system reboot) |

#### 3.5.2 Partial Failure Propagation (2 of 3 Parallel Agents Fail)

**Scenario:** review-orchestrator spawns 3 reviewers in parallel:
```
spawn_agents_parallel([
  { agent: "backend-reviewer", ... },
  { agent: "frontend-reviewer", ... },
  { agent: "standards-reviewer", ... }
])
```

**Backend and frontend succeed. Standards fails.**

**Propagation options:**

1. **Fail-fast** (`failFast: true`):
   - First failure cancels all pending
   - Returns immediately with error
   - Partial results lost
   - **Use when**: All-or-nothing operations

2. **Collect-all** (`failFast: false`):
   - Wait for all agents
   - Return array with success/failure per agent
   - Orchestrator decides how to proceed
   - **Use when**: Partial results valuable

3. **Quorum-based** (not in current proposal):
   - Succeed if N of M complete
   - Useful for resilience
   - **Consider adding**: For critical workflows

**Recommended behavior:**
```typescript
interface ParallelSpawnResult {
  status: "all_success" | "partial_success" | "all_failed";
  results: {
    agent: string;
    status: "success" | "error" | "timeout";
    output?: AgentOutput;
    error?: string;
  }[];
  successCount: number;
  failureCount: number;
}
```

#### 3.5.3 Recovery Model

**Recovery tiers:**

1. **Immediate retry**: For transient failures (rate limit, timeout)
   - Max 3 retries with exponential backoff
   - Same agent, same input

2. **Alternative agent**: For systematic failures
   - Try different agent with same capability
   - E.g., haiku-scout fails → try different scout

3. **Escalation**: For persistent failures
   - Notify orchestrator of unrecoverable state
   - Orchestrator decides: proceed without, retry manually, abort

4. **Human intervention**: For critical failures
   - Session marked NEEDS_ATTENTION
   - TUI displays intervention prompt
   - User can retry, skip, or abort

**Session recovery:**
```
On TUI restart:
  1. Scan sessions/ for RUNNING or SUSPENDED
  2. For each:
     a. If inputs and outputs preserved → mark RESUMABLE
     b. If partial → mark NEEDS_REVIEW
     c. If no state → mark LOST
  3. Present user with recovery options
```

**Confidence level**: HIGH

### 3.6 Unified Theoretical Framework

#### 3.6.1 Reconciling Hybrid vs Full-CLI

Einstein proposes a **"Graduated Spawning" model** that embraces the paradigm distinction rather than hiding it:

```
Spawning Tier 1: Task() Native (Level 0 → 1)
├── Semantics: Conversational child
├── Context: Implicit inheritance
├── Lifecycle: Managed by Claude Code
├── Use for: Direct orchestrators (mozart, review-orchestrator)
└── Constraint: Cannot spawn further via Task()

Spawning Tier 2: MCP-CLI (Level 1+ → N)
├── Semantics: Independent worker
├── Context: Explicit via schema-validated JSON
├── Lifecycle: Managed by TUI spawn_agent
├── Use for: Specialist agents, nested orchestrators
└── Constraint: Must produce schema-compliant output
```

**The key insight**: Don't treat CLI spawning as a "workaround" for Task() limitations. Treat it as a **different (and sometimes better) mechanism** for a different class of agent interaction.

#### 3.6.2 Addressing Identified Failure Modes

**Layer 1: Prevention**
- Validate schemas before spawning
- Check `can_spawn` and `spawned_by` relationships
- Enforce depth limits (max 10)

**Layer 2: Detection**
- Timeout enforcement per agent (with SIGKILL escalation)
- Output schema validation
- Process registry for orphan detection

**Layer 3: Recovery**
- Partial result collection
- Retry with backoff
- Session state preservation for resumption

**Layer 4: Cleanup**
- Signal propagation to children
- Orphan process reaping
- Session directory archival

#### 3.6.3 Actionable Architectural Principles

1. **Explicit over implicit**: CLI spawning forces explicit context passing, which is good for reliability and debugging

2. **Fail-closed for security**: Missing `GOGENT_NESTING_LEVEL` should block Task(), not allow it

3. **Schema-first contracts**: All agent I/O must be schema-defined before implementation

4. **Hierarchical session ownership**: Parent sessions own child sessions, cleanup propagates upward

5. **Partial success as first-class**: The system must handle partial parallel execution results gracefully

6. **Unified telemetry**: Both Task() and CLI spawns must emit compatible telemetry for unified tracking

7. **Graceful degradation**: System should function (with reduced capability) even if schema validation fails

#### 3.6.4 Assumptions Requiring Empirical Verification

| Assumption | Verification Method | Priority |
|------------|---------------------|----------|
| MCP tools available to Level 1 subagents | Spawn via Task(), attempt MCP call | CRITICAL |
| Task() unavailable at Level 1+ | Explicit test in controlled environment | CRITICAL |
| CLI JSON output is reliable | Stress test with large outputs | HIGH |
| Signal propagation works correctly | Test SIGTERM to parent with children | HIGH |
| Schema validation overhead acceptable | Benchmark with/without validation | MEDIUM |
| Hierarchical sessions scale reasonably | Test with 5-level deep workflows | MEDIUM |

---

## 4. Staff-Architect Practical Review (Full)

### 4.1 Layer 1: Assumption Validation

#### 4.1.1 MCP Tool Availability in Subagents - UNVERIFIED (CRITICAL)

**Status**: The most critical assumption has **zero empirical evidence**.

**Evidence from codebase**:
- The TUI's MCP server (`packages/tui/src/mcp/server.ts`) registers tools via `createSdkMcpServer`
- These tools (askUser, confirmAction, requestInput, selectOption) are designed for in-process use
- **No verification exists** that these MCP tools are accessible from Task()-spawned subagents

**Required verification before proceeding**:
```typescript
// Test: Create minimal MCP tool
const testMcpPing = tool("test_mcp_ping", "Verify MCP availability", {}, async () => {
  return { content: [{ type: "text", text: "PONG" }] };
});

// Test: Spawn subagent via Task() and have it invoke test_mcp_ping
// If this fails, the entire architecture is invalid
```

#### 4.1.2 spawn() Code Review - INCORRECT (HIGH)

**Issue in proposed code (`mcp-spawning-architecture-v2-2026-02-04.md` lines 326-329)**:
```typescript
const proc = spawn('claude', cliArgs, {
  cwd: process.cwd(),
  shell: true,  // SECURITY RISK
  stdio: ['pipe', 'pipe', 'pipe'],
```

**Problems**:
1. `shell: true` enables shell injection attacks
2. Combined with `'-p', \`"$(cat ${promptFile})"\`` (line 401) - **this is command substitution inside Node.js**, which fails completely

**Correct pattern** (from benchmark tests):
```typescript
const proc = spawn('node', [distPath], {
  stdio: ['pipe', 'pipe', 'pipe'],
  // NO shell: true
});
```

For passing prompt:
```typescript
// Option A: Write to temp file, pass via stdin
await fs.writeFile(promptFile, args.prompt, 'utf-8');
const proc = spawn('claude', ['-p'], { stdio: ['pipe', 'pipe', 'pipe'] });
proc.stdin.write(args.prompt);
proc.stdin.end();

// Option B: Use --stdin flag if available (check claude --help)
```

#### 4.1.3 CLI Flags Verification - PARTIAL (MEDIUM)

**Verified from `claude --help`**:
| Flag | Status | Notes |
|------|--------|-------|
| `-p` / `--print` | VERIFIED | Non-interactive mode |
| `--output-format json` | VERIFIED | Choices: text, json, stream-json |
| `--output-format stream-json` | VERIFIED | Real-time streaming |
| `--permission-mode delegate` | VERIFIED | Choices: acceptEdits, bypassPermissions, default, delegate, dontAsk, plan |
| `--allowedTools` | VERIFIED | Space or comma-separated |
| `--json-schema` | VERIFIED | For structured output validation |
| `--max-budget-usd` | VERIFIED | Cost control |
| `--session-id` | VERIFIED | Must be valid UUID |
| `--dangerously-skip-permissions` | VERIFIED | Bypasses ALL checks |

**Not verified/not found**:
- `--max-turns` - **NOT FOUND** in help output (proposal uses this)

#### 4.1.4 Store Interface Compatibility - BREAKING CHANGE (MEDIUM)

**Current Agent interface** (`packages/tui/src/store/types.ts` lines 38-51):
```typescript
export interface Agent {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: {
    input: number;
    output: number;
  };
}
```
**Fields: 10**

**Proposed Agent interface** (`mcp-spawning-architecture-v2` lines 117-154):
```typescript
interface Agent {
  id: string;
  agentType: string;
  epicId: string;
  parentId: string | null;
  depth: number;
  childIds: string[];
  spawnMethod: "task" | "mcp-cli";
  spawnedBy: string;
  prompt: string;
  model: "haiku" | "sonnet" | "opus";
  status: "queued" | "spawning" | "running" | "streaming" | "complete" | "error" | "timeout";
  pid?: number;
  queuedAt: number;
  startTime?: number;
  endTime?: number;
  output?: string;
  streamBuffer?: string;
  error?: string;
  tokenUsage?: { input: number; output: number };
  cost?: number;
  turns?: number;
  toolCalls?: number;
}
```
**Fields: 24**

**Impact**:
- Backward-incompatible change
- Requires migration of existing store slices
- Components like `AgentTree.tsx` and `AgentDetail.tsx` need updates

### 4.2 Layer 2: Dependency Analysis

#### 4.2.1 External Dependencies - MISSING VALIDATION (CRITICAL)

**No validation exists for**:

| Dependency | Required | Current Check | Risk |
|------------|----------|---------------|------|
| `claude` CLI in PATH | Yes | None | spawn() fails silently |
| Node.js version | 22+ (per package.json) | None | API compatibility |
| `/tmp` writable | Yes | None | Prompt file creation fails |
| `CLAUDE_PROJECT_DIR` | Optional | None | Context loss |
| `XDG_DATA_HOME` | Optional | Fallback exists | Hook telemetry fails |

**Required pre-flight check**:
```typescript
async function validateEnvironment(): Promise<{ok: boolean, errors: string[]}> {
  const errors: string[] = [];

  // Check claude CLI
  try {
    execSync('which claude', { stdio: 'pipe' });
  } catch {
    errors.push("claude CLI not found in PATH");
  }

  // Check /tmp writable
  try {
    const testFile = `/tmp/claude-spawn-test-${Date.now()}`;
    await fs.writeFile(testFile, 'test');
    await fs.unlink(testFile);
  } catch {
    errors.push("/tmp not writable");
  }

  return { ok: errors.length === 0, errors };
}
```

#### 4.2.2 Package Dependencies - ADEQUATE

Current dependencies in `package.json`:
- `@anthropic-ai/claude-agent-sdk: ^0.2.29` - for MCP server creation
- `chokidar: ^4.0.0` - for file watching (telemetry)
- `zod: ^4.3.6` - for schema validation
- Node.js `child_process` - built-in, no additional dependency needed

### 4.3 Layer 3: Failure Modes

#### 4.3.1 Process Orphaning - INADEQUATE MITIGATION (CRITICAL)

**Current shutdown handler** (`packages/tui/src/lifecycle/shutdown.ts`):
- Handles SIGINT, SIGTERM, uncaughtException
- Has `registerChildProcessCleanup()` function
- BUT: Only handles controlled shutdowns

**Missing**:
1. **No process registry** - spawned processes not tracked globally
2. **No SIGKILL escalation** - if SIGTERM doesn't work within timeout
3. **No cleanup on unhandled promise rejection** (explicitly continues without exit)

**Proposed fix**:
```typescript
// Global process registry
const activeProcesses = new Map<string, ChildProcess>();

// Enhanced shutdown
async function cleanupAllProcesses(): Promise<void> {
  const timeout = 5000; // 5s for graceful shutdown

  for (const [id, proc] of activeProcesses) {
    proc.kill('SIGTERM');
  }

  // Wait for graceful shutdown
  await new Promise(resolve => setTimeout(resolve, timeout));

  // Force kill any remaining
  for (const [id, proc] of activeProcesses) {
    if (!proc.killed) {
      proc.kill('SIGKILL');
    }
  }

  activeProcesses.clear();
}

// Register with lifecycle
onShutdown(cleanupAllProcesses);
```

#### 4.3.2 Signal Propagation - NOT IMPLEMENTED (HIGH)

**Issue**: Ctrl+C in TUI doesn't propagate to spawned CLI processes.

Current `setupSignalHandlers()` only handles TUI process signals, not child processes.

**Required**: Process group management
```typescript
const proc = spawn('claude', cliArgs, {
  stdio: ['pipe', 'pipe', 'pipe'],
  detached: false, // Keep in same process group
});

// On TUI signal, forward to children
process.on('SIGINT', () => {
  for (const [id, proc] of activeProcesses) {
    proc.kill('SIGINT');
  }
});
```

#### 4.3.3 Memory Leak from Unbounded Buffers - NOT ADDRESSED (HIGH)

**Issue in proposed code**:
```typescript
let stdout = '';
proc.stdout.on('data', (data) => {
  stdout += chunk;  // Unbounded growth
```

For long-running agents with verbose output, this will exhaust memory.

**Required**: Buffer limits
```typescript
const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB
let stdout = '';
let truncated = false;

proc.stdout.on('data', (data) => {
  if (!truncated && stdout.length < MAX_BUFFER_SIZE) {
    stdout += data.toString();
    if (stdout.length >= MAX_BUFFER_SIZE) {
      truncated = true;
      stdout += '\n[OUTPUT TRUNCATED]';
    }
  }
});
```

### 4.4 Layer 4: Cost-Benefit Analysis

#### 4.4.1 Cold Start Latency

| Metric | Estimate | Acceptable? |
|--------|----------|-------------|
| CLI startup | 2-5s | Yes |
| Context loading (CLAUDE.md, hooks) | 1-3s | Yes |
| First API call | 2-5s | Yes |
| **Total cold start** | **5-10s** | **Marginal** |

For Braintrust with 4 agents (Mozart, Einstein, Staff-Architect, Beethoven):
- Sequential: 20-40s startup overhead
- Parallel (Einstein + Staff-Architect): 15-30s startup overhead

**Verdict**: Acceptable for complex workflows, would be problematic if every agent spawn adds 10s.

#### 4.4.2 Memory Overhead

| Component | Per-Process | With 4 Concurrent Agents |
|-----------|-------------|-------------------------|
| Node.js base | ~50MB | 200MB |
| Claude CLI | ~50-100MB | 200-400MB |
| Stream buffers | ~1-10MB | 4-40MB |
| **Total** | ~100-160MB | **400-640MB** |

**Verdict**: Acceptable on modern systems, may need limits for constrained environments.

#### 4.4.3 Development Effort Assessment

**Proposed**: 5-7 days (in v2 doc), 10-12 days (in critical review)

**Staff-Architect assessment based on codebase state**:

| Phase | Proposed | Realistic | Reason |
|-------|----------|-----------|--------|
| Phase 0: Verification | 2 days | 2-3 days | MCP testing critical |
| Phase 1: Foundation | 4 days | 6-8 days | Schema + session + spawn tool |
| Phase 2: Integration | 3 days | 4-5 days | Store migration, components |
| Phase 3: Testing | 3 days | 4-5 days | Mock CLI, integration tests |
| **Total** | **12 days** | **16-21 days** | +50% buffer |

### 4.5 Layer 5: Testing Strategy

#### 4.5.1 Unit Testing Without API Credits - UNDEFINED (CRITICAL)

**No strategy exists** for testing `spawn_agent` without consuming API credits.

**Required**: Mock CLI approach
```typescript
// tests/mocks/mockClaude.ts
export function createMockClaude(behavior: MockBehavior): string {
  // Returns path to mock script
  const script = `#!/bin/bash
    # Read stdin
    input=$(cat)
    # Output based on behavior
    echo '{"type": "result", "subtype": "success", "total_cost_usd": 0.01}'
  `;
  // Write to temp file, make executable
  return tempScriptPath;
}

// In test
vi.mock('child_process', () => ({
  spawn: (cmd, args, opts) => {
    if (cmd === 'claude') {
      return spawn(mockClaudePath, [], opts);
    }
    return originalSpawn(cmd, args, opts);
  }
}));
```

#### 4.5.2 Integration Test Strategy - UNDEFINED

Current test infrastructure:
- `vitest` for unit tests
- `ink-testing-library` for component tests
- Performance benchmarks exist (`tests/performance/`)

**Missing**:
- E2E test for full spawn workflow
- Test for timeout/kill paths
- Test for error propagation

#### 4.5.3 Timeout/Kill Path Testing

**Challenge**: Hard to test without slow tests.

**Approach**:
```typescript
describe('spawn_agent timeout handling', () => {
  it('should kill process after timeout', async () => {
    // Use mock that delays forever
    const result = await spawnAgentWithMock({
      mockBehavior: 'hang',
      timeout: 100, // Short timeout for test
    });

    expect(result.status).toBe('timeout');
    expect(mockProcess.killed).toBe(true);
  });
});
```

### 4.6 Layer 6: Architecture Smells

#### 4.6.1 spawn_agent God Object - 11 RESPONSIBILITIES (HIGH)

The proposed `spawn_agent` tool does:
1. Validate input schema
2. Track epic/parent relationships
3. Update parent's children list
4. Write prompt to temp file
5. Build CLI arguments
6. Spawn process
7. Handle streaming output
8. Parse NDJSON
9. Manage timeout
10. Update store on completion
11. Clean up temp file

**Decomposition proposal**:
```
spawn_agent (orchestrator)
  ├── InputValidator
  ├── HierarchyManager (epic, parent, children)
  ├── CliBuilder (args, env vars)
  ├── ProcessRunner (spawn, streams, timeout)
  ├── OutputParser (NDJSON → structured result)
  └── CleanupManager (temp files, process registry)
```

#### 4.6.2 Circular Reference Tool ↔ Store - PRESENT (HIGH)

**In proposed code**:
```typescript
// spawnAgent.ts
import { useAgentsStore } from "../store/slices/agents";

export const spawnAgentTool = tool(
  "spawn_agent",
  // ...
  async (args, context) => {
    const store = useAgentsStore.getState();
    // ...
  }
);
```

**Problem**: MCP tool directly imports and uses Zustand store, creating tight coupling.

**Better pattern**: Dependency injection
```typescript
interface SpawnAgentDeps {
  getAgentsStore: () => AgentsStore;
  getEpicsStore: () => EpicsStore;
  processRegistry: ProcessRegistry;
}

export function createSpawnAgentTool(deps: SpawnAgentDeps) {
  return tool("spawn_agent", /* ... */, async (args) => {
    const store = deps.getAgentsStore();
    // ...
  });
}
```

#### 4.6.3 Epic Concept - POTENTIALLY PREMATURE (MEDIUM)

**Current need**: Track parent-child relationships for Braintrust.

**Proposed**: Full Epic abstraction with separate store slice.

**Assessment**:
- Epic tracking may be premature abstraction
- Start with simpler `rootAgentId` approach
- Promote to Epic when pattern proves valuable

### 4.7 Layer 7: Contractor Readiness

#### 4.7.1 TypeScript Interfaces - INCOMPLETE (HIGH)

**Missing types in proposal**:
```typescript
// Not defined:
interface SpawnArgs { /* ... */ }
interface AgentResult { /* ... */ }
interface TreeNodeProps { /* ... */ }
interface AgentConfig { /* ... */ }
interface ValidationResult { /* ... */ }
```

**Also missing**:
- Error types for spawn failures
- Event types for streaming
- Schema validation result types

#### 4.7.2 Acceptance Criteria - NOT DEFINED (HIGH)

**No acceptance criteria for**:
- What constitutes "spawn success"?
- Timeout behavior (clean vs. dirty shutdown)?
- Store state after spawn error?
- Partial output handling?

**Required for each component**:
```markdown
## spawn_agent Tool Acceptance Criteria

GIVEN a valid spawn request
WHEN the agent completes successfully
THEN:
  - Store contains agent with status "complete"
  - Output field contains final result
  - Cost/tokens are recorded
  - Temp file is cleaned up
  - Process is removed from registry

GIVEN a valid spawn request
WHEN the agent times out
THEN:
  - Process receives SIGTERM
  - After 5s grace, SIGKILL if needed
  - Store contains agent with status "timeout"
  - Error field contains timeout message
```

#### 4.7.3 Rollback Plan - NOT DEFINED (CRITICAL)

**Required rollback plan**:

1. **Feature flag**: `GOGENT_MCP_SPAWN_ENABLED=false`
2. **Code isolation**: spawn_agent in separate module, easy to disable
3. **Store compatibility**: New fields optional, old code still works
4. **Hook bypass**: `GOGENT_NESTING_LEVEL` check skippable
5. **Revert checklist**:
   - Remove spawn_agent from MCP server registration
   - Revert store types to 10-field Agent
   - Revert hook to pre-nesting-level state
   - Remove session directory code

### 4.8 Staff-Architect Severity Summary

#### CRITICAL Blockers (Must Resolve Before Implementation)

| ID | Issue | Resolution |
|----|-------|------------|
| C1 | MCP tool availability in subagents unverified | Create test script, verify empirically |
| C2 | `shell: true` + command substitution incorrect | Fix spawn pattern to use stdin piping |
| C3 | No process orphan cleanup | Implement global process registry + SIGKILL escalation |
| C4 | No mock CLI test strategy | Create mock CLI infrastructure before implementing |
| C5 | No rollback plan defined | Document feature flag, revert checklist, compatibility layer |

#### HIGH Priority (Address in Design Phase)

| ID | Issue | Resolution |
|----|-------|------------|
| H1 | Signal propagation missing | Forward signals to child processes |
| H2 | Unbounded stream buffers | Add MAX_BUFFER_SIZE limit |
| H3 | spawn_agent god object | Decompose into 6 focused modules |
| H4 | Circular tool ↔ store reference | Use dependency injection |
| H5 | TypeScript interfaces incomplete | Define all interfaces before coding |
| H6 | Acceptance criteria missing | Write AC for each component |
| H7 | `--max-turns` flag not found | Verify or remove from implementation |

#### MEDIUM Priority (Address During Iteration)

| ID | Issue | Resolution |
|----|-------|------------|
| M1 | Store interface breaking change | Use adapter pattern for migration |
| M2 | Epic abstraction premature | Start with simpler rootAgentId |
| M3 | Environment validation missing | Add pre-flight checks |
| M4 | Development estimate optimistic | Budget 16-21 days, not 12 |
| M5 | JSON vs stream-json decision | Start with json, add streaming later |
| M6 | Session directory retention | Define retention policy before implementation |

---

## 5. Beethoven Synthesis

### 5.1 Convergence Points (Both Analysts Agree)

| Finding | Einstein | Staff-Architect | Resolution |
|---------|----------|-----------------|------------|
| MCP availability MUST be verified first | CRITICAL assumption | Phase 0 gate | **Empirical test before any code** |
| Hybrid approach is sound | "Graduated Spawning" model | Implicit in roadmap | **Task for 0→1, CLI for 1→2+** |
| Fail-closed for security | Missing env = assume Level 1+ | Process registry + cleanup | **Defense in depth** |
| Schema-driven I/O is necessary | Dated semver recommended | Zod infrastructure exists | **Build on existing Zod** |
| Timeline is optimistic | Not quantified | 16-21 days realistic | **Budget 3 weeks** |

### 5.2 Divergence Points (Resolved)

| Tension | Einstein View | Staff-Architect View | Synthesis |
|---------|---------------|---------------------|-----------|
| Full-CLI vs Hybrid | Full-CLI theoretically cleaner | Hybrid pragmatically better | **Hybrid with explicit documentation** |
| Epic abstraction | Hierarchical sessions needed | Potentially premature | **Start simple, promote if valuable** |
| stream-json vs json | Not addressed | Start with json | **json for Phase 1, stream-json Phase 2** |

### 5.3 Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Hybrid over full-CLI** | Preserves Task() benefits (lower latency, implicit context) for Level 0→1 |
| **Fail-closed defaults** | Security: unknown level blocks Task(), doesn't allow it |
| **json over stream-json initially** | Simplicity: single parse, no NDJSON complexity |
| **Adapter pattern for store** | Backward compatibility: existing code continues working |
| **Dependency injection for tools** | Testability: mock store in unit tests |
| **Hierarchical sessions deferred** | Start simple: flat sessions with parentId references |

### 5.4 Critical Files for Implementation

| File | Purpose | Modification Needed |
|------|---------|---------------------|
| `packages/tui/src/mcp/server.ts` | MCP server registration | Add spawn_agent tool |
| `packages/tui/src/store/types.ts` | Agent interface | Extend with optional fields |
| `cmd/gogent-validate/main.go` | Hook validation | Add nesting level check |
| `packages/tui/src/lifecycle/shutdown.ts` | Process cleanup | Add process registry |
| `packages/tui/src/store/slices/agents.ts` | Store slice | Add hierarchy tracking |

---

## 6. Unified Architectural Framework

### 6.1 The "Graduated Spawning" Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                    GRADUATED SPAWNING MODEL                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  TIER 1: Task() Native (Level 0 → 1)                               │
│  ├── Semantics: Conversational child                               │
│  ├── Context: Implicit inheritance                                 │
│  ├── Lifecycle: Managed by Claude Code                             │
│  ├── Use for: Direct orchestrators (mozart, review-orchestrator)   │
│  └── Constraint: Cannot spawn further via Task()                   │
│                                                                     │
│  TIER 2: MCP-CLI (Level 1+ → N)                                    │
│  ├── Semantics: Independent worker                                 │
│  ├── Context: Explicit via schema-validated JSON                   │
│  ├── Lifecycle: Managed by TUI spawn_agent                         │
│  ├── Use for: Specialist agents, nested orchestrators              │
│  └── Constraint: Must produce schema-compliant output              │
│                                                                     │
│  ENFORCEMENT:                                                       │
│  ├── gogent-validate blocks Task() at Level 1+                     │
│  ├── Hook injects: "Use MCP spawn_agent instead"                   │
│  └── Fail-closed: Missing GOGENT_NESTING_LEVEL = assume Level 1    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 Correct Implementation Patterns

#### 6.2.1 spawn_agent: Correct CLI Invocation

```typescript
// WRONG (from v2 doc)
const proc = spawn('claude', ['-p', `"$(cat ${promptFile})"`], {
  shell: true,  // Security risk
});

// CORRECT
const proc = spawn('claude', ['-p', '--output-format', 'json'], {
  stdio: ['pipe', 'pipe', 'pipe'],
  env: {
    ...process.env,
    GOGENT_NESTING_LEVEL: String(currentLevel + 1),
    GOGENT_PARENT_AGENT: parentAgentId,
  },
});

// Pass prompt via stdin
proc.stdin.write(prompt);
proc.stdin.end();

// Collect output with buffer limit
const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB
let stdout = '';
let truncated = false;

proc.stdout.on('data', (chunk) => {
  if (!truncated && stdout.length < MAX_BUFFER_SIZE) {
    stdout += chunk.toString();
    if (stdout.length >= MAX_BUFFER_SIZE) {
      truncated = true;
      stdout += '\n[OUTPUT TRUNCATED]';
    }
  }
});
```

#### 6.2.2 gogent-validate: Nesting Level Check

```go
// Add to cmd/gogent-validate/main.go

// Fail-closed: missing or invalid = assume Level 1 (blocked)
func getNestingLevel() int {
    levelStr := os.Getenv("GOGENT_NESTING_LEVEL")
    if levelStr == "" {
        return 1 // Fail-closed: assume nested
    }
    level, err := strconv.Atoi(levelStr)
    if err != nil || level < 0 || level > 10 {
        return 1 // Fail-closed: invalid = assume nested
    }
    return level
}

// In main validation logic
if event.ToolName == "Task" {
    nestingLevel := getNestingLevel()
    if nestingLevel > 0 {
        return blockWithReason(
            "Task() blocked at nesting level %d. Use MCP spawn_agent instead.",
            nestingLevel,
        )
    }
}
```

#### 6.2.3 Store: Backward-Compatible Extension

```typescript
// packages/tui/src/store/types.ts

// Existing (keep as-is for compatibility)
export interface AgentV1 {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: { input: number; output: number };
}

// Extended (all new fields optional)
export interface Agent extends AgentV1 {
  agentType?: string;
  epicId?: string;
  depth?: number;
  childIds?: string[];
  spawnMethod?: "task" | "mcp-cli";
  spawnedBy?: string;
  pid?: number;
  output?: string;
  error?: string;
  cost?: number;
}

// Adapter for legacy code
export function ensureAgentV2(agent: AgentV1): Agent {
  return {
    ...agent,
    agentType: agent.description || 'unknown',
    epicId: 'legacy',
    depth: 1,
    childIds: [],
    spawnMethod: 'task',
    spawnedBy: 'router',
  };
}
```

### 6.3 Rollback Plan

#### Feature Flag
```bash
# Disable MCP spawning entirely
export GOGENT_MCP_SPAWN_ENABLED=false
```

#### Revert Checklist
1. **MCP Server**: Remove `spawn_agent` from tools array in `server.ts`
2. **Hook**: Revert gogent-validate to pre-nesting-level state
3. **Store**: Keep extended interface (backward compatible)
4. **Orchestrators**: Revert to Task() (will fail at Level 1 - acceptable)

#### Fallback: Flat Coordination
If MCP verification fails completely:
```
Router receives /braintrust
├── Router creates "braintrust plan" document
├── Router spawns Mozart via Task()
│   └── Mozart returns: "Spawn einstein with {...}, spawn staff-architect with {...}"
├── Router parses plan, spawns Einstein via Task()
├── Router parses plan, spawns Staff-Architect via Task()
├── Router collects both outputs
├── Router spawns Beethoven via Task() with both outputs
└── Router returns final synthesis
```

---

## 7. Implementation Tickets

The following section contains tickets in a format compatible with the ticket system.
Each ticket is delimited by `### ---TICKET-START---` and `### ---TICKET-END---` markers.

Use the bash script in Section 8 to extract individual ticket files.

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-001
title: MCP Tool Availability Verification (GATE)
description: Verify that MCP tools registered in TUI are accessible from Task()-spawned subagents. This is a CRITICAL GATE - failure invalidates the entire architecture.
status: pending
time_estimate: 2h
dependencies: []
phase: 0
tags: [gate, critical, verification, phase-0]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
gate_decision: null
---
```

# MCP-SPAWN-001: MCP Tool Availability Verification (GATE)

## Description

Verify that MCP tools registered in the TUI's MCP server are accessible from Task()-spawned subagents. This is the **most critical assumption** in the entire architecture and has **zero empirical evidence**.

**Source**: Einstein Analysis §3.6.4, Staff-Architect Analysis §4.1.1

## Why This Matters

The entire MCP-based agent spawning architecture depends on this assumption:
- If MCP tools ARE available to subagents → Architecture is valid, proceed
- If MCP tools are NOT available → Architecture is INVALID, must use flat coordination fallback

## Task

1. Create a minimal test MCP tool (`test_mcp_ping`)
2. Register it in the TUI's MCP server
3. Spawn a subagent via Task()
4. Have the subagent attempt to invoke `mcp__gofortress__test_mcp_ping`
5. Document the exact result

## Files

- `packages/tui/src/mcp/tools/testMcpPing.ts` — Test tool implementation
- `packages/tui/src/mcp/server.ts` — Register test tool
- `.claude/tmp/mcp-verification-result.json` — Results documentation

## Implementation

### Test Tool (`packages/tui/src/mcp/tools/testMcpPing.ts`)

```typescript
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";

/**
 * Minimal MCP tool for verifying subagent accessibility.
 * Returns PONG with timestamp to prove invocation succeeded.
 */
export const testMcpPing = tool(
  "test_mcp_ping",
  "Verify MCP tool accessibility from subagents. Returns PONG with timestamp.",
  {
    echo: z.string().optional().describe("Optional string to echo back"),
  },
  async (args) => {
    const timestamp = new Date().toISOString();
    const response = {
      status: "PONG",
      timestamp,
      echo: args.echo || null,
      message: "MCP tool successfully invoked",
    };

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(response, null, 2),
        },
      ],
    };
  }
);
```

### Register in Server (`packages/tui/src/mcp/server.ts`)

```typescript
// Add import
import { testMcpPing } from "./tools/testMcpPing";

// Add to tools array in createSdkMcpServer call
tools: [
  // ... existing tools
  testMcpPing,
],
```

### Verification Script

Run from router level (not as subagent):

```typescript
// Manual test: Spawn subagent and have it try MCP tool
Task({
  description: "MCP availability verification",
  subagent_type: "Explore",
  model: "haiku",
  prompt: `TASK: Verify MCP tool accessibility

1. Attempt to invoke the tool: mcp__gofortress__test_mcp_ping
2. Pass echo parameter: "verification-test"
3. Report the EXACT result:
   - If successful: Copy the full JSON response
   - If failed: Copy the exact error message

DO NOT fabricate results. Report exactly what happens.`
});
```

## Acceptance Criteria

- [ ] `testMcpPing` tool created and compiles without errors
- [ ] Tool registered in MCP server successfully
- [ ] TUI starts without errors with new tool
- [ ] Verification test executed from router level
- [ ] Result documented in `.claude/tmp/mcp-verification-result.json`
- [ ] Gate decision recorded: PROCEED or HALT

## Gate Decision Matrix

| Result | Gate Decision | Next Action |
|--------|---------------|-------------|
| PONG received with timestamp | **PROCEED** | Continue to MCP-SPAWN-002 |
| Tool not found error | **HALT** | Implement flat coordination fallback |
| Permission denied error | **INVESTIGATE** | May be configurable |
| Other error | **INVESTIGATE** | Document and analyze |

## Test Deliverables

- [ ] Test tool created: `packages/tui/src/mcp/tools/testMcpPing.ts`
- [ ] Tool registered in server
- [ ] Manual verification executed
- [ ] Results documented with exact output
- [ ] Gate decision recorded

## Rollback

If this ticket reveals MCP tools are NOT accessible:
1. Document the finding in `.claude/tmp/mcp-verification-result.json`
2. Update architecture to use "flat coordination" model
3. Close all subsequent MCP-SPAWN tickets as "won't fix"
4. Create new ticket series for flat coordination implementation

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-002
title: CLI I/O Verification
description: Verify that claude CLI stdin piping and JSON output work as expected. Tests the practical CLI spawning mechanism.
status: pending
time_estimate: 1h
dependencies: [MCP-SPAWN-001]
phase: 0
tags: [gate, verification, phase-0, cli]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-002: CLI I/O Verification

## Description

Verify that the `claude` CLI supports stdin piping for prompts and produces parseable JSON output. This tests the practical mechanism for CLI-based agent spawning.

**Source**: Staff-Architect Analysis §4.1.2, §4.1.3

## Why This Matters

The spawn_agent tool will pipe prompts via stdin and parse JSON output. If this doesn't work reliably, the implementation approach must change.

## Task

1. Test `claude -p` with stdin piping
2. Test `--output-format json` parsing
3. Test `--permission-mode delegate`
4. Document all verified flags

## Files

- `packages/tui/tests/verification/cli-io-test.sh` — Bash test script
- `.claude/tmp/cli-verification-result.json` — Results documentation

## Implementation

### Test Script (`packages/tui/tests/verification/cli-io-test.sh`)

```bash
#!/bin/bash
# CLI I/O Verification Script
# Tests claude CLI capabilities for spawn_agent implementation

set -e

RESULTS_FILE=".claude/tmp/cli-verification-result.json"
mkdir -p "$(dirname "$RESULTS_FILE")"

echo "Starting CLI I/O Verification..."
echo '{"tests": [], "timestamp": "'$(date -Iseconds)'"}' > "$RESULTS_FILE"

# Test 1: stdin piping works
echo "Test 1: stdin piping..."
STDIN_RESULT=$(echo "Reply with exactly: STDIN_TEST_OK" | claude -p --output-format text 2>&1 || true)
if echo "$STDIN_RESULT" | grep -q "STDIN_TEST_OK"; then
    echo "  ✅ stdin piping works"
    TEST1="pass"
else
    echo "  ❌ stdin piping failed: $STDIN_RESULT"
    TEST1="fail"
fi

# Test 2: JSON output format
echo "Test 2: JSON output format..."
JSON_RESULT=$(echo "Say hello" | claude -p --output-format json 2>&1 || true)
if echo "$JSON_RESULT" | jq -e '.result' > /dev/null 2>&1; then
    echo "  ✅ JSON output parseable"
    TEST2="pass"
else
    echo "  ❌ JSON output not parseable: $JSON_RESULT"
    TEST2="fail"
fi

# Test 3: permission-mode delegate
echo "Test 3: permission-mode delegate..."
PERM_RESULT=$(echo "What is 2+2?" | claude -p --permission-mode delegate --output-format json 2>&1 || true)
if echo "$PERM_RESULT" | jq -e '.result' > /dev/null 2>&1; then
    echo "  ✅ permission-mode delegate works"
    TEST3="pass"
else
    echo "  ❌ permission-mode delegate failed: $PERM_RESULT"
    TEST3="fail"
fi

# Test 4: allowedTools restriction
echo "Test 4: allowedTools restriction..."
TOOLS_RESULT=$(echo "List current directory" | claude -p --permission-mode delegate --allowedTools "Bash(ls:*)" --output-format json 2>&1 || true)
if [ -n "$TOOLS_RESULT" ]; then
    echo "  ✅ allowedTools flag accepted"
    TEST4="pass"
else
    echo "  ❌ allowedTools failed"
    TEST4="fail"
fi

# Write results
cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "tests": {
    "stdin_piping": "$TEST1",
    "json_output": "$TEST2",
    "permission_mode_delegate": "$TEST3",
    "allowed_tools": "$TEST4"
  },
  "verified_flags": [
    "-p / --print",
    "--output-format json",
    "--permission-mode delegate",
    "--allowedTools"
  ],
  "gate_decision": "$([ "$TEST1" = "pass" ] && [ "$TEST2" = "pass" ] && echo "PROCEED" || echo "INVESTIGATE")"
}
EOF

echo ""
echo "Results written to: $RESULTS_FILE"
cat "$RESULTS_FILE" | jq .
```

## Acceptance Criteria

- [ ] Test script created and executable
- [ ] stdin piping test passes
- [ ] JSON output parsing test passes
- [ ] permission-mode delegate test passes
- [ ] allowedTools flag test passes
- [ ] Results documented in `.claude/tmp/cli-verification-result.json`
- [ ] All verified flags documented

## Test Deliverables

- [ ] Test script created: `packages/tui/tests/verification/cli-io-test.sh`
- [ ] Script is executable (`chmod +x`)
- [ ] All 4 tests documented with pass/fail
- [ ] Verified flags list complete
- [ ] Gate decision recorded

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-003
title: Mock CLI Infrastructure
description: Create mock Claude CLI for unit testing spawn_agent without consuming API credits.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-002]
phase: 0
tags: [testing, infrastructure, phase-0]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-003: Mock CLI Infrastructure

## Description

Create a mock Claude CLI that can be used in unit tests to verify spawn_agent behavior without consuming API credits. This is essential for testing timeout handling, error scenarios, and output parsing.

**Source**: Staff-Architect Analysis §4.5.1

## Why This Matters

Without mock CLI infrastructure:
- Every test consumes API credits
- Cannot test timeout scenarios (would need real slow agents)
- Cannot test error scenarios reliably
- CI/CD pipeline cannot run tests

## Task

1. Create mock CLI script generator
2. Create vitest integration helpers
3. Create test scenarios (success, timeout, error)
4. Verify mock works with spawn()

## Files

- `packages/tui/tests/mocks/mockClaude.ts` — Mock CLI generator
- `packages/tui/tests/mocks/mockScenarios.ts` — Predefined scenarios
- `packages/tui/tests/mocks/spawnHelper.ts` — Vitest integration
- `packages/tui/tests/mocks/mockClaude.test.ts` — Self-tests for mock

## Implementation

### Mock CLI Generator (`packages/tui/tests/mocks/mockClaude.ts`)

```typescript
import * as fs from "fs/promises";
import * as path from "path";
import * as os from "os";
import { randomUUID } from "crypto";

export type MockBehavior =
  | "success"
  | "success_slow"
  | "error_max_turns"
  | "error_rate_limit"
  | "timeout"
  | "invalid_json"
  | "partial_output";

export interface MockOptions {
  behavior: MockBehavior;
  delay?: number; // milliseconds
  output?: string; // custom output
  cost?: number;
  tokens?: { input: number; output: number };
}

const MOCK_SCRIPTS: Record<MockBehavior, (opts: MockOptions) => string> = {
  success: (opts) => `#!/bin/bash
# Mock Claude CLI - Success
sleep ${(opts.delay || 100) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "success",
  "cost_usd": ${opts.cost || 0.001},
  "total_cost_usd": ${opts.cost || 0.001},
  "duration_ms": ${opts.delay || 100},
  "num_turns": 1,
  "result": "${opts.output || "Mock agent completed successfully"}",
  "session_id": "mock-session-${randomUUID()}"
}
MOCK_EOF
`,

  success_slow: (opts) => `#!/bin/bash
# Mock Claude CLI - Slow Success
sleep ${(opts.delay || 5000) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "success",
  "cost_usd": ${opts.cost || 0.01},
  "total_cost_usd": ${opts.cost || 0.01},
  "duration_ms": ${opts.delay || 5000},
  "num_turns": 5,
  "result": "Slow mock agent completed"
}
MOCK_EOF
`,

  error_max_turns: (opts) => `#!/bin/bash
# Mock Claude CLI - Max Turns Error
sleep ${(opts.delay || 100) / 1000}
cat << 'MOCK_EOF'
{
  "type": "result",
  "subtype": "error_max_turns",
  "cost_usd": ${opts.cost || 0.05},
  "total_cost_usd": ${opts.cost || 0.05},
  "duration_ms": ${opts.delay || 100},
  "num_turns": 30,
  "result": null
}
MOCK_EOF
exit 1
`,

  error_rate_limit: (opts) => `#!/bin/bash
# Mock Claude CLI - Rate Limit Error
sleep ${(opts.delay || 50) / 1000}
echo '{"error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}' >&2
exit 1
`,

  timeout: (opts) => `#!/bin/bash
# Mock Claude CLI - Timeout (hangs forever)
sleep 3600
`,

  invalid_json: (opts) => `#!/bin/bash
# Mock Claude CLI - Invalid JSON
sleep ${(opts.delay || 100) / 1000}
echo "This is not valid JSON output {{{{"
`,

  partial_output: (opts) => `#!/bin/bash
# Mock Claude CLI - Partial Output (simulates crash)
sleep ${(opts.delay || 100) / 1000}
echo '{"type": "result", "subtype":'
# Script exits mid-output
`,
};

/**
 * Creates a temporary mock Claude CLI script.
 * Returns the path to the executable script.
 */
export async function createMockClaude(
  options: MockOptions
): Promise<string> {
  const scriptContent = MOCK_SCRIPTS[options.behavior](options);
  const tempDir = os.tmpdir();
  const scriptPath = path.join(
    tempDir,
    `mock-claude-${options.behavior}-${randomUUID()}.sh`
  );

  await fs.writeFile(scriptPath, scriptContent, { mode: 0o755 });

  return scriptPath;
}

/**
 * Cleans up a mock script after use.
 */
export async function cleanupMockClaude(scriptPath: string): Promise<void> {
  try {
    await fs.unlink(scriptPath);
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Creates mock and returns cleanup function.
 * Use with try/finally or vitest afterEach.
 */
export async function withMockClaude(
  options: MockOptions
): Promise<{ path: string; cleanup: () => Promise<void> }> {
  const scriptPath = await createMockClaude(options);
  return {
    path: scriptPath,
    cleanup: () => cleanupMockClaude(scriptPath),
  };
}
```

### Vitest Integration (`packages/tui/tests/mocks/spawnHelper.ts`)

```typescript
import { spawn, ChildProcess } from "child_process";
import { createMockClaude, cleanupMockClaude, MockOptions } from "./mockClaude";

export interface SpawnResult {
  stdout: string;
  stderr: string;
  exitCode: number | null;
  killed: boolean;
  duration: number;
}

/**
 * Spawns the mock CLI and collects output.
 * Use for testing spawn_agent behavior.
 */
export async function spawnMockClaude(
  options: MockOptions,
  stdinContent?: string,
  timeout?: number
): Promise<SpawnResult> {
  const mockPath = await createMockClaude(options);
  const startTime = Date.now();

  return new Promise(async (resolve) => {
    let stdout = "";
    let stderr = "";
    let killed = false;

    const proc = spawn(mockPath, [], {
      stdio: ["pipe", "pipe", "pipe"],
    });

    // Write stdin if provided
    if (stdinContent) {
      proc.stdin.write(stdinContent);
      proc.stdin.end();
    }

    proc.stdout.on("data", (data) => {
      stdout += data.toString();
    });

    proc.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    // Timeout handling
    let timer: NodeJS.Timeout | null = null;
    if (timeout) {
      timer = setTimeout(() => {
        killed = true;
        proc.kill("SIGTERM");
        // Escalate to SIGKILL after 1s
        setTimeout(() => {
          if (!proc.killed) {
            proc.kill("SIGKILL");
          }
        }, 1000);
      }, timeout);
    }

    proc.on("close", async (code) => {
      if (timer) clearTimeout(timer);
      await cleanupMockClaude(mockPath);

      resolve({
        stdout,
        stderr,
        exitCode: code,
        killed,
        duration: Date.now() - startTime,
      });
    });
  });
}
```

### Self-Tests (`packages/tui/tests/mocks/mockClaude.test.ts`)

```typescript
import { describe, it, expect } from "vitest";
import { spawnMockClaude } from "./spawnHelper";

describe("Mock Claude CLI", () => {
  describe("success behavior", () => {
    it("should return valid JSON with success result", async () => {
      const result = await spawnMockClaude({ behavior: "success" });

      expect(result.exitCode).toBe(0);
      expect(result.killed).toBe(false);

      const output = JSON.parse(result.stdout);
      expect(output.type).toBe("result");
      expect(output.subtype).toBe("success");
      expect(output.cost_usd).toBeGreaterThan(0);
    });

    it("should accept custom output", async () => {
      const result = await spawnMockClaude({
        behavior: "success",
        output: "Custom test output",
      });

      const output = JSON.parse(result.stdout);
      expect(output.result).toBe("Custom test output");
    });
  });

  describe("error behaviors", () => {
    it("should return max_turns error with exit code 1", async () => {
      const result = await spawnMockClaude({ behavior: "error_max_turns" });

      expect(result.exitCode).toBe(1);
      const output = JSON.parse(result.stdout);
      expect(output.subtype).toBe("error_max_turns");
    });

    it("should return rate_limit error on stderr", async () => {
      const result = await spawnMockClaude({ behavior: "error_rate_limit" });

      expect(result.exitCode).toBe(1);
      expect(result.stderr).toContain("rate_limit");
    });
  });

  describe("timeout handling", () => {
    it("should kill hanging process after timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        undefined,
        200 // 200ms timeout
      );

      expect(result.killed).toBe(true);
      expect(result.duration).toBeLessThan(1000);
    });
  });

  describe("invalid output handling", () => {
    it("should return invalid JSON for parsing tests", async () => {
      const result = await spawnMockClaude({ behavior: "invalid_json" });

      expect(result.exitCode).toBe(0);
      expect(() => JSON.parse(result.stdout)).toThrow();
    });

    it("should return partial output for crash simulation", async () => {
      const result = await spawnMockClaude({ behavior: "partial_output" });

      expect(result.stdout).toContain('{"type":');
      expect(() => JSON.parse(result.stdout)).toThrow();
    });
  });

  describe("stdin handling", () => {
    it("should accept stdin content", async () => {
      // Success mock ignores stdin but accepts it
      const result = await spawnMockClaude(
        { behavior: "success" },
        "Test prompt content"
      );

      expect(result.exitCode).toBe(0);
    });
  });
});
```

## Acceptance Criteria

- [ ] Mock CLI generator creates executable scripts
- [ ] All 7 behaviors implemented (success, success_slow, error_max_turns, error_rate_limit, timeout, invalid_json, partial_output)
- [ ] Vitest integration helper works with spawn()
- [ ] Self-tests pass: `npm test -- tests/mocks/mockClaude.test.ts`
- [ ] Timeout test completes in <2s (not actually waiting for full timeout)
- [ ] Cleanup removes temporary scripts
- [ ] Code coverage ≥80% on mock infrastructure

## Test Deliverables

- [ ] Test file created: `packages/tui/tests/mocks/mockClaude.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing: `npm test -- tests/mocks/`
- [ ] Coverage ≥80%

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-004
title: Environment Validation Pre-flight Checks
description: Implement pre-flight checks for required dependencies before spawn_agent can be used.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-001]
phase: 1
tags: [infrastructure, validation, phase-1]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-004: Environment Validation Pre-flight Checks

## Description

Implement pre-flight validation that checks all required dependencies before spawn_agent can be used. Fail fast with clear error messages instead of failing later with cryptic errors.

**Source**: Staff-Architect Analysis §4.2.1

## Why This Matters

Without pre-flight checks:
- `spawn()` fails silently if `claude` not in PATH
- Temp file creation fails without clear error
- Users get cryptic errors instead of actionable guidance

## Task

1. Create environment validator module
2. Check for claude CLI in PATH
3. Check /tmp writability
4. Check required env vars
5. Integrate with TUI startup

## Files

- `packages/tui/src/spawn/validation.ts` — Validation functions
- `packages/tui/src/spawn/validation.test.ts` — Tests
- `packages/tui/src/index.tsx` — Integration point

## Implementation

### Validation Module (`packages/tui/src/spawn/validation.ts`)

```typescript
import { execSync } from "child_process";
import * as fs from "fs/promises";
import * as path from "path";
import * as os from "os";

export interface ValidationResult {
  ok: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  code: string;
  message: string;
  resolution: string;
}

export interface ValidationWarning {
  code: string;
  message: string;
  impact: string;
}

/**
 * Validates environment for spawn_agent functionality.
 * Call at TUI startup to fail fast with clear errors.
 */
export async function validateSpawnEnvironment(): Promise<ValidationResult> {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  // Check 1: claude CLI in PATH
  try {
    execSync("which claude", { stdio: "pipe" });
  } catch {
    errors.push({
      code: "E_CLAUDE_NOT_FOUND",
      message: "claude CLI not found in PATH",
      resolution:
        "Install Claude Code CLI: npm install -g @anthropic-ai/claude-code",
    });
  }

  // Check 2: /tmp writable
  const tmpTestFile = path.join(os.tmpdir(), `spawn-test-${Date.now()}`);
  try {
    await fs.writeFile(tmpTestFile, "test", "utf-8");
    await fs.unlink(tmpTestFile);
  } catch (err) {
    errors.push({
      code: "E_TMP_NOT_WRITABLE",
      message: `Cannot write to temp directory: ${os.tmpdir()}`,
      resolution:
        "Ensure temp directory exists and is writable, or set TMPDIR env var",
    });
  }

  // Check 3: GOGENT_MCP_SPAWN_ENABLED not explicitly disabled
  if (process.env.GOGENT_MCP_SPAWN_ENABLED === "false") {
    warnings.push({
      code: "W_SPAWN_DISABLED",
      message: "MCP spawn is disabled via GOGENT_MCP_SPAWN_ENABLED=false",
      impact: "spawn_agent tool will not be available",
    });
  }

  // Check 4: XDG_DATA_HOME for telemetry (warning only)
  if (!process.env.XDG_DATA_HOME) {
    warnings.push({
      code: "W_XDG_DATA_HOME_MISSING",
      message: "XDG_DATA_HOME not set",
      impact: "Telemetry will use fallback ~/.local/share",
    });
  }

  // Check 5: Node.js version
  const nodeVersion = process.versions.node;
  const [major] = nodeVersion.split(".").map(Number);
  if (major < 20) {
    errors.push({
      code: "E_NODE_VERSION",
      message: `Node.js ${nodeVersion} is below minimum required version 20`,
      resolution: "Upgrade Node.js to version 20 or higher",
    });
  }

  return {
    ok: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Formats validation result for console output.
 */
export function formatValidationResult(result: ValidationResult): string {
  const lines: string[] = [];

  if (result.ok) {
    lines.push("✅ Environment validation passed");
  } else {
    lines.push("❌ Environment validation failed");
  }

  if (result.errors.length > 0) {
    lines.push("");
    lines.push("Errors:");
    for (const err of result.errors) {
      lines.push(`  [${err.code}] ${err.message}`);
      lines.push(`    → ${err.resolution}`);
    }
  }

  if (result.warnings.length > 0) {
    lines.push("");
    lines.push("Warnings:");
    for (const warn of result.warnings) {
      lines.push(`  [${warn.code}] ${warn.message}`);
      lines.push(`    Impact: ${warn.impact}`);
    }
  }

  return lines.join("\n");
}

/**
 * Validates and throws if critical errors found.
 * Use at startup to prevent running with invalid environment.
 */
export async function assertValidSpawnEnvironment(): Promise<void> {
  const result = await validateSpawnEnvironment();

  if (!result.ok) {
    const formatted = formatValidationResult(result);
    throw new Error(`Spawn environment validation failed:\n${formatted}`);
  }

  // Log warnings but don't fail
  if (result.warnings.length > 0) {
    console.warn(formatValidationResult(result));
  }
}
```

### Tests (`packages/tui/src/spawn/validation.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as child_process from "child_process";
import * as fs from "fs/promises";
import {
  validateSpawnEnvironment,
  formatValidationResult,
  assertValidSpawnEnvironment,
} from "./validation";

// Mock child_process.execSync
vi.mock("child_process", () => ({
  execSync: vi.fn(),
}));

// Mock fs/promises
vi.mock("fs/promises", () => ({
  writeFile: vi.fn(),
  unlink: vi.fn(),
}));

describe("validateSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Default: all checks pass
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should pass when all checks succeed", async () => {
    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("should fail when claude CLI not found", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_CLAUDE_NOT_FOUND" })
    );
  });

  it("should fail when /tmp not writable", async () => {
    vi.mocked(fs.writeFile).mockRejectedValue(new Error("EACCES"));

    const result = await validateSpawnEnvironment();

    expect(result.ok).toBe(false);
    expect(result.errors).toContainEqual(
      expect.objectContaining({ code: "E_TMP_NOT_WRITABLE" })
    );
  });

  it("should warn when GOGENT_MCP_SPAWN_ENABLED=false", async () => {
    const originalEnv = process.env.GOGENT_MCP_SPAWN_ENABLED;
    process.env.GOGENT_MCP_SPAWN_ENABLED = "false";

    try {
      const result = await validateSpawnEnvironment();

      expect(result.ok).toBe(true); // Warnings don't fail
      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_SPAWN_DISABLED" })
      );
    } finally {
      process.env.GOGENT_MCP_SPAWN_ENABLED = originalEnv;
    }
  });
});

describe("formatValidationResult", () => {
  it("should format success result", () => {
    const result = { ok: true, errors: [], warnings: [] };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("✅ Environment validation passed");
  });

  it("should format errors with resolution", () => {
    const result = {
      ok: false,
      errors: [
        {
          code: "E_TEST",
          message: "Test error",
          resolution: "Fix the test",
        },
      ],
      warnings: [],
    };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("❌ Environment validation failed");
    expect(formatted).toContain("[E_TEST]");
    expect(formatted).toContain("Test error");
    expect(formatted).toContain("Fix the test");
  });
});

describe("assertValidSpawnEnvironment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(child_process.execSync).mockReturnValue(Buffer.from("/usr/bin/claude"));
    vi.mocked(fs.writeFile).mockResolvedValue(undefined);
    vi.mocked(fs.unlink).mockResolvedValue(undefined);
  });

  it("should not throw when validation passes", async () => {
    await expect(assertValidSpawnEnvironment()).resolves.not.toThrow();
  });

  it("should throw when validation fails", async () => {
    vi.mocked(child_process.execSync).mockImplementation(() => {
      throw new Error("not found");
    });

    await expect(assertValidSpawnEnvironment()).rejects.toThrow(
      /Spawn environment validation failed/
    );
  });
});
```

## Acceptance Criteria

- [ ] Validation module created with all 5 checks
- [ ] formatValidationResult produces clear, actionable output
- [ ] assertValidSpawnEnvironment throws on critical errors
- [ ] All tests pass: `npm test -- src/spawn/validation.test.ts`
- [ ] Code coverage ≥80%
- [ ] Integrated with TUI startup (call before MCP server registration)

## Test Deliverables

- [ ] Test file created: `packages/tui/src/spawn/validation.test.ts`
- [ ] Number of test functions: 7
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Mocks properly isolate external dependencies

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-005
title: Process Registry and Cleanup
description: Implement global process registry for tracking spawned CLI processes and cleanup on shutdown.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-004]
phase: 1
tags: [infrastructure, lifecycle, phase-1, critical]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-005: Process Registry and Cleanup

## Description

Implement a global process registry that tracks all spawned CLI processes and ensures cleanup on TUI shutdown. Includes SIGTERM → SIGKILL escalation for stubborn processes.

**Source**: Staff-Architect Analysis §4.3.1, §4.3.2, Einstein Analysis §3.5.1

## Why This Matters

Without process registry:
- Orphan processes accumulate if TUI crashes
- No way to kill all spawned agents on Ctrl+C
- Memory leak from abandoned processes
- System resource exhaustion over time

## Task

1. Create ProcessRegistry class
2. Implement SIGTERM → SIGKILL escalation
3. Integrate with existing shutdown handlers
4. Add signal forwarding (Ctrl+C to children)

## Files

- `packages/tui/src/spawn/processRegistry.ts` — Registry implementation
- `packages/tui/src/spawn/processRegistry.test.ts` — Tests
- `packages/tui/src/lifecycle/shutdown.ts` — Integration (modify existing)

## Implementation

### Process Registry (`packages/tui/src/spawn/processRegistry.ts`)

```typescript
import { ChildProcess } from "child_process";
import { EventEmitter } from "events";

export interface ProcessInfo {
  id: string;
  process: ChildProcess;
  agentType: string;
  startTime: number;
  status: "running" | "terminating" | "terminated";
}

export interface ProcessRegistryEvents {
  registered: (info: ProcessInfo) => void;
  unregistered: (id: string, reason: "completed" | "killed" | "crashed") => void;
  allCleaned: () => void;
}

/**
 * Global registry for tracking spawned CLI processes.
 * Ensures cleanup on shutdown with SIGTERM → SIGKILL escalation.
 */
export class ProcessRegistry extends EventEmitter {
  private processes: Map<string, ProcessInfo> = new Map();
  private cleanupInProgress = false;
  private readonly gracePeriod: number;
  private readonly forceKillDelay: number;

  constructor(options?: { gracePeriod?: number; forceKillDelay?: number }) {
    super();
    this.gracePeriod = options?.gracePeriod ?? 5000; // 5s for graceful shutdown
    this.forceKillDelay = options?.forceKillDelay ?? 1000; // 1s before SIGKILL
  }

  /**
   * Register a spawned process for tracking.
   */
  register(id: string, process: ChildProcess, agentType: string): void {
    if (this.cleanupInProgress) {
      // Don't accept new processes during cleanup
      process.kill("SIGTERM");
      return;
    }

    const info: ProcessInfo = {
      id,
      process,
      agentType,
      startTime: Date.now(),
      status: "running",
    };

    this.processes.set(id, info);
    this.emit("registered", info);

    // Auto-unregister on process exit
    process.on("exit", (code, signal) => {
      const reason = signal ? "killed" : code === 0 ? "completed" : "crashed";
      this.unregister(id, reason);
    });
  }

  /**
   * Unregister a process (called automatically on exit).
   */
  unregister(
    id: string,
    reason: "completed" | "killed" | "crashed" = "completed"
  ): void {
    if (this.processes.has(id)) {
      this.processes.delete(id);
      this.emit("unregistered", id, reason);

      if (this.cleanupInProgress && this.processes.size === 0) {
        this.emit("allCleaned");
      }
    }
  }

  /**
   * Get info about a registered process.
   */
  get(id: string): ProcessInfo | undefined {
    return this.processes.get(id);
  }

  /**
   * Get all registered process IDs.
   */
  getAll(): string[] {
    return Array.from(this.processes.keys());
  }

  /**
   * Get count of active processes.
   */
  get size(): number {
    return this.processes.size;
  }

  /**
   * Kill a specific process by ID.
   */
  async kill(id: string): Promise<boolean> {
    const info = this.processes.get(id);
    if (!info || info.status !== "running") {
      return false;
    }

    return this.terminateProcess(info);
  }

  /**
   * Clean up all processes with graceful shutdown.
   * Returns promise that resolves when all processes are terminated.
   */
  async cleanupAll(): Promise<void> {
    if (this.cleanupInProgress) {
      return; // Already cleaning up
    }

    this.cleanupInProgress = true;

    if (this.processes.size === 0) {
      this.cleanupInProgress = false;
      this.emit("allCleaned");
      return;
    }

    // Send SIGTERM to all
    const terminations = Array.from(this.processes.values()).map((info) =>
      this.terminateProcess(info)
    );

    // Wait for all with timeout
    await Promise.race([
      Promise.all(terminations),
      new Promise<void>((resolve) =>
        setTimeout(() => {
          this.forceKillAll();
          resolve();
        }, this.gracePeriod)
      ),
    ]);

    this.cleanupInProgress = false;
  }

  /**
   * Terminate a single process with SIGTERM → SIGKILL escalation.
   */
  private async terminateProcess(info: ProcessInfo): Promise<boolean> {
    if (info.status !== "running") {
      return false;
    }

    info.status = "terminating";
    info.process.kill("SIGTERM");

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        if (!info.process.killed) {
          info.process.kill("SIGKILL");
        }
        info.status = "terminated";
        resolve(true);
      }, this.forceKillDelay);

      info.process.on("exit", () => {
        clearTimeout(timeout);
        info.status = "terminated";
        resolve(true);
      });
    });
  }

  /**
   * Force kill all remaining processes (SIGKILL).
   */
  private forceKillAll(): void {
    for (const info of this.processes.values()) {
      if (!info.process.killed) {
        info.process.kill("SIGKILL");
        info.status = "terminated";
      }
    }
    this.processes.clear();
  }

  /**
   * Forward a signal to all child processes.
   */
  forwardSignal(signal: NodeJS.Signals): void {
    for (const info of this.processes.values()) {
      if (info.status === "running" && !info.process.killed) {
        info.process.kill(signal);
      }
    }
  }
}

// Singleton instance
let globalRegistry: ProcessRegistry | null = null;

/**
 * Get the global process registry instance.
 */
export function getProcessRegistry(): ProcessRegistry {
  if (!globalRegistry) {
    globalRegistry = new ProcessRegistry();
  }
  return globalRegistry;
}

/**
 * Reset global registry (for testing).
 */
export function resetProcessRegistry(): void {
  if (globalRegistry) {
    globalRegistry.removeAllListeners();
  }
  globalRegistry = null;
}
```

### Shutdown Integration (`packages/tui/src/lifecycle/shutdown.ts` modification)

```typescript
// Add to existing shutdown.ts

import { getProcessRegistry } from "../spawn/processRegistry";

// In setupSignalHandlers():
export function setupSignalHandlers(): void {
  const registry = getProcessRegistry();

  // Forward SIGINT to children
  process.on("SIGINT", async () => {
    registry.forwardSignal("SIGINT");
    await registry.cleanupAll();
    process.exit(0);
  });

  // Forward SIGTERM to children
  process.on("SIGTERM", async () => {
    registry.forwardSignal("SIGTERM");
    await registry.cleanupAll();
    process.exit(0);
  });

  // Clean up on uncaught exception
  process.on("uncaughtException", async (err) => {
    console.error("Uncaught exception:", err);
    await registry.cleanupAll();
    process.exit(1);
  });
}

// Add to onShutdown callbacks:
onShutdown(async () => {
  const registry = getProcessRegistry();
  await registry.cleanupAll();
});
```

### Tests (`packages/tui/src/spawn/processRegistry.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawn, ChildProcess } from "child_process";
import {
  ProcessRegistry,
  getProcessRegistry,
  resetProcessRegistry,
} from "./processRegistry";

describe("ProcessRegistry", () => {
  let registry: ProcessRegistry;

  beforeEach(() => {
    registry = new ProcessRegistry({
      gracePeriod: 100, // Short for tests
      forceKillDelay: 50,
    });
  });

  afterEach(() => {
    resetProcessRegistry();
  });

  describe("register", () => {
    it("should track registered processes", () => {
      const mockProcess = createMockProcess();

      registry.register("test-1", mockProcess, "test-agent");

      expect(registry.size).toBe(1);
      expect(registry.get("test-1")).toBeDefined();
      expect(registry.get("test-1")?.agentType).toBe("test-agent");
    });

    it("should emit registered event", () => {
      const mockProcess = createMockProcess();
      const listener = vi.fn();
      registry.on("registered", listener);

      registry.register("test-1", mockProcess, "test-agent");

      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({ id: "test-1" })
      );
    });
  });

  describe("unregister", () => {
    it("should remove process from registry", () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");

      registry.unregister("test-1", "completed");

      expect(registry.size).toBe(0);
      expect(registry.get("test-1")).toBeUndefined();
    });

    it("should emit unregistered event", () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");
      const listener = vi.fn();
      registry.on("unregistered", listener);

      registry.unregister("test-1", "killed");

      expect(listener).toHaveBeenCalledWith("test-1", "killed");
    });
  });

  describe("kill", () => {
    it("should kill specific process", async () => {
      const mockProcess = createMockProcess();
      registry.register("test-1", mockProcess, "test-agent");

      const result = await registry.kill("test-1");

      expect(result).toBe(true);
      expect(mockProcess.kill).toHaveBeenCalledWith("SIGTERM");
    });

    it("should return false for non-existent process", async () => {
      const result = await registry.kill("non-existent");

      expect(result).toBe(false);
    });
  });

  describe("cleanupAll", () => {
    it("should terminate all processes", async () => {
      const mock1 = createMockProcess();
      const mock2 = createMockProcess();
      registry.register("test-1", mock1, "agent1");
      registry.register("test-2", mock2, "agent2");

      await registry.cleanupAll();

      expect(mock1.kill).toHaveBeenCalled();
      expect(mock2.kill).toHaveBeenCalled();
    });

    it("should emit allCleaned when done", async () => {
      const listener = vi.fn();
      registry.on("allCleaned", listener);

      await registry.cleanupAll();

      expect(listener).toHaveBeenCalled();
    });
  });

  describe("forwardSignal", () => {
    it("should forward signal to all children", () => {
      const mock1 = createMockProcess();
      const mock2 = createMockProcess();
      registry.register("test-1", mock1, "agent1");
      registry.register("test-2", mock2, "agent2");

      registry.forwardSignal("SIGINT");

      expect(mock1.kill).toHaveBeenCalledWith("SIGINT");
      expect(mock2.kill).toHaveBeenCalledWith("SIGINT");
    });
  });
});

// Helper to create mock ChildProcess
function createMockProcess(): ChildProcess {
  const emitter = new (require("events").EventEmitter)();
  return {
    ...emitter,
    pid: Math.floor(Math.random() * 10000),
    killed: false,
    kill: vi.fn((signal) => {
      emitter.emit("exit", 0, signal);
      return true;
    }),
    stdin: { write: vi.fn(), end: vi.fn() },
    stdout: { on: vi.fn() },
    stderr: { on: vi.fn() },
  } as unknown as ChildProcess;
}
```

## Acceptance Criteria

- [ ] ProcessRegistry class implemented with all methods
- [ ] SIGTERM → SIGKILL escalation works (1s delay)
- [ ] Signal forwarding works (SIGINT, SIGTERM)
- [ ] Auto-unregister on process exit
- [ ] Integrated with existing shutdown handlers
- [ ] All tests pass: `npm test -- src/spawn/processRegistry.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/src/spawn/processRegistry.test.ts`
- [ ] Number of test functions: 9
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Integration tested with real processes (manual)

### ---TICKET-END---


### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-006
title: Store Interface Extension (Backward Compatible)
description: Extend the Agent interface with hierarchy fields while maintaining backward compatibility with existing code.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-004]
phase: 1
tags: [store, types, phase-1]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-006: Store Interface Extension (Backward Compatible)

## Description

Extend the existing Agent interface with hierarchy and spawning fields while maintaining backward compatibility. All new fields are optional to avoid breaking existing code.

**Source**: Staff-Architect Analysis §4.1.4, §4.6.3

## Why This Matters

The current Agent interface has 10 fields. The proposed has 24. A breaking change would require updating all components simultaneously. The adapter pattern allows incremental migration.

## Task

1. Extend Agent interface with optional fields
2. Create AgentV1 type alias for legacy compatibility
3. Implement ensureAgentV2 adapter function
4. Update store slice to handle new fields

## Files

- `packages/tui/src/store/types.ts` — Type extensions
- `packages/tui/src/store/adapters.ts` — Adapter functions
- `packages/tui/src/store/adapters.test.ts` — Tests

## Implementation

### Type Extensions (`packages/tui/src/store/types.ts`)

```typescript
/**
 * Legacy Agent interface (V1) - DO NOT MODIFY existing fields.
 * Kept for backward compatibility reference.
 */
export interface AgentV1 {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: {
    input: number;
    output: number;
  };
}

/**
 * Extended Agent interface (V2) - All new fields are OPTIONAL.
 * This maintains backward compatibility with V1.
 */
export interface Agent extends AgentV1 {
  // Hierarchy (optional for V1 compatibility)
  agentType?: string;
  epicId?: string;
  depth?: number;
  childIds?: string[];

  // Spawning metadata (optional)
  spawnMethod?: "task" | "mcp-cli";
  spawnedBy?: string;
  prompt?: string;

  // Process info (for MCP-CLI spawns)
  pid?: number;
  queuedAt?: number;

  // Extended status (compatible with V1 status)
  // V1 status values still valid, these are additions
  // "queued" | "streaming" | "timeout" are new options

  // Output (optional)
  output?: string;
  streamBuffer?: string;
  error?: string;

  // Extended metrics (optional)
  cost?: number;
  turns?: number;
  toolCalls?: number;
}

/**
 * Status values - union of V1 and V2
 */
export type AgentStatus =
  | "queued"      // New: waiting to spawn
  | "spawning"   // V1: CLI starting
  | "running"    // V1: executing
  | "streaming"  // New: producing output
  | "complete"   // V1: finished successfully
  | "error"      // V1: failed
  | "timeout";   // New: exceeded time limit

/**
 * Spawn method discriminator
 */
export type SpawnMethod = "task" | "mcp-cli";

/**
 * Input for creating a new agent
 */
export interface CreateAgentInput {
  // Required
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  description: string;

  // Optional hierarchy
  parentId?: string | null;
  agentType?: string;
  epicId?: string;
  spawnMethod?: SpawnMethod;
  prompt?: string;
}
```

### Adapter Functions (`packages/tui/src/store/adapters.ts`)

```typescript
import { Agent, AgentV1, CreateAgentInput } from "./types";
import { randomUUID } from "crypto";

/**
 * Ensures an agent has all V2 fields with sensible defaults.
 * Use when you need to work with extended fields on potentially V1 data.
 */
export function ensureAgentV2(agent: AgentV1 | Agent): Agent {
  // If already has V2 fields, return as-is
  if ("spawnMethod" in agent && agent.spawnMethod !== undefined) {
    return agent as Agent;
  }

  // Upgrade V1 to V2 with defaults
  return {
    ...agent,
    agentType: agent.description || "unknown",
    epicId: "legacy",
    depth: 1,
    childIds: [],
    spawnMethod: "task",
    spawnedBy: "router",
    queuedAt: agent.startTime,
  };
}

/**
 * Creates a new agent with all V2 fields populated.
 */
export function createAgent(input: CreateAgentInput): Agent {
  const now = Date.now();
  const id = randomUUID();

  return {
    // V1 fields
    id,
    parentId: input.parentId ?? null,
    model: input.model,
    tier: input.tier,
    status: "queued",
    description: input.description,
    startTime: now,

    // V2 fields
    agentType: input.agentType || input.description,
    epicId: input.epicId || "default",
    depth: input.parentId ? 2 : 1, // Will be calculated properly by caller
    childIds: [],
    spawnMethod: input.spawnMethod || "task",
    spawnedBy: input.parentId || "router",
    prompt: input.prompt,
    queuedAt: now,
  };
}

/**
 * Check if agent is V2 (has extended fields)
 */
export function isAgentV2(agent: AgentV1 | Agent): agent is Agent {
  return "spawnMethod" in agent && agent.spawnMethod !== undefined;
}

/**
 * Safely get depth, defaulting to 1 for V1 agents
 */
export function getAgentDepth(agent: AgentV1 | Agent): number {
  if ("depth" in agent && typeof agent.depth === "number") {
    return agent.depth;
  }
  return agent.parentId ? 2 : 1;
}

/**
 * Safely get childIds, defaulting to empty array for V1 agents
 */
export function getAgentChildIds(agent: AgentV1 | Agent): string[] {
  if ("childIds" in agent && Array.isArray(agent.childIds)) {
    return agent.childIds;
  }
  return [];
}
```

### Tests (`packages/tui/src/store/adapters.test.ts`)

```typescript
import { describe, it, expect } from "vitest";
import {
  ensureAgentV2,
  createAgent,
  isAgentV2,
  getAgentDepth,
  getAgentChildIds,
} from "./adapters";
import { AgentV1, Agent } from "./types";

describe("ensureAgentV2", () => {
  it("should upgrade V1 agent with defaults", () => {
    const v1Agent: AgentV1 = {
      id: "test-1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      description: "Test agent",
      startTime: 1000,
    };

    const v2Agent = ensureAgentV2(v1Agent);

    expect(v2Agent.agentType).toBe("Test agent");
    expect(v2Agent.epicId).toBe("legacy");
    expect(v2Agent.depth).toBe(1);
    expect(v2Agent.childIds).toEqual([]);
    expect(v2Agent.spawnMethod).toBe("task");
    expect(v2Agent.spawnedBy).toBe("router");
  });

  it("should return V2 agent unchanged", () => {
    const v2Agent: Agent = {
      id: "test-1",
      parentId: "parent-1",
      model: "opus",
      tier: "opus",
      status: "complete",
      startTime: 1000,
      agentType: "einstein",
      epicId: "braintrust-123",
      depth: 2,
      childIds: [],
      spawnMethod: "mcp-cli",
      spawnedBy: "mozart",
    };

    const result = ensureAgentV2(v2Agent);

    expect(result).toEqual(v2Agent);
    expect(result.spawnMethod).toBe("mcp-cli");
  });
});

describe("createAgent", () => {
  it("should create agent with all V2 fields", () => {
    const agent = createAgent({
      model: "haiku",
      tier: "haiku",
      description: "Test scout",
      agentType: "codebase-search",
      epicId: "explore-123",
      spawnMethod: "task",
    });

    expect(agent.id).toBeDefined();
    expect(agent.model).toBe("haiku");
    expect(agent.agentType).toBe("codebase-search");
    expect(agent.status).toBe("queued");
    expect(agent.childIds).toEqual([]);
    expect(agent.queuedAt).toBeDefined();
  });

  it("should use description as agentType fallback", () => {
    const agent = createAgent({
      model: "sonnet",
      tier: "sonnet",
      description: "Code implementation",
    });

    expect(agent.agentType).toBe("Code implementation");
  });
});

describe("isAgentV2", () => {
  it("should return true for V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      spawnMethod: "task",
    };

    expect(isAgentV2(agent)).toBe(true);
  });

  it("should return false for V1 agent", () => {
    const agent: AgentV1 = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(isAgentV2(agent)).toBe(false);
  });
});

describe("getAgentDepth", () => {
  it("should return depth from V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: "parent",
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      depth: 3,
      spawnMethod: "mcp-cli",
    };

    expect(getAgentDepth(agent)).toBe(3);
  });

  it("should return default depth for V1 agent", () => {
    const v1WithParent: AgentV1 = {
      id: "1",
      parentId: "parent",
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    const v1WithoutParent: AgentV1 = {
      id: "2",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(getAgentDepth(v1WithParent)).toBe(2);
    expect(getAgentDepth(v1WithoutParent)).toBe(1);
  });
});

describe("getAgentChildIds", () => {
  it("should return childIds from V2 agent", () => {
    const agent: Agent = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
      childIds: ["child-1", "child-2"],
      spawnMethod: "task",
    };

    expect(getAgentChildIds(agent)).toEqual(["child-1", "child-2"]);
  });

  it("should return empty array for V1 agent", () => {
    const agent: AgentV1 = {
      id: "1",
      parentId: null,
      model: "sonnet",
      tier: "sonnet",
      status: "running",
      startTime: 1000,
    };

    expect(getAgentChildIds(agent)).toEqual([]);
  });
});
```

## Acceptance Criteria

- [ ] Agent interface extended with all V2 fields (all optional)
- [ ] AgentV1 type alias preserved for documentation
- [ ] ensureAgentV2 adapter correctly upgrades V1 agents
- [ ] createAgent produces valid V2 agents
- [ ] Helper functions (isAgentV2, getAgentDepth, getAgentChildIds) work correctly
- [ ] All tests pass: `npm test -- src/store/adapters.test.ts`
- [ ] Code coverage ≥80%
- [ ] Existing components still compile without changes

## Test Deliverables

- [ ] Test file created: `packages/tui/src/store/adapters.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing
- [ ] Coverage ≥80%

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-007
title: gogent-validate Nesting Level Check
description: Add nesting level detection to gogent-validate hook to block Task() at Level 1+ with guidance.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-001]
phase: 1
tags: [hooks, go, validation, phase-1]
needs_planning: false
agent: go-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-007: gogent-validate Nesting Level Check

## Description

Add nesting level detection to the gogent-validate hook. Block Task() invocations at Level 1+ with clear guidance to use MCP spawn_agent instead. Use fail-closed default (missing/invalid level = assume nested).

**Source**: Einstein Analysis §3.2, Staff-Architect Analysis §4.7.3

## Why This Matters

This is the enforcement mechanism for the hybrid approach. Without it, subagents could attempt Task() and fail with cryptic errors instead of being redirected to MCP spawning.

## Task

1. Add getNestingLevel() function with fail-closed default
2. Add nesting level check to validation logic
3. Return clear block message with guidance
4. Add telemetry for blocked Task() calls

## Files

- `cmd/gogent-validate/main.go` — Add nesting check
- `pkg/routing/task_validation.go` — Core logic
- `pkg/routing/task_validation_test.go` — Tests

## Implementation

### Core Logic (`pkg/routing/task_validation.go`)

```go
package routing

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// MaxNestingDepth prevents runaway nesting
	MaxNestingDepth = 10
	
	// DefaultNestingLevel for fail-closed behavior
	DefaultNestingLevel = 1
)

// GetNestingLevel returns the current nesting level from environment.
// Fail-closed: returns 1 (blocked) if missing or invalid.
func GetNestingLevel() int {
	levelStr := os.Getenv("GOGENT_NESTING_LEVEL")
	
	// Missing = fail-closed (assume nested)
	if levelStr == "" {
		return DefaultNestingLevel
	}
	
	level, err := strconv.Atoi(levelStr)
	
	// Invalid = fail-closed
	if err != nil {
		return DefaultNestingLevel
	}
	
	// Out of range = fail-closed
	if level < 0 || level > MaxNestingDepth {
		return DefaultNestingLevel
	}
	
	return level
}

// IsNestingLevelExplicit returns true if GOGENT_NESTING_LEVEL was set explicitly.
// Used for telemetry to distinguish real Level 0 from assumed nesting.
func IsNestingLevelExplicit() bool {
	return os.Getenv("GOGENT_NESTING_LEVEL") != ""
}

// ValidateTaskNestingLevel checks if Task() is allowed at current nesting level.
// Returns nil if allowed, error with guidance if blocked.
func ValidateTaskNestingLevel() error {
	level := GetNestingLevel()
	
	if level > 0 {
		return &NestingLevelError{
			Level:   level,
			Message: fmt.Sprintf(
				"Task() blocked at nesting level %d. "+
					"Subagents cannot spawn sub-subagents via Task(). "+
					"Use MCP spawn_agent tool instead: "+
					"mcp__gofortress__spawn_agent({agent: '...', prompt: '...'})",
				level,
			),
		}
	}
	
	return nil
}

// NestingLevelError represents a Task() blocked due to nesting level.
type NestingLevelError struct {
	Level   int
	Message string
}

func (e *NestingLevelError) Error() string {
	return e.Message
}

// BlockResponseForNesting creates the standard block response for nesting violations.
func BlockResponseForNesting(level int) map[string]interface{} {
	return map[string]interface{}{
		"decision": "block",
		"reason": fmt.Sprintf(
			"Task() blocked at nesting level %d. Use MCP spawn_agent instead.",
			level,
		),
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "nesting_level_exceeded",
			"nestingLevel":             level,
			"suggestion":               "mcp__gofortress__spawn_agent({agent: '...', prompt: '...'})",
		},
	}
}
```

### Main Hook Update (`cmd/gogent-validate/main.go`)

```go
// Add to main() after parsing input, before existing Task validation

// Check nesting level for Task tool
if event.ToolName == "Task" {
    nestingLevel := routing.GetNestingLevel()
    isExplicit := routing.IsNestingLevelExplicit()
    
    if nestingLevel > 0 {
        // Log the block for telemetry
        logNestingBlock(event, nestingLevel, isExplicit)
        
        // Return block response
        response := routing.BlockResponseForNesting(nestingLevel)
        outputJSON(response)
        return
    }
}

// Helper function for telemetry
func logNestingBlock(event *Event, level int, explicit bool) {
    telemetry := map[string]interface{}{
        "timestamp":     time.Now().UTC().Format(time.RFC3339),
        "event":         "task_blocked_nesting",
        "session_id":    event.SessionID,
        "nesting_level": level,
        "level_explicit": explicit,
        "tool_name":     event.ToolName,
    }
    
    // Append to telemetry file
    telemetryPath := filepath.Join(
        os.Getenv("XDG_DATA_HOME"),
        "gogent",
        "nesting-blocks.jsonl",
    )
    
    appendJSONL(telemetryPath, telemetry)
}
```

### Tests (`pkg/routing/task_validation_test.go`)

```go
package routing

import (
	"os"
	"testing"
)

func TestGetNestingLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "missing env var returns default (fail-closed)",
			envValue: "",
			expected: DefaultNestingLevel,
		},
		{
			name:     "level 0 returns 0",
			envValue: "0",
			expected: 0,
		},
		{
			name:     "level 1 returns 1",
			envValue: "1",
			expected: 1,
		},
		{
			name:     "level 5 returns 5",
			envValue: "5",
			expected: 5,
		},
		{
			name:     "invalid string returns default (fail-closed)",
			envValue: "abc",
			expected: DefaultNestingLevel,
		},
		{
			name:     "negative returns default (fail-closed)",
			envValue: "-1",
			expected: DefaultNestingLevel,
		},
		{
			name:     "exceeds max returns default (fail-closed)",
			envValue: "100",
			expected: DefaultNestingLevel,
		},
		{
			name:     "max valid level returns correctly",
			envValue: "10",
			expected: 10,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or clear env var
			if tt.envValue == "" {
				os.Unsetenv("GOGENT_NESTING_LEVEL")
			} else {
				os.Setenv("GOGENT_NESTING_LEVEL", tt.envValue)
			}
			defer os.Unsetenv("GOGENT_NESTING_LEVEL")
			
			result := GetNestingLevel()
			
			if result != tt.expected {
				t.Errorf("GetNestingLevel() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestIsNestingLevelExplicit(t *testing.T) {
	// Test when not set
	os.Unsetenv("GOGENT_NESTING_LEVEL")
	if IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = true when env not set")
	}
	
	// Test when set (even to empty)
	os.Setenv("GOGENT_NESTING_LEVEL", "")
	// Note: os.Getenv returns "" for both unset and empty, so this tests implementation
	
	os.Setenv("GOGENT_NESTING_LEVEL", "0")
	if !IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = false when env is set")
	}
	
	os.Unsetenv("GOGENT_NESTING_LEVEL")
}

func TestValidateTaskNestingLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantError bool
	}{
		{
			name:      "level 0 allows Task",
			level:     "0",
			wantError: false,
		},
		{
			name:      "level 1 blocks Task",
			level:     "1",
			wantError: true,
		},
		{
			name:      "level 2 blocks Task",
			level:     "2",
			wantError: true,
		},
		{
			name:      "missing level blocks Task (fail-closed)",
			level:     "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.level == "" {
				os.Unsetenv("GOGENT_NESTING_LEVEL")
			} else {
				os.Setenv("GOGENT_NESTING_LEVEL", tt.level)
			}
			defer os.Unsetenv("GOGENT_NESTING_LEVEL")
			
			err := ValidateTaskNestingLevel()
			
			if tt.wantError && err == nil {
				t.Error("ValidateTaskNestingLevel() = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateTaskNestingLevel() = %v, want nil", err)
			}
		})
	}
}

func TestBlockResponseForNesting(t *testing.T) {
	response := BlockResponseForNesting(2)
	
	if response["decision"] != "block" {
		t.Errorf("decision = %v, want 'block'", response["decision"])
	}
	
	hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("hookSpecificOutput not a map")
	}
	
	if hookOutput["nestingLevel"] != 2 {
		t.Errorf("nestingLevel = %v, want 2", hookOutput["nestingLevel"])
	}
	
	if hookOutput["permissionDecision"] != "deny" {
		t.Errorf("permissionDecision = %v, want 'deny'", hookOutput["permissionDecision"])
	}
}
```

## Acceptance Criteria

- [ ] GetNestingLevel() returns correct values for all cases
- [ ] Fail-closed behavior: missing/invalid = Level 1 (blocked)
- [ ] ValidateTaskNestingLevel() blocks at Level 1+
- [ ] Block response includes clear guidance for MCP spawn_agent
- [ ] Telemetry logged for blocked Task() calls
- [ ] All tests pass: `go test ./pkg/routing/...`
- [ ] Code coverage ≥80%
- [ ] Hook compiles and runs correctly

## Test Deliverables

- [ ] Test file updated: `pkg/routing/task_validation_test.go`
- [ ] Number of test functions: 4
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: spawn subagent, attempt Task(), verify block

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-008
title: spawn_agent MCP Tool Implementation
description: Implement the core spawn_agent MCP tool that spawns Claude CLI processes with proper process management.
status: pending
time_estimate: 4h
dependencies: [MCP-SPAWN-003, MCP-SPAWN-004, MCP-SPAWN-005, MCP-SPAWN-006]
phase: 1
tags: [mcp, spawn, core, phase-1]
needs_planning: false
agent: typescript-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-008: spawn_agent MCP Tool Implementation

## Description

Implement the core `spawn_agent` MCP tool that spawns Claude CLI processes. Uses stdin piping (not shell: true), integrates with process registry, respects buffer limits, and handles timeouts.

**Source**: Staff-Architect Analysis §4.1.2, §4.3.3, §4.6.1, Einstein Analysis §3.5

## Why This Matters

This is the core mechanism for Level 1+ agent spawning. All orchestrators will use this tool to spawn specialist agents.

## Task

1. Implement spawn_agent tool with correct CLI invocation
2. Integrate with process registry
3. Add buffer limits for output
4. Handle timeout with SIGTERM → SIGKILL
5. Parse JSON output correctly

## Files

- `packages/tui/src/mcp/tools/spawnAgent.ts` — Main implementation
- `packages/tui/src/mcp/tools/spawnAgent.test.ts` — Tests
- `packages/tui/src/mcp/server.ts` — Registration

## Implementation

### spawn_agent Tool (`packages/tui/src/mcp/tools/spawnAgent.ts`)

```typescript
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn } from "child_process";
import { getProcessRegistry } from "../spawn/processRegistry";
import { randomUUID } from "crypto";

// Constants
const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB
const DEFAULT_TIMEOUT = 300000; // 5 minutes

/**
 * Result from a spawn_agent invocation
 */
export interface SpawnResult {
  agentId: string;
  agent: string;
  success: boolean;
  output?: string;
  error?: string;
  cost?: number;
  turns?: number;
  duration?: number;
  truncated?: boolean;
}

/**
 * spawn_agent MCP tool - spawns Claude CLI processes for Level 1+ agent spawning.
 */
export const spawnAgent = tool(
  "spawn_agent",
  `Spawn a Claude Code subagent with full tool access via CLI.
  
Use this tool when you need to spawn a sub-subagent (Level 2+).
The spawned agent runs as an independent CLI process with full tool access.

Example:
  spawn_agent({
    agent: "einstein",
    description: "Theoretical analysis",
    prompt: "AGENT: einstein\\n\\nAnalyze the problem...",
    model: "opus"
  })`,
  {
    agent: z.string().describe("Agent type from agents-index.json (e.g., 'einstein', 'backend-reviewer')"),
    description: z.string().describe("Brief description for logging"),
    prompt: z.string().describe("Full prompt to send to the agent"),
    model: z.enum(["haiku", "sonnet", "opus"]).optional().describe("Model to use (default: from agent config)"),
    timeout: z.number().optional().describe("Timeout in ms (default: 300000)"),
    allowedTools: z.array(z.string()).optional().describe("Restrict available tools"),
    maxBudget: z.number().optional().describe("Max budget in USD"),
  },
  async (args): Promise<{ content: Array<{ type: "text"; text: string }> }> => {
    const agentId = randomUUID();
    const registry = getProcessRegistry();
    const timeout = args.timeout ?? DEFAULT_TIMEOUT;
    const startTime = Date.now();

    // Build CLI arguments
    const cliArgs = buildCliArgs(args);

    return new Promise((resolve) => {
      // Spawn CLI process (NO shell: true)
      const proc = spawn("claude", cliArgs, {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GOGENT_NESTING_LEVEL: String(getCurrentNestingLevel() + 1),
          GOGENT_PARENT_AGENT: agentId,
          GOGENT_SPAWN_METHOD: "mcp-cli",
        },
      });

      // Register with process registry
      registry.register(agentId, proc, args.agent);

      // Output collection with buffer limit
      let stdout = "";
      let stderr = "";
      let truncated = false;

      proc.stdout.on("data", (chunk: Buffer) => {
        if (!truncated && stdout.length < MAX_BUFFER_SIZE) {
          stdout += chunk.toString();
          if (stdout.length >= MAX_BUFFER_SIZE) {
            truncated = true;
            stdout += "\n[OUTPUT TRUNCATED - exceeded 10MB limit]";
          }
        }
      });

      proc.stderr.on("data", (chunk: Buffer) => {
        // Stderr is typically small, but limit anyway
        if (stderr.length < 1024 * 1024) {
          stderr += chunk.toString();
        }
      });

      // Send prompt via stdin
      proc.stdin.write(args.prompt);
      proc.stdin.end();

      // Timeout handling
      const timer = setTimeout(() => {
        // SIGTERM first
        proc.kill("SIGTERM");

        // SIGKILL after 5s if still running
        setTimeout(() => {
          if (!proc.killed) {
            proc.kill("SIGKILL");
          }
        }, 5000);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: false,
          error: `Agent timed out after ${timeout}ms`,
          duration: Date.now() - startTime,
          truncated,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      }, timeout);

      // Process completion
      proc.on("close", (code, signal) => {
        clearTimeout(timer);

        const duration = Date.now() - startTime;
        const parsed = parseCliOutput(stdout);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: code === 0 && !signal,
          output: parsed.result || stdout,
          error: code !== 0 ? stderr || `Exit code ${code}` : undefined,
          cost: parsed.cost,
          turns: parsed.turns,
          duration,
          truncated,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      });

      proc.on("error", (err) => {
        clearTimeout(timer);

        const result: SpawnResult = {
          agentId,
          agent: args.agent,
          success: false,
          error: `Spawn error: ${err.message}`,
          duration: Date.now() - startTime,
        };

        resolve({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      });
    });
  }
);

/**
 * Build CLI arguments for claude command.
 */
function buildCliArgs(args: {
  model?: string;
  allowedTools?: string[];
  maxBudget?: number;
}): string[] {
  const cliArgs = ["-p", "--output-format", "json"];

  if (args.model) {
    cliArgs.push("--model", args.model);
  }

  // Use delegate mode instead of dangerously-skip-permissions
  cliArgs.push("--permission-mode", "delegate");

  if (args.allowedTools && args.allowedTools.length > 0) {
    cliArgs.push("--allowedTools", args.allowedTools.join(","));
  }

  if (args.maxBudget) {
    cliArgs.push("--max-budget-usd", String(args.maxBudget));
  }

  return cliArgs;
}

/**
 * Parse JSON output from claude CLI.
 */
function parseCliOutput(stdout: string): {
  result?: string;
  cost?: number;
  turns?: number;
} {
  try {
    const json = JSON.parse(stdout.trim());
    return {
      result: json.result || json.output,
      cost: json.cost_usd || json.total_cost_usd,
      turns: json.num_turns,
    };
  } catch {
    // Not valid JSON, return raw output
    return { result: stdout };
  }
}

/**
 * Get current nesting level from environment.
 */
function getCurrentNestingLevel(): number {
  const level = process.env.GOGENT_NESTING_LEVEL;
  if (!level) return 0;
  const parsed = parseInt(level, 10);
  return isNaN(parsed) ? 0 : parsed;
}
```

### Tests (`packages/tui/src/mcp/tools/spawnAgent.test.ts`)

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawnMockClaude } from "../../tests/mocks/spawnHelper";
import { resetProcessRegistry, getProcessRegistry } from "../spawn/processRegistry";

// Note: Full tests require mock CLI infrastructure from MCP-SPAWN-003

describe("spawn_agent tool", () => {
  beforeEach(() => {
    resetProcessRegistry();
  });

  afterEach(() => {
    resetProcessRegistry();
  });

  describe("buildCliArgs", () => {
    it("should include -p and --output-format json", () => {
      // Import the function for testing
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({});
      
      expect(args).toContain("-p");
      expect(args).toContain("--output-format");
      expect(args).toContain("json");
    });

    it("should include model when specified", () => {
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({ model: "opus" });
      
      expect(args).toContain("--model");
      expect(args).toContain("opus");
    });

    it("should include allowedTools when specified", () => {
      const { buildCliArgs } = require("./spawnAgent");
      
      const args = buildCliArgs({ allowedTools: ["Read", "Glob", "Grep"] });
      
      expect(args).toContain("--allowedTools");
      expect(args).toContain("Read,Glob,Grep");
    });
  });

  describe("parseCliOutput", () => {
    it("should parse valid JSON output", () => {
      const { parseCliOutput } = require("./spawnAgent");
      
      const output = JSON.stringify({
        result: "Analysis complete",
        cost_usd: 0.05,
        num_turns: 3,
      });
      
      const parsed = parseCliOutput(output);
      
      expect(parsed.result).toBe("Analysis complete");
      expect(parsed.cost).toBe(0.05);
      expect(parsed.turns).toBe(3);
    });

    it("should return raw output for invalid JSON", () => {
      const { parseCliOutput } = require("./spawnAgent");
      
      const output = "This is not JSON";
      const parsed = parseCliOutput(output);
      
      expect(parsed.result).toBe("This is not JSON");
    });
  });

  describe("getCurrentNestingLevel", () => {
    it("should return 0 when not set", () => {
      const originalEnv = process.env.GOGENT_NESTING_LEVEL;
      delete process.env.GOGENT_NESTING_LEVEL;
      
      const { getCurrentNestingLevel } = require("./spawnAgent");
      expect(getCurrentNestingLevel()).toBe(0);
      
      process.env.GOGENT_NESTING_LEVEL = originalEnv;
    });

    it("should return parsed level when set", () => {
      const originalEnv = process.env.GOGENT_NESTING_LEVEL;
      process.env.GOGENT_NESTING_LEVEL = "2";
      
      // Need to re-import to pick up new env
      vi.resetModules();
      const { getCurrentNestingLevel } = require("./spawnAgent");
      expect(getCurrentNestingLevel()).toBe(2);
      
      process.env.GOGENT_NESTING_LEVEL = originalEnv;
    });
  });

  // Integration tests with mock CLI
  describe("integration with mock CLI", () => {
    it("should handle successful spawn", async () => {
      const result = await spawnMockClaude(
        { behavior: "success", output: "Test output" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toContain("success");
    });

    it("should handle timeout", async () => {
      const result = await spawnMockClaude(
        { behavior: "timeout" },
        "Test prompt",
        100 // 100ms timeout
      );

      expect(result.killed).toBe(true);
    });

    it("should handle error response", async () => {
      const result = await spawnMockClaude(
        { behavior: "error_max_turns" },
        "Test prompt"
      );

      expect(result.exitCode).toBe(1);
    });
  });
});
```

## Acceptance Criteria

- [ ] spawn_agent tool compiles and type-checks
- [ ] Uses stdin piping (NOT shell: true)
- [ ] Integrates with process registry
- [ ] Buffer limited to 10MB with truncation indicator
- [ ] Timeout handled with SIGTERM → SIGKILL escalation
- [ ] JSON output parsed correctly
- [ ] Returns structured SpawnResult
- [ ] Nesting level incremented for child processes
- [ ] All tests pass with mock CLI
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file created: `packages/tui/src/mcp/tools/spawnAgent.test.ts`
- [ ] Number of test functions: 8
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: invoke from subagent, verify CLI spawns

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-009
title: MCP Server Registration and Integration
description: Register spawn_agent tool with MCP server and integrate with TUI startup.
status: pending
time_estimate: 1h
dependencies: [MCP-SPAWN-008]
phase: 2
tags: [mcp, integration, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-009: MCP Server Registration and Integration

## Description

Register the spawn_agent tool with the TUI's MCP server and integrate with startup validation.

**Source**: Staff-Architect Analysis §4.2.2

## Task

1. Add spawn_agent to MCP server tools
2. Add feature flag check
3. Integrate environment validation
4. Test tool availability

## Files

- `packages/tui/src/mcp/server.ts` — Add tool registration
- `packages/tui/src/index.tsx` — Add startup validation

## Implementation

### MCP Server Update (`packages/tui/src/mcp/server.ts`)

```typescript
import { createSdkMcpServer } from "@anthropic-ai/claude-agent-sdk";
import { spawnAgent } from "./tools/spawnAgent";
import { testMcpPing } from "./tools/testMcpPing";
// ... existing tool imports

/**
 * Check if MCP spawning is enabled via feature flag.
 */
function isSpawnEnabled(): boolean {
  return process.env.GOGENT_MCP_SPAWN_ENABLED !== "false";
}

/**
 * Create MCP server with all tools.
 */
export function createMcpServer() {
  const tools = [
    // Existing tools
    askUser,
    confirmAction,
    requestInput,
    selectOption,
    // Test tool (always available for verification)
    testMcpPing,
  ];

  // Conditionally add spawn tools
  if (isSpawnEnabled()) {
    tools.push(spawnAgent);
  }

  return createSdkMcpServer({
    name: "gofortress",
    version: "1.0.0",
    tools,
  });
}
```

### Startup Integration (`packages/tui/src/index.tsx`)

```typescript
import { assertValidSpawnEnvironment } from "./spawn/validation";

async function main() {
  // Validate spawn environment before starting
  try {
    await assertValidSpawnEnvironment();
  } catch (err) {
    console.error(err.message);
    // Continue with warnings but allow startup
  }

  // ... rest of startup
}
```

## Acceptance Criteria

- [ ] spawn_agent registered in MCP server
- [ ] Feature flag respected (GOGENT_MCP_SPAWN_ENABLED=false disables)
- [ ] Environment validation runs at startup
- [ ] Tool available to subagents (verified with testMcpPing)

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-010
title: Mozart Orchestrator Update
description: Update Mozart to use spawn_agent for spawning Einstein and Staff-Architect.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009]
phase: 2
tags: [orchestrator, braintrust, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-010: Mozart Orchestrator Update

## Description

Update the Mozart orchestrator (Braintrust skill) to use MCP spawn_agent for spawning Einstein and Staff-Architect instead of attempting Task().

**Source**: Einstein Analysis §3.6.1

## Task

1. Update Mozart prompt to use spawn_agent
2. Add parallel spawning for Einstein + Staff-Architect
3. Update error handling for spawn failures
4. Test full Braintrust workflow

## Files

- `~/.claude/skills/braintrust/SKILL.md` — Update Mozart instructions

## Implementation

### Updated Mozart Instructions

Mozart should be instructed to use spawn_agent like this:

```
When spawning Einstein and Staff-Architect, use the MCP spawn_agent tool:

// Spawn Einstein
mcp__gofortress__spawn_agent({
  agent: "einstein",
  description: "Theoretical analysis for Braintrust",
  prompt: `AGENT: einstein

BRAINTRUST WORKFLOW - THEORETICAL ANALYSIS

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000  // 10 minutes for complex analysis
})

// Spawn Staff-Architect (can be parallel with Einstein)
mcp__gofortress__spawn_agent({
  agent: "staff-architect-critical-review",
  description: "Practical review for Braintrust",
  prompt: `AGENT: staff-architect-critical-review

BRAINTRUST WORKFLOW - PRACTICAL REVIEW

[Problem Brief here]

[Task instructions here]`,
  model: "opus",
  timeout: 600000
})
```

## Acceptance Criteria

- [ ] Mozart uses spawn_agent instead of Task() for Level 2 spawning
- [ ] Einstein spawns successfully via MCP
- [ ] Staff-Architect spawns successfully via MCP
- [ ] Both outputs collected correctly
- [ ] Beethoven synthesis works with collected outputs
- [ ] Full Braintrust workflow completes end-to-end

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-011
title: Review-Orchestrator Update
description: Update review-orchestrator to use spawn_agent for parallel reviewer spawning.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-009]
phase: 2
tags: [orchestrator, review, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-011: Review-Orchestrator Update

## Description

Update the review-orchestrator to use MCP spawn_agent for spawning parallel reviewers (backend, frontend, standards, architect).

**Source**: Staff-Architect Analysis v2 §Part 6

## Task

1. Update review-orchestrator prompt to use spawn_agent
2. Spawn reviewers in parallel
3. Collect all results (handle partial failures)
4. Test full review workflow

## Acceptance Criteria

- [ ] review-orchestrator uses spawn_agent for reviewers
- [ ] Parallel spawning works (all reviewers start simultaneously)
- [ ] Partial failures handled (continue if 1 reviewer fails)
- [ ] All findings collected and synthesized
- [ ] Full /review workflow completes

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-012
title: Integration Testing and Documentation
description: End-to-end testing of MCP spawning and documentation updates.
status: pending
time_estimate: 4h
dependencies: [MCP-SPAWN-010, MCP-SPAWN-011]
phase: 3
tags: [testing, documentation, phase-3]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-012: Integration Testing and Documentation

## Description

End-to-end testing of the complete MCP spawning system and documentation updates for CLAUDE.md and troubleshooting guide.

**Source**: Staff-Architect Analysis §4.5.2, §4.7.2

## Task

1. Create E2E test for Braintrust workflow
2. Create E2E test for /review workflow
3. Test timeout and error scenarios
4. Update CLAUDE.md with spawning documentation
5. Create troubleshooting guide

## Files

- `packages/tui/tests/e2e/braintrust.test.ts` — E2E test
- `packages/tui/tests/e2e/review.test.ts` — E2E test
- `~/.claude/CLAUDE.md` — Documentation update
- `~/.claude/docs/mcp-spawning-troubleshooting.md` — New guide

## Acceptance Criteria

- [ ] E2E Braintrust test passes (may use mock CLI for CI)
- [ ] E2E Review test passes
- [ ] Timeout scenarios tested
- [ ] Error propagation tested
- [ ] CLAUDE.md updated with spawn_agent documentation
- [ ] Troubleshooting guide created
- [ ] 3 successful real Braintrust runs without intervention

### ---TICKET-END---

---

## 8. Bash Slicing Script

Use this script to extract individual tickets from this document.

```bash
#!/bin/bash
# slice-tickets.sh
# Extracts individual tickets from mcp-spawning-v3.md
#
# Usage: ./slice-tickets.sh [output_directory]
# Default output: ./tickets/mcp-spawn/

set -e

INPUT_FILE="${1:-$(dirname "$0")/mcp-spawning-v3.md}"
OUTPUT_DIR="${2:-$(dirname "$0")/mcp-spawn}"

# Verify input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file not found: $INPUT_FILE"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "Slicing tickets from: $INPUT_FILE"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Extract tickets using awk
awk '
BEGIN { 
    ticket_num = 0
    in_ticket = 0
    ticket_content = ""
    ticket_id = ""
}

/^### ---TICKET-START---/ {
    in_ticket = 1
    ticket_content = ""
    next
}

/^### ---TICKET-END---/ {
    if (in_ticket && ticket_id != "") {
        # Write ticket to file
        filename = output_dir "/" ticket_id ".md"
        print ticket_content > filename
        close(filename)
        print "  Created: " ticket_id ".md"
        ticket_num++
    }
    in_ticket = 0
    ticket_id = ""
    next
}

in_ticket {
    # Capture ticket ID from yaml frontmatter
    if (match($0, /^id: ([A-Z0-9-]+)/, arr)) {
        ticket_id = arr[1]
    }
    ticket_content = ticket_content $0 "\n"
}

END {
    print ""
    print "Total tickets extracted: " ticket_num
}
' output_dir="$OUTPUT_DIR" "$INPUT_FILE"

# Create index file
echo "Creating index file..."
cat > "$OUTPUT_DIR/INDEX.md" << 'EOF'
# MCP Spawning Tickets Index

Generated from: mcp-spawning-v3.md

## Phase 0: Verification (GATE)

| Ticket | Title | Time | Priority |
|--------|-------|------|----------|
| MCP-SPAWN-001 | MCP Tool Availability Verification | 2h | CRITICAL |
| MCP-SPAWN-002 | CLI I/O Verification | 1h | HIGH |
| MCP-SPAWN-003 | Mock CLI Infrastructure | 3h | CRITICAL |

## Phase 1: Foundation

| Ticket | Title | Time | Priority |
|--------|-------|------|----------|
| MCP-SPAWN-004 | Environment Validation | 2h | CRITICAL |
| MCP-SPAWN-005 | Process Registry and Cleanup | 3h | CRITICAL |
| MCP-SPAWN-006 | Store Interface Extension | 2h | HIGH |
| MCP-SPAWN-007 | gogent-validate Nesting Check | 2h | CRITICAL |
| MCP-SPAWN-008 | spawn_agent Tool Implementation | 4h | CRITICAL |

## Phase 2: Integration

| Ticket | Title | Time | Priority |
|--------|-------|------|----------|
| MCP-SPAWN-009 | MCP Server Registration | 1h | HIGH |
| MCP-SPAWN-010 | Mozart Orchestrator Update | 2h | HIGH |
| MCP-SPAWN-011 | Review-Orchestrator Update | 2h | HIGH |

## Phase 3: Testing & Documentation

| Ticket | Title | Time | Priority |
|--------|-------|------|----------|
| MCP-SPAWN-012 | Integration Testing | 4h | HIGH |

## Dependency Graph

```
Phase 0 (GATE):
  MCP-SPAWN-001 (MCP Verification) ─┬─► MCP-SPAWN-002 (CLI I/O)
                                    │
                                    └─► MCP-SPAWN-004 (Env Validation)
                                            │
  MCP-SPAWN-002 ────────────────────────────┴─► MCP-SPAWN-003 (Mock CLI)

Phase 1:
  MCP-SPAWN-003 ─┬─► MCP-SPAWN-005 (Process Registry)
                 │
  MCP-SPAWN-004 ─┼─► MCP-SPAWN-006 (Store Extension)
                 │
                 └─► MCP-SPAWN-007 (Nesting Check)
                          │
  All above ─────────────►└─► MCP-SPAWN-008 (spawn_agent)

Phase 2:
  MCP-SPAWN-008 ─► MCP-SPAWN-009 (Registration) ─┬─► MCP-SPAWN-010 (Mozart)
                                                 │
                                                 └─► MCP-SPAWN-011 (Review)

Phase 3:
  MCP-SPAWN-010 + 011 ─► MCP-SPAWN-012 (Integration Tests)
```

## Total Estimated Time

- Phase 0: 6 hours
- Phase 1: 13 hours
- Phase 2: 5 hours
- Phase 3: 4 hours
- **Total: 28 hours** (~3.5 days of focused work)

Buffer recommended: 50% → **16-21 days** with normal interruptions.
EOF

echo "  Created: INDEX.md"
echo ""
echo "Done! Tickets available in: $OUTPUT_DIR"
```

### Usage

```bash
# Make script executable
chmod +x slice-tickets.sh

# Run from document directory
./slice-tickets.sh

# Or specify paths
./slice-tickets.sh /path/to/mcp-spawning-v3.md /path/to/output/
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 3.0 | 2026-02-04 | Beethoven (Braintrust Synthesis) | Initial v3 with full Einstein + Staff-Architect analysis |

---

**END OF DOCUMENT**

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-013
title: Agent Relationship Validation Integration
description: Integrate spawn_agent with agents-index.json for relationship validation before spawning.
status: pending
time_estimate: 3h
dependencies: [MCP-SPAWN-008]
phase: 2
tags: [validation, relationships, agents-index, phase-2]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-013: Agent Relationship Validation Integration

## Description

Integrate spawn_agent with the existing agents-index.json relationship fields to validate spawns before execution. This closes the gap between the agent-relationships-schema.json design and the actual implementation.

**Source**: agent-relationships-schema.json validation_rules, agent-relationships-examples.md §MCP spawn_agent Validation

## Why This Matters

agents-index.json already defines:
- `spawned_by`: Who can spawn this agent
- `can_spawn`: Who this agent can spawn
- `max_delegations`: Maximum children allowed

Without validation:
- Mozart could spawn agents not in its `can_spawn` list
- Agents could be spawned by unauthorized parents
- Resource limits (max_delegations) not enforced
- No warning when relationships are unexpected

## Task

1. Create agents-index.json loader with caching
2. Implement validateSpawnRelationship function
3. Integrate validation into spawn_agent before spawning
4. Handle errors (block) vs warnings (log but proceed)

## Files

- `packages/tui/src/spawn/agentConfig.ts` — Config loader
- `packages/tui/src/spawn/relationshipValidation.ts` — Validation logic
- `packages/tui/src/spawn/relationshipValidation.test.ts` — Tests
- `packages/tui/src/mcp/tools/spawnAgent.ts` — Integration

## Implementation

### Agent Config Loader (`packages/tui/src/spawn/agentConfig.ts`)

```typescript
import * as fs from "fs";
import * as path from "path";

/**
 * Relationship fields from agents-index.json
 */
export interface AgentRelationships {
  id: string;
  spawned_by?: string[];
  can_spawn?: string[];
  must_delegate?: boolean;
  min_delegations?: number;
  max_delegations?: number;
  inputs?: string[];
  outputs?: string[];
  outputs_to?: string[];
}

/**
 * Full agent config from agents-index.json
 */
export interface AgentConfig extends AgentRelationships {
  name: string;
  model: string;
  tier: number | string;
  triggers?: string[];
  tools?: string[];
  description?: string;
}

interface AgentsIndex {
  version: string;
  agents: AgentConfig[];
}

// Cache for agents-index.json
let cachedIndex: AgentsIndex | null = null;
let cacheTime: number = 0;
const CACHE_TTL_MS = 60000; // 1 minute

/**
 * Get the path to agents-index.json
 */
function getAgentsIndexPath(): string {
  // Check standard locations
  const locations = [
    path.join(process.cwd(), ".claude", "agents", "agents-index.json"),
    path.join(process.env.HOME || "", ".claude", "agents", "agents-index.json"),
  ];

  for (const loc of locations) {
    if (fs.existsSync(loc)) {
      return loc;
    }
  }

  throw new Error(
    "[agentConfig] agents-index.json not found. Checked: " + locations.join(", ")
  );
}

/**
 * Load agents-index.json with caching.
 */
export function loadAgentsIndex(): AgentsIndex {
  const now = Date.now();

  // Return cached if still valid
  if (cachedIndex && now - cacheTime < CACHE_TTL_MS) {
    return cachedIndex;
  }

  const indexPath = getAgentsIndexPath();
  const content = fs.readFileSync(indexPath, "utf-8");
  cachedIndex = JSON.parse(content) as AgentsIndex;
  cacheTime = now;

  return cachedIndex;
}

/**
 * Get config for a specific agent by ID.
 */
export function getAgentConfig(agentId: string): AgentConfig | null {
  const index = loadAgentsIndex();
  return index.agents.find((a) => a.id === agentId) || null;
}

/**
 * Clear the cache (for testing).
 */
export function clearAgentConfigCache(): void {
  cachedIndex = null;
  cacheTime = 0;
}
```

### Relationship Validation (`packages/tui/src/spawn/relationshipValidation.ts`)

```typescript
import { getAgentConfig, AgentConfig } from "./agentConfig";

export interface SpawnValidationResult {
  valid: boolean;
  errors: SpawnValidationError[];
  warnings: SpawnValidationWarning[];
}

export interface SpawnValidationError {
  code: string;
  message: string;
  field: string;
}

export interface SpawnValidationWarning {
  code: string;
  message: string;
  field: string;
}

/**
 * Validate spawn relationship between parent and child agent.
 *
 * Errors are blocking (spawn will fail).
 * Warnings are logged but spawn proceeds.
 *
 * @param parentType - Agent type of the parent (null if spawned by router)
 * @param childType - Agent type to spawn
 * @param currentChildCount - Number of children already spawned by parent
 */
export function validateSpawnRelationship(
  parentType: string | null | undefined,
  childType: string,
  currentChildCount: number = 0
): SpawnValidationResult {
  const errors: SpawnValidationError[] = [];
  const warnings: SpawnValidationWarning[] = [];

  const childConfig = getAgentConfig(childType);

  // Unknown child agent - allow with warning
  if (!childConfig) {
    warnings.push({
      code: "W_UNKNOWN_CHILD",
      message: `No config found for agent '${childType}' in agents-index.json`,
      field: "childType",
    });
    return { valid: true, errors, warnings };
  }

  // 1. Check spawned_by (who is allowed to spawn this child)
  if (childConfig.spawned_by && childConfig.spawned_by.length > 0) {
    const allowedParents = childConfig.spawned_by;

    // "any" means anyone can spawn
    if (!allowedParents.includes("any")) {
      // Router is represented as null parentType
      const parentIdentifier = parentType || "router";

      if (!allowedParents.includes(parentIdentifier)) {
        errors.push({
          code: "E_SPAWNED_BY_VIOLATION",
          message:
            `'${childType}' can only be spawned by [${allowedParents.join(", ")}], ` +
            `not '${parentIdentifier}'`,
          field: "spawned_by",
        });
      }
    }
  }

  // 2. Check can_spawn (is parent allowed to spawn this child)
  if (parentType) {
    const parentConfig = getAgentConfig(parentType);

    if (parentConfig) {
      // If parent has can_spawn defined, child must be in the list
      if (parentConfig.can_spawn && parentConfig.can_spawn.length > 0) {
        if (!parentConfig.can_spawn.includes(childType)) {
          errors.push({
            code: "E_CAN_SPAWN_VIOLATION",
            message:
              `'${parentType}' cannot spawn '${childType}'. ` +
              `Allowed: [${parentConfig.can_spawn.join(", ")}]`,
            field: "can_spawn",
          });
        }
      }

      // 3. Check max_delegations
      if (parentConfig.max_delegations !== undefined) {
        if (currentChildCount >= parentConfig.max_delegations) {
          errors.push({
            code: "E_MAX_DELEGATIONS_EXCEEDED",
            message:
              `'${parentType}' at max_delegations limit ` +
              `(${currentChildCount}/${parentConfig.max_delegations})`,
            field: "max_delegations",
          });
        }
      }
    } else {
      // Unknown parent - warn but allow
      warnings.push({
        code: "W_UNKNOWN_PARENT",
        message: `No config found for parent agent '${parentType}'`,
        field: "parentType",
      });
    }
  }

  // 4. Check invoked_by for additional context (warning only)
  if (childConfig.invoked_by) {
    const expectedInvoker = childConfig.invoked_by;

    // invoked_by can be: "router", "skill:<name>", "orchestrator:<id>", "any"
    if (expectedInvoker !== "any") {
      const actualInvoker = parentType ? `orchestrator:${parentType}` : "router";

      if (
        expectedInvoker !== actualInvoker &&
        expectedInvoker !== "router" &&
        !expectedInvoker.startsWith("skill:")
      ) {
        warnings.push({
          code: "W_INVOKED_BY_MISMATCH",
          message:
            `'${childType}' expects invoked_by='${expectedInvoker}', ` +
            `actual='${actualInvoker}'`,
          field: "invoked_by",
        });
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Format validation result for logging/display.
 */
export function formatValidationResult(result: SpawnValidationResult): string {
  const lines: string[] = [];

  if (result.valid) {
    lines.push("✅ Spawn validation passed");
  } else {
    lines.push("❌ Spawn validation failed");
  }

  if (result.errors.length > 0) {
    lines.push("\nErrors:");
    for (const err of result.errors) {
      lines.push(`  [${err.code}] ${err.message}`);
    }
  }

  if (result.warnings.length > 0) {
    lines.push("\nWarnings:");
    for (const warn of result.warnings) {
      lines.push(`  [${warn.code}] ${warn.message}`);
    }
  }

  return lines.join("\n");
}
```

### Integration into spawn_agent (`packages/tui/src/mcp/tools/spawnAgent.ts`)

```typescript
// Add imports at top
import {
  validateSpawnRelationship,
  formatValidationResult,
} from "../../spawn/relationshipValidation";

// Inside the spawn_agent handler, BEFORE spawning:

export const spawnAgent = tool(
  "spawn_agent",
  // ... description ...
  // ... schema ...
  async (args): Promise<{ content: Array<{ type: "text"; text: string }> }> => {
    const agentId = randomUUID();
    const registry = getProcessRegistry();
    const store = getAgentsStore();

    // Get parent info from store
    const parentId = args.parentId || process.env.GOGENT_PARENT_AGENT;
    const parentAgent = parentId ? store.get(parentId) : null;
    const parentType = parentAgent?.agentType;
    const currentChildCount = parentAgent?.childIds?.length || 0;

    // === RELATIONSHIP VALIDATION ===
    const validation = validateSpawnRelationship(
      parentType,
      args.agent,
      currentChildCount
    );

    // Log validation result
    if (!validation.valid || validation.warnings.length > 0) {
      console.log(formatValidationResult(validation));
    }

    // Block on validation errors
    if (!validation.valid) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                agentId: null,
                agent: args.agent,
                success: false,
                error: `Spawn validation failed: ${validation.errors
                  .map((e) => e.message)
                  .join("; ")}`,
                validationErrors: validation.errors,
                validationWarnings: validation.warnings,
              },
              null,
              2
            ),
          },
        ],
      };
    }

    // === END VALIDATION ===

    // Proceed with spawn...
    // (rest of existing implementation)
  }
);
```

### Tests (`packages/tui/src/spawn/relationshipValidation.test.ts`)

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  validateSpawnRelationship,
  formatValidationResult,
} from "./relationshipValidation";
import { clearAgentConfigCache } from "./agentConfig";

// Mock agents-index.json
vi.mock("fs", () => ({
  existsSync: vi.fn(() => true),
  readFileSync: vi.fn(() =>
    JSON.stringify({
      version: "test",
      agents: [
        {
          id: "mozart",
          name: "Mozart",
          model: "opus",
          tier: 3,
          can_spawn: ["einstein", "staff-architect-critical-review", "beethoven"],
          must_delegate: true,
          min_delegations: 3,
          max_delegations: 5,
        },
        {
          id: "einstein",
          name: "Einstein",
          model: "opus",
          tier: 3,
          spawned_by: ["mozart"],
          outputs_to: ["beethoven"],
        },
        {
          id: "beethoven",
          name: "Beethoven",
          model: "opus",
          tier: 3,
          spawned_by: ["mozart"],
          can_spawn: [],
        },
        {
          id: "review-orchestrator",
          name: "Review Orchestrator",
          model: "sonnet",
          tier: 2,
          can_spawn: ["backend-reviewer", "frontend-reviewer"],
          max_delegations: 4,
        },
        {
          id: "backend-reviewer",
          name: "Backend Reviewer",
          model: "haiku",
          tier: 1.5,
          spawned_by: ["review-orchestrator"],
        },
        {
          id: "codebase-search",
          name: "Codebase Search",
          model: "haiku",
          tier: 1,
          spawned_by: ["any"],
        },
      ],
    })
  ),
}));

describe("validateSpawnRelationship", () => {
  beforeEach(() => {
    clearAgentConfigCache();
  });

  describe("spawned_by validation", () => {
    it("should allow spawn when parent is in spawned_by list", () => {
      const result = validateSpawnRelationship("mozart", "einstein");

      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it("should block spawn when parent not in spawned_by list", () => {
      const result = validateSpawnRelationship("review-orchestrator", "einstein");

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_SPAWNED_BY_VIOLATION" })
      );
    });

    it("should allow spawn when spawned_by includes 'any'", () => {
      const result = validateSpawnRelationship("random-agent", "codebase-search");

      expect(result.valid).toBe(true);
    });

    it("should allow router to spawn when spawned_by includes 'router'", () => {
      // Add router to spawned_by for this test
      const result = validateSpawnRelationship(null, "codebase-search");

      expect(result.valid).toBe(true);
    });
  });

  describe("can_spawn validation", () => {
    it("should allow spawn when child is in parent can_spawn list", () => {
      const result = validateSpawnRelationship("mozart", "einstein");

      expect(result.valid).toBe(true);
    });

    it("should block spawn when child not in parent can_spawn list", () => {
      const result = validateSpawnRelationship("mozart", "backend-reviewer");

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_CAN_SPAWN_VIOLATION" })
      );
    });

    it("should allow spawn when parent has no can_spawn defined", () => {
      // Parent without can_spawn should allow anything
      const result = validateSpawnRelationship("backend-reviewer", "codebase-search");

      // backend-reviewer has no can_spawn, so no E_CAN_SPAWN error
      // but codebase-search has spawned_by: ["any"] so it's valid
      expect(result.errors.filter((e) => e.code === "E_CAN_SPAWN_VIOLATION")).toHaveLength(
        0
      );
    });
  });

  describe("max_delegations validation", () => {
    it("should allow spawn when under max_delegations", () => {
      const result = validateSpawnRelationship("mozart", "einstein", 2);

      expect(result.valid).toBe(true);
    });

    it("should block spawn when at max_delegations", () => {
      const result = validateSpawnRelationship("mozart", "beethoven", 5);

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({ code: "E_MAX_DELEGATIONS_EXCEEDED" })
      );
    });

    it("should block spawn when over max_delegations", () => {
      const result = validateSpawnRelationship("review-orchestrator", "backend-reviewer", 4);

      expect(result.valid).toBe(false);
      expect(result.errors).toContainEqual(
        expect.objectContaining({
          code: "E_MAX_DELEGATIONS_EXCEEDED",
          message: expect.stringContaining("4/4"),
        })
      );
    });
  });

  describe("unknown agents", () => {
    it("should warn but allow unknown child agent", () => {
      const result = validateSpawnRelationship("mozart", "unknown-agent");

      // Should be valid (allow) but with warning
      expect(result.valid).toBe(true);
      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_CHILD" })
      );
    });

    it("should warn but allow unknown parent agent", () => {
      const result = validateSpawnRelationship("unknown-parent", "codebase-search");

      expect(result.warnings).toContainEqual(
        expect.objectContaining({ code: "W_UNKNOWN_PARENT" })
      );
    });
  });
});

describe("formatValidationResult", () => {
  it("should format success result", () => {
    const result = { valid: true, errors: [], warnings: [] };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("✅ Spawn validation passed");
  });

  it("should format error result with details", () => {
    const result = {
      valid: false,
      errors: [
        { code: "E_TEST", message: "Test error", field: "test" },
      ],
      warnings: [],
    };
    const formatted = formatValidationResult(result);

    expect(formatted).toContain("❌ Spawn validation failed");
    expect(formatted).toContain("[E_TEST]");
    expect(formatted).toContain("Test error");
  });
});
```

## Acceptance Criteria

- [ ] agents-index.json loaded and cached (1 minute TTL)
- [ ] validateSpawnRelationship checks spawned_by, can_spawn, max_delegations
- [ ] Errors block spawn with clear message
- [ ] Warnings logged but spawn proceeds
- [ ] Integration with spawn_agent works correctly
- [ ] All tests pass: `npm test -- src/spawn/relationshipValidation.test.ts`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file: `packages/tui/src/spawn/relationshipValidation.test.ts`
- [ ] Number of test functions: 12
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: Mozart spawns Einstein (allowed), Mozart spawns backend-reviewer (blocked)

## Schema Alignment

This ticket aligns mcp-spawning-v3 with agent-relationships-schema.json:

| Schema Field | Validated? | Behavior |
|--------------|------------|----------|
| `spawned_by` | ✅ | Block if parent not in list |
| `can_spawn` | ✅ | Block if child not in list |
| `max_delegations` | ✅ | Block if count exceeded |
| `invoked_by` | ✅ | Warning if mismatch |
| `inputs/outputs` | ❌ | Future: data flow validation |
| `outputs_to` | ❌ | Future: visualization |

### ---TICKET-END---

### ---TICKET-START---
```yaml
---
id: MCP-SPAWN-014
title: Delegation Requirement Enforcement
description: Enforce must_delegate and min_delegations at orchestrator completion via gogent-orchestrator-guard hook.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-007, MCP-SPAWN-013]
phase: 2
tags: [hooks, go, delegation, enforcement, phase-2]
needs_planning: false
agent: go-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-014: Delegation Requirement Enforcement

## Description

Enforce `must_delegate` and `min_delegations` requirements at orchestrator completion. When an orchestrator with `must_delegate: true` completes, verify it spawned at least `min_delegations` children. Block completion if requirement not met.

**Source**: agent-relationships-schema.json validation_rules.delegation_requirement

## Why This Matters

agents-index.json defines delegation requirements:
- `mozart`: must_delegate=true, min_delegations=3 (Einstein, Staff-Architect, Beethoven)
- `review-orchestrator`: must_delegate=true, min_delegations=2
- `impl-manager`: must_delegate=true, min_delegations=1

Without enforcement:
- Orchestrators could complete without spawning required specialists
- Braintrust could return without Einstein/Staff-Architect analysis
- Review could return without any reviewers running

## Task

1. Extend gogent-orchestrator-guard hook (or create if not exists)
2. Load agents-index.json for delegation requirements
3. At SubagentStop, check must_delegate and min_delegations
4. Block completion with guidance if requirements not met

## Files

- `cmd/gogent-orchestrator-guard/main.go` — Hook implementation
- `pkg/routing/delegation.go` — Delegation validation logic
- `pkg/routing/delegation_test.go` — Tests

## Implementation

### Delegation Validation (`pkg/routing/delegation.go`)

```go
package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AgentDelegationConfig holds delegation-related fields from agents-index.json
type AgentDelegationConfig struct {
	ID             string   `json:"id"`
	MustDelegate   bool     `json:"must_delegate,omitempty"`
	MinDelegations int      `json:"min_delegations,omitempty"`
	MaxDelegations int      `json:"max_delegations,omitempty"`
	CanSpawn       []string `json:"can_spawn,omitempty"`
}

// AgentsIndex represents the agents-index.json structure
type AgentsIndex struct {
	Version string                  `json:"version"`
	Agents  []AgentDelegationConfig `json:"agents"`
}

// DelegationValidationResult holds the result of delegation validation
type DelegationValidationResult struct {
	Valid       bool
	AgentID     string
	Required    int
	Actual      int
	Message     string
	Suggestion  string
}

var cachedAgentsIndex *AgentsIndex

// LoadAgentsIndex loads agents-index.json with caching
func LoadAgentsIndex() (*AgentsIndex, error) {
	if cachedAgentsIndex != nil {
		return cachedAgentsIndex, nil
	}

	// Find agents-index.json
	locations := []string{
		filepath.Join(os.Getenv("CLAUDE_PROJECT_DIR"), ".claude", "agents", "agents-index.json"),
		filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json"),
	}

	var indexPath string
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			indexPath = loc
			break
		}
	}

	if indexPath == "" {
		return nil, fmt.Errorf("[delegation] agents-index.json not found")
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("[delegation] failed to read agents-index.json: %w", err)
	}

	var index AgentsIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("[delegation] failed to parse agents-index.json: %w", err)
	}

	cachedAgentsIndex = &index
	return cachedAgentsIndex, nil
}

// GetAgentDelegationConfig retrieves delegation config for an agent
func GetAgentDelegationConfig(agentID string) (*AgentDelegationConfig, error) {
	index, err := LoadAgentsIndex()
	if err != nil {
		return nil, err
	}

	for _, agent := range index.Agents {
		if agent.ID == agentID {
			return &agent, nil
		}
	}

	return nil, nil // Not found, not an error
}

// ValidateDelegationRequirement checks if an orchestrator met its delegation requirements
func ValidateDelegationRequirement(agentType string, childCount int) *DelegationValidationResult {
	config, err := GetAgentDelegationConfig(agentType)
	if err != nil {
		// Can't validate, allow to proceed
		return &DelegationValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("Could not load config for %s: %v", agentType, err),
		}
	}

	if config == nil {
		// Unknown agent, allow
		return &DelegationValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("No config found for agent '%s'", agentType),
		}
	}

	// Check must_delegate
	if !config.MustDelegate {
		return &DelegationValidationResult{
			Valid:   true,
			AgentID: agentType,
			Message: fmt.Sprintf("%s does not require delegation", agentType),
		}
	}

	// Check min_delegations
	if childCount < config.MinDelegations {
		return &DelegationValidationResult{
			Valid:    false,
			AgentID:  agentType,
			Required: config.MinDelegations,
			Actual:   childCount,
			Message: fmt.Sprintf(
				"%s requires at least %d delegations but only spawned %d",
				agentType, config.MinDelegations, childCount,
			),
			Suggestion: fmt.Sprintf(
				"Spawn more agents before completing. Expected: %v",
				config.CanSpawn,
			),
		}
	}

	return &DelegationValidationResult{
		Valid:    true,
		AgentID:  agentType,
		Required: config.MinDelegations,
		Actual:   childCount,
		Message: fmt.Sprintf(
			"%s met delegation requirement (%d/%d)",
			agentType, childCount, config.MinDelegations,
		),
	}
}

// BlockResponseForDelegation creates the standard block response for delegation violations
func BlockResponseForDelegation(result *DelegationValidationResult) map[string]interface{} {
	return map[string]interface{}{
		"decision": "block",
		"reason":   result.Message,
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "SubagentStop",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "delegation_requirement_not_met",
			"agentId":                  result.AgentID,
			"requiredDelegations":      result.Required,
			"actualDelegations":        result.Actual,
			"suggestion":               result.Suggestion,
		},
	}
}

// ClearAgentsIndexCache clears the cached index (for testing)
func ClearAgentsIndexCache() {
	cachedAgentsIndex = nil
}
```

### Hook Implementation (`cmd/gogent-orchestrator-guard/main.go`)

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/doktersmol/gogent-fortress/pkg/routing"
)

// SubagentStopEvent represents the hook input for SubagentStop
type SubagentStopEvent struct {
	SessionID  string `json:"session_id"`
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
	ChildCount int    `json:"child_count"`
	Status     string `json:"status"` // "complete", "error", "timeout"
}

func main() {
	// Read event from stdin
	var event SubagentStopEvent
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&event); err != nil {
		outputError("Failed to parse hook input", err)
		return
	}

	// Only validate on successful completion
	if event.Status != "complete" {
		outputAllow("Agent did not complete successfully, skipping delegation check")
		return
	}

	// Validate delegation requirement
	result := routing.ValidateDelegationRequirement(event.AgentType, event.ChildCount)

	if !result.Valid {
		// Log violation for telemetry
		logDelegationViolation(event, result)

		// Output block response
		response := routing.BlockResponseForDelegation(result)
		outputJSON(response)
		return
	}

	// Log successful validation
	logDelegationSuccess(event, result)

	// Allow completion
	outputAllow(result.Message)
}

func outputAllow(message string) {
	response := map[string]interface{}{
		"decision": "allow",
		"message":  message,
	}
	outputJSON(response)
}

func outputError(message string, err error) {
	response := map[string]interface{}{
		"decision": "allow", // Allow on error (fail-open for this check)
		"error":    fmt.Sprintf("%s: %v", message, err),
	}
	outputJSON(response)
}

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func logDelegationViolation(event SubagentStopEvent, result *routing.DelegationValidationResult) {
	telemetry := map[string]interface{}{
		"timestamp":   getCurrentTimestamp(),
		"event":       "delegation_violation",
		"session_id":  event.SessionID,
		"agent_id":    event.AgentID,
		"agent_type":  event.AgentType,
		"required":    result.Required,
		"actual":      result.Actual,
		"child_count": event.ChildCount,
	}

	appendTelemetry("delegation-violations.jsonl", telemetry)
}

func logDelegationSuccess(event SubagentStopEvent, result *routing.DelegationValidationResult) {
	telemetry := map[string]interface{}{
		"timestamp":   getCurrentTimestamp(),
		"event":       "delegation_met",
		"session_id":  event.SessionID,
		"agent_id":    event.AgentID,
		"agent_type":  event.AgentType,
		"required":    result.Required,
		"actual":      result.Actual,
	}

	appendTelemetry("delegation-success.jsonl", telemetry)
}

// ... helper functions for timestamp and telemetry append
```

### Tests (`pkg/routing/delegation_test.go`)

```go
package routing

import (
	"os"
	"testing"
)

func TestValidateDelegationRequirement(t *testing.T) {
	// Set up mock agents-index.json
	mockIndex := `{
		"version": "test",
		"agents": [
			{
				"id": "mozart",
				"must_delegate": true,
				"min_delegations": 3,
				"can_spawn": ["einstein", "staff-architect", "beethoven"]
			},
			{
				"id": "review-orchestrator",
				"must_delegate": true,
				"min_delegations": 2,
				"can_spawn": ["backend-reviewer", "frontend-reviewer"]
			},
			{
				"id": "go-pro",
				"must_delegate": false
			}
		]
	}`

	// Write mock file
	tmpDir := t.TempDir()
	indexPath := tmpDir + "/agents-index.json"
	os.WriteFile(indexPath, []byte(mockIndex), 0644)
	os.Setenv("HOME", tmpDir)
	os.MkdirAll(tmpDir+"/.claude/agents", 0755)
	os.WriteFile(tmpDir+"/.claude/agents/agents-index.json", []byte(mockIndex), 0644)

	defer ClearAgentsIndexCache()

	tests := []struct {
		name       string
		agentType  string
		childCount int
		wantValid  bool
	}{
		{
			name:       "mozart with 3 children - valid",
			agentType:  "mozart",
			childCount: 3,
			wantValid:  true,
		},
		{
			name:       "mozart with 2 children - invalid",
			agentType:  "mozart",
			childCount: 2,
			wantValid:  false,
		},
		{
			name:       "mozart with 0 children - invalid",
			agentType:  "mozart",
			childCount: 0,
			wantValid:  false,
		},
		{
			name:       "review-orchestrator with 2 children - valid",
			agentType:  "review-orchestrator",
			childCount: 2,
			wantValid:  true,
		},
		{
			name:       "review-orchestrator with 1 child - invalid",
			agentType:  "review-orchestrator",
			childCount: 1,
			wantValid:  false,
		},
		{
			name:       "go-pro (no must_delegate) with 0 children - valid",
			agentType:  "go-pro",
			childCount: 0,
			wantValid:  true,
		},
		{
			name:       "unknown agent - valid (allow)",
			agentType:  "unknown-agent",
			childCount: 0,
			wantValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClearAgentsIndexCache()

			result := ValidateDelegationRequirement(tt.agentType, tt.childCount)

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateDelegationRequirement(%s, %d) = %v, want %v. Message: %s",
					tt.agentType, tt.childCount, result.Valid, tt.wantValid, result.Message)
			}
		})
	}
}

func TestBlockResponseForDelegation(t *testing.T) {
	result := &DelegationValidationResult{
		Valid:      false,
		AgentID:    "mozart",
		Required:   3,
		Actual:     2,
		Message:    "mozart requires at least 3 delegations but only spawned 2",
		Suggestion: "Spawn more agents",
	}

	response := BlockResponseForDelegation(result)

	if response["decision"] != "block" {
		t.Errorf("expected decision 'block', got %v", response["decision"])
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("hookSpecificOutput not a map")
	}

	if hookOutput["requiredDelegations"] != 3 {
		t.Errorf("expected requiredDelegations 3, got %v", hookOutput["requiredDelegations"])
	}

	if hookOutput["actualDelegations"] != 2 {
		t.Errorf("expected actualDelegations 2, got %v", hookOutput["actualDelegations"])
	}
}
```

## Acceptance Criteria

- [ ] Hook reads SubagentStop event from stdin
- [ ] Loads agents-index.json for delegation config
- [ ] Validates must_delegate and min_delegations
- [ ] Blocks completion with clear message if requirements not met
- [ ] Allows completion if requirements met or not applicable
- [ ] Logs both violations and successes for telemetry
- [ ] All tests pass: `go test ./pkg/routing/...`
- [ ] Code coverage ≥80%

## Test Deliverables

- [ ] Test file: `pkg/routing/delegation_test.go`
- [ ] Number of test functions: 2
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: Mozart completes with <3 children (blocked), Mozart completes with 3+ children (allowed)

## Schema Alignment

This ticket enforces agent-relationships-schema.json delegation rules:

| Schema Rule | Enforcement |
|-------------|-------------|
| `must_delegate` | Check at SubagentStop |
| `min_delegations` | Block if childCount < min |
| `max_delegations` | Already enforced in MCP-SPAWN-013 at spawn time |

### ---TICKET-END---
