package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewQuery(t *testing.T) {
	q := NewQuery("/test/project")
	if q.ProjectDir != "/test/project" {
		t.Errorf("Expected ProjectDir '/test/project', got: %s", q.ProjectDir)
	}
}

func TestQuerySharpEdges_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"src/main.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1705000000}
{"file":"pkg/utils.go","error_type":"nil_pointer","consecutive_failures":4,"timestamp":1705000001,"severity":"high"}`

	if err := os.WriteFile(edgesPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 2 {
		t.Errorf("Expected 2 edges, got: %d", len(edges))
	}
}

func TestQuerySharpEdges_FileFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"src/main.go","error_type":"err1","consecutive_failures":3,"timestamp":1705000000}
{"file":"src/utils.go","error_type":"err2","consecutive_failures":3,"timestamp":1705000001}
{"file":"pkg/handler.go","error_type":"err3","consecutive_failures":3,"timestamp":1705000002}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)

	// Test prefix pattern
	pattern := "src/*"
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{File: &pattern})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges with src/ prefix, got: %d", len(edges))
	}

	// Test suffix pattern
	pattern = "*.go"
	edges, err = q.QuerySharpEdges(SharpEdgeFilters{File: &pattern})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(edges) != 3 {
		t.Errorf("Expected 3 edges with .go suffix, got: %d", len(edges))
	}

	// Test contains pattern
	pattern = "*utils*"
	edges, err = q.QuerySharpEdges(SharpEdgeFilters{File: &pattern})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge containing 'utils', got: %d", len(edges))
	}

	// Test exact match
	pattern = "pkg/handler.go"
	edges, err = q.QuerySharpEdges(SharpEdgeFilters{File: &pattern})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge with exact match, got: %d", len(edges))
	}
}

func TestQuerySharpEdges_ErrorTypeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"a.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000000}
{"file":"b.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1705000001}
{"file":"c.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000002}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	errorType := "nil_pointer"
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{ErrorType: &errorType})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 2 {
		t.Errorf("Expected 2 nil_pointer edges, got: %d", len(edges))
	}
}

func TestQuerySharpEdges_SeverityFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"a.go","error_type":"err1","consecutive_failures":3,"timestamp":1705000000,"severity":"high"}
{"file":"b.go","error_type":"err2","consecutive_failures":3,"timestamp":1705000001,"severity":"low"}
{"file":"c.go","error_type":"err3","consecutive_failures":3,"timestamp":1705000002,"severity":"high"}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	severity := "high"
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{Severity: &severity})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 2 {
		t.Errorf("Expected 2 high-severity edges, got: %d", len(edges))
	}
}

func TestQuerySharpEdges_UnresolvedFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"a.go","error_type":"err1","consecutive_failures":3,"timestamp":1705000000,"resolved_at":0}
{"file":"b.go","error_type":"err2","consecutive_failures":3,"timestamp":1705000001,"resolved_at":1705000100}
{"file":"c.go","error_type":"err3","consecutive_failures":3,"timestamp":1705000002}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{Unresolved: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// a.go (resolved_at=0) and c.go (resolved_at missing = 0) should match
	if len(edges) != 2 {
		t.Errorf("Expected 2 unresolved edges, got: %d", len(edges))
	}
}

func TestQuerySharpEdges_SinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	now := time.Now().Unix()
	oldTimestamp := now - (30 * 24 * 60 * 60) // 30 days ago
	recentTimestamp := now - (5 * 24 * 60 * 60) // 5 days ago

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"old.go","error_type":"err","consecutive_failures":3,"timestamp":` + itoa(oldTimestamp) + `}
{"file":"recent.go","error_type":"err","consecutive_failures":3,"timestamp":` + itoa(recentTimestamp) + `}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	since := now - (7 * 24 * 60 * 60) // 7 days ago
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{Since: &since})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 1 {
		t.Errorf("Expected 1 recent edge, got: %d", len(edges))
	}
	if len(edges) > 0 && edges[0].File != "recent.go" {
		t.Errorf("Expected recent.go, got: %s", edges[0].File)
	}
}

func TestQuerySharpEdges_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"a.go","error_type":"err","consecutive_failures":3,"timestamp":1705000000}
{"file":"b.go","error_type":"err","consecutive_failures":3,"timestamp":1705000001}
{"file":"c.go","error_type":"err","consecutive_failures":3,"timestamp":1705000002}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 2 {
		t.Errorf("Expected 2 edges (limit), got: %d", len(edges))
	}
}

func TestQuerySharpEdges_CombinedFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"src/a.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000000,"severity":"high"}
{"file":"src/b.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000001,"severity":"low"}
{"file":"pkg/c.go","error_type":"nil_pointer","consecutive_failures":3,"timestamp":1705000002,"severity":"high"}
{"file":"src/d.go","error_type":"type_mismatch","consecutive_failures":3,"timestamp":1705000003,"severity":"high"}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	filePattern := "src/*"
	errorType := "nil_pointer"
	severity := "high"
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{
		File:      &filePattern,
		ErrorType: &errorType,
		Severity:  &severity,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Only src/a.go matches all criteria
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge matching all filters, got: %d", len(edges))
	}
	if len(edges) > 0 && edges[0].File != "src/a.go" {
		t.Errorf("Expected src/a.go, got: %s", edges[0].File)
	}
}

func TestQuerySharpEdges_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d edges", len(edges))
	}
}

func TestQuerySharpEdges_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	os.WriteFile(edgesPath, []byte(""), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d edges", len(edges))
	}
}

func TestQuerySharpEdges_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `not valid json
{"file":"valid.go","error_type":"test","consecutive_failures":3,"timestamp":1705000000}
{broken json
{"file":"also_valid.go","error_type":"test","consecutive_failures":3,"timestamp":1705000001}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	// Should only have the valid lines
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges (skipped malformed), got: %d", len(edges))
	}
}

func TestQuerySharpEdges_BackwardCompatibility(t *testing.T) {
	// Test that old JSONL format (without new fields) still parses
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	// Old format without ErrorMessage, Severity, Resolution, ResolvedAt
	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"old_format.go","error_type":"legacy_error","consecutive_failures":5,"context":"some context","timestamp":1705000000}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got: %d", len(edges))
	}

	edge := edges[0]
	if edge.File != "old_format.go" {
		t.Errorf("Expected file 'old_format.go', got: %s", edge.File)
	}
	if edge.ErrorMessage != "" {
		t.Errorf("Expected empty ErrorMessage for old format, got: %s", edge.ErrorMessage)
	}
	if edge.Severity != "" {
		t.Errorf("Expected empty Severity for old format, got: %s", edge.Severity)
	}
	if edge.Resolution != "" {
		t.Errorf("Expected empty Resolution for old format, got: %s", edge.Resolution)
	}
	if edge.ResolvedAt != 0 {
		t.Errorf("Expected zero ResolvedAt for old format, got: %d", edge.ResolvedAt)
	}
}

func TestQuerySharpEdges_NewFieldsParsed(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	edgesPath := filepath.Join(claudeDir, "pending-learnings.jsonl")
	content := `{"file":"new_format.go","error_type":"test_error","consecutive_failures":4,"context":"ctx","timestamp":1705000000,"error_message":"full error text here","severity":"high","resolution":"fixed by adding nil check","resolved_at":1705000100}`

	os.WriteFile(edgesPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	edges, err := q.QuerySharpEdges(SharpEdgeFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got: %d", len(edges))
	}

	edge := edges[0]
	if edge.ErrorMessage != "full error text here" {
		t.Errorf("Expected ErrorMessage 'full error text here', got: %s", edge.ErrorMessage)
	}
	if edge.Severity != "high" {
		t.Errorf("Expected Severity 'high', got: %s", edge.Severity)
	}
	if edge.Resolution != "fixed by adding nil check" {
		t.Errorf("Expected Resolution 'fixed by adding nil check', got: %s", edge.Resolution)
	}
	if edge.ResolvedAt != 1705000100 {
		t.Errorf("Expected ResolvedAt 1705000100, got: %d", edge.ResolvedAt)
	}
}

// matchGlob tests
func TestMatchGlob_Empty(t *testing.T) {
	if !matchGlob("anything", "") {
		t.Error("Empty pattern should match anything")
	}
	if !matchGlob("anything", "*") {
		t.Error("* pattern should match anything")
	}
}

func TestMatchGlob_Prefix(t *testing.T) {
	if !matchGlob("src/main.go", "src/*") {
		t.Error("src/* should match src/main.go")
	}
	if matchGlob("pkg/main.go", "src/*") {
		t.Error("src/* should not match pkg/main.go")
	}
}

func TestMatchGlob_Suffix(t *testing.T) {
	if !matchGlob("main.go", "*.go") {
		t.Error("*.go should match main.go")
	}
	if matchGlob("main.py", "*.go") {
		t.Error("*.go should not match main.py")
	}
}

func TestMatchGlob_Contains(t *testing.T) {
	if !matchGlob("src/utils/helper.go", "*utils*") {
		t.Error("*utils* should match src/utils/helper.go")
	}
	if matchGlob("src/main.go", "*utils*") {
		t.Error("*utils* should not match src/main.go")
	}
}

func TestMatchGlob_Exact(t *testing.T) {
	if !matchGlob("main.go", "main.go") {
		t.Error("Exact match should work")
	}
	if matchGlob("main.go", "other.go") {
		t.Error("Non-matching exact should fail")
	}
}

// FormatSharpEdge tests
func TestFormatSharpEdge_MinimalFields(t *testing.T) {
	edge := SharpEdge{
		File:                "pkg/utils.go",
		ErrorType:           "nil_pointer",
		ConsecutiveFailures: 3,
		Timestamp:           1705000000,
	}

	formatted := FormatSharpEdge(edge)

	// Should NOT have severity badge (no severity set)
	if strings.Contains(formatted, "🔴") || strings.Contains(formatted, "🟡") || strings.Contains(formatted, "🟢") {
		t.Error("Expected no severity badge for minimal format")
	}

	// Should have basic format
	if !strings.Contains(formatted, "**pkg/utils.go**") {
		t.Error("Expected bold file name")
	}
	if !strings.Contains(formatted, "nil_pointer") {
		t.Error("Expected error type")
	}
	if !strings.Contains(formatted, "(3 failures)") {
		t.Error("Expected failure count")
	}

	// Should NOT have error message or resolution
	if strings.Contains(formatted, "Error:") {
		t.Error("Expected no error message for minimal format")
	}
	if strings.Contains(formatted, "Resolved:") {
		t.Error("Expected no resolution for minimal format")
	}
}

func TestFormatSharpEdge_AllFields(t *testing.T) {
	edge := SharpEdge{
		File:                "src/main.go",
		ErrorType:           "type_mismatch",
		ConsecutiveFailures: 5,
		Context:             "test context",
		Timestamp:           1705000000,
		ErrorMessage:        "invalid type assertion: expected int, got string",
		Severity:            "high",
		Resolution:          "Added type check before assertion",
		ResolvedAt:          1705000100,
	}

	formatted := FormatSharpEdge(edge)

	// Should have severity badge
	if !strings.Contains(formatted, "🔴") {
		t.Error("Expected high severity badge")
	}

	// Should have file and error type
	if !strings.Contains(formatted, "**src/main.go**") {
		t.Error("Expected bold file name")
	}
	if !strings.Contains(formatted, "type_mismatch") {
		t.Error("Expected error type")
	}

	// Should have error message
	if !strings.Contains(formatted, "Error: `invalid type assertion") {
		t.Error("Expected error message")
	}

	// Should have resolution
	if !strings.Contains(formatted, "✅ Resolved:") {
		t.Error("Expected resolution marker")
	}
}

func TestFormatSharpEdge_SeverityBadges(t *testing.T) {
	tests := []struct {
		severity string
		badge    string
	}{
		{"high", "🔴"},
		{"medium", "🟡"},
		{"low", "🟢"},
		{"unknown", "⚪"},
	}

	for _, tc := range tests {
		edge := SharpEdge{
			File:                "test.go",
			ErrorType:           "test",
			ConsecutiveFailures: 3,
			Timestamp:           1705000000,
			Severity:            tc.severity,
		}

		formatted := FormatSharpEdge(edge)
		if !strings.Contains(formatted, tc.badge) {
			t.Errorf("Expected %s badge for severity %s, got: %s", tc.badge, tc.severity, formatted)
		}
	}
}

func TestFormatSharpEdge_ErrorMessageTruncation(t *testing.T) {
	// Create error message longer than 100 chars
	longError := strings.Repeat("x", 150)

	edge := SharpEdge{
		File:                "test.go",
		ErrorType:           "test",
		ConsecutiveFailures: 3,
		Timestamp:           1705000000,
		ErrorMessage:        longError,
	}

	formatted := FormatSharpEdge(edge)

	// Should be truncated with ...
	if !strings.Contains(formatted, "...") {
		t.Error("Expected truncation indicator ...")
	}

	// Should not contain full error
	if strings.Contains(formatted, longError) {
		t.Error("Full error message should be truncated")
	}

	// Should contain truncated version (100 chars + ...)
	truncated := longError[:100] + "..."
	if !strings.Contains(formatted, truncated) {
		t.Error("Expected truncated error message")
	}
}

func TestFormatSharpEdges_Empty(t *testing.T) {
	formatted := FormatSharpEdges([]SharpEdge{})
	if formatted != "" {
		t.Errorf("Expected empty string for empty edges, got: %s", formatted)
	}
}

func TestFormatSharpEdges_Multiple(t *testing.T) {
	edges := []SharpEdge{
		{
			File:                "a.go",
			ErrorType:           "err1",
			ConsecutiveFailures: 3,
			Timestamp:           1705000000,
			Severity:            "high",
		},
		{
			File:                "b.go",
			ErrorType:           "err2",
			ConsecutiveFailures: 4,
			Timestamp:           1705000001,
		},
	}

	formatted := FormatSharpEdges(edges)

	// Should have header
	if !strings.Contains(formatted, "## Sharp Edges") {
		t.Error("Expected Sharp Edges header")
	}

	// Should have both edges
	if !strings.Contains(formatted, "**a.go**") {
		t.Error("Expected first edge file")
	}
	if !strings.Contains(formatted, "**b.go**") {
		t.Error("Expected second edge file")
	}
}

// Helper function to convert int64 to string for test data
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ========== DECISIONS QUERY TESTS ==========

func TestQueryDecisions_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `{"timestamp":1705000000,"category":"architecture","decision":"Use JSONL","rationale":"Append-only","alternatives":"SQLite","impact":"high"}
{"timestamp":1705000001,"category":"tooling","decision":"Use Go test","rationale":"Standard","alternatives":"Ginkgo","impact":"medium"}`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	decisions, err := q.QueryDecisions(DecisionFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 2 {
		t.Errorf("Expected 2 decisions, got: %d", len(decisions))
	}
}

func TestQueryDecisions_CategoryFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `{"timestamp":1705000000,"category":"architecture","decision":"d1","rationale":"r1","alternatives":"","impact":"high"}
{"timestamp":1705000001,"category":"tooling","decision":"d2","rationale":"r2","alternatives":"","impact":"medium"}
{"timestamp":1705000002,"category":"architecture","decision":"d3","rationale":"r3","alternatives":"","impact":"low"}`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	category := "architecture"
	decisions, err := q.QueryDecisions(DecisionFilters{Category: &category})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 2 {
		t.Errorf("Expected 2 architecture decisions, got: %d", len(decisions))
	}
}

func TestQueryDecisions_ImpactFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `{"timestamp":1705000000,"category":"a","decision":"d1","rationale":"r1","alternatives":"","impact":"high"}
{"timestamp":1705000001,"category":"b","decision":"d2","rationale":"r2","alternatives":"","impact":"low"}
{"timestamp":1705000002,"category":"c","decision":"d3","rationale":"r3","alternatives":"","impact":"high"}`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	impact := "high"
	decisions, err := q.QueryDecisions(DecisionFilters{Impact: &impact})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 2 {
		t.Errorf("Expected 2 high-impact decisions, got: %d", len(decisions))
	}
}

func TestQueryDecisions_SinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	now := time.Now().Unix()
	oldTs := now - (30 * 24 * 60 * 60) // 30 days ago
	recentTs := now - (5 * 24 * 60 * 60) // 5 days ago

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `{"timestamp":` + itoa(oldTs) + `,"category":"a","decision":"old","rationale":"r","alternatives":"","impact":"high"}
{"timestamp":` + itoa(recentTs) + `,"category":"b","decision":"recent","rationale":"r","alternatives":"","impact":"low"}`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	since := now - (7 * 24 * 60 * 60) // 7 days ago
	decisions, err := q.QueryDecisions(DecisionFilters{Since: &since})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 1 {
		t.Errorf("Expected 1 recent decision, got: %d", len(decisions))
	}
	if len(decisions) > 0 && decisions[0].Decision != "recent" {
		t.Errorf("Expected 'recent' decision, got: %s", decisions[0].Decision)
	}
}

func TestQueryDecisions_LimitFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `{"timestamp":1,"category":"a","decision":"d1","rationale":"r","alternatives":"","impact":"high"}
{"timestamp":2,"category":"b","decision":"d2","rationale":"r","alternatives":"","impact":"medium"}
{"timestamp":3,"category":"c","decision":"d3","rationale":"r","alternatives":"","impact":"low"}`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	decisions, err := q.QueryDecisions(DecisionFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(decisions) != 2 {
		t.Errorf("Expected 2 decisions (limit), got: %d", len(decisions))
	}
}

func TestQueryDecisions_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := NewQuery(tmpDir)

	decisions, err := q.QueryDecisions(DecisionFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(decisions) != 0 {
		t.Errorf("Expected empty slice, got: %d", len(decisions))
	}
}

func TestQueryDecisions_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	decisionsPath := filepath.Join(claudeDir, "decisions.jsonl")
	content := `not valid json
{"timestamp":1,"category":"a","decision":"valid","rationale":"r","alternatives":"","impact":"high"}
{broken json`

	os.WriteFile(decisionsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	decisions, err := q.QueryDecisions(DecisionFilters{})
	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	if len(decisions) != 1 {
		t.Errorf("Expected 1 decision (skipped malformed), got: %d", len(decisions))
	}
}

// ========== PREFERENCES QUERY TESTS ==========

func TestQueryPreferences_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	content := `{"timestamp":1705000000,"category":"routing","key":"tier","value":"sonnet","reason":"quality","scope":"project"}
{"timestamp":1705000001,"category":"tooling","key":"test","value":"go test","reason":"standard","scope":"global"}`

	os.WriteFile(prefsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	prefs, err := q.QueryPreferences(PreferenceFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prefs) != 2 {
		t.Errorf("Expected 2 preferences, got: %d", len(prefs))
	}
}

func TestQueryPreferences_CategoryFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	content := `{"timestamp":1,"category":"routing","key":"k1","value":"v1","reason":"r1","scope":"project"}
{"timestamp":2,"category":"tooling","key":"k2","value":"v2","reason":"r2","scope":"global"}
{"timestamp":3,"category":"routing","key":"k3","value":"v3","reason":"r3","scope":"session"}`

	os.WriteFile(prefsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	category := "routing"
	prefs, err := q.QueryPreferences(PreferenceFilters{Category: &category})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prefs) != 2 {
		t.Errorf("Expected 2 routing preferences, got: %d", len(prefs))
	}
}

func TestQueryPreferences_ScopeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	content := `{"timestamp":1,"category":"a","key":"k1","value":"v1","reason":"r1","scope":"session"}
{"timestamp":2,"category":"b","key":"k2","value":"v2","reason":"r2","scope":"project"}
{"timestamp":3,"category":"c","key":"k3","value":"v3","reason":"r3","scope":"global"}`

	os.WriteFile(prefsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	scope := "project"
	prefs, err := q.QueryPreferences(PreferenceFilters{Scope: &scope})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prefs) != 1 {
		t.Errorf("Expected 1 project-scope preference, got: %d", len(prefs))
	}
	if len(prefs) > 0 && prefs[0].Key != "k2" {
		t.Errorf("Expected k2, got: %s", prefs[0].Key)
	}
}

func TestQueryPreferences_SinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	now := time.Now().Unix()
	oldTs := now - (30 * 24 * 60 * 60) // 30 days ago
	recentTs := now - (5 * 24 * 60 * 60) // 5 days ago

	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	content := `{"timestamp":` + itoa(oldTs) + `,"category":"a","key":"old","value":"v","reason":"r","scope":"project"}
{"timestamp":` + itoa(recentTs) + `,"category":"b","key":"recent","value":"v","reason":"r","scope":"global"}`

	os.WriteFile(prefsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	since := now - (7 * 24 * 60 * 60) // 7 days ago
	prefs, err := q.QueryPreferences(PreferenceFilters{Since: &since})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prefs) != 1 {
		t.Errorf("Expected 1 recent preference, got: %d", len(prefs))
	}
	if len(prefs) > 0 && prefs[0].Key != "recent" {
		t.Errorf("Expected 'recent', got: %s", prefs[0].Key)
	}
}

func TestQueryPreferences_LimitFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	prefsPath := filepath.Join(claudeDir, "preferences.jsonl")
	content := `{"timestamp":1,"category":"a","key":"k1","value":"v1","reason":"r1","scope":"session"}
{"timestamp":2,"category":"b","key":"k2","value":"v2","reason":"r2","scope":"project"}
{"timestamp":3,"category":"c","key":"k3","value":"v3","reason":"r3","scope":"global"}`

	os.WriteFile(prefsPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	prefs, err := q.QueryPreferences(PreferenceFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prefs) != 2 {
		t.Errorf("Expected 2 preferences (limit), got: %d", len(prefs))
	}
}

func TestQueryPreferences_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := NewQuery(tmpDir)

	prefs, err := q.QueryPreferences(PreferenceFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("Expected empty slice, got: %d", len(prefs))
	}
}

// ========== PERFORMANCE QUERY TESTS ==========

func TestQueryPerformance_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1705000000,"operation":"handoff","duration_ms":100,"memory_bytes":1048576,"success":true,"context":"s1"}
{"timestamp":1705000001,"operation":"validation","duration_ms":50,"memory_bytes":524288,"success":true,"context":"s2"}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	metrics, err := q.QueryPerformance(PerformanceFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got: %d", len(metrics))
	}
}

func TestQueryPerformance_OperationFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1,"operation":"handoff","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"validation","duration_ms":50,"memory_bytes":0,"success":true,"context":""}
{"timestamp":3,"operation":"handoff","duration_ms":150,"memory_bytes":0,"success":false,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	op := "handoff"
	metrics, err := q.QueryPerformance(PerformanceFilters{Operation: &op})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 handoff metrics, got: %d", len(metrics))
	}
}

func TestQueryPerformance_SlowOnlyFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	// SlowThresholdMs = 1000
	content := `{"timestamp":1,"operation":"fast","duration_ms":500,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"slow","duration_ms":1500,"memory_bytes":0,"success":true,"context":""}
{"timestamp":3,"operation":"very_slow","duration_ms":3000,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	metrics, err := q.QueryPerformance(PerformanceFilters{SlowOnly: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 slow metrics (>1000ms), got: %d", len(metrics))
	}
}

func TestQueryPerformance_SuccessOnlyFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1,"operation":"op1","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"op2","duration_ms":100,"memory_bytes":0,"success":false,"context":""}
{"timestamp":3,"operation":"op3","duration_ms":100,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	metrics, err := q.QueryPerformance(PerformanceFilters{SuccessOnly: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 success metrics, got: %d", len(metrics))
	}
}

func TestQueryPerformance_FailedOnlyFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1,"operation":"op1","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"op2","duration_ms":100,"memory_bytes":0,"success":false,"context":""}
{"timestamp":3,"operation":"op3","duration_ms":100,"memory_bytes":0,"success":false,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	metrics, err := q.QueryPerformance(PerformanceFilters{FailedOnly: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 failed metrics, got: %d", len(metrics))
	}
}

func TestQueryPerformance_SinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	now := time.Now().Unix()
	oldTs := now - (30 * 24 * 60 * 60) // 30 days ago
	recentTs := now - (5 * 24 * 60 * 60) // 5 days ago

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":` + itoa(oldTs) + `,"operation":"old","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":` + itoa(recentTs) + `,"operation":"recent","duration_ms":100,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	since := now - (7 * 24 * 60 * 60) // 7 days ago
	metrics, err := q.QueryPerformance(PerformanceFilters{Since: &since})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("Expected 1 recent metric, got: %d", len(metrics))
	}
	if len(metrics) > 0 && metrics[0].Operation != "recent" {
		t.Errorf("Expected 'recent', got: %s", metrics[0].Operation)
	}
}

func TestQueryPerformance_LimitFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1,"operation":"op1","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"op2","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":3,"operation":"op3","duration_ms":100,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	metrics, err := q.QueryPerformance(PerformanceFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics (limit), got: %d", len(metrics))
	}
}

func TestQueryPerformance_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	q := NewQuery(tmpDir)

	metrics, err := q.QueryPerformance(PerformanceFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("Expected empty slice, got: %d", len(metrics))
	}
}

// ========== PERFORMANCE SUMMARY TESTS ==========

func TestQueryPerformanceSummary_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	// handoff: 100, 200, 300 (avg 200, min 100, max 300)
	// validation: 50, 150 (avg 100, min 50, max 150)
	content := `{"timestamp":1,"operation":"handoff","duration_ms":100,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"handoff","duration_ms":200,"memory_bytes":0,"success":true,"context":""}
{"timestamp":3,"operation":"handoff","duration_ms":300,"memory_bytes":0,"success":false,"context":""}
{"timestamp":4,"operation":"validation","duration_ms":50,"memory_bytes":0,"success":true,"context":""}
{"timestamp":5,"operation":"validation","duration_ms":150,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	summaries, err := q.QueryPerformanceSummary(PerformanceFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got: %d", len(summaries))
	}

	// Find handoff summary
	var handoff *PerformanceSummary
	for i := range summaries {
		if summaries[i].Operation == "handoff" {
			handoff = &summaries[i]
			break
		}
	}

	if handoff == nil {
		t.Fatal("Expected handoff summary")
	}
	if handoff.Count != 3 {
		t.Errorf("Expected count 3, got: %d", handoff.Count)
	}
	if handoff.SuccessCount != 2 {
		t.Errorf("Expected success 2, got: %d", handoff.SuccessCount)
	}
	if handoff.FailCount != 1 {
		t.Errorf("Expected fail 1, got: %d", handoff.FailCount)
	}
	if handoff.MinMs != 100 {
		t.Errorf("Expected min 100, got: %d", handoff.MinMs)
	}
	if handoff.MaxMs != 300 {
		t.Errorf("Expected max 300, got: %d", handoff.MaxMs)
	}
	if handoff.AvgMs != 200.0 {
		t.Errorf("Expected avg 200.0, got: %.1f", handoff.AvgMs)
	}
}

func TestQueryPerformanceSummary_WithSlowFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	content := `{"timestamp":1,"operation":"fast","duration_ms":500,"memory_bytes":0,"success":true,"context":""}
{"timestamp":2,"operation":"slow","duration_ms":1500,"memory_bytes":0,"success":true,"context":""}
{"timestamp":3,"operation":"slow","duration_ms":2000,"memory_bytes":0,"success":true,"context":""}`

	os.WriteFile(perfPath, []byte(content), 0644)

	q := NewQuery(tmpDir)
	summaries, err := q.QueryPerformanceSummary(PerformanceFilters{SlowOnly: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Only slow operations should be included
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary (slow only), got: %d", len(summaries))
	}
	if len(summaries) > 0 && summaries[0].Operation != "slow" {
		t.Errorf("Expected 'slow' operation, got: %s", summaries[0].Operation)
	}
}

func TestQueryPerformanceSummary_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(claudeDir, 0755)

	perfPath := filepath.Join(claudeDir, "performance.jsonl")
	os.WriteFile(perfPath, []byte(""), 0644)

	q := NewQuery(tmpDir)
	summaries, err := q.QueryPerformanceSummary(PerformanceFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries for empty file, got: %d", len(summaries))
	}
}

func TestSlowThresholdMs_Value(t *testing.T) {
	if SlowThresholdMs != 1000 {
		t.Errorf("Expected SlowThresholdMs to be 1000, got: %d", SlowThresholdMs)
	}
}
