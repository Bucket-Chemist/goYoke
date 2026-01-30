---
id: GOgent-MCP-004
title: "MCP Config Generator"
description: "Generate ephemeral MCP configuration JSON that points Claude to the gofortress-mcp-server binary with the correct socket path."
time_estimate: "2h"
priority: HIGH
dependencies: ["GOgent-MCP-003"]
status: pending
---

# GOgent-MCP-004: MCP Config Generator


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
- [x] Config file created at /tmp/gofortress-mcp-{pid}.json
- [x] File has 0600 permissions
- [x] JSON is valid and parseable
- [x] Server binary path resolved correctly
- [x] Cleanup removes file

**Why This Matters:**
This config file is what tells Claude CLI where to find the MCP server. It must be ephemeral (per-instance) to avoid polluting global Claude configuration.


