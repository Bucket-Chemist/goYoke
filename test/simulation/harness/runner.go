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
	config          SimulationConfig
	validatePath    string
	archivePath     string
	sharpEdgePath   string
	loadContextPath string // Path to gogent-load-context binary
	generator       Generator
	scenarioEnv     map[string]string // Per-scenario environment variables
}

// NewRunner creates a runner with paths to CLI binaries.
// sharpEdgePath is optional and can be set later via SetSharpEdgePath.
func NewRunner(cfg SimulationConfig, validatePath, archivePath string, gen Generator) *DefaultRunner {
	return &DefaultRunner{
		config:        cfg,
		validatePath:  validatePath,
		archivePath:   archivePath,
		sharpEdgePath: "", // Set via SetSharpEdgePath for posttooluse scenarios
		generator:     gen,
	}
}

// SetSharpEdgePath sets the path to gogent-sharp-edge binary.
// Required for posttooluse scenario execution.
func (r *DefaultRunner) SetSharpEdgePath(path string) {
	r.sharpEdgePath = path
}

// SetLoadContextPath sets the path to gogent-load-context binary.
// Required for sessionstart scenario execution.
func (r *DefaultRunner) SetLoadContextPath(path string) {
	r.loadContextPath = path
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

	// Clear scenario-specific env vars
	r.scenarioEnv = make(map[string]string)

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
	case "posttooluse":
		if r.sharpEdgePath == "" {
			return "", -1, fmt.Errorf("posttooluse scenario requires sharpEdgePath (gogent-sharp-edge binary)")
		}
		cmdPath = r.sharpEdgePath
	case "sessionstart":
		if r.loadContextPath == "" {
			return "", -1, fmt.Errorf("sessionstart scenario requires loadContextPath (gogent-load-context binary)")
		}
		cmdPath = r.loadContextPath
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

	// PostToolUse-specific validations
	issues = append(issues, r.validatePostToolUseExpectations(expected, output)...)

	// SessionStart-specific validations
	issues = append(issues, r.validateSessionStartExpectations(expected, output)...)

	expectedStr := strings.Join(expectedParts, ", ")
	diffStr := strings.Join(issues, "\n")

	return len(issues) == 0, expectedStr, diffStr
}

// validatePostToolUseExpectations handles posttooluse-specific validation.
func (r *DefaultRunner) validatePostToolUseExpectations(expected ExpectedOutput, output string) []string {
	var issues []string

	// Check stdout_equals (exact match)
	if expected.StdoutEquals != "" {
		trimmedOutput := strings.TrimSpace(output)
		if trimmedOutput != expected.StdoutEquals {
			issues = append(issues, fmt.Sprintf("stdout_equals: got %q, want %q", trimmedOutput, expected.StdoutEquals))
		}
	}

	// Check stdout_contains (substring match)
	for _, substr := range expected.StdoutContains {
		if !strings.Contains(output, substr) {
			issues = append(issues, fmt.Sprintf("stdout_contains: %q not found in output", substr))
		}
	}

	// Check stdout_not_contains (negative substring match)
	for _, substr := range expected.StdoutNotContain {
		if strings.Contains(output, substr) {
			issues = append(issues, fmt.Sprintf("stdout_not_contains: %q found in output (should be absent)", substr))
		}
	}

	// Check files_created
	for _, relPath := range expected.FilesCreated {
		fullPath := filepath.Join(r.config.TempDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("files_created: %s not found", relPath))
		}
	}

	// Check files_not_created (negative file check)
	for _, relPath := range expected.FilesNotCreated {
		fullPath := filepath.Join(r.config.TempDir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			issues = append(issues, fmt.Sprintf("files_not_created: %s exists (should not)", relPath))
		}
	}

	// Check file_contains (substring in file content)
	for relPath, substrings := range expected.FileContains {
		fullPath := filepath.Join(r.config.TempDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("file_contains: cannot read %s: %v", relPath, err))
			continue
		}
		contentStr := string(content)
		for _, substr := range substrings {
			if !strings.Contains(contentStr, substr) {
				issues = append(issues, fmt.Sprintf("file_contains: %q not found in %s", substr, relPath))
			}
		}
	}

	// Validate sharp edge schema compliance
	if expected.ValidateSharpEdge {
		issues = append(issues, r.validateSharpEdgeSchema(expected.SharpEdgeFields)...)
	}

	return issues
}

// validateSharpEdgeSchema validates sharp edge output against expected fields.
func (r *DefaultRunner) validateSharpEdgeSchema(expectedFields map[string]interface{}) []string {
	var issues []string

	// Read the pending-learnings.jsonl file
	pendingPath := filepath.Join(r.config.TempDir, ".claude", "memory", "pending-learnings.jsonl")
	content, err := os.ReadFile(pendingPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("validate_sharp_edge: cannot read pending-learnings.jsonl: %v", err))
		return issues
	}

	// Parse the last line (most recent entry)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		issues = append(issues, "validate_sharp_edge: pending-learnings.jsonl is empty")
		return issues
	}

	lastLine := lines[len(lines)-1]
	var sharpEdge map[string]interface{}
	if err := json.Unmarshal([]byte(lastLine), &sharpEdge); err != nil {
		issues = append(issues, fmt.Sprintf("validate_sharp_edge: invalid JSON in last entry: %v", err))
		return issues
	}

	// Validate expected fields
	for field, expectedValue := range expectedFields {
		actualValue, exists := sharpEdge[field]
		if !exists {
			issues = append(issues, fmt.Sprintf("sharp_edge_fields: missing field %q", field))
			continue
		}

		// Handle numeric comparison (JSON unmarshals numbers as float64)
		switch expected := expectedValue.(type) {
		case float64:
			if actual, ok := actualValue.(float64); ok {
				if actual != expected {
					issues = append(issues, fmt.Sprintf("sharp_edge_fields: %s got %v, want %v", field, actual, expected))
				}
			} else {
				issues = append(issues, fmt.Sprintf("sharp_edge_fields: %s type mismatch: got %T, want float64", field, actualValue))
			}
		case string:
			if actual, ok := actualValue.(string); ok {
				if actual != expected {
					issues = append(issues, fmt.Sprintf("sharp_edge_fields: %s got %q, want %q", field, actual, expected))
				}
			} else {
				issues = append(issues, fmt.Sprintf("sharp_edge_fields: %s type mismatch: got %T, want string", field, actualValue))
			}
		default:
			// For other types, use string comparison
			if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
				issues = append(issues, fmt.Sprintf("sharp_edge_fields: %s got %v, want %v", field, actualValue, expectedValue))
			}
		}
	}

	return issues
}

// validateSessionStartExpectations handles sessionstart-specific validation.
func (r *DefaultRunner) validateSessionStartExpectations(expected ExpectedOutput, output string) []string {
	var issues []string

	// Skip validation if no SessionStart expectations are set
	if len(expected.AdditionalContextContains) == 0 &&
		len(expected.AdditionalContextNotContains) == 0 &&
		expected.ProjectTypeEquals == "" &&
		!expected.ToolCounterInitialized {
		return issues
	}

	// Parse output as JSON
	var outputJSON map[string]interface{}
	if err := json.Unmarshal([]byte(output), &outputJSON); err != nil {
		// If output isn't JSON, check raw content
		for _, substr := range expected.AdditionalContextContains {
			if !strings.Contains(output, substr) {
				issues = append(issues, fmt.Sprintf("additional_context_contains: %q not found", substr))
			}
		}
		return issues
	}

	// Extract additionalContext from hookSpecificOutput
	hookOutput, ok := outputJSON["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		issues = append(issues, "hookSpecificOutput missing from response")
		return issues
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		issues = append(issues, "additionalContext missing from hookSpecificOutput")
		return issues
	}

	// Check additional_context_contains
	for _, substr := range expected.AdditionalContextContains {
		if !strings.Contains(additionalContext, substr) {
			issues = append(issues, fmt.Sprintf("additional_context_contains: %q not found in context", substr))
		}
	}

	// Check additional_context_not_contains
	for _, substr := range expected.AdditionalContextNotContains {
		if strings.Contains(additionalContext, substr) {
			issues = append(issues, fmt.Sprintf("additional_context_not_contains: %q found in context (should be absent)", substr))
		}
	}

	// Check project type
	if expected.ProjectTypeEquals != "" {
		if !strings.Contains(additionalContext, "PROJECT TYPE: "+expected.ProjectTypeEquals) {
			issues = append(issues, fmt.Sprintf("project_type_equals: expected %q in context", expected.ProjectTypeEquals))
		}
	}

	// Check tool counter initialization
	if expected.ToolCounterInitialized {
		// Read tool counter from XDG path
		counterPath := filepath.Join(r.config.TempDir, ".cache", "gogent", "tool-counter")
		if _, err := os.Stat(counterPath); os.IsNotExist(err) {
			// Also check XDG_CACHE_HOME location
			xdgPath := os.Getenv("XDG_CACHE_HOME")
			if xdgPath != "" {
				counterPath = filepath.Join(xdgPath, "gogent", "tool-counter")
			}
			if _, err := os.Stat(counterPath); os.IsNotExist(err) {
				issues = append(issues, "tool_counter_initialized: counter file not found")
			}
		}
	}

	return issues
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

	// Add scenario-specific environment variables
	for key, value := range r.scenarioEnv {
		env = append(env, key+"="+value)
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

	// Load PostToolUse scenarios (sharp-edge detection)
	// Only load if sharp-edge binary path is configured
	if r.sharpEdgePath != "" {
		postToolDir := filepath.Join(r.config.FixturesDir, "deterministic", "posttooluse")
		if err := r.loadScenariosFromDir(postToolDir, "posttooluse", &scenarios); err != nil {
			return nil, err
		}
	}

	// Load SessionStart scenarios (load-context)
	// Only load if load-context binary path is configured
	if r.loadContextPath != "" {
		sessionStartDir := filepath.Join(r.config.FixturesDir, "deterministic", "sessionstart")
		if err := r.loadScenariosFromDir(sessionStartDir, "sessionstart", &scenarios); err != nil {
			return nil, err
		}
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
		// Store env vars for buildEnv to use
		if r.scenarioEnv == nil {
			r.scenarioEnv = make(map[string]string)
		}

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
			} else {
				// Store for buildEnv to propagate to CLI
				r.scenarioEnv[key] = expandedValue
			}
		}

		return nil
	}
}
