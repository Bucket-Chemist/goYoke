# Agent-Workflow-Hooks Ticket Refactoring Map

**Generated:** 2026-01-24
**Authorized By:** Einstein (Opus) + Orchestrator synthesis
**Source Documents:**
- GAP-ANALYSIS-GOgent-063-072.md
- einstein-subagent-stop-research-2026-01-24.md
- Existing tickets directory

---

## REFACTORING PROGRESS (Updated: 2026-01-24 14:30)

### ✅ Completed (6/13)
1. **GOgent-063.md** - UPDATED with actual SubagentStop schema + transcript parsing
2. **GOgent-063a.md** - CREATED (validation ticket, marked completed)
3. **GOgent-064.md** - UPDATED with (event, metadata) signature + graceful degradation
4. **GOgent-070.md** - DELETED (duplicated pkg/routing/events.go)
5. **GOgent-073.md** - CREATED (HandoffArtifacts extension spec)
6. **GOgent-074.md** - CREATED (documentation update spec)

### 🔄 Remaining (7/13)
- **GOgent-065** - XDG paths (replace `/tmp/`) + HandoffArtifacts integration
- **GOgent-066** - Test schema updates (actual SubagentStop schema) + t.TempDir()
- **GOgent-067** - Add transcript parsing step + Makefile target
- **GOgent-068** - Location change (pkg/config NOT new package) + threshold functions
- **GOgent-069** - Location change (pkg/session NOT new package) + env config
- **GOgent-071** - t.TempDir() pattern + simulation harness
- **GOgent-072** - Merge into sharp-edge (NO new CLI) + env var priority

### Next Actions
**For remaining tickets, apply changes from Section 2 below:**
- Read Section 2 entry for ticket number
- Apply BEFORE → AFTER changes specified
- Update acceptance criteria as listed
- Move to next ticket

---

## Executive Summary

| Metric | Count |
|--------|-------|
| Total existing tickets | 10 (GOgent-063 to GOgent-072) |
| Tickets to ELIMINATE | 1 (GOgent-070) |
| Tickets to REFACTOR (schema correction) | 5 (GOgent-063, GOgent-064, GOgent-065, GOgent-066, GOgent-067) |
| Tickets to REFACTOR (code deduplication) | 4 (GOgent-068, GOgent-069, GOgent-071, GOgent-072) |
| New tickets to ADD | 3 (GOgent-063a, GOgent-073, GOgent-074) |

**Critical Finding:** SubagentStop is CONFIRMED to exist but has a different schema than originally speculated. All agent-endstate tickets (063-067) require schema correction.

---

## Section 1: Tickets to ELIMINATE

### GOgent-070: PostToolUse Event Parsing

**Status:** ELIMINATE

**Reason:** Duplicates existing validated code in `pkg/routing/events.go`

**Evidence:**
```
PROPOSED (GOgent-070)                    EXISTING (USE INSTEAD)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
pkg/observability/events.go              pkg/routing/events.go
├─ PostToolUseEvent struct               ├─ PostToolEvent struct (VALIDATED)
│   ├─ type (NOT IN CORPUS)              │   ├─ tool_name
│   ├─ tool_category (NOT IN CORPUS)     │   ├─ tool_input
│   ├─ duration_ms (NOT IN CORPUS)       │   ├─ tool_response
│   └─ success (NOT IN CORPUS)           │   ├─ session_id
│                                        │   ├─ hook_event_name
│                                        │   └─ captured_at
└─ ParsePostToolUseEvent()               └─ ParsePostToolEvent() (VALIDATED)
```

**Where Functionality Goes:**
- GOgent-072 (CLI) should import `pkg/routing.ParsePostToolEvent()` directly
- No new event parsing code needed for attention-gate

**Action:** Delete ticket file. Update any references in GOgent-072.

---

## Section 2: Tickets to REFACTOR

### 2.1 Agent-Endstate Tickets (SubagentStop Schema Correction)

**Critical Issue:** All tickets GOgent-063 through GOgent-067 were based on a SPECULATED SubagentStop schema that does not match the ACTUAL Claude Code schema.

#### SubagentStop Schema Correction (APPLIES TO ALL)

**BEFORE (Speculated - WRONG):**
```go
type SubagentStopEvent struct {
    Type          string `json:"type"`           // "stop"
    HookEventName string `json:"hook_event_name"` // "SubagentStop"
    AgentID       string `json:"agent_id"`       // DOES NOT EXIST
    AgentModel    string `json:"agent_model"`    // DOES NOT EXIST
    Tier          string `json:"tier"`           // DOES NOT EXIST
    ExitCode      int    `json:"exit_code"`      // DOES NOT EXIST
    Duration      int    `json:"duration_ms"`    // DOES NOT EXIST
    OutputTokens  int    `json:"output_tokens"`  // DOES NOT EXIST
}
```

**AFTER (Actual - CORRECT):**
```go
type SubagentStopEvent struct {
    HookEventName   string `json:"hook_event_name"`  // "SubagentStop"
    SessionID       string `json:"session_id"`
    TranscriptPath  string `json:"transcript_path"`
    StopHookActive  bool   `json:"stop_hook_active"`
}

// DerivedAgentMetadata is extracted from transcript parsing (NOT from event)
type DerivedAgentMetadata struct {
    AgentID      string
    AgentModel   string
    Tier         string
    DurationMs   int
    OutputTokens int
    ExitCode     int  // Derive from completion status in transcript
}
```

---

### GOgent-063: Define SubagentStop Event Structs

**Status:** REFACTOR - Schema Correction Required

**Location:** `pkg/workflow/events.go` OR `pkg/routing/events.go` (prefer extending existing)

**Changes Required:**

1. **Replace struct definition** with actual schema
2. **Add transcript parsing function** to extract agent metadata
3. **Update validation** to check actual required fields (session_id, transcript_path)
4. **Remove min() helper** (Go 1.25 has builtin)
5. **Keep GetAgentClass()** function but document it operates on derived metadata

**BEFORE (lines 39-49):**
```go
type SubagentStopEvent struct {
    Type          string `json:"type"`
    HookEventName string `json:"hook_event_name"`
    AgentID       string `json:"agent_id"`
    AgentModel    string `json:"agent_model"`
    Tier          string `json:"tier"`
    ExitCode      int    `json:"exit_code"`
    Duration      int    `json:"duration_ms"`
    OutputTokens  int    `json:"output_tokens"`
}
```

**AFTER:**
```go
// SubagentStopEvent represents the actual Claude Code SubagentStop hook event.
// Agent metadata is NOT directly available - must parse transcript file.
type SubagentStopEvent struct {
    HookEventName   string `json:"hook_event_name"`  // "SubagentStop"
    SessionID       string `json:"session_id"`
    TranscriptPath  string `json:"transcript_path"`
    StopHookActive  bool   `json:"stop_hook_active"`
}

// ParsedAgentMetadata contains agent info extracted from transcript.
// All fields are optional as transcript parsing may fail.
type ParsedAgentMetadata struct {
    AgentID      string `json:"agent_id,omitempty"`
    AgentModel   string `json:"agent_model,omitempty"`
    Tier         string `json:"tier,omitempty"`
    DurationMs   int    `json:"duration_ms,omitempty"`
    OutputTokens int    `json:"output_tokens,omitempty"`
    ExitCode     int    `json:"exit_code,omitempty"`
}

// ParseTranscriptForMetadata reads transcript file and extracts agent metadata.
// Returns partial metadata on parsing errors rather than failing completely.
func ParseTranscriptForMetadata(transcriptPath string) (*ParsedAgentMetadata, error) {
    // Implementation: Read transcript, extract AGENT: line, model, timing
    // Graceful degradation: return defaults if parsing fails
}
```

**Test Updates:**
- Update test JSON to use actual schema (session_id, transcript_path)
- Add transcript parsing tests
- Remove timeout test with fake blocking reader (pattern already validated in pkg/routing)

**Acceptance Criteria Updates:**
- [ ] Uses ACTUAL SubagentStop schema (session_id, transcript_path, hook_event_name, stop_hook_active)
- [ ] Implements transcript parsing for agent metadata extraction
- [ ] `GetAgentClass()` works on parsed metadata
- [ ] Graceful degradation when transcript parsing fails
- [ ] Remove redundant `min()` helper (use Go builtin)

---

### GOgent-064: Tier-Specific Response Generation

**Status:** REFACTOR - Adapt to Transcript-Based Metadata

**Changes Required:**

1. **Update function signature** to accept parsed metadata OR event + parsed metadata
2. **Handle missing metadata gracefully** (generic response if parsing failed)
3. **Replace manual JSON formatting** with json.Marshal (follow existing pattern)
4. **Use routing.HookResponse** for output formatting

**BEFORE (lines 48-49):**
```go
func GenerateEndstateResponse(event *SubagentStopEvent) *EndstateResponse {
    agentClass := event.GetAgentClass()
```

**AFTER:**
```go
// GenerateEndstateResponse creates tier-specific response based on agent completion.
// If metadata is nil, generates generic response.
func GenerateEndstateResponse(event *SubagentStopEvent, metadata *ParsedAgentMetadata) *EndstateResponse {
    // If metadata is nil, use defaults
    if metadata == nil {
        metadata = &ParsedAgentMetadata{
            AgentID: "unknown",
            Tier:    "unknown",
        }
    }
    agentClass := GetAgentClass(metadata.AgentID)
```

**Additional Changes:**

Replace custom JSON formatting:
```go
// REMOVE this (lines 152-194):
func (r *EndstateResponse) FormatJSON() string { ... }
func escapeJSON(s string) string { ... }
func formatRecommendations(recs []string) string { ... }

// REPLACE with:
func (r *EndstateResponse) Marshal(w io.Writer) error {
    return json.NewEncoder(w).Encode(r)
}
// OR integrate with existing routing.HookResponse pattern
```

---

### GOgent-065: Endstate Logging & Decision Storage

**Status:** REFACTOR - Path Correction + HandoffArtifacts Integration

**Changes Required:**

1. **Replace hardcoded `/tmp/` path** with XDG-compliant path
2. **Add to HandoffArtifacts** schema (new field)
3. **Use existing JSONL patterns** from pkg/session/

**BEFORE (line 54):**
```go
logPath := "/tmp/claude-agent-endstates.jsonl"
```

**AFTER:**
```go
import "github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"

func GetEndstateLogPath() string {
    return filepath.Join(config.GetGOgentDir(), "agent-endstates.jsonl")
}

// OR for project-scoped (dual-write pattern):
func GetProjectEndstateLogPath(projectDir string) string {
    return filepath.Join(projectDir, ".claude", "memory", "agent-endstates.jsonl")
}
```

**HandoffArtifacts Integration:**
```go
// Add to HandoffArtifacts struct in pkg/session/handoff.go:
type HandoffArtifacts struct {
    // ... existing fields ...

    // v1.3 additions (omitempty for backward compatibility)
    AgentEndstates []EndstateLog `json:"agent_endstates,omitempty"`
}
```

---

### GOgent-066: Integration Tests for agent-endstate

**Status:** REFACTOR - Simulation Harness Integration

**Changes Required:**

1. **Use `t.TempDir()`** instead of hardcoded paths (prevents global state pollution)
2. **Add simulation harness tests** following existing patterns
3. **Update test JSON** to use actual SubagentStop schema

**BEFORE (lines 37-45):**
```go
eventJSON := `{
    "type": "stop",
    "hook_event_name": "SubagentStop",
    "agent_id": "orchestrator",
    ...
}`
```

**AFTER:**
```go
eventJSON := `{
    "hook_event_name": "SubagentStop",
    "session_id": "test-session-123",
    "transcript_path": "/tmp/test-transcript.jsonl",
    "stop_hook_active": true
}`

// Create mock transcript file for metadata extraction
transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
createMockTranscript(t, transcriptPath, "orchestrator", "sonnet")
```

---

### GOgent-067: Build gogent-agent-endstate CLI

**Status:** REFACTOR - Transcript Parsing Integration

**Changes Required:**

1. **Add transcript parsing step** after event parsing
2. **Update import path** (verify correct module path)
3. **Add Makefile target** for build
4. **Handle parsing failures gracefully**

**BEFORE (lines 44-61):**
```go
func main() {
    event, err := workflow.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
    if err != nil {
        outputError(...)
        os.Exit(1)
    }
    response := workflow.GenerateEndstateResponse(event)
    ...
}
```

**AFTER:**
```go
func main() {
    event, err := workflow.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
    if err != nil {
        outputError(...)
        os.Exit(1)
    }

    // NEW: Parse transcript for agent metadata
    metadata, parseErr := workflow.ParseTranscriptForMetadata(event.TranscriptPath)
    if parseErr != nil {
        fmt.Fprintf(os.Stderr, "Warning: Failed to parse transcript: %v\n", parseErr)
        // Continue with nil metadata - will use defaults
    }

    response := workflow.GenerateEndstateResponse(event, metadata)
    ...
}
```

**Add to Makefile:**
```makefile
build-agent-endstate:
    go build -o bin/gogent-agent-endstate ./cmd/gogent-agent-endstate

build-all: build-validate build-archive build-sharp-edge build-agent-endstate
```

---

### GOgent-068: Tool Counter Management

**Status:** REFACTOR - Extend Existing pkg/config/paths.go

**Critical Issue:** This ticket proposes creating `pkg/observability/counter.go` which duplicates existing code in `pkg/config/paths.go`.

**Existing Code (pkg/config/paths.go lines 80-168):**
- `GetToolCounterPath()` - Already exists
- `InitializeToolCounter()` - Already exists
- `GetToolCount()` - Already exists
- `IncrementToolCount()` - Already exists (with `syscall.Flock()` atomicity)

**Changes Required:**

1. **DO NOT create pkg/observability/counter.go**
2. **Extend pkg/config/paths.go** with threshold functions only
3. **Remove mutex-based ToolCounter struct** (existing flock is better)

**ADD to pkg/config/paths.go:**
```go
const (
    ReminderInterval = 10  // Inject reminder every N tools
    FlushInterval    = 20  // Flush learnings every N tools
)

// GetToolCountAndIncrement atomically reads current count and increments.
// Returns the count AFTER incrementing.
func GetToolCountAndIncrement() (int, error) {
    // Use existing flock pattern from IncrementToolCount
    // But return the new count value
}

// ShouldRemind returns true if reminder should be injected.
func ShouldRemind(count int) bool {
    return count > 0 && count%ReminderInterval == 0
}

// ShouldFlush returns true if pending learnings should be flushed.
func ShouldFlush(count int) bool {
    return count > 0 && count%FlushInterval == 0
}
```

**Update Acceptance Criteria:**
- [ ] Threshold functions added to pkg/config/paths.go (NOT new package)
- [ ] Uses existing `syscall.Flock()` atomicity (NOT mutex)
- [ ] `ShouldRemind()` returns true every 10 tools
- [ ] `ShouldFlush()` returns true every 20 tools
- [ ] Tests added to `pkg/config/paths_test.go`
- [ ] Remove ToolCounter struct (use functions directly)

---

### GOgent-069: Reminder & Flush Logic

**Status:** REFACTOR - Use Existing CheckPendingLearnings

**Changes Required:**

1. **Location change:** `pkg/session/` instead of `pkg/observability/`
2. **Reuse existing functions:** `CheckPendingLearnings` may already exist in context_loader.go
3. **Make thresholds configurable** via environment variables

**Check for existing code:**
```bash
grep -r "CheckPendingLearnings\|pending-learnings" pkg/session/
```

**If exists, extend. If not, add to pkg/session/ following existing patterns.**

**Configuration via environment variables:**
```go
const (
    DefaultFlushThreshold = 5
)

func GetFlushThreshold() int {
    if v := os.Getenv("GOGENT_FLUSH_THRESHOLD"); v != "" {
        if i, err := strconv.Atoi(v); err == nil && i > 0 {
            return i
        }
    }
    return DefaultFlushThreshold
}
```

---

### GOgent-071: Integration Tests for attention-gate

**Status:** REFACTOR - Use t.TempDir(), Add Simulation Harness

**Changes Required:**

1. **Replace `os.Remove(COUNTER_FILE)`** with `t.TempDir()` pattern
2. **Add simulation harness integration tests**
3. **Follow existing test patterns** in pkg/config/paths_test.go

**BEFORE (lines 36-38):**
```go
func TestAttentionGateWorkflow_ReminderAt10(t *testing.T) {
    os.Remove(COUNTER_FILE)
    defer os.Remove(COUNTER_FILE)
```

**AFTER:**
```go
func TestAttentionGateWorkflow_ReminderAt10(t *testing.T) {
    tmpDir := t.TempDir()
    // Override counter path for test isolation
    counterPath := filepath.Join(tmpDir, "tool-counter")
    // Use test-specific counter functions that accept path parameter
```

---

### GOgent-072: Build gogent-attention-gate CLI

**Status:** REFACTOR - Merge into gogent-sharp-edge

**Critical Issue:** Having TWO PostToolUse hooks (gogent-sharp-edge and gogent-attention-gate) causes configuration conflicts. Claude Code typically supports one hook per event type.

**Solution:** Merge attention-gate logic INTO existing gogent-sharp-edge CLI.

**Changes Required:**

1. **DO NOT create cmd/gogent-attention-gate/**
2. **Extend cmd/gogent-sharp-edge/main.go** with counter/reminder/flush
3. **Use existing `pkg/routing.ParsePostToolEvent()`** (not new parser)
4. **Fix environment variable priority** (GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR)

**Add to cmd/gogent-sharp-edge/main.go:**
```go
func main() {
    // Existing sharp-edge logic...
    event, _ := routing.ParsePostToolEvent(os.Stdin, timeout)
    failure := detectFailure(event)

    // NEW: Counter increment
    count, counterErr := config.GetToolCountAndIncrement()
    if counterErr != nil {
        fmt.Fprintf(os.Stderr, "[sharp-edge] Warning: counter error: %v\n", counterErr)
        count = 0
    }

    // NEW: Reminder check
    var reminderMsg string
    if config.ShouldRemind(count) {
        reminderMsg = generateRoutingReminder(count)
    }

    // NEW: Flush check
    var flushMsg string
    if config.ShouldFlush(count) {
        shouldFlush, _ := session.ShouldFlushLearnings(projectDir)
        if shouldFlush {
            ctx, _ := session.ArchivePendingLearnings(projectDir)
            flushMsg = generateFlushNotification(ctx)
        }
    }

    // Combine all context
    response := buildCombinedResponse(failure, reminderMsg, flushMsg)
    response.Marshal(os.Stdout)
}
```

**Update Acceptance Criteria:**
- [ ] Merged into gogent-sharp-edge (NO new CLI)
- [ ] Counter uses pkg/config functions (NOT new package)
- [ ] Event parsing uses pkg/routing.ParsePostToolEvent()
- [ ] Environment variable priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > CWD
- [ ] Single PostToolUse handler (no hook conflict)

---

## Section 3: New Tickets to ADD

### GOgent-063a: SubagentStop Event Validation

**Status:** COMPLETED (via einstein research)

**Reason:** The einstein-subagent-stop-research-2026-01-24.md document has validated that SubagentStop exists and documented the actual schema.

**Deliverables (already produced):**
- Confirmation: SubagentStop exists in Claude Code
- Actual schema: session_id, transcript_path, hook_event_name, stop_hook_active
- Limitation documented: Cannot identify specific agent in multi-agent sessions
- Recommendation: Transcript parsing for metadata extraction

**Action:** Mark as COMPLETED in tickets-index.json. Reference research document.

**Ticket Specification:**
```yaml
---
id: GOgent-063a
title: Validate SubagentStop Hook Event Type
status: completed
time_estimate: 1h
dependencies: []
priority: critical
week: 4
tags: ["agent-endstate", "validation", "research"]
tests_required: false
acceptance_criteria_count: 6
---

### GOgent-063a: Validate SubagentStop Hook Event Type

**Time**: 1 hour
**Status**: COMPLETED

**Validation Result**: GO - SubagentStop confirmed to exist.

**Actual Schema** (from Claude Code documentation):
```json
{
  "session_id": "string",
  "transcript_path": "string",
  "hook_event_name": "SubagentStop",
  "stop_hook_active": boolean
}
```

**Fields NOT Available** (must derive from transcript):
- agent_id
- agent_model
- tier
- exit_code
- duration_ms
- output_tokens

**Known Limitation**: In multi-agent sessions, cannot identify which specific agent stopped.

**Solution**: Parse transcript_path file to extract agent metadata.

**Reference**: `.claude/tmp/einstein-subagent-stop-research-2026-01-24.md`

**Acceptance Criteria**:
- [x] SubagentStop event type confirmed in Claude Code documentation
- [x] Actual schema documented
- [x] Fields NOT available identified
- [x] Multi-agent session limitation documented
- [x] Transcript parsing solution identified
- [x] GO decision issued for GOgent-063-067
```

---

### GOgent-073: Extend HandoffArtifacts for New Artifact Types

**Ticket Specification:**
```yaml
---
id: GOgent-073
title: Extend HandoffArtifacts for New Artifact Types
status: pending
time_estimate: 1h
dependencies: ["GOgent-065", "GOgent-069"]
priority: high
week: 4
tags: ["session-archive", "schema", "week-4"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-073: Extend HandoffArtifacts for New Artifact Types

**Time**: 1 hour
**Dependencies**: GOgent-065 (endstate logging), GOgent-069 (flush logic)

**Task**:
Add new artifact types to HandoffArtifacts struct for v1.3 schema.

**File**: `pkg/session/handoff.go`

**Changes**:

```go
// HandoffArtifacts contains references to session artifacts
type HandoffArtifacts struct {
    // Existing fields (v1.0-1.2)
    SharpEdges          []SharpEdge          `json:"sharp_edges"`
    RoutingViolations   []RoutingViolation   `json:"routing_violations"`
    ErrorPatterns       []ErrorPattern       `json:"error_patterns"`
    UserIntents         []UserIntent         `json:"user_intents"`
    Decisions           []Decision           `json:"decisions,omitempty"`
    PreferenceOverrides []PreferenceOverride `json:"preference_overrides,omitempty"`
    PerformanceMetrics  []PerformanceMetric  `json:"performance_metrics,omitempty"`

    // v1.3 additions
    AgentEndstates    []EndstateLog `json:"agent_endstates,omitempty"`
    AutoFlushArchives []string      `json:"auto_flush_archives,omitempty"`
}

// EndstateLog represents a logged agent completion event
type EndstateLog struct {
    Timestamp       int64    `json:"timestamp"`
    SessionID       string   `json:"session_id"`
    TranscriptPath  string   `json:"transcript_path"`
    AgentID         string   `json:"agent_id,omitempty"`  // Derived from transcript
    AgentClass      string   `json:"agent_class,omitempty"`
    Tier            string   `json:"tier,omitempty"`
    ExitCode        int      `json:"exit_code,omitempty"`
    DurationMs      int      `json:"duration_ms,omitempty"`
    OutputTokens    int      `json:"output_tokens,omitempty"`
    Decision        string   `json:"decision"`  // "prompt" or "silent"
    Recommendations []string `json:"recommendations,omitempty"`
}
```

**File**: `pkg/session/handoff_artifacts.go`

Add loading functions:

```go
// loadEndstates reads agent-endstates.jsonl into artifacts
func loadEndstates(artifacts *HandoffArtifacts, projectDir string) error {
    path := filepath.Join(config.GetGOgentDir(), "agent-endstates.jsonl")
    // ... JSONL parsing following existing pattern
}

// loadAutoFlushArchives finds all auto-flush-*.jsonl files
func loadAutoFlushArchives(artifacts *HandoffArtifacts, projectDir string) error {
    archiveDir := filepath.Join(projectDir, ".claude", "memory", "sharp-edges")
    // ... Glob for auto-flush-*.jsonl files
}
```

**Acceptance Criteria**:
- [ ] HandoffArtifacts extended with AgentEndstates field (omitempty)
- [ ] HandoffArtifacts extended with AutoFlushArchives field (omitempty)
- [ ] EndstateLog struct defined following research findings
- [ ] loadEndstates() loads JSONL file
- [ ] loadAutoFlushArchives() finds archive files
- [ ] Backward compatible (all new fields omitempty)
```

---

### GOgent-074: Update Systems Architecture Documentation

**Ticket Specification:**
```yaml
---
id: GOgent-074
title: Update Systems Architecture for Merged PostToolUse Handler
status: pending
time_estimate: 0.5h
dependencies: ["GOgent-072"]
priority: medium
week: 4
tags: ["documentation", "week-4"]
tests_required: false
acceptance_criteria_count: 4
---

### GOgent-074: Update Systems Architecture for Merged PostToolUse Handler

**Time**: 0.5 hours
**Dependencies**: GOgent-072 (merged CLI implementation)

**Task**:
Update documentation to reflect merged PostToolUse handler.

**File**: `docs/systems-architecture-overview.md`

**Changes**:

1. Update hook event flow diagram:
```
PostToolUse ──→ gogent-sharp-edge ──→ Failure tracking
                                  ├──→ Counter increment
                                  ├──→ Reminder injection (every 10)
                                  └──→ Auto-flush (every 20)
```

2. Remove reference to separate gogent-attention-gate CLI

3. Document combined behavior:
- Sharp edge detection (existing)
- Tool counter management (new)
- Routing compliance reminders (new)
- Pending learnings auto-flush (new)

4. Add configuration section for thresholds

**Acceptance Criteria**:
- [ ] Hook event flow diagram updated
- [ ] Single PostToolUse handler documented
- [ ] Counter/reminder/flush behavior explained
- [ ] Configuration options documented
```

---

## Section 4: Dependency Updates

### Updated Dependency Graph

```
PHASE 1: Validation (COMPLETED)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
GOgent-063a (SubagentStop Validation) ✓ COMPLETED


PHASE 2: Attention-Gate (UNBLOCKED)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GOgent-068R ──────────────────────────────────────────────────────┐
(Counter Thresholds in pkg/config)                                │
     │                                                            │
     ▼                                                            │
GOgent-069R ──────────────────────────────────────────────────────┤
(Reminder/Flush in pkg/session)                                   │
     │                                                            │
     ├──────────────────────────────────┐                         │
     ▼                                  ▼                         │
GOgent-070 ELIMINATED              GOgent-072R                    │
(Use existing parser)              (Merge into sharp-edge)        │
                                        │                         │
                                        ▼                         │
                                   GOgent-071R                    │
                                   (Integration Tests)            │
                                        │                         │
                                        ▼                         │
                                   GOgent-074                     │
                                   (Docs Update)                  │
                                                                  │
                                                                  │
PHASE 3: Agent-Endstate (UNBLOCKED - validation complete)         │
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━       │
                                                                  │
GOgent-063R ─────────────────────────────────────────────────────┐│
(SubagentStop structs with ACTUAL schema)                        ││
     │                                                           ││
     ▼                                                           ││
GOgent-064R                                                      ││
(Response generation with transcript parsing)                    ││
     │                                                           ││
     ▼                                                           ││
GOgent-065R                                                      ││
(XDG paths + HandoffArtifacts integration)                       ││
     │                                                           ││
     ▼                                                           ││
GOgent-066R                                                      ││
(Integration tests with simulation harness)                      ││
     │                                                           ││
     ▼                                                           ││
GOgent-067R                                                      ││
(CLI with transcript parsing)                                    ││
                                                                 ││
                                                                 ││
PHASE 4: Handoff Integration                                     ││
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━                                   ▼▼
                                                            GOgent-073
GOgent-069R ─────────────────────────────────────────────→ (HandoffArtifacts
GOgent-065R ─────────────────────────────────────────────→  extension)
```

### Execution Order (Linear)

**Week 4 Day 1-2 (Attention-Gate):**
1. GOgent-068R - Counter thresholds (1h)
2. GOgent-069R - Reminder/flush logic (2h)
3. GOgent-072R - Merge into sharp-edge (2h)
4. GOgent-071R - Integration tests (1.5h)

**Week 4 Day 3-4 (Agent-Endstate):**
5. GOgent-063R - SubagentStop structs (2h)
6. GOgent-064R - Response generation (2h)
7. GOgent-065R - Endstate logging (1.5h)
8. GOgent-066R - Integration tests (1.5h)
9. GOgent-067R - CLI build (1.5h)

**Week 4 Day 5 (Integration + Docs):**
10. GOgent-073 - HandoffArtifacts extension (1h)
11. GOgent-074 - Documentation update (0.5h)

**Total Refactored Estimate:** 15h (vs original 15.5h)

---

## Section 5: Execution Checklist

### Pre-Refactoring Checklist

- [ ] Backup all ticket files: `cp -r tickets/ tickets.backup-$(date +%Y%m%d)/`
- [ ] Backup tickets-index.json: `cp tickets-index.json tickets-index.json.backup-refactor`
- [ ] Verify existing tests pass: `go test ./...`
- [ ] Note current git commit: `git rev-parse HEAD`

### Ticket File Modifications

#### 5.1 Mark GOgent-063a as Completed (NEW TICKET)

Create file: `tickets/GOgent-063a.md`
```yaml
---
id: GOgent-063a
title: Validate SubagentStop Hook Event Type
status: completed
time_estimate: 1h
dependencies: []
# ... (full specification above)
---
```

#### 5.2 Update GOgent-063 (SCHEMA CORRECTION)

- [ ] Replace SubagentStopEvent struct with actual schema
- [ ] Add ParsedAgentMetadata struct
- [ ] Add ParseTranscriptForMetadata function stub
- [ ] Update validation to check session_id, transcript_path
- [ ] Remove min() helper (use Go builtin)
- [ ] Update tests with actual schema JSON
- [ ] Update acceptance criteria

#### 5.3 Update GOgent-064 (RESPONSE ADAPTATION)

- [ ] Update GenerateEndstateResponse signature to accept metadata
- [ ] Add graceful degradation for nil metadata
- [ ] Replace manual JSON formatting with json.Marshal
- [ ] Update tests

#### 5.4 Update GOgent-065 (PATH + HANDOFF)

- [ ] Replace `/tmp/` with `config.GetGOgentDir()`
- [ ] Add dual-write pattern (global + project)
- [ ] Add HandoffArtifacts integration note
- [ ] Update tests to use t.TempDir()

#### 5.5 Update GOgent-066 (TESTS)

- [ ] Update test JSON to actual schema
- [ ] Add mock transcript creation
- [ ] Use t.TempDir() for isolation
- [ ] Add simulation harness tests

#### 5.6 Update GOgent-067 (CLI)

- [ ] Add transcript parsing step
- [ ] Add Makefile target
- [ ] Update acceptance criteria
- [ ] Handle parsing failures gracefully

#### 5.7 Update GOgent-068 (LOCATION CHANGE)

- [ ] Change location from pkg/observability/ to pkg/config/
- [ ] Remove ToolCounter struct (use functions)
- [ ] Add threshold functions only
- [ ] Reference existing counter implementation
- [ ] Update tests location

#### 5.8 Update GOgent-069 (REUSE EXISTING)

- [ ] Change location to pkg/session/
- [ ] Reference existing CheckPendingLearnings if exists
- [ ] Add environment variable configuration
- [ ] Update acceptance criteria

#### 5.9 DELETE GOgent-070

- [ ] Delete file: `rm tickets/GOgent-070.md`
- [ ] Update any references in GOgent-072

#### 5.10 Update GOgent-071 (TESTS)

- [ ] Use t.TempDir() pattern
- [ ] Remove global state manipulation
- [ ] Add simulation harness integration

#### 5.11 Update GOgent-072 (MERGE)

- [ ] Change from new CLI to merge into sharp-edge
- [ ] Reference pkg/routing.ParsePostToolEvent()
- [ ] Fix environment variable priority
- [ ] Remove build script (no new CLI)
- [ ] Update acceptance criteria

#### 5.12 Create GOgent-073 (NEW TICKET)

- [ ] Create file: `tickets/GOgent-073.md`
- [ ] Add full specification (see Section 3)

#### 5.13 Create GOgent-074 (NEW TICKET)

- [ ] Create file: `tickets/GOgent-074.md`
- [ ] Add full specification (see Section 3)

### tickets-index.json Updates

```json
{
  "tickets": [
    // ADD:
    {
      "id": "GOgent-063a",
      "title": "Validate SubagentStop Hook Event Type",
      "status": "completed",
      "dependencies": [],
      "blocks": ["GOgent-063"]
    },

    // MODIFY GOgent-063:
    {
      "id": "GOgent-063",
      "dependencies": ["GOgent-063a", "GOgent-056"],  // ADD GOgent-063a
      // rest unchanged
    },

    // REMOVE:
    // Delete GOgent-070 entry

    // MODIFY GOgent-072:
    {
      "id": "GOgent-072",
      "title": "Merge Attention-Gate into gogent-sharp-edge",  // CHANGED
      "dependencies": ["GOgent-069", "GOgent-071"],  // CHANGED (removed 070)
      // Update other fields
    },

    // ADD:
    {
      "id": "GOgent-073",
      "title": "Extend HandoffArtifacts for New Artifact Types",
      "dependencies": ["GOgent-065", "GOgent-069"],
      "status": "pending"
    },
    {
      "id": "GOgent-074",
      "title": "Update Systems Architecture for Merged PostToolUse Handler",
      "dependencies": ["GOgent-072"],
      "status": "pending"
    }
  ]
}
```

### Post-Refactoring Verification

- [ ] All ticket files parse correctly (valid YAML frontmatter)
- [ ] tickets-index.json is valid JSON
- [ ] No circular dependencies introduced
- [ ] Acceptance criteria counts updated
- [ ] Total time estimates recalculated
- [ ] Run `go test ./...` (should still pass - no code changes yet)

---

## Appendix A: File Change Summary

| File | Action | Key Changes |
|------|--------|-------------|
| `tickets/GOgent-063.md` | MODIFY | Actual SubagentStop schema |
| `tickets/GOgent-063a.md` | CREATE | Validation ticket (completed) |
| `tickets/GOgent-064.md` | MODIFY | Transcript-based metadata |
| `tickets/GOgent-065.md` | MODIFY | XDG paths, HandoffArtifacts |
| `tickets/GOgent-066.md` | MODIFY | Actual schema, t.TempDir() |
| `tickets/GOgent-067.md` | MODIFY | Transcript parsing, Makefile |
| `tickets/GOgent-068.md` | MODIFY | Location to pkg/config |
| `tickets/GOgent-069.md` | MODIFY | Location to pkg/session |
| `tickets/GOgent-070.md` | DELETE | Duplicates pkg/routing |
| `tickets/GOgent-071.md` | MODIFY | t.TempDir(), harness tests |
| `tickets/GOgent-072.md` | MODIFY | Merge into sharp-edge |
| `tickets/GOgent-073.md` | CREATE | HandoffArtifacts extension |
| `tickets/GOgent-074.md` | CREATE | Documentation update |
| `tickets-index.json` | MODIFY | Add/remove/update entries |

---

## Appendix B: Cross-Reference to GAP Analysis

| GAP Recommendation | Ticket Action | Section |
|-------------------|---------------|---------|
| SubagentStop validation | GOgent-063a (completed) | Section 3 |
| SubagentStop schema correction | GOgent-063-067 | Section 2.1 |
| GOgent-070 elimination | DELETE | Section 1 |
| Counter duplication | GOgent-068 location change | Section 2, GOgent-068 |
| PostToolUse hook conflict | GOgent-072 merge | Section 2, GOgent-072 |
| HandoffArtifacts extension | GOgent-073 (new) | Section 3 |
| Documentation update | GOgent-074 (new) | Section 3 |
| XDG path compliance | GOgent-065, GOgent-068 | Section 2 |

---

**End of Refactoring Map**
