---
id: GOgent-095
title: Integration Tests for validate-routing Hook
description: "Test complete validate-routing workflow using corpus events, verify blocking logic and violation logging"
status: pending
time_estimate: 2h
dependencies: ["GOgent-094","GOgent-025"]
priority: high
week: 5
tags: ["integration-tests", "week-5"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-095: Integration Tests for validate-routing Hook

**Time**: 2 hours
**Dependencies**: GOgent-094 (harness), GOgent-025 (gogent-validate binary)

**Task**:
Test complete validate-routing workflow using corpus events, verify blocking logic and violation logging.

**File**: `test/integration/validate_routing_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/config"
)

func TestValidateRouting_Integration(t *testing.T) {
	// Setup: Build binary if not present
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found. Run: go build -o cmd/gogent-validate/gogent-validate cmd/gogent-validate/main.go")
	}

	// Setup: Create test corpus
	corpusPath := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	createValidateCorpus(t, corpusPath)

	// Setup: Create test project directory with routing schema
	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create harness
	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run all PreToolUse events through validate-routing
	results, err := harness.RunHookBatch(binaryPath, "PreToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No results returned")
	}

	// Verify results
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Hook execution error: %v", result.Error)
			continue
		}

		// All hooks must return valid JSON
		if result.ParsedJSON == nil {
			t.Errorf("Expected JSON output, got: %s", result.Stdout)
			continue
		}

		// Verify decision field present
		if _, ok := result.ParsedJSON["decision"]; !ok {
			t.Errorf("Missing 'decision' field in output: %v", result.ParsedJSON)
		}
	}

	// Print summary
	PrintSummary(results)
}

func TestValidateRouting_BlocksOpusTask(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create event attempting Task(opus)
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze this problem",
			"subagent_type": "general-purpose"
		},
		"session_id": "test-opus-block"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "opus-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify blocked
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	decision, ok := result.ParsedJSON["decision"].(string)
	if !ok || decision != "block" {
		t.Errorf("Expected decision=block for Task(opus), got: %v", decision)
	}

	// Verify reason mentions GAP document
	reason, ok := result.ParsedJSON["reason"].(string)
	if !ok || !strings.Contains(reason, "GAP") {
		t.Errorf("Expected reason to mention GAP document, got: %s", reason)
	}
}

func TestValidateRouting_AllowsValidTask(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create valid Task(haiku) event
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "haiku",
			"prompt": "AGENT: codebase-search\n\nFind auth files",
			"subagent_type": "Explore"
		},
		"session_id": "test-valid-task"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "valid-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify allowed
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	decision, ok := result.ParsedJSON["decision"].(string)
	if !ok || decision != "allow" {
		t.Errorf("Expected decision=allow for valid Task(haiku), got: %v", decision)
	}
}

func TestValidateRouting_ViolationLogging(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Override violations log path for test
	violationsLog := filepath.Join(t.TempDir(), "violations.jsonl")
	os.Setenv("GOgent_VIOLATIONS_LOG", violationsLog)
	defer os.Unsetenv("GOgent_VIOLATIONS_LOG")

	// Create event that violates tool permissions
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {
			"file_path": "/tmp/test.txt",
			"content": "test"
		},
		"session_id": "test-violation"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "violation-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Set current tier to haiku (cannot use Write)
	tierFile := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierFile), 0755)
	os.WriteFile(tierFile, []byte("haiku\n"), 0644)

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify violation logged
	if _, err := os.Stat(violationsLog); err != nil {
		t.Fatal("Violations log not created")
	}

	logData, err := os.ReadFile(violationsLog)
	if err != nil {
		t.Fatalf("Failed to read violations log: %v", err)
	}

	if len(logData) == 0 {
		t.Error("Violations log is empty")
	}

	// Parse violation
	var violation map[string]interface{}
	if err := json.Unmarshal(logData, &violation); err != nil {
		t.Fatalf("Failed to parse violation: %v", err)
	}

	if violation["violation_type"] != "tool_permission" {
		t.Errorf("Expected tool_permission violation, got: %v", violation["violation_type"])
	}
}

// Helper: Create corpus with various validation scenarios
func createValidateCorpus(t *testing.T, path string) {
	events := []string{
		// Valid Task(haiku)
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"haiku","prompt":"AGENT: codebase-search\n\nFind files","subagent_type":"Explore"},"session_id":"test-1"}`,

		// Invalid Task(opus)
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"opus","prompt":"AGENT: einstein\n\nDeep analysis","subagent_type":"general-purpose"},"session_id":"test-2"}`,

		// Valid Read
		`{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"},"session_id":"test-3"}`,

		// Invalid subagent_type
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"haiku","prompt":"AGENT: tech-docs-writer\n\nWrite docs","subagent_type":"Explore"},"session_id":"test-4"}`,
	}

	content := strings.Join(events, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create corpus: %v", err)
	}
}

// Helper: Setup minimal routing schema for tests
func setupTestRoutingSchema(t *testing.T, projectDir string) {
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	schema := `{
		"tiers": {
			"haiku": {
				"tools_allowed": ["Read", "Glob", "Grep"],
				"task_invocation_allowed": true
			},
			"sonnet": {
				"tools_allowed": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"],
				"task_invocation_allowed": true
			},
			"opus": {
				"tools_allowed": ["*"],
				"task_invocation_blocked": true,
				"blocked_reason": "Use /einstein slash command"
			}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"tech-docs-writer": "general-purpose",
			"einstein": "general-purpose"
		}
	}`

	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to create routing schema: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestValidateRouting_Integration` runs all PreToolUse events from corpus
- [ ] `TestValidateRouting_BlocksOpusTask` verifies Task(opus) blocked
- [ ] `TestValidateRouting_AllowsValidTask` verifies valid Task(haiku) allowed
- [ ] `TestValidateRouting_ViolationLogging` verifies violations logged to JSONL
- [ ] All hooks return valid JSON with "decision" field
- [ ] Integration tests pass: `go test ./test/integration -v -run TestValidateRouting`
- [ ] Test coverage ≥80% for validation workflow

**Why This Matters**: Validates that Go implementation matches Bash routing logic. Ensures cost control (Opus blocking) and tier enforcement work correctly.

---
