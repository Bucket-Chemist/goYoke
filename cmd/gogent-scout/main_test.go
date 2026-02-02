package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCountLines verifies line counting for various file types.
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty file", "", 0},
		{"single line", "hello", 1},
		{"multiple lines", "line1\nline2\nline3", 3},
		{"trailing newline", "line1\nline2\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test.txt")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			count, err := countLines(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, count)
		})
	}
}

// TestIsTestFile verifies test file pattern matching.
func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"handler_test.go", true},
		{"test_utils.py", true},
		{"utils_test.py", true},
		{"handler.go", false},
		{"main.py", false},
		{"test-helpers.R", true},
		{"spec.helper.ts", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTestFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAggregateMetrics verifies metrics aggregation logic.
func TestAggregateMetrics(t *testing.T) {
	files := []FileInfo{
		{Path: "a.go", Lines: 100, Language: "go"},
		{Path: "b.go", Lines: 200, Language: "go"},
		{Path: "c.py", Lines: 50, Language: "python"},
		{Path: "huge.go", Lines: 600, Language: "go"},
	}

	metrics := aggregateMetrics(files)

	assert.Equal(t, 4, metrics.TotalFiles)
	assert.Equal(t, 950, metrics.TotalLines)
	assert.Equal(t, 4750, metrics.EstimatedTokens) // 950 * 5
	assert.Equal(t, 600, metrics.MaxFileLines)
	assert.Equal(t, 1, metrics.FilesOver500Lines)
	assert.Contains(t, metrics.Languages, "go")
	assert.Contains(t, metrics.Languages, "python")
	assert.Equal(t, 3, metrics.FileTypes[".go"])
	assert.Equal(t, 1, metrics.FileTypes[".py"])
}

// TestGenerateRecommendation verifies tier recommendation logic.
func TestGenerateRecommendation(t *testing.T) {
	tests := []struct {
		name     string
		files    int
		lines    int
		wantTier string
	}{
		{"tiny project", 3, 200, "haiku"},
		{"small project", 10, 1000, "sonnet"},
		{"medium project", 15, 1800, "sonnet"},
		{"large project", 25, 10000, "external"},
		{"high token count", 10, 6000, "external"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ScopeMetrics{
				TotalFiles:      tt.files,
				TotalLines:      tt.lines,
				EstimatedTokens: tt.lines * 5,
			}

			ns := &NativeScout{}
			rec := ns.generateRecommendation(metrics)

			assert.Equal(t, tt.wantTier, rec.RecommendedTier)
			assert.NotEmpty(t, rec.Reasoning)
		})
	}
}

// TestGenerateRecommendationLargeFiles verifies adjustment for large files.
func TestGenerateRecommendationLargeFiles(t *testing.T) {
	// Small project but multiple large files should upgrade to sonnet
	metrics := &ScopeMetrics{
		TotalFiles:        4,
		TotalLines:        400,
		EstimatedTokens:   4000,
		FilesOver500Lines: 4,
	}

	ns := &NativeScout{}
	rec := ns.generateRecommendation(metrics)

	assert.Equal(t, "sonnet", rec.RecommendedTier)
	assert.Contains(t, rec.Reasoning, "large files")
}

// TestIdentifyKeyFiles verifies key file identification.
func TestIdentifyKeyFiles(t *testing.T) {
	files := []FileInfo{
		{Path: "small.go", Lines: 50},
		{Path: "large.go", Lines: 500},
		{Path: "medium.go", Lines: 200},
		{Path: "huge.go", Lines: 800},
	}

	keyFiles := identifyKeyFiles(files, 2)

	require.Len(t, keyFiles, 2)
	assert.Equal(t, "huge.go", keyFiles[0].Path)
	assert.Equal(t, 800, keyFiles[0].Lines)
	assert.Equal(t, "Largest file", keyFiles[0].Relevance)
	assert.Equal(t, "large.go", keyFiles[1].Path)
}

// TestRoutingScore verifies routing score calculation.
func TestRoutingScore(t *testing.T) {
	tests := []struct {
		name      string
		score     RoutingScore
		wantScore int
		wantRoute string
	}{
		{
			name:      "tiny project",
			score:     RoutingScore{FileCount: 3, TotalLines: 300, MaxFileLines: 100, LanguageCount: 1},
			wantScore: 12, // (15*40 + 6*30 + 10*20 + 25*10)/100
			wantRoute: "native",
		},
		{
			name:      "small project",
			score:     RoutingScore{FileCount: 8, TotalLines: 2000, MaxFileLines: 400, LanguageCount: 1},
			wantScore: 38, // (40*40 + 40*30 + 40*20 + 25*10)/100
			wantRoute: "native",
		},
		{
			name:      "medium project",
			score:     RoutingScore{FileCount: 15, TotalLines: 3000, MaxFileLines: 500, LanguageCount: 2},
			wantScore: 63,
			wantRoute: "gemini",
		},
		{
			name:      "large project",
			score:     RoutingScore{FileCount: 30, TotalLines: 10000, MaxFileLines: 1000, LanguageCount: 3},
			wantScore: 97,
			wantRoute: "gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.score.Score()
			assert.Equal(t, tt.wantScore, score)

			// Verify routing decision
			expectedBackend := "native"
			if score >= NativeThreshold {
				expectedBackend = "gemini"
			}
			assert.Equal(t, tt.wantRoute, expectedBackend)
		})
	}
}

// TestNativeScout_SmallProject verifies native scout on a small project.
func TestNativeScout_SmallProject(t *testing.T) {
	tmpDir := createTestFiles(t, map[string]string{
		"main.go":       "package main\n\nfunc main() {\n}\n",
		"handler.go":    "package main\n\nfunc handle() {}\n",
		"handler_test.go": "package main\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n",
	})

	scout := &NativeScout{Target: tmpDir, Instruction: "Test instruction"}
	report, err := scout.Run()

	require.NoError(t, err)
	assert.Equal(t, "native", report.Backend)
	assert.Equal(t, "1.0", report.SchemaVersion)
	assert.Equal(t, 3, report.ScopeMetrics.TotalFiles)
	assert.Contains(t, report.ScopeMetrics.Languages, "go")
	assert.True(t, report.ComplexitySignals.TestCoveragePresent)
	assert.False(t, report.ComplexitySignals.Available)
	assert.NotNil(t, report.RoutingRecommendation)
	assert.NotEmpty(t, report.KeyFiles)
}

// TestNativeScout_MixedLanguages verifies multi-language project handling.
func TestNativeScout_MixedLanguages(t *testing.T) {
	tmpDir := createTestFiles(t, map[string]string{
		"main.go":    "package main\n",
		"script.py":  "def main():\n    pass\n",
		"app.ts":     "function main() {}\n",
		"README.md":  "# Project\n",
	})

	scout := &NativeScout{Target: tmpDir}
	report, err := scout.Run()

	require.NoError(t, err)
	assert.Equal(t, 4, report.ScopeMetrics.TotalFiles)
	assert.Len(t, report.ScopeMetrics.Languages, 4)
	assert.Contains(t, report.ScopeMetrics.Languages, "go")
	assert.Contains(t, report.ScopeMetrics.Languages, "python")
	assert.Contains(t, report.ScopeMetrics.Languages, "typescript")
	assert.Contains(t, report.ScopeMetrics.Languages, "markdown")
}

// TestNativeScout_SkipsVendor verifies vendor directory skipping.
func TestNativeScout_SkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in root
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644))

	// Create vendor directory with files
	vendorDir := filepath.Join(tmpDir, "vendor")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("package lib\n"), 0644))

	// Create node_modules directory
	nodeDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nodeDir, "index.js"), []byte("module.exports = {};\n"), 0644))

	scout := &NativeScout{Target: tmpDir}
	report, err := scout.Run()

	require.NoError(t, err)
	// Should only count main.go, not vendor or node_modules files
	assert.Equal(t, 1, report.ScopeMetrics.TotalFiles)
}

// TestGenerateSyntheticReport verifies fallback report generation.
func TestGenerateSyntheticReport(t *testing.T) {
	tmpDir := createTestFiles(t, map[string]string{
		"file1.go": "package main\n",
		"file2.go": "package main\n",
	})

	primaryErr := assert.AnError
	fallbackErr := assert.AnError

	report := generateSyntheticReport(tmpDir, primaryErr, fallbackErr)

	assert.Equal(t, "synthetic_fallback", report.Backend)
	assert.Equal(t, "sonnet", report.RoutingRecommendation.RecommendedTier)
	assert.Equal(t, "low", report.RoutingRecommendation.Confidence)
	assert.Len(t, report.Warnings, 3)
	assert.Contains(t, report.Warnings[0], "Primary backend failed")
	assert.Contains(t, report.Warnings[1], "Fallback backend failed")
}

// TestAtomicWrite verifies atomic file writing.
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json")
	data := []byte(`{"test": "data"}`)

	err := atomicWrite(outputPath, data)
	require.NoError(t, err)

	// Verify file exists and contains correct data
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, data, content)

	// Verify no temp file remains
	tmpPath := outputPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err))
}

// TestValidateGeminiOutput verifies Gemini output validation.
func TestValidateGeminiOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid output",
			input: `{
				"schema_version": "1.0",
				"scout_report": {
					"backend": "gemini",
					"target": "/test",
					"timestamp": "2026-02-02T10:00:00Z",
					"scope_metrics": {
						"total_files": 10,
						"total_lines": 1000,
						"estimated_tokens": 10000,
						"languages": ["go"],
						"file_types": {".go": 10}
					},
					"complexity_signals": {
						"available": true,
						"import_density": "medium"
					},
					"routing_recommendation": {
						"recommended_tier": "sonnet",
						"confidence": "high",
						"reasoning": "Test"
					},
					"key_files": [],
					"warnings": []
				}
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name: "missing scout_report",
			input: `{
				"schema_version": "1.0"
			}`,
			wantErr: true,
		},
		{
			name: "missing scope_metrics",
			input: `{
				"schema_version": "1.0",
				"scout_report": {
					"routing_recommendation": {}
				}
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := validateGeminiOutput([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, report)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, report)
				assert.Equal(t, "gemini", report.Backend)
				assert.True(t, report.ComplexitySignals.Available)
			}
		})
	}
}

// TestScoutReport_JSONMarshaling verifies JSON serialization.
func TestScoutReport_JSONMarshaling(t *testing.T) {
	report := &ScoutReport{
		SchemaVersion: "1.0",
		Backend:       "native",
		Target:        "/test",
		Timestamp:     "2026-02-02T10:00:00Z",
		ScopeMetrics: &ScopeMetrics{
			TotalFiles:      5,
			TotalLines:      500,
			EstimatedTokens: 5000,
			Languages:       []string{"go"},
			FileTypes:       map[string]int{".go": 5},
		},
		ComplexitySignals: &ComplexitySignals{
			Available: false,
		},
		RoutingRecommendation: &RoutingRecommendation{
			RecommendedTier: "haiku",
			Confidence:      "high",
			Reasoning:       "Small scope",
		},
		KeyFiles: []KeyFile{},
		Warnings: []string{},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	// Verify it can be unmarshalled
	var decoded ScoutReport
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, report.Backend, decoded.Backend)
	assert.Equal(t, report.ScopeMetrics.TotalFiles, decoded.ScopeMetrics.TotalFiles)
}

// Benchmark tests

// BenchmarkNativeScout5Files benchmarks native scout on 5 files.
func BenchmarkNativeScout5Files(b *testing.B) {
	tmpDir := createBenchmarkFiles(b, 5)
	scout := &NativeScout{Target: tmpDir, Instruction: "Benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := scout.Run()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNativeScout15Files benchmarks native scout on 15 files.
func BenchmarkNativeScout15Files(b *testing.B) {
	tmpDir := createBenchmarkFiles(b, 15)
	scout := &NativeScout{Target: tmpDir, Instruction: "Benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scout.Run()
	}
}

// BenchmarkCalculateRoutingScore benchmarks routing score calculation.
func BenchmarkCalculateRoutingScore(b *testing.B) {
	tmpDir := createBenchmarkFiles(b, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calculateRoutingScore(tmpDir)
	}
}

// Helper functions

// createTestFiles creates a temporary directory with test files.
func createTestFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

// createBenchmarkFiles creates a directory with N Go files for benchmarking.
func createBenchmarkFiles(b *testing.B, n int) string {
	b.Helper()
	tmpDir := b.TempDir()

	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	for i := 0; i < n; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i))
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	return tmpDir
}
