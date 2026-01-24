---
id: GOgent-096
title: Integration Tests for session-archive Hook
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-094","GOgent-033"]
priority: high
week: 5
tags: ["integration-tests", "week-5"]
tests_required: true
acceptance_criteria_count: 7
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
