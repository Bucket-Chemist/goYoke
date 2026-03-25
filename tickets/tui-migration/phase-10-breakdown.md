# Phase 10: UX Overhaul — Sub-Phase Breakdown

> **Scope:** 28 tickets (TUI-043 to TUI-070)
> **Planning date:** 2026-03-24
> **Workflow:** /plan-tickets (Scout → Planner → Architect → Staff Architect Review → Synthesis)
> **Review:** APPROVE_WITH_CONDITIONS (High Confidence)
> **Estimate:** 95–135 hours, 2–3 weeks parallel (5 concurrent tracks after TUI-043)
> **New packages:** settingstree, slashcmd, search, hintbar, breadcrumb, skeleton

---

## Sub-Phase 10a: Structural Prerequisite

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-043 | Extract key handlers from app.go | 4–6h | go-pro |

**Purpose:** app.go is 995 lines. This extraction keeps it under 400 lines as pure dispatch. Creates `key_handlers.go`, `message_handlers.go`, `view_helpers.go`. Prerequisite for all other Phase 10 work.

---

## Sub-Phase 10b: Visual Foundation

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-044 | Semantic color system | 3–4h | go-pro |
| TUI-045 | Icon library with Unicode and ASCII fallback | 3–4h | go-pro |
| TUI-046 | Theme switching infrastructure | 4–6h | go-pro |
| TUI-047 | Contextual error message formatting | 3–4h | go-pro |
| TUI-048 | Status line semantic colors and context-aware display | 3–4h | go-pro |
| TUI-049 | Token usage progress bar | 3–4h | go-pro |

**Key decisions:**
- TUI-046: Theme propagation via `activeTheme` in `sharedState` (review M-5)
- TUI-044: Semantic roles (Success, Warning, Error, Info, Muted, Accent, Border) replace hardcoded colors
- TUI-045: `IconSet` interface with `UnicodeIcons` and `ASCIIIcons` implementations

---

## Sub-Phase 10c: Interaction Enhancements

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-050 | Interactive settings panel with tree navigation | 4–6h | go-pro |
| TUI-051 | High-contrast mode | 2–3h | go-pro |
| TUI-052 | Shift+Tab reverse navigation | 3–4h | go-pro |
| TUI-053 | Slash command dropdown | 4–6h | go-pro |
| TUI-054 | Slash command execution via CLI | 3–4h | go-pro |
| TUI-055 | Task board enhancement | 3–4h | go-pro |
| TUI-056 | Plan preview modal with Glamour | 3–4h | go-pro |
| TUI-057 | Plan mode UX improvements | 3–4h | go-pro |

**Key decisions:**
- TUI-052: Includes migration of 5 specific tests (review C-1)
- TUI-055: `taskBoardWidget` interface extended with `HandleMsg(tea.Msg) tea.Cmd` (review M-1)
- TUI-050: New `settingstree` package with tree navigation model

---

## Sub-Phase 10d: Layout & Navigation

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-058 | 4-tier responsive layout | 4–6h | go-pro |
| TUI-059 | Unified fuzzy search overlay | 4–6h | go-pro |
| TUI-060 | Keyboard hint bar | 3–4h | go-pro |
| TUI-061 | Tab highlight on focus change | 2–3h | go-pro |
| TUI-062 | Vim-style keybindings (optional) | 3–4h | go-pro |
| TUI-063 | Breadcrumb navigation trail | 3–4h | go-pro |

**Key decisions:**
- TUI-059: `SearchSource` interface defined in `model/interfaces.go` (review M-3)
- TUI-058: 4 breakpoints — narrow (<80 cols), medium (80–120), wide (120–160), ultrawide (>160)
- TUI-060: New `hintbar` package, context-aware key hints

---

## Sub-Phase 10e: Polish

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-064 | Spring animation framework (harmonica) | 4–6h | go-pro |
| TUI-065 | Skeleton loading screens | 3–4h | go-pro |

**Key decisions:**
- TUI-064: Uses charmbracelet/harmonica for physics-based spring animations
- TUI-065: New `skeleton` package with shimmer/pulse effects for loading states

---

## Sub-Phase 10f: Modal & Dashboard

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-066 | Rich modal styling | 3–4h | go-pro |
| TUI-067 | Two-step confirmation dialogs | 3–4h | go-pro |
| TUI-068 | Dashboard metrics collapse/expand | 3–4h | go-pro |

---

## Sub-Phase 10g: Verification

| Ticket | Title | Estimate | Agent |
|--------|-------|----------|-------|
| TUI-069 | Obsidian vault documentation | 2–3h | tech-docs-writer |
| TUI-070 | Verify-parity refresh and integration test | 4–6h | go-pro |

**Purpose:** TUI-069 updates all vault documentation to reflect Phase 10 changes. TUI-070 refreshes `verify-parity.sh` and runs the full parity verification after all UX work is done.

---

## Dependency Graph

```
TUI-043 (structural prerequisite)
    │
    ├──► Track A: TUI-044 → TUI-045 → TUI-047 (visual → icons → errors)
    │                └──► TUI-046 → TUI-050 → TUI-051 (theme → settings → high-contrast)
    │                         └──► TUI-048, TUI-049 (status line, token bar)
    │
    ├──► Track B: TUI-052 → TUI-053 → TUI-054 (keybindings → slash dropdown → execution)
    │
    ├──► Track C: TUI-058 → TUI-059 → TUI-061 (responsive → search → tab highlight)
    │                └──► TUI-060 (hint bar), TUI-062 (vim keys), TUI-063 (breadcrumbs)
    │
    ├──► Track D: TUI-064 → TUI-065 (animations → skeletons)
    │
    └──► Track E: TUI-066, TUI-067, TUI-068 (modals, confirm, dashboard)
              │
              └──► TUI-069, TUI-070 (docs, final verification)
```

**Parallelizable:** Tracks A–E can run concurrently after TUI-043 completes.

---

## Review Conditions (all incorporated into tickets)

| Condition | Ticket | Resolution |
|-----------|--------|------------|
| C-1 (critical) | TUI-052 | Shift+Tab test migration — 5 specific tests listed |
| M-1 (major) | TUI-055 | TaskBoard interface extended with `HandleMsg` |
| M-3 (major) | TUI-059 | `SearchSource` interface in `model/interfaces.go` |
| M-5 (major) | TUI-046 | Theme propagation via `activeTheme` in sharedState |

---

## Links

- Full review: `.claude/sessions/20260323-plan-tickets-tui-phase10/review-critique.md`
- Specs: `.claude/sessions/20260323-plan-tickets-tui-phase10/specs.md`
- Strategy: `.claude/sessions/20260323-plan-tickets-tui-phase10/strategy.md`
- Ticket index: [[tickets/tui-migration/tickets/overview|overview.md]]
- Migration status: [[TUI Migration Status]]
