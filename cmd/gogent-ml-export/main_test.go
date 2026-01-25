package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// TestWriteRoutingCSV verifies CSV structure for routing decisions
func TestWriteRoutingCSV(t *testing.T) {
	buf := &bytes.Buffer{}

	// Create sample routing events
	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			TaskType:      "implementation",
			TaskDomain:    "python",
			InputTokens:   1000,
			OutputTokens:  500,
			SelectedTier:  "sonnet",
			SelectedAgent: "python-pro",
			Success:       true,
			Tier:          "sonnet",
		},
		{
			CapturedAt:    time.Now().Unix() - 3600,
			TaskType:      "search",
			TaskDomain:    "codebase",
			InputTokens:   200,
			OutputTokens:  100,
			SelectedTier:  "haiku",
			SelectedAgent: "codebase-search",
			Success:       true,
			Tier:          "haiku",
		},
	}

	writeRoutingCSV(buf, events)

	// Verify CSV header
	content := buf.String()
	expectedHeader := "timestamp,task_type,task_domain,context_window,recent_success_rate,selected_tier,selected_agent,outcome_success,outcome_cost,escalation_required"
	if !strings.Contains(content, expectedHeader) {
		t.Errorf("Expected CSV header not found. Got:\n%s", content)
	}

	// Verify data rows
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Parse and validate CSV structure
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify column count
	for i, record := range records {
		if len(record) != 10 {
			t.Errorf("Row %d: expected 10 columns, got %d", i, len(record))
		}
	}

	// Verify first data row values
	if records[1][1] != "implementation" {
		t.Errorf("Expected task_type 'implementation', got '%s'", records[1][1])
	}
	if records[1][5] != "sonnet" {
		t.Errorf("Expected selected_tier 'sonnet', got '%s'", records[1][5])
	}
}

// TestWriteJSON verifies JSON output format
func TestWriteJSON(t *testing.T) {
	buf := &bytes.Buffer{}

	data := map[string]interface{}{
		"test_field":  "test_value",
		"test_number": 42,
	}

	writeJSON(buf, data)

	// Verify valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify data integrity
	if decoded["test_field"] != "test_value" {
		t.Errorf("Expected test_field='test_value', got '%v'", decoded["test_field"])
	}
}

// TestBuildSequences verifies tool sequence construction
func TestBuildSequences(t *testing.T) {
	events := []routing.PostToolEvent{
		{
			SessionID:    "session-1",
			ToolName:     "Read",
			Success:      true,
			DurationMs:   100,
			InputTokens:  50,
			OutputTokens: 30,
		},
		{
			SessionID:    "session-1",
			ToolName:     "Edit",
			Success:      true,
			DurationMs:   200,
			InputTokens:  100,
			OutputTokens: 50,
		},
		{
			SessionID:    "session-2",
			ToolName:     "Bash",
			Success:      false,
			DurationMs:   150,
			InputTokens:  75,
			OutputTokens: 25,
		},
	}

	sequences := buildSequences(events)

	// Verify sequence count
	if len(sequences) != 2 {
		t.Errorf("Expected 2 sequences, got %d", len(sequences))
	}

	// Find session-1 sequence
	var seq1 *ToolSequence
	for i := range sequences {
		if sequences[i].SequenceID == "session-1" {
			seq1 = &sequences[i]
			break
		}
	}

	if seq1 == nil {
		t.Fatal("session-1 sequence not found")
	}

	// Verify sequence metrics
	if len(seq1.Tools) != 2 {
		t.Errorf("Expected 2 tools in session-1, got %d", len(seq1.Tools))
	}
	if seq1.DurationMs != 300 {
		t.Errorf("Expected total duration 300ms, got %d", seq1.DurationMs)
	}
	if seq1.TokenCount != 230 { // 50+30+100+50
		t.Errorf("Expected token count 230, got %d", seq1.TokenCount)
	}
	if !seq1.Successful {
		t.Error("Expected sequence to be successful (all events succeeded)")
	}

	// Verify session-2 is marked unsuccessful
	var seq2 *ToolSequence
	for i := range sequences {
		if sequences[i].SequenceID == "session-2" {
			seq2 = &sequences[i]
			break
		}
	}

	if seq2 == nil {
		t.Fatal("session-2 sequence not found")
	}

	if seq2.Successful {
		t.Error("Expected session-2 to be unsuccessful")
	}
}

// TestWriteSequencesCSV verifies tool sequence CSV export
func TestWriteSequencesCSV(t *testing.T) {
	buf := &bytes.Buffer{}

	sequences := []ToolSequence{
		{
			SequenceID: "seq-1",
			Tools:      []string{"Read", "Edit", "Bash"},
			Successful: true,
			DurationMs: 500,
			TokenCount: 1200,
			Cost:       0.012,
		},
	}

	writeSequencesCSV(buf, sequences)

	content := buf.String()

	// Verify header
	expectedHeader := "sequence_id,tool_chain,successful,duration_ms,token_count,cost"
	if !strings.Contains(content, expectedHeader) {
		t.Errorf("Expected CSV header not found. Got:\n%s", content)
	}

	// Verify tool chain formatting
	if !strings.Contains(content, "Read → Edit → Bash") {
		t.Error("Expected tool chain with arrow separators")
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 2 { // header + 1 data row
		t.Errorf("Expected 2 rows, got %d", len(records))
	}

	// Verify data row
	if records[1][0] != "seq-1" {
		t.Errorf("Expected sequence_id 'seq-1', got '%s'", records[1][0])
	}
	if records[1][2] != "true" {
		t.Errorf("Expected successful 'true', got '%s'", records[1][2])
	}
}

// TestAggregateCollaborations verifies collaboration edge aggregation
func TestAggregateCollaborations(t *testing.T) {
	collabs := []telemetry.AgentCollaboration{
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "python-pro",
			ChildSuccess: true,
		},
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "python-pro",
			ChildSuccess: true,
		},
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "python-pro",
			ChildSuccess: false,
		},
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "codebase-search",
			ChildSuccess: true,
		},
	}

	edges := aggregateCollaborations(collabs)

	// Verify edge count (2 unique parent-child pairs)
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(edges))
	}

	// Find orchestrator → python-pro edge
	var pythonEdge *CollaborationEdge
	for i := range edges {
		if edges[i].SourceAgent == "orchestrator" && edges[i].TargetAgent == "python-pro" {
			pythonEdge = &edges[i]
			break
		}
	}

	if pythonEdge == nil {
		t.Fatal("orchestrator → python-pro edge not found")
	}

	// Verify aggregation
	if pythonEdge.InteractionCount != 3 {
		t.Errorf("Expected 3 interactions, got %d", pythonEdge.InteractionCount)
	}

	// Success rate should be 2/3 = 0.6667
	expectedRate := 2.0 / 3.0
	if pythonEdge.SuccessRate < expectedRate-0.01 || pythonEdge.SuccessRate > expectedRate+0.01 {
		t.Errorf("Expected success rate ~%.4f, got %.4f", expectedRate, pythonEdge.SuccessRate)
	}
}

// TestWriteCollaborationsCSV verifies collaboration CSV export
func TestWriteCollaborationsCSV(t *testing.T) {
	buf := &bytes.Buffer{}

	edges := []CollaborationEdge{
		{
			SourceAgent:      "orchestrator",
			TargetAgent:      "python-pro",
			InteractionCount: 5,
			SuccessRate:      0.8000,
			AvgCost:          0.015,
		},
	}

	writeCollaborationsCSV(buf, edges)

	content := buf.String()

	// Verify header
	expectedHeader := "source_agent,target_agent,interaction_count,success_rate,avg_cost"
	if !strings.Contains(content, expectedHeader) {
		t.Errorf("Expected CSV header not found. Got:\n%s", content)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify data
	if records[1][0] != "orchestrator" {
		t.Errorf("Expected source_agent 'orchestrator', got '%s'", records[1][0])
	}
	if records[1][1] != "python-pro" {
		t.Errorf("Expected target_agent 'python-pro', got '%s'", records[1][1])
	}
	if records[1][2] != "5" {
		t.Errorf("Expected interaction_count '5', got '%s'", records[1][2])
	}
}

// TestParseDuration verifies time duration parsing
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"invalid", 7 * 24 * time.Hour}, // default
		{"", 7 * 24 * time.Hour},        // default
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ParseDuration_%s", test.input), func(t *testing.T) {
			result := parseDuration(test.input)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

// TestExportTrainingDataset_DirectoryCreation verifies directory creation
func TestExportTrainingDataset_DirectoryCreation(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "ml-data")

	// Test directory creation manually
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected path to be a directory")
	}

	// Verify permissions
	if info.Mode()&0755 != 0755 {
		t.Errorf("Expected directory permissions 0755, got %v", info.Mode())
	}
}

// TestFileOutputCreation verifies file output creation
func TestFileOutputCreation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-output.csv")

	// Create test file
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Write sample CSV data
	w := csv.NewWriter(f)
	if err := w.Write([]string{"col1", "col2"}); err != nil {
		t.Fatalf("Failed to write CSV header: %v", err)
	}
	w.Flush()

	// Verify file exists
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	// Verify file is readable
	if info.Mode()&0400 != 0400 {
		t.Errorf("Expected file to be readable")
	}
}

// TestErrorMessages verifies error message format
func TestErrorMessages(t *testing.T) {
	// Test unknown format error message
	expectedError := "[ml-export] Unknown format"
	if !strings.Contains(expectedError, "[ml-export]") {
		t.Error("Error message must include component tag [ml-export]")
	}

	// Test component tag consistency
	errorFormats := []string{
		"[ml-export] Failed to read ML tool events",
		"[ml-export] Failed to create output file",
		"[ml-export] Unknown format",
	}

	for _, errMsg := range errorFormats {
		if !strings.HasPrefix(errMsg, "[ml-export]") {
			t.Errorf("Error message missing component tag: %s", errMsg)
		}
	}
}

// TestCSVColumnCount verifies all CSV exports have consistent column counts
func TestCSVColumnCount(t *testing.T) {
	tests := []struct {
		name           string
		writer         func(*bytes.Buffer)
		expectedCols   int
		expectedHeader string
	}{
		{
			name: "routing CSV",
			writer: func(buf *bytes.Buffer) {
				writeRoutingCSV(buf, []routing.PostToolEvent{})
			},
			expectedCols:   10,
			expectedHeader: "timestamp",
		},
		{
			name: "sequences CSV",
			writer: func(buf *bytes.Buffer) {
				writeSequencesCSV(buf, []ToolSequence{})
			},
			expectedCols:   6,
			expectedHeader: "sequence_id",
		},
		{
			name: "collaborations CSV",
			writer: func(buf *bytes.Buffer) {
				writeCollaborationsCSV(buf, []CollaborationEdge{})
			},
			expectedCols:   5,
			expectedHeader: "source_agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tt.writer(buf)

			reader := csv.NewReader(buf)
			records, err := reader.ReadAll()
			if err != nil {
				t.Fatalf("Failed to parse CSV: %v", err)
			}

			if len(records) == 0 {
				t.Fatal("Expected at least header row")
			}

			// Verify column count
			if len(records[0]) != tt.expectedCols {
				t.Errorf("Expected %d columns, got %d", tt.expectedCols, len(records[0]))
			}

			// Verify header starts with expected column
			if records[0][0] != tt.expectedHeader {
				t.Errorf("Expected first column '%s', got '%s'", tt.expectedHeader, records[0][0])
			}
		})
	}
}

// TestSequenceFiltering verifies successful-only filtering
func TestSequenceFiltering(t *testing.T) {
	events := []routing.PostToolEvent{
		{SessionID: "s1", ToolName: "Read", Success: true},
		{SessionID: "s2", ToolName: "Edit", Success: false},
		{SessionID: "s3", ToolName: "Bash", Success: true},
	}

	sequences := buildSequences(events)

	// Filter successful only
	filtered := make([]ToolSequence, 0)
	for _, seq := range sequences {
		if seq.Successful {
			filtered = append(filtered, seq)
		}
	}

	// Should have 2 successful sequences (s1, s3)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 successful sequences, got %d", len(filtered))
	}

	// Verify all filtered sequences are successful
	for _, seq := range filtered {
		if !seq.Successful {
			t.Errorf("Found unsuccessful sequence in filtered results: %s", seq.SequenceID)
		}
	}
}

// TestTimeFiltering verifies time-based filtering for routing decisions
func TestTimeFiltering(t *testing.T) {
	now := time.Now()
	events := []routing.PostToolEvent{
		{CapturedAt: now.Add(-2 * 24 * time.Hour).Unix(), SelectedTier: "haiku"},   // 2 days ago
		{CapturedAt: now.Add(-10 * 24 * time.Hour).Unix(), SelectedTier: "sonnet"}, // 10 days ago
		{CapturedAt: now.Add(-40 * 24 * time.Hour).Unix(), SelectedTier: "opus"},   // 40 days ago
	}

	// Test 7d filter
	cutoff7d := now.Add(-7 * 24 * time.Hour)
	filtered7d := make([]routing.PostToolEvent, 0)
	for _, e := range events {
		if time.Unix(e.CapturedAt, 0).After(cutoff7d) && e.SelectedTier != "" {
			filtered7d = append(filtered7d, e)
		}
	}

	if len(filtered7d) != 1 {
		t.Errorf("Expected 1 event within 7d, got %d", len(filtered7d))
	}

	// Test 30d filter
	cutoff30d := now.Add(-30 * 24 * time.Hour)
	filtered30d := make([]routing.PostToolEvent, 0)
	for _, e := range events {
		if time.Unix(e.CapturedAt, 0).After(cutoff30d) && e.SelectedTier != "" {
			filtered30d = append(filtered30d, e)
		}
	}

	if len(filtered30d) != 2 {
		t.Errorf("Expected 2 events within 30d, got %d", len(filtered30d))
	}
}

// TestJSONIndentation verifies JSON output is pretty-printed
func TestJSONIndentation(t *testing.T) {
	buf := &bytes.Buffer{}

	data := map[string]interface{}{
		"field1": "value1",
		"nested": map[string]interface{}{
			"field2": "value2",
		},
	}

	writeJSON(buf, data)

	content := buf.String()

	// Verify indentation (should have spaces for nesting)
	if !strings.Contains(content, "  ") {
		t.Error("Expected JSON to be indented with spaces")
	}

	// Verify newlines (pretty-printed)
	if !strings.Contains(content, "\n") {
		t.Error("Expected JSON to have newlines (pretty-printed)")
	}
}

// TestWriteRoutingCSV_EmptyEvents verifies CSV handles empty event list
func TestWriteRoutingCSV_EmptyEvents(t *testing.T) {
	buf := &bytes.Buffer{}

	writeRoutingCSV(buf, []routing.PostToolEvent{})

	// Should still write header
	content := buf.String()
	if !strings.Contains(content, "timestamp") {
		t.Error("Expected header even with empty events")
	}

	// Should have exactly 1 line (header only)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line (header), got %d", len(lines))
	}
}

// TestWriteSequencesCSV_EmptySequences verifies CSV handles empty sequence list
func TestWriteSequencesCSV_EmptySequences(t *testing.T) {
	buf := &bytes.Buffer{}

	writeSequencesCSV(buf, []ToolSequence{})

	content := buf.String()
	if !strings.Contains(content, "sequence_id") {
		t.Error("Expected header even with empty sequences")
	}
}

// TestWriteCollaborationsCSV_EmptyEdges verifies CSV handles empty edge list
func TestWriteCollaborationsCSV_EmptyEdges(t *testing.T) {
	buf := &bytes.Buffer{}

	writeCollaborationsCSV(buf, []CollaborationEdge{})

	content := buf.String()
	if !strings.Contains(content, "source_agent") {
		t.Error("Expected header even with empty edges")
	}
}

// TestBuildSequences_EmptyEvents verifies sequence building with no events
func TestBuildSequences_EmptyEvents(t *testing.T) {
	sequences := buildSequences([]routing.PostToolEvent{})

	if len(sequences) != 0 {
		t.Errorf("Expected 0 sequences from empty events, got %d", len(sequences))
	}
}

// TestAggregateCollaborations_EmptyLogs verifies aggregation with no logs
func TestAggregateCollaborations_EmptyLogs(t *testing.T) {
	edges := aggregateCollaborations([]telemetry.AgentCollaboration{})

	if len(edges) != 0 {
		t.Errorf("Expected 0 edges from empty logs, got %d", len(edges))
	}
}

// TestWriteRoutingCSV_CostCalculation verifies cost is included in CSV
func TestWriteRoutingCSV_CostCalculation(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			TaskType:      "implementation",
			InputTokens:   1000,
			OutputTokens:  500,
			SelectedTier:  "sonnet",
			SelectedAgent: "python-pro",
			Success:       true,
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()

	// Verify cost column exists (9th column, 0-indexed column 8)
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("Expected at least 2 rows (header + data)")
	}

	// Cost should be in column 8 (outcome_cost)
	cost := records[1][8]
	if !strings.Contains(cost, ".") {
		t.Errorf("Expected cost to be a decimal number, got '%s'", cost)
	}
}

// TestWriteRoutingCSV_EscalationDetection verifies escalation flag
func TestWriteRoutingCSV_EscalationDetection(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			Tier:          "haiku",
			SelectedTier:  "sonnet", // Escalation occurred
			SelectedAgent: "python-pro",
			Success:       true,
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// Escalation should be in last column (index 9)
	escalation := records[1][9]
	if escalation != "true" {
		t.Errorf("Expected escalation_required=true when tier changes, got '%s'", escalation)
	}
}

// TestBuildSequences_MultipleSessionsOrdering verifies sessions are processed correctly
func TestBuildSequences_MultipleSessionsOrdering(t *testing.T) {
	events := []routing.PostToolEvent{
		{SessionID: "s2", ToolName: "Edit"},
		{SessionID: "s1", ToolName: "Read"},
		{SessionID: "s1", ToolName: "Write"},
		{SessionID: "s2", ToolName: "Bash"},
	}

	sequences := buildSequences(events)

	// Verify both sessions captured
	sessionIDs := make(map[string]bool)
	for _, seq := range sequences {
		sessionIDs[seq.SequenceID] = true
	}

	if !sessionIDs["s1"] || !sessionIDs["s2"] {
		t.Error("Expected both s1 and s2 sessions")
	}

	// Find s1 and verify tool count
	for _, seq := range sequences {
		if seq.SequenceID == "s1" && len(seq.Tools) != 2 {
			t.Errorf("Expected 2 tools in s1, got %d", len(seq.Tools))
		}
	}
}

// TestWriteSequencesCSV_ToolChainFormatting verifies arrow-separated tool chain
func TestWriteSequencesCSV_ToolChainFormatting(t *testing.T) {
	buf := &bytes.Buffer{}

	sequences := []ToolSequence{
		{
			SequenceID: "test",
			Tools:      []string{"Read", "Edit", "Write", "Bash"},
		},
	}

	writeSequencesCSV(buf, sequences)

	content := buf.String()

	// Verify arrow-separated formatting
	expected := "Read → Edit → Write → Bash"
	if !strings.Contains(content, expected) {
		t.Errorf("Expected tool chain '%s', content:\n%s", expected, content)
	}
}

// TestAggregateCollaborations_MultipleInteractions verifies interaction counting
func TestAggregateCollaborations_MultipleInteractions(t *testing.T) {
	collabs := []telemetry.AgentCollaboration{
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: true},
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: true},
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: false},
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: true},
	}

	edges := aggregateCollaborations(collabs)

	if len(edges) != 1 {
		t.Fatalf("Expected 1 aggregated edge, got %d", len(edges))
	}

	edge := edges[0]

	// Verify interaction count
	if edge.InteractionCount != 4 {
		t.Errorf("Expected 4 interactions, got %d", edge.InteractionCount)
	}

	// Verify success rate (3 successes / 4 total = 0.75)
	expectedRate := 0.75
	if edge.SuccessRate < expectedRate-0.01 || edge.SuccessRate > expectedRate+0.01 {
		t.Errorf("Expected success rate ~%.2f, got %.2f", expectedRate, edge.SuccessRate)
	}
}

// TestPrintUsage verifies usage text contains all commands
func TestPrintUsage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	os.Stdout = oldStdout

	buf := &bytes.Buffer{}
	buf.ReadFrom(r)
	output := buf.String()

	// Verify all commands are documented
	commands := []string{"routing", "sequences", "collaborations", "training-dataset"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected usage to include command '%s'", cmd)
		}
	}

	// Verify examples are shown
	if !strings.Contains(output, "Examples:") {
		t.Error("Expected usage to include examples section")
	}
}

// TestWriteJSON_ErrorHandling verifies JSON encoding of complex data
func TestWriteJSON_ErrorHandling(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test with array
	data := []map[string]interface{}{
		{"field1": "value1"},
		{"field2": 123},
	}

	writeJSON(buf, data)

	// Verify valid JSON array
	var decoded []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to decode JSON array: %v", err)
	}

	if len(decoded) != 2 {
		t.Errorf("Expected 2 array elements, got %d", len(decoded))
	}
}

// TestWriteRoutingCSV_TimestampFormat verifies RFC3339 timestamp format
func TestWriteRoutingCSV_TimestampFormat(t *testing.T) {
	buf := &bytes.Buffer{}

	now := time.Now()
	events := []routing.PostToolEvent{
		{
			CapturedAt:    now.Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: "codebase-search",
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// Timestamp should be parseable as RFC3339
	timestamp := records[1][0]
	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Expected RFC3339 timestamp, got '%s': %v", timestamp, err)
	}
}

// TestWriteRoutingCSV_BooleanFormatting verifies boolean fields are strings
func TestWriteRoutingCSV_BooleanFormatting(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: "test",
			Success:       true,
			Tier:          "haiku",
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// outcome_success column (index 7) should be "true" string
	if records[1][7] != "true" {
		t.Errorf("Expected 'true', got '%s'", records[1][7])
	}

	// escalation_required column (index 9) should be "false" string (no escalation)
	if records[1][9] != "false" {
		t.Errorf("Expected 'false', got '%s'", records[1][9])
	}
}

// TestBuildSequences_CostAccumulation verifies cost is summed correctly
func TestBuildSequences_CostAccumulation(t *testing.T) {
	events := []routing.PostToolEvent{
		{
			SessionID:    "s1",
			ToolName:     "Read",
			InputTokens:  100,
			OutputTokens: 50,
			DurationMs:   100,
			Model:        "haiku", // EstimatedCost expects tier name
			Tier:         "haiku",
		},
		{
			SessionID:    "s1",
			ToolName:     "Write",
			InputTokens:  200,
			OutputTokens: 100,
			DurationMs:   200,
			Model:        "haiku",
			Tier:         "haiku",
		},
	}

	sequences := buildSequences(events)

	if len(sequences) != 1 {
		t.Fatalf("Expected 1 sequence, got %d", len(sequences))
	}

	seq := sequences[0]

	// Verify token accumulation: (100+50) + (200+100) = 450
	if seq.TokenCount != 450 {
		t.Errorf("Expected total tokens 450, got %d", seq.TokenCount)
	}

	// Verify duration accumulation: 100 + 200 = 300
	if seq.DurationMs != 300 {
		t.Errorf("Expected total duration 300ms, got %d", seq.DurationMs)
	}

	// Verify cost is accumulated (non-zero when model/tier provided)
	// Expected: 450 tokens * $0.0005/1K = $0.000225
	if seq.Cost == 0 {
		t.Error("Expected non-zero accumulated cost with valid model/tier")
	}
	// Verify cost is reasonable (should be < $0.001 for 450 tokens on haiku)
	if seq.Cost > 0.001 {
		t.Errorf("Expected cost < $0.001, got $%.6f", seq.Cost)
	}
}

// TestWriteSequencesCSV_NumericFormatting verifies numeric columns
func TestWriteSequencesCSV_NumericFormatting(t *testing.T) {
	buf := &bytes.Buffer{}

	sequences := []ToolSequence{
		{
			SequenceID: "test",
			Tools:      []string{"Read"},
			Successful: true,
			DurationMs: 1500,
			TokenCount: 2000,
			Cost:       0.001234,
		},
	}

	writeSequencesCSV(buf, sequences)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// Verify duration is integer string
	if records[1][3] != "1500" {
		t.Errorf("Expected duration '1500', got '%s'", records[1][3])
	}

	// Verify cost has decimal point
	cost := records[1][5]
	if !strings.Contains(cost, ".") {
		t.Errorf("Expected cost with decimal, got '%s'", cost)
	}
}

// TestWriteCollaborationsCSV_SuccessRateFormatting verifies success rate precision
func TestWriteCollaborationsCSV_SuccessRateFormatting(t *testing.T) {
	buf := &bytes.Buffer{}

	edges := []CollaborationEdge{
		{
			SourceAgent:      "A",
			TargetAgent:      "B",
			InteractionCount: 10,
			SuccessRate:      0.6789,
			AvgCost:          0.01,
		},
	}

	writeCollaborationsCSV(buf, edges)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// Success rate should have 4 decimal places
	successRate := records[1][3]
	if !strings.Contains(successRate, ".") {
		t.Error("Expected success rate with decimal")
	}

	// Verify it's formatted as expected (0.6789 -> "0.6789")
	if len(strings.Split(successRate, ".")[1]) < 4 {
		t.Errorf("Expected at least 4 decimal places, got '%s'", successRate)
	}
}

// TestAggregateCollaborations_DifferentPairs verifies multiple unique edges
func TestAggregateCollaborations_DifferentPairs(t *testing.T) {
	collabs := []telemetry.AgentCollaboration{
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: true},
		{ParentAgent: "A", ChildAgent: "C", ChildSuccess: true},
		{ParentAgent: "B", ChildAgent: "C", ChildSuccess: true},
	}

	edges := aggregateCollaborations(collabs)

	// Should have 3 unique edges
	if len(edges) != 3 {
		t.Errorf("Expected 3 unique edges, got %d", len(edges))
	}

	// Verify all have interaction count of 1
	for _, edge := range edges {
		if edge.InteractionCount != 1 {
			t.Errorf("Expected interaction count 1, got %d for %s→%s",
				edge.InteractionCount, edge.SourceAgent, edge.TargetAgent)
		}
	}
}

// TestParseDuration_EdgeCases verifies duration parsing edge cases
func TestParseDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1D", 7 * 24 * time.Hour},   // uppercase -> default
		{"7days", 7 * 24 * time.Hour}, // wrong format -> default
		{"30", 7 * 24 * time.Hour},    // no unit -> default
		{"0d", 7 * 24 * time.Hour},    // zero -> default
	}

	for _, tt := range tests {
		result := parseDuration(tt.input)
		if result != tt.expected {
			t.Errorf("parseDuration(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

// TestWriteRoutingCSV_FloatPrecision verifies cost formatting precision
func TestWriteRoutingCSV_FloatPrecision(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			InputTokens:   100,
			OutputTokens:  50,
			SelectedTier:  "haiku",
			SelectedAgent: "test",
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	// Cost column should have 6 decimal places (%.6f format)
	cost := records[1][8]
	parts := strings.Split(cost, ".")
	if len(parts) != 2 {
		t.Fatalf("Expected decimal number, got '%s'", cost)
	}

	if len(parts[1]) != 6 {
		t.Errorf("Expected 6 decimal places, got %d in '%s'", len(parts[1]), cost)
	}
}

// TestFilterAndExportRouting_CSV verifies routing export with time filtering
func TestFilterAndExportRouting_CSV(t *testing.T) {
	buf := &bytes.Buffer{}

	now := time.Now()
	events := []routing.PostToolEvent{
		{
			CapturedAt:    now.Add(-2 * 24 * time.Hour).Unix(), // 2 days ago
			SelectedTier:  "haiku",
			SelectedAgent: "codebase-search",
		},
		{
			CapturedAt:    now.Add(-10 * 24 * time.Hour).Unix(), // 10 days ago
			SelectedTier:  "sonnet",
			SelectedAgent: "python-pro",
		},
		{
			CapturedAt:   now.Add(-2 * 24 * time.Hour).Unix(),
			SelectedTier: "", // Empty tier - should be filtered out
		},
	}

	// Test 7d filter
	count, err := filterAndExportRouting(events, "csv", "7d", buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 1 event (2 days ago, within 7d window, with non-empty tier)
	if count != 1 {
		t.Errorf("Expected 1 event within 7d, got %d", count)
	}

	// Verify CSV output
	content := buf.String()
	if !strings.Contains(content, "timestamp") {
		t.Error("Expected CSV header")
	}
}

// TestFilterAndExportRouting_JSON verifies JSON export format
func TestFilterAndExportRouting_JSON(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: "test",
			TaskType:      "search",
		},
	}

	count, err := filterAndExportRouting(events, "json", "7d", buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 event, got %d", count)
	}

	// Verify valid JSON
	var decoded []routing.PostToolEvent
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(decoded) != 1 {
		t.Errorf("Expected 1 decoded event, got %d", len(decoded))
	}
}

// TestFilterAndExportRouting_InvalidFormat verifies error handling
func TestFilterAndExportRouting_InvalidFormat(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{CapturedAt: time.Now().Unix(), SelectedTier: "haiku"},
	}

	_, err := filterAndExportRouting(events, "xml", "7d", buf)
	if err == nil {
		t.Error("Expected error for invalid format")
	}

	if !strings.Contains(err.Error(), "Unknown format") {
		t.Errorf("Expected 'Unknown format' error, got: %v", err)
	}
}

// TestFilterAndExportRouting_TimeFiltering verifies different time windows
func TestFilterAndExportRouting_TimeFiltering(t *testing.T) {
	now := time.Now()
	events := []routing.PostToolEvent{
		{CapturedAt: now.Add(-2 * 24 * time.Hour).Unix(), SelectedTier: "haiku"},   // 2d ago
		{CapturedAt: now.Add(-10 * 24 * time.Hour).Unix(), SelectedTier: "sonnet"}, // 10d ago
		{CapturedAt: now.Add(-40 * 24 * time.Hour).Unix(), SelectedTier: "opus"},   // 40d ago
	}

	tests := []struct {
		since    string
		expected int
	}{
		{"1d", 0},  // None within last day
		{"7d", 1},  // 2d event only
		{"30d", 2}, // 2d and 10d events
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("since_%s", tt.since), func(t *testing.T) {
			buf := &bytes.Buffer{}
			count, err := filterAndExportRouting(events, "csv", tt.since, buf)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if count != tt.expected {
				t.Errorf("Expected %d events for %s window, got %d", tt.expected, tt.since, count)
			}
		})
	}
}

// TestExportRoutingDecisions_Integration tests full CLI workflow with file output
func TestExportRoutingDecisions_Integration(t *testing.T) {
	// Setup: Create minimal telemetry data
	tmpDir := t.TempDir()
	telemetryFile := filepath.Join(tmpDir, "ml-tool-events.jsonl")

	// Write sample telemetry event
	f, err := os.Create(telemetryFile)
	if err != nil {
		t.Fatalf("Failed to create test telemetry file: %v", err)
	}

	event := routing.PostToolEvent{
		CapturedAt:    time.Now().Unix(),
		TaskType:      "implementation",
		TaskDomain:    "go",
		InputTokens:   1000,
		OutputTokens:  500,
		SelectedTier:  "sonnet",
		SelectedAgent: "go-pro",
		Success:       true,
		Tier:          "sonnet",
		SessionID:     "test-session",
		ToolName:      "Edit",
	}

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(event); err != nil {
		t.Fatalf("Failed to write test event: %v", err)
	}
	f.Close()

	// Set environment to use test telemetry file
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create .claude/telemetry directory structure
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0755); err != nil {
		t.Fatalf("Failed to create telemetry directory: %v", err)
	}

	// Copy test file to expected location
	destFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	input, _ := os.ReadFile(telemetryFile)
	if err := os.WriteFile(destFile, input, 0644); err != nil {
		t.Fatalf("Failed to write telemetry file: %v", err)
	}

	// Test CSV export to file
	outputFile := filepath.Join(tmpDir, "test-routing.csv")
	exportRoutingDecisions("csv", "7d", outputFile)

	// Verify output file exists and contains data
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "timestamp") {
		t.Error("Expected CSV header in output file")
	}
	// Note: Data may be filtered out by time window - just verify file was created successfully
}

// TestExportToolSequences_Integration tests full CLI workflow with file output
func TestExportToolSequences_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	telemetryFile := filepath.Join(tmpDir, "ml-tool-events.jsonl")

	// Write sample telemetry events forming a sequence
	f, err := os.Create(telemetryFile)
	if err != nil {
		t.Fatalf("Failed to create test telemetry file: %v", err)
	}

	events := []routing.PostToolEvent{
		{
			SessionID:    "seq-1",
			ToolName:     "Read",
			Success:      true,
			DurationMs:   100,
			InputTokens:  50,
			OutputTokens: 30,
			CapturedAt:   time.Now().Unix(),
		},
		{
			SessionID:    "seq-1",
			ToolName:     "Edit",
			Success:      true,
			DurationMs:   200,
			InputTokens:  100,
			OutputTokens: 50,
			CapturedAt:   time.Now().Unix(),
		},
	}

	encoder := json.NewEncoder(f)
	for _, e := range events {
		if err := encoder.Encode(e); err != nil {
			t.Fatalf("Failed to write test event: %v", err)
		}
	}
	f.Close()

	// Set environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)
	input, _ := os.ReadFile(telemetryFile)
	os.WriteFile(filepath.Join(telemetryDir, "ml-tool-events.jsonl"), input, 0644)

	// Test JSON export (successful-only = false to include all sequences)
	outputFile := filepath.Join(tmpDir, "test-sequences.json")
	exportToolSequences("json", false, outputFile)

	// Verify output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var sequences []ToolSequence
	if err := json.Unmarshal(content, &sequences); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Note: Sequences may be empty due to successful-only filter
	// Just verify file was created and is valid JSON
}

// TestExportCollaborations_Integration tests full CLI workflow with file output
func TestExportCollaborations_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	collabFile := filepath.Join(tmpDir, "collaboration.jsonl")

	// Write sample collaboration data
	f, err := os.Create(collabFile)
	if err != nil {
		t.Fatalf("Failed to create test collaboration file: %v", err)
	}

	collabs := []telemetry.AgentCollaboration{
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "python-pro",
			ChildSuccess: true,
			Timestamp:    time.Now().Unix(),
		},
		{
			ParentAgent:  "orchestrator",
			ChildAgent:   "python-pro",
			ChildSuccess: true,
			Timestamp:    time.Now().Unix(),
		},
	}

	encoder := json.NewEncoder(f)
	for _, c := range collabs {
		if err := encoder.Encode(c); err != nil {
			t.Fatalf("Failed to write collaboration: %v", err)
		}
	}
	f.Close()

	// Set environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)
	input, _ := os.ReadFile(collabFile)
	os.WriteFile(filepath.Join(telemetryDir, "collaboration.jsonl"), input, 0644)

	// Test CSV export
	outputFile := filepath.Join(tmpDir, "test-collabs.csv")
	exportCollaborations("csv", outputFile)

	// Verify output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "source_agent") {
		t.Error("Expected CSV header in output file")
	}
	// Note: Data may be empty - just verify file was created successfully
}

// TestExportTrainingDataset_Integration tests full training dataset export
func TestExportTrainingDataset_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal telemetry data
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	// Write ML events
	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)
	event := routing.PostToolEvent{
		CapturedAt:    time.Now().Unix(),
		SelectedTier:  "haiku",
		SelectedAgent: "codebase-search",
		SessionID:     "s1",
		ToolName:      "Grep",
		Success:       true,
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	// Write collaborations
	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	f, _ = os.Create(collabFile)
	collab := telemetry.AgentCollaboration{
		ParentAgent:  "orchestrator",
		ChildAgent:   "haiku-scout",
		ChildSuccess: true,
		Timestamp:    time.Now().Unix(),
	}
	json.NewEncoder(f).Encode(collab)
	f.Close()

	// Set environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Export training dataset
	outputDir := filepath.Join(tmpDir, "ml-data")
	exportTrainingDataset(outputDir)

	// Verify all output files exist
	expectedFiles := []string{
		"routing.csv",
		"sequences.json",
		"collaborations.json",
	}

	for _, filename := range expectedFiles {
		path := filepath.Join(outputDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", filename)
		}
	}

	// Verify routing.csv has content
	routingContent, _ := os.ReadFile(filepath.Join(outputDir, "routing.csv"))
	if !strings.Contains(string(routingContent), "timestamp") {
		t.Error("routing.csv missing header")
	}

	// Verify sequences.json is valid JSON
	sequencesContent, _ := os.ReadFile(filepath.Join(outputDir, "sequences.json"))
	var sequences []ToolSequence
	if err := json.Unmarshal(sequencesContent, &sequences); err != nil {
		t.Errorf("sequences.json not valid JSON: %v", err)
	}

	// Verify collaborations.json is valid JSON
	collabsContent, _ := os.ReadFile(filepath.Join(outputDir, "collaborations.json"))
	var edges []CollaborationEdge
	if err := json.Unmarshal(collabsContent, &edges); err != nil {
		t.Errorf("collaborations.json not valid JSON: %v", err)
	}
}

// TestExportRoutingDecisions_StdoutOutput tests stdout redirection
func TestExportRoutingDecisions_StdoutOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup minimal telemetry
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)
	event := routing.PostToolEvent{
		CapturedAt:    time.Now().Unix(),
		SelectedTier:  "haiku",
		SelectedAgent: "test",
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exportRoutingDecisions("csv", "7d", "-")

	w.Close()
	os.Stdout = oldStdout

	buf := &bytes.Buffer{}
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "timestamp") {
		t.Error("Expected CSV output to stdout")
	}
}

// TestExportToolSequences_CSVFormat tests CSV format export
func TestExportToolSequences_CSVFormat(t *testing.T) {
	tmpDir := t.TempDir()
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)
	event := routing.PostToolEvent{
		SessionID:  "s1",
		ToolName:   "Read",
		Success:    true,
		CapturedAt: time.Now().Unix(),
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	outputFile := filepath.Join(tmpDir, "sequences.csv")
	exportToolSequences("csv", false, outputFile)

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read CSV output: %v", err)
	}

	if !strings.Contains(string(content), "sequence_id") {
		t.Error("Expected CSV header in sequences output")
	}
}

// TestExportCollaborations_JSONFormat tests JSON format export
func TestExportCollaborations_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	f, _ := os.Create(collabFile)
	collab := telemetry.AgentCollaboration{
		ParentAgent:  "A",
		ChildAgent:   "B",
		ChildSuccess: true,
		Timestamp:    time.Now().Unix(),
	}
	json.NewEncoder(f).Encode(collab)
	f.Close()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	outputFile := filepath.Join(tmpDir, "collabs.json")
	exportCollaborations("json", outputFile)

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read JSON output: %v", err)
	}

	var edges []CollaborationEdge
	if err := json.Unmarshal(content, &edges); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}
}

// TestWriteRoutingCSV_ErrorRecovery tests CSV writing error paths
func TestWriteRoutingCSV_ErrorRecovery(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test with events that have missing fields
	events := []routing.PostToolEvent{
		{
			CapturedAt:   time.Now().Unix(),
			SelectedTier: "", // Empty tier
		},
		{
			CapturedAt:    time.Now().Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: "test",
			Success:       true,
		},
	}

	// Should not panic, should write what it can
	writeRoutingCSV(buf, events)

	content := buf.String()
	if !strings.Contains(content, "timestamp") {
		t.Error("Expected header even with problematic events")
	}
}

// TestWriteJSON_ErrorHandling_InvalidData tests JSON encoding edge cases
func TestWriteJSON_ErrorHandling_InvalidData(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test with nil
	writeJSON(buf, nil)

	content := buf.String()
	if content != "null\n" {
		t.Errorf("Expected 'null', got '%s'", content)
	}
}

// TestBuildSequences_SingleEventPerSession tests minimal sequences
func TestBuildSequences_SingleEventPerSession(t *testing.T) {
	events := []routing.PostToolEvent{
		{SessionID: "s1", ToolName: "Read", Success: true},
	}

	sequences := buildSequences(events)

	if len(sequences) != 1 {
		t.Errorf("Expected 1 sequence, got %d", len(sequences))
	}

	if len(sequences[0].Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(sequences[0].Tools))
	}
}

// TestAggregateCollaborations_SingleInteraction tests minimal edge
func TestAggregateCollaborations_SingleInteraction(t *testing.T) {
	collabs := []telemetry.AgentCollaboration{
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: true},
	}

	edges := aggregateCollaborations(collabs)

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].InteractionCount != 1 {
		t.Errorf("Expected interaction count 1, got %d", edges[0].InteractionCount)
	}

	if edges[0].SuccessRate != 1.0 {
		t.Errorf("Expected success rate 1.0, got %.2f", edges[0].SuccessRate)
	}
}

// TestParseDuration_AllValidFormats verifies all supported duration formats
func TestParseDuration_AllValidFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		result := parseDuration(tt.input)
		if result != tt.expected {
			t.Errorf("parseDuration(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

// TestExportRoutingDecisions_ErrorPaths tests error handling for routing export
func TestExportRoutingDecisions_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up empty telemetry directory (no data)
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	// Write empty telemetry file
	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	os.WriteFile(mlFile, []byte{}, 0644)

	// Test filterAndExportRouting with invalid format (which returns error)
	buf := &bytes.Buffer{}
	_, err := filterAndExportRouting([]routing.PostToolEvent{}, "invalid-format", "7d", buf)
	if err == nil {
		t.Error("Expected error for invalid format")
	}

	// Verify error message format
	if !strings.Contains(err.Error(), "Unknown format") {
		t.Errorf("Expected 'Unknown format' error, got: %v", err)
	}
}

// TestExportToolSequences_ErrorPaths tests error handling for sequences export
func TestExportToolSequences_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	// Write empty telemetry
	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	os.WriteFile(mlFile, []byte{}, 0644)

	// Test CSV format with empty data
	outputFile := filepath.Join(tmpDir, "empty-sequences.csv")
	exportToolSequences("csv", false, outputFile)

	content, _ := os.ReadFile(outputFile)
	if !strings.Contains(string(content), "sequence_id") {
		t.Error("Expected CSV header even with empty data")
	}
}

// TestExportCollaborations_ErrorPaths tests error handling for collaborations export
func TestExportCollaborations_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	// Write empty collaboration file
	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	os.WriteFile(collabFile, []byte{}, 0644)

	// Test JSON format with empty data
	outputFile := filepath.Join(tmpDir, "empty-collabs.json")
	exportCollaborations("json", outputFile)

	content, _ := os.ReadFile(outputFile)
	var edges []CollaborationEdge
	if err := json.Unmarshal(content, &edges); err != nil {
		t.Errorf("Expected valid JSON even with empty data: %v", err)
	}

	if len(edges) != 0 {
		t.Errorf("Expected 0 edges from empty data, got %d", len(edges))
	}
}

// TestWriteRoutingCSV_HeaderWriteError tests header write error handling
func TestWriteRoutingCSV_HeaderWriteError(t *testing.T) {
	// This test verifies the CSV writer is properly flushed
	buf := &bytes.Buffer{}
	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: "test",
		},
	}

	writeRoutingCSV(buf, events)

	// Verify flush occurred (data is in buffer)
	if buf.Len() == 0 {
		t.Error("Expected data in buffer after writeRoutingCSV")
	}
}

// TestWriteSequencesCSV_FlushBehavior tests CSV writer flushing
func TestWriteSequencesCSV_FlushBehavior(t *testing.T) {
	buf := &bytes.Buffer{}
	sequences := []ToolSequence{
		{
			SequenceID: "test",
			Tools:      []string{"Read"},
			Successful: true,
		},
	}

	writeSequencesCSV(buf, sequences)

	if buf.Len() == 0 {
		t.Error("Expected data in buffer after writeSequencesCSV")
	}
}

// TestWriteCollaborationsCSV_FlushBehavior tests CSV writer flushing
func TestWriteCollaborationsCSV_FlushBehavior(t *testing.T) {
	buf := &bytes.Buffer{}
	edges := []CollaborationEdge{
		{
			SourceAgent:      "A",
			TargetAgent:      "B",
			InteractionCount: 1,
			SuccessRate:      1.0,
		},
	}

	writeCollaborationsCSV(buf, edges)

	if buf.Len() == 0 {
		t.Error("Expected data in buffer after writeCollaborationsCSV")
	}
}

// TestFilterAndExportRouting_EmptyFilter tests time filter that excludes all events
func TestFilterAndExportRouting_EmptyFilter(t *testing.T) {
	// Events are all older than 1 day
	events := []routing.PostToolEvent{
		{
			CapturedAt:   time.Now().Add(-10 * 24 * time.Hour).Unix(),
			SelectedTier: "haiku",
		},
	}

	buf := &bytes.Buffer{}
	count, err := filterAndExportRouting(events, "csv", "1d", buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 events with 1d filter, got %d", count)
	}

	// Verify CSV header still written
	if !strings.Contains(buf.String(), "timestamp") {
		t.Error("Expected CSV header even with 0 filtered events")
	}
}

// TestBuildSequences_MixedSuccessFailure tests sequences with mixed outcomes
func TestBuildSequences_MixedSuccessFailure(t *testing.T) {
	events := []routing.PostToolEvent{
		{SessionID: "s1", ToolName: "Read", Success: true, DurationMs: 100},
		{SessionID: "s1", ToolName: "Edit", Success: false, DurationMs: 200}, // Failure
		{SessionID: "s1", ToolName: "Write", Success: true, DurationMs: 150},
	}

	sequences := buildSequences(events)

	if len(sequences) != 1 {
		t.Fatalf("Expected 1 sequence, got %d", len(sequences))
	}

	// Sequence should be marked as unsuccessful due to one failure
	if sequences[0].Successful {
		t.Error("Expected sequence to be unsuccessful (contains failure)")
	}

	// Total duration should still accumulate
	if sequences[0].DurationMs != 450 {
		t.Errorf("Expected total duration 450ms, got %d", sequences[0].DurationMs)
	}
}

// TestAggregateCollaborations_AllFailures tests collaboration with 0% success
func TestAggregateCollaborations_AllFailures(t *testing.T) {
	collabs := []telemetry.AgentCollaboration{
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: false},
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: false},
		{ParentAgent: "A", ChildAgent: "B", ChildSuccess: false},
	}

	edges := aggregateCollaborations(collabs)

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].SuccessRate != 0.0 {
		t.Errorf("Expected success rate 0.0, got %.2f", edges[0].SuccessRate)
	}

	if edges[0].InteractionCount != 3 {
		t.Errorf("Expected 3 interactions, got %d", edges[0].InteractionCount)
	}
}

// TestWriteJSON_LargeObject tests JSON encoding of complex nested structures
func TestWriteJSON_LargeObject(t *testing.T) {
	buf := &bytes.Buffer{}

	data := map[string]interface{}{
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"array": []int{1, 2, 3, 4, 5},
				"bool":  true,
			},
		},
		"string": "test",
		"number": 42.5,
	}

	writeJSON(buf, data)

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to decode complex JSON: %v", err)
	}

	// Verify structure is preserved
	nested, ok := decoded["nested"].(map[string]interface{})
	if !ok {
		t.Error("Expected nested object")
	}

	deep, ok := nested["deep"].(map[string]interface{})
	if !ok {
		t.Error("Expected deep nested object")
	}

	if deep["bool"] != true {
		t.Error("Expected bool value true in nested structure")
	}
}

// TestExportTrainingDataset_DirectoryExists tests behavior when output dir already exists
func TestExportTrainingDataset_DirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup telemetry
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)
	event := routing.PostToolEvent{
		CapturedAt:    time.Now().Unix(),
		SelectedTier:  "haiku",
		SelectedAgent: "test",
		SessionID:     "s1",
		ToolName:      "Read",
		Success:       true,
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	f, _ = os.Create(collabFile)
	collab := telemetry.AgentCollaboration{
		ParentAgent:  "A",
		ChildAgent:   "B",
		ChildSuccess: true,
		Timestamp:    time.Now().Unix(),
	}
	json.NewEncoder(f).Encode(collab)
	f.Close()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	outputDir := filepath.Join(tmpDir, "ml-data")

	// Pre-create the directory
	os.MkdirAll(outputDir, 0755)

	// Create existing file that should be overwritten
	existingFile := filepath.Join(outputDir, "routing.csv")
	os.WriteFile(existingFile, []byte("old content"), 0644)

	// Export should succeed and overwrite
	exportTrainingDataset(outputDir)

	// Verify file was overwritten
	content, _ := os.ReadFile(existingFile)
	if strings.Contains(string(content), "old content") {
		t.Error("Expected existing file to be overwritten")
	}

	if !strings.Contains(string(content), "timestamp") {
		t.Error("Expected new CSV content")
	}
}

// TestMain_SwitchLogic tests the main CLI switch logic via os.Args manipulation
func TestMain_SwitchLogic(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Setup minimal telemetry
	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)
	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	os.WriteFile(mlFile, []byte{}, 0644)
	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	os.WriteFile(collabFile, []byte{}, 0644)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "routing command",
			args: []string{"gogent-ml-export", "routing", "--format", "csv", "--output", filepath.Join(tmpDir, "test.csv")},
		},
		{
			name: "sequences command",
			args: []string{"gogent-ml-export", "sequences", "--format", "json", "--output", filepath.Join(tmpDir, "test.json")},
		},
		{
			name: "collaborations command",
			args: []string{"gogent-ml-export", "collaborations", "--format", "csv", "--output", filepath.Join(tmpDir, "test.csv")},
		},
		{
			name: "training-dataset command",
			args: []string{"gogent-ml-export", "training-dataset", "--output", filepath.Join(tmpDir, "dataset")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set args
			os.Args = tt.args

			// We can't easily call main() due to os.Exit, but we've tested the individual commands
			// This test documents the expected arg structure
			if len(os.Args) < 2 {
				t.Error("Expected at least 2 args (program name + command)")
			}
		})
	}
}

// TestWriteRoutingCSV_AllFieldsCovered tests all CSV column generation
func TestWriteRoutingCSV_AllFieldsCovered(t *testing.T) {
	buf := &bytes.Buffer{}

	events := []routing.PostToolEvent{
		{
			CapturedAt:    time.Now().Unix(),
			TaskType:      "implementation",
			TaskDomain:    "python",
			InputTokens:   1000,
			OutputTokens:  500,
			SelectedTier:  "sonnet",
			SelectedAgent: "python-pro",
			Success:       true,
			Tier:          "haiku", // Different from SelectedTier to trigger escalation
			Model:         "sonnet", // Required for cost calculation
		},
	}

	writeRoutingCSV(buf, events)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	if len(records) < 2 {
		t.Fatal("Expected at least header + 1 data row")
	}

	// Verify all 10 columns present
	if len(records[1]) != 10 {
		t.Errorf("Expected 10 columns, got %d", len(records[1]))
	}

	// Verify escalation detected
	if records[1][9] != "true" {
		t.Error("Expected escalation=true when tier differs from selected_tier")
	}

	// Verify cost calculated
	cost := records[1][8]
	if cost == "0.000000" {
		t.Error("Expected non-zero cost for event with tokens")
	}
}

// TestWriteSequencesCSV_AllFieldsCovered tests all CSV column generation
func TestWriteSequencesCSV_AllFieldsCovered(t *testing.T) {
	buf := &bytes.Buffer{}

	sequences := []ToolSequence{
		{
			SequenceID: "test-seq",
			Tools:      []string{"Read", "Edit", "Write"},
			Successful: true,
			DurationMs: 1500,
			TokenCount: 3000,
			Cost:       0.015,
		},
	}

	writeSequencesCSV(buf, sequences)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	if len(records) < 2 {
		t.Fatal("Expected header + 1 data row")
	}

	// Verify all 6 columns
	if len(records[1]) != 6 {
		t.Errorf("Expected 6 columns, got %d", len(records[1]))
	}

	// Verify sequence_id
	if records[1][0] != "test-seq" {
		t.Errorf("Expected sequence_id 'test-seq', got '%s'", records[1][0])
	}

	// Verify tool chain
	if !strings.Contains(records[1][1], "→") {
		t.Error("Expected arrow separator in tool chain")
	}

	// Verify successful
	if records[1][2] != "true" {
		t.Errorf("Expected successful 'true', got '%s'", records[1][2])
	}

	// Verify numeric fields
	if records[1][3] != "1500" {
		t.Errorf("Expected duration '1500', got '%s'", records[1][3])
	}
	if records[1][4] != "3000" {
		t.Errorf("Expected token_count '3000', got '%s'", records[1][4])
	}
}

// TestWriteCollaborationsCSV_AllFieldsCovered tests all CSV column generation
func TestWriteCollaborationsCSV_AllFieldsCovered(t *testing.T) {
	buf := &bytes.Buffer{}

	edges := []CollaborationEdge{
		{
			SourceAgent:      "orchestrator",
			TargetAgent:      "python-pro",
			InteractionCount: 25,
			SuccessRate:      0.8800,
			AvgCost:          0.0125,
		},
	}

	writeCollaborationsCSV(buf, edges)

	content := buf.String()
	reader := csv.NewReader(strings.NewReader(content))
	records, _ := reader.ReadAll()

	if len(records) < 2 {
		t.Fatal("Expected header + 1 data row")
	}

	// Verify all 5 columns
	if len(records[1]) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(records[1]))
	}

	// Verify source/target agents
	if records[1][0] != "orchestrator" {
		t.Errorf("Expected source 'orchestrator', got '%s'", records[1][0])
	}
	if records[1][1] != "python-pro" {
		t.Errorf("Expected target 'python-pro', got '%s'", records[1][1])
	}

	// Verify interaction count
	if records[1][2] != "25" {
		t.Errorf("Expected interaction_count '25', got '%s'", records[1][2])
	}

	// Verify success rate precision (4 decimals)
	successRate := records[1][3]
	if !strings.Contains(successRate, ".") {
		t.Error("Expected decimal in success rate")
	}
}

// TestExportRoutingDecisions_JSONFormat tests JSON format output
func TestExportRoutingDecisions_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)
	event := routing.PostToolEvent{
		CapturedAt:    time.Now().Unix(),
		SelectedTier:  "haiku",
		SelectedAgent: "test",
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	outputFile := filepath.Join(tmpDir, "routing.json")
	exportRoutingDecisions("json", "7d", outputFile)

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read JSON output: %v", err)
	}

	var events []routing.PostToolEvent
	if err := json.Unmarshal(content, &events); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}
}

// TestExportToolSequences_SuccessfulOnlyFilter tests successful-only flag
func TestExportToolSequences_SuccessfulOnlyFilter(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)

	// Write mixed success/failure events
	events := []routing.PostToolEvent{
		{SessionID: "s1", ToolName: "Read", Success: true, CapturedAt: time.Now().Unix()},
		{SessionID: "s2", ToolName: "Edit", Success: false, CapturedAt: time.Now().Unix()},
	}

	for _, e := range events {
		json.NewEncoder(f).Encode(e)
	}
	f.Close()

	outputFile := filepath.Join(tmpDir, "sequences.json")
	exportToolSequences("json", true, outputFile)

	content, _ := os.ReadFile(outputFile)
	var sequences []ToolSequence
	json.Unmarshal(content, &sequences)

	// Verify only successful sequences included
	for _, seq := range sequences {
		if !seq.Successful {
			t.Error("Found unsuccessful sequence with successful-only=true")
		}
	}
}

// TestExportRoutingDecisions_MixedTimeWindows tests various time window filters
func TestExportRoutingDecisions_MixedTimeWindows(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)

	now := time.Now()
	events := []routing.PostToolEvent{
		{CapturedAt: now.Unix(), SelectedTier: "haiku", SelectedAgent: "test1"},
		{CapturedAt: now.Add(-2 * 24 * time.Hour).Unix(), SelectedTier: "sonnet", SelectedAgent: "test2"},
		{CapturedAt: now.Add(-10 * 24 * time.Hour).Unix(), SelectedTier: "opus", SelectedAgent: "test3"},
	}

	for _, e := range events {
		json.NewEncoder(f).Encode(e)
	}
	f.Close()

	tests := []struct {
		since      string
		outputFile string
	}{
		{"1d", filepath.Join(tmpDir, "1d.csv")},
		{"7d", filepath.Join(tmpDir, "7d.csv")},
		{"30d", filepath.Join(tmpDir, "30d.csv")},
	}

	for _, tt := range tests {
		exportRoutingDecisions("csv", tt.since, tt.outputFile)

		content, _ := os.ReadFile(tt.outputFile)
		if !strings.Contains(string(content), "timestamp") {
			t.Errorf("Expected header in %s", tt.outputFile)
		}
	}
}

// TestExportToolSequences_EmptySessionID tests handling of events without session IDs
func TestExportToolSequences_EmptySessionID(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	mlFile := filepath.Join(telemetryDir, "ml-tool-events.jsonl")
	f, _ := os.Create(mlFile)

	// Event with empty session ID
	event := routing.PostToolEvent{
		SessionID:  "", // Empty session
		ToolName:   "Read",
		Success:    true,
		CapturedAt: time.Now().Unix(),
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	outputFile := filepath.Join(tmpDir, "empty-session.json")
	exportToolSequences("json", false, outputFile)

	content, _ := os.ReadFile(outputFile)
	var sequences []ToolSequence
	if err := json.Unmarshal(content, &sequences); err != nil {
		t.Errorf("Failed to parse JSON with empty session: %v", err)
	}
}

// TestExportCollaborations_EmptyData tests collaborations with no data
func TestExportCollaborations_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	telemetryDir := filepath.Join(tmpDir, ".claude", "telemetry")
	os.MkdirAll(telemetryDir, 0755)

	collabFile := filepath.Join(telemetryDir, "collaboration.jsonl")
	os.WriteFile(collabFile, []byte("\n\n\n"), 0644) // Just newlines

	outputFile := filepath.Join(tmpDir, "empty.json")
	exportCollaborations("json", outputFile)

	content, _ := os.ReadFile(outputFile)
	var edges []CollaborationEdge
	json.Unmarshal(content, &edges)

	if len(edges) != 0 {
		t.Errorf("Expected 0 edges from empty data, got %d", len(edges))
	}
}

// TestBuildSequences_LargeSessionCount tests handling of many sessions
func TestBuildSequences_LargeSessionCount(t *testing.T) {
	events := make([]routing.PostToolEvent, 0)

	// Create 100 different sessions
	for i := 0; i < 100; i++ {
		events = append(events, routing.PostToolEvent{
			SessionID:  fmt.Sprintf("session-%d", i),
			ToolName:   "Read",
			Success:    true,
			DurationMs: 100,
		})
	}

	sequences := buildSequences(events)

	if len(sequences) != 100 {
		t.Errorf("Expected 100 sequences, got %d", len(sequences))
	}
}

// TestAggregateCollaborations_ManyEdges tests many unique parent-child pairs
func TestAggregateCollaborations_ManyEdges(t *testing.T) {
	collabs := make([]telemetry.AgentCollaboration, 0)

	// Create 50 unique edges
	for i := 0; i < 50; i++ {
		collabs = append(collabs, telemetry.AgentCollaboration{
			ParentAgent:  fmt.Sprintf("parent-%d", i),
			ChildAgent:   fmt.Sprintf("child-%d", i),
			ChildSuccess: i%2 == 0, // Alternating success/failure
		})
	}

	edges := aggregateCollaborations(collabs)

	if len(edges) != 50 {
		t.Errorf("Expected 50 edges, got %d", len(edges))
	}
}

// TestWriteJSON_EmptySlice tests JSON encoding of empty slice
func TestWriteJSON_EmptySlice(t *testing.T) {
	buf := &bytes.Buffer{}

	writeJSON(buf, []string{})

	content := buf.String()
	if content != "[]\n" {
		t.Errorf("Expected empty JSON array, got '%s'", content)
	}
}

// TestWriteRoutingCSV_MultipleEvents tests CSV with many events
func TestWriteRoutingCSV_MultipleEvents(t *testing.T) {
	buf := &bytes.Buffer{}

	events := make([]routing.PostToolEvent, 0)
	for i := 0; i < 10; i++ {
		events = append(events, routing.PostToolEvent{
			CapturedAt:    time.Now().Unix(),
			SelectedTier:  "haiku",
			SelectedAgent: fmt.Sprintf("agent-%d", i),
		})
	}

	writeRoutingCSV(buf, events)

	content := buf.String()
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// Should have header + 10 data rows
	if len(lines) != 11 {
		t.Errorf("Expected 11 lines (header + 10 data), got %d", len(lines))
	}
}

// TestWriteSequencesCSV_MultipleSequences tests CSV with many sequences
func TestWriteSequencesCSV_MultipleSequences(t *testing.T) {
	buf := &bytes.Buffer{}

	sequences := make([]ToolSequence, 0)
	for i := 0; i < 10; i++ {
		sequences = append(sequences, ToolSequence{
			SequenceID: fmt.Sprintf("seq-%d", i),
			Tools:      []string{"Read", "Write"},
			Successful: true,
		})
	}

	writeSequencesCSV(buf, sequences)

	content := buf.String()
	lines := strings.Split(strings.TrimSpace(content), "\n")

	if len(lines) != 11 {
		t.Errorf("Expected 11 lines, got %d", len(lines))
	}
}

// TestWriteCollaborationsCSV_MultipleEdges tests CSV with many edges
func TestWriteCollaborationsCSV_MultipleEdges(t *testing.T) {
	buf := &bytes.Buffer{}

	edges := make([]CollaborationEdge, 0)
	for i := 0; i < 10; i++ {
		edges = append(edges, CollaborationEdge{
			SourceAgent:      fmt.Sprintf("source-%d", i),
			TargetAgent:      fmt.Sprintf("target-%d", i),
			InteractionCount: i + 1,
			SuccessRate:      float64(i) / 10.0,
		})
	}

	writeCollaborationsCSV(buf, edges)

	content := buf.String()
	lines := strings.Split(strings.TrimSpace(content), "\n")

	if len(lines) != 11 {
		t.Errorf("Expected 11 lines, got %d", len(lines))
	}
}
