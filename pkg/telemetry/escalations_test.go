package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ===== LogEscalation Tests =====

func TestLogEscalation_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		SessionID:    "test-session",
		EscalationID: "esc-001",
		FromTier:     "sonnet",
		ToTier:       "opus",
		FromAgent:    "orchestrator",
		ToAgent:      "einstein",
		Reason:       "3 consecutive failures on validation",
		TriggerType:  "failure_cascade",
	}

	id, err := LogEscalation(esc, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if id != "esc-001" {
		t.Errorf("Expected ID 'esc-001', got: %s", id)
	}

	// Verify file created
	path := GetEscalationsLogPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Expected escalations log to exist")
	}

	// Verify content
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "failure_cascade") {
		t.Error("Expected log to contain trigger type")
	}
	if !strings.Contains(string(data), `"outcome":"pending"`) {
		t.Error("Expected default outcome 'pending'")
	}
}

func TestLogEscalation_InvalidTriggerType(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		EscalationID: "esc-bad",
		TriggerType:  "invalid_type",
	}

	_, err := LogEscalation(esc, "")
	if err == nil {
		t.Error("Expected error for invalid trigger type")
	}
	if !strings.Contains(err.Error(), "Invalid trigger type") {
		t.Errorf("Expected trigger type error, got: %v", err)
	}
}

func TestLogEscalation_ValidTriggerTypes(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	validTypes := []string{"failure_cascade", "user_request", "complexity", "timeout", "cost_ceiling"}

	for _, triggerType := range validTypes {
		esc := &EscalationEvent{
			EscalationID: "esc-" + triggerType,
			TriggerType:  triggerType,
		}

		_, err := LogEscalation(esc, "")
		if err != nil {
			t.Errorf("Expected no error for valid trigger type '%s', got: %v", triggerType, err)
		}
	}
}

func TestLogEscalation_EmptyTriggerType(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		EscalationID: "esc-empty",
		TriggerType:  "", // Empty should be allowed
	}

	_, err := LogEscalation(esc, "")
	if err != nil {
		t.Errorf("Expected no error for empty trigger type, got: %v", err)
	}
}

func TestLogEscalation_TimestampAutoPopulated(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		EscalationID: "esc-timestamp",
		TriggerType:  "user_request",
	}

	// Timestamp should be empty before logging
	if esc.Timestamp != "" {
		t.Error("Expected empty timestamp before logging")
	}

	_, err := LogEscalation(esc, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Timestamp should be populated after logging
	if esc.Timestamp == "" {
		t.Error("Expected timestamp to be populated after logging")
	}

	// Verify RFC3339 format
	if !strings.Contains(esc.Timestamp, "T") || !strings.Contains(esc.Timestamp, ":") {
		t.Errorf("Expected RFC3339 format, got: %s", esc.Timestamp)
	}

	// Verify timestamp can be parsed
	_, err = time.Parse(time.RFC3339, esc.Timestamp)
	if err != nil {
		t.Errorf("Timestamp not valid RFC3339: %v", err)
	}
}

func TestLogEscalation_DualWrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	projectDir := filepath.Join(tmpDir, "test-project")

	esc := &EscalationEvent{
		SessionID:    "dual-write-test",
		EscalationID: "esc-dual",
		FromTier:     "sonnet",
		ToTier:       "opus",
		TriggerType:  "complexity",
	}

	_, err := LogEscalation(esc, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify both files exist
	globalPath := GetEscalationsLogPath()
	projectPath := GetProjectEscalationsLogPath(projectDir)

	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Expected global log to exist")
	}
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Expected project log to exist")
	}

	// Verify project directory was populated
	data, _ := os.ReadFile(projectPath)
	if !strings.Contains(string(data), projectDir) {
		t.Errorf("Expected project log to contain project_dir")
	}
}

func TestLogEscalation_ProjectWriteFailure_GracefulDegradation(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	// Create a non-writable project directory
	projectDir := filepath.Join(tmpDir, "readonly-project")
	claudeMemDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(claudeMemDir, 0755)

	// Create the escalations file as a directory (will cause write failure)
	escalationsPath := filepath.Join(claudeMemDir, "escalations.jsonl")
	os.Mkdir(escalationsPath, 0755)

	esc := &EscalationEvent{
		SessionID:    "graceful-degradation-test",
		EscalationID: "esc-graceful",
		TriggerType:  "timeout",
	}

	// Should NOT return error - global write should succeed
	_, err := LogEscalation(esc, projectDir)
	if err != nil {
		t.Errorf("Expected no error (graceful degradation), got: %v", err)
	}

	// Verify global log was written
	globalPath := GetEscalationsLogPath()
	data, _ := os.ReadFile(globalPath)
	if !strings.Contains(string(data), "esc-graceful") {
		t.Error("Expected global log to contain escalation despite project write failure")
	}
}

func TestLogEscalation_WithGAPDoc(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	gapDocPath := "/tmp/einstein-gap-001.md"

	esc := &EscalationEvent{
		SessionID:    "gap-doc-test",
		EscalationID: "esc-gap",
		FromTier:     "sonnet",
		ToTier:       "opus",
		TriggerType:  "complexity",
		GAPDocPath:   gapDocPath,
	}

	_, err := LogEscalation(esc, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify GAP doc path was logged
	data, _ := os.ReadFile(GetEscalationsLogPath())
	if !strings.Contains(string(data), gapDocPath) {
		t.Error("Expected log to contain gap_doc_path")
	}
}

func TestLogEscalation_WithCustomOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		EscalationID: "esc-custom-outcome",
		TriggerType:  "user_request",
		Outcome:      "resolved", // Custom outcome
	}

	_, err := LogEscalation(esc, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify custom outcome preserved
	data, _ := os.ReadFile(GetEscalationsLogPath())
	if !strings.Contains(string(data), `"outcome":"resolved"`) {
		t.Error("Expected custom outcome to be preserved")
	}
}

// ===== LoadEscalations Tests =====

func TestLoadEscalations_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","escalation_id":"e1","from_tier":"sonnet","to_tier":"opus","outcome":"resolved"}
{"timestamp":"2026-01-22T11:00:00Z","escalation_id":"e2","from_tier":"sonnet","to_tier":"opus","outcome":"pending"}`

	os.WriteFile(path, []byte(content), 0644)

	escalations, err := LoadEscalations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(escalations) != 2 {
		t.Errorf("Expected 2 escalations, got: %d", len(escalations))
	}

	if escalations[0].EscalationID != "e1" {
		t.Errorf("Expected first ID 'e1', got: %s", escalations[0].EscalationID)
	}
	if escalations[1].EscalationID != "e2" {
		t.Errorf("Expected second ID 'e2', got: %s", escalations[1].EscalationID)
	}
}

func TestLoadEscalations_MissingFile(t *testing.T) {
	escalations, err := LoadEscalations("/nonexistent.jsonl")
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(escalations) != 0 {
		t.Errorf("Expected empty slice, got: %d", len(escalations))
	}
}

func TestLoadEscalations_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","escalation_id":"e1","outcome":"resolved"}
invalid json line
{"timestamp":"2026-01-22T11:00:00Z","escalation_id":"e2","outcome":"pending"}`

	os.WriteFile(path, []byte(content), 0644)

	escalations, err := LoadEscalations(path)
	if err != nil {
		t.Fatalf("Expected no error (malformed skipped), got: %v", err)
	}

	if len(escalations) != 2 {
		t.Errorf("Expected 2 valid escalations (skipping malformed), got: %d", len(escalations))
	}
}

func TestLoadEscalations_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","escalation_id":"e1","outcome":"resolved"}


{"timestamp":"2026-01-22T11:00:00Z","escalation_id":"e2","outcome":"pending"}
`

	os.WriteFile(path, []byte(content), 0644)

	escalations, err := LoadEscalations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(escalations) != 2 {
		t.Errorf("Expected 2 escalations (empty lines skipped), got: %d", len(escalations))
	}
}

func TestLoadEscalations_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")

	os.WriteFile(path, []byte(""), 0644)

	escalations, err := LoadEscalations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(escalations) != 0 {
		t.Errorf("Expected 0 escalations for empty file, got: %d", len(escalations))
	}
}

// ===== UpdateEscalationOutcome Tests =====

func TestUpdateEscalationOutcome_Success(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","escalation_id":"e1","outcome":"pending"}
{"timestamp":"2026-01-22T11:00:00Z","escalation_id":"e2","outcome":"pending"}`
	os.WriteFile(path, []byte(content), 0644)

	err := UpdateEscalationOutcome(path, "e1", "resolved", 5000, "Fixed by adjusting approach")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Reload and verify
	escalations, _ := LoadEscalations(path)
	found := false
	for _, esc := range escalations {
		if esc.EscalationID == "e1" {
			found = true
			if esc.Outcome != "resolved" {
				t.Errorf("Expected outcome 'resolved', got: %s", esc.Outcome)
			}
			if esc.ResolutionTimeMs != 5000 {
				t.Errorf("Expected resolution time 5000, got: %d", esc.ResolutionTimeMs)
			}
			if esc.ResolutionSummary != "Fixed by adjusting approach" {
				t.Errorf("Expected resolution summary, got: %s", esc.ResolutionSummary)
			}
		}
	}

	if !found {
		t.Error("Expected to find updated escalation e1")
	}
}

func TestUpdateEscalationOutcome_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")
	os.WriteFile(path, []byte(`{"escalation_id":"e1","outcome":"pending"}`), 0644)

	err := UpdateEscalationOutcome(path, "nonexistent", "resolved", 0, "")
	if err == nil {
		t.Error("Expected error for nonexistent escalation")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestUpdateEscalationOutcome_InvalidOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")
	os.WriteFile(path, []byte(`{"escalation_id":"e1","outcome":"pending"}`), 0644)

	err := UpdateEscalationOutcome(path, "e1", "invalid_outcome", 0, "")
	if err == nil {
		t.Error("Expected error for invalid outcome")
	}
	if !strings.Contains(err.Error(), "Invalid outcome") {
		t.Errorf("Expected 'Invalid outcome' error, got: %v", err)
	}
}

func TestUpdateEscalationOutcome_ValidOutcomes(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	validOutcomes := []string{"pending", "resolved", "still_blocked", "cancelled"}

	for _, outcome := range validOutcomes {
		content := `{"escalation_id":"e1","outcome":"pending"}`
		os.WriteFile(path, []byte(content), 0644)

		err := UpdateEscalationOutcome(path, "e1", outcome, 1000, "test")
		if err != nil {
			t.Errorf("Expected no error for valid outcome '%s', got: %v", outcome, err)
		}
	}
}

func TestUpdateEscalationOutcome_PreservesOtherEscalations(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "escalations.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","escalation_id":"e1","outcome":"pending"}
{"timestamp":"2026-01-22T11:00:00Z","escalation_id":"e2","outcome":"pending"}
{"timestamp":"2026-01-22T12:00:00Z","escalation_id":"e3","outcome":"resolved"}`
	os.WriteFile(path, []byte(content), 0644)

	// Update e2
	err := UpdateEscalationOutcome(path, "e2", "resolved", 3000, "Fixed")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all escalations still exist
	escalations, _ := LoadEscalations(path)
	if len(escalations) != 3 {
		t.Errorf("Expected 3 escalations preserved, got: %d", len(escalations))
	}

	// Verify e1 unchanged
	for _, esc := range escalations {
		if esc.EscalationID == "e1" {
			if esc.Outcome != "pending" {
				t.Errorf("Expected e1 outcome unchanged, got: %s", esc.Outcome)
			}
		}
		if esc.EscalationID == "e2" {
			if esc.Outcome != "resolved" {
				t.Errorf("Expected e2 outcome resolved, got: %s", esc.Outcome)
			}
		}
		if esc.EscalationID == "e3" {
			if esc.Outcome != "resolved" {
				t.Errorf("Expected e3 outcome preserved, got: %s", esc.Outcome)
			}
		}
	}
}

// ===== FilterEscalations Tests =====

func TestFilterEscalations_ByOutcome(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", Outcome: "resolved"},
		{EscalationID: "e2", Outcome: "pending"},
		{EscalationID: "e3", Outcome: "resolved"},
	}

	outcome := "resolved"
	filtered := FilterEscalations(escalations, EscalationFilters{Outcome: &outcome})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 resolved escalations, got: %d", len(filtered))
	}

	for _, esc := range filtered {
		if esc.Outcome != "resolved" {
			t.Errorf("Expected only resolved escalations, got: %s", esc.Outcome)
		}
	}
}

func TestFilterEscalations_ByTriggerType(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", TriggerType: "failure_cascade"},
		{EscalationID: "e2", TriggerType: "user_request"},
		{EscalationID: "e3", TriggerType: "failure_cascade"},
	}

	trigger := "failure_cascade"
	filtered := FilterEscalations(escalations, EscalationFilters{TriggerType: &trigger})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 failure_cascade escalations, got: %d", len(filtered))
	}

	for _, esc := range filtered {
		if esc.TriggerType != "failure_cascade" {
			t.Errorf("Expected only failure_cascade, got: %s", esc.TriggerType)
		}
	}
}

func TestFilterEscalations_ByToTier(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", ToTier: "opus"},
		{EscalationID: "e2", ToTier: "gemini"},
		{EscalationID: "e3", ToTier: "opus"},
	}

	toTier := "opus"
	filtered := FilterEscalations(escalations, EscalationFilters{ToTier: &toTier})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 opus escalations, got: %d", len(filtered))
	}
}

func TestFilterEscalations_ByFromAgent(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", FromAgent: "orchestrator"},
		{EscalationID: "e2", FromAgent: "python-pro"},
		{EscalationID: "e3", FromAgent: "orchestrator"},
	}

	fromAgent := "orchestrator"
	filtered := FilterEscalations(escalations, EscalationFilters{FromAgent: &fromAgent})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 orchestrator escalations, got: %d", len(filtered))
	}
}

func TestFilterEscalations_BySince(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	escalations := []EscalationEvent{
		{EscalationID: "e1", Timestamp: past.Format(time.RFC3339)},
		{EscalationID: "e2", Timestamp: now.Format(time.RFC3339)},
		{EscalationID: "e3", Timestamp: past.Add(-2 * time.Hour).Format(time.RFC3339)},
	}

	since := past.Unix()
	filtered := FilterEscalations(escalations, EscalationFilters{Since: &since})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 escalations since filter, got: %d", len(filtered))
	}
}

func TestFilterEscalations_WithLimit(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1"},
		{EscalationID: "e2"},
		{EscalationID: "e3"},
		{EscalationID: "e4"},
	}

	filtered := FilterEscalations(escalations, EscalationFilters{Limit: 2})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 escalations with limit, got: %d", len(filtered))
	}
}

func TestFilterEscalations_MultipleFilters(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", Outcome: "resolved", TriggerType: "failure_cascade", ToTier: "opus"},
		{EscalationID: "e2", Outcome: "pending", TriggerType: "failure_cascade", ToTier: "opus"},
		{EscalationID: "e3", Outcome: "resolved", TriggerType: "user_request", ToTier: "opus"},
		{EscalationID: "e4", Outcome: "resolved", TriggerType: "failure_cascade", ToTier: "gemini"},
	}

	outcome := "resolved"
	trigger := "failure_cascade"
	toTier := "opus"

	filtered := FilterEscalations(escalations, EscalationFilters{
		Outcome:     &outcome,
		TriggerType: &trigger,
		ToTier:      &toTier,
	})

	if len(filtered) != 1 {
		t.Errorf("Expected 1 escalation matching all filters, got: %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].EscalationID != "e1" {
		t.Errorf("Expected e1, got: %s", filtered[0].EscalationID)
	}
}

func TestFilterEscalations_NoMatches(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1", Outcome: "resolved"},
		{EscalationID: "e2", Outcome: "resolved"},
	}

	outcome := "pending"
	filtered := FilterEscalations(escalations, EscalationFilters{Outcome: &outcome})

	if len(filtered) != 0 {
		t.Errorf("Expected 0 escalations, got: %d", len(filtered))
	}
}

func TestFilterEscalations_EmptyInput(t *testing.T) {
	filtered := FilterEscalations([]EscalationEvent{}, EscalationFilters{})

	if len(filtered) != 0 {
		t.Errorf("Expected 0 escalations for empty input, got: %d", len(filtered))
	}
}

func TestFilterEscalations_NoFiltersApplied(t *testing.T) {
	escalations := []EscalationEvent{
		{EscalationID: "e1"},
		{EscalationID: "e2"},
		{EscalationID: "e3"},
	}

	filtered := FilterEscalations(escalations, EscalationFilters{})

	if len(filtered) != 3 {
		t.Errorf("Expected all 3 escalations with no filters, got: %d", len(filtered))
	}
}

// ===== Path Helper Tests =====

func TestGetEscalationsLogPath_XDGCompliance(t *testing.T) {
	// Clear XDG_RUNTIME_DIR to test XDG_CACHE_HOME priority
	tmpDir := t.TempDir()
	originalRuntime := os.Getenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer func() {
		os.Unsetenv("XDG_CACHE_HOME")
		if originalRuntime != "" {
			os.Setenv("XDG_RUNTIME_DIR", originalRuntime)
		}
	}()

	path := GetEscalationsLogPath()
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("Expected path to start with XDG_CACHE_HOME, got: %s", path)
	}
	if !strings.HasSuffix(path, "escalations.jsonl") {
		t.Errorf("Expected path to end with escalations.jsonl, got: %s", path)
	}
}

func TestGetProjectEscalationsLogPath(t *testing.T) {
	projectDir := "/home/user/my-project"
	path := GetProjectEscalationsLogPath(projectDir)

	expectedPath := filepath.Join(projectDir, ".claude", "memory", "escalations.jsonl")
	if path != expectedPath {
		t.Errorf("Expected path '%s', got: '%s'", expectedPath, path)
	}
}

// ===== Edge Case Tests =====

func TestLogEscalation_DirectoryAutoCreated(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	// Use a nested project directory that doesn't exist
	projectDir := filepath.Join(tmpDir, "deeply", "nested", "project")

	esc := &EscalationEvent{
		SessionID:    "dir-creation-test",
		EscalationID: "esc-nested",
		TriggerType:  "complexity",
	}

	_, err := LogEscalation(esc, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify directories were created
	projectPath := GetProjectEscalationsLogPath(projectDir)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Expected project log directory to be auto-created")
	}
}

func TestLogEscalation_MultipleEscalations(t *testing.T) {
	tmpDir := t.TempDir()
	// Clear XDG_RUNTIME_DIR to ensure test isolation
	originalRuntime := os.Getenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer func() {
		os.Unsetenv("XDG_CACHE_HOME")
		if originalRuntime != "" {
			os.Setenv("XDG_RUNTIME_DIR", originalRuntime)
		}
	}()

	escalations := []*EscalationEvent{
		{SessionID: "multi-test", EscalationID: "e1", TriggerType: "failure_cascade"},
		{SessionID: "multi-test", EscalationID: "e2", TriggerType: "user_request"},
		{SessionID: "multi-test", EscalationID: "e3", TriggerType: "complexity"},
	}

	for _, esc := range escalations {
		if _, err := LogEscalation(esc, ""); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	}

	// Load and verify all were written
	loaded, err := LoadEscalations(GetEscalationsLogPath())
	if err != nil {
		t.Fatalf("Failed to load escalations: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 escalations, got: %d", len(loaded))
	}

	// Verify order (oldest first)
	if loaded[0].EscalationID != "e1" {
		t.Errorf("Expected first escalation 'e1', got: %s", loaded[0].EscalationID)
	}
	if loaded[2].EscalationID != "e3" {
		t.Errorf("Expected third escalation 'e3', got: %s", loaded[2].EscalationID)
	}
}

func TestLogEscalation_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	esc := &EscalationEvent{
		SessionID:         "full-fields-test",
		EscalationID:      "esc-full",
		FromTier:          "sonnet",
		ToTier:            "opus",
		FromAgent:         "orchestrator",
		ToAgent:           "einstein",
		Reason:            "Complex architectural decision required",
		TriggerType:       "complexity",
		GAPDocPath:        "/tmp/gap-001.md",
		Outcome:           "pending",
		ResolutionTimeMs:  0,
		ResolutionSummary: "",
		TokensUsed:        0,
	}

	_, err := LogEscalation(esc, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all fields present
	data, _ := os.ReadFile(GetEscalationsLogPath())
	content := string(data)

	expectedFields := []string{
		"session_id", "escalation_id", "from_tier", "to_tier",
		"from_agent", "to_agent", "reason", "trigger_type",
		"gap_doc_path", "outcome", "project_dir",
	}

	for _, field := range expectedFields {
		if !strings.Contains(content, field) {
			t.Errorf("Expected log to contain field '%s'", field)
		}
	}
}

// ===== PATTERN ANALYSIS TESTS =====

func TestClusterEscalationsByFromAgent(t *testing.T) {
	tests := []struct {
		name       string
		escalations []EscalationEvent
		wantCount  int
		verify     func(t *testing.T, clusters map[string]*AgentEscalationStats)
	}{
		{
			name: "multiple agents with different outcomes",
			escalations: []EscalationEvent{
				{FromAgent: "orchestrator", Outcome: "resolved", TriggerType: "failure_cascade"},
				{FromAgent: "orchestrator", Outcome: "resolved", TriggerType: "complexity"},
				{FromAgent: "python-pro", Outcome: "still_blocked", TriggerType: "failure_cascade"},
			},
			wantCount: 2,
			verify: func(t *testing.T, clusters map[string]*AgentEscalationStats) {
				orchStats := clusters["orchestrator"]
				if orchStats.TotalCount != 2 {
					t.Errorf("Expected orchestrator count 2, got: %d", orchStats.TotalCount)
				}
				if orchStats.ResolvedCount != 2 {
					t.Errorf("Expected orchestrator resolved 2, got: %d", orchStats.ResolvedCount)
				}
				if orchStats.ResolutionRate != 1.0 {
					t.Errorf("Expected orchestrator resolution rate 1.0, got: %f", orchStats.ResolutionRate)
				}

				pythonStats := clusters["python-pro"]
				if pythonStats.ResolutionRate != 0.0 {
					t.Errorf("Expected python-pro resolution rate 0.0, got: %f", pythonStats.ResolutionRate)
				}
			},
		},
		{
			name:        "empty input",
			escalations: []EscalationEvent{},
			wantCount:   0,
		},
		{
			name: "unknown agent handling",
			escalations: []EscalationEvent{
				{FromAgent: "", Outcome: "resolved", TriggerType: "user_request"},
			},
			wantCount: 1,
			verify: func(t *testing.T, clusters map[string]*AgentEscalationStats) {
				if _, exists := clusters["unknown"]; !exists {
					t.Error("Expected 'unknown' cluster for empty FromAgent")
				}
			},
		},
		{
			name: "trigger breakdown",
			escalations: []EscalationEvent{
				{FromAgent: "orchestrator", TriggerType: "failure_cascade"},
				{FromAgent: "orchestrator", TriggerType: "complexity"},
				{FromAgent: "orchestrator", TriggerType: "failure_cascade"},
			},
			wantCount: 1,
			verify: func(t *testing.T, clusters map[string]*AgentEscalationStats) {
				orchStats := clusters["orchestrator"]
				if orchStats.TriggerBreakdown["failure_cascade"] != 2 {
					t.Errorf("Expected 2 failure_cascade, got: %d", orchStats.TriggerBreakdown["failure_cascade"])
				}
				if orchStats.TriggerBreakdown["complexity"] != 1 {
					t.Errorf("Expected 1 complexity, got: %d", orchStats.TriggerBreakdown["complexity"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters := ClusterEscalationsByFromAgent(tt.escalations)

			if len(clusters) != tt.wantCount {
				t.Errorf("Expected %d clusters, got: %d", tt.wantCount, len(clusters))
			}

			if tt.verify != nil {
				tt.verify(t, clusters)
			}
		})
	}
}

func TestClusterEscalationsByTrigger(t *testing.T) {
	tests := []struct {
		name       string
		escalations []EscalationEvent
		wantCount  int
		verify     func(t *testing.T, clusters map[string]*TriggerEscalationStats)
	}{
		{
			name: "multiple triggers with mixed outcomes",
			escalations: []EscalationEvent{
				{TriggerType: "failure_cascade", Outcome: "resolved"},
				{TriggerType: "failure_cascade", Outcome: "still_blocked"},
				{TriggerType: "user_request", Outcome: "resolved"},
			},
			wantCount: 2,
			verify: func(t *testing.T, clusters map[string]*TriggerEscalationStats) {
				cascadeStats := clusters["failure_cascade"]
				if cascadeStats.TotalCount != 2 {
					t.Errorf("Expected failure_cascade count 2, got: %d", cascadeStats.TotalCount)
				}
				// Resolution rate: 1 resolved / 2 completed = 0.5
				if cascadeStats.ResolutionRate < 0.49 || cascadeStats.ResolutionRate > 0.51 {
					t.Errorf("Expected resolution rate ~0.5, got: %f", cascadeStats.ResolutionRate)
				}
			},
		},
		{
			name:        "empty input",
			escalations: []EscalationEvent{},
			wantCount:   0,
		},
		{
			name: "unknown trigger handling",
			escalations: []EscalationEvent{
				{TriggerType: "", Outcome: "resolved", FromAgent: "orchestrator"},
			},
			wantCount: 1,
			verify: func(t *testing.T, clusters map[string]*TriggerEscalationStats) {
				if _, exists := clusters["unknown"]; !exists {
					t.Error("Expected 'unknown' cluster for empty TriggerType")
				}
			},
		},
		{
			name: "from agent breakdown",
			escalations: []EscalationEvent{
				{TriggerType: "failure_cascade", FromAgent: "orchestrator"},
				{TriggerType: "failure_cascade", FromAgent: "python-pro"},
				{TriggerType: "failure_cascade", FromAgent: "orchestrator"},
			},
			wantCount: 1,
			verify: func(t *testing.T, clusters map[string]*TriggerEscalationStats) {
				cascadeStats := clusters["failure_cascade"]
				if cascadeStats.FromAgentBreakdown["orchestrator"] != 2 {
					t.Errorf("Expected 2 orchestrator, got: %d", cascadeStats.FromAgentBreakdown["orchestrator"])
				}
				if cascadeStats.FromAgentBreakdown["python-pro"] != 1 {
					t.Errorf("Expected 1 python-pro, got: %d", cascadeStats.FromAgentBreakdown["python-pro"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters := ClusterEscalationsByTrigger(tt.escalations)

			if len(clusters) != tt.wantCount {
				t.Errorf("Expected %d clusters, got: %d", tt.wantCount, len(clusters))
			}

			if tt.verify != nil {
				tt.verify(t, clusters)
			}
		})
	}
}

func TestCalculateEscalationROI(t *testing.T) {
	tests := []struct {
		name        string
		escalations []EscalationEvent
		verify      func(t *testing.T, roi EscalationROI)
	}{
		{
			name: "mixed outcomes",
			escalations: []EscalationEvent{
				{Outcome: "resolved", TokensUsed: 5000},
				{Outcome: "resolved", TokensUsed: 3000},
				{Outcome: "still_blocked", TokensUsed: 4000},
				{Outcome: "pending", TokensUsed: 0},
			},
			verify: func(t *testing.T, roi EscalationROI) {
				if roi.TotalEscalations != 4 {
					t.Errorf("Expected 4 total, got: %d", roi.TotalEscalations)
				}
				if roi.CompletedCount != 3 {
					t.Errorf("Expected 3 completed, got: %d", roi.CompletedCount)
				}
				if roi.ResolvedCount != 2 {
					t.Errorf("Expected 2 resolved, got: %d", roi.ResolvedCount)
				}
				// Resolution rate: 2/3 = 0.667
				if roi.ResolutionRate < 0.66 || roi.ResolutionRate > 0.68 {
					t.Errorf("Expected resolution rate ~0.67, got: %f", roi.ResolutionRate)
				}
				if roi.TotalTokensUsed != 12000 {
					t.Errorf("Expected 12000 tokens, got: %d", roi.TotalTokensUsed)
				}
				if roi.AvgTokensPerEscalation != 3000 {
					t.Errorf("Expected avg 3000 tokens, got: %d", roi.AvgTokensPerEscalation)
				}
			},
		},
		{
			name:        "empty input",
			escalations: []EscalationEvent{},
			verify: func(t *testing.T, roi EscalationROI) {
				if roi.TotalEscalations != 0 {
					t.Errorf("Expected 0 total, got: %d", roi.TotalEscalations)
				}
				if roi.ResolutionRate != 0.0 {
					t.Errorf("Expected 0.0 resolution rate, got: %f", roi.ResolutionRate)
				}
			},
		},
		{
			name: "all resolved",
			escalations: []EscalationEvent{
				{Outcome: "resolved", TokensUsed: 1000},
				{Outcome: "resolved", TokensUsed: 1000},
			},
			verify: func(t *testing.T, roi EscalationROI) {
				if roi.ResolutionRate != 1.0 {
					t.Errorf("Expected 1.0 resolution rate, got: %f", roi.ResolutionRate)
				}
				// 2 * 0.09 = 0.18
				if roi.EstimatedCostSaved < 0.17 || roi.EstimatedCostSaved > 0.19 {
					t.Errorf("Expected cost saved ~0.18, got: %f", roi.EstimatedCostSaved)
				}
			},
		},
		{
			name: "all blocked",
			escalations: []EscalationEvent{
				{Outcome: "still_blocked", TokensUsed: 1000},
				{Outcome: "still_blocked", TokensUsed: 1000},
			},
			verify: func(t *testing.T, roi EscalationROI) {
				if roi.ResolutionRate != 0.0 {
					t.Errorf("Expected 0.0 resolution rate, got: %f", roi.ResolutionRate)
				}
				if roi.EstimatedCostSaved != 0.0 {
					t.Errorf("Expected 0.0 cost saved, got: %f", roi.EstimatedCostSaved)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roi := CalculateEscalationROI(tt.escalations)
			tt.verify(t, roi)
		})
	}
}

func TestAnalyzeEscalationLatency(t *testing.T) {
	tests := []struct {
		name        string
		escalations []EscalationEvent
		verify      func(t *testing.T, stats LatencyStats)
	}{
		{
			name: "odd number of samples",
			escalations: []EscalationEvent{
				{ResolutionTimeMs: 1000},
				{ResolutionTimeMs: 2000},
				{ResolutionTimeMs: 3000},
				{ResolutionTimeMs: 4000},
				{ResolutionTimeMs: 5000},
			},
			verify: func(t *testing.T, stats LatencyStats) {
				if stats.SampleCount != 5 {
					t.Errorf("Expected 5 samples, got: %d", stats.SampleCount)
				}
				if stats.MinMs != 1000 {
					t.Errorf("Expected min 1000, got: %d", stats.MinMs)
				}
				if stats.MaxMs != 5000 {
					t.Errorf("Expected max 5000, got: %d", stats.MaxMs)
				}
				if stats.AvgMs != 3000 {
					t.Errorf("Expected avg 3000, got: %d", stats.AvgMs)
				}
				if stats.MedianMs != 3000 {
					t.Errorf("Expected median 3000, got: %d", stats.MedianMs)
				}
				// P90 of 5 samples = index 4 (last element)
				if stats.P90Ms != 5000 {
					t.Errorf("Expected P90 5000, got: %d", stats.P90Ms)
				}
			},
		},
		{
			name: "even number of samples",
			escalations: []EscalationEvent{
				{ResolutionTimeMs: 1000},
				{ResolutionTimeMs: 2000},
				{ResolutionTimeMs: 3000},
				{ResolutionTimeMs: 4000},
			},
			verify: func(t *testing.T, stats LatencyStats) {
				if stats.SampleCount != 4 {
					t.Errorf("Expected 4 samples, got: %d", stats.SampleCount)
				}
				// Median of [1000, 2000, 3000, 4000] = (2000 + 3000) / 2 = 2500
				if stats.MedianMs != 2500 {
					t.Errorf("Expected median 2500, got: %d", stats.MedianMs)
				}
			},
		},
		{
			name:        "empty input",
			escalations: []EscalationEvent{},
			verify: func(t *testing.T, stats LatencyStats) {
				if stats.SampleCount != 0 {
					t.Errorf("Expected 0 samples, got: %d", stats.SampleCount)
				}
			},
		},
		{
			name: "single sample",
			escalations: []EscalationEvent{
				{ResolutionTimeMs: 5000},
			},
			verify: func(t *testing.T, stats LatencyStats) {
				if stats.SampleCount != 1 {
					t.Errorf("Expected 1 sample, got: %d", stats.SampleCount)
				}
				if stats.MinMs != 5000 || stats.MaxMs != 5000 || stats.MedianMs != 5000 {
					t.Errorf("Expected all stats to be 5000")
				}
			},
		},
		{
			name: "ignores zero latencies",
			escalations: []EscalationEvent{
				{ResolutionTimeMs: 0},     // Ignored
				{ResolutionTimeMs: 1000},
				{ResolutionTimeMs: 0},     // Ignored
				{ResolutionTimeMs: 2000},
			},
			verify: func(t *testing.T, stats LatencyStats) {
				if stats.SampleCount != 2 {
					t.Errorf("Expected 2 samples (ignoring zeros), got: %d", stats.SampleCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := AnalyzeEscalationLatency(tt.escalations)
			tt.verify(t, stats)
		})
	}
}

func TestGetEscalationTrend(t *testing.T) {
	tests := []struct {
		name        string
		escalations []EscalationEvent
		verify      func(t *testing.T, trend EscalationTrend)
	}{
		{
			name: "improving trend",
			escalations: []EscalationEvent{
				{Timestamp: "2026-01-22T10:00:00Z"},
				{Timestamp: "2026-01-22T10:01:00Z"},
				{Timestamp: "2026-01-22T10:02:00Z"},
				{Timestamp: "2026-01-22T11:00:00Z"}, // Only 1 in second half
			},
			verify: func(t *testing.T, trend EscalationTrend) {
				if trend.Trend != "improving" {
					t.Errorf("Expected 'improving', got: %s", trend.Trend)
				}
				if trend.EarlyCount != 3 {
					t.Errorf("Expected early count 3, got: %d", trend.EarlyCount)
				}
				if trend.LateCount != 1 {
					t.Errorf("Expected late count 1, got: %d", trend.LateCount)
				}
				if !strings.Contains(trend.Message, "reduction") {
					t.Errorf("Expected 'reduction' in message, got: %s", trend.Message)
				}
			},
		},
		{
			name: "worsening trend",
			escalations: []EscalationEvent{
				{Timestamp: "2026-01-22T10:00:00Z"},
				{Timestamp: "2026-01-22T10:01:00Z"},
				{Timestamp: "2026-01-22T11:00:00Z"},
				{Timestamp: "2026-01-22T11:01:00Z"},
				{Timestamp: "2026-01-22T11:02:00Z"},
			},
			verify: func(t *testing.T, trend EscalationTrend) {
				if trend.Trend != "worsening" {
					t.Errorf("Expected 'worsening', got: %s", trend.Trend)
				}
				if !strings.Contains(trend.Message, "increase") {
					t.Errorf("Expected 'increase' in message, got: %s", trend.Message)
				}
			},
		},
		{
			name: "stable trend",
			escalations: []EscalationEvent{
				{Timestamp: "2026-01-22T10:00:00Z"},
				{Timestamp: "2026-01-22T10:01:00Z"},
				{Timestamp: "2026-01-22T11:00:00Z"},
				{Timestamp: "2026-01-22T11:01:00Z"},
			},
			verify: func(t *testing.T, trend EscalationTrend) {
				if trend.Trend != "stable" {
					t.Errorf("Expected 'stable', got: %s", trend.Trend)
				}
				if trend.EarlyCount != 2 || trend.LateCount != 2 {
					t.Errorf("Expected equal counts (2,2), got: (%d,%d)", trend.EarlyCount, trend.LateCount)
				}
			},
		},
		{
			name: "insufficient data - single escalation",
			escalations: []EscalationEvent{
				{Timestamp: "2026-01-22T10:00:00Z"},
			},
			verify: func(t *testing.T, trend EscalationTrend) {
				if trend.Trend != "insufficient_data" {
					t.Errorf("Expected 'insufficient_data', got: %s", trend.Trend)
				}
			},
		},
		{
			name:        "insufficient data - empty",
			escalations: []EscalationEvent{},
			verify: func(t *testing.T, trend EscalationTrend) {
				if trend.Trend != "insufficient_data" {
					t.Errorf("Expected 'insufficient_data', got: %s", trend.Trend)
				}
			},
		},
		{
			name: "handles unsorted input",
			escalations: []EscalationEvent{
				{Timestamp: "2026-01-22T11:00:00Z"}, // Out of order
				{Timestamp: "2026-01-22T10:00:00Z"},
				{Timestamp: "2026-01-22T10:01:00Z"},
			},
			verify: func(t *testing.T, trend EscalationTrend) {
				// Should still work by sorting internally
				if trend.Trend == "" {
					t.Error("Expected trend to be calculated despite unsorted input")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := GetEscalationTrend(tt.escalations)
			tt.verify(t, trend)
		})
	}
}
