package routing

import (
	"fmt"
	"os"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// UpdateTierFromMetrics reads scout metrics and updates current-tier file
func UpdateTierFromMetrics(projectDir string) error {
	// Load scout metrics
	metrics, err := LoadScoutMetrics(projectDir)
	if err != nil {
		return fmt.Errorf("[tier-update] Failed to load metrics: %w", err)
	}

	// If no metrics, nothing to update
	if metrics == nil {
		return nil
	}

	// Get active tier based on freshness
	metricsConfig := DefaultMetricsConfig()
	activeTier := metrics.GetActiveTier(metricsConfig)

	// Write to current-tier file
	tierPath := config.GetTierFilePath()
	if err := os.WriteFile(tierPath, []byte(activeTier), 0644); err != nil {
		return fmt.Errorf("[tier-update] Failed to write tier file at %s: %w. Check permissions.", tierPath, err)
	}

	return nil
}

// GetCurrentTier reads current-tier file, returns default if missing
func GetCurrentTier() (string, error) {
	tierPath := config.GetTierFilePath()

	data, err := os.ReadFile(tierPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "sonnet", nil // Default tier
		}
		return "", fmt.Errorf("[tier-read] Failed to read tier file at %s: %w", tierPath, err)
	}

	tier := string(data)
	if tier == "" {
		return "sonnet", nil
	}

	return tier, nil
}
