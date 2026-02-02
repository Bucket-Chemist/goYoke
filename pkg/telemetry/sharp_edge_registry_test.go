package telemetry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// resetRegistry resets the singleton for test isolation
func resetRegistry() {
	sharpEdgeIDs = nil
	sharpEdgeIDsOnce = sync.Once{}
}

func TestIsValidSharpEdgeID(t *testing.T) {
	// Setup test directory with mock sharp-edges.yaml
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)
	resetRegistry()

	// Create test agent directory with sharp-edges.yaml
	agentDir := filepath.Join(dir, ".claude", "agents", "test-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	yaml := `sharp_edges:
  - id: sql-injection
    severity: critical
  - id: xss-vulnerability
    severity: high
  - id: missing-auth
    severity: critical
`
	if err := os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("Failed to write sharp-edges.yaml: %v", err)
	}

	// Force registry reload
	if err := LoadSharpEdgeIDs(); err != nil {
		t.Fatalf("LoadSharpEdgeIDs failed: %v", err)
	}

	// Test valid IDs
	validIDs := []string{"sql-injection", "xss-vulnerability", "missing-auth"}
	for _, id := range validIDs {
		if !IsValidSharpEdgeID(id) {
			t.Errorf("Expected %q to be valid", id)
		}
	}

	// Test invalid ID
	if IsValidSharpEdgeID("nonexistent-id") {
		t.Error("Expected 'nonexistent-id' to be invalid")
	}
}

func TestGetAllSharpEdgeIDs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)
	resetRegistry()

	// Create multiple agent directories
	agents := []struct {
		name string
		ids  []string
	}{
		{"backend-reviewer", []string{"sql-injection", "auth-bypass"}},
		{"frontend-reviewer", []string{"xss-vulnerability", "csrf-missing"}},
	}

	for _, agent := range agents {
		agentDir := filepath.Join(dir, ".claude", "agents", agent.name)
		os.MkdirAll(agentDir, 0755)

		yaml := "sharp_edges:\n"
		for _, id := range agent.ids {
			yaml += "  - id: " + id + "\n"
		}
		os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte(yaml), 0644)
	}

	LoadSharpEdgeIDs()
	allIDs := GetAllSharpEdgeIDs()

	if len(allIDs) != 4 {
		t.Errorf("Expected 4 IDs, got %d: %v", len(allIDs), allIDs)
	}
}

func TestLoadSharpEdgeIDs_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)
	resetRegistry()

	// Don't create the agents directory
	err := LoadSharpEdgeIDs()
	if err == nil {
		// Error is expected when agents dir doesn't exist
		// But the function should handle it gracefully
	}

	// Should return false for any ID when registry failed to load
	if IsValidSharpEdgeID("any-id") {
		t.Error("Expected all IDs to be invalid when registry failed to load")
	}
}

func TestLoadSharpEdgeIDs_EmptyYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)
	resetRegistry()

	agentDir := filepath.Join(dir, ".claude", "agents", "empty-agent")
	os.MkdirAll(agentDir, 0755)

	// Empty sharp_edges list
	os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte("sharp_edges: []\n"), 0644)

	LoadSharpEdgeIDs()

	if IsValidSharpEdgeID("anything") {
		t.Error("Expected no valid IDs from empty YAML")
	}
}
