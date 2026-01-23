// Package memory provides sharp edge tracking and pattern matching for debugging support.
package memory

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// GenerateBlockingResponse creates an enhanced hook response with pattern-matched remediation suggestions.
//
// This function is invoked when the failure count threshold is reached during sharp edge detection.
// It builds a blocking response that includes:
//   - Base debugging loop guidance (stop, document, analyze, escalate)
//   - Matched sharp edge patterns from YAML templates (if found)
//   - Suggested solutions from the highest-scored matches
//   - Source references for tracing back to sharp-edges.yaml files
//
// The response leverages pattern matching to transform generic "you're in a loop" messages
// into actionable guidance based on similar known issues.
//
// Parameters:
//   - edge: The current sharp edge that triggered the threshold
//   - index: Pre-built index of sharp edge templates from YAML files
//   - failureCount: Number of consecutive failures on this file/error combination
//
// Returns:
//   - *routing.HookResponse: A blocking response with decision="block" and enhanced context
//
// Example usage:
//
//	edge := &SharpEdge{
//		File:         "pkg/routing/task_validation.go",
//		ErrorType:    "TypeError",
//		ErrorMessage: "invalid type assertion: field is bool, not interface{}",
//	}
//	resp := GenerateBlockingResponse(edge, index, 3)
//	// resp.Decision == "block"
//	// resp.HookSpecificOutput["additionalContext"] contains matched patterns
//
// Nil Safety:
//   - If edge is nil, returns generic blocking response (file="unknown")
//   - If index is nil, returns blocking response without pattern matches
//   - Empty matches are handled gracefully with "No similar patterns found" message
func GenerateBlockingResponse(edge *SharpEdge, index *SharpEdgeIndex, failureCount int) *routing.HookResponse {
	// Handle nil edge gracefully
	file := "unknown"
	errorType := "unknown"
	if edge != nil {
		file = edge.File
		errorType = edge.ErrorType
	}

	// Find similar patterns (safe if index is nil or edge is nil)
	var matches []Match
	if edge != nil && index != nil {
		matches = FindSimilar(edge, index)
	}

	// Build base message using strings.Builder for efficiency
	var baseMsg strings.Builder
	baseMsg.WriteString(fmt.Sprintf(
		"🔴 DEBUGGING LOOP DETECTED (%d failures on %s):\n"+
			"1. STOP current approach\n"+
			"2. Document this sharp edge (auto-logged to pending-learnings.jsonl)\n"+
			"3. Analyze root cause - what assumption might be wrong?\n"+
			"4. Consider escalation to next tier\n"+
			"5. Check sharp-edges.yaml for similar patterns",
		failureCount,
		file,
	))

	// Add pattern matches if found
	if len(matches) > 0 {
		baseMsg.WriteString("\n\n📚 SIMILAR SHARP EDGES FOUND:\n")

		for i, match := range matches {
			baseMsg.WriteString(fmt.Sprintf("\n**Match %d** (score: %d, matched on: %s):\n",
				i+1,
				match.Score,
				strings.Join(match.MatchedOn, ", "),
			))

			baseMsg.WriteString(fmt.Sprintf("- **ID**: %s\n", match.Template.ID))
			baseMsg.WriteString(fmt.Sprintf("- **Description**: %s\n", match.Template.Description))
			baseMsg.WriteString(fmt.Sprintf("- **Suggested Solution**: %s\n", match.Template.Solution))
			baseMsg.WriteString(fmt.Sprintf("- **Source**: %s\n", match.Template.Source))
		}

		baseMsg.WriteString("\n💡 Try the suggested solution from the highest-scored match.")
	} else {
		baseMsg.WriteString("\n\nℹ️ No similar patterns found in sharp-edges.yaml. This may be a novel issue.")
	}

	// Create response using routing package helper
	resp := routing.NewBlockResponse(
		"PostToolUse",
		fmt.Sprintf("⚠️ SHARP EDGE DETECTED: %d consecutive failures on '%s' (%s)",
			failureCount, file, errorType),
	)

	// Add additionalContext field
	resp.AddField("additionalContext", baseMsg.String())

	return resp
}
