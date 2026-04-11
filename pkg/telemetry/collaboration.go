package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/google/uuid"
)

// AgentCollaboration captures a delegation relationship for ML analysis
type AgentCollaboration struct {
	CollaborationID string `json:"collaboration_id"` // UUID
	Timestamp       int64  `json:"timestamp"`
	SessionID       string `json:"session_id"`

	// Delegation relationship
	ParentAgent    string `json:"parent_agent"`    // orchestrator, architect, etc.
	ChildAgent     string `json:"child_agent"`     // python-pro, codebase-search, etc.
	DelegationType string `json:"delegation_type"` // "spawn", "escalate", "parallel"

	// Context transfer
	ContextSize     int    `json:"context_size"`     // Tokens passed to child
	TaskDescription string `json:"task_description"` // What was delegated (truncated)

	// Outcome
	ChildSuccess    bool   `json:"child_success"`
	ChildDurationMs int64  `json:"child_duration_ms"`
	HandoffFriction string `json:"handoff_friction,omitempty"` // "context_loss", "misunderstanding", "none"

	// Chain position
	ChainDepth int    `json:"chain_depth"`  // 0 = root, 1 = first delegation, etc.
	RootTaskID string `json:"root_task_id"` // Original task that spawned chain

	// Swarm coordination (Addendum A.3)
	IsSwarmMember         bool    `json:"is_swarm_member,omitempty"`
	SwarmPosition         int     `json:"swarm_position,omitempty"`
	OverlapWithPrevious   float64 `json:"overlap_with_previous,omitempty"`
	AgreementWithAdjacent float64 `json:"agreement_with_adjacent,omitempty"`
	InformationLoss       float64 `json:"information_loss,omitempty"`
}

// NewAgentCollaboration creates a new collaboration record
func NewAgentCollaboration(sessionID, parentAgent, childAgent, delegationType string) *AgentCollaboration {
	return &AgentCollaboration{
		CollaborationID: uuid.New().String(),
		Timestamp:       time.Now().Unix(),
		SessionID:       sessionID,
		ParentAgent:     parentAgent,
		ChildAgent:      childAgent,
		DelegationType:  delegationType,
	}
}

// LogCollaboration writes collaboration to JSONL storage
// Uses append-only pattern (O_APPEND flag) for thread safety under parallel agent execution.
func LogCollaboration(collab *AgentCollaboration) error {
	globalPath := getGlobalCollaborationPath()

	data, err := json.Marshal(collab)
	if err != nil {
		return fmt.Errorf("[collaboration] Failed to marshal: %w", err)
	}

	dir := filepath.Dir(globalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[collaboration] Failed to create directory: %w", err)
	}

	// O_APPEND ensures thread-safe writes (atomic append on POSIX systems)
	f, err := os.OpenFile(globalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[collaboration] Failed to open log: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("[collaboration] Failed to write: %w", err)
	}

	return nil
}

// getGlobalCollaborationPath returns XDG-compliant global collaboration log path
// Checks GOGENT_PROJECT_DIR first for test isolation, falls back to XDG paths
func getGlobalCollaborationPath() string {
	return config.GetCollaborationsPathWithProjectDir()
}

// ReadCollaborationLogs reads all collaboration logs from the global path
// Uses manual JSONL parsing without external libraries for minimal dependencies.
func ReadCollaborationLogs() ([]AgentCollaboration, error) {
	logPath := getGlobalCollaborationPath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentCollaboration{}, nil
		}
		return nil, fmt.Errorf("[collaboration] Failed to open log: %w", err)
	}
	defer file.Close()

	var logs []AgentCollaboration
	scanner := newTelemetryScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var log AgentCollaboration
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines but continue
			continue
		}

		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[collaboration] Error reading log: %w", err)
	}

	return logs, nil
}

// CalculateCollaborationStats returns aggregate metrics for agent collaborations
func CalculateCollaborationStats(logs []AgentCollaboration) map[string]interface{} {
	if len(logs) == 0 {
		return map[string]interface{}{
			"collaboration_count": 0,
			"avg_chain_depth":     0,
			"success_rate":        "0.00%",
			"avg_handoff_time":    int64(0),
			"agent_pairings":      map[string]int{},
		}
	}

	var successCount int
	var totalChainDepth int
	var totalDuration int64
	agentPairings := make(map[string]int)

	for _, log := range logs {
		if log.ChildSuccess {
			successCount++
		}
		totalChainDepth += log.ChainDepth
		totalDuration += log.ChildDurationMs

		pairing := log.ParentAgent + " → " + log.ChildAgent
		agentPairings[pairing]++
	}

	successRate := float64(successCount) / float64(len(logs)) * 100
	avgChainDepth := totalChainDepth / len(logs)
	avgDuration := totalDuration / int64(len(logs))

	return map[string]interface{}{
		"collaboration_count": len(logs),
		"avg_chain_depth":     avgChainDepth,
		"success_rate":        fmt.Sprintf("%.2f%%", successRate),
		"avg_handoff_time":    avgDuration,
		"agent_pairings":      agentPairings,
	}
}
