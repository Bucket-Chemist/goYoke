# Phase 10d: Parity Features

> **Tickets:** TUI-052, TUI-053, TUI-054, TUI-055, TUI-056, TUI-057
> **Status:** All complete
> **Packages:** `config/`, `slashcmd/`, `claude/`, `taskboard/`, `modals/`, `statusline/`, `model/`
> **Purpose:** Deliver the highest user-impact features that users notice as "missing from the Node.js TUI."

---

## TUI-052: Shift+Tab Reverse Navigation

### Purpose

Implement standard keyboard convention where Shift+Tab reverses the Tab direction. This is the most common keyboard navigation pattern in desktop and web UIs.

### Design Decision

**BREAKING CHANGE:** Shift+Tab was rebound from CycleProvider to ReverseToggleFocus. CycleProvider moved to Alt+P. This was flagged as a breaking change in the specs but accepted because the standard convention is more important than backward compatibility with an unusual binding.

### Implementation

- Shift+Tab now calls `FocusPrev()` (reverse of Tab's `FocusNext()`)
- CycleProvider moved to Alt+P
- 5 existing tests migrated per review condition C-1

### Keyboard Shortcuts

| Key | Before (Phase 9) | After (Phase 10) |
|-----|-------------------|-------------------|
| Tab | Focus next | Focus next (unchanged) |
| Shift+Tab | CycleProvider | **Reverse focus** |
| Alt+P | — | CycleProvider (moved here) |

### Testing

- 5 migrated tests + 5 new tests
- Verifies: reverse focus, wrap-around, opposite-of-tab, no CycleProvider on shift+tab, alt+P still works

---

## TUI-053: Slash Command Dropdown

### Purpose

Provide autocomplete-style slash command discovery. When a user types `/` in the chat input, a dropdown appears showing matching commands with descriptions.

### Design Decision

**Dropdown component (not modal)** because dropdowns allow continued typing while modals block input. The UX must feel like autocomplete, not interruption.

### Implementation

- **New package:** `components/slashcmd/`
- 18 commands sourced from CLAUDE.md skill registry
- Prefix filter (case-insensitive), up/down navigation
- Enter → `SlashCmdSelectedMsg`, Esc → hide
- Scroll window (`maxVisible=8`), lipgloss border with up/down scroll indicators

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| / (in chat) | Open dropdown |
| Up / Down | Navigate commands |
| Enter | Select command |
| Esc | Close dropdown |
| (continue typing) | Filter commands by prefix |

### Usage

Type `/` in the chat input to see all available commands. Continue typing to filter (e.g., `/exp` shows `/explore`). Arrow keys to select, Enter to execute.

### Testing

- 42 tests covering prefix matching, navigation, scroll, selection
- 92.1% coverage

---

## TUI-054: Slash Command Execution via CLI

### Purpose

Wire the slash command dropdown into the TUI's execution pipeline. Local commands execute immediately in the TUI; remote commands are passed through to the Claude CLI.

### Design Decision

**Two-tier execution:** Simple utility commands (`/clear`, `/help`) handled locally in the TUI for instant response. All other commands forwarded to CLI via `sendMessage`.

### Implementation

- Wired slashcmd into `claude/panel.go`: `/` triggers dropdown, filter on typing, hide on non-`/`
- **Local commands:**
  - `/clear` — Clears conversation messages
  - `/help` — Shows system message with command list
- **Remote commands:** All others sent to CLI via `sendMessage`
- Tab-complete inserts command text + space
- `SlashExecutedMsg` emitted for telemetry tracking

### Testing

- 15 new tests
- 84.8% claude panel coverage
- Verifies: local execution, remote passthrough, tab completion

---

## TUI-055: Task Board Enhancement

### Purpose

Upgrade the task board from display-only to interactive with filtering, navigation, and detailed status. Users can quickly find and inspect specific tasks.

### Design Decision

**HandleMsg added to taskBoardWidget interface** (review condition M-1). Package doc updated from "display-only" to "interactive."

### Implementation

- **TaskFilterMode enum:** All, Running, Pending, Done
- **Filter shortcuts:** `a` (all), `r` (running), `p` (pending), `d` (done)
- **Navigation:** Up/down cursor movement between tasks
- **Progress summary header:** Shows counts per status
- **Status badges** with semantic colors from TUI-044
- **Expanded detail** view on selected task

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| a | Show all tasks |
| r | Filter: running only |
| p | Filter: pending only |
| d | Filter: done only |
| Up / k | Move cursor up |
| Down / j | Move cursor down |
| Enter | Toggle detail view |

### Testing

- 16 new tests (33 total)
- 94.0% coverage
- Filter mode switching, cursor boundary, status badge rendering

---

## TUI-056: Plan Preview Modal (Glamour)

### Purpose

Render implementation plans as full-screen modals with rich Markdown formatting via Glamour. Plans can be long and complex — a scrollable modal provides better reading than inline display.

### Design Decision

**Full-screen Glamour viewport** in a new `PlanViewModal`. Content is pre-rendered in `SetContent()` (never in `View()`) for performance. Scrollable via viewport component.

### Implementation

- **PlanView** added to ModalType enum
- New `modals/plan_modal.go`: PlanViewModal with Glamour pre-rendering
- Alt+V triggers only when `RPMPlanPreview` is the active right panel mode
- `planPreviewWidget.Content()` provides plan text
- `planViewModal` stored in sharedState
- Layout overlay when active (fullscreen over main content)

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Alt+V | Open plan preview (when plan is active) |
| Up / Down / j / k | Scroll plan content |
| Page Up / Page Down | Scroll by page |
| Esc / q | Close modal |

### Testing

- 22 tests for modal rendering, scrolling, key handling
- modals/ 89.0% coverage

---

## TUI-057: Plan Mode UX Improvements

### Purpose

Provide clear visual feedback when Claude is operating in plan mode. Users need to know: (1) that plan mode is active, (2) which step is being executed, and (3) when plan mode ends.

### Design Decision

**Status line indicator** with step tracking. PlanStepMsg parsed from CLI output using regex. Toast notification on plan mode activation bridges to TUI-056 plan modal.

### Implementation

- **PlanStepMsg:** `Active bool`, `Step int`, `Total int` in messages.go
- **StatusLineModel extended:** `PlanActive`, `PlanStep`, `PlanTotalSteps` fields
- **View() renders:** `[PLAN MODE: step N/M]` or `[PLAN MODE]` with WarningStyle
- **parsePlanStep() regex:** `(?i)\b(?:step|phase)\s+(\d+)\s*(?:of|/)\s*(\d+)\b`
- **handlePlanStep():** Sets fields, emits toast on inactive→active transition
- **Toast message:** "Plan mode active — press alt+v to view plan"

### Usage

When Claude enters plan mode, the status line shows:
```
[PLAN MODE: step 2/5]  session: $0.42  ctx: 34%  ...
```

A toast notification appears on first activation directing users to Alt+V for the full plan view.

### Testing

- 9 new test functions: 6 statusline (incl 4 subtests for steps), 4 model (plan msg handling + regex patterns)
- 89.2% statusline coverage

---

## Cross-References

- **Depends on:** [[phase10-visual-foundation]] (TUI-044 semantic colors for badges/indicators), [[phase10-settings-accessibility]] (TUI-048 status line extensions)
- **Consumed by:** [[phase10-navigation-interaction]] (TUI-060 hint bar context includes plan mode hints)
- **Related:** TUI-043 (app.go decomposition, required for key handler routing)

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Generated by TUI-069._
