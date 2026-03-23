// Package claude implements the Claude conversation panel for the
// GOgent-Fortress TUI.
package claude

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// historyFileName is the file name used for persisting input history.
	historyFileName = "inputhistory.json"

	// defaultMaxSize is the default maximum number of entries retained by
	// InputHistory when the caller does not provide an explicit value.
	defaultMaxSize = 500
)

// ---------------------------------------------------------------------------
// InputHistory
// ---------------------------------------------------------------------------

// InputHistory is a bounded list of previously submitted input strings that
// can be saved to and loaded from a JSON file for cross-session persistence.
//
// The zero value is not usable; use NewInputHistory instead.
type InputHistory struct {
	// Entries holds the history entries, oldest first.
	Entries []string `json:"entries"`
	// MaxSize is the maximum number of entries retained. When exceeded, the
	// oldest entries are removed from the front of the slice.
	MaxSize int `json:"max_size"`
}

// NewInputHistory allocates and returns an InputHistory with the specified
// maximum capacity.  If maxSize is ≤ 0, defaultMaxSize (500) is used.
func NewInputHistory(maxSize int) *InputHistory {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &InputHistory{
		Entries: nil,
		MaxSize: maxSize,
	}
}

// Add appends entry to the history.
//
// Consecutive duplicates are not added: if the last entry in the history is
// identical to entry, the call is a no-op.
//
// When the history exceeds MaxSize after the append, the oldest entries are
// trimmed from the front so that exactly MaxSize entries remain.
func (h *InputHistory) Add(entry string) {
	if entry == "" {
		return
	}
	// Skip consecutive duplicate.
	if len(h.Entries) > 0 && h.Entries[len(h.Entries)-1] == entry {
		return
	}
	h.Entries = append(h.Entries, entry)
	// Trim from the front when over capacity.
	if len(h.Entries) > h.MaxSize {
		excess := len(h.Entries) - h.MaxSize
		h.Entries = h.Entries[excess:]
	}
}

// All returns a copy of the current entries slice. Callers may safely modify
// the returned slice without affecting the history.
func (h *InputHistory) All() []string {
	if len(h.Entries) == 0 {
		return nil
	}
	out := make([]string, len(h.Entries))
	copy(out, h.Entries)
	return out
}

// Save atomically writes the history to {dir}/inputhistory.json.
//
// The write is atomic: data is written to a temporary file alongside the
// target, then renamed into place so that a crash mid-write does not corrupt
// an existing history file.
//
// Save creates the directory if it does not exist.
func (h *InputHistory) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("inputhistory: mkdir %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("inputhistory: marshal: %w", err)
	}

	target := filepath.Join(dir, historyFileName)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("inputhistory: write tmp: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		// Best-effort cleanup of the temp file.
		_ = os.Remove(tmp)
		return fmt.Errorf("inputhistory: rename to %q: %w", target, err)
	}

	return nil
}

// LoadInputHistory reads input history from {dir}/inputhistory.json.
//
// If the file does not exist, an empty history with defaultMaxSize is returned
// (no error).  If the file exists but contains invalid JSON, a warning is
// logged and an empty history is returned (no error), so that a corrupted
// history file does not prevent the TUI from starting.
func LoadInputHistory(dir string) (*InputHistory, error) {
	path := filepath.Join(dir, historyFileName)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewInputHistory(defaultMaxSize), nil
	}
	if err != nil {
		return NewInputHistory(defaultMaxSize), fmt.Errorf("inputhistory: read %q: %w", path, err)
	}

	var h InputHistory
	if err := json.Unmarshal(data, &h); err != nil {
		log.Printf("[inputhistory] warning: %q contains invalid JSON, starting fresh: %v", path, err)
		return NewInputHistory(defaultMaxSize), nil
	}

	if h.MaxSize <= 0 {
		h.MaxSize = defaultMaxSize
	}

	return &h, nil
}
