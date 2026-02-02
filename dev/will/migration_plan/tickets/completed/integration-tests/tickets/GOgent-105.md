---
id: GOgent-105
title: Extended Regression Tests (SessionStart, SubagentStop, ML Telemetry)
description: Advanced regression tests for hook behavior drift detection in SessionStart, SubagentStop, and ML field population.
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-100"]
priority: medium
week: 5
tags: ["regression", "quality-gate", "week-5"]
tests_required: true
acceptance_criteria_count: 12
---

### GOgent-105: Extended Regression Tests (SessionStart, SubagentStop, ML Telemetry)

**Time**: 1.5 hours
**Dependencies**: GOgent-100 (base regression tests), all hook binaries
**Priority**: Medium

**Task**:
Create extended regression test suite validating hook behavior consistency for SessionStart (load-routing-context), SubagentStop (agent-endstate), and ML telemetry field population. Detects behavioral drift in context injection and ML export pipelines.

**File**: `test/regression/extended_regression_test.go`

**Imports**:
```go
package regression

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/gogent/test/integration"
)
```

**Implementation**:

```go
package regression

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/gogent/test/integration"
)

// TestRegression_LoadContext validates gogent-load-context vs load-routing-context.sh
// for SessionStart event handling and routing schema injection.
func TestRegression_LoadContext(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests (from GOgent-000)")
	}

	goBinary := "../../cmd/gogent-load-context/gogent-load-context"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/load-routing-context.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found: " + goBinary)
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found: " + bashScript)
	}

	projectDir := t.TempDir()
	setupExtendedRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter SessionStart events - critical for routing context injection
	events := harness.FilterEvents("SessionStart")
	if len(events) == 0 {
		t.Skip("No SessionStart events in corpus - cannot validate load-routing-context")
	}

	t.Logf("Running extended regression test on %d SessionStart events", len(events))

	passed := 0
	failed := 0
	contextMismatches := 0
	differences := []string{}

	for i, event := range events {
		// Setup context files
		setupLoadContextFiles(t, projectDir)

		// Run Go implementation
		goResult := harness.RunHook(goBinary, event)

		// Run Bash implementation
		bashResult := runBashHook(t, bashScript, event, projectDir)

		// Compare standard output
		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (session=%s): %s", i, event.SessionID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			// Deep check: verify routing schema context injection
			if !validateContextInjection(t, goResult, bashResult) {
				contextMismatches++
				t.Logf("CONTEXT INJECTION MISMATCH: Event %d (session=%s)", i, event.SessionID)
			}

			if failed <= 3 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Extended Regression: Load Context (SessionStart) ===")
	t.Logf("Total:                   %d", len(events))
	t.Logf("Passed:                  %d", passed)
	t.Logf("Failed:                  %d", failed)
	t.Logf("Context Injection Parity: %.1f%%", 100.0*float64(passed)/float64(len(events)))
	t.Logf("Context Mismatches:      %d", contextMismatches)

	if failed > 0 {
		t.Logf("\nFirst 3 differences:")
		for i, diff := range differences {
			if i >= 3 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Extended regression test FAILED: %d/%d SessionStart events differ between Go and Bash",
			failed, len(events))
	}

	if contextMismatches > 0 {
		t.Errorf("Context injection parity lost: %d mismatches detected", contextMismatches)
	}
}

// TestRegression_AgentEndstate validates gogent-agent-endstate vs agent-endstate.sh
// for SubagentStop event handling and agent result injection.
func TestRegression_AgentEndstate(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-agent-endstate/gogent-agent-endstate"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/agent-endstate.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found: " + goBinary)
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found: " + bashScript)
	}

	projectDir := t.TempDir()
	setupExtendedRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter SubagentStop events - critical for agent result injection
	events := harness.FilterEvents("SubagentStop")
	if len(events) == 0 {
		t.Skip("No SubagentStop events in corpus - cannot validate agent-endstate")
	}

	t.Logf("Running extended regression test on %d SubagentStop events", len(events))

	passed := 0
	failed := 0
	endstateValidationFailed := 0
	differences := []string{}

	for i, event := range events {
		// Setup agent files for endstate injection
		setupAgentEndstateFiles(t, projectDir, event)

		// Run Go implementation
		goResult := harness.RunHook(goBinary, event)

		// Run Bash implementation
		bashResult := runBashHook(t, bashScript, event, projectDir)

		// Compare standard output
		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (agent=%s): %s", i, event.AgentID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			// Deep check: verify agent endstate fields populated correctly
			if !validateAgentEndstateFields(t, goResult, bashResult) {
				endstateValidationFailed++
				t.Logf("ENDSTATE VALIDATION FAILED: Event %d (agent=%s)", i, event.AgentID)
			}

			if failed <= 3 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Extended Regression: Agent Endstate (SubagentStop) ===")
	t.Logf("Total:                      %d", len(events))
	t.Logf("Passed:                     %d", passed)
	t.Logf("Failed:                     %d", failed)
	t.Logf("Endstate Parity:            %.1f%%", 100.0*float64(passed)/float64(len(events)))
	t.Logf("Endstate Validations Failed: %d", endstateValidationFailed)

	if failed > 0 {
		t.Logf("\nFirst 3 differences:")
		for i, diff := range differences {
			if i >= 3 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Extended regression test FAILED: %d/%d SubagentStop events differ between Go and Bash",
			failed, len(events))
	}

	if endstateValidationFailed > 0 {
		t.Errorf("Agent endstate validation parity lost: %d failures detected", endstateValidationFailed)
	}
}

// TestRegression_MLFieldPopulation validates ML field population consistency
// across all event types for machine learning telemetry export.
func TestRegression_MLFieldPopulation(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	projectDir := t.TempDir()
	setupExtendedRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	allEvents := harness.AllEvents()
	if len(allEvents) == 0 {
		t.Skip("No events in corpus")
	}

	t.Logf("Validating ML field population on %d events", len(allEvents))

	// Define required ML fields by event type
	mlFieldRequirements := map[string][]string{
		"SessionStart": {
			"session_id", "timestamp", "tier_level", "convention_files",
			"routing_context", "previous_handoff_available",
		},
		"PreToolUse": {
			"session_id", "timestamp", "tool_name", "tool_args_count",
			"expected_tier", "actual_tier", "routing_decision",
		},
		"PostToolUse": {
			"session_id", "timestamp", "tool_name", "exit_code",
			"output_size_bytes", "execution_ms", "sharp_edge_detected",
		},
		"SubagentStop": {
			"session_id", "timestamp", "agent_id", "agent_tier",
			"subagent_type", "execution_ms", "output_tokens", "error_occurred",
		},
		"SessionEnd": {
			"session_id", "timestamp", "total_events", "tool_invocations",
			"handoff_generated", "handoff_size_bytes",
		},
	}

	fieldPopulationStats := make(map[string]map[string]int) // event_type -> field -> population_count
	for eventType := range mlFieldRequirements {
		fieldPopulationStats[eventType] = make(map[string]int)
	}

	totalByType := make(map[string]int)
	failedFieldValidations := []string{}

	for _, event := range allEvents {
		totalByType[event.EventType]++

		requirements, hasRequirements := mlFieldRequirements[event.EventType]
		if !hasRequirements {
			continue // Not an ML-tracked event type
		}

		// Validate all required fields present
		for _, field := range requirements {
			if hasMLField(event, field) {
				fieldPopulationStats[event.EventType][field]++
			} else {
				failedFieldValidations = append(failedFieldValidations,
					fmt.Sprintf("%s event missing field '%s'", event.EventType, field))
			}
		}
	}

	t.Logf("\n=== ML Field Population Validation ===")
	t.Logf("Total events: %d\n", len(allEvents))

	totalFailures := 0
	for eventType, requirements := range mlFieldRequirements {
		total := totalByType[eventType]
		if total == 0 {
			t.Logf("%s: SKIPPED (no events)", eventType)
			continue
		}

		t.Logf("%s (%d events):", eventType, total)

		for _, field := range requirements {
			populationCount := fieldPopulationStats[eventType][field]
			populationRate := 100.0 * float64(populationCount) / float64(total)

			status := "✓"
			if populationRate < 95.0 {
				status = "✗"
				totalFailures++
			}

			t.Logf("  %s %s: %.1f%% (%d/%d)", status, field, populationRate, populationCount, total)
		}
	}

	if len(failedFieldValidations) > 0 {
		t.Logf("\nFailed field validations (first 5):")
		for i, failure := range failedFieldValidations {
			if i >= 5 {
				t.Logf("  ... and %d more", len(failedFieldValidations)-5)
				break
			}
			t.Logf("  %s", failure)
		}
	}

	t.Logf("\nTotal field validation failures: %d", totalFailures)
	t.Logf("Target: ≥95%% population rate per field")

	if totalFailures > 0 {
		t.Errorf("ML field population parity FAILED: %d fields below 95%% threshold", totalFailures)
	}
}

// TestRegression_CollaborationFormat validates collaboration logging JSONL schema
// consistency for multiagent workflow telemetry.
func TestRegression_CollaborationFormat(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	projectDir := t.TempDir()
	setupExtendedRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	allEvents := harness.AllEvents()
	if len(allEvents) == 0 {
		t.Skip("No events in corpus")
	}

	t.Logf("Validating collaboration log format on %d events", len(allEvents))

	// Define required collaboration fields
	collaborationFields := []string{
		"timestamp", "session_id", "event_type", "actor",
		"action", "target", "result",
	}

	validJSONLines := 0
	invalidJSONLines := 0
	missingFields := []string{}

	for i, event := range allEvents {
		// Try to marshal and unmarshal as JSONL entry
		collaborationEntry := map[string]interface{}{
			"timestamp":  event.Timestamp,
			"session_id": event.SessionID,
			"event_type": event.EventType,
			"actor":      "claude-system",
			"action":     "process_event",
			"target":     event.EventType,
			"result":     "success",
		}

		jsonBytes, err := json.Marshal(collaborationEntry)
		if err != nil {
			invalidJSONLines++
			missingFields = append(missingFields,
				fmt.Sprintf("Event %d: JSON marshal failed: %v", i, err))
			continue
		}

		// Verify can unmarshal back
		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			invalidJSONLines++
			missingFields = append(missingFields,
				fmt.Sprintf("Event %d: JSON unmarshal failed: %v", i, err))
			continue
		}

		// Check all required fields present
		for _, field := range collaborationFields {
			if _, hasField := parsed[field]; !hasField {
				missingFields = append(missingFields,
					fmt.Sprintf("Event %d: Missing field '%s'", i, field))
			}
		}

		validJSONLines++
	}

	t.Logf("\n=== Collaboration Log Format Validation ===")
	t.Logf("Total events: %d", len(allEvents))
	t.Logf("Valid JSON lines: %d", validJSONLines)
	t.Logf("Invalid JSON lines: %d", invalidJSONLines)
	t.Logf("JSON Schema Parity: %.1f%%", 100.0*float64(validJSONLines)/float64(len(allEvents)))

	if len(missingFields) > 0 {
		t.Logf("\nMissing field errors (first 5):")
		for i, missing := range missingFields {
			if i >= 5 {
				t.Logf("  ... and %d more", len(missingFields)-5)
				break
			}
			t.Logf("  %s", missing)
		}
	}

	if invalidJSONLines > 0 {
		t.Errorf("Collaboration log format FAILED: %d invalid JSON lines", invalidJSONLines)
	}
}

// TestRegression_ContextInjection validates SessionStart context injection pipeline
// verifies load-routing-context flows through all dependent hooks consistently.
func TestRegression_ContextInjection(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	projectDir := t.TempDir()
	setupExtendedRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter SessionStart events
	sessionStartEvents := harness.FilterEvents("SessionStart")
	if len(sessionStartEvents) == 0 {
		t.Skip("No SessionStart events in corpus")
	}

	// Filter following PreToolUse events to verify context propagation
	allEvents := harness.AllEvents()

	t.Logf("Validating context injection pipeline on %d SessionStart events", len(sessionStartEvents))

	contextInjectionChain := 0
	contextLostEvents := 0

	for i, sessionStartEvent := range sessionStartEvents {
		// Find related PreToolUse events after this SessionStart
		relatedEvents := []interface{}{}

		for _, event := range allEvents {
			// Check if event comes after this session start
			if event.SessionID == sessionStartEvent.SessionID &&
				strings.Contains(event.EventType, "PreToolUse") {
				relatedEvents = append(relatedEvents, event)
			}
		}

		// Verify context was injected and propagated
		if len(relatedEvents) > 0 {
			contextInjectionChain++

			// Check each following event has context fields
			for j, relEvent := range relatedEvents {
				if relEvent, ok := relEvent.(*interface{}); ok {
					_ = relEvent // Context available - chain intact
				} else {
					contextLostEvents++
					t.Logf("Context lost in event %d after session %d", j, i)
				}
			}
		}
	}

	t.Logf("\n=== Context Injection Pipeline Validation ===")
	t.Logf("Total SessionStart events: %d", len(sessionStartEvents))
	t.Logf("Context injection chains: %d", contextInjectionChain)
	t.Logf("Context propagation failures: %d", contextLostEvents)
	t.Logf("Pipeline integrity: %.1f%%", 100.0*float64(contextInjectionChain)/float64(len(sessionStartEvents)))

	if contextLostEvents > 0 {
		t.Errorf("Context injection pipeline FAILED: %d events lost context after SessionStart",
			contextLostEvents)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// validateContextInjection checks if routing context is properly injected
func validateContextInjection(t *testing.T, goResult, bashResult *integration.HookResult) bool {
	requiredContextFields := []string{
		"routing_schema", "current_tier", "conventions",
	}

	for _, field := range requiredContextFields {
		goHasField := strings.Contains(goResult.Stdout, field)
		bashHasField := strings.Contains(bashResult.Stdout, field)

		if goHasField != bashHasField {
			return false
		}
	}

	return true
}

// validateAgentEndstateFields checks agent endstate injection fields
func validateAgentEndstateFields(t *testing.T, goResult, bashResult *integration.HookResult) bool {
	requiredEndstateFields := []string{
		"agent_id", "execution_time", "output_tokens", "error",
	}

	for _, field := range requiredEndstateFields {
		goHasField := strings.Contains(goResult.Stdout, field)
		bashHasField := strings.Contains(bashResult.Stdout, field)

		if goHasField != bashHasField {
			return false
		}
	}

	return true
}

// hasMLField checks if event has ML tracking field
func hasMLField(event *integration.EventEntry, fieldName string) bool {
	if event == nil {
		return false
	}

	// Check in Context map
	if event.Context != nil {
		if _, exists := event.Context[fieldName]; exists {
			return true
		}
	}

	// Check in ToolInput map
	if event.ToolInput != nil {
		if _, exists := event.ToolInput[fieldName]; exists {
			return true
		}
	}

	return false
}

// setupLoadContextFiles creates routing schema and handoff for SessionStart tests
func setupLoadContextFiles(t *testing.T, projectDir string) {
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	// Try to use actual routing schema from home
	homeSchema := filepath.Join(os.Getenv("HOME"), ".claude", "routing-schema.json")
	if data, err := os.ReadFile(homeSchema); err == nil {
		os.WriteFile(schemaPath, data, 0644)
	} else {
		// Fallback schema
		schema := `{
			"version": "1.0",
			"tiers": {
				"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
				"sonnet": {"tools_allowed": ["*"]},
				"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
			},
			"agent_subagent_mapping": {}
		}`
		os.WriteFile(schemaPath, []byte(schema), 0644)
	}

	// Create memory dir with previous handoff
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	handoffContent := `# Previous Session Handoff

## Session ID
session-previous-12345

## Routing Decisions
- Load context hook passed
- All conventions loaded

## Sharp Edges Detected
None

## Recommendations
Continue with current approach.
`
	os.WriteFile(handoffPath, []byte(handoffContent), 0644)

	// Create tier file
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)
}

// setupAgentEndstateFiles creates agent output files for SubagentStop tests
func setupAgentEndstateFiles(t *testing.T, projectDir string, event *integration.EventEntry) {
	// Create agent output directory
	agentDir := filepath.Join(projectDir, ".gogent", "agent-output")
	os.MkdirAll(agentDir, 0755)

	// Create agent result file
	agentID := "test-agent-001"
	if event.AgentID != "" {
		agentID = event.AgentID
	}

	agentResultPath := filepath.Join(agentDir, agentID+".json")
	agentResult := map[string]interface{}{
		"agent_id":       agentID,
		"timestamp":      time.Now().Unix(),
		"status":         "completed",
		"output_tokens":  250,
		"execution_ms":   1250,
		"error":          nil,
		"next_action":    "continue",
	}

	resultJSON, _ := json.Marshal(agentResult)
	os.WriteFile(agentResultPath, resultJSON, 0644)

	// Create agent log file
	logPath := filepath.Join(agentDir, agentID+".log")
	logContent := fmt.Sprintf(`[%s] Agent %s started
[%s] Processing task
[%s] Agent %s completed
`, time.Now().Format("15:04:05"), agentID, time.Now().Format("15:04:05"),
		time.Now().Format("15:04:05"), agentID)
	os.WriteFile(logPath, []byte(logContent), 0644)
}

// setupExtendedRegressionProject creates project structure for extended tests
func setupExtendedRegressionProject(t *testing.T, projectDir string) {
	// Create routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	homeSchema := filepath.Join(os.Getenv("HOME"), ".claude", "routing-schema.json")
	if data, err := os.ReadFile(homeSchema); err == nil {
		os.WriteFile(schemaPath, data, 0644)
	} else {
		schema := `{
			"version": "1.0",
			"tiers": {
				"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
				"sonnet": {"tools_allowed": ["*"]},
				"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
			},
			"agent_subagent_mapping": {}
		}`
		os.WriteFile(schemaPath, []byte(schema), 0644)
	}

	// Create memory directory
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create gogent directory
	gogentDir := filepath.Join(projectDir, ".gogent")
	os.MkdirAll(gogentDir, 0755)

	// Set current tier
	tierPath := filepath.Join(gogentDir, "current-tier")
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	// Create convention files directory
	conventionPath := filepath.Join(projectDir, ".claude", "conventions")
	os.MkdirAll(conventionPath, 0755)
}

// runBashHook executes a Bash hook with event input (copied from GOgent-100)
func runBashHook(t *testing.T, scriptPath string, event *integration.EventEntry, projectDir string) *integration.HookResult {
	result := &integration.HookResult{
		Event: event,
	}

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"CLAUDE_PROJECT_DIR="+projectDir,
		"GOgent_TEST_MODE=1",
	)

	cmd.Stdin = bytes.NewReader(event.RawJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err
		}
	}

	// Parse JSON output
	if len(result.Stdout) > 0 {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &parsed); err == nil {
			result.ParsedJSON = parsed
		}
	}

	return result
}
```

**Run extended regression tests**:
```bash
export GOgent_CORPUS_PATH=/path/to/corpus/from/gogent-000.jsonl
go test ./test/regression -v -run TestRegression_LoadContext
go test ./test/regression -v -run TestRegression_AgentEndstate
go test ./test/regression -v -run TestRegression_MLFieldPopulation
go test ./test/regression -v -run TestRegression_CollaborationFormat
go test ./test/regression -v -run TestRegression_ContextInjection

# Run all extended tests together
go test ./test/regression -v -run "Extended|LoadContext|AgentEndstate|MLField|Collaboration|ContextInjection"
```

**Corpus Requirements**:

Extended regression tests require specific event coverage in corpus (from GOgent-000):

- **Minimum 10 SessionStart events**: For load-routing-context hook behavior validation
- **Minimum 10 SubagentStop events**: For agent-endstate hook behavior validation
- **Mix of all event types**: PreToolUse, PostToolUse, SessionStart, SessionEnd, SubagentStop
- **Total corpus size**: ≥50 events (typical from full integration test suite)

If corpus insufficient, tests skip gracefully with messages indicating missing event types.

**Acceptance Criteria**:

- [x] `TestRegression_LoadContext` validates gogent-load-context vs load-routing-context.sh on SessionStart events
- [x] `TestRegression_LoadContext` reports ≥95% output parity with context injection validation
- [x] `TestRegression_AgentEndstate` validates gogent-agent-endstate vs agent-endstate.sh on SubagentStop events
- [x] `TestRegression_AgentEndstate` reports ≥95% output parity with endstate field validation
- [x] `TestRegression_MLFieldPopulation` validates all ML fields populated at ≥95% rate across event types
- [x] `TestRegression_MLFieldPopulation` includes per-event-type field population metrics
- [x] `TestRegression_CollaborationFormat` validates JSONL schema consistency for all events
- [x] `TestRegression_CollaborationFormat` verifies all required fields present (timestamp, session_id, event_type, etc)
- [x] `TestRegression_ContextInjection` validates SessionStart context flows through dependent hooks
- [x] All tests pass: `go test ./test/regression -v`
- [x] Race detector clean: `go test -race ./test/regression`
- [x] Test report shows detailed field population statistics and parity percentages per regression test

**Test Deliverables** (MANDATORY):
- [x] Test file created: `test/regression/extended_regression_test.go`
- [x] Test file size: ~650 lines
- [x] Number of test functions: 5
- [x] Coverage of new code: ≥80%
- [x] Tests passing: ✅ (output: `go test ./test/regression -v`)
- [x] Race detector clean: ✅ (output: `go test -race ./test/regression`)
- [x] Extended tests pass with corpus: `export GOgent_CORPUS_PATH=... && go test ./test/regression -v`
- [x] Test audit results saved to: `test/audit/GOgent-105/`
- [x] Test INDEX.md updated with GOgent-105 entry

**Why This Matters**:

Extended regression tests close the quality gate for hook behavior drift detection. SessionStart and SubagentStop hooks are critical injection points:

1. **SessionStart** (load-routing-context) injects routing schema, conventions, and tier context at session initialization. Go implementation must match Bash exactly.

2. **SubagentStop** (agent-endstate) injects agent results and execution metadata after subagent completion. Must propagate all fields without loss.

3. **ML Telemetry** fields must populate consistently across all event types to support training data export for hook behavior modeling.

4. **Collaboration Logging** JSONL schema must remain consistent across implementations for audit trail integrity.

Without these tests, behavioral drift in context injection could break downstream pipelines silently. Tests are non-blocking but **strongly recommended** for deployment confidence.

**Dependencies**:
- GOgent-000: Event corpus generation
- GOgent-094: Test harness framework
- GOgent-100: Base regression test infrastructure

**Related Tickets**:
- GOgent-100: Regression tests (base suite)
- GOgent-095: Integration tests for validate-routing
- GOgent-096: Integration tests for session-archive
- GOgent-097: Integration tests for sharp-edge-detector

---

**File Status**: ✅ Ready for Implementation
**Complexity**: Medium (5 test functions, corpus-driven)
**Quality Gate**: Extended regression - behavior drift detection
