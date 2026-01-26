package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Session represents a Claude session with its metadata
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	Cost      float64   `json:"cost"`
	ToolCalls int       `json:"tool_calls"`
}

// SessionManager handles session persistence and retrieval
type SessionManager struct {
	sessionsDir string
}

// NewSessionManager creates a SessionManager with the standard sessions directory.
// Creates ~/.claude/sessions/ if it doesn't exist.
func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir: %w", err)
	}

	sessionsDir := filepath.Join(homeDir, ".claude", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions directory: %w", err)
	}

	return &SessionManager{sessionsDir: sessionsDir}, nil
}

// ListSessions returns all sessions sorted by LastUsed descending (most recent first).
// Skips corrupt session files silently.
func (sm *SessionManager) ListSessions() ([]Session, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	var sessions []Session
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			session, err := sm.loadSession(entry.Name())
			if err != nil {
				// Skip corrupt files
				continue
			}
			sessions = append(sessions, session)
		}
	}

	// Sort by LastUsed descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastUsed.After(sessions[j].LastUsed)
	})

	return sessions, nil
}

// loadSession reads a session JSON file and returns the Session.
// Filename should be the basename, not the full path.
func (sm *SessionManager) loadSession(filename string) (Session, error) {
	path := filepath.Join(sm.sessionsDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("unmarshal session: %w", err)
	}

	return session, nil
}

// saveSession writes a Session to disk at ~/.claude/sessions/{id}.json
func (sm *SessionManager) saveSession(session Session) error {
	path := filepath.Join(sm.sessionsDir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}

// updateLastUsed updates the LastUsed timestamp for a session
func (sm *SessionManager) updateLastUsed(id string) error {
	session, err := sm.loadSession(id + ".json")
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	session.LastUsed = time.Now()
	if err := sm.saveSession(session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// DeleteSession removes a session file from disk
func (sm *SessionManager) DeleteSession(id string) error {
	path := filepath.Join(sm.sessionsDir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete session file: %w", err)
	}
	return nil
}

// ResumeSession creates a ClaudeProcess with an existing session ID.
// Updates the LastUsed timestamp and starts the process.
func (sm *SessionManager) ResumeSession(id string) (*ClaudeProcess, error) {
	cfg := Config{
		SessionID: id,
	}

	process, err := NewClaudeProcess(cfg)
	if err != nil {
		return nil, fmt.Errorf("create claude process: %w", err)
	}

	// Update last used time before starting
	if err := sm.updateLastUsed(id); err != nil {
		// Log but don't fail - this is metadata only
		// In a real implementation, might want to use a logger here
	}

	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("start claude process: %w", err)
	}

	return process, nil
}
