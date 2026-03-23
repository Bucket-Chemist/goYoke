# GOgent-Fortress Systems Architecture v1.5

> **Schema Versions:** routing-schema v2.5.0 | handoff v1.3 | ML telemetry v1.1 (review)
> **Last Updated:** 2026-02-01
> **Status:** Production Ready - Complete Implementation (Hooks + TUI + Review Telemetry)
> **TUI v1 (Legacy):** GOgent-109 through GOgent-121 (13 tickets) — superseded by TUI Migration
> **TUI v2 (Migration):** TUI-001 through TUI-042 (42 tickets, 9 phases). ✅ ALL COMPLETE (42/42, 100%). Feature parity verified.
> **Review Telemetry:** GOgent-122 through GOgent-139 (18 tasks)

---

## Overview

GOgent-Fortress is a Go-based hook orchestration framework for Claude Code. It enforces tiered routing policies, tracks debugging loops, captures user intents, manages ML telemetry, and maintains session continuity through structured handoff documents.

The system intercepts Claude Code hook events (SessionStart, PreToolUse, PostToolUse, SubagentStop, SessionEnd) and applies validation, failure tracking, telemetry capture, and archival logic defined in `routing-schema.json`.

---

## 1. System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                           GOgent-Fortress Architecture                                               │
│                                              (Production v1.0)                                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                      │
│  ┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                    CLAUDE CODE CLI PROCESS                                                    │   │
│  │                                                                                                               │   │
│  │    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐               │   │
│  │    │ SessionStart│───▶│ PreToolUse  │───▶│    Tool     │───▶│ PostToolUse │───▶│ SubagentStop│               │   │
│  │    │   Event     │    │   Event     │    │  Execution  │    │   Event     │    │   Event     │               │   │
│  │    └──────┬──────┘    └──────┬──────┘    └─────────────┘    └──────┬──────┘    └──────┬──────┘               │   │
│  │           │                  │                                     │                  │                       │   │
│  └───────────┼──────────────────┼─────────────────────────────────────┼──────────────────┼───────────────────────┘   │
│              │                  │                                     │                  │                           │
│              │ STDIN/JSON       │ STDIN/JSON                          │ STDIN/JSON       │ STDIN/JSON                │
│              ▼                  ▼                                     ▼                  ▼                           │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                           HOOK BINARIES (Go)                                                 │    │
│  │                                                                                                              │    │
│  │  ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐             │    │
│  │  │  gogent-load-      │  │   gogent-validate  │  │  gogent-sharp-edge │  │ gogent-agent-      │             │    │
│  │  │  context           │  │                    │  │                    │  │ endstate           │             │    │
│  │  │                    │  │                    │  │                    │  │                    │             │    │
│  │  │ • Language detect  │  │ • Schema validation│  │ • Tool counting    │  │ • Decision outcomes│             │    │
│  │  │ • Convention load  │  │ • Task() checks    │  │ • Failure tracking │  │ • Collab tracking  │             │    │
│  │  │ • Handoff restore  │  │ • Tier enforcement │  │ • ML telemetry     │  │ • ML updates       │             │    │
│  │  │ • Git context      │  │ • Ceiling check    │  │ • Sharp-edge cap   │  │ • Session metrics  │             │    │
│  │  └─────────┬──────────┘  └─────────┬──────────┘  └─────────┬──────────┘  └─────────┬──────────┘             │    │
│  │            │                       │                       │                       │                         │    │
│  └────────────┼───────────────────────┼───────────────────────┼───────────────────────┼─────────────────────────┘    │
│               │                       │                       │                       │                              │
│               ▼                       ▼                       ▼                       ▼                              │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                         GO PACKAGES (pkg/)                                                   │    │
│  │                                                                                                              │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │    │
│  │  │ pkg/routing  │  │ pkg/session  │  │  pkg/memory  │  │pkg/telemetry │  │  pkg/config  │  │ pkg/workflow │ │    │
│  │  │              │  │              │  │              │  │              │  │              │  │              │ │    │
│  │  │ •Schema      │  │ •Handoff     │  │ •Failure     │  │ •Invocations │  │ •Paths       │  │ •Responses   │ │    │
│  │  │ •Validation  │  │ •Events      │  │  Tracking    │  │ •Cost        │  │ •Tier        │  │ •Logging     │ │    │
│  │  │ •Violations  │  │ •Metrics     │  │ •Pattern     │  │ •Escalations │  │ •Environment │  │ •Integration │ │    │
│  │  │ •Orchestrator│  │ •Artifacts   │  │  Matching    │  │ •Scout       │  │              │  │              │ │    │
│  │  │ •Agents      │  │ •UserIntent  │  │ •Responses   │  │              │  │              │  │              │ │    │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │    │
│  │         │                 │                 │                 │                 │                 │          │    │
│  └─────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┼──────────┘    │
│            │                 │                 │                 │                 │                 │               │
│            └────────────────┬┴────────────────┬┴────────────────┬┴────────────────┬┴─────────────────┘               │
│                             │                 │                 │                                                    │
│                             ▼                 ▼                 ▼                                                    │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                        DATA PERSISTENCE LAYER                                                │    │
│  │                                                                                                              │    │
│  │  ┌────────────────────────────────┐  ┌────────────────────────────────┐  ┌────────────────────────────────┐ │    │
│  │  │     SESSION SCOPE (/tmp/)      │  │   PROJECT SCOPE (.claude/)    │  │   ML SCOPE (XDG_DATA_HOME/)    │ │    │
│  │  │                                │  │                                │  │                                │ │    │
│  │  │ • routing-violations.jsonl     │  │ • memory/handoffs.jsonl       │  │ • routing-decisions.jsonl     │ │    │
│  │  │ • tool-counter-{id}.log        │  │ • memory/user-intents.jsonl   │  │ • routing-decision-updates    │ │    │
│  │  │ • error-patterns.jsonl         │  │ • memory/decisions.jsonl      │  │   .jsonl                      │ │    │
│  │  │                                │  │ • memory/pending-learnings    │  │ • agent-collaborations.jsonl  │ │    │
│  │  │                                │  │   .jsonl                      │  │ • agent-collaboration-updates │ │    │
│  │  │                                │  │ • memory/last-handoff.md      │  │   .jsonl                      │ │    │
│  │  │                                │  │ • session-archive/            │  │                                │ │    │
│  │  └────────────────────────────────┘  └────────────────────────────────┘  └────────────────────────────────┘ │    │
│  │                                                                                                              │    │
│  │  ┌────────────────────────────────┐                                                                          │    │
│  │  │     GLOBAL SCOPE (~/.gogent/)  │  ┌────────────────────────────────────────────────────────────────────┐ │    │
│  │  │                                │  │                  CONFIGURATION (~/.claude/)                        │ │    │
│  │  │ • failure-tracker.jsonl        │  │                                                                    │ │    │
│  │  │ • agent-invocations.jsonl      │  │  • routing-schema.json (v2.2.0)    • agents/*.yaml                │ │    │
│  │  │ • escalations.jsonl            │  │  • conventions/*.md                 • skills/*/SKILL.md           │ │    │
│  │  │ • scout-recommendations.jsonl  │  │  • CLAUDE.md                        • rules/*.md                   │ │    │
│  │  │                                │  │                                                                    │ │    │
│  │  └────────────────────────────────┘  └────────────────────────────────────────────────────────────────────┘ │    │
│  │                                                                                                              │    │
│  └──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                                                      │
│                                                                                                                      │
├──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                                      │
│  ┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                                                               │   │
│  │                                    🖥️  TUI SYSTEM (GOgent-109 to GOgent-121)                                  │   │
│  │                                                                                                               │   │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │   │
│  │  │                                      Implemented Components                                              │ │   │
│  │  ├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤ │   │
│  │  │                                                                                                          │ │   │
│  │  │   📦 internal/cli                    📦 internal/tui/agents          📦 internal/tui/claude              │ │   │
│  │  │   ─────────────────                  ──────────────────────          ─────────────────────               │ │   │
│  │  │   • ClaudeProcess subprocess         • AgentTree data model          • PanelModel (conversation)        │ │   │
│  │  │   • NDJSON reader/writer             • TreeModel (view component)    • Streaming output display         │ │   │
│  │  │   • Event type parsing               • DetailModel (sidebar)         • Hook event sidebar               │ │   │
│  │  │   • Auto-restart on panic            • Status icons (⏳⟳✓✗)          • User input handling              │ │   │
│  │  │   • SessionManager                   • Expand/collapse navigation    • Cost tracking                    │ │   │
│  │  │                                                                                                          │ │   │
│  │  │   📦 internal/tui/layout             📦 internal/tui/session         📦 pkg/telemetry (file watchers)   │ │   │
│  │  │   ──────────────────────             ──────────────────────          ─────────────────────────────       │ │   │
│  │  │   • 70/30 split layout               • PickerModel (modal)           • TelemetryWatcher (fsnotify)      │ │   │
│  │  │   • Focus management (Tab)           • Session list by recency       • Agent lifecycle events           │ │   │
│  │  │   • BannerModel (nav tabs)           • Resume/delete operations      • Real-time JSONL streaming        │ │   │
│  │  │   • Number keys (1-4)                • Age formatting                • Async event dispatch             │ │   │
│  │  │   • Session info display             • Keyboard navigation           • File watching infra              │ │   │
│  │  │                                                                                                          │ │   │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │   │
│  │                                                                                                               │   │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐ │   │
│  │  │                                           TUI Architecture                                               │ │   │
│  │  ├─────────────────────────────────────────────────────────────────────────────────────────────────────────┤ │   │
│  │  │                                                                                                          │ │   │
│  │  │   +--------------------------------------------------------------------------+                           │ │   │
│  │  │   | [1] Claude  [2] Agents  [3] Stats  [4] Query   Session: abc | Cost: $0.34|  ← BannerModel           │ │   │
│  │  │   +----------------------------------------------+---------------------------+                           │ │   │
│  │  │   |                                              |                           |                           │ │   │
│  │  │   |  Claude Conversation Panel (70%)             |  Agent Tree (30% top)     |  ← TreeModel             │ │   │
│  │  │   |  ─────────────────────────────               |  > terminal               |                           │ │   │
│  │  │   |  You: Explain this function                  |    +-- orchestrator [✓]   |                           │ │   │
│  │  │   |                                              |    +-- go-tui [⟳]         |                           │ │   │
│  │  │   |  Claude: This function implements...         +---------------------------+                           │ │   │
│  │  │   |  [streaming...]                              |  Agent Detail (30% bot)   |  ← DetailModel           │ │   │
│  │  │   |                                              |  Selected: go-tui         |                           │ │   │
│  │  │   |  [Hook: validate ✓] [Tool: Read ✓]           |  Tier: sonnet             |                           │ │   │
│  │  │   +----------------------------------------------+  Duration: 4.2s...        |                           │ │   │
│  │  │   | > Type your message here...          [Enter] |  [Enter] Expand           |                           │ │   │
│  │  │   +----------------------------------------------+---------------------------+                           │ │   │
│  │  │                                                                                                          │ │   │
│  │  └─────────────────────────────────────────────────────────────────────────────────────────────────────────┘ │   │
│  │                                                                                                               │   │
│  └──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Hook Event Flow

The following sequence diagram shows the complete lifecycle of a Claude Code session from the perspective of GOgent hooks.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant SS as SessionStart
    participant PT as PreToolUse
    participant Tool as Tool Execution
    participant PO as PostToolUse
    participant SA as SubagentStop
    participant SE as SessionEnd

    CC->>SS: Session begins
    Note over SS: gogent-load-context
    SS->>SS: DetectLanguage()
    SS->>SS: LoadConventions()
    SS->>SS: LoadHandoff()
    SS->>SS: CollectGitInfo()
    SS-->>CC: Context injection (additionalContext)

    loop Tool Usage
        CC->>PT: Task() invocation
        Note over PT: gogent-validate
        PT->>PT: LoadSchema()
        PT->>PT: ValidateTask()
        PT->>PT: CheckDelegationCeiling()
        PT->>PT: CheckSubagentType()

        alt Validation Fails
            PT->>PT: LogViolation()
            PT-->>CC: Block + reason
        else Validation Passes
            PT-->>CC: Allow
            CC->>Tool: Execute tool
            Tool-->>CC: Result

            CC->>PO: Tool completed
            Note over PO: gogent-sharp-edge
            PO->>PO: IncrementToolCounter()
            PO->>PO: LogRoutingDecision()
            PO->>PO: DetectFailure()

            alt Every 10 Tools
                PO->>PO: GenerateRoutingReminder()
            end

            alt Every 20+ Tools
                PO->>PO: ArchivePendingLearnings()
            end

            alt 3+ Consecutive Failures
                PO->>PO: CaptureSharpEdge()
                PO-->>CC: Block + guidance
            else Normal
                PO-->>CC: Pass-through
            end
        end
    end

    opt Agent Delegation
        CC->>SA: Subagent completes
        Note over SA: gogent-agent-endstate
        SA->>SA: UpdateDecisionOutcome()
        SA->>SA: LogAgentCollaboration()
        SA->>SA: UpdateCollaborationOutcome()
        SA-->>CC: Tier-specific follow-up
    end

    CC->>SE: Session ends
    Note over SE: gogent-archive
    SE->>SE: CollectMetrics()
    SE->>SE: LoadArtifacts()
    SE->>SE: GenerateHandoff()
    SE->>SE: GenerateMarkdown()
    SE-->>CC: Archive confirmation
```

### 2.1 Hook Entry Points

| Hook Event | CLI Binary | Primary Responsibilities | Trigger |
|------------|------------|--------------------------|---------|
| SessionStart | `gogent-load-context` | Language detection, convention loading, handoff restoration, git context | Session initialization |
| PreToolUse | `gogent-validate` | Schema validation, Task() checks, tier enforcement, ceiling verification | Every tool call |
| PostToolUse | `gogent-sharp-edge` | Tool counting, failure tracking, ML telemetry, sharp-edge detection, routing reminders | After tool execution |
| SubagentStop | `gogent-agent-endstate` | Decision outcomes, collaboration tracking, ML updates, session metrics | Agent completion |
| SessionEnd | `gogent-archive` | Metrics collection, artifact loading, handoff generation, markdown export | Session termination |

### 2.2 PostToolUse Handler: Merged Responsibilities

The `gogent-sharp-edge` PostToolUse handler consolidates five integrated responsibilities:

#### 1. Tool Counter Management
- **Trigger**: Every tool execution (Bash, Edit, Write, Read, Glob, Grep)
- **Action**: Increments persistent counter in `/tmp/claude-tool-counter-{session_id}.log`
- **Purpose**: Track tool usage frequency for routing reminders and auto-flush

#### 2. Routing Compliance Reminders
- **Trigger**: Every 10 tools (configurable via `GOGENT_REMINDER_THRESHOLD`)
- **Action**: Generates structured reminder injected into `additionalContext`
- **Content**: Reminds agent to verify routing compliance with `routing-schema.json`

#### 3. Pending Learnings Auto-Flush
- **Trigger**: Every 20+ tools (configurable via `GOGENT_FLUSH_THRESHOLD`)
- **Action**: Archives `pending-learnings.jsonl` to `session-archive/`
- **Purpose**: Prevent unbounded growth of pending learnings file

#### 4. ML Tool Event Logging (GOgent-087d)
- **Trigger**: Every tool execution
- **Action**: Logs RoutingDecision to `routing-decisions.jsonl`
- **Fields**: Tool name, duration, input/output tokens, sequence index, model, tier
- **Thread Safety**: Append-only pattern with dual-file reconciliation

#### 5. Sharp-Edge Detection
- **Trigger**: 3+ consecutive failures on same file/error type
- **Action**: Captures learning to `pending-learnings.jsonl`, blocks execution
- **Response**: Returns blocking guidance with pattern matching from `sharp-edges.yaml`

---

## 3. Package Architecture

```mermaid
graph TD
    subgraph "CLI Layer (cmd/)"
        loadcontext[gogent-load-context<br/>SessionStart]
        validate[gogent-validate<br/>PreToolUse]
        sharpedge[gogent-sharp-edge<br/>PostToolUse]
        endstate[gogent-agent-endstate<br/>SubagentStop]
        archive[gogent-archive<br/>SessionEnd]
        intent[gogent-capture-intent<br/>Manual]
        aggregate[gogent-aggregate<br/>Analysis]
        mlexport[gogent-ml-export<br/>ML Pipeline]
        orchguard[gogent-orchestrator-guard<br/>Enforcement]
        doctheater[gogent-doc-theater<br/>Detection]
    end

    subgraph "Core Packages (pkg/)"
        routing[pkg/routing<br/>Schema · Validation · Violations]
        session[pkg/session<br/>Handoff · Events · Metrics · Intents]
        memory[pkg/memory<br/>Failure Tracking · Patterns · Responses]
        telemetry[pkg/telemetry<br/>Invocations · Cost · Escalations · Scout]
        config[pkg/config<br/>Paths · Tier · Environment]
        workflow[pkg/workflow<br/>Responses · Logging · Integration]
        enforcement[pkg/enforcement<br/>Blocking · Patterns · DocTheater]
    end

    subgraph "External Dependencies"
        schema[(routing-schema.json)]
        jsonl[(JSONL Files)]
        conventions[(conventions/*.md)]
    end

    %% CLI to Package mappings
    loadcontext --> session
    loadcontext --> routing
    loadcontext --> config

    validate --> routing
    validate --> config

    sharpedge --> memory
    sharpedge --> routing
    sharpedge --> telemetry
    sharpedge --> session

    endstate --> telemetry
    endstate --> session
    endstate --> config

    archive --> session
    archive --> config
    archive --> telemetry

    intent --> session

    aggregate --> telemetry
    aggregate --> session

    mlexport --> telemetry
    mlexport --> config

    orchguard --> enforcement
    orchguard --> routing

    doctheater --> enforcement
    doctheater --> routing

    %% Package to Data mappings
    routing --> schema
    session --> jsonl
    memory --> jsonl
    telemetry --> jsonl
    config --> conventions
```

### 3.1 Package Responsibilities

| Package | Primary Responsibility | Key Types |
|---------|------------------------|-----------|
| `pkg/routing` | Schema loading, Task validation, violation logging, tier management | `Schema`, `TierConfig`, `Violation`, `AgentSubagentMapping` |
| `pkg/session` | Handoffs, events, metrics, intents, artifact loading, markdown export | `Handoff`, `SessionMetrics`, `UserIntent`, `SharpEdge`, `Action` |
| `pkg/memory` | Failure tracking, debugging loop detection, pattern matching | `FailureInfo`, `LogFailure()`, `GetFailureCount()`, `PatternMatch` |
| `pkg/telemetry` | Invocation tracking, cost calculation, escalations, scout recommendations, **review findings** | `AgentInvocation`, `TierPricing`, `EscalationEvent`, `ScoutRecommendation`, **`ReviewFinding`, `SharpEdgeHit`** |
| `pkg/config` | Path resolution, tier configuration, environment detection | `GetGOgentDir()`, `GetViolationsLogPath()`, `GetCurrentTier()` |
| `pkg/workflow` | Response formatting, logging utilities, integration helpers | `HookResponse`, `BlockResponse`, `PassthroughResponse` |
| `pkg/enforcement` | Blocking responses, pattern detection, documentation theater checks | `BlockingResponse`, `PatternDetector`, `DocTheater` |

---

## 4. ML Telemetry System

The ML telemetry system provides comprehensive observability for routing optimization and agent collaboration learning. All writes use append-only JSONL pattern for thread-safe concurrent execution.

### 4.1 Architecture Overview

```mermaid
graph TD
    subgraph "Data Capture Hooks"
        PostToolUse[PostToolUse<br/>gogent-sharp-edge]
        SubagentStop[SubagentStop<br/>gogent-agent-endstate]
    end

    subgraph "Initial Records (Append-Only)"
        RoutingDec["routing-decisions.jsonl<br/>(RoutingDecision)"]
        AgentCollab["agent-collaborations.jsonl<br/>(AgentCollaboration)"]
    end

    subgraph "Outcome Records (Append-Only)"
        RoutingUpd["routing-decision-updates.jsonl<br/>(DecisionOutcomeUpdate)"]
        CollabUpd["agent-collaboration-updates.jsonl<br/>(CollaborationUpdate)"]
    end

    subgraph "ML Pipeline"
        MLExport["gogent-ml-export<br/>Reconciliation + Export"]
        Training["ML Training Pipeline"]
    end

    PostToolUse -->|Logs| RoutingDec
    PostToolUse -->|Captures| AgentCollab

    SubagentStop -->|Updates| RoutingUpd
    SubagentStop -->|Updates| CollabUpd

    RoutingDec -->|Read-time join| MLExport
    RoutingUpd -->|on DecisionID| MLExport
    AgentCollab -->|Read-time join| MLExport
    CollabUpd -->|on CollaborationID| MLExport

    MLExport -->|Training data| Training

    style RoutingDec fill:#e1f5ff
    style AgentCollab fill:#e1f5ff
    style RoutingUpd fill:#fff3e0
    style CollabUpd fill:#fff3e0
    style MLExport fill:#e8f5e9
```

### 4.2 Append-Only Pattern (Thread Safety)

The original design used O(n) file rewrites to update decision outcomes, causing race conditions under concurrent agent execution. The current design uses append-only writes with read-time reconciliation:

**Initial Records** (written immediately):
```jsonl
{"decision_id": "d1", "timestamp": 1704067200, "task_id": "t1", "classified_as": "routing", "selected_agent": "python-pro", ...}
{"decision_id": "d2", "timestamp": 1704067205, "task_id": "t2", "classified_as": "planning", "selected_agent": "architect", ...}
```

**Outcome Records** (appended on completion - no file rewrites):
```jsonl
{"decision_id": "d1", "outcome_success": true, "outcome_duration_ms": 150, "outcome_cost": 0.045, "update_timestamp": 1704067350}
{"decision_id": "d2", "outcome_success": false, "outcome_escalation": true, "update_timestamp": 1704067410}
```

**ML Export Reconciliation:**
```go
// Read both files
decisions := readJSONL("routing-decisions.jsonl")
updates := readJSONL("routing-decision-updates.jsonl")

// Join and apply latest update per decision
for _, update := range updates {
    if dec, ok := decisions[update.DecisionID]; ok {
        dec.Outcome = update  // Apply latest
    }
}

// Export enriched training data
exportTrainingData(decisions)
```

**Benefits:**
- Thread-safe: Append-only writes never overwrite existing data
- Atomic: Each record is complete upon write
- Fast: Single-pass appends, no file rewrites
- Recoverable: Partial writes don't corrupt existing data

### 4.3 Telemetry Data Types

#### RoutingDecision (pkg/telemetry)
```go
type RoutingDecision struct {
    DecisionID      string   // UUID for join operations
    SessionID       string   // Cross-session aggregation
    Timestamp       int64    // Unix timestamp
    TaskID          string   // Link to original task
    ClassifiedAs    string   // Task type classification
    SelectedAgent   string   // Which agent was routed to
    SelectedTier    string   // "haiku", "sonnet", "opus"
    InputTokens     int      // From PostToolEvent extension
    OutputTokens    int      // Currently zero-valued
    DurationMs      int64    // Calculated from timestamps
    SequenceIndex   int      // Position in session sequence
    Model           string   // Model used
}
```

#### Task Classification Categories
- **routing**: Tier selection, model validation, schema checks
- **planning**: Architecture, design, multi-file coordination
- **implementation**: Code generation, bug fixes, refactoring
- **research**: Library docs, best practices, external APIs
- **review**: Code review, feedback, analysis
- **testing**: Unit tests, integration tests, verification
- **documentation**: README, guides, API docs
- **observability**: Logging, metrics, telemetry

#### AgentCollaboration (pkg/telemetry)
```go
type AgentCollaboration struct {
    CollaborationID    string    // UUID for join operations
    SessionID          string    // Cross-session aggregation
    Timestamp          int64     // Unix timestamp
    PrimaryAgent       string    // Entry-point agent
    DelegatedAgents    []string  // Secondary agents spawned
    AgentCount         int       // Total agents in collaboration
    CollaborationType  string    // "sequential", "parallel", "fallback"
    SequenceIndex      int       // Position in session sequence
}
```

### 4.4 ML Export CLI

```bash
# Export routing decisions with reconciled outcomes
gogent-ml-export routing-decisions --output=decisions.jsonl [--since=YYYY-MM-DD]

# Export agent collaborations with outcomes
gogent-ml-export agent-collaborations --output=collabs.jsonl

# Export review findings with outcomes (NEW in v2.4.0)
gogent-ml-export review-findings --output=findings.jsonl

# Export sharp edge hits (correlation data) (NEW in v2.4.0)
gogent-ml-export sharp-edge-hits --output=hits.jsonl

# Generate review statistics summary (NEW in v2.4.0)
gogent-ml-export review-stats

# Export with automatic pruning after export
gogent-ml-export routing-decisions --output=decisions.jsonl --prune-after-export

# Validate reconciliation consistency
gogent-ml-export validate --check=orphaned-updates --check=missing-outcomes

# Generate summary statistics
gogent-ml-export stats
```

### 4.5 Review Telemetry System (v2.4.0)

The Review Skill ML Telemetry Integration adds comprehensive tracking of code review findings, their resolutions, and correlations with sharp edges.

#### Architecture Overview

```mermaid
graph TD
    subgraph "Review Skill Pipeline"
        ReviewSkill[/review skill]
        Orchestrator[review-orchestrator]
        Backend[backend-reviewer]
        Frontend[frontend-reviewer]
        Standards[standards-reviewer]
    end

    subgraph "Telemetry Capture"
        LogReview[gogent-log-review<br/>PreToolUse Hook]
        UpdateOutcome[gogent-update-review-outcome<br/>Manual]
    end

    subgraph "Data Persistence"
        Findings[(review-findings.jsonl)]
        Outcomes[(review-outcomes.jsonl)]
        SharpHits[(sharp-edge-hits.jsonl)]
    end

    subgraph "ML Pipeline"
        Export[gogent-ml-export]
        Training[ML Training Pipeline]
    end

    ReviewSkill --> Orchestrator
    Orchestrator --> Backend
    Orchestrator --> Frontend
    Orchestrator --> Standards

    Backend --> LogReview
    Frontend --> LogReview
    Standards --> LogReview

    LogReview -->|Writes| Findings
    LogReview -->|If correlated| SharpHits
    UpdateOutcome -->|Writes| Outcomes

    Findings -->|Read-time join| Export
    Outcomes -->|on finding_id| Export
    SharpHits -->|Read| Export

    Export --> Training

    style Findings fill:#e1f5ff
    style Outcomes fill:#fff3e0
    style SharpHits fill:#e8f5e9
    style Export fill:#f3e5f5
```

#### Integration with /review Skill

The `/review` skill coordinates specialized reviewers and captures findings:

**Workflow:**
1. User runs `/review` on codebase/module
2. Skill spawns `review-orchestrator` (Sonnet, Plan mode)
3. Orchestrator spawns domain-specific reviewers (Haiku + Thinking):
   - `backend-reviewer` - API design, security, performance
   - `frontend-reviewer` - Component structure, state management, accessibility
   - `standards-reviewer` - Naming, conventions, code quality
4. Each reviewer returns findings with:
   - `severity`: "critical", "high", "medium", "low", "info"
   - `category`: "security", "performance", "maintainability", etc.
   - `sharp_edge_id`: Optional correlation to known sharp edge
5. Findings piped to `gogent-log-review` (written to telemetry files)
6. Skill outputs to `.claude/tmp/review-telemetry.json`

**Sharp Edge Correlation:**
When findings correlate to previously captured sharp edges (from `pending-learnings.jsonl`), the system:
- Validates `sharp_edge_id` against registry
- Logs correlation to `sharp-edge-hits.jsonl`
- Enables ML learning: "This pattern triggers this sharp edge"

#### Integration with /ticket Skill

The `/ticket` skill tracks finding resolution:

**Workflow:**
1. User runs `/ticket <finding_id>` to implement fix
2. Ticket workflow executes implementation
3. On ticket completion, calls `gogent-update-review-outcome`
4. Outcome written to `review-outcomes.jsonl` with:
   - `finding_id`: Links back to original finding
   - `resolution`: "fixed", "accepted", "wontfix", "duplicate"
   - `implementation_notes`: Context from implementation
   - `outcome_timestamp`: When resolution occurred

**Outcome Reconciliation:**
Similar to routing decisions, review findings use append-only pattern:
- Initial findings never modified
- Outcomes appended separately
- Read-time join during ML export

### 4.6 Review Telemetry Data Types

#### ReviewFinding (pkg/telemetry/review_finding.go)

```go
type ReviewFinding struct {
    FindingID       string   // UUID for outcome join
    SessionID       string   // Cross-session aggregation
    Timestamp       int64    // Unix timestamp
    Reviewer        string   // Which reviewer agent generated
    Severity        string   // "critical", "high", "medium", "low", "info"
    Category        string   // "security", "performance", "maintainability", etc.
    File            string   // File path where finding was observed
    LineRange       string   // "100-105" or "42" (optional)
    Message         string   // Human-readable description
    Recommendation  string   // Suggested fix
    SharpEdgeID     string   // Optional correlation to sharp edge (optional)
    CodeSnippet     string   // Relevant code excerpt (optional)
}
```

#### ReviewOutcomeUpdate (pkg/telemetry/review_finding.go)

```go
type ReviewOutcomeUpdate struct {
    FindingID            string   // Links to ReviewFinding.FindingID
    Resolution           string   // "fixed", "accepted", "wontfix", "duplicate"
    ImplementationNotes  string   // Context from fix implementation
    OutcomeTimestamp     int64    // Unix timestamp
    TicketID             string   // Optional ticket reference (optional)
}
```

#### SharpEdgeHit (pkg/telemetry/sharp_edge_hit.go)

```go
type SharpEdgeHit struct {
    HitID           string   // UUID
    FindingID       string   // Links to ReviewFinding
    SharpEdgeID     string   // Links to sharp edge in pending-learnings.jsonl
    SessionID       string   // Session where correlation detected
    Timestamp       int64    // Unix timestamp
    CorrelationType string   // "direct_match", "pattern_similar", "user_annotated"
}
```

**Correlation Types:**
- `direct_match`: Finding directly triggered by sharp edge pattern
- `pattern_similar`: Finding resembles sharp edge but not exact match
- `user_annotated`: User manually linked finding to sharp edge

### 4.7 Sharp Edge Registry System

The Sharp Edge Registry provides validation and enumeration of valid sharp edge IDs.

**Location:** `~/.claude/memory/pending-learnings.jsonl`

**Functions (pkg/telemetry/sharp_edge_registry.go):**

```go
// LoadSharpEdgeIDs reads all sharp edge IDs from pending-learnings.jsonl
func LoadSharpEdgeIDs(path string) ([]string, error)

// IsValidSharpEdgeID checks if a given ID exists in the registry
func IsValidSharpEdgeID(sharpEdgeID string, registryPath string) (bool, error)

// GetAllSharpEdgeIDs returns complete list for enumeration
func GetAllSharpEdgeIDs() ([]string, error)
```

**Registry Validation:**
- `gogent-log-review` validates `sharp_edge_id` before writing
- Invalid IDs rejected with error (not logged to `sharp-edge-hits.jsonl`)
- Registry updates when new sharp edges captured by `gogent-sharp-edge`

**Example Usage:**
```bash
# List all known sharp edge IDs
gogent-ml-export sharp-edge-hits --list-ids

# Validate finding correlation
gogent-log-review < finding_with_sharp_edge_id.json
# → Checks registry, logs to sharp-edge-hits.jsonl if valid
```

---

## 5. Data Persistence Layer

All persistence uses JSONL (JSON Lines) format for append-only writes and streaming reads.

```mermaid
flowchart TB
    subgraph "Session Scope (/tmp/)"
        violations[/routing-violations.jsonl/]
        counter[/tool-counter-{id}.log/]
        patterns[/error-patterns.jsonl/]
    end

    subgraph "Project Scope (.claude/memory/)"
        handoffs[/handoffs.jsonl/]
        intents[/user-intents.jsonl/]
        decisions[/decisions.jsonl/]
        prefs[/preferences.jsonl/]
        pending[/pending-learnings.jsonl/]
        lasthandoff[/last-handoff.md/]
    end

    subgraph "ML Scope ($XDG_DATA_HOME/gogent/)"
        routingdec[/routing-decisions.jsonl/]
        routingupdates[/routing-decision-updates.jsonl/]
        collaborations[/agent-collaborations.jsonl/]
        collabupd[/agent-collaboration-updates.jsonl/]
        reviewfindings[/review-findings.jsonl/]
        reviewoutcomes[/review-outcomes.jsonl/]
        sharpedgehits[/sharp-edge-hits.jsonl/]
    end

    subgraph "Global Scope (~/.gogent/)"
        failures[/failure-tracker.jsonl/]
        invocations[/agent-invocations.jsonl/]
        escalations[/escalations.jsonl/]
        scoutrecs[/scout-recommendations.jsonl/]
    end

    subgraph "Archive (session-archive/)"
        archived[/learnings-{ts}.jsonl/]
        sessions[/session-{id}.jsonl/]
    end

    validate([gogent-validate]) --> violations
    sharpedge([gogent-sharp-edge]) --> failures
    sharpedge --> pending
    sharpedge --> routingdec
    endstate([gogent-agent-endstate]) --> routingupdates
    endstate --> collaborations
    endstate --> collabupd
    logreview([gogent-log-review]) --> reviewfindings
    logreview --> sharpedgehits
    updatereview([gogent-update-review-outcome]) --> reviewoutcomes
    archive([gogent-archive]) --> handoffs
    archive --> lasthandoff
    archive --> archived
    intent([gogent-capture-intent]) --> intents
    mlexport([gogent-ml-export]) -.->|reads| routingdec
    mlexport -.->|reads| routingupdates
```

### 5.1 File Reference

| File | Scope | Written By | Schema | Purpose |
|------|-------|------------|--------|---------|
| `handoffs.jsonl` | Project | gogent-archive | Handoff v1.3 | Session continuity |
| `user-intents.jsonl` | Project | gogent-capture-intent | UserIntent | User preference tracking |
| `decisions.jsonl` | Project | gogent-archive | Decision | Architectural decisions |
| `preferences.jsonl` | Project | gogent-archive | PreferenceOverride | User overrides |
| `pending-learnings.jsonl` | Project | gogent-sharp-edge | SharpEdge | Unreviewed learnings |
| `last-handoff.md` | Project | gogent-archive | Markdown | Human-readable summary |
| `failure-tracker.jsonl` | Global | gogent-sharp-edge | FailureInfo | Cross-session tracking |
| `agent-invocations.jsonl` | Global | gogent-* | AgentInvocation | Invocation telemetry |
| `escalations.jsonl` | Global | gogent-agent-endstate | EscalationEvent | Tier escalation tracking |
| `scout-recommendations.jsonl` | Global | gogent-validate | ScoutRecommendation | Scout accuracy |
| `routing-violations.jsonl` | Temp | gogent-validate | Violation | Session violations |
| `routing-decisions.jsonl` | ML | gogent-sharp-edge | RoutingDecision | ML training data |
| `routing-decision-updates.jsonl` | ML | gogent-agent-endstate | DecisionOutcome | Outcome updates |
| `agent-collaborations.jsonl` | ML | gogent-agent-endstate | AgentCollaboration | Team patterns |
| `agent-collaboration-updates.jsonl` | ML | gogent-agent-endstate | CollabOutcome | Outcome updates |
| `review-findings.jsonl` | ML | gogent-log-review | ReviewFinding | Code review findings |
| `review-outcomes.jsonl` | ML | gogent-update-review-outcome | ReviewOutcomeUpdate | Finding resolutions |
| `sharp-edge-hits.jsonl` | ML | gogent-log-review | SharpEdgeHit | Sharp edge correlations |
| `impl-progress.json` | Project | impl-manager | ImplProgress | Task progress tracking |
| `impl-violations.jsonl` | Project | impl-manager | ImplViolation | Convention violations during implementation |

---

## 6. Validation Pipeline

The `gogent-validate` binary orchestrates multiple validation checks for Task tool invocations.

```mermaid
flowchart TD
    input[Task Invocation] --> parse[Parse ToolEvent]
    parse --> check1{Tool == Task?}

    check1 -->|No| pass1[Pass through]
    check1 -->|Yes| load[Load Schema v2.2.0]

    load --> v1[Check 1: Einstein/Opus Blocking]
    v1 -->|task_invocation_blocked| block1[BLOCK: Use /einstein instead]
    v1 -->|OK| v2[Check 2: Delegation Ceiling]

    v2 -->|Tier exceeds ceiling| block2[BLOCK: Ceiling violation]
    v2 -->|OK| v3[Check 3: Subagent Type Mapping]

    v3 -->|Invalid mapping| block3[BLOCK: Wrong subagent_type]
    v3 -->|OK| v4[Check 4: Model Mismatch Warning]

    v4 -->|Mismatch| warn[WARN: Continue with warning]
    v4 -->|OK| allow[ALLOW]

    block1 --> log[Log Violation to JSONL]
    block2 --> log
    block3 --> log
    warn --> allow
```

### 6.1 Validation Checks

| Check | Blocking | Logged | Purpose |
|-------|----------|--------|---------|
| Einstein/Opus | Yes | Yes | Prevent Task(opus) - use /einstein instead |
| Delegation Ceiling | Yes | Yes | Enforce max tier from calculate-complexity |
| Subagent Type | Yes | Yes | Ensure agent-subagent_type pairing matches schema |
| Model Mismatch | No | No | Warn if requested model differs from agents-index |

### 6.2 Agent-Subagent Mapping (routing-schema v2.4.0)

#### Core System Agents (v2.2.0)

| Agent | Required subagent_type | Rationale |
|-------|------------------------|-----------|
| codebase-search | Explore | Read-only reconnaissance |
| haiku-scout | Explore | Scope assessment |
| code-reviewer | Explore | Non-destructive analysis |
| librarian | Explore | Research tasks |
| tech-docs-writer | general-purpose | Write permissions for docs |
| scaffolder | general-purpose | Create new files |
| python-pro | general-purpose | Implementation work |
| go-pro | general-purpose | Implementation work |
| orchestrator | Plan | Planning mode coordination |
| architect | Plan | Design and planning |
| gemini-slave | Bash | External shell piping |

#### Review System Agents (Added in v2.4.0)

| Agent | Required subagent_type | Rationale |
|-------|------------------------|-----------|
| typescript-pro | general-purpose | TypeScript implementation work |
| react-pro | general-purpose | React component implementation |
| backend-reviewer | Explore | Read-only API/security review |
| frontend-reviewer | Explore | Read-only UI/component review |
| standards-reviewer | Explore | Read-only code quality review |
| review-orchestrator | Plan | Multi-domain review coordination |
| staff-architect-critical-review | Plan | Critical architectural review (Opus tier) |

#### Implementation Coordination (Added in v2.5.0)

| Agent | Required subagent_type | Rationale |
|-------|------------------------|-----------|
| impl-manager | Plan | Implementation coordination and planning |

---

## 7. Cost Tracking & Tier Pricing

The system tracks costs per tier using pricing from `routing-schema.json`:

| Tier | Model | Cost/1K Tokens | Thinking Budget | Use Cases |
|------|-------|----------------|-----------------|-----------|
| haiku | haiku | $0.0005 | 0 | File search, counting, formatting |
| haiku_thinking | haiku | $0.001 | 6K | Scaffolding, documentation, review |
| sonnet | sonnet | $0.009 | 16K | Implementation, refactoring, debugging |
| opus | opus | $0.045 | 32K | Deep analysis (via /einstein only) |
| external | gemini-2.0 | $0.0001 | 0 | Large context (1M+ tokens) |

### 7.1 Cost Calculation

```go
// From pkg/telemetry/cost.go
func CalculateInvocationCost(inv AgentInvocation, pricing TierPricing) InvocationCost {
    totalTokens := inv.InputTokens + inv.OutputTokens + inv.ThinkingTokens
    rate := pricing.GetTierCostRate(inv.Tier)
    cost := float64(totalTokens) * rate / 1000.0

    return InvocationCost{
        Agent:       inv.Agent,
        Tier:        inv.Tier,
        TotalTokens: totalTokens,
        TotalCost:   cost,
    }
}
```

### 7.2 Session Cost Summary

```go
type SessionCostSummary struct {
    SessionID       string                      `json:"session_id"`
    TotalCost       float64                     `json:"total_cost"`
    TotalTokens     int                         `json:"total_tokens"`
    InvocationCount int                         `json:"invocation_count"`
    ByAgent         map[string]*AgentCostSummary `json:"by_agent"`
    ByTier          map[string]*TierCostSummary  `json:"by_tier"`
}
```

---

## 8. Handoff Schema v1.3

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
        +EndstateLog[] agent_endstates
    }

    class SessionMetrics {
        +int tool_calls
        +int errors_logged
        +int routing_violations
        +string session_id
    }

    class SharpEdge {
        +string file
        +string error_type
        +int consecutive_failures
        +string context
        +int64 timestamp
        +string type
        +string tool
        +string code_snippet
        +string status
    }

    Handoff --> SessionContext
    Handoff --> HandoffArtifacts
    SessionContext --> SessionMetrics
    HandoffArtifacts --> SharpEdge
```

### 8.1 Schema Version History

| Version | Changes |
|---------|---------|
| 1.3 | Added AgentEndstates field for SubagentStop tracking |
| 1.2 | Extended SharpEdge with type, tool, code_snippet, status |
| 1.1 | Added decisions, preference_overrides, performance_metrics |
| 1.0 | Initial schema with sharp_edges, routing_violations, error_patterns |

---

## 9. CLI Reference

### 9.1 Hook Binaries

| Binary | Hook Event | Input | Output | Lines |
|--------|------------|-------|--------|-------|
| `gogent-load-context` | SessionStart | SessionStartEvent JSON | ContextInjection JSON | ~350 |
| `gogent-validate` | PreToolUse | ToolEvent JSON | ValidationResult JSON | ~200 |
| `gogent-sharp-edge` | PostToolUse | ToolEvent JSON | HookResponse JSON | ~500 |
| `gogent-agent-endstate` | SubagentStop | SubagentEvent JSON | HookResponse JSON | ~400 |
| `gogent-archive` | SessionEnd | SessionEvent JSON | Confirmation JSON | ~1200 |
| **`gogent-log-review`** | **PreToolUse** | **ReviewFinding JSON** | **HookResponse JSON** | **~250** |
| **`gogent-update-review-outcome`** | **Manual** | **ReviewOutcomeUpdate JSON** | **Confirmation JSON** | **~150** |
| **`gogent-orchestrator-guard`** | **SubagentStop** | **Agent lifecycle** | **Enforcement** | **~300** |

**Note:** `gogent-orchestrator-guard` recognizes `impl-manager` as an orchestrating agent (alongside `orchestrator` and `review-orchestrator`) for background task collection enforcement.

### 9.2 Utility Binaries

| Binary | Purpose | Subcommands |
|--------|---------|-------------|
| `gogent-capture-intent` | Manual user intent logging | (stdin) |
| `gogent-aggregate` | Session statistics | (flags) |
| `gogent-ml-export` | ML training data export | routing-decisions, agent-collaborations, **review-findings, review-stats, sharp-edge-hits**, validate, stats |
| `gogent-orchestrator-guard` | Completion enforcement | (integration) |
| `gogent-doc-theater` | Documentation theater detection | (integration) |
| **`gogent-scout`** | **Unified pre-routing reconnaissance** | **(standalone)** |

### 9.3 Scout Utilities

The `gogent-scout` binary provides unified pre-routing reconnaissance with smart backend selection.

| Aspect | Detail |
|--------|--------|
| **Purpose** | Assess scope before committing expensive resources |
| **Backends** | Native Go (score < 40), Gemini 3 Flash (score ≥ 40), Synthetic fallback |
| **Output** | JSON to stdout + `.claude/tmp/scout_metrics.json` |
| **Latency** | Native: < 100ms (p50), Gemini: 1-3s |

**Usage:**
```bash
# Basic usage
gogent-scout <target-directory> "<instruction>"

# Examples
gogent-scout ./pkg/routing "Assess scope for refactoring"
gogent-scout ./cmd "How many files in cmd?"

# Environment overrides
SCOUT_BACKEND=native gogent-scout ./large-module "Force native"
SCOUT_THRESHOLD=60 gogent-scout ./module "Higher threshold"
```

**Backend Selection (Multi-Factor Score):**
- File count: 40% weight (20 files = 100 points)
- Line count: 30% weight (5000 lines = 100 points)
- Max file size: 20% weight (1000 lines = 100 points)
- Language count: 10% weight (4 languages = 100 points)
- **Threshold:** Score < 40 → Native, ≥ 40 → Gemini

**Fallback Chain:**
1. Primary backend (based on score)
2. Opposite backend (if primary fails)
3. Synthetic report (if both fail - always returns valid JSON)

**Output Schema (v1.0):**
```json
{
  "schema_version": "1.0",
  "backend": "native|gemini|native_fallback|synthetic_fallback",
  "scope_metrics": { "total_files", "total_lines", "estimated_tokens", ... },
  "complexity_signals": { "available", "import_density", ... },
  "routing_recommendation": { "recommended_tier", "confidence", "reasoning" },
  "key_files": [...],
  "warnings": [...]
}
```

**Integration:** Used by `/explore`, `/plan`, `/review` skills and `orchestrator` agent for pre-routing reconnaissance.

### 9.4 Archive Query Subcommands

```bash
gogent-archive list [--since=DATE] [--has-sharp-edges]
gogent-archive show <session_id>
gogent-archive stats
gogent-archive sharp-edges [--file=PATTERN] [--status=pending]
gogent-archive user-intents [--category=CATEGORY]
gogent-archive decisions [--since=DATE]
gogent-archive preferences
gogent-archive performance
gogent-archive weekly
```

---

## 10. Testing Infrastructure

### 10.1 Test Fixtures

**Location:** `test/simulation/fixtures/deterministic/`

| Hook | Fixtures | Purpose |
|------|----------|---------|
| SessionStart | 10 | Language detection, handoff loading, git context |
| PreToolUse | 8 | Validation, blocking, ceiling checks |
| PostToolUse | 6 | Tool counting, failure tracking, ML logging |
| SubagentStop | 5 | Outcome updates, collaboration tracking |

### 10.2 Coverage Metrics

| Package | Coverage | Key Functions |
|---------|----------|---------------|
| `pkg/routing` | ~88% | LoadSchema, ValidateTask, LogViolation |
| `pkg/session` | ~85% | GenerateHandoff, LoadArtifacts, DetectLanguage |
| `pkg/memory` | ~82% | LogFailure, GetFailureCount, PatternMatch |
| `pkg/telemetry` | ~80% | LogInvocation, CalculateCost, ClusterByAgent |
| `pkg/config` | ~90% | GetGOgentDir, GetCurrentTier |

### 10.3 CI/CD Workflows

```yaml
# Primary workflows
sessionstart.yml:      # SessionStart hook tests
pretooluse.yml:        # PreToolUse validation tests
posttooluse.yml:       # PostToolUse handler tests
subagent.yml:          # SubagentStop integration tests
ml-telemetry.yml:      # ML pipeline validation

# Integration
simulation.yml:        # Full simulation harness
benchmark.yml:         # Performance regression
coverage.yml:          # Coverage reporting
```

---

## 11. Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `GOGENT_PROJECT_DIR` | Project root override | `$PWD` |
| `GOGENT_ROUTING_SCHEMA` | Schema path override | `~/.claude/routing-schema.json` |
| `GOGENT_STORAGE_PATH` | Failure tracker path | `~/.gogent/failure-tracker.jsonl` |
| `GOGENT_MAX_FAILURES` | Debugging loop threshold | 3 |
| `GOGENT_FAILURE_WINDOW` | Failure window (seconds) | 300 |
| `GOGENT_REMINDER_THRESHOLD` | Routing reminder frequency | 10 |
| `GOGENT_FLUSH_THRESHOLD` | Auto-flush threshold | 20 |
| `XDG_DATA_HOME` | ML telemetry base path | `~/.local/share` |
| **`GOGENT_ENABLE_TELEMETRY`** | **Enable/disable review telemetry** | **1** |

---

## 12. Key File Paths

```
Project/
├── .claude/
│   ├── memory/
│   │   ├── handoffs.jsonl           # Session history
│   │   ├── user-intents.jsonl       # User preferences
│   │   ├── decisions.jsonl          # Architectural decisions
│   │   ├── preferences.jsonl        # Preference overrides
│   │   ├── pending-learnings.jsonl  # Unreviewed sharp edges
│   │   └── last-handoff.md          # Human-readable handoff
│   ├── tmp/
│   │   └── einstein-gap-*.md        # Escalation documents
│   └── session-archive/             # Archived session data
│
~/.gogent/
├── failure-tracker.jsonl            # Cross-session failures
├── agent-invocations.jsonl          # Invocation telemetry
├── escalations.jsonl                # Tier escalations
└── scout-recommendations.jsonl      # Scout accuracy data
│
$XDG_DATA_HOME/gogent/
├── routing-decisions.jsonl          # ML training (initial)
├── routing-decision-updates.jsonl   # ML training (outcomes)
├── agent-collaborations.jsonl       # Team patterns (initial)
├── agent-collaboration-updates.jsonl # Team patterns (outcomes)
├── review-findings.jsonl            # Code review findings (v2.4.0)
├── review-outcomes.jsonl            # Finding resolutions (v2.4.0)
└── sharp-edge-hits.jsonl            # Sharp edge correlations (v2.4.0)
│
/tmp/
├── claude-routing-violations.jsonl  # Current session violations
├── claude-tool-counter-*.log        # Tool call counters
└── claude-error-patterns.jsonl      # Session error patterns
│
~/.claude/
├── routing-schema.json              # v2.2.0 - Source of truth
├── CLAUDE.md                        # Global configuration
├── conventions/                     # Language conventions
│   ├── python.md
│   ├── go.md
│   └── R.md
├── agents/                          # Agent definitions
│   └── */{agent-name}.md            # Unified YAML frontmatter + instructions
│   └── */sharp-edges.yaml           # Known failure patterns (separate)
├── skills/                          # Skill definitions
│   └── */SKILL.md
└── rules/                           # Behavioral rules
    ├── LLM-guidelines.md
    └── agent-behavior.md
```

---

## 13. How to Extend

### 13.1 Adding a New CLI

1. Create `cmd/gogent-<name>/main.go`
2. Implement STDIN parsing with timeout (see `pkg/routing/stdin.go`)
3. Output hook-compatible JSON to STDOUT
4. Add to `Makefile` build targets
5. Document in CLI Reference section

### 13.2 Adding a New Package

1. Create `pkg/<name>/` with `doc.go`
2. Define types in dedicated files (one primary type per file)
3. Add `_test.go` files (target 80%+ coverage)
4. Update Package Architecture diagram
5. Document in Package Responsibilities table

### 13.3 Adding a New Artifact Type

1. Define struct in `pkg/session/handoff_artifacts.go`
2. Add to `HandoffArtifacts` struct with `omitempty` tag
3. Update `LoadArtifacts()` in `pkg/session/handoff.go`
4. Add query method to `pkg/session/query.go`
5. Add CLI subcommand if user-queryable
6. Bump handoff schema version if breaking change

### 13.4 Adding ML Telemetry Fields

1. Add field to `RoutingDecision` or `AgentCollaboration`
2. Update logging in appropriate hook handler
3. Update `gogent-ml-export` reconciliation logic
4. Add migration code for existing JSONL files
5. Update training pipeline documentation

---

## 14. Related Documentation

| Document | Purpose |
|----------|---------|
| `CLAUDE.md` (project) | Project-level Claude configuration |
| `~/.claude/CLAUDE.md` | Global Claude configuration with routing gates |
| `~/.claude/routing-schema.json` | Source of truth for tier definitions |
| `~/.claude/docs/agent-reference-table.md` | Complete agent reference |
| `~/.claude/rules/LLM-guidelines.md` | Behavioral guidelines |
| `dev/will/migration_plan/tickets/` | Implementation tickets |

---

## 15. TUI System Architecture (GOgent-109 to GOgent-121) — Legacy

> **Status: SUPERSEDED.** This section documents the original TUI implementation (React/Ink-adjacent Go prototype).
> A complete rewrite is underway — see Section 16 for the TUI Migration (TUI-001 to TUI-042).
> The old ticket IDs (GOgent-109 to GOgent-121) are mapped to new equivalents in `tickets/tui-migration/tickets/overview.md`.

The TUI system provides a complete terminal interface for Claude Code interaction with real-time telemetry visualization.

### 15.1 Package Structure

```
internal/
├── cli/                          # Claude CLI subprocess management
│   ├── subprocess.go             # ClaudeProcess lifecycle, NDJSON I/O
│   ├── events.go                 # Event types: system, assistant, result, error
│   ├── streams.go                # NDJSONReader, NDJSONWriter
│   ├── restart.go                # RestartPolicy, exponential backoff
│   └── session.go                # SessionManager, Session struct
│
└── tui/                          # Bubbletea components
    ├── agents/                   # Agent tree visualization
    │   ├── model.go              # AgentTree, AgentNode, AgentStatus
    │   ├── view.go               # TreeModel (tea.Model), styles
    │   └── detail.go             # DetailModel, status indicators
    │
    ├── claude/                   # Claude conversation panel
    │   ├── panel.go              # PanelModel, viewport, textarea
    │   ├── input.go              # handleInput(), sendMessage()
    │   ├── output.go             # updateViewport(), appendStreamingText()
    │   └── events.go             # handleEvent(), renderHookSidebar()
    │
    ├── layout/                   # Main layout
    │   ├── layout.go             # 70/30 split, focus management
    │   └── banner.go             # BannerModel, navigation tabs
    │
    ├── session/                  # Session picker
    │   └── picker.go             # PickerModel, list/resume/delete
    │
    └── dashboard/                # Performance dashboard shell
```

### 15.2 Component Dependencies

```mermaid
graph TD
    subgraph "CLI Layer"
        CP[ClaudeProcess] --> EV[Events]
        CP --> ST[Streams]
        CP --> RS[RestartPolicy]
        SM[SessionManager] --> CP
    end

    subgraph "TUI Layer"
        LO[Layout Model] --> CL[Claude Panel]
        LO --> AG[Agent Tree]
        LO --> DE[Agent Detail]
        LO --> BN[Banner]
        CL --> CP
        AG --> TW[TelemetryWatcher]
        SP[Session Picker] --> SM
    end

    subgraph "Data Sources"
        TW --> |fsnotify| AL[agent-lifecycle.jsonl]
        TW --> |fsnotify| RD[routing-decisions.jsonl]
        TW --> |fsnotify| IP[impl-progress.json]
    end
```

### 15.3 Implemented Tickets

| Ticket | Component | Status | Test Coverage |
|--------|-----------|--------|---------------|
| GOgent-109 | Agent Lifecycle Telemetry | ✅ Complete | ~85% |
| GOgent-110 | CLI Subprocess Management | ✅ Complete | ~85% |
| GOgent-111 | Performance Dashboard Shell | ✅ Complete | ~80% |
| GOgent-112 | Auto-Restart on Panic | ✅ Complete | ~88% |
| GOgent-113 | File Watchers for Telemetry | ✅ Complete | ~82% |
| GOgent-114 | Event System Integration | ✅ Complete | ~90% |
| GOgent-115 | Agent Tree Model | ✅ Complete | ~92% |
| GOgent-116 | Tree View Component | ✅ Complete | ~88% |
| GOgent-117 | Agent Detail Sidebar | ✅ Complete | ~85% |
| GOgent-118 | Claude Conversation Panel | ✅ Complete | ~90% |
| GOgent-119 | 70/30 Layout Integration | ✅ Complete | ~88% |
| GOgent-120 | Persistent Banner | ✅ Complete | ~85% |
| GOgent-121 | Session Management | ✅ Complete | ~85% |

### 15.4 Key Interfaces

```go
// internal/cli/subprocess.go
type ClaudeProcess struct { ... }
func (cp *ClaudeProcess) Start() error
func (cp *ClaudeProcess) Stop() error
func (cp *ClaudeProcess) Send(message string) error
func (cp *ClaudeProcess) Events() <-chan Event
func (cp *ClaudeProcess) SessionID() string
func (cp *ClaudeProcess) IsRunning() bool

// internal/tui/agents/model.go
type AgentTree struct { ... }
func (at *AgentTree) ProcessSpawn(event *telemetry.AgentLifecycleEvent) error
func (at *AgentTree) ProcessComplete(event *telemetry.AgentLifecycleEvent) error
func (at *AgentTree) GetNode(agentID string) (*AgentNode, bool)
func (at *AgentTree) WalkTree(fn func(*AgentNode) bool)

// internal/tui/layout/layout.go
type Model struct { ... }
func NewModel(claudePanel, agentTree, sessionID) Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

### 15.5 Data Flow

```
Claude CLI (subprocess)
    │
    ├─── STDOUT (NDJSON) ──► NDJSONReader ──► Events() chan
    │                                              │
    │                                              ▼
    │                                         Claude Panel
    │                                         (streaming text)
    │
    └─── STDIN (NDJSON) ◄── NDJSONWriter ◄── Send()
                                              │
                                              ▲
                                         User Input

Telemetry Files (fsnotify)
    │
    ├─── agent-lifecycle.jsonl ──► TelemetryWatcher ──► AgentTree
    │                                                      │
    │                                                      ▼
    └─── routing-decisions.jsonl                      Tree View
                                                     Detail View
```

---

*This document is designed for incremental updates. When adding new components, update the relevant section and diagram rather than rewriting prose.*

---

## 16. TUI Migration: Go/Bubble Tea Rewrite (TUI-001 to TUI-042)

> **Source of truth:** `tickets/tui-migration/tickets/overview.md`
> **Braintrust analysis:** `tickets/tui-migration/braintrust-handoff-v2.md`
> **Generated:** 2026-03-16 via /plan-tickets workflow
> **Review status:** APPROVE_WITH_CONDITIONS (Staff Architect, High Confidence)

### 16.1 Motivation

The original TUI (Section 15) was a functional prototype. Braintrust analysis identified that the planned three-process topology (Go TUI + TS MCP sidecar + CLI) had 9 failure modes and a permanent Node.js dependency. The migration replaces it with a **two-process topology** (Go TUI + Claude Code CLI), with MCP served natively from Go.

### 16.2 Two-Process Architecture

```
Go TUI Process (single binary)
  |-- Bubble Tea event loop (owns terminal stdin/stdout)
  |-- CLI Driver (manages Claude CLI subprocess via pipes)
  |-- IPC Bridge (UDS listener for MCP server communication)
  |
  +--spawns--> Claude Code CLI (--output-format stream-json)
                  |
                  +--spawns--> gofortress-mcp (Go MCP server, stdio transport)
                                  |
                                  +--connects--> TUI via UDS side channel
```

### 16.3 Phase Summary

| Phase | Description | Tickets | Status |
|-------|-------------|---------|--------|
| 1 | Prerequisite Spikes | TUI-001 to TUI-004 | COMPLETE |
| 2 | Foundation (theme, focus, keybinds, layout) | TUI-005 to TUI-011 | COMPLETE |
| 3 | CLI Driver + NDJSON + MCP Server | TUI-012 to TUI-016 | COMPLETE |
| 4 | Modal System | TUI-017, TUI-018 | COMPLETE |
| 5 | Agent Tree + Process Management | TUI-019 to TUI-021 | COMPLETE |
| 6 | Rich Features | TUI-022 to TUI-027 | COMPLETE |
| 7 | Settings, Providers, Teams | TUI-028 to TUI-032 | COMPLETE |
| 8 | Lifecycle | TUI-033 to TUI-035 | COMPLETE |
| 9 | Integration Testing | TUI-036 to TUI-042 | COMPLETE |

### 16.4 Spike Results (Phase 1)

| Spike | Key Finding | Artifact |
|-------|-------------|----------|
| TUI-001 (Permission wire format) | No control_request protocol exists. Architecture uses Option D (hybrid): acceptEdits + MCP side-channel | `spike-results/permission-protocol.md` |
| TUI-002 (Go MCP SDK POC) | v1.2.0 works, full roundtrip confirmed. MCP tools need `--allowedTools` | `spike-results/go-mcp-poc.md` |
| TUI-003 (NDJSON catalog) | 6 top-level types + stream_event catalogued. Log-and-continue handles unknowns | `spike-results/ndjson-catalog.md` |
| TUI-004 (UDS IPC POC) | Exponential backoff validated (connected attempt 1). UDS roundtrip 56us | `spike-results/ipc-protocol.md` |

### 16.5 Legacy Ticket Traceability

| Old Ticket | New Ticket(s) | Disposition |
|-----------|---------------|-------------|
| GOgent-109 | TUI-019, TUI-021 | SUPERSEDED |
| GOgent-110 | TUI-013 | SUPERSEDED |
| GOgent-111 | TUI-010, TUI-032 | SUPERSEDED |
| GOgent-112 | TUI-013, TUI-016 | PRESERVED |
| GOgent-113 | TUI-027, TUI-032 | MODIFIED |
| GOgent-114 | TUI-012 | SUPERSEDED |
| GOgent-115 | TUI-019 | SUPERSEDED |
| GOgent-116 | TUI-020 | SUPERSEDED |
| GOgent-117 | TUI-020 | SUPERSEDED |
| GOgent-118 | TUI-022 | SUPERSEDED |
| GOgent-119 | TUI-010 | SUPERSEDED |
| GOgent-120 | TUI-009 | SUPERSEDED |
| GOgent-121 | TUI-033, TUI-034 | SUPERSEDED |

### 16.6 Implemented Package Tree (Phases 2-5, see overview.md for full Phase 2-9 tree)

```
internal/tui/
├── cli/                              # NDJSON events + CLI driver + agent sync
│   ├── events.go                     # 9 event structs, ParseCLIEvent() (TUI-012)
│   ├── driver.go                     # CLIDriver: subprocess lifecycle, NDJSON streaming (TUI-013)
│   ├── agent_sync.go                 # SyncAssistantEvent/SyncUserEvent, ParseTaskInput (TUI-021)
│   └── *_test.go                     # 152 tests, 84.1% coverage, race-free
├── mcp/                              # MCP server tools + IPC protocol (TUI-014)
│   ├── protocol.go                   # IPCRequest/Response, 6 payload types
│   ├── tools.go                      # 7 tool handlers + UDSClient
│   └── tools_test.go                 # 30 tests, 74.4% coverage
├── bridge/                           # TUI-side UDS listener (TUI-015)
│   ├── server.go                     # IPCBridge, modal correlation, ResolveModalSimple (TUI-018)
│   └── server_test.go                # 10 tests, 76.2% coverage
├── state/                            # Agent state tracking (TUI-019)
│   ├── agent.go                      # AgentRegistry (RWMutex), Agent, AgentTreeNode, AgentStats
│   └── agent_test.go                 # 56 tests, 96.1% coverage, race-free
├── components/
│   ├── banner/banner.go              # BannerModel (TUI-009)
│   ├── statusline/statusline.go      # StatusLineModel (TUI-009)
│   ├── tabbar/tabbar.go              # TabBarModel (TUI-009)
│   ├── modals/                       # Modal system (TUI-017, TUI-018)
│   │   ├── types.go                  # ModalType enum, ModalRequest/Response
│   │   ├── model.go                  # ModalModel (tea.Model), ModalResponseMsg
│   │   ├── queue.go                  # ModalQueue (sequential FIFO gate)
│   │   ├── permission.go             # PermissionHandler (6 flow types, multi-step)
│   │   └── *_test.go                 # 107 tests, 88.5% coverage
│   └── agents/                       # Agent tree + detail views (TUI-020)
│       ├── tree.go                   # AgentTreeModel (scrollable, Unicode connectors)
│       ├── detail.go                 # AgentDetailModel (display-only)
│       └── *_test.go                 # 41 tests, 90.6% coverage
├── config/
│   ├── theme.go                      # Colors, styles, icons, Theme struct (TUI-005)
│   └── keys.go                       # KeyMap, 24 bindings, 5 groups (TUI-007)
└── model/
    ├── focus.go                      # FocusTarget, RightPanelMode (TUI-006)
    ├── app.go                        # AppModel, sharedState, layout, permission wiring (TUI-008/010/018)
    ├── startup.go                    # CLI startup sequence (TUI-016)
    └── messages.go                   # 18 tea.Msg types (TUI-008/016)

cmd/
├── gofortress/main.go                # TUI entry point, stdlib flag (TUI-011)
└── gofortress-mcp/main.go            # MCP server stub (TUI-011)
```

~1153+ tests across 23+ packages (as of Phase 8). See `overview.md` for full package tree and current counts.

#### Import Graph (acyclic)

```
config ←── banner, tabbar, statusline, modals, agents
state  ←── agents, cli (agent_sync)
modals ←── model (ModalResponseMsg type-switch)
model  ←── bridge (BridgeModalRequestMsg)
mcp    ←── bridge (ModalResponsePayload), pkg/routing
```

Key cycle-prevention interfaces in `model` package:
- `tabBarWidget` (breaks model ↔ tabbar)
- `cliDriverWidget` (breaks model ↔ cli)
- `bridgeWidget` with `ResolveModalSimple` (breaks model ↔ mcp)

#### sharedState (pointer-based, survives Bubbletea value-copy)

```go
type sharedState struct {
    cliDriver   cliDriverWidget           // TUI-016
    bridge      bridgeWidget              // TUI-016
    modalQueue  *modals.ModalQueue        // TUI-018
    permHandler *modals.PermissionHandler // TUI-018
}
```

### 16.7 Review Conditions (resolved)

- **C-1:** Go version constraint updated from "Go 1.22+" to "Go 1.25+" ✅ Applied to all tickets
- **C-2:** MCP SDK version v1.2.0 confirmed correct by TUI-002 spike ✅ No upgrade needed

### 16.8 References

| Document | Location |
|----------|----------|
| Implementation overview | `tickets/tui-migration/tickets/overview.md` |
| Ticket index (42 tickets) | `tickets/tui-migration/tickets/tickets-index.json` |
| Individual tickets | `tickets/tui-migration/tickets/TUI-{001..042}.md` |
| Strategy plan | `.claude/sessions/20260316-plan-tickets-tui/strategy.md` |
| Implementation specs | `.claude/sessions/20260316-plan-tickets-tui/specs.md` |
| Staff architect review | `.claude/sessions/20260316-plan-tickets-tui/review-critique.md` |
| Braintrust analysis | `tickets/tui-migration/braintrust-handoff-v2.md` |
| Spike results | `tickets/tui-migration/spike-results/` |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.7 | 2026-03-23 | **TUI MIGRATION COMPLETE.** All 42 tickets done (42/42). Phase 9 finished: TUI-041 resilience tests (91.2%), TUI-042 feature parity (16/18 pass, 2 partial stubs). verify-parity.sh: 75 pass, 0 fail, 2 skip. Updated all status to COMPLETE. |
| 1.6 | 2026-03-23 | Updated Section 16 for Phases 6-9 progress: Phases 6-8 COMPLETE, Phase 9 IN PROGRESS (5/7: TUI-036–040 done, TUI-041–042 pending). Updated test counts (~1153+ tests, 23+ packages). Added reference to overview.md for full package tree. Performance benchmarks all pass (TUI-040). |
| 1.5 | 2026-03-23 | Updated Section 16 for Phases 4-5 completion: modal system (TUI-017/018), agent tree (TUI-019/020/021). Updated package tree to 12 packages/557 tests. Added import graph, sharedState diagram, cycle-prevention interfaces. Phase status 4+5 → COMPLETE. |
| 1.4 | 2026-03-23 | Added Section 16: TUI Migration (TUI-001 to TUI-042). Marked Section 15 as legacy/superseded. Updated header to reflect v2 migration status. Fixed `.ticket-config.json` stale reference. |
| 1.3 | 2026-02-02 | Added `gogent-scout` unified scout binary (Section 9.3): smart backend routing (native Go / Gemini 3 Flash), multi-factor scoring, fallback chain, updated routing-schema.json scout_protocol |
| 1.2 | 2026-02-01 | Added Review Skill ML Telemetry System (v2.4.0): Section 4.5-4.7, 6 new agents, gogent-log-review/gogent-update-review-outcome binaries, review-findings.jsonl/review-outcomes.jsonl/sharp-edge-hits.jsonl telemetry files |
| 1.1 | 2026-01-26 | Complete TUI implementation (GOgent-109 through GOgent-121), Section 15 added |
| 1.0 | 2026-01-20 | Initial architecture documentation |

---

**Version:** 1.7
**Generated:** 2026-03-23
**Maintainer:** GOgent-Fortress Development Team
**TUI v1 (Legacy):** GOgent-109 through GOgent-121 (13 tickets, all tests passing) — superseded
**TUI v2 (Migration):** TUI-001 through TUI-042 (42 tickets, 42/42 COMPLETE ✅). Feature parity verified.
**Review Telemetry Complete:** GOgent-122 through GOgent-139 (18 tasks, all tests passing)
