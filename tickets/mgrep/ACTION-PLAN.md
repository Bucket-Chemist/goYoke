# mgrep Integration Action Plan

> **Version:** 1.0.0
> **Status:** Ready for Implementation
> **Created:** 2026-02-02
> **Source:** Einstein Analysis + Staff Architect Critical Review

---

## Executive Summary

This action plan restructures the mgrep integration from a 6-week implementation to a **validation-first approach** that proves value before committing to full integration.

### Key Changes from Original Spec

| Original | Revised | Rationale |
|----------|---------|-----------|
| 6-week phased rollout | 3-phase proof-of-value | Reduce investment risk |
| All agents get mgrep | P0 agents only initially | Focus resources |
| Full telemetry system | Minimal viable metrics | Avoid premature optimization |
| Modify 12+ files | Modify 3-4 files | Smaller blast radius |

---

## Phase 0: Validation (1 Week)

**Goal:** Prove mgrep provides measurable value before any code changes.

### 0.1 Manual Benchmarking

Run 10 representative queries manually, comparing mgrep vs grep:

| Query Type | Example | Measure |
|------------|---------|---------|
| Intent-based | "where is authentication implemented" | Files returned, precision |
| Concept discovery | "how does error handling work" | Relevance of top 5 |
| Cross-module | "what calls the event dispatcher" | Completeness |
| Pattern finding | "find rate limiting code" | False positive rate |

**Success Criteria:**
- [ ] mgrep returns ≥30% fewer irrelevant files than grep
- [ ] Top 5 mgrep results are relevant ≥70% of the time
- [ ] Query latency <3s for 90% of queries

**Deliverable:** `tickets/mgrep/benchmark-results.md` with raw data

### 0.2 Privacy/Cost Assessment

Before any integration:

- [ ] Document what mgrep indexes (file contents, paths, metadata)
- [ ] Confirm .gitignore/.mgrepignore are respected
- [ ] Get Mixedbread pricing tier for expected query volume
- [ ] Verify no sensitive data in indexed paths

**Deliverable:** `tickets/mgrep/privacy-cost-assessment.md`

### 0.3 Fallback Verification

Verify fallback works reliably:

```bash
# Test 1: mgrep unavailable
GOGENT_MGREP_ENABLED=0 # Should use grep

# Test 2: Authentication failure
unset MXBAI_API_KEY && mgrep "test" # Should error gracefully

# Test 3: Timeout
# Simulate slow network, verify 30s timeout works
```

**Success Criteria:**
- [ ] All fallback scenarios produce usable grep results
- [ ] No crashes or hangs on mgrep failure
- [ ] Error messages are actionable

---

## Phase 1: Minimal Integration (2 Weeks)

**Goal:** Integrate mgrep into ONE agent (codebase-search) with full observability.

### 1.1 /mgrep Skill (Week 1)

Create user-invokable skill for direct mgrep access.

**File:** `~/.claude/skills/mgrep/SKILL.md`

```markdown
# /mgrep Skill

## Purpose
Semantic code discovery using mgrep. Find code by describing what it does.

## Invocation
| Command | Behavior |
|---------|----------|
| `/mgrep [query]` | Search current directory |
| `/mgrep [query] [path]` | Search specific path |
| `/mgrep -a [query]` | Search and synthesize answer |

## Examples
```bash
/mgrep "where is authentication implemented"
/mgrep -a "how does error handling work"
/mgrep "rate limiting" pkg/
```

## Fallback
If mgrep unavailable, falls back to grep with extracted keywords.
```

**Acceptance Criteria:**
- [ ] `/mgrep "test query"` returns results
- [ ] Fallback to grep works when mgrep unavailable
- [ ] Output includes file:line references

### 1.2 codebase-search Integration (Week 2)

Modify codebase-search to use mgrep for intent-based queries.

**File:** `~/.claude/agents/codebase-search/agent.yaml`

**Changes:**
```yaml
# ADD to tools section
tools:
  - Glob
  - Grep
  - Read
  - Bash  # NEW: for mgrep invocation

# ADD new section
mgrep_integration:
  enabled: true
  mode: discover
  default_count: 25

  tool_selection:
    use_mgrep:
      patterns:
        - "where is .* implemented"
        - "how does .* work"
        - "find .* that handles"
        - "what .* for"

    use_grep:
      patterns:
        - "grep for .*"
        - "find uses of .*"
        - "exact match .*"
```

**File:** `~/.claude/agents/codebase-search/CLAUDE.md`

Add tool selection guide:
```markdown
## Tool Selection: mgrep vs Grep

| Query Type | Tool | Example |
|------------|------|---------|
| Intent-based | mgrep | "where is authentication implemented" |
| Concept discovery | mgrep | "how does error handling work" |
| Exact symbol | Grep | "find uses of parseEvent" |
| Pattern match | Grep | "grep for TODO comments" |

**mgrep Invocation:**
```bash
mgrep "your natural language query" path/ -m 25
```

**Fallback:** If mgrep fails or returns 0 results, retry with Grep.
```

**Acceptance Criteria:**
- [ ] codebase-search correctly routes intent queries to mgrep
- [ ] codebase-search correctly routes pattern queries to grep
- [ ] Fallback triggers on mgrep failure
- [ ] Results are actionable (file:line format)

### 1.3 Minimal Telemetry (Week 2)

Track mgrep usage without building full telemetry system.

**Approach:** Append to existing tool event log, not new files.

**File:** `cmd/gogent-sharp-edge/main.go`

Add mgrep detection:
```go
// In PostToolUse handler
func detectMgrepInvocation(event *routing.PostToolEvent) {
    if event.ToolName != "Bash" {
        return
    }

    command, ok := event.ToolInput["command"].(string)
    if !ok || !strings.HasPrefix(strings.TrimSpace(command), "mgrep ") {
        return
    }

    // Log to existing ml-tool-events.jsonl
    logToolEvent(event.SessionID, "mgrep", map[string]interface{}{
        "query": extractMgrepQuery(command),
        "mode":  extractMgrepMode(command),
        "success": event.ToolResponse["exit_code"] == 0,
    })
}
```

**Acceptance Criteria:**
- [ ] mgrep invocations appear in ml-tool-events.jsonl
- [ ] Can query: "How many mgrep calls this week?"
- [ ] Can query: "What's the mgrep success rate?"

---

## Phase 2: Expansion (2 Weeks)

**Gate:** Only proceed if Phase 1 shows measurable improvement.

**Metrics to evaluate:**
- Token reduction on exploration tasks (target: 30%+)
- False positive rate reduction (target: 40%+)
- User satisfaction (qualitative)

### 2.1 haiku-scout Integration

Add semantic scoping to scout reconnaissance.

**File:** `~/.claude/agents/haiku-scout/agent.yaml`

**Changes:**
```yaml
# ADD semantic phase to scout protocol
scout_protocol:
  phases:
    phase_1_semantic:
      description: "Assess conceptual scope via mgrep"
      condition: "mgrep available"
      command: |
        mgrep "${task_description}" ${path:-.} -m 50
      output_file: .claude/tmp/mgrep-scope.txt
      timeout_ms: 30000
      fallback: "skip to phase_2"

    phase_2_mechanical:
      description: "Gather LoC metrics"
      # ... existing mechanical metrics

    phase_3_synthesize:
      description: "Combine semantic + mechanical"
      output_file: .claude/tmp/scout_metrics.json
```

**Output Schema Extension:**
```json
{
  "semantic_scope": {
    "query": "task description",
    "file_count": 15,
    "directories_involved": ["pkg/auth/", "pkg/session/"],
    "mgrep_available": true
  },
  "mechanical_scope": {
    "total_files": 42,
    "total_lines": 3500
  },
  "routing_recommendation": {
    "recommended_tier": "sonnet",
    "confidence": 0.85
  }
}
```

**Acceptance Criteria:**
- [ ] Scout produces mgrep-scope.txt when mgrep available
- [ ] Scout falls back gracefully when mgrep unavailable
- [ ] Routing recommendations improve with semantic scope

### 2.2 librarian Integration

Add internal-first search strategy.

**File:** `~/.claude/agents/librarian/agent.yaml`

**Changes:**
```yaml
# ADD research strategy
research_strategy:
  step_1_internal:
    description: "Check if project has existing patterns"
    command: |
      mgrep "${topic} patterns in this codebase" . -m 10 -a
    condition: "mgrep available"

  step_2_assess:
    decision:
      sufficient: "Internal patterns cover use case"
      partial: "Need external augmentation"
      none: "Proceed to external search"

  step_3_external:
    # Existing WebSearch/WebFetch logic
```

**Acceptance Criteria:**
- [ ] Librarian checks internal patterns before external search
- [ ] Internal patterns are surfaced when relevant
- [ ] External search still works when needed

### 2.3 orchestrator Integration

Add semantic disambiguation for ambiguous scope.

**File:** `~/.claude/agents/orchestrator/agent.yaml`

**Changes:**
```yaml
# ADD disambiguation protocol
disambiguation_protocol:
  step_1_semantic_scope:
    description: "Understand conceptual scope"
    condition: "Scope is ambiguous"
    command: |
      mgrep --agentic "${task_description}" . -a
    output: "semantic_scope_analysis"

  step_2_verify_scope:
    description: "Validate with targeted grep"
    purpose: "Confirm findings, catch false positives"

  step_3_spawn_informed:
    description: "Spawn agents with accurate scope"
```

**Acceptance Criteria:**
- [ ] Orchestrator uses mgrep for scope disambiguation
- [ ] False positive detection via grep verification
- [ ] Agent spawning is more accurate

---

## Phase 3: Polish (1 Week)

**Gate:** Only proceed if Phase 2 shows continued improvement.

### 3.1 Configuration Files

**File:** `.mgrepignore` (project root)

```gitignore
# Build artifacts
dist/
build/
*.exe

# Dependencies
node_modules/
vendor/
.venv/

# GOgent internals
.claude/tmp/
.claude/session-archive/
.claude/statsig/

# Sensitive
.env
*.key
credentials.*
```

**File:** Add to shell profile or `.envrc`

```bash
# mgrep configuration
export MGREP_MAX_COUNT=25
export MGREP_RERANK=1
export GOGENT_MGREP_ENABLED=1
```

### 3.2 Documentation Updates

**File:** `~/.claude/CLAUDE.md`

Add mgrep reference section (NOT enforcement):
```markdown
## mgrep Integration

mgrep provides semantic search for intent-based queries. Agents automatically
select mgrep vs grep based on query type.

**Privacy Note:** mgrep indexes to Mixedbread cloud. Respects .gitignore/.mgrepignore.

**Disable:** `export GOGENT_MGREP_ENABLED=0`

**Reference:** See `tickets/mgrep/mgrep-integration-spec.md` for full specification.
```

**File:** `docs/ARCHITECTURE.md`

Add external engines section documenting mgrep alongside gemini-slave.

### 3.3 Routing Schema Update

**File:** `~/.claude/routing-schema.json`

Add mgrep engine definition:
```json
{
  "external": {
    "engines": {
      "mgrep": {
        "type": "semantic-search",
        "availability_check": "command -v mgrep && mgrep --version",
        "invocation_modes": {
          "discover": "mgrep \"${query}\" ${path} -m ${count:-25}",
          "answer": "mgrep \"${query}\" ${path} -m 10 -a"
        },
        "fallback": {
          "tool": "Grep",
          "triggers": ["not_installed", "auth_failed", "timeout", "error"]
        }
      }
    }
  }
}
```

---

## Risk Mitigations

### R1: mgrep Unavailability

**Risk:** mgrep service down, not installed, or auth expired.

**Mitigation:** Every integration MUST have grep fallback:
```yaml
fallback:
  tool: Grep
  triggers:
    - command_not_found
    - authentication_failed
    - timeout_exceeded
    - error_returned
    - zero_results
```

**Validation:** Test each fallback scenario before deployment.

### R2: False Sense of Precision

**Risk:** mgrep returns "relevant" results that aren't actually useful.

**Mitigation:**
- Verify mgrep results with targeted grep before acting
- Track precision metrics (files actually read / files returned)
- Set conservative default result count (25, not 100)

### R3: Cost Creep

**Risk:** mgrep API costs exceed budget.

**Mitigation:**
- Set MGREP_MAX_COUNT=25 (not unlimited)
- Track invocation count in telemetry
- Review costs weekly during rollout
- Hard limit: If costs exceed $X/month, disable integration

### R4: Privacy Concerns

**Risk:** Sensitive code indexed to third-party service.

**Mitigation:**
- Document what's indexed in privacy assessment
- Ensure .gitignore/.mgrepignore respected
- Provide GOGENT_MGREP_ENABLED=0 kill switch
- Never index .env, credentials, or .claude/tmp/

### R5: Integration Complexity

**Risk:** Changes break existing agent behavior.

**Mitigation:**
- Phase 1 touches only ONE agent (codebase-search)
- Expansion gated on measured improvement
- Each phase has rollback plan (remove mgrep section from yaml)

---

## Success Metrics

### Phase 0 (Validation)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Precision improvement | ≥30% | Manual benchmark |
| Top-5 relevance | ≥70% | Manual assessment |
| Query latency | <3s p90 | Manual timing |

### Phase 1 (Minimal Integration)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction | ≥20% | Compare exploration sessions |
| Fallback reliability | 100% | Automated tests |
| User complaints | 0 | Qualitative |

### Phase 2 (Expansion)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction | ≥40% | Compare exploration sessions |
| Scout routing accuracy | +30% | Track tier changes |
| False positive rate | -40% | Files returned vs used |

### Phase 3 (Polish)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Documentation complete | Yes | Review checklist |
| No regressions | Yes | Benchmark suite |
| Cost within budget | Yes | Mixedbread billing |

---

## File Manifest

### New Files

| File | Phase | Purpose |
|------|-------|---------|
| `tickets/mgrep/benchmark-results.md` | 0 | Manual benchmark data |
| `tickets/mgrep/privacy-cost-assessment.md` | 0 | Privacy/cost analysis |
| `~/.claude/skills/mgrep/SKILL.md` | 1 | User-invoked skill |
| `.mgrepignore` | 3 | Index exclusion patterns |

### Modified Files

| File | Phase | Changes |
|------|-------|---------|
| `~/.claude/agents/codebase-search/agent.yaml` | 1 | Add mgrep_integration section |
| `~/.claude/agents/codebase-search/CLAUDE.md` | 1 | Add tool selection guide |
| `cmd/gogent-sharp-edge/main.go` | 1 | Add mgrep detection |
| `~/.claude/agents/haiku-scout/agent.yaml` | 2 | Add semantic phase |
| `~/.claude/agents/librarian/agent.yaml` | 2 | Add internal-first strategy |
| `~/.claude/agents/orchestrator/agent.yaml` | 2 | Add disambiguation protocol |
| `~/.claude/routing-schema.json` | 3 | Add mgrep engine definition |
| `~/.claude/CLAUDE.md` | 3 | Add mgrep reference section |
| `docs/ARCHITECTURE.md` | 3 | Add external engines section |

---

## Rollback Plan

### Phase 1 Rollback

If Phase 1 fails to show improvement:

1. Remove mgrep_integration section from codebase-search/agent.yaml
2. Remove tool selection guide from codebase-search/CLAUDE.md
3. Remove mgrep detection from gogent-sharp-edge
4. Archive /mgrep skill (don't delete, may revisit)

**Time to rollback:** <30 minutes

### Phase 2 Rollback

If Phase 2 introduces regressions:

1. Revert agent.yaml changes for haiku-scout, librarian, orchestrator
2. Keep Phase 1 changes (proven valuable)

**Time to rollback:** <1 hour

### Full Rollback

If mgrep integration is abandoned:

1. `git revert` all mgrep-related commits
2. Set `GOGENT_MGREP_ENABLED=0` in environment
3. Document lessons learned in `tickets/mgrep/post-mortem.md`

---

## Implementation Tickets

### MGREP-001: Phase 0 Validation

**Priority:** P0
**Estimate:** 3-4 hours
**Assignee:** Manual (human)

**Tasks:**
- [ ] Install mgrep, authenticate
- [ ] Run 10 benchmark queries
- [ ] Document precision/latency results
- [ ] Complete privacy/cost assessment
- [ ] Test fallback scenarios

**Acceptance:** benchmark-results.md and privacy-cost-assessment.md created with passing metrics.

---

### MGREP-002: /mgrep Skill

**Priority:** P0
**Estimate:** 2 hours
**Assignee:** scaffolder agent

**Tasks:**
- [ ] Create `~/.claude/skills/mgrep/SKILL.md`
- [ ] Implement query parsing (extract path, flags)
- [ ] Implement mgrep invocation via Bash
- [ ] Implement fallback to grep
- [ ] Test happy path and error paths

**Acceptance:** `/mgrep "test"` returns results, fallback works.

---

### MGREP-003: codebase-search Integration

**Priority:** P0
**Estimate:** 3 hours
**Assignee:** go-pro agent

**Tasks:**
- [ ] Update agent.yaml with mgrep_integration section
- [ ] Update CLAUDE.md with tool selection guide
- [ ] Implement query classification (intent vs pattern)
- [ ] Implement fallback on mgrep failure
- [ ] Test with representative queries

**Acceptance:** Intent queries use mgrep, pattern queries use grep, fallback works.

---

### MGREP-004: Minimal Telemetry

**Priority:** P1
**Estimate:** 2 hours
**Assignee:** go-pro agent

**Tasks:**
- [ ] Add mgrep detection to gogent-sharp-edge
- [ ] Log mgrep invocations to ml-tool-events.jsonl
- [ ] Verify queryability of mgrep metrics

**Acceptance:** Can answer "How many mgrep calls this session?"

---

### MGREP-005: haiku-scout Integration

**Priority:** P1 (gated on Phase 1 success)
**Estimate:** 3 hours
**Assignee:** go-pro agent

**Tasks:**
- [ ] Add semantic phase to scout protocol
- [ ] Implement mgrep-scope.txt generation
- [ ] Update output schema with semantic_scope
- [ ] Implement fallback (skip semantic phase)
- [ ] Test routing recommendation accuracy

**Acceptance:** Scout produces semantic scope when mgrep available.

---

### MGREP-006: librarian Integration

**Priority:** P1 (gated on Phase 1 success)
**Estimate:** 2 hours
**Assignee:** go-pro agent

**Tasks:**
- [ ] Add internal-first research strategy
- [ ] Implement internal pattern detection
- [ ] Integrate with existing external search
- [ ] Test with library/best-practice queries

**Acceptance:** Librarian surfaces internal patterns before external search.

---

### MGREP-007: orchestrator Integration

**Priority:** P2 (gated on Phase 2 success)
**Estimate:** 3 hours
**Assignee:** go-pro agent

**Tasks:**
- [ ] Add disambiguation protocol
- [ ] Implement semantic scope assessment
- [ ] Implement grep verification step
- [ ] Test with ambiguous scope queries

**Acceptance:** Orchestrator disambiguates scope before spawning agents.

---

### MGREP-008: Configuration & Documentation

**Priority:** P2
**Estimate:** 2 hours
**Assignee:** tech-docs-writer agent

**Tasks:**
- [ ] Create .mgrepignore template
- [ ] Update CLAUDE.md with reference section
- [ ] Update ARCHITECTURE.md with external engines
- [ ] Update routing-schema.json with mgrep definition

**Acceptance:** Documentation complete, schema validates.

---

## Appendix: Decision Log

### D1: Why validation-first?

**Decision:** Require manual benchmarking before any code changes.

**Rationale:** Original spec assumed mgrep provides value. Staff Architect review identified this as untested assumption. Proving value first prevents wasted effort if mgrep doesn't deliver expected precision improvements.

### D2: Why one agent initially?

**Decision:** Integrate with codebase-search only in Phase 1.

**Rationale:** Reduces blast radius. If mgrep causes problems, only one agent affected. Also provides clean A/B comparison opportunity.

### D3: Why minimal telemetry?

**Decision:** Log to existing ml-tool-events.jsonl instead of new files.

**Rationale:** Original spec proposed 3 new telemetry files. This adds complexity without proven need. Minimal approach lets us measure what matters without building unused infrastructure.

### D4: Why gated phases?

**Decision:** Require measured improvement before expanding.

**Rationale:** Prevents sunk cost fallacy. If Phase 1 doesn't show improvement, we stop early rather than completing all 6 weeks of original plan.

---

**End of Action Plan**

*Ready for implementation. Start with MGREP-001 (Phase 0 Validation).*
