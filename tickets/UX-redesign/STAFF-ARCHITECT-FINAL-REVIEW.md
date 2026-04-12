# Staff-Architect Final Review: UX Redesign Tickets

**Reviewed:** 2026-04-11
**Reviewer:** Staff Architect Critical Review (Opus)
**Input Files:**
- `tickets/UX-redesign/PREFLIGHT-REVIEW.md` (router sanity check)
- `tickets/UX-redesign/BRAINTRUST-ANALYSIS-20260411.md` (Einstein + Staff-Arch + Beethoven synthesis)
- `tickets/UX-redesign/overview.md` (implementation plan)
- `tickets/UX-redesign/tickets-index.json` (28-ticket registry)
- `tickets/UX-redesign/UX-REDESIGN-SPEC.md` (1233-line spec, Sections 6-12 reviewed)
- 11 individual ticket files spot-checked
- 5 source files verified against claims

---

## Verdict: APPROVE_WITH_CONDITIONS

**Confidence Level:** HIGH

- Rationale: Plan is well-structured, Braintrust review was thorough, preflight caught the two blocking issues. Source code verification confirms most claims. Remaining conditions are corrections, not redesigns.

**Issue Counts:**
- Critical: 2 (preflight RED-1 and RED-2, confirmed — must fix before implementation)
- Major: 4 (2 upgraded from YELLOW, 2 new findings)
- Minor: 5

**Commendations:** 5

**Summary:** The 28-ticket plan is architecturally sound and well-phased. The two RED flags are straightforward fixes (wrong file path, keybinding conflict). Four YELLOW flags need correction but none are blocking. The preflight missed one significant issue (panel.go cross-phase merge conflict risk) and one phase ordering violation (UX-016 depends on a later-phase ticket). With the 6 conditions below addressed, Phase 1 can begin Monday.

**Go/No-Go Recommendation:** GO, conditional on the 6 items in Final Conditions.

---

## Preflight Flag Adjudication

### RED-1: UX-004 — Wrong file reference | CONFIRMED CRITICAL

**Preflight claim:** References `internal/tui/cli/activity.go` which doesn't exist.
**Verification:** `ExtractToolActivity()` confirmed at `internal/tui/cli/agent_sync.go:231`. The function returns `state.AgentActivity` with `ToolName`, `ToolID`, `Target`, `Preview`, `Timestamp` fields (agent_sync.go:238-245). UX-004.md line 31 still lists `activity.go`.
**Action:** Replace `internal/tui/cli/activity.go` with `internal/tui/cli/agent_sync.go` in UX-004.md Files section. Also update spec Section 3.3, Section 8, and Section 12.

### RED-2: UX-007 — Keybinding conflict | CONFIRMED CRITICAL

**Preflight claim:** `alt+p` is already bound to "cycle perm mode".
**Verification:** `CyclePermMode` at `config/keys.go:202-204`:
```go
CyclePermMode: key.NewBinding(
    key.WithKeys("alt+p"),
    key.WithHelp("alt+p", "cycle perm mode"),
),
```
UX-007.md line 25 still proposes `alt+p`.
**Action:** Choose alternative keybinding. After reviewing keys.go:195-295, available candidates:
- `alt+\\` — intuitive "toggle split" mnemonic, no current binding
- `alt+0` — "zero panels" mnemonic, no current binding
- `alt+s` — needs audit (may conflict)

Recommend `alt+\\` — the backslash evokes a vertical split, which is what the toggle controls.

### YELLOW-1: UX-003 — Effort estimate may be low | CONFIRMED, SCOPING RESOLVES

**Preflight claim:** 1.5-2s may be low given `rightPanelMode` switch with multiple modes.
**Verification:** `renderRightPanel()` at `layout.go:523-572` has a 6-way switch: `RPMAgents`, `RPMDashboard`, `RPMSettings`, `RPMTelemetry`, `RPMPlanPreview`, `RPMTeams`. Each mode calls its own `View()` method.
**Adjudication:** If icon rail applies to **agents only** (see Q2 below), the 1.5-2s estimate is correct — only `tree.go` and `detail.go` need `Render(mode)`. If all 6 modes need icon rail variants, effort is 3+ sessions. **Recommendation: scope to agents-only. Estimate stands at 1.5-2s.**

### YELLOW-2: UX-008 — Dot leader ANSI safety | CONFIRMED, KEEP AS YELLOW

**Preflight claim:** Dot count arithmetic mixing `lipgloss.Width()` measured widths with integer offsets risks off-by-one at boundary widths.
**Verification:** The concern is valid. `lipgloss.Width()` uses `ansi.StringWidth()` which handles escape codes, but the formula `dotsNeeded = w - depth*2 - 2 - nameWidth - valueWidth - 1` includes literal offsets for indent, padding, and separator that must agree with rendered output.
**Adjudication:** Keep as YELLOW. The preflight's recommendation (test helper asserting `lipgloss.Width(renderedRow) == expectedWidth`) is the right mitigation. Add to UX-008 acceptance criteria.

### YELLOW-3: UX-015 — Tool blocks already have collapse infrastructure | CONFIRMED, UPGRADE TO MAJOR

**Preflight claim:** `panel.go` already has collapse/expand for tool blocks.
**Verification:** `renderToolBlock()` at `panel.go:1006-1037` has full collapsed/expanded rendering:
- `tb.Expanded` boolean field (line 1019)
- Collapsed view: status prefix + tool name + truncated input (line 1022-1025)
- Expanded view: tool name + full input + output (line 1027-1036)
- `ToggleToolExpansion` binding at `keys.go:280` (`alt+e`)
- `CycleExpansion` binding at `keys.go:284` (`alt+E`)

**Additionally:** Ticket uses terminology `tool_use_id` but the codebase field is `ToolID` (see `AgentActivity.ToolID` at agent_sync.go:241 and `ToolBlock` struct which has the `Expanded` field).

**Upgrade reason:** Ticket is written as greenfield (0.75s) when it should be enhancement-only (0.25-0.5s). A contractor following this ticket as-is would reimplement existing functionality, wasting effort and potentially breaking the working collapse system. This is Risk #5 from the Braintrust analysis (contractor reimplements from scratch) recurring in a different ticket.

**Action:** Rewrite UX-015 as enhancement-only. State: "Existing infrastructure at `panel.go:1006-1037` with `ToggleToolExpansion` (`alt+e`) and `CycleExpansion` (`alt+E`). Enhancement: change default `Expanded` state to `false` for new blocks, verify `ToolID`-keyed state retention across re-renders." Reduce effort to 0.25-0.5s. Replace `tool_use_id` with `ToolID`.

### YELLOW-4: UX-016 — Wrong dependency | CONFIRMED, UPGRADE TO MAJOR (phase ordering violation)

**Preflight claim:** UX-016 depends on UX-024 (timestamp gutter), which makes no sense for sparkline dots.
**Verification:** `tickets-index.json` line 205-206 confirms `"dependencies": ["UX-024"]`. UX-024 is Phase 4 (P3). UX-016 is Phase 3 (P2). **This is a phase ordering violation** — a P2 ticket cannot depend on a P3 ticket without either moving UX-016 to P3 or UX-024 to P2.
**Upgrade reason:** Not just a wrong dependency — it's a phase ordering error that would block implementation if taken literally.
**Action:** Remove the UX-024 dependency entirely. UX-016 (sparkline dots) is self-contained — it reads agent state and renders dots in the status line. No dependency needed.

### YELLOW-5: UX-017 — statusLineHeight constant-to-method | DOWNGRADE TO MINOR

**Preflight claim:** Converting constant to dynamic method is risky given the 8-term height calculation.
**Verification:** The height expression at `layout.go:160`:
```go
dims.contentHeight = m.height - bannerHeight - tabBarHeight - providerTabH - statusLineHeight - taskBoardH - toastH - hintH - bcH - borderFrame
```
Has 10 terms total: 4 constants (`bannerHeight=3`, `tabBarHeight=1`, `statusLineHeight=2`, `borderFrame=2`) and 5 dynamic values (`providerTabH`, `taskBoardH`, `toastH`, `hintH`, `bcH`).

**Key finding the preflight missed:** `toastH` at `layout.go:144-146` is ALREADY a dynamic height computed via `m.shared.toasts.Height()`. The pattern of "compute height from component, subtract from content" already exists and works. Converting `statusLineHeight` from constant to method follows this established pattern.

**Downgrade reason:** The existing `toastH` precedent proves dynamic heights work in this expression. The invariant test (`lipgloss.Height(statusLine.View()) == computed`) in UX-017's AC is the right guardrail.
**Remaining risk:** Low. The method must read `m.width` directly (not `computeLayout` output) to avoid circular dependency. UX-017's implementation note already states this.

### YELLOW-6: UX-018 — UDS toast protocol | RESOLVED

**Preflight claim:** Toast emission from `spawn.go` may require UDS protocol extension.
**Verification:** The UDS/bridge protocol **already supports toasts**:
- `bridge/server.go:186-187`: `case mcp.TypeToast: b.handleToast(req)`
- `bridge/server.go:327-334`: `handleToast()` unmarshals `ToastPayload` and sends `model.ToastMsg` to TUI
- `bridge/server_test.go:207-225`: `TestToastDelivery` verifies end-to-end toast delivery
- However: `cmd/gogent-team-run/spawn.go` has **zero** toast references (grep confirmed empty)

**Resolution:** Protocol layer exists and is tested. UX-018 requires adding toast emission calls to `spawn.go` using the existing `mcp.TypeToast` message type — straightforward function calls, NOT protocol extension. Effort estimate of 0.25s is correct.
**Action:** Update UX-018 description to reference existing toast protocol at `bridge/server.go:327` and note that `spawn.go` needs to emit `TypeToast` messages at lifecycle points.

### YELLOW-7: UX-022 — Keybinding conflict with CycleExpansion | CONFIRMED MINOR

**Preflight claim:** UX-022 proposes `alt+shift+e` which conflicts with `CycleExpansion` at `keys.go:284`.
**Verification:** Confirmed. `CycleExpansion` at `keys.go:284-287`:
```go
CycleExpansion: key.NewBinding(
    key.WithKeys("alt+E"),
    key.WithHelp("alt+shift+e", "cycle expansion"),
),
```
**Adjudication:** Minor because UX-022 is P3 and there's time to resolve. Two options:
1. **Context-sensitive** (preferred): `alt+shift+e` cycles expansion when Claude panel focused, cycles density when tree panel focused. Bubbletea's focus system supports this.
2. **New binding:** e.g., `alt+d` for density.

Recommend option 1 (context-sensitive) — it's the more intuitive UX and doesn't consume another binding slot.
**Action:** Add note to UX-022 specifying context-sensitive behavior or new binding.

---

## Answers to Specific Questions

### Q1: Is `Render(mode RenderMode)` the right choice for UX-003?

**Answer: Yes.** `Render(mode RenderMode)` is the correct architectural choice.

**Evidence:**
- `renderRightPanel()` at `layout.go:536` calls `m.agentTree.View()` and `m.agentDetail.View()` for the `RPMAgents` case
- `View()` is Bubbletea's interface method — it must remain, but can delegate to `Render(mode)`
- The `LayoutTier` type already exists at `layout.go:52` for responsive breakpoints

**Rationale:**
1. **Prevents path drift** (Risk #3): A single `Render(mode)` method is the single source of truth for all rendering variants. Separate `View()`/`ViewCompact()` methods WILL diverge over time as features are added to one but not the other.
2. **Testable**: Width tiers can be tested by calling `Render(IconRail)` directly without constructing a full model at a specific width.
3. **Clean delegation**: `View()` becomes `return m.Render(m.currentMode())` — one line, no conditional logic in the interface method.
4. **Pattern alignment**: Matches the `LayoutTier` architecture already in the codebase.

**Implementation sketch:**
```go
type RenderMode int
const (
    RenderFull RenderMode = iota
    RenderIconRail
)

func (m AgentTreeModel) Render(mode RenderMode) string { ... }
func (m AgentTreeModel) View() string { return m.Render(m.mode) }
```

### Q2: Should icon rail apply to all `rightPanelMode` values or only agent panel?

**Answer: Agent panel only (`RPMAgents`).**

**Evidence:** `renderRightPanel()` at `layout.go:534-572` has 6 cases:
- `RPMAgents` (line 536): `m.agentTree.View()` + `m.agentDetail.View()` — **structured data, benefits from icon rail**
- `RPMDashboard` (line 539): `m.shared.dashboard.View()` — self-contained widget
- `RPMSettings` (line 545): `m.shared.settings.View()` — self-contained widget
- `RPMTelemetry` (line 551): `m.shared.telemetry.View()` — self-contained widget
- `RPMPlanPreview` (line 557): `m.shared.planPreview.View()` — self-contained widget
- `RPMTeams` (line 563): `m.shared.teamDetail.View()` — self-contained widget with `SetSize()` call

**Rationale:**
1. Icon rail (icon + 2-char abbreviation + cost) is semantically meaningful only for agent representation. "Icon rail for settings" is architecturally meaningless.
2. Non-agent panels are self-contained widgets that handle their own narrow-width rendering via their internal `View()` methods. Imposing an external `RenderMode` on them creates coupling for no benefit.
3. Effort delta is significant: agents-only = 1.5-2s; all modes = 3+s (each widget needs a compact variant).
4. If narrow-width rendering is needed for other panels later, each widget can implement its own internal responsive logic — this is the existing pattern.

**Action:** Add explicit note to UX-003: "Icon rail applies to `RPMAgents` mode only. Other `rightPanelMode` values use their existing `View()` methods at all widths."

### Q3: Is `statusLineHeight` constant-to-method safe?

**Answer: Yes, safe.** The risk is lower than the preflight suggests.

**Evidence:** The height calculation at `layout.go:160`:
```go
dims.contentHeight = m.height - bannerHeight - tabBarHeight - providerTabH -
    statusLineHeight - taskBoardH - toastH - hintH - bcH - borderFrame
```

5 of the 10 terms are already dynamic:
- `providerTabH` — conditional on provider tab visibility
- `taskBoardH` — conditional on task board
- `toastH` — computed via `m.shared.toasts.Height()` at `layout.go:144-146`
- `hintH` — conditional on hint bar visibility (layout.go:149-151)
- `bcH` — conditional on breadcrumb (layout.go:152-158)

The `toastH` pattern is the exact precedent: component computes its own height, value is subtracted from content height.

**Implementation requirements:**
1. Method on `AppModel` (or `StatusLineModel`) that reads `m.width` directly
2. Returns `1` for width < 120, `2` for width >= 120
3. Called in the same position as the current constant
4. Invariant test: `lipgloss.Height(statusLine.View()) == m.statusLineHeight()`
5. Must NOT call `computeLayout` (circular dependency)

**Remaining risk:** Low. The invariant test catches disagreement between computed and rendered height at build time.

### Q4: Should tree-overhaul (UX-008+009+010) be pulled into P0 with UX-003?

**Answer: Consolidate onto one branch, but keep phase boundaries for milestone tracking.**

**Evidence:** The Braintrust analysis (Risk #4) identifies 6 tree.go recommendations across 4 potential branches as guaranteed merge conflicts. UX-008 depends on UX-003 (`tickets-index.json:103-105`), and UX-009/010 depend on UX-008.

**Recommendation:**
1. Create a single `tree-overhaul` branch
2. Implement UX-003 (P0) as the first commits — this establishes `Render(mode RenderMode)`
3. Implement UX-008+009+010 (P1) as subsequent commits on the same branch
4. P0 milestone: UX-003 complete and tested. Can merge via PR at this point if P0 must ship independently.
5. P1 milestone: UX-008+009+010 complete. Full branch merges.

**Why not pull UX-008+009+010 into P0 proper:** P0's value proposition is "universal foundations" — conversation improvements, status line enhancements, and the simple toggle. Tree overhaul is a P1 concern. Pulling it into P0 inflates the P0 milestone from 3-3.5s to 5-5.5s, which delays the first deliverable. Keep phases distinct for milestone discipline.

### Q5: Is the UDS protocol toast-aware?

**Answer: Yes, fully toast-aware. No protocol work needed.**

**Evidence:**
- `bridge/server.go:186-187`: `case mcp.TypeToast: b.handleToast(req)` — toast type handled in the IPC message switch
- `bridge/server.go:327-334`: `handleToast()` unmarshals `mcp.ToastPayload`, sends `model.ToastMsg{Text: p.Message, Level: model.ToastLevel(p.Level)}` to TUI
- `bridge/server_test.go:207-225`: `TestToastDelivery` verifies full round-trip (IPC request → ToastMsg delivery)
- `cmd/gogent-team-run/spawn.go`: **Zero toast references** (confirmed via grep)

**Conclusion:** The protocol and TUI toast rendering are complete and tested. `spawn.go` simply doesn't emit toast messages yet. UX-018 requires adding `mcp.TypeToast` emission calls at lifecycle points in `spawn.go` (launch, wave complete, member failure, team complete, budget warning). This is ~20 lines of code using the existing protocol — effort 0.25s is accurate.

---

## Additional Findings (Preflight Missed)

### NEW-1 (Major): panel.go cross-phase merge conflict risk

**Issue:** `panel.go` (1264 lines) is touched by 5 tickets across 3 phases:
- P0: UX-001 (horizontal rule), UX-002 (color differentiation)
- P2: UX-014 (streaming indicator), UX-015 (collapsible tool blocks)
- P3: UX-024 (timestamp gutter)

The Braintrust flagged tree.go merge conflicts (Risk #4) and recommended branch consolidation. The same analysis was NOT applied to panel.go, which has a higher cross-phase touch count (5 tickets vs 4 for tree.go).

**Mitigation:** P0 changes (UX-001, UX-002) are additive — inserting separator strings and changing style application. They're unlikely to conflict with P2/P3 changes at different code locations. But the risk should be documented. **Recommendation:** Implement UX-001 and UX-002 on a shared `p0-conversation` branch (already planned). For P2, implement UX-014 and UX-015 together. Ensure P0 conversation changes merge before P2 starts.

### NEW-2 (Major): UX-016 phase ordering violation

**Issue:** Beyond the wrong dependency (YELLOW-4), UX-016 (P2, Phase 3) depends on UX-024 (P3, Phase 4) per `tickets-index.json:205-206`. Even if the dependency were semantically correct, a P2 ticket cannot depend on a P3 ticket — this blocks implementation ordering.

**Action:** Remove the dependency entirely (as recommended in YELLOW-4 adjudication). Verify no other cross-phase dependency violations exist. (Checked: UX-009→UX-008 are same phase. UX-023→UX-020 is P3→P2, which is correct — later phase depends on earlier. No other violations found.)

### NEW-3 (Minor): UX-012 vague file paths

**Issue:** UX-012.md lists directories instead of files:
```
- `internal/tui/components/hints/`
- `internal/tui/config/`
```
A contractor needs specific file names. The hint bar component is `hintbar.go` (referenced in the preflight as `hintbar/hintbar.go`). Config changes would be in the user config struct.

**Action:** Update UX-012 Files section with specific filenames.

### NEW-4 (Minor): UX-004 fix not yet applied

**Issue:** The preflight identified RED-1 and recommends fixing it, but the ticket itself (UX-004.md line 31) still lists `activity.go`. The fix has not been applied yet.

**Action:** Apply the fix before implementation begins.

### NEW-5 (Minor): UX-022 ticket Review Notes empty despite known keybinding conflict

**Issue:** UX-022.md "Review Notes" says "None." despite YELLOW-7 identifying a keybinding conflict with `CycleExpansion` (`alt+shift+e`). The ticket should reference this known issue.

**Action:** Add Review Notes to UX-022 describing the conflict and resolution strategy (context-sensitive or new binding).

---

## 7-Layer Review

### Layer 1: Assumption Register

| # | Assumption | Source | Verified? | Risk if False |
|---|-----------|--------|-----------|---------------|
| A-1 | Icon rail only needed for agent panel mode | Implicit in UX-003 | **Unverified — now resolved (Q2: agents-only)** | Effort doubles if all 6 modes need variants |
| A-2 | Session count tracking available in config | UX-012 | Unverified | Need to add session counter to config struct if absent |
| A-3 | `statusLineHeight` method won't create circular dependency | UX-017 | **Verified safe** — method reads `m.width`, not `computeLayout` output | Layout breaks with stack overflow |
| A-4 | UDS protocol needs toast extension for spawn.go | UX-018 | **Verified FALSE** — protocol already supports toasts | Effort would triple if protocol work needed |
| A-5 | All terminals support Unicode block characters (█/░) | Multiple tickets | Reasonable for modern terminals | Garbled rendering on legacy terminals |
| A-6 | `toastH` dynamic height pattern is stable | UX-017 (implicit) | **Verified** at `layout.go:144-146` | Pattern failure would indicate broader layout issues |
| A-7 | `ToolBlock.Expanded` defaults to true currently | UX-015 | Needs verification | If already false, UX-015 scope shrinks further |

### Layer 2: Dependency Mapping

**Verified dependency chains:**
- UX-003 → UX-008 → UX-009/UX-010: Correct. Icon rail establishes `RenderMode`, dot-leaders build on it, colors/cost build on dot-leaders.
- UX-020 → UX-023, UX-025: Correct. Reduce-motion must exist before animations ship.
- UX-008 → UX-022: Correct. Density toggle cycles through layouts that dot-leaders define.

**Violations found:**
- **UX-016 → UX-024**: Wrong dependency AND phase ordering violation (P2 → P3). Fix: remove.
- **No other cross-phase violations** found in remaining 27 tickets.

**Hidden dependencies (not in tickets):**
- UX-017 (adaptive status line) implicitly affects UX-005 and UX-006 — status line enhancements in P0 must render correctly in both 1-row and 2-row modes when UX-017 ships in P2. Not blocking, but P0 implementers should use `lipgloss.Width()`-based layout, not hard-coded column positions.
- UX-007 (simple toggle) affects all right-panel tickets — when panel is hidden, icon rail, tree overhaul, etc. are simply not rendered. No code dependency, but acceptance tests should verify graceful no-op in simple mode.

**Orphan check:** No orphan tickets found. All 28 tickets connect to the spec and serve the overall goal.

### Layer 3: Failure Mode Analysis

**Phase 1 (P0) failure modes:**
- UX-003 (icon rail) fails 50% through: Partial `Render(mode)` exists but `View()` still works via default `RenderFull` mode. **Recoverable.** Tree renders normally at all widths, icon rail simply doesn't activate.
- UX-007 (simple toggle) ships with `alt+p`: Conflicts with `CyclePermMode`. Users lose permission mode cycling. **Not recoverable without hotfix.** Must fix RED-2 before ship.

**Phase 2 (P1) failure modes:**
- UX-008 (dot-leaders) has ANSI width miscalculation: Garbled tree display at specific widths. Caught by boundary tests if YELLOW-2 mitigation is applied. **Recoverable** — revert to non-dot-leader rendering.

**Phase 3 (P2) failure modes:**
- UX-017 (adaptive status line) returns wrong height: Layout overflow — content area shrinks or expands incorrectly. **Caught by invariant test.** If invariant test is missing, this is a silent layout bug visible only at specific widths.

**Cascade risk:** The highest-risk cascade is UX-003 → UX-008 → UX-009/010. If `Render(mode RenderMode)` architecture is fundamentally wrong, all downstream tree tickets need rework. **Mitigation:** Get UX-003 PR reviewed and merged before starting UX-008. The architecture decision (Q1) is the single most important pre-implementation gate.

**Rollback:** Every phase can be independently reverted. No database migrations, no protocol changes (toasts already work), no external API dependencies. **Rollback is clean for all phases.**

### Layer 4: Cost-Benefit Assessment

**Total effort:** 12-14.5 sessions across 4 phases. After corrections: ~11-13 sessions (UX-015 downscope saves ~0.25-0.5s).

**Phase-by-phase ROI:**
| Phase | Effort | Benefit | ROI |
|-------|--------|---------|-----|
| P0 | 3-3.5s | Universal readability (3a, 3b), core differentiator (1a), PMF toggle (N1) | **Highest** — every user benefits |
| P1 | 2-3s | Tree legibility, team status, onboarding | **High** — addresses top user friction |
| P2 | 3-4s (→ 2.75-3.5s after UX-015 downscope) | Conversation polish, monitoring, accessibility | **Medium** — progressive enhancement |
| P3 | 3-4s | Animation, advanced features, research | **Low** — diminishing returns, but validates design |

**YAGNI check:** No tickets flagged as unnecessary. All 28 serve the spec's stated goals. The 3 Braintrust additions (UX-007, UX-012, UX-020) are minimal-cost, high-option-value additions — appropriately scoped.

**Complexity budget:** UX-003 (icon rail) and UX-008 (dot-leaders) are the complexity-heavy tickets. Together they represent ~2.5-3s of the 12-14.5s total — appropriate for the core architectural changes.

### Layer 5: Testing Coverage

**Strengths:**
- Boundary test matrix specified: `{15, 22, 28-32, 45, 60, 80, 120, 180}` cols — excellent coverage
- UX-017 has height invariant assertion
- UX-003 specifies `lipgloss.Width()` assertions at boundary widths
- Multiple tickets reference existing test patterns (e.g., `TabFlashMsg`)

**Gaps:**
1. **Non-agent panel behavior at narrow widths:** No test coverage specified for what happens to Dashboard, Settings, Telemetry, PlanPreview, Teams panels when `rightWidth < 30`. Even with icon rail scoped to agents-only, the question remains: do these panels gracefully degrade? Not a testing gap for this spec (these panels have their own View() methods), but worth noting.
2. **Simple mode interaction tests:** UX-007 doesn't specify what happens when simple mode is toggled while icon rail is active. Should icon rail state be preserved or reset?
3. **UX-015 missing existing-infrastructure tests:** No AC verifies that existing `ToggleToolExpansion`/`CycleExpansion` still work after enhancement. Regression risk.
4. **UX-008 missing ANSI safety test helper:** YELLOW-2 recommends `lipgloss.Width(renderedRow) == expectedWidth` helper. Not yet in UX-008's AC.

**Recommendation:** Add items 2 and 4 to respective ticket ACs. Items 1 and 3 are minor.

### Layer 6: Architecture Smell Detection

**No God Components:** Changes are well-distributed across 15+ files. No single ticket creates a >500 LoC component.

**No premature abstraction:** `RenderMode` is justified — it prevents the dual-path drift that Risk #3 identifies. It's an abstraction that serves a real, immediate need.

**No leaky abstraction:** `Render(mode)` is internal to each component. `View()` remains the public Bubbletea interface. The mode is computed from width, not leaked to callers.

**Potential coupling concern:** UX-011 (team indicator) populates status line fields from `teamsHealth` via poll tick. This creates a coupling between status line model and teams health model. Acceptable for now — the alternative (event-based) is overengineering for a status display. Revisit if team state becomes more complex.

**Branch consolidation:** The tree-overhaul branch strategy (UX-003+008+009+010) is architecturally clean. panel.go cross-phase risk (NEW-1) is manageable with ordered merging.

### Layer 7: Contractor Readiness

**Monday-morning ready (zero questions):**
- UX-001: Clear, atomic, self-contained. Acceptance criteria unambiguous.
- UX-002: Clear, references exact line numbers for existing styles.
- UX-005: Explicitly states "enhancement only, DO NOT reimplement." References `renderContextBar()` at `statusline.go:228`.
- UX-006: Same quality as UX-005. References `costStyle()` at `statusline.go:476`.

**Ready after minor fixes:**
- UX-003: Needs Q1/Q2 decisions documented in ticket. Add: "agents-only scope, use `Render(mode RenderMode)` unified method."
- UX-018: Needs toast protocol reference added. Otherwise clear.

**NOT ready (must fix before handoff):**
- UX-004: Wrong file reference (RED-1). Fix `activity.go` → `agent_sync.go`.
- UX-007: Keybinding conflict (RED-2). Fix `alt+p` → `alt+\\` or chosen alternative.
- UX-015: Written as greenfield, should be enhancement. Rewrite with existing code references.
- UX-022: Missing keybinding conflict note.
- UX-012: Vague file paths (directories, not files).

---

## Ticket-Level Notes

| Ticket | Status | Notes |
|--------|--------|-------|
| UX-001 | READY | Exemplary ticket — clear, atomic, correct references |
| UX-002 | READY | No issues |
| UX-003 | FIX NEEDED | Add Q1/Q2 decisions (Render method, agents-only scope). Currently missing explicit scope statement for `rightPanelMode` |
| UX-004 | FIX NEEDED | RED-1: Replace `activity.go` → `agent_sync.go` |
| UX-005 | READY | Good enhancement-only framing. References correct function |
| UX-006 | READY | Good enhancement-only framing |
| UX-007 | FIX NEEDED | RED-2: Replace `alt+p` with non-conflicting binding |
| UX-008 | FIX NEEDED | Add ANSI safety test helper to AC per YELLOW-2 |
| UX-009 | READY | Simple, correct dependencies |
| UX-010 | READY | Simple, correct dependencies |
| UX-011 | READY | Clear spec, correct file references |
| UX-012 | FIX NEEDED | Replace directory paths with specific filenames |
| UX-013 | READY | Follows existing collapsible pattern |
| UX-014 | READY | References existing `ToolUseMsg` and `spinnerTickMsg` patterns |
| UX-015 | FIX NEEDED | Rewrite as enhancement-only. Reference `renderToolBlock()` at `panel.go:1006`. Replace `tool_use_id` → `ToolID`. Reduce effort |
| UX-016 | FIX NEEDED | Remove UX-024 dependency (YELLOW-4 / phase ordering violation) |
| UX-017 | READY | Implementation note about circular dependency is correct. Risk lower than YELLOW-5 suggested |
| UX-018 | FIX NEEDED | Add toast protocol reference (`bridge/server.go:327`). Note spawn.go needs emission, not protocol extension |
| UX-019 | READY | Correct use of existing `TabFlashMsg` pattern |
| UX-020 | READY | Simple boolean config + conditional check |
| UX-021 | READY | References correct function `computeDrawerLayout()` |
| UX-022 | FIX NEEDED | Add keybinding conflict note (YELLOW-7). Specify resolution strategy |
| UX-023 | READY | Correct dependency on UX-020 |
| UX-024 | READY | Straightforward |
| UX-025 | READY | Correct dependency on UX-020, opt-in approach appropriate |
| UX-026 | READY | Good spec with code example |
| UX-027 | READY | Edge cases (empty, overflow, vim, dismiss) captured in AC |
| UX-028 | READY | Straightforward |

**Summary:** 18 READY, 10 need fixes (2 critical, 4 major, 4 minor). All fixes are corrections to existing tickets, not architectural changes.

---

## Effort Estimate Corrections

| Ticket | Current | Corrected | Reason |
|--------|---------|-----------|--------|
| UX-015 | 0.75s | 0.25-0.5s | Enhancement-only — collapse infrastructure already exists at `panel.go:1006-1037` |
| UX-012 | 0.25s | 0.5s | Session counting, action-based dismissal, multiple hint definitions, config persistence — more than 0.25s of work |
| UX-017 | 0.5s | 0.5s | No change — risk is lower than feared (toast precedent), but invariant test still needed |
| UX-003 | 1.5-2s | 1.5-2s | No change — agents-only scope keeps estimate valid |

**Net effect:** -0.25 to 0s (UX-015 savings offset by UX-012 increase). Total remains ~12-14.5 sessions.

---

## Final Conditions (Must Be True Before Implementation Begins)

1. **[CRITICAL] Fix RED-1:** UX-004 file reference updated from `activity.go` to `agent_sync.go`. Also update spec Sections 3.3, 8, and 12.

2. **[CRITICAL] Fix RED-2:** UX-007 keybinding changed from `alt+p` to a non-conflicting alternative (recommend `alt+\\`). Run keybinding audit of `config/keys.go` to confirm chosen slot is free.

3. **[MAJOR] Rewrite UX-015** as enhancement-only with existing code references (`panel.go:1006-1037`, `ToggleToolExpansion`, `CycleExpansion`). Replace `tool_use_id` → `ToolID`. Reduce effort to 0.25-0.5s.

4. **[MAJOR] Fix UX-016 dependency:** Remove `UX-024` from dependencies. Update `tickets-index.json` line 205-206.

5. **[DECISION] Document Q1/Q2 answers in UX-003:** "Use `Render(mode RenderMode)` unified method. Icon rail applies to `RPMAgents` mode only."

6. **[MINOR] Apply remaining ticket fixes:** UX-008 (add ANSI test helper to AC), UX-012 (specific file paths), UX-018 (toast protocol reference), UX-022 (keybinding conflict note).

---

## Sign-Off

### Phase 1 (P0): GO (after conditions 1, 2, 5 met)

UX-001 and UX-002 can start immediately — they are zero-dependency, zero-controversy, maximum-impact tickets. UX-003 can start once Q1/Q2 decisions are documented. UX-004 and UX-007 can start once their RED flags are fixed. UX-005 and UX-006 are ready now.

**Recommended implementation order within P0:**
1. UX-001 + UX-002 (conversation branch, parallel, 0.5s total)
2. UX-005 + UX-006 (statusline branch, parallel, 0.5s total)
3. UX-007 (toggle branch, 0.25s, after RED-2 fix)
4. UX-003 + UX-004 (icon-rail branch, 1.5-2.25s, after RED-1 fix + Q1/Q2 decisions)

### Phase 2 (P1): GO (after condition met, P0 UX-003 merged)

Tree-overhaul branch (UX-008+009+010) depends on UX-003's `Render(mode)` architecture. Start when UX-003 PR is merged. UX-011 and UX-012 are independent — can start in parallel with P0.

### Phase 3 (P2): GO (after conditions 3, 4 met)

UX-015 must be rewritten before implementation. UX-016 dependency must be fixed. Remaining P2 tickets are ready.

### Phase 4 (P3): GO (after condition 6 met for UX-022)

All P3 tickets are ready or have minor fixes. UX-022 keybinding resolution can be decided during P2 implementation.

---

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-04-11
**Tickets Reviewed:** 28 (11 spot-checked in detail, 17 verified via index and preflight)
**Source Files Verified:** 5 (statusline.go, layout.go, keys.go, agent_sync.go, panel.go)
**Issue Counts:** 2 Critical, 4 Major, 5 Minor, 5 Commendations

### Commendations

1. **Excellent enhancement-only framing on UX-005 and UX-006.** Explicitly stating "DO NOT reimplement" with existing function references is exactly what prevents contractor scope creep (Risk #5). This pattern should be applied to UX-015.
2. **Braintrust synthesis quality.** The 3 additions (toggle, hints, reduce-motion) are precisely scoped — minimal cost, maximum option value. No over-engineering.
3. **Preflight review thoroughness.** Catching the `activity.go` ghost file and `alt+p` conflict before implementation saves real contractor hours. The GREEN flag verification (checking file existence, function signatures, patterns) is above-average diligence.
4. **Phase ordering.** P0 leads with universal-benefit changes (conversation readability) per Einstein's recommendation. This means the first deliverable benefits ALL users, not just power users. Good prioritization.
5. **Risk register quality.** The 7-risk register in `overview.md` is honest and specific. Each risk has a concrete mitigation. No hand-waving.

---

_Generated: 2026-04-11_
_Review cost: ~$1.50 (context reads + analysis)_
_Methodology: 7-layer Staff-Architect Critical Review framework_
