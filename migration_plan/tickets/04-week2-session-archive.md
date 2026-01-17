# Week 2 Part 1: Session Archive Translation

**File**: `04-week2-session-archive.md`
**Tickets**: GOgent-026 to 033 (8 tickets)
**Total Time**: ~13 hours
**Phase**: Week 2 Part 1

---

## Navigation

- **Previous**: [03-week1-validation-cli.md](03-week1-validation-cli.md) - GOgent-020 to 025
- **Next**: [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md) - GOgent-034 to 040
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure

---

## Summary

This file translates the session-archive.sh hook from Bash to Go:

1. **Session Event Handling**: Parse SessionEnd events
2. **Metrics Collection**: Count tools, errors, violations from session
3. **Handoff Generation**: Create markdown summary for next session
4. **Pending Learnings**: Format sharp edges captured during session
5. **Violations Summary**: Summarize routing violations
6. **File Archival**: Copy transcripts, learnings, violations to archive
7. **Integration**: Wire into gogent-archive CLI binary

**Critical Context**:
- SessionEnd event includes transcript_path to full session JSONL
- Handoff file (.claude/memory/last-handoff.md) loaded by next session
- Pending learnings come from sharp-edge-detector hook
- Archive preserves full session history for analysis

---

## GOgent-026: Define Session Event Structs

**Time**: 1.5 hours
**Dependencies**: GOgent-007 (ToolEvent base structs)

**Task**:
Define SessionEvent struct for SessionEnd hook trigger. Parse session metadata including transcript path and session ID.

**File**: `pkg/session/events.go`

**Imports**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// SessionEvent represents SessionEnd hook event
type SessionEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      int64  `json:"timestamp,omitempty"`
}

// SessionMetrics represents collected session statistics
type SessionMetrics struct {
	ToolCalls          int    `json:"tool_calls"`
	ErrorsLogged       int    `json:"errors_logged"`
	RoutingViolations  int    `json:"routing_violations"`
	SessionID          string `json:"session_id"`
	Duration           int64  `json:"duration_seconds,omitempty"`
}

// ParseSessionEvent reads SessionEnd event from STDIN
func ParseSessionEvent(r io.Reader, timeout time.Duration) (*SessionEvent, error) {
	type result struct {
		event *SessionEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to read STDIN: %w", err)}
			return
		}

		var event SessionEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		// Validate required fields
		if event.SessionID == "" {
			event.SessionID = "unknown"
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[session-parser] STDIN read timeout after %v", timeout)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Tests**: `pkg/session/events_test.go`

```go
package session

import (
	"strings"
	"testing"
	"time"
)

func TestParseSessionEvent_Valid(t *testing.T) {
	jsonInput := `{
		"session_id": "abc-123",
		"transcript_path": "/tmp/session-abc-123.jsonl",
		"hook_event_name": "SessionEnd"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "abc-123" {
		t.Errorf("Expected session_id abc-123, got: %s", event.SessionID)
	}

	if event.TranscriptPath != "/tmp/session-abc-123.jsonl" {
		t.Errorf("Expected transcript path, got: %s", event.TranscriptPath)
	}
}

func TestParseSessionEvent_MissingSessionID(t *testing.T) {
	jsonInput := `{
		"transcript_path": "/tmp/session.jsonl",
		"hook_event_name": "SessionEnd"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "unknown" {
		t.Errorf("Expected default 'unknown', got: %s", event.SessionID)
	}
}

func TestParseSessionEvent_InvalidJSON(t *testing.T) {
	jsonInput := `{invalid json}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSessionEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseSessionEvent_Timeout(t *testing.T) {
	reader := &slowReader{delay: 10 * time.Second}
	_, err := ParseSessionEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout in error, got: %v", err)
	}
}

// slowReader for timeout testing
type slowReader struct {
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(r.delay)
	return 0, io.EOF
}
```

**Acceptance Criteria**:
- [ ] `ParseSessionEvent()` reads SessionEnd event from STDIN
- [ ] Validates and defaults session_id to "unknown" if missing
- [ ] Handles transcript_path field correctly
- [ ] Implements 5s timeout on STDIN read
- [ ] Tests cover valid, missing fields, invalid JSON, timeout
- [ ] `go test ./pkg/session` passes

**Why This Matters**: SessionEvent is the trigger for archival. Must parse reliably to avoid losing session data.

---

## GOgent-027: Implement Session Metrics Collection

**Time**: 2 hours
**Dependencies**: GOgent-026

**Task**:
Count session metrics from log files: tool calls, errors logged, routing violations.

**File**: `pkg/session/metrics.go`

**Imports**:
```go
package session

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/gogent-fortress/pkg/config"
)
```

**Implementation**:
```go
// CollectSessionMetrics gathers statistics from session
func CollectSessionMetrics(sessionID string) (*SessionMetrics, error) {
	metrics := &SessionMetrics{
		SessionID: sessionID,
	}

	// Count tool calls from temp counters
	toolCount, err := countToolCalls()
	if err == nil {
		metrics.ToolCalls = toolCount
	}

	// Count errors from error log
	errorCount, err := countLogLines(getErrorLogPath())
	if err == nil {
		metrics.ErrorsLogged = errorCount
	}

	// Count routing violations
	violationCount, err := countLogLines(config.GetViolationsLogPath())
	if err == nil {
		metrics.RoutingViolations = violationCount
	}

	return metrics, nil
}

// countToolCalls sums tool counter temp files
func countToolCalls() (int, error) {
	pattern := "/tmp/claude-tool-counter-*"
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return 0, nil // No counters is OK
	}

	total := 0
	for _, counterFile := range matches {
		count, err := countLogLines(counterFile)
		if err == nil {
			total += count
		}
	}

	return total, nil
}

// countLogLines counts non-empty lines in file
func countLogLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // File not existing = 0 lines
		}
		return 0, fmt.Errorf("[metrics] Failed to open %s: %w", path, err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("[metrics] Failed to scan %s: %w", path, err)
	}

	return count, nil
}

// getErrorLogPath returns path to error patterns log
func getErrorLogPath() string {
	// TODO: Make XDG-compliant in future iteration
	return "/tmp/claude-error-patterns.jsonl"
}
```

**Tests**: `pkg/session/metrics_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectSessionMetrics(t *testing.T) {
	// Create temp error log
	tmpErrorLog := "/tmp/test-error-patterns.jsonl"
	os.WriteFile(tmpErrorLog, []byte(`{"error": "test1"}
{"error": "test2"}
{"error": "test3"}
`), 0644)
	defer os.Remove(tmpErrorLog)

	// Create temp violation log
	tmpDir := t.TempDir()
	config.SetGOgentDirForTest(tmpDir)
	defer config.ResetGOgentDir()

	violationLog := config.GetViolationsLogPath()
	os.MkdirAll(filepath.Dir(violationLog), 0755)
	os.WriteFile(violationLog, []byte(`{"violation": "v1"}
{"violation": "v2"}
`), 0644)

	// Collect metrics
	metrics, err := CollectSessionMetrics("test-session")
	if err != nil {
		t.Fatalf("CollectSessionMetrics failed: %v", err)
	}

	if metrics.SessionID != "test-session" {
		t.Errorf("Expected session_id test-session, got: %s", metrics.SessionID)
	}

	// Note: Tool count will be 0 because we can't easily create temp files with glob pattern
	// Error count should be 3
	if metrics.ErrorsLogged != 3 {
		t.Errorf("Expected 3 errors, got: %d", metrics.ErrorsLogged)
	}

	// Violation count should be 2
	if metrics.RoutingViolations != 2 {
		t.Errorf("Expected 2 violations, got: %d", metrics.RoutingViolations)
	}
}

func TestCountLogLines_FileExists(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	os.WriteFile(tmpFile, []byte("line1\nline2\n\nline3\n"), 0644)

	count, err := countLogLines(tmpFile)
	if err != nil {
		t.Fatalf("countLogLines failed: %v", err)
	}

	// Should count 3 non-empty lines
	if count != 3 {
		t.Errorf("Expected 3 lines, got: %d", count)
	}
}

func TestCountLogLines_FileNotExists(t *testing.T) {
	count, err := countLogLines("/nonexistent/file.log")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 for missing file, got: %d", count)
	}
}

func TestCountLogLines_EmptyFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "empty.log")
	os.WriteFile(tmpFile, []byte(""), 0644)

	count, err := countLogLines(tmpFile)
	if err != nil {
		t.Fatalf("countLogLines failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 for empty file, got: %d", count)
	}
}
```

**Acceptance Criteria**:
- [ ] `CollectSessionMetrics()` counts tool calls from temp files
- [ ] Counts errors from error log
- [ ] Counts routing violations from violations log
- [ ] Returns 0 when log files don't exist (not an error)
- [ ] `countLogLines()` ignores empty lines
- [ ] Tests verify counts for various scenarios
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Metrics provide visibility into session activity. Used in handoff document to give next session context.

---

## GOgent-028: Implement Handoff Document Generation

**Time**: 2 hours
**Dependencies**: GOgent-027

**Task**:
Generate markdown handoff document with session metrics, pending learnings, violations summary.

**File**: `pkg/session/handoff.go`

**Imports**:
```go
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```

**Implementation**:
```go
// HandoffConfig contains paths for handoff generation
type HandoffConfig struct {
	ProjectDir      string
	HandoffPath     string
	PendingPath     string
	ViolationsPath  string
}

// DefaultHandoffConfig returns standard paths
func DefaultHandoffConfig(projectDir string) *HandoffConfig {
	memoryDir := filepath.Join(projectDir, ".claude", "memory")

	return &HandoffConfig{
		ProjectDir:     projectDir,
		HandoffPath:    filepath.Join(memoryDir, "last-handoff.md"),
		PendingPath:    filepath.Join(memoryDir, "pending-learnings.jsonl"),
		ViolationsPath: filepath.Join(memoryDir, "routing-violations.jsonl"),
	}
}

// GenerateHandoff creates handoff markdown document
func GenerateHandoff(config *HandoffConfig, metrics *SessionMetrics) error {
	timestamp := time.Now().Format("20060102-150405")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(config.HandoffPath), 0755)

	// Open handoff file for writing
	f, err := os.Create(config.HandoffPath)
	if err != nil {
		return fmt.Errorf("[handoff] Failed to create handoff file at %s: %w", config.HandoffPath, err)
	}
	defer f.Close()

	// Write header and metrics
	fmt.Fprintf(f, "# Session Handoff - %s\n\n", timestamp)
	fmt.Fprintf(f, "## Session Metrics\n")
	fmt.Fprintf(f, "- **Tool Calls**: ~%d\n", metrics.ToolCalls)
	fmt.Fprintf(f, "- **Errors Logged**: %d\n", metrics.ErrorsLogged)
	fmt.Fprintf(f, "- **Routing Violations**: %d\n", metrics.RoutingViolations)
	fmt.Fprintf(f, "- **Session ID**: %s\n\n", metrics.SessionID)

	// Add pending learnings section
	if err := writePendingLearnings(f, config.PendingPath); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to write pending learnings: %v\n", err)
	}

	// Add routing violations section
	if err := writeViolationsSummary(f, config.ViolationsPath); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to write violations summary: %v\n", err)
	}

	// Write footer with action items
	fmt.Fprintf(f, "\n## Context for Next Session\n")
	fmt.Fprintf(f, "1. Review pending learnings before continuing work\n")
	fmt.Fprintf(f, "2. Check if any sharp edges should be added to agent sharp-edges.yaml\n")
	fmt.Fprintf(f, "3. Validate routing decisions match patterns from last session\n\n")

	fmt.Fprintf(f, "## Immediate Actions\n")
	fmt.Fprintf(f, "- [ ] Process pending sharp edges\n")
	fmt.Fprintf(f, "- [ ] Review routing violations for pattern issues\n")
	fmt.Fprintf(f, "- [ ] Update TODO if tasks remain incomplete\n\n")

	fmt.Fprintf(f, "---\n")
	fmt.Fprintf(f, "*Generated automatically by session-archive hook*\n")

	return nil
}

// writePendingLearnings adds pending learnings section (stub for now)
func writePendingLearnings(f *os.File, pendingPath string) error {
	fmt.Fprintf(f, "## Pending Learnings\n")

	// Check if pending learnings exist
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		fmt.Fprintf(f, "No new sharp edges captured this session.\n\n")
		return nil
	}

	// Read and format pending learnings (GOgent-029 implements full formatting)
	fmt.Fprintf(f, "The following sharp edges were captured and need review:\n\n")
	fmt.Fprintf(f, "*See %s for details*\n\n", pendingPath)

	return nil
}

// writeViolationsSummary adds routing violations section (stub for now)
func writeViolationsSummary(f *os.File, violationsPath string) error {
	// Check if violations exist
	if _, err := os.Stat(violationsPath); os.IsNotExist(err) {
		return nil // No violations, skip section
	}

	fmt.Fprintf(f, "## Routing Violations\n")
	fmt.Fprintf(f, "The following routing violations were detected:\n\n")
	fmt.Fprintf(f, "*See %s for details*\n\n", violationsPath)

	return nil
}
```

**Tests**: `pkg/session/handoff_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateHandoff(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HandoffConfig{
		ProjectDir:     tmpDir,
		HandoffPath:    filepath.Join(tmpDir, "last-handoff.md"),
		PendingPath:    filepath.Join(tmpDir, "pending-learnings.jsonl"),
		ViolationsPath: filepath.Join(tmpDir, "routing-violations.jsonl"),
	}

	metrics := &SessionMetrics{
		ToolCalls:         42,
		ErrorsLogged:      3,
		RoutingViolations: 1,
		SessionID:         "test-123",
	}

	err := GenerateHandoff(config, metrics)
	if err != nil {
		t.Fatalf("GenerateHandoff failed: %v", err)
	}

	// Read generated handoff
	content, err := os.ReadFile(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to read handoff: %v", err)
	}

	contentStr := string(content)

	// Verify sections exist
	checks := []string{
		"Session Handoff",
		"Session Metrics",
		"Tool Calls**: ~42",
		"Errors Logged**: 3",
		"Routing Violations**: 1",
		"Session ID**: test-123",
		"Pending Learnings",
		"Context for Next Session",
		"Immediate Actions",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Handoff missing expected content: %s", check)
		}
	}
}

func TestGenerateHandoff_WithPendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HandoffConfig{
		ProjectDir:     tmpDir,
		HandoffPath:    filepath.Join(tmpDir, "last-handoff.md"),
		PendingPath:    filepath.Join(tmpDir, "pending-learnings.jsonl"),
		ViolationsPath: filepath.Join(tmpDir, "routing-violations.jsonl"),
	}

	// Create pending learnings file
	os.WriteFile(config.PendingPath, []byte(`{"file": "test.go", "error": "test error"}`), 0644)

	metrics := &SessionMetrics{
		SessionID: "test-456",
	}

	err := GenerateHandoff(config, metrics)
	if err != nil {
		t.Fatalf("GenerateHandoff failed: %v", err)
	}

	content, err := os.ReadFile(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to read handoff: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "sharp edges were captured") {
		t.Error("Expected pending learnings message")
	}
}

func TestDefaultHandoffConfig(t *testing.T) {
	projectDir := "/test/project"
	config := DefaultHandoffConfig(projectDir)

	if config.ProjectDir != projectDir {
		t.Errorf("Expected project dir %s, got: %s", projectDir, config.ProjectDir)
	}

	expectedHandoff := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if config.HandoffPath != expectedHandoff {
		t.Errorf("Expected handoff path %s, got: %s", expectedHandoff, config.HandoffPath)
	}

	expectedPending := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if config.PendingPath != expectedPending {
		t.Errorf("Expected pending path %s, got: %s", expectedPending, config.PendingPath)
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateHandoff()` creates markdown file at handoff path
- [ ] Includes session metrics (tool calls, errors, violations)
- [ ] Includes pending learnings section
- [ ] Includes routing violations section
- [ ] Includes action items checklist
- [ ] Tests verify all sections present
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Handoff document bridges sessions. Next session loads this for context continuity.

---

## GOgent-029: Format Pending Learnings

**Time**: 1.5 hours
**Dependencies**: GOgent-028

**Task**:
Parse pending-learnings.jsonl and format sharp edges into markdown bullets.

**File**: `pkg/session/learnings.go`

**Imports**:
```go
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)
```

**Implementation**:
```go
// SharpEdge represents a captured sharp edge from detector
type SharpEdge struct {
	File               string `json:"file"`
	ErrorType          string `json:"error_type"`
	ConsecutiveFailures int   `json:"consecutive_failures"`
	LastError          string `json:"last_error,omitempty"`
	Timestamp          int64  `json:"timestamp"`
}

// FormatPendingLearnings reads JSONL and formats as markdown
func FormatPendingLearnings(pendingPath string) ([]string, error) {
	f, err := os.Open(pendingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No file is OK
		}
		return nil, fmt.Errorf("[learnings] Failed to open %s: %w", pendingPath, err)
	}
	defer f.Close()

	var learnings []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var edge SharpEdge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			// Skip invalid lines
			continue
		}

		// Format as markdown bullet
		formatted := fmt.Sprintf("- **%s**: %s (%d failures)",
			edge.File,
			edge.ErrorType,
			edge.ConsecutiveFailures,
		)

		learnings = append(learnings, formatted)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[learnings] Failed to scan %s: %w", pendingPath, err)
	}

	return learnings, nil
}
```

**Update**: `pkg/session/handoff.go` (replace stub)

```go
// writePendingLearnings adds pending learnings section with full formatting
func writePendingLearnings(f *os.File, pendingPath string) error {
	fmt.Fprintf(f, "## Pending Learnings\n")

	learnings, err := FormatPendingLearnings(pendingPath)
	if err != nil {
		return err
	}

	if len(learnings) == 0 {
		fmt.Fprintf(f, "No new sharp edges captured this session.\n\n")
		return nil
	}

	fmt.Fprintf(f, "The following sharp edges were captured and need review:\n\n")
	for _, learning := range learnings {
		fmt.Fprintf(f, "%s\n", learning)
	}
	fmt.Fprintf(f, "\n")

	return nil
}
```

**Tests**: `pkg/session/learnings_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatPendingLearnings_ValidFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "pending.jsonl")

	content := `{"file": "src/main.go", "error_type": "type_mismatch", "consecutive_failures": 3, "timestamp": 1234567890}
{"file": "pkg/utils.go", "error_type": "nil_pointer", "consecutive_failures": 2, "timestamp": 1234567891}
`
	os.WriteFile(tmpFile, []byte(content), 0644)

	learnings, err := FormatPendingLearnings(tmpFile)
	if err != nil {
		t.Fatalf("FormatPendingLearnings failed: %v", err)
	}

	if len(learnings) != 2 {
		t.Fatalf("Expected 2 learnings, got: %d", len(learnings))
	}

	// Check first learning
	expected := "- **src/main.go**: type_mismatch (3 failures)"
	if learnings[0] != expected {
		t.Errorf("Expected %s, got: %s", expected, learnings[0])
	}

	// Check second learning
	expected = "- **pkg/utils.go**: nil_pointer (2 failures)"
	if learnings[1] != expected {
		t.Errorf("Expected %s, got: %s", expected, learnings[1])
	}
}

func TestFormatPendingLearnings_NoFile(t *testing.T) {
	learnings, err := FormatPendingLearnings("/nonexistent/pending.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if learnings != nil {
		t.Errorf("Expected nil for missing file, got: %v", learnings)
	}
}

func TestFormatPendingLearnings_EmptyFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "empty.jsonl")
	os.WriteFile(tmpFile, []byte(""), 0644)

	learnings, err := FormatPendingLearnings(tmpFile)
	if err != nil {
		t.Fatalf("FormatPendingLearnings failed: %v", err)
	}

	if len(learnings) != 0 {
		t.Errorf("Expected empty slice, got: %d learnings", len(learnings))
	}
}

func TestFormatPendingLearnings_InvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.jsonl")
	os.WriteFile(tmpFile, []byte("{invalid json}\n"), 0644)

	learnings, err := FormatPendingLearnings(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error (skip invalid lines), got: %v", err)
	}

	// Should skip invalid lines
	if len(learnings) != 0 {
		t.Errorf("Expected 0 learnings (invalid skipped), got: %d", len(learnings))
	}
}
```

**Acceptance Criteria**:
- [ ] `FormatPendingLearnings()` reads JSONL file
- [ ] Parses SharpEdge structs from each line
- [ ] Formats as markdown bullets with file, error type, failure count
- [ ] Skips invalid JSON lines gracefully
- [ ] Returns nil for missing file (not an error)
- [ ] Tests verify formatting, empty file, invalid JSON
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Pending learnings are critical for improving system. Formatting must be clear for human review.

---

## GOgent-030: Format Routing Violations Summary

**Time**: 1.5 hours
**Dependencies**: GOgent-029

**Task**:
Parse routing-violations.jsonl and format top 10 violations for handoff summary.

**File**: `pkg/session/violations_summary.go`

**Imports**:
```go
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yourusername/gogent-fortress/pkg/routing"
)
```

**Implementation**:
```go
// FormatViolationsSummary reads violations and formats top 10 for handoff
func FormatViolationsSummary(violationsPath string, maxLines int) ([]string, error) {
	f, err := os.Open(violationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No violations is OK
		}
		return nil, fmt.Errorf("[violations-summary] Failed to open %s: %w", violationsPath, err)
	}
	defer f.Close()

	var violations []string
	scanner := bufio.NewScanner(f)
	count := 0

	for scanner.Scan() && count < maxLines {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var v routing.Violation
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			// Skip invalid lines
			continue
		}

		// Format based on violation type
		formatted := formatViolation(&v)
		if formatted != "" {
			violations = append(violations, formatted)
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[violations-summary] Failed to scan %s: %w", violationsPath, err)
	}

	return violations, nil
}

// formatViolation creates markdown bullet for violation
func formatViolation(v *routing.Violation) string {
	switch v.ViolationType {
	case "tool_permission":
		return fmt.Sprintf("- Tool permission: Tier attempted **%s** (allowed: %s)", v.Tool, v.Allowed)

	case "blocked_task_opus":
		return fmt.Sprintf("- Einstein blocking: Attempted Task(model: opus) with agent **%s**", v.Agent)

	case "blocked_task_einstein":
		return fmt.Sprintf("- Einstein blocking: Attempted einstein agent via Task tool")

	case "delegation_ceiling":
		return fmt.Sprintf("- Delegation ceiling: Agent **%s** requested **%s** model", v.Agent, v.Model)

	case "subagent_type_mismatch":
		return fmt.Sprintf("- Subagent type: Agent **%s** - %s", v.Agent, v.Reason)

	default:
		return fmt.Sprintf("- %s: %s", v.ViolationType, v.Reason)
	}
}
```

**Update**: `pkg/session/handoff.go` (replace stub)

```go
// writeViolationsSummary adds routing violations section with formatted list
func writeViolationsSummary(f *os.File, violationsPath string) error {
	violations, err := FormatViolationsSummary(violationsPath, 10)
	if err != nil {
		return err
	}

	if len(violations) == 0 {
		return nil // No violations, skip section
	}

	fmt.Fprintf(f, "## Routing Violations\n")
	fmt.Fprintf(f, "The following routing violations were detected:\n\n")

	for _, violation := range violations {
		fmt.Fprintf(f, "%s\n", violation)
	}
	fmt.Fprintf(f, "\n")

	return nil
}
```

**Tests**: `pkg/session/violations_summary_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/routing"
)

func TestFormatViolationsSummary(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "violations.jsonl")

	v1 := routing.Violation{
		ViolationType: "tool_permission",
		Tool:          "Write",
		Allowed:       "Read, Glob, Grep",
	}

	v2 := routing.Violation{
		ViolationType: "blocked_task_opus",
		Agent:         "python-pro",
		Model:         "opus",
	}

	v3 := routing.Violation{
		ViolationType: "delegation_ceiling",
		Agent:         "architect",
		Model:         "sonnet",
	}

	// Write violations as JSONL
	f, _ := os.Create(tmpFile)
	json.NewEncoder(f).Encode(v1)
	json.NewEncoder(f).Encode(v2)
	json.NewEncoder(f).Encode(v3)
	f.Close()

	// Format violations
	violations, err := FormatViolationsSummary(tmpFile, 10)
	if err != nil {
		t.Fatalf("FormatViolationsSummary failed: %v", err)
	}

	if len(violations) != 3 {
		t.Fatalf("Expected 3 violations, got: %d", len(violations))
	}

	// Check formatting
	if !strings.Contains(violations[0], "Write") {
		t.Errorf("Expected 'Write' in first violation: %s", violations[0])
	}

	if !strings.Contains(violations[1], "opus") {
		t.Errorf("Expected 'opus' in second violation: %s", violations[1])
	}

	if !strings.Contains(violations[2], "architect") {
		t.Errorf("Expected 'architect' in third violation: %s", violations[2])
	}
}

func TestFormatViolationsSummary_MaxLines(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "many-violations.jsonl")

	f, _ := os.Create(tmpFile)
	for i := 0; i < 20; i++ {
		v := routing.Violation{
			ViolationType: "tool_permission",
			Tool:          fmt.Sprintf("Tool%d", i),
		}
		json.NewEncoder(f).Encode(v)
	}
	f.Close()

	// Request only 5 violations
	violations, err := FormatViolationsSummary(tmpFile, 5)
	if err != nil {
		t.Fatalf("FormatViolationsSummary failed: %v", err)
	}

	if len(violations) != 5 {
		t.Errorf("Expected 5 violations (maxLines), got: %d", len(violations))
	}
}

func TestFormatViolationsSummary_NoFile(t *testing.T) {
	violations, err := FormatViolationsSummary("/nonexistent/violations.jsonl", 10)

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if violations != nil {
		t.Errorf("Expected nil for missing file, got: %v", violations)
	}
}
```

**Acceptance Criteria**:
- [ ] `FormatViolationsSummary()` reads violations JSONL
- [ ] Formats each violation type appropriately
- [ ] Limits output to maxLines (10 by default)
- [ ] Returns nil for missing file
- [ ] Tests verify formatting for all violation types
- [ ] Tests verify maxLines limiting
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Violations summary helps identify systemic routing issues. Top 10 gives overview without overwhelming.

---

## GOgent-031: Implement File Archival

**Time**: 2 hours
**Dependencies**: GOgent-030

**Task**:
Archive transcript, pending learnings, and violations to timestamped files in session-archive/ directory.

**File**: `pkg/session/archive.go`

**Imports**:
```go
package session

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)
```

**Implementation**:
```go
// ArchiveConfig contains paths for file archival
type ArchiveConfig struct {
	ArchiveDir     string
	TranscriptPath string
	PendingPath    string
	ViolationsPath string
}

// ArchiveSession copies session files to archive directory
func ArchiveSession(config *ArchiveConfig) error {
	timestamp := time.Now().Format("20060102-150405")

	// Ensure archive directory exists
	if err := os.MkdirAll(config.ArchiveDir, 0755); err != nil {
		return fmt.Errorf("[archive] Failed to create archive dir: %w", err)
	}

	// Archive transcript if exists
	if config.TranscriptPath != "" && fileExists(config.TranscriptPath) {
		dest := filepath.Join(config.ArchiveDir, fmt.Sprintf("session-%s.jsonl", timestamp))
		if err := copyFile(config.TranscriptPath, dest); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "Warning: Failed to archive transcript: %v\n", err)
		}
	}

	// Archive pending learnings if exists
	if fileExists(config.PendingPath) {
		dest := filepath.Join(config.ArchiveDir, fmt.Sprintf("learnings-%s.jsonl", timestamp))
		if err := moveFile(config.PendingPath, dest); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to archive learnings: %v\n", err)
		}
	}

	// Archive violations if exists
	if fileExists(config.ViolationsPath) {
		dest := filepath.Join(config.ArchiveDir, fmt.Sprintf("violations-%s.jsonl", timestamp))
		if err := moveFile(config.ViolationsPath, dest); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to archive violations: %v\n", err)
		}
	}

	return nil
}

// fileExists checks if file exists and has size > 0
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// copyFile copies src to dest
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// moveFile moves src to dest (copy + delete)
func moveFile(src, dest string) error {
	if err := copyFile(src, dest); err != nil {
		return err
	}

	// Delete source after successful copy
	return os.Remove(src)
}

// CleanupTempFiles removes temporary session files
func CleanupTempFiles() error {
	// Remove tool counters
	matches, _ := filepath.Glob("/tmp/claude-tool-counter-*")
	for _, f := range matches {
		os.Remove(f)
	}

	// Remove error log
	os.Remove("/tmp/claude-error-patterns.jsonl")

	return nil
}
```

**Tests**: `pkg/session/archive_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveSession(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	pendingPath := filepath.Join(tmpDir, "pending.jsonl")
	violationsPath := filepath.Join(tmpDir, "violations.jsonl")

	os.WriteFile(transcriptPath, []byte("transcript content\n"), 0644)
	os.WriteFile(pendingPath, []byte("pending content\n"), 0644)
	os.WriteFile(violationsPath, []byte("violations content\n"), 0644)

	// Archive
	archiveDir := filepath.Join(tmpDir, "archive")
	config := &ArchiveConfig{
		ArchiveDir:     archiveDir,
		TranscriptPath: transcriptPath,
		PendingPath:    pendingPath,
		ViolationsPath: violationsPath,
	}

	err := ArchiveSession(config)
	if err != nil {
		t.Fatalf("ArchiveSession failed: %v", err)
	}

	// Verify archive directory created
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Error("Archive directory not created")
	}

	// Verify transcript copied (should still exist in source)
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		t.Error("Transcript should not be removed (copy, not move)")
	}

	// Verify pending moved (should not exist in source)
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("Pending should be removed (moved)")
	}

	// Verify violations moved
	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations should be removed (moved)")
	}

	// Check archived files exist
	files, _ := os.ReadDir(archiveDir)
	if len(files) != 3 {
		t.Errorf("Expected 3 archived files, got: %d", len(files))
	}
}

func TestArchiveSession_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	archiveDir := filepath.Join(tmpDir, "archive")
	config := &ArchiveConfig{
		ArchiveDir:     archiveDir,
		TranscriptPath: "/nonexistent/transcript.jsonl",
		PendingPath:    "/nonexistent/pending.jsonl",
		ViolationsPath: "/nonexistent/violations.jsonl",
	}

	// Should not error when files missing
	err := ArchiveSession(config)
	if err != nil {
		t.Errorf("Expected no error for missing files, got: %v", err)
	}

	// Archive dir should still be created
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Error("Archive directory should be created even with no files")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// File with content
	existingFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existingFile, []byte("content"), 0644)

	if !fileExists(existingFile) {
		t.Error("Expected existing file to return true")
	}

	// Empty file
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(emptyFile, []byte(""), 0644)

	if fileExists(emptyFile) {
		t.Error("Expected empty file to return false")
	}

	// Nonexistent file
	if fileExists("/nonexistent/file.txt") {
		t.Error("Expected nonexistent file to return false")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dest := filepath.Join(tmpDir, "dest.txt")

	content := "test content\n"
	os.WriteFile(src, []byte(content), 0644)

	err := copyFile(src, dest)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify dest has same content
	destContent, _ := os.ReadFile(dest)
	if string(destContent) != content {
		t.Errorf("Expected content '%s', got: '%s'", content, string(destContent))
	}

	// Verify source still exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("Source should not be removed by copyFile")
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dest := filepath.Join(tmpDir, "dest.txt")

	content := "test content\n"
	os.WriteFile(src, []byte(content), 0644)

	err := moveFile(src, dest)
	if err != nil {
		t.Fatalf("moveFile failed: %v", err)
	}

	// Verify dest exists
	destContent, _ := os.ReadFile(dest)
	if string(destContent) != content {
		t.Errorf("Expected content '%s', got: '%s'", content, string(destContent))
	}

	// Verify source removed
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("Source should be removed by moveFile")
	}
}
```

**Acceptance Criteria**:
- [ ] `ArchiveSession()` creates archive directory
- [ ] Copies transcript to session-{timestamp}.jsonl
- [ ] Moves pending learnings to learnings-{timestamp}.jsonl
- [ ] Moves violations to violations-{timestamp}.jsonl
- [ ] Handles missing files gracefully (no error)
- [ ] `CleanupTempFiles()` removes temp counters and logs
- [ ] Tests verify copy vs move behavior
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Archives preserve full session history. Moving (not copying) learnings/violations prevents re-processing.

---

## GOgent-032: Session Archive Integration Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-031

**Task**:
End-to-end tests for complete session archival workflow.

**File**: `test/integration/session_archive_test.go`

**Implementation**:
```go
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/session"
)

func TestSessionArchiveWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup paths
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	archiveDir := filepath.Join(memoryDir, "session-archive")
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	violationsPath := filepath.Join(memoryDir, "routing-violations.jsonl")
	transcriptPath := filepath.Join(tmpDir, "session-abc.jsonl")

	// Create source files
	os.MkdirAll(memoryDir, 0755)
	os.WriteFile(transcriptPath, []byte(`{"tool": "Read", "file": "test.go"}`+"\n"), 0644)
	os.WriteFile(pendingPath, []byte(`{"file": "src/main.go", "error_type": "nil_pointer", "consecutive_failures": 3}`+"\n"), 0644)
	os.WriteFile(violationsPath, []byte(`{"violation_type": "tool_permission", "tool": "Write"}`+"\n"), 0644)

	// Step 1: Collect metrics
	metrics, err := session.CollectSessionMetrics("test-abc")
	if err != nil {
		t.Fatalf("CollectSessionMetrics failed: %v", err)
	}

	if metrics.SessionID != "test-abc" {
		t.Errorf("Expected session_id test-abc, got: %s", metrics.SessionID)
	}

	// Step 2: Generate handoff
	handoffConfig := session.DefaultHandoffConfig(tmpDir)
	err = session.GenerateHandoff(handoffConfig, metrics)
	if err != nil {
		t.Fatalf("GenerateHandoff failed: %v", err)
	}

	// Verify handoff created
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Fatal("Handoff file not created")
	}

	// Read handoff content
	handoffContent, _ := os.ReadFile(handoffPath)
	handoffStr := string(handoffContent)

	// Verify sections
	if !strings.Contains(handoffStr, "Session Handoff") {
		t.Error("Handoff missing header")
	}

	if !strings.Contains(handoffStr, "Session Metrics") {
		t.Error("Handoff missing metrics section")
	}

	if !strings.Contains(handoffStr, "Pending Learnings") {
		t.Error("Handoff missing learnings section")
	}

	if !strings.Contains(handoffStr, "nil_pointer") {
		t.Error("Handoff should include pending learning")
	}

	// Step 3: Archive files
	archiveConfig := &session.ArchiveConfig{
		ArchiveDir:     archiveDir,
		TranscriptPath: transcriptPath,
		PendingPath:    pendingPath,
		ViolationsPath: violationsPath,
	}

	err = session.ArchiveSession(archiveConfig)
	if err != nil {
		t.Fatalf("ArchiveSession failed: %v", err)
	}

	// Verify archived files
	archivedFiles, _ := os.ReadDir(archiveDir)
	if len(archivedFiles) != 3 {
		t.Errorf("Expected 3 archived files, got: %d", len(archivedFiles))
	}

	// Verify pending learnings moved (not in original location)
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("Pending learnings should be moved to archive")
	}

	// Step 4: Cleanup
	err = session.CleanupTempFiles()
	if err != nil {
		t.Errorf("CleanupTempFiles failed: %v", err)
	}

	t.Log("✓ Complete session archive workflow successful")
}

func TestSessionArchive_NoLearnings(t *testing.T) {
	tmpDir := t.TempDir()

	// No pending learnings file
	handoffConfig := session.DefaultHandoffConfig(tmpDir)
	metrics := &session.SessionMetrics{
		ToolCalls:  10,
		SessionID:  "test-no-learnings",
	}

	err := session.GenerateHandoff(handoffConfig, metrics)
	if err != nil {
		t.Fatalf("GenerateHandoff failed: %v", err)
	}

	// Read handoff
	content, _ := os.ReadFile(handoffConfig.HandoffPath)
	contentStr := string(content)

	// Should have message about no learnings
	if !strings.Contains(contentStr, "No new sharp edges") {
		t.Error("Expected 'no sharp edges' message")
	}

	t.Log("✓ Handoff handles missing learnings gracefully")
}
```

**Acceptance Criteria**:
- [ ] Test executes full workflow: metrics → handoff → archive → cleanup
- [ ] Verifies handoff contains all sections
- [ ] Verifies pending learnings formatted in handoff
- [ ] Verifies files archived to correct locations
- [ ] Verifies pending learnings moved (not copied)
- [ ] Test covers scenario with no learnings
- [ ] `go test ./test/integration` passes

**Why This Matters**: Integration test validates entire session archive pipeline works end-to-end.

---

## GOgent-033: Build gogent-archive CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-032

**Task**:
Build CLI binary that reads SessionEnd event, orchestrates archival workflow.

**File**: `cmd/gogent-archive/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/session"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Get project directory
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	// Parse session event from STDIN
	event, err := session.ParseSessionEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse session event: %v", err))
		os.Exit(1)
	}

	// Collect session metrics
	metrics, err := session.CollectSessionMetrics(event.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to collect metrics: %v\n", err)
		metrics = &session.SessionMetrics{SessionID: event.SessionID}
	}

	// Generate handoff document
	handoffConfig := session.DefaultHandoffConfig(projectDir)
	if err := session.GenerateHandoff(handoffConfig, metrics); err != nil {
		outputError(fmt.Sprintf("Failed to generate handoff: %v", err))
		os.Exit(1)
	}

	// Archive session files
	archiveDir := filepath.Join(projectDir, ".claude", "memory", "session-archive")
	archiveConfig := &session.ArchiveConfig{
		ArchiveDir:     archiveDir,
		TranscriptPath: event.TranscriptPath,
		PendingPath:    handoffConfig.PendingPath,
		ViolationsPath: handoffConfig.ViolationsPath,
	}

	if err := session.ArchiveSession(archiveConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to archive files: %v\n", err)
	}

	// Cleanup temp files
	if err := session.CleanupTempFiles(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup temp files: %v\n", err)
	}

	// Output success
	outputSuccess(handoffConfig.HandoffPath)
}

// outputSuccess writes success message in hook format
func outputSuccess(handoffPath string) {
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": fmt.Sprintf("📦 SESSION ARCHIVED: Handoff saved to %s. Review pending learnings on next session start.", handoffPath),
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

// outputError writes error message in hook format
func outputError(message string) {
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": "🔴 " + message,
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}
```

**Build/Install Scripts**: Create `scripts/build-archive.sh` and `scripts/install-archive.sh` following same pattern as GOgent-025.

**Acceptance Criteria**:
- [ ] CLI reads SessionEnd event from STDIN
- [ ] Collects session metrics
- [ ] Generates handoff document
- [ ] Archives transcript, learnings, violations
- [ ] Cleans up temp files
- [ ] Outputs success message with handoff path
- [ ] Build script creates bin/gogent-archive
- [ ] Installation script copies to ~/.local/bin
- [ ] Manual test: `echo '{"session_id":"test","transcript_path":"/tmp/test.jsonl","hook_event_name":"SessionEnd"}' | ./bin/gogent-archive`

**Why This Matters**: CLI binary is the SessionEnd hook implementation. Must be reliable and complete workflow quickly.

---

## Cross-File References

- **Depends on**: [03-week1-validation-cli.md](03-week1-validation-cli.md) - GOgent-011 (violation logging)
- **Used by**: [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md) - Sharp edge detector writes to pending-learnings.jsonl
- **Standards**: [00-overview.md](00-overview.md) - STDIN timeout, error format

---

## Quick Reference

**Key Functions Added**:
- `session.ParseSessionEvent()` - Parse SessionEnd events
- `session.CollectSessionMetrics()` - Count tools, errors, violations
- `session.GenerateHandoff()` - Create markdown handoff document
- `session.FormatPendingLearnings()` - Format sharp edges as markdown
- `session.FormatViolationsSummary()` - Format top 10 violations
- `session.ArchiveSession()` - Copy/move files to archive
- `gogent-archive` CLI - SessionEnd → archive workflow

**Files Created**:
- `pkg/session/events.go`
- `pkg/session/metrics.go`
- `pkg/session/handoff.go`
- `pkg/session/learnings.go`
- `pkg/session/violations_summary.go`
- `pkg/session/archive.go`
- `cmd/gogent-archive/main.go`
- `test/integration/session_archive_test.go`

**Total Lines**: ~1300 lines of implementation + tests

---

## Completion Checklist

- [ ] All 8 tickets (GOgent-026 to 033) complete
- [ ] All functions have complete imports
- [ ] Error messages use `[component] What. Why. How.` format
- [ ] STDIN timeout implemented (5s)
- [ ] Tests cover positive, negative, edge cases
- [ ] Test coverage ≥80%
- [ ] All acceptance criteria filled
- [ ] CLI binary buildable
- [ ] Manual tests successful
- [ ] No placeholders

---

**Next**: [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md) - GOgent-034 to 040 (Sharp edge detector translation)
