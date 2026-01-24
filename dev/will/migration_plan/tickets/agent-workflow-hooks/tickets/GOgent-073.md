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

**Implementation Note - Migration Case**:

Add v1.2→v1.3 migration case to `migrateHandoff()`:

```go
// In migrateHandoff() - add case for v1.2 to v1.3
case "1.2":
    var handoff Handoff
    if err := json.Unmarshal(data, &handoff); err != nil {
        return nil, fmt.Errorf("[handoff] Failed to parse v1.2 handoff: %w", err)
    }
    if handoff.Artifacts.AgentEndstates == nil {
        handoff.Artifacts.AgentEndstates = []EndstateLog{}
    }
    if handoff.Artifacts.AutoFlushArchives == nil {
        handoff.Artifacts.AutoFlushArchives = []string{}
    }
    handoff.SchemaVersion = HandoffSchemaVersion
    return &handoff, nil
```

**Acceptance Criteria**:
- [x] HandoffArtifacts extended with AgentEndstates field (omitempty)
- [x] EndstateLog struct defined following research findings
- [x] loadEndstates() loads JSONL file
- [x] Backward compatible (all new fields omitempty)
- [x] migrateHandoff() has case "1.2" that initializes new slices
- [x] HandoffSchemaVersion bumped to "1.3"
- [x] Test coverage >90% for new code
- [x] All tests pass including race detector

---
