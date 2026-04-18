package enforcement

import (
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// AnalyzeToolEventForDocTheater analyzes a ToolEvent for documentation theater patterns.
// Returns detection results if theater patterns are found, or nil if the event should be skipped
// or the content is clean.
//
// This function integrates:
// - ToolEvent helper methods (GOgent-080) for event classification and content extraction
// - PatternDetector (GOgent-081) for theater pattern detection
//
// Filtering logic:
//  1. Non-write operations (Read, Glob, etc.) → return nil
//  2. Non-CLAUDE.md files → return nil
//  3. Write operations on CLAUDE.md files → extract content and detect patterns
//
// Returns:
//   - []DetectionResult: Theater patterns found (requires user attention)
//   - nil: Event should be skipped OR content is clean
func AnalyzeToolEventForDocTheater(event *routing.ToolEvent) []DetectionResult {
	// Filter 1: Skip non-write operations
	if !event.IsWriteOperation() {
		return nil
	}

	// Filter 2: Skip non-CLAUDE.md files
	if !event.IsClaudeMDFile() {
		return nil
	}

	// Extract content from Write or Edit tool inputs
	content := event.ExtractWriteContent()
	if content == "" {
		// No content to analyze (shouldn't happen for write operations, but defensive)
		return nil
	}

	// Detect theater patterns
	detector := NewPatternDetector()
	results := detector.Detect(content)

	// Return results (may be empty slice if content is clean)
	if len(results) == 0 {
		return nil
	}

	return results
}
