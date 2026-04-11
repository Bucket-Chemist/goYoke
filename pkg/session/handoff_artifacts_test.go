package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadArtifacts_AllMissing(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir (avoid /tmp pollution)
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Errorf("Expected no error for missing files, got: %v", err)
	}

	if len(artifacts.SharpEdges) != 0 {
		t.Errorf("Expected empty SharpEdges, got: %d", len(artifacts.SharpEdges))
	}

	if len(artifacts.RoutingViolations) != 0 {
		t.Errorf("Expected empty RoutingViolations, got: %d", len(artifacts.RoutingViolations))
	}

	if len(artifacts.ErrorPatterns) != 0 {
		t.Errorf("Expected empty ErrorPatterns, got: %d", len(artifacts.ErrorPatterns))
	}
}

func TestLoadArtifacts_Complete(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Create pending learnings
	pendingData := `{"file":"test.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1000}
{"file":"main.go","error_type":"type_mismatch","consecutive_failures":2,"timestamp":1100}`
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	// Create violations
	violationsData := `{"agent":"test-agent","violation_type":"wrong_tier","timestamp":1200}
{"agent":"other-agent","violation_type":"missing_subagent_type","timestamp":1300}`
	os.MkdirAll(filepath.Dir(config.ViolationsPath), 0755)
	os.WriteFile(config.ViolationsPath, []byte(violationsData), 0644)

	// Create error patterns
	errorData := `{"error_type":"import_error","count":5,"last_seen":1400}
{"error_type":"syntax_error","count":2,"last_seen":1500}`
	os.MkdirAll(filepath.Dir(config.ErrorPatternsPath), 0755)
	os.WriteFile(config.ErrorPatternsPath, []byte(errorData), 0644)

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(artifacts.SharpEdges) != 2 {
		t.Errorf("Expected 2 SharpEdges, got: %d", len(artifacts.SharpEdges))
	}

	if len(artifacts.RoutingViolations) != 2 {
		t.Errorf("Expected 2 RoutingViolations, got: %d", len(artifacts.RoutingViolations))
	}

	if len(artifacts.ErrorPatterns) != 2 {
		t.Errorf("Expected 2 ErrorPatterns, got: %d", len(artifacts.ErrorPatterns))
	}
}

func TestLoadPendingLearnings_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "pending.jsonl")

	data := `{"file":"test.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1000,"context":"test context"}
{"file":"main.go","error_type":"type_mismatch","consecutive_failures":2,"timestamp":1100}`
	os.WriteFile(path, []byte(data), 0644)

	edges, err := loadPendingLearnings(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 2 {
		t.Errorf("Expected 2 edges, got: %d", len(edges))
	}

	// Verify first edge
	if edges[0].File != "test.go" {
		t.Errorf("Expected File 'test.go', got: %s", edges[0].File)
	}

	if edges[0].ErrorType != "nil_pointer" {
		t.Errorf("Expected ErrorType 'nil_pointer', got: %s", edges[0].ErrorType)
	}

	if edges[0].ConsecutiveFailures != 3 {
		t.Errorf("Expected ConsecutiveFailures 3, got: %d", edges[0].ConsecutiveFailures)
	}

	if edges[0].Context != "test context" {
		t.Errorf("Expected Context 'test context', got: %s", edges[0].Context)
	}

	// Verify second edge
	if edges[1].File != "main.go" {
		t.Errorf("Expected File 'main.go', got: %s", edges[1].File)
	}
}

func TestLoadPendingLearnings_MissingFile(t *testing.T) {
	edges, err := loadPendingLearnings("/tmp/nonexistent-pending.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d edges", len(edges))
	}
}

func TestLoadPendingLearnings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	edges, err := loadPendingLearnings(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d edges", len(edges))
	}
}

func TestLoadPendingLearnings_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `not json
{"file":"valid.go","error_type":"test","consecutive_failures":1,"timestamp":1000}
also not json`
	os.WriteFile(path, []byte(data), 0644)

	edges, err := loadPendingLearnings(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge (skipped malformed), got: %d", len(edges))
	}

	if edges[0].File != "valid.go" {
		t.Errorf("Expected valid edge to be parsed, got: %s", edges[0].File)
	}
}

func TestLoadPendingLearnings_LargeLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.jsonl")
	largeContext := strings.Repeat("x", 70*1024)
	data := fmt.Sprintf(`{"file":"large.go","error_type":"test","consecutive_failures":3,"timestamp":1000,"context":"%s"}`+"\n", largeContext)
	os.WriteFile(path, []byte(data), 0644)

	edges, err := loadPendingLearnings(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got: %d", len(edges))
	}
	if edges[0].Context != largeContext {
		t.Fatalf("Expected large context to round-trip without truncation")
	}
}

func TestLoadViolations_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "violations.jsonl")

	data := `{"agent":"test-agent","violation_type":"wrong_tier","expected_tier":"haiku","actual_tier":"sonnet","timestamp":1000}
{"agent":"other-agent","violation_type":"missing_subagent_type","timestamp":1100}`
	os.WriteFile(path, []byte(data), 0644)

	violations, err := loadViolations(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got: %d", len(violations))
	}

	// Verify first violation
	if violations[0].Agent != "test-agent" {
		t.Errorf("Expected Agent 'test-agent', got: %s", violations[0].Agent)
	}

	if violations[0].ViolationType != "wrong_tier" {
		t.Errorf("Expected ViolationType 'wrong_tier', got: %s", violations[0].ViolationType)
	}

	if violations[0].ExpectedTier != "haiku" {
		t.Errorf("Expected ExpectedTier 'haiku', got: %s", violations[0].ExpectedTier)
	}

	if violations[0].ActualTier != "sonnet" {
		t.Errorf("Expected ActualTier 'sonnet', got: %s", violations[0].ActualTier)
	}

	// Verify second violation
	if violations[1].Agent != "other-agent" {
		t.Errorf("Expected Agent 'other-agent', got: %s", violations[1].Agent)
	}
}

func TestLoadViolations_MissingFile(t *testing.T) {
	violations, err := loadViolations("/tmp/nonexistent-violations.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(violations) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d violations", len(violations))
	}
}

func TestLoadViolations_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	violations, err := loadViolations(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(violations) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d violations", len(violations))
	}
}

func TestLoadViolations_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `invalid
{"agent":"valid-agent","violation_type":"test","timestamp":1000}
{broken json}`
	os.WriteFile(path, []byte(data), 0644)

	violations, err := loadViolations(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation (skipped malformed), got: %d", len(violations))
	}

	if violations[0].Agent != "valid-agent" {
		t.Errorf("Expected valid violation to be parsed, got: %s", violations[0].Agent)
	}
}

func TestLoadErrorPatterns_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "errors.jsonl")

	data := `{"error_type":"import_error","count":5,"last_seen":1000,"context":"missing module"}
{"error_type":"syntax_error","count":2,"last_seen":1100}`
	os.WriteFile(path, []byte(data), 0644)

	patterns, err := loadErrorPatterns(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got: %d", len(patterns))
	}

	// Verify first pattern
	if patterns[0].ErrorType != "import_error" {
		t.Errorf("Expected ErrorType 'import_error', got: %s", patterns[0].ErrorType)
	}

	if patterns[0].Count != 5 {
		t.Errorf("Expected Count 5, got: %d", patterns[0].Count)
	}

	if patterns[0].Context != "missing module" {
		t.Errorf("Expected Context 'missing module', got: %s", patterns[0].Context)
	}

	// Verify second pattern
	if patterns[1].ErrorType != "syntax_error" {
		t.Errorf("Expected ErrorType 'syntax_error', got: %s", patterns[1].ErrorType)
	}

	if patterns[1].Count != 2 {
		t.Errorf("Expected Count 2, got: %d", patterns[1].Count)
	}
}

func TestLoadErrorPatterns_MissingFile(t *testing.T) {
	patterns, err := loadErrorPatterns("/tmp/nonexistent-errors.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(patterns) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d patterns", len(patterns))
	}
}

func TestLoadErrorPatterns_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	patterns, err := loadErrorPatterns(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(patterns) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d patterns", len(patterns))
	}
}

func TestLoadErrorPatterns_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `bad json
{"error_type":"valid_error","count":3,"last_seen":1000}
{incomplete`
	os.WriteFile(path, []byte(data), 0644)

	patterns, err := loadErrorPatterns(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern (skipped malformed), got: %d", len(patterns))
	}

	if patterns[0].ErrorType != "valid_error" {
		t.Errorf("Expected valid pattern to be parsed, got: %s", patterns[0].ErrorType)
	}
}

func TestLoadArtifacts_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir (avoid /tmp pollution)
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	// Create only pending learnings
	pendingData := `{"file":"test.go","error_type":"test","consecutive_failures":1,"timestamp":1000}`
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	// Other files don't exist

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Errorf("Expected no error with partial files, got: %v", err)
	}

	if len(artifacts.SharpEdges) != 1 {
		t.Errorf("Expected 1 SharpEdge, got: %d", len(artifacts.SharpEdges))
	}

	if len(artifacts.RoutingViolations) != 0 {
		t.Errorf("Expected 0 RoutingViolations, got: %d", len(artifacts.RoutingViolations))
	}

	if len(artifacts.ErrorPatterns) != 0 {
		t.Errorf("Expected 0 ErrorPatterns, got: %d", len(artifacts.ErrorPatterns))
	}
}

// ===== Tests for Decision loader (GOgent-029c) =====

func TestLoadDecisions_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "decisions.jsonl")

	data := `{"timestamp":1000,"category":"architecture","decision":"Use JSONL format","rationale":"Human readable and appendable","alternatives":"SQLite, binary","impact":"high"}
{"timestamp":1100,"category":"tooling","decision":"Use Go for CLIs","rationale":"Fast compilation","alternatives":"Python, Rust","impact":"medium"}`
	os.WriteFile(path, []byte(data), 0644)

	decisions, err := loadDecisions(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 2 {
		t.Errorf("Expected 2 decisions, got: %d", len(decisions))
	}

	// Verify first decision
	if decisions[0].Timestamp != 1000 {
		t.Errorf("Expected Timestamp 1000, got: %d", decisions[0].Timestamp)
	}
	if decisions[0].Category != "architecture" {
		t.Errorf("Expected Category 'architecture', got: %s", decisions[0].Category)
	}
	if decisions[0].Decision != "Use JSONL format" {
		t.Errorf("Expected Decision 'Use JSONL format', got: %s", decisions[0].Decision)
	}
	if decisions[0].Rationale != "Human readable and appendable" {
		t.Errorf("Expected Rationale 'Human readable and appendable', got: %s", decisions[0].Rationale)
	}
	if decisions[0].Alternatives != "SQLite, binary" {
		t.Errorf("Expected Alternatives 'SQLite, binary', got: %s", decisions[0].Alternatives)
	}
	if decisions[0].Impact != "high" {
		t.Errorf("Expected Impact 'high', got: %s", decisions[0].Impact)
	}

	// Verify second decision
	if decisions[1].Category != "tooling" {
		t.Errorf("Expected Category 'tooling', got: %s", decisions[1].Category)
	}
	if decisions[1].Impact != "medium" {
		t.Errorf("Expected Impact 'medium', got: %s", decisions[1].Impact)
	}
}

func TestLoadDecisions_MissingFile(t *testing.T) {
	decisions, err := loadDecisions("/tmp/nonexistent-decisions.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(decisions) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d decisions", len(decisions))
	}
}

func TestLoadDecisions_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	decisions, err := loadDecisions(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(decisions) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d decisions", len(decisions))
	}
}

func TestLoadDecisions_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `not json
{"timestamp":1000,"category":"valid","decision":"Keep this","rationale":"test","alternatives":"none","impact":"low"}
{broken json}`
	os.WriteFile(path, []byte(data), 0644)

	decisions, err := loadDecisions(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(decisions) != 1 {
		t.Errorf("Expected 1 decision (skipped malformed), got: %d", len(decisions))
	}

	if decisions[0].Category != "valid" {
		t.Errorf("Expected valid decision to be parsed, got: %s", decisions[0].Category)
	}
}

// ===== Tests for PreferenceOverride loader (GOgent-029c) =====

func TestLoadPreferences_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "preferences.jsonl")

	data := `{"timestamp":1000,"category":"routing","key":"default_tier","value":"sonnet","reason":"Prefer quality over cost","scope":"project"}
{"timestamp":1100,"category":"formatting","key":"indent_style","value":"tabs","reason":"Personal preference","scope":"global"}`
	os.WriteFile(path, []byte(data), 0644)

	preferences, err := loadPreferences(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(preferences) != 2 {
		t.Errorf("Expected 2 preferences, got: %d", len(preferences))
	}

	// Verify first preference
	if preferences[0].Timestamp != 1000 {
		t.Errorf("Expected Timestamp 1000, got: %d", preferences[0].Timestamp)
	}
	if preferences[0].Category != "routing" {
		t.Errorf("Expected Category 'routing', got: %s", preferences[0].Category)
	}
	if preferences[0].Key != "default_tier" {
		t.Errorf("Expected Key 'default_tier', got: %s", preferences[0].Key)
	}
	if preferences[0].Value != "sonnet" {
		t.Errorf("Expected Value 'sonnet', got: %s", preferences[0].Value)
	}
	if preferences[0].Reason != "Prefer quality over cost" {
		t.Errorf("Expected Reason 'Prefer quality over cost', got: %s", preferences[0].Reason)
	}
	if preferences[0].Scope != "project" {
		t.Errorf("Expected Scope 'project', got: %s", preferences[0].Scope)
	}

	// Verify second preference
	if preferences[1].Category != "formatting" {
		t.Errorf("Expected Category 'formatting', got: %s", preferences[1].Category)
	}
	if preferences[1].Scope != "global" {
		t.Errorf("Expected Scope 'global', got: %s", preferences[1].Scope)
	}
}

func TestLoadPreferences_MissingFile(t *testing.T) {
	preferences, err := loadPreferences("/tmp/nonexistent-preferences.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(preferences) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d preferences", len(preferences))
	}
}

func TestLoadPreferences_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	preferences, err := loadPreferences(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(preferences) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d preferences", len(preferences))
	}
}

func TestLoadPreferences_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `invalid json
{"timestamp":1000,"category":"valid","key":"test_key","value":"test_value","reason":"test","scope":"session"}
{incomplete`
	os.WriteFile(path, []byte(data), 0644)

	preferences, err := loadPreferences(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(preferences) != 1 {
		t.Errorf("Expected 1 preference (skipped malformed), got: %d", len(preferences))
	}

	if preferences[0].Key != "test_key" {
		t.Errorf("Expected valid preference to be parsed, got: %s", preferences[0].Key)
	}
}

// ===== Tests for PerformanceMetric loader (GOgent-029c) =====

func TestLoadPerformance_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "performance.jsonl")

	data := `{"timestamp":1000,"operation":"handoff_generation","duration_ms":150,"memory_bytes":1024000,"success":true,"context":"full handoff"}
{"timestamp":1100,"operation":"validation","duration_ms":25,"memory_bytes":512000,"success":false,"context":"schema error"}`
	os.WriteFile(path, []byte(data), 0644)

	metrics, err := loadPerformance(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got: %d", len(metrics))
	}

	// Verify first metric
	if metrics[0].Timestamp != 1000 {
		t.Errorf("Expected Timestamp 1000, got: %d", metrics[0].Timestamp)
	}
	if metrics[0].Operation != "handoff_generation" {
		t.Errorf("Expected Operation 'handoff_generation', got: %s", metrics[0].Operation)
	}
	if metrics[0].DurationMs != 150 {
		t.Errorf("Expected DurationMs 150, got: %d", metrics[0].DurationMs)
	}
	if metrics[0].MemoryBytes != 1024000 {
		t.Errorf("Expected MemoryBytes 1024000, got: %d", metrics[0].MemoryBytes)
	}
	if !metrics[0].Success {
		t.Errorf("Expected Success true, got: false")
	}
	if metrics[0].Context != "full handoff" {
		t.Errorf("Expected Context 'full handoff', got: %s", metrics[0].Context)
	}

	// Verify second metric
	if metrics[1].Operation != "validation" {
		t.Errorf("Expected Operation 'validation', got: %s", metrics[1].Operation)
	}
	if metrics[1].Success {
		t.Errorf("Expected Success false, got: true")
	}
}

func TestLoadPerformance_MissingFile(t *testing.T) {
	metrics, err := loadPerformance("/tmp/nonexistent-performance.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(metrics) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d metrics", len(metrics))
	}
}

func TestLoadPerformance_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	metrics, err := loadPerformance(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(metrics) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d metrics", len(metrics))
	}
}

func TestLoadPerformance_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `bad json
{"timestamp":1000,"operation":"valid_op","duration_ms":100,"memory_bytes":500,"success":true,"context":"test"}
{incomplete`
	os.WriteFile(path, []byte(data), 0644)

	metrics, err := loadPerformance(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric (skipped malformed), got: %d", len(metrics))
	}

	if metrics[0].Operation != "valid_op" {
		t.Errorf("Expected valid metric to be parsed, got: %s", metrics[0].Operation)
	}
}

// ===== Tests for LoadArtifacts integration with new loaders (GOgent-029c) =====

func TestLoadArtifacts_AllMissing_IncludesNewFields(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir (avoid /tmp pollution)
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Errorf("Expected no error for missing files, got: %v", err)
	}

	// Verify new fields are initialized to empty slices (not nil)
	if artifacts.Decisions == nil {
		t.Errorf("Expected Decisions to be non-nil empty slice")
	}
	if len(artifacts.Decisions) != 0 {
		t.Errorf("Expected empty Decisions, got: %d", len(artifacts.Decisions))
	}

	if artifacts.PreferenceOverrides == nil {
		t.Errorf("Expected PreferenceOverrides to be non-nil empty slice")
	}
	if len(artifacts.PreferenceOverrides) != 0 {
		t.Errorf("Expected empty PreferenceOverrides, got: %d", len(artifacts.PreferenceOverrides))
	}

	if artifacts.PerformanceMetrics == nil {
		t.Errorf("Expected PerformanceMetrics to be non-nil empty slice")
	}
	if len(artifacts.PerformanceMetrics) != 0 {
		t.Errorf("Expected empty PerformanceMetrics, got: %d", len(artifacts.PerformanceMetrics))
	}
}

func TestLoadArtifacts_Complete_IncludesNewFields(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir (avoid /tmp pollution)
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	// Create existing artifact files
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	pendingData := `{"file":"test.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1000}`
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	// Create new artifact files
	decisionsData := `{"timestamp":1000,"category":"arch","decision":"test","rationale":"test","alternatives":"none","impact":"low"}
{"timestamp":1100,"category":"tool","decision":"test2","rationale":"test2","alternatives":"none","impact":"medium"}`
	os.WriteFile(config.DecisionsPath, []byte(decisionsData), 0644)

	preferencesData := `{"timestamp":1000,"category":"routing","key":"tier","value":"sonnet","reason":"quality","scope":"project"}`
	os.WriteFile(config.PreferencesPath, []byte(preferencesData), 0644)

	performanceData := `{"timestamp":1000,"operation":"gen","duration_ms":100,"memory_bytes":1000,"success":true,"context":"test"}
{"timestamp":1100,"operation":"val","duration_ms":50,"memory_bytes":500,"success":true,"context":"test"}
{"timestamp":1200,"operation":"load","duration_ms":25,"memory_bytes":250,"success":false,"context":"error"}`
	os.WriteFile(config.PerformancePath, []byte(performanceData), 0644)

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify existing fields still work
	if len(artifacts.SharpEdges) != 1 {
		t.Errorf("Expected 1 SharpEdge, got: %d", len(artifacts.SharpEdges))
	}

	// Verify new fields
	if len(artifacts.Decisions) != 2 {
		t.Errorf("Expected 2 Decisions, got: %d", len(artifacts.Decisions))
	}

	if len(artifacts.PreferenceOverrides) != 1 {
		t.Errorf("Expected 1 PreferenceOverride, got: %d", len(artifacts.PreferenceOverrides))
	}

	if len(artifacts.PerformanceMetrics) != 3 {
		t.Errorf("Expected 3 PerformanceMetrics, got: %d", len(artifacts.PerformanceMetrics))
	}
}

// ===== Backward compatibility test (GOgent-029c) =====

func TestLoadArtifacts_BackwardCompatibility_V10(t *testing.T) {
	// This test verifies that a v1.0 handoff (without new fields) can still be loaded
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	// Create only the v1.0 files (no decisions, preferences, or performance)
	os.MkdirAll(filepath.Dir(config.PendingPath), 0755)
	pendingData := `{"file":"legacy.go","error_type":"old_error","consecutive_failures":3,"timestamp":1000}`
	os.WriteFile(config.PendingPath, []byte(pendingData), 0644)

	violationsData := `{"agent":"old-agent","violation_type":"tier_mismatch","timestamp":1000}`
	os.WriteFile(config.ViolationsPath, []byte(violationsData), 0644)

	// New files don't exist (simulating v1.0 state)

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Fatalf("Expected no error loading v1.0 artifacts, got: %v", err)
	}

	// Verify v1.0 fields loaded correctly
	if len(artifacts.SharpEdges) != 1 {
		t.Errorf("Expected 1 SharpEdge, got: %d", len(artifacts.SharpEdges))
	}
	if len(artifacts.RoutingViolations) != 1 {
		t.Errorf("Expected 1 RoutingViolation, got: %d", len(artifacts.RoutingViolations))
	}

	// Verify new fields are empty slices (not nil) - critical for JSON serialization
	if artifacts.Decisions == nil {
		t.Errorf("Decisions should be empty slice, not nil")
	}
	if len(artifacts.Decisions) != 0 {
		t.Errorf("Expected 0 Decisions for v1.0, got: %d", len(artifacts.Decisions))
	}

	if artifacts.PreferenceOverrides == nil {
		t.Errorf("PreferenceOverrides should be empty slice, not nil")
	}
	if len(artifacts.PreferenceOverrides) != 0 {
		t.Errorf("Expected 0 PreferenceOverrides for v1.0, got: %d", len(artifacts.PreferenceOverrides))
	}

	if artifacts.PerformanceMetrics == nil {
		t.Errorf("PerformanceMetrics should be empty slice, not nil")
	}
	if len(artifacts.PerformanceMetrics) != 0 {
		t.Errorf("Expected 0 PerformanceMetrics for v1.0, got: %d", len(artifacts.PerformanceMetrics))
	}
}

// ===== DefaultHandoffConfig tests for new paths (GOgent-029c) =====

func TestDefaultHandoffConfig_NewPaths(t *testing.T) {
	projectDir := "/test/project"
	config := DefaultHandoffConfig(projectDir)

	expectedDecisionsPath := "/test/project/.gogent/memory/decisions.jsonl"
	if config.DecisionsPath != expectedDecisionsPath {
		t.Errorf("Expected DecisionsPath '%s', got: '%s'", expectedDecisionsPath, config.DecisionsPath)
	}

	expectedPreferencesPath := "/test/project/.gogent/memory/preferences.jsonl"
	if config.PreferencesPath != expectedPreferencesPath {
		t.Errorf("Expected PreferencesPath '%s', got: '%s'", expectedPreferencesPath, config.PreferencesPath)
	}

	expectedPerformancePath := "/test/project/.gogent/memory/performance.jsonl"
	if config.PerformancePath != expectedPerformancePath {
		t.Errorf("Expected PerformancePath '%s', got: '%s'", expectedPerformancePath, config.PerformancePath)
	}
}

// ===== Tests for LoadAllUserIntents (GOgent-041c) =====

func TestLoadAllUserIntents_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "user-intents.jsonl")

	data := `{"timestamp":1000,"question":"Fix bug?","response":"Yes","confidence":"explicit","source":"ask_user","category":"bug_fix","keywords":["bug","fix"],"honored":true,"outcome_note":"completed"}
{"timestamp":1100,"question":"Add feature?","response":"Add X","confidence":"explicit","source":"ask_user","category":"feature_request","keywords":["feature","add"],"honored":false,"outcome_note":"in_progress"}`
	os.WriteFile(path, []byte(data), 0644)

	intents, err := LoadAllUserIntents(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 intents, got: %d", len(intents))
	}

	// Verify first intent
	if intents[0].Timestamp != 1000 {
		t.Errorf("Expected Timestamp 1000, got: %d", intents[0].Timestamp)
	}
	if intents[0].Question != "Fix bug?" {
		t.Errorf("Expected Question 'Fix bug?', got: %s", intents[0].Question)
	}
	if intents[0].Response != "Yes" {
		t.Errorf("Expected Response 'Yes', got: %s", intents[0].Response)
	}
	if intents[0].Category != "bug_fix" {
		t.Errorf("Expected Category 'bug_fix', got: %s", intents[0].Category)
	}
	if len(intents[0].Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got: %d", len(intents[0].Keywords))
	}
	if intents[0].Keywords[0] != "bug" || intents[0].Keywords[1] != "fix" {
		t.Errorf("Expected keywords ['bug', 'fix'], got: %v", intents[0].Keywords)
	}
	if intents[0].Honored == nil || !*intents[0].Honored {
		t.Errorf("Expected Honored true, got: %v", intents[0].Honored)
	}
	if intents[0].OutcomeNote != "completed" {
		t.Errorf("Expected OutcomeNote 'completed', got: %s", intents[0].OutcomeNote)
	}

	// Verify second intent
	if intents[1].Question != "Add feature?" {
		t.Errorf("Expected Question 'Add feature?', got: %s", intents[1].Question)
	}
	if intents[1].Category != "feature_request" {
		t.Errorf("Expected Category 'feature_request', got: %s", intents[1].Category)
	}
	if intents[1].Honored == nil || *intents[1].Honored {
		t.Errorf("Expected Honored false, got: %v", intents[1].Honored)
	}
}

func TestLoadAllUserIntents_MissingFile(t *testing.T) {
	intents, err := LoadAllUserIntents("/tmp/nonexistent-intents.jsonl")

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(intents) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d intents", len(intents))
	}
}

func TestLoadAllUserIntents_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	intents, err := LoadAllUserIntents(path)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(intents) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d intents", len(intents))
	}
}

func TestLoadAllUserIntents_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")
	data := `not json
{"timestamp":1000,"question":"Valid?","response":"Yes","confidence":"explicit","source":"ask_user","category":"test","keywords":["test"]}
{incomplete json`
	os.WriteFile(path, []byte(data), 0644)

	intents, err := LoadAllUserIntents(path)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(intents) != 1 {
		t.Errorf("Expected 1 intent (skipped malformed), got: %d", len(intents))
	}

	if intents[0].Question != "Valid?" {
		t.Errorf("Expected valid intent to be parsed, got: %s", intents[0].Question)
	}
}

func TestLoadAllUserIntents_MinimalFields(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "minimal.jsonl")
	data := `{"timestamp":1000,"question":"Minimal?","response":"Yes","confidence":"explicit","source":"ask_user"}`
	os.WriteFile(path, []byte(data), 0644)

	intents, err := LoadAllUserIntents(path)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 1 {
		t.Errorf("Expected 1 intent, got: %d", len(intents))
	}

	// Verify minimal fields work (category, keywords, honored, outcome_note are optional)
	if intents[0].Question != "Minimal?" {
		t.Errorf("Expected Question 'Minimal?', got: %s", intents[0].Question)
	}
	if intents[0].Category != "" {
		t.Errorf("Expected empty Category, got: %s", intents[0].Category)
	}
	if intents[0].Honored != nil {
		t.Errorf("Expected nil Honored, got: %v", intents[0].Honored)
	}
	if intents[0].OutcomeNote != "" {
		t.Errorf("Expected empty OutcomeNote, got: %s", intents[0].OutcomeNote)
	}
}

// ===== Tests for loadEndstates (GOgent-073) =====

func TestLoadEndstates_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	endstatesPath := filepath.Join(projectDir, ".gogent", "memory", "agent-endstates.jsonl")

	// Create directory structure
	os.MkdirAll(filepath.Dir(endstatesPath), 0755)

	// Create valid endstate logs
	data := `{"timestamp":"2026-01-24T10:00:00Z","agent_id":"python-pro","agent_class":"implementation","tier":"sonnet","exit_code":0,"duration_ms":1500,"output_tokens":500,"decision":"prompt","recommendations":["Review type hints","Consider error handling"]}
{"timestamp":"2026-01-24T11:00:00Z","agent_id":"codebase-search","agent_class":"search","tier":"haiku","exit_code":0,"duration_ms":250,"output_tokens":100,"decision":"silent","recommendations":[]}`
	os.WriteFile(endstatesPath, []byte(data), 0644)

	endstates, err := loadEndstates(projectDir)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(endstates) != 2 {
		t.Errorf("Expected 2 endstates, got: %d", len(endstates))
	}

	// Verify first endstate
	if endstates[0].AgentID != "python-pro" {
		t.Errorf("Expected AgentID 'python-pro', got: %s", endstates[0].AgentID)
	}
	if endstates[0].AgentClass != "implementation" {
		t.Errorf("Expected AgentClass 'implementation', got: %s", endstates[0].AgentClass)
	}
	if endstates[0].Tier != "sonnet" {
		t.Errorf("Expected Tier 'sonnet', got: %s", endstates[0].Tier)
	}
	if endstates[0].ExitCode != 0 {
		t.Errorf("Expected ExitCode 0, got: %d", endstates[0].ExitCode)
	}
	if endstates[0].Duration != 1500 {
		t.Errorf("Expected Duration 1500, got: %d", endstates[0].Duration)
	}
	if endstates[0].OutputTokens != 500 {
		t.Errorf("Expected OutputTokens 500, got: %d", endstates[0].OutputTokens)
	}
	if endstates[0].Decision != "prompt" {
		t.Errorf("Expected Decision 'prompt', got: %s", endstates[0].Decision)
	}
	if len(endstates[0].Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got: %d", len(endstates[0].Recommendations))
	}
	if endstates[0].Recommendations[0] != "Review type hints" {
		t.Errorf("Expected first recommendation 'Review type hints', got: %s", endstates[0].Recommendations[0])
	}

	// Verify second endstate
	if endstates[1].AgentID != "codebase-search" {
		t.Errorf("Expected AgentID 'codebase-search', got: %s", endstates[1].AgentID)
	}
	if endstates[1].Decision != "silent" {
		t.Errorf("Expected Decision 'silent', got: %s", endstates[1].Decision)
	}
	if len(endstates[1].Recommendations) != 0 {
		t.Errorf("Expected 0 recommendations, got: %d", len(endstates[1].Recommendations))
	}
}

func TestLoadEndstates_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "nonexistent-project")

	endstates, err := loadEndstates(projectDir)

	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(endstates) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d endstates", len(endstates))
	}
}

func TestLoadEndstates_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	endstatesPath := filepath.Join(projectDir, ".gogent", "memory", "agent-endstates.jsonl")

	os.MkdirAll(filepath.Dir(endstatesPath), 0755)
	os.WriteFile(endstatesPath, []byte(""), 0644)

	endstates, err := loadEndstates(projectDir)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(endstates) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d endstates", len(endstates))
	}
}

func TestLoadEndstates_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	endstatesPath := filepath.Join(projectDir, ".gogent", "memory", "agent-endstates.jsonl")

	os.MkdirAll(filepath.Dir(endstatesPath), 0755)

	// Mix valid and invalid JSONL lines
	data := `not valid json
{"timestamp":"2026-01-24T10:00:00Z","agent_id":"valid-agent","agent_class":"test","tier":"haiku","exit_code":0,"duration_ms":100,"output_tokens":50,"decision":"silent","recommendations":[]}
{incomplete json`
	os.WriteFile(endstatesPath, []byte(data), 0644)

	endstates, err := loadEndstates(projectDir)

	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid line
	if len(endstates) != 1 {
		t.Errorf("Expected 1 endstate (skipped malformed), got: %d", len(endstates))
	}

	if endstates[0].AgentID != "valid-agent" {
		t.Errorf("Expected valid endstate to be parsed, got: %s", endstates[0].AgentID)
	}
}

func TestLoadEndstates_WithBlankLines(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	endstatesPath := filepath.Join(projectDir, ".gogent", "memory", "agent-endstates.jsonl")

	os.MkdirAll(filepath.Dir(endstatesPath), 0755)

	// Include blank lines and whitespace
	data := `
{"timestamp":"2026-01-24T10:00:00Z","agent_id":"agent-1","agent_class":"test","tier":"haiku","exit_code":0,"duration_ms":100,"output_tokens":50,"decision":"silent","recommendations":[]}

{"timestamp":"2026-01-24T11:00:00Z","agent_id":"agent-2","agent_class":"test","tier":"sonnet","exit_code":0,"duration_ms":200,"output_tokens":100,"decision":"prompt","recommendations":[]}

`
	os.WriteFile(endstatesPath, []byte(data), 0644)

	endstates, err := loadEndstates(projectDir)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(endstates) != 2 {
		t.Errorf("Expected 2 endstates (skipped blank lines), got: %d", len(endstates))
	}
}

// ===== Tests for LoadArtifacts integration with AgentEndstates (GOgent-073) =====

func TestLoadArtifacts_WithEndstates(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	// Create endstates file
	endstatesPath := filepath.Join(tmpDir, ".gogent", "memory", "agent-endstates.jsonl")
	os.MkdirAll(filepath.Dir(endstatesPath), 0755)
	endstatesData := `{"timestamp":"2026-01-24T10:00:00Z","agent_id":"python-pro","agent_class":"implementation","tier":"sonnet","exit_code":0,"duration_ms":1500,"output_tokens":500,"decision":"prompt","recommendations":["test"]}
{"timestamp":"2026-01-24T11:00:00Z","agent_id":"codebase-search","agent_class":"search","tier":"haiku","exit_code":0,"duration_ms":250,"output_tokens":100,"decision":"silent","recommendations":[]}`
	os.WriteFile(endstatesPath, []byte(endstatesData), 0644)

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(artifacts.AgentEndstates) != 2 {
		t.Errorf("Expected 2 AgentEndstates, got: %d", len(artifacts.AgentEndstates))
	}

	// Verify endstates loaded correctly
	if artifacts.AgentEndstates[0].AgentID != "python-pro" {
		t.Errorf("Expected first endstate AgentID 'python-pro', got: %s", artifacts.AgentEndstates[0].AgentID)
	}
	if artifacts.AgentEndstates[1].AgentID != "codebase-search" {
		t.Errorf("Expected second endstate AgentID 'codebase-search', got: %s", artifacts.AgentEndstates[1].AgentID)
	}
}

func TestLoadArtifacts_NoEndstates(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultHandoffConfig(tmpDir)

	// Override paths to use tmpDir
	config.ViolationsPath = filepath.Join(tmpDir, "violations.jsonl")
	config.ErrorPatternsPath = filepath.Join(tmpDir, "errors.jsonl")

	// Don't create endstates file - simulate missing file

	artifacts, err := LoadArtifacts(config)

	if err != nil {
		t.Fatalf("Expected no error for missing endstates, got: %v", err)
	}

	// Verify AgentEndstates is empty slice (not nil)
	if artifacts.AgentEndstates == nil {
		t.Errorf("Expected AgentEndstates to be non-nil empty slice")
	}
	if len(artifacts.AgentEndstates) != 0 {
		t.Errorf("Expected 0 AgentEndstates for missing file, got: %d", len(artifacts.AgentEndstates))
	}
}
