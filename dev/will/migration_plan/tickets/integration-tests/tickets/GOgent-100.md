---
id: GOgent-100
title: Regression Tests (Go vs Bash Comparison)
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-000","GOgent-094"]
priority: high
week: 5
tags: ["performance", "week-5"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-100: Regression Tests (Go vs Bash Comparison)

**Time**: 2 hours
**Dependencies**: GOgent-000 (corpus), GOgent-094 (harness), all hook binaries

**Task**:
Run 100-event corpus through both Go and Bash implementations, verify identical output (except timestamps).

**File**: `test/regression/regression_test.go`

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

	"github.com/yourusername/gogent-fortress/test/integration"
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
		goResult := harness.RunHook(goBinary, event)
		os.Unsetenv("GOgent_HANDOFF_PATH")

		// Run Bash implementation
		bashHandoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff-bash.md")
		os.Setenv("GOgent_HANDOFF_PATH", bashHandoffPath)
		bashResult := runBashHook(t, bashScript, event, projectDir)
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
		bashHandoff, _ := os.ReadFile(bashHandoff)

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
```

**Run regression tests**:
```bash
export GOgent_CORPUS_PATH=/path/to/corpus/from/gogent-000.jsonl
go test ./test/regression -v
```

**Acceptance Criteria**:
- [ ] `TestRegression_ValidateRouting` compares all PreToolUse events
- [ ] `TestRegression_SessionArchive` compares all SessionEnd events
- [ ] `TestRegression_SharpEdgeDetector` compares all PostToolUse events
- [ ] ≥95% of events produce identical output (Go vs Bash)
- [ ] Differences limited to timestamp formatting
- [ ] Test report shows pass/fail counts and first 5 differences
- [ ] Regression tests pass: `go test ./test/regression -v`
- [ ] Results documented in regression-report.md

**Why This Matters**: Regression tests are the final quality gate. Must verify Go implementations are drop-in replacements for Bash with no behavior changes.

---

## Summary

Week 3 Part 1 completes the Phase 0 testing suite with:

- **GOgent-004c**: Config circular dependency tests (deferred from Week 1)
- **GOgent-094**: Test harness for corpus replay (foundation for all tests)
- **GOgent-095**: Integration tests for validate-routing hook
- **GOgent-096**: Integration tests for session-archive hook
- **GOgent-097**: Integration tests for sharp-edge-detector hook
- **GOgent-098**: Performance benchmarks (<5ms p99, <10MB memory targets)
- **GOgent-099**: End-to-end workflow integration tests
- **GOgent-100**: Regression tests comparing Go vs Bash output

**Quality Gates**:
- Integration tests verify complete hook workflows
- Performance benchmarks enforce latency and memory targets
- Regression tests ensure Go output matches Bash exactly
- End-to-end tests validate cross-hook pipelines

**Next**: [07-week3-deployment-cutover.md](07-week3-deployment-cutover.md) - Installation, parallel testing, and GO/NO-GO cutover decision (GOgent-101 to 055)

---

**File Status**: ✅ Complete
**Tickets**: 7 (GOgent-004c, 094-100)
**Detail Level**: Full implementation + comprehensive tests
