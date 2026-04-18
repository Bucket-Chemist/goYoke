# ADR-028: JSONL Handoff Format with Markdown Rendering

**Status**: Accepted

**Date**: 2026-01-19

**Deciders**: goYoke Development Team

**Technical Story**: goYoke-028 series - Session handoff generation migration from bash to Go

---

## Summary

This ADR documents why we chose a dual JSONL + Markdown format for session handoffs over simpler alternatives. The decision balances machine queryability with human readability, enabling both retrospective analysis and immediate session review.

---

## Context

### The Problem We're Solving

Claude Code sessions generate valuable operational data: tool call counts, errors encountered, routing violations, and debugging patterns ("sharp edges"). This data has two audiences:

1. **Humans** who need to quickly review what happened in a session
2. **Machines** that need to query patterns across many sessions (e.g., "What were the most common violations last month?")

The original bash implementation (`~/.claude/hooks/session-archive.sh`) served only the first audience:

```bash
# Original bash approach
cat << EOF > .claude/memory/last-handoff.md
# Session Handoff
- Tool calls: $TOOL_CALLS
- Errors: $ERROR_COUNT
...
EOF
```

**Problems with this approach:**

| Limitation | Impact |
|------------|--------|
| Overwrites each session | Historical data lost; can't answer "how many errors last week?" |
| Markdown-only format | No structured querying; requires regex parsing |
| No schema versioning | Future format changes break consumers |
| No append-only log | Can't build trend analysis or dashboards |

### Requirements

The replacement Go implementation (`goyoke-archive`) must:

1. **Preserve human UX** — Immediate review via markdown must still work
2. **Enable machine queries** — "Show all sessions with >3 violations"
3. **Support historical analysis** — Never lose session data
4. **Handle schema evolution** — Future-proof against format changes
5. **Maintain hook compatibility** — `load-routing-context.sh` reads markdown on session start

### Constraints

- **SessionStart hook** expects markdown at `.claude/memory/last-handoff.md`
- **Go binary** must be single-file, no runtime dependencies
- **Production use** requires zero-downtime migration from bash

---

## Decision

**Implement dual-format handoff generation:**

```
SessionEnd event → goyoke-archive
                        │
                        ▼
              ┌─────────────────┐
              │ GenerateHandoff │
              │    (JSONL)      │
              └────────┬────────┘
                       │
          ┌────────────┴────────────┐
          ▼                         ▼
    Append to                 RenderMarkdown
  handoffs.jsonl             → last-handoff.md
   (source of truth)          (human view)
          │                         │
          ▼                         ▼
  ┌───────────────┐       ┌─────────────────┐
  │ Query history │       │ Immediate review│
  │ via CLI/code  │       │ via SessionStart│
  └───────────────┘       └─────────────────┘
```

### Primary Format: JSONL (`handoffs.jsonl`)

- **Append-only** — One JSON object per line, never overwritten
- **Schema-versioned** — `schema_version` field enables migrations
- **Machine-readable** — Parse with `jq`, Go, Python, etc.
- **Location**: `.claude/memory/handoffs.jsonl`

### Secondary Format: Markdown (`last-handoff.md`)

- **Rendered from JSONL** — Not generated independently (single source of truth)
- **Overwritten each session** — Shows only latest handoff
- **Human-readable** — Formatted tables, sections, action items
- **Location**: `.claude/memory/last-handoff.md`

---

## Schema Specification

### JSONL Schema (v1.0)

```json
{
  "schema_version": "1.0",
  "timestamp": 1737312000,
  "session_id": "abc123def456",
  "context": {
    "project_dir": "/home/user/my-project",
    "metrics": {
      "tool_calls": 42,
      "errors_logged": 5,
      "routing_violations": 3,
      "session_id": "abc123def456"
    },
    "active_ticket": "goYoke-028",
    "phase": "implementation",
    "git_info": {
      "branch": "feature/session-handoff",
      "is_dirty": true,
      "uncommitted": ["pkg/session/handoff.go", "README.md"]
    }
  },
  "artifacts": {
    "sharp_edges": [
      {
        "file": "pkg/routing/schema.go",
        "error_type": "nil_pointer_dereference",
        "consecutive_failures": 3,
        "context": "Schema validation loop",
        "timestamp": 1737311000
      }
    ],
    "routing_violations": [
      {
        "agent": "codebase-search",
        "violation_type": "wrong_subagent_type",
        "expected_tier": "Explore",
        "actual_tier": "general-purpose",
        "timestamp": 1737311500
      }
    ],
    "error_patterns": [
      {
        "error_type": "file_not_found",
        "count": 7,
        "last_seen": 1737311800,
        "context": "Missing fixture files during test"
      }
    ]
  },
  "actions": [
    {
      "priority": 1,
      "description": "Review 1 sharp edge(s) before continuing work",
      "context": "Debugging loops captured - may indicate missing patterns or documentation"
    }
  ]
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema_version` | string | Yes | Enables migration logic when format changes |
| `timestamp` | int64 | Yes | Unix timestamp of handoff generation |
| `session_id` | string | Yes | Claude Code session identifier |
| `context.project_dir` | string | Yes | Absolute path to project root |
| `context.metrics` | object | Yes | Session-level counts |
| `context.active_ticket` | string | No | Current ticket if `/ticket` workflow active |
| `context.phase` | string | No | Workflow phase (discovery, implementation, etc.) |
| `context.git_info` | object | No | Git branch and dirty state |
| `artifacts.sharp_edges` | array | Yes | Debugging loops captured by hooks |
| `artifacts.routing_violations` | array | Yes | Tier/agent routing issues |
| `artifacts.error_patterns` | array | Yes | Aggregated error types |
| `actions` | array | Yes | Prioritized next steps |

### Markdown Sections (Rendered)

The `RenderHandoffMarkdown()` function produces:

1. **Session Context** — ID, project, ticket, phase
2. **Session Metrics** — Tool calls, errors, violations
3. **Git State** — Branch, dirty status, uncommitted files
4. **Sharp Edges** — Debugging loops with context
5. **Routing Violations** — Agent misuse details
6. **Error Patterns** — Recurring error types
7. **Immediate Actions** — Prioritized next steps

---

## Alternatives Considered

### Alternative 1: Markdown-Only (Status Quo)

**Approach**: Continue with bash hook, markdown-only output.

| Aspect | Assessment |
|--------|------------|
| Implementation | Zero effort (already done) |
| Human readability | Excellent |
| Machine queryability | Poor — regex parsing required |
| Historical data | None — overwritten each session |
| Schema evolution | None — format changes break silently |

**Verdict**: Rejected. Cannot answer retrospective questions ("violations last 10 sessions").

### Alternative 2: JSONL-Only

**Approach**: Replace markdown with JSONL, require `jq` for reading.

| Aspect | Assessment |
|--------|------------|
| Implementation | Simple |
| Human readability | Poor — requires `jq` pipeline |
| Machine queryability | Excellent |
| Historical data | Full append-only log |
| Schema evolution | Supported via version field |

**Verdict**: Rejected. Breaks `load-routing-context.sh` which expects markdown. Poor UX for quick session review.

### Alternative 3: SQLite Database

**Approach**: Store sessions in local SQLite database.

| Aspect | Assessment |
|--------|------------|
| Implementation | Complex — requires schema migrations |
| Human readability | None — binary format |
| Machine queryability | Excellent — full SQL |
| Historical data | Full database history |
| Schema evolution | Requires migration scripts |

**Verdict**: Rejected. Overkill for append-only use case. Adds dependency. Breaks hook compatibility.

### Alternative 4: Dual Format (Chosen)

**Approach**: JSONL as source of truth, markdown rendered from JSONL.

| Aspect | Assessment |
|--------|------------|
| Implementation | Moderate — two file writes |
| Human readability | Excellent — markdown preserved |
| Machine queryability | Excellent — JSONL queryable |
| Historical data | Full append-only log |
| Schema evolution | Supported via version field |

**Verdict**: Accepted. Best balance of all requirements.

---

## Implementation Decisions

### CLI Architecture

**Decision**: Single binary `goyoke-archive` with mode-based operation.

```
goyoke-archive                  # Hook mode: read SessionEnd JSON from STDIN
goyoke-archive list             # Query mode: show session history
goyoke-archive list --since 7d  # Filtered query
goyoke-archive show <id>        # Single session detail
goyoke-archive stats            # Aggregate statistics
```

**Why STDIN for hook mode?**

- Claude Code hook infrastructure streams JSON directly (no intermediate files)
- Atomic: All input arrives in single read
- Testable: `echo '{"session_id": "test"}' | goyoke-archive`
- No command-line argument parsing for complex JSON structures

**Why subcommands for query mode?**

- Familiar CLI pattern (git, kubectl, docker)
- Clear separation between "hook" and "interactive" modes
- Extensible for future operations

### Project Directory Detection

**Decision**: `GOYOKE_PROJECT_DIR` environment variable with `pwd` fallback.

```go
projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
if projectDir == "" {
    projectDir, _ = os.Getwd()
}
```

**Why not auto-detect git root?**

- Not all projects are git repos
- Ambiguous with nested repos (submodules)
- Slower than environment variable lookup
- Hook knows project dir, can set explicitly

### Error Handling Format

**Decision**: Structured three-part error messages.

```
[goyoke-archive] Failed to write handoffs.jsonl: permission denied.
  Directory .claude/memory/ is not writable.
  Fix: chmod 755 .claude/memory/ or check disk space.
```

Format: `[component] What. Why. How to fix.`

**Why this structure?**

- **What**: Clear failure description for logs
- **Why**: Root cause for debugging
- **How**: Actionable remediation for operators

**Fatal vs Non-Fatal:**

| Category | Example | Behavior |
|----------|---------|----------|
| Fatal | JSONL write failure | Exit 1, detailed error |
| Non-fatal | Markdown render failure | Log warning, continue |
| Non-fatal | Git info collection failure | Use empty GitInfo{} |

Rationale: JSONL is source of truth. If it fails, session data is lost. Markdown is derived view; missing it is recoverable.

### Testing Strategy

**Decision**: Three-tier testing pyramid.

```
                    ┌─────────────┐
                    │ Compat      │  Bash parity verification
                    │ Tests       │  (metrics match, context loads)
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │   Integration Tests     │  Full workflow: STDIN → JSONL → MD
              │   (real filesystem)     │  Binary invocation via exec.Command
              └────────────┬────────────┘
                           │
    ┌──────────────────────┴──────────────────────┐
    │             Unit Tests                       │  Pure functions, mocked I/O
    │     (RenderMarkdown, GenerateHandoff)        │  ≥80% coverage target
    └──────────────────────────────────────────────┘
```

**Tier 1: Unit Tests** (`pkg/session/*_test.go`)
- Test pure functions in isolation
- Mock filesystem where possible
- Fast feedback loop (<1s)

**Tier 2: Integration Tests** (`test/integration/*_test.go`)
- Full workflow with real filesystem (`t.TempDir()`)
- Binary invocation via `exec.Command`
- Validates file I/O, environment variables, exit codes

**Tier 3: Compatibility Tests** (`test/compatibility/*`)
- Bash hook parity verification
- Metrics alignment (tool calls, errors, violations must match)
- Context loading validation (`load-routing-context.sh` reads Go-generated markdown)

---

## Consequences

### Positive

1. **Queryable History**
   ```bash
   goyoke-archive list --since 7d --has-violations
   goyoke-archive stats
   jq 'select(.artifacts.sharp_edges | length > 0)' handoffs.jsonl
   ```

2. **Schema Evolution**
   - `schema_version` field enables future migrations
   - Old JSONL entries remain readable with versioned parsers
   - `migrateHandoff()` function handles version upgrades

3. **Human UX Preserved**
   - Markdown rendering for immediate review
   - Compatible with existing `load-routing-context.sh` workflow
   - No new tools required for basic usage

4. **Append-Only Integrity**
   - Never lose historical data
   - Supports trend analysis over time
   - Enables anomaly detection ("more violations than usual")

5. **Testability**
   - Schema is documented and versioned
   - Integration tests verify full workflow
   - Compatibility tests prevent bash/Go drift

### Negative

1. **Implementation Complexity**
   - Two file writes per session (JSONL append + markdown overwrite)
   - Must keep RenderMarkdown() in sync with schema changes
   - More code to maintain than bash one-liner

2. **Disk Usage**
   - Append-only JSONL grows unbounded
   - Mitigation: Future `goyoke-archive rotate` command (not yet implemented)
   - Estimate: ~2KB per session × 100 sessions = 200KB/project (negligible)

3. **Potential Inconsistency**
   - If markdown render fails after JSONL write, views diverge
   - Mitigation: JSONL is source of truth; markdown is regenerable
   - Non-fatal error handling prevents data loss

---

## Migration Path

### From Bash to Go

| Phase | Action | Validation |
|-------|--------|------------|
| 1. Deploy | Install `goyoke-archive` binary | Binary runs without error |
| 2. Parallel | Both hooks active, compare outputs | Metrics match between bash and Go |
| 3. Cutover | Disable bash hook, Go only | `load-routing-context.sh` reads Go markdown |
| 4. Cleanup | Remove bash hook | No regressions in session start |

### Backward Compatibility

Old bash-generated `last-handoff.md` files remain readable. They are simply overwritten on next Go session end. No migration of historical bash handoffs to JSONL (data wasn't preserved anyway).

---

## Compliance

| Standard | Alignment |
|----------|-----------|
| XDG Base Directory | Files in project-local `.claude/memory/` |
| Go Conventions | Single binary, no runtime dependencies |
| JSONL Spec | Newline-delimited JSON, one object per line |
| CommonMark | Markdown output is CommonMark-compatible |

---

## References

### Implementation Files

| File | Purpose |
|------|---------|
| `pkg/session/handoff.go` | JSONL generation, schema, migration |
| `pkg/session/handoff_markdown.go` | Markdown rendering |
| `pkg/session/handoff_artifacts.go` | Artifact loading from temp files |
| `pkg/session/archive.go` | Post-handoff artifact archival |
| `cmd/goyoke-archive/main.go` | CLI entry point, subcommands |

### Test Files

| File | Purpose |
|------|---------|
| `pkg/session/handoff_test.go` | Unit tests for JSONL generation |
| `pkg/session/handoff_markdown_test.go` | Unit tests for markdown rendering |
| `test/integration/session_handoff_integration_test.go` | Full workflow integration tests |
| `test/integration/metrics_parity_test.go` | Bash/Go metrics alignment |

### Related Tickets

| Ticket | Description |
|--------|-------------|
| goYoke-028 | Core handoff implementation |
| goYoke-028a | CLI binary structure |
| goYoke-028b | Metrics parity with bash |
| goYoke-028g | Artifact validation |
| goYoke-028l | Schema versioning |
| goYoke-028m | Integration tests |

### Original Bash Implementation

`~/.claude/hooks/session-archive.sh` — Now deprecated, replaced by `goyoke-archive` binary.

---

**Approved**: 2026-01-19
**Review Date**: 2026-04-19 (quarterly review)
