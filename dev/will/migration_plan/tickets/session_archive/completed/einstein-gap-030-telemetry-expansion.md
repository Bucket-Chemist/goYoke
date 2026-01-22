# Einstein GAP Document: Agent Performance Telemetry Expansion

> **Generated:** 2026-01-22T14:30:00Z
> **Escalated By:** User (proactive architecture review)
> **Session:** Performance monitoring scope analysis
> **Ticket Series:** GOgent-030e through GOgent-030k

---

## 1. Problem Statement

### What We're Trying to Achieve

Transform GOgent from a **reactive violation logger** into a **comprehensive agent performance observatory** that captures:

1. **Positive telemetry** - What succeeded, not just what failed
2. **Cost attribution** - Where money is being spent across agent tiers
3. **Escalation tracking** - Einstein/Gemini usage patterns and ROI
4. **Scout compliance** - Whether routing recommendations are followed

The end goal is a TUI performance dashboard submenu that provides actionable insights for LLM agent orchestration optimization.

### Why This Escalated

- [x] Architectural decision required
- [x] Cross-domain synthesis needed
- [ ] 3+ consecutive failures on same task
- [ ] Complexity exceeds Sonnet tier
- [ ] User explicitly requested deep analysis

**Specific Trigger:** User identified that tickets 30-30d capture only ~40% of observable signals. Einstein analysis confirmed significant observability gaps that would limit TUI dashboard utility.

---

## 2. Analysis Summary

### Current State (Tickets 30-30d)

| Ticket | Capability | TUI Contribution |
|--------|------------|------------------|
| 030 | FormatViolationsSummary() | Basic list view |
| 030b | ClusterViolationsByType() | Pattern aggregation |
| 030c | ClusterViolationsByAgent() | Agent-centric debugging |
| 030d | AnalyzeViolationTrend() | Temporal learning feedback |

**Gap Assessment:** These tickets process `routing-violations.jsonl` comprehensively but ignore:
- Agent invocation successes (no positive telemetry)
- Token/cost tracking (no cost attribution)
- Escalation events (no Einstein/Gemini visibility)
- Scout recommendations (no routing compliance tracking)

### Identified Observability Gaps

| Gap | Impact | Priority |
|-----|--------|----------|
| No invocation logging | Cannot calculate success rates, latency, or cost | **CRITICAL** |
| No escalation tracking | Cannot measure Einstein ROI or identify bottlenecks | **HIGH** |
| No scout compliance | Cannot validate routing recommendations | **MEDIUM** |
| No delegation chain tracking | Cannot trace orchestrator cascades | **MEDIUM** |

---

## 3. Proposed Architecture

### New JSONL Telemetry Files

```
~/.cache/gogent/                          # Global (survives project deletion)
├── agent-invocations.jsonl               # NEW: All agent invocations
├── routing-violations.jsonl              # EXISTING: Violations only
├── escalations.jsonl                     # NEW: Escalation events
└── scout-recommendations.jsonl           # NEW: Scout compliance

<project>/.claude/memory/                 # Project-scoped
├── agent-invocations.jsonl               # NEW: Mirror
├── escalations.jsonl                     # NEW: Mirror
└── scout-recommendations.jsonl           # NEW: Mirror
```

### Schema Definitions

#### AgentInvocation (NEW)
```go
type AgentInvocation struct {
    Timestamp       int64    `json:"timestamp"`
    SessionID       string   `json:"session_id"`
    InvocationID    string   `json:"invocation_id"`       // Unique per invocation
    Agent           string   `json:"agent"`
    Model           string   `json:"model"`               // haiku, sonnet, opus
    Tier            string   `json:"tier"`                // haiku, haiku_thinking, sonnet, opus, external
    DurationMs      int64    `json:"duration_ms"`
    InputTokens     int      `json:"input_tokens"`
    OutputTokens    int      `json:"output_tokens"`
    ThinkingTokens  int      `json:"thinking_tokens,omitempty"`
    Success         bool     `json:"success"`
    ErrorType       string   `json:"error_type,omitempty"` // If !success
    TaskDescription string   `json:"task_description"`     // First 200 chars
    ParentTaskID    string   `json:"parent_task_id,omitempty"` // For delegation chains
    ToolsUsed       []string `json:"tools_used"`
    ProjectDir      string   `json:"project_dir,omitempty"`
}
```

#### EscalationEvent (NEW)
```go
type EscalationEvent struct {
    Timestamp       int64  `json:"timestamp"`
    SessionID       string `json:"session_id"`
    EscalationID    string `json:"escalation_id"`
    FromTier        string `json:"from_tier"`        // "sonnet"
    ToTier          string `json:"to_tier"`          // "opus"
    FromAgent       string `json:"from_agent"`       // "orchestrator"
    ToAgent         string `json:"to_agent"`         // "einstein"
    Reason          string `json:"reason"`           // "3 consecutive failures"
    TriggerType     string `json:"trigger_type"`     // "failure_cascade", "user_request", "complexity"
    GAPDocPath      string `json:"gap_doc_path,omitempty"`
    Outcome         string `json:"outcome"`          // "resolved", "still_blocked", "pending"
    ResolutionTimeMs int64 `json:"resolution_time_ms,omitempty"`
    TokensUsed      int    `json:"tokens_used,omitempty"`
    ProjectDir      string `json:"project_dir,omitempty"`
}
```

#### ScoutRecommendation (NEW)
```go
type ScoutRecommendation struct {
    Timestamp         int64   `json:"timestamp"`
    SessionID         string  `json:"session_id"`
    RecommendationID  string  `json:"recommendation_id"`
    TaskDescription   string  `json:"task_description"`
    ScoutType         string  `json:"scout_type"`          // "haiku-scout", "gemini-scout"
    RecommendedTier   string  `json:"recommended_tier"`
    ActualTier        string  `json:"actual_tier"`
    Followed          bool    `json:"followed"`
    FollowedReason    string  `json:"followed_reason,omitempty"` // Why deviated if !followed
    ScopeMetrics      ScopeMetrics `json:"scope_metrics"`
    Confidence        float64 `json:"confidence"`          // 0.0-1.0
    ProjectDir        string  `json:"project_dir,omitempty"`
}

type ScopeMetrics struct {
    TotalFiles      int `json:"total_files"`
    TotalLines      int `json:"total_lines"`
    EstimatedTokens int `json:"estimated_tokens"`
}
```

### Hook Integration Points

| Signal | Hook | Location |
|--------|------|----------|
| Agent invocations | `agent-endstate` | PostToolUse when subagent completes |
| Violations | `validate-routing` | PreToolUse (already implemented) |
| Escalations | New or instrument `/einstein` | When GAP doc generated or resolved |
| Scout recommendations | `validate-routing` | After scout Task() returns |

### Dual-Write Pattern

All new telemetry follows existing `LogViolation()` pattern:
1. Write to global XDG cache (primary, required)
2. Write to project memory (secondary, optional, graceful degradation)

---

## 4. Ticket Series: GOgent-030e through GOgent-030k

### Dependency Graph

```
030e ─┬─→ 030f ─┬─→ 030g
      │         │
030h ─┴─→ 030i  │
                │
030j ─────→ 030k ─┘
```

### Ticket Summary

| Ticket | Title | Hours | Priority |
|--------|-------|-------|----------|
| 030e | AgentInvocation Schema and Logging | 2.0 | CRITICAL |
| 030f | ClusterInvocationsByAgent Analysis | 1.5 | CRITICAL |
| 030g | CostAttribution Calculation | 1.5 | CRITICAL |
| 030h | EscalationEvent Schema and Logging | 1.5 | HIGH |
| 030i | EscalationPattern Analysis | 1.0 | HIGH |
| 030j | ScoutRecommendation Schema and Logging | 1.5 | MEDIUM |
| 030k | ScoutAccuracy Calculation | 1.0 | MEDIUM |

**Total Estimated Hours:** 10.0 hours

### Critical Path

**030e → 030f → 030g** is the critical path for TUI dashboard viability:
- Without invocation logging (030e), no positive telemetry exists
- Without clustering (030f), no agent leaderboard possible
- Without cost attribution (030g), no spend tracking possible

---

## 5. TUI Integration Strategy (Separate Scope)

The TUI performance submenu is **explicitly out of scope** for the 030e-030k series.

### TUI Tickets (Future, Dependent on 030 Series)

| Ticket | Title | Dependencies |
|--------|-------|--------------|
| TUI-PERF-01 | Performance Dashboard View | 030e-030k complete |
| TUI-PERF-02 | Agent Leaderboard Component | 030f, 030g |
| TUI-PERF-03 | Violation Trend Graph | 030d |
| TUI-PERF-04 | Cost Attribution Panel | 030g |
| TUI-PERF-05 | Escalation Timeline | 030h, 030i |
| TUI-PERF-06 | Scout Compliance Summary | 030j, 030k |

### Why TUI is Good First Test

User observation is correct: TUI tickets make excellent integration tests because:
1. They consume all telemetry APIs (validates 030e-030k)
2. They visualize agent behavior (self-documenting)
3. They provide immediate feedback on schema design
4. They scaffold future features (alerts, reports, exports)

---

## 6. Constraints

- **Backward Compatibility:** All new JSONL files must handle missing file gracefully (empty arrays, not errors)
- **Dual-Write Pattern:** Global and project-scoped logging must both work
- **XDG Compliance:** No hardcoded paths, follow existing `config.GetViolationsLogPath()` pattern
- **Test Coverage:** ≥80% for all new code
- **Ecosystem Tests:** `make test-ecosystem` must pass before any ticket completion
- **Schema Version:** Handoff schema stays at 1.1 (new files are separate, not embedded in handoff)

---

## 7. Success Criteria

### 030e-030k Series Complete When:

- [ ] All 7 tickets implemented and passing tests
- [ ] Three new JSONL files logging correctly (invocations, escalations, scout)
- [ ] Dual-write pattern working for all new telemetry
- [ ] Query functions exist for filtering/aggregating all new types
- [ ] Cost attribution calculates dollars from tokens
- [ ] `make test-ecosystem` passes

### TUI Ready When:

- [ ] 030e-030k complete
- [ ] Go Query API working for all artifact types
- [ ] Bubble Tea components can consume query results
- [ ] Performance submenu renders with real data

---

## 8. Anti-Scope

The 030e-030k series should NOT:

- Implement any TUI components (separate ticket series)
- Modify existing violation schema (additive only)
- Change hook architecture (use existing hooks where possible)
- Add external dependencies (stdlib only for telemetry)
- Implement alerting or notifications (future scope)
- Add real-time streaming (batch JSONL is sufficient for v1)

---

## 9. Files Referenced

| Path | Purpose | Lines |
|------|---------|-------|
| `pkg/routing/violations.go` | Existing violation schema and logging | 106 |
| `pkg/session/handoff.go` | Handoff generation, artifacts loading | 477 |
| `pkg/session/handoff_artifacts.go` | Artifact type definitions, loaders | 460 |
| `pkg/session/query.go` | Query API for artifacts | ~400 |
| `pkg/config/paths.go` | XDG path utilities | ~100 |
| `~/.claude/routing-schema.json` | Tier definitions, cost rates | 343 |

---

## 10. Metadata

```yaml
gap_id: einstein-gap-030-telemetry-expansion
complexity_score: 7/10
estimated_total_hours: 10.0
ticket_count: 7
files_to_create: 3
files_to_modify: 4
created_at: 2026-01-22T14:30:00Z
series: GOgent-030e through GOgent-030k
```
