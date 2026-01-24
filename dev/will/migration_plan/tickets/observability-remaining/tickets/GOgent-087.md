---
id: GOgent-087
title: PostToolUse Event for Benchmarking
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-069"]
priority: high
week: 4
tags: ["benchmark-logger", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-087: PostToolUse Event for Benchmarking

**Time**: 1 hour
**Dependencies**: GOgent-069

**Task**:
Parse PostToolUse events with emphasis on timing and performance metrics.

**File**: `pkg/observability/benchmark_events.go`

**Imports**:
```go
package observability

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// BenchmarkEvent represents a tool execution event with metrics
type BenchmarkEvent struct {
	Type          string `json:"type"`           // "post-tool-use"
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`      // Tool being executed
	ToolCategory  string `json:"tool_category"`  // "file", "execution", "search"
	Duration      int    `json:"duration_ms"`    // Execution time in milliseconds
	InputTokens   int    `json:"input_tokens"`   // Tokens consumed (if applicable)
	OutputTokens  int    `json:"output_tokens"`  // Tokens produced
	Model         string `json:"model"`          // Model tier used
	Tier          string `json:"tier"`           // "haiku", "sonnet", "opus"
	Success       bool   `json:"success"`        // Whether tool succeeded
	SessionID     string `json:"session_id"`     // Session identifier
	Timestamp     int64  `json:"timestamp_unix"` // Unix timestamp
}

// ParseBenchmarkEvent reads tool execution event from STDIN
func ParseBenchmarkEvent(r io.Reader, timeout time.Duration) (*BenchmarkEvent, error) {
	type result struct {
		event *BenchmarkEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[benchmark] Failed to read STDIN: %w", err)}
			return
		}

		var event BenchmarkEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[benchmark] Failed to parse JSON: %w", err)}
			return
		}

		// Set timestamp if not provided
		if event.Timestamp == 0 {
			event.Timestamp = time.Now().Unix()
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[benchmark] STDIN read timeout after %v", timeout)
	}
}

// TotalTokens returns input + output tokens
func (e *BenchmarkEvent) TotalTokens() int {
	return e.InputTokens + e.OutputTokens
}

// EstimatedCost calculates rough cost based on tier and tokens
func (e *BenchmarkEvent) EstimatedCost() float64 {
	var costPer1K float64

	switch e.Tier {
	case "haiku":
		costPer1K = 0.0005
	case "sonnet":
		costPer1K = 0.009
	case "opus":
		costPer1K = 0.045
	default:
		return 0.0
	}

	tokens := float64(e.TotalTokens())
	return (tokens / 1000.0) * costPer1K
}
```

**Tests**: `pkg/observability/benchmark_events_test.go`

```go
package observability

import (
	"strings"
	"testing"
	"time"
)

func TestParseBenchmarkEvent(t *testing.T) {
	jsonInput := `{
		"type": "post-tool-use",
		"hook_event_name": "PostToolUse",
		"tool_name": "Read",
		"tool_category": "file",
		"duration_ms": 150,
		"input_tokens": 1024,
		"output_tokens": 512,
		"model": "haiku",
		"tier": "haiku",
		"success": true,
		"session_id": "sess-123",
		"timestamp_unix": 1234567890
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseBenchmarkEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Read" {
		t.Errorf("Expected Read, got: %s", event.ToolName)
	}

	if event.Duration != 150 {
		t.Errorf("Expected 150ms, got: %d", event.Duration)
	}

	if !event.Success {
		t.Error("Should indicate success")
	}
}

func TestTotalTokens(t *testing.T) {
	event := &BenchmarkEvent{
		InputTokens:  1024,
		OutputTokens: 512,
	}

	total := event.TotalTokens()
	if total != 1536 {
		t.Errorf("Expected 1536 total tokens, got: %d", total)
	}
}

func TestEstimatedCost(t *testing.T) {
	tests := []struct {
		tier          string
		inputTokens   int
		outputTokens  int
		expectedRange float64
	}{
		{"haiku", 1000, 0, 0.0005},
		{"sonnet", 1000, 0, 0.009},
		{"opus", 1000, 0, 0.045},
	}

	for _, tc := range tests {
		event := &BenchmarkEvent{
			Tier:         tc.tier,
			InputTokens:  tc.inputTokens,
			OutputTokens: tc.outputTokens,
		}

		cost := event.EstimatedCost()
		if cost != tc.expectedRange {
			t.Errorf("Tier %s: expected %f, got %f", tc.tier, tc.expectedRange, cost)
		}
	}
}

func TestEstimatedCost_Unknown(t *testing.T) {
	event := &BenchmarkEvent{
		Tier: "unknown",
	}

	cost := event.EstimatedCost()
	if cost != 0.0 {
		t.Errorf("Unknown tier should cost 0, got: %f", cost)
	}
}
```

**Acceptance Criteria**:
- [ ] `ParseBenchmarkEvent()` reads tool execution events
- [ ] `TotalTokens()` sums input and output tokens
- [ ] `EstimatedCost()` calculates cost by tier
- [ ] Sets timestamp if not provided
- [ ] Implements 5s timeout
- [ ] Tests verify parsing, totals, cost estimation
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Metrics capture enables performance analysis and cost optimization.

---
