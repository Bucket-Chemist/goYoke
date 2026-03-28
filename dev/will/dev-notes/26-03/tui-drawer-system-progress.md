# TUI Drawer System — Progress Notes

> **Date:** 2026-03-26
> **Ticket System:** `tickets/tui-drawer-system/`
> **Spec:** `tickets/tui-drawer-system/spec.md`
> **Review:** `tickets/tui-agent-upgrade/review-critique.md` (staff-architect, APPROVE_WITH_CONDITIONS)

---

## Context

New ticket system created 2026-03-26 for three interconnected TUI features:
1. **Chat viewport scrollbar** (Phase 1)
2. **Interactive drawer system** replacing full-screen modal overlays (Phase 2-3)
3. **Dynamic focus ring** including expanded drawers (Phase 4)

9 tickets, ~12 files (3 new packages, 6 modified).

### Relationship to Previous Work

- Builds on top of completed GOgent-109 through GOgent-116 (TUI foundation tickets)
- The Claude panel (`panel.go`) was already complete from TUI-CLI-03 / GOgent-118 equivalent work
- The `model/interfaces.go` pattern (unexported widget interfaces to break import cycles) is well-established from TUI-032 onward
- DrawerStack follows the same widget interface pattern as `dashboardWidget`, `claudePanelWidget`, etc.

### Previous Ticket Systems (Cross-Reference)

| System | Location | Status |
|--------|----------|--------|
| GOgent-TUI (GOgent-109–121) | `dev/will/migration_plan/tickets/TUI/` | Phases 0-3 complete, 4-5 pending |
| tui-drawer-system (TDS-001–009) | `tickets/tui-drawer-system/` | **Active — this session** |
| tui-agent-upgrade | `tickets/tui-agent-upgrade/` | Spec + review complete, no tickets generated yet |
| codebase-map (CM-001–012) | `tickets/codebase-map/` | All pending (Phase 0 ready) |
| EM-Deconvoluter (PREP/VIS) | External project | All complete |

---

## Session Progress (2026-03-26)

### Completed This Session

| Ticket | Title | Agent | Cost | Duration |
|--------|-------|-------|------|----------|
| TDS-001 | Scrollbar rendering component | *pre-existing* | — | — |
| TDS-002 | Integrate scrollbar into Claude panel | go-tui (sonnet) | $0.32 | 44s |
| TDS-003 | Drawer component model (stack.go + tests) | go-tui (sonnet) | $0.81 | 4m52s |
| TDS-004 | Drawer widget interface and messages | go-tui (sonnet) | $0.41 | 59s |
| TDS-009 | Wire DrawerStack in main.go | go-tui (sonnet) | $0.30 | 27s |
| **Total** | | | **$1.84** | **~7m** |

### Key Findings

1. **TDS-001 was already implemented** — scrollbar package existed with 100% test coverage. Spec audit confirmed full compliance. Only TDS-002 (integration) was missing, which is why the scrollbar wasn't visible in the UX.

2. **TDS-003 was partially done** — `drawer.go` (DrawerModel) existed but `stack.go` (DrawerStack) and `drawer_test.go` were missing. Agent completed both with 98.1% coverage.

3. **Interface pattern is clean** — `drawerStackWidget` uses proxy methods (`SetOptionsContent`, `ClearOptionsContent`, etc.) to avoid returning concrete types through the interface. This prevents import cycles while keeping the API ergonomic.

4. **Mouse support was already enabled** — `tea.WithMouseCellMotion()` was present in `main.go` from earlier work. TDS-009 only needed the drawer import + 2 lines of wiring.

### Completed (continued — same session)

| Ticket | Title | Agent | Cost | Duration |
|--------|-------|-------|------|----------|
| TDS-005 | Wire drawers into right panel layout | go-tui (sonnet) | $0.33 | 61s |
| TDS-006 | Wire BridgeModalRequestMsg to options drawer | go-tui (sonnet) | $1.03 | 2m47s |
| TDS-007 | Wire plan content to plan drawer | go-tui (sonnet) | $1.01 | 2m37s |
| TDS-008 | Implement dynamic focus ring | go-tui (sonnet) | $0.93 | 3m23s |
| **Session Total** | **9 tickets** | | **$4.14** | **~18m agent time** |

### ALL TICKETS COMPLETE

The tui-drawer-system is fully implemented. 9/9 tickets completed in a single session.

---

## Files Modified This Session

| File | Change | Ticket |
|------|--------|--------|
| `internal/tui/components/scrollbar/scrollbar.go` | Pre-existing (verified) | TDS-001 |
| `internal/tui/components/scrollbar/scrollbar_test.go` | Pre-existing (verified) | TDS-001 |
| `internal/tui/components/claude/panel.go` | Added `viewportWithScrollbar()`, modified `View()` and `SetSize()` | TDS-002 |
| `internal/tui/components/drawer/stack.go` | **NEW** — DrawerStack with height distribution | TDS-003 |
| `internal/tui/components/drawer/drawer_test.go` | **NEW** — 27 tests, 98.1% coverage | TDS-003 |
| `internal/tui/model/interfaces.go` | Added `drawerStackWidget` interface | TDS-004 |
| `internal/tui/model/messages.go` | Added `DrawerContentMsg`, `DrawerMinimizeMsg` | TDS-004 |
| `internal/tui/model/app.go` | Added `drawerStack` field to `sharedState` | TDS-004 |
| `internal/tui/model/setters.go` | Added `SetDrawerStack()` method | TDS-004 |
| `cmd/gofortress/main.go` | Added drawer import + DrawerStack wiring | TDS-009 |

---

## Architecture Notes

### Drawer Layout Strategy

Drawers live inside the right panel (not a separate column). When expanded:
- 1 expanded: gets full height minus 1 row for minimized tab
- 2 expanded: 50/50 split
- 0 expanded: each gets 1 row (minimized tab only)
- Compact layout tier: drawers suppressed entirely

### Import Cycle Prevention

```
drawer → config (colors, styles)
model  → state (types)
main   → drawer + model (wiring)
```

`model/interfaces.go` defines `drawerStackWidget` (unexported) so `model` never imports `drawer`. The `SetDrawerStack` setter accepts the interface, and Go's structural typing ensures `*drawer.DrawerStack` satisfies it at compile time.

---

## Additional Files Modified (TDS-005 through TDS-008)

| File | Change | Ticket |
|------|--------|--------|
| `internal/tui/model/layout.go` | Drawer compositing in renderRightPanel() | TDS-005 |
| `internal/tui/model/app.go` | DrawerContentMsg/MinimizeMsg + drawer.ModalResponseMsg + drawer.PlanViewRequestMsg handlers | TDS-005, TDS-006, TDS-007 |
| `internal/tui/model/ui_event_handlers.go` | handleBridgeModalRequest drawer routing + handlePlanStep drawer push + handleWindowSize drawer propagation | TDS-005, TDS-006, TDS-007 |
| `internal/tui/model/interfaces.go` | Extended drawerStackWidget with modal methods | TDS-006 |
| `internal/tui/components/drawer/drawer.go` | Modal state fields + ModalResponseMsg + PlanViewRequestMsg + dynamic HandleKey | TDS-006, TDS-007 |
| `internal/tui/components/drawer/stack.go` | Modal proxy methods | TDS-006 |
| `internal/tui/model/focus.go` | FocusPlanDrawer + FocusOptionsDrawer + FocusRing + FocusNextInRing + FocusPrevInRing | TDS-008 |
| `internal/tui/model/key_handlers.go` | Dynamic ring Tab/Shift+Tab + handleDrawerKey + syncFocusState/breadcrumbs/hints for drawers | TDS-008 |
| `internal/tui/model/focus_test.go` | 30 new tests for ring functions | TDS-008 |

## Documentation Updates This Session

- Created `dev/will/dev-notes/26-03/tui-drawer-system-progress.md` (this file)
- Fixed 5 stale GOgent-TUI tickets (GOgent-117–121) — marked completed
- All ticket indexes verified consistent

## Notes for Next Session

- The tui-drawer-system is **complete** — all 9 tickets done
- Next ticket systems with pending work:
  - **tui-agent-upgrade** — spec + review done, needs ticket generation
  - **codebase-map** (CM-001–012) — all pending, Phase 0 ready
- Manual testing recommended: run the TUI with a live team to verify drawer rendering, modal interaction, and focus ring behavior end-to-end
