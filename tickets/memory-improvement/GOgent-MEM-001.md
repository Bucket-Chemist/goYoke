---
id: GOgent-MEM-001
title: "Structured Problem Capture for Memory Improvement"
time: "12-14 hours"
priority: HIGH
dependencies: "github.com/gofrs/flock (file locking), gopkg.in/yaml.v3 (schema parsing)"
status: pending
reviewed_by: "staff-architect-critical-review (7-layer review completed, blockers addressed in v3)"
revision: 3
---

# GOgent-MEM-001: Structured Problem Capture for Memory Improvement

**Time:** 8-10 hours (revised from 6-8 after review)
**Dependencies:** None (extends existing infrastructure)
**Priority:** HIGH (enables knowledge compounding)
**Review Status:** Staff-architect review COMPLETE - 3 blockers resolved

---

## Overview

Extend `gogent-sharp-edge` hook to capture structured problem-solution data when problems are resolved. This data feeds into `/memory-improvement` for automatic sharp-edge recommendations.

**Inspired by:** Compound Engineering Plugin's `docs/solutions/` capture mechanism
**Difference:** Machine-readable JSONL (for Gemini audit) vs human-readable Markdown

---

## Staff-Architect Review Summary

| Severity | Count | Status |
|----------|-------|--------|
| BLOCKER | 3 | ✅ Addressed in v2 |
| HIGH | 5 | ✅ Addressed in v2 |
| MEDIUM | 6 | ⚠️ 4 addressed, 2 deferred to v1.1 |
| LOW | 4 | Deferred to v1.1 |

### Resolved Issues

| Issue | Resolution |
|-------|------------|
| Session isolation violation | Session ID in filename |
| No atomicity guarantees | File locking with flock |
| Unclear Gemini integration | Phase 4 spec added |
| Success detection ambiguity | Consecutive success threshold |
| Missing SessionID validation | Explicit validation added |
| Schema compatibility | SKILL.md update included |
| Unconstrained memory growth | Rotation strategy added |
| Missing integration tests | Test scenarios specified |

---

## Architecture

### Current Flow (Failures Only)

```
Tool fails → gogent-sharp-edge detects → FailureInfo logged →
  3+ failures → SharpEdge written to pending-learnings.jsonl
```

### New Flow (Failures + Resolutions)

```
Tool fails → gogent-sharp-edge detects → FailureInfo logged →
  3+ failures → SharpEdge captured
                    ↓
Tool succeeds on same file (2+ times) → Resolution detected →
  SolvedProblem written to solved-problems.jsonl
                    ↓
/memory-improvement reads solved-problems.jsonl →
  Gemini correlates with sharp-edges.yaml →
  Recommends new sharp edges
```

---

## Data Schema

### File: `pkg/telemetry/solved_problem.go`

```go
package telemetry

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/gofrs/flock"
)

// SolvedProblem captures a resolved problem with semantic metadata
type SolvedProblem struct {
    // Identity
    SessionID string `json:"session_id"`
    Timestamp int64  `json:"timestamp"`

    // Problem Classification (enum-validated)
    ProblemType string `json:"problem_type"` // See ProblemTypeEnum
    Component   string `json:"component"`    // See ComponentEnum
    Severity    string `json:"severity"`     // critical, high, medium, low

    // Problem Details
    File         string   `json:"file"`
    ErrorType    string   `json:"error_type"`
    Symptoms     []string `json:"symptoms"`      // Observable behaviors
    ErrorMessage string   `json:"error_message"` // Actual error text

    // Resolution Details
    RootCause      string `json:"root_cause"`       // See RootCauseEnum
    ResolutionType string `json:"resolution_type"`  // See ResolutionTypeEnum
    SuccessTool    string `json:"success_tool"`     // Tool that succeeded
    SuccessInput   string `json:"success_input"`    // Truncated input that worked

    // Context
    FailedTools    []string `json:"failed_tools"`    // Tools that failed before success
    FailureCount   int      `json:"failure_count"`   // How many attempts
    TimeToResolve  int64    `json:"time_to_resolve"` // Milliseconds from first failure
    SuccessCount   int      `json:"success_count"`   // Consecutive successes before declared resolved

    // Metadata
    InferredClassification bool   `json:"inferred_classification"` // True if auto-classified
    ResolutionConfidence   string `json:"resolution_confidence"`   // "verified", "inferred"
    RelatedSharpEdge       string `json:"related_sharp_edge,omitempty"`
    AgentContext           string `json:"agent_context,omitempty"`
}

// Enums - GOgent-specific (not coupled to Compound Engineering)
var ProblemTypeEnum = []string{
    "build_error",
    "test_failure",
    "runtime_error",
    "performance_issue",
    "concurrency_issue",
    "type_error",
    "logic_error",
    "config_error",
}

var ComponentEnum = []string{
    "go_package",
    "go_test",
    "go_binary",
    "hook_binary",
    "config_file",
    "schema_file",
    "memory_system",
    "routing_system",
    "telemetry_system",
}

var RootCauseEnum = []string{
    "missing_import",
    "type_mismatch",
    "nil_pointer",
    "race_condition",
    "deadlock",
    "channel_misuse",
    "context_leak",
    "error_not_handled",
    "wrong_api",
    "config_error",
    "logic_error",
    "test_isolation",
    "missing_dependency",
}

var ResolutionTypeEnum = []string{
    "code_fix",
    "import_fix",
    "type_fix",
    "config_change",
    "test_fix",
    "dependency_update",
    "refactor",
}

var SeverityEnum = []string{
    "critical",
    "high",
    "medium",
    "low",
}

// LogSolvedProblem writes a SolvedProblem to solved-problems.jsonl with file locking
func LogSolvedProblem(problem *SolvedProblem, projectDir string) error {
    if problem.Timestamp == 0 {
        problem.Timestamp = time.Now().Unix()
    }

    data, err := json.Marshal(problem)
    if err != nil {
        return fmt.Errorf("marshal solved problem: %w", err)
    }

    solutionsPath := filepath.Join(projectDir, ".claude", "memory", "solved-problems.jsonl")

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(solutionsPath), 0755); err != nil {
        return fmt.Errorf("create directory: %w", err)
    }

    // === FILE LOCKING (BLOCKER FIX) ===
    lockPath := solutionsPath + ".lock"
    fileLock := flock.New(lockPath)

    if err := fileLock.Lock(); err != nil {
        return fmt.Errorf("acquire lock: %w", err)
    }
    defer fileLock.Unlock()

    f, err := os.OpenFile(solutionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("open file: %w", err)
    }
    defer f.Close()

    if _, err := f.Write(append(data, '\n')); err != nil {
        return fmt.Errorf("write: %w", err)
    }

    return nil
}

// RotateSolvedProblems archives old entries if file exceeds threshold
// Called by gogent-archive on session end
// HIGH FIX: Added file locking to prevent data loss during concurrent writes
func RotateSolvedProblems(projectDir string, maxEntries int) error {
    solutionsPath := filepath.Join(projectDir, ".claude", "memory", "solved-problems.jsonl")

    // === FILE LOCKING (HIGH FIX) ===
    // Acquire lock BEFORE reading to prevent race condition where:
    // 1. We read file
    // 2. Another goroutine appends
    // 3. We write truncated version → append is lost
    lockPath := solutionsPath + ".lock"
    fileLock := flock.New(lockPath)
    if err := fileLock.Lock(); err != nil {
        return fmt.Errorf("acquire lock: %w", err)
    }
    defer fileLock.Unlock()

    data, err := os.ReadFile(solutionsPath)
    if os.IsNotExist(err) {
        return nil
    }
    if err != nil {
        return err
    }

    lines := splitLines(data)
    if len(lines) <= maxEntries {
        return nil // Under threshold
    }

    // Archive oldest entries
    archivePath := filepath.Join(projectDir, ".claude", "memory", "session-archive",
        fmt.Sprintf("solved-problems-%d.jsonl", time.Now().Unix()))

    if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
        return err
    }

    // Write oldest to archive
    toArchive := lines[:len(lines)-maxEntries]
    if err := os.WriteFile(archivePath, joinLines(toArchive), 0644); err != nil {
        return err
    }

    // Keep newest in main file
    toKeep := lines[len(lines)-maxEntries:]
    return os.WriteFile(solutionsPath, joinLines(toKeep), 0644)
}

func splitLines(data []byte) [][]byte {
    var lines [][]byte
    start := 0
    for i, b := range data {
        if b == '\n' {
            if i > start {
                lines = append(lines, data[start:i])
            }
            start = i + 1
        }
    }
    if start < len(data) {
        lines = append(lines, data[start:])
    }
    return lines
}

func joinLines(lines [][]byte) []byte {
    var result []byte
    for _, line := range lines {
        result = append(result, line...)
        result = append(result, '\n')
    }
    return result
}
```

### File: `pkg/telemetry/classification.go`

```go
package telemetry

import (
    "path/filepath"
    "strings"
)

// Classification inference functions (moved from hook per review)

// InferProblemType maps error types to problem type enum
func InferProblemType(errorType, errorMsg string) string {
    errorType = strings.ToLower(errorType)
    errorMsg = strings.ToLower(errorMsg)

    switch {
    case strings.Contains(errorType, "build") || strings.Contains(errorMsg, "cannot find package"):
        return "build_error"
    case strings.Contains(errorType, "test") || strings.Contains(errorMsg, "fail"):
        return "test_failure"
    case strings.Contains(errorType, "nil") || strings.Contains(errorMsg, "nil pointer"):
        return "runtime_error"
    case strings.Contains(errorType, "race") || strings.Contains(errorMsg, "data race"):
        return "concurrency_issue"
    case strings.Contains(errorType, "type") || strings.Contains(errorMsg, "cannot use"):
        return "type_error"
    default:
        return "logic_error"
    }
}

// InferComponent maps file paths to component enum
func InferComponent(file string) string {
    switch {
    case strings.HasSuffix(file, "_test.go"):
        return "go_test"
    case strings.Contains(file, "/cmd/"):
        return "hook_binary"
    case strings.Contains(file, "/pkg/"):
        return "go_package"
    case strings.HasSuffix(file, ".json"):
        return "config_file"
    case strings.HasSuffix(file, ".yaml"):
        return "schema_file"
    case strings.Contains(file, "/memory/"):
        return "memory_system"
    case strings.Contains(file, "/routing/"):
        return "routing_system"
    case strings.Contains(file, "/telemetry/"):
        return "telemetry_system"
    default:
        return "go_package"
    }
}

// InferRootCause maps error patterns to root cause enum
func InferRootCause(errorType, errorMsg string) string {
    errorMsg = strings.ToLower(errorMsg)

    switch {
    case strings.Contains(errorMsg, "undefined"):
        return "missing_import"
    case strings.Contains(errorMsg, "cannot use") || strings.Contains(errorMsg, "type"):
        return "type_mismatch"
    case strings.Contains(errorMsg, "nil pointer"):
        return "nil_pointer"
    case strings.Contains(errorMsg, "race"):
        return "race_condition"
    case strings.Contains(errorMsg, "deadlock"):
        return "deadlock"
    case strings.Contains(errorMsg, "channel"):
        return "channel_misuse"
    case strings.Contains(errorMsg, "context"):
        return "context_leak"
    default:
        return "logic_error"
    }
}

// InferResolutionType maps tool name to resolution type enum
func InferResolutionType(toolName string) string {
    switch toolName {
    case "Edit":
        return "code_fix"
    case "Write":
        return "code_fix"
    case "Bash":
        return "config_change"
    default:
        return "code_fix"
    }
}

// InferSeverity maps failure count to severity
func InferSeverity(failureCount int) string {
    switch {
    case failureCount >= 5:
        return "critical"
    case failureCount >= 3:
        return "high"
    case failureCount >= 2:
        return "medium"
    default:
        return "low"
    }
}

// NormalizeFilePath converts to absolute path for consistent matching
func NormalizeFilePath(file string) string {
    if file == "" || file == "unknown" {
        return file
    }
    abs, err := filepath.Abs(file)
    if err != nil {
        return file
    }
    return abs
}
```

---

## Implementation

### File: `cmd/gogent-sharp-edge/resolution.go` (New File)

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/gofrs/flock"
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// === CONSTANTS ===

const (
    // ConsecutiveSuccessThreshold: require N successes to declare resolved
    // (Fixes HIGH: Success detection ambiguity)
    ConsecutiveSuccessThreshold = 2

    // PendingResolutionTTL: auto-expire after 1 hour
    // (Fixes MEDIUM: Circular dependency with gogent-archive)
    PendingResolutionTTL = 60 * 60 // seconds
)

// === TYPES ===

// PendingResolution tracks files awaiting resolution
type PendingResolution struct {
    File           string   `json:"file"`
    NormalizedFile string   `json:"normalized_file"` // Absolute path for matching
    ErrorType      string   `json:"error_type"`
    FirstFailure   int64    `json:"first_failure"`
    ExpiresAt      int64    `json:"expires_at"` // TTL-based cleanup
    FailedTools    []string `json:"failed_tools"`
    FailureCount   int      `json:"failure_count"`
    SuccessCount   int      `json:"success_count"` // Consecutive successes
    ErrorMessage   string   `json:"error_message"`
}

// === SESSION-SCOPED FILE PATHS (BLOCKER FIX) ===

// getPendingResolutionsPath returns session-scoped path
func getPendingResolutionsPath(projectDir, sessionID string) string {
    if sessionID == "" {
        sessionID = "unknown"
    }
    return filepath.Join(projectDir, ".claude", "tmp",
        fmt.Sprintf("pending-resolutions-%s.json", sessionID))
}

// === PERSISTENCE WITH LOCKING (BLOCKER FIX) ===

// loadPendingResolutions loads with file locking and TTL cleanup
func loadPendingResolutions(projectDir, sessionID string) (map[string]*PendingResolution, error) {
    path := getPendingResolutionsPath(projectDir, sessionID)

    // Acquire lock
    lockPath := path + ".lock"
    fileLock := flock.New(lockPath)
    if err := fileLock.Lock(); err != nil {
        return nil, fmt.Errorf("acquire lock: %w", err)
    }
    defer fileLock.Unlock()

    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return make(map[string]*PendingResolution), nil
    }
    if err != nil {
        return nil, err
    }

    var pending map[string]*PendingResolution
    if err := json.Unmarshal(data, &pending); err != nil {
        // Corrupted file - reset (MEDIUM: Error handling)
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: corrupted pending-resolutions, resetting\n")
        return make(map[string]*PendingResolution), nil
    }

    // TTL cleanup
    now := time.Now().Unix()
    for key, res := range pending {
        if res.ExpiresAt > 0 && now > res.ExpiresAt {
            delete(pending, key)
        }
    }

    return pending, nil
}

// savePendingResolutions persists with file locking
func savePendingResolutions(projectDir, sessionID string, pending map[string]*PendingResolution) error {
    path := getPendingResolutionsPath(projectDir, sessionID)

    // Acquire lock
    lockPath := path + ".lock"
    fileLock := flock.New(lockPath)
    if err := fileLock.Lock(); err != nil {
        return fmt.Errorf("acquire lock: %w", err)
    }
    defer fileLock.Unlock()

    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }

    data, err := json.MarshalIndent(pending, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

// === RESOLUTION DETECTION ===

// detectResolution checks if a successful tool use resolves a pending failure
func detectResolution(event *routing.PostToolEvent, projectDir string) *telemetry.SolvedProblem {
    // === VALIDATION (HIGH FIX) ===
    if event.SessionID == "" {
        return nil // Cannot track without session context
    }

    // Only check for resolution on success
    if routing.DetectFailure(event) != nil {
        return nil
    }

    // Get and normalize file path (MEDIUM FIX)
    file := routing.ExtractFilePath(event)
    normalizedFile := telemetry.NormalizeFilePath(file)
    if normalizedFile == "" || normalizedFile == "unknown" {
        return nil
    }

    // Load pending resolutions
    pending, err := loadPendingResolutions(projectDir, event.SessionID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: load pending: %v\n", err)
        return nil
    }

    // Check if this file had pending failures (use normalized path)
    resolution, exists := pending[normalizedFile]
    if !exists {
        return nil
    }

    // Increment success count
    resolution.SuccessCount++

    // === CONSECUTIVE SUCCESS THRESHOLD (HIGH FIX) ===
    if resolution.SuccessCount < ConsecutiveSuccessThreshold {
        // Not yet resolved - update and save
        if err := savePendingResolutions(projectDir, event.SessionID, pending); err != nil {
            fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: save pending: %v\n", err)
        }
        return nil
    }

    // Resolution confirmed! Build SolvedProblem
    now := time.Now().Unix()

    problem := &telemetry.SolvedProblem{
        SessionID:      event.SessionID,
        Timestamp:      now,
        File:           file,
        ErrorType:      resolution.ErrorType,
        ErrorMessage:   resolution.ErrorMessage,
        FailedTools:    resolution.FailedTools,
        FailureCount:   resolution.FailureCount,
        SuccessCount:   resolution.SuccessCount,
        SuccessTool:    event.ToolName,
        SuccessInput:   extractSuccessInput(event),
        TimeToResolve:  (now - resolution.FirstFailure) * 1000,

        // Classification (using pkg/telemetry functions)
        ProblemType:            telemetry.InferProblemType(resolution.ErrorType, resolution.ErrorMessage),
        Component:              telemetry.InferComponent(file),
        RootCause:              telemetry.InferRootCause(resolution.ErrorType, resolution.ErrorMessage),
        ResolutionType:         telemetry.InferResolutionType(event.ToolName),
        Severity:               telemetry.InferSeverity(resolution.FailureCount),
        Symptoms:               []string{resolution.ErrorMessage},
        InferredClassification: true,
        ResolutionConfidence:   "inferred",
    }

    // Remove from pending
    delete(pending, normalizedFile)
    if err := savePendingResolutions(projectDir, event.SessionID, pending); err != nil {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: save pending: %v\n", err)
    }

    return problem
}

// trackPendingResolution adds/updates a file in pending resolutions
func trackPendingResolution(event *routing.PostToolEvent, failure *routing.FailureInfo, projectDir string) {
    if event.SessionID == "" {
        return // Cannot track without session
    }

    pending, err := loadPendingResolutions(projectDir, event.SessionID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: load pending: %v\n", err)
        pending = make(map[string]*PendingResolution)
    }

    // Normalize file path for consistent matching
    normalizedFile := telemetry.NormalizeFilePath(failure.File)

    now := time.Now().Unix()

    if existing, exists := pending[normalizedFile]; exists {
        // Update existing - reset success count on new failure
        existing.FailureCount++
        existing.SuccessCount = 0 // Reset on new failure
        existing.FailedTools = append(existing.FailedTools, event.ToolName)
        existing.ExpiresAt = now + PendingResolutionTTL
        if failure.ErrorMatch != "" {
            existing.ErrorMessage = failure.ErrorMatch
        }
    } else {
        // Create new
        pending[normalizedFile] = &PendingResolution{
            File:           failure.File,
            NormalizedFile: normalizedFile,
            ErrorType:      failure.ErrorType,
            FirstFailure:   now,
            ExpiresAt:      now + PendingResolutionTTL,
            FailedTools:    []string{event.ToolName},
            FailureCount:   1,
            SuccessCount:   0,
            ErrorMessage:   failure.ErrorMatch,
        }
    }

    if err := savePendingResolutions(projectDir, event.SessionID, pending); err != nil {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: save pending: %v\n", err)
    }
}

// extractSuccessInput gets a truncated version of the successful input
func extractSuccessInput(event *routing.PostToolEvent) string {
    if event.ToolInput == nil {
        return ""
    }

    var input string
    switch event.ToolName {
    case "Edit":
        if newStr, ok := event.ToolInput["new_string"].(string); ok {
            input = newStr
        }
    case "Write":
        if content, ok := event.ToolInput["content"].(string); ok {
            input = content
        }
    case "Bash":
        if cmd, ok := event.ToolInput["command"].(string); ok {
            input = cmd
        }
    }

    // Truncate to 200 chars
    if len(input) > 200 {
        return input[:200] + "..."
    }
    return input
}
```

### Modifications to `cmd/gogent-sharp-edge/main.go` (CRITICAL FIX)

**⚠️ EXECUTION ORDER CRITICAL:** Resolution detection must run **BEFORE** the `if failure == nil` early return, otherwise it will never execute on successful tool uses.

```go
// === ML TOOL EVENT LOGGING (GOgent-087d) ===
// (existing code around line 103-107)

// === RESOLUTION DETECTION (MUST RUN BEFORE FAILURE CHECK) ===
// CRITICAL: This must run on ALL tool events, including successes
// If placed after "if failure == nil { return }", it will never execute
if solved := detectResolution(event, projectDir); solved != nil {
    if err := telemetry.LogSolvedProblem(solved, projectDir); err != nil {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Warning: log solution: %v\n", err)
        // === SURFACE ERROR (MEDIUM FIX) ===
        reminderMsg += fmt.Sprintf("\n⚠️ Failed to log resolution: %v", err)
    } else {
        fmt.Fprintf(os.Stderr, "[gogent-sharp-edge] Captured resolved problem: %s (%d attempts, %d successes)\n",
            solved.File, solved.FailureCount, solved.SuccessCount)
    }
}

// === EXISTING: SHARP-EDGE LOGIC (AFTER RESOLUTION DETECTION) ===
// Detect failure
failure := routing.DetectFailure(event)

// If no failure detected and no attention-gate messages, pass through
if failure == nil {
    if reminderMsg == "" && flushMsg == "" {
        fmt.Println("{}")
        return  // ← This early return is why resolution detection must run first
    }
    // ...
}

// ... rest of existing failure handling code ...

// === TRACK PENDING RESOLUTION ===
if failure != nil {
    trackPendingResolution(event, failure, projectDir)
}
```

**Why this order matters:**
1. `detectResolution()` only triggers on successful tool use (checks `routing.DetectFailure(event) != nil` internally)
2. Current main.go returns early on line 137: `if failure == nil { return }`
3. If resolution detection is after this line, it **never runs** on successes
4. Result: `solved-problems.jsonl` remains empty despite code correctness

**Integration Test:** See Appendix in Einstein analysis (`TestIntegration_ResolutionDetectionReachable`)

---

## Phase 4: Gemini Integration Specification (BLOCKER FIX)

### Input Files

| File | Purpose |
|------|---------|
| `.claude/memory/solved-problems.jsonl` | Problem-resolution pairs |
| `~/.claude/agents/*/sharp-edges.yaml` | Existing sharp edges |

### Gemini Prompt Template (INTEGRATION POINT 3 FIX)

Add to `memory-improvement` SKILL.md Phase 2:

```markdown
### solved-problems.jsonl Analysis

When you encounter solved-problems.jsonl in the context:

#### 0. Temporal Context Analysis (CRITICAL - Run First)

**Purpose:** Distinguish emerging patterns from one-off noise

Before pattern detection, analyze temporal distribution:

1. **Parse all entries** and extract session_id and timestamp fields
2. **Group by session_id** to count unique sessions
3. **Calculate date range:**
   - first_timestamp: earliest entry (Unix epoch → human date)
   - last_timestamp: most recent entry (Unix epoch → human date)
   - span_days: (last_timestamp - first_timestamp) / 86400
4. **Compute distribution metrics:**
   - total_sessions: count of unique session_id values
   - total_entries: count of all entries
   - avg_entries_per_session: total_entries / total_sessions
   - session_dates: map of session_id → date for temporal spread analysis

**Output format:**
```yaml
temporal_context:
  date_range: "2026-01-15 to 2026-01-30"
  span_days: 15
  total_sessions: 12
  total_entries: 187
  avg_entries_per_session: 15.6

  # Temporal spread (are sessions clustered or distributed?)
  session_distribution:
    - date: "2026-01-15"
      sessions: 3
      entries: 45
    - date: "2026-01-22"
      sessions: 5
      entries: 89
    - date: "2026-01-29"
      sessions: 4
      entries: 53

  # Data quality flags
  sparsity_warning: false  # true if total_sessions < 3 OR total_entries < 10
  single_session_dominated: false  # true if any session has >50% of entries
  temporal_clustering: "distributed"  # "clustered" if >70% entries in 20% of span_days

  # Interpretation guidance
  confidence_level: "high"  # high (≥10 sessions, ≥50 entries), medium (≥5 sessions, ≥20 entries), low (otherwise)
```

**Sparsity Warning Logic:**
```python
if total_sessions < 3 or total_entries < 10:
    sparsity_warning = true
    confidence_level = "low"
    # Flag: "Insufficient data for reliable pattern detection. Patterns below may be noise."
```

**Single Session Domination Check:**
```python
max_entries_in_session = max([count for session, count in session_entry_counts])
if max_entries_in_session / total_entries > 0.5:
    single_session_dominated = true
    # Flag: "Over 50% of entries from one session. May indicate one-off debugging loop, not recurring pattern."
```

**Why This Matters:**

| Scenario | Without Temporal Context | With Temporal Context |
|----------|-------------------------|----------------------|
| Same error 5x in 1 session (bad edit loop) | "Pattern: occurs 5x → HIGH priority" ❌ | "Single session dominated: likely one-off" ✅ |
| Same error 5x across 5 sessions | "Pattern: occurs 5x → HIGH priority" ✅ | "Distributed pattern: REAL issue" ✅ |
| 200 entries from 2 sessions | "Lots of data!" ❌ | "Sparsity warning: insufficient sessions" ✅ |

**Integration with solved-problems.jsonl schema:**

All required fields already exist in `SolvedProblem` struct:
- `session_id` (string) - for grouping
- `timestamp` (int64, Unix epoch) - for date range calculation

No schema changes needed. ✅

---

#### 1. Pattern Detection (Use Temporal Context)

Group entries by (problem_type, root_cause, component):
- Count occurrences of each pattern
- **NEW:** Count unique sessions per pattern (session diversity)
- Track files affected
- Calculate average time_to_resolve
- **NEW:** Check if pattern is temporally clustered or distributed

#### 2. Sharp Edge Correlation
For each pattern with occurrence_count >= 3:
1. Search agents/*/sharp-edges.yaml for matching symptom/error_type
2. If NO match found → recommend new sharp edge
3. If match found with different solution → flag for review

#### 3. Priority Scoring (UPDATED - Incorporates Temporal Context)

**Formula:**
```
Score = (occurrence_count × severity_weight × session_diversity_multiplier) / 30
```

**Components:**
- `occurrence_count`: Number of times pattern appears
- `severity_weight`: critical=4, high=3, medium=2, low=1
- `session_diversity_multiplier`:
  - 2.0 if pattern appears in ≥5 sessions (distributed, real pattern)
  - 1.5 if pattern appears in 3-4 sessions (moderate confidence)
  - 1.0 if pattern appears in 2 sessions (low confidence)
  - 0.5 if pattern appears in 1 session only (likely one-off noise)

**Example Calculations:**

| Pattern | Occurrences | Severity | Sessions | Multiplier | Score | Interpretation |
|---------|-------------|----------|----------|------------|-------|----------------|
| nil-pointer-init | 10 | high (3) | 6 | 2.0 | (10×3×2.0)/30 = 2.0 | **HIGH** priority (distributed pattern) |
| test-isolation | 15 | medium (2) | 1 | 0.5 | (15×2×0.5)/30 = 0.5 | **LOW** priority (one session, likely debugging loop) |
| race-condition | 5 | critical (4) | 4 | 1.5 | (5×4×1.5)/30 = 1.0 | **MEDIUM** priority (moderate distribution) |

**Higher scores → recommend first**

#### 4. Output Format (UPDATED - Includes Temporal Metadata)

```yaml
sharp_edge_recommendations:
  - priority: 1
    agent: go-pro
    action: add_sharp_edge
    sharp_edge:
      id: nil-pointer-struct-init
      severity: high
      category: runtime
      description: "Nil pointer when accessing uninitialized struct fields"
      symptom: "panic: runtime error: invalid memory address or nil pointer dereference"
      solution: |
        Always initialize struct with make() or literal:
        user := User{}  // GOOD
        var user *User  // BAD - nil
      auto_inject: true
    evidence:
      # Occurrence metrics
      occurrence_count: 5
      unique_sessions: 4  # NEW: Session diversity
      files_affected:
        - pkg/session/handoff.go
        - pkg/routing/validator.go

      # Temporal distribution (NEW)
      temporal_spread:
        first_occurrence: "2026-01-15"
        last_occurrence: "2026-01-28"
        span_days: 13
        distribution: "distributed"  # "distributed" or "clustered"

      # Resolution metrics
      avg_time_to_resolve_ms: 45000
      median_time_to_resolve_ms: 38000  # NEW: More robust than avg

      # Classification metadata
      inferred: true
      confidence: "high"  # NEW: Based on session diversity

  - priority: 2
    agent: go-pro
    action: add_sharp_edge
    sharp_edge:
      id: channel-close-race
      severity: critical
      category: concurrency
      description: "Race condition closing channels"
      symptom: "panic: send on closed channel"
      solution: |
        Use sync.Once to ensure single close:
        var closeOnce sync.Once
        closeOnce.Do(func() { close(ch) })
      auto_inject: true
    evidence:
      occurrence_count: 3
      unique_sessions: 3  # Distributed across sessions
      files_affected:
        - pkg/telemetry/ml_logging.go
      temporal_spread:
        first_occurrence: "2026-01-20"
        last_occurrence: "2026-01-29"
        span_days: 9
        distribution: "distributed"
      avg_time_to_resolve_ms: 120000
      median_time_to_resolve_ms: 95000
      inferred: true
      confidence: "high"

  - priority: 3
    agent: go-pro
    action: skip  # NEW: Explicitly mark low-confidence patterns as skipped
    reason: "Single session dominated - likely one-off debugging loop"
    pattern:
      problem_type: test_failure
      root_cause: test_isolation
      component: go_test
    evidence:
      occurrence_count: 12
      unique_sessions: 1  # ⚠️ All from one session
      temporal_spread:
        first_occurrence: "2026-01-22"
        last_occurrence: "2026-01-22"
        span_days: 0  # Same day
        distribution: "clustered"
      confidence: "low"  # Don't recommend sharp edge for noise
```

#### 5. Non-Recommendations (UPDATED - Includes Temporal Reasoning)

Also output patterns that DON'T need sharp edges, with temporal context:

```yaml
patterns_without_recommendation:
  - pattern: {problem_type: test_failure, root_cause: test_isolation}
    reason: "Already covered by go-pro sharp edge 'test-isolation'"
    matching_edge: go-pro/sharp-edges.yaml#test-isolation
    evidence:
      occurrence_count: 8
      unique_sessions: 5
      # Even though distributed, existing sharp edge covers it

  - pattern: {problem_type: build_error, root_cause: missing_import}
    reason: "Single session dominated - likely one-off refactoring session"
    temporal_analysis:
      occurrence_count: 15
      unique_sessions: 1
      span_days: 0
      distribution: "clustered"
      confidence: "low"
    # Not a recurring pattern worth documenting

  - pattern: {problem_type: logic_error, root_cause: logic_error}
    reason: "Too generic - classification likely inaccurate"
    evidence:
      occurrence_count: 20
      unique_sessions: 8
      avg_classification_confidence: 0.3  # Low inference confidence
    # Generic classifications don't produce actionable sharp edges

  - pattern: {problem_type: runtime_error, root_cause: nil_pointer}
    reason: "Pattern exists but time-to-resolve too fast (avg 5 seconds)"
    evidence:
      occurrence_count: 7
      unique_sessions: 4
      avg_time_to_resolve_ms: 5000
      median_time_to_resolve_ms: 3000
    # If developers resolve it in <10s consistently, not worth sharp edge overhead
```

**Non-Recommendation Criteria:**

1. **Already documented**: Matching sharp edge exists
2. **Low confidence**: Single session dominated OR insufficient sessions (<3)
3. **Too generic**: Classification is ambiguous (problem_type == root_cause)
4. **Fast resolution**: avg_time_to_resolve < 10000ms (developers already know solution)
5. **Insufficient occurrences**: occurrence_count < 3 (even if distributed)
```

### Memory Improvement SKILL.md Updates (HIGH FIX + INTEGRATION POINT 3 FIX)

**File:** `~/.claude/skills/memory-improvement/SKILL.md`

**Changes required:**

1. **Phase 1: Context Gathering** - Exclude archived files
```bash
# Phase 1: Context Gathering - UPDATED
FILES=$(find ~/.claude/memory ~/.claude/agents ~/.claude/conventions -type f \
    \( -name "*.md" -o -name "*.yaml" -o -name "*.jsonl" \) \
    -not -path "*session-archive*")  # Exclude archived files
```

2. **Phase 2: Gemini Prompt** - Add temporal context analysis (INTEGRATION POINT 3 FIX)

Add the complete updated prompt from Phase 4 section above, including:
- Section 0: Temporal Context Analysis (runs first)
- Updated Section 1: Pattern Detection (uses session_diversity)
- Updated Section 3: Priority Scoring (includes session_diversity_multiplier)
- Updated Section 4: Output Format (includes temporal metadata)
- Updated Section 5: Non-Recommendations (includes temporal reasoning)

**Verification:**
After updating SKILL.md, test by:
```bash
# 1. Generate sample solved-problems.jsonl with multi-session data
# 2. Run /memory-improvement
# 3. Verify Gemini output includes temporal_context section
# 4. Verify recommendations include unique_sessions and temporal_spread
```

---

## Error Handling Matrix (HIGH FIX)

| Scenario | Behavior |
|----------|----------|
| Corrupted pending-resolutions.json | Reset to empty map, log warning |
| Corrupted solved-problems.jsonl | Skip malformed lines, continue |
| File deleted during tracking | Remove from pending, log info |
| Disk full on write | Log error to stderr, surface in additionalContext |
| Multiple pending for same file | Track by normalized path (dedupe) |
| SessionID empty | Skip resolution tracking |
| Lock acquisition fails | Return error, don't corrupt data |

---

## Hook Interaction Matrix

| Hook | Interaction | Notes |
|------|-------------|-------|
| `gogent-sharp-edge` | **MODIFIED** | Adds resolution tracking |
| `gogent-load-context` | None | No changes needed |
| `gogent-validate` | None | No changes needed |
| `gogent-archive` | **EXTEND** | Cleanup + rotation |
| `gogent-agent-endstate` | None | No changes needed |

### gogent-archive Additions

```go
// Add to SessionEnd handler:

// 1. Clean up session-scoped pending resolutions
pendingPath := filepath.Join(projectDir, ".claude", "tmp",
    fmt.Sprintf("pending-resolutions-%s.json", sessionID))
os.Remove(pendingPath)
os.Remove(pendingPath + ".lock")

// 2. Rotate solved-problems.jsonl if over threshold
if err := telemetry.RotateSolvedProblems(projectDir, 500); err != nil {
    fmt.Fprintf(os.Stderr, "[gogent-archive] Warning: rotation failed: %v\n", err)
}
```

---

## Test Scenarios (HIGH FIX)

### Unit Tests: `pkg/telemetry/solved_problem_test.go`

```go
func TestLogSolvedProblem(t *testing.T)           // Basic write
func TestLogSolvedProblem_Concurrent(t *testing.T) // File locking
func TestRotateSolvedProblems(t *testing.T)       // Rotation threshold
func TestInferProblemType(t *testing.T)           // Classification
func TestNormalizeFilePath(t *testing.T)          // Path normalization
```

### Unit Tests: `cmd/gogent-sharp-edge/resolution_test.go`

```go
func TestPendingResolutions_SessionIsolation(t *testing.T) // Different session IDs
func TestPendingResolutions_TTLExpiry(t *testing.T)        // Auto-cleanup
func TestPendingResolutions_Concurrent(t *testing.T)       // File locking
func TestDetectResolution_ConsecutiveSuccess(t *testing.T) // Threshold
func TestDetectResolution_NoSessionID(t *testing.T)        // Validation
```

### Integration Tests

```go
func TestResolution_EndToEnd(t *testing.T) {
    // 1. Simulate failure event (Edit tool fails)
    // 2. Verify pending resolution created
    // 3. Simulate success event #1 (Edit tool succeeds)
    // 4. Verify still pending (threshold not met)
    // 5. Simulate success event #2
    // 6. Verify SolvedProblem written to JSONL
    // 7. Verify pending resolution removed
}

func TestResolution_MultiSession(t *testing.T) {
    // 1. Session A fails on file X
    // 2. Session B succeeds on file X
    // 3. Verify NO cross-contamination (different session IDs)
}
```

### Manual Test Checklist

- [ ] Create intentional failure (Edit with bad code)
- [ ] Fix the issue (Edit with correct code)
- [ ] Fix again (second success)
- [ ] Verify `solved-problems.jsonl` contains entry
- [ ] Run `/memory-improvement`
- [ ] Verify Gemini sees and parses the entry

---

## Deferred Issues from v2 (Now Addressed in v3)

### MEDIUM-5: Taxonomy Schema Versioning

**Problem:** External dependency on problem taxonomy without versioning.

**Solution:**

#### File: `~/.claude/schemas/problem-taxonomy.yaml`

```yaml
# GOgent Problem Taxonomy Schema
# Version: 1.0.0
# Source: Adapted from Compound Engineering patterns
# Divergence: GOgent-specific enum values for Go ecosystem

version: "1.0.0"
last_updated: "2026-01-30"

problem_types:
  - build_error
  - test_failure
  - runtime_error
  - performance_issue
  - concurrency_issue
  - type_error
  - logic_error
  - config_error

components:
  - go_package
  - go_test
  - go_binary
  - hook_binary
  - config_file
  - schema_file
  - memory_system
  - routing_system
  - telemetry_system

root_causes:
  - missing_import
  - type_mismatch
  - nil_pointer
  - race_condition
  - deadlock
  - channel_misuse
  - context_leak
  - error_not_handled
  - wrong_api
  - config_error
  - logic_error
  - test_isolation
  - missing_dependency

resolution_types:
  - code_fix
  - import_fix
  - type_fix
  - config_change
  - test_fix
  - dependency_update
  - refactor

severity_levels:
  - critical
  - high
  - medium
  - low
```

#### File: `pkg/telemetry/schema_loader.go` (NEW - BLOCKER FIX)

```go
package telemetry

import (
    "embed"
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

//go:embed schemas/problem-taxonomy.yaml
var embeddedTaxonomy embed.FS

const TaxonomySchemaVersion = "1.0.0"  // Expected schema version

type TaxonomySchema struct {
    Version        string   `yaml:"version"`
    LastUpdated    string   `yaml:"last_updated"`
    ProblemTypes   []string `yaml:"problem_types"`
    Components     []string `yaml:"components"`
    RootCauses     []string `yaml:"root_causes"`
    ResolutionTypes []string `yaml:"resolution_types"`
    SeverityLevels  []string `yaml:"severity_levels"`
}

// LoadTaxonomySchema loads and validates taxonomy with THREE-TIER FALLBACK:
// 1. Try ~/.claude/schemas/problem-taxonomy.yaml (user-modified)
// 2. Try creating from embedded resource if missing
// 3. Fall back to hardcoded enums if all fails
func LoadTaxonomySchema() (*TaxonomySchema, error) {
    home := os.Getenv("HOME")
    schemaPath := filepath.Join(home, ".claude", "schemas", "problem-taxonomy.yaml")

    // Tier 1: Try user schema
    if data, err := os.ReadFile(schemaPath); err == nil {
        schema, parseErr := parseTaxonomySchema(data)
        if parseErr == nil {
            return schema, nil
        }
        // Corrupted user schema - log warning and fall through
        fmt.Fprintf(os.Stderr, "[telemetry] Warning: corrupted schema at %s: %v\n", schemaPath, parseErr)
    }

    // Tier 2: Create from embedded resource
    embeddedData, err := embeddedTaxonomy.ReadFile("schemas/problem-taxonomy.yaml")
    if err == nil {
        // Write to user location
        if err := os.MkdirAll(filepath.Dir(schemaPath), 0755); err == nil {
            os.WriteFile(schemaPath, embeddedData, 0644)
        }

        schema, parseErr := parseTaxonomySchema(embeddedData)
        if parseErr == nil {
            return schema, nil
        }
    }

    // Tier 3: Hardcoded fallback
    fmt.Fprintf(os.Stderr, "[telemetry] Warning: using hardcoded taxonomy (schema file unavailable)\n")
    return &TaxonomySchema{
        Version:        TaxonomySchemaVersion,
        LastUpdated:    "embedded",
        ProblemTypes:   ProblemTypeEnum,
        Components:     ComponentEnum,
        RootCauses:     RootCauseEnum,
        ResolutionTypes: ResolutionTypeEnum,
        SeverityLevels:  SeverityEnum,
    }, nil
}

func parseTaxonomySchema(data []byte) (*TaxonomySchema, error) {
    var schema TaxonomySchema
    if err := yaml.Unmarshal(data, &schema); err != nil {
        return nil, fmt.Errorf("parse taxonomy schema: %w", err)
    }

    // Validate version format (semver)
    if !isValidSemver(schema.Version) {
        return nil, fmt.Errorf("invalid schema version: %s", schema.Version)
    }

    // Warn if version mismatch
    if schema.Version != TaxonomySchemaVersion {
        fmt.Fprintf(os.Stderr, "[telemetry] Warning: schema version mismatch (expected %s, got %s)\n",
            TaxonomySchemaVersion, schema.Version)
    }

    return &schema, nil
}

func isValidSemver(version string) bool {
    // Basic semver check: X.Y.Z
    parts := strings.Split(version, ".")
    if len(parts) != 3 {
        return false
    }
    for _, part := range parts {
        if _, err := strconv.Atoi(part); err != nil {
            return false
        }
    }
    return true
}
```

#### Modifications to `pkg/telemetry/solved_problem.go`

```go
// Add schema version field
type SolvedProblem struct {
    // ... existing fields ...

    // Metadata
    InferredClassification bool   `json:"inferred_classification"`
    ResolutionConfidence   string `json:"resolution_confidence"`
    RelatedSharpEdge       string `json:"related_sharp_edge,omitempty"`
    AgentContext           string `json:"agent_context,omitempty"`
    SchemaVersion          string `json:"schema_version"`  // NEW: Taxonomy version
}

// ValidateAgainstSchema checks if enum values exist in taxonomy
func (p *SolvedProblem) ValidateAgainstSchema(schema *TaxonomySchema) []string {
    var violations []string

    if !contains(schema.ProblemTypes, p.ProblemType) {
        violations = append(violations, fmt.Sprintf("unknown problem_type: %s", p.ProblemType))
    }
    if !contains(schema.Components, p.Component) {
        violations = append(violations, fmt.Sprintf("unknown component: %s", p.Component))
    }
    if !contains(schema.RootCauses, p.RootCause) {
        violations = append(violations, fmt.Sprintf("unknown root_cause: %s", p.RootCause))
    }
    if !contains(schema.ResolutionTypes, p.ResolutionType) {
        violations = append(violations, fmt.Sprintf("unknown resolution_type: %s", p.ResolutionType))
    }
    if !contains(schema.SeverityLevels, p.Severity) {
        violations = append(violations, fmt.Sprintf("unknown severity: %s", p.Severity))
    }

    return violations
}
```

#### Schema Distribution Mechanism (BLOCKER FIX)

| Stage | Mechanism | Fallback |
|-------|-----------|----------|
| **Installation** | Embedded in binary, extracted on first use | Hardcoded enums if extraction fails |
| **Updates** | Manual edit of ~/.claude/schemas/problem-taxonomy.yaml | LoadTaxonomySchema() detects and warns on version mismatch |
| **Corruption** | Logs warning, falls back to embedded or hardcoded | Never blocks execution |

**Update mechanism (future):**
```bash
# Check for taxonomy drift
gogent-ml-export taxonomy-check
# Outputs: "5 problem types not in schema" or "Schema up to date"
```

### MEDIUM-6: Edge Case Test Coverage

#### New Test File: `cmd/gogent-sharp-edge/resolution_edge_cases_test.go`

```go
package main

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

// TestPendingResolutions_FileDeleted verifies graceful handling when tracked file is deleted
func TestPendingResolutions_FileDeleted(t *testing.T) {
    tmpDir := t.TempDir()
    sessionID := "test-session"

    // Create file and track failure
    testFile := filepath.Join(tmpDir, "test.go")
    os.WriteFile(testFile, []byte("package main"), 0644)

    failure := &routing.FailureInfo{
        File:      testFile,
        ErrorType: "build_error",
    }

    event := &routing.PostToolEvent{
        SessionID: sessionID,
        ToolName:  "Edit",
    }

    trackPendingResolution(event, failure, tmpDir)

    // Delete the file
    os.Remove(testFile)

    // Success on deleted file should clean up pending resolution
    solvedEvent := &routing.PostToolEvent{
        SessionID: sessionID,
        ToolName:  "Edit",
        ToolInput: map[string]interface{}{
            "file_path": testFile,
        },
        ToolResponse: map[string]interface{}{
            "success": true,
        },
    }

    solved := detectResolution(solvedEvent, tmpDir)

    // Should still detect resolution even though file is gone
    if solved == nil {
        t.Error("Expected resolution detection even for deleted file")
    }
}

// TestResolution_RapidSuccession tests fail/success within <100ms
func TestResolution_RapidSuccession(t *testing.T) {
    tmpDir := t.TempDir()
    sessionID := "test-session"
    testFile := filepath.Join(tmpDir, "rapid.go")

    // Rapid failure
    failure := &routing.FailureInfo{
        File:      testFile,
        ErrorType: "test_failure",
        Timestamp: time.Now().Unix(),
    }

    event := &routing.PostToolEvent{
        SessionID: sessionID,
        ToolName:  "Edit",
    }

    trackPendingResolution(event, failure, tmpDir)

    // Immediate success (< 100ms later)
    time.Sleep(50 * time.Millisecond)

    successEvent := &routing.PostToolEvent{
        SessionID: sessionID,
        ToolName:  "Edit",
        ToolInput: map[string]interface{}{
            "file_path": testFile,
        },
        ToolResponse: map[string]interface{}{
            "success": true,
        },
    }

    // First success - not yet resolved (needs 2 consecutive)
    solved := detectResolution(successEvent, tmpDir)
    if solved != nil {
        t.Error("Should not resolve on first success")
    }

    // Second success - should resolve
    solved = detectResolution(successEvent, tmpDir)
    if solved == nil {
        t.Fatal("Expected resolution on second consecutive success")
    }

    // Time to resolve should be very small
    if solved.TimeToResolve > 200 {
        t.Errorf("Time to resolve too high for rapid succession: %d ms", solved.TimeToResolve)
    }
}

// TestClassification_AmbiguousError tests multiple keyword matches
func TestClassification_AmbiguousError(t *testing.T) {
    // Error message with multiple classification signals
    errorMsg := "test failed: cannot build package due to type mismatch"

    // Should prioritize "test" over "build"
    problemType := telemetry.InferProblemType("", errorMsg)
    if problemType != "test_failure" {
        t.Errorf("Expected test_failure for test-related error, got %s", problemType)
    }

    // Root cause should detect type mismatch
    rootCause := telemetry.InferRootCause("", errorMsg)
    if rootCause != "type_mismatch" {
        t.Errorf("Expected type_mismatch, got %s", rootCause)
    }
}

// TestRotation_DuringWrite simulates rotation racing with active writes
func TestRotation_DuringWrite(t *testing.T) {
    tmpDir := t.TempDir()

    // Write 600 entries concurrently while rotation is happening
    done := make(chan bool)

    // Writer goroutine
    go func() {
        for i := 0; i < 600; i++ {
            problem := &telemetry.SolvedProblem{
                SessionID: fmt.Sprintf("session-%d", i),
                File:      fmt.Sprintf("file-%d.go", i),
            }
            telemetry.LogSolvedProblem(problem, tmpDir)
            time.Sleep(1 * time.Millisecond)
        }
        done <- true
    }()

    // Rotation goroutine (triggers at 500 entries)
    go func() {
        time.Sleep(250 * time.Millisecond)
        telemetry.RotateSolvedProblems(tmpDir, 500)
        done <- true
    }()

    // Wait for both
    <-done
    <-done

    // Verify file integrity (no corruption)
    solutionsPath := filepath.Join(tmpDir, ".claude", "memory", "solved-problems.jsonl")
    data, err := os.ReadFile(solutionsPath)
    if err != nil {
        t.Fatalf("Failed to read solutions file: %v", err)
    }

    // Should be valid JSONL (each line parseable)
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
        if line == "" {
            continue
        }
        var problem telemetry.SolvedProblem
        if err := json.Unmarshal([]byte(line), &problem); err != nil {
            t.Errorf("Corrupted JSONL line: %s. Error: %v", line, err)
        }
    }
}
```

### LOW-1: Classification Accuracy Improvements

#### Additions to `pkg/telemetry/classification.go`

```go
// ClassificationMetrics tracks inference accuracy
type ClassificationMetrics struct {
    TotalInferences      int     `json:"total_inferences"`
    ManualCorrections    int     `json:"manual_corrections"`
    AccuracyRate         float64 `json:"accuracy_rate"`
    LastUpdated          int64   `json:"last_updated"`
}

// SolvedProblem additions
type SolvedProblem struct {
    // ... existing fields ...

    // Classification Quality
    InferredClassification bool   `json:"inferred_classification"`
    ReviewedBy             string `json:"reviewed_by,omitempty"`  // "human", "gemini", or empty
    CorrectedFrom          string `json:"corrected_from,omitempty"` // Original inferred value if corrected
}

// ImprovedInferProblemType uses context-aware keyword matching
func ImprovedInferProblemType(errorType, errorMsg, filePath string) string {
    errorType = strings.ToLower(errorType)
    errorMsg = strings.ToLower(errorMsg)

    // Priority 1: Explicit test context
    if strings.Contains(filePath, "_test.go") || strings.HasPrefix(errorMsg, "--- fail:") {
        return "test_failure"
    }

    // Priority 2: Build errors (avoid "rebuild" false positive)
    if strings.Contains(errorMsg, "cannot find package") ||
       strings.Contains(errorMsg, "undefined:") ||
       (strings.Contains(errorType, "build") && !strings.Contains(errorMsg, "rebuild")) {
        return "build_error"
    }

    // Priority 3: Runtime errors
    if strings.Contains(errorMsg, "panic:") || strings.Contains(errorMsg, "nil pointer") {
        return "runtime_error"
    }

    // Priority 4: Concurrency
    if strings.Contains(errorMsg, "data race") || strings.Contains(errorMsg, "deadlock") {
        return "concurrency_issue"
    }

    // Priority 5: Type errors (more specific patterns)
    if strings.Contains(errorMsg, "cannot use") && strings.Contains(errorMsg, "as type") {
        return "type_error"
    }

    // Default
    return "logic_error"
}

// LogClassificationMetrics tracks accuracy over time
func LogClassificationMetrics(projectDir string, inferred bool, corrected bool) error {
    metricsPath := filepath.Join(projectDir, ".claude", "memory", "classification-metrics.json")

    var metrics ClassificationMetrics
    data, err := os.ReadFile(metricsPath)
    if err == nil {
        json.Unmarshal(data, &metrics)
    }

    metrics.TotalInferences++
    if corrected {
        metrics.ManualCorrections++
    }

    if metrics.TotalInferences > 0 {
        metrics.AccuracyRate = 1.0 - (float64(metrics.ManualCorrections) / float64(metrics.TotalInferences))
    }
    metrics.LastUpdated = time.Now().Unix()

    outData, _ := json.MarshalIndent(metrics, "", "  ")
    return os.WriteFile(metricsPath, outData, 0644)
}
```

#### Manual Correction Workflow (MEDIUM ADDITION - Phase 4 Integration)

**CLI Tool: `gogent-ml-export review-classifications`**

```bash
# Display recent inferred classifications for review
$ gogent-ml-export review-classifications --limit 20

Recent Classifications (inferred):
  [1] file: pkg/routing/validator.go
      inferred: build_error → type_mismatch → type_fix
      confidence: low (ambiguous error message)

  [2] file: cmd/gogent-sharp-edge/main.go
      inferred: logic_error → error_not_handled → code_fix
      confidence: medium

  [3] file: pkg/telemetry/ml_logging.go
      inferred: test_failure → test_isolation → test_fix
      confidence: high

Options: [c]orrect, [s]kip, [q]uit
> c 1

Correcting classification for pkg/routing/validator.go:

Current: build_error → type_mismatch → type_fix

problem_type: [build_error|test_failure|runtime_error|...]
> runtime_error

root_cause: [missing_import|type_mismatch|nil_pointer|...]
> nil_pointer

resolution_type: [code_fix|import_fix|type_fix|...]
> code_fix

✅ Updated solved-problems.jsonl with correction
   - reviewed_by: "human"
   - corrected_from: "build_error → type_mismatch → type_fix"

Accuracy rate: 92% (18/20 correct)
```

**Implementation (deferred to v1.1 unless time permits):**

File: `cmd/gogent-ml-export/review_classifications.go`

```go
func ReviewClassifications(projectDir string, limit int) error {
    // 1. Read solved-problems.jsonl
    // 2. Filter for inferred_classification: true
    // 3. Display recent entries with confidence scores
    // 4. Accept user corrections
    // 5. Update JSONL with reviewed_by, corrected_from fields
    // 6. Recalculate accuracy metrics
}
```

### LOW-2: Resolution Re-emergence Tracking

#### Additions to `cmd/gogent-sharp-edge/resolution.go`

```go
const (
    ConsecutiveSuccessThreshold = 2
    PendingResolutionTTL = 60 * 60
    ResolutionCooldownPeriod = 3600  // NEW: 1 hour cooldown after resolution
)

// ResolvedProblemHistory tracks recent resolutions for re-emergence detection
type ResolvedProblemHistory struct {
    File         string `json:"file"`
    ErrorType    string `json:"error_type"`
    ResolvedAt   int64  `json:"resolved_at"`
    ProblemHash  string `json:"problem_hash"`  // Hash of file + error_type for matching
}

// CheckReemergence detects if a failure is a re-emergence of recently resolved problem
func CheckReemergence(failure *routing.FailureInfo, projectDir, sessionID string) (bool, *ResolvedProblemHistory) {
    historyPath := filepath.Join(projectDir, ".claude", "tmp",
        fmt.Sprintf("resolved-history-%s.json", sessionID))

    data, err := os.ReadFile(historyPath)
    if err != nil {
        return false, nil
    }

    var history []ResolvedProblemHistory
    if err := json.Unmarshal(data, &history); err != nil {
        return false, nil
    }

    now := time.Now().Unix()
    problemHash := hashProblem(failure.File, failure.ErrorType)

    for _, resolved := range history {
        // Check if same problem re-emerged within cooldown period
        if resolved.ProblemHash == problemHash {
            timeSinceResolution := now - resolved.ResolvedAt
            if timeSinceResolution < ResolutionCooldownPeriod {
                return true, &resolved
            }
        }
    }

    return false, nil
}

// RecordResolution adds to resolved history
func RecordResolution(solved *telemetry.SolvedProblem, projectDir, sessionID string) error {
    historyPath := filepath.Join(projectDir, ".claude", "tmp",
        fmt.Sprintf("resolved-history-%s.json", sessionID))

    var history []ResolvedProblemHistory
    data, _ := os.ReadFile(historyPath)
    if len(data) > 0 {
        json.Unmarshal(data, &history)
    }

    history = append(history, ResolvedProblemHistory{
        File:        solved.File,
        ErrorType:   solved.ErrorType,
        ResolvedAt:  time.Now().Unix(),
        ProblemHash: hashProblem(solved.File, solved.ErrorType),
    })

    // Keep only last 50 resolutions
    if len(history) > 50 {
        history = history[len(history)-50:]
    }

    outData, _ := json.MarshalIndent(history, "", "  ")
    return os.WriteFile(historyPath, outData, 0644)
}

func hashProblem(file, errorType string) string {
    h := sha256.New()
    h.Write([]byte(file + ":" + errorType))
    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

#### Re-emergence Tracking Scope (MEDIUM CAVEAT)

**Current implementation: Session-scoped only**

| Scenario | Detection | Rationale |
|----------|-----------|-----------|
| Problem resolved in Session A, re-emerges in Session A | ✅ DETECTED | Same session ID, history available |
| Problem resolved in Session A, re-emerges in Session B | ❌ NOT DETECTED | Different session IDs, different history files |

**Why session-scoped?**
- Cross-session re-emergence may have different root cause (not true re-emergence)
- Session-scoped history files are cleaned up automatically (no persistent clutter)
- True re-emergence within a session is more actionable

**Future enhancement (v1.2):**
If cross-session tracking is needed:
- Move history to `.claude/memory/resolved-history.jsonl` (persistent, project-wide)
- Add timestamp-based expiry (e.g., only track last 7 days)
- Trade-off: More persistent state, but detects cross-session patterns

```

// Modify detectResolution to check for re-emergence
func detectResolution(event *routing.PostToolEvent, projectDir string) *telemetry.SolvedProblem {
    // ... existing validation ...

    // NEW: Check for re-emergence before tracking
    file := routing.ExtractFilePath(event)
    pending, _ := loadPendingResolutions(projectDir, event.SessionID)

    if resolution, exists := pending[normalizedFile]; exists {
        // Check if this is a re-emergence
        isReemergence, previousResolution := CheckReemergence(
            &routing.FailureInfo{File: file, ErrorType: resolution.ErrorType},
            projectDir,
            event.SessionID,
        )

        if isReemergence {
            // Flag as re-emergence in metadata
            problem.ResolutionConfidence = "low_confidence_reemergence"
            problem.AgentContext = fmt.Sprintf(
                "Re-emerged %d seconds after previous resolution",
                time.Now().Unix()-previousResolution.ResolvedAt,
            )
        }
    }

    // ... rest of existing logic ...

    // Record in history after successful resolution
    if problem != nil {
        RecordResolution(problem, projectDir, event.SessionID)
    }

    return problem
}
```

### LOW-3: Mutation Testing Strategy

#### New File: `scripts/mutation-test.sh` (HIGH FIX: Updated to use go-gremlins)

```bash
#!/usr/bin/env bash
# Mutation testing for critical paths in solved_problem.go and resolution.go
# HIGH FIX: Using go-gremlins (actively maintained) instead of archived go-mutesting

set -euo pipefail

MUTANTS_DIR=".claude/tmp/mutants"
CRITICAL_PACKAGES=(
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
    "github.com/Bucket-Chemist/GOgent-Fortress/cmd/gogent-sharp-edge"
)

echo "=== Mutation Testing for Critical Paths ==="
mkdir -p "$MUTANTS_DIR"

# Install go-gremlins if not available (actively maintained, 2025+)
if ! command -v gremlins &> /dev/null; then
    echo "Installing go-gremlins..."
    go install github.com/gtramontina/go-gremlins@latest
fi

# Run mutation testing on each critical package
for pkg in "${CRITICAL_PACKAGES[@]}"; do
    echo ""
    echo "=== Mutating $pkg ==="

    gremlins unleash \
        --tags "critical" \
        --output "$MUTANTS_DIR/report-$(basename $pkg).json" \
        --threshold 95 \
        "$pkg"

    # Check if threshold met
    if [ $? -eq 0 ]; then
        echo "✅ $pkg: Mutation coverage ≥95%"
    else
        echo "❌ $pkg: Mutation coverage <95% (see report)"
    fi
done

# Generate summary
echo ""
echo "=== Mutation Testing Summary ==="
echo "Threshold: 95% mutant kill rate on critical paths"
echo ""
echo "Reports available in: $MUTANTS_DIR"
echo ""

# Display JSON reports
for report in "$MUTANTS_DIR"/*.json; do
    if [ -f "$report" ]; then
        echo "$(basename $report):"
        jq -r '.summary | "  Killed: \(.killed)/\(.total) (\(.coverage)%)"' "$report" 2>/dev/null || echo "  (parse error)"
    fi
done
```

#### Acceptance Criteria Update

Replace "85% coverage" with:

```markdown
### Test Coverage (Critical Path Focus)

- [ ] Unit tests: 100% coverage on critical paths:
  - `LogSolvedProblem` (file locking, JSONL append)
  - `RotateSolvedProblems` (archive threshold, race safety)
  - `detectResolution` (consecutive success threshold, session isolation)
  - `loadPendingResolutions` (TTL expiry, file locking)

- [ ] Mutation testing: >95% mutant kill rate on critical paths
  - Run `scripts/mutation-test.sh`
  - Verify mutations in locking code are caught
  - Verify mutations in threshold logic are caught

- [ ] Edge case coverage:
  - File deletion during tracking
  - Rapid fail/success succession (<100ms)
  - Ambiguous error classification
  - Rotation during concurrent writes

- [ ] Race detector clean: `go test -race ./...`
```

### LOW-4: Extract Magic Numbers to Constants

#### File: `pkg/telemetry/constants.go` (NEW)

```go
package telemetry

const (
    // Time conversion
    MillisecondsPerSecond = 1000

    // String truncation
    MaxSuccessInputLength = 200
    MaxErrorMessageLength = 500

    // Rotation thresholds
    DefaultRotationThreshold = 500
    ArchiveRetentionDays     = 90

    // Resolution detection
    ConsecutiveSuccessThreshold = 2  // Moved from resolution.go
    PendingResolutionTTL        = 3600  // 1 hour in seconds
    ResolutionCooldownPeriod    = 3600  // 1 hour in seconds

    // Classification confidence
    HighConfidenceThreshold   = 0.9
    MediumConfidenceThreshold = 0.7

    // History retention
    MaxResolvedHistorySize = 50
    MaxPendingResolutions  = 100
)
```

#### Update references in `cmd/gogent-sharp-edge/resolution.go`

```go
import "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"

// Remove local constants, use telemetry.* instead
const (
    ConsecutiveSuccessThreshold = telemetry.ConsecutiveSuccessThreshold
    PendingResolutionTTL = telemetry.PendingResolutionTTL
)

// In extractSuccessInput:
if len(input) > telemetry.MaxSuccessInputLength {
    return input[:telemetry.MaxSuccessInputLength] + "..."
}

// In time calculations:
TimeToResolve: (now - resolution.FirstFailure) * telemetry.MillisecondsPerSecond,
```

---

## Acceptance Criteria

### Code Deliverables (UPDATED - includes blocker/high fixes)
- [ ] `pkg/telemetry/solved_problem.go` - Types and logging (with locking fix)
- [ ] `pkg/telemetry/classification.go` - Inference functions (context-aware)
- [ ] `pkg/telemetry/schema_loader.go` - **NEW**: Taxonomy schema loading with 3-tier fallback
- [ ] `pkg/telemetry/constants.go` - **NEW**: Magic numbers extracted
- [ ] `pkg/telemetry/schemas/problem-taxonomy.yaml` - **NEW**: Embedded schema resource
- [ ] `cmd/gogent-sharp-edge/resolution.go` - Resolution tracking (with re-emergence detection)
- [ ] `cmd/gogent-sharp-edge/resolution_edge_cases_test.go` - **NEW**: Edge case tests
- [ ] `cmd/gogent-sharp-edge/main.go` - Integration points
- [ ] `scripts/mutation-test.sh` - **NEW**: Mutation testing with go-gremlins

### Schema Updates
- [ ] `routing-schema.json` - Document new files
- [ ] `memory-improvement/SKILL.md` - Gemini prompt updates

### Test Coverage (BLOCKER FIX: Clarified targets)
- [ ] Unit tests: **100% coverage on critical paths**:
  - `LogSolvedProblem` (file locking, JSONL append)
  - `RotateSolvedProblems` (archive threshold, race safety with locking)
  - `detectResolution` (consecutive success threshold, session isolation)
  - `loadPendingResolutions` (TTL expiry, file locking)
- [ ] Unit tests: **>85% coverage on non-critical paths**:
  - Inference functions (classification.go)
  - String formatters and helpers
  - Schema validation logic
- [ ] Mutation testing: **>95% mutant kill rate** on critical paths
  - Run `scripts/mutation-test.sh`
  - Verify mutations in locking code are caught
  - Verify mutations in threshold logic are caught
- [ ] Edge case coverage (resolution_edge_cases_test.go):
  - File deletion during tracking
  - Rapid fail/success succession (<100ms)
  - Ambiguous error classification
  - Rotation during concurrent writes
- [ ] Integration test: End-to-end flow (existing test)
- [ ] Integration test (NEW): Gemini temporal context analysis
  - Generate synthetic solved-problems.jsonl with:
    - 50 entries across 5 sessions (distributed pattern)
    - 20 entries in 1 session (clustered pattern)
  - Run `/memory-improvement`
  - Verify Gemini output includes:
    - `temporal_context` section with session_distribution
    - `unique_sessions` in evidence
    - `session_diversity_multiplier` applied to scores
    - Distributed pattern gets high priority
    - Clustered pattern marked as `action: skip`
- [ ] Race detector clean: `go test -race ./...`

### Documentation
- [ ] Error handling matrix complete
- [ ] Gemini prompt template documented
- [ ] Hook interaction documented

---

## Dependencies

### Go Dependencies (add to go.mod)

```
github.com/gofrs/flock v0.8.1  // File locking
```

---

## Rollout Plan

### Phase 1: Core Implementation (4 hours)
1. Create `pkg/telemetry/solved_problem.go`
2. Create `pkg/telemetry/classification.go`
3. Create `cmd/gogent-sharp-edge/resolution.go`
4. Write unit tests

### Phase 2: Integration (3 hours)
1. Modify `cmd/gogent-sharp-edge/main.go`
2. Update `gogent-archive` for cleanup/rotation
3. Update `memory-improvement/SKILL.md`
4. Integration tests

### Phase 3: Validation (2 hours)
1. Manual testing with real failures
2. Verify Gemini parsing
3. Run `/memory-improvement` end-to-end
4. Fix any issues

---

## Why This Matters

Without structured problem capture:
- Sharp edges manually curated (slow)
- Patterns undetected (same mistakes repeat)
- `/memory-improvement` only sees telemetry, not semantics

With structured problem capture:
- Problems auto-documented when resolved
- Patterns detected by Gemini across sessions
- Sharp edges recommended based on evidence
- Knowledge compounds automatically

**ROI:** First recurring problem prevented pays for implementation time.
