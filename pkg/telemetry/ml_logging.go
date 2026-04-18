package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// LogMLToolEvent writes ML routing tool event metrics to JSONL files (dual-write: global + project)
// Uses routing.PostToolEvent and config.GetMLToolEventsPathWithProjectDir() for XDG compliance
// with GOGENT_PROJECT_DIR override support for test isolation
func LogMLToolEvent(event *routing.PostToolEvent, projectDir string) error {
	// Marshal PostToolEvent directly to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[ml-logging] Failed to marshal event: %w", err)
	}

	// Add newline for JSONL format
	jsonlLine := append(data, '\n')

	// Write to global path using config helper (respects GOGENT_PROJECT_DIR)
	globalPath := config.GetMLToolEventsPathWithProjectDir()
	if err := appendMLToolEvent(globalPath, jsonlLine); err != nil {
		return err
	}

	// Write to project-scoped path (if directory exists and not already covered by GOGENT_PROJECT_DIR)
	// Skip project-scoped write if GOGENT_PROJECT_DIR is set to avoid duplicate writes
	if os.Getenv("GOGENT_PROJECT_DIR") == "" {
		projectPath := filepath.Join(config.ProjectMemoryDir(projectDir), "ml-tool-events.jsonl")
		if dirExists(filepath.Dir(projectPath)) {
			appendMLToolEvent(projectPath, jsonlLine) // Ignore errors for project-scoped writes
		}
	}

	return nil
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// appendMLToolEvent appends a JSONL line to a file, creating parent directories if needed
func appendMLToolEvent(path string, data []byte) error {
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

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("[ml-logging] Failed to write log: %w", err)
	}

	return nil
}

// ReadMLToolEvents reads all ML tool events from the global path
// Returns routing.PostToolEvent slice directly
// Respects GOGENT_PROJECT_DIR for test isolation
func ReadMLToolEvents() ([]routing.PostToolEvent, error) {
	logPath := config.GetMLToolEventsPathWithProjectDir()

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
		totalCost += EstimatedCost(&event)
		toolCounts[event.ToolName]++
	}

	return map[string]interface{}{
		"event_count":    len(events),
		"total_duration": totalDuration,
		"total_cost":     fmt.Sprintf("$%.4f", totalCost),
		"avg_duration":   totalDuration / int64(len(events)),
		"tool_breakdown": toolCounts,
	}
}
