package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ScoutMetrics represents output from haiku-scout agent.
// Contains file counts, LoC, complexity signals for routing decisions.
type ScoutMetrics struct {
	// Base fields (required by goYoke-013)
	FileCount       int      `json:"file_count"`
	TotalLines      int      `json:"total_lines"`
	ComplexityScore float64  `json:"complexity_score"`
	RecommendedTier string   `json:"recommended_tier"`
	Timestamp       int64    `json:"timestamp"`
	ScannedPaths    []string `json:"scanned_paths,omitempty"`

	// Future expansion fields (goYoke-014+)
	EstimatedTokens         int     `json:"estimated_tokens,omitempty"`
	Confidence              float64 `json:"confidence,omitempty"`
	ClarificationNeeded     bool    `json:"clarification_needed,omitempty"`
	ImportDensity           float64 `json:"import_density,omitempty"`
	CrossFileDependencies   int     `json:"cross_file_dependencies,omitempty"`
}

// LoadScoutMetrics reads scout_metrics.json from project tmp directory.
// Returns nil (no error) when file doesn't exist (scout hasn't run yet).
// Validates required fields and tier name before returning.
func LoadScoutMetrics(projectDir string) (*ScoutMetrics, error) {
	metricsPath := filepath.Join(projectDir, ".claude", "tmp", "scout_metrics.json")

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No metrics file is OK - scout hasn't run
		}
		return nil, fmt.Errorf("[metrics] Failed to read scout metrics at %s: %w. Check file permissions.", metricsPath, err)
	}

	var metrics ScoutMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("[metrics] Failed to parse scout metrics: %w. Check JSON format in %s", err, metricsPath)
	}

	// Validate required fields
	if metrics.ComplexityScore < 0 {
		return nil, fmt.Errorf("[metrics] Invalid complexity_score: %f. Must be >= 0.", metrics.ComplexityScore)
	}

	if metrics.RecommendedTier == "" {
		return nil, fmt.Errorf("[metrics] Missing required field: recommended_tier. Scout must specify tier.")
	}

	// Validate tier name (prevents downstream routing errors)
	validTiers := map[string]bool{
		"haiku":          true,
		"haiku_thinking": true,
		"sonnet":         true,
		"opus":           true,
		"external":       true,
	}
	if !validTiers[metrics.RecommendedTier] {
		return nil, fmt.Errorf("[metrics] Invalid recommended_tier: %q. Must be one of: haiku, haiku_thinking, sonnet, opus, external", metrics.RecommendedTier)
	}

	return &metrics, nil
}

// IsFresh returns true if metrics are less than ttlSeconds old.
// Returns false if metrics are nil (nil safety).
func (m *ScoutMetrics) IsFresh(ttlSeconds int) bool {
	if m == nil {
		return false
	}
	age := time.Now().Unix() - m.Timestamp
	return age < int64(ttlSeconds)
}

// Age returns metrics age in seconds.
// Returns -1 if metrics are nil (nil safety).
func (m *ScoutMetrics) Age() int64 {
	if m == nil {
		return -1
	}
	return time.Now().Unix() - m.Timestamp
}

// MetricsConfig defines TTL and fallback behavior
type MetricsConfig struct {
	TTLSeconds   int    // Default: 300 (5 minutes)
	FallbackTier string // Default: "sonnet"
}

// DefaultMetricsConfig returns standard config
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}
}

// GetActiveTier returns tier based on metrics freshness
// If fresh: returns recommended_tier from metrics
// If stale: returns fallback tier
func (m *ScoutMetrics) GetActiveTier(config *MetricsConfig) string {
	if m == nil {
		return config.FallbackTier
	}

	if m.IsFresh(config.TTLSeconds) {
		return m.RecommendedTier
	}

	return config.FallbackTier
}
