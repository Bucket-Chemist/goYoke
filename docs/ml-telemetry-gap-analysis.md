# ML Telemetry Pipeline - GAP Analysis

> **⚠️ DEPRECATED (2026-01-26)**
>
> This document has been **superseded** by the completion verification in tickets goYoke-110/111/112.
>
> **Final Status:** Most proposed work was already implemented at time of analysis.
>
> **What was implemented:**
> - ✅ RoutingDecision logging in `goyoke-validate` (lines 52-67)
> - ✅ AgentCollaboration logging in `goyoke-agent-endstate` (lines 70-98)
> - ✅ PostToolEvent logging in `goyoke-sharp-edge` (lines 92-97)
> - ✅ `ClassifyTask()` in `pkg/telemetry/task_classifier.go`
> - ✅ `extractAgentFromPrompt()` in `cmd/goyoke-validate/main.go`
>
> **What remains (documented limitations):**
> - ❌ Decision outcome recording (blocked by DecisionID propagation - see goYoke-111)
> - 🔮 Phase 3 ML training (requires 100+ sessions of data accumulation)
>
> **See:** `docs/ml-telemetry-completion-report.md` for final verification results.

---

**Document Version:** 1.0
**Created:** 2026-01-26
**Status:** ~~Implementation Planning~~ **DEPRECATED**
**Owner:** System Evolution Team

---

## Executive Summary

goYoke Phase 0 (deterministic enforcement) is complete. This document analyzes gaps in **Phase 2: Observability & Telemetry Layer**, which enables **Phase 3: Evolutionary Optimization** via ML-driven schema refinement.

### Strategic Vision

goYoke is not just a validation framework—it's a **self-improving agentic system**:

1. **Layer 1: Deterministic Enforcement** ✅ Complete
   - `routing-schema.json` defines rules
   - Hooks enforce programmatically
   - Blocking/validation works reliably

2. **Layer 2: Observability & Telemetry** ⚠️ 70% Complete
   - Infrastructure exists (structs, logging, export)
   - **GAP:** Systematic capture in hooks missing
   - **IMPACT:** No training corpus accumulation

3. **Layer 3: Evolutionary Optimization** 🔮 Planned
   - ML analyzes telemetry patterns
   - Proposes routing schema refinements
   - A/B tests improvements via benchmarks
   - **BLOCKED BY:** Layer 2 gaps

### Key Finding

**Infrastructure is built, wiring is incomplete.**

- ✅ `routing.PostToolEvent` struct defined
- ✅ `telemetry.LogMLToolEvent()` writes JSONL
- ✅ `goyoke-ml-export` CLI exports datasets
- ❌ **Hooks don't call logging functions systematically**

**Estimated effort to complete:** ~3.5 hours
**Unblocks:** Data-driven system evolution

---

## Current State Assessment

### What Works ✅

#### 1. Data Structures (100% Complete)

**File:** `pkg/routing/post_tool_event.go`

```go
type PostToolEvent struct {
    ToolName      string `json:"tool_name"`
    SelectedTier  string `json:"selected_tier"`
    SelectedAgent string `json:"selected_agent"`
    TaskType      string `json:"task_type"`
    TaskDomain    string `json:"task_domain"`
    InputTokens   int    `json:"input_tokens"`
    OutputTokens  int    `json:"output_tokens"`
    DurationMs    int64  `json:"duration_ms"`
    Success       bool   `json:"success"`
    CapturedAt    int64  `json:"captured_at"`
    SessionID     string `json:"session_id"`
}
```

**Status:** Ready for use, includes all ML features.

#### 2. Logging Infrastructure (100% Complete)

**File:** `pkg/telemetry/ml_logging.go`

- ✅ `LogMLToolEvent(event *PostToolEvent, projectDir string)` - Dual-write pattern
- ✅ `ReadMLToolEvents()` - Streaming JSONL reader
- ✅ `CalculateMLSessionStats(events)` - Aggregate metrics
- ✅ XDG-compliant paths (`~/.goyoke/ml-tool-events.jsonl`)
- ✅ Project-scoped writes (`.claude/memory/ml-tool-events.jsonl`)

**Status:** Production-ready with proper error handling.

#### 3. Export Infrastructure (100% Complete)

**File:** `cmd/goyoke-ml-export/main.go` (506 lines)

Commands:
- ✅ `goyoke-ml-export routing --format csv --since 7d`
- ✅ `goyoke-ml-export sequences --successful-only`
- ✅ `goyoke-ml-export collaborations --format json`
- ✅ `goyoke-ml-export training-dataset --output ./ml-data/`

**Status:** Fully functional, tested with integration tests.

#### 4. Collaboration Tracking (100% Complete)

**File:** `pkg/telemetry/collaboration.go`

```go
type AgentCollaboration struct {
    ParentAgent  string `json:"parent_agent"`
    ChildAgent   string `json:"child_agent"`
    ChildSuccess bool   `json:"child_success"`
    CapturedAt   int64  `json:"captured_at"`
    SessionID    string `json:"session_id"`
}
```

- ✅ `LogCollaboration(collab *AgentCollaboration, projectDir string)`
- ✅ `ReadCollaborationLogs()`
- ✅ Dual-write pattern (global + project-scoped)

**Status:** Ready, awaiting systematic usage.

---

### What's Missing ❌

#### GAP 1: Systematic ML Capture in goyoke-validate Hook

**Location:** `cmd/goyoke-validate/main.go`

**Current behavior:**
- Hook validates Task() calls
- Blocks invalid routing decisions
- **Does NOT capture telemetry**

**Required behavior:**
- After validation decision (allow/warn/block)
- Extract task characteristics from event
- Log `PostToolEvent` with ML features
- Write to telemetry log

**Implementation site:** After validation decision, before returning response

**Blocker impact:**
- No routing decision training data
- Cannot analyze tier selection effectiveness
- Cannot detect optimal delegation patterns

**Files affected:**
- `cmd/goyoke-validate/main.go` (add capture call)
- Possibly new: `pkg/validation/ml_features.go` (feature extraction helpers)

---

#### GAP 2: Collaboration Capture in goyoke-sharp-edge Hook

**Location:** `cmd/goyoke-sharp-edge/main.go`

**Current behavior:**
- Tracks tool counter (attention gates)
- Detects failure loops (sharp edges)
- **Does NOT capture collaborations**

**Required behavior:**
- On SubagentStop events (agent completion)
- Extract parent→child relationship from transcript
- Log `AgentCollaboration` with success/failure
- Write to collaboration log

**Implementation site:** New event handler for SubagentStop

**Blocker impact:**
- No collaboration graph data
- Cannot identify ineffective agent pairings
- Cannot optimize delegation chains

**Files affected:**
- `cmd/goyoke-sharp-edge/main.go` (add SubagentStop handler)
- Hook must handle PostToolUse AND SubagentStop events

---

#### GAP 3: Feature Extraction Utilities

**Location:** New package `pkg/telemetry/features/` or add to existing

**Current state:** Feature extraction is ad-hoc

**Required utilities:**

```go
// Extract task type from Task() prompt
func ClassifyTaskType(prompt string) string
// Returns: "file_search", "implementation", "documentation", etc.

// Extract domain from prompt
func ExtractDomain(prompt string) string
// Returns: "python", "go", "R", "markdown", "system"

// Determine selected tier from validation result
func DetermineSelectedTier(event HookEvent, decision string) string
// Returns: "haiku", "haiku_thinking", "sonnet", "opus"

// Extract agent name from Task() prompt
func ExtractAgentFromPrompt(prompt string) string
// Returns: agent ID from "AGENT: python-pro" in prompt

// Estimate token count (pre-execution)
func EstimateTokens(text string) int
// Simple heuristic: words * 1.3
```

**Blocker impact:**
- Without these, ML features will be empty/inconsistent
- Training data quality degraded
- ML models cannot learn meaningful patterns

**Files affected:**
- New: `pkg/telemetry/features.go` (or similar)
- Tests: `pkg/telemetry/features_test.go`

---

## Implementation Roadmap

### Phase 2.1: ML Capture in Hooks (Priority 1)

**Estimated effort:** 2-3 hours
**Blocking:** All ML pipeline functionality

#### Task 2.1.1: Feature Extraction Utilities

**Acceptance criteria:**
- [ ] `ClassifyTaskType(prompt)` returns task type from prompt analysis
- [ ] `ExtractDomain(prompt)` identifies language/domain
- [ ] `DetermineSelectedTier(event, decision)` maps validation result to tier
- [ ] `ExtractAgentFromPrompt(prompt)` parses agent ID
- [ ] `EstimateTokens(text)` provides token count estimate
- [ ] Unit tests cover common prompt patterns
- [ ] Test coverage ≥85%

**Files to create/modify:**
- `pkg/telemetry/features.go`
- `pkg/telemetry/features_test.go`

**Example implementation:**

```go
// pkg/telemetry/features.go
package telemetry

import (
    "regexp"
    "strings"
)

var agentPattern = regexp.MustCompile(`AGENT:\s*(\S+)`)
var taskTypePatterns = map[string]*regexp.Regexp{
    "file_search":    regexp.MustCompile(`(?i)(find|search|grep|glob|locate)`),
    "documentation":  regexp.MustCompile(`(?i)(document|readme|guide|docs)`),
    "implementation": regexp.MustCompile(`(?i)(implement|write|add|create|refactor)`),
    "review":         regexp.MustCompile(`(?i)(review|check|validate|audit)`),
    "planning":       regexp.MustCompile(`(?i)(plan|design|architect|analyze)`),
}

func ClassifyTaskType(prompt string) string {
    for taskType, pattern := range taskTypePatterns {
        if pattern.MatchString(prompt) {
            return taskType
        }
    }
    return "unknown"
}

func ExtractAgentFromPrompt(prompt string) string {
    matches := agentPattern.FindStringSubmatch(prompt)
    if len(matches) > 1 {
        return matches[1]
    }
    return "unknown"
}

func DetermineSelectedTier(model string, decision string) string {
    if decision == "block" {
        return "blocked"
    }
    return model // "haiku", "sonnet", "opus"
}

func ExtimateTokens(text string) int {
    words := len(strings.Fields(text))
    return int(float64(words) * 1.3) // Rough approximation
}
```

---

#### Task 2.1.2: ML Capture in goyoke-validate

**Acceptance criteria:**
- [ ] After Task validation decision, telemetry logged
- [ ] `PostToolEvent` includes all ML features
- [ ] Both allow and block decisions are captured
- [ ] Logs written to global + project paths
- [ ] Error handling doesn't break hook execution
- [ ] Integration test `TestMLTelemetry_RoutingDecisionCapture` passes

**Files to modify:**
- `cmd/goyoke-validate/main.go`

**Implementation location:**

```go
// In cmd/goyoke-validate/main.go
func handleTaskValidation(event HookEvent) Response {
    // ... existing validation logic ...

    decision := validateTask(event)

    // NEW: Capture ML telemetry
    captureMLTelemetry(event, decision)

    return Response{
        Decision: decision.Decision,
        Message:  decision.Message,
    }
}

func captureMLTelemetry(event HookEvent, decision ValidationDecision) {
    projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
    if projectDir == "" {
        projectDir, _ = os.Getwd()
    }

    prompt := event.Params["prompt"].(string)
    model := event.Params["model"].(string)

    mlEvent := &routing.PostToolEvent{
        ToolName:      event.ToolName,
        SelectedTier:  features.DetermineSelectedTier(model, decision.Decision),
        SelectedAgent: features.ExtractAgentFromPrompt(prompt),
        TaskType:      features.ClassifyTaskType(prompt),
        TaskDomain:    features.ExtractDomain(prompt),
        InputTokens:   features.EstimateTokens(prompt),
        OutputTokens:  0, // Not known pre-execution
        DurationMs:    0, // Validation duration negligible
        Success:       decision.Decision == "allow",
        CapturedAt:    time.Now().Unix(),
        SessionID:     event.SessionID,
    }

    // Log but don't fail validation if logging fails
    if err := telemetry.LogMLToolEvent(mlEvent, projectDir); err != nil {
        // Log error but continue
        fmt.Fprintf(os.Stderr, "[validate] ML logging failed: %v\n", err)
    }
}
```

---

#### Task 2.1.3: Collaboration Capture in goyoke-sharp-edge

**Acceptance criteria:**
- [ ] SubagentStop events trigger collaboration logging
- [ ] Parent→child relationship extracted from transcript
- [ ] Success/failure captured from event
- [ ] Logs written to collaboration JSONL
- [ ] Integration test `TestMLTelemetry_CollaborationTracking` passes

**Files to modify:**
- `cmd/goyoke-sharp-edge/main.go`

**Implementation:**

```go
// In cmd/goyoke-sharp-edge/main.go
func main() {
    // ... existing setup ...

    // Check event type from STDIN
    var hookEvent HookEvent
    json.NewDecoder(os.Stdin).Decode(&hookEvent)

    switch hookEvent.Event {
    case "PostToolUse":
        handlePostToolUse(hookEvent)
    case "SubagentStop":
        handleSubagentStop(hookEvent) // NEW
    default:
        // Ignore other events
    }
}

func handleSubagentStop(event HookEvent) {
    projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
    if projectDir == "" {
        projectDir, _ = os.Getwd()
    }

    // Extract parent agent from transcript/context
    parentAgent := extractParentAgent(event.Transcript)

    collab := &telemetry.AgentCollaboration{
        ParentAgent:  parentAgent,
        ChildAgent:   event.AgentName,
        ChildSuccess: event.Error == "",
        CapturedAt:   time.Now().Unix(),
        SessionID:    event.SessionID,
    }

    // Log collaboration
    if err := telemetry.LogCollaboration(collab, projectDir); err != nil {
        fmt.Fprintf(os.Stderr, "[sharp-edge] Collaboration logging failed: %v\n", err)
    }
}

func extractParentAgent(transcript string) string {
    // Parse transcript for parent agent context
    // For now, return "orchestrator" as default
    // TODO: Improve with transcript parsing
    return "orchestrator"
}
```

---

### Phase 2.2: Integration Testing (Priority 2)

**Estimated effort:** 30 minutes
**Depends on:** Phase 2.1 complete

#### Task 2.2.1: Run Integration Tests

**Commands:**

```bash
# Run ML telemetry integration tests
go test ./test/integration -v -run TestMLTelemetry

# Expected output: 7/7 tests pass
```

**Tests that should now pass:**
- ✅ `TestMLTelemetry_RoutingDecisionCapture` (was failing)
- ✅ `TestMLTelemetry_DecisionUpdates` (already passing)
- ✅ `TestMLTelemetry_ConcurrentWrites` (was failing)
- ✅ `TestMLTelemetry_CollaborationTracking` (was failing)
- ✅ `TestMLTelemetry_ExportReconciliation` (already passing)
- ✅ `TestMLTelemetry_RaceConditionDetection` (already passing)
- ✅ `TestMLTelemetry_SequenceIntegrity` (already passing)

**Acceptance criteria:**
- [ ] All 7 tests pass
- [ ] No race conditions detected
- [ ] Test coverage ≥80%

---

### Phase 2.3: Documentation & Deployment (Priority 3)

**Estimated effort:** 1 hour

#### Task 2.3.1: Update Documentation

**Files to update:**

1. **IMPLEMENTATION-MISSING.md**
   - Mark items #1 and #2 as COMPLETE
   - Remove item #3 (goyoke-ml-export already exists)
   - Or delete entire file if all items complete

2. **README.md**
   - Update "Implementation Status" section
   - Add "ML Telemetry Layer" to feature list
   - Document ML pipeline in architecture section

3. **docs/systems-architecture-overview.md**
   - Add ML telemetry flow diagram
   - Document capture points (validate, sharp-edge)
   - Explain data flow: hooks → JSONL → export → training

4. **New: docs/ml-pipeline-guide.md**
   - How to enable/disable ML capture
   - Data collection best practices
   - Privacy considerations (session data contains code)
   - Export commands and dataset formats

**Acceptance criteria:**
- [ ] All documentation reflects completed ML pipeline
- [ ] Architecture diagrams updated
- [ ] User guide for ML features written

---

#### Task 2.3.2: Environment Variables

**New configuration options:**

```bash
# Enable/disable ML capture (default: enabled)
export GOYOKE_ML_CAPTURE_ENABLED=true

# ML log retention (days)
export GOYOKE_ML_RETENTION_DAYS=90

# ML export path override
export GOYOKE_ML_EXPORT_PATH=/custom/path/ml-data
```

**Files to modify:**
- `pkg/config/config.go` (add ML-specific config)
- Hook CLIs check `GOYOKE_ML_CAPTURE_ENABLED` before logging

---

## Phase 3 Preview: Evolutionary Optimization

**Status:** 🔮 Future work (not blocking Phase 2)
**Estimated timeline:** 4-8 weeks after data collection begins

### Phase 3.1: Data Collection Period

**Duration:** 2-4 weeks
**Goal:** Accumulate diverse session corpus

**Metrics to track:**
- Total sessions captured: Target ≥100
- Routing decisions: Target ≥1000
- Collaboration edges: Target ≥50
- Task types covered: ≥5 categories

### Phase 3.2: Feature Engineering

**Activities:**
- Extract predictive features from corpus
- Label success/failure outcomes
- Calculate cost metrics per decision
- Build training/validation splits

**Deliverables:**
- Cleaned CSV datasets
- Feature importance analysis
- Correlation matrices

### Phase 3.3: Model Training

**ML objectives:**

1. **Tier Prediction Model**
   - Input: Task characteristics (type, domain, file count, LOC)
   - Output: Optimal tier (haiku, haiku_thinking, sonnet)
   - Metric: Cost reduction while maintaining ≥95% success rate

2. **Risk Prediction Model**
   - Input: File patterns, complexity, historical failures
   - Output: Debugging loop probability
   - Metric: Precision/recall on sharp edge prediction

3. **Collaboration Optimizer**
   - Input: Agent pairing, task type
   - Output: Success probability, expected cost
   - Metric: ROI improvement on delegation chains

### Phase 3.4: Schema Evolution

**Process:**

1. ML proposes routing schema updates
2. Changes reviewed by human (safety check)
3. A/B test via benchmark suite
4. If benchmarks pass: commit to routing-schema.json
5. Document rationale in decision log

**Example evolution:**

```json
{
  "version": "2.3.0",
  "changelog": {
    "2.3.0": {
      "date": "2026-02-15",
      "changes": [
        {
          "field": "agents.tech-docs-writer.delegation_ceiling",
          "old": "sonnet",
          "new": "haiku_thinking",
          "rationale": "ML analysis: 94% success rate with haiku_thinking, 37% cost reduction",
          "evidence": "ml-data/2026-02-analysis.csv",
          "benchmark_delta": "+12% speed, -37% cost, -2% quality (acceptable)"
        }
      ]
    }
  }
}
```

---

## Success Criteria

### Phase 2 Complete When:

- [ ] All 3 gaps closed (feature extraction, validate capture, sharp-edge capture)
- [ ] All 7 integration tests pass
- [ ] Test coverage ≥85% for new code
- [ ] Documentation updated
- [ ] Environment variables documented
- [ ] No performance degradation in hooks (capture <5ms overhead)

### Phase 3 Readiness Indicators:

- [ ] ≥100 sessions with ML telemetry
- [ ] ≥1000 routing decisions captured
- [ ] ≥5 task types represented
- [ ] Data quality validated (no missing fields >5%)
- [ ] Export pipeline works reliably

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Hook performance degradation | Low | Medium | Benchmark hooks pre/post capture |
| JSONL file corruption | Low | High | Atomic writes, validation on read |
| Feature extraction inaccuracy | Medium | Medium | Manual review of sample data, iterative improvement |
| Privacy concerns (code in logs) | Medium | High | Document clearly, allow opt-out via env var |

### Organizational Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Scope creep to full ML platform | Medium | High | Keep Phase 2 focused: capture only, defer training |
| Data volume growth | Medium | Medium | Implement retention policy (90 days default) |
| Maintenance burden | Low | Medium | Keep ML code separate from core validation logic |

---

## Cost-Benefit Analysis

### Implementation Cost

- Engineering time: ~4 hours (Phase 2.1-2.3)
- Testing time: ~1 hour
- Documentation: ~1 hour
- **Total: ~6 hours**

### Operational Cost

- Disk space: ~1-5MB per 100 sessions (JSONL is compact)
- Hook overhead: <5ms per validation (negligible)
- Maintenance: ~1 hour/month (log rotation, monitoring)

### Strategic Benefit

**Quantifiable:**
- Projected cost reduction: 20-40% via tier optimization
- Projected loop prevention: 30-50% fewer debugging failures
- Data-driven decision making vs. guesswork

**Qualitative:**
- Evidence-based system evolution
- Continuous improvement feedback loop
- Benchmark-validated schema updates
- Competitive advantage in agentic systems

**ROI:** High (6 hours investment, ongoing cost savings)

---

## Dependencies

### Internal Dependencies

- ✅ `routing.PostToolEvent` struct (exists)
- ✅ `telemetry.LogMLToolEvent()` (exists)
- ✅ `telemetry.AgentCollaboration` struct (exists)
- ✅ `goyoke-ml-export` CLI (exists)
- ❌ Feature extraction utilities (Task 2.1.1)

### External Dependencies

- None (pure Go implementation)
- Future ML training may use Python/scikit-learn (Phase 3)

---

## References

### Related Documents

- `IMPLEMENTATION-MISSING.md` - Original gap identification
- `docs/systems-architecture-overview.md` - System architecture
- `test/integration/ml_telemetry_test.go` - Integration tests (609 lines)
- `pkg/telemetry/ml_logging.go` - Logging infrastructure
- `cmd/goyoke-ml-export/main.go` - Export CLI

### Code Locations

| Component | File | Lines |
|-----------|------|-------|
| PostToolEvent struct | `pkg/routing/post_tool_event.go` | 15 |
| ML logging | `pkg/telemetry/ml_logging.go` | 150 |
| Collaboration logging | `pkg/telemetry/collaboration.go` | 120 |
| ML export CLI | `cmd/goyoke-ml-export/main.go` | 506 |
| Integration tests | `test/integration/ml_telemetry_test.go` | 609 |

### External Resources

- [JSONL Format Specification](http://jsonlines.org/)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
- Claude Code Hooks Documentation (internal)

---

## Appendix A: Data Schema

### PostToolEvent Schema

```json
{
  "tool_name": "Task",
  "selected_tier": "sonnet",
  "selected_agent": "python-pro",
  "task_type": "implementation",
  "task_domain": "python",
  "input_tokens": 1250,
  "output_tokens": 3400,
  "duration_ms": 4500,
  "success": true,
  "captured_at": 1706198400,
  "session_id": "session-abc123"
}
```

### AgentCollaboration Schema

```json
{
  "parent_agent": "orchestrator",
  "child_agent": "python-pro",
  "child_success": true,
  "captured_at": 1706198400,
  "session_id": "session-abc123"
}
```

---

## Appendix B: Sample ML Questions

### Tier Optimization

**Question:** "For Python implementation tasks with <500 LOC, which tier has the best cost/success ratio?"

**Data needed:**
- Task type = "implementation"
- Task domain = "python"
- File LOC (from context)
- Selected tier
- Success outcome
- Cost

**Expected insight:** "haiku_thinking achieves 92% success at 1/5 the cost of sonnet"

### Sharp Edge Prediction

**Question:** "What file patterns predict debugging loops?"

**Data needed:**
- File paths with 3+ consecutive failures
- File characteristics (extension, LOC, complexity)
- Agent assigned
- Task type

**Expected insight:** "pkg/*/handler.go files >200 LOC have 73% loop rate with haiku"

### Collaboration Effectiveness

**Question:** "Which agent pairings waste the most tokens?"

**Data needed:**
- Parent→child invocations
- Combined token usage
- Success rate
- Re-delegation frequency

**Expected insight:** "orchestrator → python-pro → python-ux wastes 40% tokens vs. direct python-ux routing"

---

**End of GAP Analysis**

**Next Steps:**
1. Review this document with team
2. Approve Phase 2.1 implementation plan
3. Create implementation tickets
4. Begin feature extraction utilities (Task 2.1.1)

**Document Status:** Ready for review
**Last Updated:** 2026-01-26
