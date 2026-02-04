package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

// NDJSONReader reads newline-delimited JSON from an io.Reader.
// It uses a bufio.Scanner with a custom buffer to handle long lines
// that may occur in Claude's streaming responses.
type NDJSONReader struct {
	scanner *bufio.Scanner
}

// NewNDJSONReader creates an NDJSONReader with a large buffer for long lines.
// Default scanner buffer is 64KB, but Claude responses can exceed this.
// We set a 1MB buffer to prevent "token too long" errors.
func NewNDJSONReader(r io.Reader) *NDJSONReader {
	scanner := bufio.NewScanner(r)
	// Set large buffer for long JSON lines (1MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	return &NDJSONReader{
		scanner: scanner,
	}
}

// Read reads the next line and returns the raw JSON bytes.
// Returns io.EOF when the reader is exhausted.
// Malformed JSON is not checked here - parsing is caller's responsibility.
func (nr *NDJSONReader) Read() ([]byte, error) {
	if !nr.scanner.Scan() {
		if err := nr.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	// Return a copy to avoid scanner buffer reuse issues
	line := nr.scanner.Bytes()
	result := make([]byte, len(line))
	copy(result, line)
	return result, nil
}

// NDJSONWriter writes newline-delimited JSON to an io.Writer.
// All writes are protected by a mutex for thread-safe operation.
type NDJSONWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewNDJSONWriter creates an NDJSONWriter for the given writer.
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{
		writer: w,
	}
}

// Write marshals data to JSON and writes it with a trailing newline.
// Thread-safe for concurrent writes from multiple goroutines.
// Returns error if JSON marshaling or write fails.
func (nw *NDJSONWriter) Write(data interface{}) error {
	nw.mu.Lock()
	defer nw.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write JSON followed by newline
	jsonData = append(jsonData, '\n')
	_, err = nw.writer.Write(jsonData)
	return err
}

// Sync flushes the underlying writer if it supports flushing.
// Thread-safe for concurrent use.
// Returns nil if the writer doesn't support flushing.
func (nw *NDJSONWriter) Sync() error {
	nw.mu.Lock()
	defer nw.mu.Unlock()

	// Check for *bufio.Writer or similar with Flush() method
	type flusher interface {
		Flush() error
	}
	if f, ok := nw.writer.(flusher); ok {
		return f.Flush()
	}

	// Check for *os.File or similar with Sync() method
	type syncer interface {
		Sync() error
	}
	if s, ok := nw.writer.(syncer); ok {
		return s.Sync()
	}

	// Writer doesn't support flushing - no-op
	return nil
}
