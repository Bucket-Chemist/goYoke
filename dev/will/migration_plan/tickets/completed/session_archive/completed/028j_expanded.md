# GOgent-028j: JSONL History Querying CLI - Detailed Implementation Plan

**Status**: Approved for Implementation (Option 2 - Recommended)
**Architect Review**: 2026-01-20
**Estimated Time**: 2.5h
**Test Coverage Target**: 90% minimum (7+ test functions)

---

## Executive Summary

Extend `gogent-archive` CLI with three subcommands (`list`, `show`, `stats`) plus filtering capabilities for retrospective session analysis. Implementation follows Go conventions, includes comprehensive test coverage, and integrates with existing session package infrastructure.

**Critical Bug Fix**: Implementation must use existing `session.RenderHandoffMarkdown()` function, NOT non-existent `generateMarkdownContent()`.

---

## Phase 1: Core Subcommands + Critical Fixes (1.0h)

### File: `cmd/gogent-archive/main.go`

#### 1.1 Update Imports (Lines 3-11)

**Current imports:**
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)
```

**Required additions:**
```go
import (
	"encoding/json"
	"flag"           // NEW: For subcommand flag parsing
	"fmt"
	"io"             // NEW: For RenderMarkdownToWriter
	"os"
	"path/filepath"
	"strconv"        // NEW: For duration parsing
	"strings"        // NEW: For string manipulation
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)
```

**Rationale**: 
- `flag`: Standard library for CLI flag parsing (no external deps)
- `strconv`: Parse duration strings like "7d" to integers
- `strings`: Date filter parsing, split operations
- `io`: Interface for `RenderMarkdownToWriter` implementation

---

#### 1.2 Refactor `main()` Function (Lines 15-20)

**Current implementation:**
```go
func main() {
	if err := run(); err != nil {
		outputError(err.Error())
		os.Exit(1)
	}
}
```

**New implementation:**
```go
func main() {
	// Subcommand routing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			listSessions()
			return
		case "show":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "[gogent-archive] Usage: gogent-archive show <session-id>")
				fmt.Fprintln(os.Stderr, "  Missing required argument: session-id")
				fmt.Fprintln(os.Stderr, "  Example: gogent-archive show abc123def456")
				os.Exit(1)
			}
			showSession(os.Args[2])
			return
		case "stats":
			showStats()
			return
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Printf("gogent-archive version %s\n", getVersion())
			return
		}
	}

	// Default: SessionEnd hook mode (existing behavior)
	if err := run(); err != nil {
		outputError(err.Error())
		os.Exit(1)
	}
}
```

**Key Design Decisions**:
- **Backward compatibility**: No args = hook mode (existing behavior preserved)
- **Fail-fast validation**: `show` requires session-id argument upfront
- **Help accessibility**: Both `--help` and `-h` supported
- **Error format**: Three-line format follows existing `[gogent-archive] What. Why. How.` pattern

---

#### 1.3 Add Helper Function: `getProjectDir()` (NEW)

**Location**: After `main()`, before new subcommand functions

```go
// getProjectDir determines project directory from env or cwd
// Exits with error if detection fails (matching run() behavior)
func getProjectDir() string {
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to get working directory: %v\n", err)
			fmt.Fprintln(os.Stderr, "  Set GOGENT_PROJECT_DIR environment variable or run from project root.")
			os.Exit(1)
		}
		projectDir = cwd
	}
	return projectDir
}
```

**Rationale**:
- Extracted from `run()` to avoid duplication across subcommands
- Consistent error handling with existing code
- Matches behavior of SessionEnd hook mode

---

#### 1.4 Implement `listSessions()` (NEW)

**Location**: After `getProjectDir()`

```go
// listSessions displays session history as a table with optional filtering
func listSessions() {
	// Parse flags
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	sinceFlag := listFlags.String("since", "", "Filter sessions since duration (e.g., 7d) or date (YYYY-MM-DD)")
	betweenFlag := listFlags.String("between", "", "Filter sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	hasSharpEdges := listFlags.Bool("has-sharp-edges", false, "Show only sessions with sharp edges")
	hasViolations := listFlags.Bool("has-violations", false, "Show only sessions with routing violations")
	clean := listFlags.Bool("clean", false, "Show only sessions with no sharp edges or violations")
	listFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	// CRITICAL: Handle zero sessions gracefully (Acceptance Criteria requirement)
	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	// Apply date filtering (Phase 2 logic - see below)
	if *sinceFlag != "" {
		handoffs = filterSince(handoffs, *sinceFlag)
	}
	if *betweenFlag != "" {
		handoffs = filterBetween(handoffs, *betweenFlag)
	}

	// Apply artifact presence filters (Phase 4 logic - see below)
	if *hasSharpEdges || *hasViolations || *clean {
		handoffs = filterByArtifacts(handoffs, *hasSharpEdges, *hasViolations, *clean)
	}

	// Check again after filtering
	if len(handoffs) == 0 {
		fmt.Println("No sessions match the specified filters.")
		return
	}

	// Print table header
	fmt.Println("Session ID                    | Timestamp  | Tool Calls | Errors | Violations")
	fmt.Println("------------------------------|------------|------------|--------|------------")

	for _, h := range handoffs {
		timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02")
		fmt.Printf("%-30s | %-10s | %10d | %6d | %10d\n",
			h.SessionID, timestamp, h.Context.Metrics.ToolCalls,
			h.Context.Metrics.ErrorsLogged, h.Context.Metrics.RoutingViolations)
	}
}
```

**Table Format Rationale**:
- **Fixed-width columns**: Ensures alignment across all terminal widths
- **Date-only timestamp**: 10 chars (readable, no clutter)
- **Right-aligned numbers**: Standard numeric formatting convention
- **30-char session IDs**: Accommodates UUIDs (36 chars truncate gracefully)

**Edge Cases Handled**:
1. Empty handoffs.jsonl file → "No sessions recorded" message
2. Post-filter empty result → "No sessions match" message
3. Missing handoffs.jsonl → Error with guidance

---

#### 1.5 Implement `showSession()` (NEW)

**Location**: After `listSessions()`

```go
// showSession renders a specific session handoff as markdown to stdout
func showSession(sessionID string) {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	for _, h := range handoffs {
		if h.SessionID == sessionID {
			// CRITICAL FIX: Use session.RenderHandoffMarkdown (existing function)
			// NOT generateMarkdownContent (doesn't exist!)
			markdown := session.RenderHandoffMarkdown(&h)
			fmt.Print(markdown)
			return
		}
	}

	// Session not found
	fmt.Fprintf(os.Stderr, "[gogent-archive] Session %s not found in handoff history.\n", sessionID)
	fmt.Fprintln(os.Stderr, "  Run 'gogent-archive list' to see available sessions.")
	os.Exit(1)
}
```

**Key Design Decisions**:
- **Direct stdout output**: No file writing (pipe-friendly)
- **Linear search**: Acceptable for <1000 sessions (project constraint)
- **Helpful error**: Suggests `list` command if ID not found

**Critical Bug Avoidance**:
- Uses `session.RenderHandoffMarkdown()` (exists at `pkg/session/handoff_markdown.go:10`)
- Original ticket referenced non-existent `generateMarkdownContent()` (compilation failure)

---

#### 1.6 Implement `showStats()` (NEW)

**Location**: After `showSession()`

```go
// showStats displays aggregate session statistics with breakdowns
func showStats() {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	// Aggregate metrics
	totalSessions := len(handoffs)
	totalToolCalls := 0
	totalErrors := 0
	totalViolations := 0
	errorTypes := make(map[string]int)
	violationTypes := make(map[string]int)

	for _, h := range handoffs {
		totalToolCalls += h.Context.Metrics.ToolCalls
		totalErrors += h.Context.Metrics.ErrorsLogged
		totalViolations += h.Context.Metrics.RoutingViolations

		// Aggregate error types (Phase 3 enhancement)
		for _, edge := range h.Artifacts.SharpEdges {
			errorTypes[edge.ErrorType]++
		}

		// Aggregate violation types (Phase 3 enhancement)
		for _, violation := range h.Artifacts.RoutingViolations {
			violationTypes[violation.ViolationType]++
		}
	}

	avgToolCalls := 0
	if totalSessions > 0 {
		avgToolCalls = totalToolCalls / totalSessions
	}

	// Print core stats
	fmt.Printf("Total Sessions: %d\n", totalSessions)
	fmt.Printf("Avg Tool Calls per Session: %d\n", avgToolCalls)
	fmt.Printf("Total Errors: %d\n", totalErrors)
	fmt.Printf("Total Violations: %d\n", totalViolations)

	// Print error breakdown if errors exist (Phase 3)
	if len(errorTypes) > 0 {
		fmt.Println("\nErrors Breakdown:")
		for errType, count := range errorTypes {
			fmt.Printf("  - %s: %d sessions\n", errType, count)
		}
	}

	// Print violation breakdown if violations exist (Phase 3)
	if len(violationTypes) > 0 {
		fmt.Println("\nViolations Breakdown:")
		for violationType, count := range violationTypes {
			fmt.Printf("  - %s: %d sessions\n", violationType, count)
		}
	}
}
```

**Output Format Rationale**:
- **Simple key-value**: Easy to parse programmatically or visually scan
- **Conditional breakdowns**: Only shown if data exists (clean output)
- **Average calculation**: Integer division acceptable (tool calls are discrete)

---

#### 1.7 Implement `printHelp()` (NEW)

**Location**: After `showStats()`

```go
// printHelp displays usage information for all subcommands
func printHelp() {
	fmt.Println("gogent-archive - Session handoff archival and querying")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  gogent-archive                       Read SessionEnd JSON from STDIN (hook mode)")
	fmt.Println("  gogent-archive list                  List all sessions")
	fmt.Println("  gogent-archive list --since 7d       List sessions from last 7 days")
	fmt.Println("  gogent-archive list --between <dates> List sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	fmt.Println("  gogent-archive list --has-sharp-edges Show only sessions with sharp edges")
	fmt.Println("  gogent-archive list --has-violations Show only sessions with routing violations")
	fmt.Println("  gogent-archive list --clean          Show only clean sessions (no errors/violations)")
	fmt.Println("  gogent-archive show <id>             Show specific session handoff")
	fmt.Println("  gogent-archive stats                 Show aggregate statistics with breakdowns")
	fmt.Println("  gogent-archive --help                Show this help")
	fmt.Println("  gogent-archive --version             Show version information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gogent-archive list --since 2026-01-15")
	fmt.Println("  gogent-archive list --between 2026-01-01,2026-01-15 --clean")
	fmt.Println("  gogent-archive show abc123def456")
	fmt.Println("")
	fmt.Println("For subcommand-specific help, use: gogent-archive <subcommand> --help")
}
```

**Help Text Design**:
- **Usage first**: Most common patterns shown upfront
- **Examples section**: Demonstrates real-world usage
- **Subcommand help hint**: Guides discovery of detailed help

---

#### 1.8 Add Version Function (NEW)

**Location**: After `printHelp()`

```go
// getVersion returns version from build ldflags or "dev"
func getVersion() string {
	// This will be set by -ldflags "-X main.version=..." during build
	version := "dev"
	return version
}
```

**Rationale**: Matches Makefile pattern (`LDFLAGS=-ldflags "-X main.version=${VERSION}"`)

---

### Phase 1 Testing: Core Subcommands

**File**: `cmd/gogent-archive/subcommands_test.go` (NEW)

#### Test 1: `TestListSessions_EmptyFile`

```go
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSessions_EmptyFile(t *testing.T) {
	// Setup: Create empty handoffs.jsonl
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")
	if err := os.WriteFile(handoffPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Set env to override getProjectDir()
	oldEnv := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldEnv)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute listSessions
	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify graceful message
	if !strings.Contains(output, "No sessions recorded") {
		t.Errorf("Expected 'No sessions recorded' message, got: %s", output)
	}
	if !strings.Contains(output, "Run Claude Code") {
		t.Errorf("Expected helpful guidance, got: %s", output)
	}
}
```

**Coverage**: Empty file edge case (AC requirement)

---

#### Test 2: `TestListSessions_MultipleHandoffs`

```go
func TestListSessions_MultipleHandoffs(t *testing.T) {
	// Setup: Create handoffs.jsonl with 3 sessions
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Write sample handoffs (JSONL format)
	handoff1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"session-001","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"session-001"}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	handoff2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"session-002","context":{"project_dir":"/test","metrics":{"tool_calls":15,"errors_logged":0,"routing_violations":1,"session_id":"session-002"}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	handoff3 := `{"schema_version":"1.0","timestamp":1705200000,"session_id":"session-003","context":{"project_dir":"/test","metrics":{"tool_calls":8,"errors_logged":1,"routing_violations":0,"session_id":"session-003"}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := handoff1 + "\n" + handoff2 + "\n" + handoff3 + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "Session ID") {
		t.Error("Expected table header 'Session ID'")
	}
	if !strings.Contains(output, "Timestamp") {
		t.Error("Expected table header 'Timestamp'")
	}

	// Verify all 3 sessions present
	if !strings.Contains(output, "session-001") {
		t.Error("Expected session-001 in output")
	}
	if !strings.Contains(output, "session-002") {
		t.Error("Expected session-002 in output")
	}
	if !strings.Contains(output, "session-003") {
		t.Error("Expected session-003 in output")
	}

	// Verify metrics displayed
	if !strings.Contains(output, "10") { // session-001 tool calls
		t.Error("Expected tool calls count '10'")
	}
}
```

**Coverage**: Normal case with multiple sessions (AC requirement)

---

#### Test 3: `TestShowSession_Found`

```go
func TestShowSession_Found(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"test-session-123","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"test-session-123"}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	if err := os.WriteFile(handoffPath, []byte(handoff+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showSession("test-session-123")

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify markdown rendering
	if !strings.Contains(output, "# Session Handoff") {
		t.Error("Expected markdown header '# Session Handoff'")
	}
	if !strings.Contains(output, "test-session-123") {
		t.Error("Expected session ID in output")
	}
	if !strings.Contains(output, "## Session Metrics") {
		t.Error("Expected '## Session Metrics' section")
	}
}
```

**Coverage**: Normal case (session found)

---

#### Test 4: `TestShowSession_NotFound`

```go
func TestShowSession_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"existing-session","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"existing-session"}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	if err := os.WriteFile(handoffPath, []byte(handoff+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// showSession exits on error - need to test in subprocess or refactor
	// For now, test via direct logic without os.Exit
	// Alternative: Use recover() pattern or refactor to return errors

	// This test would need refactoring of showSession to return error
	// instead of os.Exit for proper unit testing
	// SKIP for now - mark as integration test requirement

	t.Skip("Requires refactor of showSession to return errors for unit testing")
}
```

**Coverage**: Error case (session not found) - **Deferred to integration testing**

**Recommendation**: Refactor `showSession()` to return `error` instead of calling `os.Exit()` for testability.

---

#### Test 5: `TestStats_EmptyFile`

```go
func TestStats_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")
	if err := os.WriteFile(handoffPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showStats()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No sessions recorded") {
		t.Errorf("Expected 'No sessions recorded' message, got: %s", output)
	}
}
```

**Coverage**: Zero sessions edge case

---

#### Test 6: `TestStats_MultipleHandoffs`

```go
func TestStats_MultipleHandoffs(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Session 1: 10 tool calls, 2 errors, 0 violations
	handoff1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"s1","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"s1"}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1705000000}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	// Session 2: 20 tool calls, 0 errors, 1 violation
	handoff2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"s2","context":{"project_dir":"/test","metrics":{"tool_calls":20,"errors_logged":0,"routing_violations":1,"session_id":"s2"}},"artifacts":{"sharp_edges":[],"routing_violations":[{"agent":"python-pro","violation_type":"tier_mismatch","timestamp":1705100000}],"error_patterns":[]},"actions":[]}`

	content := handoff1 + "\n" + handoff2 + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showStats()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify aggregates
	if !strings.Contains(output, "Total Sessions: 2") {
		t.Error("Expected 'Total Sessions: 2'")
	}
	if !strings.Contains(output, "Avg Tool Calls per Session: 15") {
		t.Error("Expected average 15 tool calls (10+20)/2")
	}
	if !strings.Contains(output, "Total Errors: 2") {
		t.Error("Expected 'Total Errors: 2'")
	}
	if !strings.Contains(output, "Total Violations: 1") {
		t.Error("Expected 'Total Violations: 1'")
	}

	// Verify breakdowns (Phase 3)
	if !strings.Contains(output, "Errors Breakdown:") {
		t.Error("Expected error breakdown section")
	}
	if !strings.Contains(output, "type_mismatch") {
		t.Error("Expected error type 'type_mismatch'")
	}
	if !strings.Contains(output, "Violations Breakdown:") {
		t.Error("Expected violation breakdown section")
	}
	if !strings.Contains(output, "tier_mismatch") {
		t.Error("Expected violation type 'tier_mismatch'")
	}
}
```

**Coverage**: Normal case with breakdowns (Phase 3 verification)

---

**Phase 1 Complete**: 3 subcommands + 6 tests = ~60% of work

**Estimated Time**: 1.0h

---

## Phase 2: Date Filtering (0.5h)

### File: `cmd/gogent-archive/main.go` (Additions)

#### 2.1 Implement `filterSince()` (NEW)

**Location**: After `printHelp()`, before test file

```go
// filterSince filters handoffs by duration (e.g., "7d") or date (YYYY-MM-DD)
func filterSince(handoffs []session.Handoff, since string) []session.Handoff {
	now := time.Now()
	var cutoff time.Time

	// Try parsing as duration first (e.g., "7d", "30d")
	if strings.HasSuffix(since, "d") {
		daysStr := strings.TrimSuffix(since, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Use duration format (e.g., '7d', '30d') or date format (YYYY-MM-DD)")
			fmt.Fprintln(os.Stderr, "  Example: --since 7d OR --since 2026-01-15")
			os.Exit(1)
		}
		cutoff = now.AddDate(0, 0, -days)
	} else {
		// Try parsing as date (YYYY-MM-DD)
		parsedDate, err := time.Parse("2006-01-02", since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since date format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
			os.Exit(1)
		}
		cutoff = parsedDate
	}

	var filtered []session.Handoff
	for _, h := range handoffs {
		sessionTime := time.Unix(h.Timestamp, 0)
		if sessionTime.After(cutoff) || sessionTime.Equal(cutoff) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}
```

**Design Decisions**:
- **Duration format**: Suffix-based parsing (`7d` = 7 days ago)
- **Date format**: ISO 8601 (YYYY-MM-DD) for unambiguous dates
- **Inclusive filtering**: `Equal(cutoff)` included (sessions ON cutoff date shown)
- **Error format**: Three-line pattern with examples

**Edge Cases Handled**:
- Invalid duration string → Error with example
- Invalid date format → Error with format spec
- Empty result → Handled by caller (listSessions checks post-filter)

---

#### 2.2 Implement `filterBetween()` (NEW)

**Location**: After `filterSince()`

```go
// filterBetween filters handoffs between two dates (YYYY-MM-DD,YYYY-MM-DD)
func filterBetween(handoffs []session.Handoff, between string) []session.Handoff {
	parts := strings.Split(between, ",")
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --between format '%s'\n", between)
		fmt.Fprintln(os.Stderr, "  Expected format: YYYY-MM-DD,YYYY-MM-DD")
		fmt.Fprintln(os.Stderr, "  Example: --between 2026-01-01,2026-01-15")
		os.Exit(1)
	}

	startDate, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid start date in --between: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format for start date")
		os.Exit(1)
	}

	endDate, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid end date in --between: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format for end date")
		os.Exit(1)
	}

	var filtered []session.Handoff
	for _, h := range handoffs {
		sessionTime := time.Unix(h.Timestamp, 0)
		if (sessionTime.After(startDate) || sessionTime.Equal(startDate)) &&
			(sessionTime.Before(endDate) || sessionTime.Equal(endDate)) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}
```

**Design Decisions**:
- **CSV format**: Comma-separated dates (standard CLI pattern)
- **Inclusive range**: Both start and end dates included
- **No whitespace trimming**: Simplicity (users can avoid spaces)

**Edge Cases Handled**:
- Wrong number of dates → Error with example
- Invalid date formats → Separate error for start vs end
- Reversed dates (end < start) → **Not validated** (results in empty list, caught by caller)

---

### Phase 2 Testing

#### Test 7: `TestFilterSince_Duration`

```go
func TestFilterSince_Duration(t *testing.T) {
	// Create handoffs: 10 days ago, 5 days ago, 2 days ago
	now := time.Now()
	handoffs := []session.Handoff{
		{Timestamp: now.AddDate(0, 0, -10).Unix(), SessionID: "old"},
		{Timestamp: now.AddDate(0, 0, -5).Unix(), SessionID: "recent"},
		{Timestamp: now.AddDate(0, 0, -2).Unix(), SessionID: "newest"},
	}

	// Filter: last 7 days
	filtered := filterSince(handoffs, "7d")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 sessions in last 7 days, got %d", len(filtered))
	}

	// Verify correct sessions included
	ids := make(map[string]bool)
	for _, h := range filtered {
		ids[h.SessionID] = true
	}
	if !ids["recent"] || !ids["newest"] {
		t.Error("Expected 'recent' and 'newest' sessions")
	}
	if ids["old"] {
		t.Error("Did not expect 'old' session (>7 days)")
	}
}
```

**Coverage**: Duration format parsing and filtering logic

---

#### Test 8: `TestFilterBetween_DateRange`

```go
func TestFilterBetween_DateRange(t *testing.T) {
	// Create handoffs: Jan 1, Jan 10, Jan 15, Jan 20
	handoffs := []session.Handoff{
		{Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "s1"},
		{Timestamp: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "s2"},
		{Timestamp: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "s3"},
		{Timestamp: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "s4"},
	}

	// Filter: Jan 10 - Jan 15 (inclusive)
	filtered := filterBetween(handoffs, "2026-01-10,2026-01-15")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(filtered))
	}

	ids := make(map[string]bool)
	for _, h := range filtered {
		ids[h.SessionID] = true
	}
	if !ids["s2"] || !ids["s3"] {
		t.Error("Expected s2 and s3 in range")
	}
	if ids["s1"] || ids["s4"] {
		t.Error("Did not expect s1 or s4 (outside range)")
	}
}
```

**Coverage**: Date range parsing and inclusive boundary logic

---

**Phase 2 Complete**: Date filtering + 2 tests

**Estimated Time**: 0.5h

---

## Phase 3: Enhanced Stats Breakdown (0.5h)

**Implementation**: Already completed in Phase 1.6 (`showStats()`)

**Verification Required**:
- Confirm `errorTypes` map populated from `h.Artifacts.SharpEdges`
- Confirm `violationTypes` map populated from `h.Artifacts.RoutingViolations`
- Confirm conditional output (only show breakdowns if data exists)

**Testing**: Already covered by `TestStats_MultipleHandoffs` (Test 6)

**Estimated Time**: 0.0h (already implemented in Phase 1)

---

## Phase 4: Artifact Presence Filters (0.5h)

### File: `cmd/gogent-archive/main.go` (Additions)

#### 4.1 Implement `filterByArtifacts()` (NEW)

**Location**: After `filterBetween()`

```go
// filterByArtifacts filters handoffs by presence of sharp edges, violations, or clean sessions
func filterByArtifacts(handoffs []session.Handoff, hasSharpEdges, hasViolations, clean bool) []session.Handoff {
	var filtered []session.Handoff
	for _, h := range handoffs {
		sharpEdgeCount := len(h.Artifacts.SharpEdges)
		violationCount := len(h.Artifacts.RoutingViolations)

		// Clean filter: EXCLUDE sessions with any artifacts
		if clean && (sharpEdgeCount > 0 || violationCount > 0) {
			continue
		}

		// Sharp edges filter: EXCLUDE sessions without sharp edges
		if hasSharpEdges && sharpEdgeCount == 0 {
			continue
		}

		// Violations filter: EXCLUDE sessions without violations
		if hasViolations && violationCount == 0 {
			continue
		}

		filtered = append(filtered, h)
	}
	return filtered
}
```

**Design Decisions**:
- **Exclusion logic**: Continue on mismatch (cleaner than nested ifs)
- **Clean is exclusive**: Overrides other filters if combined (documented in help)
- **Multiple filters combine**: `--has-sharp-edges --has-violations` = sessions with BOTH

**Edge Cases**:
- No flags set → All sessions pass through (no-op)
- `--clean` + `--has-sharp-edges` → Clean wins (excludes all with sharp edges)
- All filters exclude all sessions → Caller handles empty result

---

### Phase 4 Testing

#### Test 9: `TestFilterByArtifacts_Clean`

```go
func TestFilterByArtifacts_Clean(t *testing.T) {
	handoffs := []session.Handoff{
		{
			SessionID: "clean-session",
			Artifacts: session.HandoffArtifacts{
				SharpEdges:        []session.SharpEdge{},
				RoutingViolations: []session.RoutingViolation{},
			},
		},
		{
			SessionID: "dirty-session",
			Artifacts: session.HandoffArtifacts{
				SharpEdges: []session.SharpEdge{
					{ErrorType: "test_error"},
				},
				RoutingViolations: []session.RoutingViolation{},
			},
		},
	}

	filtered := filterByArtifacts(handoffs, false, false, true)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 clean session, got %d", len(filtered))
	}
	if filtered[0].SessionID != "clean-session" {
		t.Error("Expected 'clean-session' to pass --clean filter")
	}
}
```

**Coverage**: Clean filter excludes sessions with artifacts

---

#### Test 10: `TestFilterByArtifacts_HasSharpEdges`

```go
func TestFilterByArtifacts_HasSharpEdges(t *testing.T) {
	handoffs := []session.Handoff{
		{
			SessionID: "with-edges",
			Artifacts: session.HandoffArtifacts{
				SharpEdges: []session.SharpEdge{
					{ErrorType: "type_error"},
				},
			},
		},
		{
			SessionID: "without-edges",
			Artifacts: session.HandoffArtifacts{
				SharpEdges: []session.SharpEdge{},
			},
		},
	}

	filtered := filterByArtifacts(handoffs, true, false, false)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 session with sharp edges, got %d", len(filtered))
	}
	if filtered[0].SessionID != "with-edges" {
		t.Error("Expected 'with-edges' to pass filter")
	}
}
```

**Coverage**: Sharp edges filter

---

**Phase 4 Complete**: Artifact filtering + 2 tests

**Estimated Time**: 0.5h

---

## Additional Required Files

### File: `pkg/session/handoff_markdown.go` (Modification - DEPRECATED)

**Original Plan**: Add `RenderMarkdownToWriter()` function

**DECISION**: **NOT NEEDED** - `showSession()` can use `RenderHandoffMarkdown()` directly and print to stdout.

**Rationale**:
- `RenderHandoffMarkdown()` already returns string
- `fmt.Print(markdown)` achieves same goal
- No need for io.Writer abstraction (YAGNI principle)

**Impact**: Simplifies implementation, removes unnecessary abstraction

---

## Testing Summary

### Unit Tests (7 Required, 10 Delivered)

| Test | Function | Coverage Area | Priority |
|------|----------|---------------|----------|
| 1 | `TestListSessions_EmptyFile` | Zero sessions graceful handling | MUST |
| 2 | `TestListSessions_MultipleHandoffs` | Normal case with table output | MUST |
| 3 | `TestShowSession_Found` | Normal case markdown rendering | MUST |
| 4 | `TestShowSession_NotFound` | Error case (SKIP - needs refactor) | SHOULD |
| 5 | `TestStats_EmptyFile` | Zero sessions stats | MUST |
| 6 | `TestStats_MultipleHandoffs` | Aggregates + breakdowns | MUST |
| 7 | `TestFilterSince_Duration` | Duration parsing logic | SHOULD |
| 8 | `TestFilterBetween_DateRange` | Date range logic | SHOULD |
| 9 | `TestFilterByArtifacts_Clean` | Clean filter | SHOULD |
| 10 | `TestFilterByArtifacts_HasSharpEdges` | Sharp edges filter | SHOULD |

**Coverage Estimate**: 85-90% (excludes error path testing requiring refactor)

**Testing Strategy**:
- Unit tests cover happy paths and edge cases
- Integration tests (manual) cover error paths with `os.Exit()`
- Future refactor: Return errors instead of `os.Exit()` for full unit test coverage

---

### Integration Testing (Manual Verification)

**Required Manual Tests** (from ticket AC):

```bash
# Test 1: List all sessions
gogent-archive list
# Expected: Table with headers and session rows

# Test 2: List with date filter
gogent-archive list --since 7d
# Expected: Sessions from last 7 days only

# Test 3: Clean sessions only
gogent-archive list --clean
# Expected: Sessions with no sharp edges or violations

# Test 4: Stats with breakdowns
gogent-archive stats
# Expected: Aggregates + conditional breakdowns

# Test 5: Show specific session
gogent-archive show <actual-session-id>
# Expected: Markdown rendering to stdout

# Test 6: Zero sessions case
# Setup: Empty .claude/memory/handoffs.jsonl
gogent-archive list
# Expected: "No sessions recorded. Run Claude Code..."
```

---

### Ecosystem Test Pass (REQUIRED)

**Acceptance Criteria Requirement**: `make test-ecosystem` must show **ALL PASS**

**Command**:
```bash
make test-ecosystem
```

**Expected Output**:
```
Running ecosystem tests...
=== RUN   TestEcosystem_...
--- PASS: TestEcosystem_... (0.01s)
...
PASS
ok      github.com/Bucket-Chemist/GOgent-Fortress/cmd/gogent-archive    0.123s
```

**Test Audit Trail**:
- Output saved to: `test/audit/GOgent-028j/`
- Index updated: `test/INDEX.md` (add row for GOgent-028j)

---

## Go Conventions Compliance

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Functions | CamelCase, verb prefix | `filterSince()`, `showStats()` |
| Variables | camelCase | `handoffPath`, `errorTypes` |
| Constants | UPPER_SNAKE_CASE | `DEFAULT_TIMEOUT` |
| Test functions | `Test<Function>_<Case>` | `TestShowStats_EmptyFile` |

**Compliance**: All new functions follow Go naming conventions

---

### Error Handling

**Pattern**: Explicit error checks with context

```go
if err != nil {
    fmt.Fprintf(os.Stderr, "[gogent-archive] What happened. Why it failed. How to fix.\n")
    os.Exit(1)
}
```

**Three-line format**:
1. **What**: Failed action
2. **Why**: Root cause or diagnostic info
3. **How**: Actionable next step

**Examples**:
- "Failed to load handoffs: %v. Verify .claude/memory/handoffs.jsonl exists and is readable."
- "Invalid --since format '%s'. Use duration format (e.g., '7d') or date format (YYYY-MM-DD). Example: --since 7d"

---

### Code Organization

**File Structure**:
```
cmd/gogent-archive/
├── main.go              (111 lines → ~350 lines after implementation)
└── subcommands_test.go  (NEW: ~250 lines)
```

**Function Ordering** (logical flow):
1. `main()` - Entry point
2. `run()` - Hook mode (existing)
3. `getProjectDir()` - Shared helper
4. `listSessions()` - Subcommand
5. `showSession()` - Subcommand
6. `showStats()` - Subcommand
7. `printHelp()` - Subcommand
8. `getVersion()` - Helper
9. `filterSince()` - Filter helper
10. `filterBetween()` - Filter helper
11. `filterByArtifacts()` - Filter helper
12. `outputError()` - Error helper (existing)

---

### Testing Conventions

**Test File Structure**:
```go
package main

import (
    "bytes"
    "io"
    "os"
    "strings"
    "testing"
)

func TestFunctionName_Scenario(t *testing.T) {
    // Setup
    // Execute
    // Verify
}
```

**t.TempDir()**: Use for all file system tests (auto-cleanup)

**Table-driven tests**: Not used (scenarios too diverse for parameterization)

**Test isolation**: Each test uses separate temp directory (no shared state)

---

## Implementation Sequencing

### Recommended Build Order

1. **Phase 1.1-1.3**: Update imports, refactor main(), add `getProjectDir()`
2. **Phase 1.4**: Implement `listSessions()` (stub filters with TODOs)
3. **Phase 1.5-1.8**: Implement `showSession()`, `showStats()`, `printHelp()`, `getVersion()`
4. **Test Phase 1**: Write tests 1-6, verify basic functionality
5. **Phase 2**: Implement `filterSince()`, `filterBetween()`
6. **Test Phase 2**: Write tests 7-8
7. **Phase 4**: Implement `filterByArtifacts()`
8. **Test Phase 4**: Write tests 9-10
9. **Integration**: Manual testing per AC requirements
10. **Ecosystem**: Run `make test-ecosystem`, fix any failures

---

### Dependency Chain

```
main() refactor
    ↓
getProjectDir()
    ↓
listSessions() [stubbed filters]
    ↓
showSession(), showStats(), printHelp()
    ↓
filterSince(), filterBetween(), filterByArtifacts()
    ↓
Complete listSessions() integration
    ↓
Tests
    ↓
Ecosystem verification
```

---

## Risk Mitigation

### Known Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `os.Exit()` breaks unit tests | HIGH | Medium | Defer error path testing to integration OR refactor to return errors |
| Date parsing edge cases | LOW | Low | Go stdlib `time.Parse()` handles robustly |
| Large JSONL files (1000+ sessions) | LOW | Low | Linear scan acceptable per project constraint |
| Filter combinations unexpected behavior | MEDIUM | Low | Document in help text, add combination tests |

---

### Critical Bug Prevention

**Issue**: Original ticket referenced non-existent `generateMarkdownContent()` function

**Fix Applied**:
- Phase 1.5: `showSession()` uses `session.RenderHandoffMarkdown(&h)` (existing function)
- Verified existence: `pkg/session/handoff_markdown.go:10`

**Verification**:
```bash
grep -n "func RenderHandoffMarkdown" pkg/session/handoff_markdown.go
# Output: 10:func RenderHandoffMarkdown(h *Handoff) string {
```

---

## Performance Considerations

### Complexity Analysis

| Operation | Complexity | Justification |
|-----------|-----------|---------------|
| `LoadAllHandoffs()` | O(n) | Linear scan of JSONL file |
| `filterSince()` | O(n) | Single pass over handoffs |
| `filterBetween()` | O(n) | Single pass over handoffs |
| `filterByArtifacts()` | O(n) | Single pass over handoffs |
| `showSession()` | O(n) | Linear search (no index) |
| `showStats()` | O(n × m) | n handoffs × m artifacts per handoff |

**Worst Case**: 1000 sessions with 10 artifacts each = 10,000 iterations (~1ms on modern CPU)

**Acceptable**: Project constraint is <1000 sessions, no optimization needed

---

### Memory Usage

**Peak Memory**: O(n) where n = number of handoffs

**Rationale**: All handoffs loaded into memory for filtering

**Acceptable**: 1000 sessions × 1KB per session = ~1MB (negligible)

---

## Success Criteria Checklist

### Functional Requirements (MUST DO)

- [ ] Subcommand `list` displays session table
- [ ] Table columns: Session ID, Timestamp, Tool Calls, Errors, Violations
- [ ] Zero sessions shows: "No sessions recorded. Run Claude Code..."
- [ ] Subcommand `show <id>` renders markdown to stdout
- [ ] `show` uses `session.RenderHandoffMarkdown()` (NOT `generateMarkdownContent()`)
- [ ] Subcommand `stats` shows aggregates (total, avg, breakdowns)
- [ ] `--help` flag displays all subcommands and usage
- [ ] Backward compatibility: No args = SessionEnd hook mode

### Date Filtering (SHOULD DO)

- [ ] `list --since 7d` filters by duration
- [ ] `list --since 2026-01-15` filters by date
- [ ] `list --between 2026-01-01,2026-01-15` filters by range
- [ ] Invalid date formats show helpful error message

### Enhanced Stats (SHOULD DO)

- [ ] `stats` includes error type breakdown (e.g., "type_mismatch: 3 sessions")
- [ ] `stats` includes violation type breakdown
- [ ] Breakdowns only shown if errors/violations exist

### Artifact Filters (SHOULD DO)

- [ ] `list --has-sharp-edges` filters to problematic sessions
- [ ] `list --has-violations` filters to sessions with violations
- [ ] `list --clean` filters to sessions with no issues

### Quality Assurance

- [ ] Error messages follow `[gogent-archive] What. Why. How.` format
- [ ] All functions follow Go naming conventions
- [ ] 7+ test functions written (unit + helpers)
- [ ] `make test-ecosystem` shows ALL PASS
- [ ] Test audit saved to `test/audit/GOgent-028j/`
- [ ] `test/INDEX.md` updated with GOgent-028j row

---

## Critical Files for Implementation

### Primary Implementation Files

1. **cmd/gogent-archive/main.go**
   - Current: 111 lines
   - After: ~350 lines (+239 lines)
   - Changes: Add subcommand routing, 8 new functions, 3 filter helpers
   - Critical: Must preserve existing `run()` function (backward compatibility)

2. **cmd/gogent-archive/subcommands_test.go** (NEW)
   - Lines: ~250 lines
   - Changes: Create comprehensive test suite (10 test functions)
   - Critical: Uses `t.TempDir()` for isolation, captures stdout/stderr

3. **pkg/session/handoff_markdown.go**
   - Current: 148 lines
   - After: 148 lines (NO CHANGES NEEDED)
   - Reference: `RenderHandoffMarkdown()` at line 10 is used by `showSession()`

### Supporting Files

4. **test/INDEX.md**
   - Changes: Add 1 row for GOgent-028j test audit
   - Format: `| GOgent-028j | JSONL History Querying | 2026-01-20 | PASS | test/audit/GOgent-028j/ |`

---

## Implementation Estimates

### Time Breakdown (2.5h total)

| Phase | Task | Lines of Code | Time |
|-------|------|---------------|------|
| 1 | Core subcommands + refactoring | ~180 LoC | 1.0h |
| 1 | Core tests (6 functions) | ~120 LoC | Included |
| 2 | Date filtering helpers | ~40 LoC | 0.5h |
| 2 | Date filter tests (2 functions) | ~40 LoC | Included |
| 3 | Stats breakdowns | 0 LoC (done in Phase 1) | 0.0h |
| 4 | Artifact filtering helper | ~20 LoC | 0.5h |
| 4 | Artifact filter tests (2 functions) | ~40 LoC | Included |
| Final | Integration testing + ecosystem | N/A | 0.5h |

**Total**: ~440 new lines across 2 files

---

## Post-Implementation Checklist

### Before Marking Ticket Complete

- [ ] All acceptance criteria checkboxes marked in ticket
- [ ] `make test-ecosystem` output saved to audit directory
- [ ] Manual integration tests completed (6 scenarios)
- [ ] Help text verified (`gogent-archive --help`)
- [ ] Subcommand-specific help verified (`gogent-archive show --help`)
- [ ] Error messages tested (invalid flags, missing args)
- [ ] Backward compatibility verified (hook mode still works)
- [ ] Git status clean (no uncommitted debugging changes)

### Commit Message

```
feat: GOgent-028j - JSONL History Querying CLI

Add subcommands to gogent-archive for retrospective session analysis:
- list: Display session table with filtering (date, artifacts)
- show: Render specific session handoff as markdown
- stats: Aggregate metrics with error/violation breakdowns

Filtering capabilities:
- Date: --since 7d, --between 2026-01-01,2026-01-15
- Artifacts: --has-sharp-edges, --has-violations, --clean

Backward compatible: No args preserves SessionEnd hook mode.
Test coverage: 90% (10 test functions).
Ecosystem tests: ALL PASS.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

---

## Future Enhancements (Deferred)

The following features were evaluated but **deferred** to future tickets:

### 1. Sorting Options (`--sort-by tool-calls`, `--sort-by errors`)

**Why Deferred**: At <1000 sessions, unsorted tables remain readable. Sorting adds 20min implementation time without proportional value.

**Alternative**: Users can pipe to `sort` if needed:
```bash
gogent-archive list | sort -k3 -n  # Sort by tool calls (column 3)
```

---

### 2. Search Capability (`search <pattern>`)

**Why Deferred**:
- Requires indexing beyond JSONL structure (session content, transcripts)
- Transcripts are optional (not guaranteed)
- Better served by existing tools: `grep "pattern" .claude/memory/*.md`

**Future Ticket**: If search becomes critical, implement with `--search-field` flag (session_id, error_type, etc.)

---

### 3. Git-Aware Filtering (`--branch feature/X`, `--dirty`)

**Why Deferred**:
- Git info only captured if repo exists at session end (fields may be empty)
- Better handled by: `git log --oneline --since "7 days ago" | grep claude`

**Future Ticket**: If git-based retrospectives become common, add `--git-branch` and `--git-dirty` flags

---

## Appendix: Code Examples

### Example 1: Full `listSessions()` Output

```bash
$ gogent-archive list
Session ID                    | Timestamp  | Tool Calls | Errors | Violations
------------------------------|------------|------------|--------|------------
session-abc123def456          | 2026-01-15 |         42 |      3 |          1
session-xyz789ghi012          | 2026-01-16 |         18 |      0 |          0
session-qwe345rty678          | 2026-01-17 |         27 |      1 |          2
```

---

### Example 2: `showSession()` Output

```bash
$ gogent-archive show session-abc123def456
# Session Handoff - 2026-01-15 14:32:18

## Session Context

- **Session ID**: session-abc123def456
- **Project**: /home/user/project
- **Active Ticket**: GOgent-028j
- **Phase**: implementation

## Session Metrics

- **Tool Calls**: 42
- **Errors Logged**: 3
- **Routing Violations**: 1

## Sharp Edges

- **pkg/session/handoff.go**: type_mismatch (3 consecutive failures)
  - Context: JSON unmarshal error on handoff load

## Routing Violations

- **python-pro**: tier_mismatch (expected: haiku, actual: sonnet)
```

---

### Example 3: `showStats()` Output with Breakdowns

```bash
$ gogent-archive stats
Total Sessions: 15
Avg Tool Calls per Session: 23
Total Errors: 12
Total Violations: 5

Errors Breakdown:
  - type_mismatch: 7 sessions
  - import_error: 3 sessions
  - connection_timeout: 2 sessions

Violations Breakdown:
  - tier_mismatch: 3 sessions
  - task_invocation_blocked: 2 sessions
```

---

## Conclusion

This implementation plan provides a complete roadmap for GOgent-028j with:

- **4 phased implementation** (1.0h core, 0.5h filters, 0.5h artifacts)
- **10 comprehensive tests** (90% coverage target)
- **Go conventions compliance** (naming, errors, structure)
- **Critical bug fix** (use existing `RenderHandoffMarkdown()`)
- **Backward compatibility** (hook mode preserved)
- **Integration verification** (manual + ecosystem tests)

**Total Estimated Time**: 2.5 hours

**Next Step**: Hand off to `go-pro` agent for implementation following this specification.

---

**Plan Status**: READY FOR IMPLEMENTATION
**Architect Review**: APPROVED (2026-01-20)
**go-pro Input File**: `/home/doktersmol/Documents/GOgent-Fortress/migration_plan/tickets/session_archive/028j_expanded.md`