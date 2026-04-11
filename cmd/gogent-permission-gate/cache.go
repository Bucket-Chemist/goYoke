package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// cachePath returns the file path for the session permission cache.
// Uses $XDG_RUNTIME_DIR when set, otherwise falls back to /tmp.
func cachePath(sessionID string) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	sum := sha256.Sum256([]byte(sessionID))
	return filepath.Join(dir, fmt.Sprintf("gofortress-perm-cache-%s.json", hex.EncodeToString(sum[:])))
}

// loadCache reads the cache file for sessionID.
// Returns an empty map when the file does not exist (first run).
func loadCache(sessionID string) (map[string]string, error) {
	path := cachePath(sessionID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		// Corrupt cache — start fresh rather than failing hard.
		return make(map[string]string), nil
	}
	return m, nil
}

// saveCache persists the cache map to disk with mode 0600.
func saveCache(sessionID string, m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	path := cachePath(sessionID)
	// WriteFile with restricted permissions — cache may contain tool names.
	return os.WriteFile(path, data, 0600)
}

// CheckCache looks up a previously stored decision for toolName in sessionID's
// cache.  Returns the cached decision and true when found, or ("", false) when
// absent.
func CheckCache(sessionID, toolName string) (string, bool) {
	m, err := loadCache(sessionID)
	if err != nil {
		return "", false
	}
	decision, ok := m[toolName]
	return decision, ok
}

// WriteCache persists an allow_session decision for toolName so that
// subsequent invocations within the same session are auto-allowed without
// re-prompting the user.
func WriteCache(sessionID, toolName, decision string) {
	m, err := loadCache(sessionID)
	if err != nil {
		m = make(map[string]string)
	}
	m[toolName] = decision
	// Best-effort — a write failure is non-fatal; the next invocation will
	// simply show the modal again.
	if err := saveCache(sessionID, m); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-permission-gate] Warning: failed to write cache: %v\n", err)
	}
}
