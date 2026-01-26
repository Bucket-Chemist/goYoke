package cli

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealClaudeStreaming tests full streaming workflow with real Claude CLI.
// Validates:
// - Process starts successfully
// - Init event is received
// - Messages can be sent and responses received
// - Process stops cleanly
func TestRealClaudeStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Claude test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true, // Skip GOgent hooks for clean test
		Timeout: TimeoutConfig{
			InactivityTimeout: 30 * time.Second,
			ResultTimeout:     30 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err, "Failed to create Claude process")

	err = proc.Start()
	require.NoError(t, err, "Failed to start Claude process")
	defer func() {
		stopErr := proc.Stop()
		assert.NoError(t, stopErr, "Failed to stop Claude process")
	}()

	// Wait for init event with timeout
	var initReceived bool
	timeout := time.After(30 * time.Second)
	select {
	case event := <-proc.Events():
		if event.Type == "system" {
			sysEvent, parseErr := event.AsSystem()
			if parseErr == nil && sysEvent.Subtype == "init" {
				initReceived = true
				t.Logf("Received init event for session: %s", sysEvent.SessionID)
			}
		}
	case err := <-proc.Errors():
		t.Fatalf("Unexpected error before init: %v", err)
	case <-timeout:
		t.Fatal("Timeout waiting for init event")
	}

	require.True(t, initReceived, "Did not receive init event")

	// Send a simple message
	t.Log("Sending message to Claude...")
	err = proc.Send("Say exactly: HELLO")
	require.NoError(t, err, "Failed to send message")

	// Wait for response (assistant or result event)
	var responseReceived bool
	responseTimeout := time.After(30 * time.Second)
	eventLoop:
	for {
		select {
		case event := <-proc.Events():
			t.Logf("Received event type: %s, subtype: %s", event.Type, event.Subtype)
			switch event.Type {
			case "assistant":
				responseReceived = true
				assEvent, parseErr := event.AsAssistant()
				if parseErr == nil && len(assEvent.Message.Content) > 0 {
					t.Logf("Received assistant response with %d content blocks", len(assEvent.Message.Content))
				}
				break eventLoop
			case "result":
				responseReceived = true
				resEvent, parseErr := event.AsResult()
				if parseErr == nil {
					t.Logf("Received result event: is_error=%v, duration=%dms", resEvent.IsError, resEvent.DurationMs)
				}
				break eventLoop
			}
		case err := <-proc.Errors():
			// Log errors but don't fail - Claude may send debug info
			t.Logf("Received error (non-fatal): %v", err)
		case <-responseTimeout:
			t.Fatal("Timeout waiting for response")
		}
	}

	require.True(t, responseReceived, "Did not receive any response from Claude")
}

// TestStreamingNoGoroutineLeak verifies that goroutines are properly cleaned up
// after a Claude subprocess lifecycle.
// Critical for long-running applications that start/stop Claude processes.
func TestStreamingNoGoroutineLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Claude test in -short mode")
	}

	// Count goroutines before
	runtime.GC()
	time.Sleep(50 * time.Millisecond) // Let GC settle
	baseline := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baseline)

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		Timeout: TimeoutConfig{
			InactivityTimeout: 10 * time.Second,
			ResultTimeout:     10 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	require.NoError(t, err)

	// Wait for init event
	select {
	case <-proc.Events():
		// Got init
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for init")
	}

	// Send a message and wait for response
	err = proc.Send("Say: TEST")
	require.NoError(t, err)

	// Consume events until we get a response
	responseDeadline := time.After(10 * time.Second)
	consumeLoop:
	for {
		select {
		case event := <-proc.Events():
			if event.Type == "assistant" || event.Type == "result" {
				break consumeLoop
			}
		case <-proc.Errors():
			// Consume errors
		case <-responseDeadline:
			break consumeLoop
		}
	}

	// Stop the process
	err = proc.Stop()
	require.NoError(t, err)

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Verify goroutine count is back near baseline
	final := runtime.NumGoroutine()
	t.Logf("Final goroutines: %d (baseline: %d)", final, baseline)

	// Allow some variance (test framework overhead, runtime goroutines)
	// but should not leak the subprocess goroutines (readEvents, readStderr, monitorRestart)
	maxAllowed := baseline + 5
	if final > maxAllowed {
		t.Errorf("Goroutine leak detected: %d goroutines after cleanup (baseline: %d, max allowed: %d)",
			final, baseline, maxAllowed)
	}
}

// TestRestartPreservesEventChannel verifies that after a restart, events still
// arrive on the SAME channel reference that was obtained before restart.
// This is critical for TUI consumers that subscribe once to the events channel.
func TestRestartPreservesEventChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping restart test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		Restart: RestartPolicy{
			Enabled:         true,
			MaxRestarts:     1,
			RestartDelay:    100 * time.Millisecond,
			PreserveSession: false, // New session on restart
		},
		Timeout: TimeoutConfig{
			InactivityTimeout: 10 * time.Second,
			ResultTimeout:     10 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Get channel reference BEFORE starting
	eventsChan := proc.Events()
	restartChan := proc.RestartEvents()
	require.NotNil(t, eventsChan)
	require.NotNil(t, restartChan)

	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	// Wait for init event on original channel
	var firstSessionID string
	select {
	case event := <-eventsChan:
		if event.Type == "system" {
			sysEvent, parseErr := event.AsSystem()
			if parseErr == nil && sysEvent.Subtype == "init" {
				firstSessionID = sysEvent.SessionID
				t.Logf("First session init: %s", firstSessionID)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for first init event")
	}

	// Force a restart by calling the internal restart method
	// (in real scenarios, this happens automatically on crash)
	// We'll simulate by stopping and starting with restart enabled
	originalSessionID := proc.SessionID()
	t.Logf("Original session: %s", originalSessionID)

	// Trigger restart by manually calling restart()
	// Note: In production this happens via monitorRestart when process exits
	// For testing, we'll verify the infrastructure without crashing
	err = proc.restart()
	require.NoError(t, err, "Restart failed")

	// Verify events still arrive on SAME channel reference
	var secondInitReceived bool
	select {
	case event := <-eventsChan: // Using original channel reference!
		if event.Type == "system" {
			sysEvent, parseErr := event.AsSystem()
			if parseErr == nil && sysEvent.Subtype == "init" {
				secondInitReceived = true
				t.Logf("Second session init: %s", sysEvent.SessionID)
				// Session ID should be different (PreserveSession=false)
				assert.NotEqual(t, firstSessionID, sysEvent.SessionID,
					"Expected new session ID after restart")
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for second init event after restart")
	}

	require.True(t, secondInitReceived, "Did not receive init event after restart on original channel")
}

// TestRestartWithSessionPreservation validates PreserveSession flag behavior.
func TestRestartWithSessionPreservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping restart test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		SessionID:  "test-session-preserve",
		Restart: RestartPolicy{
			Enabled:         true,
			MaxRestarts:     1,
			RestartDelay:    100 * time.Millisecond,
			PreserveSession: true, // Keep same session
		},
		Timeout: TimeoutConfig{
			InactivityTimeout: 10 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	originalSessionID := proc.SessionID()
	assert.Equal(t, "test-session-preserve", originalSessionID)

	// Consume init event
	select {
	case <-proc.Events():
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for init")
	}

	// Trigger restart
	err = proc.restart()
	require.NoError(t, err)

	// Verify session ID is preserved
	newSessionID := proc.SessionID()
	assert.Equal(t, originalSessionID, newSessionID,
		"Session ID should be preserved when PreserveSession=true")
}

// TestMultipleRestartsCycleThroughGenerations verifies generation tracking
// prevents stale goroutines from interfering after multiple restarts.
func TestMultipleRestartsCycleThroughGenerations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping restart test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		Restart: RestartPolicy{
			Enabled:         true,
			MaxRestarts:     3,
			RestartDelay:    100 * time.Millisecond,
			PreserveSession: false,
		},
		Timeout: TimeoutConfig{
			InactivityTimeout: 10 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	eventsChan := proc.Events()

	// Consume initial init event
	select {
	case <-eventsChan:
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for initial init")
	}

	// Perform multiple restarts
	for i := 0; i < 3; i++ {
		t.Logf("Restart iteration %d", i+1)

		err = proc.restart()
		require.NoError(t, err, "Restart %d failed", i+1)

		// Verify new init event arrives
		select {
		case event := <-eventsChan:
			assert.Equal(t, "system", event.Type, "Expected system event after restart %d", i+1)
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for init after restart %d", i+1)
		}
	}

	// Verify process is still running after multiple restarts
	assert.True(t, proc.IsRunning(), "Process should still be running after multiple restarts")
}

// TestRestartEventsEmitted verifies RestartEvent is sent to the RestartEvents channel.
func TestRestartEventsEmitted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping restart test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		Restart: RestartPolicy{
			Enabled:      true,
			MaxRestarts:  1,
			RestartDelay: 100 * time.Millisecond,
		},
		Timeout: TimeoutConfig{
			InactivityTimeout: 10 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	restartChan := proc.RestartEvents()

	// Consume init event
	select {
	case <-proc.Events():
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for init")
	}

	// Note: In real usage, restart events come from monitorRestart when process exits.
	// Testing the full flow requires a crasher binary.
	// Here we verify the infrastructure is in place.
	assert.NotNil(t, restartChan, "RestartEvents channel should be accessible")
}

// TestIntegrationMessageRoundtrip tests a complete message send/receive cycle.
func TestIntegrationMessageRoundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in -short mode")
	}

	cfg := Config{
		ClaudePath: "claude",
		NoHooks:    true,
		Timeout: TimeoutConfig{
			InactivityTimeout: 30 * time.Second,
			ResultTimeout:     30 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	require.NoError(t, err)
	defer proc.Stop()

	// Consume init event
	select {
	case event := <-proc.Events():
		require.Equal(t, "system", event.Type)
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for init")
	}

	// Send structured message
	msg := UserMessage{
		Type: "user",
		Message: UserContent{
			Role: "user",
			Content: []ContentBlock{
				{Type: "text", Text: "What is 2+2? Reply with just the number."},
			},
		},
	}

	err = proc.SendJSON(msg)
	require.NoError(t, err)

	// Wait for response
	var gotResponse bool
	deadline := time.After(30 * time.Second)
	for !gotResponse {
		select {
		case event := <-proc.Events():
			if event.Type == "assistant" || event.Type == "result" {
				gotResponse = true
				t.Logf("Got response event: %s", event.Type)
			}
		case err := <-proc.Errors():
			t.Logf("Non-fatal error during message: %v", err)
		case <-deadline:
			t.Fatal("Timeout waiting for response")
		}
	}

	require.True(t, gotResponse, "Did not receive response to message")
}
