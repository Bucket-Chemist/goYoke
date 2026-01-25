---
id: GOgent-087b
title: Routing Decision Capture
description: Implement RoutingDecision struct and logging for ML training
status: pending
time_estimate: 2.5h
dependencies: ["GOgent-087c"]
priority: high
week: 4
tags: ["ml-optimization", "routing-decision", "week-4"]
tests_required: true
acceptance_criteria_count: 12
---

### GOgent-087b: Routing Decision Capture

**Time**: 2.5 hours
**Dependencies**: GOgent-087c (requires ClassifyTask)

**Task**:
Capture EVERY routing decision for ML training, not just violations.

**File**: `pkg/telemetry/routing_decision.go`

**Imports**:
```go
package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)
```

**Implementation**:
```go
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
type DecisionOutcomeUpdate struct {
	DecisionID        string  `json:"decision_id"`
	OutcomeSuccess    bool    `json:"outcome_success"`
	OutcomeDurationMs int64   `json:"outcome_duration_ms"`
	OutcomeCost       float64 `json:"outcome_cost"`
	EscalationRequired bool   `json:"escalation_required"`
	UpdateTimestamp   int64   `json:"update_timestamp"`
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
	// XDG Base Directory specification
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		home, _ := os.UserHomeDir()
		xdgData = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdgData, "gogent", "routing-decisions.jsonl")
}

func getRoutingDecisionUpdatesPath() string {
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		home, _ := os.UserHomeDir()
		xdgData = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdgData, "gogent", "routing-decision-updates.jsonl")
}

func truncateDescription(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

**Tests**: `pkg/telemetry/routing_decision_test.go`

```go
package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRoutingDecision(t *testing.T) {
	sessionID := "test-session-123"
	taskDesc := "Implement new authentication module"
	tier := "sonnet"
	agent := "python-pro"

	decision := NewRoutingDecision(sessionID, taskDesc, tier, agent)

	if decision.DecisionID == "" {
		t.Error("Expected DecisionID to be set")
	}

	if decision.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, decision.SessionID)
	}

	if decision.TaskDescription != taskDesc {
		t.Errorf("Expected TaskDescription %s, got %s", taskDesc, decision.TaskDescription)
	}

	if decision.SelectedTier != tier {
		t.Errorf("Expected SelectedTier %s, got %s", tier, decision.SelectedTier)
	}

	if decision.SelectedAgent != agent {
		t.Errorf("Expected SelectedAgent %s, got %s", agent, decision.SelectedAgent)
	}

	if decision.Timestamp == 0 {
		t.Error("Expected Timestamp to be set")
	}
}

func TestNewRoutingDecision_TruncatesLongDescription(t *testing.T) {
	longDesc := strings.Repeat("x", 600)
	decision := NewRoutingDecision("session", longDesc, "haiku", "codebase-search")

	if len(decision.TaskDescription) > 503 {
		t.Errorf("Expected description truncated to ~500 chars, got %d", len(decision.TaskDescription))
	}

	if !strings.HasSuffix(decision.TaskDescription, "...") {
		t.Error("Expected truncated description to end with '...'")
	}
}

func TestLogRoutingDecision(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
		Timestamp:     1234567890,
	}

	err := LogRoutingDecision(decision)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created and contains decision
	path := filepath.Join(tmpDir, "gogent", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be created, got error: %v", err)
	}

	if !strings.Contains(string(data), "decision-001") {
		t.Error("Expected decision ID in file")
	}
}

func TestLogRoutingDecision_MultipleDecisions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision1 := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
	}

	decision2 := &RoutingDecision{
		DecisionID:    "decision-002",
		SessionID:     "session-001",
		SelectedTier:  "sonnet",
		SelectedAgent: "python-pro",
	}

	if err := LogRoutingDecision(decision1); err != nil {
		t.Fatalf("Failed to log first decision: %v", err)
	}

	if err := LogRoutingDecision(decision2); err != nil {
		t.Fatalf("Failed to log second decision: %v", err)
	}

	// Verify both are in file
	path := filepath.Join(tmpDir, "gogent", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "decision-001") {
		t.Error("Expected first decision in file")
	}
	if !strings.Contains(content, "decision-002") {
		t.Error("Expected second decision in file")
	}
}

func TestUpdateDecisionOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
	}

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// Update outcome
	err := UpdateDecisionOutcome("decision-001", true, 500, 0.0015, false)
	if err != nil {
		t.Fatalf("Expected no error updating outcome, got: %v", err)
	}

	// Verify outcome was appended to updates file
	updatesPath := filepath.Join(tmpDir, "gogent", "routing-decision-updates.jsonl")
	data, err := os.ReadFile(updatesPath)
	if err != nil {
		t.Fatalf("Failed to read updates file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\"decision_id\":\"decision-001\"") {
		t.Error("Expected decision_id in updates file")
	}
	if !strings.Contains(content, "\"outcome_success\":true") {
		t.Error("Expected outcome_success to be true in update")
	}
}

func TestUpdateDecisionOutcome_MultipleUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID: "decision-001",
		SessionID:  "session-001",
	}

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// First update
	if err := UpdateDecisionOutcome("decision-001", true, 100, 0.001, false); err != nil {
		t.Fatalf("Failed first update: %v", err)
	}

	// Second update
	if err := UpdateDecisionOutcome("decision-001", false, 200, 0.002, true); err != nil {
		t.Fatalf("Failed second update: %v", err)
	}

	// Verify both updates exist in separate lines
	updatesPath := filepath.Join(tmpDir, "gogent", "routing-decision-updates.jsonl")
	data, err := os.ReadFile(updatesPath)
	if err != nil {
		t.Fatalf("Failed to read updates: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 update lines, got %d", len(lines))
	}
}

func TestUnderstandingQualityFields(t *testing.T) {
	decision := &RoutingDecision{
		DecisionID:                "decision-001",
		UnderstandingCompleteness: 0.95,
		UnderstandingAccuracy:     0.87,
		SynthesisCoherence:        0.92,
		RequiredFollowUp:          true,
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	path := filepath.Join(tmpDir, "gogent", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\"understanding_completeness\":0.95") {
		t.Error("Expected understanding_completeness in output")
	}
	if !strings.Contains(content, "\"synthesis_coherence\":0.92") {
		t.Error("Expected synthesis_coherence in output")
	}
	if !strings.Contains(content, "\"required_follow_up\":true") {
		t.Error("Expected required_follow_up in output")
	}
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{strings.Repeat("x", 100), 50, strings.Repeat("x", 50) + "..."},
		{"exactly50chars" + strings.Repeat("x", 35), 50, "exactly50chars" + strings.Repeat("x", 35)},
	}

	for _, tc := range tests {
		result := truncateDescription(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("For input len %d with maxLen %d: expected %q, got %q", len(tc.input), tc.maxLen, tc.expected, result)
		}
	}
}
```

**Acceptance Criteria**:
- [x] `RoutingDecision` struct with all GAP 4.1 fields (DecisionID, Timestamp, SessionID, TaskDescription, TaskType, TaskDomain, SelectedTier, SelectedAgent, Confidence)
- [x] Understanding quality fields included per Addendum A.2 (UnderstandingCompleteness, UnderstandingAccuracy, SynthesisCoherence, RequiredFollowUp)
- [x] `DecisionOutcomeUpdate` struct defined for append-only outcome tracking
- [x] `NewRoutingDecision()` creates decision records with task classification via ClassifyTask()
- [x] `LogRoutingDecision()` appends to XDG-compliant JSONL file at ~/.local/share/gogent/routing-decisions.jsonl
- [x] `UpdateDecisionOutcome()` appends outcomes to separate updates file (not in-place edit)
- [x] Thread-safe append-only pattern implemented (no file rewrite)
- [x] Outcome updates written to separate `routing-decision-updates.jsonl` file
- [x] Read-time reconciliation documented: join decisions with updates on DecisionID
- [x] Error messages follow format: `[routing-decision] What. Why. How.`
- [x] Tests verify logging, outcome update append behavior, truncation, and understanding quality fields
- [x] `go test ./pkg/telemetry` passes with ≥80% coverage
- [x] No race conditions detected: `go test -race ./pkg/telemetry` passes

**Thread Safety Note**: The append-only design eliminates race conditions under concurrent agent execution. Original in-place update design would cause data corruption when multiple hooks fire simultaneously. ML pipelines should reconcile decisions with updates by joining on DecisionID and taking the latest update per decision.

**Why This Matters**: Routing decision capture provides labeled training data for ML routing optimization. By logging EVERY decision (not just violations) along with task classification (features), selected tier/agent (action), and post-execution outcomes (rewards), we create a supervised dataset for training ML models to improve routing accuracy over time. Understanding quality metrics enable measurement of decision quality beyond binary success/failure.
