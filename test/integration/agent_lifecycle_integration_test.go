package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// TestAgentLifecycle_SpawnAndCompleteIntegration verifies full lifecycle from spawn to completion
func TestAgentLifecycle_SpawnAndCompleteIntegration(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	sessionID := "test-lifecycle-session"

	// 1. SPAWN EVENT: Simulate gogent-validate emitting a spawn event
	spawnEvent := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Implement GOgent-109 feature",
		"decision-lifecycle-test",
	)

	if err := telemetry.LogAgentLifecycle(spawnEvent); err != nil {
		t.Fatalf("Failed to log spawn event: %v", err)
	}

	// 2. COMPLETE EVENT: Simulate gogent-agent-endstate emitting a complete event
	completeEvent := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"complete",
		"python-pro",
		"terminal",
		"sonnet",
		"",
		"decision-lifecycle-test",
	)

	success := true
	duration := int64(2500)
	completeEvent.Success = &success
	completeEvent.DurationMs = &duration

	if err := telemetry.LogAgentLifecycle(completeEvent); err != nil {
		t.Fatalf("Failed to log complete event: %v", err)
	}

	// 3. VERIFICATION: Read logs and verify both events
	logs, err := telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read lifecycle logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("Expected 2 events (spawn + complete), got: %d", len(logs))
	}

	// Verify spawn event
	spawn := logs[0]
	if spawn.EventType != "spawn" {
		t.Errorf("First event should be 'spawn', got: %s", spawn.EventType)
	}
	if spawn.AgentID != "python-pro" {
		t.Errorf("Expected AgentID 'python-pro', got: %s", spawn.AgentID)
	}
	if spawn.TaskDescription == "" {
		t.Error("Spawn event should have task description")
	}
	if spawn.Success != nil {
		t.Error("Spawn event should not have success field set")
	}

	// Verify complete event
	complete := logs[1]
	if complete.EventType != "complete" {
		t.Errorf("Second event should be 'complete', got: %s", complete.EventType)
	}
	if complete.AgentID != "python-pro" {
		t.Errorf("Expected AgentID 'python-pro', got: %s", complete.AgentID)
	}
	if complete.Success == nil || !*complete.Success {
		t.Error("Complete event should have success=true")
	}
	if complete.DurationMs == nil || *complete.DurationMs != 2500 {
		t.Errorf("Complete event should have duration=2500, got: %v", complete.DurationMs)
	}

	// Verify correlation via DecisionID
	if spawn.DecisionID != complete.DecisionID {
		t.Error("Spawn and complete events should share same DecisionID for correlation")
	}
	if spawn.DecisionID != "decision-lifecycle-test" {
		t.Errorf("Expected DecisionID 'decision-lifecycle-test', got: %s", spawn.DecisionID)
	}
}

// TestAgentLifecycle_MultipleAgentsSession verifies tracking of multiple agents in one session
func TestAgentLifecycle_MultipleAgentsSession(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	sessionID := "test-multi-agent-session"

	agents := []struct {
		agentID    string
		tier       string
		decisionID string
	}{
		{"python-pro", "sonnet", "decision-1"},
		{"orchestrator", "sonnet", "decision-2"},
		{"codebase-search", "haiku", "decision-3"},
	}

	// Spawn all agents
	for _, agent := range agents {
		spawn := telemetry.NewAgentLifecycleEvent(
			sessionID,
			"spawn",
			agent.agentID,
			"terminal",
			agent.tier,
			"Task for "+agent.agentID,
			agent.decisionID,
		)
		if err := telemetry.LogAgentLifecycle(spawn); err != nil {
			t.Fatalf("Failed to log spawn for %s: %v", agent.agentID, err)
		}
	}

	// Complete all agents
	for i, agent := range agents {
		complete := telemetry.NewAgentLifecycleEvent(
			sessionID,
			"complete",
			agent.agentID,
			"terminal",
			agent.tier,
			"",
			agent.decisionID,
		)
		success := i != 1 // Make orchestrator fail for variety
		duration := int64(1000 + i*500)
		complete.Success = &success
		complete.DurationMs = &duration

		if err := telemetry.LogAgentLifecycle(complete); err != nil {
			t.Fatalf("Failed to log complete for %s: %v", agent.agentID, err)
		}
	}

	// Verify all events logged
	logs, err := telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 6 { // 3 spawns + 3 completes
		t.Fatalf("Expected 6 events (3 spawns + 3 completes), got: %d", len(logs))
	}

	// Group events by agent
	agentEvents := make(map[string][]telemetry.AgentLifecycleEvent)
	for _, log := range logs {
		agentEvents[log.AgentID] = append(agentEvents[log.AgentID], log)
	}

	// Verify each agent has spawn + complete
	for _, agent := range agents {
		events, ok := agentEvents[agent.agentID]
		if !ok {
			t.Errorf("No events found for agent %s", agent.agentID)
			continue
		}
		if len(events) != 2 {
			t.Errorf("Expected 2 events for %s, got: %d", agent.agentID, len(events))
			continue
		}
		if events[0].EventType != "spawn" || events[1].EventType != "complete" {
			t.Errorf("Expected spawn then complete for %s, got: %s, %s",
				agent.agentID, events[0].EventType, events[1].EventType)
		}
	}
}

// TestAgentLifecycle_ErrorEvent verifies error event logging
func TestAgentLifecycle_ErrorEvent(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	sessionID := "test-error-session"

	// Spawn event
	spawn := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Task that will fail",
		"decision-error",
	)
	if err := telemetry.LogAgentLifecycle(spawn); err != nil {
		t.Fatalf("Failed to log spawn: %v", err)
	}

	// Error event
	errorEvent := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"error",
		"python-pro",
		"terminal",
		"sonnet",
		"",
		"decision-error",
	)
	success := false
	duration := int64(500)
	errorMsg := "Task failed due to timeout"
	errorEvent.Success = &success
	errorEvent.DurationMs = &duration
	errorEvent.ErrorMessage = &errorMsg

	if err := telemetry.LogAgentLifecycle(errorEvent); err != nil {
		t.Fatalf("Failed to log error event: %v", err)
	}

	// Verify events
	logs, err := telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("Expected 2 events (spawn + error), got: %d", len(logs))
	}

	// Verify error event
	errorLog := logs[1]
	if errorLog.EventType != "error" {
		t.Errorf("Expected EventType 'error', got: %s", errorLog.EventType)
	}
	if errorLog.Success == nil || *errorLog.Success {
		t.Error("Error event should have success=false")
	}
	if errorLog.ErrorMessage == nil || *errorLog.ErrorMessage != "Task failed due to timeout" {
		t.Error("Error event should have error message")
	}
}

// TestAgentLifecycle_RealTimeTracking simulates TUI polling for real-time updates
func TestAgentLifecycle_RealTimeTracking(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	sessionID := "test-realtime-session"

	// Initial read - no events yet
	logs, err := telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("Expected 0 events initially, got: %d", len(logs))
	}

	// Agent spawns
	spawn := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Long-running task",
		"decision-realtime",
	)
	if err := telemetry.LogAgentLifecycle(spawn); err != nil {
		t.Fatalf("Failed to log spawn: %v", err)
	}

	// TUI polls - should see spawn event
	logs, err = telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs after spawn: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 event after spawn, got: %d", len(logs))
	}
	if len(logs) > 0 && logs[0].EventType != "spawn" {
		t.Errorf("Expected spawn event, got: %s", logs[0].EventType)
	}

	// Simulate agent working (TUI can show "in progress")
	time.Sleep(10 * time.Millisecond)

	// Agent completes
	complete := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"complete",
		"python-pro",
		"terminal",
		"sonnet",
		"",
		"decision-realtime",
	)
	success := true
	duration := int64(100)
	complete.Success = &success
	complete.DurationMs = &duration

	if err := telemetry.LogAgentLifecycle(complete); err != nil {
		t.Fatalf("Failed to log complete: %v", err)
	}

	// TUI polls again - should see both events
	logs, err = telemetry.ReadAgentLifecycleLogs(sessionID)
	if err != nil {
		t.Fatalf("Failed to read logs after complete: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 events after complete, got: %d", len(logs))
	}

	// Verify TUI can determine agent status
	active := false
	for _, log := range logs {
		if log.EventType == "spawn" && (log.Success == nil) {
			// Spawn without matching complete = agent is active
			hasComplete := false
			for _, log2 := range logs {
				if log2.EventType == "complete" && log2.AgentID == log.AgentID {
					hasComplete = true
					break
				}
			}
			if !hasComplete {
				active = true
			}
		}
	}
	if active {
		t.Error("Agent should not be active after complete event")
	}
}

// TestAgentLifecycle_FileLocation verifies events are written to correct location
func TestAgentLifecycle_FileLocation(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	sessionID := "test-file-location"

	// Log an event
	event := telemetry.NewAgentLifecycleEvent(
		sessionID,
		"spawn",
		"python-pro",
		"terminal",
		"sonnet",
		"Test task",
		"decision-123",
	)
	if err := telemetry.LogAgentLifecycle(event); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify file location
	expectedPath := filepath.Join(testDir, ".gogent", "agent-lifecycle.jsonl")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it doesn't exist", expectedPath)
	}

	// Verify file content is valid JSONL
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read lifecycle file: %v", err)
	}
	if len(content) == 0 {
		t.Error("Lifecycle file should not be empty")
	}
}

// TestAgentLifecycle_SessionIsolation verifies session filtering works correctly
func TestAgentLifecycle_SessionIsolation(t *testing.T) {
	// Setup: Use isolated test directory
	testDir := t.TempDir()
	os.Setenv("GOGENT_PROJECT_DIR", testDir)
	defer os.Unsetenv("GOGENT_PROJECT_DIR")

	// Log events from multiple sessions
	sessions := []string{"session-A", "session-B", "session-C"}
	for _, sessionID := range sessions {
		for i := 0; i < 2; i++ { // 2 events per session
			event := telemetry.NewAgentLifecycleEvent(
				sessionID,
				"spawn",
				"python-pro",
				"terminal",
				"sonnet",
				"Task",
				"decision-"+sessionID,
			)
			if err := telemetry.LogAgentLifecycle(event); err != nil {
				t.Fatalf("Failed to log event for %s: %v", sessionID, err)
			}
		}
	}

	// Verify each session only sees its own events
	for _, sessionID := range sessions {
		logs, err := telemetry.ReadAgentLifecycleLogs(sessionID)
		if err != nil {
			t.Fatalf("Failed to read logs for %s: %v", sessionID, err)
		}
		if len(logs) != 2 {
			t.Errorf("Session %s should have 2 events, got: %d", sessionID, len(logs))
		}
		for _, log := range logs {
			if log.SessionID != sessionID {
				t.Errorf("Session %s received event from %s", sessionID, log.SessionID)
			}
		}
	}

	// Verify reading all sessions returns all events
	allLogs, err := telemetry.ReadAgentLifecycleLogs("")
	if err != nil {
		t.Fatalf("Failed to read all logs: %v", err)
	}
	if len(allLogs) != 6 { // 3 sessions * 2 events
		t.Errorf("Expected 6 total events, got: %d", len(allLogs))
	}
}
