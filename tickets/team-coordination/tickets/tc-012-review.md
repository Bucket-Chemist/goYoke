# TC-012 Code Review Report

**Ticket:** TC-012 — Implement Team Slash Commands
**Date:** 2026-02-08
**Reviewers:** Frontend, Backend, Standards, Architecture (4 parallel Haiku agents)
**Overall Verdict:** WARNING (approve with conditions)

---

## Scope of Review

### Files Modified
| File | Change Type |
|------|-------------|
| `packages/tui/src/App.tsx` | Modified — GOGENT_SESSION_DIR env var init |
| `packages/tui/src/components/StatusLine.tsx` | Modified — team count display |
| `packages/tui/src/store/slices/session.ts` | Modified — backgroundTeamCount state |
| `packages/tui/src/store/types.ts` | Modified — SessionSlice interface |

### Files Created
| File | Purpose |
|------|---------|
| `packages/tui/src/hooks/useTeamCount.ts` | Hook: polls teams dir for running teams |
| `packages/tui/tests/components/StatusLine.test.tsx` | Tests: StatusLine store integration |
| `packages/tui/tests/hooks/useTeamCount.test.ts` | Tests: team count polling logic |
| `packages/tui/tests/integration/session-env.test.ts` | Tests: GOGENT_SESSION_DIR lifecycle |
| `~/.claude/skills/team-status/SKILL.md` | Skill: /team-status command |
| `~/.claude/skills/team-result/SKILL.md` | Skill: /team-result command |
| `~/.claude/skills/team-cancel/SKILL.md` | Skill: /team-cancel command |
| `~/.claude/skills/teams/SKILL.md` | Skill: /teams command |

### Test Results
- 28/28 tests passing
- Build clean (144.6kb)

---

## Finding #1: process.env Mutation in React useEffect (CRITICAL)

**Reported by:** Backend, Frontend, Architecture (3/4 reviewers)

### Problem

`GOGENT_SESSION_DIR` is set inside a React `useEffect` in `App.tsx`. This creates a race condition: child components (including `useTeamCount`) mount and execute their first poll **before** the useEffect fires, because effects run after render.

**Timeline:**
```
1. App mounts → child components mount (including StatusLine → useTeamCount)
2. useTeamCount calls poll() immediately on mount
3. process.env["GOGENT_SESSION_DIR"] is still undefined → poll returns 0
4. useEffect in App.tsx fires → sets GOGENT_SESSION_DIR
5. Next poll (30s later) works correctly
```

**Result:** Dashboard shows 0 teams for the first 30 seconds even if teams are running.

### Current Code (App.tsx)

The env var is set in 3 separate places with identical logic (also a DRY violation — see Finding #2):

```typescript
// Line 51-52 (new session, no --session flag)
const home = process.env["HOME"] || homedir();
process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", newId);

// Line 65-66 (resumed session)
const home = process.env["HOME"] || homedir();
process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", session.id);

// Line 85-86 (error fallback)
const home = process.env["HOME"] || homedir();
process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", newId);
```

### Fix Options

**Option A (Quick fix): Block child rendering until env var is set**

In `App.tsx`, add a `sessionReady` state that gates Layout rendering:

```typescript
const [sessionReady, setSessionReady] = useState(false);

useEffect(() => {
  async function initSession() {
    // ... existing session logic ...
    setSessionDir(sessionId);  // helper from Fix #2
    setSessionReady(true);
  }
  initSession();
}, [sessionId]);

if (!sessionReady) return <LoadingScreen />;
return <Layout />;
```

**Option B (Proper fix): Pass sessionDir as explicit prop**

Remove process.env dependency entirely. Store sessionDir in Zustand and pass to useTeamCount:

```typescript
// In useTeamCount.ts — change signature:
export function useTeamCount(sessionDir: string | null, pollIntervalMs = 30000): number

// In StatusLine.tsx:
const sessionDir = useStore(s => s.sessionDir);  // from store, not env
const teamCount = useTeamCount(sessionDir);
```

This requires adding `sessionDir: string | null` to the session slice (or a new teams slice — see Finding #5).

**Option C (Minimal fix): Move env var setting before React renders**

In `packages/tui/src/index.tsx`, set the env var in the `main()` function before `render(<App />)`:

```typescript
async function main() {
  // ... existing setup ...
  const sessionId = args.session || nanoid();
  const home = process.env["HOME"] || homedir();
  process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId);

  render(<App sessionId={sessionId} />);
}
```

**Recommended:** Option A for immediate fix, Option B as follow-up.

---

## Finding #2: DRY Violation — 3x Duplicated Session Dir Logic (HIGH)

**Reported by:** Standards

### Problem

Identical 2-line block repeated 3 times in `App.tsx` (lines 51-52, 65-66, 85-86):

```typescript
const home = process.env["HOME"] || homedir();
process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", someId);
```

### Fix

Extract to helper function at top of `App.tsx`:

```typescript
function setSessionDir(sessionId: string): void {
  const home = process.env["HOME"] || homedir();
  process.env["GOGENT_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId);
}
```

Then replace all 3 occurrences:
- Line 51-52 → `setSessionDir(newId);`
- Line 65-66 → `setSessionDir(session.id);`
- Line 85-86 → `setSessionDir(newId);`

**Effort:** 5 minutes. No test changes needed.

---

## Finding #3: useTeamCount Async Cleanup Gap (HIGH)

**Reported by:** Frontend

### Problem

If the component unmounts while `poll()` is still executing (async), `setBackgroundTeamCount` will be called on an unmounted component. React warns about this as a memory leak.

### Current Code (useTeamCount.ts lines 83-107)

```typescript
useEffect(() => {
  const poll = async (): Promise<void> => {
    const sessionDir = process.env["GOGENT_SESSION_DIR"];
    if (!sessionDir) {
      setBackgroundTeamCount(0);  // ← can fire after unmount
      return;
    }
    const count = await countRunningTeams(sessionDir);
    setBackgroundTeamCount(count);  // ← can fire after unmount
  };

  void poll();
  const interval = setInterval(() => { void poll(); }, pollIntervalMs);
  return () => { clearInterval(interval); };
}, [pollIntervalMs, setBackgroundTeamCount]);
```

### Fix

Add an `isMounted` ref:

```typescript
useEffect(() => {
  let isMounted = true;

  const poll = async (): Promise<void> => {
    const sessionDir = process.env["GOGENT_SESSION_DIR"];
    if (!sessionDir) {
      if (isMounted) setBackgroundTeamCount(0);
      return;
    }
    const count = await countRunningTeams(sessionDir);
    if (isMounted) setBackgroundTeamCount(count);
  };

  void poll();
  const interval = setInterval(() => { void poll(); }, pollIntervalMs);
  return () => {
    isMounted = false;
    clearInterval(interval);
  };
}, [pollIntervalMs, setBackgroundTeamCount]);
```

**Effort:** 5 minutes. Add test case for unmount-during-poll scenario.

---

## Finding #4: Session Discovery Duplicated Across 4 Skills (HIGH)

**Reported by:** Architecture

### Problem

All 4 team skills (`team-status`, `team-result`, `team-cancel`, `teams`) contain identical session discovery instructions:

```markdown
1. Check environment variable GOGENT_SESSION_DIR
2. If unset, fallback to most recent session with teams/ subdirectory
3. Team directories at {session_dir}/teams/{team_name}/
```

If discovery logic changes (e.g., new env var name, different fallback strategy), all 4 files must be updated.

### Fix (Future — Not Blocking)

Create a shared Bash snippet or utility that skills reference:

```
~/.claude/skills/_shared/
  discover-session.md    # Shared instructions block
```

Or create a Go binary `gogent-discover-session` that skills call:
```bash
SESSION_DIR=$(gogent-discover-session)
```

**Effort:** 1-2 hours. Not blocking for TC-012 merge.

---

## Finding #5: backgroundTeamCount in Wrong Store Slice (HIGH)

**Reported by:** Architecture

### Problem

`backgroundTeamCount` was added to `SessionSlice`, but it's team-related state, not session-related state. As team features grow (team list, wave details, selected team), the session slice becomes a kitchen sink.

### Current Location

```typescript
// store/types.ts — SessionSlice
export interface SessionSlice {
  sessionId: string | null;
  totalCost: number;
  // ... session concerns ...
  backgroundTeamCount: number;          // ← doesn't belong here
  setBackgroundTeamCount: (count: number) => void;  // ← doesn't belong here
}
```

### Fix (Future — Not Blocking)

Create `store/slices/teams.ts`:

```typescript
export interface TeamsSlice {
  backgroundTeamCount: number;
  setBackgroundTeamCount: (count: number) => void;
  // Future: teamList, selectedTeamId, etc.
}

export const createTeamsSlice: StateCreator<Store, [], [], TeamsSlice> = (set) => ({
  backgroundTeamCount: 0,
  setBackgroundTeamCount: (count): void => {
    set({ backgroundTeamCount: count });
  },
});
```

Register in `store/index.ts`:
```typescript
import { createTeamsSlice } from "./slices/teams.js";

const storeConfig = (...a) => ({
  ...createSessionSlice(...a),
  ...createTeamsSlice(...a),   // ← add
  // ...
});
```

Update `Store` type in `types.ts`:
```typescript
export type Store = SessionSlice & TeamsSlice & /* ... */;
```

Remove `backgroundTeamCount` and `setBackgroundTeamCount` from `SessionSlice`.

**Effort:** 30 minutes. Requires updating useTeamCount.ts and StatusLine.tsx imports.

---

## Finding #6: PID Reuse Race Condition (MEDIUM — Accept as-is)

**Reported by:** Backend

### Problem

`isPidAlive(pid)` uses `process.kill(pid, 0)` which is inherently racy. Between reading `background_pid` from config.json and calling `kill`, the original process may have exited and a new unrelated process may have claimed the same PID.

### Assessment

This is a fundamental limitation of PID-based process monitoring. Mitigation would require:
- Storing process start time in config.json and validating against `/proc/{pid}/stat`
- Or using a PID file with flock

**Recommendation:** Accept as best-effort. Add a comment in `useTeamCount.ts`:
```typescript
// Note: PID-based liveness check has inherent race condition with PID reuse.
// This is acceptable for dashboard display — worst case is a brief false positive.
```

**No code change needed.**

---

## Finding #7: JSON Parsing Without Schema Validation (MEDIUM)

**Reported by:** Backend, Standards

### Problem

`countRunningTeams()` parses config.json with `as TeamConfig` type assertion but only checks `background_pid`. Corrupted configs are silently skipped.

### Current Code

```typescript
interface TeamConfig {
  background_pid: number | null;
  [key: string]: unknown;
}

const config = JSON.parse(configData) as TeamConfig;
```

### Fix

Add minimal validation:

```typescript
const raw = JSON.parse(configData);
if (!raw || typeof raw !== "object" || !("background_pid" in raw)) {
  continue; // Skip malformed config
}
const config = raw as TeamConfig;
```

And add debug logging:
```typescript
catch (error) {
  if (process.env["VERBOSE"]) {
    console.warn(`Skipping team ${entry.name}: ${error}`);
  }
  continue;
}
```

**Effort:** 10 minutes.

---

## Finding #8: Session Persistence Race on Rapid Cost Updates (MEDIUM)

**Reported by:** Frontend

### Problem

In `App.tsx`, the `persistSession` useEffect fires on every `totalCost` change. Rapid cost updates (e.g., streaming) can trigger overlapping `saveSession()` calls, potentially causing out-of-order writes.

### Current Code (App.tsx lines 96-124)

```typescript
useEffect(() => {
  async function persistSession() {
    if (!currentSessionId) return;
    if (totalCost === 0) return;
    await saveSession({ ... });
  }
  persistSession();
}, [totalCost, currentSessionId, verbose]);
```

### Fix

Debounce the persist:

```typescript
useEffect(() => {
  const timeout = setTimeout(async () => {
    if (!currentSessionId || totalCost === 0) return;
    await saveSession({ ... });
  }, 2000); // 2s debounce

  return () => clearTimeout(timeout);
}, [totalCost, currentSessionId, verbose]);
```

**Effort:** 5 minutes. No test changes.

---

## Finding #9: StatusLine Has 5+ Concerns (MEDIUM — Future Refactor)

**Reported by:** Architecture

### Problem

StatusLine manages: git polling, context bar, model display, agent counts, team counts, streaming spinner, cost formatting, duration calculation. It has 4 active `setInterval` timers causing ~4 re-renders/second minimum.

### Fix (Future — Not Blocking)

Extract `useStatusLineData()` hook:

```typescript
// hooks/useStatusLineData.ts
export function useStatusLineData() {
  const gitInfo = useGitInfo();
  const teamCount = useTeamCount();
  const agentCounts = useAgentCounts(); // extracted from StatusLine
  const { activeModel, totalCost, contextWindow, streaming } = useStore();
  return { gitInfo, teamCount, agentCounts, activeModel, totalCost, contextWindow, streaming };
}
```

Then StatusLine becomes a pure presentation component.

**Effort:** 1-2 hours. Not blocking for TC-012.

---

## Finding #10: Test Quality Mixed (MEDIUM)

**Reported by:** Standards

### Problems

1. **session-env.test.ts** has 13 test cases, many quasi-duplicates (testing path formation with slightly different inputs). Should use `it.each()` for parameterized tests.

2. **StatusLine.test.tsx** has only 6 test cases — minimal for a component with significant logic (git caching, context bar, model name extraction, duration formatting).

### Fix

**session-env.test.ts:** Consolidate to ~6 tests using `it.each()`:
```typescript
it.each([
  ["simple-id", "/home/user/.claude/sessions/simple-id"],
  ["id-with-dots.123", "/home/user/.claude/sessions/id-with-dots.123"],
  ["id_underscore", "/home/user/.claude/sessions/id_underscore"],
])("should create correct path for session ID %s", (sessionId, expectedPath) => {
  // Single parameterized test
});
```

**StatusLine.test.tsx:** Add tests for:
- Model name extraction (opus/sonnet/haiku/unknown)
- Duration formatting (0s, 59s, 1m, 1h+)
- Context percentage edge cases (0%, 100%, >100%)
- Agent count filtering logic

**Effort:** 30 minutes per test file.

---

## Findings Accepted As-Is (INFO)

| # | Finding | Reviewer | Reason to Accept |
|---|---------|----------|-----------------|
| 11 | Silent error swallowing in countRunningTeams | Backend | Graceful degradation by design — dashboard shouldn't crash on bad config |
| 12 | TeamConfig interface incomplete | Backend, Standards | Only background_pid needed for count; fuller type can wait for teams slice |
| 13 | ASCII mode check is static | Frontend | TERM doesn't change at runtime; acceptable |
| 14 | useTeamCount doesn't react to env var changes | Architecture | Env var set once at init; stale closure won't happen in practice |
| 15 | No backoff on polling failure | Architecture | Failures return 0 silently; no cascading cost |

---

## Action Plan

### Phase 1: Quick Fixes (before commit, ~20 min)

| # | Fix | File | Effort |
|---|-----|------|--------|
| F1 | Extract `setSessionDir()` helper | `App.tsx` | 5 min |
| F2 | Add `isMounted` ref to useTeamCount | `useTeamCount.ts` | 5 min |
| F3 | Add PID race condition comment | `useTeamCount.ts` | 2 min |
| F4 | Add minimal JSON validation + debug logging | `useTeamCount.ts` | 10 min |

### Phase 2: Structural Improvements (follow-up ticket, ~2h)

| # | Fix | Files | Effort |
|---|-----|-------|--------|
| F5 | Move GOGENT_SESSION_DIR to pre-render init or store | `index.tsx` or `types.ts` + `session.ts` | 30 min |
| F6 | Create teams slice, move backgroundTeamCount | `slices/teams.ts`, `types.ts`, `index.ts` | 30 min |
| F7 | Debounce session persistence | `App.tsx` | 5 min |
| F8 | Refactor session-env.test.ts with .each() | `session-env.test.ts` | 15 min |
| F9 | Expand StatusLine test coverage | `StatusLine.test.tsx` | 30 min |

### Phase 3: Future (when team features expand)

| # | Fix | Scope |
|---|-----|-------|
| F10 | Extract shared session discovery for skills | New utility |
| F11 | Extract useStatusLineData hook | StatusLine refactor |
| F12 | Replace polling with filesystem watchers | useTeamCount redesign |
| F13 | Pass sessionDir as explicit prop to useTeamCount | Dependency injection |

---

## Summary

The TC-012 implementation is **functionally correct** — all 28 tests pass, build is clean, all 4 skills are registered and operational. The primary concern is architectural: `process.env` mutation in a React effect creates a timing hazard that causes the first 30s of dashboard display to show 0 teams. Quick fixes (Phase 1) address the immediate issues; structural improvements (Phase 2) should be filed as a follow-up ticket before this code path becomes load-bearing.
