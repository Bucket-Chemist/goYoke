# TUI Performance Submenu - Ticket Index

> **Status:** Pending (depends on GOgent-030e through GOgent-030k)
> **Estimated Total Hours:** 8-12 hours
> **Purpose:** First integration test of agent performance telemetry

---

## Overview

The TUI Performance Submenu is the first **real user interface** for the agent telemetry system. It serves dual purposes:

1. **Integration Test**: Validates that all telemetry APIs (030e-030k) work correctly together
2. **User Value**: Provides actionable performance insights for agent orchestration

This is an ideal "first test" because:
- Consumes all telemetry types (invocations, escalations, scout recommendations)
- Visualizes agent behavior (self-documenting)
- Provides immediate feedback on schema design
- Scaffolds future features (alerts, reports, exports)

---

## Prerequisites

**ALL of these must be complete before starting TUI tickets:**

| Prerequisite | Ticket | Status |
|--------------|--------|--------|
| AgentInvocation logging | GOgent-030e | Pending |
| Invocation clustering | GOgent-030f | Pending |
| Cost attribution | GOgent-030g | Pending |
| Escalation logging | GOgent-030h | Pending |
| Escalation analysis | GOgent-030i | Pending |
| Scout logging | GOgent-030j | Pending |
| Scout accuracy | GOgent-030k | Pending |

---

## TUI Ticket Series

### TUI-PERF-01: Performance Dashboard Shell

**Estimated Hours:** 2.0
**Dependencies:** GOgent-030k complete

**Description:**
Create the main performance dashboard view with navigation structure.

**Key Features:**
- Bubble Tea model for performance submenu
- Tab navigation between views (Violations, Agents, Cost, Escalations, Scout)
- Session selector (current, today, week, all-time)
- Help overlay (keyboard shortcuts)

**Files:**
- `internal/tui/performance/dashboard.go`
- `internal/tui/performance/model.go`

**Acceptance Criteria:**
- [ ] Dashboard shell renders with tab bar
- [ ] Tab navigation works (←/→ or number keys)
- [ ] Session selector filters data
- [ ] Help overlay toggles with `?`
- [ ] Exit returns to main TUI

---

### TUI-PERF-02: Violations Summary View

**Estimated Hours:** 1.5
**Dependencies:** GOgent-030d (existing), TUI-PERF-01

**Description:**
Display violation summary using existing 030d functions.

**Key Features:**
- Clustered violations by type (030b)
- Clustered violations by agent (030c)
- Trend analysis display (030d)
- Drill-down on selection

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Routing Violations (12 total)                           │
├─────────────────────────────────────────────────────────┤
│ By Type:                          │ By Agent:           │
│ ├─ subagent_type_mismatch (5)     │ ├─ python-pro (5)   │
│ ├─ delegation_ceiling (4)         │ ├─ tech-docs (4)    │
│ └─ tool_permission (3)            │ ├─ scaffolder (2)   │
│                                   │ └─ search (1)       │
├─────────────────────────────────────────────────────────┤
│ Trend: ✅ Improving (66% reduction in second half)      │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/violations.go`

**Acceptance Criteria:**
- [ ] Type clusters render correctly
- [ ] Agent clusters render correctly
- [ ] Trend message displays with appropriate emoji
- [ ] Selection highlights work
- [ ] Enter drills down to details

---

### TUI-PERF-03: Agent Leaderboard View

**Estimated Hours:** 2.0
**Dependencies:** GOgent-030f, TUI-PERF-01

**Description:**
Display agent usage statistics with sortable columns.

**Key Features:**
- Table of agents with: Count, Success Rate, Avg Latency, Total Tokens
- Sortable columns (click header or press S)
- Highlight agents with high error rates
- Sparkline for recent activity (optional)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Agent Leaderboard (sorted by usage)                     │
├─────────────────────────────────────────────────────────┤
│ Agent            Count   Success   Latency   Tokens     │
│ ───────────────────────────────────────────────────────│
│ python-pro       47      93.6%     1.2s      52,340     │
│ orchestrator     23      100.0%    2.8s      89,210     │
│ haiku-scout      18      100.0%    0.3s      8,420      │
│ tech-docs        12      83.3%     1.5s      23,100     │
│ codebase-search  8       100.0%    0.2s      3,200      │
│ ⚠️ scaffolder     5       60.0%     1.8s      12,500     │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/agents.go`

**Acceptance Criteria:**
- [ ] Agent table renders with all columns
- [ ] Sorting works on all columns
- [ ] Low success rate agents highlighted
- [ ] Scroll works for long lists
- [ ] Enter shows agent details

---

### TUI-PERF-04: Cost Attribution Panel

**Estimated Hours:** 1.5
**Dependencies:** GOgent-030g, TUI-PERF-01

**Description:**
Display cost breakdown by tier and agent.

**Key Features:**
- Total session cost prominently displayed
- Pie chart or bar representation of tier distribution
- Top 5 most expensive agents
- Cost trend (if historical data available)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Session Cost: $2.34                                     │
├─────────────────────────────────────────────────────────┤
│ By Tier:                          │ Top Agents:         │
│ ████████░░░░░░░░░░ sonnet  $1.82  │ orchestrator $0.89  │
│ ██░░░░░░░░░░░░░░░░ opus    $0.41  │ einstein     $0.41  │
│ █░░░░░░░░░░░░░░░░░ haiku   $0.11  │ python-pro   $0.38  │
│                           (78%)   │ architect    $0.32  │
│                                   │ tech-docs    $0.18  │
├─────────────────────────────────────────────────────────┤
│ Avg cost/invocation: $0.023                             │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/cost.go`

**Acceptance Criteria:**
- [ ] Total cost displays correctly
- [ ] Tier breakdown renders as bar chart
- [ ] Top agents by cost listed
- [ ] Cost formatting handles small amounts ($0.0012)
- [ ] Press D for detailed breakdown

---

### TUI-PERF-05: Escalation Timeline View

**Estimated Hours:** 1.5
**Dependencies:** GOgent-030i, TUI-PERF-01

**Description:**
Display escalation events in timeline format with ROI metrics.

**Key Features:**
- Timeline of escalation events
- Resolution status indicators (✅ resolved, ⏳ pending, ❌ blocked)
- ROI summary (resolution rate, avg resolution time)
- Trigger type breakdown

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Escalations (3 total) - Resolution Rate: 66%            │
├─────────────────────────────────────────────────────────┤
│ Timeline:                                               │
│ 10:15 ✅ orchestrator → einstein (failure_cascade)      │
│        └─ Resolved in 45s: "Fixed validation approach"  │
│ 11:30 ⏳ python-pro → einstein (complexity)             │
│        └─ Pending...                                    │
│ 12:45 ❌ architect → einstein (failure_cascade)         │
│        └─ Still blocked after 3 attempts                │
├─────────────────────────────────────────────────────────┤
│ By Trigger: failure_cascade (2) | complexity (1)        │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/escalations.go`

**Acceptance Criteria:**
- [ ] Timeline renders chronologically
- [ ] Status icons display correctly
- [ ] Resolution summaries show when available
- [ ] ROI metrics calculate correctly
- [ ] Enter shows full escalation details

---

### TUI-PERF-06: Scout Compliance View

**Estimated Hours:** 1.5
**Dependencies:** GOgent-030k, TUI-PERF-01

**Description:**
Display scout recommendation compliance and accuracy metrics.

**Key Features:**
- Compliance rate (followed vs ignored)
- Accuracy comparison (followed vs ignored outcomes)
- Confidence correlation buckets
- Per-scout-type breakdown

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Scout Performance                                       │
├─────────────────────────────────────────────────────────┤
│ Compliance: 78% (14/18 recommendations followed)        │
│                                                         │
│ Accuracy:                                               │
│ ├─ When followed:  92% success (12/13)                  │
│ └─ When ignored:   60% success (3/5)                    │
│                                                         │
│ Recommendation: ✅ Follow scout advice                  │
├─────────────────────────────────────────────────────────┤
│ Confidence → Success:                                   │
│ 0.8-1.0: ████████ 95%                                   │
│ 0.6-0.8: ██████░░ 75%                                   │
│ 0.4-0.6: ████░░░░ 50%                                   │
│ 0.0-0.4: ██░░░░░░ 25%                                   │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/scout.go`

**Acceptance Criteria:**
- [ ] Compliance rate displays correctly
- [ ] Accuracy comparison shows delta
- [ ] Recommendation verdict displays
- [ ] Confidence buckets render as bars
- [ ] Statistical significance warning if insufficient data

---

### TUI-PERF-07: Drill-Down Detail View

**Estimated Hours:** 1.0
**Dependencies:** TUI-PERF-02 through TUI-PERF-06

**Description:**
Shared detail view component for drill-down from any summary.

**Key Features:**
- Full JSON view of selected item
- Related items (e.g., invocation → violations it caused)
- Copy to clipboard
- Export to file

**Files:**
- `internal/tui/performance/detail.go`

**Acceptance Criteria:**
- [ ] Detail view renders full object
- [ ] Related items linked
- [ ] Copy works (C key)
- [ ] Export works (E key)
- [ ] Back returns to previous view

---

## Dependency Graph

```
030e ─┬─→ 030f ─┬─→ 030g ──────────────────────┐
      │         │                              │
030h ─┴─→ 030i  │                              │
                │                              ↓
030j ─────→ 030k ────→ TUI-PERF-01 ──────→ TUI-PERF-02
                              │               │
                              ├─→ TUI-PERF-03 ─┤
                              ├─→ TUI-PERF-04 ─┤
                              ├─→ TUI-PERF-05 ─┤
                              └─→ TUI-PERF-06 ─┘
                                      │
                                      ↓
                               TUI-PERF-07
```

---

## Implementation Strategy

### Phase 1: Foundation (Current - 030 Series)
Complete all telemetry tickets (030e-030k) before starting any TUI work.

### Phase 2: Shell (TUI-PERF-01)
Build the dashboard shell with navigation but mock data. This validates the UI structure.

### Phase 3: Views (TUI-PERF-02 through 06)
Implement views in dependency order. Each view is independently testable.

### Phase 4: Integration (TUI-PERF-07)
Add drill-down and cross-view navigation.

### Phase 5: Polish
- Keyboard shortcuts documentation
- Color themes
- Performance optimization for large datasets
- Export functionality

---

## Testing Strategy

Each TUI component should have:

1. **Unit tests** for data transformation (e.g., `formatAgentStats()`)
2. **Snapshot tests** for view rendering (using Bubble Tea's testing utilities)
3. **Integration tests** that load real telemetry and verify display

Example test structure:
```go
func TestAgentLeaderboardView_Rendering(t *testing.T) {
    // Load test invocations
    invocations := loadTestInvocations(t, "fixtures/invocations.jsonl")

    // Create model
    model := NewAgentLeaderboardModel(invocations)

    // Render
    output := model.View()

    // Verify content
    if !strings.Contains(output, "python-pro") {
        t.Error("Expected python-pro in output")
    }
}
```

---

## Why TUI First?

You identified this correctly: the TUI is an excellent first integration test because:

1. **Validates Schemas**: If the TUI can display data, schemas are correct
2. **Validates APIs**: If the TUI can query/filter, APIs work
3. **Validates Aggregation**: If stats display correctly, math is right
4. **Provides Feedback**: Visual output immediately shows problems
5. **User Value**: Delivers something useful while testing

This is the "eat your own dog food" approach - the TUI is both the test and the product.

---

## Notes for Implementation

- Use Bubble Tea's `lipgloss` for styling
- Consider `bubbletea-components` library for tables
- Keep views stateless where possible (re-render from data)
- Use channels for async data loading
- Cache telemetry data, don't re-read on every keystroke

---

**Document Version:** 1.0
**Created:** 2026-01-22
**Last Updated:** 2026-01-22
**Related GAP:** einstein-gap-030-telemetry-expansion.md
