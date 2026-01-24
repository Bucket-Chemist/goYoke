---
id: GOgent-081
title: Pattern Detection for Documentation Theater
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-080"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-081: Pattern Detection for Documentation Theater

**Time**: 2 hours
**Dependencies**: GOgent-080

**Task**:
Scan content for enforcement patterns that indicate documentation theater.

**File**: `pkg/enforcement/pattern_detector.go`

**Imports**:
```go
package enforcement

import (
	"fmt"
	"regexp"
	"strings"
)
```

**Implementation**:
```go
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
```

**Tests**: `pkg/enforcement/pattern_detector_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
)

func TestPatternDetector_MustNot(t *testing.T) {
	pd := NewPatternDetector()
	content := "You MUST NOT use this feature without approval."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect MUST NOT pattern")
	}

	if results[0].Severity != "critical" {
		t.Error("MUST NOT should be critical severity")
	}
}

func TestPatternDetector_Blocked(t *testing.T) {
	pd := NewPatternDetector()
	content := "This is BLOCKED by the system."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect BLOCKED pattern")
	}
}

func TestPatternDetector_Never(t *testing.T) {
	pd := NewPatternDetector()
	content := "NEVER use this without consulting the docs."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect NEVER pattern")
	}
}

func TestPatternDetector_NoTheater(t *testing.T) {
	pd := NewPatternDetector()
	content := `# Guidelines

Follow these conventions when coding:
- Use descriptive names
- Write tests
- Document decisions
`

	results := pd.Detect(content)

	if len(results) > 0 {
		t.Errorf("Should not detect patterns in normal content, got: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not flag normal content as theater")
	}
}

func TestPatternDetector_MultipleMatches(t *testing.T) {
	pd := NewPatternDetector()
	content := `MUST NOT do X.
MUST NOT do Y.
MUST NOT do Z.
`

	results := pd.Detect(content)

	// Should detect the pattern once but with match count 3
	found := false
	for _, result := range results {
		if strings.Contains(result.Pattern, "MUST") {
			if result.MatchCount != 3 {
				t.Errorf("Expected 3 matches, got: %d", result.MatchCount)
			}
			found = true
		}
	}

	if !found {
		t.Fatal("Should find MUST pattern")
	}
}

func TestGenerateWarning(t *testing.T) {
	results := []DetectionResult{
		{
			Pattern:     "MUST NOT",
			Description: "Test pattern",
			Severity:    "critical",
			MatchCount:  1,
			FirstMatch:  "MUST NOT",
		},
	}

	warning := GenerateWarning(results, "CLAUDE.md")

	if !strings.Contains(warning, "DOCUMENTATION THEATER") {
		t.Error("Warning should mention theater")
	}

	if !strings.Contains(warning, "Enforcement Architecture") {
		t.Error("Warning should reference enforcement architecture")
	}

	if !strings.Contains(warning, "LLM-guidelines.md") {
		t.Error("Warning should reference guidelines")
	}
}
```

**Acceptance Criteria**:
- [ ] `NewPatternDetector()` creates detector with known patterns
- [ ] `Detect()` finds all enforcement anti-patterns
- [ ] Marks MUST NOT, BLOCKED, NEVER as critical
- [ ] `HasDocumentationTheater()` returns true for critical patterns
- [ ] `GenerateWarning()` explains pattern, severity, and fix
- [ ] References enforcement architecture docs
- [ ] Tests verify pattern detection and warning generation
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Pattern detection is core to preventing documentation theater anti-pattern.

---
