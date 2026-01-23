// Package memory provides sharp edge tracking and pattern matching for debugging support.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SharpEdgeTemplate represents a known sharp edge pattern from YAML.
// Sharp edges are patterns of common errors or gotchas that agents encounter,
// along with their solutions and context for pattern matching.
type SharpEdgeTemplate struct {
	ID          string   `yaml:"id"`
	ErrorType   string   `yaml:"error_type"`
	FilePattern string   `yaml:"file_pattern"`
	Keywords    []string `yaml:"keywords"`
	Description string   `yaml:"description"`
	Solution    string   `yaml:"solution"`
	Source      string   `yaml:"-"` // File path where loaded from
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

// LoadSharpEdgesIndex loads sharp-edges.yaml files from multiple agent directories
// and builds a searchable index for pattern matching.
//
// The function scans each provided agent directory for a sharp-edges.yaml file,
// parses the YAML content, and builds three indexes:
// - ByErrorType: Map from error type to matching templates
// - ByKeyword: Map from keyword to matching templates
// - All: Complete list of all templates
//
// Missing or malformed YAML files are handled gracefully:
// - Missing files are silently skipped
// - Parse errors are logged to stderr and the file is skipped
// - Processing continues with remaining valid files
//
// Parameters:
//   - agentDirs: List of directories to scan for sharp-edges.yaml files
//
// Returns:
//   - *SharpEdgeIndex: Populated index structure (never nil)
//   - error: Always returns nil (errors are logged as warnings)
//
// Example usage:
//
//	agentDirs := []string{
//		filepath.Join(os.Getenv("HOME"), ".claude", "agents", "python-pro"),
//		filepath.Join(os.Getenv("HOME"), ".claude", "agents", "go-pro"),
//	}
//	index, err := LoadSharpEdgesIndex(agentDirs)
//	if err != nil {
//		// Handle error (currently never returns error)
//	}
//	// Look up templates by error type
//	templates := index.ByErrorType["TypeError"]
func LoadSharpEdgesIndex(agentDirs []string) (*SharpEdgeIndex, error) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	for _, dir := range agentDirs {
		yamlPath := filepath.Join(dir, "sharp-edges.yaml")

		// Skip if doesn't exist (not an error - agents may not have sharp edges yet)
		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			continue
		}

		// Read YAML file
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			// Log warning but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: Failed to read %s: %v\n", yamlPath, err)
			continue
		}

		// Parse YAML into templates
		var templates []SharpEdgeTemplate
		if err := yaml.Unmarshal(data, &templates); err != nil {
			// Log warning but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", yamlPath, err)
			continue
		}

		// Index each template
		for _, tmpl := range templates {
			// Store source file path for debugging/tracing
			tmpl.Source = yamlPath

			// Add to complete list
			index.All = append(index.All, tmpl)

			// Index by error type
			index.ByErrorType[tmpl.ErrorType] = append(
				index.ByErrorType[tmpl.ErrorType], tmpl)

			// Index by each keyword
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
//
// The function uses multi-signal scoring:
//   - ERROR_TYPE_EXACT (5 points): Exact match on error type
//   - FILE_PATTERN (3 points): File matches template's glob pattern
//   - KEYWORD (2 points per keyword): Error message contains keyword (case-insensitive)
//
// Only matches with score >= SCORE_THRESHOLD (5) are returned.
// If fewer than 3 matches meet the threshold, returns fewer matches.
//
// Matching strategy:
//  1. Try exact error type matches first (highest priority)
//  2. Try keyword matches from different error types (lower priority)
//  3. Sort by score descending
//  4. Return top 3 matches
//
// Parameters:
//   - edge: The current sharp edge to match against templates
//   - index: Pre-built index of sharp edge templates
//
// Returns:
//   - []Match: Up to 3 best matches, sorted by score (highest first)
//
// Example:
//
//	edge := &SharpEdge{
//		ErrorType:    "TypeError",
//		File:         "pkg/routing/task_validation.go",
//		ErrorMessage: "invalid type assertion: field is bool, not interface{}",
//	}
//	matches := FindSimilar(edge, index)
//	for _, match := range matches {
//		fmt.Printf("Match: %s (score=%d, matched=%v)\n", match.Template.ID, match.Score, match.MatchedOn)
//	}
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
