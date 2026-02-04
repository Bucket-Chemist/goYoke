package routing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConventionContent(t *testing.T) {
	// This test requires actual convention files to exist
	// Skip if not in the right environment
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		t.Skip("Cannot determine config dir")
	}

	goConventionPath := filepath.Join(configDir, "conventions", "go.md")
	if _, err := os.Stat(goConventionPath); os.IsNotExist(err) {
		t.Skip("go.md convention file not found")
	}

	content, err := LoadConventionContent("go.md")
	if err != nil {
		t.Fatalf("Failed to load go.md: %v", err)
	}

	if len(content) == 0 {
		t.Error("go.md content is empty")
	}

	// Test caching - second load should use cache
	content2, err := LoadConventionContent("go.md")
	if err != nil {
		t.Fatalf("Failed to load go.md from cache: %v", err)
	}

	if content != content2 {
		t.Error("Cache returned different content")
	}
}

func TestLoadConventionContentNotFound(t *testing.T) {
	_, err := LoadConventionContent("nonexistent-convention.md")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestClearConventionCache(t *testing.T) {
	// Load something to populate cache
	LoadConventionContent("go.md") // Ignore error

	// Clear cache
	ClearConventionCache()

	// Verify cache is empty
	cacheMutex.RLock()
	cacheLen := len(conventionCache)
	cacheMutex.RUnlock()

	if cacheLen != 0 {
		t.Errorf("Cache not cleared, has %d entries", cacheLen)
	}
}

func TestLoadRulesContent(t *testing.T) {
	// This test requires actual rules files to exist
	// Skip if not in the right environment
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		t.Skip("Cannot determine config dir")
	}

	rulesPath := filepath.Join(configDir, "rules", "agent-behavior.md")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Skip("agent-behavior.md rules file not found")
	}

	content, err := LoadRulesContent("agent-behavior.md")
	if err != nil {
		t.Fatalf("Failed to load agent-behavior.md: %v", err)
	}

	if len(content) == 0 {
		t.Error("agent-behavior.md content is empty")
	}
}

func TestLoadMultipleConventions(t *testing.T) {
	// This test requires actual convention files to exist
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		t.Skip("Cannot determine config dir")
	}

	goConventionPath := filepath.Join(configDir, "conventions", "go.md")
	if _, err := os.Stat(goConventionPath); os.IsNotExist(err) {
		t.Skip("go.md convention file not found")
	}

	// Clear cache first
	ClearConventionCache()

	// Test with mix of valid and invalid files
	conventions := []string{"go.md", "nonexistent.md"}
	results, errors := LoadMultipleConventions(conventions)

	if len(results) != 1 {
		t.Errorf("Expected 1 successful load, got %d", len(results))
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if _, ok := results["go.md"]; !ok {
		t.Error("Expected go.md in results")
	}
}

func TestGetClaudeConfigDir(t *testing.T) {
	// Test with env var
	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	defer os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)

	os.Setenv("CLAUDE_CONFIG_DIR", "/custom/path")
	dir, err := GetClaudeConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	if dir != "/custom/path" {
		t.Errorf("Expected /custom/path, got %s", dir)
	}

	// Test without env var
	os.Unsetenv("CLAUDE_CONFIG_DIR")
	dir, err = GetClaudeConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".claude")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}
