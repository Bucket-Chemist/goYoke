package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRoutingDecision(t *testing.T) {
	sessionID := "test-session-123"
	taskDesc := "Implement new authentication module"
	tier := "sonnet"
	agent := "python-pro"

	decision := NewRoutingDecision(sessionID, taskDesc, tier, agent)

	if decision.DecisionID == "" {
		t.Error("Expected DecisionID to be set")
	}

	if decision.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, decision.SessionID)
	}

	if decision.TaskDescription != taskDesc {
		t.Errorf("Expected TaskDescription %s, got %s", taskDesc, decision.TaskDescription)
	}

	if decision.SelectedTier != tier {
		t.Errorf("Expected SelectedTier %s, got %s", tier, decision.SelectedTier)
	}

	if decision.SelectedAgent != agent {
		t.Errorf("Expected SelectedAgent %s, got %s", agent, decision.SelectedAgent)
	}

	if decision.Timestamp == 0 {
		t.Error("Expected Timestamp to be set")
	}
}

func TestNewRoutingDecision_TruncatesLongDescription(t *testing.T) {
	longDesc := strings.Repeat("x", 600)
	decision := NewRoutingDecision("session", longDesc, "haiku", "codebase-search")

	if len(decision.TaskDescription) > 503 {
		t.Errorf("Expected description truncated to ~500 chars, got %d", len(decision.TaskDescription))
	}

	if !strings.HasSuffix(decision.TaskDescription, "...") {
		t.Error("Expected truncated description to end with '...'")
	}
}

func TestLogRoutingDecision(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
		Timestamp:     1234567890,
	}

	err := LogRoutingDecision(decision)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created and contains decision
	path := filepath.Join(tmpDir, "goyoke", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be created, got error: %v", err)
	}

	if !strings.Contains(string(data), "decision-001") {
		t.Error("Expected decision ID in file")
	}
}

func TestLogRoutingDecision_MultipleDecisions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision1 := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
	}

	decision2 := &RoutingDecision{
		DecisionID:    "decision-002",
		SessionID:     "session-001",
		SelectedTier:  "sonnet",
		SelectedAgent: "python-pro",
	}

	if err := LogRoutingDecision(decision1); err != nil {
		t.Fatalf("Failed to log first decision: %v", err)
	}

	if err := LogRoutingDecision(decision2); err != nil {
		t.Fatalf("Failed to log second decision: %v", err)
	}

	// Verify both are in file
	path := filepath.Join(tmpDir, "goyoke", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "decision-001") {
		t.Error("Expected first decision in file")
	}
	if !strings.Contains(content, "decision-002") {
		t.Error("Expected second decision in file")
	}
}

func TestUpdateDecisionOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
	}

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// Update outcome
	err := UpdateDecisionOutcome("decision-001", true, 500, 0.0015, false)
	if err != nil {
		t.Fatalf("Expected no error updating outcome, got: %v", err)
	}

	// Verify outcome was appended to updates file
	updatesPath := filepath.Join(tmpDir, "goyoke", "routing-decision-updates.jsonl")
	data, err := os.ReadFile(updatesPath)
	if err != nil {
		t.Fatalf("Failed to read updates file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\"decision_id\":\"decision-001\"") {
		t.Error("Expected decision_id in updates file")
	}
	if !strings.Contains(content, "\"outcome_success\":true") {
		t.Error("Expected outcome_success to be true in update")
	}
}

func TestUpdateDecisionOutcome_MultipleUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID: "decision-001",
		SessionID:  "session-001",
	}

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// First update
	if err := UpdateDecisionOutcome("decision-001", true, 100, 0.001, false); err != nil {
		t.Fatalf("Failed first update: %v", err)
	}

	// Second update
	if err := UpdateDecisionOutcome("decision-001", false, 200, 0.002, true); err != nil {
		t.Fatalf("Failed second update: %v", err)
	}

	// Verify both updates exist in separate lines
	updatesPath := filepath.Join(tmpDir, "goyoke", "routing-decision-updates.jsonl")
	data, err := os.ReadFile(updatesPath)
	if err != nil {
		t.Fatalf("Failed to read updates: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 update lines, got %d", len(lines))
	}
}

func TestUnderstandingQualityFields(t *testing.T) {
	decision := &RoutingDecision{
		DecisionID:                "decision-001",
		UnderstandingCompleteness: 0.95,
		UnderstandingAccuracy:     0.87,
		SynthesisCoherence:        0.92,
		RequiredFollowUp:          true,
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	path := filepath.Join(tmpDir, "goyoke", "routing-decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\"understanding_completeness\":0.95") {
		t.Error("Expected understanding_completeness in output")
	}
	if !strings.Contains(content, "\"synthesis_coherence\":0.92") {
		t.Error("Expected synthesis_coherence in output")
	}
	if !strings.Contains(content, "\"required_follow_up\":true") {
		t.Error("Expected required_follow_up in output")
	}
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{strings.Repeat("x", 100), 50, strings.Repeat("x", 50) + "..."},
		{"exactly50chars" + strings.Repeat("x", 35), 50, "exactly50chars" + strings.Repeat("x", 35)},
	}

	for _, tc := range tests {
		result := truncateDescription(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("For input len %d with maxLen %d: expected %q, got %q", len(tc.input), tc.maxLen, tc.expected, result)
		}
	}
}

func TestNewRoutingDecision_IntegratesClassifyTask(t *testing.T) {
	// Test that NewRoutingDecision correctly uses ClassifyTask
	decision := NewRoutingDecision("session", "implement python authentication", "sonnet", "python-pro")

	if decision.TaskType == "" || decision.TaskType == "unknown" {
		t.Errorf("Expected TaskType to be classified, got %s", decision.TaskType)
	}

	if decision.TaskDomain == "" || decision.TaskDomain == "unknown" {
		t.Errorf("Expected TaskDomain to be classified, got %s", decision.TaskDomain)
	}

	// This specific description should be classified as implementation/python
	if decision.TaskType != "implementation" {
		t.Errorf("Expected TaskType 'implementation', got %s", decision.TaskType)
	}

	if decision.TaskDomain != "python" {
		t.Errorf("Expected TaskDomain 'python', got %s", decision.TaskDomain)
	}
}

func TestLogRoutingDecision_ErrorMessages(t *testing.T) {
	// Test error message format
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Make directory read-only to trigger error
	goyokeDir := filepath.Join(tmpDir, "goyoke")
	if err := os.MkdirAll(goyokeDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file where we expect directory to be
	testFile := filepath.Join(goyokeDir, "routing-decisions.jsonl")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it a directory (which will conflict)
	os.Remove(testFile)
	os.MkdirAll(testFile, 0755)

	decision := &RoutingDecision{
		DecisionID: "test",
		SessionID:  "test",
	}

	err := LogRoutingDecision(decision)
	if err == nil {
		t.Error("Expected error when trying to write to directory")
	}

	if !strings.HasPrefix(err.Error(), "[routing-decision]") {
		t.Errorf("Expected error to start with '[routing-decision]', got: %v", err)
	}
}

func TestDecisionFileSeparation(t *testing.T) {
	// Verify decisions and updates go to separate files
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	decision := &RoutingDecision{
		DecisionID:    "decision-001",
		SessionID:     "session-001",
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
	}

	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	if err := UpdateDecisionOutcome("decision-001", true, 500, 0.001, false); err != nil {
		t.Fatalf("Failed to update outcome: %v", err)
	}

	// Verify separate files exist
	decisionsPath := filepath.Join(tmpDir, "goyoke", "routing-decisions.jsonl")
	updatesPath := filepath.Join(tmpDir, "goyoke", "routing-decision-updates.jsonl")

	if _, err := os.Stat(decisionsPath); os.IsNotExist(err) {
		t.Error("Expected decisions file to exist")
	}

	if _, err := os.Stat(updatesPath); os.IsNotExist(err) {
		t.Error("Expected updates file to exist")
	}

	// Verify decision file does NOT contain outcome data
	decisionsData, _ := os.ReadFile(decisionsPath)
	if strings.Contains(string(decisionsData), "outcome_success") {
		t.Error("Decision file should not contain outcome_success (append-only design)")
	}

	// Verify updates file contains outcome data
	updatesData, _ := os.ReadFile(updatesPath)
	if !strings.Contains(string(updatesData), "outcome_success") {
		t.Error("Updates file should contain outcome_success")
	}
}

func TestXDGDataHomeFallback(t *testing.T) {
	// Test that when XDG_DATA_HOME is not set, it falls back to ~/.local/share
	originalXDG := os.Getenv("XDG_DATA_HOME")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", originalXDG)
		os.Setenv("HOME", originalHome)
	}()

	os.Unsetenv("XDG_DATA_HOME")
	t.Setenv("HOME", t.TempDir())

	path := getRoutingDecisionPath()
	if !strings.Contains(path, ".local/share/goyoke") {
		t.Errorf("Expected path to contain .local/share/goyoke, got %s", path)
	}

	updatesPath := getRoutingDecisionUpdatesPath()
	if !strings.Contains(updatesPath, ".local/share/goyoke") {
		t.Errorf("Expected updates path to contain .local/share/goyoke, got %s", updatesPath)
	}
}

func TestAppendDecisionUpdate_CreatesDirectoryIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// UpdateDecisionOutcome should create directory if it doesn't exist
	err := UpdateDecisionOutcome("decision-001", true, 500, 0.001, false)
	if err != nil {
		t.Fatalf("Expected UpdateDecisionOutcome to create directory, got error: %v", err)
	}

	updatesPath := filepath.Join(tmpDir, "goyoke", "routing-decision-updates.jsonl")
	if _, err := os.Stat(updatesPath); os.IsNotExist(err) {
		t.Error("Expected updates file to be created")
	}
}

func TestCompleteRoutingDecisionWorkflow(t *testing.T) {
	// Test complete workflow from decision creation to outcome update
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Step 1: Create decision
	decision := NewRoutingDecision("session-123", "implement go authentication", "sonnet", "go-pro")
	if decision.TaskType != "implementation" {
		t.Errorf("Expected TaskType 'implementation', got %s", decision.TaskType)
	}
	if decision.TaskDomain != "go" {
		t.Errorf("Expected TaskDomain 'go', got %s", decision.TaskDomain)
	}

	// Step 2: Log decision
	if err := LogRoutingDecision(decision); err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// Step 3: Update outcome
	if err := UpdateDecisionOutcome(decision.DecisionID, true, 1500, 0.0025, false); err != nil {
		t.Fatalf("Failed to update outcome: %v", err)
	}

	// Step 4: Verify both files exist and contain correct data
	decisionsPath := filepath.Join(tmpDir, "goyoke", "routing-decisions.jsonl")
	updatesPath := filepath.Join(tmpDir, "goyoke", "routing-decision-updates.jsonl")

	decisionsData, _ := os.ReadFile(decisionsPath)
	if !strings.Contains(string(decisionsData), decision.DecisionID) {
		t.Error("Expected decision ID in decisions file")
	}
	if !strings.Contains(string(decisionsData), "\"task_type\":\"implementation\"") {
		t.Error("Expected classified task type in decisions file")
	}

	updatesData, _ := os.ReadFile(updatesPath)
	if !strings.Contains(string(updatesData), decision.DecisionID) {
		t.Error("Expected decision ID in updates file")
	}
	if !strings.Contains(string(updatesData), "\"outcome_duration_ms\":1500") {
		t.Error("Expected outcome duration in updates file")
	}
}
