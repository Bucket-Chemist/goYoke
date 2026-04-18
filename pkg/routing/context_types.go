package routing

import (
	"path/filepath"
)

// ContextRequirements defines what context an agent needs at spawn time.
// This is used by goyoke-validate to inject conventions into Task prompts.
type ContextRequirements struct {
	// Rules lists rules files to inject (e.g., "agent-guidelines.md")
	Rules []string `json:"rules,omitempty"`

	// Conventions specifies convention files to inject
	Conventions ConventionRequirements `json:"conventions,omitempty"`
}

// ConventionRequirements specifies base and conditional conventions.
type ConventionRequirements struct {
	// Base conventions always injected for this agent
	Base []string `json:"base,omitempty"`

	// Conditional conventions injected based on file path patterns
	Conditional []ConditionalConvention `json:"conditional,omitempty"`
}

// ConditionalConvention represents a convention that's only loaded
// when the task involves files matching the pattern.
type ConditionalConvention struct {
	// Pattern is a glob pattern to match file paths in the task
	Pattern string `json:"pattern"`

	// Convention is the filename to load if pattern matches
	Convention string `json:"convention"`
}

// HasContextRequirements returns true if the agent has any context requirements.
func (c *ContextRequirements) HasContextRequirements() bool {
	if c == nil {
		return false
	}
	return len(c.Rules) > 0 || len(c.Conventions.Base) > 0 || len(c.Conventions.Conditional) > 0
}

// GetAllConventions returns all conventions that should be loaded,
// given a list of file paths mentioned in the task.
func (c *ContextRequirements) GetAllConventions(taskFiles []string) []string {
	if c == nil {
		return nil
	}

	conventions := make([]string, 0)

	// Add base conventions
	conventions = append(conventions, c.Conventions.Base...)

	// Add conditional conventions if patterns match
	for _, cond := range c.Conventions.Conditional {
		if matchesAnyFile(taskFiles, cond.Pattern) {
			conventions = append(conventions, cond.Convention)
		}
	}

	return conventions
}

// matchesAnyFile checks if any file path matches the glob pattern.
func matchesAnyFile(files []string, pattern string) bool {
	for _, file := range files {
		if matched, _ := filepath.Match(pattern, file); matched {
			return true
		}
		// Also try matching against just the directory path
		dir := filepath.Dir(file)
		if matched, _ := filepath.Match(pattern, dir); matched {
			return true
		}
	}
	return false
}
