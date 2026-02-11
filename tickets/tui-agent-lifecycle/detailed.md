# TUI Agent Lifecycle Visibility Design

**Version:** 1.0
**Date:** 2026-02-09
**Status:** Design Proposal

---

## 1. Problem Statement

### 1.1 What the User Sees

When a user runs `gogent-team-run` to spawn a team of workers (e.g., braintrust with Einstein, Staff-Architect, and Beethoven), the TUI's agents panel displays **"No agents yet"** throughout the entire execution. The agents panel remains empty even though:

- Team workers are spawning and executing
- `config.json` is being updated every 30-60 seconds with rich member data (PID, cost, status, health)
- The teams panel (Alt+T to switch) shows team-level summaries correctly

### 1.2 Why It Matters

The agents panel is the **primary agent lifecycle visualization** in the TUI. It provides:

- **Hierarchical tree view** of parent-child agent relationships
- **Per-agent detail panel** with duration, token usage, model, tier
- **Real-time status updates** (spawning → running → complete/error)
- **Navigation** to drill into specific agents (up/down arrows)

Without agent-level visibility for team-run workers, users have:

- **No per-worker visibility** — can't see which specific worker is running/stalled
- **No hierarchical context** — can't see worker relationships (e.g., Mozart spawned Einstein)
- **No detailed metrics** — teams panel shows totals, not per-worker breakdowns
- **Inconsistent UX** — Task()-spawned agents appear in agents panel, team-run workers don't

### 1.3 What Data IS Available But Not Surfaced

Every team member in `config.json` has:

| TeamMember Field     | Agent Field Equivalent     | Available in config.json?                 |
| -------------------- | -------------------------- | ----------------------------------------- |
| `name`               | `id` (with transformation) | ✅ Yes                                    |
| `agent`              | `agentType`                | ✅ Yes                                    |
| `model`              | `model`                    | ✅ Yes                                    |
| `status`             | `status`                   | ✅ Yes (pending/running/completed/failed) |
| `process_pid`        | `pid`                      | ✅ Yes                                    |
| `cost_usd`           | `cost`                     | ✅ Yes                                    |
| `started_at`         | `startTime`                | ✅ Yes (ISO8601)                          |
| `completed_at`       | `endTime`                  | ✅ Yes (ISO8601)                          |
| `error_message`      | `error`                    | ✅ Yes                                    |
| `health_status`      | _(new)_                    | ✅ Yes (healthy/stall_warning/stalled)    |
| `stall_count`        | _(new)_                    | ✅ Yes                                    |
| `last_activity_time` | _(new)_                    | ✅ Yes                                    |

**The data exists. The TUI just doesn't read it.**

---

## 2. Proposed Approaches

### 2.1 Approach A: Poller-Based Bridge (Extend useTeamsPoller)

**Summary:** The `useTeamsPoller` hook already polls `config.json` every 5-30 seconds and parses team summaries. Extend it to also call `addAgent()` / `updateAgent()` for each team member.

**Architecture:**

```
┌────────────────────────────────────────────────────────────┐
│                     useTeamsPoller()                        │
│  (already runs in Layout.tsx every 5-30s)                  │
└─────────┬────────────────────────────────────┬─────────────┘
          │                                    │
          │ 1. Poll config.json files          │ 2. Parse TeamMember data
          │                                    │
          ▼                                    ▼
   ┌─────────────┐                      ┌──────────────────┐
   │ setTeams()  │                      │ mapMemberToAgent()│
   │ (existing)  │                      │ (new function)    │
   └─────────────┘                      └────────┬─────────┘
                                                 │
                                                 │ 3. For each member:
                                                 │    addAgent(mappedAgent)
                                                 │    or updateAgent(id, delta)
                                                 │
                                                 ▼
                                          ┌────────────────┐
                                          │  agents store  │
                                          │  (AgentsSlice) │
                                          └────────────────┘
                                                 │
                                                 │ 4. AgentTree re-renders
                                                 │
                                                 ▼
                                          ┌────────────────┐
                                          │  User sees     │
                                          │  team workers  │
                                          │  in tree       │
                                          └────────────────┘
```

**Implementation Steps:**

1. **Add mapping function** `mapTeamMemberToAgent(member: TeamMember, teamDir: string): Agent` in `useTeams.ts`
2. **Track seen agent IDs** in a Set to detect removals (if a member disappears from config, mark stale)
3. **Call `addAgent()` or `updateAgent()`** in the polling loop after `setTeams()`
4. **Generate stable IDs** using `team-${teamDir}-${memberName}` format
5. **Set `parentId`** to team-level parent (e.g., `team-${teamDir}` as a synthetic root)

**Field Mapping Table:**

| TeamMember Field | Agent Field   | Transformation                                                                 |
| ---------------- | ------------- | ------------------------------------------------------------------------------ |
| `name`           | `id`          | `team-${teamDir}-${name}`                                                      |
| `agent`          | `agentType`   | Direct copy                                                                    |
| `agent`          | `description` | Use agent ID as description (or empty)                                         |
| `model`          | `model`       | Direct copy                                                                    |
| `model`          | `tier`        | Map: haiku→"haiku", sonnet→"sonnet", opus→"opus"                               |
| `status`         | `status`      | Map: pending→"queued", running→"running", completed→"complete", failed→"error" |
| `process_pid`    | `pid`         | Direct copy                                                                    |
| `cost_usd`       | `cost`        | Direct copy                                                                    |
| `started_at`     | `startTime`   | `new Date(started_at).getTime()` (ISO8601 → ms)                                |
| `completed_at`   | `endTime`     | `new Date(completed_at).getTime()` (or undefined)                              |
| `error_message`  | `error`       | Direct copy                                                                    |
| _(team dir)_     | `parentId`    | `team-${teamDir}` (synthetic team root)                                        |

**Parent-Child Relationships:**

```
Root Agent (router)
└── Team Root (team-20260209.123456.braintrust)
    ├── Einstein (team-20260209.123456.braintrust-einstein)
    ├── Staff-Architect (team-20260209.123456.braintrust-staff-architect)
    └── Beethoven (team-20260209.123456.braintrust-beethoven)
```

**Pros:**

- ✅ **Minimal new code** (~100 lines in `useTeams.ts`)
- ✅ **Reuses proven infrastructure** (existing polling, config parsing)
- ✅ **No new dependencies** (no inotify, no sockets)
- ✅ **config.json has 90% of needed data** (just map fields)
- ✅ **Works with existing AgentTree component** (no UI changes needed)

**Cons:**

- ⚠️ **Polling latency** (5-30s) — not instant, but acceptable for background teams
- ⚠️ **Potential store churn** (every poll writes to agents store, even if unchanged)
  - Mitigation: Compare fields before calling `updateAgent()` (delta detection)
- ⚠️ **Dual source of truth** (team members exist in both `teams` and `agents` stores)
  - Mitigation: Teams store is canonical; agents store is derived view

---

### 2.2 Approach B: Filesystem Watcher (inotify/fswatch)

**Summary:** Use Node.js `fs.watch()` or `chokidar` to watch `config.json` for changes. Parse and update agent store on each write.

**Architecture:**

```
┌──────────────────────────────────────────────────────┐
│              gogent-team-run (Go)                     │
│  Writes config.json atomically every update          │
└──────────────────┬───────────────────────────────────┘
                   │
                   │ write config.json
                   │
                   ▼
         ┌─────────────────────┐
         │  config.json        │
         │  (inode changes)    │
         └──────────┬──────────┘
                    │
                    │ inotify event
                    │
                    ▼
         ┌─────────────────────────────────┐
         │  fs.watch() / chokidar          │
         │  (useConfigWatcher hook)        │
         └──────────┬──────────────────────┘
                    │
                    │ on change:
                    │  - readFile(config.json)
                    │  - parse TeamConfig
                    │  - mapMemberToAgent()
                    │  - updateAgent()
                    │
                    ▼
         ┌─────────────────────────────────┐
         │  agents store (AgentsSlice)     │
         └─────────────────────────────────┘
```

**Pros:**

- ✅ **Near-real-time updates** (~100ms latency vs 5-30s polling)
- ✅ **Event-driven** (only parses on actual changes, not every 5s)
- ✅ **Lower CPU usage** (no constant polling)

**Cons:**

- ❌ **Platform-specific quirks** (macOS FSEvents vs Linux inotify vs Windows)
  - `fs.watch()` on Linux fires multiple events per atomic write (tmp file, rename)
  - Need debouncing to avoid parsing same write 2-3 times
- ❌ **Additional dependency** (`chokidar` is 500KB, adds complexity)
- ❌ **Failure modes**:
  - Watch setup fails (permission, too many watchers)
  - Event loss (kernel buffer overflow)
  - Stale file descriptor (config.json deleted and recreated)
- ❌ **Harder to test** (need filesystem mock, event simulation)

**Verdict:** Premature optimization. Polling works fine for background teams (not interactive).

---

### 2.3 Approach C: IPC Channel (Unix Socket / Named Pipe)

**Summary:** `gogent-team-run` writes agent lifecycle events to a Unix socket or named pipe. TUI listens and updates agent store in real-time.

**Architecture:**

```
┌────────────────────────────────────────────────────┐
│          gogent-team-run (Go)                      │
│  On member status change:                          │
│    - json.Marshal(event)                           │
│    - conn.Write(event)                             │
└──────────────┬─────────────────────────────────────┘
               │
               │ Unix socket: /tmp/gogent-team-${teamDir}.sock
               │
               ▼
    ┌──────────────────────────────┐
    │  TUI (Node.js)               │
    │  useTeamEventSocket() hook   │
    │  net.createConnection()      │
    │  on('data', parseEvent)      │
    └──────────┬───────────────────┘
               │
               │ Event: {type: "member_status", member: {...}}
               │
               ▼
    ┌──────────────────────────────┐
    │  updateAgent(id, delta)      │
    └──────────────────────────────┘
```

**Event Schema:**

```typescript
interface TeamMemberEvent {
  type:
    | "member_started"
    | "member_updated"
    | "member_completed"
    | "member_failed";
  team_dir: string;
  member_name: string;
  member_data: TeamMember;
}
```

**Pros:**

- ✅ **Real-time updates** (instant, no polling)
- ✅ **Structured events** (JSON schema, versioned protocol)
- ✅ **Bidirectional potential** (TUI could send commands to team-run)

**Cons:**

- ❌ **Significant new infrastructure**:
  - Go: socket server, event marshaling, connection pool
  - TypeScript: socket client, event parsing, reconnection logic
  - 300-500 lines of new code across both sides
- ❌ **Protocol design needed**:
  - Event versioning (what if schema changes?)
  - Backpressure handling (what if TUI can't keep up?)
  - Connection lifecycle (reconnect on crash?)
- ❌ **New failure modes**:
  - Socket creation failure (permission, path collision)
  - Connection loss (process restart, network issue)
  - Message framing (where does one event end, next begin?)
  - Partial reads (JSON split across packets)
- ❌ **Testing complexity** (need socket mock, event replay)

**Verdict:** Over-engineering for the problem. Real-time updates aren't critical for background teams.

---

### 2.4 Approach D: Hybrid (Poller + config.json Enrichment)

**Summary:** Middle ground between A and C. Enrich `config.json` with more agent-level detail (add fields team-run doesn't currently write), then poll as in Approach A.

**New fields to add to `Member` struct:**

```go
// Member struct in config.go
type Member struct {
    // ... existing fields ...

    // Agent lifecycle fields (new)
    TurnCount       int     `json:"turn_count,omitempty"`
    ToolCallCount   int     `json:"tool_call_count,omitempty"`
    TokensInput     int     `json:"tokens_input,omitempty"`
    TokensOutput    int     `json:"tokens_output,omitempty"`
    SpawnMethod     string  `json:"spawn_method,omitempty"` // "team-run"
    Description     string  `json:"description,omitempty"`  // Brief task description
}
```

**Extraction logic in `spawn.go`:**

Parse CLI output JSON for token usage, turn count, tool call count (already available in `cliOutput` struct). Write to config.json on member completion.

**Pros:**

- ✅ **More complete agent data** (token usage, turns, tool calls)
- ✅ **Still uses polling** (no new infrastructure)
- ✅ **Better parity** with Task()-spawned agents

**Cons:**

- ⚠️ **More work in Go** (extract 4 new fields from CLI output)
- ⚠️ **config.json grows** (~50 bytes per member)
- ⚠️ **Not strictly necessary** — Approach A works without these fields

**Verdict:** Nice-to-have, not critical. Can add incrementally after Approach A.

---

## 3. Recommended Approach

**Approach A: Poller-Based Bridge** is the pragmatic choice.

**Why:**

1. **Minimal code** (~100 lines in `useTeams.ts`, no Go changes)
2. **Reuses proven polling** (already works reliably for teams)
3. **config.json has 90% of data** (no new fields needed immediately)
4. **Works with existing UI** (AgentTree, AgentDetail need no changes)
5. **Low risk** (if mapping breaks, teams panel still works)

**Not choosing B/C because:**

- **Real-time isn't critical** for background teams (5s latency is acceptable)
- **Complexity doesn't justify benefit** (watchers/sockets = 5x more code)
- **Failure modes multiply** (inotify quirks, socket reconnection, protocol versioning)

**Evolution path:**

- **Phase 1**: Approach A (poller-based bridge)
- **Phase 2**: Approach D (enrich config.json with token usage, turns)
- **Phase 3**: Approach C (IPC channel) — only if user feedback demands real-time

---

## 4. Detailed Design (Approach A)

### 4.1 Field Mapping

| TeamMember Field | Agent Field   | Transformation                                                                 |
| ---------------- | ------------- | ------------------------------------------------------------------------------ |
| `name`           | `id`          | `team-${teamDir}-${name}`                                                      |
| `agent`          | `agentType`   | Direct                                                                         |
| `agent`          | `description` | `${agent} worker` (e.g., "einstein worker")                                    |
| `model`          | `model`       | Direct (haiku/sonnet/opus)                                                     |
| `model`          | `tier`        | Map: haiku→"haiku", sonnet→"sonnet", opus→"opus"                               |
| `status`         | `status`      | Map: pending→"queued", running→"running", completed→"complete", failed→"error" |
| `process_pid`    | `pid`         | Direct (nullable)                                                              |
| `cost_usd`       | `cost`        | Direct                                                                         |
| `started_at`     | `startTime`   | `new Date(started_at).getTime()` or `Date.now()` if null                       |
| `completed_at`   | `endTime`     | `new Date(completed_at).getTime()` or undefined                                |
| `error_message`  | `error`       | Direct                                                                         |
| _(synthetic)_    | `parentId`    | `team-${teamDir}`                                                              |
| _(synthetic)_    | `spawnMethod` | `"mcp-cli"` (team-run uses MCP to spawn)                                       |

**Status Mapping:**

```typescript
function mapMemberStatus(status: string): AgentStatus {
  switch (status) {
    case "pending":
      return "queued";
    case "running":
      return "running";
    case "completed":
      return "complete";
    case "failed":
      return "error";
    default:
      return "queued";
  }
}
```

**Tier Mapping:**

```typescript
function mapModelToTier(model: string): "haiku" | "sonnet" | "opus" {
  if (model.includes("haiku")) return "haiku";
  if (model.includes("sonnet")) return "sonnet";
  if (model.includes("opus")) return "opus";
  return "sonnet"; // fallback
}
```

### 4.2 Stable Agent IDs

**Format:** `team-${teamDir}-${memberName}`

**Example:**

- Team dir: `20260209.123456.braintrust`
- Member name: `einstein`
- Agent ID: `team-20260209.123456.braintrust-einstein`

**Why this works:**

- **Unique per team** (teamDir is timestamp-based)
- **Stable within team** (memberName doesn't change)
- **Doesn't collide** with Task()-spawned agents (those use UUIDs)

**Team root ID:** `team-${teamDir}` (no member name suffix)

### 4.3 Parent-Child Relationships

**Option A: Flat (all members under team root)**

```
Router (root)
└── Team: braintrust (team-20260209.123456.braintrust)
    ├── einstein (team-20260209.123456.braintrust-einstein)
    ├── staff-architect (team-20260209.123456.braintrust-staff-architect)
    └── beethoven (team-20260209.123456.braintrust-beethoven)
```

**Option B: Wave-aware (members grouped by wave)**

```
Router (root)
└── Team: braintrust (team-20260209.123456.braintrust)
    ├── Wave 1 (team-20260209.123456.braintrust-wave-1)
    │   ├── einstein
    │   └── staff-architect
    └── Wave 2 (team-20260209.123456.braintrust-wave-2)
        └── beethoven
```

**Recommendation: Option A (flat).**

- Simpler (no synthetic wave nodes)
- Matches config.json structure (waves are just grouping, not hierarchy)
- Wave info visible in AgentDetail panel (add `wave` field to Agent interface)

### 4.4 Lifecycle: When to addAgent, updateAgent, or Mark Stale

**addAgent() when:**

- First time seeing `team-${teamDir}-${memberName}` in poll
- Member status is "pending" or "running"

**updateAgent() when:**

- Agent ID already exists in store
- Any field changed (status, cost, endTime, error, health)

**Mark stale when:**

- Agent was in store last poll but missing from current config.json
- Team completed and all members finalized

**Delta detection (avoid unnecessary updates):**

```typescript
function agentNeedsUpdate(existing: Agent, incoming: Agent): boolean {
  return (
    existing.status !== incoming.status ||
    existing.cost !== incoming.cost ||
    existing.endTime !== incoming.endTime ||
    existing.error !== incoming.error ||
    existing.pid !== incoming.pid
  );
}
```

### 4.5 Teams Panel vs Agents Panel Coexistence

**Question:** Should agents from teams appear in BOTH panels, or only when agents panel is active?

**Answer:** **Both panels, always.**

- **Teams panel** shows team-level summaries (total cost, wave progress, member counts)
- **Agents panel** shows per-worker detail (individual status, duration, tokens)
- **User switches with Alt+T** to toggle between views
- **Data flows to both** (teams store + agents store populated in parallel)

**Why this works:**

- **No confusion** — panels show different granularity (team vs worker)
- **Consistent UX** — agents panel ALWAYS shows all agents (Task-spawned + team workers)
- **Natural evolution** — as team-run becomes primary workflow, agents panel becomes comprehensive

### 4.6 Edge Cases

#### Case 1: Team Restarts

**Scenario:** User runs `gogent-team-run "$dir"` twice on the same team dir (retry after failure).

**Problem:** Same agent IDs reappear, but with reset state.

**Solution:** On restart detection (team status goes from "completed"/"failed" → "pending"), call `clearAgents()` for that team's subtree before re-adding.

```typescript
// Detect restart: team exists in store with completed status, but config shows pending
if (existingTeamStatus === "completed" && currentTeamStatus === "pending") {
  // Clear all child agents
  Object.values(agents).forEach((agent) => {
    if (agent.parentId === teamRootId) {
      // Remove from store
      delete agents[agent.id];
    }
  });
}
```

#### Case 2: Member Retries

**Scenario:** Member fails, retry_count increments, member runs again (same name, different PID).

**Problem:** Agent ID stays the same, but PID changes.

**Solution:** `updateAgent()` overwrites PID field. This is correct behavior (agent lifecycle continues, new process).

#### Case 3: Stale PIDs

**Scenario:** Team completes, but PID in config.json is from hours ago (process long dead).

**Problem:** AgentDetail shows stale PID that doesn't match current system state.

**Solution:** Document that `pid` field is historical (process that RAN the agent, not necessarily alive). Add tooltip: "PID from last execution (may be stale)".

#### Case 4: Team Deleted

**Scenario:** User deletes team dir while team is in "completed" state.

**Problem:** Agents remain in store forever (orphaned).

**Solution:** On team disappearance from filesystem, mark all child agents as stale (add `stale: true` field to Agent interface, gray out in tree).

---

## 5. Implementation Plan

### Phase 1: Basic Mapping (1-2 hours)

**Files to modify:**

1. **`packages/tui/src/hooks/useTeams.ts`**
   - Add `mapTeamMemberToAgent(member, teamDir)` function
   - Add agent update logic after `setTeams()` in polling loop
   - Track seen agent IDs for staleness detection

2. **`packages/tui/src/store/types.ts`**
   - Add `stale?: boolean` field to `Agent` interface (optional)
   - Add `wave?: number` field to `Agent` interface (optional)

**Estimated complexity:** Low (100 lines TypeScript)

**Test plan:**

- Run braintrust team, verify agents appear in tree
- Check AgentDetail shows correct cost, duration, status
- Verify parent-child relationship (team root → members)

### Phase 2: Delta Detection (1 hour)

**Files to modify:**

1. **`packages/tui/src/hooks/useTeams.ts`**
   - Add `agentNeedsUpdate()` function
   - Only call `updateAgent()` when fields changed

**Estimated complexity:** Low (20 lines TypeScript)

**Test plan:**

- Verify store updates only on actual changes (use React DevTools)
- Check CPU usage during polling (should be ~0% when idle)

### Phase 3: Edge Case Handling (2-3 hours)

**Files to modify:**

1. **`packages/tui/src/hooks/useTeams.ts`**
   - Team restart detection
   - Staleness marking on team deletion
   - Clear logic for team subtrees

2. **`packages/tui/src/components/AgentTree.tsx`**
   - Gray out stale agents (dimColor prop)

**Estimated complexity:** Medium (50 lines TypeScript)

**Test plan:**

- Delete team dir, verify agents gray out
- Restart team, verify agents clear and repopulate
- Member retry, verify PID updates

### Phase 4: Health Status Display (1 hour)

**Files to modify:**

1. **`packages/tui/src/components/AgentDetail.tsx`**
   - Add health_status field display
   - Add stall_count field display
   - Add last_activity_time field display

2. **`packages/tui/src/store/types.ts`**
   - Add `healthStatus?: string` to `Agent` interface
   - Add `stallCount?: number` to `Agent` interface
   - Add `lastActivityTime?: number` to `Agent` interface

**Estimated complexity:** Low (30 lines TypeScript)

**Test plan:**

- Run team with stall detection enabled
- Verify health fields appear in AgentDetail
- Check stall_warning status displays correctly

---

## 6. Alternatives Considered (Summary)

| Approach          | Pros                                 | Cons                                         | Verdict                        |
| ----------------- | ------------------------------------ | -------------------------------------------- | ------------------------------ |
| **A: Poller**     | Minimal code, reuses infra, low risk | 5-30s latency, potential churn               | ✅ **Recommended**             |
| **B: Watcher**    | Near-real-time, event-driven         | Platform quirks, debouncing, complexity      | ❌ Deferred (overkill)         |
| **C: IPC Socket** | Real-time, structured, bidirectional | 300+ LOC, protocol design, new failure modes | ❌ Deferred (over-engineering) |
| **D: Hybrid**     | More complete data, better parity    | Extra Go work, config bloat                  | ⏸️ Future enhancement          |

---

## 7. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    gogent-team-run (Go)                         │
│  - Spawns Claude CLI workers via exec.Command                   │
│  - Updates config.json every status change                      │
│  - Writes atomically (tmp → rename)                             │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ config.json write
                 │ (atomic: .tmp.{pid}.{counter}.tmp → config.json)
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│           GOGENT_SESSION_DIR/teams/*/config.json                │
│  {                                                              │
│    "team_name": "braintrust-20260209",                          │
│    "waves": [                                                   │
│      {                                                          │
│        "wave_number": 1,                                        │
│        "members": [                                             │
│          {                                                      │
│            "name": "einstein",                                  │
│            "agent": "einstein",                                 │
│            "model": "opus",                                     │
│            "status": "running",                                 │
│            "process_pid": 12345,                                │
│            "cost_usd": 0.95,                                    │
│            "started_at": "2026-02-09T12:34:56Z",                │
│            "health_status": "healthy",                          │
│            ...                                                  │
│          }                                                      │
│        ]                                                        │
│      }                                                          │
│    ]                                                            │
│  }                                                              │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ Poll every 5-30s (adaptive)
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│              useTeamsPoller() - Layout.tsx                      │
│  - getTeamSummaries(sessionDir)                                 │
│  - parseTeamSummary() → TeamSummary                             │
│  - setTeams(summaries) [existing]                               │
│  - mapTeamMemberToAgent() [NEW]                                 │
│  - addAgent() / updateAgent() [NEW]                             │
└────────┬──────────────────────────────┬─────────────────────────┘
         │                              │
         │ setTeams()                   │ addAgent() / updateAgent()
         │                              │
         ▼                              ▼
┌─────────────────────┐      ┌──────────────────────────────────┐
│   teams store       │      │      agents store                │
│   (TeamsSlice)      │      │      (AgentsSlice)               │
│                     │      │                                  │
│  - teams[]          │      │  - agents: Record<id, Agent>     │
│  - selectedTeamDir  │      │  - selectedAgentId               │
│  - selectedDetail   │      │  - rootAgentId                   │
└──────┬──────────────┘      └──────┬───────────────────────────┘
       │                            │
       │ Used by:                   │ Used by:
       │ TeamList.tsx               │ AgentTree.tsx
       │ TeamDetail.tsx             │ AgentDetail.tsx
       │                            │
       ▼                            ▼
┌─────────────────────┐      ┌──────────────────────────────────┐
│   Right Panel       │      │    Right Panel                   │
│   (teams mode)      │      │    (agents mode)                 │
│                     │      │                                  │
│  📊 Team summaries  │      │  🌳 Agent tree (hierarchical)    │
│  💰 Total costs     │      │  📈 Per-agent detail             │
│  🔢 Wave progress   │      │  ⏱️ Duration, tokens, status     │
│  👥 Member counts   │      │  🏥 Health status (if available) │
└─────────────────────┘      └──────────────────────────────────┘
```

---

## 8. Success Criteria

### 8.1 Functional Requirements

- ✅ Team workers appear in agents panel within 5-30 seconds of spawning
- ✅ Agent tree shows correct parent-child relationships (team → members)
- ✅ AgentDetail shows cost, duration, status, PID for team workers
- ✅ Status updates flow correctly (queued → running → complete/error)
- ✅ Agent tree updates when members complete or fail
- ✅ No duplicate agents (same member doesn't appear twice)
- ✅ Agents clear when team is restarted
- ✅ Agents gray out when team is deleted

### 8.2 Performance Requirements

- ✅ Polling overhead <5ms per poll (delta detection prevents unnecessary updates)
- ✅ No memory leak (stale agents cleaned up)
- ✅ UI responsive during polling (no frame drops)

### 8.3 UX Requirements

- ✅ Agents panel shows "team workers" with clear hierarchy
- ✅ User can navigate team workers with up/down arrows
- ✅ AgentDetail distinguishes team workers from Task-spawned agents
- ✅ Health status visible in AgentDetail (if available)
- ✅ No confusion between teams panel and agents panel

---

## 9. Open Questions

### Q1: Should team root be a synthetic agent or just a grouping?

**Option A:** Create synthetic "team root" agent in agents store (id: `team-${teamDir}`, description: team name)

**Option B:** Team members have `parentId: null` (appear as top-level)

**Recommendation:** Option A. Creates clearer hierarchy in tree view, matches Task()-spawned orchestrators.

### Q2: Should agents panel auto-switch when team spawns?

**Current behavior:** Teams poller auto-switches to teams panel when a team starts.

**Proposal:** Add setting: "Auto-switch agents panel to teams mode" (default: true)

**Recommendation:** Keep current behavior (auto-switch to teams), let user manually switch to agents if they want per-worker detail.

### Q3: How to handle token usage when CLI doesn't report it?

**Problem:** Some CLI invocations don't include token usage in output (older CLI version, error cases).

**Solution:** Leave `tokenUsage` field undefined. AgentDetail already handles this gracefully (doesn't render token section if undefined).

### Q4: Should health status affect agent status color in tree?

**Current:** Status color is green (running), blue (complete), red (error), yellow (queued).

**Proposal:** If `healthStatus === "stalled"`, override status color to orange/warning.

**Recommendation:** Yes. Add to `getStatusColor()` in `AgentTree.tsx`:

```typescript
function getStatusColor(
  status: Agent["status"],
  healthStatus?: string,
): string {
  if (healthStatus === "stalled") return colors.warning; // orange
  // ... existing logic
}
```

---

## 10. Migration Plan

### 10.1 Backward Compatibility

**No breaking changes.** All modifications are additive:

- ✅ `useTeamsPoller()` already runs unconditionally
- ✅ New fields on `Agent` interface are optional
- ✅ AgentTree/AgentDetail already handle missing fields gracefully
- ✅ Teams panel continues to work identically

### 10.2 Rollout

**Phase 1:** Implement mapping (useTeams.ts changes only)
**Phase 2:** Add health status display (AgentDetail.tsx)
**Phase 3:** Add stale agent graying (AgentTree.tsx)
**Phase 4:** Add team restart detection (useTeams.ts)

Each phase is independently testable and deployable.

### 10.3 Testing Strategy

**Unit tests:**

- `mapTeamMemberToAgent()` field transformations
- `agentNeedsUpdate()` delta detection
- `mapMemberStatus()` / `mapModelToTier()` mappings

**Integration tests:**

- Spawn braintrust team, verify agents appear
- Restart team, verify agents clear and repopulate
- Delete team, verify agents gray out
- Member failure, verify error message appears in AgentDetail

**Manual tests:**

- Run multiple teams concurrently, verify no cross-contamination
- Kill team-run mid-execution, verify PIDs don't leak
- Switch between teams and agents panels rapidly, verify no flicker

---

## 11. Future Enhancements

### 11.1 Token Usage Extraction (Approach D)

Add to `Member` struct:

```go
TokensInput     int     `json:"tokens_input,omitempty"`
TokensOutput    int     `json:"tokens_output,omitempty"`
```

Extract from `cliOutput` in `finalizeSpawn()`:

```go
if cliOut != nil {
    tr.updateMember(waveIdx, memIdx, func(m *Member) {
        m.TokensInput = cliOut.TokensInput
        m.TokensOutput = cliOut.TokensOutput
    })
}
```

**Benefit:** AgentDetail shows token usage for team workers, matching Task()-spawned agents.

### 11.2 Real-Time Updates (Approach C)

If user feedback demands <1s latency, implement IPC socket:

- Go: Write events to `/tmp/gogent-team-${sessionId}.sock` on every status change
- TypeScript: `useTeamEventSocket()` hook listens and updates agent store
- Protocol: JSON lines (newline-delimited JSON)

**Benefit:** Instant updates, better UX for interactive workflows.

**Cost:** 300-500 LOC, new failure modes, testing complexity.

### 11.3 Wave Visualization

Add wave grouping to agent tree:

```
Team: braintrust
├── Wave 1
│   ├── einstein
│   └── staff-architect
└── Wave 2
    └── beethoven
```

**Implementation:** Add `wave` field to `Agent` interface, modify `AgentTree.tsx` to group by wave.

**Benefit:** Clearer visualization of team structure, matches config.json.

---

## 12. Risks and Mitigations

| Risk                                  | Probability | Impact | Mitigation                                                     |
| ------------------------------------- | ----------- | ------ | -------------------------------------------------------------- |
| **Polling causes CPU spike**          | Low         | Medium | Delta detection prevents unnecessary updates; max 5ms per poll |
| **Store churn causes UI flicker**     | Low         | Low    | React batching prevents re-renders; only update changed agents |
| **Agent IDs collide**                 | Very Low    | High   | `team-${teamDir}-${name}` format ensures uniqueness            |
| **Team restart leaves orphan agents** | Medium      | Low    | Clear agents on restart detection                              |
| **Stale PIDs confuse user**           | Medium      | Low    | Document PID as "historical" in tooltip                        |
| **Health status missing**             | Low         | Low    | AgentDetail gracefully handles undefined fields                |

---

## 13. Conclusion

**Approach A (Poller-Based Bridge)** is the right first step:

- ✅ **Minimal code** (~100-150 lines TypeScript, zero Go changes)
- ✅ **Reuses proven infrastructure** (existing teams poller)
- ✅ **Low risk** (no new dependencies, no protocol design)
- ✅ **Solves 90% of problem** (agents panel shows team workers with full detail)

**Implementation estimate:** 4-6 hours total (mapping + delta detection + edge cases + health display)

**Evolution path clear:** Can add token usage (Approach D) or real-time updates (Approach C) later if needed.

**User benefit immediate:** After implementing Approach A, users see team workers in agents panel, can navigate with arrows, view detailed status/cost/health, and understand team hierarchy — all with existing UI components, no new UX to learn.
