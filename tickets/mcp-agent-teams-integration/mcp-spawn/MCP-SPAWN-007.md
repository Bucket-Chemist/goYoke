```yaml
---
id: MCP-SPAWN-007
title: gogent-validate Nesting Level Check
description: Add nesting level detection to gogent-validate hook to block Task() at Level 1+ with guidance.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-001]
phase: 1
tags: [hooks, go, validation, phase-1]
needs_planning: false
agent: go-pro
priority: CRITICAL
coverage_target: 80
---
```

# MCP-SPAWN-007: gogent-validate Nesting Level Check

## Description

Add nesting level detection to the gogent-validate hook. Block Task() invocations at Level 1+ with clear guidance to use MCP spawn_agent instead. Use fail-closed default (missing/invalid level = assume nested).

**Source**: Einstein Analysis §3.2, Staff-Architect Analysis §4.7.3

## Why This Matters

This is the enforcement mechanism for the hybrid approach. Without it, subagents could attempt Task() and fail with cryptic errors instead of being redirected to MCP spawning.

## Task

1. Add getNestingLevel() function with fail-closed default
2. Add nesting level check to validation logic
3. Return clear block message with guidance
4. Add telemetry for blocked Task() calls

## Files

- `cmd/gogent-validate/main.go` — Add nesting check
- `pkg/routing/task_validation.go` — Core logic
- `pkg/routing/task_validation_test.go` — Tests

## Implementation

### Core Logic (`pkg/routing/task_validation.go`)

```go
package routing

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// MaxNestingDepth prevents runaway nesting
	MaxNestingDepth = 10
	
	// DefaultNestingLevel for fail-closed behavior
	DefaultNestingLevel = 1
)

// GetNestingLevel returns the current nesting level from environment.
// Fail-closed: returns 1 (blocked) if missing or invalid.
func GetNestingLevel() int {
	levelStr := os.Getenv("GOGENT_NESTING_LEVEL")
	
	// Missing = fail-closed (assume nested)
	if levelStr == "" {
		return DefaultNestingLevel
	}
	
	level, err := strconv.Atoi(levelStr)
	
	// Invalid = fail-closed
	if err != nil {
		return DefaultNestingLevel
	}
	
	// Out of range = fail-closed
	if level < 0 || level > MaxNestingDepth {
		return DefaultNestingLevel
	}
	
	return level
}

// IsNestingLevelExplicit returns true if GOGENT_NESTING_LEVEL was set explicitly.
// Used for telemetry to distinguish real Level 0 from assumed nesting.
func IsNestingLevelExplicit() bool {
	return os.Getenv("GOGENT_NESTING_LEVEL") != ""
}

// ValidateTaskNestingLevel checks if Task() is allowed at current nesting level.
// Returns nil if allowed, error with guidance if blocked.
func ValidateTaskNestingLevel() error {
	level := GetNestingLevel()
	
	if level > 0 {
		return &NestingLevelError{
			Level:   level,
			Message: fmt.Sprintf(
				"Task() blocked at nesting level %d. "+
					"Subagents cannot spawn sub-subagents via Task(). "+
					"Use MCP spawn_agent tool instead: "+
					"mcp__gofortress__spawn_agent({agent: '...', prompt: '...'})",
				level,
			),
		}
	}
	
	return nil
}

// NestingLevelError represents a Task() blocked due to nesting level.
type NestingLevelError struct {
	Level   int
	Message string
}

func (e *NestingLevelError) Error() string {
	return e.Message
}

// BlockResponseForNesting creates the standard block response for nesting violations.
func BlockResponseForNesting(level int) map[string]interface{} {
	return map[string]interface{}{
		"decision": "block",
		"reason": fmt.Sprintf(
			"Task() blocked at nesting level %d. Use MCP spawn_agent instead.",
			level,
		),
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "nesting_level_exceeded",
			"nestingLevel":             level,
			"suggestion":               "mcp__gofortress__spawn_agent({agent: '...', prompt: '...'})",
		},
	}
}
```

### Main Hook Update (`cmd/gogent-validate/main.go`)

```go
// Add to main() after parsing input, before existing Task validation

// Check nesting level for Task tool
if event.ToolName == "Task" {
    nestingLevel := routing.GetNestingLevel()
    isExplicit := routing.IsNestingLevelExplicit()
    
    if nestingLevel > 0 {
        // Log the block for telemetry
        logNestingBlock(event, nestingLevel, isExplicit)
        
        // Return block response
        response := routing.BlockResponseForNesting(nestingLevel)
        outputJSON(response)
        return
    }
}

// Helper function for telemetry
func logNestingBlock(event *Event, level int, explicit bool) {
    telemetry := map[string]interface{}{
        "timestamp":     time.Now().UTC().Format(time.RFC3339),
        "event":         "task_blocked_nesting",
        "session_id":    event.SessionID,
        "nesting_level": level,
        "level_explicit": explicit,
        "tool_name":     event.ToolName,
    }
    
    // Append to telemetry file
    telemetryPath := filepath.Join(
        os.Getenv("XDG_DATA_HOME"),
        "gogent",
        "nesting-blocks.jsonl",
    )
    
    appendJSONL(telemetryPath, telemetry)
}
```

### Tests (`pkg/routing/task_validation_test.go`)

```go
package routing

import (
	"os"
	"testing"
)

func TestGetNestingLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "missing env var returns default (fail-closed)",
			envValue: "",
			expected: DefaultNestingLevel,
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
				os.Unsetenv("GOGENT_NESTING_LEVEL")
			} else {
				os.Setenv("GOGENT_NESTING_LEVEL", tt.envValue)
			}
			defer os.Unsetenv("GOGENT_NESTING_LEVEL")
			
			result := GetNestingLevel()
			
			if result != tt.expected {
				t.Errorf("GetNestingLevel() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestIsNestingLevelExplicit(t *testing.T) {
	// Test when not set
	os.Unsetenv("GOGENT_NESTING_LEVEL")
	if IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = true when env not set")
	}
	
	// Test when set (even to empty)
	os.Setenv("GOGENT_NESTING_LEVEL", "")
	// Note: os.Getenv returns "" for both unset and empty, so this tests implementation
	
	os.Setenv("GOGENT_NESTING_LEVEL", "0")
	if !IsNestingLevelExplicit() {
		t.Error("IsNestingLevelExplicit() = false when env is set")
	}
	
	os.Unsetenv("GOGENT_NESTING_LEVEL")
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
			name:      "missing level blocks Task (fail-closed)",
			level:     "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.level == "" {
				os.Unsetenv("GOGENT_NESTING_LEVEL")
			} else {
				os.Setenv("GOGENT_NESTING_LEVEL", tt.level)
			}
			defer os.Unsetenv("GOGENT_NESTING_LEVEL")
			
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

func TestBlockResponseForNesting(t *testing.T) {
	response := BlockResponseForNesting(2)
	
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
}
```

### C3 Enhancement: Hook I/O Schema Alignment

**Requirement:** Hook inputs/outputs must conform to `~/.claude/schemas/hook-io-schema.json`

#### Go Type Definitions (align with schema)

Add to `pkg/routing/hook_types.go`:

```go
package routing

// PreToolUseInput matches hook-io-schema.json#/definitions/PreToolUseInput
type PreToolUseInput struct {
    HookEventName string                 `json:"hook_event_name"`
    SessionID     string                 `json:"session_id"`
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
}

// PreToolUseOutput matches hook-io-schema.json#/definitions/PreToolUseOutput
type PreToolUseOutput struct {
    Decision           string                 `json:"decision"` // "allow", "block", "modify"
    Reason             string                 `json:"reason,omitempty"`
    HookSpecificOutput *PreToolUseHookOutput  `json:"hookSpecificOutput,omitempty"`
}

type PreToolUseHookOutput struct {
    HookEventName            string `json:"hookEventName"`
    PermissionDecision       string `json:"permissionDecision"` // "allow", "deny"
    PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
    NestingLevel             int    `json:"nestingLevel,omitempty"`
    Suggestion               string `json:"suggestion,omitempty"`
}

// ValidateHookOutput validates output against schema (lightweight check)
func ValidateHookOutput(output interface{}) error {
    m, ok := output.(map[string]interface{})
    if !ok {
        return fmt.Errorf("hook output must be JSON object")
    }
    decision, ok := m["decision"].(string)
    if !ok {
        return fmt.Errorf("hook output missing required 'decision' field")
    }
    if decision != "allow" && decision != "block" && decision != "modify" {
        return fmt.Errorf("invalid decision: %s (must be allow/block/modify)", decision)
    }
    return nil
}
```

#### Schema Reference

The hook MUST produce output conforming to:
- Schema: `~/.claude/schemas/hook-io-schema.json`
- Definition: `PreToolUseOutput`
- Required field: `decision` (enum: "allow", "block", "modify")

## Acceptance Criteria

- [ ] GetNestingLevel() returns correct values for all cases
- [ ] Fail-closed behavior: missing/invalid = Level 1 (blocked)
- [ ] ValidateTaskNestingLevel() blocks at Level 1+
- [ ] Block response includes clear guidance for MCP spawn_agent
- [ ] Telemetry logged for blocked Task() calls
- [ ] All tests pass: `go test ./pkg/routing/...`
- [ ] Code coverage ≥80%
- [ ] Hook compiles and runs correctly
- [ ] Hook output conforms to hook-io-schema.json
- [ ] PreToolUseOutput types defined in pkg/routing/hook_types.go

## Test Deliverables

- [ ] Test file updated: `pkg/routing/task_validation_test.go`
- [ ] Number of test functions: 4
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: spawn subagent, attempt Task(), verify block

