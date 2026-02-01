package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSharpEdgeHit_ValidID(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Create test sharp-edges.yaml
	agentDir := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".claude", "agents", "test-agent")
	os.MkdirAll(agentDir, 0755)
	yaml := `sharp_edges:
  - id: sql-injection
  - id: xss-vulnerability
`
	os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte(yaml), 0644)
	ResetRegistry() // Reset to force reload

	hit, err := NewSharpEdgeHit("session1", "sql-injection", "agent1", "reviewer1", "finding1", "file.go", 10)
	if err != nil {
		t.Fatalf("NewSharpEdgeHit failed: %v", err)
	}
	if hit.SharpEdgeID != "sql-injection" {
		t.Errorf("Expected sharp_edge_id 'sql-injection', got '%s'", hit.SharpEdgeID)
	}
}

func TestNewSharpEdgeHit_InvalidID(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()
	ResetRegistry()

	_, err := NewSharpEdgeHit("session1", "nonexistent-id", "agent1", "reviewer1", "finding1", "file.go", 10)
	if err == nil {
		t.Error("Expected error for invalid sharp_edge_id")
	}
}

func TestLogSharpEdgeHit(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Create test sharp-edges.yaml
	agentDir := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".claude", "agents", "test-agent")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte("sharp_edges:\n  - id: test-edge\n"), 0644)
	ResetRegistry()

	hit, _ := NewSharpEdgeHit("session1", "test-edge", "agent1", "reviewer1", "finding1", "file.go", 10)
	err := LogSharpEdgeHit(hit)
	if err != nil {
		t.Fatalf("LogSharpEdgeHit failed: %v", err)
	}

	hits, err := ReadSharpEdgeHits()
	if err != nil {
		t.Fatalf("ReadSharpEdgeHits failed: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("Expected 1 hit, got %d", len(hits))
	}
}
