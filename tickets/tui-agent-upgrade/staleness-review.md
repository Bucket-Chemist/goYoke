# TUI Agent Upgrade Tickets — Staleness Review

**Reviewed:** 2026-03-28
**Reviewer:** Staff Architect Critical Review
**Scope:** TUI-001 through TUI-008 vs. current codebase state
**Project Root:** `/home/doktersmol/Documents/GOgent-Fortress`

---

## Executive Summary

**Overall Verdict:** APPROVE_WITH_CONDITIONS

Most ticket references are accurate. There are **no blocking structural mismatches** — all file paths exist (except one), all core types/functions are correctly referenced, and the dependency ordering is sound. However, there are **8 concrete discrepancies** that would cause confusion or incorrect test assertions during implementation. These must be corrected before handing tickets to an implementer.

**Issue Counts:**
- Critical: 0
- Major: 3 (would cause implementation failure or wrong code)
- Minor: 5 (would cause confusion but not failure)

---

## Per-Ticket Findings

### TUI-001: Extend teamconfig.Member with health monitoring fields

**VERIFIED:**
- `internal/teamconfig/config.go` — EXISTS. `Member` struct at lines 28-37.
- `internal/tui/components/teams/state.go` — EXISTS. `copyOf()` at line 42.
- `cmd/gogent-team-run/config.go` — EXISTS. Runner `Member` at lines 46-67.
- `CompletedAt` is the last field in `teamconfig.Member` (line 36) — correct insertion point.
- Runner's `ProcessPID *int` has `json:"process_pid"` (no omitempty) at line 53 — matches ticket spec.
- Runner's `HealthStatus`, `LastActivityTime`, `StallCount`, `KillReason` field types and tags at lines 64-67 — all match ticket spec exactly.
- `copyOf()` on `TeamState` (line 42) does shallow `copy()` of Members slice — pointer fields (`ProcessPID *int`, `LastActivityTime *string`) would share underlying data. Deep copy is correctly identified as needed.

**STALE:** None.

**RISK:** None. This ticket is clean.

---

### TUI-002: Promote drawers from right-panel to full-width layout

**VERIFIED:**
- `internal/tui/model/layout.go` — EXISTS. `computeLayout()` at line 111, `renderMain()` at line 295, `renderRightPanel()` at line 333.
- `internal/tui/model/ui_event_handlers.go` — EXISTS. `handleWindowSize()` at line 23.
- `internal/tui/model/layout_test.go` — EXISTS. 254 lines — line count matches.
- `layoutDims` struct at layout.go:78 with `tier` (line 93) and `contentHeight` (line 85) fields.
- Drawer compositing in `renderRightPanel()` at lines 374-420 (ticket says "~367-417" — close enough for an approximation).
- `m.shared.drawerStack` — EXISTS as `drawerStackWidget` in `sharedState` (app.go:120).
- `OptionsHasContent()` — EXISTS in `drawerStackWidget` interface (interfaces.go:426).
- `PlanHasContent()` — EXISTS in `drawerStackWidget` interface (interfaces.go:429).

**STALE:**

| # | Ticket Claim | Actual Code | Severity |
|---|-------------|-------------|----------|
| S-1 | "8 existing tests for `computeLayout`" | **6 test functions**: TestLayoutTierString, TestComputeLayout_TierBoundaries, TestComputeLayout_WideTerminal_Uses60_40, TestComputeLayout_UltraTerminal_Uses50_50, TestComputeLayout_Standard_75_25_At80, TestComputeLayout_Standard_70_30_At100 | **Minor** |
| S-2 | Size propagation happens in `handleWindowSize()` | Size propagation is in `propagateContentSizes()` (ui_event_handlers.go:83), called BY handleWindowSize(). The ticket's code sketch shows SetSize calls as if directly in handleWindowSize — implementer should modify `propagateContentSizes()` instead. | **Major** |

**RISK:** S-2 could cause an implementer to duplicate size propagation logic in handleWindowSize rather than modifying propagateContentSizes, leading to stale sizes in one path.

---

### TUI-003: Add teamsHealthWidget interface and sharedState wiring

**VERIFIED:**
- `internal/tui/model/interfaces.go` — EXISTS. Pattern matches: `dashboardWidget` (line 180), `settingsWidget` (line 202) both define `View() string`, `SetSize(w, h int)`, `SetTier(tier LayoutTier)`.
- `internal/tui/model/app.go` — EXISTS. `sharedState` struct at line 95.
- `LayoutTier` type — EXISTS at layout.go:34.
- Setter pattern in setters.go: `SetDashboard` (line 78), `SetSettings` (line 82), `SetTelemetry` (line 87), `SetPlanPreview` (line 92) — all follow the exact `func (m *AppModel) SetX(x xWidget)` pattern.
- `AppModel` struct at app.go:172 — field additions are straightforward.

**STALE:** None.

**RISK:** None. This ticket is clean. All interface patterns verified.

---

### TUI-004: Extend team polling to capture stream file sizes

**VERIFIED:**
- `internal/tui/components/teams/state.go` — EXISTS. `TeamState` at line 31, `TeamRegistry` at line 93.
- `internal/tui/components/teams/list.go` — EXISTS. `scanTeamsDir()` at line 291.
- `internal/tui/components/teams/state_test.go` — EXISTS (244 lines, 15 test functions).
- `internal/tui/components/teams/list_test.go` — EXISTS (489 lines, 20+ test functions).
- `copyOf()` on `TeamState` at state.go:42 — correctly identified for StreamSizes deep copy.

**STALE:**

| # | Ticket Claim | Actual Code | Severity |
|---|-------------|-------------|----------|
| S-3 | `Update()` accepts `streamSizes` parameter | Current signature is `func (r *TeamRegistry) Update(dir string, config TeamConfig)` (state.go:107). Adding a parameter is a breaking change. `scanTeamsDir()` at list.go:314 calls `reg.Update(teamDir, cfg)` — this call site must be updated too. The ticket mentions modifying scanTeamsDir to "stat stream files" but doesn't explicitly state the Update() call change at line 314. | **Minor** |

**RISK:** Low. The ticket implicitly covers the scanTeamsDir modification ("stat stream files in scanTeamsDir() (~15 lines)"). An attentive implementer will update the call site. But making it explicit would be safer.

---

### TUI-005: Implement TeamsHealthModel component

**VERIFIED:**
- `internal/tui/components/teams/health.go` — correctly marked **NEW** (does not exist).
- `internal/tui/components/teams/health_test.go` — correctly marked **NEW** (does not exist).
- `config.StyleSuccess` — EXISTS at theme.go:215.
- `config.StyleWarning` — EXISTS at theme.go:219.
- `config.StyleError` — EXISTS at theme.go:210.
- `config.StyleMuted` — EXISTS at theme.go:223.
- `TeamRegistry` — EXISTS at state.go:93.
- `MostRecentRunning()` — correctly identified as created by TUI-004 (dependency).
- `RLock + copyOf()` pattern — matches existing `Get()` and `All()` methods in TeamRegistry.
- `teamsHealthWidget` interface — correctly identified as created by TUI-003 (dependency).

**STALE:** None.

**RISK:** None. Dependencies correctly chained (TUI-001, TUI-003, TUI-004).

---

### TUI-006: Right panel 50/50 split for Wide/Ultra tiers

**VERIFIED:**
- `internal/tui/model/layout.go` — EXISTS.
- `internal/tui/model/ui_event_handlers.go` — EXISTS.
- `internal/tui/model/layout_test.go` — EXISTS.
- `RPMAgents` — EXISTS at focus.go:110.
- `renderRightPanel()` — EXISTS at layout.go:333.
- `agentTree` / `agentDetail` — EXISTS as fields on `AppModel` (app.go:191-192).
- `lipgloss.JoinVertical` / `JoinHorizontal` — both used in current layout code.
- `LayoutWide` — EXISTS at layout.go:46.
- `dims.rightWidth` — EXISTS at layoutDims.rightWidth (layout.go:81).
- Current `RPMAgents` case at layout.go:338-341 matches the code sketch's starting point.

**STALE:** None — but see cross-ticket note below about TUI-002 interaction.

**RISK:** After TUI-002 promotes drawers to full-width, the `dims.contentHeight` used inside `renderRightPanel()` will reflect post-drawer-subtraction height (assuming TUI-002 modifies `propagateContentSizes()`). The code sketch uses `dims.contentHeight` directly, which should be correct if TUI-002 is implemented as spec'd. However, if TUI-002 subtracts drawer height at the render level rather than the sizing level, the split heights could be wrong. Implementer should verify this interaction.

---

### TUI-007: Standard tier Alt+H toggle for health view

**VERIFIED:**
- `internal/tui/model/key_handlers.go` — EXISTS.
- `internal/tui/model/layout.go` — EXISTS.
- `rightPanelMode` — EXISTS as field on AppModel (app.go:180).
- `RPMAgents` — EXISTS at focus.go:110.
- `NextRightPanelMode()` — EXISTS at focus.go:149. Called at key_handlers.go:122 via `CycleRightPanel` keybinding.
- `config.StyleMuted` — EXISTS at theme.go:223.

**STALE:**

| # | Ticket Claim | Actual Code | Severity |
|---|-------------|-------------|----------|
| S-4 | "Use `config.StyleFocused` for active tab" | **`config.StyleFocused` does not exist.** Available styles: `config.StyleFocusedBorder` (border style, not text), `config.StyleHighlight` (theme.go:205, text highlight), `config.StyleSubtle` (theme.go:201). The intended style for an active tab indicator is most likely **`config.StyleHighlight`**. | **Major** |

**RISK:** S-4 will cause a compile error. Must be fixed before implementation.

---

### TUI-008: Wire health dashboard in main.go and extend layout tests

**VERIFIED:**
- `internal/tui/model/layout_test.go` — EXISTS. 254 lines — line count matches.
- `TestComputeLayout_TierBoundaries` — EXISTS at layout_test.go:46.
- Wiring pattern: `SetDashboard`, `SetSettings`, `SetTelemetry`, `SetPlanPreview` all exist in setters.go and are called at main.go:221-230.
- `NewTeamListModel(teamReg)` at main.go:137 — the constructor pattern for `NewTeamsHealthModel` would follow this.

**STALE:**

| # | Ticket Claim | Actual Code | Severity |
|---|-------------|-------------|----------|
| S-5 | "`main.go` or `cmd/gogent-tui/main.go`" | **`cmd/gogent-tui/` does not exist.** The TUI entry point is **`cmd/gofortress/main.go`**. | **Major** |
| S-6 | "8 tests" (same as TUI-002) | **6 test functions** (see S-1). | **Minor** |
| S-7 | "TestComputeLayout_TierBoundaries (12 table-driven cases)" | **14 table-driven cases** (compact_boundary_79, compact_mid_60, compact_min_1, standard_lower_80, standard_mid_90, standard_upper_99, standard_lower_100, standard_mid_110, standard_upper_119, wide_lower_120, wide_mid_149, wide_upper_179, ultra_lower_180, ultra_mid_240). | **Minor** |
| S-8 | Code sketch uses `teamRegistry` variable | main.go:136 uses **`teamReg`** as the variable name: `teamReg := teams.NewTeamRegistry()` | **Minor** |

**RISK:** S-5 is the most significant — an implementer looking for `cmd/gogent-tui/main.go` will waste time. The correct file and exact wiring location (after line 138, following the `app.SetTeamList(&teamList)` call) should be specified.

---

## Cross-Ticket Issues

### 1. Shared stale test count (S-1, S-6)

TUI-002, TUI-006, and TUI-008 all claim `layout_test.go` has "8 tests". The actual count is **6 test functions**. This is repeated across three tickets and would be confusing since the acceptance criteria say "All 8 existing layout_test.go tests continue to pass."

**Fix:** Update all three tickets to say "6 existing tests."

### 2. propagateContentSizes() vs handleWindowSize() (S-2)

TUI-002 and TUI-006 both reference modifying `handleWindowSize()` for size propagation. The actual size propagation code lives in `propagateContentSizes()` (ui_event_handlers.go:83-140), which is called by `handleWindowSize()`. Both tickets should reference `propagateContentSizes()` as the function to modify.

### 3. TUI-002 + TUI-006 height coordination

TUI-002 subtracts drawer height from `contentHeight` before passing to panel SetSize calls. TUI-006's code sketch uses `dims.contentHeight` for the 50/50 split. After TUI-002, this value in `propagateContentSizes()` will already be post-drawer-subtraction. But inside `renderRightPanel()`, `dims.contentHeight` comes from `computeLayout()` which does NOT subtract drawer height — that happens elsewhere. The implementer must verify which height value flows into the split calculation.

### 4. TUI-004 Update() signature change ripple

TUI-004 changes `TeamRegistry.Update()` to accept a `streamSizes` parameter. The callers are:
1. `scanTeamsDir()` at list.go:314
2. Tests in state_test.go (lines 48, 64, 83, 107, 212, 236)
3. Tests in list_test.go (lines 121, 183, 211, 226-228, 254, 305, 350, 433, 470)

The ticket mentions updating tests but should explicitly enumerate all call sites to prevent missed updates.

---

## Recommended Ticket Updates

### TUI-002
1. Change "8 existing tests" → "6 existing tests" in acceptance criteria and review notes.
2. Add note: "Size propagation lives in `propagateContentSizes()` (ui_event_handlers.go:83), not directly in `handleWindowSize()`. Modify propagateContentSizes to subtract drawer height before passing to panel SetSize calls."

### TUI-004
1. Add explicit note: "Call sites for `Update()` that need signature update: `scanTeamsDir()` at list.go:314, plus 8 calls in state_test.go and 10+ calls in list_test.go."

### TUI-007
1. Change `config.StyleFocused` → `config.StyleHighlight` (theme.go:205) for the active tab indicator. Alternatively use a custom lipgloss.Style with `config.ColorPrimary` foreground.

### TUI-008
1. Change "`main.go` or `cmd/gogent-tui/main.go`" → **"`cmd/gofortress/main.go`"** (the actual TUI entry point).
2. Add: "Wire after `app.SetTeamList(&teamList)` at line 138, following the existing `teamReg` variable (NOT `teamRegistry`)."
3. Change "8 tests" → "6 tests."
4. Change "12 table-driven cases" → "14 table-driven cases."

### TUI-006
1. Change "8 existing tests" → "6 existing tests" (if referenced in acceptance criteria).

---

## Consolidated Discrepancy Table

| ID | Ticket(s) | Claim | Reality | Severity | Fix |
|----|-----------|-------|---------|----------|-----|
| S-1 | TUI-002, TUI-008 | "8 existing tests" in layout_test.go | 6 test functions | Minor | Update count |
| S-2 | TUI-002 | Size propagation in handleWindowSize() | In `propagateContentSizes()` (ui_event_handlers.go:83) | Major | Redirect to correct function |
| S-3 | TUI-004 | Update() signature change | Breaking change — 20+ call sites across state_test.go, list_test.go, list.go | Minor | List all call sites |
| S-4 | TUI-007 | `config.StyleFocused` | Does not exist; use `config.StyleHighlight` | Major | Fix style name |
| S-5 | TUI-008 | `cmd/gogent-tui/main.go` | Does not exist; correct path is `cmd/gofortress/main.go` | Major | Fix file path |
| S-6 | TUI-008 | "8 tests" (duplicate of S-1) | 6 test functions | Minor | Update count |
| S-7 | TUI-008 | "12 table-driven cases" | 14 cases in TestComputeLayout_TierBoundaries | Minor | Update count |
| S-8 | TUI-008 | `teamRegistry` variable in code sketch | Variable is `teamReg` in main.go:136 | Minor | Fix variable name |

---

## Overall Assessment

**Can these tickets be implemented as-is?**

**No — but close.** Three major issues require fixes before implementation:

1. **S-2 (TUI-002):** An implementer following the ticket literally would modify handleWindowSize() rather than propagateContentSizes(), causing size propagation to break in other code paths (e.g., taskboard toggle).
2. **S-4 (TUI-007):** `config.StyleFocused` will cause a compile error. Trivial fix but must be corrected.
3. **S-5 (TUI-008):** `cmd/gogent-tui/main.go` does not exist. Implementer won't find the file.

The 5 minor issues (wrong counts, variable name) are cosmetic but should be corrected to prevent wasted investigation time.

**Recommendation:** Apply the 8 fixes listed above, then these tickets are ready for implementation. Total fix effort: ~15 minutes of ticket editing. No architectural or structural changes needed — the design is sound.
