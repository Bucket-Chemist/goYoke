# Braintrust Analysis: UX-REDESIGN-SPEC PMF & Architectural Sanity

**Generated:** 2026-04-11
**Team:** braintrust-ux-redesign
**Cost:** $4.39 (Einstein $1.40 + Staff-Architect $2.02 + Beethoven $0.97)
**Verdict:** APPROVE_WITH_CONDITIONS (17 keeps, 7 modifications, 0 removals, 3 additions)

---

## Executive Summary

The UX-REDESIGN-SPEC is architecturally sound and well-motivated, but optimizes exclusively for power users while claiming to serve broader audiences. Einstein identified a fundamental category error — conflating terminal width with user expertise — while Staff-Architect found the spec implementable with 5 correctable conditions (most critically: Recs 4a/4b already exist in the codebase, and interfaces.go references are factually wrong). The synthesis recommends implementing the spec as-is for its primary audience (power users) with three targeted additions for broader PMF: a simple/expert toggle (hide/show agent tree), first-run orientation hints via the existing hint bar, and a reduce-motion config flag. Priority ordering should shift: conversation improvements (3a, 3b) join P0 for universal benefit; tree overhaul changes consolidate into a single branch to avoid merge conflicts; and Recs 4a/4b shrink to enhancement-only scope. The spec's 24 recommendations produce 17 keeps, 7 modifications, and 0 removals — none should be cut, but several need implementation corrections before contractor handoff.

---

## Analysis Perspectives

### Einstein (PMF / Theoretical)

The spec conflates terminal width (hardware constraint) with user expertise (information appetite), treating responsive sizing as progressive disclosure when these are orthogonal dimensions. All 24 recommendations increase information density — serving power users while actively overwhelming vibe coders. The spec draws design principles exclusively from power-user tools (lazygit, gh-dash, spotify-tui) with no references to tools that serve non-technical users. Einstein recommends a dual-axis approach (LayoutTier for sizing + ExperienceProfile for complexity) with conversation-first defaults and priority reordering to start with universally beneficial changes (Recs 3a, 3b).

**Strengths:**
- Identified the fundamental category error (width != expertise) that neither the spec nor Staff-Architect addressed
- Applied Miller's 7+/-2 cognitive load framework to TUI regions — 9+ simultaneous channels exceed non-expert limits
- Proposed three novel approaches (ExperienceProfile, Conversation-First Default, Contextual Complexity Escalation) with honest feasibility/risk assessment for each
- Surfaced the critical meta-question: whether vibe coders would use a terminal TUI at all — saving potential wasted design effort
- Correctly identified that the Compact tier may be dead weight if no users run terminals below 80 columns

### Staff-Architect (Practical)

**Verdict:** APPROVE_WITH_CONDITIONS

**Strengths:**
- Discovered Recs 4a/4b are already partially implemented in the codebase (C-1) — the most impactful factual correction, changing P0 scope and effort
- Caught that interfaces.go modification is unnecessary — tree/detail are concrete types, not interface-backed (M-1)
- Quantified the combinatorial test surface: 48+ render paths vs 4 proposed boundary tests (M-3)
- Identified the tree.go serialization bottleneck: 6 recommendations across 4 branches create guaranteed merge conflicts (m-5)
- Provided specific code references (line numbers, function names) for every finding, making corrections immediately actionable

---

## Convergence Points

Both analyses agree on:

1. The spec is well-designed for its primary audience (power users)
2. Recs 3a+3b (conversation improvements) are universally beneficial and low-risk
3. Rec 1a (icon rail) is the most architecturally complex but also the core differentiator
4. The priority matrix is mostly correct but P0 composition should change
5. All 24 recommendations are worth keeping — none should be removed
6. Testing coverage for width-conditional rendering is currently insufficient

---

## Divergence Resolution

### 1. Whether to add an ExperienceProfile system (dual-axis: width x expertise)

- **Einstein:** Strongly recommends decoupling width from expertise. Proposes ExperienceProfile (Simple/Standard/Expert) as a new architectural concept orthogonal to LayoutTier.
- **Staff-Architect:** Does not address this — reviews the spec as-is and finds it architecturally sound. Notes that adding SetProfile() to 18+ widget interfaces would be a significant surface area change.
- **Resolution:** Adopt Einstein's fallback position: a **simple toggle** (show/hide agent tree) that captures 80% of PMF benefit at 10% of implementation cost. The full ExperienceProfile system adds 12 combinatorial render states — defer to follow-up spec after user research. Implementation: one keybinding, one config persistence, one conditional in renderRightPanel.

### 2. Whether vibe coders are a valid TUI audience

- **Einstein:** Flags as high-importance open question. Argues vibe coders may exclusively use GUI tools (VS Code extension, web interface, desktop app), making TUI-level vibe-coder PMF a potentially non-existent problem.
- **Staff-Architect:** Does not question the audience — reviews the spec on its own terms.
- **Resolution:** Treat the TUI as primarily a power-user tool. Add simple/expert toggle as the sole vibe-coder accommodation (low cost, high option value). Conduct user research in parallel to validate before deeper investment.

### 3. Priority ordering: P0 composition

- **Einstein:** Recommends reordering P0 to lead with conversation improvements (3a, 3b) and ExperienceProfile system. Current P0 (right-panel density) should become P1.
- **Staff-Architect:** Adjusts effort: 4a/4b are already implemented (remove from critical path), 1a is underestimated (1.5-2 sessions). Recommends grouping tree.go changes into single branch.
- **Resolution:** Revised P0 — see Implementation Phases below.

### 4. Cost visibility: 'most visible element' vs anxiety-inducing

- **Einstein:** Questions whether cost should be the most prominent element. For subscription users, cost visibility adds anxiety without actionability.
- **Staff-Architect:** Commends the spec's cost awareness as operationally important.
- **Resolution:** Keep prominent cost display as default. Make cost flash animation (4e) opt-in via config. In simple mode, cost present but not animated.

### 5. Accessibility: animations, motion sensitivity, WCAG 2.3.1

- **Einstein:** Flags pulse/flash/sparkline as potentially harmful for users with vestibular sensitivities or seizure conditions.
- **Staff-Architect:** Does not address motion accessibility.
- **Resolution:** Add reduce-motion config flag that disables all animations. Ship before P3 animations go live.

---

## Unified Recommendations

### Recommendation

Implement the spec's 24 recommendations with 7 modifications and 3 additions, under a revised P0 that prioritizes universal-benefit changes and corrects factual errors. Consolidate tree.go changes into a single branch. Add a simple/expert toggle, first-run hints, and reduce-motion config as the minimum viable non-power-user accommodations.

### Rationale

The spec is architecturally sound (Staff-Architect: APPROVE_WITH_CONDITIONS) and contains genuinely good UX design for its primary audience (power users). The modifications address factual errors (C-1, M-1), testing gaps (M-3), implementation risks (M-4, M-5), and the PMF gap for non-power-users (Einstein's core finding). The 3 additions capture 80% of Einstein's PMF improvement at ~10% of the cost of the full ExperienceProfile system.

### Supporting Evidence

- **From Einstein:** The dual-axis insight (width != expertise) is the key theoretical contribution. The priority reorder (3a+3b to P0) ensures the first delivered increment benefits all user profiles, not just power users.
- **From Staff-Architect:** The 5 sign-off conditions are concrete and achievable. The finding that 4a/4b already exist reclaims ~1 session of P0 budget. The tree-overhaul branch consolidation eliminates guaranteed merge conflicts.

### Not Recommended

1. **Removing or deferring any of the 24 existing recommendations** — All are well-designed for the TUI's primary audience. Einstein's PMF concerns are addressed through additions, not subtractions.
2. **Implementing the full ExperienceProfile system in P0** — Combinatorial complexity (12+ render states) would triple the already-undertested surface. Simple toggle captures 80% of benefit.
3. **Conversation-first default (hiding all panels by default)** — Undermines the TUI's core value proposition (real-time visibility into multi-agent orchestration). The toggle (opt-OUT) is safer than conversation-first (opt-IN) because it preserves discoverability.

### Secondary (Defer to Phase 4+)

1. **Full ExperienceProfile system** — Pursue if user research validates >20% non-power-user sessions AND simple toggle proves insufficient.
2. **Contextual Complexity Escalation** — Auto-adapt based on agent count. Pursue if three-profile segmentation correlates with agent count.

---

## Implementation Phases

### Phase 1 (P0): Universal foundations + core differentiator

**Description:** Implement Recs 3a+3b (conversation readability), 1a (icon rail), 1b (relative paths), 4a+4b (enhance existing), and the simple/expert toggle. Correct spec errors (C-1, M-1) before contractor handoff. Define boundary test matrix (M-3).

**Decision points:**
- Render(mode RenderMode) unified method vs separate View()/ViewCompact() — decide before Rec 1a
- Simple toggle scope: hide right panel only, or also simplify status line? Start with right-panel-only
- Tree-overhaul branch scope: include 2a+2b+2c from P1 to avoid merge conflicts, or keep P0 minimal?

**Success criteria:**
- Conversation panel has horizontal rules between messages and user/assistant color differentiation
- Icon rail renders correctly at widths 22-29 with lipgloss.Width() assertions passing
- Simple toggle hides/shows right panel with one keybinding, persisted to config
- All boundary tests pass at {15, 22, 29, 30, 31, 45, 60, 80} column widths
- Recs 4a/4b enhancements reference and extend existing renderContextBar()/costStyle()

**Estimated effort:** ~3-3.5 sessions

### Phase 2 (P1): Tree overhaul + status line integration

**Description:** Implement remaining tree improvements (2a dot leaders, 2b status colors, 2c inline cost) in consolidated tree-overhaul branch. Add team indicator to status line (5a). Add first-run orientation hints via hintBarWidget.

**Decision points:**
- Dot leader width calculation: validate ANSI-safe arithmetic before merging (m-1 risk)
- First-run hint content: which panels get orientation hints and for how many sessions?

**Success criteria:**
- Dot leaders align correctly across all width tiers with styled and unstyled strings
- Tree-overhaul branch merges without conflicts
- Status line team indicator shows active team name and member count
- First-run hints display on sessions 1-3 and suppress thereafter

**Estimated effort:** ~2-3 sessions

### Phase 3 (P2): Conversation enhancements + monitoring

**Description:** Implement Recs 3c (streaming indicator), 3d (collapsible tool blocks), 3e (timestamp gutter), 4d (adaptive status density), 5b (action-hinted toasts), 5c (auto-switch). Add reduce-motion config flag.

**Decision points:**
- Rec 4d: statusLineHeight method — validate no circular dependency with computeLayout (M-4)
- Rec 3d: collapse state storage — use tool_use_id as key per Staff-Architect failure mode analysis
- reduce-motion scope: which animations does it disable?

**Success criteria:**
- Collapsible tool blocks retain state across re-renders
- Adaptive 1-row status line renders correctly without layout overflow
- reduce-motion=true disables all animation rendering
- lipgloss.Height(statusLine.View()) == computed height invariant holds at all tiers

**Estimated effort:** ~3-4 sessions

### Phase 4 (P3): Polish + advanced features

**Description:** Implement Recs 2d (density toggle), 2e (pulse animation — respecting reduce-motion), 4c (sparklines), 4e (cost flash — opt-in), 5d (timeline), 5e (team tabs — with edge cases), 5f (diff summary). Conduct user research on audience segmentation.

**Decision points:**
- User research results: is the three-profile segmentation valid? Should full ExperienceProfile be pursued?
- Rec 5e: team tab edge cases — confirm spec additions (empty state, overflow, vim mode, dismiss semantics)
- Cost flash (4e): confirm opt-in-only stance based on user feedback

**Success criteria:**
- All P3 animations respect reduce-motion config
- Team tabs handle 0 teams, overflow, and vim mode correctly
- User research report delivered with audience segmentation data
- All 48+ render paths have boundary test coverage

**Estimated effort:** ~3-4 sessions

---

## Risk Assessment

| # | Risk | Severity | Mitigation |
|---|------|----------|------------|
| 1 | Icon rail garbled at 30-col boundary (ANSI width miscalculation) | High | Hysteresis: switch at 28 icon->text and 32 text->icon. Boundary tests at {29,30,31}. Unified Render(mode) method. |
| 2 | Dynamic statusLineHeight causes layout overflow | High | Implement as method on AppModel computing from m.width directly. Add invariant test: lipgloss.Height(statusLine.View()) == expected. |
| 3 | Dual render paths (View/ViewCompact) drift apart | Medium | Use single Render(mode RenderMode) instead of separate methods. If dual-path, add compile-time test both render same agent IDs. |
| 4 | Merge conflicts between tree.go branches (6 recs, 4 branches) | High | Consolidate 1a+2a+2b+2c into single tree-overhaul branch. Density toggle (2d) and pulse (2e) stay separate. |
| 5 | Contractor reimplements 4a/4b from scratch | High | Rewrite tickets to reference existing renderContextBar() at statusline.go:228 and costStyle() at :476. Enhancement-only scope. |
| 6 | Animations trigger WCAG 2.3.1 accessibility issues | Medium | reduce-motion config flag. Ship before any P3 animations. |
| 7 | Investing in vibe-coder PMF for a tool vibe coders may never use | Low | Keep accommodations minimal (~0.5 sessions). User research in parallel. |

---

## Open Questions

1. What is the actual user distribution across the three profiles? Is this TUI used by non-technical users at all?
2. Should the TUI explicitly position itself as a power-user tool and direct simpler users to CLI or web interface?
3. Does the Compact tier (<80 cols) currently serve anyone, or is it purely defensive code?
4. How does the existing hint bar (hintBarWidget) interact with the onboarding gap?
5. Should tree.go changes (1a, 2a, 2b, 2c) be grouped into a single tree-overhaul branch spanning P0+P1?

---

## Assumptions to Validate

| # | Assumption | Priority | Blocking | Source | Validation Method |
|---|-----------|----------|----------|--------|-------------------|
| 1 | Users can be segmented into three profiles (vibe/intermediate/power) | High | No | Einstein | Session telemetry: agent count, feature usage, duration. Cluster into tiers. |
| 2 | Vibe coders would use a terminal TUI at all | High | No | Einstein | Check if TUI is launched from GUI environments (VS Code terminal, web terminals). |
| 3 | Compact tier (<80 cols) serves actual users | Medium | No | Einstein | Session telemetry for terminal width distribution. |
| 4 | Recs 4a/4b already partially implemented | High | Yes | Staff-Arch | Verify renderContextBar() at statusline.go:228 and costStyle() at :476. (Verified by Staff-Architect.) |
| 5 | Animations are universally beneficial | Medium | No | Both | Ensure reduce-motion config option. Check terminal capability detection. |

---

## Full Agent Outputs

- Einstein: `stdout_einstein.json`
- Staff-Architect: `stdout_staff-arch.json`
- Pre-synthesis: `pre-synthesis.md`
- Beethoven: `stdout_beethoven.json`

All in: `.claude/sessions/a341f4b1-8129-407a-970e-d5b6f7b1650b/teams/20260411-084208.braintrust/`
