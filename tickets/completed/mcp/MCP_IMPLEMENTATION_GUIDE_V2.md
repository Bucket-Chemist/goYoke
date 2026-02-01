# MCP Implementation Guide v2.0 for GOgent-Fortress

**Version:** 2.0
**Status:** Architecture Design (Revised)
**Author:** Einstein Analysis System
**Date:** 2026-01-30
**Supersedes:** MCP_IMPLEMENTATION_GUIDE.md (v1.0)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Technical Specifications](#3-technical-specifications)
4. [Implementation Phases](#4-implementation-phases)
5. [Ticket Specifications](#5-ticket-specifications)
6. [Code Organization](#6-code-organization)
7. [Tool Catalog](#7-tool-catalog)
8. [Testing Strategy](#8-testing-strategy)
9. [Operational Considerations](#9-operational-considerations)
10. [Migration Path](#10-migration-path)

---

## 1. Executive Summary

### 1.1 Problem Statement

gofortress currently uses `AllowedTools` for pre-approval, which removes user control over Claude's actions. Users want:
- Interactive prompts for decisions during execution
- Ability to approve/reject actions with context
- Guidance over tool execution without terminal-blocking dialogs
- Extensible foundation for future interactive features

### 1.2 Previous Approach (v1.0) - DEPRECATED

The original MCP Implementation Guide proposed an embedded MCP server communicating via Unix sockets. **This approach was fatally flawed:**

| Assumption | Reality | Impact |
|------------|---------|--------|
| Unix socket transport | MCP only supports **stdio** and **HTTP** | Entire transport layer invalid |
| Embedded goroutine server | Claude **spawns** MCP servers as subprocesses | Architecture incompatible |
| Channel-based IPC | Go channels work **within** a process | Cross-process IPC required |
| `--mcp-config` with socket path | Config specifies **command**, not socket | Config format incorrect |

### 1.3 Corrected Architecture (v2.0)

The revised architecture uses a **three-process hierarchy** with **Unix socket HTTP callback**:

```
gofortress (grandparent)
├── Unix Socket HTTP Server (callback)
├── Claude CLI (child, spawned by gofortress)
│   └── gofortress-mcp-server (grandchild, spawned by Claude via MCP config)
│       └── HTTP client connecting back to grandparent's socket
```

**Key Innovation:** The MCP server is a separate binary that Claude spawns, but it calls back to the TUI via a Unix socket server that gofortress hosts.

### 1.4 Success Metrics

| Metric | Target |
|--------|--------|
| Prompt display latency | <100ms from tool call to visible |
| Round-trip latency | <10ms p95 for socket IPC (revised from <3ms per staff-architect review) |
| Test coverage | >80% across all new packages |
| Graceful degradation | Falls back to AllowedTools on failure |
| Session isolation | Zero interference with goclaude/claude CLI |

### 1.5 Timeline (Revised per Staff Architect Review)

| Phase | Duration | Focus |
|-------|----------|-------|
| **Pre-Phase: Hardening** | 1 week | Signal handling, crash recovery (staff-architect critical fixes) |
| Phase 1: Foundation | 1.5 weeks | Unix socket callback server, MCP server binary |
| Phase 2: TUI Integration | 1.5 weeks | Modal prompts, event handling (with channel blocking fix) |
| Phase 3: Claude Integration | 1 week | MCP config generation, E2E flow |
| Phase 4: Hardening | 1.5 weeks | Error handling, testing, documentation |

**Total:** 6-7 weeks (revised from 4 weeks per staff-architect review - production-ready timeline)

---

## 2. Architecture Overview

### 2.1 Process Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          gofortress TUI (PID: 1000)                         │
│  ┌────────────────────┐  ┌───────────────────────────────────────────────┐  │
│  │  Bubbletea Model   │  │  Unix Socket HTTP Server                      │  │
│  │  ├─ MainView       │  │  /run/user/1000/gofortress-1000.sock          │  │
│  │  ├─ PromptModal    │  │  POST /prompt → displays modal               │  │
│  │  └─ ResponseChan   │◄─┤  POST /confirm → yes/no dialog              │  │
│  └────────────────────┘  │  GET  /health → server status                │  │
│           │              └───────────────────────────────────────────────┘  │
│           │ spawns                              ▲                            │
└───────────┼─────────────────────────────────────┼────────────────────────────┘
            ▼                                     │
┌─────────────────────────────────────────────────┼────────────────────────────┐
│                    Claude Code CLI (PID: 1001)  │                            │
│  ┌────────────────────────────────────────────┐ │ (Unix socket callback)     │
│  │ --mcp-config /tmp/gofortress-mcp-1000.json │ │                            │
│  │ NDJSON event streaming to gofortress       │ │                            │
│  └────────────────────────────────────────────┘ │                            │
│           │ spawns via mcp-config               │                            │
└───────────┼─────────────────────────────────────┼────────────────────────────┘
            ▼                                     │
┌─────────────────────────────────────────────────┼────────────────────────────┐
│               gofortress-mcp-server (PID: 1002) │                            │
│  ┌────────────────────────────────────────────┐ │                            │
│  │ MCP Server (stdio transport to Claude)     │ │                            │
│  │ Tools: ask_user, confirm_action,           │─┘                            │
│  │        request_input, select_option        │                              │
│  │ Calls back to TUI via Unix socket          │                              │
│  └────────────────────────────────────────────┘                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility | Communication |
|-----------|----------------|---------------|
| **gofortress TUI** | Renders UI, spawns Claude, hosts callback server, displays modals | Reads Claude stdout (NDJSON), receives HTTP from MCP server |
| **Unix Socket Server** | Receives prompt requests from MCP server, returns responses | HTTP over Unix socket |
| **Claude Code CLI** | AI agent processing, tool execution, event streaming | stdin/stdout (NDJSON), spawns MCP server |
| **gofortress-mcp-server** | MCP protocol over stdio, calls TUI for user input | stdin/stdout (JSON-RPC), HTTP client to socket |

### 2.3 Data Flow: Interactive Prompt

```
┌────────────────────────────────────────────────────────────────────────────┐
│ 1. Claude decides to ask user                                               │
│    → calls mcp__gofortress__ask_user({message: "Which DB?", options: [...]}) │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ JSON-RPC via stdio
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 2. MCP Server receives tool call                                            │
│    → parses CallToolParams, extracts message + options                      │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ HTTP POST over Unix socket
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 3. TUI callback server receives request                                     │
│    → sends PromptRequest to TUI model via tea.Cmd                          │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ Bubbletea message
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 4. TUI displays modal overlay                                               │
│    → user sees options, presses 1/2/3/4 or types response                  │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ User input captured
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 5. TUI sends response to pending channel                                    │
│    → HTTP handler returns JSON response                                     │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ HTTP response
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 6. MCP Server receives response                                             │
│    → formats CallToolResult with user's answer                             │
└─────────────────────────────────────────┬──────────────────────────────────┘
                                          │ JSON-RPC response via stdio
                                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ 7. Claude continues with user's choice                                      │
│    → "I'll use SQLite as you requested..."                                 │
└────────────────────────────────────────────────────────────────────────────┘
```

### 2.4 Integration with Existing Hooks

The MCP system operates **alongside** existing GOgent hooks without interference:

| Component | Purpose | Interaction |
|-----------|---------|-------------|
| `gogent-validate` | Validates Task() calls | MCP tools appear as `mcp__gofortress__*`, can be validated |
| `gogent-sharp-edge` | Tracks tool usage | MCP tool calls logged to telemetry |
| `gogent-load-context` | Session initialization | Runs before MCP server spawned |
| `gogent-agent-endstate` | Subagent completion | Tracks MCP tool outcomes |
| `gogent-archive` | Session handoff | Includes MCP interaction history |

**Hook Interception of MCP Tools:**

```go
// In gogent-validate, MCP tools can be validated:
if strings.HasPrefix(event.Tool, "mcp__gofortress__") {
    toolName := strings.TrimPrefix(event.Tool, "mcp__gofortress__")
    // Apply policies to MCP tools
}
```

---

## 3. Technical Specifications

### 3.1 Go MCP SDK Reference

The official Go SDK at `github.com/modelcontextprotocol/go-sdk` (v1.2.0) provides the MCP implementation.

**Server Creation Pattern:**

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Logger MUST write to stderr (stdout reserved for MCP protocol)
    logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

    server := mcp.NewServer(
        &mcp.Implementation{
            Name:    "gofortress-mcp-server",
            Version: "1.0.0",
        },
        &mcp.ServerOptions{Logger: logger},
    )

    // Register tools (see Section 7)
    registerTools(server)

    // Run over stdio transport
    if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
        logger.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

**Tool Registration Pattern:**

```go
// Input type with JSON schema inference
type AskUserInput struct {
    Message string   `json:"message" jsonschema:"question to ask,required"`
    Options []string `json:"options,omitempty" jsonschema:"optional choices"`
    Default string   `json:"default,omitempty" jsonschema:"default value"`
}

// Output type
type AskUserOutput struct {
    Response  string `json:"response"`
    Cancelled bool   `json:"cancelled"`
}

// Handler signature
func askUserHandler(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (
    *mcp.CallToolResult, AskUserOutput, error,
) {
    // Implementation calls back to TUI
    response, err := callbackToTUI(ctx, input)
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
        }, AskUserOutput{}, nil
    }

    return nil, AskUserOutput{
        Response:  response.Value,
        Cancelled: response.Cancelled,
    }, nil
}

// Registration
func registerTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "ask_user",
        Description: "Ask user a question and wait for response",
    }, askUserHandler)
}
```

### 3.2 MCP Configuration Schema

Claude Code discovers MCP servers through JSON configuration passed via `--mcp-config`:

```json
{
  "mcpServers": {
    "gofortress": {
      "type": "stdio",
      "command": "/usr/local/bin/gofortress-mcp-server",
      "args": [],
      "env": {
        "GOFORTRESS_SOCKET": "/run/user/1000/gofortress-1000.sock",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

**Critical Points:**
- `type: "stdio"` - MCP server spawned as subprocess, communicates via stdin/stdout
- `command` - Path to the MCP server binary (not a socket path)
- `env.GOFORTRESS_SOCKET` - How the MCP server knows where to call back

### 3.3 Unix Socket HTTP Server Specification

**Socket Path Resolution:**

```go
func getSocketPath(pid int) string {
    // Prefer XDG_RUNTIME_DIR (per-user, in-memory, auto-cleaned)
    if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
        return filepath.Join(runtimeDir, fmt.Sprintf("gofortress-%d.sock", pid))
    }
    // Fallback to /tmp
    return filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-%d.sock", pid))
}
```

**Server Endpoints:**

| Endpoint | Method | Purpose | Timeout |
|----------|--------|---------|---------|
| `/prompt` | POST | Receive prompt request, block until user responds | 5 minutes |
| `/confirm` | POST | Yes/no confirmation dialog | 5 minutes |
| `/health` | GET | Verify server is running | 5 seconds |

**Request/Response Schemas:**

```go
// Request from MCP server to TUI
type PromptRequest struct {
    ID      string   `json:"id"`      // UUID for correlation
    Type    string   `json:"type"`    // "ask", "confirm", "input", "select"
    Message string   `json:"message"` // Question text
    Options []string `json:"options,omitempty"` // For selection
    Default string   `json:"default,omitempty"` // Pre-fill value
}

// Response from TUI to MCP server
type PromptResponse struct {
    ID        string `json:"id"`        // Correlation ID
    Value     string `json:"value"`     // User's response
    Cancelled bool   `json:"cancelled"` // True if user pressed ESC
    Error     string `json:"error,omitempty"`
}
```

### 3.4 Claude CLI Invocation

The `internal/cli/subprocess.go` Config needs extension:

```go
type Config struct {
    // Existing fields...
    ClaudePath      string
    SessionID       string
    AllowedTools    []string
    // ...

    // New MCP fields
    MCPConfigPath   string   // Path to generated MCP config JSON
    MCPTools        []string // MCP tools to add to AllowedTools
}

func (c *Config) buildArgs() []string {
    args := []string{
        "--print",
        "--output-format", "stream-json",
        "--input-format", "stream-json",
    }

    // Existing argument building...

    // Add MCP config if provided
    if c.MCPConfigPath != "" {
        args = append(args, "--mcp-config", c.MCPConfigPath)
    }

    return args
}
```

---

## 4. Implementation Phases

### 4.1 Phase 1: Foundation (Week 1)

**Goal:** Build the callback infrastructure and MCP server binary

**Tickets:**
- GOgent-MCP-001: Unix Socket HTTP Server
- GOgent-MCP-002: Callback Client Library
- GOgent-MCP-003: MCP Server Binary
- GOgent-MCP-004: MCP Config Generator

**Deliverables:**
- `internal/callback/server.go` - HTTP over Unix socket
- `internal/callback/client.go` - Client for MCP server
- `cmd/gofortress-mcp-server/main.go` - MCP server binary
- `internal/mcp/config.go` - Config generation

### 4.2 Phase 2: TUI Integration (Week 2)

**Goal:** Display modal prompts and capture user responses

**Tickets:**
- GOgent-MCP-005: Modal State Management
- GOgent-MCP-006: Prompt Rendering
- GOgent-MCP-007: Input Handling
- GOgent-MCP-008: External Event Integration

**Deliverables:**
- `internal/tui/claude/modal.go` - Modal component
- `internal/tui/claude/prompt.go` - Prompt rendering
- Updated `panel.go` with modal integration

### 4.3 Phase 3: Claude Integration (Week 3)

**Goal:** Wire everything together for end-to-end flow

**Tickets:**
- GOgent-MCP-009: Main Orchestration
- GOgent-MCP-010: Tool Registration
- GOgent-MCP-011: AllowedTools Integration
- GOgent-MCP-012: Session Isolation

**Deliverables:**
- Updated `cmd/gofortress/main.go`
- Integration tests with mock Claude

### 4.4 Phase 4: Hardening (Week 4)

**Goal:** Production-ready reliability

**Tickets:**
- GOgent-MCP-013: Error Handling
- GOgent-MCP-014: Graceful Degradation
- GOgent-MCP-015: Comprehensive Testing
- GOgent-MCP-016: Documentation

**Deliverables:**
- >80% test coverage
- Error recovery flows
- User documentation

---

## 5. Ticket Specifications

### 5.0 Pre-Implementation Hardening (From Staff Architect Review)

These tickets address critical issues identified in Appendix C that MUST be completed before or alongside Phase 1.

---

#### GOgent-MCP-000: Process Lifecycle and Crash Recovery

**Time:** 4 hours
**Dependencies:** None
**Priority:** CRITICAL (blocks Phase 3)

**Task:**
Implement signal handling for child process cleanup and stale socket recovery. These are critical operational requirements identified in staff-architect review.

**File:** `internal/lifecycle/process.go`

**Imports:**
```go
package lifecycle

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
)
```

**Implementation:**
```go
// ProcessManager handles process lifecycle and cleanup
type ProcessManager struct {
    childProcess *os.Process
    socketPath   string
    sigChan      chan os.Signal
    done         chan struct{}
}

// NewProcessManager creates a new process lifecycle manager
func NewProcessManager(socketPath string) *ProcessManager {
    return &ProcessManager{
        socketPath: socketPath,
        sigChan:    make(chan os.Signal, 1),
        done:       make(chan struct{}),
    }
}

// SetChildProcess registers the Claude process for cleanup
func (pm *ProcessManager) SetChildProcess(p *os.Process) {
    pm.childProcess = p
}

// StartSignalHandler begins listening for termination signals
// CRITICAL: Must be called early in main() before spawning Claude
func (pm *ProcessManager) StartSignalHandler(ctx context.Context, onShutdown func()) {
    signal.Notify(pm.sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

    go func() {
        select {
        case sig := <-pm.sigChan:
            // Propagate signal to child process
            if pm.childProcess != nil {
                pm.childProcess.Signal(sig)
            }

            // Run shutdown callback (e.g., callbackServer.Shutdown)
            if onShutdown != nil {
                onShutdown()
            }

            // Clean up socket
            os.Remove(pm.socketPath)

            close(pm.done)

        case <-ctx.Done():
            close(pm.done)
        }
    }()
}

// Wait blocks until shutdown complete
func (pm *ProcessManager) Wait() {
    <-pm.done
}

// CleanupStaleSockets removes orphaned socket files from crashed sessions
// CRITICAL: Must be called at startup before creating new socket
func CleanupStaleSockets() error {
    // Check XDG_RUNTIME_DIR first
    runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
    if runtimeDir == "" {
        runtimeDir = os.TempDir()
    }

    pattern := filepath.Join(runtimeDir, "gofortress-*.sock")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return fmt.Errorf("[lifecycle] Failed to glob socket pattern: %w", err)
    }

    cleaned := 0
    for _, path := range matches {
        pid := extractPIDFromPath(path)
        if pid > 0 && !processExists(pid) {
            if err := os.Remove(path); err != nil {
                // Log but don't fail - socket might be in use
                fmt.Fprintf(os.Stderr, "[lifecycle] Warning: could not remove stale socket %s: %v\n", path, err)
            } else {
                cleaned++
            }
        }
    }

    if cleaned > 0 {
        fmt.Fprintf(os.Stderr, "[lifecycle] Cleaned %d stale socket(s)\n", cleaned)
    }

    return nil
}

// extractPIDFromPath extracts PID from socket filename like "gofortress-12345.sock"
func extractPIDFromPath(path string) int {
    base := filepath.Base(path)
    base = strings.TrimPrefix(base, "gofortress-")
    base = strings.TrimSuffix(base, ".sock")

    pid, err := strconv.Atoi(base)
    if err != nil {
        return 0
    }
    return pid
}

// processExists checks if a process with the given PID is running
func processExists(pid int) bool {
    process, err := os.FindProcess(pid)
    if err != nil {
        return false
    }

    // On Unix, FindProcess always succeeds, so we need to send signal 0
    err = process.Signal(syscall.Signal(0))
    return err == nil
}
```

**Tests:**
```go
package lifecycle

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestCleanupStaleSockets(t *testing.T) {
    // Create temp dir
    tmpDir := t.TempDir()
    t.Setenv("XDG_RUNTIME_DIR", tmpDir)

    // Create a stale socket (non-existent PID)
    stalePath := filepath.Join(tmpDir, "gofortress-99999999.sock")
    if err := os.WriteFile(stalePath, []byte("test"), 0600); err != nil {
        t.Fatalf("Failed to create stale socket: %v", err)
    }

    // Create a valid socket (current process)
    validPath := filepath.Join(tmpDir, fmt.Sprintf("gofortress-%d.sock", os.Getpid()))
    if err := os.WriteFile(validPath, []byte("test"), 0600); err != nil {
        t.Fatalf("Failed to create valid socket: %v", err)
    }

    // Run cleanup
    if err := CleanupStaleSockets(); err != nil {
        t.Fatalf("CleanupStaleSockets failed: %v", err)
    }

    // Stale should be removed
    if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
        t.Error("Stale socket was not removed")
    }

    // Valid should remain
    if _, err := os.Stat(validPath); err != nil {
        t.Error("Valid socket was incorrectly removed")
    }
}

func TestProcessManager_SignalPropagation(t *testing.T) {
    pm := NewProcessManager("/tmp/test.sock")
    ctx, cancel := context.WithCancel(context.Background())

    shutdownCalled := false
    pm.StartSignalHandler(ctx, func() {
        shutdownCalled = true
    })

    // Cancel context to trigger shutdown path
    cancel()

    select {
    case <-pm.done:
        // Good - shutdown completed
    case <-time.After(time.Second):
        t.Error("Signal handler did not complete")
    }
}

func TestExtractPIDFromPath(t *testing.T) {
    tests := []struct {
        path     string
        expected int
    }{
        {"/run/user/1000/gofortress-12345.sock", 12345},
        {"/tmp/gofortress-1.sock", 1},
        {"/tmp/gofortress-notapid.sock", 0},
        {"/tmp/other-file.sock", 0},
    }

    for _, tc := range tests {
        got := extractPIDFromPath(tc.path)
        if got != tc.expected {
            t.Errorf("extractPIDFromPath(%q) = %d, want %d", tc.path, got, tc.expected)
        }
    }
}
```

**Acceptance Criteria:**
- [ ] Signal handler propagates SIGTERM to child Claude process
- [ ] Stale sockets from crashed sessions are cleaned at startup
- [ ] Only sockets for non-existent PIDs are removed
- [ ] Current process socket is preserved
- [ ] Cleanup runs before socket creation

**Test Deliverables:**
- [ ] Test file created: `internal/lifecycle/process_test.go`
- [ ] Coverage achieved: >90%
- [ ] Tests passing: `go test ./internal/lifecycle/...`

**Why This Matters:**
Without signal propagation, crashing gofortress leaves orphaned Claude processes consuming resources. Without stale socket cleanup, restarting after a crash fails with "address already in use". These are operational necessities for production use.

---

### 5.1 Phase 1 Tickets

---

#### GOgent-MCP-001: Unix Socket HTTP Server

**Time:** 4 hours
**Dependencies:** None
**Priority:** HIGH (critical path)

**Task:**
Implement an HTTP server that listens on a Unix socket for receiving prompt requests from the MCP server.

**File:** `internal/callback/server.go`

**Imports:**
```go
package callback

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)
```

**Implementation:**
```go
// PromptRequest represents a request from the MCP server
type PromptRequest struct {
    ID      string   `json:"id"`
    Type    string   `json:"type"`    // "ask", "confirm", "input", "select"
    Message string   `json:"message"`
    Options []string `json:"options,omitempty"`
    Default string   `json:"default,omitempty"`
}

// PromptResponse represents the TUI's response
type PromptResponse struct {
    ID        string `json:"id"`
    Value     string `json:"value"`
    Cancelled bool   `json:"cancelled"`
    Error     string `json:"error,omitempty"`
}

// Server handles HTTP requests over Unix socket
type Server struct {
    socketPath string
    listener   net.Listener
    httpServer *http.Server

    // Channel for sending prompts to TUI
    PromptChan chan PromptRequest

    // Map of pending responses (keyed by prompt ID)
    pending   map[string]chan PromptResponse
    pendingMu sync.RWMutex

    // Lifecycle
    started bool
    mu      sync.Mutex
}

// NewServer creates a new callback server
func NewServer(pid int) *Server {
    socketPath := getSocketPath(pid)
    return &Server{
        socketPath: socketPath,
        PromptChan: make(chan PromptRequest, 10),
        pending:    make(map[string]chan PromptResponse),
    }
}

// getSocketPath returns the socket path for the given PID
func getSocketPath(pid int) string {
    // Prefer XDG_RUNTIME_DIR for better security (per-user, in-memory)
    if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
        return filepath.Join(runtimeDir, fmt.Sprintf("gofortress-%d.sock", pid))
    }
    return filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-%d.sock", pid))
}

// Start begins listening for HTTP requests
func (s *Server) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.started {
        return fmt.Errorf("[callback] Server already started")
    }

    // Remove stale socket
    os.Remove(s.socketPath)

    var err error
    s.listener, err = net.Listen("unix", s.socketPath)
    if err != nil {
        return fmt.Errorf("[callback] Failed to listen on %s: %w. Check permissions and path length.", s.socketPath, err)
    }

    // Set restrictive permissions (owner only)
    if err := os.Chmod(s.socketPath, 0600); err != nil {
        s.listener.Close()
        return fmt.Errorf("[callback] Failed to set socket permissions: %w", err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/prompt", s.handlePrompt)
    mux.HandleFunc("/confirm", s.handleConfirm)
    mux.HandleFunc("/health", s.handleHealth)

    s.httpServer = &http.Server{
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 5 * time.Minute, // Long timeout for user interaction
        BaseContext:  func(_ net.Listener) context.Context { return ctx },
    }

    go func() {
        if err := s.httpServer.Serve(s.listener); err != http.ErrServerClosed {
            // Log error but don't crash - graceful degradation
            fmt.Fprintf(os.Stderr, "[callback] Server error: %v\n", err)
        }
    }()

    s.started = true
    return nil
}

// handlePrompt processes prompt requests from MCP server
func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req PromptRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
        return
    }

    // Generate ID if not provided
    if req.ID == "" {
        req.ID = fmt.Sprintf("prompt-%d", time.Now().UnixNano())
    }

    // Create response channel for this prompt
    respChan := make(chan PromptResponse, 1)
    s.pendingMu.Lock()
    s.pending[req.ID] = respChan
    s.pendingMu.Unlock()

    defer func() {
        s.pendingMu.Lock()
        delete(s.pending, req.ID)
        s.pendingMu.Unlock()
    }()

    // Send to TUI for display
    select {
    case s.PromptChan <- req:
        // Sent successfully
    case <-r.Context().Done():
        http.Error(w, "Request cancelled", http.StatusRequestTimeout)
        return
    case <-time.After(5 * time.Second):
        http.Error(w, "TUI not responding", http.StatusServiceUnavailable)
        return
    }

    // Wait for user response (long-poll)
    select {
    case resp := <-respChan:
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    case <-r.Context().Done():
        http.Error(w, "Timeout waiting for user response", http.StatusGatewayTimeout)
    }
}

// handleConfirm processes yes/no confirmation requests
func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
    // Reuse prompt handler with type="confirm"
    s.handlePrompt(w, r)
}

// handleHealth returns server status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SendResponse sends a response for a pending prompt
func (s *Server) SendResponse(resp PromptResponse) error {
    s.pendingMu.RLock()
    ch, ok := s.pending[resp.ID]
    s.pendingMu.RUnlock()

    if !ok {
        return fmt.Errorf("[callback] No pending prompt with ID: %s", resp.ID)
    }

    select {
    case ch <- resp:
        return nil
    default:
        return fmt.Errorf("[callback] Response channel full for prompt: %s", resp.ID)
    }
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !s.started {
        return nil
    }

    if s.httpServer != nil {
        if err := s.httpServer.Shutdown(ctx); err != nil {
            return fmt.Errorf("[callback] Shutdown error: %w", err)
        }
    }

    s.started = false
    return nil
}

// Cleanup removes the socket file
func (s *Server) Cleanup() {
    os.Remove(s.socketPath)
}

// SocketPath returns the socket path
func (s *Server) SocketPath() string {
    return s.socketPath
}
```

**Tests:**
```go
package callback

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "os"
    "testing"
    "time"
)

func TestServer_StartAndShutdown(t *testing.T) {
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()

    // Verify socket exists
    if _, err := os.Stat(s.SocketPath()); os.IsNotExist(err) {
        t.Error("Socket file not created")
    }

    // Shutdown
    shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := s.Shutdown(shutdownCtx); err != nil {
        t.Errorf("Shutdown error: %v", err)
    }
}

func TestServer_HealthCheck(t *testing.T) {
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

    // Create HTTP client using Unix socket
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                return net.Dial("unix", s.SocketPath())
            },
        },
    }

    resp, err := client.Get("http://unix/health")
    if err != nil {
        t.Fatalf("Health check failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}

func TestServer_PromptRoundTrip(t *testing.T) {
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

    // Start goroutine to handle prompt
    go func() {
        req := <-s.PromptChan
        s.SendResponse(PromptResponse{
            ID:    req.ID,
            Value: "test response",
        })
    }()

    // Send prompt request
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                return net.Dial("unix", s.SocketPath())
            },
        },
        Timeout: 5 * time.Second,
    }

    reqBody, _ := json.Marshal(PromptRequest{
        ID:      "test-1",
        Type:    "ask",
        Message: "Test question?",
    })

    resp, err := client.Post("http://unix/prompt", "application/json", bytes.NewReader(reqBody))
    if err != nil {
        t.Fatalf("Prompt request failed: %v", err)
    }
    defer resp.Body.Close()

    var response PromptResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        t.Fatalf("Failed to decode response: %v", err)
    }

    if response.Value != "test response" {
        t.Errorf("Expected 'test response', got %q", response.Value)
    }
}
```

**Acceptance Criteria:**
- [ ] Server starts and listens on Unix socket
- [ ] Socket has 0600 permissions (owner only)
- [ ] Health endpoint returns 200 OK
- [ ] Prompt endpoint blocks until response sent
- [ ] SendResponse delivers to correct pending channel
- [ ] Graceful shutdown closes connections
- [ ] Cleanup removes socket file

**Test Deliverables:**
- [ ] Test file created: `internal/callback/server_test.go`
- [ ] Number of test functions: 3+
- [ ] Coverage achieved: >85%
- [ ] Tests passing: `go test ./internal/callback/...`
- [ ] Race detector clean: `go test -race ./internal/callback/...`

**Why This Matters:**
This is the IPC backbone that enables the three-process architecture. Without reliable socket communication, the MCP server cannot call back to the TUI for user prompts.

---

#### GOgent-MCP-002: Callback Client Library

**Time:** 2 hours
**Dependencies:** GOgent-MCP-001
**Priority:** HIGH

**Task:**
Implement an HTTP client that the MCP server uses to communicate with the TUI's Unix socket server.

**File:** `internal/callback/client.go`

**Imports:**
```go
package callback

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "time"
)
```

**Implementation:**
```go
// Client communicates with the TUI callback server
type Client struct {
    httpClient *http.Client
    socketPath string
}

// NewClient creates a callback client from environment
func NewClient() (*Client, error) {
    socketPath := os.Getenv("GOFORTRESS_SOCKET")
    if socketPath == "" {
        return nil, fmt.Errorf("[callback-client] GOFORTRESS_SOCKET not set. MCP server must be spawned by gofortress.")
    }

    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                dialer := net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(ctx, "unix", socketPath)
            },
            MaxIdleConns:      5,
            IdleConnTimeout:   90 * time.Second,
            DisableKeepAlives: false,
        },
        Timeout: 5 * time.Minute, // Long timeout for user interaction
    }

    return &Client{
        httpClient: client,
        socketPath: socketPath,
    }, nil
}

// NewClientWithPath creates a client with explicit socket path
func NewClientWithPath(socketPath string) *Client {
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                dialer := net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(ctx, "unix", socketPath)
            },
        },
        Timeout: 5 * time.Minute,
    }

    return &Client{
        httpClient: client,
        socketPath: socketPath,
    }
}

// SendPrompt sends a prompt request and waits for response
func (c *Client) SendPrompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://unix/prompt", bytes.NewReader(body))
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Request failed: %w. Verify TUI is running.", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return PromptResponse{}, fmt.Errorf("[callback-client] Server returned %d", resp.StatusCode)
    }

    var response PromptResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return PromptResponse{}, fmt.Errorf("[callback-client] Failed to decode response: %w", err)
    }

    return response, nil
}

// HealthCheck verifies the TUI server is reachable
func (c *Client) HealthCheck(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "http://unix/health", nil)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("[callback-client] Health check failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("[callback-client] Health check returned %d", resp.StatusCode)
    }
    return nil
}
```

**Tests:**
```go
package callback

import (
    "context"
    "os"
    "testing"
    "time"
)

func TestClient_WithServer(t *testing.T) {
    // Start server
    s := NewServer(os.Getpid())
    ctx := context.Background()

    if err := s.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer s.Cleanup()
    defer s.Shutdown(ctx)

    // Create client
    c := NewClientWithPath(s.SocketPath())

    // Test health check
    if err := c.HealthCheck(ctx); err != nil {
        t.Errorf("Health check failed: %v", err)
    }

    // Test prompt round-trip
    go func() {
        req := <-s.PromptChan
        s.SendResponse(PromptResponse{
            ID:    req.ID,
            Value: "user input",
        })
    }()

    resp, err := c.SendPrompt(ctx, PromptRequest{
        ID:      "test-client-1",
        Type:    "ask",
        Message: "Test?",
    })
    if err != nil {
        t.Fatalf("SendPrompt failed: %v", err)
    }

    if resp.Value != "user input" {
        t.Errorf("Expected 'user input', got %q", resp.Value)
    }
}

func TestClient_MissingSocket(t *testing.T) {
    // Unset environment variable
    os.Unsetenv("GOFORTRESS_SOCKET")

    _, err := NewClient()
    if err == nil {
        t.Error("Expected error for missing socket path")
    }
}
```

**Acceptance Criteria:**
- [ ] Client created from GOFORTRESS_SOCKET env var
- [ ] Health check returns nil on healthy server
- [ ] SendPrompt returns user response
- [ ] Proper error messages with context
- [ ] Connection timeout handling

**Test Deliverables:**
- [ ] Test file created: `internal/callback/client_test.go`
- [ ] Coverage achieved: >85%
- [ ] Tests passing

**Why This Matters:**
The client library is used by the MCP server binary to call back to the TUI. Clean abstraction here makes the MCP server implementation straightforward.

---

#### GOgent-MCP-003: MCP Server Binary

**Time:** 6 hours
**Dependencies:** GOgent-MCP-002
**Priority:** HIGH

**Task:**
Create the MCP server binary that Claude spawns, implementing interactive tools that call back to the TUI.

**File:** `cmd/gofortress-mcp-server/main.go`

**Imports:**
```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)
```

**Implementation:**
```go
var (
    version = "1.0.0"
    client  *callback.Client
    logger  *slog.Logger
)

// Tool input types
type AskUserInput struct {
    Message string   `json:"message" jsonschema:"question to ask the user,required"`
    Options []string `json:"options,omitempty" jsonschema:"optional list of choices"`
    Default string   `json:"default,omitempty" jsonschema:"default value if user doesn't respond"`
}

type ConfirmActionInput struct {
    Action      string `json:"action" jsonschema:"description of action requiring confirmation,required"`
    Destructive bool   `json:"destructive,omitempty" jsonschema:"whether action is destructive"`
}

type RequestInputInput struct {
    Prompt      string `json:"prompt" jsonschema:"input prompt to display,required"`
    Placeholder string `json:"placeholder,omitempty" jsonschema:"placeholder text"`
}

type SelectOptionInput struct {
    Message string   `json:"message" jsonschema:"selection prompt,required"`
    Options []string `json:"options" jsonschema:"options to choose from,required"`
}

// Tool output types
type AskUserOutput struct {
    Response  string `json:"response"`
    Cancelled bool   `json:"cancelled"`
}

type ConfirmActionOutput struct {
    Confirmed bool `json:"confirmed"`
    Cancelled bool `json:"cancelled"`
}

type RequestInputOutput struct {
    Input     string `json:"input"`
    Cancelled bool   `json:"cancelled"`
}

type SelectOptionOutput struct {
    Selected  string `json:"selected"`
    Index     int    `json:"index"`
    Cancelled bool   `json:"cancelled"`
}

// Tool handlers
func askUserHandler(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (
    *mcp.CallToolResult, AskUserOutput, error,
) {
    promptType := "ask"
    if len(input.Options) > 0 {
        promptType = "select"
    }

    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("ask-%d", time.Now().UnixNano()),
        Type:    promptType,
        Message: input.Message,
        Options: input.Options,
        Default: input.Default,
    })

    if err != nil {
        logger.Error("failed to get user response", "error", err)
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to communicate with TUI: " + err.Error()}},
        }, AskUserOutput{}, nil
    }

    return nil, AskUserOutput{
        Response:  resp.Value,
        Cancelled: resp.Cancelled,
    }, nil
}

func confirmActionHandler(ctx context.Context, req *mcp.CallToolRequest, input ConfirmActionInput) (
    *mcp.CallToolResult, ConfirmActionOutput, error,
) {
    message := input.Action
    if input.Destructive {
        message = "⚠️ DESTRUCTIVE: " + message
    }

    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("confirm-%d", time.Now().UnixNano()),
        Type:    "confirm",
        Message: message,
    })

    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get confirmation: " + err.Error()}},
        }, ConfirmActionOutput{}, nil
    }

    return nil, ConfirmActionOutput{
        Confirmed: resp.Value == "yes",
        Cancelled: resp.Cancelled,
    }, nil
}

func requestInputHandler(ctx context.Context, req *mcp.CallToolRequest, input RequestInputInput) (
    *mcp.CallToolResult, RequestInputOutput, error,
) {
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("input-%d", time.Now().UnixNano()),
        Type:    "input",
        Message: input.Prompt,
        Default: input.Placeholder,
    })

    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get user input: " + err.Error()}},
        }, RequestInputOutput{}, nil
    }

    return nil, RequestInputOutput{
        Input:     resp.Value,
        Cancelled: resp.Cancelled,
    }, nil
}

func selectOptionHandler(ctx context.Context, req *mcp.CallToolRequest, input SelectOptionInput) (
    *mcp.CallToolResult, SelectOptionOutput, error,
) {
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("select-%d", time.Now().UnixNano()),
        Type:    "select",
        Message: input.Message,
        Options: input.Options,
    })

    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get selection: " + err.Error()}},
        }, SelectOptionOutput{}, nil
    }

    // Find index of selected option
    index := -1
    for i, opt := range input.Options {
        if opt == resp.Value {
            index = i
            break
        }
    }

    return nil, SelectOptionOutput{
        Selected:  resp.Value,
        Index:     index,
        Cancelled: resp.Cancelled,
    }, nil
}

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Configure logging to stderr (stdout reserved for MCP protocol)
    logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: getLogLevel(),
    }))

    // Initialize callback client
    var err error
    client, err = callback.NewClient()
    if err != nil {
        logger.Error("failed to initialize callback client", "error", err)
        os.Exit(1)
    }

    // Verify TUI is reachable
    healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
    defer healthCancel()
    if err := client.HealthCheck(healthCtx); err != nil {
        logger.Error("TUI health check failed", "error", err)
        os.Exit(1)
    }
    logger.Info("connected to TUI callback server")

    // Create MCP server
    server := mcp.NewServer(
        &mcp.Implementation{
            Name:    "gofortress-mcp-server",
            Version: version,
        },
        &mcp.ServerOptions{
            Logger: logger,
            InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
                logger.Info("MCP session initialized with Claude")
            },
        },
    )

    // Register tools
    mcp.AddTool(server, &mcp.Tool{
        Name:        "ask_user",
        Description: "Ask the user a question. Use when you need clarification, preferences, or any input. Can present multiple choice options or free-form questions.",
    }, askUserHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "confirm_action",
        Description: "Request user confirmation before proceeding. Use for destructive operations, irreversible changes, or when explicit approval is needed.",
    }, confirmActionHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "request_input",
        Description: "Request free-form text input from the user. Use when you need text content like code snippets, descriptions, or configuration values.",
    }, requestInputHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "select_option",
        Description: "Present a list of options and let user select one. Returns both the selected value and its index.",
    }, selectOptionHandler)

    // Run server
    logger.Info("starting MCP server on stdio")
    if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
        if !errors.Is(err, mcp.ErrConnectionClosed) && !errors.Is(err, context.Canceled) {
            logger.Error("server error", "error", err)
            os.Exit(1)
        }
    }
    logger.Info("MCP server shutdown complete")
}

func getLogLevel() slog.Level {
    switch os.Getenv("LOG_LEVEL") {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

**Acceptance Criteria:**
- [ ] Binary builds: `go build ./cmd/gofortress-mcp-server`
- [ ] Reads GOFORTRESS_SOCKET from environment
- [ ] Performs health check on startup
- [ ] Implements 4 tools: ask_user, confirm_action, request_input, select_option
- [ ] Logs to stderr (never stdout)
- [ ] Graceful shutdown on SIGTERM/SIGINT
- [ ] Tool errors returned as IsError=true, not panics

**Test Deliverables:**
- [ ] Integration tests with mock callback server
- [ ] Coverage: >80%
- [ ] MCP protocol conformance verified

**Why This Matters:**
This binary is what Claude spawns via MCP config. It's the bridge between Claude's tool calls and the TUI's user interaction.

---

#### GOgent-MCP-004: MCP Config Generator

**Time:** 2 hours
**Dependencies:** GOgent-MCP-003
**Priority:** HIGH

**Task:**
Generate ephemeral MCP configuration JSON that points Claude to the gofortress-mcp-server binary with the correct socket path.

**File:** `internal/mcp/config.go`

**Imports:**
```go
package mcp

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)
```

**Implementation:**
```go
// MCPConfig represents the MCP configuration for Claude CLI
type MCPConfig struct {
    MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
    Type    string            `json:"type"`
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env"`
}

// GenerateConfig creates an MCP configuration file
func GenerateConfig(pid int, socketPath, serverBinaryPath string) (configPath string, err error) {
    config := MCPConfig{
        MCPServers: map[string]MCPServerConfig{
            "gofortress": {
                Type:    "stdio",
                Command: serverBinaryPath,
                Args:    []string{},
                Env: map[string]string{
                    "GOFORTRESS_SOCKET": socketPath,
                    "LOG_LEVEL":         "info",
                },
            },
        },
    }

    // Write to temp file
    configPath = filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-mcp-%d.json", pid))
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return "", fmt.Errorf("[mcp-config] Failed to marshal config: %w", err)
    }

    if err := os.WriteFile(configPath, data, 0600); err != nil {
        return "", fmt.Errorf("[mcp-config] Failed to write config to %s: %w", configPath, err)
    }

    return configPath, nil
}

// FindServerBinary locates the gofortress-mcp-server binary
func FindServerBinary() (string, error) {
    // Check alongside main binary first
    exe, err := os.Executable()
    if err == nil {
        dir := filepath.Dir(exe)
        candidate := filepath.Join(dir, "gofortress-mcp-server")
        if _, err := os.Stat(candidate); err == nil {
            return candidate, nil
        }
    }

    // Check common installation paths
    paths := []string{
        "/usr/local/bin/gofortress-mcp-server",
        "/usr/bin/gofortress-mcp-server",
        filepath.Join(os.Getenv("HOME"), ".local/bin/gofortress-mcp-server"),
        filepath.Join(os.Getenv("HOME"), "go/bin/gofortress-mcp-server"),
    }

    for _, p := range paths {
        if _, err := os.Stat(p); err == nil {
            return p, nil
        }
    }

    return "", fmt.Errorf("[mcp-config] gofortress-mcp-server not found. Install with: go install ./cmd/gofortress-mcp-server")
}

// Cleanup removes the config file
func Cleanup(configPath string) {
    if configPath != "" {
        os.Remove(configPath)
    }
}
```

**Tests:**
```go
package mcp

import (
    "encoding/json"
    "os"
    "testing"
)

func TestGenerateConfig(t *testing.T) {
    configPath, err := GenerateConfig(12345, "/tmp/test.sock", "/usr/bin/mcp-server")
    if err != nil {
        t.Fatalf("GenerateConfig failed: %v", err)
    }
    defer os.Remove(configPath)

    // Read and verify config
    data, err := os.ReadFile(configPath)
    if err != nil {
        t.Fatalf("Failed to read config: %v", err)
    }

    var config MCPConfig
    if err := json.Unmarshal(data, &config); err != nil {
        t.Fatalf("Failed to parse config: %v", err)
    }

    server, ok := config.MCPServers["gofortress"]
    if !ok {
        t.Error("Missing 'gofortress' server in config")
    }

    if server.Type != "stdio" {
        t.Errorf("Expected type 'stdio', got %q", server.Type)
    }

    if server.Env["GOFORTRESS_SOCKET"] != "/tmp/test.sock" {
        t.Errorf("Socket path not set correctly")
    }
}
```

**Acceptance Criteria:**
- [ ] Config file created at /tmp/gofortress-mcp-{pid}.json
- [ ] File has 0600 permissions
- [ ] JSON is valid and parseable
- [ ] Server binary path resolved correctly
- [ ] Cleanup removes file

**Why This Matters:**
This config file is what tells Claude CLI where to find the MCP server. It must be ephemeral (per-instance) to avoid polluting global Claude configuration.

---

### 5.2 Phase 2 Tickets

---

#### GOgent-MCP-005: Modal State Management

**Time:** 4 hours
**Dependencies:** GOgent-MCP-001
**Priority:** HIGH

**Task:**
Add modal state management to the Claude panel for displaying prompts over the conversation view.

**File:** `internal/tui/claude/modal.go`

**Imports:**
```go
package claude

import (
    "strings"

    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)
```

**Implementation:**
```go
// ModalType identifies the kind of modal displayed
type ModalType int

const (
    NoModal ModalType = iota
    ConfirmModal
    TextInputModal
    SelectionModal
)

// ModalState holds the current modal's state
type ModalState struct {
    Active       bool
    Type         ModalType
    Prompt       callback.PromptRequest
    TextInput    textinput.Model
    SelectList   list.Model
    ResponseChan chan<- callback.PromptResponse
}

// NewModalState creates an empty modal state
func NewModalState() ModalState {
    ti := textinput.New()
    ti.Placeholder = "Type your response..."
    ti.CharLimit = 500
    ti.Width = 40

    return ModalState{
        TextInput: ti,
    }
}

// MCPPromptMsg is sent when a prompt request arrives
type MCPPromptMsg struct {
    Request      callback.PromptRequest
    ResponseChan chan<- callback.PromptResponse
}

// MCPResponseSentMsg is sent after response is delivered
type MCPResponseSentMsg struct {
    PromptID string
}

// HandlePrompt activates a modal for the given prompt
func (m *ModalState) HandlePrompt(prompt callback.PromptRequest, respChan chan<- callback.PromptResponse) tea.Cmd {
    m.Active = true
    m.Prompt = prompt
    m.ResponseChan = respChan

    switch prompt.Type {
    case "confirm":
        m.Type = ConfirmModal
        return nil

    case "input", "ask":
        if len(prompt.Options) > 0 {
            m.Type = SelectionModal
            m.SelectList = createSelectList(prompt.Options)
            return nil
        }
        m.Type = TextInputModal
        m.TextInput.Reset()
        if prompt.Default != "" {
            m.TextInput.SetValue(prompt.Default)
        }
        return m.TextInput.Focus()

    case "select":
        m.Type = SelectionModal
        m.SelectList = createSelectList(prompt.Options)
        return nil

    default:
        // Fallback to text input
        m.Type = TextInputModal
        m.TextInput.Reset()
        return m.TextInput.Focus()
    }
}

// SendResponse sends the response and closes the modal
func (m *ModalState) SendResponse(value string, cancelled bool) {
    if m.ResponseChan == nil {
        return
    }

    m.ResponseChan <- callback.PromptResponse{
        ID:        m.Prompt.ID,
        Value:     value,
        Cancelled: cancelled,
    }

    m.Active = false
    m.ResponseChan = nil
}

// listItem implements list.Item for selection
type listItem struct {
    title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }

func createSelectList(options []string) list.Model {
    items := make([]list.Item, len(options))
    for i, opt := range options {
        items[i] = listItem{title: opt}
    }

    delegate := list.NewDefaultDelegate()
    delegate.SetHeight(1)

    l := list.New(items, delegate, 40, min(len(options)+4, 12))
    l.SetShowTitle(false)
    l.SetShowStatusBar(false)
    l.SetShowHelp(false)
    l.SetFilteringEnabled(false)

    return l
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Acceptance Criteria:**
- [ ] ModalState tracks active prompt
- [ ] Handles all prompt types: confirm, input, select
- [ ] SendResponse delivers to channel
- [ ] State cleared after response

---

#### GOgent-MCP-006: Prompt Rendering

**Time:** 3 hours
**Dependencies:** GOgent-MCP-005
**Priority:** MEDIUM

**Task:**
Render modal prompts with lipgloss styling, overlaid on the conversation view. **CRITICAL:** Sanitize all MCP server prompts for ANSI escape sequence injection (staff-architect issue #3).

**File:** `internal/tui/claude/prompt.go`

**Security Note (From Staff Architect Review):**
MCP server prompts come from external tool calls and MUST be sanitized before display. Malicious prompts could inject ANSI sequences to manipulate terminal state, hide text, or create fake UI elements.

**Implementation:**
```go
package claude

import (
    "strings"

    "github.com/acarl005/stripansi"
    "github.com/charmbracelet/lipgloss"
)

// sanitizePrompt removes ANSI escape sequences from untrusted input
// CRITICAL: Must be called on all MCP server message content
func sanitizePrompt(s string) string {
    return stripansi.Strip(s)
}

var (
    modalStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1, 2).
        Width(50)

    modalTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("170"))

    modalHelpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("241"))

    destructiveStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("196")).
        Bold(true)
)

// RenderModal renders the current modal as a string
func (m *ModalState) RenderModal() string {
    if !m.Active {
        return ""
    }

    var content strings.Builder

    // Title/message - CRITICAL: Sanitize untrusted MCP input
    message := sanitizePrompt(m.Prompt.Message)
    if strings.HasPrefix(message, "⚠️") {
        content.WriteString(destructiveStyle.Render(message))
    } else {
        content.WriteString(modalTitleStyle.Render(message))
    }
    content.WriteString("\n\n")

    // Content based on type
    switch m.Type {
    case ConfirmModal:
        content.WriteString("[Y]es  [N]o  [Esc] Cancel")

    case TextInputModal:
        content.WriteString(m.TextInput.View())
        content.WriteString("\n\n")
        content.WriteString(modalHelpStyle.Render("[Enter] Submit  [Esc] Cancel"))

    case SelectionModal:
        content.WriteString(m.SelectList.View())
        content.WriteString("\n")
        content.WriteString(modalHelpStyle.Render("[Enter] Select  [↑/↓] Navigate  [Esc] Cancel"))
    }

    return modalStyle.Render(content.String())
}

// OverlayModal composites the modal over a background
func OverlayModal(background, modal string, width, height int) string {
    if modal == "" {
        return background
    }

    bgLines := strings.Split(background, "\n")
    modalLines := strings.Split(modal, "\n")

    modalWidth := lipgloss.Width(modal)
    modalHeight := len(modalLines)

    // Center the modal
    startX := max((width-modalWidth)/2, 0)
    startY := max((height-modalHeight)/2, 0)

    // Ensure background has enough lines
    for len(bgLines) < height {
        bgLines = append(bgLines, strings.Repeat(" ", width))
    }

    // Composite modal onto background
    for i, modalLine := range modalLines {
        bgY := startY + i
        if bgY < len(bgLines) {
            bgLines[bgY] = overlayLine(bgLines[bgY], modalLine, startX, width)
        }
    }

    return strings.Join(bgLines, "\n")
}

func overlayLine(background, overlay string, startX, maxWidth int) string {
    // Pad background to maxWidth
    for len(background) < maxWidth {
        background += " "
    }

    bgRunes := []rune(background)
    overlayRunes := []rune(overlay)

    for i, r := range overlayRunes {
        pos := startX + i
        if pos < len(bgRunes) {
            bgRunes[pos] = r
        }
    }

    return string(bgRunes)
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

**Acceptance Criteria:**
- [ ] Modal renders with border and padding
- [ ] Centered over background content
- [ ] Different styles for destructive actions
- [ ] Help text shows available keys
- [ ] **SECURITY:** ANSI escape sequences stripped from all MCP prompts
- [ ] **SECURITY:** Selection options also sanitized

**Security Test:**
```go
func TestSanitizePrompt(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"Normal text", "Normal text"},
        {"\x1b[31mRed text\x1b[0m", "Red text"},
        {"\x1b[2J\x1b[HClear and home", "Clear and home"},
        {"Text with\x1b]0;fake title\x07OSC", "Text withfake titleOSC"},
    }

    for _, tc := range tests {
        got := sanitizePrompt(tc.input)
        if got != tc.expected {
            t.Errorf("sanitizePrompt(%q) = %q, want %q", tc.input, got, tc.expected)
        }
    }
}
```

---

#### GOgent-MCP-007: Modal Input Handling

**Time:** 3 hours
**Dependencies:** GOgent-MCP-005, GOgent-MCP-006
**Priority:** MEDIUM

**Task:**
Handle keyboard input when a modal is active, routing to appropriate response actions.

**File:** Update `internal/tui/claude/input.go`

**Implementation (additions):**
```go
// HandleModalInput processes key presses when modal is active
// Returns true if the key was consumed by the modal
func (m *PanelModel) HandleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    if !m.modal.Active {
        return m, nil
    }

    switch msg.String() {
    case "enter":
        return m.submitModalResponse()

    case "esc":
        m.modal.SendResponse("", true)
        return m, nil

    case "y", "Y":
        if m.modal.Type == ConfirmModal {
            m.modal.SendResponse("yes", false)
            return m, nil
        }

    case "n", "N":
        if m.modal.Type == ConfirmModal {
            m.modal.SendResponse("no", false)
            return m, nil
        }
    }

    // Delegate to component
    var cmd tea.Cmd
    switch m.modal.Type {
    case TextInputModal:
        m.modal.TextInput, cmd = m.modal.TextInput.Update(msg)
    case SelectionModal:
        m.modal.SelectList, cmd = m.modal.SelectList.Update(msg)
    }

    return m, cmd
}

func (m *PanelModel) submitModalResponse() (tea.Model, tea.Cmd) {
    var value string

    switch m.modal.Type {
    case ConfirmModal:
        value = "yes"
    case TextInputModal:
        value = m.modal.TextInput.Value()
    case SelectionModal:
        if item, ok := m.modal.SelectList.SelectedItem().(listItem); ok {
            value = item.title
        }
    }

    m.modal.SendResponse(value, false)
    return m, nil
}
```

**Acceptance Criteria:**
- [ ] Enter submits current value
- [ ] Esc cancels with cancelled=true
- [ ] Y/N work for confirm modals
- [ ] Arrow keys navigate selections
- [ ] Text input captures typing

---

#### GOgent-MCP-008: External Event Integration

**Time:** 4 hours
**Dependencies:** GOgent-MCP-001, GOgent-MCP-007
**Priority:** HIGH

**Task:**
Integrate the callback server's prompt channel with Bubbletea's event loop using tea.Cmd. **CRITICAL:** Fix channel blocking issue where bare channel read blocks forever if TUI quits (staff-architect issue #4).

**File:** Update `internal/tui/claude/panel.go`

**Problem (From Staff Architect Review):**
Bare channel read `req := <-m.callbackServer.PromptChan` blocks forever if the TUI quits while waiting. This causes goroutine leaks and prevents clean shutdown.

**Implementation (additions):**
```go
// Add to PanelModel struct
type PanelModel struct {
    // ... existing fields ...

    callbackServer *callback.Server
    modal          ModalState
    ctx            context.Context // Added: cancellation context
}

// NewPanelModelWithCallback creates a panel with callback server
func NewPanelModelWithCallback(ctx context.Context, process ClaudeProcessInterface, cfg cli.Config, server *callback.Server) PanelModel {
    m := NewPanelModel(process, cfg)
    m.callbackServer = server
    m.modal = NewModalState()
    m.ctx = ctx // Store context for cancellation
    return m
}

// ListenForPrompts creates a command that waits for the next prompt
// CRITICAL: Uses select with context to prevent goroutine leak on shutdown
func (m PanelModel) ListenForPrompts() tea.Cmd {
    if m.callbackServer == nil {
        return nil
    }

    return func() tea.Msg {
        // FIXED: Use select with context.Done() to avoid blocking forever
        select {
        case req := <-m.callbackServer.PromptChan:
            // Create response channel
            respChan := make(chan callback.PromptResponse, 1)
            // Register with server
            m.callbackServer.RegisterPending(req.ID, respChan)
            return MCPPromptMsg{
                Request:      req,
                ResponseChan: respChan,
            }
        case <-m.ctx.Done():
            // TUI is shutting down, return nil to stop listening
            return nil
        }
    }
}

// Update in Update method
func (m PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case MCPPromptMsg:
        cmd := m.modal.HandlePrompt(msg.Request, msg.ResponseChan)
        return m, tea.Batch(cmd, m.ListenForPrompts())

    case tea.KeyMsg:
        if m.modal.Active {
            return m.HandleModalInput(msg)
        }
        // ... existing key handling ...
    }

    // ... existing update logic ...
}

// Update in View method
func (m PanelModel) View() string {
    main := m.renderMainContent()

    if m.modal.Active {
        return OverlayModal(main, m.modal.RenderModal(), m.width, m.height)
    }

    return main
}
```

**Acceptance Criteria:**
- [ ] Panel listens for prompts via tea.Cmd
- [ ] MCPPromptMsg triggers modal display
- [ ] Response delivered back to callback server
- [ ] Listens for next prompt after response
- [ ] Modal overlays conversation view
- [ ] **CRITICAL:** Goroutine exits cleanly on context cancellation
- [ ] **CRITICAL:** No goroutine leaks on TUI shutdown

**Shutdown Test:**
```go
func TestListenForPrompts_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    server := callback.NewServer(os.Getpid())
    _ = server.Start(ctx)
    defer server.Cleanup()

    panel := NewPanelModelWithCallback(ctx, nil, cli.Config{}, server)

    // Start listening command
    cmd := panel.ListenForPrompts()

    // Cancel context immediately
    cancel()

    // Command should return nil quickly, not block
    done := make(chan struct{})
    go func() {
        result := cmd()
        if result != nil {
            t.Errorf("Expected nil on cancellation, got %v", result)
        }
        close(done)
    }()

    select {
    case <-done:
        // Good - command exited
    case <-time.After(time.Second):
        t.Error("ListenForPrompts blocked after context cancellation")
    }
}
```

---

### 5.3 Phase 3 Tickets

---

#### GOgent-MCP-009: Main Orchestration

**Time:** 4 hours
**Dependencies:** Phase 1 + Phase 2 + GOgent-MCP-000 (lifecycle)
**Priority:** HIGH

**Task:**
Update gofortress main.go to start callback server, generate MCP config, and wire everything together. **CRITICAL:** Integrate lifecycle management for signal handling and crash recovery (staff-architect issue #1, #2).

**File:** `cmd/gofortress/main.go`

**Implementation (modifications):**
```go
package main

import (
    // ... existing imports ...

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
    "github.com/Bucket-Chemist/GOgent-Fortress/internal/lifecycle"
    "github.com/Bucket-Chemist/GOgent-Fortress/internal/mcp"
)

func main() {
    flag.Parse()

    // ... existing flag handling ...

    // CRITICAL: Clean up stale sockets from previous crashed sessions
    // Must run BEFORE creating new socket to prevent "address in use" errors
    if err := lifecycle.CleanupStaleSockets(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: stale socket cleanup failed: %v\n", err)
    }

    // Start callback server for MCP integration
    pid := os.Getpid()
    callbackServer := callback.NewServer(pid)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // CRITICAL: Set up process lifecycle manager for signal handling
    processManager := lifecycle.NewProcessManager(callbackServer.SocketPath())
    processManager.StartSignalHandler(ctx, func() {
        cancel() // Cancel context to unblock listeners
        callbackServer.Shutdown(context.Background())
    })

    var mcpConfigPath string
    var mcpEnabled bool

    if err := callbackServer.Start(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: MCP callback server failed: %v\n", err)
        fmt.Fprintf(os.Stderr, "         Interactive prompts will be disabled.\n")
    } else {
        mcpEnabled = true
        defer callbackServer.Cleanup()
        defer callbackServer.Shutdown(ctx)

        // Find MCP server binary
        serverBinary, err := mcp.FindServerBinary()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
            fmt.Fprintf(os.Stderr, "         Interactive prompts will be disabled.\n")
            mcpEnabled = false
        } else {
            // Generate MCP config
            mcpConfigPath, err = mcp.GenerateConfig(pid, callbackServer.SocketPath(), serverBinary)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
                mcpEnabled = false
            } else {
                defer mcp.Cleanup(mcpConfigPath)
            }
        }
    }

    // ... existing session manager creation ...

    // Build config
    cfg := cli.Config{
        ClaudePath:   "claude",
        SessionID:    sessionToResume,
        WorkingDir:   workDir,
        Verbose:      *verbose,
        AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput", "EnterPlanMode", "ExitPlanMode"},
    }

    // Add MCP if enabled
    if mcpEnabled {
        cfg.MCPConfigPath = mcpConfigPath
        cfg.AllowedTools = append(cfg.AllowedTools,
            "mcp__gofortress__ask_user",
            "mcp__gofortress__confirm_action",
            "mcp__gofortress__request_input",
            "mcp__gofortress__select_option",
        )
    }

    // ... existing process creation ...

    // Create TUI with callback server (pass ctx for clean shutdown)
    var claudePanel claude.PanelModel
    if mcpEnabled {
        claudePanel = claude.NewPanelModelWithCallback(ctx, process, cfg, callbackServer)
    } else {
        claudePanel = claude.NewPanelModel(process, cfg)
    }

    // CRITICAL: Register Claude process with lifecycle manager for signal propagation
    // This ensures SIGTERM is forwarded to Claude if gofortress is killed
    if claudeProcess := process.GetProcess(); claudeProcess != nil {
        processManager.SetChildProcess(claudeProcess)
    }

    // ... rest of existing main ...
}
```

**Acceptance Criteria:**
- [ ] Callback server starts before Claude process
- [ ] MCP config generated with correct paths
- [ ] MCP tools added to AllowedTools
- [ ] Graceful degradation if MCP setup fails
- [ ] Cleanup on exit (socket, config file)
- [ ] **CRITICAL:** Stale sockets cleaned at startup
- [ ] **CRITICAL:** SIGTERM propagated to Claude child process
- [ ] **CRITICAL:** Context cancelled to unblock listeners on shutdown

---

#### GOgent-MCP-010: Session Isolation Verification

**Time:** 2 hours
**Dependencies:** GOgent-MCP-009
**Priority:** HIGH

**Task:**
Verify that gofortress MCP integration has zero impact on regular goclaude/claude CLI usage.

**File:** `internal/mcp/isolation_test.go`

**Implementation:**
```go
package mcp

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestSessionIsolation_NoGlobalConfig(t *testing.T) {
    // Verify claude mcp list shows no gofortress server
    cmd := exec.Command("claude", "mcp", "list")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Skipf("Claude CLI not available: %v", err)
    }

    if strings.Contains(string(output), "gofortress") {
        t.Error("gofortress MCP server found in global config - session isolation violated!")
    }
}

func TestSessionIsolation_ConfigIsEphemeral(t *testing.T) {
    pid := 99999
    socketPath := "/tmp/test-isolation.sock"

    configPath, err := GenerateConfig(pid, socketPath, "/usr/bin/mcp-server")
    if err != nil {
        t.Fatalf("GenerateConfig failed: %v", err)
    }

    // Verify config is in /tmp, not ~/.claude/
    if !strings.HasPrefix(configPath, os.TempDir()) {
        t.Errorf("Config not in temp dir: %s", configPath)
    }

    // Verify it doesn't exist in user's claude config
    userConfig := filepath.Join(os.Getenv("HOME"), ".claude", "mcp-servers.json")
    if _, err := os.Stat(userConfig); err == nil {
        data, _ := os.ReadFile(userConfig)
        if strings.Contains(string(data), "gofortress") {
            t.Error("gofortress found in user MCP config - isolation violated!")
        }
    }

    // Cleanup
    Cleanup(configPath)

    // Verify config removed
    if _, err := os.Stat(configPath); !os.IsNotExist(err) {
        t.Error("Config file not cleaned up")
    }
}
```

**Acceptance Criteria:**
- [ ] `claude mcp list` shows no gofortress server
- [ ] Config file is in /tmp, not ~/.claude/
- [ ] Config file removed after gofortress exits
- [ ] Multiple gofortress instances don't conflict

---

### 5.4 Phase 4 Tickets

---

#### GOgent-MCP-013: Error Handling and Recovery

**Time:** 4 hours
**Dependencies:** Phase 3
**Priority:** MEDIUM

**Task:**
Implement comprehensive error handling with retries and graceful degradation.

**File:** `internal/callback/recovery.go`

**Implementation:**
```go
package callback

import (
    "context"
    "fmt"
    "time"
)

// SendPromptWithRetry attempts to send a prompt with exponential backoff
func (c *Client) SendPromptWithRetry(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    var lastErr error
    backoff := 100 * time.Millisecond
    maxRetries := 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := c.SendPrompt(ctx, req)
        if err == nil {
            return resp, nil
        }
        lastErr = err

        // Don't retry on context cancellation
        if ctx.Err() != nil {
            return PromptResponse{}, ctx.Err()
        }

        select {
        case <-ctx.Done():
            return PromptResponse{}, ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
        }
    }

    return PromptResponse{}, fmt.Errorf("[callback-client] Max retries exceeded: %w", lastErr)
}

// ServerHealthMonitor periodically checks server health
type ServerHealthMonitor struct {
    client   *Client
    interval time.Duration
    healthy  bool
    onUnhealthy func()
}

func NewHealthMonitor(client *Client, interval time.Duration, onUnhealthy func()) *ServerHealthMonitor {
    return &ServerHealthMonitor{
        client:      client,
        interval:    interval,
        healthy:     true,
        onUnhealthy: onUnhealthy,
    }
}

func (m *ServerHealthMonitor) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(m.interval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                err := m.client.HealthCheck(checkCtx)
                cancel()

                wasHealthy := m.healthy
                m.healthy = err == nil

                if wasHealthy && !m.healthy && m.onUnhealthy != nil {
                    m.onUnhealthy()
                }
            }
        }
    }()
}

func (m *ServerHealthMonitor) IsHealthy() bool {
    return m.healthy
}
```

**Acceptance Criteria:**
- [ ] Retries with exponential backoff
- [ ] Respects context cancellation
- [ ] Health monitor detects failures
- [ ] Callback on health state change

---

#### GOgent-MCP-015: Comprehensive Test Suite

**Time:** 6 hours
**Dependencies:** All previous tickets
**Priority:** HIGH

**Task:**
Create comprehensive test suite covering all MCP integration scenarios.

**File:** `internal/mcp/integration_test.go`

**Implementation:**
```go
package mcp

import (
    "context"
    "os"
    "sync"
    "testing"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

func TestMCPIntegration_FullRoundTrip(t *testing.T) {
    // Start callback server
    server := callback.NewServer(os.Getpid())
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    // Create client
    client := callback.NewClientWithPath(server.SocketPath())

    // Simulate TUI handling prompts
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for req := range server.PromptChan {
            // Simulate user response
            time.Sleep(50 * time.Millisecond)
            server.SendResponse(callback.PromptResponse{
                ID:    req.ID,
                Value: "user-response",
            })
        }
    }()

    // Send multiple prompts
    for i := 0; i < 5; i++ {
        resp, err := client.SendPrompt(ctx, callback.PromptRequest{
            Type:    "ask",
            Message: "Test question?",
        })
        if err != nil {
            t.Errorf("Prompt %d failed: %v", i, err)
        }
        if resp.Value != "user-response" {
            t.Errorf("Expected 'user-response', got %q", resp.Value)
        }
    }
}

func TestMCPIntegration_ConcurrentPrompts(t *testing.T) {
    server := callback.NewServer(os.Getpid())
    ctx := context.Background()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    client := callback.NewClientWithPath(server.SocketPath())

    // Handle prompts
    go func() {
        for req := range server.PromptChan {
            server.SendResponse(callback.PromptResponse{
                ID:    req.ID,
                Value: req.ID, // Echo back the ID
            })
        }
    }()

    // Send concurrent prompts
    var wg sync.WaitGroup
    errors := make(chan error, 10)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            resp, err := client.SendPrompt(ctx, callback.PromptRequest{
                ID:      id,
                Type:    "ask",
                Message: "Concurrent test",
            })
            if err != nil {
                errors <- err
                return
            }
            if resp.Value != id {
                errors <- fmt.Errorf("ID mismatch: expected %s, got %s", id, resp.Value)
            }
        }(fmt.Sprintf("concurrent-%d", i))
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Error(err)
    }
}

func TestMCPIntegration_Timeout(t *testing.T) {
    server := callback.NewServer(os.Getpid())
    ctx := context.Background()

    if err := server.Start(ctx); err != nil {
        t.Fatalf("Failed to start server: %v", err)
    }
    defer server.Cleanup()
    defer server.Shutdown(ctx)

    client := callback.NewClientWithPath(server.SocketPath())
    client.httpClient.Timeout = 100 * time.Millisecond

    // Don't handle prompt - should timeout
    go func() {
        <-server.PromptChan // Receive but don't respond
    }()

    _, err := client.SendPrompt(ctx, callback.PromptRequest{
        Type:    "ask",
        Message: "Should timeout",
    })

    if err == nil {
        t.Error("Expected timeout error, got nil")
    }
}
```

**Acceptance Criteria:**
- [ ] Full round-trip test passes
- [ ] Concurrent prompt handling works
- [ ] Timeout behavior correct
- [ ] Memory leak tests (24h simulation)
- [ ] Coverage >80% for all MCP packages

---

## 6. Code Organization

### 6.1 Directory Structure

```
GOgent-Fortress/
├── cmd/
│   ├── gofortress/
│   │   └── main.go                    # Updated with MCP orchestration
│   └── gofortress-mcp-server/
│       └── main.go                    # NEW: MCP server binary
│
├── internal/
│   ├── callback/
│   │   ├── server.go                  # NEW: Unix socket HTTP server
│   │   ├── client.go                  # NEW: Client for MCP server
│   │   ├── recovery.go                # NEW: Retry/health logic
│   │   ├── server_test.go
│   │   └── client_test.go
│   │
│   ├── mcp/
│   │   ├── config.go                  # NEW: Config generation
│   │   ├── isolation_test.go          # NEW: Session isolation tests
│   │   └── integration_test.go        # NEW: E2E tests
│   │
│   ├── cli/
│   │   └── subprocess.go              # Updated: MCPConfigPath field
│   │
│   └── tui/
│       └── claude/
│           ├── panel.go               # Updated: Modal integration
│           ├── modal.go               # NEW: Modal state
│           ├── prompt.go              # NEW: Prompt rendering
│           ├── input.go               # Updated: Modal input handling
│           └── modal_test.go          # NEW: Modal tests
│
└── docs/
    └── MCP_IMPLEMENTATION_GUIDE_V2.md # This document
```

### 6.2 Package Dependencies

```
cmd/gofortress
    ├── internal/callback
    ├── internal/mcp
    ├── internal/cli
    └── internal/tui/claude

cmd/gofortress-mcp-server
    └── internal/callback

internal/tui/claude
    └── internal/callback

internal/callback
    └── (standard library only)

internal/mcp
    └── (standard library only)
```

---

## 7. Tool Catalog

### 7.1 Built-in MCP Tools

| Tool Name | Claude Sees | Purpose | Input | Output |
|-----------|-------------|---------|-------|--------|
| `ask_user` | `mcp__gofortress__ask_user` | Ask question with optional choices | message, options[], default | response, cancelled |
| `confirm_action` | `mcp__gofortress__confirm_action` | Yes/no confirmation | action, destructive | confirmed, cancelled |
| `request_input` | `mcp__gofortress__request_input` | Free-form text input | prompt, placeholder | input, cancelled |
| `select_option` | `mcp__gofortress__select_option` | Select from list | message, options[] | selected, index, cancelled |

### 7.2 Tool Usage Examples

**ask_user with options:**
```json
{
  "message": "Which database should I use?",
  "options": ["SQLite", "PostgreSQL", "MySQL"]
}
// Response: {"response": "SQLite", "cancelled": false}
```

**confirm_action (destructive):**
```json
{
  "action": "Delete all test files (50 files)",
  "destructive": true
}
// Response: {"confirmed": true, "cancelled": false}
```

**request_input:**
```json
{
  "prompt": "Enter the API key:",
  "placeholder": "sk-..."
}
// Response: {"input": "sk-abc123", "cancelled": false}
```

---

## 8. Testing Strategy

### 8.1 Test Pyramid

| Level | Coverage Target | Focus |
|-------|-----------------|-------|
| Unit | >85% | Individual functions, edge cases |
| Integration | >75% | Server-client communication |
| E2E | Key flows | Full prompt round-trip |
| Manual | Acceptance | Visual, UX verification |

### 8.2 Test Fixtures

**Location:** `test/fixtures/mcp/`

```
test/fixtures/mcp/
├── prompts/
│   ├── ask_simple.json
│   ├── ask_with_options.json
│   ├── confirm_normal.json
│   ├── confirm_destructive.json
│   └── input_with_default.json
├── responses/
│   ├── success.json
│   ├── cancelled.json
│   └── timeout.json
└── configs/
    └── valid_mcp_config.json
```

### 8.3 CI/CD Integration

```yaml
# .github/workflows/mcp.yml
name: MCP Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build MCP Server
        run: go build ./cmd/gofortress-mcp-server

      - name: Run Unit Tests
        run: go test -race -coverprofile=coverage.out ./internal/callback/... ./internal/mcp/...

      - name: Run Integration Tests
        run: go test -race ./internal/mcp/... -tags=integration

      - name: Check Coverage
        run: |
          go tool cover -func=coverage.out | grep total | awk '{print $3}'
          # Fail if below 80%
```

---

## 9. Operational Considerations

### 9.1 Performance Characteristics

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Socket IPC round-trip | <3ms | Benchmark test |
| Prompt display | <100ms | User perception |
| Modal render | <16ms | 60fps refresh |
| Memory overhead | <10MB | MCP server process |
| File descriptors | <10 | Socket + pipes |

### 9.2 Resource Cleanup

**On normal exit:**
```go
defer callbackServer.Cleanup()  // Remove socket file
defer callbackServer.Shutdown() // Close connections
defer mcp.Cleanup(configPath)   // Remove config file
```

**On crash (handled by OS):**
- Socket file removed by runtime dir cleanup
- Config file removed by /tmp cleanup
- MCP server process terminated (child of Claude, grandchild of gofortress)

### 9.3 Debugging

**Enable verbose logging:**
```bash
# In gofortress
export LOG_LEVEL=debug
gofortress

# In MCP server (set in config env)
"env": {"LOG_LEVEL": "debug"}
```

**Check socket communication:**
```bash
# Test health endpoint
curl --unix-socket /run/user/1000/gofortress-12345.sock http://localhost/health
```

---

## 10. Migration Path

### 10.1 Installation

```bash
# Build both binaries
go build ./cmd/gofortress
go build ./cmd/gofortress-mcp-server

# Install to PATH
sudo mv gofortress gofortress-mcp-server /usr/local/bin/
```

### 10.2 Verification

```bash
# Check MCP isolation
claude mcp list  # Should NOT show gofortress

# Run gofortress
gofortress

# In another terminal, verify temp files
ls /tmp/gofortress-mcp-*.json  # Should exist while running
ls /run/user/$UID/gofortress-*.sock  # Socket should exist

# After gofortress exits
ls /tmp/gofortress-mcp-*.json  # Should be gone
```

### 10.3 Rollback

If MCP integration causes issues:

1. MCP is optional - gofortress works without it
2. Set `MCP_DISABLED=1` environment variable
3. Remove gofortress-mcp-server binary to fully disable

---

## Appendix A: Ticket Extraction Script

To extract tickets from this document, use the script at `scripts/extract-mcp-tickets.sh`.

See the script for usage details. The script:
- Extracts each `#### GOgent-MCP-XXX:` section into individual ticket files
- Adds YAML frontmatter with metadata (ID, time, priority, dependencies)
- Creates an index file listing all tickets
- Validates ticket structure

---

## Appendix B: Quick Reference

### Command Cheatsheet

```bash
# Build everything
make build-mcp

# Run tests
make test-mcp

# Run with debug logging
LOG_LEVEL=debug gofortress

# Check MCP server health
curl --unix-socket $XDG_RUNTIME_DIR/gofortress-$$.sock http://localhost/health
```

### Troubleshooting

| Symptom | Cause | Solution |
|---------|-------|----------|
| "GOFORTRESS_SOCKET not set" | MCP server started outside gofortress | Must be spawned via MCP config |
| "TUI health check failed" | Socket server not running | Check gofortress started correctly |
| Prompts not appearing | Modal not wired up | Check callback server in panel |
| "Socket path too long" | Path > 108 chars | Use shorter XDG_RUNTIME_DIR |

---

---

## Appendix C: Staff Architect Critical Review

**Review Date:** 2026-01-30
**Reviewer:** staff-architect-critical-review
**Overall Verdict:** APPROVE WITH CHANGES

### Critical Issues (MUST FIX)

#### 1. Process Lifecycle Gaps
**Problem:** No signal handler to propagate SIGTERM to child processes on crash.
**Fix Required:**
```go
// cmd/gofortress/main.go - add signal handler
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-sigChan
    if claudeProcess != nil {
        claudeProcess.Process.Signal(syscall.SIGTERM)
    }
    callbackServer.Shutdown(ctx)
}()
```

#### 2. Crash Recovery Missing
**Problem:** Stale sockets from previous crashes not cleaned up.
**Fix Required:**
```go
func cleanupStaleSockets() {
    pattern := filepath.Join(os.TempDir(), "gofortress-*.sock")
    matches, _ := filepath.Glob(pattern)
    for _, path := range matches {
        pid := extractPID(path)
        if !processExists(pid) {
            os.Remove(path)
        }
    }
}
```

#### 3. Input Sanitization Missing
**Problem:** MCP server prompts not sanitized for ANSI injection.
**Fix Required:**
```go
import "github.com/acarl005/stripansi"
sanitized := stripansi.Strip(m.currentPrompt.Message)
content.WriteString(titleStyle.Render(sanitized))
```

#### 4. Bubbletea External Event Handling
**Problem:** Bare channel read blocks forever if TUI quits.
**Fix Required:**
```go
func (m Model) listenForPrompts() tea.Cmd {
    return func() tea.Msg {
        select {
        case prompt := <-m.callbackServer.PromptChan:
            return MCPPromptMsg(prompt)
        case <-m.ctx.Done():
            return nil
        }
    }
}
```

### Important Issues (SHOULD FIX)

1. **Latency claims:** Change "<3ms" to "<10ms p95" with actual benchmarks
2. **Modal overlay:** Use lipgloss `Place()` instead of manual rune manipulation
3. **Tickets incomplete:** Phase 2-4 tickets need full implementation code

### Revised Timeline

| Phase | Original | Revised | Reason |
|-------|----------|---------|--------|
| Pre-implementation | - | 1 week | Fix critical issues |
| Phase 1 | 1 week | 1.5 weeks | Add crash recovery |
| Phase 2 | 1 week | 1.5 weeks | Fix event handling |
| Phase 3 | 1 week | 1 week | As planned |
| Phase 4 | 1 week | 1.5 weeks | Hardening + fixes |
| **Total** | 4 weeks | **6-7 weeks** | Production-ready |

### Verdict Summary

✅ **Architecture is SOUND** - Three-process hierarchy with stdio + Unix socket callback is correct
✅ **Technical approach is FEASIBLE** - Go MCP SDK usage is correct
✅ **Benefits justify complexity** - MCP extensibility is worth the overhead
⚠️ **Operational gaps need hardening** - Crash recovery, signal handling
⚠️ **Tickets need expansion** - Phase 2-4 not copy-paste ready

**Recommendation:** Proceed with implementation after addressing critical issues.

---

**End of MCP Implementation Guide v2.0**

*This document supersedes MCP_IMPLEMENTATION_GUIDE.md (v1.0) and incorporates corrections from einstein-mcp-synthesis-20260129.md*
