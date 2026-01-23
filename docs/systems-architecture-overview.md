# GOgent-Fortress Systems Architecture

> **Schema Versions:** routing-schema v2.2.0 | handoff v1.2
> **Last Updated:** 2026-01-23
> **Status:** Implemented through Week 3 (session_archive suite)

---

## Overview

GOgent-Fortress is a Go-based hook orchestration framework for Claude Code. It enforces tiered routing policies, tracks debugging loops, captures user intents, and maintains session continuity through structured handoff documents.

The system intercepts Claude Code hook events (PreToolUse, PostToolUse, SessionStart, SessionEnd) and applies validation, failure tracking, and archival logic defined in `routing-schema.json`.

---

## 1. Hook Event Flow

The following diagram shows the complete lifecycle of a Claude Code session from the perspective of GOgent hooks.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant SS as SessionStart
    participant PT as PreToolUse
    participant Tool as Tool Execution
    participant PO as PostToolUse
    participant SE as SessionEnd

    CC->>SS: Session begins
    Note over SS: gogent-load-context<br/>(Week 4 - pending)
    SS-->>CC: Context injection

    loop Tool Usage
        CC->>PT: Task() invocation
        Note over PT: gogent-validate

        alt Validation Fails
            PT-->>CC: Block + reason
        else Validation Passes
            PT-->>CC: Allow
            CC->>Tool: Execute tool
            Tool-->>CC: Result
            CC->>PO: Tool completed
            Note over PO: gogent-sharp-edge

            alt Failure Detected
                PO->>PO: LogFailure()
                alt 3+ Consecutive Failures
                    PO-->>CC: Sharp edge captured
                end
            end
        end
    end

    CC->>SE: Session ends
    Note over SE: gogent-archive
    SE->>SE: Collect metrics
    SE->>SE: Load artifacts
    SE->>SE: Generate handoff
    SE-->>CC: Archive confirmation
```

### Hook Entry Points

| Hook Event | CLI Binary | When Fired |
|------------|------------|------------|
| SessionStart | `gogent-load-context` | Session startup/resume (pending) |
| PreToolUse | `gogent-validate` | Before any tool executes |
| PostToolUse | `gogent-sharp-edge` | After Bash/Edit/Write tools |
| SessionEnd | `gogent-archive` | Session termination |

---

## 2. Package Dependencies

```mermaid
graph TD
    subgraph "CLI Layer (cmd/)"
        validate[gogent-validate]
        archive[gogent-archive]
        sharpedge[gogent-sharp-edge]
        intent[gogent-capture-intent]
        aggregate[gogent-aggregate]
    end

    subgraph "Core Packages (pkg/)"
        routing[pkg/routing]
        session[pkg/session]
        memory[pkg/memory]
        telemetry[pkg/telemetry]
        config[pkg/config]
    end

    subgraph "External"
        schema[(routing-schema.json)]
        jsonl[(JSONL Files)]
    end

    validate --> routing
    archive --> session
    archive --> config
    sharpedge --> memory
    sharpedge --> routing
    intent --> session
    aggregate --> telemetry

    session --> config
    memory --> routing
    routing --> schema
    session --> jsonl
    memory --> jsonl
```

### Package Responsibilities

| Package | Primary Responsibility | Key Types |
|---------|------------------------|-----------|
| `pkg/routing` | Schema loading, Task validation, violation logging | `Schema`, `ValidationOrchestrator`, `Violation` |
| `pkg/session` | Handoffs, events, metrics, intents, queries | `Handoff`, `SessionMetrics`, `UserIntent`, `Query` |
| `pkg/memory` | Failure tracking, debugging loop detection | `FailureInfo`, `LogFailure()`, `GetFailureCount()` |
| `pkg/telemetry` | Invocation tracking, cost calculation | `AgentInvocation`, `TierPricing`, `SessionCostSummary` |
| `pkg/config` | Path resolution, tier configuration | `GetGOgentDir()`, `GetViolationsLogPath()` |

---

## 3. Data Persistence Layer

All persistence uses JSONL (JSON Lines) format for append-only writes and streaming reads.

```mermaid
flowchart TB
    subgraph "Session Scope"
        violations[/violations.jsonl/]
        counter[/tool-counter.log/]
    end

    subgraph "Project Scope (.claude/memory/)"
        handoffs[/handoffs.jsonl/]
        intents[/user-intents.jsonl/]
        decisions[/decisions.jsonl/]
        prefs[/preferences.jsonl/]
        perf[/performance.jsonl/]
        pending[/pending-learnings.jsonl/]
    end

    subgraph "Global Scope (~/.gogent/)"
        failures[/failure-tracker.jsonl/]
    end

    subgraph "Archive (session-archive/)"
        archived[/learnings-{ts}.jsonl/]
        sessions[/session-{id}.jsonl/]
    end

    validate([gogent-validate]) --> violations
    sharpedge([gogent-sharp-edge]) --> failures
    sharpedge --> pending
    archive([gogent-archive]) --> handoffs
    archive --> archived
    intent([gogent-capture-intent]) --> intents
```

### File Reference

| File | Scope | Written By | Schema |
|------|-------|------------|--------|
| `handoffs.jsonl` | Project | gogent-archive | Handoff v1.2 |
| `user-intents.jsonl` | Project | gogent-capture-intent | UserIntent |
| `decisions.jsonl` | Project | gogent-archive | Decision |
| `preferences.jsonl` | Project | gogent-archive | PreferenceOverride |
| `performance.jsonl` | Project | gogent-archive | PerformanceMetric |
| `pending-learnings.jsonl` | Project | gogent-sharp-edge | SharpEdge |
| `failure-tracker.jsonl` | Global | gogent-sharp-edge | FailureInfo |
| `routing-violations.jsonl` | Temp | gogent-validate | Violation |

---

## 4. CLI Entry Points

```mermaid
graph LR
    subgraph "Hook Binaries"
        V[gogent-validate<br/>PreToolUse]
        A[gogent-archive<br/>SessionEnd]
        S[gogent-sharp-edge<br/>PostToolUse]
        I[gogent-capture-intent<br/>Manual]
        G[gogent-aggregate<br/>Analysis]
    end

    subgraph "Input"
        stdin((STDIN<br/>JSON))
        env((ENV vars))
    end

    subgraph "Output"
        stdout((STDOUT<br/>Hook JSON))
        files((JSONL Files))
    end

    stdin --> V
    stdin --> A
    stdin --> S
    env --> V
    env --> A

    V --> stdout
    V --> files
    A --> stdout
    A --> files
    S --> files
    I --> files
    G --> stdout
```

### CLI Reference

| Binary | Hook Event | Input | Output | Lines |
|--------|------------|-------|--------|-------|
| `gogent-validate` | PreToolUse | ToolEvent JSON | ValidationResult JSON | ~142 |
| `gogent-archive` | SessionEnd | SessionEvent JSON | Confirmation JSON | ~1111 |
| `gogent-sharp-edge` | PostToolUse | ToolEvent JSON | (none) | ~200 |
| `gogent-capture-intent` | Manual | UserIntent JSON | (none) | ~150 |
| `gogent-aggregate` | Manual | (flags) | Summary JSON | ~100 |

### gogent-archive Subcommands

The archive CLI includes query subcommands for inspecting session history:

| Subcommand | Purpose |
|------------|---------|
| `list` | List sessions with filters (--since, --has-sharp-edges) |
| `show <id>` | Display specific session handoff |
| `stats` | Aggregate statistics across sessions |
| `sharp-edges` | Query sharp edges with filters |
| `user-intents` | Query user intents with filters |
| `decisions` | Query architectural decisions |
| `preferences` | Query preference overrides |
| `performance` | Query performance metrics |
| `weekly` | Generate weekly intent summary |

---

## 5. Validation Pipeline

The `gogent-validate` binary orchestrates multiple validation checks for Task tool invocations.

```mermaid
flowchart TD
    input[Task Invocation] --> parse[Parse ToolEvent]
    parse --> check1{Tool == Task?}

    check1 -->|No| pass1[Pass through]
    check1 -->|Yes| load[Load Schema]

    load --> v1[Check 1: Einstein/Opus Blocking]
    v1 -->|Blocked| block1[BLOCK: Use /einstein]
    v1 -->|OK| v2[Check 2: Model Mismatch]

    v2 -->|Mismatch| warn[WARN: Continue with warning]
    v2 -->|OK| v3[Check 3: Delegation Ceiling]

    v3 -->|Exceeded| block2[BLOCK: Ceiling violation]
    v3 -->|OK| v4[Check 4: Subagent Type]

    v4 -->|Invalid| block3[BLOCK: Wrong subagent_type]
    v4 -->|OK| allow[ALLOW]

    block1 --> log[Log Violation]
    block2 --> log
    block3 --> log
    warn --> allow
```

### Validation Checks

| Check | Blocking | Logged | Purpose |
|-------|----------|--------|---------|
| Einstein/Opus | Yes | Yes | Prevent Task(opus) - use /einstein instead |
| Model Mismatch | No | No | Warn if requested model differs from agents-index |
| Delegation Ceiling | Yes | Yes | Enforce max tier from calculate-complexity |
| Subagent Type | Yes | Yes | Ensure agent-subagent_type pairing matches schema |

---

## 6. Handoff Schema

The handoff document captures session state for cross-session continuity.

```mermaid
classDiagram
    class Handoff {
        +string schema_version
        +int64 timestamp
        +string session_id
        +SessionContext context
        +HandoffArtifacts artifacts
        +Action[] actions
    }

    class SessionContext {
        +string project_dir
        +SessionMetrics metrics
        +string active_ticket
        +string phase
        +GitInfo git_info
    }

    class HandoffArtifacts {
        +SharpEdge[] sharp_edges
        +RoutingViolation[] routing_violations
        +ErrorPattern[] error_patterns
        +UserIntent[] user_intents
        +Decision[] decisions
        +PreferenceOverride[] preference_overrides
        +PerformanceMetric[] performance_metrics
    }

    class SessionMetrics {
        +int tool_calls
        +int errors_logged
        +int routing_violations
        +string session_id
    }

    Handoff --> SessionContext
    Handoff --> HandoffArtifacts
    SessionContext --> SessionMetrics
```

---

## 7. How to Extend

### Adding a New CLI

1. Create `cmd/gogent-<name>/main.go`
2. Implement STDIN parsing with timeout (see `pkg/routing/stdin.go`)
3. Output hook-compatible JSON to STDOUT
4. Add to `Makefile` build targets
5. Document in this file under "CLI Entry Points"

### Adding a New Package

1. Create `pkg/<name>/` with `doc.go`
2. Define types in dedicated files (one primary type per file)
3. Add `_test.go` files (target 80%+ coverage)
4. Update dependency diagram above
5. Document in "Package Responsibilities" table

### Adding a New Artifact Type

1. Define struct in `pkg/session/` (e.g., `handoff_artifacts.go`)
2. Add to `HandoffArtifacts` struct with `omitempty` tag
3. Update `LoadArtifacts()` in `pkg/session/handoff.go`
4. Add query method to `pkg/session/query.go`
5. Add CLI subcommand to `gogent-archive` if user-queryable
6. Update handoff schema version if breaking change

### Adding a New Validation Check

1. Create validator in `pkg/routing/` (e.g., `new_validation.go`)
2. Add check to `ValidationOrchestrator.ValidateTask()`
3. Define violation type constant
4. Add to violation logging
5. Update "Validation Checks" table above

---

## 8. Schema Version History

### routing-schema.json

| Version | Changes |
|---------|---------|
| 2.2.0 | Current - Added agent_subagent_mapping, blocked_patterns |
| 2.1.0 | Added delegation_ceiling, tier_levels |
| 2.0.0 | Complete restructure for tiered architecture |

### handoff (pkg/session)

| Version | Changes |
|---------|---------|
| 1.2 | Added SharpEdge extended fields (type, tool, code_snippet, status) |
| 1.1 | Added decisions, preference_overrides, performance_metrics |
| 1.0 | Initial schema with sharp_edges, routing_violations, error_patterns |

---

## 9. Quick Reference

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `GOGENT_PROJECT_DIR` | Project root | `$PWD` |
| `GOGENT_ROUTING_SCHEMA` | Schema path override | `~/.claude/routing-schema.json` |
| `GOGENT_STORAGE_PATH` | Failure tracker path | `~/.gogent/failure-tracker.jsonl` |
| `GOGENT_MAX_FAILURES` | Debugging loop threshold | 3 |
| `GOGENT_FAILURE_WINDOW` | Failure window (seconds) | 300 |

### Key File Paths

```
Project/
├── .claude/
│   ├── memory/
│   │   ├── handoffs.jsonl        # Session history
│   │   ├── user-intents.jsonl    # User preferences
│   │   ├── decisions.jsonl       # Architectural decisions
│   │   ├── preferences.jsonl     # Preference overrides
│   │   ├── performance.jsonl     # Performance metrics
│   │   ├── pending-learnings.jsonl  # Unreviewed sharp edges
│   │   └── last-handoff.md       # Human-readable handoff
│   ├── tmp/
│   │   └── einstein-gap-*.md     # Escalation documents
│   └── session-archive/          # Archived session data
│
~/.gogent/
└── failure-tracker.jsonl         # Cross-session failure tracking

/tmp/
├── claude-routing-violations.jsonl  # Current session violations
└── claude-tool-counter-*.log        # Tool call counters
```

---

## 10. Related Documentation

| Document | Purpose |
|----------|---------|
| `CLAUDE.md` | Project-level Claude configuration |
| `~/.claude/CLAUDE.md` | Global Claude configuration with routing gates |
| `~/.claude/routing-schema.json` | Source of truth for tier definitions |
| `dev/will/migration_plan/tickets/` | Implementation tickets |

---

*This document is designed for incremental updates. When adding new components, update the relevant section and diagram rather than rewriting prose.*
