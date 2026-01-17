# Post-Implementation Review System Architecture

**Version:** 1.0.0
**Status:** Specification (Future Implementation for GO Migration)
**Created:** 2026-01-17
**Author:** Orchestrator (via conversation synthesis)
**Target Implementation:** GO-based gogent-fortress architecture

---

## Executive Summary

This document specifies a **two-agent post-implementation review system** for code quality assurance after changes are committed but before they are merged/deployed. Unlike pre-implementation review (staff-architect-critical-review), which validates plans, this system validates executed code.

**Key Architectural Decisions:**
- **TWO separate agents** (not a mode-based single agent)
- **compliance-reviewer**: Verificatory review with specs.md, bounded by git diff
- **architectural-digest**: Exploratory review without specs.md, scout-first architecture
- **Shared framework**: code-review-framework.md for Layers 7-8 (DRY principle)
- **Different sharp edges**: perfectionism vs scope_explosion
- **Integration**: Routing schema triggers, slash command skills, hook-based invocation

**Use Cases:**
1. **Compliance Review** (with plan): Verify implementation matches specs.md
2. **Architectural Digest** (without plan): Assess quality of ad-hoc changes
3. **Pull Request Guardrails**: Automated review before merge
4. **Post-commit Retrospective**: What was actually changed and why

---

## Problem Statement

### Current State

**Pre-implementation review exists** (staff-architect-critical-review):
- Reviews plans (specs.md) BEFORE implementation
- 7-layer framework (Assumptions → Dependencies → Failure Modes → etc.)
- Prevents architectural mistakes early
- BUT: No verification that implementation matches plan

**Post-implementation review does NOT exist**:
- Code is committed without quality verification
- Deviations from plan are undetected
- Ad-hoc changes lack architectural review
- Tech debt accumulates silently

### Gap Analysis

| Scenario | Current System | Desired State |
|----------|----------------|---------------|
| Planned implementation complete | "Assume it matches specs.md" | Verify compliance, flag deviations |
| Ad-hoc changes (no plan) | No review | Assess quality, detect anti-patterns |
| PR review needed | Manual only | Automated first-pass review |
| Implementation deviated from plan | Undetected | Flagged with explanation required |

### Requirements

**R1: Compliance Verification**
Given specs.md + git diff, verify implementation matches plan.

**R2: Quality Assessment**
Given git diff (no plan), assess code quality against conventions.

**R3: Scope Control**
Prevent review from expanding beyond changed files (scope explosion).

**R4: Cost Efficiency**
Keep reviews under $0.25 per invocation (scout-first architecture).

**R5: Actionable Output**
Generate critique with specific file:line references for issues.

**R6: Integration**
Work with existing routing schema, skills system, and hooks.

---

## Architecture Overview

### Two-Agent Design

```
┌─────────────────────────────────────────────────────────────┐
│                  Post-Implementation Review                  │
└─────────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┴─────────────────┐
        │                                   │
        ▼                                   ▼
┌──────────────────┐              ┌──────────────────┐
│ compliance-      │              │ architectural-   │
│ reviewer         │              │ digest           │
├──────────────────┤              ├──────────────────┤
│ INPUTS:          │              │ INPUTS:          │
│ - specs.md       │              │ - git diff       │
│ - git diff       │              │ - conventions    │
│                  │              │                  │
│ MODE:            │              │ MODE:            │
│ Verificatory     │              │ Exploratory      │
│                  │              │                  │
│ SCOPE:           │              │ SCOPE:           │
│ Bounded (diff)   │              │ Scout-controlled │
│                  │              │                  │
│ SHARP EDGE:      │              │ SHARP EDGE:      │
│ perfectionism    │              │ scope_explosion  │
└──────────────────┘              └──────────────────┘
        │                                   │
        └─────────────────┬─────────────────┘
                          ▼
                ┌──────────────────┐
                │ code-review-     │
                │ framework.md     │
                ├──────────────────┤
                │ Shared Layers:   │
                │ 7. Code Quality  │
                │ 8. Test Coverage │
                └──────────────────┘
```

### Why Two Agents?

**Option Evaluation** (from conversation):

| Architecture | Pros | Cons | Verdict |
|--------------|------|------|---------|
| Single agent with modes | Simple invocation | Mode confusion, sharp edge overlap | ❌ Rejected |
| **Separate agents** | Clear contracts, distinct sharp edges | More files | ✅ **Selected** |
| Pipeline (scout → review) | Explicit steps | Mandatory scout overhead | ❌ Rejected |
| Hybrid (single + specialist) | Best of both? | Complexity, unclear routing | ❌ Rejected |

**Decision Rationale:**
- **Compliance review** is fundamentally verificatory (check against known truth)
- **Architectural digest** is fundamentally exploratory (discover quality patterns)
- Different mindsets require different sharp edge mitigations
- Clear contracts reduce routing confusion
- Future extensibility: add specialized reviewers (security-reviewer, performance-reviewer)

---

## Agent Specifications

### Agent 1: compliance-reviewer

#### Role & Mindset

**Role:** Verify that implementation matches specs.md. You are a QA engineer comparing expected vs actual.

**Mindset:** "Trust but verify." The plan passed pre-implementation review. Your job is to confirm implementation fidelity and flag deviations.

**Inheritance:** Inherits philosophy from staff-architect-critical-review (adversarial but constructive), but adapted for post-implementation context.

#### Input Specification

**Required:**
- `specs.md` (the plan that was approved)
- `git diff` (the actual implementation)

**Optional:**
- `review-critique.md` (from pre-implementation review, for context)
- `scout_metrics.json` (from original exploration)
- User goal (to resolve ambiguities)

**Input Format (7-Section Delegation):**
```javascript
Task({
  description: "Compliance review: verify implementation matches specs.md",
  subagent_type: "Explore",  // Read-only review
  model: "sonnet",
  prompt: `AGENT: compliance-reviewer

1. TASK: Verify implementation in git diff matches specs.md plan

2. EXPECTED OUTCOME:
   - compliance-report.md with verdict (COMPLIANT | DEVIATIONS | CRITICAL_DEVIATIONS)
   - File-line mapping of deviations
   - Explanation requirement for each deviation

3. REQUIRED SKILLS:
   - 10-layer compliance framework (agent.md)
   - Diff analysis and plan correlation
   - Deviation classification (benign vs critical)

4. REQUIRED TOOLS:
   - Read (specs.md, git diff, review-critique.md)
   - Grep (search for expected patterns in diff)
   - Task (spawn haiku-scout if scope unclear, max 1)
   - Write (compliance-report.md, compliance-metadata.json)

5. MUST DO:
   - Read specs.md completely before analyzing diff
   - Map each phase/ticket to corresponding diff hunks
   - Quote specific diff lines for deviations
   - Classify deviations (benign, concern, critical)
   - Flag missing implementations (tickets not in diff)
   - Stay within 16K thinking budget
   - Respect 2-hour time box

6. MUST NOT DO:
   - Perfectionism: nitpicking style when plan allows flexibility (sharp-edge)
   - Scope expansion: reviewing files not in diff
   - Adding requirements not in specs.md
   - Marking deviations as critical without impact analysis
   - Exceeding 1 scout spawn

7. CONTEXT:
   Specs file: .claude/tmp/specs.md
   Diff: git diff main...feature-branch
   User goal: ${originalGoal}
   Pre-review critique: .claude/tmp/review-critique.md (if exists)
   Invocation: Post-commit verification`
})
```

#### Output Specification

**File 1: compliance-report.md**

```markdown
# Compliance Review: <Feature Name>

**Reviewed:** 2026-01-17T14:30:00Z
**Reviewer:** compliance-reviewer
**Specs:** .claude/tmp/specs.md
**Diff:** git diff main...feature-auth

---

## Executive Assessment

**Overall Verdict:** COMPLIANT | DEVIATIONS | CRITICAL_DEVIATIONS

**Deviation Counts:**
- Critical: <count> (must explain before merge)
- Concern: <count> (should explain)
- Benign: <count> (acceptable variance)

**Coverage:**
- Tickets implemented: <count> / <total>
- Missing tickets: <list>

**Summary:** <2-3 sentence assessment>

**Merge Recommendation:** APPROVE | APPROVE_WITH_EXPLANATION | HOLD

---

## Deviation Register

### Critical Deviations (Must Explain Before Merge)

| ID | Ticket | File:Line | Expected | Actual | Impact |
|----|--------|-----------|----------|--------|--------|
| CD-1 | Phase 2, Ticket 3 | auth.py:45 | OAuth2 | Custom token | Security risk |

**Detail for CD-1:**
```diff
Expected (from specs.md):
"Use oauthlib for OAuth2 token exchange (Phase 2, Ticket 3)"

Actual (from diff):
+ def custom_token_exchange():
+     # Manual token handling

Impact: Security vulnerability (custom crypto), fails compliance requirement.
Required: Explain why oauthlib was not used, or revert to plan.
```

### Concern Deviations (Should Explain)

[Same table format]

### Benign Deviations (Acceptable Variance)

[Same table format, brief entries]

---

## Coverage Analysis

### Tickets Implemented

| Phase | Ticket | Status | File(s) | Notes |
|-------|--------|--------|---------|-------|
| 1 | Setup database | ✅ Complete | db/schema.py | Matches plan |
| 2 | OAuth integration | ❌ Deviated | auth.py | See CD-1 |
| 3 | User endpoints | ✅ Complete | api/users.py | Matches plan |

### Missing Tickets

| Phase | Ticket | Expected File | Impact |
|-------|--------|---------------|--------|
| 2 | Add OAuth tests | tests/test_auth.py | Testing gap |

**Recommendation:** Implement missing tickets or update specs.md to reflect reduced scope.

---

## Compliance by Layer

### Layer 1: Plan Fidelity (70%)
- 7/10 tickets implemented as specified
- 2 deviations, 1 missing ticket

### Layer 2: Architectural Alignment (90%)
- Module structure matches plan
- Import dependencies correct

[Continue for all 10 layers]

---

## Commendations

1. Strong adherence to planned module structure
2. Error handling exceeds plan requirements (good deviation)
3. Documentation added (not required but valuable)

---

## Final Sign-Off

**Reviewed By:** compliance-reviewer
**Review Date:** 2026-01-17
**Review Duration:** 8 minutes
**Thinking Budget Used:** 12400 / 16000

**Conditions for Approval:**
- [ ] Explain CD-1 (OAuth deviation) or revert
- [ ] Implement missing test ticket or defer to Phase 4

**Recommended Actions:**
1. Address Critical deviations (CD-1)
2. Provide explanation for Concern deviations in PR description
3. Track Benign deviations in commit message for audit trail

**Post-Merge Monitoring:**
- Watch auth.py for security issues (custom token logic)
- Verify test coverage after missing ticket resolved
```

**File 2: compliance-metadata.json**

```json
{
  "review_id": "uuid",
  "timestamp": "2026-01-17T14:30:00Z",
  "specs_file": ".claude/tmp/specs.md",
  "diff_source": "git diff main...feature-auth",
  "verdict": "DEVIATIONS|COMPLIANT|CRITICAL_DEVIATIONS",
  "merge_recommendation": "APPROVE|APPROVE_WITH_EXPLANATION|HOLD",
  "deviation_counts": {
    "critical": 1,
    "concern": 2,
    "benign": 5
  },
  "coverage": {
    "tickets_implemented": 7,
    "tickets_total": 10,
    "tickets_missing": 1
  },
  "scouts_spawned": [
    {
      "agent": "haiku-scout",
      "target": "auth.py",
      "reason": "Verify OAuth implementation pattern",
      "duration_seconds": 5
    }
  ],
  "cost_estimate_usd": 0.18,
  "thinking_budget_used": 12400,
  "commendations_count": 3,
  "review_duration_minutes": 8
}
```

#### 10-Layer Compliance Framework

**Layer 1: Plan Fidelity**
- Do changed files match planned files in specs.md?
- Are tickets/phases addressed in order?
- Are acceptance criteria met per ticket?

**Layer 2: Architectural Alignment**
- Does module structure match plan?
- Are imports/dependencies as planned?
- Are abstraction boundaries respected?

**Layer 3: Deviation Classification**
For each deviation from plan:
- **Benign:** Style, naming, minor refactoring (acceptable)
- **Concern:** Different implementation approach, should explain (reviewable)
- **Critical:** Security, data model, API contract change (must explain or revert)

**Layer 4: Coverage Verification**
- Are all tickets from specs.md present in diff?
- Are there extra changes not in specs.md? (scope creep)
- Are phases implemented in planned order?

**Layer 5: Acceptance Criteria Validation**
For each ticket, check:
- Are success criteria met?
- Are edge cases handled?
- Are constraints satisfied?

**Layer 6: Regression Prevention**
- Do changes touch files not mentioned in plan?
- Are there unexpected deletions?
- Are existing tests still passing (if test results provided)?

**Layer 7: Code Quality** (from shared framework)
- [Delegated to code-review-framework.md]

**Layer 8: Test Coverage** (from shared framework)
- [Delegated to code-review-framework.md]

**Layer 9: Documentation Alignment**
- Are code comments consistent with plan?
- Is inline documentation sufficient for deviations?
- Are TODO markers used appropriately?

**Layer 10: Contractor Handoff**
If this were handed to another contractor tomorrow:
- Can they understand what was implemented?
- Is there a clear audit trail (commit messages, PR description)?
- Are deviations explained inline?

#### Sharp Edges

**Primary Sharp Edge: perfectionism**

```yaml
edges:
  - name: perfectionism
    severity: critical
    description: >
      Flagging every minor deviation from plan when plan allows implementation
      flexibility. Treating plan as rigid specification when it's guidance.
      Example: Plan says "use standard auth" → code uses django.contrib.auth →
      flagged as "not exactly oauthlib" when both satisfy requirement.
    mitigation: >
      Before marking deviation, check:
      1. Does plan REQUIRE this specific implementation?
      2. Or does plan specify OUTCOME and allow implementation choice?
      3. Quote exact requirement from specs.md
      4. If requirement is outcome-based, mark COMPLIANT if outcome met
      Only flag deviations that violate EXPLICIT requirements or security/architecture.
      When in doubt: benign deviation, not concern/critical.

  - name: scope_expansion_in_review
    severity: high
    description: >
      Reviewing files not in git diff. Searching entire codebase when diff
      shows 3 files changed. Plan mentions "auth module" → reviewing all 20
      files in auth/ when diff shows only 3 modified.
    mitigation: >
      ONLY review files in git diff output. Use Grep on diff output, not
      entire directory. If context needed from unchanged files, spawn scout
      with explicit scope (max 1 scout). Do not traverse beyond diff boundary.

  - name: adding_requirements_not_in_plan
    severity: high
    description: >
      Flagging "missing error handling" when specs.md doesn't require it.
      Treating personal best practices as plan requirements.
    mitigation: >
      Every deviation MUST quote specs.md requirement being violated.
      If you can't quote it, it's not a deviation (might be enhancement).
      Separate section: "Enhancement Opportunities" (not deviations).
```

#### Scout Integration

**When to Spawn Scout:**
- Diff references file not in specs.md (verify if planned)
- Implementation pattern unclear from diff alone
- Need to verify unchanged file context

**Scout Limit:** Maximum 1 scout per compliance review.

**Example Scout Call:**
```javascript
Task({
  description: "Verify OAuth pattern in auth.py",
  subagent_type: "Explore",
  model: "haiku",
  prompt: `AGENT: haiku-scout

1. TASK: Read auth.py and confirm OAuth implementation approach
2. EXPECTED OUTCOME: Report on OAuth library used, token flow
3. REQUIRED TOOLS: Read, Grep
4. MUST DO: Focus on OAuth implementation only
5. CONTEXT: Compliance review needs to verify deviation from plan`
})
```

#### Time Boxing

- **Small diff (<100 lines):** 5 minutes
- **Medium diff (100-500 lines):** 15 minutes
- **Large diff (500-2000 lines):** 30 minutes
- **Very large (2000+ lines):** 1 hour max OR spawn scout for triage

**If exceeding time box:** Output partial review with note, escalate to user.

#### Verdict Guidelines

| Verdict | Criteria | Merge Recommendation |
|---------|----------|----------------------|
| **COMPLIANT** | All tickets implemented as specified, 0 concern/critical deviations | APPROVE |
| **DEVIATIONS** | <3 concern deviations OR all benign | APPROVE_WITH_EXPLANATION |
| **CRITICAL_DEVIATIONS** | ≥1 critical deviation OR ≥3 concern deviations | HOLD (explain or revert) |

---

### Agent 2: architectural-digest

#### Role & Mindset

**Role:** Assess code quality and architectural patterns in ad-hoc changes (no plan). You are a senior engineer performing a retrospective code review.

**Mindset:** "What was actually built and is it maintainable?" You don't have a plan to compare against—discover what exists and evaluate quality.

**Inheritance:** Inherits adversarial-but-constructive philosophy from staff-architect-critical-review, but focuses on "as-built" quality rather than plan compliance.

#### Input Specification

**Required:**
- `git diff` (the changes to review)

**Optional:**
- Conventions file (e.g., `~/.claude/conventions/python.md`)
- Commit message (for intent understanding)
- Related file context (if scout discovers dependencies)

**NO specs.md** (by definition—this is for ad-hoc changes)

**Input Format (7-Section Delegation):**
```javascript
Task({
  description: "Architectural digest: assess quality of ad-hoc changes",
  subagent_type: "Explore",
  model: "sonnet",
  prompt: `AGENT: architectural-digest

1. TASK: Perform scout-first architectural review of git diff changes

2. EXPECTED OUTCOME:
   - architectural-digest.md with quality assessment
   - Pattern detection (good and bad)
   - Technical debt identification
   - Recommendations for follow-up work

3. REQUIRED SKILLS:
   - Scout-first architecture (spawn haiku-scout BEFORE deep review)
   - 10-layer quality framework (agent.md)
   - Pattern recognition (idioms, anti-patterns)
   - Convention alignment (language-specific)

4. REQUIRED TOOLS:
   - Task (spawn haiku-scout first, MANDATORY)
   - Read (diff, conventions, related files if scout recommends)
   - Grep (pattern detection)
   - Write (digest.md, metadata.json)

5. MUST DO:
   - SPAWN SCOUT FIRST (haiku-scout or gemini-scout per scout protocol)
   - Use scout scope_metrics to bound review (CRITICAL for scope control)
   - Apply all 10 layers in order
   - Quote specific diff lines for issues
   - Identify both good and bad patterns
   - Stay within 16K thinking budget
   - Respect scout-guided time box

6. MUST NOT DO:
   - scope_explosion: reviewing beyond scout-bounded scope (sharp-edge)
   - Skip scout (mandatory protection against runaway analysis)
   - Add requirements (you don't have a plan, evaluate what exists)
   - Infinite architecture loops (respect time box)
   - Nitpick style without identifying architectural issues

7. CONTEXT:
   Diff: git diff main...feature-branch
   Conventions: ~/.claude/conventions/${language}.md
   Commit message: "${commitMsg}"
   Invocation: Post-commit quality check (no plan available)

   SCOUT PROTOCOL:
   1. Spawn scout to assess scope (files, LoC, complexity)
   2. Use scout recommendation for review depth
   3. Bound review to scout-identified critical files`
})
```

#### Output Specification

**File 1: architectural-digest.md**

```markdown
# Architectural Digest: <Commit SHA or Feature>

**Reviewed:** 2026-01-17T15:00:00Z
**Reviewer:** architectural-digest
**Diff:** git diff main...feature-xyz
**Scope:** <scout-determined scope>

---

## Executive Assessment

**Overall Quality:** EXCELLENT | GOOD | ACCEPTABLE | CONCERNS | POOR

**Confidence Level:** HIGH | MEDIUM | LOW
- Rationale: <why this confidence>

**Pattern Counts:**
- Good Patterns: <count>
- Anti-Patterns: <count>
- Tech Debt Items: <count>

**Summary:** <2-3 sentence quality assessment>

**Recommendation:** MERGE | MERGE_WITH_FOLLOW_UP | REFACTOR_FIRST

---

## Scout Report Summary

**Scout Used:** haiku-scout (5 seconds, $0.001)

**Scope Metrics:**
- Files changed: 4
- Lines added: 320
- Lines deleted: 45
- Estimated complexity: Medium

**Scout Recommendation:** "Review focuses on 3 critical files (auth.py, models.py, api.py). Config changes in settings.py are benign boilerplate."

**Review Scope Bounded To:**
- auth.py (core logic)
- models.py (data model changes)
- api.py (interface changes)

[This section prevents scope_explosion by documenting scout-based boundaries]

---

## Pattern Analysis

### Good Patterns Identified

| Pattern | Location | Rationale |
|---------|----------|-----------|
| Dependency Injection | auth.py:23-45 | Testable, follows SOLID |
| Early Returns | api.py:67 | Reduces nesting, readable |
| Type Hints | All functions | Enables static analysis |

**Detail for Dependency Injection:**
```python
# auth.py:23-45
class AuthService:
    def __init__(self, token_store: TokenStore, validator: Validator):
        self.token_store = token_store
        self.validator = validator

This follows SOLID principles (D: dependency inversion). Makes testing easy
by allowing mock injection. Recommend: replicate in other services.
```

### Anti-Patterns Identified

| Pattern | Location | Severity | Impact |
|---------|----------|----------|--------|
| God Function | api.py:120-280 | High | Hard to test, maintain |
| Missing Error Handling | auth.py:89 | Medium | Silent failures |
| Magic Numbers | models.py:34 | Low | Unclear intent |

**Detail for God Function:**
```python
# api.py:120-280 (160 lines)
def process_user_request(request):
    # Validation
    # Authentication
    # Authorization
    # Business logic
    # Database access
    # Response formatting
    # Logging
    # ...

Recommendation: Split into:
1. validate_request()
2. authenticate()
3. execute_business_logic()
4. format_response()

Benefit: Each function testable in isolation, easier to reason about.
```

---

## Technical Debt Register

| ID | Item | Location | Effort | Priority |
|----|------|----------|--------|----------|
| TD-1 | Refactor God Function | api.py:120-280 | 2 hours | High |
| TD-2 | Add error handling | auth.py:89 | 30 min | Medium |
| TD-3 | Extract magic numbers | models.py:34 | 15 min | Low |

**Estimated Debt:** 2.75 hours (if addressed now) vs 8+ hours (if deferred 6 months)

---

## Quality by Layer

### Layer 1: Scope Assessment (from Scout)
- Files: 4 (bounded)
- LoC: 320 added, 45 deleted
- Complexity: Medium

### Layer 2: Convention Alignment
- ✅ Follows python.md conventions (type hints, docstrings)
- ✅ PEP 8 compliant
- ⚠️ Some functions exceed 50 LoC (refactor opportunity)

[Continue for all 10 layers]

---

## Recommendations

### High Priority (Address Before Merge or Next Sprint)
1. **Refactor God Function (TD-1):** Split api.py:120-280 into 4 functions
2. **Add Error Handling (TD-2):** auth.py:89 needs try/except for token validation

### Medium Priority (Address This Quarter)
3. **Extract Magic Numbers (TD-3):** models.py:34 use named constant

### Low Priority (Defer to Maintenance Window)
4. Document complex regex in auth.py:45
5. Add logging to edge case handlers

---

## Commendations

1. **Excellent Type Hints:** All functions properly annotated
2. **Good Test Coverage:** 85% coverage on new code (exceeds baseline)
3. **Dependency Injection:** Modern pattern usage in auth.py
4. **Clear Commit Message:** Intent well-documented

---

## Follow-Up Work Suggested

**Create Follow-Up Tickets:**
1. Refactor api.py God Function (TD-1) - 2 hours
2. Security audit of custom token logic - 1 hour
3. Performance benchmark for auth flow - 30 min

**Potential Future Enhancements:**
- Consider caching for token validation (if auth becomes bottleneck)
- Add rate limiting to API endpoints (if user-facing)

---

## Final Sign-Off

**Reviewed By:** architectural-digest
**Review Date:** 2026-01-17
**Review Duration:** 12 minutes (scout: 5s, analysis: 11m)
**Thinking Budget Used:** 13800 / 16000
**Scout Cost:** $0.001
**Total Cost:** $0.19

**Merge Recommendation:** MERGE_WITH_FOLLOW_UP

**Conditions:**
- Document TD-1, TD-2 as follow-up tickets
- No blocking issues (code is functional)
- Technical debt tracked, not ignored

**Post-Merge Monitoring:**
- Watch api.py:120-280 for bug reports (complexity risk)
- Benchmark auth flow performance after 1 week
```

**File 2: architectural-digest-metadata.json**

```json
{
  "review_id": "uuid",
  "timestamp": "2026-01-17T15:00:00Z",
  "diff_source": "git diff main...feature-xyz",
  "conventions_file": "~/.claude/conventions/python.md",
  "quality_verdict": "GOOD|ACCEPTABLE|CONCERNS|etc",
  "confidence": "HIGH|MEDIUM|LOW",
  "pattern_counts": {
    "good_patterns": 3,
    "anti_patterns": 3,
    "tech_debt_items": 3
  },
  "scope_metrics": {
    "files_changed": 4,
    "lines_added": 320,
    "lines_deleted": 45,
    "scout_bounded_files": 3
  },
  "scout_data": {
    "agent": "haiku-scout",
    "duration_seconds": 5,
    "cost_usd": 0.001,
    "recommendation": "Medium complexity, focus on 3 files"
  },
  "cost_estimate_usd": 0.19,
  "thinking_budget_used": 13800,
  "commendations_count": 4,
  "review_duration_minutes": 12,
  "merge_recommendation": "MERGE|MERGE_WITH_FOLLOW_UP|REFACTOR_FIRST"
}
```

#### 10-Layer Quality Framework

**Layer 0: Scout-First Scope Assessment** (MANDATORY)
- Spawn haiku-scout or gemini-scout (per scout_protocol in routing-schema.json)
- Obtain scope_metrics: files, LoC, complexity estimate
- Use scout recommendation to bound review depth
- Document scout findings in digest

**Selection Logic** (from routing-schema.json):
```javascript
if (files_changed <= 3 && estimated_tokens <= 5000) {
  scout = "haiku-scout"  // Lower latency
} else {
  scout = "gemini-scout"  // Massive context window, lower cost
}
```

**Layer 1: Convention Alignment**
- Does code follow language conventions (python.md, go.md, R.md)?
- Are style guidelines respected?
- Are naming conventions consistent?

**Layer 2: Pattern Recognition**
- What design patterns are used? (Dependency Injection, Factory, Strategy, etc.)
- Are patterns appropriate for the context?
- Are there anti-patterns? (God Object, Circular Dependencies, etc.)

**Layer 3: Architectural Coherence**
- Does code fit into existing architecture?
- Are abstraction boundaries clear?
- Is coupling reasonable?

**Layer 4: Error Handling & Edge Cases**
- Are errors handled appropriately?
- Are edge cases considered?
- Is there defensive programming?

**Layer 5: Security Posture**
- Any obvious security issues? (SQL injection, XSS, auth bypass)
- Sensitive data handling appropriate?
- Input validation present?

**Layer 6: Performance Characteristics**
- Any obvious performance issues? (N+1 queries, unbounded loops)
- Appropriate data structures?
- Caching where beneficial?

**Layer 7: Code Quality** (from shared framework)
- [Delegated to code-review-framework.md]

**Layer 8: Test Coverage** (from shared framework)
- [Delegated to code-review-framework.md]

**Layer 9: Technical Debt Assessment**
- What shortcuts were taken?
- What should be refactored?
- What's the debt cost vs payoff timeline?

**Layer 10: Maintainability**
- Can another engineer understand this in 6 months?
- Is it documented sufficiently?
- Is it testable?

#### Sharp Edges

**Primary Sharp Edge: scope_explosion**

```yaml
edges:
  - name: scope_explosion
    severity: critical
    description: >
      Reviewing entire codebase when diff shows 3 files changed. Starting with
      git diff, then "discovering" 20 related files, then reading entire module,
      then analyzing architectural patterns across system. What started as
      "review this change" becomes "architectural audit of entire system."
      Cost: $0.20 → $2.50. Time: 10min → 2 hours.
    mitigation: >
      MANDATORY scout-first architecture:
      1. SPAWN SCOUT FIRST (before any analysis)
      2. Scout provides scope_metrics (files, LoC, complexity)
      3. BOUND REVIEW to scout-identified critical files (document in digest)
      4. If context needed from related files, scout ALREADY identified them
      5. Do NOT traverse beyond scout boundary
      6. Time box based on scout complexity estimate:
         - Low complexity: 5 min
         - Medium complexity: 15 min
         - High complexity: 30 min
      If scout says "3 critical files", review ONLY those 3. Not 3, then 5, then 10.

  - name: architecture_perfectionism
    severity: high
    description: >
      Flagging every minor deviation from ideal architecture when code is
      functional and maintainable. "This could use dependency injection" when
      simple function works fine. Treating every pattern as anti-pattern.
    mitigation: >
      Context matters:
      - God Function in 10-file module? Flag it. (real issue)
      - 50-line function in 2-file script? Benign. (not worth refactoring)
      - Missing error handling on critical path? Flag. (security/reliability)
      - Missing error handling on log formatting? Benign. (low impact)
      Before flagging anti-pattern, assess:
      1. What's the actual impact if not fixed?
      2. Is this code on critical path?
      3. Is this a 2-file script or 200-file system?
      If impact is low and code is maintainable, note as "Enhancement" not "Anti-Pattern."

  - name: missing_positive_patterns
    severity: medium
    description: >
      Only flagging problems, never commending good practices. Review feels
      adversarial instead of constructive. Missed opportunity to reinforce
      good patterns for future work.
    mitigation: >
      FORCE YOURSELF to find 3+ commendations before finishing review.
      Good patterns to watch for:
      - Type hints, docstrings, tests
      - Dependency injection, SOLID principles
      - Error handling, input validation
      - Clear naming, appropriate abstractions
      If you can't find 3 good things, re-read the code (you missed them).
      Ratio: Aim for 1:1 or better (commendations : issues).
```

#### Scout Integration (MANDATORY)

**Scout-First Architecture:**

```javascript
// Step 1: ALWAYS spawn scout first
const scoutTask = Task({
  description: "Assess scope of changes for architectural review",
  subagent_type: "Explore",
  model: "haiku",  // or gemini per protocol
  prompt: `AGENT: haiku-scout

1. TASK: Analyze git diff to determine review scope
2. EXPECTED OUTCOME: scope_metrics (files, LoC, complexity), critical file list
3. REQUIRED TOOLS: Read (diff), Grep
4. MUST DO: Identify which files are critical vs boilerplate
5. CONTEXT: Bounding architectural-digest review to prevent scope explosion`
})

// Step 2: Parse scout output
const scoutData = JSON.parse(scoutTask.output)
const criticalFiles = scoutData.critical_files  // e.g., ["auth.py", "models.py"]
const complexity = scoutData.complexity  // "Low", "Medium", "High"

// Step 3: Bound review to scout scope
// ONLY review files in criticalFiles list
// Set time box based on complexity:
const timeBox = {
  "Low": 5,
  "Medium": 15,
  "High": 30
}[complexity]

// Step 4: Document scout in digest
// Include scout findings in "Scout Report Summary" section
```

**Scout Protocols** (from routing-schema.json):

| Scout | Max Files | Cost | When to Use |
|-------|-----------|------|-------------|
| haiku-scout | 3 files, <5K tokens | $0.001 | Small diffs, low latency needed |
| gemini-scout | 4+ files, 5K+ tokens | $0.001 | Larger diffs, offload context |

**Scout Output Schema:**
```json
{
  "scope_metrics": {
    "total_files": 4,
    "total_lines": 365,
    "estimated_tokens": 8000
  },
  "critical_files": ["auth.py", "models.py", "api.py"],
  "boilerplate_files": ["settings.py"],
  "complexity": "Medium",
  "routing_recommendation": {
    "review_depth": "Standard",
    "time_box_minutes": 15,
    "focus_areas": ["auth logic", "data model changes"]
  }
}
```

#### Time Boxing

**Scout-Guided Time Boxes:**
- **Low complexity:** 5 minutes
- **Medium complexity:** 15 minutes
- **High complexity:** 30 minutes
- **Very high complexity:** 1 hour OR escalate to user ("scope too large for ad-hoc review")

**If Exceeding Time Box:**
1. Output partial digest with scope note
2. Recommend splitting review into multiple focused reviews
3. Suggest creating formal plan (specs.md) and using compliance-reviewer instead

#### Quality Verdict Guidelines

| Verdict | Criteria | Merge Recommendation |
|---------|----------|----------------------|
| **EXCELLENT** | 3+ good patterns, 0 anti-patterns, conventions followed | MERGE |
| **GOOD** | 2+ good patterns, <2 anti-patterns, minor issues | MERGE |
| **ACCEPTABLE** | Functional code, some debt, no critical issues | MERGE_WITH_FOLLOW_UP |
| **CONCERNS** | 2+ anti-patterns OR 1 security issue OR conventions violated | REFACTOR_FIRST (or track debt) |
| **POOR** | Multiple critical issues, security holes, unmaintainable | REFACTOR_REQUIRED |

---

## Shared Framework: code-review-framework.md

### Purpose

**DRY Principle:** Layers 7-8 are identical across compliance-reviewer and architectural-digest. Extract to shared framework to avoid duplication.

**Scope:** Code quality and test coverage analysis (language-agnostic patterns)

### Layer 7: Code Quality

**Analyze:**

1. **Readability**
   - Clear variable/function names?
   - Appropriate comment density (not too much, not too little)?
   - Nesting depth reasonable (<4 levels)?

2. **Complexity**
   - Cyclomatic complexity per function?
   - Functions under 50 LoC (guideline, not rule)?
   - Single Responsibility Principle respected?

3. **Duplication**
   - Copy-paste code?
   - Opportunities for extraction/abstraction?

4. **Documentation**
   - Docstrings for public APIs?
   - Complex logic explained?
   - TODO/FIXME markers appropriate?

**Red Flags:**
- Function >100 LoC (God Function candidate)
- Nesting >4 levels (refactor for early returns)
- Same code block repeated 3+ times (extract function)
- No comments on complex regex/algorithm

**Output Format:**
```markdown
### Layer 7: Code Quality

**Readability:** GOOD | ACCEPTABLE | POOR
- Clear naming conventions throughout
- Appropriate comment density

**Complexity:** GOOD | ACCEPTABLE | POOR
- api.py:120-280 exceeds complexity threshold (God Function)
- Other functions within reasonable bounds

**Duplication:** MINIMAL | SOME | EXCESSIVE
- Minor duplication in error handling (acceptable)

**Documentation:** EXCELLENT | ADEQUATE | INSUFFICIENT
- All public APIs documented
- Complex auth flow explained
```

### Layer 8: Test Coverage

**Analyze:**

1. **Unit Test Presence**
   - Are there tests for new/changed code?
   - Do tests cover happy path and edge cases?
   - Are tests meaningful (not just assertion-free smoke tests)?

2. **Test Quality**
   - Arrange-Act-Assert structure?
   - Clear test names (describe what's being tested)?
   - Mocking used appropriately?

3. **Coverage Metrics** (if available)
   - Line coverage percentage?
   - Branch coverage?
   - Critical paths covered?

4. **Integration Tests**
   - Are component boundaries tested?
   - Are external dependencies mocked?

**Red Flags:**
- No tests for new business logic
- Tests with no assertions (smoke tests disguised as unit tests)
- Tests coupled to implementation details (brittle)
- Missing tests for error paths

**Output Format:**
```markdown
### Layer 8: Test Coverage

**Unit Tests:** EXCELLENT | ADEQUATE | INSUFFICIENT | MISSING
- auth.py: 85% coverage, good edge case handling
- api.py: 40% coverage, missing God Function tests (blocker)

**Test Quality:** GOOD | ACCEPTABLE | POOR
- Clear test names (test_auth_with_invalid_token)
- Proper mocking of external services

**Integration Tests:** PRESENT | MISSING
- Missing: end-to-end test for full auth flow
- Recommend: Add integration test as follow-up ticket

**Overall Coverage:** 68% (baseline: 70%)
- Recommendation: Add tests for api.py God Function before merge
```

### Usage in Agents

**In compliance-reviewer:**
```markdown
### Layer 7: Code Quality
[Include output from code-review-framework.md Layer 7]

### Layer 8: Test Coverage
[Include output from code-review-framework.md Layer 8]
Compare against specs.md test requirements:
- Specs required: Unit tests for all endpoints
- Actual: 85% coverage on endpoints
- Gap: Missing tests for error handling in api.py:240
```

**In architectural-digest:**
```markdown
### Layer 7: Code Quality
[Include output from code-review-framework.md Layer 7]

### Layer 8: Test Coverage
[Include output from code-review-framework.md Layer 8]
Compare against conventions (python.md):
- Convention baseline: 70% coverage
- Actual: 68% coverage
- Recommendation: Add 2% coverage via api.py tests
```

---

## Routing Integration

### Trigger Phrases

**For compliance-reviewer:**
- "review this implementation" (if specs.md exists)
- "verify implementation matches plan"
- "compliance check"
- "did we follow the plan"
- "implementation review" (if specs.md in context)

**For architectural-digest:**
- "review this code" (if NO specs.md)
- "code quality check"
- "what changed" (exploratory)
- "assess this implementation" (no plan reference)
- "architectural review" (ad-hoc context)

### Routing Logic (for routing-schema.json)

```json
{
  "patterns": {
    "post_implementation_review": {
      "triggers": [
        "review implementation",
        "compliance check",
        "code quality",
        "assess changes",
        "what changed",
        "review this code"
      ],
      "routing_logic": {
        "if_specs_md_exists": "compliance-reviewer",
        "if_no_specs_md": "architectural-digest"
      }
    }
  }
}
```

### Subagent Type Mapping

**For routing-schema.json `agent_subagent_mapping`:**
```json
{
  "compliance-reviewer": "Explore",
  "architectural-digest": "Explore"
}
```

**Rationale:** Both agents are read-only reviewers. They analyze code but don't modify it.

### Skills Integration

#### Skill: /review-implementation

**Purpose:** Automatic routing to correct review agent based on specs.md presence.

**Invocation:**
- `/review-implementation` (auto-detect mode)
- `/review-implementation --compliance` (force compliance-reviewer)
- `/review-implementation --digest` (force architectural-digest)

**Implementation:**
```javascript
// Phase 1: Detect mode
const specsExists = fileExists(".claude/tmp/specs.md")
const mode = args.includes("--compliance") ? "compliance"
           : args.includes("--digest") ? "digest"
           : specsExists ? "compliance" : "digest"

// Phase 2: Get diff
const diff = Bash({command: "git diff main...HEAD"})

// Phase 3: Route to agent
if (mode === "compliance") {
  Task({
    description: "Compliance review of implementation",
    subagent_type: "Explore",
    model: "sonnet",
    prompt: `AGENT: compliance-reviewer
    [7-section prompt as specified above]`
  })
} else {
  Task({
    description: "Architectural digest of changes",
    subagent_type: "Explore",
    model: "sonnet",
    prompt: `AGENT: architectural-digest
    [7-section prompt as specified above]`
  })
}

// Phase 4: Present results
// [Similar to /review-plan skill output]
```

#### Skill: /review-pr

**Purpose:** PR review integration (webhook or manual).

**Usage:**
```bash
/review-pr 123  # Review PR #123
```

**Implementation:**
```javascript
// Fetch PR diff via gh CLI
const prDiff = Bash({command: `gh pr diff ${prNumber}`})
const prDescription = Bash({command: `gh pr view ${prNumber} --json body -q .body`})

// Check if PR mentions specs.md or ticket ID
const hasSpecs = prDescription.includes("Implements specs.md") || prDescription.includes("GOgent-")

// Route accordingly
const agent = hasSpecs ? "compliance-reviewer" : "architectural-digest"

// Post review as PR comment
const review = Task({...})
Bash({command: `gh pr comment ${prNumber} --body "${review}"`})
```

### Hook Integration

#### Hook: post-commit

**Purpose:** Automatic review after commit (optional, user-configurable).

**Location:** `.git/hooks/post-commit`

```bash
#!/bin/bash
# Auto-review on commit (if enabled)

if [ -f ".claude/config/auto-review.enabled" ]; then
  claude-cli /review-implementation
fi
```

**Configuration:**
```bash
# Enable auto-review
touch .claude/config/auto-review.enabled

# Disable auto-review
rm .claude/config/auto-review.enabled
```

#### Hook: pre-push

**Purpose:** Block push if critical issues found (optional guardrail).

**Location:** `.git/hooks/pre-push`

```bash
#!/bin/bash
# Block push if critical review issues

if [ -f ".claude/config/review-gate.enabled" ]; then
  review=$(claude-cli /review-implementation --json)
  critical_count=$(echo "$review" | jq '.deviation_counts.critical // .pattern_counts.anti_patterns')

  if [ "$critical_count" -gt 0 ]; then
    echo "ERROR: $critical_count critical issues found"
    echo "Run: /review-implementation to see details"
    echo "Override: git push --no-verify"
    exit 1
  fi
fi
```

---

## Cost Analysis

### Per-Review Costs

**compliance-reviewer:**
| Component | Model | Tokens | Cost |
|-----------|-------|--------|------|
| Review agent | Sonnet+16K | ~15K | $0.14 |
| Scout (if spawned) | Haiku+2K | ~2K | $0.01 |
| **Total** | | | **$0.15** |

**architectural-digest:**
| Component | Model | Tokens | Cost |
|-----------|-------|--------|------|
| Scout (mandatory) | Haiku | ~1K | $0.001 |
| Review agent | Sonnet+16K | ~14K | $0.13 |
| **Total** | | | **$0.13** |

**Note:** architectural-digest is cheaper because scout is lightweight Haiku (not thinking), and scout-first reduces main agent token usage.

### Comparison to Pre-Implementation Review

| Review Type | Cost | When Used | Output |
|-------------|------|-----------|--------|
| Pre-implementation (staff-architect) | $0.15-0.17 | Before coding | Review plan, prevent mistakes |
| Compliance (post-implementation) | $0.15 | After coding, with plan | Verify compliance, flag deviations |
| Digest (post-implementation) | $0.13 | After coding, no plan | Assess quality, identify debt |

**ROI Analysis:**
- Pre-review prevents $2-10 in rework (estimate)
- Post-compliance catches deviations before merge (prevents $5-20 in bug fixes)
- Post-digest identifies debt before it compounds (prevents $10-50 in future refactoring)

---

## Implementation Checklist (GO Migration)

### Phase 1: Foundation (Week 1-2)

**Files to Create:**
- [ ] `.claude/agents/compliance-reviewer/agent.md` (full specification from this doc)
- [ ] `.claude/agents/compliance-reviewer/sharp-edges.yaml` (perfectionism, scope_expansion, etc.)
- [ ] `.claude/agents/architectural-digest/agent.md` (full specification)
- [ ] `.claude/agents/architectural-digest/sharp-edges.yaml` (scope_explosion, etc.)
- [ ] `.claude/frameworks/code-review-framework.md` (Layers 7-8 shared)

**Agent Structure:**
```
.claude/agents/
├── compliance-reviewer/
│   ├── agent.md           # Role, mindset, 10-layer framework
│   ├── sharp-edges.yaml   # perfectionism, scope_expansion_in_review, etc.
│   └── examples/
│       ├── example-compliance-report.md
│       └── example-compliance-metadata.json
├── architectural-digest/
│   ├── agent.md           # Role, mindset, scout-first, 10-layer framework
│   ├── sharp-edges.yaml   # scope_explosion, architecture_perfectionism, etc.
│   └── examples/
│       ├── example-digest.md
│       └── example-digest-metadata.json
```

**Dependencies:**
- Existing: `haiku-scout` agent (already exists)
- Existing: `routing-schema.json` (update with new agents)
- New: `code-review-framework.md` (Layers 7-8 DRY extraction)

### Phase 2: Routing Integration (Week 2-3)

**Update routing-schema.json:**
- [ ] Add compliance-reviewer to `agent_subagent_mapping` → `"Explore"`
- [ ] Add architectural-digest to `agent_subagent_mapping` → `"Explore"`
- [ ] Add triggers to `patterns` section (see "Routing Integration" above)
- [ ] Add cost thresholds: `post_implementation_review_max_cost: 0.25`

**Update CLAUDE.md Key Triggers Table:**
- [ ] Add row: `review implementation, compliance check, code quality | compliance-reviewer or architectural-digest | NO`

**Test Routing:**
```bash
# Test 1: With specs.md (should route to compliance-reviewer)
echo "review this implementation" | claude-routing-test

# Test 2: Without specs.md (should route to architectural-digest)
rm .claude/tmp/specs.md
echo "code quality check" | claude-routing-test
```

### Phase 3: Skills Creation (Week 3-4)

**Create /review-implementation skill:**
- [ ] `.claude/skills/review-implementation/SKILL.md` (full specification from this doc)
- [ ] Implement auto-detection logic (specs.md presence)
- [ ] Implement mode flags (--compliance, --digest)
- [ ] Implement output presentation (similar to /review-plan)

**Create /review-pr skill:**
- [ ] `.claude/skills/review-pr/SKILL.md`
- [ ] Integrate with gh CLI for PR diff fetching
- [ ] Implement PR comment posting
- [ ] Handle PR description parsing for specs.md detection

**Test Skills:**
```bash
# Manual test
/review-implementation
/review-implementation --compliance
/review-implementation --digest

# PR test
/review-pr 123
```

### Phase 4: Hook Integration (Week 4-5)

**Create optional hooks:**
- [ ] `.claude/hooks/post-commit-review.sh` (optional auto-review)
- [ ] `.claude/hooks/pre-push-gate.sh` (optional critical issue blocker)
- [ ] Configuration files: `.claude/config/auto-review.enabled`, `.claude/config/review-gate.enabled`

**Test Hooks:**
```bash
# Enable auto-review
touch .claude/config/auto-review.enabled
git commit -m "test commit"
# Should trigger /review-implementation automatically

# Enable push gate
touch .claude/config/review-gate.enabled
# Introduce critical issue, try to push
# Should block with error message
```

### Phase 5: Documentation & Examples (Week 5-6)

**Create examples:**
- [ ] `.claude/agents/compliance-reviewer/examples/example-compliance-report.md` (full example)
- [ ] `.claude/agents/compliance-reviewer/examples/example-compliance-metadata.json`
- [ ] `.claude/agents/architectural-digest/examples/example-digest.md` (full example)
- [ ] `.claude/agents/architectural-digest/examples/example-digest-metadata.json`

**Update global docs:**
- [ ] Update `.claude/docs/agent-reference-table.md` with new agents
- [ ] Update `.claude/docs/workflows/review-workflow.md` (pre + post review flow)
- [ ] Create `.claude/docs/guides/post-implementation-review-guide.md` (user guide)

**Create user guide structure:**
```markdown
# Post-Implementation Review Guide

## When to Use

- **After Implementation:** Code is written, tests pass, ready for review
- **Before PR Merge:** Quality gate before merging to main
- **Ad-Hoc Changes:** Quick fixes without formal plans

## Which Agent to Use

| Scenario | Agent | Command |
|----------|-------|---------|
| Implementation with specs.md | compliance-reviewer | /review-implementation |
| Ad-hoc change without plan | architectural-digest | /review-implementation |
| Pull request review | Auto-detected | /review-pr <number> |

[Continue with examples, cost breakdown, troubleshooting, etc.]
```

### Phase 6: Testing & Validation (Week 6)

**Integration Tests:**
- [ ] Test compliance-reviewer with matching implementation (should pass)
- [ ] Test compliance-reviewer with deviating implementation (should flag)
- [ ] Test architectural-digest with good code (should commend)
- [ ] Test architectural-digest with anti-patterns (should flag)
- [ ] Test scout-first scope control (should not expand beyond scout boundary)

**Sharp Edge Validation:**
- [ ] Verify perfectionism mitigation (compliance-reviewer shouldn't nitpick)
- [ ] Verify scope_explosion mitigation (architectural-digest stays bounded)
- [ ] Verify missing_positive_patterns mitigation (both agents commend good code)

**Cost Validation:**
- [ ] Measure actual costs across 10 sample reviews
- [ ] Verify scout overhead is <$0.01 per review
- [ ] Confirm total cost stays under $0.25 threshold

### Phase 7: GO Migration Specifics (Week 7-8)

**Language-Agnostic Considerations:**
- [ ] Ensure framework references conventions dynamically (python.md, go.md, R.md)
- [ ] Test with GO code samples (verify language detection works)
- [ ] Update examples to include GO-specific patterns (goroutines, channels, errgroup)

**GO-Specific Patterns to Detect:**
- [ ] Good: Early returns, error wrapping, context propagation
- [ ] Anti-patterns: Goroutine leaks, missing context cancellation, ignored errors

**Test GO Integration:**
```bash
# Create sample GO diff
git checkout -b test-go-review
# ... make GO changes ...
git add .
git commit -m "test GO changes"

# Run review
/review-implementation

# Verify:
# - GO conventions loaded (~/.claude/conventions/go.md)
# - GO-specific patterns detected
# - Output references GO idioms
```

---

## Dependencies & Order

### Dependency Graph

```
┌─────────────────────────────────────────────┐
│ Phase 1: Foundation                         │
│ (agent.md files, sharp-edges.yaml)          │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 2: Routing Integration                │
│ (routing-schema.json, CLAUDE.md updates)    │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 3: Skills Creation                    │
│ (/review-implementation, /review-pr)        │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 4: Hook Integration (optional)        │
│ (post-commit, pre-push)                     │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 5: Documentation & Examples           │
│ (guides, examples, reference docs)          │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 6: Testing & Validation               │
│ (integration tests, cost validation)        │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│ Phase 7: GO Migration Specifics             │
│ (GO patterns, GO examples, GO testing)      │
└─────────────────────────────────────────────┘
```

### Critical Path

**Minimum Viable Implementation:**
1. Phase 1: Foundation (Week 1-2) - REQUIRED
2. Phase 2: Routing Integration (Week 2-3) - REQUIRED
3. Phase 3: /review-implementation skill (Week 3) - REQUIRED
4. Phase 6: Basic testing (Week 4) - REQUIRED

**Total MVP Time:** 4 weeks

**Full Implementation:**
- All 7 phases: 8 weeks

---

## Critical Files for Implementation

Based on this specification, here are the 5 most critical files for implementing the post-implementation review system in GO:

1. **`.claude/agents/compliance-reviewer/agent.md`** - Core compliance review agent specification with 10-layer framework, sharp edges (perfectionism), and verdict guidelines. This is the verificatory review workhorse.

2. **`.claude/agents/architectural-digest/agent.md`** - Core architectural digest agent specification with scout-first architecture, 10-layer quality framework, and scope explosion mitigation. This is the exploratory review workhorse.

3. **`.claude/frameworks/code-review-framework.md`** - Shared Layers 7-8 (Code Quality & Test Coverage) used by both agents. DRY principle prevents duplication and ensures consistent quality assessment.

4. **`routing-schema.json`** - Integration point for routing triggers, subagent_type mappings, and cost thresholds. Must be updated to route "review implementation" requests to correct agent based on specs.md presence.

5. **`.claude/skills/review-implementation/SKILL.md`** - Primary user-facing skill that auto-detects mode (compliance vs digest), invokes correct agent, and presents results. This is how users interact with the system.

**Implementation Note:** These files contain ALL architectural decisions, framework specifications, sharp edge mitigations, and integration patterns from the conversation. No conversation context is required - the specifications are complete and implementation-ready for the GO migration.
