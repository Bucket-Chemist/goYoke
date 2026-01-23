---
id: GOgent-058
title: Routing Schema Summary Formatter
description: **Task**:
status: pending
time_estimate: 1h
dependencies: []
priority: MEDIUM
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 10
---

## GOgent-058: Routing Schema Summary Formatter

**Time**: 1 hour
**Dependencies**: None
**Priority**: MEDIUM

**Task**:
Add summary formatting method to existing `Schema` type in `pkg/routing/schema.go`.

**File**: `pkg/routing/schema.go` (extend existing)

**Implementation**:
```go
// Add to existing pkg/routing/schema.go

import (
	"strings"
	// ... existing imports
)

// FormatTierSummary generates a human-readable summary of active routing tiers.
// Output is designed for context injection in SessionStart hook.
// Returns formatted string with tier names, patterns, and tools.
func (s *Schema) FormatTierSummary() string {
	var sb strings.Builder
	sb.WriteString("ROUTING TIERS ACTIVE:\n")

	// Define tier order for consistent output
	tierOrder := []string{"haiku", "haiku_thinking", "sonnet", "opus", "external"}

	for _, tierName := range tierOrder {
		tier, exists := s.Tiers[tierName]
		if !exists {
			continue
		}

		// Get first 3 patterns (or fewer if less available)
		patterns := tier.Patterns
		if len(patterns) > 3 {
			patterns = patterns[:3]
		}
		patternStr := strings.Join(patterns, ", ")
		if len(tier.Patterns) > 3 {
			patternStr += "..."
		}

		// Get first 4 tools (or fewer if less available)
		tools := tier.Tools
		if len(tools) > 4 {
			tools = tools[:4]
		}
		toolStr := strings.Join(tools, ", ")
		if len(tier.Tools) > 4 {
			toolStr += "..."
		}

		sb.WriteString(fmt.Sprintf("  • %s: patterns=[%s] → tools=[%s]\n", tierName, patternStr, toolStr))
	}

	// Add delegation ceiling info
	sb.WriteString(fmt.Sprintf("\nDELEGATION CEILING: Set by %s\n", s.DelegationCeiling.SetBy))

	return sb.String()
}

// LoadAndFormatSchemaSummary is a convenience function that loads schema and returns summary.
// Returns graceful message if schema doesn't exist (not an error).
func LoadAndFormatSchemaSummary() (string, error) {
	schema, err := LoadSchema()
	if err != nil {
		// Check if it's a file not found error
		if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found") {
			return "Routing schema not found. Routing validation disabled for this session.", nil
		}
		return "", fmt.Errorf("[routing] Failed to load schema for summary: %w", err)
	}

	return schema.FormatTierSummary(), nil
}
```

**Tests**: Add to `pkg/routing/schema_test.go`

```go
// Add to existing pkg/routing/schema_test.go

func TestSchema_FormatTierSummary(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Patterns: []string{"find files", "search codebase", "grep pattern", "locate code"},
				Tools:    []string{"Glob", "Grep", "Read", "WebFetch", "WebSearch"},
			},
			"sonnet": {
				Patterns: []string{"implement", "refactor", "debug"},
				Tools:    []string{"Read", "Write", "Edit", "Bash"},
			},
		},
		DelegationCeiling: DelegationCeiling{
			SetBy: "calculate-complexity.sh",
		},
	}

	summary := schema.FormatTierSummary()

	// Verify structure
	if !strings.Contains(summary, "ROUTING TIERS ACTIVE") {
		t.Error("Summary should contain header")
	}

	// Verify haiku tier with truncated patterns
	if !strings.Contains(summary, "haiku:") {
		t.Error("Summary should contain haiku tier")
	}
	if !strings.Contains(summary, "find files") {
		t.Error("Summary should contain first pattern")
	}
	if !strings.Contains(summary, "...") {
		t.Error("Summary should indicate truncation")
	}

	// Verify sonnet tier
	if !strings.Contains(summary, "sonnet:") {
		t.Error("Summary should contain sonnet tier")
	}

	// Verify delegation ceiling
	if !strings.Contains(summary, "DELEGATION CEILING") {
		t.Error("Summary should contain delegation ceiling")
	}
}

func TestSchema_FormatTierSummary_EmptyTiers(t *testing.T) {
	schema := &Schema{
		Tiers:             map[string]TierConfig{},
		DelegationCeiling: DelegationCeiling{SetBy: "test"},
	}

	summary := schema.FormatTierSummary()

	if !strings.Contains(summary, "ROUTING TIERS ACTIVE") {
		t.Error("Summary should still contain header for empty tiers")
	}
}
```

**Acceptance Criteria**:
- [ ] `FormatTierSummary()` method added to `Schema` type
- [ ] Limits patterns to 3, tools to 4 with "..." truncation
- [ ] Includes delegation ceiling info
- [ ] `LoadAndFormatSchemaSummary()` handles missing schema gracefully
- [ ] Tests verify formatting, truncation, empty tiers
- [ ] `go test ./pkg/routing/...` passes

**Test Deliverables**:
- [ ] Tests added to: `pkg/routing/schema_test.go`
- [ ] Number of new test functions: 2
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Schema summary provides routing context in every session. Concise formatting prevents context window bloat while maintaining routing awareness.

---
