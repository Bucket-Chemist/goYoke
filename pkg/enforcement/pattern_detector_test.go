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
	content := "This is BLOCKED (by the system)."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect BLOCKED pattern")
	}

	if results[0].Severity != "critical" {
		t.Error("BLOCKED should be critical severity")
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

func TestPatternDetector_CaseInsensitive(t *testing.T) {
	pd := NewPatternDetector()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "uppercase MUST NOT",
			content: "You MUST NOT do this",
			want:    true,
		},
		{
			name:    "lowercase must not",
			content: "You must not do this",
			want:    true,
		},
		{
			name:    "mixed case Must Not",
			content: "You Must Not do this",
			want:    true,
		},
		{
			name:    "NEVER uppercase",
			content: "NEVER use this feature",
			want:    true,
		},
		{
			name:    "never lowercase",
			content: "never use this feature",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := pd.Detect(tt.content)
			got := len(results) > 0
			if got != tt.want {
				t.Errorf("Detect() got = %v, want %v for content: %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestPatternDetector_Forbidden(t *testing.T) {
	pd := NewPatternDetector()
	content := "This operation is FORBIDDEN in production."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect FORBIDDEN pattern")
	}

	if results[0].Severity != "warning" {
		t.Errorf("FORBIDDEN should be warning severity, got: %s", results[0].Severity)
	}
}

func TestPatternDetector_YouCannot(t *testing.T) {
	pd := NewPatternDetector()
	content := "YOU CANNOT perform this action without permissions."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect YOU CANNOT pattern")
	}

	if results[0].Severity != "warning" {
		t.Errorf("YOU CANNOT should be warning severity, got: %s", results[0].Severity)
	}
}

func TestPatternDetector_AllPatterns(t *testing.T) {
	pd := NewPatternDetector()
	content := `
# Bad Documentation

You MUST NOT use this without approval.
This feature is BLOCKED (pending review).
NEVER use this in production.
This is FORBIDDEN for security reasons.
YOU CANNOT access this without admin rights.
`

	results := pd.Detect(content)

	if len(results) != 5 {
		t.Errorf("Should detect all 5 patterns, got: %d", len(results))
	}

	criticalCount := 0
	warningCount := 0
	for _, result := range results {
		if result.Severity == "critical" {
			criticalCount++
		} else if result.Severity == "warning" {
			warningCount++
		}
	}

	if criticalCount != 3 {
		t.Errorf("Expected 3 critical patterns, got: %d", criticalCount)
	}

	if warningCount != 2 {
		t.Errorf("Expected 2 warning patterns, got: %d", warningCount)
	}
}

func TestPatternDetector_HasDocumentationTheater_Critical(t *testing.T) {
	pd := NewPatternDetector()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "critical pattern present",
			content: "You MUST NOT do this",
			want:    true,
		},
		{
			name:    "only warning pattern",
			content: "This is FORBIDDEN",
			want:    false,
		},
		{
			name:    "no patterns",
			content: "Normal documentation text",
			want:    false,
		},
		{
			name:    "mixed critical and warning",
			content: "You MUST NOT do this. This is FORBIDDEN.",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pd.HasDocumentationTheater(tt.content)
			if got != tt.want {
				t.Errorf("HasDocumentationTheater() = %v, want %v for: %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestGenerateWarning_EmptyResults(t *testing.T) {
	warning := GenerateWarning([]DetectionResult{}, "CLAUDE.md")

	if warning != "" {
		t.Errorf("Expected empty warning for empty results, got: %q", warning)
	}
}

func TestGenerateWarning_MultipleResults(t *testing.T) {
	results := []DetectionResult{
		{
			Pattern:     `(?i)\bMUST\s+NOT\b`,
			Description: "Imperative enforcement",
			Severity:    "critical",
			MatchCount:  2,
			FirstMatch:  "MUST NOT",
		},
		{
			Pattern:     `(?i)\bNEVER\s+use\b`,
			Description: "NEVER language",
			Severity:    "critical",
			MatchCount:  1,
			FirstMatch:  "NEVER use",
		},
	}

	warning := GenerateWarning(results, "CLAUDE.md")

	if !strings.Contains(warning, "Found 2 enforcement pattern(s)") {
		t.Error("Warning should mention count of 2 patterns")
	}

	if !strings.Contains(warning, "1. Imperative enforcement") {
		t.Error("Warning should list first result")
	}

	if !strings.Contains(warning, "2. NEVER language") {
		t.Error("Warning should list second result")
	}

	if !strings.Contains(warning, "REQUIRED ACTION") {
		t.Error("Warning should include required action section")
	}
}

func TestPatternDetector_FirstMatchExtraction(t *testing.T) {
	pd := NewPatternDetector()
	content := "Some text here. You MUST NOT do this. More text. You MUST NOT do that."

	results := pd.Detect(content)

	found := false
	for _, result := range results {
		if strings.Contains(result.Pattern, "MUST") {
			if result.FirstMatch != "MUST NOT" {
				t.Errorf("Expected FirstMatch to be 'MUST NOT', got: %q", result.FirstMatch)
			}
			if result.MatchCount != 2 {
				t.Errorf("Expected MatchCount to be 2, got: %d", result.MatchCount)
			}
			found = true
		}
	}

	if !found {
		t.Fatal("Should find MUST pattern")
	}
}

func TestPatternDetector_BlockedWithParentheses(t *testing.T) {
	pd := NewPatternDetector()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "BLOCKED with reason in parentheses",
			content: "This is BLOCKED (pending approval)",
			want:    true,
		},
		{
			name:    "BLOCKED without parentheses",
			content: "This is BLOCKED",
			want:    false,
		},
		{
			name:    "blocked lowercase with parentheses",
			content: "This is blocked (by policy)",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := pd.Detect(tt.content)
			got := len(results) > 0
			if got != tt.want {
				t.Errorf("Detect() for BLOCKED pattern got = %v, want %v for: %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestPatternDetector_NormalDocumentationSafe(t *testing.T) {
	pd := NewPatternDetector()

	safeContent := []string{
		"This is a guide to using the API.",
		"Follow these best practices when writing code.",
		"Make sure to test your changes before committing.",
		"The system will validate your input.",
		"Consider using hooks for enforcement.",
		"Reference the LLM-guidelines.md for details.",
		"Use descriptive variable names.",
		"Write comprehensive tests.",
		"Document your architectural decisions.",
		"The hook blocks invalid operations.",
		"Never skip testing (best practice advice, not imperative)",
	}

	for _, content := range safeContent {
		t.Run(content, func(t *testing.T) {
			results := pd.Detect(content)
			if len(results) > 0 {
				t.Errorf("Safe content should not trigger detection: %q\nGot: %+v", content, results)
			}
		})
	}
}
