package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateTierFromMetrics_Fresh(t *testing.T) {
	// Setup temp directories
	tmpProject := t.TempDir()
	tmpgoYoke := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		t.Fatalf("Failed to create metrics dir: %v", err)
	}

	// Override XDG_RUNTIME_DIR for testing
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// Write fresh metrics recommending "haiku"
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: 12.5,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	// Update tier
	if err := UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("UpdateTierFromMetrics failed: %v", err)
	}

	// Verify tier file
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("GetCurrentTier failed: %v", err)
	}

	if tier != "haiku" {
		t.Errorf("Expected tier 'haiku', got: %s", tier)
	}
}

func TestUpdateTierFromMetrics_Stale(t *testing.T) {
	tmpProject := t.TempDir()
	tmpgoYoke := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		t.Fatalf("Failed to create metrics dir: %v", err)
	}

	// Override XDG_RUNTIME_DIR for testing
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// Write stale metrics (7 minutes old)
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: 12.5,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix() - 420, // 7 minutes
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	// Update tier
	if err := UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("UpdateTierFromMetrics failed: %v", err)
	}

	// Should fall back to "sonnet"
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("GetCurrentTier failed: %v", err)
	}

	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet', got: %s", tier)
	}
}

func TestGetCurrentTier_NoFile(t *testing.T) {
	tmpgoYoke := t.TempDir()

	// Override XDG_RUNTIME_DIR for testing
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// No tier file exists
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tier != "sonnet" {
		t.Errorf("Expected default tier 'sonnet', got: %s", tier)
	}
}

func TestUpdateTierFromMetrics_NoMetrics(t *testing.T) {
	tmpProject := t.TempDir()
	tmpgoYoke := t.TempDir()

	// Override XDG_RUNTIME_DIR for testing
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// No metrics file exists
	err := UpdateTierFromMetrics(tmpProject)
	if err != nil {
		t.Errorf("Expected nil error when no metrics, got: %v", err)
	}
}

func TestGetCurrentTier_EmptyFile(t *testing.T) {
	tmpgoYoke := t.TempDir()

	// Override XDG_RUNTIME_DIR for testing
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpgoYoke)

	// Create goYoke directory
	goyokeDir := filepath.Join(tmpgoYoke, "goyoke")
	if err := os.MkdirAll(goyokeDir, 0755); err != nil {
		t.Fatalf("Failed to create goyoke dir: %v", err)
	}

	// Create empty tier file
	tierPath := filepath.Join(goyokeDir, "current-tier")
	if err := os.WriteFile(tierPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty tier file: %v", err)
	}

	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tier != "sonnet" {
		t.Errorf("Expected default tier 'sonnet' for empty file, got: %s", tier)
	}
}
