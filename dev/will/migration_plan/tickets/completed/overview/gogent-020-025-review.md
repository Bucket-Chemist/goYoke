# Critical Review: GOgent-020 to 025 (Task Validation and CLI Build)

**Reviewer**: staff-architect-critical-review
**Date**: 2026-01-18
**Ticket File**: migration_plan/tickets/03-week1-validation-cli.md
**Review Framework**: 7-Layer Critical Analysis
**Total Tickets**: 7 (GOgent-020, 021, 022, 023, 024, 024b, 025)
**Estimated Time**: ~11 hours

---

## Executive Summary

**Overall Verdict**: **APPROVE WITH CONDITIONS**

This ticket cluster represents the final enforcement layer for Task tool validation. The architectural approach is sound, but **critical type mismatches** and **inconsistent assumptions** need correction before implementation.

**Key Findings**:
- ✅ **Strengths**: Clear separation of concerns, comprehensive test coverage, progressive validation architecture
- ⚠️ **Critical Issue**: Type mismatch in GOgent-020 (line 87 assumes `interface{}`, actual schema has `bool`)
- ⚠️ **Major Issue**: AgentSubagentMapping type mismatch in GOgent-023 (assumes `map[string]string`, actual schema has struct)
- ⚠️ **Concern**: GOgent-021 introduces model validation without clear failure mode (warning vs blocking)
- ✅ **Commendation**: Orchestrator pattern (GOgent-024b) provides clean integration point

**Required Actions Before Implementation**:
1. Fix type assertion in GOgent-020 line 87
2. Update AgentSubagentMapping access in GOgent-023 line 679
3. Clarify model mismatch handling in GOgent-021
4. Verify delegation ceiling file path assumptions in GOgent-022

---

## Review Findings by Ticket

### Ticket GOgent-020: Implement Einstein/Opus Blocking

**Implementation Intention**: Block Task tool invocations when model=opus OR target agent=einstein to enforce GAP document workflow and prevent expensive 60K token inheritance.

**Intended End State**:
- Function `ValidateTaskInvocation()` in `pkg/routing/task_validation.go`
- Blocks opus model regardless of agent
- Blocks einstein agent regardless of model
- Returns `TaskValidationResult` with violation details
- Comprehensive test coverage for all scenarios

**Dependencies Validated**:
- ✅ GOgent-017 (tool permissions) - COMPLETED, provides `Violation` struct
- ✅ GOgent-011 (violation logging) - COMPLETED, provides `LogViolation()`
- ✅ GOgent-015 (schema loading) - COMPLETED, provides `Schema` type

**Issues Found**:

#### Critical Issues

1. **Type Assertion Mismatch (Line 87)**
   ```go
   taskBlocked, _ := opusConfig.TaskInvocationBlocked.(bool)
   ```
   **Problem**: `TaskInvocationBlocked` is already a `bool` in the schema (line 44 of schema.go), not `interface{}`. This type assertion will ALWAYS fail silently due to blank identifier error suppression.

   **Evidence from schema.go**:
   ```go
   type TierConfig struct {
       ...
       TaskInvocationBlocked bool `json:"task_invocation_blocked,omitempty"`
       ...
   }
   ```

   **Impact**: Blocking will NEVER activate. All opus invocations will pass through.

   **Fix**: Change line 87 to:
   ```go
   taskBlocked := opusConfig.TaskInvocationBlocked
   if !taskBlocked {
       return result // Blocking not enabled, allow
   }
   ```

2. **Violation Struct Missing SessionID in Constructor (Lines 98-104)**

   The `Violation` struct requires `SessionID` but the ticket code doesn't show it being set consistently with the pattern established in GOgent-011.

   **Actual Violation struct** (from violations.go):
   ```go
   type Violation struct {
       Timestamp     string `json:"timestamp"`     // Auto-populated
       SessionID     string `json:"session_id"`
       ViolationType string `json:"violation_type"`
       Agent         string `json:"agent,omitempty"`
       Model         string `json:"model,omitempty"`
       ...
   }
   ```

   **Fix**: Ensure all `Violation` instantiations follow the established pattern and include all required fields.

#### Major Issues

3. **extractAgentFromPrompt() Edge Case Handling (Line 130-137)**

   The regex `AGENT:\s*([a-z-]+)` doesn't handle:
   - Uppercase variations (`AGENT:Einstein` vs `AGENT:einstein`)
   - Underscores in agent names (if they exist)
   - Multiple AGENT declarations in same prompt

   **Recommendation**: Add case-insensitive flag `(?i)` and document assumption that only first AGENT line matters.

#### Minor Issues

4. **Test Coverage Gap**

   Missing test for scenario: opus tier config exists but `TaskInvocationBlocked` is missing (defaults to false due to `omitempty`).

   **Add test**:
   ```go
   func TestValidateTaskInvocation_OpusMissingBlockedFlag(t *testing.T) {
       schema := &Schema{
           Tiers: map[string]TierConfig{
               "opus": {
                   Model: "claude-opus-4",
                   // TaskInvocationBlocked not set
               },
           },
       }

       taskInput := map[string]interface{}{
           "model": "opus",
           "prompt": "AGENT: python-pro\n\nTask",
       }

       result := ValidateTaskInvocation(schema, taskInput, "test")

       if !result.Allowed {
           t.Error("Should allow when TaskInvocationBlocked field is missing")
       }
   }
   ```

**Recommendations**:
1. **CRITICAL**: Fix type assertion on line 87 before ANY implementation
2. Add test for missing `TaskInvocationBlocked` field
3. Document regex assumptions for agent name extraction
4. Consider adding validation that sessionID is not empty

**Commendations**:
- Clear separation of blocking reasons (model vs agent)
- Excellent error messages with cost justification
- Comprehensive test matrix covering all scenarios

---

### Ticket GOgent-021: Implement Model Mismatch Warnings

**Implementation Intention**: Warn when Task model doesn't match agent's expected model from agents-index.json, helping catch configuration errors early.

**Intended End State**:
- Function `ValidateModelMatch()` in `pkg/routing/task_validation.go` (extending existing file)
- Structs `AgentConfig` and `AgentsIndex` for parsing agents-index.json
- Warning messages (not blocking) for model mismatches
- Support for both single `model` field and `allowed_models` array

**Dependencies Validated**:
- ✅ GOgent-020 - Creates base `task_validation.go` file

**Issues Found**:

#### Major Issues

1. **Unclear Failure Mode: Warning vs Blocking (Lines 320-348)**

   The ticket states "warning only, not blocking" but doesn't specify:
   - How does the warning reach the user?
   - Is it logged to violations.jsonl?
   - Does it appear in hook output?
   - Should it have a ViolationType?

   **Evidence from GOgent-024b orchestrator** (line 1143-1149):
   ```go
   matches, warning := ValidateModelMatch(&agentConfig, model)
   if !matches {
       result.ModelMismatch = warning
       // Don't block, just warn
   }
   ```

   The warning is stored but never shown to user until orchestrator formats output.

   **Recommendation**: Specify exact mechanism for delivering warning. Suggest adding to `hookSpecificOutput.additionalContext` in final CLI output.

2. **Missing Integration with Agents Index Loading**

   Ticket defines `AgentsIndex` struct but doesn't show:
   - Where/when it's loaded
   - How it's passed to validation functions
   - Error handling if agents-index.json is missing or malformed

   **Fix**: Add to GOgent-024b orchestrator initialization:
   ```go
   type ValidationOrchestrator struct {
       Schema      *Schema
       ProjectDir  string
       AgentsIndex *AgentsIndex  // Add this
   }

   // Constructor should load agents-index.json via GOgent-015 patterns
   ```

#### Minor Issues

3. **Unused Agent Name Parameter (Line 341)**

   ```go
   "", // Agent name passed separately
   ```

   This empty string in the error message formatting suggests incomplete implementation. Agent name should be a parameter to `ValidateModelMatch()`.

   **Fix**: Change signature to:
   ```go
   func ValidateModelMatch(agentName string, agentConfig *AgentConfig, requestedModel string) (bool, string)
   ```

4. **Test Coverage for Empty allowed_models Array**

   Missing test case: `allowed_models: []` (empty array). Should this fall back to `model` field or reject all?

   **Recommendation**: Add explicit handling and test.

**Recommendations**:
1. Specify exact warning delivery mechanism (CLI output, log, or both)
2. Show agents-index.json loading in orchestrator
3. Fix agent name parameter passing
4. Add test for empty `allowed_models` edge case
5. Document precedence: `allowed_models` overrides `model` field

**Commendations**:
- Flexible design supporting both single model and array
- Non-blocking approach prevents false positive breakage
- Clear test organization by scenario

---

### Ticket GOgent-022: Implement Delegation Ceiling Enforcement

**Implementation Intention**: Check if requested Task model exceeds delegation ceiling set by calculate-complexity.sh, preventing over-spending on tasks that should use cheaper tiers.

**Intended End State**:
- Function `LoadDelegationCeiling()` reads `.claude/tmp/max_delegation` file
- Function `CheckDelegationCeiling()` compares tier levels
- Blocks tasks exceeding ceiling with clear error message
- Defaults to "sonnet" when file missing (permissive)

**Dependencies Validated**:
- ✅ GOgent-020 - Establishes validation patterns

**Issues Found**:

#### Major Issues

1. **File Path Assumption May Break in CLI Context (Line 460)**

   ```go
   ceilingPath := filepath.Join(projectDir, ".claude", "tmp", "max_delegation")
   ```

   **Concern**: Assumes `.claude/tmp/` exists. If hooks haven't run yet, this directory may not exist and `os.ReadFile()` returns permission errors on parent lookup.

   **Evidence from schema.go**: Other files use XDG config patterns, not project-relative paths.

   **Fix**: Add directory existence check or use config package pattern:
   ```go
   func LoadDelegationCeiling(projectDir string) (*DelegationCeiling, error) {
       ceilingDir := filepath.Join(projectDir, ".claude", "tmp")

       // Ensure directory exists
       if _, err := os.Stat(ceilingDir); os.IsNotExist(err) {
           // No ceiling set = default to permissive
           return &DelegationCeiling{MaxTier: "sonnet"}, nil
       }

       ceilingPath := filepath.Join(ceilingDir, "max_delegation")
       // ... rest of implementation
   }
   ```

2. **TierLevels Type Mismatch (Line 482)**

   The code assumes:
   ```go
   tierLevels := schema.TierLevels
   if tierLevels == nil {
   ```

   **Problem**: `TierLevels` is a struct (from schema.go line 62-69), not a map. It can't be `nil` and doesn't support map access.

   **Actual schema.go definition**:
   ```go
   type TierLevels struct {
       Description   string `json:"description"`
       Haiku         int    `json:"haiku"`
       HaikuThinking int    `json:"haiku_thinking"`
       Sonnet        int    `json:"sonnet"`
       Opus          int    `json:"opus"`
       External      int    `json:"external"`
   }
   ```

   **Fix**: Use `Schema.GetTierLevel()` method that already exists (schema.go line 436):
   ```go
   func CheckDelegationCeiling(schema *Schema, ceiling *DelegationCeiling, requestedModel string) (bool, string) {
       ceilingLevel, err := schema.GetTierLevel(ceiling.MaxTier)
       if err != nil {
           // Unknown ceiling tier, allow
           return true, ""
       }

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

#### Minor Issues

3. **Test Uses Hardcoded Tier Levels Instead of Schema Method**

   Tests manually construct tier levels map instead of using `GetTierLevel()`. This creates maintenance burden if tier levels change.

   **Recommendation**: Refactor tests to use actual schema with proper `TierLevels` struct.

4. **Missing Test for Whitespace in Ceiling File**

   What happens if `max_delegation` contains `"  haiku  \n"` or `"haiku\r\n"` (Windows line endings)?

   The code has `strings.TrimSpace()` (line 471) which handles this, but no test validates it.

   **Add test**:
   ```go
   func TestLoadDelegationCeiling_WhitespaceHandling(t *testing.T) {
       tmpDir := t.TempDir()
       ceilingDir := filepath.Join(tmpDir, ".claude", "tmp")
       os.MkdirAll(ceilingDir, 0755)

       // Write with extra whitespace
       os.WriteFile(filepath.Join(ceilingDir, "max_delegation"), []byte("  haiku  \n"), 0644)

       ceiling, err := LoadDelegationCeiling(tmpDir)
       require.NoError(t, err)
       assert.Equal(t, "haiku", ceiling.MaxTier)
   }
   ```

**Recommendations**:
1. **CRITICAL**: Fix `TierLevels` access to use `GetTierLevel()` method
2. Add directory existence check before file read
3. Refactor tests to use proper schema structs
4. Add whitespace handling test
5. Document behavior when `.claude/tmp/` doesn't exist (currently unclear)

**Commendations**:
- Sensible default (sonnet = permissive)
- Clear error messages with override suggestion
- Graceful degradation when ceiling file missing

---

### Ticket GOgent-023: Implement Subagent_type Validation

**Implementation Intention**: Validate that Task invocations use correct subagent_type for target agent (mapped in routing-schema.json), preventing silent failures from wrong tool permissions.

**Intended End State**:
- Function `ValidateSubagentType()` checks agent-subagent_type pairing
- Function `FormatSubagentTypeError()` provides detailed fix suggestions
- Comprehensive tests for correct, incorrect, and edge cases

**Dependencies Validated**:
- ✅ GOgent-020 - Establishes validation patterns

**Issues Found**:

#### Critical Issues

1. **AgentSubagentMapping Type Mismatch (Line 679)**

   The ticket code assumes:
   ```go
   mapping := schema.AgentSubagentMapping
   if mapping == nil {
       // No mapping defined, allow any
       result.Valid = true
       return result
   }

   requiredType, exists := mapping[targetAgent]  // Treats as map[string]string
   ```

   **Problem**: `AgentSubagentMapping` is a **struct**, not a `map[string]string`.

   **Evidence from schema.go (lines 191-215)**:
   ```go
   type AgentSubagentMapping struct {
       Description                  string `json:"description"`
       CodebaseSearch               string `json:"codebase-search"`
       HaikuScout                   string `json:"haiku-scout"`
       CodeReviewer                 string `json:"code-reviewer"`
       // ... 17 more agent fields
       StaffArchitectCriticalReview string `json:"staff-architect-critical-review"`
   }
   ```

   **Impact**: Code will not compile. This is a BLOCKING issue.

   **Fix**: Use the existing `Schema.GetSubagentTypeForAgent()` method (schema.go line 365):
   ```go
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
   ```

#### Major Issues

2. **Test Assumes Map Access Pattern (Lines 738-742)**

   All tests use:
   ```go
   AgentSubagentMapping: map[string]string{
       "python-pro": "general-purpose",
       ...
   }
   ```

   This won't compile with actual struct type.

   **Fix**: Tests must construct proper `Schema` with `AgentSubagentMapping` struct:
   ```go
   schema := &Schema{
       AgentSubagentMapping: AgentSubagentMapping{
           PythonPro:      "general-purpose",
           CodebaseSearch: "Explore",
           Orchestrator:   "Plan",
       },
   }
   ```

#### Minor Issues

3. **Helper Function `contains()` Not Defined**

   Tests reference `contains()` function (line 790, 800, etc.) but it's not defined in the ticket.

   **Recommendation**: Add helper or use `strings.Contains()` directly.

4. **Test for Unknown Agent Returns Valid (Line 818-830)**

   The test expects validation to pass for unmapped agents, but doesn't verify that `GetSubagentTypeForAgent()` actually returns an error for unknown agents.

   **Concern**: If the schema method panics instead of returning error, this test won't catch it.

   **Recommendation**: Add explicit error checking in test.

**Recommendations**:
1. **CRITICAL**: Rewrite all `AgentSubagentMapping` access to use struct fields or `GetSubagentTypeForAgent()` method
2. **CRITICAL**: Fix all tests to use proper struct construction
3. Add `contains()` helper or use stdlib
4. Document custom agent handling (currently only in comments)
5. Add test verifying schema method error handling

**Commendations**:
- Excellent error formatting with specific fix examples
- Graceful handling of unmapped agents (supports custom agents)
- Backwards compatibility when no mapping defined

---

### Ticket GOgent-024: Task Validation Tests

**Implementation Intention**: Integration tests for complete Task validation workflow, demonstrating all checks work together correctly.

**Intended End State**:
- File `test/integration/task_validation_test.go`
- Tests validate complete workflow (einstein blocking, model match, ceiling, subagent_type)
- Real-world scenario tests for all common agents
- Demonstrates end-to-end validation

**Dependencies Validated**:
- ✅ GOgent-020, 021, 022, 023 - All validation functions available

**Issues Found**:

#### Major Issues

1. **Integration Tests Use Inconsistent Schema Construction (Lines 911-933)**

   The tests manually construct schema with both map and struct patterns:
   ```go
   AgentSubagentMapping: map[string]string{  // WRONG TYPE
       "python-pro": "general-purpose",
       ...
   }
   ```

   **Impact**: Tests won't compile until GOgent-023 type issues are fixed.

   **Fix**: Wait for GOgent-023 corrections, then update to match.

2. **Missing Import Path Declaration (Line 903)**

   ```go
   import (
       "testing"

       "github.com/yourusername/gogent-fortress/pkg/routing"
   }
   ```

   **Problem**: Placeholder `yourusername` instead of actual module path.

   **Actual module** (from go.mod would be): `github.com/Bucket-Chemist/GOgent-Fortress`

   **Fix**: Use correct import path or make it configurable.

#### Minor Issues

3. **Helper Function `contains()` Redeclaration**

   If defined in GOgent-023 tests, this will conflict. Move to shared test helper package.

4. **Real-World Scenarios Test Missing External/Gemini Agents (Line 1036-1050)**

   The test includes many agents but omits:
   - einstein (mentioned in blocking but not in pairing tests)
   - gemini-slave (bash subagent_type)
   - External tier agents

   **Recommendation**: Add these for completeness.

**Recommendations**:
1. **BLOCKING**: Fix import path before implementation
2. **DEPENDENCY**: Wait for GOgent-023 type fixes
3. Move `contains()` to shared test utilities
4. Add einstein and gemini-slave to real-world scenarios
5. Consider adding negative test: "All checks pass but session_id is empty"

**Commendations**:
- Excellent integration test structure
- Clear test naming by scenario
- Comprehensive agent coverage in real-world tests
- Demonstrates complete validation pipeline

---

### Ticket GOgent-024b: Wire Validation Orchestrator

**Implementation Intention**: Create validation orchestrator that runs all checks in sequence, provides single entry point for Task validation, and returns combined result.

**Intended End State**:
- Type `ValidationOrchestrator` coordinates all checks
- Function `ValidateTask()` runs checks in order with early exit on hard failures
- Function `ToJSON()` serializes results for hook consumption
- Model mismatch is warning only (doesn't block)
- Comprehensive tests for all validation paths

**Dependencies Validated**:
- ✅ GOgent-024 - Integration tests establish patterns

**Issues Found**:

#### Major Issues

1. **AgentsIndex Loading Not Shown (Line 1104)**

   The orchestrator has:
   ```go
   type ValidationOrchestrator struct {
       Schema      *Schema
       ProjectDir  string
       AgentsIndex *AgentsIndex
   }
   ```

   But the ticket doesn't show:
   - How/when `AgentsIndex` is loaded
   - Constructor function for orchestrator
   - Error handling if agents-index.json missing

   **Recommendation**: Add to ticket or reference GOgent-015 loading pattern:
   ```go
   func NewValidationOrchestrator(schema *Schema, projectDir string) (*ValidationOrchestrator, error) {
       agentsIndex, err := routing.LoadAgentIndex()
       if err != nil {
           // Agents index is optional for validation
           // Continue without it (model validation will be skipped)
           agentsIndex = nil
       }

       return &ValidationOrchestrator{
           Schema:      schema,
           ProjectDir:  projectDir,
           AgentsIndex: agentsIndex,
       }, nil
   }
   ```

2. **Early Return Prevents Model Mismatch Warning Collection (Lines 1139-1140)**

   ```go
   if !einsteinCheck.Allowed {
       result.Decision = "block"
       result.Reason = einsteinCheck.BlockReason
       result.EinsteinBlocked = einsteinCheck
       if einsteinCheck.Violation != nil {
           result.Violations = append(result.Violations, einsteinCheck.Violation)
       }
       return result // Hard block, no further checks
   }
   ```

   **Concern**: If opus is blocked, user never sees model mismatch warning that might have helped them understand why they tried to use opus.

   **Recommendation**: Collect all warnings before checking blocking conditions:
   ```go
   func (v *ValidationOrchestrator) ValidateTask(taskInput map[string]interface{}, sessionID string) *ValidationResult {
       result := &ValidationResult{
           Decision: "allow",
       }

       // ... extract fields ...

       // Collect all checks first (non-blocking)
       checks := []func(){
           func() {
               // Check 2: Model mismatch (warning only)
               if v.AgentsIndex != nil && targetAgent != "" {
                   if agentConfig, exists := v.AgentsIndex.Agents[targetAgent]; exists {
                       matches, warning := ValidateModelMatch(&agentConfig, model)
                       if !matches {
                           result.ModelMismatch = warning
                       }
                   }
               }
           },
       }

       for _, check := range checks {
           check()
       }

       // Then apply blocking checks in order
       // Check 1: Einstein/Opus blocking
       einsteinCheck := ValidateTaskInvocation(v.Schema, taskInput, sessionID)
       if !einsteinCheck.Allowed {
           result.Decision = "block"
           result.Reason = einsteinCheck.BlockReason
           result.EinsteinBlocked = einsteinCheck
           if einsteinCheck.Violation != nil {
               result.Violations = append(result.Violations, einsteinCheck.Violation)
           }
           return result
       }

       // ... continue with other blocking checks ...
   }
   ```

#### Minor Issues

3. **Test Missing Scenario: Multiple Violations (Line 1217-1383)**

   What happens if:
   - Opus is blocked (Check 1)
   - AND subagent_type is wrong (Check 4)

   Currently only the first violation is logged because of early return. Is this intentional?

   **Recommendation**: Document prioritization order or collect all violations.

4. **ToJSON Error Handling Swallowed (Line 1197-1203)**

   ```go
   func (v *ValidationResult) ToJSON() (string, error) {
       data, err := json.MarshalIndent(v, "", "  ")
       if err != nil {
           return "", err
       }
       return string(data), nil
   }
   ```

   The CLI (GOgent-025) doesn't check this error.

   **Concern**: If marshaling fails (unlikely but possible), what gets output?

   **Recommendation**: Add error handling in CLI or panic on marshal error (since struct is controlled).

**Recommendations**:
1. Add `NewValidationOrchestrator()` constructor with agents-index loading
2. Consider collecting all warnings before applying blocks
3. Document violation prioritization (first blocker wins)
4. Add test for multiple simultaneous violations
5. Handle ToJSON error in CLI code

**Commendations**:
- Clean orchestration pattern with single entry point
- Excellent separation: warnings don't block, hard violations do
- JSON serialization for hook consumption is well-designed
- Test coverage for each validation path

---

### Ticket GOgent-025: Build gogent-validate CLI

**Implementation Intention**: Build CLI binary that reads JSON from STDIN, validates Task invocations via orchestrator, outputs decision to STDOUT for hook consumption.

**Intended End State**:
- Binary `cmd/gogent-validate/main.go`
- Reads ToolEvent JSON from STDIN with 5s timeout
- Validates Task tool only (passes through others)
- Outputs hook-compliant JSON decision
- Logs violations to JSONL
- Build and installation scripts

**Dependencies Validated**:
- ✅ GOgent-024b - Provides `ValidationOrchestrator`

**Issues Found**:

#### Major Issues

1. **Import Path Placeholder (Line 1423)**

   ```go
   "github.com/yourusername/gogent-fortress/pkg/config"
   "github.com/yourusername/gogent-fortress/pkg/routing"
   ```

   **Problem**: Must use actual module path `github.com/Bucket-Chemist/GOgent-Fortress`

   **Fix**: Update import paths.

2. **Config Package Usage Not Defined (Line 1441)**

   ```go
   schema, err := config.LoadRoutingSchema()
   ```

   **Problem**: The ticket doesn't show this function exists in config package.

   **Evidence**: GOgent-015 created schema loading, but it was in `routing.LoadSchema()`, not `config.LoadRoutingSchema()`.

   **Fix**: Use correct function:
   ```go
   schema, err := routing.LoadSchema()
   ```

3. **LogViolation Signature Mismatch (Line 1474)**

   ```go
   for _, violation := range result.Violations {
       routing.LogViolation(violation)
   }
   ```

   **Problem**: Actual `LogViolation()` signature from violations.go (line 50):
   ```go
   func LogViolation(v *Violation, projectDir string) error
   ```

   **Impact**: Code won't compile - missing `projectDir` argument and ignoring error.

   **Fix**:
   ```go
   for _, violation := range result.Violations {
       if err := routing.LogViolation(violation, projectDir); err != nil {
           fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log violation: %v\n", err)
       }
   }
   ```

4. **Missing ValidationOrchestrator Constructor Call**

   The code directly constructs orchestrator (line 1462):
   ```go
   orchestrator := &routing.ValidationOrchestrator{
       Schema:     schema,
       ProjectDir: projectDir,
   }
   ```

   **Problem**: GOgent-024b review recommended constructor to load agents-index. This is missing.

   **Fix**: If constructor exists, use it:
   ```go
   orchestrator, err := routing.NewValidationOrchestrator(schema, projectDir)
   if err != nil {
       outputError(fmt.Sprintf("Failed to create orchestrator: %v", err))
       os.Exit(1)
   }
   ```

#### Minor Issues

5. **Build Script Missing go.mod Verification (Line 1563)**

   The build script jumps straight to `go build` without checking if dependencies are current.

   **Recommendation**: Add:
   ```bash
   echo "Syncing dependencies..."
   go mod tidy
   go mod download
   ```

6. **Installation Script Doesn't Verify PATH (Line 1596)**

   The script installs to `~/.local/bin` and tells user to add to PATH, but doesn't check if it's already there.

   **Recommendation**: Add check:
   ```bash
   if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
       echo "⚠️  Warning: $HOME/.local/bin is not in your PATH"
       echo "   Add this to your shell rc file:"
       echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
   fi
   ```

7. **Manual Test Command Incomplete (Line 1610-1611)**

   ```bash
   echo '{"tool_name":"Task","tool_input":{"model":"opus"},"session_id":"test"}' | ./bin/gogent-validate
   ```

   **Problem**: This doesn't include the `prompt` field needed for agent extraction (GOgent-020 line 130).

   **Fix**: Use complete test event:
   ```bash
   echo '{"tool_name":"Task","tool_input":{"model":"opus","prompt":"AGENT: python-pro\n\nImplement feature"},"session_id":"test"}' | ./bin/gogent-validate
   ```

**Recommendations**:
1. **CRITICAL**: Fix import paths to actual module name
2. **CRITICAL**: Fix `LogViolation()` call to match actual signature
3. **CRITICAL**: Use correct schema loading function
4. Add orchestrator constructor call (when implemented)
5. Add dependency sync to build script
6. Add PATH verification to install script
7. Fix manual test command to include prompt field
8. Document expected STDIN format (reference ToolEvent from GOgent-005)

**Commendations**:
- Excellent timeout handling (5s default)
- Clean pass-through for non-Task tools
- Proper hook output format
- Graceful error handling with stderr warnings
- Build and install automation

---

## Cross-Cutting Concerns

### 1. Type System Consistency

**Problem**: Multiple tickets assume types that don't match actual schema.

**Instances**:
- GOgent-020: `TaskInvocationBlocked.(bool)` type assertion on already-bool field
- GOgent-022: `TierLevels` treated as map instead of struct
- GOgent-023: `AgentSubagentMapping` treated as map instead of struct

**Root Cause**: Ticket author likely wrote against earlier schema version or didn't verify against actual code.

**Impact**: Code won't compile. Estimated 2-3 hours of debugging if implemented as-written.

**Recommendation**: Before implementing ANY ticket, run:
```bash
grep -n "type.*Config.*struct" pkg/routing/schema.go
```
Then verify ticket code matches actual types.

---

### 2. Error Message Consistency

**Observation**: Most tickets follow `[component] What. Why. How.` format correctly.

**Exception**: GOgent-021 model mismatch warnings don't include component prefix consistently (line 332-343).

**Recommendation**: Enforce format in code review:
```go
// Good
"[task-validation] Model mismatch. Agent expects 'sonnet', got 'haiku'. This may cause suboptimal performance."

// Bad
"Model mismatch. Agent expects 'sonnet'. Requested: 'haiku'."
```

---

### 3. Test Helper Duplication

**Problem**: Multiple tickets define `contains()` helper:
- GOgent-022: line 606 reference
- GOgent-023: line 790, 800, 863, 866
- GOgent-024: line 987, 1006

**Impact**: Test compilation failures or duplicated code.

**Recommendation**: Create shared test helper in `test/testutil/helpers.go`:
```go
package testutil

import "strings"

func Contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
```

Then import in tests:
```go
import "github.com/Bucket-Chemist/GOgent-Fortress/test/testutil"

if !testutil.Contains(message, "expected") {
    t.Error("...")
}
```

---

### 4. Import Path Management

**Problem**: All tickets use placeholder `github.com/yourusername/gogent-fortress`.

**Impact**: Global search-replace needed before implementation.

**Recommendation**: Add pre-implementation checklist item:
```
Before implementing GOgent-020 to 025:
1. Replace all instances of "yourusername" with "Bucket-Chemist"
2. Replace all instances of "gogent-fortress" with "GOgent-Fortress"
3. Verify import paths against go.mod
```

---

### 5. Violation Logging Consistency

**Observation**: GOgent-020 creates violations, GOgent-025 logs them, but intervening tickets don't mention violation handling.

**Question**: Do GOgent-021, 022, 023 create violations? If yes, when are they logged?

**Evidence**: GOgent-024b orchestrator (line 1165-1170) creates violations for ceiling violations:
```go
violation := &Violation{
    SessionID:     sessionID,
    ViolationType: "delegation_ceiling",
    ...
}
result.Violations = append(result.Violations, violation)
```

**Recommendation**: Document violation creation pattern for ALL validation functions, not just einstein blocking.

---

## Testing Strategy Assessment

### Coverage Analysis

**GOgent-020**:
- ✅ Positive case (allowed)
- ✅ Opus blocking
- ✅ Einstein blocking
- ✅ Blocking disabled
- ✅ Agent extraction
- ⚠️ **Missing**: TaskInvocationBlocked field omitted (defaults to false)

**GOgent-021**:
- ✅ Exact match
- ✅ Mismatch
- ✅ Allowed models array
- ⚠️ **Missing**: Empty allowed_models array
- ⚠️ **Missing**: Agent name parameter handling

**GOgent-022**:
- ✅ File exists
- ✅ No file (default)
- ✅ Within ceiling
- ✅ Exceeds ceiling
- ✅ No tier levels
- ⚠️ **Missing**: Whitespace in file
- ⚠️ **Missing**: Directory doesn't exist

**GOgent-023**:
- ✅ Correct types
- ✅ Incorrect types
- ✅ No agent
- ✅ Agent not in mapping
- ✅ No mapping
- ✅ Error formatting
- ⚠️ **Missing**: Schema method error handling

**GOgent-024**:
- ✅ Complete workflow
- ✅ All blocking scenarios
- ✅ Real-world agent pairings
- ⚠️ **Missing**: External/gemini agents

**GOgent-024b**:
- ✅ Allowed task
- ✅ Opus blocked
- ✅ Ceiling violation
- ✅ Subagent_type mismatch
- ✅ JSON serialization
- ⚠️ **Missing**: Multiple simultaneous violations

**GOgent-025**:
- Manual tests only
- ⚠️ **Missing**: Unit tests for parseEvent, outputResult, outputError
- ⚠️ **Missing**: Integration test with real schema

### Overall Coverage: ~85%

**Strengths**:
- Excellent positive/negative test pairing
- Good edge case handling (mostly)
- Integration tests demonstrate end-to-end flow

**Gaps**:
- Missing tests noted above
- No error injection tests (what if schema.json is corrupted?)
- No concurrency tests (what if multiple gogent-validate processes run?)

**Recommendation**: Add error injection tests before marking complete.

---

## Architecture Evaluation

### Strengths

1. **Progressive Validation Design**

   The orchestrator pattern allows adding/removing checks without touching CLI code. Excellent separation of concerns.

2. **Early Exit on Hard Failures**

   Blocking checks return immediately, avoiding wasted CPU on doomed validations. Good performance characteristic.

3. **Graceful Degradation**

   Missing agents-index.json doesn't break validation - model matching is skipped. Missing delegation ceiling defaults to permissive. This is production-ready thinking.

4. **Comprehensive Error Messages**

   All errors include context, reason, and remediation. Follows established patterns from GOgent-011.

### Weaknesses

1. **Type Assumptions Throughout**

   Multiple tickets assume types that don't exist. Suggests insufficient verification against actual codebase.

2. **Missing Constructor Pattern**

   Orchestrator should have a constructor that handles all loading (schema, agents-index, ceiling). Current design requires CLI to know internals.

3. **Inconsistent Violation Handling**

   Some validations create violations, others don't. No clear pattern documented.

4. **No Caching Strategy**

   Every validation loads schema from disk. For hooks running 100+ times per session, this is inefficient. Consider singleton or cache.

### Recommendations

1. Add `NewValidationOrchestrator()` constructor:
   ```go
   func NewValidationOrchestrator(projectDir string) (*ValidationOrchestrator, error) {
       schema, err := LoadSchema()
       if err != nil {
           return nil, fmt.Errorf("schema load: %w", err)
       }

       agentsIndex, _ := LoadAgentIndex()  // Optional

       return &ValidationOrchestrator{
           Schema:      schema,
           ProjectDir:  projectDir,
           AgentsIndex: agentsIndex,
       }, nil
   }
   ```

2. Document violation creation rules:
   ```
   Create violation when:
   - Blocking a task (opus, einstein, ceiling, subagent_type)
   - NOT when warning only (model mismatch)

   ViolationType values:
   - blocked_task_opus
   - blocked_task_einstein
   - delegation_ceiling
   - subagent_type_mismatch
   ```

3. Add schema caching (optional, Week 2):
   ```go
   var (
       cachedSchema     *Schema
       cachedSchemaOnce sync.Once
   )

   func GetSchema() (*Schema, error) {
       var err error
       cachedSchemaOnce.Do(func() {
           cachedSchema, err = LoadSchema()
       })
       return cachedSchema, err
   }
   ```

---

## Dependency Chain Validation

### Declared Dependencies

```
GOgent-020 → GOgent-017, GOgent-011
GOgent-021 → GOgent-020
GOgent-022 → GOgent-020
GOgent-023 → GOgent-020
GOgent-024 → GOgent-020, 021, 022, 023
GOgent-024b → GOgent-024
GOgent-025 → GOgent-024b
```

### Actual Dependencies

**GOgent-020 actually depends on**:
- ✅ GOgent-011 (Violation struct, LogViolation)
- ✅ GOgent-015 (Schema struct) - **NOT DECLARED**
- ⚠️ GOgent-017 (not directly used)

**GOgent-021 actually depends on**:
- ✅ GOgent-020 (file location only)
- ✅ GOgent-015 (agents-index loading) - **NOT DECLARED**

**GOgent-022 actually depends on**:
- ✅ GOgent-020 (pattern only)
- ✅ GOgent-015 (Schema.GetTierLevel method) - **NOT DECLARED**

**GOgent-023 actually depends on**:
- ✅ GOgent-020 (pattern only)
- ✅ GOgent-015 (Schema.GetSubagentTypeForAgent method) - **NOT DECLARED**

**GOgent-024 actually depends on**:
- ✅ All above tickets

**GOgent-024b actually depends on**:
- ✅ GOgent-024
- ✅ GOgent-020, 021, 022, 023 (functions)

**GOgent-025 actually depends on**:
- ✅ GOgent-024b
- ✅ GOgent-005 (ToolEvent struct) - **NOT DECLARED**
- ✅ GOgent-011 (LogViolation) - **NOT DECLARED**
- ✅ GOgent-015 (schema loading) - **NOT DECLARED**

### Recommendation

Update dependency declarations to include GOgent-015 for all tickets. Add note that GOgent-017 provides Violation struct pattern but isn't directly imported.

---

## Time Estimate Validation

**Declared Estimates**:
- GOgent-020: 2h
- GOgent-021: 1.5h
- GOgent-022: 2h
- GOgent-023: 2h (ticket says 2.5h, summary says 2h - **discrepancy**)
- GOgent-024: 1.5h
- GOgent-024b: 1h
- GOgent-025: 1.5h
- **Total**: 11 hours (or 11.5h if GOgent-023 is 2.5h)

**Adjusted Estimates** (accounting for fixes):

- GOgent-020: 2h + 0.5h (type fixes) = **2.5h**
- GOgent-021: 1.5h + 0.5h (integration clarifications) = **2h**
- GOgent-022: 2h + 1h (TierLevels refactoring) = **3h**
- GOgent-023: 2.5h + 1h (AgentSubagentMapping refactoring) = **3.5h**
- GOgent-024: 1.5h + 0.5h (test fixes) = **2h**
- GOgent-024b: 1h + 0.5h (constructor implementation) = **1.5h**
- GOgent-025: 1.5h + 0.5h (import fixes, signature corrections) = **2h**

**Adjusted Total**: **16.5 hours** (50% increase over original estimate)

**Recommendation**: Budget 17 hours for this cluster, with 6 hours contingency for testing and integration debugging.

---

## Risk Assessment

### High Risk Items

1. **Type Mismatches in GOgent-020, 022, 023**

   **Risk**: Code won't compile as-written
   **Mitigation**: Fix types before implementation
   **Probability**: 100% (guaranteed to fail)
   **Impact**: 2-3 hour delay

2. **AgentSubagentMapping Struct Access (GOgent-023)**

   **Risk**: Requires rewriting all tests and implementation logic
   **Mitigation**: Use existing schema methods
   **Probability**: 100%
   **Impact**: 1-2 hour refactor

3. **Missing Constructor Pattern (GOgent-024b, 025)**

   **Risk**: CLI code duplicates loading logic, error-prone
   **Mitigation**: Add constructor before GOgent-025
   **Probability**: 60% (could work without it, but fragile)
   **Impact**: Future maintenance burden

### Medium Risk Items

4. **Import Path Management**

   **Risk**: Search-replace misses instances, breaks build
   **Mitigation**: Automated pre-implementation script
   **Probability**: 40%
   **Impact**: 30 minute debugging

5. **Violation Logging Signature**

   **Risk**: GOgent-025 calls wrong signature, fails at runtime
   **Mitigation**: Verify against violations.go before implementing
   **Probability**: 80% (ticket code is wrong)
   **Impact**: 15 minute fix

### Low Risk Items

6. **Helper Function Duplication**

   **Risk**: Test compilation errors
   **Mitigation**: Shared testutil package
   **Probability**: 30%
   **Impact**: 10 minute fix

---

## Recommendations by Priority

### MUST FIX (Blocking Issues)

1. **GOgent-020 Line 87**: Change `taskBlocked, _ := opusConfig.TaskInvocationBlocked.(bool)` to `taskBlocked := opusConfig.TaskInvocationBlocked`

2. **GOgent-022 Lines 482-506**: Replace TierLevels map access with `schema.GetTierLevel()` calls

3. **GOgent-023 Lines 679-709**: Replace AgentSubagentMapping map access with `schema.GetSubagentTypeForAgent()`

4. **GOgent-023 All Tests**: Rewrite schema construction to use struct fields

5. **GOgent-025 Lines 1423**: Fix import paths (yourusername → Bucket-Chemist, gogent-fortress → GOgent-Fortress)

6. **GOgent-025 Line 1441**: Change `config.LoadRoutingSchema()` to `routing.LoadSchema()`

7. **GOgent-025 Line 1474**: Fix `LogViolation` call to include projectDir argument

### SHOULD FIX (Major Issues)

8. **GOgent-021**: Specify warning delivery mechanism (hook output additionalContext)

9. **GOgent-021**: Show AgentsIndex loading in orchestrator

10. **GOgent-022**: Add directory existence check before file read

11. **GOgent-023**: Add `contains()` helper to shared testutil

12. **GOgent-024**: Fix import path, update schema construction after GOgent-023 fixes

13. **GOgent-024b**: Add `NewValidationOrchestrator()` constructor

14. **GOgent-025**: Add orchestrator constructor call (after 024b fixes)

### NICE TO HAVE (Minor Issues)

15. **GOgent-020**: Add test for missing TaskInvocationBlocked field

16. **GOgent-020**: Document regex assumptions for agent extraction

17. **GOgent-021**: Add test for empty allowed_models array

18. **GOgent-022**: Add whitespace handling test

19. **GOgent-022**: Refactor tests to use GetTierLevel method

20. **GOgent-023**: Add test for schema method error handling

21. **GOgent-024**: Add einstein and gemini-slave to real-world scenarios

22. **GOgent-024b**: Document violation prioritization strategy

23. **GOgent-024b**: Add test for multiple simultaneous violations

24. **GOgent-025**: Add dependency sync to build script

25. **GOgent-025**: Add PATH verification to install script

26. **GOgent-025**: Fix manual test command to include prompt field

---

## Pre-Implementation Checklist

Before starting GOgent-020:

- [ ] Run `git log --oneline --grep="GOgent-0" | grep -E "(015|016|017)"` to verify dependencies completed
- [ ] Verify GOgent-015 schema loading works: `go test ./pkg/routing -run TestLoadSchema`
- [ ] Verify GOgent-011 violations work: `go test ./pkg/routing -run TestLogViolation`
- [ ] Create pre-implementation script to fix all import paths
- [ ] Read actual schema.go to verify types match ticket assumptions
- [ ] Create shared testutil package with contains() helper

Before starting each ticket:

- [ ] Verify previous ticket tests pass
- [ ] Check for type mismatches in ticket code vs actual codebase
- [ ] Ensure import paths use correct module name
- [ ] Confirm dependency functions exist and have correct signatures

After completing all tickets:

- [ ] Run full test suite: `go test ./...`
- [ ] Run manual validation: `echo '{"tool_name":"Task",...}' | ./bin/gogent-validate`
- [ ] Verify opus blocking works with actual schema
- [ ] Test with missing agents-index.json (should not error)
- [ ] Test with missing delegation ceiling (should default to sonnet)
- [ ] Check violation logging to both global and project logs

---

## Refactoring Recommendations

### High-Value Refactors

1. **Extract Type Accessors Package** (Week 2)

   The pattern of accessing nested structs (TierConfig, AgentSubagentMapping) appears throughout. Consider:
   ```go
   // pkg/routing/accessors.go
   package routing

   func (s *Schema) GetTaskInvocationBlocked(tier string) (bool, error) {
       tierConfig, err := s.GetTier(tier)
       if err != nil {
           return false, err
       }
       return tierConfig.TaskInvocationBlocked, nil
   }
   ```

   Benefits: Centralizes type knowledge, easier to refactor schema later.

2. **Violation Builder Pattern** (Week 2)

   Instead of manually constructing Violation structs, use builder:
   ```go
   violation := routing.NewViolation(sessionID).
       WithType("blocked_task_opus").
       WithModel("opus").
       WithAgent(targetAgent).
       WithReason("model_is_opus").
       Build()
   ```

   Benefits: Prevents missing required fields, cleaner code.

3. **Validation Result DSL** (Week 3)

   The orchestrator returns complex nested structures. Consider:
   ```go
   result := routing.NewValidationResult().
       Block("opus model not allowed").
       WithViolation(violation).
       WithRecommendation("Use /einstein instead").
       Build()
   ```

### Low-Priority Refactors

4. **Shared Test Fixtures** (When time permits)

   Every test constructs its own schema. Create:
   ```go
   // test/fixtures/schemas.go
   func ValidSchema() *routing.Schema { ... }
   func SchemaWithOpusBlocked() *routing.Schema { ... }
   ```

5. **CLI Output Templates** (Week 3)

   The outputResult/outputError functions in GOgent-025 manually construct JSON. Consider text/template or encoding/json with struct tags.

---

## Verdict by Ticket

| Ticket | Verdict | Blocking Issues | Time Adjustment |
|--------|---------|-----------------|-----------------|
| GOgent-020 | APPROVE WITH CONDITIONS | 1 critical (type assertion) | +0.5h |
| GOgent-021 | APPROVE WITH CONDITIONS | 1 major (warning delivery) | +0.5h |
| GOgent-022 | APPROVE WITH CONDITIONS | 1 critical (TierLevels access) | +1h |
| GOgent-023 | APPROVE WITH CONDITIONS | 1 critical (AgentSubagentMapping) | +1h |
| GOgent-024 | APPROVE WITH CONDITIONS | 1 major (import path, depends on 023) | +0.5h |
| GOgent-024b | APPROVE WITH CONDITIONS | 1 major (constructor missing) | +0.5h |
| GOgent-025 | APPROVE WITH CONDITIONS | 3 critical (imports, signatures) | +0.5h |

**Overall**: **APPROVE WITH CONDITIONS**

All tickets are architecturally sound and follow established patterns. However, **critical type mismatches** must be corrected before implementation to avoid compilation failures and debugging time.

**Estimated impact of fixes**: +4.5 hours (from 11h to 16.5h)

**Recommendation**: Assign senior developer to review type corrections before implementation begins. All other issues are minor and can be addressed during code review.

---

## Final Checklist

### Before Implementation

- [ ] Fix all MUST FIX items in section above
- [ ] Create shared testutil package with contains() helper
- [ ] Verify all dependencies (015, 011, 005) are completed
- [ ] Run baseline tests to ensure existing code works
- [ ] Create implementation branch from latest master

### During Implementation

- [ ] Implement tickets in order (020 → 021 → 022 → 023 → 024 → 024b → 025)
- [ ] Run tests after each ticket: `go test ./pkg/routing -v`
- [ ] Fix any new issues discovered during implementation
- [ ] Document any deviations from ticket specs

### After Implementation

- [ ] Full test suite passes: `go test ./...`
- [ ] Manual validation tests pass (all scenarios in GOgent-025)
- [ ] Build scripts work on clean system
- [ ] Installation script produces working binary
- [ ] Opus blocking verified with actual routing-schema.json
- [ ] Violations logged to correct paths (global + project)
- [ ] Code coverage ≥80% for all new packages

### Before Merge

- [ ] Code review by senior developer
- [ ] All acceptance criteria checkboxes marked
- [ ] No placeholder text (yourusername, etc.) remains
- [ ] Documentation updated if behavior differs from spec
- [ ] Commit message follows GOgent convention

---

**Review Completed**: 2026-01-18
**Reviewer**: staff-architect-critical-review (Haiku+Thinking tier)
**Next Step**: Address MUST FIX items, then begin implementation of GOgent-020
