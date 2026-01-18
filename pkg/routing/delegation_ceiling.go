package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DelegationCeilingRuntime represents max allowed delegation tier
type DelegationCeilingRuntime struct {
	MaxTier string // e.g., "haiku", "sonnet"
}

// LoadDelegationCeiling reads max_delegation file from project
func LoadDelegationCeiling(projectDir string) (*DelegationCeilingRuntime, error) {
	ceilingPath := filepath.Join(projectDir, ".claude", "tmp", "max_delegation")

	data, err := os.ReadFile(ceilingPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No ceiling file = no restriction (default: sonnet)
			return &DelegationCeilingRuntime{MaxTier: "sonnet"}, nil
		}
		return nil, fmt.Errorf("[delegation] Failed to read ceiling file at %s: %w", ceilingPath, err)
	}

	maxTier := strings.TrimSpace(string(data))
	if maxTier == "" {
		maxTier = "sonnet" // Default
	}

	return &DelegationCeilingRuntime{MaxTier: maxTier}, nil
}

// CheckDelegationCeiling validates if requested model is within ceiling
func CheckDelegationCeiling(schema *Schema, ceiling *DelegationCeilingRuntime, requestedModel string) (bool, string) {
	// Get tier level for ceiling
	ceilingLevel, err := schema.GetTierLevel(ceiling.MaxTier)
	if err != nil {
		// Unknown ceiling tier, allow (permissive fallback)
		return true, ""
	}

	// Get tier level for requested model
	requestedLevel, err := schema.GetTierLevel(requestedModel)
	if err != nil {
		// Unknown requested tier, allow
		return true, ""
	}

	if requestedLevel > ceilingLevel {
		return false, fmt.Sprintf(
			"[delegation] Requested model '%s' (level %d) exceeds delegation ceiling '%s' (level %d). Complexity analysis determined max tier. Use --force-delegation=%s to override.",
			requestedModel,
			requestedLevel,
			ceiling.MaxTier,
			ceilingLevel,
			requestedModel,
		)
	}

	return true, ""
}
