package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidTiers defines the allowed tier values from routing schema.
var ValidTiers = []string{
	"haiku",
	"haiku_thinking",
	"sonnet",
	"opus",
}

// GetCurrentTier reads the current tier from ~/.gogent/current-tier.
// Returns "sonnet" as fallback if file doesn't exist or is empty.
// Validates tier value against routing schema.
//
// Error format: "[tier] What. Why. How to fix."
func GetCurrentTier() (string, error) {
	return GetCurrentTierFromPath(GetTierFilePath())
}

// GetCurrentTierFromPath reads tier from specified path (extracted for testing).
// Returns "sonnet" as fallback if file doesn't exist or is empty.
// Validates tier value against routing schema.
//
// This function is exported to allow tests to inject custom file paths
// while GetCurrentTier() provides the production path via GetTierFilePath().
func GetCurrentTierFromPath(path string) (string, error) {
	// Read file contents
	content, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist → return sonnet fallback
		if os.IsNotExist(err) {
			return "sonnet", nil
		}
		return "", fmt.Errorf("[tier] Failed to read current-tier file. Filesystem error. Check permissions on %s: %w", path, err)
	}

	// Trim whitespace
	tier := strings.TrimSpace(string(content))

	// Empty file → return sonnet fallback
	if tier == "" {
		return "sonnet", nil
	}

	// Validate tier value
	if !isValidTier(tier) {
		return "", fmt.Errorf("[tier] Invalid tier value '%s'. Must be one of: %v. Fix: Write valid tier to %s", tier, ValidTiers, path)
	}

	return tier, nil
}

// isValidTier checks if tier value is in ValidTiers list.
func isValidTier(tier string) bool {
	for _, valid := range ValidTiers {
		if tier == valid {
			return true
		}
	}
	return false
}
