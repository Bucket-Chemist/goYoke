# Week 2 Part 2: Sharp Edge Detection and Memory

**File**: `05-week2-sharp-edge-memory.md`
**Tickets**: GOgent-034 to 040 (7 tickets)
**Total Time**: ~11 hours
**Phase**: Week 2 Part 2

---

## Navigation

- **Previous**: [04-week2-session-archive.md](04-week2-session-archive.md) - GOgent-026 to 033
- **Next**: [06-week3-integration-tests.md](06-week3-integration-tests.md) - GOgent-041 to 047
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure

---

## Summary

This file translates the sharp-edge-detector.sh hook from Bash to Go:

1. **PostToolUse Event Parsing**: Handle Bash, Edit, Write, Task tool responses
2. **Failure Detection**: Identify errors via exit codes, keywords, explicit flags
3. **Consecutive Failure Tracking**: Count failures on same file within time window
4. **Sharp Edge Capture**: Auto-log to pending-learnings.jsonl at threshold
5. **Blocking Response**: Prevent further attempts after 3 failures
6. **Warning System**: Alert at 2 failures before blocking
7. **CLI Integration**: Build gogent-sharp-edge binary

**Critical Context**:
- PostToolUse events include tool_response with success/failure signals
- Failure window: 5 minutes (configurable via CLAUDE_FAILURE_WINDOW)
- Max failures: 3 (configurable via CLAUDE_MAX_FAILURES)
- Sharp edges logged to `.claude/memory/pending-learnings.jsonl`
- Session-archive hook formats these for handoff document

---

## GOgent-034: Define PostToolUse Event Structs

**Time**: 1.5 hours
**Dependencies**: GOgent-007 (base event parsing)

**Task**:
Define PostToolUseEvent struct with tool_response field containing success/failure signals.

**File**: `pkg/memory/events.go`

**Imports**:
```go
package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// PostToolUseEvent represents PostToolUse hook event
type PostToolUseEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  map[string]interface{} `json:"tool_response"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
}

// ToolResponse contains success/failure signals from tool execution
type ToolResponse struct {
	Success  *bool  `json:"success,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ParsePostToolUseEvent reads PostToolUse event from STDIN
func ParsePostToolUseEvent(r io.Reader, timeout time.Duration) (*PostToolUseEvent, error) {
	type result struct {
		event *PostToolUseEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[sharp-edge] Failed to read STDIN: %w", err)}
			return
		}

		var event PostToolUseEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[sharp-edge] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[sharp-edge] STDIN read timeout after %v", timeout)
	}
}

// ExtractFilePath gets file path from tool input
func (e *PostToolUseEvent) ExtractFilePath() string {
	// Try file_path first (Edit, Write, Read)
	if filePath, ok := e.ToolInput["file_path"].(string); ok {
		return filePath
	}

	// Try command (Bash)
	if command, ok := e.ToolInput["command"].(string); ok {
		return command
	}

	return "unknown"
}

// ParseToolResponse extracts structured response from tool_response map
func (e *PostToolUseEvent) ParseToolResponse() *ToolResponse {
	resp := &ToolResponse{}

	if success, ok := e.ToolResponse["success"].(bool); ok {
		resp.Success = &success
	}

	if exitCode, ok := e.ToolResponse["exit_code"].(float64); ok {
		code := int(exitCode)
		resp.ExitCode = &code
	}

	if output, ok := e.ToolResponse["output"].(string); ok {
		resp.Output = output
	}

	if errMsg, ok := e.ToolResponse["error"].(string); ok {
		resp.Error = errMsg
	}

	return resp
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Tests**: `pkg/memory/events_test.go`

```go
package memory

import (
	"strings"
	"testing"
	"time"
)

func TestParsePostToolUseEvent_Valid(t *testing.T) {
	jsonInput := `{
		"tool_name": "Edit",
		"tool_input": {"file_path": "src/main.go", "old_string": "foo", "new_string": "bar"},
		"tool_response": {"success": true},
		"session_id": "test-123",
		"hook_event_name": "PostToolUse"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParsePostToolUseEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.ToolName != "Edit" {
		t.Errorf("Expected tool_name Edit, got: %s", event.ToolName)
	}

	if event.ExtractFilePath() != "src/main.go" {
		t.Errorf("Expected file_path src/main.go, got: %s", event.ExtractFilePath())
	}
}

func TestExtractFilePath_FilePath(t *testing.T) {
	event := &PostToolUseEvent{
		ToolInput: map[string]interface{}{
			"file_path": "/path/to/file.go",
		},
	}

	filePath := event.ExtractFilePath()
	if filePath != "/path/to/file.go" {
		t.Errorf("Expected /path/to/file.go, got: %s", filePath)
	}
}

func TestExtractFilePath_Command(t *testing.T) {
	event := &PostToolUseEvent{
		ToolInput: map[string]interface{}{
			"command": "go test ./...",
		},
	}

	filePath := event.ExtractFilePath()
	if filePath != "go test ./..." {
		t.Errorf("Expected command, got: %s", filePath)
	}
}

func TestExtractFilePath_Unknown(t *testing.T) {
	event := &PostToolUseEvent{
		ToolInput: map[string]interface{}{},
	}

	filePath := event.ExtractFilePath()
	if filePath != "unknown" {
		t.Errorf("Expected 'unknown', got: %s", filePath)
	}
}

func TestParseToolResponse(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"success":   false,
			"exit_code": float64(1),
			"output":    "some output",
			"error":     "error message",
		},
	}

	resp := event.ParseToolResponse()

	if resp.Success == nil || *resp.Success != false {
		t.Error("Expected success=false")
	}

	if resp.ExitCode == nil || *resp.ExitCode != 1 {
		t.Error("Expected exit_code=1")
	}

	if resp.Output != "some output" {
		t.Errorf("Expected output, got: %s", resp.Output)
	}

	if resp.Error != "error message" {
		t.Errorf("Expected error, got: %s", resp.Error)
	}
}
```

**Acceptance Criteria**:
- [ ] `ParsePostToolUseEvent()` reads PostToolUse events from STDIN
- [ ] Implements 5s timeout on STDIN read
- [ ] `ExtractFilePath()` tries file_path, then command, then "unknown"
- [ ] `ParseToolResponse()` extracts success, exit_code, output, error fields
- [ ] Tests cover valid event, file path extraction variants, response parsing
- [ ] `go test ./pkg/memory` passes

**Why This Matters**: PostToolUseEvent is trigger for failure detection. Must parse all relevant signals correctly.

---

## GOgent-035: Implement Failure Detection

**Time**: 2 hours
**Dependencies**: GOgent-034

**Task**:
Detect failures from exit codes, error keywords in output, and explicit success=false flags.

**File**: `pkg/memory/failure_detection.go`

**Imports**:
```go
package memory

import (
	"regexp"
	"strings"
)
```

**Implementation**:
```go
// FailureDetection represents detected failure information
type FailureDetection struct {
	IsError    bool
	ErrorType  string
	ErrorMatch string // The specific error keyword that matched
}

// DetectFailure analyzes tool response for failure signals
func DetectFailure(event *PostToolUseEvent) *FailureDetection {
	detection := &FailureDetection{
		IsError:   false,
		ErrorType: "unknown",
	}

	resp := event.ParseToolResponse()

	// Check 1: Explicit success=false
	if resp.Success != nil && !*resp.Success {
		detection.IsError = true
		detection.ErrorType = "explicit_failure"
		return detection
	}

	// Check 2: Non-zero exit code
	if resp.ExitCode != nil && *resp.ExitCode != 0 {
		detection.IsError = true
		detection.ErrorType = formatExitCode(*resp.ExitCode)
		return detection
	}

	// Check 3: Error keywords in output or error field
	combined := resp.Output + " " + resp.Error

	if errorType, match := detectErrorKeywords(combined); errorType != "" {
		detection.IsError = true
		detection.ErrorType = errorType
		detection.ErrorMatch = match
		return detection
	}

	return detection
}

// detectErrorKeywords searches for common error patterns
func detectErrorKeywords(text string) (string, string) {
	// Specific error types (check these first for precision)
	specificErrors := map[string]*regexp.Regexp{
		"TypeError":      regexp.MustCompile(`(?i)TypeError`),
		"ValueError":     regexp.MustCompile(`(?i)ValueError`),
		"AttributeError": regexp.MustCompile(`(?i)AttributeError`),
		"ImportError":    regexp.MustCompile(`(?i)ImportError`),
		"SyntaxError":    regexp.MustCompile(`(?i)SyntaxError`),
		"NameError":      regexp.MustCompile(`(?i)NameError`),
		"KeyError":       regexp.MustCompile(`(?i)KeyError`),
		"IndexError":     regexp.MustCompile(`(?i)IndexError`),
	}

	for errorType, pattern := range specificErrors {
		if pattern.MatchString(text) {
			return errorType, errorType
		}
	}

	// Generic error keywords
	genericPatterns := []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{regexp.MustCompile(`(?i)\berror\b`), "generic_error"},
		{regexp.MustCompile(`(?i)\bfailed\b`), "generic_failure"},
		{regexp.MustCompile(`(?i)\bexception\b`), "exception"},
		{regexp.MustCompile(`(?i)\btraceback\b`), "traceback"},
	}

	for _, p := range genericPatterns {
		if match := p.pattern.FindString(text); match != "" {
			return p.name, match
		}
	}

	return "", ""
}

// formatExitCode creates error type from exit code
func formatExitCode(code int) string {
	return fmt.Sprintf("exit_code_%d", code)
}
```

**Tests**: `pkg/memory/failure_detection_test.go`

```go
package memory

import (
	"testing"
)

func TestDetectFailure_ExplicitFailure(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"success": false,
		},
	}

	detection := DetectFailure(event)

	if !detection.IsError {
		t.Error("Expected error detection for success=false")
	}

	if detection.ErrorType != "explicit_failure" {
		t.Errorf("Expected explicit_failure, got: %s", detection.ErrorType)
	}
}

func TestDetectFailure_ExitCode(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"exit_code": float64(1),
		},
	}

	detection := DetectFailure(event)

	if !detection.IsError {
		t.Error("Expected error detection for exit_code=1")
	}

	if detection.ErrorType != "exit_code_1" {
		t.Errorf("Expected exit_code_1, got: %s", detection.ErrorType)
	}
}

func TestDetectFailure_TypeError(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"output": "Traceback: TypeError: 'NoneType' object is not callable",
		},
	}

	detection := DetectFailure(event)

	if !detection.IsError {
		t.Error("Expected error detection for TypeError")
	}

	if detection.ErrorType != "TypeError" {
		t.Errorf("Expected TypeError, got: %s", detection.ErrorType)
	}
}

func TestDetectFailure_GenericError(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"output": "Command failed with error",
		},
	}

	detection := DetectFailure(event)

	if !detection.IsError {
		t.Error("Expected error detection for generic error")
	}

	// Should match either "error" or "failed"
	if detection.ErrorType == "" {
		t.Error("Expected error type to be set")
	}
}

func TestDetectFailure_Success(t *testing.T) {
	event := &PostToolUseEvent{
		ToolResponse: map[string]interface{}{
			"success": true,
			"output":  "Operation completed successfully",
		},
	}

	detection := DetectFailure(event)

	if detection.IsError {
		t.Error("Expected no error detection for successful operation")
	}
}

func TestDetectErrorKeywords(t *testing.T) {
	tests := []struct {
		text          string
		expectedType  string
		shouldBeError bool
	}{
		{"ValueError: invalid literal", "ValueError", true},
		{"ImportError: No module named foo", "ImportError", true},
		{"SyntaxError: invalid syntax", "SyntaxError", true},
		{"operation failed", "generic_failure", true},
		{"an error occurred", "generic_error", true},
		{"All tests passed", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.text[:min(30, len(tt.text))], func(t *testing.T) {
			errorType, _ := detectErrorKeywords(tt.text)

			if tt.shouldBeError && errorType == "" {
				t.Errorf("Expected error detection, got none for: %s", tt.text)
			}

			if !tt.shouldBeError && errorType != "" {
				t.Errorf("Expected no error, got: %s for: %s", errorType, tt.text)
			}

			if tt.shouldBeError && tt.expectedType != "" && errorType != tt.expectedType {
				t.Errorf("Expected type %s, got: %s", tt.expectedType, errorType)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Acceptance Criteria**:
- [ ] `DetectFailure()` detects explicit success=false
- [ ] Detects non-zero exit codes
- [ ] Detects specific Python error types (TypeError, ValueError, etc.)
- [ ] Detects generic error keywords (error, failed, exception)
- [ ] Returns no error for successful operations
- [ ] Tests cover all detection paths
- [ ] `go test ./pkg/memory` passes with ≥80% coverage

**Why This Matters**: Accurate failure detection is critical. False positives block valid work; false negatives miss real issues.

---

## GOgent-036: Implement Consecutive Failure Tracking

**Time**: 2 hours
**Dependencies**: GOgent-035

**Task**:
Track failures per file within sliding time window. Count recent failures to determine if threshold reached.

**File**: `pkg/memory/failure_tracking.go`

**Imports**:
```go
package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/config"
)
```

**Implementation**:
```go
// FailureEntry represents a logged failure
type FailureEntry struct {
	Timestamp int64  `json:"ts"`
	File      string `json:"file"`
	Tool      string `json:"tool"`
	ErrorType string `json:"error_type"`
}

// FailureTracker tracks consecutive failures per file
type FailureTracker struct {
	ErrorLogPath   string
	FailureWindow  int // seconds
	MaxFailures    int
}

// DefaultFailureTracker creates tracker with standard config
func DefaultFailureTracker() *FailureTracker {
	// Read from environment or use defaults
	failureWindow := getEnvInt("CLAUDE_FAILURE_WINDOW", 300)  // 5 minutes
	maxFailures := getEnvInt("CLAUDE_MAX_FAILURES", 3)

	return &FailureTracker{
		ErrorLogPath:  getErrorLogPath(),
		FailureWindow: failureWindow,
		MaxFailures:   maxFailures,
	}
}

// LogFailure appends failure to error log
func (ft *FailureTracker) LogFailure(filePath, tool, errorType string) error {
	entry := FailureEntry{
		Timestamp: time.Now().Unix(),
		File:      filePath,
		Tool:      tool,
		ErrorType: errorType,
	}

	f, err := os.OpenFile(ft.ErrorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[failure-tracker] Failed to open error log: %w", err)
	}
	defer f.Close()

	data, _ := json.Marshal(entry)
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("[failure-tracker] Failed to write error log: %w", err)
	}

	return nil
}

// CountRecentFailures counts failures on file within time window
func (ft *FailureTracker) CountRecentFailures(filePath string) (int, error) {
	f, err := os.Open(ft.ErrorLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // No log file = 0 failures
		}
		return 0, fmt.Errorf("[failure-tracker] Failed to open error log: %w", err)
	}
	defer f.Close()

	cutoff := time.Now().Unix() - int64(ft.FailureWindow)
	count := 0

	// Read last 50 lines (performance optimization)
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 50 {
			lines = lines[1:] // Keep only last 50
		}
	}

	// Count matching failures
	for _, line := range lines {
		var entry FailureEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid lines
		}

		if entry.File == filePath && entry.Timestamp > cutoff {
			count++
		}
	}

	return count, nil
}

// getEnvInt reads int from environment with default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// getErrorLogPath returns path to error log
func getErrorLogPath() string {
	// TODO: Make XDG-compliant in future iteration
	return "/tmp/claude-error-patterns.jsonl"
}
```

**Tests**: `pkg/memory/failure_tracking_test.go`

```go
package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogFailure(t *testing.T) {
	tmpLog := filepath.Join(t.TempDir(), "errors.jsonl")

	tracker := &FailureTracker{
		ErrorLogPath:  tmpLog,
		FailureWindow: 300,
		MaxFailures:   3,
	}

	err := tracker.LogFailure("src/main.go", "Edit", "TypeError")
	if err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	// Verify log file created
	if _, err := os.Stat(tmpLog); os.IsNotExist(err) {
		t.Error("Error log file not created")
	}

	// Verify content
	content, _ := os.ReadFile(tmpLog)
	if len(content) == 0 {
		t.Error("Error log is empty")
	}
}

func TestCountRecentFailures(t *testing.T) {
	tmpLog := filepath.Join(t.TempDir(), "errors.jsonl")

	tracker := &FailureTracker{
		ErrorLogPath:  tmpLog,
		FailureWindow: 300, // 5 minutes
		MaxFailures:   3,
	}

	now := time.Now().Unix()

	// Create log with mixed failures
	entries := []FailureEntry{
		{Timestamp: now - 100, File: "src/main.go", Tool: "Edit", ErrorType: "TypeError"},
		{Timestamp: now - 200, File: "src/main.go", Tool: "Edit", ErrorType: "TypeError"},
		{Timestamp: now - 400, File: "src/main.go", Tool: "Edit", ErrorType: "TypeError"}, // Outside window
		{Timestamp: now - 150, File: "src/other.go", Tool: "Edit", ErrorType: "SyntaxError"},
	}

	f, _ := os.Create(tmpLog)
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
	f.Close()

	// Count failures on src/main.go
	count, err := tracker.CountRecentFailures("src/main.go")
	if err != nil {
		t.Fatalf("CountRecentFailures failed: %v", err)
	}

	// Should count 2 (first two entries, third is outside window)
	if count != 2 {
		t.Errorf("Expected 2 recent failures, got: %d", count)
	}

	// Count on different file
	count, err = tracker.CountRecentFailures("src/other.go")
	if err != nil {
		t.Fatalf("CountRecentFailures failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 failure for other.go, got: %d", count)
	}
}

func TestCountRecentFailures_NoLog(t *testing.T) {
	tracker := &FailureTracker{
		ErrorLogPath:  "/nonexistent/errors.jsonl",
		FailureWindow: 300,
		MaxFailures:   3,
	}

	count, err := tracker.CountRecentFailures("src/main.go")
	if err != nil {
		t.Errorf("Expected no error for missing log, got: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 for missing log, got: %d", count)
	}
}

func TestDefaultFailureTracker(t *testing.T) {
	tracker := DefaultFailureTracker()

	if tracker.FailureWindow != 300 {
		t.Errorf("Expected default window 300s, got: %d", tracker.FailureWindow)
	}

	if tracker.MaxFailures != 3 {
		t.Errorf("Expected default max 3, got: %d", tracker.MaxFailures)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test default
	val := getEnvInt("NONEXISTENT_VAR", 42)
	if val != 42 {
		t.Errorf("Expected default 42, got: %d", val)
	}

	// Test with env var
	os.Setenv("TEST_INT_VAR", "99")
	defer os.Unsetenv("TEST_INT_VAR")

	val = getEnvInt("TEST_INT_VAR", 42)
	if val != 99 {
		t.Errorf("Expected 99 from env, got: %d", val)
	}
}
```

**Acceptance Criteria**:
- [ ] `LogFailure()` appends JSONL entry to error log
- [ ] `CountRecentFailures()` counts failures within time window
- [ ] Ignores failures outside window
- [ ] Filters by file path
- [ ] Handles missing log file (returns 0, not error)
- [ ] Reads FailureWindow and MaxFailures from environment
- [ ] Tests verify window filtering, file filtering
- [ ] `go test ./pkg/memory` passes

**Why This Matters**: Consecutive failure tracking is the core of sharp edge detection. Must be accurate within sliding window.

---

## GOgent-037: Implement Sharp Edge Capture

**Time**: 1.5 hours
**Dependencies**: GOgent-036

**Task**:
Write sharp edge entry to pending-learnings.jsonl when failure threshold reached.

**File**: `pkg/memory/sharp_edge.go`

**Imports**:
```go
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```

**Implementation**:
```go
// SharpEdge represents a captured debugging loop
type SharpEdge struct {
	Timestamp           int64  `json:"ts"`
	Type                string `json:"type"`
	File                string `json:"file"`
	Tool                string `json:"tool"`
	ErrorType           string `json:"error_type"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Status              string `json:"status"`
}

// CaptureSharpEdge logs sharp edge to pending learnings
func CaptureSharpEdge(projectDir, filePath, tool, errorType string, failureCount int) error {
	edge := SharpEdge{
		Timestamp:           time.Now().Unix(),
		Type:                "sharp_edge",
		File:                filePath,
		Tool:                tool,
		ErrorType:           errorType,
		ConsecutiveFailures: failureCount,
		Status:              "pending_review",
	}

	// Get pending learnings path
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(pendingPath), 0755)

	// Open in append mode
	f, err := os.OpenFile(pendingPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[sharp-edge] Failed to open pending learnings: %w", err)
	}
	defer f.Close()

	// Write JSONL entry
	data, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("[sharp-edge] Failed to marshal edge: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("[sharp-edge] Failed to write edge: %w", err)
	}

	return nil
}
```

**Tests**: `pkg/memory/sharp_edge_test.go`

```go
package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureSharpEdge(t *testing.T) {
	tmpDir := t.TempDir()

	err := CaptureSharpEdge(tmpDir, "src/main.go", "Edit", "TypeError", 3)
	if err != nil {
		t.Fatalf("CaptureSharpEdge failed: %v", err)
	}

	// Verify file created
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Fatal("Pending learnings file not created")
	}

	// Read and verify content
	data, _ := os.ReadFile(pendingPath)

	var edge SharpEdge
	if err := json.Unmarshal(data, &edge); err != nil {
		t.Fatalf("Failed to parse edge: %v", err)
	}

	if edge.File != "src/main.go" {
		t.Errorf("Expected file src/main.go, got: %s", edge.File)
	}

	if edge.ErrorType != "TypeError" {
		t.Errorf("Expected TypeError, got: %s", edge.ErrorType)
	}

	if edge.ConsecutiveFailures != 3 {
		t.Errorf("Expected 3 failures, got: %d", edge.ConsecutiveFailures)
	}

	if edge.Status != "pending_review" {
		t.Errorf("Expected pending_review, got: %s", edge.Status)
	}
}

func TestCaptureSharpEdge_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Capture multiple edges
	CaptureSharpEdge(tmpDir, "src/main.go", "Edit", "TypeError", 3)
	CaptureSharpEdge(tmpDir, "pkg/utils.go", "Write", "SyntaxError", 4)

	// Read all edges
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
	data, _ := os.ReadFile(pendingPath)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 edges, got: %d", len(lines))
	}
}
```

**Acceptance Criteria**:
- [ ] `CaptureSharpEdge()` writes to pending-learnings.jsonl
- [ ] Creates directory if doesn't exist
- [ ] Appends (doesn't overwrite) multiple edges
- [ ] Includes all required fields (timestamp, type, file, tool, error_type, failures, status)
- [ ] Tests verify single and multiple captures
- [ ] `go test ./pkg/memory` passes

**Why This Matters**: Sharp edge capture preserves debugging loop context for review. Critical for system improvement.

---

## GOgent-038: Implement Hook Response Generation

**Time**: 1.5 hours
**Dependencies**: GOgent-037

**Task**:
Generate appropriate hook responses: blocking at threshold, warning at threshold-1, pass-through otherwise.

**File**: `pkg/memory/responses.go`

**Imports**:
```go
package memory

import (
	"encoding/json"
	"fmt"
)
```

**Implementation**:
```go
// HookResponse represents output for Claude Code hook system
type HookResponse struct {
	Decision           string                 `json:"decision,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	HookSpecificOutput map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}

// GenerateBlockingResponse creates response that blocks further attempts
func GenerateBlockingResponse(filePath, errorType string, failureCount int) *HookResponse {
	return &HookResponse{
		Decision: "block",
		Reason:   fmt.Sprintf("⚠️ SHARP EDGE DETECTED: %d consecutive failures on '%s' (%s)", failureCount, filePath, errorType),
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": "PostToolUse",
			"additionalContext": fmt.Sprintf(
				"🔴 DEBUGGING LOOP DETECTED (%d failures on %s):\n"+
					"1. STOP current approach\n"+
					"2. Document this sharp edge (auto-logged to pending-learnings.jsonl)\n"+
					"3. Analyze root cause - what assumption might be wrong?\n"+
					"4. Consider escalation to next tier\n"+
					"5. Check sharp-edges.yaml for similar patterns",
				failureCount,
				filePath,
			),
		},
	}
}

// GenerateWarningResponse creates response warning of approaching threshold
func GenerateWarningResponse(filePath string, failureCount int) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": "PostToolUse",
			"additionalContext": fmt.Sprintf(
				"⚠️ WARNING: %d failures on '%s'. One more failure triggers sharp edge capture and potential escalation.",
				failureCount,
				filePath,
			),
		},
	}
}

// GeneratePassThroughResponse creates empty response (allow)
func GeneratePassThroughResponse() *HookResponse {
	return &HookResponse{}
}

// ToJSON serializes response to JSON
func (r *HookResponse) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

**Tests**: `pkg/memory/responses_test.go`

```go
package memory

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateBlockingResponse(t *testing.T) {
	resp := GenerateBlockingResponse("src/main.go", "TypeError", 3)

	if resp.Decision != "block" {
		t.Errorf("Expected decision=block, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.Reason, "SHARP EDGE") {
		t.Error("Reason should mention sharp edge")
	}

	if !strings.Contains(resp.Reason, "src/main.go") {
		t.Error("Reason should include file path")
	}

	context, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Missing additionalContext")
	}

	if !strings.Contains(context, "DEBUGGING LOOP") {
		t.Error("Context should mention debugging loop")
	}

	if !strings.Contains(context, "pending-learnings.jsonl") {
		t.Error("Context should mention where logged")
	}
}

func TestGenerateWarningResponse(t *testing.T) {
	resp := GenerateWarningResponse("pkg/utils.go", 2)

	if resp.Decision != "" {
		t.Error("Warning should not have decision (allows)")
	}

	context, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Missing additionalContext")
	}

	if !strings.Contains(context, "WARNING") {
		t.Error("Context should contain warning")
	}

	if !strings.Contains(context, "2 failures") {
		t.Error("Context should mention failure count")
	}

	if !strings.Contains(context, "pkg/utils.go") {
		t.Error("Context should mention file")
	}
}

func TestGeneratePassThroughResponse(t *testing.T) {
	resp := GeneratePassThroughResponse()

	if resp.Decision != "" {
		t.Error("Pass-through should have no decision")
	}

	if resp.Reason != "" {
		t.Error("Pass-through should have no reason")
	}
}

func TestToJSON(t *testing.T) {
	resp := GenerateBlockingResponse("test.go", "error", 3)

	jsonStr, err := resp.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["decision"] != "block" {
		t.Error("JSON should contain decision")
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateBlockingResponse()` creates decision=block with context
- [ ] Includes 5-step debugging guidance
- [ ] `GenerateWarningResponse()` creates warning without blocking
- [ ] `GeneratePassThroughResponse()` returns empty response
- [ ] `ToJSON()` produces valid JSON
- [ ] Tests verify all response types
- [ ] `go test ./pkg/memory` passes

**Why This Matters**: Hook responses control Claude behavior. Blocking stops wasteful loops; warnings provide early signal.

---

## GOgent-039: Sharp Edge Detection Integration Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-038

**Task**:
End-to-end tests for complete sharp edge detection workflow.

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

	"github.com/yourusername/gogent-fortress/pkg/memory"
)

func TestSharpEdgeDetectionWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	tmpLog := filepath.Join(tmpDir, "errors.jsonl")

	tracker := &memory.FailureTracker{
		ErrorLogPath:  tmpLog,
		FailureWindow: 300,
		MaxFailures:   3,
	}

	filePath := "src/main.go"

	// Simulate 3 consecutive failures
	for i := 1; i <= 3; i++ {
		t.Logf("Failure %d", i)

		// Log failure
		err := tracker.LogFailure(filePath, "Edit", "TypeError")
		if err != nil {
			t.Fatalf("LogFailure failed: %v", err)
		}

		// Count failures
		count, err := tracker.CountRecentFailures(filePath)
		if err != nil {
			t.Fatalf("CountRecentFailures failed: %v", err)
		}

		if count != i {
			t.Errorf("Expected %d failures, got: %d", i, count)
		}

		// Generate appropriate response
		var resp *memory.HookResponse

		if count >= tracker.MaxFailures {
			// Capture sharp edge
			err := memory.CaptureSharpEdge(tmpDir, filePath, "Edit", "TypeError", count)
			if err != nil {
				t.Fatalf("CaptureSharpEdge failed: %v", err)
			}

			resp = memory.GenerateBlockingResponse(filePath, "TypeError", count)

			if resp.Decision != "block" {
				t.Error("Expected blocking response at threshold")
			}

			t.Log("✓ Sharp edge captured and blocked")

		} else if count == tracker.MaxFailures-1 {
			resp = memory.GenerateWarningResponse(filePath, count)

			context := resp.HookSpecificOutput["additionalContext"].(string)
			if !strings.Contains(context, "WARNING") {
				t.Error("Expected warning at threshold-1")
			}

			t.Log("✓ Warning issued")

		} else {
			resp = memory.GeneratePassThroughResponse()
			t.Log("✓ Pass-through (below threshold)")
		}
	}

	// Verify sharp edge captured
	pendingPath := filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Fatal("Sharp edge not captured to pending learnings")
	}

	// Read and verify
	data, _ := os.ReadFile(pendingPath)
	var edge memory.SharpEdge
	json.Unmarshal(data, &edge)

	if edge.File != filePath {
		t.Errorf("Expected file %s, got: %s", filePath, edge.File)
	}

	if edge.ConsecutiveFailures != 3 {
		t.Errorf("Expected 3 failures, got: %d", edge.ConsecutiveFailures)
	}

	t.Log("✓ Complete sharp edge workflow successful")
}

func TestSharpEdgeDetection_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	tmpLog := filepath.Join(tmpDir, "errors.jsonl")

	tracker := &memory.FailureTracker{
		ErrorLogPath:  tmpLog,
		FailureWindow: 300,
		MaxFailures:   3,
	}

	// Failures on file A
	tracker.LogFailure("fileA.go", "Edit", "TypeError")
	tracker.LogFailure("fileA.go", "Edit", "TypeError")

	// Failures on file B
	tracker.LogFailure("fileB.go", "Write", "SyntaxError")

	// Check counts
	countA, _ := tracker.CountRecentFailures("fileA.go")
	countB, _ := tracker.CountRecentFailures("fileB.go")

	if countA != 2 {
		t.Errorf("Expected 2 failures on fileA, got: %d", countA)
	}

	if countB != 1 {
		t.Errorf("Expected 1 failure on fileB, got: %d", countB)
	}

	t.Log("✓ Per-file tracking works correctly")
}

func TestFailureDetection_VariousSignals(t *testing.T) {
	tests := []struct {
		name      string
		response  map[string]interface{}
		shouldErr bool
		errorType string
	}{
		{
			name:      "Explicit failure",
			response:  map[string]interface{}{"success": false},
			shouldErr: true,
			errorType: "explicit_failure",
		},
		{
			name:      "Exit code",
			response:  map[string]interface{}{"exit_code": float64(1)},
			shouldErr: true,
			errorType: "exit_code_1",
		},
		{
			name:      "TypeError in output",
			response:  map[string]interface{}{"output": "TypeError: invalid type"},
			shouldErr: true,
			errorType: "TypeError",
		},
		{
			name:      "Success",
			response:  map[string]interface{}{"success": true},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &memory.PostToolUseEvent{
				ToolResponse: tt.response,
			}

			detection := memory.DetectFailure(event)

			if detection.IsError != tt.shouldErr {
				t.Errorf("Expected error=%v, got: %v", tt.shouldErr, detection.IsError)
			}

			if tt.shouldErr && tt.errorType != "" && detection.ErrorType != tt.errorType {
				t.Errorf("Expected type %s, got: %s", tt.errorType, detection.ErrorType)
			}
		})
	}

	t.Log("✓ Failure detection handles all signal types")
}
```

**Acceptance Criteria**:
- [ ] Test simulates 3 consecutive failures
- [ ] Verifies warning at failure 2
- [ ] Verifies blocking at failure 3
- [ ] Verifies sharp edge captured to pending-learnings.jsonl
- [ ] Test covers multiple files (per-file tracking)
- [ ] Test covers various failure signals (exit code, keywords, explicit)
- [ ] `go test ./test/integration` passes

**Why This Matters**: Integration test validates complete detection pipeline works correctly.

---

## GOgent-040: Build gogent-sharp-edge CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-039

**Task**:
Build CLI binary that reads PostToolUse events, detects failures, tracks consecutive failures, captures sharp edges.

**File**: `cmd/gogent-sharp-edge/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/memory"
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

	// Parse PostToolUse event
	event, err := memory.ParsePostToolUseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only process specific tools (Bash, Edit, Write, Task)
	relevantTools := map[string]bool{
		"Bash": true, "Edit": true, "Write": true, "Task": true,
	}

	if !relevantTools[event.ToolName] {
		// Pass through for other tools
		fmt.Println("{}")
		return
	}

	// Detect failure
	detection := memory.DetectFailure(event)

	if !detection.IsError {
		// No error, pass through
		fmt.Println("{}")
		return
	}

	// Log failure
	tracker := memory.DefaultFailureTracker()
	filePath := event.ExtractFilePath()

	if err := tracker.LogFailure(filePath, event.ToolName, detection.ErrorType); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log failure: %v\n", err)
	}

	// Count recent failures
	count, err := tracker.CountRecentFailures(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to count failures: %v\n", err)
		count = 1
	}

	// Generate appropriate response
	var resp *memory.HookResponse

	if count >= tracker.MaxFailures {
		// Capture sharp edge
		if err := memory.CaptureSharpEdge(projectDir, filePath, event.ToolName, detection.ErrorType, count); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to capture sharp edge: %v\n", err)
		}

		resp = memory.GenerateBlockingResponse(filePath, detection.ErrorType, count)

	} else if count == tracker.MaxFailures-1 {
		resp = memory.GenerateWarningResponse(filePath, count)

	} else {
		resp = memory.GeneratePassThroughResponse()
	}

	// Output response
	jsonStr, _ := resp.ToJSON()
	fmt.Println(jsonStr)
}

// outputError writes error in hook format
func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build/Install Scripts**: Create `scripts/build-sharp-edge.sh` and `scripts/install-sharp-edge.sh` following same pattern.

**Acceptance Criteria**:
- [ ] CLI reads PostToolUse events from STDIN
- [ ] Only processes Bash, Edit, Write, Task tools
- [ ] Detects failures via multiple signals
- [ ] Logs failures to error log
- [ ] Counts consecutive failures per file
- [ ] Captures sharp edge at threshold
- [ ] Outputs blocking response at threshold
- [ ] Outputs warning at threshold-1
- [ ] Build script creates bin/gogent-sharp-edge
- [ ] Manual test with simulated failures

**Why This Matters**: CLI binary is PostToolUse hook implementation. Must detect loops reliably without false positives.

---

## Cross-File References

- **Depends on**: [04-week2-session-archive.md](04-week2-session-archive.md) - Session archive formats pending learnings
- **Used by**: Week 3 integration tests will verify sharp edge detection
- **Standards**: [00-overview.md](00-overview.md) - STDIN timeout, error format

---

## Quick Reference

**Key Functions Added**:
- `memory.ParsePostToolUseEvent()` - Parse PostToolUse events
- `memory.DetectFailure()` - Identify failures from tool responses
- `memory.FailureTracker.LogFailure()` - Log failure to error log
- `memory.FailureTracker.CountRecentFailures()` - Count within window
- `memory.CaptureSharpEdge()` - Write to pending-learnings.jsonl
- `memory.GenerateBlockingResponse()` - Create blocking hook response
- `gogent-sharp-edge` CLI - PostToolUse → detection workflow

**Files Created**:
- `pkg/memory/events.go`
- `pkg/memory/failure_detection.go`
- `pkg/memory/failure_tracking.go`
- `pkg/memory/sharp_edge.go`
- `pkg/memory/responses.go`
- `cmd/gogent-sharp-edge/main.go`
- `test/integration/sharp_edge_test.go`

**Total Lines**: ~1200 lines of implementation + tests

---

## Completion Checklist

- [ ] All 7 tickets (GOgent-034 to 040) complete
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

**Next**: [06-week3-integration-tests.md](06-week3-integration-tests.md) - GOgent-041 to 047 (Integration and regression tests)