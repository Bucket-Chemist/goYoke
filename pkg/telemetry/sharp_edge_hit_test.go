package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

func setupSharpEdgeHitTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)

	// Create minimal structure
	os.MkdirAll(filepath.Join(dir, ".gogent"), 0755)
	os.MkdirAll(filepath.Join(dir, ".gogent", "memory"), 0755)

	// Create test agent with sharp-edges.yaml for registry
	agentDir := filepath.Join(dir, ".claude", "agents", "test-reviewer")
	os.MkdirAll(agentDir, 0755)
	yaml := `sharp_edges:
  - id: sql-injection
    severity: critical
  - id: xss-vulnerability
    severity: high
`
	os.WriteFile(filepath.Join(agentDir, "sharp-edges.yaml"), []byte(yaml), 0644)

	// Reset the registry to pick up new test data
	ResetRegistry()

	return func() { /* TempDir auto-cleans */ }
}

func TestNewSharpEdgeHit_Valid(t *testing.T) {
	cleanup := setupSharpEdgeHitTestDir(t)
	defer cleanup()

	hit, err := NewSharpEdgeHit("session1", "sql-injection", "test-reviewer", "backend-reviewer", "finding-123", "file.go", 42)
	if err != nil {
		t.Fatalf("NewSharpEdgeHit failed: %v", err)
	}

	if hit.HitID == "" {
		t.Error("HitID should not be empty")
	}
	if hit.SharpEdgeID != "sql-injection" {
		t.Errorf("Expected SharpEdgeID 'sql-injection', got %q", hit.SharpEdgeID)
	}
	if hit.MatchConfidence != 1.0 {
		t.Errorf("Expected MatchConfidence 1.0, got %f", hit.MatchConfidence)
	}
	if hit.Timestamp == 0 {
		t.Error("Timestamp should not be 0")
	}
}

func TestNewSharpEdgeHit_Invalid(t *testing.T) {
	cleanup := setupSharpEdgeHitTestDir(t)
	defer cleanup()

	_, err := NewSharpEdgeHit("session1", "nonexistent-id", "agent", "reviewer", "finding", "file.go", 1)
	if err == nil {
		t.Error("Expected error for invalid sharp edge ID")
	}
	if !strings.Contains(err.Error(), "invalid sharp_edge_id") {
		t.Errorf("Error message should mention invalid ID: %v", err)
	}
}

func TestLogSharpEdgeHit(t *testing.T) {
	cleanup := setupSharpEdgeHitTestDir(t)
	defer cleanup()

	hit, err := NewSharpEdgeHit("session1", "sql-injection", "test-reviewer", "backend-reviewer", "finding-123", "file.go", 42)
	if err != nil {
		t.Fatalf("NewSharpEdgeHit failed: %v", err)
	}

	err = LogSharpEdgeHit(hit)
	if err != nil {
		t.Fatalf("LogSharpEdgeHit failed: %v", err)
	}

	// Verify file written
	path := config.GetSharpEdgeHitsPathWithProjectDir()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), hit.HitID) {
		t.Error("File should contain hit ID")
	}
	if !strings.Contains(string(content), "sql-injection") {
		t.Error("File should contain sharp edge ID")
	}
}

func TestLogSharpEdgeHit_Concurrent(t *testing.T) {
	cleanup := setupSharpEdgeHitTestDir(t)
	defer cleanup()

	// Pre-load registry before concurrent operations
	if err := LoadSharpEdgeIDs(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hit, err := NewSharpEdgeHit("session1", "sql-injection", "agent", "reviewer", fmt.Sprintf("finding-%d", n), "file.go", n)
			if err != nil {
				errors <- err
				return
			}
			if err := LogSharpEdgeHit(hit); err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent write failed: %v", err)
	}

	// Verify all 20 writes succeeded
	path := config.GetSharpEdgeHitsPathWithProjectDir()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 20 {
		t.Errorf("Expected 20 lines, got %d", len(lines))
	}
}