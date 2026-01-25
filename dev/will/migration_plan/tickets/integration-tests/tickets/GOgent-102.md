---
id: GOgent-102
title: SessionStart Integration Tests
description: Integration tests for gogent-load-context hook
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-094"]
priority: medium
week: 5
tags: ["integration-tests", "week-5", "session-start"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-102: SessionStart Integration Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-094 (test harness)

**Task**:
Test complete gogent-load-context (SessionStart) hook workflow using corpus events, verifying language detection, convention loading, handoff injection, and error recovery.

**File**: `test/integration/session_start_test.go`

**Imports**:
```go
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/config"
)
```

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/config"
)

// TestSessionStart_Integration runs SessionStart with all fixture scenarios
func TestSessionStart_Integration(t *testing.T) {
	// Setup: Build binary if not present
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found. Run: go build -o cmd/gogent-load-context/gogent-load-context cmd/gogent-load-context/main.go")
	}

	// Setup: Create test corpus
	corpusPath := filepath.Join(t.TempDir(), "session-start-corpus.jsonl")
	createSessionStartCorpus(t, corpusPath)

	// Setup: Create test project directory with routing schema and conventions
	projectDir := t.TempDir()
	setupTestSessionStartEnvironment(t, projectDir)

	// Create harness
	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run all SessionStart events through gogent-load-context hook
	results, err := harness.RunHookBatch(binaryPath, "SessionStart")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No results returned")
	}

	// Verify results
	for i, result := range results {
		if result.Error != nil {
			t.Errorf("Hook execution error at index %d: %v", i, result.Error)
			continue
		}

		// All hooks must return valid JSON
		if result.ParsedJSON == nil {
			t.Errorf("Expected JSON output at index %d, got: %s", i, result.Stdout)
			continue
		}

		// Verify required fields present
		if _, ok := result.ParsedJSON["status"]; !ok {
			t.Errorf("Missing 'status' field at index %d in output: %v", i, result.ParsedJSON)
		}

		if _, ok := result.ParsedJSON["context_loaded"]; !ok {
			t.Errorf("Missing 'context_loaded' field at index %d in output: %v", i, result.ParsedJSON)
		}
	}

	// Print summary
	PrintSummary(results)
}

// TestSessionStart_LanguageDetection verifies Python/Go/R language detection
func TestSessionStart_LanguageDetection(t *testing.T) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found")
	}

	testCases := []struct {
		name           string
		setupDir       func(string)
		expectedLang   string
	}{
		{
			name: "Python project",
			setupDir: func(projectDir string) {
				// Create Python project indicators
				os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte("[tool.poetry]\n"), 0644)
			},
			expectedLang: "Python",
		},
		{
			name: "Go project",
			setupDir: func(projectDir string) {
				// Create Go project indicators
				os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module github.com/test/project\n"), 0644)
			},
			expectedLang: "Go",
		},
		{
			name: "R project",
			setupDir: func(projectDir string) {
				// Create R project indicators
				os.WriteFile(filepath.Join(projectDir, "DESCRIPTION"), []byte("Package: testpkg\n"), 0644)
			},
			expectedLang: "R",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			setupTestSessionStartEnvironment(t, projectDir)
			tc.setupDir(projectDir)

			// Create minimal SessionStart event
			eventJSON := fmt.Sprintf(`{
				"hook_event_name": "SessionStart",
				"session_id": "test-lang-%s",
				"project_dir": "%s"
			}`, strings.ToLower(tc.expectedLang), projectDir)

			tmpCorpus := filepath.Join(t.TempDir(), "lang-corpus.jsonl")
			os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

			harness, _ := NewTestHarness(tmpCorpus, projectDir)
			harness.LoadCorpus()

			result := harness.RunHook(binaryPath, harness.Events[0])

			// Verify language detected
			if result.ParsedJSON == nil {
				t.Fatalf("Expected JSON output, got: %s", result.Stdout)
			}

			lang, ok := result.ParsedJSON["detected_language"].(string)
			if !ok {
				t.Errorf("Missing or non-string detected_language field")
				return
			}

			if lang != tc.expectedLang {
				t.Errorf("Expected language %s, got: %s", tc.expectedLang, lang)
			}
		})
	}
}

// TestSessionStart_ConventionLoading verifies conventions are injected into context
func TestSessionStart_ConventionLoading(t *testing.T) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found")
	}

	projectDir := t.TempDir()
	setupTestSessionStartEnvironment(t, projectDir)

	// Create Go project
	os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module github.com/test/project\n"), 0644)

	// Create custom Go conventions
	claudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	conventions := `# Go Conventions
- Use error wrapping with %w
- Test naming: TestFunctionName_Scenario
`
	convPath := filepath.Join(claudeDir, "conventions", "go.md")
	os.MkdirAll(filepath.Dir(convPath), 0755)
	os.WriteFile(convPath, []byte(conventions), 0644)

	// Create SessionStart event
	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "test-conventions",
		"project_dir": "%s"
	}`, projectDir)

	tmpCorpus := filepath.Join(t.TempDir(), "conventions-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify conventions loaded
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	conventions_loaded, ok := result.ParsedJSON["conventions_loaded"].(bool)
	if !ok || !conventions_loaded {
		t.Errorf("Expected conventions_loaded=true, got: %v", result.ParsedJSON["conventions_loaded"])
	}

	// Verify context contains convention content
	context_data, ok := result.ParsedJSON["context"].(string)
	if !ok {
		t.Errorf("Expected string context field, got: %v", result.ParsedJSON["context"])
		return
	}

	if !strings.Contains(context_data, "Go Conventions") {
		t.Errorf("Expected conventions content in context, got: %s", context_data)
	}
}

// TestSessionStart_HandoffInjection verifies last-handoff.md is injected if present
func TestSessionStart_HandoffInjection(t *testing.T) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found")
	}

	projectDir := t.TempDir()
	setupTestSessionStartEnvironment(t, projectDir)

	// Create Go project
	os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module github.com/test/project\n"), 0644)

	// Create previous session handoff
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	handoff := `# Last Handoff

## Sharp Edges Found
- Error handling in config loader needs timeout

## Decisions Made
- Use sync.Once for singleton initialization

## Next Steps
- Test handoff injection
`
	os.WriteFile(filepath.Join(memoryDir, "last-handoff.md"), []byte(handoff), 0644)

	// Create SessionStart event
	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "test-handoff",
		"project_dir": "%s"
	}`, projectDir)

	tmpCorpus := filepath.Join(t.TempDir(), "handoff-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify handoff injected
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	handoff_injected, ok := result.ParsedJSON["handoff_injected"].(bool)
	if !ok || !handoff_injected {
		t.Errorf("Expected handoff_injected=true, got: %v", result.ParsedJSON["handoff_injected"])
	}

	// Verify context contains handoff content
	context_data, ok := result.ParsedJSON["context"].(string)
	if !ok {
		t.Errorf("Expected string context field, got: %v", result.ParsedJSON["context"])
		return
	}

	if !strings.Contains(context_data, "Last Handoff") {
		t.Errorf("Expected handoff content in context, got: %s", context_data)
	}

	if !strings.Contains(context_data, "Sharp Edges Found") {
		t.Errorf("Expected sharp edges in handoff context, got: %s", context_data)
	}
}

// TestSessionStart_RoutingSchemaLoad verifies routing schema is loaded and parsed
func TestSessionStart_RoutingSchemaLoad(t *testing.T) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found")
	}

	projectDir := t.TempDir()
	setupTestSessionStartEnvironment(t, projectDir)

	// Create custom routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	schema := `{
		"version": "1.0",
		"tiers": {
			"haiku": {
				"tools_allowed": ["Read", "Glob", "Grep"],
				"task_invocation_allowed": true
			},
			"sonnet": {
				"tools_allowed": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"],
				"task_invocation_allowed": true
			}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"tech-docs-writer": "general-purpose"
		}
	}`
	os.WriteFile(schemaPath, []byte(schema), 0644)

	// Create SessionStart event
	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "test-routing-schema",
		"project_dir": "%s"
	}`, projectDir)

	tmpCorpus := filepath.Join(t.TempDir(), "routing-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify routing schema loaded
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	schema_loaded, ok := result.ParsedJSON["routing_schema_loaded"].(bool)
	if !ok || !schema_loaded {
		t.Errorf("Expected routing_schema_loaded=true, got: %v", result.ParsedJSON["routing_schema_loaded"])
	}

	// Verify schema version parsed
	schema_version, ok := result.ParsedJSON["schema_version"].(string)
	if !ok || schema_version == "" {
		t.Errorf("Expected schema_version in output, got: %v", result.ParsedJSON["schema_version"])
	}

	if schema_version != "1.0" {
		t.Errorf("Expected schema_version 1.0, got: %s", schema_version)
	}

	// Verify tiers parsed
	tiers, ok := result.ParsedJSON["tiers_available"].([]interface{})
	if !ok {
		t.Errorf("Expected tiers_available array, got: %v", result.ParsedJSON["tiers_available"])
		return
	}

	if len(tiers) < 2 {
		t.Errorf("Expected at least 2 tiers, got: %d", len(tiers))
	}
}

// TestSessionStart_ErrorRecovery tests graceful fallback on errors
func TestSessionStart_ErrorRecovery(t *testing.T) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-load-context binary not found")
	}

	testCases := []struct {
		name          string
		setupDir      func(string)
		expectSuccess bool
		expectedField string
	}{
		{
			name: "Missing project directory",
			setupDir: func(projectDir string) {
				// Don't create anything
			},
			expectSuccess: true, // Should gracefully handle missing dir
			expectedField: "warning",
		},
		{
			name: "Invalid routing schema JSON",
			setupDir: func(projectDir string) {
				schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
				os.MkdirAll(filepath.Dir(schemaPath), 0755)
				os.WriteFile(schemaPath, []byte("{invalid json"), 0644)
			},
			expectSuccess: true, // Should gracefully degrade
			expectedField: "error",
		},
		{
			name: "Empty conventions directory",
			setupDir: func(projectDir string) {
				os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\n"), 0644)
				claudeDir := filepath.Join(projectDir, ".claude", "conventions")
				os.MkdirAll(claudeDir, 0755)
				// Create empty conventions dir
			},
			expectSuccess: true, // Should handle empty conventions
			expectedField: "status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			setupTestSessionStartEnvironment(t, projectDir)
			tc.setupDir(projectDir)

			// Create SessionStart event
			eventJSON := fmt.Sprintf(`{
				"hook_event_name": "SessionStart",
				"session_id": "test-error-%s",
				"project_dir": "%s"
			}`, strings.ToLower(strings.ReplaceAll(tc.name, " ", "-")), projectDir)

			tmpCorpus := filepath.Join(t.TempDir(), "error-corpus.jsonl")
			os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

			harness, _ := NewTestHarness(tmpCorpus, projectDir)
			harness.LoadCorpus()

			result := harness.RunHook(binaryPath, harness.Events[0])

			// Verify graceful handling
			if result.ParsedJSON == nil {
				t.Fatalf("Expected JSON output even on error, got: %s", result.Stdout)
			}

			// Verify hook returns valid status
			status, ok := result.ParsedJSON["status"].(string)
			if !ok || status == "" {
				t.Errorf("Expected status field, got: %v", result.ParsedJSON["status"])
			}

			// Verify error recovery field present if error expected
			if !tc.expectSuccess {
				if _, ok := result.ParsedJSON[tc.expectedField]; !ok {
					t.Errorf("Expected %s field in output on error", tc.expectedField)
				}
			}
		})
	}
}

// Helper: Create corpus with various SessionStart scenarios
func createSessionStartCorpus(t *testing.T, path string) {
	events := []string{
		// Python project
		`{"hook_event_name":"SessionStart","session_id":"test-python","project_dir":"/tmp/python-proj","detected_language":"Python"}`,

		// Go project
		`{"hook_event_name":"SessionStart","session_id":"test-go","project_dir":"/tmp/go-proj","detected_language":"Go"}`,

		// R project
		`{"hook_event_name":"SessionStart","session_id":"test-r","project_dir":"/tmp/r-proj","detected_language":"R"}`,

		// Home directory (no language)
		`{"hook_event_name":"SessionStart","session_id":"test-home","project_dir":"/home/user","detected_language":""}`,

		// With routing schema
		`{"hook_event_name":"SessionStart","session_id":"test-routing","project_dir":"/tmp/routing-proj","detected_language":"Go","has_routing_schema":true}`,
	}

	content := strings.Join(events, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create corpus: %v", err)
	}
}

// Helper: Setup test SessionStart environment with all fixtures
func setupTestSessionStartEnvironment(t *testing.T, projectDir string) {
	// Create .claude directory structure
	claudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create conventions directory
	convDir := filepath.Join(claudeDir, "conventions")
	os.MkdirAll(convDir, 0755)

	// Create memory directory
	memDir := filepath.Join(claudeDir, "memory")
	os.MkdirAll(memDir, 0755)

	// Create routing schema
	schemaPath := filepath.Join(claudeDir, "routing-schema.json")
	schema := `{
		"version": "1.0",
		"tiers": {
			"haiku": {
				"tools_allowed": ["Read", "Glob", "Grep"],
				"task_invocation_allowed": true
			},
			"sonnet": {
				"tools_allowed": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"],
				"task_invocation_allowed": true
			},
			"opus": {
				"tools_allowed": ["*"],
				"task_invocation_blocked": true
			}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"tech-docs-writer": "general-purpose",
			"python-pro": "general-purpose",
			"go-pro": "general-purpose",
			"r-pro": "general-purpose"
		}
	}`
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to create routing schema: %v", err)
	}

	// Create Python conventions
	pythonConv := `# Python Conventions
- Use type hints for all functions
- Test naming: test_function_name_scenario
- Use pytest for testing
`
	if err := os.WriteFile(filepath.Join(convDir, "python.md"), []byte(pythonConv), 0644); err != nil {
		t.Fatalf("Failed to create python conventions: %v", err)
	}

	// Create Go conventions
	goConv := `# Go Conventions
- Use error wrapping with %w
- Test naming: TestFunctionName_Scenario
- Use go test for testing
`
	if err := os.WriteFile(filepath.Join(convDir, "go.md"), []byte(goConv), 0644); err != nil {
		t.Fatalf("Failed to create go conventions: %v", err)
	}

	// Create R conventions
	rConv := `# R Conventions
- Use <- for assignment
- Test naming: test_function_name
- Use testthat for testing
`
	if err := os.WriteFile(filepath.Join(convDir, "R.md"), []byte(rConv), 0644); err != nil {
		t.Fatalf("Failed to create r conventions: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestSessionStart_Integration` runs all SessionStart events from corpus and verifies JSON output with status and context_loaded fields
- [ ] `TestSessionStart_LanguageDetection` verifies Python/Go/R detection and returns correct detected_language field
- [ ] `TestSessionStart_ConventionLoading` verifies conventions are loaded and injected into context for Go projects
- [ ] `TestSessionStart_HandoffInjection` verifies last-handoff.md content is injected when present
- [ ] `TestSessionStart_RoutingSchemaLoad` verifies routing schema is parsed and schema_version/tiers_available fields returned
- [ ] `TestSessionStart_ErrorRecovery` gracefully handles missing directories, invalid JSON, and empty convention directories
- [ ] All hooks return valid JSON with status field
- [ ] Integration tests pass: `go test ./test/integration -v -run TestSessionStart`
- [ ] Test coverage ≥80% for SessionStart workflow

**Why This Matters**: SessionStart hook is entry point for every agent session. Untested at integration level, this ticket verifies language detection, convention loading, routing schema injection, and error recovery work correctly with real corpus events and multi-language project detection. Critical for session initialization reliability.

---
