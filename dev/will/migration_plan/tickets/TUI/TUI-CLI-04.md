# TUI-CLI-04: Main Layout Integration

> **Estimated Hours:** 2.0
> **Priority:** P1 - Integration
> **Dependencies:** TUI-CLI-03, TUI-AGENT-02
> **Phase:** 4 - Integration

---

## Description

Integrate Claude panel with agent tree in 70/30 split layout.

---

## Layout

```
+------------------------------------+--------------------+
|        Claude Interface            |   Agent Tree       |
|            (70%)                   |     (30%)          |
|                                    |                    |
|  +------------------------------+  | > terminal         |
|  | Conversation viewport        |  |   +-- orchestrator |
|  |                              |  |   +-- go-tui [hour]|
|  |                              |  |                    |
|  +------------------------------+  +--------------------+
|  +------------------------------+  | Selected: go-tui   |
|  | > Input...            [Enter]|  | Tier: sonnet       |
|  +------------------------------+  | Duration: 2.3s...  |
+------------------------------------+--------------------+
```

---

## Tasks

### 1. Define Layout Constants and Model

**File:** `internal/tui/main/layout.go`

```go
package main

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/agents"
    "github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/claude"
)

const (
    LeftPanelRatio  = 0.70
    RightPanelRatio = 0.30
    MinLeftWidth    = 40
    MinRightWidth   = 20
)

type FocusedPanel int

const (
    FocusLeft FocusedPanel = iota
    FocusRight
)

type Model struct {
    claudePanel claude.PanelModel
    agentTree   agents.TreeModel
    agentDetail agents.DetailModel
    width       int
    height      int
    focused     FocusedPanel
}

func NewModel(claudePanel claude.PanelModel, agentTree agents.TreeModel) Model {
    return Model{
        claudePanel: claudePanel,
        agentTree:   agentTree,
        agentDetail: agents.NewDetailModel(),
        focused:     FocusLeft,
    }
}
```

### 2. Implement Layout Calculation

```go
func (m Model) calculateLayout() (leftWidth, rightWidth int) {
    available := m.width - 1 // Border

    leftWidth = int(float64(available) * LeftPanelRatio)
    rightWidth = available - leftWidth

    // Enforce minimums
    if leftWidth < MinLeftWidth {
        leftWidth = MinLeftWidth
        rightWidth = available - leftWidth
    }
    if rightWidth < MinRightWidth {
        rightWidth = MinRightWidth
        leftWidth = available - rightWidth
    }

    return leftWidth, rightWidth
}

func (m *Model) updateSizes() {
    leftWidth, rightWidth := m.calculateLayout()

    // Update child components
    m.claudePanel.SetSize(leftWidth, m.height-2) // Reserve for borders
    m.agentTree.SetSize(rightWidth, m.height/2)
    m.agentDetail.SetSize(rightWidth, m.height/2)
}
```

### 3. Implement Update Handler

```go
func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.updateSizes()

    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            // Toggle focus
            if m.focused == FocusLeft {
                m.focused = FocusRight
                m.claudePanel.SetFocused(false)
                m.agentTree.SetFocused(true)
            } else {
                m.focused = FocusLeft
                m.claudePanel.SetFocused(true)
                m.agentTree.SetFocused(false)
            }
            return m, nil

        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }

    // Update focused panel
    if m.focused == FocusLeft {
        var cmd tea.Cmd
        m.claudePanel, cmd = m.claudePanel.Update(msg)
        cmds = append(cmds, cmd)
    } else {
        var cmd tea.Cmd
        m.agentTree, cmd = m.agentTree.Update(msg)
        cmds = append(cmds, cmd)

        // Update detail panel with selected agent
        m.agentDetail.SetAgent(m.agentTree.SelectedAgent())
    }

    return m, tea.Batch(cmds...)
}
```

### 4. Implement View Rendering

```go
func (m Model) View() string {
    leftWidth, rightWidth := m.calculateLayout()

    // Left panel (Claude interface)
    leftPanel := leftPanelStyle.
        Width(leftWidth).
        Height(m.height).
        Render(m.claudePanel.View())

    // Right panel (Agent tree + detail)
    treeView := m.agentTree.View()
    detailView := m.agentDetail.View()

    rightContent := lipgloss.JoinVertical(
        lipgloss.Left,
        treeView,
        detailView,
    )

    rightPanel := rightPanelStyle.
        Width(rightWidth).
        Height(m.height).
        Render(rightContent)

    // Add focus indicator
    if m.focused == FocusLeft {
        leftPanel = focusedStyle.Render(leftPanel)
    } else {
        rightPanel = focusedStyle.Render(rightPanel)
    }

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        leftPanel,
        rightPanel,
    )
}

// Styles
var (
    leftPanelStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        BorderRight(true)

    rightPanelStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder())

    focusedStyle = lipgloss.NewStyle().
        BorderForeground(lipgloss.Color("cyan"))
)
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/main/layout.go` | Main layout integration |
| `internal/tui/main/layout_test.go` | Layout calculation tests |

---

## Acceptance Criteria

- [ ] 70/30 split layout renders correctly
- [ ] Tab switches focus between panels
- [ ] Resize terminal updates layout proportionally
- [ ] Minimum widths enforced
- [ ] Focus indicator visible (border color change)
- [ ] Agent detail updates when tree selection changes

---

## Test Strategy

### Unit Tests
- Layout calculation at various widths
- Minimum width enforcement
- Focus toggle behavior

### Visual Tests
- Terminal resize handling
- Focus indicator visibility
- Panel proportion accuracy
