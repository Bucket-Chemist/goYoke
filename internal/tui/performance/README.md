# Performance Dashboard - TUI Component

This package implements the main dashboard shell for the GOgent Fortress TUI, providing navigation structure and layout management for all TUI views.

## Ticket

**GOgent-111 (TUI-PERF-01)**: Performance Dashboard Shell

## Architecture

The dashboard provides a Bubble Tea model that serves as the container for all TUI views:

```
┌─────────────────────────────────────────────────┐
│ [1] Claude │ [2] Agents │ [3] Stats │ [4] Query │ Filter: current
├─────────────────────────────────────────────────┤
│                                                 │
│              Active View Content                │
│          (Claude/Agents/Stats/Query)            │
│                                                 │
├─────────────────────────────────────────────────┤
│ [Tab] Switch View  [?] Help  [q] Quit    Cost: $0.00 │
└─────────────────────────────────────────────────┘
```

## Components

### Files

- **dashboard.go**: Main Bubble Tea model with navigation logic
- **styles.go**: Lipgloss style definitions with adaptive colors
- **dashboard_test.go**: Comprehensive test suite (32 tests)
- **example_main.go**: Standalone demonstration program

### Model Structure

```go
type Model struct {
    width, height int       // Terminal dimensions
    ready         bool      // Has received WindowSizeMsg
    activeView    ViewID    // Current view (0-3)
    showHelp      bool      // Help overlay toggle
    sessionFilter string    // "current", "today", "week", "all"
    sessionCost   float64   // Placeholder for cost calculation
}
```

### View IDs

```go
const (
    ViewClaude ViewID = iota  // Claude conversation panel
    ViewAgents                 // Agent tree view
    ViewStats                  // Performance statistics
    ViewQuery                  // Query interface
)
```

## Keyboard Shortcuts

### Navigation
- **1-4**: Switch to specific view by number
- **Tab**: Cycle forward through views
- **Shift+Tab**: Cycle backward through views
- **?**: Toggle help overlay

### General
- **q** or **Ctrl+C**: Quit application
- **Any key**: Dismiss help overlay (when showing)

## Usage

### As a Library

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/performance"
)

func main() {
    p := tea.NewProgram(performance.New(), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        // Handle error
    }
}
```

### Run Example

```bash
go run internal/tui/performance/example_main.go
```

## Testing

```bash
# Run all tests
go test ./internal/tui/performance/

# Run with verbose output
go test -v ./internal/tui/performance/

# Run with race detection
go test -race ./internal/tui/performance/
```

### Test Coverage

- Initial state and construction
- Window resize handling
- View switching (number keys, Tab, Shift+Tab)
- Help overlay toggle
- Quit conditions
- Rendering components (banner, content, status bar, help)
- Edge cases (minimum dimensions, unknown keys, etc.)

**Total: 32 test functions, 100% acceptance criteria coverage**

## Features

### Responsive Layout

The dashboard adjusts to terminal size:
- Banner width matches terminal width
- Content height calculated dynamically (total height - banner - status bar)
- Status bar stretches full width
- Graceful handling of minimum dimensions

### Adaptive Styling

Colors automatically adjust for light/dark terminals:

```go
subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
```

### Help System

Press `?` to display centered help overlay with all keyboard shortcuts. Press any key to dismiss.

## Integration Points

The dashboard is designed as a thin coordination layer. Future tickets will add actual view components:

### GOgent-118: Claude Conversation Panel
```go
// Will be added to Model:
claudePanel claude.Model
```

### GOgent-116: Agent Tree View
```go
// Will be added to Model:
agentTree agents.Model
```

### Future: Stats Panel
```go
// Will be added to Model:
statsPanel stats.Model
```

### Future: Query Interface
```go
// Will be added to Model:
queryPanel query.Model
```

## Design Decisions

### Placeholder Content

Currently, all views show placeholder text indicating which component will be implemented:
- "Claude Conversation Panel (Implementation: GOgent-118)"
- "Agent Tree View (Implementation: GOgent-116)"
- "Performance Statistics (Future implementation)"
- "Query Interface (Future implementation)"

### Session Cost

The `sessionCost` field is a placeholder (always $0.00) that will be calculated when session metrics are integrated.

### Session Filter

The `sessionFilter` field is displayed but not yet functional. Switching logic will be added when data loading is implemented.

## Conventions Compliance

Follows **go.md** conventions:
- Table-driven tests
- Proper error handling patterns
- Package-level documentation
- Exported/unexported naming conventions
- Race detector clean

## Dependencies

- **github.com/charmbracelet/bubbletea**: TUI framework
- **github.com/charmbracelet/lipgloss**: Terminal styling
- **github.com/stretchr/testify**: Test assertions

## Status

**Status**: Complete
**Tests**: All passing (32/32)
**Next Tickets**: GOgent-116 (Agent Tree), GOgent-118 (Claude Panel)
