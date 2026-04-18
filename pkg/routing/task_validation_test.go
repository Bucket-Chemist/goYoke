package routing

import (
	"os"
	"strings"
	"testing"
)

func TestValidateTaskInvocation_OpusModelBlocked_NotInAllowlist(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nImplement feature",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected Task(model: opus) to be blocked for non-allowlisted agent")
	}

	if result.Violation.ViolationType != "blocked_task_opus" {
		t.Errorf("Expected violation type blocked_task_opus, got: %s", result.Violation.ViolationType)
	}

	if !strings.Contains(result.BlockReason, "not in allowlist") {
		t.Errorf("Block reason should mention allowlist, got: %s", result.BlockReason)
	}
}

func TestValidateTaskInvocation_OpusAllowlist_Planner(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: planner\n\nCreate strategic plan",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected Task(opus, planner) to be allowed via allowlist, got blocked: %s", result.BlockReason)
	}

	if result.Violation != nil {
		t.Error("Expected no violation for allowlisted agent")
	}
}

func TestValidateTaskInvocation_OpusAllowlist_Architect(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: architect\n\nCreate implementation plan",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected Task(opus, architect) to be allowed via allowlist, got blocked: %s", result.BlockReason)
	}
}

func TestValidateTaskInvocation_OpusAllowlist_StaffArchitect(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: staff-architect-critical-review\n\nReview this plan",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected Task(opus, staff-architect-critical-review) to be allowed via allowlist, got blocked: %s", result.BlockReason)
	}
}

func TestValidateTaskInvocation_EinsteinAllowed_WhenInAllowlistWithOpus(t *testing.T) {
	// Einstein should be allowed when in allowlist AND model is opus
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "einstein"}, // einstein in list
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: einstein\n\nDeep analysis",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected einstein with opus model to be allowed via allowlist, got blocked: %s", result.BlockReason)
	}

	if result.Violation != nil {
		t.Error("Expected no violation for allowlisted einstein with opus model")
	}
}

func TestIsInAllowlist(t *testing.T) {
	allowlist := []string{"planner", "architect", "staff-architect-critical-review"}

	tests := []struct {
		agent    string
		expected bool
	}{
		{"planner", true},
		{"architect", true},
		{"staff-architect-critical-review", true},
		{"python-pro", false},
		{"einstein", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			result := isInAllowlist(tt.agent, allowlist)
			if result != tt.expected {
				t.Errorf("isInAllowlist(%q) = %v, want %v", tt.agent, result, tt.expected)
			}
		})
	}
}

func TestIsInAllowlist_EmptyList(t *testing.T) {
	if isInAllowlist("planner", nil) {
		t.Error("Expected false for nil allowlist")
	}

	if isInAllowlist("planner", []string{}) {
		t.Error("Expected false for empty allowlist")
	}
}

func TestValidateTaskInvocation_EinsteinRequiresOpus(t *testing.T) {
	// Einstein is in allowlist but requires opus model - sonnet should be blocked
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "einstein"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: einstein\n\nAnalyze this problem",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected einstein with sonnet model to be blocked - requires opus")
	}

	if result.Violation == nil {
		t.Fatal("Expected violation to be set")
	}

	if result.Violation.Agent != "einstein" {
		t.Errorf("Expected agent einstein, got: %s", result.Violation.Agent)
	}

	if result.Violation.ViolationType != "opus_agent_wrong_model" {
		t.Errorf("Expected opus_agent_wrong_model violation, got: %s", result.Violation.ViolationType)
	}

	if !strings.Contains(result.BlockReason, "requires model: opus") {
		t.Errorf("Block reason should mention opus requirement, got: %s", result.BlockReason)
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

func TestValidateTaskInvocation_OpusAgentWrongModel_StaffArchitect(t *testing.T) {
	// staff-architect-critical-review is in allowlist but invoked with sonnet
	// Should be BLOCKED - opus agents must run at opus
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet", // Wrong model for opus-tier agent
		"prompt": "AGENT: staff-architect-critical-review\n\nReview this plan",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected staff-architect with model:sonnet to be blocked - opus agents require opus")
	}

	if result.Violation == nil {
		t.Fatal("Expected violation to be set")
	}

	if result.Violation.ViolationType != "opus_agent_wrong_model" {
		t.Errorf("Expected violation type opus_agent_wrong_model, got: %s", result.Violation.ViolationType)
	}

	if !strings.Contains(result.BlockReason, "requires model: opus") {
		t.Errorf("Block reason should mention opus requirement, got: %s", result.BlockReason)
	}

	if !strings.Contains(result.Recommendation, "model: \"opus\"") {
		t.Errorf("Recommendation should include fix example, got: %s", result.Recommendation)
	}
}

func TestValidateTaskInvocation_OpusAgentWrongModel_Planner(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "haiku", // Wrong model
		"prompt": "AGENT: planner\n\nCreate strategy",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected planner with model:haiku to be blocked")
	}

	if result.Violation.ViolationType != "opus_agent_wrong_model" {
		t.Errorf("Expected opus_agent_wrong_model, got: %s", result.Violation.ViolationType)
	}
}

func TestValidateTaskInvocation_OpusAgentWrongModel_NoModel(t *testing.T) {
	// No model specified (empty string) should also be blocked
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "staff-architect-critical-review"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"prompt": "AGENT: architect\n\nCreate plan",
		// model not specified
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected architect with no model to be blocked - opus agents require explicit opus")
	}

	if result.Violation.ViolationType != "opus_agent_wrong_model" {
		t.Errorf("Expected opus_agent_wrong_model, got: %s", result.Violation.ViolationType)
	}
}

func TestGetNestingLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "missing env var returns 0 (root when CLAUDE_CODE_NESTING_LEVEL also unset)",
			envValue: "",
			expected: 0,
		},
		{
			name:     "level 0 returns 0",
			envValue: "0",
			expected: 0,
		},
		{
			name:     "level 1 returns 1",
			envValue: "1",
			expected: 1,
		},
		{
			name:     "level 5 returns 5",
			envValue: "5",
			expected: 5,
		},
		{
			name:     "invalid string returns default (fail-closed)",
			envValue: "abc",
			expected: DefaultNestingLevel,
		},
		{
			name:     "negative returns default (fail-closed)",
			envValue: "-1",
			expected: DefaultNestingLevel,
		},
		{
			name:     "exceeds max returns default (fail-closed)",
			envValue: "100",
			expected: DefaultNestingLevel,
		},
		{
			name:     "max valid level returns correctly",
			envValue: "10",
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or clear env var
			if tt.envValue == "" {
				os.Unsetenv("GOYOKE_NESTING_LEVEL")
			} else {
				os.Setenv("GOYOKE_NESTING_LEVEL", tt.envValue)
			}
			defer os.Unsetenv("GOYOKE_NESTING_LEVEL")
			// Also clear CLAUDE_CODE_NESTING_LEVEL to isolate tests
			os.Unsetenv("CLAUDE_CODE_NESTING_LEVEL")
			defer os.Unsetenv("CLAUDE_CODE_NESTING_LEVEL")

			result := GetNestingLevel()

			if result != tt.expected {
				t.Errorf("GetNestingLevel() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestIsNestingLevelExplicit(t *testing.T) {
	// Test when not set
	os.Unsetenv("GOYOKE_NESTING_LEVEL")
	if IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = true when env not set")
	}

	// Test when set (even to empty)
	os.Setenv("GOYOKE_NESTING_LEVEL", "")
	// Note: os.Getenv returns "" for both unset and empty, so this tests implementation

	os.Setenv("GOYOKE_NESTING_LEVEL", "0")
	if !IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = false when env is set")
	}

	os.Unsetenv("GOYOKE_NESTING_LEVEL")
}

func TestValidateTaskNestingLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantError bool
	}{
		{
			name:      "level 0 allows Task",
			level:     "0",
			wantError: false,
		},
		{
			name:      "level 1 blocks Task",
			level:     "1",
			wantError: true,
		},
		{
			name:      "level 2 blocks Task",
			level:     "2",
			wantError: true,
		},
		{
			name:      "missing level allows Task (root when CLAUDE_CODE_NESTING_LEVEL also unset)",
			level:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.level == "" {
				os.Unsetenv("GOYOKE_NESTING_LEVEL")
			} else {
				os.Setenv("GOYOKE_NESTING_LEVEL", tt.level)
			}
			defer os.Unsetenv("GOYOKE_NESTING_LEVEL")
			os.Unsetenv("CLAUDE_CODE_NESTING_LEVEL")
			defer os.Unsetenv("CLAUDE_CODE_NESTING_LEVEL")

			err := ValidateTaskNestingLevel()

			if tt.wantError && err == nil {
				t.Error("ValidateTaskNestingLevel() = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateTaskNestingLevel() = %v, want nil", err)
			}
		})
	}
}

func TestValidateTaskInvocation_OpusResume_BypassesAllowlist(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect", "mozart"},
			},
		},
	}

	// Resume call with no AGENT: prefix in prompt (typical resume scenario)
	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "USER RESPONSE: Option 1 - proceed with minimal scope",
		"resume": "a391659",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected opus resume to bypass allowlist, got blocked: %s", result.BlockReason)
	}

	if result.Violation != nil {
		t.Error("Expected no violation for resume call")
	}
}

func TestValidateTaskInvocation_OpusResume_WithAgentPrefix(t *testing.T) {
	// Even if someone includes AGENT: in a resume prompt, allowlist should work normally
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"mozart"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: mozart\n\nContinue analysis",
		"resume": "b123456",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected opus resume with allowlisted agent to be allowed, got blocked: %s", result.BlockReason)
	}
}

func TestValidateTaskInvocation_OpusNoResume_StillBlocked(t *testing.T) {
	// Without resume, non-allowlisted agent should still be blocked
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner", "architect"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "Do something without AGENT prefix",
		// no resume field
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected opus without resume and without allowlisted agent to be blocked")
	}
}

func TestValidateTaskInvocation_NonOpusResume_Unchanged(t *testing.T) {
	// Resume with non-opus model should pass through as normal
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked:   true,
				TaskInvocationAllowlist: []string{"planner"},
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "Continue previous work",
		"resume": "c789012",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if !result.Allowed {
		t.Errorf("Expected sonnet resume to be allowed, got blocked: %s", result.BlockReason)
	}
}

func TestBlockResponseForNesting(t *testing.T) {
	response := BlockResponseForNesting(2, "opus", "opus model requested at Level 2")

	if response["decision"] != "block" {
		t.Errorf("decision = %v, want 'block'", response["decision"])
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("hookSpecificOutput not a map")
	}

	if hookOutput["nestingLevel"] != 2 {
		t.Errorf("nestingLevel = %v, want 2", hookOutput["nestingLevel"])
	}

	if hookOutput["permissionDecision"] != "deny" {
		t.Errorf("permissionDecision = %v, want 'deny'", hookOutput["permissionDecision"])
	}

	if hookOutput["permissionDecisionReason"] != "opus_blocked_at_nesting_level" {
		t.Errorf("permissionDecisionReason = %v, want 'opus_blocked_at_nesting_level'",
			hookOutput["permissionDecisionReason"])
	}

	if hookOutput["model"] != "opus" {
		t.Errorf("model = %v, want 'opus'", hookOutput["model"])
	}
}
