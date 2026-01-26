# TUI-MAIN-01: Banner & Navigation

> **Estimated Hours:** 1.5
> **Priority:** P1 - Polish
> **Dependencies:** TUI-CLI-04
> **Phase:** 4 - Integration

---

## Description

Persistent vim-style top banner with navigation tabs and session info.

---

## Layout

```
+-----------------------------------------------------------------------------+
| [1] Claude  [2] Agents  [3] Stats  [4] Query    Session: abc | Cost: $0.34  |
+-----------------------------------------------------------------------------+
```

---

## Features

- Always visible (1 line height)
- Tab highlighting for active view
- Session ID (truncated to 8 chars)
- Running cost total
- Keyboard hints (number keys)

---

## Tasks

### 1. Create Banner Component

**File:** `internal/tui/main/banner.go`

```go
package main

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

type View int

const (
    ViewClaude View = iota
    ViewAgents
    ViewStats
    ViewQuery
)

type BannerModel struct {
    activeView View
    sessionID  string
    cost       float64
    width      int
}

func NewBannerModel(sessionID string) BannerModel {
    return BannerModel{
        activeView: ViewClaude,
        sessionID:  sessionID,
    }
}

func (m *BannerModel) SetActiveView(view View) {
    m.activeView = view
}

func (m *BannerModel) SetCost(cost float64) {
    m.cost = cost
}

func (m *BannerModel) SetWidth(width int) {
    m.width = width
}
```

### 2. Implement Tab Rendering

```go
func (m BannerModel) View() string {
    tabs := []struct {
        key   string
        label string
        view  View
    }{
        {"1", "Claude", ViewClaude},
        {"2", "Agents", ViewAgents},
        {"3", "Stats", ViewStats},
        {"4", "Query", ViewQuery},
    }

    var tabStrings []string
    for _, tab := range tabs {
        label := fmt.Sprintf("[%s] %s", tab.key, tab.label)
        if tab.view == m.activeView {
            tabStrings = append(tabStrings, activeTabStyle.Render(label))
        } else {
            tabStrings = append(tabStrings, inactiveTabStyle.Render(label))
        }
    }

    tabSection := strings.Join(tabStrings, "  ")

    // Session info (right-aligned)
    sessionInfo := fmt.Sprintf("Session: %s | Cost: $%.2f",
        truncateSessionID(m.sessionID),
        m.cost,
    )

    // Calculate padding for right alignment
    usedWidth := lipgloss.Width(tabSection) + lipgloss.Width(sessionInfo)
    padding := m.width - usedWidth - 4 // Account for borders
    if padding < 2 {
        padding = 2
    }

    content := tabSection + strings.Repeat(" ", padding) + sessionInfoStyle.Render(sessionInfo)

    return bannerStyle.Width(m.width).Render(content)
}

func truncateSessionID(id string) string {
    if len(id) <= 8 {
        return id
    }
    return id[:8]
}

// Styles
var (
    bannerStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("236")).
        Padding(0, 1)

    activeTabStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("cyan")).
        Background(lipgloss.Color("236"))

    inactiveTabStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("250")).
        Background(lipgloss.Color("236"))

    sessionInfoStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("245")).
        Background(lipgloss.Color("236"))
)
```

### 3. Integrate with Main Model

```go
// Add to main/layout.go

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... existing code ...

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "1":
            m.banner.SetActiveView(ViewClaude)
            m.activeView = ViewClaude
        case "2":
            m.banner.SetActiveView(ViewAgents)
            m.activeView = ViewAgents
        case "3":
            m.banner.SetActiveView(ViewStats)
            m.activeView = ViewStats
        case "4":
            m.banner.SetActiveView(ViewQuery)
            m.activeView = ViewQuery
        }
    }

    // ... rest of update ...
}

func (m Model) View() string {
    banner := m.banner.View()

    // Main content below banner
    content := m.renderActiveView()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        banner,
        content,
    )
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/main/banner.go` | Banner component |
| `internal/tui/main/banner_test.go` | Banner rendering tests |

---

## Acceptance Criteria

- [ ] Banner renders at top, always visible
- [ ] Active tab highlighted (bold + color)
- [ ] Session ID truncated to 8 chars
- [ ] Cost updates in real-time
- [ ] Number keys (1-4) switch views
- [ ] Responsive width handling

---

## Test Strategy

### Unit Tests
- Tab rendering with each active view
- Session ID truncation
- Width calculation and padding

### Visual Tests
- Various terminal widths
- Tab highlighting visibility
