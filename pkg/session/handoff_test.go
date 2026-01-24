package session

import (
	"encoding/json"
	"os"
	"os/exec"
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

	handoff, hMetrics, err := GenerateHandoff(nil, metrics)

	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}

	if handoff != nil {
		t.Error("Expected nil handoff for nil config")
	}

	if hMetrics != nil {
		t.Error("Expected nil metrics for nil config")
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

	handoff, hMetrics, err := GenerateHandoff(config, nil)

	if err == nil {
		t.Error("Expected error for nil metrics, got nil")
	}

	if handoff != nil {
		t.Error("Expected nil handoff for nil metrics")
	}

	if hMetrics != nil {
		t.Error("Expected nil HandoffMetrics for nil session metrics")
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
	// Override ViolationsPath to use temp dir (avoid picking up system-wide violations)
	config.ViolationsPath = filepath.Join(tmpDir, ".claude", "memory", "routing-violations.jsonl")

	metrics := &SessionMetrics{
		ToolCalls:         5,
		ErrorsLogged:      0,
		RoutingViolations: 0,
		SessionID:         "test-123",
	}

	handoff, hMetrics, err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if handoff == nil {
		t.Fatal("Expected handoff, got nil")
	}

	if hMetrics == nil {
		t.Fatal("Expected HandoffMetrics, got nil")
	}

	// Verify HandoffMetrics
	if hMetrics.GenerationTimeMs < 0 {
		t.Errorf("Expected non-negative GenerationTimeMs, got: %d", hMetrics.GenerationTimeMs)
	}

	if hMetrics.SharpEdgeCount != 0 {
		t.Errorf("Expected SharpEdgeCount 0, got: %d", hMetrics.SharpEdgeCount)
	}

	if hMetrics.ViolationCount != 0 {
		t.Errorf("Expected ViolationCount 0, got: %d", hMetrics.ViolationCount)
	}

	if hMetrics.PatternCount != 0 {
		t.Errorf("Expected PatternCount 0, got: %d", hMetrics.PatternCount)
	}

	// Verify file was created
	if _, err := os.Stat(config.HandoffPath); os.IsNotExist(err) {
		t.Fatal("Handoff file was not created")
	}

	// Load and verify from file
	loadedHandoff, err := LoadHandoff(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoff: %v", err)
	}

	if loadedHandoff == nil {
		t.Fatal("Expected loaded handoff, got nil")
	}

	if loadedHandoff.SessionID != "test-123" {
		t.Errorf("Expected SessionID 'test-123', got: %s", loadedHandoff.SessionID)
	}

	if loadedHandoff.SchemaVersion != HandoffSchemaVersion {
		t.Errorf("Expected SchemaVersion '%s', got: %s", HandoffSchemaVersion, loadedHandoff.SchemaVersion)
	}

	if loadedHandoff.Context.Metrics.ToolCalls != 5 {
		t.Errorf("Expected ToolCalls 5, got: %d", loadedHandoff.Context.Metrics.ToolCalls)
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

	// Create violations file with multiple violations of different types
	violationsData := `{"agent":"test-agent","violation_type":"wrong_tier","timestamp":1200}
{"agent":"other-agent","violation_type":"wrong_tier","timestamp":1300}
{"agent":"third-agent","violation_type":"missing_delegation","timestamp":1400}`
	os.MkdirAll(filepath.Dir(config.ViolationsPath), 0755)
	os.WriteFile(config.ViolationsPath, []byte(violationsData), 0644)

	metrics := &SessionMetrics{
		ToolCalls:         42,
		ErrorsLogged:      5,
		RoutingViolations: 3,
		SessionID:         "session-456",
	}

	handoff, hMetrics, err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if handoff == nil {
		t.Fatal("Expected handoff, got nil")
	}

	if hMetrics == nil {
		t.Fatal("Expected HandoffMetrics, got nil")
	}

	// Verify HandoffMetrics counts
	if hMetrics.SharpEdgeCount != 2 {
		t.Errorf("Expected SharpEdgeCount 2, got: %d", hMetrics.SharpEdgeCount)
	}

	if hMetrics.ViolationCount != 3 {
		t.Errorf("Expected ViolationCount 3, got: %d", hMetrics.ViolationCount)
	}

	// PatternCount should be 2 (wrong_tier and missing_delegation)
	if hMetrics.PatternCount != 2 {
		t.Errorf("Expected PatternCount 2 (unique violation types), got: %d", hMetrics.PatternCount)
	}

	if hMetrics.GenerationTimeMs < 0 {
		t.Errorf("Expected non-negative GenerationTimeMs, got: %d", hMetrics.GenerationTimeMs)
	}

	// Load and verify artifacts from file
	loadedHandoff, err := LoadHandoff(config.HandoffPath)
	if err != nil {
		t.Fatalf("Failed to load handoff: %v", err)
	}

	if len(loadedHandoff.Artifacts.SharpEdges) != 2 {
		t.Errorf("Expected 2 sharp edges, got: %d", len(loadedHandoff.Artifacts.SharpEdges))
	}

	if len(loadedHandoff.Artifacts.RoutingViolations) != 3 {
		t.Errorf("Expected 3 routing violations, got: %d", len(loadedHandoff.Artifacts.RoutingViolations))
	}

	// Verify sharp edge details
	if loadedHandoff.Artifacts.SharpEdges[0].File != "test.go" {
		t.Errorf("Expected first edge file 'test.go', got: %s", loadedHandoff.Artifacts.SharpEdges[0].File)
	}

	if loadedHandoff.Artifacts.SharpEdges[0].ErrorType != "nil_pointer" {
		t.Errorf("Expected error type 'nil_pointer', got: %s", loadedHandoff.Artifacts.SharpEdges[0].ErrorType)
	}

	// Verify violation details
	if loadedHandoff.Artifacts.RoutingViolations[0].Agent != "test-agent" {
		t.Errorf("Expected agent 'test-agent', got: %s", loadedHandoff.Artifacts.RoutingViolations[0].Agent)
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

	handoff, _, err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if handoff == nil {
		t.Fatal("Expected handoff, got nil")
	}

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
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "nonexistent.jsonl")

	handoff, err := LoadHandoff(handoffPath)

	// Missing file should return nil without error (normal case)
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

	// Empty file should return nil without error (acceptable case)
	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if handoff != nil {
		t.Errorf("Expected nil for empty file, got: %v", handoff)
	}
}

func TestLoadHandoff_CurrentVersion(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "handoffs.jsonl")

	// Write v1.1 handoff (current version)
	handoff := &Handoff{
		SessionID:     "test-v1",
		Timestamp:     1234567890,
		SchemaVersion: "1.1",
		Context: SessionContext{
			Metrics: SessionMetrics{ToolCalls: 5},
		},
	}

	jsonData, _ := json.Marshal(handoff)
	os.WriteFile(handoffPath, jsonData, 0644)

	loaded, err := LoadHandoff(handoffPath)

	if err != nil {
		t.Fatalf("Expected no error loading v1.1, got: %v", err)
	}

	// v1.1 handoffs are migrated to current version (1.3)
	if loaded.SchemaVersion != "1.3" {
		t.Errorf("Expected schema version 1.3 (migrated), got: %s", loaded.SchemaVersion)
	}

	if loaded.SessionID != "test-v1" {
		t.Errorf("Expected session ID test-v1, got: %s", loaded.SessionID)
	}
}

func TestLoadHandoff_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "handoffs.jsonl")

	// Write multiple JSONL entries
	handoff1 := &Handoff{SessionID: "session-1", SchemaVersion: "1.0", Timestamp: 100}
	handoff2 := &Handoff{SessionID: "session-2", SchemaVersion: "1.0", Timestamp: 200}
	handoff3 := &Handoff{SessionID: "session-3", SchemaVersion: "1.0", Timestamp: 300}

	json1, _ := json.Marshal(handoff1)
	json2, _ := json.Marshal(handoff2)
	json3, _ := json.Marshal(handoff3)

	content := string(json1) + "\n" + string(json2) + "\n" + string(json3) + "\n"
	os.WriteFile(handoffPath, []byte(content), 0644)

	loaded, err := LoadHandoff(handoffPath)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should load LAST entry
	if loaded.SessionID != "session-3" {
		t.Errorf("Expected last session (session-3), got: %s", loaded.SessionID)
	}
}

func TestLoadHandoff_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "handoffs.jsonl")

	os.WriteFile(handoffPath, []byte("{invalid json}"), 0644)

	_, err := LoadHandoff(handoffPath)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "[handoff]") {
		t.Errorf("Expected error with [handoff] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to parse") {
		t.Errorf("Expected 'Failed to parse' in error, got: %v", err)
	}
}

func TestLoadHandoff_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "handoffs.jsonl")

	// Write future schema version
	futureHandoff := map[string]interface{}{
		"schema_version": "3.5",
		"session_id":     "future-session",
		"timestamp":      1234567890,
	}

	jsonData, _ := json.Marshal(futureHandoff)
	os.WriteFile(handoffPath, jsonData, 0644)

	_, err := LoadHandoff(handoffPath)

	if err == nil {
		t.Error("Expected error for unsupported version 3.5, got nil")
	}

	if !strings.Contains(err.Error(), "Unsupported schema version 3.5") {
		t.Errorf("Expected 'Unsupported schema version 3.5' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Upgrade gogent-archive") {
		t.Errorf("Expected 'Upgrade gogent-archive' guidance in error, got: %v", err)
	}
}

func TestMigrateHandoff_V10ToV11(t *testing.T) {
	// v1.0 handoff without new fields (simulating legacy data)
	handoff := &Handoff{
		SessionID:     "migrate-test",
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
	}

	jsonData, _ := json.Marshal(handoff)

	migrated, err := migrateHandoff("1.0", jsonData)

	if err != nil {
		t.Fatalf("Expected no error migrating v1.0 to v1.1, got: %v", err)
	}

	if migrated.SessionID != "migrate-test" {
		t.Errorf("Expected session ID migrate-test, got: %s", migrated.SessionID)
	}

	// v1.0 handoffs are migrated to current version (1.3)
	if migrated.SchemaVersion != "1.3" {
		t.Errorf("Expected schema version 1.3 after migration, got: %s", migrated.SchemaVersion)
	}

	// New fields should be initialized to empty slices
	if migrated.Artifacts.Decisions == nil {
		t.Error("Expected Decisions to be non-nil after migration")
	}
	if migrated.Artifacts.PreferenceOverrides == nil {
		t.Error("Expected PreferenceOverrides to be non-nil after migration")
	}
	if migrated.Artifacts.PerformanceMetrics == nil {
		t.Error("Expected PerformanceMetrics to be non-nil after migration")
	}
}

func TestMigrateHandoff_V11Direct(t *testing.T) {
	// v1.1 handoff with all new fields
	handoff := &Handoff{
		SessionID:     "v11-test",
		SchemaVersion: "1.1",
		Timestamp:     1234567890,
		Artifacts: HandoffArtifacts{
			Decisions:           []Decision{{Timestamp: 100, Category: "test"}},
			PreferenceOverrides: []PreferenceOverride{{Timestamp: 100, Key: "test"}},
			PerformanceMetrics:  []PerformanceMetric{{Timestamp: 100, Operation: "test"}},
		},
	}

	jsonData, _ := json.Marshal(handoff)

	migrated, err := migrateHandoff("1.1", jsonData)

	if err != nil {
		t.Fatalf("Expected no error for v1.1 direct, got: %v", err)
	}

	if migrated.SessionID != "v11-test" {
		t.Errorf("Expected session ID v11-test, got: %s", migrated.SessionID)
	}

	// v1.1 handoffs are migrated to current version (1.3)
	if migrated.SchemaVersion != "1.3" {
		t.Errorf("Expected schema version 1.3, got: %s", migrated.SchemaVersion)
	}

	if len(migrated.Artifacts.Decisions) != 1 {
		t.Errorf("Expected 1 decision, got: %d", len(migrated.Artifacts.Decisions))
	}
}

func TestMigrateHandoff_V12ToV13(t *testing.T) {
	// v1.2 handoff without AgentEndstates (simulating legacy data)
	handoff := &Handoff{
		SessionID:     "v12-test",
		SchemaVersion: "1.2",
		Timestamp:     1234567890,
		Artifacts: HandoffArtifacts{
			SharpEdges:          []SharpEdge{{File: "test.go", ErrorType: "test", ConsecutiveFailures: 3, Timestamp: 100}},
			Decisions:           []Decision{{Timestamp: 100, Category: "test"}},
			PreferenceOverrides: []PreferenceOverride{{Timestamp: 100, Key: "test"}},
			PerformanceMetrics:  []PerformanceMetric{{Timestamp: 100, Operation: "test"}},
		},
	}

	jsonData, _ := json.Marshal(handoff)

	migrated, err := migrateHandoff("1.2", jsonData)

	if err != nil {
		t.Fatalf("Expected no error migrating v1.2 to v1.3, got: %v", err)
	}

	if migrated.SessionID != "v12-test" {
		t.Errorf("Expected session ID v12-test, got: %s", migrated.SessionID)
	}

	// v1.2 handoffs should be migrated to v1.3
	if migrated.SchemaVersion != "1.3" {
		t.Errorf("Expected schema version 1.3 after migration, got: %s", migrated.SchemaVersion)
	}

	// Existing fields should be preserved
	if len(migrated.Artifacts.SharpEdges) != 1 {
		t.Errorf("Expected 1 sharp edge preserved, got: %d", len(migrated.Artifacts.SharpEdges))
	}
	if len(migrated.Artifacts.Decisions) != 1 {
		t.Errorf("Expected 1 decision preserved, got: %d", len(migrated.Artifacts.Decisions))
	}

	// New v1.3 field should be initialized to empty slice (not nil)
	if migrated.Artifacts.AgentEndstates == nil {
		t.Error("Expected AgentEndstates to be non-nil after migration from v1.2")
	}
	if len(migrated.Artifacts.AgentEndstates) != 0 {
		t.Errorf("Expected 0 agent endstates for v1.2 migration, got: %d", len(migrated.Artifacts.AgentEndstates))
	}
}

func TestMigrateHandoff_V13Direct(t *testing.T) {
	// v1.3 handoff with AgentEndstates
	handoff := &Handoff{
		SessionID:     "v13-test",
		SchemaVersion: "1.3",
		Timestamp:     1234567890,
		Artifacts: HandoffArtifacts{
			Decisions: []Decision{{Timestamp: 100, Category: "test"}},
			AgentEndstates: []EndstateLog{
				{
					AgentID:    "test-agent",
					AgentClass: "implementation",
					Tier:       "sonnet",
				},
			},
		},
	}

	jsonData, _ := json.Marshal(handoff)

	migrated, err := migrateHandoff("1.3", jsonData)

	if err != nil {
		t.Fatalf("Expected no error for v1.3 direct, got: %v", err)
	}

	if migrated.SessionID != "v13-test" {
		t.Errorf("Expected session ID v13-test, got: %s", migrated.SessionID)
	}

	// v1.3 should remain v1.3
	if migrated.SchemaVersion != "1.3" {
		t.Errorf("Expected schema version 1.3, got: %s", migrated.SchemaVersion)
	}

	if len(migrated.Artifacts.AgentEndstates) != 1 {
		t.Errorf("Expected 1 agent endstate, got: %d", len(migrated.Artifacts.AgentEndstates))
	}

	if migrated.Artifacts.AgentEndstates[0].AgentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got: %s", migrated.Artifacts.AgentEndstates[0].AgentID)
	}
}

func TestMigrateHandoff_FutureVersionStub(t *testing.T) {
	// This test will fail when v2.0 is actually implemented
	// It documents the migration path exists

	futureHandoff := map[string]interface{}{
		"schema_version": "2.0",
		"session_id":     "future",
	}

	jsonData, _ := json.Marshal(futureHandoff)

	_, err := migrateHandoff("2.0", jsonData)

	if err == nil {
		t.Error("Expected error for unimplemented v2.0 migration, got nil")
	}

	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' in error, got: %v", err)
	}
}

func TestMigrateHandoff_EmptyVersion(t *testing.T) {
	// Test handling of empty schema version
	handoff := map[string]interface{}{
		"session_id": "no-version",
		"timestamp":  1234567890,
	}

	jsonData, _ := json.Marshal(handoff)

	_, err := migrateHandoff("", jsonData)

	if err == nil {
		t.Error("Expected error for empty schema version, got nil")
	}

	if !strings.Contains(err.Error(), "Unsupported schema version (empty)") {
		t.Errorf("Expected 'Unsupported schema version (empty)' in error, got: %v", err)
	}
}

func TestMigrateHandoff_UnknownVersion(t *testing.T) {
	// Test handling of unknown schema version
	handoff := map[string]interface{}{
		"schema_version": "99.0",
		"session_id":     "unknown-version",
	}

	jsonData, _ := json.Marshal(handoff)

	_, err := migrateHandoff("99.0", jsonData)

	if err == nil {
		t.Error("Expected error for unknown version 99.0, got nil")
	}

	if !strings.Contains(err.Error(), "Unsupported schema version 99.0") {
		t.Errorf("Expected 'Unsupported schema version 99.0' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Upgrade gogent-archive") {
		t.Errorf("Expected 'Upgrade gogent-archive' guidance in error, got: %v", err)
	}
}

func TestLoadHandoff_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	handoffPath := filepath.Join(tmpDir, "malformed.jsonl")

	// Write valid JSON in first line, malformed JSON in last line
	// LoadHandoff only reads the last non-empty line
	os.WriteFile(handoffPath, []byte("{\"session_id\":\"valid\",\"schema_version\":\"1.0\"}\n{not valid json}"), 0644)

	_, err := LoadHandoff(handoffPath)

	// Should error because last line is malformed JSON
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}

	if !strings.Contains(err.Error(), "[handoff]") {
		t.Errorf("Expected error with [handoff] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to parse") {
		t.Errorf("Expected 'Failed to parse' in error, got: %v", err)
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
		_, _, err := GenerateHandoff(config, metrics)
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

func TestCollectGitInfo_NonGitDir(t *testing.T) {
	// Create a truly isolated temp directory outside any git repo
	tmpDir, err := os.MkdirTemp("/tmp", "non-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := collectGitInfo(tmpDir)

	// Non-git directory should return empty GitInfo
	if info.Branch != "" {
		t.Errorf("Expected empty Branch for non-git directory, got: %s", info.Branch)
	}

	if info.IsDirty {
		t.Errorf("Expected IsDirty=false for non-git directory, got: %v", info.IsDirty)
	}

	if len(info.Uncommitted) > 0 {
		t.Errorf("Expected empty Uncommitted for non-git directory, got: %v", info.Uncommitted)
	}
}

func TestCollectGitInfo_ValidRepo(t *testing.T) {
	// This test requires running in a git repository
	// Skip if not in git environment
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		t.Skip("Not in git repo, skipping git info test")
	}

	// Use current directory (the GOgent-Fortress repo itself)
	info := collectGitInfo(".")

	// In a valid git repo, we should get a branch name
	if info.Branch == "" {
		t.Error("Expected non-empty branch name in git repo")
	}

	// Log the collected info for visibility
	t.Logf("Git info collected: Branch=%s, IsDirty=%v, Uncommitted=%v", info.Branch, info.IsDirty, info.Uncommitted)
}

func TestCollectGitInfo_CleanRepo(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git user for this repo
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create and commit a file to have a branch
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Collect git info
	info := collectGitInfo(tmpDir)

	// Should have branch name (master or main depending on git version)
	if info.Branch == "" {
		t.Error("Expected branch name, got empty string")
	}

	// Should not be dirty (no uncommitted changes)
	if info.IsDirty {
		t.Errorf("Expected clean repo (IsDirty=false), got: %v", info.IsDirty)
	}

	// Should have no uncommitted files
	if len(info.Uncommitted) > 0 {
		t.Errorf("Expected no uncommitted files in clean repo, got: %v", info.Uncommitted)
	}
}

func TestCollectGitInfo_DirtyRepo(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git user
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create and commit initial file
	testFile1 := filepath.Join(tmpDir, "committed.txt")
	os.WriteFile(testFile1, []byte("committed content"), 0644)

	cmd = exec.Command("git", "add", "committed.txt")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create uncommitted file
	testFile2 := filepath.Join(tmpDir, "uncommitted.txt")
	os.WriteFile(testFile2, []byte("uncommitted content"), 0644)

	// Modify committed file
	os.WriteFile(testFile1, []byte("modified content"), 0644)

	// Collect git info
	info := collectGitInfo(tmpDir)

	// Should have branch name
	if info.Branch == "" {
		t.Error("Expected branch name, got empty string")
	}

	// Should be dirty
	if !info.IsDirty {
		t.Error("Expected dirty repo (IsDirty=true), got false")
	}

	// Should have uncommitted files
	if len(info.Uncommitted) == 0 {
		t.Error("Expected uncommitted files, got empty list")
	}

	// Verify we captured both files
	hasCommittedFile := false
	hasUncommittedFile := false
	for _, file := range info.Uncommitted {
		if strings.Contains(file, "committed.txt") {
			hasCommittedFile = true
		}
		if strings.Contains(file, "uncommitted.txt") {
			hasUncommittedFile = true
		}
	}

	if !hasCommittedFile {
		t.Errorf("Expected to find modified committed.txt in uncommitted files, got: %v", info.Uncommitted)
	}

	if !hasUncommittedFile {
		t.Errorf("Expected to find uncommitted.txt in uncommitted files, got: %v", info.Uncommitted)
	}

	t.Logf("Dirty repo info: Branch=%s, IsDirty=%v, Uncommitted=%v", info.Branch, info.IsDirty, info.Uncommitted)
}

func TestCollectGitInfo_DetachedHead(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git user
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(output))

	// Checkout the commit directly to enter detached HEAD state
	cmd = exec.Command("git", "checkout", commitHash)
	cmd.Dir = tmpDir
	cmd.Run() // Ignore error - some git versions handle this differently

	// Collect git info
	info := collectGitInfo(tmpDir)

	// Should have branch name (will be "HEAD" in detached state)
	if info.Branch == "" {
		t.Error("Expected branch/HEAD name, got empty string")
	}

	// Log detached HEAD state
	t.Logf("Detached HEAD info: Branch=%s (expected 'HEAD' or commit hash)", info.Branch)
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
	// Schema version 1.3 adds: AgentEndstates to HandoffArtifacts
	if HandoffSchemaVersion != "1.3" {
		t.Errorf("Expected schema version '1.3', got: %s", HandoffSchemaVersion)
	}
}

func TestCountPatterns(t *testing.T) {
	tests := []struct {
		name       string
		violations []RoutingViolation
		expected   int
	}{
		{
			name:       "empty violations",
			violations: []RoutingViolation{},
			expected:   0,
		},
		{
			name: "single violation type",
			violations: []RoutingViolation{
				{Agent: "a", ViolationType: "wrong_tier"},
				{Agent: "b", ViolationType: "wrong_tier"},
			},
			expected: 1,
		},
		{
			name: "multiple unique violation types",
			violations: []RoutingViolation{
				{Agent: "a", ViolationType: "wrong_tier"},
				{Agent: "b", ViolationType: "missing_delegation"},
				{Agent: "c", ViolationType: "invalid_subagent"},
			},
			expected: 3,
		},
		{
			name: "mixed duplicates and unique",
			violations: []RoutingViolation{
				{Agent: "a", ViolationType: "wrong_tier"},
				{Agent: "b", ViolationType: "wrong_tier"},
				{Agent: "c", ViolationType: "missing_delegation"},
				{Agent: "d", ViolationType: "wrong_tier"},
				{Agent: "e", ViolationType: "missing_delegation"},
			},
			expected: 2,
		},
		{
			name: "empty violation type ignored",
			violations: []RoutingViolation{
				{Agent: "a", ViolationType: "wrong_tier"},
				{Agent: "b", ViolationType: ""},
				{Agent: "c", ViolationType: "missing_delegation"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countPatterns(tt.violations)
			if result != tt.expected {
				t.Errorf("countPatterns() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestHandoffMetrics_TimingAccuracy(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)
	// Override ViolationsPath to use temp dir (avoid picking up system-wide violations)
	config.ViolationsPath = filepath.Join(tmpDir, ".claude", "memory", "routing-violations.jsonl")

	metrics := &SessionMetrics{
		ToolCalls:         1,
		ErrorsLogged:      0,
		RoutingViolations: 0,
		SessionID:         "timing-test",
	}

	_, hMetrics, err := GenerateHandoff(config, metrics)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Timing should be non-negative (generation takes some time)
	if hMetrics.GenerationTimeMs < 0 {
		t.Errorf("Expected non-negative GenerationTimeMs, got: %d", hMetrics.GenerationTimeMs)
	}

	// Timing should be reasonable (less than 5 seconds for a simple handoff)
	if hMetrics.GenerationTimeMs > 5000 {
		t.Errorf("GenerationTimeMs seems too high: %d ms", hMetrics.GenerationTimeMs)
	}

	t.Logf("Handoff generation time: %d ms", hMetrics.GenerationTimeMs)
}
