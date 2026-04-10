package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationOrchestrator_AllowedTask(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: true},
		},
		TierLevels: TierLevels{
			Haiku: 10, Sonnet: 20,
		},
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: NewFlexibleSubagentType("Python Pro"),
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	taskInput := map[string]interface{}{
		"model":         "sonnet",
		"prompt":        "AGENT: python-pro\n\nImplement feature",
		"subagent_type": "Python Pro",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "allow" {
		t.Errorf("Expected allow, got: %s (reason: %s)", result.Decision, result.Reason)
	}

	if len(result.Violations) > 0 {
		t.Errorf("Expected no violations, got: %d", len(result.Violations))
	}
}

func TestValidationOrchestrator_OpusBlocked(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: true},
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nComplex task",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Opus should be blocked")
	}

	if result.EinsteinBlocked == nil {
		t.Error("Expected einstein blocked result")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violation logged")
	}
}

func TestValidationOrchestrator_CeilingViolation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create delegation ceiling file
	ceilingDir := filepath.Join(tmpDir, ".gogent", "tmp")
	os.MkdirAll(ceilingDir, 0755)
	os.WriteFile(filepath.Join(ceilingDir, "max_delegation"), []byte("haiku"), 0644)

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: false}, // Allow opus at schema level
		},
		TierLevels: TierLevels{
			Haiku: 10, Sonnet: 20,
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: python-pro\n\nTask",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Ceiling violation should block")
	}

	if result.CeilingViolation == "" {
		t.Error("Expected ceiling violation message")
	}
}

func TestValidationOrchestrator_SubagentTypeMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: false},
		},
		AgentSubagentMapping: AgentSubagentMapping{
			CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	taskInput := map[string]interface{}{
		"model":         "sonnet",
		"prompt":        "AGENT: codebase-search\n\nFind files",
		"subagent_type": "Python Pro", // Wrong!
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Subagent_type mismatch should block")
	}

	if result.SubagentTypeInvalid == nil {
		t.Error("Expected subagent type validation result")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violation logged")
	}
}

func TestValidationOrchestrator_OpusAllowlistBypassesCeiling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create delegation ceiling file set to haiku (very restrictive)
	ceilingDir := filepath.Join(tmpDir, ".gogent", "tmp")
	os.MkdirAll(ceilingDir, 0755)
	os.WriteFile(filepath.Join(ceilingDir, "max_delegation"), []byte("haiku"), 0644)

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
		TierLevels: TierLevels{
			Haiku: 1, HaikuThinking: 2, Sonnet: 3, Opus: 4,
		},
		AgentSubagentMapping: AgentSubagentMapping{
			Planner: NewFlexibleSubagentType("Planner"),
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	taskInput := map[string]interface{}{
		"model":         "opus",
		"prompt":        "AGENT: planner\n\nCreate strategic plan",
		"subagent_type": "Planner",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "allow" {
		t.Errorf("Allowlisted opus agent should bypass delegation ceiling. Got: %s, Reason: %s", result.Decision, result.Reason)
	}

	if result.CeilingViolation != "" {
		t.Errorf("Should not have ceiling violation for allowlisted agent: %s", result.CeilingViolation)
	}
}

func TestValidationOrchestrator_NonAllowlistedOpusStillBlockedByCeiling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create delegation ceiling file set to sonnet
	ceilingDir := filepath.Join(tmpDir, ".gogent", "tmp")
	os.MkdirAll(ceilingDir, 0755)
	os.WriteFile(filepath.Join(ceilingDir, "max_delegation"), []byte("sonnet"), 0644)

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect"}, // python-pro NOT in list
			},
		},
		TierLevels: TierLevels{
			Haiku: 1, HaikuThinking: 2, Sonnet: 3, Opus: 4,
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil, nil)

	// Try opus with non-allowlisted agent - should be blocked by opus check, not ceiling
	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nImplement feature",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Non-allowlisted opus agent should still be blocked")
	}

	// Should be blocked by opus check (EinsteinBlocked), not ceiling
	if result.EinsteinBlocked == nil {
		t.Error("Should be blocked by opus/allowlist check, not ceiling")
	}
}

func TestValidationResult_ToJSON(t *testing.T) {
	result := &ValidationResult{
		Decision: "block",
		Reason:   "Test block reason",
		Violations: []*Violation{
			{
				ViolationType: "test_violation",
				Reason:        "Test reason",
			},
		},
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	if parsed["decision"] != "block" {
		t.Error("JSON should contain decision field")
	}
}
