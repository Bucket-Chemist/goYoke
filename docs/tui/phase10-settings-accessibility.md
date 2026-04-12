# Phase 10c: Settings & Accessibility

> **Tickets:** TUI-048, TUI-049, TUI-050, TUI-051
> **Status:** All complete
> **Packages:** `statusline/`, `settingstree/`, `config/`, `model/`
> **Purpose:** Make the TUI accessible and configurable. Transform settings from display-only to interactive. Ensure WCAG AA compliance.

---

## TUI-048: Status Line Semantic Colors

### Purpose

Replace static status line text with semantically color-coded values. Users can glance at the status line and instantly understand system health through color cues.

### Design Decision

**Threshold-based coloring** for numeric values. Each field has green/yellow/red thresholds calibrated to meaningful ranges.

| Field | Green | Yellow | Red |
|-------|-------|--------|-----|
| Cost | <$0.10 | <$1.00 | >=$1.00 |
| Context | <50% | <80% | >=80% |
| Permission | Default mode | Plan mode | Allow-all |
| Auth | Authenticated | — | Not authenticated |
| Streaming | Always InfoStyle | — | — |

### Implementation

- Semantic color methods from TUI-044 applied to each status field
- New fields: `UncommittedCount` (from git status), `AgentCount`
- `SetTheme()` method for dynamic theme switching
- Context-aware display adapts to terminal width

### Usage

Status line renders automatically at the bottom of every view. Colors update in real-time as values change (e.g., cost accumulates, context fills).

### Testing

- 14 new tests (56 total statusline tests)
- 88.1% coverage
- Threshold boundary tests for each colored field

---

## TUI-049: Token Usage Progress Bar

### Purpose

Visualize context window usage as a graphical progress bar alongside the percentage. Provides an at-a-glance indicator of remaining context capacity.

### Design Decision

**10-character visual bar** using equals signs and spaces with the same green/yellow/red threshold scheme as status line semantic colors.

### Implementation

- `renderContextBar()`: Renders `[=====     ] 52%` format
- Same thresholds as TUI-048 (green <50%, yellow <80%, red >=80%)
- **Narrow fallback:** Width <80 columns renders text-only `ctx:52%`
- Values clamped to [0, 100] range

### Usage

```
Standard view:  [========  ] 80%
Narrow view:    ctx:80%
```

The progress bar appears in the status line when context usage data is available from CLI events.

### Testing

- 10 new tests (70 total statusline tests)
- 88.7% coverage
- Edge cases: 0%, 100%, negative values, narrow terminal

---

## TUI-050: Interactive Settings Panel

### Purpose

Transform the settings view from display-only (Phase 1-9) to an interactive tree with keyboard navigation. Users can toggle settings, cycle options, and see changes take effect immediately.

### Design Decision

**New `settingstree/` package** instead of extending the existing `settings/` (156 lines, display-only). Clean separation avoids refactor risk on already-tested code.

| Alternative | Why Not Chosen |
|-------------|----------------|
| Extend existing settings/ | High refactor risk on tested code |
| Use bubbles/list | Not tree-shaped, wrong UX paradigm |
| Custom tree (like agent tree) | Chosen — consistent with agents/tree.go pattern |

### Implementation

- **3 sections:** Display, Session, Status
- **SettingType enum:** Toggle, Select, Display (read-only)
- **Navigation:** Up/down moves cursor, Enter toggles or cycles
- **SettingChangedMsg** emitted on every change
- **settingsTreeWidget interface** in `model/interfaces.go`

### Usage

Navigate to Settings tab, use arrow keys to move between settings:

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Up / k | Move cursor up |
| Down / j | Move cursor down |
| Enter | Toggle boolean / cycle select / collapse section |
| Esc | Return to previous panel |

### Testing

- 24 tests covering navigation, toggles, selects, section collapse
- 87.4% coverage
- Verified: SettingChangedMsg emission on every state change

---

## TUI-051: High-Contrast Mode

### Purpose

Ensure the TUI is usable by visually impaired users. Provide a high-contrast theme variant that meets WCAG AA contrast ratio requirements (4.5:1 minimum for normal text).

### Design Decision

**ContrastRatio() function** implementing WCAG 2.1 sRGB linearization with relative luminance calculation. Returns values in [1.0, 21.0] range. ValidateContrast() checks all 5 semantic colors against the background.

### Implementation

- `ContrastRatio(fg, bg lipgloss.Color)`: WCAG 2.1 sRGB linearization + relative luminance
- `ValidateContrast()`: Checks 5 semantic colors against #000000 at 4.5:1 threshold
- HighContrast theme passes all validation checks
- **SettingChangedMsg wiring:**
  - `"theme"` key → cycles ThemeVariant (triggers ThemeChangedMsg)
  - `"ascii_icons"` key → toggles UseASCII on active theme

### Usage

Activate via Settings panel → Display section → Theme → cycle to "High Contrast"

```go
// Programmatic validation
ratio := config.ContrastRatio(errorColor, backgroundColor)
// Returns: 8.59 (passes 4.5:1 WCAG AA threshold)

ok := config.ValidateContrast(theme)
// Returns: true for HighContrast theme
```

### Testing

- 13 new config tests (96.3% config coverage)
- 8 new model tests (89.3% model coverage)
- Verified: HighContrast variant passes ValidateContrast
- Verified: SettingChangedMsg → ThemeChangedMsg propagation chain

---

---

## UX-016: Agent Count Sparkline Dots

> **Added in:** UX Redesign P2 sprint (commit `9b899a23`)

### Purpose

Replace the plain `agents:N` text in the status line with a per-status sparkline that conveys agent health at a glance. Each dot represents one agent, colored by its current status.

### Design Decision

**Per-status dot coloring** matching the agent tree's `StatusRowStyle()` from [[phase10-overview|UX-009]]. The dot order is: running → pending → complete → error → killed, reading left to right from "needs attention" to "done".

| Status | Symbol | Style |
|--------|--------|-------|
| Running | ● | `SuccessStyle` (green) |
| Pending | ○ | `StyleMuted` (grey) |
| Complete | ● | `SuccessStyle` + Bold (bright green) |
| Error | ● | `ErrorStyle` (red) |
| Killed | ● | `WarningStyle` (yellow) |

### Implementation

- `renderAgentSparkline()` in `statusline.go`
- `AgentStats` struct replaces `AgentCount int` — carries per-status counts
- Prefix: `agents: {running}/{total}` followed by colored dots
- Empty string when `Total == 0` (no wasted space)
- `ui_event_handlers.go` wires `.AgentStats` from registry on every agent event

### Testing

- Sparkline rendering tests with various status combinations
- Zero-agent hiding test
- Integration with `AgentStats` struct propagation

---

## UX-017: Adaptive Status Line (1-Row at Narrow Widths)

> **Added in:** UX Redesign P2 sprint (commit `9b899a23`)

### Purpose

The two-row status line wastes vertical space at narrow terminal widths where every row counts. Provide a single-row compact variant that shows only critical fields.

### Design Decision

**Width-based dispatch** at 120 columns. Wide terminals get the full two-row layout; standard/compact terminals get a single row with only cost, model, sparkline, context bar, elapsed time, and streaming indicator. Permission mode, auth, git branch, CWD, and vim/mouse badges are omitted in compact mode.

| Width | Rows | Fields Shown |
|-------|------|-------------|
| ≥120 | 2 | All fields (unchanged from TUI-048/049) |
| <120 | 1 | Cost, plan badge, model, agent sparkline, context bar, elapsed, streaming |

### Implementation

- `View()` dispatches to `viewFull()` or `viewCompact()` based on `m.width`
- `Height() int` method returns 2 or 1 — used by `layout.go` instead of hardcoded `statusLineHeight` constant
- `viewCompact()` uses `joinLeftRight()` for single-row alignment
- Both views share `renderAgentSparkline()` and `renderContextBar()`

### Testing

- Height assertion tests at various widths
- Compact layout field presence/absence tests
- Layout integration: `computeLayout()` uses dynamic `m.statusLine.Height()`

---

## UX-020: Reduce Motion Config Flag

> **Added in:** UX Redesign P2 sprint (commit `9b899a23`)

### Purpose

Provide a global reduce-motion toggle for users with vestibular disorders or motion sensitivity. Meets WCAG 2.3.1 ("Three Flashes or Below Threshold") by disabling all animations.

### Design Decision

**Single boolean flag** propagated through `sharedState` to all animation sites. Each component checks `reduceMotion` and substitutes static equivalents.

| Animation | Normal | Reduce-Motion |
|-----------|--------|---------------|
| Streaming spinner | Braille animation cycle (`⠋⠙⠹...`) | Static `⠿ streaming` |
| Ultrathink indicator | `RainbowGradient("Thinking...")` | Plain `thinking...` (muted) |
| Tab flash | Animated flash highlight | Suppressed (no-op) |
| Spinner tick scheduling | Continuous `tea.Tick` | No tick scheduled (saves CPU) |

### Implementation

- `settingstree.go`: New "Reduce Motion" toggle in Display section (default: off)
- `app.go`: `sharedState.reduceMotion` flag
- `ui_event_handlers.go`: `handleSettingChanged` wires `"reduce_motion"` → `sharedState` + `statusLine.ReduceMotion` + `claudePanel.SetReduceMotion()`
- `statusline.go`: `spinnerTickMsg` handler returns `nil` cmd (no rescheduling); `viewFull()`/`viewCompact()` render static indicator
- `panel.go`: `renderMessages()` replaces `RainbowGradient()` with `StyleMuted.Render("thinking...")`
- `ui_event_handlers.go`: `handleTabFlash()` returns early when `reduceMotion` is true
- `interfaces.go`: `SetReduceMotion(v bool)` on `claudePanelWidget`

### Testing

- `reduce_motion_test.go` — **new file**: setting propagation, tab flash suppression, interface compliance
- `panel_test.go` — reduce-motion ultrathink indicator test
- `statusline_test.go` — reduce-motion spinner suppression test

---

---

## UX-025: Cost Flash-on-Change (Opt-In)

> **Added in:** UX Redesign P3 sprint (commit `9d2e4b36`)

### Purpose

Provide a visual cue when session cost increases, helping cost-conscious users notice spend in real time. Made opt-in per Braintrust recommendation (Einstein flagged anxiety risk for non-billing users).

### Design Decision

**Time-limited bright flash** using `costFlashUntil time.Time`. When SessionCost increases and the feature is enabled, the cost badge renders in bright white bold for 500ms, then reverts to normal threshold coloring. Double-gated: requires both opt-in AND reduce-motion off.

| Condition | Flash Behavior |
|-----------|---------------|
| Disabled (default) | No flash, normal threshold colors |
| Enabled + reduce-motion off | 500ms bright white bold on cost increase |
| Enabled + reduce-motion on | No flash (WCAG 2.3.1 compliance) |
| Cost decreases or unchanged | No flash |

### Implementation

- `CostFlashEnabled bool`, `costFlashUntil time.Time`, `prevCost float64` on `StatusLineModel`
- `CheckCostFlash() tea.Cmd` — called by `cli_event_handlers.go` after every `SessionCost` update
- `activeCostStyle()` — returns bright white bold during active flash, otherwise `costStyle()`
- `CostFlashExpiredMsg` + `scheduleFlashExpiry()` — 500ms tick clears flash
- Both `viewFull()` and `viewCompact()` use `activeCostStyle()`
- Settings toggle: "Cost Flash" in Display section (default off)
- Wired via `handleSettingChanged` → `m.statusLine.CostFlashEnabled`

### Testing

- 5-case table test: enabled+increase, disabled, reduce-motion, decrease, same cost
- Flash expiry test: `CostFlashExpiredMsg` zeros `costFlashUntil`
- Default-off assertion
- `activeCostStyle` bright-white sub-test

---

## Cross-References

- **Depends on:** [[phase10-visual-foundation]] — TUI-044 (semantic colors), TUI-045 (icons), TUI-046 (theme switching)
- **Consumed by:** [[phase10-parity-features]] — TUI-057 (plan mode UX uses status line extensions)
- **Consumed by:** [[phase10-navigation-interaction]] — TUI-062 (vim mode indicator in status line)
- **Extended by (P2):** UX-016 (sparkline dots), UX-017 (adaptive layout), UX-020 (reduce-motion)
- **Extended by (P3):** UX-025 (cost flash)
- **UX-020 consumed by (P3):** UX-023 (pulse animation), UX-025 (cost flash) — both respect reduce-motion flag

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Updated with UX Redesign P3 (2026-04-12)._
