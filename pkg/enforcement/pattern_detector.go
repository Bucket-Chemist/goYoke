package enforcement

import (
	"fmt"
	"regexp"
	"strings"
)

// EnforcementPattern represents a documentation theater anti-pattern
type EnforcementPattern struct {
	Pattern     string
	Description string
	Severity    string // "warning", "critical"
}

// PatternDetector scans content for enforcement theater
type PatternDetector struct {
	patterns []EnforcementPattern
}

// NewPatternDetector creates detector with known anti-patterns
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: []EnforcementPattern{
			{
				Pattern:     `(?i)\bMUST\s+NOT\b`,
				Description: "Imperative enforcement without programmatic backing",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bBLOCKED\b.*\(.*\)`,
				Description: "Claims of blocking without hook enforcement",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bNEVER\s+use\b`,
				Description: "NEVER language without validation hook",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bFORBIDDEN\b`,
				Description: "Forbidden declarations without enforcement",
				Severity:    "warning",
			},
			{
				Pattern:     `(?i)\bYOU\s+CANNOT\b`,
				Description: "Prohibition without mechanism",
				Severity:    "warning",
			},
		},
	}
}

// Detect scans content for anti-patterns
func (pd *PatternDetector) Detect(content string) []DetectionResult {
	var results []DetectionResult

	for _, ep := range pd.patterns {
		regex := regexp.MustCompile(ep.Pattern)
		matches := regex.FindAllStringIndex(content, -1)

		if len(matches) > 0 {
			results = append(results, DetectionResult{
				Pattern:     ep.Pattern,
				Description: ep.Description,
				Severity:    ep.Severity,
				MatchCount:  len(matches),
				FirstMatch:  content[matches[0][0]:matches[0][1]],
			})
		}
	}

	return results
}

// HasDocumentationTheater checks if critical patterns found
func (pd *PatternDetector) HasDocumentationTheater(content string) bool {
	results := pd.Detect(content)
	for _, result := range results {
		if result.Severity == "critical" {
			return true
		}
	}
	return false
}

// DetectionResult represents a found anti-pattern
type DetectionResult struct {
	Pattern     string
	Description string
	Severity    string
	MatchCount  int
	FirstMatch  string
}

// GenerateWarning creates warning message for detected patterns
func GenerateWarning(results []DetectionResult, filename string) string {
	if len(results) == 0 {
		return ""
	}

	var warning strings.Builder
	warning.WriteString(fmt.Sprintf(
		"⚠️ DOCUMENTATION THEATER DETECTED in %s\n\n"+
			"Found %d enforcement pattern(s) without programmatic backing:\n\n",
		filename,
		len(results),
	))

	for i, result := range results {
		warning.WriteString(fmt.Sprintf(
			"%d. %s\n"+
				"   Pattern: %s\n"+
				"   Found: %q\n"+
				"   Severity: %s\n\n",
			i+1,
			result.Description,
			result.Pattern,
			result.FirstMatch,
			result.Severity,
		))
	}

	warning.WriteString(
		"ENFORCEMENT ARCHITECTURE:\n\n" +
		"Text instructions are probabilistic (LLM may ignore).\n" +
		"Real enforcement requires three components:\n" +
		"1. Declarative Rule (routing-schema.json)\n" +
		"2. Programmatic Check (validate-routing.sh or hook)\n" +
		"3. Reference Documentation (CLAUDE.md points to enforcement)\n\n" +
		"See: ~/.claude/rules/LLM-guidelines.md § Enforcement Architecture\n\n" +
		"REQUIRED ACTION:\n" +
		"Implement hook enforcement FIRST, then update CLAUDE.md to REFERENCE it.\n" +
		"Do NOT add enforcement language without the hook.\n",
	)

	return warning.String()
}
