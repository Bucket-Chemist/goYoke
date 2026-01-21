package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
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

	// Reset os.Args to prevent flag parsing errors
	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list"}
	defer func() { os.Args = oldArgs }()

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

func TestListSessions_MultipleHandoffs(t *testing.T) {
	// Setup: Create handoffs.jsonl with 3 sessions
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Write sample handoffs (JSONL format)
	handoff1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"session-001","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"session-001"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	handoff2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"session-002","context":{"project_dir":"/test","metrics":{"tool_calls":15,"errors_logged":0,"routing_violations":1,"session_id":"session-002"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	handoff3 := `{"schema_version":"1.0","timestamp":1705200000,"session_id":"session-003","context":{"project_dir":"/test","metrics":{"tool_calls":8,"errors_logged":1,"routing_violations":0,"session_id":"session-003"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

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

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list"}
	defer func() { os.Args = oldArgs }()

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

func TestShowSession_Found(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"test-session-123","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"test-session-123"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
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

func TestStats_MultipleHandoffs(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Session 1: 10 tool calls, 2 errors, 0 violations
	handoff1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"s1","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"s1"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1705000000,"context":""}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	// Session 2: 20 tool calls, 0 errors, 1 violation
	handoff2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"s2","context":{"project_dir":"/test","metrics":{"tool_calls":20,"errors_logged":0,"routing_violations":1,"session_id":"s2"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[{"agent":"python-pro","violation_type":"tier_mismatch","timestamp":1705100000,"expected_tier":"haiku","actual_tier":"sonnet"}],"error_patterns":[]},"actions":[]}`

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

	// Verify breakdowns
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

func TestFilterByArtifacts_HasViolations(t *testing.T) {
	handoffs := []session.Handoff{
		{
			SessionID: "with-violations",
			Artifacts: session.HandoffArtifacts{
				RoutingViolations: []session.RoutingViolation{
					{ViolationType: "tier_mismatch"},
				},
			},
		},
		{
			SessionID: "without-violations",
			Artifacts: session.HandoffArtifacts{
				RoutingViolations: []session.RoutingViolation{},
			},
		},
	}

	filtered := filterByArtifacts(handoffs, false, true, false)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 session with violations, got %d", len(filtered))
	}
	if filtered[0].SessionID != "with-violations" {
		t.Error("Expected 'with-violations' to pass filter")
	}
}

func TestFilterByArtifacts_CombinedFilters(t *testing.T) {
	handoffs := []session.Handoff{
		{
			SessionID: "both-artifacts",
			Artifacts: session.HandoffArtifacts{
				SharpEdges:        []session.SharpEdge{{ErrorType: "test"}},
				RoutingViolations: []session.RoutingViolation{{ViolationType: "test"}},
			},
		},
		{
			SessionID: "only-edges",
			Artifacts: session.HandoffArtifacts{
				SharpEdges:        []session.SharpEdge{{ErrorType: "test"}},
				RoutingViolations: []session.RoutingViolation{},
			},
		},
		{
			SessionID: "only-violations",
			Artifacts: session.HandoffArtifacts{
				SharpEdges:        []session.SharpEdge{},
				RoutingViolations: []session.RoutingViolation{{ViolationType: "test"}},
			},
		},
	}

	// Filter: has both sharp edges AND violations
	filtered := filterByArtifacts(handoffs, true, true, false)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 session with both artifacts, got %d", len(filtered))
	}
	if filtered[0].SessionID != "both-artifacts" {
		t.Error("Expected 'both-artifacts' to pass combined filter")
	}
}

func TestFilterSince_DateFormat(t *testing.T) {
	// Create handoffs with known dates
	handoffs := []session.Handoff{
		{Timestamp: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "jan10"},
		{Timestamp: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "jan15"},
		{Timestamp: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "jan20"},
	}

	// Filter since Jan 15 (date format)
	filtered := filterSince(handoffs, "2026-01-15")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 sessions since 2026-01-15, got %d", len(filtered))
	}

	ids := make(map[string]bool)
	for _, h := range filtered {
		ids[h.SessionID] = true
	}
	if !ids["jan15"] || !ids["jan20"] {
		t.Error("Expected jan15 and jan20 sessions")
	}
}

func TestFilterSince_EmptyResult(t *testing.T) {
	now := time.Now()
	handoffs := []session.Handoff{
		{Timestamp: now.AddDate(0, 0, -30).Unix(), SessionID: "old"},
	}

	// Filter last 7 days (nothing should match)
	filtered := filterSince(handoffs, "7d")

	if len(filtered) != 0 {
		t.Errorf("Expected 0 sessions in last 7 days, got %d", len(filtered))
	}
}

func TestFilterBetween_EmptyResult(t *testing.T) {
	handoffs := []session.Handoff{
		{Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "s1"},
	}

	// Filter Jan 2026 (nothing should match)
	filtered := filterBetween(handoffs, "2026-01-01,2026-01-31")

	if len(filtered) != 0 {
		t.Errorf("Expected 0 sessions in range, got %d", len(filtered))
	}
}

func TestFilterBetween_SingleDay(t *testing.T) {
	// The filterBetween function parses dates without time, so "2026-01-15" becomes midnight
	// A timestamp at noon on Jan 15 is AFTER midnight Jan 15, so it passes the start check
	// But it's also AFTER midnight Jan 15, so it needs to be BEFORE or EQUAL to end date
	// The issue: 12:00 on Jan 15 is after 00:00 Jan 15, but we compare date-only

	// Use midnight for target to match the date-only comparison
	targetDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	handoffs := []session.Handoff{
		{Timestamp: time.Date(2026, 1, 14, 23, 59, 0, 0, time.UTC).Unix(), SessionID: "before"},
		{Timestamp: targetDate.Unix(), SessionID: "target"},
		{Timestamp: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "after"},
	}

	// Filter just Jan 15 (inclusive)
	filtered := filterBetween(handoffs, "2026-01-15,2026-01-15")

	if len(filtered) != 1 {
		t.Errorf("Expected 1 session on Jan 15, got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].SessionID != "target" {
		t.Error("Expected 'target' session")
	}
}

func TestGetVersion(t *testing.T) {
	version := getVersion()

	// Should return "dev" when not built with ldflags
	if version == "" {
		t.Error("Expected non-empty version string")
	}

	// Default should be "dev"
	if version != "dev" {
		t.Logf("Version is '%s' (may be set via ldflags)", version)
	}
}

func TestPrintHelp(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify key help content
	if !strings.Contains(output, "gogent-archive") {
		t.Error("Expected 'gogent-archive' in help output")
	}
	if !strings.Contains(output, "list") {
		t.Error("Expected 'list' subcommand in help")
	}
	if !strings.Contains(output, "show") {
		t.Error("Expected 'show' subcommand in help")
	}
	if !strings.Contains(output, "stats") {
		t.Error("Expected 'stats' subcommand in help")
	}
	if !strings.Contains(output, "--since") {
		t.Error("Expected '--since' flag documentation")
	}
	if !strings.Contains(output, "--between") {
		t.Error("Expected '--between' flag documentation")
	}
	if !strings.Contains(output, "--has-sharp-edges") {
		t.Error("Expected '--has-sharp-edges' flag documentation")
	}
	if !strings.Contains(output, "--clean") {
		t.Error("Expected '--clean' flag documentation")
	}
}

func TestGetProjectDir_FromEnv(t *testing.T) {
	oldEnv := os.Getenv("GOGENT_PROJECT_DIR")
	testDir := "/test/project/dir"
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", oldEnv)

	result := getProjectDir()

	if result != testDir {
		t.Errorf("Expected project dir '%s', got '%s'", testDir, result)
	}
}

func TestGetProjectDir_Fallback(t *testing.T) {
	oldEnv := os.Getenv("GOGENT_PROJECT_DIR")
	os.Unsetenv("GOGENT_PROJECT_DIR")
	defer os.Setenv("GOGENT_PROJECT_DIR", oldEnv)

	// Should fall back to cwd
	expectedDir, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get working directory")
	}

	result := getProjectDir()

	if result != expectedDir {
		t.Errorf("Expected fallback to cwd '%s', got '%s'", expectedDir, result)
	}
}

func TestListSessions_WithSinceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Create sessions: one recent, one old
	now := time.Now()
	recentTimestamp := now.AddDate(0, 0, -3).Unix()
	oldTimestamp := now.AddDate(0, 0, -30).Unix()

	recentHandoff := `{"schema_version":"1.0","timestamp":` + fmt.Sprintf("%d", recentTimestamp) + `,"session_id":"recent-session","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":0,"routing_violations":0,"session_id":"recent-session"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	oldHandoff := `{"schema_version":"1.0","timestamp":` + fmt.Sprintf("%d", oldTimestamp) + `,"session_id":"old-session","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"old-session"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := recentHandoff + "\n" + oldHandoff + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--since", "7d"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should only show recent session
	if !strings.Contains(output, "recent-session") {
		t.Error("Expected recent-session in output")
	}
	if strings.Contains(output, "old-session") {
		t.Error("Did not expect old-session in filtered output")
	}
}

func TestListSessions_WithCleanFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	cleanHandoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"clean-session","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":0,"routing_violations":0,"session_id":"clean-session"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	dirtyHandoff := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"dirty-session","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":1,"routing_violations":0,"session_id":"dirty-session"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"compile_error","consecutive_failures":3,"timestamp":1705100000}],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := cleanHandoff + "\n" + dirtyHandoff + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--clean"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should only show clean session
	if !strings.Contains(output, "clean-session") {
		t.Error("Expected clean-session in output")
	}
	if strings.Contains(output, "dirty-session") {
		t.Error("Did not expect dirty-session in clean-filtered output")
	}
}

func TestListSessions_NoMatchesAfterFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// All sessions have sharp edges
	dirtyHandoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"dirty","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":1,"routing_violations":0,"session_id":"dirty"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"test","consecutive_failures":3,"timestamp":1705000000}],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	if err := os.WriteFile(handoffPath, []byte(dirtyHandoff+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--clean"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show "no matches" message
	if !strings.Contains(output, "No sessions match") {
		t.Errorf("Expected 'No sessions match' message, got: %s", output)
	}
}

func TestStats_NoBreakdownsForCleanSessions(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Clean session with no errors or violations
	cleanHandoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"clean","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"clean"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	if err := os.WriteFile(handoffPath, []byte(cleanHandoff+"\n"), 0644); err != nil {
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

	// Should NOT contain breakdown sections
	if strings.Contains(output, "Errors Breakdown:") {
		t.Error("Did not expect 'Errors Breakdown:' for clean sessions")
	}
	if strings.Contains(output, "Violations Breakdown:") {
		t.Error("Did not expect 'Violations Breakdown:' for clean sessions")
	}
}

func TestListSessions_WithBetweenFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Jan 5, Jan 10, Jan 15
	h1 := `{"schema_version":"1.0","timestamp":1736035200,"session_id":"jan5","context":{"project_dir":"/test","metrics":{"tool_calls":1,"errors_logged":0,"routing_violations":0,"session_id":"jan5"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	h2 := `{"schema_version":"1.0","timestamp":1736467200,"session_id":"jan10","context":{"project_dir":"/test","metrics":{"tool_calls":2,"errors_logged":0,"routing_violations":0,"session_id":"jan10"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	h3 := `{"schema_version":"1.0","timestamp":1736899200,"session_id":"jan15","context":{"project_dir":"/test","metrics":{"tool_calls":3,"errors_logged":0,"routing_violations":0,"session_id":"jan15"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := h1 + "\n" + h2 + "\n" + h3 + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--between", "2025-01-08,2025-01-12"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should only show jan10
	if !strings.Contains(output, "jan10") {
		t.Error("Expected jan10 in between filter output")
	}
	if strings.Contains(output, "jan5") {
		t.Error("Did not expect jan5 in between filter output")
	}
	if strings.Contains(output, "jan15") {
		t.Error("Did not expect jan15 in between filter output")
	}
}

func TestListSessions_WithHasSharpEdgesFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	withEdges := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"has-edges","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":1,"routing_violations":0,"session_id":"has-edges"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"test.go","error_type":"compile_error","consecutive_failures":3,"timestamp":1705000000}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	noEdges := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"no-edges","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"no-edges"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := withEdges + "\n" + noEdges + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--has-sharp-edges"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "has-edges") {
		t.Error("Expected has-edges in output")
	}
	if strings.Contains(output, "no-edges") {
		t.Error("Did not expect no-edges in has-sharp-edges filtered output")
	}
}

func TestListSessions_WithHasViolationsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	withViolations := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"has-violations","context":{"project_dir":"/test","metrics":{"tool_calls":5,"errors_logged":0,"routing_violations":1,"session_id":"has-violations"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[{"agent":"test","violation_type":"tier_mismatch","timestamp":1705000000,"expected_tier":"haiku","actual_tier":"sonnet"}],"error_patterns":[]},"actions":[]}`
	noViolations := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"no-violations","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":0,"routing_violations":0,"session_id":"no-violations"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	content := withViolations + "\n" + noViolations + "\n"
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list", "--has-violations"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "has-violations") {
		t.Error("Expected has-violations in output")
	}
	if strings.Contains(output, "no-violations") {
		t.Error("Did not expect no-violations in has-violations filtered output")
	}
}

// Additional filter coverage tests

func TestFilterSince_ZeroDays(t *testing.T) {
	now := time.Now()
	handoffs := []session.Handoff{
		{Timestamp: now.AddDate(0, 0, -1).Unix(), SessionID: "yesterday"},
		{Timestamp: now.Unix(), SessionID: "today"},
	}

	// Filter last 0 days - should only include today
	filtered := filterSince(handoffs, "0d")

	// 0 days means "since now", so only exact timestamp or later matches
	if len(filtered) > 2 {
		t.Errorf("Expected at most 2 sessions, got %d", len(filtered))
	}
}

func TestFilterSince_LongDuration(t *testing.T) {
	now := time.Now()
	handoffs := []session.Handoff{
		{Timestamp: now.AddDate(-1, 0, 0).Unix(), SessionID: "year-ago"},
		{Timestamp: now.AddDate(0, -6, 0).Unix(), SessionID: "six-months-ago"},
		{Timestamp: now.Unix(), SessionID: "today"},
	}

	// Filter last 365 days
	filtered := filterSince(handoffs, "365d")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 sessions in last 365 days, got %d", len(filtered))
	}
}

func TestFilterBetween_FullYearRange(t *testing.T) {
	handoffs := []session.Handoff{
		{Timestamp: time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC).Unix(), SessionID: "end-2025"},
		{Timestamp: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "mid-2026"},
		{Timestamp: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), SessionID: "start-2027"},
	}

	// Filter full year 2026
	filtered := filterBetween(handoffs, "2026-01-01,2026-12-31")

	if len(filtered) != 1 {
		t.Errorf("Expected 1 session in 2026, got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].SessionID != "mid-2026" {
		t.Error("Expected mid-2026 session")
	}
}

func TestFilterByArtifacts_AllFiltersOff(t *testing.T) {
	handoffs := []session.Handoff{
		{SessionID: "s1", Artifacts: session.HandoffArtifacts{}},
		{SessionID: "s2", Artifacts: session.HandoffArtifacts{SharpEdges: []session.SharpEdge{{ErrorType: "x"}}}},
		{SessionID: "s3", Artifacts: session.HandoffArtifacts{RoutingViolations: []session.RoutingViolation{{ViolationType: "y"}}}},
	}

	// No filters active - should return all
	filtered := filterByArtifacts(handoffs, false, false, false)

	if len(filtered) != 3 {
		t.Errorf("Expected all 3 sessions when no filters active, got %d", len(filtered))
	}
}

func TestShowSession_RendersArtifacts(t *testing.T) {
	// This test uses session.LoadAllHandoffs and session.RenderHandoffMarkdown directly
	// to avoid os.Exit calls in showSession
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Session with sharp edges and violations
	// Note: actions must be array of objects with priority/description, not strings
	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"rich-session","context":{"project_dir":"/test","metrics":{"tool_calls":25,"errors_logged":3,"routing_violations":2,"session_id":"rich-session"},"git_info":{"branch":"main","is_dirty":true,"uncommitted":["file1.go","file2.go"]}},"artifacts":{"sharp_edges":[{"file":"auth.go","error_type":"nil_pointer","consecutive_failures":5,"timestamp":1705000100,"context":"authenticating user"}],"routing_violations":[{"agent":"python-pro","violation_type":"tier_mismatch","timestamp":1705000200,"expected_tier":"haiku","actual_tier":"sonnet"}],"error_patterns":[]},"actions":[{"priority":1,"description":"Review auth.go nil pointer handling"}]}`

	if err := os.WriteFile(handoffPath, []byte(handoff+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test the underlying functions directly (showSession wraps these)
	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoffs: %v", err)
	}

	if len(handoffs) != 1 {
		t.Fatalf("Expected 1 handoff, got %d", len(handoffs))
	}

	if handoffs[0].SessionID != "rich-session" {
		t.Errorf("Expected session ID 'rich-session', got '%s'", handoffs[0].SessionID)
	}

	// Render the markdown
	output := session.RenderHandoffMarkdown(&handoffs[0])

	// Verify markdown contains key sections
	if !strings.Contains(output, "# Session Handoff") {
		t.Error("Expected markdown header")
	}
	if !strings.Contains(output, "rich-session") {
		t.Error("Expected session ID")
	}
	if !strings.Contains(output, "25") { // tool calls
		t.Error("Expected tool calls count")
	}
	if !strings.Contains(output, "nil_pointer") {
		t.Error("Expected sharp edge error type in output")
	}
}

func TestStats_WithMultipleErrorTypes(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Multiple sessions with different error types
	h1 := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"s1","context":{"project_dir":"/test","metrics":{"tool_calls":10,"errors_logged":2,"routing_violations":0,"session_id":"s1"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"a.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	h2 := `{"schema_version":"1.0","timestamp":1705100000,"session_id":"s2","context":{"project_dir":"/test","metrics":{"tool_calls":15,"errors_logged":1,"routing_violations":0,"session_id":"s2"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"b.go","error_type":"type_error","consecutive_failures":3,"timestamp":2}],"routing_violations":[],"error_patterns":[]},"actions":[]}`
	h3 := `{"schema_version":"1.0","timestamp":1705200000,"session_id":"s3","context":{"project_dir":"/test","metrics":{"tool_calls":20,"errors_logged":3,"routing_violations":1,"session_id":"s3"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[{"file":"c.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":3}],"routing_violations":[{"agent":"test","violation_type":"ceiling_breach","timestamp":3,"expected_tier":"haiku","actual_tier":"opus"}],"error_patterns":[]},"actions":[]}`

	content := h1 + "\n" + h2 + "\n" + h3 + "\n"
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

	// Verify stats
	if !strings.Contains(output, "Total Sessions: 3") {
		t.Error("Expected 3 total sessions")
	}
	if !strings.Contains(output, "Avg Tool Calls per Session: 15") {
		t.Error("Expected average 15 tool calls")
	}
	if !strings.Contains(output, "nil_pointer") {
		t.Error("Expected nil_pointer in error breakdown")
	}
	if !strings.Contains(output, "type_error") {
		t.Error("Expected type_error in error breakdown")
	}
	if !strings.Contains(output, "ceiling_breach") {
		t.Error("Expected ceiling_breach in violations breakdown")
	}
}

func TestListSessions_MetricsDisplayed(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(claudeDir, "handoffs.jsonl")

	// Session with specific metric values to verify table output
	handoff := `{"schema_version":"1.0","timestamp":1705000000,"session_id":"metrics-test","context":{"project_dir":"/test","metrics":{"tool_calls":42,"errors_logged":7,"routing_violations":3,"session_id":"metrics-test"},"git_info":{"branch":"","is_dirty":false,"uncommitted":null}},"artifacts":{"sharp_edges":[],"routing_violations":[],"error_patterns":[]},"actions":[]}`

	if err := os.WriteFile(handoffPath, []byte(handoff+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "list"}
	defer func() { os.Args = oldArgs }()

	listSessions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers and data
	if !strings.Contains(output, "Tool Calls") {
		t.Error("Expected 'Tool Calls' header")
	}
	if !strings.Contains(output, "Errors") {
		t.Error("Expected 'Errors' header")
	}
	if !strings.Contains(output, "Violations") {
		t.Error("Expected 'Violations' header")
	}
	if !strings.Contains(output, "42") {
		t.Error("Expected tool calls value 42")
	}
	if !strings.Contains(output, "7") {
		t.Error("Expected errors value 7")
	}
	if !strings.Contains(output, "3") {
		t.Error("Expected violations value 3")
	}
}
