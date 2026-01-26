package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/google/uuid"
)

// RoutingDecision captures a single routing choice for ML training
type RoutingDecision struct {
	// Identity
	DecisionID string `json:"decision_id"`
	Timestamp  int64  `json:"timestamp"`
	SessionID  string `json:"session_id"`

	// Input Context (Features for ML)
	TaskDescription   string   `json:"task_description"`
	TaskType          string   `json:"task_type"`
	TaskDomain        string   `json:"task_domain"`
	DetectedPatterns  []string `json:"detected_patterns"`
	ContextWindowUsed int      `json:"context_window_used"`
	SessionToolCount  int      `json:"session_tool_count"`
	RecentSuccessRate float64  `json:"recent_success_rate"`

	// Decision Made (Action)
	SelectedTier       string   `json:"selected_tier"`
	SelectedAgent      string   `json:"selected_agent"`
	AlternativeTiers   []string `json:"alternative_tiers,omitempty"`
	AlternativeAgents  []string `json:"alternative_agents,omitempty"`
	Confidence         float64  `json:"confidence"`

	// Override Information
	WasOverridden  bool   `json:"was_overridden"`
	OverrideReason string `json:"override_reason,omitempty"`

	// Outcome (populated after execution)
	OutcomeSuccess     bool    `json:"outcome_success,omitempty"`
	OutcomeDurationMs  int64   `json:"outcome_duration_ms,omitempty"`
	OutcomeCost        float64 `json:"outcome_cost,omitempty"`
	EscalationRequired bool    `json:"escalation_required,omitempty"`
	RetryCount         int     `json:"retry_count,omitempty"`

	// Understanding Quality (Addendum A.2)
	UnderstandingCompleteness float64 `json:"understanding_completeness,omitempty"`
	UnderstandingAccuracy     float64 `json:"understanding_accuracy,omitempty"`
	SynthesisCoherence        float64 `json:"synthesis_coherence,omitempty"`
	RequiredFollowUp          bool    `json:"required_follow_up,omitempty"`

	// Correlation
	InvocationID string `json:"invocation_id,omitempty"`
}

// DecisionOutcomeUpdate represents an outcome update (append-only)
//
// Read-time Reconciliation:
// ML pipelines should join routing-decisions.jsonl with routing-decision-updates.jsonl
// on DecisionID field. For decisions with multiple updates, take the latest
// update (max UpdateTimestamp) per DecisionID. This append-only design enables
// thread-safe concurrent writes while maintaining complete audit history.
type DecisionOutcomeUpdate struct {
	DecisionID         string  `json:"decision_id"`
	OutcomeSuccess     bool    `json:"outcome_success"`
	OutcomeDurationMs  int64   `json:"outcome_duration_ms"`
	OutcomeCost        float64 `json:"outcome_cost"`
	EscalationRequired bool    `json:"escalation_required"`
	UpdateTimestamp    int64   `json:"update_timestamp"`
}

// NewRoutingDecision creates a new decision record with classification
func NewRoutingDecision(sessionID, taskDesc, selectedTier, selectedAgent string) *RoutingDecision {
	taskType, taskDomain := ClassifyTask(taskDesc)

	return &RoutingDecision{
		DecisionID:      uuid.New().String(),
		Timestamp:       time.Now().Unix(),
		SessionID:       sessionID,
		TaskDescription: truncateDescription(taskDesc, 500),
		TaskType:        taskType,
		TaskDomain:      taskDomain,
		SelectedTier:    selectedTier,
		SelectedAgent:   selectedAgent,
	}
}

// LogRoutingDecision writes decision to JSONL storage
func LogRoutingDecision(decision *RoutingDecision) error {
	path := getRoutingDecisionPath()

	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("[routing-decision] Failed to marshal decision: %w. Ensure all fields are JSON-serializable.", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[routing-decision] Failed to create directory %s: %w. Check file permissions and disk space.", dir, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[routing-decision] Failed to open file %s: %w. Check file permissions and path.", path, err)
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
	if err != nil {
		return fmt.Errorf("[routing-decision] Failed to write decision to file: %w. Disk may be full.", err)
	}

	return nil
}

// UpdateDecisionOutcome appends outcome update (thread-safe, no rewrite)
func UpdateDecisionOutcome(decisionID string, success bool, durationMs int64, cost float64, escalated bool) error {
	update := DecisionOutcomeUpdate{
		DecisionID:         decisionID,
		OutcomeSuccess:     success,
		OutcomeDurationMs:  durationMs,
		OutcomeCost:        cost,
		EscalationRequired: escalated,
		UpdateTimestamp:    time.Now().Unix(),
	}

	// Append to separate updates file (thread-safe)
	return appendDecisionUpdate(update)
}

// appendDecisionUpdate appends an outcome update to the updates file
func appendDecisionUpdate(update DecisionOutcomeUpdate) error {
	path := getRoutingDecisionUpdatesPath()

	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("[routing-decision] Failed to marshal update: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[routing-decision] Failed to create directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[routing-decision] Failed to open updates file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
	return err
}

func getRoutingDecisionPath() string {
	return config.GetRoutingDecisionsPathWithProjectDir()
}

func getRoutingDecisionUpdatesPath() string {
	return config.GetRoutingDecisionUpdatesPathWithProjectDir()
}

func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
