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
	FixturesDir    string        `json:"fixtures_dir"` // Path to test fixtures (deterministic scenarios)
	ReportFormat   string        `json:"report_format"` // "json", "markdown", "tap"
	Verbose        bool          `json:"verbose"`
}

// SetupFunc prepares test environment before scenario execution.
type SetupFunc func(cfg SimulationConfig) error

// TeardownFunc cleans up after scenario execution.
type TeardownFunc func(cfg SimulationConfig) error

// FixtureSetup represents the setup configuration from fixture JSON files.
// It defines directories to create, files to write, and environment variables.
type FixtureSetup struct {
	// CreateDirs lists directories to create relative to TempDir
	CreateDirs []string `json:"create_dirs,omitempty"`
	// Files maps relative paths to file contents
	Files map[string]string `json:"files,omitempty"`
	// Env maps environment variable names to values.
	// Special handling: GOGENT_DELEGATION_CEILING writes to .claude/tmp/max_delegation
	Env map[string]string `json:"env,omitempty"`
}

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

	// PostToolUse-specific expectations (sharp-edge detection)
	FilesNotCreated  []string            `json:"files_not_created,omitempty"`
	StdoutContains   []string            `json:"stdout_contains,omitempty"`
	StdoutNotContain []string            `json:"stdout_not_contains,omitempty"`
	StdoutEquals     string              `json:"stdout_equals,omitempty"`
	FileContains     map[string][]string `json:"file_contains,omitempty"`

	// Sharp edge schema validation
	ValidateSharpEdge bool                   `json:"validate_sharp_edge,omitempty"`
	SharpEdgeFields   map[string]interface{} `json:"sharp_edge_fields,omitempty"`

	// SessionStart-specific expectations
	AdditionalContextContains    []string `json:"additional_context_contains,omitempty"`
	AdditionalContextNotContains []string `json:"additional_context_not_contains,omitempty"`
	ProjectTypeEquals            string   `json:"project_type_equals,omitempty"`
	ToolCounterInitialized       bool     `json:"tool_counter_initialized,omitempty"`
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
