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

| Tier | Width | Panel Split | Rationale |
|------|-------|-------------|-----------|
| Compact | <80 | Single panel | Narrow SSH/mobile terminal |
| Standard | 80-119 | 75/25 (80-99) or 70/30 (100-119) | Common terminal widths |
| Wide | 120-179 | 60/40 | Wide terminal / split screen |
| Ultra | >=180 | 50/50 | Ultra-wide monitor |

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

## Cross-References

- **Depends on:** TUI-043 (app.go decomposition for key handler routing), [[phase10-settings-accessibility]] (TUI-050 settings panel for vim toggle, TUI-058 SetTier interfaces)
- **Consumed by:** [[phase10-polish-animation]] (TUI-063 breadcrumbs use focus context from hint bar)
- **Interacts with:** [[phase10-parity-features]] (TUI-052 Shift+Tab, TUI-057 plan mode) — hint bar shows context-appropriate shortcuts for these features

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Generated by TUI-069._
