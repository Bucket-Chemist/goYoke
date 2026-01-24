---
id: GOgent-076
title: Transcript Analysis & Task Tracking
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-075"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-076: Transcript Analysis & Task Tracking

**Time**: 2 hours
**Dependencies**: GOgent-075

**Task**:
Extend `pkg/routing/transcript.go` to track background task spawns and collections using ID-based tracking (not count-based). Use JSON parsing for structured tool calls, with regex fallback for prose patterns.

**File**: `pkg/routing/transcript.go`

**Imports**:
```go
package routing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)
```

**Implementation**:

```go
// TaskTracker tracks spawned and collected background tasks by ID
type TaskTracker struct {
	SpawnedIDs   map[string]bool // task_id -> true when spawned
	CollectedIDs map[string]bool // task_id -> true when collected
}

// NewTaskTracker creates a new TaskTracker instance
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		SpawnedIDs:   make(map[string]bool),
		CollectedIDs: make(map[string]bool),
	}
}

// GetUncollected returns slice of task IDs that were spawned but not collected
func (t *TaskTracker) GetUncollected() []string {
	var uncollected []string
	for id := range t.SpawnedIDs {
		if !t.CollectedIDs[id] {
			uncollected = append(uncollected, id)
		}
	}
	return uncollected
}

// HasUncollected returns true if any spawned tasks have not been collected
func (t *TaskTracker) HasUncollected() bool {
	return len(t.GetUncollected()) > 0
}

// GetStats returns spawn/collect counts and uncollected list
func (t *TaskTracker) GetStats() (spawned, collected int, uncollected []string) {
	spawned = len(t.SpawnedIDs)
	collected = len(t.CollectedIDs)
	uncollected = t.GetUncollected()
	return
}

// TranscriptAnalyzer scans transcript for background task patterns
type TranscriptAnalyzer struct {
	filePath string
	tracker  *TaskTracker
}

// NewTranscriptAnalyzer creates analyzer instance
func NewTranscriptAnalyzer(transcriptPath string) *TranscriptAnalyzer {
	return &TranscriptAnalyzer{
		filePath: transcriptPath,
		tracker:  NewTaskTracker(),
	}
}

// BashToolCall represents a parsed Bash tool invocation
type BashToolCall struct {
	RunInBackground bool   `json:"run_in_background"`
	Description     string `json:"description"`
}

// TaskOutputCall represents a parsed TaskOutput invocation
type TaskOutputCall struct {
	TaskID string `json:"task_id"`
	Block  bool   `json:"block"`
}

// Analyze scans transcript for background task spawns and collections
func (ta *TranscriptAnalyzer) Analyze() error {
	file, err := os.Open(ta.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No transcript - assume no background tasks
			return nil
		}
		return fmt.Errorf("[transcript-analyzer] Failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNum int

	// Regex patterns for fallback detection in prose
	spawnPattern := regexp.MustCompile(`(?i)run_in_background[:\s=]+true`)
	taskIDPattern := regexp.MustCompile(`task_id[:\s=]+"([^"]+)"`)
	collectPattern := regexp.MustCompile(`(?i)TaskOutput.*task_id[:\s=]+"([^"]+)"`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Attempt JSON parsing for structured tool calls
		if strings.Contains(line, `"run_in_background"`) {
			var bashCall BashToolCall
			if err := json.Unmarshal([]byte(line), &bashCall); err == nil {
				if bashCall.RunInBackground {
					// Extract task ID from same line if present
					matches := taskIDPattern.FindStringSubmatch(line)
					if len(matches) > 1 {
						taskID := matches[1]
						ta.tracker.SpawnedIDs[taskID] = true
					}
				}
				continue // Successfully parsed as JSON
			}
		}

		if strings.Contains(line, `"task_id"`) && strings.Contains(line, "TaskOutput") {
			var taskOutput TaskOutputCall
			if err := json.Unmarshal([]byte(line), &taskOutput); err == nil {
				if taskOutput.TaskID != "" {
					ta.tracker.CollectedIDs[taskOutput.TaskID] = true
				}
				continue // Successfully parsed as JSON
			}
		}

		// Fallback: Regex for prose patterns
		if spawnPattern.MatchString(line) {
			matches := taskIDPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				taskID := matches[1]
				ta.tracker.SpawnedIDs[taskID] = true
			}
		}

		if collectPattern.MatchString(line) {
			matches := collectPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				taskID := matches[1]
				ta.tracker.CollectedIDs[taskID] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("[transcript-analyzer] Scanner error: %w", err)
	}

	return nil
}

// HasUncollectedTasks checks if there are uncollected background tasks
func (ta *TranscriptAnalyzer) HasUncollectedTasks() bool {
	return ta.tracker.HasUncollected()
}

// GetSummary returns analysis summary
func (ta *TranscriptAnalyzer) GetSummary() string {
	spawned, collected, uncollected := ta.tracker.GetStats()

	if spawned == 0 {
		return "No background tasks detected"
	}

	return fmt.Sprintf(
		"Background tasks: %d spawned, %d collected, %d uncollected",
		spawned,
		collected,
		len(uncollected),
	)
}

// GetUncollectedList returns formatted list of uncollected task IDs
func (ta *TranscriptAnalyzer) GetUncollectedList() string {
	uncollected := ta.tracker.GetUncollected()
	if len(uncollected) == 0 {
		return ""
	}

	var list strings.Builder
	list.WriteString("Uncollected tasks:\n")
	for i, id := range uncollected {
		list.WriteString(fmt.Sprintf("  %d. %s\n", i+1, id))
	}

	return list.String()
}
```

**Tests**: `pkg/routing/transcript_test.go` (add to existing file)

```go
func TestTaskTracker_IDBasedTracking(t *testing.T) {
	tracker := NewTaskTracker()

	// Spawn tasks
	tracker.SpawnedIDs["task-1"] = true
	tracker.SpawnedIDs["task-2"] = true
	tracker.SpawnedIDs["task-3"] = true

	// Collect only task-1
	tracker.CollectedIDs["task-1"] = true

	uncollected := tracker.GetUncollected()
	if len(uncollected) != 2 {
		t.Errorf("Expected 2 uncollected, got %d", len(uncollected))
	}

	// Verify specific IDs are uncollected
	uncollectedMap := make(map[string]bool)
	for _, id := range uncollected {
		uncollectedMap[id] = true
	}

	if !uncollectedMap["task-2"] || !uncollectedMap["task-3"] {
		t.Error("Expected task-2 and task-3 to be uncollected")
	}

	if uncollectedMap["task-1"] {
		t.Error("task-1 should not be uncollected")
	}
}

func TestTaskTracker_DuplicateIDs(t *testing.T) {
	tracker := NewTaskTracker()

	// Spawn same ID twice (should be idempotent)
	tracker.SpawnedIDs["task-1"] = true
	tracker.SpawnedIDs["task-1"] = true

	spawned, _, _ := tracker.GetStats()
	if spawned != 1 {
		t.Errorf("Expected 1 unique spawned task, got %d", spawned)
	}

	// Collect same ID twice (should be idempotent)
	tracker.CollectedIDs["task-1"] = true
	tracker.CollectedIDs["task-1"] = true

	if tracker.HasUncollected() {
		t.Error("Should not have uncollected tasks after collecting all")
	}
}

func TestTranscriptAnalyzer_NoBackgroundTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

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

	summary := analyzer.GetSummary()
	if summary != "No background tasks detected" {
		t.Errorf("Unexpected summary: %s", summary)
	}
}

func TestTranscriptAnalyzer_JSONParsing(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Structured JSON tool calls
	content := `# Transcript

{"tool": "Bash", "run_in_background": true, "task_id": "bg-1"}
{"tool": "Bash", "run_in_background": true, "task_id": "bg-2"}

... other work ...

{"tool": "TaskOutput", "task_id": "bg-1", "block": true}
{"tool": "TaskOutput", "task_id": "bg-2", "block": true}
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	if analyzer.HasUncollectedTasks() {
		t.Error("Should not have uncollected tasks when all collected")
	}

	spawned, collected, _ := analyzer.tracker.GetStats()
	if spawned != 2 {
		t.Errorf("Expected 2 spawned, got %d", spawned)
	}
	if collected != 2 {
		t.Errorf("Expected 2 collected, got %d", collected)
	}
}

func TestTranscriptAnalyzer_RegexFallback(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Prose patterns (not valid JSON)
	content := `# Transcript

Spawning background task with run_in_background: true, task_id: "prose-1"
Spawning background task with run_in_background: true, task_id: "prose-2"

Later...

TaskOutput with task_id: "prose-1"
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	if !analyzer.HasUncollectedTasks() {
		t.Error("Should detect uncollected task")
	}

	uncollected := analyzer.tracker.GetUncollected()
	if len(uncollected) != 1 {
		t.Errorf("Expected 1 uncollected, got %d", len(uncollected))
	}

	if uncollected[0] != "prose-2" {
		t.Errorf("Expected prose-2 uncollected, got %s", uncollected[0])
	}
}

func TestTranscriptAnalyzer_UncollectedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	content := `# Transcript

{"tool": "Bash", "run_in_background": true, "task_id": "bg-1"}
{"tool": "Bash", "run_in_background": true, "task_id": "bg-2"}
{"tool": "Bash", "run_in_background": true, "task_id": "bg-3"}

{"tool": "TaskOutput", "task_id": "bg-1", "block": true}
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	analyzer.Analyze()

	if !analyzer.HasUncollectedTasks() {
		t.Error("Should detect uncollected tasks")
	}

	list := analyzer.GetUncollectedList()
	if !strings.Contains(list, "bg-2") || !strings.Contains(list, "bg-3") {
		t.Error("List should contain bg-2 and bg-3")
	}

	if strings.Contains(list, "bg-1") {
		t.Error("List should not contain bg-1 (was collected)")
	}
}

func TestTranscriptAnalyzer_MissingFile(t *testing.T) {
	analyzer := NewTranscriptAnalyzer("/nonexistent/path.md")
	err := analyzer.Analyze()

	// Should NOT error on missing file (no transcript = no tasks)
	if err != nil {
		t.Errorf("Should not error on missing file, got: %v", err)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Missing file should imply no tasks")
	}
}

func TestTranscriptAnalyzer_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.md")

	// Mix of valid JSON, malformed JSON, and prose
	content := `# Transcript

{"tool": "Bash", "run_in_background": true, "task_id": "valid-1"}
{malformed json here with task_id: "malformed-1"}
Prose line with run_in_background: true, task_id: "prose-1"

{"tool": "TaskOutput", "task_id": "valid-1", "block": true}
TaskOutput with task_id: "prose-1"
`
	os.WriteFile(transcriptPath, []byte(content), 0644)

	analyzer := NewTranscriptAnalyzer(transcriptPath)
	err := analyzer.Analyze()

	// Should not error on malformed JSON - just fall back to regex
	if err != nil {
		t.Fatalf("Should handle malformed JSON gracefully: %v", err)
	}

	// Should detect both valid-1 (JSON) and prose-1 (regex fallback)
	spawned, collected, uncollected := analyzer.tracker.GetStats()
	if spawned != 2 {
		t.Errorf("Expected 2 spawned (valid-1 + prose-1), got %d", spawned)
	}
	if collected != 2 {
		t.Errorf("Expected 2 collected, got %d", collected)
	}
	if len(uncollected) != 0 {
		t.Errorf("Expected 0 uncollected, got %d: %v", len(uncollected), uncollected)
	}
}
```

**Acceptance Criteria**:
- [ ] `TaskTracker` uses ID-based tracking (not count-based)
- [ ] `GetUncollected()` returns correct task IDs
- [ ] `Analyze()` parses JSON tool calls for structured data
- [ ] Regex fallback handles prose patterns
- [ ] Duplicate task IDs handled idempotently
- [ ] Malformed JSON falls back to regex without error
- [ ] `GetSummary()` returns accurate spawn/collect/uncollected counts
- [ ] `GetUncollectedList()` lists only uncollected task IDs
- [ ] `go test ./pkg/routing` passes with new tests

**Why This Matters**: ID-based tracking prevents false positives/negatives that occur with count-based tracking. JSON parsing provides robustness for structured tool calls, while regex fallback handles prose descriptions.

**Key Fixes from Previous Version**:
1. Replaced count-based tracking with ID-based maps
2. Added JSON parsing for structured tool invocations
3. Extended existing `pkg/routing/transcript.go` (not new `pkg/enforcement/` file)
4. Added tests for duplicate IDs, malformed JSON, and idempotent operations

---
