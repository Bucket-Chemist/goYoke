package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// LoadHandoffSummary loads the last session handoff for context resumption.
// It returns the first 30 lines of the handoff file with a truncation message if longer.
// Returns empty string (not error) if file doesn't exist - this is normal for first sessions.
// Returns error if file is too large (>50KB) to protect against memory issues.
func LoadHandoffSummary(projectDir string) (string, error) {
	handoffPath := filepath.Join(config.ProjectMemoryDir(projectDir), "last-handoff.md")

	// Check if file exists - missing handoff is normal
	info, err := os.Stat(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Graceful degradation - no previous session
		}
		return "", fmt.Errorf("[context-loader] Failed to stat handoff at %s: %w. Check file permissions.", handoffPath, err)
	}

	// Protect against reading very large files
	const maxFileSize = 50 * 1024 // 50KB
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("[context-loader] Handoff file too large (%d bytes > %d bytes). File: %s. Skipping load to prevent memory issues.", info.Size(), maxFileSize, handoffPath)
	}

	// Open file for reading
	file, err := os.Open(handoffPath)
	if err != nil {
		return "", fmt.Errorf("[context-loader] Failed to open handoff at %s: %w. Check file permissions.", handoffPath, err)
	}
	defer file.Close()

	// Read first 30 lines
	const maxLines = 30
	var lines []string
	scanner := newSessionScanner(file)
	totalLines := 0

	for scanner.Scan() {
		totalLines++
		if len(lines) < maxLines {
			lines = append(lines, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("[context-loader] Failed to read handoff at %s: %w. File may be corrupted.", handoffPath, err)
	}

	// Build output with truncation notice if needed
	result := strings.Join(lines, "\n")
	if totalLines > maxLines {
		truncationMsg := fmt.Sprintf("\n\n(... %d lines truncated. Full handoff: %s)", totalLines-maxLines, handoffPath)
		result += truncationMsg
	}

	return result, nil
}

// CheckPendingLearnings checks for sharp edges captured by previous sessions.
// Returns a formatted warning message if pending learnings exist, empty string otherwise.
// This is a non-blocking check - missing or empty file returns empty string (not error).
func CheckPendingLearnings(projectDir string) (string, error) {
	learningsPath := filepath.Join(config.ProjectMemoryDir(projectDir), "pending-learnings.jsonl")

	// Check if file exists - missing learnings is normal
	file, err := os.Open(learningsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No pending learnings - this is normal
		}
		return "", fmt.Errorf("[context-loader] Failed to open pending learnings at %s: %w. Check file permissions.", learningsPath, err)
	}
	defer file.Close()

	// Count lines (each line = one sharp edge)
	scanner := newSessionScanner(file)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("[context-loader] Failed to read pending learnings at %s: %w. File may be corrupted.", learningsPath, err)
	}

	// Return empty string if no learnings (not an error, just normal)
	if count == 0 {
		return "", nil
	}

	// Format warning message
	return fmt.Sprintf("⚠️  PENDING LEARNINGS: %d sharp edge(s) from previous sessions need review.\n   Path: %s", count, learningsPath), nil
}

// FormatGitInfo formats git repository state for session context.
// Returns empty string if not a git repository (graceful degradation).
// Reuses collectGitInfo() from handoff.go to gather git state.
func FormatGitInfo(projectDir string) string {
	info := collectGitInfo(projectDir)

	// Not a git repo - return empty string silently
	if info.Branch == "" {
		return ""
	}

	// Format output based on dirty state
	if info.IsDirty {
		fileCount := len(info.Uncommitted)
		return fmt.Sprintf("GIT: Branch: %s | Uncommitted: %d file(s)", info.Branch, fileCount)
	}

	return fmt.Sprintf("GIT: Branch: %s | Clean working tree", info.Branch)
}
