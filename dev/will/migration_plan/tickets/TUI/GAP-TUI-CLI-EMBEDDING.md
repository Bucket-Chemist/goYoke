# GAP Document: TUI-CLI Embedding Architecture

> **GAP ID:** GAP-TUI-002
> **Created:** 2026-01-25
> **Author:** Einstein Analysis
> **Status:** Ready for Planning
> **Depends On:** GAP-TUI-001 (Telemetry Expansion)
> **Priority:** P1 - Core Feature

---

## 1. Executive Summary

**Question:** Can we embed the Claude CLI interface directly into a Bubble Tea TUI, relaying text bidirectionally?

**Answer:** **YES, absolutely feasible.** Claude CLI provides native support for programmatic control via `--input-format stream-json` and `--output-format stream-json` flags. This enables:

1. **Bidirectional streaming** - NDJSON (newline-delimited JSON) in both directions
2. **Real-time character streaming** - `--include-partial-messages` for typewriter effect
3. **Session persistence** - `--session-id` and `--resume` for continuity
4. **Full event visibility** - Hook events, tool calls, assistant messages all structured

**Architecture Pattern:**
```
┌────────────────────────────────────────────────────────────┐
│                    GOgent TUI (Bubble Tea)                 │
├────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────────────────┐   │
│  │  Performance     │  │  Claude Interface Panel      │   │
│  │  Dashboard       │  │  ┌────────────────────────┐  │   │
│  │  (TUI-PERF-*)    │  │  │ [stdin]  User Input   │  │   │
│  │                  │  │  ├────────────────────────┤  │   │
│  │  - Violations    │  │  │ [stdout] Claude Output │  │   │
│  │  - Agents        │  │  │  (streaming NDJSON)   │  │   │
│  │  - Cost          │  │  ├────────────────────────┤  │   │
│  │  - Telemetry     │  │  │ [status] Hook Events  │  │   │
│  │                  │  │  └────────────────────────┘  │   │
│  └──────────────────┘  └──────────────────────────────┘   │
├────────────────────────────────────────────────────────────┤
│            Go Subprocess Manager (exec.Command)            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ claude --print --verbose                              │  │
│  │        --input-format stream-json                     │  │
│  │        --output-format stream-json                    │  │
│  │        --include-partial-messages                     │  │
│  │        --session-id <uuid>                            │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

---

## 2. Technical Feasibility

### 2.1 Claude CLI Stream-JSON Capabilities

**Verified via testing and [official documentation](https://code.claude.com/docs/en/cli-reference):**

| Flag | Purpose | Verified |
|------|---------|----------|
| `--print` | Non-interactive mode for programmatic control | ✅ |
| `--output-format stream-json` | NDJSON output with all events | ✅ |
| `--input-format stream-json` | NDJSON input for messages | ✅ |
| `--include-partial-messages` | Real-time character streaming | ✅ |
| `--verbose` | Full event visibility (hooks, tools) | ✅ |
| `--session-id <uuid>` | Explicit session management | ✅ |
| `--resume <id>` | Continue previous session | ✅ |
| `--settings <json>` | Custom settings override | ✅ |

### 2.2 Output Event Types (stream-json)

From testing with `--verbose --output-format stream-json`:

```jsonl
{"type":"system","subtype":"hook_started","hook_id":"...","hook_name":"SessionStart:startup",...}
{"type":"system","subtype":"hook_response","hook_id":"...","stdout":"...","exit_code":0,...}
{"type":"system","subtype":"init","cwd":"...","session_id":"...","tools":[...],"model":"...",...}
{"type":"assistant","message":{"content":[{"type":"text","text":"..."}],...},...}
{"type":"result","subtype":"success","duration_ms":...,"total_cost_usd":...,"session_id":"...",...}
```

**Event Categories:**

| Event Type | Subtype | Contains |
|------------|---------|----------|
| `system` | `hook_started` | Hook execution begins |
| `system` | `hook_response` | Hook output + exit code |
| `system` | `init` | Session init, tools, model, plugins |
| `assistant` | - | Model response with content blocks |
| `result` | `success`/`error` | Final result, cost, usage |

With `--include-partial-messages`, `assistant` events stream character-by-character.

### 2.3 Input Message Format (stream-json)

Based on [stream chaining documentation](https://github.com/ruvnet/claude-flow/wiki/Stream-Chaining):

```jsonl
{"type":"user","content":"Your message here"}
```

For multi-turn:
```jsonl
{"type":"user","content":"First message"}
{"type":"user","content":"Follow-up message"}
```

The `--replay-user-messages` flag echoes input back on stdout for acknowledgment.

### 2.4 Session Persistence

```bash
# Start with explicit session ID
claude -p --session-id "550e8400-e29b-41d4-a716-446655440000" \
       --input-format stream-json --output-format stream-json

# Resume later
claude --resume "550e8400-e29b-41d4-a716-446655440000" \
       --input-format stream-json --output-format stream-json
```

---

## 3. Implementation Architecture

### 3.1 Core Components

```
internal/
├── cli/
│   ├── subprocess.go      # Claude subprocess lifecycle management
│   ├── streams.go         # NDJSON reader/writer for stdin/stdout
│   ├── events.go          # Event type definitions and parsing
│   └── session.go         # Session ID management and persistence
│
├── tui/
│   ├── main/
│   │   └── model.go       # Root TUI model (layout management)
│   │
│   ├── claude/
│   │   ├── panel.go       # Claude interface panel (center)
│   │   ├── input.go       # User input handling
│   │   ├── output.go      # Streaming output display
│   │   └── events.go      # Event display (hooks, tools)
│   │
│   └── performance/
│       └── (existing TUI-PERF-* views)
```

### 3.2 Subprocess Manager

```go
// internal/cli/subprocess.go
package cli

import (
    "bufio"
    "encoding/json"
    "io"
    "os/exec"
    "sync"
)

type ClaudeProcess struct {
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    stdout    io.ReadCloser
    stderr    io.ReadCloser
    sessionID string
    events    chan Event
    mu        sync.Mutex
}

func NewClaudeProcess(sessionID string, configPath string) (*ClaudeProcess, error) {
    args := []string{
        "--print",
        "--verbose",
        "--input-format", "stream-json",
        "--output-format", "stream-json",
        "--include-partial-messages",
        "--session-id", sessionID,
    }

    if configPath != "" {
        args = append(args, "--settings", configPath)
    }

    cmd := exec.Command("claude", args...)

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    return &ClaudeProcess{
        cmd:       cmd,
        stdin:     stdin,
        stdout:    stdout,
        stderr:    stderr,
        sessionID: sessionID,
        events:    make(chan Event, 100),
    }, nil
}

func (cp *ClaudeProcess) Start() error {
    if err := cp.cmd.Start(); err != nil {
        return err
    }

    // Start goroutines for reading stdout/stderr
    go cp.readEvents()
    go cp.readErrors()

    return nil
}

func (cp *ClaudeProcess) Send(message string) error {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    msg := UserMessage{Type: "user", Content: message}
    data, _ := json.Marshal(msg)
    _, err := cp.stdin.Write(append(data, '\n'))
    return err
}

func (cp *ClaudeProcess) Events() <-chan Event {
    return cp.events
}
```

### 3.3 Event Types

```go
// internal/cli/events.go
package cli

type Event struct {
    Type    string          `json:"type"`
    Subtype string          `json:"subtype,omitempty"`
    Raw     json.RawMessage `json:"-"`
}

type SystemEvent struct {
    Event
    HookID    string `json:"hook_id,omitempty"`
    HookName  string `json:"hook_name,omitempty"`
    CWD       string `json:"cwd,omitempty"`
    SessionID string `json:"session_id"`
    Tools     []string `json:"tools,omitempty"`
    Model     string `json:"model,omitempty"`
}

type AssistantEvent struct {
    Event
    Message   AssistantMessage `json:"message"`
    SessionID string           `json:"session_id"`
}

type AssistantMessage struct {
    Content []ContentBlock `json:"content"`
    Model   string         `json:"model"`
    Usage   Usage          `json:"usage"`
}

type ContentBlock struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}

type ResultEvent struct {
    Event
    IsError       bool    `json:"is_error"`
    DurationMs    int64   `json:"duration_ms"`
    Result        string  `json:"result"`
    SessionID     string  `json:"session_id"`
    TotalCostUSD  float64 `json:"total_cost_usd"`
}

type UserMessage struct {
    Type    string `json:"type"` // "user"
    Content string `json:"content"`
}
```

### 3.4 TUI Claude Panel

```go
// internal/tui/claude/panel.go
package claude

import (
    "strings"

    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

type Model struct {
    process    *cli.ClaudeProcess
    viewport   viewport.Model
    input      textarea.Model
    output     strings.Builder
    events     []cli.Event
    streaming  bool
    sessionID  string
}

func New(sessionID string) Model {
    input := textarea.New()
    input.Placeholder = "Type your message..."
    input.Focus()

    vp := viewport.New(80, 20)

    return Model{
        viewport:  vp,
        input:     input,
        sessionID: sessionID,
    }
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,
        m.startProcess(),
        m.listenForEvents(),
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case cli.Event:
        return m.handleEvent(msg)

    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if !m.streaming {
                return m.sendMessage()
            }
        case "ctrl+c":
            return m, tea.Quit
        }
    }

    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    // Layout: output viewport on top, input at bottom
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.renderOutput(),
        m.renderInput(),
        m.renderStatus(),
    )
}

func (m Model) renderOutput() string {
    style := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(0, 1)

    m.viewport.SetContent(m.output.String())
    return style.Render(m.viewport.View())
}

func (m Model) renderInput() string {
    style := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("240"))

    return style.Render(m.input.View())
}

func (m Model) renderStatus() string {
    status := "Ready"
    if m.streaming {
        status = "⏳ Claude is responding..."
    }

    costStr := ""
    if len(m.events) > 0 {
        // Extract cost from last result event
    }

    return lipgloss.NewStyle().
        Foreground(lipgloss.Color("241")).
        Render(status + costStr)
}

func (m *Model) handleEvent(event cli.Event) (tea.Model, tea.Cmd) {
    m.events = append(m.events, event)

    switch event.Type {
    case "assistant":
        // Stream text to output
        var ae cli.AssistantEvent
        json.Unmarshal(event.Raw, &ae)
        for _, block := range ae.Message.Content {
            if block.Type == "text" {
                m.output.WriteString(block.Text)
            }
        }
        m.streaming = true

    case "result":
        m.streaming = false
        m.output.WriteString("\n---\n")

    case "system":
        if event.Subtype == "hook_started" {
            // Could show hook activity in status bar
        }
    }

    return m, m.listenForEvents()
}
```

---

## 4. TUI Ticket Integration

### New Tickets for CLI Embedding

#### TUI-CLI-01: Claude Subprocess Manager

**Estimated Hours:** 3.0
**Dependencies:** None (can start immediately)
**Priority:** P0 - Foundation

**Description:**
Implement Go subprocess manager for Claude CLI with stream-json I/O.

**Key Features:**
- Start/stop Claude process
- NDJSON event parsing from stdout
- NDJSON message writing to stdin
- Session ID management
- Graceful shutdown with signal handling

**Files:**
- `internal/cli/subprocess.go`
- `internal/cli/streams.go`
- `internal/cli/events.go`
- `internal/cli/subprocess_test.go`

**Acceptance Criteria:**
- [ ] Process starts with correct flags
- [ ] Events parsed from stdout correctly
- [ ] Messages sent via stdin work
- [ ] Session ID preserved across restarts
- [ ] Process cleanup on exit

---

#### TUI-CLI-02: Event Type Definitions

**Estimated Hours:** 1.5
**Dependencies:** TUI-CLI-01
**Priority:** P0 - Foundation

**Description:**
Define Go structs for all Claude CLI stream-json event types.

**Key Features:**
- System events (init, hook_started, hook_response)
- Assistant events (with content blocks)
- Result events (success, error)
- User message format
- Partial message handling

**Files:**
- `internal/cli/events.go`
- `internal/cli/events_test.go`

**Acceptance Criteria:**
- [ ] All event types unmarshal correctly
- [ ] Unknown event types don't panic
- [ ] Partial messages handled
- [ ] Content blocks extracted

---

#### TUI-CLI-03: Claude Interface Panel

**Estimated Hours:** 4.0
**Dependencies:** TUI-CLI-01, TUI-CLI-02, TUI-PERF-01
**Priority:** P1 - Core Feature

**Description:**
Bubble Tea component for Claude CLI interaction in TUI.

**Key Features:**
- Viewport for streaming output
- Textarea for user input
- Real-time text streaming (typewriter effect)
- Hook event sidebar/status
- Cost display in status bar
- Session indicator

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Claude Code - Session: abc123           Cost: $0.23    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  You: Explain this function                            │
│                                                         │
│  Claude: This function implements a binary search...   │
│  [streaming...]                                         │
│                                                         │
│  ─────────────────────────────────────────────────────  │
│  │ Hook: gogent-validate ✅                           │ │
│  │ Tool: Read /src/main.go ✅                         │ │
│  └─────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────┤
│ > Type your message here...                    [Enter] │
└─────────────────────────────────────────────────────────┘
```

**Files:**
- `internal/tui/claude/panel.go`
- `internal/tui/claude/input.go`
- `internal/tui/claude/output.go`
- `internal/tui/claude/events.go`

**Acceptance Criteria:**
- [ ] Text streams character-by-character
- [ ] Input submits on Enter
- [ ] Hook events display in sidebar
- [ ] Cost updates after each response
- [ ] Scroll works in viewport
- [ ] Session ID displayed

---

#### TUI-CLI-04: Main Layout Integration

**Estimated Hours:** 2.0
**Dependencies:** TUI-CLI-03, TUI-PERF-01
**Priority:** P1 - Integration

**Description:**
Integrate Claude panel with performance dashboard in split layout.

**Key Features:**
- Vertical split: Claude (center) | Performance (right)
- Tab navigation between performance views
- Focus switching between panels
- Resize handles (future)
- Keyboard shortcuts for panel switching

**Layout:**
```
┌────────────────────────────────────┬────────────────────┐
│        Claude Interface            │   Performance      │
│                                    │   [1]Violations    │
│  ┌──────────────────────────────┐  │   [2]Agents       │
│  │ You: ...                     │  │   [3]Cost         │
│  │ Claude: ...                  │  │   [4]Collab       │
│  │                              │  ├────────────────────┤
│  │                              │  │ ┌────────────────┐ │
│  │                              │  │ │ Active View   │ │
│  │                              │  │ │ (violations,  │ │
│  │                              │  │ │  agents, etc) │ │
│  └──────────────────────────────┘  │ │               │ │
│  ┌──────────────────────────────┐  │ └────────────────┘ │
│  │ > Input...            [Enter]│  │                    │
│  └──────────────────────────────┘  │                    │
└────────────────────────────────────┴────────────────────┘
```

**Files:**
- `internal/tui/main/model.go`
- `internal/tui/main/layout.go`
- `internal/tui/main/keymap.go`

**Acceptance Criteria:**
- [ ] Split layout renders correctly
- [ ] Tab switches performance views
- [ ] Focus moves between panels
- [ ] Resize terminal updates layout
- [ ] Keyboard shortcuts documented

---

#### TUI-CLI-05: Session Management

**Estimated Hours:** 1.5
**Dependencies:** TUI-CLI-01
**Priority:** P2 - Enhancement

**Description:**
Session picker, history, and continuation support.

**Key Features:**
- Session history display
- Resume previous session
- Fork session option
- Session naming
- Session deletion

**Files:**
- `internal/cli/session.go`
- `internal/tui/session/picker.go`

**Acceptance Criteria:**
- [ ] Sessions listed by recency
- [ ] Resume continues context
- [ ] Fork creates new ID
- [ ] Delete removes session data

---

## 5. Updated Dependency Graph

```
                     ┌─────────────────┐
                     │ TUI-PREREQ-01   │ ← Wire Telemetry
                     │ (BLOCKING)      │
                     └────────┬────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
┌────────▼────────┐  ┌────────▼────────┐  ┌────────▼────────┐
│  TUI-CLI-01     │  │  TUI-PERF-10    │  │  TUI-PERF-01    │
│  Subprocess     │  │  Data Status    │  │  Dashboard      │
│  Manager        │  │  (diagnostic)   │  │  Shell          │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │
┌────────▼────────┐           │           ┌───────┴───────┐
│  TUI-CLI-02     │           │           │               │
│  Event Types    │           │      (TUI-PERF-02..09)    │
└────────┬────────┘           │           │               │
         │                    │           └───────┬───────┘
         │                    │                   │
┌────────▼────────────────────▼───────────────────▼───────┐
│                       TUI-CLI-03                         │
│                  Claude Interface Panel                  │
└────────────────────────────┬────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  TUI-CLI-04     │
                    │  Main Layout    │
                    │  Integration    │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  TUI-CLI-05     │
                    │  Session Mgmt   │
                    └─────────────────┘
```

---

## 6. Effort Summary

### CLI Embedding Tickets

| Ticket | Description | Hours |
|--------|-------------|-------|
| TUI-CLI-01 | Subprocess Manager | 3.0 |
| TUI-CLI-02 | Event Types | 1.5 |
| TUI-CLI-03 | Claude Interface Panel | 4.0 |
| TUI-CLI-04 | Main Layout Integration | 2.0 |
| TUI-CLI-05 | Session Management | 1.5 |
| **Total** | | **12.0** |

### Combined with Telemetry Expansion (GAP-TUI-001)

| Category | Hours |
|----------|-------|
| Telemetry Wiring (PREREQ-01) | 1.0 |
| New Telemetry Views (PERF-08,09,10) | 5.0 |
| Original TUI Views (PERF-01..07) | 11.0 |
| CLI Embedding (CLI-01..05) | 12.0 |
| **Grand Total** | **29.0** |

---

## 7. Technical Considerations

### 7.1 Performance

- **Goroutine per stream**: Stdout and stderr read in separate goroutines
- **Buffered channels**: Event channel buffered to prevent blocking
- **Viewport optimization**: Only render visible lines in viewport
- **Rate limiting**: Throttle UI updates during fast streaming

### 7.2 Error Handling

- **Process crash**: Restart with session continuation
- **Hook failures**: Display in event sidebar, don't block
- **Invalid JSON**: Log and skip malformed events
- **Timeout**: Configurable timeout for responses

### 7.3 Configuration

```go
type CLIConfig struct {
    ClaudePath      string        // Path to claude binary
    SessionID       string        // Explicit session ID
    SettingsPath    string        // Custom settings.json
    Timeout         time.Duration // Response timeout
    MaxEventHistory int           // Events to keep in memory
}
```

### 7.4 Testing Strategy

- **Unit tests**: Event parsing, message formatting
- **Integration tests**: Subprocess lifecycle
- **Mock Claude**: Test binary that emits scripted events
- **TUI tests**: Bubble Tea test utilities

---

## 8. Future Enhancements

### Phase 2 (After MVP)

- **Tool call visualization**: Show tool inputs/outputs in expandable blocks
- **Markdown rendering**: Render Claude's markdown in TUI
- **Code syntax highlighting**: glamour or chroma for code blocks
- **Image handling**: Display image paths, open in viewer
- **Copy to clipboard**: Copy responses or code blocks

### Phase 3 (Advanced)

- **Multi-session tabs**: Multiple Claude sessions in tabs
- **Agent panel**: Visualize subagent spawning
- **Cost budget widget**: Spending limits and alerts
- **Export session**: Save session as markdown

---

## 9. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Claude CLI changes stream format | Breaking | Pin to specific version, version check |
| High memory from long sessions | OOM | Event history limit, viewport windowing |
| Subprocess doesn't exit cleanly | Zombie processes | Signal handling, process group kill |
| Input race conditions | Corrupted messages | Mutex on stdin writes |
| Hook output floods events | UI lag | Filter/throttle hook events |

---

## 10. Sources

- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference)
- [Stream-JSON Chaining Wiki](https://github.com/ruvnet/claude-flow/wiki/Stream-Chaining)
- [Claude Agent SDK Streaming](https://hexdocs.pm/claude_agent_sdk/ClaudeAgentSDK.Streaming.html)
- [Claude Code SSE Stream Processing](https://kotrotsos.medium.com/claude-code-internals-part-7-sse-stream-processing-c620ae9d64a1)

---

## 11. Next Actions

1. **Merge with GAP-TUI-001** - Update TUI-PERF-INDEX.md with all tickets
2. **Create TUI-CLI-01 first** - Subprocess manager is foundation
3. **Build mock Claude binary** - For testing without API calls
4. **Prototype panel layout** - Validate UX before full implementation

---

**Document Version:** 1.0
**GAP Template Version:** 1.0
**Related Documents:**
- `GAP-TUI-TELEMETRY-EXPANSION.md` (telemetry views)
- `TUI-PERF-INDEX.md` (original performance views)
- Claude Code CLI documentation
