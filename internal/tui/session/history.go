package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// LoadConversationHistory
// ---------------------------------------------------------------------------

// LoadConversationHistory reads the persisted conversation history for the
// given session and provider.
//
// Returns nil, nil when the history file does not exist (new session or the
// history was previously cleared). Returns a non-nil error for I/O or JSON
// decode failures.
func (s *Store) LoadConversationHistory(sessionID string, provider state.ProviderID) ([]state.DisplayMessage, error) {
	if sessionID == "" {
		return nil, ErrEmptySessionID
	}

	path := s.historyFilePath(sessionID, provider)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load history session=%q provider=%q: read file: %w", sessionID, provider, err)
	}

	var msgs []state.DisplayMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("load history session=%q provider=%q: decode JSON: %w", sessionID, provider, err)
	}

	return msgs, nil
}

// ---------------------------------------------------------------------------
// SaveConversationHistory
// ---------------------------------------------------------------------------

// SaveConversationHistory writes the conversation history for the given session
// and provider to disk using an atomic temp-file-then-rename strategy.
//
// When messages is nil or empty the history file is removed instead; this
// prevents accumulation of empty files across sessions.
//
// Returns ErrEmptySessionID if sessionID is empty.
func (s *Store) SaveConversationHistory(sessionID string, provider state.ProviderID, messages []state.DisplayMessage) error {
	if sessionID == "" {
		return ErrEmptySessionID
	}

	path := s.historyFilePath(sessionID, provider)

	// Empty history — remove the file and return early.
	if len(messages) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("save history session=%q provider=%q: remove empty history: %w", sessionID, provider, err)
		}
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save history session=%q provider=%q: create dir: %w", sessionID, provider, err)
	}

	encoded, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("save history session=%q provider=%q: marshal JSON: %w", sessionID, provider, err)
	}

	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return fmt.Errorf("save history session=%q provider=%q: write tmp file: %w", sessionID, provider, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save history session=%q provider=%q: rename to final path: %w", sessionID, provider, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// historyFilePath returns the path to the conversation history file for the
// given session and provider: {baseDir}/{sessionID}/history-{provider}.json.
func (s *Store) historyFilePath(sessionID string, provider state.ProviderID) string {
	filename := "history-" + string(provider) + ".json"
	return filepath.Join(s.baseDir, sessionID, filename)
}
