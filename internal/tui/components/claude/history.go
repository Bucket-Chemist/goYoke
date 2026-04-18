// Package claude implements the Claude conversation panel for the
// goYoke TUI.
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
	// historyFileName matches the TS TUI's file name so both TUIs share
	// the same cross-session prompt history.
	historyFileName = "input-history.json"

	// maxHistorySize matches the TS TUI's MAX_HISTORY_SIZE (100).
	maxHistorySize = 100
)

// ---------------------------------------------------------------------------
// InputHistory
// ---------------------------------------------------------------------------

// InputHistory is a bounded list of previously submitted input strings that
// can be saved to and loaded from a JSON file for cross-session persistence.
//
// Storage format is a plain JSON array of strings (newest first), compatible
// with the TS TUI's ~/.claude/input-history.json.
//
// The zero value is not usable; use NewInputHistory instead.
type InputHistory struct {
	// entries holds history items, newest first (index 0 = most recent).
	entries []string
	// dir is the directory containing the history file (e.g. ~/.claude).
	dir string
}

// NewInputHistory allocates and returns an empty InputHistory that will
// save to {dir}/input-history.json.
func NewInputHistory(dir string) *InputHistory {
	return &InputHistory{dir: dir}
}

// Add prepends entry to the history (newest first).
//
// Any-position deduplication: if entry already exists anywhere in the
// history, it is removed from its old position and re-inserted at the
// front. This matches the TS TUI's dedup behavior.
//
// When the history exceeds maxHistorySize, the oldest entries (at the
// end of the slice) are trimmed.
func (h *InputHistory) Add(entry string) {
	if entry == "" {
		return
	}
	// Remove any existing occurrence (any-position dedup).
	for i, e := range h.entries {
		if e == entry {
			h.entries = append(h.entries[:i], h.entries[i+1:]...)
			break
		}
	}
	// Prepend (newest first).
	h.entries = append([]string{entry}, h.entries...)
	// Trim oldest (from the end) when over capacity.
	if len(h.entries) > maxHistorySize {
		h.entries = h.entries[:maxHistorySize]
	}
}

// All returns a copy of entries, newest first.
func (h *InputHistory) All() []string {
	if len(h.entries) == 0 {
		return nil
	}
	out := make([]string, len(h.entries))
	copy(out, h.entries)
	return out
}

// Len returns the number of entries.
func (h *InputHistory) Len() int {
	return len(h.entries)
}

// Get returns the entry at index i (0 = newest). Returns "" if out of range.
func (h *InputHistory) Get(i int) string {
	if i < 0 || i >= len(h.entries) {
		return ""
	}
	return h.entries[i]
}

// Save atomically writes the history to {dir}/input-history.json as a plain
// JSON array (newest first), matching the TS TUI format.
func (h *InputHistory) Save() error {
	if h.dir == "" {
		return nil
	}
	if err := os.MkdirAll(h.dir, 0o700); err != nil {
		return fmt.Errorf("inputhistory: mkdir %q: %w", h.dir, err)
	}

	data, err := json.Marshal(h.entries)
	if err != nil {
		return fmt.Errorf("inputhistory: marshal: %w", err)
	}

	target := filepath.Join(h.dir, historyFileName)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("inputhistory: write tmp: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("inputhistory: rename to %q: %w", target, err)
	}

	return nil
}

// LoadInputHistory reads input history from {dir}/input-history.json.
//
// Supports two formats:
//  1. Plain JSON array (TS TUI format): ["newest", "older", ...]
//  2. Object with entries field (legacy Go format): {"entries": [...], "max_size": N}
//
// If the file does not exist, an empty history is returned (no error).
// If the file contains invalid JSON, a warning is logged and an empty
// history is returned so that a corrupted file does not prevent startup.
func LoadInputHistory(dir string) *InputHistory {
	h := NewInputHistory(dir)
	path := filepath.Join(dir, historyFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		// Also try the legacy filename for backward compat.
		legacyPath := filepath.Join(dir, "inputhistory.json")
		data, err = os.ReadFile(legacyPath)
		if err != nil {
			return h // no history file — start fresh
		}
	}

	// Try plain JSON array first (TS TUI format, newest-first).
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		h.entries = arr
		return h
	}

	// Fall back to legacy Go object format (oldest-first).
	var legacy struct {
		Entries []string `json:"entries"`
	}
	if err := json.Unmarshal(data, &legacy); err == nil && len(legacy.Entries) > 0 {
		// Reverse to newest-first.
		for i, j := 0, len(legacy.Entries)-1; i < j; i, j = i+1, j-1 {
			legacy.Entries[i], legacy.Entries[j] = legacy.Entries[j], legacy.Entries[i]
		}
		h.entries = legacy.Entries
		return h
	}

	log.Printf("[inputhistory] warning: %q contains invalid JSON, starting fresh", path)
	return h
}
