package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRunner_MatchesFilter(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{
			ScenarioFilter: []string{"V00", "S001"},
		},
	}

	tests := []struct {
		id       string
		expected bool
	}{
		{"V001", true},
		{"V002", true},
		{"S001", true},
		{"S002", false},
		{"X001", false},
	}

	for _, tt := range tests {
		if got := r.matchesFilter(tt.id); got != tt.expected {
			t.Errorf("matchesFilter(%s) = %v, want %v", tt.id, got, tt.expected)
		}
	}
}

func TestRunner_MatchesFilter_NoFilter(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{
			ScenarioFilter: []string{},
		},
	}

	// Empty filter should match everything
	if !r.matchesFilter("anything") {
		t.Error("Expected empty filter to match all scenarios")
	}
}

func TestRunner_ValidateOutput_Decision(t *testing.T) {
	r := &DefaultRunner{}

	decision := "block"
	expected := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	output := `{"decision": "block", "reason": "opus blocked"}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if !passed {
		t.Errorf("Expected validation to pass, got diff: %s", diff)
	}
}

func TestRunner_ValidateOutput_DecisionMismatch(t *testing.T) {
	r := &DefaultRunner{}

	decision := "block"
	expected := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	output := `{"decision": "allow", "reason": "haiku allowed"}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if passed {
		t.Error("Expected validation to fail on decision mismatch")
	}
	if !strings.Contains(diff, "decision") {
		t.Errorf("Expected diff to mention decision, got: %s", diff)
	}
}

func TestRunner_ValidateOutput_ExitCodeMismatch(t *testing.T) {
	r := &DefaultRunner{}

	expected := ExpectedOutput{
		ExitCode: 0,
	}

	passed, _, diff := r.validateOutput(expected, "{}", 1)

	if passed {
		t.Error("Expected validation to fail on exit code mismatch")
	}
	if !strings.Contains(diff, "exit code") {
		t.Errorf("Expected diff to mention exit code, got: %s", diff)
	}
}

func TestRunner_ValidateOutput_ReasonPattern(t *testing.T) {
	r := &DefaultRunner{}

	expected := ExpectedOutput{
		ReasonPattern: "blocked.*opus",
		ExitCode:      0,
	}

	output := `{"decision": "block", "reason": "blocked due to opus tier"}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if !passed {
		t.Errorf("Expected validation to pass, got diff: %s", diff)
	}
}

func TestRunner_ValidateOutput_ReasonPatternMismatch(t *testing.T) {
	r := &DefaultRunner{}

	pattern := regexp.MustCompile("something-else")
	expected := ExpectedOutput{
		ReasonMatch: pattern,
		ExitCode:    0,
	}

	output := `{"decision": "block", "reason": "blocked due to opus tier"}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if passed {
		t.Error("Expected validation to fail on reason pattern mismatch")
	}
	if diff == "" {
		t.Error("Expected diff to be non-empty")
	}
}

func TestRunner_ValidateOutput_StderrPattern(t *testing.T) {
	r := &DefaultRunner{}

	expected := ExpectedOutput{
		StderrPattern: "warning",
		ExitCode:      0,
	}

	output := "some output\n[STDERR]\nwarning: deprecated usage"
	passed, _, diff := r.validateOutput(expected, output, 0)

	if !passed {
		t.Errorf("Expected validation to pass, got diff: %s", diff)
	}
}

func TestRunner_ValidateOutput_MultipleIssues(t *testing.T) {
	r := &DefaultRunner{}

	decision := "block"
	expected := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	output := `{"decision": "allow"}`
	passed, _, diff := r.validateOutput(expected, output, 1)

	if passed {
		t.Error("Expected validation to fail with multiple issues")
	}
	// Should report both exit code and decision issues
	if !strings.Contains(diff, "exit code") || !strings.Contains(diff, "decision") {
		t.Errorf("Expected diff to mention both issues, got: %s", diff)
	}
}

func TestRunner_BuildEnv(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{
			SchemaPath: "/test/schema.json",
			AgentsPath: "/test/agents.json",
			TempDir:    "/tmp/sim",
		},
	}

	env := r.buildEnv()

	hasSchema := false
	hasAgents := false
	hasProject := false

	for _, e := range env {
		if e == "GOGENT_ROUTING_SCHEMA=/test/schema.json" {
			hasSchema = true
		}
		if e == "GOGENT_AGENTS_INDEX=/test/agents.json" {
			hasAgents = true
		}
		if e == "GOGENT_PROJECT_DIR=/tmp/sim" {
			hasProject = true
		}
	}

	if !hasSchema {
		t.Error("Expected GOGENT_ROUTING_SCHEMA to be set")
	}
	if !hasAgents {
		t.Error("Expected GOGENT_AGENTS_INDEX to be set")
	}
	if !hasProject {
		t.Error("Expected GOGENT_PROJECT_DIR to be set")
	}
}

func TestRunner_BuildEnv_EmptyPaths(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{},
	}

	env := r.buildEnv()

	// Should still return environment, just without GOGENT_ overrides
	if len(env) == 0 {
		t.Error("Expected environment to be non-empty")
	}
}

func TestSimulationResult_Duration(t *testing.T) {
	result := SimulationResult{
		ScenarioID: "test",
		Duration:   100 * time.Millisecond,
	}

	if result.Duration < 100*time.Millisecond {
		t.Errorf("Duration should be >= 100ms, got: %v", result.Duration)
	}
}

func TestRunner_LoadScenariosFromDir_NonexistentDir(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir: "/nonexistent/path",
		},
	}

	var scenarios []Scenario
	err := r.loadScenariosFromDir("/nonexistent/path", "pretooluse", &scenarios)

	// Should not error on nonexistent directory
	if err != nil {
		t.Errorf("Expected no error for nonexistent directory, got: %v", err)
	}
	if len(scenarios) != 0 {
		t.Errorf("Expected no scenarios loaded, got %d", len(scenarios))
	}
}

func TestRunner_LoadScenariosFromDir_ValidFixtures(t *testing.T) {
	// Create temporary test fixtures
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Write test fixture
	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"tool_name": "Task",
			"session_id": "test-session",
		},
		"expected": map[string]interface{}{
			"decision":  "allow",
			"exit_code": 0,
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(fixtureDir, "test-scenario.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Load scenarios
	r := &DefaultRunner{}
	var scenarios []Scenario
	err := r.loadScenariosFromDir(fixtureDir, "pretooluse", &scenarios)

	if err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].ID != "test-scenario" {
		t.Errorf("Expected scenario ID 'test-scenario', got %s", scenarios[0].ID)
	}
	if scenarios[0].Category != "pretooluse" {
		t.Errorf("Expected category 'pretooluse', got %s", scenarios[0].Category)
	}
}

func TestRunner_ExecuteScenario_UnknownCategory(t *testing.T) {
	r := &DefaultRunner{
		validatePath: "/bin/true",
	}

	scenario := Scenario{
		Category: "unknown",
		Input:    map[string]string{},
	}

	_, _, err := r.executeScenario(scenario)
	if err == nil {
		t.Error("Expected error for unknown category")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("Expected 'unknown category' error, got: %v", err)
	}
}

func TestRunner_ExecuteScenario_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode (takes 30s)")
	}

	// Create a mock script that hangs
	tmpDir := t.TempDir()
	hangScript := filepath.Join(tmpDir, "hang.sh")
	scriptContent := "#!/bin/bash\nsleep 60\n"
	if err := os.WriteFile(hangScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create hang script: %v", err)
	}

	r := &DefaultRunner{
		validatePath: hangScript,
	}

	scenario := Scenario{
		Category: "pretooluse",
		Input:    map[string]string{"test": "data"},
	}

	// executeScenario has a 30s timeout internally
	// We expect it to timeout and return an error
	start := time.Now()
	_, _, err := r.executeScenario(scenario)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("Expected context deadline error, got: %v", err)
	}
	// Should timeout at 30s, not wait for full 60s
	if duration > 35*time.Second {
		t.Errorf("Timeout took too long: %v (expected ~30s)", duration)
	}
	if duration < 29*time.Second {
		t.Errorf("Timeout happened too quickly: %v (expected ~30s)", duration)
	}
}

func TestRunner_RunScenario_SetupFailure(t *testing.T) {
	r := &DefaultRunner{
		validatePath: "/bin/true",
	}

	setupCalled := false
	scenario := Scenario{
		ID:       "test",
		Category: "pretooluse",
		Input:    map[string]string{},
		Setup: func(cfg SimulationConfig) error {
			setupCalled = true
			return fmt.Errorf("setup failed")
		},
	}

	result := r.RunScenario(scenario)

	if !setupCalled {
		t.Error("Expected setup to be called")
	}
	if result.Error == nil {
		t.Error("Expected error from setup failure")
	}
	if !strings.Contains(result.ErrorMsg, "setup failed") {
		t.Errorf("Expected 'setup failed' in error message, got: %s", result.ErrorMsg)
	}
	if result.Passed {
		t.Error("Expected scenario to fail")
	}
}

func TestRunner_RunScenario_TeardownFailure(t *testing.T) {
	// Create a mock CLI that succeeds
	tmpDir := t.TempDir()
	successScript := filepath.Join(tmpDir, "success.sh")
	scriptContent := "#!/bin/bash\necho '{\"decision\": \"allow\"}'\nexit 0\n"
	if err := os.WriteFile(successScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create success script: %v", err)
	}

	r := &DefaultRunner{
		validatePath: successScript,
		config:       SimulationConfig{Verbose: false},
	}

	teardownCalled := false
	scenario := Scenario{
		ID:       "test",
		Category: "pretooluse",
		Input:    map[string]string{"tool_name": "Read"},
		Expected: ExpectedOutput{ExitCode: 0},
		Teardown: func(cfg SimulationConfig) error {
			teardownCalled = true
			return fmt.Errorf("teardown failed")
		},
	}

	result := r.RunScenario(scenario)

	if !teardownCalled {
		t.Error("Expected teardown to be called")
	}
	// Teardown failures should not fail the test
	if !result.Passed {
		t.Error("Expected scenario to pass despite teardown failure")
	}
}

func TestRunner_RunScenario_LifecycleOrder(t *testing.T) {
	// Create a mock CLI that succeeds
	tmpDir := t.TempDir()
	successScript := filepath.Join(tmpDir, "success.sh")
	scriptContent := "#!/bin/bash\necho '{\"decision\": \"allow\"}'\nexit 0\n"
	if err := os.WriteFile(successScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create success script: %v", err)
	}

	r := &DefaultRunner{
		validatePath: successScript,
	}

	var order []string
	scenario := Scenario{
		ID:       "test",
		Category: "pretooluse",
		Input:    map[string]string{"tool_name": "Read"},
		Expected: ExpectedOutput{ExitCode: 0},
		Setup: func(cfg SimulationConfig) error {
			order = append(order, "setup")
			return nil
		},
		Teardown: func(cfg SimulationConfig) error {
			order = append(order, "teardown")
			return nil
		},
	}

	result := r.RunScenario(scenario)

	if len(order) != 2 {
		t.Errorf("Expected 2 lifecycle calls, got %d", len(order))
	}
	if order[0] != "setup" {
		t.Errorf("Expected setup first, got %s", order[0])
	}
	if order[1] != "teardown" {
		t.Errorf("Expected teardown last, got %s", order[1])
	}
	if !result.Passed {
		t.Error("Expected scenario to pass")
	}
}

func TestRunner_Run_Integration(t *testing.T) {
	// Create temporary fixture directory
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create mock CLI
	cliPath := filepath.Join(tmpDir, "mock-validate")
	cliContent := "#!/bin/bash\ncat\nexit 0\n" // Echo STDIN to STDOUT
	if err := os.WriteFile(cliPath, []byte(cliContent), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	// Create test fixtures
	fixtures := []struct {
		id       string
		decision string
	}{
		{"V001-test", "allow"},
		{"V002-test", "block"},
		{"S001-test", "allow"},
	}

	for _, f := range fixtures {
		fixture := map[string]interface{}{
			"input": map[string]interface{}{
				"tool_name":  "Task",
				"session_id": "test-session",
			},
			"expected": map[string]interface{}{
				"exit_code": 0,
			},
		}
		fixtureBytes, _ := json.Marshal(fixture)
		fixturePath := filepath.Join(fixtureDir, f.id+".json")
		if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
			t.Fatalf("Failed to write fixture: %v", err)
		}
	}

	// Run simulation
	cfg := SimulationConfig{
		TempDir:        tmpDir,
		FixturesDir:    filepath.Join(tmpDir, "fixtures"),
		ScenarioFilter: []string{"V00"}, // Only V00* scenarios
		Verbose:        false,
	}

	r := NewRunner(cfg, cliPath, cliPath, nil)
	results, err := r.Run(cfg)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should only run V001 and V002 (filter excludes S001)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if !strings.HasPrefix(result.ScenarioID, "V00") {
			t.Errorf("Unexpected scenario ID: %s", result.ScenarioID)
		}
	}
}

func TestRunner_ExecuteScenario_RealCLI(t *testing.T) {
	// Check if gogent-validate exists (skip test if not available)
	validatePath, err := exec.LookPath("gogent-validate")
	if err != nil {
		t.Skip("gogent-validate not found in PATH, skipping real CLI test")
	}

	r := &DefaultRunner{
		validatePath: validatePath,
	}

	// Test with valid PreToolUse input
	scenario := Scenario{
		Category: "pretooluse",
		Input: map[string]interface{}{
			"tool_name":       "Read",
			"session_id":      "test-session",
			"hook_event_name": "PreToolUse",
			"captured_at":     time.Now().Unix(),
			"tool_input": map[string]interface{}{
				"file_path": "/tmp/test.txt",
			},
		},
	}

	output, exitCode, err := r.executeScenario(scenario)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Output should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.Split(output, "\n[STDERR]")[0]), &result); err != nil {
		t.Errorf("Expected valid JSON output, got parse error: %v", err)
	}
}

func TestRunner_LoadScenarios_InvalidJSON(t *testing.T) {
	// Create temporary fixture with invalid JSON
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Write invalid JSON
	fixturePath := filepath.Join(fixtureDir, "invalid.json")
	if err := os.WriteFile(fixturePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir:     tmpDir,
			FixturesDir: filepath.Join(tmpDir, "fixtures"),
		},
	}

	_, err := r.loadScenarios()
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestRunner_LoadScenarios_ReadError(t *testing.T) {
	// Create directory as a file to cause read error
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create a directory with no read permissions
	noReadDir := filepath.Join(fixtureDir, "subdir")
	if err := os.Mkdir(noReadDir, 0755); err != nil {
		t.Fatalf("Failed to create no-read dir: %v", err)
	}
	// Create a file inside that we'll make unreadable
	badFile := filepath.Join(fixtureDir, "test.json")
	if err := os.WriteFile(badFile, []byte("{}"), 0000); err != nil {
		t.Fatalf("Failed to write unreadable file: %v", err)
	}
	defer os.Chmod(badFile, 0644) // Clean up

	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir:     tmpDir,
			FixturesDir: filepath.Join(tmpDir, "fixtures"),
		},
	}

	_, err := r.loadScenarios()
	if err == nil {
		t.Error("Expected error for unreadable file")
	}
}

func TestRunner_ExecuteScenario_InvalidInputJSON(t *testing.T) {
	r := &DefaultRunner{
		validatePath: "/bin/true",
	}

	// Use a circular reference to force JSON marshal error
	type circular struct {
		Self interface{}
	}
	c := &circular{}
	c.Self = c

	scenario := Scenario{
		Category: "pretooluse",
		Input:    c,
	}

	_, _, err := r.executeScenario(scenario)
	if err == nil {
		t.Error("Expected error for unmarshalable input")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("Expected marshal error, got: %v", err)
	}
}

func TestRunner_ExecuteScenario_SessionEnd(t *testing.T) {
	// Create a mock archive CLI
	tmpDir := t.TempDir()
	archiveScript := filepath.Join(tmpDir, "mock-archive")
	scriptContent := "#!/bin/bash\necho '{\"status\": \"archived\"}'\nexit 0\n"
	if err := os.WriteFile(archiveScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock archive CLI: %v", err)
	}

	r := &DefaultRunner{
		archivePath: archiveScript,
	}

	scenario := Scenario{
		Category: "sessionend",
		Input: map[string]interface{}{
			"session_id":      "test-session",
			"hook_event_name": "SessionEnd",
		},
	}

	output, exitCode, err := r.executeScenario(scenario)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "archived") {
		t.Errorf("Expected output to contain 'archived', got: %s", output)
	}
}

func TestRunner_ValidateOutput_DecisionMissing(t *testing.T) {
	r := &DefaultRunner{}

	decision := "block"
	expected := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	// Output without decision field
	output := `{"reason": "some reason"}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if passed {
		t.Error("Expected validation to fail when decision field missing")
	}
	if !strings.Contains(diff, "decision field missing") {
		t.Errorf("Expected diff to mention missing decision, got: %s", diff)
	}
}

func TestRunner_ValidateOutput_EmptyDecision(t *testing.T) {
	r := &DefaultRunner{}

	decision := ""
	expected := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	// Empty decision should not cause missing field error
	output := `{}`
	passed, _, diff := r.validateOutput(expected, output, 0)

	if !passed {
		t.Errorf("Expected validation to pass for empty decision, got diff: %s", diff)
	}
}

func TestRunner_ValidateOutput_InvalidJSON(t *testing.T) {
	r := &DefaultRunner{}

	// Test with stderr pattern on non-JSON output
	expected := ExpectedOutput{
		StderrPattern: "warning",
		ExitCode:      0,
	}

	// Non-JSON output should still validate against stderr patterns
	output := "some output with warning message"
	passed, _, _ := r.validateOutput(expected, output, 0)

	if !passed {
		t.Error("Expected validation to pass when stderr pattern matches even with non-JSON")
	}

	// Test with no JSON and decision expectation
	decision := "allow"
	expected2 := ExpectedOutput{
		Decision: &decision,
		ExitCode: 0,
	}

	output2 := "not json at all"
	passed2, _, _ := r.validateOutput(expected2, output2, 0)

	// Should pass because JSON parsing failed so decision check is skipped
	if !passed2 {
		t.Error("Expected validation to pass when JSON parsing fails (no JSON checks performed)")
	}
}

func TestRunner_Run_NoScenarios(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := SimulationConfig{
		TempDir: tmpDir,
		Verbose: false,
	}

	r := NewRunner(cfg, "/bin/true", "/bin/true", nil)
	results, err := r.Run(cfg)

	if err != nil {
		t.Errorf("Expected no error for empty scenario set, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestRunner_Run_Verbose(t *testing.T) {
	// Create temporary fixtures
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create mock CLI that succeeds
	cliPath := filepath.Join(tmpDir, "mock-validate")
	cliContent := "#!/bin/bash\necho '{\"decision\": \"allow\"}'\nexit 0\n"
	if err := os.WriteFile(cliPath, []byte(cliContent), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	// Create fixture
	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"tool_name": "Read",
		},
		"expected": map[string]interface{}{
			"exit_code": 0,
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(fixtureDir, "test.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	cfg := SimulationConfig{
		TempDir:     tmpDir,
		FixturesDir: filepath.Join(tmpDir, "fixtures"),
		Verbose:     true, // Enable verbose output
	}

	r := NewRunner(cfg, cliPath, cliPath, nil)
	results, err := r.Run(cfg)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestRunner_RunScenario_InputSerialization(t *testing.T) {
	// Create a mock CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "mock-validate")
	cliContent := "#!/bin/bash\necho '{\"decision\": \"allow\"}'\nexit 0\n"
	if err := os.WriteFile(cliPath, []byte(cliContent), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	r := &DefaultRunner{
		validatePath: cliPath,
	}

	scenario := Scenario{
		ID:       "test",
		Category: "pretooluse",
		Input: map[string]string{
			"key": "value",
		},
		Expected: ExpectedOutput{ExitCode: 0},
	}

	result := r.RunScenario(scenario)

	if result.Input == "" {
		t.Error("Expected input to be serialized in result")
	}

	var inputParsed map[string]string
	if err := json.Unmarshal([]byte(result.Input), &inputParsed); err != nil {
		t.Errorf("Failed to parse result input JSON: %v", err)
	}
	if inputParsed["key"] != "value" {
		t.Errorf("Expected input key=value, got: %v", inputParsed)
	}
}

func TestRunner_ExecuteScenario_NonZeroExit(t *testing.T) {
	// Create a mock CLI that exits with non-zero
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "mock-validate")
	cliContent := "#!/bin/bash\necho '{\"decision\": \"block\"}'\nexit 1\n"
	if err := os.WriteFile(cliPath, []byte(cliContent), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	r := &DefaultRunner{
		validatePath: cliPath,
	}

	scenario := Scenario{
		Category: "pretooluse",
		Input:    map[string]string{},
	}

	output, exitCode, err := r.executeScenario(scenario)

	// Non-zero exit should not be an error
	if err != nil {
		t.Errorf("Expected no error for non-zero exit, got: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	if output == "" {
		t.Error("Expected output despite non-zero exit")
	}
}

func TestRunner_ExecuteScenario_WithStderr(t *testing.T) {
	// Create a mock CLI that writes to both stdout and stderr
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "mock-validate")
	cliContent := "#!/bin/bash\necho '{\"decision\": \"allow\"}'\necho 'debug output' >&2\nexit 0\n"
	if err := os.WriteFile(cliPath, []byte(cliContent), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	r := &DefaultRunner{
		validatePath: cliPath,
	}

	scenario := Scenario{
		Category: "pretooluse",
		Input:    map[string]string{},
	}

	output, exitCode, err := r.executeScenario(scenario)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	// Output should contain both stdout and stderr
	if !strings.Contains(output, "decision") {
		t.Error("Expected stdout in output")
	}
	if !strings.Contains(output, "[STDERR]") {
		t.Error("Expected stderr section in output")
	}
	if !strings.Contains(output, "debug output") {
		t.Error("Expected stderr content in output")
	}
}

func TestRunner_LoadScenarios_SessionEnd(t *testing.T) {
	// Test loading sessionend scenarios
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "sessionend")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"session_id": "test-session",
		},
		"expected": map[string]interface{}{
			"exit_code": 0,
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(fixtureDir, "session-test.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir:     tmpDir,
			FixturesDir: filepath.Join(tmpDir, "fixtures"),
		},
	}

	scenarios, err := r.loadScenarios()
	if err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Category != "sessionend" {
		t.Errorf("Expected category 'sessionend', got %s", scenarios[0].Category)
	}
}

func TestRunner_Run_LoadError(t *testing.T) {
	// Create a runner with invalid fixture path
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "pretooluse")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Write invalid JSON
	badFixture := filepath.Join(fixtureDir, "bad.json")
	if err := os.WriteFile(badFixture, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write bad fixture: %v", err)
	}

	cfg := SimulationConfig{
		TempDir:     tmpDir,
		FixturesDir: filepath.Join(tmpDir, "fixtures"),
	}

	r := NewRunner(cfg, "/bin/true", "/bin/true", nil)
	_, err := r.Run(cfg)

	if err == nil {
		t.Error("Expected error from load scenarios")
	}
}

func TestRunner_ValidateOutput_ReasonPattern_String(t *testing.T) {
	r := &DefaultRunner{}

	// Test ReasonPattern (string) vs ReasonMatch (regexp)
	expected := ExpectedOutput{
		ReasonPattern: "opus",
		ExitCode:      0,
	}

	output := `{"decision": "block", "reason": "blocked opus tier"}`
	passed, _, _ := r.validateOutput(expected, output, 0)

	if !passed {
		t.Error("Expected validation to pass when reason pattern matches")
	}
}

func TestRunner_ValidateOutput_AllFields(t *testing.T) {
	r := &DefaultRunner{}

	// Test that combines multiple validation types
	decision := "block"
	expected := ExpectedOutput{
		Decision:      &decision,
		ReasonPattern: "opus.*blocked",
		ExitCode:      0,
	}

	// Pure JSON output (no stderr) so JSON parsing succeeds
	output := `{"decision": "block", "reason": "opus tier blocked"}`

	passed, expectedStr, diff := r.validateOutput(expected, output, 0)

	if !passed {
		t.Errorf("Expected validation to pass when all fields match, got diff: %s", diff)
	}
	if !strings.Contains(expectedStr, "exit_code=0") {
		t.Errorf("Expected exit_code in expected string, got: %s", expectedStr)
	}
	if !strings.Contains(expectedStr, "decision=block") {
		t.Errorf("Expected decision in expected string, got: %s", expectedStr)
	}
}

// TestRunner_SessionStartCategory tests sessionstart scenario execution
func TestRunner_SessionStartCategory(t *testing.T) {
	// Create temp dirs and fixtures
	tmpDir := t.TempDir()
	fixturesDir := filepath.Join(tmpDir, "fixtures", "deterministic", "sessionstart")
	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		t.Fatalf("Failed to create fixtures dir: %v", err)
	}

	// Create a simple test fixture
	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"type":            "startup",
			"session_id":      "test-001",
			"hook_event_name": "SessionStart",
		},
		"expected": map[string]interface{}{
			"exit_code":                    0,
			"additional_context_contains":  []string{"SESSION INITIALIZED"},
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	if err := os.WriteFile(filepath.Join(fixturesDir, "startup-basic.json"), fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Build mock binary that echoes valid response
	mockBinary := createMockLoadContextBinary(t, tmpDir)

	cfg := SimulationConfig{
		Mode:        "deterministic",
		FixturesDir: filepath.Join(tmpDir, "fixtures"),
		TempDir:     tmpDir,
		Verbose:     false,
	}

	runner := NewRunner(cfg, "/bin/true", "/bin/true", nil)
	runner.SetLoadContextPath(mockBinary)

	results, err := runner.Run(cfg)
	if err != nil {
		t.Fatalf("Runner.Run failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Verify scenario was executed
	result := results[0]
	if result.ScenarioID != "startup-basic" {
		t.Errorf("Expected scenario ID 'startup-basic', got %s", result.ScenarioID)
	}
	if !result.Passed {
		t.Errorf("Expected scenario to pass, got diff: %s", result.Diff)
	}
}

// TestRunner_SessionStartCategory_NoPath tests error when loadContextPath not set
func TestRunner_SessionStartCategory_NoPath(t *testing.T) {
	r := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	scenario := Scenario{
		Category: "sessionstart",
		Input:    map[string]string{},
	}

	_, _, err := r.executeScenario(scenario)
	if err == nil {
		t.Fatal("Expected error when loadContextPath not set")
	}
	if !strings.Contains(err.Error(), "loadContextPath") {
		t.Errorf("Expected error to mention loadContextPath, got: %v", err)
	}
}

// TestValidateSessionStartExpectations tests SessionStart validation logic
func TestValidateSessionStartExpectations(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "🚀 SESSION INITIALIZED (startup)\n\nPROJECT TYPE: go\n\nRouting hooks are ACTIVE."
		}
	}`

	expected := ExpectedOutput{
		AdditionalContextContains:    []string{"SESSION INITIALIZED", "go"},
		AdditionalContextNotContains: []string{"ERROR"},
		ProjectTypeEquals:            "go",
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) > 0 {
		t.Errorf("Unexpected validation issues: %v", issues)
	}
}

// TestValidateSessionStartExpectations_ContextContainsFails tests substring not found
func TestValidateSessionStartExpectations_ContextContainsFails(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "SESSION INITIALIZED"
		}
	}`

	expected := ExpectedOutput{
		AdditionalContextContains: []string{"MISSING_STRING"},
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when substring not found")
	}
	if !strings.Contains(issues[0], "additional_context_contains") {
		t.Errorf("Expected issue to mention additional_context_contains, got: %s", issues[0])
	}
}

// TestValidateSessionStartExpectations_ContextNotContainsFails tests negative check
func TestValidateSessionStartExpectations_ContextNotContainsFails(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "SESSION INITIALIZED with ERROR"
		}
	}`

	expected := ExpectedOutput{
		AdditionalContextNotContains: []string{"ERROR"},
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when forbidden substring found")
	}
	if !strings.Contains(issues[0], "additional_context_not_contains") {
		t.Errorf("Expected issue to mention additional_context_not_contains, got: %s", issues[0])
	}
}

// TestValidateSessionStartExpectations_ProjectTypeFails tests project type validation
func TestValidateSessionStartExpectations_ProjectTypeFails(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "PROJECT TYPE: python"
		}
	}`

	expected := ExpectedOutput{
		ProjectTypeEquals: "go",
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when project type doesn't match")
	}
	if !strings.Contains(issues[0], "project_type_equals") {
		t.Errorf("Expected issue to mention project_type_equals, got: %s", issues[0])
	}
}

// TestValidateSessionStartExpectations_InvalidJSON tests non-JSON output
func TestValidateSessionStartExpectations_InvalidJSON(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := "not valid json"

	expected := ExpectedOutput{
		AdditionalContextContains: []string{"SESSION"},
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	// Should fall back to raw content check
	if len(issues) == 0 {
		t.Error("Expected validation to fail when substring not in raw output")
	}
}

// TestValidateSessionStartExpectations_MissingHookOutput tests missing hookSpecificOutput
func TestValidateSessionStartExpectations_MissingHookOutput(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{"some": "data"}`

	expected := ExpectedOutput{
		AdditionalContextContains: []string{"SESSION"},
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when hookSpecificOutput missing")
	}
	if !strings.Contains(issues[0], "hookSpecificOutput missing") {
		t.Errorf("Expected issue about missing hookSpecificOutput, got: %s", issues[0])
	}
}

// TestValidateSessionStartExpectations_MissingAdditionalContext tests missing additionalContext field
func TestValidateSessionStartExpectations_MissingAdditionalContext(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{"hookSpecificOutput": {"hookEventName": "SessionStart"}}`

	expected := ExpectedOutput{
		AdditionalContextContains: []string{"SESSION"},
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when additionalContext missing")
	}
	if !strings.Contains(issues[0], "additionalContext missing") {
		t.Errorf("Expected issue about missing additionalContext, got: %s", issues[0])
	}
}

// TestValidateSessionStartExpectations_ToolCounterInitialized tests counter file check
func TestValidateSessionStartExpectations_ToolCounterInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: tmpDir},
	}

	// Create the tool counter file
	counterPath := filepath.Join(tmpDir, ".cache", "gogent", "tool-counter")
	if err := os.MkdirAll(filepath.Dir(counterPath), 0755); err != nil {
		t.Fatalf("Failed to create counter dir: %v", err)
	}
	if err := os.WriteFile(counterPath, []byte("0"), 0644); err != nil {
		t.Fatalf("Failed to write counter file: %v", err)
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "SESSION INITIALIZED"
		}
	}`

	expected := ExpectedOutput{
		ToolCounterInitialized: true,
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) > 0 {
		t.Errorf("Expected validation to pass when counter file exists, got: %v", issues)
	}
}

// TestValidateSessionStartExpectations_ToolCounterMissing tests counter file missing
func TestValidateSessionStartExpectations_ToolCounterMissing(t *testing.T) {
	runner := &DefaultRunner{
		config: SimulationConfig{TempDir: t.TempDir()},
	}

	output := `{
		"hookSpecificOutput": {
			"hookEventName": "SessionStart",
			"additionalContext": "SESSION INITIALIZED"
		}
	}`

	expected := ExpectedOutput{
		ToolCounterInitialized: true,
	}

	issues := runner.validateSessionStartExpectations(expected, output)

	if len(issues) == 0 {
		t.Error("Expected validation to fail when counter file missing")
	}
	if !strings.Contains(issues[0], "tool_counter_initialized") {
		t.Errorf("Expected issue about tool_counter_initialized, got: %s", issues[0])
	}
}

// TestRunner_LoadScenarios_SessionStart tests loading sessionstart scenarios
func TestRunner_LoadScenarios_SessionStart(t *testing.T) {
	// Create temporary fixture directory
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "sessionstart")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create test fixture
	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"session_id": "test-session",
		},
		"expected": map[string]interface{}{
			"exit_code": 0,
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(fixtureDir, "session-test.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Create runner with loadContextPath set
	mockBinary := createMockLoadContextBinary(t, tmpDir)

	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir:     tmpDir,
			FixturesDir: filepath.Join(tmpDir, "fixtures"),
		},
		loadContextPath: mockBinary,
	}

	scenarios, err := r.loadScenarios()
	if err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Category != "sessionstart" {
		t.Errorf("Expected category 'sessionstart', got %s", scenarios[0].Category)
	}
}

// TestRunner_LoadScenarios_SessionStartNotLoaded tests scenarios not loaded when path empty
func TestRunner_LoadScenarios_SessionStartNotLoaded(t *testing.T) {
	// Create temporary fixture directory
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "fixtures", "deterministic", "sessionstart")
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create test fixture
	fixture := map[string]interface{}{
		"input": map[string]interface{}{
			"session_id": "test-session",
		},
		"expected": map[string]interface{}{
			"exit_code": 0,
		},
	}
	fixtureBytes, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(fixtureDir, "session-test.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Create runner WITHOUT loadContextPath set
	r := &DefaultRunner{
		config: SimulationConfig{
			TempDir:     tmpDir,
			FixturesDir: filepath.Join(tmpDir, "fixtures"),
		},
		// loadContextPath is empty string
	}

	scenarios, err := r.loadScenarios()
	if err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	// Should not load sessionstart scenarios when path is empty
	if len(scenarios) != 0 {
		t.Errorf("Expected 0 scenarios when loadContextPath empty, got %d", len(scenarios))
	}
}

// createMockLoadContextBinary creates a mock binary for testing
func createMockLoadContextBinary(t *testing.T, tmpDir string) string {
	mockBinary := filepath.Join(tmpDir, "mock-load-context")
	scriptContent := `#!/bin/bash
cat << 'EOF'
{
	"hookSpecificOutput": {
		"hookEventName": "SessionStart",
		"additionalContext": "🚀 SESSION INITIALIZED (startup)\n\nPROJECT TYPE: go\n\nRouting hooks are ACTIVE."
	}
}
EOF
exit 0
`
	if err := os.WriteFile(mockBinary, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}
	return mockBinary
}
