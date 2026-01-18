package routing

import (
	"strings"
	"testing"
)

func TestValidateTaskInvocation_OpusModelBlocked(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nImplement feature",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected Task(model: opus) to be blocked")
	}

	if result.Violation.ViolationType != "blocked_task_opus" {
		t.Errorf("Expected violation type blocked_task_opus, got: %s", result.Violation.ViolationType)
	}

	if !strings.Contains(result.BlockReason, "60K token") {
		t.Error("Block reason should mention token inheritance")
	}
}

func TestValidateTaskInvocation_EinsteinAgentBlocked(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: einstein\n\nAnalyze this problem",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected einstein agent to be blocked")
	}

	if result.Violation.Agent != "einstein" {
		t.Errorf("Expected agent einstein, got: %s", result.Violation.Agent)
	}

	if !strings.Contains(result.Recommendation, "GAP document") {
		t.Error("Recommendation should mention GAP document")
	}
}

func TestValidateTaskInvocation_AllowedAgent(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: python-pro\n\nImplement feature",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected Task(sonnet, python-pro) to be allowed, got blocked: %s", result.BlockReason)
	}

	if result.Violation != nil {
		t.Error("Expected no violation for allowed invocation")
	}
}

func TestValidateTaskInvocation_BlockingDisabled(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: false,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: einstein\n\nDeep analysis",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Error("Expected invocation to be allowed when blocking disabled")
	}
}

func TestExtractAgentFromPrompt(t *testing.T) {
	tests := []struct {
		prompt   string
		expected string
	}{
		{"AGENT: python-pro\n\nImplement X", "python-pro"},
		{"AGENT:einstein\n\nAnalyze Y", "einstein"},
		{"AGENT:  codebase-search  \n\nFind files", "codebase-search"},
		{"No agent specified", ""},
		{"agent: lowercase-not-matched", ""},
	}

	for _, tt := range tests {
		t.Run(tt.prompt[:min(20, len(tt.prompt))], func(t *testing.T) {
			agent := extractAgentFromPrompt(tt.prompt)
			if agent != tt.expected {
				t.Errorf("Expected agent '%s', got '%s'", tt.expected, agent)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestValidateModelMatch_ExactMatch(t *testing.T) {
	agentConfig := &AgentConfig{
		Model: "sonnet",
	}

	matches, warning := ValidateModelMatch("test-agent", agentConfig, "sonnet")

	if !matches {
		t.Error("Expected exact model match")
	}

	if warning != "" {
		t.Errorf("Expected no warning, got: %s", warning)
	}
}

func TestValidateModelMatch_Mismatch(t *testing.T) {
	agentConfig := &AgentConfig{
		Model: "sonnet",
	}

	matches, warning := ValidateModelMatch("test-agent", agentConfig, "haiku")

	if matches {
		t.Error("Expected model mismatch detection")
	}

	if warning == "" {
		t.Error("Expected warning for model mismatch")
	}

	if !strings.Contains(warning, "sonnet") || !strings.Contains(warning, "haiku") {
		t.Errorf("Warning should mention both expected and requested models: %s", warning)
	}
}

func TestValidateModelMatch_AllowedModels(t *testing.T) {
	agentConfig := &AgentConfig{
		Model:         "sonnet",
		AllowedModels: []string{"sonnet", "haiku"},
	}

	tests := []struct {
		model    string
		expected bool
	}{
		{"sonnet", true},
		{"haiku", true},
		{"opus", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			matches, _ := ValidateModelMatch("test-agent", agentConfig, tt.model)
			if matches != tt.expected {
				t.Errorf("Model %s: expected match=%v, got %v", tt.model, tt.expected, matches)
			}
		})
	}
}
