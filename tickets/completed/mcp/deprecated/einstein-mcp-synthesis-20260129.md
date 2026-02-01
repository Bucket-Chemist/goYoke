# Einstein Synthesis: MCP Implementation Plan Critical Analysis

**Date:** 2026-01-29
**Analysts:** Einstein (Opus) + Staff-Architect (Sonnet)
**Subject:** Critical evaluation of MCP_IMPLEMENTATION_GUIDE.md against actual MCP framework capabilities
**Status:** CRITICAL GAPS IDENTIFIED - Plan requires significant amendment

---

## Executive Summary

The MCP Implementation Guide proposes an elegant architecture that **will not work as described**. Three fundamental assumptions are incorrect:

1. **Unix socket transport is not supported** by MCP or Claude Code
2. **`--mcp-config` flag does not exist** in Claude CLI
3. **Embedded MCP server as goroutine is incompatible** with how Claude spawns MCP servers

The plan requires either a complete architectural revision or abandonment of MCP in favor of a simpler custom solution.

---

## 1. CRITICAL ERRORS IN CURRENT PLAN

### Error #1: Unix Socket Transport Assumption (FATAL)

**Plan Claims (Lines 88-120):**
```
MCP Server communicates with Claude CLI via Unix socket at /tmp/gofortress-mcp.sock
```

**Reality:**
MCP specification supports **only two transports**:
- **stdio** (standard input/output) - server spawned as subprocess
- **HTTP** (streamable HTTP) - server as HTTP endpoint

**Evidence from `claude mcp add --help`:**
```
-t, --transport <transport>  Transport type (stdio, sse, http). Defaults to stdio
```

No Unix socket option exists. The entire Phase 1 transport implementation (Task 1.2) is based on an unsupported mechanism.

---

### Error #2: --mcp-config Flag Does Not Exist (FATAL)

**Plan Claims (Lines 1346-1374):**
```json
// Generated at /tmp/gofortress-mcp-{PID}.json
{"mcpServers": {"gofortress": {"command": "...", "transport": "unix"}}}
// Passed via: --mcp-config /tmp/gofortress-mcp-{PID}.json
```

**Reality:**
Actual Claude CLI MCP flags (from `claude --help`):
```
--mcp-config <configs...>    Load MCP servers from JSON files or strings
--strict-mcp-config          Only use MCP servers from --mcp-config
```

The `--mcp-config` flag **does exist** but:
1. Expects JSON in specific format (not the one in the guide)
2. Servers must use **stdio or HTTP transport** (not Unix socket)
3. For stdio, Claude **spawns the command** - it doesn't connect to an existing process

The plan's ephemeral config approach could work IF the transport and process model were corrected.

---

### Error #3: Embedded Server vs Subprocess Model (FATAL)

**Plan Claims (Architecture Diagram Lines 70-110):**
```
gofortress (main process)
  ├─ TUI Event Loop (goroutine)
  ├─ MCP Server (goroutine)  ← Embedded, shares memory
  └─ Claude CLI (subprocess)
       └─ connects to MCP server via channels
```

**Reality - How MCP Stdio Actually Works:**
```
gofortress (main process)
  ├─ TUI Event Loop (goroutine)
  └─ Claude CLI (subprocess)
       └─ spawns MCP Server (sub-subprocess via command in config)
            ├─ stdin  ← Claude writes JSON-RPC requests
            └─ stdout → Claude reads JSON-RPC responses
```

**Critical Implication:**
The MCP server runs as a **grandchild process** of gofortress, NOT as a sibling goroutine. There is no shared memory, no direct channel access. The MCP server has no way to communicate with the TUI event loop.

---

## 2. ARCHITECTURE GAPS

### Gap #1: Process Hierarchy Problem

**Challenge:** How does the MCP server (spawned by Claude) send prompts to the TUI (in gofortress)?

**Plan's Channel-Based IPC (Won't Work):**
```go
// From MCP_IMPLEMENTATION_GUIDE.md
toolRequests  chan ToolRequest   // Send to TUI
toolResponses chan ToolResponse  // Receive from TUI
```

Go channels only work between goroutines **in the same process**. The MCP server is in a different process.

**Required: Cross-Process IPC**
Options:
- HTTP callback (MCP server POSTs to gofortress)
- Domain socket (MCP server connects to pre-existing socket)
- Named pipe
- Shared file with polling

---

### Gap #2: MCP Server Lifecycle

**Questions Not Addressed:**
1. When is the MCP server registered? (Before Claude starts)
2. How is registration cleaned up on gofortress exit?
3. What happens on gofortress crash? (Stale MCP registration)
4. How do multiple gofortress instances coexist?

**Current Flow in `cmd/gofortress/main.go`:**
```go
// Line 103-106
process, err = cli.NewClaudeProcess(cfg)
process.Start()
```

MCP registration must happen BEFORE this, but is not in the plan.

---

### Gap #3: AllowedTools Contradiction

**Plan (Line 1664):**
```go
claudeCfg.AllowedTools = append(claudeCfg.AllowedTools, "mcp__gofortress__ask_user")
```

**Problem:**
- `AllowedTools` = pre-approved, no permission prompt
- `ask_user` tool = designed to prompt user

If the tool is pre-approved, Claude uses it without prompting. The TUI needs to intercept the tool execution and display the prompt, but this requires the MCP server to communicate back to the TUI - which brings us back to the IPC problem.

---

### Gap #4: Existing TUI Event Stream Integration

**Current Event Flow (`internal/cli/subprocess.go`):**
```go
func (cp *ClaudeProcess) readEvents() {
    // Reads NDJSON from Claude stdout
    // Sends to cp.events channel
    // TUI subscribes via Events()
}
```

**Current TUI Handling (`internal/tui/claude/events.go`):**
```go
func (m PanelModel) handleEvent(event cli.Event) PanelModel {
    switch event.Type {
    case "assistant": // Handle text streaming
    case "result":    // Handle completion
    case "system":    // Handle hooks
    // No MCP event handling
    }
}
```

**Missing:** How do MCP tool calls appear in the event stream? The plan doesn't specify the event format or TUI handling.

---

## 3. COMPATIBILITY WITH EXISTING GO IMPLEMENTATION

### What's Already Built (GOgent-109 to 121)

| Component | Location | Status |
|-----------|----------|--------|
| Claude subprocess management | `internal/cli/subprocess.go` | ✅ Complete |
| NDJSON streaming | `internal/cli/streams.go` | ✅ Complete |
| Event parsing | `internal/cli/events.go` | ✅ Complete |
| TUI conversation panel | `internal/tui/claude/panel.go` | ✅ Complete |
| Agent tree visualization | `internal/tui/agents/model.go` | ✅ Complete |
| Layout management | `internal/tui/layout/layout.go` | ✅ Complete |
| Session picker | `internal/tui/session/picker.go` | ✅ Complete |

### What MCP Plan Proposes to Add

| Component | Location (Proposed) | Compatibility |
|-----------|---------------------|---------------|
| MCP Protocol | `internal/mcp/protocol/` | ✅ Can add |
| Unix Transport | `internal/mcp/transport/unix.go` | ❌ Won't work |
| Tool Registry | `internal/mcp/tools/registry.go` | ✅ Can add |
| ask_user Tool | `internal/mcp/tools/ask_user.go` | ⚠️ Needs IPC fix |
| TUI MCP Integration | `internal/tui/claude/mcp_integration.go` | ⚠️ Needs redesign |

### Key Integration Points

**`ClaudeProcess.Config` needs:**
```go
// Current (subprocess.go line 18-79)
type Config struct {
    AllowedTools    []string
    // ... existing fields
}

// Needed additions:
type Config struct {
    MCPServers      []MCPServerConfig  // Pre-registration
    MCPCallbackPort int                // For IPC back to TUI
}
```

**`PanelModel` needs:**
```go
// Current (panel.go line 80-97)
type PanelModel struct {
    process   ClaudeProcessInterface
    // ... existing fields
}

// Needed additions:
type PanelModel struct {
    mcpCallback   *http.Server       // HTTP server for MCP callbacks
    pendingPrompt *MCPPromptRequest  // Current prompt awaiting response
}
```

---

## 4. RECOMMENDED ARCHITECTURE OPTIONS

### Option A: Stdio Transport + HTTP Callback (RECOMMENDED)

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

**MCP Server Binary (`cmd/gofortress-mcp-server/main.go`):**
```go
func main() {
    callbackURL := os.Getenv("GOFORTRESS_CALLBACK")

    // Standard MCP stdio server using Go SDK
    server := mcp.NewServer()
    server.AddTool("ask_user", func(input AskUserInput) (string, error) {
        // POST prompt to gofortress TUI
        resp, _ := http.Post(callbackURL+"/prompt", "application/json", ...)
        // Wait for response
        return resp.Answer, nil
    })
    server.ServeStdio()
}
```

**Pros:**
- MCP-compliant (stdio transport)
- Works with Claude Code's actual integration
- Extensible to more MCP tools

**Cons:**
- Requires separate binary
- HTTP callback adds latency (~10-50ms)
- Callback port discovery complexity

**Estimated Effort:** 8-10 weeks

---

### Option B: Custom Tool Approval (NO MCP)

**Architecture:**
```
gofortress
  │
  └─ Claude subprocess
       │
       └─ stdout: NDJSON events including tool_use
            │
            ├─ TUI intercepts tool_use events
            ├─ Displays approval prompt
            ├─ On approve: let Claude continue
            └─ On reject: ... (complex - may need --permission-mode)
```

**Key Insight:**
Claude Code already has a permission system. The challenge is:
1. Currently gofortress uses `AllowedTools` to bypass prompts
2. Without AllowedTools, Claude prompts via its own UI (not TUI)
3. With `--permission-mode delegate`, prompts appear in stream

**Implementation in TUI:**
```go
// In events.go handleEvent()
case "user":
    // "user" events contain permission requests
    // Already being captured (lines 68-81 in events.go)
    // Need to respond via stdin
```

**Gap:** The current implementation captures "user" events but doesn't respond to them. Need to:
1. Parse permission request from "user" event
2. Display prompt in TUI
3. Send response via `process.SendJSON()` with approval/rejection

**Pros:**
- No separate binary needed
- Uses existing Claude permission infrastructure
- Fastest to implement (existing event capture)

**Cons:**
- Not extensible to custom MCP tools
- Tied to Claude's permission event format (may change)

**Estimated Effort:** 2-4 weeks

---

### Option C: Domain Socket IPC

Similar to Option A but uses Unix domain socket instead of HTTP for the callback. Lower latency but more complex lifecycle management.

**Not Recommended:** HTTP is simpler and latency difference is negligible for user prompts.

---

## 5. AMENDED TICKET STRUCTURE

If proceeding with **Option A (Stdio + HTTP Callback)**:

### Phase 1: Foundation (Revised)

| Ticket | Original | Amended |
|--------|----------|---------|
| 1.1 MCP Protocol | Keep | Add Go SDK integration |
| 1.2 Unix Transport | **DELETE** | Replace with HTTP Callback Server |
| 1.3 Tool Registry | Keep | Minimal changes |
| 1.4 Testing | Keep | Add HTTP callback tests |

**New Ticket 1.2a: HTTP Callback Server**
```markdown
# Task 1.2a: HTTP Callback Server

**Files:** `internal/mcp/callback/server.go`
**Complexity:** Medium
**Time:** 2-3 days

**Subtasks:**
1. HTTP server with /prompt endpoint
2. Random port selection with retry
3. JSON request/response parsing
4. Timeout handling (60s)
5. Graceful shutdown
6. Unit tests
```

**New Ticket 1.5: MCP Server Binary**
```markdown
# Task 1.5: MCP Server Binary

**Files:** `cmd/gofortress-mcp-server/main.go`
**Complexity:** Medium
**Time:** 3-4 days

**Subtasks:**
1. Stdio MCP server using Go SDK
2. ask_user tool implementation
3. HTTP callback to GOFORTRESS_CALLBACK
4. Error handling for callback failures
5. Integration tests with mock callback
```

### Phase 2: Integration (Revised)

| Ticket | Original | Amended |
|--------|----------|---------|
| 2.1 ask_user Tool | Move to 1.5 | In MCP server binary |
| 2.2 MCP Server Loop | **DELETE** | Replaced by Go SDK |
| 2.3 TUI Integration | Revise | HTTP callback handling |
| 2.4 Prompt Rendering | Keep | Minor changes |
| 2.5 Input Handling | Keep | Handle HTTP responses |
| 2.6 E2E Integration | Revise | MCP registration flow |

**Revised Ticket 2.3: TUI HTTP Callback Integration**
```markdown
# Task 2.3a: TUI HTTP Callback Integration

**Files:** `internal/tui/claude/mcp_callback.go`, `panel.go`
**Complexity:** High
**Time:** 4-5 days

**Subtasks:**
1. Start callback server in TUI init
2. Handle /prompt POST requests
3. Display prompt in conversation panel
4. Collect user response (1-4 keys)
5. Return response via HTTP
6. Timeout handling (close request after 60s)
7. Integration with existing event loop
```

**Revised Ticket 2.6: E2E Integration**
```markdown
# Task 2.6a: End-to-End Integration

**Files:** `cmd/gofortress/main.go`
**Complexity:** High
**Time:** 3-4 days

**Subtasks:**
1. Start HTTP callback server FIRST
2. Generate MCP config JSON with callback URL
3. Write config to temp file
4. Pass --mcp-config to Claude subprocess
5. Verify tool availability in Claude
6. Test prompt round-trip
7. Cleanup on exit (remove temp file)
```

---

## 6. IF PROCEEDING WITH OPTION B (Custom Approval)

**Much Simpler - Only 3 New Tickets:**

### Ticket B.1: Permission Event Parser
```markdown
# Task B.1: Permission Event Parser

**Files:** `internal/cli/events.go`, `permissions.go`
**Complexity:** Medium
**Time:** 2-3 days

**Subtasks:**
1. Define PermissionRequest struct
2. Parse "user" events containing permissions
3. Extract tool name, arguments, context
4. Unit tests with captured events
```

### Ticket B.2: Permission Response Handler
```markdown
# Task B.2: Permission Response Handler

**Files:** `internal/cli/subprocess.go`
**Complexity:** Medium
**Time:** 2-3 days

**Subtasks:**
1. Define PermissionResponse struct
2. SendPermissionResponse method
3. Format as JSON for Claude stdin
4. Integration tests
```

### Ticket B.3: TUI Permission Prompt
```markdown
# Task B.3: TUI Permission Prompt

**Files:** `internal/tui/claude/permissions.go`, `panel.go`
**Complexity:** Medium
**Time:** 3-4 days

**Subtasks:**
1. Permission prompt UI component
2. Display in conversation panel
3. Key handlers (y/n, 1-4 for options)
4. Send response via process.SendPermissionResponse()
5. Clear prompt state
6. Integration tests
```

---

## 7. DECISION MATRIX

| Factor | Option A (MCP + HTTP) | Option B (Custom) |
|--------|----------------------|-------------------|
| **Implementation Time** | 8-10 weeks | 2-4 weeks |
| **Complexity** | High | Medium |
| **MCP Compliance** | ✅ Full | ❌ None |
| **Extensibility** | ✅ Add more tools | ❌ Fixed to permissions |
| **Separate Binary** | Yes | No |
| **Claude Updates Risk** | Low (MCP is stable) | Medium (permission format may change) |
| **Debugging** | Harder (multi-process) | Easier (single process) |

---

## 8. FINAL RECOMMENDATION

### For GOgent-Fortress's Stated Goals:

**Primary Goal:** "Users want interactive prompts for decisions, ability to approve/reject actions, guidance over tool execution"

**Recommendation:** Start with **Option B (Custom Approval)** because:

1. **Fastest path to value** - 2-4 weeks vs 8-10 weeks
2. **Lower risk** - Single process, uses existing event stream
3. **Already partially implemented** - "user" events are being captured (events.go:68-81)
4. **Solves the actual problem** - User approval for tool execution

### For Future Extensibility:

If you later need:
- Custom MCP tools beyond permissions
- Integration with broader MCP ecosystem
- Third-party MCP servers

Then implement **Option A (Stdio + HTTP Callback)** as a v2.0 feature.

---

## 9. IMMEDIATE NEXT STEPS

### If Proceeding with Option B:

1. **Analyze captured permission events** - Check `/tmp/user-event-*.json` files to understand exact format
2. **Document permission response format** - What JSON does Claude expect on stdin?
3. **Create tickets B.1, B.2, B.3** - Replace current MCP tickets
4. **Update ARCHITECTURE.md** - Remove MCP section, add Permission Approval section

### If Proceeding with Option A:

1. **Delete tickets 1.2, 2.2** - Unix transport and embedded server
2. **Create tickets 1.2a, 1.5** - HTTP callback and MCP server binary
3. **Revise tickets 2.3, 2.6** - TUI callback integration and E2E flow
4. **Update MCP_IMPLEMENTATION_GUIDE.md** - New architecture diagrams

---

## APPENDIX: Sources

1. MCP Transports Specification: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
2. Claude Code MCP Documentation: https://docs.anthropic.com/en/docs/claude-code/mcp
3. Go MCP SDK: https://github.com/modelcontextprotocol/go-sdk
4. Existing codebase: `internal/cli/subprocess.go`, `internal/tui/claude/`
5. Claude CLI help: `claude --help`, `claude mcp add --help`

---

**End of Einstein Synthesis**

*Generated by Einstein (Opus) with Staff-Architect (Sonnet) critical review*
*Agent IDs: einstein-session, a703acf*
