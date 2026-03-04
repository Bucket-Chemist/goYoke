package integration

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestTaskValidation_CompleteWorkflow(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review", "python-architect", "mozart", "einstein", "beethoven"},
			},
			"sonnet": {
				Model: "claude-3.5-sonnet",
			},
			"haiku": {
				Model: "claude-3-haiku",
			},
		},
		TierLevels: routing.TierLevels{
			Haiku:  10,
			Sonnet: 20,
			Opus:   30,
		},
		AgentSubagentMapping: routing.AgentSubagentMapping{
			PythonPro:      routing.NewFlexibleSubagentType("Python Pro"),
			CodebaseSearch: routing.NewFlexibleSubagentType("Codebase Search"),
			TechDocsWriter: routing.NewFlexibleSubagentType("Tech Docs Writer"),
		},
	}

	t.Run("Valid Task invocation", func(t *testing.T) {
		taskInput := map[string]interface{}{
			"model":         "sonnet",
			"prompt":        "AGENT: python-pro\n\nImplement feature",
			"subagent_type": "Python Pro",
		}

		// Einstein blocking
		einsteinResult := routing.ValidateTaskInvocation(schema, taskInput, "test-session")
		if !einsteinResult.Allowed {
			t.Errorf("Valid invocation blocked: %s", einsteinResult.BlockReason)
		}

		// Subagent type
		subagentResult := routing.ValidateSubagentType(schema, "python-pro", "Python Pro", nil)
		if !subagentResult.Valid {
			t.Errorf("Valid subagent_type rejected: %s", subagentResult.ErrorMessage)
		}

		t.Log("✓ Valid Task invocation passed all checks")
	})

	t.Run("Opus model blocked", func(t *testing.T) {
		taskInput := map[string]interface{}{
			"model":  "opus",
			"prompt": "AGENT: python-pro\n\nComplex task",
		}

		result := routing.ValidateTaskInvocation(schema, taskInput, "test-session")
		if result.Allowed {
			t.Error("Opus model should be blocked")
		}

		if result.Violation.ViolationType != "blocked_task_opus" {
			t.Errorf("Wrong violation type: %s", result.Violation.ViolationType)
		}

		t.Log("✓ Opus model correctly blocked")
	})

	t.Run("Einstein requires opus model", func(t *testing.T) {
		// Einstein is allowlisted but must be invoked with model: opus
		// ValidateTaskInvocation enforces this: allowlisted agent + non-opus model = blocked
		taskInput := map[string]interface{}{
			"model":  "sonnet",
			"prompt": "AGENT: einstein\n\nDeep analysis",
		}

		result := routing.ValidateTaskInvocation(schema, taskInput, "test-session")
		if result.Allowed {
			t.Error("Einstein with sonnet model should be blocked (requires opus)")
		}

		if result.Violation == nil {
			t.Fatal("Expected violation for einstein with wrong model")
		}

		if result.Violation.ViolationType != "opus_agent_wrong_model" {
			t.Errorf("Wrong violation type: expected opus_agent_wrong_model, got %s", result.Violation.ViolationType)
		}

		// Verify einstein with opus is allowed
		taskInputOpus := map[string]interface{}{
			"model":  "opus",
			"prompt": "AGENT: einstein\n\nDeep analysis",
		}

		resultOpus := routing.ValidateTaskInvocation(schema, taskInputOpus, "test-session")
		if !resultOpus.Allowed {
			t.Errorf("Einstein with opus should be allowed, got blocked: %s", resultOpus.BlockReason)
		}

		t.Log("✓ Einstein correctly requires opus model")
	})

	t.Run("Wrong subagent_type", func(t *testing.T) {
		// codebase-search requires "Codebase Search", using "Python Pro" instead
		result := routing.ValidateSubagentType(schema, "codebase-search", "Python Pro", nil)

		if result.Valid {
			t.Error("Wrong subagent_type should be rejected")
		}

		if len(result.AllowedTypes) == 0 || result.AllowedTypes[0] != "Codebase Search" {
			t.Errorf("Expected allowed types to include 'Codebase Search', got: %v", result.AllowedTypes)
		}

		formatted := result.FormatSubagentTypeError()
		if !contains(formatted, "Fix:") {
			t.Error("Error should include fix suggestion")
		}

		t.Log("✓ Subagent_type mismatch correctly detected")
	})
}

func TestTaskValidation_RealWorldScenarios(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
		AgentSubagentMapping: routing.AgentSubagentMapping{
			PythonPro:      routing.NewFlexibleSubagentType("Python Pro"),
			PythonUX:       routing.NewFlexibleSubagentType("Python UX (PySide6)"),
			RPro:           routing.NewFlexibleSubagentType("R Pro"),
			RShinyPro:      routing.NewFlexibleSubagentType("R Shiny Pro"),
			CodebaseSearch: routing.NewFlexibleSubagentType("Codebase Search"),
			Scaffolder:     routing.NewFlexibleSubagentType("Scaffolder"),
			TechDocsWriter: routing.NewFlexibleSubagentType("Tech Docs Writer"),
			Librarian:      routing.NewFlexibleSubagentType("Librarian"),
			CodeReviewer:   routing.NewFlexibleSubagentType("Code Reviewer"),
			Orchestrator:   routing.NewFlexibleSubagentType("Orchestrator"),
			Architect:      routing.NewFlexibleSubagentType("Architect"),
		},
	}

	tests := []struct {
		name          string
		agent         string
		subagentType  string
		shouldBeValid bool
	}{
		{"Python implementation (correct)", "python-pro", "Python Pro", true},
		{"Python implementation (wrong)", "python-pro", "Codebase Search", false},
		{"Codebase search (correct)", "codebase-search", "Codebase Search", true},
		{"Codebase search (wrong)", "codebase-search", "Python Pro", false},
		{"Tech docs writer (correct)", "tech-docs-writer", "Tech Docs Writer", true},
		{"Tech docs writer (wrong)", "tech-docs-writer", "Codebase Search", false},
		{"Orchestrator (correct)", "orchestrator", "Orchestrator", true},
		{"Orchestrator (wrong)", "orchestrator", "Python Pro", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := routing.ValidateSubagentType(schema, tt.agent, tt.subagentType, nil)

			if result.Valid != tt.shouldBeValid {
				t.Errorf("Agent %s with type %s: expected valid=%v, got %v (error: %s)",
					tt.agent, tt.subagentType, tt.shouldBeValid, result.Valid, result.ErrorMessage)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
