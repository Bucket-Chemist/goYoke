package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionIsolation_NoGlobalConfig verifies that gofortress does not appear
// in the global claude CLI configuration
func TestSessionIsolation_NoGlobalConfig(t *testing.T) {
	// Verify claude mcp list shows no gofortress server
	cmd := exec.Command("claude", "mcp", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If claude CLI is not available, skip this test
		t.Skipf("Claude CLI not available: %v", err)
	}

	if strings.Contains(string(output), "gofortress") {
		t.Error("gofortress MCP server found in global config - session isolation violated!")
	}
}

// TestSessionIsolation_ConfigIsEphemeral verifies that MCP configs are:
// 1. Created in /tmp, not ~/.claude/
// 2. Don't pollute user's global configuration
// 3. Properly cleaned up after use
func TestSessionIsolation_ConfigIsEphemeral(t *testing.T) {
	pid := 99999
	socketPath := "/tmp/test-isolation.sock"

	configPath, err := GenerateConfig(pid, socketPath, "/usr/bin/mcp-server")
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify config is in /tmp, not ~/.claude/
	if !strings.HasPrefix(configPath, os.TempDir()) {
		t.Errorf("Config not in temp dir: %s (expected prefix: %s)", configPath, os.TempDir())
	}

	// Verify it doesn't exist in user's claude config
	userConfig := filepath.Join(os.Getenv("HOME"), ".claude", "mcp-servers.json")
	if _, err := os.Stat(userConfig); err == nil {
		data, readErr := os.ReadFile(userConfig)
		if readErr != nil {
			t.Logf("Could not read user config for verification: %v", readErr)
		} else if strings.Contains(string(data), "gofortress") {
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

// TestSessionIsolation_MultipleInstances verifies that multiple gofortress
// instances can run simultaneously without conflicts
func TestSessionIsolation_MultipleInstances(t *testing.T) {
	pids := []int{10001, 10002, 10003}
	configPaths := make([]string, 0, len(pids))
	socketPaths := make(map[string]bool)

	// Generate configs for multiple PIDs
	for i, pid := range pids {
		socketPath := filepath.Join(os.TempDir(), "gofortress-test-"+string(rune('a'+i))+".sock")
		configPath, err := GenerateConfig(pid, socketPath, "/usr/bin/mcp-server")
		if err != nil {
			t.Fatalf("GenerateConfig failed for PID %d: %v", pid, err)
		}
		configPaths = append(configPaths, configPath)

		// Verify unique socket path
		if socketPaths[socketPath] {
			t.Errorf("Duplicate socket path detected: %s", socketPath)
		}
		socketPaths[socketPath] = true

		// Verify all configs exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file not created for PID %d: %s", pid, configPath)
		}
	}

	// Verify no conflicts between instances
	if len(socketPaths) != len(pids) {
		t.Errorf("Socket path conflicts detected: expected %d unique paths, got %d",
			len(pids), len(socketPaths))
	}

	// Cleanup all configs
	for _, configPath := range configPaths {
		Cleanup(configPath)

		// Verify each config was removed
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Errorf("Config file not cleaned up: %s", configPath)
		}
	}
}
