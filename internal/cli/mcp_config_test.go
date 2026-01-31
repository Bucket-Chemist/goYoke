package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClaudeProcess_MCPConfig verifies --mcp-config flag is added when MCPConfigPath is set
func TestNewClaudeProcess_MCPConfig(t *testing.T) {
	// Create a temporary config file
	configPath := os.TempDir() + "/test-mcp-config.json"
	err := os.WriteFile(configPath, []byte(`{"mcpServers":{}}`), 0600)
	require.NoError(t, err)
	defer os.Remove(configPath)

	cfg := Config{
		ClaudePath:    "./testdata/mock-claude",
		MCPConfigPath: configPath,
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Check that --mcp-config flag was added
	args := proc.cmd.Args
	found := false
	for i, arg := range args {
		if arg == "--mcp-config" {
			found = true
			require.Less(t, i+1, len(args), "Should have value after --mcp-config")
			assert.Equal(t, configPath, args[i+1], "MCP config path should match")
			break
		}
	}

	assert.True(t, found, "Should have --mcp-config flag")
}

// TestNewClaudeProcess_NoMCPConfig verifies --mcp-config is omitted when not set
func TestNewClaudeProcess_NoMCPConfig(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Check that --mcp-config flag was NOT added
	args := proc.cmd.Args
	for _, arg := range args {
		assert.NotEqual(t, "--mcp-config", arg, "Should not have --mcp-config flag when not configured")
	}
}

// TestNewClaudeProcess_MCPConfigWithAllowedTools verifies MCP tools can be combined with other allowed tools
func TestNewClaudeProcess_MCPConfigWithAllowedTools(t *testing.T) {
	configPath := os.TempDir() + "/test-mcp-config.json"
	err := os.WriteFile(configPath, []byte(`{"mcpServers":{}}`), 0600)
	require.NoError(t, err)
	defer os.Remove(configPath)

	cfg := Config{
		ClaudePath:    "./testdata/mock-claude",
		MCPConfigPath: configPath,
		AllowedTools: []string{
			"Bash",
			"Read",
			"mcp__gofortress__ask_user",
			"mcp__gofortress__confirm_action",
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Build complete command string for inspection
	cmdString := strings.Join(proc.cmd.Args, " ")

	// Check both --mcp-config and --allowed-tools are present
	assert.Contains(t, cmdString, "--mcp-config", "Should have --mcp-config")
	assert.Contains(t, cmdString, "--allowed-tools Bash", "Should allow Bash")
	assert.Contains(t, cmdString, "--allowed-tools mcp__gofortress__ask_user", "Should allow MCP ask_user")

	// Count --allowed-tools flags
	allowedCount := 0
	for _, arg := range proc.cmd.Args {
		if arg == "--allowed-tools" {
			allowedCount++
		}
	}
	assert.Equal(t, 4, allowedCount, "Should have 4 --allowed-tools flags")
}

// TestGetProcess verifies GetProcess returns the underlying os.Process
func TestGetProcess(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Before start, should return nil
	osproc := proc.GetProcess()
	assert.Nil(t, osproc, "Should return nil before process starts")

	// Start process
	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	// After start, should return non-nil
	osproc = proc.GetProcess()
	assert.NotNil(t, osproc, "Should return os.Process after start")
	assert.Greater(t, osproc.Pid, 0, "PID should be positive")

	// Stop process
	proc.Stop()

	// After stop, may still have process reference (depends on timing)
	// We don't assert on this because it's implementation-dependent
}
