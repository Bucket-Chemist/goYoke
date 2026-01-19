package session

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
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

// countToolCalls counts total tool calls from /tmp/claude-tool-counter-* files.
// Each file contains a count on the first line. Returns 0 if no counter files exist.
func countToolCalls() (int, error) {
	pattern := "/tmp/claude-tool-counter-*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("glob failed for %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return 0, nil // No counter files is normal
	}

	total := 0
	for _, path := range matches {
		count, err := countLogLines(path)
		if err == nil {
			total += count
		}
		// Ignore errors reading individual counter files
	}

	return total, nil
}

// countLogLines counts non-empty lines in a file.
// Returns 0 if file doesn't exist (not an error).
// Ignores empty lines and lines with only whitespace.
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
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading %s: %w", path, err)
	}

	return count, nil
}

// getErrorLogPath returns the path to the error patterns log.
// Currently uses /tmp location - may move to XDG-compliant location later.
func getErrorLogPath() string {
	return "/tmp/claude-error-patterns.jsonl"
}
