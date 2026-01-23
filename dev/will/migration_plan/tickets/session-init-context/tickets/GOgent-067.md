---
id: GOgent-067
title: Extend Runner with SessionStart Category
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: [  - GOgent-062]
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 11
---

## GOgent-067: Extend Runner with SessionStart Category

**Time**: 1.5 hours
**Dependencies**: GOgent-062
**Priority**: HIGH

**Task**:
Extend `DefaultRunner` to support `sessionstart` category execution against `gogent-load-context`.

**File**: `test/simulation/harness/runner.go` (modify existing)

**Implementation**:
```go
// Add to NewRunner constructor - after sharpEdgePath
type DefaultRunner struct {
	// ... existing fields
	loadContextPath string // NEW: path to gogent-load-context binary
}

// Add setter method
// SetLoadContextPath sets the path to gogent-load-context binary.
// Required for sessionstart scenario execution.
func (r *DefaultRunner) SetLoadContextPath(path string) {
	r.loadContextPath = path
}

// Modify executeScenario switch statement:
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
	case "sessionstart":  // NEW CASE
		if r.loadContextPath == "" {
			return "", -1, fmt.Errorf("sessionstart scenario requires loadContextPath (gogent-load-context binary)")
		}
		cmdPath = r.loadContextPath
	default:
		return "", -1, fmt.Errorf("unknown category: %s", s.Category)
	}
	// ... rest of function unchanged
}

// Modify loadScenarios to include sessionstart directory:
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

	// Load PostToolUse scenarios
	if r.sharpEdgePath != "" {
		postToolDir := filepath.Join(r.config.FixturesDir, "deterministic", "posttooluse")
		if err := r.loadScenariosFromDir(postToolDir, "posttooluse", &scenarios); err != nil {
			return nil, err
		}
	}

	// NEW: Load SessionStart scenarios
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
```

**Add to harness/types.go ExpectedOutput struct**:
```go
// SessionStart-specific expectations
type ExpectedOutput struct {
	// ... existing fields

	// SessionStart-specific expectations
	AdditionalContextContains    []string `json:"additional_context_contains,omitempty"`
	AdditionalContextNotContains []string `json:"additional_context_not_contains,omitempty"`
	ProjectTypeEquals            string   `json:"project_type_equals,omitempty"`
	ToolCounterInitialized       bool     `json:"tool_counter_initialized,omitempty"`
}
```

**Add validation method**:
```go
// validateSessionStartExpectations handles sessionstart-specific validation.
func (r *DefaultRunner) validateSessionStartExpectations(expected ExpectedOutput, output string) []string {
	var issues []string

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
```

**Update validateOutput to call new method**:
```go
func (r *DefaultRunner) validateOutput(expected ExpectedOutput, output string, exitCode int) (bool, string, string) {
	// ... existing code

	// PostToolUse-specific validations
	issues = append(issues, r.validatePostToolUseExpectations(expected, output)...)

	// SessionStart-specific validations (NEW)
	issues = append(issues, r.validateSessionStartExpectations(expected, output)...)

	// ... rest unchanged
}
```

**Tests**: Add to `test/simulation/harness/runner_test.go`

```go
func TestRunner_SessionStartCategory(t *testing.T) {
	// Create temp dirs and fixtures
	tmpDir := t.TempDir()
	fixturesDir := filepath.Join(tmpDir, "fixtures", "deterministic", "sessionstart")
	os.MkdirAll(fixturesDir, 0755)

	// Create a simple test fixture
	fixture := `{
		"input": {
			"type": "startup",
			"session_id": "test-001",
			"hook_event_name": "SessionStart"
		},
		"expected": {
			"exit_code": 0,
			"additional_context_contains": ["SESSION INITIALIZED"]
		}
	}`
	os.WriteFile(filepath.Join(fixturesDir, "startup-basic.json"), []byte(fixture), 0644)

	// Build mock binary that echoes valid response
	// (In real tests, use actual binary)
	mockBinary := createMockLoadContextBinary(t, tmpDir)

	cfg := SimulationConfig{
		Mode:        "deterministic",
		FixturesDir: filepath.Join(tmpDir, "fixtures"),
		TempDir:     tmpDir,
		Verbose:     true,
	}

	runner := NewRunner(cfg, "", "", nil)
	runner.SetLoadContextPath(mockBinary)

	results, err := runner.Run(cfg)
	if err != nil {
		t.Fatalf("Runner.Run failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}
}

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
```

**Acceptance Criteria**:
- [ ] `SetLoadContextPath()` method added to runner
- [ ] `sessionstart` category handled in `executeScenario()`
- [ ] `sessionstart` directory loaded in `loadScenarios()`
- [ ] `ExpectedOutput` extended with SessionStart-specific fields
- [ ] `validateSessionStartExpectations()` validates context content
- [ ] Tests verify category handling and validation
- [ ] `go test ./test/simulation/harness/...` passes

**Test Deliverables**:
- [ ] Tests added to: `test/simulation/harness/runner_test.go`
- [ ] Number of new test functions: 2
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Runner extension enables all downstream simulation modes to test SessionStart hook.

---
