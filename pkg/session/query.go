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
