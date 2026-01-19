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
