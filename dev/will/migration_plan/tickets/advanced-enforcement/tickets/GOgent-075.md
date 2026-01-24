---
id: GOgent-075
title: SubagentStop Event Parsing for Orchestrator
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-063"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 5
---

### GOgent-075: SubagentStop Event Parsing for Orchestrator

**Time**: 1 hour (reduced from 1.5h - reusing existing infrastructure)
**Dependencies**: GOgent-063

**Task**:
Add orchestrator/architect detection logic to `pkg/routing/` package using existing SubagentStop event schema and transcript parsing.

**Context**:
GOgent-063a research confirmed agent metadata is NOT directly available in SubagentStop events. Must parse transcript file to extract agent information. All required types and functions already exist in `pkg/routing/events.go`.

**File**: `pkg/routing/orchestrator.go` (new file in existing package)

**Imports**:
```go
package routing

import (
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// ParseOrchestratorStopEvent reads SubagentStop event and extracts agent metadata
func ParseOrchestratorStopEvent(r io.Reader, timeout time.Duration) (*ParsedAgentMetadata, error) {
	// Use existing SubagentStopEvent parser
	event, err := ParseSubagentStopEvent(r, timeout)
	if err != nil {
		return nil, fmt.Errorf("[orchestrator-guard] Failed to parse event: %w", err)
	}

	// Extract agent metadata from transcript
	metadata, err := ParseTranscriptForMetadata(event.TranscriptPath)
	if err != nil {
		return nil, fmt.Errorf("[orchestrator-guard] Failed to parse transcript: %w", err)
	}

	return metadata, nil
}

// IsOrchestratorType checks if parsed metadata represents orchestrator/architect agent
func IsOrchestratorType(metadata *ParsedAgentMetadata) bool {
	return metadata.AgentID == "orchestrator" || metadata.AgentID == "architect"
}
```

**Tests**: `pkg/routing/orchestrator_test.go`

```go
package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseOrchestratorStopEvent(t *testing.T) {
	// Create temporary transcript file
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	transcriptContent := `{"role": "assistant", "content": "AGENT: orchestrator", "timestamp": 1000}
{"role": "assistant", "content": "some work", "timestamp": 2000}
{"model": "sonnet", "timestamp": 3000}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create SubagentStop event JSON
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session",
		"transcript_path": "` + transcriptPath + `",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(jsonInput)
	metadata, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.AgentID != "orchestrator" {
		t.Errorf("Expected agent_id='orchestrator', got: %s", metadata.AgentID)
	}

	if metadata.AgentModel != "sonnet" {
		t.Errorf("Expected model='sonnet', got: %s", metadata.AgentModel)
	}

	if !IsOrchestratorType(metadata) {
		t.Error("Should identify as orchestrator type")
	}
}

func TestIsOrchestratorType(t *testing.T) {
	tests := []struct {
		agentID string
		isOrch  bool
	}{
		{"orchestrator", true},
		{"architect", true},
		{"python-pro", false},
		{"code-reviewer", false},
		{"", false},
	}

	for _, tc := range tests {
		metadata := &ParsedAgentMetadata{AgentID: tc.agentID}
		if got := IsOrchestratorType(metadata); got != tc.isOrch {
			t.Errorf("AgentID %s: expected %v, got %v", tc.agentID, tc.isOrch, got)
		}
	}
}

func TestParseOrchestratorStopEvent_MissingTranscript(t *testing.T) {
	jsonInput := `{
		"hook_event_name": "SubagentStop",
		"session_id": "test-session",
		"transcript_path": "/nonexistent/transcript.jsonl",
		"stop_hook_active": true
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for missing transcript file")
	}
}

func TestParseOrchestratorStopEvent_InvalidJSON(t *testing.T) {
	reader := strings.NewReader("not valid json")
	_, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
```

**Acceptance Criteria**:
- [ ] `ParseOrchestratorStopEvent()` uses existing `ParseSubagentStopEvent()` and `ParseTranscriptForMetadata()`
- [ ] `IsOrchestratorType()` correctly identifies orchestrator/architect agents
- [ ] Implementation reuses types from `pkg/routing/events.go` (no new event structs)
- [ ] Tests verify parsing and type detection with actual transcript files
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Orchestrator-guard (GOgent-076) needs to detect orchestrator/architect completions to trigger follow-up actions.

**Design Note**: This ticket deliberately avoids creating new event types or parsing logic. All infrastructure already exists in `pkg/routing/events.go` and `pkg/routing/transcript.go`. We're just adding orchestrator-specific convenience functions.

---
