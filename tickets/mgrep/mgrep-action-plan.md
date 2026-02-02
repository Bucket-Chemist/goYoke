# mgrep Integration Action Plan

> **Version:** 1.0.0
> **Status:** Review Complete - Awaiting Phase 0 Execution
> **Reviews Completed:** Einstein Analysis + Staff Architect 7-Layer Review
> **Date:** 2026-02-02
> **Verdict:** CONDITIONAL GO

---

## Executive Summary

The mgrep integration specification has been critically reviewed by both Einstein (deep analysis) and Staff Architect (7-layer framework). Both reviews converged on the same core findings with Staff Architect identifying 2 additional issues.

**Recommendation:** Proceed with implementation AFTER completing mandatory Phase 0 (spec corrections).

### Finding Summary

| Severity | Count | Blockers |
|----------|-------|----------|
| Critical | 3 | Must fix before any implementation |
| High | 4 | Should fix before Phase 1 |
| Medium | 3 | Fix during implementation |
| Low | 2 | Optional improvements |

---

## Part 1: Review Synthesis

### 1.1 Critical Findings (Blockers)

#### CRITICAL-1: Agent File Format Mismatch

**Source:** Einstein #2, Staff Architect Finding 1.1

**Problem:** The spec assumes agents are defined in `agent.yaml` files:
```
~/.claude/agents/codebase-search/agent.yaml  ← DOES NOT EXIST
```

**Reality:** GOgent-Fortress uses unified Markdown files with YAML frontmatter:
```
~/.claude/agents/codebase-search/codebase-search.md  ← ACTUAL FILE
```

**Impact:** Every agent modification section (4.2-4.9) targets non-existent files. A developer would immediately fail.

**Required Action:**
1. Audit all agent paths in spec against actual filesystem
2. Rewrite all agent YAML blocks as frontmatter format
3. Verify paths with: `ls ~/.claude/agents/*/`

**Verification:**
```bash
# This should return .md files, not .yaml
find ~/.claude/agents -name "*.md" -type f | head -10
```

---

#### CRITICAL-2: Selection Guidance Creates Third Routing Mechanism

**Source:** Einstein #7, Staff Architect Finding 6.1

**Problem:** The spec adds tool selection logic in THREE places:
1. `routing-schema.json` → `selection_guidance` block
2. `agents-index.json` → trigger patterns
3. Agent definitions → `mgrep_integration.tool_selection.patterns`

**Current Architecture:** GOgent-Fortress uses exactly TWO routing mechanisms:
1. Trigger-based routing (`agents-index.json`)
2. Hook-enforced validation (`gogent-validate`)

**Impact:** Adding a third mechanism violates the "enforcement in hooks, not documentation" principle (LLM-guidelines.md §Enforcement Architecture).

**Required Action:**
Choose ONE integration pattern:
- **Option A:** Schema-driven (add to `routing-schema.json`, enforce via hook)
- **Option B:** Agent-instruction-driven (keep selection logic in agent .md files only)

**Recommendation:** Option B. Keep selection logic in agent instructions. Remove `selection_guidance` from schema entirely.

**Files to Modify:**
- Remove: `routing-schema.json` → `external.engines.mgrep.selection_guidance`
- Keep: Agent-level tool selection in individual agent instructions

---

#### CRITICAL-3: Parallel Telemetry Files Fragment ML Pipeline

**Source:** Einstein #3, Staff Architect Finding 2.2

**Problem:** The spec proposes new telemetry files:
- `mgrep-invocations.jsonl`
- `mgrep-outcomes.jsonl`

**Current Architecture:** GOgent-Fortress uses consolidated telemetry:
```
$XDG_DATA_HOME/gogent/
├── routing-decisions.jsonl          ← ALL tool decisions here
├── routing-decision-updates.jsonl   ← ALL outcomes here
└── ...
```

**Impact:** Creating parallel files fragments the ML export pipeline and violates the append-only consolidation principle.

**Required Action:**
Extend existing `RoutingDecision` struct instead of creating new files:

```go
// pkg/telemetry/routing_decision.go - ADD these fields

type RoutingDecision struct {
    // ... existing fields ...

    // mgrep-specific context (nil for non-mgrep calls)
    MgrepQuery    string `json:"mgrep_query,omitempty"`
    MgrepMode     string `json:"mgrep_mode,omitempty"`     // discover, answer, agentic, web_blended
    MgrepPath     string `json:"mgrep_path,omitempty"`
    MgrepFallback bool   `json:"mgrep_fallback,omitempty"` // true if grep fallback was used
}
```

**Files to Modify:**
- `pkg/telemetry/routing_decision.go` → Add optional mgrep fields
- `cmd/gogent-sharp-edge/main.go` → Populate fields when mgrep detected
- Remove: All references to `mgrep-invocations.jsonl` and `mgrep-outcomes.jsonl`

---

### 1.2 High Severity Findings

#### HIGH-1: Gemini-Slave Pattern Mischaracterization

**Source:** Einstein #1, Staff Architect Finding 1.2

**Problem:** Spec claims to follow "gemini-slave architectural pattern" but actually proposes a hybrid approach.

| Aspect | gemini-slave | mgrep (as proposed) |
|--------|--------------|---------------------|
| Invocation | Bash only (`cat files \| gemini-slave`) | Bash from agents |
| Agent Changes | None | Extensive `mgrep_integration:` blocks |
| Schema | Not in schema | New `external.engines.mgrep` |

**Required Action:**
Remove all "follows gemini-slave pattern" language. Rebrand as "native tool enhancement" which is what the spec actually describes.

**Files to Modify:**
- `tickets/mgrep/mgrep-integration-spec.md` Section 1.2 → Remove gemini-slave reference
- Keep the actual integration approach (it's fine, just mislabeled)

---

#### HIGH-2: Missing Agents in Priority Tiers

**Source:** Einstein #4, Staff Architect Finding 2.1

**Problem:** Several important agents are missing from the priority tiers:

| Missing Agent | Category | Benefit from mgrep |
|---------------|----------|-------------------|
| `impl-manager` | coordination | Find similar implementations, pattern discovery |
| `planner` | architecture | Understand codebase scope for strategy |
| `architect-reviewer` | review | Find design patterns for comparison |
| `typescript-pro` | implementation | Find TypeScript patterns |
| `react-pro` | implementation | Find component patterns |

**Required Action:**
Update Section 4.1 priority tiers:
- P1: Add `impl-manager` (coordination agent, high value)
- P2: Add `architect-reviewer`, `planner`
- P3: Already implicitly includes `*-pro` agents (verify explicit listing)

---

#### HIGH-3: detectMgrepInvocation Not in Deliverables

**Source:** Einstein #6, Staff Architect Finding 7.1

**Problem:** Section 5.3 documents `detectMgrepInvocation()` function but it's NOT listed in Phase 2 deliverables (Section 8).

**Required Action:**
Add to Phase 2 deliverables:
```markdown
6. [ ] Implement detectMgrepInvocation() in cmd/gogent-sharp-edge/main.go
```

---

#### HIGH-4: Zero Results Treated as Failure

**Source:** Einstein #5, Staff Architect Finding 3.1

**Problem:** Section 2.7 treats zero mgrep results as a fallback trigger:
```
Empty results | Zero matches returned | Use Grep with relaxed pattern
```

**Reality:** Zero semantic results is often a **valid signal**:
- "Find auth bugs" → 0 results = no obvious bugs (good news)
- "Where is feature X" → 0 results = feature doesn't exist (useful info)

**Required Action:**
Differentiate between:
- `mgrep_no_results` → Valid semantic signal, may NOT need fallback
- `mgrep_error` → Actual failure, fallback appropriate

Update Section 2.7:
```markdown
| Empty results (error) | mgrep returns error code | Use Grep |
| Empty results (valid) | Zero matches, exit 0 | Report "not found" - don't auto-fallback |
```

Add query-type-aware fallback logic:
- Intent discovery ("where is X") → Zero results is valid answer
- Scope assessment ("how big is X") → Zero results means minimal scope
- Pattern finding ("find similar to X") → Zero results MAY trigger fallback

---

### 1.3 Medium Severity Findings

#### MEDIUM-1: Missing Pre-Scout Integration

**Source:** Staff Architect Finding N.1 (NEW)

**Problem:** Spec adds mgrep to `haiku-scout` but doesn't address relationship to existing scout hierarchy:

```json
// routing-schema.json - current
"scout_protocol": {
  "primary": "gemini-slave scout",
  "fallback": "haiku-scout"
}
```

**Question:** Where does mgrep fit?
- Replace gemini-slave scout? (major change)
- Augment haiku-scout only? (inconsistent)
- Add as third option? (complexity)

**Required Action:**
Add explicit section defining mgrep's scout role:

```markdown
## 4.3.1 Scout Hierarchy with mgrep

mgrep augments haiku-scout's Phase 1 (semantic scope). It does NOT replace gemini-slave scout.

| Scout | Role | mgrep Usage |
|-------|------|-------------|
| gemini-slave scout | Large context analysis (1M+ tokens) | None (different purpose) |
| haiku-scout | Quick scope assessment | Phase 1: semantic scope |
| mgrep direct | User-invoked search | `/mgrep` skill |

Flow:
1. If scope unknown → gemini-slave scout (if available) OR haiku-scout
2. haiku-scout Phase 1 uses mgrep for semantic scope
3. Phase 2 uses traditional metrics
4. Combined output informs routing
```

---

#### MEDIUM-2: Missing Privacy/Security Review

**Source:** Staff Architect Finding N.2 (NEW)

**Problem:** Section 9.3 acknowledges mgrep indexes code to Mixedbread's cloud but doesn't address:
- Data retention policy
- Whether `.claude/` directory is indexed (memory, decisions, handoffs)
- Compliance considerations (credentials, proprietary code)
- User consent mechanism

**Required Action:**
Add Section 9.4 (or expand 9.3):

```markdown
## 9.4 Security and Privacy

### 9.4.1 Data Handling

**Indexed by default:**
- All files not in .gitignore or .mgrepignore
- File paths and structure
- Code content

**Excluded (via .mgrepignore):**
- `.claude/` directory (memory, decisions, handoffs, tmp)
- `.env`, `*.key`, `*.pem`, `credentials.*`
- `node_modules/`, `vendor/`, `.venv/`

### 9.4.2 Mixedbread Data Retention

[Document actual retention policy from Mixedbread ToS]

### 9.4.3 Compliance Considerations

For projects with sensitive code:
1. Review .mgrepignore before first `mgrep watch`
2. Consider `GOGENT_MGREP_ENABLED=0` for sensitive projects
3. Audit indexed content via `mgrep status` (if available)

### 9.4.4 Opt-Out

```bash
# Disable mgrep globally
export GOGENT_MGREP_ENABLED=0

# Or per-project via .mgrepignore
echo "**/*" >> .mgrepignore  # Exclude everything
```
```

---

#### MEDIUM-3: Haiku Agent Complexity Concern

**Source:** Einstein #8, Staff Architect Finding 6.2

**Problem:** Adding mgrep reasoning to `codebase-search` (Haiku tier, no thinking budget) may exceed Haiku's capabilities.

**Current identity:**
> "Fast file and code discovery specialist. Haiku-tier for mechanical extraction work."

**Proposed:** Tool selection logic, pattern matching, fallback handling.

**Options:**
1. Add thinking budget to codebase-search (2000 tokens)
2. Keep mgrep usage simple (no selection logic)
3. Create separate `semantic-search` agent at Haiku+Thinking tier

**Recommendation:** Option 2 (simplify). Keep codebase-search mechanical:
- Use mgrep FIRST for all queries
- Fallback to grep on ERROR only (not zero results)
- No pattern matching for tool selection

This preserves Haiku's fast/cheap purpose while adding semantic capability.

---

### 1.4 Low Severity Findings

#### LOW-1: Timeout Cascade Risk

**Source:** Staff Architect Finding 3.2

**Problem:** If mgrep times out (30s), then grep fallback runs (10-30s), total latency could reach 60s+.

**Mitigation:** Consider parallel execution or adaptive timeouts. Low priority - can address in Phase 5 polish.

---

#### LOW-2: Incomplete Test Coverage

**Source:** Staff Architect Finding 5.1

**Problem:** Test specification missing:
- Mock strategy for testing without live mgrep API
- CI integration (GitHub Actions)
- Regression tests for existing Grep behavior
- Concurrent execution tests

**Required Action:** Add to Phase 5 deliverables. Low priority for initial implementation.

---

## Part 2: Restructured Implementation Phases

### Original vs Recommended

| Original | Duration | Recommended | Duration |
|----------|----------|-------------|----------|
| Phase 1: Foundation | Week 1 | **Phase 0: Spec Corrections** | 1 week |
| Phase 2: Critical Agents | Weeks 2-3 | **Phase 1: Foundation + Skill** | 1 week |
| Phase 3: Research Agents | Week 4 | **Phase 2: Scout Integration** | 2 weeks |
| Phase 4: Review Pipeline | Week 5 | **Phase 3: Agent Rollout** | 2 weeks |
| Phase 5: Telemetry & Polish | Week 6 | (Merged into Phase 2-3) | - |
| | 6 weeks | | 6 weeks |

The restructuring adds a mandatory **Phase 0** and consolidates telemetry work into earlier phases rather than deferring it.

---

## Part 3: Phase Specifications

### Phase 0: Spec Corrections (MANDATORY)

**Duration:** 1 week
**Gate:** Must complete before ANY implementation

#### Deliverables

| # | Task | File(s) | Effort |
|---|------|---------|--------|
| 0.1 | Audit agent file paths | `ls ~/.claude/agents/*/` | 30m |
| 0.2 | Rewrite all agent YAML → frontmatter format | Spec Sections 4.2-4.9 | 4h |
| 0.3 | Remove `selection_guidance` from schema section | Spec Section 3.1 | 30m |
| 0.4 | Remove parallel telemetry file definitions | Spec Sections 3.2, 5.1-5.4 | 1h |
| 0.5 | Add `detectMgrepInvocation` to Phase 2 deliverables | Spec Section 8 | 15m |
| 0.6 | Update priority tiers (add missing agents) | Spec Section 4.1 | 30m |
| 0.7 | Fix zero-results fallback logic | Spec Section 2.7 | 30m |
| 0.8 | Remove gemini-slave pattern claims | Spec Section 1.2 | 15m |
| 0.9 | Add scout hierarchy clarification | New Section 4.3.1 | 1h |
| 0.10 | Add security/privacy section | New Section 9.4 | 1h |

**Total Estimated Effort:** 8-10 hours

#### Validation Checklist

- [ ] All agent paths in spec match actual filesystem paths
- [ ] No `agent.yaml` references remain (all converted to `.md` frontmatter)
- [ ] No `selection_guidance` in routing-schema.json section
- [ ] No `mgrep-invocations.jsonl` or `mgrep-outcomes.jsonl` references
- [ ] `detectMgrepInvocation` listed in Phase 2 deliverables
- [ ] impl-manager, planner, architect-reviewer appear in priority tiers
- [ ] Zero-results distinguished from errors in fallback logic
- [ ] No "follows gemini-slave pattern" claims
- [ ] Scout hierarchy documented
- [ ] Security/privacy section added

---

### Phase 1: Foundation + Skill

**Duration:** 1 week
**Prerequisites:** Phase 0 complete
**Goal:** Validate mgrep works end-to-end before agent integration

#### Deliverables

| # | Task | File(s) |
|---|------|---------|
| 1.1 | Install mgrep, test authentication | Local environment |
| 1.2 | Create `/mgrep` skill | `~/.claude/skills/mgrep/SKILL.md` |
| 1.3 | Add mgrep engine definition to schema | `~/.claude/routing-schema.json` |
| 1.4 | Create .mgrepignore template | Project root `.mgrepignore` |
| 1.5 | Add mgrep fields to RoutingDecision struct | `pkg/telemetry/routing_decision.go` |
| 1.6 | Document mgrep in CLAUDE.md (reference only) | `~/.claude/CLAUDE.md` |

#### Detailed Specifications

##### 1.2 /mgrep Skill

**File:** `~/.claude/skills/mgrep/SKILL.md`

```markdown
# /mgrep Skill

## Purpose

Semantic code discovery using mgrep. Find code by describing what it does,
not what patterns it matches.

## Invocation

| Command | Behavior |
|---------|----------|
| `/mgrep [query]` | Search current directory |
| `/mgrep [query] [path]` | Search specific path |
| `/mgrep -a [query]` | Search and synthesize answer |
| `/mgrep --agentic [query]` | Deep multi-query analysis |
| `/mgrep --web [query]` | Include external web results |

## Workflow

1. Check mgrep availability: `command -v mgrep && mgrep --version`
2. If unavailable: Report and offer grep alternative
3. Execute: `mgrep "${query}" ${path:-.} -m 25`
4. Format output with file:line references
5. Offer to read top results

## Fallback

If mgrep unavailable:
- Extract keywords from natural language query
- Run grep with extracted keywords
- Note limitation in output
```

##### 1.5 RoutingDecision Extension

**File:** `pkg/telemetry/routing_decision.go`

Add to existing struct:
```go
type RoutingDecision struct {
    // ... existing fields (DecisionID, SessionID, Timestamp, etc.) ...

    // mgrep-specific context (optional, nil for non-mgrep calls)
    MgrepQuery    string `json:"mgrep_query,omitempty"`
    MgrepMode     string `json:"mgrep_mode,omitempty"`     // discover, answer, agentic, web_blended
    MgrepPath     string `json:"mgrep_path,omitempty"`
    MgrepResults  int    `json:"mgrep_results,omitempty"`  // number of files returned
    MgrepFallback bool   `json:"mgrep_fallback,omitempty"` // true if grep fallback used
    MgrepDurationMs int64 `json:"mgrep_duration_ms,omitempty"`
}
```

#### Validation

- [ ] `/mgrep "test query"` executes successfully
- [ ] Fallback to grep works when `GOGENT_MGREP_ENABLED=0`
- [ ] RoutingDecision struct compiles with new fields
- [ ] .mgrepignore excludes `.claude/` directory

---

### Phase 2: Scout Integration

**Duration:** 2 weeks
**Prerequisites:** Phase 1 complete
**Goal:** Semantic scope assessment integrated into scout workflow

#### Deliverables

| # | Task | File(s) |
|---|------|---------|
| 2.1 | Update haiku-scout with semantic phase | `~/.claude/agents/haiku-scout/haiku-scout.md` |
| 2.2 | Implement detectMgrepInvocation in hook | `cmd/gogent-sharp-edge/main.go` |
| 2.3 | Update calculate-complexity.sh for semantic scope | `~/.claude/scripts/calculate-complexity.sh` |
| 2.4 | Update scout_metrics.json schema | Documentation + validation |
| 2.5 | Add mgrep detection tests | `cmd/gogent-sharp-edge/mgrep_test.go` |

#### Detailed Specifications

##### 2.1 haiku-scout Semantic Phase

**File:** `~/.claude/agents/haiku-scout/haiku-scout.md`

Add to frontmatter:
```yaml
scout_protocol:
  phases:
    phase_1_semantic:
      description: "Assess conceptual scope via mgrep"
      condition: "mgrep available (command -v mgrep)"
      command: "mgrep \"${task_description}\" . -m 50"
      timeout_ms: 30000
      fallback: "skip to phase_2"

    phase_2_mechanical:
      description: "Gather LoC metrics"
      # ... existing mechanical metrics ...

    phase_3_synthesize:
      description: "Combine semantic + mechanical"
      output_schema:
        semantic_scope:
          query: string
          file_count: int
          mgrep_available: bool
        mechanical_scope:
          total_files: int
          total_lines: int
        routing_recommendation:
          tier: string
          confidence: float
```

##### 2.2 detectMgrepInvocation

**File:** `cmd/gogent-sharp-edge/main.go`

```go
// detectMgrepInvocation checks if a Bash command is an mgrep call
// and extracts relevant telemetry fields
func detectMgrepInvocation(event *routing.PostToolEvent) *MgrepContext {
    if event.ToolName != "Bash" {
        return nil
    }

    command, ok := event.ToolInput["command"].(string)
    if !ok {
        return nil
    }

    // Check if command starts with mgrep
    trimmed := strings.TrimSpace(command)
    if !strings.HasPrefix(trimmed, "mgrep ") {
        return nil
    }

    ctx := &MgrepContext{
        Query: extractMgrepQuery(trimmed),
        Mode:  extractMgrepMode(trimmed),
        Path:  extractMgrepPath(trimmed),
    }

    // Parse result count from output
    if output, ok := event.ToolResponse["output"].(string); ok {
        ctx.ResultCount = countMgrepResults(output)
    }

    // Check for errors
    if exitCode, ok := event.ToolResponse["exit_code"].(float64); ok && exitCode != 0 {
        ctx.Error = true
    }

    return ctx
}

type MgrepContext struct {
    Query       string
    Mode        string // discover, answer, agentic, web_blended
    Path        string
    ResultCount int
    Error       bool
}
```

##### 2.4 Updated scout_metrics.json Schema

```json
{
  "semantic_scope": {
    "query": "refactor authentication module",
    "file_count": 12,
    "directories_involved": ["pkg/auth/", "cmd/api/"],
    "mgrep_available": true,
    "mgrep_duration_ms": 1250
  },
  "mechanical_scope": {
    "total_files": 23,
    "total_lines": 5420,
    "estimated_tokens": 18500
  },
  "routing_recommendation": {
    "recommended_tier": "sonnet",
    "confidence": 0.85,
    "reasoning": "12 semantically relevant files, moderate complexity"
  }
}
```

#### Validation

- [ ] haiku-scout produces both semantic and mechanical scope
- [ ] `.claude/tmp/scout_metrics.json` contains `semantic_scope` field
- [ ] `gogent-sharp-edge` detects mgrep commands and populates RoutingDecision fields
- [ ] `gogent-ml-export routing-decisions` shows mgrep-specific fields
- [ ] calculate-complexity.sh uses semantic file count in tier calculation

---

### Phase 3: Agent Rollout

**Duration:** 2 weeks
**Prerequisites:** Phase 2 complete
**Goal:** Integrate mgrep into P0, P1, P2 agents

#### Week 1: P0 + P1 Agents

| # | Agent | File | Integration Type |
|---|-------|------|------------------|
| 3.1 | codebase-search | `~/.claude/agents/codebase-search/codebase-search.md` | Primary search tool |
| 3.2 | librarian | `~/.claude/agents/librarian/librarian.md` | Internal-first research |
| 3.3 | orchestrator | `~/.claude/agents/orchestrator/orchestrator.md` | Disambiguation protocol |
| 3.4 | impl-manager | `~/.claude/agents/impl-manager/impl-manager.md` | Pattern discovery |

#### Week 2: P2 Agents

| # | Agent | File | Integration Type |
|---|-------|------|------------------|
| 3.5 | architect | `~/.claude/agents/architect/architect.md` | Pattern discovery |
| 3.6 | review-orchestrator | `~/.claude/agents/review-orchestrator/review-orchestrator.md` | Domain detection |
| 3.7 | architect-reviewer | `~/.claude/agents/architect-reviewer/architect-reviewer.md` | Design pattern comparison |
| 3.8 | planner | `~/.claude/agents/planner/planner.md` | Scope understanding |

#### Detailed Specifications

##### 3.1 codebase-search Integration

**Approach:** Simple mgrep-first, no complex selection logic.

Add to frontmatter:
```yaml
tools:
  - Glob
  - Grep
  - Read
  - Bash  # For mgrep

mgrep:
  enabled: true
  mode: discover
  default_count: 25
  strategy: mgrep_first_grep_fallback
```

Add to instructions:
```markdown
## Search Strategy

1. **Try mgrep first** for all queries:
   ```bash
   mgrep "${query}" . -m 25
   ```

2. **Fallback to grep** ONLY if mgrep returns an error (not zero results):
   - Extract keywords from query
   - Run: `grep -r "${keywords}" . --include="*.go" -l`

3. **Report zero results** as valid finding if mgrep succeeds with no matches.
```

##### 3.2 librarian Integration (Internal-First)

Add to instructions:
```markdown
## Research Strategy

**Step 1: Check internal patterns first**
```bash
mgrep "${topic} patterns in this codebase" . -m 10 -a
```

**Step 2: Assess sufficiency**
- If internal patterns cover the use case → return with caveats
- If partial → augment with external search
- If none → proceed to external search

**Step 3: External search (if needed)**
- For best practices: `mgrep --web "${topic}" -a`
- For API docs: Use WebSearch/WebFetch

**Step 4: Reconcile**
- Compare internal practice to external recommendation
- Note discrepancies and suggest alignment
```

##### 3.3 orchestrator Disambiguation

Add to instructions:
```markdown
## Semantic Disambiguation Protocol

When scope is ambiguous or spans modules:

**Step 1: Semantic scope**
```bash
mgrep --agentic "${task_description}" . -a
```

**Step 2: Parse output for:**
- Modules/directories involved
- Cross-module relationships
- Recommended specialist agents

**Step 3: Verify with targeted grep**
Confirm semantic findings, identify false positives.

**Step 4: Spawn informed agents**
Include semantically-informed scope in agent prompts.
```

#### Validation

- [ ] codebase-search uses mgrep for intent queries, falls back on error only
- [ ] librarian checks internal patterns before external search
- [ ] orchestrator uses semantic disambiguation for ambiguous scope
- [ ] impl-manager finds similar implementations via mgrep
- [ ] All modified agents have Bash in their tools list
- [ ] Telemetry captures mgrep invocations from all agents

---

## Part 4: Validation Matrix

### Phase 0 Exit Criteria

| Criterion | Verification Method |
|-----------|---------------------|
| All agent paths verified | `ls ~/.claude/agents/*/` matches spec |
| No agent.yaml references | `grep -r "agent.yaml" tickets/mgrep/` returns empty |
| No selection_guidance in schema | `grep "selection_guidance" spec` returns empty |
| No parallel telemetry files | `grep "mgrep-invocations\|mgrep-outcomes" spec` returns empty |
| Missing agents added | impl-manager, planner, architect-reviewer in spec |
| Security section exists | Section 9.4 present |

### Phase 1 Exit Criteria

| Criterion | Verification Method |
|-----------|---------------------|
| /mgrep skill works | `/mgrep "test"` returns results |
| Fallback works | `GOGENT_MGREP_ENABLED=0 /mgrep "test"` uses grep |
| Schema updated | `jq '.external.engines.mgrep' routing-schema.json` |
| Telemetry struct extended | `go build ./pkg/telemetry/...` succeeds |

### Phase 2 Exit Criteria

| Criterion | Verification Method |
|-----------|---------------------|
| Scout semantic phase | `.claude/tmp/scout_metrics.json` has `semantic_scope` |
| Hook detection | `gogent-ml-export routing-decisions` shows mgrep fields |
| Complexity calculation | `calculate-complexity.sh` considers semantic scope |

### Phase 3 Exit Criteria

| Criterion | Verification Method |
|-----------|---------------------|
| P0 agents integrated | codebase-search, haiku-scout use mgrep |
| P1 agents integrated | librarian, orchestrator, impl-manager use mgrep |
| P2 agents integrated | architect, review-orchestrator use mgrep |
| Telemetry complete | All agent mgrep calls appear in routing-decisions.jsonl |

---

## Part 5: Risk Mitigation

### Risk 1: mgrep API Changes

**Mitigation:**
- Pin mgrep version in documentation
- Wrap mgrep calls in helper function for easy updates
- Fallback to grep ensures functionality continues

### Risk 2: Cost Overruns

**Mitigation:**
- Track mgrep costs via telemetry
- Set query count limits per agent
- Monitor via `gogent-ml-export stats`

### Risk 3: Privacy Concerns

**Mitigation:**
- .mgrepignore excludes sensitive directories by default
- Document data handling in CLAUDE.md
- Provide opt-out mechanism (`GOGENT_MGREP_ENABLED=0`)

### Risk 4: Performance Degradation

**Mitigation:**
- Timeouts on all mgrep calls (30s default)
- Fallback to grep prevents blocking
- Monitor latency via telemetry

---

## Part 6: Success Metrics

### Quantitative Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction on exploration | 40-60% | Pre/post comparison on benchmark queries |
| Scout routing accuracy | +40% | Track tier changes after scout |
| False positive rate | -50% | Files returned but not read |
| mgrep success rate | >90% | Invocations without fallback |
| mgrep latency P95 | <3s | Telemetry DurationMs |

### Qualitative Targets

- Agents find relevant code with natural language queries
- Scout produces more accurate scope assessments
- Librarian checks internal patterns before external search
- Orchestrator disambiguates scope more effectively

---

## Part 7: File Manifest

### New Files

| File | Purpose | Phase |
|------|---------|-------|
| `~/.claude/skills/mgrep/SKILL.md` | User-invoked skill | 1 |
| `.mgrepignore` | Index exclusion patterns | 1 |
| `cmd/gogent-sharp-edge/mgrep_test.go` | Detection tests | 2 |

### Modified Files

| File | Changes | Phase |
|------|---------|-------|
| `pkg/telemetry/routing_decision.go` | Add mgrep fields | 1 |
| `~/.claude/routing-schema.json` | Add mgrep engine | 1 |
| `~/.claude/CLAUDE.md` | Add mgrep reference | 1 |
| `cmd/gogent-sharp-edge/main.go` | Add mgrep detection | 2 |
| `~/.claude/scripts/calculate-complexity.sh` | Use semantic scope | 2 |
| `~/.claude/agents/haiku-scout/haiku-scout.md` | Add semantic phase | 2 |
| `~/.claude/agents/codebase-search/codebase-search.md` | Add mgrep integration | 3 |
| `~/.claude/agents/librarian/librarian.md` | Internal-first strategy | 3 |
| `~/.claude/agents/orchestrator/orchestrator.md` | Disambiguation protocol | 3 |
| `~/.claude/agents/impl-manager/impl-manager.md` | Pattern discovery | 3 |
| `~/.claude/agents/architect/architect.md` | Pattern discovery | 3 |
| `~/.claude/agents/review-orchestrator/review-orchestrator.md` | Domain detection | 3 |
| `docs/ARCHITECTURE.md` | Add mgrep integration section | 3 |

### Deprecated (Remove from Original Spec)

| File Reference | Reason |
|----------------|--------|
| `mgrep-invocations.jsonl` | Use RoutingDecision instead |
| `mgrep-outcomes.jsonl` | Use RoutingDecision instead |
| `pkg/telemetry/mgrep_invocation.go` | Merged into routing_decision.go |
| `pkg/telemetry/mgrep_logging.go` | Not needed |
| `agent.yaml` references | Use .md with frontmatter |

---

## Appendix A: Corrected Agent File Paths

Based on filesystem audit (`ls ~/.claude/agents/*/`):

| Spec Reference | Actual Path |
|----------------|-------------|
| `codebase-search/agent.yaml` | `codebase-search/codebase-search.md` |
| `haiku-scout/agent.yaml` | `haiku-scout/haiku-scout.md` |
| `librarian/agent.yaml` | `librarian/librarian.md` |
| `orchestrator/agent.yaml` | `orchestrator/orchestrator.md` |
| `architect/agent.yaml` | `architect/architect.md` |
| `review-orchestrator/agent.yaml` | `review-orchestrator/review-orchestrator.md` |
| `impl-manager/agent.yaml` | `impl-manager/impl-manager.md` |
| `memory-archivist/agent.yaml` | `memory-archivist/memory-archivist.md` |

---

## Appendix B: Quick Reference Commands

```bash
# Verify mgrep installation
command -v mgrep && mgrep --version

# Check mgrep authentication
mgrep "test" . -m 1

# Index project
mgrep watch

# Manual sync before search
mgrep search -s "query"

# Disable mgrep (fallback to grep)
export GOGENT_MGREP_ENABLED=0

# Check telemetry for mgrep calls
gogent-ml-export routing-decisions | jq 'select(.mgrep_query != null)'

# Monitor mgrep usage
gogent-ml-export stats | grep mgrep
```

---

**End of Action Plan**

*This document supersedes the implementation phases in the original spec. Execute Phase 0 corrections before proceeding.*
