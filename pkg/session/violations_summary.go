package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// FormatViolationsSummary reads routing violations from a JSONL file and formats them
// for human-readable display. Returns the most recent violations first, limited to maxLines.
//
// Returns:
//   - nil, nil: File doesn't exist (normal condition)
//   - empty slice, nil: File exists but is empty
//   - formatted strings, nil: Violations formatted successfully
//   - nil, error: File exists but couldn't be read (permission error, etc.)
func FormatViolationsSummary(violationsPath string, maxLines int) ([]string, error) {
	file, err := os.Open(violationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open violations file %s: %w", violationsPath, err)
	}
	defer file.Close()

	// Load all violations from JSONL
	var violations []*routing.Violation
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var v routing.Violation
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			// Skip malformed lines but continue processing
			continue
		}
		violations = append(violations, &v)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading violations file %s: %w", violationsPath, err)
	}

	// Empty file case
	if len(violations) == 0 {
		return []string{}, nil
	}

	// Determine effective limit (non-positive means "return all")
	effectiveLimit := len(violations)
	if maxLines > 0 && maxLines < len(violations) {
		effectiveLimit = maxLines
	}

	// Take last N violations (most recent first)
	startIdx := len(violations) - effectiveLimit

	// Format violations in reverse order (most recent first)
	result := make([]string, 0, effectiveLimit)
	for i := len(violations) - 1; i >= startIdx; i-- {
		formatted := formatViolation(violations[i])
		result = append(result, formatted)
	}

	return result, nil
}

// ViolationCluster represents a group of violations of the same type.
// Used for pattern detection in routing violations.
type ViolationCluster struct {
	Type    string               // ViolationType value
	Count   int                  // Total occurrences
	Samples []*routing.Violation // First 3 violations as representative samples
}

// ClusterViolationsByType groups violations by their ViolationType field.
// Returns a map where keys are violation types and values are clusters
// containing the count and first 3 samples of each type.
//
// Returns:
//   - Empty map if violations is nil or empty
//   - Map with one entry per unique ViolationType otherwise
func ClusterViolationsByType(violations []*routing.Violation) map[string]*ViolationCluster {
	result := make(map[string]*ViolationCluster)

	if len(violations) == 0 {
		return result
	}

	for _, v := range violations {
		if v == nil {
			continue
		}

		cluster, exists := result[v.ViolationType]
		if !exists {
			cluster = &ViolationCluster{
				Type:    v.ViolationType,
				Count:   0,
				Samples: make([]*routing.Violation, 0, 3),
			}
			result[v.ViolationType] = cluster
		}

		cluster.Count++

		// Keep first 3 violations as samples
		if len(cluster.Samples) < 3 {
			cluster.Samples = append(cluster.Samples, v)
		}
	}

	return result
}

// formatViolation converts a routing.Violation into a human-readable string.
// Format varies by violation type for clarity.
func formatViolation(v *routing.Violation) string {
	switch v.ViolationType {
	case "tool_permission":
		return fmt.Sprintf("- Tool permission: Tier attempted **%s** (allowed: %s)", v.Tool, v.Allowed)

	case "blocked_task_opus":
		return fmt.Sprintf("- Einstein blocking: Attempted Task(model: opus) with agent **%s**", v.Agent)

	case "subagent_type_mismatch":
		return fmt.Sprintf("- Subagent type: Agent **%s** - %s", v.Agent, v.Reason)

	default:
		return fmt.Sprintf("- %s: %s", v.ViolationType, v.Reason)
	}
}

