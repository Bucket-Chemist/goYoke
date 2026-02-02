package main

import (
	"fmt"
	"os"
	"strconv"
)

// RoutingScore represents factors used for backend selection.
type RoutingScore struct {
	FileCount     int
	TotalLines    int
	MaxFileLines  int
	LanguageCount int
}

// Score calculates a weighted composite score for routing decisions.
// Score range: 0-100
// - Below 40: Native scout (fast, basic metrics)
// - 40+: Gemini scout (semantic analysis)
func (rs RoutingScore) Score() int {
	// Normalize each factor to 0-100 range, then apply weights
	fileScore := min(rs.FileCount*5, 100)        // 20 files = 100
	lineScore := min(rs.TotalLines/50, 100)      // 5000 lines = 100
	maxFileScore := min(rs.MaxFileLines/10, 100) // 1000 line file = 100
	langScore := min(rs.LanguageCount*25, 100)   // 4 languages = 100

	// Weighted composite: file count (40%), total lines (30%), max file (20%), languages (10%)
	return (fileScore*40 + lineScore*30 + maxFileScore*20 + langScore*10) / 100
}

const (
	// NativeThreshold is the score below which native scout is used.
	NativeThreshold = 40 // Score 40 ≈ 8 files, 2000 lines, max 400 lines, 1 language
)

// selectBackend determines which backend to use based on target scope.
func selectBackend(target string) (string, error) {
	// 1. Check environment override
	if backend := os.Getenv("SCOUT_BACKEND"); backend != "" {
		if backend == "native" || backend == "gemini" {
			return backend, nil
		}
		return "", fmt.Errorf("invalid SCOUT_BACKEND: %s (use 'native' or 'gemini')", backend)
	}

	// 2. Calculate routing score
	score, err := calculateRoutingScore(target)
	if err != nil {
		// Can't even count files → try native anyway
		return "native", nil
	}

	// 3. Apply threshold (configurable via env)
	threshold := NativeThreshold
	if t := os.Getenv("SCOUT_THRESHOLD"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			threshold = parsed
		}
	}

	if score.Score() < threshold {
		return "native", nil
	}
	return "gemini", nil
}

// calculateRoutingScore performs a quick pass to compute routing factors.
func calculateRoutingScore(target string) (RoutingScore, error) {
	// Do a quick walk to gather basic stats
	var files []FileInfo
	err := walkTarget(target, func(fi FileInfo) {
		files = append(files, fi)
	})

	if err != nil {
		return RoutingScore{}, err
	}

	if len(files) == 0 {
		return RoutingScore{}, fmt.Errorf("no supported files found")
	}

	// Aggregate for scoring
	langSet := make(map[string]bool)
	var totalLines, maxLines int

	for _, f := range files {
		totalLines += f.Lines
		if f.Lines > maxLines {
			maxLines = f.Lines
		}
		langSet[f.Language] = true
	}

	return RoutingScore{
		FileCount:     len(files),
		TotalLines:    totalLines,
		MaxFileLines:  maxLines,
		LanguageCount: len(langSet),
	}, nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
