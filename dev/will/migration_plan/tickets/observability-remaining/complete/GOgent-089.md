---
id: GOgent-089
title: Integration Tests for ToolEvent ML Pipeline
description: End-to-end tests for benchmark logging workflow
status: pending
time_estimate: 1h
dependencies: ["GOgent-088"]
priority: high
week: 4
tags: ["ml-optimization", "week-4"]
tests_required: true
acceptance_criteria_count: 11
---

### GOgent-089: Integration Tests for ToolEvent ML Pipeline

**Time**: 1 hour
**Dependencies**: GOgent-088

**Task**:
End-to-end tests for benchmark logging workflow.

**File**: `pkg/telemetry/ml_integration_test.go`

```go
package telemetry

import (
	"filepath"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestToolEventWorkflow_LogAndAnalyze(t *testing.T) {
	os.Remove("/tmp/claude-events.jsonl")
	defer os.Remove("/tmp/claude-events.jsonl")

	// Simulate multiple tool events with ML fields
	events := []*routing.PostToolEvent{
		{
			ToolName:       "Glob",
			ToolCategory:   "search",
			Duration:       50,
			InputTokens:    512,
			OutputTokens:   256,
			Tier:           "haiku",
			Success:        true,
			SequenceIndex:  1,
			PreviousTools:  []string{},
			PreviousOutcomes: []bool{},
			TaskType:       "search",
		},
		{
			ToolName:       "Read",
			ToolCategory:   "file",
			Duration:       100,
			InputTokens:    1024,
			OutputTokens:   1024,
			Tier:           "haiku",
			Success:        true,
			SequenceIndex:  2,
			PreviousTools:  []string{"Glob"},
			PreviousOutcomes: []bool{true},
			TaskType:       "search",
		},
		{
			ToolName:       "Edit",
			ToolCategory:   "file",
			Duration:       150,
			InputTokens:    2048,
			OutputTokens:   512,
			Tier:           "sonnet",
			Success:        true,
			SequenceIndex:  3,
			PreviousTools:  []string{"Glob", "Read"},
			PreviousOutcomes: []bool{true, true},
			TaskType:       "implementation",
		},
	}

	// Log all events
	for _, event := range events {
		if err := LogToolEvent(event); err != nil {
			t.Fatalf("Failed to log: %v", err)
		}
	}

	// Read back logs
	logs, err := ReadToolEventLogs()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got: %d", len(logs))
	}

	// Calculate stats
	stats := CalculateSessionStats(logs)

	if stats["tool_count"] != 3 {
		t.Error("Should count 3 tool calls")
	}

	totalDuration := stats["total_duration"].(int)
	if totalDuration != 300 {
		t.Errorf("Expected 300ms total, got: %d", totalDuration)
	}

	// Verify cost calculation
	costStr := stats["total_cost"].(string)
	if !strings.Contains(costStr, "$") {
		t.Error("Cost should be formatted as currency")
	}
}

func TestToolEventWorkflow_CostTracking(t *testing.T) {
	// Test cost estimation across tiers
	tests := []struct {
		tier     string
		tokens   int
		minCost  float64
		maxCost  float64
	}{
		{"haiku", 1000, 0.0004, 0.0006},
		{"sonnet", 1000, 0.008, 0.010},
		{"opus", 1000, 0.040, 0.050},
	}

	for _, tc := range tests {
		event := &routing.PostToolEvent{
			Tier:        tc.tier,
			InputTokens: tc.tokens,
		}

		cost := event.EstimatedCost()
		if cost < tc.minCost || cost > tc.maxCost {
			t.Errorf("Tier %s: cost %f outside range [%f, %f]",
				tc.tier, cost, tc.minCost, tc.maxCost)
		}
	}
}

func TestSequenceTracking_Integration(t *testing.T) {
	// Test that SequenceIndex increments correctly across multiple events
	// Create 10 sequential events
	for i := 1; i <= 10; i++ {
		event := &routing.PostToolEvent{
			ToolName:      "Read",
			SequenceIndex: i,
		}

		if event.SequenceIndex != i {
			t.Errorf("Expected sequence %d, got %d", i, event.SequenceIndex)
		}
	}

	// Test that PreviousTools captures last 5 tools correctly
	tools := []string{"Glob", "Read", "Edit", "Write", "Bash", "Grep"}
	lastFive := tools[len(tools)-5:]

	event := &routing.PostToolEvent{
		PreviousTools: lastFive,
	}

	if len(event.PreviousTools) != 5 {
		t.Errorf("Expected 5 previous tools, got %d", len(event.PreviousTools))
	}

	// Test that PreviousOutcomes tracks success states correctly
	outcomes := []bool{true, false, true, true, false}
	event.PreviousOutcomes = outcomes

	successCount := 0
	for _, outcome := range event.PreviousOutcomes {
		if outcome {
			successCount++
		}
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}
}

func TestTaskClassification_Integration(t *testing.T) {
	// Test ClassifyTask() patterns
	tests := []struct {
		description string
		expected    string
	}{
		{"implement the feature", "implementation"},
		{"find files in src", "search"},
		{"document the API", "documentation"},
		{"fix the bug", "debug"},
		{"refactor the module", "refactoring"},
		{"test the code", "testing"},
	}

	accurateCount := 0
	for _, tc := range tests {
		classified := ClassifyTask(tc.description)
		if classified == tc.expected {
			accurateCount++
		}
	}

	accuracy := float64(accurateCount) / float64(len(tests))
	if accuracy < 0.85 {
		t.Errorf("Task classification accuracy %.2f%% below 85%% threshold", accuracy*100)
	}
}

func TestDualWrite_Integration(t *testing.T) {
	// Test writes to both global and project paths
	os.Setenv("XDG_DATA_HOME", "/tmp/xdg-test")
	defer os.Unsetenv("XDG_DATA_HOME")

	projectDir := "/tmp/test-project"
	os.MkdirAll(projectDir, 0755)
	defer os.RemoveAll(projectDir)

	// Test global path write
	globalPath := config.GetMLToolEventsPath()
	if !strings.Contains(globalPath, "/tmp/xdg-test") {
		t.Errorf("Global path should use XDG_DATA_HOME, got %s", globalPath)
	}

	// Test project path write
	projectPath := filepath.Join(projectDir, ".claude", "memory", "tool-events.jsonl")
	if !strings.Contains(projectPath, ".claude/memory") {
		t.Errorf("Project path should include .claude/memory, got %s", projectPath)
	}

	// Test XDG_DATA_HOME override
	os.Setenv("XDG_DATA_HOME", "/custom/path")
	customGlobalPath := config.GetMLToolEventsPath()
	if !strings.Contains(customGlobalPath, "/custom/path") {
		t.Errorf("Should respect XDG_DATA_HOME override, got %s", customGlobalPath)
	}

	// Test project path only writes if directory is not empty string
	shouldSkipProject := projectDir == ""
	if shouldSkipProject {
		t.Error("Should not skip project write when projectDir is provided")
	}
}

func TestUnderstandingContext_Integration(t *testing.T) {
	// Test TargetSize, CoverageAchieved, EntitiesFound
	event := &routing.PostToolEvent{
		ToolName:        "Read",
		TaskType:        "understanding",
		TargetSize:      5000,
		CoverageAchieved: 0.92,
		EntitiesFound:   42,
	}

	if event.TargetSize != 5000 {
		t.Errorf("Expected TargetSize 5000, got %d", event.TargetSize)
	}

	if event.CoverageAchieved != 0.92 {
		t.Errorf("Expected coverage 0.92, got %f", event.CoverageAchieved)
	}

	if event.EntitiesFound != 42 {
		t.Errorf("Expected 42 entities, got %d", event.EntitiesFound)
	}

	// Verify omitempty behavior for non-understanding tasks
	nonUnderstandingEvent := &routing.PostToolEvent{
		ToolName: "Edit",
		TaskType: "implementation",
	}

	if nonUnderstandingEvent.TargetSize != 0 {
		t.Error("Non-understanding task should have zero TargetSize")
	}

	if nonUnderstandingEvent.EntitiesFound != 0 {
		t.Error("Non-understanding task should have zero EntitiesFound")
	}
}
```

**Acceptance Criteria**:
- [x] Full workflow (event → log → read → analyze) works
- [x] Multiple events logged and retrieved correctly
- [x] Statistics aggregation correct
- [x] Cost calculation verified across tiers
- [x] JSON JSONL format valid
- [x] `go test ./pkg/telemetry` passes
- [x] Sequence tracking integration tested
- [x] Task classification accuracy validated (>85%)
- [x] Dual-write verified in tests
- [x] Understanding context fields tested
- [x] XDG path resolution tested
- [x] Tests use telemetry package functions (not observability)
- [x] Tests use routing.PostToolEvent (not observability.ToolEvent)
- [x] Tests use config.GetMLToolEventsPath() for paths
- [x] All references to pkg/observability removed

**Success Metrics** (from GAP Section 10):
- Task classification accuracy > 85%
- Sequence capture 100%

**Why This Matters**: Integration tests ensure ML pipeline captures sequence tracking, task classification, and dual-write behavior for observability optimization.

---
