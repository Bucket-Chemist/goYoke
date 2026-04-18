package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

const (
	defaultTimeout = 5 * time.Second
	staleTTL       = 30 * time.Minute // Used by legacy fallback only
	guardFileName  = "active-skill.json"
)

// tuiTranslation is injected as additionalContext when a Skill tool is invoked
// in TUI mode (GOYOKE_MCP_CONFIG set). It tells the model to translate
// Task() calls in SKILL.md instructions to the async spawn_agent + get_agent_result
// pattern. This keeps SKILL.md files dual-purpose: they work with native Claude Code
// (which has Task) and with the goYoke TUI (which replaces Task with MCP spawn_agent).
const tuiTranslation = `TOOL TRANSLATION (TUI mode active):

The Task() tool is NOT available in this session. Wherever the skill instructions say Task(), use this two-step pattern instead:

1. SPAWN: Call mcp__goyoke-interactive__spawn_agent with:
   - agent: the agent ID from the prompt's "AGENT: xxx" line
   - description: same as Task description
   - prompt: same as Task prompt
   - model: same as Task model (e.g. "opus", "sonnet", "haiku")

2. WAIT: Call mcp__goyoke-interactive__get_agent_result with:
   - agent_id: the agentId returned by spawn_agent
   - wait: true
   - timeout_ms: 600000

Key rules:
- subagent_type from Task() is NOT needed for spawn_agent.
- Task() was synchronous. spawn_agent is async — you MUST call get_agent_result to get the output.
- For parallel spawns: call spawn_agent multiple times, then call get_agent_result for each.
- Do NOT translate mcp__goyoke-interactive__team_run or any other mcp__goyoke-interactive__* calls. These already work natively. Only translate Task() calls.
- If no "AGENT: xxx" line exists in the Task prompt, infer the agent ID from context: the Task description, the agent name mentioned in surrounding text, or the model tier (haiku tasks → "haiku-scout").`

// ActiveSkill is a type alias for config.ActiveSkill.
// Kept for backward compatibility with test code; use config.ActiveSkill directly in new code.
type ActiveSkill = config.ActiveSkill

// SkillGuardConfig from agents-index.json skill_guards section.
type SkillGuardConfig struct {
	RouterAllowedTools []string `json:"router_allowed_tools"`
	TeamDirSuffix      string   `json:"team_dir_suffix"`
}

// isTUIMode returns true when the skill-guard is running inside the goYoke TUI.
// Detection: the TUI sets GOYOKE_MCP_CONFIG to the path of the temporary
// MCP config file it generates for the claude subprocess.
func isTUIMode() bool {
	return os.Getenv("GOYOKE_MCP_CONFIG") != ""
}

// emitSetupResponse prints the PreToolUse response for a Skill tool invocation.
// In TUI mode it injects the Task→spawn_agent translation as additionalContext.
// In native Claude Code mode it returns bare {} (no translation needed).
func emitSetupResponse() {
	if isTUIMode() {
		resp := map[string]string{"additionalContext": tuiTranslation}
		data, err := json.Marshal(resp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: failed to marshal translation response:", err)
			fmt.Println("{}")
			return
		}
		fmt.Println(string(data))
	} else {
		fmt.Println("{}")
	}
}

func main() {
	// --hold-lock dispatch must be at the top, before any stdin reads.
	if len(os.Args) > 1 && os.Args[1] == "--hold-lock" {
		runHoldLock()
		return
	}

	event, err := routing.ParseToolEvent(os.Stdin, defaultTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: parse error:", err)
		fmt.Println("{}")
		return
	}

	// ── SETUP MODE: Skill tool invocation ──
	if event.ToolName == "Skill" {
		handleSetupMode(event)
		return
	}

	// ── GUARD MODE: All other tools ──
	output := handleGuardMode(event)
	fmt.Println(output)
}

// handleSetupMode processes Skill tool invocations.
func handleSetupMode(event *routing.ToolEvent) {
	skillName := extractSkillName(event.ToolInput)
	if skillName == "" {
		fmt.Println("{}")
		return
	}
	guardConfig := loadSkillGuardConfig(skillName)

	sessionID := event.SessionID
	sessionDir := resolveSessionDir()

	if sessionID == "" {
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: empty session_id, falling back to legacy guard path")
		guardPath := filepath.Join(sessionDir, guardFileName)
		handleSetupModeWithConfig(skillName, guardConfig, sessionDir, guardPath)
		return
	}

	// Non-team skill: no guard needed (but still inject translation in TUI mode).
	if guardConfig == nil {
		emitSetupResponse()
		return
	}

	// Session-scoped guard paths (XDG-based, keyed by session ID).
	guardPath := config.GetGuardFilePath(sessionID)
	lockPath := config.GetGuardLockPath(sessionID)

	// Kill existing lock-holder if a guard file with a live PID exists.
	if data, err := os.ReadFile(guardPath); err == nil {
		var existing config.ActiveSkill
		if json.Unmarshal(data, &existing) == nil && existing.HolderPID > 0 {
			syscall.Kill(existing.HolderPID, syscall.SIGTERM) //nolint:errcheck
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Create team directory (still co-located with the project session dir).
	timestamp := time.Now().Unix()
	teamDir := filepath.Join(sessionDir, "teams",
		fmt.Sprintf("%d.%s", timestamp, guardConfig.TeamDirSuffix))
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to create team dir: %v\n", err)
		emitSetupResponse()
		return
	}

	// Get CC PID (our parent — the Claude Code process).
	ccPID := os.Getppid()
	if ccPID == 1 {
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: parent PID is 1 (init), may be running in container")
	}

	// Create pipe for lock-holder readiness signal.
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to create pipe: %v, continuing unguarded\n", err)
		writeSessionGuard(skillName, guardConfig, teamDir, sessionID, guardPath, 0, ccPID)
		emitSetupResponse()
		return
	}

	// Fork lock-holder process. ExtraFiles[0] (fd 3) is the write end of the readiness pipe.
	cmd := exec.Command(os.Args[0], "--hold-lock", lockPath, strconv.Itoa(ccPID), "3")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{writePipe}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to start lock-holder: %v, continuing unguarded\n", err)
		writePipe.Close()
		readPipe.Close()
		writeSessionGuard(skillName, guardConfig, teamDir, sessionID, guardPath, 0, ccPID)
		emitSetupResponse()
		return
	}
	writePipe.Close() // Parent closes the write end; lock-holder has its own copy.

	// Wait for readiness byte with 2-second timeout.
	ready := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 1)
		readPipe.Read(buf) //nolint:errcheck
		ready <- struct{}{}
	}()

	holderPID := cmd.Process.Pid
	select {
	case <-ready:
		// Lock holder signalled readiness.
	case <-time.After(2 * time.Second):
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: lock-holder readiness timeout, continuing unguarded")
		holderPID = 0
	}
	readPipe.Close()

	writeSessionGuard(skillName, guardConfig, teamDir, sessionID, guardPath, holderPID, ccPID)
	cmd.Process.Release() //nolint:errcheck
	emitSetupResponse()
}

// writeSessionGuard writes a v2 session-scoped guard file.
func writeSessionGuard(skillName string, guardConfig *SkillGuardConfig, teamDir, sessionID, guardPath string, holderPID, ccPID int) {
	guard := config.ActiveSkill{
		FormatVersion:      2,
		Skill:              skillName,
		TeamDir:            teamDir,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
		HolderPID:          holderPID,
		CCPID:              ccPID,
	}
	data, err := json.MarshalIndent(guard, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to marshal guard file: %v\n", err)
		return
	}
	if err := os.WriteFile(guardPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to write guard file: %v\n", err)
	}
}

// handleSetupModeWithConfig is the testable core for the legacy (no sessionID) path.
func handleSetupModeWithConfig(skillName string, guardConfig *SkillGuardConfig, sessionDir, guardPath string) {
	if guardConfig == nil {
		// Non-team skill (e.g., /dummies-guide), no guard needed.
		emitSetupResponse()
		return
	}

	// Create team directory.
	timestamp := time.Now().Unix()
	teamDir := filepath.Join(sessionDir, "teams",
		fmt.Sprintf("%d.%s", timestamp, guardConfig.TeamDirSuffix))
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to create team dir: %v\n", err)
		emitSetupResponse()
		return
	}

	// Write legacy guard file (FormatVersion 0 = omitted).
	guard := config.ActiveSkill{
		Skill:              skillName,
		TeamDir:            teamDir,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(guard, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to marshal guard file: %v\n", err)
		emitSetupResponse()
		return
	}
	if err := os.WriteFile(guardPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: failed to write guard file: %v\n", err)
	}

	emitSetupResponse()
}

// handleGuardMode checks the session-scoped guard and returns JSON output string.
func handleGuardMode(event *routing.ToolEvent) string {
	sessionID := event.SessionID

	if sessionID == "" {
		// TODO: Remove legacy fallback after v1.x release cycle
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	guardPath := config.GetGuardFilePath(sessionID)
	data, err := os.ReadFile(guardPath)
	if err != nil {
		// Guard file not found — check legacy fallback.
		// TODO: Remove legacy fallback after v1.x release cycle
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	var guard config.ActiveSkill
	if err := json.Unmarshal(data, &guard); err != nil {
		// Malformed guard file — delete and pass through.
		os.Remove(guardPath) //nolint:errcheck
		return "{}"
	}

	// Liveness check: if flock is stale, the lock-holder has exited.
	lockPath := config.GetGuardLockPath(sessionID)
	if isGuardStale(lockPath) {
		os.Remove(guardPath) //nolint:errcheck
		os.Remove(lockPath)  //nolint:errcheck
		// TODO: Remove legacy fallback after v1.x release cycle
		return legacyCheckGuard(event.ToolName, filepath.Join(resolveSessionDir(), guardFileName))
	}

	return checkAllowList(event.ToolName, &guard)
}

// legacyCheckGuard checks the old session-dir-scoped active-skill.json with 30-min TTL.
// TODO: Remove legacy fallback after v1.x release cycle
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

	// Staleness check (legacy path only — 30-minute TTL).
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
	for _, allowed := range guard.RouterAllowedTools {
		if toolName == allowed {
			return "{}"
		}
	}

	// Block — tool not in allowed list.
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

	// Try project dir resolution chain.
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

func extractSkillName(toolInput map[string]interface{}) string {
	if skill, ok := toolInput["skill"]; ok {
		if s, ok := skill.(string); ok {
			return s
		}
	}
	return ""
}

func loadSkillGuardConfig(skillName string) *SkillGuardConfig {
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		return nil
	}
	indexPath := filepath.Join(configDir, "agents", "agents-index.json")
	return loadSkillGuardConfigFrom(skillName, indexPath)
}

// loadSkillGuardConfigFrom loads the skill guard config from a specific index file path.
// This is the testable core function that allows tests to provide fixture paths.
func loadSkillGuardConfigFrom(skillName, indexPath string) *SkillGuardConfig {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}

	var index struct {
		SkillGuards map[string]*SkillGuardConfig `json:"skill_guards"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil
	}

	if index.SkillGuards == nil {
		return nil
	}

	return index.SkillGuards[skillName]
}
