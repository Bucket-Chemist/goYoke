# GAP Document: mgrep Integration Specification Review

> **Version:** 1.0.0
> **Date:** 2026-02-02
> **Reviewers:** Einstein (Opus) + Staff Architect (Sonnet)
> **Document Reviewed:** tickets/mgrep/mgrep-integration-spec.md
> **Architecture Reference:** docs/ARCHITECTURE.md (routing-schema v2.5.0)

---

## Executive Summary

The mgrep-integration-spec.md has undergone dual critical review:
1. **Einstein Analysis** - Deep architectural analysis focusing on agent benefit, alignment, and gaps
2. **Staff Architect Review** - 7-layer critical review focusing on risks and contractor readiness

**Consensus Recommendation: APPROVE WITH CONDITIONS**

Both reviewers agree the specification is fundamentally sound and well-aligned with GOgent-Fortress architecture. However, implementation should be conditional on addressing 7 mandatory conditions and 12 recommended improvements.

---

## Part 1: Einstein Analysis Findings

### 1.1 Agent Benefit Assessment

#### Strongly Benefit from mgrep

| Agent | Current Pain Point | mgrep Value | Priority |
|-------|-------------------|-------------|----------|
| **codebase-search** | Regex patterns miss intent | Semantic "where is auth?" queries | P0 (Critical) |
| **haiku-scout** | Mechanical grep underestimates semantic scope | Improved routing accuracy | P0 (Critical) |
| **librarian** | External-first research ignores internal patterns | Internal-first saves API calls | P1 (High) |
| **orchestrator** | Disambiguation requires multiple grep iterations | Agentic mode provides cross-module understanding | P1 (High) |
| **review-orchestrator** | Domain detection via file patterns is brittle | Semantic classification of changes | P2 (Medium) |

#### Questionable Benefit

| Agent | Concern | Recommendation |
|-------|---------|----------------|
| **memory-archivist** | Semantic search on structured YAML frontmatter is overkill | **REMOVE from scope** |
| **architect** | Gemini already provides pattern discovery | **OPTIONAL** - lower priority |
| **Reviewer agents** | "Find similar code" is speculative value | **OPTIONAL** - P3 at best |

#### Missing from Spec

| Agent | Potential Use Case | Priority |
|-------|-------------------|----------|
| **impl-manager** | Find convention examples during implementation | Medium |
| **architect-reviewer** | Discover anti-patterns across codebase | Low |

### 1.2 Architectural Alignment

**Well-Aligned:**
- ✅ Follows gemini-slave pattern (external engine via Bash)
- ✅ Uses append-only JSONL for telemetry
- ✅ Maintains fallback strategy (grep)
- ✅ Respects tier boundaries

**Concerns Identified:**

#### Concern 1: Schema Complexity Growth
- Adds ~80 lines to `routing-schema.json`
- Schema at v2.5.0 already substantial
- **Recommendation**: Extract to `~/.claude/external-engines.json`

#### Concern 2: Agent Definition Sprawl
- Each agent gains `mgrep_integration`, `tool_selection`, `invocation`, `fallback` sections
- When mgrep V2 changes, 9 files need updates
- **Recommendation**: Create shared `~/.claude/mgrep-patterns.yaml`

#### Concern 3: Hook Detection Fragility
- Spec requires parsing arbitrary Bash commands in `gogent-sharp-edge`
- Command parsing is inherently fragile
- **Recommendation**: Use structured invocation wrapper or marker file

### 1.3 Critical Gaps

| Gap | Severity | Location | Recommendation |
|-----|----------|----------|----------------|
| Scout protocol mismatch | HIGH | Section 4.3 | Integrate mgrep INTO gather-scout-metrics.sh |
| Librarian scope creep | MEDIUM | Section 4.4 | Route internal queries to codebase-search |
| Orchestrator mode selection undefined | MEDIUM | Section 4.5 | Define criteria for discover vs agentic mode |
| Cost model unverified | HIGH | Section 1.4, 3.1 | Benchmark before commitment |
| Parallel P0 updates risk | HIGH | Phase 2 | Sequential agent updates |
| Memory-archivist misfit | LOW | Section 4.9 | Remove from scope |

---

## Part 2: Staff Architect 7-Layer Review

### 2.1 Assumptions

| Finding | Severity | Details |
|---------|----------|---------|
| **Cost model unstated** | CRITICAL | $0.001/query assumed but unvalidated against Mixedbread pricing |
| **Query translation assumed** | WARNING | No validation agents can formulate effective semantic queries |
| **Index freshness assumed** | WARNING | No handling of stale index after refactoring |
| **Privacy model unstated** | INFO | Mixedbread data retention policy not referenced |

### 2.2 Dependencies

| Finding | Severity | Details |
|---------|----------|---------|
| **Authentication flow brittleness** | CRITICAL | Hooks (headless) cannot use device login |
| **Network dependency explosion** | CRITICAL | Every codebase-search now depends on external API |
| **npm global install assumption** | WARNING | May not work on Arch Linux |
| **NDJSON compatibility** | WARNING | mgrep output format changes break integration |

### 2.3 Failure Modes

| Finding | Severity | Details |
|---------|----------|---------|
| **Partial indexing silent failure** | CRITICAL | No detection for "10% of codebase indexed" |
| **Query timeout cascade** | CRITICAL | 3x agentic mode = 180s blocked, no aggregate ceiling |
| **Fallback thrashing** | WARNING | No session-level "mgrep is down" circuit breaker |
| **Zero results ambiguity** | WARNING | Can't distinguish "not found" from error |
| **Rate limiting not addressed** | INFO | Burst workflows may hit 429 |

### 2.4 Cost-Benefit

| Finding | Severity | Details |
|---------|----------|---------|
| **Implementation effort underestimated** | WARNING | Similar work took 18 tasks; 8-10 weeks realistic |
| **Token savings unvalidated** | WARNING | 40-60% claim has no baseline data |
| **Maintenance burden not quantified** | WARNING | Expect 2-4 hours/month ongoing |

### 2.5 Testing Strategy

| Finding | Severity | Details |
|---------|----------|---------|
| **Integration test gap** | CRITICAL | No E2E workflow test |
| **Fallback testing incomplete** | WARNING | No "mgrep returns garbage" test |
| **Benchmark prompts too simple** | WARNING | No adversarial queries |

### 2.6 Architecture Smells

| Finding | Severity | Details |
|---------|----------|---------|
| **Schema complexity growth** | CRITICAL | 7 sub-sections, 4 modes; extract to engine config |
| **Agent definition sprawl** | WARNING | 9 files need update when mgrep V2 ships |
| **Bash-only invocation** | WARNING | Acceptable given Claude Code limits; document |

### 2.7 Contractor Readiness

| Finding | Severity | Details |
|---------|----------|---------|
| **Ambiguous telemetry spec** | CRITICAL | WHERE to log MgrepInvocation not specified |
| **Missing sequencing** | WARNING | No dependency graph for Phase 2 tasks |
| **Unclear scout integration** | WARNING | calculate-complexity.sh consumption of mgrep-scope.txt unclear |

---

## Part 3: Synthesized Risk Assessment

### Top 5 Risks (Prioritized)

#### Risk #1: Cost Model Mismatch (CRITICAL)
- **Source**: Einstein + Staff Architect
- **Description**: Mixedbread pricing may be per-embedding not per-query, causing 100x cost overrun
- **Likelihood**: Medium
- **Impact**: Critical ($100/month → $10,000/month)
- **Mitigations**:
  - [ ] PHASE 0: Validate Mixedbread pricing model
  - [ ] Add cost ceiling: `max_mgrep_cost_per_session: 0.05`
  - [ ] Implement session abort at threshold
  - [ ] Add weekly cost report command

#### Risk #2: Authentication Failure in Hooks (CRITICAL)
- **Source**: Staff Architect
- **Description**: Headless hooks can't use device login; mgrep unavailable in automation
- **Likelihood**: High
- **Impact**: High (all exploration degrades to grep)
- **Mitigations**:
  - [ ] Document API key as REQUIRED in Phase 1
  - [ ] Add pre-flight check in gogent-load-context
  - [ ] Create `scripts/setup-mgrep.sh` validation script

#### Risk #3: Parallel P0 Updates Cascade (HIGH)
- **Source**: Einstein
- **Description**: Updating codebase-search AND haiku-scout simultaneously risks exploration cascade failure
- **Likelihood**: Medium
- **Impact**: High (routing unreliable)
- **Mitigations**:
  - [ ] Week 2: codebase-search only
  - [ ] Week 3: haiku-scout (after codebase-search stable)

#### Risk #4: Semantic Search Quality Degradation (HIGH)
- **Source**: Staff Architect
- **Description**: mgrep returns false positives, negating token savings
- **Likelihood**: High (unvalidated)
- **Impact**: High (40-60% savings → 10-20%)
- **Mitigations**:
  - [ ] PHASE 1: Run 20 test queries with precision/recall
  - [ ] Define quality floor: precision >70% or fallback
  - [ ] Add precision field to telemetry

#### Risk #5: Memory-Archivist Integration Misfit (MEDIUM)
- **Source**: Einstein
- **Description**: Semantic search inappropriate for structured YAML frontmatter
- **Likelihood**: High (by design)
- **Impact**: Medium (wasted effort, incorrect deduplication)
- **Mitigations**:
  - [ ] Remove memory-archivist from scope entirely

---

## Part 4: Conditions for Approval

### Mandatory Conditions (BLOCKING)

| ID | Condition | Source | Phase Impact |
|----|-----------|--------|--------------|
| **C1** | Validate Mixedbread pricing model and update cost estimates | Both | Add Phase 0 |
| **C2** | Add cost ceiling enforcement to routing-schema.json with session abort | Staff Architect | Phase 1 |
| **C3** | Add API key requirement to Phase 1 with pre-flight validation | Staff Architect | Phase 1 |
| **C4** | Add end-to-end integration test | Staff Architect | Phase 2 |
| **C5** | Add partial indexing health check (`mgrep status`) | Staff Architect | Phase 1 |
| **C6** | Add session-level circuit breaker (3 failures → disable) | Both | Phase 1 |
| **C7** | Specify exact telemetry integration points | Staff Architect | Phase 2 |
| **C8** | Remove memory-archivist from scope | Einstein | Immediate |
| **C9** | Restructure Phase 2 for sequential P0 agent updates | Einstein | Phase 2 |

### Recommended Conditions (QUALITY)

| ID | Condition | Source | Phase Impact |
|----|-----------|--------|--------------|
| **R1** | Extract engine configs from routing-schema.json | Both | Phase 1 |
| **R2** | Create shared mgrep-patterns.yaml | Einstein | Phase 1 |
| **R3** | Add dependency graph to Phase 2-3 tasks | Staff Architect | Phase 2 |
| **R4** | Add adversarial benchmark queries | Staff Architect | Phase 5 |
| **R5** | Revise timeline to 8-10 weeks | Staff Architect | Planning |
| **R6** | Document maintenance overhead expectation | Staff Architect | Phase 5 |
| **R7** | Define orchestrator mode selection criteria | Einstein | Phase 3 |
| **R8** | Integrate mgrep into gather-scout-metrics.sh | Einstein | Phase 2 |
| **R9** | Route librarian internal queries to codebase-search | Einstein | Phase 3 |
| **R10** | Add structured invocation pattern for hook detection | Einstein | Phase 2 |
| **R11** | Establish baseline metrics before claiming improvements | Both | Phase 1 |
| **R12** | Add security review for credential/IP protection | Einstein | Phase 1 |

---

## Part 5: Proposed Phase Restructuring

### Phase 0: Validation (NEW - Week 0)
**Deliverables:**
1. [ ] Validate Mixedbread pricing model (C1)
2. [ ] Security/privacy review of Mixedbread ToS (R12)
3. [ ] Run 20 benchmark queries to establish baseline (R11)
4. [ ] Validate API key authentication works headlessly (C3)

**Exit Criteria:**
- Cost model validated
- Privacy acceptable
- Baseline metrics recorded
- Headless auth confirmed

### Phase 1: Foundation (Week 1)
**Deliverables:**
1. [ ] Install mgrep, configure authentication
2. [ ] Create `/mgrep` skill with full SKILL.md
3. [ ] Add mgrep engine definition to routing-schema.json WITH cost ceiling (C2)
4. [ ] Create .mgrepignore template
5. [ ] Add session-level circuit breaker (C6)
6. [ ] Add health check for partial indexing (C5)
7. [ ] Create `scripts/setup-mgrep.sh` validation script

**Validation:**
- `/mgrep "test query"` works
- Fallback triggers correctly
- Circuit breaker activates after 3 failures

### Phase 2: Critical Agents (Weeks 2-4 - EXTENDED)
**Week 2: codebase-search only**
1. [ ] Update codebase-search/agent.yaml
2. [ ] Update codebase-search/CLAUDE.md with tool selection guide
3. [ ] Add pkg/telemetry/mgrep_invocation.go (C7)
4. [ ] Add mgrep detection to gogent-sharp-edge (C7)

**Week 3: Validate, then haiku-scout**
- Validate codebase-search works before proceeding
5. [ ] Update haiku-scout/agent.yaml with semantic phase
6. [ ] Update calculate-complexity.sh to consume mgrep-scope.txt

**Week 4: Integration**
7. [ ] Add E2E integration test (C4)

**Validation:**
- codebase-search uses mgrep for intent queries
- Scout produces semantic + mechanical scope
- Telemetry captures mgrep invocations
- E2E test passes

### Phase 3: Research Agents (Week 5)
**Deliverables:**
1. [ ] Update librarian/agent.yaml (route internal to codebase-search) (R9)
2. [ ] Update orchestrator/agent.yaml with disambiguation protocol
3. [ ] Define mode selection criteria (discover vs agentic) (R7)
4. [ ] Update architect/agent.yaml with pattern discovery

**Validation:**
- Librarian routes internal queries correctly
- Orchestrator selects appropriate mgrep mode

### Phase 4: Review Pipeline (Week 6)
**Deliverables:**
1. [ ] Update review-orchestrator/agent.yaml with domain detection
2. [ ] Update reviewer agents with optional context gathering

**Validation:**
- Review-orchestrator correctly classifies backend vs frontend

### Phase 5: Telemetry & Polish (Week 7)
**Deliverables:**
1. [ ] Add mgrep-invocations and mgrep-outcomes export
2. [ ] Add mgrep-stats command
3. [ ] Update ARCHITECTURE.md with mgrep integration
4. [ ] Run benchmark comparison (validate 40-60% claim)
5. [ ] Document maintenance overhead (R6)

**Validation:**
- `gogent-ml-export mgrep-stats` produces output
- Token savings measured against baseline
- Documentation complete

---

## Part 6: Removed from Scope

### memory-archivist (C8)
**Reason**: Semantic search is inappropriate for structured YAML frontmatter queries. Grep-based exact match correctly handles tags, status, and type fields. Semantic search may incorrectly deduplicate distinct entries.

**Original spec sections to remove:**
- Section 4.9 (P3: memory-archivist)
- Line 1258 item 4 in Phase 5

---

## Appendix A: Files Requiring Changes

### New Files
| File | Purpose | Phase |
|------|---------|-------|
| `~/.claude/skills/mgrep/SKILL.md` | User-invoked skill | 1 |
| `pkg/telemetry/mgrep_invocation.go` | Telemetry types | 2 |
| `pkg/telemetry/mgrep_logging.go` | Logging functions | 2 |
| `cmd/gogent-sharp-edge/mgrep_detection.go` | Hook detection | 2 |
| `test/integration/mgrep_integration_test.go` | E2E test | 2 |
| `.mgrepignore` | Index exclusions | 1 |
| `scripts/setup-mgrep.sh` | Validation script | 1 |
| `~/.claude/mgrep-patterns.yaml` | Shared config (R2) | 1 |

### Modified Files
| File | Changes | Phase |
|------|---------|-------|
| `~/.claude/routing-schema.json` | Add mgrep engine + cost ceiling | 1 |
| `~/.claude/agents/codebase-search/codebase-search.md` | Add mgrep integration | 2 |
| `~/.claude/agents/haiku-scout/haiku-scout.md` | Add semantic phase | 2 |
| `~/.claude/agents/librarian/librarian.md` | Route internal queries | 3 |
| `~/.claude/agents/orchestrator/orchestrator.md` | Add disambiguation | 3 |
| `~/.claude/agents/architect/architect.md` | Add pattern discovery | 3 |
| `~/.claude/agents/review-orchestrator/review-orchestrator.md` | Add domain detection | 4 |
| `~/.claude/scripts/gather-scout-metrics.sh` | Integrate mgrep (R8) | 2 |
| `docs/ARCHITECTURE.md` | Add mgrep section | 5 |

---

## Appendix B: Telemetry Integration Points (C7)

### MgrepInvocation Struct
**File**: `pkg/telemetry/mgrep_invocation.go`
```go
type MgrepInvocation struct {
    InvocationID   string    `json:"invocation_id"`
    SessionID      string    `json:"session_id"`
    Timestamp      int64     `json:"timestamp"`
    InvokingAgent  string    `json:"invoking_agent"`
    InvokingTool   string    `json:"invoking_tool"`  // "Bash"
    Query          string    `json:"query"`
    Path           string    `json:"path"`
    Mode           string    `json:"mode"`  // discover, answer, agentic
    Flags          []string  `json:"flags"`
    ResultCount    int       `json:"result_count"`
    FilesReturned  []string  `json:"files_returned,omitempty"`
    DurationMs     int64     `json:"duration_ms"`
    Success        bool      `json:"success"`
    ErrorMessage   string    `json:"error_message,omitempty"`
    FallbackUsed   bool      `json:"fallback_used"`
}
```

### Detection Location
**File**: `cmd/gogent-sharp-edge/mgrep_detection.go` (NEW)
**Function**: `detectMgrepInvocation(event *routing.PostToolEvent) *telemetry.MgrepInvocation`
**Called from**: `handlePostToolUse()` in main.go after existing tool processing

### Storage Location
**File**: `${XDG_DATA_HOME}/gogent/mgrep-invocations.jsonl`
**Written by**: `telemetry.LogMgrepInvocation()`

---

## Appendix C: Circuit Breaker Specification (C6)

### State Machine
```
CLOSED (normal) ──[3 failures]──> OPEN (disabled)
                                      │
                                      │ [session ends OR 5 min timeout]
                                      v
                                   CLOSED
```

### Implementation
**File**: `pkg/telemetry/mgrep_circuit_breaker.go`
```go
type MgrepCircuitBreaker struct {
    State          string    // "closed", "open"
    FailureCount   int
    LastFailure    time.Time
    SessionID      string
}

func (cb *MgrepCircuitBreaker) RecordFailure() {
    cb.FailureCount++
    cb.LastFailure = time.Now()
    if cb.FailureCount >= 3 {
        cb.State = "open"
        log.Printf("[mgrep-circuit-breaker] OPEN: 3 failures, mgrep disabled for session")
    }
}

func (cb *MgrepCircuitBreaker) ShouldUseMgrep() bool {
    if cb.State == "open" {
        if time.Since(cb.LastFailure) > 5*time.Minute {
            cb.State = "closed"
            cb.FailureCount = 0
            return true
        }
        return false
    }
    return true
}
```

### Usage in Agents
Before invoking mgrep, agents check:
```bash
# Check circuit breaker state file
if [ -f /tmp/mgrep-circuit-open-${SESSION_ID} ]; then
    echo "[mgrep] Circuit open, using grep fallback"
    # Use grep
else
    # Use mgrep
fi
```

---

**End of GAP Document**

*Reviewed by: Einstein (Opus) + Staff Architect (Sonnet)*
*Date: 2026-02-02*
*Status: Ready for implementation planning with conditions*
