package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestLogMLToolEvent_GlobalPath(t *testing.T) {
	// Test global path creation and writing
	event := &routing.PostToolEvent{
		ToolName:   "Read",
		DurationMs: 150,
		Model:      "haiku",
	}

	projectDir := t.TempDir()
	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	// Verify global file created via config helper
	globalPath := config.GetMLToolEventsPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global log file should exist at %s", globalPath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestLogMLToolEvent_ProjectPath(t *testing.T) {
	// Test project path writing when directory exists
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".claude", "memory"), 0755)

	event := &routing.PostToolEvent{
		ToolName:   "Read",
		DurationMs: 150,
		Model:      "haiku",
	}

	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".claude", "memory", "ml-tool-events.jsonl")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Fatalf("Project log file should exist at %s", projectPath)
	}
}

func TestLogMLToolEvent_NoProjectPath(t *testing.T) {
	// Test that logging succeeds even when project path doesn't exist
	// (project write errors are silently ignored)
	projectDir := t.TempDir()
	// Do NOT create .claude/memory directory

	event := &routing.PostToolEvent{
		ToolName:   "Read",
		DurationMs: 150,
		Model:      "haiku",
	}

	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log (should succeed even without project dir): %v", err)
	}

	// Verify global path still written
	globalPath := config.GetMLToolEventsPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global log file should exist at %s", globalPath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestReadMLToolEvents_Empty(t *testing.T) {
	// Test reading when log doesn't exist
	// Remove global file if it exists
	globalPath := config.GetMLToolEventsPath()
	os.Remove(globalPath)

	events, err := ReadMLToolEvents()

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got: %d", len(events))
	}
}

func TestReadMLToolEvents(t *testing.T) {
	// Test reading multiple logged events
	projectDir := t.TempDir()

	// Clean up global path first
	globalPath := config.GetMLToolEventsPath()
	os.Remove(globalPath)

	for i := 0; i < 3; i++ {
		event := &routing.PostToolEvent{
			ToolName:   "Read",
			DurationMs: int64(100 + i*50),
			Model:      "haiku",
		}
		LogMLToolEvent(event, projectDir)
	}

	events, err := ReadMLToolEvents()

	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got: %d", len(events))
	}

	// Verify events were parsed correctly
	if events[0].ToolName != "Read" {
		t.Errorf("Expected ToolName 'Read', got: %s", events[0].ToolName)
	}

	if events[0].DurationMs != 100 {
		t.Errorf("Expected DurationMs 100, got: %d", events[0].DurationMs)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestReadMLToolEvents_MalformedLines(t *testing.T) {
	// Test graceful handling of malformed JSONL lines
	globalPath := config.GetMLToolEventsPath()

	// Clean up first
	os.Remove(globalPath)

	// Write some valid and invalid lines
	dir := filepath.Dir(globalPath)
	os.MkdirAll(dir, 0755)

	f, err := os.Create(globalPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Valid event
	f.WriteString(`{"tool_name":"Read","duration_ms":100,"model":"haiku"}` + "\n")
	// Malformed JSON
	f.WriteString(`{invalid json}` + "\n")
	// Another valid event
	f.WriteString(`{"tool_name":"Write","duration_ms":200,"model":"sonnet"}` + "\n")
	f.Close()

	events, err := ReadMLToolEvents()
	if err != nil {
		t.Fatalf("Should not error on malformed lines, got: %v", err)
	}

	// Should have skipped the malformed line
	if len(events) != 2 {
		t.Errorf("Expected 2 valid events, got: %d", len(events))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestCalculateMLSessionStats(t *testing.T) {
	events := []routing.PostToolEvent{
		{ToolName: "Read", DurationMs: 100, Model: "haiku", InputTokens: 1000, OutputTokens: 0},
		{ToolName: "Write", DurationMs: 200, Model: "sonnet", InputTokens: 1000, OutputTokens: 0},
		{ToolName: "Read", DurationMs: 150, Model: "haiku", InputTokens: 1000, OutputTokens: 0},
	}

	stats := CalculateMLSessionStats(events)

	if stats["event_count"] != 3 {
		t.Errorf("Expected 3 events, got: %v", stats["event_count"])
	}

	if stats["total_duration"] != int64(450) {
		t.Errorf("Expected 450ms total, got: %v", stats["total_duration"])
	}

	if stats["avg_duration"] != int64(150) {
		t.Errorf("Expected 150ms average, got: %v", stats["avg_duration"])
	}

	breakdown := stats["tool_breakdown"].(map[string]int)
	if breakdown["Read"] != 2 {
		t.Errorf("Expected 2 Read calls, got: %d", breakdown["Read"])
	}

	if breakdown["Write"] != 1 {
		t.Errorf("Expected 1 Write call, got: %d", breakdown["Write"])
	}
}

func TestCalculateMLSessionStats_Empty(t *testing.T) {
	// Test stats calculation with no events
	events := []routing.PostToolEvent{}
	stats := CalculateMLSessionStats(events)

	if stats["event_count"] != 0 {
		t.Errorf("Expected 0 events, got: %v", stats["event_count"])
	}

	if stats["total_duration"] != 0 {
		t.Errorf("Expected 0 duration, got: %v", stats["total_duration"])
	}

	if stats["total_cost"] != 0.0 {
		t.Errorf("Expected 0.0 cost, got: %v", stats["total_cost"])
	}
}

func TestDualWrite(t *testing.T) {
	// Test that events are written to both global and project paths when available
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".claude", "memory"), 0755)

	// Clean up global path first
	globalPath := config.GetMLToolEventsPath()
	os.Remove(globalPath)

	event := &routing.PostToolEvent{
		ToolName:   "Read",
		DurationMs: 150,
		Model:      "haiku",
	}

	err := LogMLToolEvent(event, projectDir)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	// Verify both files exist
	projectPath := filepath.Join(projectDir, ".claude", "memory", "ml-tool-events.jsonl")

	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global log file should exist at %s", globalPath)
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Fatalf("Project log file should exist at %s", projectPath)
	}

	// Verify both files have content
	globalData, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("Failed to read global file: %v", err)
	}
	if len(globalData) == 0 {
		t.Error("Global file should have content")
	}

	projectData, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("Failed to read project file: %v", err)
	}
	if len(projectData) == 0 {
		t.Error("Project file should have content")
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestDirExists(t *testing.T) {
	// Test dirExists helper function
	tempDir := t.TempDir()

	// Existing directory
	if !dirExists(tempDir) {
		t.Error("dirExists should return true for existing directory")
	}

	// Non-existent directory
	if dirExists(filepath.Join(tempDir, "nonexistent")) {
		t.Error("dirExists should return false for non-existent directory")
	}

	// File (not a directory)
	filePath := filepath.Join(tempDir, "testfile")
	os.WriteFile(filePath, []byte("test"), 0644)
	if dirExists(filePath) {
		t.Error("dirExists should return false for files")
	}
}

func TestAppendMLToolEvent(t *testing.T) {
	// Test appendMLToolEvent helper function
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "subdir", "test.jsonl")

	// First append (creates file and directories)
	err := appendMLToolEvent(filePath, []byte(`{"test":"data1"}`+"\n"))
	if err != nil {
		t.Fatalf("First append failed: %v", err)
	}

	// Second append (appends to existing file)
	err = appendMLToolEvent(filePath, []byte(`{"test":"data2"}`+"\n"))
	if err != nil {
		t.Fatalf("Second append failed: %v", err)
	}

	// Verify file has both lines
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	expectedLines := 2
	actualLines := 0
	for _, c := range content {
		if c == '\n' {
			actualLines++
		}
	}

	if actualLines != expectedLines {
		t.Errorf("Expected %d lines, got %d. Content: %s", expectedLines, actualLines, content)
	}
}

func TestXDGCompliance(t *testing.T) {
	// Test that config.GetMLToolEventsPath() uses config.GetGOgentDataDir()
	// This verifies XDG Base Directory specification compliance
	mlPath := config.GetMLToolEventsPath()
	dataDir := config.GetGOgentDataDir()

	// Verify ML path is under data directory
	if !filepath.HasPrefix(mlPath, dataDir) {
		t.Errorf("ML tool events path should be under data directory. Got: %s, Expected prefix: %s", mlPath, dataDir)
	}

	// Verify filename is tool-events.jsonl
	expectedFilename := "tool-events.jsonl"
	actualFilename := filepath.Base(mlPath)
	if actualFilename != expectedFilename {
		t.Errorf("Expected filename %s, got: %s", expectedFilename, actualFilename)
	}
}
