# MCP-Based Agent Spawning Architecture: Critical Review & Synthesis

**Document ID**: `braintrust-synthesis-mcp-spawning-2026-02-04`
**Status**: REFERENCE FOR NEXT BRAINTRUST ITERATION
**Authors**: Einstein (theoretical), Staff-Architect (practical), Beethoven (synthesis)
**Date**: 2026-02-04

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Original Proposed Solution](#3-original-proposed-solution)
4. [Einstein Theoretical Analysis](#4-einstein-theoretical-analysis)
5. [Staff-Architect Practical Review](#5-staff-architect-practical-review)
6. [Convergent Findings](#6-convergent-findings)
7. [User's Refined Architecture Vision](#7-users-refined-architecture-vision)
8. [Synthesis: Hybrid Task/CLI Architecture](#8-synthesis-hybrid-taskcli-architecture)
9. [Schema-Driven I/O Design](#9-schema-driven-io-design)
10. [Session Containerization Model](#10-session-containerization-model)
11. [CLI Flag Strategy](#11-cli-flag-strategy)
12. [Hook-Based Level Detection](#12-hook-based-level-detection)
13. [Implementation Roadmap](#13-implementation-roadmap)
14. [Risk Matrix](#14-risk-matrix)
15. [Open Questions for Next Braintrust](#15-open-questions-for-next-braintrust)
16. [Appendices](#16-appendices)

---

## 1. Executive Summary

### The Problem
Claude Code's Task tool is only available at nesting level 0 (router). Subagents cannot spawn sub-subagents, breaking orchestrator patterns like Braintrust and review-orchestrator.

### Original Solution
MCP tools in TUI spawn CLI processes to bypass Task limitation.

### Critical Finding
The original solution is **theoretically sound** but:
1. One CRITICAL assumption (MCP availability in subagents) is **unverified**
2. Treats CLI spawning as a "workaround" when it's actually a **paradigm shift**
3. Underestimates complexity (5-7 days → 10+ days realistic)

### Refined Architecture (User's Vision)
A **hybrid approach** that:
- Preserves Task() for all direct agent invocations
- Uses `claude -p` only when subagents need to spawn sub-subagents
- Hook detects nesting level and blocks/redirects appropriately
- Schema-driven JSON I/O for all `claude -p` invocations
- Session containerization for traceability and TUI visualization

### Verdict
**CONDITIONAL APPROVAL** - Verify assumptions, implement hybrid approach with schema-driven I/O.

---

## 2. Problem Statement

### 2.1 The Limitation

```
Claude Code Nesting Levels:

Level 0 (Router)     → Has Task tool     → Can spawn Level 1
Level 1 (Subagent)   → NO Task tool      → Cannot spawn Level 2
Level 2 (Would-be)   → N/A               → Cannot exist via Task
```

### 2.2 Evidence

| Evidence | Source | Strength |
|----------|--------|----------|
| Mozart stated "I do not have access to the Task tool" | Session transcript | High |
| review-orchestrator used TaskCreate instead of Task | Behavioral observation | Medium |
| Router Task() succeeded | Controlled test | High |

### 2.3 Impact

Affected workflows:
- **Braintrust**: Mozart → Einstein + Staff-Architect → Beethoven
- **Review**: review-orchestrator → backend/frontend/standards reviewers
- **Ticket**: impl-manager → language-specific implementation agents
- **Any future multi-level orchestration**

---

## 3. Original Proposed Solution

### 3.1 Core Mechanism

```
Subagent → MCP spawn_agent tool → TUI Process → spawn(claude CLI) → New Session
```

### 3.2 Key Assumptions

1. MCP tools are available to subagents (UNVERIFIED - CRITICAL)
2. CLI spawning provides full tool access (VERIFIED - by design)
3. Context can be serialized via prompts (TRUE - but burden is high)

### 3.3 Proposed Implementation

From `mcp-spawning-architecture-v2-2026-02-04.md`:

```typescript
export const spawnAgentTool = tool(
  "spawn_agent",
  "Spawn a Claude Code subagent with full tool access.",
  { /* args */ },
  async (args) => {
    const proc = spawn('claude', [
      '-p', args.prompt,
      '--output-format', 'stream-json',
      '--dangerously-skip-permissions',
      '--max-turns', '30'
    ], { cwd: args.workdir });
    // ... collect output
  }
);
```

### 3.4 Pattern Comparison

| Aspect | Pattern A: Ephemeral | Pattern B: Pooled |
|--------|---------------------|-------------------|
| Complexity | LOW | HIGH |
| Latency | 5-10s/spawn | <1s (warm) |
| Memory | Transient | Persistent |
| Recommended | Phase 1 | Phase 2+ if needed |

---

## 4. Einstein Theoretical Analysis

### 4.1 Fundamental Assumptions

#### Assumption 1: Task() Unavailable to Subagents
**Confidence**: HIGH but UNVERIFIED EMPIRICALLY

The limitation was diagnosed through behavioral observation, not controlled testing. A subagent should be explicitly tested:

```javascript
Task({
  description: "Test nesting capability",
  prompt: "Attempt to spawn a subagent via Task() and report exact result"
})
```

#### Assumption 2: MCP Tools Available to Subagents
**Confidence**: UNKNOWN - CRITICAL GAP

The plan assumes MCP tools bypass Claude Code's restriction system. This is the **most critical assumption** and has **zero empirical evidence**.

**Theoretical counterargument**: Claude Code may filter ALL tools for subagents, including MCP tools.

**Required verification**:
1. Register simple MCP tool (e.g., `test_mcp_ping`)
2. Spawn subagent via Task()
3. Have subagent attempt MCP tool invocation
4. Observe result

### 4.2 Conceptual Framework

#### The Paradigm Shift

CLI spawning is NOT equivalent to Task() spawning:

| Dimension | Task() Subagent | CLI-Spawned Agent |
|-----------|-----------------|-------------------|
| Context | Inherited (implicit) | Injected (explicit) |
| Tool restrictions | Nesting-level dependent | Full (Level 0) |
| Process model | In-process/managed | Separate OS process |
| Session continuity | Same session | NEW session |
| Cost tracking | Unified | Fragmented |

**Key insight**: CLI spawns create **independent sessions**, not child subagents. The "Epic → Parent → Child" hierarchy is a **logical abstraction**, not a process reality.

### 4.3 Data Flow Analysis

#### What Is Preserved
- Working directory (via `cwd`)
- CLAUDE.md (reloaded on CLI startup)
- Hooks (fire independently per session)
- Environment variables (explicitly passed)

#### What Is Lost
- Conversation history
- Session-level cost tracking
- Unified agent tree (must be reconstructed)
- Incremental context (must be serialized)

### 4.4 Risk Assessment

| Failure Mode | Probability | Impact | Detection |
|--------------|-------------|--------|-----------|
| MCP tools unavailable to subagents | MEDIUM | CRITICAL | Immediate architecture failure |
| TUI crash orphans processes | LOW | HIGH | Process accumulation |
| Recursive spawning exhausts resources | LOW | HIGH | OOM killer |
| Context serialization errors | MEDIUM | MEDIUM | Wrong agent output |

### 4.5 Alternative Approaches Not Considered

1. **Flat coordination**: Router handles all spawning, orchestrators return plans
2. **File-based coordination**: Task files polled by router
3. **API direct access**: Custom tool definitions via Anthropic Messages API
4. **Feature request**: Ask Anthropic to fix nested Task() limitation

### 4.6 Severity-Rated Concerns

| ID | Concern | Severity |
|----|---------|----------|
| E-C1 | MCP tool availability unverified | CRITICAL |
| E-C2 | Task() limitation not empirically verified | MEDIUM |
| E-H1 | CLI spawning is paradigm shift, not workaround | HIGH |
| E-H2 | Context serialization burden on orchestrators | HIGH |
| E-H3 | Session fragmentation breaks unified tracking | HIGH |
| E-M1 | Relationship schema needs MCP-CLI extensions | MEDIUM |
| E-M2 | No depth/concurrency limits | MEDIUM |
| E-L1 | Sequential spawn dependencies not addressed | LOW |

---

## 5. Staff-Architect Practical Review

### 5.1 Layer 1: Assumption Validation

#### TUI Infrastructure
- Current Agent interface: 10 fields
- Proposed Agent interface: 24 fields
- **Finding**: Backward-incompatible change requires migration strategy

#### Node.js child_process Patterns
```typescript
// INCORRECT (in original plan)
spawn('claude', cliArgs, { shell: true });  // Shell injection risk
'-p', `"$(cat ${promptFile})"`              // Command substitution fails

// CORRECT
spawn('claude', cliArgs, { stdio: ['pipe', 'pipe', 'pipe'] });
proc.stdin.write(prompt);  // Pipe prompt via stdin
proc.stdin.end();
```

#### Security Concern
`--dangerously-skip-permissions` bypasses ALL permission checks.

**Better alternatives from CLI help**:
- `--permission-mode delegate` - Safer permission handling
- `--allowedTools "Bash(git:*) Edit Read Glob Grep"` - Restrict tools per agent

### 5.2 Layer 2: Dependency Analysis

#### External Dependencies (MISSING VALIDATION)

| Dependency | Required | Risk if Missing |
|------------|----------|-----------------|
| `claude` CLI in PATH | Yes | Spawn fails |
| `GOGENT_*` env vars | Optional | Hooks fail silently |
| `/tmp` writable | Yes | Prompt file creation fails |
| `agents-index.json` | Yes | Agent lookup fails |

### 5.3 Layer 3: Failure Modes

| Failure | Current Mitigation | Gap |
|---------|-------------------|-----|
| CLI hangs | Timeout + SIGTERM | No SIGKILL escalation |
| TUI crashes | None | Orphan processes |
| Signals (Ctrl+C) | None | Children continue running |
| Memory leak | None | Unbounded stream buffer |

### 5.4 Layer 4: Cost-Benefit

| Factor | Assessment |
|--------|------------|
| Cold start latency | 5-10s acceptable with parallel spawning |
| Memory overhead | ~100-200MB per CLI process |
| Development estimate | 5-7 days optimistic → **10 days realistic** |
| Not implementing | Braintrust/review broken, competitive disadvantage |

### 5.5 Layer 5: Testing Gaps

| Test Type | Strategy Defined | Gap |
|-----------|-----------------|-----|
| Unit tests | None | Need mock CLI |
| Integration tests | None | API credits consumed |
| Timeout/kill tests | None | Hard to test without slow tests |

### 5.6 Layer 6: Architecture Smells

1. **God Object**: `spawn_agent` does 11 things (validate, track, spawn, stream, parse, update, cleanup...)
2. **Implicit Coupling**: Environment variables for parent-child relationships
3. **Potential Overengineering**: Epic concept may be premature

### 5.7 Layer 7: Implementation Readiness

| Item | Status |
|------|--------|
| TypeScript interfaces | Incomplete (missing states, types) |
| Acceptance criteria | Not defined |
| Rollback plan | Not defined |

### 5.8 Severity-Rated Concerns

| ID | Concern | Severity |
|----|---------|----------|
| S-C1 | `--dangerously-skip-permissions` security | CRITICAL |
| S-C2 | Missing dependency validation | CRITICAL |
| S-C3 | Orphan processes on crash | CRITICAL |
| S-C4 | No mock test strategy | CRITICAL |
| S-C5 | No rollback plan | CRITICAL |
| S-H1 | `shell: true` + `spawn()` incorrect | HIGH |
| S-H2 | Circular reference tool ↔ store | HIGH |
| S-H3 | Signals not propagated | HIGH |
| S-H4 | Unbounded stream buffer | HIGH |
| S-H5 | spawn_agent too many responsibilities | HIGH |
| S-H6 | Implementation ambiguities | HIGH |
| S-H7 | No acceptance criteria | HIGH |
| S-M1 | Store interface incompatibility | MEDIUM |
| S-M2 | Development estimate optimistic | MEDIUM |

---

## 6. Convergent Findings

### Both Analysts Agree On:

| Finding | Einstein | Staff-Architect | Resolution |
|---------|----------|-----------------|------------|
| MCP availability unverified | E-C1 | S-C2 | **Verify before implementing** |
| CLI ≠ Task paradigm | E-H1 | Implicit | **Document as architectural decision** |
| Process cleanup on crash | E-risk | S-C3 | **Register shutdown handler** |
| Context serialization burden | E-H2 | Implicit | **Schema-driven I/O** |
| Estimate optimistic | N/A | S-M2 | **Budget 10 days** |

---

## 7. User's Refined Architecture Vision

### 7.1 Key Insight

The user proposes a **hybrid approach**:

> "The TUI (epic) can spawn orchestrators directly with Task(). ALL agents should be spawnable by Task() for direct implementation. BUT - if a subagent is specifically trying to spawn a subagent - there should be a hook that blocks Task() with feedback to invoke via the claude -p system."

### 7.2 Level-Based Routing

```
Level 0 (Router/TUI)
    │
    ├── Task(orchestrator) → Works natively
    │       │
    │       └── Level 1 (Orchestrator)
    │               │
    │               └── Task(specialist) → BLOCKED by hook
    │                       ↓
    │                   Hook feedback: "Use claude -p"
    │                       ↓
    │                   claude -p (via TUI) → Level 2 agent
```

### 7.3 Schema-Driven I/O

> "The infra needs to support standard JSON structures for stdin for each use case - whether it be implementation task, ticket system, braintrust etc. We need to have this containerized in schemas/"

```
schemas/
├── braintrust/
│   ├── input.schema.json
│   └── output.schema.json
├── review/
│   ├── input.schema.json
│   └── output.schema.json
├── implementation/
│   ├── input.schema.json
│   └── output.schema.json
└── ticket/
    ├── input.schema.json
    └── output.schema.json
```

### 7.4 Session Containerization

> "Have each orchestrator create a session_id in sessions/ to containerize all I/O, use the draft to compose the JSON(s) into each relevant claude-p task, have each output their JSON into the session."

```
sessions/
└── braintrust-2026-02-04-abc123/
    ├── session.json           # Metadata
    ├── inputs/
    │   ├── problem-brief.json
    │   ├── einstein-input.json
    │   └── staff-architect-input.json
    ├── outputs/
    │   ├── einstein-output.json
    │   ├── staff-architect-output.json
    │   └── beethoven-synthesis.json
    └── logs/
        ├── einstein.log
        └── staff-architect.log
```

### 7.5 Benefits

1. **Traceability**: All I/O for a workflow in one place
2. **TUI Visualization**: Easy to display session tree
3. **Debugging**: Inspect exact inputs/outputs
4. **Replay**: Re-run with modified inputs
5. **Cost Attribution**: Per-session cost tracking

---

## 8. Synthesis: Hybrid Task/CLI Architecture

### 8.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│ TUI (Epic Manager)                                                  │
│                                                                     │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐   │
│  │ Session Manager │   │ Schema Loader   │   │ Process Manager │   │
│  │                 │   │                 │   │                 │   │
│  │ sessions/       │   │ schemas/        │   │ activeProcesses │   │
│  └────────┬────────┘   └────────┬────────┘   └────────┬────────┘   │
│           │                     │                     │             │
│           └──────────┬──────────┴──────────┬──────────┘             │
│                      │                     │                        │
│  ┌───────────────────▼─────────────────────▼────────────────────┐   │
│  │                    Spawn Dispatcher                           │   │
│  │                                                               │   │
│  │   Level 0 request → Task() directly                          │   │
│  │   Level 1+ request → claude -p with schema-validated JSON    │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌─────────────────────┐      ┌─────────────────────────┐
│ Task() Subagent     │      │ claude -p Process       │
│ (Level 1)           │      │ (Level 2+)              │
│                     │      │                         │
│ - Native context    │      │ - Schema-validated JSON │
│ - Restricted tools  │      │ - Full tool access      │
│ - Same session      │      │ - New session           │
│ - Hook blocks       │      │ - Output to session/    │
│   nested Task()     │      │                         │
└─────────────────────┘      └─────────────────────────┘
```

### 8.2 Decision Tree

```
Agent spawn request arrives
    │
    ├── Is this from Level 0 (Router/TUI)?
    │       YES → Use Task() directly (native behavior)
    │
    ├── Is this from Level 1+ (Subagent)?
    │       │
    │       ├── Is it via Task() tool?
    │       │       YES → BLOCK with hook
    │       │             Inject: "Use claude -p via MCP spawn_agent"
    │       │
    │       └── Is it via MCP spawn_agent?
    │               YES → Validate schema → Create session → Spawn CLI
    │
    └── Unknown level?
            → Log warning, allow Task() (fail-safe)
```

### 8.3 Hook Modification for Level Detection

Current `gogent-validate` blocks Task(opus) and validates subagent_type.

**New capability needed**: Detect nesting level and block all Task() from Level 1+.

```go
// pkg/routing/task_validation.go

func ValidateTaskInvocation(schema *Schema, taskInput map[string]interface{}, sessionID string) *TaskValidationResult {
    // NEW: Check nesting level
    nestingLevel := getNestingLevel()  // From env var GOGENT_NESTING_LEVEL

    if nestingLevel > 0 {
        return &TaskValidationResult{
            Allowed: false,
            BlockReason: fmt.Sprintf(
                "Task() blocked at nesting level %d. Subagents cannot spawn sub-subagents via Task(). "+
                "Use MCP spawn_agent tool instead, which invokes claude -p with schema-validated JSON.",
                nestingLevel,
            ),
            Recommendation: "Call mcp__gofortress__spawn_agent({agent: '...', schema: '...', input: {...}})",
        }
    }

    // ... existing opus/allowlist validation
}
```

---

## 9. Schema-Driven I/O Design

### 9.1 Schema Structure

Each workflow type has input and output schemas:

```json
// schemas/braintrust/input.schema.json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Braintrust Agent Input",
  "type": "object",
  "required": ["agent", "task", "context"],
  "properties": {
    "agent": {
      "type": "string",
      "enum": ["einstein", "staff-architect-critical-review", "beethoven"]
    },
    "task": {
      "type": "object",
      "required": ["type", "description"],
      "properties": {
        "type": { "type": "string" },
        "description": { "type": "string" },
        "constraints": { "type": "array", "items": { "type": "string" } }
      }
    },
    "context": {
      "type": "object",
      "properties": {
        "problem_brief": { "type": "string" },
        "relevant_files": { "type": "array", "items": { "type": "string" } },
        "prior_analyses": { "type": "array" }
      }
    },
    "session": {
      "type": "object",
      "required": ["id", "epic_id"],
      "properties": {
        "id": { "type": "string", "format": "uuid" },
        "epic_id": { "type": "string" },
        "parent_agent": { "type": "string" }
      }
    }
  }
}
```

```json
// schemas/braintrust/output.schema.json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Braintrust Agent Output",
  "type": "object",
  "required": ["status", "result"],
  "properties": {
    "status": {
      "type": "string",
      "enum": ["success", "error", "partial"]
    },
    "result": {
      "type": "object",
      "properties": {
        "summary": { "type": "string" },
        "findings": { "type": "array" },
        "recommendations": { "type": "array" },
        "severity_breakdown": { "type": "object" }
      }
    },
    "metadata": {
      "type": "object",
      "properties": {
        "duration_ms": { "type": "number" },
        "tokens_used": { "type": "object" },
        "cost_usd": { "type": "number" },
        "tools_called": { "type": "number" }
      }
    },
    "error": {
      "type": "object",
      "properties": {
        "code": { "type": "string" },
        "message": { "type": "string" },
        "stack": { "type": "string" }
      }
    }
  }
}
```

### 9.2 Schema Registry

```typescript
// packages/tui/src/schemas/registry.ts

interface SchemaRegistry {
  workflows: {
    braintrust: { input: JSONSchema; output: JSONSchema };
    review: { input: JSONSchema; output: JSONSchema };
    implementation: { input: JSONSchema; output: JSONSchema };
    ticket: { input: JSONSchema; output: JSONSchema };
  };

  validate(workflow: string, direction: 'input' | 'output', data: unknown): ValidationResult;
  getSchema(workflow: string, direction: 'input' | 'output'): JSONSchema;
}
```

### 9.3 Using `--json-schema` Flag

Claude CLI supports structured output validation:

```bash
claude -p "Analyze the problem" \
  --output-format json \
  --json-schema '{"type":"object","properties":{"findings":{"type":"array"},"summary":{"type":"string"}},"required":["findings","summary"]}'
```

This ensures agent outputs conform to expected structure.

---

## 10. Session Containerization Model

### 10.1 Session Lifecycle

```
1. Orchestrator starts
   └── Create session directory: sessions/{workflow}-{timestamp}-{uuid}/

2. Orchestrator prepares child agent invocation
   └── Write input JSON: sessions/.../inputs/{agent}-input.json
   └── Validate against schema

3. Orchestrator spawns child via claude -p
   └── Pass input via stdin
   └── Set --output-format json
   └── Set --json-schema for output validation

4. Child agent executes
   └── Outputs structured JSON to stdout

5. TUI collects output
   └── Write to sessions/.../outputs/{agent}-output.json
   └── Validate against output schema
   └── Update session.json with status

6. Orchestrator receives result
   └── Continues workflow or handles error

7. Workflow completes
   └── Update session.json with final status
   └── Calculate total cost
   └── Archive if configured
```

### 10.2 Session Directory Structure

```
sessions/
└── braintrust-2026-02-04-abc123/
    ├── session.json
    │   {
    │     "id": "abc123",
    │     "workflow": "braintrust",
    │     "status": "complete",
    │     "started_at": "2026-02-04T10:30:00Z",
    │     "completed_at": "2026-02-04T10:35:00Z",
    │     "agents": ["mozart", "einstein", "staff-architect", "beethoven"],
    │     "total_cost_usd": 0.234,
    │     "parent_session": null
    │   }
    │
    ├── inputs/
    │   ├── einstein-input.json
    │   └── staff-architect-input.json
    │
    ├── outputs/
    │   ├── einstein-output.json
    │   ├── staff-architect-output.json
    │   └── beethoven-synthesis.json
    │
    └── logs/
        ├── spawn.log          # TUI spawn commands
        ├── einstein.stderr    # Agent stderr
        └── staff-architect.stderr
```

### 10.3 TUI Visualization

The session structure enables rich TUI visualization:

```
┌─ Sessions ──────────────────────────────────────────────────────────┐
│                                                                     │
│ braintrust-2026-02-04-abc123  [COMPLETE]  $0.234  5m 23s           │
│   ├── einstein           ✅  $0.089  2m 10s                        │
│   ├── staff-architect    ✅  $0.102  2m 34s                        │
│   └── beethoven          ✅  $0.043  39s                           │
│                                                                     │
│ review-2026-02-04-def456  [RUNNING]  $0.156  3m 12s                │
│   ├── backend-reviewer   ✅  $0.052  1m 02s                        │
│   ├── frontend-reviewer  🔄  $0.067  2m 10s (streaming)            │
│   └── standards-reviewer ⏳  queued                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 11. CLI Flag Strategy

### 11.1 Recommended Flags for `claude -p`

Based on CLI help output analysis:

| Flag | Purpose | Value |
|------|---------|-------|
| `-p` / `--print` | Non-interactive mode | Required |
| `--output-format` | Structured output | `json` (not stream-json for simplicity) |
| `--json-schema` | Validate output structure | Per-workflow schema |
| `--permission-mode` | Security | `delegate` (not `bypassPermissions`) |
| `--allowedTools` | Restrict tools | Per-agent allowlist |
| `--model` | Model selection | From agent config |
| `--max-budget-usd` | Cost control | Per-agent limit |
| `--session-id` | Session tracking | UUID from session manager |

### 11.2 Security: Permission Modes

Instead of `--dangerously-skip-permissions`:

| Mode | Behavior | Use Case |
|------|----------|----------|
| `default` | Prompts for permissions | Interactive |
| `delegate` | Delegates permission decisions | **Recommended for CLI spawning** |
| `acceptEdits` | Auto-accept edits only | Conservative |
| `bypassPermissions` | Skip all checks | Only for trusted sandboxes |

### 11.3 Tool Restriction per Agent Type

```typescript
const AGENT_TOOL_ALLOWLISTS: Record<string, string[]> = {
  'einstein': ['Read', 'Glob', 'Grep'],  // Analysis only
  'staff-architect-critical-review': ['Read', 'Glob', 'Grep'],
  'beethoven': ['Read', 'Write'],  // Can write synthesis doc
  'backend-reviewer': ['Read', 'Glob', 'Grep'],
  'go-pro': ['Read', 'Write', 'Edit', 'Bash', 'Glob', 'Grep'],
  'scaffolder': ['Read', 'Write', 'Glob'],
};
```

### 11.4 Example Spawn Command

```bash
claude -p \
  --model opus \
  --output-format json \
  --json-schema "$(cat schemas/braintrust/output.schema.json)" \
  --permission-mode delegate \
  --allowedTools "Read Glob Grep" \
  --max-budget-usd 0.50 \
  --session-id "einstein-abc123-def456" \
  < sessions/braintrust-abc123/inputs/einstein-input.json \
  > sessions/braintrust-abc123/outputs/einstein-output.json \
  2> sessions/braintrust-abc123/logs/einstein.stderr
```

---

## 12. Hook-Based Level Detection

### 12.1 Current Hook Chain

| Event | Hook | Purpose |
|-------|------|---------|
| SessionStart | gogent-load-context | Context injection |
| PreToolUse (Task) | gogent-validate | Block opus, validate subagent_type |
| PostToolUse | gogent-sharp-edge | Telemetry, routing reminders |
| SubagentStop | gogent-agent-endstate | Decision outcomes |
| SessionEnd | gogent-archive | Handoff generation |

### 12.2 New: Nesting Level Propagation

```
Session Start
    │
    └── gogent-load-context sets GOGENT_NESTING_LEVEL=0

Task() invocation at Level 0
    │
    ├── gogent-validate checks level (0 = allow)
    │
    └── Subagent spawns with GOGENT_NESTING_LEVEL=1

Task() invocation at Level 1
    │
    ├── gogent-validate checks level (1 = BLOCK)
    │
    └── Returns: "Use MCP spawn_agent instead"

MCP spawn_agent at Level 1
    │
    ├── Validates schema
    ├── Creates session directory
    └── Spawns claude -p with GOGENT_NESTING_LEVEL=2
```

### 12.3 Hook Modification

```go
// cmd/gogent-validate/main.go

func main() {
    // ... existing code ...

    // Get nesting level from environment
    nestingLevel := 0
    if levelStr := os.Getenv("GOGENT_NESTING_LEVEL"); levelStr != "" {
        nestingLevel, _ = strconv.Atoi(levelStr)
    }

    // Block Task() at nesting level 1+
    if event.ToolName == "Task" && nestingLevel > 0 {
        output := map[string]any{
            "decision": "block",
            "reason": fmt.Sprintf(
                "Task() blocked at nesting level %d. "+
                "Subagents cannot spawn sub-subagents via Task(). "+
                "Use MCP spawn_agent tool instead.",
                nestingLevel,
            ),
            "hookSpecificOutput": map[string]any{
                "hookEventName":            "PreToolUse",
                "permissionDecision":       "deny",
                "permissionDecisionReason": "nesting_level_exceeded",
            },
        }
        data, _ := json.MarshalIndent(output, "", "  ")
        fmt.Println(string(data))
        return
    }

    // ... existing validation for opus/allowlist ...
}
```

### 12.4 Environment Variable Propagation

When TUI spawns via `claude -p`:

```typescript
const proc = spawn('claude', cliArgs, {
  env: {
    ...process.env,
    GOGENT_NESTING_LEVEL: String(currentLevel + 1),
    GOGENT_PARENT_AGENT: parentAgentId,
    GOGENT_EPIC_ID: epicId,
    GOGENT_SESSION_DIR: sessionDir,
  },
});
```

---

## 13. Implementation Roadmap

### Phase 0: Verification (2 days)

**Gate**: Must pass before proceeding

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| V1 | Test MCP availability in subagents | MCP tool invocable from Level 1 |
| V2 | Test Task() failure in subagents | Confirm Task() unavailable at Level 1 |
| V3 | Test `claude -p` with JSON I/O | Structured input/output works |

### Phase 1: Foundation (4 days)

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| F1 | Create schema directory structure | `schemas/` with workflow schemas |
| F2 | Implement schema registry | Load, validate schemas in TypeScript |
| F3 | Implement session manager | Create, update, query sessions |
| F4 | Implement `spawn_agent` MCP tool | Schema validation, CLI spawn, output collection |
| F5 | Modify gogent-validate for nesting | Block Task() at Level 1+ |
| F6 | Register with shutdown handler | Clean up spawned processes |

### Phase 2: Integration (3 days)

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| I1 | Update TUI store for sessions | Session tracking in Zustand |
| I2 | Session visualization component | Display active/completed sessions |
| I3 | Update Mozart for MCP spawning | Uses spawn_agent instead of Task() |
| I4 | Update review-orchestrator | Uses spawn_agent for parallel reviewers |

### Phase 3: Testing & Polish (3 days)

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| T1 | Create mock Claude CLI | For unit tests without API calls |
| T2 | Unit tests for spawn_agent | Schema validation, error handling |
| T3 | Integration tests | Full Braintrust workflow |
| T4 | Documentation | Update CLAUDE.md, agent docs |

### Total: 12 days (with buffer)

---

## 14. Risk Matrix

### Critical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| MCP tools unavailable to subagents | Medium | Critical | Verify in Phase 0; fallback to flat coordination |
| `claude -p` JSON I/O unreliable | Low | High | Test extensively in Phase 0 |
| Orphan processes on TUI crash | Low | High | Shutdown handler + process registry |
| Schema validation overhead | Low | Medium | Cache schemas; async validation |

### Medium Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Cold start latency unacceptable | Medium | Medium | Parallel spawning; consider pooling later |
| Session directory disk usage | Medium | Low | Retention policy; archive old sessions |
| Hook complexity | Medium | Medium | Comprehensive testing |

### Rollback Plan

1. **Feature flag**: `GOGENT_MCP_SPAWN_ENABLED=false`
2. **Revert hook change**: Remove nesting level check
3. **Fallback**: Use flat coordination (router does all spawning)

---

## 15. Open Questions for Next Braintrust

### Architectural Questions

1. **Stream-JSON vs JSON**: Should we use `--output-format stream-json` for real-time progress, or `json` for simplicity? What are the parsing overhead implications?

2. **Session persistence**: How long to retain session directories? Archive strategy?

3. **Resumable sessions**: Should interrupted workflows be resumable from session state?

4. **Cross-workflow dependencies**: Can one workflow's output be input to another?

### Implementation Questions

5. **Schema versioning**: How to handle schema evolution?

6. **Partial output handling**: What if agent outputs partial JSON (network drop)?

7. **Concurrent session limits**: How many sessions can run simultaneously?

8. **Cost attribution**: Per-session cost tracking vs per-epic aggregation?

### UX Questions

9. **TUI session management**: How to display many sessions? Filtering? Search?

10. **Error surfacing**: How to show agent errors in TUI without overwhelming user?

---

## 16. Appendices

### Appendix A: Current TUI Infrastructure Summary

| File | Purpose | Relevant Lines |
|------|---------|----------------|
| `packages/tui/src/mcp/server.ts` | MCP tool registration | 16-20 |
| `packages/tui/src/store/slices/agents.ts` | Agent tracking | Full file |
| `packages/tui/src/store/types.ts` | Type definitions | 38-51 (Agent) |
| `packages/tui/src/lifecycle/shutdown.ts` | Process cleanup | 126+ |

### Appendix B: CLI Flags Reference

From `claude --help`:

| Flag | Type | Notes |
|------|------|-------|
| `--output-format` | `text\|json\|stream-json` | `json` recommended for structured |
| `--input-format` | `text\|stream-json` | `text` sufficient for JSON stdin |
| `--json-schema` | JSON string | Validates output structure |
| `--permission-mode` | enum | `delegate` recommended |
| `--allowedTools` | string list | Space or comma separated |
| `--max-budget-usd` | number | Cost control |
| `--session-id` | UUID | Session tracking |

### Appendix C: Example Schema Files

See `schemas/braintrust/input.schema.json` and `schemas/braintrust/output.schema.json` in Section 9.

### Appendix D: Session Manager Interface

```typescript
interface SessionManager {
  // Lifecycle
  createSession(workflow: string, epicId: string): Promise<Session>;
  completeSession(sessionId: string, status: 'success' | 'error'): Promise<void>;

  // I/O
  writeInput(sessionId: string, agent: string, input: unknown): Promise<string>;
  writeOutput(sessionId: string, agent: string, output: unknown): Promise<void>;
  readOutput(sessionId: string, agent: string): Promise<unknown>;

  // Queries
  getSession(sessionId: string): Promise<Session>;
  listSessions(filter?: SessionFilter): Promise<Session[]>;
  getSessionCost(sessionId: string): Promise<number>;

  // Cleanup
  archiveSession(sessionId: string): Promise<void>;
  deleteOldSessions(olderThan: Date): Promise<number>;
}

interface Session {
  id: string;
  workflow: string;
  epicId: string;
  status: 'running' | 'success' | 'error' | 'archived';
  startedAt: Date;
  completedAt?: Date;
  agents: AgentRecord[];
  totalCostUsd: number;
  inputDir: string;
  outputDir: string;
}
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-04 | Beethoven | Initial synthesis from Einstein + Staff-Architect |

---

**END OF DOCUMENT**

This document serves as the complete reference for the next Braintrust iteration on MCP-based agent spawning architecture.
