---
id: GOgent-078
title: Integration Tests for orchestrator-guard
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-077"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-078: Integration Tests for orchestrator-guard

**Time**: 1.5 hours
**Dependencies**: GOgent-077

**Task**:
End-to-end tests for orchestrator-guard workflow.

**File**: `pkg/enforcement/orchestrator_integration_test.go`

```go
package enforcement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOrchestratorGuardWorkflow_AllowCompletion(t *testing.T) {
	// Full workflow: event → analysis → response
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with proper fan-in/fan-out
	content := `# Transcript

Bash({..., run_in_background: true})
Bash({..., run_in_background: true})

TaskOutput({task_id: "bg-1"})
TaskOutput({task_id: "bg-2"})

Workflow complete.
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	// Parse event
	eventJSON := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"transcript_path": "` + transcriptPath + `"
	}`

	event := &OrchestratorStopEvent{
		AgentID:        "orchestrator",
		TranscriptPath: transcriptPath,
	}

	// Analyze
	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	// Generate response
	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Error("Should allow completion when all tasks collected")
	}
}

func TestOrchestratorGuardWorkflow_BlockCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with uncollected background tasks
	content := `# Transcript

Bash({..., run_in_background: true, task_id: "bg-1"})
Bash({..., run_in_background: true, task_id: "bg-2"})

// ERROR: Only one TaskOutput!
TaskOutput({task_id: "bg-1"})

// bg-2 was never collected - VIOLATION
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	event := &OrchestratorStopEvent{
		AgentID:        "orchestrator",
		TranscriptPath: transcriptPath,
	}

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "block" {
		t.Error("Should block completion when tasks uncollected")
	}

	if !strings.Contains(response.AdditionalContext, "bg-2") {
		t.Error("Should mention uncollected task ID")
	}

	jsonResponse := response.FormatJSON()
	if !strings.Contains(jsonResponse, "block") {
		t.Error("JSON response should contain block decision")
	}
}

func TestOrchestratorGuardWorkflow_NoBackgroundTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with no background tasks
	content := `# Transcript

Direct task execution.
No background spawning.
Done.
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	event := &OrchestratorStopEvent{
		AgentID:        "architect",
		TranscriptPath: transcriptPath,
	}

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Error("Should allow when no background tasks")
	}
}
```

**Acceptance Criteria**:
- [ ] Full workflow (event → analysis → response) works
- [ ] Allows completion when all background tasks collected
- [ ] Blocks completion when tasks uncollected
- [ ] No false positives on direct (non-background) execution
- [ ] JSON response valid
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Integration tests ensure orchestrator-guard catches real violations.

---
