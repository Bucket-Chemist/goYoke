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
acceptance_criteria_count: 8
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
