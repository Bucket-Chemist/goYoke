package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Reporter generates simulation reports in various formats.
type Reporter interface {
	Generate(results []SimulationResult, cfg SimulationConfig) (string, error)
	GenerateToFile(results []SimulationResult, cfg SimulationConfig, path string) error
}

// Report represents a complete simulation report.
type Report struct {
	RunID                string           `json:"run_id"`
	GeneratedAt          time.Time        `json:"generated_at"`
	Config               SimulationConfig `json:"config"`
	Summary              ReportSummary    `json:"summary"`
	DeterministicResults []ScenarioReport `json:"deterministic_results,omitempty"`
	FuzzResults          *FuzzReport      `json:"fuzz_results,omitempty"`
}

// ReportSummary provides aggregate statistics.
type ReportSummary struct {
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration_ms"`
	PassRate float64       `json:"pass_rate"`
}

// ScenarioReport summarizes a single scenario result.
type ScenarioReport struct {
	ID       string        `json:"id"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration_ms"`
	Error    string        `json:"error,omitempty"`
	Diff     string        `json:"diff,omitempty"`
}

// FuzzReport summarizes fuzz testing results.
type FuzzReport struct {
	Iterations        int         `json:"iterations"`
	Seed              int64       `json:"seed"`
	Crashes           int         `json:"crashes"`
	InvariantFailures int         `json:"invariant_failures"`
	Failures          []FuzzCrash `json:"failures,omitempty"`
}

// JSONReporter generates JSON format reports.
type JSONReporter struct{}

// MarkdownReporter generates Markdown format reports.
type MarkdownReporter struct{}

// TAPReporter generates TAP (Test Anything Protocol) format.
type TAPReporter struct{}

// NewReporter creates a reporter for the given format.
func NewReporter(format string) Reporter {
	switch format {
	case "markdown":
		return &MarkdownReporter{}
	case "tap":
		return &TAPReporter{}
	default:
		return &JSONReporter{}
	}
}

// Generate creates a JSON report string.
func (r *JSONReporter) Generate(results []SimulationResult, cfg SimulationConfig) (string, error) {
	report := buildReport(results, cfg)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}

	return string(data), nil
}

// GenerateToFile writes a JSON report to file.
func (r *JSONReporter) GenerateToFile(results []SimulationResult, cfg SimulationConfig, path string) error {
	output, err := r.Generate(results, cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}

	return nil
}

// Generate creates a Markdown report string.
func (r *MarkdownReporter) Generate(results []SimulationResult, cfg SimulationConfig) (string, error) {
	report := buildReport(results, cfg)

	var sb strings.Builder

	// Header
	sb.WriteString("# Simulation Test Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s\n", report.RunID))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Mode:** %s\n\n", cfg.Mode))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total | %d |\n", report.Summary.Total))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", report.Summary.Passed))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", report.Summary.Failed))
	sb.WriteString(fmt.Sprintf("| Pass Rate | %.1f%% |\n", report.Summary.PassRate*100))
	sb.WriteString(fmt.Sprintf("| Duration | %v |\n", report.Summary.Duration))
	sb.WriteString("\n")

	// Deterministic Results
	if len(report.DeterministicResults) > 0 {
		sb.WriteString("## Deterministic Results\n\n")
		sb.WriteString("| Scenario | Status | Duration |\n")
		sb.WriteString("|----------|--------|----------|\n")

		for _, r := range report.DeterministicResults {
			status := ":white_check_mark: Pass"
			if !r.Passed {
				status = ":x: Fail"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %v |\n", r.ID, status, r.Duration))
		}
		sb.WriteString("\n")

		// Failed details
		var failures []ScenarioReport
		for _, r := range report.DeterministicResults {
			if !r.Passed {
				failures = append(failures, r)
			}
		}

		if len(failures) > 0 {
			sb.WriteString("### Failures\n\n")
			for _, f := range failures {
				sb.WriteString(fmt.Sprintf("#### %s\n\n", f.ID))
				if f.Error != "" {
					sb.WriteString(fmt.Sprintf("**Error:** %s\n\n", f.Error))
				}
				if f.Diff != "" {
					sb.WriteString("<details>\n<summary>Diff</summary>\n\n```\n")
					sb.WriteString(f.Diff)
					sb.WriteString("\n```\n</details>\n\n")
				}
			}
		}
	}

	// Fuzz Results
	if report.FuzzResults != nil {
		sb.WriteString("## Fuzz Results\n\n")
		sb.WriteString(fmt.Sprintf("- **Iterations:** %d\n", report.FuzzResults.Iterations))
		sb.WriteString(fmt.Sprintf("- **Seed:** %d\n", report.FuzzResults.Seed))
		sb.WriteString(fmt.Sprintf("- **Crashes:** %d\n", report.FuzzResults.Crashes))
		sb.WriteString(fmt.Sprintf("- **Invariant Failures:** %d\n\n", report.FuzzResults.InvariantFailures))

		if len(report.FuzzResults.Failures) > 0 {
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
		}
	}

	return sb.String(), nil
}

// GenerateToFile writes a Markdown report to file.
func (r *MarkdownReporter) GenerateToFile(results []SimulationResult, cfg SimulationConfig, path string) error {
	output, err := r.Generate(results, cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}

	return nil
}

// Generate creates a TAP format report string.
func (r *TAPReporter) Generate(results []SimulationResult, cfg SimulationConfig) (string, error) {
	var sb strings.Builder

	sb.WriteString("TAP version 13\n")
	sb.WriteString(fmt.Sprintf("1..%d\n", len(results)))

	for i, result := range results {
		if result.Passed {
			sb.WriteString(fmt.Sprintf("ok %d - %s\n", i+1, result.ScenarioID))
		} else {
			sb.WriteString(fmt.Sprintf("not ok %d - %s\n", i+1, result.ScenarioID))
			if result.ErrorMsg != "" {
				sb.WriteString(fmt.Sprintf("  ---\n  message: %s\n  ...\n", escapeYAML(result.ErrorMsg)))
			}
			if result.Diff != "" {
				sb.WriteString(fmt.Sprintf("  ---\n  diff: |\n    %s\n  ...\n", strings.ReplaceAll(result.Diff, "\n", "\n    ")))
			}
		}
	}

	return sb.String(), nil
}

// GenerateToFile writes a TAP report to file.
func (r *TAPReporter) GenerateToFile(results []SimulationResult, cfg SimulationConfig, path string) error {
	output, err := r.Generate(results, cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}

	return nil
}

// buildReport creates a Report from simulation results.
func buildReport(results []SimulationResult, cfg SimulationConfig) Report {
	runID := fmt.Sprintf("sim-%s", time.Now().Format("20060102-150405"))

	report := Report{
		RunID:       runID,
		GeneratedAt: time.Now(),
		Config:      cfg,
	}

	// Calculate summary
	var totalDuration time.Duration
	for _, r := range results {
		totalDuration += r.Duration
		if r.Passed {
			report.Summary.Passed++
		} else {
			report.Summary.Failed++
		}
	}

	report.Summary.Total = len(results)
	report.Summary.Duration = totalDuration
	if report.Summary.Total > 0 {
		report.Summary.PassRate = float64(report.Summary.Passed) / float64(report.Summary.Total)
	}

	// Separate deterministic and fuzz results
	for _, r := range results {
		if strings.HasPrefix(r.ScenarioID, "FUZZ-") {
			if report.FuzzResults == nil {
				report.FuzzResults = &FuzzReport{
					Seed: cfg.FuzzSeed,
				}
			}
			report.FuzzResults.Iterations++
			if !r.Passed {
				report.FuzzResults.Crashes++
			}
		} else {
			report.DeterministicResults = append(report.DeterministicResults, ScenarioReport{
				ID:       r.ScenarioID,
				Passed:   r.Passed,
				Duration: r.Duration,
				Error:    r.ErrorMsg,
				Diff:     r.Diff,
			})
		}
	}

	return report
}

// escapeYAML escapes special characters for YAML strings in TAP output.
func escapeYAML(s string) string {
	// Replace problematic characters for YAML strings
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
