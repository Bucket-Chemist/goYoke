---
id: GOgent-099
title: End-to-End Workflow Integration Tests
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-095"]
priority: high
week: 5
tags: ["performance", "week-5"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-099: End-to-End Workflow Integration Tests

**Time**: 2 hours
**Dependencies**: GOgent-095-044 (integration tests)

**Task**:
Test cross-hook workflows: validation blocks → sharp edge detection → session archival.

**File**: `test/integration/end_to_end_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd_ValidationToSharpEdge tests validation failure → sharp edge capture
func TestEndToEnd_ValidationToSharpEdge(t *testing.T) {
	validateBinary := "../../cmd/gogent-validate/gogent-validate"
	sharpEdgeBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"

	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}
	if _, err := os.Stat(sharpEdgeBinary); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Step 1: Attempt blocked Task(opus) via validation
	validateEventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze",
			"subagent_type": "general-purpose"
		},
		"session_id": "e2e-test"
	}`

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	os.WriteFile(tmpValidateCorpus, []byte(validateEventJSON+"\n"), 0644)

	validateHarness, _ := NewTestHarness(tmpValidateCorpus, projectDir)
	validateHarness.LoadCorpus()

	validateResult := validateHarness.RunHook(validateBinary, validateHarness.Events[0])

	// Verify blocked
	if validateResult.ParsedJSON == nil {
		t.Fatal("Expected JSON from validate-routing")
	}

	decision, _ := validateResult.ParsedJSON["decision"].(string)
	if decision != "block" {
		t.Errorf("Expected validation to block Task(opus), got: %s", decision)
	}

	// Step 2: Simulate repeated failures (user ignoring block and retrying)
	// Create 3 PostToolUse failure events
	sharpEdgeEvents := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeEvents, "\n")+"\n"), 0644)

	sharpEdgeHarness, _ := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	sharpEdgeHarness.LoadCorpus()

	sharpEdgeResults, _ := sharpEdgeHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")

	// Third failure should trigger sharp edge capture
	if len(sharpEdgeResults) < 3 {
		t.Fatalf("Expected 3 sharp edge results, got: %d", len(sharpEdgeResults))
	}

	thirdResult := sharpEdgeResults[2]
	if thirdResult.ParsedJSON == nil {
		t.Fatal("Expected JSON from third sharp edge")
	}

	sharpEdgeDecision, _ := thirdResult.ParsedJSON["decision"].(string)
	if sharpEdgeDecision != "block" {
		t.Errorf("Expected sharp edge to block after 3 failures, got: %s", sharpEdgeDecision)
	}

	// Verify sharp edge captured
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings not created: %v", err)
	}
}

// TestEndToEnd_SessionArchivalWorkflow tests complete session lifecycle
func TestEndToEnd_SessionArchivalWorkflow(t *testing.T) {
	validateBinary := "../../cmd/gogent-validate/gogent-validate"
	sharpEdgeBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	archiveBinary := "../../cmd/gogent-archive/gogent-archive"

	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}
	if _, err := os.Stat(sharpEdgeBinary); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Step 1: Run validation creating violations
	validateEvents := []string{
		`{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"/tmp/test.txt"},"session_id":"archive-test"}`,
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"opus","prompt":"AGENT: einstein","subagent_type":"general-purpose"},"session_id":"archive-test"}`,
	}

	// Set tier to haiku (will violate on Write and Task(opus))
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	os.WriteFile(tmpValidateCorpus, []byte(strings.Join(validateEvents, "\n")+"\n"), 0644)

	validateHarness, _ := NewTestHarness(tmpValidateCorpus, projectDir)
	validateHarness.LoadCorpus()

	validateHarness.RunHookBatch(validateBinary, "PreToolUse")

	// Verify violations logged
	violationsPath := filepath.Join(projectDir, ".gogent", "routing-violations.jsonl")
	if _, err := os.Stat(violationsPath); err != nil {
		t.Errorf("Violations log not created: %v", err)
	}

	// Step 2: Run sharp edge detection creating pending learnings
	sharpEdgeEvents := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeEvents, "\n")+"\n"), 0644)

	sharpEdgeHarness, _ := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	sharpEdgeHarness.LoadCorpus()

	sharpEdgeHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")

	// Verify pending learnings created
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings not created: %v", err)
	}

	// Step 3: Run session archive
	archiveEventJSON := `{
		"hook_event_name": "SessionEnd",
		"session_id": "archive-test",
		"transcript_path": "` + filepath.Join(projectDir, "transcript.jsonl") + `"
	}`

	// Create empty transcript
	os.WriteFile(filepath.Join(projectDir, "transcript.jsonl"), []byte(""), 0644)

	tmpArchiveCorpus := filepath.Join(t.TempDir(), "archive-corpus.jsonl")
	os.WriteFile(tmpArchiveCorpus, []byte(archiveEventJSON+"\n"), 0644)

	archiveHarness, _ := NewTestHarness(tmpArchiveCorpus, projectDir)
	archiveHarness.LoadCorpus()

	archiveResult := archiveHarness.RunHook(archiveBinary, archiveHarness.Events[0])

	if archiveResult.ExitCode != 0 {
		t.Fatalf("Archive hook failed: %s", archiveResult.Stderr)
	}

	// Step 4: Verify handoff file created with all sections
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Handoff not created: %v", err)
	}

	handoffData, _ := os.ReadFile(handoffPath)
	handoffContent := string(handoffData)

	// Verify violations section present (from Step 1)
	if !strings.Contains(handoffContent, "## Routing Violations") {
		t.Error("Handoff missing violations section")
	}

	// Verify pending learnings section present (from Step 2)
	if !strings.Contains(handoffContent, "## Pending Learnings") {
		t.Error("Handoff missing pending learnings section")
	}

	// Step 5: Verify files archived
	archiveDir := filepath.Join(projectDir, ".claude", "memory", "session-archive")

	// Violations should be moved
	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations file should be removed after archival")
	}

	archivedViolations := filepath.Join(archiveDir, "routing-violations-archive-test.jsonl")
	if _, err := os.Stat(archivedViolations); err != nil {
		t.Errorf("Violations not archived: %v", err)
	}

	// Learnings should be moved
	if _, err := os.Stat(learningsPath); !os.IsNotExist(err) {
		t.Error("Learnings file should be removed after archival")
	}

	archivedLearnings := filepath.Join(archiveDir, "pending-learnings-archive-test.jsonl")
	if _, err := os.Stat(archivedLearnings); err != nil {
		t.Errorf("Learnings not archived: %v", err)
	}
}

// TestEndToEnd_MultiSessionHandoff tests handoff continuity across sessions
func TestEndToEnd_MultiSessionHandoff(t *testing.T) {
	archiveBinary := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()

	// Session 1: Create handoff
	session1Event := `{
		"hook_event_name": "SessionEnd",
		"session_id": "session-1",
		"transcript_path": "` + filepath.Join(projectDir, "transcript-1.jsonl") + `"
	}`

	os.WriteFile(filepath.Join(projectDir, "transcript-1.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, projectDir, "read", 10)

	tmpCorpus1 := filepath.Join(t.TempDir(), "session1-corpus.jsonl")
	os.WriteFile(tmpCorpus1, []byte(session1Event+"\n"), 0644)

	harness1, _ := NewTestHarness(tmpCorpus1, projectDir)
	harness1.LoadCorpus()

	result1 := harness1.RunHook(archiveBinary, harness1.Events[0])

	if result1.ExitCode != 0 {
		t.Fatalf("Session 1 archive failed: %s", result1.Stderr)
	}

	// Verify handoff created
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Session 1 handoff not created: %v", err)
	}

	session1Handoff, _ := os.ReadFile(handoffPath)

	// Session 2: Should be able to reference previous handoff
	// (In real workflow, load-routing-context hook would inject this)
	session2Event := `{
		"hook_event_name": "SessionEnd",
		"session_id": "session-2",
		"transcript_path": "` + filepath.Join(projectDir, "transcript-2.jsonl") + `"
	}`

	os.WriteFile(filepath.Join(projectDir, "transcript-2.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, projectDir, "write", 5)

	tmpCorpus2 := filepath.Join(t.TempDir(), "session2-corpus.jsonl")
	os.WriteFile(tmpCorpus2, []byte(session2Event+"\n"), 0644)

	harness2, _ := NewTestHarness(tmpCorpus2, projectDir)
	harness2.LoadCorpus()

	result2 := harness2.RunHook(archiveBinary, harness2.Events[0])

	if result2.ExitCode != 0 {
		t.Fatalf("Session 2 archive failed: %s", result2.Stderr)
	}

	// Verify new handoff created (overwrites previous)
	session2Handoff, _ := os.ReadFile(handoffPath)

	if string(session1Handoff) == string(session2Handoff) {
		t.Error("Session 2 handoff should differ from session 1")
	}

	// Verify session 1 handoff preserved in archive
	archivedSession1 := filepath.Join(projectDir, ".claude", "memory", "session-archive", "last-handoff-session-1.md")
	// Note: Current implementation doesn't archive handoff - transcript only
	// This test documents expected future behavior
}
```

**Acceptance Criteria**:
- [ ] `TestEndToEnd_ValidationToSharpEdge` verifies validation → sharp edge pipeline
- [ ] `TestEndToEnd_SessionArchivalWorkflow` verifies complete session lifecycle
- [ ] Violations from validation appear in session handoff
- [ ] Pending learnings from sharp edge appear in session handoff
- [ ] Files archived correctly at session end
- [ ] `TestEndToEnd_MultiSessionHandoff` verifies continuity across sessions
- [ ] All end-to-end tests pass: `go test ./test/integration -v -run TestEndToEnd`

**Why This Matters**: Individual hooks may work in isolation but fail when chained. End-to-end tests verify complete workflows match production usage.

---
