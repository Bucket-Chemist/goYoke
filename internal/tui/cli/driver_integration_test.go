package cli

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Integration helpers
// ---------------------------------------------------------------------------

// mockCLIPath returns the absolute path to mock-claude.sh relative to this
// source file. It uses runtime.Caller so the path is correct regardless of
// the working directory from which tests are invoked.
func mockCLIPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "mock-claude.sh")
}

// newIntegrationDriver creates a CLIDriver that uses mock-claude.sh as the
// adapter binary. Additional environment variables may be passed via envVars.
// The test's cleanup function calls Shutdown on the driver.
func newIntegrationDriver(t *testing.T, envVars map[string]string) *CLIDriver {
	t.Helper()
	opts := CLIDriverOpts{
		AdapterPath: mockCLIPath(t),
		ProjectDir:  t.TempDir(),
		EnvVars:     envVars,
	}
	d := NewCLIDriver(opts)
	t.Cleanup(func() {
		_ = d.Shutdown()
	})
	return d
}

// waitForMsg calls cmd() and returns the resulting tea.Msg. It must complete
// within 5 seconds or the test is failed with a timeout message.
//
// Use this as a drop-in replacement for direct cmd() calls to enforce the
// per-test timeout without requiring every test to spawn its own goroutine.
func waitForMsg(t *testing.T, cmd func() interface{}, label string) interface{} {
	t.Helper()
	ch := make(chan interface{}, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
		return nil // unreachable
	}
}

// ---------------------------------------------------------------------------
// Test 1: Normal flow — system.init → message → assistant + result
// ---------------------------------------------------------------------------

// TestIntegration_NormalFlow exercises the full happy-path sequence:
//
//  1. Start driver → CLIStartedMsg
//  2. WaitForEvent → SystemInitEvent
//  3. SendMessage  → nil (success)
//  4. WaitForEvent → AssistantEvent
//  5. WaitForEvent → ResultEvent
//  6. Shutdown
func TestIntegration_NormalFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	d := newIntegrationDriver(t, map[string]string{
		"MOCK_SESSION_ID": "test-session-normal",
		"MOCK_COST":       "0.0042",
		"MOCK_RESPONSE":   "hello from mock",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Step 1: Start → CLIStartedMsg.
		startMsg := waitForMsg(t, func() interface{} { return d.Start()() }, "CLIStartedMsg")
		started, ok := startMsg.(CLIStartedMsg)
		require.True(t, ok, "expected CLIStartedMsg, got %T", startMsg)
		assert.Greater(t, started.PID, 0, "PID should be positive")

		// Step 2: WaitForEvent → SystemInitEvent.
		initMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "SystemInitEvent")
		initEv, ok := initMsg.(SystemInitEvent)
		require.True(t, ok, "expected SystemInitEvent, got %T", initMsg)
		assert.Equal(t, "test-session-normal", initEv.SessionID)
		assert.Equal(t, "claude-sonnet-4-20250514", initEv.Model)
		tools := initEv.ToolNames()
		assert.NotEmpty(t, tools, "ToolNames should not be empty")

		// Step 3: SendMessage → nil (success).
		sendResult := waitForMsg(t, func() interface{} { return d.SendMessage("hello")() }, "SendMessage")
		assert.Nil(t, sendResult, "SendMessage should return nil on success")

		// Step 4: WaitForEvent → AssistantEvent.
		assistantMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "AssistantEvent")
		assistantEv, ok := assistantMsg.(AssistantEvent)
		require.True(t, ok, "expected AssistantEvent, got %T", assistantMsg)
		require.NotEmpty(t, assistantEv.Message.Content, "assistant content should not be empty")
		assert.Equal(t, "text", assistantEv.Message.Content[0].Type)
		assert.Equal(t, "hello from mock", assistantEv.Message.Content[0].Text)

		// Step 5: WaitForEvent → ResultEvent.
		resultMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "ResultEvent")
		resultEv, ok := resultMsg.(ResultEvent)
		require.True(t, ok, "expected ResultEvent, got %T", resultMsg)
		assert.Equal(t, "test-session-normal", resultEv.SessionID)
		assert.InDelta(t, 0.0042, resultEv.TotalCostUSD, 1e-9, "cost mismatch")
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TestIntegration_NormalFlow: overall test timeout")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Crash recovery — process exits with non-zero status immediately
// ---------------------------------------------------------------------------

// TestIntegration_CrashRecovery verifies that when the subprocess exits
// immediately with a non-zero exit code, the driver delivers CLIDisconnectedMsg
// and transitions to DriverError or DriverDead.
func TestIntegration_CrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	d := newIntegrationDriver(t, map[string]string{
		"MOCK_CRASH": "true",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Start → CLIStartedMsg (process starts, then exits immediately with code 1).
		startMsg := waitForMsg(t, func() interface{} { return d.Start()() }, "CLIStartedMsg")
		_, ok := startMsg.(CLIStartedMsg)
		require.True(t, ok, "expected CLIStartedMsg, got %T", startMsg)

		// WaitForEvent → CLIDisconnectedMsg (stdout pipe closes when process exits).
		discMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "CLIDisconnectedMsg")
		disc, ok := discMsg.(CLIDisconnectedMsg)
		require.True(t, ok, "expected CLIDisconnectedMsg after crash, got %T", discMsg)
		// The scanner.Err() is nil on clean EOF even when exit code != 0.
		// The key assertion is that the driver stopped, not the error value.
		_ = disc

		// Allow the consumeEvents goroutine to set the final state.
		time.Sleep(50 * time.Millisecond)

		state := d.State()
		assert.True(t,
			state == DriverDead || state == DriverError,
			"expected DriverDead or DriverError after crash, got %s", state,
		)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TestIntegration_CrashRecovery: overall test timeout")
	}
}

// ---------------------------------------------------------------------------
// Test 3: Interrupt — SIGINT causes process exit
// ---------------------------------------------------------------------------

// TestIntegration_Interrupt verifies that calling Interrupt() after the process
// has started causes it to exit and the driver to deliver CLIDisconnectedMsg.
func TestIntegration_Interrupt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	d := newIntegrationDriver(t, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Step 1: Start → CLIStartedMsg.
		startMsg := waitForMsg(t, func() interface{} { return d.Start()() }, "CLIStartedMsg")
		_, ok := startMsg.(CLIStartedMsg)
		require.True(t, ok, "expected CLIStartedMsg, got %T", startMsg)

		// Step 2: WaitForEvent → SystemInitEvent (mock emits this immediately).
		initMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "SystemInitEvent")
		_, ok = initMsg.(SystemInitEvent)
		require.True(t, ok, "expected SystemInitEvent, got %T", initMsg)

		// Step 3: Send SIGINT.
		err := d.Interrupt()
		require.NoError(t, err, "Interrupt should not error when process is running")

		// Step 4: WaitForEvent → CLIDisconnectedMsg (mock exits on SIGINT).
		discMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "CLIDisconnectedMsg after SIGINT")
		_, ok = discMsg.(CLIDisconnectedMsg)
		assert.True(t, ok, "expected CLIDisconnectedMsg after SIGINT, got %T", discMsg)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TestIntegration_Interrupt: overall test timeout")
	}
}

// ---------------------------------------------------------------------------
// Test 4: Shutdown — SIGTERM causes process exit
// ---------------------------------------------------------------------------

// TestIntegration_Shutdown verifies that calling Shutdown() after the process
// has started causes it to exit and a pending WaitForEvent to unblock with
// CLIDisconnectedMsg.
func TestIntegration_Shutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	d := newIntegrationDriver(t, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Step 1: Start → CLIStartedMsg.
		startMsg := waitForMsg(t, func() interface{} { return d.Start()() }, "CLIStartedMsg")
		_, ok := startMsg.(CLIStartedMsg)
		require.True(t, ok, "expected CLIStartedMsg, got %T", startMsg)

		// Step 2: WaitForEvent → SystemInitEvent.
		initMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "SystemInitEvent")
		_, ok = initMsg.(SystemInitEvent)
		require.True(t, ok, "expected SystemInitEvent, got %T", initMsg)

		// Step 3: Shutdown (returns immediately; process terminates asynchronously).
		err := d.Shutdown()
		require.NoError(t, err, "Shutdown should not return an error")

		// Step 4: WaitForEvent → CLIDisconnectedMsg (shutdownCh closes the select).
		discMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "CLIDisconnectedMsg after Shutdown")
		_, ok = discMsg.(CLIDisconnectedMsg)
		assert.True(t, ok, "expected CLIDisconnectedMsg after Shutdown, got %T", discMsg)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TestIntegration_Shutdown: overall test timeout")
	}
}

// ---------------------------------------------------------------------------
// Test 5: Unknown event — driver forwards CLIUnknownEvent and keeps parsing
// ---------------------------------------------------------------------------

// TestIntegration_UnknownEvent verifies that the driver forwards an
// unrecognised event type as CLIUnknownEvent and continues parsing
// subsequent well-formed events without dropping them.
func TestIntegration_UnknownEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	d := newIntegrationDriver(t, map[string]string{
		"MOCK_UNKNOWN":  "true",
		"MOCK_RESPONSE": "still works after unknown",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Step 1: Start → CLIStartedMsg.
		startMsg := waitForMsg(t, func() interface{} { return d.Start()() }, "CLIStartedMsg")
		_, ok := startMsg.(CLIStartedMsg)
		require.True(t, ok, "expected CLIStartedMsg, got %T", startMsg)

		// Step 2: WaitForEvent → SystemInitEvent (emitted before MOCK_UNKNOWN event).
		initMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "SystemInitEvent")
		_, ok = initMsg.(SystemInitEvent)
		require.True(t, ok, "expected SystemInitEvent, got %T", initMsg)

		// Step 3: WaitForEvent → CLIUnknownEvent (the "xyzzy" type line).
		unknownMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "CLIUnknownEvent")
		unknownEv, ok := unknownMsg.(CLIUnknownEvent)
		require.True(t, ok, "expected CLIUnknownEvent, got %T", unknownMsg)
		assert.Equal(t, "xyzzy", unknownEv.Type, "unknown event type mismatch")

		// Step 4: SendMessage → nil (success). The mock waits for stdin at this point.
		sendResult := waitForMsg(t, func() interface{} { return d.SendMessage("test message")() }, "SendMessage")
		assert.Nil(t, sendResult, "SendMessage should return nil on success")

		// Step 5: WaitForEvent → AssistantEvent (parsing continues after unknown event).
		assistantMsg := waitForMsg(t, func() interface{} { return d.WaitForEvent()() }, "AssistantEvent after unknown")
		assistantEv, ok := assistantMsg.(AssistantEvent)
		require.True(t, ok, "expected AssistantEvent after unknown event, got %T", assistantMsg)
		require.NotEmpty(t, assistantEv.Message.Content, "assistant content should not be empty")
		assert.Equal(t, "still works after unknown", assistantEv.Message.Content[0].Text)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TestIntegration_UnknownEvent: overall test timeout")
	}
}
