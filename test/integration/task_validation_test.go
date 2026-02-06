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
			PythonPro:      routing.NewFlexibleSubagentType("general-purpose"),
			CodebaseSearch: routing.NewFlexibleSubagentType("Explore"),
			TechDocsWriter: routing.NewFlexibleSubagentType("general-purpose"),
		},
	}

	t.Run("Valid Task invocation", func(t *testing.T) {
		taskInput := map[string]interface{}{
			"model":         "sonnet",
			"prompt":        "AGENT: python-pro\n\nImplement feature",
			"subagent_type": "general-purpose",
		}

		// Einstein blocking
		einsteinResult := routing.ValidateTaskInvocation(schema, taskInput, "test-session")
		if !einsteinResult.Allowed {
			t.Errorf("Valid invocation blocked: %s", einsteinResult.BlockReason)
		}

		// Subagent type
		subagentResult := routing.ValidateSubagentType(schema, "python-pro", "general-purpose", nil)
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
		// codebase-search requires "Explore", using "general-purpose" instead
		result := routing.ValidateSubagentType(schema, "codebase-search", "general-purpose", nil)

		if result.Valid {
			t.Error("Wrong subagent_type should be rejected")
		}

		if len(result.AllowedTypes) == 0 || result.AllowedTypes[0] != "Explore" {
			t.Errorf("Expected allowed types to include 'Explore', got: %v", result.AllowedTypes)
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
			PythonPro:      routing.NewFlexibleSubagentType("general-purpose"),
			PythonUX:       routing.NewFlexibleSubagentType("general-purpose"),
			RPro:           routing.NewFlexibleSubagentType("general-purpose"),
			RShinyPro:      routing.NewFlexibleSubagentType("general-purpose"),
			CodebaseSearch: routing.NewFlexibleSubagentType("Explore"),
			Scaffolder:     routing.NewFlexibleSubagentType("general-purpose"),
			TechDocsWriter: routing.NewFlexibleSubagentType("general-purpose"),
			Librarian:      routing.NewFlexibleSubagentType("Explore"),
			CodeReviewer:   routing.NewFlexibleSubagentType("Explore"),
			Orchestrator:   routing.NewFlexibleSubagentType("Plan"),
			Architect:      routing.NewFlexibleSubagentType("Plan"),
		},
	}

	tests := []struct {
		name          string
		agent         string
		subagentType  string
		shouldBeValid bool
	}{
		{"Python implementation (correct)", "python-pro", "general-purpose", true},
		{"Python implementation (wrong)", "python-pro", "Explore", false},
		{"Codebase search (correct)", "codebase-search", "Explore", true},
		{"Codebase search (wrong)", "codebase-search", "general-purpose", false},
		{"Tech docs writer (correct)", "tech-docs-writer", "general-purpose", true},
		{"Tech docs writer (wrong)", "tech-docs-writer", "Explore", false},
		{"Orchestrator (correct)", "orchestrator", "Plan", true},
		{"Orchestrator (wrong)", "orchestrator", "general-purpose", false},
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
