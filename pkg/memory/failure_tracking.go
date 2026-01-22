package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	// DefaultMaxFailures is the default threshold for debugging loop detection
	DefaultMaxFailures = 3

	// DefaultFailureWindow is the default time window in seconds for failure counting
	DefaultFailureWindow = 300
)

var (
	// DefaultStoragePath is the default location for the failure tracker JSONL file
	// Can be overridden via GOGENT_STORAGE_PATH environment variable or for testing
	DefaultStoragePath = "~/.gogent/failure-tracker.jsonl"
)

// LogFailure appends a failure entry to the tracker JSONL file.
// Creates the file and parent directory if they don't exist.
// Returns an error if the write fails.
func LogFailure(info *routing.FailureInfo) error {
	if info == nil {
		return fmt.Errorf("failure info cannot be nil")
	}

	// Resolve storage path
	path, err := expandPath(getStoragePath())
	if err != nil {
		return fmt.Errorf("failed to resolve storage path: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open file in append mode (create if doesn't exist)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	// Marshal to JSON
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal failure info: %w", err)
	}

	// Append newline-delimited JSON
	if _, err := fmt.Fprintf(file, "%s\n", data); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// GetFailureCount returns the number of failures for the specified file and error type
// within the configured time window. Uses a composite key (file + error_type) to prevent
// false positives from different error types.
//
// If the tracker file doesn't exist, returns 0 (not an error).
// Returns an error only if the file exists but cannot be read or parsed.
func GetFailureCount(file, errorType string) (int, error) {
	path, err := expandPath(getStoragePath())
	if err != nil {
		return 0, fmt.Errorf("failed to resolve storage path: %w", err)
	}

	// If file doesn't exist, there are no failures
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}

	// Get time window threshold
	window := getFailureWindow()
	threshold := time.Now().Unix() - int64(window)

	// Count matching failures within time window
	count := 0
	if err := scanFailures(path, func(info *routing.FailureInfo) bool {
		// Check composite key match AND time window
		if info.File == file && info.ErrorType == errorType && info.Timestamp > threshold {
			count++
		}
		return true // continue scanning
	}); err != nil {
		return 0, err
	}

	return count, nil
}

// ClearFailures removes all failure entries for the specified file and error type.
// This is typically called after a successful resolution to reset the debugging loop counter.
//
// Implementation uses a read-filter-write strategy:
//  1. Read all entries
//  2. Filter out matching entries
//  3. Rewrite the file with remaining entries
//
// If the file doesn't exist, this is a no-op (not an error).
func ClearFailures(file, errorType string) error {
	path, err := expandPath(getStoragePath())
	if err != nil {
		return fmt.Errorf("failed to resolve storage path: %w", err)
	}

	// If file doesn't exist, nothing to clear
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	// Collect all entries that don't match the filter
	var remaining []routing.FailureInfo
	if err := scanFailures(path, func(info *routing.FailureInfo) bool {
		// Keep entries that DON'T match the composite key
		if info.File != file || info.ErrorType != errorType {
			remaining = append(remaining, *info)
		}
		return true // continue scanning
	}); err != nil {
		return err
	}

	// Rewrite file with remaining entries
	return rewriteFailures(path, remaining)
}

// scanFailures reads the JSONL file and calls the visitor function for each entry.
// The visitor returns true to continue scanning, false to stop.
// Returns an error if the file cannot be read or entries cannot be parsed.
func scanFailures(path string, visitor func(*routing.FailureInfo) bool) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // skip empty lines
		}

		var info routing.FailureInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			// Skip malformed lines (graceful degradation)
			continue
		}

		if !visitor(&info) {
			break // visitor requested stop
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// rewriteFailures atomically replaces the tracker file with the provided entries.
// Uses a write-to-temp + atomic-rename strategy for safety.
func rewriteFailures(path string, entries []routing.FailureInfo) error {
	// Write to temporary file
	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write all entries
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		if _, err := fmt.Fprintf(tmpFile, "%s\n", data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// getStoragePath returns the storage path, checking GOGENT_STORAGE_PATH env var first
func getStoragePath() string {
	if path := os.Getenv("GOGENT_STORAGE_PATH"); path != "" {
		return path
	}
	return DefaultStoragePath
}

// expandPath converts ~ to the user's home directory
func expandPath(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if len(path) == 1 {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}

// getMaxFailures returns the configured maximum failure threshold from GOGENT_MAX_FAILURES.
// Returns DefaultMaxFailures if not set or invalid.
func getMaxFailures() int {
	if val := os.Getenv("GOGENT_MAX_FAILURES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxFailures
}

// getFailureWindow returns the configured failure time window from GOGENT_FAILURE_WINDOW.
// Returns DefaultFailureWindow if not set or invalid.
func getFailureWindow() int {
	if val := os.Getenv("GOGENT_FAILURE_WINDOW"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return DefaultFailureWindow
}

// IsDebuggingLoop checks if the number of failures for the given file and error type
// exceeds the configured threshold, indicating a debugging loop.
func IsDebuggingLoop(file, errorType string) (bool, error) {
	count, err := GetFailureCount(file, errorType)
	if err != nil {
		return false, err
	}
	return count >= getMaxFailures(), nil
}
