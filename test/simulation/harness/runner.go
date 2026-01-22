package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// DefaultRunner implements the Runner interface for CLI execution.
type DefaultRunner struct {
	config       SimulationConfig
	validatePath string
	archivePath  string
	generator    Generator
}

// NewRunner creates a runner with paths to CLI binaries.
func NewRunner(cfg SimulationConfig, validatePath, archivePath string, gen Generator) *DefaultRunner {
	return &DefaultRunner{
		config:       cfg,
		validatePath: validatePath,
		archivePath:  archivePath,
		generator:    gen,
	}
}

// SetTempDir updates the temp directory for test isolation.
// Used by fuzz runner to set per-iteration directories.
func (r *DefaultRunner) SetTempDir(dir string) {
	r.config.TempDir = dir
}

// Run executes all scenarios matching the configuration.
func (r *DefaultRunner) Run(cfg SimulationConfig) ([]SimulationResult, error) {
	r.config = cfg

	scenarios, err := r.loadScenarios()
	if err != nil {
		return nil, fmt.Errorf("load scenarios: %w", err)
	}

	var results []SimulationResult
	for _, s := range scenarios {
		if !r.matchesFilter(s.ID) {
			continue
		}

		result := r.RunScenario(s)
		results = append(results, result)

		if r.config.Verbose {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			fmt.Printf("[%s] %s (%v)\n", status, s.ID, result.Duration)
		}
	}

	return results, nil
}

// RunScenario executes a single scenario and returns the result.
func (r *DefaultRunner) RunScenario(s Scenario) SimulationResult {
	start := time.Now()
	result := SimulationResult{
		ScenarioID: s.ID,
	}

	// Clean up temp directory for test isolation
	// Each scenario starts with a clean slate
	if r.config.TempDir != "" {
		claudeDir := filepath.Join(r.config.TempDir, ".claude")
		os.RemoveAll(claudeDir)
	}

	// Setup
	if s.Setup != nil {
		if err := s.Setup(r.config); err != nil {
			result.Error = fmt.Errorf("setup failed: %w", err)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Execute
	output, exitCode, err := r.executeScenario(s)
	result.Duration = time.Since(start)
	result.Output = output

	if err != nil {
		result.Error = err
		result.ErrorMsg = err.Error()
	}

	// Validate
	result.Passed, result.Expected, result.Diff = r.validateOutput(s.Expected, output, exitCode)

	// Serialize input for debugging
	if inputBytes, err := json.Marshal(s.Input); err == nil {
		result.Input = string(inputBytes)
	}

	// Teardown
	if s.Teardown != nil {
		if err := s.Teardown(r.config); err != nil {
			// Log but don't fail test for teardown errors
			if r.config.Verbose {
				fmt.Printf("Warning: teardown failed for %s: %v\n", s.ID, err)
			}
		}
	}

	return result
}

// executeScenario runs the appropriate CLI with the scenario input.
func (r *DefaultRunner) executeScenario(s Scenario) (string, int, error) {
	var cmdPath string
	switch s.Category {
	case "pretooluse":
		cmdPath = r.validatePath
	case "sessionend":
		cmdPath = r.archivePath
	default:
		return "", -1, fmt.Errorf("unknown category: %s", s.Category)
	}

	inputBytes, err := json.Marshal(s.Input)
	if err != nil {
		return "", -1, fmt.Errorf("marshal input: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath)
	cmd.Stdin = bytes.NewReader(inputBytes)
	cmd.Env = r.buildEnv()

	// Create a new process group so we can kill the entire tree on timeout.
	// Without this, child processes (e.g., sleep in a bash script) survive
	// context cancellation because CommandContext only signals the direct child.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Cancel sends SIGKILL to the process group (negative PID).
	// This ensures all descendants are terminated on context deadline.
	cmd.Cancel = func() error {
		// Kill entire process group by sending signal to negative PGID
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	// WaitDelay gives child processes time to exit after Cancel before
	// forcibly terminating. 100ms is sufficient for clean shutdown.
	cmd.WaitDelay = 100 * time.Millisecond

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Check context timeout/cancellation first - this takes priority
	if ctx.Err() != nil {
		return "", -1, fmt.Errorf("execute command: %w", ctx.Err())
	}

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil // Non-zero exit is not necessarily an error
	} else if err != nil {
		return "", -1, fmt.Errorf("execute command: %w", err)
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[STDERR]\n" + stderr.String()
	}

	return output, exitCode, nil
}

// validateOutput checks if output matches expectations.
func (r *DefaultRunner) validateOutput(expected ExpectedOutput, output string, exitCode int) (bool, string, string) {
	var expectedParts []string
	var issues []string

	// Check exit code
	if expected.ExitCode != exitCode {
		issues = append(issues, fmt.Sprintf("exit code: got %d, want %d", exitCode, expected.ExitCode))
	}
	expectedParts = append(expectedParts, fmt.Sprintf("exit_code=%d", expected.ExitCode))

	// Parse output as JSON for structured validation
	var outputJSON map[string]interface{}
	if err := json.Unmarshal([]byte(output), &outputJSON); err == nil {
		// Check decision
		if expected.Decision != nil {
			if decision, ok := outputJSON["decision"].(string); ok {
				if decision != *expected.Decision {
					issues = append(issues, fmt.Sprintf("decision: got %s, want %s", decision, *expected.Decision))
				}
			} else if *expected.Decision != "" {
				issues = append(issues, "decision field missing")
			}
			expectedParts = append(expectedParts, fmt.Sprintf("decision=%s", *expected.Decision))
		}

		// Check reason pattern
		if expected.ReasonMatch != nil || expected.ReasonPattern != "" {
			pattern := expected.ReasonMatch
			if pattern == nil && expected.ReasonPattern != "" {
				pattern = regexp.MustCompile(expected.ReasonPattern)
			}
			if pattern != nil {
				if reason, ok := outputJSON["reason"].(string); ok {
					if !pattern.MatchString(reason) {
						issues = append(issues, fmt.Sprintf("reason pattern: %q not in %q", expected.ReasonPattern, reason))
					}
				}
			}
		}
	}

	// Check stderr pattern
	if expected.StderrMatch != nil || expected.StderrPattern != "" {
		pattern := expected.StderrMatch
		if pattern == nil && expected.StderrPattern != "" {
			pattern = regexp.MustCompile(expected.StderrPattern)
		}
		if pattern != nil && !pattern.MatchString(output) {
			issues = append(issues, fmt.Sprintf("stderr pattern: %q not found", expected.StderrPattern))
		}
	}

	expectedStr := strings.Join(expectedParts, ", ")
	diffStr := strings.Join(issues, "\n")

	return len(issues) == 0, expectedStr, diffStr
}

// buildEnv creates a minimal, controlled environment for CLI execution.
// Instead of inheriting the full os.Environ() (which differs between CI and local),
// we construct a minimal set of required variables for reproducible behavior.
func (r *DefaultRunner) buildEnv() []string {
	// Start with minimal required environment variables
	env := []string{
		// Basic shell requirements
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		// Locale for consistent string handling
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		// Disable color output for consistent parsing
		"NO_COLOR=1",
		"TERM=dumb",
	}

	// Add test isolation paths
	if r.config.SchemaPath != "" {
		env = append(env, "GOGENT_ROUTING_SCHEMA="+r.config.SchemaPath)
	}
	if r.config.AgentsPath != "" {
		env = append(env, "GOGENT_AGENTS_INDEX="+r.config.AgentsPath)
	}
	if r.config.TempDir != "" {
		env = append(env, "GOGENT_PROJECT_DIR="+r.config.TempDir)
	}

	return env
}

// matchesFilter checks if scenario ID matches any filter pattern.
func (r *DefaultRunner) matchesFilter(id string) bool {
	if len(r.config.ScenarioFilter) == 0 {
		return true
	}

	for _, filter := range r.config.ScenarioFilter {
		if strings.HasPrefix(id, filter) {
			return true
		}
	}
	return false
}

// loadScenarios loads all scenario definitions from fixtures.
func (r *DefaultRunner) loadScenarios() ([]Scenario, error) {
	var scenarios []Scenario

	// Load PreToolUse scenarios
	preToolDir := filepath.Join(r.config.FixturesDir, "deterministic", "pretooluse")
	if err := r.loadScenariosFromDir(preToolDir, "pretooluse", &scenarios); err != nil {
		return nil, err
	}

	// Load SessionEnd scenarios
	sessionDir := filepath.Join(r.config.FixturesDir, "deterministic", "sessionend")
	if err := r.loadScenariosFromDir(sessionDir, "sessionend", &scenarios); err != nil {
		return nil, err
	}

	if r.config.Verbose {
		fmt.Printf("[INFO] Loaded %d deterministic scenarios\n", len(scenarios))
	}

	return scenarios, nil
}

// loadScenariosFromDir loads scenarios from a directory.
func (r *DefaultRunner) loadScenariosFromDir(dir, category string, scenarios *[]Scenario) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var fixture struct {
			Input    interface{}    `json:"input"`
			Setup    *FixtureSetup  `json:"setup,omitempty"`
			Expected ExpectedOutput `json:"expected"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		scenario := Scenario{
			ID:       id,
			Category: category,
			Input:    fixture.Input,
			Expected: fixture.Expected,
		}

		// Create SetupFunc from fixture setup if present
		if fixture.Setup != nil {
			scenario.Setup = r.createSetupFunc(fixture.Setup)
		}

		*scenarios = append(*scenarios, scenario)
	}

	return nil
}

// createSetupFunc converts a FixtureSetup into a SetupFunc that prepares the test environment.
func (r *DefaultRunner) createSetupFunc(setup *FixtureSetup) SetupFunc {
	return func(cfg SimulationConfig) error {
		baseDir := cfg.TempDir
		if baseDir == "" {
			return fmt.Errorf("TempDir not set in config")
		}

		// Create directories
		for _, dir := range setup.CreateDirs {
			fullPath := filepath.Join(baseDir, dir)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", dir, err)
			}
		}

		// Create files
		for relPath, content := range setup.Files {
			fullPath := filepath.Join(baseDir, relPath)

			// Ensure parent directory exists
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("create parent dir for %s: %w", relPath, err)
			}

			// Handle ${TEMP_DIR} placeholder in content
			expandedContent := strings.ReplaceAll(content, "${TEMP_DIR}", baseDir)

			if err := os.WriteFile(fullPath, []byte(expandedContent), 0644); err != nil {
				return fmt.Errorf("write file %s: %w", relPath, err)
			}
		}

		// Handle environment variables
		// Special case: GOGENT_DELEGATION_CEILING writes to file, not env var
		for key, value := range setup.Env {
			// Expand ${TEMP_DIR} in env values
			expandedValue := strings.ReplaceAll(value, "${TEMP_DIR}", baseDir)

			if key == "GOGENT_DELEGATION_CEILING" {
				// Write ceiling to the file the CLI reads
				ceilingPath := filepath.Join(baseDir, ".claude", "tmp", "max_delegation")
				ceilingDir := filepath.Dir(ceilingPath)
				if err := os.MkdirAll(ceilingDir, 0755); err != nil {
					return fmt.Errorf("create ceiling dir: %w", err)
				}
				if err := os.WriteFile(ceilingPath, []byte(expandedValue), 0644); err != nil {
					return fmt.Errorf("write ceiling file: %w", err)
				}
			}
			// Note: Other env vars would need to be passed via buildEnv()
			// Currently, fixture env vars other than GOGENT_DELEGATION_CEILING
			// are not propagated to the CLI. This could be extended if needed.
		}

		return nil
	}
}
