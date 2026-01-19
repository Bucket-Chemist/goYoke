package session

import (
	"os"
	"path/filepath"
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
