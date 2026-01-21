# Einstein GAP Document: GOgent-029 Series Schema Architecture

**Generated**: 2026-01-21
**Escalated By**: Staff-architect review + user design intent clarification
**Severity**: CRITICAL - Foundation architecture affecting entire learning subsystem

---

## 1. Problem Statement

The GOgent-029 ticket series was designed to implement a learning capture and retrieval system, but:

1. **029 and 029b were not refactored** when scope expanded to 029c-f
2. **Original tickets assume markdown-only output** - inadequate for agentic consumption
3. **Missing user intent capture** - decisions, preferences, responses not architected
4. **No hybrid query architecture** - should be dual JSONL with agentic-optimized schema

### User's Design Intent (Not Currently Reflected)

> "If you are going to go to the effort of formatting the learnings:
> - Why go to the effort of doing this without catching more than simply the sharp edges?
> - And why only markdown? That seems completely retarded when I am engineering a systems-architecture-reviewer on top of it.
> - This needs to be a hybrid system - dual jsonl with a query schema optimised specifically for agentic query in a context lite manner
> - Weekly aggregation → weekly reviews → Once reviewed capture and actioned → ARCHIVE
> - Where am I capturing user decision inputs - how am I storing responses - user intent etc?"

---

## 2. Current State Analysis

### Ticket Dependency Chain

```
029 (Format Pending Learnings) → 029b (Add Error Messages) → 029c (Expand Schema)
                                                                    ↓
                                                           029d (CLI Query)
                                                           029e (Go Query API)
                                                           029f (Weekly Aggregation)
```

### What 029 Currently Does

- Parses `pending-learnings.jsonl`
- Outputs **markdown bullets only**: `- **file**: error_type (N failures)`
- SharpEdge struct with: File, ErrorType, ConsecutiveFailures, LastError, Timestamp

### What 029b Currently Adds

- ErrorMessage field to SharpEdge
- Truncated error display in markdown
- Still **markdown-only output**

### What 029c-f Add (The Expanded Vision)

| Ticket | Contribution | Gap vs User Intent |
|--------|--------------|-------------------|
| 029c | Decision, PreferenceOverride, PerformanceMetric structs | ✅ Captures user decisions |
| 029d | CLI subcommands (decisions, preferences, performance) | ⚠️ Human-facing only |
| 029e | Go Query API with filters | ✅ Agentic query capability |
| 029f | Weekly aggregation + archival | ✅ Solves file bloat |

### Critical Gap

**029 and 029b are orphaned from the expanded architecture:**
- They format to markdown (human-readable)
- They don't produce JSONL output for agentic consumption
- They duplicate SharpEdge definition instead of using extended schema from 029c
- systems-architecture-reviewer cannot efficiently query their output

---

## 3. Architectural Context

### Existing Codebase State

**pkg/session/handoff.go** (lines 46-52):
```go
type SharpEdge struct {
    File               string `json:"file"`
    ErrorType          string `json:"error_type"`
    ConsecutiveFailures int    `json:"consecutive_failures"`
    Context            string `json:"context,omitempty"`
    Timestamp          int64  `json:"timestamp"`
}
```

**pkg/session/handoff_artifacts.go** contains:
- `loadPendingLearnings()` - already parses JSONL
- `LoadArtifacts()` - aggregates multiple JSONL sources

### User's Target Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    DUAL OUTPUT SYSTEM                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Source JSONL Files          Formatted Output                │
│  ─────────────────          ─────────────────               │
│  pending-learnings.jsonl    → markdown (human review)        │
│  decisions.jsonl            → JSONL (agentic query)         │
│  preferences.jsonl                                          │
│  performance.jsonl                                          │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                    QUERY LAYER                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  CLI (Human)                Go API (Agentic)                │
│  ───────────               ────────────────                 │
│  gogent-archive decisions   QueryDecisions(filters)         │
│  gogent-archive performance QueryPerformance(filters)       │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                    LIFECYCLE                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Active JSONL → Weekly Review → Action Items → ARCHIVE      │
│                                                              │
│  systems-architecture-reviewer queries active files         │
│  Weekly aggregation moves to archive/YYYY-Www-*.jsonl      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 4. Attempt Log

### Attempt 1: Original Ticket Design (029, 029b)

**Approach**: Create FormatPendingLearnings() for markdown output
**Result**: Markdown-only, no agentic query support
**Failure Mode**: Designed for human reading, not agent consumption

### Attempt 2: Scope Expansion (029c-f)

**Approach**: Add new artifact types, query APIs, aggregation
**Result**: Good architecture for new artifacts
**Failure Mode**: Did not backport improvements to 029/029b

### Attempt 3: Staff-Architect Review

**Approach**: Identify 029/029b complementarity issues
**Result**: Found sequential modification anti-pattern
**Failure Mode**: Focused on implementation pattern, not architectural gap

---

## 5. Constraints

1. **Backward Compatibility**: Existing SharpEdge struct in handoff.go must be extended, not replaced
2. **Agentic Efficiency**: Output must be queryable without loading full context
3. **Human Reviewability**: Must still produce markdown for manual review workflows
4. **File Size Management**: Cannot allow unbounded JSONL growth
5. **systems-architecture-reviewer Integration**: Must support context-lite agentic queries

---

## 6. File Excerpts

### Current 029 Output Format (markdown only)
```markdown
- **src/main.go**: type_mismatch (3 failures)
- **pkg/utils.go**: nil_pointer (2 failures)
```

### 029c Decision Struct (the model for 029/029b)
```go
type Decision struct {
    Timestamp    int64  `json:"timestamp"`
    Category     string `json:"category"`
    Decision     string `json:"decision"`
    Rationale    string `json:"rationale"`
    Alternatives string `json:"alternatives"`
    Impact       string `json:"impact"`
}
```

### 029e Query API Pattern (should apply to sharp edges too)
```go
func (q *Query) QueryDecisions(filters DecisionFilters) ([]Decision, error)
```

---

## 7. Primary Question

**How should GOgent-029 and GOgent-029b be refactored to:**

1. Align with the dual-output (markdown + JSONL) architecture established in 029c-f?
2. Support agentic query via the same Query API pattern as 029e?
3. Properly capture user intent/decisions beyond just sharp edges?
4. Integrate with the weekly aggregation lifecycle from 029f?

**Sub-questions:**
- Should 029 and 029b be merged?
- What new fields are needed on SharpEdge for parity with Decision/Preference structs?
- Should FormatPendingLearnings() be renamed/refactored to produce both outputs?
- Where does user intent capture fit in the ticket sequence?

---

## 8. Anti-Scope

**DO NOT address:**
- Implementation details of 029c-f (already well-specified)
- Hook system integration (separate concern)
- CLI UX improvements (covered by 029d)
- Performance optimization (premature)

**FOCUS ON:**
- Architectural alignment of 029/029b with 029c-f vision
- Schema unification across all learning artifact types
- Agentic query enablement for sharp edges
- Gap analysis for user intent capture

---

## 9. Success Criteria

A successful analysis will provide:

1. **Refactored ticket specifications** for 029 and 029b (or merged ticket)
2. **Unified schema design** showing how SharpEdge aligns with Decision/Preference
3. **Query API extension** showing QuerySharpEdges() parallel to QueryDecisions()
4. **User intent gap remediation** - where/how to capture decision inputs
5. **Dependency chain update** showing corrected relationships
