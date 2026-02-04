```yaml
---
id: MCP-SPAWN-014
title: Delegation Requirement Enforcement
description: Enforce must_delegate and min_delegations at orchestrator completion via gogent-orchestrator-guard hook.
status: pending
time_estimate: 2h
dependencies: [MCP-SPAWN-007, MCP-SPAWN-013]
phase: 2
tags: [hooks, go, delegation, enforcement, phase-2]
needs_planning: false
agent: go-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-014: Delegation Requirement Enforcement

## Description

Enforce `must_delegate` and `min_delegations` requirements at orchestrator completion. When an orchestrator with `must_delegate: true` completes, verify it spawned at least `min_delegations` children. Block completion if requirement not met.

**Source**: agent-relationships-schema.json validation_rules.delegation_requirement

## Why This Matters

agents-index.json defines delegation requirements:
- `mozart`: must_delegate=true, min_delegations=3 (Einstein, Staff-Architect, Beethoven)
- `review-orchestrator`: must_delegate=true, min_delegations=2
- `impl-manager`: must_delegate=true, min_delegations=1

Without enforcement:
- Orchestrators could complete without spawning required specialists
- Braintrust could return without Einstein/Staff-Architect analysis
- Review could return without any reviewers running

## Task

1. Extend gogent-orchestrator-guard hook (or create if not exists)
2. Load agents-index.json for delegation requirements
3. At SubagentStop, check must_delegate and min_delegations
4. Block completion with guidance if requirements not met

## Files

- `cmd/gogent-orchestrator-guard/main.go` — Hook implementation
- `pkg/routing/delegation.go` — Delegation validation logic
- `pkg/routing/delegation_test.go` — Tests

## Implementation

### Delegation Validation (`pkg/routing/delegation.go`)

```go
package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AgentDelegationConfig holds delegation-related fields from agents-index.json
type AgentDelegationConfig struct {
	ID             string   `json:"id"`
	MustDelegate   bool     `json:"must_delegate,omitempty"`
	MinDelegations int      `json:"min_delegations,omitempty"`
	MaxDelegations int      `json:"max_delegations,omitempty"`
	CanSpawn       []string `json:"can_spawn,omitempty"`
}

// AgentsIndex represents the agents-index.json structure
type AgentsIndex struct {
	Version string                  `json:"version"`
	Agents  []AgentDelegationConfig `json:"agents"`
}

// DelegationValidationResult holds the result of delegation validation
type DelegationValidationResult struct {
	Valid       bool
	AgentID     string
	Required    int
	Actual      int
	Message     string
	Suggestion  string
}

var cachedAgentsIndex *AgentsIndex

// LoadAgentsIndex loads agents-index.json with caching
func LoadAgentsIndex() (*AgentsIndex, error) {
	if cachedAgentsIndex != nil {
		return cachedAgentsIndex, nil
	}

	// Find agents-index.json
	locations := []string{
		filepath.Join(os.Getenv("CLAUDE_PROJECT_DIR"), ".claude", "agents", "agents-index.json"),
		filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json"),
	}

	var indexPath string
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			indexPath = loc
			break
		}
	}

	if indexPath == "" {
		return nil, fmt.Errorf("[delegation] agents-index.json not found")
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("[delegation] failed to read agents-index.json: %w", err)
	}

	var index AgentsIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("[delegation] failed to parse agents-index.json: %w", err)
	}

	cachedAgentsIndex = &index
	return cachedAgentsIndex, nil
}

// GetAgentDelegationConfig retrieves delegation config for an agent
func GetAgentDelegationConfig(agentID string) (*AgentDelegationConfig, error) {
	index, err := LoadAgentsIndex()
	if err != nil {
		return nil, err
	}

	for _, agent := range index.Agents {
		if agent.ID == agentID {
			return &agent, nil
		}
	}

	return nil, nil // Not found, not an error
}

// ValidateDelegationRequirement checks if an orchestrator met its delegation requirements
func ValidateDelegationRequirement(agentType string, childCount int) *DelegationValidationResult {
	config, err := GetAgentDelegationConfig(agentType)
	if err != nil {
		// Can't validate, allow to proceed
		return &DelegationValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("Could not load config for %s: %v", agentType, err),
		}
	}

	if config == nil {
		// Unknown agent, allow
		return &DelegationValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("No config found for agent '%s'", agentType),
		}
	}

	// Check must_delegate
	if !config.MustDelegate {
		return &DelegationValidationResult{
			Valid:   true,
			AgentID: agentType,
			Message: fmt.Sprintf("%s does not require delegation", agentType),
		}
	}

	// Check min_delegations
	if childCount < config.MinDelegations {
		return &DelegationValidationResult{
			Valid:    false,
			AgentID:  agentType,
			Required: config.MinDelegations,
			Actual:   childCount,
			Message: fmt.Sprintf(
				"%s requires at least %d delegations but only spawned %d",
				agentType, config.MinDelegations, childCount,
			),
			Suggestion: fmt.Sprintf(
				"Spawn more agents before completing. Expected: %v",
				config.CanSpawn,
			),
		}
	}

	return &DelegationValidationResult{
		Valid:    true,
		AgentID:  agentType,
		Required: config.MinDelegations,
		Actual:   childCount,
		Message: fmt.Sprintf(
			"%s met delegation requirement (%d/%d)",
			agentType, childCount, config.MinDelegations,
		),
	}
}

// BlockResponseForDelegation creates the standard block response for delegation violations
func BlockResponseForDelegation(result *DelegationValidationResult) map[string]interface{} {
	return map[string]interface{}{
		"decision": "block",
		"reason":   result.Message,
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "SubagentStop",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "delegation_requirement_not_met",
			"agentId":                  result.AgentID,
			"requiredDelegations":      result.Required,
			"actualDelegations":        result.Actual,
			"suggestion":               result.Suggestion,
		},
	}
}

// ClearAgentsIndexCache clears the cached index (for testing)
func ClearAgentsIndexCache() {
	cachedAgentsIndex = nil
}
```

### Hook Implementation (`cmd/gogent-orchestrator-guard/main.go`)

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/doktersmol/gogent-fortress/pkg/routing"
)

// SubagentStopEvent represents the hook input for SubagentStop
type SubagentStopEvent struct {
	SessionID  string `json:"session_id"`
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
	ChildCount int    `json:"child_count"`
	Status     string `json:"status"` // "complete", "error", "timeout"
}

func main() {
	// Read event from stdin
	var event SubagentStopEvent
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&event); err != nil {
		outputError("Failed to parse hook input", err)
		return
	}

	// Only validate on successful completion
	if event.Status != "complete" {
		outputAllow("Agent did not complete successfully, skipping delegation check")
		return
	}

	// Validate delegation requirement
	result := routing.ValidateDelegationRequirement(event.AgentType, event.ChildCount)

	if !result.Valid {
		// Log violation for telemetry
		logDelegationViolation(event, result)

		// Output block response
		response := routing.BlockResponseForDelegation(result)
		outputJSON(response)
		return
	}

	// Log successful validation
	logDelegationSuccess(event, result)

	// Allow completion
	outputAllow(result.Message)
}

func outputAllow(message string) {
	response := map[string]interface{}{
		"decision": "allow",
		"message":  message,
	}
	outputJSON(response)
}

func outputError(message string, err error) {
	response := map[string]interface{}{
		"decision": "allow", // Allow on error (fail-open for this check)
		"error":    fmt.Sprintf("%s: %v", message, err),
	}
	outputJSON(response)
}

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func logDelegationViolation(event SubagentStopEvent, result *routing.DelegationValidationResult) {
	telemetry := map[string]interface{}{
		"timestamp":   getCurrentTimestamp(),
		"event":       "delegation_violation",
		"session_id":  event.SessionID,
		"agent_id":    event.AgentID,
		"agent_type":  event.AgentType,
		"required":    result.Required,
		"actual":      result.Actual,
		"child_count": event.ChildCount,
	}

	appendTelemetry("delegation-violations.jsonl", telemetry)
}

func logDelegationSuccess(event SubagentStopEvent, result *routing.DelegationValidationResult) {
	telemetry := map[string]interface{}{
		"timestamp":   getCurrentTimestamp(),
		"event":       "delegation_met",
		"session_id":  event.SessionID,
		"agent_id":    event.AgentID,
		"agent_type":  event.AgentType,
		"required":    result.Required,
		"actual":      result.Actual,
	}

	appendTelemetry("delegation-success.jsonl", telemetry)
}

// ... helper functions for timestamp and telemetry append
```

### Tests (`pkg/routing/delegation_test.go`)

```go
package routing

import (
	"os"
	"testing"
)

func TestValidateDelegationRequirement(t *testing.T) {
	// Set up mock agents-index.json
	mockIndex := `{
		"version": "test",
		"agents": [
			{
				"id": "mozart",
				"must_delegate": true,
				"min_delegations": 3,
				"can_spawn": ["einstein", "staff-architect", "beethoven"]
			},
			{
				"id": "review-orchestrator",
				"must_delegate": true,
				"min_delegations": 2,
				"can_spawn": ["backend-reviewer", "frontend-reviewer"]
			},
			{
				"id": "go-pro",
				"must_delegate": false
			}
		]
	}`

	// Write mock file
	tmpDir := t.TempDir()
	indexPath := tmpDir + "/agents-index.json"
	os.WriteFile(indexPath, []byte(mockIndex), 0644)
	os.Setenv("HOME", tmpDir)
	os.MkdirAll(tmpDir+"/.claude/agents", 0755)
	os.WriteFile(tmpDir+"/.claude/agents/agents-index.json", []byte(mockIndex), 0644)

	defer ClearAgentsIndexCache()

	tests := []struct {
		name       string
		agentType  string
		childCount int
		wantValid  bool
	}{
		{
			name:       "mozart with 3 children - valid",
			agentType:  "mozart",
			childCount: 3,
			wantValid:  true,
		},
		{
			name:       "mozart with 2 children - invalid",
			agentType:  "mozart",
			childCount: 2,
			wantValid:  false,
		},
		{
			name:       "mozart with 0 children - invalid",
			agentType:  "mozart",
			childCount: 0,
			wantValid:  false,
		},
		{
			name:       "review-orchestrator with 2 children - valid",
			agentType:  "review-orchestrator",
			childCount: 2,
			wantValid:  true,
		},
		{
			name:       "review-orchestrator with 1 child - invalid",
			agentType:  "review-orchestrator",
			childCount: 1,
			wantValid:  false,
		},
		{
			name:       "go-pro (no must_delegate) with 0 children - valid",
			agentType:  "go-pro",
			childCount: 0,
			wantValid:  true,
		},
		{
			name:       "unknown agent - valid (allow)",
			agentType:  "unknown-agent",
			childCount: 0,
			wantValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClearAgentsIndexCache()

			result := ValidateDelegationRequirement(tt.agentType, tt.childCount)

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateDelegationRequirement(%s, %d) = %v, want %v. Message: %s",
					tt.agentType, tt.childCount, result.Valid, tt.wantValid, result.Message)
			}
		})
	}
}

func TestBlockResponseForDelegation(t *testing.T) {
	result := &DelegationValidationResult{
		Valid:      false,
		AgentID:    "mozart",
		Required:   3,
		Actual:     2,
		Message:    "mozart requires at least 3 delegations but only spawned 2",
		Suggestion: "Spawn more agents",
	}

	response := BlockResponseForDelegation(result)

	if response["decision"] != "block" {
		t.Errorf("expected decision 'block', got %v", response["decision"])
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("hookSpecificOutput not a map")
	}

	if hookOutput["requiredDelegations"] != 3 {
		t.Errorf("expected requiredDelegations 3, got %v", hookOutput["requiredDelegations"])
	}

	if hookOutput["actualDelegations"] != 2 {
		t.Errorf("expected actualDelegations 2, got %v", hookOutput["actualDelegations"])
	}
}
```

### C3 Enhancement: Hook I/O Schema Alignment

**Requirement:** Hook inputs/outputs must conform to `~/.claude/schemas/hook-io-schema.json`

#### Go Type Definitions (align with schema)

The SubagentStopEvent struct should align with schema:

```go
// SubagentStopInput matches hook-io-schema.json#/definitions/SubagentStopInput
type SubagentStopInput struct {
    HookEventName string `json:"hook_event_name"` // Must be "SubagentStop"
    SessionID     string `json:"session_id"`
    AgentID       string `json:"agent_id"`
    AgentType     string `json:"agent_type"`
    ChildCount    int    `json:"child_count"`
    Status        string `json:"status"` // "complete", "error", "timeout"
}

// SubagentStopOutput matches hook-io-schema.json#/definitions/SubagentStopOutput
type SubagentStopOutput struct {
    Decision           string                   `json:"decision"`
    Reason             string                   `json:"reason,omitempty"`
    Message            string                   `json:"message,omitempty"`
    Error              string                   `json:"error,omitempty"`
    HookSpecificOutput *SubagentStopHookOutput  `json:"hookSpecificOutput,omitempty"`
}

type SubagentStopHookOutput struct {
    HookEventName            string `json:"hookEventName"`
    PermissionDecision       string `json:"permissionDecision"`
    PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
    AgentID                  string `json:"agentId,omitempty"`
    RequiredDelegations      int    `json:"requiredDelegations,omitempty"`
    ActualDelegations        int    `json:"actualDelegations,omitempty"`
    Suggestion               string `json:"suggestion,omitempty"`
}
```

#### Schema Reference

The hook MUST produce output conforming to:
- Schema: `~/.claude/schemas/hook-io-schema.json`
- Definition: `SubagentStopOutput`
- Required field: `decision` (enum: "allow", "block", "modify")

### M4 Enhancement: Additional Test Coverage

Add these test cases to `pkg/routing/delegation_test.go`:

```go
func TestLoadAgentsIndex_FileNotFound(t *testing.T) {
    // Clear cache and set invalid path
    ClearAgentsIndexCache()
    originalHome := os.Getenv("HOME")
    os.Setenv("HOME", "/nonexistent")
    defer os.Setenv("HOME", originalHome)

    _, err := LoadAgentsIndex()

    if err == nil {
        t.Error("Expected error when agents-index.json not found")
    }
    if !strings.Contains(err.Error(), "not found") {
        t.Errorf("Expected 'not found' in error, got: %v", err)
    }
}

func TestValidateDelegationRequirement_ConfigLoadError(t *testing.T) {
    // Simulate config load failure - should fail-open
    ClearAgentsIndexCache()
    originalHome := os.Getenv("HOME")
    os.Setenv("HOME", "/nonexistent")
    defer os.Setenv("HOME", originalHome)

    result := ValidateDelegationRequirement("mozart", 0)

    // Fail-open: should allow even though we couldn't load config
    if !result.Valid {
        t.Error("Expected fail-open behavior (Valid=true) on config load error")
    }
    if !strings.Contains(result.Message, "Could not load config") {
        t.Errorf("Expected error message, got: %s", result.Message)
    }
}

func TestValidateDelegationRequirement_MoreThanMinimum(t *testing.T) {
    // Setup mock with min_delegations: 3
    setupMockAgentsIndex(t)
    defer ClearAgentsIndexCache()

    // Mozart with 5 children (more than min 3)
    result := ValidateDelegationRequirement("mozart", 5)

    if !result.Valid {
        t.Error("Expected valid when exceeding min_delegations")
    }
    if result.Actual != 5 {
        t.Errorf("Expected actual=5, got %d", result.Actual)
    }
}

func TestBlockResponseForDelegation_IncludesSuggestion(t *testing.T) {
    result := &DelegationValidationResult{
        Valid:      false,
        AgentID:    "review-orchestrator",
        Required:   2,
        Actual:     1,
        Message:    "review-orchestrator requires at least 2 delegations",
        Suggestion: "Spawn backend-reviewer or frontend-reviewer",
    }

    response := BlockResponseForDelegation(result)

    hookOutput := response["hookSpecificOutput"].(map[string]interface{})
    if hookOutput["suggestion"] != "Spawn backend-reviewer or frontend-reviewer" {
        t.Errorf("Expected suggestion in response, got: %v", hookOutput["suggestion"])
    }
}
```

## Acceptance Criteria

- [ ] Hook reads SubagentStop event from stdin
- [ ] Loads agents-index.json for delegation config
- [ ] Validates must_delegate and min_delegations
- [ ] Blocks completion with clear message if requirements not met
- [ ] Allows completion if requirements met or not applicable
- [ ] Logs both violations and successes for telemetry
- [ ] All tests pass: `go test ./pkg/routing/...`
- [ ] Code coverage ≥80%
- [ ] Hook output conforms to hook-io-schema.json
- [ ] SubagentStopOutput types defined in pkg/routing
- [ ] Config load error handled with fail-open behavior

## Test Deliverables

- [ ] Test file: `pkg/routing/delegation_test.go`
- [ ] Number of test functions: 6
- [ ] All tests passing
- [ ] Coverage ≥80%
- [ ] Manual test: Mozart completes with <3 children (blocked), Mozart completes with 3+ children (allowed)

## Schema Alignment

This ticket enforces agent-relationships-schema.json delegation rules:

| Schema Rule | Enforcement |
|-------------|-------------|
| `must_delegate` | Check at SubagentStop |
| `min_delegations` | Block if childCount < min |
| `max_delegations` | Already enforced in MCP-SPAWN-013 at spawn time |

