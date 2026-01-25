# GAP Document: TUI Telemetry Expansion

> **GAP ID:** GAP-TUI-001
> **Created:** 2026-01-25
> **Author:** Einstein Analysis
> **Status:** Ready for Planning
> **Related:** TUI-PERF-INDEX.md, GOgent-088 series

---

## 1. Problem Statement

### Primary Question

**TUI-PERF-INDEX.md is stale.** Since its creation (2026-01-22), significant telemetry infrastructure was added via GOgent-087/088 tickets that is NOT reflected in the TUI plans:

1. `AgentCollaboration` tracking (GOgent-088b)
2. `RoutingDecision` logging (GOgent-087d)
3. `PostToolEvent` ML enrichment (GOgent-088)
4. Task classification system (GOgent-087c)
5. Understanding quality metrics (Addendum A.2)
6. Swarm coordination fields (Addendum A.3)

### Blocking Issue

**The telemetry logging calls are NOT wired into hook binaries.** The `~/.local/share/gogent/` directory exists but is EMPTY:

```bash
$ ls -la ~/.local/share/gogent/
total 0
drwxr-xr-x 1 doktersmol doktersmol   0 Jan 25 10:55 .
```

This means:
- `LogMLToolEvent()` exists but isn't called from `gogent-sharp-edge`
- `LogRoutingDecision()` exists but isn't called from `gogent-validate`
- `LogCollaboration()` exists but isn't called from `gogent-agent-endstate`

---

## 2. Context

### Current Telemetry Infrastructure

| Package | File | Key Types | Logging Function |
|---------|------|-----------|------------------|
| `pkg/telemetry` | `collaboration.go` | `AgentCollaboration` | `LogCollaboration()` |
| `pkg/telemetry` | `routing_decision.go` | `RoutingDecision`, `DecisionOutcomeUpdate` | `LogRoutingDecision()`, `UpdateDecisionOutcome()` |
| `pkg/telemetry` | `ml_logging.go` | - | `LogMLToolEvent()` |
| `pkg/telemetry` | `ml_tool_event.go` | - | `TotalTokens()`, `EstimatedCost()`, `EnrichWithSequence()` |
| `pkg/telemetry` | `task_classifier.go` | - | `ClassifyTask()` |
| `pkg/telemetry` | `escalations.go` | `EscalationEvent` | `LogEscalation()` |
| `pkg/telemetry` | `scout.go` | `ScoutRecommendation` | `LogScoutRecommendation()` |

### Current TUI Plans (TUI-PERF-INDEX.md)

| Ticket | View | Data Source |
|--------|------|-------------|
| TUI-PERF-01 | Dashboard Shell | Navigation only |
| TUI-PERF-02 | Violations Summary | `routing-violations.jsonl` |
| TUI-PERF-03 | Agent Leaderboard | `AgentInvocation` (030f) |
| TUI-PERF-04 | Cost Attribution | `SessionCostSummary` (030g) |
| TUI-PERF-05 | Escalation Timeline | `EscalationEvent` (030h/i) |
| TUI-PERF-06 | Scout Compliance | `ScoutRecommendation` (030j/k) |
| TUI-PERF-07 | Drill-Down Detail | Shared component |

### Gap: Missing TUI Views

| New Data | TUI Coverage |
|----------|--------------|
| `AgentCollaboration` | **NONE** |
| `RoutingDecision` | **PARTIAL** (violations only) |
| `PostToolEvent` ML fields | **NONE** |
| ML data collection status | **NONE** (can't validate logging works) |

---

## 3. Proposed Solution

### Phase 0: Telemetry Wiring (BLOCKING)

Before ANY TUI work, wire telemetry calls into hooks:

**Ticket: TUI-PREREQ-01 - Wire Telemetry Logging**

| Hook Binary | Function to Call | Trigger |
|-------------|------------------|---------|
| `cmd/gogent-sharp-edge/main.go` | `telemetry.LogMLToolEvent()` | Every PostToolUse |
| `cmd/gogent-validate/main.go` | `telemetry.LogRoutingDecision()` | Every Task validation |
| `cmd/gogent-agent-endstate/main.go` | `telemetry.LogCollaboration()` | Every SubagentStop |

**Acceptance Criteria:**
- [ ] After running `claudeGO`, `~/.local/share/gogent/tool-events.jsonl` has entries
- [ ] After running `claudeGO`, `~/.local/share/gogent/routing-decisions.jsonl` has entries
- [ ] After running `claudeGO`, `~/.local/share/gogent/agent-collaborations.jsonl` has entries

### Phase 1: New TUI Views

#### TUI-PERF-08: Agent Collaboration Network

**Estimated Hours:** 2.0
**Dependencies:** TUI-PREREQ-01, TUI-PERF-01
**Data Source:** `agent-collaborations.jsonl`

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

#### TUI-PERF-09: Routing Decision Analysis

**Estimated Hours:** 2.0
**Dependencies:** TUI-PREREQ-01, TUI-PERF-01
**Data Source:** `routing-decisions.jsonl`, `routing-decision-updates.jsonl`

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

#### TUI-PERF-10: Data Collection Status (Diagnostic)

**Estimated Hours:** 1.0
**Dependencies:** TUI-PERF-01
**Data Source:** File system checks

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

### Phase 2: Enhanced Existing Views

#### TUI-PERF-03-ENHANCED: Agent Leaderboard with ML Fields

Add columns for new ML telemetry:
- Task type distribution per agent
- Average context window used
- Understanding quality score (if available)

#### TUI-PERF-05-ENHANCED: Escalation Timeline with Collaboration

Link escalations to their collaboration chain:
- Show parent agent that triggered escalation
- Display chain depth at escalation point
- Correlate with handoff friction

---

## 4. Updated Dependency Graph

```
┌─────────────────────────────────────────────────────────┐
│                   TUI-PREREQ-01                         │
│              (Wire Telemetry Logging)                   │
│                     BLOCKING                            │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   TUI-PERF-10                           │
│            (Data Collection Status)                     │
│              DIAGNOSTIC - DO FIRST                      │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   TUI-PERF-01                           │
│               (Dashboard Shell)                         │
└─────────────────────┬───────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┬─────────────┐
        │             │             │             │
┌───────▼───────┐ ┌───▼───┐ ┌───────▼───────┐ ┌───▼───────┐
│ TUI-PERF-02   │ │  03   │ │ TUI-PERF-08   │ │ TUI-PERF-09│
│ Violations    │ │Agents │ │ Collaborations│ │ Routing    │
│ (existing)    │ │(exist)│ │ (NEW)         │ │ (NEW)      │
└───────┬───────┘ └───┬───┘ └───────┬───────┘ └───┬───────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
┌───────▼───────┐ ┌───▼───┐ ┌───────▼───────┐
│ TUI-PERF-04   │ │  05   │ │ TUI-PERF-06   │
│ Cost          │ │Escal. │ │ Scout         │
│ (existing)    │ │(exist)│ │ (existing)    │
└───────┬───────┘ └───┬───┘ └───────┬───────┘
        │             │             │
        └─────────────┴─────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   TUI-PERF-07                           │
│              (Drill-Down Detail)                        │
│                   (existing)                            │
└─────────────────────────────────────────────────────────┘
```

---

## 5. File Inventory

### New Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/performance/datacheck.go` | TUI-PERF-10 Data Status view |
| `internal/tui/performance/collaborations.go` | TUI-PERF-08 Collaboration Network view |
| `internal/tui/performance/routing.go` | TUI-PERF-09 Routing Decision view |

### Files to Modify

| File | Modification |
|------|--------------|
| `cmd/gogent-sharp-edge/main.go` | Add `telemetry.LogMLToolEvent()` call |
| `cmd/gogent-validate/main.go` | Add `telemetry.LogRoutingDecision()` call |
| `cmd/gogent-agent-endstate/main.go` | Add `telemetry.LogCollaboration()` call |
| `internal/tui/performance/agents.go` | Add ML field columns (enhancement) |
| `internal/tui/performance/escalations.go` | Link to collaboration data (enhancement) |

### Existing Files (Reference)

| File | Used By |
|------|---------|
| `pkg/telemetry/collaboration.go` | TUI-PERF-08 |
| `pkg/telemetry/routing_decision.go` | TUI-PERF-09 |
| `pkg/telemetry/ml_logging.go` | TUI-PERF-10 |
| `pkg/config/paths.go` | All views (path resolution) |

---

## 6. Effort Estimates

| Ticket | Hours | Priority |
|--------|-------|----------|
| TUI-PREREQ-01 | 1.0 | **P0 - BLOCKING** |
| TUI-PERF-10 | 1.0 | **P0 - Diagnostic** |
| TUI-PERF-01 | 2.0 | P1 |
| TUI-PERF-08 | 2.0 | P1 (unique feature) |
| TUI-PERF-09 | 2.0 | P1 |
| TUI-PERF-02 | 1.5 | P2 |
| TUI-PERF-03 | 2.0 | P2 |
| TUI-PERF-04 | 1.5 | P2 |
| TUI-PERF-05 | 1.5 | P2 |
| TUI-PERF-06 | 1.5 | P2 |
| TUI-PERF-07 | 1.0 | P3 |
| **Total** | **17.0** | |

---

## 7. Success Criteria

### Telemetry Wiring Complete When:
```bash
# All three files have entries after a claudeGO session
$ wc -l ~/.local/share/gogent/*.jsonl
  15 tool-events.jsonl
   8 routing-decisions.jsonl
   5 agent-collaborations.jsonl
```

### TUI Ready for Testing When:
1. TUI-PERF-10 shows all files as ✅ Active
2. Dashboard shell navigates between all views
3. Each view renders real data (not mocks)
4. Drill-down works from any summary to detail

### ML Training Data Ready When:
1. 1000+ tool events logged
2. 500+ routing decisions with outcomes
3. 200+ collaboration records
4. Schema validation passes on all files

---

## 8. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Logging performance overhead | Slow hooks | Profile; use goroutines for async writes |
| JSONL file grows unbounded | Disk full | Add rotation/archival in TUI-PERF-10 |
| Outcome updates never populated | Poor ML data | Add explicit outcome tracking in hooks |
| Swarm fields never populated | Missing feature | Document as future enhancement |

---

## 9. Open Questions

1. **Should tool-events.jsonl have retention policy?** (e.g., rotate weekly)
2. **Should TUI-PERF-10 have auto-refresh?** (polling vs manual)
3. **Should collaboration chains be visualized as graph?** (ASCII vs lipgloss)
4. **Should routing decisions show raw vs aggregated?** (toggle?)

---

## 10. Next Actions

1. **Create ticket TUI-PREREQ-01** - Wire telemetry logging (BLOCKING)
2. **Update TUI-PERF-INDEX.md** - Add new tickets 08, 09, 10
3. **Create TUI-PERF-10 first** - Validates logging before other views
4. **Run claudeGO session** - Generate test data for development

---

**Document Version:** 1.0
**GAP Template Version:** 1.0
**Related Documents:**
- `TUI-PERF-INDEX.md` (to be updated)
- `pkg/telemetry/*.go` (data sources)
- `cmd/gogent-*/main.go` (hook binaries)
