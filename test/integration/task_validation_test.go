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
				TaskInvocationBlocked: true,
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
			PythonPro:      "general-purpose",
			CodebaseSearch: "Explore",
			TechDocsWriter: "general-purpose",
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
		subagentResult := routing.ValidateSubagentType(schema, "python-pro", "general-purpose")
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

	t.Run("Einstein agent blocked", func(t *testing.T) {
		taskInput := map[string]interface{}{
			"model":  "sonnet",
			"prompt": "AGENT: einstein\n\nDeep analysis",
		}

		result := routing.ValidateTaskInvocation(schema, taskInput, "test-session")
		if result.Allowed {
			t.Error("Einstein agent should be blocked")
		}

		if !contains(result.Recommendation, "GAP document") {
			t.Error("Should recommend GAP document workflow")
		}

		t.Log("✓ Einstein agent correctly blocked")
	})

	t.Run("Wrong subagent_type", func(t *testing.T) {
		// codebase-search requires "Explore", using "general-purpose" instead
		result := routing.ValidateSubagentType(schema, "codebase-search", "general-purpose")

		if result.Valid {
			t.Error("Wrong subagent_type should be rejected")
		}

		if result.RequiredType != "Explore" {
			t.Errorf("Expected required type 'Explore', got: %s", result.RequiredType)
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
			PythonPro:      "general-purpose",
			PythonUX:       "general-purpose",
			RPro:           "general-purpose",
			RShinyPro:      "general-purpose",
			CodebaseSearch: "Explore",
			Scaffolder:     "general-purpose",
			TechDocsWriter: "general-purpose",
			Librarian:      "Explore",
			CodeReviewer:   "Explore",
			Orchestrator:   "Plan",
			Architect:      "Plan",
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
			result := routing.ValidateSubagentType(schema, tt.agent, tt.subagentType)

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
