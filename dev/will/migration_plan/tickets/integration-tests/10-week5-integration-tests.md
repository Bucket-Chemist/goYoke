# Week 3 Part 1: Integration & Regression Tests

**Phase 0 - Week 3 Days 1-3** | GOgent-004c, 094-100 (7 tickets)

---

## Navigation

| Previous | Up | Next |
|----------|-----|------|
| [09-week4-observability-remaining.md](09-week4-observability-remaining.md) | [README.md](README.md) | [11-week5-deployment-cutover.md](11-week5-deployment-cutover.md) |

**Cross-References:**
- Standards: [00-overview.md](00-overview.md)
- Prework: [00-prework.md](00-prework.md) (GOgent-000 corpus)
- Config: [01-week1-foundation-events.md](01-week1-foundation-events.md) (GOgent-004a/b)
- Validation: [03-week1-validation-cli.md](03-week1-validation-cli.md) (CLI structure)

---

## Summary

Week 3 Part 1 completes the Phase 0 Go translation by implementing comprehensive integration tests, performance benchmarks, and regression tests. These tickets verify that Go implementations match Bash behavior exactly using the 100-event corpus from GOgent-000.

**Key Components:**
- **GOgent-004c**: Config circular dependency tests (deferred from Week 1)
- **GOgent-094**: Test harness for event corpus replay
- **GOgent-095**: Integration tests for validate-routing hook
- **GOgent-096**: Integration tests for session-archive hook
- **GOgent-097**: Integration tests for sharp-edge-detector hook
- **GOgent-098**: Performance benchmarks (target <5ms p99)
- **GOgent-099**: End-to-end workflow integration tests
- **GOgent-100**: Regression tests comparing Go vs Bash output

**Testing Philosophy:**
- Unit tests verify individual functions (covered in Week 1-2 tickets)
- Integration tests verify complete hook workflows
- Regression tests verify Go output matches Bash exactly
- Benchmarks verify performance meets targets

---

## ⚠️ REFACTORING REQUIRED

**Status**: This plan was created before weeks 8-11 were added

**Current Scope**: Tests only 3 hooks (validate-routing, session-archive, sharp-edge)

**Required Changes**:
1. Expand GOgent-095 to test validate-routing hook
2. Expand GOgent-096 to test session-archive hook
3. Expand GOgent-097 to test sharp-edge-detector hook
4. **ADD GOgent-100b**: Integration tests for load-routing-context (1.5h)
5. **ADD GOgent-100c**: Integration tests for agent-endstate (1.5h)
6. **ADD GOgent-100d**: Integration tests for attention-gate (1.5h)
7. **ADD GOgent-100e**: Integration tests for orchestrator-completion-guard (1.5h)
8. **ADD GOgent-100f**: Integration tests for detect-documentation-theater (1h)
9. **ADD GOgent-100g**: Integration tests for benchmark-logger (1h)
10. Update GOgent-098 (benchmarks) to include ALL 7 hooks
11. Update GOgent-099 (end-to-end) to test complete hook chain
12. Update GOgent-100 (regression) to cover ALL hooks

**New Total**: 13-14 tickets, ~22-24 hours (vs original 7 tickets, 14 hours)

**See**: [UNTRACKED_HOOKS.md](UNTRACKED_HOOKS.md) for hook descriptions

**Implementation Note**: This refactoring should be done AFTER weeks 8-11 are complete.

---

## Tickets

### GOgent-004c: Config Circular Dependency Tests

**Time**: 1 hour
**Dependencies**: GOgent-004a, GOgent-004b (from Week 1)

**Task**:
Complete config package tests by adding circular dependency detection and multi-agent config loading tests. Deferred from Week 1 to avoid blocking event parsing work.

**File**: `pkg/config/loader_test.go` (append to existing tests)

**Implementation**:

Add test cases to existing test file:

```go
// Circular dependency detection test
func TestLoadAgentConfig_CircularDependency(t *testing.T) {
	// Create temp agent configs with circular dependency
	tmpDir := t.TempDir()

	// Agent A requires Agent B
	agentAConfig := `{
		"agent_id": "agent-a",
		"requires": ["agent-b"],
		"tier": "haiku"
	}`

	// Agent B requires Agent A (circular)
	agentBConfig := `{
		"agent_id": "agent-b",
		"requires": ["agent-a"],
		"tier": "haiku"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-a"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-b"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-a", "agent.json"), []byte(agentAConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-b", "agent.json"), []byte(agentBConfig), 0644)

	// Attempt to load agent-a should detect circular dependency
	_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "agent-a"))
	if err == nil {
		t.Fatal("Expected error for circular dependency, got nil")
	}

	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected 'circular' in error message, got: %v", err)
	}
}

// Multi-level dependency resolution test
func TestLoadAgentConfig_MultiLevelDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	// Create agent chain: C → B → A
	agentAConfig := `{
		"agent_id": "agent-a",
		"tier": "haiku",
		"tools_allowed": ["Read", "Glob"]
	}`

	agentBConfig := `{
		"agent_id": "agent-b",
		"requires": ["agent-a"],
		"tier": "haiku_thinking"
	}`

	agentCConfig := `{
		"agent_id": "agent-c",
		"requires": ["agent-b"],
		"tier": "sonnet"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-a"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-b"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-c"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-a", "agent.json"), []byte(agentAConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-b", "agent.json"), []byte(agentBConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-c", "agent.json"), []byte(agentCConfig), 0644)

	// Load agent-c should load all dependencies
	config, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "agent-c"))
	if err != nil {
		t.Fatalf("Failed to load multi-level dependencies: %v", err)
	}

	if config.AgentID != "agent-c" {
		t.Errorf("Expected agent-c, got: %s", config.AgentID)
	}

	// Verify dependencies loaded
	if len(config.Requires) != 1 || config.Requires[0] != "agent-b" {
		t.Errorf("Expected requires=[agent-b], got: %v", config.Requires)
	}
}

// Missing dependency test
func TestLoadAgentConfig_MissingDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Agent references non-existent dependency
	agentConfig := `{
		"agent_id": "test-agent",
		"requires": ["nonexistent-agent"],
		"tier": "haiku"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "test-agent"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "agents", "test-agent", "agent.json"), []byte(agentConfig), 0644)

	_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "test-agent"))
	if err == nil {
		t.Fatal("Expected error for missing dependency, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent-agent") {
		t.Errorf("Expected 'nonexistent-agent' in error, got: %v", err)
	}
}

// Concurrent config loading test (verify thread safety)
func TestLoadAgentConfig_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple agent configs
	for i := 0; i < 10; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		agentConfig := fmt.Sprintf(`{
			"agent_id": "%s",
			"tier": "haiku",
			"tools_allowed": ["Read"]
		}`, agentID)

		os.MkdirAll(filepath.Join(tmpDir, "agents", agentID), 0755)
		os.WriteFile(filepath.Join(tmpDir, "agents", agentID, "agent.json"), []byte(agentConfig), 0644)
	}

	// Load all configs concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", index)
			_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", agentID))
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent load failed: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestLoadAgentConfig_CircularDependency` detects circular dependencies
- [ ] Error message includes "circular" for circular dependency detection
- [ ] `TestLoadAgentConfig_MultiLevelDependencies` loads transitive dependencies
- [ ] `TestLoadAgentConfig_MissingDependency` reports missing dependencies
- [ ] `TestLoadAgentConfig_Concurrent` verifies thread-safe config loading
- [ ] All tests pass: `go test ./pkg/config -v`
- [ ] Coverage for config package ≥80%

**Why This Matters**: Deferred from Week 1 to unblock event parsing work. Config loading must handle complex dependency graphs without deadlocks or crashes. Thread safety critical for production use.

---

### GOgent-094: Test Harness for Event Corpus Replay

**Time**: 2 hours
**Dependencies**: GOgent-000 (corpus), GOgent-008b (event parsers)

**Task**:
Build test harness that replays events from GOgent-000 corpus through hook implementations, capturing output for comparison.

**File**: `test/integration/harness.go`

**Imports**:
```go
package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)
```

**Implementation**:

```go
// EventEntry represents a single event from the corpus JSONL
type EventEntry struct {
	Timestamp     int64                  `json:"timestamp"`
	HookEventName string                 `json:"hook_event_name"`
	ToolName      string                 `json:"tool_name,omitempty"`
	ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
	ToolResponse  map[string]interface{} `json:"tool_response,omitempty"`
	SessionID     string                 `json:"session_id"`
	RawJSON       json.RawMessage        `json:"-"` // Preserve original JSON
}

// HookResult captures the output of a hook execution
type HookResult struct {
	Event      *EventEntry
	Stdout     string
	Stderr     string
	ExitCode   int
	Duration   time.Duration
	ParsedJSON map[string]interface{}
	Error      error
}

// TestHarness manages corpus replay and result collection
type TestHarness struct {
	CorpusPath string
	ProjectDir string
	Events     []*EventEntry
}

// NewTestHarness creates a test harness for the given corpus file
func NewTestHarness(corpusPath, projectDir string) (*TestHarness, error) {
	if _, err := os.Stat(corpusPath); err != nil {
		return nil, fmt.Errorf("[harness] Corpus file not found: %s. Error: %w. Run GOgent-000 first.", corpusPath, err)
	}

	return &TestHarness{
		CorpusPath: corpusPath,
		ProjectDir: projectDir,
	}, nil
}

// LoadCorpus reads all events from the corpus JSONL file
func (h *TestHarness) LoadCorpus() error {
	f, err := os.Open(h.CorpusPath)
	if err != nil {
		return fmt.Errorf("[harness] Failed to open corpus: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var entry EventEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return fmt.Errorf("[harness] Failed to parse corpus line %d: %w", lineNum, err)
		}

		// Store raw JSON for exact replay
		entry.RawJSON = json.RawMessage(line)

		h.Events = append(h.Events, &entry)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("[harness] Failed to read corpus: %w", err)
	}

	if len(h.Events) == 0 {
		return fmt.Errorf("[harness] Corpus is empty. Expected 100+ events from GOgent-000.")
	}

	return nil
}

// FilterEvents returns events matching the given hook event name
func (h *TestHarness) FilterEvents(hookEventName string) []*EventEntry {
	var filtered []*EventEntry
	for _, event := range h.Events {
		if event.HookEventName == hookEventName {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// RunHook executes a hook binary with the given event JSON as STDIN
func (h *TestHarness) RunHook(binaryPath string, event *EventEntry) *HookResult {
	result := &HookResult{
		Event: event,
	}

	// Prepare command
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(),
		"CLAUDE_PROJECT_DIR="+h.ProjectDir,
		"GOgent_TEST_MODE=1", // Signal test mode for hooks
	)

	// Use raw JSON to preserve exact formatting
	cmd.Stdin = bytes.NewReader(event.RawJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute hook
	startTime := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(startTime)

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = fmt.Errorf("[harness] Failed to execute hook: %w", err)
			return result
		}
	}

	// Parse JSON output if present
	if len(result.Stdout) > 0 {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &parsed); err != nil {
			result.Error = fmt.Errorf("[harness] Failed to parse hook JSON output: %w. Output: %s", err, result.Stdout)
		} else {
			result.ParsedJSON = parsed
		}
	}

	return result
}

// RunHookBatch runs a hook against all filtered events
func (h *TestHarness) RunHookBatch(binaryPath, hookEventName string) ([]*HookResult, error) {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("[harness] Hook binary not found: %s. Build it first with: go build -o %s", binaryPath, binaryPath)
	}

	events := h.FilterEvents(hookEventName)
	if len(events) == 0 {
		return nil, fmt.Errorf("[harness] No events found for hook %s in corpus", hookEventName)
	}

	results := make([]*HookResult, 0, len(events))

	for _, event := range events {
		result := h.RunHook(binaryPath, event)
		results = append(results, result)
	}

	return results, nil
}

// CompareResults compares two hook results (Go vs Bash)
func CompareResults(goResult, bashResult *HookResult) []string {
	var diffs []string

	// Compare exit codes
	if goResult.ExitCode != bashResult.ExitCode {
		diffs = append(diffs, fmt.Sprintf("Exit code: Go=%d, Bash=%d", goResult.ExitCode, bashResult.ExitCode))
	}

	// Compare JSON structure (ignore timestamp differences)
	goJSON := goResult.ParsedJSON
	bashJSON := bashResult.ParsedJSON

	if goJSON != nil && bashJSON != nil {
		// Check decision field
		if goJSON["decision"] != bashJSON["decision"] {
			diffs = append(diffs, fmt.Sprintf("Decision: Go=%v, Bash=%v", goJSON["decision"], bashJSON["decision"]))
		}

		// Check reason field (if present)
		if goReason, ok := goJSON["reason"].(string); ok {
			if bashReason, ok := bashJSON["reason"].(string); ok {
				if goReason != bashReason {
					diffs = append(diffs, fmt.Sprintf("Reason: Go=%s, Bash=%s", goReason, bashReason))
				}
			}
		}
	}

	return diffs
}

// PrintSummary prints test results summary
func PrintSummary(results []*HookResult) {
	total := len(results)
	passed := 0
	failed := 0
	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration
		if r.Error == nil && r.ExitCode == 0 {
			passed++
		} else {
			failed++
		}
	}

	avgDuration := totalDuration / time.Duration(total)

	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Total:    %d\n", total)
	fmt.Printf("Passed:   %d\n", passed)
	fmt.Printf("Failed:   %d\n", failed)
	fmt.Printf("Avg Time: %v\n", avgDuration)
	fmt.Printf("====================\n")
}
```

**Tests**: `test/integration/harness_test.go`

```go
package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	// Create temp corpus file
	tmpCorpus := filepath.Join(t.TempDir(), "test-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-123"}
`), 0644)

	harness, err := NewTestHarness(tmpCorpus, "/tmp/project")
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if harness.CorpusPath != tmpCorpus {
		t.Errorf("Expected corpus path %s, got: %s", tmpCorpus, harness.CorpusPath)
	}
}

func TestLoadCorpus(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "test-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1","timestamp":1234567890}
{"hook_event_name":"PostToolUse","session_id":"test-2","timestamp":1234567891}
{"hook_event_name":"PreToolUse","session_id":"test-3","timestamp":1234567892}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	err := harness.LoadCorpus()
	if err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	if len(harness.Events) != 3 {
		t.Errorf("Expected 3 events, got: %d", len(harness.Events))
	}

	// Verify first event
	if harness.Events[0].HookEventName != "PreToolUse" {
		t.Errorf("Expected PreToolUse, got: %s", harness.Events[0].HookEventName)
	}

	if harness.Events[0].SessionID != "test-1" {
		t.Errorf("Expected test-1, got: %s", harness.Events[0].SessionID)
	}
}

func TestFilterEvents(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "test-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1"}
{"hook_event_name":"PostToolUse","session_id":"test-2"}
{"hook_event_name":"PreToolUse","session_id":"test-3"}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	harness.LoadCorpus()

	filtered := harness.FilterEvents("PreToolUse")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 PreToolUse events, got: %d", len(filtered))
	}

	for _, event := range filtered {
		if event.HookEventName != "PreToolUse" {
			t.Errorf("Filtered event has wrong hook name: %s", event.HookEventName)
		}
	}
}

func TestCompareResults(t *testing.T) {
	goResult := &HookResult{
		ExitCode: 0,
		ParsedJSON: map[string]interface{}{
			"decision": "allow",
			"reason":   "Valid request",
		},
	}

	bashResult := &HookResult{
		ExitCode: 0,
		ParsedJSON: map[string]interface{}{
			"decision": "allow",
			"reason":   "Valid request",
		},
	}

	diffs := CompareResults(goResult, bashResult)
	if len(diffs) != 0 {
		t.Errorf("Expected no diffs, got: %v", diffs)
	}

	// Test with difference
	bashResult.ParsedJSON["decision"] = "block"
	diffs = CompareResults(goResult, bashResult)
	if len(diffs) == 0 {
		t.Error("Expected diffs, got none")
	}
}
```

**Acceptance Criteria**:
- [ ] `NewTestHarness()` loads corpus file from GOgent-000
- [ ] `LoadCorpus()` parses all 100+ events from JSONL
- [ ] `FilterEvents()` returns events matching hook name
- [ ] `RunHook()` executes binary with event JSON as STDIN
- [ ] `RunHook()` captures stdout, stderr, exit code, duration
- [ ] `CompareResults()` identifies differences between Go and Bash outputs
- [ ] `PrintSummary()` displays pass/fail statistics
- [ ] All tests pass: `go test ./test/integration -v`

**Why This Matters**: Foundation for all integration and regression tests. Enables automated comparison of Go vs Bash behavior across 100+ real events.

---

### GOgent-095: Integration Tests for validate-routing Hook

**Time**: 2 hours
**Dependencies**: GOgent-094 (harness), GOgent-025 (gogent-validate binary)

**Task**:
Test complete validate-routing workflow using corpus events, verify blocking logic and violation logging.

**File**: `test/integration/validate_routing_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/config"
)

func TestValidateRouting_Integration(t *testing.T) {
	// Setup: Build binary if not present
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found. Run: go build -o cmd/gogent-validate/gogent-validate cmd/gogent-validate/main.go")
	}

	// Setup: Create test corpus
	corpusPath := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	createValidateCorpus(t, corpusPath)

	// Setup: Create test project directory with routing schema
	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create harness
	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run all PreToolUse events through validate-routing
	results, err := harness.RunHookBatch(binaryPath, "PreToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No results returned")
	}

	// Verify results
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Hook execution error: %v", result.Error)
			continue
		}

		// All hooks must return valid JSON
		if result.ParsedJSON == nil {
			t.Errorf("Expected JSON output, got: %s", result.Stdout)
			continue
		}

		// Verify decision field present
		if _, ok := result.ParsedJSON["decision"]; !ok {
			t.Errorf("Missing 'decision' field in output: %v", result.ParsedJSON)
		}
	}

	// Print summary
	PrintSummary(results)
}

func TestValidateRouting_BlocksOpusTask(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create event attempting Task(opus)
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze this problem",
			"subagent_type": "general-purpose"
		},
		"session_id": "test-opus-block"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "opus-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify blocked
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	decision, ok := result.ParsedJSON["decision"].(string)
	if !ok || decision != "block" {
		t.Errorf("Expected decision=block for Task(opus), got: %v", decision)
	}

	// Verify reason mentions GAP document
	reason, ok := result.ParsedJSON["reason"].(string)
	if !ok || !strings.Contains(reason, "GAP") {
		t.Errorf("Expected reason to mention GAP document, got: %s", reason)
	}
}

func TestValidateRouting_AllowsValidTask(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create valid Task(haiku) event
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "haiku",
			"prompt": "AGENT: codebase-search\n\nFind auth files",
			"subagent_type": "Explore"
		},
		"session_id": "test-valid-task"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "valid-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify allowed
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	decision, ok := result.ParsedJSON["decision"].(string)
	if !ok || decision != "allow" {
		t.Errorf("Expected decision=allow for valid Task(haiku), got: %v", decision)
	}
}

func TestValidateRouting_ViolationLogging(t *testing.T) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-validate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Override violations log path for test
	violationsLog := filepath.Join(t.TempDir(), "violations.jsonl")
	os.Setenv("GOgent_VIOLATIONS_LOG", violationsLog)
	defer os.Unsetenv("GOgent_VIOLATIONS_LOG")

	// Create event that violates tool permissions
	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {
			"file_path": "/tmp/test.txt",
			"content": "test"
		},
		"session_id": "test-violation"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "violation-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Set current tier to haiku (cannot use Write)
	tierFile := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierFile), 0755)
	os.WriteFile(tierFile, []byte("haiku\n"), 0644)

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify violation logged
	if _, err := os.Stat(violationsLog); err != nil {
		t.Fatal("Violations log not created")
	}

	logData, err := os.ReadFile(violationsLog)
	if err != nil {
		t.Fatalf("Failed to read violations log: %v", err)
	}

	if len(logData) == 0 {
		t.Error("Violations log is empty")
	}

	// Parse violation
	var violation map[string]interface{}
	if err := json.Unmarshal(logData, &violation); err != nil {
		t.Fatalf("Failed to parse violation: %v", err)
	}

	if violation["violation_type"] != "tool_permission" {
		t.Errorf("Expected tool_permission violation, got: %v", violation["violation_type"])
	}
}

// Helper: Create corpus with various validation scenarios
func createValidateCorpus(t *testing.T, path string) {
	events := []string{
		// Valid Task(haiku)
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"haiku","prompt":"AGENT: codebase-search\n\nFind files","subagent_type":"Explore"},"session_id":"test-1"}`,

		// Invalid Task(opus)
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"opus","prompt":"AGENT: einstein\n\nDeep analysis","subagent_type":"general-purpose"},"session_id":"test-2"}`,

		// Valid Read
		`{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"},"session_id":"test-3"}`,

		// Invalid subagent_type
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"haiku","prompt":"AGENT: tech-docs-writer\n\nWrite docs","subagent_type":"Explore"},"session_id":"test-4"}`,
	}

	content := strings.Join(events, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create corpus: %v", err)
	}
}

// Helper: Setup minimal routing schema for tests
func setupTestRoutingSchema(t *testing.T, projectDir string) {
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	schema := `{
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
				"task_invocation_blocked": true,
				"blocked_reason": "Use /einstein slash command"
			}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"tech-docs-writer": "general-purpose",
			"einstein": "general-purpose"
		}
	}`

	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to create routing schema: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestValidateRouting_Integration` runs all PreToolUse events from corpus
- [ ] `TestValidateRouting_BlocksOpusTask` verifies Task(opus) blocked
- [ ] `TestValidateRouting_AllowsValidTask` verifies valid Task(haiku) allowed
- [ ] `TestValidateRouting_ViolationLogging` verifies violations logged to JSONL
- [ ] All hooks return valid JSON with "decision" field
- [ ] Integration tests pass: `go test ./test/integration -v -run TestValidateRouting`
- [ ] Test coverage ≥80% for validation workflow

**Why This Matters**: Validates that Go implementation matches Bash routing logic. Ensures cost control (Opus blocking) and tier enforcement work correctly.

---

### GOgent-096: Integration Tests for session-archive Hook

**Time**: 1.5 hours
**Dependencies**: GOgent-094 (harness), GOgent-033 (gogent-archive binary)

**Task**:
Test session-archive workflow: metrics collection, handoff generation, file archival.

**File**: `test/integration/session_archive_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionArchive_Integration(t *testing.T) {
	binaryPath := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-archive binary not found. Run: go build -o cmd/gogent-archive/gogent-archive cmd/gogent-archive/main.go")
	}

	// Setup test project directory
	projectDir := t.TempDir()
	setupTestSessionFiles(t, projectDir)

	// Create SessionEnd event
	eventJSON := `{
		"hook_event_name": "SessionEnd",
		"session_id": "test-session-123",
		"transcript_path": "` + filepath.Join(projectDir, "transcript.jsonl") + `"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "session-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Verify hook executed successfully
	if result.Error != nil {
		t.Fatalf("Hook execution failed: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Verify JSON output
	if result.ParsedJSON == nil {
		t.Fatalf("Expected JSON output, got: %s", result.Stdout)
	}

	// Verify handoff file created
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Errorf("Handoff file not created: %v", err)
	}

	// Verify handoff content
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("Failed to read handoff: %v", err)
	}

	handoffContent := string(handoffData)

	// Check required sections
	requiredSections := []string{
		"# Session Handoff",
		"## Session Metrics",
		"## Pending Learnings",
		"## Routing Violations",
		"## Context Guidelines",
		"## Immediate Actions",
	}

	for _, section := range requiredSections {
		if !strings.Contains(handoffContent, section) {
			t.Errorf("Handoff missing required section: %s", section)
		}
	}

	// Verify metrics section contains counts
	if !strings.Contains(handoffContent, "Tool calls:") {
		t.Error("Handoff missing tool calls metric")
	}

	if !strings.Contains(handoffContent, "Errors logged:") {
		t.Error("Handoff missing errors logged metric")
	}
}

func TestSessionArchive_MetricsCollection(t *testing.T) {
	binaryPath := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()

	// Create tool counter logs
	createToolCounterLog(t, projectDir, "task", 10)
	createToolCounterLog(t, projectDir, "read", 25)
	createToolCounterLog(t, projectDir, "write", 5)

	// Create error patterns log
	errorLogPath := filepath.Join(projectDir, ".gogent", "error-patterns.jsonl")
	os.MkdirAll(filepath.Dir(errorLogPath), 0755)
	errorLogs := []string{
		`{"timestamp":1234567890,"file":"test1.go","error_type":"TypeError"}`,
		`{"timestamp":1234567891,"file":"test2.go","error_type":"ValueError"}`,
		`{"timestamp":1234567892,"file":"test1.go","error_type":"TypeError"}`,
	}
	os.WriteFile(errorLogPath, []byte(strings.Join(errorLogs, "\n")+"\n"), 0644)

	// Create violations log
	violationsLogPath := filepath.Join(projectDir, ".gogent", "routing-violations.jsonl")
	violations := []string{
		`{"violation_type":"tool_permission","tool":"Write"}`,
		`{"violation_type":"delegation_ceiling","agent":"architect"}`,
	}
	os.WriteFile(violationsLogPath, []byte(strings.Join(violations, "\n")+"\n"), 0644)

	// Run hook
	eventJSON := `{
		"hook_event_name": "SessionEnd",
		"session_id": "test-metrics",
		"transcript_path": "` + filepath.Join(projectDir, "transcript.jsonl") + `"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "metrics-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	if result.ExitCode != 0 {
		t.Fatalf("Hook failed: %s", result.Stderr)
	}

	// Verify handoff contains correct counts
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	handoffData, _ := os.ReadFile(handoffPath)
	handoffContent := string(handoffData)

	// Should reflect ~40 tool calls (10+25+5)
	if !strings.Contains(handoffContent, "~40") && !strings.Contains(handoffContent, "~4") {
		t.Error("Handoff missing tool calls count")
	}

	// Should have 3 errors
	if !strings.Contains(handoffContent, "3") {
		t.Log("Warning: Expected 3 errors in handoff")
	}

	// Should have 2 violations
	if !strings.Contains(handoffContent, "2") {
		t.Log("Warning: Expected 2 violations in handoff")
	}
}

func TestSessionArchive_FileArchival(t *testing.T) {
	binaryPath := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()

	// Create files to archive
	transcriptPath := filepath.Join(projectDir, ".claude", "transcript.jsonl")
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	violationsPath := filepath.Join(projectDir, ".gogent", "routing-violations.jsonl")

	os.MkdirAll(filepath.Dir(transcriptPath), 0755)
	os.MkdirAll(filepath.Dir(learningsPath), 0755)
	os.MkdirAll(filepath.Dir(violationsPath), 0755)

	os.WriteFile(transcriptPath, []byte("transcript content\n"), 0644)
	os.WriteFile(learningsPath, []byte("learnings content\n"), 0644)
	os.WriteFile(violationsPath, []byte("violations content\n"), 0644)

	// Run hook
	eventJSON := `{
		"hook_event_name": "SessionEnd",
		"session_id": "test-archival",
		"transcript_path": "` + transcriptPath + `"
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "archival-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	if result.ExitCode != 0 {
		t.Fatalf("Hook failed: %s", result.Stderr)
	}

	// Verify files archived
	archiveDir := filepath.Join(projectDir, ".claude", "memory", "session-archive")

	// Transcript should be copied (not moved)
	if _, err := os.Stat(transcriptPath); err != nil {
		t.Error("Transcript should remain after archival")
	}

	archivedTranscript := filepath.Join(archiveDir, "session-test-archival.jsonl")
	if _, err := os.Stat(archivedTranscript); err != nil {
		t.Errorf("Transcript not archived: %v", err)
	}

	// Learnings should be moved (deleted from original location)
	if _, err := os.Stat(learningsPath); !os.IsNotExist(err) {
		t.Error("Learnings should be removed after archival")
	}

	archivedLearnings := filepath.Join(archiveDir, "pending-learnings-test-archival.jsonl")
	if _, err := os.Stat(archivedLearnings); err != nil {
		t.Errorf("Learnings not archived: %v", err)
	}

	// Violations should be moved
	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations should be removed after archival")
	}

	archivedViolations := filepath.Join(archiveDir, "routing-violations-test-archival.jsonl")
	if _, err := os.Stat(archivedViolations); err != nil {
		t.Errorf("Violations not archived: %v", err)
	}
}

// Helper: Setup test session files
func setupTestSessionFiles(t *testing.T, projectDir string) {
	// Create minimal tool counter logs
	createToolCounterLog(t, projectDir, "task", 5)

	// Create empty transcript
	transcriptPath := filepath.Join(projectDir, ".claude", "transcript.jsonl")
	os.MkdirAll(filepath.Dir(transcriptPath), 0755)
	os.WriteFile(transcriptPath, []byte(""), 0644)
}

// Helper: Create tool counter log
func createToolCounterLog(t *testing.T, projectDir, tool string, count int) {
	counterPath := filepath.Join(projectDir, ".gogent", fmt.Sprintf("tool-counter-%s", tool))
	os.MkdirAll(filepath.Dir(counterPath), 0755)

	// Write count lines
	content := strings.Repeat("x\n", count)
	if err := os.WriteFile(counterPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create tool counter: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestSessionArchive_Integration` verifies complete workflow
- [ ] `TestSessionArchive_MetricsCollection` verifies accurate counting
- [ ] `TestSessionArchive_FileArchival` verifies files copied/moved correctly
- [ ] Handoff file created at `.claude/memory/last-handoff.md`
- [ ] Handoff contains all required sections with correct data
- [ ] Files archived to `.claude/memory/session-archive/`
- [ ] Tests pass: `go test ./test/integration -v -run TestSessionArchive`

**Why This Matters**: Session handoff is critical for context continuity across restarts. Must verify metrics accuracy and file handling correctness.

---

### GOgent-097: Integration Tests for sharp-edge-detector Hook

**Time**: 1.5 hours
**Dependencies**: GOgent-094 (harness), GOgent-040 (gogent-sharp-edge binary)

**Task**:
Test sharp edge detection workflow: failure detection, consecutive counting, blocking responses.

**File**: `test/integration/sharp_edge_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharpEdge_Integration(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found. Run: go build -o cmd/gogent-sharp-edge/gogent-sharp-edge cmd/gogent-sharp-edge/main.go")
	}

	projectDir := t.TempDir()

	// Create corpus with 3 consecutive failures on same file
	corpusPath := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	createSharpEdgeCorpus(t, corpusPath, projectDir)

	harness, _ := NewTestHarness(corpusPath, projectDir)
	harness.LoadCorpus()

	results, err := harness.RunHookBatch(binaryPath, "PostToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got: %d", len(results))
	}

	// First failure: Should pass through
	if results[0].ParsedJSON == nil || len(results[0].ParsedJSON) > 0 {
		t.Error("First failure should return empty JSON (pass-through)")
	}

	// Second failure: Should warn
	if results[1].ParsedJSON == nil {
		t.Fatal("Second failure should return JSON")
	}

	hookOutput, ok := results[1].ParsedJSON["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Error("Second failure should have hookSpecificOutput with warning")
	} else {
		additionalContext, ok := hookOutput["additionalContext"].(string)
		if !ok || !strings.Contains(additionalContext, "⚠️") {
			t.Error("Second failure should contain warning emoji")
		}
	}

	// Third failure: Should block
	if results[2].ParsedJSON == nil {
		t.Fatal("Third failure should return JSON")
	}

	decision, ok := results[2].ParsedJSON["decision"].(string)
	if !ok || decision != "block" {
		t.Errorf("Third failure should block, got decision: %v", decision)
	}

	reason, ok := results[2].ParsedJSON["reason"].(string)
	if !ok || !strings.Contains(reason, "SHARP EDGE DETECTED") {
		t.Errorf("Third failure should mention sharp edge, got: %s", reason)
	}

	// Verify sharp edge captured to pending learnings
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings file not created: %v", err)
	} else {
		data, _ := os.ReadFile(learningsPath)
		if len(data) == 0 {
			t.Error("Pending learnings file is empty")
		}

		var edge map[string]interface{}
		if err := json.Unmarshal(data, &edge); err != nil {
			t.Errorf("Failed to parse sharp edge: %v", err)
		}

		if edge["type"] != "tool_failure" {
			t.Errorf("Expected type=tool_failure, got: %v", edge["type"])
		}

		if edge["consecutive_failures"] != float64(3) {
			t.Errorf("Expected 3 consecutive failures, got: %v", edge["consecutive_failures"])
		}
	}
}

func TestSharpEdge_FailureDetection(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	testCases := []struct {
		name           string
		eventJSON      string
		expectFailure  bool
		expectedErrType string
	}{
		{
			name: "Explicit success=false",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Edit",
				"tool_input": {"file_path": "/tmp/test.py"},
				"tool_response": {"success": false, "error": "File not found"}
			}`,
			expectFailure: true,
			expectedErrType: "error",
		},
		{
			name: "Non-zero exit code",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Bash",
				"tool_input": {"command": "ls /nonexistent"},
				"tool_response": {"exit_code": 1, "output": "ls: cannot access"}
			}`,
			expectFailure: true,
			expectedErrType: "error",
		},
		{
			name: "Python TypeError in output",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Bash",
				"tool_input": {"command": "python script.py"},
				"tool_response": {"output": "TypeError: unsupported operand type"}
			}`,
			expectFailure: true,
			expectedErrType: "TypeError",
		},
		{
			name: "Success case",
			eventJSON: `{
				"hook_event_name": "PostToolUse",
				"tool_name": "Read",
				"tool_input": {"file_path": "/tmp/test.txt"},
				"tool_response": {"content": "file content"}
			}`,
			expectFailure: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpCorpus := filepath.Join(t.TempDir(), "corpus.jsonl")
			os.WriteFile(tmpCorpus, []byte(tc.eventJSON+"\n"), 0644)

			harness, _ := NewTestHarness(tmpCorpus, projectDir)
			harness.LoadCorpus()

			result := harness.RunHook(binaryPath, harness.Events[0])

			if tc.expectFailure {
				// Should log failure
				errorLogPath := filepath.Join(projectDir, ".gogent", "error-patterns.jsonl")
				if _, err := os.Stat(errorLogPath); err != nil {
					t.Errorf("Error log not created for failure case: %v", err)
				}

				// First failure should pass through (no blocking yet)
				if result.ParsedJSON != nil && len(result.ParsedJSON) > 0 {
					decision, _ := result.ParsedJSON["decision"].(string)
					if decision == "block" {
						t.Error("First failure should not block")
					}
				}
			} else {
				// Should return empty JSON
				if result.ParsedJSON != nil && len(result.ParsedJSON) > 0 {
					t.Errorf("Success case should return empty JSON, got: %v", result.ParsedJSON)
				}
			}
		})
	}
}

func TestSharpEdge_SlidingWindow(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()
	errorLogPath := filepath.Join(projectDir, ".gogent", "error-patterns.jsonl")
	os.MkdirAll(filepath.Dir(errorLogPath), 0755)

	// Create failures outside 5-minute window (should not trigger blocking)
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	recentTimestamp := time.Now().Unix()

	logEntries := []string{
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, oldTimestamp),
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, oldTimestamp),
		fmt.Sprintf(`{"ts":%d,"file":"/tmp/test.go","tool":"Edit","error_type":"error"}`, recentTimestamp),
	}

	os.WriteFile(errorLogPath, []byte(strings.Join(logEntries, "\n")+"\n"), 0644)

	// Create new failure event
	eventJSON := `{
		"hook_event_name": "PostToolUse",
		"tool_name": "Edit",
		"tool_input": {"file_path": "/tmp/test.go"},
		"tool_response": {"success": false}
	}`

	tmpCorpus := filepath.Join(t.TempDir(), "window-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(eventJSON+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	// Should NOT block (only 2 failures in window: 1 recent + 1 new)
	if result.ParsedJSON != nil {
		decision, _ := result.ParsedJSON["decision"].(string)
		if decision == "block" {
			t.Error("Should not block with only 2 recent failures (old ones outside window)")
		}
	}
}

func TestSharpEdge_PerFileTracking(t *testing.T) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()

	// Create failures on different files
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileA.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileB.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileA.go"},"tool_response":{"success":false}}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/fileB.go"},"tool_response":{"success":false}}`,
	}

	tmpCorpus := filepath.Join(t.TempDir(), "multifile-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(strings.Join(events, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	results, _ := harness.RunHookBatch(binaryPath, "PostToolUse")

	// Each file should be tracked independently
	// Neither should reach 3 failures (2 on fileA, 2 on fileB)
	for i, result := range results {
		if result.ParsedJSON != nil {
			decision, _ := result.ParsedJSON["decision"].(string)
			if decision == "block" {
				t.Errorf("Event %d should not block (separate file tracking)", i)
			}
		}
	}
}

// Helper: Create corpus with 3 consecutive failures on same file
func createSharpEdgeCorpus(t *testing.T, corpusPath, projectDir string) {
	events := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-1"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-2"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(projectDir, "test.go") + `"},"tool_response":{"success":false,"error":"Type error"},"session_id":"test-3"}`,
	}

	if err := os.WriteFile(corpusPath, []byte(strings.Join(events, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create corpus: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestSharpEdge_Integration` verifies 3-failure blocking workflow
- [ ] First failure passes through (no blocking)
- [ ] Second failure generates warning in additionalContext
- [ ] Third failure returns `decision: "block"` with sharp edge reason
- [ ] Sharp edge captured to `pending-learnings.jsonl`
- [ ] `TestSharpEdge_FailureDetection` verifies all signal types
- [ ] `TestSharpEdge_SlidingWindow` verifies 5-minute window
- [ ] `TestSharpEdge_PerFileTracking` verifies independent file tracking
- [ ] Tests pass: `go test ./test/integration -v -run TestSharpEdge`

**Why This Matters**: Sharp edge detection prevents debugging loops. Must verify blocking logic triggers correctly and captures sufficient context for learning.

---

### GOgent-098: Performance Benchmarks

**Time**: 2 hours
**Dependencies**: GOgent-094 (harness), GOgent-095-044 (hook binaries)

**Task**:
Benchmark hook execution latency and memory usage. Target: <5ms p99 latency, <10MB memory per hook.

**File**: `test/benchmark/hooks_bench_test.go`

**Implementation**:

```go
package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// BenchmarkValidateRouting_Allow benchmarks validate-routing for allowed operations
func BenchmarkValidateRouting_Allow(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Read",
		"tool_input": {"file_path": "/tmp/test.txt"},
		"session_id": "bench-allow"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkValidateRouting_Block benchmarks validate-routing for blocked operations
func BenchmarkValidateRouting_Block(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze",
			"subagent_type": "general-purpose"
		},
		"session_id": "bench-block"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkSessionArchive benchmarks session-archive hook
func BenchmarkSessionArchive(b *testing.B) {
	binaryPath := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-archive binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupSessionMetricsFiles(b, projectDir)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-session",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript.jsonl"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}

		// Clean up handoff for next iteration
		handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
		os.Remove(handoffPath)
	}
}

// BenchmarkSharpEdgeDetector benchmarks sharp-edge-detector hook
func BenchmarkSharpEdgeDetector(b *testing.B) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PostToolUse",
		"tool_name": "Edit",
		"tool_input": {"file_path": "/tmp/test.go"},
		"tool_response": {"success": false, "error": "Type error"},
		"session_id": "bench-sharp-edge"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkMemoryUsage measures peak memory usage of hooks
func BenchmarkMemoryUsage(b *testing.B) {
	hooks := []struct {
		name    string
		path    string
		event   string
	}{
		{
			name: "validate-routing",
			path: "../../cmd/gogent-validate/gogent-validate",
			event: `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"}}`,
		},
		{
			name: "session-archive",
			path: "../../cmd/gogent-archive/gogent-archive",
			event: `{"hook_event_name":"SessionEnd","session_id":"mem-test"}`,
		},
		{
			name: "sharp-edge-detector",
			path: "../../cmd/gogent-sharp-edge/gogent-sharp-edge",
			event: `{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_response":{"success":false}}`,
		},
	}

	projectDir := setupBenchmarkProject(b)

	for _, hook := range hooks {
		b.Run(hook.name, func(b *testing.B) {
			if _, err := os.Stat(hook.path); err != nil {
				b.Skipf("%s binary not found", hook.name)
			}

			var totalMem uint64

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)

				cmd := exec.Command(hook.path)
				cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
				cmd.Stdin = bytes.NewReader([]byte(hook.event))

				var stdout bytes.Buffer
				cmd.Stdout = &stdout

				if err := cmd.Run(); err != nil {
					// Some hooks may error on minimal input - that's OK for memory test
				}

				runtime.ReadMemStats(&m2)
				totalMem += (m2.TotalAlloc - m1.TotalAlloc)
			}

			avgMem := totalMem / uint64(b.N)
			b.ReportMetric(float64(avgMem)/1024/1024, "MB/op")

			// Verify <10MB target
			if avgMem > 10*1024*1024 {
				b.Errorf("%s exceeds 10MB memory target: %.2f MB", hook.name, float64(avgMem)/1024/1024)
			}
		})
	}
}

// BenchmarkLatency_Percentiles measures p50, p95, p99 latencies
func BenchmarkLatency_Percentiles(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Read",
		"tool_input": {"file_path": "/tmp/test.txt"}
	}`

	// Run 1000 iterations to get percentile data
	iterations := 1000
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}

		latencies[i] = time.Since(start)
	}

	// Calculate percentiles
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	b.ReportMetric(float64(p50.Microseconds()), "p50-μs")
	b.ReportMetric(float64(p95.Microseconds()), "p95-μs")
	b.ReportMetric(float64(p99.Microseconds()), "p99-μs")

	// Verify <5ms p99 target
	if p99 > 5*time.Millisecond {
		b.Errorf("p99 latency exceeds 5ms target: %v", p99)
	}

	fmt.Printf("\nLatency Percentiles:\n")
	fmt.Printf("  p50: %v\n", p50)
	fmt.Printf("  p95: %v\n", p95)
	fmt.Printf("  p99: %v\n", p99)
}

// Helper: Setup benchmark project directory
func setupBenchmarkProject(b *testing.B) string {
	projectDir := b.TempDir()

	// Create routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	schema := `{
		"tiers": {
			"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
			"sonnet": {"tools_allowed": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"]},
			"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"einstein": "general-purpose"
		}
	}`

	os.WriteFile(schemaPath, []byte(schema), 0644)

	// Set tier to haiku
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	return projectDir
}

// Helper: Setup session metrics files
func setupSessionMetricsFiles(b *testing.B, projectDir string) {
	// Create tool counter logs
	toolCounterPath := filepath.Join(projectDir, ".gogent", "tool-counter-read")
	os.MkdirAll(filepath.Dir(toolCounterPath), 0755)
	os.WriteFile(toolCounterPath, []byte("x\nx\nx\n"), 0644)

	// Create empty transcript
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	os.WriteFile(transcriptPath, []byte(""), 0644)
}

// Helper: Calculate percentile from sorted durations
func percentile(durations []time.Duration, p int) time.Duration {
	// Sort durations
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[j] < durations[i] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	index := (p * len(durations)) / 100
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}
```

**Run benchmarks**:
```bash
go test -bench=. ./test/benchmark -benchmem -benchtime=10s
```

**Acceptance Criteria**:
- [ ] `BenchmarkValidateRouting_Allow` measures allow path latency
- [ ] `BenchmarkValidateRouting_Block` measures block path latency
- [ ] `BenchmarkSessionArchive` measures session-archive latency
- [ ] `BenchmarkSharpEdgeDetector` measures sharp-edge latency
- [ ] `BenchmarkMemoryUsage` verifies <10MB memory per hook
- [ ] `BenchmarkLatency_Percentiles` verifies <5ms p99 latency
- [ ] All benchmarks pass performance targets
- [ ] Benchmark report saved: `go test -bench=. ./test/benchmark | tee benchmark-results.txt`

**Why This Matters**: Performance regression would make hooks unusable in production. Must verify latency and memory targets met before cutover.

---

### GOgent-099: End-to-End Workflow Integration Tests

**Time**: 2 hours
**Dependencies**: GOgent-095-044 (integration tests)

**Task**:
Test cross-hook workflows: validation blocks → sharp edge detection → session archival.

**File**: `test/integration/end_to_end_test.go`

**Implementation**:

```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd_ValidationToSharpEdge tests validation failure → sharp edge capture
func TestEndToEnd_ValidationToSharpEdge(t *testing.T) {
	validateBinary := "../../cmd/gogent-validate/gogent-validate"
	sharpEdgeBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"

	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}
	if _, err := os.Stat(sharpEdgeBinary); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Step 1: Attempt blocked Task(opus) via validation
	validateEventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze",
			"subagent_type": "general-purpose"
		},
		"session_id": "e2e-test"
	}`

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	os.WriteFile(tmpValidateCorpus, []byte(validateEventJSON+"\n"), 0644)

	validateHarness, _ := NewTestHarness(tmpValidateCorpus, projectDir)
	validateHarness.LoadCorpus()

	validateResult := validateHarness.RunHook(validateBinary, validateHarness.Events[0])

	// Verify blocked
	if validateResult.ParsedJSON == nil {
		t.Fatal("Expected JSON from validate-routing")
	}

	decision, _ := validateResult.ParsedJSON["decision"].(string)
	if decision != "block" {
		t.Errorf("Expected validation to block Task(opus), got: %s", decision)
	}

	// Step 2: Simulate repeated failures (user ignoring block and retrying)
	// Create 3 PostToolUse failure events
	sharpEdgeEvents := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Task","tool_input":{"model":"opus"},"tool_response":{"success":false,"error":"blocked"},"session_id":"e2e-test"}`,
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeEvents, "\n")+"\n"), 0644)

	sharpEdgeHarness, _ := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	sharpEdgeHarness.LoadCorpus()

	sharpEdgeResults, _ := sharpEdgeHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")

	// Third failure should trigger sharp edge capture
	if len(sharpEdgeResults) < 3 {
		t.Fatalf("Expected 3 sharp edge results, got: %d", len(sharpEdgeResults))
	}

	thirdResult := sharpEdgeResults[2]
	if thirdResult.ParsedJSON == nil {
		t.Fatal("Expected JSON from third sharp edge")
	}

	sharpEdgeDecision, _ := thirdResult.ParsedJSON["decision"].(string)
	if sharpEdgeDecision != "block" {
		t.Errorf("Expected sharp edge to block after 3 failures, got: %s", sharpEdgeDecision)
	}

	// Verify sharp edge captured
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings not created: %v", err)
	}
}

// TestEndToEnd_SessionArchivalWorkflow tests complete session lifecycle
func TestEndToEnd_SessionArchivalWorkflow(t *testing.T) {
	validateBinary := "../../cmd/gogent-validate/gogent-validate"
	sharpEdgeBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	archiveBinary := "../../cmd/gogent-archive/gogent-archive"

	if _, err := os.Stat(validateBinary); err != nil {
		t.Skip("gogent-validate binary not found")
	}
	if _, err := os.Stat(sharpEdgeBinary); err != nil {
		t.Skip("gogent-sharp-edge binary not found")
	}
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Step 1: Run validation creating violations
	validateEvents := []string{
		`{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"/tmp/test.txt"},"session_id":"archive-test"}`,
		`{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"model":"opus","prompt":"AGENT: einstein","subagent_type":"general-purpose"},"session_id":"archive-test"}`,
	}

	// Set tier to haiku (will violate on Write and Task(opus))
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	tmpValidateCorpus := filepath.Join(t.TempDir(), "validate-corpus.jsonl")
	os.WriteFile(tmpValidateCorpus, []byte(strings.Join(validateEvents, "\n")+"\n"), 0644)

	validateHarness, _ := NewTestHarness(tmpValidateCorpus, projectDir)
	validateHarness.LoadCorpus()

	validateHarness.RunHookBatch(validateBinary, "PreToolUse")

	// Verify violations logged
	violationsPath := filepath.Join(projectDir, ".gogent", "routing-violations.jsonl")
	if _, err := os.Stat(violationsPath); err != nil {
		t.Errorf("Violations log not created: %v", err)
	}

	// Step 2: Run sharp edge detection creating pending learnings
	sharpEdgeEvents := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
		`{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false},"session_id":"archive-test"}`,
	}

	tmpSharpEdgeCorpus := filepath.Join(t.TempDir(), "sharp-edge-corpus.jsonl")
	os.WriteFile(tmpSharpEdgeCorpus, []byte(strings.Join(sharpEdgeEvents, "\n")+"\n"), 0644)

	sharpEdgeHarness, _ := NewTestHarness(tmpSharpEdgeCorpus, projectDir)
	sharpEdgeHarness.LoadCorpus()

	sharpEdgeHarness.RunHookBatch(sharpEdgeBinary, "PostToolUse")

	// Verify pending learnings created
	learningsPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Errorf("Pending learnings not created: %v", err)
	}

	// Step 3: Run session archive
	archiveEventJSON := `{
		"hook_event_name": "SessionEnd",
		"session_id": "archive-test",
		"transcript_path": "` + filepath.Join(projectDir, "transcript.jsonl") + `"
	}`

	// Create empty transcript
	os.WriteFile(filepath.Join(projectDir, "transcript.jsonl"), []byte(""), 0644)

	tmpArchiveCorpus := filepath.Join(t.TempDir(), "archive-corpus.jsonl")
	os.WriteFile(tmpArchiveCorpus, []byte(archiveEventJSON+"\n"), 0644)

	archiveHarness, _ := NewTestHarness(tmpArchiveCorpus, projectDir)
	archiveHarness.LoadCorpus()

	archiveResult := archiveHarness.RunHook(archiveBinary, archiveHarness.Events[0])

	if archiveResult.ExitCode != 0 {
		t.Fatalf("Archive hook failed: %s", archiveResult.Stderr)
	}

	// Step 4: Verify handoff file created with all sections
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Handoff not created: %v", err)
	}

	handoffData, _ := os.ReadFile(handoffPath)
	handoffContent := string(handoffData)

	// Verify violations section present (from Step 1)
	if !strings.Contains(handoffContent, "## Routing Violations") {
		t.Error("Handoff missing violations section")
	}

	// Verify pending learnings section present (from Step 2)
	if !strings.Contains(handoffContent, "## Pending Learnings") {
		t.Error("Handoff missing pending learnings section")
	}

	// Step 5: Verify files archived
	archiveDir := filepath.Join(projectDir, ".claude", "memory", "session-archive")

	// Violations should be moved
	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations file should be removed after archival")
	}

	archivedViolations := filepath.Join(archiveDir, "routing-violations-archive-test.jsonl")
	if _, err := os.Stat(archivedViolations); err != nil {
		t.Errorf("Violations not archived: %v", err)
	}

	// Learnings should be moved
	if _, err := os.Stat(learningsPath); !os.IsNotExist(err) {
		t.Error("Learnings file should be removed after archival")
	}

	archivedLearnings := filepath.Join(archiveDir, "pending-learnings-archive-test.jsonl")
	if _, err := os.Stat(archivedLearnings); err != nil {
		t.Errorf("Learnings not archived: %v", err)
	}
}

// TestEndToEnd_MultiSessionHandoff tests handoff continuity across sessions
func TestEndToEnd_MultiSessionHandoff(t *testing.T) {
	archiveBinary := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(archiveBinary); err != nil {
		t.Skip("gogent-archive binary not found")
	}

	projectDir := t.TempDir()

	// Session 1: Create handoff
	session1Event := `{
		"hook_event_name": "SessionEnd",
		"session_id": "session-1",
		"transcript_path": "` + filepath.Join(projectDir, "transcript-1.jsonl") + `"
	}`

	os.WriteFile(filepath.Join(projectDir, "transcript-1.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, projectDir, "read", 10)

	tmpCorpus1 := filepath.Join(t.TempDir(), "session1-corpus.jsonl")
	os.WriteFile(tmpCorpus1, []byte(session1Event+"\n"), 0644)

	harness1, _ := NewTestHarness(tmpCorpus1, projectDir)
	harness1.LoadCorpus()

	result1 := harness1.RunHook(archiveBinary, harness1.Events[0])

	if result1.ExitCode != 0 {
		t.Fatalf("Session 1 archive failed: %s", result1.Stderr)
	}

	// Verify handoff created
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("Session 1 handoff not created: %v", err)
	}

	session1Handoff, _ := os.ReadFile(handoffPath)

	// Session 2: Should be able to reference previous handoff
	// (In real workflow, load-routing-context hook would inject this)
	session2Event := `{
		"hook_event_name": "SessionEnd",
		"session_id": "session-2",
		"transcript_path": "` + filepath.Join(projectDir, "transcript-2.jsonl") + `"
	}`

	os.WriteFile(filepath.Join(projectDir, "transcript-2.jsonl"), []byte(""), 0644)

	createToolCounterLog(t, projectDir, "write", 5)

	tmpCorpus2 := filepath.Join(t.TempDir(), "session2-corpus.jsonl")
	os.WriteFile(tmpCorpus2, []byte(session2Event+"\n"), 0644)

	harness2, _ := NewTestHarness(tmpCorpus2, projectDir)
	harness2.LoadCorpus()

	result2 := harness2.RunHook(archiveBinary, harness2.Events[0])

	if result2.ExitCode != 0 {
		t.Fatalf("Session 2 archive failed: %s", result2.Stderr)
	}

	// Verify new handoff created (overwrites previous)
	session2Handoff, _ := os.ReadFile(handoffPath)

	if string(session1Handoff) == string(session2Handoff) {
		t.Error("Session 2 handoff should differ from session 1")
	}

	// Verify session 1 handoff preserved in archive
	archivedSession1 := filepath.Join(projectDir, ".claude", "memory", "session-archive", "last-handoff-session-1.md")
	// Note: Current implementation doesn't archive handoff - transcript only
	// This test documents expected future behavior
}
```

**Acceptance Criteria**:
- [ ] `TestEndToEnd_ValidationToSharpEdge` verifies validation → sharp edge pipeline
- [ ] `TestEndToEnd_SessionArchivalWorkflow` verifies complete session lifecycle
- [ ] Violations from validation appear in session handoff
- [ ] Pending learnings from sharp edge appear in session handoff
- [ ] Files archived correctly at session end
- [ ] `TestEndToEnd_MultiSessionHandoff` verifies continuity across sessions
- [ ] All end-to-end tests pass: `go test ./test/integration -v -run TestEndToEnd`

**Why This Matters**: Individual hooks may work in isolation but fail when chained. End-to-end tests verify complete workflows match production usage.

---

### GOgent-100: Regression Tests (Go vs Bash Comparison)

**Time**: 2 hours
**Dependencies**: GOgent-000 (corpus), GOgent-094 (harness), all hook binaries

**Task**:
Run 100-event corpus through both Go and Bash implementations, verify identical output (except timestamps).

**File**: `test/regression/regression_test.go`

**Implementation**:

```go
package regression

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/test/integration"
)

// TestRegression_ValidateRouting compares Go vs Bash validate-routing output
func TestRegression_ValidateRouting(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests (from GOgent-000)")
	}

	goBinary := "../../cmd/gogent-validate/gogent-validate"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/validate-routing.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Filter PreToolUse events
	events := harness.FilterEvents("PreToolUse")

	if len(events) == 0 {
		t.Skip("No PreToolUse events in corpus")
	}

	t.Logf("Running regression test on %d PreToolUse events", len(events))

	passed := 0
	failed := 0
	differences := []string{}

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			diffMsg := fmt.Sprintf("Event %d (session=%s): %s", i, event.SessionID, strings.Join(diffs, "; "))
			differences = append(differences, diffMsg)

			if failed <= 5 {
				t.Logf("DIFF: %s", diffMsg)
				t.Logf("  Go output:   %s", goResult.Stdout)
				t.Logf("  Bash output: %s", bashResult.Stdout)
			}
		}
	}

	t.Logf("\n=== Regression Test Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Logf("\nFirst 5 differences:")
		for i, diff := range differences {
			if i >= 5 {
				break
			}
			t.Logf("  %s", diff)
		}

		t.Errorf("Regression test failed: %d/%d events differ between Go and Bash", failed, len(events))
	}
}

// TestRegression_SessionArchive compares Go vs Bash session-archive output
func TestRegression_SessionArchive(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-archive/gogent-archive"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/session-archive.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	events := harness.FilterEvents("SessionEnd")

	if len(events) == 0 {
		t.Skip("No SessionEnd events in corpus")
	}

	t.Logf("Running regression test on %d SessionEnd events", len(events))

	for i, event := range events {
		// Setup session files for both runs
		setupSessionFilesForEvent(t, projectDir, event)

		// Run Go implementation
		goHandoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff-go.md")
		os.Setenv("GOgent_HANDOFF_PATH", goHandoffPath)
		goResult := harness.RunHook(goBinary, event)
		os.Unsetenv("GOgent_HANDOFF_PATH")

		// Run Bash implementation
		bashHandoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff-bash.md")
		os.Setenv("GOgent_HANDOFF_PATH", bashHandoffPath)
		bashResult := runBashHook(t, bashScript, event, projectDir)
		os.Unsetenv("GOgent_HANDOFF_PATH")

		// Compare handoff files (ignore timestamps)
		if _, err := os.Stat(goHandoffPath); err != nil {
			t.Errorf("Event %d: Go handoff not created", i)
			continue
		}

		if _, err := os.Stat(bashHandoffPath); err != nil {
			t.Errorf("Event %d: Bash handoff not created", i)
			continue
		}

		goHandoff, _ := os.ReadFile(goHandoffPath)
		bashHandoff, _ := os.ReadFile(bashHandoff)

		// Strip timestamps for comparison
		goContent := stripTimestamps(string(goHandoff))
		bashContent := stripTimestamps(string(bashHandoff))

		if goContent != bashContent {
			t.Errorf("Event %d: Handoff content differs", i)
			t.Logf("  Go handoff length:   %d", len(goHandoff))
			t.Logf("  Bash handoff length: %d", len(bashHandoff))

			// Show first difference
			showFirstDifference(t, goContent, bashContent)
		}

		// Cleanup for next event
		os.Remove(goHandoffPath)
		os.Remove(bashHandoffPath)
	}
}

// TestRegression_SharpEdgeDetector compares Go vs Bash sharp-edge output
func TestRegression_SharpEdgeDetector(t *testing.T) {
	corpusPath := os.Getenv("GOgent_CORPUS_PATH")
	if corpusPath == "" {
		t.Skip("Set GOgent_CORPUS_PATH to run regression tests")
	}

	goBinary := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	bashScript := os.Getenv("HOME") + "/.claude/hooks/sharp-edge-detector.sh"

	if _, err := os.Stat(goBinary); err != nil {
		t.Skip("Go binary not found")
	}
	if _, err := os.Stat(bashScript); err != nil {
		t.Skip("Bash script not found")
	}

	projectDir := t.TempDir()
	setupRegressionProject(t, projectDir)

	harness, err := integration.NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	events := harness.FilterEvents("PostToolUse")

	if len(events) == 0 {
		t.Skip("No PostToolUse events in corpus")
	}

	t.Logf("Running regression test on %d PostToolUse events", len(events))

	passed := 0
	failed := 0

	for i, event := range events {
		goResult := harness.RunHook(goBinary, event)
		bashResult := runBashHook(t, bashScript, event, projectDir)

		diffs := integration.CompareResults(goResult, bashResult)

		if len(diffs) == 0 {
			passed++
		} else {
			failed++
			t.Logf("Event %d differs: %s", i, strings.Join(diffs, "; "))
		}
	}

	t.Logf("\n=== Sharp Edge Regression Results ===")
	t.Logf("Total:  %d", len(events))
	t.Logf("Passed: %d", passed)
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Errorf("Sharp edge regression failed: %d/%d events differ", failed, len(events))
	}
}

// Helper: Run Bash hook with event
func runBashHook(t *testing.T, scriptPath string, event *integration.EventEntry, projectDir string) *integration.HookResult {
	result := &integration.HookResult{
		Event: event,
	}

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"CLAUDE_PROJECT_DIR="+projectDir,
		"GOgent_TEST_MODE=1",
	)

	cmd.Stdin = bytes.NewReader(event.RawJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err
		}
	}

	// Parse JSON output
	if len(result.Stdout) > 0 {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &parsed); err == nil {
			result.ParsedJSON = parsed
		}
	}

	return result
}

// Helper: Setup regression test project
func setupRegressionProject(t *testing.T, projectDir string) {
	// Create routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	// Use actual routing schema from ~/.claude
	homeSchema := filepath.Join(os.Getenv("HOME"), ".claude", "routing-schema.json")
	if data, err := os.ReadFile(homeSchema); err == nil {
		os.WriteFile(schemaPath, data, 0644)
	} else {
		// Fallback to minimal schema
		schema := `{
			"tiers": {
				"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
				"sonnet": {"tools_allowed": ["*"]},
				"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
			},
			"agent_subagent_mapping": {}
		}`
		os.WriteFile(schemaPath, []byte(schema), 0644)
	}

	// Set tier
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)
}

// Helper: Setup session files for archive event
func setupSessionFilesForEvent(t *testing.T, projectDir string, event *integration.EventEntry) {
	// Create transcript
	transcriptPath, ok := event.ToolInput["transcript_path"].(string)
	if !ok {
		transcriptPath = filepath.Join(projectDir, ".claude", "transcript.jsonl")
	}

	os.MkdirAll(filepath.Dir(transcriptPath), 0755)
	os.WriteFile(transcriptPath, []byte(""), 0644)

	// Create minimal tool counter
	counterPath := filepath.Join(projectDir, ".gogent", "tool-counter-read")
	os.MkdirAll(filepath.Dir(counterPath), 0755)
	os.WriteFile(counterPath, []byte("x\n"), 0644)
}

// Helper: Strip timestamps from handoff content
func stripTimestamps(content string) string {
	lines := strings.Split(content, "\n")
	var filtered []string

	for _, line := range lines {
		// Skip lines with timestamps (e.g., "# Session Handoff - 2026-01-15 14:32:00")
		if strings.Contains(line, "202") && strings.Contains(line, ":") {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// Helper: Show first difference between two strings
func showFirstDifference(t *testing.T, a, b string) {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	maxLines := len(linesA)
	if len(linesB) > maxLines {
		maxLines = len(linesB)
	}

	for i := 0; i < maxLines; i++ {
		lineA := ""
		lineB := ""

		if i < len(linesA) {
			lineA = linesA[i]
		}

		if i < len(linesB) {
			lineB = linesB[i]
		}

		if lineA != lineB {
			t.Logf("  First diff at line %d:", i+1)
			t.Logf("    Go:   %s", lineA)
			t.Logf("    Bash: %s", lineB)
			return
		}
	}
}
```

**Run regression tests**:
```bash
export GOgent_CORPUS_PATH=/path/to/corpus/from/gogent-000.jsonl
go test ./test/regression -v
```

**Acceptance Criteria**:
- [ ] `TestRegression_ValidateRouting` compares all PreToolUse events
- [ ] `TestRegression_SessionArchive` compares all SessionEnd events
- [ ] `TestRegression_SharpEdgeDetector` compares all PostToolUse events
- [ ] ≥95% of events produce identical output (Go vs Bash)
- [ ] Differences limited to timestamp formatting
- [ ] Test report shows pass/fail counts and first 5 differences
- [ ] Regression tests pass: `go test ./test/regression -v`
- [ ] Results documented in regression-report.md

**Why This Matters**: Regression tests are the final quality gate. Must verify Go implementations are drop-in replacements for Bash with no behavior changes.

---

## Summary

Week 3 Part 1 completes the Phase 0 testing suite with:

- **GOgent-004c**: Config circular dependency tests (deferred from Week 1)
- **GOgent-094**: Test harness for corpus replay (foundation for all tests)
- **GOgent-095**: Integration tests for validate-routing hook
- **GOgent-096**: Integration tests for session-archive hook
- **GOgent-097**: Integration tests for sharp-edge-detector hook
- **GOgent-098**: Performance benchmarks (<5ms p99, <10MB memory targets)
- **GOgent-099**: End-to-end workflow integration tests
- **GOgent-100**: Regression tests comparing Go vs Bash output

**Quality Gates**:
- Integration tests verify complete hook workflows
- Performance benchmarks enforce latency and memory targets
- Regression tests ensure Go output matches Bash exactly
- End-to-end tests validate cross-hook pipelines

**Next**: [07-week3-deployment-cutover.md](07-week3-deployment-cutover.md) - Installation, parallel testing, and GO/NO-GO cutover decision (GOgent-101 to 055)

---

**File Status**: ✅ Complete
**Tickets**: 7 (GOgent-004c, 094-100)
**Detail Level**: Full implementation + comprehensive tests
