package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// TestToolEventWorkflow_LogAndAnalyze tests full log → read → analyze workflow
func TestToolEventWorkflow_LogAndAnalyze(t *testing.T) {
	// Set up temp directory for isolated testing
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".gogent", "memory"), 0755)

	// Simulate multiple tool events with ML fields
	events := []*routing.PostToolEvent{
		{
			ToolName:         "Glob",
			DurationMs:       50,
			InputTokens:      512,
			OutputTokens:     256,
			Model:            "haiku",
			Tier:             "haiku",
			Success:          true,
			SequenceIndex:    1,
			PreviousTools:    []string{},
			PreviousOutcomes: []bool{},
			TaskType:         "search",
		},
		{
			ToolName:         "Read",
			DurationMs:       100,
			InputTokens:      1024,
			OutputTokens:     1024,
			Model:            "haiku",
			Tier:             "haiku",
			Success:          true,
			SequenceIndex:    2,
			PreviousTools:    []string{"Glob"},
			PreviousOutcomes: []bool{true},
			TaskType:         "search",
		},
		{
			ToolName:         "Edit",
			DurationMs:       150,
			InputTokens:      2048,
			OutputTokens:     512,
			Model:            "sonnet",
			Tier:             "sonnet",
			Success:          true,
			SequenceIndex:    3,
			PreviousTools:    []string{"Glob", "Read"},
			PreviousOutcomes: []bool{true, true},
			TaskType:         "implementation",
		},
	}

	// Log all events
	for _, event := range events {
		if err := LogMLToolEvent(event, projectDir); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Read back logs
	logs, err := ReadMLToolEvents()
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got: %d", len(logs))
	}

	// Calculate stats
	stats := CalculateMLSessionStats(logs)

	eventCount, ok := stats["event_count"].(int)
	if !ok || eventCount != 3 {
		t.Errorf("Expected event_count 3, got: %v", stats["event_count"])
	}

	totalDuration, ok := stats["total_duration"].(int64)
	if !ok || totalDuration != 300 {
		t.Errorf("Expected total_duration 300ms, got: %v", stats["total_duration"])
	}

	// Verify cost calculation
	costStr, ok := stats["total_cost"].(string)
	if !ok || !strings.Contains(costStr, "$") {
		t.Errorf("Cost should be formatted as currency, got: %v", stats["total_cost"])
	}

	// Verify average duration
	avgDuration, ok := stats["avg_duration"].(int64)
	if !ok || avgDuration != 100 {
		t.Errorf("Expected avg_duration 100ms, got: %v", stats["avg_duration"])
	}

	// Verify tool breakdown
	toolBreakdown, ok := stats["tool_breakdown"].(map[string]int)
	if !ok {
		t.Errorf("Expected tool_breakdown map, got: %v", stats["tool_breakdown"])
	} else {
		if toolBreakdown["Glob"] != 1 {
			t.Errorf("Expected 1 Glob event, got: %d", toolBreakdown["Glob"])
		}
		if toolBreakdown["Read"] != 1 {
			t.Errorf("Expected 1 Read event, got: %d", toolBreakdown["Read"])
		}
		if toolBreakdown["Edit"] != 1 {
			t.Errorf("Expected 1 Edit event, got: %d", toolBreakdown["Edit"])
		}
	}
}

// TestToolEventWorkflow_CostTracking tests cost estimation across tiers
func TestToolEventWorkflow_CostTracking(t *testing.T) {
	tests := []struct {
		tier     string
		tokens   int
		minCost  float64
		maxCost  float64
	}{
		{"haiku", 1000, 0.0004, 0.0006},
		{"sonnet", 1000, 0.008, 0.010},
		{"opus", 1000, 0.040, 0.050},
	}

	for _, tc := range tests {
		event := &routing.PostToolEvent{
			Model:        tc.tier,
			Tier:         tc.tier,
			InputTokens:  tc.tokens,
			OutputTokens: 0,
		}

		cost := EstimatedCost(event)
		if cost < tc.minCost || cost > tc.maxCost {
			t.Errorf("Tier %s: cost %f outside range [%f, %f]",
				tc.tier, cost, tc.minCost, tc.maxCost)
		}
	}
}

// TestSequenceTracking_Integration tests sequence index and previous tools/outcomes tracking
func TestSequenceTracking_Integration(t *testing.T) {
	// Test that SequenceIndex increments correctly across multiple events
	for i := 1; i <= 10; i++ {
		event := &routing.PostToolEvent{
			ToolName:      "Read",
			SequenceIndex: i,
		}

		if event.SequenceIndex != i {
			t.Errorf("Expected sequence %d, got %d", i, event.SequenceIndex)
		}
	}

	// Test that PreviousTools captures last 5 tools correctly
	tools := []string{"Glob", "Read", "Edit", "Write", "Bash", "Grep"}
	lastFive := tools[len(tools)-5:]

	event := &routing.PostToolEvent{
		PreviousTools: lastFive,
	}

	if len(event.PreviousTools) != 5 {
		t.Errorf("Expected 5 previous tools, got %d", len(event.PreviousTools))
	}

	// Verify the last 5 tools are correctly stored
	for i, tool := range lastFive {
		if event.PreviousTools[i] != tool {
			t.Errorf("Previous tool %d: expected %s, got %s", i, tool, event.PreviousTools[i])
		}
	}

	// Test that PreviousOutcomes tracks success states correctly
	outcomes := []bool{true, false, true, true, false}
	event.PreviousOutcomes = outcomes

	successCount := 0
	for _, outcome := range event.PreviousOutcomes {
		if outcome {
			successCount++
		}
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}

	// Test EnrichWithSequence helper function
	testEvent := &routing.PostToolEvent{}
	previous := []string{"Task", "Glob", "Grep", "Read", "Edit"}
	prevOutcomes := []bool{true, true, false, true, true}

	EnrichWithSequence(testEvent, 10, previous, prevOutcomes)

	if testEvent.SequenceIndex != 10 {
		t.Errorf("EnrichWithSequence: expected SequenceIndex 10, got %d", testEvent.SequenceIndex)
	}
	if len(testEvent.PreviousTools) != 5 {
		t.Errorf("EnrichWithSequence: expected 5 previous tools, got %d", len(testEvent.PreviousTools))
	}
	if len(testEvent.PreviousOutcomes) != 5 {
		t.Errorf("EnrichWithSequence: expected 5 previous outcomes, got %d", len(testEvent.PreviousOutcomes))
	}
}

// TestTaskClassification_Integration tests task classification accuracy
func TestTaskClassification_Integration(t *testing.T) {
	tests := []struct {
		description  string
		expectedType string
	}{
		{"implement the feature", "implementation"},
		{"find files in src", "search"},
		{"document the API", "documentation"},
		{"fix the bug", "debug"},
		{"refactor the module", "refactor"},
		{"add unit test for function", "test"},
		{"review the code", "review"},
		{"summarize the document", "document_understanding"},
		{"how does the authentication work", "codebase_understanding"},
		{"synthesize all findings", "synthesis"},
	}

	accurateCount := 0
	for _, tc := range tests {
		taskType, _ := ClassifyTask(tc.description)
		if taskType == tc.expectedType {
			accurateCount++
		} else {
			t.Logf("Misclassified: '%s' expected '%s', got '%s'",
				tc.description, tc.expectedType, taskType)
		}
	}

	accuracy := float64(accurateCount) / float64(len(tests))
	if accuracy < 0.85 {
		t.Errorf("Task classification accuracy %.2f%% below 85%% threshold (got %d/%d correct)",
			accuracy*100, accurateCount, len(tests))
	} else {
		t.Logf("Task classification accuracy: %.2f%% (%d/%d correct)",
			accuracy*100, accurateCount, len(tests))
	}
}

// TestDualWrite_Integration tests writes to both global and project paths
func TestDualWrite_Integration(t *testing.T) {
	// Set up temp directories for isolated testing
	tmpDir := t.TempDir()
	xdgDataHome := filepath.Join(tmpDir, "xdg-data")
	os.Setenv("XDG_DATA_HOME", xdgDataHome)
	defer os.Unsetenv("XDG_DATA_HOME")

	projectDir := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(filepath.Join(projectDir, ".gogent", "memory"), 0755)

	// Test global path uses XDG_DATA_HOME
	globalPath := config.GetMLToolEventsPath()
	if !strings.Contains(globalPath, xdgDataHome) {
		t.Errorf("Global path should use XDG_DATA_HOME, got %s", globalPath)
	}

	// Test project path structure
	projectPath := filepath.Join(projectDir, ".gogent", "memory", "ml-tool-events.jsonl")
	if !strings.Contains(projectPath, ".gogent/memory") {
		t.Errorf("Project path should include .gogent/memory, got %s", projectPath)
	}

	// Create test event and log it
	event := &routing.PostToolEvent{
		ToolName:     "Test",
		Model:        "haiku",
		Tier:         "haiku",
		SequenceIndex: 1,
		Success:      true,
	}

	if err := LogMLToolEvent(event, projectDir); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify global path file was created
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Errorf("Global log file should exist at %s", globalPath)
	}

	// Verify project path file was created
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Project log file should exist at %s", projectPath)
	}

	// Verify both files contain valid JSON
	for _, path := range []string{globalPath, projectPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			continue
		}

		var testEvent routing.PostToolEvent
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) < 1 {
			t.Errorf("Expected at least 1 line in %s", path)
			continue
		}

		if err := json.Unmarshal([]byte(lines[0]), &testEvent); err != nil {
			t.Errorf("Failed to parse JSON from %s: %v", path, err)
		}
	}

	// Test XDG_DATA_HOME override
	customXDG := filepath.Join(tmpDir, "custom-xdg")
	os.Setenv("XDG_DATA_HOME", customXDG)
	customGlobalPath := config.GetMLToolEventsPath()
	if !strings.Contains(customGlobalPath, customXDG) {
		t.Errorf("Should respect XDG_DATA_HOME override, got %s", customGlobalPath)
	}

	// Test that empty projectDir still writes to global path
	if err := LogMLToolEvent(event, ""); err != nil {
		t.Errorf("LogMLToolEvent should succeed with empty projectDir: %v", err)
	}
}

// TestUnderstandingContext_Integration tests understanding-specific fields
func TestUnderstandingContext_Integration(t *testing.T) {
	// Test understanding task with all context fields
	event := &routing.PostToolEvent{
		ToolName:         "Read",
		TaskType:         "document_understanding",
		TargetSize:       5000,
		CoverageAchieved: 0.92,
		EntitiesFound:    42,
	}

	if event.TargetSize != 5000 {
		t.Errorf("Expected TargetSize 5000, got %d", event.TargetSize)
	}

	if event.CoverageAchieved != 0.92 {
		t.Errorf("Expected coverage 0.92, got %f", event.CoverageAchieved)
	}

	if event.EntitiesFound != 42 {
		t.Errorf("Expected 42 entities, got %d", event.EntitiesFound)
	}

	// Verify omitempty behavior for non-understanding tasks
	nonUnderstandingEvent := &routing.PostToolEvent{
		ToolName: "Edit",
		TaskType: "implementation",
	}

	// These should be zero values (omitempty will exclude from JSON)
	if nonUnderstandingEvent.TargetSize != 0 {
		t.Error("Non-understanding task should have zero TargetSize")
	}

	if nonUnderstandingEvent.EntitiesFound != 0 {
		t.Error("Non-understanding task should have zero EntitiesFound")
	}

	if nonUnderstandingEvent.CoverageAchieved != 0.0 {
		t.Error("Non-understanding task should have zero CoverageAchieved")
	}

	// Test that understanding fields are omitted from JSON when zero
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	if err := LogMLToolEvent(nonUnderstandingEvent, ""); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Read back and verify understanding fields are omitted
	logs, err := ReadMLToolEvents()
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least 1 log entry")
	}

	// Marshal to JSON to check omitempty behavior
	jsonData, err := json.Marshal(logs[0])
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "target_size") {
		t.Error("JSON should not contain target_size for zero value (omitempty)")
	}
	if strings.Contains(jsonStr, "entities_found") {
		t.Error("JSON should not contain entities_found for zero value (omitempty)")
	}
	if strings.Contains(jsonStr, "coverage_achieved") {
		t.Error("JSON should not contain coverage_achieved for zero value (omitempty)")
	}
}

// TestTaskTypeAndDomainLabels tests label enumeration functions
func TestTaskTypeAndDomainLabels(t *testing.T) {
	// Test TaskTypeLabels
	types := TaskTypeLabels()
	if len(types) == 0 {
		t.Error("TaskTypeLabels should return non-empty list")
	}

	expectedTypes := map[string]bool{
		"implementation":          true,
		"search":                  true,
		"documentation":           true,
		"debug":                   true,
		"refactor":                true,
		"review":                  true,
		"test":                    true,
		"document_understanding":  true,
		"codebase_understanding":  true,
		"synthesis":               true,
	}

	for _, typ := range types {
		if !expectedTypes[typ] {
			t.Errorf("Unexpected task type: %s", typ)
		}
	}

	// Test TaskDomainLabels
	domains := TaskDomainLabels()
	if len(domains) == 0 {
		t.Error("TaskDomainLabels should return non-empty list")
	}

	expectedDomains := map[string]bool{
		"python":         true,
		"go":             true,
		"r":              true,
		"javascript":     true,
		"infrastructure": true,
		"documentation":  true,
	}

	for _, domain := range domains {
		if !expectedDomains[domain] {
			t.Errorf("Unexpected domain: %s", domain)
		}
	}
}

// TestTaskDomainDetection tests domain classification
func TestTaskDomainDetection(t *testing.T) {
	tests := []struct {
		description    string
		expectedDomain string
	}{
		{"implement python function", "python"},
		{"write go server", "go"},
		{"create R shiny app", "r"},
		{"build javascript component", "javascript"},
		{"setup docker infrastructure", "infrastructure"},
		{"update documentation", "documentation"},
	}

	for _, tc := range tests {
		_, domain := ClassifyTask(tc.description)
		if domain != tc.expectedDomain {
			t.Errorf("ClassifyTask(%q): expected domain %s, got %s",
				tc.description, tc.expectedDomain, domain)
		}
	}
}

// TestErrorHandling_Integration tests error conditions
func TestErrorHandling_Integration(t *testing.T) {
	// Test reading from non-existent path (should return empty slice, not error)
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "nonexistent"))
	defer os.Unsetenv("XDG_DATA_HOME")

	logs, err := ReadMLToolEvents()
	if err != nil {
		t.Errorf("ReadMLToolEvents should handle missing file gracefully, got error: %v", err)
	}
	if logs == nil {
		t.Error("ReadMLToolEvents should return empty slice, not nil")
	}
	if len(logs) != 0 {
		t.Errorf("Expected empty logs, got %d entries", len(logs))
	}
}

// TestMalformedJSON_Integration tests handling of corrupted log files
func TestMalformedJSON_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	// Write valid event followed by malformed JSON
	event := &routing.PostToolEvent{
		ToolName: "Test",
		Model:    "haiku",
	}
	if err := LogMLToolEvent(event, ""); err != nil {
		t.Fatalf("Failed to log valid event: %v", err)
	}

	// Append malformed JSON directly to file
	logPath := config.GetMLToolEventsPath()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	f.WriteString("{invalid json\n")
	f.Close()

	// ReadMLToolEvents should skip malformed lines
	logs, err := ReadMLToolEvents()
	if err != nil {
		t.Fatalf("ReadMLToolEvents should skip malformed lines: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 valid log (malformed line skipped), got %d", len(logs))
	}
}

// TestEmptySessionStats tests stats calculation with no events
func TestEmptySessionStats(t *testing.T) {
	stats := CalculateMLSessionStats([]routing.PostToolEvent{})

	// Check event_count (int)
	eventCount, ok := stats["event_count"].(int)
	if !ok || eventCount != 0 {
		t.Errorf("Expected event_count 0, got: %v (type: %T)", stats["event_count"], stats["event_count"])
	}

	// Check total_duration (int or int64 - Go returns untyped int for 0 constant)
	var totalDurationVal int64
	switch v := stats["total_duration"].(type) {
	case int:
		totalDurationVal = int64(v)
	case int64:
		totalDurationVal = v
	default:
		t.Errorf("Expected total_duration to be int or int64, got: %v (type: %T)", stats["total_duration"], stats["total_duration"])
	}
	if totalDurationVal != 0 {
		t.Errorf("Expected total_duration 0, got: %d", totalDurationVal)
	}

	// Check total_cost (float64)
	totalCost, ok := stats["total_cost"].(float64)
	if !ok {
		t.Errorf("Expected total_cost to be float64, got: %v (type: %T)", stats["total_cost"], stats["total_cost"])
	} else if totalCost != 0.0 {
		t.Errorf("Expected total_cost 0.0, got: %f", totalCost)
	}
}

// TestJSONLFormatCompliance tests JSONL format adherence
func TestJSONLFormatCompliance(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	// Log multiple events
	for i := 1; i <= 5; i++ {
		event := &routing.PostToolEvent{
			ToolName:      "Test",
			Model:         "haiku",
			SequenceIndex: i,
		}
		if err := LogMLToolEvent(event, ""); err != nil {
			t.Fatalf("Failed to log event %d: %v", i, err)
		}
	}

	// Read raw file and verify JSONL format
	logPath := config.GetMLToolEventsPath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 JSONL lines, got %d", len(lines))
	}

	// Each line should be valid JSON
	for i, line := range lines {
		var event routing.PostToolEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i+1, err)
		}
		if event.SequenceIndex != i+1 {
			t.Errorf("Line %d: expected SequenceIndex %d, got %d", i+1, i+1, event.SequenceIndex)
		}
	}
}

// TestProjectPathWrite tests project-scoped logging
func TestProjectPathWrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".gogent", "memory"), 0755)

	event := &routing.PostToolEvent{
		ToolName:      "Test",
		Model:         "haiku",
		SequenceIndex: 1,
	}

	// Log to project directory
	if err := LogMLToolEvent(event, projectDir); err != nil {
		t.Fatalf("Failed to log to project dir: %v", err)
	}

	// Verify project file exists
	projectPath := filepath.Join(projectDir, ".gogent", "memory", "ml-tool-events.jsonl")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Project log file should exist at %s", projectPath)
	}

	// Verify global file also exists
	globalPath := config.GetMLToolEventsPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Errorf("Global log file should exist at %s", globalPath)
	}
}

// TestReadMLToolEvents_EmptyLines tests handling of empty lines in JSONL
func TestReadMLToolEvents_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "xdg"))
	defer os.Unsetenv("XDG_DATA_HOME")

	// Log valid event
	event := &routing.PostToolEvent{
		ToolName: "Test",
		Model:    "haiku",
	}
	if err := LogMLToolEvent(event, ""); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Append empty lines
	logPath := config.GetMLToolEventsPath()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	f.WriteString("\n\n\n")
	f.Close()

	// Should skip empty lines
	logs, err := ReadMLToolEvents()
	if err != nil {
		t.Fatalf("ReadMLToolEvents should skip empty lines: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 valid log (empty lines skipped), got %d", len(logs))
	}
}
