# GOgent-Fortress TUI Migration Plan
## React/Ink/TypeScript → Go/Bubble Tea

**Version**: 1.0
**Date**: 2026-03-13
**Architecture Decision**: Three-process topology (Go TUI + TS MCP sidecar + Claude Code CLI)

---

## Table of Contents

1. [Architectural Overview](#1-architectural-overview)
2. [Process Topology & IPC Design](#2-process-topology--ipc-design)
3. [State Model Translation](#3-state-model-translation)
4. [Phase 0 — Scaffolding & Go Module Bootstrap](#4-phase-0--scaffolding--go-module-bootstrap)
5. [Phase 1 — Core TUI Shell (Bubble Tea)](#5-phase-1--core-tui-shell)
6. [Phase 2 — Claude Code CLI Driver (NDJSON Stream Parser)](#6-phase-2--claude-code-cli-driver)
7. [Phase 3 — TS MCP Sidecar Extraction & Bridge API](#7-phase-3--ts-mcp-sidecar-extraction--bridge-api)
8. [Phase 4 — Interactive Modal System](#8-phase-4--interactive-modal-system)
9. [Phase 5 — Agent Tree & Monitoring](#9-phase-5--agent-tree--monitoring)
10. [Phase 6 — Team Orchestration Panel](#10-phase-6--team-orchestration-panel)
11. [Phase 7 — Settings, Telemetry, Plan Mode & Tabs](#11-phase-7--settings-telemetry-plan-mode--tabs)
12. [Phase 8 — Session Persistence & Lifecycle](#12-phase-8--session-persistence--lifecycle)
13. [Phase 9 — Integration Testing & Cutover](#13-phase-9--integration-testing--cutover)
14. [MCP Tool ↔ Event Mapping (TS → Go)](#14-mcp-tool--event-mapping)
15. [Zustand Slice → Go Struct Field Mapping](#15-zustand-slice--go-struct-field-mapping)
16. [Keybinding Inventory](#16-keybinding-inventory)
17. [Risk Register & Design Recommendations](#17-risk-register--design-recommendations)
18. [Dependency Inventory](#18-dependency-inventory)

---

## 1. Architectural Overview

### Current Architecture (Ink/TS — being replaced)

```
┌─────────────────────────────────────────────────┐
│            Ink TUI Process (Node.js)             │
│                                                  │
│  ┌──────────┐  ┌────────────┐  ┌─────────────┐  │
│  │  React   │  │  Zustand   │  │  TS MCP     │  │
│  │  Render  │←→│  Store     │←→│  Server     │  │
│  │  Tree    │  │  (9 slices)│  │  (in-proc)  │  │
│  └──────────┘  └────────────┘  └──────┬──────┘  │
│       ↑                               │         │
│       │ useClaudeQuery                 │         │
│       ↓                               ↓         │
│  ┌──────────────────────────────────────┐        │
│  │  SessionManager (AsyncGenerator)     │        │
│  │  ← query() from @anthropic-ai/      │        │
│  │     claude-agent-sdk                 │        │
│  └──────────────┬───────────────────────┘        │
│                 │ spawns CLI subprocess           │
│                 ↓                                 │
│         Claude Code CLI                          │
└─────────────────────────────────────────────────┘
```

**Problems being solved**: Ink rendering bugs, React reconciler overhead in terminal, Zustand→React bridge race conditions, Node.js dependency for distribution, complex async generator lifecycle management in SessionManager.

### Target Architecture (Go/Bubble Tea)

```
┌──────────────────────────────────────────────────────────┐
│                   Go TUI Process                          │
│                                                           │
│  ┌────────────┐  ┌─────────────┐  ┌───────────────────┐  │
│  │  Bubble Tea│  │  AppModel   │  │  Bridge API       │  │
│  │  Runtime   │←→│  (root      │←→│  :9199/modal/*    │  │
│  │  (Elm arch)│  │   struct)   │  │  :9199/agent/*    │  │
│  └────────────┘  └─────────────┘  └────────┬──────────┘  │
│       ↑                                     │             │
│       │ tea.Cmd / tea.Msg                   │ HTTP        │
│       ↓                                     │             │
│  ┌──────────────────────┐                   │             │
│  │  CLI Driver          │                   │             │
│  │  (NDJSON parser)     │                   │             │
│  │  exec.Command →      │                   │             │
│  │  bufio.Scanner →     │                   │             │
│  │  channel → tea.Cmd   │                   │             │
│  └──────────┬───────────┘                   │             │
│             │ subprocess                     │             │
│             ↓                                ↓             │
│      Claude Code CLI ←──── MCP ────→ TS MCP Sidecar      │
│      --output-format                  (Node.js :9198)     │
│        stream-json                    createSdkMcpServer  │
└──────────────────────────────────────────────────────────┘
```

### Why this topology

| Decision | Rationale |
|----------|-----------|
| Keep TS MCP server | `createSdkMcpServer` is an Anthropic primitive that tightly tracks Claude Code CLI updates. Go MCP SDKs lag behind. |
| Go drives CLI directly | Eliminates the TS Agent SDK's `query()` wrapper. The `--output-format stream-json` NDJSON protocol is stable and well-documented. |
| HTTP bridge (not stdio) | Bubble Tea owns stdin/stdout. MCP sidecar and bridge API both use localhost HTTP. |
| Single Go binary + thin TS sidecar | Go binary is the primary artifact. TS sidecar is a `npx`-runnable package or bundled esbuild artifact. |

---

## 2. Process Topology & IPC Design

### Process Lifecycle

```
Go TUI starts
  ├── Spawns TS MCP sidecar on :9198 (Node.js process)
  │     └── Waits for /health 200 OK
  ├── Starts Bridge API on :9199 (net/http in goroutine)
  ├── Spawns Claude Code CLI subprocess
  │     └── --output-format stream-json
  │     └── --mcp-config pointing to localhost:9198
  │     └── stdin for user messages (line-delimited JSON)
  │     └── stdout for NDJSON event stream
  └── Runs tea.Program (Bubble Tea event loop)
```

### Bridge API Endpoints (Go → TS MCP callbacks)

The Go TUI exposes these HTTP endpoints on `localhost:9199` for the TS MCP sidecar to call when interactive tools fire:

| Endpoint | Method | Purpose | TS MCP caller |
|----------|--------|---------|---------------|
| `POST /modal/ask` | POST | Ask user a question with options | `ask_user`, `canUseTool` |
| `POST /modal/confirm` | POST | Yes/No confirmation | `confirm_action`, `EnterPlanMode` |
| `POST /modal/input` | POST | Free-text input | `request_input`, deny-reason flows |
| `POST /modal/select` | POST | Select from list | `select_option` |
| `POST /agent/register` | POST | Register new agent in tree | `spawn_agent` |
| `POST /agent/update` | POST | Update agent status/activity | `spawn_agent` (on complete/error) |
| `POST /agent/activity` | POST | Live activity update | Process stdout parsing |
| `POST /toast` | POST | Show toast notification | Any MCP tool |
| `GET /health` | GET | Health check | Startup probe |

**Request/Response contract** (modal endpoints):

```
POST /modal/ask
← { "message": "...", "header": "...", "options": [...], "timeout_ms": 60000 }
→ { "type": "ask", "value": "Allow" }  // blocks until user responds
```

The Go handler receives the HTTP request, injects a `ModalRequestMsg` into the Bubble Tea event loop via `Program.Send()`, and blocks the HTTP handler goroutine on a `chan ModalResponse` until the user responds in the TUI. This replaces `useStore.getState().enqueue()`.

### TS MCP Sidecar Changes (Minimal)

The existing MCP tools need a thin adapter layer. Replace direct store calls with HTTP:

```typescript
// BEFORE (current — in-process store access)
const response = await useStore.getState().enqueue({
  type: "ask",
  payload: { message: args.message, options: ... }
});

// AFTER (sidecar — HTTP bridge call)
const response = await fetch(`http://localhost:${BRIDGE_PORT}/modal/ask`, {
  method: "POST",
  body: JSON.stringify({ message: args.message, options: ... })
}).then(r => r.json());
```

Similarly for `spawn_agent`:
```typescript
// BEFORE
useStore.getState().addAgent({ id: toolUseId, ... });

// AFTER
await fetch(`http://localhost:${BRIDGE_PORT}/agent/register`, {
  method: "POST",
  body: JSON.stringify({ id: agentId, parentId, model, tier, ... })
});
```

The `createSdkMcpServer`, tool schemas, and MCP protocol handling remain **completely untouched**.

---

## 3. State Model Translation

### Zustand Store → Go Root Model

The nine Zustand slices collapse into a single Go struct tree. Bubble Tea's Elm architecture means all state lives on the model — no external store needed.

```go
// Root application model — replaces entire Zustand store
type AppModel struct {
    // Terminal dimensions (from tea.WindowSizeMsg)
    width  int
    height int

    // Focus management (replaces UISlice.focusedPanel)
    focus       FocusTarget  // claude | agents | modal
    activeTab   TabID        // chat | agentConfig | teamConfig | telemetry

    // Child models (each implements tea.Model pattern)
    banner      BannerModel
    tabBar      TabBarModel
    claudePanel ClaudePanelModel  // conversation + input
    agentTree   AgentTreeModel    // unified tree
    agentDetail AgentDetailModel  // selected agent info
    dashboard   DashboardModel
    settings    SettingsModel
    planPreview PlanPreviewModel
    taskBoard   TaskBoardModel
    statusLine  StatusLineModel
    toast       ToastModel

    // Modal system (replaces ModalSlice)
    modalQueue  []ModalRequest
    modalActive *ModalModel  // nil when no modal

    // Session state (replaces SessionSlice)
    session     SessionState

    // Agent registry (replaces AgentsSlice)
    agents      AgentRegistry

    // Team state (replaces TeamsSlice)
    teams       TeamRegistry

    // Provider state (replaces multi-provider SessionSlice fields)
    providers   ProviderState

    // Telemetry (replaces TelemetrySlice)
    telemetry   TelemetryState

    // Process handles
    cliDriver   *CLIDriver        // Claude Code subprocess manager
    mcpSidecar  *SidecarProcess   // TS MCP process handle
    bridgeAPI   *BridgeServer     // HTTP callback server
    program     *tea.Program      // for Send() from bridge handlers
}
```

### Key Type Translations

| TS Type | Go Type | Notes |
|---------|---------|-------|
| `Message` (role/content/partial) | `Message` struct with `ContentBlock` union | Use discriminated union via interface |
| `ContentBlock` (text\|tool_use\|tool_result) | `ContentBlock` interface + concrete types | Type switch in rendering |
| `Agent` (V1+V2 fields) | `Agent` struct with optional pointer fields | `*string` for optional fields |
| `AgentActivity` | `AgentActivity` struct | Direct translation |
| `AgentStatus` | `AgentStatus` string enum via `type AgentStatus string` | iota less readable for serialization |
| `ProviderId` | `ProviderID` string type | Same four values |
| `ModalResponse` | `ModalResponse` interface + variants | Channel-based resolution |
| `SessionData` | `SessionData` struct | JSON-compatible with existing .json files |
| `TeamConfig` / `TeamSummary` | Struct translations | Must match Go team format |
| `UnifiedNode` | `UnifiedNode` struct | View-layer projection, computed in View() |

---

## 4. Phase 0 — Scaffolding & Go Module Bootstrap

**Goal**: Empty Go module that compiles and runs a blank Bubble Tea program.

### Tickets

#### P0-1: Initialize Go module and directory structure
```
go mod init github.com/doktersmol/gogent-fortress/tui

Directory layout:
tui/
├── cmd/
│   └── gofortress/
│       └── main.go              # CLI entry point (cobra/pflag)
├── internal/
│   ├── model/                   # Root AppModel + child models
│   │   ├── app.go               # Root model (Init/Update/View)
│   │   ├── focus.go             # Focus state enum + cycling
│   │   └── messages.go          # All custom tea.Msg types
│   ├── components/              # UI component models
│   │   ├── banner/
│   │   ├── tabbar/
│   │   ├── claude/              # ClaudePanel equivalent
│   │   ├── agents/              # AgentTree + AgentDetail
│   │   ├── modals/              # Modal system
│   │   ├── statusline/
│   │   ├── taskboard/
│   │   ├── toast/
│   │   ├── dashboard/
│   │   ├── settings/
│   │   ├── telemetry/
│   │   └── planpreview/
│   ├── cli/                     # Claude Code CLI driver
│   │   ├── driver.go            # Subprocess management
│   │   ├── parser.go            # NDJSON stream parser
│   │   ├── events.go            # SDK event type definitions
│   │   └── driver_test.go
│   ├── bridge/                  # HTTP bridge API
│   │   ├── server.go            # net/http server
│   │   ├── handlers.go          # Modal + agent endpoints
│   │   └── types.go             # Request/response types
│   ├── sidecar/                 # TS MCP sidecar launcher
│   │   ├── launcher.go          # Process spawn + health wait
│   │   └── launcher_test.go
│   ├── session/                 # Session persistence
│   │   ├── persistence.go       # Load/save session JSON
│   │   └── history.go           # Conversation history
│   ├── state/                   # Shared state types
│   │   ├── agent.go             # Agent, AgentActivity types
│   │   ├── message.go           # Message, ContentBlock types
│   │   ├── session.go           # SessionData, TokenCount
│   │   ├── team.go              # TeamConfig, TeamSummary
│   │   ├── provider.go          # ProviderID, ProviderState
│   │   └── modal.go             # ModalRequest, ModalResponse
│   ├── config/                  # Theme, keybindings
│   │   ├── theme.go             # Colors, borders, icons
│   │   └── keys.go              # Key bindings registry
│   └── util/                    # Utilities
│       ├── markdown.go          # Glamour markdown renderer
│       ├── logger.go            # Structured logging
│       └── truncate.go          # String helpers
├── sidecar/                     # TS MCP sidecar package
│   ├── package.json
│   ├── src/
│   │   ├── index.ts             # Entry point — HTTP MCP server
│   │   ├── bridge.ts            # HTTP client for Go bridge API
│   │   └── tools/               # Migrated tool implementations
│   └── tsconfig.json
├── go.mod
├── go.sum
└── Makefile
```

#### P0-2: Add core dependencies to go.mod
```
charm.land/bubbletea/v2          # or github.com/charmbracelet/bubbletea (pin to v1 if v2 too fresh)
github.com/charmbracelet/bubbles # viewport, textinput, spinner, list, help, key
github.com/charmbracelet/lipgloss
github.com/charmbracelet/glamour # Markdown rendering
github.com/charmbracelet/log     # Structured logging
github.com/spf13/cobra           # CLI flags (--session-id, --verbose)
```

**Design recommendation**: Pin to **Bubble Tea v1** (`github.com/charmbracelet/bubbletea` v1.x) initially. v2 shipped Feb 2026 and the new `tea.View` struct API is still stabilizing. v1's string-based `View()` is battle-tested. Migrate to v2 later when the ecosystem (Bubbles, Lip Gloss) has caught up.

#### P0-3: Minimal main.go + empty AppModel
Create `cmd/gofortress/main.go` with Cobra CLI parsing and a blank Bubble Tea program that renders "GOgent-Fortress" in a bordered box. Validates the toolchain works end-to-end.

---

## 5. Phase 1 — Core TUI Shell

**Goal**: Multi-panel layout with focus cycling, responsive sizing, and themed borders. No data — just the chrome.

### Tickets

#### P1-1: Theme system (`internal/config/theme.go`)
Translate `src/config/theme.ts` to Go Lip Gloss styles:
```go
var (
    ColorPrimary   = lipgloss.Color("6")    // cyan
    ColorSecondary = lipgloss.Color("4")    // blue
    ColorAccent    = lipgloss.Color("5")    // magenta
    ColorSuccess   = lipgloss.Color("2")    // green
    ColorWarning   = lipgloss.Color("3")    // yellow
    ColorError     = lipgloss.Color("1")    // red
    ColorMuted     = lipgloss.Color("8")    // gray

    StyleFocusedBorder = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorPrimary)
    StyleUnfocusedBorder = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(ColorMuted)
)
```
Map all `colors`, `borders`, `icons` constants from `theme.ts`.

**Design recommendation**: Use ANSI color numbers (0-15) rather than hex codes for the base theme. This respects the user's terminal colorscheme. Reserve hex for accent colors that must be stable.

#### P1-2: Focus management (`internal/model/focus.go`)
```go
type FocusTarget int
const (
    FocusClaude FocusTarget = iota
    FocusAgents
)
```
Tab key cycles focus. Store `focus` on root `AppModel`. Route `tea.KeyMsg` only to the focused child in `Update()`. Visual indication via swapping `StyleFocusedBorder` / `StyleUnfocusedBorder`.

Maps to: `UISlice.focusedPanel`, `setFocusedPanel()`

#### P1-3: Banner component (`internal/components/banner/`)
Static render of "GOgent-Fortress" with rounded border. Fixed height 3 rows. Direct translation of `Banner.tsx`.

#### P1-4: Tab bar component (`internal/components/tabbar/`)
Horizontal tab strip: Chat | Agent Config | Team Config | Telemetry. Alt+C/A/T/Y shortcuts. Active tab highlighted with underline or inverse. Fixed height 1 row.

Maps to: `UISlice.activeTab`, `TabDefinition[]`, `TabBar.tsx`, `TabBar.test.tsx`

#### P1-5: Root layout compositor (`internal/model/app.go` View method)
Implement the `View()` method using Lip Gloss layout:
```
┌── Banner (3 rows, full width) ──────────────────┐
├── TabBar (1 row, full width) ───────────────────┤
├── Left Panel (70%) ──┬── Right Panel (30%) ─────┤
│  Claude Panel        │  Agent Tree (60%)         │
│                      ├───────────────────────────│
│                      │  Agent Detail (40%)       │
├──────────────────────┴───────────────────────────┤
├── Task Board (8 rows) ──────────────────────────┤
├── Status Line (2 rows) ─────────────────────────┤
└─────────────────────────────────────────────────┘
```

Responsive breakpoints from `Layout.tsx`:
- `width < 80`: Hide right panel entirely (single column)
- `width < 100`: Left 75%, Right 25%
- `width >= 100`: Left 70%, Right 30%

Handle `tea.WindowSizeMsg` to store dimensions and propagate to all children.

Maps to: `Layout.tsx` (entire component)

#### P1-6: Status line component (`internal/components/statusline/`)
Two-row status bar at bottom. Display: session cost, token usage, context window %, permission mode, active model, provider, git branch, auth status.

Maps to: `StatusLine.tsx` (~330 lines). 

**Design recommendation**: Replace the `execSync("claude auth status --json")` and `execSync("git ...")` polling with `tea.Every(30*time.Second, ...)` ticking commands that run `exec.Command` in the background. Never block `Update()` or `View()` with subprocess calls.

#### P1-7: Keybinding registry (`internal/config/keys.go`)
Define all key bindings using `bubbles/key`:
```go
type KeyMap struct {
    ToggleFocus    key.Binding
    CycleRightPanel key.Binding
    CyclePermMode   key.Binding
    Interrupt       key.Binding
    ForceQuit       key.Binding
    ClearScreen     key.Binding
    ToggleTaskBoard key.Binding
    // Tab shortcuts
    TabChat         key.Binding
    TabAgentConfig  key.Binding
    TabTeamConfig   key.Binding
    TabTelemetry    key.Binding
    // Claude panel
    Submit          key.Binding
    HistoryPrev     key.Binding
    HistoryNext     key.Binding
    // Agent panel
    AgentUp         key.Binding
    AgentDown       key.Binding
}
```

Maps to: `keybindings.ts` (entire file — see Section 16 for full inventory)

---

## 6. Phase 2 — Claude Code CLI Driver

**Goal**: Spawn Claude Code CLI as subprocess, parse NDJSON events, feed into Bubble Tea event loop.

This replaces: `SessionManager.ts` (~700 lines), `useClaudeQuery.ts` (~180 lines), `session/types.ts`.

### Tickets

#### P2-1: NDJSON event type definitions (`internal/cli/events.go`)
Define Go structs for every event type emitted by `claude --output-format stream-json`:

```go
// Top-level discriminator
type CLIEvent struct {
    Type    string          `json:"type"`    // system | assistant | user | result | stream_event
    Subtype string          `json:"subtype"` // init | status | compact_boundary (for system)
    Raw     json.RawMessage `json:"-"`       // preserve for full parsing
}

// system.init
type SystemInitEvent struct {
    SessionID string `json:"session_id"`
    Model     string `json:"model"`
    Tools     []struct {
        Name string `json:"name"`
    } `json:"tools"`
}

// system.status
type SystemStatusEvent struct {
    Status         *string `json:"status"`          // "compacting" | null
    PermissionMode string  `json:"permissionMode"`
}

// system.compact_boundary
type CompactBoundaryEvent struct {
    CompactMetadata struct {
        Trigger   string `json:"trigger"`    // manual | auto
        PreTokens int    `json:"pre_tokens"`
    } `json:"compact_metadata"`
}

// assistant — maps to SDKAssistantMessage
type AssistantEvent struct {
    Message struct {
        ID      string         `json:"id"`
        Content []ContentBlock `json:"content"`
        Usage   *MessageUsage  `json:"usage"`
    } `json:"message"`
    ParentToolUseID *string `json:"parent_tool_use_id"`
}

// result — maps to SDKResultMessage
type ResultEvent struct {
    Subtype      string  `json:"subtype"`       // success | error
    TotalCostUSD float64 `json:"total_cost_usd"`
    Duration     int     `json:"duration_ms"`
    SessionID    string  `json:"session_id"`
    Usage        *struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage"`
    ModelUsage map[string]struct {
        ContextWindow int `json:"contextWindow"`
    } `json:"modelUsage"`
    Errors []string `json:"errors"`
}
```

Maps to: `types/events.ts` (SystemEvent, AssistantEvent, ResultEvent, StatusEvent)

#### P2-2: CLI subprocess driver (`internal/cli/driver.go`)
```go
type CLIDriver struct {
    cmd      *exec.Cmd
    stdin    io.WriteCloser      // send user messages
    stdout   *bufio.Scanner      // read NDJSON events
    eventCh  chan tea.Msg         // feed into Bubble Tea
    state    DriverState          // IDLE | STREAMING | ERROR | DEAD
    session  string               // current session ID
}

type DriverState int
const (
    DriverIdle DriverState = iota
    DriverStreaming
    DriverError
    DriverDead
)

func NewCLIDriver(opts CLIDriverOpts) *CLIDriver { ... }
func (d *CLIDriver) Start(ctx context.Context) tea.Cmd { ... }
func (d *CLIDriver) SendMessage(text string) tea.Cmd { ... }
func (d *CLIDriver) Interrupt() tea.Cmd { ... }
func (d *CLIDriver) SetModel(modelID string) tea.Cmd { ... }
func (d *CLIDriver) Shutdown() tea.Cmd { ... }
```

Replaces: `SessionManager` class (entire file — state machine, message generator, event consumption, error classification, reconnection).

**Key architectural difference**: The current `SessionManager` uses an `AsyncGenerator` pattern with the Agent SDK's `query()`. The Go driver is simpler — it's a subprocess with stdin/stdout pipes. User messages go to stdin as line-delimited JSON. Events come from stdout as NDJSON. No generator coordination needed.

#### P2-3: NDJSON stream parser (`internal/cli/parser.go`)
Goroutine that runs `bufio.Scanner` on CLI stdout, JSON-unmarshals each line, and sends typed messages through the event channel:

```go
func (d *CLIDriver) consumeEvents() {
    for d.stdout.Scan() {
        line := d.stdout.Bytes()
        var base CLIEvent
        json.Unmarshal(line, &base)
        
        switch base.Type {
        case "system":
            switch base.Subtype {
            case "init":
                var evt SystemInitEvent
                json.Unmarshal(line, &evt)
                d.eventCh <- SystemInitMsg(evt)
            case "status":
                // ...
            case "compact_boundary":
                // ...
            }
        case "assistant":
            var evt AssistantEvent
            json.Unmarshal(line, &evt)
            d.eventCh <- AssistantMsg(evt)
        case "result":
            var evt ResultEvent
            json.Unmarshal(line, &evt)
            d.eventCh <- ResultMsg(evt)
        }
    }
}
```

The channel-to-Cmd bridge uses the standard re-subscription pattern:
```go
func waitForCLIEvent(ch <-chan tea.Msg) tea.Cmd {
    return func() tea.Msg {
        return <-ch
    }
}
```

Maps to: `SessionManager.consumeEvents()`, `handleSystemEvent()`, `handleAssistantEvent()`, `handleUserEvent()`, `handleResultEvent()`, `handleStatusEvent()`, `handleCompactBoundaryEvent()`

#### P2-4: CLI spawn configuration
```go
type CLIDriverOpts struct {
    SessionID    string            // --resume flag
    Model        string            // --model flag  
    MCPConfig    string            // --mcp-config path (points to sidecar)
    ProjectDir   string            // --cwd flag
    Verbose      bool              // --verbose
    PermMode     string            // initial permission mode
}
```

Build the `exec.Command`:
```go
args := []string{
    "claude",
    "--output-format", "stream-json",
    "--verbose",
}
if opts.SessionID != "" {
    args = append(args, "--resume", opts.SessionID)
}
if opts.Model != "" {
    args = append(args, "--model", opts.Model)
}
if opts.MCPConfig != "" {
    args = append(args, "--mcp-config", opts.MCPConfig)
}
```

#### P2-5: Error classification and reconnection
Translate `SessionManager.classifyError()` logic. Implement retry with backoff (max 3 attempts). State transitions: IDLE → STREAMING → IDLE (on result) or ERROR → reconnect → IDLE.

Maps to: `classifyError()`, `attemptReconnect()`, `VALID_TRANSITIONS` map

#### P2-6: User message delivery
Write user messages to CLI stdin as JSON:
```go
func (d *CLIDriver) SendMessage(text string) tea.Cmd {
    return func() tea.Msg {
        msg := map[string]string{"type": "user", "text": text}
        data, _ := json.Marshal(msg)
        d.stdin.Write(append(data, '\n'))
        return MessageSentMsg{}
    }
}
```

Maps to: `SessionManager.enqueue()`, `createMessageGenerator()` yield

---

## 7. Phase 3 — TS MCP Sidecar Extraction & Bridge API

**Goal**: Extract existing MCP tools into a standalone HTTP sidecar. Build the Go HTTP bridge.

### Tickets

#### P3-1: Create sidecar package (`sidecar/`)
New package.json with minimal dependencies:
```json
{
  "name": "@gofortress/mcp-sidecar",
  "dependencies": {
    "@anthropic-ai/claude-agent-sdk": "^0.2.66",
    "zod": "^4.3.6"
  }
}
```

Copy from current TUI:
- `mcp/server.ts` → `sidecar/src/server.ts`
- `mcp/tools/*.ts` → `sidecar/src/tools/`
- `spawn/` directory → `sidecar/src/spawn/`
- `cost/tracker.ts` → `sidecar/src/cost/tracker.ts`

#### P3-2: Bridge client module (`sidecar/src/bridge.ts`)
HTTP client that replaces all `useStore.getState()` calls:

```typescript
const BRIDGE_URL = process.env.BRIDGE_URL || "http://localhost:9199";

export async function bridgeModalAsk(payload: AskPayload): Promise<ModalResponse> {
    const res = await fetch(`${BRIDGE_URL}/modal/ask`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
    return res.json();
}

export async function bridgeAgentRegister(agent: AgentData): Promise<void> {
    await fetch(`${BRIDGE_URL}/agent/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(agent),
    });
}

export async function bridgeAgentUpdate(id: string, data: Partial<AgentData>): Promise<void> {
    await fetch(`${BRIDGE_URL}/agent/update`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, ...data }),
    });
}

export async function bridgeToast(message: string, type: string): Promise<void> {
    await fetch(`${BRIDGE_URL}/toast`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message, type }),
    });
}
```

#### P3-3: Refactor each MCP tool to use bridge client
For each tool, replace direct store access:

| Tool | Current store calls | Bridge replacement |
|------|--------------------|--------------------|
| `askUser` | `useStore.getState().enqueue({type: "ask"})` | `bridgeModalAsk()` |
| `confirmAction` | `useStore.getState().enqueue({type: "confirm"})` | `bridgeModalConfirm()` |
| `requestInput` | `useStore.getState().enqueue({type: "input"})` | `bridgeModalInput()` |
| `selectOption` | `useStore.getState().enqueue({type: "select"})` | `bridgeModalSelect()` |
| `spawnAgent` | `useStore.getState().addAgent()`, `getProcessRegistry()`, `getAgentsStore()` | `bridgeAgentRegister()`, keep process registry local to sidecar |
| `teamRun` | `useStore.getState()` for team state | `bridgeAgentRegister()` per team member |

**Important**: `spawnAgent`'s process spawning (`exec.spawn`) stays in the TS sidecar. Only state notification crosses the bridge. Process lifecycle (PID tracking, timeout, output buffering) remains in `sidecar/src/spawn/`.

#### P3-4: Sidecar HTTP entry point (`sidecar/src/index.ts`)
```typescript
import { createMcpServer } from "./server.js";
import http from "http";

const mcpServer = createMcpServer();
// Expose via HTTP transport for Claude Code CLI
const httpServer = http.createServer(/* MCP HTTP handler */);
httpServer.listen(parseInt(process.env.MCP_PORT || "9198"), "127.0.0.1");
```

The `createSdkMcpServer` needs to be configured for HTTP transport rather than in-process. Check the SDK docs for `createStreamableHTTPServer()` or equivalent.

#### P3-5: Go sidecar launcher (`internal/sidecar/launcher.go`)
```go
type SidecarProcess struct {
    cmd  *exec.Cmd
    port int
}

func LaunchSidecar(port int, bridgePort int) (*SidecarProcess, error) {
    cmd := exec.Command("node", "path/to/sidecar/dist/index.js")
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("MCP_PORT=%d", port),
        fmt.Sprintf("BRIDGE_URL=http://localhost:%d", bridgePort),
    )
    cmd.Start()
    // Poll /health until 200 or timeout
    waitForHealth(fmt.Sprintf("http://localhost:%d/health", port), 10*time.Second)
    return &SidecarProcess{cmd: cmd, port: port}, nil
}
```

#### P3-6: Go bridge API server (`internal/bridge/server.go`)
```go
type BridgeServer struct {
    program *tea.Program  // for Send()
    port    int
    server  *http.Server
}

func (b *BridgeServer) handleModalAsk(w http.ResponseWriter, r *http.Request) {
    var req ModalAskRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Create response channel
    respCh := make(chan ModalResponse, 1)
    
    // Inject into Bubble Tea event loop
    b.program.Send(ModalRequestMsg{
        Type:       "ask",
        Payload:    req,
        ResponseCh: respCh,
    })
    
    // Block until TUI user responds (or timeout)
    select {
    case resp := <-respCh:
        json.NewEncoder(w).Encode(resp)
    case <-time.After(time.Duration(req.TimeoutMs) * time.Millisecond):
        json.NewEncoder(w).Encode(ModalResponse{Type: "ask", Value: "timeout"})
    }
}
```

Maps to: `useStore.getState().enqueue()` → `Promise<ModalResponse>` flow

---

## 8. Phase 4 — Interactive Modal System

**Goal**: Full modal queue with ask/confirm/input/select variants, keyboard handling, and the permission flow.

This replaces: `Modal.tsx` (~190 lines), `modals/AskModal.tsx` (~240 lines), `modals/ConfirmModal.tsx`, `modals/InputModal.tsx`, `modals/SelectModal.tsx`, `store/slices/modal.ts`.

### Tickets

#### P4-1: Modal model types (`internal/state/modal.go`)
```go
type ModalRequest struct {
    ID         string
    Type       ModalType  // ask | confirm | input | select
    Payload    interface{}
    ResponseCh chan<- ModalResponse
    TimeoutMs  int
}

type ModalType string
const (
    ModalAsk     ModalType = "ask"
    ModalConfirm ModalType = "confirm"
    ModalInput   ModalType = "input"
    ModalSelect  ModalType = "select"
)

type AskPayload struct {
    Message string       `json:"message"`
    Header  string       `json:"header"`
    Options []AskOption  `json:"options"`
}

type AskOption struct {
    Label       string `json:"label"`
    Value       string `json:"value"`
    Description string `json:"description"`
}
```

#### P4-2: Modal overlay model (`internal/components/modals/modal.go`)
```go
type ModalModel struct {
    request    ModalRequest
    selected   int           // current option index
    textInput  textinput.Model  // for input/free-text modes
    isTextMode bool           // toggled by "Other" option
    width      int
    height     int
}
```

Key behaviors from existing `AskModal.tsx`:
- Options rendered as selectable list (Up/Down to navigate, Enter to select)
- "Other" option always appended (switches to free-text input)
- Enter on first option (Allow) is the fast-path for permissions
- Escape cancels (sends cancel response)
- Compact rendering mode for permission prompts (lives at bottom of left panel, not full-screen overlay)

#### P4-3: Modal rendering with Lip Gloss overlay
```go
func (m ModalModel) View() string {
    // Double-border box centered on screen
    content := m.renderContent()
    box := lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        BorderForeground(theme.ColorPrimary).
        Padding(1, 2).
        MaxWidth(m.width - 4).
        Render(content)
    
    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center, box)
}
```

For compact mode (permissions during streaming), render as a strip at the bottom of the left panel — not full overlay. This matches current `Layout.tsx` behavior.

#### P4-4: Modal queue management in root model
```go
// In AppModel.Update()
case ModalRequestMsg:
    m.modalQueue = append(m.modalQueue, msg.Request)
    if m.modalActive == nil {
        m.modalActive = NewModalModel(m.modalQueue[0], m.width, m.height)
        m.focus = FocusModal
    }
    return m, nil

case ModalResponseMsg:
    // Send response through channel
    if m.modalActive != nil {
        m.modalActive.request.ResponseCh <- msg.Response
        m.modalQueue = m.modalQueue[1:]
        if len(m.modalQueue) > 0 {
            m.modalActive = NewModalModel(m.modalQueue[0], m.width, m.height)
        } else {
            m.modalActive = nil
            m.focus = FocusClaude // restore focus
        }
    }
```

Maps to: `ModalSlice.enqueue()`, `ModalSlice.dequeue()`, `ModalSlice.cancel()`

#### P4-5: canUseTool permission flow
The `canUseTool` callback is the most complex modal flow. Currently in `SessionManager.handleCanUseTool()`. In the new architecture, this is handled entirely in the TS MCP sidecar (it calls `bridgeModalAsk()` for the permission prompt), but the Go TUI needs to handle the permission-specific UI:

- `EnterPlanMode`: Confirm modal → allow/deny
- `ExitPlanMode`: Ask modal with Approve/Request changes/Reject → optional follow-up input modal for feedback → load plan preview into right panel
- `AskUserQuestion`: Sequential ask modals (1-4 questions)
- Standard tools: Ask modal with Allow/Deny → optional input modal for deny reason
- `acceptEdits` mode: Auto-approve (sidecar handles, no bridge call needed)

These flows remain in the TS sidecar's `canUseTool` handler, which makes sequential `bridgeModal*()` calls. The Go TUI just processes individual modal requests — it doesn't need to know about the multi-step flows.

---

## 9. Phase 5 — Agent Tree & Monitoring

**Goal**: Unified agent/team tree with live activity monitoring.

Replaces: `UnifiedTree.tsx` (~270 lines), `UnifiedDetail.tsx` (~310 lines), `AgentTree.tsx`, `AgentDetail.tsx`, `hooks/useAgentSync.ts` (~200 lines), `hooks/useUnifiedTree.ts`, `hooks/useAgentTree.ts`, `utils/agentActivity.ts` (~250 lines).

### Tickets

#### P5-1: Agent registry (`internal/state/agent.go`)
```go
type AgentRegistry struct {
    agents       map[string]*Agent
    rootAgentID  string
    selectedID   string
}

type Agent struct {
    ID          string
    ParentID    string
    Model       string
    Tier        AgentTier  // haiku | sonnet | opus
    Status      AgentStatus
    Description string
    AgentType   string
    SpawnMethod string     // task | mcp-cli
    StartTime   time.Time
    EndTime     *time.Time
    
    // Process info (MCP-CLI spawns)
    PID         *int
    
    // Metrics
    Cost        float64
    Turns       int
    ToolCalls   int
    TokenUsage  *TokenCount
    
    // Live activity
    Activity    *AgentActivity
    
    // Output
    Output      string
    Error       string
    
    // Hierarchy
    ChildIDs    []string
    Depth       int
}
```

#### P5-2: Agent sync from NDJSON events
In the current TUI, `useAgentSync` scans messages for `Task()` tool_use blocks. In Go, we extract this from the NDJSON stream directly:

```go
// In AppModel.Update() handling AssistantMsg
case AssistantMsg:
    for _, block := range msg.Content {
        if block.Type == "tool_use" && (block.Name == "Agent" || block.Name == "Task") {
            m.agents.RegisterFromToolUse(block)
        }
    }
```

Agent completion detection from tool_result blocks:
```go
// In handling UserMsg (tool results)
case UserMsg:
    for _, block := range msg.Content {
        if block.Type == "tool_result" {
            if agent, ok := m.agents.ByToolUseID(block.ToolUseID); ok {
                status := "complete"
                if block.IsError { status = "error" }
                m.agents.Update(agent.ID, AgentUpdate{Status: status})
            }
        }
    }
```

Maps to: `useAgentSync.ts` `syncTaskAgents()`, `syncActivityFromMessages()`

#### P5-3: Agent activity extraction
Translate `utils/agentActivity.ts` `activityFromTaskBlocks()` and `extractToolTarget()`:
```go
func ExtractActivity(blocks []ContentBlock) *AgentActivity {
    var activity AgentActivity
    for i := len(blocks) - 1; i >= 0; i-- {
        switch b := blocks[i].(type) {
        case TextBlock:
            if activity.LastText == "" {
                activity.LastText = truncate(b.Text, 120)
            }
        case ToolUseBlock:
            if activity.CurrentTool == nil {
                activity.CurrentTool = &ToolInfo{
                    Name:   b.Name,
                    Target: extractToolTarget(b.Name, b.Input),
                }
            }
        }
    }
    return &activity
}
```

#### P5-4: Unified tree model (`internal/components/agents/tree.go`)
Tree rendering with indentation, status icons, and activity preview:
```
▶ Router (opus) ● running
  ├─ backend-reviewer (sonnet) ● Read src/main.go
  ├─ einstein (opus) ◐ spawning
  └─ refactorer (sonnet) ✓ complete (0.12s)
```

Uses: `icons.treeIndent`, `icons.treeBranch`, `icons.treeLeaf` from theme.
Up/Down navigation when agents panel focused.

Maps to: `UnifiedTree.tsx`, `useUnifiedTree.ts` (node projection logic)

#### P5-5: Agent detail model (`internal/components/agents/detail.go`)
Shows selected agent's details in a viewport:
- Status, model, tier, spawn method
- Duration, cost, token usage
- Live activity (current tool, last text)
- Error output if failed

Maps to: `UnifiedDetail.tsx`

#### P5-6: Bridge handlers for agent registration
Handle `POST /agent/register` and `POST /agent/update` from sidecar:
```go
func (b *BridgeServer) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
    var req AgentRegisterRequest
    json.NewDecoder(r.Body).Decode(&req)
    b.program.Send(AgentRegisteredMsg(req))
    w.WriteHeader(200)
}
```

---

## 10. Phase 6 — Team Orchestration Panel

**Goal**: Team polling, team list, team detail with wave/member status.

Replaces: `TeamList.tsx`, `TeamDetail.tsx`, `hooks/useTeams.ts`, `hooks/useTeamCount.ts`, `mcp/tools/teamRun.ts` (sidecar retains this), `utils/teamFormatting.ts`.

### Tickets

#### P6-1: Team registry and types (`internal/state/team.go`)
Direct translation of `TeamConfig`, `TeamSummary`, `TeamMemberRow`, `TeamWave`, `TeamMember` types.

#### P6-2: Team polling via tea.Every
Replace `useTeamsPoller` (chokidar file watcher) with periodic polling:
```go
func pollTeams(sessionDir string) tea.Cmd {
    return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
        teams := scanTeamDirs(sessionDir)
        return TeamsUpdatedMsg{Teams: teams}
    })
}
```

Scans `$GOGENT_SESSION_DIR/teams/*/config.json` same as current `useTeams.ts`.

#### P6-3: Team list component
Renders list of active/completed teams with status indicators.

#### P6-4: Team detail component
Wave-by-wave member table with status, cost, duration per member.

---

## 11. Phase 7 — Settings, Telemetry, Plan Mode & Tabs

### Tickets

#### P7-1: Right panel mode cycling
Cycle agents → dashboard → settings via Alt+R. Maps to: `UISlice.rightPanelMode`, `cycleRightPanel()`

#### P7-2: Dashboard view
Session stats summary. Maps to: `DashboardView.tsx`

#### P7-3: Settings view
Display current configuration. Maps to: `SettingsView.tsx`

#### P7-4: Telemetry view
Routing decisions, handoffs, sharp edges display. Maps to: `TelemetryView.tsx`, `store/slices/telemetry.ts`

#### P7-5: Plan preview panel
When `ExitPlanMode` fires, the sidecar loads the plan .md file and sends it to the bridge API. Go TUI renders it in the right panel using Glamour for markdown.

Maps to: `PlanPreview.tsx`, the `ExitPlanMode` handler in `SessionManager.handleCanUseTool()`

#### P7-6: Task board component
Compact strip showing active/completed tasks. Tab toggle via Alt+B.

Maps to: `TaskBoard.tsx` (~200 lines)

#### P7-7: Provider tabs and switching
Provider tab strip (Anthropic | Google | OpenAI | Local) with Shift+Tab cycling. Per-provider message history, session ID, and model.

Maps to: `ProviderTabs.tsx`, `config/providers.ts`, per-provider state in `SessionSlice`

---

## 12. Phase 8 — Session Persistence & Lifecycle

**Goal**: Session save/load, conversation history persistence, graceful shutdown.

Replaces: `hooks/useSession.ts`, `lifecycle/shutdown.ts`, `lifecycle/restart.ts`, `lifecycle/index.ts`.

### Tickets

#### P8-1: Session persistence (`internal/session/persistence.go`)
Load/save `~/.claude/sessions/{id}/session.json`:
```go
type SessionData struct {
    ID        string  `json:"id"`
    Name      string  `json:"name,omitempty"`
    CreatedAt string  `json:"created_at"`
    LastUsed  string  `json:"last_used"`
    Cost      float64 `json:"cost"`
    ToolCalls int     `json:"tool_calls"`
}

func LoadSession(id string) (*SessionData, error) { ... }
func SaveSession(data *SessionData) error { ... }
```

#### P8-2: Conversation history
Load/save per-provider message history for session resume.

Maps to: `loadConversationHistory()` in `useSession.ts`

#### P8-3: Graceful shutdown
Handle SIGINT, SIGTERM:
1. Save session state
2. Interrupt active CLI query
3. Shutdown CLI subprocess
4. Shutdown MCP sidecar
5. Wait for Go hooks (gogent-archive)
6. Exit

Maps to: `lifecycle/shutdown.ts` `initiateShutdown()`

#### P8-4: Session directory management
Setup `GOGENT_SESSION_DIR`, write `.claude/current-session` marker, create `.claude/tmp` symlink.

Maps to: `App.tsx` `setSessionDir()`, `setupSessionFiles()`

#### P8-5: Auto-save on cost changes
Debounced session save when cost increments (from ResultMsg events).

Maps to: `App.tsx` useEffect that watches `totalCost`

---

## 13. Phase 9 — Integration Testing & Cutover

### Tickets

#### P9-1: Component unit tests
Test each component model's `Update()` and `View()` in isolation using `teatest` or direct function calls.

#### P9-2: CLI driver integration test
Mock Claude Code CLI subprocess with a script that emits known NDJSON events. Verify correct message parsing and state transitions.

#### P9-3: Bridge API integration test  
Start bridge server, POST modal requests, verify Bubble Tea model receives correct messages.

#### P9-4: Sidecar integration test
Start full sidecar + bridge, trigger MCP tools, verify modal round-trip.

#### P9-5: End-to-end smoke test
Full startup: Go TUI → sidecar → CLI → send message → receive response → display.

#### P9-6: Feature parity checklist
Verify all features work:
- [ ] Session resume (--session-id flag)
- [ ] Multi-provider switching
- [ ] Permission prompts (allow/deny with reason)
- [ ] Plan mode (enter → write plan → exit → approve/reject)
- [ ] Agent spawning (Task + MCP spawn_agent)
- [ ] Team orchestration (teamRun + polling)
- [ ] Context compaction notification
- [ ] Model switching
- [ ] Permission mode cycling
- [ ] Input history (up/down)
- [ ] Slash commands
- [ ] Markdown rendering
- [ ] Responsive layout (narrow/very narrow)
- [ ] Graceful shutdown
- [ ] Toast notifications

#### P9-7: Remove Ink TUI package
Once Go TUI achieves feature parity, remove `packages/tui/` from the monorepo.

---

## 14. MCP Tool ↔ Event Mapping

Complete mapping of every MCP tool to its bridge API interaction:

| MCP Tool | Current TS Store Call | Go Bridge Endpoint | Response Flow |
|----------|----------------------|-------------------|---------------|
| `ask_user` | `enqueue({type:"ask"})` | `POST /modal/ask` | Blocks → user selects option → JSON response |
| `confirm_action` | `enqueue({type:"confirm"})` | `POST /modal/confirm` | Blocks → user confirms/cancels → JSON response |
| `request_input` | `enqueue({type:"input"})` | `POST /modal/input` | Blocks → user types text → JSON response |
| `select_option` | `enqueue({type:"select"})` | `POST /modal/select` | Blocks → user picks item → JSON response |
| `spawn_agent` | `addAgent()`, `updateAgent()`, `getProcessRegistry()` | `POST /agent/register`, `POST /agent/update` | Fire-and-forget (agent state notification) |
| `teamRun` | Team state updates | `POST /agent/register` (per member) | Fire-and-forget |
| `testMcpPing` | None (returns static response) | None needed | No bridge interaction |

### canUseTool Permission Flows (TS sidecar → Go bridge)

| Tool Name | Sidecar Bridge Calls | Notes |
|-----------|---------------------|-------|
| `EnterPlanMode` | `POST /modal/confirm` | Single confirm modal |
| `ExitPlanMode` | `POST /modal/ask` (approve/changes/reject) → optionally `POST /modal/input` (feedback) | Also sends `POST /planpreview` with .md content |
| `AskUserQuestion` | N × `POST /modal/ask` | Sequential, 1-4 questions |
| Standard tool | `POST /modal/ask` (allow/deny) → optionally `POST /modal/input` (deny reason) | Two-step flow |
| Any tool in `acceptEdits` mode | No bridge call | Auto-approved in sidecar |

### SDK Event → Bubble Tea Message Mapping

| NDJSON Event | Current Handler | Go tea.Msg Type | State Mutation |
|-------------|----------------|-----------------|----------------|
| `system.init` | `handleSystemEvent()` | `SystemInitMsg` | Set sessionID, model, register root agent |
| `system.status` | `handleStatusEvent()` | `StatusUpdateMsg` | Update permissionMode, compacting flag |
| `system.compact_boundary` | `handleCompactBoundaryEvent()` | `CompactMsg` | Show toast, clear compacting |
| `assistant` | `handleAssistantEvent()` | `AssistantMsg` | Append/update message, extract agent activity |
| `user` (tool results) | `handleUserEvent()` | `ToolResultMsg` | Add tool result message, detect agent completion |
| `result` | `handleResultEvent()` | `ResultMsg` | Update cost, tokens, context window, resolve streaming |

---

## 15. Zustand Slice → Go Struct Field Mapping

| Zustand Slice | Fields | Go Location | Notes |
|---------------|--------|-------------|-------|
| **MessagesSlice** | `messages[]` | `AppModel.providers.messages[provider]` | Per-provider message arrays |
| | `addMessage()` | Direct append in `Update()` | |
| | `updateLastMessage()` | Modify last element in-place | |
| | `clearMessages()` | Reset slice | |
| **AgentsSlice** | `agents{}` | `AppModel.agents.agents` map | |
| | `selectedAgentId` | `AppModel.agents.selectedID` | |
| | `rootAgentId` | `AppModel.agents.rootAgentID` | |
| | `addAgent()` | `agents.Register()` in `Update()` | |
| | `updateAgent()` | `agents.Update()` in `Update()` | |
| | `updateAgentActivity()` | `agents.SetActivity()` in `Update()` | |
| **SessionSlice** | `totalCost` | `AppModel.session.TotalCost` | |
| | `tokenCount` | `AppModel.session.TokenCount` | |
| | `contextWindow` | `AppModel.session.ContextWindow` | |
| | `permissionMode` | `AppModel.session.PermissionMode` | |
| | `isCompacting` | `AppModel.session.IsCompacting` | |
| | `providerMessages` | `AppModel.providers.Messages` | |
| | `providerSessionIds` | `AppModel.providers.SessionIDs` | |
| | `providerModels` | `AppModel.providers.Models` | |
| **UISlice** | `streaming` | `AppModel.session.Streaming` | |
| | `focusedPanel` | `AppModel.focus` | |
| | `rightPanelMode` | `AppModel.rightPanelMode` | |
| | `activeTab` | `AppModel.activeTab` | |
| | `activeProvider` | `AppModel.providers.Active` | |
| | `interruptQuery` | `AppModel.cliDriver.Interrupt()` | Direct method call |
| | `planPreviewContent` | `AppModel.planPreview.Content` | |
| | `currentPlanFile` | `AppModel.planPreview.FilePath` | |
| **InputSlice** | `inputHistory[]` | `AppModel.claudePanel.history` | |
| | `inputHistoryIndex` | `AppModel.claudePanel.historyIdx` | |
| **ModalSlice** | `modalQueue[]` | `AppModel.modalQueue` | |
| | `enqueue()` | Bridge API → `ModalRequestMsg` | |
| | `dequeue()` | `ModalResponseMsg` handler | |
| **TelemetrySlice** | All fields | `AppModel.telemetry` struct | |
| **ToastSlice** | `toasts[]` | `AppModel.toast.items` | Auto-dismiss via `tea.Every` |
| **TeamsSlice** | `teams[]` | `AppModel.teams` struct | |

---

## 16. Keybinding Inventory

Complete keybinding map — every binding must be implemented in Go.

### Global (always active when no modal)
| Key | Action | Current Source | Go Binding |
|-----|--------|---------------|------------|
| `Tab` | Toggle panel focus (claude ↔ agents) | `keybindings.ts` `toggleFocus` | `keys.ToggleFocus` |
| `Shift+Tab` | Cycle provider | `ProviderTabs.tsx` | `keys.CycleProvider` |
| `Alt+R` | Cycle right panel mode | `keybindings.ts` `cycleRightPanel` | `keys.CycleRightPanel` |
| `Alt+P` | Cycle permission mode | `keybindings.ts` `cyclePermissionMode` | `keys.CyclePermMode` |
| `Escape` | Interrupt query / Cancel modal | `keybindings.ts` `interruptQuery` | `keys.Interrupt` |
| `Ctrl+C` | Force quit | `keybindings.ts` `forceQuit` | `keys.ForceQuit` |
| `Ctrl+L` | Clear screen | `keybindings.ts` `clearScreen` | `keys.ClearScreen` |
| `Alt+B` | Toggle task board Active/Done | `keybindings.ts` `toggleTaskBoardTab` | `keys.ToggleTaskBoard` |

### Tab Shortcuts
| Key | Action | Tab Target |
|-----|--------|-----------|
| `Alt+C` | Switch to Chat | `chat` |
| `Alt+A` | Switch to Agent Config | `agentConfig` |
| `Alt+T` | Switch to Team Config | `teamConfig` |
| `Alt+Y` | Switch to Telemetry | `telemetry` |

### Claude Panel (focused + no modal)
| Key | Action | Current Source |
|-----|--------|---------------|
| `Enter` | Submit message | `ClaudePanel.tsx` submit handler |
| `Up` | Previous history entry | `ClaudePanel.tsx` history navigation |
| `Down` | Next history entry | `ClaudePanel.tsx` history navigation |
| `Alt+E` | Toggle tool expansion | `ClaudePanel.tsx` expansion toggle |
| `Alt+Shift+E` | Cycle expansion level (0→1→2) | `ClaudePanel.tsx` expansion cycle |

### Agents Panel (focused + no modal)
| Key | Action |
|-----|--------|
| `Up` | Select previous agent |
| `Down` | Select next agent |
| `Enter` | Expand agent details |

### Modal Active
| Key | Action |
|-----|--------|
| `Up/Down` | Navigate options |
| `Enter` | Select current option |
| `Escape` | Cancel modal |
| Typing | Switch to free-text mode (AskModal "Other") |

---

## 17. Risk Register & Design Recommendations

### Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Bubble Tea v2 instability** | API changes mid-development | Pin to v1 initially. v2's `tea.View` struct is compelling but ecosystem hasn't caught up. Migrate later. |
| **NDJSON protocol undocumented edge cases** | Missed event types cause state desync | Comprehensive logging of unknown event types. Fuzzy-test with real Claude sessions. The current `SessionManager` handles ~6 event types — unlikely many more exist. |
| **TS MCP sidecar startup latency** | Cold Node.js start adds 500ms-1s to launch | Pre-bundle with esbuild (single .js file). Consider `bun` runtime if installed. Show "Starting MCP..." in status line. |
| **HTTP bridge reliability** | Network errors between Go ↔ TS processes | Localhost-only, retry with backoff, health check loop. If sidecar dies, attempt restart. |
| **canUseTool multi-step flows** | Complex sequential modal calls may timeout | Generous per-step timeouts. Log each step for debugging. |
| **Markdown rendering quality** | Glamour may differ from marked-terminal | Compare output early. Glamour is actively maintained and handles code blocks, tables, etc. well. |
| **stdin message format** | Claude CLI stdin format for `--output-format stream-json` may not accept raw text | Verify: may need to send structured JSON or use `--input-format` flag. Fallback: pipe user text directly. |

### Design Recommendations

**1. Use value receivers for all models.** Bubble Tea's `Update()` should use `func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` with value receivers. This prevents accidental mutation outside the event loop and makes the data flow provably unidirectional.

**2. Define all tea.Msg types in a single `messages.go` file.** With 15+ message types (SystemInitMsg, AssistantMsg, ResultMsg, ModalRequestMsg, AgentRegisteredMsg, etc.), centralizing them prevents import cycles and makes the message vocabulary discoverable.

**3. Use lipgloss.GetFrameSize() religiously.** Every dimension calculation must account for border overhead. The #1 Bubble Tea layout bug is off-by-two errors from forgotten borders.

**4. Implement a debug log viewer.** Add `--debug` flag that writes all tea.Msg traffic to a log file. Replaces the current `utils/logger.ts`. Use `charmbracelet/log` for structured output. Critical for diagnosing state issues without printf-debugging in the terminal.

**5. Consider `tea.Program.Send()` concurrency.** The bridge API handlers run in HTTP goroutines but inject messages via `Send()`. This is safe — Bubble Tea's `Send()` is explicitly goroutine-safe. But document this pattern clearly for contributors.

**6. Build the ClaudePanel viewport with auto-scroll.** The viewport must auto-scroll to bottom on new content (streaming) but respect manual scroll position when the user scrolls up. This is the single most important UX detail — the current Ink `ScrollView.tsx` (250 lines) fights this constantly. In Bubble Tea, use `viewport.GotoBottom()` conditionally:
```go
if m.autoScroll {
    m.viewport.GotoBottom()
}
// Disable autoScroll on manual scroll-up, re-enable on new content at bottom
```

**7. Bundle the TS sidecar as a single file.** Use the existing esbuild config to produce one `.js` file. Ship it alongside the Go binary or embed it with `go:embed`. This way the user only needs `node` installed, not `npm install`.

**8. Use `tea.Batch()` for Init().** The root model's `Init()` should batch all startup commands:
```go
func (m AppModel) Init() tea.Cmd {
    return tea.Batch(
        m.launchSidecar(),
        m.startBridgeAPI(),
        m.banner.Init(),
        m.statusLine.Init(),
        tea.EnterAltScreen,
        m.pollTerminalSize(),
    )
}
```

**9. Gate CLI spawn on sidecar readiness.** Don't spawn Claude Code CLI until the MCP sidecar is healthy. Startup sequence: sidecar → health check → write MCP config → spawn CLI → wait for system.init → READY.

---

## 18. Dependency Inventory

### Go Dependencies (go.mod)

| Package | Purpose | Replaces (TS) |
|---------|---------|---------------|
| `github.com/charmbracelet/bubbletea` | TUI framework | `ink` |
| `github.com/charmbracelet/bubbles` | Components (viewport, textinput, spinner, list, help, key) | `ink` built-ins, custom primitives |
| `github.com/charmbracelet/lipgloss` | Styling + layout | `ink` Box/Text styles |
| `github.com/charmbracelet/glamour` | Markdown rendering | `marked` + `marked-terminal` |
| `github.com/charmbracelet/log` | Structured logging | `utils/logger.ts` |
| `github.com/spf13/cobra` | CLI argument parsing | `commander` |
| `github.com/fsnotify/fsnotify` | File watching (team config polling) | `chokidar` |

### TS Sidecar Dependencies (package.json)

| Package | Purpose | Status |
|---------|---------|--------|
| `@anthropic-ai/claude-agent-sdk` | MCP server + SDK types | **Kept** (core rationale) |
| `zod` | Tool schema validation | **Kept** |
| `nanoid` | ID generation | **Kept** (for spawn IDs) |

### Removed (no longer needed)

| Package | Reason |
|---------|--------|
| `ink` | Replaced by Bubble Tea |
| `react` | Replaced by Elm architecture |
| `zustand` | Replaced by model struct |
| `ink-select-input` | Replaced by bubbles/list |
| `ink-spinner` | Replaced by bubbles/spinner |
| `marked` / `marked-terminal` | Replaced by Glamour |
| `ink-testing-library` | Replaced by teatest |
| `@anthropic-ai/sdk` | Only needed for TS types — sidecar uses agent-sdk |
| `commander` | Replaced by Cobra |
| `chokidar` | Replaced by fsnotify or polling |
| `openai` / `ollama` | Provider adapters stay in sidecar if needed |
| `async-mutex` | Go has sync.Mutex natively |

---

## Ticket Summary by Phase

| Phase | Tickets | Estimated Complexity | Key Deliverable |
|-------|---------|---------------------|-----------------|
| P0: Scaffolding | 3 | Low | Empty Bubble Tea app compiles |
| P1: Core Shell | 7 | Medium | Multi-panel layout with focus + theme |
| P2: CLI Driver | 6 | High | NDJSON parser + subprocess management |
| P3: MCP Sidecar | 6 | High | TS sidecar extracted + HTTP bridge |
| P4: Modals | 5 | Medium-High | Full modal queue + permission flows |
| P5: Agent Tree | 6 | Medium | Agent tree + live monitoring |
| P6: Teams | 4 | Medium | Team polling + detail views |
| P7: Settings/Tabs | 7 | Medium | All secondary panels + plan mode |
| P8: Persistence | 5 | Medium | Session save/load + shutdown |
| P9: Testing | 7 | Medium | Integration tests + feature parity |
| **Total** | **56 tickets** | | |

### Recommended Execution Order

**Phases 0-2 are the critical path.** Get the Go shell rendering and the CLI driver working first — this proves the architecture. Phase 3 (sidecar extraction) can be parallelized with Phase 1 if two people are working.

**Phase 4 (modals) blocks Phase 5+ features** because agent spawning and permissions depend on the modal system.

**Phases 5-7 are parallelizable** once the modal system works.

**Phase 8 can start as soon as Phase 2 is done** (session persistence is independent of UI components).
