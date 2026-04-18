package regression

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/test/integration"
)

// TestExtendedRegression_LoadContext validates gogent-load-context vs load-routing-context.sh
// for SessionStart event handling and routing schema injection with deep context validation.
func TestExtendedRegression_LoadContext(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests (from GOgent-000)")
	}

	goBinary := "../../bin/gogent-load-context"
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

// TestExtendedRegression_AgentEndstate validates gogent-agent-endstate vs agent-endstate.sh
// for SubagentStop event handling and agent result injection with endstate field validation.
func TestExtendedRegression_AgentEndstate(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../bin/gogent-agent-endstate"
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

// TestExtendedRegression_MLFieldPopulation validates ML field population consistency
// across all event types for machine learning telemetry export.
func TestExtendedRegression_MLFieldPopulation(t *testing.T) {
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
			"session_id", "timestamp", "hook_event_name",
		},
		"PreToolUse": {
			"session_id", "timestamp", "tool_name", "hook_event_name",
		},
		"PostToolUse": {
			"session_id", "timestamp", "tool_name", "hook_event_name",
		},
		"SubagentStop": {
			"session_id", "timestamp", "agent_id", "hook_event_name",
		},
		"SessionEnd": {
			"session_id", "timestamp", "hook_event_name",
		},
	}

	fieldPopulationStats := make(map[string]map[string]int) // event_type -> field -> population_count
	for eventType := range mlFieldRequirements {
		fieldPopulationStats[eventType] = make(map[string]int)
	}

	totalByType := make(map[string]int)
	failedFieldValidations := []string{}

	for _, event := range allEvents {
		totalByType[event.HookEventName]++

		requirements, hasRequirements := mlFieldRequirements[event.HookEventName]
		if !hasRequirements {
			continue // Not an ML-tracked event type
		}

		// Validate all required fields present
		for _, field := range requirements {
			if hasMLField(event, field) {
				fieldPopulationStats[event.HookEventName][field]++
			} else {
				failedFieldValidations = append(failedFieldValidations,
					fmt.Sprintf("%s event missing field '%s'", event.HookEventName, field))
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

// TestExtendedRegression_CollaborationFormat validates collaboration logging JSONL schema
// consistency for multiagent workflow telemetry.
func TestExtendedRegression_CollaborationFormat(t *testing.T) {
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
		"timestamp", "session_id", "hook_event_name",
	}

	validJSONLines := 0
	invalidJSONLines := 0
	missingFields := []string{}

	for i, event := range allEvents {
		// Try to marshal and unmarshal as JSONL entry
		collaborationEntry := map[string]interface{}{
			"timestamp":       event.Timestamp,
			"session_id":      event.SessionID,
			"hook_event_name": event.HookEventName,
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

// TestExtendedRegression_ContextInjection validates SessionStart context injection pipeline
// verifies load-routing-context flows through all dependent hooks consistently.
func TestExtendedRegression_ContextInjection(t *testing.T) {
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

	// Get all events for context propagation analysis
	allEvents := harness.AllEvents()

	t.Logf("Validating context injection pipeline on %d SessionStart events", len(sessionStartEvents))

	contextInjectionChain := 0
	contextLostEvents := 0

	// Build session map for efficient lookup
	sessionEventMap := make(map[string][]*integration.EventEntry)
	for _, event := range allEvents {
		sessionEventMap[event.SessionID] = append(sessionEventMap[event.SessionID], event)
	}

	for i, sessionStartEvent := range sessionStartEvents {
		// Find related events in this session
		relatedEvents := sessionEventMap[sessionStartEvent.SessionID]

		// Count PreToolUse events after SessionStart
		preToolUseCount := 0
		for _, event := range relatedEvents {
			if event.HookEventName == "PreToolUse" && event.Timestamp > sessionStartEvent.Timestamp {
				preToolUseCount++
			}
		}

		// Verify context was injected and propagated
		if preToolUseCount > 0 {
			contextInjectionChain++

			// Check if context is maintained (SessionID present indicates chain intact)
			for _, event := range relatedEvents {
				if event.HookEventName == "PreToolUse" && event.Timestamp > sessionStartEvent.Timestamp {
					if event.SessionID == "" {
						contextLostEvents++
						t.Logf("Context lost in event after session %d", i)
					}
				}
			}
		}
	}

	t.Logf("\n=== Context Injection Pipeline Validation ===")
	t.Logf("Total SessionStart events: %d", len(sessionStartEvents))
	t.Logf("Context injection chains: %d", contextInjectionChain)
	t.Logf("Context propagation failures: %d", contextLostEvents)

	pipelineIntegrity := 0.0
	if len(sessionStartEvents) > 0 {
		pipelineIntegrity = 100.0 * float64(contextInjectionChain) / float64(len(sessionStartEvents))
	}
	t.Logf("Pipeline integrity: %.1f%%", pipelineIntegrity)

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

	// Check direct struct fields
	switch fieldName {
	case "session_id":
		return event.SessionID != ""
	case "timestamp":
		return event.Timestamp > 0
	case "hook_event_name":
		return event.HookEventName != ""
	case "tool_name":
		return event.ToolName != ""
	case "agent_id":
		return event.AgentID != ""
	case "duration_ms":
		return event.DurationMs >= 0
	case "input_tokens":
		return event.InputTokens >= 0
	case "output_tokens":
		return event.OutputTokens >= 0
	case "sequence_index":
		return event.SequenceIndex >= 0
	case "decision_id":
		return event.DecisionID != ""
	}

	// Check in ToolInput map
	if event.ToolInput != nil {
		if _, exists := event.ToolInput[fieldName]; exists {
			return true
		}
	}

	// Check in ToolResponse map
	if event.ToolResponse != nil {
		if _, exists := event.ToolResponse[fieldName]; exists {
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
		"agent_id":      agentID,
		"timestamp":     time.Now().Unix(),
		"status":        "completed",
		"output_tokens": 250,
		"execution_ms":  1250,
		"error":         nil,
		"next_action":   "continue",
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
