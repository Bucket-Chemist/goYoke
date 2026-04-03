// Package cli provides Go struct definitions for all NDJSON event types
// emitted by the Claude Code CLI when invoked with --output-format stream-json.
//
// driver.go implements CLIDriver, which manages the claude subprocess lifecycle,
// reads its NDJSON output stream, and bridges parsed events into the Bubbletea
// command/message loop.
//
// The channel-to-Cmd re-subscription pattern keeps the event pump alive:
// after processing each message the root AppModel must call d.WaitForEvent()
// to schedule the next read from the internal event channel.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// DriverState
// ---------------------------------------------------------------------------

// DriverState represents the subprocess lifecycle state.
type DriverState int

const (
	// DriverIdle is the state before Start() is called.
	DriverIdle DriverState = iota

	// DriverStarting indicates Start() was called and the process is launching.
	DriverStarting

	// DriverStreaming indicates the process is running and events are being read.
	DriverStreaming

	// DriverError indicates the process exited with a non-zero status.
	DriverError

	// DriverDead indicates the process exited cleanly or was killed.
	DriverDead
)

// String returns a human-readable name for the state.
func (s DriverState) String() string {
	switch s {
	case DriverIdle:
		return "idle"
	case DriverStarting:
		return "starting"
	case DriverStreaming:
		return "streaming"
	case DriverError:
		return "error"
	case DriverDead:
		return "dead"
	default:
		return fmt.Sprintf("DriverState(%d)", int(s))
	}
}

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// CLIStartedMsg is sent when the subprocess starts successfully.
type CLIStartedMsg struct {
	PID int
}

// CLIDisconnectedMsg is sent when the subprocess exits or the stdout pipe
// breaks. Err is nil for a clean exit.
type CLIDisconnectedMsg struct {
	Err error
}

// ---------------------------------------------------------------------------
// CLIDriverOpts
// ---------------------------------------------------------------------------

// CLIDriverOpts configures the CLI subprocess.
type CLIDriverOpts struct {
	// SessionID resumes an existing claude session. Empty means new session.
	SessionID string

	// Model overrides the default model for this session. Empty means default.
	Model string

	// MCPConfigPath is the path to the MCP configuration file. Empty omits the flag.
	MCPConfigPath string

	// ProjectDir sets the working directory for the claude process.
	// Empty means the current working directory.
	ProjectDir string

	// Verbose enables verbose output from the claude subprocess.
	Verbose bool

	// PermissionMode sets the permission mode (e.g., "acceptEdits", "plan").
	// Empty defaults to "acceptEdits".
	PermissionMode string

	// Debug causes the driver to log debug-level messages via slog.
	Debug bool

	// AdapterPath is the adapter binary name for non-Anthropic providers
	// (e.g. "gemini-adapter", "openai-adapter").  When non-empty the adapter
	// binary is invoked instead of "claude" and the adapter is responsible
	// for translating between the provider's API and the Claude stream-json
	// protocol.  Empty means use the native Anthropic claude binary.
	AdapterPath string

	// EnvVars are additional environment variables to pass to the subprocess.
	// Used for provider-specific credentials such as OPENAI_API_KEY or
	// OLLAMA_ENDPOINT.  The current process environment is always inherited;
	// EnvVars entries override or extend it.
	EnvVars map[string]string

	// ConfigDir overrides the Claude config directory (e.g. ~/.claude-em). Empty means default.
	ConfigDir string
}

// ---------------------------------------------------------------------------
// CLIDriver
// ---------------------------------------------------------------------------

// CLIDriver manages a claude CLI subprocess and bridges its NDJSON output
// stream into the Bubbletea event loop.
//
// The zero value is not usable; use NewCLIDriver instead.
//
// Concurrency: all exported methods are goroutine-safe. Internal state is
// protected by mu. The eventCh is written by the consumeEvents goroutine and
// read by WaitForEvent commands scheduled by the Bubbletea runtime.
//
// Lifecycle channels:
//   - shutdownCh is closed by Shutdown to signal both consumeEvents (producer)
//     and WaitForEvent (consumer) to stop selecting on eventCh, preventing
//     goroutine leaks on provider switch.
//   - waitDone is closed by consumeEvents after cmd.Wait() returns, indicating
//     the subprocess has exited. The SIGKILL escalation goroutine in Shutdown
//     selects on waitDone to cancel itself when the process exits cleanly.
type CLIDriver struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	eventCh    chan any
	shutdownCh chan struct{} // closed by Shutdown; cancels producer and consumer
	waitDone   chan struct{} // closed by consumeEvents after cmd.Wait() returns
	state      DriverState
	mu         sync.Mutex
	opts       CLIDriverOpts
}

// NewCLIDriver creates a CLIDriver configured with the given options.
// The driver starts in DriverIdle state; call Start() to launch the subprocess.
func NewCLIDriver(opts CLIDriverOpts) *CLIDriver {
	return &CLIDriver{
		opts:       opts,
		state:      DriverIdle,
		eventCh:    make(chan any, 64),
		shutdownCh: make(chan struct{}),
		waitDone:   make(chan struct{}),
	}
}

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

// Start builds the claude subprocess command, creates stdio pipes, and
// launches the process. It returns a tea.Cmd that produces CLIStartedMsg on
// success or CLIDisconnectedMsg on failure.
//
// Start must be called at most once per CLIDriver. Calling Start on an
// already-started driver returns CLIDisconnectedMsg with an error.
func (d *CLIDriver) Start() tea.Cmd {
	return func() tea.Msg {
		d.mu.Lock()
		if d.state != DriverIdle {
			d.mu.Unlock()
			return CLIDisconnectedMsg{
				Err: fmt.Errorf("cli driver: Start called in non-idle state %s", d.state),
			}
		}
		d.state = DriverStarting
		d.mu.Unlock()

		args := d.buildArgs()

		if d.opts.Debug {
			slog.Debug("cli driver: launching subprocess", "args", args)
		}

		binary := "claude"
		if d.opts.AdapterPath != "" {
			binary = d.opts.AdapterPath
		}

		cmd := exec.Command(binary, args...) //nolint:gosec // args are built from structured opts
		if d.opts.ProjectDir != "" {
			cmd.Dir = d.opts.ProjectDir
		}

		// Run in a dedicated process group so Interrupt() can signal the
		// entire tree (claude CLI + MCP servers + API calls) via Kill(-pgid).
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// Merge extra env vars into the subprocess environment.
		if len(d.opts.EnvVars) > 0 {
			env := os.Environ()
			for k, v := range d.opts.EnvVars {
				env = append(env, k+"="+v)
			}
			cmd.Env = env
		}

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			d.setState(DriverError)
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: create stdin pipe: %w", err)}
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			stdinPipe.Close()
			d.setState(DriverError)
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: create stdout pipe: %w", err)}
		}

		if err := cmd.Start(); err != nil {
			stdinPipe.Close()
			stdoutPipe.Close()
			d.setState(DriverError)
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: start subprocess: %w", err)}
		}

		d.mu.Lock()
		d.cmd = cmd
		d.stdin = stdinPipe
		d.stdout = stdoutPipe
		d.state = DriverStreaming
		d.mu.Unlock()

		go d.consumeEvents()

		if d.opts.Debug {
			slog.Debug("cli driver: subprocess started", "pid", cmd.Process.Pid)
		}

		return CLIStartedMsg{PID: cmd.Process.Pid}
	}
}

// buildArgs constructs the argument slice for the claude command.
func (d *CLIDriver) buildArgs() []string {
	// --verbose is REQUIRED: claude CLI 2.1.81+ requires --verbose when
	// using --output-format stream-json, even in interactive (non-print) mode.
	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
	}

	// NOTE: --config-dir is NOT a valid claude CLI flag. The config directory
	// override is propagated via the CLAUDE_CONFIG_DIR environment variable,
	// which is set in cmd/gofortress/main.go when --config-dir is passed to
	// the TUI. The claude subprocess inherits this env var automatically.

	permMode := d.opts.PermissionMode
	if permMode == "" {
		permMode = "acceptEdits"
	}
	args = append(args, "--permission-mode", permMode)

	if d.opts.SessionID != "" {
		args = append(args, "--resume", d.opts.SessionID)
	}

	if d.opts.Model != "" {
		args = append(args, "--model", d.opts.Model)
	}

	if d.opts.MCPConfigPath != "" {
		args = append(args, "--mcp-config", d.opts.MCPConfigPath)
		args = append(args, "--allowedTools",
			"Bash,Read,Write,Edit,Glob,Grep,WebSearch,WebFetch,NotebookEdit,"+
				"TodoWrite,EnterPlanMode,ExitPlanMode,Skill,ToolSearch,AskUserQuestion,"+
				"mcp__gofortress-interactive__*")
	}

	// Block the built-in Agent tool — all agent spawning must go through
	// mcp__gofortress-interactive__spawn_agent which injects identity,
	// conventions, and rules via buildFullAgentContext().
	// Agent() bypasses all PreToolUse hooks and fires no context injection.
	// Enforcement: routing-schema.json → this CLI flag → CLAUDE.md reference.
	args = append(args, "--disallowedTools", "Agent")

	// NOTE: --verbose is already unconditionally included above (required by
	// claude CLI 2.1.81+ for stream-json output). The d.opts.Verbose flag
	// controls debug logging in the TUI itself, not the CLI subprocess.

	return args
}

// ---------------------------------------------------------------------------
// consumeEvents goroutine
// ---------------------------------------------------------------------------

// scannerBufSize is the maximum line length accepted by the NDJSON scanner.
// 1 MB accommodates large tool outputs such as file reads.
const scannerBufSize = 1024 * 1024

// consumeEvents reads NDJSON lines from d.stdout, parses each line with
// ParseCLIEvent, and sends parsed events to d.eventCh. It runs in its own
// goroutine for the lifetime of the subprocess.
//
// When the stdout pipe closes (either because the process exited or because
// the pipe was broken), consumeEvents sends a CLIDisconnectedMsg to eventCh,
// sets the driver state to DriverDead, and waits for the subprocess to exit.
//
// The scanner runs in a dedicated inner goroutine so that the main loop can
// select on both scanner output and shutdownCh. This ensures that closing
// shutdownCh causes consumeEvents to exit immediately even when no data is
// flowing on stdout — preventing goroutine leaks on provider switch.
func (d *CLIDriver) consumeEvents() {
	defer close(d.waitDone)

	scanner := bufio.NewScanner(d.stdout)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)

	// scanResult carries one parsed event or the terminal state of the scanner.
	type scanResult struct {
		event any
		done  bool
		err   error
	}

	// Buffer of 1 so the inner goroutine never blocks after the main loop exits.
	scanCh := make(chan scanResult, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Bytes()

			event, err := ParseCLIEvent(line)
			if err != nil {
				slog.Warn("cli driver: parse error, skipping line",
					"err", err,
					"line_len", len(line),
				)
				continue
			}

			if event == nil {
				continue
			}

			select {
			case scanCh <- scanResult{event: event}:
			case <-d.shutdownCh:
				return
			}
		}
		select {
		case scanCh <- scanResult{done: true, err: scanner.Err()}:
		case <-d.shutdownCh:
		}
	}()

	for {
		select {
		case <-d.shutdownCh:
			d.setState(DriverDead)
			if d.cmd != nil {
				_ = d.cmd.Wait()
			}
			return

		case result := <-scanCh:
			if result.done {
				if d.opts.Debug && result.err != nil {
					slog.Debug("cli driver: scanner error", "err", result.err)
				}

				select {
				case d.eventCh <- CLIDisconnectedMsg{Err: result.err}:
				case <-d.shutdownCh:
				}

				d.setState(DriverDead)
				if d.cmd != nil {
					_ = d.cmd.Wait()
				}
				return
			}

			select {
			case d.eventCh <- result.event:
			case <-d.shutdownCh:
				d.setState(DriverDead)
				if d.cmd != nil {
					_ = d.cmd.Wait()
				}
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// WaitForEvent
// ---------------------------------------------------------------------------

// WaitForEvent returns a tea.Cmd that blocks until one event is available on
// the internal channel and returns it as a tea.Msg.
//
// IMPORTANT: The root AppModel must call d.WaitForEvent() after processing
// each CLI event to maintain the subscription. This is the standard Bubbletea
// channel-to-Cmd re-subscription pattern.
//
// When the channel is closed or Shutdown is called, WaitForEvent returns
// CLIDisconnectedMsg. The shutdownCh select arm prevents a pending
// WaitForEvent from blocking forever after a provider switch.
func (d *CLIDriver) WaitForEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-d.eventCh:
			if !ok {
				return CLIDisconnectedMsg{Err: nil}
			}
			// tea.Msg is defined as interface{}/any in Bubbletea, so this cast
			// always succeeds for any non-nil value.
			return event.(tea.Msg) //nolint:forcetypeassert
		case <-d.shutdownCh:
			return CLIDisconnectedMsg{Err: nil}
		}
	}
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

// userMessagePayload is the JSON structure written to claude's stdin.
type userMessagePayload struct {
	Type    string             `json:"type"`
	Message userMessageContent `json:"message"`
}

type userMessageContent struct {
	Role    string            `json:"role"`
	Content []userTextContent `json:"content"`
}

type userTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SendMessage writes a user message to the subprocess stdin and returns a
// tea.Cmd. The command returns nil on success or CLIDisconnectedMsg on error.
//
// The message is serialised as:
//
//	{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<text>"}]}}
func (d *CLIDriver) SendMessage(text string) tea.Cmd {
	return func() tea.Msg {
		payload := userMessagePayload{
			Type: "user",
			Message: userMessageContent{
				Role: "user",
				Content: []userTextContent{
					{Type: "text", Text: text},
				},
			},
		}

		data, err := json.Marshal(payload)
		if err != nil {
			// json.Marshal on plain structs never errors in practice.
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: marshal message: %w", err)}
		}

		d.mu.Lock()
		stdinPipe := d.stdin
		d.mu.Unlock()

		if stdinPipe == nil {
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: stdin not open")}
		}

		// Write JSON line followed by newline.
		if _, err := fmt.Fprintf(stdinPipe, "%s\n", data); err != nil {
			return CLIDisconnectedMsg{Err: fmt.Errorf("cli driver: write to stdin: %w", err)}
		}

		return nil
	}
}

// ---------------------------------------------------------------------------
// Interrupt
// ---------------------------------------------------------------------------

// Interrupt sends SIGINT to the subprocess, requesting a graceful
// cancellation of the current operation. It does not wait for the process
// to exit.
func (d *CLIDriver) Interrupt() error {
	d.mu.Lock()
	cmd := d.cmd
	d.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("cli driver: subprocess not running")
	}

	// Send SIGINT to the entire process group (negative PID) so that all
	// child processes (MCP servers, spawned agents, API calls) also receive
	// the signal.  Matches the spawner pattern in mcp/spawner.go:244.
	pid := cmd.Process.Pid
	if err := syscall.Kill(-pid, syscall.SIGINT); err != nil {
		// Fallback: try single-process signal if group signal fails.
		if err2 := cmd.Process.Signal(syscall.SIGINT); err2 != nil {
			return fmt.Errorf("cli driver: send SIGINT (group=%v, proc=%v)", err, err2)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shutdown
// ---------------------------------------------------------------------------

// Shutdown terminates the subprocess gracefully. It:
//  1. Sets the driver state to DriverDead.
//  2. Closes shutdownCh to unblock any pending WaitForEvent and consumeEvents.
//  3. Sends SIGTERM to the subprocess.
//  4. After 2 seconds, sends SIGKILL if the process has not yet exited.
//     The escalation goroutine is cancelled early via waitDone if the process
//     exits cleanly before the 2-second deadline.
//  5. Closes the stdin pipe.
//
// Shutdown returns immediately; the SIGKILL escalation runs in a goroutine.
// Calling Shutdown multiple times is safe: shutdownCh is closed at most once
// under the driver mutex.
func (d *CLIDriver) Shutdown() error {
	d.setState(DriverDead)

	// Close shutdownCh exactly once so that both the consumeEvents goroutine
	// (producer) and any blocked WaitForEvent Cmd (consumer) exit promptly.
	d.mu.Lock()
	select {
	case <-d.shutdownCh:
		// Already closed by a previous Shutdown call — nothing to do.
	default:
		close(d.shutdownCh)
	}
	cmd := d.cmd
	stdinPipe := d.stdin
	d.mu.Unlock()

	if stdinPipe != nil {
		_ = stdinPipe.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited; best-effort SIGKILL.
		_ = cmd.Process.Signal(syscall.SIGKILL)
		return nil
	}

	// SIGKILL escalation after 2 seconds if the process is still running.
	// waitDone is closed by consumeEvents once cmd.Wait() returns, so the
	// goroutine cancels itself when the process exits cleanly before the
	// deadline — preventing accumulated goroutines on repeated provider
	// switches.
	proc := cmd.Process
	waitDone := d.waitDone
	go func() {
		select {
		case <-waitDone:
			// Process already exited; SIGKILL not needed.
		case <-time.After(2 * time.Second):
			// Best-effort; ignore error if process already exited.
			_ = proc.Signal(syscall.SIGKILL)
		}
	}()

	return nil
}

// ---------------------------------------------------------------------------
// State accessor
// ---------------------------------------------------------------------------

// State returns the current lifecycle state of the driver. Thread-safe.
func (d *CLIDriver) State() DriverState {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state
}

// setState updates the driver state under the mutex.
func (d *CLIDriver) setState(s DriverState) {
	d.mu.Lock()
	d.state = s
	d.mu.Unlock()
}
