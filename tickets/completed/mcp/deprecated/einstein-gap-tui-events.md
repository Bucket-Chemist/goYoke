# TUI Event Handling GAP Document

## Problem Statement

GOgent-Fortress TUI (`gofortress`) has two remaining issues with Claude CLI event handling:

1. **Permission requests not displayed** - When Claude needs tool approval, the TUI shows `[Event: user]` but doesn't render the actual permission prompt or allow user response
2. **No collapsible UI** - Tool use and event details are verbose; need expand/collapse for readability

---

## Current State

### What Works
- ✅ Streaming text responses
- ✅ Tool use display: `🔧 [Bash] $ command...`
- ✅ Thinking display: `💭 thinking text...`
- ✅ `/model` and `/context` commands
- ✅ Model switching with process restart
- ✅ System messages render in yellow

### What's Broken
- ❌ `[Event: user]` shows but content not parsed/displayed
- ❌ User cannot respond to permission prompts
- ❌ Write tool fails repeatedly (permission never granted)
- ❌ No way to collapse verbose tool output

---

## Technical Context

### Key Files
| File | Purpose |
|------|---------|
| `internal/tui/claude/events.go` | Event handling, tool display |
| `internal/tui/claude/panel.go` | TUI model, state, Update() |
| `internal/tui/claude/input.go` | User input handling |
| `internal/cli/events.go` | Event types, parsing |
| `internal/cli/subprocess.go` | Process I/O, Send() |

### Event Flow
```
Claude CLI stdout → NDJSONReader → parseEvent() → cp.events channel
→ waitForEvent() → panel.Update(cli.Event) → handleEvent()
```

### Current handleEvent() Cases
- `"assistant"` → text, tool_use, thinking blocks
- `"result"` → cost update, streaming=false
- `"system"` → hook responses
- `"error"` → error display
- `default` → shows `[Event: type/subtype]`

---

## Task 1: Permission Request Handling

### Problem
Claude sends `type: "user"` events for permission prompts. We log `[Event: user]` but don't:
1. Parse the permission details
2. Display the prompt to user
3. Provide a way to respond (approve/deny)
4. Send the response back to Claude

### Investigation Needed
1. What fields does a "user" event contain?
   - Likely: `message`, `options`, `prompt` or similar
2. How to respond in stream-json mode?
   - Likely: Send `{"type": "user", "message": {...}}` with approval

### Implementation Approach
```go
// In events.go handleEvent()
case "user":
    // Parse permission request
    var userEvent struct {
        Message string   `json:"message"`
        Options []string `json:"options"` // e.g., ["Allow", "Deny"]
    }
    json.Unmarshal(event.Raw, &userEvent)

    // Display prompt
    m.showPermissionPrompt(userEvent.Message, userEvent.Options)
    m.awaitingPermission = true

// In input.go - handle permission response
if m.awaitingPermission {
    // Map key to response
    // Send response via m.process.Send() or SendJSON()
}
```

### Acceptance Criteria
- [ ] Permission prompts display with options
- [ ] User can approve/deny via keyboard
- [ ] Response sent to Claude
- [ ] Write tool works after approval

---

## Task 2: Collapsible UI Elements

### Problem
Tool output can be verbose (long bash commands, file paths). Need expand/collapse.

### Design
```
Collapsed: 🔧 [Bash] $ go build... ▶
Expanded:  🔧 [Bash] ▼
           $ go build -o ~/.local/bin/gofortress ./cmd/gofortress
           [output if captured]
```

### Implementation Approach

1. **Track expandable items:**
```go
type ExpandableItem struct {
    ID        string
    Collapsed bool
    Summary   string  // Short version
    Details   string  // Full version
}

// In PanelModel
expandables map[string]*ExpandableItem
```

2. **Toggle mechanism:**
- Click or key (e.g., `Tab` when item focused)
- Or number keys to toggle specific items

3. **Render logic:**
```go
func (m PanelModel) renderExpandable(item *ExpandableItem) string {
    if item.Collapsed {
        return item.Summary + " ▶"
    }
    return item.Summary + " ▼\n" + item.Details
}
```

### Acceptance Criteria
- [ ] Tool use blocks are collapsible
- [ ] Default state is collapsed
- [ ] Visual indicator (▶/▼) shows state
- [ ] Toggle via keyboard

---

## Dependencies

### For Permission Handling
- Need to capture actual `user` event structure (run gofortress, trigger permission, check logs)
- May need to add event type to `internal/cli/events.go`

### For Collapsible UI
- Pure TUI change, no CLI dependencies
- Consider lipgloss styling for indicators

---

## Files to Modify

| Task | Files |
|------|-------|
| Permissions | `events.go`, `panel.go`, `input.go`, possibly `cli/events.go` |
| Collapsible | `events.go`, `output.go`, `panel.go` |

---

## Testing

### Manual Test: Permissions
```bash
gofortress
> create a python script that prints hello world
# Should show permission prompt for Write tool
# Should allow approval
# Should complete successfully
```

### Manual Test: Collapsible
```bash
gofortress
> run ls -la
# Tool use should show collapsed by default
# Should expand on toggle
```

---

## Priority

1. **Permission handling** - Blocking issue, can't use Write/Edit tools
2. **Collapsible UI** - Nice to have for readability

---

## Session Context

This session fixed:
- Goroutine leak in readEvents() (single reader pattern)
- Channel replacement on restart (preserve original channels)
- Send-on-closed-channel panics (select guards)
- Missing system message rendering
- Tool use display (🔧 format with details)
- Native commands (/model, /context)

Sharp edges documented in:
- `~/.claude/agents/go-concurrent/sharp-edges.yaml` (v1.1)
- `~/.claude/agents/go-tui/sharp-edges.yaml` (v1.1)

---

## Detailed Implementation Guide

### Task 1: Permission Request Handling - Full Details

#### Step 1: Discover Event Structure

First, capture a raw "user" event to understand its structure. Add temporary logging:

```go
// In internal/tui/claude/events.go, in the default case:
default:
    // Log full event JSON for debugging
    if event.Type == "user" {
        log.Printf("USER EVENT RAW: %s", string(event.Raw))
    }
```

Or run gofortress with stderr logging visible to capture the event.

#### Step 2: Add UserEvent Type

In `internal/cli/events.go`, add:

```go
// UserEvent represents a user interaction request (permissions, input prompts)
type UserEvent struct {
    Event
    Message   string            `json:"message,omitempty"`
    Prompt    string            `json:"prompt,omitempty"`
    Options   []string          `json:"options,omitempty"`
    ToolName  string            `json:"tool_name,omitempty"`
    ToolInput map[string]interface{} `json:"tool_input,omitempty"`
}

// AsUser attempts to parse the event as UserEvent.
func (e Event) AsUser() (*UserEvent, error) {
    if e.Type != "user" {
        return nil, fmt.Errorf("event type is %q, not user", e.Type)
    }
    var ue UserEvent
    if err := json.Unmarshal(e.Raw, &ue); err != nil {
        return nil, fmt.Errorf("unmarshal user event: %w", err)
    }
    return &ue, nil
}
```

#### Step 3: Add Permission State to PanelModel

In `internal/tui/claude/panel.go`:

```go
type PanelModel struct {
    // ... existing fields ...

    // Permission handling
    awaitingPermission bool
    permissionPrompt   string
    permissionOptions  []string
    permissionToolName string
}

// PermissionResponse represents user's response to a permission request
type permissionResponseMsg struct {
    approved bool
    toolName string
}
```

#### Step 4: Handle User Event in handleEvent()

In `internal/tui/claude/events.go`:

```go
case "user":
    // Handle permission/input request
    if ue, err := event.AsUser(); err == nil {
        // Format the permission prompt
        prompt := ue.Message
        if prompt == "" {
            prompt = ue.Prompt
        }
        if ue.ToolName != "" {
            prompt = fmt.Sprintf("Allow tool '%s'?\n%s", ue.ToolName, prompt)
        }

        m.awaitingPermission = true
        m.permissionPrompt = prompt
        m.permissionOptions = ue.Options
        m.permissionToolName = ue.ToolName

        // Display in chat
        m.messages = append(m.messages, Message{
            Role:    "permission",  // New role type
            Content: prompt,
        })
        m.updateViewport()
    }
```

#### Step 5: Add Permission Role Rendering

In `internal/tui/claude/output.go`:

```go
case "permission":
    b.WriteString(permissionStyle.Render("⚠️ Permission Required: "))
    b.WriteString(msg.Content)
    b.WriteString("\n")
    b.WriteString(permissionStyle.Render("[Y]es / [N]o"))

// Add style
var permissionStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("yellow")).
    Background(lipgloss.Color("236"))
```

#### Step 6: Handle Permission Input

In `internal/tui/claude/input.go`:

```go
func (m PanelModel) handleInput(msg tea.KeyMsg) (PanelModel, tea.Cmd) {
    // Handle permission responses first
    if m.awaitingPermission {
        switch strings.ToLower(msg.String()) {
        case "y", "yes":
            m.awaitingPermission = false
            return m, m.sendPermissionResponse(true)
        case "n", "no":
            m.awaitingPermission = false
            return m, m.sendPermissionResponse(false)
        }
        // Ignore other keys while awaiting permission
        return m, nil
    }

    // ... existing input handling ...
}

// sendPermissionResponse sends approval/denial to Claude
func (m PanelModel) sendPermissionResponse(approved bool) tea.Cmd {
    return func() tea.Msg {
        // The response format depends on Claude CLI's expectation
        // Option 1: Simple text response
        response := "no"
        if approved {
            response = "yes"
        }

        // Option 2: Structured response (if needed)
        // m.process.SendJSON(cli.UserMessage{
        //     Type: "user",
        //     Message: cli.UserContent{
        //         Role: "user",
        //         Content: []cli.ContentBlock{{Type: "text", Text: response}},
        //     },
        // })

        err := m.process.Send(response)
        if err != nil {
            return errMsg{err}
        }

        // Add confirmation to chat
        return permissionResponseMsg{approved: approved, toolName: m.permissionToolName}
    }
}
```

#### Step 7: Handle Response Confirmation

In `panel.go` Update():

```go
case permissionResponseMsg:
    status := "denied"
    if msg.approved {
        status = "approved"
    }
    m.messages = append(m.messages, Message{
        Role:    "system",
        Content: fmt.Sprintf("Tool '%s' %s", msg.toolName, status),
    })
    m.updateViewport()
    return m, waitForEvent(m.process.Events())
```

---

### Task 2: Collapsible UI - Full Details

#### Step 1: Define Expandable Structure

In `internal/tui/claude/panel.go`:

```go
// ExpandableBlock represents a collapsible content block
type ExpandableBlock struct {
    ID        string    // Unique identifier (e.g., tool use ID)
    Type      string    // "tool_use", "thinking", etc.
    Summary   string    // One-line collapsed view
    Details   string    // Full expanded content
    Collapsed bool      // Current state
    Timestamp time.Time // For ordering
}

type PanelModel struct {
    // ... existing fields ...

    // Collapsible blocks
    expandables    []ExpandableBlock
    focusedBlockID string // Which block has focus (for toggling)
}
```

#### Step 2: Create Expandable Blocks on Tool Use

In `internal/tui/claude/events.go`:

```go
case "tool_use":
    // Create expandable block
    block := ExpandableBlock{
        ID:        block.ID, // From ContentBlock
        Type:      "tool_use",
        Summary:   formatToolSummary(block.Name, block.Input),
        Details:   formatToolDetails(block.Name, block.Input),
        Collapsed: true, // Default collapsed
        Timestamp: time.Now(),
    }
    m.expandables = append(m.expandables, block)

    // Add reference to messages for rendering
    m.appendStreamingText(fmt.Sprintf("\n{{EXPANDABLE:%s}}\n", block.ID))

// Helper functions
func formatToolSummary(name string, input map[string]interface{}) string {
    switch name {
    case "Bash":
        if cmd, ok := input["command"].(string); ok {
            return fmt.Sprintf("🔧 [Bash] $ %s", truncateText(cmd, 40))
        }
    case "Write":
        if path, ok := input["file_path"].(string); ok {
            return fmt.Sprintf("🔧 [Write] %s", filepath.Base(path))
        }
    // ... other tools
    }
    return fmt.Sprintf("🔧 [%s]", name)
}

func formatToolDetails(name string, input map[string]interface{}) string {
    // Pretty print the full input
    formatted, _ := json.MarshalIndent(input, "", "  ")
    return string(formatted)
}
```

#### Step 3: Render Expandable Blocks

In `internal/tui/claude/output.go`:

```go
func (m *PanelModel) updateViewport() {
    var b strings.Builder

    for _, msg := range m.messages {
        // ... existing role handling ...

        // Handle expandable placeholders
        content := msg.Content
        for _, exp := range m.expandables {
            placeholder := fmt.Sprintf("{{EXPANDABLE:%s}}", exp.ID)
            if strings.Contains(content, placeholder) {
                rendered := m.renderExpandable(exp)
                content = strings.Replace(content, placeholder, rendered, 1)
            }
        }
        b.WriteString(content)
        b.WriteString("\n\n")
    }

    m.viewport.SetContent(b.String())
    m.viewport.GotoBottom()
}

func (m PanelModel) renderExpandable(exp ExpandableBlock) string {
    indicator := "▶" // Collapsed
    if !exp.Collapsed {
        indicator = "▼" // Expanded
    }

    // Highlight if focused
    style := expandableStyle
    if exp.ID == m.focusedBlockID {
        style = expandableFocusedStyle
    }

    if exp.Collapsed {
        return style.Render(fmt.Sprintf("%s %s", exp.Summary, indicator))
    }

    return style.Render(fmt.Sprintf("%s %s\n%s",
        exp.Summary, indicator,
        expandableDetailStyle.Render(exp.Details)))
}

// Styles
var (
    expandableStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("magenta"))

    expandableFocusedStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("magenta")).
        Background(lipgloss.Color("236")).
        Bold(true)

    expandableDetailStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        PaddingLeft(2)
)
```

#### Step 4: Toggle Mechanism

In `internal/tui/claude/input.go`:

```go
// Add to handleInput switch
case "tab":
    // Toggle focused expandable block
    if m.focusedBlockID != "" {
        for i := range m.expandables {
            if m.expandables[i].ID == m.focusedBlockID {
                m.expandables[i].Collapsed = !m.expandables[i].Collapsed
                m.updateViewport()
                break
            }
        }
    }
    return m, nil

case "up", "k":
    // Move focus to previous expandable
    m.moveFocus(-1)
    return m, nil

case "down", "j":
    // Move focus to next expandable (if not in textarea)
    if !m.textareaFocused {
        m.moveFocus(1)
        return m, nil
    }

// Helper method in panel.go
func (m *PanelModel) moveFocus(direction int) {
    if len(m.expandables) == 0 {
        return
    }

    currentIdx := -1
    for i, exp := range m.expandables {
        if exp.ID == m.focusedBlockID {
            currentIdx = i
            break
        }
    }

    newIdx := currentIdx + direction
    if newIdx < 0 {
        newIdx = 0
    } else if newIdx >= len(m.expandables) {
        newIdx = len(m.expandables) - 1
    }

    m.focusedBlockID = m.expandables[newIdx].ID
    m.updateViewport()
}
```

---

## Alternative Simpler Approach for Collapsibles

If full focus management is too complex, use numbered shortcuts:

```go
// Display with numbers
🔧 [1] [Bash] $ go build... ▶
🔧 [2] [Write] main.py ▶

// Toggle with number keys
case "1", "2", "3", "4", "5", "6", "7", "8", "9":
    idx, _ := strconv.Atoi(msg.String())
    if idx > 0 && idx <= len(m.expandables) {
        m.expandables[idx-1].Collapsed = !m.expandables[idx-1].Collapsed
        m.updateViewport()
    }
```

---

## Debugging Tips

### Capture Raw Events
```go
// Temporary: log all events to file
f, _ := os.OpenFile("/tmp/tui-events.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
defer f.Close()
f.WriteString(fmt.Sprintf("%s: %s\n", event.Type, string(event.Raw)))
```

### Test Permission Flow
```bash
# In gofortress, trigger a Write:
> write a file called test.txt with "hello"

# Watch for user event in logs
tail -f /tmp/tui-events.log
```

---

## Build Commands

```bash
# Build and install
go build -o ~/.local/bin/gofortress ./cmd/gofortress

# Run tests
go test -short ./internal/tui/claude/...
go test -short ./internal/cli/...

# Check coverage
go test -short -cover ./internal/tui/claude/...
```

---

## Related Documentation

- `TUI_GAP.md` - Original streaming fix documentation
- `internal/tui/claude/commands.go` - Native command implementation pattern
- `~/.claude/agents/go-tui/sharp-edges.yaml` - TUI-specific gotchas
