# TUI-AGENT-02: Agent Tree View

> **Estimated Hours:** 3.0
> **Priority:** P1 - Agent Tree
> **Dependencies:** TUI-AGENT-01, TUI-PERF-01
> **Phase:** 3 - Agent Tree

---

## Description

Bubble Tea component that renders the agent delegation tree with status indicators and selection.

---

## Layout

```
Agent Delegation Tree
---------------------
> terminal
  +-- orchestrator [check] 2.3s
  |  +-- python-pro [check] 1.1s
  +-- go-tui [hourglass] 4.2s...
  |    "Implement TUI panel"
  +-- haiku-scout [check] 0.3s
```

---

## Status Indicators

| Status | Indicator | Color |
|--------|-----------|-------|
| Spawning | hourglass | Yellow |
| Running (>5s) | refresh | Blue |
| Completed | check | Green |
| Failed | x | Red |

---

## Tasks

### 1. Create Tree Model

**File:** `internal/tui/agents/view.go`

```go
package agents

import (
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type TreeModel struct {
    tree     *AgentTree
    cursor   int
    expanded map[string]bool
    width    int
    height   int
    focused  bool
}

func NewTreeModel(tree *AgentTree) TreeModel {
    return TreeModel{
        tree:     tree,
        expanded: make(map[string]bool),
    }
}

func (m TreeModel) Init() tea.Cmd {
    return nil
}
```

### 2. Implement Update Handler

```go
func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
    if !m.focused {
        return m, nil
    }

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            agents := m.tree.GetAll()
            if m.cursor < len(agents)-1 {
                m.cursor++
            }
        case "enter", "space":
            agents := m.tree.GetAll()
            if m.cursor < len(agents) {
                id := agents[m.cursor].ID
                m.expanded[id] = !m.expanded[id]
            }
        }
    }

    return m, nil
}
```

### 3. Implement View Rendering

```go
func (m TreeModel) View() string {
    var b strings.Builder

    // Header
    b.WriteString(headerStyle.Render("Agent Delegation Tree"))
    b.WriteString("\n")
    b.WriteString(strings.Repeat("-", m.width-2))
    b.WriteString("\n")

    // Render tree
    agents := m.tree.GetAll()
    for i, agent := range agents {
        line := m.renderNode(agent, i == m.cursor)
        b.WriteString(line)
        b.WriteString("\n")
    }

    // Pending count
    pending := m.tree.GetPending()
    if len(pending) > 0 {
        b.WriteString(fmt.Sprintf("\n%d agent(s) running", len(pending)))
    }

    return b.String()
}
```

### 4. Implement Node Rendering

```go
func (m TreeModel) renderNode(node *AgentNode, selected bool) string {
    // Status indicator
    var indicator string
    var style lipgloss.Style

    switch node.Status {
    case StatusSpawning:
        indicator = "[hourglass]"
        style = spawningStyle
    case StatusRunning:
        indicator = "[refresh]"
        style = runningStyle
    case StatusCompleted:
        indicator = "[check]"
        style = completedStyle
    case StatusFailed:
        indicator = "[x]"
        style = failedStyle
    }

    // Duration
    var duration string
    if node.CompleteTime != nil {
        duration = fmt.Sprintf(" %.1fs", node.Duration.Seconds())
    } else if node.Status == StatusRunning || node.Status == StatusSpawning {
        elapsed := time.Since(node.SpawnTime)
        duration = fmt.Sprintf(" %.1fs...", elapsed.Seconds())
    }

    // Build line
    line := fmt.Sprintf("%s %s%s", indicator, node.AgentID, duration)

    if selected {
        line = selectedStyle.Render("> " + line)
    } else {
        line = "  " + style.Render(line)
    }

    // Task description (if expanded)
    if m.expanded[node.ID] && node.TaskDescription != "" {
        desc := truncate(node.TaskDescription, m.width-6)
        line += "\n" + descStyle.Render("    \""+desc+"\"")
    }

    return line
}
```

### 5. Implement Helpers and Styles

```go
func (m TreeModel) SelectedAgent() *AgentNode {
    agents := m.tree.GetAll()
    if m.cursor < len(agents) {
        return agents[m.cursor]
    }
    return nil
}

func (m *TreeModel) SetFocused(focused bool) {
    m.focused = focused
}

// Styles
var (
    headerStyle    = lipgloss.NewStyle().Bold(true)
    spawningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
    runningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
    completedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
    failedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
    selectedStyle  = lipgloss.NewStyle().Bold(true)
    descStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/agents/view.go` | Tree view Bubble Tea component |
| `internal/tui/agents/view_test.go` | View rendering tests |

---

## Acceptance Criteria

- [ ] Tree renders with proper indentation
- [ ] Status indicators display correctly
- [ ] Duration shows (completed) or elapsed (running)
- [ ] Up/down navigation works
- [ ] Enter expands/collapses task description
- [ ] Selected agent highlighted
- [ ] Colors adapt to status
- [ ] Real-time updates as agents spawn/complete

---

## Test Strategy

### Unit Tests
- Render empty tree
- Render tree with single agent
- Render tree with nested agents
- Navigation bounds checking
- Expand/collapse behavior

### Visual Tests
- Manual testing with different terminal sizes
- Color rendering verification
