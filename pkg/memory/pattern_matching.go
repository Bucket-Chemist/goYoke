// Package memory provides sharp edge tracking and pattern matching for debugging support.
package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
	"gopkg.in/yaml.v3"
)

// SharpEdgeTemplate represents a known sharp edge pattern from YAML.
// Sharp edges are patterns of common errors or gotchas that agents encounter,
// along with their solutions and context for pattern matching.
type SharpEdgeTemplate struct {
	ID          string   `yaml:"id,omitempty"`
	Name        string   `yaml:"name,omitempty"`        // Alternative to ID (python-pro style)
	ErrorType   string   `yaml:"error_type,omitempty"`
	FilePattern string   `yaml:"file_pattern,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Solution    string   `yaml:"solution,omitempty"`
	Mitigation  string   `yaml:"mitigation,omitempty"`  // Alternative to Solution (python-pro style)
	Severity    string   `yaml:"severity,omitempty"`    // e.g., "critical", "high", "medium"
	Category    string   `yaml:"category,omitempty"`    // e.g., "concurrency", "networking"
	Symptom     string   `yaml:"symptom,omitempty"`     // Human-readable symptom description
	AutoInject  bool     `yaml:"auto_inject,omitempty"` // Whether to auto-inject on match
	Source      string   `yaml:"-"`                     // Resolver path where loaded from
}

// GetID returns the template ID, preferring ID over Name for backwards compatibility.
func (t SharpEdgeTemplate) GetID() string {
	if t.ID != "" {
		return t.ID
	}
	return t.Name
}

// GetSolution returns the solution text, preferring Solution over Mitigation.
func (t SharpEdgeTemplate) GetSolution() string {
	if t.Solution != "" {
		return t.Solution
	}
	return t.Mitigation
}

// SharpEdgesFile represents the YAML file structure for sharp-edges.yaml.
// This wrapper type handles multiple versioned formats used by agent configurations:
// - Format 1: { sharp_edges: [...] } (go-pro style)
// - Format 2: { edges: [...] } (python-pro style)
// - Format 3: Direct array [...] (legacy)
type SharpEdgesFile struct {
	Version    string              `yaml:"version,omitempty"`
	Updated    string              `yaml:"updated,omitempty"`
	SharpEdges []SharpEdgeTemplate `yaml:"sharp_edges,omitempty"`
	Edges      []SharpEdgeTemplate `yaml:"edges,omitempty"` // Alternative key used by some agents
}

// SharpEdgeIndex provides fast lookup of sharp edge templates.
// It maintains multiple indexes for different search patterns:
// - ByErrorType: Look up templates by error type (e.g., "TypeError", "nil_pointer")
// - ByKeyword: Look up templates by keywords (e.g., "type assertion", "map access")
// - All: Complete list of all loaded templates
type SharpEdgeIndex struct {
	ByErrorType map[string][]SharpEdgeTemplate
	ByKeyword   map[string][]SharpEdgeTemplate
	All         []SharpEdgeTemplate
}

// LoadSharpEdgesIndex loads sharp-edges.yaml files for the given agent IDs via Resolver
// and builds a searchable index for pattern matching.
//
// The Resolver union-reads from disk and embedded layers, so agents in either layer
// are automatically discovered. For each agent ID, the path
// "agents/{id}/sharp-edges.yaml" is resolved.
//
// Missing files are silently skipped. Parse errors are logged to stderr and the
// file is skipped. Processing continues with remaining valid files.
//
// Parameters:
//   - agentIDs: Agent IDs to load (e.g., "go-pro", "python-pro")
//
// Returns:
//   - *SharpEdgeIndex: Populated index structure (never nil)
//   - error: Always returns nil (errors are logged as warnings)
func LoadSharpEdgesIndex(agentIDs []string) (*SharpEdgeIndex, error) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	if len(agentIDs) == 0 {
		return index, nil
	}

	r, err := resolve.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create resolver: %v\n", err)
		return index, nil
	}

	for _, id := range agentIDs {
		agentPath := "agents/" + id + "/sharp-edges.yaml"

		data, err := r.ReadFile(agentPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "Warning: Failed to read %s: %v\n", agentPath, err)
			}
			continue
		}

		// Parse YAML into templates
		// Try versioned formats first (with wrapper), then fall back to raw array
		var templates []SharpEdgeTemplate
		var fileWrapper SharpEdgesFile
		if err := yaml.Unmarshal(data, &fileWrapper); err == nil {
			// Check both possible keys: sharp_edges (go-pro style) and edges (python-pro style)
			if len(fileWrapper.SharpEdges) > 0 {
				templates = fileWrapper.SharpEdges
			} else if len(fileWrapper.Edges) > 0 {
				templates = fileWrapper.Edges
			}
		}
		// If wrapper parsing didn't find templates, try direct array format (legacy)
		if len(templates) == 0 {
			if err := yaml.Unmarshal(data, &templates); err != nil {
				// Neither format worked - skip silently (agents may have different schemas)
				continue
			}
		}

		// Index each template
		for _, tmpl := range templates {
			tmpl.Source = agentPath

			index.All = append(index.All, tmpl)

			index.ByErrorType[tmpl.ErrorType] = append(
				index.ByErrorType[tmpl.ErrorType], tmpl)

			for _, keyword := range tmpl.Keywords {
				index.ByKeyword[keyword] = append(
					index.ByKeyword[keyword], tmpl)
			}
		}
	}

	return index, nil
}

// SharpEdge represents a current error or problem encountered during execution.
// It differs from SharpEdgeTemplate in that it captures runtime error information,
// while templates are pre-defined patterns from YAML files.
type SharpEdge struct {
	ErrorType    string // Type of error (e.g., "TypeError", "nil_pointer")
	File         string // File where error occurred
	ErrorMessage string // The actual error message text
}

// Match represents a template match with similarity score.
// It indicates how well a SharpEdgeTemplate matches a current SharpEdge,
// along with which signals contributed to the match (error type, file pattern, keywords).
type Match struct {
	Template  SharpEdgeTemplate // The matching template with solution
	Score     int               // Similarity score (higher is better)
	MatchedOn []string          // What signals matched (e.g., ["error_type", "file_pattern", "keyword:bool"])
}

// Scoring constants for pattern matching
const (
	SCORE_ERROR_TYPE_EXACT = 5 // Exact error type match
	SCORE_FILE_PATTERN     = 3 // File matches glob pattern
	SCORE_KEYWORD          = 2 // Error message contains keyword
	SCORE_THRESHOLD        = 5 // Minimum score to return match
)

// FindSimilar compares a SharpEdge against an index of templates and returns
// the top 3 most similar matches ranked by similarity score.
func FindSimilar(edge *SharpEdge, index *SharpEdgeIndex) []Match {
	// Use map to deduplicate matches by template ID
	matchMap := make(map[string]*Match)

	// Try exact error type first
	if templates, ok := index.ByErrorType[edge.ErrorType]; ok {
		for _, tmpl := range templates {
			score := SCORE_ERROR_TYPE_EXACT
			matchedOn := []string{"error_type"}

			// Check file pattern
			if matched, _ := filepath.Match(tmpl.FilePattern, edge.File); matched {
				score += SCORE_FILE_PATTERN
				matchedOn = append(matchedOn, "file_pattern")
			}

			// Check keywords in error message
			for _, keyword := range tmpl.Keywords {
				if strings.Contains(strings.ToLower(edge.ErrorMessage), strings.ToLower(keyword)) {
					score += SCORE_KEYWORD
					matchedOn = append(matchedOn, fmt.Sprintf("keyword:%s", keyword))
				}
			}

			if score >= SCORE_THRESHOLD {
				matchMap[tmpl.ID] = &Match{
					Template:  tmpl,
					Score:     score,
					MatchedOn: matchedOn,
				}
			}
		}
	}

	// Try keyword matches for different error types
	errorWords := strings.Fields(strings.ToLower(edge.ErrorMessage))
	for _, word := range errorWords {
		if templates, ok := index.ByKeyword[word]; ok {
			for _, tmpl := range templates {
				// Skip if already matched by error type
				if tmpl.ErrorType == edge.ErrorType {
					continue
				}

				// Check if we already have a match for this template
				if existing, exists := matchMap[tmpl.ID]; exists {
					// Update existing match with additional keyword
					existing.Score += SCORE_KEYWORD
					existing.MatchedOn = append(existing.MatchedOn, fmt.Sprintf("keyword:%s", word))
				} else {
					score := SCORE_KEYWORD
					matchedOn := []string{fmt.Sprintf("keyword:%s", word)}

					// Check file pattern
					if matched, _ := filepath.Match(tmpl.FilePattern, edge.File); matched {
						score += SCORE_FILE_PATTERN
						matchedOn = append(matchedOn, "file_pattern")
					}

					if score >= SCORE_THRESHOLD {
						matchMap[tmpl.ID] = &Match{
							Template:  tmpl,
							Score:     score,
							MatchedOn: matchedOn,
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	matches := make([]Match, 0, len(matchMap))
	for _, match := range matchMap {
		matches = append(matches, *match)
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Return top 3
	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}
