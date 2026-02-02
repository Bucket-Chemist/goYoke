package routing

import (
	"testing"
)

func TestValidateSubagentType_Correct(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro:      NewFlexibleSubagentType("general-purpose"),
			CodebaseSearch: NewFlexibleSubagentType("Explore"),
			Orchestrator:   NewFlexibleSubagentType("Plan"),
		},
	}

	tests := []struct {
		agent        string
		subagentType string
	}{
		{"python-pro", "general-purpose"},
		{"codebase-search", "Explore"},
		{"orchestrator", "Plan"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			result := ValidateSubagentType(schema, tt.agent, tt.subagentType)

			if !result.Valid {
				t.Errorf("Expected valid for %s with %s, got error: %s",
					tt.agent, tt.subagentType, result.ErrorMessage)
			}
		})
	}
}

func TestValidateSubagentType_Incorrect(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro:      NewFlexibleSubagentType("general-purpose"),
			CodebaseSearch: NewFlexibleSubagentType("Explore"),
		},
	}

	// Wrong type for python-pro
	result := ValidateSubagentType(schema, "python-pro", "Explore")

	if result.Valid {
		t.Error("Expected invalid result for wrong subagent_type")
	}

	if len(result.AllowedTypes) != 1 || result.AllowedTypes[0] != "general-purpose" {
		t.Errorf("Expected allowed types ['general-purpose'], got: %v", result.AllowedTypes)
	}

	if result.ErrorMessage == "" {
		t.Error("Expected error message")
	}

	// Check error contains key info
	if !contains(result.ErrorMessage, "python-pro") {
		t.Error("Error should mention agent name")
	}

	if !contains(result.ErrorMessage, "general-purpose") {
		t.Error("Error should mention required type")
	}

	if !contains(result.ErrorMessage, "Explore") {
		t.Error("Error should mention requested type")
	}
}

func TestValidateSubagentType_NoAgent(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: NewFlexibleSubagentType("general-purpose"),
		},
	}

	// No agent specified
	result := ValidateSubagentType(schema, "", "Explore")

	if !result.Valid {
		t.Error("Expected valid when no agent specified")
	}
}

func TestValidateSubagentType_AgentNotInMapping(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: NewFlexibleSubagentType("general-purpose"),
		},
	}

	// Custom agent not in mapping
	result := ValidateSubagentType(schema, "custom-agent", "general-purpose")

	if !result.Valid {
		t.Error("Expected valid for unmapped agent (might be custom)")
	}
}

func TestValidateSubagentType_NoMapping(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{},
	}

	result := ValidateSubagentType(schema, "python-pro", "Explore")

	if !result.Valid {
		t.Error("Expected valid when no mapping defined")
	}
}

func TestFormatSubagentTypeError(t *testing.T) {
	result := &SubagentTypeValidation{
		Valid:         false,
		Agent:         "tech-docs-writer",
		RequestedType: "Explore",
		AllowedTypes:  []string{"general-purpose"},
		ErrorMessage:  "[task-validation] Invalid subagent_type",
	}

	formatted := result.FormatSubagentTypeError()

	if formatted == "" {
		t.Error("Expected non-empty formatted error")
	}

	// Check for fix suggestion
	if !contains(formatted, "Fix:") {
		t.Error("Formatted error should include fix suggestion")
	}

	if !contains(formatted, "general-purpose") {
		t.Error("Fix should show correct subagent_type")
	}

	if !contains(formatted, "tech-docs-writer") {
		t.Error("Fix should reference the agent")
	}
}

func TestFormatSubagentTypeError_Valid(t *testing.T) {
	result := &SubagentTypeValidation{
		Valid:         true,
		Agent:         "python-pro",
		RequestedType: "general-purpose",
		AllowedTypes:  []string{"general-purpose"},
		ErrorMessage:  "",
	}

	formatted := result.FormatSubagentTypeError()

	if formatted != "" {
		t.Errorf("Expected empty string for valid validation, got: %s", formatted)
	}
}

func TestValidateSubagentType_MultiType_FirstType(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Plan", "Explore"),
		},
	}

	// First type should be valid
	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Plan")

	if !result.Valid {
		t.Errorf("Expected valid for first type in multi-type agent, got error: %s", result.ErrorMessage)
	}

	if len(result.AllowedTypes) != 2 {
		t.Errorf("Expected 2 allowed types, got: %d", len(result.AllowedTypes))
	}
}

func TestValidateSubagentType_MultiType_SecondType(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Plan", "Explore"),
		},
	}

	// Second type should also be valid
	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Explore")

	if !result.Valid {
		t.Errorf("Expected valid for second type in multi-type agent, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_MultiType_InvalidType(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Plan", "Explore"),
		},
	}

	// Type not in allowed list should be invalid
	result := ValidateSubagentType(schema, "staff-architect-critical-review", "general-purpose")

	if result.Valid {
		t.Error("Expected invalid result for type not in multi-type agent's allowed list")
	}

	if len(result.AllowedTypes) != 2 {
		t.Errorf("Expected 2 allowed types in error, got: %d", len(result.AllowedTypes))
	}

	if !contains(result.ErrorMessage, "Plan") || !contains(result.ErrorMessage, "Explore") {
		t.Error("Error should mention both allowed types")
	}

	if !contains(result.ErrorMessage, "general-purpose") {
		t.Error("Error should mention requested type")
	}
}

func TestValidateSubagentType_MultiType_ErrorFormat(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Plan", "Explore"),
		},
	}

	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Bash")

	formatted := result.FormatSubagentTypeError()

	if formatted == "" {
		t.Error("Expected non-empty formatted error for multi-type agent")
	}

	// Should suggest the first type (primary)
	if !contains(formatted, "Plan") {
		t.Error("Formatted error should suggest primary type (Plan)")
	}

	// Should show the fix suggestion
	if !contains(formatted, "Fix:") {
		t.Error("Formatted error should include fix suggestion")
	}
}
