# TUI-PERF-01: Dashboard Shell

> **Estimated Hours:** 2.0
> **Priority:** P1 - Foundation
> **Dependencies:** None
> **Phase:** 1 - Foundation

---

## Description

Create the main dashboard shell component that provides navigation structure and layout management for all TUI views. This is the container that hosts Claude panel, agent tree, and performance views.

**Features:**
- Tab-based navigation between views
- Session selector (current, today, week, all-time)
- Help overlay with keyboard shortcuts
- Responsive layout handling

---

## Tasks

### 1. Create Dashboard Model

**File:** `internal/tui/performance/dashboard.go`

```go
package performance

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type ViewID int

const (
    ViewClaude ViewID = iota
    ViewAgents
    ViewStats
    ViewQuery
)

type Model struct {
    width       int
    height      int
    ready       bool
    activeView  ViewID
    showHelp    bool

    // Session filter
    sessionFilter string // "current", "today", "week", "all"

    // Sub-components (will be added in later tickets)
    // claudePanel  claude.Model
    // agentTree    agents.Model
    // statsPanel   stats.Model
}

func New() Model {
    return Model{
        activeView:    ViewClaude,
        sessionFilter: "current",
    }
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        return m, nil

    case tea.KeyMsg:
        return m.handleKey(msg)
    }

    return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Global keys
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "?":
        m.showHelp = !m.showHelp
        return m, nil
    case "1":
        m.activeView = ViewClaude
    case "2":
        m.activeView = ViewAgents
    case "3":
        m.activeView = ViewStats
    case "4":
        m.activeView = ViewQuery
    case "tab":
        m.activeView = (m.activeView + 1) % 4
    case "shift+tab":
        m.activeView = (m.activeView + 3) % 4
    }

    return m, nil
}

func (m Model) View() string {
    if !m.ready {
        return "Initializing..."
    }

    if m.showHelp {
        return m.renderHelp()
    }

    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.renderBanner(),
        m.renderContent(),
        m.renderStatusBar(),
    )
}
```

### 2. Create Banner Component

```go
func (m Model) renderBanner() string {
    tabs := []struct {
        id    ViewID
        label string
        key   string
    }{
        {ViewClaude, "Claude", "1"},
        {ViewAgents, "Agents", "2"},
        {ViewStats, "Stats", "3"},
        {ViewQuery, "Query", "4"},
    }

    var rendered []string
    for _, tab := range tabs {
        label := fmt.Sprintf("[%s] %s", tab.key, tab.label)
        if tab.id == m.activeView {
            rendered = append(rendered, activeTabStyle.Render(label))
        } else {
            rendered = append(rendered, inactiveTabStyle.Render(label))
        }
    }

    tabBar := strings.Join(rendered, " │ ")

    // Right side: session info
    sessionInfo := fmt.Sprintf("Session: %s │ Filter: %s",
        truncateID(m.sessionID, 8), m.sessionFilter)

    // Combine with spacing
    leftWidth := lipgloss.Width(tabBar)
    rightWidth := lipgloss.Width(sessionInfo)
    padding := m.width - leftWidth - rightWidth - 4
    if padding < 1 {
        padding = 1
    }

    return bannerStyle.Width(m.width).Render(
        tabBar + strings.Repeat(" ", padding) + sessionInfo,
    )
}
```

### 3. Create Status Bar

```go
func (m Model) renderStatusBar() string {
    left := "[Tab] Switch View  [?] Help  [q] Quit"
    right := fmt.Sprintf("Cost: $%.2f", m.sessionCost)

    padding := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
    if padding < 1 {
        padding = 1
    }

    return statusBarStyle.Width(m.width).Render(
        left + strings.Repeat(" ", padding) + right,
    )
}
```

### 4. Create Help Overlay

```go
func (m Model) renderHelp() string {
    help := `
GOgent TUI - Keyboard Shortcuts
═══════════════════════════════

Navigation
  [1-4]       Switch to view by number
  [Tab]       Next view
  [Shift+Tab] Previous view
  [?]         Toggle this help

Claude Panel
  [Enter]     Send message
  [Esc]       Cancel input
  [Ctrl+L]    Clear conversation

Agent Tree
  [↑/↓]       Navigate agents
  [Enter]     View agent details
  [q]         Query selected agent

General
  [Ctrl+C]    Quit
  [r]         Refresh data

Press any key to close...
`

    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        helpStyle.Render(help),
    )
}
```

### 5. Define Styles

**File:** `internal/tui/performance/styles.go`

```go
package performance

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
    highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
    accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

    // Banner
    bannerStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("#1a1a2e")).
        Foreground(lipgloss.Color("#eaeaea")).
        Padding(0, 1)

    activeTabStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(accent)

    inactiveTabStyle = lipgloss.NewStyle().
        Foreground(subtle)

    // Status bar
    statusBarStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("#16213e")).
        Foreground(lipgloss.Color("#a0a0a0")).
        Padding(0, 1)

    // Help
    helpStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(highlight).
        Padding(1, 2).
        Width(50)
)
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/performance/dashboard.go` | Main dashboard model |
| `internal/tui/performance/styles.go` | Lipgloss style definitions |
| `internal/tui/performance/dashboard_test.go` | Unit tests |

---

## Acceptance Criteria

- [ ] Dashboard shell renders with banner and status bar
- [ ] Tab navigation works (1-4, Tab, Shift+Tab)
- [ ] Help overlay toggles with `?`
- [ ] Window resize updates layout
- [ ] Session filter selector works
- [ ] Cost displays in status bar
- [ ] `q` or `Ctrl+C` quits application
- [ ] Placeholder content for each view
- [ ] Styles use adaptive colors (light/dark)

---

## Notes

- Keep dashboard as thin coordination layer
- Views will be added as child components in later tickets
- Use `m.ready` flag to handle initial WindowSizeMsg
- Consider saving view state for session persistence
