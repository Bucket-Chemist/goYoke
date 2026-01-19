package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveArtifacts_Success(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create artifacts
	learningsPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(learningsPath, []byte(`{"file":"test.go","error":"test error"}
`), 0644)

	violationsPath := filepath.Join(memoryDir, "routing-violations.jsonl")
	os.WriteFile(violationsPath, []byte(`{"violation":"test"}
`), 0644)

	cfg := HandoffConfig{
		HandoffPath:    filepath.Join(memoryDir, "handoffs.jsonl"),
		ViolationsPath: violationsPath,
	}

	// Archive
	err := ArchiveArtifacts(cfg, "test-session-123")
	if err != nil {
		t.Fatalf("ArchiveArtifacts failed: %v", err)
	}

	// Verify files moved
	archiveDir := filepath.Join(memoryDir, "session-archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Error("Archive directory not created")
	}

	// Original files should be gone
	if _, err := os.Stat(learningsPath); !os.IsNotExist(err) {
		t.Error("Learnings file not moved")
	}

	if _, err := os.Stat(violationsPath); !os.IsNotExist(err) {
		t.Error("Violations file not moved")
	}

	// Check archived files exist
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("Failed to read archive dir: %v", err)
	}

	if len(entries) < 2 {
		t.Errorf("Expected at least 2 archived files, got %d", len(entries))
	}

	hasLearnings := false
	hasViolations := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".jsonl" {
			if len(entry.Name()) > 9 && entry.Name()[:9] == "learnings" {
				hasLearnings = true
			}
			if len(entry.Name()) > 10 && entry.Name()[:10] == "violations" {
				hasViolations = true
			}
		}
	}

	if !hasLearnings {
		t.Error("Learnings file not found in archive")
	}
	if !hasViolations {
		t.Error("Violations file not found in archive")
	}
}

func TestArchiveArtifacts_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	cfg := HandoffConfig{
		HandoffPath:    filepath.Join(memoryDir, "handoffs.jsonl"),
		ViolationsPath: filepath.Join(memoryDir, "routing-violations.jsonl"), // Doesn't exist
	}

	// Should not error when files missing
	err := ArchiveArtifacts(cfg, "test-session")
	if err != nil {
		t.Errorf("ArchiveArtifacts should gracefully handle missing files, got error: %v", err)
	}
}

func TestArchiveArtifacts_TranscriptCopy(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	os.WriteFile(transcriptPath, []byte(`{"event":"test"}
`), 0644)

	cfg := HandoffConfig{
		HandoffPath:    filepath.Join(memoryDir, "handoffs.jsonl"),
		TranscriptPath: transcriptPath,
		ViolationsPath: "",
	}

	err := ArchiveArtifacts(cfg, "session-456")
	if err != nil {
		t.Fatalf("ArchiveArtifacts failed: %v", err)
	}

	// Check transcript copied
	archiveDir := filepath.Join(memoryDir, "session-archive")
	transcriptCopy := filepath.Join(archiveDir, "session-session-456.jsonl")

	if _, err := os.Stat(transcriptCopy); os.IsNotExist(err) {
		t.Error("Transcript was not copied to archive")
	}

	// Original should still exist (copy, not move)
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		t.Error("Original transcript should not be deleted")
	}
}

func TestArchiveArtifacts_ErrorFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Use non-writable directory to trigger error
	memoryDir := filepath.Join(tmpDir, "readonly", ".claude", "memory")
	os.MkdirAll(filepath.Dir(memoryDir), 0755)
	os.Chmod(filepath.Dir(memoryDir), 0444) // Read-only
	defer os.Chmod(filepath.Dir(memoryDir), 0755)

	cfg := HandoffConfig{
		HandoffPath: filepath.Join(memoryDir, "handoffs.jsonl"),
	}

	err := ArchiveArtifacts(cfg, "test")
	if err == nil {
		t.Error("Expected error for non-writable directory")
	}

	// Error should have [session-archive] component tag
	if err != nil && !strings.Contains(err.Error(), "[session-archive]") {
		t.Errorf("Error missing [session-archive] component tag: %v", err)
	}
}
