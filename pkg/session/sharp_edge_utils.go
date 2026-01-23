package session

import (
	"bufio"
	"os"
	"strings"
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
	scanner := bufio.NewScanner(f)
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
