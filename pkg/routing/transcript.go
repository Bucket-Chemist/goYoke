package routing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
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
