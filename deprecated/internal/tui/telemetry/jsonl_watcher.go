package telemetry

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// JSONLWatcher watches a single JSONL file and emits new lines as they're appended.
// Maintains file offset to efficiently read only new content on each write event.
type JSONLWatcher struct {
	path       string
	offset     int64
	mu         sync.Mutex
	file       *os.File
	watcher    *fsnotify.Watcher
	events     chan interface{}
	errors     chan error
	parseFunc  func([]byte) (interface{}, error)
	done       chan struct{}
	watcherDir string // Directory being watched (when file doesn't exist yet)
}

// NewJSONLWatcher creates a new JSONL file watcher with custom parse function.
// parseFunc is called for each new line to convert JSON to typed event.
func NewJSONLWatcher(path string, parseFunc func([]byte) (interface{}, error)) (*JSONLWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	jw := &JSONLWatcher{
		path:      path,
		watcher:   watcher,
		events:    make(chan interface{}, 100), // Buffered to handle bursts
		errors:    make(chan error, 10),
		parseFunc: parseFunc,
		done:      make(chan struct{}),
	}

	return jw, nil
}

// Start begins watching the file. If file doesn't exist yet, watches directory.
// Reads existing content and seeks to end, so only new events are reported.
func (jw *JSONLWatcher) Start() error {
	jw.mu.Lock()
	defer jw.mu.Unlock()

	// Try to open file
	file, err := os.Open(jw.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - watch directory for creation
			dir := filepath.Dir(jw.path)
			jw.watcherDir = dir
			if err := jw.watcher.Add(dir); err != nil {
				return fmt.Errorf("failed to watch directory %s: %w", dir, err)
			}
			jw.offset = 0
		} else {
			return fmt.Errorf("failed to open file %s: %w", jw.path, err)
		}
	} else {
		// File exists - seek to end (only watch new events)
		jw.file = file
		offset, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("failed to seek to end of file: %w", err)
		}
		jw.offset = offset

		// Watch the file itself
		if err := jw.watcher.Add(jw.path); err != nil {
			return fmt.Errorf("failed to watch file %s: %w", jw.path, err)
		}
	}

	// Start watch goroutine
	go jw.watch()

	return nil
}

// watch is the main event loop, handling fsnotify events
func (jw *JSONLWatcher) watch() {
	for {
		select {
		case event, ok := <-jw.watcher.Events:
			if !ok {
				return
			}

			// Check if this is the file we care about (or our watched directory)
			if event.Name != jw.path && event.Name != jw.watcherDir {
				continue
			}

			// Handle file creation (when we were watching directory)
			if event.Op&fsnotify.Create == fsnotify.Create && event.Name == jw.watcherDir {
				// Directory was created - check if our file appeared
				jw.handleFileCreation()
			}

			// Handle writes to our file
			if event.Name == jw.path && event.Op&fsnotify.Write == fsnotify.Write {
				if err := jw.readNewLines(); err != nil {
					select {
					case jw.errors <- fmt.Errorf("failed to read new lines: %w", err):
					case <-jw.done:
						return
					}
				}
			}

		case err, ok := <-jw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case jw.errors <- fmt.Errorf("fsnotify error: %w", err):
			case <-jw.done:
				return
			}

		case <-jw.done:
			return
		}
	}
}

// handleFileCreation checks if the file we're waiting for was created
func (jw *JSONLWatcher) handleFileCreation() {
	jw.mu.Lock()
	defer jw.mu.Unlock()

	if jw.file != nil {
		return // Already have file
	}

	// Check if file exists now
	if _, err := os.Stat(jw.path); err == nil {
		// File exists - open it and start watching
		file, err := os.Open(jw.path)
		if err != nil {
			return
		}
		jw.file = file
		jw.offset = 0

		// Switch from directory to file watching
		if jw.watcherDir != "" {
			jw.watcher.Remove(jw.watcherDir)
			jw.watcherDir = ""
		}
		jw.watcher.Add(jw.path)

		// Read any existing content
		jw.readNewLinesLocked()
	}
}

// readNewLines reads new lines from current offset to end of file
func (jw *JSONLWatcher) readNewLines() error {
	jw.mu.Lock()
	defer jw.mu.Unlock()
	return jw.readNewLinesLocked()
}

// readNewLinesLocked reads new lines (caller must hold lock)
func (jw *JSONLWatcher) readNewLinesLocked() error {
	if jw.file == nil {
		// Try to open file if it exists now
		file, err := os.Open(jw.path)
		if err != nil {
			return nil // File still doesn't exist
		}
		jw.file = file
	}

	// Check if file was truncated (offset > file size)
	fileInfo, err := jw.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if jw.offset > fileInfo.Size() {
		// File was truncated - reset to beginning
		jw.offset = 0
	}

	// Seek to last read position
	if _, err := jw.file.Seek(jw.offset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to offset %d: %w", jw.offset, err)
	}

	// Read new lines
	scanner := bufio.NewScanner(jw.file)
	linesRead := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse line with provided function
		if jw.parseFunc != nil {
			parsed, err := jw.parseFunc(line)
			if err != nil {
				// Log parse error but continue (don't break on malformed lines)
				select {
				case jw.errors <- fmt.Errorf("failed to parse line: %w", err):
				case <-jw.done:
					return nil
				default:
					// Error channel full, skip
				}
				continue
			}

			// Emit parsed event
			select {
			case jw.events <- parsed:
				linesRead++
			case <-jw.done:
				return nil
			default:
				// Event channel full - skip this event
				// (prevents blocking hook writes)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Update offset to current position
	newOffset, err := jw.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("failed to get current offset: %w", err)
	}
	jw.offset = newOffset

	return nil
}

// Events returns the read-only events channel
func (jw *JSONLWatcher) Events() <-chan interface{} {
	return jw.events
}

// Errors returns the read-only errors channel
func (jw *JSONLWatcher) Errors() <-chan error {
	return jw.errors
}

// Stop gracefully stops the watcher
func (jw *JSONLWatcher) Stop() error {
	close(jw.done)

	// Close watcher
	if err := jw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close fsnotify watcher: %w", err)
	}

	// Close file
	jw.mu.Lock()
	if jw.file != nil {
		if err := jw.file.Close(); err != nil {
			jw.mu.Unlock()
			return fmt.Errorf("failed to close file: %w", err)
		}
	}
	jw.mu.Unlock()

	// Close channels
	close(jw.events)
	close(jw.errors)

	return nil
}
