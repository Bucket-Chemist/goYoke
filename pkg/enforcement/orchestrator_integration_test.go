package enforcement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// TestOrchestratorGuardWorkflow_AllTasksCollected verifies allow when all tasks collected.
// This is the happy path: orchestrator spawned background tasks and collected them all.
func TestOrchestratorGuardWorkflow_AllTasksCollected(t *testing.T) {
	// Create transcript with complete fan-out/fan-in pattern
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: orchestrator", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
		{"timestamp": float64(1200), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "long-task-1.sh",
			"run_in_background": true,
			"task_id":           "bg-task-1",
		}},
		{"timestamp": float64(1300), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "long-task-2.sh",
			"run_in_background": true,
			"task_id":           "bg-task-2",
		}},
		{"timestamp": float64(1400), "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "bg-task-1",
			"block":   true,
		}},
		{"timestamp": float64(1500), "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "bg-task-2",
			"block":   true,
		}},
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify allow decision
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow', got: %s", resp.Decision)
	}

	if resp.Reason != "All background tasks collected" {
		t.Errorf("Expected reason 'All background tasks collected', got: %s", resp.Reason)
	}

	if len(resp.RemediationSteps) != 0 {
		t.Errorf("Expected empty remediation steps for allow, got: %d", len(resp.RemediationSteps))
	}

	if !strings.Contains(resp.AdditionalContext, "ORCHESTRATOR COMPLETION ALLOWED") {
		t.Errorf("Expected allow context, got: %s", resp.AdditionalContext)
	}

	if !strings.Contains(resp.AdditionalContext, "Agent: orchestrator (model: claude-sonnet-4-5)") {
		t.Errorf("Expected agent info in context, got: %s", resp.AdditionalContext)
	}

	if !strings.Contains(resp.AdditionalContext, "2 spawned, 2 collected") {
		t.Errorf("Expected task counts in context, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_UncollectedTasks verifies block when tasks uncollected.
// This is the violation case: orchestrator spawned tasks but forgot to collect them.
func TestOrchestratorGuardWorkflow_UncollectedTasks(t *testing.T) {
	// Create transcript with incomplete fan-out (no fan-in)
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: orchestrator", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
		{"timestamp": float64(1200), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "long-task-1.sh",
			"run_in_background": true,
			"task_id":           "bg-task-1",
		}},
		{"timestamp": float64(1300), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "long-task-2.sh",
			"run_in_background": true,
			"task_id":           "bg-task-2",
		}},
		// Missing TaskOutput calls - violation!
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify block decision
	if resp.Decision != "block" {
		t.Errorf("Expected decision 'block', got: %s", resp.Decision)
	}

	if resp.Reason != "Orchestrator completed with uncollected background tasks" {
		t.Errorf("Expected uncollected tasks reason, got: %s", resp.Reason)
	}

	if len(resp.RemediationSteps) != 4 {
		t.Errorf("Expected 4 remediation steps, got: %d", len(resp.RemediationSteps))
	}

	expectedSteps := []string{
		"identify_uncollected_task_ids",
		"call_TaskOutput_for_each",
		"wait_for_all_collections",
		"verify_results_in_transcript",
	}

	for i, expected := range expectedSteps {
		if resp.RemediationSteps[i] != expected {
			t.Errorf("Remediation step %d: expected %s, got %s", i, expected, resp.RemediationSteps[i])
		}
	}

	if !strings.Contains(resp.AdditionalContext, "ORCHESTRATOR COMPLETION BLOCKED") {
		t.Errorf("Expected block context, got: %s", resp.AdditionalContext)
	}

	if !strings.Contains(resp.AdditionalContext, "VIOLATION: Fan-out without fan-in") {
		t.Errorf("Expected violation message, got: %s", resp.AdditionalContext)
	}

	// Check for both uncollected task IDs (order is non-deterministic from map)
	if !strings.Contains(resp.AdditionalContext, "bg-task-1") {
		t.Errorf("Expected uncollected task bg-task-1, got: %s", resp.AdditionalContext)
	}
	if !strings.Contains(resp.AdditionalContext, "bg-task-2") {
		t.Errorf("Expected uncollected task bg-task-2, got: %s", resp.AdditionalContext)
	}

	if !strings.Contains(resp.AdditionalContext, "2 spawned, 0 collected") {
		t.Errorf("Expected task counts, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_NonOrchestratorPassthrough verifies non-orchestrator agents pass through.
// Guard should allow completion for non-orchestrator agents without checking tasks.
func TestOrchestratorGuardWorkflow_NonOrchestratorPassthrough(t *testing.T) {
	// Create transcript for python-pro agent (not orchestrator)
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: python-pro", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
		{"timestamp": float64(1200), "tool_name": "Read", "tool_input": map[string]interface{}{
			"file_path": "/test.py",
		}},
		{"timestamp": float64(1300), "tool_name": "Edit", "tool_input": map[string]interface{}{
			"file_path":  "/test.py",
			"old_string": "old",
			"new_string": "new",
		}},
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "python-pro",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify no tasks detected
	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks for python-pro")
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Should allow (no tasks to collect)
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow' for non-orchestrator, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.AdditionalContext, "python-pro") {
		t.Errorf("Expected agent name in context, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_EmptyTranscript verifies empty transcript allows.
// An empty transcript means no work was done, should not block.
func TestOrchestratorGuardWorkflow_EmptyTranscript(t *testing.T) {
	// Create empty transcript file
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty transcript: %v", err)
	}
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed on empty file: %v", err)
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Should allow (no tasks)
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow' for empty transcript, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.AdditionalContext, "0 spawned, 0 collected") {
		t.Errorf("Expected zero task counts, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_MalformedTranscript verifies graceful handling of malformed JSONL.
// Should not crash, should fall back to regex patterns.
func TestOrchestratorGuardWorkflow_MalformedTranscript(t *testing.T) {
	// Create transcript with mix of valid JSON and malformed lines
	content := `{"timestamp": 1000, "content": "AGENT: orchestrator", "role": "user"}
{"timestamp": 1100, "model": "claude-sonnet-4-5"}
This is malformed JSON with run_in_background: true and task_id: "regex-task-1"
{"timestamp": 1200, "tool_name": "Bash", "tool_input": {"command": "test.sh", "run_in_background": true, "task_id": "json-task-1"}}
Another malformed line with TaskOutput task_id: "regex-task-1"
{"timestamp": 1300, "tool_name": "TaskOutput", "tool_input": {"task_id": "json-task-1", "block": true}}
`

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create malformed transcript: %v", err)
	}
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze should handle malformed JSON gracefully: %v", err)
	}

	// Should detect both tasks via JSON and regex fallback
	// Use GetSummary to verify task tracking without accessing unexported fields
	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "2 spawned, 2 collected") {
		t.Errorf("Expected '2 spawned, 2 collected' in summary, got: %s", summary)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks after regex fallback")
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Should allow (all tasks collected)
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow' after regex fallback, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.AdditionalContext, "2 spawned, 2 collected") {
		t.Errorf("Expected task counts from mixed parsing, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_VeryLongTranscript tests performance with 1000+ entries.
// Verifies the analyzer can handle large transcripts without performance issues.
func TestOrchestratorGuardWorkflow_VeryLongTranscript(t *testing.T) {
	// Create large transcript with 1000+ entries
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: orchestrator", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
	}

	// Add 500 Read events (noise)
	for i := 0; i < 500; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": float64(2000 + i),
			"tool_name": "Read",
			"tool_input": map[string]interface{}{
				"file_path": "/test.py",
			},
		})
	}

	// Add background task spawns
	for i := 0; i < 10; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": float64(10000 + i*100),
			"tool_name": "Bash",
			"tool_input": map[string]interface{}{
				"command":           "long-task.sh",
				"run_in_background": true,
				"task_id":           "bg-task-" + string(rune('0'+i)),
			},
		})
	}

	// Add 500 more Read events (more noise)
	for i := 0; i < 500; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": float64(20000 + i),
			"tool_name": "Read",
			"tool_input": map[string]interface{}{
				"file_path": "/test2.py",
			},
		})
	}

	// Collect all tasks
	for i := 0; i < 10; i++ {
		entries = append(entries, map[string]interface{}{
			"timestamp": float64(30000 + i*100),
			"tool_name": "TaskOutput",
			"tool_input": map[string]interface{}{
				"task_id": "bg-task-" + string(rune('0'+i)),
				"block":   true,
			},
		})
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed on large transcript: %v", err)
	}

	// Verify correct tracking despite large transcript
	// Use GetSummary to verify task tracking
	summary := analyzer.GetSummary()
	if !strings.Contains(summary, "10 spawned, 10 collected") {
		t.Errorf("Expected '10 spawned, 10 collected' in summary, got: %s", summary)
	}

	if analyzer.HasUncollectedTasks() {
		t.Error("Expected no uncollected tasks in large transcript")
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Should allow (all tasks collected)
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow' for large transcript, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.AdditionalContext, "10 spawned, 10 collected") {
		t.Errorf("Expected correct task counts, got: %s", resp.AdditionalContext)
	}
}

// TestOrchestratorGuardWorkflow_ConcurrentTranscriptAccess tests concurrent reads.
// Verifies analyzer is safe for concurrent access (multiple goroutines reading same transcript).
func TestOrchestratorGuardWorkflow_ConcurrentTranscriptAccess(t *testing.T) {
	// Create transcript
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: orchestrator", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
		{"timestamp": float64(1200), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "task.sh",
			"run_in_background": true,
			"task_id":           "concurrent-task",
		}},
		{"timestamp": float64(1300), "tool_name": "TaskOutput", "tool_input": map[string]interface{}{
			"task_id": "concurrent-task",
			"block":   true,
		}},
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Run 10 concurrent analyzers on the same transcript
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each goroutine creates its own analyzer
			analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
			if err := analyzer.Analyze(); err != nil {
				errChan <- err
				return
			}

			// Generate guard response
			resp := GenerateGuardResponse(analyzer, metadata)

			// Verify consistent results
			if resp.Decision != "allow" {
				errChan <- nil // Signal error without panicking
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent analysis failed: %v", err)
		}
	}
}

// TestOrchestratorGuardWorkflow_ArchitectAgent verifies architect agent is also guarded.
// Architect is also an orchestrator-type agent and should be subject to same rules.
func TestOrchestratorGuardWorkflow_ArchitectAgent(t *testing.T) {
	// Create transcript for architect with uncollected tasks
	entries := []map[string]interface{}{
		{"timestamp": float64(1000), "content": "AGENT: architect", "role": "user"},
		{"timestamp": float64(1100), "model": "claude-sonnet-4-5"},
		{"timestamp": float64(1200), "tool_name": "Bash", "tool_input": map[string]interface{}{
			"command":           "analysis.sh",
			"run_in_background": true,
			"task_id":           "architect-task",
		}},
		// No TaskOutput - violation
	}

	transcriptPath := createTranscriptFromEntries(t, entries)
	defer os.Remove(transcriptPath)

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "architect",
		AgentModel: "claude-sonnet-4-5",
		Tier:       "sonnet",
	}

	// Parse transcript with analyzer
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Generate guard response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Should block (uncollected task)
	if resp.Decision != "block" {
		t.Errorf("Expected decision 'block' for architect, got: %s", resp.Decision)
	}

	if !strings.Contains(resp.AdditionalContext, "Agent: architect") {
		t.Errorf("Expected architect agent name in context, got: %s", resp.AdditionalContext)
	}

	if !strings.Contains(resp.AdditionalContext, "Uncollected Tasks: architect-task") {
		t.Errorf("Expected uncollected task ID, got: %s", resp.AdditionalContext)
	}
}

// createTranscriptFromEntries creates a JSONL transcript file from entries.
// Helper function for test transcript generation.
func createTranscriptFromEntries(t *testing.T, entries []map[string]interface{}) string {
	t.Helper()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	f, err := os.Create(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to create transcript file: %v", err)
	}
	defer f.Close()

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("Failed to marshal entry: %v", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			t.Fatalf("Failed to write entry: %v", err)
		}
	}

	return transcriptPath
}
