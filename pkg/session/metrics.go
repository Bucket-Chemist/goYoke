package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// CollectSessionMetrics gathers statistics from session logs and temp files.
// Returns SessionMetrics with counts from tool counters, error log, and violations log.
// Missing files are not errors - returns 0 for those metrics.
func CollectSessionMetrics(sessionID string) (*SessionMetrics, error) {
	metrics := &SessionMetrics{
		SessionID: sessionID,
	}

	// Count tool calls from temp counters
	toolCount, err := countToolCalls()
	if err == nil {
		metrics.ToolCalls = toolCount
	}

	// Count errors from error log
	errorCount, err := countLogLines(getErrorLogPath())
	if err == nil {
		metrics.ErrorsLogged = errorCount
	}

	// Count routing violations
	violationCount, err := countLogLines(config.GetViolationsLogPath())
	if err == nil {
		metrics.RoutingViolations = violationCount
	}

	return metrics, nil
}

// countToolCalls reads the single tool-counter file containing the integer count.
// New format: single integer value (atomically incremented by config.IncrementToolCount).
// Returns 0 if counter file doesn't exist (normal for first session).
func countToolCalls() (int, error) {
	// Use config package's GetToolCount which handles the single-file format
	count, err := config.GetToolCount()
	if err != nil {
		return 0, fmt.Errorf("failed to read tool count: %w", err)
	}
	return count, nil
}

// countLogLines counts all lines in a file, matching bash wc -l behavior.
// Returns 0 if file doesn't exist (not an error).
// Counts ALL lines including empty lines to match bash hook implementation.
func countLogLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // Missing file is normal
		}
		return 0, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	count := 0
	scanner := newSessionScanner(file)
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading %s: %w", path, err)
	}

	return count, nil
}

// getErrorLogPath returns the path to the error patterns log.
// Uses XDG-compliant location from config package.
func getErrorLogPath() string {
	return filepath.Join(config.GetgoYokeDir(), "claude-error-patterns.jsonl")
}
