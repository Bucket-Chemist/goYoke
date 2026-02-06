package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// =============================================================================
// Convention Injection Integration Tests
// =============================================================================

// setupTestEnvironment creates a test environment with schema, agents-index,
// and convention files. Returns the temp directory path and cleanup function.
func setupTestEnvironment(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir := t.TempDir()

	// Setup .claude directory structure
	claudeDir := filepath.Join(tmpDir, ".claude")
	agentsDir := filepath.Join(claudeDir, "agents")
	conventionsDir := filepath.Join(claudeDir, "conventions")
	rulesDir := filepath.Join(claudeDir, "rules")

	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(conventionsDir, 0755)
	os.MkdirAll(rulesDir, 0755)

	// Create minimal routing schema
	schema := `{
		"version": "2.5.0",
		"tiers": {
			"haiku": {"model": "haiku"},
			"sonnet": {"model": "sonnet"}
		},
		"delegation_ceiling": {"default": "sonnet"},
		"agent_subagent_mapping": {
			"go-pro": "general-purpose",
			"codebase-search": "Explore",
			"python-pro": "general-purpose"
		},
		"escalation_rules": {}
	}`
	os.WriteFile(filepath.Join(claudeDir, "routing-schema.json"), []byte(schema), 0644)

	// Create agents-index.json with context_requirements
	agentsIndex := `{
		"version": "2.5.0",
		"generated_at": "2026-02-05T00:00:00Z",
		"agents": [
			{
				"id": "go-pro",
				"name": "Go Implementation Specialist",
				"model": "sonnet",
				"thinking": false,
				"tier": 2,
				"category": "implementation",
				"path": "prompts/agents/tier-2-sonnet/go-pro.md",
				"subagent_type": "general-purpose",
				"triggers": ["Go implementation"],
				"tools": ["Read", "Write", "Edit", "Bash"],
				"context_requirements": {
					"rules": ["agent-guidelines.md"],
					"conventions": {
						"base": ["go.md"]
					}
				},
				"description": "Go implementation specialist"
			},
			{
				"id": "codebase-search",
				"name": "Codebase Search",
				"model": "haiku",
				"thinking": false,
				"tier": 1,
				"category": "exploration",
				"path": "prompts/agents/tier-1-haiku/codebase-search.md",
				"subagent_type": "Explore",
				"triggers": ["find", "search"],
				"tools": ["Grep", "Glob"],
				"description": "Fast codebase search"
			},
			{
				"id": "python-pro",
				"name": "Python Implementation Specialist",
				"model": "sonnet",
				"thinking": false,
				"tier": 2,
				"category": "implementation",
				"path": "prompts/agents/tier-2-sonnet/python-pro.md",
				"subagent_type": "general-purpose",
				"triggers": ["Python implementation"],
				"tools": ["Read", "Write", "Edit", "Bash"],
				"context_requirements": {
					"rules": ["agent-guidelines.md"],
					"conventions": {
						"base": ["python.md"],
						"conditional": [
							{
								"pattern": "*/data/*",
								"convention": "python-datasci.md"
							},
							{
								"pattern": "*/models/*",
								"convention": "python-ml.md"
							}
						]
					}
				},
				"description": "Python implementation specialist"
			}
		],
		"routing_rules": {
			"intent_gate": {
				"description": "Pre-classification",
				"types": []
			},
			"scout_first_protocol": {
				"description": "Scout protocol",
				"triggers": [],
				"skip_when": [],
				"primary": "haiku-scout",
				"fallback": "codebase-search",
				"output": ".claude/tmp/scout_metrics.json"
			},
			"complexity_routing": {
				"description": "Complexity-based routing",
				"calculator": "gogent-scout",
				"thresholds": {},
				"force_external_if": "score > 1000"
			},
			"auto_fire": {},
			"model_tiers": {
				"haiku": ["codebase-search"],
				"sonnet": ["go-pro", "python-pro"]
			}
		},
		"state_management": {
			"description": "State passing",
			"tmp_directory": ".claude/tmp",
			"files": {},
			"cleanup": {
				"trigger": "SessionEnd",
				"action": "archive"
			}
		}
	}`
	os.WriteFile(filepath.Join(agentsDir, "agents-index.json"), []byte(agentsIndex), 0644)

	// Create convention files
	goConvention := `# Go Conventions

## Code Style
- Use gofmt for formatting
- Follow effective Go guidelines

## Naming
- Exported names start with capital letters
- Use camelCase for variables`
	os.WriteFile(filepath.Join(conventionsDir, "go.md"), []byte(goConvention), 0644)

	pythonConvention := `# Python Conventions

## Code Style
- Use PEP 8 style guide
- Type hints for function signatures

## Imports
- Standard library first
- Third-party packages second`
	os.WriteFile(filepath.Join(conventionsDir, "python.md"), []byte(pythonConvention), 0644)

	pythonDatasciConvention := `# Python Data Science Conventions

## VST Transforms
- Use pyOpenMS for mass spec data
- Baseline correction before analysis`
	os.WriteFile(filepath.Join(conventionsDir, "python-datasci.md"), []byte(pythonDatasciConvention), 0644)

	pythonMLConvention := `# Python ML Conventions

## PyTorch Patterns
- Use nn.Module for models
- Implement forward() method`
	os.WriteFile(filepath.Join(conventionsDir, "python-ml.md"), []byte(pythonMLConvention), 0644)

	// Create rules file
	agentGuidelines := `# Agent Guidelines

## Behavior
- Always validate input
- Provide clear error messages

## Output Format
- Use structured responses
- Include relevant context`
	os.WriteFile(filepath.Join(rulesDir, "agent-guidelines.md"), []byte(agentGuidelines), 0644)

	// Set environment to use test directory
	oldHome := os.Getenv("HOME")
	oldClaudeConfig := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("HOME", tmpDir)
	os.Setenv("CLAUDE_CONFIG_DIR", claudeDir)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
		if oldClaudeConfig != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldClaudeConfig)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
		routing.ClearConventionCache()
	}

	return tmpDir, cleanup
}

// TestConventionInjection_GoProAgent tests convention injection for go-pro agent
func TestConventionInjection_GoProAgent(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load schema
	schema, err := routing.LoadSchema()
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Create Task event with AGENT: go-pro
	toolInput := map[string]interface{}{
		"prompt":        "AGENT: go-pro\n\nTASK: Implement authentication handler",
		"model":         "sonnet",
		"subagent_type": "general-purpose",
		"description":   "Implement Go feature",
	}

	// Validate and check for convention injection
	orchestrator := routing.NewValidationOrchestrator(schema, tmpDir, nil, nil)
	result := orchestrator.ValidateTask(toolInput, "test-session")

	if result.Decision == "block" {
		t.Fatalf("Validation blocked: %s", result.Reason)
	}

	// Parse the task input to get the agent
	taskInput, err := routing.ParseTaskInput(toolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Load agent config
	agentConfig, err := loadAgentConfig("go-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	if agentConfig == nil || agentConfig.ContextRequirements == nil {
		t.Fatal("Expected agent config with context requirements")
	}

	// Build augmented prompt
	taskFiles := routing.ExtractFilesFromPrompt(taskInput.Prompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		taskInput.Prompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Verify conventions were injected
	if !strings.Contains(augmentedPrompt, routing.ConventionsMarker) {
		t.Error("Conventions marker not found in augmented prompt")
	}

	if !strings.Contains(augmentedPrompt, "go.md") {
		t.Error("go.md convention not referenced in augmented prompt")
	}

	if !strings.Contains(augmentedPrompt, "agent-guidelines.md") {
		t.Error("agent-guidelines.md not referenced in augmented prompt")
	}

	if !strings.Contains(augmentedPrompt, "AGENT: go-pro") {
		t.Error("Original prompt content missing")
	}

	if !strings.Contains(augmentedPrompt, "Use gofmt for formatting") {
		t.Error("Go convention content not found in augmented prompt")
	}
}

// TestConventionInjection_NoRequirements tests agents without context requirements
func TestConventionInjection_NoRequirements(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load schema
	schema, err := routing.LoadSchema()
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Create Task event with AGENT: codebase-search (no context_requirements)
	toolInput := map[string]interface{}{
		"prompt":        "AGENT: codebase-search\n\nTASK: Find all Go files",
		"model":         "haiku",
		"subagent_type": "Explore",
		"description":   "Search files",
	}

	// Validate
	orchestrator := routing.NewValidationOrchestrator(schema, tmpDir, nil, nil)
	result := orchestrator.ValidateTask(toolInput, "test-session")

	if result.Decision == "block" {
		t.Fatalf("Validation blocked: %s", result.Reason)
	}

	// Parse task input
	taskInput, err := routing.ParseTaskInput(toolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Load agent config
	agentConfig, err := loadAgentConfig("codebase-search")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Agent has no context requirements, so prompt should be unchanged
	if agentConfig != nil && agentConfig.ContextRequirements != nil {
		if agentConfig.ContextRequirements.HasContextRequirements() {
			t.Error("codebase-search should not have context requirements")
		}
	}

	// Verify prompt is unchanged
	taskFiles := routing.ExtractFilesFromPrompt(taskInput.Prompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		taskInput.Prompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	if augmentedPrompt != taskInput.Prompt {
		t.Error("Prompt should be unchanged for agent without context requirements")
	}

	if strings.Contains(augmentedPrompt, routing.ConventionsMarker) {
		t.Error("No conventions should be injected for agent without requirements")
	}
}

// TestConventionInjection_UnknownAgent tests graceful handling of unknown agents
func TestConventionInjection_UnknownAgent(t *testing.T) {
	testDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load schema
	schema, err := routing.LoadSchema()
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Create Task event with unknown agent
	toolInput := map[string]interface{}{
		"prompt":        "AGENT: nonexistent-agent\n\nTASK: Do something",
		"model":         "sonnet",
		"subagent_type": "general-purpose",
		"description":   "Unknown agent",
	}

	// Validate - should not crash
	orchestrator := routing.NewValidationOrchestrator(schema, testDir, nil, nil)
	result := orchestrator.ValidateTask(toolInput, "test-session")

	// May or may not block depending on validation rules, but shouldn't crash
	t.Logf("Validation result for unknown agent: %s", result.Decision)

	// Load agent config for unknown agent
	agentConfig, err := loadAgentConfig("nonexistent-agent")
	if err != nil {
		t.Fatalf("Unexpected error loading unknown agent: %v", err)
	}

	if agentConfig != nil {
		t.Error("Expected nil config for unknown agent")
	}
}

// TestConventionInjection_AlreadyAugmented tests prevention of double-injection
func TestConventionInjection_AlreadyAugmented(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a prompt that already contains the conventions marker
	existingPrompt := routing.ConventionsMarker + `

--- go.md ---
# Go Conventions (already injected)

` + routing.ConventionsEndMarker + `

---

AGENT: go-pro

TASK: Implement feature (already augmented)`

	// Load agent config
	agentConfig, err := loadAgentConfig("go-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Attempt to augment already-augmented prompt
	taskFiles := routing.ExtractFilesFromPrompt(existingPrompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		existingPrompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Should return original prompt unchanged
	if augmentedPrompt != existingPrompt {
		t.Error("Already-augmented prompt should be returned unchanged")
	}

	// Verify marker appears only once
	markerCount := strings.Count(augmentedPrompt, routing.ConventionsMarker)
	if markerCount != 1 {
		t.Errorf("Expected conventions marker to appear once, found %d times", markerCount)
	}
}

// TestConventionInjection_PythonWithConditional tests conditional convention matching
func TestConventionInjection_PythonWithConditional(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create Task event mentioning a file in /data/ directory
	toolInput := map[string]interface{}{
		"prompt":        "AGENT: python-pro\n\nTASK: Process mass spec data in src/data/loader.py",
		"model":         "sonnet",
		"subagent_type": "general-purpose",
		"description":   "Python data processing",
	}

	// Parse task input
	taskInput, err := routing.ParseTaskInput(toolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Load agent config
	agentConfig, err := loadAgentConfig("python-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Extract files and build augmented prompt
	taskFiles := routing.ExtractFilesFromPrompt(taskInput.Prompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		taskInput.Prompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Verify base convention is included
	if !strings.Contains(augmentedPrompt, "python.md") {
		t.Error("Expected python.md base convention to be referenced")
	}

	if !strings.Contains(augmentedPrompt, "PEP 8 style guide") {
		t.Error("Expected python.md content in augmented prompt")
	}

	// Verify conditional convention is included (data pattern matched)
	if !strings.Contains(augmentedPrompt, "python-datasci.md") {
		t.Error("Expected python-datasci.md conditional convention to be referenced")
	}

	if !strings.Contains(augmentedPrompt, "pyOpenMS") {
		t.Error("Expected python-datasci.md content in augmented prompt")
	}

	// Verify ML convention is NOT included (no /models/ path)
	if strings.Contains(augmentedPrompt, "python-ml.md") {
		t.Error("python-ml.md should not be included (no /models/ path in prompt)")
	}
}

// TestConventionInjection_PreservesTaskFields tests that all Task fields are preserved
func TestConventionInjection_PreservesTaskFields(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load schema
	schema, err := routing.LoadSchema()
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Create Task with all optional fields
	toolInput := map[string]interface{}{
		"prompt":            "AGENT: go-pro\n\nTASK: Implement feature",
		"model":             "sonnet",
		"subagent_type":     "general-purpose",
		"description":       "Test task with all fields",
		"max_turns":         float64(5),
		"run_in_background": true,
	}

	// Validate
	orchestrator := routing.NewValidationOrchestrator(schema, tmpDir, nil, nil)
	result := orchestrator.ValidateTask(toolInput, "test-session")

	if result.Decision == "block" {
		t.Fatalf("Validation blocked: %s", result.Reason)
	}

	// Parse task input
	taskInput, err := routing.ParseTaskInput(toolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Load agent config and augment
	agentConfig, err := loadAgentConfig("go-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	taskFiles := routing.ExtractFilesFromPrompt(taskInput.Prompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		taskInput.Prompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Build updatedInput as main.go would
	updatedInput := map[string]interface{}{
		"prompt":        augmentedPrompt,
		"model":         taskInput.Model,
		"subagent_type": taskInput.SubagentType,
		"description":   taskInput.Description,
	}

	if taskInput.MaxTurns > 0 {
		updatedInput["max_turns"] = taskInput.MaxTurns
	}
	if taskInput.RunInBackground {
		updatedInput["run_in_background"] = taskInput.RunInBackground
	}

	// Verify all fields are present
	if updatedInput["model"] != "sonnet" {
		t.Errorf("Expected model 'sonnet', got: %v", updatedInput["model"])
	}

	if updatedInput["subagent_type"] != "general-purpose" {
		t.Errorf("Expected subagent_type 'general-purpose', got: %v", updatedInput["subagent_type"])
	}

	if updatedInput["description"] != "Test task with all fields" {
		t.Errorf("Expected description preserved, got: %v", updatedInput["description"])
	}

	// MaxTurns will be int in updatedInput (from TaskInput struct)
	if maxTurns, ok := updatedInput["max_turns"].(int); !ok || maxTurns != 5 {
		t.Errorf("Expected max_turns 5 (int), got: %v (type: %T)", updatedInput["max_turns"], updatedInput["max_turns"])
	}

	if updatedInput["run_in_background"] != true {
		t.Errorf("Expected run_in_background true, got: %v", updatedInput["run_in_background"])
	}

	// Verify prompt was augmented
	promptStr, ok := updatedInput["prompt"].(string)
	if !ok {
		t.Fatal("Prompt is not a string")
	}

	if !strings.Contains(promptStr, routing.ConventionsMarker) {
		t.Error("Prompt should be augmented with conventions")
	}
}

// TestConventionInjection_ResponseFormat tests the modify response structure
func TestConventionInjection_ResponseFormat(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load agent config
	agentConfig, err := loadAgentConfig("go-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Create and augment prompt
	originalPrompt := "AGENT: go-pro\n\nTASK: Implement feature"
	taskFiles := routing.ExtractFilesFromPrompt(originalPrompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		originalPrompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Create modify response as main.go does
	updatedInput := map[string]interface{}{
		"prompt":        augmentedPrompt,
		"model":         "sonnet",
		"subagent_type": "general-purpose",
		"description":   "Test task",
	}

	resp := routing.NewModifyResponse("PreToolUse", updatedInput)

	// Verify response structure
	if resp.Decision != "" {
		t.Errorf("Modify response should not have decision, got: %s", resp.Decision)
	}

	if resp.HookSpecificOutput == nil {
		t.Fatal("Expected hookSpecificOutput in modify response")
	}

	if resp.HookSpecificOutput["hookEventName"] != "PreToolUse" {
		t.Errorf("Expected hookEventName 'PreToolUse', got: %v", resp.HookSpecificOutput["hookEventName"])
	}

	if !resp.HasUpdatedInput() {
		t.Error("Expected response to have updatedInput")
	}

	updatedInputFromResp := resp.GetUpdatedInput()
	if updatedInputFromResp == nil {
		t.Fatal("Expected updatedInput to be retrievable")
	}

	if updatedInputFromResp["model"] != "sonnet" {
		t.Errorf("Expected model 'sonnet' in updatedInput, got: %v", updatedInputFromResp["model"])
	}

	// Verify response marshals to valid JSON
	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Parse back to verify structure
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatalf("Failed to parse marshaled response: %v", err)
	}

	hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput in parsed JSON")
	}

	if hookOutput["hookEventName"] != "PreToolUse" {
		t.Errorf("Expected hookEventName in parsed JSON, got: %v", hookOutput["hookEventName"])
	}
}

// TestConventionInjection_PythonMLConditional tests ML convention conditional matching
func TestConventionInjection_PythonMLConditional(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create Task mentioning a file in /models/ directory
	originalPrompt := "AGENT: python-pro\n\nTASK: Implement neural network in src/models/transformer.py"

	// Load agent config
	agentConfig, err := loadAgentConfig("python-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Extract files and build augmented prompt
	taskFiles := routing.ExtractFilesFromPrompt(originalPrompt)
	t.Logf("Extracted files: %v", taskFiles)

	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		originalPrompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Verify base convention
	if !strings.Contains(augmentedPrompt, "python.md") {
		t.Error("Expected python.md base convention")
	}

	// Verify ML convention is included (models pattern matched)
	if !strings.Contains(augmentedPrompt, "python-ml.md") {
		t.Error("Expected python-ml.md conditional convention for /models/ path")
	}

	if !strings.Contains(augmentedPrompt, "nn.Module") {
		t.Error("Expected python-ml.md content in augmented prompt")
	}

	// Verify data science convention is NOT included (no /data/ path)
	if strings.Contains(augmentedPrompt, "python-datasci.md") {
		t.Error("python-datasci.md should not be included (no /data/ path in prompt)")
	}
}

// TestConventionInjection_StripConventions tests convention removal
func TestConventionInjection_StripConventions(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Load agent config
	agentConfig, err := loadAgentConfig("go-pro")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Create and augment prompt
	originalPrompt := "AGENT: go-pro\n\nTASK: Implement authentication"
	taskFiles := routing.ExtractFilesFromPrompt(originalPrompt)
	augmentedPrompt, err := routing.BuildAugmentedPrompt(
		originalPrompt,
		agentConfig.ContextRequirements,
		taskFiles,
	)

	if err != nil {
		t.Fatalf("Failed to build augmented prompt: %v", err)
	}

	// Strip conventions
	strippedPrompt := routing.StripConventionsFromPrompt(augmentedPrompt)

	// Verify conventions were removed
	if strings.Contains(strippedPrompt, routing.ConventionsMarker) {
		t.Error("Conventions marker should be removed")
	}

	if strings.Contains(strippedPrompt, "go.md") {
		t.Error("Convention file references should be removed")
	}

	if strings.Contains(strippedPrompt, "Use gofmt") {
		t.Error("Convention content should be removed")
	}

	// Verify original prompt content is preserved
	if !strings.Contains(strippedPrompt, "AGENT: go-pro") {
		t.Error("Original prompt content should be preserved")
	}

	if !strings.Contains(strippedPrompt, "TASK: Implement authentication") {
		t.Error("Original task description should be preserved")
	}
}
