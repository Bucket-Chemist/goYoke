# Critical Review: TUI Health Monitor Dashboard

**Reviewed:** 2026-03-26T14:30:00Z
**Reviewer:** Staff Architect Critical Review
**Input:** /home/doktersmol/.claude/sessions/f5b8176c-7e42-4a6c-83e6-6afe6e4bd3e2/specs.md
**Original Spec:** /home/doktersmol/Documents/GOgent-Fortress/tickets/tui-agent-upgrade/spec.md

---

## Executive Assessment

**Overall Verdict:** APPROVE_WITH_CONDITIONS

**Confidence Level:** HIGH

- Rationale: Plan is clear, well-structured, and I was able to verify all major claims against the actual codebase. The domain (Ink/React TUI with Zustand store) is straightforward. All referenced files exist and match the plan's description of current state.

**Issue Counts:**

- Critical: 0 (must fix)
- Major: 3 (should fix)
- Minor: 4 (consider fixing)

**Commendations:** 5

**Summary:** This is a well-crafted incremental plan that correctly identifies and corrects flaws in the original spec, decomposes work into independently testable tickets, and keeps scope tight (~260 lines). The major issues center on a data model gap in TeamMember (the full config type already in types.ts), an incomplete spec for how TeamDashboard resolves its team data from a selected node, and the explicit deferral of unit tests. None are blocking, but all should be addressed before implementation begins.

**Go/No-Go Recommendation:**
Yes, a contractor could start Monday. The three major issues require 15-20 minutes of clarification added to the specs, not redesign. Approve with conditions below.

---

## Issue Register

### Critical Issues (Must Fix Before Proceeding)

None.

### Major Issues (Should Fix, Can Proceed with Caution)

| ID  | Layer             | Location     | Issue                                      | Impact                               | Recommendation                          |
| --- | ----------------- | ------------ | ------------------------------------------ | ------------------------------------ | --------------------------------------- |
| M-1 | Assumption        | TUI-001      | TeamMember type (line 445 of types.ts) not updated | Runtime fields exist but full-config type is stale | Add health fields to TeamMember interface too |
| M-2 | Contractor Ready  | TUI-004/005  | TeamDashboard prop resolution is underspecified | Implementer must guess how to find team from selectedNode | Specify the exact prop-drilling path |
| M-3 | Testing Coverage  | All phases   | No test tickets; "npx tsc --noEmit" is the only validation | Type errors caught, logic errors missed | Add at least 1 ticket for formatter unit tests |

---

**Detail for M-1: TeamMember full-config type is stale**

The plan's TUI-001 ticket says:

> Add optional fields to TeamMemberRow (6 fields), TeamSummary (1 field), TeamMember (4 fields)

The plan correctly identifies that `TeamMemberRow` (line 386-398) needs new fields. However, `TeamMember` (line 445-462) is a separate interface -- the **full config type** matching the Go struct. Examining the Go-side `Member` struct in `cmd/gogent-team-run/config.go:46-67`, the Go struct already has `health_status`, `last_activity_time`, `stall_count`, `kill_reason`, and `error_message`. The TypeScript `TeamMember` interface at types.ts:445 is currently missing all five of these fields.

The plan mentions adding 4 fields to TeamMember but does not list them explicitly. The original spec lists `process_pid`, `health_status`, `last_activity_time`, `stall_count`, `error_message`, `kill_reason` -- six fields, not four. This discrepancy needs resolution.

**Why it matters:** `TeamMember` is used by `TeamConfig` (line 422-436) which is stored in `TeamsSlice.selectedTeamDetail`. If any downstream code (current or future) reads from `selectedTeamDetail.waves[n].members[m]`, the types would be wrong. More immediately, the `TeamConfigJSON` parsing interface in `useTeams.ts` (line 14-35) must also be extended -- TUI-002 says it will be, but TUI-001 should ensure the destination types are complete.

**Recommendation:** In TUI-001, explicitly list all fields being added to each of the three interfaces:
- `TeamMemberRow`: +processPid, +healthStatus, +lastActivityTime, +stallCount, +errorMessage, +killReason, +streamBytes (7 fields)
- `TeamSummary`: +totalStreamBytes (1 field)
- `TeamMember`: +health_status, +last_activity_time, +stall_count, +kill_reason, +error_message (5 fields, matching Go snake_case JSON tags)

---

**Detail for M-2: TeamDashboard prop resolution is underspecified**

The plan says TUI-005 will replace the team-root/team-member dispatch branches with a unified TeamDashboard. The original spec suggests:

```typescript
<TeamDashboard
  teamDir={selectedNode.teamDir ?? selectedNode.parentTeamDir}
  highlightMember={selectedNode.kind === "team-member" ? selectedNode.displayName : undefined}
/>
```

But the implementation plan (specs.md) correctly notes that `parentTeamDir` is unnecessary because `teamDir` is already set on team-member nodes (useUnifiedTree.ts:94). However, specs.md never provides the corrected JSX for integration. The implementer of TUI-005 must figure out:

1. What props does `TeamDashboard` accept?
2. How does TeamDashboard look up the `TeamSummary` -- via store selector using `teamDir`, or passed as a prop?
3. What does "highlightMember" mean in layout terms -- scroll position? Background color? Bold text?

**Why it matters:** TUI-004 (create component) and TUI-005 (integrate) are separate tickets potentially assigned to different implementers. The interface contract between them is implicit.

**Recommendation:** Add to the TUI-004 ticket description:
```typescript
// TeamDashboard props interface
interface TeamDashboardProps {
  teamDir: string;
  highlightMember?: string; // member name to visually emphasize
}
```
And specify that TeamDashboard uses `useStore(s => s.teams.find(t => t.dir === teamDir))` internally. For TUI-005, provide the exact JSX replacement:
```typescript
{(selectedNode?.kind === "team-root" || selectedNode?.kind === "team-member") &&
  selectedNode.teamDir !== undefined && (
  <TeamDashboard
    teamDir={selectedNode.teamDir}
    highlightMember={selectedNode.kind === "team-member" ? selectedNode.displayName : undefined}
  />
)}
```

---

**Detail for M-3: Testing is type-check only**

The plan says validation for every phase is `npx tsc --noEmit`. The "Out of Scope" section explicitly defers unit tests:

> Unit tests for formatting utilities (could be a follow-up)

While `tsc --noEmit` catches type errors, it misses:
- `formatRelativeTime` producing wrong output for edge cases (negative timestamps, future dates, NaN)
- `formatBytes` rounding incorrectly
- `getHealthColor`/`getHealthIcon` returning wrong values for unexpected status strings
- BudgetBar percentage capping (what if `used > max`?)
- Wave grouping logic with edge cases (empty waves, 0 members)

These are pure functions with no dependencies -- trivially testable.

**Why it matters:** These formatters will be called on every 5s poll render. A bug in `formatRelativeTime` would be visible to every user but hard to diagnose without tests.

**Recommendation:** Add TUI-006 (or fold into TUI-003): unit tests for `formatRelativeTime`, `formatBytes`, `getHealthColor`, `getHealthIcon`, and `BudgetBar` percentage calculation. Estimated 30-40 lines of test code. This is low-hanging fruit that protects the most logic-dense code in the plan.

---

### Minor Issues (Consider Addressing)

| ID  | Layer             | Location  | Issue                                       | Impact                          | Recommendation                        |
| --- | ----------------- | --------- | ------------------------------------------- | ------------------------------- | ------------------------------------- |
| m-1 | Architecture      | TUI-002   | attachStreamActivity stat() sequentialized  | Slower polling for teams with many members | Parallelize stat() calls with Promise.all |
| m-2 | Failure Modes     | TUI-002   | stat() on stream files during write race    | Possible stale/zero size read   | Acceptable but document the known race |
| m-3 | Cost-Benefit      | TUI-004   | 180 lines in single file with 6 sub-components | Near God Component threshold   | Acceptable at current size, revisit if it grows |
| m-4 | Contractor Ready  | TUI-003   | ActivitySection extraction mechanics not specified | Which imports update? Does the export change? | Add explicit before/after for UnifiedDetail imports |

---

**Detail for m-1: Sequential stat() calls in attachStreamActivity**

The plan's proposed `attachStreamActivity` (from spec.md) iterates members sequentially:

```typescript
for (const member of summary.members) {
  try {
    const streamPath = join(teamPath, `stream_${member.agent}.ndjson`);
    const fileStat = await stat(streamPath);
    // ...
  }
}
```

The existing code already uses `Promise.all` for parallel processing across teams (useTeams.ts:251). The per-member loop should follow the same pattern, especially since stat() is I/O-bound.

**Recommendation:** Use `Promise.all(summary.members.map(async (member) => { ... }))` for the stat+read loop. With ~10 members max this is a minor optimization, but it follows the existing pattern and costs nothing in complexity.

---

**Detail for m-2: stat()/read race with gogent-team-run writer**

`gogent-team-run` writes stream files continuously while the TUI polls every 5s. A stat() call could land during a write, returning a size that is immediately stale. More concerning: if the TUI reads the last 4KB chunk while gogent-team-run is appending, the chunk boundary could split a JSON line.

**Current mitigation (already in code):** `readStreamActivity` already handles this -- it skips the first line of a chunk if not reading from position 0 (useTeams.ts:101). The plan's addition of stat() for size is safe because size being off by a few bytes is cosmetically irrelevant.

**Recommendation:** No code change needed, but add a one-line comment in TUI-002 explaining the race is benign: "// Size may be slightly stale due to concurrent writes -- acceptable for display purposes."

---

**Detail for m-3: TeamDashboard.tsx at 180 lines**

The plan notes all sub-components in a single file because "they're small (10-30 lines each), tightly coupled, not reused elsewhere." This is a reasonable judgment call. At 180 lines with 6 sub-components, it is not yet a God Component. However, if features are added later (keyboard navigation, member expansion, log preview), it could cross the threshold.

**Recommendation:** No action now. Note in the file header: "// If this file exceeds ~300 lines, extract WaveSection and MemberRow to separate files."

---

**Detail for m-4: ActivitySection extraction details**

TUI-003 says "Extract ActivitySection from UnifiedDetail.tsx to standalone component." This is a mechanical operation but the ticket should specify:

1. The new file path: `packages/tui/src/components/ActivitySection.tsx`
2. The export: `export function ActivitySection({ activity }: { activity: AgentActivity }): JSX.Element | null`
3. The import change in UnifiedDetail.tsx: `import { ActivitySection } from "./ActivitySection.js";`
4. That the internal helper variables (`resultColor`, `resultLabel`) move with it
5. That the `colors` import moves to the new file

**Recommendation:** Add these 5 points to TUI-003's description. A contractor unfamiliar with the codebase should not have to figure out extraction mechanics.

---

## Assumption Register

| #   | Assumption | Source | Verified? | Risk if False | Mitigation |
| --- | --- | --- | --- | --- | --- |
| A-1 | `health_status`, `stall_count`, `last_activity_time`, `kill_reason`, `error_message` are written to config.json by gogent-team-run | Spec + Plan TUI-002 | **Verified** -- Go struct at config.go:46-67 has all fields with JSON tags | N/A (confirmed) | N/A |
| A-2 | `teamDir` is set on team-member UnifiedNodes | Plan "Spec Corrections" table | **Verified** -- useUnifiedTree.ts:94 sets `teamDir: team.dir` on team-member nodes | N/A (confirmed) | N/A |
| A-3 | Stream files follow naming pattern `stream_{agent}.ndjson` | Plan TUI-002 | **Verified** -- readStreamActivity uses this pattern at useTeams.ts:74 | N/A (confirmed) | N/A |
| A-4 | Stream file naming uses `member.agent`, not `member.name` | Plan proposes stat on `stream_${member.agent}.ndjson` | **CONFLICT** -- existing code at useTeams.ts:130 uses `member.name` not `member.agent` | stat() would fail on every member, returning 0 bytes silently | Verify against gogent-team-run file creation; use whichever field matches |
| A-5 | config.json health fields are present for currently running teams | Plan assumes "already in config.json, just not parsed" | **Partially verified** -- fields exist in Go struct but `omitempty` means they may be absent until first health check (30s after spawn) | Fields may be empty strings or absent for first 30s of member life | Already mitigated: all new fields are optional in TypeScript types |
| A-6 | Ink `Text` component supports `wrap="truncate"` | Plan Risk Register mentions it | **Unverified** -- need to check Ink API docs | Layout overflow in narrow terminals | Verify against Ink API; fallback is manual string truncation |
| A-7 | Maximum ~10 members per team | Plan Risk Register performance mitigation | **Reasonable assumption** -- braintrust teams have 2-4 members typically | If someone creates 50-member team, 50 stat() calls per 5s poll | Add a guard or cap; unlikely in practice |

---

## Dependency Mapping

### Dependency Graph (from plan)

```
TUI-001 --> TUI-002 --+
                       +--> TUI-004 --> TUI-005
TUI-003 --------------+
```

**Verification results:**

1. **Circular dependencies:** None detected. Clean DAG.
2. **Hidden dependencies:** None. TUI-003 is truly independent of Phase 1 (touches theme.ts, teamFormatting.ts, and extracts from UnifiedDetail.tsx -- none of which are modified by TUI-001 or TUI-002).
3. **Order violations:** None. Parallelizable pair (TUI-001 + TUI-003) correctly identified.
4. **External dependencies:** gogent-team-run writing health fields to config.json. **Verified** -- the Go code already does this.
5. **Human dependencies:** None.
6. **Bottleneck analysis:** TUI-004 has 3 dependencies (TUI-001, TUI-002, TUI-003). This is acceptable -- it is the natural convergence point and is correctly the largest ticket.

### File Overlap Analysis

| File | Modified By | Conflict Risk |
| --- | --- | --- |
| types.ts | TUI-001 only | None |
| useTeams.ts | TUI-002 only | None |
| theme.ts | TUI-003 only | None |
| teamFormatting.ts | TUI-003 only | None |
| UnifiedDetail.tsx | TUI-003 (extract), TUI-005 (integrate) | **Low** -- TUI-003 extracts ActivitySection, TUI-005 replaces team branches. Non-overlapping sections. Sequential by dependency graph. |
| ActivitySection.tsx | TUI-003 creates, TUI-004 imports | None -- correct order |
| TeamDashboard.tsx | TUI-004 creates, TUI-005 imports | None -- correct order |

**Verdict:** Clean dependency graph with no conflicts.

---

## Failure Mode Analysis

### Phase 1 (TUI-001 + TUI-002) fails midway

**TUI-001 fails:** Only type additions. If it fails, types.ts is in a partially modified state. **Rollback:** Revert the type additions. No runtime impact since nothing consumes the new fields yet. **Risk:** Trivial.

**TUI-002 fails midway:** Config parsing is partially modified. Some fields parsed, others not. **Impact:** New fields that are parsed but not consumed by any component yet -- no user-visible effect. `attachStreamActivity` changes are the riskiest part (it touches the polling hot path). **Rollback:** Revert useTeams.ts changes. Previous behavior restored. **Risk:** Low -- stat() failure is caught, worst case is undefined streamBytes.

**Phase 1 succeeds, Phase 2 fails:** System has new types and parsing but no dashboard consuming them. **Impact:** Zero user-visible change. New data flows into store but is never rendered. This is a safe intermediate state.

### Phase 2 (TUI-003) fails midway

**ActivitySection extraction incomplete:** UnifiedDetail.tsx references removed function. **Impact:** Type error -- tsc catches this immediately. **Rollback:** Revert all TUI-003 changes (theme.ts, teamFormatting.ts, ActivitySection.tsx, UnifiedDetail.tsx). **Risk:** Low -- all changes are in disjoint sections.

**Phase 2 succeeds, Phase 3 fails:** System has extracted ActivitySection and new formatters, but no dashboard. **Impact:** Zero user-visible change. ActivitySection renders identically from new file location.

### Phase 3 (TUI-004) fails

**TeamDashboard.tsx partially implemented:** File exists but is incomplete or has bugs. **Impact:** No user-visible impact -- nothing imports it yet (TUI-005 has not run). **Rollback:** Delete the file. **Risk:** None.

### Phase 4 (TUI-005) fails

**This is the riskiest phase.** It modifies the dispatch logic in UnifiedDetail.tsx and removes dead code.

**Failure mode 1:** TeamDashboard imported but dispatch logic wrong. **Impact:** Selecting team nodes shows blank or crashes. SDK agent detail potentially broken if cleanup goes wrong. **Detection:** tsc catches type errors; visual testing catches dispatch errors. **Rollback:** Revert UnifiedDetail.tsx to pre-TUI-005 state. TeamDashboard.tsx file remains but is unused.

**Failure mode 2:** Dead code removal (TeamRootDetail, TeamMemberDetail) accidentally removes SdkAgentDetail. **Impact:** Agent detail panel breaks for all SDK agents. **Mitigation:** Plan already identifies this risk. tsc would catch missing component references. **Rollback:** Revert UnifiedDetail.tsx.

**Mandatory Rollback Test verdict:** Every phase has a clean rollback path. No phase burns bridges. The system is in a usable state at every phase boundary. **PASS.**

---

## Commendations

1. **Spec correction table is excellent.** The plan explicitly identifies and corrects 4 inaccuracies in the original spec (parentTeamDir unnecessary, emoji vs theme icons, uptimeMs as derived value, missing TeamMember fields). This prevents implementers from following wrong instructions. This is a model for how plans should handle spec divergence.

2. **Dependency graph enables parallelism.** TUI-001 and TUI-003 are correctly identified as parallelizable (disjoint file sets). This could save wall-clock time if two agents execute concurrently. The graph is a clean DAG with no unnecessary sequential constraints.

3. **Decision log with alternatives considered.** Each design decision documents the chosen approach, the rationale, and the rejected alternatives. This prevents re-litigation during implementation and helps future maintainers understand why choices were made.

4. **Risk register with concrete mitigations.** Six risks are identified with likelihood, impact, and specific mitigations. The performance risk for stat() includes actual measurements ("< 1ms per file, only runs every 5s, max ~10 members"). This is evidence-based risk assessment, not hand-waving.

5. **Correct use of existing patterns.** The plan follows existing codebase conventions: theme.ts for icons (not emoji), Zustand store selectors for data access, Ink components for rendering, teamFormatting.ts for utilities. No new patterns introduced unnecessarily.

---

## Recommendations

### High Priority (address before implementation starts)

1. **Address M-1:** Explicitly list all field additions for all three interfaces in TUI-001 ticket description. Ensure `TeamMember` (the full config type) is also updated to match Go struct.

2. **Address M-2:** Define `TeamDashboardProps` interface in TUI-004 ticket. Provide exact JSX for TUI-005 integration. Specify that `teamDir` from `selectedNode` is the lookup key.

3. **Address A-4 (stream file naming):** The plan's spec.md proposes `stream_${member.agent}.ndjson` but existing code at useTeams.ts:130 uses `member.name`. Verify which is correct by checking what `gogent-team-run` actually names the files. Use the matching field consistently. If `agent` and `name` can differ, this is a silent bug that produces 0B stream sizes for every member.

### Medium Priority (address during implementation)

4. **Address M-3:** Add a TUI-006 ticket (or fold into TUI-003) for unit tests of pure formatting functions. ~30-40 lines, high value-to-effort ratio.

5. **Address m-1:** Use `Promise.all` for parallel stat() calls in the per-member loop, matching the existing parallel pattern at useTeams.ts:251.

### Low Priority (consider addressing, can defer)

6. **Address m-4:** Add extraction mechanics to TUI-003 description (5 bullet points listed in detail above).

7. **Address m-3:** Add a comment in TeamDashboard.tsx header noting the extraction threshold if the file grows beyond ~300 lines.

---

## Final Sign-Off

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-03-26
**Scouts Spawned:** 0 (all claims verifiable from direct file reads)

**Conditions for Approval:**

- [ ] M-1: TUI-001 explicitly lists all field additions for TeamMemberRow, TeamSummary, AND TeamMember
- [ ] M-2: TUI-004 defines TeamDashboardProps; TUI-005 provides exact integration JSX
- [ ] A-4: Stream file naming (`member.agent` vs `member.name`) verified against gogent-team-run output

**Recommended Actions:**

1. Resolve the 3 conditions above (15-20 min of spec clarification)
2. Consider adding unit test ticket (M-3)
3. Proceed with implementation -- plan is architecturally sound

**Post-Approval Monitoring:**

- Watch TUI-002 for the stat() performance on real running teams (verify <1ms claim)
- Watch TUI-004 for terminal width issues with MemberRow layout -- may need iteration
- Watch TUI-005 carefully for SdkAgentDetail regression during dead code removal
- After TUI-005, run the full TUI with a live team and verify 5s poll updates render correctly
