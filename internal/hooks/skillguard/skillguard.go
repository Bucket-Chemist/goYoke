// Package skillguard implements the goyoke-skill-guard PreToolUse hook.
package skillguard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

const (
	defaultTimeout = 5 * time.Second
	staleTTL       = 30 * time.Minute // Used by legacy fallback only
	guardFileName  = "active-skill.json"
)

// Main is the entrypoint for the goyoke-skill-guard hook.
func Main() {
	// CLI subcommand dispatch must be at the top, before any stdin reads.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--hold-lock":
			runHoldLock()
			return
		case "--setup":
			runSetup()
			return
		case "--release":
			runRelease()
			return
		}
	}

	event, err := routing.ParseToolEvent(os.Stdin, defaultTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: parse error:", err)
		fmt.Println("{}")
		return
	}

	output := handleGuardMode(event)
	fmt.Println(output)
}

// handleGuardMode checks the session-scoped guard and returns JSON output string.
func handleGuardMode(event *routing.ToolEvent) string {
	sessionID := event.SessionID

	if sessionID == "" {
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	guardPath := config.GetGuardFilePath(sessionID)
	data, err := os.ReadFile(guardPath)
	if err != nil {
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	var guard config.ActiveSkill
	if err := json.Unmarshal(data, &guard); err != nil {
		os.Remove(guardPath) //nolint:errcheck
		return "{}"
	}

	lockPath := config.GetGuardLockPath(sessionID)
	if isGuardStale(lockPath) {
		os.Remove(guardPath) //nolint:errcheck
		os.Remove(lockPath)  //nolint:errcheck
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	return checkAllowList(event.ToolName, &guard)
}

// legacyCheckGuard checks the old session-dir-scoped active-skill.json with 30-min TTL.
func legacyCheckGuard(toolName, guardPath string) string {
	data, err := os.ReadFile(guardPath)
	if err != nil {
		return "{}"
	}

	var guard config.ActiveSkill
	if err := json.Unmarshal(data, &guard); err != nil {
		os.Remove(guardPath) //nolint:errcheck
		return "{}"
	}

	if createdAt, err := time.Parse(time.RFC3339, guard.CreatedAt); err == nil {
		if time.Since(createdAt) > staleTTL {
			os.Remove(guardPath) //nolint:errcheck
			return "{}"
		}
	}

	return checkAllowList(toolName, &guard)
}

// checkAllowList checks toolName against guard's RouterAllowedTools.
func checkAllowList(toolName string, guard *config.ActiveSkill) string {
	if slices.Contains(guard.RouterAllowedTools, toolName) {
		return "{}"
	}

	resp := routing.NewBlockResponse("PreToolUse",
		fmt.Sprintf("[skill-guard] %s blocked during /%s skill. Router should dispatch via Task(), not use %s directly.",
			toolName, guard.Skill, toolName))

	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Error: marshal failed: %v\n", err)
		return "{}"
	}
	return buf.String()
}

func resolveSessionDir() string {
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
