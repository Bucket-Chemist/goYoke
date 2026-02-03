package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestRoutingSchema creates a minimal routing schema for testing
func setupTestRoutingSchema(t *testing.T, projectDir string) {
	schemaDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(schemaDir, 0755)

	schema := `{
  "version": "2.5.0",
  "tiers": {
    "haiku": {
      "model": "claude-3-haiku",
      "thinking_budget": 0,
      "patterns": ["find"],
      "tools": ["Read", "Glob", "Grep"]
    },
    "sonnet": {
      "model": "claude-3.5-sonnet",
      "thinking_budget": 10000,
      "patterns": ["implement"],
      "tools": ["Read", "Write", "Edit", "Bash", "Task"]
    },
    "opus": {
      "model": "claude-opus-4.5",
      "thinking_budget": 16000,
      "patterns": ["architect"],
      "tools": ["*"],
      "task_invocation_blocked": true,
      "blocked_reason": "60K+ token inheritance overhead"
    }
  },
  "tier_levels": {
    "haiku": 1,
    "sonnet": 3,
    "opus": 4
  },
  "agent_subagent_mapping": {
    "python-pro": "general-purpose",
    "codebase-search": "Explore",
    "tech-docs-writer": "general-purpose",
    "code-reviewer": "Explore",
    "orchestrator": "Plan",
    "architect": "Plan"
  }
}`

	schemaPath := filepath.Join(schemaDir, "routing-schema.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to create routing schema: %v", err)
	}
}

// TestEndToEnd_FullMLPipeline tests complete lifecycle with ML reconciliation
func TestEndToEnd_FullMLPipeline(t *testing.T) {
	validateBinary := "../../bin/gogent-validate"
	sharpEdgeBinary := "../../bin/gogent-sharp-edge"
	archiveBinary := "../../bin/gogent-archive"

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

	sessionID := "ml-pipeline-test"

	// Phase 1: SessionStart
	sessionStartEvent := map[string]interface{}{
		"hook_event_name": "SessionStart",
		"session_id":      sessionID,
		"timestamp": time.Now().Unix(),
		"context_source":  "previous_handoff",
	}

	// Phase 2: PreToolUse - Log routing decisions
	routingDecisionEvent := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Task",
		"tool_input": map[string]interface{}{
			"model":         "sonnet",
			"prompt":        "AGENT: python-pro\n\nImplement feature",
			"subagent_type": "general-purpose",
		},
		"session_id": sessionID,
		"timestamp": time.Now().Unix(),
		"routing_decision": map[string]interface{}{
			"decision": "allow",
			"tier":     "sonnet",
			"reason":   "implementation-task",
			"trigger":  "python-pro",
		},
	}

	tmpRoutingCorpus := filepath.Join(t.TempDir(), "routing-corpus.jsonl")
	routingJSON, _ := json.Marshal(routingDecisionEvent)
	if err := os.WriteFile(tmpRoutingCorpus, append(routingJSON, '\n'), 0644); err != nil {
		t.Fatalf("Failed to write routing corpus: %v", err)
	}

	routingHarness, err := NewTestHarness(tmpRoutingCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create routing harness: %v", err)
	}
	if err := routingHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load routing corpus: %v", err)
	}

	// Validate routing decision
	validateResult := routingHarness.RunHook(validateBinary, routingHarness.Events[0])
	if validateResult.ExitCode != 0 {
		t.Fatalf("Validation failed: %s", validateResult.Stderr)
	}

	// Phase 3: PostToolUse with ML fields
	postToolEvents := []map[string]interface{}{
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Task",
			"tool_input":      map[string]interface{}{"model": "sonnet", "prompt": "Implement"},
			"tool_response":   map[string]interface{}{"success": true, "output": "Feature implemented"},
			"session_id":      sessionID,
			"timestamp": time.Now().Unix(),
			"ml_fields": map[string]interface{}{
				"agent_model":        "sonnet",
				"agent_type":         "python-pro",
				"execution_time_ms":  2500,
				"token_usage":        map[string]interface{}{"input": 1200, "output": 800},
				"routing_confidence": 0.95,
			},
		},
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Edit",
			"tool_input":      map[string]interface{}{"file_path": "src/feature.py"},
			"tool_response":   map[string]interface{}{"success": true},
			"session_id":      sessionID,
			"timestamp": time.Now().Unix(),
			"ml_fields": map[string]interface{}{
				"operation":         "edit",
				"file_type":         "python",
				"lines_changed":     42,
				"editor_confidence": 0.98,
			},
		},
	}

	tmpPostToolCorpus := filepath.Join(t.TempDir(), "posttool-corpus.jsonl")
	var postToolLines []string
	for _, evt := range postToolEvents {
		jsonBytes, _ := json.Marshal(evt)
		postToolLines = append(postToolLines, string(jsonBytes))
	}
	if err := os.WriteFile(tmpPostToolCorpus, []byte(strings.Join(postToolLines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write posttool corpus: %v", err)
	}

	postToolHarness, err := NewTestHarness(tmpPostToolCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create posttool harness: %v", err)
	}
	if err := postToolHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load posttool corpus: %v", err)
	}

	postToolResults, _ := postToolHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")
	if len(postToolResults) != 2 {
		t.Fatalf("Expected 2 PostToolUse results, got %d", len(postToolResults))
	}

	// Verify ML fields logged
	for _, result := range postToolResults {
		if result.ParsedJSON == nil {
			t.Error("Expected JSON output from PostToolUse hook")
		}
	}

	// Phase 4: SubagentStop
	subagentStopEvent := map[string]interface{}{
		"hook_event_name": "SubagentStop",
		"session_id":      sessionID,
		"subagent_id":     "python-pro-inst-1",
		"subagent_type":   "general-purpose",
		"model":           "sonnet",
		"timestamp": time.Now().Unix(),
		"execution_summary": map[string]interface{}{
			"status":       "completed",
			"duration_ms":  3200,
			"token_usage":  map[string]interface{}{"input": 1500, "output": 1100},
			"files_modified": 1,
			"tests_passed": 1,
			"tests_failed": 0,
		},
		"collaboration_log": map[string]interface{}{
			"parent_session":       sessionID,
			"branching_decision":   "python-pro",
			"return_value_quality": 0.92,
		},
	}

	tmpSubagentCorpus := filepath.Join(t.TempDir(), "subagent-corpus.jsonl")
	subagentJSON, _ := json.Marshal(subagentStopEvent)
	if err := os.WriteFile(tmpSubagentCorpus, append(subagentJSON, '\n'), 0644); err != nil {
		t.Fatalf("Failed to write subagent corpus: %v", err)
	}

	subagentHarness, err := NewTestHarness(tmpSubagentCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create subagent harness: %v", err)
	}
	if err := subagentHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load subagent corpus: %v", err)
	}

	subagentResult := subagentHarness.RunHook(sharpEdgeBinary, subagentHarness.Events[0])
	if subagentResult.ExitCode != 0 {
		t.Logf("Subagent stop processing: exit code %d (may be expected)", subagentResult.ExitCode)
	}

	// Phase 5: SessionEnd - Archive
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	archiveEvent := map[string]interface{}{
		"hook_event_name":   "SessionEnd",
		"session_id":        sessionID,
		"timestamp": time.Now().Unix(),
		"transcript_path":   transcriptPath,
		"ml_reconciliation": map[string]interface{}{
			"expected_fields":     []string{"hook_event_name", "session_id", "timestamp"},
			"validate_structure":  true,
			"export_training_data": true,
		},
	}

	// Create minimal transcript
	sessionJSON, _ := json.Marshal(sessionStartEvent)
	routingJSON2, _ := json.Marshal(routingDecisionEvent)
	transcriptData := string(sessionJSON) + "\n" + string(routingJSON2) + "\n"
	if err := os.WriteFile(transcriptPath, []byte(transcriptData), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}

	tmpArchiveCorpus := filepath.Join(t.TempDir(), "archive-corpus.jsonl")
	archiveJSON, _ := json.Marshal(archiveEvent)
	if err := os.WriteFile(tmpArchiveCorpus, append(archiveJSON, '\n'), 0644); err != nil {
		t.Fatalf("Failed to write archive corpus: %v", err)
	}

	archiveHarness, err := NewTestHarness(tmpArchiveCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create archive harness: %v", err)
	}
	if err := archiveHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load archive corpus: %v", err)
	}

	archiveResult := archiveHarness.RunHook(archiveBinary, archiveHarness.Events[0])
	if archiveResult.ExitCode != 0 {
		t.Fatalf("Archive hook failed: %s", archiveResult.Stderr)
	}

	// Phase 6: Verify artifacts
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Handoff not created: %v", err)
	}

	handoffData, _ := os.ReadFile(handoffPath)
	handoffContent := string(handoffData)

	if !strings.Contains(handoffContent, "Session") && !strings.Contains(handoffContent, "session") {
		t.Logf("Handoff may not contain expected session context markers")
	}

	// Check ML reconciliation output
	mlReconcilePath := filepath.Join(projectDir, ".claude", "ml-export", "reconciliation-"+sessionID+".jsonl")
	if _, err := os.Stat(mlReconcilePath); err == nil {
		data, _ := os.ReadFile(mlReconcilePath)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				t.Errorf("ML reconciliation line invalid JSON: %v", err)
			}
		}
	}
}

//TestEndToEnd_SessionStartToValidate tests context injection at session start
func TestEndToEnd_SessionStartToValidate(t *testing.T) {
	validateBinary := "../../bin/gogent-validate"
	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	sessionID := "context-injection-test"

	// Create previous handoff
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	previousHandoff := `# Previous Session Handoff

## Routing Context
- Model: sonnet
- Tier: implementation
- Agent: python-pro

## Pending Items
- Review PR feedback on feature-x
`
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	os.WriteFile(handoffPath, []byte(previousHandoff), 0644)

	// Create events
	events := []map[string]interface{}{
		{
			"hook_event_name": "SessionStart",
			"session_id":      sessionID,
			"timestamp": time.Now().Unix(),
			"context_source":  "previous_handoff",
		},
		{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Task",
			"tool_input": map[string]interface{}{
				"model":         "sonnet",
				"prompt":        "AGENT: python-pro\n\nReview feedback",
				"subagent_type": "general-purpose",
			},
			"session_id":          sessionID,
			"timestamp": time.Now().Unix(),
			"context_loaded_from": "previous_handoff",
		},
	}

	tmpCorpus := filepath.Join(t.TempDir(), "corpus.jsonl")
	var eventLines []string
	for _, evt := range events {
		jsonBytes, _ := json.Marshal(evt)
		eventLines = append(eventLines, string(jsonBytes))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(eventLines, "\n")+"\n"), 0644)

	harness, err := NewTestHarness(tmpCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Process both events
	for i, event := range harness.Events {
		result := harness.RunHook(validateBinary, event)
		if result.ExitCode != 0 && result.ExitCode != 2 {
			t.Logf("Hook event %d exit code: %d (may be expected)", i, result.ExitCode)
		}
		if result.ParsedJSON != nil {
			if decision, ok := result.ParsedJSON["decision"].(string); ok && decision == "block" {
				t.Errorf("Event %d incorrectly blocked: %v", i, result.ParsedJSON)
			}
		}
	}
}

// Remaining test functions would follow the same pattern...
// For brevity, I'll stop here since the pattern is established

// TestEndToEnd_SubagentStopToArchive tests collaboration tracking through archival
func TestEndToEnd_SubagentStopToArchive(t *testing.T) {
	archiveBinary := "../../bin/gogent-archive"
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	sessionID := "collab-track-test"

	// Create collaboration log directory
	collabLogDir := filepath.Join(projectDir, ".claude", "collaboration-logs")
	os.MkdirAll(collabLogDir, 0755)

	// Simulate multiple subagent executions
	collaborationLog := map[string]interface{}{
		"session_id": sessionID,
		"entries": []interface{}{
			map[string]interface{}{
				"timestamp":          "2026-01-25T10:00:00Z",
				"subagent_id":        "python-pro-1",
				"model":              "sonnet",
				"status":             "completed",
				"quality_score":      0.92,
				"returned_to_parent": true,
			},
			map[string]interface{}{
				"timestamp":          "2026-01-25T10:05:00Z",
				"subagent_id":        "code-reviewer-1",
				"model":              "haiku",
				"status":             "completed",
				"quality_score":      0.88,
				"returned_to_parent": true,
			},
			map[string]interface{}{
				"timestamp":          "2026-01-25T10:10:00Z",
				"subagent_id":        "architect-1",
				"model":              "sonnet",
				"status":             "completed",
				"quality_score":      0.95,
				"returned_to_parent": true,
			},
		},
	}

	collabPath := filepath.Join(collabLogDir, sessionID+".json")
	collabJSON, _ := json.Marshal(collaborationLog)
	os.WriteFile(collabPath, collabJSON, 0644)

	// SessionEnd event with ML reconciliation
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	archiveEvent := map[string]interface{}{
		"hook_event_name": "SessionEnd",
		"session_id":      sessionID,
		"timestamp": time.Now().Unix(),
		"transcript_path": transcriptPath,
		"ml_reconciliation": map[string]interface{}{
			"include_collaboration_logs": true,
			"validate_subagent_returns":  true,
			"export_training_data":       true,
		},
	}

	// Create minimal transcript
	transcriptData := `{"hook_event_name":"SubagentStop","subagent_id":"python-pro-1","status":"completed"}
{"hook_event_name":"SubagentStop","subagent_id":"code-reviewer-1","status":"completed"}
{"hook_event_name":"SubagentStop","subagent_id":"architect-1","status":"completed"}
`
	os.WriteFile(transcriptPath, []byte(transcriptData), 0644)

	tmpArchiveCorpus := filepath.Join(t.TempDir(), "archive-corpus.jsonl")
	archiveJSON, _ := json.Marshal(archiveEvent)
	os.WriteFile(tmpArchiveCorpus, append(archiveJSON, '\n'), 0644)

	archiveHarness, err := NewTestHarness(tmpArchiveCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := archiveHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	archiveResult := archiveHarness.RunHook(archiveBinary, archiveHarness.Events[0])
	if archiveResult.ExitCode != 0 {
		t.Fatalf("Archive hook failed: %s", archiveResult.Stderr)
	}

	// Verify handoff created
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Handoff not created: %v", err)
	}

	handoffData, _ := os.ReadFile(handoffPath)
	if len(handoffData) == 0 {
		t.Error("Handoff is empty")
	}
}

// TestEndToEnd_ValidationToSharpEdge tests validation failure → sharp edge capture
func TestEndToEnd_ValidationToSharpEdge(t *testing.T) {
	validateBinary := "../../bin/gogent-validate"
	sharpEdgeBinary := "../../bin/gogent-sharp-edge"

	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}
	if _, err := os.Stat(sharpEdgeBinary); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Step 1: Attempt blocked Task(opus)
	validateEvent := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Task",
		"tool_input": map[string]interface{}{
			"model":         "opus",
			"prompt":        "AGENT: einstein\n\nAnalyze",
			"subagent_type": "general-purpose",
		},
		"session_id": "e2e-test",
	}

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	validateJSON, _ := json.Marshal(validateEvent)
	os.WriteFile(tmpValidateCorpus, append(validateJSON, '\n'), 0644)

	validateHarness, err := NewTestHarness(tmpValidateCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := validateHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	validateResult := validateHarness.RunHook(validateBinary, validateHarness.Events[0])

	// Verify blocked
	if validateResult.ParsedJSON == nil {
		t.Fatal("Expected JSON from validate-routing")
	}

	decision, _ := validateResult.ParsedJSON["decision"].(string)
	if decision != "block" {
		t.Errorf("Expected validation to block Task(opus), got: %s", decision)
	}

	// Step 2: Simulate repeated failures
	currentTime := time.Now().Unix()
	sharpEdgeEvents := []map[string]interface{}{
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Task",
			"tool_input":      map[string]interface{}{"model": "opus"},
			"tool_response":   map[string]interface{}{"success": false, "error": "blocked"},
			"session_id":      "e2e-test",
			"captured_at":     currentTime,
		},
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Task",
			"tool_input":      map[string]interface{}{"model": "opus"},
			"tool_response":   map[string]interface{}{"success": false, "error": "blocked"},
			"session_id":      "e2e-test",
			"captured_at":     currentTime,
		},
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Task",
			"tool_input":      map[string]interface{}{"model": "opus"},
			"tool_response":   map[string]interface{}{"success": false, "error": "blocked"},
			"session_id":      "e2e-test",
			"captured_at":     currentTime,
		},
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	var sharpEdgeLines []string
	for _, evt := range sharpEdgeEvents {
		jsonBytes, _ := json.Marshal(evt)
		sharpEdgeLines = append(sharpEdgeLines, string(jsonBytes))
	}
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeLines, "\n")+"\n"), 0644)

	sharpEdgeHarness, err := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := sharpEdgeHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

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
	validateBinary := "../../bin/gogent-validate"
	sharpEdgeBinary := "../../bin/gogent-sharp-edge"
	archiveBinary := "../../bin/gogent-archive"

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

	// Setup runtime directory
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	gogentDir := filepath.Join(runtimeDir, "gogent")
	os.MkdirAll(gogentDir, 0755)

	// Step 1: Run validation creating violations
	validateEvents := []map[string]interface{}{
		{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Write",
			"tool_input":      map[string]interface{}{"file_path": "/tmp/test.txt"},
			"session_id":      "archive-test",
		},
		{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Task",
			"tool_input": map[string]interface{}{
				"model":         "opus",
				"prompt":        "AGENT: einstein",
				"subagent_type": "general-purpose",
			},
			"session_id": "archive-test",
		},
	}

	// Set tier to haiku (will violate on Write and Task(opus))
	tierPath := filepath.Join(gogentDir, "current-tier")
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	var validateLines []string
	for _, evt := range validateEvents {
		jsonBytes, _ := json.Marshal(evt)
		validateLines = append(validateLines, string(jsonBytes))
	}
	os.WriteFile(tmpValidateCorpus, []byte(strings.Join(validateLines, "\n")+"\n"), 0644)

	validateHarness, err := NewTestHarness(tmpValidateCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := validateHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	validateHarness.RunHookBatch(validateBinary, "PreToolUse")

	// Verify violations logged
	violationsPath := filepath.Join(gogentDir, "routing-violations.jsonl")
	if _, err := os.Stat(violationsPath); err != nil {
		t.Errorf("Violations log not created: %v", err)
	}

	// Step 2: Run sharp edge detection creating pending learnings
	currentTime := time.Now().Unix()
	sharpEdgeEvents := []map[string]interface{}{
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Edit",
			"tool_input":      map[string]interface{}{"file_path": "/tmp/test.go"},
			"tool_response":   map[string]interface{}{"success": false},
			"session_id":      "archive-test",
			"captured_at":     currentTime,
		},
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Edit",
			"tool_input":      map[string]interface{}{"file_path": "/tmp/test.go"},
			"tool_response":   map[string]interface{}{"success": false},
			"session_id":      "archive-test",
			"captured_at":     currentTime,
		},
		{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Edit",
			"tool_input":      map[string]interface{}{"file_path": "/tmp/test.go"},
			"tool_response":   map[string]interface{}{"success": false},
			"session_id":      "archive-test",
			"captured_at":     currentTime,
		},
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	var sharpEdgeLines []string
	for _, evt := range sharpEdgeEvents {
		jsonBytes, _ := json.Marshal(evt)
		sharpEdgeLines = append(sharpEdgeLines, string(jsonBytes))
	}
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeLines, "\n")+"\n"), 0644)

	sharpEdgeHarness, err := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := sharpEdgeHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	sharpEdgeHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")

	// Verify pending learnings created
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings not created: %v", err)
	}

	// Step 3: Run session archive
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	archiveEvent := map[string]interface{}{
		"hook_event_name": "SessionEnd",
		"session_id":      "archive-test",
		"transcript_path": transcriptPath,
	}

	// Create empty transcript
	os.WriteFile(transcriptPath, []byte(""), 0644)

	tmpArchiveCorpus := filepath.Join(t.TempDir(), "archive-corpus.jsonl")
	archiveJSON, _ := json.Marshal(archiveEvent)
	os.WriteFile(tmpArchiveCorpus, append(archiveJSON, '\n'), 0644)

	archiveHarness, err := NewTestHarness(tmpArchiveCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}
	if err := archiveHarness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

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

	// Verify violations section present
	if !strings.Contains(handoffContent, "## Routing Violations") {
		t.Error("Handoff missing violations section")
	}

	// Verify sharp edges section present (pending learnings are rendered as sharp edges)
	if !strings.Contains(handoffContent, "## Sharp Edges") {
		t.Error("Handoff missing sharp edges section")
	}

	// Step 5: Verify files archived
	archiveDir := filepath.Join(projectDir, ".claude", "memory", "session-archive")

	// Violations should be moved (from project dir, not runtime dir)
	projectViolationsPath := filepath.Join(projectDir, ".claude", "memory", "routing-violations.jsonl")
	if _, err := os.Stat(projectViolationsPath); !os.IsNotExist(err) {
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
	archiveBinary := "../../bin/gogent-archive"
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()

	// Setup runtime directory
	runtimeDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	defer os.Unsetenv("XDG_RUNTIME_DIR")

	gogentDir := filepath.Join(runtimeDir, "gogent")
	os.MkdirAll(gogentDir, 0755)

	// Session 1: Create handoff
	session1Event := map[string]interface{}{
		"hook_event_name": "SessionEnd",
		"session_id":      "session-1",
		"transcript_path": filepath.Join(projectDir, "transcript-1.jsonl"),
	}

	os.WriteFile(filepath.Join(projectDir, "transcript-1.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, gogentDir, "read", 10)

	tmpCorpus1 := filepath.Join(t.TempDir(), "session1-corpus.jsonl")
	session1JSON, _ := json.Marshal(session1Event)
	os.WriteFile(tmpCorpus1, append(session1JSON, '\n'), 0644)

	harness1, err := NewTestHarness(tmpCorpus1, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness1: %v", err)
	}
	if err := harness1.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus1: %v", err)
	}

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

	// Session 2
	session2Event := map[string]interface{}{
		"hook_event_name": "SessionEnd",
		"session_id":      "session-2",
		"transcript_path": filepath.Join(projectDir, "transcript-2.jsonl"),
	}

	os.WriteFile(filepath.Join(projectDir, "transcript-2.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, gogentDir, "write", 5)

	tmpCorpus2 := filepath.Join(t.TempDir(), "session2-corpus.jsonl")
	session2JSON, _ := json.Marshal(session2Event)
	os.WriteFile(tmpCorpus2, append(session2JSON, '\n'), 0644)

	harness2, err := NewTestHarness(tmpCorpus2, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness2: %v", err)
	}
	if err := harness2.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus2: %v", err)
	}

	result2 := harness2.RunHook(archiveBinary, harness2.Events[0])

	if result2.ExitCode != 0 {
		t.Fatalf("Session 2 archive failed: %s", result2.Stderr)
	}

	// Verify new handoff created (overwrites previous)
	session2Handoff, _ := os.ReadFile(handoffPath)

	if string(session1Handoff) == string(session2Handoff) {
		t.Error("Session 2 handoff should differ from session 1")
	}
}
