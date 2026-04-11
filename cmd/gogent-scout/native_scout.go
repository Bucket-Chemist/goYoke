package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SupportedExtensions maps file extensions to language names.
var SupportedExtensions = map[string]string{
	".go":   "go",
	".py":   "python",
	".r":    "r",
	".R":    "r",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".md":   "markdown",
	".yaml": "yaml",
	".yml":  "yaml",
	".json": "json",
	".toml": "toml",
}

// TestPatterns identifies test files by naming convention.
var TestPatterns = []string{
	"_test.go",
	"test_",
	"_test.py",
	".test.",
	"test-",
	"spec.",
}

// NativeScout performs basic metrics-only scouting without semantic analysis.
type NativeScout struct {
	Target      string
	Instruction string
	Files       []string
}

// Run executes the native scout and returns a basic scout report.
func (ns *NativeScout) Run() (*ScoutReport, error) {
	files, err := ns.collectFiles()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported files found in %s", ns.Target)
	}

	// 2. Aggregate metrics
	metrics := aggregateMetrics(files)

	// 3. Generate routing recommendation
	recommendation := ns.generateRecommendation(metrics)

	// 4. Identify key files (top 5 by size)
	keyFiles := identifyKeyFiles(files, 5)

	return &ScoutReport{
		SchemaVersion: "1.0",
		Backend:       "native",
		Target:        ns.Target,
		Timestamp:     time.Now().Format(time.RFC3339),
		ScopeMetrics:  metrics,
		ComplexitySignals: &ComplexitySignals{
			Available:           false,
			TestCoveragePresent: hasTestFiles(files),
			Note:                "Semantic analysis unavailable - basic metrics only",
		},
		RoutingRecommendation: recommendation,
		KeyFiles:              keyFiles,
		Warnings:              []string{},
	}, nil
}

func (ns *NativeScout) collectFiles() ([]FileInfo, error) {
	if len(ns.Files) > 0 {
		files := make([]FileInfo, 0, len(ns.Files))
		for _, path := range ns.Files {
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
			if info.IsDir() {
				dirFiles, err := collectFilesFromDir(path)
				if err != nil {
					return nil, err
				}
				files = append(files, dirFiles...)
				continue
			}

			fileInfo, ok, err := collectFileInfo(path)
			if err != nil {
				return nil, err
			}
			if ok {
				files = append(files, fileInfo)
			}
		}
		return files, nil
	}

	return collectFilesFromDir(ns.Target)
}

func collectFilesFromDir(target string) ([]FileInfo, error) {
	var files []FileInfo
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable paths
			return nil
		}

		if d.IsDir() {
			// Skip hidden directories and common vendor directories
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		fileInfo, ok, err := collectFileInfo(path)
		if err != nil {
			return err
		}
		if ok {
			files = append(files, fileInfo)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return files, nil
}

func collectFileInfo(path string) (FileInfo, bool, error) {
	ext := filepath.Ext(path)
	lang, ok := SupportedExtensions[ext]
	if !ok {
		return FileInfo{}, false, nil
	}

	lines, err := countLines(path)
	if err != nil {
		return FileInfo{}, false, err
	}

	return FileInfo{
		Path:     path,
		Lines:    lines,
		Language: lang,
		IsTest:   isTestFile(path),
	}, true, nil
}

// countLines counts the number of lines in a file.
func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// isTestFile checks if a file matches test file patterns.
func isTestFile(path string) bool {
	name := filepath.Base(path)
	for _, pattern := range TestPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

// hasTestFiles checks if any files in the list are test files.
func hasTestFiles(files []FileInfo) bool {
	for _, f := range files {
		if f.IsTest {
			return true
		}
	}
	return false
}

// aggregateMetrics computes scope metrics from file list.
func aggregateMetrics(files []FileInfo) *ScopeMetrics {
	metrics := &ScopeMetrics{
		FileTypes: make(map[string]int),
	}

	langSet := make(map[string]bool)
	var maxLines int

	for _, f := range files {
		metrics.TotalFiles++
		metrics.TotalLines += f.Lines
		ext := filepath.Ext(f.Path)
		metrics.FileTypes[ext]++
		langSet[f.Language] = true

		if f.Lines > maxLines {
			maxLines = f.Lines
		}
		if f.Lines > 500 {
			metrics.FilesOver500Lines++
		}
	}

	// Extract languages as sorted list
	for lang := range langSet {
		metrics.Languages = append(metrics.Languages, lang)
	}
	sort.Strings(metrics.Languages)

	metrics.MaxFileLines = maxLines
	metrics.EstimatedTokens = metrics.TotalLines * 5 // ~5 tokens per line (heuristic)

	return metrics
}

// generateRecommendation creates a basic routing recommendation using heuristics.
func (ns *NativeScout) generateRecommendation(m *ScopeMetrics) *RoutingRecommendation {
	var tier, confidence, reasoning string

	isMultiLang := len(m.Languages) > 1

	switch {
	case m.TotalFiles < 5 && m.TotalLines < 500 && !isMultiLang:
		tier = "haiku"
		confidence = "high"
		reasoning = fmt.Sprintf("Small scope: %d files, %d lines", m.TotalFiles, m.TotalLines)

	case m.TotalFiles <= 15 && m.TotalLines < 2000 && !isMultiLang:
		tier = "sonnet"
		confidence = "high"
		reasoning = fmt.Sprintf("Medium scope: %d files, %d lines", m.TotalFiles, m.TotalLines)

	case m.TotalFiles > 15 || m.TotalLines >= 2000 || isMultiLang || m.EstimatedTokens > 50000:
		tier = "opus"
		confidence = "high"
		reasoning = fmt.Sprintf("Large or complex scope: %d files, %d lines, %d languages",
			m.TotalFiles, m.TotalLines, len(m.Languages))
		if m.EstimatedTokens > 50000 {
			reasoning += " (high token count)"
		}

	default:
		tier = "sonnet"
		confidence = "medium"
		reasoning = "Moderate scope"
	}

	// Adjust for complexity signals we CAN detect
	if m.FilesOver500Lines > 3 && tier == "haiku" {
		tier = "sonnet"
		reasoning += "; multiple large files detected"
	}

	return &RoutingRecommendation{
		RecommendedTier:     tier,
		Confidence:          confidence,
		Reasoning:           reasoning,
		ClarificationNeeded: nil,
	}
}

// identifyKeyFiles returns the N largest files by line count.
func identifyKeyFiles(files []FileInfo, n int) []KeyFile {
	// Sort by line count descending
	sorted := make([]FileInfo, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Lines > sorted[j].Lines
	})

	// Take top N
	if n > len(sorted) {
		n = len(sorted)
	}

	keyFiles := make([]KeyFile, 0, n)
	for i := 0; i < n; i++ {
		relevance := "Largest file"
		if i == 0 {
			relevance = "Largest file"
		} else if i < 3 {
			relevance = fmt.Sprintf("%d largest file", i+1)
		} else {
			relevance = "Large file"
		}

		keyFiles = append(keyFiles, KeyFile{
			Path:      sorted[i].Path,
			Lines:     sorted[i].Lines,
			Relevance: relevance,
		})
	}

	return keyFiles
}
