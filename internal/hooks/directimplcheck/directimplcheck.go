// Package directimplcheck implements the goyoke-direct-impl-check hook.
// It warns when the router writes implementation code directly instead of delegating.
package directimplcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// DefaultTimeout is the read timeout for stdin events.
const DefaultTimeout = 5 * time.Second

// GetLogPath returns the path for direct implementation check logs.
func GetLogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "goyoke-fortress", "direct-impl-check.jsonl")
}

// LogDirectImplCheck writes a direct implementation detection event to the log file.
func LogDirectImplCheck(sessionID, filePath, toolName string, lineCount int, suggestedAgent string) {
	logPath := GetLogPath()

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Unix()
	entry := fmt.Sprintf(`{"timestamp":%d,"session_id":%q,"file_path":%q,"tool_name":%q,"line_count":%d,"suggested_agent":%q}%s`,
		timestamp, sessionID, filePath, toolName, lineCount, suggestedAgent, "\n")
	f.WriteString(entry)
}

// SuggestAgent suggests the appropriate agent based on file path and content.
func SuggestAgent(filePath, content string) string {
	if strings.Contains(filePath, "/tui/") || strings.Contains(filePath, "tui_") {
		return "go-tui"
	}
	if strings.Contains(filePath, "/cmd/") {
		return "go-cli"
	}
	if strings.Contains(filePath, "/api/") || strings.Contains(filePath, "api_") {
		return "go-api"
	}

	if strings.Contains(content, "goroutine") ||
		strings.Contains(content, "errgroup") ||
		strings.Contains(content, "channel") ||
		strings.Contains(content, "sync.") ||
		strings.Contains(content, "go func") {
		return "go-concurrent"
	}

	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go-pro"
	case ".py":
		return "python-pro"
	case ".r", ".R":
		return "r-pro"
	case ".ts", ".js":
		return "typescript-pro"
	default:
		return "implementation-agent"
	}
}

// IsExcluded checks if file matches exclusion patterns.
func IsExcluded(filePath string, excludePatterns []string) bool {
	filename := filepath.Base(filePath)

	for _, pattern := range excludePatterns {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}

		if strings.Contains(pattern, "/") {
			if strings.Contains(filePath, strings.TrimSuffix(pattern, "*")) {
				return true
			}
		}
	}
	return false
}

// IsImplementationFile checks if file is an implementation file based on config.
func IsImplementationFile(filePath string, config *routing.DirectImplCheckConfig) bool {
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

	for _, implPath := range config.ImplementationPaths {
		if strings.Contains(filePath, implPath) {
			return true
		}
	}
	return false
}

// CountLines returns the number of lines in the content string.
func CountLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

// Main is the entrypoint for the goyoke-direct-impl-check hook.
func Main() {
	schema, err := routing.LoadSchema()
	if err != nil {
		fmt.Println("{}")
		return
	}

	if !schema.DirectImplCheck.Enabled {
		fmt.Println("{}")
		return
	}

	event, err := routing.ParseToolEvent(os.Stdin, DefaultTimeout)
	if err != nil {
		fmt.Println("{}")
		return
	}

	if event.ToolName != "Write" && event.ToolName != "Edit" {
		fmt.Println("{}")
		return
	}

	filePath := event.ExtractFilePath()
	if filePath == "" {
		fmt.Println("{}")
		return
	}

	if !IsImplementationFile(filePath, &schema.DirectImplCheck) {
		fmt.Println("{}")
		return
	}

	if IsExcluded(filePath, schema.DirectImplCheck.ExcludedPatterns) {
		fmt.Println("{}")
		return
	}

	var lineCount int
	if event.ToolName == "Write" {
		content := event.ExtractWriteContent()
		lineCount = CountLines(content)
	} else if event.ToolName == "Edit" {
		newContent := event.ExtractWriteContent()
		oldContent, _ := event.ToolInput["old_string"].(string)
		newLines := CountLines(newContent)
		oldLines := CountLines(oldContent)
		lineCount = newLines - oldLines
		if lineCount < 0 {
			lineCount = 0
		}
	}

	threshold := schema.DirectImplCheck.WriteThresholdLines
	if event.ToolName == "Edit" {
		threshold = schema.DirectImplCheck.EditThresholdLines
	}

	if lineCount < threshold {
		fmt.Println("{}")
		return
	}

	content := event.ExtractWriteContent()
	suggestedAgent := SuggestAgent(filePath, content)
	LogDirectImplCheck(event.SessionID, filePath, event.ToolName, lineCount, suggestedAgent)

	warningMsg := fmt.Sprintf(
		"[ROUTING CHECK] Direct implementation detected (%d lines to %s). Consider delegating to %s. If intentional, continue.",
		lineCount,
		filepath.Base(filePath),
		suggestedAgent,
	)

	response := routing.NewPassResponse("PreToolUse")
	response.AddField("additionalContext", warningMsg)

	if err := response.Marshal(os.Stdout); err != nil {
		fmt.Println("{}")
	}
}
