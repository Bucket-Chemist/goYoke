# Einstein GAP Analysis: GOgent-075 through GOgent-083

> **Generated:** 2026-01-24T14:32:00Z
> **Escalated By:** user (via /einstein)
> **Analysis Type:** Pre-implementation architectural review
> **Reviewers:** Einstein (primary) + Staff-Architect (orthogonal)

---

## 1. Problem Statement

### What We're Trying to Achieve

Validate that implementation tickets GOgent-075 through GOgent-083 (Advanced Enforcement) are:
1. Correctly scoped for the GOgent-Fortress hook system
2. Compatible with existing memory/handoff schemas
3. Properly integrated with the hook event architecture
4. Free of architectural anti-patterns

### Why This Analysis Was Requested

- [x] Architectural decision required
- [x] Cross-domain synthesis needed
- [ ] 3+ consecutive failures on same task
- [ ] Complexity exceeds Sonnet tier
- [ ] User explicitly requested deep analysis

**Specific Concern:** Tickets were written before GOgent-063 (SubagentStop schema research) was completed. Need to verify schema alignment.

---

## 2. Analysis Summary

### Critical Findings (P0 - BLOCKING)

| ID | Issue | Tickets Affected | Impact |
|----|-------|------------------|--------|
| GAP-001 | SubagentStop schema mismatch | GOgent-075, GOgent-079 | Hooks will fail - `agent_id` not in event |
| GAP-002 | PreToolUse content access broken | GOgent-080, GOgent-083 | Doc-theater cannot inspect content |
| GAP-003 | STDIN double-read | GOgent-083 | CLI crashes on second read |

### High Priority Findings (P1)

| ID | Issue | Tickets Affected | Impact |
|----|-------|------------------|--------|
| GAP-004 | Unnecessary package fragmentation | All tickets | Maintenance burden |
| GAP-005 | No violation logging | GOgent-077, GOgent-083 | No audit trail |
| GAP-006 | Pattern/schema misalignment | GOgent-081 | Missing 3 detection patterns |

### Medium Priority Findings (P2)

| ID | Issue | Tickets Affected | Impact |
|----|-------|------------------|--------|
| GAP-007 | Brittle regex patterns | GOgent-076 | False negatives in production |
| GAP-008 | No context-aware detection | GOgent-081 | False positives on legitimate references |
| GAP-009 | Multi-hook coordination undefined | GOgent-080-083 | Undefined behavior |

---

## 3. Detailed Gap Analysis

### GAP-001: SubagentStop Schema Mismatch

**Severity:** CRITICAL (P0)
**Tickets:** GOgent-075, GOgent-079

#### Current Design (WRONG)

```go
// GOgent-075 assumes this schema
type OrchestratorStopEvent struct {
    Type          string `json:"type"`           // "stop"
    HookEventName string `json:"hook_event_name"` // "SubagentStop"
    AgentID       string `json:"agent_id"`       // ← NOT AVAILABLE
    AgentModel    string `json:"agent_model"`
    ExitCode      int    `json:"exit_code"`
    TranscriptPath string `json:"transcript_path"`
    Duration      int    `json:"duration_ms"`
    OutputTokens  int    `json:"output_tokens"`
}
```

#### Actual Schema (from GOgent-063 research)

```go
// Actual Claude Code SubagentStop event
type SubagentStopEvent struct {
    HookEventName   string `json:"hook_event_name"` // "SubagentStop"
    SessionID       string `json:"session_id"`
    TranscriptPath  string `json:"transcript_path"`
    StopHookActive  bool   `json:"stop_hook_active"`
    // NO agent_id, agent_model, exit_code, duration, output_tokens
}
```

#### Impact

- `event.IsOrchestratorType()` will always fail (field is empty/missing)
- Hook silently passes through ALL SubagentStop events
- Orchestrator-guard provides zero value

#### Required Fix

```go
// 1. Use correct schema
type OrchestratorStopEvent struct {
    HookEventName   string `json:"hook_event_name"`
    SessionID       string `json:"session_id"`
    TranscriptPath  string `json:"transcript_path"`
    StopHookActive  bool   `json:"stop_hook_active"`
}

// 2. Extract agent_id from transcript
func (e *OrchestratorStopEvent) ExtractAgentID() (string, error) {
    // Parse transcript file header for agent metadata
    metadata, err := ParseTranscriptMetadata(e.TranscriptPath)
    if err != nil {
        return "", err
    }
    return metadata.AgentID, nil
}

// 3. Check after extraction
func IsOrchestratorType(agentID string) bool {
    return agentID == "orchestrator" || agentID == "architect"
}
```

---

### GAP-002: PreToolUse Content Access Broken

**Severity:** CRITICAL (P0)
**Tickets:** GOgent-080, GOgent-083

#### Current Design (WRONG)

```go
// GOgent-080 assumes file_path at root level
type PreToolUseEvent struct {
    Type          string `json:"type"`
    HookEventName string `json:"hook_event_name"`
    ToolName      string `json:"tool_name"`
    FilePath      string `json:"file_path"`      // ← NOT AT ROOT
    SessionID     string `json:"session_id"`
}
```

#### Actual Schema (from pkg/routing/events.go)

```go
// Existing ToolEvent in codebase
type ToolEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"` // ← FILE PATH IS HERE
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`
}

// tool_input structure for Write tool:
// {"file_path": "/path/to/file", "content": "file contents"}

// tool_input structure for Edit tool:
// {"file_path": "/path/to/file", "old_string": "...", "new_string": "..."}
```

#### Impact

- `event.FilePath` is always empty
- `IsClaudeMDFile()` always returns false
- Doc-theater hook never activates

#### Required Fix

```go
// Use existing ToolEvent, add helper functions
func ExtractFilePath(event *ToolEvent) string {
    if path, ok := event.ToolInput["file_path"].(string); ok {
        return path
    }
    return ""
}

func ExtractContent(event *ToolEvent) string {
    // For Write tool
    if content, ok := event.ToolInput["content"].(string); ok {
        return content
    }
    // For Edit tool - scan new_string
    if newString, ok := event.ToolInput["new_string"].(string); ok {
        return newString
    }
    return ""
}

func IsClaudeMDFile(event *ToolEvent) bool {
    filePath := ExtractFilePath(event)
    if filePath == "" {
        return false
    }
    filename := filepath.Base(filePath)
    return filename == "CLAUDE.md" ||
           (strings.HasPrefix(filename, "CLAUDE.") && strings.HasSuffix(filename, ".md"))
}
```

---

### GAP-003: STDIN Double-Read

**Severity:** CRITICAL (P0)
**Tickets:** GOgent-083

#### Current Design (BROKEN)

```go
func main() {
    // First read: Parse event (consumes all STDIN)
    event, err := enforcement.ParsePreToolUseEvent(os.Stdin, DEFAULT_TIMEOUT)

    // ... later ...

    // Second read: Try to get content (STDIN is empty!)
    content := os.Getenv("TOOL_INPUT_CONTENT")
    if content == "" {
        data, err := io.ReadAll(os.Stdin)  // ← Returns empty, STDIN exhausted
        if err == nil && len(data) > 0 {
            content = string(data)
        }
    }
    // content is ALWAYS empty
}
```

#### Impact

- Content is never obtained
- Pattern detection runs on empty string
- Always returns "allow" (no patterns in empty string)

#### Required Fix

```go
func main() {
    // Single read: Parse full ToolEvent including tool_input
    event, err := routing.ParseToolEvent(os.Stdin, DEFAULT_TIMEOUT)
    if err != nil {
        outputError("Failed to parse event")
        os.Exit(1)
    }

    // Extract content from tool_input (no second read needed)
    content := routing.ExtractContent(event)

    if content == "" || !routing.IsWriteOperation(event) || !routing.IsClaudeMDFile(event) {
        outputAllow()
        os.Exit(0)
    }

    // Now we have content to scan
    pd := routing.NewPatternDetector()
    results := pd.Detect(content)
    // ...
}
```

---

### GAP-004: Unnecessary Package Fragmentation

**Severity:** HIGH (P1)
**Tickets:** All

#### Current Design

Tickets propose creating new `pkg/enforcement/` package with:
- `orchestrator_events.go`
- `transcript_analyzer.go`
- `blocking_response.go`
- `doc_events.go`
- `pattern_detector.go`

#### Existing Package Structure

```
pkg/
├── routing/
│   ├── events.go          ← Already has ToolEvent, event parsing
│   ├── transcript.go      ← Already exists for transcript analysis
│   ├── response.go        ← Already has response types
│   ├── validator.go       ← Has ValidationOrchestrator
│   └── ...
├── session/
├── memory/
├── telemetry/
└── config/
```

#### Impact

- Code fragmentation - related concepts spread across packages
- Import complexity increases
- Future maintainers confused about boundaries

#### Recommended Consolidation

| Proposed File | Recommended Location | Rationale |
|---------------|----------------------|-----------|
| `enforcement/orchestrator_events.go` | `routing/events.go` | Event types belong together |
| `enforcement/transcript_analyzer.go` | `routing/transcript.go` | **File already exists** |
| `enforcement/blocking_response.go` | `routing/response.go` | Response types belong together |
| `enforcement/doc_events.go` | DELETE | Use existing `ToolEvent` |
| `enforcement/pattern_detector.go` | `routing/doc_theater.go` | New file, validation concern |

---

### GAP-005: No Violation Logging

**Severity:** HIGH (P1)
**Tickets:** GOgent-077, GOgent-083

#### Current Design

Both hooks emit responses but do not log to `routing-violations.jsonl`:

```go
// GOgent-077 - blocking response generation
response.Decision = "block"
response.Reason = "Background tasks not collected"
// No logging!

// GOgent-083 - warning response generation
return `{"decision": "warn", ...}`
// No logging!
```

#### Impact

- No audit trail for blocked orchestrator completions
- No tracking of doc-theater warnings
- Session handoff missing these events
- Cannot measure hook effectiveness

#### Required Fix

```go
// In GOgent-077 (orchestrator-guard)
if response.Decision == "block" {
    routing.LogViolation(routing.Violation{
        Type:      "orchestrator_completion_blocked",
        Timestamp: time.Now().Unix(),
        Agent:     agentID,
        Reason:    response.Reason,
        Context:   analyzer.GetSummary(),
    })
}

// In GOgent-083 (doc-theater)
if len(results) > 0 {
    routing.LogViolation(routing.Violation{
        Type:      "documentation_theater_detected",
        Timestamp: time.Now().Unix(),
        FilePath:  filePath,
        Patterns:  extractPatternNames(results),
        Severity:  maxSeverity(results),
    })
}
```

---

### GAP-006: Pattern/Schema Misalignment

**Severity:** HIGH (P1)
**Tickets:** GOgent-081

#### Current Design

GOgent-081 hardcodes these patterns:

```go
patterns: []EnforcementPattern{
    {`(?i)\bMUST\s+NOT\b`, "critical"},
    {`(?i)\bBLOCKED\b.*\(.*\)`, "critical"},
    {`(?i)\bNEVER\s+use\b`, "critical"},
    {`(?i)\bFORBIDDEN\b`, "warning"},
    {`(?i)\bYOU\s+CANNOT\b`, "warning"},
}
```

#### routing-schema.json Defines

```json
"detection_patterns": [
    "MUST NOT",
    "NEVER use",
    "is BLOCKED",
    "SHALL NOT",        // ← MISSING from implementation
    "ALWAYS .* instead", // ← MISSING from implementation
    "FORBIDDEN",
    "PROHIBITED"        // ← MISSING from implementation
]
```

#### Impact

- 3 patterns from schema not implemented
- Hardcoded patterns require code changes to update
- Source of truth fragmented

#### Required Fix

```go
// Load patterns from routing-schema.json
func NewPatternDetector(schema *routing.Schema) *PatternDetector {
    patterns := []EnforcementPattern{}
    for _, p := range schema.MetaRules.DocumentationTheater.DetectionPatterns {
        patterns = append(patterns, EnforcementPattern{
            Pattern:  convertToRegex(p),
            Severity: determineSeverity(p),
        })
    }
    return &PatternDetector{patterns: patterns}
}

func convertToRegex(pattern string) string {
    // "MUST NOT" → `(?i)\bMUST\s+NOT\b`
    // "SHALL NOT" → `(?i)\bSHALL\s+NOT\b`
    // "ALWAYS .* instead" → `(?i)\bALWAYS\s+.*\s+instead\b`
    // etc.
}
```

---

### GAP-007: Brittle Regex Patterns

**Severity:** MEDIUM (P2)
**Tickets:** GOgent-076

#### Current Design

```go
spawnPattern := regexp.MustCompile(`(?i)run_in_background[:\s=]+true|spawn.*task|background.*task`)
taskIdPattern := regexp.MustCompile(`"task_id"\s*:\s*"([^"]+)"`)
collectPattern := regexp.MustCompile(`TaskOutput.*task_id|collecting.*task|await.*task`)
```

#### Failure Cases

1. **JSON without whitespace:**
   ```json
   {"run_in_background":true,"task_id":"bg-1"}
   ```
   `taskIdPattern` won't match (expects space after colon)

2. **Multiline JSON:**
   ```json
   TaskOutput({
     task_id: "bg-1",
     block: true
   })
   ```
   `collectPattern` requires same line

3. **False positives:**
   ```
   "I will spawn a new task for handling this"
   ```
   `spawnPattern` matches prose description

#### Recommended Fix

```go
// Use JSON parsing for structured data
func (ta *TranscriptAnalyzer) analyzeStructuredLine(line string) (*TaskInfo, error) {
    // Try to extract JSON from line
    jsonStart := strings.Index(line, "{")
    jsonEnd := strings.LastIndex(line, "}")
    if jsonStart >= 0 && jsonEnd > jsonStart {
        jsonStr := line[jsonStart : jsonEnd+1]
        var info struct {
            RunInBackground bool   `json:"run_in_background"`
            TaskID          string `json:"task_id"`
        }
        if err := json.Unmarshal([]byte(jsonStr), &info); err == nil {
            if info.RunInBackground {
                return &TaskInfo{Spawned: true, TaskID: info.TaskID}, nil
            }
        }
    }
    // Fall back to regex for unstructured text
    return ta.analyzeWithRegex(line)
}
```

---

### GAP-008: No Context-Aware Detection

**Severity:** MEDIUM (P2)
**Tickets:** GOgent-081

#### Current Design

Pattern detector flags ALL matches regardless of context:

```go
func (pd *PatternDetector) Detect(content string) []DetectionResult {
    for _, ep := range pd.patterns {
        matches := regex.FindAllStringIndex(content, -1)
        if len(matches) > 0 {
            results = append(results, DetectionResult{...})  // Always adds
        }
    }
}
```

#### Legitimate Uses That Would Be Flagged

```markdown
## Enforcement Architecture

Task(opus) invocation is BLOCKED by validate-routing.sh (see line 87).
```

This is **legitimate documentation** referencing enforcement, not theater.

#### Recommended Fix

```go
func (pd *PatternDetector) DetectWithContext(content string) []DetectionResult {
    results := pd.Detect(content)

    for i := range results {
        // Get surrounding context (50 chars before/after)
        context := getContext(content, results[i].Position, 50)

        // Check for enforcement references
        if hasEnforcementReference(context) {
            results[i].Severity = "info"  // Downgrade
            results[i].Note = "Legitimate enforcement reference detected"
        }
    }
    return results
}

func hasEnforcementReference(context string) bool {
    references := []string{
        "hook", "validate-routing", "routing-schema",
        "enforced by", "see line", "see §",
    }
    lower := strings.ToLower(context)
    for _, ref := range references {
        if strings.Contains(lower, ref) {
            return true
        }
    }
    return false
}
```

---

### GAP-009: Multi-Hook Coordination Undefined

**Severity:** MEDIUM (P2)
**Tickets:** GOgent-080, GOgent-081, GOgent-082, GOgent-083

#### Current Architecture

Two hooks respond to PreToolUse events:
1. `gogent-validate` (existing) - validates Task() calls
2. `gogent-doc-theater` (proposed) - detects documentation theater

#### Undefined Behavior

- Which hook runs first?
- What if both want to block/warn?
- How are responses merged?
- Does Claude Code support multiple hooks per event?

#### Recommended Documentation

Add to systems-architecture-overview.md:

```markdown
## Multi-Hook Execution Order

When multiple hooks are registered for the same event:

| Event | Hook Order | Merge Strategy |
|-------|------------|----------------|
| PreToolUse | 1. gogent-validate, 2. gogent-doc-theater | First block wins |
| PostToolUse | 1. gogent-sharp-edge | N/A |
| SubagentStop | 1. gogent-orchestrator-guard | N/A |

### Response Merging Rules

1. If any hook returns `"decision": "block"`, the action is blocked
2. Multiple warnings are concatenated in `additionalContext`
3. Allow decisions only propagate if no block/warn present
```

**Alternative:** Integrate doc-theater detection INTO `gogent-validate` as a 5th validation check, avoiding multi-hook complexity.

---

## 4. Constraints

- Must maintain backward compatibility with existing handoff schema (v1.2)
- Must not break existing `gogent-validate` PreToolUse handling
- Must work with actual Claude Code event schemas (not speculated ones)
- Must follow existing error message format: `[component] What. Why. How.`
- Must respect XDG paths for configuration and storage

---

## 5. Remediation Plan

### Phase 1: Critical Fixes (BLOCKING - 4h)

| Task | Ticket | Estimate | Owner |
|------|--------|----------|-------|
| Rewrite GOgent-075 with correct schema | GOgent-075 | 1.5h | TBD |
| Add transcript metadata extraction | GOgent-075 | 0.5h | TBD |
| Fix GOgent-080 to use existing ToolEvent | GOgent-080 | 0.5h | TBD |
| Fix GOgent-083 content extraction | GOgent-083 | 1h | TBD |
| Remove STDIN double-read | GOgent-083 | 0.5h | TBD |

### Phase 2: High Priority Fixes (5h)

| Task | Ticket | Estimate | Owner |
|------|--------|----------|-------|
| Consolidate into pkg/routing | All | 2h | TBD |
| Add violation logging | GOgent-077, 083 | 1h | TBD |
| Load patterns from routing-schema.json | GOgent-081 | 1h | TBD |
| Add missing patterns | GOgent-081 | 0.5h | TBD |
| Update tests for schema changes | GOgent-078, 082 | 0.5h | TBD |

### Phase 3: Medium Priority (3h)

| Task | Ticket | Estimate | Owner |
|------|--------|----------|-------|
| Improve regex with JSON parsing | GOgent-076 | 1.5h | TBD |
| Add context-aware detection | GOgent-081 | 1h | TBD |
| Document multi-hook coordination | N/A | 0.5h | TBD |

### Total Remediation Estimate

- **Original estimate:** 13.5 hours
- **Remediation overhead:** +5 hours
- **Revised estimate:** 18.5 hours

---

## 6. Ticket-by-Ticket Remediation Guide

### GOgent-075: SubagentStop Event Parsing

**Status:** REQUIRES REWRITE

**Changes Required:**
1. Replace speculated schema with actual Claude Code schema
2. Remove `AgentID`, `AgentModel`, `ExitCode`, `Duration`, `OutputTokens` fields
3. Add `SessionID`, `StopHookActive` fields
4. Add `ExtractAgentID()` method that parses transcript
5. Update `IsOrchestratorType()` to be standalone function
6. Update all tests

**New Acceptance Criteria:**
- [ ] Uses actual SubagentStop schema from Claude Code
- [ ] Extracts agent_id from transcript metadata
- [ ] `IsOrchestratorType(agentID)` works correctly
- [ ] Tests use real schema, not speculated one

---

### GOgent-076: Transcript Analysis

**Status:** NEEDS ENHANCEMENT

**Changes Required:**
1. Move to `pkg/routing/transcript.go` (extend existing)
2. Add JSON parsing alongside regex
3. Add multiline buffer for JSON objects
4. Add more test cases with real transcript formats

**New Acceptance Criteria:**
- [ ] Handles JSON without whitespace
- [ ] Handles multiline JSON objects
- [ ] Avoids false positives on prose descriptions
- [ ] Tests cover real transcript variations

---

### GOgent-077: Blocking Response

**Status:** NEEDS ENHANCEMENT

**Changes Required:**
1. Add violation logging when blocking
2. Move to `pkg/routing/response.go`

**New Acceptance Criteria:**
- [ ] Logs to routing-violations.jsonl when blocking
- [ ] Violation includes agent_id, reason, summary

---

### GOgent-078: Integration Tests

**Status:** NEEDS UPDATE

**Changes Required:**
1. Update tests to use corrected schemas
2. Add tests for transcript metadata extraction

---

### GOgent-079: CLI Build

**Status:** NEEDS UPDATE

**Changes Required:**
1. Update to use corrected event parsing
2. Add transcript metadata extraction step before IsOrchestratorType check

---

### GOgent-080: PreToolUse Event Parsing

**Status:** REQUIRES SIMPLIFICATION

**Changes Required:**
1. DELETE proposed `PreToolUseEvent` struct
2. Create helper functions for existing `ToolEvent`
3. Move to `pkg/routing/doc_theater.go`

**Reduced Scope:**
- Original: 1.5h
- Revised: 0.5h (just helper functions)

---

### GOgent-081: Pattern Detection

**Status:** NEEDS ENHANCEMENT

**Changes Required:**
1. Load patterns from routing-schema.json
2. Add missing patterns (SHALL NOT, ALWAYS instead, PROHIBITED)
3. Add context-aware detection
4. Move to `pkg/routing/doc_theater.go`

---

### GOgent-082: Integration Tests

**Status:** NEEDS UPDATE

**Changes Required:**
1. Update tests to use existing ToolEvent
2. Add tests for context-aware detection
3. Add tests for schema-loaded patterns

---

### GOgent-083: CLI Build

**Status:** REQUIRES REWRITE

**Changes Required:**
1. Remove STDIN double-read
2. Extract content from tool_input
3. Use existing ToolEvent parsing
4. Add violation logging when warning

---

## 7. Verification Checklist

Before implementing any ticket, verify:

- [ ] Schema matches actual Claude Code event (not speculated)
- [ ] Imports from existing pkg/routing where applicable
- [ ] No duplicate struct definitions
- [ ] Violation logging included
- [ ] Tests use real event formats
- [ ] Error messages follow `[component] What. Why. How.` format

---

## Metadata

```yaml
analysis_id: GAP-ADV-ENF-2026-01-24
complexity_score: high
estimated_remediation: 5h
tickets_affected: 9
critical_issues: 3
high_issues: 3
medium_issues: 3
files_referenced: 15
created_at: 2026-01-24T14:32:00Z
reviewers:
  - einstein (primary)
  - staff-architect (orthogonal)
convergence: high (both identified same critical issues)
```

---

## Appendix A: Correct Schema Reference

### SubagentStop Event (Actual)

```json
{
  "hook_event_name": "SubagentStop",
  "session_id": "sess-abc123",
  "transcript_path": "/tmp/claude-transcript-xyz.md",
  "stop_hook_active": true
}
```

### PreToolUse Event (Actual - ToolEvent)

```json
{
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/home/user/.claude/CLAUDE.md",
    "old_string": "existing content",
    "new_string": "You MUST NOT do this"
  },
  "session_id": "sess-abc123",
  "hook_event_name": "PreToolUse",
  "captured_at": 1706108400
}
```

### Write Tool Input

```json
{
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/home/user/.claude/CLAUDE.md",
    "content": "Full file content here including BLOCKED patterns"
  },
  "session_id": "sess-abc123",
  "hook_event_name": "PreToolUse",
  "captured_at": 1706108400
}
```

---

## Appendix B: Package Consolidation Map

```
BEFORE (proposed):                    AFTER (recommended):

pkg/enforcement/                      pkg/routing/
├── orchestrator_events.go      →     ├── events.go (extend)
├── transcript_analyzer.go      →     ├── transcript.go (extend)
├── blocking_response.go        →     ├── response.go (extend)
├── doc_events.go               →     │   (DELETE - use ToolEvent)
├── pattern_detector.go         →     └── doc_theater.go (new)
└── *_test.go                   →     └── *_test.go (corresponding)
```

---

*GAP Analysis Complete. This document should be reviewed before implementing any ticket in the GOgent-075 to GOgent-083 range.*
