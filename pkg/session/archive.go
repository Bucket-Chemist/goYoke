package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// ArchiveArtifacts moves session artifacts to timestamped archive directory
func ArchiveArtifacts(cfg HandoffConfig, sessionID string) error {
	timestamp := time.Now().Unix()

	// Ensure archive directory exists
	archiveDir := filepath.Join(filepath.Dir(cfg.HandoffPath), "session-archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("[session-archive] Failed to create archive directory %s: %w. Check write permissions.", archiveDir, err)
	}

	// Move pending learnings
	learningsPath := filepath.Join(filepath.Dir(cfg.HandoffPath), "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err == nil {
		destPath := filepath.Join(archiveDir, fmt.Sprintf("learnings-%d.jsonl", timestamp))
		if err := moveFile(learningsPath, destPath); err != nil {
			return fmt.Errorf("[session-archive] Failed to move learnings to %s: %w. File may be locked.", destPath, err)
		}
	}
	// Not fatal if missing - learnings may be empty

	// Move routing violations
	violationsPath := cfg.ViolationsPath
	if violationsPath != "" {
		if _, err := os.Stat(violationsPath); err == nil {
			destPath := filepath.Join(archiveDir, fmt.Sprintf("violations-%d.jsonl", timestamp))
			if err := moveFile(violationsPath, destPath); err != nil {
				return fmt.Errorf("[session-archive] Failed to move violations to %s: %w. File may be locked.", destPath, err)
			}
		}
	}

	// Copy transcript if available (optional)
	if cfg.TranscriptPath != "" {
		if _, err := os.Stat(cfg.TranscriptPath); err == nil {
			destPath := filepath.Join(archiveDir, fmt.Sprintf("session-%s.jsonl", sessionID))
			if err := copyFile(cfg.TranscriptPath, destPath); err != nil {
				// Non-fatal - transcript archival is best-effort
				fmt.Fprintf(os.Stderr, "[session-archive] Warning: Failed to copy transcript: %v\n", err)
			}
		}
	}

	return nil
}

// moveFile moves src to dst, handling cross-filesystem moves
func moveFile(src, dst string) error {
	// Try os.Rename first (fast path for same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename failed due to cross-device link, copy and delete
	if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err.Error() == "invalid cross-device link" {
		if copyErr := copyFile(src, dst); copyErr != nil {
			return fmt.Errorf("[session-archive] Failed to copy file during cross-device move: %w", copyErr)
		}
		if removeErr := os.Remove(src); removeErr != nil {
			return fmt.Errorf("[session-archive] Failed to remove source file after copy: %w", removeErr)
		}
		return nil
	}

	// Other rename errors are unexpected
	return err
}

// copyFile copies src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}

	return nil
}

// CleanupTempFiles removes session-specific temporary files
// Missing files are not treated as errors - only unexpected failures are reported
func CleanupTempFiles() error {
	gogentDir := config.GetGOgentDir()

	// Clean tool counter logs (glob pattern)
	counterPattern := filepath.Join(gogentDir, "claude-tool-counter-*.log")
	matches, err := filepath.Glob(counterPattern)
	if err != nil {
		return fmt.Errorf("[session-archive] Failed to glob pattern %s: %w", counterPattern, err)
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("[session-archive] Failed to remove %s: %w", match, err)
		}
	}

	// Clean current-tier file
	currentTierPath := filepath.Join(gogentDir, "current-tier")
	if err := os.Remove(currentTierPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("[session-archive] Failed to remove %s: %w", currentTierPath, err)
	}

	// Clean max_delegation file
	maxDelegationPath := filepath.Join(gogentDir, "max_delegation")
	if err := os.Remove(maxDelegationPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("[session-archive] Failed to remove %s: %w", maxDelegationPath, err)
	}

	return nil
}
