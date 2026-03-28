# Phase 10f+g: Polish, Animation & Dashboard

> **Tickets:** TUI-063, TUI-064, TUI-065, TUI-066, TUI-067, TUI-068
> **Status:** All complete
> **Packages:** `breadcrumb/`, `util/`, `skeleton/`, `modals/`, `dashboard/`
> **Purpose:** Add the finishing touches that distinguish a good TUI from a great one.

---

## TUI-063: Breadcrumb Navigation Trail

### Purpose

Show the user's current navigation path as a breadcrumb trail. Provides spatial orientation in a multi-panel TUI where context can be non-obvious.

### Design Decision

**Arrow-separated breadcrumb** using theme icons. Last item renders in normal style (current location), ancestors render muted. Width truncation removes leftmost items first (preserving current location).

### Implementation

- **New package:** `components/breadcrumb/`
- `BreadcrumbItem{Label, Key}` structs
- `SetCrumbs([]string)` for simple interface, `SetCrumbItems([]BreadcrumbItem)` for full items
- Arrow separator from `theme.Icons().Arrow` (respects ASCII/Unicode)
- Width truncation from left with "..." prefix when trail exceeds available space
- **breadcrumbWidget interface** in interfaces.go, `breadcrumbHeight=1` in layout
- **updateBreadcrumbs()** sets 7 context trails based on focus + rightPanelMode

### Keyboard Shortcuts

Breadcrumbs update automatically on:

| Action | Trail Example |
|--------|---------------|
| Tab to Claude | `Home > Chat` |
| Tab to right panel | `Home > Agents` or `Home > Settings` |
| Cycle right panel | `Home > Dashboard` or `Home > Teams` |
| Open plan | `Home > Chat > Plan Mode` |
| Search overlay | `Home > Chat > Search` |
| Vim h/l navigation | Updates on focus change |

### Testing

- 17 tests covering all 7 contexts, width truncation, arrow rendering
- 98.2% coverage (highest in Phase 10)

---

## TUI-064: Spring Animation Framework (Harmonica)

### Purpose

Enable smooth, physics-based animations in the TUI. Spring animations feel natural and responsive — they overshoot slightly and settle, unlike linear or eased transitions.

### Design Decision

**charmbracelet/harmonica** — Official Charm library, integrates naturally with Bubbletea tick cycle. O(1) spring update per frame. Added as a direct dependency in go.mod.

| Alternative | Why Not Chosen |
|-------------|----------------|
| CSS-like transition library | None exists for terminal |
| Manual easing functions | Reinventing the wheel |
| No animation | Less polished UX |

### Implementation

- **harmonica v0.2.0** added to go.mod
- **util/animate.go:** `SpringAnimation` wrapping `harmonica.Spring`
- `NewSpring(angularFrequency, dampingRatio)`: Typical values 6.0, 0.5
- `SetTarget()`, `Tick()`, `Value()`, `IsSettled()` — settle threshold = 0.1
- **AnimateTickMsg + AnimateTickCmd()** for 60fps (16.67ms) tick scheduling
- **Consumer pattern:** `Tick()` in Update(), `AnimateTickCmd()` until settled

### Usage

```go
spring := util.NewSpring(6.0, 0.5) // frequency, damping
spring.SetTarget(100.0)             // animate to value 100
// In Update():
spring.Tick()
if !spring.IsSettled() {
    return m, util.AnimateTickCmd()  // schedule next 16ms frame
}
// In View():
position := spring.Value() // smoothly approaches 100
```

### Testing

- 13 convergence tests verifying spring settles within expected iterations
- 94.1% coverage
- Used by: TUI-065 (skeleton shimmer), TUI-061 (tab flash timing reference)

---

## TUI-065: Skeleton Loading Screens

### Purpose

Display placeholder content while the CLI subprocess initializes. Skeleton screens reduce perceived load time and prevent the "blank screen" experience during startup.

### Design Decision

**4 variants** matching the TUI's panel structure. Shimmer animation using AnimateTickMsg from TUI-064. 500ms threshold guard prevents flash of skeleton on fast connections.

### Implementation

- **New package:** `components/skeleton/`
- **4 variants:** SkeletonConversation, SkeletonAgentTree, SkeletonSettings, SkeletonDashboard
- **Shimmer:** Lighter band slides left-to-right, row-staggered for organic feel
- **500ms threshold guard:** `ShouldShow(elapsed)` returns true only when CLI init exceeds 500ms (prevents flash on fast startup)
- **lineSpec pattern:** indent + widthFrac ratios cycled to fill height, scales to any terminal width
- **Settings variant:** Separate key (25%) + value (50%) columns with gap
- Theme integration: `config.ColorMuted` (base) + `config.ColorSecondary` (shimmer highlight)

### Usage

Skeleton screens appear automatically during CLI startup if initialization takes longer than 500ms. They are replaced by actual content once the first CLI events arrive.

### Testing

- 37 tests covering all 4 variants, shimmer positions, threshold guard
- 94.6% coverage

---

## TUI-066: Rich Modal Styling

### Purpose

Upgrade modals from uniform appearance to type-specific visual treatment. Permission modals should look different from informational modals — visual cues communicate urgency and type.

### Design Decision

**4 visual upgrades** to the existing modal system, keeping the core modal queue mechanics unchanged.

### Implementation

1. **Type-specific border colors:**
   - Permission modals → `ColorWarning` (yellow border)
   - All other modals → `ColorPrimary` (cyan border)
   - `modalBorderColor()` helper replaces static `modalBorderStyle`

2. **Header icons:**
   - `headerIcon()` prepends type-specific icon from `config.UnicodeIcons`
   - Error detection: if header contains "error" (case-insensitive), uses `Icons().Error`

3. **Shadow effect:**
   - `appendRightShadow()` + `renderShadow()` — block characters on right/bottom edges
   - Creates depth perception for floating modals

4. **Radio buttons:**
   - `renderOptionList()` uses `(*)` / `( )` instead of `>` / space prefix
   - Clearer selection state for multi-option modals

### Testing

- 24 new tests covering border colors, header icons, shadow rendering, radio buttons
- 89.8% modals coverage

---

## TUI-067: Two-Step Confirmation Dialogs

### Purpose

Prevent accidental execution of destructive actions. Users must type a confirmation phrase (e.g., "delete") to proceed, providing a deliberate friction point for irreversible operations.

### Design Decision

**Type-to-confirm pattern** for destructive actions. The modal shows the required phrase; the user must type it exactly to enable the confirm button.

### Implementation

- **`modals/confirm2.go`**: TwoStepConfirmModal struct with textinput, phrase matching, progress rendering
- **phraseMatches()**: Case-insensitive comparison with `strings.TrimSpace` on input
- **renderProgress()**: Visual prefix match — green for matched portion, dim for unmatched
- Requires exact phrase match to enable Enter (Enter ignored before match)
- Cancel (Esc) always available regardless of typed text
- DangerStyle border (ColorError red) from TUI-066, error icon in header
- `WithRequestID()` and `WithResponseCh()` for bridge/queue integration
- Hint line changes: "type phrase to enable enter" → "enter: confirm" when matched

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| (type phrase) | Enter confirmation phrase |
| Enter | Confirm (only enabled after exact match) |
| Esc | Cancel |

### Testing

- 24 tests covering construction, phrase matching, enter key behavior, escape, responseCh, view rendering
- Table-driven phrase matching: 10 cases (exact, case-insensitive, partial, no match, whitespace trimming)
- Table-driven enter key: 4 cases (after match, case-insensitive match, partial, empty)
- 91.2% modals coverage

---

## TUI-068: Dashboard Collapse/Expand

### Purpose

The session dashboard has 4 data sections that can be individually collapsed/expanded. Users focus on the metrics they care about and hide the rest.

### Design Decision

**Collapsible sections with cursor navigation** following the proven agent tree pattern. Section 0 (Session Overview) starts expanded; sections 1-3 start collapsed.

### Implementation

- **DashboardSection struct:** `Title string`, `Expanded bool`
- 4 sections: Session Overview (expanded), Performance, Agent Activity, System Health (collapsed)
- **Cursor navigation:** up/k, down/j between section headers
- **Enter/Space:** Toggle section expand/collapse
- **Update(msg tea.Msg) tea.Cmd** — pointer receiver, no-ops when unfocused
- **SetFocused(bool)** — propagated via `syncFocusState` when RPMDashboard active
- **View():** Sections rendered with icons (triangle expanded / circle collapsed), selected+focused headers reverse-colored
- **dashboardWidget interface extended:** `Update(tea.Msg) tea.Cmd` + `SetFocused(bool)`
- **key_handlers.go:** `handleAgentsKey` routes to dashboard when `rightPanelMode == RPMDashboard`

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Up / k | Move to previous section |
| Down / j | Move to next section |
| Enter / Space | Toggle expand/collapse |

### Usage

Navigate to Dashboard tab. Use arrow keys to move between sections. Enter to expand a collapsed section and see its metrics.

### Testing

- 35 new tests (47 total), 94.9% coverage
- Cursor boundaries, toggle states, focus propagation, icon rendering

---

## Cross-References

- **Depends on:** [[phase10-visual-foundation]] (TUI-044 semantic colors for modal borders/badges, TUI-045 icons for headers/breadcrumbs/dashboard)
- **Depends on:** TUI-064 (animation framework) feeds TUI-065 (skeleton shimmer)
- **Consumed by:** TUI-068 dashboard uses all Phase 10 visual infrastructure
- **Interacts with:** [[phase10-navigation-interaction]] (TUI-060 hint bar shows breadcrumb context, TUI-063 breadcrumbs use same focus tracking)

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Generated by TUI-069._
