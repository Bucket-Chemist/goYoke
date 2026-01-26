package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event types are now defined in events.go (GOgent-114).
// This file uses the Event type from events.go for full event parsing.

// Config holds configuration for starting a Claude subprocess.
type Config struct {
	// ClaudePath is the path to the claude binary (default: "claude")
	ClaudePath string

	// SessionID is an explicit session ID. If empty, a UUID is generated.
	SessionID string

	// SettingsPath is a custom settings.json path.
	// If provided, passed as --settings flag.
	SettingsPath string

	// WorkingDir is the working directory for the claude process.
	// If empty, inherits from parent process.
	WorkingDir string

	// Verbose enables verbose output (--verbose flag)
	Verbose bool

	// IncludePartial enables partial message streaming (--include-partial-messages)
	IncludePartial bool

	// Restart defines auto-restart behavior for the subprocess.
	// If not set, DefaultRestartPolicy() is used.
	Restart RestartPolicy
}

// ClaudeProcess manages the lifecycle of a claude subprocess.
// It handles stdin/stdout communication via NDJSON streams,
// event reading in a background goroutine, and graceful shutdown.
type ClaudeProcess struct {
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	writer        *NDJSONWriter
	config        Config
	sessionID     string
	events        chan Event
	errors        chan error
	restartEvents chan RestartEvent
	done          chan struct{}
	exitChan      chan error
	mu            sync.Mutex
	running       bool
	explicitStop  bool // Set to true when Stop() is called
	restartState  RestartState
}

// NewClaudeProcess creates a ClaudeProcess with the given config.
// Does not start the process - call Start() to begin execution.
// SessionID is generated if not provided in config.
func NewClaudeProcess(cfg Config) (*ClaudeProcess, error) {
	// Default claude path
	if cfg.ClaudePath == "" {
		cfg.ClaudePath = "claude"
	}

	// Generate session ID if not provided
	sessionID := cfg.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Use default restart policy if not configured.
	// We detect "not configured" by checking if RestartDelay is zero.
	// If user wants to disable restart, they should set Enabled=false AND
	// any other field (e.g., MaxRestarts=0) to prevent default application.
	if cfg.Restart.RestartDelay == 0 && cfg.Restart.MaxRestarts == 0 && cfg.Restart.MaxDelay == 0 {
		// Completely zero-valued struct - apply defaults
		cfg.Restart = DefaultRestartPolicy()
	}
	// Otherwise user has configured something, use their values

	// Build command arguments
	args := []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--session-id", sessionID,
	}

	if cfg.Verbose {
		args = append(args, "--verbose")
	}

	if cfg.IncludePartial {
		args = append(args, "--include-partial-messages")
	}

	if cfg.SettingsPath != "" {
		args = append(args, "--settings", cfg.SettingsPath)
	}

	cmd := exec.Command(cfg.ClaudePath, args...)
	if cfg.WorkingDir != "" {
		cmd.Dir = cfg.WorkingDir
	}

	return &ClaudeProcess{
		cmd:           cmd,
		config:        cfg,
		sessionID:     sessionID,
		events:        make(chan Event, 100),        // Buffered to prevent blocking
		errors:        make(chan error, 10),         // Buffered for errors
		restartEvents: make(chan RestartEvent, 10),  // Buffered for restart events
		done:          make(chan struct{}),
		exitChan:      make(chan error, 1),          // Buffered for process exit
		explicitStop:  false,
	}, nil
}

// Start spawns the claude subprocess and begins reading events.
// Returns error if process fails to start or pipes cannot be created.
// Does not block - events are read in a background goroutine.
func (cp *ClaudeProcess) Start() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.running {
		return fmt.Errorf("process already running")
	}

	// Set up pipes
	stdin, err := cp.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	cp.stdin = stdin
	cp.writer = NewNDJSONWriter(stdin)

	stdout, err := cp.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cp.stdout = stdout

	stderr, err := cp.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	cp.stderr = stderr

	// Start the process
	if err := cp.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude process: %w", err)
	}

	cp.running = true
	cp.explicitStop = false
	cp.restartState.RecordSuccess()

	// Start reading events in background
	go cp.readEvents()
	go cp.readStderr()

	// Always start restart monitor to handle process exit
	// It will honor the Enabled flag for restart behavior
	go cp.monitorRestart()

	return nil
}

// Stop gracefully shuts down the claude process.
// Closes stdin to signal EOF, waits up to 5 seconds for clean exit,
// then sends SIGKILL if process hasn't terminated.
// Safe to call multiple times - returns nil if already stopped.
// Prevents auto-restart when called explicitly.
func (cp *ClaudeProcess) Stop() error {
	cp.mu.Lock()

	if !cp.running {
		cp.mu.Unlock()
		return nil
	}

	// Mark as explicit stop to prevent auto-restart
	cp.explicitStop = true
	cp.mu.Unlock()

	// Signal goroutines to stop
	close(cp.done)

	// Close stdin to signal EOF to claude
	if cp.stdin != nil {
		cp.stdin.Close()
	}

	// Wait for process exit via exitChan (populated by monitorRestart)
	// Don't call cmd.Wait() here - monitorRestart already does that
	select {
	case err := <-cp.exitChan:
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
		return err
	case <-time.After(5 * time.Second):
		// Force kill after timeout
		if cp.cmd.Process != nil {
			cp.cmd.Process.Kill()
		}
		// Wait a bit for exitChan after kill
		select {
		case <-cp.exitChan:
		case <-time.After(100 * time.Millisecond):
		}
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
		return fmt.Errorf("process killed after timeout")
	}
}

// Send sends a text message to the claude process.
// Thread-safe - can be called from multiple goroutines.
// Returns error if write fails or process is not running.
func (cp *ClaudeProcess) Send(message string) error {
	return cp.SendJSON(UserMessage{Content: message})
}

// SendJSON sends a structured message to the claude process.
// Thread-safe - can be called from multiple goroutines.
// Returns error if JSON encoding or write fails.
func (cp *ClaudeProcess) SendJSON(msg UserMessage) error {
	cp.mu.Lock()
	running := cp.running
	cp.mu.Unlock()

	if !running {
		return fmt.Errorf("process not running")
	}

	return cp.writer.Write(msg)
}

// Events returns a receive-only channel for parsed events.
// Channel is buffered (size 100) to prevent blocking slow consumers.
// Channel is closed when the process stops.
func (cp *ClaudeProcess) Events() <-chan Event {
	return cp.events
}

// Errors returns a receive-only channel for errors.
// Includes both stderr output and event parsing errors.
// Channel is closed when the process stops.
func (cp *ClaudeProcess) Errors() <-chan error {
	return cp.errors
}

// RestartEvents returns a receive-only channel for restart events.
// Events are sent when the subprocess is being restarted or when
// max restart attempts are exceeded.
func (cp *ClaudeProcess) RestartEvents() <-chan RestartEvent {
	return cp.restartEvents
}

// IsRunning returns whether the subprocess is currently running.
// Thread-safe.
func (cp *ClaudeProcess) IsRunning() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.running
}

// SessionID returns the session ID for this process.
func (cp *ClaudeProcess) SessionID() string {
	return cp.sessionID
}

// readEvents reads NDJSON events from stdout in a loop.
// Runs in a background goroutine started by Start().
// Respects done channel for shutdown and closes event channel on exit.
func (cp *ClaudeProcess) readEvents() {
	defer close(cp.events)

	reader := NewNDJSONReader(cp.stdout)

	for {
		// Check for shutdown signal
		select {
		case <-cp.done:
			return
		default:
		}

		// Read next line with context
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		dataChan := make(chan []byte, 1)
		errChan := make(chan error, 1)

		go func() {
			data, err := reader.Read()
			if err != nil {
				errChan <- err
			} else {
				dataChan <- data
			}
		}()

		var data []byte
		var readErr error

		select {
		case <-ctx.Done():
			cancel()
			continue
		case readErr = <-errChan:
			cancel()
			if readErr != nil && readErr != io.EOF {
				cp.errors <- fmt.Errorf("read error: %w", readErr)
			}
			return
		case data = <-dataChan:
			cancel()
		}

		// Parse event
		event, err := parseEvent(data)
		if err != nil {
			cp.errors <- fmt.Errorf("parse error: %w", err)
			continue
		}

		// Send event (non-blocking due to buffer)
		select {
		case cp.events <- event:
		case <-cp.done:
			return
		}
	}
}

// readStderr reads stderr output and forwards to errors channel.
// Runs in a background goroutine started by Start().
func (cp *ClaudeProcess) readStderr() {
	defer close(cp.errors)

	reader := NewNDJSONReader(cp.stderr)
	for {
		select {
		case <-cp.done:
			return
		default:
		}

		data, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				cp.errors <- fmt.Errorf("stderr read error: %w", err)
			}
			return
		}

		// Forward stderr as error
		cp.errors <- fmt.Errorf("stderr: %s", string(data))
	}
}

// parseEvent parses raw JSON bytes into an Event struct.
// Uses the full event parsing implementation from events.go (GOgent-114).
func parseEvent(data []byte) (Event, error) {
	return ParseEvent(data)
}

// monitorRestart watches for process exit and handles auto-restart logic.
// Always runs in a background goroutine to centralize cmd.Wait() calls.
// This is the ONLY goroutine that should call cmd.Wait() to avoid data races.
func (cp *ClaudeProcess) monitorRestart() {
	// Wait for process to exit - this is the single Wait() call
	err := cp.cmd.Wait()

	// Send exit notification to exitChan for Stop() to consume if needed
	select {
	case cp.exitChan <- err:
	default:
		// Channel full or closed, continue
	}

	cp.mu.Lock()
	explicitStop := cp.explicitStop
	running := cp.running
	restartEnabled := cp.config.Restart.Enabled
	cp.mu.Unlock()

	// If this was an explicit stop or we're not running, don't restart
	if explicitStop || !running {
		cp.sendExitEvent(err, false)
		return
	}

	// If restart is disabled, just exit
	if !restartEnabled {
		cp.sendExitEvent(err, false)
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
		return
	}

	// Check if we should restart based on policy
	if !cp.restartState.ShouldRestart(cp.config.Restart) {
		// Max restarts exceeded
		cp.sendMaxRestartsEvent(err)
		return
	}

	// Calculate backoff delay
	delay := cp.restartState.NextDelay(cp.config.Restart)
	cp.restartState.RecordAttempt()

	// Send restart event
	cp.sendRestartEvent(err, delay)

	// Wait for backoff period
	select {
	case <-time.After(delay):
		// Continue with restart
	case <-cp.done:
		// Stop was called during backoff
		return
	}

	// Attempt restart
	if restartErr := cp.restart(); restartErr != nil {
		// Restart failed, send error
		select {
		case cp.errors <- fmt.Errorf("restart failed: %w", restartErr):
		default:
		}
	}
}

// restart recreates and starts the subprocess with the same or new session ID.
func (cp *ClaudeProcess) restart() error {
	cp.mu.Lock()

	// Determine session ID for restart
	sessionID := cp.sessionID
	if !cp.config.Restart.PreserveSession {
		sessionID = uuid.New().String()
	}

	// Update config with session ID
	cfg := cp.config
	cfg.SessionID = sessionID

	cp.mu.Unlock()

	// Create new command
	newProc, err := NewClaudeProcess(cfg)
	if err != nil {
		return fmt.Errorf("create new process: %w", err)
	}

	// Transfer restart state
	cp.mu.Lock()
	newProc.restartState = cp.restartState
	newProc.restartEvents = cp.restartEvents
	newProc.events = cp.events
	newProc.errors = cp.errors
	newProc.done = make(chan struct{}) // New done channel
	cp.mu.Unlock()

	// Start new process
	if err := newProc.startProcess(); err != nil {
		return fmt.Errorf("start new process: %w", err)
	}

	// Update self with new process state
	cp.mu.Lock()
	cp.cmd = newProc.cmd
	cp.stdin = newProc.stdin
	cp.stdout = newProc.stdout
	cp.stderr = newProc.stderr
	cp.writer = newProc.writer
	cp.sessionID = sessionID
	cp.done = newProc.done
	cp.running = true
	cp.mu.Unlock()

	return nil
}

// startProcess starts the subprocess without creating a new ClaudeProcess.
// Used internally by restart logic.
func (cp *ClaudeProcess) startProcess() error {
	// Set up pipes
	stdin, err := cp.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	cp.stdin = stdin
	cp.writer = NewNDJSONWriter(stdin)

	stdout, err := cp.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cp.stdout = stdout

	stderr, err := cp.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	cp.stderr = stderr

	// Start the process
	if err := cp.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude process: %w", err)
	}

	cp.running = true
	cp.restartState.RecordSuccess()

	// Start reading events in background
	go cp.readEvents()
	go cp.readStderr()

	// Always start restart monitor to handle process exit
	// It will honor the Enabled flag for restart behavior
	go cp.monitorRestart()

	return nil
}

// sendRestartEvent sends a RestartEvent indicating a restart is about to happen.
func (cp *ClaudeProcess) sendRestartEvent(err error, delay time.Duration) {
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	event := RestartEvent{
		Reason:     classifyExitReason(err),
		AttemptNum: cp.restartState.Attempts + 1,
		SessionID:  cp.sessionID,
		WillResume: cp.config.Restart.PreserveSession,
		NextDelay:  delay,
		Timestamp:  time.Now(),
		ExitCode:   exitCode,
	}

	select {
	case cp.restartEvents <- event:
	default:
		// Channel full, skip event
	}
}

// sendMaxRestartsEvent sends a RestartEvent indicating max restarts exceeded.
func (cp *ClaudeProcess) sendMaxRestartsEvent(err error) {
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	event := RestartEvent{
		Reason:     "max_restarts_exceeded",
		AttemptNum: cp.restartState.Attempts,
		SessionID:  cp.sessionID,
		WillResume: false,
		NextDelay:  0,
		Timestamp:  time.Now(),
		ExitCode:   exitCode,
	}

	select {
	case cp.restartEvents <- event:
	default:
	}

	// Mark as stopped
	cp.mu.Lock()
	cp.running = false
	cp.mu.Unlock()
}

// sendExitEvent sends a RestartEvent for explicit stop.
func (cp *ClaudeProcess) sendExitEvent(err error, willRestart bool) {
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	event := RestartEvent{
		Reason:     "explicit_stop",
		AttemptNum: cp.restartState.Attempts,
		SessionID:  cp.sessionID,
		WillResume: willRestart,
		NextDelay:  0,
		Timestamp:  time.Now(),
		ExitCode:   exitCode,
	}

	select {
	case cp.restartEvents <- event:
	default:
	}
}

// classifyExitReason determines the reason for process exit based on error.
func classifyExitReason(err error) string {
	if err == nil {
		return "normal_exit"
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == -1 {
			return "signal"
		}
		return "crash"
	}

	return "error"
}
