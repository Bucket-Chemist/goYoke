---
id: GOgent-088
title: Benchmark Metrics Logging
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-087"]
priority: high
week: 4
tags: ["benchmark-logger", "week-4"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-088: Benchmark Metrics Logging

**Time**: 1.5 hours
**Dependencies**: GOgent-087

**Task**:
Store benchmark metrics in JSONL format for analysis.

**File**: `pkg/observability/benchmark_logger.go`

**Imports**:
```go
package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)
```

**Implementation**:
```go
// BenchmarkLog represents logged performance metrics
type BenchmarkLog struct {
	Timestamp      time.Time `json:"timestamp"`
	ToolName       string    `json:"tool_name"`
	ToolCategory   string    `json:"tool_category"`
	Duration       int       `json:"duration_ms"`
	TotalTokens    int       `json:"total_tokens"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	Tier           string    `json:"tier"`
	Success        bool      `json:"success"`
	EstimatedCost  float64   `json:"estimated_cost"`
}

// LogBenchmark writes metrics to JSONL file
func LogBenchmark(event *BenchmarkEvent) error {
	logPath := "/tmp/claude-benchmarks.jsonl"

	log := BenchmarkLog{
		Timestamp:     time.Now().UTC(),
		ToolName:      event.ToolName,
		ToolCategory:  event.ToolCategory,
		Duration:      event.Duration,
		TotalTokens:   event.TotalTokens(),
		InputTokens:   event.InputTokens,
		OutputTokens:  event.OutputTokens,
		Tier:          event.Tier,
		Success:       event.Success,
		EstimatedCost: event.EstimatedCost(),
	}

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("[benchmark] Failed to marshal log: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[benchmark] Failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("[benchmark] Failed to write log: %w", err)
	}

	return nil
}

// ReadBenchmarkLogs reads all benchmark logs
func ReadBenchmarkLogs() ([]BenchmarkLog, error) {
	logPath := "/tmp/claude-benchmarks.jsonl"

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return []BenchmarkLog{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[benchmark] Failed to read logs: %w", err)
	}

	var logs []BenchmarkLog
	offset := 0
	content := string(data)

	for {
		// Find next newline
		newlineIdx := -1
		for i := offset; i < len(content); i++ {
			if content[i] == '\n' {
				newlineIdx = i
				break
			}
		}

		if newlineIdx == -1 {
			if offset < len(content) {
				newlineIdx = len(content)
			} else {
				break
			}
		}

		line := content[offset:newlineIdx]
		if line == "" {
			offset = newlineIdx + 1
			continue
		}

		var log BenchmarkLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines
			offset = newlineIdx + 1
			continue
		}

		logs = append(logs, log)
		offset = newlineIdx + 1
	}

	return logs, nil
}

// CalculateSessionStats returns aggregate metrics for session
func CalculateSessionStats(logs []BenchmarkLog) map[string]interface{} {
	if len(logs) == 0 {
		return map[string]interface{}{
			"tool_count":     0,
			"total_duration": 0,
			"total_tokens":   0,
			"total_cost":     0.0,
		}
	}

	var totalDuration int
	var totalTokens int
	var totalCost float64
	toolCounts := make(map[string]int)

	for _, log := range logs {
		totalDuration += log.Duration
		totalTokens += log.TotalTokens
		totalCost += log.EstimatedCost
		toolCounts[log.ToolName]++
	}

	return map[string]interface{}{
		"tool_count":      len(logs),
		"total_duration":  totalDuration,
		"total_tokens":    totalTokens,
		"total_cost":      fmt.Sprintf("$%.4f", totalCost),
		"avg_duration":    totalDuration / len(logs),
		"tool_breakdown":  toolCounts,
	}
}
```

**Tests**: `pkg/observability/benchmark_logger_test.go`

```go
package observability

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogBenchmark(t *testing.T) {
	// Clean up any existing logs
	os.Remove("/tmp/claude-benchmarks.jsonl")

	event := &BenchmarkEvent{
		ToolName:     "Read",
		ToolCategory: "file",
		Duration:     150,
		InputTokens:  1024,
		OutputTokens: 512,
		Tier:         "haiku",
		Success:      true,
	}

	err := LogBenchmark(event)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	// Verify file created
	if _, err := os.Stat("/tmp/claude-benchmarks.jsonl"); os.IsNotExist(err) {
		t.Fatal("Log file should exist")
	}

	// Cleanup
	os.Remove("/tmp/claude-benchmarks.jsonl")
}

func TestReadBenchmarkLogs_Empty(t *testing.T) {
	os.Remove("/tmp/claude-benchmarks.jsonl")

	logs, err := ReadBenchmarkLogs()

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs, got: %d", len(logs))
	}
}

func TestReadBenchmarkLogs(t *testing.T) {
	os.Remove("/tmp/claude-benchmarks.jsonl")

	// Log multiple events
	for i := 0; i < 3; i++ {
		event := &BenchmarkEvent{
			ToolName: "Read",
			Duration: 100 + i*50,
			Tier:     "haiku",
			Success:  true,
		}
		LogBenchmark(event)
	}

	logs, err := ReadBenchmarkLogs()

	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got: %d", len(logs))
	}

	// Cleanup
	os.Remove("/tmp/claude-benchmarks.jsonl")
}

func TestCalculateSessionStats(t *testing.T) {
	logs := []BenchmarkLog{
		{ToolName: "Read", Duration: 100, TotalTokens: 1000},
		{ToolName: "Write", Duration: 200, TotalTokens: 500},
		{ToolName: "Read", Duration: 150, TotalTokens: 800},
	}

	stats := CalculateSessionStats(logs)

	if stats["tool_count"] != 3 {
		t.Errorf("Expected 3 tools, got: %v", stats["tool_count"])
	}

	if stats["total_duration"] != 450 {
		t.Errorf("Expected 450ms total, got: %v", stats["total_duration"])
	}

	if stats["total_tokens"] != 2300 {
		t.Errorf("Expected 2300 tokens total, got: %v", stats["total_tokens"])
	}

	breakdown := stats["tool_breakdown"].(map[string]int)
	if breakdown["Read"] != 2 {
		t.Errorf("Expected 2 Read calls, got: %d", breakdown["Read"])
	}
}
```

**Acceptance Criteria**:
- [ ] `LogBenchmark()` writes to /tmp/claude-benchmarks.jsonl
- [ ] Appends JSONL format
- [ ] Creates file if missing
- [ ] `ReadBenchmarkLogs()` parses all logs correctly
- [ ] Handles missing file gracefully
- [ ] `CalculateSessionStats()` aggregates metrics correctly
- [ ] Tests verify logging, reading, aggregation
- [ ] `go test ./pkg/observability` passes

**Why This Matters**: Logging enables performance analysis, cost optimization, and routing efficiency verification.

---
