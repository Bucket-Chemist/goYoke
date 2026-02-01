# MCP Implementation Guide for GOgent-Fortress

**Version:** 1.0
**Status:** Architecture Design
**Author:** Einstein Analysis System
**Date:** 2026-01-27

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Implementation Phases](#implementation-phases)
4. [Technical Specifications](#technical-specifications)
5. [Code Organization](#code-organization)
6. [Tool Catalog Design](#tool-catalog-design)
7. [Configuration Format](#configuration-format)
8. [Implementation Tasks](#implementation-tasks)
9. [Migration Path](#migration-path)
10. [Extension Points](#extension-points)
11. [Operational Considerations](#operational-considerations)

---

## Executive Summary

### Problem

gofortress currently uses AllowedTools for pre-approval, which removes user control over Claude's actions. Users want:
- Interactive prompts for decisions
- Ability to approve/reject actions
- Guidance over tool execution
- Extensible foundation for future features

### Solution

Implement a **custom MCP (Model Context Protocol) server** that:
- Runs embedded in gofortress as a goroutine
- Provides interactive tool: `mcp__gofortress__ask_user`
- Communicates with TUI via Unix sockets + channels
- Enables 10+ custom tools over time
- Maintains single-binary simplicity

### Benefits

| Benefit | Impact |
|---------|--------|
| **Interactive Prompts** | Users can guide Claude's decisions |
| **Production-Grade** | Error handling, timeouts, graceful degradation |
| **Extensible** | Easy to add new tools (3-5 tools per quarter) |
| **Simple Deployment** | Single binary, auto-configuration |
| **Low Latency** | <1ms IPC overhead |
| **Elegant Architecture** | Leverages full SDK capabilities |

### Key Metrics

- **Implementation Time:** 6-8 weeks (4 phases)
- **Test Coverage Target:** >80%
- **Performance:** <100ms prompt display latency
- **Reliability:** Graceful degradation on errors
- **Extensibility:** <50 lines of code per new tool

---

## Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       gofortress                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Main Process                        │   │
│  │                                                        │   │
│  │  ┌──────────────┐         ┌────────────────────┐     │   │
│  │  │ TUI Event    │<───────>│  MCP Server        │     │   │
│  │  │ Loop         │ Channels│  Goroutine         │     │   │
│  │  │ (Bubbletea)  │         │                    │     │   │
│  │  └──────┬───────┘         └────────┬───────────┘     │   │
│  │         │                           │                 │   │
│  │         │ Display                   │ Unix Socket     │   │
│  │         │ Prompts                   │                 │   │
│  │         ▼                           ▼                 │   │
│  │  ┌─────────────┐         ┌────────────────────┐     │   │
│  │  │   Terminal  │         │  /tmp/gofortress-  │     │   │
│  │  │   Output    │         │  mcp.sock          │     │   │
│  │  └─────────────┘         └────────────────────┘     │   │
│  └──────────────────────────────────┬───────────────────┘   │
│                                     │                        │
│                                     │ Spawn                  │
│                                     ▼                        │
│                          ┌─────────────────────┐            │
│                          │   Claude CLI        │            │
│                          │   --output-format   │            │
│                          │   stream-json       │            │
│                          │   --mcp-config      │            │
│                          │   gofortress.json   │            │
│                          └──────────┬──────────┘            │
│                                     │                        │
└─────────────────────────────────────┼────────────────────────┘
                                      │
                                      │ NDJSON
                                      │ Events
                                      ▼
                            ┌──────────────────────┐
                            │  Claude API          │
                            │  (Anthropic)         │
                            └──────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Communication |
|-----------|----------------|---------------|
| **TUI Event Loop** | User interface, display prompts, collect input | Reads from mcpRequests channel, writes to mcpResponses channel |
| **MCP Server Goroutine** | MCP protocol handling, tool routing | Listens on Unix socket, sends/receives via channels |
| **Unix Socket** | IPC transport between MCP server and Claude CLI | Bidirectional JSON-RPC |
| **Tool Registry** | Tool definitions, schema validation, handler dispatch | In-memory registry, thread-safe |
| **Claude CLI** | Claude Code subprocess | stdin (user messages), stdout (NDJSON events) |

### Data Flow: Interactive Prompt

```
1. User: "Should I use SQLite or PostgreSQL?"
        ↓
2. Claude decides to ask user
        ↓
3. Claude calls: mcp__gofortress__ask_user
        ↓
4. MCP protocol over Unix socket
        ↓
5. MCP Server receives tool call
        ↓
6. Validates schema, creates ToolRequest
        ↓
7. Send to toolRequests channel (non-blocking)
        ↓
8. TUI Event Loop receives ToolRequest
        ↓
9. Display prompt with options in TUI
        ↓
10. User presses "1" (SQLite)
        ↓
11. TUI sends ToolResponse to toolResponses channel
        ↓
12. MCP Server receives response
        ↓
13. Format as MCP result
        ↓
14. Send back to Claude CLI via Unix socket
        ↓
15. Claude continues: "I'll use SQLite..."
```

### State Management

**Principle:** TUI owns all user-facing state. MCP server is stateless.

```go
// TUI State (internal/tui/claude/panel.go)
type PanelModel struct {
    // Existing state
    messages      []Message
    streaming     bool

    // MCP integration state
    mcpEnabled    bool
    mcpRequests   <-chan mcp.ToolRequest   // Read from MCP server
    mcpResponses  chan<- mcp.ToolResponse  // Write to MCP server
    awaitingMCP   bool
    currentPrompt *mcp.ToolRequest
    promptTimeout *time.Timer
    promptHistory []PromptRecord           // For debugging/logging
}

// MCP Server State (internal/mcp/server/server.go)
type Server struct {
    // Configuration
    config       Config
    transport    Transport

    // Tool system
    registry     *tools.Registry

    // IPC channels
    toolRequests chan ToolRequest   // Send to TUI
    toolResponses chan ToolResponse // Receive from TUI

    // Lifecycle
    done         chan struct{}
    running      atomic.Bool
}
```

### Error Handling Strategy

```
┌─────────────────────────────────────────────────────┐
│             Error Detection & Recovery               │
├─────────────────────────────────────────────────────┤
│                                                       │
│  MCP Server Startup Failure                          │
│  ├─ Log warning                                      │
│  ├─ Fall back to AllowedTools mode                  │
│  └─ Display: "Interactive prompts unavailable"      │
│                                                       │
│  Unix Socket Creation Failure                        │
│  ├─ Try /tmp/gofortress-mcp-{pid}.sock              │
│  ├─ If still fails, try HTTP transport              │
│  └─ Last resort: AllowedTools mode                  │
│                                                       │
│  Tool Call Timeout (>60s)                           │
│  ├─ Cancel prompt in TUI                            │
│  ├─ Return timeout error to Claude                  │
│  └─ Claude adjusts approach                         │
│                                                       │
│  Invalid Tool Input                                  │
│  ├─ Validate against JSON schema                    │
│  ├─ Return validation error to Claude               │
│  └─ Claude reformulates request                     │
│                                                       │
│  IPC Channel Deadlock                                │
│  ├─ Use buffered channels (size: 10)               │
│  ├─ Timeout on send (1s)                           │
│  └─ Log warning, continue                           │
│                                                       │
│  MCP Server Goroutine Panic                         │
│  ├─ Recover in deferred function                    │
│  ├─ Log stack trace                                 │
│  ├─ Attempt restart (max 3 retries)                │
│  └─ Fall back to AllowedTools if restart fails     │
│                                                       │
└─────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-2)

**Goal:** MCP protocol implementation and basic infrastructure

**Tasks:**
1. Implement MCP JSON-RPC protocol
2. Build Unix socket transport
3. Create tool registry system
4. Set up testing infrastructure

**Deliverables:**
- `internal/mcp/protocol/` - JSON-RPC implementation
- `internal/mcp/transport/unix.go` - Unix socket transport
- `internal/mcp/tools/registry.go` - Tool registry
- Tests with >80% coverage

**Acceptance Criteria:**
- [ ] MCP protocol handles initialize, tools/list, tools/call
- [ ] Unix socket can send/receive JSON-RPC messages
- [ ] Tool registry supports registration and lookup
- [ ] All tests pass

**Agent Assignment:** `go-pro` (Sonnet)

**Example Code:**

```go
// internal/mcp/protocol/jsonrpc.go
package protocol

type Request struct {
    JSONRPC string          `json:"jsonrpc"` // "2.0"
    ID      interface{}     `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
}

type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// internal/mcp/transport/unix.go
package transport

type UnixSocketTransport struct {
    socketPath string
    listener   net.Listener
    conn       net.Conn
    mu         sync.Mutex
}

func (u *UnixSocketTransport) Listen() error {
    // Remove existing socket file
    os.Remove(u.socketPath)

    listener, err := net.Listen("unix", u.socketPath)
    if err != nil {
        return fmt.Errorf("listen on unix socket: %w", err)
    }

    u.listener = listener
    return nil
}

func (u *UnixSocketTransport) Accept() (net.Conn, error) {
    conn, err := u.listener.Accept()
    if err != nil {
        return nil, fmt.Errorf("accept connection: %w", err)
    }

    u.mu.Lock()
    u.conn = conn
    u.mu.Unlock()

    return conn, nil
}
```

---

### Phase 2: Interactive Prompts (Weeks 3-4)

**Goal:** Implement `ask_user` tool and TUI integration

**Tasks:**
1. Define `ask_user` tool schema
2. Implement tool handler with channel IPC
3. Integrate MCP requests into TUI event loop
4. Add prompt rendering in TUI
5. Implement user input handlers for prompts

**Deliverables:**
- `internal/mcp/tools/ask_user.go` - Tool implementation
- `internal/tui/claude/mcp_integration.go` - TUI MCP code
- `internal/tui/claude/prompt_renderer.go` - Prompt UI
- Integration tests

**Acceptance Criteria:**
- [ ] `ask_user` tool registered and callable
- [ ] TUI displays prompts with options
- [ ] User can select options via keyboard
- [ ] Response sent back to Claude
- [ ] Claude continues with user's choice

**Agent Assignment:** `go-tui` (Sonnet)

**Example Code:**

```go
// internal/mcp/tools/ask_user.go
package tools

type AskUserInput struct {
    Question   string   `json:"question"`
    Options    []Option `json:"options"`
    MultiSelect bool    `json:"multiSelect"`
}

type Option struct {
    Label       string `json:"label"`
    Description string `json:"description"`
}

type AskUserOutput struct {
    Answer string `json:"answer"`
}

func init() {
    DefaultRegistry.Register(&ToolDefinition{
        Name: "ask_user",
        Description: "Display an interactive prompt to the user with multiple-choice options",
        InputSchema: generateSchema(AskUserInput{}),
        Handler: handleAskUser,
    })
}

func handleAskUser(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
    var params AskUserInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("parse input: %w", err)
    }

    // Validate input
    if params.Question == "" {
        return nil, fmt.Errorf("question is required")
    }
    if len(params.Options) < 2 || len(params.Options) > 4 {
        return nil, fmt.Errorf("must have 2-4 options, got %d", len(params.Options))
    }

    // Create tool request
    req := ToolRequest{
        ID:      uuid.New().String(),
        Tool:    "ask_user",
        Input:   params,
        Created: time.Now(),
    }

    // Send to TUI (with timeout)
    select {
    case toolRequests <- req:
        // Sent successfully
    case <-time.After(1 * time.Second):
        return nil, fmt.Errorf("TUI not responding")
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // Wait for response (with timeout)
    select {
    case resp := <-toolResponses:
        if resp.ID != req.ID {
            return nil, fmt.Errorf("response ID mismatch")
        }
        if resp.Error != nil {
            return nil, resp.Error
        }
        return &ToolResult{
            Content: []ContentBlock{{
                Type: "text",
                Text: resp.Result,
            }},
        }, nil

    case <-time.After(60 * time.Second):
        return nil, fmt.Errorf("user response timeout (60s)")

    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// internal/tui/claude/mcp_integration.go
package claude

func (m PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Check for MCP tool requests (non-blocking)
    select {
    case req := <-m.mcpRequests:
        return m.handleMCPRequest(req)
    default:
        // Continue with normal event handling
    }

    // ... existing Update logic ...
}

func (m PanelModel) handleMCPRequest(req mcp.ToolRequest) (PanelModel, tea.Cmd) {
    m.awaitingMCP = true
    m.currentPrompt = &req

    // Add prompt to messages for display
    m.messages = append(m.messages, Message{
        Role: "prompt",
        Content: formatPrompt(req),
    })
    m.updateViewport()

    // Set timeout timer
    m.promptTimeout = time.AfterFunc(60*time.Second, func() {
        // Send timeout response
        m.mcpResponses <- mcp.ToolResponse{
            ID:    req.ID,
            Error: fmt.Errorf("user response timeout"),
        }
    })

    return m, nil
}

// Handle user input when awaiting MCP response
func (m PanelModel) handleInput(msg tea.KeyMsg) (PanelModel, tea.Cmd) {
    if m.awaitingMCP && m.currentPrompt != nil {
        switch msg.String() {
        case "1", "2", "3", "4":
            return m.sendMCPResponse(msg.String())
        case "esc":
            return m.cancelMCPPrompt()
        }
        return m, nil // Ignore other keys while awaiting
    }

    // ... existing input handling ...
}

func (m PanelModel) sendMCPResponse(choice string) (PanelModel, tea.Cmd) {
    if m.currentPrompt == nil {
        return m, nil
    }

    // Parse choice
    input := m.currentPrompt.Input.(AskUserInput)
    choiceIdx, _ := strconv.Atoi(choice)
    choiceIdx-- // 0-indexed

    if choiceIdx < 0 || choiceIdx >= len(input.Options) {
        return m, nil // Invalid choice
    }

    answer := input.Options[choiceIdx].Label

    // Cancel timeout
    if m.promptTimeout != nil {
        m.promptTimeout.Stop()
    }

    // Send response
    m.mcpResponses <- mcp.ToolResponse{
        ID:     m.currentPrompt.ID,
        Result: answer,
    }

    // Clear prompt state
    m.awaitingMCP = false
    m.currentPrompt = nil
    m.promptTimeout = nil

    // Add confirmation message
    m.messages = append(m.messages, Message{
        Role: "system",
        Content: fmt.Sprintf("✓ Selected: %s", answer),
    })
    m.updateViewport()

    return m, nil
}
```

---

### Phase 3: Production Hardening (Weeks 5-6)

**Goal:** Error handling, testing, reliability

**Tasks:**
1. Comprehensive error handling
2. Timeout management
3. Graceful degradation (fallback to AllowedTools)
4. Unit tests (>80% coverage)
5. Integration tests with mock Claude CLI
6. Performance optimization

**Deliverables:**
- Error handling in all components
- Test suite with >80% coverage
- Performance benchmarks
- Graceful degradation logic

**Acceptance Criteria:**
- [ ] All error scenarios handled
- [ ] Tests achieve >80% coverage
- [ ] Graceful degradation works
- [ ] <100ms prompt display latency
- [ ] No goroutine leaks

**Agent Assignment:** `go-pro` (Sonnet)

**Error Handling Example:**

```go
// cmd/gofortress/main.go
func startMCPServer(cfg mcp.Config) (*mcp.Server, error) {
    server := mcp.NewServer(cfg)

    // Try to start with retries
    var err error
    for attempt := 1; attempt <= 3; attempt++ {
        if err = server.Start(); err == nil {
            return server, nil
        }

        log.Printf("MCP server start attempt %d failed: %v", attempt, err)

        // Try alternative socket path
        if attempt == 2 {
            cfg.SocketPath = fmt.Sprintf("/tmp/gofortress-mcp-%d.sock", os.Getpid())
        }

        time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
    }

    return nil, fmt.Errorf("failed to start MCP server after 3 attempts: %w", err)
}

func main() {
    // Attempt to start MCP server
    mcpServer, err := startMCPServer(mcp.DefaultConfig())
    if err != nil {
        fmt.Fprintf(os.Stderr, "⚠️  MCP server unavailable: %v\n", err)
        fmt.Fprintf(os.Stderr, "    Interactive prompts disabled\n")
        fmt.Fprintf(os.Stderr, "    Using auto-approval mode\n\n")
        // Continue without MCP
    }

    // Configure Claude CLI
    claudeCfg := cli.Config{
        AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task"},
    }

    if mcpServer != nil {
        claudeCfg.MCPConfig = mcpServer.ConfigPath()
        claudeCfg.AllowedTools = append(claudeCfg.AllowedTools, "mcp__gofortress__ask_user")
        defer mcpServer.Stop()
    }

    // ... rest of startup ...
}
```

---

### Phase 4: Extensibility (Weeks 7-8)

**Goal:** Plugin system, documentation, examples

**Tasks:**
1. HTTP transport implementation
2. Plugin system design
3. Tool developer documentation
4. Example custom tools
5. Performance testing
6. User documentation

**Deliverables:**
- HTTP transport support
- `CUSTOM_TOOLS.md` documentation
- 3+ example tools
- User guide

**Acceptance Criteria:**
- [ ] HTTP transport works
- [ ] Developers can add tools in <50 LOC
- [ ] Examples demonstrate key patterns
- [ ] User documentation complete

**Agent Assignment:** `go-pro` (Sonnet), `tech-docs-writer` (Haiku+Thinking)

**Example Custom Tool:**

```go
// examples/custom-tools/confirm_action.go
package main

import (
    "context"
    "encoding/json"
    "github.com/yourusername/GOgent-Fortress/internal/mcp/tools"
)

type ConfirmActionInput struct {
    Action      string `json:"action"`
    Description string `json:"description"`
    Dangerous   bool   `json:"dangerous"`
}

type ConfirmActionOutput struct {
    Confirmed bool   `json:"confirmed"`
    Reason    string `json:"reason,omitempty"`
}

func init() {
    tools.DefaultRegistry.Register(&tools.ToolDefinition{
        Name: "confirm_action",
        Description: "Ask user to confirm a potentially dangerous action",
        InputSchema: tools.GenerateSchema(ConfirmActionInput{}),
        Handler: handleConfirmAction,
    })
}

func handleConfirmAction(ctx context.Context, input json.RawMessage) (*tools.ToolResult, error) {
    var params ConfirmActionInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, err
    }

    // Display confirmation prompt
    question := fmt.Sprintf("Confirm: %s?", params.Action)
    if params.Description != "" {
        question += fmt.Sprintf("\n%s", params.Description)
    }

    options := []tools.Option{
        {Label: "Yes", Description: "Proceed with action"},
        {Label: "No", Description: "Cancel action"},
    }

    // Use ask_user under the hood
    result, err := tools.AskUser(ctx, question, options, false)
    if err != nil {
        return nil, err
    }

    confirmed := result == "Yes"

    return &tools.ToolResult{
        Content: []tools.ContentBlock{{
            Type: "text",
            Text: fmt.Sprintf(`{"confirmed": %t}`, confirmed),
        }},
    }, nil
}
```

---

## Technical Specifications

### MCP Protocol Implementation

**JSON-RPC 2.0 Messages:**

```go
// internal/mcp/protocol/messages.go
package protocol

// Initialize request
type InitializeParams struct {
    ProtocolVersion string            `json:"protocolVersion"` // "2024-11-05"
    Capabilities    ClientCapabilities `json:"capabilities"`
    ClientInfo      ClientInfo         `json:"clientInfo"`
}

type ClientCapabilities struct {
    Sampling map[string]interface{} `json:"sampling,omitempty"`
}

type ClientInfo struct {
    Name    string `json:"name"`    // "claude-cli"
    Version string `json:"version"` // e.g., "0.1.0"
}

// Initialize response
type InitializeResult struct {
    ProtocolVersion  string              `json:"protocolVersion"`
    Capabilities     ServerCapabilities  `json:"capabilities"`
    ServerInfo       ServerInfo          `json:"serverInfo"`
}

type ServerCapabilities struct {
    Tools map[string]interface{} `json:"tools,omitempty"`
}

type ServerInfo struct {
    Name    string `json:"name"`    // "gofortress-mcp"
    Version string `json:"version"` // "1.0.0"
}

// tools/list request (no params)

// tools/list response
type ListToolsResult struct {
    Tools []ToolInfo `json:"tools"`
}

type ToolInfo struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema interface{} `json:"inputSchema"` // JSON Schema
}

// tools/call request
type CallToolParams struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments,omitempty"`
}

// tools/call response
type CallToolResult struct {
    Content []ContentItem `json:"content"`
    IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
    Type string `json:"type"` // "text", "image", "resource"
    Text string `json:"text,omitempty"`
    Data string `json:"data,omitempty"`
    MimeType string `json:"mimeType,omitempty"`
}
```

**Protocol Handler:**

```go
// internal/mcp/server/handler.go
package server

func (s *Server) handleRequest(req *protocol.Request) (*protocol.Response, error) {
    switch req.Method {
    case "initialize":
        return s.handleInitialize(req)
    case "tools/list":
        return s.handleToolsList(req)
    case "tools/call":
        return s.handleToolsCall(req)
    default:
        return nil, &protocol.Error{
            Code:    -32601,
            Message: fmt.Sprintf("method not found: %s", req.Method),
        }
    }
}

func (s *Server) handleInitialize(req *protocol.Request) (*protocol.Response, error) {
    var params protocol.InitializeParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return nil, &protocol.Error{
            Code:    -32602,
            Message: "invalid params",
            Data:    err.Error(),
        }
    }

    result := protocol.InitializeResult{
        ProtocolVersion: "2024-11-05",
        Capabilities: protocol.ServerCapabilities{
            Tools: map[string]interface{}{},
        },
        ServerInfo: protocol.ServerInfo{
            Name:    "gofortress-mcp",
            Version: "1.0.0",
        },
    }

    resultBytes, _ := json.Marshal(result)
    return &protocol.Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  resultBytes,
    }, nil
}

func (s *Server) handleToolsList(req *protocol.Request) (*protocol.Response, error) {
    tools := s.registry.List()

    result := protocol.ListToolsResult{
        Tools: make([]protocol.ToolInfo, len(tools)),
    }

    for i, tool := range tools {
        result.Tools[i] = protocol.ToolInfo{
            Name:        tool.Name,
            Description: tool.Description,
            InputSchema: tool.InputSchema,
        }
    }

    resultBytes, _ := json.Marshal(result)
    return &protocol.Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  resultBytes,
    }, nil
}

func (s *Server) handleToolsCall(req *protocol.Request) (*protocol.Response, error) {
    var params protocol.CallToolParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return nil, &protocol.Error{
            Code:    -32602,
            Message: "invalid params",
        }
    }

    // Look up tool
    tool, err := s.registry.Get(params.Name)
    if err != nil {
        return nil, &protocol.Error{
            Code:    -32602,
            Message: fmt.Sprintf("tool not found: %s", params.Name),
        }
    }

    // Call handler
    ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
    defer cancel()

    result, err := tool.Handler(ctx, params.Arguments)
    if err != nil {
        return nil, &protocol.Error{
            Code:    -32000,
            Message: fmt.Sprintf("tool execution error: %v", err),
        }
    }

    // Format response
    callResult := protocol.CallToolResult{
        Content: []protocol.ContentItem{},
        IsError: false,
    }

    for _, block := range result.Content {
        callResult.Content = append(callResult.Content, protocol.ContentItem{
            Type: block.Type,
            Text: block.Text,
        })
    }

    resultBytes, _ := json.Marshal(callResult)
    return &protocol.Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  resultBytes,
    }, nil
}
```

---

## Code Organization

### Proposed Directory Structure

```
GOgent-Fortress/
├── cmd/
│   ├── gofortress/                    # Main TUI binary
│   │   ├── main.go                    # Entry point, MCP server startup
│   │   └── config.go                  # Configuration loading
│   └── gofortress-mcp/                # Standalone MCP server (optional, future)
│       └── main.go
│
├── internal/
│   ├── mcp/
│   │   ├── server/
│   │   │   ├── server.go              # MCP server implementation
│   │   │   ├── handler.go             # Request handler
│   │   │   ├── server_test.go
│   │   │   └── handler_test.go
│   │   ├── protocol/
│   │   │   ├── jsonrpc.go             # JSON-RPC types
│   │   │   ├── messages.go            # MCP message types
│   │   │   ├── jsonrpc_test.go
│   │   │   └── messages_test.go
│   │   ├── transport/
│   │   │   ├── transport.go           # Transport interface
│   │   │   ├── unix.go                # Unix socket implementation
│   │   │   ├── http.go                # HTTP implementation (future)
│   │   │   ├── unix_test.go
│   │   │   └── http_test.go
│   │   └── tools/
│   │       ├── registry.go            # Tool registry
│   │       ├── ask_user.go            # ask_user tool
│   │       ├── confirm_action.go      # confirm_action tool (future)
│   │       ├── types.go               # Shared types
│   │       ├── schema.go              # JSON schema generation
│   │       ├── registry_test.go
│   │       └── ask_user_test.go
│   │
│   ├── ipc/
│   │   ├── channels.go                # Channel-based IPC
│   │   └── channels_test.go
│   │
│   ├── tui/
│   │   └── claude/
│   │       ├── panel.go               # Main TUI model
│   │       ├── events.go              # Event handlers
│   │       ├── input.go               # User input
│   │       ├── output.go              # Rendering
│   │       ├── mcp_integration.go     # MCP-specific code
│   │       ├── prompt_renderer.go     # Prompt UI rendering
│   │       ├── panel_test.go
│   │       └── mcp_integration_test.go
│   │
│   └── cli/
│       ├── subprocess.go              # Claude CLI wrapper
│       └── ...                        # Existing CLI code
│
├── pkg/
│   └── mcptools/                      # Public API for custom tools
│       ├── tool.go                    # Tool definition helpers
│       ├── registry.go                # Public registry access
│       └── examples.go                # Example patterns
│
├── examples/
│   └── custom-tools/
│       ├── confirm_action/
│       │   ├── main.go
│       │   └── README.md
│       ├── select_file/
│       │   ├── main.go
│       │   └── README.md
│       └── ...
│
├── docs/
│   ├── MCP_IMPLEMENTATION_GUIDE.md    # This file
│   ├── CUSTOM_TOOLS.md                # Tool developer guide
│   ├── USER_GUIDE.md                  # End-user documentation
│   └── ARCHITECTURE.md                # System architecture
│
├── configs/
│   └── gofortress-mcp.json            # Default MCP config
│
└── README.md
```

### Package Responsibilities

| Package | Responsibility | Dependencies |
|---------|----------------|--------------|
| `cmd/gofortress` | Main entry point, orchestration | `internal/mcp/server`, `internal/tui/claude`, `internal/cli` |
| `internal/mcp/server` | MCP server lifecycle | `internal/mcp/protocol`, `internal/mcp/transport`, `internal/mcp/tools` |
| `internal/mcp/protocol` | MCP protocol implementation | Standard library only |
| `internal/mcp/transport` | Transport abstractions | Standard library, `net` |
| `internal/mcp/tools` | Tool registry and implementations | `internal/ipc` |
| `internal/ipc` | Channel-based IPC | Standard library |
| `internal/tui/claude` | TUI implementation | `github.com/charmbracelet/bubbletea`, `internal/mcp/tools` |
| `pkg/mcptools` | Public tool API | `internal/mcp/tools` |

---

## Tool Catalog Design

### Built-in Tools

#### 1. `ask_user` - Interactive Prompts

**Purpose:** Display multiple-choice questions to users

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "question": {
      "type": "string",
      "description": "The question to ask the user"
    },
    "options": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "label": {"type": "string"},
          "description": {"type": "string"}
        },
        "required": ["label", "description"]
      },
      "minItems": 2,
      "maxItems": 4
    },
    "multiSelect": {
      "type": "boolean",
      "description": "Allow multiple selections",
      "default": false
    }
  },
  "required": ["question", "options"]
}
```

**Output:**
```json
{
  "answer": "selected_label"  // Or comma-separated for multiSelect
}
```

**Example Usage (from Claude's perspective):**
```
Claude: I need to choose a database. Let me ask the user.

Tool: mcp__gofortress__ask_user
Input: {
  "question": "Which database should I use?",
  "options": [
    {"label": "SQLite", "description": "Lightweight, file-based"},
    {"label": "PostgreSQL", "description": "Full-featured, scalable"}
  ]
}

Result: {"answer": "SQLite"}

Claude: I'll use SQLite as requested...
```

---

#### 2. `confirm_action` - Yes/No Confirmations

**Purpose:** Confirm potentially dangerous or irreversible actions

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "description": "The action to confirm (e.g., 'Delete all test files')"
    },
    "description": {
      "type": "string",
      "description": "Additional context about the action"
    },
    "dangerous": {
      "type": "boolean",
      "description": "Mark as dangerous (displays warning)",
      "default": false
    }
  },
  "required": ["action"]
}
```

**Output:**
```json
{
  "confirmed": true  // or false
}
```

**Example Usage:**
```
Claude: I'm about to delete 50 test files. Let me confirm.

Tool: mcp__gofortress__confirm_action
Input: {
  "action": "Delete 50 test files",
  "description": "This will remove all files matching *_test.go",
  "dangerous": true
}

Result: {"confirmed": true}

Claude: Proceeding with deletion...
```

---

### Future Tools (Extensibility Examples)

#### 3. `select_file` - File Picker

**Purpose:** Let user select a file from TUI file browser

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "directory": {"type": "string", "description": "Starting directory"},
    "pattern": {"type": "string", "description": "File pattern (e.g., '*.go')"},
    "allowMultiple": {"type": "boolean", "default": false}
  }
}
```

**Output:**
```json
{
  "selected": "/path/to/file.go"  // or array for multiple
}
```

---

#### 4. `get_preference` - Persistent Preferences

**Purpose:** Store and retrieve user preferences

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "key": {"type": "string"},
    "prompt": {"type": "string"},
    "default": {"type": "string"}
  },
  "required": ["key", "prompt"]
}
```

**Output:**
```json
{
  "value": "user_preference_value"
}
```

**Example:** "What's your preferred Go test framework?" → saves to `~/.config/gofortress/preferences.json`

---

#### 5. `run_test` - Interactive Test Runner

**Purpose:** Display test results and let user choose actions

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "testCommand": {"type": "string"},
    "failures": {"type": "array", "items": {"type": "string"}}
  }
}
```

**Output:**
```json
{
  "action": "retry_failed" | "debug_first" | "continue"
}
```

---

#### 6. `review_diff` - Code Review Prompts

**Purpose:** Display diff and collect review feedback

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "diff": {"type": "string"},
    "file": {"type": "string"}
  }
}
```

**Output:**
```json
{
  "approved": true,
  "comments": "optional feedback"
}
```

---

### Tool Registry Interface

```go
// pkg/mcptools/tool.go
package mcptools

// ToolDefinition defines a custom tool
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema interface{} // JSON Schema or Go struct
    Handler     ToolHandler
}

// ToolHandler processes tool calls
type ToolHandler func(ctx context.Context, input json.RawMessage) (*ToolResult, error)

// ToolResult is returned by handlers
type ToolResult struct {
    Content []ContentBlock
}

type ContentBlock struct {
    Type string // "text", "image"
    Text string
    Data []byte
}

// Register a custom tool
func Register(def *ToolDefinition) error {
    return tools.DefaultRegistry.Register(def)
}

// Helper to ask user (for use within other tools)
func AskUser(ctx context.Context, question string, options []Option, multiSelect bool) (string, error) {
    // Implementation calls ask_user tool internally
}
```

---

## Configuration Format

### gofortress Configuration File

**Location:** `~/.config/gofortress/config.yaml` or `--config` flag

```yaml
# gofortress configuration
version: 1

# Claude CLI settings
claude:
  model: sonnet
  max_turns: 100
  working_dir: "."

# MCP server settings
mcp:
  enabled: true

  # Server configuration
  server:
    transport: unix-socket  # or "http"
    socket_path: /tmp/gofortress-mcp.sock
    http_port: 8765  # if transport=http

  # Enabled tools
  tools:
    - ask_user
    - confirm_action
    # Add more as implemented

  # IPC configuration
  ipc:
    channel_buffer_size: 10
    request_timeout: 1s
    response_timeout: 60s

  # Fallback behavior
  fallback:
    on_error: allow_tools  # or "deny_tools"
    allowed_tools:
      - Bash
      - Read
      - Write
      - Edit
      - Glob
      - Grep
      - Task

# TUI settings
tui:
  theme: dark
  show_thinking: true

# Logging
logging:
  level: info  # debug, info, warn, error
  file: ~/.local/state/gofortress/gofortress.log
```

### MCP Server Config (Generated Per-Instance)

**IMPORTANT:** MCP config is generated PER gofortress instance in `/tmp`, NOT globally registered. This prevents interference with regular `goclaude` or `claude` CLI usage.

**Location:** `/tmp/gofortress-mcp-{PID}.json` (ephemeral, unique per instance)

**Generated automatically by gofortress on startup:**

```json
{
  "mcpServers": {
    "gofortress": {
      "command": "/tmp/gofortress-mcp-{PID}.sock",
      "transport": "unix",
      "env": {},
      "args": []
    }
  }
}
```

**Passed to Claude CLI via:** `--mcp-config /tmp/gofortress-mcp-{PID}.json`

**Cleanup:** Removed automatically on gofortress exit

**Isolation Guarantee:**
- Regular `goclaude` sessions: Unaffected (no MCP config)
- Other `gofortress` instances: Each gets unique config
- No global MCP registration: Uses `--mcp-config` flag, NOT `claude mcp add`

---

## Implementation Tasks

### Complete Task Breakdown

#### Phase 1: Foundation

**Task 1.1: MCP Protocol Implementation**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/protocol/*.go`
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** None

**Subtasks:**
1. Implement JSON-RPC 2.0 request/response types
2. Implement MCP message types (initialize, tools/list, tools/call)
3. Write request parser
4. Write response formatter
5. Unit tests (>80% coverage)

**Acceptance:**
- [ ] All MCP message types defined
- [ ] Request parser handles all formats
- [ ] Response formatter produces valid JSON-RPC
- [ ] Tests pass with >80% coverage

---

**Task 1.2: Unix Socket Transport**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/transport/*.go`
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** None

**Subtasks:**
1. Define Transport interface
2. Implement UnixSocketTransport
3. Handle connection errors gracefully
4. Add connection timeout handling
5. Unit tests

**Acceptance:**
- [ ] Transport interface defined
- [ ] Unix socket listen/accept/send/receive works
- [ ] Error handling for missing socket, permissions
- [ ] Tests with mock connections

---

**Task 1.3: Tool Registry**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/tools/registry.go`, `pkg/mcptools/tool.go`
- **Complexity:** Low
- **Time:** 1-2 days
- **Dependencies:** None

**Subtasks:**
1. Define ToolDefinition struct
2. Implement Registry with thread-safe operations
3. Add Register/Get/List methods
4. Create public API in pkg/mcptools
5. Unit tests

**Acceptance:**
- [ ] Registry supports registration and lookup
- [ ] Thread-safe (can register from multiple goroutines)
- [ ] List returns all registered tools
- [ ] Tests verify concurrency safety

---

**Task 1.4: Testing Infrastructure**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/server/server_test.go`, test utilities
- **Complexity:** Medium
- **Time:** 2 days
- **Dependencies:** Tasks 1.1, 1.2

**Subtasks:**
1. Create mock Transport for tests
2. Create mock Tool handlers
3. Write integration test helpers
4. Set up test fixtures
5. Configure test coverage reporting

**Acceptance:**
- [ ] Mock transport works in tests
- [ ] Can test MCP server without real sockets
- [ ] Test helpers simplify test writing
- [ ] Coverage reporting configured

---

#### Phase 2: Interactive Prompts

**Task 2.1: ask_user Tool Implementation**
- **Owner:** go-tui (Sonnet)
- **Files:** `internal/mcp/tools/ask_user.go`
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** Task 1.3 (Registry)

**Subtasks:**
1. Define AskUserInput/Output types
2. Generate JSON schema
3. Implement handler with channel IPC
4. Add input validation
5. Handle timeout (60s)
6. Unit tests

**Acceptance:**
- [ ] Tool registers successfully
- [ ] Input validation works
- [ ] Handler sends request to channel
- [ ] Handler waits for response with timeout
- [ ] Tests mock channel communication

---

**Task 2.2: MCP Server Main Loop**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/server/server.go`
- **Complexity:** High
- **Time:** 3-4 days
- **Dependencies:** Tasks 1.1, 1.2, 1.3, 2.1

**Subtasks:**
1. Implement Server struct
2. Start/Stop lifecycle methods
3. Accept connections in goroutine
4. Handle requests in goroutine per connection
5. Route requests to handlers
6. Send responses back to client
7. Handle errors and panics
8. Integration tests

**Acceptance:**
- [ ] Server starts and listens
- [ ] Accepts connections
- [ ] Routes requests correctly
- [ ] Returns valid responses
- [ ] Handles errors gracefully
- [ ] Tests verify full request/response cycle

---

**Task 2.3: TUI MCP Integration**
- **Owner:** go-tui (Sonnet)
- **Files:** `internal/tui/claude/mcp_integration.go`, `panel.go`
- **Complexity:** High
- **Time:** 3-4 days
- **Dependencies:** Task 2.1, 2.2

**Subtasks:**
1. Add MCP channels to PanelModel
2. Monitor mcpRequests channel in Update loop
3. Implement handleMCPRequest
4. Add prompt state management
5. Set timeout timers
6. Integration with existing event loop
7. Unit tests

**Acceptance:**
- [ ] TUI receives MCP requests
- [ ] Prompt state managed correctly
- [ ] Timeout handling works
- [ ] No blocking in event loop
- [ ] Tests mock MCP requests

---

**Task 2.4: Prompt Rendering**
- **Owner:** go-tui (Sonnet)
- **Files:** `internal/tui/claude/prompt_renderer.go`, `output.go`
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** Task 2.3

**Subtasks:**
1. Design prompt UI with lipgloss
2. Render question text
3. Render options with numbers
4. Show keyboard shortcuts
5. Add awaiting indicator
6. Style for different prompt types
7. Visual tests

**Acceptance:**
- [ ] Prompts display clearly
- [ ] Options numbered correctly
- [ ] Keyboard shortcuts shown
- [ ] Awaiting indicator visible
- [ ] Lipgloss styles applied

---

**Task 2.5: User Input Handling**
- **Owner:** go-tui (Sonnet)
- **Files:** `internal/tui/claude/input.go`
- **Complexity:** Medium
- **Time:** 2 days
- **Dependencies:** Task 2.3, 2.4

**Subtasks:**
1. Detect when awaiting MCP response
2. Handle number key presses (1-4)
3. Handle ESC for cancellation
4. Send response to mcpResponses channel
5. Clear prompt state after response
6. Add confirmation message
7. Unit tests

**Acceptance:**
- [ ] Number keys select options
- [ ] ESC cancels prompt
- [ ] Response sent to channel
- [ ] State cleared correctly
- [ ] Confirmation message displays

---

**Task 2.6: End-to-End Integration**
- **Owner:** go-tui (Sonnet)
- **Files:** `cmd/gofortress/main.go`
- **Complexity:** Medium
- **Time:** 2 days
- **Dependencies:** Tasks 2.2, 2.3, 2.5

**Subtasks:**
1. Start MCP server in main
2. Create channels for IPC
3. Pass channels to TUI and MCP server
4. Configure Claude CLI with MCP config (ISOLATED, not global)
5. Add mcp__gofortress__ask_user to AllowedTools
6. Manual testing
7. Integration tests

**Acceptance:**
- [ ] gofortress starts with MCP server
- [ ] Claude CLI connects to MCP server
- [ ] ask_user tool calls work end-to-end
- [ ] User can see and respond to prompts
- [ ] Claude continues with user's choice
- [ ] Regular `goclaude` sessions unaffected (CRITICAL)

**Implementation Example:**

```go
// cmd/gofortress/main.go
func main() {
    // Get unique paths for this instance
    pid := os.Getpid()
    socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-mcp-%d.sock", pid))
    mcpConfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-mcp-%d.json", pid))

    // Start embedded MCP server
    mcpServer, err := mcp.NewServer(mcp.Config{
        Transport:  "unix-socket",
        SocketPath: socketPath,
    })
    if err != nil {
        log.Warn("MCP server failed to start: %v", err)
        // Fall back to AllowedTools only
    } else {
        go mcpServer.Start()
        defer mcpServer.Stop()
        defer os.Remove(socketPath)

        // Generate MCP config for THIS gofortress instance only
        mcpConfigJSON := fmt.Sprintf(`{
          "mcpServers": {
            "gofortress": {
              "command": "%s",
              "transport": "unix"
            }
          }
        }`, socketPath)

        if err := os.WriteFile(mcpConfigPath, []byte(mcpConfigJSON), 0600); err != nil {
            log.Warn("Failed to write MCP config: %v", err)
        }
        defer os.Remove(mcpConfigPath)
    }

    // Configure Claude CLI
    claudeCfg := cli.Config{
        AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task"},
    }

    // Add MCP config ONLY if server started successfully
    if mcpServer != nil {
        claudeCfg.MCPConfig = mcpConfigPath  // Isolated, not global!
        claudeCfg.AllowedTools = append(claudeCfg.AllowedTools, "mcp__gofortress__ask_user")
    }

    // Start TUI with channels connected to MCP server
    // ... rest of startup
}
```

**Key Points:**
- ✅ Uses PID in paths (unique per instance)
- ✅ Temporary files in `/tmp` (auto-cleaned)
- ✅ Uses `--mcp-config` flag (NOT `claude mcp add`)
- ✅ Removes files on exit (`defer`)
- ✅ Zero impact on global Claude CLI configuration

---

#### Phase 3: Production Hardening

**Task 3.1: Comprehensive Error Handling**
- **Owner:** go-pro (Sonnet)
- **Files:** All components
- **Complexity:** High
- **Time:** 3-4 days
- **Dependencies:** Phase 2 complete

**Subtasks:**
1. Add error handling to all public methods
2. Wrap errors with context
3. Handle goroutine panics
4. Add timeout handling everywhere
5. Log errors appropriately
6. Return user-friendly error messages
7. Tests for error scenarios

**Acceptance:**
- [ ] All errors have context
- [ ] Panics recovered gracefully
- [ ] Timeouts handled uniformly
- [ ] Error messages are clear
- [ ] Tests cover error paths

---

**Task 3.2: Graceful Degradation**
- **Owner:** go-pro (Sonnet)
- **Files:** `cmd/gofortress/main.go`, fallback logic
- **Complexity:** Medium
- **Time:** 2 days
- **Dependencies:** Task 3.1

**Subtasks:**
1. Detect MCP server startup failure
2. Fall back to AllowedTools mode
3. Display warning to user
4. Log degradation reason
5. Attempt MCP restart on errors
6. Tests for fallback scenarios

**Acceptance:**
- [ ] Startup failure detected
- [ ] Falls back to AllowedTools
- [ ] User notified clearly
- [ ] Can restart MCP server
- [ ] Tests verify fallback works

---

**Task 3.3: Comprehensive Testing**
- **Owner:** go-pro (Sonnet)
- **Files:** All test files
- **Complexity:** High
- **Time:** 4-5 days
- **Dependencies:** All Phase 2 tasks

**Subtasks:**
1. Write missing unit tests
2. Write integration tests
3. Write end-to-end tests with mock Claude CLI
4. Achieve >80% coverage
5. Add benchmark tests
6. Performance profiling
7. Fix any discovered issues

**Acceptance:**
- [ ] Coverage >80% for all packages
- [ ] Integration tests pass
- [ ] E2E tests with mock CLI work
- [ ] No flaky tests
- [ ] Performance acceptable

---

**Task 3.4: Performance Optimization**
- **Owner:** go-pro (Sonnet)
- **Files:** Critical path code
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** Task 3.3 (profiling)

**Subtasks:**
1. Profile MCP server
2. Profile TUI event loop
3. Optimize hot paths
4. Reduce allocations
5. Optimize channel operations
6. Benchmark improvements
7. Document performance characteristics

**Acceptance:**
- [ ] <100ms prompt display latency
- [ ] <1ms IPC overhead
- [ ] No goroutine leaks
- [ ] Memory usage acceptable
- [ ] Benchmarks show improvements

---

#### Phase 4: Extensibility

**Task 4.1: HTTP Transport**
- **Owner:** go-pro (Sonnet)
- **Files:** `internal/mcp/transport/http.go`
- **Complexity:** Medium
- **Time:** 2-3 days
- **Dependencies:** Task 1.2 (Transport interface)

**Subtasks:**
1. Implement HTTPTransport
2. HTTP server setup
3. Handle POST requests
4. JSON-RPC over HTTP
5. Error handling
6. Configuration
7. Tests

**Acceptance:**
- [ ] HTTP transport implements interface
- [ ] Server listens on configured port
- [ ] Handles JSON-RPC requests
- [ ] Tests verify HTTP flow

---

**Task 4.2: Plugin System Design**
- **Owner:** go-pro (Sonnet) + architect (Sonnet)
- **Files:** `pkg/mcptools/*.go`, design docs
- **Complexity:** High
- **Time:** 3-4 days
- **Dependencies:** Phase 3 complete

**Subtasks:**
1. Design plugin API
2. Document tool creation process
3. Create tool template/generator
4. Design schema generation from structs
5. Add helper functions
6. Write developer guide
7. Create example plugin

**Acceptance:**
- [ ] Plugin API well-defined
- [ ] Developers can create tools easily
- [ ] Schema generation works
- [ ] Documentation complete
- [ ] Example demonstrates patterns

---

**Task 4.3: Example Custom Tools**
- **Owner:** go-pro (Sonnet)
- **Files:** `examples/custom-tools/*`
- **Complexity:** Low-Medium
- **Time:** 3-4 days
- **Dependencies:** Task 4.2

**Subtasks:**
1. Implement confirm_action tool
2. Implement select_file tool
3. Implement get_preference tool
4. Write README for each
5. Create integration tests
6. Document patterns used

**Acceptance:**
- [ ] 3+ example tools implemented
- [ ] Each has README
- [ ] Examples demonstrate key patterns
- [ ] Tests verify functionality

---

**Task 4.4: Documentation**
- **Owner:** tech-docs-writer (Haiku+Thinking)
- **Files:** `docs/*.md`, `README.md`
- **Complexity:** Medium
- **Time:** 3-4 days
- **Dependencies:** All other tasks

**Subtasks:**
1. Write CUSTOM_TOOLS.md (developer guide)
2. Write USER_GUIDE.md (end-user guide)
3. Update README.md
4. Write ARCHITECTURE.md
5. Add diagrams
6. Write troubleshooting guide
7. Review and polish

**Acceptance:**
- [ ] Documentation complete
- [ ] Examples clear
- [ ] Diagrams included
- [ ] Troubleshooting guide helpful
- [ ] User feedback positive

---

## Migration Path

### For Existing gofortress Users

**Goal:** Seamless upgrade with no breaking changes

#### Backward Compatibility

**Scenario 1: User upgrades gofortress binary**

```
Old: gofortress with AllowedTools only
New: gofortress with MCP support

Migration: Auto-detected, MCP enabled by default
Fallback: If MCP fails, uses AllowedTools mode
User Impact: None (works the same or better)
```

**Scenario 2: User has custom config**

```
Old config: Only Claude CLI settings
New config: Adds MCP section (optional)

Migration: MCP section added with defaults if missing
Existing settings: Preserved unchanged
User Impact: None (config extends, not replaces)
```

#### Auto-Migration Steps

**On First Run with MCP:**

1. **Detect old config format**
   ```go
   if !config.HasMCPSection() {
       config.MCP = mcp.DefaultConfig()
       config.Save()
   }
   ```

2. **Generate MCP config file**
   ```go
   if !fileExists("~/.config/gofortress/mcp-server.json") {
       generateMCPConfig()
   }
   ```

3. **Start MCP server**
   ```go
   mcpServer, err := startMCPServer(config.MCP)
   if err != nil {
       log.Warn("MCP unavailable, using AllowedTools mode")
       // Continue without MCP
   }
   ```

4. **Display migration notice** (once)
   ```
   ┌────────────────────────────────────────────────┐
   │  🎉 gofortress now supports interactive       │
   │     prompts!                                   │
   │                                                 │
   │  Claude can now ask you questions and wait     │
   │  for your input before proceeding.             │
   │                                                 │
   │  Press any key to continue...                  │
   └────────────────────────────────────────────────┘
   ```

#### Testing Migration

**Test Cases:**

1. **Clean install (no existing config)**
   - Expected: MCP enabled by default
   - Verify: Interactive prompts work

2. **Upgrade from v0.1 (AllowedTools only)**
   - Expected: Config auto-migrated, MCP enabled
   - Verify: Old config preserved, MCP section added

3. **Upgrade with custom AllowedTools**
   - Expected: Custom tools preserved, MCP tools added
   - Verify: Both AllowedTools and MCP tools work

4. **MCP server fails to start**
   - Expected: Falls back to AllowedTools mode
   - Verify: Warning displayed, gofortress still works

5. **Multiple concurrent instances**
   - Expected: Each gets unique socket path
   - Verify: No socket conflicts

---

## Extension Points

### Adding New Custom Tools

**Process (for contributors):**

1. **Define tool schema**
   ```go
   type MyToolInput struct {
       Param1 string `json:"param1"`
       Param2 int    `json:"param2"`
   }
   ```

2. **Implement handler**
   ```go
   func handleMyTool(ctx context.Context, input json.RawMessage) (*tools.ToolResult, error) {
       var params MyToolInput
       json.Unmarshal(input, &params)

       // Tool logic here

       return &tools.ToolResult{
           Content: []tools.ContentBlock{{
               Type: "text",
               Text: "result",
           }},
       }, nil
   }
   ```

3. **Register in init**
   ```go
   func init() {
       tools.DefaultRegistry.Register(&tools.ToolDefinition{
           Name: "my_tool",
           Description: "Does something useful",
           InputSchema: tools.GenerateSchema(MyToolInput{}),
           Handler: handleMyTool,
       })
   }
   ```

4. **Add to config**
   ```yaml
   mcp:
     tools:
       - my_tool
   ```

**Estimated effort:** <50 lines of code, 1-2 hours

---

### Plugin System (Future)

**Design:**

```go
// pkg/mcptools/plugin.go
type Plugin interface {
    Name() string
    Version() string
    Tools() []*ToolDefinition
    Initialize(ctx context.Context) error
    Shutdown(ctx context.Context) error
}

// Load plugins from directory
func LoadPlugins(dir string) ([]Plugin, error)

// Register all tools from a plugin
func RegisterPlugin(p Plugin) error
```

**Usage:**

```go
// Load plugins from ~/.config/gofortress/plugins/
plugins, _ := mcptools.LoadPlugins("~/.config/gofortress/plugins/")

for _, plugin := range plugins {
    plugin.Initialize(ctx)
    mcptools.RegisterPlugin(plugin)
}
```

**Example Plugin:**

```go
// ~/.config/gofortress/plugins/jira/main.go
package main

type JiraPlugin struct {}

func (p *JiraPlugin) Name() string { return "jira" }
func (p *JiraPlugin) Version() string { return "1.0.0" }

func (p *JiraPlugin) Tools() []*mcptools.ToolDefinition {
    return []*mcptools.ToolDefinition{
        {
            Name: "create_jira_ticket",
            // ...
        },
        {
            Name: "search_jira",
            // ...
        },
    }
}

// Plugin entry point
var Plugin = &JiraPlugin{}
```

---

### Third-Party Tool Integration

**Scenario:** User wants to add tool from external package

**Option 1: Import as Go package**

```go
// cmd/gofortress/main.go
import _ "github.com/user/gofortress-slack-tools"

// Tools auto-register in init()
```

**Option 2: HTTP MCP server**

```yaml
# config.yaml
mcp:
  external_servers:
    - name: slack
      transport: http
      url: http://localhost:9000
      tools:
        - send_slack_message
        - create_slack_channel
```

**Option 3: Stdio MCP server**

```bash
# Register external MCP server
claude mcp add slack-tools -- node /path/to/slack-mcp-server.js

# gofortress detects and uses automatically
```

---

## Operational Considerations

### Logging and Debugging

**Log Levels:**

| Level | Usage | Examples |
|-------|-------|----------|
| DEBUG | Development, troubleshooting | MCP protocol messages, channel operations |
| INFO | Normal operations | Server started, tool called, user responded |
| WARN | Degraded mode, recoverable errors | MCP server restart, timeout warnings |
| ERROR | Failures requiring attention | Server startup failed, invalid tool input |

**Log Locations:**

- **Console:** INFO and above (unless --debug)
- **File:** `~/.local/state/gofortress/gofortress.log` (all levels)
- **Structured:** JSON format for machine parsing (optional)

**Debug Mode:**

```bash
# Enable MCP debug logging
gofortress --mcp-debug

# Full debug logging
gofortress --debug

# Debug specific component
gofortress --debug=mcp,ipc
```

**Log Example:**

```json
{
  "timestamp": "2026-01-27T10:30:45Z",
  "level": "info",
  "component": "mcp-server",
  "message": "tool called",
  "tool": "ask_user",
  "request_id": "req-abc123",
  "duration_ms": 1234
}
```

---

### Performance Impact

**Baseline (without MCP):**
- Memory: ~20MB
- CPU: <5% idle, 15-30% during streaming
- Startup time: ~500ms

**With MCP (embedded):**
- Memory: +5MB (~25MB total) - MCP server goroutine
- CPU: +2% idle - channel monitoring
- Startup time: +100ms - MCP server initialization
- IPC latency: <1ms (Unix socket)

**Per-tool call overhead:**
- Channel send: <100μs
- Unix socket round-trip: <1ms
- JSON encode/decode: <200μs
- Total: <2ms (imperceptible to user)

**Optimization targets:**
- Prompt display: <100ms from tool call to visible
- User response: <50ms from keypress to sent
- No memory leaks over 24h session
- <1% CPU when idle with MCP active

---

### Resource Usage

**File Descriptors:**
- Unix socket: 1-2 FDs
- Claude CLI pipes: 3 FDs (stdin/stdout/stderr)
- Log file: 1 FD
- **Total:** ~7 FDs (well below limits)

**Goroutines:**
- Main TUI loop: 1
- MCP server listener: 1
- MCP connection handler: 1 per connection (usually 1)
- Channel monitors: 2
- **Total:** ~6 goroutines (minimal)

**Network:**
- Unix socket only (no external network)
- HTTP transport (optional): localhost only
- No remote connections

---

### Security Considerations

**Unix Socket Security:**
- Socket file permissions: 0600 (owner only)
- Location: /tmp (ephemeral, per-user)
- Validation: Check peer credentials (if supported)

**Tool Input Validation:**
- JSON schema validation on all inputs
- Sanitize user-provided strings
- Limit array sizes (max 4 options)
- Timeout all operations (prevent DoS)

**Error Messages:**
- Don't leak sensitive paths
- Don't include full stack traces in production
- Log detailed errors, show generic to user

**Privilege Separation:**
- MCP server runs as same user as gofortress
- No elevated privileges required
- No setuid binaries

---

### Multi-Instance Handling

**Scenario:** User runs multiple gofortress instances simultaneously

**Strategy:**

1. **Unique socket paths**
   ```go
   socketPath := fmt.Sprintf("/tmp/gofortress-mcp-%d.sock", os.Getpid())
   ```

2. **Separate MCP configs**
   ```go
   mcpConfigPath := fmt.Sprintf("/tmp/gofortress-mcp-%d.json", os.Getpid())
   ```

3. **No shared state**
   - Each instance has own MCP server
   - Each Claude CLI connects to its own socket
   - No conflicts

4. **Cleanup on exit**
   ```go
   defer os.Remove(socketPath)
   defer os.Remove(mcpConfigPath)
   ```

**Testing:**
```bash
# Terminal 1
gofortress

# Terminal 2
gofortress

# Both work independently
```

---

### Regular Claude CLI Isolation

**CRITICAL:** gofortress MCP implementation does NOT interfere with regular Claude Code usage.

**User has `goclaude` alias that spawns Claude with hooks:**

```bash
# Regular workflow (UNCHANGED)
goclaude
# → Claude CLI with hooks (gogent-load-context, gogent-validate, etc.)
# → NO MCP config
# → Routing works normally
# → ZERO interference from gofortress
```

**How isolation is achieved:**

1. **No global MCP registration**
   - gofortress does NOT use `claude mcp add`
   - MCP config is passed via `--mcp-config` flag only to gofortress subprocess
   - Global Claude CLI config unaffected

2. **Temporary MCP configs**
   - Config files in `/tmp` with unique PIDs
   - Removed on gofortress exit
   - Never written to `~/.config/claude/`

3. **Process-scoped sockets**
   - Unix sockets in `/tmp` with PIDs
   - Only accessible to specific gofortress instance
   - Cleaned up automatically

**Verification:**

```bash
# Before gofortress exists
claude mcp list
# → No MCP servers configured

# Run gofortress
gofortress
# (in another terminal)
claude mcp list
# → Still no MCP servers configured (gofortress uses --mcp-config, not global)

# After gofortress exits
claude mcp list
# → Still no MCP servers configured

# Your goclaude alias continues to work identically
goclaude
# → Works exactly as before
```

**Design Principle:** gofortress is a self-contained TUI. It brings its own MCP server when running, takes it away when exiting. Zero global state pollution.

---

## Appendix

### MCP Protocol Reference

**Specification:** https://spec.modelcontextprotocol.io/ (if available)

**Key Concepts:**
- JSON-RPC 2.0 over stdio or HTTP
- Stateless request/response
- Capabilities negotiation
- Tool discovery and execution

**Message Types:**
1. `initialize` - Handshake, capabilities
2. `tools/list` - Discover available tools
3. `tools/call` - Execute a tool

---

### JSON Schema Generation

```go
// internal/mcp/tools/schema.go
package tools

import (
    "encoding/json"
    "reflect"
)

// GenerateSchema creates JSON Schema from Go struct
func GenerateSchema(v interface{}) interface{} {
    // Use reflection to generate schema
    // Or use library like github.com/invopop/jsonschema

    schema := /* generate schema */
    return schema
}

// Example
type Example struct {
    Name string `json:"name" jsonschema:"required,description=User name"`
    Age  int    `json:"age" jsonschema:"minimum=0,maximum=150"`
}

schema := GenerateSchema(Example{})
// Returns JSON Schema object
```

---

### Testing Checklist

Before release, verify:

- [ ] All unit tests pass (>80% coverage)
- [ ] Integration tests pass
- [ ] E2E tests with mock Claude CLI pass
- [ ] Manual testing completed
- [ ] Performance benchmarks meet targets
- [ ] No goroutine leaks
- [ ] No memory leaks over 24h
- [ ] MCP server restart works
- [ ] Graceful degradation works
- [ ] Multiple instances don't conflict
- [ ] Unix socket permissions correct
- [ ] Error messages user-friendly
- [ ] Logging doesn't leak sensitive data
- [ ] Documentation complete
- [ ] Migration tested on real configs
- [ ] Backward compatibility verified

---

### Troubleshooting Guide

**Issue:** Interactive prompts don't appear

**Diagnosis:**
1. Check MCP server running: `ps aux | grep gofortress`
2. Check socket exists: `ls -la /tmp/gofortress-mcp*.sock`
3. Check logs: `tail ~/.local/state/gofortress/gofortress.log`
4. Enable debug: `gofortress --mcp-debug`

**Solution:**
- If socket missing: Check permissions on /tmp
- If server not running: Check startup errors in log
- If tool not registered: Verify config includes tool name

---

**Issue:** "MCP server unavailable" warning on startup

**Diagnosis:**
1. Check previous instance: `ps aux | grep gofortress`
2. Check socket in use: `lsof /tmp/gofortress-mcp.sock`
3. Check logs for startup error

**Solution:**
- Kill previous instance: `pkill gofortress`
- Remove stale socket: `rm /tmp/gofortress-mcp*.sock`
- Restart gofortress

---

**Issue:** Tool call times out after 60s

**Diagnosis:**
1. User didn't respond in time
2. TUI event loop blocked
3. Channel deadlock

**Solution:**
- Check TUI responsiveness
- Enable debug logging to see channel operations
- Check for goroutine blocking

---

### Glossary

| Term | Definition |
|------|------------|
| **MCP** | Model Context Protocol - protocol for custom tools |
| **JSON-RPC** | Remote procedure call protocol using JSON |
| **stdio** | Standard input/output transport |
| **Unix socket** | Local inter-process communication mechanism |
| **IPC** | Inter-Process Communication |
| **Tool** | Custom function callable by Claude |
| **Registry** | Central store of tool definitions |
| **Handler** | Function that implements tool logic |
| **Schema** | JSON Schema defining tool input structure |
| **Transport** | Communication mechanism (stdio, HTTP) |

---

### References

- Claude Agent SDK Documentation (user-provided)
- Claude Code CLI Help: `claude --help`, `claude mcp --help`
- MCP Specification: https://spec.modelcontextprotocol.io/
- Go Conventions: `.claude/conventions/go.md`
- GOgent-Fortress Architecture: `ARCHITECTURE.md`

---

## Conclusion

This guide provides a comprehensive roadmap for implementing MCP-based interactive prompts in gofortress. The architecture is:

- **Elegant:** Clean separation of concerns, idiomatic Go
- **Extensible:** Easy to add new tools (<50 LOC each)
- **Reliable:** >80% test coverage, graceful degradation
- **Simple:** Single binary, auto-configuration
- **Production-ready:** Error handling, logging, performance

**Estimated Timeline:** 6-8 weeks for complete implementation

**Next Steps:**
1. Review this guide with team
2. Begin Phase 1 (Foundation) tasks
3. Iterate based on testing feedback
4. Ship Phase 2 (Interactive Prompts) as MVP
5. Enhance with Phase 3-4 based on user needs

---

**End of MCP Implementation Guide**
