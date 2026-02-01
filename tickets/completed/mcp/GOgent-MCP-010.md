---
id: GOgent-MCP-010
title: "Session Isolation Verification"
description: "Verify that gofortress MCP integration has zero impact on regular goclaude/claude CLI usage"
time_estimate: "2h"
priority: HIGH
dependencies: ["GOgent-MCP-009"]
status: completed
---

# GOgent-MCP-010: Session Isolation Verification


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
- [x] `claude mcp list` shows no gofortress server
- [x] Config file is in /tmp, not ~/.claude/
- [x] Config file removed after gofortress exits
- [x] Multiple gofortress instances don't conflict


