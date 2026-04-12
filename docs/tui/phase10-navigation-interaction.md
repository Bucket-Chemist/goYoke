# Phase 10e: Navigation & Interaction

> **Tickets:** TUI-058, TUI-059, TUI-060, TUI-061, TUI-062
> **Status:** All complete
> **Packages:** `model/`, `search/`, `hintbar/`, `tabbar/`, `config/`, `state/`, `claude/`, `agents/`
> **Purpose:** Layer interaction refinements that make the TUI feel professional: responsive layout, fuzzy search, keyboard hints, and vim bindings.

---

## TUI-058: 4-Tier Responsive Layout

### Purpose

Adapt the TUI layout to the actual terminal width. A narrow SSH terminal gets a compact layout; an ultra-wide monitor gets balanced panels. The layout should feel intentional at every size.

### Design Decision

**4-tier LayoutTier enum** covering real-world terminal widths from 80-char SSH to ultra-wide monitors. Components branch on tier for adaptive rendering.

| Tier | Width | Panel Split (original) | Rationale |
|------|-------|------------------------|-----------|
| Compact | <80 | Single panel | Narrow SSH/mobile terminal |
| Standard | 80-119 | 75/25 (80-99) or 70/30 (100-119) | Common terminal widths |
| Wide | 120-179 | 60/40 | Wide terminal / split screen |
| Ultra | >=180 | 50/50 | Ultra-wide monitor |

> **Note:** These fixed ratios were superseded by [[#UX-021 Focus-Driven DrawerContent Split|UX-021]] which makes ratios focus-dependent. The tiers remain; the ratios within each tier now vary by focus target.

### Implementation

- **LayoutTier enum:** Compact, Standard, Wide, Ultra
- `computeLayout()` extended with tier selection and per-tier split ratios
- Standard preserves Phase 1-9 sub-breakpoints (75/25 for 80-99, 70/30 for 100-119)
- `tier` field added to `layoutDims` struct
- **SetTier()** added to 7 widget interfaces: claude, toast, dashboard, settings, telemetry, planpreview, taskboard
- All 7 implementations have no-op `SetTier()` (future tickets add tier-aware rendering)
- Mock types updated in app_test.go, app_coverage_test.go, bench_test.go

### Testing

- New `layout_test.go`: 14 boundary subtests at widths 79, 80, 99, 100, 119, 120, 179, 180
- LayoutTier String() tests
- 90.0% computeLayout coverage, 87.0% model total

---

## TUI-059: Unified Fuzzy Search Overlay

### Purpose

Provide a single search interface that searches across all TUI panels. Users can find conversation messages, agent entries, and settings from one overlay.

### Design Decision

**SearchSource interface** defined in `state/search.go` (not in search/ package) to avoid import cycles (review condition M-3). Multi-source search with score-based sorting.

### Implementation

- **New package:** `components/search/` — SearchOverlayModel with textinput + results list
- **SearchResult + SearchSource** in `state/search.go` (cycle-break)
- **searchOverlayWidget interface** in `model/interfaces.go`
- **HandleMsg** pointer-receiver pattern (matches claudePanelWidget, toastWidget)
- **Ctrl+F** global keybinding triggers overlay
- **Z-order:** modals > search overlay > main (overlay not rendered when modal active)
- **Key dispatch priority:** planViewModal > searchOverlay > modalQueue > global keys
- **Search implementations:**
  - `ClaudePanelModel.Search()`: Case-insensitive substring, prefix scores higher
  - `AgentTreeModel.Search()`: Matches Type + Description, dedup per node

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Ctrl+F | Open search overlay |
| (type) | Filter results in real-time |
| Up / Down | Navigate results |
| Enter | Jump to selected result |
| Esc | Close overlay |

### Testing

- 43 new tests: 25 overlay, 9 claude search, 9 agent search
- 89.1% search coverage
- All 26 packages green

---

## TUI-060: Keyboard Hint Bar

### Purpose

Show context-sensitive keyboard shortcuts at the bottom of the TUI. Reduces the learning curve by displaying available actions for the current context.

### Design Decision

**5 context sets** that switch automatically based on the current focus and overlay state. Width-adaptive: truncates hints from right with "..." on narrow terminals.

### Implementation

- **New package:** `components/hintbar/`
- **5 contexts:**
  - Main: Tab / Shift+Tab / Ctrl+F / /
  - Settings: Up/Down / Enter / Esc
  - Search: (type) / Up/Down / Enter / Esc
  - Modal: Up/Down / Enter / Esc
  - Plan: Alt+V / Up/Down / Esc
- **hintBarWidget interface** in interfaces.go
- **Layout:** Renders between toasts and status line, `hintBarHeight=1` subtracted from content
- **updateHintContext():** Priority-ordered (plan → modal → search → settings → main)
- Context updates on: focus change, search overlay, modal activation, plan mode, right panel cycle

### Testing

- 22 tests covering all 5 contexts and width truncation
- 86.7% coverage

---

## TUI-061: Tab Highlight on Focus Change

### Purpose

Provide visual feedback when switching between tabs. A brief flash animation on the target tab helps users track their navigation.

### Design Decision

**300ms tick-based flash animation.** The flash uses the accent background color and naturally decays. Simple timing with `time.Since()` comparison.

### Implementation

- `flashTab` + `flashStart` fields on TabBarModel
- `TabFlashMsg` in messages.go, `tabFlashTickMsg` (unexported) for tick scheduling
- **HandleMsg** on tabBarWidget interface (pointer receiver)
- `ToggleFocus` / `ReverseToggleFocus` emit `TabFlashMsg` via `tabFlashCmd` helper
- `View()` uses flashStyle (accent background) when `flashTab == activeTab` within 300ms
- `scheduleFlashTick()` computes remaining duration, guarantees >=1ms tick

### Testing

- 7 new tests
- 91.5% tabbar coverage
- Verified: flash starts, decays after 300ms, does not interfere with normal rendering

---

## TUI-062: Vim-Style Keybindings (Optional)

### Purpose

Provide familiar navigation for vim users. Vim bindings are an overlay — standard arrow key navigation always works, vim adds h/j/k/l and mode switching.

### Design Decision

**Overlay approach** with VimEnabled bool on KeyMap. Non-destructive: when disabled, zero impact on key handling. Normal/Insert mode toggle gives vim users expected behavior without confusing non-vim users.

### Implementation

- **New file:** `config/vim_keys.go` — VimKeys struct (8 bindings), VimMode enum (Normal/Insert)
- `VimEnabled` + `VimMode` + `Vim` fields added to KeyMap (off by default)
- **handleVimKey() / handleVimNormalKey()** in key_handlers.go — overlaid before standard dispatch

| Mode | Key | Action |
|------|-----|--------|
| Normal | j | Down (synthetic arrow key) |
| Normal | k | Up (synthetic arrow key) |
| Normal | h | Focus prev |
| Normal | l | Focus next |
| Normal | g | Home |
| Normal | G | End |
| Normal | i | → Insert mode |
| Insert | Esc | → Normal mode |

- **SettingChangedMsg** "vim_keys" wires enable/disable + mode reset
- **Status line indicator:** `[NORMAL]` (muted) or `[INSERT]` (info/cyan) when vim enabled

### Keyboard Shortcuts

| Key | Normal Mode | Insert Mode |
|-----|-------------|-------------|
| h | Focus previous panel | (passthrough) |
| j | Move down | (passthrough) |
| k | Move up | (passthrough) |
| l | Focus next panel | (passthrough) |
| g | Scroll to top | (passthrough) |
| G | Scroll to bottom | (passthrough) |
| i | Enter insert mode | — |
| Esc | — | Enter normal mode |

### Usage

Enable in Settings → Display → Vim Keys → Toggle On. Status line shows current mode.

### Testing

- 22 new tests: config (8) + model (14)
- Mode transitions, key consumption, passthrough verification

---

---

## UX-021: Focus-Driven Drawer/Content Split

> **Added in:** UX Redesign P3 sprint (commit `9d2e4b36`)

### Purpose

The fixed panel ratios from TUI-058 treat all focus states equally — when the user focuses the right panel, it has the same width as when focused on chat. This wastes space: the focused panel should get more room.

### Design Decision

**Focus-aware ratio override** in `computeLayout()`. Replaces the fixed per-tier ratios with a `switch m.focus` inside each tier case. The existing Tab/Shift+Tab focus cycling (TUI-052) now causes visible layout shifts.

| Focus | Standard | Wide | Ultra |
|-------|----------|------|-------|
| `FocusClaude` | 55/45 | 55/45 | 50/50 |
| `FocusAgents` | 70/30 | 65/35 | 60/40 |
| Drawer focuses (Plan/Options/Teams) | 30/70 | 35/65 | 40/60 |

Effect is most dramatic at Standard tier (55→70→30), most subtle at Ultra (50→60→40). Compact tier unchanged (single column).

### Implementation

- `computeLayout()` in `layout.go`: replaced fixed `leftRatio` assignments with `switch m.focus` blocks per tier
- Previous sub-breakpoints (75/25 at 80-99, 70/30 at 100-119) removed — focus-driven ratios subsume them
- All drawer focus types handled: `FocusPlanDrawer`, `FocusOptionsDrawer`, `FocusTeamsDrawer`
- No new keybindings needed — existing Tab/Shift+Tab focus ring drives the layout

### Testing

- `TestComputeLayout_FocusAwareRatios` — 23-case table covering every focus × tier combination
- Updated pre-existing ratio tests to expect UX-021 values
- Updated `app_test.go` assertions for new defaults

---

## UX-022: Tree Density Toggle

> **Added in:** UX Redesign P3 sprint (commit `9d2e4b36`)

### Purpose

Different tasks need different levels of agent detail. Scanning 20 agents needs compact; debugging one agent needs verbose. Let users cycle between views.

### Design Decision

**3-mode TreeDensity enum** cycled by `alt+d`. Context-sensitive: only fires when agent panel is focused. Added to help modal for discoverability.

| Mode | Rendering | Use Case |
|------|-----------|----------|
| Standard (default) | Two-column dot-leader from UX-008 | Normal operation |
| Compact | Icon + 2-char uppercase abbreviation | Scanning many agents |
| Verbose | Standard + indented metadata line (status, tier, duration, cost) | Debugging |

### Implementation

- `TreeDensity` type with `DensityStandard`, `DensityCompact`, `DensityVerbose`
- `CycleDensity()` advances through Standard → Compact → Verbose → wrap
- `renderCompactDensity()` + `renderCompactNode()`: tree prefix + icon + abbreviation
- `renderVerboseDensity()` + `renderVerboseMeta()`: standard row + indented metadata
- `alt+d` in `config/keys.go` AgentKeys group
- Dispatched in `key_handlers.go` `handleAgentsKey()` when FocusAgents + RPMAgents
- Added to help modal Agent Panel section

### Testing

- 14 table-driven tests: density cycling, compact rendering, verbose metadata, empty tree, wrapping

---

## UX-023: Pulse Animation on Active Agent

> **Added in:** UX Redesign P3 sprint (commit `9d2e4b36`)

### Purpose

Running agents should be visually distinct from pending ones. A subtle pulse draws the eye without being distracting. Respects reduce-motion for accessibility.

### Design Decision

**Lazy 500ms tick** that only runs when running agents exist. Toggles `pulseBright` boolean, which the icon renderer uses to alternate between bright and dim styles. Zero CPU cost when all agents are idle.

| State | Rendering |
|-------|-----------|
| Running + pulse bright | Bright green icon |
| Running + pulse dim | Dim green icon |
| Running + reduce-motion | Always bright (static) |
| Non-running | Normal status color (unchanged) |

### Implementation

- `pulseBright bool`, `tickRunning bool`, `reduceMotion bool` on `AgentTreeModel`
- `TreePulseTickMsg` + `SchedulePulseTick()` — 500ms tick
- `MaybeStartPulseTick()` — starts tick only when running agents exist and no tick in flight
- `hasRunningAgents()` scan; lazy reschedule in Update handler
- `SetReduceMotion(v bool)` wired from `handleSettingChanged`
- Icon rendering branches on `pulseBright` for StatusRunning agents

### Testing

- Pulse toggle on tick message
- Reduce-motion suppression (no tick rescheduled)
- No tick when no running agents
- `MaybeStartPulseTick` idempotency

---

## Cross-References

- **Depends on:** TUI-043 (app.go decomposition for key handler routing), [[phase10-settings-accessibility]] (TUI-050 settings panel for vim toggle, TUI-058 SetTier interfaces)
- **Consumed by:** [[phase10-polish-animation]] (TUI-063 breadcrumbs use focus context from hint bar)
- **Interacts with:** [[phase10-parity-features]] (TUI-052 Shift+Tab, TUI-057 plan mode) — hint bar shows context-appropriate shortcuts for these features
- **Extended by (P3):** UX-021 (focus-driven layout overrides TUI-058 fixed ratios), UX-022 (tree density), UX-023 (pulse animation)
- **UX-021 depends on:** TUI-058 (4-tier layout), TUI-052 (focus ring cycling)
- **UX-023 depends on:** UX-020 (reduce-motion flag from [[phase10-settings-accessibility]])

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Updated with UX Redesign P3 (2026-04-12)._
