package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSharpEdgeIDs(t *testing.T) {
	// Setup: Create temporary test directory structure
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".claude", "agents")

	// Create test agent directories with sharp-edges.yaml
	agents := []struct {
		name string
		yaml string
	}{
		{
			name: "agent1",
			yaml: `sharp_edges:
  - id: "SE-001"
    title: "Test Edge 1"
  - id: "SE-002"
    title: "Test Edge 2"
`,
		},
		{
			name: "agent2",
			yaml: `sharp_edges:
  - id: "SE-003"
    title: "Test Edge 3"
`,
		},
		{
			name: "agent3-no-edges",
			yaml: `sharp_edges: []
`,
		},
	}

	for _, agent := range agents {
		agentDir := filepath.Join(agentsDir, agent.name)
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			t.Fatalf("Failed to create agent dir: %v", err)
		}
		yamlPath := filepath.Join(agentDir, "sharp-edges.yaml")
		if err := os.WriteFile(yamlPath, []byte(agent.yaml), 0644); err != nil {
			t.Fatalf("Failed to write yaml: %v", err)
		}
	}

	// Set environment to use test directory
	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	// Reset registry to ensure clean state
	ResetRegistry()

	// Test LoadSharpEdgeIDs
	err := LoadSharpEdgeIDs()
	if err != nil {
		t.Fatalf("LoadSharpEdgeIDs failed: %v", err)
	}

	// Verify all IDs are loaded
	expectedIDs := map[string]bool{
		"SE-001": true,
		"SE-002": true,
		"SE-003": true,
	}

	for id := range expectedIDs {
		if !IsValidSharpEdgeID(id) {
			t.Errorf("Expected ID %s to be valid", id)
		}
	}

	// Verify invalid ID returns false
	if IsValidSharpEdgeID("SE-999") {
		t.Error("Expected SE-999 to be invalid")
	}

	// Verify GetAllSharpEdgeIDs
	allIDs := GetAllSharpEdgeIDs()
	if len(allIDs) != len(expectedIDs) {
		t.Errorf("Expected %d IDs, got %d", len(expectedIDs), len(allIDs))
	}
}

func TestIsValidSharpEdgeID_LazyLoad(t *testing.T) {
	// Reset registry
	ResetRegistry()

	// Setup test directory
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".claude", "agents", "test-agent")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	yamlContent := `sharp_edges:
  - id: "LAZY-001"
`
	yamlPath := filepath.Join(agentsDir, "sharp-edges.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write yaml: %v", err)
	}

	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	// IsValidSharpEdgeID should trigger lazy load
	if !IsValidSharpEdgeID("LAZY-001") {
		t.Error("Lazy load failed, ID should be valid")
	}
}

func TestLoadSharpEdgeIDs_MissingDirectory(t *testing.T) {
	// Reset registry
	ResetRegistry()

	// Point to non-existent directory
	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", "/tmp/nonexistent-gogent-test-dir")
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	// Should handle gracefully
	err := LoadSharpEdgeIDs()
	if err == nil {
		t.Error("Expected error for missing directory")
	}
}

func TestLoadSharpEdgeIDs_InvalidYAML(t *testing.T) {
	// Reset registry
	ResetRegistry()

	// Setup directory with invalid YAML
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".claude", "agents", "bad-agent")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	invalidYAML := `this is not: valid: yaml: content`
	yamlPath := filepath.Join(agentsDir, "sharp-edges.yaml")
	if err := os.WriteFile(yamlPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write yaml: %v", err)
	}

	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	// Should handle gracefully (skip invalid files)
	err := LoadSharpEdgeIDs()
	if err != nil {
		t.Errorf("Should not error on invalid YAML: %v", err)
	}
}

func TestResetRegistry(t *testing.T) {
	// Reset first to ensure clean state
	ResetRegistry()

	// Load some data
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".claude", "agents", "test")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "sharp-edges.yaml"), []byte(`sharp_edges:
  - id: "RESET-001"
`), 0644); err != nil {
		t.Fatalf("Failed to write yaml: %v", err)
	}

	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	err := LoadSharpEdgeIDs()
	if err != nil {
		t.Fatalf("LoadSharpEdgeIDs failed: %v", err)
	}

	if !IsValidSharpEdgeID("RESET-001") {
		t.Fatal("Setup failed, ID should be valid")
	}

	// Reset
	ResetRegistry()

	// After reset, registry should be nil (will reload on next access)
	if sharpEdgeIDs != nil {
		t.Error("Registry should be nil after reset")
	}

	// Verify it can be reloaded
	if !IsValidSharpEdgeID("RESET-001") {
		t.Error("After reset, lazy reload should restore RESET-001")
	}
}

func TestGetAllSharpEdgeIDs(t *testing.T) {
	ResetRegistry()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".claude", "agents", "test")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "sharp-edges.yaml"), []byte(`sharp_edges:
  - id: "GET-001"
  - id: "GET-002"
  - id: "GET-003"
`), 0644)

	oldProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldProjectDir)

	allIDs := GetAllSharpEdgeIDs()
	if len(allIDs) != 3 {
		t.Errorf("Expected 3 IDs, got %d", len(allIDs))
	}

	// Verify all expected IDs are present
	expectedIDs := map[string]bool{
		"GET-001": false,
		"GET-002": false,
		"GET-003": false,
	}

	for _, id := range allIDs {
		if _, exists := expectedIDs[id]; exists {
			expectedIDs[id] = true
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected ID %s not found in result", id)
		}
	}
}
