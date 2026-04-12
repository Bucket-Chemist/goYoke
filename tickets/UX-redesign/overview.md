# Implementation Plan: UX Redesign

> Generated: 2026-04-11
> Workflow: /plan-tickets v1.0 (synthesis from braintrust-reviewed spec)
> Review Status: APPROVE_WITH_CONDITIONS
> Total: 28 tickets across 4 phases (~12-14.5 sessions)

## Executive Summary

Comprehensive UX overhaul of the GOgent-Fortress TUI across 5 areas: right panel density, agent tree legibility, conversation UX, status line feedback, and team monitoring. The original 24 recommendations were reviewed by Braintrust (Einstein + Staff-Architect + Beethoven) and augmented with 3 new recommendations for multi-profile PMF: a simple/expert toggle, first-run hints, and a reduce-motion accessibility flag. Priority ordering was revised to lead with universal-benefit changes (conversation improvements) and correct factual errors (existing implementations for 4a/4b).

## Strategic Approach

**Core insight from Braintrust:** The spec is excellent for power users but conflates terminal width with user expertise. The fix is surgical — 3 additions (~0.75 sessions) capture 80% of PMF improvement for non-power-users without changing any existing recommendation.

**Key corrections:**
- Recs 4a/4b already partially implemented — enhancement-only scope
- interfaces.go does NOT need modification — concrete types, not interfaces
- Tree.go recs consolidated into single branch to avoid merge conflicts
- All animations require reduce-motion config before shipping

## Implementation Phases

| Phase | Priority | Description | Tickets | Dependencies | Effort |
|-------|----------|-------------|---------|-------------|--------|
| 1 | P0 | Universal foundations + core differentiator | UX-001 through UX-007 | None | ~3-3.5 sessions |
| 2 | P1 | Tree overhaul + status integration | UX-008 through UX-012 | UX-003 (icon rail) for tree branch | ~2-3 sessions |
| 3 | P2 | Conversation enhancements + monitoring | UX-013 through UX-020 | UX-020 blocks P3 animations | ~3-4 sessions |
| 4 | P3 | Polish + advanced features | UX-021 through UX-028 | UX-020 (reduce-motion) for animations | ~3-4 sessions |

### Phase 1 (P0): Universal Foundations

| Ticket | Title | Effort | Branch |
|--------|-------|--------|--------|
| UX-001 | Conversation: horizontal rule between turns | 0.25s | p0-conversation |
| UX-002 | Conversation: user/assistant color differentiation | 0.25s | p0-conversation |
| UX-003 | Right panel: icon rail mode (< 30 cols) | 1.5-2s | p0-icon-rail |
| UX-004 | Activity: relative paths (strip project root) | 0.25s | p0-icon-rail |
| UX-005 | Status line: enhance existing context bar | 0.25s | p0-statusline |
| UX-006 | Status line: enhance existing cost display | 0.25s | p0-statusline |
| UX-007 | Simple/expert toggle (hide/show right panel) | 0.25s | p0-toggle |

**Decision points before Phase 1:**
- [ ] Render(mode) unified method vs View()/ViewCompact() — decide before UX-003
- [ ] Simple toggle scope: right panel only (recommended) or also simplify status line?
- [ ] Include P1 tree items (2a+2b+2c) in tree-overhaul branch with P0 icon rail?

### Phase 2 (P1): Tree Overhaul + Status

| Ticket | Title | Effort | Branch |
|--------|-------|--------|--------|
| UX-008 | Tree: two-column dot-leader layout | 1s | p1-tree-overhaul |
| UX-009 | Tree: full-row color by agent status | 0.25s | p1-tree-overhaul |
| UX-010 | Tree: inline cost per agent | 0.25s | p1-tree-overhaul |
| UX-011 | Status line: team indicator | 0.75s | p1-status-hints |
| UX-012 | First-run orientation hints | 0.25s | p1-status-hints |

### Phase 3 (P2): Conversation + Monitoring

| Ticket | Title | Effort | Branch |
|--------|-------|--------|--------|
| UX-013 | Detail: collapsible overview to one-liner | 0.5s | p2 |
| UX-014 | Conversation: inline streaming tool indicator | 0.75s | p2 |
| UX-015 | Conversation: collapsible tool-use blocks | 0.75s | p2 |
| UX-016 | Status line: agent count sparkline dots | 0.25s | p2 |
| UX-017 | Status line: adaptive 1-row at narrow widths | 0.5s | p2 |
| UX-018 | Teams: action-hinted toasts | 0.25s | p2 |
| UX-019 | Teams: auto-switch on completion | 0.25s | p2 |
| UX-020 | Reduce-motion config flag | 0.25s | p2 |

### Phase 4 (P3): Polish + Advanced

| Ticket | Title | Effort | Branch |
|--------|-------|--------|--------|
| UX-021 | Layout: focus-driven drawer/content split | 1s | p3 |
| UX-022 | Tree: density toggle | 0.25s | p3 |
| UX-023 | Tree: pulse animation (requires UX-020) | 0.25s | p3 |
| UX-024 | Conversation: timestamp gutter | 0.5s | p3 |
| UX-025 | Status line: cost flash (opt-in, requires UX-020) | 0.25s | p3 |
| UX-026 | Teams: timeline progress bars | 1s | p3 |
| UX-027 | Teams: tabs in drawer | 0.75s | p3 |
| UX-028 | Teams: diff summary on completion | 0.5s | p3 |

## Risk Register

| Risk | Likelihood | Impact | Mitigation | Tickets Affected |
|------|-----------|--------|------------|-----------------|
| Icon rail ANSI width miscalculation at 30-col boundary | Medium | High | Hysteresis (28/32 thresholds), boundary tests | UX-003 |
| Dynamic statusLineHeight causes layout overflow | Medium | High | Standalone method on AppModel, height invariant test | UX-017 |
| Dual render paths (View/ViewCompact) drift apart | High | Medium | Unified Render(mode) method | UX-003 |
| Merge conflicts in tree.go (6 recs, multiple branches) | High | High | Consolidated tree-overhaul branch | UX-003, UX-008-010 |
| Contractor reimplements 4a/4b from scratch | Medium | High | Tickets explicitly state enhancement-only | UX-005, UX-006 |
| Animations trigger WCAG 2.3.1 issues | Low | High | reduce-motion config (UX-020) ships before P3 | UX-023, UX-025 |
| Vibe-coder PMF for a tool they may never use | Low | Low | Minimal investment (~0.75 sessions), user research in P4 | UX-007, UX-012 |

## Review Summary

**Verdict:** APPROVE_WITH_CONDITIONS
**Source:** Braintrust (Einstein + Staff-Architect + Beethoven) — $4.39
**Critical Issues:** 0
**Corrections Required:** 5 (all addressed in ticket descriptions)
**New Recommendations Added:** 3 (N1, N2, N3)

### Conditions (addressed)

1. **C-1:** Recs 4a/4b reference existing code — addressed in UX-005, UX-006
2. **M-1:** interfaces.go not needed — removed from UX-003
3. **M-3:** Test coverage expanded — boundary test matrix in all relevant tickets
4. **M-4:** statusLineHeight method — addressed in UX-017
5. **m-5:** Branch consolidation — addressed in branching strategy

## Success Criteria

1. Conversation panel readable at all width tiers with clear turn separation
2. Agent tree legible at Standard tier (80-119 cols) via icon rail
3. Cost and context visible at a glance in status line
4. Simple toggle provides escape hatch for non-power-users
5. All animations respect reduce-motion config
6. Test coverage: boundary tests at {15, 22, 28-32, 45, 60, 80, 120, 180} cols

## Next Steps

1. Resolve Phase 1 decision points (Render method, toggle scope, branch consolidation)
2. Run `/ticket` to begin implementation with UX-001 (lowest risk, highest universal impact)
3. Address review conditions during implementation
4. Conduct user research in Phase 4 to validate audience segmentation

---

_Generated by /plan-tickets skill from braintrust-reviewed UX-REDESIGN-SPEC.md_
_Review critique: BRAINTRUST-ANALYSIS-20260411.md_
_Ticket index: tickets-index.json (28 tickets)_
