# TUI Development - Complete Ticket Index

> **Status:** Ready for Implementation
> **Estimated Total Hours:** 29 hours
> **Purpose:** GOgent-Fortress Terminal User Interface
> **Last Updated:** 2026-01-25
> **Related GAPs:** GAP-TUI-001 (Telemetry Expansion), GAP-TUI-002 (CLI Embedding)

---

## Quick Navigation

| Section | Tickets | Hours |
|---------|---------|-------|
| [Prerequisites](#prerequisites) | TUI-PREREQ-01 | 1.0 |
| [CLI Embedding](#cli-embedding-tickets) | TUI-CLI-01..05 | 12.0 |
| [Performance Views](#tui-performance-views) | TUI-PERF-01..10 | 16.0 |
| **Total** | | **29.0** |

---

## Critical Path

```
TUI-PREREQ-01 (Wire Telemetry) ← BLOCKING
        │
        ├──────────────────────────────────────┐
        │                                      │
TUI-PERF-10 (Data Status)              TUI-CLI-01 (Subprocess)
        │                                      │
TUI-PERF-01 (Dashboard Shell)          TUI-CLI-02 (Event Types)
        │                                      │
        ├──────────────────────────────────────┤
        │                                      │
   (PERF-02..09)                        TUI-CLI-03 (Claude Panel)
        │                                      │
        └──────────────────────────────────────┤
                                               │
                                        TUI-CLI-04 (Layout Integration)
                                               │
                                        TUI-CLI-05 (Session Mgmt)
```

---

## Prerequisites

### TUI-PREREQ-01: Wire Telemetry Logging to Hooks

**Estimated Hours:** 1.0
**Priority:** P0 - BLOCKING
**Status:** Pending

**Description:**
The telemetry logging functions exist but aren't called from hook binaries. Wire them up.

**Tasks:**
| Hook Binary | Function to Add | Location |
|-------------|-----------------|----------|
| `cmd/gogent-sharp-edge/main.go` | `telemetry.LogMLToolEvent()` | After successful PostToolUse |
| `cmd/gogent-validate/main.go` | `telemetry.LogRoutingDecision()` | After Task validation |
| `cmd/gogent-agent-endstate/main.go` | `telemetry.LogCollaboration()` | On SubagentStop |

**Acceptance Criteria:**
- [ ] After `claudeGO` session, `~/.local/share/gogent/tool-events.jsonl` has entries
- [ ] After `claudeGO` session, `~/.local/share/gogent/routing-decisions.jsonl` has entries
- [ ] After `claudeGO` session, `~/.local/share/gogent/agent-collaborations.jsonl` has entries

**Verification:**
```bash
wc -l ~/.local/share/gogent/*.jsonl
# Expected: Non-zero counts for all three files
```

---

## CLI Embedding Tickets

These tickets enable embedding Claude CLI directly in the TUI for bidirectional interaction.

### TUI-CLI-01: Claude Subprocess Manager

**Estimated Hours:** 3.0
**Dependencies:** None
**Priority:** P0 - Foundation

**Description:**
Implement Go subprocess manager for Claude CLI with stream-json I/O.

**Key Flags Used:**
```bash
claude --print --verbose \
       --input-format stream-json \
       --output-format stream-json \
       --include-partial-messages \
       --session-id <uuid>
```

**Files:**
- `internal/cli/subprocess.go`
- `internal/cli/streams.go`
- `internal/cli/subprocess_test.go`

**Acceptance Criteria:**
- [ ] Process starts with correct flags
- [ ] Events parsed from stdout correctly
- [ ] Messages sent via stdin work
- [ ] Session ID preserved across restarts
- [ ] Process cleanup on exit

---

### TUI-CLI-02: Event Type Definitions

**Estimated Hours:** 1.5
**Dependencies:** TUI-CLI-01
**Priority:** P0 - Foundation

**Description:**
Define Go structs for all Claude CLI stream-json event types.

**Event Types:**
| Type | Subtype | Purpose |
|------|---------|---------|
| `system` | `init` | Session initialization |
| `system` | `hook_started` | Hook execution begins |
| `system` | `hook_response` | Hook output |
| `assistant` | - | Model response |
| `result` | `success`/`error` | Final result |

**Files:**
- `internal/cli/events.go`
- `internal/cli/events_test.go`

**Acceptance Criteria:**
- [ ] All event types unmarshal correctly
- [ ] Unknown event types don't panic
- [ ] Partial messages handled
- [ ] Content blocks extracted

---

### TUI-CLI-03: Claude Interface Panel

**Estimated Hours:** 4.0
**Dependencies:** TUI-CLI-01, TUI-CLI-02, TUI-PERF-01
**Priority:** P1 - Core Feature

**Description:**
Bubble Tea component for Claude CLI interaction in TUI.

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Claude Code - Session: abc123           Cost: $0.23    │
├─────────────────────────────────────────────────────────┤
│  You: Explain this function                            │
│                                                         │
│  Claude: This function implements a binary search...   │
│  [streaming...]                                         │
│  ─────────────────────────────────────────────────────  │
│  │ Hook: gogent-validate ✅ │ Tool: Read ✅          │ │
├─────────────────────────────────────────────────────────┤
│ > Type your message here...                    [Enter] │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/claude/panel.go`
- `internal/tui/claude/input.go`
- `internal/tui/claude/output.go`
- `internal/tui/claude/events.go`

**Acceptance Criteria:**
- [ ] Text streams character-by-character
- [ ] Input submits on Enter
- [ ] Hook events display in sidebar
- [ ] Cost updates after each response
- [ ] Scroll works in viewport
- [ ] Session ID displayed

---

### TUI-CLI-04: Main Layout Integration

**Estimated Hours:** 2.0
**Dependencies:** TUI-CLI-03, TUI-PERF-01
**Priority:** P1 - Integration

**Description:**
Integrate Claude panel with performance dashboard in split layout.

**Layout:**
```
┌────────────────────────────────────┬────────────────────┐
│        Claude Interface            │   Performance      │
│        (TUI-CLI-03)                │   Dashboard        │
│                                    │   (TUI-PERF-*)     │
│  ┌──────────────────────────────┐  │                    │
│  │ Conversation viewport        │  │  [1] Violations   │
│  │                              │  │  [2] Agents       │
│  │                              │  │  [3] Cost         │
│  │                              │  │  [4] Collab       │
│  └──────────────────────────────┘  │  [5] Routing      │
│  ┌──────────────────────────────┐  │  [0] Data Status  │
│  │ > Input...                   │  │                    │
│  └──────────────────────────────┘  │                    │
└────────────────────────────────────┴────────────────────┘
```

**Files:**
- `internal/tui/main/model.go`
- `internal/tui/main/layout.go`
- `internal/tui/main/keymap.go`

**Acceptance Criteria:**
- [ ] Split layout renders correctly
- [ ] Tab switches performance views
- [ ] Focus moves between panels (Tab key)
- [ ] Resize terminal updates layout
- [ ] Keyboard shortcuts documented

---

### TUI-CLI-05: Session Management

**Estimated Hours:** 1.5
**Dependencies:** TUI-CLI-01
**Priority:** P2 - Enhancement

**Description:**
Session picker, history, and continuation support.

**Files:**
- `internal/cli/session.go`
- `internal/tui/session/picker.go`

**Acceptance Criteria:**
- [ ] Sessions listed by recency
- [ ] Resume continues context
- [ ] Fork creates new ID
- [ ] Delete removes session data

---

## TUI Performance Views

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

### TUI-PERF-08: Agent Collaboration Network

**Estimated Hours:** 2.0
**Dependencies:** TUI-PREREQ-01, TUI-PERF-01
**Data Source:** `agent-collaborations.jsonl`
**Priority:** P1 (unique feature)

**Description:**
Visualize parent→child delegation patterns, chain depths, and handoff friction.

**Key Features:**
- Top agent pairings by frequency
- Chain depth analysis (max, average)
- Handoff friction breakdown (context_loss, misunderstanding, none)
- Delegation success rate
- Swarm coordination metrics (if applicable)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Agent Collaboration Network (14 delegations)            │
├─────────────────────────────────────────────────────────┤
│ Top Pairings:              │ Chain Analysis:            │
│ orchestrator → python-pro 5│ Max Depth: 3               │
│ orchestrator → architect  3│ Avg Depth: 1.8             │
│ architect → go-pro        2│ Success Rate: 87%          │
│                            │                            │
├─────────────────────────────────────────────────────────┤
│ Handoff Friction:                                       │
│ ████████████████░░░░ none (79%)                        │
│ ███░░░░░░░░░░░░░░░░░ context_loss (14%)                │
│ █░░░░░░░░░░░░░░░░░░░ misunderstanding (7%)             │
├─────────────────────────────────────────────────────────┤
│ By Delegation Type:                                     │
│ spawn: 10  │  escalate: 3  │  parallel: 1              │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/collaborations.go`

**Acceptance Criteria:**
- [ ] Pairing frequency table renders
- [ ] Chain depth stats calculate correctly
- [ ] Friction breakdown shows percentages
- [ ] Delegation type breakdown displays
- [ ] Enter drills down to collaboration details

---

### TUI-PERF-09: Routing Decision Analysis

**Estimated Hours:** 2.0
**Dependencies:** TUI-PREREQ-01, TUI-PERF-01
**Data Source:** `routing-decisions.jsonl`, `routing-decision-updates.jsonl`
**Priority:** P1

**Description:**
Analyze routing tier selections, task classification accuracy, and override patterns.

**Key Features:**
- Task type breakdown (implementation, search, documentation, debug)
- Task domain breakdown (python, go, r, infrastructure)
- Override analysis (user-initiated vs auto-escalated)
- Confidence → Success correlation
- Understanding quality metrics (if populated)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Routing Decisions (23 total)                 Session ▼  │
├─────────────────────────────────────────────────────────┤
│ By Task Type:              │ By Domain:                 │
│ ████████████░░ impl (52%)  │ ████████░░░░ python (39%)  │
│ █████░░░░░░░░ search (22%) │ ███████░░░░░ go (35%)      │
│ ████░░░░░░░░░ docs (17%)   │ ██████░░░░░░ infra (26%)   │
│ ██░░░░░░░░░░░ debug (9%)   │                            │
├─────────────────────────────────────────────────────────┤
│ Tier Selection:                                         │
│ haiku: 8 (35%)  │  sonnet: 14 (61%)  │  opus: 1 (4%)   │
├─────────────────────────────────────────────────────────┤
│ Override Analysis:                                      │
│ Total overrides: 3 (13%)                               │
│ ├─ User-initiated: 2                                   │
│ └─ Auto-escalated: 1                                   │
│                                                         │
│ Confidence → Success: r=0.82 ✅ Strong correlation     │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/routing.go`

**Acceptance Criteria:**
- [ ] Task type breakdown renders with bars
- [ ] Domain breakdown renders
- [ ] Tier selection summary displays
- [ ] Override count and breakdown shows
- [ ] Confidence correlation calculates (requires outcome updates)
- [ ] Join with updates file works correctly

---

### TUI-PERF-10: Data Collection Status (Diagnostic)

**Estimated Hours:** 1.0
**Dependencies:** TUI-PERF-01
**Data Source:** File system checks
**Priority:** P0 - Diagnostic (DO FIRST)

**Description:**
Diagnostic dashboard showing status of all telemetry log files. This is the FIRST view to implement as it validates the logging infrastructure.

**Key Features:**
- File existence check for all JSONL logs
- Line count for each file
- Status indicator (active, empty, missing)
- Last modified timestamp
- Actionable guidance for empty/missing files
- Export and reset controls

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ ML Data Collection Status                    [R]efresh  │
├─────────────────────────────────────────────────────────┤
│ Global Logs (~/.local/share/gogent/)                    │
│ ─────────────────────────────────────────────────────── │
│ tool-events.jsonl          │ 0 entries   ⚠️ NOT LOGGING │
│ routing-decisions.jsonl    │ 0 entries   ⚠️ NOT LOGGING │
│ agent-collaborations.jsonl │ 0 entries   ⚠️ NOT LOGGING │
├─────────────────────────────────────────────────────────┤
│ Runtime Logs (~/.gogent/)                               │
│ ─────────────────────────────────────────────────────── │
│ failure-tracker.jsonl      │ 3 entries   ✅ Active      │
├─────────────────────────────────────────────────────────┤
│ Project Logs (.claude/memory/)                          │
│ ─────────────────────────────────────────────────────── │
│ pending-learnings.jsonl    │ 5 entries   ✅ Active      │
│ handoffs.jsonl             │ 12 entries  ✅ Active      │
│ agent-endstates.jsonl      │ 8 entries   ✅ Active      │
│ user-intents.jsonl         │ 0 entries   ⚠️ Empty       │
├─────────────────────────────────────────────────────────┤
│ ⚠️ 3 files not logging. Run TUI-PREREQ-01 first.       │
│                                                         │
│ [E]xport all  [V]alidate schemas  [?] Help              │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/performance/datacheck.go`

**Acceptance Criteria:**
- [ ] All log file paths checked correctly
- [ ] Line counts accurate
- [ ] Status icons display (✅/⚠️/❌)
- [ ] Refresh updates counts
- [ ] Actionable message shows when files empty
- [ ] Export functionality works

---

## Dependency Graph

```
                    TUI-PREREQ-01 ←────────────────────────────────────┐
                  (Wire Telemetry)                                     │
                         │ BLOCKING                                    │
                         │                                             │
        ┌────────────────┼────────────────┐                            │
        │                │                │                            │
        ↓                ↓                ↓                            │
  TUI-PERF-10      TUI-PERF-08      TUI-PERF-09                       │
  (Data Status)    (Collaborations) (Routing)                         │
  [DO FIRST]            │                │                            │
        │               │                │                            │
        ↓               │                │                            │
  TUI-PERF-01 ←─────────┴────────────────┘                            │
  (Dashboard)                                                          │
        │                                                              │
        ├──────────┬──────────┬──────────┬──────────┐                 │
        │          │          │          │          │                 │
        ↓          ↓          ↓          ↓          ↓                 │
  TUI-PERF-02 TUI-PERF-03 TUI-PERF-04 TUI-PERF-05 TUI-PERF-06        │
  (Violations) (Agents)  (Cost)     (Escalations) (Scout)            │
        │          │          │          │          │                 │
        └──────────┴──────────┴──────────┴──────────┘                 │
                         │                                             │
                         ↓                                             │
                   TUI-PERF-07                                         │
                 (Drill-Down Detail)                                   │
                                                                       │
                                                                       │
  TUI-CLI-01 ──→ TUI-CLI-02 ──→ TUI-CLI-03 ──────────────────────────┘
  (Subprocess)   (Events)        (Claude Panel)
                                       │
                                       ↓
                                 TUI-CLI-04
                               (Layout Integration)
                                       │
                                       ↓
                                 TUI-CLI-05
                               (Session Management)
```

### Execution Order (Recommended)

1. **TUI-PREREQ-01** - Wire telemetry (BLOCKING for PERF-08, 09, 10)
2. **TUI-PERF-10** - Validate logging works before building views
3. **TUI-CLI-01** - Start subprocess work in parallel
4. **TUI-PERF-01** - Dashboard shell
5. **TUI-PERF-08, 09** - New telemetry views (unique features)
6. **TUI-PERF-02-06** - Existing views (parallel)
7. **TUI-CLI-02-05** - Complete CLI integration
8. **TUI-PERF-07** - Drill-down (depends on all views)

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

**Document Version:** 2.0
**Created:** 2026-01-22
**Last Updated:** 2026-01-25
**Related GAPs:**
- `GAP-TUI-TELEMETRY-EXPANSION.md` (telemetry wiring, new views)
- `GAP-TUI-CLI-EMBEDDING.md` (Claude CLI bidirectional relay)
