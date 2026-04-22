package archive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/process"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

// analyzeAndUpdateIntentOutcomes analyzes intent outcomes and updates user-intents.jsonl.
func analyzeAndUpdateIntentOutcomes(projectDir, sessionID string) error {
	q := session.NewQuery(projectDir)
	filters := session.UserIntentFilters{}
	if sessionID != "" {
		filters.SessionID = &sessionID
	}

	intents, err := q.QueryUserIntents(filters)
	if err != nil {
		return err
	}

	if len(intents) == 0 {
		return nil
	}

	actions := buildSessionActions(projectDir, sessionID)

	outcomes := session.AnalyzeIntentOutcomes(intents, actions)

	return updateIntentsWithOutcomes(projectDir, intents, outcomes)
}

// buildSessionActions constructs SessionActions from available session data.
func buildSessionActions(projectDir, sessionID string) session.SessionActions {
	actions := session.SessionActions{
		ToolsUsed:      []string{},
		ModelsUsed:     []string{},
		FilesEdited:    []string{},
		CommandsRun:    []string{},
		ActionsStopped: false,
	}

	handoffPath := filepath.Join(config.ProjectMemoryDir(projectDir), "handoffs.jsonl")
	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		return actions
	}

	for _, h := range handoffs {
		if h.SessionID == sessionID {
			if h.Context.Metrics.ToolCalls > 0 {
				actions.ToolsUsed = append(actions.ToolsUsed, "Edit", "Write", "Read")
			}

			if h.Context.GitInfo.IsDirty {
				actions.FilesEdited = h.Context.GitInfo.Uncommitted
			}

			break
		}
	}

	return actions
}

// updateIntentsWithOutcomes updates user-intents.jsonl with outcome analysis.
func updateIntentsWithOutcomes(projectDir string, analyzedIntents []session.UserIntent, outcomes []session.IntentOutcome) error {
	intentsPath := filepath.Join(config.ProjectMemoryDir(projectDir), "user-intents.jsonl")

	allIntents, err := session.LoadAllUserIntents(intentsPath)
	if err != nil {
		return fmt.Errorf("failed to load all intents: %w", err)
	}

	outcomeMap := make(map[int64]session.IntentOutcome)
	for i, intent := range analyzedIntents {
		if i < len(outcomes) {
			outcomeMap[intent.Timestamp] = outcomes[i]
		}
	}

	for i := range allIntents {
		if outcome, found := outcomeMap[allIntents[i].Timestamp]; found {
			allIntents[i].Honored = &outcome.Honored
			allIntents[i].OutcomeNote = outcome.Note
		}
	}

	file, err := os.Create(intentsPath)
	if err != nil {
		return fmt.Errorf("failed to open intents file for writing: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, intent := range allIntents {
		if err := encoder.Encode(intent); err != nil {
			return fmt.Errorf("failed to write intent: %w", err)
		}
	}

	return nil
}

// cleanupPermCache removes the permission gate session cache for the given session ID.
func cleanupPermCache(sessionID string) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}

	sum := sha256.Sum256([]byte(sessionID))
	path := filepath.Join(dir, fmt.Sprintf("goyoke-perm-cache-%s.json", hex.EncodeToString(sum[:])))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Warning: failed to remove permission cache %s: %v\n", path, err)
	}
}

// cleanupSkillGuard removes the skill guard files for the given session ID.
func cleanupSkillGuard(sessionID string) {
	guardPath := config.GetGuardFilePath(sessionID)

	data, err := os.ReadFile(guardPath)
	if err == nil && len(data) > 0 {
		var active config.ActiveSkill
		if jsonErr := json.Unmarshal(data, &active); jsonErr == nil && active.HolderPID > 0 {
			if killErr := process.Kill(active.HolderPID, syscall.SIGTERM); killErr != nil {
				fmt.Fprintf(os.Stderr, "[goyoke-archive] Warning: failed to SIGTERM skill guard holder PID %d: %v\n", active.HolderPID, killErr)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err := os.Remove(guardPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Warning: failed to remove skill guard file %s: %v\n", guardPath, err)
	}

	lockPath := config.GetGuardLockPath(sessionID)
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Warning: failed to remove skill guard lock %s: %v\n", lockPath, err)
	}

	fmt.Fprintf(os.Stderr, "[goyoke-archive] Cleaned up skill guard for session %s\n", sessionID)
}

// getProjectDir determines project directory from env or cwd.
func getProjectDir() string {
	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to get working directory: %v\n", err)
			fmt.Fprintln(os.Stderr, "  Set GOYOKE_PROJECT_DIR environment variable or run from project root.")
			os.Exit(1)
		}
		projectDir = cwd
	}
	return projectDir
}
