package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
