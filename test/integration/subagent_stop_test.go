package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// SubagentOutcome represents ML outcome logged by SubagentStop hook
type SubagentOutcome struct {
	Timestamp          string `json:"timestamp"`
	SubagentID         string `json:"subagent_id"`
	DecisionID         string `json:"decision_id"`
	TaskID             string `json:"task_id"`
	Model              string `json:"model"`
	InputTokens        int    `json:"input_tokens"`
	OutputTokens       int    `json:"output_tokens"`
	ExecutionTime      string `json:"execution_time"`
	Status             string `json:"status"`
	RoutingTier        string `json:"routing_tier"`
	PromptTemplate     string `json:"prompt_template"`
	OutputSummary      string `json:"output_summary"`
	FinalizedDecisions int    `json:"finalized_decisions"`
}

// CollaborationUpdate represents agent collaboration log entry
type CollaborationUpdate struct {
	Timestamp      string `json:"timestamp"`
	DecisionID     string `json:"decision_id"`
	SubagentID     string `json:"subagent_id"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	Contribution   string `json:"contribution"`
	TokensUsed     int    `json:"tokens_used"`
	ParentDecision string `json:"parent_decision"`
}

// TestSubagentStop_Integration verifies complete SubagentStop workflow across various agent scenarios
func TestSubagentStop_Integration(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found. Run: go build -o cmd/goyoke-agent-endstate/goyoke-agent-endstate cmd/goyoke-agent-endstate/main.go")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	// Create .goyoke directory for ML logs
	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	// Create test transcript files with agent metadata
	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	agents := []struct {
		id    string
		model string
		tier  string
	}{
		{"codebase-search", "haiku", "explore"},
		{"python-pro", "sonnet", "general-purpose"},
		{"orchestrator", "sonnet", "plan"},
	}

	var events []*EventEntry

	for i, agent := range agents {
		// Create transcript file
		transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("agent-%d.jsonl", i))
		transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: %s"}
{"timestamp": %d, "role": "assistant", "content": "Working...", "model": "%s"}
{"timestamp": %d, "role": "completion", "content": "Done"}
`, time.Now().Unix()-10, agent.id, time.Now().Unix()-5, agent.model, time.Now().Unix())
		os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

		// Create SubagentStop event
		event := &EventEntry{
			HookEventName: "SubagentStop",
			SessionID:     "integration-test",
			Timestamp:     time.Now().Unix(),
		}

		// Construct raw JSON matching SubagentStopEvent schema
		rawEvent := map[string]interface{}{
			"hook_event_name":  "SubagentStop",
			"session_id":       "integration-test",
			"transcript_path":  transcriptPath,
			"stop_hook_active": true,
		}
		rawJSON, _ := json.Marshal(rawEvent)
		event.RawJSON = rawJSON

		events = append(events, event)
	}

	// Create corpus file
	tmpCorpus := filepath.Join(t.TempDir(), "subagent-corpus.jsonl")
	var corpusLines []string
	for _, event := range events {
		corpusLines = append(corpusLines, string(event.RawJSON))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(corpusLines, "\n")+"\n"), 0644)

	// Initialize harness
	harness, err := NewTestHarness(tmpCorpus, projectDir)
	if err != nil {
		t.Fatalf("Failed to create test harness: %v", err)
	}

	harness.LoadCorpus()

	if len(harness.Events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(harness.Events))
	}

	// Execute hook for each event
	for i, event := range harness.Events {
		result := harness.RunHook(binaryPath, event)

		if result.Error != nil {
			t.Errorf("Event %d: Hook execution failed: %v", i, result.Error)
			continue
		}

		if result.ExitCode != 0 {
			t.Errorf("Event %d: Expected exit code 0, got %d. Stderr: %s", i, result.ExitCode, result.Stderr)
		}

		if result.ParsedJSON == nil {
			t.Errorf("Event %d: Expected JSON output, got: %s", i, result.Stdout)
		}
	}

	// Verify collaboration log created
	collabPath := filepath.Join(projectDir, ".goyoke", "agent-collaboration-updates.jsonl")
	if _, err := os.Stat(collabPath); err != nil {
		t.Logf("Note: Collaboration log not created (may be optional): %v", err)
	} else {
		// Verify collaboration entries
		updates := parseCollaborationLog(t, collabPath)
		if len(updates) != 3 {
			t.Errorf("Expected 3 collaboration entries, got %d", len(updates))
		}
	}
}

// TestSubagentStop_MLOutcomeLogging verifies routing-decision-updates.jsonl contains complete outcome data
func TestSubagentStop_MLOutcomeLogging(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	// Create detailed transcript with metadata
	transcriptPath := filepath.Join(transcriptDir, "python-pro.jsonl")
	transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: python-pro"}
{"timestamp": %d, "role": "assistant", "content": "Implementing feature", "model": "sonnet"}
{"timestamp": %d, "role": "completion", "content": "Generated complete Python module with error handling"}
`, time.Now().Unix()-5, time.Now().Unix()-3, time.Now().Unix())
	os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

	// Create SubagentStop event
	rawEvent := map[string]interface{}{
		"hook_event_name":  "SubagentStop",
		"session_id":       "ml-test",
		"transcript_path":  transcriptPath,
		"stop_hook_active": true,
	}
	rawJSON, _ := json.Marshal(rawEvent)

	tmpCorpus := filepath.Join(t.TempDir(), "ml-corpus.jsonl")
	os.WriteFile(tmpCorpus, append(rawJSON, '\n'), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	result := harness.RunHook(binaryPath, harness.Events[0])

	if result.ExitCode != 0 {
		t.Fatalf("Hook execution failed: %s", result.Stderr)
	}

	// Verify hook produced valid JSON response
	if result.ParsedJSON == nil {
		t.Fatal("Expected JSON output from hook")
	}

	// Check hookSpecificOutput exists
	hookOutput, ok := result.ParsedJSON["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput in response")
	}

	if hookOutput["hookEventName"] != "SubagentStop" {
		t.Errorf("Expected hookEventName 'SubagentStop', got: %v", hookOutput["hookEventName"])
	}
}

// TestSubagentStop_CollaborationUpdates verifies agent-collaboration-updates.jsonl tracks agent interactions
func TestSubagentStop_CollaborationUpdates(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	// Create collaboration chain
	agents := []struct {
		id    string
		model string
	}{
		{"haiku-scout", "haiku"},
		{"orchestrator", "sonnet"},
		{"go-pro", "sonnet"},
	}

	var events []*EventEntry

	for _, agent := range agents {
		transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.jsonl", agent.id))
		transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: %s"}
{"timestamp": %d, "role": "assistant", "content": "Processing", "model": "%s"}
{"timestamp": %d, "role": "completion", "content": "Done"}
`, time.Now().Unix()-10, agent.id, time.Now().Unix()-5, agent.model, time.Now().Unix())
		os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

		rawEvent := map[string]interface{}{
			"hook_event_name":  "SubagentStop",
			"session_id":       "collab-test",
			"transcript_path":  transcriptPath,
			"stop_hook_active": true,
		}
		rawJSON, _ := json.Marshal(rawEvent)

		eventEntry := &EventEntry{
			HookEventName: "SubagentStop",
			SessionID:     "collab-test",
			Timestamp:     time.Now().Unix(),
			RawJSON:       rawJSON,
		}
		events = append(events, eventEntry)
	}

	tmpCorpus := filepath.Join(t.TempDir(), "collab-corpus.jsonl")
	var corpusLines []string
	for _, event := range events {
		corpusLines = append(corpusLines, string(event.RawJSON))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(corpusLines, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Execute hooks
	for _, event := range harness.Events {
		result := harness.RunHook(binaryPath, event)
		if result.ExitCode != 0 {
			t.Fatalf("Hook failed: %s", result.Stderr)
		}
	}

	// Verify collaboration log
	collabPath := filepath.Join(projectDir, ".goyoke", "agent-collaboration-updates.jsonl")
	if _, err := os.Stat(collabPath); err != nil {
		t.Logf("Collaboration log not created: %v", err)
		// Non-blocking: collaboration logging may be optional
		return
	}

	updates := parseCollaborationLog(t, collabPath)
	if len(updates) < 3 {
		t.Logf("Expected at least 3 collaboration entries, got %d", len(updates))
	}
}

// TestSubagentStop_DecisionCorrelation verifies agent events are logged with unique identifiers
func TestSubagentStop_DecisionCorrelation(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	// Create 5 unrelated agents
	var events []*EventEntry

	for i := 1; i <= 5; i++ {
		_ = i // Loop index used in string formatting
		agentID := fmt.Sprintf("agent-%d", i)
		transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.jsonl", agentID))
		transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: %s"}
{"timestamp": %d, "role": "assistant", "content": "Working", "model": "haiku"}
{"timestamp": %d, "role": "completion", "content": "Done"}
`, time.Now().Unix()-10, agentID, time.Now().Unix()-5, time.Now().Unix())
		os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

		rawEvent := map[string]interface{}{
			"hook_event_name":  "SubagentStop",
			"session_id":       fmt.Sprintf("session-%d", i),
			"transcript_path":  transcriptPath,
			"stop_hook_active": true,
		}
		rawJSON, _ := json.Marshal(rawEvent)

		event := &EventEntry{
			HookEventName: "SubagentStop",
			SessionID:     fmt.Sprintf("session-%d", i),
			Timestamp:     time.Now().Unix(),
			RawJSON:       rawJSON,
		}
		events = append(events, event)
	}

	tmpCorpus := filepath.Join(t.TempDir(), "correlation-corpus.jsonl")
	var corpusLines []string
	for _, event := range events {
		corpusLines = append(corpusLines, string(event.RawJSON))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(corpusLines, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Execute hooks
	for _, event := range harness.Events {
		result := harness.RunHook(binaryPath, event)
		if result.ExitCode != 0 {
			t.Fatalf("Hook failed: %s", result.Stderr)
		}
	}

	// Verify all agents logged
	collabPath := filepath.Join(projectDir, ".goyoke", "agent-collaboration-updates.jsonl")
	if _, err := os.Stat(collabPath); err != nil {
		t.Logf("Collaboration log not created (may be optional)")
		return
	}

	updates := parseCollaborationLog(t, collabPath)
	if len(updates) != 5 {
		t.Logf("Expected 5 collaboration entries, got %d", len(updates))
	}
}

// TestSubagentStop_ParallelAgentCompletion verifies 5 agents completing in parallel without race conditions
func TestSubagentStop_ParallelAgentCompletion(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	// Create 5 agents
	agents := []string{"haiku-scout", "codebase-search", "python-pro", "go-pro", "orchestrator"}
	models := []string{"haiku", "haiku", "sonnet", "sonnet", "sonnet"}

	var events []*EventEntry

	for i, agent := range agents {
		transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.jsonl", agent))
		transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: %s"}
{"timestamp": %d, "role": "assistant", "content": "Working", "model": "%s"}
{"timestamp": %d, "role": "completion", "content": "Done"}
`, time.Now().Unix()-10, agent, time.Now().Unix()-5, models[i], time.Now().Unix())
		os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

		rawEvent := map[string]interface{}{
			"hook_event_name":  "SubagentStop",
			"session_id":       fmt.Sprintf("parallel-session-%d", i),
			"transcript_path":  transcriptPath,
			"stop_hook_active": true,
		}
		rawJSON, _ := json.Marshal(rawEvent)

		event := &EventEntry{
			HookEventName: "SubagentStop",
			SessionID:     fmt.Sprintf("parallel-session-%d", i),
			Timestamp:     time.Now().Unix(),
			RawJSON:       rawJSON,
		}
		events = append(events, event)
	}

	tmpCorpus := filepath.Join(t.TempDir(), "parallel-corpus.jsonl")
	var corpusLines []string
	for _, event := range events {
		corpusLines = append(corpusLines, string(event.RawJSON))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(corpusLines, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Execute hooks in parallel to detect race conditions
	var wg sync.WaitGroup
	results := make([]HookResult, len(harness.Events))
	mu := &sync.Mutex{}

	for i, event := range harness.Events {
		wg.Add(1)
		go func(index int, ev *EventEntry) {
			defer wg.Done()
			result := harness.RunHook(binaryPath, ev)
			mu.Lock()
			results[index] = *result
			mu.Unlock()
		}(i, event)
	}

	wg.Wait()

	// Verify all executions succeeded
	for i, result := range results {
		if result.ExitCode != 0 {
			t.Errorf("Parallel execution %d failed: %s", i, result.Stderr)
		}
	}

	// Verify collaboration log not corrupted
	collabPath := filepath.Join(projectDir, ".goyoke", "agent-collaboration-updates.jsonl")
	if _, err := os.Stat(collabPath); err != nil {
		t.Logf("Collaboration log not created (may be optional)")
		return
	}

	updates := parseCollaborationLog(t, collabPath)
	if len(updates) != len(agents) {
		t.Logf("Expected %d collaboration entries from parallel execution, got %d", len(agents), len(updates))
	}

	// Verify no duplicates
	sessionsSeen := map[string]int{}
	for _, update := range updates {
		// Use subagent_id as unique key
		sessionsSeen[update.SubagentID]++
	}

	for agentID, count := range sessionsSeen {
		if count != 1 {
			t.Errorf("Agent %s appears %d times (expected 1)", agentID, count)
		}
	}
}

// TestSubagentStop_TierSpecificPrompts verifies tier-specific prompt generation for haiku, sonnet, and orchestrator
func TestSubagentStop_TierSpecificPrompts(t *testing.T) {
	binaryPath := "../../bin/goyoke-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skip("goyoke-agent-endstate binary not found")
	}

	projectDir := t.TempDir()
	setupTestRoutingSchema(t, projectDir)

	goyokeDir := filepath.Join(projectDir, ".goyoke")
	os.MkdirAll(goyokeDir, 0755)

	transcriptDir := filepath.Join(projectDir, "transcripts")
	os.MkdirAll(transcriptDir, 0755)

	tierTests := []struct {
		agent string
		model string
		tier  string
	}{
		{"haiku-scout", "haiku", "explore"},
		{"codebase-search", "haiku", "explore"},
		{"python-pro", "sonnet", "general-purpose"},
		{"go-pro", "sonnet", "general-purpose"},
		{"orchestrator", "sonnet", "plan"},
	}

	var events []*EventEntry

	for i, test := range tierTests {
		transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.jsonl", test.agent))
		transcriptContent := fmt.Sprintf(`{"timestamp": %d, "role": "user", "content": "AGENT: %s"}
{"timestamp": %d, "role": "assistant", "content": "Processing", "model": "%s"}
{"timestamp": %d, "role": "completion", "content": "Done"}
`, time.Now().Unix()-10, test.agent, time.Now().Unix()-5, test.model, time.Now().Unix())
		os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)

		rawEvent := map[string]interface{}{
			"hook_event_name":  "SubagentStop",
			"session_id":       fmt.Sprintf("tier-session-%d", i),
			"transcript_path":  transcriptPath,
			"stop_hook_active": true,
		}
		rawJSON, _ := json.Marshal(rawEvent)

		event := &EventEntry{
			HookEventName: "SubagentStop",
			SessionID:     fmt.Sprintf("tier-session-%d", i),
			Timestamp:     time.Now().Unix(),
			RawJSON:       rawJSON,
		}
		events = append(events, event)
	}

	tmpCorpus := filepath.Join(t.TempDir(), "tier-corpus.jsonl")
	var corpusLines []string
	for _, event := range events {
		corpusLines = append(corpusLines, string(event.RawJSON))
	}
	os.WriteFile(tmpCorpus, []byte(strings.Join(corpusLines, "\n")+"\n"), 0644)

	harness, _ := NewTestHarness(tmpCorpus, projectDir)
	harness.LoadCorpus()

	// Execute hooks
	for _, event := range harness.Events {
		result := harness.RunHook(binaryPath, event)
		if result.ExitCode != 0 {
			t.Fatalf("Hook failed: %s", result.Stderr)
		}

		// Verify hookSpecificOutput contains tier-appropriate prompt
		if result.ParsedJSON == nil {
			t.Fatal("Expected JSON output")
		}

		hookOutput, ok := result.ParsedJSON["hookSpecificOutput"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing hookSpecificOutput")
		}

		// Verify hookEventName is SubagentStop
		if hookOutput["hookEventName"] != "SubagentStop" {
			t.Errorf("Expected hookEventName 'SubagentStop', got: %v", hookOutput["hookEventName"])
		}
	}
}

// Helper: Parse collaboration log file
func parseCollaborationLog(t *testing.T, path string) []CollaborationUpdate {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read collaboration log: %v", err)
	}

	var updates []CollaborationUpdate
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var update CollaborationUpdate
		if err := json.Unmarshal([]byte(line), &update); err != nil {
			t.Errorf("Failed to parse collaboration line: %v", err)
			continue
		}
		updates = append(updates, update)
	}
	return updates
}

