---
id: GOgent-MCP-009
title: "Main Orchestration"
time: "4 hours"
priority: HIGH
dependencies: "Phase 1 + Phase 2 + GOgent-MCP-000 (lifecycle)"
status: pending
---

# GOgent-MCP-009: Main Orchestration


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


