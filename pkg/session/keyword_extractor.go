package session

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Known keywords to extract from user responses
// Organized by category: tools, models, test frameworks, build tools, VCS
var knownKeywords = map[string]bool{
	// Tools
	"edit":     true,
	"bash":     true,
	"read":     true,
	"write":    true,
	"glob":     true,
	"grep":     true,
	"task":     true,
	"webfetch": true,
	// Models
	"sonnet": true,
	"haiku":  true,
	"opus":   true,
	// Test frameworks
	"pytest":    true,
	"jest":      true,
	"mocha":     true,
	"unittest":  true,
	"go test":   true,
	"rspec":     true,
	"testthat":  true,
	"vitest":    true,
	"ava":       true,
	"tape":      true,
	// Build tools
	"make":     true,
	"npm":      true,
	"yarn":     true,
	"pnpm":     true,
	"cargo":    true,
	"go build": true,
	"pip":      true,
	"uv":       true,
	"poetry":   true,
	"bundle":   true,
	// VCS
	"git":    true,
	"commit": true,
	"push":   true,
	"pull":   true,
	"merge":  true,
	"rebase": true,
	// Other common terms
	"docker":     true,
	"kubernetes": true,
	"terraform":  true,
	"ansible":    true,
}

// filePathPattern matches common file paths with extensions
// Captures: .go, .py, .js, .ts, .rs, .r, .md, .json, .yaml, .yml, .toml
// Order matters: longer extensions (tsx, jsx, jsonl) before shorter ones (ts, js, json)
// Use word boundary at end to ensure full extension match
var filePathPattern = regexp.MustCompile(`[\w./\-]+\.(tsx|jsx|jsonl|yaml|bash|json|go|py|js|ts|rs|r|R|md|yml|toml|sh)\b`)

// ExtractKeywords extracts relevant keywords from a user response.
// Extracts:
//   - Known tool names (edit, bash, task, etc.)
//   - Model names (sonnet, haiku, opus)
//   - Test frameworks (pytest, jest, go test, etc.)
//   - Build tools (make, npm, go build, etc.)
//   - VCS commands (git, commit, push, etc.)
//   - File paths (extracts just the filename)
//
// Returns at most 10 keywords, deduplicated.
// Case-insensitive matching, lowercase output.
//
// Example:
//
//	keywords := ExtractKeywords("Use Edit not sed, and run bash after. Check pkg/session/query.go")
//	// Returns: ["edit", "bash", "query.go"]
func ExtractKeywords(response string) []string {
	lower := strings.ToLower(response)
	seen := make(map[string]bool)
	var keywords []string

	// Extract known keywords
	for keyword := range knownKeywords {
		if strings.Contains(lower, keyword) {
			if !seen[keyword] {
				seen[keyword] = true
				keywords = append(keywords, keyword)
			}
		}
	}

	// Extract file paths (filename only, not full path)
	matches := filePathPattern.FindAllString(response, -1)
	for _, match := range matches {
		// Use just the filename, not full path
		name := filepath.Base(match)
		nameLower := strings.ToLower(name)
		if !seen[nameLower] {
			seen[nameLower] = true
			keywords = append(keywords, nameLower)
		}
	}

	// Limit to 10 keywords (performance + relevance)
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}
