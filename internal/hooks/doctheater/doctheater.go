// Package doctheater implements the gogent-doc-theater hook.
// It detects documentation theater patterns in CLAUDE.md writes.
package doctheater

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/enforcement"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// DefaultTimeout is the parse timeout for stdin events.
const DefaultTimeout = 5 * time.Second

// EscapeJSON escapes a string for embedding in a JSON string literal.
func EscapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// AllowResponse returns a hook response JSON string with decision "allow".
func AllowResponse() string {
	return `{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`
}

// WarnResponse returns a hook response JSON string with decision "warn".
func WarnResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "warn",
    "additionalContext": "%s"
  }
}`, EscapeJSON(message))
}

// BlockResponse returns a hook response JSON string with decision "block".
func BlockResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "block",
    "additionalContext": "🚫 BLOCKED: %s"
  }
}`, EscapeJSON(message))
}

// OutputAllow prints an allow response to stdout.
func OutputAllow() {
	fmt.Println(AllowResponse())
}

// OutputError prints an error-degraded-to-allow response to stdout.
func OutputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, EscapeJSON(message))
}

// GenerateDocTheaterResponse builds the hook response based on detection results.
func GenerateDocTheaterResponse(event *routing.ToolEvent, results []enforcement.DetectionResult) string {
	if len(results) == 0 {
		return AllowResponse()
	}

	warning := enforcement.GenerateWarning(results, event.ExtractFilePath())
	blockEnabled := os.Getenv("GOGENT_DOC_THEATER_BLOCK") == "true"

	hasCritical := false
	for _, result := range results {
		if result.Severity == "critical" {
			hasCritical = true
			break
		}
	}

	if blockEnabled && hasCritical {
		return BlockResponse(warning)
	}

	return WarnResponse(warning)
}

// Main is the entrypoint for the gogent-doc-theater hook.
func Main() {
	event, err := routing.ParseToolEvent(os.Stdin, DefaultTimeout)
	if err != nil {
		OutputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	if !event.IsWriteOperation() {
		OutputAllow()
		return
	}

	if !event.IsClaudeMDFile() {
		OutputAllow()
		return
	}

	content := event.ExtractWriteContent()
	if content == "" {
		OutputAllow()
		return
	}

	results := enforcement.AnalyzeToolEventForDocTheater(event)
	response := GenerateDocTheaterResponse(event, results)
	fmt.Println(response)
}
