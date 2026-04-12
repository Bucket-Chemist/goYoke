package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestDriver creates a CLIDriver wired to the provided reader as stdout
// and the provided writer as stdin. This bypasses exec.Cmd entirely, letting
// tests drive the driver via pipe I/O without launching a real subprocess.
//
// The returned cleanup function should be deferred; it closes the pipes and
// waits briefly for goroutines to settle.
func newTestDriver(t *testing.T, opts CLIDriverOpts) (*CLIDriver, *io.PipeWriter, *bytes.Buffer) {
	t.Helper()

	// stdout side: test writes NDJSON lines, driver reads them.
	stdoutReader, stdoutWriter := io.Pipe()

	// stdin side: driver writes JSON messages, test reads them.
	stdinBuf := &bytes.Buffer{}

	d := &CLIDriver{
		opts:       opts,
		state:      DriverStreaming, // already "started"
		eventCh:    make(chan any, 64),
		shutdownCh: make(chan struct{}),
		waitDone:   make(chan struct{}),
		stdin:      &nopWriteCloser{stdinBuf},
		stdout:     stdoutReader,
		// cmd intentionally nil — not launching real subprocess
	}

	// Start the consume goroutine.
	go d.consumeEvents()

	return d, stdoutWriter, stdinBuf
}

// nopWriteCloser wraps a bytes.Buffer so it satisfies io.WriteCloser.
type nopWriteCloser struct {
	*bytes.Buffer
}

func (n *nopWriteCloser) Close() error { return nil }

// drainChannel reads up to max items from ch within timeout, or returns early
// once the channel is quiet.
func drainChannel(ch <-chan any, max int, timeout time.Duration) []any {
	var items []any
	deadline := time.After(timeout)
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return items
			}
			items = append(items, item)
			if len(items) >= max {
				return items
			}
		case <-deadline:
			return items
		}
	}
}

// ---------------------------------------------------------------------------
// DriverState.String
// ---------------------------------------------------------------------------

func TestDriverState_String(t *testing.T) {
	tests := []struct {
		state DriverState
		want  string
	}{
		{DriverIdle, "idle"},
		{DriverStarting, "starting"},
		{DriverStreaming, "streaming"},
		{DriverError, "error"},
		{DriverDead, "dead"},
		{DriverState(99), "DriverState(99)"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.state.String())
		})
	}
}

// ---------------------------------------------------------------------------
// NewCLIDriver
// ---------------------------------------------------------------------------

func TestNewCLIDriver_DefaultState(t *testing.T) {
	opts := CLIDriverOpts{ProjectDir: "/tmp"}
	d := NewCLIDriver(opts)

	assert.Equal(t, DriverIdle, d.State())
	assert.Equal(t, opts, d.opts)
	assert.NotNil(t, d.eventCh)
}

// ---------------------------------------------------------------------------
// buildArgs
// ---------------------------------------------------------------------------

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		opts     CLIDriverOpts
		contains []string
		absent   []string
	}{
		{
			name: "minimal opts uses acceptEdits default",
			opts: CLIDriverOpts{},
			contains: []string{
				"--input-format", "stream-json",
				"--output-format", "stream-json",
				"--verbose",
				"--include-partial-messages",
				"--permission-mode", "acceptEdits",
			},
			absent: []string{"--resume", "--model", "--mcp-config", "--allowedTools"},
		},
		{
			name: "all opts populated",
			opts: CLIDriverOpts{
				SessionID:      "sess-abc",
				Model:          "claude-opus-4-6",
				MCPConfigPath:  "/etc/mcp.json",
				PermissionMode: "plan",
				Verbose:        true,
			},
			contains: []string{
				"--resume", "sess-abc",
				"--model", "claude-opus-4-6",
				"--mcp-config", "/etc/mcp.json",
				"--allowedTools", "mcp__gofortress-interactive__*",
				"--permission-mode", "plan",
				"--verbose",
			},
		},
		{
			name:   "mcp-config omits allowedTools when path empty",
			opts:   CLIDriverOpts{MCPConfigPath: ""},
			absent: []string{"--allowedTools"},
		},
		{
			name:   "config-dir is never passed as CLI flag (not supported by claude CLI)",
			opts:   CLIDriverOpts{ConfigDir: "/home/user/.claude-em"},
			absent: []string{"--config-dir"},
		},
		{
			name:     "effort flag included when set",
			opts:     CLIDriverOpts{Effort: "high"},
			contains: []string{"--effort", "high"},
		},
		{
			name:   "effort flag omitted when empty",
			opts:   CLIDriverOpts{},
			absent: []string{"--effort"},
		},
		{
			name:   "effort flag omitted when auto",
			opts:   CLIDriverOpts{Effort: "auto"},
			absent: []string{"--effort"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := NewCLIDriver(tc.opts)
			args := d.buildArgs()
			joined := strings.Join(args, " ")

			for _, want := range tc.contains {
				assert.Contains(t, joined, want, "args should contain %q", want)
			}
			for _, absent := range tc.absent {
				assert.NotContains(t, joined, absent, "args should NOT contain %q", absent)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// consumeEvents — pipe-based tests
// ---------------------------------------------------------------------------

func TestConsumeEvents_ParsedEventsReachChannel(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})
	defer writer.Close()

	// Write two known NDJSON lines.
	line1 := `{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"ok","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u1"}`
	line2 := `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"five_hour","overageStatus":"ok","isUsingOverage":false},"uuid":"u2","session_id":"s"}`

	fmt.Fprintln(writer, line1)
	fmt.Fprintln(writer, line2)
	writer.Close()

	items := drainChannel(d.eventCh, 10, 2*time.Second)

	// Expect: ResultEvent, RateLimitEvent, CLIDisconnectedMsg (EOF).
	require.GreaterOrEqual(t, len(items), 3)

	_, isResult := items[0].(ResultEvent)
	assert.True(t, isResult, "first event should be ResultEvent, got %T", items[0])

	_, isRateLimit := items[1].(RateLimitEvent)
	assert.True(t, isRateLimit, "second event should be RateLimitEvent, got %T", items[1])

	disc, isDisconnect := items[2].(CLIDisconnectedMsg)
	assert.True(t, isDisconnect, "third item should be CLIDisconnectedMsg, got %T", items[2])
	assert.NoError(t, disc.Err)
}

func TestConsumeEvents_BlankLinesSkipped(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Write blank lines mixed with one valid event.
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "   ")
	fmt.Fprintln(writer, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`)
	fmt.Fprintln(writer, "")
	writer.Close()

	items := drainChannel(d.eventCh, 10, 2*time.Second)

	// Should only get RateLimitEvent + CLIDisconnectedMsg — no items for blank lines.
	require.GreaterOrEqual(t, len(items), 2)
	_, isRateLimit := items[0].(RateLimitEvent)
	assert.True(t, isRateLimit, "first non-blank event should be RateLimitEvent, got %T", items[0])
}

func TestConsumeEvents_MalformedLineSkipped(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Malformed JSON line followed by a valid event.
	fmt.Fprintln(writer, `{not valid json`)
	fmt.Fprintln(writer, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`)
	writer.Close()

	items := drainChannel(d.eventCh, 10, 2*time.Second)

	// Malformed line is skipped; should still get RateLimitEvent + disconnect.
	require.GreaterOrEqual(t, len(items), 2)
	_, isRateLimit := items[0].(RateLimitEvent)
	assert.True(t, isRateLimit, "expected RateLimitEvent after bad line, got %T", items[0])
}

func TestConsumeEvents_DisconnectMsgOnEOF(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Close immediately — no events.
	writer.Close()

	items := drainChannel(d.eventCh, 5, 2*time.Second)

	require.Len(t, items, 1)
	disc, ok := items[0].(CLIDisconnectedMsg)
	require.True(t, ok, "expected CLIDisconnectedMsg on EOF, got %T", items[0])
	assert.NoError(t, disc.Err)
}

func TestConsumeEvents_StateBecomesDead(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	writer.Close()

	// Wait for the goroutine to drain.
	drainChannel(d.eventCh, 5, 2*time.Second)

	// Give goroutine time to set state.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, DriverDead, d.State())
}

func TestConsumeEvents_MultipleEventTypes(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	lines := []string{
		`{"type":"system","subtype":"init","cwd":"/","session_id":"s","model":"m","permissionMode":"acceptEdits","claude_code_version":"1.0","tools":[],"uuid":"u1"}`,
		`{"type":"assistant","message":{"id":"m1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"hi"}],"stop_reason":null,"usage":{"input_tokens":1,"output_tokens":1}},"parent_tool_use_id":null,"session_id":"s","uuid":"u2"}`,
		`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u3"}`,
	}

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	writer.Close()

	items := drainChannel(d.eventCh, 10, 2*time.Second)
	require.GreaterOrEqual(t, len(items), 4) // 3 events + disconnect

	_, isInit := items[0].(SystemInitEvent)
	assert.True(t, isInit, "expected SystemInitEvent, got %T", items[0])

	_, isAssistant := items[1].(AssistantEvent)
	assert.True(t, isAssistant, "expected AssistantEvent, got %T", items[1])

	_, isResult := items[2].(ResultEvent)
	assert.True(t, isResult, "expected ResultEvent, got %T", items[2])
}

// ---------------------------------------------------------------------------
// WaitForEvent
// ---------------------------------------------------------------------------

func TestWaitForEvent_ReturnsEventFromChannel(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})

	expected := ResultEvent{Type: "result", Subtype: "success"}
	d.eventCh <- expected

	cmd := d.WaitForEvent()
	msg := cmd()

	got, ok := msg.(ResultEvent)
	require.True(t, ok, "expected ResultEvent, got %T", msg)
	assert.Equal(t, expected, got)
}

func TestWaitForEvent_ClosedChannelReturnsCLIDisconnectedMsg(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	close(d.eventCh)

	cmd := d.WaitForEvent()
	msg := cmd()

	disc, ok := msg.(CLIDisconnectedMsg)
	require.True(t, ok, "expected CLIDisconnectedMsg on closed channel, got %T", msg)
	assert.NoError(t, disc.Err)
}

func TestWaitForEvent_ResubscriptionPattern(t *testing.T) {
	// Verify the re-subscription pattern: WaitForEvent should be safe to call
	// repeatedly, each call consuming exactly one event.
	d := NewCLIDriver(CLIDriverOpts{})

	events := []any{
		SystemInitEvent{Type: "system", Subtype: "init"},
		AssistantEvent{Type: "assistant"},
		CLIDisconnectedMsg{},
	}

	for _, ev := range events {
		d.eventCh <- ev
	}

	for i, expected := range events {
		cmd := d.WaitForEvent()
		msg := cmd()
		assert.Equal(t, expected, msg, "event %d mismatch", i)
	}
}

func TestWaitForEvent_ImplementsTeaCmd(t *testing.T) {
	// Verify WaitForEvent returns a tea.Cmd (func() tea.Msg).
	d := NewCLIDriver(CLIDriverOpts{})
	d.eventCh <- CLIDisconnectedMsg{}

	var cmd tea.Cmd = d.WaitForEvent()
	assert.NotNil(t, cmd)

	msg := cmd()
	assert.NotNil(t, msg)
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

func TestSendMessage_WritesCorrectJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	d := &CLIDriver{
		opts:    CLIDriverOpts{},
		state:   DriverStreaming,
		eventCh: make(chan any, 4),
		stdin:   &nopWriteCloser{buf},
	}

	cmd := d.SendMessage("hello world")
	result := cmd()

	// Result should be nil on success.
	assert.Nil(t, result, "SendMessage should return nil on success, got %v", result)

	// Validate the written JSON.
	line := strings.TrimSpace(buf.String())
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &payload))

	assert.Equal(t, "user", payload["type"])

	msgField, ok := payload["message"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", msgField["role"])

	content, ok := msgField["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 1)

	block, ok := content[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", block["type"])
	assert.Equal(t, "hello world", block["text"])
}

func TestSendMessage_EndsWithNewline(t *testing.T) {
	buf := &bytes.Buffer{}
	d := &CLIDriver{
		opts:    CLIDriverOpts{},
		state:   DriverStreaming,
		eventCh: make(chan any, 4),
		stdin:   &nopWriteCloser{buf},
	}

	cmd := d.SendMessage("test")
	cmd()

	assert.True(t, strings.HasSuffix(buf.String(), "\n"),
		"message should end with newline, got: %q", buf.String())
}

func TestSendMessage_NilStdinReturnsError(t *testing.T) {
	d := &CLIDriver{
		opts:    CLIDriverOpts{},
		state:   DriverStreaming,
		eventCh: make(chan any, 4),
		stdin:   nil,
	}

	cmd := d.SendMessage("test")
	result := cmd()

	disc, ok := result.(CLIDisconnectedMsg)
	require.True(t, ok, "expected CLIDisconnectedMsg for nil stdin, got %T", result)
	assert.Error(t, disc.Err)
}

func TestSendMessage_MultipleMessages(t *testing.T) {
	buf := &bytes.Buffer{}
	d := &CLIDriver{
		opts:    CLIDriverOpts{},
		state:   DriverStreaming,
		eventCh: make(chan any, 4),
		stdin:   &nopWriteCloser{buf},
	}

	messages := []string{"first", "second", "third"}
	for _, msg := range messages {
		cmd := d.SendMessage(msg)
		result := cmd()
		assert.Nil(t, result)
	}

	// Each message is a separate JSON line.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 3)

	for i, line := range lines {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &payload), "line %d: %s", i, line)
	}
}

// ---------------------------------------------------------------------------
// Start — double-start guard
// ---------------------------------------------------------------------------

func TestStart_NonIdleStateReturnsError(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	d.setState(DriverStreaming)

	cmd := d.Start()
	msg := cmd()

	disc, ok := msg.(CLIDisconnectedMsg)
	require.True(t, ok, "expected CLIDisconnectedMsg, got %T", msg)
	assert.Error(t, disc.Err)
	assert.Contains(t, disc.Err.Error(), "non-idle state")
}

// ---------------------------------------------------------------------------
// Interrupt — nil process guard
// ---------------------------------------------------------------------------

func TestInterrupt_NilProcessReturnsError(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	// cmd is nil (not started)

	err := d.Interrupt()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// ---------------------------------------------------------------------------
// Shutdown — nil process is a no-op
// ---------------------------------------------------------------------------

func TestShutdown_NilProcessNoError(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	// Attach a no-op stdin so Close() succeeds.
	d.mu.Lock()
	d.stdin = &nopWriteCloser{&bytes.Buffer{}}
	d.mu.Unlock()

	err := d.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, DriverDead, d.State())
}

// ---------------------------------------------------------------------------
// State — thread safety
// ---------------------------------------------------------------------------

func TestState_ThreadSafe(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})

	var wg sync.WaitGroup
	const goroutines = 100

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = d.State()
		}()
	}

	// Concurrently mutate state.
	wg.Add(goroutines)
	states := []DriverState{DriverIdle, DriverStarting, DriverStreaming, DriverError, DriverDead}
	for i := range goroutines {
		i := i
		go func() {
			defer wg.Done()
			d.setState(states[i%len(states)])
		}()
	}

	wg.Wait()
	// As long as no race is detected (-race flag), this passes.
}

// ---------------------------------------------------------------------------
// Interrupt and Shutdown with a real subprocess
// ---------------------------------------------------------------------------

// startSleepProcess starts a "sleep 60" subprocess (or equivalent) and
// returns the driver with cmd/stdin/stdout wired up.
// The caller is responsible for calling Shutdown or Interrupt.
func startSleepProcess(t *testing.T) *CLIDriver {
	t.Helper()

	cmd := exec.Command("sleep", "60")
	stdinPipe, err := cmd.StdinPipe()
	require.NoError(t, err, "stdin pipe")

	stdoutPipe, err := cmd.StdoutPipe()
	require.NoError(t, err, "stdout pipe")

	require.NoError(t, cmd.Start(), "start sleep")

	d := &CLIDriver{
		opts:       CLIDriverOpts{},
		state:      DriverStreaming,
		eventCh:    make(chan any, 64),
		shutdownCh: make(chan struct{}),
		waitDone:   make(chan struct{}),
		cmd:        cmd,
		stdin:      stdinPipe,
		stdout:     stdoutPipe,
	}
	go d.consumeEvents()
	return d
}

func TestInterrupt_LiveProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live-process test in short mode")
	}

	d := startSleepProcess(t)
	err := d.Interrupt()
	assert.NoError(t, err)

	// Clean up.
	_ = d.Shutdown()
	time.Sleep(100 * time.Millisecond)
}

func TestShutdown_LiveProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live-process test in short mode")
	}

	d := startSleepProcess(t)
	err := d.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, DriverDead, d.State())

	// Give goroutine time to settle.
	time.Sleep(100 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// CLIDisconnectedMsg and CLIStartedMsg basic coverage
// ---------------------------------------------------------------------------

func TestCLIStartedMsg(t *testing.T) {
	msg := CLIStartedMsg{PID: 12345}
	assert.Equal(t, 12345, msg.PID)
}

func TestCLIDisconnectedMsg_NilErr(t *testing.T) {
	msg := CLIDisconnectedMsg{Err: nil}
	assert.NoError(t, msg.Err)
}

func TestCLIDisconnectedMsg_WithErr(t *testing.T) {
	msg := CLIDisconnectedMsg{Err: fmt.Errorf("pipe broken")}
	assert.Error(t, msg.Err)
}

// ---------------------------------------------------------------------------
// Channel buffering — verify capacity 64
// ---------------------------------------------------------------------------

func TestEventChannel_Capacity(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})

	// Should be able to send 64 events without blocking.
	for range 64 {
		d.eventCh <- CLIDisconnectedMsg{}
	}

	assert.Len(t, d.eventCh, 64)
}

// ---------------------------------------------------------------------------
// setState — direct coverage
// ---------------------------------------------------------------------------

func TestSetState_UpdatesUnderMutex(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	assert.Equal(t, DriverIdle, d.State())

	d.setState(DriverStreaming)
	assert.Equal(t, DriverStreaming, d.State())

	d.setState(DriverDead)
	assert.Equal(t, DriverDead, d.State())
}

// ---------------------------------------------------------------------------
// buildArgs — PermissionMode override
// ---------------------------------------------------------------------------

func TestBuildArgs_PermissionModeOverride(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{"empty defaults to acceptEdits", "", "acceptEdits"},
		{"plan explicit", "plan", "plan"},
		{"bypassPermissions explicit", "bypassPermissions", "bypassPermissions"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := NewCLIDriver(CLIDriverOpts{PermissionMode: tc.mode})
			args := d.buildArgs()
			joined := strings.Join(args, " ")
			assert.Contains(t, joined, "--permission-mode "+tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// SendMessage — verifies the exact JSON contract
// ---------------------------------------------------------------------------

func TestSendMessage_JSONStructure(t *testing.T) {
	buf := &bytes.Buffer{}
	d := &CLIDriver{
		opts:    CLIDriverOpts{},
		state:   DriverStreaming,
		eventCh: make(chan any, 4),
		stdin:   &nopWriteCloser{buf},
	}

	cmd := d.SendMessage("ping")
	result := cmd()
	assert.Nil(t, result)

	var payload userMessagePayload
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &payload))

	assert.Equal(t, "user", payload.Type)
	assert.Equal(t, "user", payload.Message.Role)
	require.Len(t, payload.Message.Content, 1)
	assert.Equal(t, "text", payload.Message.Content[0].Type)
	assert.Equal(t, "ping", payload.Message.Content[0].Text)
}

// ---------------------------------------------------------------------------
// WaitForEvent — CLIDisconnectedMsg flows through channel correctly
// ---------------------------------------------------------------------------

func TestWaitForEvent_DisconnectEventPassthrough(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	expected := CLIDisconnectedMsg{Err: fmt.Errorf("broken pipe")}
	d.eventCh <- expected

	cmd := d.WaitForEvent()
	msg := cmd()

	disc, ok := msg.(CLIDisconnectedMsg)
	require.True(t, ok)
	assert.EqualError(t, disc.Err, "broken pipe")
}

// ---------------------------------------------------------------------------
// Large line handling (1MB buffer)
// ---------------------------------------------------------------------------

func TestConsumeEvents_LargeLineHandled(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Construct a valid JSON event with a large text field (512 KB).
	largeText := strings.Repeat("x", 512*1024)
	largeEvent := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":    "msg_large",
			"type":  "message",
			"role":  "assistant",
			"model": "m",
			"content": []map[string]any{
				{"type": "text", "text": largeText},
			},
			"stop_reason": nil,
			"usage": map[string]any{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		},
		"parent_tool_use_id": nil,
		"session_id":         "s",
		"uuid":               "u_large",
	}

	data, err := json.Marshal(largeEvent)
	require.NoError(t, err)

	fmt.Fprintln(writer, string(data))
	writer.Close()

	items := drainChannel(d.eventCh, 5, 3*time.Second)
	require.GreaterOrEqual(t, len(items), 1)

	ev, ok := items[0].(AssistantEvent)
	require.True(t, ok, "expected AssistantEvent for large line, got %T", items[0])
	require.Len(t, ev.Message.Content, 1)
	assert.Len(t, ev.Message.Content[0].Text, len(largeText))
}

func TestConsumeEvents_VeryLargeLineHandled(t *testing.T) {
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Valid assistant events can exceed 1 MB when a tool result embeds a large
	// file payload. The driver must keep streaming instead of disconnecting.
	largeText := strings.Repeat("x", 2*1024*1024)
	largeEvent := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":    "msg_very_large",
			"type":  "message",
			"role":  "assistant",
			"model": "m",
			"content": []map[string]any{
				{"type": "text", "text": largeText},
			},
			"stop_reason": nil,
			"usage": map[string]any{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		},
		"parent_tool_use_id": nil,
		"session_id":         "s",
		"uuid":               "u_very_large",
	}

	data, err := json.Marshal(largeEvent)
	require.NoError(t, err)

	fmt.Fprintln(writer, string(data))
	writer.Close()

	items := drainChannel(d.eventCh, 5, 3*time.Second)
	require.GreaterOrEqual(t, len(items), 1)

	ev, ok := items[0].(AssistantEvent)
	require.True(t, ok, "expected AssistantEvent for very large line, got %T", items[0])
	require.Len(t, ev.Message.Content, 1)
	assert.Len(t, ev.Message.Content[0].Text, len(largeText))
}

// ---------------------------------------------------------------------------
// W-3/M-1: shutdownCh / waitDone — SIGKILL goroutine cancellation
// ---------------------------------------------------------------------------

// TestShutdown_SIGKILLGoroutineCancelledWhenProcessExitsCleanly verifies that
// when a process exits before the 2-second SIGKILL deadline, the escalation
// goroutine cancels itself via waitDone and does NOT send SIGKILL. We test
// this indirectly by verifying that waitDone is closed promptly (indicating
// consumeEvents reached cmd.Wait()) and that the driver reaches DriverDead.
func TestShutdown_SIGKILLGoroutineCancelledWhenProcessExitsCleanly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live-process test in short mode")
	}

	// Use a process that exits quickly on its own (sleep 0).
	cmd := exec.Command("sleep", "0")
	stdinPipe, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdoutPipe, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	d := &CLIDriver{
		opts:       CLIDriverOpts{},
		state:      DriverStreaming,
		eventCh:    make(chan any, 64),
		shutdownCh: make(chan struct{}),
		waitDone:   make(chan struct{}),
		cmd:        cmd,
		stdin:      stdinPipe,
		stdout:     stdoutPipe,
	}
	go d.consumeEvents()

	// waitDone must close well before the 2-second SIGKILL deadline.
	select {
	case <-d.waitDone:
		// Good — process exited, consumeEvents closed waitDone.
	case <-time.After(2 * time.Second):
		t.Fatal("waitDone was not closed within 2s — process did not exit or consumeEvents stalled")
	}

	assert.Equal(t, DriverDead, d.State())
}

// TestShutdown_ShutdownChUnblocksPendingWaitForEvent verifies that a
// WaitForEvent Cmd that is already blocking on the event channel returns
// CLIDisconnectedMsg immediately when Shutdown closes shutdownCh.
func TestShutdown_ShutdownChUnblocksPendingWaitForEvent(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	// eventCh is empty — WaitForEvent would block indefinitely without shutdownCh.

	resultCh := make(chan tea.Msg, 1)
	go func() {
		cmd := d.WaitForEvent()
		resultCh <- cmd()
	}()

	// Give the goroutine time to block on WaitForEvent.
	time.Sleep(20 * time.Millisecond)

	// Shutdown closes shutdownCh, which should unblock the goroutine.
	require.NoError(t, d.Shutdown())

	select {
	case msg := <-resultCh:
		disc, ok := msg.(CLIDisconnectedMsg)
		require.True(t, ok, "expected CLIDisconnectedMsg, got %T", msg)
		assert.NoError(t, disc.Err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForEvent was not unblocked within 2s after Shutdown")
	}
}

// TestShutdown_IdempotentDoesNotPanic verifies that calling Shutdown twice on
// the same driver does not panic (shutdownCh is closed at most once).
func TestShutdown_IdempotentDoesNotPanic(t *testing.T) {
	d := NewCLIDriver(CLIDriverOpts{})
	d.mu.Lock()
	d.stdin = &nopWriteCloser{&bytes.Buffer{}}
	d.mu.Unlock()

	assert.NotPanics(t, func() {
		_ = d.Shutdown()
		_ = d.Shutdown() // second call must not panic
	})
}

// TestConsumeEvents_ExitsOnShutdownCh verifies that consumeEvents stops
// sending to eventCh and exits promptly when shutdownCh is closed mid-stream.
func TestConsumeEvents_ExitsOnShutdownCh(t *testing.T) {
	stdoutReader, stdoutWriter := io.Pipe()
	stdinBuf := &bytes.Buffer{}

	d := &CLIDriver{
		opts:       CLIDriverOpts{},
		state:      DriverStreaming,
		eventCh:    make(chan any, 2), // small buffer so it fills up
		shutdownCh: make(chan struct{}),
		waitDone:   make(chan struct{}),
		stdin:      &nopWriteCloser{stdinBuf},
		stdout:     stdoutReader,
	}

	go d.consumeEvents()

	// Close shutdownCh while stdoutWriter is still open — consumeEvents should
	// exit via the shutdownCh arm without waiting for EOF.
	close(d.shutdownCh)

	// waitDone must close promptly (consumeEvents reaches its end).
	select {
	case <-d.waitDone:
		// Good.
	case <-time.After(2 * time.Second):
		t.Fatal("consumeEvents did not exit after shutdownCh closed")
	}

	// Clean up the pipe.
	stdoutWriter.Close()
}
