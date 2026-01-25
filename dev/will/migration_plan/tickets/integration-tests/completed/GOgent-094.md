---
id: GOgent-094
title: Test Harness for Event Corpus Replay
description: "Build test harness that replays events from GOgent-000 corpus through hook implementations, capturing output for comparison"
status: pending
time_estimate: 2h
dependencies: ["GOgent-000","GOgent-008b"]
priority: high
week: 5
tags: ["integration-tests", "week-5"]
tests_required: true
acceptance_criteria_count: 13
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
	Timestamp      int64                  `json:"timestamp"`
	HookEventName  string                 `json:"hook_event_name"`
	ToolName       string                 `json:"tool_name,omitempty"`
	ToolInput      map[string]interface{} `json:"tool_input,omitempty"`
	ToolResponse   map[string]interface{} `json:"tool_response,omitempty"`
	SessionID      string                 `json:"session_id"`
	DurationMs     int64                  `json:"duration_ms,omitempty"` // ML telemetry: execution duration
	InputTokens    int64                  `json:"input_tokens,omitempty"` // ML telemetry: LLM input tokens
	OutputTokens   int64                  `json:"output_tokens,omitempty"` // ML telemetry: LLM output tokens
	SequenceIndex  int64                  `json:"sequence_index,omitempty"` // ML telemetry: sequence position
	DecisionID     string                 `json:"decision_id,omitempty"` // ML telemetry: unique decision identifier
	AgentID        string                 `json:"agent_id,omitempty"` // ML telemetry: agent identifier
	RawJSON        json.RawMessage        `json:"-"` // Preserve original JSON
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

// CompareMLFields compares ML telemetry fields between two events
func CompareMLFields(goEvent, bashEvent *EventEntry) []string {
	var diffs []string

	// Compare token counts
	if goEvent.InputTokens != bashEvent.InputTokens {
		diffs = append(diffs, fmt.Sprintf("InputTokens: Go=%d, Bash=%d", goEvent.InputTokens, bashEvent.InputTokens))
	}

	if goEvent.OutputTokens != bashEvent.OutputTokens {
		diffs = append(diffs, fmt.Sprintf("OutputTokens: Go=%d, Bash=%d", goEvent.OutputTokens, bashEvent.OutputTokens))
	}

	// Compare duration (allow 10% tolerance for timing variations)
	goDuration := goEvent.DurationMs
	bashDuration := bashEvent.DurationMs
	if goDuration > 0 && bashDuration > 0 {
		tolerance := bashDuration / 10 // 10% tolerance
		if goDuration < bashDuration-tolerance || goDuration > bashDuration+tolerance {
			diffs = append(diffs, fmt.Sprintf("DurationMs: Go=%d, Bash=%d (diff: %d ms)", goDuration, bashDuration, goDuration-bashDuration))
		}
	}

	// Compare sequence index
	if goEvent.SequenceIndex != bashEvent.SequenceIndex {
		diffs = append(diffs, fmt.Sprintf("SequenceIndex: Go=%d, Bash=%d", goEvent.SequenceIndex, bashEvent.SequenceIndex))
	}

	// Compare decision ID
	if goEvent.DecisionID != bashEvent.DecisionID && goEvent.DecisionID != "" && bashEvent.DecisionID != "" {
		diffs = append(diffs, fmt.Sprintf("DecisionID: Go=%s, Bash=%s", goEvent.DecisionID, bashEvent.DecisionID))
	}

	// Compare agent ID
	if goEvent.AgentID != bashEvent.AgentID && goEvent.AgentID != "" && bashEvent.AgentID != "" {
		diffs = append(diffs, fmt.Sprintf("AgentID: Go=%s, Bash=%s", goEvent.AgentID, bashEvent.AgentID))
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

func TestCompareMLFields(t *testing.T) {
	goEvent := &EventEntry{
		SessionID:     "test-123",
		InputTokens:   1500,
		OutputTokens:  800,
		DurationMs:    250,
		SequenceIndex: 1,
		DecisionID:    "dec-001",
		AgentID:       "agent-a",
	}

	bashEvent := &EventEntry{
		SessionID:     "test-123",
		InputTokens:   1500,
		OutputTokens:  800,
		DurationMs:    255,
		SequenceIndex: 1,
		DecisionID:    "dec-001",
		AgentID:       "agent-a",
	}

	diffs := CompareMLFields(goEvent, bashEvent)
	if len(diffs) != 0 {
		t.Errorf("Expected no diffs with timing tolerance, got: %v", diffs)
	}

	// Test with token difference
	bashEvent.InputTokens = 1600
	diffs = CompareMLFields(goEvent, bashEvent)
	if len(diffs) == 0 {
		t.Error("Expected diffs for token mismatch, got none")
	}

	// Test with sequence index difference
	bashEvent.InputTokens = 1500
	bashEvent.SequenceIndex = 2
	diffs = CompareMLFields(goEvent, bashEvent)
	if len(diffs) == 0 {
		t.Error("Expected diffs for sequence index mismatch, got none")
	}
}
```

**Acceptance Criteria**:
- [x] `NewTestHarness()` loads corpus file from GOgent-000
- [x] `LoadCorpus()` parses all 100+ events from JSONL with ML telemetry fields
- [x] `FilterEvents()` returns events matching hook name
- [x] `RunHook()` executes binary with event JSON as STDIN
- [x] `RunHook()` captures stdout, stderr, exit code, duration
- [x] EventEntry struct includes 6 ML telemetry fields: DurationMs, InputTokens, OutputTokens, SequenceIndex, DecisionID, AgentID
- [x] All ML telemetry fields have proper json tags for marshaling/unmarshaling
- [x] `CompareResults()` identifies differences between Go and Bash outputs
- [x] `CompareMLFields()` compares ML telemetry with 10% tolerance for timing variations
- [x] `CompareMLFields()` returns empty slice for matching events, differences for mismatches
- [x] `PrintSummary()` displays pass/fail statistics
- [x] All tests pass: `go test ./test/integration -v` including TestCompareMLFields
- [x] ML field validation integrated into test harness verification workflow

**Why This Matters**: Foundation for all integration and regression tests. Enables automated comparison of Go vs Bash behavior across 100+ real events.

---
