# GOgent Fortress

**Programmatically Enforced Agentic Cooperation**

**Status:** Production Ready
**Version:** 1.0.0
**Schema:** routing-schema v2.2.0 | handoff v1.3 | ML telemetry v1.0

---

## Overview

GOgent Fortress is a Go-based hook orchestration framework for Claude Code that enforces tiered routing policies, tracks debugging loops, captures ML telemetry, and maintains session continuity through deterministic validation—not LLM instructions.

**Key Insight:** Enforcement via code, not prompts. Text instructions are probabilistic suggestions; runtime hooks are deterministic rules.

### What We Built

A Go-based hook system that intercepts Claude Code tool events (SessionStart, PreToolUse, PostToolUse, SubagentStop, SessionEnd) and applies programmatic validation:

- **Task Validation**: Blocks invalid model/subagent_type pairings, enforces delegation ceilings
- **Sharp-Edge Detection**: Captures debugging loops after 3+ consecutive failures
- **ML Telemetry**: Logs routing decisions and agent collaborations for optimization
- **Session Continuity**: Structured handoff documents preserve context across sessions
- **Orchestrator Guard**: Prevents premature completion when background tasks are running

---

## Quick Start

```bash
# Build and install
cd ~/Documents/GOgent-Fortress
make build-all
make install

# Run Claude with GOgent-Fortress hooks
goclaude
```

See [INSTALL-GUIDE.md](INSTALL-GUIDE.md) for detailed setup instructions.

---

## Architecture

The complete hook enforcement flow, from session initialization through tool validation to archival:

```mermaid
flowchart TD
    subgraph "SessionStart Hook"
        SS[gogent-load-context] --> LH[Load Previous Handoff]
        LH --> DL[Detect Language]
        DL --> LC[Load Conventions]
        LC --> INJ[Inject Context]
    end

    subgraph "PreToolUse Hook - Task Validation"
        PT[gogent-validate] --> CHK1{Tool == Task?}
        CHK1 -->|No| PASS[Pass Through]
        CHK1 -->|Yes| LOAD[Load routing-schema.json]
        LOAD --> V1{Einstein/Opus Block?}
        V1 -->|Yes| BLK1[BLOCK: Use /einstein]
        V1 -->|No| V2{Model Mismatch?}
        V2 -->|Yes| WARN[WARN: Continue]
        V2 -->|No| V3{Delegation Ceiling?}
        V3 -->|Exceeded| BLK2[BLOCK: Ceiling Violation]
        V3 -->|OK| V4{Subagent Type Valid?}
        V4 -->|No| BLK3[BLOCK: Wrong Type]
        V4 -->|Yes| ALLOW[ALLOW]
        BLK1 --> LOG[Log Violation]
        BLK2 --> LOG
        BLK3 --> LOG
        LOG --> VFILE[(routing-violations.jsonl)]
    end

    subgraph "PostToolUse Hook - Merged Handler"
        PO[gogent-sharp-edge] --> INC[Increment Tool Counter]
        INC --> MLLOG[Log ML Telemetry]
        MLLOG --> CHK10{Every 10 Tools?}
        CHK10 -->|Yes| REM[Inject Routing Reminder]
        CHK10 -->|No| CHK20{Every 20+ Tools?}
        CHK20 -->|Yes| FLUSH[Archive Pending Learnings]
        CHK20 -->|No| DET[Detect Failure]
        REM --> DET
        FLUSH --> DET
        DET --> FAIL{Failure?}
        FAIL -->|Yes| LOGF[Log Failure]
        LOGF --> CNT{3+ Consecutive?}
        CNT -->|Yes| CAP[Capture Sharp Edge]
        CAP --> BLKS[BLOCK + Guidance]
        CNT -->|No| CONT[Continue]
        FAIL -->|No| CONT
        BLKS --> PEND[(pending-learnings.jsonl)]
        LOGF --> TRACK[(failure-tracker.jsonl)]
        MLLOG --> MLFILE[(routing-decisions.jsonl)]
    end

    subgraph "SubagentStop Hook"
        SA[gogent-agent-endstate] --> OUTCOME[Record Decision Outcome]
        OUTCOME --> COLLAB[Log Collaboration]
        COLLAB --> COLLFILE[(agent-collaborations.jsonl)]
    end

    subgraph "SessionEnd Hook - Archive & Handoff"
        SE[gogent-archive] --> METRICS[Collect Metrics]
        METRICS --> ARTF[Load Artifacts]
        ARTF --> GEN[Generate Handoff]
        GEN --> SAVE[Save to Memory]
        SAVE --> HAND[(handoffs.jsonl)]
        SAVE --> LAST[(last-handoff.md)]
    end

    INJ --> |Session Active| PT
    ALLOW --> |Tool Executes| PO
    WARN --> |Tool Executes| PO
    CONT --> |Continue Session| PT
    PO --> |Subagent Completes| SA
    SA --> |Session Ends| SE
    HAND --> |Next Session| SS

    style BLK1 fill:#f96,stroke:#333,stroke-width:2px
    style BLK2 fill:#f96,stroke:#333,stroke-width:2px
    style BLK3 fill:#f96,stroke:#333,stroke-width:2px
    style BLKS fill:#f96,stroke:#333,stroke-width:2px
    style ALLOW fill:#9f9,stroke:#333,stroke-width:2px
    style WARN fill:#ff9,stroke:#333,stroke-width:2px
```

### Hook Binaries

| Event | Binary | Responsibility |
|-------|--------|----------------|
| **SessionStart** | `gogent-load-context` | Language detection, convention loading, handoff restoration, git context |
| **PreToolUse** | `gogent-validate` | Task validation, model checking, delegation ceiling, subagent_type enforcement |
| **PostToolUse** | `gogent-sharp-edge` | Tool counting, routing reminders, failure tracking, ML telemetry, sharp-edge detection |
| **SubagentStop** | `gogent-agent-endstate` | Decision outcomes, collaboration tracking, ML updates |
| **SubagentStop** | `gogent-orchestrator-guard` | Background task collection enforcement |
| **SessionEnd** | `gogent-archive` | Metrics collection, artifact loading, handoff generation |

### Enforcement Guarantees

| Hook | Enforcement Mechanism |
|------|----------------------|
| `gogent-validate` | Blocks Tool execution via `{"decision": "block"}` response |
| `gogent-sharp-edge` | Tool counter + failure log → blocks after 3 consecutive failures |
| `gogent-orchestrator-guard` | Blocks completion when background tasks pending |
| `gogent-load-context` | Injects context before LLM receives prompt |
| `gogent-archive` | Writes structured handoff for next session |

**Why this works:** Hooks run **before/after** the LLM, not inside it. Blocking decisions happen in code, not in token predictions.

---

## Package Structure

```
GOgent-Fortress/
├── cmd/                          # CLI entry points
│   ├── gogent-validate/          # PreToolUse hook
│   ├── gogent-sharp-edge/        # PostToolUse hook (merged)
│   ├── gogent-load-context/      # SessionStart hook
│   ├── gogent-agent-endstate/    # SubagentStop hook
│   ├── gogent-orchestrator-guard/# Orchestrator completion guard
│   ├── gogent-archive/           # SessionEnd hook
│   ├── gogent-ml-export/         # ML telemetry export
│   ├── gogent-aggregate/         # Session statistics
│   ├── gogent-capture-intent/    # Manual intent logging
│   └── gogent-doc-theater/       # Documentation theater detection
├── pkg/                          # Core packages
│   ├── routing/                  # Schema validation, violations
│   ├── session/                  # Handoffs, metrics, artifacts
│   ├── memory/                   # Failure tracking, sharp edges
│   ├── telemetry/                # ML telemetry, cost tracking
│   ├── config/                   # Path resolution, XDG compliance
│   ├── workflow/                 # Orchestrator guard logic
│   └── enforcement/              # Validation orchestration
├── .claude/                      # Claude Code configuration
│   ├── CLAUDE.md                 # Router instructions
│   ├── routing-schema.json       # Source of truth
│   ├── settings.json             # Hook configuration
│   ├── agents/                   # Agent definitions
│   ├── conventions/              # Language conventions
│   ├── rules/                    # Behavioral guidelines
│   └── skills/                   # Slash commands
└── test/
    ├── simulation/               # Deterministic fixtures
    └── integration/              # Full lifecycle tests
```

---

## ML Telemetry System

GOgent-Fortress captures routing decisions and agent collaborations for optimization analysis.

### Data Captured

| Data Type | Written By | Location |
|-----------|------------|----------|
| Routing Decisions | `gogent-sharp-edge` | `$XDG_DATA_HOME/gogent-fortress/routing-decisions.jsonl` |
| Decision Outcomes | `gogent-agent-endstate` | `$XDG_DATA_HOME/gogent-fortress/routing-decision-updates.jsonl` |
| Agent Collaborations | `gogent-agent-endstate` | `$XDG_DATA_HOME/gogent-fortress/agent-collaborations.jsonl` |
| Collaboration Outcomes | `gogent-agent-endstate` | `$XDG_DATA_HOME/gogent-fortress/agent-collaboration-updates.jsonl` |

### Append-Only Pattern

Initial records are written immediately; outcomes are appended separately (no file rewrites). ML export reconciles at read time:

```go
// Read-time join for training data
decisions := readJSONL("routing-decisions.jsonl")
updates := readJSONL("routing-decision-updates.jsonl")
for _, update := range updates {
    decisions[update.DecisionID].Outcome = update
}
exportTrainingData(decisions)
```

### Export Utilities

```bash
# Export routing decisions with reconciled outcomes
gogent-ml-export routing-decisions --output=decisions.jsonl

# Export agent collaborations
gogent-ml-export agent-collaborations --output=collabs.jsonl

# Generate summary statistics
gogent-ml-export stats

# Validate data consistency
gogent-ml-export validate --check=orphaned-updates
```

---

## Installation & Usage

### Prerequisites

- Go 1.21+
- Claude Code CLI installed
- `~/.local/bin` in PATH

### Build & Install

```bash
cd ~/Documents/GOgent-Fortress

# Build all binaries
make build-all

# Install to ~/.local/bin
make install

# Verify installation
which gogent-validate gogent-load-context gogent-sharp-edge gogent-archive
```

### Running with GOgent-Fortress

```bash
# Use the goclaude wrapper (recommended)
goclaude

# Or with arguments
goclaude -p "Explain this codebase"
```

The `goclaude` command:
- Verifies all binaries are installed
- Ensures `~/.claude` symlink points to repo config
- Launches Claude with GOgent-Fortress hooks active

### Hook Configuration

Hooks are configured in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {"type": "command", "command": "gogent-load-context", "timeout": 10}
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          {"type": "command", "command": "gogent-validate", "timeout": 10}
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|Task",
        "hooks": [
          {"type": "command", "command": "gogent-sharp-edge", "timeout": 5}
        ]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [
          {"type": "command", "command": "gogent-agent-endstate", "timeout": 15},
          {"type": "command", "command": "gogent-orchestrator-guard", "timeout": 10}
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {"type": "command", "command": "gogent-archive", "timeout": 30}
        ]
      }
    ]
  }
}
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test ./... -cover

# Run with race detector
go test -race ./...

# Run simulation suite
./test/simulation/harness sessionstart-suite
```

---

## Data Persistence

All session data stored in JSONL (JSON Lines) format for append-only writes and streaming reads.

### File Locations

```
Project/
├── .claude/
│   ├── memory/
│   │   ├── handoffs.jsonl          # Session history
│   │   ├── user-intents.jsonl      # User preferences
│   │   ├── decisions.jsonl         # Architectural decisions
│   │   ├── preferences.jsonl       # Preference overrides
│   │   ├── performance.jsonl       # Performance metrics
│   │   ├── pending-learnings.jsonl # Unreviewed sharp edges
│   │   └── last-handoff.md         # Human-readable summary
│   └── session-archive/            # Archived session data

~/.gogent/
├── failure-tracker.jsonl           # Cross-session failure tracking
├── agent-invocations.jsonl         # Invocation telemetry
├── escalations.jsonl               # Tier escalations
└── scout-recommendations.jsonl     # Scout accuracy data

$XDG_DATA_HOME/gogent-fortress/     # ML Telemetry (default: ~/.local/share)
├── routing-decisions.jsonl         # ML training data (initial)
├── routing-decision-updates.jsonl  # ML training data (outcomes)
├── agent-collaborations.jsonl      # Team patterns (initial)
└── agent-collaboration-updates.jsonl # Team patterns (outcomes)

/tmp/
├── claude-routing-violations.jsonl # Current session violations
└── claude-tool-counter-*.log       # Tool call counters
```

### Schema Versions

| Schema | Version | Key Features |
|--------|---------|--------------|
| routing-schema.json | 2.2.0 | agent_subagent_mapping, delegation_ceiling, GO agents |
| Handoff | 1.3 | Extended SharpEdge, decisions, preferences, agent_endstates |
| ML Telemetry | 1.0 | Append-only pattern, read-time reconciliation |

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOGENT_PROJECT_DIR` | `$PWD` | Project root |
| `GOGENT_ROUTING_SCHEMA` | `~/.claude/routing-schema.json` | Schema path override |
| `GOGENT_STORAGE_PATH` | `~/.gogent/failure-tracker.jsonl` | Failure tracker path |
| `GOGENT_MAX_FAILURES` | 3 | Sharp-edge threshold |
| `GOGENT_REMINDER_THRESHOLD` | 10 | Routing reminder frequency |
| `GOGENT_FLUSH_THRESHOLD` | 20 | Auto-flush frequency |
| `XDG_DATA_HOME` | `~/.local/share` | ML telemetry base path |

---

## CLI Reference

### Hook Binaries

| Binary | Event | Purpose |
|--------|-------|---------|
| `gogent-load-context` | SessionStart | Context injection |
| `gogent-validate` | PreToolUse | Task validation |
| `gogent-sharp-edge` | PostToolUse | Failure tracking + ML telemetry |
| `gogent-agent-endstate` | SubagentStop | Outcome recording |
| `gogent-orchestrator-guard` | SubagentStop | Background task enforcement |
| `gogent-archive` | SessionEnd | Handoff generation |

### Utility Binaries

| Binary | Purpose |
|--------|---------|
| `gogent-ml-export` | Export ML training data |
| `gogent-aggregate` | Session statistics |
| `gogent-capture-intent` | Manual intent logging |
| `gogent-doc-theater` | Documentation theater detection |

### Archive Query Subcommands

```bash
gogent-archive list [--since=DATE] [--has-sharp-edges]
gogent-archive show <session_id>
gogent-archive stats
gogent-archive sharp-edges [--file=PATTERN] [--status=pending]
gogent-archive user-intents [--category=CATEGORY]
gogent-archive decisions [--since=DATE]
gogent-archive preferences
gogent-archive performance
```

---

## Testing Infrastructure

### Test Coverage

| Package | Coverage | Key Functions Tested |
|---------|----------|---------------------|
| `pkg/routing` | ~88% | Schema loading, Task validation, violation logging |
| `pkg/session` | ~85% | Handoff generation, language detection, context injection |
| `pkg/memory` | ~82% | Failure tracking, sharp-edge detection |
| `pkg/telemetry` | ~80% | ML logging, cost calculation, collaboration tracking |
| `pkg/workflow` | ~82% | Transcript analysis, background task detection |

### Simulation Harness

**Location:** `test/simulation/harness`

```bash
# Run single fixture
./harness sessionstart 01_home_startup.json

# Run full suite
./harness sessionstart-suite

# Run with verbose output
./harness sessionstart 03_go_startup.json --verbose
```

### GitHub Actions

Three-tier CI/CD workflow:
1. **Unit tests** - Fast feedback on package changes
2. **Simulation tests** - Validates all deterministic fixtures
3. **Integration tests** - Full Claude Code CLI lifecycle

---

## Documentation

| Document | Purpose |
|----------|---------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Complete system architecture with diagrams |
| [INSTALL-GUIDE.md](INSTALL-GUIDE.md) | Step-by-step installation instructions |
| `.claude/CLAUDE.md` | Router instructions for Claude |
| `.claude/routing-schema.json` | Source of truth for routing rules |
| `.claude/agents/agents-index.json` | Agent definitions with triggers |

---

## Development Workflow

### Standards

- **Error Messages:** `[component] What happened. Why it was blocked/failed. How to fix.`
- **XDG Compliance:** Use `$XDG_DATA_HOME`, `$XDG_RUNTIME_DIR`, never hardcoded paths
- **STDIN Timeout:** All hooks implement 5-second timeout on STDIN reads
- **Test Coverage:** Maintain ≥80% coverage per package
- **Append-Only Writes:** Never rewrite JSONL files; use dual-file reconciliation

### Commit Format

```
GOgent-XXX: Title

- Implementation detail 1
- Implementation detail 2
- Test coverage: XX%

Co-Authored-By: Claude <model>@anthropic.com
```

### Workflow

1. Create branch: `gogent-XXX-description`
2. Implement with tests: `go test ./...`
3. Verify coverage: `go test ./... -cover`
4. Run race detector: `go test -race ./...`
5. CI validates: unit → simulation → integration
6. Merge to master

---

## License

Copyright 2025 William Klare. All rights reserved.

This software and its associated documentation, architecture, and design are proprietary. This work was done entirely on my own initiative, finances, and spare time without any form of compensation. No license is granted to use, copy, modify, or distribute any part of this codebase without explicit written permission from the author.

---

## Project Status

**Status:** Production Ready
**Version:** 1.0.0
**Implementation:** GOgent-000 through GOgent-112+
**All hooks:** Complete and operational
**ML Telemetry:** Implemented with append-only pattern
