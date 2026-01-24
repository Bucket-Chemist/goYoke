package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/enforcement"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse PreToolUse event from STDIN
	event, err := routing.ParseToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Filter 1: Only check Write/Edit operations
	if !event.IsWriteOperation() {
		outputAllow()
		return
	}

	// Filter 2: Only check CLAUDE.md files
	if !event.IsClaudeMDFile() {
		outputAllow()
		return
	}

	// Extract content from tool_input (NOT re-reading STDIN)
	content := event.ExtractWriteContent()
	if content == "" {
		outputAllow() // No content to analyze
		return
	}

	// Detect theater patterns using enforcement.AnalyzeToolEventForDocTheater
	results := enforcement.AnalyzeToolEventForDocTheater(event)

	// Generate response based on detected patterns
	response := generateDocTheaterResponse(event, results)
	fmt.Println(response)
}

// generateDocTheaterResponse builds hook response based on detection results
func generateDocTheaterResponse(event *routing.ToolEvent, results []enforcement.DetectionResult) string {
	if len(results) == 0 {
		// No patterns detected - allow
		return allowResponse()
	}

	// Build warning message
	warning := enforcement.GenerateWarning(results, event.ExtractFilePath())

	// Check if blocking is enabled (default: warn only)
	blockEnabled := os.Getenv("GOGENT_DOC_THEATER_BLOCK") == "true"

	// Check for critical patterns
	hasCritical := false
	for _, result := range results {
		if result.Severity == "critical" {
			hasCritical = true
			break
		}
	}

	if blockEnabled && hasCritical {
		return blockResponse(warning)
	}

	return warnResponse(warning)
}

func allowResponse() string {
	return `{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`
}

func warnResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "warn",
    "additionalContext": "%s"
  }
}`, escapeJSON(message))
}

func blockResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "block",
    "additionalContext": "🚫 BLOCKED: %s"
  }
}`, escapeJSON(message))
}

func outputAllow() {
	fmt.Println(allowResponse())
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, escapeJSON(message))
}

func escapeJSON(s string) string {
	// Basic JSON escaping for embedded strings
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
