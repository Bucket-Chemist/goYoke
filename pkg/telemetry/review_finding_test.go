package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)
	os.MkdirAll(filepath.Join(dir, ".gogent"), 0755)
	os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755)
	return func() {}
}

func TestNewReviewFinding(t *testing.T) {
	finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test message")

	if finding.FindingID == "" {
		t.Error("FindingID should not be empty")
	}
	if finding.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", finding.Severity)
	}
	if finding.SessionID != "session1" {
		t.Errorf("Expected session 'session1', got '%s'", finding.SessionID)
	}
}

func TestLogReviewFinding(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test message")
	err := LogReviewFinding(finding)
	if err != nil {
		t.Fatalf("LogReviewFinding failed: %v", err)
	}

	// Read back and verify
	findings, err := ReadReviewFindings()
	if err != nil {
		t.Fatalf("ReadReviewFindings failed: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
}

func TestLookupFindingTimestamp(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	finding := NewReviewFinding("session1", "reviewer", "warning", "perf", "file.go", 1, "msg")
	LogReviewFinding(finding)

	ts, err := LookupFindingTimestamp(finding.FindingID)
	if err != nil {
		t.Fatalf("LookupFindingTimestamp failed: %v", err)
	}
	if ts != finding.Timestamp {
		t.Errorf("Expected timestamp %d, got %d", finding.Timestamp, ts)
	}
}

func TestUpdateReviewFindingOutcome(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	err := UpdateReviewFindingOutcome("finding-123", "fixed", "TICKET-1", "abc123", 5000)
	if err != nil {
		t.Fatalf("UpdateReviewFindingOutcome failed: %v", err)
	}
}

func TestCalculateReviewStats(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "critical", Reviewer: "backend-reviewer", Category: "security"},
		{Severity: "warning", Reviewer: "backend-reviewer", Category: "performance"},
		{Severity: "critical", Reviewer: "frontend-reviewer", Category: "security"},
	}

	stats := CalculateReviewStats(findings)

	if stats["total_findings"].(int) != 3 {
		t.Errorf("Expected 3 total findings")
	}
}
