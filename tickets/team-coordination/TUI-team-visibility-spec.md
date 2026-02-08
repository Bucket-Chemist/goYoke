# TUI Team Visibility — Exploration Spec

**Purpose:** Define what data the TUI needs to display background team orchestration, what's already available, and what components need building.

**Status:** Exploration spec — not a ticket. Read this before scoping implementation.

---

## Current State

### What the TUI knows today

| Data | Source | Component |
|------|--------|-----------|
| Running team count | `useTeamCount` polls `$GOGENT_SESSION_DIR/teams/*/config.json` every 30s | `StatusLine.tsx` shows "🏗️ N teams running" |
| Agent tree (MCP-spawned) | `AgentsSlice` in Zustand store, populated by `spawn_agent` events | `AgentTree.tsx` recursive tree |
| Agent detail (MCP-spawned) | Same store, selected agent | `AgentDetail.tsx` shows model, tier, status, duration, tokens |
| Background team count | `TeamsSlice.backgroundTeamCount` (integer) | Status line badge |

### What the TUI does NOT know today

| Missing | Why | Impact |
|---------|-----|--------|
| Which teams exist | Only counts, doesn't enumerate | Can't list teams or show names |
| Team workflow type | Not read from config.json | Can't distinguish braintrust vs review vs implementation |
| Wave structure | Not parsed | Can't show progress through waves |
| Member status per wave | Not parsed | Can't show which workers are running/done/failed |
| Member costs | Not parsed | Can't show cost breakdown |
| Team budget | Not parsed | Can't show budget usage |
| Stdout results | Not read | Can't show what workers produced |
| Team-run spawned agents | `gogent-team-run` spawns `claude -p` processes directly — bypasses `spawn_agent` MCP | Workers are invisible in AgentTree |

**Key gap:** `gogent-team-run` workers are invisible to the TUI. They're raw `claude -p` processes, not MCP-spawned agents. The AgentTree only shows agents created via `spawn_agent`.

---

## Data Model: What config.json provides

Every team directory has a `config.json` that `gogent-team-run` updates in real-time:

```typescript
interface TeamConfig {
  team_name: string;              // "implementation-1770550714"
  workflow_type: string;          // "implementation" | "braintrust" | "review"
  project_root: string;
  session_id: string;
  created_at: string;             // ISO-8601
  budget_max_usd: number;
  budget_remaining_usd: number;
  warning_threshold_usd: number;
  status: "pending" | "running" | "completed" | "failed";
  background_pid: number | null;
  started_at: string | null;      // ISO-8601
  completed_at: string | null;    // ISO-8601
  waves: TeamWave[];
}

interface TeamWave {
  wave_number: number;            // 1-indexed
  description: string;
  on_complete_script: string | null;
  members: TeamMember[];
}

interface TeamMember {
  name: string;                   // "task-001"
  agent: string;                  // "go-pro", "einstein", etc.
  model: string;                  // "sonnet", "opus"
  stdin_file: string;             // "stdin_task-001.json"
  stdout_file: string;            // "stdout_task-001.json"
  status: "pending" | "running" | "completed" | "failed" | "skipped";
  process_pid: number | null;
  exit_code: number | null;
  cost_usd: number;
  cost_status: string;            // "ok" | "fallback" | "error"
  error_message: string;
  retry_count: number;
  max_retries: number;
  timeout_ms: number;
  started_at: string | null;
  completed_at: string | null;
}
```

### What stdout files provide (per worker)

Each completed worker writes `stdout_task-NNN.json`:

```typescript
interface WorkerStdout {
  $schema: string;                // "implementation-worker"
  worker: string;                 // "go-pro"
  task_id: string;
  status: "complete" | "partial" | "failed";
  summary: string;
  files_modified: Array<{
    path: string;
    action: "created" | "modified" | "deleted";
    lines_changed: number;
    description: string;
  }>;
  tests_written: Array<{
    file: string;
    test_count: number;
    test_names: string[];
  }>;
  acceptance_criteria_met: Array<{
    criterion: string;
    status: "met" | "not_met" | "partial";
    evidence: string;
  }>;
  build_status: {
    compiled: boolean;
    tests_passed: boolean;
    race_detector_clean: boolean;
  };
  blockers: Array<{
    blocker: string;
    impact: string;
    needs: string;
  }>;
}
```

---

## Proposed TUI Components

### 1. TeamList (new component)

**Where:** Right panel, new mode `rightPanelMode: "teams"`

**What it shows:**

```
Teams (2)

  ✓ implementation-1770550714  [completed]
    impl · 2 waves · 3 workers · $0.85 · 1m 44s

  ⏳ braintrust-1770551000     [running]
    braintrust · wave 2/3 · $1.20 · 2m 15s
```

**Data needed:**
- Enumerate team dirs from `$GOGENT_SESSION_DIR/teams/`
- Parse each `config.json` for name, type, status, budget, timing
- Compute current wave (first wave with non-completed members)

### 2. TeamDetail (new component)

**Where:** Replaces AgentDetail when a team is selected

**What it shows:**

```
Team: implementation-1770550714
Type: implementation
Status: RUNNING (PID 437079)
Budget: $1.50 / $10.00 (85%)
Started: 18:32:04 · Elapsed: 1m 22s

Wave 1: COMPLETED (1/1)
  ✓ task-001  go-pro  $0.28  42s

Wave 2: RUNNING (1/3)
  ⏳ task-002  go-pro  $0.00  running
  ⏳ task-003  go-pro  $0.00  running
  ⏸ task-004  go-pro  pending
```

**Data needed:**
- Full `config.json` parse with wave/member detail
- PID liveness check (existing `isPidAlive` function)
- Real-time: re-read config.json on poll interval

### 3. TeamTree (extend AgentTree)

**Option A: Separate tree.** Teams appear as a second tree below the agent tree:

```
Agents (3)
  ◉ opus: router
    ├─ ⏳ sonnet: architect
    └─ ✓ haiku: codebase-search

Teams (1)
  🏗️ implementation-1770550714  [running]
    Wave 1: ✓
      └─ ✓ task-001 (go-pro)
    Wave 2: ⏳
      ├─ ⏳ task-002 (go-pro)
      ├─ ⏳ task-003 (go-pro)
      └─ ⏸ task-004 (go-pro)
```

**Option B: Unified tree.** Teams appear as children of the agent that spawned them (requires tracking which agent called `/implement` or `/ticket`):

```
Agents (3)
  ◉ opus: router
    ├─ ⏳ sonnet: architect
    │   └─ 🏗️ implementation-1770550714
    │       ├─ Wave 1: ✓ task-001 (go-pro)
    │       └─ Wave 2: ⏳ task-002, task-003, task-004
    └─ ✓ haiku: codebase-search
```

**Recommendation:** Option A first — simpler, no parent tracking needed. Option B as follow-up.

### 4. StatusLine integration (enhance existing)

Current: `🏗️ 2 teams running`

Enhanced:
```
🏗️ 2 teams · wave 3/5 · $2.40
```

Shows: team count + furthest wave progress + total team spend.

---

## Store Changes

### New TeamsSlice (replace current minimal version)

```typescript
interface TeamSummary {
  dir: string;                    // Full path to team dir
  name: string;
  workflowType: string;
  status: "pending" | "running" | "completed" | "failed";
  backgroundPid: number | null;
  alive: boolean;                 // PID liveness
  budgetMax: number;
  budgetRemaining: number;
  startedAt: string | null;
  completedAt: string | null;
  totalCost: number;              // sum of member costs
  waveCount: number;
  currentWave: number;            // first non-completed wave
  memberCount: number;
  completedMembers: number;
  failedMembers: number;
}

interface TeamsSlice {
  teams: TeamSummary[];           // All teams in session
  selectedTeamDir: string | null;
  selectedTeamDetail: TeamConfig | null;  // Full config for selected team
  setTeams: (teams: TeamSummary[]) => void;
  selectTeam: (dir: string | null) => void;
  setTeamDetail: (config: TeamConfig | null) => void;
}
```

### New hook: useTeams (replace useTeamCount)

```typescript
function useTeams(pollIntervalMs = 5000): TeamSummary[] {
  // 1. Enumerate $GOGENT_SESSION_DIR/teams/*/config.json
  // 2. Parse each, compute summary fields
  // 3. Check PID liveness
  // 4. Return sorted by created_at desc
  // Polling: 5s for running teams (config.json changes frequently)
  //          30s if all teams completed (nothing to update)
}
```

### Faster polling for running teams

`useTeamCount` polls every 30s. For active team visibility, running teams need 5s polling (config.json updates on every member status change). Completed teams can stay at 30s.

---

## What /team-status, /team-result, /team-cancel return

These are CLI slash commands (SKILL.md based). They read config.json and stdout files directly. For TUI integration, the same data is available — just read via hooks instead of skills.

| Slash Command | What it reads | Data returned |
|---------------|---------------|---------------|
| `/team-status` | `config.json` | Team status, wave progress, member status/cost/retries, PID liveness, budget |
| `/team-result` | `stdout_*.json` | Per-worker structured JSON: files_modified, tests, acceptance_criteria, blockers |
| `/team-cancel` | `config.json` → `background_pid` | Sends SIGTERM to team-run process, waits for graceful shutdown |
| `/teams` | All `config.json` files | Summary list: name, type, status, duration, cost |

For TUI: the hooks (`useTeams`, `useTeamDetail`) provide the same data. No new Go binaries needed — just filesystem reads from TypeScript.

---

## Orchestrator Visibility

When `/implement` or `/ticket` launches the pipeline:

```
User: /implement "add health endpoint"
  │
  ├─ Router (opus) — visible in AgentTree as root
  │   │
  │   ├─ Architect (sonnet, Task) — visible in AgentTree as child
  │   │   └─ writes .claude/tmp/implementation-plan.json
  │   │
  │   ├─ gogent-plan-impl (Go binary) — NOT an agent, NOT visible
  │   │   └─ writes config.json + stdin files
  │   │
  │   └─ gogent-team-run (Go binary, background) — NOT an agent
  │       ├─ Wave 1: claude -p worker — NOT visible in AgentTree
  │       └─ Wave 2: claude -p workers — NOT visible in AgentTree
  │
  └─ Team: implementation-XXXXXXX — visible in TeamList/TeamTree
      ├─ Wave 1: task-001 → visible as TeamMember
      └─ Wave 2: task-002, task-003 → visible as TeamMembers
```

**Visibility boundary:**
- AgentTree shows MCP-spawned agents (router → architect)
- TeamList/TeamTree shows team-run workers (waves + members)
- The Go binaries (gogent-plan-impl, gogent-team-run) are infrastructure — invisible, by design

**Connection point:** The architect agent is the last visible node in AgentTree before the team takes over. For Option B (unified tree), the team would appear as a child of the architect's agent node.

---

## Implementation Priority

| Component | Effort | Value | Priority |
|-----------|--------|-------|----------|
| `useTeams` hook (replace `useTeamCount`) | S | High | P0 — foundation for everything |
| `TeamList` component | M | High | P0 — basic team enumeration |
| `TeamDetail` component | M | High | P1 — wave/member drill-down |
| StatusLine enhancement | S | Medium | P1 — quick glance info |
| TeamTree (Option A: separate) | L | Medium | P2 — visual wave progress |
| TeamTree (Option B: unified) | XL | Low | P3 — nice but complex parent tracking |
| Stdout result viewer | L | Medium | P2 — see what workers produced |
| Real-time wave animations | M | Low | P3 — polish |

---

## Open Questions

1. **Poll interval tradeoff:** 5s polling on config.json is frequent. Should we use filesystem watchers (`fs.watch`) instead? Risk: watchers are unreliable on some Linux filesystems.

2. **Team-agent linkage:** To support Option B (unified tree), we need to know which agent spawned the team. Options: (a) Store team_dir in agent's output field, (b) Add `spawned_team` field to Agent type, (c) Infer from timing.

3. **Historical teams:** Should completed teams persist in the list for the session? Currently config.json stays on disk. Clearing on session end happens via `gogent-archive`.

4. **Notification on completion:** Should the TUI toast when a background team completes? `useTeams` could detect status transitions and fire `addToast`.

5. **Cost aggregation:** Should team costs roll up into `SessionSlice.totalCost`? Currently team costs are separate (tracked in config.json budget, not in the TUI session cost).
