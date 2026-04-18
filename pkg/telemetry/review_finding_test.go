package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

func setupReviewTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)

	// Create minimal structure
	os.MkdirAll(filepath.Join(dir, ".gogent"), 0755)
	os.MkdirAll(filepath.Join(dir, ".gogent", "memory"), 0755)

	return func() { /* TempDir auto-cleans */ }
}

func TestNewReviewFinding(t *testing.T) {
	finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test message")

	if finding.FindingID == "" {
		t.Error("FindingID should not be empty")
	}
	if finding.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got %q", finding.Severity)
	}
	if finding.Reviewer != "backend-reviewer" {
		t.Errorf("Expected reviewer 'backend-reviewer', got %q", finding.Reviewer)
	}
	if finding.Timestamp == 0 {
		t.Error("Timestamp should not be 0")
	}
}

func TestLogReviewFinding(t *testing.T) {
	cleanup := setupReviewTestDir(t)
	defer cleanup()

	finding := NewReviewFinding("session1", "backend-reviewer", "critical", "security", "file.go", 10, "test message")
	err := LogReviewFinding(finding)
	if err != nil {
		t.Fatalf("LogReviewFinding failed: %v", err)
	}

	// Verify file written
	path := config.GetReviewFindingsPathWithProjectDir()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(content) == 0 {
		t.Error("File should not be empty")
	}
	if !strings.Contains(string(content), finding.FindingID) {
		t.Error("File should contain finding ID")
	}
}

func TestLookupFindingTimestamp(t *testing.T) {
	cleanup := setupReviewTestDir(t)
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

func TestLookupFindingTimestamp_NotFound(t *testing.T) {
	cleanup := setupReviewTestDir(t)
	defer cleanup()

	_, err := LookupFindingTimestamp("nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent finding")
	}
}

func TestUpdateReviewFindingOutcome(t *testing.T) {
	cleanup := setupReviewTestDir(t)
	defer cleanup()

	err := UpdateReviewFindingOutcome("finding-123", "fixed", "TICKET-1", "abc123", 5000)
	if err != nil {
		t.Fatalf("UpdateReviewFindingOutcome failed: %v", err)
	}

	// Verify appended to outcomes file
	path := config.GetReviewOutcomesPathWithProjectDir()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "finding-123") {
		t.Error("File should contain finding ID")
	}
	if !strings.Contains(string(content), "fixed") {
		t.Error("File should contain resolution")
	}
}

func TestTruncateMessage(t *testing.T) {
	short := "short message"
	if truncateMessage(short, 1000) != short {
		t.Error("Short message should not be truncated")
	}

	long := strings.Repeat("a", 2000)
	truncated := truncateMessage(long, 1000)
	if len(truncated) != 1003 { // 1000 + "..."
		t.Errorf("Expected length 1003, got %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("Truncated message should end with ...")
	}
}

func TestReadReviewFindings(t *testing.T) {
	cleanup := setupReviewTestDir(t)
	defer cleanup()

	// Write multiple findings
	for i := 0; i < 3; i++ {
		finding := NewReviewFinding("session1", "reviewer", "warning", "test", "file.go", i, "msg")
		LogReviewFinding(finding)
	}

	findings, err := ReadReviewFindings()
	if err != nil {
		t.Fatalf("ReadReviewFindings failed: %v", err)
	}
	if len(findings) != 3 {
		t.Errorf("Expected 3 findings, got %d", len(findings))
	}
}

func TestReadReviewFindings_Empty(t *testing.T) {
	cleanup := setupReviewTestDir(t)
	defer cleanup()

	findings, err := ReadReviewFindings()
	if err != nil {
		t.Fatalf("ReadReviewFindings failed: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
}
