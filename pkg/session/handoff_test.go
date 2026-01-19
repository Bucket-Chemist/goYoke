package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultHandoffConfig(t *testing.T) {
	projectDir := "/tmp/test-project"

	config := DefaultHandoffConfig(projectDir)

	if config.ProjectDir != projectDir {
		t.Errorf("Expected ProjectDir %s, got: %s", projectDir, config.ProjectDir)
	}

	expectedHandoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")
	if config.HandoffPath != expectedHandoffPath {
		t.Errorf("Expected HandoffPath %s, got: %s", expectedHandoffPath, config.HandoffPath)
	}

	expectedPendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")
	if config.PendingPath != expectedPendingPath {
		t.Errorf("Expected PendingPath %s, got: %s", expectedPendingPath, config.PendingPath)
	}
}

func TestGenerateHandoff_NilConfig(t *testing.T) {
	metrics := &SessionMetrics{
		ToolCalls:         10,
		ErrorsLogged:      2,
		RoutingViolations: 1,
		SessionID:         "test-session",
	}

	err := GenerateHandoff(nil, metrics)

	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}

	if !strings.Contains(err.Error(), "[handoff]") {
		t.Errorf("Expected error with [handoff] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Config nil") {
		t.Errorf("Expected 'Config nil' in error, got: %v", err)
	}
}

func TestGenerateHandoff_NilMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	err := GenerateHandoff(config, nil)

	if err == nil {
		t.Error("Expected error for nil metrics, got nil")
	}

	if !strings.Contains(err.Error(), "[handoff]") {
		t.Errorf("Expected error with [handoff] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Metrics nil") {
		t.Errorf("Expected 'Metrics nil' in error, got: %v", err)
	}
}

func TestGenerateHandoff_MinimalSession(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	metrics := &SessionMetrics{
		ToolCalls:         5,
		ErrorsLogged:      0,
		RoutingViolations: 0,
		SessionID:         "test-123",
	}

	err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(config.HandoffPath); os.IsNotExist(err) {
		t.Fatal("Handoff file was not created")
	}

	// Load and verify
	handoff, err := LoadHandoff(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoff: %v", err)
	}

	if handoff == nil {
		t.Fatal("Expected handoff, got nil")
	}

	if handoff.SessionID != "test-123" {
		t.Errorf("Expected SessionID 'test-123', got: %s", handoff.SessionID)
	}

	if handoff.SchemaVersion != HandoffSchemaVersion {
		t.Errorf("Expected SchemaVersion '%s', got: %s", HandoffSchemaVersion, handoff.SchemaVersion)
	}

	if handoff.Context.Metrics.ToolCalls != 5 {
		t.Errorf("Expected ToolCalls 5, got: %d", handoff.Context.Metrics.ToolCalls)
	}
}

func TestGenerateHandoff_WithArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Create pending learnings file
	pendingData := `{"file":"test.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1000}
{"file":"main.go","error_type":"type_mismatch","consecutive_failures":2,"timestamp":1100}`
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	// Create violations file
	violationsData := `{"agent":"test-agent","violation_type":"wrong_tier","timestamp":1200}`
	os.MkdirAll(filepath.Dir(config.ViolationsPath), 0755)
	os.WriteFile(config.ViolationsPath, []byte(violationsData), 0644)

	metrics := &SessionMetrics{
		ToolCalls:         42,
		ErrorsLogged:      5,
		RoutingViolations: 1,
		SessionID:         "session-456",
	}

	err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Load and verify artifacts
	handoff, err := LoadHandoff(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoff: %v", err)
	}

	if len(handoff.Artifacts.SharpEdges) != 2 {
		t.Errorf("Expected 2 sharp edges, got: %d", len(handoff.Artifacts.SharpEdges))
	}

	if len(handoff.Artifacts.RoutingViolations) != 1 {
		t.Errorf("Expected 1 routing violation, got: %d", len(handoff.Artifacts.RoutingViolations))
	}

	// Verify sharp edge details
	if handoff.Artifacts.SharpEdges[0].File != "test.go" {
		t.Errorf("Expected first edge file 'test.go', got: %s", handoff.Artifacts.SharpEdges[0].File)
	}

	if handoff.Artifacts.SharpEdges[0].ErrorType != "nil_pointer" {
		t.Errorf("Expected error type 'nil_pointer', got: %s", handoff.Artifacts.SharpEdges[0].ErrorType)
	}

	// Verify violation details
	if handoff.Artifacts.RoutingViolations[0].Agent != "test-agent" {
		t.Errorf("Expected agent 'test-agent', got: %s", handoff.Artifacts.RoutingViolations[0].Agent)
	}
}

func TestGenerateHandoff_Actions(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Create artifacts to generate actions
	pendingData := `{"file":"test.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1000}`
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	metrics := &SessionMetrics{
		ToolCalls:         10,
		ErrorsLogged:      1,
		RoutingViolations: 0,
		SessionID:         "test-789",
	}

	err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	handoff, _ := LoadHandoff(config.HandoffPath)

	if len(handoff.Actions) == 0 {
		t.Error("Expected actions to be generated, got none")
	}

	// First action should be about sharp edges
	if !strings.Contains(handoff.Actions[0].Description, "sharp edge") {
		t.Errorf("Expected sharp edge action, got: %s", handoff.Actions[0].Description)
	}

	if handoff.Actions[0].Priority != 1 {
		t.Errorf("Expected priority 1 for first action, got: %d", handoff.Actions[0].Priority)
	}
}

func TestLoadHandoff_MissingFile(t *testing.T) {
	handoff, err := LoadHandoff("/tmp/nonexistent-handoff.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if handoff != nil {
		t.Errorf("Expected nil for missing file, got: %v", handoff)
	}
}

func TestLoadHandoff_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(handoffPath, []byte(""), 0644)

	handoff, err := LoadHandoff(handoffPath)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if handoff != nil {
		t.Errorf("Expected nil for empty file, got: %v", handoff)
	}
}

func TestLoadHandoff_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "malformed.jsonl")
	os.WriteFile(handoffPath, []byte("not json\n{\"some\":\"invalid\"}"), 0644)

	handoff, err := LoadHandoff(handoffPath)

	// Should not error, just skip malformed lines
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Second line can unmarshal to Handoff (with zero values), so we get a handoff
	// This is expected behavior - JSON unmarshaling succeeds even with missing fields
	if handoff == nil {
		t.Error("Expected handoff (even with zero values), got nil")
	}
}

func TestLoadAllHandoffs_MultipleHandoffs(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Generate multiple handoffs
	for i := 1; i <= 3; i++ {
		metrics := &SessionMetrics{
			ToolCalls:         i * 10,
			ErrorsLogged:      i,
			RoutingViolations: 0,
			SessionID:         "session-" + string(rune('0'+i)),
		}
		err := GenerateHandoff(config, metrics)
		if err != nil {
			t.Fatalf("Failed to generate handoff %d: %v", i, err)
		}
	}

	// Load all handoffs
	handoffs, err := LoadAllHandoffs(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoffs: %v", err)
	}

	if len(handoffs) != 3 {
		t.Errorf("Expected 3 handoffs, got: %d", len(handoffs))
	}

	// Verify they're in order
	if handoffs[0].Context.Metrics.ToolCalls != 10 {
		t.Errorf("Expected first handoff ToolCalls 10, got: %d", handoffs[0].Context.Metrics.ToolCalls)
	}

	if handoffs[2].Context.Metrics.ToolCalls != 30 {
		t.Errorf("Expected third handoff ToolCalls 30, got: %d", handoffs[2].Context.Metrics.ToolCalls)
	}
}

func TestLoadAllHandoffs_MissingFile(t *testing.T) {
	handoffs, err := LoadAllHandoffs("/tmp/nonexistent-all.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(handoffs) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %v", handoffs)
	}
}

func TestBuildSessionContext(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	metrics := &SessionMetrics{
		ToolCalls:         25,
		ErrorsLogged:      3,
		RoutingViolations: 1,
		SessionID:         "test-context",
	}

	context := buildSessionContext(config, metrics)

	if context.ProjectDir != tmpDir {
		t.Errorf("Expected ProjectDir %s, got: %s", tmpDir, context.ProjectDir)
	}

	if context.Metrics.ToolCalls != 25 {
		t.Errorf("Expected ToolCalls 25, got: %d", context.Metrics.ToolCalls)
	}

	if context.Metrics.SessionID != "test-context" {
		t.Errorf("Expected SessionID 'test-context', got: %s", context.Metrics.SessionID)
	}
}

func TestGetActiveTicket_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	ticket := getActiveTicket(tmpDir)

	if ticket != "" {
		t.Errorf("Expected empty string for missing file, got: %s", ticket)
	}
}

func TestGetActiveTicket_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	ticketPath := filepath.Join(tmpDir, ".ticket-current")
	os.WriteFile(ticketPath, []byte("GOgent-028\n"), 0644)

	ticket := getActiveTicket(tmpDir)

	if ticket != "GOgent-028" {
		t.Errorf("Expected 'GOgent-028', got: %s", ticket)
	}
}

func TestCollectGitInfo(t *testing.T) {
	tmpDir := t.TempDir()
	info := collectGitInfo(tmpDir)

	// Currently placeholder - should not error
	if info.Branch != "" {
		t.Logf("Git info collected: %+v", info)
	}
}

func TestGenerateActions_NoArtifacts(t *testing.T) {
	artifacts := HandoffArtifacts{
		SharpEdges:        []SharpEdge{},
		RoutingViolations: []RoutingViolation{},
		ErrorPatterns:     []ErrorPattern{},
	}

	actions := generateActions(artifacts)

	if len(actions) != 0 {
		t.Errorf("Expected no actions for empty artifacts, got: %d", len(actions))
	}
}

func TestGenerateActions_AllArtifacts(t *testing.T) {
	artifacts := HandoffArtifacts{
		SharpEdges: []SharpEdge{
			{File: "test.go", ErrorType: "nil_pointer", ConsecutiveFailures: 3},
		},
		RoutingViolations: []RoutingViolation{
			{Agent: "test-agent", ViolationType: "wrong_tier"},
		},
		ErrorPatterns: []ErrorPattern{
			{ErrorType: "import_error", Count: 5},
		},
	}

	actions := generateActions(artifacts)

	if len(actions) != 3 {
		t.Errorf("Expected 3 actions, got: %d", len(actions))
	}

	// Verify priority order
	for i, action := range actions {
		if action.Priority != i+1 {
			t.Errorf("Expected action %d to have priority %d, got: %d", i, i+1, action.Priority)
		}
	}

	// Verify descriptions
	if !strings.Contains(actions[0].Description, "sharp edge") {
		t.Errorf("Expected sharp edge in action 1, got: %s", actions[0].Description)
	}

	if !strings.Contains(actions[1].Description, "violation") {
		t.Errorf("Expected violation in action 2, got: %s", actions[1].Description)
	}

	if !strings.Contains(actions[2].Description, "error pattern") {
		t.Errorf("Expected error pattern in action 3, got: %s", actions[2].Description)
	}
}

func TestHandoffJSONSerialization(t *testing.T) {
	handoff := Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-serialize",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         100,
				ErrorsLogged:      5,
				RoutingViolations: 2,
				SessionID:         "test-serialize",
			},
		},
		Artifacts: HandoffArtifacts{
			SharpEdges: []SharpEdge{
				{File: "test.go", ErrorType: "test", ConsecutiveFailures: 3, Timestamp: 1000},
			},
		},
		Actions: []Action{
			{Priority: 1, Description: "Test action", Context: "Test context"},
		},
	}

	// Serialize
	data, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("Failed to marshal handoff: %v", err)
	}

	// Deserialize
	var deserialized Handoff
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Fatalf("Failed to unmarshal handoff: %v", err)
	}

	// Verify
	if deserialized.SessionID != handoff.SessionID {
		t.Errorf("SessionID mismatch after serialization")
	}

	if deserialized.Context.Metrics.ToolCalls != handoff.Context.Metrics.ToolCalls {
		t.Errorf("ToolCalls mismatch after serialization")
	}

	if len(deserialized.Artifacts.SharpEdges) != len(handoff.Artifacts.SharpEdges) {
		t.Errorf("SharpEdges count mismatch after serialization")
	}
}

func TestHandoffSchemaVersion(t *testing.T) {
	if HandoffSchemaVersion != "1.0" {
		t.Errorf("Expected schema version '1.0', got: %s", HandoffSchemaVersion)
	}
}
