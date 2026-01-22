package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONReporter_Generate(t *testing.T) {
	tests := []struct {
		name    string
		results []SimulationResult
		cfg     SimulationConfig
		wantErr bool
		check   func(t *testing.T, output string)
	}{
		{
			name: "basic results",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true, Duration: 100 * time.Millisecond},
				{ScenarioID: "V002", Passed: false, Duration: 50 * time.Millisecond, ErrorMsg: "test error"},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			check: func(t *testing.T, output string) {
				var report Report
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				if report.Summary.Total != 2 {
					t.Errorf("Expected 2 total, got: %d", report.Summary.Total)
				}
				if report.Summary.Passed != 1 {
					t.Errorf("Expected 1 passed, got: %d", report.Summary.Passed)
				}
				if report.Summary.Failed != 1 {
					t.Errorf("Expected 1 failed, got: %d", report.Summary.Failed)
				}
				if report.Summary.PassRate != 0.5 {
					t.Errorf("Expected 0.5 pass rate, got: %f", report.Summary.PassRate)
				}
			},
		},
		{
			name:    "empty results",
			results: []SimulationResult{},
			cfg:     SimulationConfig{Mode: "deterministic"},
			check: func(t *testing.T, output string) {
				var report Report
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}
				if report.Summary.Total != 0 {
					t.Errorf("Expected 0 total, got: %d", report.Summary.Total)
				}
			},
		},
		{
			name: "fuzz results",
			results: []SimulationResult{
				{ScenarioID: "FUZZ-P-0", Passed: true, Duration: 10 * time.Millisecond},
				{ScenarioID: "FUZZ-P-1", Passed: false, Duration: 15 * time.Millisecond},
				{ScenarioID: "FUZZ-S-0", Passed: true, Duration: 20 * time.Millisecond},
			},
			cfg: SimulationConfig{Mode: "fuzz", FuzzSeed: 12345},
			check: func(t *testing.T, output string) {
				var report Report
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				if report.FuzzResults == nil {
					t.Fatal("Expected fuzz results to be populated")
				}
				if report.FuzzResults.Iterations != 3 {
					t.Errorf("Expected 3 fuzz iterations, got: %d", report.FuzzResults.Iterations)
				}
				if report.FuzzResults.Crashes != 1 {
					t.Errorf("Expected 1 crash, got: %d", report.FuzzResults.Crashes)
				}
				if report.FuzzResults.Seed != 12345 {
					t.Errorf("Expected seed 12345, got: %d", report.FuzzResults.Seed)
				}
			},
		},
		{
			name: "mixed results",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true, Duration: 100 * time.Millisecond},
				{ScenarioID: "FUZZ-P-0", Passed: false, Duration: 10 * time.Millisecond},
			},
			cfg: SimulationConfig{Mode: "mixed"},
			check: func(t *testing.T, output string) {
				var report Report
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				if len(report.DeterministicResults) != 1 {
					t.Errorf("Expected 1 deterministic result, got: %d", len(report.DeterministicResults))
				}
				if report.FuzzResults == nil || report.FuzzResults.Iterations != 1 {
					t.Error("Expected 1 fuzz iteration")
				}
			},
		},
		{
			name: "unicode in error messages",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: false, ErrorMsg: "Error: 日本語 テスト €∞"},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			check: func(t *testing.T, output string) {
				var report Report
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}
				if !strings.Contains(output, "日本語") {
					t.Error("Unicode characters were not preserved")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &JSONReporter{}
			output, err := reporter.Generate(tt.results, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, output)
			}
		})
	}
}

func TestMarkdownReporter_Generate(t *testing.T) {
	tests := []struct {
		name       string
		results    []SimulationResult
		cfg        SimulationConfig
		wantErr    bool
		wantSubstr []string
	}{
		{
			name: "basic structure",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true, Duration: 100 * time.Millisecond},
				{ScenarioID: "V002", Passed: false, Duration: 50 * time.Millisecond, Diff: "expected != actual"},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			wantSubstr: []string{
				"# Simulation Test Report",
				"## Summary",
				"| V001 |",
				"### Failures",
				":white_check_mark: Pass",
				":x: Fail",
			},
		},
		{
			name:    "empty results",
			results: []SimulationResult{},
			cfg:     SimulationConfig{Mode: "deterministic"},
			wantSubstr: []string{
				"# Simulation Test Report",
				"## Summary",
			},
		},
		{
			name: "fuzz results",
			results: []SimulationResult{
				{ScenarioID: "FUZZ-P-0", Passed: false, Duration: 10 * time.Millisecond},
			},
			cfg: SimulationConfig{Mode: "fuzz", FuzzSeed: 99999},
			wantSubstr: []string{
				"## Fuzz Results",
				"**Iterations:** 1",
				"**Seed:** 99999",
				"**Crashes:** 1",
			},
		},
		{
			name: "failure details",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: false, ErrorMsg: "connection timeout", Diff: "--- expected\n+++ actual"},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			wantSubstr: []string{
				"#### V001",
				"**Error:** connection timeout",
				"<details>",
				"<summary>Diff</summary>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &MarkdownReporter{}
			output, err := reporter.Generate(tt.results, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				for _, substr := range tt.wantSubstr {
					if !strings.Contains(output, substr) {
						t.Errorf("Output missing expected substring: %q", substr)
					}
				}
			}
		})
	}
}

func TestTAPReporter_Generate(t *testing.T) {
	tests := []struct {
		name    string
		results []SimulationResult
		cfg     SimulationConfig
		wantErr bool
		check   func(t *testing.T, output string)
	}{
		{
			name: "basic TAP format",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
				{ScenarioID: "V002", Passed: false, ErrorMsg: "assertion failed"},
			},
			cfg: SimulationConfig{},
			check: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				if len(lines) < 4 {
					t.Fatalf("Expected at least 4 lines, got %d", len(lines))
				}

				if lines[0] != "TAP version 13" {
					t.Errorf("Expected TAP version 13, got: %s", lines[0])
				}
				if lines[1] != "1..2" {
					t.Errorf("Expected 1..2, got: %s", lines[1])
				}
				if !strings.HasPrefix(lines[2], "ok 1 -") {
					t.Errorf("Expected 'ok 1 -', got: %s", lines[2])
				}
				if !strings.HasPrefix(lines[3], "not ok 2 -") {
					t.Errorf("Expected 'not ok 2 -', got: %s", lines[3])
				}
			},
		},
		{
			name:    "empty results",
			results: []SimulationResult{},
			cfg:     SimulationConfig{},
			check: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if lines[0] != "TAP version 13" {
					t.Error("Missing TAP version header")
				}
				if lines[1] != "1..0" {
					t.Errorf("Expected 1..0 for empty results, got: %s", lines[1])
				}
			},
		},
		{
			name: "error and diff details",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: false, ErrorMsg: "timeout error", Diff: "line1\nline2\nline3"},
			},
			cfg: SimulationConfig{},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "message: timeout error") {
					t.Error("Error message not formatted correctly")
				}
				if !strings.Contains(output, "diff: |") {
					t.Error("Diff section missing")
				}
				if !strings.Contains(output, "    line1") {
					t.Error("Diff lines not properly indented")
				}
			},
		},
		{
			name: "special characters in error",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: false, ErrorMsg: "Error with \"quotes\" and \nnewlines\t and tabs"},
			},
			cfg: SimulationConfig{},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "message:") {
					t.Error("Error message section missing")
				}
			},
		},
		{
			name: "large result set",
			results: func() []SimulationResult {
				results := make([]SimulationResult, 100)
				for i := range results {
					results[i] = SimulationResult{
						ScenarioID: "V" + string(rune(i)),
						Passed:     i%2 == 0,
					}
				}
				return results
			}(),
			cfg: SimulationConfig{},
			check: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if lines[0] != "TAP version 13" {
					t.Error("Missing TAP version")
				}
				if lines[1] != "1..100" {
					t.Errorf("Expected 1..100, got: %s", lines[1])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &TAPReporter{}
			output, err := reporter.Generate(tt.results, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, output)
			}
		})
	}
}

func TestReporter_GenerateToFile(t *testing.T) {
	tests := []struct {
		name     string
		reporter Reporter
		filename string
		results  []SimulationResult
		cfg      SimulationConfig
		wantErr  bool
	}{
		{
			name:     "JSON to file",
			reporter: &JSONReporter{},
			filename: "test-report.json",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
			},
			cfg: SimulationConfig{},
		},
		{
			name:     "Markdown to file",
			reporter: &MarkdownReporter{},
			filename: "test-report.md",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
			},
			cfg: SimulationConfig{},
		},
		{
			name:     "TAP to file",
			reporter: &TAPReporter{},
			filename: "test-report.tap",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
			},
			cfg: SimulationConfig{},
		},
		{
			name:     "nested directory creation",
			reporter: &JSONReporter{},
			filename: "deeply/nested/path/report.json",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
			},
			cfg: SimulationConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, tt.filename)

			err := tt.reporter.GenerateToFile(tt.results, tt.cfg, path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GenerateToFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("Report file was not created")
				}

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read report file: %v", err)
				}
				if len(data) == 0 {
					t.Error("Report file is empty")
				}
			}
		})
	}
}

func TestNewReporter(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"json", "*harness.JSONReporter"},
		{"markdown", "*harness.MarkdownReporter"},
		{"tap", "*harness.TAPReporter"},
		{"unknown", "*harness.JSONReporter"}, // Default to JSON
		{"", "*harness.JSONReporter"},        // Empty defaults to JSON
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			reporter := NewReporter(tt.format)
			typeName := getTypeName(reporter)
			if typeName != tt.expected {
				t.Errorf("NewReporter(%s) = %s, want %s", tt.format, typeName, tt.expected)
			}
		})
	}
}

func TestBuildReport_FuzzResults(t *testing.T) {
	tests := []struct {
		name    string
		results []SimulationResult
		cfg     SimulationConfig
		check   func(t *testing.T, report Report)
	}{
		{
			name: "separates deterministic and fuzz",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true},
				{ScenarioID: "FUZZ-P-0", Passed: true},
				{ScenarioID: "FUZZ-P-1", Passed: false},
				{ScenarioID: "FUZZ-S-0", Passed: true},
			},
			cfg: SimulationConfig{Mode: "mixed", FuzzSeed: 12345},
			check: func(t *testing.T, report Report) {
				if len(report.DeterministicResults) != 1 {
					t.Errorf("Expected 1 deterministic result, got: %d", len(report.DeterministicResults))
				}

				if report.FuzzResults == nil {
					t.Fatal("Expected fuzz results to be populated")
				}

				if report.FuzzResults.Iterations != 3 {
					t.Errorf("Expected 3 fuzz iterations, got: %d", report.FuzzResults.Iterations)
				}

				if report.FuzzResults.Crashes != 1 {
					t.Errorf("Expected 1 crash, got: %d", report.FuzzResults.Crashes)
				}
			},
		},
		{
			name: "all passing",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true, Duration: 10 * time.Millisecond},
				{ScenarioID: "V002", Passed: true, Duration: 20 * time.Millisecond},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			check: func(t *testing.T, report Report) {
				if report.Summary.PassRate != 1.0 {
					t.Errorf("Expected 100%% pass rate, got: %.2f", report.Summary.PassRate*100)
				}
				if report.Summary.Failed != 0 {
					t.Errorf("Expected 0 failures, got: %d", report.Summary.Failed)
				}
			},
		},
		{
			name: "duration calculation",
			results: []SimulationResult{
				{ScenarioID: "V001", Passed: true, Duration: 100 * time.Millisecond},
				{ScenarioID: "V002", Passed: true, Duration: 200 * time.Millisecond},
				{ScenarioID: "V003", Passed: true, Duration: 300 * time.Millisecond},
			},
			cfg: SimulationConfig{Mode: "deterministic"},
			check: func(t *testing.T, report Report) {
				expected := 600 * time.Millisecond
				if report.Summary.Duration != expected {
					t.Errorf("Expected duration %v, got: %v", expected, report.Summary.Duration)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := buildReport(tt.results, tt.cfg)
			tt.check(t, report)
		})
	}
}

func TestEscapeYAML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: `simple text`, expected: `simple text`},
		{input: `text with "quotes"`, expected: `text with \"quotes\"`},
		{input: "text with\nnewline", expected: `text with\nnewline`},
		{input: "text with\ttab", expected: `text with\ttab`},
		{input: `text with \ backslash`, expected: `text with \\ backslash`},
		{input: "multi\nline\ntext", expected: `multi\nline\ntext`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeYAML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeYAML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReport_JSONRoundtrip(t *testing.T) {
	results := []SimulationResult{
		{ScenarioID: "V001", Passed: true, Duration: 100 * time.Millisecond},
		{ScenarioID: "V002", Passed: false, Duration: 50 * time.Millisecond, ErrorMsg: "test error", Diff: "diff output"},
	}
	cfg := SimulationConfig{Mode: "deterministic"}

	reporter := &JSONReporter{}
	output, err := reporter.Generate(results, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var report Report
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("Failed to unmarshal report: %v", err)
	}

	// Verify round-trip preserved data
	if report.Summary.Total != 2 {
		t.Errorf("Round-trip failed: expected 2 total, got %d", report.Summary.Total)
	}
	if len(report.DeterministicResults) != 2 {
		t.Errorf("Round-trip failed: expected 2 deterministic results, got %d", len(report.DeterministicResults))
	}
	if report.DeterministicResults[1].Error != "test error" {
		t.Errorf("Round-trip failed: error message not preserved")
	}
}

func TestMarkdownReporter_FuzzCrashDetails(t *testing.T) {
	// Test markdown generation with fuzz crash details including failed invariants
	results := []SimulationResult{
		{
			ScenarioID: "FUZZ-P-0",
			Passed:     false,
			Duration:   10 * time.Millisecond,
			Diff:       "[P1] exit code was non-zero\n[P2] output is not valid JSON",
		},
	}

	cfg := SimulationConfig{
		Mode:     "fuzz",
		FuzzSeed: 54321,
	}

	// Create a report with fuzz crashes
	report := buildReport(results, cfg)

	// Add fuzz crash details manually for testing markdown rendering
	report.FuzzResults.Failures = []FuzzCrash{
		{
			Iteration: 0,
			Seed:      54321,
			Category:  "pretooluse",
			Input:     map[string]interface{}{"tool_name": "Task", "model": "opus"},
			FailedInvariants: []InvariantResult{
				{InvariantID: "P1", Passed: false, Message: "exit code was non-zero"},
				{InvariantID: "P2", Passed: false, Message: "output is not valid JSON"},
			},
		},
	}
	report.FuzzResults.InvariantFailures = 2

	reporter := &MarkdownReporter{}

	// Generate markdown from the report manually
	var sb strings.Builder
	sb.WriteString("# Simulation Test Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s\n", report.RunID))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Mode:** %s\n\n", report.Config.Mode))

	sb.WriteString("## Fuzz Results\n\n")
	sb.WriteString(fmt.Sprintf("- **Iterations:** %d\n", report.FuzzResults.Iterations))
	sb.WriteString(fmt.Sprintf("- **Seed:** %d\n", report.FuzzResults.Seed))
	sb.WriteString(fmt.Sprintf("- **Crashes:** %d\n", report.FuzzResults.Crashes))
	sb.WriteString(fmt.Sprintf("- **Invariant Failures:** %d\n\n", report.FuzzResults.InvariantFailures))

	sb.WriteString("### Crash Details\n\n")
	for _, crash := range report.FuzzResults.Failures {
		sb.WriteString(fmt.Sprintf("#### Crash at iteration %d (seed: %d)\n\n", crash.Iteration, crash.Seed))
		sb.WriteString(fmt.Sprintf("**Category:** %s\n\n", crash.Category))

		if len(crash.FailedInvariants) > 0 {
			sb.WriteString("**Failed Invariants:**\n")
			for _, inv := range crash.FailedInvariants {
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", inv.InvariantID, inv.Message))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("<details>\n<summary>Input</summary>\n\n```json\n")
		inputJSON, _ := json.MarshalIndent(crash.Input, "", "  ")
		sb.WriteString(string(inputJSON))
		sb.WriteString("\n```\n</details>\n\n")
	}

	output := sb.String()

	// Verify key sections are present
	if !strings.Contains(output, "### Crash Details") {
		t.Error("Missing crash details section")
	}
	if !strings.Contains(output, "**Failed Invariants:**") {
		t.Error("Missing failed invariants section")
	}
	if !strings.Contains(output, "[P1] exit code was non-zero") {
		t.Error("Missing P1 invariant failure")
	}

	// Also verify full generation works
	_, err := reporter.Generate(results, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestGenerateToFile_ErrorHandling(t *testing.T) {
	results := []SimulationResult{
		{ScenarioID: "V001", Passed: true},
	}
	cfg := SimulationConfig{}

	t.Run("invalid path", func(t *testing.T) {
		// Try to write to a path that can't be created
		reporter := &JSONReporter{}
		err := reporter.GenerateToFile(results, cfg, "/proc/invalid/path/report.json")
		if err == nil {
			t.Error("Expected error when writing to invalid path")
		}
	})
}

func TestReportSummary_PassRateEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		results  []SimulationResult
		wantRate float64
	}{
		{
			name:     "all pass",
			results:  []SimulationResult{{Passed: true}, {Passed: true}},
			wantRate: 1.0,
		},
		{
			name:     "all fail",
			results:  []SimulationResult{{Passed: false}, {Passed: false}},
			wantRate: 0.0,
		},
		{
			name:     "half pass",
			results:  []SimulationResult{{Passed: true}, {Passed: false}},
			wantRate: 0.5,
		},
		{
			name:     "one third pass",
			results:  []SimulationResult{{Passed: true}, {Passed: false}, {Passed: false}},
			wantRate: 1.0 / 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add scenario IDs to results
			for i := range tt.results {
				tt.results[i].ScenarioID = fmt.Sprintf("V%03d", i+1)
			}

			report := buildReport(tt.results, SimulationConfig{})
			if report.Summary.PassRate != tt.wantRate {
				t.Errorf("Expected pass rate %.4f, got %.4f", tt.wantRate, report.Summary.PassRate)
			}
		})
	}
}

// getTypeName returns the type name of the reporter for testing.
func getTypeName(reporter Reporter) string {
	switch reporter.(type) {
	case *JSONReporter:
		return "*harness.JSONReporter"
	case *MarkdownReporter:
		return "*harness.MarkdownReporter"
	case *TAPReporter:
		return "*harness.TAPReporter"
	default:
		return "unknown"
	}
}
