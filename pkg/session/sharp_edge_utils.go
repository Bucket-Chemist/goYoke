package session

import (
	"fmt"
	"os"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// ExtractCodeSnippet reads a file and extracts a context window around the specified line.
// Returns a snippet of [lineNumber-window : lineNumber+window] lines centered on lineNumber.
// Returns empty string (not error) for non-fatal issues like missing files or binary content.
//
// Parameters:
//   - filePath: Path to the file to read
//   - lineNumber: Target line number (1-indexed, as shown in editors)
//   - window: Number of lines to include before and after the target line
//
// Edge cases handled:
//   - File doesn't exist: returns empty string, no error
//   - File can't be opened: returns empty string, no error
//   - Empty file: returns empty string, no error
//   - Line number out of bounds: adjusts window to file boundaries
//   - Binary file (contains null bytes): returns empty string, no error
func ExtractCodeSnippet(filePath string, lineNumber int, window int) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil // File doesn't exist, return empty (not error)
	}

	// Read file
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil // Can't open, return empty
	}
	defer f.Close()

	// Read all lines
	var lines []string
	scanner := newSessionScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return "", nil // Empty file
	}

	// Calculate window bounds (lineNumber is 1-indexed)
	start := max(0, lineNumber-window-1) // -1 for 0-indexing
	end := min(len(lines), lineNumber+window)

	// Handle case where lineNumber is past EOF
	if start >= len(lines) {
		start = max(0, len(lines)-window-1)
	}
	if start >= end {
		start = max(0, end-1)
	}

	// Extract snippet
	snippet := strings.Join(lines[start:end], "\n")

	// Check if likely binary (contains null bytes)
	if strings.Contains(snippet, "\x00") {
		return "", nil // Binary file, skip
	}

	return snippet, nil
}

// ExtractAttemptedChange extracts what was attempted during a tool failure.
// Returns a formatted string showing the attempted change for different tool types.
//
// For Edit: Shows "old_string → new_string" transformation (truncated to 60 chars each)
// For Write: Shows first 3 lines of content being written
// For Bash: Shows the command being executed
// For other tools: Returns empty string
//
// Parameters:
//   - event: PostToolEvent containing tool input to analyze
//
// Returns:
//   - Formatted string describing the attempted change, or empty string if not applicable
func ExtractAttemptedChange(event *routing.PostToolEvent) string {
	if event == nil || event.ToolInput == nil {
		return ""
	}

	switch event.ToolName {
	case "Edit":
		return extractEditChange(event.ToolInput)
	case "Write":
		return extractWriteChange(event.ToolInput)
	case "Bash":
		return extractBashChange(event.ToolInput)
	default:
		return ""
	}
}

// extractEditChange formats Edit tool changes as "old → new"
func extractEditChange(toolInput map[string]interface{}) string {
	oldStr, _ := toolInput["old_string"].(string)
	newStr, _ := toolInput["new_string"].(string)

	// Truncate for display (60 chars max per side)
	oldStr = truncateString(oldStr, 60)
	newStr = truncateString(newStr, 60)

	// Handle empty strings
	if oldStr == "" && newStr == "" {
		return ""
	}
	if oldStr == "" {
		return fmt.Sprintf("(empty) → %s", newStr)
	}
	if newStr == "" {
		return fmt.Sprintf("%s → (empty)", oldStr)
	}

	return fmt.Sprintf("%s → %s", oldStr, newStr)
}

// extractWriteChange formats Write tool changes as first 3 lines preview
func extractWriteChange(toolInput map[string]interface{}) string {
	content, ok := toolInput["content"].(string)
	if !ok || content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")

	// Show first 3 lines
	maxLines := 3
	if len(lines) < maxLines {
		maxLines = len(lines)
	}

	preview := strings.Join(lines[:maxLines], "\n")
	if len(lines) > 3 {
		preview += "\n..."
	}

	return fmt.Sprintf("Write content:\n%s", preview)
}

// extractBashChange formats Bash tool changes as the command
func extractBashChange(toolInput map[string]interface{}) string {
	command, ok := toolInput["command"].(string)
	if !ok || command == "" {
		return ""
	}

	return fmt.Sprintf("Command: %s", command)
}
