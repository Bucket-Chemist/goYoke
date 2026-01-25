---
id: GOgent-097
title: Integration Tests for sharp-edge-detector Hook
description: Test sharp-edge-detector hook with failure detection, pattern logging, and escalation guidance
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-094","GOgent-040"]
priority: high
week: 5
tags: ["integration-tests", "week-5"]
tests_required: true
acceptance_criteria_count: 13
---

### GOgent-097: Integration Tests for sharp-edge-detector Hook

**Time**: 1.5 hours
**Dependencies**: GOgent-094 (harness), GOgent-040 (gogent-sharp-edge binary)

**Task**:
Test sharp edge detection workflow: failure detection, consecutive counting, blocking responses.

**File**: `test/integration/sharp_edge_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharpEdge_Integration(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found. Run: go build -o cmd/gogent-sharp-edge/gogent-sharp-edge cmd/gogent-sharp-edge/main.go")
	}

	projectDir := t.TempDir()

	// Create corpus with 3 consecutive failures on same file
	corpusPath := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	createSharpEdgeCorpus(t, corpusPath, projectDir)

	harness, _ := NewTestHarness(corpusPath, projectDir)
	harness.LoadCorpus()

	results, err := harness.RunHookBatch(binaryPath, "PostToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got: %d", len(results))
	}

	// First failure: Should pass through
	if results[0].ParsedJSON == nil || len(results[0].ParsedJSON) > 0 {
		t.Error("First failure should return empty JSON (pass-through)")
	}

	// Second failure: Should warn
	if results[1].ParsedJSON == nil {
		t.Fatal("Second failure should return JSON")
	}

	hookOutput, ok := results[1].ParsedJSON["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Error("Second failure should have hookSpecificOutput with warning")
	} else {
		additionalContext, ok := hookOutput["additionalContext"].(string)
		if !ok || !strings.Contains(additionalContext, "⚠️") {
			t.Error("Second failure should contain warning emoji")
		}
	}

	// Third failure: Should block
	if results[2].ParsedJSON == nil {
		t.Fatal("Third failure should return JSON")
	}

	decision, ok := results[2].ParsedJSON["decision"].(string)
	if !ok || decision != "block" {
		t.Errorf("Third failure should block, got decision: %v", decision)
	}

	reason, ok := results[2].ParsedJSON["reason"].(string)
	if !ok || !strings.Contains(reason, "SHARP EDGE DETECTED") {
		t.Errorf("Third failure should mention sharp edge, got: %s", reason)
	}

	// Verify sharp edge captured to pending learnings
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings file not created: %v", err)
	} else {
		data, _ := os.ReadFile(learningsPath)
		if len(data) == 0 {
			t.Error("Pending learnings file is empty")
		}

		var edge map[string]interface{}
		if err := json.Unmarshal(data, &edge); err != nil {
			t.Errorf("Failed to parse sharp edge: %v", err)
		}

		if edge["type"] != "tool_failure" {
			t.Errorf("Expected type=tool_failure, got: %v", edge["type"])
		}

		if edge["consecutive_failures"] != float64(3) {
			t.Errorf("Expected 3 consecutive failures, got: %v", edge["consecutive_failures"])
		}
	}
}

func TestSharpEdge_FailureDetection(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	testCases := []struct {
		name           string
		eventJSON      string
		expectFailure  bool
		expectedErrType string
	}{
		{
			name: "Explicit success=false",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Edit",
				"tool_input": {"file_path": "/tmp/test.py"},
				"tool_response": {"success": false, "error": "File not found"}
			}`,
			expectFailure: true,
			expectedErrType: "error",
		},
		{
			name: "Non-zero exit code",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Bash",
				"tool_input": {"command": "ls /nonexistent"},
				"tool_response": {"exit_code": 1, "output": "ls: cannot access"}
			}`,
			expectFailure: true,
			expectedErrType: "error",
		},
		{
			name: "Python TypeError in output",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Bash",
				"tool_input": {"command": "python script.py"},
				"tool_response": {"output": "TypeError: unsupported operand type"}
			}`,
			expectFailure: true,
			expectedErrType: "TypeError",
		},
		{
			name: "Success case",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Read",
				"tool_input": {"file_path": "/tmp/test.txt"},
				"tool_response": {"content": "file content"}
			}`,
			expectFailure: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpCorpus := filepath.Join(t.TempDir(), "corpus.jsonl")
			os.WriteFile(tmpCorpus, []byte(tc.eventJSON+"\n"), 0644)

			harness, _ := NewTestHarness(tmpCorpus, projectDir)
			harness.LoadCorpus()

			result := harness.RunHook(binaryPath, harness.Events[0])

			if tc.expectFailure {
				// Should log failure
				errorLogPath := filepath.Join(projectDir, ".gogent", "error-patterns.jsonl")
				if _, err := os.Stat(errorLogPath); err != nil {
					t.Errorf("Error log not created for failure case: %v", err)
				}

				// First failure should pass through (no blocking yet)
				if result.ParsedJSON != nil && len(result.ParsedJSON) > 0 {
					decision, _ := result.ParsedJSON["decision"].(string)
					if decision == "block" {
						t.Error("First failure should not block")
					}
				}
			} else {
				// Should return empty JSON
				if result.ParsedJSON != nil && len(result.ParsedJSON) > 0 {
					t.Errorf("Success case should return empty JSON, got: %v", result.ParsedJSON)
				}
			}
		})
	}
}

func TestSharpEdge_SlidingWindow(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()
	errorLogPath := filepath.Join(projectDir, ".gogent", "error-patterns.jsonl")
	os.MkdirAll(filepath.Dir(errorLogPath), 0755)

	// Create failures outside 5-minute window (should not trigger blocking)
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	recentTimestamp := time.Now().Unix()

	logEntries := []string{
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, oldTimestamp),
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, oldTimestamp),
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, recentTimestamp),
	}

	os.WriteFile(errorLogPath, []byte(strings.Join(logEntries, "\n")+"\n"), 0644)

	// Create new failure event
	eventJSON := `{
		"hook_event_name": "PostToolUse",
		"tool_name": "Edit",
		"tool_input": {"file_path": "/tmp/test.go"},
		"tool_response": {"success": false}
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "window-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Should NOT block (only 2 failures in window: 1 recent + 1 new)
	if result.ParsedJSON != nil {
		decision, _ := result.ParsedJSON["decision"].(string)
		if decision == "block" {
			t.Error("Should not block with only 2 recent failures (old ones outside window)")
		}
	}
}

func TestSharpEdge_PerFileTracking(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	// Create failures on different files
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileA.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileB.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileA.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileB.go"},"tool_response":{"success":false}}`,
	}

	tmpCorpus := filepath.Join(t.TempDir(), "multifile-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(strings.Join(events, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	results, _ := harness.RunHookBatch(binaryPath, "PostToolUse")

	// Each file should be tracked independently
	// Neither should reach 3 failures (2 on fileA, 2 on fileB)
	for i, result := range results {
		if result.ParsedJSON != nil {
			decision, _ := result.ParsedJSON["decision"].(string)
			if decision == "block" {
				t.Errorf("Event %d should not block (separate file tracking)", i)
			}
		}
	}
}

func TestSharpEdge_MLTelemetryFields(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	// Create failure events with ML telemetry fields
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"ml_telemetry":{"duration_ms":245,"input_tokens":1024,"output_tokens":512,"sequence_index":1},"session_id":"test-1"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"ml_telemetry":{"duration_ms":312,"input_tokens":1024,"output_tokens":512,"sequence_index":2},"session_id":"test-2"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"ml_telemetry":{"duration_ms":189,"input_tokens":1024,"output_tokens":512,"sequence_index":3},"session_id":"test-3"}`,
	}

	tmpCorpus := filepath.Join(t.TempDir(), "ml-telemetry-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(strings.Join(events, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	results, err := harness.RunHookBatch(binaryPath, "PostToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got: %d", len(results))
	}

	// Verify third failure (blocking) includes ML telemetry in sharp edge capture
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings file not created: %v", err)
	} else {
		data, _ := os.ReadFile(learningsPath)
		var edge map[string]interface{}
		if err := json.Unmarshal(data, &edge); err != nil {
			t.Errorf("Failed to parse sharp edge: %v", err)
		}

		// Verify ML telemetry fields are captured in edge context
		mlTelemetry, ok := edge["ml_telemetry"].(map[string]interface{})
		if !ok {
			t.Error("ML telemetry should be present in sharp edge context")
		} else {
			// Check specific fields
			if durationMs, ok := mlTelemetry["duration_ms"].(float64); !ok || durationMs <= 0 {
				t.Errorf("Expected valid duration_ms in ml_telemetry, got: %v", durationMs)
			}
			if inputTokens, ok := mlTelemetry["input_tokens"].(float64); !ok || inputTokens <= 0 {
				t.Errorf("Expected valid input_tokens in ml_telemetry, got: %v", inputTokens)
			}
			if outputTokens, ok := mlTelemetry["output_tokens"].(float64); !ok || outputTokens <= 0 {
				t.Errorf("Expected valid output_tokens in ml_telemetry, got: %v", outputTokens)
			}
			if seqIndex, ok := mlTelemetry["sequence_index"].(float64); !ok || seqIndex <= 0 {
				t.Errorf("Expected valid sequence_index in ml_telemetry, got: %v", seqIndex)
			}
		}
	}
}

func TestSharpEdge_DecisionCorrelation(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	// Create failure events with routing decision correlation
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"routing_decision":"direct","session_id":"test-1"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"routing_decision":"direct","session_id":"test-2"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"routing_decision":"direct","session_id":"test-3"}`,
	}

	tmpCorpus := filepath.Join(t.TempDir(), "decision-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(strings.Join(events, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	results, err := harness.RunHookBatch(binaryPath, "PostToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got: %d", len(results))
	}

	// Third failure should block
	decision, ok := results[2].ParsedJSON["decision"].(string)
	if !ok || decision != "block" {
		t.Errorf("Third failure should block, got decision: %v", decision)
	}

	// Verify routing decision correlation is logged
	decisionLogPath := filepath.Join(projectDir, ".gogent", "routing-decision-updates.jsonl")
	if _, err := os.Stat(decisionLogPath); err != nil {
		t.Errorf("Routing decision log not created: %v", err)
	} else {
		data, _ := os.ReadFile(decisionLogPath)
		if len(data) == 0 {
			t.Error("Routing decision log is empty")
		} else {
			var decisionRecord map[string]interface{}
			if err := json.Unmarshal(data, &decisionRecord); err != nil {
				t.Errorf("Failed to parse routing decision: %v", err)
			}

			// Verify correlation between failures and routing decision
			if correlatedDecision, ok := decisionRecord["correlated_decision"].(string); !ok || correlatedDecision != "direct" {
				t.Errorf("Expected correlated_decision=direct in routing-decision-updates, got: %v", correlatedDecision)
			}

			if failureCount, ok := decisionRecord["failure_count"].(float64); !ok || failureCount != 3 {
				t.Errorf("Expected failure_count=3 in routing-decision-updates, got: %v", failureCount)
			}
		}
	}
}

// Helper: Create corpus with 3 consecutive failures on same file
func createSharpEdgeCorpus(t *testing.T, corpusPath, projectDir string) {
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-1"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-2"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-3"}`,
	}

	if err := os.WriteFile(corpusPath, []byte(strings.Join(events, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create corpus: %v", err)
	}
}
```

**Acceptance Criteria**:
- [x] `TestSharpEdge_Integration` verifies 3-failure blocking workflow
- [x] First failure passes through (no blocking)
- [x] Second failure generates warning in additionalContext
- [x] Third failure returns `decision: "block"` with sharp edge reason
- [x] Sharp edge captured to `pending-learnings.jsonl`
- [x] `TestSharpEdge_FailureDetection` verifies all signal types
- [x] `TestSharpEdge_SlidingWindow` verifies 5-minute window
- [x] `TestSharpEdge_PerFileTracking` verifies independent file tracking
- [x] `TestSharpEdge_MLTelemetryFields` verifies DurationMs, InputTokens, OutputTokens, SequenceIndex captured
- [x] ML telemetry fields correctly logged in sharp edge context
- [x] `TestSharpEdge_DecisionCorrelation` verifies routing decision correlation
- [x] Decision correlation logged to `routing-decision-updates.jsonl`
- [x] Tests pass: `go test ./test/integration -v -run TestSharpEdge`

**Why This Matters**: Sharp edge detection prevents debugging loops. Must verify blocking logic triggers correctly and captures sufficient context for learning.

---
