package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HookEvent represents the JSON structure of hook events from Claude Code.
type HookEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
	ToolResponse  map[string]interface{} `json:"tool_response,omitempty"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "corpus-logger: %v\n", err)
		// Non-zero exit would break hook chain - avoid for non-critical errors
	}
}

func run() error {
	// Read STDIN with timeout
	inputCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			errCh <- fmt.Errorf("reading stdin: %w", err)
			return
		}
		inputCh <- data
	}()

	var input []byte
	select {
	case input = <-inputCh:
		// Success
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("stdin read timeout after 5 seconds")
	}

	// Echo unchanged input to STDOUT immediately (pass-through for hook chain)
	if _, err := os.Stdout.Write(input); err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}

	// Parse and process the event
	if err := processEvent(input); err != nil {
		// Log error but don't fail the hook chain
		fmt.Fprintf(os.Stderr, "corpus-logger: failed to process event: %v\n", err)
		return nil // Don't propagate error to exit code
	}

	return nil
}

func processEvent(input []byte) error {
	// Skip empty input
	if len(input) == 0 {
		return nil
	}

	// Parse JSON
	var event HookEvent
	if err := json.Unmarshal(input, &event); err != nil {
		return fmt.Errorf("parsing json: %w", err)
	}

	// Add timestamp
	event.CapturedAt = time.Now().Unix()

	// Determine output path using XDG conventions
	outputPath, err := resolveOutputPath()
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Append to JSONL file
	if err := appendEvent(outputPath, &event); err != nil {
		return fmt.Errorf("appending event: %w", err)
	}

	return nil
}

func resolveOutputPath() (string, error) {
	// Try XDG_RUNTIME_DIR first (most specific)
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "gogent", "event-corpus-raw.jsonl"), nil
	}

	// Try XDG_CACHE_HOME
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "gogent", "event-corpus-raw.jsonl"), nil
	}

	// Fallback to ~/.cache/gogent
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	return filepath.Join(homeDir, ".cache", "gogent", "event-corpus-raw.jsonl"), nil
}

func appendEvent(path string, event *HookEvent) error {
	// Open file in append mode, create if doesn't exist
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", path, err)
	}
	defer f.Close()

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	// Write as single line with newline
	writer := bufio.NewWriter(f)
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}
	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flushing buffer: %w", err)
	}

	return nil
}
