package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

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

// AgentViolationCluster represents violations grouped by agent name.
// Enables agent-specific debugging by showing which agents have configuration issues.
type AgentViolationCluster struct {
	Agent      string               // Agent name (or "unknown" if empty)
	TotalCount int                  // Total violations for this agent
	ByType     map[string]int       // ViolationType -> count
	Samples    []*routing.Violation // First 3 violations as representative samples
}

// TrendAnalysis represents the temporal trend of violations over a session.
// Compares early vs late violations to detect improvement or degradation patterns.
type TrendAnalysis struct {
	EarlyCount int    // Violations in first half of session
	LateCount  int    // Violations in second half of session
	Trend      string // "improving", "stable", "worsening", "insufficient_data"
	Message    string // Human-readable explanation of the trend
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

// ClusterViolationsByAgent groups violations by their Agent field.
// Returns a map where keys are agent names and values are clusters
// containing the count, violation types breakdown, and first 3 samples.
//
// If the Agent field is empty, the violation is grouped under "unknown".
//
// Returns:
//   - Empty map if violations is nil or empty
//   - Map with one entry per unique Agent otherwise
func ClusterViolationsByAgent(violations []*routing.Violation) map[string]*AgentViolationCluster {
	result := make(map[string]*AgentViolationCluster)

	if len(violations) == 0 {
		return result
	}

	for _, v := range violations {
		if v == nil {
			continue
		}

		agent := v.Agent
		if agent == "" {
			agent = "unknown"
		}

		cluster, exists := result[agent]
		if !exists {
			cluster = &AgentViolationCluster{
				Agent:      agent,
				TotalCount: 0,
				ByType:     make(map[string]int),
				Samples:    make([]*routing.Violation, 0, 3),
			}
			result[agent] = cluster
		}

		cluster.TotalCount++
		cluster.ByType[v.ViolationType]++

		// Keep first 3 violations as samples
		if len(cluster.Samples) < 3 {
			cluster.Samples = append(cluster.Samples, v)
		}
	}

	return result
}

// AnalyzeViolationTrend analyzes the temporal distribution of violations.
// It parses RFC3339 timestamps, finds the session midpoint, and compares
// early vs late violation counts to determine if behavior is improving.
//
// Returns:
//   - "improving" if late violations < early violations
//   - "worsening" if late violations > early violations
//   - "stable" if late violations == early violations
//   - "insufficient_data" if fewer than 2 violations
func AnalyzeViolationTrend(violations []*routing.Violation) *TrendAnalysis {
	result := &TrendAnalysis{
		EarlyCount: 0,
		LateCount:  0,
		Trend:      "insufficient_data",
		Message:    "Not enough violations to analyze trend (need at least 2)",
	}

	// Filter out nil violations and those with invalid/empty timestamps
	type timestampedViolation struct {
		v *routing.Violation
		t time.Time
	}
	var valid []timestampedViolation

	for _, v := range violations {
		if v == nil || v.Timestamp == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, v.Timestamp)
		if err != nil {
			continue
		}
		valid = append(valid, timestampedViolation{v: v, t: t})
	}

	// Need at least 2 violations for meaningful trend analysis
	if len(valid) < 2 {
		if len(valid) == 1 {
			result.Message = "Only 1 violation - insufficient data for trend analysis"
		}
		return result
	}

	// Sort by timestamp (ascending - oldest first)
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].t.Before(valid[j].t)
	})

	// Calculate midpoint time
	firstTime := valid[0].t
	lastTime := valid[len(valid)-1].t

	// Handle edge case: all violations have the same timestamp
	if firstTime.Equal(lastTime) {
		result.EarlyCount = len(valid)
		result.LateCount = 0
		result.Trend = "stable"
		result.Message = "All violations occurred at the same timestamp"
		return result
	}

	// Midpoint is the average of first and last timestamps
	midpoint := firstTime.Add(lastTime.Sub(firstTime) / 2)

	// Count violations in each half
	for _, tv := range valid {
		if tv.t.Before(midpoint) || tv.t.Equal(midpoint) {
			result.EarlyCount++
		} else {
			result.LateCount++
		}
	}

	// Determine trend
	switch {
	case result.LateCount < result.EarlyCount:
		result.Trend = "improving"
		result.Message = fmt.Sprintf("Violation rate decreased: %d early vs %d late", result.EarlyCount, result.LateCount)
	case result.LateCount > result.EarlyCount:
		result.Trend = "worsening"
		result.Message = fmt.Sprintf("Violation rate increased: %d early vs %d late", result.EarlyCount, result.LateCount)
	default:
		result.Trend = "stable"
		result.Message = fmt.Sprintf("Violation rate unchanged: %d early vs %d late", result.EarlyCount, result.LateCount)
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

