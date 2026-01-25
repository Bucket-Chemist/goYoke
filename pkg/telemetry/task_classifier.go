package telemetry

import (
	"strings"
)

// ClassifyTask extracts task type and domain from description for ML labeling
func ClassifyTask(description string) (taskType, taskDomain string) {
	descLower := strings.ToLower(description)

	// Task type detection - ordered by specificity (more specific patterns first)
	taskType = "unknown"

	// Order matters - check more specific patterns before general ones
	typePatterns := []struct {
		typeName string
		patterns []string
	}{
		// Understanding tasks (most specific, check first)
		{"document_understanding", []string{"summarize", "extract from", "analyze document", "key points", "what does this say", "extract key points"}},
		{"codebase_understanding", []string{"how does", "explain how", "trace through", "architecture of", "what is the structure"}},
		{"synthesis", []string{"synthesize", "combine all", "merge findings", "consolidate", "bring together"}},

		// Search patterns (check before debug to avoid "error" false positives)
		{"search", []string{"find where", "find all", "search", "locate", "where is", "which files"}},

		// Specific action types
		{"review", []string{"review", "audit", "verify code"}},
		{"test", []string{"unit test", "test coverage", "add test", "write unit tests"}},
		{"refactor", []string{"refactor", "clean up", "restructure", "reorganize", "simplify"}},
		{"debug", []string{"debug", "fix the", "fix ", " bug", "broken", "error"}},

		// Broad action types (check last)
		{"documentation", []string{"document", "readme", "docstring", "comment", "create api"}},
		{"implementation", []string{"implement", "create a", "add ", "build the", "write code"}},
	}

	for _, tp := range typePatterns {
		for _, p := range tp.patterns {
			if strings.Contains(descLower, p) {
				taskType = tp.typeName
				goto domainDetection
			}
		}
	}

domainDetection:
	// Domain detection
	taskDomain = "unknown"
	domainPatterns := map[string][]string{
		"python":         {"python", ".py", "pip", "pytest", "pyproject", "django", "flask"},
		"go":             {"go ", "golang", ".go", "cobra", "bubbletea", "go test"},
		"r":              {" r ", "shiny", "golem", ".r", "tidyverse", "ggplot"},
		"javascript":     {"javascript", "typescript", ".js", ".ts", "npm", "node"},
		"infrastructure": {"docker", "kubernetes", "ci/cd", "deploy", "terraform", "ansible"},
		"documentation":  {"readme", "docs/", "markdown", ".md", "documentation"},
	}

	for domain, patterns := range domainPatterns {
		for _, p := range patterns {
			if strings.Contains(descLower, p) {
				taskDomain = domain
				return taskType, taskDomain
			}
		}
	}

	return taskType, taskDomain
}

// TaskTypeLabels returns all valid task type labels
func TaskTypeLabels() []string {
	return []string{
		"implementation", "search", "documentation", "debug", "refactor",
		"review", "test", "document_understanding", "codebase_understanding", "synthesis",
	}
}

// TaskDomainLabels returns all valid domain labels
func TaskDomainLabels() []string {
	return []string{
		"python", "go", "r", "javascript", "infrastructure", "documentation",
	}
}
