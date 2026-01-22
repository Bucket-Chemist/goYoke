package harness

import (
	"regexp"
	"time"
)

// SimulationConfig controls simulation execution behavior.
type SimulationConfig struct {
	Mode           string        `json:"mode"`            // "deterministic", "fuzz", "mixed"
	ScenarioFilter []string      `json:"scenario_filter"`
	FuzzIterations int           `json:"fuzz_iterations"`
	FuzzSeed       int64         `json:"fuzz_seed"`
	FuzzTimeout    time.Duration `json:"fuzz_timeout"`
	SchemaPath     string        `json:"schema_path"`
	AgentsPath     string        `json:"agents_path"`
	TempDir        string        `json:"temp_dir"`
	ReportFormat   string        `json:"report_format"` // "json", "markdown", "tap"
	Verbose        bool          `json:"verbose"`
}

// SetupFunc prepares test environment before scenario execution.
type SetupFunc func(cfg SimulationConfig) error

// TeardownFunc cleans up after scenario execution.
type TeardownFunc func(cfg SimulationConfig) error

// Scenario defines a single test case for simulation.
type Scenario struct {
	ID          string         `json:"id"`
	Category    string         `json:"category"`
	Description string         `json:"description"`
	Input       interface{}    `json:"input"`
	Setup       SetupFunc      `json:"-"`
	Expected    ExpectedOutput `json:"expected"`
	Teardown    TeardownFunc   `json:"-"`
}

// ExpectedOutput defines what a scenario should produce.
type ExpectedOutput struct {
	Decision      *string                `json:"decision,omitempty"`
	ReasonMatch   *regexp.Regexp         `json:"-"`
	ReasonPattern string                 `json:"reason_pattern,omitempty"`
	HasViolation  *string                `json:"has_violation,omitempty"`
	HandoffFields map[string]interface{} `json:"handoff_fields,omitempty"`
	FilesCreated  []string               `json:"files_created,omitempty"`
	ExitCode      int                    `json:"exit_code"`
	StderrMatch   *regexp.Regexp         `json:"-"`
	StderrPattern string                 `json:"stderr_pattern,omitempty"`
}

// SimulationResult captures the outcome of running a scenario.
type SimulationResult struct {
	ScenarioID string        `json:"scenario_id"`
	Passed     bool          `json:"passed"`
	Duration   time.Duration `json:"duration"`
	Input      string        `json:"input"`
	Output     string        `json:"output"`
	Expected   string        `json:"expected"`
	Diff       string        `json:"diff,omitempty"`
	Error      error         `json:"-"`
	ErrorMsg   string        `json:"error,omitempty"`
}

// Runner executes simulation scenarios.
type Runner interface {
	Run(cfg SimulationConfig) ([]SimulationResult, error)
	RunScenario(s Scenario) SimulationResult
}

// DefaultConfig returns a SimulationConfig with sensible defaults.
func DefaultConfig() SimulationConfig {
	return SimulationConfig{
		Mode:           "deterministic",
		FuzzIterations: 1000,
		FuzzTimeout:    5 * time.Minute,
		ReportFormat:   "json",
		Verbose:        false,
	}
}
