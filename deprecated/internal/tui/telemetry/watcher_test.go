package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	pkgtel "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONLWatcher_NewFile tests watching a file that doesn't exist yet
func TestJSONLWatcher_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create watcher for non-existent file
	watcher, err := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		return string(data), nil
	})
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// Write to file (should be detected)
	time.Sleep(100 * time.Millisecond) // Give watcher time to set up
	err = os.WriteFile(testFile, []byte("line1\n"), 0644)
	require.NoError(t, err)

	// Verify event received
	select {
	case event := <-watcher.Events():
		assert.Equal(t, "line1", event)
	case err := <-watcher.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestJSONLWatcher_ExistingFile tests watching an existing file
func TestJSONLWatcher_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create file with initial content
	err := os.WriteFile(testFile, []byte("existing\n"), 0644)
	require.NoError(t, err)

	// Create watcher (should seek to end, ignoring existing content)
	watcher, err := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		return string(data), nil
	})
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// Append new line
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("new\n")
	require.NoError(t, err)
	f.Close()

	// Should only receive new line, not existing
	select {
	case event := <-watcher.Events():
		assert.Equal(t, "new", event)
	case err := <-watcher.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestJSONLWatcher_MultipleLines tests multiple rapid writes
func TestJSONLWatcher_MultipleLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	watcher, err := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		return string(data), nil
	})
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	time.Sleep(100 * time.Millisecond)

	// Write multiple lines
	f, err := os.Create(testFile)
	require.NoError(t, err)
	_, err = f.WriteString("line1\nline2\nline3\n")
	require.NoError(t, err)
	f.Close()

	// Collect all events
	var events []string
	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case event := <-watcher.Events():
			events = append(events, event.(string))
		case <-timeout:
			t.Fatalf("timeout waiting for events, got %d/3", len(events))
		}
	}

	assert.ElementsMatch(t, []string{"line1", "line2", "line3"}, events)
}

// TestJSONLWatcher_MalformedJSON tests handling of parse errors
func TestJSONLWatcher_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	parseCount := 0
	watcher, err := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		parseCount++
		var result map[string]string
		err := json.Unmarshal(data, &result)
		return result, err
	})
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	time.Sleep(100 * time.Millisecond)

	// Write valid and invalid JSON
	err = os.WriteFile(testFile, []byte(`{"valid": "json"}`+"\n"+`invalid json`+"\n"+`{"more": "valid"}`+"\n"), 0644)
	require.NoError(t, err)

	// Should receive 2 valid events (invalid line is silently skipped)
	var validEvents int

	timeout := time.After(2 * time.Second)
	for validEvents < 2 {
		select {
		case event := <-watcher.Events():
			if event != nil {
				validEvents++
			}
		case err := <-watcher.Errors():
			// Errors may or may not be emitted for parse failures
			// (implementation is resilient and continues on error)
			t.Logf("received error: %v", err)
		case <-timeout:
			t.Fatalf("timeout: got %d valid events", validEvents)
		}
	}

	assert.Equal(t, 2, validEvents, "should receive 2 valid events")
	// Verify all 3 lines were attempted to parse
	assert.Equal(t, 3, parseCount, "should have attempted to parse all 3 lines")
}

// TestJSONLWatcher_FileTruncation tests handling of file truncation
func TestJSONLWatcher_FileTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create file with content
	err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)
	require.NoError(t, err)

	watcher, err := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		return string(data), nil
	})
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// Truncate file (simulating rotation)
	time.Sleep(100 * time.Millisecond)
	err = os.Truncate(testFile, 0)
	require.NoError(t, err)

	// Write new content
	err = os.WriteFile(testFile, []byte("after truncate\n"), 0644)
	require.NoError(t, err)

	// Should receive new content
	select {
	case event := <-watcher.Events():
		assert.Equal(t, "after truncate", event)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event after truncation")
	}
}

// TestTelemetryWatcher_Integration tests full watcher with real events
func TestTelemetryWatcher_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOGENT_PROJECT_DIR for path resolution
	originalProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", originalProjectDir)

	// Create telemetry directory
	telemetryDir := filepath.Join(tmpDir, ".gogent")
	err := os.MkdirAll(telemetryDir, 0755)
	require.NoError(t, err)

	// Create watcher
	watcher, err := NewTelemetryWatcher()
	require.NoError(t, err)

	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	time.Sleep(100 * time.Millisecond)

	// Write agent lifecycle event
	lifecyclePath := filepath.Join(telemetryDir, "agent-lifecycle.jsonl")
	lifecycleEvent := &pkgtel.AgentLifecycleEvent{
		EventID:         "test-event-1",
		SessionID:       "test-session",
		Timestamp:       time.Now().Unix(),
		EventType:       "spawn",
		AgentID:         "test-agent",
		ParentAgent:     "terminal",
		Tier:            "sonnet",
		TaskDescription: "test task",
		DecisionID:      "test-decision",
	}
	eventData, err := json.Marshal(lifecycleEvent)
	require.NoError(t, err)
	err = os.WriteFile(lifecyclePath, append(eventData, '\n'), 0644)
	require.NoError(t, err)

	// Verify event received
	events := watcher.Events()
	select {
	case received := <-events.AgentLifecycle:
		assert.Equal(t, "test-event-1", received.EventID)
		assert.Equal(t, "test-agent", received.AgentID)
		assert.Equal(t, "spawn", received.EventType)
	case err := <-events.Errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for lifecycle event")
	}

	// Write routing decision
	decisionsPath := filepath.Join(telemetryDir, "routing-decisions.jsonl")
	decision := &pkgtel.RoutingDecision{
		DecisionID:      "test-decision",
		Timestamp:       time.Now().Unix(),
		SessionID:       "test-session",
		TaskDescription: "test task",
		SelectedTier:    "sonnet",
		SelectedAgent:   "test-agent",
	}
	decisionData, err := json.Marshal(decision)
	require.NoError(t, err)
	err = os.WriteFile(decisionsPath, append(decisionData, '\n'), 0644)
	require.NoError(t, err)

	// Verify decision received
	select {
	case received := <-events.RoutingDecisions:
		assert.Equal(t, "test-decision", received.DecisionID)
		assert.Equal(t, "sonnet", received.SelectedTier)
	case err := <-events.Errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for routing decision")
	}
}

// TestTelemetryAggregator_AgentLifecycle tests aggregator state management
func TestTelemetryAggregator_AgentLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOGENT_PROJECT_DIR
	originalProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", originalProjectDir)

	telemetryDir := filepath.Join(tmpDir, ".gogent")
	err := os.MkdirAll(telemetryDir, 0755)
	require.NoError(t, err)

	// Create aggregator
	aggregator, err := NewTelemetryAggregator()
	require.NoError(t, err)

	err = aggregator.Start()
	require.NoError(t, err)
	defer aggregator.Stop()

	time.Sleep(100 * time.Millisecond)

	// Write spawn event
	lifecyclePath := filepath.Join(telemetryDir, "agent-lifecycle.jsonl")
	spawnEvent := &pkgtel.AgentLifecycleEvent{
		EventID:         "spawn-1",
		SessionID:       "test-session",
		Timestamp:       time.Now().Unix(),
		EventType:       "spawn",
		AgentID:         "agent-1",
		ParentAgent:     "terminal",
		Tier:            "sonnet",
		TaskDescription: "test task",
		DecisionID:      "decision-1",
	}
	data, _ := json.Marshal(spawnEvent)
	f, err := os.Create(lifecyclePath)
	require.NoError(t, err)
	_, err = f.Write(append(data, '\n'))
	require.NoError(t, err)
	f.Close()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check active agents
	activeAgents := aggregator.GetActiveAgents()
	assert.Len(t, activeAgents, 1)
	assert.Equal(t, "agent-1", activeAgents[0].AgentID)
	assert.Equal(t, "running", activeAgents[0].Status)

	// Write complete event
	success := true
	completeEvent := &pkgtel.AgentLifecycleEvent{
		EventID:         "complete-1",
		SessionID:       "test-session",
		Timestamp:       time.Now().Unix(),
		EventType:       "complete",
		AgentID:         "agent-1",
		ParentAgent:     "terminal",
		Tier:            "sonnet",
		TaskDescription: "test task",
		DecisionID:      "decision-1",
		Success:         &success,
	}
	data, _ = json.Marshal(completeEvent)
	f, err = os.OpenFile(lifecyclePath, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.Write(append(data, '\n'))
	require.NoError(t, err)
	f.Close()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check completed agents
	completedAgents := aggregator.GetCompletedAgents()
	assert.Len(t, completedAgents, 1)
	assert.Equal(t, "agent-1", completedAgents[0].AgentID)
	assert.Equal(t, "completed", completedAgents[0].Status)

	// Check stats
	stats := aggregator.Stats()
	assert.Equal(t, 1, stats.TotalAgents)
	assert.Equal(t, 0, stats.ActiveAgents)
	assert.Equal(t, 1, stats.CompletedAgents)
	assert.Equal(t, 1.0, stats.SuccessRate)
}

// TestTelemetryAggregator_MultipleAgents tests concurrent agent tracking
func TestTelemetryAggregator_MultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()

	originalProjectDir := os.Getenv("GOGENT_PROJECT_DIR")
	os.Setenv("GOGENT_PROJECT_DIR", tmpDir)
	defer os.Setenv("GOGENT_PROJECT_DIR", originalProjectDir)

	telemetryDir := filepath.Join(tmpDir, ".gogent")
	err := os.MkdirAll(telemetryDir, 0755)
	require.NoError(t, err)

	aggregator, err := NewTelemetryAggregator()
	require.NoError(t, err)

	err = aggregator.Start()
	require.NoError(t, err)
	defer aggregator.Stop()

	time.Sleep(100 * time.Millisecond)

	lifecyclePath := filepath.Join(telemetryDir, "agent-lifecycle.jsonl")
	f, err := os.Create(lifecyclePath)
	require.NoError(t, err)

	// Spawn 3 agents
	now := time.Now().Unix()
	for i := 1; i <= 3; i++ {
		event := &pkgtel.AgentLifecycleEvent{
			EventID:         "spawn-" + string(rune('0'+i)),
			SessionID:       "test-session",
			Timestamp:       now,
			EventType:       "spawn",
			AgentID:         "agent-" + string(rune('0'+i)),
			ParentAgent:     "terminal",
			Tier:            "sonnet",
			TaskDescription: "test task",
		}
		data, _ := json.Marshal(event)
		f.Write(append(data, '\n'))
	}
	f.Close()

	time.Sleep(200 * time.Millisecond)

	// All should be active
	activeAgents := aggregator.GetActiveAgents()
	assert.Len(t, activeAgents, 3)

	// Complete 2 agents
	f, err = os.OpenFile(lifecyclePath, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	for i := 1; i <= 2; i++ {
		success := i == 1 // First succeeds, second fails
		event := &pkgtel.AgentLifecycleEvent{
			EventID:   "complete-" + string(rune('0'+i)),
			SessionID: "test-session",
			Timestamp: now + 10,
			EventType: "complete",
			AgentID:   "agent-" + string(rune('0'+i)),
			Success:   &success,
		}
		data, _ := json.Marshal(event)
		f.Write(append(data, '\n'))
	}
	f.Close()

	time.Sleep(200 * time.Millisecond)

	// Check final state
	activeAgents = aggregator.GetActiveAgents()
	completedAgents := aggregator.GetCompletedAgents()
	assert.Len(t, activeAgents, 1)
	assert.Len(t, completedAgents, 2)

	stats := aggregator.Stats()
	assert.Equal(t, 3, stats.TotalAgents)
	assert.Equal(t, 1, stats.ActiveAgents)
	assert.Equal(t, 2, stats.CompletedAgents)
	assert.Equal(t, 0.5, stats.SuccessRate) // 1 success / 2 completed
}

// Benchmark tests
func BenchmarkJSONLWatcher_SingleLine(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	watcher, _ := NewJSONLWatcher(testFile, func(data []byte) (interface{}, error) {
		return string(data), nil
	})
	watcher.Start()
	defer watcher.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.WriteFile(testFile, []byte("test line\n"), 0644)
		<-watcher.Events()
	}
}
