package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestNewTestHarness_MissingFile(t *testing.T) {
	_, err := NewTestHarness("/nonexistent/corpus.jsonl", "/tmp/project")
	if err == nil {
		t.Error("Expected error for missing corpus file, got nil")
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

func TestLoadCorpus_EmptyFile(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "empty-corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(""), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	err := harness.LoadCorpus()
	if err == nil {
		t.Error("Expected error for empty corpus, got nil")
	}
}

func TestLoadCorpus_MalformedJSON(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "malformed-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1"}
{this is not valid json}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	err := harness.LoadCorpus()
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}

func TestLoadCorpus_BlankLines(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "blank-lines-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1"}

{"hook_event_name":"PostToolUse","session_id":"test-2"}

{"hook_event_name":"PreToolUse","session_id":"test-3"}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	err := harness.LoadCorpus()
	if err != nil {
		t.Fatalf("Failed to load corpus with blank lines: %v", err)
	}

	if len(harness.Events) != 3 {
		t.Errorf("Expected 3 events (blank lines ignored), got: %d", len(harness.Events))
	}
}

func TestLoadCorpus_MLTelemetryFields(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "ml-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1","duration_ms":250,"input_tokens":1500,"output_tokens":800,"sequence_index":1,"decision_id":"dec-001","agent_id":"agent-a"}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	err := harness.LoadCorpus()
	if err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	event := harness.Events[0]
	if event.DurationMs != 250 {
		t.Errorf("Expected DurationMs=250, got: %d", event.DurationMs)
	}
	if event.InputTokens != 1500 {
		t.Errorf("Expected InputTokens=1500, got: %d", event.InputTokens)
	}
	if event.OutputTokens != 800 {
		t.Errorf("Expected OutputTokens=800, got: %d", event.OutputTokens)
	}
	if event.SequenceIndex != 1 {
		t.Errorf("Expected SequenceIndex=1, got: %d", event.SequenceIndex)
	}
	if event.DecisionID != "dec-001" {
		t.Errorf("Expected DecisionID=dec-001, got: %s", event.DecisionID)
	}
	if event.AgentID != "agent-a" {
		t.Errorf("Expected AgentID=agent-a, got: %s", event.AgentID)
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

func TestFilterEvents_NoMatches(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "test-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1"}
{"hook_event_name":"PostToolUse","session_id":"test-2"}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	harness.LoadCorpus()

	filtered := harness.FilterEvents("NonExistentHook")
	if len(filtered) != 0 {
		t.Errorf("Expected 0 events, got: %d", len(filtered))
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

func TestCompareResults_ExitCodeDifference(t *testing.T) {
	goResult := &HookResult{
		ExitCode:   0,
		ParsedJSON: map[string]interface{}{},
	}

	bashResult := &HookResult{
		ExitCode:   1,
		ParsedJSON: map[string]interface{}{},
	}

	diffs := CompareResults(goResult, bashResult)
	if len(diffs) != 1 {
		t.Errorf("Expected 1 diff for exit code, got: %d", len(diffs))
	}
}

func TestCompareResults_ReasonDifference(t *testing.T) {
	goResult := &HookResult{
		ExitCode: 0,
		ParsedJSON: map[string]interface{}{
			"decision": "allow",
			"reason":   "Reason A",
		},
	}

	bashResult := &HookResult{
		ExitCode: 0,
		ParsedJSON: map[string]interface{}{
			"decision": "allow",
			"reason":   "Reason B",
		},
	}

	diffs := CompareResults(goResult, bashResult)
	if len(diffs) != 1 {
		t.Errorf("Expected 1 diff for reason, got: %d", len(diffs))
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

func TestCompareMLFields_TimingTolerance(t *testing.T) {
	goEvent := &EventEntry{
		DurationMs: 100,
	}

	bashEvent := &EventEntry{
		DurationMs: 110, // 10% difference from bashEvent = tolerance of 11ms, range [99, 121]
	}

	diffs := CompareMLFields(goEvent, bashEvent)
	if len(diffs) != 0 {
		t.Errorf("Expected no diffs within 10%% tolerance, got: %v", diffs)
	}

	// Test clearly outside tolerance
	// bashDuration=110, tolerance=11, range=[99, 121]
	// goDuration=125 is beyond upper bound
	goEvent.DurationMs = 125
	diffs = CompareMLFields(goEvent, bashEvent)
	if len(diffs) == 0 {
		t.Error("Expected diffs beyond 10% tolerance, got none")
	}
}

func TestCompareMLFields_SmallDurations(t *testing.T) {
	goEvent := &EventEntry{
		DurationMs: 5,
	}

	bashEvent := &EventEntry{
		DurationMs: 6, // Small durations should have minimum 1ms tolerance
	}

	diffs := CompareMLFields(goEvent, bashEvent)
	if len(diffs) != 0 {
		t.Errorf("Expected no diffs with minimum 1ms tolerance, got: %v", diffs)
	}

	// Test beyond minimum tolerance
	bashEvent.DurationMs = 7
	diffs = CompareMLFields(goEvent, bashEvent)
	if len(diffs) == 0 {
		t.Error("Expected diffs beyond minimum tolerance, got none")
	}
}

func TestCompareMLFields_EmptyStringHandling(t *testing.T) {
	goEvent := &EventEntry{
		DecisionID: "",
		AgentID:    "",
	}

	bashEvent := &EventEntry{
		DecisionID: "",
		AgentID:    "",
	}

	// Both empty - should not flag as difference
	diffs := CompareMLFields(goEvent, bashEvent)
	if len(diffs) != 0 {
		t.Errorf("Expected no diffs when both empty, got: %v", diffs)
	}

	// One empty, one populated - should flag
	bashEvent.DecisionID = "dec-123"
	diffs = CompareMLFields(goEvent, bashEvent)
	if len(diffs) == 0 {
		t.Error("Expected diff when one DecisionID is empty and one is populated")
	}
}

func TestCompareMLFields_AllFields(t *testing.T) {
	goEvent := &EventEntry{
		InputTokens:   1000,
		OutputTokens:  500,
		DurationMs:    100,
		SequenceIndex: 5,
		DecisionID:    "dec-a",
		AgentID:       "agent-x",
	}

	bashEvent := &EventEntry{
		InputTokens:   2000, // Different
		OutputTokens:  600,  // Different
		DurationMs:    150,  // Different but within tolerance
		SequenceIndex: 6,    // Different
		DecisionID:    "dec-b", // Different
		AgentID:       "agent-y", // Different
	}

	diffs := CompareMLFields(goEvent, bashEvent)

	// Should flag: InputTokens, OutputTokens, SequenceIndex, DecisionID, AgentID
	// Should NOT flag: DurationMs (within 10% tolerance: 100 * 1.5 = 150, tolerance is ±10)
	// Actually 150 is 50% more than 100, so it should be flagged
	expectedDiffs := 6
	if len(diffs) != expectedDiffs {
		t.Errorf("Expected %d diffs, got: %d. Diffs: %v", expectedDiffs, len(diffs), diffs)
	}
}

func TestPrintSummary_EmptyResults(t *testing.T) {
	// Should not panic on empty results
	results := []*HookResult{}
	PrintSummary(results) // Should print "No results to summarize"
}

func TestPrintSummary_MixedResults(t *testing.T) {
	results := []*HookResult{
		{
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
			Error:    nil,
		},
		{
			ExitCode: 1,
			Duration: 150 * time.Millisecond,
			Error:    nil,
		},
		{
			ExitCode: 0,
			Duration: 200 * time.Millisecond,
			Error:    nil,
		},
	}

	// Should print summary without panic
	PrintSummary(results)
}

func TestRunHook_RawJSONPreservation(t *testing.T) {
	tmpCorpus := filepath.Join(t.TempDir(), "test-corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1","custom_field":"preserve_me"}`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, "/tmp/project")
	harness.LoadCorpus()

	event := harness.Events[0]
	if len(event.RawJSON) == 0 {
		t.Error("Expected RawJSON to be preserved, got empty")
	}

	// Verify RawJSON contains the original content
	if string(event.RawJSON) != content {
		t.Errorf("Expected RawJSON to match original, got: %s", string(event.RawJSON))
	}
}

func TestRunHook_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple test hook that echoes JSON
	hookScript := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo '{"decision":"allow","reason":"test passed"}'
exit 0
`
	os.WriteFile(hookScript, []byte(scriptContent), 0755)

	// Create corpus with event
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	result := harness.RunHook(hookScript, harness.Events[0])

	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", result.ExitCode)
	}

	if result.ParsedJSON == nil {
		t.Fatal("Expected parsed JSON, got nil")
	}

	if result.ParsedJSON["decision"] != "allow" {
		t.Errorf("Expected decision=allow, got: %v", result.ParsedJSON["decision"])
	}
}

func TestRunHook_NonZeroExit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook that exits with non-zero
	hookScript := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo '{"decision":"block","reason":"test failed"}'
exit 1
`
	os.WriteFile(hookScript, []byte(scriptContent), 0755)

	// Create corpus with event
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	result := harness.RunHook(hookScript, harness.Events[0])

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got: %d", result.ExitCode)
	}

	if result.ParsedJSON["decision"] != "block" {
		t.Errorf("Expected decision=block, got: %v", result.ParsedJSON["decision"])
	}
}

func TestRunHook_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook that outputs invalid JSON
	hookScript := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo 'not valid json'
exit 0
`
	os.WriteFile(hookScript, []byte(scriptContent), 0755)

	// Create corpus with event
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	result := harness.RunHook(hookScript, harness.Events[0])

	if result.Error == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if result.ParsedJSON != nil {
		t.Error("Expected ParsedJSON to be nil for invalid JSON")
	}
}

func TestRunHook_MissingBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create corpus with event
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	result := harness.RunHook("/nonexistent/hook", harness.Events[0])

	if result.Error == nil {
		t.Error("Expected error for missing binary, got nil")
	}
}

func TestRunHookBatch_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple test hook
	hookScript := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo '{"decision":"allow","reason":"batch test"}'
exit 0
`
	os.WriteFile(hookScript, []byte(scriptContent), 0755)

	// Create corpus with multiple events
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	content := `{"hook_event_name":"PreToolUse","session_id":"test-1"}
{"hook_event_name":"PreToolUse","session_id":"test-2"}
{"hook_event_name":"PostToolUse","session_id":"test-3"}
`
	os.WriteFile(tmpCorpus, []byte(content), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	results, err := harness.RunHookBatch(hookScript, "PreToolUse")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got: %d", len(results))
	}

	for _, result := range results {
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got: %d", result.ExitCode)
		}
	}
}

func TestRunHookBatch_MissingBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create corpus with event
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	_, err := harness.RunHookBatch("/nonexistent/hook", "PreToolUse")
	if err == nil {
		t.Error("Expected error for missing binary, got nil")
	}
}

func TestRunHookBatch_NoMatchingEvents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test hook
	hookScript := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo '{}'
exit 0
`
	os.WriteFile(hookScript, []byte(scriptContent), 0755)

	// Create corpus with events that don't match
	tmpCorpus := filepath.Join(tmpDir, "corpus.jsonl")
	os.WriteFile(tmpCorpus, []byte(`{"hook_event_name":"PreToolUse","session_id":"test-1"}`), 0644)

	harness, _ := NewTestHarness(tmpCorpus, tmpDir)
	harness.LoadCorpus()

	_, err := harness.RunHookBatch(hookScript, "NonExistentHook")
	if err == nil {
		t.Error("Expected error for no matching events, got nil")
	}
}
