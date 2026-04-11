# Session Report: UX Overhaul — Braintrust Review + P0/P1/P2 Implementation

## Summary

Comprehensive multi-session effort covering UX redesign evaluation, ticket generation, and full P0+P1+P2 implementation. The UX-REDESIGN-SPEC.md (1233 lines, 24 recommendations) was evaluated via Braintrust (Einstein + Staff-Architect + Beethoven), producing a multi-perspective analysis that identified a fundamental PMF gap. 28 tickets were generated, preflight-reviewed against the codebase, and passed through Staff-Architect final review. All 20 P0+P1+P2 tickets have been implemented and shipped.

**Branch:** `ux-overhaul`
**Session Cost:** ~$25 total (Braintrust $6.14 + P0 agents ~$4 + P1 agents ~$4.90 + P2 ~$5 + router overhead)
**Commits:** 19 (6 bug fixes + 1 schema/config + 12 UX implementation + ticket management)

---

## Phase 1: Braintrust Review ($6.14)

### Problem Statement

Evaluate UX-REDESIGN-SPEC.md for:
1. **PMF** across experience profiles (vibe coders, intermediate devs, power users)
2. **Architectural sanity** of 24 proposed Bubbletea/lipgloss implementations

### Agents Invoked

| Agent | Model | Cost | Duration | Output |
|-------|-------|------|----------|--------|
| Mozart (orchestrator) | Opus | $1.75 | 7m26s | Problem brief + config files |
| Einstein (PMF) | Opus | $1.40 | ~5m | Theoretical analysis |
| Staff-Architect (arch) | Opus | $2.02 | ~6m | Architectural review |
| Beethoven (synthesis) | Opus | $0.97 | ~5m | Unified recommendations |

### Key Findings

1. **Category error:** Spec conflates terminal width with user expertise — narrow terminal does not equal beginner
2. **All 24 recs increase info density** — serving power users, potentially overwhelming vibe coders
3. **Recs 4a/4b already implemented:** `renderContextBar()` at statusline.go:228, `costStyle()` at statusline.go:476
4. **interfaces.go modification unnecessary:** `AgentTreeModel`/`AgentDetailModel` are structs, not interfaces
5. **3 additions recommended:** Simple/expert toggle (N1), first-run hints (N2), reduce-motion config (N3)

### Output Files

- `tickets/UX-redesign/BRAINTRUST-ANALYSIS-20260411.md`
- `.claude/sessions/*/teams/20260411-084208.braintrust/` (raw agent outputs)

---

## Phase 2: Ticket Generation (28 tickets)

### Generated

| File | Purpose |
|------|---------|
| `tickets/UX-redesign/overview.md` | Implementation plan with 4 phases |
| `tickets/UX-redesign/tickets-index.json` | Machine-readable registry |
| `tickets/UX-redesign/UX-001.md` — `UX-028.md` | Individual tickets with frontmatter |
| `tickets/UX-redesign/PREFLIGHT-REVIEW.md` | Red/yellow/green codebase sanity check |
| `tickets/UX-redesign/STAFF-ARCHITECT-FINAL-REVIEW.md` | 7-layer final review |

### Revised Priority Matrix

| Phase | Tickets | Effort | Status |
|-------|---------|--------|--------|
| P0 | UX-001 through UX-007 (7 tickets) | ~3-3.5 sessions | **COMPLETE** (7/7) |
| P1 | UX-008 through UX-012 (5 tickets) | ~2-3 sessions | **COMPLETE** (5/5) |
| P2 | UX-013 through UX-020 (8 tickets) | ~3-4 sessions | **COMPLETE** (8/8) |
| P3 | UX-021 through UX-028 (8 tickets) | ~3-4 sessions | Pending |

### Staff-Architect Final Review

**Verdict:** APPROVE_WITH_CONDITIONS (HIGH confidence)
**6 conditions identified, all applied:**

| # | Severity | Condition | Resolution |
|---|----------|-----------|------------|
| 1 | CRITICAL | UX-004 wrong file ref | `activity.go` → `agent_sync.go:231` |
| 2 | CRITICAL | UX-007 keybinding conflict | `alt+p` → `alt+\` |
| 3 | MAJOR | UX-015 is enhancement not greenfield | Rewritten, effort 0.75→0.25-0.5s |
| 4 | MAJOR | UX-016 phase ordering violation | Removed UX-024 dependency |
| 5 | DECISION | UX-003 arch decisions | `Render(mode)` + agents-only scope documented |
| 6 | MINOR | 4 small ticket fixes | UX-008, UX-012, UX-018, UX-022 updated |

---

## Phase 3: Bug Fix Commits (pre-existing)

Six commits for bugs discovered during recent sessions:

| Commit | What Fixed |
|--------|-----------|
| `ff5e40dc` | **JSONL scanner overflow:** Default 64KB `bufio.Scanner` silently dropped data from long sessions. Added 10MB-buffered scanners across 5 packages (memory, session, telemetry, routing, workflow). |
| `71237fa7` | **Scout stdin handling:** `filepath.Dir(files[0])` broke with multi-directory file lists. Added `commonRoot()` + expanded stdin buffer + scanner error checking. |
| `ef8bbde5` | **Filesystem robustness:** `os.MkdirAll` succeeded on dirs owned by other users. Added writable-probe check, 107-byte Unix socket path limit, UID-scoped /tmp fallback. |
| `d8b9a149` | **MCP spawner overflow:** Unbounded stdout buffering from agents. Added `cliOutputCollector` with cap + incremental result parsing. |
| `cba095bc` | **Agent sharp-edges:** Updated 5 agent configs + extended team schemas from recent sessions. |
| `63fb0cf4` | **Timeout alignment:** All docs/schemas/configs referenced 300000ms (5 min) but runtime was 600000ms (10 min). Aligned 10 files to 600000ms. |

---

## Phase 4: P0 Ticket Implementation

| Ticket | Title | Agent | Cost | Duration | Changes |
|--------|-------|-------|------|----------|---------|
| UX-001 | Conversation: horizontal rules | go-tui (Sonnet) | $1.12 | 2m31s | panel.go +12L, panel_test.go +78L |
| UX-002 | Conversation: user/assistant colors | go-tui (Sonnet) | $1.12 | 5m (timeout, code complete) | panel.go +22L, panel_rendering_test.go fix |
| UX-005 | Status line: context bar enhancement | go-tui (Sonnet) | $1.00 | 2m55s | statusline.go ▓→█ + reposition to Row 1 |
| UX-006 | Status line: cost display enhancement | go-tui (Sonnet) | $0.84 | 4m12s | statusline.go cost→Row 1 first + bold thresholds |

### What Changed in the TUI

**Conversation panel (`panel.go`):**
- Unicode ─ horizontal rule between role transitions (You↔Claude)
- User content: cyan (`config.ColorPrimary`)
- Assistant content: green (`config.ColorSuccess`)
- System content: muted/grey (`config.StyleMuted`)

**Status line (`statusline.go`):**
- Context bar: moved from Row 2 to Row 1, filled char `▓` → `█`
- Cost badge: moved to first position in Row 1, bold styling, thresholds green<$1/yellow$1-5/red>$5
- Row 2: simplified (elapsed + streaming indicator only)

---

## Phase 4b: Remaining P0 Tickets (completed session 2)

| Ticket | Title | Agent | Cost | Duration | Changes |
|--------|-------|-------|------|----------|---------|
| UX-003 | Icon rail mode (< 30 cols) | go-tui (Sonnet) | — | — | tree.go `Render(mode)` + `renderIconRail()` |
| UX-004 | Relative paths in activity | go-tui (Sonnet) | — | — | Strip project root from activity paths |
| UX-007 | Simple/expert toggle | go-tui (Sonnet) | — | — | `alt+\` toggle, hide/show right panel |

---

## Phase 5: P1 Ticket Implementation (completed session 2)

| Ticket | Title | Agent | Cost | Duration | Changes |
|--------|-------|-------|------|----------|---------|
| UX-008 | Tree: two-column dot-leader layout | go-tui (Sonnet) | $1.42 | 6m53s | Rewrote `renderNode()` — indent + dot leaders + right-aligned values. Removed box-drawing chars. ANSI-safe via `lipgloss.Width()`. 5 new tests at widths {22,30,45,80}. |
| UX-009 | Tree: full-row color by agent status | go-tui (Sonnet) | $1.47 | 8m15s | Added `StatusRowStyle()` — running=dim green, complete=bold green, error=red, killed=yellow+strikethrough, pending=grey. Entire row colored, not just icon. |
| UX-010 | Tree: inline cost per agent | direct (trivial) | $0.00 | <1m | Already implemented by UX-008's `buildTreeValue()`. Added 3 dedicated cost-display tests. |
| UX-011 | Status line: team indicator | go-tui (Sonnet) | $1.04 | 2m42s | `renderTeamIndicator()` renders `⚡name ●●○○ 2/4 $0.38` in Row 1. Colored member dots. `TeamIndicatorData` struct + `TeamIndicator()` interface method. Wired to poll tick in app.go. 7 new tests. |
| UX-012 | First-run orientation hints | go-tui (Sonnet) | $0.97 | 4m52s | Onboarding layer on `HintBarModel` — hints during sessions 1-3, per-hint dismissal, persistence to `$XDG_DATA_HOME/gogent/onboarding.json`. New `onboarding.go`. 25 new tests. |

**P1 total agent cost:** $4.90
**P1 commit:** `44f5696b` — 16 files, +1097/-116 lines

### What Changed in the TUI (P1)

**Agent tree (`tree.go`):**
- Two-column dot-leader layout: `agent-name ......... status $0.12`
- Indentation-based hierarchy (2 spaces/depth) replaces box-drawing characters
- Full-row coloring by agent status (dim green running, bold green complete, red error, yellow+strikethrough killed)
- Right-aligned value column: cost (`$X.XX`) or status word + optional AC progress

**Status line (`statusline.go`):**
- Compact team indicator in Row 1: `⚡name ●●○○ 2/4 $0.38`
- Member dots colored by status (green/grey/red/bold-green/yellow)
- Auto-hides when no team is active

**Hint bar (`hintbar.go` + `onboarding.go`):**
- Orientation hints for sessions 1-3: Tab/arrows/Enter
- Cyan-styled keys to distinguish from normal muted hints
- Per-hint dismissal when user performs the action
- Persisted to `$XDG_DATA_HOME/gogent/onboarding.json`

**Interfaces (`interfaces.go`):**
- `TeamIndicatorData` struct for status line team data
- `TeamIndicator()` method added to `teamsHealthWidget` interface

**Teams health (`health.go`):**
- `TeamIndicator()` implementation pulling from `MostRecentRunning()`

**App wiring (`app.go`):**
- Poll tick handler populates status line team fields from `teamsHealth.TeamIndicator()`

---

## Phase 6: P2 Ticket Implementation (completed session 3)

| Ticket | Title | Changes |
|--------|-------|---------|
| UX-013 | Detail: collapsible overview to one-liner | Overview section defaults collapsed; added `renderCollapsed` callback to `DetailSection`; compact one-line format: `Status · AgentType · Model · $Cost · Duration` |
| UX-014 | Conversation: inline streaming tool indicator | Tool blocks now render inline in ALL layout tiers (not just Compact); `⏳` pending indicator for in-flight tools; `StartedAt`/`Duration` tracking on `ToolBlock`; elapsed time on completed tools (ms/s/m format) |
| UX-015 | Conversation: collapsible tool-use blocks | `toggleLastToolExpansion` (single block) and `cycleAllToolExpansion` (all blocks) keybindings; collapsed shows name + input summary + duration; expanded adds full I/O |
| UX-016 | Status line: agent count sparkline dots | Replaced `agents:N` with per-status sparkline: running=green●, complete=bright-green●, pending=grey○, error=red●, killed=yellow●; `AgentStats` struct replaces plain `int` |
| UX-017 | Status line: adaptive 1-row at narrow widths | Width ≥120: two-row layout (unchanged); width <120: single-row compact (cost, model, sparkline, context bar, elapsed, streaming); `Height()` method; `viewFull()`/`viewCompact()` split |
| UX-018 | Teams: action-hinted toasts | Toast messages include actionable hints (`/team-status`, `/team-result`); budget warning toast (fire-once via `atomic.Bool`); member failure toasts with retry count; new UDS types: `toast`, `team_update` |
| UX-019 | Teams: auto-switch on completion | On team complete/error: flash Teams tab + auto-switch to Teams panel; guarded when streaming or user has input text; `HasInput()` on `ClaudePanelModel` and `claudePanelWidget` interface |
| UX-020 | Reduce-motion config flag | "Reduce Motion" toggle in Settings → Display; disables spinner animation (static `⠿` instead); disables rainbow ultrathink gradient; suppresses tab flash animation; wired through `sharedState` → `statusLine` + `claudePanel`; WCAG 2.3.1 compliance |

**P2 commit:** `9b899a23` — 29 files, +1315/-94 lines

### What Changed in the TUI (P2)

**Agent detail (`detail.go`):**
- Overview section starts collapsed with one-line summary
- New `renderCollapsed` callback on `DetailSection` for per-section compact rendering
- `renderOverviewCompact()`: `Status · AgentType · Model · $Cost · Duration`

**Conversation panel (`panel.go`):**
- Inline tool blocks in all layout tiers (previously Compact-only)
- Pending tool indicator (`⏳`) while tool is in-flight
- Duration display on completed tools (`fmtToolDuration()`)
- Collapsible tool blocks: per-block toggle + cycle-all keybindings
- Reduce-motion: rainbow ultrathink → plain "thinking..." when enabled

**Status line (`statusline.go`):**
- `renderAgentSparkline()`: per-status colored dots replacing plain count
- Adaptive layout: `viewFull()` (2 rows, ≥120 cols) / `viewCompact()` (1 row, <120 cols)
- `Height()` method replaces hardcoded `statusLineHeight` constant in `layout.go`
- Reduce-motion: static `⠿ streaming` replaces animated spinner

**Team daemon (`cmd/gogent-team-run/`):**
- Action-hinted toast notifications via UDS (`toastPayload`, `teamUpdatePayload`)
- Budget warning toast (fires once when remaining drops below `WarningThresholdUSD`)
- Member failure toasts with retry count
- Team completion toast with total cost summary
- Team update notification for tab flash / auto-switch (UX-019)

**MCP tools (`mcp/tools.go`):**
- Toast messages rewritten with actionable hints (`/team-status to inspect`, `/team-result to view findings`)

**Settings (`settingstree.go`):**
- New "Reduce Motion" toggle in Display section

**App model (`app.go`, `ui_event_handlers.go`):**
- `sharedState.reduceMotion` flag wired to statusline + claude panel
- Team completion handler: tab flash + auto-switch (guarded by streaming/input state)
- Tab flash suppressed when reduce-motion enabled

**Interfaces (`interfaces.go`):**
- `HasInput() bool` added to `claudePanelWidget`
- `SetReduceMotion(v bool)` added to `claudePanelWidget`

**State (`state/provider.go`):**
- `ToolBlock.StartedAt` and `ToolBlock.Duration` fields for tool timing

### Test Coverage (P2)

- 615 new lines of test code across 4 modified + 2 new test files
- `cmd/gogent-team-run/toast_test.go` — **new**: toast payload, action hints, budget warning
- `internal/tui/model/reduce_motion_test.go` — **new**: setting propagation, tab flash suppression
- `statusline_test.go` — sparkline rendering, compact/full layout, height assertions
- `panel_test.go` — tool duration formatting, tool expansion toggle/cycle, reduce-motion indicator
- `detail_test.go` — collapsed overview rendering, section visibility
- `team_drawer_test.go` — auto-switch on completion, streaming/input guards

---

## Configuration Changes

### Timeout Default Alignment

All agent timeout references aligned to 600000ms (10 min) across:
- `.claude/CLAUDE.md` (spawn_agent docs)
- `.claude/schemas/teams/implementation.json`
- `.claude/schemas/teams/common-types.md`
- `.claude/skills/review/SKILL.md`
- `.claude/agents/review-orchestrator/review-orchestrator.md`
- `.claude/braintrust/mcp-spawning-architecture-v2.md`
- `docs/TEAM-RUN-FRAMEWORK.md`
- `docs/teams/SKILL-AUTHORING-GUIDE.md`
- `cmd/gogent-team-run/testdata/review_config.json`

### Braintrust Skill Budget

- Fixed Q4 prompt text: $25 → $50 default (matched validation limits)
- Location: `~/.claude-em/skills/braintrust/SKILL.md`

---

## Architectural Decisions Made

| Decision | Resolution | Source |
|----------|-----------|--------|
| Rendering approach for icon rail | `Render(mode RenderMode)` unified method | Staff-Architect Q1 |
| Icon rail scope | `RPMAgents` mode only (not all 6 panel modes) | Staff-Architect Q2 |
| `statusLineHeight` conversion | Safe — `toastH` dynamic height precedent exists | Staff-Architect Q3 |
| Tree-overhaul branch strategy | Single branch, keep phase boundaries | Staff-Architect Q4 |
| UDS toast protocol | Already functional (`bridge/server.go:327`) | Staff-Architect Q5 |
| Simple toggle keybinding | `alt+\` (backslash = vertical split mnemonic) | Staff-Architect RED-2 |

---

## Files Modified (all sessions)

### Go Source — P1 (committed `44f5696b`)
- `internal/tui/components/agents/tree.go` — dot-leader layout, `StatusRowStyle()`, `buildTreeValue()`
- `internal/tui/components/agents/tree_test.go` — 15+ new tests (dot-leader, ANSI width, status color, cost)
- `internal/tui/components/agents/pipeline_test.go` — updated assertions for dot-leader output
- `internal/tui/components/statusline/statusline.go` — team indicator fields + `renderTeamIndicator()`
- `internal/tui/components/statusline/statusline_test.go` — 7 new team indicator tests
- `internal/tui/components/statusline/export_test.go` — team indicator test helpers
- `internal/tui/components/hintbar/hintbar.go` — onboarding layer, `SetOnboarding()`, `DismissHint()`
- `internal/tui/components/hintbar/hintbar_test.go` — 19 new onboarding tests
- `internal/tui/components/hintbar/onboarding.go` — **new**: persistence for onboarding state
- `internal/tui/components/hintbar/onboarding_test.go` — **new**: 6 persistence tests
- `internal/tui/components/teams/health.go` — `TeamIndicator()` implementation
- `internal/tui/model/interfaces.go` — `TeamIndicatorData` struct, `TeamIndicator()` interface method
- `internal/tui/model/app.go` — poll tick wiring for team indicator
- `internal/tui/model/ui_event_handlers.go` — (carried from P0)
- `internal/tui/model/team_drawer_test.go` — mock updated for new interface method

### Go Source — P0 (committed earlier)
- `internal/tui/components/claude/panel.go` — turn separators + role colors
- `internal/tui/components/claude/panel_test.go` — 4 separator tests
- `internal/tui/components/claude/panel_rendering_test.go` — import fix
- `internal/tui/components/statusline/statusline.go` — context bar + cost repositioning
- `internal/tui/components/statusline/statusline_test.go` — threshold tests
- `pkg/memory/scanner.go` — new (10MB JSONL scanner)
- `pkg/session/scanner.go` — new
- `pkg/telemetry/scanner.go` — new
- `pkg/workflow/scanner.go` — new
- `pkg/config/paths.go` — writable dir probe
- `internal/tui/bridge/server.go` — socket path limit
- `internal/tui/mcp/spawner.go` — output collector
- `cmd/gogent-scout/main.go` — common root + stdin buffer
- `cmd/gogent-scout/native_scout.go` — refactored file collection
- + 20 more (tests, session, telemetry, routing packages)

### Go Source — P2 (committed `9b899a23`)
- `internal/tui/components/agents/detail.go` — `renderCollapsed` callback, `renderOverviewCompact()`
- `internal/tui/components/agents/detail_test.go` — collapsed overview tests
- `internal/tui/components/agents/pipeline_test.go` — updated for AgentStats
- `internal/tui/components/claude/panel.go` — inline tools all tiers, duration, expansion, reduce-motion
- `internal/tui/components/claude/panel_test.go` — duration format, expansion toggle, reduce-motion
- `internal/tui/components/settingstree/settingstree.go` — reduce_motion toggle
- `internal/tui/components/statusline/statusline.go` — sparkline, adaptive layout, reduce-motion
- `internal/tui/components/statusline/statusline_test.go` — sparkline + compact layout tests
- `internal/tui/components/statusline/export_test.go` — AgentStats test helpers
- `internal/tui/mcp/tools.go` — action-hinted toast messages
- `internal/tui/model/app.go` — `sharedState.reduceMotion`
- `internal/tui/model/app_test.go` — updated for new interface methods
- `internal/tui/model/bench_test.go` — updated for AgentStats
- `internal/tui/model/event_pipeline_test.go` — updated for AgentStats
- `internal/tui/model/interfaces.go` — `HasInput()`, `SetReduceMotion()`
- `internal/tui/model/layout.go` — dynamic `statusLine.Height()` replaces constant
- `internal/tui/model/layout_test.go` — updated for dynamic height
- `internal/tui/model/reduce_motion_test.go` — **new**: reduce-motion propagation tests
- `internal/tui/model/team_drawer_test.go` — auto-switch + flash guard tests
- `internal/tui/model/ui_event_handlers.go` — team completion handler, reduce-motion wiring
- `internal/tui/phase10_integration_test.go` — updated for interface changes
- `internal/tui/state/provider.go` — `ToolBlock.StartedAt`, `ToolBlock.Duration`
- `cmd/gogent-team-run/daemon.go` — `budgetWarnSent atomic.Bool`
- `cmd/gogent-team-run/main.go` — completion/failure toasts, team update notifications
- `cmd/gogent-team-run/spawn.go` — member failure toasts
- `cmd/gogent-team-run/uds.go` — `toastPayload`, `teamUpdatePayload` types
- `cmd/gogent-team-run/wave.go` — budget warning toast logic
- `cmd/gogent-team-run/toast_test.go` — **new**: toast payload + action hint tests
- `tickets/UX-redesign/tickets-index.json` — UX-013..020 marked completed

### Tickets/Docs (committed)
- `tickets/UX-redesign/` — 33 new files (28 tickets + analysis + overview + preflight + final review + index)
- `tickets/UX-redesign/UX-REDESIGN-SPEC.md` — Sections 8-12 added/revised

### Config (.claude/) (committed)
- `.claude/CLAUDE.md` — timeout references
- `.claude/schemas/teams/` — timeout + schema extensions
- `.claude/skills/review/SKILL.md` — timeout
- `.claude/agents/review-orchestrator/` — timeout
- `.claude/agents/*/sharp-edges.yaml` — 5 agents updated

---

## Git Log (all sessions)

```
9b899a23 feat(tui): implement P2 UX redesign — detail collapse, inline tools, sparklines, adaptive status, toasts, auto-switch, reduce-motion (UX-013..020)
fe40f72b docs: update session report with P1 completion — P0+P1 COMPLETE (12/12)
44f5696b feat(tui): implement P1 UX redesign — tree overhaul, status hints, onboarding (UX-008..012)
d361ab87 chore(tickets): mark UX-004 completed — P0 COMPLETE (7/7)
93980b4d feat(tui): strip project root from activity paths, split compound commands (UX-004)
43f6ee39 chore(tickets): mark UX-003 completed
9f65e87c feat(tui): add icon rail mode for agent panel at narrow widths (UX-003)
3241b84d chore(tickets): mark UX-007 completed
fe6d8ecb feat(tui): add simple/expert toggle to hide right panel (UX-007)
3d743891 docs: add session report for 2026-04-11 UX overhaul
a44a6aa8 chore(tickets): mark UX-006 completed
ee2ca55f feat(tui): make cost first element in Row 1 with bold threshold colors (UX-006)
e2fe2729 chore(tickets): mark UX-005 completed
13157ece feat(tui): reposition context bar to Row 1 and use full block char (UX-005)
63fb0cf4 fix(config): align all timeout references to 600000ms (10 min) default
5e6be636 chore(tickets): mark UX-002 completed
119bd79c feat(tui): differentiate user/assistant message colors in conversation (UX-002)
c279981a chore(tickets): mark UX-001 completed
4d1ee237 feat(tui): add horizontal rule between conversation turns (UX-001)
9c46fac4 fix(tickets): apply all 6 staff-architect conditions before implementation
bc5b2f01 docs(ux): add staff-architect final review — APPROVE_WITH_CONDITIONS
290f1a69 docs(ux): add braintrust analysis, 28 implementation tickets, and preflight review
cba095bc chore(agents): update sharp-edges from recent sessions, extend team schemas
d8b9a149 fix(tui): harden MCP spawner output collection and CLI event parsing
ef8bbde5 fix(paths): harden directory creation, socket paths, and permission cache
71237fa7 fix(scout): handle piped stdin file lists and compute common root correctly
ff5e40dc fix(jsonl): replace default bufio.Scanner with 10MB-buffered scanners across all JSONL readers
```

---

## Remaining Work (P3)

8 tickets remain in Phase 4:

| Phase | Tickets | Key Items |
|-------|---------|-----------|
| P3 (8) | UX-021..028 | Focus-driven layout, tree density toggle, pulse animation, timestamp gutter, cost flash, timeline progress bars, team tabs, diff summary |

All P2 dependencies are now satisfied. P3 has two dependencies on P2 work (now done): UX-023→UX-020 (reduce-motion), UX-025→UX-020 (reduce-motion). UX-022→UX-008 was already done in P1.
