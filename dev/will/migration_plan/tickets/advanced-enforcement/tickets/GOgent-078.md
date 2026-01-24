---
id: GOgent-078
title: Integration Tests for orchestrator-guard
description: End-to-end tests for orchestrator-guard workflow using simulation harness and realistic JSONL transcript format
status: pending
time_estimate: 2h
dependencies: ["GOgent-077"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5", "simulation-harness"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-078: Integration Tests for orchestrator-guard

**Time**: 2 hours
**Dependencies**: GOgent-077

**Task**:
End-to-end tests for orchestrator-guard workflow using simulation harness with realistic JSONL transcript data.

**Files**:
- `pkg/enforcement/orchestrator_integration_test.go` (tests)
- `test/simulation/fixtures/deterministic/orchestrator-guard/*.json` (fixtures)

**Implementation**:

```go
package enforcement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/test/simulation/harness"
)

// createTranscriptFromEntries creates a realistic JSONL transcript file
func createTranscriptFromEntries(t *testing.T, path string, entries []map[string]interface{}) {
	t.Helper()

	var lines []string
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("Failed to marshal entry: %v", err)
		}
		lines = append(lines, string(data))
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write transcript: %v", err)
	}
}

func TestOrchestratorGuardWorkflow_AllTasksCollected(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Realistic JSONL transcript with background tasks properly collected
	entries := []map[string]interface{}{
		{"timestamp": 1700000000.0, "content": "AGENT: orchestrator", "role": "system"},
		{"timestamp": 1700000001.0, "model": "sonnet"},
		{"timestamp": 1700000100.0, "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "gemini-slave architect 'Analyze module'",
			"run_in_background": true,
		}},
		{"timestamp": 1700000110.0, "tool_response": map[string]interface{}{
			"task_id": "bg-task-1",
		}},
		{"timestamp": 1700000200.0, "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "gemini-slave mapper 'List core files'",
			"run_in_background": true,
		}},
		{"timestamp": 1700000210.0, "tool_response": map[string]interface{}{
			"task_id": "bg-task-2",
		}},
		// FAN-IN: Collect all background tasks before completion
		{"timestamp": 1700000300.0, "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "bg-task-1",
			"block":   true,
		}},
		{"timestamp": 1700000350.0, "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "bg-task-2",
			"block":   true,
		}},
		{"timestamp": 1700000400.0, "content": "Orchestration complete", "role": "assistant"},
	}

	createTranscriptFromEntries(t, transcriptPath, entries)

	// Parse SubagentStop event (ACTUAL schema from GOgent-063a)
	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-001",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// Parse transcript for background task tracking
	metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)

	// Verify all tasks collected
	if len(metadata.UncollectedTasks) != 0 {
		t.Errorf("Expected 0 uncollected tasks, got %d: %v",
			len(metadata.UncollectedTasks), metadata.UncollectedTasks)
	}

	// Generate guard response
	response := generateGuardResponse(event, metadata)

	if response.Decision != "allow" {
		t.Errorf("Expected allow, got: %s (reason: %s)",
			response.Decision, response.AdditionalContext)
	}

	if !strings.Contains(response.AdditionalContext, "background tasks collected") {
		t.Error("Should mention task collection verification")
	}
}

func TestOrchestratorGuardWorkflow_UncollectedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Realistic JSONL with uncollected background tasks (VIOLATION)
	entries := []map[string]interface{}{
		{"timestamp": 1700000000.0, "content": "AGENT: orchestrator", "role": "system"},
		{"timestamp": 1700000001.0, "model": "sonnet"},
		{"timestamp": 1700000100.0, "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "gemini-slave architect 'Analyze module'",
			"run_in_background": true,
		}},
		{"timestamp": 1700000110.0, "tool_response": map[string]interface{}{
			"task_id": "bg-task-1",
		}},
		{"timestamp": 1700000200.0, "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "gemini-slave mapper 'List core files'",
			"run_in_background": true,
		}},
		{"timestamp": 1700000210.0, "tool_response": map[string]interface{}{
			"task_id": "bg-task-2",
		}},
		// ERROR: Only collected bg-task-1, bg-task-2 orphaned
		{"timestamp": 1700000300.0, "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "bg-task-1",
			"block":   true,
		}},
		// MISSING: TaskOutput for bg-task-2
		{"timestamp": 1700000400.0, "content": "Orchestration complete", "role": "assistant"},
	}

	createTranscriptFromEntries(t, transcriptPath, entries)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-002",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)

	// Verify violation detected
	if len(metadata.UncollectedTasks) != 1 {
		t.Errorf("Expected 1 uncollected task, got %d", len(metadata.UncollectedTasks))
	}

	if len(metadata.UncollectedTasks) > 0 && metadata.UncollectedTasks[0] != "bg-task-2" {
		t.Errorf("Expected uncollected task bg-task-2, got: %s", metadata.UncollectedTasks[0])
	}

	response := generateGuardResponse(event, metadata)

	if response.Decision != "block" {
		t.Errorf("Expected block, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "bg-task-2") {
		t.Error("Should mention uncollected task ID in response")
	}

	if !strings.Contains(response.AdditionalContext, "VIOLATION") {
		t.Error("Should flag as violation")
	}
}

func TestOrchestratorGuardWorkflow_NonOrchestratorPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	// Non-orchestrator agent (python-pro) - should passthrough without guard checks
	entries := []map[string]interface{}{
		{"timestamp": 1700000000.0, "content": "AGENT: python-pro", "role": "system"},
		{"timestamp": 1700000001.0, "model": "sonnet"},
		{"timestamp": 1700000100.0, "tool_name": "Edit", "tool_input": map[string]interface{}{
			"file_path": "/tmp/test.py",
		}},
		{"timestamp": 1700000200.0, "content": "Implementation complete", "role": "assistant"},
	}

	createTranscriptFromEntries(t, transcriptPath, entries)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-003",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// Parse agent metadata to detect non-orchestrator
	agentMetadata, _ := routing.ParseTranscriptForMetadata(transcriptPath)

	if agentMetadata.AgentID == "orchestrator" || agentMetadata.AgentID == "architect" {
		t.Fatalf("Test setup error: expected non-orchestrator agent, got: %s", agentMetadata.AgentID)
	}

	// Guard should passthrough for non-orchestrator agents
	response := generateGuardResponseWithAgentCheck(event, agentMetadata)

	if response.Decision != "silent" {
		t.Errorf("Expected silent passthrough, got: %s", response.Decision)
	}
}

func TestOrchestratorGuardWorkflow_EmptyTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty-transcript.jsonl")

	// Empty transcript (edge case - agent crashed immediately?)
	if err := os.WriteFile(transcriptPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-004",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)

	// Empty transcript should allow (no tasks to check)
	response := generateGuardResponse(event, metadata)

	if response.Decision != "allow" {
		t.Errorf("Expected allow for empty transcript, got: %s", response.Decision)
	}
}

func TestOrchestratorGuardWorkflow_MalformedTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed-transcript.jsonl")

	// Malformed JSONL (invalid JSON lines)
	content := `{"timestamp": 1700000000.0, "content": "AGENT: orchestrator"}
invalid json line here
{"timestamp": 1700000100.0, "tool_name": "Bash", "tool_input": {"run_in_background": true}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-005",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	// Should gracefully handle parse errors
	metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)

	response := generateGuardResponse(event, metadata)

	// Graceful degradation: allow on parse error (fail open)
	if response.Decision == "" {
		t.Error("Should produce valid response even on parse error")
	}
}

func TestOrchestratorGuardWorkflow_VeryLongTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "long-transcript.jsonl")

	// Performance test: 1000+ entries
	var entries []map[string]interface{}
	baseTime := 1700000000.0

	entries = append(entries, map[string]interface{}{
		"timestamp": baseTime,
		"content":   "AGENT: orchestrator",
		"role":      "system",
	})

	// Spawn 100 background tasks
	for i := 0; i < 100; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": baseTime + float64(i*10),
			"tool_name": "Bash",
			"tool_input": map[string]interface{}{
				"command":           "gemini-slave mapper 'Task'",
				"run_in_background": true,
			},
		})
		entries = append(entries, map[string]interface{}{
			"timestamp": baseTime + float64(i*10+1),
			"tool_response": map[string]interface{}{
				"task_id": "bg-task-" + string(rune('0'+i)),
			},
		})
	}

	// Collect all tasks
	for i := 0; i < 100; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": baseTime + 2000.0 + float64(i),
			"tool_name": "TaskOutput",
			"tool_input": map[string]interface{}{
				"task_id": "bg-task-" + string(rune('0'+i)),
			},
		})
	}

	createTranscriptFromEntries(t, transcriptPath, entries)

	event := &routing.SubagentStopEvent{
		HookEventName:  "SubagentStop",
		SessionID:      "test-guard-006",
		TranscriptPath: transcriptPath,
		StopHookActive: true,
	}

	start := time.Now()
	metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)
	duration := time.Since(start)

	// Performance check: should parse 1000+ lines in <100ms
	if duration > 100*time.Millisecond {
		t.Errorf("Parsing took too long: %v (expected <100ms)", duration)
	}

	response := generateGuardResponse(event, metadata)

	if response.Decision != "allow" {
		t.Errorf("Expected allow after collecting all tasks, got: %s", response.Decision)
	}
}

func TestOrchestratorGuardWorkflow_ConcurrentTranscriptAccess(t *testing.T) {
	// Test concurrent access to same transcript file
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "concurrent-transcript.jsonl")

	entries := []map[string]interface{}{
		{"timestamp": 1700000000.0, "content": "AGENT: orchestrator", "role": "system"},
		{"timestamp": 1700000001.0, "model": "sonnet"},
	}

	createTranscriptFromEntries(t, transcriptPath, entries)

	// Spawn 10 concurrent readers
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := &routing.SubagentStopEvent{
				HookEventName:  "SubagentStop",
				SessionID:      "test-guard-concurrent",
				TranscriptPath: transcriptPath,
				StopHookActive: true,
			}

			metadata := parseTranscriptForBackgroundTasks(t, transcriptPath)
			response := generateGuardResponse(event, metadata)

			if response.Decision == "" {
				results <- fmt.Errorf("goroutine %d: no decision", id)
				return
			}
			results <- nil
		}(i)
	}

	// Verify all concurrent reads succeeded
	for i := 0; i < 10; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}

// Helper functions (to be implemented in pkg/enforcement)

func parseTranscriptForBackgroundTasks(t *testing.T, path string) *BackgroundTaskMetadata {
	// TODO: Implement in GOgent-077
	return &BackgroundTaskMetadata{}
}

func generateGuardResponse(event *routing.SubagentStopEvent, metadata *BackgroundTaskMetadata) *GuardResponse {
	// TODO: Implement in GOgent-077
	return &GuardResponse{}
}

func generateGuardResponseWithAgentCheck(event *routing.SubagentStopEvent, agentMeta *routing.ParsedAgentMetadata) *GuardResponse {
	// TODO: Implement in GOgent-077
	return &GuardResponse{}
}

type BackgroundTaskMetadata struct {
	UncollectedTasks []string
}

type GuardResponse struct {
	Decision          string
	AdditionalContext string
}
```

**Fixture Definitions**:

Create fixture files in `test/simulation/fixtures/deterministic/orchestrator-guard/`:

**File: `01_allow_all_collected.json`**
```json
{
  "id": "OG001_all_collected",
  "description": "Orchestrator completes with all background tasks collected",
  "setup": {
    "create_dirs": [".claude/tmp"]
  },
  "input": {
    "hook_event_name": "SubagentStop",
    "session_id": "test-og-001",
    "transcript_path": "${TEMP_DIR}/transcript.jsonl",
    "stop_hook_active": true
  },
  "transcript_entries": [
    {"timestamp": 1700000000.0, "content": "AGENT: orchestrator", "role": "system"},
    {"timestamp": 1700000001.0, "model": "sonnet"},
    {"timestamp": 1700000100.0, "tool_name": "Bash", "tool_input": {"run_in_background": true}},
    {"timestamp": 1700000110.0, "tool_response": {"task_id": "bg-1"}},
    {"timestamp": 1700000200.0, "tool_name": "TaskOutput", "tool_input": {"task_id": "bg-1"}}
  ],
  "expected": {
    "decision": "allow",
    "exit_code": 0,
    "stdout_contains": ["background tasks collected"]
  }
}
```

**File: `02_block_uncollected.json`**
```json
{
  "id": "OG002_uncollected_task",
  "description": "Block orchestrator completion when background task uncollected",
  "input": {
    "hook_event_name": "SubagentStop",
    "session_id": "test-og-002",
    "transcript_path": "${TEMP_DIR}/transcript.jsonl",
    "stop_hook_active": true
  },
  "transcript_entries": [
    {"timestamp": 1700000000.0, "content": "AGENT: orchestrator", "role": "system"},
    {"timestamp": 1700000100.0, "tool_name": "Bash", "tool_input": {"run_in_background": true}},
    {"timestamp": 1700000110.0, "tool_response": {"task_id": "bg-orphan"}}
  ],
  "expected": {
    "decision": "block",
    "exit_code": 0,
    "stdout_contains": ["VIOLATION", "bg-orphan", "uncollected"]
  }
}
```

**File: `03_non_orchestrator_passthrough.json`**
```json
{
  "id": "OG003_passthrough",
  "description": "Non-orchestrator agents bypass guard checks",
  "input": {
    "hook_event_name": "SubagentStop",
    "session_id": "test-og-003",
    "transcript_path": "${TEMP_DIR}/transcript.jsonl",
    "stop_hook_active": true
  },
  "transcript_entries": [
    {"timestamp": 1700000000.0, "content": "AGENT: python-pro", "role": "system"},
    {"timestamp": 1700000100.0, "tool_name": "Edit", "tool_input": {"file_path": "/tmp/test.py"}}
  ],
  "expected": {
    "decision": "silent",
    "exit_code": 0
  }
}
```

**File: `04_empty_transcript.json`**
```json
{
  "id": "OG004_empty",
  "description": "Empty transcript allows completion (no tasks to check)",
  "input": {
    "hook_event_name": "SubagentStop",
    "session_id": "test-og-004",
    "transcript_path": "${TEMP_DIR}/empty.jsonl",
    "stop_hook_active": true
  },
  "transcript_entries": [],
  "expected": {
    "decision": "allow",
    "exit_code": 0
  }
}
```

**File: `05_malformed_transcript.json`**
```json
{
  "id": "OG005_malformed",
  "description": "Gracefully handle malformed JSONL",
  "input": {
    "hook_event_name": "SubagentStop",
    "session_id": "test-og-005",
    "transcript_path": "${TEMP_DIR}/malformed.jsonl",
    "stop_hook_active": true
  },
  "transcript_raw": "invalid json\n{\"timestamp\": 1700000000.0}\nmore garbage",
  "expected": {
    "exit_code": 0,
    "stdout_not_contains": ["panic", "fatal"]
  }
}
```

**Acceptance Criteria**:
- [ ] All tests use realistic JSONL transcript format (not pseudo-syntax)
- [ ] Tests use simulation harness patterns from `test/simulation/harness/`
- [ ] Fixture files created in `test/simulation/fixtures/deterministic/orchestrator-guard/`
- [ ] Tests cover: allow all collected, block uncollected, non-orchestrator passthrough
- [ ] Edge cases tested: empty transcript, malformed JSONL, very long transcript, concurrent access
- [ ] Helper functions integrated with GOgent-077 implementation
- [ ] `go test ./pkg/enforcement` passes
- [ ] Integration test demonstrates full event → parse → guard → response workflow

**Why This Matters**:
- Ensures orchestrator-guard correctly enforces background task collection
- Uses realistic transcript format matching actual Claude Code output
- Leverages simulation harness for consistent test infrastructure
- Provides comprehensive edge case coverage for production reliability

**Implementation Notes**:
- Helper functions (`parseTranscriptForBackgroundTasks`, `generateGuardResponse`) will be implemented in GOgent-077
- Fixture files use `${TEMP_DIR}` substitution handled by simulation harness
- Tests follow existing patterns from `pkg/workflow/integration_test.go`
- JSONL format matches Claude Code's actual transcript structure (validated in GOgent-063a)

---
