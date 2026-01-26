# GO Bubbletea TUI Conventions - Lisan al-Gaib

## Overview

Bubbletea implements The Elm Architecture (TEA/MVU) for terminal UIs. These conventions ensure professional TUIs with proper state management, component composition, and styling.

## The Elm Architecture

### Core Principles

1. **Model**: All application state in a single struct
2. **View**: Pure function that renders state to string (no side effects)
3. **Update**: Pure function that handles messages and returns new state
4. **Commands**: The ONLY way to perform I/O

### Basic Structure

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// Model holds ALL application state
type Model struct {
    width    int
    height   int
    items    []string
    cursor   int
    selected map[int]struct{}
    loading  bool
    err      error
}

// Init returns initial commands to run
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        fetchItems,      // Load initial data
        tea.EnterAltScreen, // Use alternate screen buffer
    )
}

// Update handles messages and returns new model + commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
        
    case tea.KeyMsg:
        return m.handleKey(msg)
        
    case itemsLoadedMsg:
        m.items = msg.items
        m.loading = false
        return m, nil
        
    case errMsg:
        m.err = msg.err
        m.loading = false
        return m, nil
    }
    
    return m, nil
}

// View renders the UI - MUST be fast, NO I/O
func (m Model) View() string {
    if m.loading {
        return "Loading..."
    }
    if m.err != nil {
        return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
    }
    return m.renderList()
}
```

## Messages and Commands

### Custom Message Types

```go
// Define message types for your data flows
type itemsLoadedMsg struct {
    items []string
}

type errMsg struct {
    err error
}

type statusUpdateMsg string

type tickMsg time.Time
```

### Commands (The ONLY Way to Do I/O)

```go
// Commands are functions that return a Msg
func fetchItems() tea.Msg {
    items, err := api.FetchItems()
    if err != nil {
        return errMsg{err}
    }
    return itemsLoadedMsg{items}
}

// Returning commands from Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            // Return command to process selected item
            return m, processItem(m.items[m.cursor])
        case "r":
            // Return command to refresh
            m.loading = true
            return m, fetchItems
        }
    }
    return m, nil
}

// Command factory
func processItem(item string) tea.Cmd {
    return func() tea.Msg {
        result, err := api.Process(item)
        if err != nil {
            return errMsg{err}
        }
        return processCompleteMsg{result}
    }
}
```

### NEVER Modify State in Goroutines

```go
// WRONG: Race condition, undefined behavior
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        go func() {
            data := fetchData()
            m.data = data  // BUG: Modifying model outside Update!
        }()
    }
    return m, nil
}

// CORRECT: Return command, let Bubbletea manage goroutine
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.String() == "enter" {
        return m, fetchDataCmd
    }
    return m, nil
}

func fetchDataCmd() tea.Msg {
    data := fetchData()
    return dataLoadedMsg{data}
}
```

## Keyboard Handling

### Pattern for Key Messages

```go
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Global keys (work in any mode)
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    }
    
    // Mode-specific keys
    switch m.mode {
    case modeNormal:
        return m.handleNormalKey(msg)
    case modeInput:
        return m.handleInputKey(msg)
    case modeHelp:
        return m.handleHelpKey(msg)
    }
    
    return m, nil
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
    case "down", "j":
        if m.cursor < len(m.items)-1 {
            m.cursor++
        }
    case "enter":
        return m, processSelected(m.items[m.cursor])
    case "/":
        m.mode = modeInput
        m.input = ""
    case "?":
        m.mode = modeHelp
    }
    return m, nil
}
```

### Special Key Sequences

```go
switch msg.Type {
case tea.KeyCtrlC:
    return m, tea.Quit
case tea.KeyEsc:
    m.mode = modeNormal
case tea.KeyEnter:
    return m.submitInput()
case tea.KeyBackspace:
    if len(m.input) > 0 {
        m.input = m.input[:len(m.input)-1]
    }
case tea.KeyRunes:
    m.input += string(msg.Runes)
}
```

## Component Composition

### Parent-Child Pattern

```go
// Child component
type ListComponent struct {
    items    []string
    cursor   int
    focused  bool
}

func (l ListComponent) Update(msg tea.Msg) (ListComponent, tea.Cmd) {
    if !l.focused {
        return l, nil
    }
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if l.cursor > 0 {
                l.cursor--
            }
        case "down", "j":
            if l.cursor < len(l.items)-1 {
                l.cursor++
            }
        case "enter":
            // Return custom message for parent to handle
            return l, func() tea.Msg {
                return itemSelectedMsg{l.items[l.cursor]}
            }
        }
    }
    return l, nil
}

func (l ListComponent) View() string {
    var b strings.Builder
    for i, item := range l.items {
        cursor := " "
        if l.focused && i == l.cursor {
            cursor = "â–¸"
        }
        b.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
    }
    return b.String()
}

// Parent model
type Model struct {
    list    ListComponent
    detail  DetailComponent
    focus   string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    // Handle component-specific messages
    switch msg := msg.(type) {
    case itemSelectedMsg:
        m.detail = NewDetailComponent(msg.item)
        return m, nil
    case tea.KeyMsg:
        if msg.String() == "tab" {
            // Toggle focus
            if m.focus == "list" {
                m.focus = "detail"
                m.list.focused = false
                m.detail.focused = true
            } else {
                m.focus = "list"
                m.list.focused = true
                m.detail.focused = false
            }
            return m, nil
        }
    }
    
    // Update focused component
    var cmd tea.Cmd
    if m.focus == "list" {
        m.list, cmd = m.list.Update(msg)
        cmds = append(cmds, cmd)
    } else {
        m.detail, cmd = m.detail.Update(msg)
        cmds = append(cmds, cmd)
    }
    
    return m, tea.Batch(cmds...)
}
```

## Lipgloss Styling

### Style Definitions

```go
var (
    // Colors
    subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
    highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
    special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
    
    // Styles
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(highlight).
        Padding(0, 1)
    
    itemStyle = lipgloss.NewStyle().
        PaddingLeft(2)
    
    selectedItemStyle = lipgloss.NewStyle().
        PaddingLeft(2).
        Foreground(special).
        Bold(true)
    
    statusBarStyle = lipgloss.NewStyle().
        Foreground(subtle).
        Padding(0, 1)
    
    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#626262"))
    
    // Box styles
    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(subtle).
        Padding(1, 2)
)
```

### Responsive Layout

```go
func (m Model) View() string {
    // Calculate available space
    contentWidth := m.width - 4  // Account for borders
    contentHeight := m.height - 6  // Account for header/footer
    
    // Build components with dynamic sizing
    header := titleStyle.Width(m.width).Render("My App")
    
    content := boxStyle.
        Width(contentWidth).
        Height(contentHeight).
        Render(m.renderContent())
    
    footer := statusBarStyle.Width(m.width).Render(m.status)
    
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### Layout Helpers

```go
// Horizontal layout
row := lipgloss.JoinHorizontal(lipgloss.Top,
    leftPanel.Render(leftContent),
    rightPanel.Render(rightContent),
)

// Vertical layout
column := lipgloss.JoinVertical(lipgloss.Left,
    header,
    content,
    footer,
)

// Centering
centered := lipgloss.Place(
    m.width, m.height,
    lipgloss.Center, lipgloss.Center,
    content,
)

// Inline styling
text := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#FF0000")).
    Render("Error!")
```

## Spinners and Progress

### Spinner Component

```go
import "github.com/charmbracelet/bubbles/spinner"

type Model struct {
    spinner  spinner.Model
    loading  bool
}

func NewModel() Model {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    return Model{spinner: s, loading: true}
}

func (m Model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m Model) View() string {
    if m.loading {
        return fmt.Sprintf("%s Loading...", m.spinner.View())
    }
    return "Done!"
}
```

### Progress Bar

```go
import "github.com/charmbracelet/bubbles/progress"

type Model struct {
    progress progress.Model
    percent  float64
}

func NewModel() Model {
    p := progress.New(
        progress.WithDefaultGradient(),
        progress.WithWidth(40),
    )
    return Model{progress: p}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case progressMsg:
        m.percent = float64(msg)
        return m, nil
    case progress.FrameMsg:
        pm, cmd := m.progress.Update(msg)
        m.progress = pm.(progress.Model)
        return m, cmd
    }
    return m, nil
}

func (m Model) View() string {
    return m.progress.ViewAs(m.percent)
}
```

## Text Input

```go
import "github.com/charmbracelet/bubbles/textinput"

type Model struct {
    input    textinput.Model
    mode     string
}

func NewModel() Model {
    ti := textinput.New()
    ti.Placeholder = "Enter text..."
    ti.Focus()
    ti.CharLimit = 156
    ti.Width = 40
    return Model{input: ti, mode: "input"}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEnter:
            value := m.input.Value()
            m.input.Reset()
            return m, processInput(value)
        case tea.KeyEsc:
            m.mode = "normal"
            m.input.Blur()
        }
    }
    
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    return m.input.View()
}
```

## Ticking/Animation

```go
type tickMsg time.Time

func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m Model) Init() tea.Cmd {
    return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tickMsg:
        m.currentTime = time.Time(msg)
        return m, tickCmd()  // Continue ticking
    }
    return m, nil
}
```

## Program Options

### Starting the Program

```go
func main() {
    p := tea.NewProgram(
        NewModel(),
        tea.WithAltScreen(),        // Use alternate screen buffer
        tea.WithMouseCellMotion(),  // Enable mouse support
    )
    
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Sending Messages from Outside

```go
func main() {
    p := tea.NewProgram(NewModel())
    
    // Send message from another goroutine
    go func() {
        time.Sleep(5 * time.Second)
        p.Send(externalUpdateMsg{data: "new data"})
    }()
    
    p.Run()
}
```

## Testing TUI Components

```go
func TestModelUpdate(t *testing.T) {
    m := NewModel()
    
    // Simulate key press
    newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
    updatedM := newModel.(Model)
    
    assert.Equal(t, 1, updatedM.cursor)
}

func TestModelView(t *testing.T) {
    m := Model{
        items:  []string{"a", "b", "c"},
        cursor: 0,
    }
    
    view := m.View()
    assert.Contains(t, view, "â–¸ a")  // Selected indicator
    assert.Contains(t, view, "  b")  // Non-selected
}
```

## Sharp Edges

### 1. View Must Be Fast

```go
// WRONG: I/O in View
func (m Model) View() string {
    data, _ := os.ReadFile("data.txt")  // BUG: Blocking I/O
    return string(data)
}

// CORRECT: Load data via commands, render from state
func (m Model) View() string {
    return m.data  // Already loaded in state
}
```

### 2. Don't Forget tea.Batch

```go
// WRONG: Lost command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd1, cmd2 tea.Cmd
    m.spinner, cmd1 = m.spinner.Update(msg)
    m.list, cmd2 = m.list.Update(msg)
    return m, cmd1  // BUG: cmd2 lost!
}

// CORRECT: Batch all commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)
    
    m.list, cmd = m.list.Update(msg)
    cmds = append(cmds, cmd)
    
    return m, tea.Batch(cmds...)
}
```

### 3. WindowSizeMsg on Startup

```go
// The first message is often WindowSizeMsg
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true  // Now safe to render
    }
    // ...
}

func (m Model) View() string {
    if !m.ready {
        return "Initializing..."  // Don't render until we have dimensions
    }
    // ...
}
```

### 4. Alt Screen Buffer

```go
// Use alt screen to preserve terminal history
p := tea.NewProgram(model, tea.WithAltScreen())

// Exit cleanly restores original screen
// Don't use os.Exit() - let Run() return normally
```
