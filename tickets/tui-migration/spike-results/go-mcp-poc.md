# TUI-002 Spike: Go MCP SDK POC with Claude CLI

**Date:** 2026-03-23
**Claude Code Version:** 2.1.76
**Go MCP SDK Version:** v1.2.0 (`github.com/modelcontextprotocol/go-sdk`)
**Go Version:** 1.25.5

---

## Executive Summary

**The Go MCP SDK v1.2.0 works perfectly with Claude Code CLI.** `mcp.StdioTransport{}` exists, tool registration is clean via generics, and the full roundtrip (server start → tool discovery → invocation → response) completes in a single CLI session. No version upgrade needed.

---

## 1. SDK Version Resolution (Review C-2)

**Resolved:** v1.2.0 is the correct version. It has all required APIs:

| API | Status in v1.2.0 |
|-----|-------------------|
| `mcp.StdioTransport{}` | ✅ Present |
| `mcp.NewServer()` | ✅ Present |
| `mcp.AddTool()` (generic) | ✅ Present |
| `mcp.IOTransport{}` | ✅ Present |
| `mcp.InMemoryTransport` | ✅ Present (for tests) |
| `mcp.NewStreamableHTTPHandler` | ✅ Present |

The v1.3.0 reference in strategy.md was speculative. **No upgrade needed.**

---

## 2. Minimal Server Pattern (~50 lines)

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input type — struct fields become the JSON schema automatically.
// Empty struct = no arguments.
type PingInput struct{}

// Output type — automatically marshaled to structured content.
type PingOutput struct {
    Status    string `json:"status"`
    Timestamp int64  `json:"timestamp"`
}

// Handler signature: ToolHandlerFor[In, Out]
// - ctx: standard context
// - req: raw request (rarely needed)
// - input: auto-unmarshaled from tool arguments
// Returns: (*CallToolResult, Out, error)
//   - Return nil for CallToolResult to let SDK auto-build from Out
//   - Return error to auto-set IsError=true
func pingHandler(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input PingInput,
) (*mcp.CallToolResult, PingOutput, error) {
    return nil, PingOutput{
        Status:    "pong",
        Timestamp: time.Now().Unix(),
    }, nil
}

func main() {
    server := mcp.NewServer(
        &mcp.Implementation{
            Name:    "gofortress-mcp-poc",
            Version: "0.1.0",
        },
        nil, // ServerOptions (nil = defaults)
    )

    // Generic AddTool infers JSON schema from PingInput struct tags
    mcp.AddTool(server, &mcp.Tool{
        Name:        "test_mcp_ping",
        Description: "Returns pong with timestamp",
    }, pingHandler)

    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatalf("MCP server error: %v", err)
    }
}
```

---

## 3. Key API Patterns

### 3.1 Server Creation

```go
server := mcp.NewServer(
    &mcp.Implementation{
        Name:    "server-name",   // Required: used in MCP handshake
        Version: "v1.0.0",       // Reported to client
    },
    &mcp.ServerOptions{
        Logger:    slog.Default(), // Optional: structured logging
        KeepAlive: 30 * time.Second, // Optional: ping interval
    },
)
```

### 3.2 Tool Registration (Generic — Recommended)

```go
// Package-level function, not a method:
mcp.AddTool(server, &mcp.Tool{
    Name:        "tool_name",
    Description: "LLM-visible description",
}, handlerFunc)
```

The generic `AddTool[In, Out]` automatically:
- Infers JSON schema from `In` struct tags
- Validates input against schema before calling handler
- Unmarshals arguments into typed `In`
- Marshals `Out` to structured content in response

### 3.3 Tool Registration (Raw — Full Control)

```go
server.AddTool(&mcp.Tool{
    Name: "raw_tool",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: "response"},
        },
    }, nil
})
```

### 3.4 Input Schema via Struct Tags

```go
type WeatherInput struct {
    City string `json:"city" jsonschema:"description=City name"`
    Days int    `json:"days" jsonschema:"description=Forecast days,minimum=1,maximum=7"`
}
```

The `jsonschema` struct tag maps to JSON Schema properties. Schema is generated at registration time via `github.com/google/jsonschema-go`.

### 3.5 Transports

| Transport | Use Case | Constructor |
|-----------|----------|-------------|
| `StdioTransport{}` | Production: CLI spawns server | Zero-config struct literal |
| `IOTransport{Reader, Writer}` | Custom I/O | Wrap any `io.ReadCloser` / `io.WriteCloser` |
| `InMemoryTransport` | Unit tests | `mcp.NewInMemoryTransports()` returns pair |
| `CommandTransport` | Client-side | Wraps `exec.Command` |

### 3.6 Error Handling

```go
// Return error → auto-wrapped in CallToolResult{IsError: true}
func handler(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (*mcp.CallToolResult, MyOutput, error) {
    return nil, MyOutput{}, fmt.Errorf("something went wrong")
}

// Manual error result:
return &mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: "error details"}},
    IsError: true,
}, MyOutput{}, nil
```

---

## 4. Claude CLI Integration

### 4.1 MCP Config Format

```json
{
  "mcpServers": {
    "server-name": {
      "type": "stdio",
      "command": "/absolute/path/to/binary"
    }
  }
}
```

**Notes:**
- Path must be absolute
- Binary must be pre-built (no `go run`)
- `type: "stdio"` is the only transport Claude CLI supports for local servers

### 4.2 CLI Flags

```bash
claude -p \
  --mcp-config /path/to/mcp-config.json \
  --strict-mcp-config \               # Only use servers from this config
  --allowedTools "mcp__server-name__tool_name" \  # Pre-approve MCP tools
  ...
```

### 4.3 Tool Naming Convention

Claude CLI prefixes MCP tools with `mcp__<server-name>__`:

```
Server name: "gofortress-poc"
Tool name:   "test_mcp_ping"
CLI tool ID: "mcp__gofortress-poc__test_mcp_ping"
```

This is important for `--allowedTools` and for parsing tool_use events in the NDJSON stream.

### 4.4 Permission Behavior

| Permission Mode | MCP Tool Behavior |
|----------------|-------------------|
| `acceptEdits` | ❌ **DENIED** — MCP tools are NOT auto-approved |
| `acceptEdits` + `--allowedTools` | ✅ Approved (must specify full `mcp__x__y` name) |
| `bypassPermissions` | ✅ Approved |
| `default` | ❌ Denied |

**Critical finding:** `acceptEdits` does NOT cover MCP tools. The Go TUI must either:
1. Use `--allowedTools` to pre-approve known MCP tools at CLI startup
2. Use `--dangerously-skip-permissions` (not recommended)
3. Have the user approve MCP tools interactively (not possible in pipe mode)

**Recommended:** Pre-approve all gofortress MCP tools via `--allowedTools "mcp__gofortress__*"` (glob pattern — verify this works).

### 4.5 Discovery in system.init

The MCP server and its tools appear in the `system:init` event:

```json
{
  "type": "system",
  "subtype": "init",
  "tools": ["...", "mcp__gofortress-poc__test_mcp_ping"],
  "mcp_servers": [{"name": "gofortress-poc", "status": "connected"}]
}
```

### 4.6 Tool Invocation in NDJSON Stream

```json
// Claude invokes the tool:
{"type": "assistant", "message": {"content": [
  {"type": "tool_use", "name": "mcp__gofortress-poc__test_mcp_ping", "input": {}}
]}}

// Tool result returned:
{"type": "user", "message": {"content": [
  {"type": "tool_result", "content": "{\"status\":\"pong\",\"timestamp\":1774240595}"}
]}}
```

---

## 5. Gotchas & Sharp Edges

1. **MCP tools need explicit `--allowedTools`** — `acceptEdits` doesn't cover them. This is unlike built-in tools (Read, Write, Bash).

2. **`--strict-mcp-config`** — Without this, Claude CLI also loads MCP servers from `~/.claude/settings.json` and project `.mcp.json`. Use `--strict-mcp-config` to only load from `--mcp-config`.

3. **Binary path must be absolute** — Relative paths in mcp-config.json don't resolve correctly.

4. **Server stderr** — The MCP server's stderr goes to... nowhere visible in stream-json mode. Use `--debug` on the CLI to see MCP server errors. For production, log to a file.

5. **`ToolSearch` is used first** — Claude doesn't have MCP tool schemas loaded until it calls `ToolSearch`. The first invocation of an MCP tool in a session typically has a ToolSearch → tool_use sequence.

6. **Structured output** — The Go SDK's generic handler returns `Out` which gets marshaled as structured JSON in the tool result. Claude sees this as a JSON string in the `content` field.

---

## 6. Test Evidence

### Test 1: Tool Discovery
```
mcp_servers: [{"name": "gofortress-poc", "status": "connected"}]
tools: [..., "mcp__gofortress-poc__test_mcp_ping"]
```
✅ Server connected, tool discovered.

### Test 2: Tool Invocation (with --allowedTools)
```
assistant: tool_use name=mcp__gofortress-poc__test_mcp_ping input={}
user: tool_result err=False "{"status":"pong","timestamp":1774240595}"
result: success=True cost=$0.178 denials=0
```
✅ Tool invoked, correct pong response, no permission denials.

### Test 3: Permission Denial (without --allowedTools)
```
user: tool_result err=True "Claude requested permissions to use mcp__gofortress-poc__test_mcp_ping, but you haven't granted it yet."
permission_denials: [{"tool_name": "mcp__gofortress-poc__test_mcp_ping", "tool_input": {}}]
```
✅ Confirms MCP tools need explicit approval.

---

## 7. Implications for TUI Architecture

| Component | Finding |
|-----------|---------|
| **TUI-014** (MCP server) | SDK v1.2.0 is sufficient. Use generic `mcp.AddTool` for all 7 tools. StdioTransport for production, InMemoryTransport for tests. |
| **TUI-016** (startup) | Must include `--allowedTools "mcp__gofortress__*"` in CLI spawn args. Verify glob pattern support. |
| **TUI-018** (permissions) | MCP tools for interactive prompts (AskUserQuestion, etc.) will be auto-approved via `--allowedTools`, so the TUI controls the interaction flow. |
| **TUI-038** (MCP tests) | Use `mcp.NewInMemoryTransports()` for unit tests — no need to spawn real processes. |

---

## 8. POC Binary Location

```
cmd/gofortress-mcp-poc/main.go  — Source
bin/gofortress-mcp-poc          — Built binary
```

This POC binary can be reused for TUI-004 (IPC spike) and TUI-014 (production MCP server) as a starting point.

---

_Generated by TUI-002 spike, 2026-03-23_
