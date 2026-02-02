# Week 5: Advanced Enforcement (orchestrator-guard + doc-theater)

**File**: `10-week5-advanced-enforcement.md`
**Tickets**: GOgent-075 to 086 (12 tickets)
**Total Time**: ~18 hours
**Phase**: Week 5 (concurrent with week 4)

---

## Navigation

- **Previous**: [07-week3-agent-workflow-hooks.md](07-week3-agent-workflow-hooks.md) - GOgent-063 to 074
- **Next**: [09-week4-observability-remaining.md](09-week4-observability-remaining.md) - GOgent-087 to 093
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure
- **Untracked Hooks**: [UNTRACKED_HOOKS.md](UNTRACKED_HOOKS.md) - Hook inventory and planning

---

## Summary

This week translates `orchestrator-completion-guard.sh` and `detect-documentation-theater.sh` hooks from Bash to Go:

### orchestrator-completion-guard (Tickets 075-080)
1. **SubagentStop Event Parsing**: Detect orchestrator/architect completion
2. **Transcript Analysis**: Scan for `run_in_background` spawns
3. **Task Tracking**: Count TaskOutput collections
4. **Blocking Response**: Block completion if tasks uncollected
5. **Integration Tests**: Comprehensive test coverage
6. **CLI Build**: Create gogent-orchestrator-guard binary

### detect-documentation-theater (Tickets 081-086)
1. **PreToolUse Event Parsing**: Detect Write/Edit on CLAUDE.md files
2. **Content Scanning**: Look for enforcement patterns (MUST NOT, BLOCKED, NEVER)
3. **Pattern Matching**: Identify documentation theater anti-pattern
4. **Warning Response**: Inject warning without blocking
5. **Integration Tests**: Comprehensive test coverage
6. **CLI Build**: Create gogent-doc-theater binary

**Critical Dependencies**:
- GOgent-063 (SubagentStop parsing)
- GOgent-069 (PostToolUse parsing pattern)
- Routing schema

**Hook Triggers**:
- `orchestrator-completion-guard`: SubagentStop (specifically orchestrator/architect)
- `detect-documentation-theater`: PreToolUse (specifically Write/Edit on CLAUDE.md)

---

## Part 1: Orchestrator-Completion-Guard (GOgent-075 to 080)

### GOgent-075: SubagentStop Event Parsing for Orchestrator

**Time**: 1.5 hours
**Dependencies**: GOgent-063

**Task**:
Parse SubagentStop events specifically for orchestrator/architect agents.

**File**: `pkg/enforcement/orchestrator_events.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// OrchestratorStopEvent represents orchestrator completion
type OrchestratorStopEvent struct {
	Type          string `json:"type"`           // "stop"
	HookEventName string `json:"hook_event_name"` // "SubagentStop"
	AgentID       string `json:"agent_id"`       // "orchestrator", "architect"
	AgentModel    string `json:"agent_model"`
	ExitCode      int    `json:"exit_code"`
	TranscriptPath string `json:"transcript_path"` // Path to agent output
	Duration      int    `json:"duration_ms"`
	OutputTokens  int    `json:"output_tokens"`
}

// IsOrchestratorType checks if this is an orchestrator/architect agent
func (e *OrchestratorStopEvent) IsOrchestratorType() bool {
	return e.AgentID == "orchestrator" || e.AgentID == "architect"
}

// ParseOrchestratorStopEvent reads and validates orchestrator stop event
func ParseOrchestratorStopEvent(r io.Reader, timeout time.Duration) (*OrchestratorStopEvent, error) {
	type result struct {
		event *OrchestratorStopEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[orchestrator-guard] Failed to read STDIN: %w", err)}
			return
		}

		var event OrchestratorStopEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[orchestrator-guard] Failed to parse JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[orchestrator-guard] STDIN read timeout after %v", timeout)
	}
}
```

**Tests**: `pkg/enforcement/orchestrator_events_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestParseOrchestratorStopEvent(t *testing.T) {
	jsonInput := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"agent_model": "sonnet",
		"exit_code": 0,
		"transcript_path": "/tmp/transcript.md",
		"duration_ms": 5000,
		"output_tokens": 2048
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseOrchestratorStopEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !event.IsOrchestratorType() {
		t.Error("Should identify as orchestrator type")
	}
}

func TestIsOrchestratorType(t *testing.T) {
	tests := []struct {
		agentID    string
		isOrch     bool
	}{
		{"orchestrator", true},
		{"architect", true},
		{"python-pro", false},
		{"code-reviewer", false},
	}

	for _, tc := range tests {
		event := &OrchestratorStopEvent{AgentID: tc.agentID}
		if got := event.IsOrchestratorType(); got != tc.isOrch {
			t.Errorf("AgentID %s: expected %v, got %v", tc.agentID, tc.isOrch, got)
		}
	}
}
```

**Acceptance Criteria**:
- [ ] `ParseOrchestratorStopEvent()` reads SubagentStop events
- [ ] `IsOrchestratorType()` correctly identifies orchestrator/architect
- [ ] Implements 5s timeout
- [ ] Tests verify parsing and type detection
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Orchestrator-guard only activates for orchestrator/architect agents.

---

### GOgent-076: Transcript Analysis & Task Tracking

**Time**: 2 hours
**Dependencies**: GOgent-075

**Task**:
Scan transcript for background task spawns and collections.

**File**: `pkg/enforcement/transcript_analyzer.go`

**Imports**:
```go
package enforcement

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)
```

**Implementation**:
```go
// TaskTracker tracks spawned and collected background tasks
type TaskTracker struct {
	SpawnedCount    int
	CollectedCount  int
	SpawnedTasks    []string
	UncollectedIDs  []string
}

// TranscriptAnalyzer scans transcript for task patterns
type TranscriptAnalyzer struct {
	filePath string
	tracker  *TaskTracker
}

// NewTranscriptAnalyzer creates analyzer instance
func NewTranscriptAnalyzer(transcriptPath string) *TranscriptAnalyzer {
	return &TranscriptAnalyzer{
		filePath: transcriptPath,
		tracker: &TaskTracker{
			SpawnedTasks: []string{},
		},
	}
}

// Analyze scans transcript for background tasks
func (ta *TranscriptAnalyzer) Analyze() error {
	file, err := os.Open(ta.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No transcript - assume no background tasks
			return nil
		}
		return fmt.Errorf("[orchestrator-guard] Failed to open transcript: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNum int

	// Patterns to match
	spawnPattern := regexp.MustCompile(`(?i)run_in_background[:\s=]+true|spawn.*task|background.*task`)
	taskIdPattern := regexp.MustCompile(`"task_id"\s*:\s*"([^"]+)"`)
	collectPattern := regexp.MustCompile(`TaskOutput.*task_id|collecting.*task|await.*task`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for spawn patterns
		if spawnPattern.MatchString(line) {
			ta.tracker.SpawnedCount++

			// Try to extract task ID if present
			matches := taskIdPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				ta.tracker.SpawnedTasks = append(ta.tracker.SpawnedTasks, matches[1])
			}
		}

		// Check for collection patterns
		if collectPattern.MatchString(line) {
			ta.tracker.CollectedCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("[orchestrator-guard] Error reading transcript: %w", err)
	}

	// Calculate uncollected tasks
	uncollected := ta.tracker.SpawnedCount - ta.tracker.CollectedCount
	if uncollected > 0 {
		ta.tracker.UncollectedIDs = make([]string, uncollected)
		for i := 0; i < uncollected && i < len(ta.tracker.SpawnedTasks); i++ {
			ta.tracker.UncollectedIDs[i] = ta.tracker.SpawnedTasks[i]
		}
	}

	return nil
}

// HasUncollectedTasks checks if there are uncollected background tasks
func (ta *TranscriptAnalyzer) HasUncollectedTasks() bool {
	return ta.tracker.SpawnedCount > ta.tracker.CollectedCount
}

// GetSummary returns analysis summary
func (ta *TranscriptAnalyzer) GetSummary() string {
	if ta.tracker.SpawnedCount == 0 {
		return "No background tasks detected"
	}

	return fmt.Sprintf(
		"Background tasks: %d spawned, %d collected, %d uncollected",
		ta.tracker.SpawnedCount,
		ta.tracker.CollectedCount,
		ta.tracker.SpawnedCount - ta.tracker.CollectedCount,
	)
}

// GetUncollectedList returns formatted list of uncollected tasks
func (ta *TranscriptAnalyzer) GetUncollectedList() string {
	if len(ta.tracker.UncollectedIDs) == 0 {
		return ""
	}

	var list strings.Builder
	list.WriteString("Uncollected tasks:\n")
	for i, id := range ta.tracker.UncollectedIDs {
		if id != "" {
			list.WriteString(fmt.Sprintf("  %d. %s\n", i+1, id))
		}
	}

	return list.String()
}
```

**Tests**: `pkg/enforcement/transcript_analyzer_test.go`

```go
package enforcement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptAnalyzer_NoBackgroundTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript without background tasks
	content := `# Transcript

Executed some direct operations.
No background tasks here.
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	err := analyzer.Analyze()

	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Should not detect background tasks")
	}

	if analyzer.tracker.SpawnedCount != 0 {
		t.Errorf("Expected 0 spawned, got: %d", analyzer.tracker.SpawnedCount)
	}
}

func TestTranscriptAnalyzer_SpawnAndCollect(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with spawn and collection
	content := `# Transcript

Bash({..., run_in_background: true, task_id: "bg-1"})
Bash({..., run_in_background: true, task_id: "bg-2"})

... do other work ...

TaskOutput({task_id: "bg-1", block: true})
TaskOutput({task_id: "bg-2", block: true})
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	if analyzer.HasUncollectedTasks() {
		t.Error("Should not have uncollected tasks when all collected")
	}

	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "2 spawned") {
		t.Errorf("Summary should mention 2 spawned, got: %s", summary)
	}
}

func TestTranscriptAnalyzer_UncollectedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with uncollected tasks
	content := `# Transcript

Bash({..., run_in_background: true, task_id: "bg-1"})
Bash({..., run_in_background: true, task_id: "bg-2"})
Bash({..., run_in_background: true, task_id: "bg-3"})

TaskOutput({task_id: "bg-1", block: true})
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	if !analyzer.HasUncollectedTasks() {
		t.Error("Should detect uncollected tasks")
	}

	if analyzer.tracker.SpawnedCount != 3 {
		t.Errorf("Expected 3 spawned, got: %d", analyzer.tracker.SpawnedCount)
	}

	if analyzer.tracker.CollectedCount != 1 {
		t.Errorf("Expected 1 collected, got: %d", analyzer.tracker.CollectedCount)
	}

	list := analyzer.GetUncollectedList()
	if !strings.Contains(list, "Uncollected") {
		t.Error("List should indicate uncollected tasks")
	}
}

func TestTranscriptAnalyzer_MissingFile(t *testing.T) {
	analyzer := NewTranscriptAnalyzer("/nonexistent/path.md")
	err := analyzer.Analyze()

	// Should not error on missing file
	if err == nil {
		t.Fatal("Expected error for missing file")
	}
}
```

**Acceptance Criteria**:
- [ ] `NewTranscriptAnalyzer()` creates analyzer for transcript
- [ ] `Analyze()` scans for run_in_background patterns
- [ ] Counts spawned and collected tasks
- [ ] `HasUncollectedTasks()` returns true if spawn > collect
- [ ] `GetSummary()` returns task count summary
- [ ] `GetUncollectedList()` lists uncollected task IDs
- [ ] Tests verify no tasks, full collection, partial collection
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Transcript analysis detects orphaned background tasks that would otherwise fail silently.

---

### GOgent-077: Blocking Response Generation

**Time**: 1.5 hours
**Dependencies**: GOgent-076

**Task**:
Generate blocking response if background tasks uncollected.

**File**: `pkg/enforcement/blocking_response.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"strings"
)
```

**Implementation**:
```go
// GuardResponse represents orchestrator-guard decision
type GuardResponse struct {
	HookEventName     string `json:"hookEventName"`
	Decision          string `json:"decision"` // "allow" or "block"
	Reason            string `json:"reason"`
	AdditionalContext string `json:"additionalContext"`
	RemediationSteps  []string `json:"remediation_steps"`
}

// GenerateGuardResponse decides whether to allow orchestrator completion
func GenerateGuardResponse(analyzer *TranscriptAnalyzer, event *OrchestratorStopEvent) *GuardResponse {
	response := &GuardResponse{
		HookEventName: "SubagentStop",
	}

	// If no background tasks, allow completion
	if !analyzer.HasUncollectedTasks() {
		response.Decision = "allow"
		response.Reason = "No uncollected background tasks detected"
		response.AdditionalContext = fmt.Sprintf(
			"✅ ORCHESTRATOR COMPLETION ALLOWED\n\n"+
				"Agent: %s\n"+
				"Summary: %s\n\n"+
				"Safe to proceed with next steps.",
			event.AgentID,
			analyzer.GetSummary(),
		)
		return response
	}

	// Uncollected tasks - BLOCK
	response.Decision = "block"
	response.Reason = "Background tasks not collected"
	response.AdditionalContext = fmt.Sprintf(
		"🛑 ORCHESTRATOR COMPLETION BLOCKED\n\n"+
			"Agent: %s\nStatus: %s\n\n"+
			"VIOLATION: Background task fan-out/fan-in pattern not completed.\n\n"+
			"%s\n\n"+
			"From LLM-guidelines.md § 2.2 MANDATORY: Background Task Collection:\n"+
			"\"If you spawn background tasks, you MUST call TaskOutput() before concluding.\"\n\n"+
			"REQUIRED ACTIONS:\n"+
			"1. Spawn any missing TaskOutput calls\n"+
			"2. Use task_id from uncollected tasks above\n"+
			"3. Set block: true to wait for results\n"+
			"4. Proceed once all TaskOutput calls complete",
		event.AgentID,
		analyzer.GetSummary(),
		analyzer.GetUncollectedList(),
	)
	response.RemediationSteps = []string{
		"identify_uncollected_task_ids",
		"call_TaskOutput_for_each",
		"wait_for_all_collections",
		"verify_results_in_transcript",
	}

	return response
}

// FormatResponseJSON creates hook response format
func (r *GuardResponse) FormatJSON() string {
	remedJson := formatStringArray(r.RemediationSteps)

	output := fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "%s",
    "decision": "%s",
    "reason": "%s",
    "additionalContext": "%s",
    "remediation": %s
  }
}`,
		r.HookEventName,
		r.Decision,
		escapeJSON(r.Reason),
		escapeJSON(r.AdditionalContext),
		remedJson,
	)

	return output
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func formatStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	var quoted []string
	for _, item := range arr {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, item))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
```

**Tests**: `pkg/enforcement/blocking_response_test.go`

```go
package enforcement

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateGuardResponse_AllowCompletion(t *testing.T) {
	// No uncollected tasks
	analyzer := &TranscriptAnalyzer{
		tracker: &TaskTracker{
			SpawnedCount:   0,
			CollectedCount: 0,
		},
	}

	event := &OrchestratorStopEvent{
		AgentID: "orchestrator",
	}

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Errorf("Expected allow, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "ALLOWED") {
		t.Error("Should indicate completion allowed")
	}
}

func TestGenerateGuardResponse_BlockCompletion(t *testing.T) {
	// Uncollected tasks
	analyzer := &TranscriptAnalyzer{
		tracker: &TaskTracker{
			SpawnedCount:    3,
			CollectedCount:  1,
			UncollectedIDs:  []string{"bg-2", "bg-3"},
		},
	}

	event := &OrchestratorStopEvent{
		AgentID: "orchestrator",
	}

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "block" {
		t.Errorf("Expected block, got: %s", response.Decision)
	}

	if !strings.Contains(response.AdditionalContext, "BLOCKED") {
		t.Error("Should indicate completion blocked")
	}

	if !strings.Contains(response.AdditionalContext, "2 uncollected") {
		t.Error("Should mention uncollected count")
	}

	if !strings.Contains(response.AdditionalContext, "TaskOutput") {
		t.Error("Should mention TaskOutput fix")
	}
}

func TestFormatResponseJSON_Valid(t *testing.T) {
	response := &GuardResponse{
		HookEventName:    "SubagentStop",
		Decision:         "block",
		Reason:           "Test reason",
		AdditionalContext: "Test context",
		RemediationSteps: []string{"step1", "step2"},
	}

	jsonStr := response.FormatJSON()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if _, ok := parsed["hookSpecificOutput"]; !ok {
		t.Fatal("Missing hookSpecificOutput")
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateGuardResponse()` allows if no uncollected tasks
- [ ] Blocks if uncollected tasks detected
- [ ] Block response includes remediation steps
- [ ] References LLM-guidelines.md fan-out/fan-in pattern
- [ ] `FormatResponseJSON()` outputs valid JSON
- [ ] Tests verify allow and block paths
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Blocking response enforces fan-out/fan-in discipline programmatically.

---

### GOgent-078: Integration Tests for orchestrator-guard

**Time**: 1.5 hours
**Dependencies**: GOgent-077

**Task**:
End-to-end tests for orchestrator-guard workflow.

**File**: `pkg/enforcement/orchestrator_integration_test.go`

```go
package enforcement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOrchestratorGuardWorkflow_AllowCompletion(t *testing.T) {
	// Full workflow: event → analysis → response
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with proper fan-in/fan-out
	content := `# Transcript

Bash({..., run_in_background: true})
Bash({..., run_in_background: true})

TaskOutput({task_id: "bg-1"})
TaskOutput({task_id: "bg-2"})

Workflow complete.
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	// Parse event
	eventJSON := `{
		"type": "stop",
		"hook_event_name": "SubagentStop",
		"agent_id": "orchestrator",
		"transcript_path": "` + transcriptPath + `"
	}`

	event := &OrchestratorStopEvent{
		AgentID:        "orchestrator",
		TranscriptPath: transcriptPath,
	}

	// Analyze
	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	// Generate response
	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Error("Should allow completion when all tasks collected")
	}
}

func TestOrchestratorGuardWorkflow_BlockCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with uncollected background tasks
	content := `# Transcript

Bash({..., run_in_background: true, task_id: "bg-1"})
Bash({..., run_in_background: true, task_id: "bg-2"})

// ERROR: Only one TaskOutput!
TaskOutput({task_id: "bg-1"})

// bg-2 was never collected - VIOLATION
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	event := &OrchestratorStopEvent{
		AgentID:        "orchestrator",
		TranscriptPath: transcriptPath,
	}

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "block" {
		t.Error("Should block completion when tasks uncollected")
	}

	if !strings.Contains(response.AdditionalContext, "bg-2") {
		t.Error("Should mention uncollected task ID")
	}

	jsonResponse := response.FormatJSON()
	if !strings.Contains(jsonResponse, "block") {
		t.Error("JSON response should contain block decision")
	}
}

func TestOrchestratorGuardWorkflow_NoBackgroundTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Create transcript with no background tasks
	content := `# Transcript

Direct task execution.
No background spawning.
Done.
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	event := &OrchestratorStopEvent{
		AgentID:        "architect",
		TranscriptPath: transcriptPath,
	}

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	response := GenerateGuardResponse(analyzer, event)

	if response.Decision != "allow" {
		t.Error("Should allow when no background tasks")
	}
}
```

**Acceptance Criteria**:
- [ ] Full workflow (event → analysis → response) works
- [ ] Allows completion when all background tasks collected
- [ ] Blocks completion when tasks uncollected
- [ ] No false positives on direct (non-background) execution
- [ ] JSON response valid
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Integration tests ensure orchestrator-guard catches real violations.

---

### GOgent-079: Build gogent-orchestrator-guard CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-078

**Task**:
Build CLI binary for orchestrator-completion-guard hook.

**File**: `cmd/gogent-orchestrator-guard/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent/pkg/enforcement"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event
	event, err := enforcement.ParseOrchestratorStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only process orchestrator/architect agents
	if !event.IsOrchestratorType() {
		// Silent pass-through for other agents
		fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow"
  }
}`)
		os.Exit(0)
	}

	// Analyze transcript if available
	var analyzer *enforcement.TranscriptAnalyzer
	if event.TranscriptPath != "" {
		analyzer = enforcement.NewTranscriptAnalyzer(event.TranscriptPath)
		if err := analyzer.Analyze(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to analyze transcript: %v\n", err)
			analyzer = &enforcement.TranscriptAnalyzer{
				tracker: &enforcement.TaskTracker{},
			}
		}
	}

	// Generate guard response
	response := enforcement.GenerateGuardResponse(analyzer, event)

	// Output response
	fmt.Println(response.FormatJSON())
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build Script**: `scripts/build-orchestrator-guard.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-orchestrator-guard..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-orchestrator-guard ./cmd/gogent-orchestrator-guard

echo "✓ Built: bin/gogent-orchestrator-guard"
```

**Acceptance Criteria**:
- [ ] CLI reads SubagentStop events
- [ ] Passes through non-orchestrator agents
- [ ] Analyzes transcript for orchestrator agents
- [ ] Generates block/allow decision
- [ ] Outputs valid hook response
- [ ] Build script creates executable

**Why This Matters**: CLI is orchestrator-completion-guard hook implementation.

---

## Part 2: Detect-Documentation-Theater (GOgent-080 to 086)

### GOgent-080: PreToolUse Event Parsing

**Time**: 1.5 hours
**Dependencies**: GOgent-069 (event parsing pattern)

**Task**:
Parse PreToolUse events for Write/Edit on CLAUDE.md files.

**File**: `pkg/enforcement/doc_events.go`

**Imports**:
```go
package enforcement

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)
```

**Implementation**:
```go
// PreToolUseEvent represents tool usage before execution
type PreToolUseEvent struct {
	Type          string `json:"type"`           // "pre-tool-use"
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`      // "Write", "Edit", etc.
	FilePath      string `json:"file_path"`      // Path being modified
	SessionID     string `json:"session_id"`
}

// IsClaude MDFile checks if file is a CLAUDE.md configuration file
func (e *PreToolUseEvent) IsClaudeMDFile() bool {
	filename := filepath.Base(e.FilePath)
	// Check for CLAUDE.md or variants like CLAUDE.en.md
	if filename == "CLAUDE.md" || strings.HasPrefix(filename, "CLAUDE.") && strings.HasSuffix(filename, ".md") {
		return true
	}
	return false
}

// IsWriteOperation checks if this is a write/edit operation
func (e *PreToolUseEvent) IsWriteOperation() bool {
	return e.ToolName == "Write" || e.ToolName == "Edit"
}

// ParsePreToolUseEvent reads PreToolUse event from STDIN
func ParsePreToolUseEvent(r io.Reader, timeout time.Duration) (*PreToolUseEvent, error) {
	type result struct {
		event *PreToolUseEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[doc-theater] Failed to read STDIN: %w", err)}
			return
		}

		var event PreToolUseEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[doc-theater] Failed to parse JSON: %w", err)}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[doc-theater] STDIN read timeout after %v", timeout)
	}
}
```

**Tests**: `pkg/enforcement/doc_events_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestParsePreToolUseEvent(t *testing.T) {
	jsonInput := `{
		"type": "pre-tool-use",
		"hook_event_name": "PreToolUse",
		"tool_name": "Edit",
		"file_path": "/home/user/.claude/CLAUDE.md",
		"session_id": "sess-123"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParsePreToolUseEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Edit" {
		t.Errorf("Expected Edit, got: %s", event.ToolName)
	}
}

func TestIsClaudeMDFile(t *testing.T) {
	tests := []struct {
		path      string
		isClaude  bool
	}{
		{"/path/to/CLAUDE.md", true},
		{"/path/to/CLAUDE.en.md", true},
		{"./CLAUDE.md", true},
		{"/path/to/other.md", false},
		{"/path/to/CLAUDE.txt", false},
	}

	for _, tc := range tests {
		event := &PreToolUseEvent{FilePath: tc.path}
		if got := event.IsClaudeMDFile(); got != tc.isClaude {
			t.Errorf("File %s: expected %v, got %v", tc.path, tc.isClaude, got)
		}
	}
}

func TestIsWriteOperation(t *testing.T) {
	tests := []struct {
		tool      string
		isWrite   bool
	}{
		{"Write", true},
		{"Edit", true},
		{"Read", false},
		{"Bash", false},
	}

	for _, tc := range tests {
		event := &PreToolUseEvent{ToolName: tc.tool}
		if got := event.IsWriteOperation(); got != tc.isWrite {
			t.Errorf("Tool %s: expected %v, got %v", tc.tool, tc.isWrite, got)
		}
	}
}
```

**Acceptance Criteria**:
- [ ] `ParsePreToolUseEvent()` reads PreToolUse events
- [ ] `IsClaudeMDFile()` detects CLAUDE.md variants
- [ ] `IsWriteOperation()` detects Write/Edit tools
- [ ] Implements 5s timeout
- [ ] Tests verify parsing and detection
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Event filtering is required to target only CLAUDE.md writes.

---

### GOgent-081: Pattern Detection for Documentation Theater

**Time**: 2 hours
**Dependencies**: GOgent-080

**Task**:
Scan content for enforcement patterns that indicate documentation theater.

**File**: `pkg/enforcement/pattern_detector.go`

**Imports**:
```go
package enforcement

import (
	"fmt"
	"regexp"
	"strings"
)
```

**Implementation**:
```go
// EnforcementPattern represents a documentation theater anti-pattern
type EnforcementPattern struct {
	Pattern     string
	Description string
	Severity    string // "warning", "critical"
}

// PatternDetector scans content for enforcement theater
type PatternDetector struct {
	patterns []EnforcementPattern
}

// NewPatternDetector creates detector with known anti-patterns
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: []EnforcementPattern{
			{
				Pattern:     `(?i)\bMUST\s+NOT\b`,
				Description: "Imperative enforcement without programmatic backing",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bBLOCKED\b.*\(.*\)`,
				Description: "Claims of blocking without hook enforcement",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bNEVER\s+use\b`,
				Description: "NEVER language without validation hook",
				Severity:    "critical",
			},
			{
				Pattern:     `(?i)\bFORBIDDEN\b`,
				Description: "Forbidden declarations without enforcement",
				Severity:    "warning",
			},
			{
				Pattern:     `(?i)\bYOU\s+CANNOT\b`,
				Description: "Prohibition without mechanism",
				Severity:    "warning",
			},
		},
	}
}

// Detect scans content for anti-patterns
func (pd *PatternDetector) Detect(content string) []DetectionResult {
	var results []DetectionResult

	for _, ep := range pd.patterns {
		regex := regexp.MustCompile(ep.Pattern)
		matches := regex.FindAllStringIndex(content, -1)

		if len(matches) > 0 {
			results = append(results, DetectionResult{
				Pattern:     ep.Pattern,
				Description: ep.Description,
				Severity:    ep.Severity,
				MatchCount:  len(matches),
				FirstMatch:  content[matches[0][0]:matches[0][1]],
			})
		}
	}

	return results
}

// HasDocumentationTheater checks if critical patterns found
func (pd *PatternDetector) HasDocumentationTheater(content string) bool {
	results := pd.Detect(content)
	for _, result := range results {
		if result.Severity == "critical" {
			return true
		}
	}
	return false
}

// DetectionResult represents a found anti-pattern
type DetectionResult struct {
	Pattern     string
	Description string
	Severity    string
	MatchCount  int
	FirstMatch  string
}

// GenerateWarning creates warning message for detected patterns
func GenerateWarning(results []DetectionResult, filename string) string {
	if len(results) == 0 {
		return ""
	}

	var warning strings.Builder
	warning.WriteString(fmt.Sprintf(
		"⚠️ DOCUMENTATION THEATER DETECTED in %s\n\n"+
			"Found %d enforcement pattern(s) without programmatic backing:\n\n",
		filename,
		len(results),
	))

	for i, result := range results {
		warning.WriteString(fmt.Sprintf(
			"%d. %s\n"+
				"   Pattern: %s\n"+
				"   Found: %q\n"+
				"   Severity: %s\n\n",
			i+1,
			result.Description,
			result.Pattern,
			result.FirstMatch,
			result.Severity,
		))
	}

	warning.WriteString(
		"ENFORCEMENT ARCHITECTURE:\n\n" +
		"Text instructions are probabilistic (LLM may ignore).\n" +
		"Real enforcement requires three components:\n" +
		"1. Declarative Rule (routing-schema.json)\n" +
		"2. Programmatic Check (validate-routing.sh or hook)\n" +
		"3. Reference Documentation (CLAUDE.md points to enforcement)\n\n" +
		"See: ~/.claude/rules/LLM-guidelines.md § Enforcement Architecture\n\n" +
		"REQUIRED ACTION:\n" +
		"Implement hook enforcement FIRST, then update CLAUDE.md to REFERENCE it.\n" +
		"Do NOT add enforcement language without the hook.\n",
	)

	return warning.String()
}
```

**Tests**: `pkg/enforcement/pattern_detector_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
)

func TestPatternDetector_MustNot(t *testing.T) {
	pd := NewPatternDetector()
	content := "You MUST NOT use this feature without approval."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect MUST NOT pattern")
	}

	if results[0].Severity != "critical" {
		t.Error("MUST NOT should be critical severity")
	}
}

func TestPatternDetector_Blocked(t *testing.T) {
	pd := NewPatternDetector()
	content := "This is BLOCKED by the system."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect BLOCKED pattern")
	}
}

func TestPatternDetector_Never(t *testing.T) {
	pd := NewPatternDetector()
	content := "NEVER use this without consulting the docs."

	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect NEVER pattern")
	}
}

func TestPatternDetector_NoTheater(t *testing.T) {
	pd := NewPatternDetector()
	content := `# Guidelines

Follow these conventions when coding:
- Use descriptive names
- Write tests
- Document decisions
`

	results := pd.Detect(content)

	if len(results) > 0 {
		t.Errorf("Should not detect patterns in normal content, got: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not flag normal content as theater")
	}
}

func TestPatternDetector_MultipleMatches(t *testing.T) {
	pd := NewPatternDetector()
	content := `MUST NOT do X.
MUST NOT do Y.
MUST NOT do Z.
`

	results := pd.Detect(content)

	// Should detect the pattern once but with match count 3
	found := false
	for _, result := range results {
		if strings.Contains(result.Pattern, "MUST") {
			if result.MatchCount != 3 {
				t.Errorf("Expected 3 matches, got: %d", result.MatchCount)
			}
			found = true
		}
	}

	if !found {
		t.Fatal("Should find MUST pattern")
	}
}

func TestGenerateWarning(t *testing.T) {
	results := []DetectionResult{
		{
			Pattern:     "MUST NOT",
			Description: "Test pattern",
			Severity:    "critical",
			MatchCount:  1,
			FirstMatch:  "MUST NOT",
		},
	}

	warning := GenerateWarning(results, "CLAUDE.md")

	if !strings.Contains(warning, "DOCUMENTATION THEATER") {
		t.Error("Warning should mention theater")
	}

	if !strings.Contains(warning, "Enforcement Architecture") {
		t.Error("Warning should reference enforcement architecture")
	}

	if !strings.Contains(warning, "LLM-guidelines.md") {
		t.Error("Warning should reference guidelines")
	}
}
```

**Acceptance Criteria**:
- [ ] `NewPatternDetector()` creates detector with known patterns
- [ ] `Detect()` finds all enforcement anti-patterns
- [ ] Marks MUST NOT, BLOCKED, NEVER as critical
- [ ] `HasDocumentationTheater()` returns true for critical patterns
- [ ] `GenerateWarning()` explains pattern, severity, and fix
- [ ] References enforcement architecture docs
- [ ] Tests verify pattern detection and warning generation
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Pattern detection is core to preventing documentation theater anti-pattern.

---

### GOgent-082: Integration Tests for doc-theater

**Time**: 1.5 hours
**Dependencies**: GOgent-081

**Task**:
End-to-end tests for documentation theater detection.

**File**: `pkg/enforcement/doc_theater_integration_test.go`

```go
package enforcement

import (
	"strings"
	"testing"
	"time"
)

func TestDocTheaterWorkflow_TheaterDetected(t *testing.T) {
	// Parse event
	eventJSON := `{
		"type": "pre-tool-use",
		"hook_event_name": "PreToolUse",
		"tool_name": "Edit",
		"file_path": "/home/user/.claude/CLAUDE.md"
	}`

	event := &PreToolUseEvent{
		ToolName:  "Edit",
		FilePath:  "/home/user/.claude/CLAUDE.md",
	}

	// Event validation
	if !event.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md file")
	}

	if !event.IsWriteOperation() {
		t.Fatal("Should detect Edit operation")
	}

	// Content with theater patterns
	content := `## Gate 6: Task Invocation

You MUST NOT invoke Task(opus) directly.
This is BLOCKED by the system.
Never use direct opus calls.
`

	// Detect patterns
	pd := NewPatternDetector()
	results := pd.Detect(content)

	if len(results) == 0 {
		t.Fatal("Should detect theater patterns")
	}

	if !pd.HasDocumentationTheater(content) {
		t.Error("Should identify as documentation theater")
	}

	// Generate warning
	warning := GenerateWarning(results, "CLAUDE.md")
	if !strings.Contains(warning, "DOCUMENTATION THEATER") {
		t.Error("Warning should indicate theater detected")
	}
}

func TestDocTheaterWorkflow_LegitimateContent(t *testing.T) {
	event := &PreToolUseEvent{
		ToolName: "Edit",
		FilePath: "/home/user/.claude/CLAUDE.md",
	}

	if !event.IsClaudeMDFile() {
		t.Fatal("Should detect CLAUDE.md")
	}

	// Legitimate content (no theater)
	content := `## Enforcement Architecture

Enforcement requires three components:
1. Declarative Rule (routing-schema.json)
2. Programmatic Check (validate-routing.sh hook)
3. Reference Documentation (CLAUDE.md points to enforcement)

See LLM-guidelines.md § Enforcement Architecture.
`

	pd := NewPatternDetector()
	results := pd.Detect(content)

	if len(results) > 0 {
		t.Errorf("Should not flag legitimate content, found: %v", results)
	}

	if pd.HasDocumentationTheater(content) {
		t.Error("Should not identify legitimate content as theater")
	}
}

func TestDocTheaterWorkflow_NonClaude(t *testing.T) {
	// Writing to non-CLAUDE.md file
	event := &PreToolUseEvent{
		ToolName: "Edit",
		FilePath: "/path/to/project.md",
	}

	if event.IsClaudeMDFile() {
		t.Error("Should not match non-CLAUDE.md files")
	}

	// Even with theater patterns, should not trigger
	content := "MUST NOT use this BLOCKED NEVER"

	pd := NewPatternDetector()
	results := pd.Detect(content)

	// Pattern detector would find them, but hook should skip
	// since not CLAUDE.md file
	if len(results) > 0 && event.IsClaudeMDFile() {
		t.Error("Hook should skip non-CLAUDE.md files")
	}
}
```

**Acceptance Criteria**:
- [ ] Event parsing for CLAUDE.md detection works
- [ ] Theater patterns detected correctly
- [ ] Legitimate content not flagged
- [ ] Non-CLAUDE.md files skipped
- [ ] Warning generated with remediation
- [ ] JSON response valid
- [ ] `go test ./pkg/enforcement` passes

**Why This Matters**: Integration tests ensure doc-theater hook catches real documentation theater.

---

### GOgent-083: Build gogent-doc-theater CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-082

**Task**:
Build CLI binary for detect-documentation-theater hook.

**File**: `cmd/gogent-doc-theater/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yourusername/gogent/pkg/enforcement"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse PreToolUse event
	event, err := enforcement.ParsePreToolUseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only check Write/Edit operations on CLAUDE.md files
	if !event.IsClaudeMDFile() || !event.IsWriteOperation() {
		// Silent pass-through for other operations
		fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`)
		os.Exit(0)
	}

	// If content passed via environment, scan it
	content := os.Getenv("TOOL_INPUT_CONTENT")
	if content == "" {
		// Try to read from stdin (after event)
		data, err := io.ReadAll(os.Stdin)
		if err == nil && len(data) > 0 {
			content = string(data)
		}
	}

	// Detect patterns
	pd := enforcement.NewPatternDetector()
	results := pd.Detect(content)

	// Generate response
	response := generateDocTheaterResponse(event, results)

	// Output response
	fmt.Println(response)
}

func generateDocTheaterResponse(event *enforcement.PreToolUseEvent, results []enforcement.DetectionResult) string {
	if len(results) == 0 {
		// No patterns detected
		return `{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`
	}

	// Patterns detected - inject warning
	pd := enforcement.NewPatternDetector()
	content := ""
	for _, result := range results {
		content += result.Description + "\n"
	}

	warning := enforcement.GenerateWarning(results, event.FilePath)

	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "warn",
    "additionalContext": "%s"
  }
}`, escapeJSON(warning))
}

func escapeJSON(s string) string {
	// Minimal escaping for JSON output
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

Note: Add `import "strings"` to imports.

**Build Script**: `scripts/build-doc-theater.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-doc-theater..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-doc-theater ./cmd/gogent-doc-theater

echo "✓ Built: bin/gogent-doc-theater"
```

**Acceptance Criteria**:
- [ ] CLI reads PreToolUse events
- [ ] Passes through non-CLAUDE.md operations
- [ ] Detects theater patterns in content
- [ ] Generates warning response (not blocking)
- [ ] Outputs valid hook response
- [ ] Build script creates executable

**Why This Matters**: CLI is detect-documentation-theater hook implementation. Prevents anti-pattern before commit.

---

## Cross-File References

- **Depends on**:
  - GOgent-063, 069 (event parsing patterns)
  - GOgent-008a (hook response format)
  - Routing schema
- **Used by**:
  - Week 11 (final testing and deployment)
- **Standards**: [00-overview.md](00-overview.md) - error format, XDG paths

---

## Quick Reference

**Orchestrator-Guard Functions**:
- `enforcement.ParseOrchestratorStopEvent()` - Parse SubagentStop
- `enforcement.NewTranscriptAnalyzer()` - Analyze for background tasks
- `enforcement.GenerateGuardResponse()` - Allow/block decision
- `gogent-orchestrator-guard` CLI

**Doc-Theater Functions**:
- `enforcement.ParsePreToolUseEvent()` - Parse PreToolUse
- `enforcement.NewPatternDetector()` - Create pattern detector
- `enforcement.Detect()` - Find anti-patterns
- `enforcement.GenerateWarning()` - Create warning
- `gogent-doc-theater` CLI

**Files Created**:
- `pkg/enforcement/orchestrator_events.go`, tests
- `pkg/enforcement/transcript_analyzer.go`, tests
- `pkg/enforcement/blocking_response.go`, tests
- `pkg/enforcement/orchestrator_integration_test.go`
- `cmd/gogent-orchestrator-guard/main.go`
- `pkg/enforcement/doc_events.go`, tests
- `pkg/enforcement/pattern_detector.go`, tests
- `pkg/enforcement/doc_theater_integration_test.go`
- `cmd/gogent-doc-theater/main.go`
- Build scripts

**Total Lines**: ~1000 implementation + ~800 tests = ~1800 lines

---

## Completion Checklist

- [ ] All 12 tickets (GOgent-075 to 086) complete
- [ ] Orchestrator-guard: event → transcript analysis → block/allow
- [ ] Doc-theater: event → pattern detection → warning
- [ ] All functions have complete imports
- [ ] Error messages use `[component] What. Why. How.` format
- [ ] STDIN timeout implemented (5s)
- [ ] Transcript pattern matching works correctly
- [ ] Documentation theater patterns detected
- [ ] Tests cover all code paths
- [ ] Test coverage ≥80%
- [ ] Both CLI binaries buildable
- [ ] No placeholders or TODOs

---

**Next**: [11-week5-observability-remaining.md](11-week5-observability-remaining.md) - GOgent-087 to 093 (benchmark + stop-gate investigation)
