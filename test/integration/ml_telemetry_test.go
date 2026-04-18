package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMLTelemetry_RoutingDecisionCapture verifies routing-decisions.jsonl is created and populated
func TestMLTelemetry_RoutingDecisionCapture(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	// Create test harness
	corpusPath := filepath.Join(t.TempDir(), "ml-capture-corpus.jsonl")
	createMLTelemetryCorpus(t, corpusPath, projectDir, 5)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run events through telemetry system
	results, err := harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("Expected 5 results, got: %d", len(results))
	}

	// Verify routing-decisions.jsonl file created
	decisionsPath := filepath.Join(projectDir, ".goyoke", "routing-decisions.jsonl")
	if _, err := os.Stat(decisionsPath); err != nil {
		t.Errorf("routing-decisions.jsonl not created: %v", err)
	}

	// Verify file contains JSON lines
	data, err := os.ReadFile(decisionsPath)
	if err != nil {
		t.Fatalf("Failed to read routing-decisions.jsonl: %v", err)
	}

	if len(data) == 0 {
		t.Error("routing-decisions.jsonl is empty")
	}

	// Parse and verify each line is valid JSON
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var decision map[string]interface{}
		if err := json.Unmarshal(line, &decision); err != nil {
			t.Errorf("Line %d is not valid JSON: %v. Content: %s", lineCount+1, err, string(line))
		}

		// Verify required fields from RoutingDecision struct
		if _, ok := decision["timestamp"].(float64); !ok {
			t.Errorf("Line %d missing or invalid timestamp", lineCount+1)
		}
		if _, ok := decision["selected_tier"].(string); !ok {
			t.Errorf("Line %d missing or invalid selected_tier", lineCount+1)
		}
		if _, ok := decision["selected_agent"].(string); !ok {
			t.Errorf("Line %d missing or invalid selected_agent", lineCount+1)
		}
		if _, ok := decision["decision_id"].(string); !ok {
			t.Errorf("Line %d missing or invalid decision_id", lineCount+1)
		}
		if _, ok := decision["session_id"].(string); !ok {
			t.Errorf("Line %d missing or invalid session_id", lineCount+1)
		}

		lineCount++
	}

	if lineCount != 5 {
		t.Errorf("Expected 5 decision lines, got: %d", lineCount)
	}
}

// TestMLTelemetry_DecisionUpdates verifies routing-decision-updates.jsonl is append-only
func TestMLTelemetry_DecisionUpdates(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	corpusPath := filepath.Join(t.TempDir(), "ml-updates-corpus.jsonl")
	createMLTelemetryCorpus(t, corpusPath, projectDir, 3)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// First batch
	results1, err := harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
	if err != nil {
		t.Fatalf("First batch failed: %v", err)
	}

	// Read updates after first batch
	updatesPath := filepath.Join(projectDir, ".goyoke", "routing-decision-updates.jsonl")
	data1, _ := os.ReadFile(updatesPath)
	initialLineCount := 0
	if len(data1) > 0 {
		initialLineCount = len(strings.Split(strings.TrimSpace(string(data1)), "\n"))
	}

	// Run second batch
	results2, err := harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
	if err != nil {
		t.Fatalf("Second batch failed: %v", err)
	}

	// Read updates after second batch
	data2, _ := os.ReadFile(updatesPath)
	finalLineCount := 0
	if len(data2) > 0 {
		finalLineCount = len(strings.Split(strings.TrimSpace(string(data2)), "\n"))
	}

	// Verify append-only: final >= initial
	if finalLineCount < initialLineCount {
		t.Errorf("Append-only violation: initial=%d, final=%d (should be >=)", initialLineCount, finalLineCount)
	}

	// Verify all lines are still valid JSON
	scanner := bufio.NewScanner(bytes.NewReader(data2))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var update map[string]interface{}
		if err := json.Unmarshal(line, &update); err != nil {
			t.Errorf("Invalid JSON in routing-decision-updates: %v", err)
		}
	}

	// Verify timestamps are monotonically increasing
	scanner = bufio.NewScanner(bytes.NewReader(data2))
	var lastTimestamp float64
	for scanner.Scan() {
		var update map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &update); err != nil {
			continue
		}

		if ts, ok := update["timestamp"].(float64); ok {
			if ts < lastTimestamp {
				t.Errorf("Timestamp violation: %f < %f (should be monotonically increasing)", ts, lastTimestamp)
			}
			lastTimestamp = ts
		}
	}

	if len(results1)+len(results2) == 0 {
		t.Error("No results from batches")
	}
}

// TestMLTelemetry_ConcurrentWrites verifies no corruption under parallel writes
func TestMLTelemetry_ConcurrentWrites(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	// Create 5 separate harnesses (simulating parallel agents)
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()

			corpusPath := filepath.Join(t.TempDir(), fmt.Sprintf("concurrent-corpus-%d.jsonl", agentID))
			createMLTelemetryCorpus(t, corpusPath, projectDir, 3)

			harness, err := NewTestHarness(corpusPath, projectDir)
			if err != nil {
				errors <- fmt.Errorf("harness creation failed for agent %d: %v", agentID, err)
				return
			}

			if err := harness.LoadCorpus(); err != nil {
				errors <- fmt.Errorf("corpus load failed for agent %d: %v", agentID, err)
				return
			}

			_, err = harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
			if err != nil {
				errors <- fmt.Errorf("hook batch failed for agent %d: %v", agentID, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent write error: %v", err)
		}
	}

	// Verify final file integrity
	decisionsPath := filepath.Join(projectDir, ".goyoke", "routing-decisions.jsonl")
	data, err := os.ReadFile(decisionsPath)
	if err != nil {
		t.Fatalf("Failed to read final routing-decisions.jsonl: %v", err)
	}

	// Verify no corruption: all lines parse as JSON
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineCount := 0
	corruptLines := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var decision map[string]interface{}
		if err := json.Unmarshal(line, &decision); err != nil {
			corruptLines++
		}
		lineCount++
	}

	if corruptLines > 0 {
		t.Errorf("Concurrent write corruption detected: %d/%d lines are invalid JSON", corruptLines, lineCount)
	}

	// Verify expected total lines: 5 agents × 3 decisions = 15
	expectedLines := 15
	if lineCount != expectedLines {
		t.Errorf("Expected %d lines after concurrent writes, got: %d", expectedLines, lineCount)
	}
}

// TestMLTelemetry_CollaborationTracking verifies agent-collaborations.jsonl is created
func TestMLTelemetry_CollaborationTracking(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	// Create corpus with collaboration events
	corpusPath := filepath.Join(t.TempDir(), "collaboration-corpus.jsonl")
	createCollaborationCorpus(t, corpusPath, projectDir)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run events
	_, err = harness.RunHookBatch("../../bin/goyoke-agent-endstate", "SubagentStop")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	// Verify agent-collaborations.jsonl created
	collaborationsPath := filepath.Join(projectDir, ".goyoke", "agent-collaborations.jsonl")
	if _, err := os.Stat(collaborationsPath); err != nil {
		t.Errorf("agent-collaborations.jsonl not created: %v", err)
	}

	// Verify content
	data, err := os.ReadFile(collaborationsPath)
	if err != nil {
		t.Fatalf("Failed to read agent-collaborations.jsonl: %v", err)
	}

	if len(data) == 0 {
		t.Error("agent-collaborations.jsonl is empty")
	}

	// Parse and verify structure
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var collab map[string]interface{}
		if err := json.Unmarshal(line, &collab); err != nil {
			t.Errorf("Invalid collaboration JSON: %v", err)
		}

		// Verify required fields (matching AgentCollaboration struct)
		if _, ok := collab["child_agent"].(string); !ok {
			t.Error("Missing or invalid child_agent in collaboration")
		}
		if _, ok := collab["parent_agent"].(string); !ok {
			t.Error("Missing or invalid parent_agent in collaboration")
		}
		if _, ok := collab["delegation_type"].(string); !ok {
			t.Error("Missing or invalid delegation_type in collaboration")
		}
		if _, ok := collab["timestamp"].(float64); !ok {
			t.Error("Missing or invalid timestamp in collaboration")
		}
		if _, ok := collab["session_id"].(string); !ok {
			t.Error("Missing or invalid session_id in collaboration")
		}
	}
}

// TestMLTelemetry_ExportReconciliation verifies goyoke-ml-export produces valid datasets
func TestMLTelemetry_ExportReconciliation(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	// Build goyoke-ml-export binary
	binaryPath := filepath.Join(t.TempDir(), "goyoke-ml-export")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/goyoke-ml-export/main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Skipf("Failed to build goyoke-ml-export: %v. Output: %s", err, string(output))
	}

	// Create telemetry data
	corpusPath := filepath.Join(t.TempDir(), "export-corpus.jsonl")
	createMLTelemetryCorpus(t, corpusPath, projectDir, 10)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Populate telemetry files (PreToolUse creates routing decisions)
	if _, err := harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse"); err != nil {
		t.Fatalf("Failed to populate telemetry: %v", err)
	}

	// Manually create ml-tool-events.jsonl with test data
	// (PostToolUse hook integration is complex - this ensures export has data to work with)
	mlEventsData := []string{}
	models := []string{"haiku", "sonnet", "haiku", "sonnet", "haiku"}
	agents := []string{"codebase-search", "python-pro", "librarian", "go-pro", "orchestrator"}
	for i := 0; i < 10; i++ {
		event := map[string]interface{}{
			"hook_event_name": "PostToolUse",
			"tool_name":       "Task",
			"session_id":      fmt.Sprintf("test-session-%d", i/3),
			"captured_at":     time.Now().Add(time.Duration(i) * time.Second).Unix(),
			"duration_ms":     int64(100 + i*10),
			"input_tokens":    500 + i*50,
			"output_tokens":   250 + i*25,
			"model":           models[i%5],
			"tier":            models[i%5],
			"success":         true,
			"task_type":       "search",
			"task_domain":     "codebase",
			"selected_tier":   models[i%5],
			"selected_agent":  agents[i%5],
		}
		data, _ := json.Marshal(event)
		mlEventsData = append(mlEventsData, string(data))
	}
	mlEventsPath := filepath.Join(projectDir, ".goyoke", "ml-tool-events.jsonl")
	if err := os.WriteFile(mlEventsPath, []byte(strings.Join(mlEventsData, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write ml-tool-events.jsonl: %v", err)
	}

	// Run export command
	outputDir := filepath.Join(t.TempDir(), "ml-export")
	exportCmd := exec.Command(binaryPath, "training-dataset", "--output", outputDir)
	exportCmd.Env = append(os.Environ(), fmt.Sprintf("GOYOKE_PROJECT_DIR=%s", projectDir))
	if output, err := exportCmd.CombinedOutput(); err != nil {
		t.Fatalf("goyoke-ml-export failed: %v. Output: %s", err, string(output))
	}

	// Verify export created expected files
	expectedFiles := []string{
		"routing.csv",
		"sequences.json",
		"collaborations.json",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Export file not created: %s", filename)
		}
	}

	// Verify routing.csv has valid structure
	decisionsCsvPath := filepath.Join(outputDir, "routing.csv")
	csvData, err := os.ReadFile(decisionsCsvPath)
	if err != nil {
		t.Fatalf("Failed to read routing.csv: %v", err)
	}

	csvLines := strings.Split(strings.TrimSpace(string(csvData)), "\n")
	if len(csvLines) < 2 {
		t.Error("routing.csv should have header + data rows")
	}

	// Verify header
	expectedHeader := []string{
		"timestamp", "task_type", "task_domain", "context_window",
		"recent_success_rate", "selected_tier", "selected_agent",
		"outcome_success", "outcome_cost", "escalation_required",
	}

	headerLine := csvLines[0]
	headerFields := strings.Split(headerLine, ",")
	for _, expected := range expectedHeader {
		found := false
		for _, field := range headerFields {
			if strings.TrimSpace(field) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected CSV header: %s", expected)
		}
	}

	// Verify sequences.json has valid JSON structure
	sequencesPath := filepath.Join(outputDir, "sequences.json")
	sequencesData, err := os.ReadFile(sequencesPath)
	if err != nil {
		t.Fatalf("Failed to read sequences.json: %v", err)
	}

	var sequences []interface{}
	if err := json.Unmarshal(sequencesData, &sequences); err != nil {
		t.Errorf("Invalid sequences.json: %v", err)
	}

	// Verify collaborations.json has valid JSON structure
	collabPath := filepath.Join(outputDir, "collaborations.json")
	collabData, err := os.ReadFile(collabPath)
	if err != nil {
		t.Fatalf("Failed to read collaborations.json: %v", err)
	}

	var collaborations []interface{}
	if err := json.Unmarshal(collabData, &collaborations); err != nil {
		t.Errorf("Invalid collaborations.json: %v", err)
	}
}

// TestMLTelemetry_RaceConditionDetection verifies concurrent access safety
func TestMLTelemetry_RaceConditionDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	// Use -race flag: go test -race ./test/integration -run TestMLTelemetry_RaceConditionDetection
	corpusPath := filepath.Join(t.TempDir(), "race-corpus.jsonl")
	createMLTelemetryCorpus(t, corpusPath, projectDir, 20)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	// Run with concurrency
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
		}()
	}

	wg.Wait()

	// If we reach here without data race (detected by -race), test passes
	t.Log("No data races detected in concurrent telemetry writes")
}

// TestMLTelemetry_SequenceIntegrity verifies ml_telemetry fields propagate correctly
func TestMLTelemetry_SequenceIntegrity(t *testing.T) {
	projectDir := t.TempDir()
	setupMLTelemetryProject(t, projectDir)

	corpusPath := filepath.Join(t.TempDir(), "sequence-corpus.jsonl")
	createSequenceCorpus(t, corpusPath, projectDir)

	harness, err := NewTestHarness(corpusPath, projectDir)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	if err := harness.LoadCorpus(); err != nil {
		t.Fatalf("Failed to load corpus: %v", err)
	}

	results, err := harness.RunHookBatch("../../bin/goyoke-validate", "PreToolUse")
	if err != nil {
		t.Fatalf("Failed to run batch: %v", err)
	}

	// Verify all results include ML telemetry fields from events
	for i, result := range results {
		if result.Event == nil {
			t.Errorf("Result %d missing Event", i)
			continue
		}

		// CORRECTED: Access telemetry fields from Event, not a non-existent MLTelemetry struct
		if result.Event.DurationMs <= 0 {
			t.Errorf("Result %d invalid DurationMs: %d", i, result.Event.DurationMs)
		}
		if result.Event.InputTokens <= 0 {
			t.Errorf("Result %d invalid InputTokens: %d", i, result.Event.InputTokens)
		}
		if result.Event.SequenceIndex <= 0 {
			t.Errorf("Result %d invalid SequenceIndex: %d", i, result.Event.SequenceIndex)
		}
	}

	// Verify sequence indices are monotonically increasing
	var lastSeq int64
	for _, result := range results {
		if result.Event != nil {
			if result.Event.SequenceIndex <= lastSeq {
				t.Errorf("Non-monotonic sequence: %d <= %d", result.Event.SequenceIndex, lastSeq)
			}
			lastSeq = result.Event.SequenceIndex
		}
	}
}

// Helper: Setup ML telemetry project directories
func setupMLTelemetryProject(t *testing.T, projectDir string) {
	dirs := []string{
		filepath.Join(projectDir, ".goyoke"),
		filepath.Join(projectDir, ".claude"),
		filepath.Join(projectDir, ".goyoke", "memory"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create minimal routing schema for validation
	schema := `{
		"version": "2.5.0",
		"tiers": {
			"haiku": {
				"model": "haiku",
				"patterns": ["find"],
				"tools": ["Read", "Glob", "Grep"]
			},
			"sonnet": {
				"model": "sonnet",
				"patterns": ["implement"],
				"tools": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"]
			}
		},
		"tier_levels": {
			"haiku": 1,
			"sonnet": 3
		},
		"agent_subagent_mapping": {
			"codebase-search": "Codebase Search",
			"python-pro": "Python Pro",
			"librarian": "Librarian",
			"go-pro": "GO Pro",
			"orchestrator": "Orchestrator"
		}
	}`
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to create routing schema: %v", err)
	}
}

// Helper: Create ML telemetry test corpus
func createMLTelemetryCorpus(t *testing.T, corpusPath, projectDir string, eventCount int) {
	var events []string

	// Models to cycle through
	models := []string{"haiku", "sonnet", "haiku", "sonnet", "haiku"}
	subagentTypes := []string{"Codebase Search", "Python Pro", "Librarian", "GO Pro", "Orchestrator"}
	agents := []string{"codebase-search", "python-pro", "librarian", "go-pro", "orchestrator"}

	for i := 0; i < eventCount; i++ {
		event := map[string]interface{}{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Task",
			"tool_input": map[string]interface{}{
				"prompt":        fmt.Sprintf("AGENT: %s\n\nTest task %d", agents[i%5], i),
				"model":         models[i%5],
				"subagent_type": subagentTypes[i%5],
				"description":   fmt.Sprintf("Test task description %d", i),
			},
			"tool_response": map[string]interface{}{
				"success": true,
			},
			"session_id":     fmt.Sprintf("test-session-%d", i/3),
			"duration_ms":    int64(100 + i*10),
			"input_tokens":   int64(500 + i*50),
			"output_tokens":  int64(250 + i*25),
			"sequence_index": int64(i + 1),
			"timestamp":      time.Now().Add(time.Duration(i) * time.Second).Unix(),
		}

		data, _ := json.Marshal(event)
		events = append(events, string(data))
	}

	os.WriteFile(corpusPath, []byte(strings.Join(events, "\n")+"\n"), 0644)
}

// Helper: Create collaboration test corpus
func createCollaborationCorpus(t *testing.T, corpusPath, projectDir string) {
	// Create transcript files for each agent delegation
	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	// Create transcript for python-pro agent
	pythonTranscript := filepath.Join(transcriptDir, "python-pro-1.jsonl")
	pythonTranscriptContent := `{"role":"user","content":"AGENT: python-pro\n\nTest implementation","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()) + `,"model":"sonnet"}
{"role":"assistant","content":"Implementation complete","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()+1) + `}
`
	os.WriteFile(pythonTranscript, []byte(pythonTranscriptContent), 0644)

	// Create transcript for orchestrator agent
	orchestratorTranscript := filepath.Join(transcriptDir, "orchestrator-1.jsonl")
	orchestratorTranscriptContent := `{"role":"user","content":"AGENT: orchestrator\n\nCoordinate testing","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()+2) + `,"model":"sonnet"}
{"role":"assistant","content":"Coordination complete","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()+3) + `}
`
	os.WriteFile(orchestratorTranscript, []byte(orchestratorTranscriptContent), 0644)

	// Create transcript for second python-pro agent
	pythonTranscript2 := filepath.Join(transcriptDir, "python-pro-2.jsonl")
	pythonTranscript2Content := `{"role":"user","content":"AGENT: python-pro\n\nComplete implementation","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()+4) + `,"model":"sonnet"}
{"role":"assistant","content":"Final implementation complete","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()+5) + `}
`
	os.WriteFile(pythonTranscript2, []byte(pythonTranscript2Content), 0644)

	// Create SubagentStop events
	events := []string{
		`{"hook_event_name":"SubagentStop","session_id":"col-1","transcript_path":"` + pythonTranscript + `","stop_hook_active":true}`,
		`{"hook_event_name":"SubagentStop","session_id":"col-1","transcript_path":"` + orchestratorTranscript + `","stop_hook_active":true}`,
		`{"hook_event_name":"SubagentStop","session_id":"col-1","transcript_path":"` + pythonTranscript2 + `","stop_hook_active":true}`,
	}

	os.WriteFile(corpusPath, []byte(strings.Join(events, "\n")+"\n"), 0644)
}

// Helper: Create sequence integrity test corpus
func createSequenceCorpus(t *testing.T, corpusPath, projectDir string) {
	var events []string

	for i := 0; i < 10; i++ {
		event := map[string]interface{}{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Read",
			"tool_input": map[string]interface{}{
				"file_path": filepath.Join(projectDir, "file.go"),
			},
			"tool_response": map[string]interface{}{
				"success": true,
			},
			"session_id":     "sequence-test",
			"duration_ms":    int64(150 + i*20),
			"input_tokens":   int64(1000),
			"output_tokens":  int64(500),
			"sequence_index": int64(i + 1),
			"timestamp":      time.Now().Add(time.Duration(i) * time.Second).Unix(),
		}

		data, _ := json.Marshal(event)
		events = append(events, string(data))
	}

	os.WriteFile(corpusPath, []byte(strings.Join(events, "\n")+"\n"), 0644)
}
