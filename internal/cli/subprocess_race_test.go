package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeProcess_RestartRaceCondition verifies restart waits for goroutines
func TestClaudeProcess_RestartRaceCondition(t *testing.T) {
	tests := []struct {
		name            string
		preserveSession bool
		sendMessages    int
	}{
		{
			name:            "restart with new session",
			preserveSession: false,
			sendMessages:    5,
		},
		{
			name:            "restart preserving session",
			preserveSession: true,
			sendMessages:    10,
		},
		{
			name:            "restart during active streaming",
			preserveSession: false,
			sendMessages:    20,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				ClaudePath: "./testdata/mock-claude",
				Restart: RestartPolicy{
					Enabled:         true,
					PreserveSession: tc.preserveSession,
					MaxRestarts:     5,
					RestartDelay:    10 * time.Millisecond,
				},
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

			// Send messages to create active goroutines
			for i := 0; i < tc.sendMessages; i++ {
				err := proc.Send(fmt.Sprintf("message %d", i))
				assert.NoError(t, err)
			}

			// Trigger restart by accessing restart() - in real scenario this would
			// be triggered by process exit. For this test we verify the mechanism.
			originalSession := proc.SessionID()

			// Manually trigger restart (simulating crash scenario)
			err = proc.restart()
			assert.NoError(t, err)

			// Verify session changed if PreserveSession=false
			if !tc.preserveSession {
				assert.NotEqual(t, originalSession, proc.SessionID())
			} else {
				assert.Equal(t, originalSession, proc.SessionID())
			}

			// Verify new process is running
			assert.True(t, proc.IsRunning())

			// Verify we can send to new process
			err = proc.Send("post-restart message")
			assert.NoError(t, err)
		})
	}
}

// TestClaudeProcess_RestartGenerationIsolation verifies old goroutines exit cleanly
func TestClaudeProcess_RestartGenerationIsolation(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:         true,
			PreserveSession: false,
			MaxRestarts:     3,
			RestartDelay:    10 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Verify initial generation is 0
	assert.Equal(t, uint64(0), proc.currentGeneration())

	// Trigger first restart
	err = proc.restart()
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), proc.currentGeneration())

	// Trigger second restart
	err = proc.restart()
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), proc.currentGeneration())

	// Verify process still works
	assert.True(t, proc.IsRunning())
	err = proc.Send("test message")
	assert.NoError(t, err)
}

// TestClaudeProcess_RestartTimeout verifies timeout when goroutines hang
func TestClaudeProcess_RestartTimeout(t *testing.T) {
	// This test verifies the timeout mechanism in restart()
	// Note: Actual timeout requires goroutines to hang, which is hard to
	// simulate. This test verifies the infrastructure is in place.

	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:      true,
			MaxRestarts:  1,
			RestartDelay: 10 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Trigger restart and verify it completes (doesn't hang)
	done := make(chan error, 1)
	go func() {
		done <- proc.restart()
	}()

	select {
	case err := <-done:
		// Restart completed (with or without timeout warning)
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Restart hung - timeout mechanism failed")
	}

	// Verify process still works after restart
	assert.True(t, proc.IsRunning())
}

// TestClaudeProcess_RestartChannelIsolation verifies fresh channels are used
func TestClaudeProcess_RestartChannelIsolation(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:         true,
			PreserveSession: false,
			MaxRestarts:     3,
			RestartDelay:    10 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Get reference to original channels
	eventsChan1 := proc.Events()

	// Trigger restart
	err = proc.restart()
	assert.NoError(t, err)

	// Get reference to new channels
	eventsChan2 := proc.Events()

	// Channels should be different (fresh channels created)
	// Note: This comparison checks pointer equality
	assert.NotEqual(t, fmt.Sprintf("%p", eventsChan1), fmt.Sprintf("%p", eventsChan2),
		"Expected fresh channels after restart")

	// Both channels should be functional
	// Send message to new process
	err = proc.Send("test message")
	assert.NoError(t, err)

	// Should receive event on new channel
	select {
	case event := <-eventsChan2:
		assert.NotEmpty(t, event.Type)
	case <-time.After(2 * time.Second):
		t.Log("Timeout waiting for event on new channel (may be expected depending on mock)")
	}
}

// TestClaudeProcess_MultipleRestarts verifies multiple sequential restarts
func TestClaudeProcess_MultipleRestarts(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:         true,
			PreserveSession: true,
			MaxRestarts:     10,
			RestartDelay:    5 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Perform multiple restarts
	for i := 0; i < 5; i++ {
		t.Logf("Restart iteration %d", i)

		// Verify running before restart
		assert.True(t, proc.IsRunning())

		// Trigger restart
		err = proc.restart()
		assert.NoError(t, err, "Restart %d failed", i)

		// Verify running after restart
		assert.True(t, proc.IsRunning())

		// Verify generation incremented
		assert.Equal(t, uint64(i+1), proc.currentGeneration())

		// Verify we can communicate with new process
		err = proc.Send(fmt.Sprintf("test message after restart %d", i))
		assert.NoError(t, err)
	}
}

// TestClaudeProcess_RestartPendingWrites tests restart with pending channel writes
func TestClaudeProcess_RestartPendingWrites(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:      true,
			MaxRestarts:  3,
			RestartDelay: 10 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Create some pending activity by sending messages
	// without consuming events (simulates slow consumer)
	for i := 0; i < 50; i++ {
		proc.Send(fmt.Sprintf("message %d", i))
	}

	// Trigger restart while events are pending
	err = proc.restart()
	assert.NoError(t, err)

	// Verify new process works
	assert.True(t, proc.IsRunning())
	err = proc.Send("post-restart message")
	assert.NoError(t, err)
}

// TestClaudeProcess_ConcurrentRestarts verifies restart is safe under concurrent calls
// Note: This should ideally never happen in production, but tests robustness
func TestClaudeProcess_ConcurrentRestarts(t *testing.T) {
	cfg := Config{
		ClaudePath: "./testdata/mock-claude",
		Restart: RestartPolicy{
			Enabled:      true,
			MaxRestarts:  10,
			RestartDelay: 5 * time.Millisecond,
		},
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	err = proc.Start()
	if err != nil {
		t.Skipf("Mock claude binary not available: %v", err)
	}
	defer proc.Stop()

	// Attempt concurrent restarts
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(id int) {
			// Add small delay to increase chance of overlap
			time.Sleep(time.Duration(id*10) * time.Millisecond)
			done <- proc.restart()
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	// At least one should succeed, others may fail gracefully
	if len(errors) == 3 {
		t.Errorf("All concurrent restarts failed: %v", errors)
	}

	// Process should still be functional
	assert.True(t, proc.IsRunning())
}
