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
}

// ClaudeProcess manages the lifecycle of a claude subprocess.
// It handles stdin/stdout communication via NDJSON streams,
// event reading in a background goroutine, and graceful shutdown.
type ClaudeProcess struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	writer    *NDJSONWriter
	sessionID string
	events    chan Event
	errors    chan error
	done      chan struct{}
	mu        sync.Mutex
	running   bool
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
		cmd:       cmd,
		sessionID: sessionID,
		events:    make(chan Event, 100), // Buffered to prevent blocking
		errors:    make(chan error, 10),  // Buffered for errors
		done:      make(chan struct{}),
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

	// Start reading events in background
	go cp.readEvents()
	go cp.readStderr()

	return nil
}

// Stop gracefully shuts down the claude process.
// Closes stdin to signal EOF, waits up to 5 seconds for clean exit,
// then sends SIGKILL if process hasn't terminated.
// Safe to call multiple times - returns nil if already stopped.
func (cp *ClaudeProcess) Stop() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.running {
		return nil
	}

	// Signal goroutines to stop
	close(cp.done)

	// Close stdin to signal EOF to claude
	if cp.stdin != nil {
		cp.stdin.Close()
	}

	// Wait for process with timeout
	done := make(chan error, 1)
	go func() {
		done <- cp.cmd.Wait()
	}()

	select {
	case err := <-done:
		cp.running = false
		return err
	case <-time.After(5 * time.Second):
		// Force kill after timeout
		if cp.cmd.Process != nil {
			cp.cmd.Process.Kill()
		}
		cp.running = false
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
