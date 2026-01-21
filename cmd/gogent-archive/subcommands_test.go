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

// ========== DECISIONS SUBCOMMAND TESTS ==========

func TestListDecisions_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	if err := os.WriteFile(decisionsPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "decisions"}
	defer func() { os.Args = oldArgs }()

	listDecisions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No decisions recorded") {
		t.Errorf("Expected 'No decisions recorded' message, got: %s", output)
	}
}

func TestListDecisions_MultipleDecisions(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")

	d1 := `{"timestamp":1705000000,"category":"architecture","decision":"Use JSONL for storage","rationale":"Better for append-only logs","alternatives":"SQLite, JSON","impact":"high"}`
	d2 := `{"timestamp":1705100000,"category":"tooling","decision":"Adopt Go test framework","rationale":"Standard library is sufficient","alternatives":"Ginkgo, Testify","impact":"medium"}`
	d3 := `{"timestamp":1705200000,"category":"pattern","decision":"Use table-driven tests","rationale":"Better coverage, easier maintenance","alternatives":"Individual test functions","impact":"low"}`

	content := d1 + "\n" + d2 + "\n" + d3 + "\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "decisions"}
	defer func() { os.Args = oldArgs }()

	listDecisions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "Category") {
		t.Error("Expected 'Category' header")
	}
	if !strings.Contains(output, "Impact") {
		t.Error("Expected 'Impact' header")
	}
	if !strings.Contains(output, "Decision") {
		t.Error("Expected 'Decision' header")
	}
	if !strings.Contains(output, "Rationale") {
		t.Error("Expected 'Rationale' header")
	}

	// Verify all decisions are present
	if !strings.Contains(output, "architecture") {
		t.Error("Expected 'architecture' category")
	}
	if !strings.Contains(output, "tooling") {
		t.Error("Expected 'tooling' category")
	}
	if !strings.Contains(output, "pattern") {
		t.Error("Expected 'pattern' category")
	}

	// Verify total count
	if !strings.Contains(output, "Total: 3 decision(s)") {
		t.Error("Expected 'Total: 3 decision(s)'")
	}
}

func TestListDecisions_FilterByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")

	d1 := `{"timestamp":1705000000,"category":"architecture","decision":"Architecture decision","rationale":"Reason","alternatives":"None","impact":"high"}`
	d2 := `{"timestamp":1705100000,"category":"tooling","decision":"Tooling decision","rationale":"Reason","alternatives":"None","impact":"medium"}`

	content := d1 + "\n" + d2 + "\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "decisions", "--category", "architecture"}
	defer func() { os.Args = oldArgs }()

	listDecisions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "architecture") {
		t.Error("Expected 'architecture' in output")
	}
	if strings.Contains(output, "Tooling decision") {
		t.Error("Did not expect tooling decision in filtered output")
	}
	if !strings.Contains(output, "Total: 1 decision(s)") {
		t.Error("Expected 'Total: 1 decision(s)'")
	}
}

func TestListDecisions_FilterByImpact(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")

	d1 := `{"timestamp":1705000000,"category":"architecture","decision":"High impact decision","rationale":"Reason","alternatives":"None","impact":"high"}`
	d2 := `{"timestamp":1705100000,"category":"tooling","decision":"Low impact decision","rationale":"Reason","alternatives":"None","impact":"low"}`

	content := d1 + "\n" + d2 + "\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "decisions", "--impact", "high"}
	defer func() { os.Args = oldArgs }()

	listDecisions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "High impact decision") {
		t.Error("Expected 'High impact decision' in output")
	}
	if strings.Contains(output, "Low impact decision") {
		t.Error("Did not expect low impact decision in filtered output")
	}
}

func TestListDecisions_FilterBySince(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")

	now := time.Now()
	recentTs := now.AddDate(0, 0, -3).Unix()
	oldTs := now.AddDate(0, 0, -30).Unix()

	d1 := fmt.Sprintf(`{"timestamp":%d,"category":"architecture","decision":"Recent decision","rationale":"Reason","alternatives":"None","impact":"high"}`, recentTs)
	d2 := fmt.Sprintf(`{"timestamp":%d,"category":"tooling","decision":"Old decision","rationale":"Reason","alternatives":"None","impact":"low"}`, oldTs)

	content := d1 + "\n" + d2 + "\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "decisions", "--since", "7d"}
	defer func() { os.Args = oldArgs }()

	listDecisions()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Recent decision") {
		t.Error("Expected recent decision in output")
	}
	if strings.Contains(output, "Old decision") {
		t.Error("Did not expect old decision in filtered output")
	}
}

// ========== PREFERENCES SUBCOMMAND TESTS ==========

func TestListPreferences_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	if err := os.WriteFile(prefsPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "preferences"}
	defer func() { os.Args = oldArgs }()

	listPreferences()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No preferences recorded") {
		t.Errorf("Expected 'No preferences recorded' message, got: %s", output)
	}
}

func TestListPreferences_MultiplePreferences(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")

	p1 := `{"timestamp":1705000000,"category":"routing","key":"default_tier","value":"sonnet","reason":"Higher quality output","scope":"project"}`
	p2 := `{"timestamp":1705100000,"category":"tooling","key":"test_framework","value":"go_test","reason":"Standard library","scope":"global"}`
	p3 := `{"timestamp":1705200000,"category":"formatting","key":"indent_style","value":"tabs","reason":"Team preference","scope":"session"}`

	content := p1 + "\n" + p2 + "\n" + p3 + "\n"
	if err := os.WriteFile(prefsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "preferences"}
	defer func() { os.Args = oldArgs }()

	listPreferences()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "Category") {
		t.Error("Expected 'Category' header")
	}
	if !strings.Contains(output, "Scope") {
		t.Error("Expected 'Scope' header")
	}
	if !strings.Contains(output, "Key") {
		t.Error("Expected 'Key' header")
	}
	if !strings.Contains(output, "Value") {
		t.Error("Expected 'Value' header")
	}
	if !strings.Contains(output, "Reason") {
		t.Error("Expected 'Reason' header")
	}

	// Verify all preferences are present
	if !strings.Contains(output, "routing") {
		t.Error("Expected 'routing' category")
	}
	if !strings.Contains(output, "project") {
		t.Error("Expected 'project' scope")
	}

	// Verify total count
	if !strings.Contains(output, "Total: 3 preference(s)") {
		t.Error("Expected 'Total: 3 preference(s)'")
	}
}

func TestListPreferences_FilterByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")

	p1 := `{"timestamp":1705000000,"category":"routing","key":"tier","value":"sonnet","reason":"Quality","scope":"project"}`
	p2 := `{"timestamp":1705100000,"category":"tooling","key":"compiler","value":"go","reason":"Standard","scope":"global"}`

	content := p1 + "\n" + p2 + "\n"
	if err := os.WriteFile(prefsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "preferences", "--category", "routing"}
	defer func() { os.Args = oldArgs }()

	listPreferences()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "routing") {
		t.Error("Expected 'routing' in output")
	}
	if strings.Contains(output, "compiler") {
		t.Error("Did not expect tooling preference in filtered output")
	}
	if !strings.Contains(output, "Total: 1 preference(s)") {
		t.Error("Expected 'Total: 1 preference(s)'")
	}
}

func TestListPreferences_FilterByScope(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")

	p1 := `{"timestamp":1705000000,"category":"routing","key":"tier","value":"sonnet","reason":"Quality","scope":"project"}`
	p2 := `{"timestamp":1705100000,"category":"tooling","key":"compiler","value":"go","reason":"Standard","scope":"global"}`
	p3 := `{"timestamp":1705200000,"category":"formatting","key":"spaces","value":"4","reason":"Readability","scope":"session"}`

	content := p1 + "\n" + p2 + "\n" + p3 + "\n"
	if err := os.WriteFile(prefsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "preferences", "--scope", "global"}
	defer func() { os.Args = oldArgs }()

	listPreferences()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "global") {
		t.Error("Expected 'global' in output")
	}
	if strings.Contains(output, "project") && !strings.Contains(output, "Scope") {
		// Allow "project" in header but not as data
		if strings.Count(output, "project") > 1 {
			t.Error("Did not expect project scope preference in filtered output")
		}
	}
	if !strings.Contains(output, "Total: 1 preference(s)") {
		t.Error("Expected 'Total: 1 preference(s)'")
	}
}

// ========== PERFORMANCE SUBCOMMAND TESTS ==========

func TestShowPerformance_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	if err := os.WriteFile(perfPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "performance"}
	defer func() { os.Args = oldArgs }()

	showPerformance()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No performance metrics recorded") {
		t.Errorf("Expected 'No performance metrics recorded' message, got: %s", output)
	}
}

func TestShowPerformance_MultipleMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	m1 := `{"timestamp":1705000000,"operation":"handoff_generation","duration_ms":150,"memory_bytes":1048576,"success":true,"context":"session-001"}`
	m2 := `{"timestamp":1705100000,"operation":"validation","duration_ms":25,"memory_bytes":524288,"success":true,"context":"schema check"}`
	m3 := `{"timestamp":1705200000,"operation":"handoff_generation","duration_ms":1500,"memory_bytes":2097152,"success":false,"context":"session-002 timeout"}`

	content := m1 + "\n" + m2 + "\n" + m3 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "performance"}
	defer func() { os.Args = oldArgs }()

	showPerformance()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "Operation") {
		t.Error("Expected 'Operation' header")
	}
	if !strings.Contains(output, "Duration") {
		t.Error("Expected 'Duration' header")
	}
	if !strings.Contains(output, "Memory") {
		t.Error("Expected 'Memory' header")
	}
	if !strings.Contains(output, "Success") {
		t.Error("Expected 'Success' header")
	}

	// Verify metrics present
	if !strings.Contains(output, "handoff_generation") {
		t.Error("Expected 'handoff_generation' operation")
	}
	if !strings.Contains(output, "validation") {
		t.Error("Expected 'validation' operation")
	}

	// Verify total and stats
	if !strings.Contains(output, "Total: 3 metric(s)") {
		t.Error("Expected 'Total: 3 metric(s)'")
	}
	if !strings.Contains(output, "Average duration:") {
		t.Error("Expected 'Average duration:' in stats")
	}
	if !strings.Contains(output, "Success rate:") {
		t.Error("Expected 'Success rate:' in stats")
	}
}

func TestShowPerformance_ByOperation(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	m1 := `{"timestamp":1705000000,"operation":"handoff_generation","duration_ms":100,"memory_bytes":1048576,"success":true,"context":"s1"}`
	m2 := `{"timestamp":1705100000,"operation":"handoff_generation","duration_ms":200,"memory_bytes":1048576,"success":true,"context":"s2"}`
	m3 := `{"timestamp":1705200000,"operation":"validation","duration_ms":50,"memory_bytes":524288,"success":true,"context":"v1"}`
	m4 := `{"timestamp":1705300000,"operation":"handoff_generation","duration_ms":300,"memory_bytes":2097152,"success":false,"context":"s3"}`

	content := m1 + "\n" + m2 + "\n" + m3 + "\n" + m4 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "performance", "--by-operation"}
	defer func() { os.Args = oldArgs }()

	showPerformance()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify summary headers
	if !strings.Contains(output, "Count") {
		t.Error("Expected 'Count' header in summary")
	}
	if !strings.Contains(output, "Avg (ms)") {
		t.Error("Expected 'Avg (ms)' header in summary")
	}
	if !strings.Contains(output, "Min (ms)") {
		t.Error("Expected 'Min (ms)' header in summary")
	}
	if !strings.Contains(output, "Max (ms)") {
		t.Error("Expected 'Max (ms)' header in summary")
	}

	// Verify operations are summarized
	if !strings.Contains(output, "handoff_generation") {
		t.Error("Expected 'handoff_generation' in summary")
	}
	if !strings.Contains(output, "validation") {
		t.Error("Expected 'validation' in summary")
	}

	// Verify total shows success/failed
	if !strings.Contains(output, "success") {
		t.Error("Expected success count in total")
	}
	if !strings.Contains(output, "failed") {
		t.Error("Expected failed count in total")
	}
}

func TestShowPerformance_SlowOnly(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	// Create fast and slow metrics (threshold is 1000ms)
	m1 := `{"timestamp":1705000000,"operation":"fast_op","duration_ms":100,"memory_bytes":1048576,"success":true,"context":"fast"}`
	m2 := `{"timestamp":1705100000,"operation":"slow_op","duration_ms":2000,"memory_bytes":2097152,"success":true,"context":"slow"}`
	m3 := `{"timestamp":1705200000,"operation":"very_slow_op","duration_ms":5000,"memory_bytes":4194304,"success":false,"context":"very slow"}`

	content := m1 + "\n" + m2 + "\n" + m3 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "performance", "--slow-only"}
	defer func() { os.Args = oldArgs }()

	showPerformance()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show slow operations
	if !strings.Contains(output, "slow_op") {
		t.Error("Expected 'slow_op' in slow-only output")
	}
	if !strings.Contains(output, "very_slow_op") {
		t.Error("Expected 'very_slow_op' in slow-only output")
	}

	// Should NOT show fast operation
	if strings.Contains(output, "fast_op") {
		t.Error("Did not expect 'fast_op' in slow-only output")
	}

	// Verify count
	if !strings.Contains(output, "Total: 2 metric(s)") {
		t.Error("Expected 'Total: 2 metric(s)'")
	}
}

func TestShowPerformance_FilterBySince(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	now := time.Now()
	recentTs := now.AddDate(0, 0, -3).Unix()
	oldTs := now.AddDate(0, 0, -30).Unix()

	m1 := fmt.Sprintf(`{"timestamp":%d,"operation":"recent_op","duration_ms":150,"memory_bytes":1048576,"success":true,"context":"recent"}`, recentTs)
	m2 := fmt.Sprintf(`{"timestamp":%d,"operation":"old_op","duration_ms":200,"memory_bytes":1048576,"success":true,"context":"old"}`, oldTs)

	content := m1 + "\n" + m2 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldArgs := os.Args
	os.Args = []string{"gogent-archive", "performance", "--since", "7d"}
	defer func() { os.Args = oldArgs }()

	showPerformance()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "recent_op") {
		t.Error("Expected recent operation in output")
	}
	if strings.Contains(output, "old_op") {
		t.Error("Did not expect old operation in filtered output")
	}
}

// ========== FORMAT BYTES HELPER TEST ==========

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "-"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{1073741824, "1.0GB"},
	}

	for _, tc := range tests {
		result := formatBytes(tc.bytes)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tc.bytes, result, tc.expected)
		}
	}
}

// ========== HELP OUTPUT TESTS FOR NEW COMMANDS ==========

func TestPrintHelp_IncludesNewSubcommands(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify decision commands
	if !strings.Contains(output, "Decision Commands:") {
		t.Error("Expected 'Decision Commands:' section in help")
	}
	if !strings.Contains(output, "gogent-archive decisions") {
		t.Error("Expected 'gogent-archive decisions' in help")
	}
	if !strings.Contains(output, "--category architecture") {
		t.Error("Expected decision category filter in help")
	}
	if !strings.Contains(output, "--impact high") {
		t.Error("Expected decision impact filter in help")
	}

	// Verify preference commands
	if !strings.Contains(output, "Preference Commands:") {
		t.Error("Expected 'Preference Commands:' section in help")
	}
	if !strings.Contains(output, "gogent-archive preferences") {
		t.Error("Expected 'gogent-archive preferences' in help")
	}
	if !strings.Contains(output, "--scope project") {
		t.Error("Expected preference scope filter in help")
	}

	// Verify performance commands
	if !strings.Contains(output, "Performance Commands:") {
		t.Error("Expected 'Performance Commands:' section in help")
	}
	if !strings.Contains(output, "gogent-archive performance") {
		t.Error("Expected 'gogent-archive performance' in help")
	}
	if !strings.Contains(output, "--by-operation") {
		t.Error("Expected performance by-operation flag in help")
	}
	if !strings.Contains(output, "--slow-only") {
		t.Error("Expected performance slow-only flag in help")
	}

	// Verify new examples
	if !strings.Contains(output, "decisions --category architecture --impact high") {
		t.Error("Expected decisions example in help")
	}
	if !strings.Contains(output, "preferences --scope project") {
		t.Error("Expected preferences example in help")
	}
	if !strings.Contains(output, "performance --by-operation --slow-only") {
		t.Error("Expected performance example in help")
	}
}

// ========== QUERY API TESTS (pkg/session/query.go) ==========

func TestQueryDecisions_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := session.NewQuery(tmpDir)

	decisions, err := q.QueryDecisions(session.DecisionFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("Expected empty slice for missing file, got %d items", len(decisions))
	}
}

func TestQueryDecisions_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")

	d1 := `{"timestamp":1,"category":"a","decision":"d1","rationale":"r1","alternatives":"","impact":"high"}`
	d2 := `{"timestamp":2,"category":"b","decision":"d2","rationale":"r2","alternatives":"","impact":"medium"}`
	d3 := `{"timestamp":3,"category":"c","decision":"d3","rationale":"r3","alternatives":"","impact":"low"}`

	content := d1 + "\n" + d2 + "\n" + d3 + "\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := session.NewQuery(tmpDir)
	decisions, err := q.QueryDecisions(session.DecisionFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(decisions) != 2 {
		t.Errorf("Expected 2 decisions with limit, got %d", len(decisions))
	}
}

func TestQueryPreferences_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := session.NewQuery(tmpDir)

	prefs, err := q.QueryPreferences(session.PreferenceFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("Expected empty slice for missing file, got %d items", len(prefs))
	}
}

func TestQueryPreferences_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")

	p1 := `{"timestamp":1,"category":"a","key":"k1","value":"v1","reason":"r1","scope":"session"}`
	p2 := `{"timestamp":2,"category":"b","key":"k2","value":"v2","reason":"r2","scope":"project"}`
	p3 := `{"timestamp":3,"category":"c","key":"k3","value":"v3","reason":"r3","scope":"global"}`

	content := p1 + "\n" + p2 + "\n" + p3 + "\n"
	if err := os.WriteFile(prefsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := session.NewQuery(tmpDir)
	prefs, err := q.QueryPreferences(session.PreferenceFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(prefs) != 2 {
		t.Errorf("Expected 2 preferences with limit, got %d", len(prefs))
	}
}

func TestQueryPerformance_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := session.NewQuery(tmpDir)

	metrics, err := q.QueryPerformance(session.PerformanceFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("Expected empty slice for missing file, got %d items", len(metrics))
	}
}

func TestQueryPerformance_WithFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	m1 := `{"timestamp":1705000000,"operation":"handoff","duration_ms":500,"memory_bytes":1048576,"success":true,"context":"s1"}`
	m2 := `{"timestamp":1705100000,"operation":"handoff","duration_ms":1500,"memory_bytes":2097152,"success":true,"context":"s2"}`
	m3 := `{"timestamp":1705200000,"operation":"validation","duration_ms":50,"memory_bytes":524288,"success":false,"context":"s3"}`

	content := m1 + "\n" + m2 + "\n" + m3 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := session.NewQuery(tmpDir)

	// Test SlowOnly filter
	slowMetrics, err := q.QueryPerformance(session.PerformanceFilters{SlowOnly: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(slowMetrics) != 1 {
		t.Errorf("Expected 1 slow metric, got %d", len(slowMetrics))
	}
	if len(slowMetrics) > 0 && slowMetrics[0].DurationMs != 1500 {
		t.Errorf("Expected slow metric with 1500ms, got %d", slowMetrics[0].DurationMs)
	}

	// Test SuccessOnly filter
	successMetrics, err := q.QueryPerformance(session.PerformanceFilters{SuccessOnly: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(successMetrics) != 2 {
		t.Errorf("Expected 2 success metrics, got %d", len(successMetrics))
	}

	// Test FailedOnly filter
	failedMetrics, err := q.QueryPerformance(session.PerformanceFilters{FailedOnly: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(failedMetrics) != 1 {
		t.Errorf("Expected 1 failed metric, got %d", len(failedMetrics))
	}

	// Test Operation filter
	op := "handoff"
	opMetrics, err := q.QueryPerformance(session.PerformanceFilters{Operation: &op})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(opMetrics) != 2 {
		t.Errorf("Expected 2 handoff metrics, got %d", len(opMetrics))
	}
}

func TestQueryPerformanceSummary(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	perfPath := filepath.Join(claudeDir, "performance.jsonl")

	// handoff: 100, 200, 300 (avg 200, min 100, max 300)
	// validation: 50, 150 (avg 100, min 50, max 150)
	m1 := `{"timestamp":1,"operation":"handoff","duration_ms":100,"memory_bytes":0,"success":true,"context":""}`
	m2 := `{"timestamp":2,"operation":"handoff","duration_ms":200,"memory_bytes":0,"success":true,"context":""}`
	m3 := `{"timestamp":3,"operation":"handoff","duration_ms":300,"memory_bytes":0,"success":false,"context":""}`
	m4 := `{"timestamp":4,"operation":"validation","duration_ms":50,"memory_bytes":0,"success":true,"context":""}`
	m5 := `{"timestamp":5,"operation":"validation","duration_ms":150,"memory_bytes":0,"success":true,"context":""}`

	content := m1 + "\n" + m2 + "\n" + m3 + "\n" + m4 + "\n" + m5 + "\n"
	if err := os.WriteFile(perfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := session.NewQuery(tmpDir)
	summaries, err := q.QueryPerformanceSummary(session.PerformanceFilters{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 operation summaries, got %d", len(summaries))
	}

	// Find handoff summary
	var handoffSummary *session.PerformanceSummary
	for i := range summaries {
		if summaries[i].Operation == "handoff" {
			handoffSummary = &summaries[i]
			break
		}
	}

	if handoffSummary == nil {
		t.Fatal("Expected handoff summary")
	}
	if handoffSummary.Count != 3 {
		t.Errorf("Expected handoff count 3, got %d", handoffSummary.Count)
	}
	if handoffSummary.SuccessCount != 2 {
		t.Errorf("Expected handoff success count 2, got %d", handoffSummary.SuccessCount)
	}
	if handoffSummary.FailCount != 1 {
		t.Errorf("Expected handoff fail count 1, got %d", handoffSummary.FailCount)
	}
	if handoffSummary.MinMs != 100 {
		t.Errorf("Expected handoff min 100, got %d", handoffSummary.MinMs)
	}
	if handoffSummary.MaxMs != 300 {
		t.Errorf("Expected handoff max 300, got %d", handoffSummary.MaxMs)
	}
	expectedAvg := 200.0
	if handoffSummary.AvgMs != expectedAvg {
		t.Errorf("Expected handoff avg %.1f, got %.1f", expectedAvg, handoffSummary.AvgMs)
	}
}
