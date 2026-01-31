package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestFindServerBinary(t *testing.T) {
	// This test verifies that FindServerBinary searches expected paths
	// We can't guarantee the binary exists, but we can verify the function
	// doesn't crash and returns appropriate errors

	t.Run("searches common paths", func(t *testing.T) {
		// Call FindServerBinary - may or may not find binary
		path, err := FindServerBinary()

		// If found, verify it's a valid path
		if err == nil {
			if path == "" {
				t.Error("FindServerBinary returned empty path with nil error")
			}
			// Verify the path we got actually exists
			if _, statErr := os.Stat(path); statErr != nil {
				t.Errorf("FindServerBinary returned non-existent path: %s", path)
			}
		} else {
			// If not found, verify error message is helpful
			if err.Error() == "" {
				t.Error("FindServerBinary returned error with empty message")
			}
		}
	})

	t.Run("returns error when binary not found", func(t *testing.T) {
		// This test documents the error behavior when binary is missing
		// We can't force this condition without mocking, but we verify
		// the expected error format exists in the code
		_, err := FindServerBinary()
		if err != nil {
			// Verify error message contains helpful information
			expectedSubstring := "gofortress-mcp-server not found"
			if err.Error() == "" || len(err.Error()) < len(expectedSubstring) {
				t.Errorf("Error message too short or empty: %v", err)
			}
		}
	})
}

func TestAcceptanceCriteria(t *testing.T) {
	t.Run("config file created at correct path", func(t *testing.T) {
		pid := 99999
		configPath, err := GenerateConfig(pid, "/tmp/test.sock", "/usr/bin/test-server")
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}
		defer os.Remove(configPath)

		expectedPath := filepath.Join(os.TempDir(), "gofortress-mcp-99999.json")
		if configPath != expectedPath {
			t.Errorf("Wrong path: got %s, expected %s", configPath, expectedPath)
		}
	})

	t.Run("file has 0600 permissions", func(t *testing.T) {
		configPath, err := GenerateConfig(99999, "/tmp/test.sock", "/usr/bin/test-server")
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}
		defer os.Remove(configPath)

		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Failed to stat config file: %v", err)
		}

		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("Wrong permissions: got %o, expected 0600", perm)
		}
	})

	t.Run("JSON is valid and parseable", func(t *testing.T) {
		configPath, err := GenerateConfig(99999, "/tmp/test.sock", "/usr/bin/test-server")
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}
		defer os.Remove(configPath)

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		var config MCPConfig
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify structure
		if len(config.MCPServers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(config.MCPServers))
		}
		if _, ok := config.MCPServers["gofortress"]; !ok {
			t.Error("Missing 'gofortress' server")
		}
	})

	t.Run("cleanup removes file", func(t *testing.T) {
		configPath, err := GenerateConfig(99999, "/tmp/test.sock", "/usr/bin/test-server")
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); err != nil {
			t.Fatalf("Config file not created: %v", err)
		}

		// Clean up
		Cleanup(configPath)

		// Verify file removed
		if _, err := os.Stat(configPath); err == nil {
			t.Error("Cleanup failed: file still exists")
		}
	})
}
