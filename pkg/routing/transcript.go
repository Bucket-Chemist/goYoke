package routing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseTranscript reads a JSONL session transcript file and returns a slice of ToolEvent structs.
// Each line in the file should be a valid JSON object representing a ToolEvent.
//
// Parameters:
//   - transcriptPath: absolute path to the JSONL transcript file
//
// Returns:
//   - []ToolEvent: slice of parsed events (empty slice if file is empty)
//   - error: nil on success, error with context on failure
//
// Error cases:
//   - File not found: returns descriptive error
//   - Malformed JSON: returns error with line number
//   - File read error: returns error with context
//
// Empty lines are skipped silently.
func ParseTranscript(transcriptPath string) ([]ToolEvent, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("[transcript] File not found: %s", transcriptPath)
		}
		return nil, fmt.Errorf("[transcript] Failed to open file: %w", err)
	}
	defer f.Close()

	var events []ToolEvent
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue // Skip empty lines
		}

		var event ToolEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("[transcript] Malformed JSON at line %d: %w", lineNum, err)
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[transcript] Failed to scan file: %w", err)
	}

	return events, nil
}

// AnalyzeToolDistribution counts tool usage across all events.
// Returns a map of tool names to usage counts.
//
// Parameters:
//   - events: slice of ToolEvent structs to analyze
//
// Returns:
//   - map[string]int: tool names mapped to occurrence counts
//
// Edge cases:
//   - nil slice: returns empty map (not nil)
//   - empty slice: returns empty map
//   - unknown tool names: counted like any other tool
//
// Example output:
//
//	map[string]int{
//	    "Read":  30,
//	    "Edit":  10,
//	    "Write": 5,
//	    "Task":  2,
//	    "Bash":  8,
//	}
func AnalyzeToolDistribution(events []ToolEvent) map[string]int {
	distribution := make(map[string]int)

	// Handle nil slice gracefully
	if events == nil {
		return distribution
	}

	for _, event := range events {
		distribution[event.ToolName]++
	}

	return distribution
}

// SessionPhase represents a detected work phase within a session
type SessionPhase struct {
	Phase     string // "discovery", "implementation", "debugging", "delegation", "mixed"
	StartTime int64
	Duration  int64
	ToolCount int
}

// DetectPhases identifies session work phases based on tool usage patterns
// Uses threshold heuristics: 70% for discovery/implementation/delegation, 50% for debugging
func DetectPhases(events []ToolEvent) []SessionPhase {
	if len(events) == 0 {
		return []SessionPhase{}
	}

	// Analyze tool distribution
	dist := AnalyzeToolDistribution(events)

	total := len(events)
	readCount := dist["Read"] + dist["Glob"] + dist["Grep"]
	editCount := dist["Edit"] + dist["Write"]
	taskCount := dist["Task"]
	bashCount := dist["Bash"]

	// Apply heuristics in priority order
	var phase string
	if float64(readCount)/float64(total) >= 0.7 {
		phase = "discovery"
	} else if float64(editCount)/float64(total) >= 0.7 {
		phase = "implementation"
	} else if float64(taskCount)/float64(total) >= 0.7 {
		phase = "delegation"
	} else if float64(bashCount)/float64(total) >= 0.5 {
		phase = "debugging"
	} else {
		phase = "mixed"
	}

	// Calculate time range
	startTime := events[0].CapturedAt
	endTime := events[len(events)-1].CapturedAt

	return []SessionPhase{{
		Phase:     phase,
		StartTime: startTime,
		Duration:  endTime - startTime,
		ToolCount: total,
	}}
}

// TaskTracker provides ID-based tracking of background task spawns and collections.
// Uses map-based tracking to support idempotent duplicate handling and precise state.
type TaskTracker struct {
	SpawnedIDs   map[string]bool // task_id → spawned
	CollectedIDs map[string]bool // task_id → collected
}

// NewTaskTracker creates a new TaskTracker with initialized ID maps.
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		SpawnedIDs:   make(map[string]bool),
		CollectedIDs: make(map[string]bool),
	}
}

// GetUncollected returns a slice of task IDs that were spawned but not collected.
func (t *TaskTracker) GetUncollected() []string {
	uncollected := []string{}
	for id := range t.SpawnedIDs {
		if !t.CollectedIDs[id] {
			uncollected = append(uncollected, id)
		}
	}
	return uncollected
}

// HasUncollected returns true if any spawned tasks have not been collected.
func (t *TaskTracker) HasUncollected() bool {
	for id := range t.SpawnedIDs {
		if !t.CollectedIDs[id] {
			return true
		}
	}
	return false
}

// GetStats returns tracking statistics.
// Returns: spawned count, collected count, uncollected task IDs
func (t *TaskTracker) GetStats() (int, int, []string) {
	return len(t.SpawnedIDs), len(t.CollectedIDs), t.GetUncollected()
}

// BashToolCall represents a Bash tool call with run_in_background parameter.
type BashToolCall struct {
	Command         string `json:"command"`
	RunInBackground bool   `json:"run_in_background"`
}

// TaskOutputCall represents a TaskOutput tool call for collecting background tasks.
type TaskOutputCall struct {
	TaskID string `json:"task_id"`
}

// TranscriptAnalyzer analyzes session transcripts for background task usage patterns.
type TranscriptAnalyzer struct {
	filePath string
	tracker  *TaskTracker
}

// NewTranscriptAnalyzer creates a TranscriptAnalyzer for the given transcript file.
func NewTranscriptAnalyzer(transcriptPath string) *TranscriptAnalyzer {
	return &TranscriptAnalyzer{
		filePath: transcriptPath,
		tracker:  NewTaskTracker(),
	}
}

// Analyze parses the transcript and tracks background task spawns and collections.
// Uses both JSON parsing and regex fallback for robustness.
//
// Returns:
//   - error: nil on success, error with context on file read failure
//
// Graceful handling:
//   - Missing file: returns nil (assumes no tasks)
//   - Malformed JSON: falls back to regex pattern matching
//   - Duplicate IDs: idempotent (same ID can be spawned/collected multiple times)
func (a *TranscriptAnalyzer) Analyze() error {
	// Attempt to open transcript file
	f, err := os.Open(a.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing transcript is not an error - assume no background tasks
			return nil
		}
		return fmt.Errorf("[transcript] Failed to open file: %w", err)
	}
	defer f.Close()

	// Compile regex patterns once
	bgPattern := regexp.MustCompile(`(?i)run_in_background[:\s=]+true`)
	taskIDPattern := regexp.MustCompile(`task_id[:\s=]+"([^"]+)"`)
	collectPattern := regexp.MustCompile(`(?i)TaskOutput.*task_id[:\s=]+"([^"]+)"`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try JSON parsing first
		var event ToolEvent
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			// Successfully parsed as ToolEvent
			a.processToolEvent(&event, taskIDPattern)
			continue
		}

		// JSON parse failed - fall back to regex
		a.processLineRegex(line, bgPattern, taskIDPattern, collectPattern)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("[transcript] Failed to scan file: %w", err)
	}

	return nil
}

// processToolEvent handles structured ToolEvent JSON parsing.
func (a *TranscriptAnalyzer) processToolEvent(event *ToolEvent, taskIDPattern *regexp.Regexp) {
	// Check for background Bash spawns
	if event.ToolName == "Bash" {
		if bg, ok := event.ToolInput["run_in_background"].(bool); ok && bg {
			// Extract task_id directly from ToolInput
			if taskID, ok := event.ToolInput["task_id"].(string); ok && taskID != "" {
				a.tracker.SpawnedIDs[taskID] = true
			}
		}
	}

	// Check for TaskOutput collections
	if event.ToolName == "TaskOutput" {
		if taskID, ok := event.ToolInput["task_id"].(string); ok && taskID != "" {
			a.tracker.CollectedIDs[taskID] = true
		}
	}
}

// processLineRegex handles regex-based pattern matching for prose descriptions.
func (a *TranscriptAnalyzer) processLineRegex(line string, bgPattern, taskIDPattern, collectPattern *regexp.Regexp) {
	// Check for background task spawn pattern
	if bgPattern.MatchString(line) {
		matches := taskIDPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			a.tracker.SpawnedIDs[matches[1]] = true
		}
	}

	// Check for TaskOutput collection pattern
	if matches := collectPattern.FindStringSubmatch(line); len(matches) > 1 {
		a.tracker.CollectedIDs[matches[1]] = true
	}
}

// HasUncollectedTasks returns true if any background tasks remain uncollected.
func (a *TranscriptAnalyzer) HasUncollectedTasks() bool {
	return a.tracker.HasUncollected()
}

// GetSummary returns a human-readable summary of background task tracking.
func (a *TranscriptAnalyzer) GetSummary() string {
	spawned, collected, uncollected := a.tracker.GetStats()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Background Tasks: %d spawned, %d collected\n", spawned, collected))

	if len(uncollected) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  Uncollected Tasks: %d\n", len(uncollected)))
		for _, id := range uncollected {
			sb.WriteString(fmt.Sprintf("  - %s\n", id))
		}
	} else if spawned > 0 {
		sb.WriteString("✅ All background tasks collected\n")
	} else {
		sb.WriteString("No background tasks detected\n")
	}

	return sb.String()
}

// GetUncollectedList returns a formatted list of uncollected task IDs.
// Returns empty string if all tasks collected.
func (a *TranscriptAnalyzer) GetUncollectedList() string {
	uncollected := a.tracker.GetUncollected()
	if len(uncollected) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, id := range uncollected {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(id)
	}

	return sb.String()
}
