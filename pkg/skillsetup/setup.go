package skillsetup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
	"github.com/google/uuid"
)

// SkillGuardConfig holds the guard configuration for a team-based skill,
// loaded from the skill_guards section of agents-index.json.
type SkillGuardConfig struct {
	RouterAllowedTools []string `json:"router_allowed_tools"`
	TeamDirSuffix      string   `json:"team_dir_suffix"`
}

// LoadSkillGuardConfig loads the skill guard config for skillName from
// agents-index.json. Returns nil, nil if the skill has no guard config
// (non-team skill like /dummies-guide).
func LoadSkillGuardConfig(skillName string) (*SkillGuardConfig, error) {
	r, err := resolve.NewFromEnv()
	if err != nil {
		return nil, fmt.Errorf("resolve config dir: %w", err)
	}
	results, err := r.ReadFileAll("agents/agents-index.json")
	if err != nil {
		return nil, fmt.Errorf("read agents-index.json: %w", err)
	}

	var data []byte
	if len(results) >= 2 {
		merged, mergeErr := resolve.MergeAgentIndexJSON(results[1], results[0])
		if mergeErr != nil {
			return nil, fmt.Errorf("merge agents-index.json: %w", mergeErr)
		}
		data = merged
	} else {
		data = results[0]
	}

	var index struct {
		SkillGuards map[string]*SkillGuardConfig `json:"skill_guards"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse agents-index.json: %w", err)
	}

	if index.SkillGuards == nil {
		return nil, nil
	}

	return index.SkillGuards[skillName], nil
}

// LoadSkillGuardConfigFrom loads the skill guard config from a specific
// agents-index.json path. Exported for testing with fixture files.
func LoadSkillGuardConfigFrom(skillName, indexPath string) (*SkillGuardConfig, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read agents-index.json: %w", err)
	}

	var index struct {
		SkillGuards map[string]*SkillGuardConfig `json:"skill_guards"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse agents-index.json: %w", err)
	}

	if index.SkillGuards == nil {
		return nil, nil
	}

	return index.SkillGuards[skillName], nil
}

// CreateTeamDir creates a team directory under sessionDir with the given suffix.
// Returns the absolute path to the created directory.
// Format: {sessionDir}/teams/{unix-timestamp}.{suffix}/
func CreateTeamDir(sessionDir, suffix string) (string, error) {
	timestamp := time.Now().Unix()
	teamDir := filepath.Join(sessionDir, "teams",
		fmt.Sprintf("%d.%s", timestamp, suffix))
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		return "", fmt.Errorf("create team dir %s: %w", teamDir, err)
	}
	return teamDir, nil
}

// WriteGuardFile writes the guard file for the given ActiveSkill to the
// XDG-scoped guard path determined by guard.SessionID.
func WriteGuardFile(guard *config.ActiveSkill) error {
	if guard.SessionID == "" {
		return fmt.Errorf("guard.SessionID is empty")
	}
	guardPath := config.GetGuardFilePath(guard.SessionID)
	data, err := json.MarshalIndent(guard, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal guard: %w", err)
	}
	if err := os.WriteFile(guardPath, data, 0644); err != nil {
		return fmt.Errorf("write guard to %s: %w", guardPath, err)
	}
	return nil
}

// RemoveGuardFiles removes both the guard JSON file and the lock file for
// the given session ID. Idempotent — ignores not-exist errors.
func RemoveGuardFiles(sessionID string) error {
	guardPath := config.GetGuardFilePath(sessionID)
	lockPath := config.GetGuardLockPath(sessionID)

	var firstErr error
	if err := os.Remove(guardPath); err != nil && !os.IsNotExist(err) {
		firstErr = fmt.Errorf("remove guard %s: %w", guardPath, err)
	}
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		if firstErr == nil {
			firstErr = fmt.Errorf("remove lock %s: %w", lockPath, err)
		}
	}
	return firstErr
}

// ResolveSessionDir determines the session directory from environment
// variables or the project's current-session marker file.
func ResolveSessionDir() string {
	if dir := os.Getenv("GOYOKE_SESSION_DIR"); dir != "" {
		return dir
	}

	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	if projectDir != "" {
		data, err := os.ReadFile(filepath.Join(config.RuntimeDir(projectDir), "current-session"))
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return filepath.Join(".goyoke", "sessions", "unknown")
}

// ResolveSessionID determines the session ID using a resolution chain:
//  1. GOYOKE_SESSION_ID env var (explicit, highest priority)
//  2. filepath.Base(sessionDir) if sessionDir is non-empty
//  3. Generate a new UUID as last resort
func ResolveSessionID(sessionDir string) string {
	if id := os.Getenv("GOYOKE_SESSION_ID"); id != "" {
		return id
	}
	if sessionDir != "" {
		base := filepath.Base(sessionDir)
		if base != "." && base != "/" && base != "unknown" {
			return base
		}
	}
	return uuid.New().String()
}
