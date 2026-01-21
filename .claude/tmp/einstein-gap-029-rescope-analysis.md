# GAP Document: 029 Series Rescoping Analysis

**Generated**: 2026-01-21
**Context**: Einstein escalation for architectural review of 029-029f tickets against 028k-o implementation

---

## 1. Problem Statement

The 029 series tickets were designed before 028k-o implementation was complete. Now that 028k-o has landed with:
- ADR-028: Dual JSONL+Markdown format with schema versioning
- `handoff.go`: Schema v1.0 with `HandoffArtifacts` struct
- `main.go`: CLI with `list/show/stats` subcommands
- Integration tests and deployment runbook

**Critical question**: Do tickets 029-029f need rescoping to align with the implemented architecture?

---

## 2. Context Gathered

### 2.1 Implemented Architecture (028k-o)

**Schema (v1.0)**:
```go
type HandoffArtifacts struct {
    SharpEdges         []SharpEdge       `json:"sharp_edges"`
    RoutingViolations  []RoutingViolation `json:"routing_violations"`
    ErrorPatterns      []ErrorPattern     `json:"error_patterns"`
}

type SharpEdge struct {
    File               string `json:"file"`
    ErrorType          string `json:"error_type"`
    ConsecutiveFailures int   `json:"consecutive_failures"`
    Context            string `json:"context,omitempty"`
    Timestamp          int64  `json:"timestamp"`
}
```

**CLI Pattern**:
```
gogent-archive              # Hook mode (STDIN)
gogent-archive list         # Session-centric listing
gogent-archive show <id>    # Session-centric detail
gogent-archive stats        # Aggregate statistics
```

**File Architecture**:
- Primary: `.claude/memory/handoffs.jsonl` (source of truth, append-only)
- Secondary: `.claude/memory/last-handoff.md` (rendered, overwritten)
- Sources: `pending-learnings.jsonl`, `routing-violations.jsonl` (loaded into HandoffArtifacts)

### 2.2 Proposed Architecture (029 Series)

**GOgent-029**: Extend SharpEdge schema
```go
type SharpEdge struct {
    // Existing fields...
    ErrorMessage string `json:"error_message,omitempty"`  // NEW
    Severity     string `json:"severity,omitempty"`       // NEW
    Resolution   string `json:"resolution,omitempty"`     // NEW
    ResolvedAt   int64  `json:"resolved_at,omitempty"`    // NEW
}
```

**GOgent-029a**: Add UserIntent struct
```go
type UserIntent struct {
    Question   string `json:"question"`
    Response   string `json:"response"`
    Confidence string `json:"confidence"`
    Context    string `json:"context,omitempty"`
    Source     string `json:"source,omitempty"`
    ActionTaken string `json:"action_taken,omitempty"`
    Timestamp  int64  `json:"timestamp"`
}
```

**GOgent-029b/029d**: Add artifact-centric CLI
```
gogent-archive sharp-edges [flags]
gogent-archive user-intents [flags]
gogent-archive decisions [flags]
gogent-archive preferences [flags]
gogent-archive performance [flags]
```

**GOgent-029c**: Extend HandoffArtifacts
```go
type HandoffArtifacts struct {
    SharpEdges         []SharpEdge
    UserIntents        []UserIntent        // NEW
    RoutingViolations  []RoutingViolation
    ErrorPatterns      []ErrorPattern
    Decisions          []Decision          // NEW
    Preferences        []Preference        // NEW
    Performance        []PerformanceMetric // NEW
}
```

**GOgent-029e**: Add Query API (pkg/session/query.go)
**GOgent-029f**: Add aggregation CLI for file rotation

---

## 3. Conflicts Identified

### 3.1 Schema Version Impact

| Change | Schema Impact | Migration Complexity |
|--------|---------------|---------------------|
| Extend SharpEdge (029) | New fields optional → backward compatible | LOW |
| Add UserIntent (029a) | New artifact type → requires migration | MEDIUM |
| Extend HandoffArtifacts (029c) | New array fields → requires migration | MEDIUM |

**Decision Required**:
- Bump to v1.1 (additive, backward-compatible)?
- Or bump to v2.0 (breaking change, requires migration)?

ADR-028 explicitly prepared for this via `migrateHandoff()`.

### 3.2 CLI Pattern Conflict

**Current (028)**: Session-centric
- `list` → shows sessions
- `show <id>` → shows one session
- `stats` → aggregates across sessions

**Proposed (029b/029d)**: Artifact-centric
- `sharp-edges` → queries sharp edges across sessions
- `decisions` → queries decisions across sessions
- etc.

**Conflict**: These are orthogonal patterns. Both valid. Need to decide:
- Option A: Keep both patterns (session + artifact views)
- Option B: Replace session-centric with artifact-centric
- Option C: Hybrid (`gogent-archive list --artifact=sharp-edges`)

### 3.3 File Architecture Tension

**Current**: Artifacts loaded from separate files, aggregated into handoffs.jsonl
- `pending-learnings.jsonl` → SharpEdges
- `routing-violations.jsonl` → RoutingViolations

**Proposed (029a/029c)**: More separate files
- `user-intents.jsonl` → UserIntents
- `decisions.jsonl` → Decisions
- `preferences.jsonl` → Preferences
- `performance.jsonl` → Performance

**Question**: Should these be:
- Embedded in handoffs.jsonl per session (current pattern)?
- OR kept separate with cross-session querying (029 series pattern)?

The 029f aggregation assumes separate files for rotation.

### 3.4 HandoffConfig Extension

Current `HandoffConfig` has:
```go
type HandoffConfig struct {
    ProjectDir        string
    HandoffPath       string // handoffs.jsonl
    PendingPath       string // pending-learnings.jsonl
    ViolationsPath    string // routing-violations.jsonl
    ErrorPatternsPath string // /tmp/claude-error-patterns.jsonl
    TranscriptPath    string // Optional
}
```

029a/029c need:
```go
    UserIntentsPath   string // user-intents.jsonl
    DecisionsPath     string // decisions.jsonl
    PreferencesPath   string // preferences.jsonl
    PerformancePath   string // performance.jsonl
```

This is additive and compatible.

---

## 4. Questions for Einstein

### Q1: Schema Strategy
Given ADR-028's schema versioning infrastructure, should 029 series changes:
- A) Use v1.1 (additive fields only, no migration needed)
- B) Use v2.0 (full migration, clean break)
- C) Something else?

### Q2: CLI Architecture
The CLI now has session-centric commands. 029b/d propose artifact-centric commands. How should these coexist:
- A) Parallel namespaces (`list` sessions + `sharp-edges` artifacts)
- B) Unified namespace (`list --type=sessions|sharp-edges|decisions`)
- C) Artifact commands as subcommands of session (`show <id> --sharp-edges`)

### Q3: Separate vs Embedded Artifacts
Current pattern embeds artifacts into per-session handoffs. 029 pattern assumes separate JSONL files for cross-session queries. Which is correct:
- A) Embed all in handoffs.jsonl (single source of truth per session)
- B) Keep separate files (cross-session queries, weekly rotation)
- C) Hybrid (embed for session view, separate for queries, dual-write)

### Q4: Which 029 Tickets Need Rescoping?

For each ticket, does it conflict with 028k-o or just extend it?

| Ticket | Status | Reason |
|--------|--------|--------|
| 029 | ? | Extends SharpEdge schema |
| 029a | ? | Adds UserIntent struct |
| 029b | ? | Adds CLI subcommands |
| 029c | ? | Extends HandoffArtifacts |
| 029d | ? | Extends CLI further |
| 029e | ? | Adds Query API |
| 029f | ? | Adds aggregation (depends on file structure) |

---

## 5. Anti-Scope

**NOT in scope for this analysis**:
- Rewriting ADR-028 (it's accepted)
- Changing 028k-o implementation (already landed)
- Redesigning the dual JSONL+Markdown pattern
- Hook infrastructure changes

**Focus**: Rescope 029 series to harmonize with landed 028 architecture.

---

## 6. Evidence Files

| File | Relevance |
|------|-----------|
| `docs/decisions/ADR-028-jsonl-handoff-format.md` | Canonical architecture decisions |
| `pkg/session/handoff.go:46-52` | Current SharpEdge struct |
| `pkg/session/handoff.go:38-43` | Current HandoffArtifacts struct |
| `cmd/gogent-archive/main.go:18-44` | Current CLI pattern |
| `migration_plan/tickets/session_archive/029.md` | Proposed SharpEdge extension |
| `migration_plan/tickets/session_archive/029a.md` | Proposed UserIntent |
| `migration_plan/tickets/session_archive/029b.md` | Proposed CLI subcommands |

---

## 7. Constraints

1. ADR-028 is **accepted** - can extend but not contradict
2. 028k-o is **landed** - must build on top of it
3. Schema versioning via `schema_version` field is the migration mechanism
4. `load-routing-context.sh` depends on markdown format
5. Separate JSONL files exist for pending-learnings and routing-violations already

---

**Status**: Ready for Einstein analysis
**Complexity**: HIGH - architectural integration decisions
**Urgency**: MEDIUM - 029 series blocked until rescoped
