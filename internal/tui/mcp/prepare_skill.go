package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/skillsetup"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// tuiTranslation is injected into the prepare_skill response when running in
// TUI mode (GOYOKE_MCP_CONFIG set). It tells the router to translate Task()
// calls in SKILL.md instructions to the async spawn_agent + get_agent_result
// pattern.
const skillTUITranslation = `TOOL TRANSLATION (TUI mode active):

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

// LockStore holds flock file descriptors across the MCP server's lifetime.
// In TUI mode, the TUI process is the natural long-lived flock holder — when
// it exits, the OS releases all flocks, making guards stale (correct behavior).
type LockStore struct {
	mu    sync.Mutex
	locks map[string]*os.File // sessionID -> held lock fd with LOCK_EX
}

// NewLockStore creates an empty LockStore.
func NewLockStore() *LockStore {
	return &LockStore{locks: make(map[string]*os.File)}
}

// Acquire obtains an exclusive flock on the guard lock file for sessionID.
// Idempotent — if a lock is already held for this session, it's a no-op.
func (ls *LockStore) Acquire(sessionID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if _, held := ls.locks[sessionID]; held {
		return nil
	}

	lockPath := config.GetGuardLockPath(sessionID)
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_EX); err != nil {
		fd.Close()
		return fmt.Errorf("acquire LOCK_EX: %w", err)
	}
	ls.locks[sessionID] = fd
	return nil
}

// Release releases the flock, closes the fd, and removes guard+lock files.
// Idempotent — safe to call even if no lock is held.
func (ls *LockStore) Release(sessionID string) {
	ls.mu.Lock()
	fd, held := ls.locks[sessionID]
	if held {
		delete(ls.locks, sessionID)
	}
	ls.mu.Unlock()

	if held && fd != nil {
		syscall.Flock(int(fd.Fd()), syscall.LOCK_UN) //nolint:errcheck
		fd.Close()
	}
	skillsetup.RemoveGuardFiles(sessionID) //nolint:errcheck
}

// CloseAll releases all held locks. Called during TUI shutdown for clean
// cleanup. The OS would release flocks on process exit regardless, but this
// also removes the guard/lock files from disk.
func (ls *LockStore) CloseAll() {
	ls.mu.Lock()
	ids := make([]string, 0, len(ls.locks))
	for id := range ls.locks {
		ids = append(ids, id)
	}
	ls.mu.Unlock()

	for _, id := range ids {
		ls.Release(id)
	}
}

// PrepareSkillInput is the input for the prepare_skill tool.
type PrepareSkillInput struct {
	Skill   string `json:"skill" jsonschema:"Skill name from agents-index.json skill_guards (e.g. braintrust, review)"`
	Release bool   `json:"release,omitempty" jsonschema:"Set true to tear down: release flock, remove guard files"`
}

// PrepareSkillOutput is the response from prepare_skill.
type PrepareSkillOutput struct {
	Skill              string   `json:"skill"`
	TeamDir            string   `json:"team_dir,omitempty"`
	GuardActive        bool     `json:"guard_active"`
	RouterAllowedTools []string `json:"router_allowed_tools,omitempty"`
	TUITranslation     string   `json:"tui_translation,omitempty"`
	Released           bool     `json:"released,omitempty"`
	Error              string   `json:"error,omitempty"`
}

// registerPrepareSkill registers the prepare_skill tool on the MCP server.
func registerPrepareSkill(server *mcpsdk.Server, uds *UDSClient, lockStore *LockStore) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "prepare_skill",
		Description: "Set up or tear down execution environment for a team-based skill. Call with release=false (default) at skill start to create team dir and guard. Call with release=true at skill end to clean up.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input PrepareSkillInput) (*mcpsdk.CallToolResult, PrepareSkillOutput, error) {
		return handlePrepareSkill(ctx, req, input, uds, lockStore)
	})
}

// handlePrepareSkill handles the prepare_skill tool call.
func handlePrepareSkill(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input PrepareSkillInput,
	uds *UDSClient,
	lockStore *LockStore,
) (*mcpsdk.CallToolResult, PrepareSkillOutput, error) {
	if input.Skill == "" {
		return nil, PrepareSkillOutput{}, fmt.Errorf("prepare_skill: skill is required")
	}

	sessionDir := os.Getenv("GOYOKE_SESSION_DIR")
	if sessionDir == "" {
		return nil, PrepareSkillOutput{
			Skill: input.Skill,
			Error: "GOYOKE_SESSION_DIR not set — skill setup requires a session directory",
		}, nil
	}
	sessionID := skillsetup.ResolveSessionID(sessionDir)

	if input.Release {
		return handlePrepareSkillRelease(input.Skill, sessionID, lockStore)
	}
	return handlePrepareSkillSetup(input.Skill, sessionID, sessionDir, uds, lockStore)
}

func handlePrepareSkillSetup(
	skill, sessionID, sessionDir string,
	uds *UDSClient,
	lockStore *LockStore,
) (*mcpsdk.CallToolResult, PrepareSkillOutput, error) {
	guardConfig, err := skillsetup.LoadSkillGuardConfig(skill)
	if err != nil {
		slog.Warn("prepare_skill: failed to load guard config", "skill", skill, "err", err)
	}

	tui := ""
	if os.Getenv("GOYOKE_MCP_CONFIG") != "" {
		tui = skillTUITranslation
	}

	if guardConfig == nil {
		return nil, PrepareSkillOutput{
			Skill:          skill,
			GuardActive:    false,
			TUITranslation: tui,
		}, nil
	}

	// Kill existing lock-holder if a guard with a live PID exists.
	if data, readErr := os.ReadFile(config.GetGuardFilePath(sessionID)); readErr == nil {
		var existing config.ActiveSkill
		if json.Unmarshal(data, &existing) == nil && existing.HolderPID > 0 {
			syscall.Kill(existing.HolderPID, syscall.SIGTERM) //nolint:errcheck
			time.Sleep(100 * time.Millisecond)
		}
	}

	teamDir, err := skillsetup.CreateTeamDir(sessionDir, guardConfig.TeamDirSuffix)
	if err != nil {
		slog.Warn("prepare_skill: failed to create team dir", "err", err)
		return nil, PrepareSkillOutput{
			Skill: skill,
			Error: fmt.Sprintf("failed to create team dir: %v", err),
		}, nil
	}

	holderPID := os.Getpid()
	if lockErr := lockStore.Acquire(sessionID); lockErr != nil {
		slog.Warn("prepare_skill: flock acquire failed, proceeding unguarded", "err", lockErr)
		holderPID = 0
	}

	guard := &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              skill,
		TeamDir:            teamDir,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
		HolderPID:          holderPID,
	}
	if writeErr := skillsetup.WriteGuardFile(guard); writeErr != nil {
		slog.Warn("prepare_skill: failed to write guard file", "err", writeErr)
	}

	ensureTeamVisible(teamDir)

	uds.notify(TypeToast, ToastPayload{
		Message: fmt.Sprintf("/%s skill activated", skill),
		Level:   "info",
	})

	return nil, PrepareSkillOutput{
		Skill:              skill,
		TeamDir:            teamDir,
		GuardActive:        true,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		TUITranslation:     tui,
	}, nil
}

func handlePrepareSkillRelease(
	skill, sessionID string,
	lockStore *LockStore,
) (*mcpsdk.CallToolResult, PrepareSkillOutput, error) {
	lockStore.Release(sessionID)
	return nil, PrepareSkillOutput{
		Skill:    skill,
		Released: true,
	}, nil
}
