package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// conventionCache caches loaded convention content to avoid repeated disk I/O.
// Key: filename (e.g., "go.md"), Value: file content
var (
	conventionCache = make(map[string]string)
	cacheMutex      sync.RWMutex
)

// GetClaudeConfigDir returns the path to ~/.claude directory.
// Uses CLAUDE_CONFIG_DIR env var if set, otherwise ~/.claude
func GetClaudeConfigDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".claude"), nil
}

// LoadConventionContent loads a convention file by name.
// Returns the file content as a string.
// Results are cached for the duration of the process.
func LoadConventionContent(conventionName string) (string, error) {
	// Check cache first
	cacheMutex.RLock()
	if content, ok := conventionCache[conventionName]; ok {
		cacheMutex.RUnlock()
		return content, nil
	}
	cacheMutex.RUnlock()

	// Load from disk
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(configDir, "conventions", conventionName)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to load convention %s: %w", conventionName, err)
	}

	// Cache the result
	cacheMutex.Lock()
	conventionCache[conventionName] = string(content)
	cacheMutex.Unlock()

	return string(content), nil
}

// LoadRulesContent loads a rules file by name.
// Returns the file content as a string.
func LoadRulesContent(rulesName string) (string, error) {
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(configDir, "rules", rulesName)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to load rules %s: %w", rulesName, err)
	}

	return string(content), nil
}

// LoadMultipleConventions loads multiple convention files and returns them as a map.
// Continues loading remaining files even if one fails (logs warning).
func LoadMultipleConventions(conventionNames []string) (map[string]string, []error) {
	results := make(map[string]string)
	var errors []error

	for _, name := range conventionNames {
		content, err := LoadConventionContent(name)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		results[name] = content
	}

	return results, errors
}

// ClearConventionCache clears the convention cache.
// Useful for testing or when conventions might have changed.
func ClearConventionCache() {
	cacheMutex.Lock()
	conventionCache = make(map[string]string)
	cacheMutex.Unlock()
}
