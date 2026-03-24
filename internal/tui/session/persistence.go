// Package session provides session persistence for the GOgent-Fortress TUI.
// It handles saving and loading session metadata, conversation history, and
// managing the session directory layout under ~/.claude/sessions/.
//
// All writes are atomic (temp file + rename) to prevent partial writes on
// crash or power loss. Missing files are handled gracefully (nil, nil return).
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrEmptySessionID is returned when an operation requires a non-empty session
// ID but an empty string was provided.
var ErrEmptySessionID = fmt.Errorf("session ID must not be empty")

// ---------------------------------------------------------------------------
// SessionData
// ---------------------------------------------------------------------------

// SessionData is the serializable snapshot of a TUI session.
// It is written to {baseDir}/{id}/session.json on every save.
type SessionData struct {
	// ID is the unique session identifier in YYYYMMDD.{UUID} format.
	ID string `json:"id"`
	// Name is an optional human-readable label for the session.
	Name string `json:"name,omitempty"`
	// CreatedAt is the time the session was first created.
	CreatedAt time.Time `json:"created_at"`
	// LastUsed is updated to time.Now() on every SaveSession call.
	LastUsed time.Time `json:"last_used"`
	// Cost is the cumulative USD cost for the session.
	Cost float64 `json:"cost"`
	// ToolCalls is the total number of tool calls made in the session.
	ToolCalls int `json:"tool_calls"`
	// ProviderSessionIDs holds the per-provider CLI session identifiers.
	ProviderSessionIDs map[state.ProviderID]string `json:"provider_session_ids,omitempty"`
	// ProviderModels holds the per-provider active model selections.
	ProviderModels map[state.ProviderID]string `json:"provider_models,omitempty"`
	// ActiveProvider is the provider that was active when the session was saved.
	ActiveProvider state.ProviderID `json:"active_provider"`
	// ThemeVariant is the active color theme at the time the session was saved.
	// Stored as int to avoid an import cycle (session → config would introduce
	// a cycle because config is a foundational package; session builds on top
	// of state, not config). The caller (model package) converts between
	// config.ThemeVariant and int at the save/restore boundary.
	// ThemeDark (0) is the default and is omitted from JSON by omitempty.
	ThemeVariant int `json:"theme_variant,omitempty"`
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// Store manages session files rooted at baseDir.
// Its zero value is not usable; use NewStore instead.
type Store struct {
	baseDir string
}

// NewStore creates a Store backed by baseDir.
// If baseDir is empty, DefaultBaseDir() is used.
func NewStore(baseDir string) *Store {
	if baseDir == "" {
		baseDir = DefaultBaseDir()
	}
	return &Store{baseDir: baseDir}
}

// DefaultBaseDir returns the canonical base directory for session persistence.
// If the CLAUDE_CONFIG_DIR environment variable is set, it returns
// $CLAUDE_CONFIG_DIR/sessions. Otherwise it falls back to $HOME/.claude/sessions.
// It panics if HOME is not set (should never happen on a fully initialised system).
func DefaultBaseDir() string {
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, "sessions")
	}
	home := os.Getenv("HOME")
	if home == "" {
		panic("session: HOME environment variable is not set")
	}
	return filepath.Join(home, ".claude", "sessions")
}

// NewSessionID generates a time-stamped unique session identifier with the
// format YYYYMMDD.{UUID}, e.g. "20260323.550e8400-e29b-41d4-a716-446655440000".
func NewSessionID() string {
	return time.Now().Format("20060102") + "." + uuid.New().String()
}

// ---------------------------------------------------------------------------
// LoadSession
// ---------------------------------------------------------------------------

// LoadSession reads the session metadata for id from disk.
// Returns nil, nil when the session file does not exist (first run or
// already cleaned up). Returns a non-nil error for I/O or JSON decode
// failures.
func (s *Store) LoadSession(id string) (*SessionData, error) {
	if id == "" {
		return nil, ErrEmptySessionID
	}

	path := s.sessionFilePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load session %q: read file: %w", id, err)
	}

	var sd SessionData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("load session %q: decode JSON: %w", id, err)
	}

	if sd.ID == "" {
		return nil, fmt.Errorf("load session %q: decoded session has empty ID", id)
	}

	return &sd, nil
}

// ---------------------------------------------------------------------------
// SaveSession
// ---------------------------------------------------------------------------

// SaveSession writes data to {baseDir}/{data.ID}/session.json using an atomic
// temp-file-then-rename strategy. It updates data.LastUsed to the current time
// before serialising.
//
// Returns ErrEmptySessionID if data.ID is empty.
func (s *Store) SaveSession(data *SessionData) error {
	if data.ID == "" {
		return ErrEmptySessionID
	}

	dir := filepath.Join(s.baseDir, data.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save session %q: create dir: %w", data.ID, err)
	}

	// Update LastUsed immediately before serialisation.
	data.LastUsed = time.Now()

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("save session %q: marshal JSON: %w", data.ID, err)
	}

	finalPath := s.sessionFilePath(data.ID)
	tmpPath := finalPath + ".tmp"

	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return fmt.Errorf("save session %q: write tmp file: %w", data.ID, err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		// Best-effort cleanup of the orphaned tmp file.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save session %q: rename to final path: %w", data.ID, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// SetupSessionDir
// ---------------------------------------------------------------------------

// SetupSessionDir creates the session directory for sessionID and configures
// two filesystem conveniences:
//
//  1. {claudeDir}/current-session — a plain text file containing the absolute
//     path to the session directory (overwrites any existing file).
//  2. {claudeDir}/tmp — a symbolic link pointing to the session directory
//     (removes and recreates any existing symlink or directory at that path).
//
// Returns the absolute path to the created session directory, or an error if
// any file-system operation fails.
func (s *Store) SetupSessionDir(sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrEmptySessionID
	}

	sessionDir := filepath.Join(s.baseDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", fmt.Errorf("setup session dir %q: create session dir: %w", sessionID, err)
	}

	// claudeDir is the parent of baseDir, e.g. ~/.claude when baseDir is
	// ~/.claude/sessions.
	claudeDir := filepath.Dir(s.baseDir)

	// Write current-session marker file.
	markerPath := filepath.Join(claudeDir, "current-session")
	if err := os.WriteFile(markerPath, []byte(sessionDir), 0o644); err != nil {
		return "", fmt.Errorf("setup session dir %q: write current-session marker: %w", sessionID, err)
	}

	// Create/update .claude/tmp symlink pointing to the session directory.
	tmpLink := filepath.Join(claudeDir, "tmp")

	// Remove any existing symlink or directory at the tmp path so we can
	// re-create it unconditionally.
	if err := os.Remove(tmpLink); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("setup session dir %q: remove existing tmp link: %w", sessionID, err)
	}

	if err := os.Symlink(sessionDir, tmpLink); err != nil {
		return "", fmt.Errorf("setup session dir %q: create tmp symlink: %w", sessionID, err)
	}

	return sessionDir, nil
}

// ---------------------------------------------------------------------------
// ListSessions
// ---------------------------------------------------------------------------

// ListSessions reads all session metadata files from the store's base directory
// and returns them sorted by LastUsed descending (most recent first).
// Returns an empty slice (not error) when no sessions exist.
func (s *Store) ListSessions() ([]*SessionData, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SessionData{}, nil
		}
		return nil, fmt.Errorf("list sessions: read dir %q: %w", s.baseDir, err)
	}

	var sessions []*SessionData
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name(), "session.json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Skip missing or unreadable files (corrupted sessions).
			continue
		}

		var sd SessionData
		if err := json.Unmarshal(data, &sd); err != nil {
			// Skip JSON decode failures (corrupted sessions).
			continue
		}

		if sd.ID == "" {
			continue
		}

		sessions = append(sessions, &sd)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastUsed.After(sessions[j].LastUsed)
	})

	return sessions, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// SessionDir returns the directory path for the given session ID:
// {baseDir}/{id}.
func (s *Store) SessionDir(id string) string {
	return filepath.Join(s.baseDir, id)
}

// sessionFilePath returns the full path to the session JSON file for id:
// {baseDir}/{id}/session.json.
func (s *Store) sessionFilePath(id string) string {
	return filepath.Join(s.baseDir, id, "session.json")
}
