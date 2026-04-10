# GOgent-Fortress TUI: UX Redesign Specification

**Author:** UX Audit — April 2026
**Status:** Approved for implementation
**Branch:** Create from `routing-restructure` after PR merge
**Reference TUIs:** lazygit, gh-dash, spotify-tui, Bubbletea examples catalogue

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State Assessment](#2-current-state-assessment)
3. [Area 1: Right Panel Density & The 22-Column Problem](#3-area-1-right-panel-density--the-22-column-problem)
4. [Area 2: Agent Tree Legibility & Hierarchy](#4-area-2-agent-tree-legibility--hierarchy)
5. [Area 3: Conversation Panel & Chat UX](#5-area-3-conversation-panel--chat-ux)
6. [Area 4: Status Line & Progress Feedback](#6-area-4-status-line--progress-feedback)
7. [Area 5: Team & Workflow Monitoring](#7-area-5-team--workflow-monitoring)
8. [Priority Matrix](#8-priority-matrix)
9. [Implementation Notes](#9-implementation-notes)

---

## 1. Executive Summary

The GOgent-Fortress TUI's core value proposition is **real-time visibility into multi-agent orchestration** — something no other Claude Code interface provides. The current implementation is functionally complete but has critical UX gaps that undermine this value at the most common terminal widths.

The five areas targeted in this spec are:

| Area | Core Problem | Impact |
|------|-------------|--------|
| Right Panel Density | 22-column width renders agent data illegible | Critical — defeats the purpose of the TUI |
| Agent Tree Legibility | Truncated names, wasted space on box-drawing chars | High — core feature is unreadable |
| Conversation Panel | Wall of same-colored text, no turn separation | Medium — 80% of user time spent here |
| Status Line | Cost and context buried in dense text | High — safety-critical signals hidden |
| Team Monitoring | Teams only visible in expandable drawer | Medium — most expensive operations least visible |

**Design Philosophy:** Every recommendation follows three principles drawn from best-in-class TUIs:

1. **Progressive disclosure** (lazygit): Show summary by default, detail on demand
2. **Glanceability** (spotify-tui): Critical state readable in <1 second without focus
3. **Spatial mapping** (gh-dash): UI structure mirrors conceptual structure

---

## 2. Current State Assessment

### 2.1 Layout Structure (Current)

```
+------------------------------------------------------------------+
| GOgent Fortress (Go)                                              |
+------------------------------------------------------------------+
| GOgent-Fortress                                                   |
+------------------------------------------------------------------+
| Chat | Agent Config | Team Config | Telemetry                     |
| Anthropic | Google | OpenAI | Local / Ollama                      |
+------------------------------------------------------------------+
|                              |                                    |
|  Conversation Panel          |  Agent Tree        (22 cols)       |
|  (Claude + You messages)     |  > router: Router                 |
|  ~75% width                  |    ├─* staff-archi...              |
|                              |    ├─* team-run: te...             |
|                              |  ──────────────────                |
|                              |  Agent Detail                      |
|                              |    Overview / Context / Activity   |
|                              |                                    |
|                              +------------------------------------+
|                              | ⚙ Options (minimized)             |
|                              | 📋 Plan    (minimized)             |
|                              | 📊 Teams   (minimized/expanded)    |
+------------------------------------------------------------------+
| [M] [opus-4-6[1m]] [acceptEdits] ■ anthropic                     |
| agents:3 · "admin@exactmass.org" · ⏱ 162m 47s                    |
+------------------------------------------------------------------+
```

### 2.2 Terminal Width Distribution

The layout system defines 4 tiers:

| Tier | Width | Left/Right Split | Right Panel Width |
|------|-------|-------------------|-------------------|
| Compact | <80 | 100/0 | Hidden |
| Standard | 80-119 | 75/25 or 70/30 | **18-35 cols** |
| Wide | 120-179 | 60/40 | 46-70 cols |
| Ultra | 180+ | 50/50 | 88+ cols |

**The problem:** Most developers run terminals at 80-120 columns. At Standard tier, the right panel gets 18-35 characters of inner content width. This is where 90% of the agent visibility happens, and it's nearly illegible.

### 2.3 Key Observations from Screenshots

1. Agent names truncated mid-word (`staff-archite...`, `team-run: te...`)
2. Activity entries show full absolute paths that overflow into ellipsis
3. No visual separation between conversation turns
4. Cost display not prominent — buried in status line text
5. Context window percentage not visually represented
6. Teams drawer minimized by default — easy to miss
7. Agent tree uses 4-5 characters for box-drawing prefixes, wasting scarce horizontal space
8. All text is essentially the same green — no semantic color differentiation

---

## 3. Area 1: Right Panel Density & The 22-Column Problem

### 3.1 Problem Statement

The right panel exists to give users visibility into the agent harness. At `LayoutStandard` (80-119 cols), `rightWidth` is 18-35 characters. After subtracting border frame (2 cols), the inner content width drops to 16-33 characters. Agent names, file paths, and key-value pairs all compete for this space and lose.

**Evidence from debug log:**
```
computeDrawerLayout: tier=standard ... rightWidth=22
```

At 22 characters, the tree prefix (`├─* `) takes 4, leaving 18 for agent name + status. `staff-architect-critical-review` (33 chars) becomes `staff-archi...`.

### 3.2 Recommendation 1a: Adaptive Right-Panel Content

**Principle:** Responsive design, not just responsive sizing. The content strategy should change at each width tier, not just shrink.

#### Icon Rail Mode (rightWidth < 30)

When the panel is too narrow for readable text, switch to an icon-only rail — inspired by VS Code's Activity Bar. The rail shows status at a glance; detail is available via focus expansion or the detail panel.

```
CURRENT (22 cols):                    PROPOSED ICON RAIL (22 cols):
                                      
> router: Router                      ● R  router          
  [Bash: git status --                  ├ ● SA  $1.98       
  short]                                ├ ✕ TR  fail        
  ├─* staff-archite…                    ├ ● TR  $0.12       
  [Grep:                                ├ ● TR  $0.08       
  mcp\.RegisterAll|tools                └ ◻ TR  wait        
  \.RegisterA…]                                             
  ├─* team-run: te…                   ─────────────────────
  ├─* team-run: te…                   Overview              
  ├─* team-run: te…                   Running · opus · 162m 
  ├─* team-run: te…                   $1.98 · 0 tokens     
  ├─* team-run: te…                                        
                                      Activity (218)        
▼ Overview                            ✓ Bash git diff ...  
  Status:    Running                  ✓ Bash git add ...   
  Type:      router                   ✓ Bash git commit... 
  Model:     claude-                                       
             opus-4-6[1m]                                  
  Tier:      opus                                          
  Duration:  162m 47s                                      
  Cost:      $0.000                                        
  Tokens:    0                                             
```

**Why this is better:**
- Agent abbreviations (`R`, `SA`, `TR`) fit in 2-3 chars
- Status icon + cost visible for every agent at a glance
- Overview collapses to 3 lines instead of 8
- Activity entries truncate intelligently (verb + filename, not full path)

#### Implementation:

Add a method to `AgentTreeModel`:

```go
func (m AgentTreeModel) ViewCompact() string {
    // Render icon + 2-char abbreviation + cost
    // for each agent in the tree
}
```

In `renderRightPanel`, check `dims.rightWidth`:

```go
if dims.rightWidth < 30 {
    treeView = m.agentTree.ViewCompact()
    detailView = m.agentDetail.ViewCompact()
} else {
    treeView = m.agentTree.View()
    detailView = m.agentDetail.View()
}
```

**Files to modify:**
- `internal/tui/components/agents/tree.go` — add `ViewCompact()`
- `internal/tui/components/agents/detail.go` — add `ViewCompact()`
- `internal/tui/model/layout.go` — width-conditional rendering in `renderRightPanel`
- `internal/tui/model/interfaces.go` — add `ViewCompact()` to widget interfaces

#### Standard Text Mode (rightWidth 30-45)

```
● router              Running
  ● staff-architect    $1.98
  ✕ team-run           fail
  ● team-run           $0.12
  ◻ team-run           wait
```

Two-column layout: name left-aligned (truncated at available space - 8), status/cost right-aligned.

#### Full Detail Mode (rightWidth > 45)

Current layout works well at this width. No changes needed.

### 3.3 Recommendation 1b: Relative Paths in Activity Entries

**Current:**
```
✓  Bash  git add .claude/agents/mozart/
mozart.md && git commit -m '$(cat
<<'EOF' fix(agen...
```

**Proposed:**
```
✓ Bash  git add .claude/agents/mozart/mozart.md
✓ Bash  git commit -m 'fix(agents)...'
```

**Why:** Absolute paths `/home/doktersmol/Documents/GOgent-Fortress/` consume 50+ characters of a 22-character panel. Strip to project-relative paths. If the activity is a compound command (`&&`), split into separate entries.

**Implementation:**

In `cli/activity.go` or wherever `ExtractToolActivity` produces the target string, strip the project root prefix:

```go
func stripProjectRoot(path string) string {
    cwd, err := os.Getwd()
    if err != nil {
        return path
    }
    rel, err := filepath.Rel(cwd, path)
    if err != nil {
        return path
    }
    return rel
}
```

**Files to modify:**
- `internal/tui/cli/activity.go` — strip project root from extracted paths
- `internal/tui/components/agents/detail.go` — truncate activity.Target display

### 3.4 Recommendation 1c: Collapsible Overview to One-Line Summary

**Current (8 lines when expanded):**
```
▼ Overview
  Status:    Running
  Type:      router
  Model:     claude-opus-4-6[1m]
  Tier:      opus
  Duration:  162m 47s
  Cost:      $0.000
  Tokens:    0
```

**Proposed when unfocused (1 line):**
```
▸ router · opus · Running · $0.00 · 162m
```

**Proposed when focused (expand on Enter):**
```
▾ Overview
  Status:    Running
  Type:      router
  Model:     claude-opus-4-6[1m]
  Tier:      opus
  Duration:  162m 47s
  Cost:      $0.000
  Tokens:    0
```

**Why:** The Overview section is mostly static metadata. It changes only when the agent status transitions. Showing 8 lines of static data in a 22-column panel means the dynamic content (Activity feed) gets pushed below the viewport. The one-liner preserves the information while reclaiming 7 rows.

**UX reference:** Lazygit's file panel shows a one-line summary per file; expanding shows the diff. The same progressive disclosure pattern applies here.

**Implementation:**

In `detail.go`, modify the Overview `DetailSection`:

```go
{
    Title:    "Overview",
    Expanded: true,
    render: func(a *state.Agent, w int) string {
        if !m.focused {
            return renderOverviewCompact(a, w) // single line
        }
        return renderOverview(a, w) // full detail
    },
}
```

**Files to modify:**
- `internal/tui/components/agents/detail.go` — add `renderOverviewCompact()`

### 3.5 Recommendation 1d: Focus-Driven Drawer/Content Split

**Current:** Drawers always get 40% of `contentHeight` when any has content.

**Proposed:** The allocation shifts based on focus:

| Focus State | Agent Content | Drawer Area |
|-------------|--------------|-------------|
| Focus on agents | 70% | 30% (tabs visible, content compressed) |
| Focus on drawer | 30% | 70% (full content area) |
| No focus (Claude panel) | 55% | 45% (balanced default) |

**Why:** The Bubbletea `split-editors` example demonstrates this pattern. The focused pane gets more space. This is especially important in the Standard tier where every row counts.

**UX reference:** Every modern IDE (VS Code, JetBrains) makes the focused panel grow and unfocused panels shrink. Terminal TUIs like lazygit do the same with their panel splits.

**Implementation:**

Modify `computeDrawerLayout` to accept focus state:

```go
func (m AppModel) computeDrawerLayout(dims layoutDims) (height, width int) {
    // ...
    focusOnDrawer := m.focus == FocusOptionsDrawer || 
                     m.focus == FocusPlanDrawer || 
                     m.focus == FocusTeamsDrawer
    
    var drawerPct int
    if focusOnDrawer {
        drawerPct = 70
    } else if m.focus == FocusAgents {
        drawerPct = 30
    } else {
        drawerPct = 45
    }
    // ...
}
```

**Files to modify:**
- `internal/tui/model/layout.go` — focus-aware `computeDrawerLayout`

---

## 4. Area 2: Agent Tree Legibility & Hierarchy

### 4.1 Problem Statement

The agent tree is the TUI's core differentiator. At narrow widths, Unicode box-drawing characters (`├─*`) consume 4-5 characters per row — 20% of a 22-column panel — purely for decoration. Agent names are truncated mid-word. Status is only conveyed by a small icon that requires close reading.

### 4.2 Recommendation 2a: Two-Column Tree Layout

**Current:**
```
> router: Router
  [Bash: git status --short]
  ├─* staff-archite…
  [Grep: mcp\.RegisterAll|tools
  \.RegisterA…]
  ├─* team-run: te…
  ├─* team-run: te…
```

Tool context (`[Bash: ...]`, `[Grep: ...]`) is interleaved with the tree structure, making it hard to distinguish agents from activities.

**Proposed:**
```
● router ············· Running
  ● staff-architect ·· $1.98
  ✕ team-run ········· failed
  ● team-run ········· $0.12
  ● team-run ········· $0.08
  ◻ team-run ········· waiting
```

**Design principles at work:**

1. **No box-drawing characters.** Indentation (2 spaces) conveys hierarchy. This saves 3-4 characters per row.

2. **Dot leaders.** The dots between name and status work like a table of contents — they guide the eye from left to right across a variable-width gap. This is a centuries-old typographic technique.

3. **Right-aligned value.** Status/cost right-aligned creates a scannable column. Your eye can read down the right edge to check all statuses in <1 second.

4. **Tool context removed from tree.** Active tool calls belong in the detail panel's Activity section, not inline with the tree. The tree should be a pure structural overview.

**Narrow variant (< 30 cols):**
```
● R ···· Running
  ● SA · $1.98
  ✕ TR · failed
  ● TR · $0.12
```

**Implementation:**

Replace the current tree rendering in `tree.go`:

```go
func (m AgentTreeModel) renderNode(node state.TreeNode, depth int, w int) string {
    indent := strings.Repeat("  ", depth)
    icon := statusIcon(node.Agent.Status)
    iconStr := statusStyle(node.Agent.Status).Render(string(icon))
    
    name := node.Agent.Description
    if len(name) > w-depth*2-12 {
        name = name[:w-depth*2-12]
    }
    
    value := formatValue(node.Agent) // "$1.98" or "Running" or "failed"
    
    // Dot leaders fill the gap
    nameWidth := lipgloss.Width(name)
    valueWidth := lipgloss.Width(value)
    dotsNeeded := w - depth*2 - 2 - nameWidth - valueWidth - 1
    if dotsNeeded < 1 {
        dotsNeeded = 1
    }
    dots := config.StyleMuted.Render(strings.Repeat("·", dotsNeeded))
    
    return fmt.Sprintf("%s%s %s %s %s", indent, iconStr, name, dots, value)
}
```

**Files to modify:**
- `internal/tui/components/agents/tree.go` — rewrite `renderNode` and `View()`

### 4.3 Recommendation 2b: Full-Row Color by Status

**Current:** Only the status icon gets color. The agent name is always the default green.

**Proposed:** The entire row gets a subtle color tint based on status:

```
● router ············· Running    ← dim green text (running)
  ● staff-architect ·· Complete   ← bright green text (complete)
  ✕ team-run ········· failed     ← red text (failed)
  ● team-run ········· $0.12      ← dim green text (running)
  ◻ team-run ········· waiting    ← grey/muted text (pending)
```

**Why:** Color is processed pre-attentively — the brain registers color bands before reading text. A red row in a sea of green immediately draws attention to the failure without requiring the user to scan individual icons. Spotify-tui uses this for the currently-playing track (bright highlight vs dim list items).

**Implementation:**

In `tree.go`, wrap the entire row in the status style:

```go
rowStyle := statusRowStyle(node.Agent.Status)
return rowStyle.Render(fmt.Sprintf("%s%s %s %s %s", indent, icon, name, dots, value))
```

Where `statusRowStyle` returns:
- Running: `lipgloss.NewStyle().Foreground(config.ColorSuccess)` (dim green)
- Complete: `lipgloss.NewStyle().Foreground(config.ColorSuccess).Bold(true)`
- Failed: `lipgloss.NewStyle().Foreground(config.ColorError)`
- Pending: `lipgloss.NewStyle().Foreground(config.ColorMuted)`
- Killed: `lipgloss.NewStyle().Foreground(config.ColorWarning).Strikethrough(true)`

**Files to modify:**
- `internal/tui/components/agents/tree.go` — add `statusRowStyle()`, apply in render

### 4.4 Recommendation 2c: Inline Cost per Agent

**Current:** Cost is only visible in the Overview section of the detail panel. To check the cost of 5 agents, you must select each one.

**Proposed:** Show cost right-aligned in the tree (replaces status text when cost > 0):

```
● router ············· $0.00
  ● staff-architect ·· $1.98     ← cost visible at a glance
  ● team-run ········· $0.12
  ● team-run ········· $0.08
  ◻ team-run ········· waiting   ← no cost yet, show status
```

**Why:** For a multi-agent system that burns real money, cost-per-agent is the single most actionable piece of monitoring data. "Which agent is expensive?" should be answerable by glancing at the tree, not clicking through each agent.

**UX reference:** gh-dash shows PR review status inline with each PR row rather than requiring click-through.

**Implementation:**

```go
func formatValue(agent *state.Agent) string {
    if agent.Cost > 0 {
        return fmt.Sprintf("$%.2f", agent.Cost)
    }
    switch agent.Status {
    case state.StatusRunning:
        return "Running"
    case state.StatusComplete:
        return "Done"
    case state.StatusError:
        return "failed"
    case state.StatusPending:
        return "waiting"
    default:
        return string(agent.Status)
    }
}
```

**Files to modify:**
- `internal/tui/components/agents/tree.go` — add `formatValue()`, use in `renderNode`

### 4.5 Recommendation 2d: Tree Density Toggle

Add `alt+shift+e` (or similar) to cycle through 3 density levels:

**Compact:**
```
● R  ● SA  ✕ TR  ● TR  ● TR  ◻ TR
```
Single row, all agents as icon+abbreviation. Maximum density.

**Standard (default):**
```
● router ············· $0.00
  ● staff-architect ·· $1.98
  ✕ team-run ········· failed
```
The two-column layout from 2a.

**Verbose:**
```
● router               Running    opus   162m
  ● staff-architect    $1.98      opus   9m
  ✕ team-run           failed     sonnet 2m
  ● team-run           $0.12      sonnet 45s
```
Full metadata: status, tier, duration.

**Files to modify:**
- `internal/tui/components/agents/tree.go` — add density state, 3 render modes
- `internal/tui/model/key_handlers.go` — wire the keyboard shortcut
- `internal/tui/config/keys.go` — add key binding

### 4.6 Recommendation 2e: Pulse Animation on Active Agent

When an agent is actively streaming (tool call in progress), pulse its icon between bright and dim on a 500ms tick cycle:

```
Frame 1: ● staff-architect ·· $1.98    (bright green)
Frame 2: ● staff-architect ·· $1.98    (dim green)
```

**Why:** In a tree with 6+ agents, finding "which one is doing something right now" requires reading status text. A pulsing icon draws the eye unconsciously. The Bubbletea `spinner` example demonstrates this tick-based animation pattern.

**Implementation:**

Add a `streamingAgentIDs` set to the tree model. On each 500ms tick, toggle a `pulsePhase` bool. In the render, agents whose ID is in the streaming set get bright/dim styling based on phase.

```go
type AgentTreeModel struct {
    // ...
    streamingIDs map[string]bool
    pulsePhase   bool
}
```

**Files to modify:**
- `internal/tui/components/agents/tree.go` — add pulse state, tick handling
- `internal/tui/model/cli_event_handlers.go` — update `streamingIDs` on tool_use/tool_result

---

## 5. Area 3: Conversation Panel & Chat UX

### 5.1 Problem Statement

Users spend 80%+ of their time in the conversation panel. Currently, turns flow into each other with only a `You:` / `Claude:` label change. In long sessions (162+ minutes as seen in screenshots), this becomes a wall of same-colored green text that's hard to navigate.

### 5.2 Recommendation 3a: Horizontal Rule Between Turns

**Current:**
```
Claude:

  Done. abf4a3ff on routing-restructure.

You:
all the files to logically commit - please do :)

Claude:

  Let me look at the diffs of the modified files to understand
  what changed, then group them logically.
```

**Proposed:**
```
Claude:

  Done. abf4a3ff on routing-restructure.

────────────────────────────────────────────────────────────

You:
all the files to logically commit - please do :)

────────────────────────────────────────────────────────────

Claude:

  Let me look at the diffs of the modified files to understand
  what changed, then group them logically.
```

**Why:** A thin horizontal rule creates a visual "paragraph break" between turns. The eye can jump from rule to rule to find turn boundaries without reading labels. This is the single lowest-effort, highest-impact readability improvement.

**UX reference:** Every chat application (Slack, Discord, iMessage) uses visual separators between messages. Terminal apps like lazygit use thin dividers between conceptual groups.

**Implementation:**

In the Claude panel's message rendering, insert a separator between turns:

```go
if i > 0 && messages[i].Role != messages[i-1].Role {
    separator := config.StyleMuted.Render(strings.Repeat("─", panelWidth))
    parts = append(parts, separator)
}
```

**Files to modify:**
- `internal/tui/components/claude/panel.go` — add separator in render loop

### 5.3 Recommendation 3b: User vs Assistant Color Differentiation

**Current:** Both `You:` and `Claude:` messages are rendered in the same green.

**Proposed:**
```
You:                                    ← cyan/white text
all the files to logically commit

Claude:                                 ← green text (unchanged)
Let me look at the diffs...
```

**Why:** Color differentiation works even in peripheral vision. When scrolling through a long conversation, the alternating color pattern lets you identify who said what without reading the label. Gh-dash uses distinct colors for PR authors vs reviewers.

**Specific colors:**
- `You:` messages: `lipgloss.Color("6")` (cyan) or `lipgloss.Color("7")` (white)
- `Claude:` messages: keep current green
- System messages: keep current muted/grey

**Files to modify:**
- `internal/tui/components/claude/panel.go` — role-based text styling

### 5.4 Recommendation 3c: Inline Streaming Tool Indicator

**Current during streaming:** Status bar shows `streaming` with spinner. The conversation shows nothing until the result appears.

**Proposed during streaming:**
```
Claude:

  [ROUTING] → staff-architect-critical-review

  ⠋ Bash  git diff --staged --stat
```

The last line updates in real-time as tool calls stream. When the tool completes, it collapses to a success/fail one-liner.

**After completion:**
```
  ✓ Bash  git diff --staged --stat (0.3s)
```

**Why:** The conversation is where the user is looking. The status bar requires a deliberate eye movement to check. An inline indicator keeps the user informed without breaking their reading flow.

**Files to modify:**
- `internal/tui/components/claude/panel.go` — inline tool block rendering
- `internal/tui/model/cli_event_handlers.go` — forward streaming tool_use to panel

### 5.5 Recommendation 3d: Collapsible Tool-Use Blocks

**Current:** Tool blocks are either fully expanded or hidden.

**Proposed collapsed (default):**
```
  ✓ Bash  git add .claude/agents/mozart/mozart.md && git commit...
```

**Proposed expanded (toggle with Enter or alt+e):**
```
  ✓ Bash  git add .claude/agents/mozart/mozart.md && git commit -m '$(cat <<'EOF'
  fix(agents): update mozart to use MCP tools instead of Task/AskUserQuestion
  
  Mozart was still referencing Task, TaskList, TaskGet...
  EOF
  )'
  
  [routing-restructure cdeaad96] fix(agents): update mozart...
  1 file changed, 51 insertions(+), 73 deletions(-)
```

**Why:** Progressive disclosure. Most tool calls succeed and the user doesn't need to see the full output. Showing it collapsed saves vertical space and reduces noise. Expanding on demand gives full detail when needed.

**UX reference:** Lazygit's file list shows file names; expanding shows the diff. Same principle.

**Files to modify:**
- `internal/tui/components/claude/panel.go` — add `ToolBlock` component with collapsed/expanded state
- `internal/tui/model/key_handlers.go` — wire expansion toggle

### 5.6 Recommendation 3e: Timestamp Gutter

**Proposed (optional, toggle in settings):**
```
       Claude:
         Done. abf4a3ff on routing-restructure.
  2m   ──────────────────────────────────────
       You:
         all the files to logically commit
  2m   ──────────────────────────────────────
       Claude:
         Let me look at the diffs...
  5m   ──────────────────────────────────────
       You:
         yeah looks good, commit it
```

**Why:** In long sessions (162+ minutes), users often want to find "that thing Claude said about X." Relative timestamps help orient: "it was about 30 minutes ago" → scan for the `30m` timestamp.

**Implementation:** Add a narrow (5-char) gutter to the left of the conversation viewport. Render relative timestamps at turn boundaries only (not every line). Updates every 60s via an existing tick.

**Files to modify:**
- `internal/tui/components/claude/panel.go` — add gutter rendering
- `internal/tui/components/claude/panel.go` — store timestamps per message

---

## 6. Area 4: Status Line & Progress Feedback

### 6.1 Problem Statement

The status line packs 12+ data fields into 2 rows of text. At a glance, it reads as a wall of text rather than a dashboard. The two most critical signals for a multi-agent system — **cost** and **context window usage** — are presented as text that requires active reading.

### 6.2 Recommendation 4a: Context Window Progress Bar

**Current:**
```
[M] [claude-opus-4-6[1m]] [acceptEdits] ■ anthropic
```
Context percentage exists in the data model but is not prominently displayed.

**Proposed:**
```
Ctx [████████░░░░░░░░░░░░] 42%  $1.98  opus  162m  agents:2/3
```

The progress bar uses Unicode block characters:
- `█` (U+2588) for filled
- `░` (U+2591) for empty

Color thresholds (matching `budgetColor` in teams/health.go):
- Green (`config.StyleSuccess`): < 70%
- Yellow (`config.StyleWarning`): 70-90%
- Red (`config.StyleError`): > 90%

**Visual examples at different fill levels:**

```
 12%  [██░░░░░░░░░░░░░░░░░░]   ← green
 45%  [█████████░░░░░░░░░░░]   ← green
 72%  [██████████████░░░░░░]   ← YELLOW — attention
 91%  [██████████████████░░]   ← RED — danger
 99%  [███████████████████░]   ← RED — critical
```

**Why:** A progress bar is processed pre-attentively — the brain reads the fill level before conscious text parsing. "Am I running out of context?" becomes a sub-second glance instead of reading `ContextPercent: 72.3%`. This is a safety feature for long sessions.

**UX reference:** Spotify-tui's song progress bar. The Bubbletea `progress-bar` example. Your own `budgetColor` bar in teams health.

**Implementation:**

```go
func renderContextBar(pct float64, width int) string {
    barWidth := width
    if barWidth > 20 {
        barWidth = 20
    }
    filled := int(float64(barWidth) * pct / 100)
    empty := barWidth - filled
    
    style := config.StyleSuccess
    if pct >= 90 {
        style = config.StyleError
    } else if pct >= 70 {
        style = config.StyleWarning
    }
    
    bar := style.Render(strings.Repeat("█", filled)) +
           config.StyleMuted.Render(strings.Repeat("░", empty))
    
    return fmt.Sprintf("Ctx [%s] %d%%", bar, int(pct))
}
```

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — add `renderContextBar`, integrate into Row 1

### 6.3 Recommendation 4b: Prominent Cost Display

**Current:** Cost is somewhere in the status line text, same font weight as everything else.

**Proposed:** Cost is the first element in Row 1, bold/bright:

```
$1.98  Ctx [████████░░░░] 42%  opus  162m  agents:2/3
```

When cost exceeds thresholds, color changes:
- `< $1.00`: green
- `$1.00 - $5.00`: yellow
- `> $5.00`: red

**Why:** This system can burn $50 in a braintrust session. Cost should be the most visible element — it's a safety signal. "How much have I spent?" should be answerable without conscious effort.

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — reorder Row 1, add cost coloring

### 6.4 Recommendation 4c: Agent Count Sparkline

**Current:** `agents:3`

**Proposed:** `agents: 2/3 ●●◻`

Where:
- `2/3` = running count / total count
- `●` = running (green)
- `◻` = pending (grey)
- `✕` = failed (red, if any)

**Example states:**
```
agents: 3/3 ●●●          ← all running
agents: 2/4 ●●◻◻         ← 2 running, 2 pending
agents: 1/3 ●✕◻          ← 1 running, 1 failed, 1 pending
agents: 0/3 ✓✓✓          ← all complete
```

**Why:** The dots are a miniature version of the agent tree. At a glance, you know "how many agents, what state." Without opening the right panel.

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — replace agent count with sparkline

### 6.5 Recommendation 4d: Context-Adaptive Status Line Density

**At Standard width (80-119 cols):** Compress to 1 row with only critical fields:

```
$1.98  Ctx [████████░░░░] 42%  opus  agents:2/3  162m
```

Dropped from 1-row: permission mode, auth email, git branch, provider name, CWD. These are viewable in the Settings tab and rarely change mid-session.

**At Wide width (120+ cols):** Full 2-row layout with all fields:

```
Row 1: $1.98  Ctx [████████████░░░░░░░░] 42%  opus  agents:2/3 ●●◻  ⏱ 162m 47s
Row 2: anthropic  acceptEdits  routing-restructure  admin@exactmass.org  ~/Documents/GOgent-Fortress
```

**Why:** At narrow widths, every row of status bar is a row stolen from conversation content. The 2→1 row compression reclaims a content row where it matters most.

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — tier-aware row count
- `internal/tui/model/layout.go` — adjust `statusLineHeight` based on tier

### 6.6 Recommendation 4e: Cost Flash-on-Change

When `SessionCost` increases (after a `ResultEvent`), flash the cost field bright white for 500ms:

```
Frame 0 (normal):     $1.98
Frame 1-3 (flash):    $2.15     ← bright white, 500ms
Frame 4+ (normal):    $2.15     ← back to green
```

**Why:** Creates subconscious cost awareness without active monitoring. You notice the flash in your peripheral vision. The `TabFlashMsg` pattern already implements this exact mechanic for tabs.

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — add `costFlashUntil time.Time`, apply flash style

---

## 7. Area 5: Team & Workflow Monitoring

### 7.1 Problem Statement

Teams are the most complex and expensive operations in the system. A braintrust session with 3 opus agents can cost $5+ in minutes. But team state is only visible in the drawer — which starts minimized and requires active expansion. There's no persistent indicator of team health.

### 7.2 Recommendation 5a: Status Line Team Indicator

**When no team is running:**
```
$1.98  Ctx [████████░░░░] 42%  opus  agents:2/3  162m
```
(No team indicator — no wasted space)

**When a team is running:**
```
$1.98  Ctx [████████░░░░] 42%  opus  agents:2/3  162m  team:review ●● 2/4 $2.30
```

The team indicator shows:
- Team name (truncated)
- Member dots (colored by status, same as agent sparkline)
- Wave progress (`2/4` = wave 2 of 4)
- Team cost

**Why:** This makes team state always-visible without requiring drawer expansion. The drawer becomes the detail view for drill-down.

**Implementation:**

Add team summary fields to `StatusLineModel`:

```go
// Team monitoring (populated by poll tick when a team is running).
TeamName       string
TeamMemberDots string  // pre-rendered "●●◻" string
TeamWaveProgress string // "2/4"
TeamCost       float64
```

Populate from the teams health widget on each poll tick in `app.go`.

**Files to modify:**
- `internal/tui/components/statusline/statusline.go` — add team fields, render in Row 1
- `internal/tui/model/app.go` — populate team fields from teamsHealth in poll tick handler

### 7.3 Recommendation 5b: Action-Hinted Team Toasts

**Current toast:** `team_run started (pid 12345): /path/to/team`

**Proposed toasts:**

```
Team launch:      "review team launched — /team-status to monitor"
Wave complete:    "Wave 1 complete (2/4 members) — 3 files changed"
Member failure:   "scout failed: timeout after 300s — /team-status for details"
Team complete:    "review team done — $2.30 — /team-result to view findings"
Budget warning:   "review team at 80% budget ($4.00/$5.00)"
```

**Why:** Every toast should answer "what happened?" AND "what do I do next?". Including the slash command makes the toast actionable — the user can immediately type the suggested command.

**Files to modify:**
- `internal/tui/mcp/tools.go` — enhance team_run toasts
- `cmd/gogent-team-run/spawn.go` — emit lifecycle toasts via UDS

### 7.4 Recommendation 5c: Auto-Switch on Team Completion

When a background team completes:
1. Flash the Teams tab (`TabFlashMsg` — already implemented)
2. Show a toast with `/team-result` hint
3. If the conversation panel is idle (not streaming), optionally auto-switch left panel to Team Config tab to show results

**Why:** Completed team results are time-sensitive — the user launched the team to get an answer. Bringing attention to the result automatically reduces the feedback loop.

**Files to modify:**
- `internal/tui/model/app.go` — poll tick handler: detect completion, emit flash + toast
- `internal/tui/model/ui_event_handlers.go` — optional auto-switch logic

### 7.5 Recommendation 5d: Team Timeline Progress View

**Current teams health view:**
```
Team: review-20260410  running  PID 12345  4m 23s
Budget: [████████░░░░░░░░░░░░] 23% ($0.46 / $2.00)

── Wave 1 (current)
  ● backend-rev  running   PID 54321  2 stalls  3m ago    12KB
  ● standards    running   PID 54322  0 stalls  1m ago     8KB

── Wave 2
  ◻ architect    pending   waiting for Wave 1
```

**Proposed timeline view:**
```
Team: review  Running  4m 23s  $0.46/$2.00
Budget [████████░░░░░░░░░░░░] 23%

Wave 1 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━░░░░░░ 72%
  ● backend-rev  ████████████████░░  3m · $0.22
  ● standards    ████████████░░░░░░  2m · $0.18

Wave 2 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ Pending
  ◻ architect    ░░░░░░░░░░░░░░░░░░  waiting
```

**Why:** Horizontal bars communicate progress intuitively — you see how far along each member is without reading numbers. The wave-level bar aggregates member progress. The Bubbletea `progress-animated` example shows the rendering technique.

**Color coding:**
- Running member bars: green
- Complete member bars: bright green (full)
- Failed member bars: red (partial)
- Pending member bars: grey
- Wave aggregate: derived from member states

**Implementation:**

Add a `renderTimelineView` to teams health model that renders per-member progress bars based on elapsed time vs timeout:

```go
func (m *TeamsHealthModel) renderMemberBar(member Member, width int) string {
    if member.Status == "pending" {
        return config.StyleMuted.Render(strings.Repeat("░", width))
    }
    // Estimate progress from elapsed time vs timeout
    elapsed := time.Since(parseTime(member.StartedAt))
    timeout := time.Duration(member.TimeoutMs) * time.Millisecond
    pct := float64(elapsed) / float64(timeout)
    if pct > 1 { pct = 1 }
    
    filled := int(float64(width) * pct)
    empty := width - filled
    
    style := memberBarStyle(member.Status)
    return style.Render(strings.Repeat("━", filled)) +
           config.StyleMuted.Render(strings.Repeat("░", empty))
}
```

**Files to modify:**
- `internal/tui/components/teams/health.go` — add timeline rendering mode

### 7.6 Recommendation 5e: Team Diff Summary on Completion

When a team completes, add a summary line to the toast and team detail:

```
review team done — 3 files modified, +45 -12 lines — $2.30
```

**Implementation:** Parse stdout files for tool_result blocks with `structuredPatch` fields, aggregate file counts and line deltas.

**Files to modify:**
- `internal/tui/components/teams/detail.go` — add completion summary
- Team completion toast logic

---

## 8. Priority Matrix

### P0 — Do First (transforms daily experience)

| ID | Recommendation | Impact | Effort | Files |
|----|---------------|--------|--------|-------|
| 1a | Adaptive right-panel (icon rail < 30 cols) | Critical | Medium | tree.go, detail.go, layout.go, interfaces.go |
| 1b | Relative paths in activity entries | High | Low | activity.go, detail.go |
| 4a | Context window progress bar | High | Low | statusline.go |
| 4b | Prominent cost display (first in row, colored) | High | Low | statusline.go |

**Estimated combined effort:** 2-3 sessions

### P1 — Do Next (major readability gains)

| ID | Recommendation | Impact | Effort | Files |
|----|---------------|--------|--------|-------|
| 2a | Two-column tree layout (dots, right-aligned) | High | Medium | tree.go |
| 2b | Full-row color by agent status | High | Low | tree.go |
| 3a | Horizontal rule between turns | High | Low | panel.go |
| 3b | User/assistant color differentiation | Medium | Low | panel.go |
| 5a | Status line team indicator | High | Medium | statusline.go, app.go |

**Estimated combined effort:** 2-3 sessions

### P2 — Polish (meaningful improvements)

| ID | Recommendation | Impact | Effort | Files |
|----|---------------|--------|--------|-------|
| 1c | Collapsible Overview to one-liner | Medium | Medium | detail.go |
| 2c | Inline cost per agent in tree | Medium | Low | tree.go |
| 3d | Collapsible tool-use blocks | Medium | Medium | panel.go |
| 4c | Agent count sparkline dots | Medium | Low | statusline.go |
| 5b | Action-hinted team toasts | Medium | Low | tools.go, spawn.go |
| 5d | Team timeline progress bars | Medium | High | health.go |

**Estimated combined effort:** 3-4 sessions

### P3 — Nice-to-have (refinements)

| ID | Recommendation | Impact | Effort | Files |
|----|---------------|--------|--------|-------|
| 1d | Focus-driven drawer/content split | Medium | High | layout.go |
| 2d | Tree density toggle (compact/standard/verbose) | Low | Low | tree.go, key_handlers.go |
| 2e | Pulse animation on active agent | Low | Low | tree.go |
| 3c | Inline streaming tool indicator | Medium | Medium | panel.go |
| 3e | Timestamp gutter | Low | Medium | panel.go |
| 4d | Adaptive 1-row status line at narrow widths | Medium | Medium | statusline.go, layout.go |
| 4e | Cost flash-on-change | Low | Low | statusline.go |
| 5c | Auto-switch on team completion | Low | Low | app.go |
| 5e | Team diff summary on completion | Low | Medium | detail.go |

---

## 9. Implementation Notes

### 9.1 Branching Strategy

Create from `routing-restructure` after PR merge:
```
git checkout -b ux-redesign-p0
```

Each priority tier should be its own branch/PR:
- `ux-redesign-p0` — 4 items, 2-3 sessions
- `ux-redesign-p1` — 5 items, 2-3 sessions
- `ux-redesign-p2` — 6 items, 3-4 sessions
- `ux-redesign-p3` — 9 items, as time permits

### 9.2 Testing Strategy

Each visual change should include:
1. Unit test for the rendering function (string assertions)
2. Width boundary tests (verify behavior at 22, 30, 45, 80 cols)
3. Screenshot comparison if possible

### 9.3 Backward Compatibility

All visual changes should be theme-aware. Use `config.Style*` accessors rather than hardcoded colors. The high-contrast theme (TUI-051) must remain WCAG compliant after all changes.

### 9.4 Reference Materials

| Reference | Pattern to Study |
|-----------|-----------------|
| lazygit | Panel splits, progressive disclosure, focus model |
| gh-dash | Inline status, dot leaders, YAML customization |
| spotify-tui | Progress bars, real-time status, color semantics |
| Bubbletea `split-editors` | Focus-driven pane sizing |
| Bubbletea `progress-bar` | Unicode progress rendering |
| Bubbletea `realtime` | Channel-based live updates |
| Bubbletea `table-resize` | Responsive column layout |

### 9.5 Key Lipgloss Techniques

**Dot leaders:**
```go
dotsNeeded := width - lipgloss.Width(left) - lipgloss.Width(right)
dots := config.StyleMuted.Render(strings.Repeat("·", max(1, dotsNeeded)))
row := left + " " + dots + " " + right
```

**Progress bar:**
```go
filled := int(float64(barWidth) * pct)
bar := style.Render(strings.Repeat("█", filled)) +
       config.StyleMuted.Render(strings.Repeat("░", barWidth-filled))
```

**Right-alignment in fixed-width panel:**
```go
padding := width - lipgloss.Width(left) - lipgloss.Width(right)
row := left + strings.Repeat(" ", max(1, padding)) + right
```

**Conditional rendering by width:**
```go
if panelWidth < 30 {
    return m.ViewCompact()
}
return m.View()
```
