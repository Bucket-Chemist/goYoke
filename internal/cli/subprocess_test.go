package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClaudeProcess_Defaults(t *testing.T) {
	cfg := Config{}
	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Should generate session ID
	assert.NotEmpty(t, proc.SessionID())

	// Should not be running yet
	assert.False(t, proc.IsRunning())

	// Channels should be accessible
	assert.NotNil(t, proc.Events())
	assert.NotNil(t, proc.Errors())
}

func TestNewClaudeProcess_CustomConfig(t *testing.T) {
	cfg := Config{
		ClaudePath:     "/custom/claude",
		SessionID:      "test-session-123",
		Verbose:        true,
		IncludePartial: true,
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	assert.Equal(t, "test-session-123", proc.SessionID())

	// Verify command args
	args := proc.cmd.Args
	assert.Contains(t, args, "--verbose")
	assert.Contains(t, args, "--include-partial-messages")
	assert.Contains(t, args, "--session-id")
	assert.Contains(t, args, "test-session-123")
	assert.Contains(t, args, "--input-format")
	assert.Contains(t, args, "stream-json")
	assert.Contains(t, args, "--output-format")
	assert.Contains(t, args, "stream-json")
}

func TestClaudeProcess_StartStop(t *testing.T) {
	// This test requires a mock claude binary
	// Skip if mock is not available
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Start process
	err = proc.Start()
	if err != nil {
		// Mock binary not available, skip test
		t.Skipf("Mock claude binary not available: %v", err)
	}

	// Should be running
	assert.True(t, proc.IsRunning())

	// Stop process
	err = proc.Stop()
	assert.NoError(t, err)

	// Should not be running
	assert.False(t, proc.IsRunning())
}

func TestClaudeProcess_DoubleStart(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Second start should fail
	err = proc.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestClaudeProcess_StopIdempotent(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}

	// First stop
	err = proc.Stop()
	assert.NoError(t, err)

	// Second stop should be safe
	err = proc.Stop()
	assert.NoError(t, err)
}

func TestClaudeProcess_SendBeforeStart(t *testing.T) {
	cfg := Config{}
	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Send should fail if not started
	err = proc.Send("test message")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestClaudeProcess_SendJSON(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Send a message
	msg := UserMessage{Content: "Hello"}
	err = proc.SendJSON(msg)
	assert.NoError(t, err)
}

func TestClaudeProcess_EventReading(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Wait for init event from mock
	select {
	case event := <-proc.Events():
		// Mock should emit init event
		assert.NotEmpty(t, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for init event")
	}
}

func TestClaudeProcess_EchoMessage(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Consume init event
	select {
	case <-proc.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for init event")
	}

	// Send message
	err = proc.Send("test message")
	require.NoError(t, err)

	// Wait for echo response
	select {
	case event := <-proc.Events():
		assert.NotEmpty(t, event.Type)
		// Mock echoes as assistant event
		assert.Equal(t, "assistant", event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for echo response")
	}
}

func TestClaudeProcess_GracefulShutdown(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}

	// Stop should complete within timeout
	done := make(chan error, 1)
	go func() {
		done <- proc.Stop()
	}()

	select {
	case err := <-done:
		// Should shutdown gracefully
		assert.NoError(t, err)
	case <-time.After(6 * time.Second):
		t.Fatal("Shutdown timeout - took longer than 6 seconds")
	}

	assert.False(t, proc.IsRunning())
}

func TestClaudeProcess_ChannelsClosed(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}

	eventsChan := proc.Events()
	errorsChan := proc.Errors()

	err = proc.Stop()
	require.NoError(t, err)

	// Channels should eventually close
	time.Sleep(100 * time.Millisecond)

	// Reading from closed channels should not block
	select {
	case _, ok := <-eventsChan:
		if ok {
			// May still have buffered events
		}
	default:
		// Channel closed or empty
	}

	select {
	case _, ok := <-errorsChan:
		if ok {
			// May still have buffered errors
		}
	default:
		// Channel closed or empty
	}
}

// parseEvent tests are now in events_test.go (GOgent-114).
// This file tests subprocess integration with events.

func TestClaudeProcess_RestartDisabled(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:      false,
			MaxRestarts:  1, // Set non-zero to prevent defaults
			RestartDelay: 1 * time.Second,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Verify restart is disabled
	assert.False(t, proc.config.Restart.Enabled)
}

func TestClaudeProcess_RestartDefaultPolicy(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Should have default restart policy
	assert.True(t, proc.config.Restart.Enabled)
	assert.Equal(t, 3, proc.config.Restart.MaxRestarts)
	assert.Equal(t, 1*time.Second, proc.config.Restart.RestartDelay)
	assert.Equal(t, 30*time.Second, proc.config.Restart.MaxDelay)
	assert.Equal(t, 2.0, proc.config.Restart.BackoffFactor)
	assert.Equal(t, 60*time.Second, proc.config.Restart.ResetAfter)
	assert.True(t, proc.config.Restart.PreserveSession)
}

func TestClaudeProcess_RestartEventsChannel(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// RestartEvents channel should be accessible
	restartChan := proc.RestartEvents()
	assert.NotNil(t, restartChan)
}

func TestClaudeProcess_ExplicitStopPreventsRestart(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:      true,
			MaxRestarts:  3,
			RestartDelay: 100 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}

	// Explicitly stop
	err = proc.Stop()
	assert.NoError(t, err)

	// Wait a bit to ensure no restart happens
	time.Sleep(300 * time.Millisecond)

	// Should still be stopped
	assert.False(t, proc.IsRunning())

	// Check for explicit stop event
	select {
	case event := <-proc.RestartEvents():
		assert.Equal(t, "explicit_stop", event.Reason)
		assert.False(t, event.WillResume)
	case <-time.After(100 * time.Millisecond):
		// No event is also acceptable
	}
}

func TestClaudeProcess_SessionIDPreserved(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		SessionID:  "test-session-preserve",
		Restart: RestartPolicy{
			Enabled:         true,
			PreserveSession: true,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	originalSession := proc.SessionID()
	assert.Equal(t, "test-session-preserve", originalSession)
	assert.True(t, proc.config.Restart.PreserveSession)
}

func TestClaudeProcess_ClassifyExitReason(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "normal_exit",
		},
		{
			name:     "non-exit error",
			err:      fmt.Errorf("some error"),
			expected: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason := classifyExitReason(tc.err)
			assert.Equal(t, tc.expected, reason)
		})
	}
}

func TestClaudeProcess_RestartStateInitialized(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Restart state should be initialized to zero values
	assert.Equal(t, 0, proc.restartState.Attempts)
	assert.True(t, proc.restartState.LastAttempt.IsZero())
	assert.True(t, proc.restartState.LastSuccess.IsZero())
	assert.Equal(t, time.Duration(0), proc.restartState.CurrentDelay)
}

func TestClaudeProcess_RestartConfigCustom(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:         true,
			MaxRestarts:     5,
			RestartDelay:    500 * time.Millisecond,
			MaxDelay:        1 * time.Minute,
			BackoffFactor:   1.5,
			ResetAfter:      2 * time.Minute,
			PreserveSession: false,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	assert.Equal(t, 5, proc.config.Restart.MaxRestarts)
	assert.Equal(t, 500*time.Millisecond, proc.config.Restart.RestartDelay)
	assert.Equal(t, 1*time.Minute, proc.config.Restart.MaxDelay)
	assert.Equal(t, 1.5, proc.config.Restart.BackoffFactor)
	assert.Equal(t, 2*time.Minute, proc.config.Restart.ResetAfter)
	assert.False(t, proc.config.Restart.PreserveSession)
}

// Note: Integration tests that actually crash and restart the process
// would require a mock-claude-crasher binary. These tests verify the
// infrastructure is in place and configured correctly.
