# Preflight Review: UX Redesign Tickets

**Reviewer:** Router (automated sanity check)
**Date:** 2026-04-11
**Scope:** 28 tickets (UX-001 through UX-028)
**Purpose:** Red/Yellow/Green flag audit before Staff-Architect final pass

---

## Summary

| Flag | Count | Tickets |
|------|-------|---------|
| RED (blocking) | 2 | UX-004, UX-007 |
| YELLOW (concern) | 7 | UX-003, UX-008, UX-015, UX-016, UX-017, UX-018, UX-022 |
| GREEN (good to go) | 19 | UX-001, UX-002, UX-005, UX-006, UX-009, UX-010, UX-011, UX-012, UX-013, UX-014, UX-019, UX-020, UX-021, UX-023, UX-024, UX-025, UX-026, UX-027, UX-028 |

**Verdict:** 2 red flags require correction before implementation. Both are straightforward fixes (wrong file path, keybinding conflict). No architectural blockers.

---

## RED Flags (Must Fix)

### RED-1: UX-004 — Wrong file reference

**Ticket:** UX-004 (Activity: relative paths)
**Issue:** References `internal/tui/cli/activity.go` — this file does not exist.
**Actual location:** Activity extraction lives in `internal/tui/cli/agent_sync.go`, specifically `ExtractToolActivity()` at line 231 and `AgentActivity` struct construction at line 238.
**Fix:** Replace `activity.go` with `cli/agent_sync.go` in the ticket's Files section. Update the spec's Section 3.3 file list as well.
**Also affects:** UX-REDESIGN-SPEC.md Section 3.3, Section 8 priority matrix, Section 12 ticket index.

### RED-2: UX-007 — Keybinding conflict

**Ticket:** UX-007 (Simple/expert toggle)
**Issue:** Proposes `alt+p` as the toggle keybinding. `alt+p` is already bound to "cycle perm mode" at `config/keys.go:203-204`.
**Fix:** Choose a different keybinding. Candidates:
- `alt+\\` (toggle panel — backslash mnemonic for "split")
- `alt+]` (right panel — bracket mnemonic)
- `ctrl+\\` (if not taken)
- Or add to the existing `alt+shift+` chord namespace
**Verification needed:** Run a full keybinding audit of `config/keys.go` for available slots before deciding.

---

## YELLOW Flags (Concern — Verify Before Starting)

### YELLOW-1: UX-003 — Effort estimate may still be low

**Ticket:** UX-003 (Icon rail mode)
**Concern:** Estimated at 1.5-2 sessions. This ticket introduces:
- A new `RenderMode` type and unified `Render(mode)` method (architectural decision)
- Hysteresis logic at 28/32 boundaries
- Compact rendering for both tree.go (361L) and detail.go (625L)
- Boundary tests at 8+ widths
- Interaction with `renderRightPanel()` which has a `rightPanelMode` switch (layout.go:534)

The `rightPanelMode` switch at layout.go:534 means the icon rail must work across multiple panel modes (agents, config, teams), not just the agent tree. This multiplies the surface area.

**Recommendation:** Verify whether icon rail applies only to agent panel mode or all `rightPanelMode` values. If all modes, effort is closer to 2.5-3 sessions.

### YELLOW-2: UX-008 — Dot leader ANSI safety not trivial

**Ticket:** UX-008 (Two-column dot-leader layout)
**Concern:** Dot leader width calculation using `lipgloss.Width()` is correct for measuring rendered strings, but the construction logic must also account for:
- Status icons (●, ✕, ◻) which are multi-byte but single-width
- Agent names that may contain Unicode (though unlikely)
- The interaction between `lipgloss.Width()` for measurement and `len()` for string construction

`lipgloss.Width()` uses `ansi.StringWidth()` internally which handles ANSI escape codes, but the dot count calculation `dotsNeeded = w - depth*2 - 2 - nameWidth - valueWidth - 1` mixes measured widths with arithmetic offsets. Off-by-one errors at boundary widths are likely.

**Recommendation:** Require a dedicated test helper that verifies `lipgloss.Width(renderedRow) == expectedWidth` for every width tier. Add this to acceptance criteria.

### YELLOW-3: UX-015 — Tool blocks already have collapse infrastructure

**Ticket:** UX-015 (Collapsible tool-use blocks)
**Concern:** The ticket describes this as net-new work, but `panel.go` already has:
- `ToolBlock.Expanded` field (toggleable)
- `renderToolBlock()` at line 1006 with collapsed/expanded rendering
- Collapsed view shows tool name, expanded shows full content
- Expansion state is explicitly reset on restore (lines 1148-1182)

This is closer to an **enhancement** than greenfield. The existing `CycleExpansion` keybinding (`alt+shift+e`, config/keys.go:284) already cycles tool expansion.

**Recommendation:** Rewrite ticket as enhancement-only (similar to how UX-005/UX-006 were downscoped). Reduce effort from 0.75s to 0.25-0.5s. Verify: does the current collapse default to collapsed (desired) or expanded?
**Also:** Ticket says "use `tool_use_id` as collapse state key" — the codebase uses `ToolID` (field name on `ToolBlock`). Update terminology.

### YELLOW-4: UX-016 — Dependency may be wrong

**Ticket:** UX-016 (Agent count sparkline dots)
**Concern:** Listed dependency is UX-024 (timestamp gutter). This makes no sense — sparkline dots in the status line have nothing to do with conversation timestamps.
**Fix:** Remove the UX-024 dependency. The ticket should depend on nothing, or optionally on UX-020 (reduce-motion) if we consider the dots "animation" — but static dots don't animate, so no dependency needed.

### YELLOW-5: UX-017 — statusLineHeight is currently a hardcoded constant

**Ticket:** UX-017 (Adaptive 1-row status line)
**Concern:** The ticket says to implement `statusLineHeight` as "a method on AppModel computing from m.width directly." Currently, `statusLineHeight` is a package-level constant `= 2` at layout.go:41, used in the layout calculation at line 160:
```go
dims.contentHeight = m.height - bannerHeight - tabBarHeight - providerTabH - statusLineHeight - taskBoardH - toastH - hintH - bcH - borderFrame
```
Changing this from a constant to a dynamic method is riskier than the ticket implies because:
1. It's used in a single expression with 8 other height terms
2. If the method returns a value that disagrees with the actual rendered height, layout breaks
3. There's no existing pattern for dynamic heights — all other terms (bannerHeight, tabBarHeight) are also constants

**Recommendation:** The ticket's acceptance criterion "lipgloss.Height(statusLine.View()) == computed height" is the right safety check, but flag this as higher risk than "0.5s" suggests. Consider 0.75s with dedicated integration test.

### YELLOW-6: UX-018 — spawn.go is complex, toast integration unclear

**Ticket:** UX-018 (Action-hinted toasts)
**Concern:** References `cmd/gogent-team-run/spawn.go` (1017 lines) for emitting lifecycle toasts. But spawn.go runs as a separate process — it communicates with the TUI via UDS (Unix Domain Socket), not direct function calls. Toast emission from spawn.go requires:
1. The UDS message protocol to support toast payloads
2. The TUI to parse and render toast messages from the team-run process

The ticket's "Files to modify" lists `tools.go` and `spawn.go` but doesn't mention the UDS protocol layer. If toasts are already supported over UDS, this is simple. If not, it's a protocol extension.

**Recommendation:** Verify whether the UDS protocol between gogent-team-run and the TUI already supports toast messages. If not, the effort is higher and needs an additional file (the UDS message handler).

### YELLOW-7: UX-022 — Depends on UX-008 but says UX-008

**Ticket:** UX-022 (Tree density toggle)
**Concern:** Minor — dependency is correctly listed as UX-008 (dot-leader layout), but the ticket says `alt+shift+e` for the keybinding. This keybinding is already used for `CycleExpansion` in the Claude panel (config/keys.go:284). The density toggle would need either:
- A different keybinding when tree panel is focused (context-sensitive), or
- A new keybinding entirely

**Recommendation:** Clarify whether the keybinding is context-sensitive (tree focus = density, claude focus = expansion) or needs a unique binding.

---

## GREEN Flags (Good to Go)

### UX-001 — Conversation: horizontal rule between turns
- Files exist: panel.go (1264L) ✓
- Role transition already detected at line 925-927 (userRoleStyle/assistantRoleStyle switch) ✓
- Simple string insertion between turns ✓
- No dependencies ✓

### UX-002 — Conversation: user/assistant color differentiation
- `userRoleStyle` at line 53, `assistantRoleStyle` at line 58 confirmed ✓
- Currently style the "You:"/"Claude:" prefix only, not content text ✓
- Extending to content text is straightforward ✓

### UX-005 — Status line: enhance existing context bar
- `renderContextBar()` confirmed at statusline.go:228 ✓
- Already has semantic color thresholds via `budgetColor()` pattern in health.go ✓
- Enhancement-only scope correctly identified ✓

### UX-006 — Status line: enhance existing cost display
- `costStyle()` confirmed at statusline.go:476 ✓
- Already returns lipgloss.Style based on cost thresholds ✓
- Enhancement-only scope correctly identified ✓

### UX-009 — Tree: full-row color by agent status
- `AgentTreeModel` is a struct (not interface) ✓
- Simple style wrapping per row ✓
- Depends on UX-008 (tree-overhaul branch) — correct ✓

### UX-010 — Tree: inline cost per agent
- Agent cost data available in state ✓
- `formatValue()` is pure logic — straightforward ✓
- Same branch as UX-008/009 ✓

### UX-011 — Status line: team indicator
- Teams health data available via health.go ✓
- `budgetColor()` pattern at health.go:411 provides reusable color logic ✓
- Poll tick handler in app.go exists for populating ✓

### UX-012 — First-run orientation hints
- `hintbar/hintbar.go` exists with test file ✓
- Referenced across 9 files — well-integrated component ✓
- Zero new infrastructure — uses existing component ✓

### UX-013 — Detail: collapsible overview to one-liner
- detail.go (625L) has existing collapsible sections pattern ✓
- `renderOverviewCompact()` is a new function but follows existing patterns ✓

### UX-014 — Conversation: inline streaming tool indicator
- `ToolUseMsg` handler at panel.go:211 ✓
- `ToolBlock` already has rendering infrastructure ✓
- `spinnerTickMsg` pattern exists in statusline.go for animation reference ✓

### UX-019 — Teams: auto-switch on completion
- `TabFlashMsg` exists across 9 files — mature pattern ✓
- app.go (691L) has poll tick handler for detection ✓
- ui_event_handlers.go (971L) for switch logic ✓

### UX-020 — Reduce-motion config flag
- Config infrastructure exists ✓
- `spinnerTickMsg` is the main animation to gate ✓
- Simple boolean check before animation ✓

### UX-021 — Layout: focus-driven drawer/content split
- `computeDrawerLayout()` at layout.go:232 is the right function ✓
- Focus state available in AppModel ✓

### UX-023 — Tree: pulse animation
- Depends on UX-020 (reduce-motion) — correct ✓
- `spinnerTickMsg` provides existing tick pattern ✓

### UX-024 — Conversation: timestamp gutter
- panel.go (1264L) has message rendering loop ✓
- Existing tick infrastructure for updates ✓

### UX-025 — Status line: cost flash (opt-in)
- `costStyle()` at statusline.go:476 is the render point ✓
- `TabFlashMsg` provides flash pattern reference ✓
- Depends on UX-020 — correct ✓

### UX-026 — Teams: timeline progress bars
- health.go (420L) has existing budget bar with `budgetColor()` ✓
- Member data structure available ✓

### UX-027 — Teams: tabs in drawer
- health.go + detail.go in teams/ ✓
- key_handlers.go for navigation ✓
- Edge cases (empty, overflow, dismiss) captured in AC ✓

### UX-028 — Teams: diff summary on completion
- detail.go (325L) is the render target ✓
- stdout JSON parsing is existing pattern ✓

---

## Cross-Cutting Observations

### 1. File reference consistency
The spec (Section 12) and several tickets reference `activity.go` which doesn't exist. All references should be updated to `cli/agent_sync.go`.

### 2. Keybinding namespace is getting crowded
Three tickets propose new keybindings (UX-007, UX-022, and the spec's density toggle). A keybinding audit of config/keys.go should be done before P0 starts to reserve slots.

### 3. Tool block collapse is partially implemented
UX-015 overlaps with existing infrastructure. If downscoped, this frees ~0.5 sessions of P2 budget.

### 4. UDS protocol assumption in UX-018
Toast emission from gogent-team-run may require protocol work not accounted for in the ticket.

### 5. statusLineHeight conversion (UX-017)
Changing a constant to a dynamic value in a tightly-coupled layout calculation is the single riskiest non-P0 change. The invariant test in the AC is essential.

---

## Recommended Actions Before Staff-Architect Pass

1. **Fix RED-1:** Update UX-004 file reference from `activity.go` to `cli/agent_sync.go`
2. **Fix RED-2:** Run keybinding audit, choose alternative for UX-007
3. **Fix YELLOW-4:** Remove incorrect UX-024 dependency from UX-016
4. **Verify YELLOW-3:** Confirm current tool block collapse behavior and downscope UX-015 if appropriate
5. **Verify YELLOW-6:** Check UDS protocol for toast support from gogent-team-run

---

## For Staff-Architect Review

**Questions for the final pass:**

1. Is the `Render(mode RenderMode)` unified method the right architectural choice for UX-003, or would a simpler approach (e.g., width-aware conditional inside existing `View()`) be preferable?
2. Should icon rail mode (UX-003) apply to all `rightPanelMode` values or only the agent panel? The effort delta is significant.
3. Is the `statusLineHeight` constant-to-method conversion (UX-017) safe given the 8-term height calculation at layout.go:160?
4. Should UX-008+009+010 (tree-overhaul) be pulled into the P0 branch with UX-003 to avoid the merge conflict risk entirely?
5. Is the UDS protocol between gogent-team-run and TUI toast-aware, or does UX-018 need protocol extension work?

---

_Generated: 2026-04-11_
_Source: tickets/UX-redesign/UX-001.md through UX-028.md_
_Cross-referenced against: codebase grep of all referenced files_
