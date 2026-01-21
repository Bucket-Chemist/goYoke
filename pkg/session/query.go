package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Query provides programmatic access to session learning artifacts
type Query struct {
	ProjectDir string
}

// NewQuery creates a query instance for the given project directory
func NewQuery(projectDir string) *Query {
	return &Query{ProjectDir: projectDir}
}

// SharpEdgeFilters defines filter criteria for sharp edges
type SharpEdgeFilters struct {
	File       *string // Glob pattern for file matching
	ErrorType  *string // Filter by error type (exact match)
	Severity   *string // Filter by severity level (high/medium/low)
	Unresolved bool    // Only return unresolved edges (ResolvedAt == 0)
	Since      *int64  // Filter by timestamp (edges after this time)
	Limit      int     // Maximum results to return (0 = unlimited)
}

// QuerySharpEdges retrieves sharp edges with optional filters
// Returns all edges if no filters specified
// Missing file returns empty slice (not error)
//
// Example:
//
//	q := NewQuery("/project/dir")
//	severity := "high"
//	edges, err := q.QuerySharpEdges(SharpEdgeFilters{
//	    Severity:   &severity,
//	    Unresolved: true,
//	})
func (q *Query) QuerySharpEdges(filters SharpEdgeFilters) ([]SharpEdge, error) {
	edgesPath := filepath.Join(q.ProjectDir, ".claude", "memory", "pending-learnings.jsonl")

	file, err := os.Open(edgesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SharpEdge{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open pending learnings: %w", err)
	}
	defer file.Close()

	var edges []SharpEdge
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var edge SharpEdge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if filters.File != nil && !matchGlob(edge.File, *filters.File) {
			continue
		}
		if filters.ErrorType != nil && edge.ErrorType != *filters.ErrorType {
			continue
		}
		if filters.Severity != nil && edge.Severity != *filters.Severity {
			continue
		}
		if filters.Unresolved && edge.ResolvedAt != 0 {
			continue
		}
		if filters.Since != nil && edge.Timestamp < *filters.Since {
			continue
		}

		edges = append(edges, edge)

		if filters.Limit > 0 && len(edges) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading pending learnings: %w", err)
	}

	return edges, nil
}

// UserIntentFilters defines filter criteria for user intents
type UserIntentFilters struct {
	Source     *string // Filter by capture source (ask_user, hook_prompt, manual)
	Confidence *string // Filter by confidence level (explicit, inferred, default)
	HasAction  bool    // Only return intents with ActionTaken != ""
	Since      *int64  // Filter by timestamp (intents after this time)
	Limit      int     // Maximum results to return (0 = unlimited)
}

// QueryUserIntents retrieves user intents with optional filters
// Returns all intents if no filters specified
// Missing file returns empty slice (not error)
//
// Example:
//
//	q := NewQuery("/project/dir")
//	source := "ask_user"
//	intents, err := q.QueryUserIntents(UserIntentFilters{
//	    Source:    &source,
//	    HasAction: true,
//	})
func (q *Query) QueryUserIntents(filters UserIntentFilters) ([]UserIntent, error) {
	intentsPath := filepath.Join(q.ProjectDir, ".claude", "memory", "user-intents.jsonl")

	file, err := os.Open(intentsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []UserIntent{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open user intents: %w", err)
	}
	defer file.Close()

	var intents []UserIntent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var intent UserIntent
		if err := json.Unmarshal([]byte(line), &intent); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if filters.Source != nil && intent.Source != *filters.Source {
			continue
		}
		if filters.Confidence != nil && intent.Confidence != *filters.Confidence {
			continue
		}
		if filters.HasAction && intent.ActionTaken == "" {
			continue
		}
		if filters.Since != nil && intent.Timestamp < *filters.Since {
			continue
		}

		intents = append(intents, intent)

		if filters.Limit > 0 && len(intents) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading user intents: %w", err)
	}

	return intents, nil
}

// DecisionFilters defines filter criteria for decisions
type DecisionFilters struct {
	Category *string // Filter by category (architecture, tooling, pattern)
	Impact   *string // Filter by impact level (high, medium, low)
	Since    *int64  // Filter by timestamp (decisions after this time)
	Limit    int     // Maximum results to return (0 = unlimited)
}

// QueryDecisions retrieves decisions with optional filters
// Returns all decisions if no filters specified
// Missing file returns empty slice (not error)
//
// Example:
//
//	q := NewQuery("/project/dir")
//	category := "architecture"
//	decisions, err := q.QueryDecisions(DecisionFilters{
//	    Category: &category,
//	    Impact:   ptr("high"),
//	})
func (q *Query) QueryDecisions(filters DecisionFilters) ([]Decision, error) {
	decisionsPath := filepath.Join(q.ProjectDir, ".claude", "memory", "decisions.jsonl")

	file, err := os.Open(decisionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Decision{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open decisions: %w", err)
	}
	defer file.Close()

	var decisions []Decision
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var d Decision
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if filters.Category != nil && d.Category != *filters.Category {
			continue
		}
		if filters.Impact != nil && d.Impact != *filters.Impact {
			continue
		}
		if filters.Since != nil && d.Timestamp < *filters.Since {
			continue
		}

		decisions = append(decisions, d)

		if filters.Limit > 0 && len(decisions) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading decisions: %w", err)
	}

	return decisions, nil
}

// PreferenceFilters defines filter criteria for preference overrides
type PreferenceFilters struct {
	Category *string // Filter by category (routing, tooling, formatting)
	Scope    *string // Filter by scope (session, project, global)
	Since    *int64  // Filter by timestamp (preferences after this time)
	Limit    int     // Maximum results to return (0 = unlimited)
}

// QueryPreferences retrieves preference overrides with optional filters
// Returns all preferences if no filters specified
// Missing file returns empty slice (not error)
//
// Example:
//
//	q := NewQuery("/project/dir")
//	scope := "project"
//	preferences, err := q.QueryPreferences(PreferenceFilters{
//	    Scope: &scope,
//	})
func (q *Query) QueryPreferences(filters PreferenceFilters) ([]PreferenceOverride, error) {
	preferencesPath := filepath.Join(q.ProjectDir, ".claude", "memory", "preferences.jsonl")

	file, err := os.Open(preferencesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PreferenceOverride{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open preferences: %w", err)
	}
	defer file.Close()

	var preferences []PreferenceOverride
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var p PreferenceOverride
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if filters.Category != nil && p.Category != *filters.Category {
			continue
		}
		if filters.Scope != nil && p.Scope != *filters.Scope {
			continue
		}
		if filters.Since != nil && p.Timestamp < *filters.Since {
			continue
		}

		preferences = append(preferences, p)

		if filters.Limit > 0 && len(preferences) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading preferences: %w", err)
	}

	return preferences, nil
}

// PerformanceFilters defines filter criteria for performance metrics
type PerformanceFilters struct {
	Operation   *string // Filter by operation type
	SlowOnly    bool    // Only return metrics with DurationMs > SlowThresholdMs
	SuccessOnly bool    // Only return successful operations
	FailedOnly  bool    // Only return failed operations
	Since       *int64  // Filter by timestamp (metrics after this time)
	Limit       int     // Maximum results to return (0 = unlimited)
}

// SlowThresholdMs defines the threshold for "slow" operations (1000ms = 1 second)
const SlowThresholdMs = 1000

// QueryPerformance retrieves performance metrics with optional filters
// Returns all metrics if no filters specified
// Missing file returns empty slice (not error)
//
// Example:
//
//	q := NewQuery("/project/dir")
//	metrics, err := q.QueryPerformance(PerformanceFilters{
//	    SlowOnly: true,
//	})
func (q *Query) QueryPerformance(filters PerformanceFilters) ([]PerformanceMetric, error) {
	performancePath := filepath.Join(q.ProjectDir, ".claude", "memory", "performance.jsonl")

	file, err := os.Open(performancePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PerformanceMetric{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open performance metrics: %w", err)
	}
	defer file.Close()

	var metrics []PerformanceMetric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var m PerformanceMetric
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if filters.Operation != nil && m.Operation != *filters.Operation {
			continue
		}
		if filters.SlowOnly && m.DurationMs <= SlowThresholdMs {
			continue
		}
		if filters.SuccessOnly && !m.Success {
			continue
		}
		if filters.FailedOnly && m.Success {
			continue
		}
		if filters.Since != nil && m.Timestamp < *filters.Since {
			continue
		}

		metrics = append(metrics, m)

		if filters.Limit > 0 && len(metrics) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading performance metrics: %w", err)
	}

	return metrics, nil
}

// PerformanceSummary aggregates performance metrics by operation
type PerformanceSummary struct {
	Operation    string  // Operation type
	Count        int     // Total operations
	SuccessCount int     // Successful operations
	FailCount    int     // Failed operations
	TotalMs      int64   // Total duration in ms
	MinMs        int64   // Minimum duration
	MaxMs        int64   // Maximum duration
	AvgMs        float64 // Average duration
}

// QueryPerformanceSummary returns aggregated performance metrics grouped by operation
func (q *Query) QueryPerformanceSummary(filters PerformanceFilters) ([]PerformanceSummary, error) {
	metrics, err := q.QueryPerformance(filters)
	if err != nil {
		return nil, err
	}

	// Aggregate by operation
	summaryMap := make(map[string]*PerformanceSummary)

	for _, m := range metrics {
		summary, exists := summaryMap[m.Operation]
		if !exists {
			summary = &PerformanceSummary{
				Operation: m.Operation,
				MinMs:     m.DurationMs,
				MaxMs:     m.DurationMs,
			}
			summaryMap[m.Operation] = summary
		}

		summary.Count++
		summary.TotalMs += m.DurationMs

		if m.Success {
			summary.SuccessCount++
		} else {
			summary.FailCount++
		}

		if m.DurationMs < summary.MinMs {
			summary.MinMs = m.DurationMs
		}
		if m.DurationMs > summary.MaxMs {
			summary.MaxMs = m.DurationMs
		}
	}

	// Calculate averages and convert to slice
	var summaries []PerformanceSummary
	for _, s := range summaryMap {
		if s.Count > 0 {
			s.AvgMs = float64(s.TotalMs) / float64(s.Count)
		}
		summaries = append(summaries, *s)
	}

	return summaries, nil
}

// matchGlob performs simple glob matching (supports * wildcard)
// Patterns:
//   - "*" or "" matches everything
//   - "*suffix" matches strings ending with suffix
//   - "prefix*" matches strings starting with prefix
//   - "*middle*" matches strings containing middle
//   - "exact" matches exactly
func matchGlob(s, pattern string) bool {
	// Empty pattern or * matches everything
	if pattern == "" || pattern == "*" {
		return true
	}

	// Contains pattern: *middle*
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 2 {
		return strings.Contains(s, pattern[1:len(pattern)-1])
	}

	// Suffix pattern: *suffix
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}

	// Prefix pattern: prefix*
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}

	// Exact match
	return s == pattern
}
