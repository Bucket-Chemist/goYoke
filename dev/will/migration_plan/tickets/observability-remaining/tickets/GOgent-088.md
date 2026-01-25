---
id: GOgent-088
title: Tool Event Logging with Dual-Write & XDG Compliance
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-086a", "GOgent-087"]
priority: high
week: 4
tags: ["tool-event-logger", "xdg-compliance", "week-4"]
tests_required: true
acceptance_criteria_count: 16
---

### GOgent-088: Tool Event Logging with Dual-Write & XDG Compliance

**Time**: 1.5 hours
**Dependencies**: GOgent-087, GOgent-086a

**Import Note**: This ticket requires GOgent-086a to be completed first as it uses:
- `config.GetMLToolEventsPath()` from pkg/config/paths.go
- `config.GetGOgentDataDir()` for XDG compliance

Ensure these functions exist before implementing.

**Task**:
Log ML tool events in JSONL format with dual-write to global and project-scoped paths, supporting XDG Base Directory specification via config.GetGOgentDataDir().

**File**: `pkg/telemetry/ml_logging.go`

**Imports**:
```go
package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gogent/internal/config"
	"gogent/pkg/routing"
)
```

**Implementation**:
```go
// LogMLToolEvent writes ML routing tool event metrics to JSONL files (dual-write: global + project)
// Uses routing.PostToolEvent and config.GetGOgentDataDir() for XDG compliance
func LogMLToolEvent(event *routing.PostToolEvent, projectDir string) error {
	// Marshal PostToolEvent directly to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[ml-logging] Failed to marshal event: %w", err)
	}

	// Write to global path using config helper
	globalPath := config.GetMLToolEventsPath()
	if err := appendToFile(globalPath, data); err != nil {
		return err
	}

	// Write to project-scoped path (if directory exists)
	projectPath := filepath.Join(projectDir, ".claude", "memory", "ml-tool-events.jsonl")
	if dirExists(filepath.Dir(projectPath)) {
		appendToFile(projectPath, data) // Ignore errors for project-scoped writes
	}

	return nil
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// appendToFile appends a JSON line to a file, creating parent directories if needed
func appendToFile(path string, data []byte) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[ml-logging] Failed to create directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[ml-logging] Failed to open log file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("[ml-logging] Failed to write log: %w", err)
	}

	return nil
}

// ReadMLToolEvents reads all ML tool events from the global path
// Returns routing.PostToolEvent slice directly
func ReadMLToolEvents() ([]routing.PostToolEvent, error) {
	logPath := config.GetMLToolEventsPath()

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return []routing.PostToolEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[ml-logging] Failed to read logs: %w", err)
	}

	var events []routing.PostToolEvent
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

		var event routing.PostToolEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines
			offset = newlineIdx + 1
			continue
		}

		events = append(events, event)
		offset = newlineIdx + 1
	}

	return events, nil
}

// CalculateMLSessionStats returns aggregate metrics for ML session
func CalculateMLSessionStats(events []routing.PostToolEvent) map[string]interface{} {
	if len(events) == 0 {
		return map[string]interface{}{
			"event_count":    0,
			"total_duration": 0,
			"total_cost":     0.0,
		}
	}

	var totalDuration int64
	var totalCost float64
	toolCounts := make(map[string]int)

	for _, event := range events {
		totalDuration += event.DurationMs
		totalCost += event.EstimatedCost
		toolCounts[event.ToolName]++
	}

	return map[string]interface{}{
		"event_count":     len(events),
		"total_duration":  totalDuration,
		"total_cost":      fmt.Sprintf("$%.4f", totalCost),
		"avg_duration":    totalDuration / int64(len(events)),
		"tool_breakdown":  toolCounts,
	}
}
```

**Tests**: `pkg/telemetry/ml_logging_test.go`

```go
package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"gogent/internal/config"
	"gogent/pkg/routing"
)

func TestLogMLToolEvent_GlobalPath(t *testing.T) {
	// Test global path creation and writing
	event := &routing.PostToolEvent{
		ToolName:      "Read",
		DurationMs:    150,
		EstimatedCost: 0.001,
	}

	projectDir := t.TempDir()
	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	// Verify global file created via config helper
	globalPath := config.GetMLToolEventsPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global log file should exist at %s", globalPath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestLogMLToolEvent_ProjectPath(t *testing.T) {
	// Test project path writing when directory exists
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".claude", "memory"), 0755)

	event := &routing.PostToolEvent{
		ToolName:      "Read",
		DurationMs:    150,
		EstimatedCost: 0.001,
	}

	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".claude", "memory", "ml-tool-events.jsonl")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Fatalf("Project log file should exist at %s", projectPath)
	}
}

func TestReadMLToolEvents_Empty(t *testing.T) {
	// Test reading when log doesn't exist
	events, err := ReadMLToolEvents()

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got: %d", len(events))
	}
}

func TestReadMLToolEvents(t *testing.T) {
	// Test reading multiple logged events
	projectDir := t.TempDir()
	for i := 0; i < 3; i++ {
		event := &routing.PostToolEvent{
			ToolName:      "Read",
			DurationMs:    int64(100 + i*50),
			EstimatedCost: 0.001,
		}
		LogMLToolEvent(event, projectDir)
	}

	events, err := ReadMLToolEvents()

	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got: %d", len(events))
	}

	// Cleanup
	globalPath := config.GetMLToolEventsPath()
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestCalculateMLSessionStats(t *testing.T) {
	events := []routing.PostToolEvent{
		{ToolName: "Read", DurationMs: 100, EstimatedCost: 0.001},
		{ToolName: "Write", DurationMs: 200, EstimatedCost: 0.002},
		{ToolName: "Read", DurationMs: 150, EstimatedCost: 0.001},
	}

	stats := CalculateMLSessionStats(events)

	if stats["event_count"] != 3 {
		t.Errorf("Expected 3 events, got: %v", stats["event_count"])
	}

	if stats["total_duration"] != int64(450) {
		t.Errorf("Expected 450ms total, got: %v", stats["total_duration"])
	}

	breakdown := stats["tool_breakdown"].(map[string]int)
	if breakdown["Read"] != 2 {
		t.Errorf("Expected 2 Read calls, got: %d", breakdown["Read"])
	}
}

func TestDualWrite(t *testing.T) {
	// Test that events are written to both global and project paths when available
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".claude", "memory"), 0755)

	event := &routing.PostToolEvent{
		ToolName:      "Read",
		DurationMs:    150,
		EstimatedCost: 0.001,
	}

	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	// Verify both files exist
	globalPath := config.GetMLToolEventsPath()
	projectPath := filepath.Join(projectDir, ".claude", "memory", "ml-tool-events.jsonl")

	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global log file should exist at %s", globalPath)
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Fatalf("Project log file should exist at %s", projectPath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}
```

**Acceptance Criteria**:
- [ ] NO pkg/observability created
- [ ] `LogMLToolEvent()` writes to path from config.GetMLToolEventsPath() (global)
- [ ] `LogMLToolEvent()` writes to `.claude/memory/ml-tool-events.jsonl` (project) when directory exists
- [ ] Uses config.GetGOgentDataDir() for global path resolution
- [ ] Dual-write architecture implemented per GAP 5.2
- [ ] Creates directories if missing (os.MkdirAll with 0755)
- [ ] Function signature: `LogMLToolEvent(event *routing.PostToolEvent, projectDir string) error`
- [ ] `ReadMLToolEvents()` parses all logs correctly from global path
- [ ] `ReadMLToolEvents()` returns `[]routing.PostToolEvent` directly (no custom struct)
- [ ] Handles missing file gracefully (returns empty slice, no error)
- [ ] `CalculateMLSessionStats()` aggregates metrics correctly
- [ ] Tests verify global path creation and writing
- [ ] Tests verify project path writing when directory exists
- [ ] Tests verify dual-write architecture
- [ ] Imports include `gogent/internal/config` and `gogent/pkg/routing`
- [ ] `go test ./pkg/telemetry` passes

**Why This Matters**: ML event logging enables performance analysis, routing efficiency verification, and cost optimization via centralized config path helpers.

---
