package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// ReminderContext represents routing reminder context
type ReminderContext struct {
	ToolCount      int    `json:"tool_count"`
	CurrentSession string `json:"session_id"`
	TiersSummary   string `json:"routing_summary"`
}

// FlushContext represents learning flush context
type FlushContext struct {
	EntryCount       int    `json:"entries_flushed"`
	ArchivedFile     string `json:"archived_to"`
	PendingRemaining int    `json:"remaining_entries"`
}

const (
	DefaultFlushThreshold = 5
)

// GetFlushThreshold returns configurable flush threshold from env var.
// Defaults to 5 if not set or invalid.
func GetFlushThreshold() int {
	if v := os.Getenv("GOYOKE_FLUSH_THRESHOLD"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return DefaultFlushThreshold
}

// GenerateRoutingReminder creates attention-gate reminder message
func GenerateRoutingReminder(toolCount int, routingSummary string) string {
	reminder := fmt.Sprintf(`🔔 ROUTING CHECKPOINT (Tool #%d)

Session routing compliance check:

ACTIVE ROUTING TIERS:
%s

At this checkpoint, verify:
1. ✅ Are you delegating exploratory work to codebase-search?
2. ✅ Are you using haiku for mechanical tasks?
3. ✅ Are you using sonnet for implementation?
4. ✅ Have you scouted unknown-scope tasks first?

If ANY of these need correction, pause and re-route.
See routing-schema.json for complete tier mappings.`,
		toolCount, routingSummary)

	return reminder
}

// CountPendingLearnings counts entries in pending-learnings.jsonl
// Returns the count as int for use in threshold comparisons.
// This complements the existing CheckPendingLearnings() which returns a formatted warning string.
func CountPendingLearnings(projectDir string) (int, error) {
	pendingPath := filepath.Join(config.ProjectMemoryDir(projectDir), "pending-learnings.jsonl")

	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return 0, nil // No pending learnings
	}
	if err != nil {
		return 0, fmt.Errorf("[session] Failed to read pending learnings: %w", err)
	}

	// Count lines
	lineCount := strings.Count(string(data), "\n")

	return lineCount, nil
}

// ShouldFlushLearnings checks if pending learnings exceed configurable threshold
func ShouldFlushLearnings(projectDir string) (bool, int, error) {
	count, err := CountPendingLearnings(projectDir)
	if err != nil {
		return false, 0, err
	}

	threshold := GetFlushThreshold()
	return count >= threshold, count, nil
}

// ArchivePendingLearnings moves entries to timestamped archive
func ArchivePendingLearnings(projectDir string) (*FlushContext, error) {
	pendingPath := filepath.Join(config.ProjectMemoryDir(projectDir), "pending-learnings.jsonl")
	sharpEdgesDir := filepath.Join(config.ProjectMemoryDir(projectDir), "sharp-edges")

	// Check if pending file exists before acquiring lock
	data, err := os.ReadFile(pendingPath)
	if os.IsNotExist(err) {
		return &FlushContext{EntryCount: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[session] Failed to read pending: %w", err)
	}

	// Acquire lock to prevent concurrent flush
	lockPath := pendingPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("[session] Failed to create lock file: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return nil, fmt.Errorf("[session] Failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	// Create archive directory
	if err := os.MkdirAll(sharpEdgesDir, 0755); err != nil {
		return nil, fmt.Errorf("[session] Failed to create archive dir: %w", err)
	}

	// Create timestamped archive file
	timestamp := time.Now().Format("20060102-150405")
	archivePath := filepath.Join(sharpEdgesDir, fmt.Sprintf("auto-flush-%s.jsonl", timestamp))

	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return nil, fmt.Errorf("[session] Failed to write archive: %w", err)
	}

	// Clear pending learnings
	if err := os.WriteFile(pendingPath, []byte(""), 0644); err != nil {
		return nil, fmt.Errorf("[session] Failed to clear pending: %w", err)
	}

	// Count entries
	entryCount := strings.Count(string(data), "\n")

	return &FlushContext{
		EntryCount:       entryCount,
		ArchivedFile:     archivePath,
		PendingRemaining: 0,
	}, nil
}

// GenerateFlushNotification creates notification about flushed learnings
func GenerateFlushNotification(ctx *FlushContext) string {
	notification := fmt.Sprintf(`📦 LEARNING AUTO-FLUSH

Archived %d sharp edges to:
%s

This prevents data loss on session interruption (Ctrl+C).

After session: Review auto-flush entries and decide:
- ✅ Merge into permanent sharp-edges if pattern confirmed
- ✅ Add to agent sharp-edges.yaml if agent-specific
- ❌ Delete if false alarm

See memory/sharp-edges/ for all archived learnings.`,
		ctx.EntryCount, ctx.ArchivedFile)

	return notification
}

// GenerateGateResponse creates attention-gate hook response
func GenerateGateResponse(shouldRemind bool, shouldFlush bool, reminderMsg string, flushMsg string) string {
	var contextParts []string

	if shouldRemind {
		contextParts = append(contextParts, reminderMsg)
	}

	if shouldFlush {
		contextParts = append(contextParts, flushMsg)
	}

	additionalContext := strings.Join(contextParts, "\n\n")

	response := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PostToolUse",
			"additionalContext": additionalContext,
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}
