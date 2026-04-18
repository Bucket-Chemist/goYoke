// Package util provides shared utility functions for the GOgent-Fortress TUI.
package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ErrorDisplay holds a formatted error with category, message, and optional
// suggestion. It is a pure data type; rendering is delegated to FormatError.
type ErrorDisplay struct {
	// Category is a short classifier such as "Connection" or "Timeout".
	Category string
	// Message is the human-readable error description.
	Message string
	// Suggestion is an optional remediation hint. When empty, FormatError
	// omits the suggestion line entirely.
	Suggestion string
	// Timestamp records when the error was captured.
	Timestamp time.Time
}

// FormatError renders an ErrorDisplay using the theme's error styling.
//
// Output format (single line when no suggestion):
//
//	[error_icon] Category: message
//
// When Suggestion is non-empty a second line is appended:
//
//	  Suggestion: <suggestion text>  (muted style)
func FormatError(ed ErrorDisplay, theme config.Theme) string {
	icon := theme.Icons().Error
	categoryPart := theme.ErrorStyle().Render(fmt.Sprintf("%s %s:", icon, ed.Category))
	line := fmt.Sprintf("%s %s", categoryPart, ed.Message)

	if ed.Suggestion == "" {
		return line
	}

	suggestionPart := theme.Muted.Render(fmt.Sprintf("  Suggestion: %s", ed.Suggestion))
	return line + "\n" + suggestionPart
}

// FormatWarning renders a warning-level message using the theme's warning
// styling.
//
// Output format:
//
//	[warning_icon] Category: message
func FormatWarning(category, message string, theme config.Theme) string {
	icon := theme.Icons().Warning
	categoryPart := theme.WarningStyle().Render(fmt.Sprintf("%s %s:", icon, category))
	return fmt.Sprintf("%s %s", categoryPart, message)
}

// ClassifyError maps common error message patterns to a short category string.
// Matching is case-insensitive. If err is nil or no pattern matches, "Error"
// is returned.
//
// Recognised patterns → category:
//   - "connection refused" | "no such host" | "dial" → "Connection"
//   - "permission denied" | "access denied"          → "Permission"
//   - "timeout" | "deadline exceeded"                → "Timeout"
//   - "rate limit" | "429" | "too many requests"     → "Rate Limit"
func ClassifyError(err error) string {
	if err == nil {
		return "Error"
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial"):
		return "Connection"

	case strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "access denied"):
		return "Permission"

	case strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded"):
		return "Timeout"

	case strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "too many requests"):
		return "Rate Limit"

	default:
		return "Error"
	}
}
