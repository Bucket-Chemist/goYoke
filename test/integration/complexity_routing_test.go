package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

func TestComplexityRoutingWorkflow(t *testing.T) {
	// Setup temp directories
	tmpProject := t.TempDir()
	tmpgoYoke := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Save and restore XDG_RUNTIME_DIR to isolate test
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// Step 1: Scout writes metrics (simulated)
	scoutMetrics := routing.ScoutMetrics{
		FileCount:       85,
		TotalLines:      7200,
		ComplexityScore: 42.8,
		RecommendedTier: "sonnet",
		Timestamp:       time.Now().Unix(),
		ScannedPaths:    []string{"src/", "pkg/"},
	}

	metricsData, _ := json.Marshal(scoutMetrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, metricsData, 0644)

	// Step 2: Load and verify metrics
	loaded, err := routing.LoadScoutMetrics(tmpProject)
	if err != nil {
		t.Fatalf("Failed to load scout metrics: %v", err)
	}

	if !loaded.IsFresh(300) {
		t.Error("Metrics should be fresh")
	}

	// Step 3: Update tier from metrics
	if err := routing.UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("Failed to update tier: %v", err)
	}

	// Step 4: Verify tier updated correctly
	currentTier, err := routing.GetCurrentTier()
	if err != nil {
		t.Fatalf("Failed to get current tier: %v", err)
	}

	if currentTier != "sonnet" {
		t.Errorf("Expected tier 'sonnet', got: %s", currentTier)
	}

	t.Logf("✓ Complexity routing workflow complete: scout → metrics → tier")
}

func TestComplexityRoutingStaleMetrics(t *testing.T) {
	tmpProject := t.TempDir()
	tmpgoYoke := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Save and restore XDG_RUNTIME_DIR to isolate test
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// Write stale metrics (10 minutes old)
	staleMetrics := routing.ScoutMetrics{
		FileCount:       5,
		TotalLines:      200,
		ComplexityScore: 8.2,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix() - 600, // 10 minutes
	}

	metricsData, _ := json.Marshal(staleMetrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, metricsData, 0644)

	// Update tier
	if err := routing.UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("Failed to update tier: %v", err)
	}

	// Should fall back to "sonnet"
	currentTier, err := routing.GetCurrentTier()
	if err != nil {
		t.Fatalf("Failed to get current tier: %v", err)
	}

	if currentTier != "sonnet" {
		t.Errorf("Expected fallback to 'sonnet' for stale metrics, got: %s", currentTier)
	}

	t.Logf("✓ Stale metrics correctly fall back to default tier")
}
