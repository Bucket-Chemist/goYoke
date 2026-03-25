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

## Cross-References

- **Depends on:** [[phase10-visual-foundation]] — TUI-044 (semantic colors), TUI-045 (icons), TUI-046 (theme switching)
- **Consumed by:** [[phase10-parity-features]] — TUI-057 (plan mode UX uses status line extensions)
- **Consumed by:** [[phase10-navigation-interaction]] — TUI-062 (vim mode indicator in status line)

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Generated by TUI-069._
