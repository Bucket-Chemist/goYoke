# TUI-CLI-03: Claude Interface Panel

> **Estimated Hours:** 4.0
> **Priority:** P1 - Core Feature
> **Dependencies:** TUI-CLI-02
> **Phase:** 4 - Integration

---

## Description

Bubble Tea component for Claude CLI interaction with streaming output display.

---

## Layout

```
+-----------------------------------------------------------+
| Claude Code - Session: abc123           Cost: $0.23       |
+-----------------------------------------------------------+
|  You: Explain this function                               |
|                                                           |
|  Claude: This function implements a binary search...      |
|  [streaming...]                                           |
|  ---------------------------------------------------------|
|  | Hook: gogent-validate [check] | Tool: Read [check] |   |
+-----------------------------------------------------------+
| > Type your message here...                       [Enter] |
+-----------------------------------------------------------+
```

---

## Key Features

- Viewport for scrollable conversation history
- Textarea for user input
- Real-time streaming text display (typewriter effect)
- Hook event sidebar showing recent hook activity
- Cost display updated after each response
- Session ID in header

---

## Tasks

### 1. Create Main Panel Model

**File:** `internal/tui/claude/panel.go`

```go
package claude

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

type PanelModel struct {
    process   *cli.ClaudeProcess
    viewport  viewport.Model
    textarea  textarea.Model
    messages  []Message
    hooks     []HookEvent
    cost      float64
    sessionID string
    width     int
    height    int
    focused   bool
    streaming bool
}

type Message struct {
    Role    string // "user" or "assistant"
    Content string
}

type HookEvent struct {
    Name    string
    Success bool
}

func NewPanelModel(process *cli.ClaudeProcess) PanelModel {
    ta := textarea.New()
    ta.Placeholder = "Type your message here..."
    ta.Focus()
    ta.CharLimit = 4096
    ta.SetWidth(80)
    ta.SetHeight(1)

    vp := viewport.New(80, 20)

    return PanelModel{
        process:   process,
        viewport:  vp,
        textarea:  ta,
        messages:  make([]Message, 0),
        hooks:     make([]HookEvent, 0),
        sessionID: process.SessionID(),
    }
}
```

### 2. Create Input Handling

**File:** `internal/tui/claude/input.go`

```go
package claude

import (
    tea "github.com/charmbracelet/bubbletea"
)

func (m PanelModel) handleInput(msg tea.KeyMsg) (PanelModel, tea.Cmd) {
    switch msg.String() {
    case "enter":
        if !m.streaming && m.textarea.Value() != "" {
            content := m.textarea.Value()
            m.messages = append(m.messages, Message{
                Role:    "user",
                Content: content,
            })
            m.textarea.Reset()
            m.streaming = true

            // Send to Claude process
            return m, m.sendMessage(content)
        }
    case "esc":
        m.textarea.Reset()
    case "ctrl+l":
        // Clear conversation (visual only)
        m.messages = make([]Message, 0)
        m.updateViewport()
    }

    var cmd tea.Cmd
    m.textarea, cmd = m.textarea.Update(msg)
    return m, cmd
}

func (m PanelModel) sendMessage(content string) tea.Cmd {
    return func() tea.Msg {
        err := m.process.Send(content)
        if err != nil {
            return errMsg{err}
        }
        return nil
    }
}

type errMsg struct{ error }
```

### 3. Create Output Display

**File:** `internal/tui/claude/output.go`

```go
package claude

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

func (m *PanelModel) updateViewport() {
    var b strings.Builder

    for _, msg := range m.messages {
        switch msg.Role {
        case "user":
            b.WriteString(userStyle.Render("You: "))
            b.WriteString(msg.Content)
        case "assistant":
            b.WriteString(assistantStyle.Render("Claude: "))
            b.WriteString(msg.Content)
        }
        b.WriteString("\n\n")
    }

    if m.streaming {
        b.WriteString(streamingStyle.Render("[streaming...]"))
    }

    m.viewport.SetContent(b.String())
    m.viewport.GotoBottom()
}

func (m *PanelModel) appendStreamingText(text string) {
    if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
        m.messages[len(m.messages)-1].Content += text
    } else {
        m.messages = append(m.messages, Message{
            Role:    "assistant",
            Content: text,
        })
    }
    m.updateViewport()
}

var (
    userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan"))
    assistantStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("green"))
    streamingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)
```

### 4. Create Hook Event Sidebar

**File:** `internal/tui/claude/events.go`

```go
package claude

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

func (m PanelModel) renderHookSidebar() string {
    var b strings.Builder

    b.WriteString(hookHeaderStyle.Render("Recent Hooks"))
    b.WriteString("\n")

    // Show last 5 hooks
    start := 0
    if len(m.hooks) > 5 {
        start = len(m.hooks) - 5
    }

    for _, hook := range m.hooks[start:] {
        indicator := "[check]"
        style := hookSuccessStyle
        if !hook.Success {
            indicator = "[x]"
            style = hookFailStyle
        }
        b.WriteString(style.Render(fmt.Sprintf("%s %s", indicator, hook.Name)))
        b.WriteString("\n")
    }

    return b.String()
}

func (m *PanelModel) addHookEvent(name string, success bool) {
    m.hooks = append(m.hooks, HookEvent{
        Name:    name,
        Success: success,
    })
}

var (
    hookHeaderStyle  = lipgloss.NewStyle().Bold(true)
    hookSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
    hookFailStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
)
```

### 5. Implement Main Update and View

```go
func (m PanelModel) Update(msg tea.Msg) (PanelModel, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.focused {
            var cmd tea.Cmd
            m, cmd = m.handleInput(msg)
            cmds = append(cmds, cmd)
        }

    case cli.Event:
        m = m.handleEvent(msg)

    case errMsg:
        // Handle error
        m.streaming = false
    }

    return m, tea.Batch(cmds...)
}

func (m PanelModel) handleEvent(event cli.Event) PanelModel {
    switch event.Type {
    case "assistant":
        if ae, err := event.AsAssistant(); err == nil {
            for _, block := range ae.Message.Content {
                if block.Type == "text" {
                    m.appendStreamingText(block.Text)
                }
            }
        }
    case "result":
        if re, err := event.AsResult(); err == nil {
            m.cost = re.TotalCostUSD
            m.streaming = false
        }
    case "system":
        if se, err := event.AsSystem(); err == nil {
            if se.Subtype == "hook_response" {
                m.addHookEvent(se.HookName, se.ExitCode == 0)
            }
        }
    }
    return m
}

func (m PanelModel) View() string {
    // Header
    header := headerStyle.Render(fmt.Sprintf(
        "Claude Code - Session: %s           Cost: $%.2f",
        truncate(m.sessionID, 8),
        m.cost,
    ))

    // Main content with hook sidebar
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        m.viewport.View(),
        m.renderHookSidebar(),
    )

    // Input area
    input := m.textarea.View()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        content,
        input,
    )
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/claude/panel.go` | Main panel model |
| `internal/tui/claude/input.go` | User input handling |
| `internal/tui/claude/output.go` | Streaming output display |
| `internal/tui/claude/events.go` | Hook event sidebar |

---

## Acceptance Criteria

- [ ] Text streams character-by-character with partial messages
- [ ] Input submits on Enter
- [ ] Hook events display in sidebar
- [ ] Cost updates after each response
- [ ] Scroll works in conversation viewport
- [ ] Session ID displayed in header
- [ ] Esc clears/cancels input
- [ ] Ctrl+L clears conversation (visual only)

---

## Test Strategy

### Unit Tests
- Message appending
- Event handling for each type
- Hook event tracking
- Viewport content formatting

### Integration Tests
- End-to-end with mock Claude process
- Streaming behavior verification
