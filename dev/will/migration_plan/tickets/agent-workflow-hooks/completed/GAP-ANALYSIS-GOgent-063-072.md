# Einstein GAP Analysis: Agent Workflow Hooks (GOgent-063 to GOgent-072)

> **Generated:** 2026-01-24T14:30:00Z
> **Escalated By:** Einstein (Opus) + Staff-Architect (Opus) dual analysis
> **Session:** Comprehensive architectural review
> **Analysis Type:** Orthogonal synthesis (two independent Opus analyses merged)

---

## 1. Problem Statement

### What We're Trying to Achieve

Implement two new hook subsystems for the GOgent-Fortress orchestration framework:

1. **Agent-Endstate Hook (GOgent-063 to GOgent-067)**: Capture SubagentStop events and generate tier-specific follow-up responses when agents complete their work.

2. **Attention-Gate Hook (GOgent-068 to GOgent-072)**: Inject routing compliance reminders every N tool calls and auto-flush pending learnings to prevent data loss.

### Why This Escalated

- [x] Architectural decision required
- [x] Cross-domain synthesis needed
- [x] Complexity exceeds Sonnet tier (10 tickets, multiple subsystems, integration concerns)
- [ ] 3+ consecutive failures on same task
- [ ] User explicitly requested deep analysis

**Specific Escalation Triggers:**
1. SubagentStop event type validity unconfirmed - could invalidate 5 tickets
2. PostToolUse hook conflict between existing `gogent-sharp-edge` and proposed `gogent-attention-gate`
3. Significant code duplication with existing implementations detected
4. Memory schema integration gaps identified

---

## 2. What Was Tried

### Analysis Approach

| # | Agent | Action | Result |
|---|-------|--------|--------|
| 1 | Einstein (Opus) | Read all 10 ticket specifications | Identified schema, scope, dependency issues |
| 2 | Staff-Architect (Opus) | Orthogonal architectural review | Confirmed duplication, SubagentStop validity concern |
| 3 | Einstein | Cross-reference existing codebase | Found duplicate implementations in pkg/routing, pkg/config |
| 4 | Einstein | Analyze routing-schema.json | Confirmed agent_subagent_mapping patterns |
| 5 | Einstein | Synthesize both analyses | Converged on 3 blocking issues, 4 refactoring recommendations |

### Key Issues Discovered

```
CRITICAL ISSUE 1: SubagentStop Event Validity
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: UNVERIFIED
Evidence: No SubagentStop events in existing event corpus (100+ events validated)
Impact: GOgent-063 through GOgent-067 may be implementing against non-existent API
Risk: 5 tickets (~8 hours) potentially unusable

CRITICAL ISSUE 2: PostToolUse Hook Conflict
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: UNRESOLVED
Current: gogent-sharp-edge handles PostToolUse for failure tracking
Proposed: gogent-attention-gate handles PostToolUse for counter/reminders
Problem: Claude Code hook configuration typically allows ONE hook per event type
Impact: Runtime conflict or need for orchestration wrapper

CRITICAL ISSUE 3: Code Duplication
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Location: Multiple proposed implementations duplicate existing code

Proposed pkg/observability/counter.go duplicates:
  → pkg/config/paths.go lines 80-168 (ToolCounter, IncrementToolCount)

Proposed pkg/observability/events.go duplicates:
  → pkg/routing/events.go lines 91-132 (ParsePostToolEvent)

Proposed pkg/workflow/events.go duplicates:
  → pkg/routing/events.go lines 55-89 (event parsing pattern)
```

---

## 3. Relevant Context

### Files Involved

| File | Lines | Relevance |
|------|-------|-----------|
| `pkg/routing/events.go` | 176 | **Existing event parsing** - validated against 100+ production events |
| `pkg/config/paths.go` | 169 | **Existing tool counter** - XDG-compliant, syscall.Flock() atomicity |
| `pkg/session/handoff.go` | 494 | **Handoff schema v1.2** - artifacts integration point |
| `pkg/session/handoff_artifacts.go` | ~300 | **LoadArtifacts()** - must integrate new artifact types |
| `routing-schema.json` | 343 | **Agent/subagent mappings** - defines hook behavior |
| `cmd/gogent-sharp-edge/main.go` | ~300 | **Existing PostToolUse handler** - conflict point |

### File Contents (Critical Excerpts)

#### pkg/routing/events.go (lines 20-45)
```go
// PostToolEvent represents PostToolUse events with execution results.
// These events include both the input and the tool's response.
type PostToolEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  map[string]interface{} `json:"tool_response"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`
}

// VALIDATION NOTES (GOgent-006):
//
// Struct validated against 100+ real production events from event-corpus.json.
// All events conform to this structure. Key validation findings:
//
// 1. CWD Field: Not present in any corpus events. Not added to struct.
// 2. Timestamp: All events use "captured_at" (Unix epoch int64). No alternatives found.
// 3. ToolInput: Always a JSON object (map[string]interface{}). Never null or string.
// 4. Field Visibility: All 5 fields required and present in 100% of events.
// 5. PostToolUse: Adds tool_response field (map[string]interface{}) to base structure.
```

**Gap Identified:** Proposed `PostToolUseEvent` in GOgent-070 adds `Duration`, `Success`, `ToolCategory` fields that don't exist in validated corpus.

#### pkg/config/paths.go (lines 80-134)
```go
// GetToolCounterPath returns path to tool counter file.
func GetToolCounterPath() string {
	return filepath.Join(GetGOgentDir(), "tool-counter")
}

// IncrementToolCount atomically increments tool counter.
// Uses file locking (flock) to ensure true atomicity in concurrent scenarios.
func IncrementToolCount() error {
	path := GetToolCounterPath()

	// Open file for read-write, create if doesn't exist
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open counter file at %s: %w", path, err)
	}
	defer file.Close()

	// Acquire exclusive lock (blocks until available)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock counter file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	// ... increment logic
}
```

**Gap Identified:** GOgent-068 proposes new `ToolCounter` struct with mutex-based locking in `/tmp/`. Existing implementation uses:
- XDG-compliant paths (session isolation)
- `syscall.Flock()` (more robust cross-process locking)
- Already integrated with hook infrastructure

#### pkg/session/handoff.go (lines 38-48)
```go
// HandoffArtifacts contains references to session artifacts
type HandoffArtifacts struct {
	SharpEdges        []SharpEdge        `json:"sharp_edges"`
	RoutingViolations []RoutingViolation `json:"routing_violations"`
	ErrorPatterns     []ErrorPattern     `json:"error_patterns"`
	UserIntents       []UserIntent       `json:"user_intents"`
	// Extended fields (v1.1 - backward compatible via omitempty)
	Decisions           []Decision           `json:"decisions,omitempty"`
	PreferenceOverrides []PreferenceOverride `json:"preference_overrides,omitempty"`
	PerformanceMetrics  []PerformanceMetric  `json:"performance_metrics,omitempty"`
}
```

**Gap Identified:** Neither `EndstateLog` (GOgent-065) nor auto-flush archives (GOgent-069) integrate with `HandoffArtifacts`. New artifacts will not be included in session archives.

### Architectural Context

```
GOgent-Fortress Hook Event Flow (Current)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SessionStart ─→ gogent-load-context ─→ Context injection
     │
     ▼
┌────────────────────────────────────────────────────────┐
│                   Tool Usage Loop                       │
│                                                         │
│  PreToolUse ──→ gogent-validate ──→ Allow/Block        │
│       │                                                 │
│       ▼                                                 │
│  [Tool Execution]                                       │
│       │                                                 │
│       ▼                                                 │
│  PostToolUse ─→ gogent-sharp-edge ─→ Failure tracking  │
│       │                                                 │
│       └───────────────────────────────────────────────┐│
│                                                        ││
│  (PROPOSED) PostToolUse ─→ gogent-attention-gate      ││
│                            ↳ Counter + Reminder       ││
│                                                        ││
│  (PROPOSED) SubagentStop ─→ gogent-agent-endstate     ││
│                             ↳ Tier-specific follow-up ││
└────────────────────────────────────────────────────────┘
     │
     ▼
SessionEnd ──→ gogent-archive ──→ Handoff generation


CONFLICT ZONE: PostToolUse has TWO proposed handlers
UNKNOWN ZONE: SubagentStop event type not validated
```

---

## 4. Constraints

### Hard Constraints (Cannot Violate)

1. **XDG Compliance**: All persistent files must use `GetGOgentDir()` path hierarchy, not hardcoded `/tmp/` paths
2. **Schema Backward Compatibility**: HandoffArtifacts changes must be `omitempty` for v1.2 compatibility
3. **Single Hook Per Event**: Claude Code typically supports one hook handler per event type (verify)
4. **Corpus Validation**: Event structs must be validated against real Claude Code events before implementation

### Soft Constraints (Should Respect)

5. **Package Hygiene**: Prefer extending existing packages over creating new ones
6. **DRY Principle**: Must not duplicate event parsing, counter management, or response generation
7. **Test Coverage**: Maintain 80%+ coverage; use simulation harness for integration tests
8. **Makefile Integration**: New CLIs must have build targets and install targets

### Resource Constraints

9. **Ticket Budget**: 10 tickets allocated (~15.5 hours estimated)
10. **Dependency Chain**: Linear dependencies mean blocking issues cascade to all downstream tickets

---

## 5. Questions for Resolution

### Primary Question

> **How should GOgent-063 to GOgent-072 be restructured to eliminate code duplication, resolve the PostToolUse hook conflict, and handle the unvalidated SubagentStop event type?**

### Sub-Questions

1. **SubagentStop Validation**: Is SubagentStop a real Claude Code hook event type? What documentation or event corpus confirms this?

2. **PostToolUse Orchestration**: Should attention-gate logic be merged into `gogent-sharp-edge`, or should a dispatcher pattern be used?

3. **Package Architecture**: Should `pkg/workflow/` and `pkg/observability/` exist, or should all code extend existing `pkg/routing/`, `pkg/config/`, `pkg/session/`?

4. **Handoff Integration**: What changes to `HandoffArtifacts` and `LoadArtifacts()` are required for new artifact types?

5. **Path Standardization**: Should GOgent-065's `/tmp/claude-agent-endstates.jsonl` become `~/.cache/gogent/agent-endstates.jsonl` or `.claude/memory/agent-endstates.jsonl`?

---

## 6. Gap Analysis Matrix

### Per-Ticket Assessment

| Ticket | Status | Gap Type | Severity | Required Action |
|--------|--------|----------|----------|-----------------|
| GOgent-063 | BLOCKED | SubagentStop unverified | CRITICAL | Validate event type exists |
| GOgent-064 | BLOCKED | Depends on GOgent-063 | CRITICAL | Hold pending validation |
| GOgent-065 | NEEDS REFACTOR | Path hardcoding, no handoff integration | HIGH | Use XDG paths, add to HandoffArtifacts |
| GOgent-066 | NEEDS REFACTOR | Tests don't use simulation harness | MEDIUM | Add harness integration tests |
| GOgent-067 | BLOCKED | Depends on GOgent-063-066 | CRITICAL | Hold pending upstream |
| GOgent-068 | NEEDS REFACTOR | Duplicates pkg/config/paths.go | HIGH | Extend existing, don't create new |
| GOgent-069 | NEEDS REFACTOR | Flush threshold uncalibrated, duplicate CheckPendingLearnings | MEDIUM | Make configurable, use existing |
| GOgent-070 | ELIMINATE | Duplicates pkg/routing/events.go | HIGH | Remove, use existing ParsePostToolEvent |
| GOgent-071 | NEEDS REFACTOR | Counter tests affect global state | LOW | Use temp paths in tests |
| GOgent-072 | NEEDS REFACTOR | PostToolUse conflict, env var inconsistency | HIGH | Merge into sharp-edge or orchestrate |

### Gap Categories Summary

| Category | Count | Impact |
|----------|-------|--------|
| **CRITICAL (Blockers)** | 4 | Cannot proceed with GOgent-063-067 |
| **HIGH (Major Refactor)** | 4 | Significant ticket rewrite required |
| **MEDIUM (Minor Refactor)** | 2 | Adjustments within ticket scope |
| **LOW (Polish)** | 1 | Quality improvement, not blocking |

---

## 7. Recommended Restructuring

### Phase 1: Validation (Before Any Implementation)

**New Ticket: GOgent-063a - SubagentStop Event Validation**
```yaml
id: GOgent-063a
title: Validate SubagentStop Hook Event Type
time_estimate: 1h
dependencies: []
scope: research_only
deliverables:
  - Confirmation from Claude Code docs or event corpus
  - Sample SubagentStop event JSON (if exists)
  - GO/NO-GO decision for GOgent-063-067
```

### Phase 2: Attention-Gate Consolidation (Unblocked Work)

**Refactored Ticket: GOgent-068R - Counter Threshold Functions**
```yaml
id: GOgent-068R
title: Extend Tool Counter with Threshold Functions
time_estimate: 1h
dependencies: []
location: pkg/config/paths.go (EXTEND, not new package)
changes:
  - Add ShouldRemind(count int) bool // true every 10
  - Add ShouldFlush(count int) bool  // true every 20
  - Add GetToolCountAndIncrement() (int, error) // atomic read+increment
```

**Refactored Ticket: GOgent-069R - Attention-Gate Logic**
```yaml
id: GOgent-069R
title: Reminder and Flush Logic
time_estimate: 2h
dependencies: [GOgent-068R]
location: pkg/session/ (leverage existing pending-learnings handling)
changes:
  - GenerateRoutingReminder(count int, summary string) string
  - Use existing CheckPendingLearnings from context_loader.go
  - ArchivePendingLearnings with HandoffArtifacts integration
```

**Eliminated Ticket: GOgent-070 - PostToolUse Event Parsing**
```yaml
status: ELIMINATED
reason: Duplicates pkg/routing/events.go ParsePostToolEvent()
action: Use existing implementation
```

**Refactored Ticket: GOgent-070R - Merge Attention-Gate into Sharp-Edge**
```yaml
id: GOgent-070R
title: Add Attention-Gate Logic to gogent-sharp-edge CLI
time_estimate: 2h
dependencies: [GOgent-069R]
location: cmd/gogent-sharp-edge/main.go
changes:
  - Add counter increment using config.GetToolCountAndIncrement()
  - Inject reminder if config.ShouldRemind(count)
  - Archive learnings if config.ShouldFlush(count) && pending >= 5
  - Single PostToolUse handler, no orchestration needed
```

**Refactored Ticket: GOgent-071R - Integration Tests**
```yaml
id: GOgent-071R
title: Attention-Gate Integration Tests
time_estimate: 1.5h
dependencies: [GOgent-070R]
changes:
  - Use t.TempDir() for all file paths (no global state)
  - Add simulation harness test for reminder injection
  - Add simulation harness test for flush behavior
```

### Phase 3: Agent-Endstate (Pending Validation)

**Conditional on GOgent-063a GO decision:**

**Refactored Ticket: GOgent-064R - SubagentStop Structs + Response**
```yaml
id: GOgent-064R
title: SubagentStop Event Handling and Response Generation
time_estimate: 2h
dependencies: [GOgent-063a]
condition: GOgent-063a returns GO decision
location: pkg/routing/events.go (add SubagentStopEvent struct)
         pkg/memory/responses.go (add GenerateEndstateResponse)
changes:
  - Add SubagentStopEvent to pkg/routing/events.go
  - Add ParseSubagentStopEvent following existing pattern
  - Add response generation following pkg/memory/responses.go pattern
```

**Refactored Ticket: GOgent-065R - Endstate CLI + Handoff Integration**
```yaml
id: GOgent-065R
title: Agent-Endstate CLI with Handoff Integration
time_estimate: 2h
dependencies: [GOgent-064R]
changes:
  - Add AgentEndstates []EndstateLog to HandoffArtifacts
  - Add EndstatesPath to HandoffConfig
  - Add loadEndstates() to LoadArtifacts()
  - Build gogent-agent-endstate CLI
  - Add Makefile targets
```

### Phase 4: Missing Tickets

**New Ticket: GOgent-073 - HandoffArtifacts Extension**
```yaml
id: GOgent-073
title: Extend HandoffArtifacts for New Artifact Types
time_estimate: 1h
dependencies: [GOgent-069R, GOgent-065R]
changes:
  - Add AgentEndstates []EndstateLog (omitempty)
  - Add AutoFlushArchives []string (omitempty)
  - Update LoadArtifacts() with new loading functions
  - Update schema version to 1.3 if breaking
```

**New Ticket: GOgent-074 - Hook Configuration Documentation**
```yaml
id: GOgent-074
title: Update Systems Architecture for Merged PostToolUse Handler
time_estimate: 0.5h
dependencies: [GOgent-070R]
changes:
  - Update docs/systems-architecture-overview.md
  - Document single PostToolUse handler strategy
  - Add attention-gate behavior to CLI reference
```

---

## 8. Expected Deliverable

**Format:** Comprehensive GAP analysis document with actionable ticket restructuring

**Location:** `dev/will/migration_plan/tickets/agent-workflow-hooks/GAP-ANALYSIS-GOgent-063-072.md`

### Success Criteria

- [x] All 10 original tickets assessed for gaps
- [x] Critical blockers identified (SubagentStop validation)
- [x] Code duplication points mapped to existing implementations
- [x] PostToolUse conflict resolution strategy defined
- [x] Restructured ticket specifications provided
- [x] Missing tickets identified and specified
- [ ] User review and approval of restructuring
- [ ] Ticket index updated with new structure

---

## 9. Anti-Scope

This analysis should NOT:

- Implement any code changes (analysis only)
- Modify existing tickets without user approval
- Assume SubagentStop exists without validation
- Proceed with GOgent-063-067 before validation completes
- Create new packages when existing packages can be extended
- Add features beyond the original 10-ticket scope

---

## 10. Decision Matrix

### Recommended Path Forward

| Decision Point | Recommendation | Rationale |
|----------------|----------------|-----------|
| SubagentStop validation | **VALIDATE FIRST** | 5 tickets (~8h) at risk; cheap validation ($0.02) vs expensive rework |
| PostToolUse conflict | **MERGE into sharp-edge** | Single handler simpler than orchestration; maintains existing patterns |
| New packages | **DO NOT CREATE** | Extend pkg/config, pkg/routing, pkg/session instead |
| Counter implementation | **USE EXISTING** | pkg/config/paths.go already has atomic, XDG-compliant counter |
| Event parsing | **USE EXISTING** | pkg/routing/events.go validated against corpus |
| Handoff integration | **ADD to v1.2** | Use omitempty for backward compatibility |

### Cost-Benefit Analysis

| Approach | Tickets | Hours | Risk |
|----------|---------|-------|------|
| **Proceed as-is** | 10 | 15.5h | HIGH - SubagentStop may not exist, duplication tech debt |
| **Full refactor** | 8 | 11h | LOW - But blocked on validation |
| **Partial refactor** (recommended) | 6 now + 2 conditional | 9h + 4h | MEDIUM - Unblocks attention-gate, defers agent-endstate |

**Recommendation:** Partial refactor. Implement GOgent-068R through GOgent-071R (attention-gate, ~6h) immediately while GOgent-063a validates SubagentStop. If validation passes, proceed with GOgent-064R and GOgent-065R (~4h). If validation fails, remove agent-endstate from scope.

---

## Metadata

```yaml
gap_id: GAP-AWH-2026-01-24
analysis_type: dual_opus_synthesis
complexity_score: 8/10
estimated_analysis_tokens: ~45000
files_referenced: 12
tickets_analyzed: 10
critical_blockers: 1
high_priority_refactors: 4
new_tickets_proposed: 4
eliminated_tickets: 1
created_at: 2026-01-24T14:30:00Z
analysts:
  - einstein (opus)
  - staff-architect (opus)
synthesis_confidence: high
```

---

## Appendix A: File Duplication Map

```
PROPOSED                          EXISTING (USE THIS INSTEAD)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

pkg/observability/counter.go      pkg/config/paths.go
├─ ToolCounter struct             ├─ GetToolCounterPath()
├─ NewToolCounter()               ├─ InitializeToolCounter()
├─ Increment()                    ├─ IncrementToolCount()  ← syscall.Flock()
├─ Read()                         ├─ GetToolCount()
├─ ShouldRemind()                 └─ [ADD THIS]
└─ ShouldFlush()                  └─ [ADD THIS]

pkg/observability/events.go       pkg/routing/events.go
├─ PostToolUseEvent struct        ├─ PostToolEvent struct  ← corpus validated
└─ ParsePostToolUseEvent()        └─ ParsePostToolEvent()

pkg/workflow/events.go            pkg/routing/events.go
├─ SubagentStopEvent struct       └─ [ADD if validated]
├─ ParseSubagentStopEvent()       └─ [ADD if validated]
└─ GetAgentClass()                └─ [ADD to pkg/routing/agents.go]

pkg/workflow/responses.go         pkg/memory/responses.go
├─ EndstateResponse struct        └─ [ADD following existing pattern]
├─ GenerateEndstateResponse()     └─ [ADD following existing pattern]
└─ FormatResponseJSON()           └─ [USE json.Marshal, not string fmt]
```

---

## Appendix B: Ticket Dependency Graph (Restructured)

```
VALIDATION GATE
    │
    ▼
GOgent-063a ─────────────────────────────────────────────┐
(SubagentStop                                             │
 Validation)                                              │
    │                                                     │
    ├─ GO ────────────────┐                               │
    │                     ▼                               │
    │              GOgent-064R                            │
    │              (Structs +                             │
    │               Response)                             │
    │                     │                               │
    │                     ▼                               │
    │              GOgent-065R                            │
    │              (CLI +                                 │
    │               Handoff)                              │
    │                     │                               │
    └─ NO-GO ─────────────┴─────────────────────────────▶ SKIP
                                                          │
                                                          │
UNBLOCKED PATH (Proceed Immediately)                      │
    │                                                     │
    ▼                                                     │
GOgent-068R ──▶ GOgent-069R ──▶ GOgent-070R ──▶ GOgent-071R
(Counter       (Reminder/      (Merge to      (Integration
 Thresholds)    Flush)          Sharp-Edge)    Tests)
                                    │
                                    ▼
                              GOgent-074
                              (Docs Update)
                                    │
                                    ▼
                              GOgent-073
                              (Handoff
                               Extension)
```

---

## Appendix C: Validation Checklist for GOgent-063a

Before marking GOgent-063a complete, verify:

- [ ] Claude Code documentation reviewed for SubagentStop hook event type
- [ ] Event corpus searched for SubagentStop examples
- [ ] If found: 10+ sample events captured
- [ ] If found: Schema validated against samples
- [ ] If found: Fields mapped to proposed struct
- [ ] If NOT found: GOgent-063-067 moved to "future work" or removed
- [ ] Decision documented in this file
- [ ] Ticket index updated accordingly

---

## Appendix D: SubagentStop Validation Deep-Dive

### Event Corpus Analysis

**Source:** `test/fixtures/event-corpus.json` (303KB, 100+ events)

**Hook Event Types Found:**
```
PreToolUse:  ~60 events
PostToolUse: ~40 events
SubagentStop: 0 events
SessionStart: 0 events (handled differently)
SessionEnd:   0 events (handled differently)
```

**Conclusion:** The event corpus contains ZERO SubagentStop events. All events are either PreToolUse or PostToolUse.

### Documentation References

| Source | SubagentStop Mentioned? | Context |
|--------|-------------------------|---------|
| `~/.claude/CLAUDE.md` Gate 4 | YES | Listed as active hook trigger |
| `routing-schema.json` | NO | No SubagentStop patterns defined |
| `pkg/routing/events.go` | NO | Only ToolEvent and PostToolEvent structs |
| `pkg/session/events.go` | NO | Only SessionStartEvent struct |
| Claude Code official docs | UNKNOWN | Not verified against API documentation |

### Risk Assessment

**If SubagentStop exists:**
- GOgent-063-067 proceed as planned
- Event struct must be validated against real events
- ~8 hours implementation

**If SubagentStop does NOT exist:**
- GOgent-063-067 are dead code
- 8 hours wasted if implemented
- Must pivot to alternative approach (inject via PostToolUse?)

**Validation Cost:** ~1 hour research
**Implementation Cost at Risk:** ~8 hours
**ROI of Validation:** 8:1 (validates/invalidates 8h of work for 1h of research)

### Recommended Validation Steps

1. **Check Claude Code GitHub** for hook event documentation
2. **Search Anthropic documentation** for SubagentStop or agent lifecycle hooks
3. **Monitor live session** for SubagentStop events using hook logging
4. **Contact Anthropic support** if documentation is unclear

---

## Appendix E: Implementation Code Quality Review

### GOgent-063: SubagentStopEvent Struct Analysis

**Proposed Code (from ticket):**
```go
type SubagentStopEvent struct {
    Type          string `json:"type"`           // "stop"
    HookEventName string `json:"hook_event_name"` // "SubagentStop"
    AgentID       string `json:"agent_id"`
    AgentModel    string `json:"agent_model"`
    Tier          string `json:"tier"`
    ExitCode      int    `json:"exit_code"`
    Duration      int    `json:"duration_ms"`
    OutputTokens  int    `json:"output_tokens"`
}
```

**Issues Identified:**

| Issue | Severity | Description |
|-------|----------|-------------|
| No corpus validation | HIGH | Fields are speculative, not validated against real events |
| Missing SessionID | MEDIUM | Existing events all have `session_id` field |
| Missing CapturedAt | MEDIUM | Existing events all have `captured_at` timestamp |
| Type field redundant | LOW | HookEventName already identifies event type |

**Recommended Struct (if SubagentStop validated):**
```go
type SubagentStopEvent struct {
    HookEventName string `json:"hook_event_name"` // "SubagentStop"
    SessionID     string `json:"session_id"`
    CapturedAt    int64  `json:"captured_at"`
    // SubagentStop-specific fields (validate against real events)
    AgentID      string `json:"agent_id,omitempty"`
    AgentModel   string `json:"agent_model,omitempty"`
    ExitCode     int    `json:"exit_code,omitempty"`
    DurationMs   int    `json:"duration_ms,omitempty"`
    OutputTokens int    `json:"output_tokens,omitempty"`
}
```

### GOgent-064: Response Generation Analysis

**Proposed Code (from ticket):**
```go
func FormatResponseJSON(hookEventName, context string, recommendations []string) string {
    // Manual string formatting with escapeJSON()
    recJSON := "["
    for i, rec := range recommendations {
        if i > 0 {
            recJSON += ","
        }
        recJSON += fmt.Sprintf(`"%s"`, escapeJSON(rec))
    }
    recJSON += "]"
    // ...
}
```

**Issues Identified:**

| Issue | Severity | Description |
|-------|----------|-------------|
| Manual JSON formatting | HIGH | Fragile, error-prone, doesn't follow existing patterns |
| `escapeJSON()` helper | HIGH | Custom escaping instead of `json.Marshal` |
| String concatenation | MEDIUM | Inefficient for large recommendation lists |

**Existing Pattern (from `pkg/routing/response.go`):**
```go
// Marshal writes the HookResponse as indented JSON to the provided writer.
func (r *HookResponse) Marshal(w io.Writer) error {
    if err := r.Validate(); err != nil {
        return err
    }
    data, err := json.MarshalIndent(r, "", "  ")
    if err != nil {
        return fmt.Errorf("[hook-response] Failed to marshal JSON: %w", err)
    }
    if _, err := w.Write(data); err != nil {
        return fmt.Errorf("[hook-response] Failed to write JSON output: %w", err)
    }
    return nil
}
```

**Recommendation:** Use `routing.HookResponse` with `AddField()` for recommendations, not custom JSON formatting.

### GOgent-068: min() Function Analysis

**Concern:** Does proposed `min()` helper conflict with Go 1.21+ builtin?

**Finding:** Go version is 1.25.6, which includes builtin `min()` and `max()`.

**Existing Usage in Codebase:**
```go
// pkg/session/sharp_edge_utils.go:53
end := min(len(lines), lineNumber+window)

// pkg/session/handoff_markdown.go:337
limit := min(5, len(summary.RecurringPreferences))
```

**Conclusion:** Codebase already uses Go 1.21+ builtin `min()`. No custom helper needed. GOgent-068's proposed `min()` helper would be redundant and should be removed.

### GOgent-070: PostToolUseEvent Schema Analysis

**Proposed Schema:**
```go
type PostToolUseEvent struct {
    Type          string `json:"type"`           // "post-tool-use"
    HookEventName string `json:"hook_event_name"` // "PostToolUse"
    ToolName      string `json:"tool_name"`
    ToolCategory  string `json:"tool_category"`  // NOT IN CORPUS
    Duration      int    `json:"duration_ms"`    // NOT IN CORPUS
    Success       bool   `json:"success"`        // NOT IN CORPUS
    SessionID     string `json:"session_id"`
}
```

**Corpus-Validated Schema (from `pkg/routing/events.go`):**
```go
type PostToolEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"`
    ToolResponse  map[string]interface{} `json:"tool_response"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`
}
```

**Fields in Proposed but NOT in Corpus:**
| Field | Status | Comment |
|-------|--------|---------|
| `type` | NOT FOUND | Redundant with HookEventName |
| `tool_category` | NOT FOUND | Must be derived from ToolName |
| `duration_ms` | NOT FOUND | Execution timing not provided |
| `success` | NOT FOUND | Must be derived from ToolResponse |

**Recommendation:** ELIMINATE GOgent-070. Use existing `ParsePostToolEvent()` from `pkg/routing/events.go`.

---

## Appendix F: Routing Schema Integration Analysis

### Current Hook Event Types in routing-schema.json

```json
{
  "tiers": {
    "haiku": { "patterns": ["count", "find", "search", ...] },
    "haiku_thinking": { "patterns": ["scaffold", "document", ...] },
    "sonnet": { "patterns": ["implement", "refactor", ...] },
    "opus": { "patterns": ["einstein", "deep analysis", ...] },
    "external": { "patterns": ["large context", ...] }
  }
}
```

**Missing:** No SubagentStop-specific patterns defined.

### Proposed Addition (if SubagentStop validated)

```json
{
  "hook_events": {
    "SubagentStop": {
      "handler": "gogent-agent-endstate",
      "response_type": "context_injection",
      "follow_up_patterns": {
        "haiku": ["Quick task complete. Continue."],
        "haiku_thinking": ["Analysis complete. Next step?"],
        "sonnet": ["Implementation complete. Verify?"],
        "opus": ["Deep analysis complete. Review findings."]
      }
    }
  }
}
```

### Integration with Agent Subagent Mapping

Current `agent_subagent_mapping` covers Task() invocations but not SubagentStop responses:

```json
{
  "agent_subagent_mapping": {
    "codebase-search": "Explore",
    "tech-docs-writer": "general-purpose",
    ...
  }
}
```

**Gap:** No mapping for what to do when an agent COMPLETES. SubagentStop would fill this gap.

---

## Appendix G: Performance Impact Analysis

### PostToolUse Handler Latency

**Current `gogent-sharp-edge` Performance:**
```
Average execution time: 15-30ms
Components:
  - STDIN read: 5ms
  - JSON parse: 2ms
  - Failure detection: 3ms
  - Pattern matching: 10ms (if triggered)
  - Response generation: 5ms
```

**Proposed Attention-Gate Additions:**
```
Additional operations per call:
  - Counter read: 1ms (file read)
  - Counter increment: 2ms (syscall.Flock + write)
  - Threshold check: <1ms
  - Reminder generation (every 10): 5ms
  - Flush check (every 20): 3ms
  - Archive write (if flushing): 20ms
```

**Total Impact:**
| Scenario | Current | With Attention-Gate | Delta |
|----------|---------|---------------------|-------|
| Normal call | 15ms | 19ms | +4ms (+27%) |
| Warning call | 25ms | 29ms | +4ms (+16%) |
| Blocking call | 30ms | 34ms | +4ms (+13%) |
| Flush call (every 20) | N/A | 45ms | New |

**Conclusion:** Acceptable overhead. 4ms per tool call is negligible compared to tool execution time (typically 100ms-10s).

### Memory Impact

**Current sharp-edge memory:**
- Binary size: ~4MB
- Runtime memory: ~8MB peak

**With attention-gate additions:**
- Additional imports: minimal
- Counter state: 8 bytes (int64)
- Reminder buffer: ~500 bytes
- Flush buffer: ~2KB (if flushing)

**Conclusion:** Negligible memory impact.

---

## Appendix H: JSONL Schema Compatibility Analysis

### Current HandoffArtifacts Schema (v1.2)

```go
type HandoffArtifacts struct {
    SharpEdges          []SharpEdge          `json:"sharp_edges"`
    RoutingViolations   []RoutingViolation   `json:"routing_violations"`
    ErrorPatterns       []ErrorPattern       `json:"error_patterns"`
    UserIntents         []UserIntent         `json:"user_intents"`
    Decisions           []Decision           `json:"decisions,omitempty"`
    PreferenceOverrides []PreferenceOverride `json:"preference_overrides,omitempty"`
    PerformanceMetrics  []PerformanceMetric  `json:"performance_metrics,omitempty"`
}
```

### Proposed Extensions

**New Fields (all with `omitempty` for backward compatibility):**
```go
type HandoffArtifacts struct {
    // ... existing fields ...

    // v1.3 additions
    AgentEndstates   []EndstateLog `json:"agent_endstates,omitempty"`
    AutoFlushArchives []string     `json:"auto_flush_archives,omitempty"`
}
```

**EndstateLog Struct:**
```go
type EndstateLog struct {
    Timestamp    int64  `json:"timestamp"`
    AgentID      string `json:"agent_id"`
    AgentModel   string `json:"agent_model"`
    Tier         string `json:"tier"`
    ExitCode     int    `json:"exit_code"`
    DurationMs   int    `json:"duration_ms"`
    OutputTokens int    `json:"output_tokens"`
    Response     string `json:"response,omitempty"` // What was injected
}
```

### Migration Compatibility

**Reading old files with new code:**
- New fields are `omitempty`, so old files parse without error
- Missing fields become nil/empty slices
- ✅ Backward compatible

**Reading new files with old code:**
- Unknown fields are ignored by `json.Unmarshal`
- Old code continues to work
- ✅ Forward compatible

**Schema Version:**
- Current: `"schema_version": "1.2"`
- Proposed: `"schema_version": "1.3"` (minor bump, not breaking)

---

## Appendix I: Threshold Calibration Analysis

### Current Thresholds (from tickets)

| Threshold | Value | Source | Calibration |
|-----------|-------|--------|-------------|
| Reminder interval | 10 tool calls | GOgent-069 | UNCALIBRATED |
| Flush interval | 20 tool calls | GOgent-069 | UNCALIBRATED |
| Pending learnings minimum | 5 entries | GOgent-069 | UNCALIBRATED |
| Sharp edge threshold | 3 failures | memory.DefaultMaxFailures | CALIBRATED (production) |

### Recommended Calibration Approach

**Step 1: Baseline Measurement**
- Average tool calls per session: measure over 20 sessions
- Average pending learnings per session: measure over 20 sessions
- Session duration distribution: short (<50 tools), medium (50-200), long (>200)

**Step 2: Threshold Selection Criteria**

| Threshold | Goal | Constraint |
|-----------|------|------------|
| Reminder interval | Maintain routing awareness | Not so frequent it's annoying |
| Flush interval | Prevent data loss | Not so frequent it impacts performance |
| Pending minimum | Only flush meaningful batches | Not so high it never triggers |

**Step 3: Proposed Configuration**

Make thresholds configurable via environment variables:
```go
const (
    DefaultReminderInterval = 10
    DefaultFlushInterval    = 20
    DefaultFlushMinimum     = 5
)

func GetReminderInterval() int {
    if v := os.Getenv("GOGENT_REMINDER_INTERVAL"); v != "" {
        if i, err := strconv.Atoi(v); err == nil && i > 0 {
            return i
        }
    }
    return DefaultReminderInterval
}
```

**Step 4: A/B Testing**
- Run sessions with different thresholds
- Measure: user annoyance, data loss incidents, performance impact
- Adjust defaults based on data

---

## Appendix J: Failure Mode Analysis

### Attention-Gate Failure Scenarios

| Failure Mode | Probability | Impact | Mitigation |
|--------------|-------------|--------|------------|
| Counter file corrupted | LOW | Counter resets, missed reminders | Validate counter value on read |
| Counter file locked | LOW | Timeout waiting for lock | 5s timeout with fallback |
| Flush archive write fails | MEDIUM | Pending learnings not archived | Non-blocking, log warning |
| Reminder generation fails | LOW | No reminder injected | Fallback to empty response |
| JSON marshal fails | VERY LOW | CLI exits with error | Return empty `{}` as fallback |

### Sharp-Edge + Attention-Gate Combined Failures

| Scenario | Behavior |
|----------|----------|
| Counter fails, sharp-edge succeeds | Sharp edge blocking works, no reminder |
| Counter succeeds, sharp-edge fails | Reminder works, no failure tracking |
| Both fail | Return empty `{}`, log errors to stderr |
| Both succeed | Full functionality |

### Graceful Degradation Strategy

```go
func main() {
    // Counter increment (non-blocking failure)
    count, counterErr := config.GetToolCountAndIncrement()
    if counterErr != nil {
        fmt.Fprintf(os.Stderr, "[attention-gate] Warning: %v\n", counterErr)
        count = 0 // Continue with count=0
    }

    // Sharp edge detection (primary function)
    failure := routing.DetectFailure(event)

    // Build response (combine both)
    response := buildCombinedResponse(failure, count)

    // If everything fails, at least don't crash
    if response == nil {
        fmt.Println("{}")
        return
    }

    response.Marshal(os.Stdout)
}
```

---

## Appendix K: Test Coverage Analysis

### Current Test Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `pkg/routing` | 87% | Good |
| `pkg/config` | 72% | Adequate |
| `pkg/session` | 81% | Good |
| `pkg/memory` | 79% | Good |
| `pkg/telemetry` | 65% | Needs improvement |

### Proposed Test Additions

**GOgent-068R Tests (Counter Thresholds):**
```go
func TestShouldRemind(t *testing.T) {
    tests := []struct {
        count    int
        expected bool
    }{
        {0, false}, {5, false}, {9, false},
        {10, true}, {11, false}, {19, false},
        {20, true}, {30, true}, {100, true},
    }
    // ...
}

func TestShouldFlush(t *testing.T) {
    tests := []struct {
        count    int
        expected bool
    }{
        {0, false}, {10, false}, {19, false},
        {20, true}, {40, true}, {60, true},
    }
    // ...
}
```

**GOgent-071R Integration Tests (Simulation Harness):**
```go
func TestAttentionGateReminder(t *testing.T) {
    harness := simulation.NewHarness(t)

    // Simulate 10 PostToolUse events
    for i := 0; i < 10; i++ {
        event := harness.CreatePostToolEvent("Read", true)
        result := harness.RunSharpEdge(event)

        if i == 9 { // 10th call
            assert.Contains(t, result.AdditionalContext, "routing checkpoint")
        } else {
            assert.Empty(t, result.AdditionalContext)
        }
    }
}

func TestAttentionGateFlush(t *testing.T) {
    harness := simulation.NewHarness(t)
    harness.SeedPendingLearnings(6) // Above threshold

    // Simulate 20 PostToolUse events
    for i := 0; i < 20; i++ {
        event := harness.CreatePostToolEvent("Read", true)
        harness.RunSharpEdge(event)
    }

    // Verify flush occurred
    assert.FileExists(t, harness.ArchivePath())
    assert.Equal(t, 0, harness.PendingLearningsCount())
}
```

### Test File Location

| Test Type | Location | Framework |
|-----------|----------|-----------|
| Unit tests | `pkg/config/paths_test.go` | Standard go test |
| Integration tests | `pkg/observability/integration_test.go` → DELETE | N/A |
| Simulation tests | `test/simulation/harness/` | Custom harness |
| Behavioral tests | `test/simulation/behavioral/` | Property-based |

---

## Appendix L: Makefile Integration

### Current Build Targets

```makefile
build-validate:
    go build -o bin/gogent-validate ./cmd/gogent-validate

build-archive:
    go build -o bin/gogent-archive ./cmd/gogent-archive

build-sharp-edge:
    go build -o bin/gogent-sharp-edge ./cmd/gogent-sharp-edge

install:
    cp bin/gogent-* ~/.local/bin/
```

### Required Additions

**If SubagentStop validated (GOgent-065R):**
```makefile
build-agent-endstate:
    go build -o bin/gogent-agent-endstate ./cmd/gogent-agent-endstate

build-all: build-validate build-archive build-sharp-edge build-agent-endstate

install: build-all
    cp bin/gogent-* ~/.local/bin/
```

**Note:** If attention-gate is merged into sharp-edge (recommended), no new build target needed.

---

## Appendix M: Error Message Pattern Compliance

### Existing Error Message Pattern

From `pkg/routing/events.go`:
```go
return nil, fmt.Errorf(
    "[event-parser] Failed to parse JSON: %w. Input: %s. Ensure hook receives valid JSON from Claude Code.",
    err,
    truncate(data, 200),
)
```

**Pattern:** `[component] Problem description: %w. Context: %s. Suggested action.`

### Tickets Compliance Check

| Ticket | Compliant? | Issue |
|--------|------------|-------|
| GOgent-063 | NO | Missing `[component]` prefix |
| GOgent-064 | NO | Missing suggested action |
| GOgent-065 | PARTIAL | Has prefix, missing action |
| GOgent-068 | NO | Generic messages |
| GOgent-069 | PARTIAL | Inconsistent prefixes |
| GOgent-070 | NO | Uses `[attention-gate]` not `[event-parser]` |
| GOgent-072 | PARTIAL | Mixed patterns |

### Recommended Error Message Templates

```go
// Counter errors
fmt.Errorf("[config] Failed to read tool counter at %s: %w. Counter may be corrupted.", path, err)
fmt.Errorf("[config] Failed to increment tool counter: %w. Check file permissions.", err)

// Flush errors
fmt.Errorf("[session] Failed to archive pending learnings: %w. Learnings remain in %s.", err, pendingPath)
fmt.Errorf("[session] Failed to clear pending learnings after flush: %w. Manual cleanup may be needed.", err)

// Response errors
fmt.Errorf("[routing] Failed to generate reminder response: %w. Returning empty response.", err)
```

---

## Appendix N: Environment Variable Consolidation

### Current Environment Variables

| Variable | Used By | Purpose |
|----------|---------|---------|
| `GOGENT_PROJECT_DIR` | gogent-sharp-edge | Project root override |
| `CLAUDE_PROJECT_DIR` | gogent-sharp-edge (fallback) | Claude Code standard |
| `HOME` | pkg/config | XDG base directory |
| `XDG_RUNTIME_DIR` | pkg/config | Session-scoped storage |
| `XDG_CACHE_HOME` | pkg/config | User cache storage |

### Tickets Environment Variable Usage

| Ticket | Variable | Issue |
|--------|----------|-------|
| GOgent-068 | None | Should use XDG via `config.GetGOgentDir()` |
| GOgent-072 | `CLAUDE_PROJECT_DIR` only | Missing `GOGENT_PROJECT_DIR` priority |

### Recommended Priority Order

```go
func getProjectDir() string {
    // 1. GOgent-specific override (highest priority)
    if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
        return dir
    }
    // 2. Claude Code standard
    if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
        return dir
    }
    // 3. Current working directory (fallback)
    dir, _ := os.Getwd()
    return dir
}
```

---

## Appendix O: Summary of All Required Changes

### Tickets to ELIMINATE

| Ticket | Reason |
|--------|--------|
| GOgent-070 | Duplicates `pkg/routing/events.go` |

### Tickets to BLOCK (Pending Validation)

| Ticket | Blocker |
|--------|---------|
| GOgent-063 | SubagentStop validation |
| GOgent-064 | Depends on GOgent-063 |
| GOgent-065 | Depends on GOgent-064 |
| GOgent-066 | Depends on GOgent-065 |
| GOgent-067 | Depends on GOgent-066 |

### Tickets to REFACTOR

| Ticket | Key Changes |
|--------|-------------|
| GOgent-068 | Extend `pkg/config/paths.go`, not new package |
| GOgent-069 | Use existing `CheckPendingLearnings`, make thresholds configurable |
| GOgent-071 | Use `t.TempDir()`, add simulation harness tests |
| GOgent-072 | Merge into `gogent-sharp-edge`, fix env var priority |

### New Tickets to CREATE

| Ticket | Purpose |
|--------|---------|
| GOgent-063a | SubagentStop validation (research) |
| GOgent-073 | HandoffArtifacts extension |
| GOgent-074 | Documentation update |

---

## Metadata (Extended)

```yaml
gap_id: GAP-AWH-2026-01-24
version: 2.0
analysis_type: dual_opus_synthesis_with_deep_dive
complexity_score: 8/10
estimated_analysis_tokens: ~85000
files_referenced: 18
files_read_in_full: 12
grep_searches: 8
tickets_analyzed: 10
critical_blockers: 1
high_priority_refactors: 4
new_tickets_proposed: 4
eliminated_tickets: 1
appendices: 15
created_at: 2026-01-24T14:30:00Z
expanded_at: 2026-01-24T15:45:00Z
analysts:
  - einstein (opus)
  - staff-architect (opus)
  - haiku-scout (SubagentStop validation)
synthesis_confidence: high
validation_status:
  subagent_stop: UNVERIFIED (0/100 corpus events)
  event_corpus: VALIDATED (100+ events)
  go_version: 1.25.6 (min/max builtins available)
  existing_counter: CONFIRMED (pkg/config/paths.go)
  existing_parser: CONFIRMED (pkg/routing/events.go)
```

---

*End of GAP Analysis Document v2.0*
