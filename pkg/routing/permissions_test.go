package routing

import (
	"strings"
	"testing"
)

// TestCheckToolPermission_Allowed verifies that tools in the tier's allowed list are permitted.
func TestCheckToolPermission_Allowed(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read", "Grep", "Glob"},
			},
			"sonnet": {
				Tools: []string{"Read", "Write", "Edit", "Bash"},
			},
		},
	}

	tests := []struct {
		name        string
		tier        string
		tool        string
		wantAllowed bool
	}{
		{
			name:        "haiku allows Read",
			tier:        "haiku",
			tool:        "Read",
			wantAllowed: true,
		},
		{
			name:        "haiku allows Grep",
			tier:        "haiku",
			tool:        "Grep",
			wantAllowed: true,
		},
		{
			name:        "sonnet allows Write",
			tier:        "sonnet",
			tool:        "Write",
			wantAllowed: true,
		},
		{
			name:        "sonnet allows Bash",
			tier:        "sonnet",
			tool:        "Bash",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckToolPermission(schema, tt.tier, tt.tool)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("CheckToolPermission() Allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}

			if result.CurrentTier != tt.tier {
				t.Errorf("CheckToolPermission() CurrentTier = %v, want %v", result.CurrentTier, tt.tier)
			}

			if result.Tool != tt.tool {
				t.Errorf("CheckToolPermission() Tool = %v, want %v", result.Tool, tt.tool)
			}
		})
	}
}

// TestCheckToolPermission_Denied verifies that tools not in the allowed list are denied
// and that the RecommendedTier is populated correctly.
func TestCheckToolPermission_Denied(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read", "Grep"},
			},
			"sonnet": {
				Tools: []string{"Read", "Write", "Edit"},
			},
			"opus": {
				Tools: []string{"Read", "Write", "Edit", "Task"},
			},
		},
	}

	tests := []struct {
		name              string
		tier              string
		tool              string
		wantAllowed       bool
		wantRecommended   string
	}{
		{
			name:            "haiku denies Write",
			tier:            "haiku",
			tool:            "Write",
			wantAllowed:     false,
			wantRecommended: "sonnet", // Write first appears in sonnet
		},
		{
			name:            "haiku denies Task",
			tier:            "haiku",
			tool:            "Task",
			wantAllowed:     false,
			wantRecommended: "opus", // Task only in opus
		},
		{
			name:            "sonnet denies Task",
			tier:            "sonnet",
			tool:            "Task",
			wantAllowed:     false,
			wantRecommended: "opus",
		},
		{
			name:            "haiku denies unknown tool",
			tier:            "haiku",
			tool:            "UnknownTool",
			wantAllowed:     false,
			wantRecommended: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckToolPermission(schema, tt.tier, tt.tool)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("CheckToolPermission() Allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}

			if result.RecommendedTier != tt.wantRecommended {
				t.Errorf("CheckToolPermission() RecommendedTier = %v, want %v", result.RecommendedTier, tt.wantRecommended)
			}
		})
	}
}

// TestCheckToolPermission_Wildcard verifies that a tier with ["*"] allows all tools.
func TestCheckToolPermission_Wildcard(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				Tools: []string{"*"},
			},
		},
	}

	tests := []struct {
		name string
		tool string
	}{
		{name: "wildcard allows Read", tool: "Read"},
		{name: "wildcard allows Write", tool: "Write"},
		{name: "wildcard allows Task", tool: "Task"},
		{name: "wildcard allows Bash", tool: "Bash"},
		{name: "wildcard allows custom tool", tool: "CustomTool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckToolPermission(schema, "opus", tt.tool)

			if !result.Allowed {
				t.Errorf("CheckToolPermission() with wildcard should allow %v, got Allowed = false", tt.tool)
			}

			if len(result.AllowedTools) != 1 || result.AllowedTools[0] != "*" {
				t.Errorf("CheckToolPermission() AllowedTools = %v, want [*]", result.AllowedTools)
			}
		})
	}
}

// TestCheckToolPermission_UnknownTier verifies behavior when tier doesn't exist.
func TestCheckToolPermission_UnknownTier(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read"},
			},
		},
	}

	result := CheckToolPermission(schema, "nonexistent", "Read")

	if result.Allowed {
		t.Errorf("CheckToolPermission() for unknown tier should deny, got Allowed = true")
	}

	if result.RecommendedTier != "unknown" {
		t.Errorf("CheckToolPermission() RecommendedTier = %v, want 'unknown'", result.RecommendedTier)
	}
}

// TestFindTierForTool verifies the search order: haiku → haiku_thinking → sonnet → opus.
// Tests with explicit tools only (no wildcard) to verify search order works correctly.
func TestFindTierForTool(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read", "Grep"},
			},
			"haiku_thinking": {
				Tools: []string{"Read", "Grep", "TodoWrite"},
			},
			"sonnet": {
				Tools: []string{"Read", "Write", "Edit"},
			},
			"opus": {
				Tools: []string{"Read", "Write", "Edit", "Task"},
			},
		},
	}

	tests := []struct {
		name     string
		tool     string
		wantTier string
	}{
		{
			name:     "Read found in haiku (first tier)",
			tool:     "Read",
			wantTier: "haiku",
		},
		{
			name:     "TodoWrite found in haiku_thinking",
			tool:     "TodoWrite",
			wantTier: "haiku_thinking",
		},
		{
			name:     "Write found in sonnet",
			tool:     "Write",
			wantTier: "sonnet",
		},
		{
			name:     "Task found in opus",
			tool:     "Task",
			wantTier: "opus",
		},
		{
			name:     "Unknown tool returns unknown",
			tool:     "NonexistentTool",
			wantTier: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTierForTool(schema, tt.tool)

			if result != tt.wantTier {
				t.Errorf("findTierForTool(%v) = %v, want %v", tt.tool, result, tt.wantTier)
			}
		})
	}
}

// TestFindTierForTool_WildcardPriority verifies that wildcard is found in correct order.
func TestFindTierForTool_WildcardPriority(t *testing.T) {
	// Test that wildcard in earlier tier takes precedence
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read"},
			},
			"sonnet": {
				Tools: []string{"*"},
			},
			"opus": {
				Tools: []string{"*"},
			},
		},
	}

	result := findTierForTool(schema, "AnyTool")

	if result != "sonnet" {
		t.Errorf("findTierForTool() with wildcard in sonnet and opus should return 'sonnet', got %v", result)
	}
}

// TestFormatPermissionError verifies the error message format follows the standard.
func TestFormatPermissionError(t *testing.T) {
	tests := []struct {
		name       string
		permission *ToolPermission
		wantParts  []string // Parts that must appear in the error message
	}{
		{
			name: "basic permission error",
			permission: &ToolPermission{
				Allowed:         false,
				CurrentTier:     "haiku",
				Tool:            "Write",
				AllowedTools:    []string{"Read", "Grep"},
				RecommendedTier: "sonnet",
			},
			wantParts: []string{
				"[routing]",
				"Tool 'Write'",
				"tier 'haiku'",
				"Allowed tools for haiku: [Read, Grep]",
				"requires tier: sonnet",
				"--force-tier=sonnet",
			},
		},
		{
			name: "single allowed tool",
			permission: &ToolPermission{
				Allowed:         false,
				CurrentTier:     "haiku",
				Tool:            "Edit",
				AllowedTools:    []string{"Read"},
				RecommendedTier: "sonnet",
			},
			wantParts: []string{
				"[routing]",
				"Tool 'Edit'",
				"tier 'haiku'",
				"Allowed tools for haiku: [Read]",
				"requires tier: sonnet",
			},
		},
		{
			name: "wildcard allowed tools",
			permission: &ToolPermission{
				Allowed:         false,
				CurrentTier:     "haiku",
				Tool:            "CustomTool",
				AllowedTools:    []string{"*"},
				RecommendedTier: "opus",
			},
			wantParts: []string{
				"[routing]",
				"Tool 'CustomTool'",
				"tier 'haiku'",
				"Allowed tools for haiku: [*]",
				"requires tier: opus",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorMsg := tt.permission.FormatPermissionError()

			// Verify all expected parts are present
			for _, part := range tt.wantParts {
				if !strings.Contains(errorMsg, part) {
					t.Errorf("FormatPermissionError() missing expected part %q in message:\n%s", part, errorMsg)
				}
			}

			// Verify error follows format: "[component] What. Why. How to fix."
			if !strings.HasPrefix(errorMsg, "[routing]") {
				t.Errorf("FormatPermissionError() should start with '[routing]', got: %s", errorMsg)
			}

			// Verify contains "not permitted" (What)
			if !strings.Contains(errorMsg, "not permitted") {
				t.Errorf("FormatPermissionError() should explain 'not permitted', got: %s", errorMsg)
			}

			// Verify contains "requires tier" (Why)
			if !strings.Contains(errorMsg, "requires tier") {
				t.Errorf("FormatPermissionError() should explain 'requires tier', got: %s", errorMsg)
			}

			// Verify contains "Use --force-tier" (How to fix)
			if !strings.Contains(errorMsg, "Use --force-tier") {
				t.Errorf("FormatPermissionError() should provide 'Use --force-tier' fix, got: %s", errorMsg)
			}
		})
	}
}

// TestCheckToolPermission_AllowedToolsPopulated verifies that AllowedTools field is set correctly.
func TestCheckToolPermission_AllowedToolsPopulated(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read", "Grep", "Glob"},
			},
		},
	}

	result := CheckToolPermission(schema, "haiku", "Read")

	if len(result.AllowedTools) != 3 {
		t.Errorf("CheckToolPermission() AllowedTools length = %v, want 3", len(result.AllowedTools))
	}

	expectedTools := map[string]bool{
		"Read": true,
		"Grep": true,
		"Glob": true,
	}

	for _, tool := range result.AllowedTools {
		if !expectedTools[tool] {
			t.Errorf("CheckToolPermission() AllowedTools contains unexpected tool %v", tool)
		}
	}
}

// TestCheckToolPermission_IntegrationWithSchema verifies integration with actual schema structure.
func TestCheckToolPermission_IntegrationWithSchema(t *testing.T) {
	// This test simulates real schema structure from routing-schema.json
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Description: "Mechanical work",
				Tools:       []string{"Read", "Grep", "Glob", "WebFetch"},
			},
			"haiku_thinking": {
				Description: "Structured output",
				Tools:       []string{"Read", "Grep", "Glob", "Write", "Edit", "TodoWrite"},
			},
			"sonnet": {
				Description: "Reasoning required",
				Tools:       []string{"Read", "Write", "Edit", "Bash", "Task"},
			},
			"opus": {
				Description: "All tools allowed",
				Tools:       []string{"*"},
			},
		},
	}

	tests := []struct {
		name              string
		tier              string
		tool              string
		wantAllowed       bool
		wantRecommended   string
	}{
		{
			name:        "haiku can Read",
			tier:        "haiku",
			tool:        "Read",
			wantAllowed: true,
		},
		{
			name:            "haiku cannot Write",
			tier:            "haiku",
			tool:            "Write",
			wantAllowed:     false,
			wantRecommended: "haiku_thinking",
		},
		{
			name:        "haiku_thinking can TodoWrite",
			tier:        "haiku_thinking",
			tool:        "TodoWrite",
			wantAllowed: true,
		},
		{
			name:            "haiku_thinking cannot Task",
			tier:            "haiku_thinking",
			tool:            "Task",
			wantAllowed:     false,
			wantRecommended: "sonnet",
		},
		{
			name:        "sonnet can Bash",
			tier:        "sonnet",
			tool:        "Bash",
			wantAllowed: true,
		},
		{
			name:        "opus wildcard allows anything",
			tier:        "opus",
			tool:        "AnythingAtAll",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckToolPermission(schema, tt.tier, tt.tool)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("CheckToolPermission() Allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}

			if !tt.wantAllowed && tt.wantRecommended != "" {
				if result.RecommendedTier != tt.wantRecommended {
					t.Errorf("CheckToolPermission() RecommendedTier = %v, want %v", result.RecommendedTier, tt.wantRecommended)
				}
			}
		})
	}
}

// TestFindTierForTool_MissingTiers verifies behavior when some tiers are missing.
func TestFindTierForTool_MissingTiers(t *testing.T) {
	// Schema with only some tiers defined
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []string{"Read"},
			},
			// haiku_thinking missing
			"sonnet": {
				Tools: []string{"Write"},
			},
			// opus missing
			// external missing
		},
	}

	tests := []struct {
		name     string
		tool     string
		wantTier string
	}{
		{
			name:     "Read found in haiku",
			tool:     "Read",
			wantTier: "haiku",
		},
		{
			name:     "Write found in sonnet (skips missing haiku_thinking)",
			tool:     "Write",
			wantTier: "sonnet",
		},
		{
			name:     "Unknown tool returns unknown (missing opus wildcard)",
			tool:     "Task",
			wantTier: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTierForTool(schema, tt.tool)

			if result != tt.wantTier {
				t.Errorf("findTierForTool(%v) = %v, want %v", tt.tool, result, tt.wantTier)
			}
		})
	}
}
