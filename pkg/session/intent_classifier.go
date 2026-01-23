package session

import (
	"regexp"
	"strings"
)

// IntentCategory represents the type of user intent
type IntentCategory string

const (
	CategoryRouting    IntentCategory = "routing"
	CategoryTooling    IntentCategory = "tooling"
	CategoryStyle      IntentCategory = "style"
	CategoryWorkflow   IntentCategory = "workflow"
	CategoryDomain     IntentCategory = "domain"
	CategoryCorrection IntentCategory = "correction"
	CategoryApproval   IntentCategory = "approval"
	CategoryRejection  IntentCategory = "rejection"
	CategoryGeneral    IntentCategory = "general"
)

// categoryPatterns maps categories to compiled regex patterns
// Pre-compiled for performance (<10ms classification requirement)
// Note: Rejection checked before Correction to catch "wrong approach" before "wrong"
var categoryPatterns = map[IntentCategory]*regexp.Regexp{
	CategoryRouting:    regexp.MustCompile(`(?i)\b(tier|model|agent|delegate|sonnet|haiku|opus|use\s+\w+\s+for)\b`),
	CategoryTooling:    regexp.MustCompile(`(?i)\b(tool|edit|bash|grep|glob|read|write)\b|(?i)\buse\s+\w+\s+not\b|(?i)\bprefer\s+\w+\s+over\b|(?i)\bdon'?t\s+use\b`),
	CategoryStyle:      regexp.MustCompile(`(?i)\b(concise|verbose|format|output|shorter|longer|brief|detailed|response\s+style)\b`),
	CategoryWorkflow:   regexp.MustCompile(`(?i)\b(always|never|after\s+\w+\s+do|before\s+\w+\s+do|sequence|order|first\s+\w+\s+then)\b`),
	CategoryDomain:     regexp.MustCompile(`(?i)\b(we\s+use|our\s+project|codebase|convention|framework|library|this\s+project)\b`),
	CategoryCorrection: regexp.MustCompile(`(?i)\b(no,?\s*(I|you)\s+meant|actually|not\s+that|incorrect|the\s+other|wrong)\b`),
	CategoryApproval:   regexp.MustCompile(`(?i)^\s*(yes|proceed|go\s+ahead|looks\s+good|confirmed|approve|do\s+it|ok|okay|sure|yep|yeah)\s*$`),
	CategoryRejection:  regexp.MustCompile(`(?i)^\s*(stop|cancel|abort|wrong\s+approach|undo|halt|wait)\s*$|(?i)^\s*don'?t\s*$|(?i)^\s*no\s+don'?t\s*$|(?i)\bwrong\s+approach\b`),
}

// ClassifyIntent determines the category of a user intent using deterministic pattern matching.
// Uses pattern matching on combined question + response text for classification.
// Priority order: correction/rejection/approval (strong signals) checked first,
// then domain-specific categories.
//
// Returns CategoryGeneral if no patterns match.
//
// Performance: <10ms per classification (pre-compiled patterns).
// Deterministic: Same input always produces same output.
//
// Example:
//
//	category := ClassifyIntent("Which model?", "Use sonnet for this")
//	// Returns: CategoryRouting
func ClassifyIntent(question, response string) IntentCategory {
	combined := strings.ToLower(question + " " + response)
	responseLower := strings.ToLower(response)

	// Check patterns in priority order
	// Rejection before correction to catch "wrong approach" before "wrong"
	// Tooling checked before routing to catch "don't use X" patterns
	// Approval and rejection use response-only matching (they're standalone responses)
	priorityOrder := []IntentCategory{
		CategoryRejection,  // Must be before correction to catch "wrong approach"
		CategoryCorrection,
		CategoryApproval,
		CategoryTooling, // Checked before routing to catch "don't use" patterns
		CategoryRouting,
		CategoryWorkflow,
		CategoryDomain,
		CategoryStyle,
	}

	for _, category := range priorityOrder {
		if pattern, ok := categoryPatterns[category]; ok {
			// Approval and rejection patterns match response only (standalone answers)
			if category == CategoryApproval || category == CategoryRejection {
				if pattern.MatchString(responseLower) {
					return category
				}
			} else {
				// Other patterns use combined text
				if pattern.MatchString(combined) {
					return category
				}
			}
		}
	}

	return CategoryGeneral
}
