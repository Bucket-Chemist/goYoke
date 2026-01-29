package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
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

	// NoHooks disables Claude Code hooks for testing.
	// When true, adds --no-hooks to CLI arguments.
	// Useful for testing TUI without GOgent hook interference.
	NoHooks bool

	// Restart defines auto-restart behavior for the subprocess.
	// If not set, DefaultRestartPolicy() is used.
	Restart RestartPolicy

	// Timeout configures event timeout behavior.
	// If not set, DefaultTimeoutConfig() is used.
	Timeout TimeoutConfig

	// SystemPrompt overrides the default system prompt.
	// If set, passed as --system-prompt flag.
	// Cannot be used with AppendPrompt.
	SystemPrompt string

	// AppendPrompt appends to the default system prompt.
	// If set, passed as --append-prompt flag.
	// Cannot be used with SystemPrompt.
	AppendPrompt string

	// AllowedTools is the whitelist of permitted tools.
	// If set, each tool is passed as --allowed-tools flag.
	// Supports patterns like "Bash(git *)".
	AllowedTools []string

	// DisallowedTools is the blacklist of forbidden tools.
	// If set, each tool is passed as --disallowed-tools flag.
	DisallowedTools []string

	// MaxTurns limits the number of agentic turns.
	// If > 0, passed as --max-turns flag.
	MaxTurns int

	// Model overrides the default model.
	// Accepts: "claude-3-opus", "claude-3-sonnet", "claude-3-haiku" or aliases "opus", "sonnet", "haiku".
	// If set, passed as --model flag.
	Model string
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
	closeOnce     sync.Once       // Ensures channels are closed only once
	readWg        *sync.WaitGroup // Coordinates pipe reads before Wait() (Fix 3)
	generation    uint64         // Process generation for restart isolation (Fix 5)
	genMu         sync.RWMutex   // Protects generation field
	stderrBuf     strings.Builder // Captures stderr for error classification
	stderrMu      sync.Mutex      // Protects stderrBuf
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

	// Use default timeout config if not configured.
	if cfg.Timeout.InactivityTimeout == 0 {
		cfg.Timeout = DefaultTimeoutConfig()
	}

	// Build command arguments
	// NOTE: --verbose is REQUIRED when using --print with --output-format=stream-json
	args := []string{
		"--print",
		"--verbose",           // Required for stream-json output
		"--debug-to-stderr",   // Keeps stdout clean for JSON (Fix 2)
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--session-id", sessionID,
	}

	// Add --no-hooks if configured (for testing without GOgent hooks)
	if cfg.NoHooks {
		args = append(args, "--no-hooks")
	}

	if cfg.IncludePartial {
		args = append(args, "--include-partial-messages")
	}

	if cfg.SettingsPath != "" {
		args = append(args, "--settings", cfg.SettingsPath)
	}

	// Add system prompt (mutually exclusive with append)
	if cfg.SystemPrompt != "" {
		args = append(args, "--system-prompt", cfg.SystemPrompt)
	} else if cfg.AppendPrompt != "" {
		args = append(args, "--append-prompt", cfg.AppendPrompt)
	}

	// Add tool restrictions
	for _, tool := range cfg.AllowedTools {
		args = append(args, "--allowed-tools", tool)
	}
	for _, tool := range cfg.DisallowedTools {
		args = append(args, "--disallowed-tools", tool)
	}

	// Add max turns limit
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns))
	}

	// Add model override
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
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
		readWg:        &sync.WaitGroup{},            // Pointer to avoid copy issues
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

	// Initialize stderr buffer
	cp.stderrBuf.Reset()

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

	// Start reading events in background with WaitGroup coordination (Fix 3)
	cp.readWg.Add(2) // For readEvents and readStderr
	go func() {
		defer cp.readWg.Done()
		cp.readEvents()
	}()
	go func() {
		defer cp.readWg.Done()
		cp.readStderr()
	}()

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
// Closes events and errors channels exactly once via sync.Once.
func (cp *ClaudeProcess) Stop() error {
	cp.mu.Lock()

	if !cp.running {
		cp.mu.Unlock()
		return nil
	}

	// Mark as explicit stop to prevent auto-restart
	cp.explicitStop = true
	cp.mu.Unlock()

	// Signal goroutines to stop - safe to call multiple times
	select {
	case <-cp.done:
		// Already closed
	default:
		close(cp.done)
	}

	// Close stdin to signal EOF to claude
	if cp.stdin != nil {
		cp.stdin.Close()
	}

	// Wait for process exit via exitChan (populated by monitorRestart)
	// Don't call cmd.Wait() here - monitorRestart already does that
	var exitErr error
	select {
	case exitErr = <-cp.exitChan:
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
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
		exitErr = fmt.Errorf("process killed after timeout")
	}

	// Wait for read goroutines to finish before closing channels
	// This prevents "send on closed channel" panics
	waitDone := make(chan struct{})
	go func() {
		cp.readWg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// Goroutines finished
	case <-time.After(1 * time.Second):
		// Timeout - continue anyway to prevent deadlock
	}

	// Close channels exactly once, after read goroutines are done
	cp.closeOnce.Do(func() {
		close(cp.events)
		close(cp.errors)
	})

	return exitErr
}

// Send sends a text message to the claude process.
// Thread-safe - can be called from multiple goroutines.
// Returns error if write fails or process is not running.
func (cp *ClaudeProcess) Send(message string) error {
	return cp.SendJSON(UserMessage{
		Type: "user",
		Message: UserContent{
			Role: "user",
			Content: []ContentBlock{
				{Type: "text", Text: message},
			},
		},
	})
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

// currentGeneration returns the current process generation.
// Used to detect stale goroutines after restart (Fix 5).
func (cp *ClaudeProcess) currentGeneration() uint64 {
	cp.genMu.RLock()
	defer cp.genMu.RUnlock()
	return cp.generation
}

// incrementGeneration increments and returns the new generation number.
// Called during restart to invalidate old goroutines (Fix 5).
func (cp *ClaudeProcess) incrementGeneration() uint64 {
	cp.genMu.Lock()
	defer cp.genMu.Unlock()
	cp.generation++
	return cp.generation
}

// readEvents reads NDJSON events from stdout in a loop.
// Runs in a background goroutine started by Start().
// Respects done channel for shutdown.
// NOTE: Does NOT close cp.events - channel closure is handled by Stop().
func (cp *ClaudeProcess) readEvents() {
	// Recover from send on closed channel panic (can happen during Stop race)
	defer func() {
		if r := recover(); r != nil {
			// Ignore "send on closed channel" panic during shutdown
			// This is expected when Stop() closes channels while goroutine is sending
		}
	}()

	// Capture generation at start to detect stale goroutines (Fix 5)
	myGen := cp.currentGeneration()
	reader := NewNDJSONReader(cp.stdout)

	// Single persistent reader goroutine with shared channels
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	readerDone := make(chan struct{})

	// Spawn ONE reader goroutine that persists for the entire lifecycle
	go func() {
		defer close(readerDone)
		for {
			data, err := reader.Read()
			if err != nil {
				select {
				case errChan <- err:
				case <-cp.done:
				}
				return
			}
			select {
			case dataChan <- data:
			case <-cp.done:
				return
			}
		}
	}()

	// Create inactivity timer
	inactivityTimer := time.NewTimer(cp.config.Timeout.InactivityTimeout)
	if cp.config.Timeout.InactivityTimeout == 0 {
		// Disable timer if no timeout configured
		if !inactivityTimer.Stop() {
			<-inactivityTimer.C
		}
	}
	defer inactivityTimer.Stop()

	for {
		// Check if this is a stale goroutine from previous generation (Fix 5)
		if cp.currentGeneration() != myGen {
			return
		}

		select {
		case <-cp.done:
			return

		case <-readerDone:
			// Reader goroutine exited (likely EOF or error already sent)
			return

		case readErr := <-errChan:
			if readErr != nil && readErr != io.EOF {
				// Only send error if still current generation
				if cp.currentGeneration() == myGen {
					select {
					case cp.errors <- fmt.Errorf("read error: %w", readErr):
					case <-cp.done:
					}
				}
			}
			return

		case data := <-dataChan:
			// Reset inactivity timer on successful read
			if cp.config.Timeout.InactivityTimeout > 0 {
				if !inactivityTimer.Stop() {
					select {
					case <-inactivityTimer.C:
					default:
					}
				}
				inactivityTimer.Reset(cp.config.Timeout.InactivityTimeout)
			}

			// Parse event
			event, err := parseEvent(data)
			if err != nil {
				// Only send error if still current generation
				if cp.currentGeneration() == myGen {
					select {
					case cp.errors <- fmt.Errorf("parse error: %w", err):
					case <-cp.done:
					}
				}
				continue
			}

			// Send event only if still current generation (non-blocking due to buffer)
			if cp.currentGeneration() != myGen {
				return
			}
			select {
			case cp.events <- event:
			case <-cp.done:
				return
			}

		case <-inactivityTimer.C:
			// Inactivity timeout fired
			if cp.config.Timeout.InactivityTimeout > 0 {
				if cp.currentGeneration() == myGen {
					select {
					case cp.errors <- fmt.Errorf("timeout: no events for %v", cp.config.Timeout.InactivityTimeout):
					case <-cp.done:
					}
				}
			}
			return
		}
	}
}

// readStderr reads stderr output and forwards to errors channel.
// Runs in a background goroutine started by Start().
// NOTE: Does NOT close cp.errors - channel closure is handled by Stop().
func (cp *ClaudeProcess) readStderr() {
	// Recover from send on closed channel panic (can happen during Stop race)
	defer func() {
		if r := recover(); r != nil {
			// Ignore "send on closed channel" panic during shutdown
			// This is expected when Stop() closes channels while goroutine is sending
		}
	}()

	// Capture generation at start to detect stale goroutines (Fix 5)
	myGen := cp.currentGeneration()
	reader := NewNDJSONReader(cp.stderr)

	// Single persistent reader goroutine with shared channels
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	readerDone := make(chan struct{})

	// Spawn ONE reader goroutine that persists for the entire lifecycle
	go func() {
		defer close(readerDone)
		for {
			data, err := reader.Read()
			if err != nil {
				select {
				case errChan <- err:
				case <-cp.done:
				}
				return
			}
			select {
			case dataChan <- data:
			case <-cp.done:
				return
			}
		}
	}()

	for {
		// Check if this is a stale goroutine from previous generation (Fix 5)
		if cp.currentGeneration() != myGen {
			return
		}

		select {
		case <-cp.done:
			return

		case <-readerDone:
			// Reader goroutine exited (likely EOF or error already sent)
			return

		case readErr := <-errChan:
			if readErr != nil && readErr != io.EOF && cp.currentGeneration() == myGen {
				select {
				case cp.errors <- fmt.Errorf("stderr read error: %w", readErr):
				case <-cp.done:
				}
			}
			return

		case data := <-dataChan:
			// Buffer stderr for error classification
			cp.stderrMu.Lock()
			cp.stderrBuf.Write(data)
			cp.stderrBuf.WriteByte('\n')
			cp.stderrMu.Unlock()

			// Parse and send ClaudeError only if still current generation
			if cp.currentGeneration() == myGen {
				claudeErr := ParseError(string(data), 0)
				select {
				case cp.errors <- claudeErr:
				case <-cp.done:
				}
			}
		}
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

	// Get exit code and stderr for error classification
	var exitCode int
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	cp.stderrMu.Lock()
	stderrContent := cp.stderrBuf.String()
	cp.stderrMu.Unlock()

	// Classify the error
	var claudeErr *ClaudeError
	if err != nil {
		claudeErr = ParseError(stderrContent, exitCode)
		claudeErr.Original = err
	}

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
		cp.sendExitEvent(claudeErr, false)
		return
	}

	// If restart is disabled, just exit
	if !restartEnabled {
		cp.sendExitEvent(claudeErr, false)
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
		return
	}

	// DON'T RESTART on authentication errors - they won't self-heal
	if claudeErr != nil && claudeErr.Type == ErrorAuthentication {
		cp.sendExitEvent(claudeErr, false)
		cp.mu.Lock()
		cp.running = false
		cp.mu.Unlock()
		return
	}

	// Check if we should restart based on policy
	if !cp.restartState.ShouldRestart(cp.config.Restart) {
		// Max restarts exceeded
		cp.sendMaxRestartsEvent(claudeErr)
		return
	}

	// Calculate backoff delay
	delay := cp.restartState.NextDelay(cp.config.Restart)
	cp.restartState.RecordAttempt()

	// Send restart event
	cp.sendRestartEvent(claudeErr, delay)

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
// Preserves the original events/errors channels so TUI consumers continue receiving
// events after restart without needing to re-subscribe.
func (cp *ClaudeProcess) restart() error {
	// Signal old goroutines to stop by closing done channel
	cp.mu.Lock()
	oldDone := cp.done
	cp.mu.Unlock()

	select {
	case <-oldDone:
		// Already closed
	default:
		close(oldDone)
	}

	// Wait for previous generation's goroutines to complete pipe reading.
	// Use timeout to prevent indefinite blocking.
	waitDone := make(chan struct{})
	go func() {
		cp.readWg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// Old goroutines finished draining pipes
	case <-time.After(5 * time.Second):
		// Timeout - log warning but continue with restart
		// This prevents indefinite blocking if goroutines hang
		select {
		case cp.errors <- fmt.Errorf("warning: restart timeout waiting for pipe readers, continuing anyway"):
		default:
			// Errors channel may be full, continue anyway
		}
	}

	// NOW safe to increment generation - old goroutines are done
	cp.incrementGeneration()

	// Clear stderr buffer for new process
	cp.stderrMu.Lock()
	cp.stderrBuf.Reset()
	cp.stderrMu.Unlock()

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

	// Create new command with fresh channels
	newProc, err := NewClaudeProcess(cfg)
	if err != nil {
		return fmt.Errorf("create new process: %w", err)
	}

	// Transfer restart state and restartEvents channel (not event/error channels)
	cp.mu.Lock()
	newProc.restartState = cp.restartState
	newProc.restartEvents = cp.restartEvents
	// CRITICAL: Transfer the ORIGINAL event/error channels so TUI consumers
	// still receive events after restart. The new reader goroutines will
	// write to these same channels. (Fix for channel replacement bug)
	newProc.events = cp.events
	newProc.errors = cp.errors
	newProc.done = make(chan struct{}) // New done channel
	newProc.readWg = &sync.WaitGroup{} // Fresh WaitGroup for new generation
	cp.mu.Unlock()

	// Start new process
	if err := newProc.startProcess(); err != nil {
		return fmt.Errorf("start new process: %w", err)
	}

	// Update self with new process state
	// CRITICAL: Do NOT replace events/errors channels - keep original channels
	// so TUI consumers continue receiving events after restart.
	cp.mu.Lock()
	cp.cmd = newProc.cmd
	cp.stdin = newProc.stdin
	cp.stdout = newProc.stdout
	cp.stderr = newProc.stderr
	cp.writer = newProc.writer
	cp.sessionID = sessionID
	cp.done = newProc.done
	cp.readWg = newProc.readWg        // Use fresh WaitGroup
	// events/errors channels are NOT replaced - newProc already uses cp's channels
	// closeOnce is NOT reset - channels should only be closed once at final Stop()
	cp.running = true
	cp.mu.Unlock()

	return nil
}

// startProcess starts the subprocess without creating a new ClaudeProcess.
// Used internally by restart logic.
func (cp *ClaudeProcess) startProcess() error {
	// Initialize stderr buffer
	cp.stderrBuf.Reset()

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

	// Start reading events in background with WaitGroup coordination
	cp.readWg.Add(2) // For readEvents and readStderr
	go func() {
		defer cp.readWg.Done()
		cp.readEvents()
	}()
	go func() {
		defer cp.readWg.Done()
		cp.readStderr()
	}()

	// Always start restart monitor to handle process exit
	// It will honor the Enabled flag for restart behavior
	go cp.monitorRestart()

	return nil
}

// sendRestartEvent sends a RestartEvent indicating a restart is about to happen.
func (cp *ClaudeProcess) sendRestartEvent(claudeErr *ClaudeError, delay time.Duration) {
	exitCode := 0
	reason := "unknown"

	if claudeErr != nil {
		exitCode = claudeErr.Code
		reason = claudeErr.Type.String()
	} else {
		reason = "normal_exit"
	}

	event := RestartEvent{
		Reason:     reason,
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
func (cp *ClaudeProcess) sendMaxRestartsEvent(claudeErr *ClaudeError) {
	exitCode := 0
	if claudeErr != nil {
		exitCode = claudeErr.Code
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
func (cp *ClaudeProcess) sendExitEvent(claudeErr *ClaudeError, willRestart bool) {
	exitCode := 0
	if claudeErr != nil {
		exitCode = claudeErr.Code
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
