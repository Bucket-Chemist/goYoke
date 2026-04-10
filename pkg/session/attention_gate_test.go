package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetFlushThreshold_Default(t *testing.T) {
	// Clear env var
	os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d, got: %d", DefaultFlushThreshold, threshold)
	}
}

func TestGetFlushThreshold_EnvVar(t *testing.T) {
	// Set custom threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "10")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != 10 {
		t.Errorf("Expected 10 from env var, got: %d", threshold)
	}
}

func TestGetFlushThreshold_InvalidEnvVar(t *testing.T) {
	// Set invalid threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "invalid")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d on invalid env, got: %d", DefaultFlushThreshold, threshold)
	}
}

func TestGetFlushThreshold_ZeroEnvVar(t *testing.T) {
	// Set zero threshold (invalid - must be positive)
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "0")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d when env var is 0, got: %d", DefaultFlushThreshold, threshold)
	}
}

func TestGetFlushThreshold_NegativeEnvVar(t *testing.T) {
	// Set negative threshold (invalid - must be positive)
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "-5")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	threshold := GetFlushThreshold()
	if threshold != DefaultFlushThreshold {
		t.Errorf("Expected default %d when env var is negative, got: %d", DefaultFlushThreshold, threshold)
	}
}

func TestGenerateRoutingReminder(t *testing.T) {
	summary := "haiku: find, search... sonnet: implement..."
	reminder := GenerateRoutingReminder(10, summary)

	if !strings.Contains(reminder, "Tool #10") {
		t.Error("Should include tool count")
	}

	if !strings.Contains(reminder, "codebase-search") {
		t.Error("Should mention codebase-search")
	}

	if !strings.Contains(reminder, "routing-schema.json") {
		t.Error("Should reference routing schema")
	}

	if !strings.Contains(reminder, summary) {
		t.Error("Should include routing summary")
	}
}

func TestCountPendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings file
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"ts":123,"file":"test.go"}
{"ts":456,"file":"main.go"}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	count, err := CountPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to check: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 entries, got: %d", count)
	}
}

func TestCountPendingLearnings_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := CountPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing file: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 entries when file missing, got: %d", count)
	}
}

func TestCountPendingLearnings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(pendingPath, []byte(""), 0644)

	count, err := CountPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to check empty file: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 entries for empty file, got: %d", count)
	}
}

func TestShouldFlushLearnings_BelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Set custom threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "5")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	// Create pending learnings with 3 entries (below threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 3; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if shouldFlush {
		t.Error("Should not flush when count < threshold")
	}

	if count != 3 {
		t.Errorf("Expected 3 entries, got: %d", count)
	}
}

func TestShouldFlushLearnings_ConfigurableThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Set custom threshold
	os.Setenv("GOGENT_FLUSH_THRESHOLD", "3")
	defer os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	// Create pending learnings with 4 entries (above threshold of 3)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 4; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count >= custom threshold")
	}

	if count != 4 {
		t.Errorf("Expected 4 entries, got: %d", count)
	}
}

func TestShouldFlushLearnings_ExactlyAtThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Use default threshold
	os.Unsetenv("GOGENT_FLUSH_THRESHOLD")

	// Create exactly DefaultFlushThreshold entries
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < DefaultFlushThreshold; i++ {
		lines = append(lines, "{}")
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count == threshold")
	}

	if count != DefaultFlushThreshold {
		t.Errorf("Expected %d entries, got: %d", DefaultFlushThreshold, count)
	}
}

func TestArchivePendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"ts":123}
{"ts":456}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	ctx, err := ArchivePendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to archive: %v", err)
	}

	if ctx.EntryCount != 2 {
		t.Errorf("Expected 2 entries archived, got: %d", ctx.EntryCount)
	}

	// Verify pending is cleared
	data, _ := os.ReadFile(pendingPath)
	if string(data) != "" {
		t.Error("Pending learnings should be cleared")
	}

	// Verify archive exists
	if _, err := os.Stat(ctx.ArchivedFile); os.IsNotExist(err) {
		t.Error("Archive file should exist")
	}

	// Verify archive directory was created
	sharpEdgesDir := filepath.Join(tmpDir, ".gogent", "memory", "sharp-edges")
	if _, err := os.Stat(sharpEdgesDir); os.IsNotExist(err) {
		t.Error("Archive directory should be created")
	}

	// Verify archive file has correct format
	if !strings.Contains(ctx.ArchivedFile, "auto-flush-") {
		t.Error("Archive filename should contain 'auto-flush-'")
	}
	if !strings.HasSuffix(ctx.ArchivedFile, ".jsonl") {
		t.Error("Archive filename should end with '.jsonl'")
	}

	// Verify remaining count is 0
	if ctx.PendingRemaining != 0 {
		t.Errorf("Expected 0 remaining entries, got: %d", ctx.PendingRemaining)
	}
}

func TestArchivePendingLearnings_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, err := ArchivePendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing file: %v", err)
	}

	if ctx.EntryCount != 0 {
		t.Errorf("Expected 0 entries when file missing, got: %d", ctx.EntryCount)
	}
}

func TestArchivePendingLearnings_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".gogent", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	os.WriteFile(pendingPath, []byte("{}\n"), 0644)

	// sharp-edges directory should NOT exist yet
	sharpEdgesDir := filepath.Join(tmpDir, ".gogent", "memory", "sharp-edges")
	if _, err := os.Stat(sharpEdgesDir); !os.IsNotExist(err) {
		t.Fatal("sharp-edges directory should not exist before archival")
	}

	_, err := ArchivePendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Failed to archive: %v", err)
	}

	// Directory should be created now
	if _, err := os.Stat(sharpEdgesDir); os.IsNotExist(err) {
		t.Error("Archive directory should be created by ArchivePendingLearnings")
	}
}

func TestGenerateFlushNotification(t *testing.T) {
	ctx := &FlushContext{
		EntryCount:   3,
		ArchivedFile: "/path/to/archive.jsonl",
	}

	notification := GenerateFlushNotification(ctx)

	if !strings.Contains(notification, "3") {
		t.Error("Should include entry count")
	}

	if !strings.Contains(notification, "archive") {
		t.Error("Should mention archive")
	}

	if !strings.Contains(notification, "/path/to/archive.jsonl") {
		t.Error("Should include archive file path")
	}

	if !strings.Contains(notification, "sharp-edges") {
		t.Error("Should reference sharp-edges")
	}
}

func TestGenerateGateResponse_OnlyReminder(t *testing.T) {
	reminder := "Test reminder message"
	response := GenerateGateResponse(true, false, reminder, "")

	if !strings.Contains(response, "PostToolUse") {
		t.Error("Should include PostToolUse event name")
	}

	if !strings.Contains(response, "Test reminder message") {
		t.Error("Should include reminder message")
	}

	if !strings.Contains(response, "additionalContext") {
		t.Error("Should include additionalContext field")
	}
}

func TestGenerateGateResponse_OnlyFlush(t *testing.T) {
	flush := "Test flush message"
	response := GenerateGateResponse(false, true, "", flush)

	if !strings.Contains(response, "PostToolUse") {
		t.Error("Should include PostToolUse event name")
	}

	if !strings.Contains(response, "Test flush message") {
		t.Error("Should include flush message")
	}
}

func TestGenerateGateResponse_Both(t *testing.T) {
	reminder := "Test reminder"
	flush := "Test flush"
	response := GenerateGateResponse(true, true, reminder, flush)

	if !strings.Contains(response, "Test reminder") {
		t.Error("Should include reminder message")
	}

	if !strings.Contains(response, "Test flush") {
		t.Error("Should include flush message")
	}
}

func TestGenerateGateResponse_Neither(t *testing.T) {
	response := GenerateGateResponse(false, false, "", "")

	if !strings.Contains(response, "PostToolUse") {
		t.Error("Should include PostToolUse event name")
	}

	if !strings.Contains(response, "additionalContext") {
		t.Error("Should include additionalContext field")
	}

	// Should have empty additionalContext
	if !strings.Contains(response, `"additionalContext": ""`) {
		t.Error("Should have empty additionalContext when neither reminder nor flush")
	}
}
