# Task 2.6: End-to-End Integration

**Phase:** Phase 4: Extensibility (Weeks 7-8)
**Task ID:** 2.6
**Status:** Not Started

---

- **Owner:** go-tui (Sonnet)
- **Files:** `cmd/goyoke/main.go`
- **Complexity:** Medium
- **Time:** 2 days
- **Dependencies:** Tasks 2.2, 2.3, 2.5

**Subtasks:**
1. Start MCP server in main
2. Create channels for IPC
3. Pass channels to TUI and MCP server
4. Configure Claude CLI with MCP config (ISOLATED, not global)
5. Add mcp__goyoke__ask_user to AllowedTools
6. Manual testing
7. Integration tests

**Acceptance:**
- [ ] goyoke starts with MCP server
- [ ] Claude CLI connects to MCP server
- [ ] ask_user tool calls work end-to-end
- [ ] User can see and respond to prompts
- [ ] Claude continues with user's choice
- [ ] Regular `goclaude` sessions unaffected (CRITICAL)

**Implementation Example:**

```go
// cmd/goyoke/main.go
func main() {
    // Get unique paths for this instance
    pid := os.Getpid()
    socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("goyoke-mcp-%d.sock", pid))
    mcpConfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("goyoke-mcp-%d.json", pid))

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

        // Generate MCP config for THIS goyoke instance only
        mcpConfigJSON := fmt.Sprintf(`{
          "mcpServers": {
            "goyoke": {
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
        claudeCfg.AllowedTools = append(claudeCfg.AllowedTools, "mcp__goyoke__ask_user")
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

## Status Tracking

- [ ] Task assigned to agent
- [ ] Dependencies reviewed
- [ ] Implementation started
- [ ] Code written
- [ ] Tests written
- [ ] Tests passing
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Task complete

## Notes

(Add implementation notes, blockers, or questions here)
