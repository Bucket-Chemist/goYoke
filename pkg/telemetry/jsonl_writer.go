package telemetry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/goYoke/pkg/filelock"
)

// AppendJSONL safely appends a line to a JSONL file with file locking
// to prevent corruption from concurrent writes.
func AppendJSONL(path string, data []byte) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Open file for append
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock (blocks until available)
	if err := filelock.Lock(int(f.Fd())); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer filelock.Unlock(int(f.Fd())) //nolint:errcheck

	// Write data with newline
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
