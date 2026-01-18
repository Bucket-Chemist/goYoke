package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadScoutMetrics_Valid(t *testing.T) {
	// Create temp project dir
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write valid metrics
	metrics := ScoutMetrics{
		FileCount:       42,
		TotalLines:      3500,
		ComplexityScore: 38.5,
		RecommendedTier: "sonnet",
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Load metrics
	loaded, err := LoadScoutMetrics(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load metrics: %v", err)
	}

	if loaded.FileCount != 42 {
		t.Errorf("Expected file_count 42, got: %d", loaded.FileCount)
	}

	if loaded.RecommendedTier != "sonnet" {
		t.Errorf("Expected recommended_tier sonnet, got: %s", loaded.RecommendedTier)
	}
}

func TestLoadScoutMetrics_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Load when file doesn't exist (should return nil, not error)
	metrics, err := LoadScoutMetrics(tmpDir)
	if err != nil {
		t.Errorf("Expected no error when file missing, got: %v", err)
	}

	if metrics != nil {
		t.Error("Expected nil metrics when file missing")
	}
}

func TestLoadScoutMetrics_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write invalid JSON
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, []byte("{invalid json}"), 0644)

	// Load should fail
	_, err := LoadScoutMetrics(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadScoutMetrics_InvalidTier(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write metrics with invalid tier
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: 15.0,
		RecommendedTier: "super-mega-tier", // INVALID
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Load should fail with tier validation error
	_, err := LoadScoutMetrics(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid recommended_tier")
	}

	// Verify error message mentions valid tiers
	if err != nil {
		errMsg := err.Error()
		expectedSubstrings := []string{"Invalid recommended_tier", "super-mega-tier", "haiku", "sonnet", "opus"}
		for _, substr := range expectedSubstrings {
			if !containsString(errMsg, substr) {
				t.Errorf("Error message missing %q: %s", substr, errMsg)
			}
		}
	}
}

func TestLoadScoutMetrics_MissingRecommendedTier(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write metrics without recommended_tier
	metrics := map[string]any{
		"file_count":       10,
		"total_lines":      500,
		"complexity_score": 15.0,
		"timestamp":        time.Now().Unix(),
		// recommended_tier deliberately omitted
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Load should fail
	_, err := LoadScoutMetrics(tmpDir)
	if err == nil {
		t.Error("Expected error for missing recommended_tier")
	}
}

func TestLoadScoutMetrics_NegativeComplexity(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write metrics with negative complexity
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: -5.0, // INVALID
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Load should fail
	_, err := LoadScoutMetrics(tmpDir)
	if err == nil {
		t.Error("Expected error for negative complexity_score")
	}
}

func TestLoadScoutMetrics_AllValidTiers(t *testing.T) {
	validTiers := []string{"haiku", "haiku_thinking", "sonnet", "opus", "external"}

	for _, tier := range validTiers {
		t.Run(tier, func(t *testing.T) {
			tmpDir := t.TempDir()
			metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
			os.MkdirAll(metricsDir, 0755)

			metrics := ScoutMetrics{
				FileCount:       10,
				TotalLines:      500,
				ComplexityScore: 15.0,
				RecommendedTier: tier,
				Timestamp:       time.Now().Unix(),
			}

			data, _ := json.Marshal(metrics)
			metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
			os.WriteFile(metricsPath, data, 0644)

			loaded, err := LoadScoutMetrics(tmpDir)
			if err != nil {
				t.Fatalf("Valid tier %q should not error: %v", tier, err)
			}

			if loaded.RecommendedTier != tier {
				t.Errorf("Expected tier %q, got %q", tier, loaded.RecommendedTier)
			}
		})
	}
}

func TestLoadScoutMetrics_FutureExpansionFields(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write metrics with future expansion fields
	metricsJSON := `{
		"file_count": 42,
		"total_lines": 3500,
		"complexity_score": 38.5,
		"recommended_tier": "sonnet",
		"timestamp": ` + jsonInt64(time.Now().Unix()) + `,
		"estimated_tokens": 25000,
		"confidence": 0.85,
		"clarification_needed": false,
		"import_density": 0.12,
		"cross_file_dependencies": 8
	}`

	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, []byte(metricsJSON), 0644)

	loaded, err := LoadScoutMetrics(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load metrics with future fields: %v", err)
	}

	// Verify base fields
	if loaded.FileCount != 42 {
		t.Errorf("Expected file_count 42, got: %d", loaded.FileCount)
	}

	// Verify future expansion fields
	if loaded.EstimatedTokens != 25000 {
		t.Errorf("Expected estimated_tokens 25000, got: %d", loaded.EstimatedTokens)
	}
	if loaded.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got: %f", loaded.Confidence)
	}
	if loaded.ClarificationNeeded != false {
		t.Errorf("Expected clarification_needed false, got: %v", loaded.ClarificationNeeded)
	}
	if loaded.ImportDensity != 0.12 {
		t.Errorf("Expected import_density 0.12, got: %f", loaded.ImportDensity)
	}
	if loaded.CrossFileDependencies != 8 {
		t.Errorf("Expected cross_file_dependencies 8, got: %d", loaded.CrossFileDependencies)
	}
}

func TestIsFresh(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name       string
		timestamp  int64
		ttlSeconds int
		expected   bool
	}{
		{"Fresh (1 min old, 5 min TTL)", now - 60, 300, true},
		{"Stale (6 min old, 5 min TTL)", now - 360, 300, false},
		{"Exactly at TTL", now - 300, 300, false},
		{"Zero TTL", now, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ScoutMetrics{Timestamp: tt.timestamp}
			if fresh := metrics.IsFresh(tt.ttlSeconds); fresh != tt.expected {
				t.Errorf("IsFresh() = %v, expected %v", fresh, tt.expected)
			}
		})
	}
}

func TestIsFresh_Nil(t *testing.T) {
	var metrics *ScoutMetrics
	if metrics.IsFresh(300) {
		t.Error("Nil metrics should not be fresh")
	}
}

func TestAge(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name      string
		timestamp int64
		minAge    int64
		maxAge    int64
	}{
		{"1 minute old", now - 60, 59, 61},
		{"5 minutes old", now - 300, 299, 301},
		{"Just created", now, -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ScoutMetrics{Timestamp: tt.timestamp}
			age := metrics.Age()
			if age < tt.minAge || age > tt.maxAge {
				t.Errorf("Age() = %d, expected between %d and %d", age, tt.minAge, tt.maxAge)
			}
		})
	}
}

func TestAge_Nil(t *testing.T) {
	var metrics *ScoutMetrics
	age := metrics.Age()
	if age != -1 {
		t.Errorf("Nil metrics Age() should return -1, got: %d", age)
	}
}

func TestGetActiveTier_Fresh(t *testing.T) {
	now := time.Now().Unix()
	metrics := &ScoutMetrics{
		RecommendedTier: "haiku",
		Timestamp:       now - 60, // 1 minute old
	}

	config := &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}

	tier := metrics.GetActiveTier(config)
	if tier != "haiku" {
		t.Errorf("Expected fresh tier 'haiku', got: %s", tier)
	}
}

func TestGetActiveTier_Stale(t *testing.T) {
	now := time.Now().Unix()
	metrics := &ScoutMetrics{
		RecommendedTier: "haiku",
		Timestamp:       now - 400, // 6.7 minutes old
	}

	config := &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}

	tier := metrics.GetActiveTier(config)
	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet', got: %s", tier)
	}
}

func TestGetActiveTier_Nil(t *testing.T) {
	var metrics *ScoutMetrics
	config := DefaultMetricsConfig()

	tier := metrics.GetActiveTier(config)
	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet' for nil metrics, got: %s", tier)
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	if config.TTLSeconds != 300 {
		t.Errorf("Expected default TTL 300s, got: %d", config.TTLSeconds)
	}

	if config.FallbackTier != "sonnet" {
		t.Errorf("Expected default fallback 'sonnet', got: %s", config.FallbackTier)
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper to convert int64 to JSON string representation
func jsonInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}
