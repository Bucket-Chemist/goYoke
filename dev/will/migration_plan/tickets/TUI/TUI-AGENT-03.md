# TUI-AGENT-03: Agent Detail Sidebar

> **Estimated Hours:** 1.5
> **Priority:** P2 - Enhancement
> **Dependencies:** TUI-AGENT-02
> **Phase:** 3 - Agent Tree

---

## Description

Detail panel showing full information about the selected agent.

---

## Layout

```
+------------------------+
| Selected: go-tui       |
| Tier: sonnet           |
| Status: Running [hour] |
| Duration: 4.2s...      |
+------------------------+
| Task Description:      |
| Implement TUI panel    |
| with agent delegation  |
| tree view...           |
+------------------------+
| [Enter] Expand         |
| [q] Query agent        |
+------------------------+
```

---

## Tasks

### 1. Create Detail Panel Component

**File:** `internal/tui/agents/detail.go`

```go
package agents

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss"
)

type DetailModel struct {
    agent  *AgentNode
    width  int
    height int
}

func NewDetailModel() DetailModel {
    return DetailModel{}
}

func (m *DetailModel) SetAgent(agent *AgentNode) {
    m.agent = agent
}

func (m *DetailModel) SetSize(width, height int) {
    m.width = width
    m.height = height
}
```

### 2. Implement View Rendering

```go
func (m DetailModel) View() string {
    if m.agent == nil {
        return emptyDetailStyle.Render("No agent selected")
    }

    var b strings.Builder

    // Header section
    b.WriteString(detailHeaderStyle.Render(fmt.Sprintf("Selected: %s", m.agent.AgentID)))
    b.WriteString("\n")
    b.WriteString(fmt.Sprintf("Tier: %s", m.agent.Tier))
    b.WriteString("\n")
    b.WriteString(fmt.Sprintf("Status: %s %s", m.agent.Status, m.statusIndicator()))
    b.WriteString("\n")
    b.WriteString(fmt.Sprintf("Duration: %s", m.durationString()))
    b.WriteString("\n")

    // Separator
    b.WriteString(strings.Repeat("-", m.width-2))
    b.WriteString("\n")

    // Task description section
    b.WriteString(detailLabelStyle.Render("Task Description:"))
    b.WriteString("\n")
    if m.agent.TaskDescription != "" {
        // Word wrap the description
        wrapped := wordWrap(m.agent.TaskDescription, m.width-4)
        b.WriteString(wrapped)
    } else {
        b.WriteString(emptyDetailStyle.Render("(no description)"))
    }
    b.WriteString("\n")

    // Separator
    b.WriteString(strings.Repeat("-", m.width-2))
    b.WriteString("\n")

    // Keyboard hints
    b.WriteString(hintStyle.Render("[Enter] Expand"))
    b.WriteString("\n")
    b.WriteString(hintStyle.Render("[q] Query agent"))

    return b.String()
}

func (m DetailModel) statusIndicator() string {
    switch m.agent.Status {
    case StatusSpawning:
        return "[hourglass]"
    case StatusRunning:
        return "[refresh]"
    case StatusCompleted:
        return "[check]"
    case StatusFailed:
        return "[x]"
    default:
        return ""
    }
}

func (m DetailModel) durationString() string {
    if m.agent.CompleteTime != nil {
        return fmt.Sprintf("%.1fs", m.agent.Duration.Seconds())
    }
    elapsed := time.Since(m.agent.SpawnTime)
    return fmt.Sprintf("%.1fs...", elapsed.Seconds())
}
```

### 3. Implement Helper Functions

```go
func wordWrap(s string, width int) string {
    if width <= 0 {
        return s
    }

    var result strings.Builder
    words := strings.Fields(s)
    lineLen := 0

    for i, word := range words {
        if lineLen+len(word)+1 > width && lineLen > 0 {
            result.WriteString("\n")
            lineLen = 0
        }
        if lineLen > 0 {
            result.WriteString(" ")
            lineLen++
        }
        result.WriteString(word)
        lineLen += len(word)

        if i < len(words)-1 && lineLen > 0 {
            // Continue on same line if possible
        }
    }

    return result.String()
}

// Styles
var (
    detailHeaderStyle = lipgloss.NewStyle().Bold(true).Underline(true)
    detailLabelStyle  = lipgloss.NewStyle().Bold(true)
    emptyDetailStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
    hintStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/agents/detail.go` | Agent detail sidebar component |
| `internal/tui/agents/detail_test.go` | View rendering tests |

---

## Acceptance Criteria

- [ ] Shows full agent details
- [ ] Task description not truncated (word-wrapped)
- [ ] Duration updates in real-time
- [ ] Keyboard shortcuts displayed
- [ ] Handles no selection gracefully
- [ ] Adapts to panel width

---

## Test Strategy

### Unit Tests
- Render with nil agent
- Render with agent (all statuses)
- Word wrap at various widths
- Duration formatting

### Visual Tests
- Various terminal widths
- Long task descriptions
