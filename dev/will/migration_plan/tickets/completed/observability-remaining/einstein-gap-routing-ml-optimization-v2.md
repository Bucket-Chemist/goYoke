# GAP Analysis v2: GOgent-087 to GOgent-093 ML Telemetry Pipeline

**Document Type:** Einstein GAP Synthesis (Refactoring Authority)
**Version:** 2.0
**Generated:** 2026-01-25
**Analyses Synthesized:** Einstein (primary) + Staff-Architect (orthogonal)
**Purpose:** Authoritative refactoring guide for ticket series - addresses all blocking issues

---

## Executive Summary

The GOgent-087 to GOgent-093 ticket series implements an ML Telemetry Pipeline for routing optimization. The design captures tool events, routing decisions, and agent collaboration patterns for supervised learning.

### Critical Finding

**9 blocking issues** identified that prevent safe implementation:

| # | Issue | Resolution |
|---|-------|------------|
| 1 | Package fragmentation (observability vs telemetry) | Merge into pkg/telemetry |
| 2 | ToolEvent name collision with routing.ToolEvent | Extend routing.PostToolEvent |
| 3 | Tests call non-existent methods | Fix API references |
| 4 | Missing hook integration specs | Add 3 new integration tickets |
| 5 | XDG path inconsistency | Add GetGOgentDataDir() |
| 6 | Timestamp type mismatch (int64 vs time.Time) | Use time.Time |
| 7 | Missing SelectedTier/Agent fields | Add to struct |
| 8 | Duplicate PostToolUse hook | Merge into gogent-sharp-edge |
| 9 | Wrong binary name in docs | Update references |

### Recommendation

**DO NOT proceed with implementation until tickets are refactored per this guide.**

---

## Table of Contents

1. [Critical Issues - Detailed Analysis](#1-critical-issues---detailed-analysis)
2. [Existing Code Baseline](#2-existing-code-baseline)
3. [Required New Tickets](#3-required-new-tickets)
4. [Ticket Refactoring Specifications](#4-ticket-refactoring-specifications)
5. [Corrected Dependency Graph](#5-corrected-dependency-graph)
6. [Implementation Checklist](#6-implementation-checklist)

---

## 1. Critical Issues - Detailed Analysis

### 1.1 Package Fragmentation: BLOCKING

**Problem:** Tickets create conflicting package structure.

| Ticket | Creates | Location |
|--------|---------|----------|
| GOgent-087 | ToolEvent struct | `pkg/observability/` |
| GOgent-088 | ToolEventLog, LogToolEvent() | `pkg/observability/` |
| GOgent-089 | Integration tests | `pkg/observability/` |
| GOgent-087b | RoutingDecision | `pkg/telemetry/` |
| GOgent-087c | ClassifyTask() | `pkg/telemetry/` |
| GOgent-088b | AgentCollaboration | `pkg/telemetry/` |

**Impact:**
- 70%+ field overlap between `ToolEvent` and existing `AgentInvocation`
- Circular import risk: telemetry.ClassifyTask() called by observability.ToolEvent
- Duplicate dual-write implementations
- Maintenance burden of two similar packages

**Resolution:**
```
CANCEL pkg/observability creation
ALL ML telemetry code goes in pkg/telemetry
Extend existing routing.PostToolEvent with ML fields
```

**Affected Tickets:** GOgent-087, GOgent-088, GOgent-089

---

### 1.2 ToolEvent Name Collision: BLOCKING

**Problem:** GOgent-087 creates `ToolEvent` struct, but one already exists.

**Existing Definition** (`pkg/routing/events.go:16-22`):
```go
type ToolEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`
}
```

**Also exists** (`pkg/routing/events.go:26-33`):
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

**Resolution:**
```
Option A (RECOMMENDED): Extend routing.PostToolEvent with ML fields using omitempty
Option B: Create telemetry.MLToolMetrics as composition wrapper

DO NOT create new observability.ToolEvent
```

**Affected Tickets:** GOgent-087, GOgent-088, GOgent-089

---

### 1.3 Tests Call Non-Existent Methods: BLOCKING

**Problem:** GOgent-089 integration tests reference undefined methods.

**Called in GOgent-089 tests:**
```go
globalPath := event.GetGlobalPath()           // NOT DEFINED
projectPath := event.GetProjectPath()         // NOT DEFINED
shouldSkipProject := !event.ShouldWriteProjectPath()  // NOT DEFINED
```

**GOgent-087 actually defines:**
- `ParseToolEvent()`
- `TotalTokens()`
- `EstimatedCost()`

**Resolution:**
```
Either:
  A) Add missing methods to GOgent-087 specification
  B) Rewrite GOgent-089 tests to use correct API

RECOMMENDED: Option B - use telemetry package functions instead:
  - telemetry.GetGlobalMLLogPath()
  - telemetry.GetProjectMLLogPath(projectDir)
  - telemetry.ShouldWriteProjectLog(projectDir)
```

**Affected Tickets:** GOgent-087, GOgent-089

---

### 1.4 Missing Hook Integration Points: BLOCKING

**Problem:** Tickets define WHAT to log but never specify WHO calls the functions.

**Current Hook Handlers:**
| CLI | Hook Event | Current Behavior |
|-----|------------|------------------|
| `gogent-sharp-edge` | PostToolUse | Sharp-edge detection, attention-gate |
| `gogent-validate` | PreToolUse | Routing validation, tier checking |
| `gogent-agent-endstate` | SubagentStop | Agent completion logging |

**New Functions Without Callers:**
| Function | Expected Caller | Currently Specified? |
|----------|-----------------|---------------------|
| `LogToolEvent()` | PostToolUse | GOgent-090 creates NEW CLI |
| `LogRoutingDecision()` | PreToolUse (Task) | **NOT SPECIFIED** |
| `UpdateDecisionOutcome()` | PostToolUse | **NOT SPECIFIED** |
| `LogCollaboration()` | SubagentStop | **NOT SPECIFIED** |

**Resolution:**
```
Add 3 new integration tickets:
  GOgent-087d: Integrate LogMLToolEvent() into gogent-sharp-edge
  GOgent-087e: Integrate LogRoutingDecision() into gogent-validate
  GOgent-088c: Integrate LogCollaboration() into gogent-agent-endstate

DEPRECATE GOgent-090 (duplicate hook)
```

**Affected Tickets:** GOgent-087, GOgent-087b, GOgent-088, GOgent-088b, GOgent-090

---

### 1.5 XDG Path Inconsistency: NON-BLOCKING

**Problem:** Different XDG resolution between existing and new code.

**Existing** (`pkg/config/paths.go:25-63`):
```go
func GetGOgentDir() string {
    // Priority: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent
}
```

**New tickets specify:**
```go
xdgData := os.Getenv("XDG_DATA_HOME")
if xdgData == "" {
    xdgData = filepath.Join(home, ".local", "share")
}
return filepath.Join(xdgData, "gogent", "tool-events.jsonl")
```

**Result - Files in Different Locations:**
```
~/.cache/gogent/agent-invocations.jsonl      (existing - cache)
~/.local/share/gogent/tool-events.jsonl      (new - data)
```

**Resolution:**
```
Add config.GetGOgentDataDir() for persistent data files:
  - XDG_DATA_HOME > ~/.local/share/gogent
  - Use for ML telemetry (persistent training data)
  - Keep GetGOgentDir() for runtime/cache files
```

**Affected Tickets:** GOgent-087b, GOgent-088, GOgent-088b

---

### 1.6 Timestamp Type Mismatch: BLOCKING

**Problem:** Type conflict between definition and usage.

**GOgent-087b defines:**
```go
type RoutingDecision struct {
    Timestamp int64  // Unix epoch
}
```

**GOgent-089b export code calls:**
```go
formatted := d.Timestamp.Format(time.RFC3339)  // Requires time.Time!
```

**Resolution:**
```
Change GOgent-087b to use time.Time for consistency with AgentInvocation.Timestamp

type RoutingDecision struct {
    Timestamp time.Time `json:"timestamp"`
}
```

**Affected Tickets:** GOgent-087b, GOgent-089b

---

### 1.7 Missing SelectedTier/Agent Fields: BLOCKING

**Problem:** GOgent-088 accesses fields not defined in GOgent-087.

**GOgent-088 LogToolEvent() accesses:**
```go
log.SelectedTier = event.SelectedTier    // NOT in GOgent-087
log.SelectedAgent = event.SelectedAgent  // NOT in GOgent-087
```

**GOgent-087 ToolEvent fields (current spec):**
- SequenceIndex, PreviousTools, PreviousOutcomes
- TaskBatchID, IsRetry, RetryOf
- TaskType, TaskDomain
- TargetSize, CoverageAchieved, EntitiesFound
- **MISSING:** SelectedTier, SelectedAgent

**Resolution:**
```
Add to GOgent-087 (or GOgent-086b if extending PostToolEvent):
  SelectedTier   string `json:"selected_tier,omitempty"`
  SelectedAgent  string `json:"selected_agent,omitempty"`
```

**Affected Tickets:** GOgent-087, GOgent-088

---

### 1.8 Duplicate PostToolUse Hook: NON-BLOCKING

**Problem:** GOgent-090 creates new CLI that duplicates existing handler.

**Current:**
- `gogent-sharp-edge`: PostToolUse handler (sharp-edge, attention-gate)

**GOgent-090 creates:**
- `gogent-tool-event-logger`: NEW PostToolUse handler (ML logging)

**Impact:**
- Two hooks on same event
- Race conditions possible
- Configuration complexity

**Resolution:**
```
DEPRECATE GOgent-090
Merge ML logging into gogent-sharp-edge via GOgent-087d
```

**Affected Tickets:** GOgent-090

---

### 1.9 Wrong Binary Name in Docs: NON-BLOCKING

**Problem:** GOgent-093 references wrong binary name.

**GOgent-093 references:** `gogent-benchmark-logger`
**GOgent-090 creates:** `gogent-tool-event-logger`

**Resolution:**
```
Update GOgent-093 to reference gogent-sharp-edge (after merge)
```

**Affected Tickets:** GOgent-093

---

## 2. Existing Code Baseline

### 2.1 pkg/routing/events.go

**Types Available:**
| Type | Lines | Purpose |
|------|-------|---------|
| `ToolEvent` | 16-22 | PreToolUse events |
| `PostToolEvent` | 26-33 | PostToolUse events with response |
| `TaskInput` | 94-99 | Task tool parameters |
| `SubagentStopEvent` | 217-222 | Agent completion |
| `ParsedAgentMetadata` | 226-233 | Extracted agent info |

**Helper Methods (GOgent-080):**
| Method | Purpose |
|--------|---------|
| `ExtractFilePath()` | Get file_path from tool_input |
| `ExtractWriteContent()` | Get content for Write/Edit |
| `IsClaudeMDFile()` | Check if target is CLAUDE.md |
| `IsWriteOperation()` | Check if Write or Edit |

**Parsing Functions:**
| Function | Purpose |
|----------|---------|
| `ParseToolEvent()` | PreToolUse parsing |
| `ParsePostToolEvent()` | PostToolUse parsing |
| `ParseTaskInput()` | Task tool_input parsing |
| `ParseSubagentStopEvent()` | SubagentStop parsing |
| `ParseTranscriptForMetadata()` | Extract agent metadata |

### 2.2 pkg/telemetry/invocations.go

**Types Available:**
| Type | Lines | Purpose |
|------|-------|---------|
| `AgentInvocation` | 18-46 | Agent execution record |
| `AgentInvocationStats` | 159-171 | Aggregated metrics |
| `TierInvocationStats` | 174-182 | Tier-level metrics |
| `AgentRanking` | 185-189 | Sortable ranking |

**Path Functions:**
| Function | Returns |
|----------|---------|
| `GetInvocationsLogPath()` | Global path via config.GetGOgentDir() |
| `GetProjectInvocationsLogPath()` | Project-scoped path |

**Logging:**
| Function | Purpose |
|----------|---------|
| `LogInvocation()` | Dual-write (global + project) |
| `LoadInvocations()` | Read JSONL |

### 2.3 pkg/config/paths.go

**Current Functions:**
| Function | Purpose |
|----------|---------|
| `GetGOgentDir()` | XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent |
| `GetTierFilePath()` | current-tier state |
| `GetMaxDelegationPath()` | max_delegation ceiling |
| `GetViolationsLogPath()` | routing-violations.jsonl |
| `GetProjectViolationsLogPath()` | project-scoped violations |
| `GetToolCounterPath()` | tool counter |

**MISSING (Required by Tickets):**
```go
GetGOgentDataDir() // XDG_DATA_HOME > ~/.local/share/gogent
```

---

## 3. Required New Tickets

### 3.1 GOgent-086a: XDG Data Directory Helper

```yaml
---
id: GOgent-086a
title: Add config.GetGOgentDataDir() for XDG_DATA_HOME
type: implementation
status: pending
priority: high
estimated_time: 30m
dependencies: []
week: 4
---

## Description

Add XDG_DATA_HOME compliant directory helper for persistent data files.

## Rationale

ML telemetry files are persistent training data, not cache. Per XDG spec:
- XDG_CACHE_HOME: Non-essential cached data (current GetGOgentDir)
- XDG_DATA_HOME: Portable user data (needed for ML logs)

## Implementation

File: `pkg/config/paths.go`

```go
// GetGOgentDataDir returns XDG-compliant data directory for persistent files.
// Priority: XDG_DATA_HOME > ~/.local/share/gogent
// Use for: ML telemetry, training datasets, long-term logs
func GetGOgentDataDir() string {
    if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        if err := os.MkdirAll(dir, 0755); err == nil {
            return dir
        }
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return filepath.Join(os.TempDir(), "gogent-data")
    }
    dir := filepath.Join(home, ".local", "share", "gogent")
    os.MkdirAll(dir, 0755)
    return dir
}

// GetMLToolEventsPath returns path for ML tool events log.
func GetMLToolEventsPath() string {
    return filepath.Join(GetGOgentDataDir(), "tool-events.jsonl")
}

// GetRoutingDecisionsPath returns path for routing decisions log.
func GetRoutingDecisionsPath() string {
    return filepath.Join(GetGOgentDataDir(), "routing-decisions.jsonl")
}

// GetCollaborationsPath returns path for agent collaborations log.
func GetCollaborationsPath() string {
    return filepath.Join(GetGOgentDataDir(), "agent-collaborations.jsonl")
}
```

## Acceptance Criteria

- [ ] GetGOgentDataDir() implemented
- [ ] Respects XDG_DATA_HOME environment variable
- [ ] Falls back to ~/.local/share/gogent
- [ ] Creates directory if not exists
- [ ] Separate from GetGOgentDir() (data vs cache)
- [ ] Path helper functions for each log type
- [ ] Unit tests cover both XDG and fallback paths
- [ ] ≥90% coverage
```

---

### 3.2 GOgent-086b: Extend PostToolEvent with ML Fields

```yaml
---
id: GOgent-086b
title: Extend routing.PostToolEvent with ML telemetry fields
type: implementation
status: pending
priority: high
estimated_time: 45m
dependencies: []
week: 4
---

## Description

Add ML telemetry fields to existing PostToolEvent struct using omitempty for backward compatibility.

## Rationale

Instead of creating new ToolEvent in pkg/observability (which causes name collision), extend existing PostToolEvent. This:
- Avoids import cycles
- Leverages existing parsing
- Maintains backward compatibility via omitempty

## Implementation

File: `pkg/routing/events.go`

```go
type PostToolEvent struct {
    // Existing fields (DO NOT MODIFY)
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"`
    ToolResponse  map[string]interface{} `json:"tool_response"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`

    // === ML Telemetry Fields (GOgent-086b) ===
    // All omitempty for backward compatibility

    // Performance metrics
    DurationMs   int64 `json:"duration_ms,omitempty"`
    InputTokens  int   `json:"input_tokens,omitempty"`
    OutputTokens int   `json:"output_tokens,omitempty"`

    // Model context
    Model string `json:"model,omitempty"`
    Tier  string `json:"tier,omitempty"`

    // Outcome
    Success bool `json:"success,omitempty"`

    // Sequence tracking (GAP 4.2)
    SequenceIndex    int      `json:"sequence_index,omitempty"`
    PreviousTools    []string `json:"previous_tools,omitempty"`
    PreviousOutcomes []bool   `json:"previous_outcomes,omitempty"`

    // Task classification (GAP 4.4)
    TaskType   string `json:"task_type,omitempty"`
    TaskDomain string `json:"task_domain,omitempty"`

    // Routing info (for Task() events)
    SelectedTier  string `json:"selected_tier,omitempty"`
    SelectedAgent string `json:"selected_agent,omitempty"`

    // Correlation
    EventID string `json:"event_id,omitempty"`

    // Understanding context (Addendum A.4)
    TargetSize       int64   `json:"target_size,omitempty"`
    CoverageAchieved float64 `json:"coverage_achieved,omitempty"`
    EntitiesFound    int     `json:"entities_found,omitempty"`
}
```

## Acceptance Criteria

- [ ] ML fields added with omitempty tags
- [ ] Existing ParsePostToolEvent() unchanged (backward compat)
- [ ] All new fields are optional
- [ ] Existing tests still pass
- [ ] New unit tests for ML field access
- [ ] Documentation updated
- [ ] ≥90% coverage on new code
```

---

### 3.3 GOgent-087d: Hook Integration - Sharp Edge

```yaml
---
id: GOgent-087d
title: Integrate ML tool event logging into gogent-sharp-edge
type: implementation
status: pending
priority: high
estimated_time: 1h
dependencies: [GOgent-088]
week: 4
---

## Description

Add ML tool event logging to existing PostToolUse handler instead of creating separate CLI.

## Rationale

GOgent-090 was going to create gogent-tool-event-logger, but this duplicates gogent-sharp-edge. Merge functionality instead.

## Implementation

File: `cmd/gogent-sharp-edge/main.go`

After parsing PostToolEvent (around line 84), add:

```go
// Log ML tool event (GOgent-087d)
if err := telemetry.LogMLToolEvent(event, projectDir); err != nil {
    // Log error but don't fail hook - ML logging is non-critical
    fmt.Fprintf(os.Stderr, "[sharp-edge] ML logging warning: %v\n", err)
}
```

## Acceptance Criteria

- [ ] LogMLToolEvent() called on every PostToolUse
- [ ] Errors logged to stderr, hook continues (non-blocking)
- [ ] No performance regression (< 10ms added latency)
- [ ] Integration test verifies JSONL written
- [ ] Dual-write to global and project paths
- [ ] ≥80% coverage
```

---

### 3.4 GOgent-087e: Hook Integration - Validate

```yaml
---
id: GOgent-087e
title: Integrate routing decision logging into gogent-validate
type: implementation
status: pending
priority: high
estimated_time: 1h
dependencies: [GOgent-087b]
week: 4
---

## Description

Log routing decisions when Task() tool is invoked via PreToolUse.

## Implementation

File: `cmd/gogent-validate/main.go`

On PreToolUse for Task tool:

```go
// Log routing decision for Task() calls (GOgent-087e)
if event.ToolName == "Task" {
    taskInput, err := routing.ParseTaskInput(event.ToolInput)
    if err == nil {
        decision := telemetry.NewRoutingDecision(
            event.SessionID,
            taskInput.Prompt,
            taskInput.Model,      // SelectedTier
            extractAgentFromPrompt(taskInput.Prompt), // SelectedAgent
        )
        if err := telemetry.LogRoutingDecision(decision); err != nil {
            fmt.Fprintf(os.Stderr, "[validate] Routing decision logging warning: %v\n", err)
        }
    }
}
```

## Acceptance Criteria

- [ ] LogRoutingDecision() called on every Task() PreToolUse
- [ ] DecisionID generated (UUID)
- [ ] TaskDescription extracted from tool_input.prompt
- [ ] SelectedTier extracted from tool_input.model
- [ ] SelectedAgent extracted from AGENT: prefix in prompt
- [ ] Non-blocking (errors logged, hook continues)
- [ ] ≥80% coverage
```

---

### 3.5 GOgent-088c: Hook Integration - Agent Endstate

```yaml
---
id: GOgent-088c
title: Integrate collaboration logging into gogent-agent-endstate
type: implementation
status: pending
priority: high
estimated_time: 1h
dependencies: [GOgent-088b]
week: 4
---

## Description

Log agent collaboration when subagent completes via SubagentStop.

## Implementation

File: `cmd/gogent-agent-endstate/main.go`

After parsing SubagentStopEvent:

```go
// Log collaboration (GOgent-088c)
metadata, _ := routing.ParseTranscriptForMetadata(event.TranscriptPath)

collab := telemetry.NewAgentCollaboration(
    event.SessionID,
    "terminal",           // Parent (terminal is always parent in this context)
    metadata.AgentID,     // Child
    "spawn",              // DelegationType
)
collab.ChildSuccess = metadata.IsSuccess()
collab.ChildDurationMs = int64(metadata.DurationMs)
collab.ChainDepth = 1 // Root delegation

if err := telemetry.LogCollaboration(collab); err != nil {
    fmt.Fprintf(os.Stderr, "[agent-endstate] Collaboration logging warning: %v\n", err)
}
```

## Acceptance Criteria

- [ ] LogCollaboration() called on every SubagentStop
- [ ] Parent agent derived from session context
- [ ] Child agent derived from transcript metadata
- [ ] Success/duration captured from ParsedAgentMetadata
- [ ] Non-blocking (errors logged, hook continues)
- [ ] ≥80% coverage
```

---

## 4. Ticket Refactoring Specifications

### 4.1 GOgent-087: REWRITE

**Original Scope:** Create `pkg/observability/tool_event.go`

**Revised Scope:** ML helper functions in `pkg/telemetry`, NO new package

```yaml
---
id: GOgent-087
title: ML Tool Event Helper Functions (REVISED)
type: implementation
status: pending
priority: high
estimated_time: 2h
dependencies: [GOgent-086b]
week: 4
---

## IMPORTANT: Scope Change

DO NOT create pkg/observability.
Use extended routing.PostToolEvent from GOgent-086b.
Add helper functions to pkg/telemetry.

## Implementation

File: `pkg/telemetry/ml_tool_event.go`

```go
package telemetry

import (
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// TotalTokens returns sum of input and output tokens.
func TotalTokens(event *routing.PostToolEvent) int {
    return event.InputTokens + event.OutputTokens
}

// EstimatedCost calculates approximate cost based on tier and tokens.
func EstimatedCost(event *routing.PostToolEvent) float64 {
    inputCost := map[string]float64{
        "haiku":  0.00025,
        "sonnet": 0.003,
        "opus":   0.015,
    }
    outputCost := map[string]float64{
        "haiku":  0.00125,
        "sonnet": 0.015,
        "opus":   0.075,
    }
    tier := event.Tier
    if tier == "" {
        tier = "sonnet"
    }
    return (float64(event.InputTokens) * inputCost[tier] / 1000) +
           (float64(event.OutputTokens) * outputCost[tier] / 1000)
}

// EnrichWithSequence adds sequence tracking to event.
func EnrichWithSequence(event *routing.PostToolEvent, index int, previous []string, outcomes []bool) {
    event.SequenceIndex = index
    event.PreviousTools = previous
    event.PreviousOutcomes = outcomes
}

// EnrichWithClassification adds task classification to event.
func EnrichWithClassification(event *routing.PostToolEvent) {
    if event.TaskType == "" || event.TaskDomain == "" {
        taskType, taskDomain := ClassifyTask(extractDescription(event))
        event.TaskType = taskType
        event.TaskDomain = taskDomain
    }
}
```

## Acceptance Criteria

- [ ] NO pkg/observability created
- [ ] TotalTokens() helper function
- [ ] EstimatedCost() helper function with tier-based pricing
- [ ] EnrichWithSequence() for sequence tracking
- [ ] EnrichWithClassification() using ClassifyTask()
- [ ] Uses routing.PostToolEvent (not new struct)
- [ ] ≥80% coverage
- [ ] No races
```

---

### 4.2 GOgent-087b: FIX

**Issue:** Timestamp type mismatch

**Changes Required:**

```diff
type RoutingDecision struct {
-   Timestamp   int64  `json:"timestamp"`
+   Timestamp   time.Time `json:"timestamp"`

+   // Correlation
+   EventID     string `json:"event_id"`
}
```

**Add to Acceptance Criteria:**
- [ ] Timestamp uses time.Time (not int64)
- [ ] EventID field for correlation with ToolEvent
- [ ] Uses config.GetGOgentDataDir() for path
- [ ] Format timestamp as RFC3339 in JSON

---

### 4.3 GOgent-088: REWRITE

**Original Scope:** Create `pkg/observability/benchmark_logger.go`

**Revised Scope:** ML logging in `pkg/telemetry`, NO new package

```yaml
---
id: GOgent-088
title: ML Tool Event Logging (REVISED)
type: implementation
status: pending
priority: high
estimated_time: 1.5h
dependencies: [GOgent-087, GOgent-086a]
week: 4
---

## IMPORTANT: Scope Change

DO NOT create pkg/observability.
Implement LogMLToolEvent() in pkg/telemetry.

## Implementation

File: `pkg/telemetry/ml_logging.go`

```go
package telemetry

import (
    "encoding/json"
    "os"
    "path/filepath"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// LogMLToolEvent writes ML-enriched tool event to JSONL.
// Dual-write: global (XDG_DATA_HOME) + project (.claude/memory/).
func LogMLToolEvent(event *routing.PostToolEvent, projectDir string) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    data = append(data, '\n')

    // Global write (required)
    globalPath := config.GetMLToolEventsPath()
    if err := appendToFile(globalPath, data); err != nil {
        return err
    }

    // Project write (optional)
    if projectDir != "" {
        projectPath := filepath.Join(projectDir, ".claude", "memory", "tool-events.jsonl")
        if err := appendToFile(projectPath, data); err != nil {
            // Log warning, don't fail
            fmt.Fprintf(os.Stderr, "[telemetry] Project log warning: %v\n", err)
        }
    }

    return nil
}

// ReadMLToolEvents loads tool events from JSONL.
func ReadMLToolEvents(path string) ([]routing.PostToolEvent, error) {
    // Implementation similar to LoadInvocations()
}

// CalculateMLSessionStats aggregates ML metrics.
func CalculateMLSessionStats(events []routing.PostToolEvent) map[string]interface{} {
    // Aggregate: total cost, tool counts, success rate, etc.
}
```

## Acceptance Criteria

- [ ] NO pkg/observability created
- [ ] LogMLToolEvent() in pkg/telemetry
- [ ] Uses config.GetGOgentDataDir() for global path
- [ ] Dual-write pattern (global + project)
- [ ] ReadMLToolEvents() for reading logs
- [ ] CalculateMLSessionStats() for aggregation
- [ ] ≥80% coverage
```

---

### 4.4 GOgent-088b: ADD FIELDS

**Issue:** Missing swarm coordination fields, missing EventID

**Add to struct:**

```go
type AgentCollaboration struct {
    // ... existing fields ...

    // Correlation (ADD)
    EventID string `json:"event_id"`

    // Swarm coordination (Addendum A.3)
    IsSwarmMember         bool    `json:"is_swarm_member,omitempty"`
    SwarmPosition         int     `json:"swarm_position,omitempty"`
    OverlapWithPrevious   float64 `json:"overlap_with_previous,omitempty"`
    AgreementWithAdjacent float64 `json:"agreement_with_adjacent,omitempty"`
    InformationLoss       float64 `json:"information_loss,omitempty"`
}
```

---

### 4.5 GOgent-089: REWRITE TESTS

**Issue:** Tests call non-existent methods

**Changes Required:**

```diff
- globalPath := event.GetGlobalPath()
+ globalPath := config.GetMLToolEventsPath()

- projectPath := event.GetProjectPath()
+ projectPath := filepath.Join(projectDir, ".claude", "memory", "tool-events.jsonl")

- shouldSkipProject := !event.ShouldWriteProjectPath()
+ shouldSkipProject := projectDir == ""
```

**Update Acceptance Criteria:**
- [ ] Tests use telemetry package functions (not observability)
- [ ] Tests use routing.PostToolEvent (not observability.ToolEvent)
- [ ] Tests use config.GetMLToolEventsPath() for paths
- [ ] Remove ALL references to pkg/observability

---

### 4.6 GOgent-089b: FIX TYPE

**Issue:** Timestamp.Format() requires time.Time

**Changes Required:**

```diff
- formatted := d.Timestamp.Format(time.RFC3339)
+ // Timestamp is now time.Time per GOgent-087b fix
+ formatted := d.Timestamp.Format(time.RFC3339)
```

No code change needed IF GOgent-087b is fixed first.

**Add to Dependencies:**
```yaml
dependencies: [GOgent-089, GOgent-087b]  # Added 087b
```

---

### 4.7 GOgent-090: DEPRECATE

**Reason:** Functionality merged into GOgent-087d

```yaml
---
id: GOgent-090
title: DEPRECATED - Build gogent-tool-event-logger CLI
type: deprecated
status: cancelled
---

## Deprecation Notice

This ticket has been CANCELLED.

Functionality merged into GOgent-087d which integrates ML logging
into existing gogent-sharp-edge PostToolUse handler.

## Rationale

Creating a separate CLI would:
- Duplicate PostToolUse handling
- Create potential race conditions
- Increase configuration complexity

## Resolution

- GOgent-087d implements equivalent functionality
- Remove GOgent-090 from dependency graph
- Update any tickets that depend on GOgent-090
```

---

### 4.8 GOgent-091: REMOVE DEPENDENCY

**Issue:** Investigation should be independent

**Change:**
```yaml
dependencies: []  # Was: [GOgent-088b]
```

Investigation can proceed in parallel with implementation.

---

### 4.9 GOgent-093: FIX REFERENCES

**Issue:** Wrong binary name

**Changes Required:**
- Replace `gogent-benchmark-logger` with `gogent-sharp-edge`
- Update all binary references to reflect merged functionality

---

## 5. Corrected Dependency Graph

```
                    GOgent-069 (existing)
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
        GOgent-086a                GOgent-086b
   (GetGOgentDataDir)          (Extend PostToolEvent)
              │                         │
              └────────────┬────────────┘
                           ▼
                      GOgent-087
               (ML telemetry helpers)
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
        GOgent-087c                GOgent-088
       (ClassifyTask)           (LogMLToolEvent)
              │                         │
              ▼                         ▼
        GOgent-087b                GOgent-088b
     (RoutingDecision)         (Collaboration)
              │                         │
              └────────────┬────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        GOgent-087d   GOgent-087e   GOgent-088c
    (sharp-edge int) (validate int) (endstate int)
              │            │            │
              └────────────┴────────────┘
                           │
                           ▼
                      GOgent-089
                   (Integration tests)
                           │
                           ▼
                      GOgent-089b
                      (ML Export)
                           │
                           ▼
        ┌──────────────────┴──────────────────┐
        ▼                                     ▼
   GOgent-091                            GOgent-093
(Stop-gate investigation)             (Final docs)
        │
        ▼
   GOgent-092
(Translation/deprecation)
```

**Key Changes:**
- Added GOgent-086a, GOgent-086b as prerequisites
- Added GOgent-087d, GOgent-087e, GOgent-088c for hook integration
- Removed GOgent-090 (deprecated)
- GOgent-091 is now independent (no dependencies)
- GOgent-093 depends on GOgent-089b (not GOgent-092)

---

## 6. Implementation Checklist

### Pre-Implementation Gate

Before ANY ticket work begins:

- [ ] GOgent-086a implemented and merged (XDG data path)
- [ ] GOgent-086b implemented and merged (PostToolEvent extension)
- [ ] All existing tests still pass
- [ ] No pkg/observability directory exists

### Per-Ticket Verification

For each ticket implementation:

- [ ] No `pkg/observability` references anywhere
- [ ] Uses `config.GetGOgentDataDir()` for data paths
- [ ] Uses `routing.PostToolEvent` (not new ToolEvent)
- [ ] Hook caller explicitly integrated
- [ ] Timestamp type is `time.Time`
- [ ] EventID field present for correlation
- [ ] Dependencies match corrected graph
- [ ] Tests use actual API (not undefined methods)
- [ ] ≥80% test coverage
- [ ] No race conditions (`go test -race`)

### Post-Implementation Verification

After all tickets complete:

- [ ] `~/.local/share/gogent/tool-events.jsonl` populated
- [ ] `~/.local/share/gogent/routing-decisions.jsonl` populated
- [ ] `~/.local/share/gogent/agent-collaborations.jsonl` populated
- [ ] Project-scoped mirrors in `.claude/memory/`
- [ ] gogent-ml-export CLI functional
- [ ] All acceptance criteria from GAP v1 still met

---

## Summary: Action Items

### Immediate (Before Implementation)

1. **Create GOgent-086a** - XDG data path helper
2. **Create GOgent-086b** - PostToolEvent ML fields
3. **Create GOgent-087d** - Sharp-edge hook integration
4. **Create GOgent-087e** - Validate hook integration
5. **Create GOgent-088c** - Endstate hook integration

### Refactor Existing Tickets

6. **Rewrite GOgent-087** - Use telemetry, not observability
7. **Fix GOgent-087b** - time.Time timestamp, add EventID
8. **Rewrite GOgent-088** - Use telemetry, not observability
9. **Fix GOgent-088b** - Add EventID, swarm fields
10. **Rewrite GOgent-089** - Fix method references
11. **Deprecate GOgent-090** - Merge into GOgent-087d
12. **Fix GOgent-091** - Remove dependency
13. **Fix GOgent-093** - Correct binary names

---

**Document End**

*This GAP v2 document supersedes einstein-gap-routing-ml-optimization.md*
*Archive both to `.claude/gap_logger/` after implementation begins*
