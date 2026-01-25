package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EventEntry represents a single event from the corpus JSONL
type EventEntry struct {
	Timestamp      int64                  `json:"timestamp"`
	HookEventName  string                 `json:"hook_event_name"`
	ToolName       string                 `json:"tool_name,omitempty"`
	ToolInput      map[string]interface{} `json:"tool_input,omitempty"`
	ToolResponse   map[string]interface{} `json:"tool_response,omitempty"`
	SessionID      string                 `json:"session_id"`
	DurationMs     int64                  `json:"duration_ms,omitempty"`     // ML telemetry: execution duration
	InputTokens    int64                  `json:"input_tokens,omitempty"`    // ML telemetry: LLM input tokens
	OutputTokens   int64                  `json:"output_tokens,omitempty"`   // ML telemetry: LLM output tokens
	SequenceIndex  int64                  `json:"sequence_index,omitempty"`  // ML telemetry: sequence position
	DecisionID     string                 `json:"decision_id,omitempty"`     // ML telemetry: unique decision identifier
	AgentID        string                 `json:"agent_id,omitempty"`        // ML telemetry: agent identifier
	RawJSON        json.RawMessage        `json:"-"`                         // Preserve original JSON
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
		return fmt.Errorf("[harness] Corpus is empty. Expected events from GOgent-000.")
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
		"GOGENT_PROJECT_DIR="+h.ProjectDir,
		"GOGENT_STORAGE_PATH="+filepath.Join(h.ProjectDir, ".gogent", "failure-tracker.jsonl"),
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
	// FIXED: Use integer arithmetic that won't truncate to zero
	goDuration := goEvent.DurationMs
	bashDuration := bashEvent.DurationMs
	if goDuration > 0 && bashDuration > 0 {
		tolerance := (bashDuration * 10) / 100 // 10% tolerance, multiply first to avoid truncation
		if tolerance < 1 {
			tolerance = 1 // Minimum 1ms tolerance for very fast operations
		}
		if goDuration < bashDuration-tolerance || goDuration > bashDuration+tolerance {
			diffs = append(diffs, fmt.Sprintf("DurationMs: Go=%d, Bash=%d (diff: %d ms)", goDuration, bashDuration, goDuration-bashDuration))
		}
	}

	// Compare sequence index
	if goEvent.SequenceIndex != bashEvent.SequenceIndex {
		diffs = append(diffs, fmt.Sprintf("SequenceIndex: Go=%d, Bash=%d", goEvent.SequenceIndex, bashEvent.SequenceIndex))
	}

	// Compare decision ID (flag if either is non-empty and they differ)
	if goEvent.DecisionID != bashEvent.DecisionID {
		// Only flag as difference if at least one is non-empty
		if goEvent.DecisionID != "" || bashEvent.DecisionID != "" {
			diffs = append(diffs, fmt.Sprintf("DecisionID: Go=%s, Bash=%s", goEvent.DecisionID, bashEvent.DecisionID))
		}
	}

	// Compare agent ID (flag if either is non-empty and they differ)
	if goEvent.AgentID != bashEvent.AgentID {
		// Only flag as difference if at least one is non-empty
		if goEvent.AgentID != "" || bashEvent.AgentID != "" {
			diffs = append(diffs, fmt.Sprintf("AgentID: Go=%s, Bash=%s", goEvent.AgentID, bashEvent.AgentID))
		}
	}

	return diffs
}

// PrintSummary prints test results summary
func PrintSummary(results []*HookResult) {
	total := len(results)

	// Guard against division by zero
	if total == 0 {
		fmt.Printf("\n=== Test Summary ===\n")
		fmt.Printf("No results to summarize\n")
		fmt.Printf("====================\n")
		return
	}

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
