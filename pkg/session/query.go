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
