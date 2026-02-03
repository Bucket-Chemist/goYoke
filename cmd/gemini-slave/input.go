package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// CaptureInput reads input from stdin or file arguments.
// CRITICAL: This must be called FIRST in main() before any other I/O operations
// to avoid the race condition that plagued the bash version.
//
// Priority:
// 1. Stdin (if piped)
// 2. File arguments
// 3. Empty string (valid for some protocols)
func CaptureInput(fileArgs []string) (string, error) {
	// Check stdin FIRST, before any other I/O
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("checking stdin: %w", err)
	}

	// Stdin is piped - read ALL of it immediately
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	// No stdin - check for file arguments
	if len(fileArgs) > 0 {
		var parts []string
		for _, f := range fileArgs {
			data, err := os.ReadFile(f)
			if err != nil {
				return "", fmt.Errorf("reading file %s: %w", f, err)
			}
			parts = append(parts, string(data))
		}
		return strings.Join(parts, "\n"), nil
	}

	// No input is valid for some protocols (they use only instruction)
	return "", nil
}
