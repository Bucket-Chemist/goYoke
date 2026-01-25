package regression

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/test/integration"
)

// TestRegression_ValidateRouting compares Go vs Bash validate-routing output
func TestRegression_ValidateRouting(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests (from GOgent-000)")
	}

	goBinary := "../../cmd/gogent-validate/gogent-validate"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/validate-routing.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter PreToolUse events
	events := harness.FilterEvents("PreToolUse")

	if len(events) == 0 {
		t.Skip("No PreToolUse events in corpus")
	}

	t.Logf("Running regression test on %d PreToolUse events", len(events))

	passed := 0
	failed := 0
	differences := []string{}

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (session=%s): %s", i, event.SessionID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			if failed <= 5 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Regression Test Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Logf("\nFirst 5 differences:")
		for i, diff := range differences {
			if i >= 5 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Regression test failed: %d/%d events differ between Go and Bash", failed, len(events))
	}
}

// TestRegression_SessionArchive compares Go vs Bash session-archive output
func TestRegression_SessionArchive(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-archive/gogent-archive"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/session-archive.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	events := harness.FilterEvents("SessionEnd")

	if len(events) == 0 {
		t.Skip("No SessionEnd events in corpus")
	}

	t.Logf("Running regression test on %d SessionEnd events", len(events))

	for i, event := range events {
		// Setup session files for both runs
		setupSessionFilesForEvent(t, projectDir, event)

		// Run Go implementation
		goHandoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff-go.md")
		os.Setenv("GOgent_HANDOFF_PATH", goHandoffPath)
		_ = harness.RunHook(goBinary, event)
		os.Unsetenv("GOgent_HANDOFF_PATH")

		// Run Bash implementation
		bashHandoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff-bash.md")
		os.Setenv("GOgent_HANDOFF_PATH", bashHandoffPath)
		_ = runBashHook(t, bashScript, event, projectDir)
		os.Unsetenv("GOgent_HANDOFF_PATH")

		// Compare handoff files (ignore timestamps)
		if _, err := os.Stat(goHandoffPath); err != nil {
			t.Errorf("Event %d: Go handoff not created", i)
			continue
		}

		if _, err := os.Stat(bashHandoffPath); err != nil {
			t.Errorf("Event %d: Bash handoff not created", i)
			continue
		}

		goHandoff, _ := os.ReadFile(goHandoffPath)
		bashHandoff, _ := os.ReadFile(bashHandoffPath)

		// Strip timestamps for comparison
		goContent := stripTimestamps(string(goHandoff))
		bashContent := stripTimestamps(string(bashHandoff))

		if goContent != bashContent {
			t.Errorf("Event %d: Handoff content differs", i)
			t.Logf("  Go handoff length:   %d", len(goHandoff))
			t.Logf("  Bash handoff length: %d", len(bashHandoff))

			// Show first difference
			showFirstDifference(t, goContent, bashContent)
		}

		// Cleanup for next event
		os.Remove(goHandoffPath)
		os.Remove(bashHandoffPath)
	}
}

// TestRegression_SharpEdgeDetector compares Go vs Bash sharp-edge output
func TestRegression_SharpEdgeDetector(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/sharp-edge-detector.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	events := harness.FilterEvents("PostToolUse")

	if len(events) == 0 {
		t.Skip("No PostToolUse events in corpus")
	}

	t.Logf("Running regression test on %d PostToolUse events", len(events))

	passed := 0
	failed := 0

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			t.Logf("Event %d differs: %s", i, strings.Join(diffs, "; "))
		}
	}

	t.Logf("\n=== Sharp Edge Regression Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Errorf("Sharp edge regression failed: %d/%d events differ", failed, len(events))
	}
}

// TestRegression_LoadContext compares Go vs Bash context loading for SessionStart
func TestRegression_LoadContext(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-load-context/gogent-load-context"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/load-routing-context.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	// Setup previous handoff for context loading
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	os.WriteFile(handoffPath, []byte("# Previous Session\nRouting decisions: ...\n"), 0644)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter SessionStart events
	events := harness.FilterEvents("SessionStart")

	if len(events) == 0 {
		t.Skip("No SessionStart events in corpus")
	}

	t.Logf("Running regression test on %d SessionStart events", len(events))

	passed := 0
	failed := 0
	differences := []string{}

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (session=%s): %s", i, event.SessionID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			if failed <= 5 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Load Context Regression Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Logf("\nFirst 5 differences:")
		for i, diff := range differences {
			if i >= 5 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Load context regression failed: %d/%d events differ between Go and Bash", failed, len(events))
	}
}

// TestRegression_AgentEndstate compares Go vs Bash agent endstate processing for SubagentStop
func TestRegression_AgentEndstate(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-agent-endstate/gogent-agent-endstate"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/agent-endstate.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter SubagentStop events
	events := harness.FilterEvents("SubagentStop")

	if len(events) == 0 {
		t.Skip("No SubagentStop events in corpus")
	}

	t.Logf("Running regression test on %d SubagentStop events", len(events))

	passed := 0
	failed := 0
	differences := []string{}

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (agent=%s): %s", i, event.AgentID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			if failed <= 5 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Agent Endstate Regression Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Logf("\nFirst 5 differences:")
		for i, diff := range differences {
			if i >= 5 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Agent endstate regression failed: %d/%d events differ between Go and Bash", failed, len(events))
	}
}

// TestRegression_MLTelemetry validates ML telemetry export consistency for hook tracing
func TestRegression_MLTelemetry(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Collect all events for ML telemetry validation
	allEvents := harness.AllEvents()
	if len(allEvents) == 0 {
		t.Skip("No events in corpus")
	}

	t.Logf("Validating ML telemetry on %d events", len(allEvents))

	// Validate event structure consistency for ML pipeline
	eventTypes := make(map[string]int)
	requiresCorpus := 0
	sessionStartCount := 0
	sessionEndCount := 0
	subagentStopCount := 0

	for _, event := range allEvents {
		eventTypes[event.HookEventName]++

		// Track critical hook events
		if event.HookEventName == "SessionStart" {
			sessionStartCount++
		}

		if event.HookEventName == "SessionEnd" {
			sessionEndCount++
		}

		if event.HookEventName == "SubagentStop" {
			subagentStopCount++
		}

		// Count events requiring corpus
		if event.HookEventName == "PreToolUse" || event.HookEventName == "PostToolUse" ||
			event.HookEventName == "SessionStart" || event.HookEventName == "SessionEnd" ||
			event.HookEventName == "SubagentStop" {
			requiresCorpus++
		}
	}

	// Log telemetry distribution
	t.Logf("\n=== ML Telemetry Summary ===")
	t.Logf("Total events: %d", len(allEvents))
	t.Logf("SessionStart: %d", sessionStartCount)
	t.Logf("SessionEnd: %d", sessionEndCount)
	t.Logf("SubagentStop: %d", subagentStopCount)
	t.Logf("Events for ML pipeline: %d", requiresCorpus)

	t.Logf("\nEvent distribution:")
	for eventType, count := range eventTypes {
		t.Logf("  %s: %d", eventType, count)
	}

	// Verify critical event types present
	if sessionStartCount == 0 {
		t.Errorf("No SessionStart events found in corpus (required for hook testing)")
	}
	if sessionEndCount == 0 {
		t.Errorf("No SessionEnd events found in corpus (required for hook testing)")
	}
	if subagentStopCount == 0 {
		t.Errorf("No SubagentStop events found in corpus (required for agent endstate testing)")
	}

	// Corpus should have min 1 of each critical event
	minRequiredEvents := 3
	if requiresCorpus < minRequiredEvents {
		t.Errorf("Insufficient events for ML pipeline validation: got %d, need at least %d",
			requiresCorpus, minRequiredEvents)
	}
}

// Helper: Run Bash hook with event
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

// Helper: Setup regression test project
func setupRegressionProject(t *testing.T, projectDir string) {
	// Create routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	// Use actual routing schema from ~/.claude
	homeSchema := filepath.Join(os.Getenv("HOME"), ".claude", "routing-schema.json")
	if data, err := os.ReadFile(homeSchema); err == nil {
		os.WriteFile(schemaPath, data, 0644)
	} else {
		// Fallback to minimal schema
		schema := `{
			"tiers": {
				"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
				"sonnet": {"tools_allowed": ["*"]},
				"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
			},
			"agent_subagent_mapping": {}
		}`
		os.WriteFile(schemaPath, []byte(schema), 0644)
	}

	// Set tier
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)
}

// Helper: Setup session files for archive event
func setupSessionFilesForEvent(t *testing.T, projectDir string, event *integration.EventEntry) {
	// Create transcript
	transcriptPath, ok := event.ToolInput["transcript_path"].(string)
	if !ok {
		transcriptPath = filepath.Join(projectDir, ".claude", "transcript.jsonl")
	}

	os.MkdirAll(filepath.Dir(transcriptPath), 0755)
	os.WriteFile(transcriptPath, []byte(""), 0644)

	// Create minimal tool counter
	counterPath := filepath.Join(projectDir, ".gogent", "tool-counter-read")
	os.MkdirAll(filepath.Dir(counterPath), 0755)
	os.WriteFile(counterPath, []byte("x\n"), 0644)
}

// Helper: Strip timestamps from handoff content
func stripTimestamps(content string) string {
	lines := strings.Split(content, "\n")
	var filtered []string

	for _, line := range lines {
		// Skip lines with timestamps (e.g., "# Session Handoff - 2026-01-15 14:32:00")
		if strings.Contains(line, "202") && strings.Contains(line, ":") {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// Helper: Show first difference between two strings
func showFirstDifference(t *testing.T, a, b string) {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	maxLines := len(linesA)
	if len(linesB) > maxLines {
		maxLines = len(linesB)
	}

	for i := 0; i < maxLines; i++ {
		lineA := ""
		lineB := ""

		if i < len(linesA) {
			lineA = linesA[i]
		}

		if i < len(linesB) {
			lineB = linesB[i]
		}

		if lineA != lineB {
			t.Logf("  First diff at line %d:", i+1)
			t.Logf("    Go:   %s", lineA)
			t.Logf("    Bash: %s", lineB)
			return
		}
	}
}
