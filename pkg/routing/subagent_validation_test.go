package routing

import (
	"testing"
)

func TestValidateSubagentType_Correct(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro:      NewFlexibleSubagentType("Python Pro"),
			CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
			Orchestrator:   NewFlexibleSubagentType("Orchestrator"),
		},
	}

	tests := []struct {
		agent        string
		subagentType string
	}{
		{"python-pro", "Python Pro"},
		{"codebase-search", "Codebase Search"},
		{"orchestrator", "Orchestrator"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			result := ValidateSubagentType(schema, tt.agent, tt.subagentType, nil)

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
			PythonPro:      NewFlexibleSubagentType("Python Pro"),
			CodebaseSearch: NewFlexibleSubagentType("Codebase Search"),
		},
	}

	// Wrong type for python-pro
	result := ValidateSubagentType(schema, "python-pro", "Codebase Search", nil)

	if result.Valid {
		t.Error("Expected invalid result for wrong subagent_type")
	}

	if len(result.AllowedTypes) != 1 || result.AllowedTypes[0] != "Python Pro" {
		t.Errorf("Expected allowed types ['Python Pro'], got: %v", result.AllowedTypes)
	}

	if result.ErrorMessage == "" {
		t.Error("Expected error message")
	}

	// Check error contains key info
	if !contains(result.ErrorMessage, "python-pro") {
		t.Error("Error should mention agent name")
	}

	if !contains(result.ErrorMessage, "Python Pro") {
		t.Error("Error should mention required type")
	}

	if !contains(result.ErrorMessage, "Codebase Search") {
		t.Error("Error should mention requested type")
	}
}

func TestValidateSubagentType_NoAgent(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: NewFlexibleSubagentType("Python Pro"),
		},
	}

	// No agent specified
	result := ValidateSubagentType(schema, "", "Codebase Search", nil)

	if !result.Valid {
		t.Error("Expected valid when no agent specified")
	}
}

func TestValidateSubagentType_AgentNotInMapping(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: NewFlexibleSubagentType("Python Pro"),
		},
	}

	// Custom agent not in mapping
	result := ValidateSubagentType(schema, "custom-agent", "Python Pro", nil)

	if !result.Valid {
		t.Error("Expected valid for unmapped agent (might be custom)")
	}
}

func TestValidateSubagentType_NoMapping(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{},
	}

	result := ValidateSubagentType(schema, "python-pro", "Codebase Search", nil)

	if !result.Valid {
		t.Error("Expected valid when no mapping defined")
	}
}

func TestFormatSubagentTypeError(t *testing.T) {
	result := &SubagentTypeValidation{
		Valid:         false,
		Agent:         "tech-docs-writer",
		RequestedType: "Codebase Search",
		AllowedTypes:  []string{"Tech Docs Writer"},
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

	if !contains(formatted, "Tech Docs Writer") {
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
		RequestedType: "Python Pro",
		AllowedTypes:  []string{"Python Pro"},
		ErrorMessage:  "",
	}

	formatted := result.FormatSubagentTypeError()

	if formatted != "" {
		t.Errorf("Expected empty string for valid validation, got: %s", formatted)
	}
}

func TestValidateSubagentType_MultiType_FirstType(t *testing.T) {
	// Multi-type is now legacy — each agent has a single CC type name.
	// But the mechanism still works if someone configures multiple types.
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review"),
		},
	}

	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Staff Architect Critical Review", nil)

	if !result.Valid {
		t.Errorf("Expected valid for CC type name, got error: %s", result.ErrorMessage)
	}

	if len(result.AllowedTypes) != 1 {
		t.Errorf("Expected 1 allowed type, got: %d", len(result.AllowedTypes))
	}
}

func TestValidateSubagentType_MultiType_SecondType(t *testing.T) {
	// Test that multi-type still works for backward compatibility
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review", "Explore"),
		},
	}

	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Explore", nil)

	if !result.Valid {
		t.Errorf("Expected valid for second type in multi-type agent, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_MultiType_InvalidType(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review", "Explore"),
		},
	}

	// Type not in allowed list should be invalid
	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Python Pro", nil)

	if result.Valid {
		t.Error("Expected invalid result for type not in multi-type agent's allowed list")
	}

	if len(result.AllowedTypes) != 2 {
		t.Errorf("Expected 2 allowed types in error, got: %d", len(result.AllowedTypes))
	}

	if !contains(result.ErrorMessage, "Staff Architect Critical Review") || !contains(result.ErrorMessage, "Explore") {
		t.Error("Error should mention both allowed types")
	}

	if !contains(result.ErrorMessage, "Python Pro") {
		t.Error("Error should mention requested type")
	}
}

func TestValidateSubagentType_MultiType_ErrorFormat(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review", "Explore"),
		},
	}

	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Bash", nil)

	formatted := result.FormatSubagentTypeError()

	if formatted == "" {
		t.Error("Expected non-empty formatted error for multi-type agent")
	}

	// Should suggest the first type (primary)
	if !contains(formatted, "Staff Architect Critical Review") {
		t.Error("Formatted error should suggest primary type (Staff Architect Critical Review)")
	}

	// Should show the fix suggestion
	if !contains(formatted, "Fix:") {
		t.Error("Formatted error should include fix suggestion")
	}
}

func TestValidateSubagentType_WithAgentTaskNames_DirectMatch(t *testing.T) {
	// With the new CC type names, the schema mapping directly matches CC names.
	// The agentTaskNames fallback is now redundant but should still work
	// for any edge cases where mapping and CC name differ.
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			Einstein: NewFlexibleSubagentType("Einstein"),
		},
	}

	// Direct match — no fallback needed
	result := ValidateSubagentType(schema, "einstein", "Einstein", nil)

	if !result.Valid {
		t.Errorf("Expected valid for direct CC type name 'Einstein', got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_WithAgentTaskNames_FallbackStillWorks(t *testing.T) {
	// Test that the agentTaskNames fallback still works if mapping has
	// a different value than the CC type name (backward compat)
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			Einstein: NewFlexibleSubagentType("OldCategory"),
		},
	}

	agentTaskNames := map[string]string{
		"einstein": "Einstein",
	}

	// Request with CC type name, schema has old category — fallback should accept
	result := ValidateSubagentType(schema, "einstein", "Einstein", agentTaskNames)

	if !result.Valid {
		t.Errorf("Expected valid via agentTaskNames fallback, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_WithAgentTaskNames_RejectsRandomName(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			Einstein: NewFlexibleSubagentType("Einstein"),
		},
	}

	agentTaskNames := map[string]string{
		"einstein": "Einstein",
	}

	// Request with unrelated name
	result := ValidateSubagentType(schema, "einstein", "RandomName", agentTaskNames)

	if result.Valid {
		t.Error("Expected invalid result for random name not in mapping or task names")
	}

	// Error should mention the CC type name
	if !contains(result.ErrorMessage, "Einstein") {
		t.Error("Error should mention CC type name")
	}
}

func TestValidateSubagentType_StaffArchitectDirectMatch(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			StaffArchitectCriticalReview: NewFlexibleSubagentType("Staff Architect Critical Review"),
		},
	}

	// Direct match with CC type name
	result := ValidateSubagentType(schema, "staff-architect-critical-review", "Staff Architect Critical Review", nil)

	if !result.Valid {
		t.Errorf("Expected valid for CC type name, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_BeethovenDirectMatch(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			Beethoven: NewFlexibleSubagentType("Beethoven"),
			Einstein:  NewFlexibleSubagentType("Einstein"),
		},
	}

	// Beethoven with its CC type name should work directly
	result := ValidateSubagentType(schema, "beethoven", "Beethoven", nil)

	if !result.Valid {
		t.Errorf("Expected valid for Beethoven CC type name, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_MozartDirectMatch(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			Mozart: NewFlexibleSubagentType("Mozart"),
		},
	}

	result := ValidateSubagentType(schema, "mozart", "Mozart", nil)

	if !result.Valid {
		t.Errorf("Expected valid for Mozart CC type name, got error: %s", result.ErrorMessage)
	}
}

func TestValidateSubagentType_PythonArchitectDirectMatch(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonArchitect: NewFlexibleSubagentType("Python ML Architect"),
		},
	}

	result := ValidateSubagentType(schema, "python-architect", "Python ML Architect", nil)

	if !result.Valid {
		t.Errorf("Expected valid for Python ML Architect CC type name, got error: %s", result.ErrorMessage)
	}
}
