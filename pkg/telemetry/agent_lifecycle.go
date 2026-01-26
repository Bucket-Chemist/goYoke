package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/google/uuid"
)

// AgentLifecycleEvent captures agent spawn and completion events for TUI real-time tracking
type AgentLifecycleEvent struct {
	// Identity
	EventID   string `json:"event_id"`   // UUID
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	EventType string `json:"event_type"` // "spawn" | "complete" | "error"

	// Agent identity
	AgentID     string `json:"agent_id"`      // "python-pro", etc.
	ParentAgent string `json:"parent_agent"`  // "terminal" or parent agent
	Tier        string `json:"tier"`          // "haiku", "sonnet", etc.

	// Task context
	TaskDescription string `json:"task_description"`
	DecisionID      string `json:"decision_id"` // Links to routing-decisions.jsonl

	// Completion data (only for "complete"/"error")
	Success      *bool   `json:"success,omitempty"`
	DurationMs   *int64  `json:"duration_ms,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// NewAgentLifecycleEvent creates a new lifecycle event with auto-populated timestamp and UUID
func NewAgentLifecycleEvent(sessionID, eventType, agentID, parentAgent, tier, taskDesc, decisionID string) *AgentLifecycleEvent {
	return &AgentLifecycleEvent{
		EventID:         uuid.New().String(),
		SessionID:       sessionID,
		Timestamp:       time.Now().Unix(),
		EventType:       eventType,
		AgentID:         agentID,
		ParentAgent:     parentAgent,
		Tier:            tier,
		TaskDescription: truncateDescription(taskDesc, 100),
		DecisionID:      decisionID,
	}
}

// LogAgentLifecycle writes lifecycle event to JSONL storage
// Uses append-only pattern (O_APPEND flag) for thread safety under parallel agent execution.
func LogAgentLifecycle(event *AgentLifecycleEvent) error {
	path := getAgentLifecyclePath()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[agent-lifecycle] Failed to marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[agent-lifecycle] Failed to create directory: %w", err)
	}

	// O_APPEND ensures thread-safe writes (atomic append on POSIX systems)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[agent-lifecycle] Failed to open log: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("[agent-lifecycle] Failed to write: %w", err)
	}

	return nil
}

// ReadAgentLifecycleLogs reads lifecycle logs, optionally filtered by session ID
// Uses manual JSONL parsing without external libraries for minimal dependencies.
func ReadAgentLifecycleLogs(sessionID string) ([]AgentLifecycleEvent, error) {
	logPath := getAgentLifecyclePath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentLifecycleEvent{}, nil
		}
		return nil, fmt.Errorf("[agent-lifecycle] Failed to open log: %w", err)
	}
	defer file.Close()

	var logs []AgentLifecycleEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var log AgentLifecycleEvent
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines but continue
			continue
		}

		// Filter by session ID if provided
		if sessionID != "" && log.SessionID != sessionID {
			continue
		}

		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[agent-lifecycle] Error reading log: %w", err)
	}

	return logs, nil
}

// getAgentLifecyclePath returns XDG-compliant agent lifecycle log path
// Checks GOGENT_PROJECT_DIR first for test isolation, falls back to XDG paths
func getAgentLifecyclePath() string {
	return config.GetAgentLifecyclePathWithProjectDir()
}
