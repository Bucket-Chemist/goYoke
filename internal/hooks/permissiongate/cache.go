package permissiongate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CachePath returns the file path for the session permission cache.
func CachePath(sessionID string) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	sum := sha256.Sum256([]byte(sessionID))
	return filepath.Join(dir, fmt.Sprintf("goyoke-perm-cache-%s.json", hex.EncodeToString(sum[:])))
}

func loadCache(sessionID string) (map[string]string, error) {
	path := CachePath(sessionID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]string), nil
	}
	return m, nil
}

func saveCache(sessionID string, m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	return os.WriteFile(CachePath(sessionID), data, 0600)
}

// CheckCache looks up a previously stored decision for toolName in sessionID's cache.
func CheckCache(sessionID, toolName string) (string, bool) {
	m, err := loadCache(sessionID)
	if err != nil {
		return "", false
	}
	decision, ok := m[toolName]
	return decision, ok
}

// WriteCache persists an allow_session decision for toolName.
func WriteCache(sessionID, toolName, decision string) {
	m, err := loadCache(sessionID)
	if err != nil {
		m = make(map[string]string)
	}
	m[toolName] = decision
	if err := saveCache(sessionID, m); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-permission-gate] Warning: failed to write cache: %v\n", err)
	}
}
