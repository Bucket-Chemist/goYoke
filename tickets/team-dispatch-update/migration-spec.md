# .gogent/ Runtime I/O Migration Specification

## Overview

Migrate all runtime writes from `.claude/` to `.gogent/` at the CWD root. Introduce a standardised `InitWorkspace()` tool that any skill, agent, or orchestrator calls to set up workspace directories. Keep `.claude/` as the static, read-only configuration source.

---

## Architecture: Before and After

### Before (Broken)

```
{cwd}/.claude/                          ← CC sandbox: READ-ONLY
  ├── agents/                           ← Agent definitions (static)
  ├── schemas/                          ← Schema definitions (static)
  ├── skills/                           ← Skill definitions (static)
  ├── conventions/                      ← Coding conventions (static)
  ├── settings.json                     ← Settings (static)
  ├── routing-schema.json               ← Routing rules (static)
  ├── sessions/{id}/                    ← ⚠️ RUNTIME WRITES (BLOCKED)
  │   ├── session.json
  │   ├── active-skill.json
  │   └── teams/{ts}.braintrust/
  │       ├── config.json
  │       ├── stdin_*.json
  │       ├── stdout_*.json
  │       ├── runner.log
  │       └── ...
  ├── current-session                   ← ⚠️ RUNTIME WRITE (BLOCKED)
  ├── tmp → sessions/{id}/              ← ⚠️ RUNTIME WRITE (BLOCKED)
  └── braintrust/                       ← ⚠️ RUNTIME WRITES (BLOCKED)
```

### After (Fixed)

```
{cwd}/.claude/                          ← STATIC, READ-ONLY (unchanged)
  ├── agents/
  ├── schemas/
  ├── skills/
  ├── conventions/
  ├── settings.json
  └── routing-schema.json

{cwd}/.gogent/                          ← RUNTIME, READ-WRITE (new)
  ├── current-session                   ← Marker file (session dir path)
  ├── sessions/{id}/                    ← Per-session runtime state
  │   ├── session.json                  ← Session metadata
  │   ├── active-skill.json             ← Skill guard state
  │   └── teams/                        ← Team execution directories
  │       ├── {ts}.braintrust/
  │       │   ├── config.json
  │       │   ├── stdin_*.json
  │       │   ├── stdout_*.json
  │       │   ├── stream_*.ndjson
  │       │   ├── pre-synthesis.md
  │       │   ├── runner.log
  │       │   ├── heartbeat.json
  │       │   └── team-run.pid
  │       ├── {ts}.code-review/
  │       └── {ts}.implementation/
  ├── tmp → sessions/{id}/              ← Symlink to active session
  ├── braintrust/                       ← Braintrust analysis outputs
  └── active-schemas/                   ← Mirrored schemas (optional)
```

---

## The Standardised Workspace Init Tool

### Purpose

Replace the scattered `mkdir -p` + `os.WriteFile` patterns across skill-guard, SKILL.md instructions, and agent prompts with a single, flexible tool.

### Interface

```go
// Package workspace provides standardised directory setup for GOgent-Fortress
// runtime operations. All runtime writes go to .gogent/ at the project root.
package workspace

// Config holds the resolved paths for a workspace.
type Config struct {
    // RuntimeRoot is the .gogent/ directory at project root.
    RuntimeRoot string

    // SessionDir is the active session directory.
    // e.g., .gogent/sessions/20260331.abc123/
    SessionDir string

    // TeamDir is the team execution directory (nil if not a team workflow).
    // e.g., .gogent/sessions/20260331.abc123/teams/1774939872.braintrust/
    TeamDir string

    // SchemaDir is where active (writable) schema copies live.
    // e.g., .gogent/active-schemas/teams/
    SchemaDir string

    // TmpDir is the convenience symlink target.
    // e.g., .gogent/tmp → .gogent/sessions/20260331.abc123/
    TmpDir string
}

// InitOptions configures workspace initialization.
type InitOptions struct {
    // ProjectRoot is the project directory. Required.
    // Resolved from: GOGENT_PROJECT_ROOT → GOGENT_PROJECT_DIR → cwd
    ProjectRoot string

    // SessionID is the CC session ID. If empty, generates timestamp-based ID.
    SessionID string

    // Skill is the skill being invoked (e.g., "braintrust", "review", "implement").
    // Empty if not a skill-driven workflow.
    Skill string

    // Agent is the agent requesting workspace init (e.g., "mozart", "router").
    Agent string

    // Level is the nesting level (0=router, 1=sub-agent, 2=team-spawned).
    Level int

    // TeamDirSuffix overrides the default team directory suffix.
    // Default: derived from Skill (e.g., "braintrust", "code-review").
    TeamDirSuffix string

    // MirrorSchemas controls whether .claude/schemas/ are copied to .gogent/.
    // Default: false (schemas are read directly from .claude/).
    MirrorSchemas bool
}

// Init creates the workspace directory structure and returns resolved paths.
// This is the SINGLE ENTRY POINT for all runtime directory creation.
//
// What it does:
//   1. Creates .gogent/ at ProjectRoot if it doesn't exist
//   2. Creates .gogent/sessions/{SessionID}/ if it doesn't exist
//   3. Writes .gogent/current-session marker file
//   4. Creates .gogent/tmp symlink → active session dir
//   5. If Skill is set: creates .gogent/sessions/{id}/teams/{ts}.{suffix}/
//   6. If MirrorSchemas: copies .claude/schemas/teams/ → .gogent/active-schemas/
//
// Thread safety: safe for concurrent calls (uses os.MkdirAll, atomic writes).
func Init(opts InitOptions) (*Config, error)
```

### CLI Binary Wrapper

For use from Bash (skill instructions, hooks):

```bash
# Create workspace for braintrust skill
gogent-workspace-init --skill=braintrust --agent=router --level=0

# Output (JSON to stdout):
{
  "runtime_root": "/home/doktersmol/Documents/GOgent-Fortress/.gogent",
  "session_dir": "/home/doktersmol/Documents/GOgent-Fortress/.gogent/sessions/20260331.abc123",
  "team_dir": "/home/doktersmol/Documents/GOgent-Fortress/.gogent/sessions/20260331.abc123/teams/1774939872.braintrust",
  "schema_dir": "/home/doktersmol/Documents/GOgent-Fortress/.gogent/active-schemas/teams",
  "tmp_dir": "/home/doktersmol/Documents/GOgent-Fortress/.gogent/tmp"
}
```

### Usage Patterns

**From a hook (gogent-skill-guard):**
```go
import "github.com/Bucket-Chemist/GOgent-Fortress/pkg/workspace"

cfg, err := workspace.Init(workspace.InitOptions{
    ProjectRoot: resolveProjectRoot(),
    SessionID:   resolveSessionID(),
    Skill:       skillName,
    Agent:       "skill-guard",
    Level:       0,
})
// cfg.TeamDir is the created team directory
```

**From a skill instruction (SKILL.md):**
```bash
# Step 1: Initialize workspace
workspace=$(gogent-workspace-init --skill=braintrust --agent=router)
team_dir=$(echo "$workspace" | jq -r '.team_dir')

# Step 2: Mozart writes config + stdin files to team_dir
# Step 3: Launch team-run
gogent-team-run "$team_dir"
```

**From gogent-load-context hook:**
```go
cfg, err := workspace.Init(workspace.InitOptions{
    ProjectRoot: projectDir,
    SessionID:   event.SessionID,
    // No Skill — just session setup
})
// cfg.SessionDir is the created session directory
```

---

## File-by-File Migration Plan

### Phase 1: Core Path Resolution (2 files)

These are the foundation — everything else depends on them.

#### 1a. `pkg/session/session_dir.go`

**Current:**
```go
func CreateSessionDir(projectDir, sessionID string) (string, error) {
    sessionDir := filepath.Join(projectDir, ".claude", "sessions", sessionID)
    if err := os.MkdirAll(sessionDir, 0755); err != nil {
        return "", fmt.Errorf("create session dir: %w", err)
    }
    return sessionDir, nil
}

func WriteCurrentSession(projectDir, sessionDir string) error {
    claudeDir := filepath.Join(projectDir, ".claude")
    currentSessionPath := filepath.Join(claudeDir, "current-session")
    // ...
}

func ReadCurrentSession(projectDir string) (string, error) {
    currentSessionPath := filepath.Join(projectDir, ".claude", "current-session")
    // ...
}
```

**Change to:**
```go
const runtimeDirName = ".gogent"

func RuntimeDir(projectDir string) string {
    return filepath.Join(projectDir, runtimeDirName)
}

func CreateSessionDir(projectDir, sessionID string) (string, error) {
    sessionDir := filepath.Join(RuntimeDir(projectDir), "sessions", sessionID)
    if err := os.MkdirAll(sessionDir, 0755); err != nil {
        return "", fmt.Errorf("create session dir: %w", err)
    }
    return sessionDir, nil
}

func WriteCurrentSession(projectDir, sessionDir string) error {
    runtimeRoot := RuntimeDir(projectDir)
    if err := os.MkdirAll(runtimeRoot, 0755); err != nil {
        return fmt.Errorf("create runtime dir: %w", err)
    }
    currentSessionPath := filepath.Join(runtimeRoot, "current-session")
    // ...
}

func ReadCurrentSession(projectDir string) (string, error) {
    // Try .gogent/ first, fall back to .claude/ for backward compat
    currentSessionPath := filepath.Join(RuntimeDir(projectDir), "current-session")
    content, err := os.ReadFile(currentSessionPath)
    if err != nil && os.IsNotExist(err) {
        // Fallback: try legacy .claude/ path
        currentSessionPath = filepath.Join(projectDir, ".claude", "current-session")
        content, err = os.ReadFile(currentSessionPath)
    }
    // ...
}
```

**Test updates:** `pkg/session/session_dir_test.go` — change fixture paths from `.claude/sessions/` to `.gogent/sessions/`.

#### 1b. `internal/tui/session/persistence.go`

**Current:**
```go
func DefaultBaseDir() string {
    if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
        return filepath.Join(configDir, "sessions")
    }
    home := os.Getenv("HOME")
    return filepath.Join(home, ".claude", "sessions")
}
```

**Change to:**
```go
func DefaultBaseDir() string {
    // Check for explicit runtime dir override
    if runtimeDir := os.Getenv("GOGENT_RUNTIME_DIR"); runtimeDir != "" {
        return filepath.Join(runtimeDir, "sessions")
    }
    // Project-local .gogent/ if GOGENT_PROJECT_ROOT is set
    if projectRoot := os.Getenv("GOGENT_PROJECT_ROOT"); projectRoot != "" {
        return filepath.Join(projectRoot, ".gogent", "sessions")
    }
    // Fall back to home directory
    home := os.Getenv("HOME")
    return filepath.Join(home, ".gogent", "sessions")
}
```

**SetupSessionDir** — change `claudeDir` references:
```go
func (s *Store) SetupSessionDir(sessionID string) (string, error) {
    // ...
    // runtimeDir is the parent of baseDir, e.g. ~/.gogent when baseDir is ~/.gogent/sessions
    runtimeDir := filepath.Dir(s.baseDir)

    // Write current-session marker file
    markerPath := filepath.Join(runtimeDir, "current-session")
    // ...

    // Create/update .gogent/tmp symlink
    tmpLink := filepath.Join(runtimeDir, "tmp")
    // ...
}
```

**Test updates:** `internal/tui/session/persistence_test.go` — change all `.claude/sessions/` references.

### Phase 2: Hook Updates (2 files)

#### 2a. `cmd/gogent-load-context/main.go`

**Change:** The hook calls `session.CreateSessionDir()` which now writes to `.gogent/`.

No direct code change needed in this file if Phase 1 is complete — it delegates to `pkg/session`.

**Optional addition:** Add schema mirroring:
```go
// After session dir creation
if sessionDir != "" {
    // Mirror schemas for agent use (optional)
    srcSchemas := filepath.Join(configDir, "schemas", "teams")
    dstSchemas := filepath.Join(session.RuntimeDir(projectDir), "active-schemas", "teams")
    if _, err := os.Stat(srcSchemas); err == nil {
        copyDir(srcSchemas, dstSchemas)
    }
}
```

#### 2b. `cmd/gogent-skill-guard/main.go`

**Current:**
```go
func resolveSessionDir() string {
    if dir := os.Getenv("GOGENT_SESSION_DIR"); dir != "" {
        return dir
    }
    // ...
    data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "current-session"))
    // ...
    return filepath.Join(".claude", "sessions", "unknown")
}
```

**Change to:**
```go
func resolveSessionDir() string {
    if dir := os.Getenv("GOGENT_SESSION_DIR"); dir != "" {
        return dir
    }
    projectDir := resolveProjectDir()
    if projectDir != "" {
        // Try .gogent/ first
        data, err := os.ReadFile(filepath.Join(projectDir, ".gogent", "current-session"))
        if err == nil {
            return strings.TrimSpace(string(data))
        }
        // Fallback to .claude/ for backward compat
        data, err = os.ReadFile(filepath.Join(projectDir, ".claude", "current-session"))
        if err == nil {
            return strings.TrimSpace(string(data))
        }
    }
    return filepath.Join(".gogent", "sessions", "unknown")
}
```

**OR** replace the entire function with a call to `workspace.Init()`:
```go
cfg, err := workspace.Init(workspace.InitOptions{
    ProjectRoot:   resolveProjectDir(),
    SessionID:     "", // will be resolved from current-session
    Skill:         skillName,
    TeamDirSuffix: guardConfig.TeamDirSuffix,
})
if err != nil {
    // fallback
}
teamDir := cfg.TeamDir
```

### Phase 3: Workspace Init Tool (new files)

#### 3a. `pkg/workspace/init.go` (NEW)

The standardised workspace init function as described in the Interface section above.

#### 3b. `cmd/gogent-workspace-init/main.go` (NEW)

CLI wrapper that calls `workspace.Init()` and outputs JSON to stdout.

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/workspace"
)

func main() {
    skill := flag.String("skill", "", "Skill being invoked")
    agent := flag.String("agent", "router", "Agent requesting workspace")
    level := flag.Int("level", 0, "Nesting level")
    flag.Parse()

    cfg, err := workspace.Init(workspace.InitOptions{
        ProjectRoot: resolveProjectRoot(),
        SessionID:   resolveSessionID(),
        Skill:       *skill,
        Agent:       *agent,
        Level:       *level,
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
        os.Exit(1)
    }

    json.NewEncoder(os.Stdout).Encode(cfg)
}
```

### Phase 4: Skill Instruction Updates (5 files)

All SKILL.md files that reference `.claude/sessions/`, `SESSION_DIR`, or `active-skill.json` paths.

#### 4a. `.claude/skills/braintrust/SKILL.md`

**Changes:**
- Line 102: `.claude/braintrust/analysis-{timestamp}.md` → `.gogent/braintrust/analysis-{timestamp}.md`
- Line 169: `{session_dir}/teams/{timestamp}.braintrust/` → now created by `workspace.Init()` in `.gogent/`
- Line 178: `Read({ file_path: \`${session_dir}/active-skill.json\` })` — session_dir now resolves to `.gogent/`
- Line 323: `{team_dir} = {session_dir}/teams/...` — update path documentation
- Line 387: `output_location: ".claude/braintrust/"` → `output_location: ".gogent/braintrust/"`

#### 4b. `.claude/skills/review/SKILL.md`

**Changes:**
- Line 53: `{session_dir}/teams/{timestamp}.code-review/` — session_dir now under `.gogent/`
- Line 64-68: `Read({ file_path: \`${session_dir}/active-skill.json\` })` — path updated
- Line 336: Telemetry path `.claude/tmp/review-telemetry.json` → `.gogent/tmp/review-telemetry.json`
- Line 496: `{team_dir}` path documentation update

#### 4c. `.claude/skills/implement/SKILL.md`

**Changes:**
- Line 34: `{session_dir}/teams/{timestamp}.implementation/` — now under `.gogent/`
- Line 68-69: active-skill.json path
- Line 124-125: `session_dir` resolution — now checks `.gogent/current-session` first
- Line 213: Example path update

#### 4d. `.claude/skills/team-status/SKILL.md`

**Changes:** Team dir path references.

#### 4e. `.claude/skills/team-result/SKILL.md`

**Changes:** Team dir path references.

### Phase 5: Agent Instruction Updates (3 files)

#### 5a. `.claude/agents/mozart/mozart.md`

**Changes:** All references to writing config.json and stdin files to session_dir/teams/ paths. Mozart should use `gogent-workspace-init` or write directly to the `.gogent/` path provided in its prompt.

#### 5b. `.claude/agents/architect/architect.md`

**Changes:** `SESSION_DIR/implementation-plan.json` and `SESSION_DIR/specs.md` — these now resolve to `.gogent/sessions/{id}/`.

#### 5c. `.claude/agents/planner/planner.md`

**Changes:** SESSION_DIR references.

### Phase 6: Context Response Update (1 file)

#### 6a. `pkg/session/context_response.go`

**Current (line 71):**
```go
sessionInfo := fmt.Sprintf("SESSION_DIR: %s\nAll session artifacts are written to this directory. .claude/tmp/ symlinks here.", ctx.SessionDir)
```

**Change to:**
```go
sessionInfo := fmt.Sprintf("SESSION_DIR: %s\nAll session artifacts are written to this directory. .gogent/tmp/ symlinks here.", ctx.SessionDir)
```

### Phase 7: Environment Variable Updates

#### New environment variables:

| Variable | Purpose | Default |
|----------|---------|---------|
| `GOGENT_RUNTIME_DIR` | Override runtime directory | `{project_root}/.gogent` |

#### Updated resolution chains:

**Session dir resolution (everywhere):**
1. `GOGENT_SESSION_DIR` (explicit)
2. Read `{project_root}/.gogent/current-session`
3. Read `{project_root}/.claude/current-session` (backward compat)
4. Fallback: `.gogent/sessions/unknown`

**Runtime root resolution:**
1. `GOGENT_RUNTIME_DIR` (explicit)
2. `{GOGENT_PROJECT_ROOT}/.gogent`
3. `{cwd}/.gogent`

### Phase 8: Test Fixture Updates (~30 files)

All test files that create fixtures with `.claude/sessions/`, `.claude/tmp/`, or `.claude/memory/` paths need updating.

**Strategy:** Create a test helper function:

```go
// testutil/paths.go
func RuntimeDir(projectDir string) string {
    return filepath.Join(projectDir, ".gogent")
}

func SessionDir(projectDir, sessionID string) string {
    return filepath.Join(RuntimeDir(projectDir), "sessions", sessionID)
}
```

Then find-and-replace across test files:
```bash
# Find all test files with .claude/sessions references
grep -rl '\.claude.*sessions' --include='*_test.go' .

# Find all test files with .claude/tmp references
grep -rl '\.claude.*tmp' --include='*_test.go' .

# Find all test files with .claude/memory references
grep -rl '\.claude.*memory' --include='*_test.go' .
```

### Phase 9: Documentation Updates (4 files)

- `.claude/schemas/teams/PROJECT-ROOT-RESOLUTION.md` — update path resolution docs
- `.claude/schemas/teams/TASK-ACCESS-POLICY.md` — update team dir path references
- `docs/TEAM-RUN-FRAMEWORK.md` — update all path examples
- `.claude/CLAUDE.md` — update SESSION_DIR docs, add `.gogent/` explanation

### Phase 10: Gitignore

Add to `.gitignore`:
```
.gogent/
```

---

## Implementation Order

```
Phase 1 (Core paths)     ← Everything depends on this
    │
    ├── Phase 2 (Hooks)  ← Uses new paths from Phase 1
    │
    ├── Phase 3 (Workspace init)  ← New tool, uses Phase 1
    │
    ├── Phase 6 (Context response)  ← Quick change
    │
    └── Phase 7 (Env vars)  ← Config changes
         │
         ├── Phase 4 (Skill docs)  ← Reference updates
         │
         ├── Phase 5 (Agent docs)  ← Reference updates
         │
         ├── Phase 8 (Tests)  ← Can be done in parallel
         │
         ├── Phase 9 (Docs)  ← Can be done in parallel
         │
         └── Phase 10 (Gitignore)  ← Trivial
```

**Critical path:** Phase 1 → Phase 2 → Phase 3 → Phases 4-10 (parallel)

**Minimum viable fix (unblocks team workflows):** Phase 1 + Phase 2 only. This changes the path resolution so all downstream components automatically use `.gogent/`. Phases 3-10 are improvements, not blockers.

---

## Backward Compatibility

### Old sessions in `.claude/sessions/`

**Decision: Read fallback, no migration.**

- `ReadCurrentSession()` tries `.gogent/current-session` first, falls back to `.claude/current-session`
- `resolveSessionDir()` in hooks tries both paths
- Old sessions are not migrated — they become inaccessible for new writes (which is correct, since they'd be blocked anyway)
- `ListSessions()` in TUI persistence reads from the new `.gogent/sessions/` path only

### Existing team directories

Team directories are self-contained (everything is relative to teamDir). `gogent-team-run` is already path-agnostic — it takes teamDir as argv[1]. No migration needed; new teams automatically go to `.gogent/`.

### Environment variables

No existing env vars change meaning. New `GOGENT_RUNTIME_DIR` is additive. `GOGENT_SESSION_DIR` continues to work as before — it just points to `.gogent/sessions/{id}/` now instead of `.claude/sessions/{id}/`.

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Missed write path in obscure code | Medium | High | Grep for all `.claude` write patterns before declaring done |
| Test fixtures not fully updated | High | Medium | Run full test suite; grep for `.claude/sessions` in test files |
| TUI session persistence breaks | Medium | High | Phase 1b is carefully designed with env var fallback |
| Old sessions lost | Low | Low | Old sessions are already broken (can't write). No data loss. |
| `.gogent/` conflicts with other tools | Low | Low | Unique name, gitignored |
| Race condition: multiple sessions writing to `.gogent/` | Low | Medium | Same as current `.claude/` behavior — session IDs are unique |

---

## Verification Checklist

After implementation:

- [ ] `bin/gogent-team-run` works with `.gogent/` team dir (already proven by bootstrap)
- [ ] `/braintrust` skill creates team dir in `.gogent/sessions/{id}/teams/`
- [ ] `/review` skill creates team dir in `.gogent/sessions/{id}/teams/`
- [ ] `/implement` skill creates team dir in `.gogent/sessions/{id}/teams/`
- [ ] `gogent-load-context` creates session in `.gogent/sessions/{id}/`
- [ ] `gogent-skill-guard` writes active-skill.json to `.gogent/sessions/{id}/`
- [ ] TUI session persistence reads/writes `.gogent/sessions/`
- [ ] `go test ./...` passes
- [ ] No remaining writes to `.claude/` from CC agents (grep verification)
- [ ] `.gogent/` is in `.gitignore`
