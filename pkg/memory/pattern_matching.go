// Package memory provides sharp edge tracking and pattern matching for debugging support.
package memory

import (
	"fmt"
	"os"
	"path/filepath"

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
