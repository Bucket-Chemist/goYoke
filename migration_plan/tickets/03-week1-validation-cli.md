# Week 1 Day 6-7: Task Validation and CLI Build

**File**: `03-week1-validation-cli.md`
**Tickets**: GOgent-020 to 025 (7 tickets including 024b)
**Total Time**: ~11 hours
**Phase**: Week 1 Day 6-7

---

## Navigation

- **Previous**: [02-week1-overrides-permissions.md](02-week1-overrides-permissions.md) - GOgent-010 to 019
- **Next**: [04-week2-session-archive.md](04-week2-session-archive.md) - GOgent-026 to 033
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure

---

## Summary

This file covers Task tool validation - the final piece of routing enforcement:

1. **Einstein/Opus Blocking (Day 6)**: Prevent expensive Task(opus) invocations
2. **Model Validation**: Check model/agent compatibility
3. **Delegation Ceiling**: Enforce max_delegation limits
4. **Subagent Type Validation**: Verify correct subagent_type for each agent
5. **Integration**: Wire everything into gogent-validate CLI binary

**Critical Context**:
- Task tool causes 60K+ token inheritance → expensive for opus
- Einstein should use /einstein slash command, not Task tool
- Delegation ceiling prevents spawning agents above complexity-determined tier
- Subagent_type mismatch causes silent failures (wrong tool permissions)

---

## Day 6: Task Validation Logic (GOgent-020 to 023)

### GOgent-020: Implement Einstein/Opus Blocking

**Time**: 2 hours
**Dependencies**: GOgent-017 (tool permissions), GOgent-011 (violation logging)

**Task**:
Block Task tool invocations when model=opus OR target agent=einstein. These must use /einstein slash command to avoid 60K token inheritance overhead.

**File**: `pkg/routing/task_validation.go`

**Imports**:
```go
package routing

import (
	"fmt"
	"regexp"
	"strings"
)
```

**Implementation**:
```go
// TaskValidationResult represents result of Task tool validation
type TaskValidationResult struct {
	Allowed       bool
	BlockReason   string
	Violation     *Violation
	Recommendation string
}

// ValidateTaskInvocation checks if Task tool usage is allowed
func ValidateTaskInvocation(schema *Schema, taskInput map[string]interface{}, sessionID string) *TaskValidationResult {
	result := &TaskValidationResult{Allowed: true}

	// Extract model and prompt
	model, _ := taskInput["model"].(string)
	prompt, _ := taskInput["prompt"].(string)

	// Extract target agent from prompt (pattern: "AGENT: agent-id")
	targetAgent := extractAgentFromPrompt(prompt)

	// Check if opus invocations are blocked
	opusConfig, exists := schema.Tiers["opus"]
	if !exists {
		return result // No opus config, allow
	}

	taskBlocked := opusConfig.TaskInvocationBlocked
	if !taskBlocked {
		return result // Blocking not enabled, allow
	}

	// Block 1: Model is opus (regardless of target agent)
	if model == "opus" {
		result.Allowed = false
		result.BlockReason = "Task(model: opus) causes 60K token inheritance ($3.30 cost). Use /einstein slash command instead ($0.92 cost)."
		result.Recommendation = "Generate GAP document to .claude/tmp/einstein-gap-{timestamp}.md, then notify user to run /einstein. See GAP-003b for rationale."

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "blocked_task_opus",
			Model:         "opus",
			Agent:         targetAgent,
			Reason:        "model_is_opus",
		}

		return result
	}

	// Block 2: Target agent is einstein (regardless of model specified)
	if targetAgent == "einstein" {
		result.Allowed = false
		result.BlockReason = fmt.Sprintf("Einstein must be invoked via /einstein slash command, not Task tool (even with model: %s). Task tool causes 60K token inheritance.", model)
		result.Recommendation = "Generate GAP document, then notify user to run /einstein."

		result.Violation = &Violation{
			SessionID:     sessionID,
			ViolationType: "blocked_task_einstein",
			Model:         model,
			Agent:         "einstein",
			Reason:        "agent_is_einstein",
		}

		return result
	}

	return result
}

// extractAgentFromPrompt finds "AGENT: agent-id" pattern in prompt
func extractAgentFromPrompt(prompt string) string {
	re := regexp.MustCompile(`AGENT:\s*([a-z-]+)`)
	matches := re.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
```

**Tests**: `pkg/routing/task_validation_test.go`

```go
package routing

import (
	"testing"
)

func TestValidateTaskInvocation_OpusModelBlocked(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nImplement feature",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected Task(model: opus) to be blocked")
	}

	if result.Violation.ViolationType != "blocked_task_opus" {
		t.Errorf("Expected violation type blocked_task_opus, got: %s", result.Violation.ViolationType)
	}

	if !strings.Contains(result.BlockReason, "60K token") {
		t.Error("Block reason should mention token inheritance")
	}
}

func TestValidateTaskInvocation_EinsteinAgentBlocked(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				TaskInvocationBlocked: true,
			},
		},
	}

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: einstein\n\nAnalyze this problem",
	}

	result := ValidateTaskInvocation(schema, taskInput, "test-session")

	if result.Allowed {
		t.Error("Expected einstein agent to be blocked")
	}

	if result.Violation.Agent != "einstein" {
		t.Errorf("Expected agent einstein, got: %s", result.Violation.Agent)
	}

	if !strings.Contains(result.Recommendation, "GAP document") {
		t.Error("Recommendation should mention GAP document")
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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

**Acceptance Criteria**:
- [x] `ValidateTaskInvocation()` blocks Task(model: opus)
- [x] Blocks Task with target agent=einstein (any model)
- [x] Allows normal Task invocations (sonnet, haiku agents)
- [x] Creates violation record with correct type
- [x] Violation includes model, agent, and reason
- [x] Tests cover opus model, einstein agent, allowed cases
- [x] `go test ./pkg/routing` passes

**Why This Matters**: Prevents $3+ cost waste from Task(opus) calls. Forces use of /einstein which is 72% cheaper.

---

### GOgent-021: Implement Model Mismatch Warnings

**Time**: 1.5 hours
**Dependencies**: GOgent-020

**Task**:
Warn when Task model doesn't match agent's expected model from agents-index.json.

**File**: `pkg/routing/task_validation.go` (extend existing)

**Add to existing file**:
```go
// AgentConfig represents agent metadata from agents-index.json
type AgentConfig struct {
	Model          string   `json:"model"`
	SubagentType   string   `json:"subagent_type"`
	AllowedModels  []string `json:"allowed_models,omitempty"`
}

// AgentsIndex represents the full agents-index.json structure
type AgentsIndex struct {
	Agents map[string]AgentConfig `json:"agents"`
}

// ValidateModelMatch checks if Task model matches agent's expected model
// Warning messages are logged to violations.jsonl with type "model_mismatch_warning"
// and included in CLI output's additionalContext field
func ValidateModelMatch(agentName string, agentConfig *AgentConfig, requestedModel string) (bool, string) {
	// If agent specifies allowed_models, check against that list
	if len(agentConfig.AllowedModels) > 0 {
		for _, allowed := range agentConfig.AllowedModels {
			if allowed == requestedModel {
				return true, ""
			}
		}

		return false, fmt.Sprintf(
			"[task-validation] Model mismatch. Agent expects models: %v. Requested: %s. This may cause unexpected behavior.",
			agentConfig.AllowedModels,
			requestedModel,
		)
	}

	// Otherwise check against single model field
	if agentConfig.Model != requestedModel {
		return false, fmt.Sprintf(
			"[task-validation] Model mismatch. Agent '%s' expects model '%s'. Requested: '%s'. This may cause suboptimal performance.",
			agentName,
			agentConfig.Model,
			requestedModel,
		)
	}

	return true, ""
}
```

**Tests**: `pkg/routing/task_validation_test.go` (extend existing)

**Add to existing test file**:
```go
func TestValidateModelMatch_ExactMatch(t *testing.T) {
	agentConfig := &AgentConfig{
		Model: "sonnet",
	}

	matches, warning := ValidateModelMatch(agentConfig, "sonnet")

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

	matches, warning := ValidateModelMatch(agentConfig, "haiku")

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
			matches, _ := ValidateModelMatch(agentConfig, tt.model)
			if matches != tt.expected {
				t.Errorf("Model %s: expected match=%v, got %v", tt.model, tt.expected, matches)
			}
		})
	}
}
```

**Acceptance Criteria**:
- [ ] `ValidateModelMatch()` returns true when models match
- [ ] Returns false + warning when mismatch detected
- [ ] Handles allowed_models array (any match is OK)
- [ ] Warning message includes expected and requested models
- [ ] Tests cover exact match, mismatch, and allowed array
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Model mismatches can cause subtle failures. Warnings help users catch configuration errors early.

---

### GOgent-022: Implement Delegation Ceiling Enforcement

**Time**: 2 hours
**Dependencies**: GOgent-020

**Task**:
Check if requested Task model exceeds delegation ceiling set by calculate-complexity.sh.

**File**: `pkg/routing/delegation.go`

**Imports**:
```go
package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)
```

**Implementation**:
```go
// DelegationCeiling represents max allowed delegation tier
type DelegationCeiling struct {
	MaxTier string // e.g., "haiku", "sonnet"
}

// LoadDelegationCeiling reads max_delegation file from project
func LoadDelegationCeiling(projectDir string) (*DelegationCeiling, error) {
	ceilingPath := filepath.Join(projectDir, ".claude", "tmp", "max_delegation")

	data, err := os.ReadFile(ceilingPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No ceiling file = no restriction (default: sonnet)
			return &DelegationCeiling{MaxTier: "sonnet"}, nil
		}
		return nil, fmt.Errorf("[delegation] Failed to read ceiling file at %s: %w", ceilingPath, err)
	}

	maxTier := strings.TrimSpace(string(data))
	if maxTier == "" {
		maxTier = "sonnet" // Default
	}

	return &DelegationCeiling{MaxTier: maxTier}, nil
}

// CheckDelegationCeiling validates if requested model is within ceiling
func CheckDelegationCeiling(schema *Schema, ceiling *DelegationCeiling, requestedModel string) (bool, string) {
	// Get tier level for ceiling
	ceilingLevel, err := schema.GetTierLevel(ceiling.MaxTier)
	if err != nil {
		// Unknown ceiling tier, allow (permissive fallback)
		return true, ""
	}

	// Get tier level for requested model
	requestedLevel, err := schema.GetTierLevel(requestedModel)
	if err != nil {
		// Unknown requested tier, allow
		return true, ""
	}

	if requestedLevel > ceilingLevel {
		return false, fmt.Sprintf(
			"[delegation] Requested model '%s' (level %d) exceeds delegation ceiling '%s' (level %d). Complexity analysis determined max tier. Use --force-delegation=%s to override.",
			requestedModel,
			requestedLevel,
			ceiling.MaxTier,
			ceilingLevel,
			requestedModel,
		)
	}

	return true, ""
}
```

**Tests**: `pkg/routing/delegation_test.go`

```go
package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDelegationCeiling_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	ceilingDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(ceilingDir, 0755)

	ceilingPath := filepath.Join(ceilingDir, "max_delegation")
	os.WriteFile(ceilingPath, []byte("haiku"), 0644)

	ceiling, err := LoadDelegationCeiling(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load ceiling: %v", err)
	}

	if ceiling.MaxTier != "haiku" {
		t.Errorf("Expected max tier 'haiku', got: %s", ceiling.MaxTier)
	}
}

func TestLoadDelegationCeiling_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	ceiling, err := LoadDelegationCeiling(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error when file missing, got: %v", err)
	}

	if ceiling.MaxTier != "sonnet" {
		t.Errorf("Expected default 'sonnet', got: %s", ceiling.MaxTier)
	}
}

func TestCheckDelegationCeiling_WithinCeiling(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{
			Haiku:  10,
			Sonnet: 20,
			Opus:   30,
		},
	}

	ceiling := &DelegationCeiling{MaxTier: "sonnet"}

	// Request haiku (below ceiling)
	allowed, msg := CheckDelegationCeiling(schema, ceiling, "haiku")
	if !allowed {
		t.Errorf("haiku should be allowed under sonnet ceiling: %s", msg)
	}

	// Request sonnet (at ceiling)
	allowed, msg = CheckDelegationCeiling(schema, ceiling, "sonnet")
	if !allowed {
		t.Errorf("sonnet should be allowed at sonnet ceiling: %s", msg)
	}
}

func TestCheckDelegationCeiling_ExceedsCeiling(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{
			Haiku:  10,
			Sonnet: 20,
			Opus:   30,
		},
	}

	ceiling := &DelegationCeiling{MaxTier: "haiku"}

	// Request sonnet (above haiku ceiling)
	allowed, msg := CheckDelegationCeiling(schema, ceiling, "sonnet")
	if allowed {
		t.Error("sonnet should not be allowed under haiku ceiling")
	}

	if msg == "" {
		t.Error("Expected error message for ceiling violation")
	}

	if !contains(msg, "haiku") || !contains(msg, "sonnet") {
		t.Errorf("Message should mention both tiers: %s", msg)
	}

	if !contains(msg, "--force-delegation=") {
		t.Error("Message should suggest override flag")
	}
}

func TestCheckDelegationCeiling_NoTierLevels(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{},
	}

	ceiling := &DelegationCeiling{MaxTier: "haiku"}

	// Should allow when no tier levels defined
	allowed, _ := CheckDelegationCeiling(schema, ceiling, "opus")
	if !allowed {
		t.Error("Should allow all when tier levels not defined")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

**Acceptance Criteria**:
- [ ] `LoadDelegationCeiling()` reads max_delegation file
- [ ] Returns default "sonnet" when file doesn't exist
- [ ] `CheckDelegationCeiling()` compares tier levels correctly
- [ ] Allows models at or below ceiling level
- [ ] Blocks models above ceiling level
- [ ] Error message includes both tiers and override suggestion
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Delegation ceiling prevents over-spending on complex tasks that scout determined should use cheaper tiers.

---

### GOgent-023: Implement Subagent_type Validation

**Time**: 2.5 hours
**Dependencies**: GOgent-020

**Task**:
Validate that Task invocations use the correct subagent_type for the target agent (mapped in routing-schema.json).

**File**: `pkg/routing/subagent_validation.go`

**Imports**:
```go
package routing

import (
	"fmt"
)
```

**Implementation**:
```go
// SubagentTypeValidation represents result of subagent_type check
type SubagentTypeValidation struct {
	Valid            bool
	RequestedType    string
	RequiredType     string
	Agent            string
	ErrorMessage     string
}

// ValidateSubagentType checks if Task uses correct subagent_type for agent
func ValidateSubagentType(schema *Schema, targetAgent string, requestedType string) *SubagentTypeValidation {
	result := &SubagentTypeValidation{
		Agent:         targetAgent,
		RequestedType: requestedType,
	}

	// If no agent specified, can't validate
	if targetAgent == "" {
		result.Valid = true
		return result
	}

	// Use schema method to get required type
	requiredType, err := schema.GetSubagentTypeForAgent(targetAgent)
	if err != nil {
		// Agent not in mapping, allow (might be custom agent)
		result.Valid = true
		return result
	}

	result.RequiredType = requiredType

	// Check if types match
	if requestedType != requiredType {
		result.Valid = false
		result.ErrorMessage = fmt.Sprintf(
			"[task-validation] Invalid subagent_type for agent '%s'. Required: '%s'. Requested: '%s'. Subagent_type mismatch causes wrong tool permissions. See routing-schema.json → agent_subagent_mapping.",
			targetAgent,
			requiredType,
			requestedType,
		)
		return result
	}

	result.Valid = true
	return result
}

// FormatSubagentTypeError creates detailed error with fix suggestion
func (v *SubagentTypeValidation) FormatSubagentTypeError() string {
	if v.Valid {
		return ""
	}

	return fmt.Sprintf(
		"%s\n\nFix: Change subagent_type to '%s' in Task() call.\nExample: Task({subagent_type: '%s', prompt: 'AGENT: %s\\n\\n...'})",
		v.ErrorMessage,
		v.RequiredType,
		v.RequiredType,
		v.Agent,
	)
}
```

**Tests**: `pkg/routing/subagent_validation_test.go`

```go
package routing

import (
	"strings"
	"testing"
)

func TestValidateSubagentType_Correct(t *testing.T) {
	schema := &Schema{
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro:      "general-purpose",
			CodebaseSearch: "Explore",
			Orchestrator:   "Plan",
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
			PythonPro:      "general-purpose",
			CodebaseSearch: "Explore",
		},
	}

	// Wrong type for python-pro
	result := ValidateSubagentType(schema, "python-pro", "Explore")

	if result.Valid {
		t.Error("Expected invalid result for wrong subagent_type")
	}

	if result.RequiredType != "general-purpose" {
		t.Errorf("Expected required type 'general-purpose', got: %s", result.RequiredType)
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
			PythonPro: "general-purpose",
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
			PythonPro: "general-purpose",
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
		RequiredType:  "general-purpose",
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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

**Acceptance Criteria**:
- [ ] `ValidateSubagentType()` returns valid when types match
- [ ] Returns invalid + error when mismatch detected
- [ ] Looks up required type from schema.AgentSubagentMapping
- [ ] Allows unmapped agents (custom agents)
- [ ] Allows when no mapping defined (backwards compatibility)
- [ ] `FormatSubagentTypeError()` includes fix with correct syntax
- [ ] `go test ./pkg/routing` passes with ≥80% coverage

**Why This Matters**: Subagent_type mismatch is a silent killer - agent gets wrong tool permissions and fails mysteriously. Programmatic enforcement catches this immediately.

---

## Day 7: Integration and CLI (GOgent-024, 024b, 025)

### GOgent-024: Task Validation Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-023

**Task**:
Integration tests for complete Task validation workflow.

**File**: `test/integration/task_validation_test.go`

**Implementation**:
```go
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
		name           string
		agent          string
		subagentType   string
		shouldBeValid  bool
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
```

**Acceptance Criteria**:
- [ ] Tests validate complete Task workflow (all checks)
- [ ] Tests cover valid invocation passing all checks
- [ ] Tests verify opus blocking, einstein blocking
- [ ] Tests verify subagent_type validation
- [ ] Real-world scenarios test all common agents
- [ ] `go test ./test/integration` passes
- [ ] Tests demonstrate end-to-end validation

**Why This Matters**: Integration tests ensure all validation layers work together correctly.

---

### GOgent-024b: Wire Validation Orchestrator

**Time**: 1 hour
**Dependencies**: GOgent-024

**Task**:
Create validation orchestrator that runs all checks in sequence and returns combined result.

**File**: `pkg/routing/validator.go`

**Imports**:
```go
package routing

import (
	"encoding/json"
	"fmt"
)
```

**Implementation**:
```go
// NewValidationOrchestrator creates orchestrator with all dependencies loaded
func NewValidationOrchestrator(schema *Schema, projectDir string, agentsIndex *AgentsIndex) *ValidationOrchestrator {
	return &ValidationOrchestrator{
		Schema:      schema,
		ProjectDir:  projectDir,
		AgentsIndex: agentsIndex,
	}
}

// ValidationOrchestrator coordinates all Task validation checks
type ValidationOrchestrator struct {
	Schema      *Schema
	ProjectDir  string
	AgentsIndex *AgentsIndex
}

// ValidationResult combines all validation outcomes
type ValidationResult struct {
	Decision            string                  `json:"decision"` // "allow" or "block"
	Reason              string                  `json:"reason,omitempty"`
	EinsteinBlocked     *TaskValidationResult   `json:"einstein_blocked,omitempty"`
	ModelMismatch       string                  `json:"model_mismatch,omitempty"`
	CeilingViolation    string                  `json:"ceiling_violation,omitempty"`
	SubagentTypeInvalid *SubagentTypeValidation `json:"subagent_type_invalid,omitempty"`
	Violations          []*Violation            `json:"violations,omitempty"`
}

// ValidateTask runs all validation checks on Task invocation
func (v *ValidationOrchestrator) ValidateTask(taskInput map[string]interface{}, sessionID string) *ValidationResult {
	result := &ValidationResult{
		Decision: "allow",
	}

	// Extract fields
	model, _ := taskInput["model"].(string)
	prompt, _ := taskInput["prompt"].(string)
	subagentType, _ := taskInput["subagent_type"].(string)
	targetAgent := extractAgentFromPrompt(prompt)

	// Check 1: Einstein/Opus blocking
	einsteinCheck := ValidateTaskInvocation(v.Schema, taskInput, sessionID)
	if !einsteinCheck.Allowed {
		result.Decision = "block"
		result.Reason = einsteinCheck.BlockReason
		result.EinsteinBlocked = einsteinCheck
		if einsteinCheck.Violation != nil {
			result.Violations = append(result.Violations, einsteinCheck.Violation)
		}
		return result // Hard block, no further checks
	}

	// Check 2: Model mismatch (warning only, not blocking)
	if v.AgentsIndex != nil && targetAgent != "" {
		if agentConfig, exists := v.AgentsIndex.Agents[targetAgent]; exists {
			matches, warning := ValidateModelMatch(targetAgent, &agentConfig, model)
			if !matches {
				result.ModelMismatch = warning
				// Don't block, just warn
			}
		}
	}

	// Check 3: Delegation ceiling
	ceiling, err := LoadDelegationCeiling(v.ProjectDir)
	if err == nil && ceiling != nil {
		allowed, ceilingMsg := CheckDelegationCeiling(v.Schema, ceiling, model)
		if !allowed {
			result.Decision = "block"
			result.Reason = ceilingMsg
			result.CeilingViolation = ceilingMsg

			// Log violation
			violation := &Violation{
				SessionID:     sessionID,
				ViolationType: "delegation_ceiling",
				Model:         model,
				Agent:         targetAgent,
				Reason:        fmt.Sprintf("Ceiling: %s, Requested: %s", ceiling.MaxTier, model),
			}
			result.Violations = append(result.Violations, violation)
			return result // Hard block
		}
	}

	// Check 4: Subagent_type validation
	subagentCheck := ValidateSubagentType(v.Schema, targetAgent, subagentType)
	if !subagentCheck.Valid {
		result.Decision = "block"
		result.Reason = subagentCheck.FormatSubagentTypeError()
		result.SubagentTypeInvalid = subagentCheck

		// Log violation
		violation := &Violation{
			SessionID:     sessionID,
			ViolationType: "subagent_type_mismatch",
			Agent:         targetAgent,
			Reason:        fmt.Sprintf("Required: %s, Requested: %s", subagentCheck.RequiredType, subagentCheck.RequestedType),
		}
		result.Violations = append(result.Violations, violation)
		return result // Hard block
	}

	return result
}

// ToJSON serializes validation result to JSON
func (v *ValidationResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

**Tests**: `pkg/routing/validator_test.go`

```go
package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationOrchestrator_AllowedTask(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: true},
		},
		TierLevels: TierLevels{
			Haiku: 10, Sonnet: 20,
		},
		AgentSubagentMapping: AgentSubagentMapping{
			PythonPro: "general-purpose",
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil)

	taskInput := map[string]interface{}{
		"model":         "sonnet",
		"prompt":        "AGENT: python-pro\n\nImplement feature",
		"subagent_type": "general-purpose",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "allow" {
		t.Errorf("Expected allow, got: %s (reason: %s)", result.Decision, result.Reason)
	}

	if len(result.Violations) > 0 {
		t.Errorf("Expected no violations, got: %d", len(result.Violations))
	}
}

func TestValidationOrchestrator_OpusBlocked(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: true},
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil)

	taskInput := map[string]interface{}{
		"model":  "opus",
		"prompt": "AGENT: python-pro\n\nComplex task",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Opus should be blocked")
	}

	if result.EinsteinBlocked == nil {
		t.Error("Expected einstein blocked result")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violation logged")
	}
}

func TestValidationOrchestrator_CeilingViolation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create delegation ceiling file
	ceilingDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(ceilingDir, 0755)
	os.WriteFile(filepath.Join(ceilingDir, "max_delegation"), []byte("haiku"), 0644)

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: false}, // Allow opus at schema level
		},
		TierLevels: TierLevels{
			Haiku: 10, Sonnet: 20,
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil)

	taskInput := map[string]interface{}{
		"model":  "sonnet",
		"prompt": "AGENT: python-pro\n\nTask",
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Ceiling violation should block")
	}

	if result.CeilingViolation == "" {
		t.Error("Expected ceiling violation message")
	}
}

func TestValidationOrchestrator_SubagentTypeMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {TaskInvocationBlocked: false},
		},
		AgentSubagentMapping: AgentSubagentMapping{
			CodebaseSearch: "Explore",
		},
	}

	orchestrator := NewValidationOrchestrator(schema, tmpDir, nil)

	taskInput := map[string]interface{}{
		"model":         "sonnet",
		"prompt":        "AGENT: codebase-search\n\nFind files",
		"subagent_type": "general-purpose", // Wrong!
	}

	result := orchestrator.ValidateTask(taskInput, "test-session")

	if result.Decision != "block" {
		t.Error("Subagent_type mismatch should block")
	}

	if result.SubagentTypeInvalid == nil {
		t.Error("Expected subagent type validation result")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violation logged")
	}
}

func TestValidationResult_ToJSON(t *testing.T) {
	result := &ValidationResult{
		Decision: "block",
		Reason:   "Test block reason",
		Violations: []*Violation{
			{
				ViolationType: "test_violation",
				Reason:        "Test reason",
			},
		},
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	if parsed["decision"] != "block" {
		t.Error("JSON should contain decision field")
	}
}
```

**Acceptance Criteria**:
- [x] `ValidationOrchestrator` runs all checks in sequence
- [x] Returns "allow" when all checks pass
- [x] Returns "block" on first hard failure (opus, ceiling, subagent_type)
- [x] Model mismatch is warning only, doesn't block
- [x] Collects all violations for logging
- [x] `ToJSON()` outputs valid JSON structure
- [x] Tests cover allow, opus block, ceiling block, subagent_type block
- [x] `go test ./pkg/routing` passes

**Why This Matters**: Orchestrator provides single entry point for all Task validation. Makes hook implementation simpler.

---

### GOgent-025: Build gogent-validate CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-024b

**Task**:
Build CLI binary that reads JSON from STDIN, validates Task invocations, outputs decision.

**File**: `cmd/gogent-validate/main.go`

**Imports**:
```go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Get project directory from environment or current directory
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	// Load routing schema
	schema, err := routing.LoadSchema()
	if err != nil {
		outputError(fmt.Sprintf("Failed to load routing schema: %v", err))
		os.Exit(1)
	}

	// Parse event from STDIN with timeout
	event, err := parseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only validate Task tool
	if event.ToolName != "Task" {
		// Pass through for non-Task tools
		fmt.Println("{}")
		return
	}

	// Create validation orchestrator
	orchestrator := routing.NewValidationOrchestrator(schema, projectDir, nil)

	// Validate task
	result := orchestrator.ValidateTask(event.ToolInput, event.SessionID)

	// Output result
	outputResult(result, event.SessionID)

	// Log violations if any
	for _, violation := range result.Violations {
		if err := routing.LogViolation(violation, projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log violation: %v\n", err)
		}
	}
}

// parseEvent reads and parses ToolEvent from STDIN with timeout
func parseEvent(r io.Reader, timeout time.Duration) (*routing.ToolEvent, error) {
	type result struct {
		event *routing.ToolEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		reader := bufio.NewReader(r)
		data, err := io.ReadAll(reader)
		if err != nil {
			ch <- result{nil, err}
			return
		}

		var event routing.ToolEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("invalid JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("STDIN read timeout after %v", timeout)
	}
}

// outputResult writes validation result as JSON to STDOUT
func outputResult(result *routing.ValidationResult, sessionID string) {
	output := make(map[string]interface{})

	if result.Decision == "block" {
		output["decision"] = "block"
		output["reason"] = result.Reason
		output["hookSpecificOutput"] = map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": result.Reason,
		}
	} else {
		// Allow with optional warnings
		if result.ModelMismatch != "" {
			output["hookSpecificOutput"] = map[string]interface{}{
				"hookEventName":     "PreToolUse",
				"additionalContext": "⚠️ " + result.ModelMismatch,
			}
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

// outputError writes error message in hook format
func outputError(message string) {
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PreToolUse",
			"additionalContext": "🔴 " + message,
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}
```

**Build Script**: `scripts/build-validate.sh`

```bash
#!/bin/bash
# Build gogent-validate binary

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

echo "Building gogent-validate..."
go build -o bin/gogent-validate cmd/gogent-validate/main.go

echo "✓ Built: bin/gogent-validate"
echo ""
echo "Test with:"
echo "  echo '{\"tool_name\":\"Task\",\"tool_input\":{\"model\":\"opus\"},\"session_id\":\"test\"}' | ./bin/gogent-validate"
```

**Installation Script**: `scripts/install-validate.sh`

```bash
#!/bin/bash
# Install gogent-validate to ~/.local/bin

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Build first
"$SCRIPT_DIR/build-validate.sh"

# Install
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

cp "$PROJECT_ROOT/bin/gogent-validate" "$INSTALL_DIR/gogent-validate"
chmod +x "$INSTALL_DIR/gogent-validate"

echo "✓ Installed to: $INSTALL_DIR/gogent-validate"
echo ""
echo "Make sure $INSTALL_DIR is in your PATH:"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
```

**Acceptance Criteria**:
- [ ] CLI reads JSON from STDIN with 5s timeout
- [ ] Validates Task invocations using ValidationOrchestrator
- [ ] Outputs decision as JSON to STDOUT
- [ ] Passes through non-Task tools unchanged
- [ ] Logs violations to JSONL file
- [ ] Build script creates bin/gogent-validate
- [ ] Installation script copies to ~/.local/bin
- [ ] Manual test: `echo '{"tool_name":"Task","tool_input":{"model":"opus"},"session_id":"test"}' | ./bin/gogent-validate`
- [ ] Manual test verifies opus blocking works

**Why This Matters**: CLI binary is the interface used by Claude Code hooks. Must be reliable, fast (<5ms p99), and produce correct JSON.

---

## Cross-File References

- **Depends on**: [02-week1-overrides-permissions.md](02-week1-overrides-permissions.md) - GOgent-017 (permissions), GOgent-011 (violations)
- **Used by**: Week 2 session-archive hook will invoke gogent-validate
- **Standards**: [00-overview.md](00-overview.md) - Error format, STDIN timeout, testing strategy

---

## Quick Reference

**Key Functions Added**:
- `routing.ValidateTaskInvocation()` - Einstein/opus blocking
- `routing.ValidateModelMatch()` - Model compatibility check
- `routing.LoadDelegationCeiling()` - Read max_delegation file
- `routing.CheckDelegationCeiling()` - Enforce tier ceiling
- `routing.ValidateSubagentType()` - Verify subagent_type mapping
- `routing.ValidationOrchestrator.ValidateTask()` - Run all checks
- `gogent-validate` CLI - STDIN → validation → STDOUT

**Files Created**:
- `pkg/routing/task_validation.go`
- `pkg/routing/delegation.go`
- `pkg/routing/subagent_validation.go`
- `pkg/routing/validator.go`
- `cmd/gogent-validate/main.go`
- `test/integration/task_validation_test.go`
- `scripts/build-validate.sh`
- `scripts/install-validate.sh`

**Total Lines**: ~1100 lines of implementation + tests

---

## Completion Checklist

Before marking this file complete, verify:

- [ ] All 7 tickets (GOgent-020 to 025 including 024b) have complete implementations
- [ ] All functions include complete imports
- [ ] All error messages follow `[component] What. Why. How.` format
- [ ] STDIN timeout implemented (5s default)
- [ ] All tests include positive, negative, and edge cases
- [ ] Test coverage ≥80% for all packages
- [ ] All acceptance criteria checkboxes filled
- [ ] CLI binary buildable with provided scripts
- [ ] Manual tests verify opus/einstein blocking works
- [ ] Cross-references to other files accurate
- [ ] No "omitted for brevity" or "implement logic here" placeholders

---

**Next**: [04-week2-session-archive.md](04-week2-session-archive.md) - GOgent-026 to 033 (Session archive hook translation)
