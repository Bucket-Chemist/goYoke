# TUI Agent Detail Panel Upgrade: Health Monitor Dashboard

## Problem

The current agent detail panel (`UnifiedDetail.tsx`) shows basic metadata when you select a single node — model, status, duration, cost, and recent tool activity. For teams, `TeamRootDetail` shows budget and progress, while `TeamMemberDetail` shows individual agent status. You can only view one node at a time.

Meanwhile, `gogent-team-run` writes rich health telemetry into `config.json` every 30 seconds:

- `health_status`: healthy | stall_warning | stalled
- `stall_count`: consecutive stall intervals
- `last_activity_time`: ISO 8601 timestamp of last stdout output
- `process_pid`: actual CLI process ID
- Stream files (`stream_{agent}.ndjson`) with live output volume

**None of this is surfaced in the TUI.** The `useTeams` hook parses config.json but skips `health_status`, `stall_count`, `last_activity_time`, and `process_pid`. Stream file sizes aren't tracked. The detail panel shows a flat key-value list instead of an operational dashboard.

## Proposed Design

Replace the current team detail views with a unified **Health Monitor Dashboard** that shows the full team state at a glance — inspired by this layout:

```
┌─────────────────────────────────────────────────────────┐
│  BRAINTRUST TEAM: 1773453175                            │
│  Status: RUNNING    PID: 58935    Uptime: 12m34s        │
│  Budget: $42.18 / $50.00  ████████████████░░░░  84%     │
├─────────────────────────────────────────────────────────┤
│  Wave 1: Parallel theoretical + practical analysis      │
│                                                         │
│  🟢 einstein           RUNNING   PID 58944              │
│     health=healthy  stalls=0  last=2s ago               │
│     stream: 305KB  ▸ Read → go.mod                      │
│                                                         │
│  🟢 staff-architect    RUNNING   PID 58945              │
│     health=healthy  stalls=0  last=5s ago               │
│     stream: 469KB  ▸ Grep → "createSdkMcpServer"       │
│                                                         │
│  Wave 2: Synthesis                                      │
│  ⏸️  beethoven          PENDING                          │
│     waiting for Wave 1                                  │
├─────────────────────────────────────────────────────────┤
│  Totals: 2/3 running, 0 failed, $7.82 spent            │
└─────────────────────────────────────────────────────────┘
```

### Key differences from current panel

| Feature | Current | Proposed |
|---------|---------|----------|
| View scope | Single node at a time | All team members at once |
| Health status | Not shown | 🟢 healthy / ⚠️ stall_warning / 🔴 stalled |
| Stall count | Not tracked | Shown per member |
| Last activity | Activity text only | "Xs ago" relative timestamp |
| Stream volume | Not tracked | KB/MB output indicator |
| Process PID | Not shown (per member) | Shown per running member |
| Budget | Text only | Visual progress bar + percentage |
| Wave grouping | Flat list | Members grouped under wave headers |
| Totals | None | Running/failed/cost summary footer |

## Data Model Changes

### 1. Extend `TeamConfigJSON` parsing in `useTeams.ts`

The `TeamConfigJSON` interface's `members` array currently omits health fields. Extend it:

```typescript
// In useTeams.ts TeamConfigJSON.waves.members
interface TeamConfigWaveMember {
  name: string;
  agent: string;
  model: string;
  status: string;
  cost_usd: number;
  started_at: string | null;
  completed_at: string | null;
  // === NEW FIELDS (already in config.json, just not parsed) ===
  process_pid: number | null;
  health_status?: string;      // "healthy" | "stall_warning" | "stalled"
  last_activity_time?: string; // ISO 8601
  stall_count?: number;
  error_message?: string;
  kill_reason?: string;
}
```

### 2. Extend `TeamMemberRow` in `store/types.ts`

```typescript
export interface TeamMemberRow {
  name: string;
  agent: string;
  model: string;
  status: string;
  wave: number;
  cost: number;
  startedAt: string | null;
  completedAt: string | null;
  // === NEW FIELDS ===
  processPid?: number | null;
  healthStatus?: string;
  lastActivityTime?: string;
  stallCount?: number;
  errorMessage?: string;
  killReason?: string;
  streamBytes?: number;        // Size of stream_{agent}.ndjson
  // === EXISTING (keep) ===
  latestActivity?: string;
  activity?: AgentActivity;
}
```

### 3. Extend `TeamSummary` in `store/types.ts`

```typescript
export interface TeamSummary {
  // ... existing fields ...
  // === NEW FIELDS ===
  totalStreamBytes: number;    // Sum of all stream file sizes
  uptimeMs: number | null;     // Since started_at (computed)
}
```

### 4. Add stream size tracking in `useTeams.ts`

In `attachStreamActivity`, also capture the file size:

```typescript
async function attachStreamActivity(
  summary: TeamSummary,
  teamsBasePath: string
): Promise<void> {
  const teamPath = join(teamsBasePath, summary.dir);
  let totalBytes = 0;

  for (const member of summary.members) {
    // Always try to get stream size, even for non-running members
    try {
      const streamPath = join(teamPath, `stream_${member.agent}.ndjson`);
      const fileStat = await stat(streamPath);
      member.streamBytes = fileStat.size;
      totalBytes += fileStat.size;
    } catch {
      // No stream file yet — expected for pending members
    }

    if (member.status === "running") {
      const result = await readStreamActivity(teamPath, member.agent);
      member.latestActivity = result.text?.slice(0, 100) ?? undefined;
      member.activity = result.activity ?? undefined;
    }
  }

  summary.totalStreamBytes = totalBytes;
}
```

### 5. Wire new fields in `parseTeamSummary`

```typescript
members.push({
  name: member.name,
  agent: member.agent,
  model: member.model,
  status: member.status,
  wave: wave.wave_number,
  cost: member.cost_usd,
  startedAt: member.started_at,
  completedAt: member.completed_at,
  // NEW
  processPid: member.process_pid,
  healthStatus: member.health_status,
  lastActivityTime: member.last_activity_time,
  stallCount: member.stall_count,
  errorMessage: member.error_message,
  killReason: member.kill_reason,
});
```

## Component Changes

### 1. New: `TeamDashboard` component

Replace `TeamRootDetail` and `TeamMemberDetail` in `UnifiedDetail.tsx` with a single `TeamDashboard` component that renders all members grouped by wave.

**File:** `packages/tui/src/components/TeamDashboard.tsx`

The dashboard renders when any team-related node is selected (team-root or team-member). It always shows the full team — selecting a specific member scrolls/highlights that member within the dashboard but doesn't hide the others.

#### Sub-components

```
TeamDashboard
├── TeamHeader        (name, status, PID, uptime)
├── BudgetBar         (visual progress bar with color coding)
├── WaveSection[]     (one per wave)
│   ├── WaveHeader    (wave number + description)
│   └── MemberRow[]   (health, status, PID, stream, activity)
└── TeamFooter        (totals: running/failed/cost/stream)
```

### 2. `BudgetBar` component

Visual budget indicator using Unicode block characters:

```typescript
function BudgetBar({ used, max }: { used: number; max: number }): JSX.Element {
  const pct = max > 0 ? Math.round((used / max) * 100) : 0;
  const barWidth = 20;
  const filled = Math.round((pct / 100) * barWidth);
  const empty = barWidth - filled;
  const bar = "█".repeat(filled) + "░".repeat(empty);
  const color = pct >= 90 ? colors.error : pct >= 70 ? colors.warning : colors.success;

  return (
    <Box>
      <Text color={colors.muted}>Budget: </Text>
      <Text color={color}>${(max - used).toFixed(2)}</Text>
      <Text color={colors.muted}> / ${max.toFixed(2)}  </Text>
      <Text color={color}>{bar}</Text>
      <Text color={colors.muted}>  {pct}%</Text>
    </Box>
  );
}
```

### 3. `MemberRow` component

Each member shows a compact 2-3 line summary:

```
🟢 einstein           RUNNING   PID 58944
   health=healthy  stalls=0  last=2s ago
   stream: 305KB  ▸ Read → go.mod
```

Line 1: status icon + name + status + PID
Line 2: health indicator + stall count + relative last activity time
Line 3: stream size + current tool (if running)

For stalled members:
```
🔴 einstein           RUNNING   PID 58944
   health=STALLED  stalls=5  last=4m23s ago ⚠️
   stream: 305KB (frozen)
```

For failed members:
```
❌ einstein           FAILED    exit=1
   error: timeout after 30m
   stream: 2.1MB
```

For pending members:
```
⏸️  beethoven          PENDING
   waiting for Wave 1
```

### 4. Health status colors

```typescript
function getHealthColor(status?: string): string {
  switch (status) {
    case "healthy":      return colors.success;
    case "stall_warning": return colors.warning;
    case "stalled":      return colors.error;
    default:             return colors.muted;
  }
}

function getHealthIcon(status?: string): string {
  switch (status) {
    case "healthy":      return "🟢";
    case "stall_warning": return "⚠️";
    case "stalled":      return "🔴";
    default:             return "⏸️";
  }
}
```

### 5. Relative time formatting

```typescript
function formatRelativeTime(isoTime?: string): string {
  if (!isoTime) return "-";
  const ms = Date.now() - new Date(isoTime).getTime();
  if (ms < 1000) return "just now";
  if (ms < 60000) return `${Math.floor(ms / 1000)}s ago`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m${Math.floor((ms % 60000) / 1000)}s ago`;
  return `${Math.floor(ms / 3600000)}h${Math.floor((ms % 3600000) / 60000)}m ago`;
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === 0) return "0B";
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(0)}KB`;
  return `${(bytes / 1048576).toFixed(1)}MB`;
}
```

## Integration Points

### UnifiedDetail dispatch

In `UnifiedDetail.tsx`, when a team-root or team-member node is selected, render `TeamDashboard` instead of the current separate views:

```typescript
// Replace:
{selectedNode?.kind === "team-root" && ...}
{selectedNode?.kind === "team-member" && ...}

// With:
{(selectedNode?.kind === "team-root" || selectedNode?.kind === "team-member") && (
  <TeamDashboard
    teamDir={selectedNode.teamDir ?? selectedNode.parentTeamDir}
    highlightMember={selectedNode.kind === "team-member" ? selectedNode.displayName : undefined}
  />
)}
```

### UnifiedNode type extension

Add `parentTeamDir` to `UnifiedNode` for team-member nodes so the dashboard can find the team:

```typescript
export interface UnifiedNode {
  // ... existing ...
  parentTeamDir?: string; // For team-member nodes: the parent team's dir
}
```

This requires a small change in `useUnifiedTree.ts` where team-member nodes are constructed.

### Polling rate

The current 5s polling interval for running teams is sufficient. The health monitor data in config.json updates every 30s (from gogent-team-run's health check interval), and stream files update continuously. The 5s poll gives a responsive feel for stream size changes while not missing health updates.

## Cross-Session Team Discovery

**Out of scope for this ticket** but worth noting: the current `useTeams` hook only watches `GOGENT_SESSION_DIR/teams/`. Teams launched from other sessions (like the recovery scenario that inspired this spec) are invisible. A follow-up ticket could add a "cross-session team watcher" that scans recent sessions for alive teams.

## Files to Modify

| File | Change |
|------|--------|
| `packages/tui/src/store/types.ts` | Extend `TeamMemberRow`, `TeamSummary` |
| `packages/tui/src/hooks/useTeams.ts` | Parse new config fields, track stream sizes |
| `packages/tui/src/hooks/useUnifiedTree.ts` | Add `parentTeamDir` to team-member nodes |
| `packages/tui/src/components/UnifiedDetail.tsx` | Route team nodes to `TeamDashboard` |
| `packages/tui/src/components/TeamDashboard.tsx` | **NEW** — the dashboard component |
| `packages/tui/src/utils/teamFormatting.ts` | Add `formatRelativeTime`, `formatBytes` |

## Acceptance Criteria

1. Selecting any team node (root or member) shows the full dashboard with all waves and members
2. Health status (healthy/stall_warning/stalled) is visible per member with color coding
3. Stall count and relative last-activity time are shown
4. Budget has a visual progress bar with color thresholds (green < 70%, yellow 70-90%, red > 90%)
5. Stream file size is shown per member and updates on each poll
6. Current tool activity is shown inline for running members
7. Failed members show error message and kill reason
8. Pending members show which wave they're waiting for
9. SDK agent detail panel (`SdkAgentDetail`) is unchanged
10. Dashboard updates reactively on the existing 5s poll cycle

## Estimated Scope

- **Types/data**: ~30 lines changed across 2 files
- **useTeams parsing**: ~20 lines changed
- **TeamDashboard component**: ~180 lines new
- **UnifiedDetail integration**: ~10 lines changed
- **Formatting utilities**: ~20 lines new
- **Total**: ~260 lines, 1 new file, 5 modified files
