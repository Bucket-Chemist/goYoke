# GOgent-Fortress MCP Implementation Handoff Document

**Date:** 2026-01-29
**Authors:** Einstein (Opus) + Staff-Architect (Sonnet)
**Status:** CRITICAL ARCHITECTURAL GAPS IDENTIFIED
**Purpose:** Comprehensive handoff for deep research continuation

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Context](#2-project-context)
3. [Existing Architecture](#3-existing-architecture)
4. [Proposed MCP Implementation](#4-proposed-mcp-implementation)
5. [Critical Analysis Results](#5-critical-analysis-results)
6. [Viable Alternative Architectures](#6-viable-alternative-architectures)
7. [Key Source Files](#7-key-source-files)
8. [Deep Research Questions](#8-deep-research-questions)
9. [Decision Framework](#9-decision-framework)
10. [Appendix: Full Ticket List](#10-appendix-full-ticket-list)

---

## 1. Executive Summary

### The Problem

GOgent-Fortress is a Go-based hook orchestration framework for Claude Code. The TUI component (`gofortress`) currently spawns Claude CLI as a subprocess and communicates via NDJSON streaming. However, it uses `AllowedTools` for pre-approval, which **removes user control** over Claude's actions.

**User wants:**
- Interactive prompts for decisions
- Ability to approve/reject actions
- Guidance over tool execution

### The Proposed Solution (MCP_IMPLEMENTATION_GUIDE.md)

An embedded MCP server running as a goroutine in gofortress, communicating with Claude CLI via Unix socket, providing an `mcp__gofortress__ask_user` tool for interactive prompts.

### Critical Finding: THE PROPOSED SOLUTION WILL NOT WORK

**3 Fatal Architectural Errors:**

| Error | Plan Says | Reality |
|-------|-----------|---------|
| **Transport** | Unix socket | MCP only supports **stdio** and **HTTP** |
| **Process Model** | Embedded goroutine | MCP servers must be **spawned by Claude as subprocess** |
| **Config Flag** | `--mcp-config path.json` | Flag exists but expects Claude to **spawn the command**, not connect to existing server |

### Recommended Path Forward

**Option A (Recommended for MCP compliance):** Stdio + HTTP Callback (8-10 weeks)
- Separate MCP server binary (`gofortress-mcp-server`)
- HTTP callback from MCP server to TUI for prompt display
- Full MCP ecosystem compatibility

**Option B (Recommended for speed):** Custom Tool Approval (2-4 weeks)
- No MCP - intercept existing permission events
- Already 60% implemented (events.go captures "user" events)
- Fastest path to user prompts

---

## 2. Project Context

### What is GOgent-Fortress?

A Go-based hook orchestration framework for Claude Code that:
1. **Intercepts hook events** (SessionStart, PreToolUse, PostToolUse, SubagentStop, SessionEnd)
2. **Enforces routing policies** via `routing-schema.json`
3. **Tracks failures** and captures "sharp edges" (debugging loops)
4. **Manages ML telemetry** for routing optimization
5. **Maintains session continuity** via structured handoff documents

### Project Statistics

| Metric | Value |
|--------|-------|
| Total Go LOC | ~27,000 |
| Hook Binaries | 5 (`gogent-load-context`, `gogent-validate`, `gogent-sharp-edge`, `gogent-agent-endstate`, `gogent-archive`) |
| TUI Tickets | 13 (GOgent-109 through GOgent-121, all complete) |
| Test Coverage | ~85% average |
| Schema Version | routing-schema v2.2.0, handoff v1.3 |

### Key Configuration Files

| File | Location | Purpose |
|------|----------|---------|
| `routing-schema.json` | `~/.claude/` | Source of truth for tiers, agents, thresholds |
| `CLAUDE.md` | `~/.claude/` | Global Claude configuration, routing gates |
| `conventions/*.md` | `~/.claude/conventions/` | Language-specific coding conventions |
| `agents/*.yaml` | `~/.claude/agents/` | Agent definitions with triggers |
| `handoffs.jsonl` | `.claude/memory/` | Session continuity data |

### Build & Run

```bash
# Build all hook binaries
make build

# Run TUI
go run cmd/gofortress/main.go

# Run tests
go test ./...

# Run specific hook
echo '{"type":"session_start",...}' | ./bin/gogent-load-context
```

---

## 3. Existing Architecture

### 3.1 System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                                   GOgent-Fortress Architecture                           │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                          │
│  ┌────────────────────────────────────────────────────────────────────────────────┐     │
│  │                            CLAUDE CODE CLI PROCESS                              │     │
│  │                                                                                 │     │
│  │    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │     │
│  │    │ SessionStart│───▶│ PreToolUse  │───▶│    Tool     │───▶│ PostToolUse │   │     │
│  │    │   Event     │    │   Event     │    │  Execution  │    │   Event     │   │     │
│  │    └──────┬──────┘    └──────┬──────┘    └─────────────┘    └──────┬──────┘   │     │
│  │           │                  │                                     │           │     │
│  └───────────┼──────────────────┼─────────────────────────────────────┼───────────┘     │
│              │ STDIN/JSON       │ STDIN/JSON                          │ STDIN/JSON      │
│              ▼                  ▼                                     ▼                 │
│  ┌────────────────────────────────────────────────────────────────────────────────┐     │
│  │                                HOOK BINARIES (Go)                               │     │
│  │                                                                                 │     │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐              │     │
│  │  │ gogent-load-     │  │  gogent-validate │  │ gogent-sharp-edge│              │     │
│  │  │ context          │  │                  │  │                  │              │     │
│  │  │ • Language detect│  │ • Schema validate│  │ • Tool counting  │              │     │
│  │  │ • Convention load│  │ • Task() checks  │  │ • Failure track  │              │     │
│  │  │ • Handoff restore│  │ • Tier enforce   │  │ • ML telemetry   │              │     │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────┘              │     │
│  └────────────────────────────────────────────────────────────────────────────────┘     │
│                                                                                          │
│  ┌────────────────────────────────────────────────────────────────────────────────┐     │
│  │                          TUI SYSTEM (GOgent-109 to GOgent-121)                  │     │
│  │                                                                                 │     │
│  │   📦 internal/cli                    📦 internal/tui/claude                     │     │
│  │   • ClaudeProcess subprocess         • PanelModel (conversation)                │     │
│  │   • NDJSON reader/writer             • Streaming output display                 │     │
│  │   • Event type parsing               • Hook event sidebar                       │     │
│  │   • Auto-restart on panic            • User input handling                      │     │
│  │                                                                                 │     │
│  │   📦 internal/tui/agents             📦 internal/tui/layout                     │     │
│  │   • AgentTree data model             • 70/30 split layout                       │     │
│  │   • TreeModel (view)                 • Focus management (Tab)                   │     │
│  │   • DetailModel (sidebar)            • BannerModel (nav tabs)                   │     │
│  └────────────────────────────────────────────────────────────────────────────────┘     │
│                                                                                          │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Claude Subprocess Management (`internal/cli/subprocess.go`)

**Key Components:**

```go
// ClaudeProcess manages the lifecycle of a claude subprocess
type ClaudeProcess struct {
    cmd           *exec.Cmd
    stdin         io.WriteCloser
    stdout        io.ReadCloser
    writer        *NDJSONWriter
    config        Config
    sessionID     string
    events        chan Event      // Buffered channel (size 100)
    errors        chan error
    restartEvents chan RestartEvent
    done          chan struct{}
    // ... additional fields for restart, generation tracking
}

// Config holds configuration for starting a Claude subprocess
type Config struct {
    ClaudePath      string
    SessionID       string
    AllowedTools    []string   // Pre-approved tools (THE PROBLEM)
    DisallowedTools []string
    MaxTurns        int
    Model           string
    // ... other fields
}
```

**How Claude CLI is spawned:**

```go
// From subprocess.go lines 141-186
args := []string{
    "--print",
    "--verbose",
    "--debug-to-stderr",
    "--input-format", "stream-json",
    "--output-format", "stream-json",
    "--session-id", sessionID,
}

// Tool restrictions
for _, tool := range cfg.AllowedTools {
    args = append(args, "--allowed-tools", tool)
}

cmd := exec.Command(cfg.ClaudePath, args...)
```

**Key Insight:** The current implementation has NO MCP-related flags. Adding `--mcp-config` would be the integration point.

### 3.3 TUI Event Flow (`internal/tui/claude/`)

**PanelModel State:**

```go
// From panel.go
type PanelModel struct {
    process        ClaudeProcessInterface
    viewport       viewport.Model
    textarea       textarea.Model
    messages       []Message
    hooks          []HookEvent
    cost           float64
    state          ProcessState  // Connecting, Ready, Streaming, Restarting, Stopped, Error
    currentModel   string
    config         cli.Config
    // NO MCP fields currently
}
```

**Event Handling:**

```go
// From events.go - handleEvent()
func (m PanelModel) handleEvent(event cli.Event) PanelModel {
    switch event.Type {
    case "assistant":
        // Handle text streaming, tool use display
    case "result":
        // Handle completion, cost update
    case "system":
        // Handle hook responses
    case "user":
        // IMPORTANT: Permission events captured here (lines 68-81)
        // Currently just saved to /tmp/user-event-*.json for debugging
        // THIS IS THE INTEGRATION POINT FOR OPTION B
    }
}
```

### 3.4 Data Persistence Paths

```
Project/
├── .claude/
│   ├── memory/
│   │   ├── handoffs.jsonl           # Session history
│   │   ├── pending-learnings.jsonl  # Sharp edges
│   │   └── last-handoff.md          # Human-readable
│   ├── tmp/
│   │   └── einstein-gap-*.md        # Escalation documents
│   └── session-archive/             # Archived sessions

~/.gogent/
├── failure-tracker.jsonl            # Cross-session failures
└── agent-invocations.jsonl          # Invocation telemetry

$XDG_DATA_HOME/gogent-fortress/
├── routing-decisions.jsonl          # ML training data
└── agent-collaborations.jsonl       # Team patterns

/tmp/
├── claude-tool-counter-*.log        # Tool call counters
└── user-event-*.json                # DEBUG: Captured permission events
```

---

## 4. Proposed MCP Implementation

### 4.1 Original Architecture (FROM MCP_IMPLEMENTATION_GUIDE.md)

```
┌─────────────────────────────────────────────────────────────┐
│                       gofortress                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Main Process                        │  │
│  │                                                       │  │
│  │  ┌──────────────┐         ┌────────────────────┐    │  │
│  │  │ TUI Event    │<───────>│  MCP Server        │    │  │
│  │  │ Loop         │ Channels│  Goroutine         │    │  │
│  │  │ (Bubbletea)  │         │                    │    │  │
│  │  └──────────────┘         └────────┬───────────┘    │  │
│  │                                    │ Unix Socket    │  │
│  │                                    ▼                │  │
│  │                          ┌────────────────────┐    │  │
│  │                          │  /tmp/gofortress-  │    │  │
│  │                          │  mcp.sock          │    │  │
│  │                          └────────────────────┘    │  │
│  └──────────────────────────────┬───────────────────┘  │
│                                 │ Spawn                │
│                                 ▼                      │
│                       ┌─────────────────────┐         │
│                       │   Claude CLI        │         │
│                       │   --mcp-config      │         │
│                       │   gofortress.json   │         │
│                       └─────────────────────┘         │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Why This Won't Work

**Problem 1: Unix Socket Not Supported**

MCP specification (https://modelcontextprotocol.io/specification/2025-06-18/basic/transports):
> "The protocol currently defines two standard transport mechanisms: stdio and Streamable HTTP"

Claude CLI `claude mcp add --help`:
```
-t, --transport <transport>  Transport type (stdio, sse, http). Defaults to stdio
```

No Unix socket option.

**Problem 2: Process Model Inverted**

MCP stdio transport model:
```
Claude Code Process
  └─ spawns subprocess: command + args (the MCP server)
       ├─ stdin  ← Claude writes MCP requests
       └─ stdout → Claude reads MCP responses
```

The plan assumes gofortress spawns Claude, and Claude connects to gofortress's MCP server. But with stdio, **Claude spawns the MCP server**, making it a grandchild of gofortress.

**Problem 3: No Shared Memory**

With separate processes:
- No Go channels between MCP server and TUI
- Need IPC: HTTP, domain socket, or named pipe
- Adds complexity, latency, failure modes

### 4.3 Original Ticket Structure (18 tickets)

| Phase | Tickets | Focus |
|-------|---------|-------|
| 1: Foundation | 1.1-1.4 | MCP Protocol, Unix Transport, Registry, Testing |
| 2: Interactive | 2.1-2.6 | ask_user Tool, Server Loop, TUI Integration |
| 3: Hardening | 3.1-3.4 | Error Handling, Degradation, Testing, Performance |
| 4: Extensibility | 4.1-4.4 | HTTP Transport, Plugin System, Examples, Docs |

**Estimated Timeline:** 6-8 weeks
**Status:** NOT VIABLE as written

---

## 5. Critical Analysis Results

### 5.1 Converged Findings (Einstein + Staff-Architect)

Both analyses independently identified the same 3 fatal flaws:

| Issue | Einstein Finding | Staff-Architect Finding | Severity |
|-------|------------------|-------------------------|----------|
| Unix Socket Transport | "Not a standard MCP transport" | "95% likelihood of failure" | CRITICAL |
| Embedded Goroutine | "Grandchild process problem" | "90% likelihood incompatible" | CRITICAL |
| Blocking Tool Calls | "60s timeout vs 5min user response" | "70% likelihood of timeout" | HIGH |

### 5.2 Process Hierarchy Problem

**What Plan Assumes:**
```
gofortress (parent)
  ├─ TUI (goroutine)
  ├─ MCP Server (goroutine)  ← Shares memory
  └─ Claude CLI (subprocess)
       └─ connects to MCP server
```

**What Actually Happens with stdio:**
```
gofortress (parent)
  ├─ TUI (goroutine)
  └─ Claude CLI (subprocess)
       └─ spawns MCP Server (sub-subprocess)
            ├─ stdin ← Claude
            └─ stdout → Claude
            └─ ??? → How to reach TUI?
```

The MCP server is a **grandchild** with no direct communication path to the TUI.

### 5.3 AllowedTools Contradiction

From `cmd/gofortress/main.go`:
```go
AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task"}
```

The plan proposes:
```go
claudeCfg.AllowedTools = append(claudeCfg.AllowedTools, "mcp__gofortress__ask_user")
```

**Problem:** `AllowedTools` = pre-approved without prompting. The `ask_user` tool would execute immediately without Claude prompting the user. The prompt needs to come from TUI intercepting the tool execution, not from Claude's permission system.

### 5.4 Risk Assessment

| Risk | Severity | Likelihood | Impact |
|------|----------|------------|--------|
| Unix socket not supported | CRITICAL | 95% | Blocks all MCP |
| Embedded goroutine incompatible | CRITICAL | 90% | Requires rewrite |
| Tool calls timeout waiting for user | HIGH | 70% | Poor UX |
| MCP server not auto-registered | MEDIUM | 60% | User confusion |
| Channel deadlock under load | MEDIUM | 40% | System hang |

---

## 6. Viable Alternative Architectures

### 6.1 Option A: Stdio + HTTP Callback (RECOMMENDED for MCP compliance)

**Architecture:**

```
gofortress starts
  │
  ├─1─ Start HTTP callback server on localhost:{random_port}
  │
  ├─2─ Write MCP config JSON:
  │    {
  │      "mcpServers": {
  │        "gofortress": {
  │          "command": "/path/to/gofortress-mcp-server",
  │          "args": [],
  │          "env": {"GOFORTRESS_CALLBACK": "http://localhost:{port}"}
  │        }
  │      }
  │    }
  │
  ├─3─ Spawn Claude: claude --mcp-config /tmp/gofortress-mcp.json ...
  │
  └─4─ TUI event loop + HTTP callback handler
       │
       ├─ Events from Claude stdout (existing)
       └─ Prompts from MCP server via HTTP POST /prompt
```

**New Components:**

1. **`cmd/gofortress-mcp-server/main.go`** - Separate binary
   - MCP server using Go SDK
   - Stdio transport (JSON-RPC with Claude)
   - HTTP client to callback URL

2. **`internal/mcp/callback/server.go`** - HTTP callback server
   - Runs in TUI process
   - Receives prompt requests from MCP server
   - Returns user responses

3. **`internal/tui/claude/prompt.go`** - Prompt display
   - Renders prompts in conversation panel
   - Collects user input (1-4 keys)
   - Returns via HTTP response

**Data Flow:**

```
Claude → MCP tool call (JSON-RPC over stdio) → gofortress-mcp-server
         ↓
gofortress-mcp-server → HTTP POST /prompt → gofortress TUI
         ↓
TUI displays prompt, user presses "1"
         ↓
TUI → HTTP 200 {answer: "SQLite"} → gofortress-mcp-server
         ↓
gofortress-mcp-server → JSON-RPC result → Claude
         ↓
Claude continues: "I'll use SQLite..."
```

**Pros:**
- MCP-compliant (stdio transport)
- Extensible to more MCP tools
- Works with Claude Code's actual integration

**Cons:**
- Requires separate binary (two binaries total)
- HTTP callback adds latency (~10-50ms)
- More complex lifecycle management

**Estimated Effort:** 8-10 weeks

### 6.2 Option B: Custom Tool Approval (RECOMMENDED for speed)

**Architecture:**

```
gofortress
  │
  └─ Claude subprocess (--permission-mode delegate)
       │
       └─ stdout: NDJSON events including "user" type (permission requests)
            │
            ├─ TUI intercepts "user" events
            ├─ Displays approval prompt
            ├─ On approve: send approval response via stdin
            └─ On reject: send rejection response via stdin
```

**Key Insight:** The TUI already captures "user" events (events.go lines 68-81):

```go
case "user":
    // TEMPORARY: Capture raw event for schema discovery
    debugPath := fmt.Sprintf("/tmp/user-event-%d.json", time.Now().Unix())
    if err := os.WriteFile(debugPath, event.Raw, 0644); err != nil {
        // Log error
    }
```

**What's Needed:**

1. **Parse permission request** from "user" event
2. **Display prompt** in TUI conversation panel
3. **Collect user response** (y/n or 1-4)
4. **Send response** via `process.SendJSON()`

**New Tickets (replacing 18 MCP tickets):**

| Ticket | Name | Effort |
|--------|------|--------|
| B.1 | Permission Event Parser | 2-3 days |
| B.2 | Permission Response Handler | 2-3 days |
| B.3 | TUI Permission Prompt | 3-4 days |

**Pros:**
- Fastest implementation (2-4 weeks)
- Single binary (no MCP server)
- Uses existing event stream
- Already 60% implemented

**Cons:**
- Not MCP-compliant
- Not extensible to custom MCP tools
- Tied to Claude's permission event format (may change)

**Estimated Effort:** 2-4 weeks

### 6.3 Comparison Matrix

| Factor | Option A (MCP + HTTP) | Option B (Custom) |
|--------|----------------------|-------------------|
| **Implementation Time** | 8-10 weeks | 2-4 weeks |
| **Complexity** | High | Medium |
| **MCP Compliance** | Full | None |
| **Extensibility** | Add more tools | Fixed to permissions |
| **Separate Binary** | Yes | No |
| **Debugging** | Harder (multi-process) | Easier (single process) |
| **Risk of Claude Updates** | Low (MCP stable) | Medium (format may change) |

---

## 7. Key Source Files

### 7.1 TUI Implementation

| File | Lines | Purpose |
|------|-------|---------|
| `internal/cli/subprocess.go` | ~937 | Claude subprocess lifecycle, NDJSON I/O |
| `internal/cli/events.go` | ~300 | Event type parsing, stream handling |
| `internal/cli/streams.go` | ~200 | NDJSONReader, NDJSONWriter |
| `internal/cli/restart.go` | ~150 | RestartPolicy, exponential backoff |
| `internal/tui/claude/panel.go` | ~491 | Main TUI model, viewport, textarea |
| `internal/tui/claude/events.go` | ~287 | Event handling, hook sidebar |
| `internal/tui/claude/input.go` | ~150 | User input handling |
| `internal/tui/claude/output.go` | ~200 | Viewport rendering |
| `internal/tui/layout/layout.go` | ~300 | 70/30 split, focus management |
| `internal/tui/agents/model.go` | ~400 | AgentTree data model |

### 7.2 Hook Binaries

| Binary | File | Purpose |
|--------|------|---------|
| `gogent-load-context` | `cmd/gogent-load-context/main.go` | SessionStart handler |
| `gogent-validate` | `cmd/gogent-validate/main.go` | PreToolUse validation |
| `gogent-sharp-edge` | `cmd/gogent-sharp-edge/main.go` | PostToolUse handler |
| `gogent-agent-endstate` | `cmd/gogent-agent-endstate/main.go` | SubagentStop handler |
| `gogent-archive` | `cmd/gogent-archive/main.go` | SessionEnd archival |

### 7.3 Core Packages

| Package | Purpose |
|---------|---------|
| `pkg/routing` | Schema loading, Task validation, tier management |
| `pkg/session` | Handoffs, events, metrics, artifacts |
| `pkg/memory` | Failure tracking, pattern matching |
| `pkg/telemetry` | ML training data, cost calculation |
| `pkg/config` | Path resolution, environment detection |

### 7.4 Configuration Files

| File | Location |
|------|----------|
| `routing-schema.json` | `~/.claude/routing-schema.json` |
| `CLAUDE.md` | `~/.claude/CLAUDE.md` |
| `go.md` | `~/.claude/conventions/go.md` |
| `agents-index.json` | `~/.claude/agents-index.json` |

### 7.5 MCP Tickets (Original)

Location: `.claude/mcp/`

```
001-1-1-mcp-protocol-implementation.md
002-1-2-unix-socket-transport.md  ← DELETE (invalid transport)
003-1-3-tool-registry.md
004-1-4-testing-infrastructure.md
005-2-1-ask-user-tool-implementation.md
006-2-2-mcp-server-main-loop.md  ← REVISE (separate binary)
007-2-3-tui-mcp-integration.md  ← REVISE (HTTP callback)
008-2-4-prompt-rendering.md
009-2-5-user-input-handling.md
010-2-6-end-to-end-integration.md  ← REVISE (registration flow)
011-3-1-comprehensive-error-handling.md
012-3-2-graceful-degradation.md
013-3-3-comprehensive-testing.md
014-3-4-performance-optimization.md
015-4-1-http-transport.md
016-4-2-plugin-system-design.md
017-4-3-example-custom-tools.md
018-4-4-documentation.md
```

---

## 8. Deep Research Questions

For the next Opus instance with deep research capabilities, investigate:

### 8.1 MCP Protocol Questions

1. **Does Claude Code support MCP Tasks (async primitives)?**
   - Check: `claude --help`, `claude mcp --help` for task-related flags
   - If yes: Use async tools for long-running prompts
   - If no: Must keep prompts under Claude's timeout limit

2. **What is Claude Code's actual timeout for tool calls?**
   - Test: Create simple MCP server that sleeps 60s, see when Claude gives up
   - Critical for sizing prompt timeout

3. **What is the exact format of `--mcp-config` JSON?**
   - Documentation says it loads MCP servers from JSON
   - Need actual schema validation

4. **How does `--permission-mode delegate` work?**
   - Does it emit "user" events for tool approval?
   - What's the response format?

### 8.2 Implementation Questions

5. **Can MCP server output be streamed back to Claude?**
   - Or must entire response be buffered?
   - Impacts UX for multi-step confirmations

6. **What happens if MCP server crashes mid-session?**
   - Does Claude CLI auto-restart it?
   - Or must gofortress detect and re-register?

7. **How does plugin-provided MCP config work?**
   - Can `.mcp.json` in project root auto-register servers?
   - Or must user run `claude mcp add` manually?

### 8.3 Alternative Research

8. **What is the exact format of "user" permission events?**
   - Check `/tmp/user-event-*.json` files
   - Document schema for Option B implementation

9. **Can we use `--permission-mode` to intercept approvals?**
   - Modes: acceptEdits, bypassPermissions, default, delegate, dontAsk, plan
   - Which mode exposes approval events to TUI?

10. **Are there other Claude Code extensibility mechanisms?**
    - Plugins (`claude plugin --help`)
    - Custom settings
    - Alternative to MCP for our use case

---

## 9. Decision Framework

### 9.1 Choose Option A (MCP + HTTP) If:

- Need MCP ecosystem compatibility
- Plan to add more custom tools (3+ tools over time)
- Can accept 8-10 week implementation timeline
- Willing to maintain two binaries

### 9.2 Choose Option B (Custom Approval) If:

- Only need tool approval prompts (not custom tools)
- Want fastest path to user value (2-4 weeks)
- Prefer single binary simplicity
- Acceptable risk that permission event format may change

### 9.3 Go/No-Go Criteria

**DO NOT PROCEED with current plan if:**
1. Must maintain single binary (incompatible with MCP stdio)
2. Need multi-minute deliberation prompts (will hit timeouts)
3. Cannot modify Claude Code config (no way to register MCP server)

**PROCEED with revised plan (Option A or B) if:**
1. Accept architecture revision
2. Add feasibility validation phase FIRST
3. Willing to test MCP integration before full implementation

---

## 10. Appendix: Full Ticket List

### 10.1 Original MCP Tickets (18)

<details>
<summary>Click to expand full ticket list</summary>

#### Phase 1: Foundation

**1.1 MCP Protocol Implementation**
- Owner: go-pro (Sonnet)
- Files: `internal/mcp/protocol/*.go`
- Subtasks: JSON-RPC 2.0 types, MCP messages, parser, formatter
- Status: KEEP

**1.2 Unix Socket Transport**
- Owner: go-pro (Sonnet)
- Files: `internal/mcp/transport/*.go`
- Status: **DELETE** (Unix socket not supported)

**1.3 Tool Registry**
- Owner: go-pro (Sonnet)
- Files: `internal/mcp/tools/registry.go`
- Status: KEEP

**1.4 Testing Infrastructure**
- Owner: go-pro (Sonnet)
- Status: KEEP

#### Phase 2: Interactive Prompts

**2.1 ask_user Tool Implementation**
- Owner: go-tui (Sonnet)
- Files: `internal/mcp/tools/ask_user.go`
- Status: **MOVE** to MCP server binary

**2.2 MCP Server Main Loop**
- Owner: go-pro (Sonnet)
- Files: `internal/mcp/server/server.go`
- Status: **REVISE** (separate binary, use Go SDK)

**2.3 TUI MCP Integration**
- Owner: go-tui (Sonnet)
- Files: `internal/tui/claude/mcp_integration.go`
- Status: **REVISE** (HTTP callback handling)

**2.4 Prompt Rendering**
- Owner: go-tui (Sonnet)
- Status: KEEP

**2.5 User Input Handling**
- Owner: go-tui (Sonnet)
- Status: KEEP

**2.6 End-to-End Integration**
- Owner: go-tui (Sonnet)
- Status: **REVISE** (MCP registration flow)

#### Phase 3: Hardening

**3.1-3.4** - KEEP (error handling, degradation, testing, performance)

#### Phase 4: Extensibility

**4.1 HTTP Transport**
- Status: **REPURPOSE** as HTTP callback server

**4.2-4.4** - KEEP (plugin system, examples, docs)

</details>

### 10.2 Revised Tickets for Option A

| ID | Name | Effort | Notes |
|----|------|--------|-------|
| A.1 | HTTP Callback Server | 2-3 days | In gofortress, receives prompts |
| A.2 | MCP Server Binary | 3-4 days | Separate binary, Go SDK |
| A.3 | MCP Protocol (keep 1.1) | 2-3 days | JSON-RPC types |
| A.4 | Tool Registry (keep 1.3) | 1-2 days | Thread-safe registry |
| A.5 | TUI HTTP Integration | 4-5 days | Replaces 2.3 |
| A.6 | Prompt Rendering (keep 2.4) | 2-3 days | |
| A.7 | User Input (keep 2.5) | 2 days | |
| A.8 | E2E Integration | 3-4 days | MCP registration |
| A.9-A.12 | Phase 3 hardening | 10-12 days | |
| A.13-A.16 | Phase 4 extensibility | 10-12 days | |

**Total:** 16 tickets, 8-10 weeks

### 10.3 Revised Tickets for Option B

| ID | Name | Effort | Notes |
|----|------|--------|-------|
| B.1 | Permission Event Parser | 2-3 days | Parse "user" events |
| B.2 | Permission Response Handler | 2-3 days | SendPermissionResponse |
| B.3 | TUI Permission Prompt | 3-4 days | Display and collect |

**Total:** 3 tickets, 2-4 weeks

---

## Summary

This handoff document provides complete context for continuing the MCP implementation analysis:

1. **Project is well-established:** ~27K LOC, 85% test coverage, 13 TUI tickets complete
2. **Original MCP plan is not viable:** 3 fatal architectural errors identified
3. **Two viable alternatives exist:** Option A (MCP + HTTP, 8-10 weeks) or Option B (Custom Approval, 2-4 weeks)
4. **Deep research questions are defined:** 10 specific questions for investigation
5. **All source file locations documented:** Ready for implementation

**Recommended Next Step:**
1. Run deep research to answer questions in Section 8
2. Choose Option A or B based on findings
3. Create revised tickets
4. Begin implementation

---

**Document Version:** 1.0
**Created:** 2026-01-29
**Authors:** Einstein (Opus), Staff-Architect (Sonnet)
**Agent IDs:** einstein-session, a703acf
