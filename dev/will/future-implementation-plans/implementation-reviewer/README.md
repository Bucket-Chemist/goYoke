# Post-Implementation Review System

**Status:** Future Implementation (Specification Complete)
**Created:** 2026-01-17
**Target:** GO-based gogent-fortress architecture

---

## Overview

This directory contains the complete specification for a **two-agent post-implementation review system** that validates code quality after implementation but before merge/deploy.

**Key Features:**
- **compliance-reviewer**: Verificatory review comparing implementation to specs.md
- **architectural-digest**: Exploratory review assessing ad-hoc changes without a plan
- **Scout-first architecture**: Prevents scope explosion in exploratory reviews
- **Shared framework**: DRY principle for code quality and test coverage layers

---

## What's In This Directory

### ARCHITECTURE.md (MAIN DOCUMENT)

**85KB comprehensive specification** containing:

1. **Problem Statement & Requirements** - Why this system is needed, what gaps it fills
2. **Architecture Overview** - Two-agent design rationale, option evaluation
3. **Complete Agent Specifications**:
   - compliance-reviewer: Role, mindset, 10-layer framework, input/output specs, sharp edges
   - architectural-digest: Role, mindset, scout-first protocol, 10-layer framework, sharp edges
4. **Shared Framework** - code-review-framework.md specification (Layers 7-8)
5. **Routing Integration** - Trigger phrases, routing logic, subagent_type mappings
6. **Skills Integration** - /review-implementation and /review-pr specifications
7. **Hook Integration** - Optional post-commit and pre-push hooks
8. **Cost Analysis** - Per-review costs, ROI estimates
9. **Implementation Checklist** - 7-phase implementation plan (8 weeks full, 4 weeks MVP)
10. **Critical Files List** - Top 5 files needed for implementation

**This document is COMPLETE and IMPLEMENTATION-READY.**

---

## Quick Start (For Implementer)

### Phase 1: Read ARCHITECTURE.md (1 hour)

Read the entire specification to understand:
- Why two agents are required (not a single mode-based agent)
- How scout-first architecture prevents scope explosion
- What sharp edges exist and how to mitigate them
- How routing integration works

### Phase 2: Implement Foundation (Week 1-2)

Create these 5 critical files (extract from ARCHITECTURE.md):

1. `.claude/agents/compliance-reviewer/agent.md`
2. `.claude/agents/architectural-digest/agent.md`
3. `.claude/frameworks/code-review-framework.md`
4. Update `routing-schema.json` with new agents
5. Create `/review-implementation` skill

### Phase 3: Test & Validate (Week 3-4)

- Test compliance-reviewer with matching/deviating implementations
- Test architectural-digest with good/bad code
- Verify scout-first scope control works
- Validate costs stay under $0.25/review

### Phase 4: GO-Specific Adaptation (Week 5-6)

- Add GO-specific patterns (goroutines, channels, errgroup)
- Test with GO code samples
- Update examples for GO idioms

---

## Architecture Decisions (Key Highlights)

### Why Two Agents?

**Compliance Review** (with specs.md):
- Verificatory mindset ("Did we build what we said we'd build?")
- Bounded scope (git diff only)
- Binary assessment (matches plan or doesn't)
- Sharp edge: perfectionism (nitpicking when plan allows flexibility)

**Architectural Digest** (without specs.md):
- Exploratory mindset ("What was built and is it maintainable?")
- Unbounded scope (requires scout-first protection)
- Subjective assessment (quality evaluation against conventions)
- Sharp edge: scope_explosion (reviewing entire codebase instead of changed files)

**These are fundamentally different use cases requiring different agent contracts.**

### Why Scout-First for Architectural Digest?

Without specs.md, architectural digest has **no natural scope boundary**.

**Problem:**
```
User: "Review this change"
Agent: Reads 3 files → discovers 5 dependencies → reads those → discovers 10 more
Result: $0.20 → $2.50, 10 min → 2 hours
```

**Solution: Mandatory Scout-First**
```
1. Spawn haiku-scout to assess scope (files, LoC, complexity)
2. Scout recommends review depth and critical files
3. User approves scope (with cost estimate)
4. architectural-digest reviews ONLY scout-bounded files
```

**Cost:** Scout adds $0.001, saves $2.30+ in prevented scope explosion.

### Why Shared Framework?

Layers 7-8 (Code Quality & Test Coverage) are **identical** across both agents:
- Same metrics (cyclomatic complexity, LoC, test coverage)
- Same tools (linters, test runners, coverage analyzers)
- Same output format

**DRY Principle:** Extract to `.claude/frameworks/code-review-framework.md`, both agents reference it.

**Benefit:** Single source of truth, easier to maintain, consistent quality assessment.

---

## Integration Points

### Routing Schema

Add to `routing-schema.json`:
```json
{
  "agent_subagent_mapping": {
    "compliance-reviewer": "Explore",
    "architectural-digest": "Explore"
  },
  "patterns": {
    "post_implementation_review": {
      "triggers": ["review implementation", "compliance check", "code quality"],
      "routing_logic": {
        "if_specs_md_exists": "compliance-reviewer",
        "if_no_specs_md": "architectural-digest"
      }
    }
  }
}
```

### Skills

`/review-implementation`:
- Auto-detects mode (specs.md presence)
- Routes to correct agent
- Presents results

`/review-pr <number>`:
- Fetches PR diff via gh CLI
- Detects mode from PR description
- Posts review as PR comment

### Hooks (Optional)

- `post-commit`: Auto-review after commit (if `.claude/config/auto-review.enabled`)
- `pre-push`: Block push if critical issues (if `.claude/config/review-gate.enabled`)

---

## Cost & Time Estimates

### Per-Review Costs

| Agent | Cost | Duration |
|-------|------|----------|
| compliance-reviewer | $0.15 | 5-30 min |
| architectural-digest | $0.13 | 5-30 min |

**ROI:**
- Pre-review (staff-architect): $0.17, prevents $2-10 rework
- Post-compliance: $0.15, prevents $5-20 bug fixes
- Post-digest: $0.13, prevents $10-50 future refactoring

### Implementation Time

**MVP (4 weeks):**
- Week 1-2: Foundation (agent.md files, shared framework)
- Week 2-3: Routing integration
- Week 3: /review-implementation skill
- Week 4: Testing

**Full (8 weeks):**
- MVP + Hooks + Documentation + GO-specific adaptation

---

## Sharp Edges (Critical for Implementation)

### compliance-reviewer: perfectionism

**Problem:** Flagging every minor deviation from plan when plan allows flexibility.

**Example:**
```
Plan: "Use standard authentication library"
Code: Uses django.contrib.auth
Agent flags: "Not oauthlib" ← WRONG (both satisfy requirement)
```

**Mitigation:** Only flag deviations that violate EXPLICIT requirements. Quote exact requirement from specs.md.

### architectural-digest: scope_explosion

**Problem:** Reviewing entire codebase when diff shows 3 files.

**Example:**
```
User: "Review auth.py changes"
Agent: Reads auth.py → discovers middleware.py → discovers models.py → reads entire auth/ module
Result: 3 files → 20 files, $0.20 → $2.50
```

**Mitigation:** MANDATORY scout-first. ONLY review scout-identified critical files. Document scope boundary in digest.

---

## GO Migration Notes

### Language-Agnostic Design

This specification is **language-agnostic** and works for:
- Python (existing)
- GO (migration target)
- R (existing)
- Any language with conventions file

### GO-Specific Patterns to Add

**Good Patterns:**
- Early returns for error handling
- Error wrapping with `fmt.Errorf("%w", err)`
- Context propagation in function signatures
- Proper use of `defer` for cleanup
- Channel usage with proper closing

**Anti-Patterns:**
- Goroutine leaks (no termination)
- Missing context cancellation
- Ignored errors (`_ = someFunc()`)
- Mutex without defer unlock
- Panic in library code

### GO Integration Testing

After implementation, test with:
```bash
# Create sample GO diff
git checkout -b test-go-review
# Make GO changes to simulate real scenarios
git commit -m "test: GO authentication refactor"

# Run review
/review-implementation

# Verify:
# - GO conventions loaded (~/.claude/conventions/go.md)
# - GO patterns detected (goroutines, channels, error handling)
# - Output references GO idioms
```

---

## Next Steps

1. **Read ARCHITECTURE.md completely** (1 hour investment, critical for success)
2. **Extract specifications** from sections into actual files
3. **Update routing-schema.json** with agent mappings
4. **Create /review-implementation skill** with mode detection
5. **Test with sample diffs** (Python first, then GO)
6. **Validate sharp edge mitigations** work as designed
7. **Measure costs** across 10 reviews to confirm <$0.25 threshold

---

## Questions During Implementation?

**All answers are in ARCHITECTURE.md.** The specification is comprehensive and includes:
- Complete agent contracts (role, mindset, frameworks, sharp edges)
- Example input/output formats
- Integration patterns
- Cost analysis
- Implementation checklists

**No conversation context is required.** This document is self-contained and implementation-ready.

---

## Provenance

This specification was synthesized by the orchestrator agent from a comprehensive conversation analyzing:
- Pre vs post implementation review requirements
- Compliance review (with plan) vs architectural digest (without plan)
- Four architecture options (single agent, separate agents, pipeline, hybrid)
- Scope control strategies (scout-first architecture)
- Sharp edge detection and mitigation
- Cost-benefit analysis
- Integration patterns

**Conversation Date:** 2026-01-17
**Participants:** User + Sonnet (terminal) + Orchestrator (synthesis)
**Total Analysis:** ~50K tokens of architectural reasoning

**Result:** Complete, actionable specification for GO migration.
