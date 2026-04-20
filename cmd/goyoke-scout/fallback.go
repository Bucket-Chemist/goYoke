package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// generateSyntheticReport creates a degraded report when all backends fail.
// Provides conservative estimates to avoid blocking router decisions.
func generateSyntheticReport(target string, primaryErr, fallbackErr error) *ScoutReport {
	// Quick file count - absolute minimum
	fileCount := 0
	_ = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if _, ok := SupportedExtensions[filepath.Ext(path)]; ok {
				fileCount++
			}
		}
		return nil
	})

	// Conservative estimates based on file count
	estimatedLines := fileCount * 100   // Assume 100 lines per file
	estimatedTokens := estimatedLines * 10 // 10 tokens per line

	warnings := []string{
		fmt.Sprintf("Primary backend failed: %v", primaryErr),
	}
	if fallbackErr != nil {
		warnings = append(warnings, fmt.Sprintf("Fallback backend failed: %v", fallbackErr))
	}
	warnings = append(warnings, "Using synthetic metrics - review recommended tier manually")

	return &ScoutReport{
		SchemaVersion: "1.0",
		Backend:       "synthetic_fallback",
		Target:        target,
		Timestamp:     time.Now().Format(time.RFC3339),
		ScopeMetrics: &ScopeMetrics{
			TotalFiles:      fileCount,
			TotalLines:      estimatedLines,
			EstimatedTokens: estimatedTokens,
			Languages:       []string{"unknown"},
			FileTypes:       map[string]int{},
		},
		ComplexitySignals: &ComplexitySignals{
			Available: false,
			Note:      "Scout backends unavailable",
		},
		RoutingRecommendation: &RoutingRecommendation{
			RecommendedTier: "sonnet", // Safe default
			Confidence:      "low",
			Reasoning:       "Scout failure - using conservative sonnet tier",
		},
		KeyFiles: []KeyFile{},
		Warnings: warnings,
	}
}

// quickFileCount performs a fast file count without line counting.
// Used in emergency fallback scenarios.
func quickFileCount(target string) int {
	count := 0
	_ = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := SupportedExtensions[filepath.Ext(path)]; ok {
			count++
		}
		return nil
	})
	return count
}
