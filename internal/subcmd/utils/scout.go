package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RunScout implements the goyoke-scout utility.
func RunScout(_ context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: scout <target> \"<instruction>\" [--output=<file>]\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  SCOUT_OUTPUT=<file>          Output file path\n")
		return fmt.Errorf("scout: requires <target> and <instruction> arguments")
	}

	target := args[0]
	instruction := args[1]
	var inputFiles []string

	if target == "-" {
		scanner := bufio.NewScanner(stdin)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				inputFiles = append(inputFiles, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("scout: read stdin file list: %w", err)
		}
		if len(inputFiles) == 0 {
			return fmt.Errorf("scout: no files provided via stdin")
		}
		target = scCommonRoot(inputFiles)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("scout: target not found: %s", target)
	}

	report, err := scRunScout(target, instruction, inputFiles)
	if err != nil {
		return fmt.Errorf("scout: %w", err)
	}

	outputPath := os.Getenv("SCOUT_OUTPUT")
	if outputPath == "" {
		for i, arg := range args {
			if strings.HasPrefix(arg, "--output=") {
				outputPath = strings.TrimPrefix(arg, "--output=")
			} else if arg == "--output" && i+1 < len(args) {
				outputPath = args[i+1]
			}
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("scout: marshal JSON: %w", err)
	}

	if outputPath != "" {
		if err := scAtomicWrite(outputPath, data); err != nil {
			return fmt.Errorf("scout: write output file: %w", err)
		}
	}

	fmt.Fprintln(stdout, string(data))
	return nil
}

// scReport is the unified output structure for scout operations.
type scReport struct {
	SchemaVersion string `json:"schema_version"`

	Backend   string `json:"backend"`
	Target    string `json:"target"`
	Timestamp string `json:"timestamp"`

	ScopeMetrics          *scScopeMetrics    `json:"scope_metrics"`
	ComplexitySignals     *scComplexitySignals `json:"complexity_signals"`
	RoutingRecommendation *scRoutingRec       `json:"routing_recommendation"`
	KeyFiles              []scKeyFile         `json:"key_files"`
	Warnings              []string            `json:"warnings"`
}

type scScopeMetrics struct {
	TotalFiles        int            `json:"total_files"`
	TotalLines        int            `json:"total_lines"`
	EstimatedTokens   int            `json:"estimated_tokens"`
	Languages         []string       `json:"languages"`
	FileTypes         map[string]int `json:"file_types"`
	MaxFileLines      int            `json:"max_file_lines"`
	FilesOver500Lines int            `json:"files_over_500_lines"`
}

type scComplexitySignals struct {
	Available             bool    `json:"available"`
	ImportDensity         *string `json:"import_density,omitempty"`
	CrossFileDependencies *int    `json:"cross_file_dependencies,omitempty"`
	TestCoveragePresent   bool    `json:"test_coverage_present"`
	Note                  string  `json:"note,omitempty"`
}

type scRoutingRec struct {
	RecommendedTier     string  `json:"recommended_tier"`
	Confidence          string  `json:"confidence"`
	Reasoning           string  `json:"reasoning"`
	ClarificationNeeded *string `json:"clarification_needed,omitempty"`
}

type scKeyFile struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Relevance string `json:"relevance"`
}

type scFileInfo struct {
	Path     string
	Lines    int
	Language string
	IsTest   bool
}

var scSupportedExtensions = map[string]string{
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

var scTestPatterns = []string{
	"_test.go",
	"test_",
	"_test.py",
	".test.",
	"test-",
	"spec.",
}

type scNativeScout struct {
	Target      string
	Instruction string
	Files       []string
}

func (ns *scNativeScout) run() (*scReport, error) {
	files, err := ns.collectFiles()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported files found in %s", ns.Target)
	}

	metrics := scAggregateMetrics(files)
	recommendation := ns.generateRecommendation(metrics)
	keyFiles := scIdentifyKeyFiles(files, 5)

	return &scReport{
		SchemaVersion: "1.0",
		Backend:       "native",
		Target:        ns.Target,
		Timestamp:     time.Now().Format(time.RFC3339),
		ScopeMetrics:  metrics,
		ComplexitySignals: &scComplexitySignals{
			Available:           false,
			TestCoveragePresent: scHasTestFiles(files),
			Note:                "Semantic analysis unavailable - basic metrics only",
		},
		RoutingRecommendation: recommendation,
		KeyFiles:              keyFiles,
		Warnings:              []string{},
	}, nil
}

func (ns *scNativeScout) collectFiles() ([]scFileInfo, error) {
	if len(ns.Files) > 0 {
		files := make([]scFileInfo, 0, len(ns.Files))
		for _, path := range ns.Files {
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
			if info.IsDir() {
				dirFiles, err := scCollectFilesFromDir(path)
				if err != nil {
					return nil, err
				}
				files = append(files, dirFiles...)
				continue
			}

			fi, ok, err := scCollectFileInfo(path)
			if err != nil {
				return nil, err
			}
			if ok {
				files = append(files, fi)
			}
		}
		return files, nil
	}

	return scCollectFilesFromDir(ns.Target)
}

func (ns *scNativeScout) generateRecommendation(m *scScopeMetrics) *scRoutingRec {
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

	if m.FilesOver500Lines > 3 && tier == "haiku" {
		tier = "sonnet"
		reasoning += "; multiple large files detected"
	}

	return &scRoutingRec{
		RecommendedTier:     tier,
		Confidence:          confidence,
		Reasoning:           reasoning,
		ClarificationNeeded: nil,
	}
}

func scCollectFilesFromDir(target string) ([]scFileInfo, error) {
	var files []scFileInfo
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
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
		fi, ok, err := scCollectFileInfo(path)
		if err != nil {
			return err
		}
		if ok {
			files = append(files, fi)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	return files, nil
}

func scCollectFileInfo(path string) (scFileInfo, bool, error) {
	ext := filepath.Ext(path)
	lang, ok := scSupportedExtensions[ext]
	if !ok {
		return scFileInfo{}, false, nil
	}

	lines, err := scCountLines(path)
	if err != nil {
		return scFileInfo{}, false, err
	}

	return scFileInfo{
		Path:     path,
		Lines:    lines,
		Language: lang,
		IsTest:   scIsTestFile(path),
	}, true, nil
}

func scCountLines(path string) (int, error) {
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

func scIsTestFile(path string) bool {
	name := filepath.Base(path)
	for _, pattern := range scTestPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

func scHasTestFiles(files []scFileInfo) bool {
	for _, f := range files {
		if f.IsTest {
			return true
		}
	}
	return false
}

func scAggregateMetrics(files []scFileInfo) *scScopeMetrics {
	metrics := &scScopeMetrics{
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

	for lang := range langSet {
		metrics.Languages = append(metrics.Languages, lang)
	}
	sort.Strings(metrics.Languages)

	metrics.MaxFileLines = maxLines
	metrics.EstimatedTokens = metrics.TotalLines * 5

	return metrics
}

func scIdentifyKeyFiles(files []scFileInfo, n int) []scKeyFile {
	sorted := make([]scFileInfo, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Lines > sorted[j].Lines
	})

	if n > len(sorted) {
		n = len(sorted)
	}

	keyFiles := make([]scKeyFile, 0, n)
	for i := 0; i < n; i++ {
		relevance := "Largest file"
		if i >= 1 && i < 3 {
			relevance = fmt.Sprintf("%d largest file", i+1)
		} else if i >= 3 {
			relevance = "Large file"
		}
		keyFiles = append(keyFiles, scKeyFile{
			Path:      sorted[i].Path,
			Lines:     sorted[i].Lines,
			Relevance: relevance,
		})
	}

	return keyFiles
}

func scGenerateSyntheticReport(target string, primaryErr, fallbackErr error) *scReport {
	fileCount := 0
	_ = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if _, ok := scSupportedExtensions[filepath.Ext(path)]; ok {
				fileCount++
			}
		}
		return nil
	})

	estimatedLines := fileCount * 100
	estimatedTokens := estimatedLines * 10

	warnings := []string{
		fmt.Sprintf("Primary backend failed: %v", primaryErr),
	}
	if fallbackErr != nil {
		warnings = append(warnings, fmt.Sprintf("Fallback backend failed: %v", fallbackErr))
	}
	warnings = append(warnings, "Using synthetic metrics - review recommended tier manually")

	return &scReport{
		SchemaVersion: "1.0",
		Backend:       "synthetic_fallback",
		Target:        target,
		Timestamp:     time.Now().Format(time.RFC3339),
		ScopeMetrics: &scScopeMetrics{
			TotalFiles:      fileCount,
			TotalLines:      estimatedLines,
			EstimatedTokens: estimatedTokens,
			Languages:       []string{"unknown"},
			FileTypes:       map[string]int{},
		},
		ComplexitySignals: &scComplexitySignals{
			Available: false,
			Note:      "Scout backends unavailable",
		},
		RoutingRecommendation: &scRoutingRec{
			RecommendedTier: "sonnet",
			Confidence:      "low",
			Reasoning:       "Scout failure - using conservative sonnet tier",
		},
		KeyFiles: []scKeyFile{},
		Warnings: warnings,
	}
}

func scRunScout(target, instruction string, files []string) (*scReport, error) {
	scout := &scNativeScout{Target: target, Instruction: instruction, Files: files}
	report, err := scout.run()

	if err == nil {
		return report, nil
	}

	log.Printf("Native scout failed: %v, generating synthetic report", err)
	return scGenerateSyntheticReport(target, err, nil), nil
}

func scCommonRoot(paths []string) string {
	if len(paths) == 0 {
		return "."
	}

	common := scNormalizePathRoot(paths[0])
	for _, path := range paths[1:] {
		common = scSharedPathPrefix(common, scNormalizePathRoot(path))
		if common == "." || common == string(filepath.Separator) {
			break
		}
	}
	return common
}

func scNormalizePathRoot(path string) string {
	clean := filepath.Clean(path)
	if info, err := os.Stat(clean); err == nil && !info.IsDir() {
		return filepath.Dir(clean)
	}
	return clean
}

func scSharedPathPrefix(a, b string) string {
	if a == b {
		return a
	}

	aParts := strings.Split(filepath.Clean(a), string(filepath.Separator))
	bParts := strings.Split(filepath.Clean(b), string(filepath.Separator))
	maxParts := len(aParts)
	if len(bParts) < maxParts {
		maxParts = len(bParts)
	}

	var shared []string
	for i := 0; i < maxParts; i++ {
		if aParts[i] != bParts[i] {
			break
		}
		shared = append(shared, aParts[i])
	}

	switch len(shared) {
	case 0:
		return "."
	case 1:
		if shared[0] == "" {
			return string(filepath.Separator)
		}
	}

	return filepath.Join(shared...)
}

func scAtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
