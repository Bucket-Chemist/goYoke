package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAgentLifecycleEvent(t *testing.T) {
	event := NewAgentLifecycleEvent(
		"session-123",
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Implement feature X",
		"decision-456",
	)

	if event.EventID == "" {
		t.Error("EventID should be populated with UUID")
	}

	if event.SessionID != "session-123" {
		t.Errorf("Expected SessionID 'session-123', got: %s", event.SessionID)
	}

	if event.EventType != "spawn" {
		t.Errorf("Expected EventType 'spawn', got: %s", event.EventType)
	}

	if event.AgentID != "python-pro" {
		t.Errorf("Expected AgentID 'python-pro', got: %s", event.AgentID)
	}

	if event.ParentAgent != "terminal" {
		t.Errorf("Expected ParentAgent 'terminal', got: %s", event.ParentAgent)
	}

	if event.Tier != "sonnet" {
		t.Errorf("Expected Tier 'sonnet', got: %s", event.Tier)
	}

	if event.TaskDescription != "Implement feature X" {
		t.Errorf("Expected TaskDescription 'Implement feature X', got: %s", event.TaskDescription)
	}

	if event.DecisionID != "decision-456" {
		t.Errorf("Expected DecisionID 'decision-456', got: %s", event.DecisionID)
	}

	if event.Timestamp == 0 {
		t.Error("Timestamp should be populated")
	}

	// Completion fields should be nil for spawn events
	if event.Success != nil {
		t.Error("Success should be nil for spawn events")
	}
	if event.DurationMs != nil {
		t.Error("DurationMs should be nil for spawn events")
	}
	if event.ErrorMessage != nil {
		t.Error("ErrorMessage should be nil for spawn events")
	}
}

func TestNewAgentLifecycleEvent_CompleteEvent(t *testing.T) {
	event := NewAgentLifecycleEvent(
		"session-789",
		"complete",
		"orchestrator",
		"terminal",
		"sonnet",
		"",
		"decision-999",
	)

	// Set completion fields
	success := true
	duration := int64(1500)
	event.Success = &success
	event.DurationMs = &duration

	if event.EventType != "complete" {
		t.Errorf("Expected EventType 'complete', got: %s", event.EventType)
	}

	if event.Success == nil || !*event.Success {
		t.Error("Expected Success to be true")
	}

	if event.DurationMs == nil || *event.DurationMs != 1500 {
		t.Error("Expected DurationMs to be 1500")
	}
}

func TestNewAgentLifecycleEvent_ErrorEvent(t *testing.T) {
	event := NewAgentLifecycleEvent(
		"session-error",
		"error",
		"python-pro",
		"terminal",
		"sonnet",
		"Failed task",
		"decision-error",
	)

	// Set error fields
	success := false
	duration := int64(500)
	errorMsg := "Task failed due to timeout"
	event.Success = &success
	event.DurationMs = &duration
	event.ErrorMessage = &errorMsg

	if event.EventType != "error" {
		t.Errorf("Expected EventType 'error', got: %s", event.EventType)
	}

	if event.Success == nil || *event.Success {
		t.Error("Expected Success to be false")
	}

	if event.ErrorMessage == nil || *event.ErrorMessage != "Task failed due to timeout" {
		t.Error("Expected ErrorMessage to be set")
	}
}

func TestLogAgentLifecycle(t *testing.T) {
	// Setup: Clear existing logs
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	event := NewAgentLifecycleEvent(
		"session-456",
		"spawn",
		"codebase-search",
		"terminal",
		"haiku",
		"Find all Python files",
		"decision-123",
	)

	err := LogAgentLifecycle(event)
	if err != nil {
		t.Fatalf("Failed to log lifecycle event: %v", err)
	}

	// Verify file created
	if _, err := os.Stat(lifecyclePath); os.IsNotExist(err) {
		t.Fatalf("Lifecycle log should exist at %s", lifecyclePath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestReadAgentLifecycleLogs_Empty(t *testing.T) {
	// Setup: Clear existing logs
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	logs, err := ReadAgentLifecycleLogs("")

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs, got: %d", len(logs))
	}
}

func TestReadAgentLifecycleLogs(t *testing.T) {
	// Setup: Clear existing logs
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	// Log multiple events
	for i := 0; i < 3; i++ {
		event := NewAgentLifecycleEvent(
			"session-789",
			"spawn",
			"python-pro",
			"terminal",
			"sonnet",
			"Task description",
			"decision-123",
		)
		LogAgentLifecycle(event)
	}

	logs, err := ReadAgentLifecycleLogs("")

	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got: %d", len(logs))
	}

	// Verify fields populated
	if logs[0].SessionID != "session-789" {
		t.Errorf("Expected SessionID populated in read logs")
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestReadAgentLifecycleLogs_FilterBySession(t *testing.T) {
	// Setup: Clear existing logs
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	// Log events from different sessions
	sessions := []string{"session-1", "session-2", "session-1", "session-3", "session-1"}
	for _, sessionID := range sessions {
		event := NewAgentLifecycleEvent(
			sessionID,
			"spawn",
			"python-pro",
			"terminal",
			"sonnet",
			"Task",
			"decision-123",
		)
		LogAgentLifecycle(event)
	}

	// Filter by session-1
	logs, err := ReadAgentLifecycleLogs("session-1")

	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs for session-1, got: %d", len(logs))
	}

	// Verify all returned logs are for session-1
	for _, log := range logs {
		if log.SessionID != "session-1" {
			t.Errorf("Expected all logs to be for session-1, got: %s", log.SessionID)
		}
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestReadAgentLifecycleLogs_MalformedLines(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	// Write malformed JSONL
	dir := filepath.Dir(lifecyclePath)
	os.MkdirAll(dir, 0755)
	content := `{"event_id":"valid1","timestamp":123,"session_id":"test","event_type":"spawn","agent_id":"agent1","parent_agent":"terminal","tier":"haiku","task_description":"Task 1","decision_id":"dec1"}
not-json-data
{"event_id":"valid2","timestamp":456,"session_id":"test","event_type":"complete","agent_id":"agent2","parent_agent":"terminal","tier":"sonnet","task_description":"Task 2","decision_id":"dec2"}

{"incomplete":
{"event_id":"valid3","timestamp":789,"session_id":"test","event_type":"spawn","agent_id":"agent3","parent_agent":"terminal","tier":"haiku","task_description":"Task 3","decision_id":"dec3"}
`
	os.WriteFile(lifecyclePath, []byte(content), 0644)

	logs, err := ReadAgentLifecycleLogs("")
	if err != nil {
		t.Fatalf("Should not error on malformed lines, got: %v", err)
	}

	// Should skip malformed lines and return only valid ones
	if len(logs) != 3 {
		t.Errorf("Expected 3 valid logs, got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestReadAgentLifecycleLogs_EmptyLines(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	// Log events with extra empty lines
	event1 := NewAgentLifecycleEvent("session-1", "spawn", "agent1", "terminal", "haiku", "Task 1", "dec1")
	LogAgentLifecycle(event1)

	// Append extra empty lines
	f, _ := os.OpenFile(lifecyclePath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("\n\n")
	f.Close()

	event2 := NewAgentLifecycleEvent("session-2", "complete", "agent2", "terminal", "sonnet", "Task 2", "dec2")
	LogAgentLifecycle(event2)

	logs, err := ReadAgentLifecycleLogs("")
	if err != nil {
		t.Fatalf("Should handle empty lines, got: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 logs (empty lines skipped), got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestAgentLifecycle_ThreadSafety(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	// Simulate concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := NewAgentLifecycleEvent(
				"session-concurrent",
				"spawn",
				"python-pro",
				"terminal",
				"sonnet",
				"Task",
				"decision-123",
			)
			LogAgentLifecycle(event)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	logs, err := ReadAgentLifecycleLogs("")
	if err != nil {
		t.Fatalf("Failed to read concurrent logs: %v", err)
	}

	if len(logs) != 10 {
		t.Errorf("Expected 10 logs from concurrent writes, got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestAgentLifecycle_AllFieldsSerialized(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	event := NewAgentLifecycleEvent(
		"session-full",
		"complete",
		"orchestrator",
		"terminal",
		"sonnet",
		"Implement full feature with comprehensive error handling",
		"decision-456",
	)

	success := true
	duration := int64(2500)
	event.Success = &success
	event.DurationMs = &duration

	err := LogAgentLifecycle(event)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	logs, err := ReadAgentLifecycleLogs("")
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one log")
	}

	loaded := logs[0]

	// Verify all fields
	if loaded.EventID == "" {
		t.Error("EventID not preserved")
	}
	if loaded.SessionID != "session-full" {
		t.Errorf("Expected SessionID 'session-full', got: %s", loaded.SessionID)
	}
	if loaded.EventType != "complete" {
		t.Errorf("Expected EventType 'complete', got: %s", loaded.EventType)
	}
	if loaded.AgentID != "orchestrator" {
		t.Errorf("Expected AgentID 'orchestrator', got: %s", loaded.AgentID)
	}
	if loaded.ParentAgent != "terminal" {
		t.Errorf("Expected ParentAgent 'terminal', got: %s", loaded.ParentAgent)
	}
	if loaded.Tier != "sonnet" {
		t.Errorf("Expected Tier 'sonnet', got: %s", loaded.Tier)
	}
	if loaded.TaskDescription != "Implement full feature with comprehensive error handling" {
		t.Errorf("Expected TaskDescription preserved, got: %s", loaded.TaskDescription)
	}
	if loaded.DecisionID != "decision-456" {
		t.Errorf("Expected DecisionID 'decision-456', got: %s", loaded.DecisionID)
	}
	if loaded.Success == nil || !*loaded.Success {
		t.Error("Expected Success true")
	}
	if loaded.DurationMs == nil || *loaded.DurationMs != 2500 {
		t.Errorf("Expected DurationMs 2500, got: %v", loaded.DurationMs)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestAgentLifecycle_TaskDescriptionTruncation(t *testing.T) {
	longDesc := "This is a very long task description that exceeds one hundred characters and should be truncated to avoid bloating the JSONL file size"

	event := NewAgentLifecycleEvent(
		"session-truncate",
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		longDesc,
		"decision-789",
	)

	if len(event.TaskDescription) > 103 { // 100 chars + "..."
		t.Errorf("TaskDescription should be truncated to 100 chars, got: %d", len(event.TaskDescription))
	}

	if !strings.HasSuffix(event.TaskDescription, "...") {
		t.Error("Truncated TaskDescription should end with '...'")
	}
}

func TestXDGComplianceAgentLifecycle(t *testing.T) {
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := "/tmp/test-xdg-lifecycle"
	os.Setenv("XDG_DATA_HOME", testPath)

	// Clear GOGENT_PROJECT_DIR to ensure XDG path is used
	origProject := os.Getenv("GOGENT_PROJECT_DIR")
	os.Unsetenv("GOGENT_PROJECT_DIR")
	defer os.Setenv("GOGENT_PROJECT_DIR", origProject)

	path := getAgentLifecyclePath()
	expected := filepath.Join(testPath, "gogent", "agent-lifecycle.jsonl")

	if path != expected {
		t.Errorf("XDG_DATA_HOME not respected. Got %s, expected %s", path, expected)
	}
}

func TestAgentLifecycle_SpawnAndCompleteFlow(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	sessionID := "session-flow"
	decisionID := "decision-flow-123"

	// 1. Log spawn event
	spawnEvent := NewAgentLifecycleEvent(
		sessionID,
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Implement feature X",
		decisionID,
	)
	if err := LogAgentLifecycle(spawnEvent); err != nil {
		t.Fatalf("Failed to log spawn event: %v", err)
	}

	// 2. Log complete event
	completeEvent := NewAgentLifecycleEvent(
		sessionID,
		"complete",
		"python-pro",
		"terminal",
		"sonnet",
		"",
		decisionID,
	)
	success := true
	duration := int64(1500)
	completeEvent.Success = &success
	completeEvent.DurationMs = &duration

	if err := LogAgentLifecycle(completeEvent); err != nil {
		t.Fatalf("Failed to log complete event: %v", err)
	}

	// 3. Read and verify both events
	logs, err := ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("Expected 2 events (spawn + complete), got: %d", len(logs))
	}

	// Verify spawn event
	if logs[0].EventType != "spawn" {
		t.Errorf("First event should be 'spawn', got: %s", logs[0].EventType)
	}
	if logs[0].TaskDescription == "" {
		t.Error("Spawn event should have task description")
	}
	if logs[0].Success != nil {
		t.Error("Spawn event should not have success field")
	}

	// Verify complete event
	if logs[1].EventType != "complete" {
		t.Errorf("Second event should be 'complete', got: %s", logs[1].EventType)
	}
	if logs[1].Success == nil || !*logs[1].Success {
		t.Error("Complete event should have success=true")
	}
	if logs[1].DurationMs == nil || *logs[1].DurationMs != 1500 {
		t.Error("Complete event should have duration")
	}

	// Verify correlation via DecisionID
	if logs[0].DecisionID != logs[1].DecisionID {
		t.Error("Spawn and complete events should share same DecisionID")
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}

func TestAgentLifecycle_MultipleAgentsParallel(t *testing.T) {
	lifecyclePath := getAgentLifecyclePath()
	os.RemoveAll(filepath.Dir(lifecyclePath))

	sessionID := "session-parallel"

	// Simulate multiple agents spawning in parallel
	agents := []string{"python-pro", "orchestrator", "codebase-search"}
	for _, agent := range agents {
		spawnEvent := NewAgentLifecycleEvent(
			sessionID,
			"spawn",
			agent,
			"terminal",
			"sonnet",
			"Parallel task",
			"decision-"+agent,
		)
		if err := LogAgentLifecycle(spawnEvent); err != nil {
			t.Fatalf("Failed to log spawn for %s: %v", agent, err)
		}
	}

	// Simulate completions
	for _, agent := range agents {
		completeEvent := NewAgentLifecycleEvent(
			sessionID,
			"complete",
			agent,
			"terminal",
			"sonnet",
			"",
			"decision-"+agent,
		)
		success := true
		duration := int64(1000)
		completeEvent.Success = &success
		completeEvent.DurationMs = &duration
		if err := LogAgentLifecycle(completeEvent); err != nil {
			t.Fatalf("Failed to log complete for %s: %v", agent, err)
		}
	}

	logs, err := ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 6 { // 3 spawns + 3 completes
		t.Errorf("Expected 6 events (3 spawns + 3 completes), got: %d", len(logs))
	}

	// Count events by type
	spawnCount := 0
	completeCount := 0
	for _, log := range logs {
		if log.EventType == "spawn" {
			spawnCount++
		}
		if log.EventType == "complete" {
			completeCount++
		}
	}

	if spawnCount != 3 {
		t.Errorf("Expected 3 spawn events, got: %d", spawnCount)
	}
	if completeCount != 3 {
		t.Errorf("Expected 3 complete events, got: %d", completeCount)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(lifecyclePath))
}
