---
id: GOgent-089
title: Integration Tests for benchmark-logger
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-088"]
priority: high
week: 4
tags: ["benchmark-logger", "week-4"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-089: Integration Tests for benchmark-logger

**Time**: 1 hour
**Dependencies**: GOgent-088

**Task**:
End-to-end tests for benchmark logging workflow.

**File**: `pkg/observability/benchmark_integration_test.go`

```go
package observability

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkWorkflow_LogAndAnalyze(t *testing.T) {
	os.Remove("/tmp/claude-benchmarks.jsonl")
	defer os.Remove("/tmp/claude-benchmarks.jsonl")

	// Simulate multiple tool calls
	events := []*BenchmarkEvent{
		{
			ToolName:     "Glob",
			ToolCategory: "search",
			Duration:     50,
			InputTokens:  512,
			OutputTokens: 256,
			Tier:         "haiku",
			Success:      true,
		},
		{
			ToolName:     "Read",
			ToolCategory: "file",
			Duration:     100,
			InputTokens:  1024,
			OutputTokens: 1024,
			Tier:         "haiku",
			Success:      true,
		},
		{
			ToolName:     "Edit",
			ToolCategory: "file",
			Duration:     150,
			InputTokens:  2048,
			OutputTokens: 512,
			Tier:         "sonnet",
			Success:      true,
		},
	}

	// Log all events
	for _, event := range events {
		if err := LogBenchmark(event); err != nil {
			t.Fatalf("Failed to log: %v", err)
		}
	}

	// Read back logs
	logs, err := ReadBenchmarkLogs()
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

func TestBenchmarkWorkflow_CostTracking(t *testing.T) {
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
		event := &BenchmarkEvent{
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
```

**Acceptance Criteria**:
- [ ] Full workflow (event → log → read → analyze) works
- [ ] Multiple events logged and retrieved correctly
- [ ] Statistics aggregation correct
- [ ] Cost calculation verified across tiers
- [ ] JSON JSONL format valid
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Integration tests ensure metrics pipeline works end-to-end.

---
