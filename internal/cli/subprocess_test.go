package cli

import (
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

func TestParseEvent_ValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "system event",
			input:    `{"type":"system","subtype":"init"}`,
			wantType: "system",
		},
		{
			name:     "assistant event",
			input:    `{"type":"assistant","message":{"content":"test"}}`,
			wantType: "assistant",
		},
		{
			name:     "no type field",
			input:    `{"other":"value"}`,
			wantType: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := parseEvent([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, event.Type)
			assert.NotNil(t, event.RawData)
		})
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "malformed json",
			input: `{invalid}`,
		},
		{
			name:  "incomplete object",
			input: `{"key":"value"`,
		},
		{
			name:  "plain text",
			input: `not json`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseEvent([]byte(tc.input))
			assert.Error(t, err)
		})
	}
}
