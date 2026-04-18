package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

func TestAttentionGateWorkflow_FullFlush(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings (6 entries, above default threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 6; i++ {
		lines = append(lines, `{"file":"test.go","line":10}`)
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	// Check flush trigger
	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ShouldFlushLearnings failed: %v", err)
	}

	if !shouldFlush {
		t.Error("Should flush when count >= threshold")
	}

	if count != 6 {
		t.Errorf("Expected 6 entries, got: %d", count)
	}

	// Execute flush
	ctx, err := ArchivePendingLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ArchivePendingLearnings failed: %v", err)
	}

	if ctx.EntryCount != 6 {
		t.Errorf("Expected 6 entries archived, got: %d", ctx.EntryCount)
	}

	// Verify pending is cleared
	data, _ := os.ReadFile(pendingPath)
	if string(data) != "" {
		t.Error("Pending learnings should be cleared after flush")
	}

	// Verify archive exists
	if _, err := os.Stat(ctx.ArchivedFile); os.IsNotExist(err) {
		t.Error("Archive file should exist")
	}

	// Verify archive contains expected data
	archiveData, _ := os.ReadFile(ctx.ArchivedFile)
	archiveLines := strings.Count(string(archiveData), "\n")
	if archiveLines != 6 {
		t.Errorf("Archive should have 6 lines, got: %d", archiveLines)
	}
}

func TestAttentionGateWorkflow_NoFlushBelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings (4 entries, BELOW default threshold of 5)
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var lines []string
	for i := 0; i < 4; i++ {
		lines = append(lines, `{}`)
	}
	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(pendingPath, []byte(content), 0644)

	shouldFlush, count, err := ShouldFlushLearnings(tmpDir)
	if err != nil {
		t.Fatalf("ShouldFlushLearnings failed: %v", err)
	}

	if shouldFlush {
		t.Error("Should NOT flush when count < threshold")
	}

	if count != 4 {
		t.Errorf("Expected 4 entries, got: %d", count)
	}
}

func TestAttentionGateWorkflow_SimulationHarness(t *testing.T) {
	// Simulation harness integration test
	// Simulates complete attention-gate workflow across 30 tool calls

	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)
	memoryDir := filepath.Join(tmpDir, ".goyoke", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create initial pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	var initialLines []string
	for i := 0; i < 7; i++ {
		initialLines = append(initialLines, `{}`)
	}
	os.WriteFile(pendingPath, []byte(strings.Join(initialLines, "\n")+"\n"), 0644)

	var reminderCount, flushCount int

	// Simulate 30 tool calls
	for i := 1; i <= 30; i++ {
		count, err := config.GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Tool %d increment failed: %v", i, err)
		}

		// Check reminder
		if config.ShouldRemind(count) {
			reminderCount++
			reminder := GenerateRoutingReminder(count, "haiku: search... sonnet: implement...")
			if !strings.Contains(reminder, "CHECKPOINT") {
				t.Errorf("Tool %d: reminder should contain checkpoint", i)
			}
		}

		// Check flush
		if config.ShouldFlush(count) {
			shouldFlush, pendingCount, _ := ShouldFlushLearnings(tmpDir)
			if shouldFlush {
				flushCount++
				ctx, err := ArchivePendingLearnings(tmpDir)
				if err != nil {
					t.Errorf("Tool %d: flush failed: %v", i, err)
				} else {
					if ctx.EntryCount != pendingCount {
						t.Errorf("Tool %d: flushed %d but expected %d", i, ctx.EntryCount, pendingCount)
					}
				}
			}
		}
	}

	// Verify simulation results
	if reminderCount != 3 {
		t.Errorf("Expected 3 reminders (at 10, 20, 30), got: %d", reminderCount)
	}

	if flushCount == 0 {
		t.Error("Expected at least 1 flush during 30 tool calls")
	}
}
