# Review Skill Upgrade Plan - ML Telemetry Integration

**Generated**: 2026-02-01
**Author**: Einstein Analysis (Opus)
**Revised**: 2026-02-02 (Einstein v2.2 - Added architect-reviewer to review team)
**Status**: Ready for Implementation
**Estimated Tasks**: 27

---

## Executive Summary

The Multi-Domain Code Review System implementation is architecturally complete but lacks ML telemetry integration. This plan addresses all gaps to enable downstream actionability of review findings.

**Revision Notes (v2.1)**: This plan has been critically reviewed and updated to address:
- Concurrency safety for JSONL writes (new Task 2.0)
- Schema validation before modification (new Task 0.1)
- Sharp edge ID registry for validation (new Task 2.3)
- Dependency graph correction (Task 5.2 moved to Phase 3)
- Feature flag for rollback capability
- Accurate resolution time tracking
- **NEW (v2.1)**: impl-manager agent integration (added in schema v2.5.0)
- **NEW (v2.1)**: Updated sharp_edges_count values from actual files
- **NEW (v2.1)**: impl-violations.jsonl ↔ review-findings.jsonl relationship documented

---

## Phase 0: Validation (Pre-Flight)

### Task 0.1: Validate agents-index.json Schema

**File**: `.claude/agents/agents-index.json`
**Priority**: CRITICAL (blocks Task 1.3)
**Agent**: haiku-scout
**Status**: ✅ COMPLETE (validated during schema v2.5.0 upgrade)

**Purpose**: Confirm the JSON structure before assuming we can add `sharp_edges_count` fields.

> **v2.1 Note**: This task was completed during the schema v2.5.0 upgrade. The schema is
> confirmed to be a flat array of agent objects that can accept additional fields.
> Version is now 2.5.0, structure validated.

**Actions**:
1. Read `agents-index.json`
2. Document actual schema structure
3. Verify each agent entry can accept additional fields
4. Output schema summary to `.claude/tmp/agents-index-schema.json`

**Expected Output**:
```json
{
  "schema_version": "detected or null",
  "entry_structure": "flat_array | nested_object | other",
  "sample_entry_keys": ["id", "name", "triggers", "..."],
  "can_add_fields": true,
  "notes": "Any anomalies found"
}
```

**Failure Action**: If schema is incompatible, Task 1.3 must be redesigned.

---

## Phase 1: Foundation (No Dependencies, Parallel Execution)

### Task 1.1: Add Path Helpers for Telemetry Files

**File**: `pkg/config/paths.go`
**Priority**: CRITICAL (blocks all telemetry tasks)
**Agent**: go-pro

**Implementation**:

```go
// Add after existing path helpers (near GetCollaborationsPathWithProjectDir)

// GetReviewFindingsPathWithProjectDir returns path for review findings log
func GetReviewFindingsPathWithProjectDir() string {
    return getMLTelemetryPath("review-findings.jsonl")
}

// GetReviewOutcomesPathWithProjectDir returns path for review outcome updates
func GetReviewOutcomesPathWithProjectDir() string {
    return getMLTelemetryPath("review-outcomes.jsonl")
}

// GetSharpEdgeHitsPathWithProjectDir returns path for sharp edge correlation log
func GetSharpEdgeHitsPathWithProjectDir() string {
    return getMLTelemetryPath("sharp-edge-hits.jsonl")
}

// Helper function (if not exists)
func getMLTelemetryPath(filename string) string {
    // Check GOGENT_PROJECT_DIR first (test isolation)
    if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
        return filepath.Join(dir, ".claude", "memory", filename)
    }
    // XDG compliance
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        home := os.Getenv("HOME")
        dataHome = filepath.Join(home, ".local", "share")
    }
    return filepath.Join(dataHome, "gogent-fortress", filename)
}
```

**Tests**: Add to `pkg/config/paths_test.go`

---

### Task 1.2: Update gogent-sharp-edge Agent List

**File**: `cmd/gogent-sharp-edge/main.go`
**Priority**: CRITICAL
**Agent**: go-pro

**Location**: Lines 43-60, `getAgentDirectories()` function

**Change**:
```go
// Replace the agents slice with:
agents := []string{
    "python-pro",
    "python-ux",
    "go-pro",
    "go-cli",
    "go-tui",
    "go-api",
    "go-concurrent",
    "r-pro",
    "r-shiny-pro",
    "codebase-search",
    "scaffolder",
    "tech-docs-writer",
    "librarian",
    "code-reviewer",
    "orchestrator",
    "architect",
    // NEW AGENTS (added in schema v2.4.0):
    "typescript-pro",
    "react-pro",
    "backend-reviewer",
    "frontend-reviewer",
    "standards-reviewer",
    "review-orchestrator",
    // NEW AGENT (added in schema v2.5.0):
    "impl-manager",
}
```

**Test**: Add test case in `cmd/gogent-sharp-edge/main_test.go` verifying all agents are included.

---

### Task 1.3: Add sharp_edges_count to agents-index.json

**File**: `.claude/agents/agents-index.json`
**Priority**: HIGH
**Agent**: go-pro
**Blocked By**: Task 0.1

**Changes** (add `sharp_edges_count` field to each entry):

**Agents from v2.4.0** (verify counts from actual sharp-edges.yaml files):
```json
{
  "id": "typescript-pro",
  "sharp_edges_count": 10  // Verify from .claude/agents/typescript-pro/sharp-edges.yaml
}

{
  "id": "react-pro",
  "sharp_edges_count": 10  // Verify from .claude/agents/react-pro/sharp-edges.yaml
}

{
  "id": "backend-reviewer",
  "sharp_edges_count": 10  // Verify from .claude/agents/backend-reviewer/sharp-edges.yaml
}

{
  "id": "frontend-reviewer",
  "sharp_edges_count": 10  // Verify from .claude/agents/frontend-reviewer/sharp-edges.yaml
}

{
  "id": "standards-reviewer",
  "sharp_edges_count": 10  // Verify from .claude/agents/standards-reviewer/sharp-edges.yaml
}
```

**Agents with sharp-edges created in v2.5.0** (actual counts):
```json
{
  "id": "orchestrator",
  "sharp_edges_count": 12  // 7 existing + 5 new (scout-skip, circular-escalation, background-task-orphan, compound-trigger-miss, escalation-loop)
}

{
  "id": "planner",
  "sharp_edges_count": 4  // scope-creep, missing-constraints, dependency-blindness, strategy-without-risks
}

{
  "id": "review-orchestrator",
  "sharp_edges_count": 4  // reviewer-timeout, finding-duplication, severity-inflation, missing-telemetry
}

{
  "id": "gemini-slave",
  "sharp_edges_count": 4  // rate-limit-hit, context-overflow, json-parse-failure, protocol-mismatch
}

{
  "id": "memory-archivist",
  "sharp_edges_count": 3  // incomplete-archive, duplicate-entries, stale-specs
}

{
  "id": "haiku-scout",
  "sharp_edges_count": 2  // scope-underestimate, output-format-mismatch
}

{
  "id": "impl-manager",
  "sharp_edges_count": 7  // specs-drift, convention-bypass, test-gap, scope-creep, orphan-task, missing-acceptance-criteria, parallel-conflict
}
```

**Note**: Counts verified from actual sharp-edges.yaml files created in schema v2.5.0 upgrade.

---

### Task 1.4: Add conventions_required to backend-reviewer

**File**: `.claude/agents/backend-reviewer/agent.yaml`
**Priority**: HIGH
**Pre-requisite**: Verify `.claude/conventions/typescript.md` exists

**Add after `tools:` section**:
```yaml
conventions_required:
  - go.md
  - python.md
  - typescript.md
```

---

### Task 1.5: Add conventions_required to frontend-reviewer

**File**: `.claude/agents/frontend-reviewer/agent.yaml`
**Priority**: HIGH
**Pre-requisite**: Verify `.claude/conventions/react.md` and `.claude/conventions/typescript.md` exist

**Add after `tools:` section**:
```yaml
conventions_required:
  - react.md
  - typescript.md
```

---

### Task 1.6: Add conventions_required to standards-reviewer

**File**: `.claude/agents/standards-reviewer/agent.yaml`
**Priority**: HIGH
**Pre-requisite**: Verify all convention files exist

**Add after `tools:` section**:
```yaml
# Standards reviewer references all conventions for language-specific naming rules
conventions_required:
  - go.md
  - python.md
  - typescript.md
  - react.md
```

---

### Task 1.7: Add Sharp Edge Pattern Matching to Reviewers

**Files**:
- `.claude/agents/backend-reviewer/CLAUDE.md`
- `.claude/agents/frontend-reviewer/CLAUDE.md`
- `.claude/agents/standards-reviewer/CLAUDE.md`

**Add section to each CLAUDE.md**:

```markdown
## Sharp Edge Correlation

When identifying issues, check if they match known sharp edge patterns from sharp-edges.yaml.

For each finding that matches a sharp edge:
1. Include `sharp_edge_id` in output (must be valid ID from Task 2.3 registry)
2. Use the exact symptom description
3. Reference the documented solution

**Output format for correlated findings**:
```json
{
  "severity": "critical",
  "file": "path/to/file.go",
  "line": 45,
  "message": "Issue description",
  "sharp_edge_id": "sql-injection",
  "recommendation": "Use parameterized queries"
}
```

**Available Sharp Edge IDs**:
[List all IDs from this agent's sharp-edges.yaml]
```

---

### Task 1.8: Update CLAUDE.md Routing Tables

**File**: `.claude/CLAUDE.md`
**Priority**: MEDIUM

**Add to "Tier 1.5: Haiku + Thinking" table**:
```markdown
| review backend, api review, security review | `backend-reviewer` | Explore |
| review frontend, component review, ui review | `frontend-reviewer` | Explore |
| review standards, code quality, naming review | `standards-reviewer` | Explore |
```

**Add to "Tier 2: Sonnet" table**:
```markdown
| typescript, ts code, type system, generics | `typescript-pro` | general-purpose |
| react, component, hook, useState, ink | `react-pro` | general-purpose |
| code review, full review, review changes | `review-orchestrator` | Plan |
| implement from specs, execute todos, implement plan | `impl-manager` | Plan |
```

**Add to "Slash Commands" table**:
```markdown
| `/review` | Multi-domain code review with severity-grouped findings |
```

---

### Task 1.9: Document impl-manager Telemetry Relationship (NEW in v2.1)

**File**: `.claude/agents/impl-manager/CLAUDE.md`
**Priority**: MEDIUM
**Added**: Post schema v2.5.0 upgrade

**Purpose**: Document the relationship between impl-manager's real-time violations and review-orchestrator's post-hoc findings.

**Add section to impl-manager CLAUDE.md**:

```markdown
## Telemetry Relationship

impl-manager produces `impl-violations.jsonl` during implementation, which is conceptually related to but distinct from `review-findings.jsonl`:

| Telemetry File | Written By | Phase | Purpose |
|----------------|------------|-------|---------|
| `impl-violations.jsonl` | impl-manager | During implementation | Real-time convention enforcement, blocking if critical |
| `review-findings.jsonl` | review-orchestrator | Post-implementation | Code review feedback, advisory |

**Key Differences:**
- **impl-violations**: Caught DURING implementation, can block task completion
- **review-findings**: Caught AFTER implementation, advisory only

**Unified Schema (Future v2.6.0 consideration)**:
Both could share a common finding schema:
```json
{
  "finding_id": "uuid",
  "source": "impl-manager" | "review-orchestrator",
  "phase": "implementation" | "review",
  "file": "path",
  "line": 42,
  "severity": "critical" | "warning" | "info",
  "message": "description",
  "convention_ref": "go.md#error-handling"
}
```

For v2.5.0, files remain separate. Unification deferred to v2.6.0.
```

---

### Task 1.10: Create architect-reviewer Agent (NEW in v2.2)

**Directory**: `.claude/agents/architect-reviewer/`
**Priority**: HIGH
**Agent**: go-pro
**Added**: Post Einstein analysis for review skill enhancement
**Status**: ✅ COMPLETE

**Purpose**: Add 4th specialized reviewer to the review-orchestrator team. While backend/frontend/standards reviewers check for implementation bugs, architect-reviewer checks for structural patterns, dependency health, and design smells.

**Files Created**:

1. `agent.yaml` - Sonnet tier, 12K thinking budget, subagent_type: Explore
2. `agent.md` - Full role, workflow, severity classification, output format
3. `sharp-edges.yaml` - 12 architectural anti-patterns:
   - `circular-dependency` (critical)
   - `god-module` (critical)
   - `leaky-abstraction` (critical)
   - `tight-coupling` (high)
   - `high-fan-out` (high)
   - `shotgun-surgery` (high)
   - `missing-abstraction` (medium)
   - `premature-abstraction` (medium)
   - `feature-envy` (medium)
   - `inappropriate-intimacy` (medium)
   - `unstable-dependency` (medium)
   - `missing-interface` (low)

**Updates to Existing Files**:

1. `agents-index.json`:
   - Added architect-reviewer entry with `sharp_edges_count: 12`
   - Added to `routing_rules.model_tiers.sonnet` array

2. `review-orchestrator/agent.md`:
   - Updated Phase 1 Detection to include architecture
   - Updated Phase 2 to spawn 4th reviewer (sonnet tier)
   - Updated output format with Architecture Review section
   - Updated BLOCK criteria with architectural issues
   - Updated WARNING criteria with design smells

**Cost Impact**:
| Before (3 reviewers) | After (4 reviewers) |
|----------------------|---------------------|
| ~$0.08-$0.13 | ~$0.15-$0.24 |

Increase of ~$0.07-$0.11 per review due to Sonnet-tier architect-reviewer.

**Rationale**:
- Runs in **parallel** with other reviewers (orthogonal judgment, no context bias)
- Uses **Sonnet** tier (architectural judgment requires more reasoning)
- Focuses on **structure not bugs** (complements implementation reviewers)
- Produces **telemetry-compatible output** (sharp_edge_id correlations)

---

## Phase 2: Core Telemetry

### Task 2.0: Implement JSONL Append with File Locking (NEW)

**File**: `pkg/telemetry/jsonl_writer.go` (NEW)
**Priority**: CRITICAL (blocks Tasks 2.1, 2.2)
**Agent**: go-pro

**Purpose**: Prevent JSONL corruption when parallel reviewers write concurrently.

**Full Implementation**:

```go
package telemetry

import (
    "fmt"
    "os"
    "path/filepath"
    "syscall"
)

// AppendJSONL safely appends a line to a JSONL file with file locking
// to prevent corruption from concurrent writes.
func AppendJSONL(path string, data []byte) error {
    // Ensure parent directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("mkdir: %w", err)
    }

    // Open file for append
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("open: %w", err)
    }
    defer f.Close()

    // Acquire exclusive lock (blocks until available)
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return fmt.Errorf("flock: %w", err)
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    // Write data with newline
    if _, err := f.Write(append(data, '\n')); err != nil {
        return fmt.Errorf("write: %w", err)
    }

    return nil
}
```

**Tests**: Add to `pkg/telemetry/jsonl_writer_test.go`:
- `TestAppendJSONL_Basic`: Single write succeeds
- `TestAppendJSONL_CreatesDir`: Creates parent directories
- `TestAppendJSONL_Concurrent`: 100+ goroutines writing simultaneously, verify no corruption

---

### Task 2.1: Create ReviewFinding Telemetry Struct

**File**: `pkg/telemetry/review_finding.go` (NEW)
**Priority**: CRITICAL
**Agent**: go-pro
**Blocked By**: Tasks 1.1, 2.0

**Full Implementation**:

```go
package telemetry

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
    "github.com/google/uuid"
)

// ReviewFinding captures a single code review finding for ML analysis
type ReviewFinding struct {
    // Identity
    FindingID   string `json:"finding_id"`
    Timestamp   int64  `json:"timestamp"`
    SessionID   string `json:"session_id"`

    // Review context
    ReviewScope   string `json:"review_scope"`    // "staged", "all", "glob", "explicit"
    FilesReviewed int    `json:"files_reviewed"`

    // Finding details
    Severity       string `json:"severity"`        // "critical", "warning", "info"
    Reviewer       string `json:"reviewer"`        // "backend-reviewer", "frontend-reviewer", "standards-reviewer"
    Category       string `json:"category"`        // "security", "performance", "accessibility", etc.
    File           string `json:"file"`
    Line           int    `json:"line,omitempty"`
    Message        string `json:"message"`
    Recommendation string `json:"recommendation,omitempty"`

    // Sharp edge correlation
    SharpEdgeID string `json:"sharp_edge_id,omitempty"`

    // Outcome (populated later)
    WasFixed  bool   `json:"was_fixed,omitempty"`
    FixCommit string `json:"fix_commit,omitempty"`
}

// ReviewOutcomeUpdate represents an outcome update (append-only)
type ReviewOutcomeUpdate struct {
    FindingID       string `json:"finding_id"`
    Resolution      string `json:"resolution"`       // "fixed", "wontfix", "false_positive", "deferred"
    ResolutionMs    int64  `json:"resolution_ms"`    // Time from finding to resolution
    TicketID        string `json:"ticket_id,omitempty"`
    CommitHash      string `json:"commit_hash,omitempty"`
    UpdateTimestamp int64  `json:"update_timestamp"`
}

// NewReviewFinding creates a new finding record
func NewReviewFinding(sessionID, reviewer, severity, category, file string, line int, message string) *ReviewFinding {
    return &ReviewFinding{
        FindingID:   uuid.New().String(),
        Timestamp:   time.Now().Unix(),
        SessionID:   sessionID,
        Reviewer:    reviewer,
        Severity:    severity,
        Category:    category,
        File:        file,
        Line:        line,
        Message:     truncateMessage(message, 1000), // Increased from 500 for stack traces
    }
}

// LogReviewFinding writes finding to JSONL storage (concurrency-safe)
func LogReviewFinding(finding *ReviewFinding) error {
    path := config.GetReviewFindingsPathWithProjectDir()

    data, err := json.Marshal(finding)
    if err != nil {
        return fmt.Errorf("[review-finding] marshal: %w", err)
    }

    return AppendJSONL(path, data)
}

// UpdateReviewFindingOutcome appends outcome update (concurrency-safe)
func UpdateReviewFindingOutcome(findingID, resolution, ticketID, commitHash string, resolutionMs int64) error {
    update := ReviewOutcomeUpdate{
        FindingID:       findingID,
        Resolution:      resolution,
        ResolutionMs:    resolutionMs,
        TicketID:        ticketID,
        CommitHash:      commitHash,
        UpdateTimestamp: time.Now().Unix(),
    }

    path := config.GetReviewOutcomesPathWithProjectDir()

    data, err := json.Marshal(update)
    if err != nil {
        return fmt.Errorf("[review-outcome] marshal: %w", err)
    }

    return AppendJSONL(path, data)
}

// LookupFindingTimestamp retrieves the original timestamp for a finding
// Used to calculate accurate resolution time
func LookupFindingTimestamp(findingID string) (int64, error) {
    findings, err := ReadReviewFindings()
    if err != nil {
        return 0, err
    }
    for _, f := range findings {
        if f.FindingID == findingID {
            return f.Timestamp, nil
        }
    }
    return 0, fmt.Errorf("finding not found: %s", findingID)
}

func truncateMessage(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

---

### Task 2.2: Create SharpEdgeHit Telemetry Struct

**File**: `pkg/telemetry/sharp_edge_hit.go` (NEW)
**Priority**: CRITICAL
**Agent**: go-pro
**Blocked By**: Tasks 1.1, 2.0, 2.3

**Full Implementation**:

```go
package telemetry

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
    "github.com/google/uuid"
)

// SharpEdgeHit tracks when a reviewer catches a known sharp edge pattern
type SharpEdgeHit struct {
    HitID           string  `json:"hit_id"`
    Timestamp       int64   `json:"timestamp"`
    SessionID       string  `json:"session_id"`
    SharpEdgeID     string  `json:"sharp_edge_id"`     // From sharp-edges.yaml (validated)
    AgentID         string  `json:"agent_id"`          // Which agent owns the sharp edge
    ReviewerID      string  `json:"reviewer_id"`       // Which reviewer caught it
    FindingID       string  `json:"finding_id"`        // Links to ReviewFinding
    File            string  `json:"file"`
    Line            int     `json:"line,omitempty"`
    MatchConfidence float64 `json:"match_confidence"`  // 0.0-1.0
    WasActioned     bool    `json:"was_actioned"`      // Did user fix it
}

// NewSharpEdgeHit creates a new hit record
// Returns error if sharpEdgeID is not in the registry
func NewSharpEdgeHit(sessionID, sharpEdgeID, agentID, reviewerID, findingID, file string, line int) (*SharpEdgeHit, error) {
    // Validate sharp edge ID against registry
    if !IsValidSharpEdgeID(sharpEdgeID) {
        return nil, fmt.Errorf("invalid sharp_edge_id: %s", sharpEdgeID)
    }

    return &SharpEdgeHit{
        HitID:           uuid.New().String(),
        Timestamp:       time.Now().Unix(),
        SessionID:       sessionID,
        SharpEdgeID:     sharpEdgeID,
        AgentID:         agentID,
        ReviewerID:      reviewerID,
        FindingID:       findingID,
        File:            file,
        Line:            line,
        MatchConfidence: 1.0, // Default to exact match; can be overridden
    }, nil
}

// LogSharpEdgeHit writes hit to JSONL storage (concurrency-safe)
func LogSharpEdgeHit(hit *SharpEdgeHit) error {
    path := config.GetSharpEdgeHitsPathWithProjectDir()

    data, err := json.Marshal(hit)
    if err != nil {
        return fmt.Errorf("[sharp-edge-hit] marshal: %w", err)
    }

    return AppendJSONL(path, data)
}
```

---

### Task 2.3: Create Sharp Edge ID Registry (NEW)

**File**: `pkg/telemetry/sharp_edge_registry.go` (NEW)
**Priority**: HIGH
**Agent**: go-pro
**Blocked By**: Task 1.7

**Purpose**: Validate sharp_edge_id values against known patterns to prevent silent correlation failures.

**Implementation**:

```go
package telemetry

import (
    "os"
    "path/filepath"
    "sync"

    "gopkg.in/yaml.v3"
)

var (
    sharpEdgeIDs     map[string]bool
    sharpEdgeIDsOnce sync.Once
)

// SharpEdgesYAML represents the structure of sharp-edges.yaml
type SharpEdgesYAML struct {
    SharpEdges []struct {
        ID string `yaml:"id"`
    } `yaml:"sharp_edges"`
}

// LoadSharpEdgeIDs scans all agent sharp-edges.yaml files and builds registry
func LoadSharpEdgeIDs() error {
    var loadErr error
    sharpEdgeIDsOnce.Do(func() {
        sharpEdgeIDs = make(map[string]bool)

        // Find .claude/agents directory
        agentsDir := filepath.Join(os.Getenv("HOME"), ".claude", "agents")
        if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
            agentsDir = filepath.Join(projectDir, ".claude", "agents")
        }

        // Walk agent directories
        entries, err := os.ReadDir(agentsDir)
        if err != nil {
            loadErr = err
            return
        }

        for _, entry := range entries {
            if !entry.IsDir() {
                continue
            }
            yamlPath := filepath.Join(agentsDir, entry.Name(), "sharp-edges.yaml")
            data, err := os.ReadFile(yamlPath)
            if err != nil {
                continue // Agent may not have sharp edges
            }

            var se SharpEdgesYAML
            if err := yaml.Unmarshal(data, &se); err != nil {
                continue
            }

            for _, edge := range se.SharpEdges {
                if edge.ID != "" {
                    sharpEdgeIDs[edge.ID] = true
                }
            }
        }
    })
    return loadErr
}

// IsValidSharpEdgeID checks if an ID exists in the registry
func IsValidSharpEdgeID(id string) bool {
    if sharpEdgeIDs == nil {
        LoadSharpEdgeIDs()
    }
    return sharpEdgeIDs[id]
}

// GetAllSharpEdgeIDs returns all registered IDs (for documentation)
func GetAllSharpEdgeIDs() []string {
    if sharpEdgeIDs == nil {
        LoadSharpEdgeIDs()
    }
    ids := make([]string, 0, len(sharpEdgeIDs))
    for id := range sharpEdgeIDs {
        ids = append(ids, id)
    }
    return ids
}
```

**Tests**: Add to `pkg/telemetry/sharp_edge_registry_test.go`

---

## Phase 3: CLI Tools and Orchestrator Integration

### Task 3.1: Create gogent-log-review Binary

**File**: `cmd/gogent-log-review/main.go` (NEW)
**Priority**: HIGH
**Agent**: go-pro
**Blocked By**: Tasks 2.1, 2.2, 2.3

**Full Implementation**:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

const DEFAULT_TIMEOUT = 5 * time.Second

// ReviewInput represents the JSON input from /review skill
type ReviewInput struct {
    SessionID     string          `json:"session_id"`
    ReviewScope   string          `json:"review_scope"`
    FilesReviewed int             `json:"files_reviewed"`
    Findings      []FindingInput  `json:"findings"`
}

type FindingInput struct {
    Severity       string `json:"severity"`
    Reviewer       string `json:"reviewer"`
    Category       string `json:"category"`
    File           string `json:"file"`
    Line           int    `json:"line"`
    Message        string `json:"message"`
    Recommendation string `json:"recommendation"`
    SharpEdgeID    string `json:"sharp_edge_id"`
}

type LogOutput struct {
    Logged           int      `json:"logged"`
    FindingIDs       []string `json:"finding_ids"`
    SharpEdgeHits    int      `json:"sharp_edge_hits"`
    InvalidEdgeIDs   []string `json:"invalid_edge_ids,omitempty"`
}

func main() {
    // Read JSON from stdin
    var input ReviewInput
    decoder := json.NewDecoder(os.Stdin)
    if err := decoder.Decode(&input); err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
        os.Exit(1)
    }

    // Load sharp edge registry for validation
    if err := telemetry.LoadSharpEdgeIDs(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not load sharp edge registry: %v\n", err)
    }

    output := LogOutput{
        FindingIDs:     make([]string, 0, len(input.Findings)),
        InvalidEdgeIDs: make([]string, 0),
    }

    for _, f := range input.Findings {
        // Create and log finding
        finding := telemetry.NewReviewFinding(
            input.SessionID,
            f.Reviewer,
            f.Severity,
            f.Category,
            f.File,
            f.Line,
            f.Message,
        )
        finding.ReviewScope = input.ReviewScope
        finding.FilesReviewed = input.FilesReviewed
        finding.Recommendation = f.Recommendation
        finding.SharpEdgeID = f.SharpEdgeID

        if err := telemetry.LogReviewFinding(finding); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: failed to log finding: %v\n", err)
            continue
        }

        output.FindingIDs = append(output.FindingIDs, finding.FindingID)
        output.Logged++

        // Log sharp edge hit if correlated (with validation)
        if f.SharpEdgeID != "" {
            hit, err := telemetry.NewSharpEdgeHit(
                input.SessionID,
                f.SharpEdgeID,
                f.Reviewer,
                f.Reviewer,
                finding.FindingID,
                f.File,
                f.Line,
            )
            if err != nil {
                output.InvalidEdgeIDs = append(output.InvalidEdgeIDs, f.SharpEdgeID)
                fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
                continue
            }
            if err := telemetry.LogSharpEdgeHit(hit); err != nil {
                fmt.Fprintf(os.Stderr, "Warning: failed to log sharp edge hit: %v\n", err)
            } else {
                output.SharpEdgeHits++
            }
        }
    }

    // Output summary
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.Encode(output)
}
```

**Makefile**: Add to build targets:
```makefile
.PHONY: telemetry-tools
telemetry-tools: gogent-log-review gogent-update-review-outcome

gogent-log-review:
	go build -o bin/gogent-log-review ./cmd/gogent-log-review

gogent-update-review-outcome:
	go build -o bin/gogent-update-review-outcome ./cmd/gogent-update-review-outcome

.PHONY: all
all: hooks telemetry-tools  # Ensure telemetry-tools is in the all target
```

---

### Task 3.2: Create gogent-update-review-outcome Binary

**File**: `cmd/gogent-update-review-outcome/main.go` (NEW)
**Priority**: HIGH
**Agent**: go-pro
**Blocked By**: Task 2.1

**Full Implementation** (with finding lookup for accurate resolution time):

```go
package main

import (
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

func main() {
    findingID := flag.String("finding-id", "", "Finding ID to update (required)")
    resolution := flag.String("resolution", "", "Resolution: fixed, wontfix, false_positive, deferred (required)")
    ticketID := flag.String("ticket-id", "", "Associated ticket ID (optional)")
    commit := flag.String("commit", "", "Commit hash that fixed it (optional)")
    flag.Parse()

    if *findingID == "" || *resolution == "" {
        fmt.Fprintln(os.Stderr, "Usage: gogent-update-review-outcome --finding-id=ID --resolution=TYPE [--ticket-id=ID] [--commit=HASH]")
        os.Exit(1)
    }

    // Validate resolution
    validResolutions := map[string]bool{
        "fixed": true, "wontfix": true, "false_positive": true, "deferred": true,
    }
    if !validResolutions[*resolution] {
        fmt.Fprintf(os.Stderr, "Invalid resolution: %s\n", *resolution)
        os.Exit(1)
    }

    // Calculate accurate resolution time by looking up original finding
    var resolutionMs int64 = 0
    if origTimestamp, err := telemetry.LookupFindingTimestamp(*findingID); err == nil {
        resolutionMs = (time.Now().Unix() - origTimestamp) * 1000
    } else {
        fmt.Fprintf(os.Stderr, "Warning: could not lookup finding timestamp: %v\n", err)
    }

    err := telemetry.UpdateReviewFindingOutcome(*findingID, *resolution, *ticketID, *commit, resolutionMs)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Updated finding %s: resolution=%s, resolution_time=%dms\n", *findingID, *resolution, resolutionMs)
}
```

---

### Task 3.3: Integrate Telemetry into review-orchestrator (MOVED from Phase 5)

**File**: `.claude/agents/review-orchestrator/CLAUDE.md`
**Priority**: HIGH (must complete before Phase 4)
**Blocked By**: Task 1.7

**Add to "Synthesis Phase" section**:

```markdown
### Telemetry Requirements

After collecting findings, ensure output includes telemetry-compatible format:

```json
{
  "session_id": "[from context or generate UUID]",
  "status": "BLOCKED",
  "summary": { "critical": 2, "warnings": 3, "info": 1 },
  "findings": [
    {
      "severity": "critical",
      "reviewer": "backend-reviewer",
      "category": "security",
      "file": "src/api/handler.go",
      "line": 45,
      "message": "SQL injection via string concatenation",
      "recommendation": "Use parameterized queries",
      "sharp_edge_id": "sql-injection"
    }
  ]
}
```

**Required fields per finding**:
- `severity`: critical, warning, info
- `reviewer`: Which specialist found it
- `category`: security, performance, accessibility, maintainability, etc.
- `file`: Full file path
- `line`: Line number (0 if not applicable)
- `message`: Issue description
- `recommendation`: Fix suggestion
- `sharp_edge_id`: If matches known pattern (optional, must be valid ID)

**IMPORTANT**: The `session_id` field is REQUIRED for telemetry correlation.
If not available from context, generate a UUID.
```

---

## Phase 4: Skill Integration

### Task 4.1: Update /review Skill for Telemetry

**File**: `.claude/skills/review/SKILL.md`
**Priority**: HIGH
**Blocked By**: Tasks 3.1, 3.3

**Add to Phase 4 (Report Generation)**:

```markdown
### Phase 4: Report Generation and Telemetry

After orchestrator completes:

```bash
# Read review result
review_result=$(cat .claude/tmp/review-result.json)

# Check if telemetry is enabled (default: enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]]; then
    # Extract telemetry data and log (non-blocking)
    session_id=$(echo "$review_result" | jq -r '.session_id // "unknown"')
    review_scope="${review_scope:-staged}"
    files_count=$(echo "$files" | wc -l)

    # Build telemetry input
    telemetry_input=$(jq -n \
        --arg sid "$session_id" \
        --arg scope "$review_scope" \
        --argjson files "$files_count" \
        --argjson findings "$(echo "$review_result" | jq '.findings')" \
        '{session_id: $sid, review_scope: $scope, files_reviewed: $files, findings: $findings}')

    # Log to ML telemetry (non-blocking, errors ignored)
    echo "$telemetry_input" | gogent-log-review > .claude/tmp/review-telemetry.json 2>/dev/null || true
fi

# Continue with display...
```

**Add new section**:

```markdown
## ML Telemetry

This skill logs all findings to ML telemetry for downstream analysis:

| File | Purpose |
|------|---------|
| `$XDG_DATA_HOME/gogent-fortress/review-findings.jsonl` | All review findings |
| `$XDG_DATA_HOME/gogent-fortress/sharp-edge-hits.jsonl` | Sharp edge correlations |
| `.claude/tmp/review-telemetry.json` | Session telemetry output |

Telemetry is non-blocking - skill continues even if logging fails.

### Disabling Telemetry

Set `GOGENT_ENABLE_TELEMETRY=0` to disable telemetry logging (useful for debugging or privacy).
```

---

### Task 4.2: Update /ticket Phase 7.6 for Outcome Tracking

**File**: `.claude/skills/ticket/SKILL.md`
**Priority**: HIGH
**Blocked By**: Task 3.2

**Update Phase 7.6**:

```markdown
### Phase 7.6: Code Review (Blocking)

After audit, run code review if enabled:

```bash
# ... existing review invocation code ...

# Store finding IDs for outcome tracking (if telemetry enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f .claude/tmp/review-telemetry.json ]]; then
    jq -r '.finding_ids[]' .claude/tmp/review-telemetry.json > "$tickets_dir/.review-findings-$ticket_id" 2>/dev/null || true
fi
```

**Update Phase 8 (Completion)**:

```markdown
### Phase 8: Completion Workflow

```bash
# ... existing completion code ...

# Log review outcomes if code review was run (if telemetry enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f "$tickets_dir/.review-findings-$ticket_id" ]]; then
    commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "")
    while IFS= read -r finding_id; do
        gogent-update-review-outcome \
            --finding-id="$finding_id" \
            --resolution="fixed" \
            --ticket-id="$ticket_id" \
            --commit="$commit_hash" 2>/dev/null || true
    done < "$tickets_dir/.review-findings-$ticket_id"
    rm -f "$tickets_dir/.review-findings-$ticket_id"
fi
```

---

## Phase 5: Extended Features

### Task 5.1: Add gogent-ml-export Review Support

**File**: `cmd/gogent-ml-export/main.go`
**Priority**: MEDIUM
**Agent**: go-pro
**Blocked By**: Tasks 2.1, 2.2

**Add subcommands**:

```go
case "review-findings":
    // Export all review findings
    findings, err := telemetry.ReadReviewFindings()
    if err != nil {
        return err
    }
    return outputJSON(findings, outputPath)

case "review-stats":
    // Show review statistics
    findings, _ := telemetry.ReadReviewFindings()
    stats := telemetry.CalculateReviewStats(findings)
    return outputJSON(stats, "")

case "sharp-edge-hits":
    // Export sharp edge correlations
    hits, err := telemetry.ReadSharpEdgeHits()
    if err != nil {
        return err
    }
    return outputJSON(hits, outputPath)
```

**Add read functions to `pkg/telemetry/review_finding.go`**:

```go
// ReadReviewFindings reads all findings from storage
func ReadReviewFindings() ([]ReviewFinding, error) {
    path := config.GetReviewFindingsPathWithProjectDir()
    // Read JSONL file line by line
    // Unmarshal each line into ReviewFinding
    // Return slice
}

// CalculateReviewStats returns aggregate metrics
func CalculateReviewStats(findings []ReviewFinding) map[string]interface{} {
    // Group by severity, reviewer, category
    // Calculate fix rates, average resolution time, etc.
    return map[string]interface{}{
        "total_findings": len(findings),
        "by_severity": map[string]int{...},
        "by_reviewer": map[string]int{...},
        "by_category": map[string]int{...},
        "avg_resolution_ms": ...,
    }
}
```

---

## Phase 6: Testing

### Task 6.1: Add Telemetry Tests

**Files**:
- `pkg/telemetry/jsonl_writer_test.go` (NEW)
- `pkg/telemetry/review_finding_test.go` (NEW)
- `pkg/telemetry/sharp_edge_hit_test.go` (NEW)
- `pkg/telemetry/sharp_edge_registry_test.go` (NEW)

**Priority**: HIGH
**Agent**: go-pro

**Test isolation helper** (add to each test file):

```go
func setupTestDir(t *testing.T) func() {
    t.Helper()
    dir := t.TempDir()
    t.Setenv("GOGENT_PROJECT_DIR", dir)

    // Create minimal .claude structure
    os.MkdirAll(filepath.Join(dir, ".claude", "memory"), 0755)
    os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755)

    return func() { /* TempDir auto-cleans */ }
}
```

**Test cases for jsonl_writer_test.go**:

```go
func TestAppendJSONL_Basic(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()
    // Write single line, verify content
}

func TestAppendJSONL_CreatesDir(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()
    // Write to nested path, verify dirs created
}

func TestAppendJSONL_Concurrent(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()

    path := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), "test.jsonl")

    // Spawn 100 goroutines writing simultaneously
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            data := fmt.Sprintf(`{"id":%d}`, n)
            err := AppendJSONL(path, []byte(data))
            assert.NoError(t, err)
        }(i)
    }
    wg.Wait()

    // Verify: 100 valid JSON lines, no corruption
    content, _ := os.ReadFile(path)
    lines := strings.Split(strings.TrimSpace(string(content)), "\n")
    assert.Equal(t, 100, len(lines))

    for _, line := range lines {
        var obj map[string]int
        err := json.Unmarshal([]byte(line), &obj)
        assert.NoError(t, err, "Line should be valid JSON: %s", line)
    }
}
```

**Test cases for review_finding_test.go**:

```go
func TestNewReviewFinding(t *testing.T) {
    finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test")
    assert.NotEmpty(t, finding.FindingID)
    assert.Equal(t, "critical", finding.Severity)
}

func TestLogReviewFinding(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()

    finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test message")
    err := LogReviewFinding(finding)
    assert.NoError(t, err)

    // Verify file written
    path := config.GetReviewFindingsPathWithProjectDir()
    content, err := os.ReadFile(path)
    assert.NoError(t, err)
    assert.Contains(t, string(content), finding.FindingID)
}

func TestLookupFindingTimestamp(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()

    finding := NewReviewFinding("session1", "reviewer", "warning", "perf", "file.go", 1, "msg")
    LogReviewFinding(finding)

    ts, err := LookupFindingTimestamp(finding.FindingID)
    assert.NoError(t, err)
    assert.Equal(t, finding.Timestamp, ts)
}

func TestUpdateReviewFindingOutcome(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()

    err := UpdateReviewFindingOutcome("finding-123", "fixed", "TICKET-1", "abc123", 5000)
    assert.NoError(t, err)

    // Verify appended to outcomes file
    path := config.GetReviewOutcomesPathWithProjectDir()
    content, _ := os.ReadFile(path)
    assert.Contains(t, string(content), "finding-123")
    assert.Contains(t, string(content), "fixed")
}
```

**Test cases for sharp_edge_registry_test.go**:

```go
func TestIsValidSharpEdgeID(t *testing.T) {
    cleanup := setupTestDir(t)
    defer cleanup()

    // Create test sharp-edges.yaml
    agentDir := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".claude", "agents", "test-agent")
    os.MkdirAll(agentDir, 0755)
    yaml := `sharp_edges:
  - id: sql-injection
  - id: xss-vulnerability
`
    os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte(yaml), 0644)

    LoadSharpEdgeIDs() // Force reload

    assert.True(t, IsValidSharpEdgeID("sql-injection"))
    assert.True(t, IsValidSharpEdgeID("xss-vulnerability"))
    assert.False(t, IsValidSharpEdgeID("nonexistent-id"))
}
```

---

## Phase 7: Nice-to-Have Features

### Task 7.1: Reviewer Accuracy Feedback

**Files**:
- `pkg/telemetry/review_finding.go` (extend)
- `cmd/gogent-review-feedback/main.go` (NEW)

**Extend ReviewOutcomeUpdate**:
```go
UserFeedback    string `json:"user_feedback,omitempty"`    // "helpful", "not_helpful", "partially_helpful"
FeedbackComment string `json:"feedback_comment,omitempty"`
```

**CLI**:
```bash
gogent-review-feedback --finding-id=abc123 --feedback=helpful
```

---

### Task 7.2: Cross-Session Sharp Edge Candidate Detection

**Files**:
- `pkg/telemetry/sharp_edge_candidate.go` (NEW)
- `cmd/gogent-aggregate/main.go` (extend)

**Algorithm**:
1. Read all review-findings.jsonl
2. Group by (category, message similarity)
3. If count >= 3 across different sessions, create candidate
4. Output to sharp-edge-candidates.jsonl

---

## Execution Order Summary

```
Phase 0 (Pre-Flight - Validation):
└── Task 0.1: Validate agents-index.json schema [CRITICAL] ✅ DONE (v2.5.0)

Phase 1 (Parallel - After 0.1 for 1.3):
├── Task 1.1: Path helpers [CRITICAL]
├── Task 1.2: Update agent list [CRITICAL] (includes impl-manager from v2.5.0)
├── Task 1.3: sharp_edges_count (blocked by 0.1) [HIGH] (actual counts from v2.5.0)
├── Task 1.4-1.6: conventions_required [HIGH]
├── Task 1.7: Sharp edge matching [HIGH]
├── Task 1.8: CLAUDE.md updates [MEDIUM] (includes impl-manager routing)
└── Task 1.9: impl-manager telemetry docs [MEDIUM] ← NEW in v2.1

Phase 2 (After 1.1, 1.7):
├── Task 2.0: JSONL file locking [CRITICAL] ← NEW
├── Task 2.1: ReviewFinding struct (blocked by 2.0) [CRITICAL]
├── Task 2.2: SharpEdgeHit struct (blocked by 2.0, 2.3) [CRITICAL]
└── Task 2.3: Sharp edge ID registry (blocked by 1.7) [HIGH] ← NEW

Phase 3 (After Phase 2):
├── Task 3.1: gogent-log-review [HIGH]
├── Task 3.2: gogent-update-review-outcome [HIGH]
└── Task 3.3: orchestrator integration [HIGH] ← MOVED from Phase 5

Phase 4 (After Phase 3, including 3.3):
├── Task 4.1: /review skill telemetry [HIGH]
└── Task 4.2: /ticket outcome tracking [HIGH]

Phase 5 (After Phase 2):
└── Task 5.1: ml-export support [MEDIUM]

Phase 6 (After Phase 2):
└── Task 6.1: Tests (with isolation helper) [HIGH]

Phase 7 (Optional):
├── Task 7.1: Accuracy feedback [MEDIUM]
└── Task 7.2: Candidate detection [MEDIUM]
```

---

## Verification Checklist

After implementation, verify:

### Core Functionality
- [ ] `gogent-log-review` binary builds and runs
- [ ] `gogent-update-review-outcome` binary builds and runs
- [ ] `/review` creates entries in review-findings.jsonl
- [ ] `/ticket complete` updates outcomes in review-outcomes.jsonl
- [ ] `gogent-ml-export review-findings` outputs data
- [ ] `gogent-ml-export review-stats` shows metrics
- [ ] Sharp edge correlations logged to sharp-edge-hits.jsonl
- [ ] New agents appear in gogent-sharp-edge index
- [ ] `impl-manager` appears in gogent-sharp-edge agent list (v2.5.0)
- [ ] `impl-manager` has `sharp_edges_count: 7` in agents-index.json (v2.5.0)
- [ ] impl-violations.jsonl schema documented in ARCHITECTURE.md (v2.5.0)

### Data Integrity
- [ ] JSONL files parseable after concurrent writes (100+ goroutine test)
- [ ] All sharp_edge_id references validate against registry
- [ ] Resolution time accurately calculated (not 0)

### Rollback & Integration
- [ ] Telemetry doesn't break existing /review workflow
- [ ] Telemetry doesn't break existing /ticket workflow
- [ ] `GOGENT_ENABLE_TELEMETRY=0` successfully disables telemetry
- [ ] Convention files (typescript.md, react.md) exist before agent updates

### Performance & CI
- [ ] Telemetry adds <100ms to /review execution
- [ ] All tests pass (including concurrent writes)
- [ ] Makefile builds all binaries in clean environment
- [ ] CI/CD pipeline updated for new binaries

---

## Rollback Procedure

If telemetry integration causes issues:

1. **Disable telemetry**: `export GOGENT_ENABLE_TELEMETRY=0`
2. **Revert skill changes**: `git checkout -- .claude/skills/review/SKILL.md .claude/skills/ticket/SKILL.md`
3. **Keep binaries**: They're harmless if not invoked
4. **Investigate**: Check `.claude/tmp/review-telemetry.json` for errors

---

**End of Plan**
