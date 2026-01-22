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
)

// EscalationEvent captures when an agent escalates to a higher tier.
// Tracks Einstein/Gemini invocations and their outcomes.
type EscalationEvent struct {
	// Core identification
	Timestamp    string `json:"timestamp"`      // RFC3339 format
	SessionID    string `json:"session_id"`
	EscalationID string `json:"escalation_id"`  // UUID for tracking

	// Escalation context
	FromTier  string `json:"from_tier"`  // e.g., "sonnet"
	ToTier    string `json:"to_tier"`    // e.g., "opus"
	FromAgent string `json:"from_agent"` // e.g., "orchestrator"
	ToAgent   string `json:"to_agent"`   // e.g., "einstein"

	// Trigger information
	Reason      string `json:"reason"`       // e.g., "3 consecutive failures"
	TriggerType string `json:"trigger_type"` // "failure_cascade", "user_request", "complexity"

	// Associated artifacts
	GAPDocPath string `json:"gap_doc_path,omitempty"` // Path to GAP document if generated

	// Outcome tracking (updated after resolution)
	Outcome           string `json:"outcome"`                        // "pending", "resolved", "still_blocked"
	ResolutionTimeMs  int64  `json:"resolution_time_ms,omitempty"`
	ResolutionSummary string `json:"resolution_summary,omitempty"`
	TokensUsed        int    `json:"tokens_used,omitempty"`

	// Context
	ProjectDir string `json:"project_dir,omitempty"`
}

// Valid trigger types for escalation events
var ValidTriggerTypes = map[string]bool{
	"failure_cascade": true, // 3+ consecutive failures
	"user_request":    true, // User explicitly asked for Einstein
	"complexity":      true, // Scout/orchestrator determined higher tier needed
	"timeout":         true, // Lower tier couldn't complete in time
	"cost_ceiling":    true, // Lower tier exceeded budget without result
}

// Valid outcome states for escalation events
var ValidOutcomes = map[string]bool{
	"pending":       true, // Escalation in progress
	"resolved":      true, // Successfully resolved
	"still_blocked": true, // Einstein couldn't solve it either
	"cancelled":     true, // User cancelled before completion
}

// GetEscalationsLogPath returns the global escalations log path.
func GetEscalationsLogPath() string {
	baseDir := config.GetGOgentDir()
	return filepath.Join(baseDir, "escalations.jsonl")
}

// GetProjectEscalationsLogPath returns project-scoped escalations log path.
func GetProjectEscalationsLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "escalations.jsonl")
}

// LogEscalation appends escalation event to BOTH global and project logs.
// Timestamp is auto-populated. Returns the escalation ID for tracking.
func LogEscalation(esc *EscalationEvent, projectDir string) (string, error) {
	// Auto-populate timestamp
	esc.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory if provided
	if projectDir != "" {
		esc.ProjectDir = projectDir
	}

	// Set initial outcome if not specified
	if esc.Outcome == "" {
		esc.Outcome = "pending"
	}

	// Validate trigger type
	if esc.TriggerType != "" && !ValidTriggerTypes[esc.TriggerType] {
		return "", fmt.Errorf("[escalations] Invalid trigger type '%s'. Valid types: failure_cascade, user_request, complexity, timeout, cost_ceiling", esc.TriggerType)
	}

	// Marshal once, write twice
	data, err := json.Marshal(esc)
	if err != nil {
		return "", fmt.Errorf("[escalations] Failed to marshal escalation: %w", err)
	}
	data = append(data, '\n')

	// WRITE 1: Global XDG cache
	globalPath := GetEscalationsLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return "", fmt.Errorf("[escalations] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (optional)
	if projectDir != "" {
		projectPath := GetProjectEscalationsLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "[escalations] Warning: Failed project log: %v\n", err)
		}
	}

	return esc.EscalationID, nil
}

// LoadEscalations reads all escalation events from a JSONL file.
// Returns empty slice for missing file.
func LoadEscalations(path string) ([]EscalationEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []EscalationEvent{}, nil
		}
		return nil, fmt.Errorf("[escalations] Failed to open %s: %w", path, err)
	}
	defer file.Close()

	var escalations []EscalationEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var esc EscalationEvent
		if err := json.Unmarshal([]byte(line), &esc); err != nil {
			continue // Skip malformed lines
		}
		escalations = append(escalations, esc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[escalations] Error reading %s: %w", path, err)
	}

	return escalations, nil
}

// UpdateEscalationOutcome marks an escalation as resolved or still blocked.
// This rewrites the entire file to update the specific escalation.
// Note: For high-frequency updates, consider a more efficient approach.
func UpdateEscalationOutcome(path string, escalationID string, outcome string, resolutionMs int64, summary string) error {
	if !ValidOutcomes[outcome] {
		return fmt.Errorf("[escalations] Invalid outcome '%s'. Valid: pending, resolved, still_blocked, cancelled", outcome)
	}

	escalations, err := LoadEscalations(path)
	if err != nil {
		return err
	}

	found := false
	for i := range escalations {
		if escalations[i].EscalationID == escalationID {
			escalations[i].Outcome = outcome
			escalations[i].ResolutionTimeMs = resolutionMs
			escalations[i].ResolutionSummary = summary
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("[escalations] Escalation ID '%s' not found in %s", escalationID, path)
	}

	// Rewrite file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("[escalations] Failed to rewrite %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, esc := range escalations {
		if err := encoder.Encode(esc); err != nil {
			return fmt.Errorf("[escalations] Failed to write escalation: %w", err)
		}
	}

	return nil
}

// EscalationFilters defines filter criteria for escalation queries.
type EscalationFilters struct {
	TriggerType *string // Filter by trigger type
	Outcome     *string // Filter by outcome
	ToTier      *string // Filter by destination tier
	FromAgent   *string // Filter by source agent
	Since       *int64  // Filter by timestamp (Unix seconds)
	Limit       int     // Maximum results
}

// FilterEscalations applies filters to a list of escalations.
func FilterEscalations(escalations []EscalationEvent, filters EscalationFilters) []EscalationEvent {
	var filtered []EscalationEvent

	for _, esc := range escalations {
		// Apply filters
		if filters.TriggerType != nil && esc.TriggerType != *filters.TriggerType {
			continue
		}
		if filters.Outcome != nil && esc.Outcome != *filters.Outcome {
			continue
		}
		if filters.ToTier != nil && esc.ToTier != *filters.ToTier {
			continue
		}
		if filters.FromAgent != nil && esc.FromAgent != *filters.FromAgent {
			continue
		}
		if filters.Since != nil {
			escTime, _ := time.Parse(time.RFC3339, esc.Timestamp)
			if escTime.Unix() < *filters.Since {
				continue
			}
		}

		filtered = append(filtered, esc)

		if filters.Limit > 0 && len(filtered) >= filters.Limit {
			break
		}
	}

	return filtered
}
