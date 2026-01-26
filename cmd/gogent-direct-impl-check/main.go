package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

// getLogPath returns the path for direct implementation check logs.
func getLogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "gogent-fortress", "direct-impl-check.jsonl")
}

// logDirectImplCheck writes a direct implementation detection event to the log file.
func logDirectImplCheck(sessionID, filePath, toolName string, lineCount int, suggestedAgent string) {
	logPath := getLogPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return // Silent fail - don't block hook execution
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silent fail
	}
	defer f.Close()

	timestamp := time.Now().Unix()

	// JSONL format for easy parsing
	entry := fmt.Sprintf(`{"timestamp":%d,"session_id":%q,"file_path":%q,"tool_name":%q,"line_count":%d,"suggested_agent":%q}%s`,
		timestamp, sessionID, filePath, toolName, lineCount, suggestedAgent, "\n")

	f.WriteString(entry)
}

// suggestAgent suggests the appropriate agent based on file path and content.
func suggestAgent(filePath, content string) string {
	// Check path patterns first (more reliable)
	if strings.Contains(filePath, "/tui/") || strings.Contains(filePath, "tui_") {
		return "go-tui"
	}
	if strings.Contains(filePath, "/cmd/") {
		return "go-cli"
	}
	if strings.Contains(filePath, "/api/") || strings.Contains(filePath, "api_") {
		return "go-api"
	}

	// Check content patterns
	if strings.Contains(content, "goroutine") ||
		strings.Contains(content, "errgroup") ||
		strings.Contains(content, "channel") ||
		strings.Contains(content, "sync.") ||
		strings.Contains(content, "go func") {
		return "go-concurrent"
	}

	// Check file extension for language-specific agents
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go-pro"
	case ".py":
		// Could be enhanced with more specific Python agent detection
		return "python-pro"
	case ".r", ".R":
		return "r-pro"
	case ".ts", ".js":
		return "typescript-pro" // Placeholder - adjust based on your agent roster
	default:
		return "implementation-agent"
	}
}

// isExcluded checks if file matches exclusion patterns.
func isExcluded(filePath string, excludePatterns []string) bool {
	filename := filepath.Base(filePath)

	for _, pattern := range excludePatterns {
		// Handle glob patterns
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}

		// Handle directory patterns (e.g., "testdata/*")
		if strings.Contains(pattern, "/") {
			// Check if any part of the path matches
			if strings.Contains(filePath, strings.TrimSuffix(pattern, "*")) {
				return true
			}
		}
	}
	return false
}

// isImplementationFile checks if file is an implementation file based on config.
func isImplementationFile(filePath string, config *routing.DirectImplCheckConfig) bool {
	// Check extension
	ext := filepath.Ext(filePath)
	validExt := false
	for _, validExtension := range config.ImplementationExtensions {
		if ext == validExtension {
			validExt = true
			break
		}
	}
	if !validExt {
		return false
	}

	// Check if path contains any implementation paths
	for _, implPath := range config.ImplementationPaths {
		if strings.Contains(filePath, implPath) {
			return true
		}
	}

	return false
}

// countLines returns the number of lines in the content string.
func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func main() {
	// Load routing schema
	schema, err := routing.LoadSchema()
	if err != nil {
		// Graceful failure: output empty JSON and exit successfully
		fmt.Println("{}")
		return
	}

	// Check if direct_impl_check is enabled
	if !schema.DirectImplCheck.Enabled {
		fmt.Println("{}")
		return
	}

	// Parse event from STDIN with timeout
	event, err := routing.ParseToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		// Graceful failure: output empty JSON
		fmt.Println("{}")
		return
	}

	// Only process Write or Edit tools
	if event.ToolName != "Write" && event.ToolName != "Edit" {
		fmt.Println("{}")
		return
	}

	// Extract file path
	filePath := event.ExtractFilePath()
	if filePath == "" {
		// No file path, pass through
		fmt.Println("{}")
		return
	}

	// Check if this is an implementation file
	if !isImplementationFile(filePath, &schema.DirectImplCheck) {
		fmt.Println("{}")
		return
	}

	// Check exclusion patterns
	if isExcluded(filePath, schema.DirectImplCheck.ExcludedPatterns) {
		fmt.Println("{}")
		return
	}

	// Calculate line count based on tool type
	var lineCount int
	if event.ToolName == "Write" {
		// For Write, count all lines in content
		content := event.ExtractWriteContent()
		lineCount = countLines(content)
	} else if event.ToolName == "Edit" {
		// For Edit, calculate net addition (new_string - old_string)
		newContent := event.ExtractWriteContent() // This gets new_string for Edit
		oldContent, _ := event.ToolInput["old_string"].(string)

		newLines := countLines(newContent)
		oldLines := countLines(oldContent)

		// Net addition (can be negative for deletions)
		lineCount = newLines - oldLines

		// Only warn on additions, not deletions
		if lineCount < 0 {
			lineCount = 0
		}
	}

	// Check threshold
	threshold := schema.DirectImplCheck.WriteThresholdLines
	if event.ToolName == "Edit" {
		threshold = schema.DirectImplCheck.EditThresholdLines
	}

	if lineCount < threshold {
		// Below threshold, pass through
		fmt.Println("{}")
		return
	}

	// Threshold exceeded - suggest agent
	content := event.ExtractWriteContent()
	suggestedAgent := suggestAgent(filePath, content)

	// Log the detection
	logDirectImplCheck(event.SessionID, filePath, event.ToolName, lineCount, suggestedAgent)

	// Create warning message
	warningMsg := fmt.Sprintf(
		"[ROUTING CHECK] Direct implementation detected (%d lines to %s). Consider delegating to %s. If intentional, continue.",
		lineCount,
		filepath.Base(filePath),
		suggestedAgent,
	)

	// Create pass response with additionalContext (warning, not blocking)
	response := routing.NewPassResponse("PreToolUse")
	response.AddField("additionalContext", warningMsg)

	// Output response
	if err := response.Marshal(os.Stdout); err != nil {
		// Fallback: output empty JSON on marshal error
		fmt.Println("{}")
	}
}
