package routing

import (
	"fmt"
	"regexp"
	"strings"
)

// FailureInfo captures detected failure information
// Compatible with session.SharpEdge struct for seamless integration
type FailureInfo struct {
	File       string `json:"file"`
	ErrorType  string `json:"error_type"`
	Timestamp  int64  `json:"timestamp"`
	Tool       string `json:"tool,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	ErrorMatch string `json:"error_match,omitempty"` // The specific text that matched
}

// DetectFailure analyzes PostToolEvent for failure signals
// Returns nil if no failure detected
func DetectFailure(event *PostToolEvent) *FailureInfo {
	if event == nil || event.ToolResponse == nil {
		return nil
	}

	info := &FailureInfo{
		File:      ExtractFilePath(event),
		Timestamp: event.CapturedAt,
		Tool:      event.ToolName,
	}

	// Priority 1: Explicit success=false
	if success, ok := event.ToolResponse["success"].(bool); ok && !success {
		info.ErrorType = "explicit_failure"
		return info
	}

	// Priority 2: Non-zero exit code
	if exitCode := extractExitCode(event.ToolResponse); exitCode != 0 {
		info.ErrorType = formatExitCode(exitCode)
		info.ExitCode = exitCode
		return info
	}

	// Priority 3: Error patterns in output
	output := extractOutput(event.ToolResponse)
	if errorType, match := detectErrorKeywords(output); errorType != "" {
		info.ErrorType = errorType
		info.ErrorMatch = match
		return info
	}

	return nil
}

// ExtractFilePath extracts the target file from event
// Tries: file_path → command (first arg) → "unknown"
func ExtractFilePath(event *PostToolEvent) string {
	if event == nil || event.ToolInput == nil {
		return "unknown"
	}

	// Try file_path first
	if fp, ok := event.ToolInput["file_path"].(string); ok && fp != "" {
		return fp
	}

	// Try command (extract first path-like argument)
	if cmd, ok := event.ToolInput["command"].(string); ok {
		return extractPathFromCommand(cmd)
	}

	return "unknown"
}

// Error detection patterns
var pythonErrors = regexp.MustCompile(`(?i)(TypeError|ValueError|AttributeError|ImportError|SyntaxError|NameError|KeyError|IndexError|FileNotFoundError|RuntimeError)`)
var genericErrors = regexp.MustCompile(`(?i)\b(error|failed|exception|traceback)\b`)

func detectErrorKeywords(text string) (errorType, match string) {
	// Check Python-specific errors first
	if m := pythonErrors.FindString(text); m != "" {
		return strings.ToLower(m), m
	}

	// Check generic error keywords
	if m := genericErrors.FindString(text); m != "" {
		return "generic_" + strings.ToLower(m), m
	}

	return "", ""
}

func extractExitCode(response map[string]interface{}) int {
	if code, ok := response["exit_code"].(float64); ok {
		return int(code)
	}
	if code, ok := response["exit_code"].(int); ok {
		return code
	}
	return 0
}

func extractOutput(response map[string]interface{}) string {
	if out, ok := response["output"].(string); ok {
		return out
	}
	if err, ok := response["error"].(string); ok {
		return err
	}
	return ""
}

func extractPathFromCommand(cmd string) string {
	// Simple heuristic: look for path-like arguments
	parts := strings.Fields(cmd)
	for _, part := range parts {
		if strings.Contains(part, "/") || strings.Contains(part, ".") {
			return part
		}
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func formatExitCode(code int) string {
	// Map common exit codes to semantic names for better composite keys
	switch code {
	case 1:
		return "general_error"
	case 2:
		return "misuse"
	case 126:
		return "not_executable"
	case 127:
		return "command_not_found"
	case 128:
		return "invalid_exit"
	case 130:
		return "interrupted"
	default:
		return fmt.Sprintf("exit_code_%d", code)
	}
}
